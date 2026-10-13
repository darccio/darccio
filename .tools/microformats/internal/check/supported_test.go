package check

import (
	"strings"
	"testing"
)

func TestEnglishSlashEntryKeepsEnglishLanguageAndCanonicalIdentity(t *testing.T) {
	post := examplePost()
	canonical := homeURL + "en/now/"
	input := strings.ReplaceAll(fixture(t, "post.html"), post.URL, canonical)
	input = strings.Replace(input, `<time class="dt-published" datetime="2024-07-01T10:00:00+02:00">1 July 2024</time>`, `Now`, 1)
	expected, err := parseHTML(post.Body)
	if err != nil {
		t.Fatal(err)
	}
	doc, data := parsed(t, input, canonical)
	if err := validateSlashEntry(doc, data, expected, canonical, post.Updated, "en"); err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{`class="h-entry" lang="en"`, `class="h-entry" lang="ca"`},
		{canonical, homeURL + "now/"},
		{`class="p-author h-card" href="https://dario.cat/"`, `class="p-author h-card" href="https://dario.cat/en/"`},
	} {
		if !strings.Contains(input, change[0]) {
			t.Fatalf("mutation does not match fixture: %s", change[0])
		}
		doc, data := parsed(t, strings.ReplaceAll(input, change[0], change[1]), canonical)
		if err := validateSlashEntry(doc, data, expected, canonical, post.Updated, "en"); err == nil {
			t.Fatal("translated language, URL or personal identity regression was accepted")
		}
	}
}

func TestEnglishContactAdapterOnlyReadsItsReviewedLiteralPartial(t *testing.T) {
	root := t.TempDir()
	file := "en/contact/index.md"
	source := `<!--
entry: true
-->
<h1 class="p-name">Contact</h1>
<embed src="../../.zas/partials/contact-en.html" type="text/html">
`
	put(t, root, file, source)
	put(t, root, ".zas/partials/contact-en.html", `<nav aria-label="Contact"><a class="u-email" href="mailto:cc@dario.cat">cc@dario.cat</a></nav>`)
	_, doc, err := publicSource(root, file)
	if err != nil || doc == nil || !strings.Contains(textContent(doc), email) {
		t.Fatalf("reviewed English contact embed did not render: %v", err)
	}
	for _, path := range []string{"../.zas/partials/contact.html", "../../CV/MAC.json", "../../.zas/partials/other.html"} {
		put(t, root, file, strings.Replace(source, "../../.zas/partials/contact-en.html", path, 1))
		if _, _, err := publicSource(root, file); err == nil {
			t.Fatalf("unreviewed embed was accepted: %s", path)
		}
	}
}

func TestDatedSlashEntryFixture(t *testing.T) {
	post := examplePost()
	input := strings.ReplaceAll(fixture(t, "post.html"), post.URL, homeURL+"now/")
	input = strings.ReplaceAll(input, `lang="en"`, `lang="ca"`)
	input = strings.Replace(input, `<time class="dt-published" datetime="2024-07-01T10:00:00+02:00">1 July 2024</time>`, `Now`, 1)
	expected, err := parseHTML(post.Body)
	if err != nil {
		t.Fatal(err)
	}
	doc, data := parsed(t, input, homeURL+"now/")
	if err := validateSlashEntry(doc, data, expected, homeURL+"now/", post.Updated, "ca"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, from, to string }{
		{"missing entry", `class="h-entry"`, `class="entry"`},
		{"invented publication", `Now</a>`, `<time class="dt-published" datetime="0001-01-01T00:00:00Z">Now</time></a>`},
		{"changed update", `class="dt-updated" datetime="2024-08-01T12:30:00+02:00"`, `class="dt-updated" datetime="2024-08-01T12:30:00+01:00"`},
		{"changed content", `Some article text`, `Different article text`},
		{"missing author card", `class="p-author h-card"`, `class="p-author"`},
		{"URL property leakage", `<div class="e-content">`, `<div class="e-content"><a class="u-url" href="https://sifa.id/p/dario.cat">Sifa</a>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(input, test.from) {
				t.Fatal("fixture mutation no longer matches")
			}
			doc, data := parsed(t, strings.Replace(input, test.from, test.to, 1), homeURL+"now/")
			if err := validateSlashEntry(doc, data, expected, homeURL+"now/", post.Updated, "ca"); err == nil {
				t.Fatal("slash regression was accepted")
			}
		})
	}
}

func TestProjectsSlashesAndEventsFixtures(t *testing.T) {
	feedPrefix := `<section class="h-feed"><h1 class="p-name">Index</h1><a class="u-url" href="PAGEURL" hidden>Index</a><a class="p-author h-card" href="https://dario.cat/" hidden><span class="p-name">Dario Castañé</span></a>`
	cases := []struct {
		slug, typ, input string
		changes          [][2]string
	}{
		{"projects", "h-item", feedPrefix + `<section class="h-item"><h2 class="p-name">Zas</h2><p class="p-summary">A Go generator.</p><a class="u-url" href="https://github.com/darccio/zas#readme">Documentation</a></section></section>`, [][2]string{
			{`class="h-item"`, `class="item"`},
			{`A Go generator.`, `Different summary.`},
			{`https://github.com/darccio/zas#readme`, `https://example.com/`},
		}},
		{"slashes", "h-entry", feedPrefix + `<ul><li class="h-entry"><a class="p-name u-url" href="/now/">Ara</a><time class="dt-updated" datetime="2026-10-11">11 October 2026</time></li></ul></section>`, [][2]string{
			{`class="h-entry"`, `class="entry"`},
			{`datetime="2026-10-11"`, `datetime="2026-10-11T00:00:00Z"`},
			{`href="/now/"`, `href="/missing/"`},
		}},
		{"talks", "h-event", `<h3>Undated talk</h3><section class="h-event"><h3 class="p-name">An online talk</h3><time class="dt-start" datetime="2020-06-03">3 June 2020</time><span class="p-location">en línia</span><a class="u-url" href="https://www.youtube.com/watch?v=i0UOYHs0eDI">Recording</a></section>`, [][2]string{
			{`class="h-event"`, `class="event"`},
			{`datetime="2020-06-03"`, `datetime="2020-06-03T19:00:00+02:00"`},
			{`class="p-location"`, `class="location"`},
			{`<h3>Undated talk</h3>`, ``},
		}},
	}
	for _, test := range cases {
		t.Run(test.slug, func(t *testing.T) {
			pageURL := homeURL + test.slug + "/"
			input := strings.ReplaceAll(test.input, "PAGEURL", pageURL)
			expected, source := parsed(t, input, pageURL)
			doc, output := parsed(t, input, pageURL)
			if err := validateSupportedChildren(test.slug, test.typ, pageURL, expected, doc, source, output); err != nil {
				t.Fatal(err)
			}
			for _, change := range test.changes {
				doc, output := parsed(t, strings.Replace(input, change[0], change[1], 1), pageURL)
				if err := validateSupportedChildren(test.slug, test.typ, pageURL, expected, doc, source, output); err == nil {
					t.Fatalf("supported markup regression accepted: %v", change)
				}
			}
		})
	}
}
