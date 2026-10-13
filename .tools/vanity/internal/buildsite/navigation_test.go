package buildsite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dario.cat/site-tools/internal/vanity"
	"golang.org/x/net/html"
)

func TestNavigationInternalOriginsAndEscapedDestinations(t *testing.T) {
	for _, tc := range []struct{ original, target string }{
		{"https://dario.cat/posts/", "/posts/"},
		{"HTTP://DARIO.CAT:80/posts/?x=1#part", "/posts/?x=1#part"},
		{"https://dario.cat:443/", "/"},
		{"//dario.cat/posts/", "/posts/"},
		{"//DARIO.CAT:443/posts/", "/posts/"},
		{"https://dario.cat", "/"},
		{"https://dario.cat/a%2Fb/%C3%A9/?x=a%26b&next=%2Fposts%2F#part%20two", "/a%2Fb/%C3%A9/?x=a%26b&next=%2Fposts%2F#part%20two"},
		{"https://dario.cat/path?#", "/path?#"},
	} {
		t.Run(tc.original, func(t *testing.T) {
			got, internal, err := navigationTarget(tc.original)
			if err != nil || !internal || got != tc.target {
				t.Fatalf("got %q internal=%v err=%v; want %q", got, internal, err, tc.target)
			}
			preview, _ := url.Parse("https://preview.example/article/")
			link, err := url.Parse(got)
			if err != nil || preview.ResolveReference(link).Host != preview.Host {
				t.Fatal("normalized navigation escaped the preview origin")
			}
		})
	}
	for _, original := range []string{
		"https://other.example/posts/", "https://www.dario.cat/posts/", "https://go.dario.cat/posts/",
		"https://dario.cat.evil.example/posts/", "https://dario.cat:444/posts/", "http://dario.cat:443/posts/",
		"https://dario.cat:80/posts/", "//dario.cat:80/posts/", "https://dario.cat@other.example/posts/", "https://dario.cat./posts/",
		"mailto:cc@dario.cat", "at://did:plc:example/site.standard.document/key", "/posts/?x=1#part", "../posts/", "#part",
	} {
		t.Run("untouched "+original, func(t *testing.T) {
			if _, internal, err := navigationTarget(original); err != nil || internal {
				t.Fatalf("external/root/relative link was selected: internal=%v err=%v", internal, err)
			}
		})
	}
	for _, original := range []string{
		"https://dario.cat//other.example/path", "https://dario.cat/%2Fother.example/path", "https://dario.cat/%5Cother.example/path",
		"https://dario.cat/%00", "https://dario.cat/%zz",
		"https://user@dario.cat/posts/", "https://dario.cat:invalid/posts/", `https://dario.cat\posts/`, `https:\\dario.cat\posts/`, `//dario.cat\posts/`,
	} {
		t.Run("unsafe "+original, func(t *testing.T) {
			if _, _, err := navigationTarget(original); err == nil {
				t.Fatal("unsafe internal path accepted")
			}
		})
	}
}

