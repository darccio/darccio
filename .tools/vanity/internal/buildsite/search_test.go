package buildsite

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSiteSearchBuildsReviewedPublicCatalogueWithoutBodies(t *testing.T) {
	stage, deploy, base := siteSearchFixture(t)
	entries := []siteSearchEntry{
		{URL: "/z/", Title: `Go <maps> & "quotes" [en]`, Summary: `Safe <script>text</script> & "data"`, Language: "en"},
		{URL: "/a/", Title: "Una pàgina", Language: "ca"},
		{URL: "/zas/configuration/", Title: "Configuration", Summary: "Public project docs", Language: "en"},
	}
	for _, entry := range entries {
		siteSearchPutPage(t, stage, deploy, entry, "")
	}
	for _, item := range []struct{ path, metadata string }{
		{"private-post", "private: true"},
		{"draft-post", "draft: true"},
		{"unpublished", "publish: false"},
		{"no-markdown", "publish_markdown: false"},
		{"ancestor/sub/post", ""},
		{"nearest/post", ""},
	} {
		entry := siteSearchEntry{URL: "/" + item.path + "/", Title: "DO_NOT_INDEX " + item.path, Summary: "DO_NOT_INDEX", Language: "ca"}
		entries = append(entries, entry)
		siteSearchPutPage(t, stage, deploy, entry, item.metadata)
	}
	put(t, stage, "ancestor/.zas.yml", "private: true\n")
	put(t, stage, "ancestor/sub/.zas.yml", "language: ca\n")
	put(t, stage, "nearest/.zas.yml", "publish: false\n")
	// These paths must be omitted before their absent, invalid or symlinked
	// content can be read. No CV content is needed to enforce the exclusion.
	for _, path := range []string{"/CV/MAC.html", "/cv/", "/en/cv/", "/private/page/", "/docs/site-plan/T23.html", "/README.html", "/en/search/", "/.secret/", "/stale/"} {
		entries = append(entries, siteSearchEntry{URL: path, Title: "DO_NOT_INDEX " + path, Summary: "DO_NOT_INDEX", Language: "ca"})
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "never-read"), filepath.Join(stage, "CV")); err != nil {
		t.Fatal(err)
	}
	siteSearchPutCatalogues(t, deploy, entries)
	index, err := finalizeSiteSearch(stage, deploy, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(index) != 3 || index[0].URL != "/a/" || index[1].URL != "/z/" || index[2].URL != "/zas/configuration/" {
		t.Fatalf("unexpected catalogue: %+v", index)
	}
	for _, language := range []string{"ca", "en", "es"} {
		for _, extension := range []string{"html", "md"} {
			body := readMarkdownDiscoveryFile(t, deploy, agentContextPrefix(language)+"search/index."+extension)
			for _, forbidden := range []string{"DO_NOT_INDEX", "SECRET_BODY", "<script>text</script>", "&amp;lt;maps"} {
				if bytes.Contains(body, []byte(forbidden)) {
					t.Errorf("%s %s leaked %q", language, extension, forbidden)
				}
			}
			if !bytes.Contains(body, []byte(`Go &lt;maps&gt; &amp; &#34;quotes&#34; [en]`)) || !bytes.Contains(body, []byte(`href="/a/"`)) || !bytes.Contains(body, []byte(`name="q" value="" data-search-query`)) {
				t.Errorf("%s %s did not preserve escaped metadata/local navigation/form", language, extension)
			}
			candidates, err := siteSearchCandidates(body)
			if err != nil || !reflect.DeepEqual(candidates, index) {
				t.Fatalf("final %s %s catalogue = %+v, err %v", language, extension, candidates, err)
			}
		}
	}
}

