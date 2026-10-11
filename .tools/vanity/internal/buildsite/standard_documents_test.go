package buildsite

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const standardFixtureURI = standardDocumentPrefix + "3jzfcijpj2z2a"
const standardFixtureDate = "2018-03-28T23:26:00+02:00"

func TestStandardDocumentManifestValidatesExactIdentityAndCanonicalPaths(t *testing.T) {
	valid := `{"/posts/reviewed/":"` + standardFixtureURI + `"}`
	manifest, err := decodeStandardDocumentManifest([]byte(valid))
	if err != nil || manifest["/posts/reviewed/"] != standardFixtureURI {
		t.Fatalf("valid manifest: %v, %v", manifest, err)
	}
	for _, value := range []string{"", " \n", "{}"} {
		if manifest, err := decodeStandardDocumentManifest([]byte(value)); err != nil || len(manifest) != 0 {
			t.Fatalf("empty manifest %q: %v, %v", value, manifest, err)
		}
	}
	badURIs := []string{
		strings.Replace(standardFixtureURI, "at://", "https://", 1),
		strings.Replace(standardFixtureURI, "aj77r5uwt72o6oimdjfplqoz", "zzzzzzzzzzzzzzzzzzzzzzzz", 1),
		strings.Replace(standardFixtureURI, "site.standard.document", "site.standard.publication", 1),
		standardFixtureURI + "?x=1", standardFixtureURI + "#fragment", standardFixtureURI + "/",
		standardFixtureURI + "\"", standardFixtureURI + "\n", standardFixtureURI + "\x00",
		standardDocumentPrefix + "self", standardDocumentPrefix + "3JZFCIJPJ2Z2A",
		standardDocumentPrefix + "zzzzzzzzzzzzz", standardDocumentPrefix + "3jzf-cij-pj2z-2a",
	}
	for _, uri := range badURIs {
		t.Run(uri, func(t *testing.T) {
			data, _ := json.Marshal(map[string]string{"/posts/reviewed/": uri})
			if _, err := decodeStandardDocumentManifest(data); err == nil {
				t.Fatal("invalid AT URI accepted")
			}
		})
	}
	for _, pathname := range []string{
		"/posts/reviewed", "posts/reviewed/", "/posts/../contact/", "/posts//reviewed/", "/posts/reviewed/?x=1",
		"/posts/%72eviewed/", "/posts/reviewed/#x", "/CV/private/", "/mergo/", "/about/", "/en/posts/../private/",
		"/posts/reviewed\x00/", "/posts/reviewed\\nested/", "/posts/reviewed.html/", "/posts/private/page/", "/posts/cv/page/",
	} {
		data, _ := json.Marshal(map[string]string{pathname: standardFixtureURI})
		if _, err := decodeStandardDocumentManifest(data); err == nil {
			t.Errorf("unsafe or non-post canonical path accepted: %q", pathname)
		}
	}
	for _, data := range []string{
		"null", "[]", valid + "{}", strings.TrimSuffix(valid, "}"),
		`{"/posts/reviewed/":"` + standardFixtureURI + `","/posts/reviewed/":"` + standardFixtureURI + `"}`,
		`{"/posts/reviewed/":"` + standardFixtureURI + `","/en/posts/reviewed/":"` + standardFixtureURI + `"}`,
		`{"/posts/reviewed/":null}`, `{"/posts/reviewed/":{}}`,
	} {
		if _, err := decodeStandardDocumentManifest([]byte(data)); err == nil {
			t.Errorf("malformed or colliding manifest accepted: %s", data)
		}
	}
}

func TestStandardDocumentOfflineAssemblyPreservesUnrelatedBytes(t *testing.T) {
	root, stage, deploy := standardDocumentFixture(t)
	before := httpWorkerDeploySnapshot(t, deploy)
	if err := finalizeStandardDocuments(root, stage, deploy); err != nil {
		t.Fatal(err)
	}
	after := httpWorkerDeploySnapshot(t, deploy)
	for file, body := range before {
		if file == "posts/reviewed/index.html" {
			continue
		}
		if after[file] != body {
			t.Errorf("unapproved page or asset changed: %s", file)
		}
	}
	updated := []byte(after["posts/reviewed/index.html"])
	if _, linked, err := standardDocumentHead(updated, standardFixtureURI); err != nil || !linked {
		t.Fatalf("final document link did not survive serialization: %v", err)
	}
	link := "\n<link rel=\"site.standard.document\" href=\"" + standardFixtureURI + "\">\n"
	if strings.Count(string(updated), link) != 1 || bytes.Contains(updated, []byte("ZgotmplZ")) {
		t.Fatal("AT URI was sanitized or duplicated")
	}
	if strings.Replace(string(updated), link, "", 1) != before["posts/reviewed/index.html"] {
		t.Fatal("insertion reserialized unrelated HTML bytes")
	}
	if err := finalizeStandardDocuments(root, stage, deploy); err != nil {
		t.Fatalf("idempotent second offline assembly: %v", err)
	}
	if latest := httpWorkerDeploySnapshot(t, deploy); !reflect.DeepEqual(after, latest) {
		t.Fatal("second offline assembly made changes")
	}
}

