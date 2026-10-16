package check

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/darccio/zas"
)

var generatedArchiveRoutes = []struct{ path, language string }{
	{"posts", ""}, {"posts/ca", "ca"}, {"en/posts", "en"}, {"es/posts", "es"},
}

type archiveFixtureArticle struct {
	path, title, language, date, flags string
}

func (article archiveFixtureArticle) post(t *testing.T) Post {
	t.Helper()
	date, err := time.Parse(time.RFC3339, article.date)
	if err != nil {
		t.Fatal(err)
	}
	return Post{URL: homeURL + article.path + "/", Title: article.title, Language: article.language, Date: date}
}

// Render the actual checked-in archive pages and shared partials, rather than a
// second template implementation that could pass while the live archive drifts.
func archiveGenerationFixture(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, archive := range generatedArchiveRoutes {
		path := archive.path + "/index.md"
		input, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		put(t, root, path, string(input))
	}
	partials, err := filepath.Glob(filepath.Join(source, ".zas", "partials", "archive-*.html"))
	if err != nil || len(partials) == 0 {
		t.Fatalf("archive partials: %v (%d found)", err, len(partials))
	}
	for _, path := range partials {
		input, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		put(t, root, ".zas/partials/"+filepath.Base(path), string(input))
	}
	put(t, root, ".zas/config.yml", `site:
  baseurl: https://dario.cat
  language: ca
  page_listings: true
  publish_markdown: true
zas:
  layout: .zas/layout.html
  deploy: .zas/deploy
mimetypes:
  text/html: html
  text/markdown: markdown
`)
	put(t, root, ".zas/layout.html", `<!doctype html><html lang="{{.Language}}"><head><title>{{.Title}}</title></head><body><main>{{.Body}}</main></body></html>`)
	put(t, root, "en/.zas.yml", "language: en\n")
	put(t, root, "es/.zas.yml", "language: es\n")
	return root
}

func writeArchiveFixtureArticle(t *testing.T, root string, article archiveFixtureArticle) {
	t.Helper()
	date := ""
	if article.date != "" {
		date = "date: " + strconv.Quote(article.date) + "\n"
	}
	put(t, root, article.path+"/index.md", fmt.Sprintf("<!--\ntitle: %s\narticle: true\nlanguage: %s\n%s%s-->\n# Fixture body\n\nA sample article.\n", strconv.Quote(article.title), article.language, date, article.flags))
}

