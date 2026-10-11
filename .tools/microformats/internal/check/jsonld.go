package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const personID = homeURL + "#person"

// These seven reviewed head fragments deliberately bypass the shared layout.
// No other HTML page is exempt, including the legacy contact fragment and drafts.
var rawVanityHTML = map[string]bool{
	"mergo/index.html": true, "darna/index.html": true, "cocoon/index.html": true,
	"errors/index.html": true, "xdr/index.html": true, "awaken/index.html": true, "x/index.html": true,
}

func validateJSONLDPages(deploy string, inv inventory) error {
	posts := make(map[string]Post, len(inv.Posts))
	for _, post := range inv.Posts {
		posts[post.File] = post
	}
	var failures []error
	seen := make(map[string]bool)
	err := filepath.WalkDir(deploy, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.EqualFold(entry.Name(), "CV") {
				// Preserve existing public machine-readable files without opening
				// them. Inspect names only to reject a new HTML render, then skip
				// this tree entirely rather than processing CV data.
				if err := filepath.WalkDir(path, func(file string, child fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !child.IsDir() && strings.EqualFold(filepath.Ext(file), ".html") {
						return fmt.Errorf("unexpected CV HTML output; validation does not process CV data")
					}
					return nil
				}); err != nil {
					return err
				}
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("HTML output is a symlink: %s", path)
		}
		file, err := filepath.Rel(deploy, path)
		if err != nil {
			return err
		}
		file = filepath.ToSlash(file)
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := parseHTML(input)
		if err != nil {
			return err
		}
		if rawVanityHTML[file] && !projectDocsLanding(doc, file) {
			if len(jsonLDScripts(doc)) != 0 {
				failures = append(failures, fmt.Errorf("%s: raw vanity fragment acquired JSON-LD", file))
			}
			return nil
		}
		pagePath := file
		if file == "index.html" || strings.HasSuffix(file, "/index.html") {
			pagePath = strings.TrimSuffix(file, "index.html")
		}
		pageURL := homeURL + pagePath
		var post *Post
		if value, ok := posts[file]; ok {
			post = &value
			seen[file] = true
		}
		if err := validateJSONLD(doc, pageURL, post); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", file, err))
		}
		return nil
	})
	if err != nil {
		failures = append(failures, err)
	}
	for file := range posts {
		if !seen[file] {
			failures = append(failures, fmt.Errorf("%s: public article missing from structured-data output", file))
		}
	}
	return errors.Join(failures...)
}

func jsonLDScripts(doc *htmlNode) []*htmlNode {
	return find(doc, func(n *htmlNode) bool {
		return isTag(n, "script") && strings.EqualFold(strings.TrimSpace(attribute(n, "type")), "application/ld+json")
	})
}

