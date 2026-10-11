// Package vanity protects the reviewed import metadata independently of Zas.
package vanity

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

//go:embed baseline.json
var baselineJSON []byte

type Metadata struct {
	Import []string `json:"go_import"`
	Source []string `json:"go_source"`
}

type Route struct {
	Path string `json:"path"`
	Metadata
}

func (r Route) File() string { return strings.Trim(r.Path, "/") + "/index.html" }

// Routes returns a fresh copy, never a baseline derived from generated output.
func Routes() []Route {
	var routes []Route
	if err := json.Unmarshal(baselineJSON, &routes); err != nil {
		panic(err)
	}
	return routes
}

// Parse reads tokens rather than looking for metadata strings. Head fragments
// are accepted, as are full documents. Once the head ends, it cannot reopen.
// Actual metadata in a body is rejected, even if correct metadata preceded it.
func Parse(r io.Reader) (Metadata, error) {
	z := html.NewTokenizer(r)
	var result Metadata
	inHead, rawText := true, ""
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if err := z.Err(); err != io.EOF {
				return result, err
			}
			return result, nil
		}
		token := z.Token()
		if rawText != "" {
			if kind == html.EndTagToken && token.Data == rawText {
				rawText = ""
			}
			continue
		}
		if kind == html.TextToken && strings.TrimSpace(token.Data) != "" {
			inHead = false
		}
		if kind == html.EndTagToken && (token.Data == "head" || token.Data == "html" || token.Data == "body" || token.Data == "frameset") {
			inHead = false
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		switch token.Data {
		case "html", "head", "base", "basefont", "bgsound", "link", "meta":
		case "title", "style", "script", "noscript":
			rawText = token.Data
		default:
			inHead = false
		}
		if token.Data != "meta" {
			continue
		}
		attrs := map[string]string{}
		// x/net/html retains the first duplicate attribute, matching Go's
		// discovery parser. Duplicate metadata tags still retain multiplicity.
		for _, attr := range token.Attr {
			attrs[attr.Key] = attr.Val
		}
		name := attrs["name"]
		if !strings.EqualFold(name, "go-import") && !strings.EqualFold(name, "go-source") {
			continue
		}
		if name != "go-import" && name != "go-source" {
			return result, fmt.Errorf("metadata name must be exactly go-import or go-source: %q", name)
		}
		if !inHead {
			return result, fmt.Errorf("%s outside head", name)
		}
		fields := strings.Fields(attrs["content"])
		want := 3
		if name == "go-source" {
			want = 4
		}
		if len(fields) != want {
			return result, fmt.Errorf("%s needs %d fields, got %d", name, want, len(fields))
		}
		content := strings.Join(fields, " ")
		if name == "go-import" {
			result.Import = append(result.Import, content)
		} else {
			result.Source = append(result.Source, content)
		}
	}
}

func (m Metadata) Empty() bool { return len(m.Import)+len(m.Source) == 0 }

func Validate(r Route, got Metadata) error {
	// Sort copies, preserving multiplicity: duplicate tags never disappear.
	for _, pair := range []struct {
		name      string
		got, want []string
	}{{"go-import", got.Import, r.Import}, {"go-source", got.Source, r.Source}} {
		actual, expected := slices.Clone(pair.got), slices.Clone(pair.want)
		slices.Sort(actual)
		slices.Sort(expected)
		if !slices.Equal(actual, expected) {
			return fmt.Errorf("%s %s: got %q, want %q", r.Path, pair.name, actual, expected)
		}
	}
	return nil
}

// CheckDir scans every HTML file, including routes outside the baseline.
func CheckDir(dir string) error {
	expected := map[string]Route{}
	for _, route := range Routes() {
		expected[route.File()] = route
	}
	var problems []error
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("deploy contains a symlink: %s", path)
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		metadata, parseErr := Parse(f)
		closeErr := f.Close()
		if closeErr != nil {
			return closeErr
		}
		if parseErr != nil {
			problems = append(problems, fmt.Errorf("%s: %w", rel, parseErr))
		}
		if route, known := expected[rel]; known {
			if err := Validate(route, metadata); err != nil {
				problems = append(problems, err)
			}
			delete(expected, rel)
		} else if !metadata.Empty() {
			problems = append(problems, fmt.Errorf("unexpected vanity metadata at %s", rel))
		}
		return nil
	})
	if err != nil {
		problems = append(problems, err)
	}
	var missing []string
	for file := range expected {
		missing = append(missing, file)
	}
	slices.Sort(missing)
	for _, file := range missing {
		problems = append(problems, fmt.Errorf("missing vanity file: %s", file))
	}
	return errors.Join(problems...)
}
