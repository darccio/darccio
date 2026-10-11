package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func structuredDocument(t *testing.T, raw string) string {
	t.Helper()
	return strings.Replace(fixture(t, "post.html"), "</head>", `<script type="application/ld+json">`+raw+`</script></head>`, 1)
}

func jsonFixture(t *testing.T) map[string]any {
	t.Helper()
	data, err := decodeJSONObject(fixture(t, "jsonld-post.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func marshalFixture(t *testing.T, value any) string {
	t.Helper()
	input, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(input)
}

func graphNode(data map[string]any, index int) map[string]any {
	return data["@graph"].([]any)[index].(map[string]any)
}

func TestJSONLDFixtureAndSemanticRegressions(t *testing.T) {
	post := examplePost()
	doc, mf := parsed(t, structuredDocument(t, fixture(t, "jsonld-post.json")), post.URL)
	if err := validatePost(post, doc, mf); err != nil {
		t.Fatal(err)
	}
	if err := validateJSONLD(doc, post.URL, &post); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"wrong context", func(d map[string]any) { d["@context"] = "http://schema.org" }},
		{"organization identity", func(d map[string]any) { graphNode(d, 0)["name"] = "Datadog" }},
		{"wrong person ID", func(d map[string]any) { graphNode(d, 0)["@id"] = homeURL + "#other" }},
		{"wrong person URL", func(d map[string]any) { graphNode(d, 0)["url"] = homeURL + "about/" }},
		{"remote avatar", func(d map[string]any) { graphNode(d, 0)["image"] = "https://gravatar.com/avatar/example" }},
		{"wrong job title", func(d map[string]any) { graphNode(d, 0)["jobTitle"] = "Principal Software Engineer" }},
		{"unnested organization", func(d map[string]any) { graphNode(d, 0)["worksFor"] = "Datadog" }},
		{"wrong organization", func(d map[string]any) { graphNode(d, 0)["worksFor"].(map[string]any)["name"] = "Example" }},
		{"missing Sifa", func(d map[string]any) {
			p := graphNode(d, 0)
			p["sameAs"] = append(p["sameAs"].([]any)[:4], p["sameAs"].([]any)[5:]...)
		}},
		{"duplicate profile", func(d map[string]any) { graphNode(d, 0)["sameAs"].([]any)[4] = profiles[0] }},
		{"missing Speaker Deck", func(d map[string]any) { graphNode(d, 0)["sameAs"] = graphNode(d, 0)["sameAs"].([]any)[:6] }},
		{"string profiles", func(d map[string]any) { graphNode(d, 0)["sameAs"] = strings.Join(profiles, ",") }},
		{"missing Person", func(d map[string]any) { d["@graph"] = d["@graph"].([]any)[1:] }},
		{"duplicate Person", func(d map[string]any) { d["@graph"].([]any)[1] = graphNode(d, 0) }},
		{"duplicate article", func(d map[string]any) { d["@graph"].([]any)[0] = graphNode(d, 1) }},
		{"string graph", func(d map[string]any) { d["@graph"] = "Person" }},
		{"quoted graph node", func(d map[string]any) { d["@graph"].([]any)[1] = "BlogPosting" }},
		{"unexpected type", func(d map[string]any) { graphNode(d, 1)["@type"] = "WebPage" }},
		{"wrong headline", func(d map[string]any) { graphNode(d, 1)["headline"] = "Different article" }},
		{"wrong article canonical", func(d map[string]any) { graphNode(d, 1)["url"] = post.URL + "index.html" }},
		{"wrong description", func(d map[string]any) { graphNode(d, 1)["description"] = "Different summary" }},
		{"wrong tags", func(d map[string]any) { graphNode(d, 1)["keywords"] = []any{"go", "ruby"} }},
		{"string tags", func(d map[string]any) { graphNode(d, 1)["keywords"] = "go,rss" }},
		{"wrong language", func(d map[string]any) { graphNode(d, 1)["inLanguage"] = "ca" }},
		{"string author", func(d map[string]any) { graphNode(d, 1)["author"] = owner }},
		{"wrong author", func(d map[string]any) {
			graphNode(d, 1)["author"].(map[string]any)["@id"] = "https://github.com/darccio"
		}},
		{"wrong publication offset", func(d map[string]any) { graphNode(d, 1)["datePublished"] = "2024-07-01T10:00:00+01:00" }},
		{"normalized publication offset", func(d map[string]any) { graphNode(d, 1)["datePublished"] = "2024-07-01T08:00:00Z" }},
		{"wrong update date", func(d map[string]any) { graphNode(d, 1)["dateModified"] = "2024-08-02T12:30:00+02:00" }},
		{"zero publication date", func(d map[string]any) { graphNode(d, 1)["datePublished"] = "0001-01-01T00:00:00Z" }},
		{"null publication date", func(d map[string]any) { graphNode(d, 1)["datePublished"] = nil }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			data := jsonFixture(t)
			test.mutate(data)
			doc, _ := parsed(t, structuredDocument(t, marshalFixture(t, data)), post.URL)
			if err := validateJSONLD(doc, post.URL, &post); err == nil {
				t.Fatal("structured-data regression accepted")
			}
		})
	}
}