func TestSiteSearchLifecycleAndCreationOrderDeterminism(t *testing.T) {
	stage, deploy, base := siteSearchFixture(t)
	a := siteSearchEntry{URL: "/a/", Title: "First", Summary: "Original", Language: "ca"}
	b := siteSearchEntry{URL: "/b/", Title: "Second", Language: "es"}
	siteSearchPutPage(t, stage, deploy, a, "")
	siteSearchPutPage(t, stage, deploy, b, "")
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{b, a})
	first, err := finalizeSiteSearch(stage, deploy, base)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes := readMarkdownDiscoveryFile(t, deploy, "search/index.html")
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{a, b})
	second, err := finalizeSiteSearch(stage, deploy, base)
	if err != nil || !reflect.DeepEqual(first, second) || !bytes.Equal(firstBytes, readMarkdownDiscoveryFile(t, deploy, "search/index.html")) {
		t.Fatalf("same catalogue depended on generator traversal order: %v", err)
	}
	// Repeated finalization is also byte-identical, including metadata escaping.
	if _, err := finalizeSiteSearch(stage, deploy, base); err != nil || !bytes.Equal(firstBytes, readMarkdownDiscoveryFile(t, deploy, "search/index.html")) {
		t.Fatalf("second finalization changed bytes: %v", err)
	}
	a.Title, a.Summary = "Changed title", "Changed summary"
	c := siteSearchEntry{URL: "/c/", Title: "New page", Language: "en"}
	siteSearchPutPage(t, stage, deploy, c, "")
	if err := os.Remove(filepath.Join(stage, "b/index.md")); err != nil {
		t.Fatal(err)
	}
	// A stale deleted candidate must not survive even if its deploy file does.
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{b, c, a})
	index, err := finalizeSiteSearch(stage, deploy, base)
	if err != nil || !reflect.DeepEqual(index, []siteSearchEntry{a, c}) {
		t.Fatalf("new/edit/delete lifecycle = %+v, %v", index, err)
	}
	put(t, stage, "a/index.md", "<!-- private: true -->\nSECRET_BODY")
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{a, c})
	index, err = finalizeSiteSearch(stage, deploy, base)
	if err != nil || !reflect.DeepEqual(index, []siteSearchEntry{c}) {
		t.Fatalf("public-to-private lifecycle = %+v, %v", index, err)
	}
}

func TestSiteSearchRejectsInvalidCataloguesBeforeAnyArtifactWrite(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string, string)
	}{
		{"missing results marker", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, deploy, "en/search/index.html", siteSearchResultsStart, "")
		}},
		{"duplicate status marker", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, deploy, "es/search/index.md", siteSearchStatusStart, siteSearchStatusStart+siteSearchStatusStart)
		}},
		{"reversed results markers", func(t *testing.T, stage, deploy string) {
			body := readMarkdownDiscoveryFile(t, deploy, "search/index.html")
			body = bytes.ReplaceAll(body, []byte(siteSearchResultsStart), []byte("TMP_SEARCH_MARKER"))
			body = bytes.ReplaceAll(body, []byte(siteSearchResultsEnd), []byte(siteSearchResultsStart))
			body = bytes.ReplaceAll(body, []byte("TMP_SEARCH_MARKER"), []byte(siteSearchResultsEnd))
			put(t, deploy, "search/index.html", string(body))
		}},
		{"duplicate URL", func(t *testing.T, stage, deploy string) {
			entry := siteSearchEntry{URL: "/a/", Title: "A", Language: "ca"}
			siteSearchPutCatalogues(t, deploy, []siteSearchEntry{entry, entry})
		}},
		{"locale inventory disagreement", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, deploy, "es/search/index.html", ">A</a>", ">Different</a>")
		}},
		{"missing twin", func(t *testing.T, stage, deploy string) {
			if err := os.Remove(filepath.Join(deploy, "es/search/index.md")); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing locale", func(t *testing.T, stage, deploy string) {
			if err := os.Remove(filepath.Join(deploy, "en/search/index.html")); err != nil {
				t.Fatal(err)
			}
		}},
		{"canonical mismatch", func(t *testing.T, stage, deploy string) {
			put(t, deploy, "a/index.html", httpWorkerHTML("https://evil.example/a/", "Body"))
		}},
		{"deployed language mismatch", func(t *testing.T, stage, deploy string) {
			put(t, deploy, "a/index.html", strings.Replace(httpWorkerHTML("https://dario.cat/a/", "Body"), `lang="ca"`, `lang="en"`, 1))
		}},
		{"source symlink", func(t *testing.T, stage, deploy string) {
			if err := os.Remove(filepath.Join(stage, "a/index.md")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("never-read", filepath.Join(stage, "a/index.md")); err != nil {
				t.Fatal(err)
			}
		}},
		{"duplicate query input", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, deploy, "es/search/index.md", `name="q" value="" data-search-query`, `name="q" value="" data-search-query name="q" value="" data-search-query`)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stage, deploy, base := siteSearchFixture(t)
			entry := siteSearchEntry{URL: "/a/", Title: "A", Language: "ca"}
			siteSearchPutPage(t, stage, deploy, entry, "")
			siteSearchPutCatalogues(t, deploy, []siteSearchEntry{entry})
			test.mutate(t, stage, deploy)
			before := readMarkdownDiscoveryFile(t, deploy, "search/index.md")
			if _, err := finalizeSiteSearch(stage, deploy, base); err == nil {
				t.Fatal("invalid catalogue accepted")
			}
			if !bytes.Equal(before, readMarkdownDiscoveryFile(t, deploy, "search/index.md")) {
				t.Fatal("invalid catalogue partially rewrote an earlier artifact")
			}
		})
	}
}

