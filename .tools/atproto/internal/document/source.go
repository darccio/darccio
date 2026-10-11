package document

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/darccio/zas"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

var articlePathPattern = regexp.MustCompile(`^/(?:posts|en/posts|es/posts|rizoma)/[a-z0-9]+(?:[a-z0-9/-]*[a-z0-9])?/$`)

func canonicalPath(value string) bool {
	u, err := url.Parse(value)
	return err == nil && articlePathPattern.MatchString(value) && path.Clean(value)+"/" == value && u.Path == value && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && !strings.ContainsAny(value, "#\\\x00")
}

// Zas's config-comment parser is private. Preserve the literal YAML scalar
// spelling before using its pinned public typed metadata helpers.
func configComment(input []byte) (map[interface{}]interface{}, []byte, error) {
	input = bytes.TrimSpace(bytes.TrimPrefix(input, []byte{0xef, 0xbb, 0xbf}))
	config := make(map[interface{}]interface{})
	if !bytes.HasPrefix(input, []byte("<!--")) {
		return config, input, nil
	}
	end := bytes.Index(input[4:], []byte("-->"))
	if end < 0 {
		return nil, nil, errors.New("unterminated metadata comment")
	}
	end += 4
	var nodes map[interface{}]yaml.Node
	if err := yaml.Unmarshal(input[4:end], &config); err != nil {
		return nil, nil, err
	}
	if err := yaml.Unmarshal(input[4:end], &nodes); err != nil {
		return nil, nil, err
	}
	for _, key := range []string{"date", "updated"} {
		node, ok := nodes[key]
		if !ok {
			continue
		}
		for node.Kind == yaml.AliasNode && node.Alias != nil {
			node = *node.Alias
		}
		if node.ShortTag() == "!!timestamp" {
			config[key] = node.Value
		}
	}
	return config, input[end+3:], nil
}

