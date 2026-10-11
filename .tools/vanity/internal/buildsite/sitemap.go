package buildsite

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dario.cat/site-tools/internal/vanity"
	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

const (
	sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"
	xhtmlNamespace   = "http://www.w3.org/1999/xhtml"
)

type siteSitemap struct {
	XMLName xml.Name         `xml:"urlset"`
	XMLNS   string           `xml:"xmlns,attr"`
	XHTML   string           `xml:"xmlns:xhtml,attr,omitempty"`
	URLs    []siteSitemapURL `xml:"url"`
}

type siteSitemapURL struct {
	Location string             `xml:"loc"`
	Lastmod  string             `xml:"lastmod,omitempty"`
	Links    []sitemapAlternate `xml:"link"`
}

type sitemapAlternate struct {
	Rel      string `xml:"rel,attr"`
	Language string `xml:"hreflang,attr"`
	URL      string `xml:"href,attr"`
}

func (link sitemapAlternate) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "xhtml:link"}
	type attributes sitemapAlternate
	return encoder.EncodeElement(attributes(link), start)
}

type sitemapPage struct {
	file, canonical, language string
}

// Zas collects normal rendered pages before the assembler restores raw vanity
// fragments. Keep its entries, dates and translation groups, align their URLs
// with the actual HTML, and append only the assembler-owned missing pages.
func finalizeSitemap(ctx context.Context, root, stage, deploy string) error {
	configBytes, err := os.ReadFile(filepath.Join(stage, ".zas", "config.yml"))
	if err != nil {
		return err
	}
	var config struct {
		Site struct {
			Sitemap bool   `yaml:"sitemap"`
			BaseURL string `yaml:"baseurl"`
		} `yaml:"site"`
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	if !config.Site.Sitemap {
		return nil
	}
	base, err := localSitemapURL(config.Site.BaseURL, nil)
	if err != nil {
		return fmt.Errorf("site.baseurl: %w", err)
	}
	files, err := os.ReadDir(deploy)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "sitemap-"), ".xml")
		shard := strings.HasPrefix(file.Name(), "sitemap-") && strings.HasSuffix(file.Name(), ".xml") && name != "" && strings.Trim(name, "0123456789") == ""
		if file.Name() == "sitemap-index.xml" || shard {
			return fmt.Errorf("sharded sitemaps are unsupported by the site assembler; no deployment was promoted")
		}
	}
	pages, references, err := collectSitemapPages(deploy, base)
	if err != nil {
		return err
	}
	path := filepath.Join(deploy, "sitemap.xml")
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("native sitemap.xml must be a regular file: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var sitemap siteSitemap
	if err := readNativeSitemap(data, &sitemap); err != nil {
		return err
	}
	entries := make(map[string]*siteSitemapURL)
	for i := range sitemap.URLs {
		entry := &sitemap.URLs[i]
		location, err := localSitemapURL(entry.Location, base)
		if err != nil {
			return fmt.Errorf("sitemap location %q: %w", entry.Location, err)
		}
		page := references[location.String()]
		if page == nil {
			return fmt.Errorf("sitemap location %q has no deployed HTML page", entry.Location)
		}
		if entries[page.file] != nil {
			return fmt.Errorf("duplicate sitemap page %s", page.file)
		}
		entry.Location = page.canonical
		entries[page.file] = entry
		for j := range entry.Links {
			link := &entry.Links[j]
			href, err := localSitemapURL(link.URL, base)
			if err != nil {
				return fmt.Errorf("hreflang URL %q: %w", link.URL, err)
			}
			target := references[href.String()]
			if target == nil || link.Rel != "alternate" || link.Language == "" || !strings.EqualFold(link.Language, target.language) {
				return fmt.Errorf("invalid hreflang target %q (%s)", link.URL, link.Language)
			}
			link.URL = target.canonical
		}
	}
	// Do not invent missing normal pages: native collection remains responsible
	// for those. Only the seven reviewed fragments bypass its collection.
	raw := make(map[string]bool)
	for _, route := range vanity.Routes() {
		raw[route.File()] = true
		page := pages[route.File()]
		if page == nil {
			return fmt.Errorf("restored vanity page %s is missing", route.File())
		}
		if entries[page.file] == nil {
			lastmod, err := gitOutput(ctx, root, "log", "-1", "--format=%cI", "--", page.file)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			date := strings.TrimSpace(string(lastmod))
			if err != nil {
				date = ""
			}
			if date != "" {
				if _, err := time.Parse(time.RFC3339, date); err != nil {
					return fmt.Errorf("invalid Git lastmod for %s: %w", page.file, err)
				}
			}
			sitemap.URLs = append(sitemap.URLs, siteSitemapURL{Location: page.canonical, Lastmod: date})
		}
	}
	for file := range pages {
		if entries[file] == nil && !raw[file] {
			return fmt.Errorf("native sitemap omitted deployed HTML page %s", file)
		}
	}
	// Check the existing translation groups after canonical resolution. A bad
	// reference must fail before the staged output can replace production.
	byCanonical := make(map[string]*siteSitemapURL)
	for i := range sitemap.URLs {
		entry := &sitemap.URLs[i]
		byCanonical[entry.Location] = entry
	}
	for _, entry := range sitemap.URLs {
		languages := make(map[string]bool)
		source := references[entry.Location]
		for _, link := range entry.Links {
			language := strings.ToLower(link.Language)
			if languages[language] {
				return fmt.Errorf("duplicate hreflang %s on %s", link.Language, entry.Location)
			}
			languages[language] = true
			reciprocal := false
			for _, other := range byCanonical[link.URL].Links {
				if other.URL == entry.Location && strings.EqualFold(other.Language, source.language) {
					reciprocal = true
				}
			}
			if !reciprocal {
				return fmt.Errorf("nonreciprocal hreflang from %s to %s", entry.Location, link.URL)
			}
		}
	}
	for i := range sitemap.URLs {
		sort.Slice(sitemap.URLs[i].Links, func(a, b int) bool {
			return sitemap.URLs[i].Links[a].Language < sitemap.URLs[i].Links[b].Language
		})
	}
	sort.Slice(sitemap.URLs, func(i, j int) bool { return sitemap.URLs[i].Location < sitemap.URLs[j].Location })
	sitemap.XMLName = xml.Name{Local: "urlset"}
	sitemap.XMLNS = sitemapNamespace
	sitemap.XHTML = ""
	for _, entry := range sitemap.URLs {
		if len(entry.Links) != 0 {
			sitemap.XHTML = xhtmlNamespace
		}
	}
	output, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return err
	}
	output = append([]byte(xml.Header), output...)
	output = append(output, '\n')
	return os.WriteFile(path, output, 0644)
}

