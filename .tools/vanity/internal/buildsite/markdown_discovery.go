package buildsite

import (
	"bytes"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"io/fs"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

type markdownDiscoveryPage struct {
	file string
	mode fs.FileMode
	body []byte
}

type markdownDiscoveryChange struct {
	page markdownDiscoveryPage
	body []byte
}

func finalizeMarkdownDiscovery(stage, deploy string) error {
	configBytes, err := os.ReadFile(filepath.Join(stage, ".zas", "config.yml"))
	if err != nil {
		return err
	}
	var config struct {
		Site struct {
			PublishMarkdown bool   `yaml:"publish_markdown"`
			BaseURL         string `yaml:"baseurl"`
		} `yaml:"site"`
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	if !config.Site.PublishMarkdown {
		return nil
	}
	return addMarkdownDiscovery(deploy, config.Site.BaseURL)
}

// addMarkdownDiscovery advertises only Markdown twins in the assembled deploy
// inventory. Call it after restoring raw vanity pages, with staged site.baseurl,
// when site.publish_markdown is enabled. It validates the complete inventory
// before changing any HTML and inserts bytes without reserializing the page.
func addMarkdownDiscovery(deploy, baseURL string) error {
	if err := auditOutput(deploy); err != nil {
		return err
	}
	root, err := os.Lstat(deploy)
	if err != nil {
		return err
	}
	if !root.IsDir() || root.Mode()&os.ModeSymlink != 0 {
		return errors.New("Markdown discovery requires a regular deploy directory")
	}
	base, err := markdownDiscoveryBase(baseURL)
	if err != nil {
		return fmt.Errorf("Markdown discovery site.baseurl: %w", err)
	}
	pages := make(map[string]markdownDiscoveryPage)
	twins := make(map[string]bool)
	err = filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if file == deploy {
			return nil
		}
		extension := filepath.Ext(file)
		if extension != ".html" && extension != ".md" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Markdown discovery artifact %q is not a regular file", file)
		}
		rel, err := filepath.Rel(deploy, file)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if extension == ".md" {
			twins[rel] = true
			return nil
		}
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		pages[rel] = markdownDiscoveryPage{file: file, mode: info.Mode(), body: body}
		return nil
	})
	if err != nil {
		return err
	}
	var changes []markdownDiscoveryChange
	for rel, page := range pages {
		twin := strings.TrimSuffix(rel, ".html") + ".md"
		expected := ""
		if twins[twin] {
			location := *base
			location.Path = strings.TrimRight(base.Path, "/") + "/" + twin
			location.RawPath = ""
			expected = location.String()
		}
		position, linked, err := markdownDiscoveryHead(page.body, expected)
		if err != nil {
			return fmt.Errorf("Markdown discovery %s: %w", rel, err)
		}
		if expected == "" || linked {
			continue
		}
		link := []byte("\n<link rel=\"alternate\" type=\"text/markdown\" href=\"" + stdhtml.EscapeString(expected) + "\">\n")
		body := make([]byte, 0, len(page.body)+len(link))
		body = append(body, page.body[:position]...)
		body = append(body, link...)
		body = append(body, page.body[position:]...)
		changes = append(changes, markdownDiscoveryChange{page: page, body: body})
	}
	for _, change := range changes {
		if err := writeMarkdownDiscovery(change); err != nil {
			return err
		}
	}
	return nil
}

func markdownDiscoveryBase(raw string) (*url.URL, error) {
	base, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.Opaque != "" {
		return nil, errors.New("expected an absolute HTTP(S) URL without credentials, query or fragment")
	}
	if base.Path != "" && path.Clean(base.Path) != strings.TrimRight(base.Path, "/") && base.Path != "/" {
		return nil, errors.New("base URL path must not contain traversal or repeated separators")
	}
	return base, nil
}

