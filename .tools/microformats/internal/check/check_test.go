package check

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"willnorris.com/go/microformats"
	"willnorris.com/go/microformats/rhc"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	input, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(input)
}

func parsed(t *testing.T, input, pageURL string) (*htmlNode, *microformats.Data) {
	t.Helper()
	doc, err := parseHTML([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		t.Fatal(err)
	}
	return doc, microformats.ParseNode(doc, base)
}

func examplePost() Post {
	date, _ := time.Parse(time.RFC3339, "2024-07-01T10:00:00+02:00")
	updated, _ := time.Parse(time.RFC3339, "2024-08-01T12:30:00+02:00")
	return Post{
		Source: "en/posts/example/index.md", File: "en/posts/example/index.html", URL: "https://dario.cat/en/posts/example/",
		Title: "Example article", Language: "en", Summary: "A sample article.", Date: date, Updated: updated, Tags: []string{"go", "rss"},
		Body: []byte(`<h1>Example article</h1><p>Some article text with a <a href="/projects/">project link</a>.</p>`),
	}
}

func TestPostFixtureAndSemanticRegressions(t *testing.T) {
	input := fixture(t, "post.html")
	post := examplePost()
	doc, data := parsed(t, input, post.URL)
	if err := validatePost(post, doc, data); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, from, to string }{
		{"missing entry", `class="h-entry"`, `class="entry"`},
		{"wrong canonical", `rel="canonical" href="https://dario.cat/en/posts/example/"`, `rel="canonical" href="https://dario.cat/en/posts/example/index.html"`},
		{"wrong Open Graph URL", `property="og:url" content="https://dario.cat/en/posts/example/"`, `property="og:url" content="https://dario.cat/en/posts/other/"`},
		{"wrong publication offset", `class="dt-published" datetime="2024-07-01T10:00:00+02:00"`, `class="dt-published" datetime="2024-07-01T10:00:00+01:00"`},
		{"wrong updated", `class="dt-updated" datetime="2024-08-01T12:30:00+02:00"`, `class="dt-updated" datetime="2024-08-02T12:30:00+02:00"`},
		{"wrong author", `class="p-author h-card" href="https://dario.cat/"`, `class="p-author h-card" href="https://github.com/darccio"`},
		{"missing nested author", `class="p-author h-card"`, `class="p-author"`},
		{"wrong article language", `<article class="h-entry" lang="en">`, `<article class="h-entry" lang="ca">`},
		{"wrong document language", `<html lang="en">`, `<html lang="ca">`},
		{"wrong summary", `class="p-summary" hidden>A sample article.`, `class="p-summary" hidden>A different article.`},
		{"wrong category", `class="p-category">go`, `class="p-category">ruby`},
		{"empty content", `<div class="e-content"><h1 class="p-name">Example article</h1><p>Some article text with a <a href="/projects/">project link</a>.</p></div>`, `<h1 class="p-name">Example article</h1><div class="e-content"></div>`},
		{"page chrome in content", `<div class="e-content">`, `<div class="e-content"><nav>Site navigation</nav>`},
		{"content destination drift", `href="/projects/"`, `href="https://example.com/"`},
		{"second H1", `<main>`, `<main><h1>Site title</h1>`},
		{"organization header identity", `<span class="p-name">Dario Castañé</span></a></div></header>`, `<span class="p-name">Datadog</span></a></div></header>`},
		{"missing Sifa verification", `<link rel="me" href="https://sifa.id/p/dario.cat">`, ``},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(input, test.from) {
				t.Fatalf("fixture mutation no longer matches %q", test.from)
			}
			doc, data := parsed(t, strings.Replace(input, test.from, test.to, 1), post.URL)
			if err := validatePost(post, doc, data); err == nil {
				t.Fatal("semantic regression was accepted")
			}
		})
	}
}

func TestProfileVerificationLinkMustBeInHead(t *testing.T) {
	input := fixture(t, "identity.html")
	link := `<link rel="me" href="https://sifa.id/p/dario.cat">`
	input = strings.Replace(input, link, "", 1)
	input = strings.Replace(input, "<body>", "<body>"+link, 1)
	doc, data := parsed(t, input, homeURL)
	if !contains(data.Rels["me"], "https://sifa.id/p/dario.cat") {
		t.Fatal("fixture must retain the profile relation in the body")
	}
	if err := validateProfiles(doc, data); err == nil {
		t.Fatal("body-only verification link accepted as head discovery")
	}
}

func TestAboutCardUsesItsOwnDocumentURL(t *testing.T) {
	deploy := t.TempDir()
	input := strings.Replace(fixture(t, "identity.html"), `src="/static/avatar.webp"`, `src="static/avatar.webp"`, 1)
	put(t, deploy, "about/index.html", input)
	_, data, err := readPage(deploy, "about/index.html", homeURL+"about/")
	if err != nil {
		t.Fatal(err)
	}
	card := rhc.RepresentativeHcard(data, homeURL+"about/")
	if card == nil || validateCard(card, true) == nil {
		t.Fatal("about-relative avatar must resolve under /about/ and fail local identity photo check")
	}
}

