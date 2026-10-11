package buildsite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectDocsDeclaredTwinsAndVanityNegotiation(t *testing.T) {
	deploy := t.TempDir()
	files := map[string]string{
		"mergo/index.html":         `<!doctype html><html lang="en"><head><link rel="canonical" href="https://dario.cat/mergo/"><meta name="go-import" content="dario.cat/mergo git https://github.com/imdario/mergo"></head><body>Mergo docs</body></html>`,
		"mergo/index.md":           "# Mergo\n",
		"zas/templates/index.html": `<!doctype html><html lang="en"><head><link rel="canonical" href="https://dario.cat/zas/templates/"></head><body>Templates</body></html>`,
		"zas/templates.md":         "# Templates\n\n`{{.Title}}`\n",
	}
	for name, body := range files {
		file := filepath.Join(deploy, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(file), 0755)
		if err := os.WriteFile(file, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	docs := map[string]string{"mergo/index.html": "mergo/index.md", "zas/templates/index.html": "zas/templates.md"}
	if err := addMarkdownDiscoveryWithContexts(deploy, "https://dario.cat", false, docs); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(deploy, "zas/templates/index.html"))
	if !strings.Contains(string(body), `href="https://dario.cat/zas/templates.md"`) {
		t.Fatal("wrong docs twin advertised")
	}
	base, _ := markdownDiscoveryBase("https://dario.cat")
	pages, routes, err := collectHTTPWorkerPages(deploy, base, docs)
	if err != nil {
		t.Fatal(err)
	}
	if *pages["/zas/templates/"].Path != "/zas/templates.md" || pages["/mergo/"].Path == nil {
		t.Fatalf("docs pages absent or wrong twins: %#v", pages)
	}
	for _, excluded := range routes.Exclude {
		if excluded == "/mergo/" || excluded == "/mergo" || excluded == "/mergo/index.html" {
			t.Fatal("docs vanity landing bypasses negotiation")
		}
	}
	if err := httpWorkerMarkdown([]byte("# Page\n{{.Title}}"), false); err == nil {
		t.Fatal("ordinary pages accepted unrendered template")
	}
}

func TestInvalidDocsManifestPreservesPriorDeployBeforeGenerator(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/projects.yml", "version: 99\nprojects: []\n")
	options := Options{Root: root, Zas: generator(t, "touch generator-ran\nexit 9\n")}
	err := Build(context.Background(), options)
	if err == nil || !strings.Contains(err.Error(), "assemble project docs") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "generator-ran")); !os.IsNotExist(err) {
		t.Fatal("generator executed before docs validation")
	}
	prior, err := os.ReadFile(filepath.Join(root, ".zas/deploy/previous.txt"))
	if err != nil || string(prior) != "last successful deployment" {
		t.Fatal("lost previous deployment", err)
	}
}
