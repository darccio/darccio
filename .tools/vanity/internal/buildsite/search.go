package buildsite

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

const siteSearchMarker = "/* SITE_SEARCH_INDEX */ []"

const (
	siteSearchResultsStart = "<!-- SITE_SEARCH_RESULTS_START -->"
	siteSearchResultsEnd   = "<!-- SITE_SEARCH_RESULTS_END -->"
	siteSearchStatusStart  = "<!-- SITE_SEARCH_STATUS_START -->"
	siteSearchStatusEnd    = "<!-- SITE_SEARCH_STATUS_END -->"
)

type siteSearchEntry struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Language string `json:"language"`
}

// The generator supplies escaped metadata through .Pages; it does not decide
// this site's privacy policy. Recheck sources and final deployed pages before
// exposing a title or summary in HTML, Markdown or the server-side index.
// No page bodies are indexed, and CV paths are rejected before opening files.
func finalizeSiteSearch(stage, deploy string, base *url.URL, docs ...map[string]string) ([]siteSearchEntry, error) {
	locales := []string{"ca", "en", "es"}
	var files []string
	for _, language := range locales {
		prefix := agentContextPrefix(language)
		name := prefix + "search/index.html"
		if _, err := os.Lstat(filepath.Join(deploy, filepath.FromSlash(name))); err == nil {
			files = append(files, name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) != len(locales) {
		return nil, errors.New("site search requires all ca, en and es search pages")
	}
	var candidates []siteSearchEntry
	for i, file := range files {
		body, err := readAgentContextFile(deploy, file)
		if err != nil {
			return nil, err
		}
		entries, err := siteSearchCandidates(body)
		if err != nil {
			return nil, fmt.Errorf("site search %s: %w", file, err)
		}
		if i == 0 {
			candidates = entries
		} else if !reflect.DeepEqual(candidates, entries) {
			return nil, errors.New("site search candidate catalogues disagree between languages")
		}
	}
	index := make([]siteSearchEntry, 0, len(candidates))
	for _, entry := range candidates {
		eligible, err := siteSearchPublicEntry(stage, deploy, base, &entry, docs...)
		if err != nil {
			return nil, fmt.Errorf("site search %s: %w", entry.URL, err)
		}
		if eligible {
			index = append(index, entry)
		}
	}
	// Validate all six artifacts first. Writes stay in the isolated build and
	// cannot promote a partially assembled catalogue over the previous deploy.
	type change struct {
		name string
		body []byte
	}
	var changes []change
	for i, file := range files {
		for _, name := range []string{file, strings.TrimSuffix(file, ".html") + ".md"} {
			body, err := readAgentContextFile(deploy, name)
			if err != nil {
				return nil, fmt.Errorf("site search requires %s: %w", name, err)
			}
			body, err = siteSearchReplace(body, siteSearchResultsStart, siteSearchResultsEnd, siteSearchResults(index, locales[i]))
			if err != nil {
				return nil, fmt.Errorf("site search %s: %w", name, err)
			}
			body, err = siteSearchReplace(body, siteSearchStatusStart, siteSearchStatusEnd, siteSearchStatus(len(index), locales[i]))
			if err != nil {
				return nil, fmt.Errorf("site search %s: %w", name, err)
			}
			if bytes.Count(body, []byte(`name="q" value="" data-search-query`)) != 1 {
				return nil, fmt.Errorf("site search %s requires exactly one empty query input", name)
			}
			changes = append(changes, change{name, body})
		}
	}
	for _, change := range changes {
		if err := os.WriteFile(filepath.Join(deploy, filepath.FromSlash(change.name)), change.body, 0644); err != nil {
			return nil, err
		}
	}
	return index, nil
}

func siteSearchCandidates(body []byte) ([]siteSearchEntry, error) {
	fragment, err := siteSearchBetween(body, siteSearchResultsStart, siteSearchResultsEnd)
	if err != nil {
		return nil, err
	}
	tree, err := html.Parse(bytes.NewReader(fragment))
	if err != nil {
		return nil, err
	}
	var entries []siteSearchEntry
	seen := make(map[string]bool)
	var walk func(*html.Node) error
	walk = func(node *html.Node) error {
		if node.Type == html.ElementNode && node.Data == "li" {
			attrs := siteSearchAttributes(node)
			entry := siteSearchEntry{URL: attrs["data-search-url"], Language: attrs["data-search-language"]}
			if _, err := siteSearchURL(entry.URL); err != nil {
				return err
			}
			if seen[entry.URL] {
				return fmt.Errorf("duplicate candidate URL %q", entry.URL)
			}
			seen[entry.URL] = true
			if entry.Language != "ca" && entry.Language != "en" && entry.Language != "es" {
				return fmt.Errorf("invalid search language %q", entry.Language)
			}
			title, summary := siteSearchFindClass(node, "search-result__title"), siteSearchFindClass(node, "search-result__summary")
			if title == nil || title.Data != "a" || siteSearchAttributes(title)["href"] != entry.URL {
				return fmt.Errorf("candidate %q requires its matching title link", entry.URL)
			}
			entry.Title = siteSearchText(title)
			if summary != nil {
				entry.Summary = siteSearchText(summary)
			}
			if entry.Title == "" || len(entry.Title) > 4096 || len(entry.Summary) > 16384 || strings.IndexFunc(entry.Title+entry.Summary, unicode.IsControl) != -1 {
				return errors.New("candidate title and summary must be plain text")
			}
			entries = append(entries, entry)
			return nil
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(tree); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].URL < entries[j].URL })
	return entries, nil
}

func siteSearchPublicEntry(stage, deploy string, base *url.URL, entry *siteSearchEntry, docs ...map[string]string) (bool, error) {
	u, err := siteSearchURL(entry.URL)
	if err != nil {
		return false, err
	}
	basePath := strings.TrimSuffix(base.Path, "/")
	if basePath != "" && !strings.HasPrefix(u.Path, basePath+"/") {
		return false, errors.New("candidate lies outside site.baseurl")
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(u.Path, basePath), "/")
	for _, part := range strings.Split(relative, "/") {
		lower := strings.ToLower(part)
		if strings.HasPrefix(part, ".") || lower == "cv" || lower == "private" || lower == "draft" || lower == "search" || lower == "readme.html" || lower == "readme" {
			return false, nil
		}
	}
	if !allowedSource(relative) || relative == "docs" || strings.HasPrefix(relative, "docs/") || strings.HasPrefix(relative, "static/") || strings.HasPrefix(relative, "assets/") {
		return false, nil
	}
	file := relative
	if file == "" || strings.HasSuffix(file, "/") {
		file += "index.html"
	} else if path.Ext(file) != ".html" {
		return false, errors.New("candidate must name a canonical HTML page")
	}
	// Ownership comes only from the trusted project-docs compiler's manifest,
	// never a URL prefix or user-supplied docs_project metadata field.
	declaredTwin := declaredDocsTwin(docs, file)
	var source string
	for _, candidate := range []string{strings.TrimSuffix(file, ".html") + ".md", file} {
		if _, err := agentContextRegularPath(stage, candidate, false); err == nil {
			if source != "" {
				return false, errors.New("candidate has ambiguous Markdown and HTML sources")
			}
			source = candidate
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	if source == "" {
		return false, nil // A stale artifact must not acquire search visibility.
	}
	for parent := path.Dir(source); ; parent = path.Dir(parent) {
		name := path.Join(parent, ".zas.yml")
		if _, err := os.Lstat(filepath.Join(stage, filepath.FromSlash(name))); err == nil {
			body, err := readAgentContextFile(stage, name)
			if err != nil {
				return false, err
			}
			var metadata agentContextMetadata
			if err := yaml.Unmarshal(body, &metadata); err != nil {
				return false, err
			}
			if declaredTwin != "" {
				metadata.PublishMarkdown = nil
			}
			if agentContextEligible(metadata) != nil {
				return false, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if parent == "." {
			break
		}
	}
	// Source metadata is an independent fail-closed gate. An invalid, private,
	// draft or unpublished source is never indexed. A Markdown opt-out is
	// allowed only when the trusted compiler owns and supplies its public twin.
	if err := agentContextSourceVisibility(stage, source, entry.Language, declaredTwin != ""); err != nil {
		return false, nil
	}
	if declaredTwin != "" {
		if err := siteSearchDocsTwin(deploy, declaredTwin); err != nil {
			return false, err
		}
	}
	body, err := readAgentContextFile(deploy, file)
	if err != nil {
		return false, err
	}
	canonical, language, err := sitemapHTMLMetadata(body)
	if err != nil {
		return false, err
	}
	expected := *base
	expected.Path, expected.RawPath = u.Path, ""
	if canonical != expected.String() || language != entry.Language {
		return false, errors.New("candidate canonical URL or language differs from deployed HTML")
	}
	if entry.Summary == "" {
		entry.Summary, err = siteSearchHeadDescription(body)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func siteSearchDocsTwin(deploy, relative string) error {
	if err := httpWorkerSafePath(relative); err != nil {
		return err
	}
	if path.Ext(relative) != ".md" {
		return errors.New("search project-docs twin must be Markdown")
	}
	for _, part := range strings.Split(relative, "/") {
		if strings.HasPrefix(part, ".") || strings.EqualFold(part, "cv") || strings.EqualFold(part, "private") {
			return errors.New("search project-docs twin has an excluded path")
		}
	}
	body, err := readAgentContextFile(deploy, relative)
	if err != nil {
		return err
	}
	return httpWorkerMarkdown(body, true)
}

// Descriptions are already public page metadata. Inspect only the head and
// never derive search text from article, project-docs or other page bodies.
func siteSearchHeadDescription(body []byte) (string, error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	inHead := false
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return "", z.Err()
			}
			return "", nil
		}
		token := z.Token()
		if kind == html.EndTagToken && token.Data == "head" {
			return "", nil
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		if token.Data == "body" {
			return "", nil
		}
		if token.Data == "head" {
			inHead = true
		}
		if inHead && token.Data == "meta" {
			attrs := make(map[string]string)
			for _, attr := range token.Attr {
				attrs[attr.Key] = attr.Val
			}
			if strings.EqualFold(attrs["name"], "description") {
				text := strings.Join(strings.Fields(attrs["content"]), " ")
				if len(text) > 16384 || strings.IndexFunc(text, unicode.IsControl) != -1 {
					return "", errors.New("search description metadata is invalid")
				}
				return text, nil
			}
		}
	}
}

func siteSearchURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\?#") || strings.IndexFunc(raw, unicode.IsControl) != -1 || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("unsafe search URL %q", raw)
	}
	clean := path.Clean(u.Path)
	if strings.HasSuffix(u.Path, "/") && clean != "/" {
		clean += "/"
	}
	if clean != u.Path || u.EscapedPath() != raw || strings.IndexFunc(u.Path, unicode.IsControl) != -1 || strings.Contains(strings.ToLower(raw), "%2f") || strings.Contains(strings.ToLower(raw), "%5c") {
		return nil, fmt.Errorf("noncanonical search URL %q", raw)
	}
	return u, nil
}

func siteSearchBetween(body []byte, start, end string) ([]byte, error) {
	if bytes.Count(body, []byte(start)) != 1 || bytes.Count(body, []byte(end)) != 1 {
		return nil, errors.New("search markers must occur exactly once")
	}
	first := bytes.Index(body, []byte(start)) + len(start)
	last := bytes.Index(body, []byte(end))
	if first > last {
		return nil, errors.New("search markers are reversed")
	}
	return body[first:last], nil
}

func siteSearchReplace(body []byte, start, end, replacement string) ([]byte, error) {
	fragment, err := siteSearchBetween(body, start, end)
	if err != nil {
		return nil, err
	}
	first := bytes.Index(body, []byte(start)) + len(start)
	out := append([]byte(nil), body[:first]...)
	out = append(out, '\n')
	out = append(out, replacement...)
	out = append(out, '\n')
	out = append(out, body[first+len(fragment):]...)
	return out, nil
}

func siteSearchResults(index []siteSearchEntry, language string) string {
	var out strings.Builder
	out.WriteString(`<ul class="search-results">` + "\n")
	for _, entry := range index {
		caption := map[string]map[string]string{
			"ca": {"ca": "en català", "en": "en anglès", "es": "en castellà"},
			"en": {"ca": "in Catalan", "en": "in English", "es": "in Spanish"},
			"es": {"ca": "en catalán", "en": "en inglés", "es": "en castellano"},
		}[language][entry.Language]
		escape := stdhtml.EscapeString
		fmt.Fprintf(&out, `<li class="search-result" lang="%s" data-search-url="%s" data-search-language="%s"><h2><a class="search-result__title" href="%s">%s</a><span class="lang-tag" aria-hidden="true">%s</span><span class="visually-hidden" lang="%s"> (%s)</span></h2><p class="search-result__summary">%s</p></li>`+"\n", entry.Language, escape(entry.URL), entry.Language, escape(entry.URL), escape(entry.Title), entry.Language, language, caption, escape(entry.Summary))
	}
	out.WriteString("</ul>")
	return out.String()
}

func siteSearchStatus(count int, language string) string {
	format := map[string]string{
		"ca": "%d pàgines. Cerca per títol o resum, o explora el catàleg.",
		"en": "%d pages. Search titles and summaries, or browse the catalogue.",
		"es": "%d páginas. Busca por título o resumen, o explora el catálogo.",
	}[language]
	return `<p class="search-status" role="status">` + fmt.Sprintf(format, count) + `</p>`
}

func siteSearchAttributes(node *html.Node) map[string]string {
	attrs := make(map[string]string)
	for _, attr := range node.Attr {
		attrs[attr.Key] = attr.Val
	}
	return attrs
}

func siteSearchFindClass(node *html.Node, class string) *html.Node {
	if node.Type == html.ElementNode && navigationClass(siteSearchAttributes(node)["class"], class) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := siteSearchFindClass(child, class); found != nil {
			return found
		}
	}
	return nil
}

func siteSearchText(node *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(text.String()), " ")
}

func siteSearchInject(template []byte, index []siteSearchEntry) ([]byte, error) {
	count := bytes.Count(template, []byte(siteSearchMarker))
	if count == 0 && index == nil {
		return template, nil // Existing sites without search keep their adapter.
	}
	if count != 1 {
		return nil, errors.New("HTTP worker template must contain exactly one search-index marker")
	}
	if index == nil {
		index = []siteSearchEntry{}
	}
	encoded, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}
	return bytes.Replace(template, []byte(siteSearchMarker), encoded, 1), nil
}
