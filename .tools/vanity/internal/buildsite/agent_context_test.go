package buildsite

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var agentContextOutputs = []string{"llms.txt", "llms-full.txt", "en/llms.txt", "en/llms-full.txt", "es/llms.txt", "es/llms-full.txt"}

func agentContextFixture(t *testing.T) (stage, deploy string) {
	t.Helper()
	stage, deploy = t.TempDir(), t.TempDir()
	manifest, err := os.ReadFile("testdata/agent-context/manifest.yml")
	if err != nil {
		t.Fatal(err)
	}
	put(t, stage, ".zas/llms.yml", string(manifest))
	put(t, stage, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\n")
	for _, pair := range []struct{ source, language string }{{"about/index.md", "ca"}, {"en/posts/example/index.md", "en"}, {"es/posts/example/index.md", "es"}} {
		put(t, stage, pair.source, "<!--\nlanguage: "+pair.language+"\npublish: true\n-->\n# {{.Site.Title}}\nRAW_SOURCE_ONLY\n<embed src='raw-private-asset'>\n")
		put(t, deploy, pair.source, "# Published "+pair.language+"\n\nExact generated body, accented words: català, español.\n")
		htmlPath := strings.TrimSuffix(pair.source, ".md") + ".html"
		canonical := "https://dario.cat/" + strings.TrimSuffix(htmlPath, "index.html")
		put(t, deploy, htmlPath, fmt.Sprintf(`<html lang="%s"><head><link rel="canonical" href="%s"></head><body>Public page</body></html>`, pair.language, canonical))
		put(t, deploy, agentContextPrefix(pair.language)+"feed.xml", `<?xml version="1.0"?><rss version="2.0"><channel><title>Public RSS</title><link>https://dario.cat/</link><description>Feed</description></channel></rss>`)
	}
	body, err := os.ReadFile("testdata/agent-context/rendered-ca.md")
	if err != nil {
		t.Fatal(err)
	}
	put(t, deploy, "about/index.md", string(body))
	return stage, deploy
}

func TestAgentContextUsesCuratedRenderedTwinsDeterministically(t *testing.T) {
	stage, deploy := agentContextFixture(t)
	// Existing public CV JSON remains available, but no inventory/read pass
	// touches it. Unselected private and draft content also stays unexamined.
	put(t, stage, "CV/MAC.json", "invalid private JSON; do not parse")
	put(t, deploy, "CV/MAC.json", "invalid private JSON; do not parse")
	put(t, stage, "posts/unselected/index.md", "<!--\ndraft: true\n-->\nUNREVIEWED_BODY")
	put(t, deploy, "posts/unselected/index.md", "UNREVIEWED_BODY")
	if err := finalizeAgentContext(stage, deploy); err != nil {
		t.Fatal(err)
	}
	ca := string(readMarkdownDiscoveryFile(t, deploy, "llms.txt"))
	for _, expected := range []string{
		"# Dario Castañé\n\n> Escrits públics en català.\n",
		"\nPersonal site with public writing in three languages.\n",
		"## Sobre mi\n- [Sobre mi](https://dario.cat/about/index.md): Programari lliure, tecnologia i comunitat.",
		"[Català](https://dario.cat/llms.txt)", "[English](https://dario.cat/en/llms.txt)", "[Español](https://dario.cat/es/llms.txt)",
		"## Optional\n- [context complet](https://dario.cat/llms-full.txt)", "[RSS](https://dario.cat/feed.xml)",
	} {
		if !strings.Contains(ca, expected) {
			t.Errorf("missing summary structure %q in %s", expected, ca)
		}
	}
	twin := readMarkdownDiscoveryFile(t, deploy, "about/index.md")
	full := readMarkdownDiscoveryFile(t, deploy, "llms-full.txt")
	if !bytes.Contains(full, twin) || !bytes.Contains(full, []byte("Language: ca\nSource: https://dario.cat/about/index.md\nCanonical: https://dario.cat/about/\n")) {
		t.Fatalf("full context changed generated body or lacks provenance: %s", full)
	}
	for _, file := range agentContextOutputs {
		body := readMarkdownDiscoveryFile(t, deploy, file)
		for _, forbidden := range []string{"RAW_SOURCE_ONLY", "UNREVIEWED_BODY", "invalid private JSON", "raw-private-asset", "{{"} {
			if bytes.Contains(body, []byte(forbidden)) {
				t.Errorf("%s leaked %s", file, forbidden)
			}
		}
	}
	for _, locale := range []string{"en", "es"} {
		summary := string(readMarkdownDiscoveryFile(t, deploy, locale+"/llms.txt"))
		if !strings.Contains(summary, "https://dario.cat/"+locale+"/posts/example/index.md") || !strings.Contains(summary, "https://dario.cat/"+locale+"/feed.xml") || !strings.Contains(summary, "https://dario.cat/"+locale+"/llms-full.txt") {
			t.Errorf("%s summary points to another language's artifacts: %s", locale, summary)
		}
		if bytes.Contains(readMarkdownDiscoveryFile(t, deploy, locale+"/llms-full.txt"), twin) {
			t.Errorf("%s full context silently includes Catalan body", locale)
		}
	}
	before := make(map[string][]byte)
	for _, file := range agentContextOutputs {
		before[file] = readMarkdownDiscoveryFile(t, deploy, file)
	}
	if err := finalizeAgentContext(stage, deploy); err != nil {
		t.Fatal(err)
	}
	for _, file := range agentContextOutputs {
		if !bytes.Equal(before[file], readMarkdownDiscoveryFile(t, deploy, file)) {
			t.Errorf("%s changed on repeated assembly", file)
		}
	}
}

func TestAgentContextRejectsInvalidSelectionsBeforeAnyWrite(t *testing.T) {
	cases := []struct {
		name, message string
		mutate        func(*testing.T, string, string)
	}{
		{"missing source", "no such file", func(t *testing.T, stage, deploy string) { removeAgentContextFile(t, stage, "about/index.md") }},
		{"missing twin", "Markdown twin", func(t *testing.T, stage, deploy string) { removeAgentContextFile(t, deploy, "about/index.md") }},
		{"missing HTML", "HTML", func(t *testing.T, stage, deploy string) { removeAgentContextFile(t, deploy, "about/index.html") }},
		{"empty twin", "empty", func(t *testing.T, stage, deploy string) { put(t, deploy, "about/index.md", " \n") }},
		{"unrendered twin", "unrendered", func(t *testing.T, stage, deploy string) { put(t, deploy, "about/index.md", "# {{.Title}}") }},
		{"HTML fallback twin", "HTML document", func(t *testing.T, stage, deploy string) {
			put(t, deploy, "about/index.md", "<!DOCTYPE html><html><head></head><body>Homepage fallback</body></html>")
		}},
		{"language mismatch", "language", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, deploy, "about/index.html", `lang="ca"`, `lang="en"`)
		}},
		{"wrong canonical", "canonical", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, deploy, "about/index.html", "https://dario.cat/about/", "https://other.example/about/")
		}},
		{"missing feed", "feed", func(t *testing.T, stage, deploy string) { removeAgentContextFile(t, deploy, "es/feed.xml") }},
		{"feed HTML fallback", "RSS", func(t *testing.T, stage, deploy string) {
			put(t, deploy, "es/feed.xml", "<html><body>Not a feed</body></html>")
		}},
		{"unpublished source", "unpublished", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!--\npublish: false\n-->\nPRIVATE BODY")
			removeAgentContextFile(t, deploy, "about/index.md")
		}},
		{"draft source", "draft", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!--\ndraft: true\n-->\nDRAFT BODY")
			removeAgentContextFile(t, deploy, "about/index.md")
		}},
		{"private source", "private", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!--\nprivate: true\n-->\nPRIVATE BODY")
		}},
		{"inline private metadata", "unpublished", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!-- publish: false -->\nPRIVATE BODY")
		}},
		{"doctype unpublished metadata with stale twin", "unpublished", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!DOCTYPE html>\n<!--\npublish: false\n-->\nPRIVATE BODY")
		}},
		{"inline doctype draft metadata with stale twin", "draft", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!doctype html><!-- draft: true -->\nDRAFT BODY")
		}},
		{"multiline doctype private metadata with stale twin", "private", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!DoCtYpE\nhtml>\n<!-- private: true -->\nPRIVATE BODY")
		}},
		{"unclosed doctype", "unterminated source doctype", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!DOCTYPE html\n<!-- private: true -->")
		}},
		{"Markdown opt-out with stale twin", "Markdown opt-out", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!--\npublish_markdown: false\n-->\nHTML-only page")
		}},
		{"directory unpublished with stale twin", "directory metadata", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/.zas.yml", "publish: false\n")
		}},
		{"directory draft with stale twin", "draft", func(t *testing.T, stage, deploy string) {
			put(t, stage, "en/.zas.yml", "draft: true\nlanguage: en\n")
		}},
		{"directory private with stale twin", "private", func(t *testing.T, stage, deploy string) {
			put(t, stage, "es/posts/.zas.yml", "private: true\nlanguage: es\n")
		}},
		{"directory malformed metadata", "cannot unmarshal", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/.zas.yml", "publish: secret-not-a-bool\n")
		}},
		{"directory language with stale twin", "source language", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!-- title: Public page -->\nBODY")
			put(t, stage, "about/.zas.yml", "language: en\n")
		}},
		{"source language mismatch", "source language", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!--\nlanguage: en\n-->\n")
		}},
		{"invalid metadata", "cannot unmarshal", func(t *testing.T, stage, deploy string) {
			put(t, stage, "about/index.md", "<!--\npublish: not-a-bool\n-->")
		}},
		{"unterminated metadata", "unterminated", func(t *testing.T, stage, deploy string) { put(t, stage, "about/index.md", "<!--\npublish: false\n") }},
		{"disabled Markdown", "publish_markdown", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/config.yml", "publish_markdown: true", "publish_markdown: false")
		}},
		{"unsafe base", "site.baseurl", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/config.yml", "https://dario.cat", "https://user:password@dario.cat")
		}},
		{"version", "version", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "version: 1", "version: 2")
		}},
		{"wrong locale order", "ca, en, es", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "language: ca", "language: es")
		}},
		{"unknown manifest field", "field mystery", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "version: 1", "version: 1\nmystery: true")
		}},
		{"extra YAML document", "one YAML", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "version: 1", "---\nversion: 1")
			in := readMarkdownDiscoveryFile(t, stage, ".zas/llms.yml")
			put(t, stage, ".zas/llms.yml", string(in)+"\n---\n{}\n")
		}},
		{"duplicate source", "duplicate", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: en/posts/example/index.md", "source: about/index.md")
		}},
		{"traversal", "unsafe", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: about/index.md", "source: ../private.md")
		}},
		{"CV exclusion", "excluded", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: about/index.md", "source: CV/MAC.md")
		}},
		{"private path exclusion", "excluded", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: about/index.md", "source: posts/private/index.md")
		}},
		{"operations exclusion", "unsafe", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: about/index.md", "source: docs/site-plan/tasks/T20.md")
		}},
		{"unreviewed inventory", "unreviewed", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: about/index.md", "source: docs/public.md")
		}},
		{"wrong extension", "unsafe", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "source: about/index.md", "source: about/index.html")
		}},
		{"forged summary link", "plain title", func(t *testing.T, stage, deploy string) {
			replaceAgentContextFile(t, stage, ".zas/llms.yml", "summary: Escrits públics en català.", "summary: '[Private](https://dario.cat/CV/MAC.json)'")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stage, deploy := agentContextFixture(t)
			put(t, deploy, "llms.txt", "previous valid context\n")
			tc.mutate(t, stage, deploy)
			err := finalizeAgentContext(stage, deploy)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error = %v, expected %q", err, tc.message)
			}
			if string(readMarkdownDiscoveryFile(t, deploy, "llms.txt")) != "previous valid context\n" {
				t.Fatal("validation failure changed previous artifact")
			}
			for _, file := range agentContextOutputs[1:] {
				if _, err := os.Lstat(filepath.Join(deploy, filepath.FromSlash(file))); !os.IsNotExist(err) {
					t.Fatalf("validation failure wrote %s: %v", file, err)
				}
			}
		})
	}
}

