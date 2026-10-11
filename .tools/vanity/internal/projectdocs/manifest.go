// Package projectdocs assembles reviewed, immutable project documentation as
// data. Fetched Markdown never reaches Zas's templates or embed handlers.
package projectdocs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Manifest struct {
	Version  int       `yaml:"version"`
	Projects []Project `yaml:"projects"`
}
type Project struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Repo        string `yaml:"repo"`
	Ref         string `yaml:"ref"`
	Commit      string `yaml:"commit"`
	Language    string `yaml:"language"`
	License     string `yaml:"license"`
	LicenseFile string `yaml:"license_file"`
	Pages       []Page `yaml:"pages"`
}
type Page struct {
	Source  string `yaml:"source"`
	Title   string `yaml:"title"`
	Summary string `yaml:"summary"`
}

var repositories = map[string]string{"mergo": "darccio/mergo", "zas": "darccio/zas", "darna": "darccio/darna"}
var licenses = map[string]string{"mergo": "BSD-3-Clause", "zas": "AGPL-3.0-or-later", "darna": "Apache-2.0"}
var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var pathPattern = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)

func Read(root string) (Manifest, error) {
	var m Manifest
	name := filepath.Join(root, ".zas", "projects.yml")
	info, err := os.Lstat(name)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if !info.Mode().IsRegular() {
		return m, fmt.Errorf("project manifest must be a regular file")
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return m, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&m); err != nil {
		return m, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return m, fmt.Errorf("project manifest must contain exactly one document")
	}
	return m, m.Validate()
}
func (m Manifest) Validate() error {
	if m.Version != 1 || len(m.Projects) == 0 {
		return fmt.Errorf("project manifest requires version 1 and projects")
	}
	seen := map[string]bool{}
	for _, p := range m.Projects {
		if repositories[p.ID] == "" || p.Repo != repositories[p.ID] || seen[p.ID] {
			return fmt.Errorf("unknown, mismatched or duplicate project %q", p.ID)
		}
		seen[p.ID] = true
		if !shaPattern.MatchString(p.Commit) || !plain(p.Name) || !plain(p.Ref) || p.Language != "en" || p.License != licenses[p.ID] || p.LicenseFile != "docs/LICENSE.txt" {
			return fmt.Errorf("invalid pin, metadata or license for %s", p.ID)
		}
		paths := map[string]bool{}
		outputs := map[string]bool{}
		for _, page := range p.Pages {
			if !safePath(page.Source) || !strings.HasPrefix(page.Source, "docs/") || !strings.HasSuffix(page.Source, ".md") || paths[page.Source] || !plain(page.Title) || !plain(page.Summary) {
				return fmt.Errorf("invalid or duplicate docs page %q", page.Source)
			}
			destination := htmlPath(p, page)
			if outputs[destination] {
				return fmt.Errorf("duplicate docs destination %s", destination)
			}
			outputs[destination] = true
			paths[page.Source] = true
		}
		if !paths["docs/index.md"] {
			return fmt.Errorf("%s has no landing page", p.ID)
		}
	}
	return nil
}
func plain(s string) bool {
	return strings.TrimSpace(s) == s && s != "" && !strings.ContainsAny(s, "\r\n\x00<>")
}
func safePath(s string) bool {
	return pathPattern.MatchString(s) && path.Clean(s) == s && !strings.HasPrefix(s, "/") && s != "." && s != ".." && !strings.HasPrefix(s, "../") && !strings.Contains(s, "/.")
}
func htmlPath(p Project, page Page) string {
	stem := strings.TrimSuffix(strings.TrimPrefix(page.Source, "docs/"), ".md")
	if path.Base(stem) == "index" {
		return path.Join(p.ID, stem) + ".html"
	}
	return path.Join(p.ID, stem, "index.html")
}
func markdownPath(p Project, page Page) string {
	return path.Join(p.ID, strings.TrimPrefix(page.Source, "docs/"))
}
func canonical(file string) string { return "/" + strings.TrimSuffix(file, "index.html") }
