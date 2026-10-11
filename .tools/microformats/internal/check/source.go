// Package check validates semantic microformats against independent public
// sources. It performs no network requests and never renders private CV data.
package check

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/darccio/zas"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"go.yaml.in/yaml/v3"
)

type Post struct {
	Source, File, URL, Title, Language, Summary string
	Date, Updated                               time.Time
	Tags                                        []string
	Body                                        []byte
	CalendarDate                                string
}

type inventory struct {
	Posts    []Post
	Excluded []Post
}

// Zas keeps extraction of its leading configuration comment private. The
// site's article sources use one literal YAML mapping with no templates; after
// extraction we use the exact pinned Zas date, summary, tag and URL helpers.
func sourceConfig(input []byte) (map[interface{}]interface{}, []byte, error) {
	input = bytes.TrimSpace(bytes.TrimPrefix(input, []byte{0xef, 0xbb, 0xbf}))
	config := make(map[interface{}]interface{})
	if !bytes.HasPrefix(input, []byte("<!--")) {
		return config, input, nil
	}
	end := bytes.Index(input[4:], []byte("-->"))
	if end < 0 {
		return nil, nil, fmt.Errorf("unterminated leading config comment")
	}
	end += 4
	if err := yaml.Unmarshal(input[4:end], &config); err != nil {
		return nil, nil, err
	}
	return config, input[end+3:], nil
}

func loadInventory(root string) (inventory, error) {
	var result inventory
	for _, section := range []string{"posts", "en/posts", "es/posts", "rizoma"} {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(section)), func(path string, entry fs.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil // Small offline test sites need not contain all sections.
			}
			if err != nil {
				return err
			}
			if strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("article source is a symlink: %s", path)
			}
			input, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			config, markdown, err := sourceConfig(input)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if config["article"] != true {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			file := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".html"
			data := zas.ZasData{Page: config, Path: "/" + file, Site: zas.ZasSiteData{BaseURL: "https://dario.cat"}}
			post := Post{Source: rel, File: file, URL: data.Canonical()}
			if config["draft"] == true || config["publish"] == false {
				result.Excluded = append(result.Excluded, post)
				return nil
			}
			post.Date, err = data.Date()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			post.Updated, err = data.Updated()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if _, explicit := config["updated"]; !explicit {
				post.Updated, err = gitUpdated(root, rel)
				if err != nil {
					return err
				}
			}
			post.Summary, err = data.Summary()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			post.Tags, err = data.Tags()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			post.Language, err = sourceLanguage(root, path, config)
			if err != nil {
				return err
			}
			if post.Date.IsZero() {
				precision, _ := config["date_precision"].(string)
				created, _ := config["date_created"].(string)
				if (precision != "day" && precision != "local-time") || len(created) < 10 {
					return fmt.Errorf("%s: missing publication date without explicit source precision", rel)
				}
				post.CalendarDate = created[:10]
				if _, err := time.Parse("2006-01-02", post.CalendarDate); err != nil {
					return fmt.Errorf("%s: invalid known calendar date: %w", rel, err)
				}
			}
			// Current article sources have no template actions or embeds. Fail
			// explicitly if that changes rather than comparing unexecuted text.
			if bytes.Contains(markdown, []byte("{{")) || bytes.Contains(markdown, []byte("<embed")) {
				return fmt.Errorf("%s: article content requires a rendered source adapter", rel)
			}
			var body bytes.Buffer
			converter := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote), goldmark.WithRendererOptions(goldhtml.WithUnsafe()))
			if err := converter.Convert(markdown, &body); err != nil {
				return err
			}
			post.Body = body.Bytes()
			node, err := parseHTML(post.Body)
			if err != nil {
				return err
			}
			headings := find(node, func(n *htmlNode) bool { return isTag(n, "h1") })
			if len(headings) != 1 {
				return fmt.Errorf("%s: source must have exactly one article H1, got %d", rel, len(headings))
			}
			data.FirstTitle = textContent(headings[0])
			post.Title = data.Title()
			result.Posts = append(result.Posts, post)
			return nil
		})
		if err != nil {
			return inventory{}, err
		}
	}
	if len(result.Posts) == 0 {
		return inventory{}, fmt.Errorf("no public article sources found")
	}
	return result, nil
}

func sourceLanguage(root, path string, config map[interface{}]interface{}) (string, error) {
	if language, ok := config["language"].(string); ok && language != "" {
		return language, nil
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		input, err := os.ReadFile(filepath.Join(dir, ".zas.yml"))
		if err == nil {
			var section map[string]interface{}
			if err := yaml.Unmarshal(input, &section); err != nil {
				return "", err
			}
			if language, ok := section["language"].(string); ok && language != "" {
				return language, nil
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if dir == root || dir == filepath.Dir(dir) {
			break
		}
	}
	return "ca", nil
}

func gitUpdated(root, source string) (time.Time, error) {
	// Match Zas's fallback: only currently tracked files have Git dates.
	tracked := exec.Command("git", "-C", root, "--literal-pathspecs", "ls-files", "--error-unmatch", "--", source)
	if tracked.Run() != nil {
		return time.Time{}, nil
	}
	command := exec.Command("git", "-C", root, "--literal-pathspecs", "log", "-1", "--format=%cI", "--", source)
	output, err := command.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("Git updated date for %s: %w", source, err)
	}
	value := strings.TrimSpace(string(output))
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}
