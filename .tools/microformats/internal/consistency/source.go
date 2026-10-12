package consistency

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

type siteConfig struct {
	Site struct {
		BaseURL, Language, Title string
		Feed                     bool
		FeedTitle                string   `yaml:"feed_title"`
		FeedDescription          string   `yaml:"feed_description"`
		FeedSections             []string `yaml:"feed_sections"`
		FeedLimit                int      `yaml:"feed_limit"`
		FeedPerLanguage          bool     `yaml:"feed_per_language"`
	}
}

type sourcePost struct {
	Source, File, Path, URL, Title, Language, Summary, Creator string
	OpenGraphTitle                                             string
	Date, Updated                                              time.Time
	Tags                                                       []string
	Feed, Standard, Excluded                                   bool
	PartialDate                                                string
}

func loadSources(root string) (siteConfig, []sourcePost, error) {
	var config siteConfig
	input, err := regularFile(root, ".zas/config.yml")
	if err != nil {
		return config, nil, err
	}
	if err := yaml.Unmarshal(input, &config); err != nil {
		return config, nil, err
	}
	if config.Site.BaseURL != "https://dario.cat" || !config.Site.Feed || config.Site.Language != "ca" || len(config.Site.FeedSections) == 0 || config.Site.FeedLimit < 1 {
		return config, nil, fmt.Errorf("site must declare its canonical base URL, Catalan default and enabled, capped feed sections")
	}
	var posts []sourcePost
	seen := make(map[string]bool)
	for _, section := range config.Site.FeedSections {
		if section != "posts" && section != "en/posts" && section != "es/posts" && section != "rizoma" {
			return config, nil, fmt.Errorf("unreviewed article section %q", section)
		}
		if seen[section] {
			return config, nil, fmt.Errorf("duplicate article section %q", section)
		}
		seen[section] = true
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(section)), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.EqualFold(entry.Name(), "CV") || strings.EqualFold(entry.Name(), "private") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("article path is a symlink: %s", path)
			}
			if entry.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			input, err := regularFile(root, rel)
			if err != nil {
				return err
			}
			meta, body, err := sourceComment(input)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if meta["article"] != true {
				return nil
			}
			post, err := readSource(root, rel, meta, body, config)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			posts = append(posts, post)
			return nil
		})
		if err != nil {
			return config, nil, err
		}
	}
	if len(posts) == 0 {
		return config, nil, fmt.Errorf("no article sources found")
	}
	return config, posts, nil
}

func sourceComment(input []byte) (map[interface{}]interface{}, []byte, error) {
	input = bytes.TrimSpace(bytes.TrimPrefix(input, []byte{0xef, 0xbb, 0xbf}))
	meta := make(map[interface{}]interface{})
	if !bytes.HasPrefix(input, []byte("<!--")) {
		return meta, input, nil
	}
	end := bytes.Index(input[4:], []byte("-->"))
	if end < 0 {
		return nil, nil, fmt.Errorf("unterminated page configuration")
	}
	end += 4
	if err := yaml.Unmarshal(input[4:end], &meta); err != nil {
		return nil, nil, err
	}
	// Preserve scalar spellings for the shared RFC3339 helper: YAML otherwise
	// converts an unquoted calendar-only value into a time.Time silently.
	var fields map[string]yaml.Node
	if err := yaml.Unmarshal(input[4:end], &fields); err != nil {
		return nil, nil, err
	}
	for _, name := range []string{"date", "updated", "date_created"} {
		if node, ok := fields[name]; ok {
			for node.Kind == yaml.AliasNode && node.Alias != nil {
				node = *node.Alias
			}
			if node.Kind == yaml.ScalarNode {
				meta[name] = node.Value
			}
		}
	}
	return meta, input[end+3:], nil
}

