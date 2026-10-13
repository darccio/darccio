package buildsite

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

const resumeFixtureConfig = `version: 1
source: CV/MAC.json
locale: en
output: cv/index.html
updated: '2026-10-12'
public_email: cc@dario.cat
expiry_dates: [/knowledge/studies/0/finishDate]
date_precision:
  /experience/jobs/0/roles/1/startDate: year
  /knowledge/studies/1/startDate: year
  /knowledge/studies/1/finishDate: year
`

const resumeFixtureSource = `{
 "settings":{"MACVersion":"0.3"},
 "aboutMe":{"profile":{"name":"Dario","surnames":"Castañé <script>","title":"Engineer & maintainer","location":{"municipality":"Terrassa","region":"Catalonia","country":"Spain","address":"NEVER_PROJECT_PRIVATE_ADDRESS"},"description":"Use **Go** and [safe links](https://example.org/?a=1&b=2).\n\n<script>alert('raw')</script>\n\nLiteral {{.Exec \"private\"}}."},
 "relevantLinks":[{"URL":"https://example.org/profile","description":"Public <profile>"}]},
 "careerPreferences":{"secret":"NEVER_PROJECT_PRIVATE_PREFERENCES"},
 "contact":{"email":"NEVER_PROJECT_PRIVATE_CONTACT"},
 "experience":{
  "jobs":[{"organization":{"name":"Example <Org>","URL":"https://example.org/"},"roles":[
   {"name":"Engineer","startDate":"2024-02-01","notes":"Maintains [Go](https://go.dev/).","challenges":[{"description":"Builds safe systems.","actions":["Retains punctuation & intent."]}],"competences":[{"name":"Go"}]},
   {"name":"Earlier role","startDate":"2006-09-01","finishDate":"2010-06-01"}]}],
  "projects":[{"details":{"name":"Project","description":"A preserved project.","URL":"http://example.org/project"},"roles":[{"name":"Maintainer","startDate":"2020-10-01","finishDate":"2021-12-01"}]}]},
 "knowledge":{
  "hardSkills":[{"skill":{"name":"Go"}}],"softSkills":[{"skill":{"name":"Collaboration"}}],
  "languages":[{"name":"English","level":"Professional"},{"name":"Catalan","level":"Native"}],
  "studies":[
   {"studyType":"certification","name":"AWS (historical)","startDate":"2015-12-01","finishDate":"2020-12-01","description":"Does not claim current certification.","institution":{"name":"AWS","description":"Valid until December 2020."}},
   {"studyType":"officialDegree","name":"Computer science","startDate":"2006-09-01","finishDate":"2010-06-01","institution":{"name":"UOC"}}]}
}`

func resumeFixture(t *testing.T) (string, string) {
	t.Helper()
	root, stage := t.TempDir(), t.TempDir()
	put(t, root, ".zas/resume.yml", resumeFixtureConfig)
	put(t, root, "CV/MAC.json", resumeFixtureSource)
	// The assembler copies this static artifact before generating the derived page.
	put(t, stage, "CV/MAC.json", resumeFixtureSource)
	return root, stage
}

