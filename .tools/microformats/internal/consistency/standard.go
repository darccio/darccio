package consistency

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type standardPlan struct {
	Version     int               `json:"version"`
	Publication string            `json:"publication"`
	Documents   []plannedDocument `json:"documents"`
	Candidates  any               `json:"candidates,omitempty"`
}

type plannedDocument struct {
	Path     string         `json:"path"`
	URI      string         `json:"uri"`
	CID      string         `json:"cid,omitempty"`
	Action   string         `json:"action"`
	Approved bool           `json:"approved,omitempty"`
	Record   map[string]any `json:"record"`
}

type allowlist struct {
	Version   int `json:"version"`
	Documents []struct {
		Path               string `json:"path"`
		PublishedAt        string `json:"publishedAt"`
		CorrectPublishedAt bool   `json:"correctPublishedAt,omitempty"`
	} `json:"documents"`
}

func atURI(uri, collection string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(uri, "at://"), "/")
	if !strings.HasPrefix(uri, "at://") || len(parts) != 3 || !strings.HasPrefix(parts[0], "did:") || parts[1] != collection || len(parts[2]) != 13 || !strings.Contains("234567abcdefghij", parts[2][:1]) {
		return "", fmt.Errorf("invalid %s AT URI %q", collection, uri)
	}
	for _, char := range parts[2] {
		if !strings.ContainsRune("234567abcdefghijklmnopqrstuvwxyz", char) {
			return "", fmt.Errorf("invalid TID in AT URI %q", uri)
		}
	}
	return parts[0], nil
}