func loadSources(root string) (map[string]source, error) {
	result := make(map[string]source)
	for _, section := range []string{"posts", "en/posts", "es/posts", "rizoma"} {
		start := filepath.Join(root, filepath.FromSlash(section))
		if err := regularParents(root, start); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		err := filepath.WalkDir(start, func(file string, entry fs.DirEntry, err error) error {
			if errors.Is(err, os.ErrNotExist) && file == start {
				return nil
			}
			if err != nil {
				return err
			}
			if strings.EqualFold(entry.Name(), "private") || strings.EqualFold(entry.Name(), "CV") {
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
				return fmt.Errorf("symlink in article inventory: %s", file)
			}
			if entry.IsDir() || filepath.Ext(file) != ".md" {
				return nil
			}
			input, err := readRegular(file)
			if err != nil {
				return err
			}
			config, body, err := configComment(input)
			if err != nil {
				return fmt.Errorf("%s: %w", file, err)
			}
			for _, key := range []string{"article", "draft", "private", "publish", "feed", "atproto"} {
				if value, present := config[key]; present {
					if _, ok := value.(bool); !ok {
						return fmt.Errorf("%s: %s must be boolean", file, key)
					}
				}
			}
			if config["article"] != true {
				return nil
			}
			rel, err := filepath.Rel(root, file)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			htmlFile := strings.TrimSuffix(rel, ".md") + ".html"
			data := zas.ZasData{Page: config, Path: "/" + htmlFile, Site: zas.ZasSiteData{BaseURL: "https://dario.cat"}}
			canonical := strings.TrimPrefix(data.Canonical(), "https://dario.cat")
			if !canonicalPath(canonical) {
				return fmt.Errorf("%s: expected canonical trailing-slash URL", rel)
			}
			if _, duplicate := result[canonical]; duplicate {
				return fmt.Errorf("duplicate article URL: %s", canonical)
			}
			s := source{Candidate: Candidate{Path: canonical, Source: rel, Language: "ca"}, file: htmlFile, config: config, body: body}
			if language, present := config["language"]; present {
				var ok bool
				s.Language, ok = language.(string)
				if !ok {
					return fmt.Errorf("%s: language must be string", rel)
				}
			} else {
				s.Language, err = directoryLanguage(root, file)
				if err != nil {
					return err
				}
			}
			if s.Language != "ca" && s.Language != "en" && s.Language != "es" {
				return fmt.Errorf("%s: unsupported language tag", rel)
			}
			s.date, err = data.Date()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if !s.date.IsZero() {
				s.PublishedAt = s.date.Format(time.RFC3339Nano)
			}
			s.summary, err = data.Summary()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			s.Summary = s.summary
			var candidateBody bytes.Buffer
			if err := articleConverter().Convert(body, &candidateBody); err != nil {
				return err
			}
			candidateHTML, err := html.Parse(bytes.NewReader(candidateBody.Bytes()))
			if err != nil {
				return err
			}
			var firstHeading func(*html.Node) *html.Node
			firstHeading = func(n *html.Node) *html.Node {
				if n.Type == html.ElementNode && n.Data == "h1" {
					return n
				}
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					if found := firstHeading(child); found != nil {
						return found
					}
				}
				return nil
			}
			if heading := firstHeading(candidateHTML); heading != nil {
				data.FirstTitle = normalized(plaintext(heading))
			}
			s.Title = data.Title()
			s.tags, err = data.Tags()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			s.updated, err = data.Updated()
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if _, present := config["updated"]; !present {
				s.updated, err = gitUpdated(root, rel)
				if err != nil {
					return err
				}
			}
			s.Eligible = true
			switch {
			case config["draft"] == true:
				s.Reason = "draft"
			case config["private"] == true:
				s.Reason = "private"
			case config["publish"] == false:
				s.Reason = "publish:false"
			case config["feed"] == false:
				s.Reason = "feed:false"
			case config["atproto"] == false:
				s.Reason = "atproto:false"
			case config["date_precision"] != nil && config["date_precision"] != "instant":
				s.Reason = "publication timestamp precision unresolved"
			case s.date.IsZero():
				s.Reason = "publication timestamp unresolved"
			case s.date.Before(time.Date(2013, 1, 1, 0, 0, 0, 0, time.UTC)):
				s.Reason = "before reviewed archive cutoff (2013)"
			}
			if s.Reason != "" {
				s.Eligible = false
			}
			result[canonical] = s
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func directoryLanguage(root, file string) (string, error) {
	for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
		configPath := filepath.Join(dir, ".zas.yml")
		input, err := readRegular(configPath)
		if err == nil {
			var config map[string]any
			if err := yaml.Unmarshal(input, &config); err != nil {
				return "", err
			}
			if value, present := config["language"]; present {
				language, ok := value.(string)
				if !ok {
					return "", errors.New("directory language must be string")
				}
				return language, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if dir == root || dir == filepath.Dir(dir) {
			break
		}
	}
	return "ca", nil
}

func gitUpdated(root, rel string) (time.Time, error) {
	tracked := exec.Command("git", "-C", root, "--literal-pathspecs", "ls-files", "--error-unmatch", "--", rel)
	if tracked.Run() != nil {
		return time.Time{}, nil
	}
	output, err := exec.Command("git", "-C", root, "--literal-pathspecs", "log", "-1", "--format=%cI", "--", rel).Output()
	if err != nil {
		return time.Time{}, errors.New("cannot obtain tracked source modification date")
	}
	value := strings.TrimSpace(string(output))
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

func desiredRecord(s source, rendered, publication string) (map[string]any, error) {
	input, err := readRegular(filepath.Join(rendered, filepath.FromSlash(s.file)))
	if err != nil {
		return nil, fmt.Errorf("%s: rendered article required: %w", s.Source, err)
	}
	doc, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	article, err := unique(doc, func(n *html.Node) bool { return hasClass(n, "h-entry") }, "article h-entry")
	if err != nil {
		return nil, err
	}
	if attr(article, "lang") != s.Language {
		return nil, fmt.Errorf("%s: rendered language differs from source", s.Source)
	}
	content, err := unique(article, func(n *html.Node) bool { return hasClass(n, "e-content") }, "article e-content")
	if err != nil {
		return nil, err
	}
	heading, err := unique(content, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "h1" }, "article H1")
	if err != nil {
		return nil, err
	}
	canonical, err := unique(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "link" && attr(n, "rel") == "canonical"
	}, "canonical link")
	if err != nil || attr(canonical, "href") != "https://dario.cat"+s.Path {
		return nil, fmt.Errorf("%s: rendered canonical URL differs from source", s.Source)
	}
	date, err := unique(article, func(n *html.Node) bool { return hasClass(n, "dt-published") }, "publication time")
	if err != nil {
		return nil, err
	}
	renderedDate, err := time.Parse(time.RFC3339Nano, attr(date, "datetime"))
	if err != nil || !renderedDate.Equal(s.date) {
		return nil, fmt.Errorf("%s: rendered publication date differs from source", s.Source)
	}
	if bytes.Contains(s.body, []byte("{{")) || bytes.Contains(s.body, []byte("<embed")) {
		return nil, fmt.Errorf("%s: source needs a supported rendered-body adapter", s.Source)
	}
	var expected bytes.Buffer
	converter := articleConverter()
	if err := converter.Convert(s.body, &expected); err != nil {
		return nil, err
	}
	parsedSource, err := html.Parse(bytes.NewReader(expected.Bytes()))
	if err != nil {
		return nil, err
	}
	if normalized(plaintext(content)) != normalized(plaintext(parsedSource)) {
		return nil, fmt.Errorf("%s: rendered content is stale or differs from full source", s.Source)
	}
	data := zas.ZasData{Page: s.config, FirstTitle: normalized(plaintext(heading))}
	title := data.Title()
	if title != normalized(plaintext(heading)) {
		return nil, fmt.Errorf("%s: rendered title differs from source metadata", s.Source)
	}
	tags := append([]string{}, s.tags...)
	languageTag := "lang:" + s.Language
	found := false
	for _, tag := range tags {
		if tag == languageTag {
			found = true
		}
		if strings.HasPrefix(tag, "lang:") && tag != languageTag {
			return nil, fmt.Errorf("%s: conflicting language tag", s.Source)
		}
	}
	if !found {
		tags = append(tags, languageTag)
	}
	record := map[string]any{"$type": "site.standard.document", "site": publication, "path": s.Path, "title": title, "publishedAt": s.PublishedAt, "description": s.summary, "textContent": plaintext(content), "tags": tags}
	if !s.updated.IsZero() {
		record["updatedAt"] = s.updated.Format(time.RFC3339Nano)
	}
	if err := validateRecord(record); err != nil {
		return nil, fmt.Errorf("%s: %w", s.Source, err)
	}
	return record, nil
}

func validateRecord(record map[string]any) error {
	for _, spec := range []struct {
		key          string
		bytes, runes int
	}{{"title", 5000, 500}, {"description", 30000, 3000}} {
		value, ok := record[spec.key].(string)
		if !ok || !utf8.ValidString(value) || len(value) > spec.bytes || utf8.RuneCountInString(value) > spec.runes {
			return fmt.Errorf("%s exceeds conservative schema bounds", spec.key)
		}
		if spec.key == "title" && value == "" {
			return errors.New("document title is empty")
		}
	}
	text, ok := record["textContent"].(string)
	if !ok || !utf8.ValidString(text) || strings.TrimSpace(text) == "" {
		return errors.New("full document plaintext is empty or invalid")
	}
	for _, tag := range record["tags"].([]string) {
		if !utf8.ValidString(tag) || len(tag) > 1280 || utf8.RuneCountInString(tag) > 128 {
			return errors.New("tag exceeds conservative schema bounds")
		}
	}
	encoded, err := json.Marshal(record)
	if err != nil || len(encoded) > 1<<20 {
		return errors.New("document exceeds conservative record size limit")
	}
	return nil
}

func articleConverter() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote), goldmark.WithRendererOptions(goldhtml.WithUnsafe()))
}

