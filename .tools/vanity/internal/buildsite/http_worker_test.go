package buildsite

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"dario.cat/site-tools/internal/vanity"
)

func TestHTTPWorkerBuildsFinalPublicInventoryAndStaticBypasses(t *testing.T) {
	root, stage, deploy := httpWorkerFixture(t)
	body := httpWorkerHTML("https://dario.cat/about/", "<h1>Final HTML after discovery</h1>")
	markdown := "# Rendered About\n\nA substituted value, rather than the source template.\n"
	put(t, deploy, "about/index.html", body)
	put(t, deploy, "about/index.md", markdown)
	put(t, deploy, "contact.html", httpWorkerHTML("https://dario.cat/contact.html", "Legacy contact"))
	put(t, deploy, "contact.md", "# Legacy contact\n")
	put(t, deploy, "contact/index.html", httpWorkerHTML("https://dario.cat/contact/", "Current contact"))
	put(t, deploy, "contact/index.md", "# Current contact\n")
	put(t, deploy, "draft/index.html", httpWorkerHTML("https://dario.cat/draft/", "A live draft has no public twin"))
	put(t, deploy, "CV/private.html", "not HTML: this must never be opened")
	put(t, deploy, "CV/private.md", "{{do not process CV}}")
	put(t, deploy, "private/page.html", "not HTML: this must never be opened")
	put(t, deploy, "docs/site-plan/private.html", "not HTML: this must never be opened")
	put(t, deploy, "docs/HANDOFF-FACTS.html", "not HTML: operations documentation must never be opened")
	put(t, deploy, "docs/DOMAIN-OPERATIONS.html", "not HTML: operations documentation must never be opened")
	put(t, deploy, "static/image.png", "static asset")
	put(t, deploy, "assets/icon.png", "static asset")
	put(t, deploy, ".well-known/security.txt", "security contact")
	put(t, deploy, "feed.xml", "<rss/>")
	put(t, deploy, "en/llms.txt", "# Context\n")
	put(t, deploy, "_headers", "/*\n  Content-Signal: search=yes, ai-input=yes, ai-train=no\n")
	put(t, deploy, "orphan.md", "# Plain Markdown asset\n")
	for _, route := range vanity.Routes() {
		put(t, deploy, route.File(), "head fragment without canonical or body")
	}
	if err := finalizeHTTPWorker(root, stage, deploy); err != nil {
		t.Fatal(err)
	}
	pages := readHTTPWorkerInventory(t, deploy)
	expectedKeys := []string{"/", "/about/", "/contact", "/contact.html", "/contact/", "/draft/"}
	keys := make([]string, 0, len(pages))
	for key := range pages {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	slices.Sort(expectedKeys)
	if !slices.Equal(keys, expectedKeys) {
		t.Fatalf("inventory keys = %q, want %q", keys, expectedKeys)
	}
	page := pages["/about/"]
	if page.Path == nil || *page.Path != "/about/index.md" || page.SHA256 == nil || *page.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(markdown))) || page.HTMLSHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(body))) {
		t.Fatalf("about inventory does not hash the final twins: %+v", page)
	}
	for _, key := range []string{"/", "/draft/"} {
		if pages[key].Path != nil || pages[key].SHA256 != nil || pages[key].HTMLSHA256 == "" {
			t.Fatalf("HTML-only %s invented a twin or lost its digest: %+v", key, pages[key])
		}
	}
	if !reflect.DeepEqual(pages["/contact"], pages["/contact.html"]) || reflect.DeepEqual(pages["/contact"], pages["/contact/"]) {
		t.Fatal("extensionless alias collapsed the two distinct contact pages")
	}
	if _, ok := pages["/about"]; ok {
		t.Fatal("index/bare aliases must remain ASSETS redirects")
	}
	routes := readHTTPWorkerRoutes(t, deploy)
	if routes.Version != 1 || !slices.Equal(routes.Include, []string{"/*"}) || !slices.IsSorted(routes.Exclude) || len(routes.Include)+len(routes.Exclude) > 100 {
		t.Fatalf("unexpected Pages routes: %+v", routes)
	}
	for _, expected := range []string{"/*.md", "/static/*", "/assets/*", "/CV/*", "/private/*", "/docs/site-plan/*", "/docs/HANDOFF-FACTS.html", "/docs/DOMAIN-OPERATIONS.html", "/.well-known/*", "/feed.xml", "/en/llms.txt"} {
		if !slices.Contains(routes.Exclude, expected) {
			t.Errorf("missing static bypass %s", expected)
		}
	}
	for _, route := range vanity.Routes() {
		bare := "/" + strings.Trim(route.Path, "/")
		for _, expected := range []string{bare, bare + "/", bare + "/index.html"} {
			if !slices.Contains(routes.Exclude, expected) {
				t.Errorf("raw import alias %s does not bypass Functions", expected)
			}
		}
		if slices.Contains(routes.Exclude, bare+"/*") {
			t.Errorf("broad %s exclusion would block future nested documentation", bare)
		}
	}
	for _, forbidden := range []string{"/about/index.md", "/orphan.md", "/static/image.png", "/assets/icon.png", "/CV/private.html", "/_headers"} {
		if slices.Contains(routes.Exclude, forbidden) {
			t.Errorf("redundant or reserved route bypass %s", forbidden)
		}
	}
	if got := readMarkdownDiscoveryFile(t, deploy, "about/index.html"); string(got) != body {
		t.Fatal("HTTP assembly changed HTML bytes")
	}
	if got := readMarkdownDiscoveryFile(t, deploy, "about/index.md"); string(got) != markdown {
		t.Fatal("HTTP assembly changed Markdown bytes")
	}
}