func TestStandardDocumentMissingOrEmptyManifestRequiresNothingElse(t *testing.T) {
	for _, value := range []string{"missing", "", "{}"} {
		t.Run(value, func(t *testing.T) {
			stage := t.TempDir()
			if value != "missing" {
				put(t, stage, ".zas/atproto.json", value)
			}
			if err := finalizeStandardDocuments("absent-root", stage, "absent-deploy"); err != nil {
				t.Fatalf("optional state changed legacy build: %v", err)
			}
		})
	}
}

func TestStandardDocumentRejectsBadHeadLinksBeforeWriting(t *testing.T) {
	body := `<html><head><script>const text = "</head>";</script></head><body><h1>Public article</h1></body></html>`
	position, linked, err := standardDocumentHead([]byte(body), standardFixtureURI)
	if err != nil || linked || position != strings.Index(body, "</head><body>") {
		t.Fatalf("script closing-head text confused insertion: position=%d linked=%t err=%v", position, linked, err)
	}
	link := `<link rel="site.standard.document" href="` + standardFixtureURI + `">`
	for name, content := range map[string]string{
		"sanitized":               `<head><link rel="site.standard.document" href="#ZgotmplZ"></head>`,
		"duplicate":               "<head>" + link + link + "</head>",
		"body":                    "<head></head><body>" + link + "</body>",
		"template":                "<head><template>" + link + "</template></head>",
		"noscript":                "<head><noscript>" + link + "</noscript></head>",
		"duplicate href":          `<head><link rel="site.standard.document" href="` + standardFixtureURI + `" href="` + standardFixtureURI + `"></head>`,
		"hidden duplicate rel":    `<head><link rel="site.standard.document" rel="stylesheet" href="` + standardFixtureURI + `"></head>`,
		"different URI":           `<head><link rel="site.standard.document" href="` + standardDocumentPrefix + `3jzfcijpj2z2b"></head>`,
		"duplicate head":          "<head></head><head>" + link + "</head>",
		"missing head":            "<body>Public article</body>",
		"browser body relocation": "<head><div>Body</div>" + link + "</head>",
		"browser text relocation": "<head>Body text" + link + "</head>",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := standardDocumentHead([]byte(content), standardFixtureURI); err == nil {
				t.Fatal("invalid or inactive document link accepted")
			}
		})
	}
	if _, _, err := standardDocumentHead([]byte("<head>"+link+"</head>"), ""); err == nil {
		t.Fatal("unapproved article retained a document link")
	}
}

func TestStandardDocumentEligibilityAndAllowlistFailuresAreAtomic(t *testing.T) {
	cases := map[string]func(*testing.T, string, string, string){
		"no approval": func(t *testing.T, root, stage, deploy string) {
			put(t, root, ".tools/atproto/documents-allowlist.json", `{"version":1,"documents":[]}`)
		},
		"wrong reviewed instant": func(t *testing.T, root, stage, deploy string) {
			standardDocumentApprovalFixture(t, root, "/posts/reviewed/", "2018-03-29T23:26:00+02:00")
		},
		"partial reviewed date": func(t *testing.T, root, stage, deploy string) {
			standardDocumentApprovalFixture(t, root, "/posts/reviewed/", "2018-03-28")
		},
		"duplicate approval": func(t *testing.T, root, stage, deploy string) {
			entry := `{"path":"/posts/reviewed/","publishedAt":"` + standardFixtureDate + `"}`
			put(t, root, ".tools/atproto/documents-allowlist.json", `{"version":1,"documents":[`+entry+`,`+entry+`]}`)
		},
		"different canonical": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "posts/reviewed/index.html", `<html lang="ca"><head><link rel="canonical" href="https://dario.cat/posts/other/"></head></html>`)
		},
		"unapproved link": func(t *testing.T, root, stage, deploy string) {
			put(t, deploy, "posts/unreviewed/index.html", `<head><link rel="site.standard.document" href="`+standardFixtureURI+`"></head>`)
		},
		"missing generated page": func(t *testing.T, root, stage, deploy string) {
			os.Remove(filepath.Join(deploy, "posts/reviewed/index.html"))
		},
		"source symlink": func(t *testing.T, root, stage, deploy string) {
			os.Remove(filepath.Join(stage, "posts/reviewed/index.md"))
			if err := os.Symlink("/missing-private-source", filepath.Join(stage, "posts/reviewed/index.md")); err != nil {
				t.Skip(err)
			}
		},
	}
	for _, flag := range []string{"article: false", "draft: true", "private: true", "publish: false", "feed: false", "atproto: false", "date_precision: day"} {
		cases[flag] = func(t *testing.T, root, stage, deploy string) {
			metadata := "article: true\ndate: \"" + standardFixtureDate + "\"\n" + flag + "\n"
			if flag == "article: false" {
				metadata = strings.ReplaceAll(metadata, "article: true\n", "")
			}
			put(t, stage, "posts/reviewed/index.md", "<!--\n"+metadata+"-->\n# Source\n")
		}
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			root, stage, deploy := standardDocumentFixture(t)
			mutate(t, root, stage, deploy)
			before := httpWorkerDeploySnapshot(t, deploy)
			if err := finalizeStandardDocuments(root, stage, deploy); err == nil {
				t.Fatal("invalid document publication state accepted")
			}
			if after := httpWorkerDeploySnapshot(t, deploy); !reflect.DeepEqual(before, after) {
				t.Fatal("rejected document state changed deployment bytes")
			}
		})
	}
}