// Token offsets identify the real closing head even when scripts/styles contain
// strings such as </head>. No template or HTML repair pass runs over the bytes.
func markdownDiscoveryHead(body []byte, expected string) (position int, linked bool, err error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	position = -1
	offset, heads, closes, links := 0, 0, 0, 0
	inHead, inBody := false, false
	templateDepth, noscriptDepth := 0, 0
	rawText := ""
	var structureErr error
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return -1, false, z.Err()
			}
			break
		}
		start := offset
		raw := bytes.Clone(z.Raw())
		offset += len(raw)
		token := z.Token()
		if rawText != "" {
			if kind == html.EndTagToken && token.Data == rawText {
				rawText = ""
			}
			continue
		}
		if kind == html.TextToken && inHead && templateDepth == 0 && strings.TrimSpace(token.Data) != "" {
			structureErr = errors.New("non-metadata text in head")
		}
		if kind == html.EndTagToken {
			switch token.Data {
			case "head":
				if !inHead || templateDepth != 0 || noscriptDepth != 0 {
					structureErr = errors.New("malformed closing head")
				}
				inHead = false
				closes++
				position = start
			case "body", "html":
				if inHead {
					structureErr = errors.New("head was not closed before body/html")
				}
			case "template":
				if templateDepth > 0 {
					templateDepth--
				}
			case "noscript":
				if noscriptDepth > 0 {
					noscriptDepth--
				}
			}
			continue
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		switch token.Data {
		case "head":
			if inHead || heads != 0 || inBody || kind == html.SelfClosingTagToken {
				structureErr = errors.New("malformed or duplicate head")
			}
			heads++
			inHead = true
		case "body":
			if inHead {
				structureErr = errors.New("head was not closed before body")
			}
			inBody = true
		case "script", "style", "title":
			rawText = token.Data
		case "textarea", "xmp", "iframe", "noembed", "noframes":
			if inHead && templateDepth == 0 {
				structureErr = fmt.Errorf("unexpected <%s> in head", token.Data)
			}
			rawText = token.Data
		case "template":
			templateDepth++
		case "noscript":
			noscriptDepth++
			// Inspect the disabled-script branch too: a link hidden there must
			// not count as the page's active discovery link.
			z.NextIsNotRawText()
		case "link":
			attrs := make(map[string]string)
			markdown := false
			for _, attr := range token.Attr {
				attrs[attr.Key] = attr.Val
				if attr.Key == "type" && markdownDiscoveryType(attr.Val) {
					markdown = true
				}
			}
			if markdownDiscoveryDuplicateAttributes(raw) {
				return -1, false, errors.New("duplicate attributes on discovery link")
			}
			if !markdown {
				continue
			}
			mediaType, _, typeErr := mime.ParseMediaType(attrs["type"])
			if typeErr != nil || mediaType != "text/markdown" {
				return -1, false, errors.New("invalid Markdown discovery type")
			}
			if !inHead || templateDepth != 0 || noscriptDepth != 0 {
				return -1, false, errors.New("Markdown discovery link outside the active head")
			}
			if !strings.EqualFold(strings.TrimSpace(attrs["rel"]), "alternate") {
				return -1, false, errors.New("Markdown discovery link must declare rel=alternate")
			}
			if expected == "" {
				return -1, false, errors.New("Markdown discovery link has no published twin")
			}
			if attrs["href"] != expected {
				return -1, false, fmt.Errorf("Markdown discovery href %q does not match published twin %q", attrs["href"], expected)
			}
			links++
		default:
			if inHead && templateDepth == 0 && token.Data != "html" && token.Data != "base" && token.Data != "meta" && token.Data != "basefont" && token.Data != "bgsound" {
				structureErr = fmt.Errorf("unexpected <%s> in head", token.Data)
			}
		}
	}
	if expected == "" && links == 0 {
		return -1, false, nil // Includes the seven raw head-only vanity fragments.
	}
	if structureErr != nil {
		return -1, false, structureErr
	}
	if heads != 1 || closes != 1 || position < 0 || inHead || templateDepth != 0 || noscriptDepth != 0 {
		return -1, false, errors.New("expected one complete explicit head")
	}
	if links > 1 {
		return -1, false, errors.New("duplicate Markdown discovery links")
	}
	return position, links == 1, nil
}

// The HTML tokenizer deliberately discards duplicate attributes. Scan only
// names in its already-delimited raw tag, respecting quoted/unquoted values,
// so ambiguous existing links fail instead of silently choosing one value.
func markdownDiscoveryDuplicateAttributes(raw []byte) bool {
	space := func(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }
	i := 1 // Skip < and the tag name.
	for i < len(raw) && !space(raw[i]) && raw[i] != '/' && raw[i] != '>' {
		i++
	}
	seen := make(map[string]bool)
	for i < len(raw) {
		for i < len(raw) && (space(raw[i]) || raw[i] == '/') {
			i++
		}
		if i == len(raw) || raw[i] == '>' {
			break
		}
		start := i
		if raw[i] == '=' { // A leading = belongs to an HTML attribute name.
			i++
		}
		for i < len(raw) && !space(raw[i]) && raw[i] != '=' && raw[i] != '/' && raw[i] != '>' {
			i++
		}
		name := strings.ToLower(strings.ReplaceAll(string(raw[start:i]), "\x00", "\uFFFD"))
		if seen[name] {
			return true
		}
		seen[name] = true
		for i < len(raw) && space(raw[i]) {
			i++
		}
		if i == len(raw) || raw[i] != '=' {
			continue
		}
		i++
		for i < len(raw) && space(raw[i]) {
			i++
		}
		if i < len(raw) && (raw[i] == '\'' || raw[i] == '"') {
			quote := raw[i]
			i++
			for i < len(raw) && raw[i] != quote {
				i++
			}
			if i < len(raw) {
				i++
			}
		} else {
			for i < len(raw) && !space(raw[i]) && raw[i] != '>' {
				i++
			}
		}
	}
	return false
}

func markdownDiscoveryType(raw string) bool {
	mediaType, _, err := mime.ParseMediaType(raw)
	if err == nil {
		return mediaType == "text/markdown"
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "text/markdown")
}

func writeMarkdownDiscovery(change markdownDiscoveryChange) (err error) {
	file, err := os.CreateTemp(filepath.Dir(change.page.file), ".markdown-discovery-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err = file.Write(change.body); err != nil {
		file.Close()
		return err
	}
	if err = file.Chmod(change.page.mode.Perm()); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, change.page.file)
}