func TestHTTPWorkerOutputIsDeterministicAcrossCreationOrder(t *testing.T) {
	var outputs [2][2][]byte
	for run := range outputs {
		root, stage, deploy := httpWorkerFixture(t)
		files := []string{"z/index.html", "a/index.html", "nested/deep/page.html"}
		if run == 1 {
			slices.Reverse(files)
		}
		for _, file := range files {
			canonical := "https://dario.cat/" + file
			if strings.HasSuffix(file, "/index.html") {
				canonical = strings.TrimSuffix(canonical, "index.html")
			}
			put(t, deploy, file, httpWorkerHTML(canonical, "Body"))
			put(t, deploy, strings.TrimSuffix(file, ".html")+".md", "# Rendered Markdown\n")
		}
		if err := finalizeHTTPWorker(root, stage, deploy); err != nil {
			t.Fatal(err)
		}
		outputs[run][0] = readMarkdownDiscoveryFile(t, deploy, "_worker.js")
		outputs[run][1] = readMarkdownDiscoveryFile(t, deploy, "_routes.json")
	}
	for index := range outputs[0] {
		if !bytes.Equal(outputs[0][index], outputs[1][index]) {
			t.Fatalf("output %d depends on filesystem creation order", index)
		}
	}
}

func TestHTTPWorkerDisabledPreservesLegacyRequirements(t *testing.T) {
	for _, setting := range []string{"", "  markdown_negotiation: false\n"} {
		stage := t.TempDir()
		put(t, stage, ".zas/config.yml", "site:\n  title: Legacy\n"+setting)
		if err := finalizeHTTPWorker(filepath.Join(stage, "missing-root"), stage, filepath.Join(stage, "missing-deploy")); err != nil {
			t.Fatalf("disabled adapter changed legacy build: %v", err)
		}
	}
}

func TestHTTPWorkerRequiresPublishedMarkdown(t *testing.T) {
	root, stage, deploy := httpWorkerFixture(t)
	put(t, stage, ".zas/config.yml", "site:\n  markdown_negotiation: true\n")
	if err := finalizeHTTPWorker(root, stage, deploy); err == nil || !strings.Contains(err.Error(), "site.publish_markdown") {
		t.Fatalf("missing publish_markdown error = %v", err)
	}
	assertNoHTTPWorkerOutputs(t, deploy)
}

func TestHTTPWorkerRejectsInvalidArtifactsBeforeOutput(t *testing.T) {
	cases := map[string]func(*testing.T, string, string, string){
		"unrendered twin": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.md", "# {{.Title}}\n")
		},
		"HTML fallback twin": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.md", "<!DOCTYPE html><html>Fallback</html>")
		},
		"empty twin": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.md", " \n")
		},
		"twin directory": func(t *testing.T, root, stage, deploy string) {
			if err := os.Mkdir(filepath.Join(deploy, "page.md"), 0755); err != nil {
				t.Fatal(err)
			}
		},
		"external canonical": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.html", httpWorkerHTML("https://other.example/page.html", "Body"))
		},
		"different canonical": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.html", httpWorkerHTML("https://dario.cat/other.html", "Body"))
		},
		"query canonical": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.html", httpWorkerHTML("https://dario.cat/page.html?secret=1", "Body"))
		},
		"missing canonical": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "page.html", "<html><head></head><body>Body</body></html>")
		},
		"missing marker": func(t *testing.T, root, stage, deploy string) {
			put(t, root, ".tools/http/worker.mjs", "const PAGE_ROUTES = {};\n")
		},
		"duplicate marker": func(t *testing.T, root, stage, deploy string) {
			put(t, root, ".tools/http/worker.mjs", httpWorkerMarker+httpWorkerMarker)
		},
		"template directory": func(t *testing.T, root, stage, deploy string) {
			if err := os.Remove(filepath.Join(root, ".tools/http/worker.mjs")); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(root, ".tools/http/worker.mjs"), 0755); err != nil {
				t.Fatal(err)
			}
		},
		"unsafe path": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "unsafe*.txt", "Asset")
		},
		"invalid base URL": func(t *testing.T, root, stage, deploy string) {
			put(t, stage, ".zas/config.yml", "site:\n  markdown_negotiation: true\n  publish_markdown: true\n  baseurl: https://dario.cat/../\n")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			root, stage, deploy := httpWorkerFixture(t)
			put(t, deploy, "page.html", httpWorkerHTML("https://dario.cat/page.html", "Body"))
			mutate(t, root, stage, deploy)
			before := readMarkdownDiscoveryFile(t, deploy, "index.html")
			if err := finalizeHTTPWorker(root, stage, deploy); err == nil {
				t.Fatal("invalid artifact accepted")
			}
			assertNoHTTPWorkerOutputs(t, deploy)
			if !bytes.Equal(before, readMarkdownDiscoveryFile(t, deploy, "index.html")) {
				t.Fatal("failed assembly changed a deployed page")
			}
		})
	}
}

