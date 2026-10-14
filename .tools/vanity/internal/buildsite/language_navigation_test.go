package buildsite

import (
	"bytes"
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const languageMenuFixture = `<nav class="site-language-nav" aria-label="Languages"><a href="/">Català</a><a href="/en/">English</a></nav>`

func languageFixture(t *testing.T) (string, string) {
	t.Helper()
	root, deploy := sitemapFixture(t)
	put(t, root, ".zas/translations.yml", "version: 1\ngroups:\n  - ca: /posts/original/\n    en: /en/posts/translated/\n    es: /es/posts/traducido/\n")
	for path, language := range map[string]string{
		"/": "ca", "/en/": "en", "/es/guide/": "es", "/en/guide/": "en",
		"/posts/": "ca", "/en/posts/": "en", "/es/posts/": "es",
		"/posts/ca/": "ca", "/posts/original/": "ca", "/en/posts/translated/": "en",
		"/es/posts/traducido/": "es", "/cv/": "en", "/es/posts/standalone/": "es",
	} {
		put(t, deploy, strings.TrimPrefix(path, "/")+"index.html", `<html lang="`+language+`"><head><link rel="canonical" href="https://dario.cat`+path+`"></head><body><header>`+languageMenuFixture+`</header><main>unchanged</main></body></html>`)
	}
	// Legacy extension URLs also retain their self link, not an invented /en/
	// sibling. Every ordinary page gets a menu; raw vanity fragments do not.
	for _, name := range []string{"contact.html", "myindex.html"} {
		put(t, deploy, name, `<html lang="ca"><head><link rel="canonical" href="https://dario.cat/`+name+`"></head><body><header>`+languageMenuFixture+`</header></body></html>`)
	}
	var sitemap siteSitemap
	for _, path := range []string{"/", "/en/", "/posts/", "/en/posts/", "/es/posts/", "/posts/original/", "/en/posts/translated/", "/es/posts/traducido/"} {
		sitemap.URLs = append(sitemap.URLs, siteSitemapURL{Location: "https://dario.cat" + path, Lastmod: "2024-01-02"})
	}
	sitemap.XMLNS = sitemapNamespace
	output, err := xml.Marshal(sitemap)
	if err != nil {
		t.Fatal(err)
	}
	put(t, deploy, "sitemap.xml", string(output))
	return root, deploy
}

func TestLanguageNavigationLinksToPublishedCounterparts(t *testing.T) {
	root, deploy := languageFixture(t)
	raw, _ := os.ReadFile(filepath.Join(deploy, "xdr/index.html"))
	if err := finalizeLanguageNavigation(root, deploy); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		file, current, catalan, english, spanish string
	}{
		{"en/posts/index.html", "en", "/posts/", "/en/posts/", "/es/posts/"},
		{"en/posts/translated/index.html", "en", "/posts/original/", "/en/posts/translated/", "/es/posts/traducido/"},
		{"posts/original/index.html", "ca", "/posts/original/", "/en/posts/translated/", "/es/posts/traducido/"},
		{"es/posts/traducido/index.html", "es", "/posts/original/", "/en/posts/translated/", "/es/posts/traducido/"},
		{"en/guide/index.html", "en", "/", "/en/guide/", "/es/guide/"},
		{"posts/ca/index.html", "ca", "/posts/ca/", "/en/", ""},
		{"cv/index.html", "en", "/", "/cv/", ""},
		{"es/posts/standalone/index.html", "es", "/", "/en/", "/es/posts/standalone/"},
		{"contact.html", "ca", "/contact.html", "/en/", ""},
	} {
		t.Run(test.file, func(t *testing.T) {
			body, _ := os.ReadFile(filepath.Join(deploy, test.file))
			for language, target := range map[string]string{"ca": test.catalan, "en": test.english, "es": test.spanish} {
				if target == "" {
					if bytes.Contains(body, []byte(`hreflang="`+language+`"`)) {
						t.Fatal("invented language destination")
					}
					continue
				}
				want := `href="` + target + `" lang="` + language + `" hreflang="` + language + `"`
				label := map[string]string{"ca": "Català", "en": "English", "es": "Castellano"}[language]
				want += ` aria-label="` + label + `"`
				if language == test.current {
					want += ` aria-current="page"`
				}
				want += `>` + strings.ToUpper(language) + `</a>`
				if !bytes.Contains(body, []byte(want)) {
					t.Fatalf("missing %s in %s", want, body)
				}
			}
			if bytes.Count(body, []byte(`aria-current="page"`)) != 1 {
				t.Fatal("current language not unique")
			}
		})
	}
	after, _ := os.ReadFile(filepath.Join(deploy, "xdr/index.html"))
	if !bytes.Equal(raw, after) {
		t.Fatal("raw vanity bytes changed")
	}
	data, _ := os.ReadFile(filepath.Join(deploy, "sitemap.xml"))
	var sitemap siteSitemap
	if err := readNativeSitemap(data, &sitemap); err != nil {
		t.Fatal(err)
	}
	for _, entry := range sitemap.URLs {
		want := 3
		if entry.Location == "https://dario.cat/" || entry.Location == "https://dario.cat/en/" {
			want = 2
		}
		if len(entry.Links) != want || entry.Lastmod != "2024-01-02" {
			t.Fatalf("sitemap alternates or original lastmod lost: %+v", entry)
		}
		for _, link := range entry.Links {
			if link.Rel != "alternate" || !strings.HasPrefix(link.URL, "https://dario.cat/") {
				t.Fatalf("invalid canonical alternate: %+v", link)
			}
		}
	}
	if err := finalizeLanguageNavigation(root, deploy); err != nil {
		t.Fatal(err)
	}
	repeated, _ := os.ReadFile(filepath.Join(deploy, "sitemap.xml"))
	if !bytes.Equal(data, repeated) {
		t.Fatal("sitemap rewrite is not deterministic")
	}
}

