package consistency

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct{ root, deploy, plan string }

func fixtureSite(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join("testdata", "site")
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dir, rel)
		if entry.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, input, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{filepath.Join(dir, "source"), filepath.Join(dir, "deploy"), filepath.Join(dir, "plan.json")}
}

func replace(t *testing.T, file, before, after string) {
	t.Helper()
	input, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(input), before) {
		t.Fatalf("fixture lacks requested replacement %q in %s", before, file)
	}
	if err := os.WriteFile(file, []byte(strings.Replace(string(input), before, after, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mutateJSON(t *testing.T, file string, mutate func(map[string]any)) {
	t.Helper()
	input, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := json.Unmarshal(input, &data); err != nil {
		t.Fatal(err)
	}
	mutate(data)
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, output, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mutatePlan(t *testing.T, f fixture, mutate func(map[string]any)) {
	t.Helper()
	mutateJSON(t, f.plan, func(data map[string]any) {
		row := data["documents"].([]any)[0].(map[string]any)
		mutate(row["record"].(map[string]any))
	})
}

func mutateLD(t *testing.T, file string, mutate func(map[string]any)) {
	t.Helper()
	input, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	body := string(input)
	start := strings.Index(body, `<script type="application/ld+json">`)
	if start < 0 {
		t.Fatal("fixture lacks JSON-LD")
	}
	start += len(`<script type="application/ld+json">`)
	end := start + strings.Index(body[start:], "</script>")
	var data map[string]any
	if err := json.Unmarshal([]byte(body[start:end]), &data); err != nil {
		t.Fatal(err)
	}
	mutate(data["@graph"].([]any)[1].(map[string]any))
	output, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body[:start]+string(output)+body[end:]), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDirectoryComparesIndependentFormatsOffline(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	f := fixtureSite(t)
	// This sentinel deliberately cannot be opened as a regular file; the
	// validator must never select or process the public CV directory.
	if err := os.MkdirAll(filepath.Join(f.root, "CV", "MAC.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(f.deploy, "CV", "MAC.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(f.root, "posts", "CV"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(f.root, "posts", "CV", "sensitive.md"), "<!--\narticle: true\ndate: invalid\n-->\nNever process this CV sentinel.\n")
	report, err := Directory(f.root, f.deploy, f.plan)
	if err != nil {
		t.Fatal(err)
	}
	want := Report{Posts: 4, Dated: 3, Partial: 1, Excluded: 1, Feeds: 2, Items: 4, Approved: 1}
	if report != want {
		t.Fatalf("got %+v, want %+v", report, want)
	}
	// Root is capped to Alpha/Beta while the English feed retains Beta/Old;
	// partial dates and drafts intentionally have no RSS or Standard records.
}

func TestPublishedSubsetRequiresVerifiedStateAndMatchingLink(t *testing.T) {
	f := fixtureSite(t)
	const uri = "at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.document/3aaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(f.root, ".zas", "atproto.json"), []byte(`{"/posts/alpha/":"`+uri+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	replace(t, filepath.Join(f.deploy, "posts", "alpha", "index.html"), "</head>", `<link rel="site.standard.document" href="`+uri+`"></head>`)
	mutateJSON(t, f.plan, func(data map[string]any) {
		row := data["documents"].([]any)[0].(map[string]any)
		row["action"] = "unchanged"
		row["cid"] = "bafyreitest"
	})
	report, err := Directory(f.root, f.deploy, f.plan)
	if err != nil {
		t.Fatal(err)
	}
	if report.Published != 1 || report.Approved != 1 {
		t.Fatalf("unexpected publication counts %+v", report)
	}
}

func TestEmptyReviewedAllowlistStillRequiresAnExplicitOfflinePlan(t *testing.T) {
	f := fixtureSite(t)
	mutateJSON(t, filepath.Join(f.root, ".tools", "atproto", "documents-allowlist.json"), func(data map[string]any) { data["documents"] = []any{} })
	mutateJSON(t, f.plan, func(data map[string]any) { data["documents"] = []any{} })
	report, err := Directory(f.root, f.deploy, f.plan)
	if err != nil || report.Approved != 0 || report.Published != 0 {
		t.Fatalf("empty reviewed plan failed: %+v %v", report, err)
	}
	if _, err := Directory(f.root, f.deploy, ""); err == nil {
		t.Fatal("missing plan must fail even when no documents are approved")
	}
}

func TestEquivalentTimezoneInstantsAgree(t *testing.T) {
	f := fixtureSite(t)
	mutateJSON(t, filepath.Join(f.root, ".tools", "atproto", "documents-allowlist.json"), func(data map[string]any) {
		data["documents"].([]any)[0].(map[string]any)["publishedAt"] = "2024-01-01T09:00:00Z"
	})
	mutatePlan(t, f, func(record map[string]any) {
		record["publishedAt"] = "2024-01-01T04:00:00-05:00"
		record["updatedAt"] = "2024-02-01T10:00:00+01:00"
	})
	if _, err := Directory(f.root, f.deploy, f.plan); err != nil {
		t.Fatal(err)
	}
}

func TestDirectoryRejectsPublishingDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, fixture)
	}{
		{"missing article", func(t *testing.T, f fixture) { remove(t, filepath.Join(f.deploy, "posts/alpha/index.html")) }},
		{"missing combined feed", func(t *testing.T, f fixture) { remove(t, filepath.Join(f.deploy, "feed.xml")) }},
		{"missing language feed", func(t *testing.T, f fixture) { remove(t, filepath.Join(f.deploy, "en/feed.xml")) }},
		{"missing offline plan", func(t *testing.T, f fixture) { remove(t, f.plan) }},
		{"missing verified manifest", func(t *testing.T, f fixture) { remove(t, filepath.Join(f.root, ".zas/atproto.json")) }},
		{"missing allowlist", func(t *testing.T, f fixture) {
			remove(t, filepath.Join(f.root, ".tools/atproto/documents-allowlist.json"))
		}},
		{"missing publication verification", func(t *testing.T, f fixture) {
			remove(t, filepath.Join(f.deploy, ".well-known/site.standard.publication"))
		}},
		{"source date without timezone", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.root, "posts/alpha/index.md"), `date: "2024-01-01T10:00:00+01:00"`, `date: "2024-01-01T10:00:00"`)
		}},
		{"unquoted date cannot imply midnight", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.root, "posts/alpha/index.md"), `date: "2024-01-01T10:00:00+01:00"`, `date: 2024-01-01`)
		}},
		{"missing source summary", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.root, "posts/alpha/index.md"), `summary: "A & B <literal>."`, "")
		}},
		{"partial date opted into feed", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.root, "posts/partial/index.md"), "feed: false", "feed: true")
		}},
		{"partial date given invented JSON-LD instant", func(t *testing.T, f fixture) {
			mutateLD(t, filepath.Join(f.deploy, "posts/partial/index.html"), func(record map[string]any) { record["datePublished"] = "2020-01-01T00:00:00Z" })
		}},
		{"canonical index URL", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `rel="canonical" href="https://dario.cat/posts/alpha/"`, `rel="canonical" href="https://dario.cat/posts/alpha/index.html"`)
		}},
		{"Open Graph URL", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `property="og:url" content="https://dario.cat/posts/alpha/"`, `property="og:url" content="https://dario.cat/posts/other/"`)
		}},
		{"Open Graph publication", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `property="article:published_time" content="2024-01-01T09:00:00Z"`, `property="article:published_time" content="2024-02-01T09:00:00Z"`)
		}},
		{"h-entry publication", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `class="dt-published" datetime="2024-01-01T09:00:00Z"`, `class="dt-published" datetime="2024-01-01T10:00:00Z"`)
		}},
		{"h-entry update replaced with publication", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `class="dt-updated" datetime="2024-02-01T09:00:00Z"`, `class="dt-updated" datetime="2024-01-01T09:00:00Z"`)
		}},
		{"h-entry URL", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `class="u-url u-uid" href="https://dario.cat/posts/alpha/"`, `class="u-url u-uid" href="https://dario.cat/posts/alpha/index.html"`)
		}},
		{"h-entry tag", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), `<span class="p-category">go</span>`, `<span class="p-category">other</span>`)
		}},
		{"JSON-LD publication", func(t *testing.T, f fixture) {
			mutateLD(t, filepath.Join(f.deploy, "posts/alpha/index.html"), func(record map[string]any) { record["datePublished"] = "2024-02-01T09:00:00Z" })
		}},
		{"JSON-LD update", func(t *testing.T, f fixture) {
			mutateLD(t, filepath.Join(f.deploy, "posts/alpha/index.html"), func(record map[string]any) { record["dateModified"] = "2024-01-01T09:00:00Z" })
		}},
		{"JSON-LD summary", func(t *testing.T, f fixture) {
			mutateLD(t, filepath.Join(f.deploy, "posts/alpha/index.html"), func(record map[string]any) { record["description"] = "Changed" })
		}},
		{"JSON-LD tags", func(t *testing.T, f fixture) {
			mutateLD(t, filepath.Join(f.deploy, "posts/alpha/index.html"), func(record map[string]any) { record["keywords"] = []string{"other"} })
		}},
		{"RSS publication", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), "Mon, 01 Jan 2024 09:00:00 +0000", "Mon, 01 Jan 2024 10:00:00 +0000")
		}},
		{"RSS publication used as update", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), "Thu, 01 Feb 2024 09:00:00 +0000", "Mon, 01 Jan 2024 09:00:00 +0000")
		}},
		{"RSS duplicate GUID", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), `<guid isPermaLink="true">https://dario.cat/en/posts/beta/</guid>`, `<guid isPermaLink="true">https://dario.cat/posts/alpha/</guid>`)
		}},
		{"RSS link drift", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), `<link>https://dario.cat/posts/alpha/</link>`, `<link>https://dario.cat/posts/alpha/index.html</link>`)
		}},
		{"RSS summary escaping", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), "A &amp;amp; B &amp;lt;literal&amp;gt;.", "A &amp; B &lt;literal&gt;.")
		}},
		{"RSS category", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), "<category>go</category>", "<category>other</category>")
		}},
		{"RSS item language", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), `<item xml:lang="ca">`, `<item xml:lang="en">`)
		}},
		{"RSS extension namespace", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), "http://purl.org/rss/1.0/modules/content/", "https://invalid.example/")
		}},
		{"RSS full content", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "feed.xml"), "Public alpha body.", "Unexpected body.")
		}},
		{"RSS cap changed without artifact update", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.root, ".zas/config.yml"), "feed_limit: 2", "feed_limit: 1")
		}},
		{"Standard publication date", func(t *testing.T, f fixture) {
			mutatePlan(t, f, func(record map[string]any) { record["publishedAt"] = "2024-02-01T09:00:00Z" })
		}},
		{"Standard update date", func(t *testing.T, f fixture) {
			mutatePlan(t, f, func(record map[string]any) { record["updatedAt"] = "2024-01-01T09:00:00Z" })
		}},
		{"Standard canonical path", func(t *testing.T, f fixture) {
			mutatePlan(t, f, func(record map[string]any) { record["path"] = "/posts/alpha/index.html" })
		}},
		{"Standard summary", func(t *testing.T, f fixture) {
			mutatePlan(t, f, func(record map[string]any) { record["description"] = "Changed" })
		}},
		{"Standard language tag", func(t *testing.T, f fixture) {
			mutatePlan(t, f, func(record map[string]any) { record["tags"] = []string{"go", "writing", "lang:en"} })
		}},
		{"Standard full text", func(t *testing.T, f fixture) {
			mutatePlan(t, f, func(record map[string]any) { record["textContent"] = "Different text" })
		}},
		{"unapproved document plan", func(t *testing.T, f fixture) {
			mutateJSON(t, filepath.Join(f.root, ".tools/atproto/documents-allowlist.json"), func(data map[string]any) { data["documents"] = []any{} })
		}},
		{"reviewed date changed", func(t *testing.T, f fixture) {
			mutateJSON(t, filepath.Join(f.root, ".tools/atproto/documents-allowlist.json"), func(data map[string]any) {
				data["documents"].([]any)[0].(map[string]any)["publishedAt"] = "2024-01-02T09:00:00Z"
			})
		}},
		{"reviewed duplicate", func(t *testing.T, f fixture) {
			mutateJSON(t, filepath.Join(f.root, ".tools/atproto/documents-allowlist.json"), func(data map[string]any) {
				rows := data["documents"].([]any)
				data["documents"] = append(rows, rows[0])
			})
		}},
		{"partial approved", func(t *testing.T, f fixture) {
			mutateJSON(t, filepath.Join(f.root, ".tools/atproto/documents-allowlist.json"), func(data map[string]any) { data["documents"].([]any)[0].(map[string]any)["path"] = "/posts/partial/" })
		}},
		{"verified URI drift", func(t *testing.T, f fixture) {
			write(t, filepath.Join(f.root, ".zas/atproto.json"), `{"/posts/alpha/":"at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.document/3bbbbbbbbbbbb"}`)
		}},
		{"verified link missing", func(t *testing.T, f fixture) {
			write(t, filepath.Join(f.root, ".zas/atproto.json"), `{"/posts/alpha/":"at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.document/3aaaaaaaaaaaa"}`)
		}},
		{"unverified document link", func(t *testing.T, f fixture) {
			replace(t, filepath.Join(f.deploy, "posts/alpha/index.html"), "</head>", `<link rel="site.standard.document" href="at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.document/3aaaaaaaaaaaa"></head>`)
		}},
		{"duplicate JSON keys", func(t *testing.T, f fixture) { replace(t, f.plan, `"version": 1`, `"version": 0, "version": 1`) }},
		{"multiple JSON values", func(t *testing.T, f fixture) { write(t, filepath.Join(f.root, ".zas/atproto.json"), "{} {}") }},
		{"symlink artifact", func(t *testing.T, f fixture) {
			path := filepath.Join(f.deploy, "feed.xml")
			remove(t, path)
			if err := os.Symlink("en/feed.xml", path); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := fixtureSite(t)
			test.mutate(t, f)
			if _, err := Directory(f.root, f.deploy, f.plan); err == nil {
				t.Fatal("purposeful publishing drift was accepted")
			}
		})
	}
}

func remove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