func TestHTTPWorkerRejectsSymlinksWithoutReadingTargets(t *testing.T) {
	for _, relative := range []string{".tools/http/worker.mjs", ".tools/http", "page.md", "page.html", "CV"} {
		t.Run(relative, func(t *testing.T) {
			root, stage, deploy := httpWorkerFixture(t)
			put(t, deploy, "page.html", httpWorkerHTML("https://dario.cat/page.html", "Body"))
			directory := deploy
			if strings.HasPrefix(relative, ".tools/") {
				directory = root
			}
			file := filepath.Join(directory, filepath.FromSlash(relative))
			if err := os.RemoveAll(file); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(t.TempDir(), "never-open-this-missing-target"), file); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if err := finalizeHTTPWorker(root, stage, deploy); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("symlink error = %v", err)
			}
			assertNoHTTPWorkerOutputs(t, deploy)
		})
	}
}

func TestHTTPWorkerRejectsReservedOutputCollisions(t *testing.T) {
	for _, name := range []string{"_worker.js", "_routes.json"} {
		for _, inRoot := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/root=%t", name, inRoot), func(t *testing.T) {
				root, stage, deploy := httpWorkerFixture(t)
				directory := deploy
				if inRoot {
					directory = root
				}
				put(t, directory, name, "Existing ownership sentinel\n")
				if err := finalizeHTTPWorker(root, stage, deploy); err == nil || !strings.Contains(err.Error(), "refuses existing") {
					t.Fatalf("collision error = %v", err)
				}
				if string(readMarkdownDiscoveryFile(t, directory, name)) != "Existing ownership sentinel\n" {
					t.Fatal("collision overwrote existing artifact")
				}
				other := "_worker.js"
				if name == other {
					other = "_routes.json"
				}
				if _, err := os.Lstat(filepath.Join(deploy, other)); !os.IsNotExist(err) {
					t.Fatalf("collision generated other artifact: %v", err)
				}
			})
		}
	}
}

func TestHTTPWorkerRouteBudgetAndLengthFailBeforeOutput(t *testing.T) {
	for _, count := range []int{1, 101} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root, stage, deploy := httpWorkerFixture(t)
			for index := range count {
				name := fmt.Sprintf("file-%03d.txt", index)
				if count == 1 {
					name = strings.Repeat("x", 100) + ".txt"
				}
				put(t, deploy, name, "Uncovered static asset\n")
			}
			if err := finalizeHTTPWorker(root, stage, deploy); err == nil || !strings.Contains(err.Error(), "100") {
				t.Fatalf("Pages routing limit error = %v", err)
			}
			assertNoHTTPWorkerOutputs(t, deploy)
		})
	}
}

func TestHTTPWorkerRejectsUnsupportedPathPunctuation(t *testing.T) {
	for _, punctuation := range "!'()[]+$^{}|*?%#\\" {
		for _, inBase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%c/base=%t", punctuation, inBase), func(t *testing.T) {
				root, stage, deploy := httpWorkerFixture(t)
				if inBase {
					put(t, stage, ".zas/config.yml", fmt.Sprintf("site:\n  markdown_negotiation: true\n  publish_markdown: true\n  baseurl: %q\n", "https://dario.cat/unsafe"+string(punctuation)+"path"))
				} else {
					put(t, deploy, "unsafe"+string(punctuation)+"path.txt", "Uncovered static asset\n")
				}
				if err := finalizeHTTPWorker(root, stage, deploy); err == nil || !strings.Contains(err.Error(), "HTTP worker") {
					t.Fatalf("unsupported punctuation was accepted: %v", err)
				}
				assertNoHTTPWorkerOutputs(t, deploy)
			})
		}
	}
}