func TestSiteSearchURLsAreLocalCanonicalPaths(t *testing.T) {
	for _, raw := range []string{"https://dario.cat/a/", "//evil.example/", "/a?key=secret", "/a#part", "/a/../b/", "/a//b/", "/a%2fb/", "/a%5cb/", "/a\\b/", "/%2e%2e/private/", "/a\n/", "/a%00/", "/a%0a/", "/a%20b/%", ""} {
		if _, err := siteSearchURL(raw); err == nil {
			t.Errorf("accepted unsafe URL %q", raw)
		}
	}
	for _, raw := range []string{"/", "/a/", "/en/posts/a/", "/caf%C3%A8/", "/legacy.html"} {
		if _, err := siteSearchURL(raw); err != nil {
			t.Errorf("rejected canonical URL %q: %v", raw, err)
		}
	}
}

func TestSiteSearchWorkerIndexAndFeatureAbsenceCompatibility(t *testing.T) {
	index := []siteSearchEntry{{URL: "/a/", Title: `Closing </script> is data`, Language: "ca"}}
	worker, err := siteSearchInject([]byte("const SEARCH_INDEX = "+siteSearchMarker+";"), index)
	if err != nil || bytes.Contains(worker, []byte("</script>")) {
		t.Fatalf("unsafe Worker JSON or injection failure: %s, %v", worker, err)
	}
	var decoded []siteSearchEntry
	if err := json.Unmarshal(bytes.TrimSuffix(bytes.TrimPrefix(worker, []byte("const SEARCH_INDEX = ")), []byte(";")), &decoded); err != nil || !reflect.DeepEqual(decoded, index) {
		t.Fatalf("Worker index changed escaped metadata: %+v %v", decoded, err)
	}
	for _, template := range []string{"no marker", siteSearchMarker + siteSearchMarker} {
		if _, err := siteSearchInject([]byte(template), index); err == nil {
			t.Error("missing or duplicate index marker accepted for search")
		}
	}
	if got, err := siteSearchInject([]byte("legacy adapter"), nil); err != nil || string(got) != "legacy adapter" {
		t.Fatalf("search absence changed existing adapter: %q %v", got, err)
	}
	stage, deploy, base := siteSearchFixture(t)
	if got, err := finalizeSiteSearch(stage, deploy, base); err != nil || got != nil {
		t.Fatalf("absence should produce no output: %+v %v", got, err)
	}
}