func TestRepresentativeIdentitySkipsOrganizations(t *testing.T) {
	input := fixture(t, "identity.html")
	for _, pageURL := range []string{homeURL, homeURL + "about/"} {
		t.Run(pageURL, func(t *testing.T) {
			doc, data := parsed(t, input, pageURL)
			card := rhc.RepresentativeHcard(data, pageURL)
			if card == nil {
				t.Fatal("no representative h-card")
			}
			if err := validateCard(card, true); err != nil {
				t.Fatal(err)
			}
			if err := validateProfiles(doc, data); err != nil {
				t.Fatal(err)
			}
		})
	}
	cases := []struct{ name, from, to string }{
		{"organization wins discovery", `https://www.datadoghq.com/`, homeURL},
		{"wrong owner", `<h1 class="p-name">Dario Castañé</h1>`, `<h1 class="p-name">Datadog</h1>`},
		{"remote avatar", `src="/static/avatar.webp"`, `src="https://gravatar.com/avatar/example"`},
		{"obfuscated email", `href="mailto:cc@dario.cat"`, `href="/cdn-cgi/l/email-protection"`},
		{"wrong uid", `class="u-url u-uid" rel="me"`, `class="u-url" rel="me"`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, data := parsed(t, strings.ReplaceAll(input, test.from, test.to), homeURL)
			card := rhc.RepresentativeHcard(data, homeURL)
			if card != nil && validateCard(card, true) == nil {
				t.Fatal("invalid representative identity was accepted")
			}
		})
	}
}

func TestArchiveFixtureAndSemanticRegressions(t *testing.T) {
	input := fixture(t, "archive.html")
	posts := []Post{examplePost()}
	archiveURL := homeURL + "posts/"
	doc, data := parsed(t, input, archiveURL)
	if err := validateArchive(archiveURL, "", posts, doc, data); err != nil {
		t.Fatal(err)
	}
	badge := `<span class="lang-tag" aria-hidden="true">en</span>`
	spoken := `<span class="visually-hidden" lang="ca"> (en anglès)</span>`
	cases := []struct{ name, from, to string }{
		{"missing entry", `class="h-entry"`, `class="entry"`},
		{"duplicate entry", `<ul>`, `<ul><li class="h-entry" lang="en"><a class="p-name u-url u-uid" href="https://dario.cat/en/posts/example/">Example article</a></li>`},
		{"missing visible language", badge, ``},
		{"wrong badge code", badge, `<span class="lang-tag" aria-hidden="true">ca</span>`},
		{"badge lacks aria-hidden", badge, `<span class="lang-tag">en</span>`},
		{"badge aria-hidden false", badge, `<span class="lang-tag" aria-hidden="false">en</span>`},
		{"duplicate language badge", badge, badge + badge},
		{"hidden language label", badge, `<span class="lang-tag" aria-hidden="true" hidden>en</span>`},
		{"hidden language ancestor", badge, `<span hidden>` + badge + `</span>`},
		{"hidden language style", badge, `<span class="lang-tag" aria-hidden="true" style="display: none">en</span>`},
		{"visually hidden badge", badge, `<span class="lang-tag visually-hidden" aria-hidden="true">en</span>`},
		{"hidden archive ancestor", `<ul>`, `<ul hidden>`},
		{"missing spoken language", spoken, ``},
		{"wrong full language name", spoken, `<span class="visually-hidden" lang="ca"> (en català)</span>`},
		{"wrong label localization", spoken, `<span class="visually-hidden" lang="ca"> (in English)</span>`},
		{"spoken language lacks page lang", spoken, `<span class="visually-hidden"> (en anglès)</span>`},
		{"spoken language inherits row lang", spoken, `<span class="visually-hidden" lang="en"> (en anglès)</span>`},
		{"aria-hidden spoken language", spoken, `<span class="visually-hidden" lang="ca" aria-hidden="true"> (en anglès)</span>`},
		{"aria-hidden spoken ancestor", spoken, `<span aria-hidden="true">` + spoken + `</span>`},
		{"hidden spoken language", spoken, `<span class="visually-hidden" lang="ca" hidden> (en anglès)</span>`},
		{"inert spoken language", spoken, `<span class="visually-hidden" lang="ca" inert> (en anglès)</span>`},
		{"missing external spoken language", `<span class="visually-hidden" lang="ca"> (en castellà)</span>`, ``},
		{"language labels in p-name", `Example article</a> ` + badge + spoken, `Example article ` + badge + spoken + `</a>`},
		{"wrong language", `class="h-entry" lang="en"`, `class="h-entry" lang="ca"`},
		{"wrong date", `2024-07-01T10:00:00+02:00`, `2024-07-01T10:00:00+01:00`},
		{"wrong URL", `https://dario.cat/en/posts/example/`, `https://dario.cat/en/posts/example/index.html`},
		{"draft exposure", `https://dario.cat/en/posts/example/`, `https://dario.cat/posts/delete-facebook-3/`},
		{"external mistaken for local", `class="h-cite"`, `class="h-entry"`},
		{"missing external", `<li class="h-cite" lang="ca">`, `<li lang="ca">`},
		{"external URL drift", `https://www.nuvol.com/pantalles/per-al-lusions-22931`, `https://example.com/`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(input, test.from) {
				t.Fatal("fixture mutation no longer matches")
			}
			doc, data := parsed(t, strings.Replace(input, test.from, test.to, 1), archiveURL)
			if err := validateArchive(archiveURL, "", posts, doc, data); err == nil {
				t.Fatal("archive regression was accepted")
			}
		})
	}
}

