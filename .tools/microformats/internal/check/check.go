package check

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"willnorris.com/go/microformats"
	"willnorris.com/go/microformats/rhc"
)

const homeURL = "https://dario.cat/"
const owner = "Dario Castañé"
const email = "cc@dario.cat"

var profiles = []string{
	"https://bsky.app/profile/dario.cat",
	"https://mastodont.cat/@dario",
	"https://github.com/darccio",
	"https://linkedin.com/in/darccio",
	"https://sifa.id/p/dario.cat",
	"https://rizoma.dario.cat/",
	"https://speakerdeck.com/dario",
}

type Report struct{ Posts, Dated int }

// Directory checks a completed deployment before it replaces the previous
// output. Source paths and deployed paths are deliberately separate arguments.
func Directory(root, deploy string) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	resume, err := loadResume(root)
	if err != nil {
		return Report{}, fmt.Errorf("resume: %w", err)
	}
	inv, err := loadInventory(root)
	if err != nil {
		return Report{}, err
	}
	var failures []error
	check := func(label string, err error) {
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", label, err))
		}
	}
	for _, file := range []string{"index.html", "about/index.html"} {
		sourceURL := homeURL
		if file == "about/index.html" {
			sourceURL += "about/"
		}
		doc, data, err := readPage(deploy, file, sourceURL)
		if err == nil {
			card := rhc.RepresentativeHcard(data, sourceURL)
			if card == nil {
				err = fmt.Errorf("no representative h-card")
			} else {
				err = validateCard(card, true)
			}
			if err == nil {
				err = validateProfiles(doc, data)
			}
		}
		check(file, err)
	}
	report := Report{Posts: len(inv.Posts)}
	for _, post := range inv.Posts {
		if !post.Date.IsZero() {
			report.Dated++
		}
		doc, data, err := readPage(deploy, post.File, post.URL)
		if err == nil {
			err = validatePost(post, doc, data)
		}
		check(post.File, err)
	}
	for _, post := range inv.Excluded {
		_, data, err := readPage(deploy, post.File, post.URL)
		if os.IsNotExist(err) {
			continue // Unpublished sources do not have output.
		}
		if err == nil && len(itemsOfType(data.Items, "h-entry")) != 0 {
			err = fmt.Errorf("draft or unpublished source exposed as h-entry")
		}
		check(post.File, err)
	}
	for _, archive := range []struct{ file, language string }{{"posts/index.html", ""}, {"posts/ca/index.html", "ca"}, {"en/posts/index.html", "en"}, {"es/posts/index.html", "es"}} {
		archiveURL := homeURL + strings.TrimSuffix(archive.file, "index.html")
		doc, data, err := readPage(deploy, archive.file, archiveURL)
		if err == nil {
			err = validateArchive(archiveURL, archive.language, inv.Posts, doc, data)
		}
		check(archive.file, err)
	}
	check("supported slash pages, projects and talks", validateSupportedPages(root, deploy))
	check("configured resume", validateResume(deploy, resume))
	check("shared JSON-LD", validateJSONLDPages(deploy, inv, resume != nil))
	for _, path := range []string{".tools", "docs/site-plan", "docs/HANDOFF-FACTS.md"} {
		if _, err := os.Stat(filepath.Join(deploy, filepath.FromSlash(path))); !os.IsNotExist(err) {
			check(path, fmt.Errorf("private tooling or planning path present in deployment"))
		}
	}
	return report, errors.Join(failures...)
}

func readPage(deploy, file, pageURL string) (*htmlNode, *microformats.Data, error) {
	input, err := os.ReadFile(filepath.Join(deploy, filepath.FromSlash(file)))
	if err != nil {
		return nil, nil, err
	}
	doc, err := parseHTML(input)
	if err != nil {
		return nil, nil, err
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, nil, err
	}
	return doc, microformats.ParseNode(doc, base), nil
}

func validateCard(card *microformats.Microformat, full bool) error {
	if !hasType(card, "h-card") {
		return fmt.Errorf("identity is not an h-card")
	}
	if err := property(card, "name", owner); err != nil {
		return err
	}
	if !contains(propertyStrings(card, "url"), homeURL) {
		return fmt.Errorf("identity lacks homepage URL")
	}
	if err := property(card, "uid", homeURL); err != nil {
		return err
	}
	photos := propertyStrings(card, "photo")
	if !contains(photos, homeURL+"static/avatar.webp") && !contains(photos, homeURL+"static/avatar-96.webp") {
		return fmt.Errorf("identity lacks a self-hosted avatar: %v", photos)
	}
	if full {
		if err := property(card, "email", "mailto:"+email); err != nil {
			return err
		}
	}
	return nil
}

func validateProfiles(doc *htmlNode, data *microformats.Data) error {
	head, err := uniqueNode(doc, func(n *htmlNode) bool { return isTag(n, "head") }, "page head")
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if !contains(data.Rels["me"], profile) {
			return fmt.Errorf("missing rel=me profile %s", profile)
		}
		links := find(head, func(n *htmlNode) bool {
			return isTag(n, "link") && contains(strings.Fields(attribute(n, "rel")), "me") && attribute(n, "href") == profile
		})
		if len(links) != 1 {
			return fmt.Errorf("expected one head rel=me link for %s, got %d", profile, len(links))
		}
	}
	return nil
}

