package buildsite

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

type agentContextManifest struct {
	Version   int                    `yaml:"version"`
	Languages []agentContextLanguage `yaml:"languages"`
}

type agentContextLanguage struct {
	Language     string                `yaml:"language"`
	Title        string                `yaml:"title"`
	Summary      string                `yaml:"summary"`
	Introduction string                `yaml:"introduction,omitempty"`
	Feed         bool                  `yaml:"feed"`
	Sections     []agentContextSection `yaml:"sections"`
}

type agentContextSection struct {
	Title string             `yaml:"title"`
	Pages []agentContextPage `yaml:"pages"`
}

type agentContextPage struct {
	Source      string `yaml:"source"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type agentContextMetadata struct {
	Publish         *bool  `yaml:"publish"`
	PublishMarkdown *bool  `yaml:"publish_markdown"`
	Draft           bool   `yaml:"draft"`
	Private         bool   `yaml:"private"`
	Language        string `yaml:"language"`
}

// The manifest is a reviewed inventory, not a filesystem scan. Assemble only
// generated Markdown twins, and validate every selected source and destination
// before writing any context files into the isolated deployment.
func finalizeAgentContext(stage, deploy string) error {
	manifestPath := filepath.Join(stage, ".zas", "llms.yml")
	if _, err := os.Lstat(manifestPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	manifestBytes, err := readAgentContextFile(stage, ".zas/llms.yml")
	if err != nil {
		return err
	}
	var manifest agentContextManifest
	decoder := yaml.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("agent context manifest: %w", err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("agent context manifest must contain one YAML document")
	}
	configBytes, err := readAgentContextFile(stage, ".zas/config.yml")
	if err != nil {
		return err
	}
	var config struct {
		Site struct {
			BaseURL         string `yaml:"baseurl"`
			PublishMarkdown bool   `yaml:"publish_markdown"`
		} `yaml:"site"`
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	if !config.Site.PublishMarkdown {
		return errors.New("agent context requires site.publish_markdown")
	}
	base, err := markdownDiscoveryBase(config.Site.BaseURL)
	if err != nil {
		return fmt.Errorf("agent context site.baseurl: %w", err)
	}
	if manifest.Version != 1 || len(manifest.Languages) != 3 {
		return errors.New("agent context requires manifest version 1 and languages ca, en, es")
	}
	locales := []string{"ca", "en", "es"}
	seen := make(map[string]bool)
	// Output order comes from the manifest; map iteration never affects bytes.
	type artifact struct{ file, body string }
	var artifacts []artifact
	for i, group := range manifest.Languages {
		if group.Language != locales[i] {
			return errors.New("agent context languages must be ca, en, es in that order")
		}
		if !agentContextText(group.Title, false) || !agentContextText(group.Summary, false) || !agentContextText(group.Introduction, true) || len(group.Sections) == 0 {
			return fmt.Errorf("agent context %s requires plain title, summary and sections", group.Language)
		}
		var summary, full strings.Builder
		fmt.Fprintf(&summary, "# %s\n\n> %s\n", group.Title, group.Summary)
		if group.Introduction != "" {
			fmt.Fprintf(&summary, "\n%s\n", group.Introduction)
		}
		labels := agentContextLabels(group.Language)
		fmt.Fprintf(&full, "# %s — %s\n\n> %s\n\n%s\n", group.Title, labels[0], group.Summary, labels[1])
		sectionTitles := make(map[string]bool)
		for _, section := range group.Sections {
			if !agentContextText(section.Title, false) || sectionTitles[section.Title] || len(section.Pages) == 0 {
				return fmt.Errorf("agent context %s has an empty or duplicate section", group.Language)
			}
			sectionTitles[section.Title] = true
			fmt.Fprintf(&summary, "\n## %s\n", section.Title)
			for _, page := range section.Pages {
				if err := agentContextSourcePath(page.Source); err != nil {
					return err
				}
				if seen[page.Source] {
					return fmt.Errorf("duplicate agent context source %s", page.Source)
				}
				seen[page.Source] = true
				if !agentContextText(page.Title, false) || !agentContextText(page.Description, false) {
					return fmt.Errorf("agent context %s requires a plain title and description", page.Source)
				}
				if err := agentContextPublicSource(stage, page.Source, group.Language); err != nil {
					return fmt.Errorf("agent context %s: %w", page.Source, err)
				}
				twin, err := readAgentContextFile(deploy, page.Source)
				if err != nil {
					return fmt.Errorf("agent context Markdown twin %s: %w", page.Source, err)
				}
				if err := agentContextMarkdown(twin); err != nil {
					return fmt.Errorf("agent context Markdown twin %s: %w", page.Source, err)
				}
				htmlPath := strings.TrimSuffix(page.Source, ".md") + ".html"
				pageHTML, err := readAgentContextFile(deploy, htmlPath)
				if err != nil {
					return fmt.Errorf("agent context HTML %s: %w", htmlPath, err)
				}
				canonical, language, err := sitemapHTMLMetadata(pageHTML)
				if err != nil || language != group.Language {
					return fmt.Errorf("agent context HTML %s has invalid metadata or language %q: %v", htmlPath, language, err)
				}
				canonicalPath := htmlPath
				if path.Base(canonicalPath) == "index.html" {
					canonicalPath = strings.TrimSuffix(canonicalPath, "index.html")
				}
				if canonical != agentContextURL(base, canonicalPath) {
					return fmt.Errorf("agent context HTML %s canonical does not match its deployed path", htmlPath)
				}
				twinURL := agentContextURL(base, page.Source)
				fmt.Fprintf(&summary, "- [%s](%s): %s\n", agentContextEscape(page.Title), twinURL, page.Description)
				fmt.Fprintf(&full, "\n---\n\n## %s\n\nLanguage: %s\nSource: %s\nCanonical: %s\n\n", page.Title, language, twinURL, canonical)
				full.Write(twin)
				if twin[len(twin)-1] != '\n' {
					full.WriteByte('\n')
				}
			}
		}
		if group.Feed {
			feedPath := agentContextPrefix(group.Language) + "feed.xml"
			feed, err := readAgentContextFile(deploy, feedPath)
			if err != nil {
				return fmt.Errorf("agent context feed %s: %w", feedPath, err)
			}
			var rss struct {
				XMLName xml.Name `xml:"rss"`
				Version string   `xml:"version,attr"`
			}
			if err := xml.Unmarshal(feed, &rss); err != nil || rss.XMLName.Space != "" || rss.Version != "2.0" {
				return fmt.Errorf("agent context feed %s is not RSS 2.0", feedPath)
			}
			fmt.Fprintf(&summary, "\n## RSS\n- [RSS](%s)\n", agentContextURL(base, feedPath))
		}
		fmt.Fprintf(&summary, "\n## %s\n", labels[2])
		for _, other := range locales {
			fmt.Fprintf(&summary, "- [%s](%s)\n", map[string]string{"ca": "Català", "en": "English", "es": "Español"}[other], agentContextURL(base, agentContextPrefix(other)+"llms.txt"))
		}
		fmt.Fprintf(&summary, "\n## Optional\n- [%s](%s)\n", labels[0], agentContextURL(base, agentContextPrefix(group.Language)+"llms-full.txt"))
		artifacts = append(artifacts, artifact{agentContextPrefix(group.Language) + "llms.txt", summary.String()}, artifact{agentContextPrefix(group.Language) + "llms-full.txt", full.String()})
	}
	// Validate output parents and collisions without reading existing artifacts.
	for _, output := range artifacts {
		if _, err := agentContextRegularPath(deploy, output.file, true); err != nil {
			return err
		}
	}
	for _, output := range artifacts {
		file := filepath.Join(deploy, filepath.FromSlash(output.file))
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(file, []byte(output.body), 0644); err != nil {
			return err
		}
	}
	return nil
}

func agentContextPrefix(language string) string {
	if language == "ca" {
		return ""
	}
	return language + "/"
}

func agentContextLabels(language string) [3]string {
	switch language {
	case "ca":
		return [3]string{"context complet", "Les pàgines següents conserven el Markdown publicat. Els enllaços relatius es resolen respecte de la seva URL canònica.", "Idiomes"}
	case "es":
		return [3]string{"contexto completo", "Las páginas siguientes conservan el Markdown publicado. Los enlaces relativos se resuelven desde su URL canónica.", "Idiomas"}
	default:
		return [3]string{"full context", "The following pages preserve the published Markdown. Resolve relative links against each page's canonical URL.", "Languages"}
	}
}

func agentContextURL(base *url.URL, relative string) string {
	u := *base
	u.Path = strings.TrimRight(base.Path, "/") + "/" + relative
	u.RawPath = ""
	return u.String()
}

func agentContextText(text string, optional bool) bool {
	if text == "" {
		return optional
	}
	return strings.TrimSpace(text) == text && !strings.ContainsAny(text, "\r\n[]<>") && strings.IndexFunc(text, unicode.IsControl) == -1
}

func agentContextEscape(text string) string {
	return strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_").Replace(text)
}

func agentContextSourcePath(source string) error {
	if source == "" || path.Clean(source) != source || path.IsAbs(source) || strings.ContainsAny(source, "\\%?#:()") || path.Ext(source) != ".md" || !allowedSource(source) {
		return fmt.Errorf("unsafe agent context source %q", source)
	}
	parts := strings.Split(source, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") || strings.EqualFold(part, "CV") || strings.EqualFold(part, "private") || strings.IndexFunc(part, unicode.IsControl) != -1 {
			return fmt.Errorf("excluded agent context source %q", source)
		}
	}
	root := parts[0]
	if root == "en" || root == "es" {
		if len(parts) < 3 {
			return fmt.Errorf("unreviewed agent context source %q", source)
		}
		root = parts[1]
	}
	switch root {
	case "about", "now", "projects", "contact", "hello", "colophon", "ai", "feeds", "slashes", "talks", "languages", "elsewhere", "sponsor", "uses", "blogroll", "posts", "rizoma":
		return nil
	default:
		return fmt.Errorf("unreviewed agent context source %q", source)
	}
}

func agentContextPublicSource(stage, source, language string) error {
	directory, err := agentContextDirectoryMetadata(stage, source)
	if err != nil {
		return err
	}
	if err := agentContextEligible(directory); err != nil {
		return fmt.Errorf("directory metadata: %w", err)
	}
	file, err := agentContextRegularPath(stage, source, false)
	if err != nil {
		return err
	}
	in, err := os.Open(file)
	if err != nil {
		return err
	}
	defer in.Close()
	// Inspect only leading metadata, allowing Zas's optional leading doctype.
	// Buffered reads may prefetch body bytes; raw bodies never supply context.
	scanner := bufio.NewScanner(io.LimitReader(in, 64<<10))
	scanner.Buffer(make([]byte, 1024), 64<<10)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "<!doctype") {
			for {
				close := strings.Index(line, ">")
				if comment := strings.Index(line, "<!--"); comment >= 0 && (close < 0 || comment < close) {
					return errors.New("unterminated source doctype before metadata")
				}
				if close >= 0 {
					line = strings.TrimSpace(line[close+1:])
					break
				}
				if !scanner.Scan() {
					return errors.New("unterminated source doctype")
				}
				line = scanner.Text()
			}
			for line == "" && scanner.Scan() {
				line = strings.TrimSpace(scanner.Text())
			}
		}
		if !strings.HasPrefix(line, "<!--") {
			return agentContextMetadataLanguage(directory.Language, language)
		}
		var header strings.Builder
		line = strings.TrimPrefix(line, "<!--")
		for {
			if close := strings.Index(line, "-->"); close >= 0 {
				header.WriteString(line[:close])
				var metadata agentContextMetadata
				if err := yaml.Unmarshal([]byte(header.String()), &metadata); err != nil {
					return err
				}
				if err := agentContextEligible(metadata); err != nil {
					return err
				}
				if metadata.Language == "" {
					metadata.Language = directory.Language
				}
				return agentContextMetadataLanguage(metadata.Language, language)
			}
			header.WriteString(line)
			header.WriteByte('\n')
			if !scanner.Scan() {
				return errors.New("unterminated source metadata comment")
			}
			line = scanner.Text()
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return agentContextMetadataLanguage(directory.Language, language)
}

// Zas uses the nearest .zas.yml, rather than merging every ancestor. Deny
// private/publication opt-outs there even if stale output remains in deploy.
func agentContextDirectoryMetadata(stage, source string) (agentContextMetadata, error) {
	for parent := path.Dir(source); ; parent = path.Dir(parent) {
		relative := path.Join(parent, ".zas.yml")
		file := filepath.Join(stage, filepath.FromSlash(relative))
		if _, err := os.Lstat(file); err == nil {
			body, err := readAgentContextFile(stage, relative)
			if err != nil {
				return agentContextMetadata{}, err
			}
			var metadata agentContextMetadata
			if err := yaml.Unmarshal(body, &metadata); err != nil {
				return metadata, fmt.Errorf("directory metadata %s: %w", relative, err)
			}
			return metadata, nil
		} else if !os.IsNotExist(err) {
			return agentContextMetadata{}, err
		}
		if parent == "." {
			return agentContextMetadata{}, nil
		}
	}
}

func agentContextEligible(metadata agentContextMetadata) error {
	if metadata.Draft || metadata.Private || (metadata.Publish != nil && !*metadata.Publish) || (metadata.PublishMarkdown != nil && !*metadata.PublishMarkdown) {
		return errors.New("source is draft, private or unpublished (including Markdown opt-out)")
	}
	return nil
}

func agentContextMetadataLanguage(actual, expected string) error {
	if actual != "" && actual != expected {
		return fmt.Errorf("source language %q does not match selected %q", actual, expected)
	}
	return nil
}

func agentContextMarkdown(body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || strings.Contains(trimmed, "{{") {
		return errors.New("empty or unrendered Markdown")
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") {
		return errors.New("HTML document cannot substitute for a Markdown twin")
	}
	return nil
}

// Check each path component with Lstat: checking only the final file would
// follow a symlinked directory. CV/private paths are rejected before this.
func agentContextRegularPath(root, relative string, allowMissing bool) (string, error) {
	file := filepath.Join(root, filepath.FromSlash(relative))
	current := root
	components := append([]string{""}, strings.Split(relative, "/")...)
	for i, component := range components {
		if component != "" {
			current = filepath.Join(current, component)
		}
		info, err := os.Lstat(current)
		if allowMissing && os.IsNotExist(err) {
			return file, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing agent context symlink %s", current)
		}
		if i == len(components)-1 {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("agent context artifact %s is not regular", current)
			}
		} else if !info.IsDir() {
			return "", fmt.Errorf("agent context parent %s is not a directory", current)
		}
	}
	return file, nil
}

func readAgentContextFile(root, relative string) ([]byte, error) {
	file, err := agentContextRegularPath(root, relative, false)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}