func attr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func hasClass(n *html.Node, key string) bool {
	for _, class := range strings.Fields(attr(n, "class")) {
		if class == key {
			return true
		}
	}
	return false
}
func unique(root *html.Node, match func(*html.Node) bool, label string) (*html.Node, error) {
	var nodes []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if match(n) {
			nodes = append(nodes, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	if len(nodes) != 1 {
		return nil, fmt.Errorf("expected exactly one %s, got %d", label, len(nodes))
	}
	return nodes[0], nil
}
func normalized(value string) string { return strings.Join(strings.Fields(value), " ") }

// Plaintext includes every text node and preserves paragraph/code boundaries.
// Layout headers/footers lie outside e-content; images are non-text content.
func plaintext(root *html.Node) string {
	var out strings.Builder
	var parts []string
	flush := func() {
		for _, line := range strings.Split(out.String(), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				parts = append(parts, line)
			}
		}
		out.Reset()
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.CommentNode || (n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style")) {
			return
		}
		if n.Type == html.ElementNode && n.Data == "pre" {
			flush()
			var raw strings.Builder
			var code func(*html.Node)
			code = func(child *html.Node) {
				if child.Type == html.TextNode {
					raw.WriteString(child.Data)
				}
				for next := child.FirstChild; next != nil; next = next.NextSibling {
					code(next)
				}
			}
			code(n)
			if value := strings.Trim(raw.String(), "\r\n"); value != "" {
				parts = append(parts, value)
			}
			return
		}
		block := n.Type == html.ElementNode && strings.Contains(" p div h1 h2 h3 h4 h5 h6 li blockquote pre tr section article br hr ", " "+n.Data+" ")
		if block {
			out.WriteByte('\n')
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
			out.WriteByte(' ')
		}
		if block {
			out.WriteByte('\n')
		}
	}
	walk(root)
	flush()
	return strings.Join(parts, "\n")
}

func sortedSources(sources map[string]source) []Candidate {
	result := make([]Candidate, 0, len(sources))
	for _, s := range sources {
		result = append(result, s.Candidate)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}
