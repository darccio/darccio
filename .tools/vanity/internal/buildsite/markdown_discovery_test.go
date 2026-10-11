package buildsite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownDiscoveryUsesPublishedTwinInventory(t *testing.T) {
	deploy := t.TempDir()
	body := "<body data-value='a & b'><h1>Title</h1><pre>{{.Title}}</pre><script>const sample = '</head>';</script></body></html>"
	original := "<!DOCTYPE html>\n<html><head><title>Title &amp; other</title><script>const sample = '</head>';</script></head>" + body
	for _, file := range []string{"posts/index.html", "en/posts/example/index.html", "contact.html", "section/INDEX.html"} {
		put(t, deploy, file, original)
		put(t, deploy, strings.TrimSuffix(file, ".html")+".md", "# Published Markdown\n")
	}
	for _, file := range []string{"index.html", "draft/index.html", "html-only.html"} {
		put(t, deploy, file, original)
	}
	vanity := `<meta name="go-import" content="dario.cat/xdr git https://github.com/darccio/xdr">`
	put(t, deploy, "xdr/index.html", vanity)
	put(t, deploy, "orphan.md", "A Markdown asset without an HTML page.\n")
	if err := os.Chmod(filepath.Join(deploy, "contact.html"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := addMarkdownDiscovery(deploy, "https://dario.cat/"); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"posts/index.html", "en/posts/example/index.html", "contact.html", "section/INDEX.html"} {
		got := readMarkdownDiscoveryFile(t, deploy, file)
		expected := "https://dario.cat/" + strings.TrimSuffix(file, ".html") + ".md"
		link := "\n<link rel=\"alternate\" type=\"text/markdown\" href=\"" + expected + "\">\n"
		if strings.Count(string(got), link) != 1 {
			t.Errorf("%s does not advertise its exact published twin: %s", file, got)
		}
		if string(bytes.Replace(got, []byte(link), nil, 1)) != original {
			t.Errorf("%s changed bytes outside the inserted link", file)
		}
		if !bytes.HasSuffix(got, []byte(body)) {
			t.Errorf("%s changed the body bytes", file)
		}
	}
	for _, file := range []string{"index.html", "draft/index.html", "html-only.html"} {
		if string(readMarkdownDiscoveryFile(t, deploy, file)) != original {
			t.Errorf("%s without a published twin changed", file)
		}
	}
	if string(readMarkdownDiscoveryFile(t, deploy, "xdr/index.html")) != vanity {
		t.Fatal("raw head-only vanity fragment changed")
	}
	info, err := os.Stat(filepath.Join(deploy, "contact.html"))
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("HTML file mode changed: %v", err)
	}
	before := readMarkdownDiscoveryFile(t, deploy, "contact.html")
	if err := addMarkdownDiscovery(deploy, "https://dario.cat"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readMarkdownDiscoveryFile(t, deploy, "contact.html"), before) {
		t.Fatal("repeated discovery added or changed an existing link")
	}
	if err := auditOutput(deploy); err != nil {
		t.Fatalf("temporary ownership/output artifacts leaked: %v", err)
	}
}

func TestMarkdownDiscoveryEscapesURLsAndSupportsBasePath(t *testing.T) {
	deploy := t.TempDir()
	put(t, deploy, "quoted & path.html", "<html><head></head><body>Original</body></html>")
	put(t, deploy, "quoted & path.md", "# Markdown\n")
	if err := addMarkdownDiscovery(deploy, "https://example.com/blog"); err != nil {
		t.Fatal(err)
	}
	got := readMarkdownDiscoveryFile(t, deploy, "quoted & path.html")
	if !bytes.Contains(got, []byte(`href="https://example.com/blog/quoted%20&amp;%20path.md"`)) {
		t.Fatalf("incorrect URL encoding/escaping: %s", got)
	}
	if err := addMarkdownDiscovery(deploy, "https://example.com/blog/"); err != nil {
		t.Fatal(err)
	}
}