func readSource(root, source string, meta map[interface{}]interface{}, body []byte, config siteConfig) (sourcePost, error) {
	file := strings.TrimSuffix(source, ".md") + ".html"
	data := zas.ZasData{Page: meta, Path: "/" + file, Site: zas.ZasSiteData{BaseURL: config.Site.BaseURL}}
	post := sourcePost{Source: source, File: file, URL: data.Canonical()}
	post.Path = strings.TrimPrefix(post.URL, config.Site.BaseURL)
	for _, key := range []string{"article", "draft", "private", "publish", "feed", "atproto"} {
		if value, present := meta[key]; present {
			if _, ok := value.(bool); !ok {
				return post, fmt.Errorf("%s must be boolean", key)
			}
		}
	}
	if meta["draft"] == true || meta["private"] == true || meta["publish"] == false {
		post.Excluded = true
		return post, nil
	}
	var err error
	post.Date, err = data.Date()
	if err != nil {
		return post, err
	}
	post.Updated, err = data.Updated()
	if err != nil {
		return post, err
	}
	if _, explicit := meta["updated"]; !explicit {
		post.Updated, err = gitUpdated(root, source)
		if err != nil {
			return post, err
		}
	}
	post.Summary, err = data.Summary()
	if err != nil || strings.TrimSpace(post.Summary) == "" {
		return post, fmt.Errorf("missing or invalid article summary: %v", err)
	}
	post.Tags, err = data.Tags()
	if err != nil || post.Tags == nil {
		return post, fmt.Errorf("missing or invalid article tag list: %v", err)
	}
	post.Language, err = sourceLanguage(root, source, meta, config.Site.Language)
	if err != nil {
		return post, err
	}
	post.Creator = config.Site.FeedTitle
	if value, present := meta["author"]; present {
		creator, ok := value.(string)
		if !ok || creator == "" {
			return post, fmt.Errorf("invalid article author")
		}
		post.Creator = creator
	}
	post.Feed, post.Standard = meta["feed"] != false, meta["atproto"] != false
	if value, present := meta["external"]; present && value != "" {
		if meta["feed"] != false || meta["atproto"] != false {
			return post, fmt.Errorf("off-site article must explicitly disable RSS and Standard.site")
		}
	}
	if post.Date.IsZero() {
		precision, _ := meta["date_precision"].(string)
		created, _ := meta["date_created"].(string)
		if (precision != "day" && precision != "local-time") || len(created) < 10 || post.Feed || post.Standard {
			return post, fmt.Errorf("unknown publication instant requires explicit partial precision and feed:false/atproto:false")
		}
		post.PartialDate = created[:10]
		if _, err := time.Parse("2006-01-02", post.PartialDate); err != nil {
			return post, fmt.Errorf("invalid known publication calendar date: %w", err)
		}
	}
	if bytes.Contains(body, []byte("{{")) || bytes.Contains(body, []byte("<embed")) {
		return post, fmt.Errorf("article title requires a reviewed rendered source adapter")
	}
	var rendered bytes.Buffer
	converter := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote), goldmark.WithRendererOptions(goldhtml.WithUnsafe()))
	if err := converter.Convert(body, &rendered); err != nil {
		return post, err
	}
	doc, err := parseHTML(rendered.Bytes())
	if err != nil {
		return post, err
	}
	headings := find(doc, func(n *htmlNode) bool { return isTag(n, "h1") })
	if len(headings) != 1 {
		return post, fmt.Errorf("expected one source H1, got %d", len(headings))
	}
	data.FirstTitle = text(headings[0])
	post.Title = data.Title()
	post.OpenGraphTitle = post.Title + " | " + config.Site.Title
	return post, nil
}

func sourceLanguage(root, source string, meta map[interface{}]interface{}, fallback string) (string, error) {
	if value, present := meta["language"]; present {
		language, ok := value.(string)
		if !ok || language == "" {
			return "", fmt.Errorf("invalid page language")
		}
		return language, nil
	}
	for dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(source))); ; dir = filepath.Dir(dir) {
		rel, err := filepath.Rel(root, filepath.Join(dir, ".zas.yml"))
		if err != nil {
			return "", err
		}
		input, err := regularFile(root, filepath.ToSlash(rel))
		if err == nil {
			var section struct{ Language string }
			if err := yaml.Unmarshal(input, &section); err != nil {
				return "", err
			}
			if section.Language != "" {
				return section.Language, nil
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if dir == root || filepath.Dir(dir) == dir {
			return fallback, nil
		}
	}
}

func gitUpdated(root, source string) (time.Time, error) {
	tracked := exec.Command("git", "-C", root, "--literal-pathspecs", "ls-files", "--error-unmatch", "--", source)
	if tracked.Run() != nil {
		return time.Time{}, nil
	}
	output, err := exec.Command("git", "-C", root, "--literal-pathspecs", "log", "-1", "--format=%cI", "--", source).Output()
	if err != nil {
		return time.Time{}, err
	}
	value := strings.TrimSpace(string(output))
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}