func TestNavigationPatchesOnlyRealBodyAttributes(t *testing.T) {
	input := `<!DOCTYPE html>
<html><head><link rel="canonical" href="https://dario.cat/posts/">
<meta property="og:url" content="https://dario.cat/posts/">
<script type="application/ld+json">{"url":"https://dario.cat/","href":"https://dario.cat/"}</script>
<a href="https://dario.cat/head-only/">Head bytes</a></head><body>
<!-- <a href="https://dario.cat/comment/">Comment</a> -->
<script>const example = '<a href="https://dario.cat/script/">Script</a>';</script>
<a title="href='https://dario.cat/title/' class='u-uid'" href="https://dario.cat/a%2Fb/?a=1&amp;b=%26#part%20two">Real link</a>
<a class='p-name u-url u-uid' href='http://dario.cat:80/posts/?x=1&amp;y=2#'>Title &amp; text</a>
<a href="/already-root/?x=1&amp;y=2">Already root</a>
<a href="https://go.dario.cat/">Subdomain</a><a href="https://elsewhere.example/">External</a>
<a class="u-url u-uid" href="https://dario.cat/" hidden>Hidden identity</a>
<div hidden><a class="u-url u-uid" href="https://dario.cat/">Inherited hidden identity</a></div>
<div hidden><a href="https://dario.cat/hidden-navigation/">Hidden navigation</a></div>
<a class="visually-hidden" href="https://dario.cat/screen-reader/">Accessible navigation</a>
<!--email_off--><a class="u-email" href="mailto:cc@dario.cat">cc@dario.cat</a><!--/email_off-->
<template><a href="https://dario.cat/template/">Template bytes</a></template>
<svg><a href="https://dario.cat/svg/"><text>SVG bytes</text></a></svg>
</body></html>`
	want := strings.Replace(input, `href="https://dario.cat/a%2Fb/?a=1&amp;b=%26#part%20two"`, `href="/a%2Fb/?a=1&amp;b=%26#part%20two"`, 1)
	want = strings.Replace(want, `<a class='p-name u-url u-uid' href='http://dario.cat:80/posts/?x=1&amp;y=2#'>`, `<a class='p-name u-url' href='/posts/?x=1&amp;y=2#'><data class="u-uid" value="http://dario.cat:80/posts/?x=1&amp;y=2#" hidden></data>`, 1)
	want = strings.Replace(want, `href="https://dario.cat/svg/"`, `href="/svg/"`, 1)
	want = strings.Replace(want, `href="https://dario.cat/hidden-navigation/"`, `href="/hidden-navigation/"`, 1)
	want = strings.Replace(want, `href="https://dario.cat/screen-reader/"`, `href="/screen-reader/"`, 1)
	got, err := navigationHTML([]byte(input))
	if err != nil || string(got) != want {
		t.Fatalf("unexpected byte changes: %v\ngot:\n%s\nwant:\n%s", err, got, want)
	}
	second, err := navigationHTML(got)
	if err != nil || !bytes.Equal(second, got) {
		t.Fatalf("navigation is not idempotent: %v", err)
	}
}