func TestLanguageNavigationRejectsInvalidManifestBeforeRewriting(t *testing.T) {
	for _, manifest := range []string{
		"version: 2\n", "version: 1\nunknown: true\n", "version: 1\n---\nversion: 1\n",
		"version: 1\ngroups:\n  - ca: /posts/original/\n",
		"version: 1\ngroups:\n  - ca: /missing/\n    en: /en/\n",
		"version: 1\ngroups:\n  - ca: https://dario.cat/\n    en: /en/\n",
		"version: 1\ngroups:\n  - ca: /cv/\n    en: /en/\n",
		"version: 1\ngroups:\n  - ca: /posts/\n    en: /en/posts/translated/\n",
		"version: 1\ngroups:\n  - ca: /posts/original/\n    en: /en/posts/translated/\n  - ca: /posts/original/\n    es: /es/posts/traducido/\n",
	} {
		t.Run(manifest, func(t *testing.T) {
			root, deploy := languageFixture(t)
			before, _ := os.ReadFile(filepath.Join(deploy, "index.html"))
			put(t, root, ".zas/translations.yml", manifest)
			if err := finalizeLanguageNavigation(root, deploy); err == nil {
				t.Fatal("invalid manifest accepted")
			}
			after, _ := os.ReadFile(filepath.Join(deploy, "index.html"))
			if !bytes.Equal(before, after) {
				t.Fatal("failed validation rewrote deployed HTML")
			}
		})
	}
}

func TestLanguageNavigationChangesOnlyTheRealHeaderMenu(t *testing.T) {
	before := `<html lang="en"><head><script>"` + languageMenuFixture + `"</script></head><body><header><!-- ` + languageMenuFixture + ` --><template>` + languageMenuFixture + `</template>` + languageMenuFixture + `</header><main>` + languageMenuFixture + `</main></body></html>`
	updated, err := languageNavigationHTML([]byte(before), "en", map[string]string{"ca": "/about/", "en": "/en/about/"})
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(before, "</template>") + len("</template>")
	end := strings.Index(before[start:], "</header>") + start
	if !bytes.HasPrefix(updated, []byte(before[:start])) || !bytes.HasSuffix(updated, []byte(before[end:])) {
		t.Fatal("bytes outside the real header menu changed")
	}
	for _, bad := range []string{"<body><header></header></body>", "<body><header>" + languageMenuFixture + languageMenuFixture + "</header></body>", `<body><header><nav class="site-language-nav">`, `<body><header><nav class="site-language-nav"><nav></nav></nav></header>`, `<body><header><nav class="other" class="site-language-nav"></nav></header>`} {
		if _, err := languageNavigationHTML([]byte(bad), "en", nil); err == nil {
			t.Fatalf("invalid header menu accepted: %s", bad)
		}
	}
}

func TestLanguageNavigationFailurePreservesLastSuccessfulDeployment(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	put(t, root, ".zas/translations.yml", "version: 2\n")
	err := Build(context.Background(), Options{Root: root, Zas: generator(t, ordinaryOutput), Output: &bytes.Buffer{}})
	if err == nil || !strings.Contains(err.Error(), "assembled language navigation rejected") {
		t.Fatalf("expected language-navigation failure, got %v", err)
	}
	previous, err := os.ReadFile(filepath.Join(root, ".zas/deploy/previous.txt"))
	if err != nil || string(previous) != "last successful deployment" {
		t.Fatalf("prior deployment lost on failed language validation: %q %v", previous, err)
	}
}
