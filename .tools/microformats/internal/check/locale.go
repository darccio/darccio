package check

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// The English section must be complete before its navigation is exposed.
// Shared resources (CV, project docs, licences and canonical identity) keep
// their established URLs; only translated site navigation uses /en/.
func validateEnglishNavigation(deploy string) error {
	for _, slug := range []string{"", "about", "now", "projects", "talks", "contact", "hello", "feeds", "slashes", "colophon", "ai", "languages", "elsewhere", "sponsor", "uses", "blogroll", "changelog", "posts"} {
		file := filepath.Join(deploy, "en", filepath.FromSlash(slug), "index.html")
		if info, err := os.Stat(file); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("missing translated English page %s", file)
		}
	}
	type page struct {
		path, file, language string
		doc                  *html.Node
	}
	var pages []page
	languages := make(map[string]string)
	err := filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(file) != ".html" {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing language-navigation symlink %s", file)
		}
		input, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		doc, err := parseHTML(input)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(deploy, file)
		if err != nil {
			return err
		}
		root, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "html") }, "HTML root")
		if err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		language := attribute(root, "lang")
		if strings.HasPrefix(filepath.ToSlash(relative), "en/") && language != "en" {
			return fmt.Errorf("%s: English page lost its English language", relative)
		}
		path := "/" + filepath.ToSlash(relative)
		if filepath.Base(file) == "index.html" {
			path = strings.TrimSuffix(path, "index.html")
		}
		path = (&url.URL{Path: path}).EscapedPath()
		languages[path] = language
		pages = append(pages, page{path: path, file: relative, language: language, doc: doc})
		return nil
	})
	if err != nil {
		return err
	}
	translations, err := sitemapLanguageLinks(deploy)
	if err != nil {
		return err
	}
	for _, page := range pages {
		if page.language == "en" {
			if err := validateLocaleNavigation(page.doc, page.path); err != nil {
				return fmt.Errorf("%s: %w", page.file, err)
			}
		} else if len(find(page.doc, func(n *html.Node) bool {
			return hasClass(n, "site-language-nav") || hasClass(n, "site-logo") || hasClass(n, "primary-navigation")
		})) == 0 {
			continue // Raw vanity fragments have no site navigation.
		}
		links, err := validateSiteLanguageNavigation(page.doc, page.path, page.language)
		if err != nil {
			return fmt.Errorf("%s: %w", page.file, err)
		}
		for _, link := range links {
			target, _ := localLanguagePath(attribute(link, "href"))
			language := attribute(link, "lang")
			actual, exists := languages[target]
			if !exists {
				return fmt.Errorf("%s: %s language link has no deployed page %s", page.file, language, target)
			}
			if actual != language {
				return fmt.Errorf("%s: %s language link target %s uses language %q", page.file, language, target, actual)
			}
		}
		expected := map[string]string{"ca": "/", "en": "/en/"}
		for language, target := range translations[page.path] {
			if language == page.language && target != page.path {
				return fmt.Errorf("%s: sitemap current-language counterpart must point to this page", page.file)
			}
			expected[language] = target
		}
		expected[page.language] = page.path
		if len(links) != len(expected) {
			return fmt.Errorf("%s: language switch omitted or invented a translation", page.file)
		}
		for _, link := range links {
			language := attribute(link, "lang")
			if attribute(link, "href") != expected[language] {
				return fmt.Errorf("%s: %s language link must point to %s", page.file, language, expected[language])
			}
		}
	}
	return nil
}