func validateJSONLD(doc *htmlNode, pageURL string, post *Post) error {
	head, err := uniqueNode(doc, func(n *htmlNode) bool { return isTag(n, "head") }, "page head")
	if err != nil {
		return err
	}
	scripts := jsonLDScripts(doc)
	if len(scripts) != 1 || len(jsonLDScripts(head)) != 1 {
		return fmt.Errorf("expected exactly one JSON-LD script in head, got %d", len(scripts))
	}
	var input strings.Builder
	for child := scripts[0].FirstChild; child != nil; child = child.NextSibling {
		input.WriteString(child.Data) // Raw script text; HTML entities must not be decoded.
	}
	data, err := decodeJSONObject(input.String())
	if err != nil {
		return fmt.Errorf("JSON-LD: %w", err)
	}
	if err := jsonString(data, "@context", "https://schema.org"); err != nil {
		return err
	}
	graph, ok := data["@graph"].([]any)
	project := structuredProject(pageURL)
	expected := 1
	if project != "" {
		expected++
	}
	if post != nil {
		expected++
	}
	if !ok || len(graph) != expected {
		return fmt.Errorf("JSON-LD graph must contain Person and the eligible article or project record")
	}
	var person, article, software map[string]any
	for _, value := range graph {
		node, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("JSON-LD graph node must be an object")
		}
		switch node["@type"] {
		case "Person":
			if person != nil {
				return fmt.Errorf("duplicate Person in JSON-LD graph")
			}
			person = node
		case "SoftwareSourceCode":
			if software != nil {
				return fmt.Errorf("duplicate SoftwareSourceCode in graph")
			}
			software = node
		case "BlogPosting":
			if article != nil {
				return fmt.Errorf("duplicate BlogPosting in JSON-LD graph")
			}
			article = node
		default:
			return fmt.Errorf("unexpected JSON-LD graph type %v", node["@type"])
		}
	}
	if person == nil || (post == nil && article != nil) || (post != nil && article == nil) {
		return fmt.Errorf("JSON-LD graph types disagree with public article eligibility")
	}
	for key, expected := range map[string]string{
		"@id": personID, "name": owner, "url": homeURL, "image": homeURL + "static/avatar.webp", "jobTitle": "Senior Software Engineer",
	} {
		if err := jsonString(person, key, expected); err != nil {
			return fmt.Errorf("Person: %w", err)
		}
	}
	organization, ok := person["worksFor"].(map[string]any)
	if !ok || organization["@type"] != "Organization" || organization["name"] != "Datadog" {
		return fmt.Errorf("Person worksFor must identify Datadog as an Organization")
	}
	sameAs, err := jsonStrings(person["sameAs"])
	if err != nil || len(sameAs) != len(profiles) {
		return fmt.Errorf("Person sameAs must contain the seven verified profiles")
	}
	slices.Sort(sameAs)
	expectedProfiles := slices.Clone(profiles)
	slices.Sort(expectedProfiles)
	if !slices.Equal(sameAs, expectedProfiles) {
		return fmt.Errorf("Person sameAs disagrees with verified profiles: %v", sameAs)
	}
	for _, profile := range profiles {
		links := find(head, func(n *htmlNode) bool {
			return isTag(n, "link") && contains(strings.Fields(attribute(n, "rel")), "me") && attribute(n, "href") == profile
		})
		if len(links) != 1 {
			return fmt.Errorf("Person profile lacks matching head rel=me link: %s", profile)
		}
	}
	canonicals := find(head, func(n *htmlNode) bool { return isTag(n, "link") && attribute(n, "rel") == "canonical" })
	if len(canonicals) != 1 || attribute(canonicals[0], "href") != pageURL {
		return fmt.Errorf("JSON-LD page canonical disagrees with deployed path %q", pageURL)
	}
	if project != "" {
		if software == nil {
			return fmt.Errorf("project landing lacks SoftwareSourceCode")
		}
		name := map[string]string{"mergo": "Mergo", "zas": "Zas", "darna": "darna"}[project]
		for key, value := range map[string]string{"name": name, "url": pageURL, "codeRepository": "https://github.com/darccio/" + project, "programmingLanguage": "Go", "license": homeURL + project + "/LICENSE.txt"} {
			if err := jsonString(software, key, value); err != nil {
				return fmt.Errorf("SoftwareSourceCode: %w", err)
			}
		}
	} else if software != nil {
		return fmt.Errorf("unexpected software record outside reviewed project roots")
	}
	if post == nil {
		return nil
	}
	for key, expected := range map[string]string{
		"headline": post.Title, "url": post.URL, "description": post.Summary, "inLanguage": post.Language,
	} {
		if err := jsonString(article, key, expected); err != nil {
			return fmt.Errorf("BlogPosting: %w", err)
		}
	}
	author, ok := article["author"].(map[string]any)
	if !ok || author["@id"] != personID {
		return fmt.Errorf("BlogPosting author must reference the shared Person")
	}
	tags, err := jsonStrings(article["keywords"])
	if err != nil || !slices.Equal(tags, post.Tags) {
		return fmt.Errorf("BlogPosting keywords disagree with source tags: %v", tags)
	}
	for key, date := range map[string]time.Time{"datePublished": post.Date, "dateModified": post.Updated} {
		if date.IsZero() {
			if _, present := article[key]; present {
				return fmt.Errorf("BlogPosting invented unknown %s", key)
			}
		} else if err := jsonString(article, key, date.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("BlogPosting: %w", err)
		}
	}
	return nil
}

func jsonString(object map[string]any, key, expected string) error {
	actual, ok := object[key].(string)
	if !ok || actual != expected {
		return fmt.Errorf("%s must equal %q, got %v", key, expected, object[key])
	}
	return nil
}

func jsonStrings(value any) ([]string, error) {
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a JSON string array")
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected a JSON string")
		}
		result = append(result, text)
	}
	return result, nil
}

// encoding/json normally accepts duplicate keys, retaining only the last value.
// Token decoding rejects that ambiguity in every nested object as well as
// quoted documents and trailing values before structured metadata is checked.
func decodeJSONObject(input string) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(input))
	value, err := decodeJSONValue(decoder)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a JSON object, not a quoted document or array")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after JSON object")
	}
	return object, nil
}

func decodeJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		object := make(map[string]any)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("object key must be a string")
			}
			if _, exists := object[name]; exists {
				return nil, fmt.Errorf("duplicate JSON key %q", name)
			}
			object[name], err = decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
		}
		_, err := decoder.Token()
		return object, err
	case json.Delim('['):
		array := make([]any, 0)
		for decoder.More() {
			value, err := decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		_, err := decoder.Token()
		return array, err
	default:
		return token, nil
	}
}

// Only reviewed docs roots can replace the corresponding raw import fragment.
// They receive every ordinary structured-data check, rather than an exemption.
func projectDocsLanding(doc *htmlNode, file string) bool {
	project := map[string]string{"mergo/index.html": "mergo", "darna/index.html": "darna"}[file]
	if project == "" {
		return false
	}
	markers := find(doc, func(n *htmlNode) bool {
		return isTag(n, "meta") && attribute(n, "name") == "project-docs" && attribute(n, "content") == project
	})
	return len(markers) == 1
}

func structuredProject(pageURL string) string {
	for _, project := range []string{"mergo", "zas", "darna"} {
		if pageURL == homeURL+project+"/" {
			return project
		}
	}
	return ""
}