func localSitemapURL(raw string, base *url.URL) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("expected an absolute HTTP(S) URL without credentials, query or fragment")
	}
	if base != nil && (u.Scheme != base.Scheme || !strings.EqualFold(u.Host, base.Host)) {
		return nil, fmt.Errorf("URL is outside configured site.baseurl")
	}
	return u, nil
}

func collectSitemapPages(deploy string, base *url.URL) (map[string]*sitemapPage, map[string]*sitemapPage, error) {
	pages, references := make(map[string]*sitemapPage), make(map[string]*sitemapPage)
	err := filepath.WalkDir(deploy, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing deploy symlink %s", path)
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular HTML page %s", path)
		}
		rel, err := filepath.Rel(deploy, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		canonical, language, err := sitemapHTMLMetadata(body)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		u, err := localSitemapURL(canonical, base)
		if err != nil {
			return fmt.Errorf("%s canonical: %w", rel, err)
		}
		native, _ := url.Parse(strings.TrimRight(base.String(), "/") + "/" + rel)
		expected := native.String()
		if filepath.Base(rel) == "index.html" {
			expected = strings.TrimSuffix(expected, "index.html")
		}
		if u.String() != expected {
			return fmt.Errorf("%s canonical %q does not resolve to its deployed page %q", rel, canonical, expected)
		}
		if references[u.String()] != nil {
			return fmt.Errorf("duplicate canonical URL %s", canonical)
		}
		page := &sitemapPage{file: rel, canonical: u.String(), language: language}
		pages[rel], references[u.String()], references[native.String()] = page, page, page
		return nil
	})
	if err == nil && len(pages) == 0 {
		err = fmt.Errorf("no deployed HTML pages")
	}
	return pages, references, err
}