func checkStandard(root, deploy, planPath string, posts map[string]sourcePost, rendered map[string]renderedPost) (int, int, error) {
	var publication struct {
		URI, CID, DID, PDS, URL string
	}
	input, err := regularFile(root, ".zas/atproto-publication.json")
	if err != nil {
		return 0, 0, err
	}
	if err := decodeJSON(input, &publication); err != nil {
		return 0, 0, err
	}
	did, err := atURI(publication.URI, "site.standard.publication")
	if err != nil || publication.URL != "https://dario.cat" || publication.DID != did {
		return 0, 0, fmt.Errorf("publication verification metadata disagrees with canonical site: %v", err)
	}
	verification, err := regularFile(deploy, ".well-known/site.standard.publication")
	if err != nil {
		return 0, 0, err
	}
	if string(verification) != publication.URI {
		return 0, 0, fmt.Errorf("publication verification file must return exactly the recorded AT URI")
	}
	var approved allowlist
	input, err = regularFile(root, ".tools/atproto/documents-allowlist.json")
	if err != nil {
		return 0, 0, err
	}
	if err := decodeJSON(input, &approved); err != nil {
		return 0, 0, err
	}
	if approved.Version != 1 || approved.Documents == nil {
		return 0, 0, fmt.Errorf("reviewed document allowlist must declare version 1 and its explicit document list")
	}
	var plan standardPlan
	input, err = absoluteRegularFile(planPath)
	if err != nil {
		return 0, 0, fmt.Errorf("Standard.site record plan: %w", err)
	}
	if err := decodeJSON(input, &plan); err != nil {
		return 0, 0, err
	}
	if plan.Version != 1 || plan.Publication != publication.URI || plan.Documents == nil {
		return 0, 0, fmt.Errorf("Standard.site plan must declare version 1, the verified publication and explicit documents")
	}
	allowed := make(map[string]time.Time)
	for _, row := range approved.Documents {
		post, present := posts[row.Path]
		if !present || post.Excluded || !post.Feed || !post.Standard || post.Date.IsZero() {
			return 0, 0, fmt.Errorf("allowlist selects ineligible or unknown article %q", row.Path)
		}
		if _, duplicate := allowed[row.Path]; duplicate {
			return 0, 0, fmt.Errorf("duplicate approved article %q", row.Path)
		}
		date, err := time.Parse(time.RFC3339Nano, row.PublishedAt)
		if err != nil || !date.Equal(post.Date) {
			return 0, 0, fmt.Errorf("%s: approved publication instant disagrees with source date", row.Path)
		}
		allowed[row.Path] = date
	}
	if len(plan.Documents) != len(allowed) {
		return 0, 0, fmt.Errorf("record plan has %d documents, reviewed allowlist has %d", len(plan.Documents), len(allowed))
	}
	planned := make(map[string]string)
	seenURI := make(map[string]bool)
	for _, document := range plan.Documents {
		post := posts[document.Path]
		if _, approved := allowed[document.Path]; !approved {
			return 0, 0, fmt.Errorf("record plan selects unapproved article %q", document.Path)
		}
		if _, duplicate := planned[document.Path]; duplicate || seenURI[document.URI] {
			return 0, 0, fmt.Errorf("duplicate planned document path or URI")
		}
		documentDID, err := atURI(document.URI, "site.standard.document")
		if err != nil || documentDID != did {
			return 0, 0, fmt.Errorf("%s: planned document URI does not belong to publication owner", document.Path)
		}
		if document.Action != "create" && document.Action != "update" && document.Action != "unchanged" {
			return 0, 0, fmt.Errorf("%s: unsupported build plan action %q", document.Path, document.Action)
		}
		for key, want := range map[string]string{"$type": "site.standard.document", "site": publication.URI, "path": post.Path, "title": post.Title, "description": post.Summary} {
			if document.Record[key] != want {
				return 0, 0, fmt.Errorf("%s: Standard.site %s got %v, want %q", post.Path, key, document.Record[key], want)
			}
		}
		for key, want := range map[string]time.Time{"publishedAt": post.Date, "updatedAt": post.Updated} {
			var values []string
			if value, present := document.Record[key]; present {
				value, ok := value.(string)
				if !ok {
					return 0, 0, fmt.Errorf("%s: Standard.site %s must be an instant string", post.Path, key)
				}
				values = []string{value}
			}
			if err := instant(values, want, "Standard.site "+key); err != nil {
				return 0, 0, fmt.Errorf("%s: %w", post.Path, err)
			}
		}
		var tags []string
		values, ok := document.Record["tags"].([]any)
		if !ok {
			return 0, 0, fmt.Errorf("%s: Standard.site tags must be a string array", post.Path)
		}
		for _, value := range values {
			tag, ok := value.(string)
			if !ok {
				return 0, 0, fmt.Errorf("%s: Standard.site tag must be a string", post.Path)
			}
			tags = append(tags, tag)
		}
		wantTags := append(slices.Clone(post.Tags), "lang:"+post.Language)
		if !slices.Equal(tags, wantTags) {
			return 0, 0, fmt.Errorf("%s: Standard.site tags disagree with source and declared language", post.Path)
		}
		plain, ok := document.Record["textContent"].(string)
		if !ok || normalized(plain) != rendered[post.Path].Text {
			return 0, 0, fmt.Errorf("%s: Standard.site textContent differs from public h-entry content", post.Path)
		}
		planned[post.Path], seenURI[document.URI] = document.URI, true
	}
	var published map[string]string
	input, err = regularFile(root, ".zas/atproto.json")
	if err != nil {
		return 0, 0, err
	}
	if err := decodeJSON(input, &published); err != nil || published == nil {
		return 0, 0, fmt.Errorf("verified document manifest must be an explicit flat object: %v", err)
	}
	seenURI = make(map[string]bool)
	for path, uri := range published {
		if planned[path] == "" || uri != planned[path] || seenURI[uri] {
			return 0, 0, fmt.Errorf("%s: verified document URI differs from approved plan or is duplicated", path)
		}
		seenURI[uri] = true
	}
	for path, post := range posts {
		if post.Excluded {
			continue
		}
		input, err := regularFile(deploy, post.File)
		if err != nil {
			return 0, 0, err
		}
		doc, err := parseHTML(input)
		if err != nil {
			return 0, 0, err
		}
		links := find(doc, func(n *htmlNode) bool {
			return isTag(n, "link") && slices.Contains(strings.Fields(attribute(n, "rel")), "site.standard.document")
		})
		for _, link := range links {
			if link.Parent == nil || !isTag(link.Parent, "head") {
				return 0, 0, fmt.Errorf("%s: document verification link must be in the page head", path)
			}
		}
		uri, exists := published[path]
		if exists && (len(links) != 1 || attribute(links[0], "href") != uri) {
			return 0, 0, fmt.Errorf("%s: published document link differs from verified state", path)
		}
		if !exists && len(links) != 0 {
			return 0, 0, fmt.Errorf("%s: unverified or excluded article gained a document link", path)
		}
	}
	return len(allowed), len(published), nil
}
