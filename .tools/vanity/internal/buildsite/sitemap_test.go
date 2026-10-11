package buildsite

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"dario.cat/site-tools/internal/vanity"
)

const nativeSitemapFixture = `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
  <url><loc>https://dario.cat/myindex.html</loc></url>
  <url><loc>https://dario.cat/es/guide/index.html</loc><lastmod>2025-07-09T10:00:00+02:00</lastmod>
    <xhtml:link rel="alternate" hreflang="es" href="https://dario.cat/es/guide/index.html" />
    <xhtml:link rel="alternate" hreflang="en" href="https://dario.cat/en/guide/index.html" />
  </url>
  <url><loc>https://dario.cat/contact.html</loc></url>
  <url><loc>https://dario.cat/index.html</loc></url>
  <url><loc>https://dario.cat/en/guide/index.html</loc>
    <xhtml:link rel="alternate" hreflang="es" href="https://dario.cat/es/guide/index.html" />
    <xhtml:link rel="alternate" hreflang="en" href="https://dario.cat/en/guide/index.html" />
  </url>
</urlset>`

func sitemapFixture(t *testing.T) (root, deploy string) {
	t.Helper()
	root = sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  sitemap: true\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	deploy = filepath.Join(root, ".zas", "deploy")
	pages := map[string]string{
		"index.html":          `<html lang="ca"><head><link rel="canonical" href="https://dario.cat/"></head><body>Home</body></html>`,
		"en/guide/index.html": `<html lang="en"><head><link rel="canonical" href="https://dario.cat/en/guide/"></head><body>Guide</body></html>`,
		"es/guide/index.html": `<html lang="es"><head><link rel="canonical" href="https://dario.cat/es/guide/"></head><body>Guía</body></html>`,
		"contact.html":        `<html lang="ca"><head><link rel="canonical" href="https://dario.cat/contact.html"></head><body>Contact</body></html>`,
		"myindex.html":        `<html lang="ca"><head><link rel="canonical" href="https://dario.cat/myindex.html"></head><body>Standalone</body></html>`,
	}
	for file, body := range pages {
		put(t, deploy, file, body)
	}
	for _, route := range vanity.Routes() {
		data, err := os.ReadFile(filepath.Join(root, route.File()))
		if err != nil {
			t.Fatal(err)
		}
		put(t, deploy, route.File(), string(data))
	}
	put(t, deploy, "sitemap.xml", nativeSitemapFixture)
	return root, deploy
}