func TestResumeAbsentConfigNeverOpensCV(t *testing.T) {
	root, stage := t.TempDir(), t.TempDir()
	// A directory is unreadable as JSON even when tests run as a privileged user.
	if err := os.MkdirAll(filepath.Join(root, "CV", "MAC.json"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := prepareResume(root, stage); err != nil {
		t.Fatalf("missing opt-in should not inspect CV: %v", err)
	}
	entries, err := os.ReadDir(stage)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unconfigured stage changed: %v, %v", entries, err)
	}
}

func TestResumeProjectsPublicFieldsWithSupportedPrecision(t *testing.T) {
	root, stage := resumeFixture(t)
	if err := prepareResume(root, stage); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	pointers := map[string]*html.Node{}
	var inspect func(*html.Node)
	inspect = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, class := range strings.Fields(resumeAttr(n, "class")) {
				counts[class]++
			}
			if pointer := resumeAttr(n, "data-cv-source"); pointer != "" {
				pointers[pointer] = n
			}
			if n.Data == "script" || strings.HasPrefix(resumeAttr(n, "href"), "javascript:") {
				t.Fatal("active unreviewed markup reached CV")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			inspect(c)
		}
	}
	inspect(doc)
	if counts["h-resume"] != 1 || counts["h-card"] != 1 || counts["p-experience"] != 3 || counts["p-education"] != 2 {
		t.Fatalf("missing projected resume entries: %v", counts)
	}
	for pointer, expected := range map[string]string{
		"/experience/jobs/0/roles/0":     "2024-02",
		"/experience/jobs/0/roles/1":     "2006",
		"/experience/projects/0/roles/0": "2020-10",
		"/knowledge/studies/0":           "2015-12",
		"/knowledge/studies/1":           "2006",
	} {
		entry := pointers[pointer]
		if entry == nil || resumeFindDate(entry, "dt-start") != expected {
			t.Fatalf("%s start date: want %s", pointer, expected)
		}
	}
	for _, required := range []string{
		`class="p-name">Dario Castañé &lt;script&gt;`, `class="p-org" href="https://example.org/">Example &lt;Org&gt;`,
		`width="180" height="180"`, `mailto:cc@dario.cat`, `href="/CV/MAC.json"`,
		`class="p-locality">Terrassa`, `class="p-region">Catalonia`, `class="p-country-name">Spain`,
		`"template":false`, `"publish_markdown":false`, `"feed":false`, `"atproto":false`, `"language":"en"`,
		`datetime="2026-10-12"`, `Issued:`, `class="cv-end-label">Expired</span>:`, `Does not claim current certification.`, `Valid until December 2020.`,
		`A preserved project.`, `Retains punctuation &amp; intent.`, `{{.Exec`, `data-cv-language`,
	} {
		if !strings.Contains(string(body), required) {
			t.Errorf("missing %q", required)
		}
	}
	for _, forbidden := range []string{"NEVER_PROJECT_PRIVATE", `<script>alert`, `ZgotmplZ`, `"article"`, `"entry"`, `"date"`, `"updated"`} {
		if strings.Contains(string(body), forbidden) {
			t.Errorf("unexpected rendered value %q", forbidden)
		}
	}
	for _, base := range []string{root, stage} {
		data, err := os.ReadFile(filepath.Join(base, "CV", "MAC.json"))
		if err != nil || string(data) != resumeFixtureSource {
			t.Fatal("preserved MAC JSON changed")
		}
	}
	other := t.TempDir()
	if err := prepareResume(root, other); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(other, "cv", "index.html"))
	if !bytes.Equal(body, second) {
		t.Fatal("resume output is nondeterministic")
	}
}

func resumeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func resumeFindDate(n *html.Node, class string) string {
	if n.Type == html.ElementNode && resumeAttr(n, "class") == class {
		return resumeAttr(n, "datetime")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if value := resumeFindDate(c, class); value != "" {
			return value
		}
	}
	return ""
}

func TestResumeDayPrecisionAndCatalanLabels(t *testing.T) {
	root, stage := resumeFixture(t)
	put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, "2024-02-01", "2024-02-23", 1))
	put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "locale: en", "locale: ca", 1)+"  /experience/jobs/0/roles/0/startDate: day\n")
	if err := prepareResume(root, stage); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
	for _, value := range []string{`lang="ca"`, `datetime="2024-02-23"`, `23 febrer 2024`, `Experiència`, `Emès:`, `class="cv-end-label">Caducat</span>:`} {
		if !strings.Contains(string(data), value) {
			t.Errorf("missing localized/full date %q", value)
		}
	}
}

func TestResumeMonthDisplayRetainsStoredDay(t *testing.T) {
	root, stage := resumeFixture(t)
	source := strings.Replace(resumeFixtureSource, "2024-02-01", "2024-02-23", 1)
	put(t, root, "CV/MAC.json", source)
	if err := prepareResume(root, stage); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
	if !strings.Contains(string(body), `datetime="2024-02">February 2024`) || strings.Contains(string(body), `datetime="2024-02-23"`) {
		t.Fatal("month display did not coarsen the source day")
	}
	stored, _ := os.ReadFile(filepath.Join(root, "CV", "MAC.json"))
	if string(stored) != source {
		t.Fatal("display precision changed the preserved source")
	}
}

