package buildsite

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

type translationManifest struct {
	Version int                `yaml:"version"`
	Groups  []translationGroup `yaml:"groups"`
}

type translationGroup struct {
	Catalan string `yaml:"ca"`
	English string `yaml:"en"`
	Spanish string `yaml:"es"`
}

func (group translationGroup) paths() map[string]string {
	paths := make(map[string]string)
	for language, path := range map[string]string{"ca": group.Catalan, "en": group.English, "es": group.Spanish} {
		if path != "" {
			paths[language] = path
		}
	}
	return paths
}

// Use the published HTML inventory, rather than guessing sibling URLs or
// trusting directory-config inheritance. Reviewed exceptions cover translated
// articles with different slugs. The same groups supply sitemap alternates.
func finalizeLanguageNavigation(stage, deploy string) error {
	if _, err := os.Lstat(filepath.Join(stage, ".zas/translations.yml")); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	data, err := readAgentContextFile(stage, ".zas/translations.yml")
	if err != nil {
		return err
	}
	var manifest translationManifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("translation manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("translation manifest must contain one YAML document")
	}
	if manifest.Version != 1 {
		return errors.New("translation manifest requires version 1")
	}
	configBytes, err := readAgentContextFile(stage, ".zas/config.yml")
	if err != nil {
		return err
	}
	var config struct {
		Site struct {
			BaseURL string `yaml:"baseurl"`
			Sitemap bool   `yaml:"sitemap"`
		} `yaml:"site"`
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	base, err := localSitemapURL(config.Site.BaseURL, nil)
	if err != nil {
		return err
	}
	pages, _, err := collectSitemapPages(deploy, base)
	if err != nil {
		return err
	}
	byPath := make(map[string]*sitemapPage)
	groups := make(map[string]map[string]string)
	for _, page := range pages {
		canonical, _ := url.Parse(page.canonical)
		path := canonical.EscapedPath()
		byPath[path] = page
		key := path
		switch page.language {
		case "ca":
			if strings.HasPrefix(path, "/en/") || strings.HasPrefix(path, "/es/") {
				continue
			}
		case "en", "es":
			prefix := "/" + page.language + "/"
			if !strings.HasPrefix(path, prefix) {
				continue // Shared English CV/docs retain their root paths.
			}
			key = strings.TrimPrefix(path, "/"+page.language)
		default:
			continue // Raw import fragments have no language menu.
		}
		if groups[key] == nil {
			groups[key] = make(map[string]string)
		}
		groups[key][page.language] = path
	}
	translations := make(map[string]map[string]string)
	for _, group := range groups {
		for _, path := range group {
			translations[path] = group
		}
	}
	seen := make(map[string]bool)
	for _, exception := range manifest.Groups {
		group := exception.paths()
		if len(group) < 2 {
			return errors.New("reviewed translation group needs at least two languages")
		}
		for language, path := range group {
			page := byPath[path]
			if page == nil || page.language != language || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
				return fmt.Errorf("translation target %q is not a published %s page", path, language)
			}
			if seen[path] {
				return fmt.Errorf("duplicate reviewed translation target %s", path)
			}
			seen[path] = true
			for otherLanguage, otherPath := range translations[path] {
				if group[otherLanguage] != otherPath {
					return fmt.Errorf("reviewed translation conflicts with matching paths for %s", path)
				}
			}
		}
		for _, path := range group {
			translations[path] = group
		}
	}
	for language, path := range map[string]string{"ca": "/", "en": "/en/"} {
		if page := byPath[path]; page == nil || page.language != language {
			return fmt.Errorf("language fallback %s must be a published %s homepage", path, language)
		}
	}
	var changes []markdownDiscoveryChange
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		page := byPath[path]
		body, err := readAgentContextFile(deploy, page.file)
		if err != nil {
			return err
		}
		targets := map[string]string{"ca": "/", "en": "/en/"}
		for language, target := range translations[path] {
			targets[language] = target
		}
		targets[page.language] = path
		updated, err := languageNavigationHTML(body, page.language, targets)
		if err != nil {
			return fmt.Errorf("language navigation %s: %w", path, err)
		}
		if !bytes.Equal(body, updated) {
			file := filepath.Join(deploy, filepath.FromSlash(page.file))
			info, err := os.Lstat(file)
			if err != nil {
				return err
			}
			changes = append(changes, markdownDiscoveryChange{page: markdownDiscoveryPage{file: file, mode: info.Mode()}, body: updated})
		}
	}
	if config.Site.Sitemap {
		data, err := readAgentContextFile(deploy, "sitemap.xml")
		if err != nil {
			return err
		}
		var sitemap siteSitemap
		if err := readNativeSitemap(data, &sitemap); err != nil {
			return err
		}
		for i := range sitemap.URLs {
			entry := &sitemap.URLs[i]
			canonical, err := localSitemapURL(entry.Location, base)
			if err != nil || byPath[canonical.EscapedPath()] == nil {
				return fmt.Errorf("translation sitemap location %q has no published page", entry.Location)
			}
			group := translations[canonical.EscapedPath()]
			if len(group) < 2 {
				continue
			}
			entry.Links = nil
			for _, language := range []string{"ca", "en", "es"} {
				if target := group[language]; target != "" {
					entry.Links = append(entry.Links, sitemapAlternate{Rel: "alternate", Language: language, URL: byPath[target].canonical})
				}
			}
		}
		sitemap.XMLName = xml.Name{Local: "urlset"}
		sitemap.XMLNS, sitemap.XHTML = sitemapNamespace, xhtmlNamespace
		output, err := xml.MarshalIndent(sitemap, "", "  ")
		if err != nil {
			return err
		}
		output = append(append([]byte(xml.Header), output...), '\n')
		changes = append(changes, markdownDiscoveryChange{page: markdownDiscoveryPage{file: filepath.Join(deploy, "sitemap.xml"), mode: 0644}, body: output})
	}
	// Plan every change before writing; the caller promotes only a fully checked
	// isolated build. Page content, identity and raw vanity bytes are untouched.
	for _, change := range changes {
		if err := writeMarkdownDiscovery(change); err != nil {
			return err
		}
	}
	return nil
}