func buildArchiveFixture(t *testing.T, generator *zas.Generator) {
	t.Helper()
	if err := generator.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedArchivesFollowPublicationLifecycle(t *testing.T) {
	root := archiveGenerationFixture(t)
	articles := []archiveFixtureArticle{
		{"posts/zeta", "Català & <signe>", "ca", "2024-12-01T11:00:00+01:00", ""},
		{"en/posts/alpha", "English & <sign>", "en", "2024-12-01T10:00:00Z", ""},
		{"posts/late-2020", "Final de 2020", "ca", "2020-12-31T23:00:00+01:00", ""},
		{"en/posts/early-2020", "Start of 2020", "en", "2020-01-01T01:00:00+01:00", ""},
		{"es/posts/late-2015", "Final de 2015", "es", "2015-12-31T23:00:00+01:00", ""},
		{"rizoma/early-2015", "Rizoma de 2015", "ca", "2015-01-01T01:00:00+01:00", ""},
		{"posts/old", "Article antic", "ca", "2010-01-01T10:00:00+01:00", ""},
	}
	for _, article := range articles {
		writeArchiveFixtureArticle(t, root, article)
	}
	for _, excluded := range []archiveFixtureArticle{
		{"posts/private", "Private article", "ca", "2030-01-01T00:00:00Z", "publish: false\n"},
		{"en/posts/draft", "Draft article", "en", "2030-01-01T00:00:00Z", "draft: true\n"},
		{"es/posts/undated", "Undated article", "es", "", ""},
		{"projects/dated", "Dated project page", "ca", "2030-01-01T00:00:00Z", ""},
		{"rizoma/index-page", "Undated newsletter index", "ca", "", ""},
	} {
		writeArchiveFixtureArticle(t, root, excluded)
	}
	t.Chdir(root)
	generator := zas.NewGenerator(false, true, true)
	buildArchiveFixture(t, generator)
	assertGeneratedArchives(t, root, articles)
	generator.Full = false

	// New posts must appear without editing any archive source; metadata changes
	// must refresh every applicable archive, even with an older source mtime.
	newArticle := archiveFixtureArticle{"es/posts/new", "Nou article", "es", "2027-06-01T12:00:00+02:00", ""}
	articles = append(articles, newArticle)
	writeArchiveFixtureArticle(t, root, newArticle)
	articles[1].title, articles[1].date = "A renamed English article", "2014-06-01T12:00:00+02:00"
	writeArchiveFixtureArticle(t, root, articles[1])
	old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	changed := filepath.Join(root, "en/posts/alpha/index.md")
	if err := os.Chtimes(changed, old, old); err != nil {
		t.Fatal(err)
	}
	buildArchiveFixture(t, generator)
	assertGeneratedArchives(t, root, articles)
	assertArchiveFullBuildMatchesIncremental(t, root)

	// Deletion and each public/private, draft and missing-date transition remove
	// archive entries; restoring publication must restore the very same URL.
	if err := os.Remove(filepath.Join(root, "posts/old/index.md")); err != nil {
		t.Fatal(err)
	}
	articles = slices.DeleteFunc(articles, func(article archiveFixtureArticle) bool { return article.path == "posts/old" })
	for i, flags := range []string{"publish: false\n", "draft: true\n", ""} {
		changed := articles[i]
		changed.flags = flags
		if i == 2 {
			changed.date = ""
		}
		writeArchiveFixtureArticle(t, root, changed)
	}
	buildArchiveFixture(t, generator)
	assertGeneratedArchives(t, root, articles[3:])
	assertArchiveFullBuildMatchesIncremental(t, root)
	for _, article := range articles[:3] {
		writeArchiveFixtureArticle(t, root, article)
	}
	buildArchiveFixture(t, generator)
	assertGeneratedArchives(t, root, articles)
	assertArchiveFullBuildMatchesIncremental(t, root)
}

func TestGeneratedArchivesKeepYearOnlyCitationsWithoutLocalPosts(t *testing.T) {
	root := archiveGenerationFixture(t)
	t.Chdir(root)
	buildArchiveFixture(t, zas.NewGenerator(false, true, true))
	assertGeneratedArchives(t, root, nil)
}

func assertGeneratedArchives(t *testing.T, root string, articles []archiveFixtureArticle) {
	t.Helper()
	posts := make([]Post, 0, len(articles))
	for _, article := range articles {
		posts = append(posts, article.post(t))
	}
	for _, archive := range generatedArchiveRoutes {
		for _, extension := range []string{"html", "md"} {
			name := archive.path + "/index." + extension
			t.Run(name, func(t *testing.T) {
				input, err := os.ReadFile(filepath.Join(root, ".zas/deploy", filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(input, []byte("{{")) || bytes.Contains(input, []byte("<embed")) {
					t.Fatal("archive output leaked an unrendered template or embed")
				}
				archiveURL := homeURL + archive.path + "/"
				doc, data := parsed(t, string(input), archiveURL)
				if err := validateArchive(archiveURL, archive.language, posts, doc, data); err != nil {
					t.Fatal(err)
				}
				assertArchiveChronology(t, doc)
				// The mf2 parser expands URL attributes in place. Check original
				// navigation markup with a fresh DOM, before that normalization.
				raw, err := parseHTML(input)
				if err != nil {
					t.Fatal(err)
				}
				preview := "https://preview.example/" + archive.path + "/"
				for _, row := range find(raw, func(node *htmlNode) bool { return hasClass(node, "h-entry") }) {
					link, err := uniqueNode(row, func(node *htmlNode) bool { return isTag(node, "a") && hasClass(node, "u-url") }, "article link")
					if err != nil {
						t.Fatal(err)
					}
					href := attribute(link, "href")
					if !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "//") || resolve(preview, href) != "https://preview.example"+href {
						t.Errorf("article link %q leaves the preview origin", href)
					}
				}
			})
		}
	}
}

// A year-only external citation has no invented publication timestamp: rows
// must respect its year, while dated local rows compare complete instants.
func assertArchiveChronology(t *testing.T, doc *htmlNode) {
	t.Helper()
	lastYear := int(^uint(0) >> 1)
	var lastDate time.Time
	lastURL := ""
	for _, row := range find(doc, func(node *htmlNode) bool { return hasClass(node, "archive-item") }) {
		rail, err := uniqueNode(row, func(node *htmlNode) bool { return hasClass(node, "archive-item__when") }, "year rail")
		if err != nil {
			t.Fatal(err)
		}
		year, err := strconv.Atoi(textContent(rail))
		if err != nil || year > lastYear {
			t.Fatalf("archive years must descend, got %q after %d", textContent(rail), lastYear)
		}
		lastYear = year
		if hasClass(row, "h-cite") {
			if times := find(row, func(node *htmlNode) bool { return isTag(node, "time") || hasClass(node, "dt-published") }); len(times) != 0 {
				t.Fatal("year-only citation gained an unsupported publication date")
			}
			continue
		}
		stamp, err := uniqueNode(row, func(node *htmlNode) bool { return hasClass(node, "dt-published") }, "publication time")
		if err != nil {
			t.Fatal(err)
		}
		date, err := time.Parse(time.RFC3339, attribute(stamp, "datetime"))
		if err != nil {
			t.Fatal(err)
		}
		link, err := uniqueNode(row, func(node *htmlNode) bool { return isTag(node, "a") && hasClass(node, "u-url") }, "article URL")
		if err != nil {
			t.Fatal(err)
		}
		url := attribute(link, "href")
		if !lastDate.IsZero() && (date.After(lastDate) || date.Equal(lastDate) && url < lastURL) {
			t.Fatalf("article chronology/tie order changed: %s %s after %s %s", date, url, lastDate, lastURL)
		}
		lastDate, lastURL = date, url
	}
}

func assertArchiveFullBuildMatchesIncremental(t *testing.T, root string) {
	t.Helper()
	before := make(map[string][]byte)
	for _, archive := range generatedArchiveRoutes {
		for _, extension := range []string{"html", "md"} {
			path := filepath.Join(root, ".zas/deploy", filepath.FromSlash(archive.path), "index."+extension)
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			before[path] = input
		}
	}
	buildArchiveFixture(t, zas.NewGenerator(false, true, true))
	for path, input := range before {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(input, after) {
			t.Errorf("fresh full build differs from incremental archive %s: %v", path, err)
		}
	}
}
