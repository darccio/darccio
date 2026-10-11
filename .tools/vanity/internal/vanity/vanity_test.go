package vanity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fragment(route Route) string {
	return fmt.Sprintf(`<meta name="go-import" content="%s">`, route.Import[0])
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, route := range Routes() {
		write(t, dir, route.File(), fragment(route))
	}
	return dir
}

func TestTrackedSourcesMatchReviewedBaseline(t *testing.T) {
	if len(Routes()) != 7 {
		t.Fatal("reviewed baseline must cover seven routes")
	}
	for _, route := range Routes() {
		file, err := os.Open(filepath.Join("../../../..", route.File()))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := Validate(route, got); err != nil {
			t.Fatal(err)
		}
		if len(route.Source) != 0 {
			t.Fatal("reviewed baseline has no go-source metadata")
		}
	}
}

func TestParserAcceptsFragmentsAndFormatting(t *testing.T) {
	route := Routes()[0]
	for _, content := range []string{
		fragment(route),
		`<!doctype html><html><head>` + fragment(route) + `</head><body>page</body></html>`,
		`<META CONTENT=' dario.cat/mergo` + "\n\tgit   https://github.com/imdario/mergo ' NAME='go-import' />",
		`<MeTa name=go-import content='dario.cat/mergo git https://github.com/imdario/mergo'>`,
		`<meta NAME='go&#45;import' CONTENT="dario.cat/mergo&#x9;git&#32;https://github.com/imdario/mergo" disabled>`,
		`<meta name="description" content="valid HTML with raw < and &" custom@key="x">` + fragment(route),
		`<title>Title</title><style>.x{color:red}</style><script>const x='<meta name="go-import" content="fake git fake">';</script>` + fragment(route),
		`<!-- <meta name="go-import" content="fake git fake"> -->` + fragment(route),
	} {
		got, err := Parse(strings.NewReader(content))
		if err != nil {
			t.Fatal(err)
		}
		if err := Validate(route, got); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMetadataRegressionCases(t *testing.T) {
	route := Routes()[0]
	correct := fragment(route)
	cases := map[string]string{
		"missing":                   `<html><head></head><body>homepage fallback</body></html>`,
		"changed prefix":            strings.Replace(correct, "dario.cat/mergo", "dario.cat/other", 1),
		"changed VCS":               strings.Replace(correct, " git ", " hg ", 1),
		"changed identity":          strings.Replace(correct, "/imdario/", "/darccio/", 1),
		"duplicate":                 correct + correct,
		"conflicting":               correct + strings.Replace(correct, "/imdario/", "/darccio/", 1),
		"comment only":              "<!--" + correct + "-->",
		"script only":               "<script>let value='" + correct + "';</script>",
		"escaped example":           `&lt;meta name="go-import" content="dario.cat/mergo git https://github.com/imdario/mergo"&gt;`,
		"inside body":               "<html><head></head><body>" + correct + "</body></html>",
		"after head":                "<head></head>" + correct,
		"implicit body":             "<p>body</p>" + correct,
		"text before metadata":      "visible text" + correct,
		"head cannot reopen":        "<body></body><head>" + correct + "</head>",
		"extra body metadata":       correct + "<body>" + correct,
		"wrong name case":           strings.Replace(correct, "go-import", "GO-IMPORT", 1),
		"duplicate attribute":       strings.Replace(correct, "name=", `content="fake git fake" name=`, 1),
		"reversed conflicting name": strings.Replace(correct, `name="go-import"`, `name="viewport" name="go-import"`, 1),
		"hidden extra conflict":     correct + `<meta name="go-import" content="dario.cat/mergo git https://evil.example/repo" name="description">`,
		"malformed tuple":           `<meta name="go-import" content="dario.cat/mergo git">`,
		"unexpected source":         correct + `<meta name="go-source" content="dario.cat/mergo https://example.org https://example.org/tree{/dir} https://example.org/blob{/dir}/{file}#L{line}">`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(content))
			if err == nil {
				err = Validate(route, got)
			}
			if err == nil {
				t.Fatal("regression passed")
			}
		})
	}
}

func TestDuplicateAttributesUseFirstOccurrence(t *testing.T) {
	route := Routes()[0]
	correct := fragment(route)
	cases := []struct {
		name, markup string
		wantError    bool
	}{
		{"first name matches", strings.Replace(correct, `name="go-import"`, `name="go-import" name="description"`, 1), false},
		{"first name unrelated", strings.Replace(correct, `name="go-import"`, `name="description" name="go-import"`, 1), true},
		{"first content matches", strings.TrimSuffix(correct, ">") + ` content="dario.cat/mergo git https://evil.example/repo">`, false},
		{"first content conflicts", strings.Replace(correct, "content=", `content="dario.cat/mergo git https://evil.example/repo" content=`, 1), true},
		{"mixed attribute casing", strings.Replace(correct, `name="go-import"`, `NaMe="go-import" NAME="description"`, 1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tc.markup))
			if err == nil {
				err = Validate(route, got)
			}
			if (err != nil) != tc.wantError {
				t.Fatalf("error %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

func TestCheckDirRequiresEveryFileAndScansOtherRoutes(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		dir := fixture(t)
		write(t, dir, "docs/example.html", `<p>&lt;meta name="go-import"&gt;</p><!-- <meta name="go-source" content="bad"> --><script>let x='<meta name="go-import" content="bad">'</script>`)
		if err := CheckDir(dir); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		dir := fixture(t)
		if err := os.Remove(filepath.Join(dir, "cocoon/index.html")); err != nil {
			t.Fatal(err)
		}
		if err := CheckDir(dir); err == nil || !strings.Contains(err.Error(), "cocoon/index.html") {
			t.Fatalf("got %v, want missing cocoon error", err)
		}
	})
	t.Run("unknown route", func(t *testing.T) {
		dir := fixture(t)
		write(t, dir, "unknown/index.html", fragment(Routes()[0]))
		if err := CheckDir(dir); err == nil || !strings.Contains(err.Error(), "unexpected vanity metadata") {
			t.Fatalf("got %v, want unknown-route error", err)
		}
	})
	t.Run("source on unknown route", func(t *testing.T) {
		dir := fixture(t)
		write(t, dir, "unknown/index.html", `<meta name="go-source" content="example.org/a https://a https://b https://c">`)
		if err := CheckDir(dir); err == nil {
			t.Fatal("unexpected go-source passed")
		}
	})
}