func TestSitemapFinalizationPreservesMetadataAndAddsRawPages(t *testing.T) {
	root, deploy := sitemapFixture(t)
	for _, args := range [][]string{
		{"add", "mergo/index.html"},
		{"-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "commit", "-q", "-m", "fixture"},
	} {
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2024-03-01T12:00:00+01:00", "GIT_COMMITTER_DATE=2024-03-01T12:00:00+01:00")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("fixture git %v: %v %s", args, err, output)
		}
	}
	before, err := os.ReadFile(filepath.Join(deploy, "mergo/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if err := finalizeSitemap(context.Background(), root, root, deploy); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(deploy, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var sitemap siteSitemap
	if err := readNativeSitemap(data, &sitemap); err != nil {
		t.Fatalf("output is not supported namespaced XML: %v\n%s", err, data)
	}
	if len(sitemap.URLs) != 5+len(vanity.Routes()) {
		t.Fatalf("wrong entry count: %d", len(sitemap.URLs))
	}
	locations := make([]string, 0, len(sitemap.URLs))
	for _, entry := range sitemap.URLs {
		locations = append(locations, entry.Location)
		if strings.HasSuffix(entry.Location, "/index.html") {
			t.Errorf("uncanonical location: %s", entry.Location)
		}
		switch entry.Location {
		case "https://dario.cat/es/guide/":
			if entry.Lastmod != "2025-07-09T10:00:00+02:00" || len(entry.Links) != 2 {
				t.Fatalf("native lastmod or alternates lost: %#v", entry)
			}
		case "https://dario.cat/mergo/":
			if entry.Lastmod != "2024-03-01T12:00:00+01:00" {
				t.Fatalf("raw lastmod did not use source Git history: %s", entry.Lastmod)
			}
		case "https://dario.cat/darna/":
			if entry.Lastmod != "" {
				t.Fatalf("untracked raw page invented a lastmod: %s", entry.Lastmod)
			}
		}
		for _, link := range entry.Links {
			if link.URL != "https://dario.cat/"+link.Language+"/guide/" {
				t.Errorf("alternate not canonical: %#v", link)
			}
		}
	}
	if !slices.IsSorted(locations) || !slices.Contains(locations, "https://dario.cat/contact.html") || !slices.Contains(locations, "https://dario.cat/myindex.html") {
		t.Fatalf("location order or standalone HTML URLs changed: %v", locations)
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	links := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if element, ok := token.(xml.StartElement); ok && element.Name.Local == "link" {
			links++
			if element.Name.Space != xhtmlNamespace {
				t.Fatalf("alternate has wrong namespace: %#v", element.Name)
			}
		}
	}
	if links != 4 {
		t.Fatalf("wrong namespace-aware alternate count: %d", links)
	}
	after, _ := os.ReadFile(filepath.Join(deploy, "mergo/index.html"))
	if !bytes.Equal(before, after) {
		t.Fatal("raw vanity bytes changed")
	}
	if err := finalizeSitemap(context.Background(), root, root, deploy); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(filepath.Join(deploy, "sitemap.xml"))
	if !bytes.Equal(data, again) {
		t.Fatal("second finalization is not byte-identical")
	}
}

func TestSitemapFinalizationRejectsIncompleteOrUnsafeOutput(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{"offsite canonical", func(t *testing.T, deploy string) {
			put(t, deploy, "contact.html", `<link rel="canonical" href="https://evil.example/contact.html">`)
		}},
		{"wrong canonical path", func(t *testing.T, deploy string) {
			put(t, deploy, "contact.html", `<link rel="canonical" href="https://dario.cat/">`)
		}},
		{"duplicate canonical links", func(t *testing.T, deploy string) {
			put(t, deploy, "contact.html", `<link rel="canonical" href="https://dario.cat/contact.html"><link rel="canonical" href="https://dario.cat/contact.html">`)
		}},
		{"missing canonical", func(t *testing.T, deploy string) {
			put(t, deploy, "contact.html", `<head><script>const example = '<link rel="canonical" href="https://dario.cat/contact.html">';</script></head><body>Example</body>`)
		}},
		{"canonical outside head", func(t *testing.T, deploy string) {
			put(t, deploy, "contact.html", `<head></head><body><link rel="canonical" href="https://dario.cat/contact.html"></body>`)
		}},
		{"sharded sitemap", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap-index.xml", "index")
		}},
		{"missing native page", func(t *testing.T, deploy string) {
			put(t, deploy, "other.html", `<link rel="canonical" href="https://dario.cat/other.html">`)
		}},
		{"stale native reference", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, "https://dario.cat/contact.html", "https://dario.cat/missing.html", 1))
		}},
		{"duplicate native page", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, "</urlset>", `<url><loc>https://dario.cat/</loc></url></urlset>`, 1))
		}},
		{"empty native loc", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, "<loc>https://dario.cat/contact.html</loc>", "<loc></loc>", 1))
		}},
		{"duplicate native loc field", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, "<loc>https://dario.cat/contact.html</loc>", "<loc>https://dario.cat/contact.html</loc><loc>https://dario.cat/contact.html</loc>", 1))
		}},
		{"offsite alternate", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, `href="https://dario.cat/es/guide/index.html"`, `href="https://evil.example/es/guide/index.html"`, 1))
		}},
		{"missing alternate page", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, `href="https://dario.cat/es/guide/index.html"`, `href="https://dario.cat/es/missing/index.html"`, 1))
		}},
		{"wrong alternate language", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, `hreflang="es"`, `hreflang="ca"`, 1))
		}},
		{"nonreciprocal alternate", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, `<xhtml:link rel="alternate" hreflang="en" href="https://dario.cat/en/guide/index.html" />`, "", 1))
		}},
		{"unknown native field", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.Replace(nativeSitemapFixture, "</urlset>", "<future>unsupported</future></urlset>", 1))
		}},
		{"wrong XML namespace", func(t *testing.T, deploy string) {
			put(t, deploy, "sitemap.xml", strings.ReplaceAll(nativeSitemapFixture, sitemapNamespace, "https://evil.example/ns"))
		}},
		{"sitemap symlink", func(t *testing.T, deploy string) {
			if err := os.Remove(filepath.Join(deploy, "sitemap.xml")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(deploy, "contact.html"), filepath.Join(deploy, "sitemap.xml")); err != nil {
				t.Skip(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, deploy := sitemapFixture(t)
			tc.mutate(t, deploy)
			if err := finalizeSitemap(context.Background(), root, root, deploy); err == nil {
				t.Fatal("unsafe or incomplete sitemap was accepted")
			}
		})
	}
}

func TestSitemapGuardFailurePreservesPreviousDeployment(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  sitemap: true\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	script := generator(t, `
mkdir -p .zas/deploy
cat > .zas/deploy/index.html <<'EOF'
<html><head><link rel="canonical" href="https://evil.example/"></head><body>Unsafe</body></html>
EOF
cat > .zas/deploy/sitemap.xml <<'EOF'
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://dario.cat/index.html</loc></url></urlset>
EOF
`)
	if err := Build(context.Background(), Options{Root: root, Zas: script}); err == nil || !strings.Contains(err.Error(), "assembled sitemap rejected") {
		t.Fatalf("expected sitemap guard failure, got %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".zas/deploy/previous.txt"))
	if err != nil || string(data) != "last successful deployment" {
		t.Fatalf("previous deployment lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".zas/deploy/index.html")); !os.IsNotExist(err) {
		t.Fatal("rejected page reached previous deployment")
	}
}