func TestJSONLDSyntaxAndScriptPlacement(t *testing.T) {
	post := examplePost()
	raw := fixture(t, "jsonld-post.json")
	good := structuredDocument(t, raw)
	cases := []struct{ name, input string }{
		{"no script", fixture(t, "post.html")},
		{"duplicate script", strings.Replace(good, "</head>", `<script type="application/ld+json">`+raw+`</script></head>`, 1)},
		{"body script", strings.Replace(strings.Replace(good, `<script type="application/ld+json">`+raw+`</script>`, "", 1), "</body>", `<script type="application/ld+json">`+raw+`</script></body>`, 1)},
		{"wrong MIME type", strings.Replace(good, "application/ld+json", "application/json", 1)},
		{"malformed JSON", structuredDocument(t, strings.TrimSuffix(strings.TrimSpace(raw), "}"))},
		{"quoted document", structuredDocument(t, marshalFixture(t, raw))},
		{"array document", structuredDocument(t, "["+raw+"]")},
		{"trailing document", structuredDocument(t, raw+"{}")},
		{"duplicate top key", structuredDocument(t, strings.Replace(raw, `"@context": "https://schema.org",`, `"@context": "http://schema.org", "@context": "https://schema.org",`, 1))},
		{"duplicate nested key", structuredDocument(t, strings.Replace(raw, `"name": "Dario Castañé",`, `"name": "Datadog", "name": "Dario Castañé",`, 1))},
		{"HTML entities in script", structuredDocument(t, strings.Replace(raw, `"headline": "Example article"`, `"headline": "Example &quot;article&quot;"`, 1))},
		{"unescaped script injection", structuredDocument(t, strings.Replace(raw, `"headline": "Example article"`, `"headline": "</script><script>alert(1)</script>"`, 1))},
		{"wrong head canonical", strings.Replace(good, `rel="canonical" href="`+post.URL+`"`, `rel="canonical" href="`+homeURL+`other/"`, 1)},
		{"body-only profile", strings.Replace(strings.Replace(good, `<link rel="me" href="https://sifa.id/p/dario.cat">`, "", 1), "<body>", `<body><link rel="me" href="https://sifa.id/p/dario.cat">`, 1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			doc, _ := parsed(t, test.input, post.URL)
			if err := validateJSONLD(doc, post.URL, &post); err == nil {
				t.Fatal("invalid JSON-LD script accepted")
			}
		})
	}
}

func TestJSONLDSafeTextAndFractionalDates(t *testing.T) {
	post := examplePost()
	post.Title = `Quotes " & català </script><script>alert("x")</script>` + "\u2028\u2029"
	post.Summary = `A "quoted" summary & Unicode: 日本語.`
	post.Tags = []string{"go", `quotes " & català`}
	post.Date, _ = time.Parse(time.RFC3339Nano, "2024-07-01T10:00:00.123456789+02:00")
	post.Updated, _ = time.Parse(time.RFC3339Nano, "2024-08-01T12:30:00.987654321+02:00")
	data := jsonFixture(t)
	article := graphNode(data, 1)
	article["headline"], article["description"], article["keywords"] = post.Title, post.Summary, post.Tags
	article["datePublished"], article["dateModified"] = post.Date.Format(time.RFC3339Nano), post.Updated.Format(time.RFC3339Nano)
	doc, _ := parsed(t, structuredDocument(t, marshalFixture(t, data)), post.URL)
	if err := validateJSONLD(doc, post.URL, &post); err != nil {
		t.Fatal(err)
	}
	if len(find(doc, func(n *htmlNode) bool { return isTag(n, "script") })) != 1 {
		t.Fatal("encoded text introduced an executable script")
	}
	// Cloudflare adds its own beacon outside the metadata block after deployment.
	doc, _ = parsed(t, strings.Replace(structuredDocument(t, marshalFixture(t, data)), "</body>", `<script defer src="https://static.cloudflareinsights.com/beacon.min.js"></script></body>`, 1), post.URL)
	if err := validateJSONLD(doc, post.URL, &post); err != nil {
		t.Fatal(err)
	}
}