func TestHTTPWorkerSupportsCanonicalBasePath(t *testing.T) {
	deploy := t.TempDir()
	put(t, deploy, "about/index.html", httpWorkerHTML("https://example.com/blog/about/", "Body"))
	put(t, deploy, "about/index.md", "# About\n")
	base, _ := url.Parse("https://example.com/blog/")
	pages, routes, err := collectHTTPWorkerPages(deploy, base)
	if err != nil {
		t.Fatal(err)
	}
	page := pages["/blog/about/"]
	if page.Path == nil || *page.Path != "/blog/about/index.md" || !slices.Contains(routes.Exclude, "/blog/*.md") {
		t.Fatalf("base path not retained: pages=%+v routes=%+v", pages, routes)
	}
}

func TestHTTPWorkerUsesEncodedURLPathnames(t *testing.T) {
	deploy := t.TempDir()
	put(t, deploy, "català & notes/index.html", httpWorkerHTML("https://dario.cat/catal%C3%A0%20&%20notes/", "Body"))
	put(t, deploy, "català & notes/index.md", "# Notes\n")
	put(t, deploy, "public català.txt", "Public static asset\n")
	base, _ := url.Parse("https://dario.cat")
	pages, routes, err := collectHTTPWorkerPages(deploy, base)
	if err != nil {
		t.Fatal(err)
	}
	page := pages["/catal%C3%A0%20&%20notes/"]
	if page.Path == nil || *page.Path != "/catal%C3%A0%20&%20notes/index.md" || !slices.Contains(routes.Exclude, "/public%20catal%C3%A0.txt") {
		t.Fatalf("inventory does not match URL.pathname encoding: pages=%+v routes=%+v", pages, routes)
	}
}

func TestHTTPWorkerAliasCollisionFailsBeforeOutput(t *testing.T) {
	root, stage, deploy := httpWorkerFixture(t)
	put(t, deploy, "name.html", httpWorkerHTML("https://dario.cat/name.html", "First body"))
	put(t, deploy, "name.html.html", httpWorkerHTML("https://dario.cat/name.html.html", "Second body"))
	if err := finalizeHTTPWorker(root, stage, deploy); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("duplicate route was accepted: %v", err)
	}
	assertNoHTTPWorkerOutputs(t, deploy)
}

func httpWorkerFixture(t *testing.T) (root, stage, deploy string) {
	t.Helper()
	root, stage = t.TempDir(), t.TempDir()
	deploy = filepath.Join(stage, ".zas", "deploy")
	put(t, stage, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\n  markdown_negotiation: true\n")
	template, err := os.ReadFile(filepath.Join("testdata", "http-worker", "worker.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".tools/http/worker.mjs", string(template))
	put(t, deploy, "index.html", httpWorkerHTML("https://dario.cat/", "Root has no twin"))
	return root, stage, deploy
}

func httpWorkerHTML(canonical, body string) string {
	return `<html lang="ca"><head><link rel="canonical" href="` + canonical + `"></head><body>` + body + `</body></html>`
}

func readHTTPWorkerInventory(t *testing.T, deploy string) map[string]httpWorkerPage {
	t.Helper()
	worker := readMarkdownDiscoveryFile(t, deploy, "_worker.js")
	declaration := []byte("const PAGE_ROUTES = ")
	start := bytes.Index(worker, declaration)
	if start < 0 {
		t.Fatal("missing worker inventory")
	}
	var pages map[string]httpWorkerPage
	decoder := json.NewDecoder(bytes.NewReader(worker[start+len(declaration):]))
	if err := decoder.Decode(&pages); err != nil {
		t.Fatal(err)
	}
	return pages
}

func readHTTPWorkerRoutes(t *testing.T, deploy string) httpWorkerRoutes {
	t.Helper()
	var routes httpWorkerRoutes
	if err := json.Unmarshal(readMarkdownDiscoveryFile(t, deploy, "_routes.json"), &routes); err != nil {
		t.Fatal(err)
	}
	return routes
}

func assertNoHTTPWorkerOutputs(t *testing.T, deploy string) {
	t.Helper()
	for _, name := range []string{"_worker.js", "_routes.json"} {
		if _, err := os.Lstat(filepath.Join(deploy, name)); !os.IsNotExist(err) {
			t.Fatalf("failed validation emitted %s: %v", name, err)
		}
	}
}