func TestAgentContextRejectsSymlinksAndDirectoriesBeforeReadingOrWriting(t *testing.T) {
	for _, selected := range []string{"source", "manifest", "config", "directory metadata", "twin", "html", "feed", "output", "source parent", "deploy parent"} {
		t.Run(selected, func(t *testing.T) {
			stage, deploy := agentContextFixture(t)
			root, relative := deploy, "about/index.md"
			switch selected {
			case "source":
				root = stage
			case "manifest":
				root, relative = stage, ".zas/llms.yml"
			case "config":
				root, relative = stage, ".zas/config.yml"
			case "directory metadata":
				root, relative = stage, "about/.zas.yml"
			case "html":
				relative = "about/index.html"
			case "feed":
				relative = "feed.xml"
			case "output":
				relative = "llms-full.txt"
			case "source parent":
				root, relative = stage, "about"
			case "deploy parent":
				relative = "about"
			}
			target := filepath.Join(root, filepath.FromSlash(relative))
			if err := os.RemoveAll(target); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(t.TempDir(), "unreadable-secret")
			if err := os.Symlink(outside, target); err != nil {
				t.Skip(err)
			}
			if err := finalizeAgentContext(stage, deploy); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("symlink error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(deploy, "llms.txt")); !os.IsNotExist(err) {
				t.Fatalf("context written before symlink rejection: %v", err)
			}
		})
	}
	stage, deploy := agentContextFixture(t)
	removeAgentContextFile(t, deploy, "es/posts/example/index.md")
	if err := os.Mkdir(filepath.Join(deploy, "es/posts/example/index.md"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := finalizeAgentContext(stage, deploy); err == nil || !strings.Contains(err.Error(), "not regular") {
		t.Fatalf("nonregular twin error = %v", err)
	}
}

func TestAgentContextAllowsDoctypeAndNearestDirectoryLanguage(t *testing.T) {
	stage, deploy := agentContextFixture(t)
	put(t, stage, "about/index.md", " \n<!DoCtYpE html>\n<!--\npublish: true\nlanguage: ca\n-->\nRAW_SOURCE_ONLY")
	// Page language overrides the nearest directory's language, as in Zas.
	put(t, stage, "about/.zas.yml", "language: en\n")
	put(t, stage, "en/.zas.yml", "language: en\n")
	put(t, stage, "es/.zas.yml", "language: es\n")
	put(t, stage, "en/posts/example/index.md", "<!-- publish: true -->\nRAW_SOURCE_ONLY")
	put(t, stage, "es/posts/example/index.md", "<!doctype html><!-- publish: true -->\nRAW_SOURCE_ONLY")
	if err := finalizeAgentContext(stage, deploy); err != nil {
		t.Fatalf("valid doctype or language inheritance rejected: %v", err)
	}
	if !bytes.Contains(readMarkdownDiscoveryFile(t, deploy, "en/llms-full.txt"), []byte("Language: en")) {
		t.Fatal("inherited language lost")
	}
}

func TestAgentContextMissingManifestDoesNotChangeLegacyBuildRequirements(t *testing.T) {
	stage := t.TempDir()
	if err := finalizeAgentContext(stage, filepath.Join(stage, "missing-deploy")); err != nil {
		t.Fatalf("absent manifest activated feature: %v", err)
	}
}

func replaceAgentContextFile(t *testing.T, root, relative, old, replacement string) {
	t.Helper()
	body := readMarkdownDiscoveryFile(t, root, relative)
	if !bytes.Contains(body, []byte(old)) {
		t.Fatalf("fixture %s lacks %q", relative, old)
	}
	put(t, root, relative, string(bytes.Replace(body, []byte(old), []byte(replacement), 1)))
}

func removeAgentContextFile(t *testing.T, root, relative string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
		t.Fatal(err)
	}
}
