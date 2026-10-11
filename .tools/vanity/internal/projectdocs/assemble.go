package projectdocs

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.yaml.in/yaml/v3"
	xhtml "golang.org/x/net/html"
)

type Assembly struct {
	Twins    map[string]string
	LastMod  map[string]time.Time
	files    map[string][]byte
	projects []Project
}
type compiled struct {
	wrappers map[string][]byte
	files    map[string][]byte
	twins    map[string]string
}

func Prepare(ctx context.Context, root, stage string, fetch Fetcher) (*Assembly, error) {
	m, err := Read(root)
	if err != nil {
		return nil, err
	}
	a := &Assembly{Twins: map[string]string{}, LastMod: map[string]time.Time{}, files: map[string][]byte{}, projects: m.Projects}
	if fetch == nil {
		fetch = Fetch
	}
	for _, p := range m.Projects {
		rev, err := fetch(ctx, p)
		if err != nil {
			return nil, err
		}
		doc, err := compile(p, rev)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.ID, err)
		}
		for name, body := range doc.wrappers {
			if err := writeNew(stage, name, body); err != nil {
				return nil, err
			}
			if err := os.Chtimes(filepath.Join(stage, filepath.FromSlash(name)), rev.Updated, rev.Updated); err != nil {
				return nil, err
			}
		}
		for name, body := range doc.files {
			if _, err := os.Lstat(filepath.Join(stage, filepath.FromSlash(name))); !os.IsNotExist(err) {
				return nil, fmt.Errorf("project docs output collides with source %s", name)
			}
			a.files[name] = body
		}
		for h, md := range doc.twins {
			a.Twins[h] = md
			a.LastMod[h] = rev.Updated
		}
	}
	return a, nil
}
func compile(p Project, rev Revision) (compiled, error) {
	c := compiled{wrappers: map[string][]byte{}, files: map[string][]byte{}, twins: map[string]string{}}
	if rev.Updated.IsZero() {
		return c, fmt.Errorf("missing source commit timestamp")
	}
	license, ok := rev.Files[p.LicenseFile]
	if !ok || len(bytes.TrimSpace(license)) == 0 {
		return c, fmt.Errorf("missing source license")
	}
	c.files[p.ID+"/LICENSE.txt"] = license
	var nav strings.Builder
	fmt.Fprintf(&nav, `<nav aria-label="%s documentation"><ul>`, html.EscapeString(p.Name))
	links := map[string]string{p.LicenseFile: "/" + p.ID + "/LICENSE.txt"}
	for _, page := range p.Pages {
		h := htmlPath(p, page)
		links[page.Source] = canonical(h)
		fmt.Fprintf(&nav, `<li><a href="%s">%s</a></li>`, canonical(h), html.EscapeString(page.Title))
	}
	nav.WriteString("</ul></nav>\n")
	var index, full strings.Builder
	fmt.Fprintf(&index, "# %s\n\n> %s\n\n## Documentation\n", p.Name, p.Pages[0].Summary)
	fmt.Fprintf(&full, "# %s — full documentation\n\nSource: https://github.com/%s/tree/%s/docs\n\nLicense: %s (https://dario.cat/%s/LICENSE.txt). Each section identifies its Markdown URL for resolving relative links.\n", p.Name, p.Repo, p.Commit, p.License, p.ID)
	for _, page := range p.Pages {
		source, ok := rev.Files[page.Source]
		if !ok {
			return c, fmt.Errorf("missing selected page %s", page.Source)
		}
		md, err := stripConfig(source)
		if err != nil {
			return c, err
		}
		rendered, err := render(md, page.Source, links)
		if err != nil {
			return c, err
		}
		h, twin := htmlPath(p, page), markdownPath(p, page)
		metadata := map[string]any{"title": page.Title, "summary": page.Summary, "language": "en", "template": false, "publish_markdown": false, "feed": false, "entry": true, "updated": rev.Updated.Format(time.RFC3339), "docs_project": p.ID, "docs_license": p.License, "docs_license_url": "/" + p.ID + "/LICENSE.txt"}
		if page.Source == "docs/index.md" {
			metadata["code_repository"] = "https://github.com/" + p.Repo
			metadata["project_name"] = p.Name
		}
		config, err := yaml.Marshal(metadata)
		if err != nil {
			return c, err
		}
		wrapper := append([]byte("<!--\n"), config...)
		wrapper = append(wrapper, []byte("-->\n"+nav.String())...)
		wrapper = append(wrapper, rendered...)
		provenance := fmt.Sprintf("\n<p>Source: <a href=\"https://github.com/%s/blob/%s/%s\">%s at %s</a>. <a href=\"/%s/LICENSE.txt\">%s license</a>. <a href=\"/%s/llms.txt\">Documentation index</a> · <a href=\"/%s/llms-full.txt\">Full documentation</a>.</p>\n", p.Repo, p.Commit, page.Source, html.EscapeString(p.Name), p.Commit[:12], p.ID, p.License, p.ID, p.ID)
		wrapper = append(wrapper, []byte(provenance)...)
		c.wrappers[h] = wrapper
		c.twins[h] = twin
		c.files[twin] = md
		fmt.Fprintf(&index, "- [%s](https://dario.cat/%s): %s\n", page.Title, twin, page.Summary)
		fmt.Fprintf(&full, "\n---\n\nSource: https://dario.cat/%s\n\n%s", twin, md)
	}
	fmt.Fprintf(&index, "\n## Optional\n- [Full documentation](https://dario.cat/%s/llms-full.txt)\n- [Source revision](https://github.com/%s/tree/%s/docs)\n- [License](https://dario.cat/%s/LICENSE.txt): %s\n", p.ID, p.Repo, p.Commit, p.ID, p.License)
	c.files[p.ID+"/llms.txt"] = []byte(index.String())
	c.files[p.ID+"/llms-full.txt"] = []byte(full.String())
	return c, nil
}
func stripConfig(source []byte) ([]byte, error) {
	source = bytes.TrimPrefix(source, []byte{0xef, 0xbb, 0xbf})
	source = bytes.TrimSpace(source)
	if bytes.HasPrefix(source, []byte("<!--")) {
		end := bytes.Index(source, []byte("-->"))
		if end < 0 {
			return nil, fmt.Errorf("unclosed leading docs comment")
		}
		source = bytes.TrimSpace(source[end+3:])
	}
	if len(source) == 0 || !bytes.HasPrefix(source, []byte("# ")) {
		return nil, fmt.Errorf("selected documentation must begin with an H1")
	}
	return append(bytes.Clone(source), '\n'), nil
}
func render(md []byte, source string, links map[string]string) ([]byte, error) {
	engine := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithParserOptions(parser.WithAutoHeadingID()))
	doc := engine.Parser().Parse(text.NewReader(md))
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var dest *[]byte
		switch n := n.(type) {
		case *ast.Link:
			dest = &n.Destination
		case *ast.Image:
			dest = &n.Destination
		}
		if dest == nil {
			return ast.WalkContinue, nil
		}
		location, err := url.Parse(string(*dest))
		if err != nil {
			return ast.WalkStop, err
		}
		if location.IsAbs() || location.Host != "" || location.Path == "" {
			return ast.WalkContinue, nil
		}
		if strings.HasPrefix(location.Path, "/") {
			return ast.WalkStop, fmt.Errorf("%s: root-relative docs link needs an explicit external URL", source)
		}
		target := path.Clean(path.Join(path.Dir(source), location.Path))
		rewritten, ok := links[target]
		if !ok {
			return ast.WalkStop, fmt.Errorf("%s: local link %s is outside selected documentation", source, *dest)
		}
		location.Path, location.RawPath = rewritten, ""
		*dest = []byte(location.String())
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	var body bytes.Buffer
	if err := engine.Renderer().Render(&body, md, doc); err != nil {
		return nil, err
	}
	return bytes.Replace(body.Bytes(), []byte("<h1"), []byte("<h1 class=\"p-name\""), 1), nil
}
func (a *Assembly) Owns(name string) bool { _, ok := a.Twins[name]; return ok }
func (a *Assembly) Finalize(root, deploy string) error {
	for name, body := range a.files {
		if err := writeNew(deploy, name, body); err != nil {
			return err
		}
	}
	for _, p := range a.projects {
		source := filepath.Join(root, p.ID, "index.html")
		if p.ID == "zas" {
			continue
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		tags, err := vanityTags(raw)
		if err != nil {
			return err
		}
		dest := filepath.Join(deploy, p.ID, "index.html")
		body, err := os.ReadFile(dest)
		if err != nil {
			return err
		}
		if bytes.Count(body, []byte("</head>")) != 1 {
			return fmt.Errorf("%s docs landing has no unique closing head", p.ID)
		}
		body = bytes.Replace(body, []byte("</head>"), append(tags, []byte("</head>")...), 1)
		if err := os.WriteFile(dest, body, 0644); err != nil {
			return err
		}
	}
	return nil
}

// Append only a link index, never third-party content, to the generated site
// context indexes. Their reviewed personal full-text corpora remain unchanged.
func (a *Assembly) LinkContexts(deploy string) error {
	if len(a.projects) == 0 {
		return nil
	}
	for _, file := range []string{"llms.txt", "en/llms.txt"} {
		name := filepath.Join(deploy, filepath.FromSlash(file))
		body, err := os.ReadFile(name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var links strings.Builder
		links.WriteString("\n## Open source documentation\n")
		for _, p := range a.projects {
			fmt.Fprintf(&links, "- [%s](https://dario.cat/%s/llms.txt): English guides, Markdown pages and original %s license.\n", p.Name, p.ID, p.License)
		}
		if err := os.WriteFile(name, append(body, []byte(links.String())...), 0644); err != nil {
			return err
		}
	}
	return nil
}
func vanityTags(raw []byte) ([]byte, error) {
	z := xhtml.NewTokenizer(bytes.NewReader(raw))
	var tags []byte
	for {
		kind := z.Next()
		if kind == xhtml.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			break
		}
		if kind != xhtml.StartTagToken && kind != xhtml.SelfClosingTagToken {
			continue
		}
		original := bytes.Clone(z.Raw())
		token := z.Token()
		if token.Data != "meta" {
			continue
		}
		for _, attr := range token.Attr {
			if attr.Key == "name" && (attr.Val == "go-import" || attr.Val == "go-source") {
				tags = append(tags, original...)
				tags = append(tags, '\n')
				break
			}
		}
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("no vanity metadata in original source")
	}
	return tags, nil
}
func writeNew(root, name string, body []byte) error {
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("docs artifact collision %s: %w", name, err)
	}
	_, err = out.Write(body)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func writeManifest(root string, m Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	name := filepath.Join(root, ".zas", "projects.yml")
	temporary, err := os.CreateTemp(filepath.Dir(name), ".projects-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0644); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), name)
}
