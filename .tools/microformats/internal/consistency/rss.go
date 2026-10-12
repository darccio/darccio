package consistency

import (
	"bytes"
	"encoding/xml"
	"fmt"
	stdhtml "html"
	"io"
	"slices"
	"sort"
	"strings"
	"time"
)

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel []struct {
		Self []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
			Type string `xml:"type,attr"`
		} `xml:"http://www.w3.org/2005/Atom link"`
		Title         []string  `xml:"title"`
		Link          []string  `xml:"link"`
		Description   []string  `xml:"description"`
		Language      []string  `xml:"language"`
		LastBuildDate []string  `xml:"lastBuildDate"`
		Items         []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       []string `xml:"title"`
	Link        []string `xml:"link"`
	PubDate     []string `xml:"pubDate"`
	Description []string `xml:"description"`
	Language    string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Creator     []string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	DCLanguage  []string `xml:"http://purl.org/dc/elements/1.1/ language"`
	Categories  []string `xml:"category"`
	Content     []string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	GUID        []struct {
		Value     string `xml:",chardata"`
		Permalink string `xml:"isPermaLink,attr"`
	} `xml:"guid"`
}

func expectedFeeds(config siteConfig, posts []sourcePost) map[string][]sourcePost {
	var dated []sourcePost
	for _, post := range posts {
		if !post.Excluded && post.Feed {
			dated = append(dated, post)
		}
	}
	sort.Slice(dated, func(i, j int) bool {
		if dated[i].Date.Equal(dated[j].Date) {
			return dated[i].URL < dated[j].URL
		}
		return dated[i].Date.After(dated[j].Date)
	})
	sets := map[string][]sourcePost{"feed.xml": dated}
	if config.Site.FeedPerLanguage {
		for _, post := range dated {
			if post.Language != config.Site.Language {
				file := post.Language + "/feed.xml"
				sets[file] = append(sets[file], post)
			}
		}
	}
	for name, posts := range sets {
		if len(posts) > config.Site.FeedLimit {
			sets[name] = posts[:config.Site.FeedLimit]
		}
	}
	return sets
}

func rssTime(values []string, expected time.Time, label string) error {
	if expected.IsZero() {
		if len(values) != 0 {
			return fmt.Errorf("%s invents an update instant", label)
		}
		return nil
	}
	if len(values) != 1 {
		return fmt.Errorf("%s needs one RFC1123Z timestamp, got %q", label, values)
	}
	value, err := time.Parse(time.RFC1123Z, values[0])
	// RSS's RFC822 timestamps intentionally carry whole seconds; the richer
	// formats preserve the full RFC3339 instant independently.
	if err != nil || !value.Equal(expected.Truncate(time.Second)) {
		return fmt.Errorf("%s got %q, want instant %s", label, values[0], expected.Truncate(time.Second).Format(time.RFC3339))
	}
	return nil
}

func checkFeed(config siteConfig, file string, expected []sourcePost, rendered map[string]renderedPost, input []byte) error {
	// encoding/xml's unqualified field names also match foreign namespaces.
	// Validate the extension vocabulary separately before decoding fields.
	vocabulary := xml.NewDecoder(bytes.NewReader(input))
	for {
		token, err := vocabulary.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("RSS XML: %w", err)
		}
		if element, ok := token.(xml.StartElement); ok && element.Name.Space != "" {
			name := element.Name
			allowed := name.Space == "http://www.w3.org/2005/Atom" && name.Local == "link" ||
				name.Space == "http://purl.org/dc/elements/1.1/" && (name.Local == "creator" || name.Local == "language") ||
				name.Space == "http://purl.org/rss/1.0/modules/content/" && name.Local == "encoded"
			if !allowed {
				return fmt.Errorf("unexpected namespaced RSS element %s %s", name.Space, name.Local)
			}
		}
	}
	var feed rssFeed
	decoder := xml.NewDecoder(bytes.NewReader(input))
	if err := decoder.Decode(&feed); err != nil {
		return fmt.Errorf("RSS XML: %w", err)
	}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if value, ok := token.(xml.CharData); !ok || strings.TrimSpace(string(value)) != "" {
			return fmt.Errorf("extra XML after RSS document")
		}
	}
	if feed.XMLName.Space != "" || feed.Version != "2.0" || len(feed.Channel) != 1 {
		return fmt.Errorf("expected one RSS 2.0 channel")
	}
	channel := feed.Channel[0]
	language := config.Site.Language
	title := config.Site.FeedTitle
	if file != "feed.xml" {
		language = strings.TrimSuffix(file, "/feed.xml")
		label := map[string]string{"en": "English", "es": "Español"}[language]
		if label == "" {
			label = language
		}
		title += " (" + label + ")"
	}
	for _, value := range []struct {
		got            []string
		want, property string
	}{
		{channel.Title, title, "channel title"},
		{channel.Link, config.Site.BaseURL + "/", "channel URL"},
		{channel.Description, config.Site.FeedDescription, "channel description"},
		{channel.Language, language, "channel language"},
	} {
		if err := one(value.got, value.want, value.property); err != nil {
			return err
		}
	}
	if len(channel.Self) != 1 || channel.Self[0].Href != config.Site.BaseURL+"/"+file || channel.Self[0].Rel != "self" || channel.Self[0].Type != "application/rss+xml" {
		return fmt.Errorf("RSS self link does not match its canonical feed URL")
	}
	if len(channel.Items) != len(expected) {
		return fmt.Errorf("RSS has %d items, expected ordered/capped set of %d", len(channel.Items), len(expected))
	}
	seen := make(map[string]bool)
	var latest time.Time
	for index, item := range channel.Items {
		post := expected[index]
		if len(item.GUID) != 1 || item.GUID[0].Permalink != "true" || item.GUID[0].Value != post.URL {
			return fmt.Errorf("item %d GUID does not match ordered source URL %q", index, post.URL)
		}
		guid := item.GUID[0].Value
		if seen[guid] {
			return fmt.Errorf("duplicate RSS GUID %q", guid)
		}
		seen[guid] = true
		for _, value := range []struct {
			got            []string
			want, property string
		}{
			{item.Title, post.Title, "item title"}, {item.Link, post.URL, "item URL"},
			{item.Description, stdhtml.EscapeString(post.Summary), "item summary"},
			{item.Creator, post.Creator, "item creator"}, {item.DCLanguage, post.Language, "item dc:language"},
		} {
			if err := one(value.got, value.want, value.property); err != nil {
				return fmt.Errorf("%s: %w", post.Path, err)
			}
		}
		if item.Language != post.Language || !slices.Equal(item.Categories, post.Tags) {
			return fmt.Errorf("%s: RSS language or categories disagree with source", post.Path)
		}
		if err := rssTime(item.PubDate, post.Date, "RSS publication"); err != nil {
			return fmt.Errorf("%s: %w", post.Path, err)
		}
		if len(item.Content) != 1 || strings.TrimSpace(item.Content[0]) == "" {
			return fmt.Errorf("%s: missing full RSS content", post.Path)
		}
		doc, err := parseHTML([]byte(item.Content[0]))
		if err != nil || text(doc) != rendered[post.Path].Text {
			return fmt.Errorf("%s: full RSS text differs from h-entry content", post.Path)
		}
		if post.Updated.After(latest) {
			latest = post.Updated
		}
	}
	return rssTime(channel.LastBuildDate, latest, "RSS lastBuildDate")
}
