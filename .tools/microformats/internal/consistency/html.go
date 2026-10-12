package consistency

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"
	"willnorris.com/go/microformats"
)

type htmlNode = html.Node

func parseHTML(input []byte) (*htmlNode, error) { return html.Parse(bytes.NewReader(input)) }

func find(node *htmlNode, predicate func(*htmlNode) bool) []*htmlNode {
	var result []*htmlNode
	var walk func(*htmlNode)
	walk = func(node *htmlNode) {
		if predicate(node) {
			result = append(result, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return result
}

func isTag(node *htmlNode, tag string) bool { return node.Type == html.ElementNode && node.Data == tag }

func attribute(node *htmlNode, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func hasClass(node *htmlNode, value string) bool {
	return slices.Contains(strings.Fields(attribute(node, "class")), value)
}

func normalized(value string) string { return strings.Join(strings.Fields(value), " ") }

func text(node *htmlNode) string {
	var result strings.Builder
	var walk func(*htmlNode)
	walk = func(node *htmlNode) {
		if node.Type == html.TextNode {
			result.WriteString(node.Data)
			return
		}
		if isTag(node, "script") || isTag(node, "style") {
			return
		}
		block := isTag(node, "br") || isTag(node, "p") || isTag(node, "li") || isTag(node, "pre") || isTag(node, "blockquote") || strings.HasPrefix(node.Data, "h") && len(node.Data) == 2
		if block {
			result.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if block {
			result.WriteByte(' ')
		}
	}
	walk(node)
	return normalized(result.String())
}

func one(values []string, expected, label string) error {
	if len(values) != 1 || values[0] != expected {
		return fmt.Errorf("%s got %q, want one %q", label, values, expected)
	}
	return nil
}

func property(entry *microformats.Microformat, key string) ([]string, error) {
	var result []string
	for _, value := range entry.Properties[key] {
		value, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("h-entry %s must be plain strings", key)
		}
		result = append(result, value)
	}
	return result, nil
}

func instant(values []string, expected time.Time, label string) error {
	if expected.IsZero() {
		if len(values) != 0 {
			return fmt.Errorf("%s invents an unknown instant: %q", label, values)
		}
		return nil
	}
	if len(values) != 1 {
		return fmt.Errorf("%s needs exactly one explicit instant, got %q", label, values)
	}
	got, err := time.Parse(time.RFC3339Nano, values[0])
	if err != nil || !got.Equal(expected) {
		return fmt.Errorf("%s got %q, want instant %s", label, values[0], expected.Format(time.RFC3339Nano))
	}
	return nil
}

func meta(doc *htmlNode, name string) []string {
	var values []string
	for _, node := range find(doc, func(n *htmlNode) bool {
		return isTag(n, "meta") && (attribute(n, "name") == name || attribute(n, "property") == name)
	}) {
		values = append(values, attribute(node, "content"))
	}
	return values
}

type renderedPost struct {
	Text string
}

func checkArticle(post sourcePost, input []byte) (renderedPost, error) {
	doc, err := parseHTML(input)
	if err != nil {
		return renderedPost{}, err
	}
	canonicals := find(doc, func(n *htmlNode) bool {
		return isTag(n, "link") && slices.Contains(strings.Fields(attribute(n, "rel")), "canonical")
	})
	var canonicalValues []string
	for _, node := range canonicals {
		canonicalValues = append(canonicalValues, attribute(node, "href"))
	}
	if err := one(canonicalValues, post.URL, "canonical URL"); err != nil {
		return renderedPost{}, err
	}
	for name, want := range map[string]string{"og:url": post.URL, "og:type": "article", "description": post.Summary, "og:description": post.Summary, "og:title": post.OpenGraphTitle} {
		if err := one(meta(doc, name), want, name); err != nil {
			return renderedPost{}, err
		}
	}
	if err := instant(meta(doc, "article:published_time"), post.Date, "Open Graph publication"); err != nil {
		return renderedPost{}, err
	}
	base, err := url.Parse(post.URL)
	if err != nil {
		return renderedPost{}, err
	}
	parsed := microformats.ParseNode(doc, base)
	var entries []*microformats.Microformat
	var collect func([]*microformats.Microformat)
	collect = func(items []*microformats.Microformat) {
		for _, item := range items {
			if slices.Contains(item.Type, "h-entry") {
				entries = append(entries, item)
			}
			collect(item.Children)
		}
	}
	collect(parsed.Items)
	if len(entries) != 1 {
		return renderedPost{}, fmt.Errorf("expected one parsed h-entry, got %d", len(entries))
	}
	entry := entries[0]
	for key, want := range map[string]string{"name": post.Title, "url": post.URL, "uid": post.URL, "summary": post.Summary} {
		values, err := property(entry, key)
		if err != nil {
			return renderedPost{}, err
		}
		if err := one(values, want, "h-entry "+key); err != nil {
			return renderedPost{}, err
		}
	}
	for key, want := range map[string]time.Time{"published": post.Date, "updated": post.Updated} {
		values, err := property(entry, key)
		if err != nil {
			return renderedPost{}, err
		}
		if err := instant(values, want, "h-entry "+key); err != nil {
			return renderedPost{}, err
		}
	}
	tags, err := property(entry, "category")
	if err != nil || !slices.Equal(tags, post.Tags) {
		return renderedPost{}, fmt.Errorf("h-entry categories differ from source: %q", tags)
	}
	articles := find(doc, func(n *htmlNode) bool { return hasClass(n, "h-entry") })
	if len(articles) != 1 || attribute(articles[0], "lang") != post.Language {
		return renderedPost{}, fmt.Errorf("h-entry language differs from source %q", post.Language)
	}
	content := find(articles[0], func(n *htmlNode) bool { return hasClass(n, "e-content") })
	if len(content) != 1 || text(content[0]) == "" {
		return renderedPost{}, fmt.Errorf("missing nonempty article e-content")
	}
	if err := checkJSONLD(doc, post); err != nil {
		return renderedPost{}, err
	}
	return renderedPost{Text: text(content[0])}, nil
}

func checkJSONLD(doc *htmlNode, post sourcePost) error {
	scripts := find(doc, func(n *htmlNode) bool { return isTag(n, "script") && attribute(n, "type") == "application/ld+json" })
	if len(scripts) != 1 {
		return fmt.Errorf("expected one JSON-LD script, got %d", len(scripts))
	}
	var raw strings.Builder
	for child := scripts[0].FirstChild; child != nil; child = child.NextSibling {
		raw.WriteString(child.Data)
	}
	var graph struct {
		Context string           `json:"@context"`
		Graph   []map[string]any `json:"@graph"`
	}
	if err := decodeJSON([]byte(raw.String()), &graph); err != nil {
		return fmt.Errorf("JSON-LD: %w", err)
	}
	if graph.Context != "https://schema.org" {
		return fmt.Errorf("JSON-LD context must be Schema.org")
	}
	var article map[string]any
	for _, node := range graph.Graph {
		if node["@type"] == "BlogPosting" {
			if article != nil {
				return fmt.Errorf("duplicate BlogPosting")
			}
			article = node
		}
	}
	if article == nil {
		return fmt.Errorf("missing BlogPosting")
	}
	for key, want := range map[string]string{"url": post.URL, "headline": post.Title, "description": post.Summary, "inLanguage": post.Language} {
		if article[key] != want {
			return fmt.Errorf("BlogPosting %s got %v, want %q", key, article[key], want)
		}
	}
	for key, want := range map[string]time.Time{"datePublished": post.Date, "dateModified": post.Updated} {
		var values []string
		if value, present := article[key]; present {
			value, ok := value.(string)
			if !ok {
				return fmt.Errorf("BlogPosting %s must be a timestamp string", key)
			}
			values = []string{value}
		}
		if err := instant(values, want, "BlogPosting "+key); err != nil {
			return err
		}
	}
	var tags []string
	input, err := json.Marshal(article["keywords"])
	if err != nil || json.Unmarshal(input, &tags) != nil || !slices.Equal(tags, post.Tags) {
		return fmt.Errorf("BlogPosting keywords differ from source")
	}
	return nil
}