func TestMarkdownDiscoveryRejectsInvalidHeadsBeforeMutation(t *testing.T) {
	link := `<link rel="alternate" type="text/markdown" href="https://dario.cat/bad.md">`
	cases := map[string]string{
		"missing head":             "<html><body>No head</body></html>",
		"unclosed head":            "<html><head><title>Title</title><body>Body</body></html>",
		"truncated head":           "<html><head><title>Title</title>",
		"duplicate head":           "<html><head></head><head></head><body>Body</body></html>",
		"body element in head":     "<html><head><div>Body</div></head><body>Body</body></html>",
		"raw body element in head": "<html><head><textarea>Body</textarea></head><body>Body</body></html>",
		"body element in noscript": "<html><head><noscript><div>Body</div></noscript></head><body>Body</body></html>",
		"duplicate links":          "<html><head>" + link + link + "</head><body>Body</body></html>",
		"relative URL":             `<html><head><link rel="alternate" type="text/markdown" href="/bad.md"></head><body>Body</body></html>`,
		"different local twin":     `<html><head><link rel="alternate" type="text/markdown" href="https://dario.cat/good.md"></head><body>Body</body></html>`,
		"external URL":             `<html><head><link rel="alternate" type="text/markdown" href="https://other.example/bad.md"></head><body>Body</body></html>`,
		"wrong relation":           `<html><head><link rel="canonical" type="text/markdown" href="https://dario.cat/bad.md"></head><body>Body</body></html>`,
		"duplicate attributes":     `<html><head><link rel="alternate" type="text/markdown" type="text/plain" href="https://dario.cat/bad.md"></head><body>Body</body></html>`,
		"duplicate type reversed":  `<html><head><link rel="alternate" type="text/plain" TYPE="text/markdown" href="https://dario.cat/bad.md"></head><body>Body</body></html>`,
		"duplicate href":           `<html><head><link rel='alternate' type=text/markdown href='https://dario.cat/bad.md' HREF='https://elsewhere.example/page.md'></head><body>Body</body></html>`,
		"outside head":             "<html><head></head><body>" + link + "</body></html>",
		"inside template":          "<html><head><template>" + link + "</template></head><body>Body</body></html>",
		"inside noscript":          "<html><head><noscript>" + link + "</noscript></head><body>Body</body></html>",
		"invalid Markdown type":    `<html><head><link rel="alternate" type="text/markdown; garbage" href="https://dario.cat/bad.md"></head><body>Body</body></html>`,
		"self-closing head":        "<html><head/><body>Body</body></html>",
		"closing head unopened":    "<html></head><body>Body</body></html>",
		"non-metadata head text":   "<html><head>Unexpected text</head><body>Body</body></html>",
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			deploy := t.TempDir()
			good := "<html><head></head><body>Unchanged</body></html>"
			put(t, deploy, "good.html", good)
			put(t, deploy, "good.md", "# Good\n")
			put(t, deploy, "bad.html", bad)
			put(t, deploy, "bad.md", "# Bad\n")
			if err := addMarkdownDiscovery(deploy, "https://dario.cat"); err == nil {
				t.Fatal("invalid discovery head was accepted")
			}
			if string(readMarkdownDiscoveryFile(t, deploy, "good.html")) != good || string(readMarkdownDiscoveryFile(t, deploy, "bad.html")) != bad {
				t.Fatal("validation failure mutated HTML")
			}
		})
	}
}

func TestMarkdownDiscoveryRejectsMissingTwinAdvertisement(t *testing.T) {
	deploy := t.TempDir()
	original := `<html><head><link rel="alternate" type="text/markdown" href="https://dario.cat/draft/index.md"></head><body>Live draft</body></html>`
	put(t, deploy, "draft/index.html", original)
	if err := addMarkdownDiscovery(deploy, "https://dario.cat"); err == nil || !strings.Contains(err.Error(), "no published twin") {
		t.Fatalf("missing twin error = %v", err)
	}
	if string(readMarkdownDiscoveryFile(t, deploy, "draft/index.html")) != original {
		t.Fatal("missing-twin validation changed HTML")
	}
}

func TestMarkdownDiscoveryValidExistingLinkIsUnchanged(t *testing.T) {
	deploy := t.TempDir()
	original := `<html><head><noscript><link rel="stylesheet" href="/static/style.css"></noscript><link REL='alternate' TYPE='TEXT/MARKDOWN; charset=utf-8' HREF='https://dario.cat/page.md' data-example='type="not an attribute"'></head><body>Unchanged {{.Title}}</body></html>`
	put(t, deploy, "page.html", original)
	put(t, deploy, "page.md", "# Markdown\n")
	if err := addMarkdownDiscovery(deploy, "https://dario.cat"); err != nil {
		t.Fatal(err)
	}
	if string(readMarkdownDiscoveryFile(t, deploy, "page.html")) != original {
		t.Fatal("valid existing link was rewritten")
	}
}