func TestResumeCertificationCompletionRequiresExplicitExpiry(t *testing.T) {
	for _, tc := range []struct {
		name, finish, label string
		explicitExpiry      bool
	}{
		{"past completion", "2020-12-01", "Completed", false},
		{"future completion", "2030-12-01", "Completed", false},
		{"reviewed past expiry", "2020-12-01", "Expired", true},
		{"reviewed future expiry", "2030-12-01", "Expires", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, stage := resumeFixture(t)
			config := resumeFixtureConfig
			if !tc.explicitExpiry {
				config = strings.Replace(config, "expiry_dates: [/knowledge/studies/0/finishDate]\n", "", 1)
			}
			put(t, root, ".zas/resume.yml", config)
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, "2020-12-01", tc.finish, 1))
			if err := prepareResume(root, stage); err != nil {
				t.Fatal(err)
			}
			body, _ := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
			if !strings.Contains(string(body), `class="cv-end-label">`+tc.label+`</span>: <time class="dt-end" datetime="`+tc.finish[:7]+`"`) {
				t.Fatalf("certification finish date should display %s", tc.label)
			}
			if !tc.explicitExpiry && (strings.Contains(string(body), `class="cv-end-label">Expired`) || strings.Contains(string(body), `class="cv-end-label">Expires`)) {
				t.Fatal("completion date was presented as expiry without review")
			}
		})
	}
}

func TestResumeRejectsUnsafeOrAmbiguousInputBeforeOutput(t *testing.T) {
	for name, mutate := range map[string]func(*testing.T, string, string){
		"unknown config": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", resumeFixtureConfig+"unknown: true\n")
		},
		"duplicate expiry pointer": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "expiry_dates: [/knowledge/studies/0/finishDate]", "expiry_dates: [/knowledge/studies/0/finishDate, /knowledge/studies/0/finishDate]", 1))
		},
		"non-certification expiry": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "expiry_dates: [/knowledge/studies/0/finishDate]", "expiry_dates: [/knowledge/studies/1/finishDate]", 1))
		},
		"unmatched expiry pointer": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "expiry_dates: [/knowledge/studies/0/finishDate]", "expiry_dates: [/knowledge/studies/999/finishDate]", 1))
		},
		"empty certification expiry": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `"finishDate":"2020-12-01",`, "", 1))
		},
		"fractional version": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "version: 1", "version: 1.5", 1))
		},
		"second config document": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", resumeFixtureConfig+"---\nversion: 1\n")
		},
		"wrong source": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "CV/MAC.json", "CV/private.json", 1))
		},
		"unreviewed email": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "cc@dario.cat", "private@example.org", 1))
		},
		"unknown date pointer": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", resumeFixtureConfig+"  /aboutMe/profile/birthDate: year\n")
		},
		"absent date pointer": func(t *testing.T, root, stage string) {
			put(t, root, ".zas/resume.yml", resumeFixtureConfig+"  /experience/jobs/0/roles/0/finishDate: year\n")
		},
		"invalid date": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, "2024-02-01", "2024-02-31", 1))
		},
		"wrong version": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `"0.3"`, `"0.2"`, 1))
		},
		"duplicate JSON key": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `"MACVersion":"0.3"`, `"MACVersion":"0.3","MACVersion":"0.2"`, 1))
		},
		"trailing JSON": func(t *testing.T, root, stage string) { put(t, root, "CV/MAC.json", resumeFixtureSource+"{}") },
		"unsafe profile URL": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, "https://example.org/profile", "javascript:alert(1)", 1))
		},
		"unsafe Markdown URL": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, "https://go.dev/", "file:///private", 1))
		},
		"Markdown image": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, "A preserved project.", "![track](https://example.org/pixel)", 1))
		},
		"oversize source": func(t *testing.T, root, stage string) {
			put(t, root, "CV/MAC.json", strings.Repeat(" ", resumeSourceLimit+1))
		},
		"oversize config": func(t *testing.T, root, stage string) { put(t, root, ".zas/resume.yml", strings.Repeat(" ", 32<<10+1)) },
		"source symlink": func(t *testing.T, root, stage string) {
			if err := os.Remove(filepath.Join(root, "CV", "MAC.json")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(stage, "CV", "MAC.json"), filepath.Join(root, "CV", "MAC.json")); err != nil {
				t.Fatal(err)
			}
		},
		"output collision": func(t *testing.T, root, stage string) { put(t, stage, "cv/index.html", "existing source") },
	} {
		t.Run(name, func(t *testing.T) {
			root, stage := resumeFixture(t)
			mutate(t, root, stage)
			if err := prepareResume(root, stage); err == nil {
				t.Fatal("invalid input accepted")
			}
			data, err := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
			if !os.IsNotExist(err) && string(data) != "existing source" {
				t.Fatal("failed preparation wrote derived output")
			}
		})
	}
}