// Replace only the real header nav's contents, preserving all other HTML bytes.
func languageNavigationHTML(body []byte, current string, targets map[string]string) ([]byte, error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	offset, start, end, depth, found := 0, -1, -1, 0, 0
	inBody, inHeader, inTemplate := false, false, 0
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return nil, z.Err()
			}
			break
		}
		raw := z.Raw()
		position := offset
		offset += len(raw)
		token := z.Token()
		if kind == html.StartTagToken {
			switch token.Data {
			case "body":
				inBody = true
			case "header":
				inHeader = inBody
			case "template":
				inTemplate++
			case "nav":
				if depth > 0 {
					return nil, errors.New("nested language navigation")
				} else if inHeader && inTemplate == 0 {
					classSeen := false
					for _, attr := range token.Attr {
						if attr.Key != "class" {
							continue
						}
						if classSeen {
							return nil, errors.New("duplicate header navigation class attribute")
						}
						classSeen = true
						if navigationClass(attr.Val, "site-language-nav") {
							found++
							start, depth = offset, 1
						}
					}
				}
			}
		} else if kind == html.EndTagToken {
			switch token.Data {
			case "body":
				inBody = false
			case "header":
				inHeader = false
			case "template":
				inTemplate--
			case "nav":
				if depth > 0 {
					depth--
					if depth == 0 {
						end = position
					}
				}
			}
		}
	}
	if found == 0 {
		if current == "" {
			return body, nil // Undocumented import fragments have no header.
		}
		return nil, errors.New("missing header language navigation")
	}
	if found != 1 || start < 0 || end < start || depth != 0 {
		return nil, errors.New("expected exactly one complete header language navigation")
	}
	if current != "ca" && current != "en" && current != "es" {
		return nil, fmt.Errorf("unsupported page language %q", current)
	}
	var links strings.Builder
	for _, locale := range []struct{ code, label string }{{"ca", "Català"}, {"en", "English"}, {"es", "Castellano"}} {
		target := targets[locale.code]
		if target == "" {
			continue
		}
		fmt.Fprintf(&links, "\n        <a href=\"%s\" lang=\"%s\" hreflang=\"%s\"", stdhtml.EscapeString(target), locale.code, locale.code)
		if locale.code == current {
			links.WriteString(` aria-current="page"`)
		}
		fmt.Fprintf(&links, ">%s</a>", locale.label)
	}
	links.WriteString("\n    ")
	result := append(bytes.Clone(body[:start]), []byte(links.String())...)
	return append(result, body[end:]...), nil
}
