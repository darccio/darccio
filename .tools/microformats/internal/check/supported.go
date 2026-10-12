package check

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darccio/zas"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
	"willnorris.com/go/microformats"
)

// These are existing public pages with supported facts. Deliberately do not
// walk CV or derive events for undated talks.
func validateSupportedPages(root, deploy string) error {
	for _, slug := range []string{"now", "ai", "colophon", "feeds", "contact", "hello", "elsewhere", "languages", "sponsor", "uses", "blogroll"} {
		pageURL := homeURL + slug + "/"
		config, expected, err := publicSource(root, slug+"/index.md")
		if err != nil {
			return err
		}
		if config["entry"] != true {
			return fmt.Errorf("%s: supported dated slash page must retain entry metadata", slug)
		}
		data := zas.ZasData{Page: config, Path: "/" + slug + "/index.html", Site: zas.ZasSiteData{BaseURL: strings.TrimSuffix(homeURL, "/")}}
		updated, err := data.Updated()
		if err != nil || updated.IsZero() {
			return fmt.Errorf("%s: missing explicit source updated timestamp", slug)
		}
		doc, parsed, err := readPage(deploy, slug+"/index.html", pageURL)
		if err != nil {
			return err
		}
		if err := validateSlashEntry(doc, parsed, expected, data.Canonical(), updated); err != nil {
			return fmt.Errorf("%s: %w", slug, err)
		}
	}
	for _, spec := range []struct{ slug, childType string }{{"projects", "h-item"}, {"slashes", "h-entry"}, {"talks", "h-event"}} {
		pageURL := homeURL + spec.slug + "/"
		_, expected, err := publicSource(root, spec.slug+"/index.md")
		if err != nil {
			return err
		}
		base, _ := url.Parse(pageURL)
		source := microformats.ParseNode(expected, base)
		doc, output, err := readPage(deploy, spec.slug+"/index.html", pageURL)
		if err != nil {
			return err
		}
		if err := validateSupportedChildren(spec.slug, spec.childType, pageURL, expected, doc, source, output); err != nil {
			return fmt.Errorf("%s: %w", spec.slug, err)
		}
	}
	return nil
}

func publicSource(root, file string) (map[interface{}]interface{}, *html.Node, error) {
	input, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil, nil, err
	}
	config, markdown, err := sourceConfig(input)
	if err != nil {
		return nil, nil, err
	}
	if bytes.Contains(markdown, []byte("{{")) {
		return nil, nil, fmt.Errorf("%s: unsupported template action in validation source", file)
	}
	var body bytes.Buffer
	converter := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote), goldmark.WithRendererOptions(goldhtml.WithUnsafe()))
	if err := converter.Convert(markdown, &body); err != nil {
		return nil, nil, err
	}
	doc, err := parseHTML(body.Bytes())
	if err != nil {
		return nil, nil, err
	}
	for _, embed := range find(doc, func(n *html.Node) bool { return isTag(n, "embed") }) {
		// Contact has one literal, public HTML fragment. No scripts or templates
		// are executed, and no other private source may be read by this adapter.
		if file != "contact/index.md" || attribute(embed, "src") != "../.zas/partials/contact.html" || attribute(embed, "type") != "text/html" {
			return nil, nil, fmt.Errorf("%s: unsupported public embed", file)
		}
		input, err := os.ReadFile(filepath.Join(root, ".zas", "partials", "contact.html"))
		if err != nil {
			return nil, nil, err
		}
		fragment, err := parseHTML(input)
		if err != nil {
			return nil, nil, err
		}
		body, err := uniqueNode(fragment, func(n *html.Node) bool { return isTag(n, "body") }, "embedded HTML body")
		if err != nil {
			return nil, nil, err
		}
		for child := body.FirstChild; child != nil; {
			next := child.NextSibling
			body.RemoveChild(child)
			embed.Parent.InsertBefore(child, embed)
			child = next
		}
		embed.Parent.RemoveChild(embed)
	}
	return config, doc, nil
}