func TestNavigationRootUIDKeepsCanonicalIdentityOnPreview(t *testing.T) {
	input := `<html><head></head><body><article class="h-entry"><a class="p-name u-url u-uid" href="/posts/example/?x=1&amp;y=2#part">Example article</a></article></body></html>`
	body, err := navigationHTML([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var link, identity *html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			link = n
		}
		if n.Type == html.ElementNode && navigationTestAttr(n, "class") == "u-uid" {
			if identity != nil {
				t.Fatal("duplicate canonical UID")
			}
			identity = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	if link == nil || identity == nil || identity.Data != "data" || identity.Parent != link || navigationTestAttr(link, "class") != "p-name u-url" {
		t.Fatal("canonical identity is not separate from the navigation URL")
	}
	for _, origin := range []string{"https://preview.example/", "https://dario.cat/"} {
		base, _ := url.Parse(origin)
		href, _ := url.Parse(navigationTestAttr(link, "href"))
		if base.ResolveReference(href).Host != base.Host {
			t.Fatalf("link left %s", origin)
		}
		if navigationTestAttr(identity, "value") != "https://dario.cat/posts/example/?x=1&y=2#part" {
			t.Fatal("canonical UID changed with the viewing origin")
		}
	}
	if !strings.Contains(string(body), `href="/posts/example/?x=1&amp;y=2#part"`) || link.LastChild.Type != html.TextNode || link.LastChild.Data != "Example article" {
		t.Fatal("root href bytes or article name changed")
	}
}

func navigationTestAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func TestNavigationUnquotedAndUppercaseAttributes(t *testing.T) {
	input := `<html><body><A TITLE="literal HREF='https://dario.cat/fake/'" CLASS=u-uid HREF=https://dario.cat/posts/>Name</A></body></html>`
	want := `<html><body><A TITLE="literal HREF='https://dario.cat/fake/'" CLASS="" HREF="/posts/"><data class="u-uid" value="https://dario.cat/posts/" hidden></data>Name</A></body></html>`
	got, err := navigationHTML([]byte(input))
	if err != nil || string(got) != want {
		t.Fatalf("attribute scan changed lookalike text: %v\n%s", err, got)
	}
	duplicateTitle := `<html><body><a title="first" TITLE="second" href="https://dario.cat/">Same harmless text attributes</a></body></html>`
	got, err = navigationHTML([]byte(duplicateTitle))
	if err != nil || string(got) != strings.Replace(duplicateTitle, `href="https://dario.cat/"`, `href="/"`, 1) {
		t.Fatalf("nonidentity attribute triggered a generic validation policy: %v", err)
	}
	for _, anchor := range []string{
		`<a href="https://dario.cat/" HREF="https://other.example/">Duplicate</a>`,
		`<a class="u-url" CLASS="u-uid" href="https://dario.cat/">Duplicate</a>`,
		`<a class="u-uid" href="/%2Fother.example/">Unsafe root identity</a>`,
	} {
		if _, err := navigationHTML([]byte("<html><body>" + anchor + "</body></html>")); err == nil {
			t.Fatalf("ambiguous/unsafe anchor accepted: %s", anchor)
		}
	}
	if _, err := navigationHTML([]byte(`<html><body><svg><a class="u-uid" href="https://dario.cat/">Foreign UID</a></svg></body></html>`)); err == nil {
		t.Fatal("foreign-namespace identity transfer accepted without a defined data-element contract")
	}
}

func TestNavigationAbsoluteUIDRetainsOriginalURI(t *testing.T) {
	for _, tc := range []struct{ original, identity string }{
		{"http://dario.cat:80/posts/?x=1&amp;y=2", "http://dario.cat:80/posts/?x=1&y=2"},
		{"https://dario.cat:443/posts/", "https://dario.cat:443/posts/"},
		{"//dario.cat:443/posts/", "https://dario.cat:443/posts/"},
	} {
		input := `<html><body><a class="u-uid" href="` + tc.original + `">Identity</a></body></html>`
		body, err := navigationHTML([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := html.Parse(bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		var identity string
		var visit func(*html.Node)
		visit = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "data" {
				identity = navigationTestAttr(n, "value")
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				visit(c)
			}
		}
		visit(doc)
		if identity != tc.identity {
			t.Fatalf("absolute canonical UID changed: got %q, want %q", identity, tc.identity)
		}
	}
}

func TestNavigationInventoryPreservesRawVanityAndNonHTML(t *testing.T) {
	deploy := t.TempDir()
	original := `<html><head></head><body><a href="https://dario.cat/posts/">Archive</a></body></html>`
	put(t, deploy, "index.html", original)
	for _, route := range vanity.Routes() {
		put(t, deploy, route.File(), original)
	}
	for _, name := range []string{"feed.xml", "index.md", "record.json", "llms-full.txt"} {
		put(t, deploy, name, original)
	}
	if err := os.Chmod(filepath.Join(deploy, "index.html"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := finalizeNavigation(deploy, func(name string) bool { return name == "mergo/index.html" }); err != nil {
		t.Fatal(err)
	}
	for _, route := range vanity.Routes() {
		body, _ := os.ReadFile(filepath.Join(deploy, route.File()))
		if route.File() == "mergo/index.html" {
			if string(body) == original || !strings.Contains(string(body), `href="/posts/"`) {
				t.Fatal("documented vanity landing did not normalize")
			}
		} else if string(body) != original {
			t.Fatalf("raw vanity bytes changed at %s", route.File())
		}
	}
	for _, name := range []string{"feed.xml", "index.md", "record.json", "llms-full.txt"} {
		body, _ := os.ReadFile(filepath.Join(deploy, name))
		if string(body) != original {
			t.Fatalf("non-HTML artifact changed: %s", name)
		}
	}
	info, _ := os.Stat(filepath.Join(deploy, "index.html"))
	if info.Mode().Perm() != 0600 {
		t.Fatal("normalization changed file permissions")
	}
	before := httpWorkerDeploySnapshot(t, deploy)
	if err := finalizeNavigation(deploy, func(name string) bool { return name == "mergo/index.html" }); err != nil {
		t.Fatal(err)
	}
	after := httpWorkerDeploySnapshot(t, deploy)
	for name, body := range before {
		if after[name] != body {
			t.Fatalf("repeated normalization changed %s", name)
		}
	}
}

func TestNavigationValidatesCompleteInventoryBeforeWriting(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory.html")
	put(t, filepath.Dir(file), filepath.Base(file), "regular file")
	if err := finalizeNavigation(file, nil); err == nil {
		t.Fatal("deploy root accepted as a regular file")
	}
	for _, mode := range []string{"late ambiguous anchor", "late symlink"} {
		t.Run(mode, func(t *testing.T) {
			deploy := t.TempDir()
			original := `<html><body><a href="https://dario.cat/posts/">Archive</a></body></html>`
			put(t, deploy, "a.html", original)
			if mode == "late ambiguous anchor" {
				put(t, deploy, "z.html", `<html><body><a href="https://dario.cat/" href="https://other.example/">Ambiguous</a></body></html>`)
			} else {
				outside := filepath.Join(t.TempDir(), "private.json")
				put(t, filepath.Dir(outside), filepath.Base(outside), "outside bytes")
				if err := os.Symlink(outside, filepath.Join(deploy, "z.json")); err != nil {
					t.Fatal(err)
				}
			}
			if err := finalizeNavigation(deploy, nil); err == nil {
				t.Fatal("invalid inventory accepted")
			}
			body, _ := os.ReadFile(filepath.Join(deploy, "a.html"))
			if string(body) != original {
				t.Fatal("an earlier page was written before complete validation")
			}
		})
	}
}

func TestNavigationBuildFailurePreservesPreviousDeployment(t *testing.T) {
	root := sourceTree(t)
	zas := generator(t, ordinaryOutput+`
cat > .zas/deploy/index.html <<'HTML'
<html><head></head><body><a href="https://dario.cat//other.example/">Unsafe</a></body></html>
HTML
`)
	if err := Build(context.Background(), Options{Root: root, Zas: zas}); err == nil || !strings.Contains(err.Error(), "assembled navigation rejected") {
		t.Fatalf("expected staged navigation rejection: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, ".zas", "deploy", "previous.txt"))
	if err != nil || string(body) != "last successful deployment" {
		t.Fatal("invalid navigation damaged the previous deployment")
	}
}

func TestNavigationRunsBeforeHTTPInventoryHashes(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\n  markdown_negotiation: true\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	worker, err := os.ReadFile(filepath.Join("testdata", "http-worker", "worker.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".tools/http/worker.mjs", string(worker))
	zas := generator(t, ordinaryOutput+`
cat > .zas/deploy/index.html <<'HTML'
<html lang="ca"><head><link rel="canonical" href="https://dario.cat/"></head><body><a href="https://dario.cat/posts/">Archive</a></body></html>
HTML
printf '# Rendered Markdown\n' > .zas/deploy/index.md
`)
	if err := Build(context.Background(), Options{Root: root, Zas: zas}); err != nil {
		t.Fatal(err)
	}
	deploy := filepath.Join(root, ".zas", "deploy")
	body, _ := os.ReadFile(filepath.Join(deploy, "index.html"))
	assembled, _ := os.ReadFile(filepath.Join(deploy, "_worker.js"))
	if !bytes.Contains(body, []byte(`href="/posts/"`)) || !bytes.Contains(assembled, []byte(fmt.Sprintf(`"htmlSha256":"%x"`, sha256.Sum256(body)))) {
		t.Fatal("HTTP inventory did not hash the normalized final HTML")
	}
}