func sitemapHTMLMetadata(body []byte) (canonical, language string, err error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	inHead, rawText := true, ""
	count := 0
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				return "", "", z.Err()
			}
			break
		}
		token := z.Token()
		if rawText != "" {
			if kind == html.EndTagToken && token.Data == rawText {
				rawText = ""
			}
			continue
		}
		if kind == html.TextToken && strings.TrimSpace(token.Data) != "" {
			inHead = false
		}
		if kind == html.EndTagToken && (token.Data == "head" || token.Data == "body" || token.Data == "html") {
			inHead = false
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		attrs := make(map[string]string)
		for _, attr := range token.Attr {
			if _, exists := attrs[attr.Key]; !exists {
				attrs[attr.Key] = attr.Val
			}
		}
		if token.Data == "html" {
			language = attrs["lang"]
		}
		switch token.Data {
		case "html", "head", "link", "meta", "base", "basefont", "bgsound":
		case "title", "style", "script", "noscript":
			rawText = token.Data
		default:
			inHead = false
		}
		if token.Data == "link" {
			for _, rel := range strings.Fields(attrs["rel"]) {
				if strings.EqualFold(rel, "canonical") {
					if !inHead {
						return "", "", fmt.Errorf("canonical link outside head")
					}
					canonical = attrs["href"]
					count++
				}
			}
		}
	}
	if count != 1 || canonical == "" {
		return "", "", fmt.Errorf("expected exactly one nonempty canonical link, got %d", count)
	}
	return canonical, language, nil
}

// Reject an unexpected native XML shape instead of silently discarding fields
// that a future generator might add. Only the site's current single sitemap
// format is supported by this post-build adjustment.
func readNativeSitemap(data []byte, sitemap *siteSitemap) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var parents []xml.Name
	roots, locations, lastmods := 0, 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			valid := false
			switch len(parents) {
			case 0:
				roots++
				valid = roots == 1 && token.Name == (xml.Name{Space: sitemapNamespace, Local: "urlset"})
			case 1:
				valid = token.Name == (xml.Name{Space: sitemapNamespace, Local: "url"})
				locations, lastmods = 0, 0
			case 2:
				valid = (token.Name.Space == sitemapNamespace && (token.Name.Local == "loc" || token.Name.Local == "lastmod")) || token.Name == (xml.Name{Space: xhtmlNamespace, Local: "link"})
				if token.Name.Local == "loc" {
					locations++
				}
				if token.Name.Local == "lastmod" {
					lastmods++
				}
				valid = valid && locations <= 1 && lastmods <= 1
			}
			if !valid {
				return fmt.Errorf("unsupported native sitemap element {%s}%s", token.Name.Space, token.Name.Local)
			}
			attributes := make(map[xml.Name]bool)
			for _, attr := range token.Attr {
				declaration := (attr.Name.Local == "xmlns" && attr.Name.Space == "") || attr.Name.Space == "xmlns"
				known := declaration && (attr.Value == sitemapNamespace || attr.Value == xhtmlNamespace)
				if token.Name.Space == xhtmlNamespace && attr.Name.Space == "" && (attr.Name.Local == "rel" || attr.Name.Local == "hreflang" || attr.Name.Local == "href") {
					known = true
				}
				if !known || attributes[attr.Name] {
					return fmt.Errorf("unsupported or duplicate sitemap attribute %s", attr.Name.Local)
				}
				attributes[attr.Name] = true
			}
			parents = append(parents, token.Name)
		case xml.EndElement:
			if len(parents) == 2 && locations != 1 {
				return fmt.Errorf("sitemap page must have exactly one loc")
			}
			parents = parents[:len(parents)-1]
		case xml.CharData:
			if strings.TrimSpace(string(token)) != "" && (len(parents) != 3 || parents[2].Space != sitemapNamespace) {
				return fmt.Errorf("unexpected text in native sitemap")
			}
		}
	}
	if err := xml.Unmarshal(data, sitemap); err != nil {
		return err
	}
	if len(sitemap.URLs) == 0 {
		return fmt.Errorf("native sitemap contains no pages")
	}
	return nil
}
