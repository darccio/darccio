package check

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"willnorris.com/go/microformats"
)

type htmlNode = html.Node

func parseHTML(input []byte) (*html.Node, error) {
	return html.Parse(bytes.NewReader(input))
}

func isTag(node *html.Node, tag string) bool {
	return node.Type == html.ElementNode && node.Data == tag
}

func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func hasAttribute(node *html.Node, key string) bool {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return true
		}
	}
	return false
}

func hasClass(node *html.Node, class string) bool {
	for _, value := range strings.Fields(attribute(node, "class")) {
		if value == class {
			return true
		}
	}
	return false
}

func find(root *html.Node, matches func(*html.Node) bool) []*html.Node {
	var result []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if matches(node) {
			result = append(result, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return result
}

func normalized(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func textContent(root *html.Node) string {
	var body strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.CommentNode || isTag(node, "script") || isTag(node, "style") {
			return
		}
		if node.Type == html.TextNode {
			body.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return normalized(body.String())
}

func resolve(base, reference string) string {
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	r, err := url.Parse(reference)
	if err != nil {
		return ""
	}
	return u.ResolveReference(r).String()
}

// Fingerprint checks the source-derived text and every link/image destination.
// Presentation-only classes and serialization choices do not affect it.
func fingerprint(node *html.Node, base string) string {
	var urls []string
	for _, child := range find(node, func(n *html.Node) bool { return isTag(n, "a") || isTag(n, "img") }) {
		for _, key := range []string{"href", "src"} {
			if value := attribute(child, key); value != "" {
				urls = append(urls, child.Data+":"+key+":"+resolve(base, value))
			}
		}
	}
	return textContent(node) + "\n" + strings.Join(urls, "\n")
}

func propertyStrings(item *microformats.Microformat, name string) []string {
	var result []string
	for _, value := range item.Properties[name] {
		switch value := value.(type) {
		case string:
			result = append(result, value)
		case map[string]string:
			result = append(result, value["value"])
		}
	}
	return result
}

func property(item *microformats.Microformat, name, expected string) error {
	values := propertyStrings(item, name)
	if len(values) != 1 || normalized(values[0]) != normalized(expected) {
		return fmt.Errorf("%s must equal %q once, got %q", name, expected, values)
	}
	return nil
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func hasType(item *microformats.Microformat, typ string) bool {
	return contains(item.Type, typ)
}

func itemsOfType(items []*microformats.Microformat, typ string) []*microformats.Microformat {
	var result []*microformats.Microformat
	for _, item := range items {
		if hasType(item, typ) {
			result = append(result, item)
		}
		result = append(result, itemsOfType(item.Children, typ)...)
		for _, values := range item.Properties {
			for _, value := range values {
				if nested, ok := value.(*microformats.Microformat); ok {
					result = append(result, itemsOfType([]*microformats.Microformat{nested}, typ)...)
				}
			}
		}
	}
	return result
}

func uniqueNode(doc *html.Node, matches func(*html.Node) bool, label string) (*html.Node, error) {
	nodes := find(doc, matches)
	if len(nodes) != 1 {
		return nil, fmt.Errorf("expected exactly one %s, got %d", label, len(nodes))
	}
	return nodes[0], nil
}

func meta(doc *html.Node, name string) []string {
	var values []string
	for _, node := range find(doc, func(n *html.Node) bool {
		return isTag(n, "meta") && (attribute(n, "name") == name || attribute(n, "property") == name)
	}) {
		values = append(values, attribute(node, "content"))
	}
	return values
}

func oneValue(values []string, expected, label string) error {
	if len(values) != 1 || values[0] != expected {
		return fmt.Errorf("%s must equal %q once, got %q", label, expected, values)
	}
	return nil
}
