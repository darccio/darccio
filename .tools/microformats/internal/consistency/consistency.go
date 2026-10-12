// Package consistency checks the completed site against the reviewed article
// sources using independent XML, HTML, microformats and JSON parsers. It makes
// no network requests and does not open CV data.
package consistency

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Report struct {
	Posts, Dated, Partial, Excluded, Feeds, Items, Approved, Published int
}

// Directory validates source facts against every expected built representation.
// standardPlanPath is a required offline plan artifact, including when its
// reviewed document set is empty. Planned URIs never imply published records.
func Directory(root, deploy, standardPlanPath string) (Report, error) {
	var report Report
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return report, err
	}
	deploy, err = filepath.Abs(deploy)
	if err != nil {
		return report, err
	}
	config, sources, err := loadSources(root)
	if err != nil {
		return report, err
	}
	posts := make(map[string]sourcePost)
	rendered := make(map[string]renderedPost)
	var failures []error
	for _, post := range sources {
		if _, duplicate := posts[post.Path]; duplicate {
			failures = append(failures, fmt.Errorf("duplicate canonical source path %q", post.Path))
			continue
		}
		posts[post.Path] = post
		if post.Excluded {
			report.Excluded++
			input, err := regularFile(deploy, post.File)
			if os.IsNotExist(err) {
				continue
			}
			if err == nil {
				doc, parseErr := parseHTML(input)
				err = parseErr
				if err == nil && len(find(doc, func(n *htmlNode) bool {
					return hasClass(n, "h-entry") || isTag(n, "link") && strings.Contains(attribute(n, "rel"), "site.standard.document")
				})) != 0 {
					err = fmt.Errorf("draft or unpublished article acquired syndication markup")
				}
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", post.File, err))
			}
			continue
		}
		report.Posts++
		if post.Date.IsZero() {
			report.Partial++
		} else {
			report.Dated++
		}
		input, err := regularFile(deploy, post.File)
		if err == nil {
			rendered[post.Path], err = checkArticle(post, input)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", post.File, err))
		}
	}
	feeds := expectedFeeds(config, sources)
	var names []string
	for name := range feeds {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		input, err := regularFile(deploy, name)
		if err == nil {
			err = checkFeed(config, name, feeds[name], rendered, input)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
		}
		report.Feeds++
		report.Items += len(feeds[name])
	}
	for _, language := range []string{"en", "es"} {
		name := language + "/feed.xml"
		if _, expected := feeds[name]; !expected {
			if _, err := os.Lstat(filepath.Join(deploy, filepath.FromSlash(name))); !os.IsNotExist(err) {
				failures = append(failures, fmt.Errorf("unexpected stale language feed %s", name))
			}
		}
	}
	report.Approved, report.Published, err = checkStandard(root, deploy, standardPlanPath, posts, rendered)
	if err != nil {
		failures = append(failures, fmt.Errorf("Standard.site: %w", err))
	}
	return report, errors.Join(failures...)
}

func regularFile(root, rel string) ([]byte, error) {
	if filepath.IsAbs(rel) || filepath.ToSlash(filepath.Clean(rel)) != rel || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "\\") {
		return nil, fmt.Errorf("unsafe artifact path %q", rel)
	}
	current := root
	parts := strings.Split(rel, "/")
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("artifact path is a symlink: %s", current)
		}
		if index != len(parts)-1 && !info.IsDir() || index == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("artifact path is not a regular file: %s", current)
		}
	}
	return os.ReadFile(current)
}

func absoluteRegularFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("an offline record plan is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return regularFile(filepath.VolumeName(absolute)+string(filepath.Separator), filepath.ToSlash(strings.TrimPrefix(absolute, filepath.VolumeName(absolute)+string(filepath.Separator))))
}

func decodeJSON(input []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	if err := uniqueJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("JSON must contain exactly one value")
	}
	decoder = json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func uniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	seen := make(map[string]bool)
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate or invalid JSON key %q", key)
			}
			seen[name] = true
		}
		if err := uniqueJSONValue(decoder); err != nil {
			return err
		}
	}
	end, err := decoder.Token()
	if err != nil || delimiter == '{' && end != json.Delim('}') || delimiter == '[' && end != json.Delim(']') {
		return fmt.Errorf("unterminated JSON value")
	}
	return nil
}