func TestMarkdownDiscoveryRejectsUnsafeBaseBeforeMutation(t *testing.T) {
	for _, base := range []string{"", "/", "//dario.cat", "https://user:secret@dario.cat", "https://dario.cat?query=1", "https://dario.cat#fragment", "file:///tmp", "https://dario.cat/blog/../", "https://dario.cat/blog//child"} {
		t.Run(base, func(t *testing.T) {
			deploy := t.TempDir()
			original := "<html><head></head><body>Unchanged</body></html>"
			put(t, deploy, "page.html", original)
			put(t, deploy, "page.md", "# Markdown\n")
			if err := addMarkdownDiscovery(deploy, base); err == nil {
				t.Fatal("unsafe base URL was accepted")
			}
			if string(readMarkdownDiscoveryFile(t, deploy, "page.html")) != original {
				t.Fatal("invalid base URL mutated HTML")
			}
		})
	}
}

func TestMarkdownDiscoveryAuditsSymlinksBeforeMutation(t *testing.T) {
	deploy := t.TempDir()
	original := "<html><head></head><body>Unchanged</body></html>"
	put(t, deploy, "good.html", original)
	put(t, deploy, "good.md", "# Markdown\n")
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("Outside\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(deploy, "bad.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := addMarkdownDiscovery(deploy, "https://dario.cat"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("deploy symlink audit error = %v", err)
	}
	if string(readMarkdownDiscoveryFile(t, deploy, "good.html")) != original {
		t.Fatal("symlink audit failure changed HTML")
	}
}

func TestMarkdownDiscoveryConfigDisabledRequiresNoBaseOrDeploy(t *testing.T) {
	stage := t.TempDir()
	put(t, stage, ".zas/config.yml", "site:\n  title: Legacy site\n")
	if err := finalizeMarkdownDiscovery(stage, filepath.Join(stage, "absent-deploy")); err != nil {
		t.Fatalf("disabled discovery changed legacy config requirements: %v", err)
	}
}

func TestMarkdownDiscoveryBuildUsesConfigAndPublishedInventory(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	script := generator(t, ordinaryOutput+`
printf '# Published\n' > .zas/deploy/index.md
mkdir -p .zas/deploy/draft
printf '<html><head></head><body>Live draft</body></html>' > .zas/deploy/draft/index.html
`)
	if err := Build(context.Background(), Options{Root: root, Zas: script}); err != nil {
		t.Fatal(err)
	}
	deploy := filepath.Join(root, ".zas", "deploy")
	if !bytes.Contains(readMarkdownDiscoveryFile(t, deploy, "index.html"), []byte(`href="https://dario.cat/index.md"`)) {
		t.Fatal("enabled assembly did not advertise the actual root Markdown twin")
	}
	if bytes.Contains(readMarkdownDiscoveryFile(t, deploy, "draft/index.html"), []byte("text/markdown")) {
		t.Fatal("live draft without a published twin was advertised")
	}
	before, err := os.ReadFile(filepath.Join(root, "xdr/index.html"))
	if err != nil || !bytes.Equal(before, readMarkdownDiscoveryFile(t, deploy, "xdr/index.html")) {
		t.Fatalf("raw vanity source bytes changed: %v", err)
	}
}

func TestMarkdownDiscoveryBuildFailurePreservesPreviousDeployment(t *testing.T) {
	for _, output := range []string{
		`printf '# Markdown\n' > .zas/deploy/index.md
printf '<html><body>No head</body></html>' > .zas/deploy/index.html
`,
		`printf '<html><head><link rel="alternate" type="text/markdown" href="https://dario.cat/index.md"></head><body>Missing twin</body></html>' > .zas/deploy/index.html
`,
	} {
		t.Run(output, func(t *testing.T) {
			root := sourceTree(t)
			put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
			err := Build(context.Background(), Options{Root: root, Zas: generator(t, ordinaryOutput+output)})
			if err == nil || !strings.Contains(err.Error(), "assembled Markdown discovery rejected") {
				t.Fatalf("expected discovery guard failure, got %v", err)
			}
			deploy := filepath.Join(root, ".zas", "deploy")
			if string(readMarkdownDiscoveryFile(t, deploy, "previous.txt")) != "last successful deployment" {
				t.Fatal("previous successful deployment changed")
			}
			if _, err := os.Stat(filepath.Join(deploy, "index.html")); !os.IsNotExist(err) {
				t.Fatal("rejected partial output reached previous deployment")
			}
		})
	}
}

func readMarkdownDiscoveryFile(t *testing.T, deploy, rel string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(deploy, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func putMarkdownContextArtifacts(t *testing.T, deploy string) {
	t.Helper()
	for _, prefix := range []string{"", "en/", "es/"} {
		for _, file := range []string{"llms.txt", "llms-full.txt"} {
			put(t, deploy, prefix+file, "# Dario Castañé\n\nGenerated reviewed context.\n")
		}
	}
}

func markdownContextTestLinks(language string) string {
	var links strings.Builder
	for i, locale := range []string{"ca", "en", "es"} {
		prefix, rel := "", "alternate"
		if locale != "ca" {
			prefix = locale + "/"
		}
		if locale == language {
			rel += " describedby"
		}
		title := []string{"Català", "English", "Español"}[i]
		fmt.Fprintf(&links, `<link rel="%s" type="text/markdown" title="LLM Context (%s)" hreflang="%s" href="https://dario.cat/%sllms.txt">`, rel, title, locale, prefix)
	}
	prefix := ""
	if language != "ca" {
		prefix = language + "/"
	}
	fmt.Fprintf(&links, `<link rel="alternate" type="text/markdown" title="Full LLM Context" hreflang="%s" href="https://dario.cat/%sllms-full.txt">`, language, prefix)
	return links.String()
}

func markdownContextTestPage(language, links string) string {
	return `<html lang="` + language + `"><head><title>Context fixture</title>` + links + `</head><body>Unchanged</body></html>`
}

func TestMarkdownDiscoveryContextsRemainSeparateFromPublishedTwins(t *testing.T) {
	deploy := t.TempDir()
	putMarkdownContextArtifacts(t, deploy)
	for _, language := range []string{"ca", "en", "es"} {
		file := language + "/page.html"
		put(t, deploy, file, markdownContextTestPage(language, markdownContextTestLinks(language)))
		put(t, deploy, language+"/page.md", "# Published twin\n")
	}
	noTwin := markdownContextTestPage("ca", markdownContextTestLinks("ca"))
	put(t, deploy, "draft/index.html", noTwin)
	vanity := `<meta name="go-import" content="dario.cat/xdr git https://github.com/darccio/xdr">`
	put(t, deploy, "xdr/index.html", vanity)
	if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat", true); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"ca", "en", "es"} {
		file := language + "/page.html"
		got := readMarkdownDiscoveryFile(t, deploy, file)
		twin := "https://dario.cat/" + language + "/page.md"
		if strings.Count(string(got), `type="text/markdown"`) != 5 || strings.Count(string(got), `href="`+twin+`"`) != 1 {
			t.Fatalf("%s: contexts changed twin advertisement count: %s", file, got)
		}
		base, err := markdownDiscoveryBase("https://dario.cat")
		if err != nil {
			t.Fatal(err)
		}
		artifacts, err := markdownContextArtifacts(deploy, base)
		if err != nil {
			t.Fatal(err)
		}
		if _, linked, err := markdownDiscoveryHeadWithContexts(got, twin, artifacts); err != nil || !linked {
			t.Fatalf("%s: independently classify one published twin: linked=%t err=%v", file, linked, err)
		}
		before := bytes.Clone(got)
		if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat/", true); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, readMarkdownDiscoveryFile(t, deploy, file)) {
			t.Fatal("repeat context discovery changed existing links")
		}
	}
	if string(readMarkdownDiscoveryFile(t, deploy, "draft/index.html")) != noTwin || string(readMarkdownDiscoveryFile(t, deploy, "xdr/index.html")) != vanity {
		t.Fatal("context-only page or raw vanity fragment changed")
	}
}

func TestMarkdownDiscoveryLegacyModeRejectsContextLinks(t *testing.T) {
	deploy := t.TempDir()
	putMarkdownContextArtifacts(t, deploy)
	original := markdownContextTestPage("ca", markdownContextTestLinks("ca"))
	put(t, deploy, "index.html", original)
	put(t, deploy, "index.md", "# Published\n")
	if err := addMarkdownDiscovery(deploy, "https://dario.cat"); err == nil {
		t.Fatal("context-looking titles and six .txt files bypassed legacy twin validation")
	}
	if string(readMarkdownDiscoveryFile(t, deploy, "index.html")) != original {
		t.Fatal("legacy rejection changed HTML")
	}
}

func TestMarkdownDiscoveryContextExceptionsKeepTwinValidationStrict(t *testing.T) {
	twin := `<link rel="alternate" type="text/markdown" href="https://dario.cat/index.md">`
	for _, mode := range []string{"missing twin", "duplicate twin"} {
		t.Run(mode, func(t *testing.T) {
			deploy := t.TempDir()
			putMarkdownContextArtifacts(t, deploy)
			links := markdownContextTestLinks("ca") + twin
			if mode == "duplicate twin" {
				put(t, deploy, "index.md", "# Published twin\n")
				links += twin
			}
			original := markdownContextTestPage("ca", links)
			put(t, deploy, "index.html", original)
			if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat", true); err == nil {
				t.Fatal("registered context links weakened published-twin validation")
			}
			if string(readMarkdownDiscoveryFile(t, deploy, "index.html")) != original {
				t.Fatal("failed twin validation changed HTML")
			}
		})
	}
}