func validatePost(post Post, doc *htmlNode, data *microformats.Data) error {
	entries := itemsOfType(data.Items, "h-entry")
	if len(entries) != 1 {
		return fmt.Errorf("expected one h-entry, got %d", len(entries))
	}
	entry := entries[0]
	for key, expected := range map[string]string{"name": post.Title, "url": post.URL, "uid": post.URL, "summary": post.Summary} {
		if err := property(entry, key, expected); err != nil {
			return err
		}
	}
	if err := author(entry); err != nil {
		return err
	}
	if !slices.Equal(propertyStrings(entry, "category"), post.Tags) {
		return fmt.Errorf("categories disagree with source tags: got %v, want %v", propertyStrings(entry, "category"), post.Tags)
	}
	article, err := uniqueNode(doc, func(n *htmlNode) bool { return hasClass(n, "h-entry") }, "article h-entry element")
	if err != nil {
		return err
	}
	if attribute(article, "lang") != post.Language {
		return fmt.Errorf("article language got %q, want %q", attribute(article, "lang"), post.Language)
	}
	root, err := uniqueNode(doc, func(n *htmlNode) bool { return isTag(n, "html") }, "HTML root")
	if err != nil || attribute(root, "lang") != post.Language {
		return fmt.Errorf("HTML root language does not match source %q", post.Language)
	}
	heading, err := uniqueNode(doc, func(n *htmlNode) bool { return isTag(n, "h1") }, "page H1")
	if err != nil || !hasClass(heading, "p-name") || textContent(heading) != normalized(post.Title) {
		return fmt.Errorf("sole H1 must be article p-name %q", post.Title)
	}
	content, err := uniqueNode(article, func(n *htmlNode) bool { return hasClass(n, "e-content") }, "article e-content")
	if err != nil {
		return err
	}
	if len(find(content, func(n *htmlNode) bool {
		return isTag(n, "nav") || isTag(n, "header") || isTag(n, "footer") || isTag(n, "script")
	})) != 0 {
		return fmt.Errorf("article content contains page chrome or scripts")
	}
	values := entry.Properties["content"]
	if len(values) != 1 {
		return fmt.Errorf("expected one parsed content value, got %d", len(values))
	}
	parsed, ok := values[0].(map[string]string)
	if !ok || normalized(parsed["value"]) == "" || strings.TrimSpace(parsed["html"]) == "" {
		return fmt.Errorf("parsed content must contain both text and HTML")
	}
	expected, err := parseHTML(post.Body)
	if err != nil {
		return err
	}
	if fingerprint(content, post.URL) != fingerprint(expected, post.URL) {
		return fmt.Errorf("article content differs from its public Markdown source")
	}
	for name, date := range map[string]time.Time{"published": post.Date, "updated": post.Updated} {
		if err := validateTime(article, entry, name, date); err != nil {
			return err
		}
	}
	if err := oneValue(meta(doc, "og:url"), post.URL, "Open Graph URL"); err != nil {
		return err
	}
	if err := oneValue(meta(doc, "og:type"), "article", "Open Graph type"); err != nil {
		return err
	}
	if post.Date.IsZero() {
		if len(meta(doc, "article:published_time")) != 0 {
			return fmt.Errorf("unknown publication instant emitted in Open Graph")
		}
	} else if err := oneValue(meta(doc, "article:published_time"), post.Date.Format(time.RFC3339Nano), "Open Graph publication date"); err != nil {
		return err
	}
	if err := oneValue(meta(doc, "description"), post.Summary, "meta description"); err != nil {
		return err
	}
	if err := oneValue(meta(doc, "og:description"), post.Summary, "Open Graph description"); err != nil {
		return err
	}
	canonicals := find(doc, func(n *htmlNode) bool { return isTag(n, "link") && attribute(n, "rel") == "canonical" })
	if len(canonicals) != 1 || attribute(canonicals[0], "href") != post.URL {
		return fmt.Errorf("canonical link does not match article URL")
	}
	// Posts have an identity card in the page header. Representative discovery
	// is meaningful for home/about; on a post its stable URL remains the home.
	headers := find(doc, func(n *htmlNode) bool { return isTag(n, "header") })
	if len(headers) == 0 {
		return fmt.Errorf("missing page header identity")
	}
	base, _ := url.Parse(post.URL)
	cards := itemsOfType(microformats.ParseNode(headers[0], base).Items, "h-card")
	if len(cards) != 1 {
		return fmt.Errorf("expected one header h-card, got %d", len(cards))
	}
	if err := validateCard(cards[0], false); err != nil {
		return err
	}
	return validateProfiles(doc, data)
}

func author(entry *microformats.Microformat) error {
	values := entry.Properties["author"]
	if len(values) != 1 {
		return fmt.Errorf("expected one author h-card, got %d", len(values))
	}
	card, ok := values[0].(*microformats.Microformat)
	if !ok || !hasType(card, "h-card") {
		return fmt.Errorf("author must be a nested h-card")
	}
	if err := property(card, "name", owner); err != nil {
		return err
	}
	if !contains(propertyStrings(card, "url"), homeURL) {
		return fmt.Errorf("author must link to the owner's homepage")
	}
	return nil
}

func validateTime(node *htmlNode, item *microformats.Microformat, name string, date time.Time) error {
	dom := find(node, func(n *htmlNode) bool { return hasClass(n, "dt-"+name) })
	values := propertyStrings(item, name)
	if date.IsZero() {
		if len(values) != 0 || len(dom) != 0 {
			return fmt.Errorf("%s must be omitted for an unknown instant", name)
		}
		return nil
	}
	expected := date.Format(time.RFC3339Nano)
	if len(dom) != 1 || attribute(dom[0], "datetime") != expected {
		return fmt.Errorf("%s datetime must retain source timestamp and offset %q", name, expected)
	}
	if len(values) != 1 || values[0] != expected {
		return fmt.Errorf("parsed %s must equal %q, got %q", name, expected, values)
	}
	return nil
}