func TestStandardDocumentSourceDateComparesInstants(t *testing.T) {
	root, stage, deploy := standardDocumentFixture(t)
	standardDocumentApprovalFixture(t, root, "/posts/reviewed/", "2018-03-28T21:26:00Z")
	if err := finalizeStandardDocuments(root, stage, deploy); err != nil {
		t.Fatalf("equivalent reviewed timezone rejected: %v", err)
	}
}

func TestStandardDocumentBuildFailurePreservesPreviousDeployment(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	put(t, root, "posts/reviewed/index.md", "<!--\narticle: true\ndate: \""+standardFixtureDate+"\"\n-->\n# Reviewed\n")
	put(t, root, ".zas/atproto.json", `{"/posts/reviewed/":"`+standardFixtureURI+`"}`)
	standardDocumentApprovalFixture(t, root, "/posts/reviewed/", standardFixtureDate)
	script := generator(t, ordinaryOutput+`
mkdir -p .zas/deploy/posts/reviewed
printf '<html lang="ca"><head><link rel="canonical" href="https://dario.cat/posts/reviewed/"></head><body>Final HTML</body></html>' > .zas/deploy/posts/reviewed/index.html
`)
	if err := Build(context.Background(), Options{Root: root, Zas: script, Output: io.Discard}); err != nil {
		t.Fatal(err)
	}
	deploy := filepath.Join(root, ".zas", "deploy")
	before := httpWorkerDeploySnapshot(t, deploy)
	if !strings.Contains(before["posts/reviewed/index.html"], standardFixtureURI) {
		t.Fatal("build did not integrate the offline document assembler")
	}
	put(t, root, ".zas/atproto.json", `{"/posts/reviewed/":"at://invalid"}`)
	if err := Build(context.Background(), Options{Root: root, Zas: script, Output: io.Discard}); err == nil || !strings.Contains(err.Error(), "Standard.site documents rejected") {
		t.Fatalf("invalid manifest did not fail before promotion: %v", err)
	}
	if after := httpWorkerDeploySnapshot(t, deploy); !reflect.DeepEqual(before, after) {
		t.Fatal("failed assembly replaced the previous deployment")
	}
}

func standardDocumentFixture(t *testing.T) (root, stage, deploy string) {
	t.Helper()
	root, stage, deploy = t.TempDir(), t.TempDir(), t.TempDir()
	put(t, stage, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n")
	put(t, stage, ".zas/atproto.json", `{"/posts/reviewed/":"`+standardFixtureURI+`"}`)
	put(t, stage, "posts/reviewed/index.md", "<!--\narticle: true\ndate: \""+standardFixtureDate+"\"\n-->\n# Source\n")
	put(t, deploy, "posts/reviewed/index.html", `<html lang="ca"><head><link rel="canonical" href="https://dario.cat/posts/reviewed/"><script>const text = "</head>";</script></head><body>Final HTML</body></html>`)
	put(t, deploy, "posts/reviewed/index.md", "# Rendered article\n")
	put(t, deploy, "posts/unreviewed/index.html", "<head></head><body>Unreviewed public article</body>")
	put(t, deploy, "posts/delete-facebook-3/index.html", "<head></head><body>Existing live draft</body>")
	put(t, deploy, "mergo/index.html", "raw vanity import bytes")
	put(t, deploy, "CV/private.html", "never parse CV")
	put(t, deploy, "docs/site-plan/private.html", "never parse private operations")
	standardDocumentApprovalFixture(t, root, "/posts/reviewed/", standardFixtureDate)
	return root, stage, deploy
}

func standardDocumentApprovalFixture(t *testing.T, root, pathname, published string) {
	t.Helper()
	data, err := json.Marshal(struct {
		Version   int                        `json:"version"`
		Documents []standardDocumentApproval `json:"documents"`
	}{1, []standardDocumentApproval{{Path: pathname, PublishedAt: published}}})
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".tools/atproto/documents-allowlist.json", string(data))
}
