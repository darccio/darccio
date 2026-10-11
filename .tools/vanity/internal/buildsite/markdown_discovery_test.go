package buildsite

import (
	"bytes"
	"context"
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