func TestResumeURLPolicy(t *testing.T) {
	for _, link := range []string{"https://example.org/a?q=1&b=2", "http://example.org/old", "mailto:cc@dario.cat", "/mergo/", "#projects"} {
		if err := resumeURL(link); err != nil {
			t.Errorf("safe URL %s: %v", link, err)
		}
	}
	for _, link := range []string{"javascript:alert(1)", "data:text/html,active", "//example.org/", "https://user:password@example.org/", "mailto:private@example.org", "mailto:cc@dario.cat?body=private", "/../private", "/%2e%2e/private", "https://example.org/\ntrack"} {
		if err := resumeURL(link); err == nil {
			t.Errorf("unsafe URL accepted: %s", link)
		}
	}
}

func TestResumeInvalidBuildPreservesPreviousDeployment(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/resume.yml", resumeFixtureConfig)
	put(t, root, "CV/MAC.json", `{"settings":`)
	marker := filepath.Join(t.TempDir(), "generator-ran")
	zas := generator(t, "touch '"+strings.ReplaceAll(marker, "'", "'\\''")+"'\n"+ordinaryOutput)
	if err := Build(context.Background(), Options{Root: root, Zas: zas}); err == nil || !strings.Contains(err.Error(), "assemble reviewed CV") {
		t.Fatalf("expected pre-generation CV failure: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("generator ran before CV validation")
	}
	data, err := os.ReadFile(filepath.Join(root, ".zas", "deploy", "previous.txt"))
	if err != nil || string(data) != "last successful deployment" {
		t.Fatal("failed CV build damaged previous deployment")
	}
}

func TestResumeMetadataCannotEscapeComment(t *testing.T) {
	root, stage := resumeFixture(t)
	var source map[string]any
	if err := json.Unmarshal([]byte(resumeFixtureSource), &source); err != nil {
		t.Fatal(err)
	}
	profile := source["aboutMe"].(map[string]any)["profile"].(map[string]any)
	profile["title"] = `Engineer --> <script> {{.Exec "secret"}}`
	data, _ := json.Marshal(source)
	put(t, root, "CV/MAC.json", string(data))
	if err := prepareResume(root, stage); err != nil {
		t.Fatal(err)
	}
	output, _ := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
	end := bytes.Index(output, []byte("-->"))
	if end < 0 {
		t.Fatal("missing metadata terminator")
	}
	var metadata map[string]any
	if err := yamlMetadata(output[:end], &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["template"] != false || metadata["title"] != "Curriculum vitae" || metadata["description"] != profile["title"] {
		t.Fatalf("escaped metadata changed meaning: %v", metadata)
	}
}

func TestResumePreservesCloudflareEmailBoundaries(t *testing.T) {
	root, stage := resumeFixture(t)
	if err := prepareResume(root, stage); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(filepath.Join(stage, "cv", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	want := `<!--email_off--><a class="u-email" href="mailto:cc@dario.cat">cc@dario.cat</a><!--/email_off-->`
	if !bytes.Contains(output, []byte(want)) {
		t.Fatal("Cloudflare email exclusions were removed by template rendering")
	}
}

func yamlMetadata(comment []byte, target any) error {
	return json.Unmarshal(bytes.TrimSpace(bytes.TrimPrefix(comment, []byte("<!--"))), target)
}
