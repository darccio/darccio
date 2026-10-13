package check

import (
	"fmt"
	"strings"

	"willnorris.com/go/microformats"
)

var externalArticles = map[string]struct{ title, language string }{
	"https://www.elsaltodiario.com/opinion/queer-marxismo-transfobia-contra-compartimentacion": {"Contra la compartimentación", "es"},
	"https://www.nuvol.com/pantalles/per-al-lusions-22931":                                     {"Per al·lusions", "ca"},
}

func validateArchive(archiveURL, language string, posts []Post, doc *htmlNode, data *microformats.Data) error {
	feeds := itemsOfType(data.Items, "h-feed")
	if len(feeds) != 1 {
		return fmt.Errorf("expected one h-feed, got %d", len(feeds))
	}
	feed := feeds[0]
	if len(propertyStrings(feed, "name")) != 1 || normalized(propertyStrings(feed, "name")[0]) == "" {
		return fmt.Errorf("feed must have exactly one nonempty name")
	}
	if err := property(feed, "url", archiveURL); err != nil {
		return err
	}
	if err := author(feed); err != nil {
		return err
	}
	feedNode, err := uniqueNode(doc, func(n *htmlNode) bool { return hasClass(n, "h-feed") }, "h-feed element")
	if err != nil {
		return err
	}
	pageLanguage := language
	if pageLanguage == "" {
		pageLanguage = "ca"
	}
	expected := make(map[string]Post)
	for _, post := range posts {
		if language == "" || post.Language == language {
			expected[post.URL] = post
		}
	}
	seen := make(map[string]bool)
	entries := itemsOfType(feed.Children, "h-entry")
	if len(entries) != len(expected) {
		return fmt.Errorf("archive has %d local entries, want %d", len(entries), len(expected))
	}
	for _, entry := range entries {
		values := propertyStrings(entry, "url")
		if len(values) != 1 {
			return fmt.Errorf("archive entry must have one URL, got %v", values)
		}
		post, ok := expected[values[0]]
		if !ok {
			return fmt.Errorf("unexpected local article URL %s", values[0])
		}
		if seen[post.URL] {
			return fmt.Errorf("duplicate archive article %s", post.URL)
		}
		seen[post.URL] = true
		if err := property(entry, "name", post.Title); err != nil {
			return fmt.Errorf("%s: %w", post.URL, err)
		}
		if err := property(entry, "uid", post.URL); err != nil {
			return err
		}
		node, err := archiveItemNode(feedNode, "h-entry", post.URL, archiveURL)
		if err != nil {
			return err
		}
		if err := archiveLanguage(node, post.Language, pageLanguage); err != nil {
			return err
		}
		if err := validateTime(node, entry, "published", post.Date); err != nil {
			return fmt.Errorf("%s: %w", post.URL, err)
		}
		if post.Date.IsZero() {
			calendar := find(node, func(n *htmlNode) bool {
				return isTag(n, "time") && attribute(n, "datetime") == post.CalendarDate && !hasClass(n, "dt-published")
			})
			if len(calendar) != 1 {
				return fmt.Errorf("%s: missing visibly sourced partial calendar date", post.URL)
			}
		}
	}
	for pageURL := range expected {
		if !seen[pageURL] {
			return fmt.Errorf("missing archive article %s", pageURL)
		}
	}
	cites := itemsOfType(feed.Children, "h-cite")
	if language != "" {
		if len(cites) != 0 {
			return fmt.Errorf("language archive has unexpected external articles")
		}
		return nil
	}
	if len(cites) != len(externalArticles) {
		return fmt.Errorf("root archive has %d external citations, want %d", len(cites), len(externalArticles))
	}
	seenCites := make(map[string]bool)
	for _, cite := range cites {
		values := propertyStrings(cite, "url")
		if len(values) != 1 {
			return fmt.Errorf("external citation must have one URL")
		}
		external, ok := externalArticles[values[0]]
		if !ok || seenCites[values[0]] {
			return fmt.Errorf("unexpected or duplicate external citation %s", values[0])
		}
		seenCites[values[0]] = true
		if err := property(cite, "name", external.title); err != nil {
			return err
		}
		node, err := archiveItemNode(feedNode, "h-cite", values[0], archiveURL)
		if err != nil {
			return err
		}
		if err := archiveLanguage(node, external.language, pageLanguage); err != nil {
			return err
		}
	}
	return nil
}

func archiveItemNode(feed *htmlNode, class, pageURL, archiveURL string) (*htmlNode, error) {
	return uniqueNode(feed, func(n *htmlNode) bool {
		if !hasClass(n, class) {
			return false
		}
		links := find(n, func(a *htmlNode) bool {
			return hasClass(a, "u-url") && resolve(archiveURL, attribute(a, "href")) == pageURL
		})
		return len(links) != 0
	}, "archive item for "+pageURL)
}

var archiveLanguageNames = map[string]map[string]string{
	"ca": {"ca": "en català", "en": "en anglès", "es": "en castellà"},
	"en": {"ca": "in Catalan", "en": "in English", "es": "in Spanish"},
	"es": {"ca": "en catalán", "en": "en inglés", "es": "en español"},
}

func archiveLanguage(node *htmlNode, language, pageLanguage string) error {
	if attribute(node, "lang") != language {
		return fmt.Errorf("archive item lang got %q, want %q", attribute(node, "lang"), language)
	}
	name := archiveLanguageNames[pageLanguage][language]
	if name == "" {
		return fmt.Errorf("unsupported archive language %q on %q page", language, pageLanguage)
	}
	badges := find(node, func(n *htmlNode) bool { return hasClass(n, "lang-tag") })
	if len(badges) != 1 || !isTag(badges[0], "span") || textContent(badges[0]) != language || attribute(badges[0], "aria-hidden") != "true" {
		return fmt.Errorf("archive item must have one decorative %s language badge with aria-hidden=true", language)
	}
	spoken := find(node, func(n *htmlNode) bool { return hasClass(n, "visually-hidden") })
	if len(spoken) != 1 || !isTag(spoken[0], "span") || textContent(spoken[0]) != "("+name+")" || attribute(spoken[0], "lang") != pageLanguage {
		return fmt.Errorf("archive item must have one spoken (%s) label with lang=%s", name, pageLanguage)
	}
	for _, label := range []*htmlNode{badges[0], spoken[0]} {
		for ancestor := label; ancestor != nil; ancestor = ancestor.Parent {
			style := strings.ToLower(strings.Join(strings.Fields(attribute(ancestor, "style")), ""))
			if hasClass(ancestor, "p-name") {
				return fmt.Errorf("archive language labels must be outside p-name")
			}
			if hasAttribute(ancestor, "hidden") || strings.Contains(style, "display:none") || strings.Contains(style, "visibility:hidden") || strings.Contains(style, "visibility:collapse") {
				return fmt.Errorf("archive %s language label has a hidden ancestor", language)
			}
			if label == badges[0] && hasClass(ancestor, "visually-hidden") {
				return fmt.Errorf("archive %s badge is visually hidden", language)
			}
			if label == spoken[0] && (strings.EqualFold(strings.TrimSpace(attribute(ancestor, "aria-hidden")), "true") || hasAttribute(ancestor, "inert")) {
				return fmt.Errorf("archive (%s) label is hidden from assistive technology", name)
			}
		}
	}
	return nil
}