func TestSiteSearchPrecedesHTTPInventoryHashes(t *testing.T) {
	root, stage, deploy := httpWorkerFixture(t)
	entry := siteSearchEntry{URL: "/a/", Title: "A", Language: "ca"}
	doc := siteSearchEntry{URL: "/mergo/", Title: "Mergo", Language: "en"}
	siteSearchPutPage(t, stage, deploy, entry, "")
	siteSearchPutPage(t, stage, deploy, doc, "publish_markdown: false")
	put(t, deploy, "mergo/index.md", "# Portable Mergo docs\n\nLiteral {{.Pages \"posts\"}} example.\n")
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{entry, doc, {URL: "/CV/MAC.html", Title: "DO_NOT_INDEX", Language: "ca"}})
	template := string(readMarkdownDiscoveryFile(t, root, ".tools/http/worker.mjs")) + "\nconst SEARCH_INDEX = " + siteSearchMarker + ";\n"
	put(t, root, ".tools/http/worker.mjs", template)
	if err := finalizeHTTPWorker(root, stage, deploy, map[string]string{"mergo/index.html": "mergo/index.md"}); err != nil {
		t.Fatal(err)
	}
	pages := readHTTPWorkerInventory(t, deploy)
	for _, prefix := range []string{"", "en/", "es/"} {
		page := pages["/"+prefix+"search/"]
		body := readMarkdownDiscoveryFile(t, deploy, prefix+"search/index.html")
		if page.HTMLSHA256 != httpWorkerDigest(body) || bytes.Contains(body, []byte("DO_NOT_INDEX")) || !bytes.Contains(body, []byte(`href="/mergo/"`)) {
			t.Fatal("HTTP inventory hashed raw candidates rather than final public catalogue")
		}
	}
	if bytes.Contains(readMarkdownDiscoveryFile(t, deploy, "_worker.js"), []byte("DO_NOT_INDEX")) {
		t.Fatal("Worker index leaked private metadata")
	}
}

func siteSearchFixture(t *testing.T) (stage, deploy string, base *url.URL) {
	t.Helper()
	stage = t.TempDir()
	deploy = filepath.Join(stage, ".zas/deploy")
	if err := os.MkdirAll(deploy, 0755); err != nil {
		t.Fatal(err)
	}
	base, _ = url.Parse("https://dario.cat")
	return stage, deploy, base
}

func siteSearchPutPage(t *testing.T, stage, deploy string, entry siteSearchEntry, metadata string) {
	t.Helper()
	file := strings.TrimPrefix(entry.URL, "/")
	if strings.HasSuffix(file, "/") || file == "" {
		file += "index.html"
	}
	put(t, stage, strings.TrimSuffix(file, ".html")+".md", "<!--\nlanguage: "+entry.Language+"\n"+metadata+"\n-->\n# SECRET_BODY — never index the source or rendered content\n")
	put(t, deploy, file, strings.Replace(httpWorkerHTML("https://dario.cat"+entry.URL, "SECRET_BODY"), `lang="ca"`, `lang="`+entry.Language+`"`, 1))
}

func siteSearchPutCatalogues(t *testing.T, deploy string, entries []siteSearchEntry) {
	t.Helper()
	var rows strings.Builder
	rows.WriteString(`<ul class="search-results">`)
	for _, entry := range entries {
		fmt.Fprintf(&rows, `<li data-search-url="%s" data-search-language="%s"><h2><a class="search-result__title" href="%s">%s</a></h2><p class="search-result__summary">%s</p></li>`, stdhtml.EscapeString(entry.URL), entry.Language, stdhtml.EscapeString(entry.URL), stdhtml.EscapeString(entry.Title), stdhtml.EscapeString(entry.Summary))
	}
	rows.WriteString(`</ul>`)
	body := `<form><input name="q" value="" data-search-query></form>` + siteSearchStatusStart + `<p>Browse</p>` + siteSearchStatusEnd + siteSearchResultsStart + rows.String() + siteSearchResultsEnd
	for _, language := range []string{"ca", "en", "es"} {
		prefix := agentContextPrefix(language)
		put(t, deploy, prefix+"search/index.html", strings.Replace(httpWorkerHTML("https://dario.cat/"+prefix+"search/", body), `lang="ca"`, `lang="`+language+`"`, 1))
		put(t, deploy, prefix+"search/index.md", "# Search\n\n"+body+"\n")
	}
}