func validateSlashEntry(doc *html.Node, data *microformats.Data, expected *html.Node, canonical string, updated time.Time) error {
	entries := itemsOfType(data.Items, "h-entry")
	if len(entries) != 1 {
		return fmt.Errorf("expected one h-entry, got %d", len(entries))
	}
	entry := entries[0]
	heading, err := uniqueNode(expected, func(n *html.Node) bool { return isTag(n, "h1") }, "source H1")
	if err != nil {
		return err
	}
	for key, value := range map[string]string{"name": textContent(heading), "url": canonical, "uid": canonical} {
		if err := property(entry, key, value); err != nil {
			return err
		}
	}
	if err := author(entry); err != nil {
		return err
	}
	article, err := uniqueNode(doc, func(n *html.Node) bool { return hasClass(n, "h-entry") }, "slash h-entry")
	if err != nil {
		return err
	}
	if attribute(article, "lang") != "ca" {
		return fmt.Errorf("slash entry must keep Catalan language")
	}
	content, err := uniqueNode(article, func(n *html.Node) bool { return hasClass(n, "e-content") }, "slash e-content")
	if err != nil {
		return err
	}
	if fingerprint(content, canonical) != fingerprint(expected, canonical) {
		return fmt.Errorf("slash content differs from its public source")
	}
	if err := validateTime(article, entry, "updated", updated); err != nil {
		return err
	}
	return validateTime(article, entry, "published", time.Time{})
}

func validateSupportedChildren(slug, typ, pageURL string, expected, doc *html.Node, source, output *microformats.Data) error {
	want := itemsOfType(source.Items, typ)
	got := itemsOfType(output.Items, typ)
	if len(want) == 0 || len(got) != len(want) {
		return fmt.Errorf("%s count got %d, want %d nonempty source entries", typ, len(got), len(want))
	}
	if slug != "talks" {
		feeds := itemsOfType(output.Items, "h-feed")
		if len(feeds) != 1 {
			return fmt.Errorf("expected one h-feed")
		}
		if err := property(feeds[0], "url", pageURL); err != nil {
			return err
		}
		if err := author(feeds[0]); err != nil {
			return err
		}
	} else {
		wantHeadings := find(expected, func(n *html.Node) bool { return isTag(n, "h3") })
		gotHeadings := find(doc, func(n *html.Node) bool { return isTag(n, "h3") })
		if len(wantHeadings) != len(gotHeadings) {
			return fmt.Errorf("talk catalogue count differs from source")
		}
	}
	seen := make(map[string]bool)
	for _, item := range got {
		urls := propertyStrings(item, "url")
		if len(urls) != 1 || seen[urls[0]] {
			return fmt.Errorf("%s missing, ambiguous or duplicate URL: %v", typ, urls)
		}
		seen[urls[0]] = true
		var original *microformats.Microformat
		for _, candidate := range want {
			if contains(propertyStrings(candidate, "url"), urls[0]) {
				original = candidate
				break
			}
		}
		if original == nil {
			return fmt.Errorf("unexpected %s URL %s", typ, urls[0])
		}
		keys := []string{"name"}
		switch slug {
		case "projects":
			keys = append(keys, "summary")
		case "slashes":
			keys = append(keys, "updated")
		case "talks":
			keys = append(keys, "start", "location")
		}
		for _, key := range keys {
			values := propertyStrings(original, key)
			if len(values) != 1 || normalized(values[0]) == "" {
				return fmt.Errorf("source %s must have one nonempty %s", typ, key)
			}
			if err := property(item, key, values[0]); err != nil {
				return err
			}
			if key == "start" || key == "updated" {
				if _, err := time.Parse("2006-01-02", values[0]); err != nil {
					return fmt.Errorf("supported %s must preserve known day precision", key)
				}
			}
		}
	}
	return nil
}