func TestJSONLDOmitsUnknownDatesAndNonArticleNodes(t *testing.T) {
	post := examplePost()
	post.Date, post.Updated = time.Time{}, time.Time{}
	data := jsonFixture(t)
	article := graphNode(data, 1)
	delete(article, "datePublished")
	delete(article, "dateModified")
	doc, _ := parsed(t, structuredDocument(t, marshalFixture(t, data)), post.URL)
	if err := validateJSONLD(doc, post.URL, &post); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"datePublished", "dateModified"} {
		for _, value := range []any{"0001-01-01T00:00:00Z", "2024-07-01T00:00:00+02:00", "", nil} {
			article[key] = value
			doc, _ := parsed(t, structuredDocument(t, marshalFixture(t, data)), post.URL)
			if err := validateJSONLD(doc, post.URL, &post); err == nil {
				t.Fatalf("unknown %s was accepted as %v", key, value)
			}
		}
		delete(article, key)
	}
	for _, slug := range []string{"posts/", "now/", "projects/", "posts/delete-facebook-3/"} {
		t.Run(slug, func(t *testing.T) {
			data := jsonFixture(t)
			data["@graph"] = data["@graph"].([]any)[:1]
			pageURL := homeURL + slug
			input := strings.ReplaceAll(structuredDocument(t, marshalFixture(t, data)), post.URL, pageURL)
			doc, _ := parsed(t, input, pageURL)
			if err := validateJSONLD(doc, pageURL, nil); err != nil {
				t.Fatal(err)
			}
			data["@graph"] = append(data["@graph"].([]any), article)
			doc, _ = parsed(t, strings.ReplaceAll(structuredDocument(t, marshalFixture(t, data)), post.URL, pageURL), pageURL)
			if err := validateJSONLD(doc, pageURL, nil); err == nil {
				t.Fatal("nonarticle exposed as BlogPosting")
			}
		})
	}
}

func TestJSONLDDirectoryExemptsOnlyReviewedVanityFragments(t *testing.T) {
	post := examplePost()
	inv := inventory{Posts: []Post{post}}
	newDeploy := func(t *testing.T) string {
		deploy := t.TempDir()
		put(t, deploy, post.File, structuredDocument(t, fixture(t, "jsonld-post.json")))
		person := jsonFixture(t)
		person["@graph"] = person["@graph"].([]any)[:1]
		put(t, deploy, "index.html", strings.ReplaceAll(structuredDocument(t, marshalFixture(t, person)), post.URL, homeURL))
		for file := range rawVanityHTML {
			put(t, deploy, file, `<head><meta name="go-import" content="existing baseline preserved"></head>`)
		}
		put(t, deploy, "CV/MAC.json", "invalid JSON which must never be opened by this validator")
		if err := os.Chmod(filepath.Join(deploy, "CV", "MAC.json"), 0); err != nil {
			t.Fatal(err)
		}
		return deploy
	}
	if err := validateJSONLDPages(newDeploy(t), inv); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, file, input string }{
		{"missing on normal page", "about/index.html", `<head></head><body>About</body>`},
		{"unreviewed vanity exception", "unreviewed/index.html", `<head><meta name="go-import" content="unreviewed"></head>`},
		{"structured data in raw vanity", "mergo/index.html", structuredDocument(t, fixture(t, "jsonld-post.json"))},
		{"unexpected CV output", "CV/index.html", `private data must not be read`},
		{"missing article structured data", post.File, fixture(t, "post.html")},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			deploy := newDeploy(t)
			put(t, deploy, test.file, test.input)
			if err := validateJSONLDPages(deploy, inv); err == nil {
				t.Fatal("incomplete structured-data deployment accepted")
			}
		})
	}
	t.Run("CV HTML rejected before opening", func(t *testing.T) {
		deploy := newDeploy(t)
		put(t, deploy, "CV/rendered/index.html", "CV body must never be opened by this validator")
		if err := os.Chmod(filepath.Join(deploy, "CV", "rendered", "index.html"), 0); err != nil {
			t.Fatal(err)
		}
		err := validateJSONLDPages(deploy, inv)
		if err == nil || !strings.Contains(err.Error(), "unexpected CV HTML output") {
			t.Fatalf("expected CV filename rejection before reading file, got %v", err)
		}
	})
	t.Run("missing eligible output", func(t *testing.T) {
		missing := examplePost()
		missing.File = "en/posts/missing/index.html"
		if err := validateJSONLDPages(newDeploy(t), inventory{Posts: []Post{post, missing}}); err == nil {
			t.Fatal("missing published article output accepted")
		}
	})
}

func TestSoftwareSourceCodeUsesVerifiedProjectMetadata(t *testing.T) {
	for project, name := range map[string]string{"mergo": "Mergo", "zas": "Zas", "darna": "darna"} {
		t.Run(project, func(t *testing.T) {
			pageURL := homeURL + project + "/"
			data := jsonFixture(t)
			data["@graph"] = []any{graphNode(data, 0), map[string]any{"@type": "SoftwareSourceCode", "name": name, "url": pageURL, "codeRepository": "https://github.com/darccio/" + project, "programmingLanguage": "Go", "license": homeURL + project + "/LICENSE.txt"}}
			document := func() string {
				return strings.ReplaceAll(structuredDocument(t, marshalFixture(t, data)), examplePost().URL, pageURL)
			}
			doc, _ := parsed(t, document(), pageURL)
			if err := validateJSONLD(doc, pageURL, nil); err != nil {
				t.Fatal(err)
			}
			for key := range graphNode(data, 1) {
				t.Run(key, func(t *testing.T) {
					node := graphNode(data, 1)
					original := node[key]
					node[key] = "incorrect"
					defer func() { node[key] = original }()
					doc, _ := parsed(t, document(), pageURL)
					if err := validateJSONLD(doc, pageURL, nil); err == nil {
						t.Fatal("accepted mismatched project metadata")
					}
				})
			}
		})
	}
}
