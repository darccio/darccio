package buildsite

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

const standardDocumentPrefix = "at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.document/"

// This is the current AT Protocol tid syntax, including its first-character
// restriction: https://atproto.com/specs/tid#tid-syntax.
var standardDocumentTID = regexp.MustCompile(`^[234567abcdefghij][234567abcdefghijklmnopqrstuvwxyz]{12}$`)
var standardDocumentPath = regexp.MustCompile(`^/(?:posts|en/posts|es/posts|rizoma)/[a-z0-9]+(?:[a-z0-9/-]*[a-z0-9])?/$`)

type standardDocumentApproval struct {
	Path               string `json:"path"`
	PublishedAt        string `json:"publishedAt"`
	CorrectPublishedAt bool   `json:"correctPublishedAt,omitempty"`
}

// Public manifests contain only successfully verified record identifiers. The
// build consumes them offline and never contacts a PDS or reads publisher state.
func finalizeStandardDocuments(root, stage, deploy string) error {
	if _, err := os.Lstat(filepath.Join(stage, ".zas", "atproto.json")); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	data, err := readAgentContextFile(stage, ".zas/atproto.json")
	if err != nil {
		return err
	}
	manifest, err := decodeStandardDocumentManifest(data)
	if err != nil {
		return err
	}
	if len(manifest) == 0 {
		return nil
	}
	approvals, err := readStandardDocumentApprovals(root)
	if err != nil {
		return err
	}
	config, err := readAgentContextFile(stage, ".zas/config.yml")
	if err != nil {
		return err
	}
	var settings struct {
		Site struct {
			BaseURL string `yaml:"baseurl"`
		} `yaml:"site"`
	}
	if err := yaml.Unmarshal(config, &settings); err != nil {
		return err
	}
	base, err := markdownDiscoveryBase(settings.Site.BaseURL)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(manifest))
	for pathname := range manifest {
		paths = append(paths, pathname)
	}
	sort.Strings(paths)
	selected := make(map[string]string, len(paths))
	for _, pathname := range paths {
		approval, exists := approvals[pathname]
		if !exists {
			return fmt.Errorf("Standard.site document %s has no reviewed approval", pathname)
		}
		source := strings.TrimPrefix(pathname, "/") + "index.md"
		if err := validateStandardDocumentSource(stage, source, approval); err != nil {
			return fmt.Errorf("Standard.site document %s: %w", pathname, err)
		}
		file := strings.TrimPrefix(pathname, "/") + "index.html"
		body, err := readAgentContextFile(deploy, file)
		if err != nil {
			return err
		}
		canonical, _, err := sitemapHTMLMetadata(body)
		if err != nil || canonical != agentContextURL(base, strings.TrimPrefix(pathname, "/")) {
			return fmt.Errorf("Standard.site document %s has inconsistent canonical metadata", pathname)
		}
		selected[file] = manifest[pathname]
	}
	// Validate all public article heads before writing. This also rejects an
	// unreviewed article acquiring a link through layout or front matter. Raw vanity
	// fragments, CV and operations documents are outside this traversal.
	var changes []markdownDiscoveryChange
	for _, section := range []string{"posts", "en/posts", "es/posts", "rizoma"} {
		directory := filepath.Join(deploy, filepath.FromSlash(section))
		if _, err := os.Lstat(directory); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		err := filepath.WalkDir(directory, func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(deploy, file)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("Standard.site refuses deploy symlink %s", relative)
			}
			if httpWorkerExcluded(relative) {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() || filepath.Ext(relative) != ".html" {
				return nil
			}
			body, err := readAgentContextFile(deploy, relative)
			if err != nil {
				return err
			}
			position, linked, err := standardDocumentHead(body, selected[relative])
			if err != nil {
				return fmt.Errorf("Standard.site %s: %w", relative, err)
			}
			if uri := selected[relative]; uri != "" && !linked {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				link := []byte("\n<link rel=\"site.standard.document\" href=\"" + stdhtml.EscapeString(uri) + "\">\n")
				updated := make([]byte, 0, len(body)+len(link))
				updated = append(updated, body[:position]...)
				updated = append(updated, link...)
				updated = append(updated, body[position:]...)
				if _, linked, err := standardDocumentHead(updated, uri); err != nil || !linked {
					return fmt.Errorf("Standard.site inserted link failed validation: %v", err)
				}
				changes = append(changes, markdownDiscoveryChange{
					page: markdownDiscoveryPage{file: file, mode: info.Mode(), body: body}, body: updated,
				})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	for _, change := range changes {
		if err := writeMarkdownDiscovery(change); err != nil {
			return err
		}
	}
	return nil
}

func decodeStandardDocumentManifest(data []byte) (map[string]string, error) {
	manifest := make(map[string]string)
	if len(bytes.TrimSpace(data)) == 0 {
		return manifest, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, errors.New("Standard.site public manifest must be a JSON object")
	}
	seenURI := make(map[string]bool)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		pathname, ok := key.(string)
		if !ok || !validStandardDocumentPath(pathname) {
			return nil, fmt.Errorf("Standard.site invalid canonical post path %q", key)
		}
		if _, exists := manifest[pathname]; exists {
			return nil, fmt.Errorf("Standard.site duplicate manifest path %s", pathname)
		}
		var uri string
		if err := decoder.Decode(&uri); err != nil {
			return nil, err
		}
		if err := validStandardDocumentURI(uri); err != nil {
			return nil, err
		}
		if seenURI[uri] {
			return nil, errors.New("Standard.site one record URI cannot verify multiple pages")
		}
		manifest[pathname], seenURI[uri] = uri, true
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
		return nil, errors.New("Standard.site public manifest is not a complete JSON object")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("Standard.site public manifest must contain one JSON object")
	}
	return manifest, nil
}

func validStandardDocumentURI(uri string) error {
	if !strings.HasPrefix(uri, standardDocumentPrefix) || !standardDocumentTID.MatchString(strings.TrimPrefix(uri, standardDocumentPrefix)) {
		return fmt.Errorf("Standard.site invalid verified document URI %q", uri)
	}
	return nil
}

func validStandardDocumentPath(pathname string) bool {
	return standardDocumentPath.MatchString(pathname) && strings.TrimSuffix(pathname, "/") == path.Clean(pathname) &&
		!httpWorkerExcluded(strings.TrimPrefix(pathname, "/")+"index.md")
}

func readStandardDocumentApprovals(root string) (map[string]standardDocumentApproval, error) {
	data, err := readAgentContextFile(root, ".tools/atproto/documents-allowlist.json")
	if err != nil {
		return nil, fmt.Errorf("Standard.site reviewed allowlist: %w", err)
	}
	var list struct {
		Version   int                        `json:"version"`
		Documents []standardDocumentApproval `json:"documents"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&list); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) || list.Version != 1 {
		return nil, errors.New("Standard.site reviewed allowlist requires one version 1 JSON document")
	}
	approvals := make(map[string]standardDocumentApproval, len(list.Documents))
	for _, approval := range list.Documents {
		if !validStandardDocumentPath(approval.Path) {
			return nil, fmt.Errorf("Standard.site invalid approved post path %q", approval.Path)
		}
		if _, err := time.Parse(time.RFC3339Nano, approval.PublishedAt); err != nil {
			return nil, fmt.Errorf("Standard.site %s requires a reviewed publication instant", approval.Path)
		}
		if _, duplicate := approvals[approval.Path]; duplicate {
			return nil, fmt.Errorf("Standard.site duplicate approval for %s", approval.Path)
		}
		approvals[approval.Path] = approval
	}
	return approvals, nil
}

func validateStandardDocumentSource(stage, source string, approval standardDocumentApproval) error {
	data, err := readAgentContextFile(stage, source)
	if err != nil {
		return err
	}
	data = bytes.TrimSpace(data)
	if !bytes.HasPrefix(data, []byte("<!--")) {
		return errors.New("article requires leading page configuration")
	}
	end := bytes.Index(data[4:], []byte("-->"))
	if end == -1 {
		return errors.New("article page configuration is unterminated")
	}
	var metadata struct {
		Article       bool   `yaml:"article"`
		Date          string `yaml:"date"`
		DatePrecision string `yaml:"date_precision"`
		Publish       *bool  `yaml:"publish"`
		Feed          *bool  `yaml:"feed"`
		Atproto       *bool  `yaml:"atproto"`
		Draft         bool   `yaml:"draft"`
		Private       bool   `yaml:"private"`
	}
	if err := yaml.Unmarshal(data[4:4+end], &metadata); err != nil {
		return err
	}
	if !metadata.Article || metadata.Draft || metadata.Private ||
		(metadata.Publish != nil && !*metadata.Publish) || (metadata.Feed != nil && !*metadata.Feed) || (metadata.Atproto != nil && !*metadata.Atproto) {
		return errors.New("page is not an eligible public article")
	}
	if metadata.DatePrecision != "" && metadata.DatePrecision != "instant" {
		return errors.New("article publication date is not a known instant")
	}
	published, err := time.Parse(time.RFC3339Nano, metadata.Date)
	if err != nil {
		return errors.New("article requires an explicit publication instant")
	}
	approved, _ := time.Parse(time.RFC3339Nano, approval.PublishedAt)
	if !published.Equal(approved) {
		return errors.New("article publication instant differs from its reviewed approval")
	}
	return nil
}

// Token offsets preserve every unrelated byte and avoid treating script text
// containing a closing-head string as markup. No html/template URL trust bypass
// is needed: the only accepted href is the fixed DID/collection plus a valid TID.
func standardDocumentHead(body []byte, expected string) (position int, linked bool, err error) {
	if expected != "" {
		if err := validStandardDocumentURI(expected); err != nil {
			return 0, false, err
		}
	}
	z := html.NewTokenizer(bytes.NewReader(body))
	offset, heads, closes, templates := 0, 0, 0, 0
	inHead, inBody := false, false
	rawText := ""
	for {
		typeOf := z.Next()
		start := offset
		raw := bytes.Clone(z.Raw())
		offset += len(raw)
		if typeOf == html.ErrorToken {
			if err := z.Err(); !errors.Is(err, io.EOF) {
				return 0, false, err
			}
			break
		}
		token := z.Token()
		if rawText != "" {
			if typeOf == html.EndTagToken && token.Data == rawText {
				rawText = ""
			}
			continue
		}
		if typeOf == html.TextToken && inHead && templates == 0 && strings.TrimSpace(token.Data) != "" {
			return 0, false, errors.New("non-metadata text moves the link outside the active head")
		}
		if typeOf != html.StartTagToken && typeOf != html.SelfClosingTagToken && typeOf != html.EndTagToken {
			continue
		}
		if token.Data == "template" || token.Data == "noscript" {
			if typeOf == html.EndTagToken {
				if templates > 0 {
					templates--
				}
			} else {
				templates++
				if token.Data == "noscript" {
					z.NextIsNotRawText()
				}
			}
		}
		if token.Data == "head" && templates == 0 {
			if typeOf == html.StartTagToken {
				if inHead || inBody || heads != 0 {
					return 0, false, errors.New("malformed or duplicate head")
				}
				heads++
				inHead = true
			} else if typeOf == html.EndTagToken {
				if !inHead {
					return 0, false, errors.New("head closes before it starts")
				}
				closes++
				position, inHead = start, false
			} else {
				return 0, false, errors.New("self-closing head is invalid")
			}
		}
		if token.Data == "body" && typeOf != html.EndTagToken {
			if inHead {
				return 0, false, errors.New("body starts before head closes")
			}
			inBody = true
		}
		if inHead && templates == 0 && typeOf != html.EndTagToken {
			switch token.Data {
			case "head", "html", "base", "meta", "link":
			case "script", "style", "title":
				rawText = token.Data
			default:
				return 0, false, fmt.Errorf("unexpected <%s> moves the link outside the active head", token.Data)
			}
		}
		if token.Data != "link" || typeOf == html.EndTagToken {
			continue
		}
		attrs := make(map[string]string)
		document := false
		if markdownDiscoveryDuplicateAttributes(raw) {
			return 0, false, errors.New("duplicate attributes on a head link")
		}
		for _, attr := range token.Attr {
			attrs[attr.Key] = attr.Val
			if attr.Key == "rel" {
				for _, relation := range strings.Fields(attr.Val) {
					if strings.EqualFold(relation, "site.standard.document") {
						document = true
					}
				}
			}
		}
		if !document {
			continue
		}
		if !inHead || templates != 0 || linked || expected == "" || attrs["href"] != expected {
			return 0, false, errors.New("document link must be unique, reviewed and exact in the active head")
		}
		if err := validStandardDocumentURI(attrs["href"]); err != nil {
			return 0, false, err
		}
		linked = true
	}
	if heads != 1 || closes != 1 || inHead || templates != 0 {
		return 0, false, errors.New("article must contain one explicit head")
	}
	return position, linked, nil
}