func TestMarkdownDiscoveryContextPolicyRequiresRegularManifest(t *testing.T) {
	for _, mode := range []string{"absent", "regular", "directory", "symlink", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			stage, deploy := t.TempDir(), t.TempDir()
			put(t, stage, ".zas/config.yml", "site:\n  publish_markdown: true\n  baseurl: https://dario.cat\n")
			putMarkdownContextArtifacts(t, deploy)
			put(t, deploy, "index.html", markdownContextTestPage("ca", markdownContextTestLinks("ca")))
			put(t, deploy, "index.md", "# Published\n")
			manifest := filepath.Join(stage, ".zas", "llms.yml")
			switch mode {
			case "regular", "disabled":
				put(t, stage, ".zas/llms.yml", "version: 1\n")
				if mode == "disabled" {
					put(t, stage, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n")
				}
			case "directory":
				if err := os.Mkdir(manifest, 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				outside := filepath.Join(t.TempDir(), "llms.yml")
				if err := os.WriteFile(outside, []byte("version: 1\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, manifest); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			err := finalizeMarkdownDiscovery(stage, deploy)
			if (err == nil) != (mode == "regular") {
				t.Fatalf("manifest %s: error = %v", mode, err)
			}
		})
	}
}

func TestMarkdownDiscoveryRejectsContextLinkForgeriesBeforeMutation(t *testing.T) {
	links := markdownContextTestLinks("en")
	full := `<link rel="alternate" type="text/markdown" title="Full LLM Context" hreflang="en" href="https://dario.cat/en/llms-full.txt">`
	summary := `<link rel="alternate describedby" type="text/markdown" title="LLM Context (English)" hreflang="en" href="https://dario.cat/en/llms.txt">`
	cases := map[string]string{
		"missing summary":      strings.Replace(links, summary, "", 1),
		"missing full":         strings.Replace(links, full, "", 1),
		"duplicate summary":    links + summary,
		"duplicate full":       links + full,
		"wrong full locale":    strings.Replace(links, full, strings.ReplaceAll(strings.ReplaceAll(full, `hreflang="en"`, `hreflang="ca"`), "/en/llms-full.txt", "/llms-full.txt"), 1),
		"two full locales":     links + strings.ReplaceAll(strings.ReplaceAll(full, `hreflang="en"`, `hreflang="ca"`), "/en/llms-full.txt", "/llms-full.txt"),
		"wrong language":       strings.Replace(links, `hreflang="en"`, `hreflang="ca"`, 1),
		"wrong title":          strings.Replace(links, "LLM Context (English)", "LLM Context (Català)", 1),
		"missing alternate":    strings.Replace(links, "alternate describedby", "describedby", 1),
		"missing describedby":  strings.Replace(links, "alternate describedby", "alternate", 1),
		"foreign describedby":  strings.Replace(links, `rel="alternate"`, `rel="alternate describedby"`, 1),
		"extra relation":       strings.Replace(links, "alternate describedby", "alternate describedby author", 1),
		"duplicate relation":   strings.Replace(links, "alternate describedby", "alternate describedby alternate", 1),
		"wrong type":           strings.Replace(links, `type="text/markdown"`, `type="text/plain"`, 1),
		"missing type":         strings.Replace(links, `type="text/markdown"`, "", 1),
		"invalid type":         strings.Replace(links, `type="text/markdown"`, `type="text/markdown; garbage"`, 1),
		"duplicate href":       strings.Replace(links, `<link `, `<link href="https://other.example/llms.txt" `, 1),
		"inactive template":    strings.Replace(links, summary, "<template>"+summary+"</template>", 1),
		"inactive noscript":    strings.Replace(links, summary, "<noscript>"+summary+"</noscript>", 1),
		"inactive media":       strings.Replace(links, `<link `, `<link media="print" `, 1),
		"disabled link":        strings.Replace(links, `<link `, `<link disabled `, 1),
		"relative href":        strings.Replace(links, "https://dario.cat/llms.txt", "/llms.txt", 1),
		"foreign href":         strings.Replace(links, "https://dario.cat/llms.txt", "https://other.example/llms.txt", 1),
		"suffix spoof":         strings.Replace(links, "https://dario.cat/llms.txt", "https://dario.cat/spoof-llms.txt", 1),
		"query spoof":          strings.Replace(links, "https://dario.cat/llms.txt", "https://dario.cat/llms.txt?foo=1", 1),
		"fragment spoof":       strings.Replace(links, "https://dario.cat/llms.txt", "https://dario.cat/llms.txt#foo", 1),
		"extra markdown link":  links + `<link rel="alternate" type="text/markdown" href="https://dario.cat/unreviewed.txt" title="LLM Context (English)">`,
		"spoof twin exception": links + `<link rel="alternate" type="text/markdown" href="https://dario.cat/other.md" title="Full LLM Context">`,
	}
	for name, changed := range cases {
		t.Run(name, func(t *testing.T) {
			deploy := t.TempDir()
			putMarkdownContextArtifacts(t, deploy)
			good := markdownContextTestPage("ca", markdownContextTestLinks("ca"))
			bad := markdownContextTestPage("en", changed)
			put(t, deploy, "good.html", good)
			put(t, deploy, "good.md", "# Good\n")
			put(t, deploy, "bad.html", bad)
			put(t, deploy, "bad.md", "# Bad\n")
			if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat", true); err == nil {
				t.Fatal("invalid agent-context discovery was accepted")
			}
			if string(readMarkdownDiscoveryFile(t, deploy, "good.html")) != good || string(readMarkdownDiscoveryFile(t, deploy, "bad.html")) != bad {
				t.Fatal("context validation failure mutated HTML")
			}
		})
	}
}

func TestMarkdownDiscoveryRejectsMisplacedContextsAndBadLocale(t *testing.T) {
	links := markdownContextTestLinks("ca")
	for name, original := range map[string]string{
		"outside head":    `<html lang="ca"><head></head><body>` + links + `</body></html>`,
		"no html lang":    `<html><head>` + links + `</head><body>Body</body></html>`,
		"wrong html lang": markdownContextTestPage("es", links),
		"duplicate html":  `<html lang="ca">` + markdownContextTestPage("ca", links),
		"duplicate lang":  strings.Replace(markdownContextTestPage("ca", links), `lang="ca"`, `lang="ca" LANG="es"`, 1),
		"raw vanity link": `<meta name="go-import" content="dario.cat/xdr git https://github.com/darccio/xdr">` + links,
	} {
		t.Run(name, func(t *testing.T) {
			deploy := t.TempDir()
			putMarkdownContextArtifacts(t, deploy)
			file := "index.html"
			if name == "raw vanity link" {
				file = "xdr/index.html"
			}
			put(t, deploy, file, original)
			if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat", true); err == nil {
				t.Fatal("misplaced or wrongly localized contexts were accepted")
			}
		})
	}
}

func TestMarkdownDiscoveryRejectsMissingOrBogusContextArtifacts(t *testing.T) {
	for _, mode := range []string{"missing", "directory", "empty", "HTML", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			deploy := t.TempDir()
			putMarkdownContextArtifacts(t, deploy)
			original := markdownContextTestPage("ca", markdownContextTestLinks("ca"))
			put(t, deploy, "index.html", original)
			put(t, deploy, "index.md", "# Published\n")
			file := filepath.Join(deploy, "es", "llms-full.txt")
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "directory":
				if err := os.Mkdir(file, 0755); err != nil {
					t.Fatal(err)
				}
			case "empty":
				put(t, deploy, "es/llms-full.txt", "")
			case "HTML":
				put(t, deploy, "es/llms-full.txt", "<!doctype html><html><body>Not generated Markdown</body></html>")
			case "symlink":
				outside := filepath.Join(t.TempDir(), "context.txt")
				if err := os.WriteFile(outside, []byte("# Outside\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, file); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat", true); err == nil {
				t.Fatal("missing or invalid registered context artifact accepted")
			}
			if string(readMarkdownDiscoveryFile(t, deploy, "index.html")) != original {
				t.Fatal("context artifact failure mutated page")
			}
		})
	}
}