func TestLanguageArchivesUseTheirOwnSpokenLanguage(t *testing.T) {
	var lines []string
	for _, line := range strings.Split(fixture(t, "archive.html"), "\n") {
		if !strings.Contains(line, `class="h-cite"`) {
			lines = append(lines, line)
		}
	}
	for _, tc := range []struct{ language, label string }{{"en", "in English"}, {"es", "en español"}} {
		t.Run(tc.language, func(t *testing.T) {
			post := examplePost()
			post.Language = tc.language
			post.URL = homeURL + tc.language + "/posts/example/"
			archiveURL := homeURL + tc.language + "/posts/"
			input := strings.Join(lines, "\n")
			input = strings.ReplaceAll(input, `lang="ca"`, `lang="`+tc.language+`"`)
			input = strings.Replace(input, `class="h-entry" lang="en"`, `class="h-entry" lang="`+tc.language+`"`, 1)
			input = strings.Replace(input, `href="https://dario.cat/posts/"`, `href="`+archiveURL+`"`, 1)
			input = strings.Replace(input, `href="https://dario.cat/en/posts/example/"`, `href="`+post.URL+`"`, 1)
			input = strings.Replace(input, `aria-hidden="true">en</span>`, `aria-hidden="true">`+tc.language+`</span>`, 1)
			input = strings.Replace(input, `(en anglès)`, `(`+tc.label+`)`, 1)
			doc, data := parsed(t, input, archiveURL)
			if err := validateArchive(archiveURL, tc.language, []Post{post}, doc, data); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPartialPublicationDateIsNotInvented(t *testing.T) {
	input := fixture(t, "archive.html")
	post := examplePost()
	post.Date = time.Time{}
	post.CalendarDate = "2024-07-01"
	input = strings.Replace(input, `class="dt-published" datetime="2024-07-01T10:00:00+02:00"`, `datetime="2024-07-01"`, 1)
	doc, data := parsed(t, input, homeURL+"posts/")
	if err := validateArchive(homeURL+"posts/", "", []Post{post}, doc, data); err != nil {
		t.Fatal(err)
	}
	for _, datetime := range []string{"0001-01-01T00:00:00Z", "2024-07-01T00:00:00+02:00"} {
		broken := strings.Replace(input, `datetime="2024-07-01"`, fmt.Sprintf(`class="dt-published" datetime=%q`, datetime), 1)
		doc, data := parsed(t, broken, homeURL+"posts/")
		if err := validateArchive(homeURL+"posts/", "", []Post{post}, doc, data); err == nil {
			t.Fatalf("invented timestamp accepted: %s", datetime)
		}
	}
}

func put(t *testing.T, root, file, input string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceInventoryUsesPinnedMetadataHelpersAndExcludesDraft(t *testing.T) {
	root := t.TempDir()
	put(t, root, "en/posts/example/index.md", "<!--\narticle: true\ndate: '2024-07-01T10:00:00+02:00'\nupdated: '2024-08-01T12:30:00+02:00'\nlanguage: en\nsummary: A sample article.\ntags: [go, rss]\n-->\n# Example article\n\nSome article text with a [project link](/projects/).\n")
	put(t, root, "posts/draft/index.md", "<!--\narticle: true\ndraft: true\n-->\n# Draft\n")
	put(t, root, "posts/index.md", "# Archive\n")
	inv, err := loadInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Posts) != 1 || len(inv.Excluded) != 1 || inv.Posts[0].URL != examplePost().URL || inv.Posts[0].Date.Format(time.RFC3339) != "2024-07-01T10:00:00+02:00" {
		t.Fatalf("wrong source inventory: %+v", inv)
	}
	put(t, root, "en/posts/example/index.md", "<!--\narticle: true\ndate: '2024-07-01'\n-->\n# Example\n")
	if _, err := loadInventory(root); err == nil {
		t.Fatal("date-only source accepted as an instant")
	}
}