func TestSiteSearchBasePathAndHTMLSources(t *testing.T) {
	stage, deploy, _ := siteSearchFixture(t)
	base, _ := url.Parse("https://dario.cat/site")
	entry := siteSearchEntry{URL: "/site/legacy.html", Title: "Legacy", Language: "ca"}
	put(t, stage, "legacy.html", "<!doctype html><!-- language: ca -->\n<h1>Legacy</h1>")
	put(t, deploy, "legacy.html", httpWorkerHTML("https://dario.cat/site/legacy.html", "Body"))
	if ok, err := siteSearchPublicEntry(stage, deploy, base, &entry); err != nil || !ok {
		t.Fatalf("base path or HTML source lost: %v %v", ok, err)
	}
	entry.URL = "/outside/"
	if _, err := siteSearchPublicEntry(stage, deploy, base, &entry); err == nil {
		t.Fatal("outside-base candidate accepted")
	}
}

func TestSiteSearchEmptyCatalogueRemainsWorking(t *testing.T) {
	stage, deploy, base := siteSearchFixture(t)
	siteSearchPutCatalogues(t, deploy, nil)
	index, err := finalizeSiteSearch(stage, deploy, base)
	if err != nil || index == nil || len(index) != 0 {
		t.Fatalf("empty catalogue disabled search: %+v %v", index, err)
	}
	for _, language := range []string{"ca", "en", "es"} {
		body := string(readMarkdownDiscoveryFile(t, deploy, agentContextPrefix(language)+"search/index.html"))
		if !strings.Contains(body, `class="search-results"`) || !strings.Contains(body, `role="status">0 `) {
			t.Fatalf("empty %s catalogue missing usable status/list", language)
		}
	}
}

func TestSiteSearchOptionalSummaryMayBeRemovedByHTMLRendering(t *testing.T) {
	stage, deploy, base := siteSearchFixture(t)
	entry := siteSearchEntry{URL: "/a/", Title: "No summary", Language: "ca"}
	siteSearchPutPage(t, stage, deploy, entry, "")
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{entry})
	for _, language := range []string{"ca", "en", "es"} {
		file := agentContextPrefix(language) + "search/index.html"
		replaceAgentContextFile(t, deploy, file, `<p class="search-result__summary"></p>`, "")
	}
	if index, err := finalizeSiteSearch(stage, deploy, base); err != nil || !reflect.DeepEqual(index, []siteSearchEntry{entry}) {
		t.Fatalf("optional missing summary rejected: %+v %v", index, err)
	}
}

func TestSiteSearchUsesOnlyPublicHeadDescriptionAsOptionalFallback(t *testing.T) {
	stage, deploy, base := siteSearchFixture(t)
	entry := siteSearchEntry{URL: "/a/", Title: "A", Language: "ca"}
	siteSearchPutPage(t, stage, deploy, entry, "")
	put(t, deploy, "a/index.html", strings.Replace(httpWorkerHTML("https://dario.cat/a/", `<meta name="description" content="SECRET_BODY">`), "</head>", `<meta name="description" content="Metadata &amp; &lt;escaped&gt;"></head>`, 1))
	siteSearchPutCatalogues(t, deploy, []siteSearchEntry{entry})
	index, err := finalizeSiteSearch(stage, deploy, base)
	if err != nil || len(index) != 1 || index[0].Summary != "Metadata & <escaped>" {
		t.Fatalf("description fallback changed: %+v %v", index, err)
	}
	// A supplied summary wins over the description; no page-body fallback is
	// allowed when neither metadata field exists.
	entry.Summary = "Explicit summary"
	if ok, err := siteSearchPublicEntry(stage, deploy, base, &entry); err != nil || !ok || entry.Summary != "Explicit summary" {
		t.Fatalf("explicit summary was replaced: %+v %v %v", entry, ok, err)
	}
	put(t, deploy, "a/index.html", httpWorkerHTML("https://dario.cat/a/", `<meta name="description" content="SECRET_BODY">`))
	entry.Summary = ""
	if ok, err := siteSearchPublicEntry(stage, deploy, base, &entry); err != nil || !ok || entry.Summary != "" {
		t.Fatalf("page body became metadata: %+v %v %v", entry, ok, err)
	}
}