// The assembler enriches reciprocal hreflang links from published siblings
// and reviewed exceptions. Check the same relationships without duplicating
// that source manifest here.
func sitemapLanguageLinks(deploy string) (map[string]map[string]string, error) {
	input, err := os.ReadFile(filepath.Join(deploy, "sitemap.xml"))
	if err != nil {
		return nil, err
	}
	var sitemap struct {
		Pages []struct {
			Location string `xml:"loc"`
			Links    []struct {
				Language string `xml:"hreflang,attr"`
				URL      string `xml:"href,attr"`
			} `xml:"http://www.w3.org/1999/xhtml link"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(input, &sitemap); err != nil {
		return nil, err
	}
	translations := make(map[string]map[string]string)
	for _, page := range sitemap.Pages {
		location, err := url.Parse(page.Location)
		if err != nil {
			return nil, err
		}
		links := make(map[string]string)
		for _, link := range page.Links {
			target, err := url.Parse(link.URL)
			if err != nil {
				return nil, err
			}
			if target.Scheme != location.Scheme || target.Host != location.Host || target.RawQuery != "" || target.Fragment != "" || links[link.Language] != "" {
				return nil, fmt.Errorf("invalid or duplicate sitemap language link for %s", page.Location)
			}
			links[link.Language] = target.EscapedPath()
		}
		translations[location.EscapedPath()] = links
	}
	return translations, nil
}

func localLanguagePath(href string) (string, error) {
	parsed, err := url.Parse(href)
	if err != nil || !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "//") || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", fmt.Errorf("language link must use a local root-relative page path: %q", href)
	}
	return parsed.EscapedPath(), nil
}

func validateSiteLanguageNavigation(doc *html.Node, pagePath, language string) ([]*html.Node, error) {
	languages, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "nav") && hasClass(n, "site-language-nav") }, "site languages")
	if err != nil {
		return nil, err
	}
	labels := map[string]string{"ca": "Idioma del web", "en": "Site language", "es": "Idioma del sitio"}
	if labels[language] == "" || attribute(languages, "aria-label") != labels[language] {
		return nil, fmt.Errorf("language switch is not labelled in the page language")
	}
	links := find(languages, func(n *html.Node) bool { return isTag(n, "a") })
	if len(links) < 2 || len(links) > 3 || (language == "es" && len(links) != 3) {
		return nil, fmt.Errorf("language switch must contain Catalan, English and optional Spanish")
	}
	for i, want := range []struct{ code, label string }{{"ca", "Català"}, {"en", "English"}, {"es", "Castellano"}}[:len(links)] {
		link := links[i]
		if attribute(link, "lang") != want.code || attribute(link, "hreflang") != want.code || normalized(textContent(link)) != strings.ToUpper(want.code) {
			return nil, fmt.Errorf("language switch needs unique ca, en, optional es links in order with visible CA, EN, ES codes")
		}
		if attribute(link, "aria-label") != want.label {
			return nil, fmt.Errorf("%s language link needs the accessible name %q", want.code, want.label)
		}
		if _, err := localLanguagePath(attribute(link, "href")); err != nil {
			return nil, err
		}
		current := attribute(link, "aria-current")
		if want.code == language {
			if current != "page" || attribute(link, "href") != pagePath {
				return nil, fmt.Errorf("current %s language link must point to this page with aria-current=page", language)
			}
		} else if current != "" {
			return nil, fmt.Errorf("only the current language link may have aria-current")
		}
	}
	return links, nil
}

func validateLocaleNavigation(doc *html.Node, pagePath string) error {
	page, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "html") }, "HTML root")
	if err != nil {
		return err
	}
	if attribute(page, "lang") != "en" {
		return fmt.Errorf("English page lost its English language")
	}
	brand, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "a") && hasClass(n, "site-logo") }, "site brand")
	if err != nil {
		return err
	}
	if attribute(brand, "href") != "/en/" || normalized(textContent(brand)) != owner {
		return fmt.Errorf("English brand must link to English home and retain the owner name")
	}
	skip, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "a") && hasClass(n, "skip-link") }, "skip link")
	if err != nil {
		return err
	}
	if attribute(skip, "href") != "#main-content" || normalized(textContent(skip)) != "Skip to content" || attribute(skip, "lang") != "en" {
		return fmt.Errorf("English skip link lost its language or target")
	}
	nav, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "nav") && hasClass(n, "primary-navigation") }, "primary navigation")
	if err != nil {
		return err
	}
	if attribute(nav, "lang") != "en" || attribute(nav, "aria-label") != "Main navigation" {
		return fmt.Errorf("English navigation is not labelled in English")
	}
	links := find(nav, func(n *html.Node) bool { return isTag(n, "a") })
	expected := []struct{ path, label string }{{"/en/", "Home"}, {"/en/posts/", "Articles"}, {"/en/projects/", "Projects"}, {"/en/talks/", "Talks"}, {"/en/contact/", "Contact"}}
	if len(links) != len(expected) {
		return fmt.Errorf("English navigation has %d links, want five", len(links))
	}
	section := ""
	switch {
	case pagePath == "/en/" || pagePath == "/en/about/":
		section = "/en/"
	case strings.HasPrefix(pagePath, "/en/posts/") || pagePath == "/en/feeds/":
		section = "/en/posts/"
	case pagePath == "/en/projects/" || strings.HasPrefix(pagePath, "/mergo/") || strings.HasPrefix(pagePath, "/darna/") || strings.HasPrefix(pagePath, "/zas/"):
		section = "/en/projects/"
	case pagePath == "/en/talks/":
		section = "/en/talks/"
	case pagePath == "/en/contact/" || pagePath == "/en/hello/":
		section = "/en/contact/"
	}
	for i, want := range expected {
		link := links[i]
		if attribute(link, "href") != want.path || normalized(textContent(link)) != want.label {
			return fmt.Errorf("English %s link must point to %s", want.label, want.path)
		}
		current := attribute(link, "aria-current")
		if (want.path == section && current != "page") || (want.path != section && current != "") {
			return fmt.Errorf("English %s link has incorrect current-section state", want.label)
		}
	}
	if _, err := validateSiteLanguageNavigation(doc, pagePath, "en"); err != nil {
		return err
	}
	footer, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "footer") && isTag(n.Parent, "body") }, "site footer")
	if err != nil {
		return err
	}
	footerLinks := find(footer, func(n *html.Node) bool { return isTag(n, "a") })
	for i, want := range []struct{ path, label string }{{"/en/feeds/", "Feeds"}, {"/en/slashes/", "All pages"}, {"/en/colophon/", "About this site"}} {
		if len(footerLinks) <= i || attribute(footerLinks[i], "href") != want.path || normalized(textContent(footerLinks[i])) != want.label {
			return fmt.Errorf("English footer %s link must point to %s", want.label, want.path)
		}
	}
	return nil
}
