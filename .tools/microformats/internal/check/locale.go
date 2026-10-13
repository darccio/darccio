package check

import (
	"fmt"
	"io/fs"
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
	return filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(file) != ".html" {
			return nil
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
		if len(find(doc, func(n *html.Node) bool { return isTag(n, "html") && attribute(n, "lang") == "en" })) == 0 {
			if strings.HasPrefix(filepath.ToSlash(relative), "en/") {
				return fmt.Errorf("%s: English page lost its English language", relative)
			}
			return nil
		}
		path := "/" + strings.TrimSuffix(filepath.ToSlash(relative), "index.html")
		if err := validateLocaleNavigation(doc, path); err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		return nil
	})
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
	languages, err := uniqueNode(doc, func(n *html.Node) bool { return isTag(n, "nav") && hasClass(n, "site-language-nav") }, "site languages")
	if err != nil {
		return err
	}
	if attribute(languages, "aria-label") != "Site language" {
		return fmt.Errorf("English language switch is not labelled in English")
	}
	languageLinks := find(languages, func(n *html.Node) bool { return isTag(n, "a") })
	if len(languageLinks) != 2 || attribute(languageLinks[0], "href") != "/" || attribute(languageLinks[0], "hreflang") != "ca" || attribute(languageLinks[0], "lang") != "ca" || attribute(languageLinks[0], "aria-current") != "" || attribute(languageLinks[1], "href") != "/en/" || attribute(languageLinks[1], "hreflang") != "en" || attribute(languageLinks[1], "lang") != "en" || attribute(languageLinks[1], "aria-current") != "true" {
		return fmt.Errorf("English language switch lost its destinations or current language")
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