func TestSiteSearchIndexesOnlyManifestOwnedProjectDocsWithFinalTwins(t *testing.T) {
	stage, deploy, base := siteSearchFixture(t)
	var candidates []siteSearchEntry
	owned := make(map[string]string)
	for _, test := range []struct {
		path     string
		metadata string
		owned    bool
	}{
		{"mergo/options", "", true},
		{"zas/templates", "", true},
		{"mergo/forged", "docs_project: mergo", false},
		{"mergo/unowned", "", false},
		{"mergo/private-page", "private: true", true},
		{"mergo/draft-page", "draft: true", true},
		{"mergo/unpublished", "publish: false", true},
		{"hidden-docs/page", "", true},
	} {
		entry := siteSearchEntry{URL: "/" + test.path + "/", Title: test.path, Language: "en"}
		siteSearchPutPage(t, stage, deploy, entry, "publish_markdown: false\n"+test.metadata)
		candidates = append(candidates, entry)
		if test.owned {
			file := test.path + "/index.html"
			twin := test.path + "/index.md"
			owned[file] = twin
			// These portable compiler docs deliberately preserve literal template
			// examples. Treating them as ordinary Zas twins rejects valid docs.
			put(t, deploy, twin, "# Public documentation\n\nLiteral {{.Pages \"posts\"}} example.\n")
		}
	}
	put(t, stage, "hidden-docs/.zas.yml", "private: true\n")
	put(t, stage, "mergo/.zas.yml", "publish_markdown: false\nlanguage: en\n")
	siteSearchPutCatalogues(t, deploy, candidates)
	index, err := finalizeSiteSearch(stage, deploy, base, owned)
	if err != nil || len(index) != 2 || index[0].URL != "/mergo/options/" || index[1].URL != "/zas/templates/" {
		t.Fatalf("compiler-owned docs selection = %+v, %v", index, err)
	}
	for _, language := range []string{"ca", "en", "es"} {
		for _, extension := range []string{"html", "md"} {
			body := readMarkdownDiscoveryFile(t, deploy, agentContextPrefix(language)+"search/index."+extension)
			if !bytes.Contains(body, []byte("/mergo/options/")) || !bytes.Contains(body, []byte("/zas/templates/")) || bytes.Contains(body, []byte("forged")) || bytes.Contains(body, []byte("private-page")) || bytes.Contains(body, []byte("hidden-docs")) {
				t.Errorf("wrong %s %s docs visibility", language, extension)
			}
		}
	}
	// Search's compiler exception must not relax llms/context publication rules.
	if err := agentContextPublicSource(stage, "mergo/options/index.md", "en"); err == nil {
		t.Fatal("project-docs exception leaked into ordinary agent-context eligibility")
	}
}

func TestSiteSearchManifestOwnershipRequiresSafeFinalMarkdownTwin(t *testing.T) {
	for _, test := range []struct {
		name string
		twin string
		body string
	}{
		{"missing", "mergo/index.md", ""},
		{"HTML fallback", "mergo/index.md", "<!doctype html><html>Fallback</html>"},
		{"outside deploy", "../elsewhere.md", ""},
		{"CV", "CV/private.md", ""},
		{"private", "private/context.md", ""},
		{"wrong extension", "mergo/index.html", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			stage, deploy, base := siteSearchFixture(t)
			entry := siteSearchEntry{URL: "/mergo/", Title: "Mergo", Language: "en"}
			siteSearchPutPage(t, stage, deploy, entry, "publish_markdown: false")
			if test.body != "" {
				put(t, deploy, test.twin, test.body)
			}
			siteSearchPutCatalogues(t, deploy, []siteSearchEntry{entry})
			if _, err := finalizeSiteSearch(stage, deploy, base, map[string]string{"mergo/index.html": test.twin}); err == nil {
				t.Fatal("manifest-owned docs accepted an absent or unsafe finalized twin")
			}
		})
	}
}
