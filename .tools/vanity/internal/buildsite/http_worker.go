package buildsite

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"dario.cat/site-tools/internal/vanity"
	"go.yaml.in/yaml/v3"
)

const httpWorkerMarker = "/* SITE_PAGE_ROUTES */ {}"

type httpWorkerPage struct {
	Path       *string `json:"path"`
	SHA256     *string `json:"sha256"`
	HTMLSHA256 string  `json:"htmlSha256"`
}

type httpWorkerRoutes struct {
	Version int      `json:"version"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// The Pages adapter is assembled only after Markdown discovery has validated
// the public twins. Its inventory hashes the final deploy bytes, and all output
// is written inside the isolated build before the previous deploy is promoted.
func finalizeHTTPWorker(root, stage, deploy string) error {
	configBytes, err := readAgentContextFile(stage, ".zas/config.yml")
	if err != nil {
		return err
	}
	var config struct {
		Site struct {
			MarkdownNegotiation bool   `yaml:"markdown_negotiation"`
			PublishMarkdown     bool   `yaml:"publish_markdown"`
			BaseURL             string `yaml:"baseurl"`
		} `yaml:"site"`
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	if !config.Site.MarkdownNegotiation {
		return nil
	}
	if !config.Site.PublishMarkdown {
		return errors.New("HTTP Markdown negotiation requires site.publish_markdown")
	}
	base, err := markdownDiscoveryBase(config.Site.BaseURL)
	if err != nil {
		return fmt.Errorf("HTTP worker site.baseurl: %w", err)
	}
	if basePath := strings.Trim(base.Path, "/"); basePath != "" {
		if err := httpWorkerSafePath(basePath); err != nil {
			return fmt.Errorf("HTTP worker site.baseurl: %w", err)
		}
	}
	for _, name := range []string{"_worker.js", "_routes.json"} {
		for _, directory := range []string{root, deploy} {
			if _, err := os.Lstat(filepath.Join(directory, name)); err == nil {
				return fmt.Errorf("HTTP worker refuses existing %s in %s", name, directory)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	template, err := readAgentContextFile(root, ".tools/http/worker.mjs")
	if err != nil {
		return fmt.Errorf("HTTP worker template: %w", err)
	}
	if bytes.Count(template, []byte(httpWorkerMarker)) != 1 {
		return errors.New("HTTP worker template must contain exactly one page-inventory marker")
	}
	pages, routes, err := collectHTTPWorkerPages(deploy, base)
	if err != nil {
		return err
	}
	inventory, err := json.Marshal(pages)
	if err != nil {
		return err
	}
	worker := bytes.Replace(template, []byte(httpWorkerMarker), inventory, 1)
	routeBytes, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}
	routeBytes = append(routeBytes, '\n')
	if err := os.WriteFile(filepath.Join(deploy, "_worker.js"), worker, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(deploy, "_routes.json"), routeBytes, 0644)
}

func collectHTTPWorkerPages(deploy string, base *url.URL) (map[string]httpWorkerPage, httpWorkerRoutes, error) {
	info, err := os.Lstat(deploy)
	if err != nil {
		return nil, httpWorkerRoutes{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, httpWorkerRoutes{}, errors.New("HTTP worker requires a regular deploy directory")
	}
	pages := make(map[string]httpWorkerPage)
	exclusions := make(map[string]bool)
	// Exclude known asset subtrees without opening or hashing their contents.
	// Pages uses glob matching here, so this suffix rule covers nested twins.
	exclusions[httpWorkerEscapedPath(base, "*.md")] = true
	for _, directory := range []string{"static", "assets", "CV", ".well-known"} {
		exclusions[httpWorkerEscapedPath(base, directory+"/*")] = true
	}
	raw := make(map[string]bool)
	for _, route := range vanity.Routes() {
		raw[route.File()] = true
		prefix := strings.Trim(route.Path, "/")
		exclusions[httpWorkerEscapedPath(base, prefix)] = true
		exclusions[httpWorkerEscapedPath(base, prefix+"/")] = true
		exclusions[httpWorkerEscapedPath(base, route.File())] = true
	}
	err = filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if file == deploy {
			return nil
		}
		relative, err := filepath.Rel(deploy, file)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("HTTP worker refuses deploy symlink %s", relative)
		}
		if err := httpWorkerSafePath(relative); err != nil {
			return err
		}
		if raw[relative] {
			return nil
		}
		if httpWorkerExcluded(relative) {
			if entry.IsDir() {
				exclusions[httpWorkerEscapedPath(base, relative+"/*")] = true
				return fs.SkipDir
			}
			exclusions[httpWorkerEscapedPath(base, relative)] = true
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("HTTP worker artifact %s is not a regular file", relative)
		}
		if !strings.EqualFold(filepath.Ext(relative), ".html") {
			if !strings.HasSuffix(relative, ".md") && relative != "_headers" {
				exclusions[httpWorkerEscapedPath(base, relative)] = true
			}
			return nil
		}
		body, err := readAgentContextFile(deploy, relative)
		if err != nil {
			return err
		}
		canonical, _, err := sitemapHTMLMetadata(body)
		if err != nil {
			return fmt.Errorf("HTTP worker %s: %w", relative, err)
		}
		location, err := localSitemapURL(canonical, base)
		if err != nil {
			return fmt.Errorf("HTTP worker %s canonical: %w", relative, err)
		}
		expected := *base
		expected.Path, expected.RawPath = httpWorkerPath(base, relative), ""
		if path.Base(relative) == "index.html" {
			expected.Path = strings.TrimSuffix(expected.Path, "index.html")
		}
		if location.String() != expected.String() {
			return fmt.Errorf("HTTP worker %s canonical %q does not resolve to its deployed page %q", relative, canonical, expected.String())
		}
		page := httpWorkerPage{HTMLSHA256: httpWorkerDigest(body)}
		twin := strings.TrimSuffix(relative, filepath.Ext(relative)) + ".md"
		if _, err := os.Lstat(filepath.Join(deploy, filepath.FromSlash(twin))); err == nil {
			markdown, err := readAgentContextFile(deploy, twin)
			if err != nil {
				return fmt.Errorf("HTTP worker %s twin: %w", relative, err)
			}
			if err := agentContextMarkdown(markdown); err != nil {
				return fmt.Errorf("HTTP worker %s twin: %w", relative, err)
			}
			twinPath, digest := httpWorkerEscapedPath(base, twin), httpWorkerDigest(markdown)
			page.Path, page.SHA256 = &twinPath, &digest
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := addHTTPWorkerPage(pages, location.EscapedPath(), page); err != nil {
			return err
		}
		// Pages redirects index/bare aliases itself. Only non-index HTML pages
		// need an explicit extensionless alias, without collapsing directories.
		if path.Base(relative) != "index.html" {
			alias := strings.TrimSuffix(location.EscapedPath(), filepath.Ext(relative))
			if err := addHTTPWorkerPage(pages, alias, page); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, httpWorkerRoutes{}, err
	}
	if len(pages) == 0 {
		return nil, httpWorkerRoutes{}, errors.New("HTTP worker found no normal deployed HTML pages")
	}
	routes := httpWorkerRoutes{Version: 1, Include: []string{"/*"}}
	for exclusion := range exclusions {
		// Remove exact paths already covered by a terminal subtree wildcard.
		covered := false
		for group := range exclusions {
			if strings.HasSuffix(group, "/*") && exclusion != group && strings.HasPrefix(exclusion, strings.TrimSuffix(group, "*")) {
				covered = true
				break
			}
		}
		if !covered {
			if len(exclusion) > 100 {
				return nil, httpWorkerRoutes{}, fmt.Errorf("HTTP worker route %q exceeds Cloudflare Pages' 100-character limit", exclusion)
			}
			routes.Exclude = append(routes.Exclude, exclusion)
		}
	}
	sort.Strings(routes.Exclude)
	if len(routes.Include)+len(routes.Exclude) > 100 {
		return nil, httpWorkerRoutes{}, fmt.Errorf("HTTP worker routing requires %d rules; Cloudflare Pages permits at most 100", len(routes.Include)+len(routes.Exclude))
	}
	return pages, routes, nil
}

func addHTTPWorkerPage(pages map[string]httpWorkerPage, pathname string, page httpWorkerPage) error {
	if _, exists := pages[pathname]; exists {
		return fmt.Errorf("HTTP worker route collision at %s", pathname)
	}
	pages[pathname] = page
	return nil
}

func httpWorkerPath(base *url.URL, relative string) string {
	return strings.TrimRight(base.Path, "/") + "/" + relative
}

func httpWorkerEscapedPath(base *url.URL, relative string) string {
	location := &url.URL{Path: httpWorkerPath(base, relative)}
	// Literal asterisks are rejected in inventory and base paths. These are
	// only the explicit Pages route wildcards constructed by the assembler.
	return strings.ReplaceAll(location.EscapedPath(), "%2A", "*")
}

func httpWorkerDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func httpWorkerExcluded(relative string) bool {
	parts := strings.Split(relative, "/")
	for _, part := range parts {
		if strings.EqualFold(part, "CV") || strings.EqualFold(part, "private") {
			return true
		}
	}
	if relative == "docs/HANDOFF-FACTS.html" || relative == "docs/DOMAIN-OPERATIONS.html" {
		return true
	}
	return !allowedSource(relative) || parts[0] == "static" || parts[0] == "assets" || parts[0] == ".well-known"
}

func httpWorkerSafePath(relative string) error {
	// Pages' route matcher treats regex punctuation as syntax, and its URL
	// pathname encoding differs from Go's for several other punctuation marks.
	// Refuse those names instead of silently routing the wrong representation.
	if relative == "" || relative != path.Clean(relative) || strings.HasPrefix(relative, "/") || strings.ContainsAny(relative, "*?%#\\!'()[]+$^{}|") || strings.IndexFunc(relative, unicode.IsControl) != -1 {
		return fmt.Errorf("HTTP worker cannot safely route deploy path %q", relative)
	}
	return nil
}
