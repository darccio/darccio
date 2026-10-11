package projectdocs

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture() (Project, Revision) {
	p := Project{ID: "zas", Name: "Zas", Repo: "darccio/zas", Ref: "master", Commit: strings.Repeat("a", 40), Language: "en", License: "AGPL-3.0-or-later", LicenseFile: "docs/LICENSE.txt", Pages: []Page{{"docs/index.md", "Zas", "Static sites."}, {"docs/templates.md", "Templates", "Template reference."}}}
	r := Revision{Updated: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC), Files: map[string][]byte{"docs/LICENSE.txt": []byte("upstream license\n"), "docs/index.md": []byte("<!--\ntemplate: true\nlayout: /secret\n-->\n# Zas\n\n[Templates](templates.md#layout) and [license](LICENSE.txt).\n"), "docs/templates.md": []byte("# Templates\n\n```html\n{{.E \"Hello\"}}\n<embed src=\"../../CV/MAC.json\" type=\"text/plain\">\n```\n\n<script type=\"application/zas+exec\">steal()</script>\n\n<embed src=\"../../CV/MAC.json\">\n\n[Unsafe](javascript:alert(1))\n")}}
	return p, r
}
func TestCompileLiteralDocumentation(t *testing.T) {
	p, r := fixture()
	c, err := compile(p, r)
	if err != nil {
		t.Fatal(err)
	}
	landing := string(c.wrappers["zas/index.html"])
	if !strings.Contains(landing, `href="/zas/templates/#layout"`) || !strings.Contains(landing, `href="/zas/LICENSE.txt"`) || !strings.Contains(landing, "template: false") || strings.Contains(landing, "layout: /secret") {
		t.Fatal(landing)
	}
	body := string(c.wrappers["zas/templates/index.html"])
	if !strings.Contains(body, "{{.E") || !strings.Contains(body, "&lt;embed") || strings.Contains(body, "<script type=") || strings.Contains(body, "<embed ") || strings.Contains(body, `href="javascript:`) {
		t.Fatal(body)
	}
	md := c.files["zas/templates.md"]
	if !bytes.Equal(md, r.Files["docs/templates.md"]) || c.twins["zas/templates/index.html"] != "zas/templates.md" {
		t.Fatal("literal twin or mapping changed")
	}
	if !bytes.Equal(c.files["zas/LICENSE.txt"], r.Files[p.LicenseFile]) {
		t.Fatal("license bytes changed")
	}
	if !strings.Contains(string(c.files["zas/llms-full.txt"]), "Source: https://dario.cat/zas/templates.md") {
		t.Fatal("missing section provenance")
	}
	c2, err := compile(p, r)
	if err != nil || !bytes.Equal(c.files["zas/llms-full.txt"], c2.files["zas/llms-full.txt"]) {
		t.Fatal("nondeterministic corpus")
	}
}
func TestManifestRejectsUnsafeAndAmbiguousInputs(t *testing.T) {
	p, _ := fixture()
	cases := map[string]func(*Project){"repo": func(p *Project) { p.Repo = "evil/repo" }, "pin": func(p *Project) { p.Commit = "main" }, "traversal": func(p *Project) { p.Pages[1].Source = "docs/../../CV/MAC.md" }, "dotfile": func(p *Project) { p.Pages[1].Source = "docs/.secret.md" }, "duplicate": func(p *Project) { p.Pages[1].Source = p.Pages[0].Source }, "license": func(p *Project) { p.License = "CC-BY-SA-4.0" }, "language": func(p *Project) { p.Language = "ca" }, "route collision": func(p *Project) {
		p.Pages = append(p.Pages, Page{"docs/templates/index.md", "Duplicate", "Duplicate route."})
	}}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := p
			candidate.Pages = append([]Page(nil), p.Pages...)
			mutate(&candidate)
			if (Manifest{Version: 1, Projects: []Project{candidate}}).Validate() == nil {
				t.Fatal("accepted unsafe manifest")
			}
		})
	}
}
func TestLocalLinksMustStayWithinSelectedDocs(t *testing.T) {
	p, r := fixture()
	for _, link := range []string{"../../CV/MAC.json", "missing.md", "/CV/MAC.json"} {
		t.Run(link, func(t *testing.T) {
			r.Files["docs/index.md"] = []byte("# Zas\n\n[Link](" + link + ")\n")
			if _, err := compile(p, r); err == nil {
				t.Fatal("accepted unknown local target")
			}
		})
	}
}
func TestPreparePreservesSourcesAndRejectsCollisions(t *testing.T) {
	root, stage := t.TempDir(), t.TempDir()
	p, r := fixture()
	if err := os.Mkdir(filepath.Join(root, ".zas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(root, Manifest{Version: 1, Projects: []Project{p}}); err != nil {
		t.Fatal(err)
	}
	fetch := func(context.Context, Project) (Revision, error) { return r, nil }
	a, err := Prepare(context.Background(), root, stage, fetch)
	if err != nil {
		t.Fatal(err)
	}
	deploy := t.TempDir()
	if err := a.Finalize(root, deploy); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(context.Background(), root, stage, fetch); err == nil {
		t.Fatal("overwrote assembled source")
	}
	if err := a.Finalize(root, deploy); err == nil {
		t.Fatal("overwrote existing artifact")
	}
	if _, err := os.Stat(filepath.Join(root, "zas")); !os.IsNotExist(err) {
		t.Fatal("modified actual source root")
	}
}
func TestUpdateRequiresReviewedSourceAndValidatedRevision(t *testing.T) {
	root := t.TempDir()
	p, r := fixture()
	os.Mkdir(filepath.Join(root, ".zas"), 0755)
	if err := writeManifest(root, Manifest{Version: 1, Projects: []Project{p}}); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, ".zas", "projects.yml")
	original, _ := os.ReadFile(name)
	fetch := func(context.Context, Project) (Revision, error) { return r, nil }
	if err := Update(context.Background(), root, "evil/repo", strings.Repeat("b", 40), fetch); err == nil {
		t.Fatal("accepted unknown repo")
	}
	if err := Update(context.Background(), root, p.Repo, "main", fetch); err == nil {
		t.Fatal("accepted moving pin")
	}
	broken := func(context.Context, Project) (Revision, error) { return Revision{}, nil }
	if err := Update(context.Background(), root, p.Repo, strings.Repeat("b", 40), broken); err == nil {
		t.Fatal("accepted invalid source")
	}
	after, _ := os.ReadFile(name)
	if !bytes.Equal(original, after) {
		t.Fatal("failed update modified manifest")
	}
	if err := Update(context.Background(), root, p.Repo, strings.Repeat("b", 40), fetch); err != nil {
		t.Fatal(err)
	}
	updated, err := Read(root)
	if err != nil || updated.Projects[0].Commit != strings.Repeat("b", 40) {
		t.Fatal("valid update failed", err)
	}
}
func TestManifestRejectsSymlinkAndUnknownFields(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".zas"), 0755)
	name := filepath.Join(root, ".zas", "projects.yml")
	if err := os.Symlink("/etc/passwd", name); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root); err == nil {
		t.Fatal("accepted symlink manifest")
	}
	os.Remove(name)
	os.WriteFile(name, []byte("version: 1\nunknown: true\nprojects: []\n"), 0644)
	if _, err := Read(root); err == nil {
		t.Fatal("accepted unknown field")
	}
}
func TestVanityTagBytesRemainExact(t *testing.T) {
	raw := []byte("<meta name=\"go-import\"\n content=\"dario.cat/mergo git https://github.com/imdario/mergo\">\n<meta http-equiv=\"refresh\" content=\"0; url=old\">")
	tags, err := vanityTags(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tags, append(bytes.Split(raw, []byte("\n<meta http-equiv"))[0], '\n')) {
		t.Fatalf("changed tag bytes: %s", tags)
	}
}
