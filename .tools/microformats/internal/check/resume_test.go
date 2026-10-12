package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const resumeFixtureConfig = `version: 1
source: CV/MAC.json
locale: en
output: cv/index.html
updated: 2026-10-12
public_email: cc@dario.cat
date_precision:
  /knowledge/studies/0/startDate: year
  /knowledge/studies/0/finishDate: year
  /knowledge/studies/1/startDate: day
`

const resumeFixtureSource = `{
	"settings": {"MACVersion": "0.3"},
  "aboutMe": {"profile": {
    "name": "Dario", "surnames": "Castañé", "title": "Senior Go Engineer",
    "description": "A **public** summary with [work](/projects/).\n\nSecond paragraph.",
    "location": {"municipality": "Terrassa", "region": "Catalonia", "country": "Spain"}
  }},
  "experience": {
    "jobs": [{"organization": {"name": "Datadog"}, "roles": [
      {"name": "Senior Engineer", "startDate": "2023-08-01"},
      {"name": "Engineering Manager", "startDate": "2020-04-01", "finishDate": "2023-07-01"}
    ]}],
    "projects": [{"details": {"name": "Zas"}, "roles": [
      {"name": "Maintainer", "startDate": "2013-03-01", "competences": [{"name": "CSS"}]}
    ]}]
  },
  "knowledge": {
    "studies": [
      {"name": "Bachelor", "institution": {"name": "UOC"}, "startDate": "2006-09-01", "finishDate": "2010-06-01"},
      {"name": "Reviewed Certificate", "institution": {"name": "Example Issuer"}, "startDate": "2023-01-15"}
    ],
    "hardSkills": [{"skill": {"name": "Go"}}, {"skill": {"name": "GitHub"}}],
    "softSkills": [{"skill": {"name": "Leadership"}}],
    "languages": [{"name": "English", "level": "Full professional proficiency"}, {"name": "Catalan", "level": "Native or bilingual proficiency"}]
  },
  "careerPreferences": {"contact": {"privateTestValue": "not rendered"}}
}
`

func resumeFixtureHTML(t *testing.T) string {
	t.Helper()
	data := jsonFixture(t)
	data["@graph"] = data["@graph"].([]any)[:1]
	page := strings.ReplaceAll(structuredDocument(t, marshalFixture(t, data)), examplePost().URL, homeURL+"cv/")
	page = strings.Replace(page, `property="og:type" content="article"`, `property="og:type" content="website"`, 1)
	start, end := strings.Index(page, "<main>"), strings.Index(page, "</main>")
	if start < 0 || end < start {
		t.Fatal("shared metadata fixture lost main")
	}
	const body = `<main><section class="h-resume" lang="en">
<h1 class="p-name">Dario Castañé</h1>
<div class="p-contact h-card">
<a class="p-name u-url u-uid" href="https://dario.cat/">Dario Castañé</a>
<img class="u-photo" src="/static/avatar.webp" width="180" height="180" alt="">
<p class="p-job-title">Senior Go Engineer</p>
<p><span class="p-locality">Terrassa</span>, <span class="p-region">Catalonia</span>, <span class="p-country-name">Spain</span></p>
<a class="u-email" href="mailto:cc@dario.cat">cc@dario.cat</a>
</div>
<div class="p-summary"><p>A <strong>public</strong> summary with <a href="/projects/">work</a>.</p>
<p>Second paragraph.</p>
</div>
<time class="dt-updated" datetime="2026-10-12">12 Oct 2026</time>
<a href="/CV/MAC.json">Original JSON</a>
<article class="p-experience h-event" data-cv-source="/experience/jobs/0/roles/0">
<h3 class="p-name">Senior Engineer</h3><span class="p-org">Datadog</span>
<time class="dt-start" datetime="2023-08">Aug 2023</time>
</article>
<article class="p-experience h-event" data-cv-source="/experience/jobs/0/roles/1">
<h3 class="p-name">Engineering Manager</h3><span class="p-org">Datadog</span>
<time class="dt-start" datetime="2020-04">Apr 2020</time><time class="dt-end" datetime="2023-07">Jul 2023</time>
</article>
<article class="p-experience h-event" data-cv-source="/experience/projects/0/roles/0">
<h3 class="p-name">Maintainer</h3><span class="p-org">Zas</span><time class="dt-start" datetime="2013-03">Mar 2013</time>
<span class="p-skill">CSS</span>
</article>
<article class="p-education h-event" data-cv-source="/knowledge/studies/0">
<h3 class="p-name">Bachelor</h3><span class="p-org">UOC</span><time class="dt-start" datetime="2006">2006</time><time class="dt-end" datetime="2010">2010</time>
</article>
<article class="p-education h-event" data-cv-source="/knowledge/studies/1">
<h3 class="p-name">Reviewed Certificate</h3><span class="p-org">Example Issuer</span><time class="dt-start" datetime="2023-01-15">15 Jan 2023</time>
</article>
<div data-cv-skills><span class="p-skill">Go</span><span class="p-skill">GitHub</span><span class="p-skill">Leadership</span></div>
<ul><li data-cv-language="English"><span class="cv-language-name">English</span> (<span class="cv-language-level">Full professional proficiency</span>)</li>
<li data-cv-language="Catalan"><span class="cv-language-name">Catalan</span> (<span class="cv-language-level">Native or bilingual proficiency</span>)</li></ul>
</section></main>`
	return page[:start] + body + page[end+len("</main>"):]
}

func resumeFixture(t *testing.T) (string, string, *resumeExpectation) {
	t.Helper()
	root, deploy := t.TempDir(), t.TempDir()
	put(t, root, ".zas/resume.yml", resumeFixtureConfig)
	put(t, root, "CV/MAC.json", resumeFixtureSource)
	put(t, deploy, "CV/MAC.json", resumeFixtureSource)
	put(t, deploy, resumeOutput, resumeFixtureHTML(t))
	expected, err := loadResume(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, deploy, expected
}

func TestResumeSourceAndPrecisionMatchOutput(t *testing.T) {
	_, deploy, expected := resumeFixture(t)
	if err := validateResume(deploy, expected); err != nil {
		t.Fatal(err)
	}
	if len(expected.Events) != 5 || expected.Events[3].Start != "2006" || expected.Events[3].End != "2010" {
		t.Fatal("preserved UOC months must render at configured year precision")
	}
	if err := validateJSONLDPages(deploy, inventory{}, true); err != nil {
		t.Fatal(err)
	}
}

func TestResumeSourceToOutputRegressions(t *testing.T) {
	cases := []struct{ name, from, to string }{
		{"resume root missing", `class="h-resume"`, `class="resume"`},
		{"wrong owner", `<h1 class="p-name">Dario Castañé</h1>`, `<h1 class="p-name">Different Person</h1>`},
		{"wrong summary", `Second paragraph.`, `A different paragraph.`},
		{"summary destination drift", `href="/projects/"`, `href="/other-projects/"`},
		{"wrong job title", `class="p-job-title">Senior Go Engineer`, `class="p-job-title">Principal Engineer`},
		{"wrong public locality", `class="p-locality">Terrassa`, `class="p-locality">Barcelona`},
		{"wrong public region", `class="p-region">Catalonia`, `class="p-region">A different region`},
		{"wrong public country", `class="p-country-name">Spain`, `class="p-country-name">A different country`},
		{"wrong public email", `class="u-email" href="mailto:cc@dario.cat"`, `class="u-email" href="mailto:other@example.test"`},
		{"wrong contact identity", `class="p-name u-url u-uid" href="https://dario.cat/"`, `class="p-name u-url u-uid" href="https://example.test/"`},
		{"wrong role", `<h3 class="p-name">Senior Engineer</h3>`, `<h3 class="p-name">Principal Engineer</h3>`},
		{"wrong organization", `<span class="p-org">Zas</span>`, `<span class="p-org">Different Project</span>`},
		{"missing project event", `class="p-experience h-event" data-cv-source="/experience/projects/0/roles/0"`, `class="p-experience" data-cv-source="/experience/projects/0/roles/0"`},
		{"wrong source pointer", `data-cv-source="/experience/jobs/0/roles/0"`, `data-cv-source="/experience/jobs/99/roles/0"`},
		{"duplicate source pointer", `data-cv-source="/experience/jobs/0/roles/1"`, `data-cv-source="/experience/jobs/0/roles/0"`},
		{"month precision inflated", `datetime="2023-08"`, `datetime="2023-08-01"`},
		{"year precision inflated", `datetime="2006"`, `datetime="2006-09"`},
		{"wrong finish date", `datetime="2023-07"`, `datetime="2023-08"`},
		{"invented finish", `<time class="dt-start" datetime="2023-08">Aug 2023</time>`, `<time class="dt-start" datetime="2023-08">Aug 2023</time><time class="dt-end" datetime="2026-10">Oct 2026</time>`},
		{"wrong study classification", `class="p-education h-event" data-cv-source="/knowledge/studies/0"`, `class="p-experience h-event" data-cv-source="/knowledge/studies/0"`},
		{"wrong study", `<h3 class="p-name">Bachelor</h3>`, `<h3 class="p-name">Doctorate</h3>`},
		{"wrong skill", `class="p-skill">GitHub`, `class="p-skill">A different skill`},
		{"wrong role competence", `class="p-skill">CSS`, `class="p-skill">A different competence`},
		{"unscoped global skill list", `<div data-cv-skills>`, `<div>`},
		{"wrong language level", `class="cv-language-level">Full professional proficiency`, `class="cv-language-level">Native proficiency`},
		{"wrong review date", `class="dt-updated" datetime="2026-10-12"`, `class="dt-updated" datetime="2026-10-11"`},
		{"wrong locale", `class="h-resume" lang="en"`, `class="h-resume" lang="ca"`},
		{"missing original JSON", `href="/CV/MAC.json"`, `href="/CV/other.json"`},
		{"entry exposure", `class="h-resume" lang="en"`, `class="h-resume h-entry" lang="en"`},
		{"script in summary", `Second paragraph.`, `Second paragraph.<script>alert(1)</script>`},
		{"wrong canonical", `rel="canonical" href="https://dario.cat/cv/"`, `rel="canonical" href="https://dario.cat/cv/index.html"`},
		{"wrong shared identity", `"name":"Dario Castañé"`, `"name":"Different Person"`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, deploy, expected := resumeFixture(t)
			input := resumeFixtureHTML(t)
			if !strings.Contains(input, test.from) {
				t.Fatalf("fixture mutation lost %q", test.from)
			}
			put(t, deploy, resumeOutput, strings.Replace(input, test.from, test.to, 1))
			if err := validateResume(deploy, expected); err == nil {
				t.Fatal("resume/source disagreement accepted")
			}
		})
	}
}

func TestResumePreservesJSONAndOmitsMarkdownTwin(t *testing.T) {
	for _, name := range []string{"changed JSON bytes", "Markdown twin", "missing HTML", "symlinked output"} {
		t.Run(name, func(t *testing.T) {
			_, deploy, expected := resumeFixture(t)
			switch name {
			case "changed JSON bytes":
				put(t, deploy, "CV/MAC.json", strings.TrimSpace(resumeFixtureSource))
			case "Markdown twin":
				put(t, deploy, "cv/index.md", "must not publish the source CV as a Markdown corpus")
			case "missing HTML":
				if err := os.Remove(filepath.Join(deploy, resumeOutput)); err != nil {
					t.Fatal(err)
				}
			case "symlinked output":
				path := filepath.Join(deploy, resumeOutput)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(deploy, "CV", "MAC.json"), path); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateResume(deploy, expected); err == nil {
				t.Fatal("unsafe CV artifact accepted")
			}
		})
	}
}

func TestResumeConfigurationOptInAndStrictness(t *testing.T) {
	root := t.TempDir()
	put(t, root, "CV/MAC.json", "invalid private data must not be opened")
	if err := os.Chmod(filepath.Join(root, "CV", "MAC.json"), 0); err != nil {
		t.Fatal(err)
	}
	if expected, err := loadResume(root); err != nil || expected != nil {
		t.Fatalf("absent configuration processed CV: %v", err)
	}
	if err := validateResume(t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, from, to string }{
		{"unsupported version", "version: 1", "version: 2"},
		{"noninteger version", "version: 1", "version: 1.5"},
		{"different source", "source: CV/MAC.json", "source: other.json"},
		{"source traversal", "source: CV/MAC.json", "source: ../CV/MAC.json"},
		{"legacy output", "output: cv/index.html", "output: CV/index.html"},
		{"different output", "output: cv/index.html", "output: cv/nested/index.html"},
		{"unsupported locale", "locale: en", "locale: es"},
		{"wrong email", "public_email: cc@dario.cat", "public_email: other@example.test"},
		{"invalid review day", "updated: 2026-10-12", "updated: 2026-02-30"},
		{"unknown option", "version: 1", "version: 1\nprivate: true"},
		{"duplicate key", "version: 1", "version: 1\nversion: 1"},
		{"multiple documents", "version: 1", "version: 1\n---\nversion: 1"},
		{"unused precision", "/knowledge/studies/0/startDate: year", "/aboutMe/profile/name: year"},
		{"missing date precision", "/knowledge/studies/0/startDate: year", "/experience/jobs/0/roles/0/finishDate: year"},
		{"invalid precision", "/knowledge/studies/0/startDate: year", "/knowledge/studies/0/startDate: guessed"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root, _, _ := resumeFixture(t)
			put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, test.from, test.to, 1))
			if _, err := loadResume(root); err == nil {
				t.Fatal("invalid resume configuration accepted")
			}
		})
	}
	t.Run("default public email", func(t *testing.T) {
		root, _, _ := resumeFixture(t)
		put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "public_email: cc@dario.cat\n", "", 1))
		if expected, err := loadResume(root); err != nil || expected.Config.PublicEmail != email {
			t.Fatalf("default contact failed: %v", err)
		}
	})
}

func TestResumeRejectsMalformedOrIndirectSources(t *testing.T) {
	for _, name := range []string{"invalid JSON", "duplicate JSON key", "wrong field type", "unsupported MAC version", "invalid source date", "unreadable JSON", "symlinked JSON", "symlinked configuration", "symlinked source directory"} {
		t.Run(name, func(t *testing.T) {
			root, _, _ := resumeFixture(t)
			sourcePath := filepath.Join(root, "CV", "MAC.json")
			switch name {
			case "invalid JSON":
				put(t, root, "CV/MAC.json", "invalid")
			case "duplicate JSON key":
				put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `"name": "Dario"`, `"name":"Other","name":"Dario"`, 1))
			case "wrong field type":
				put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `"startDate": "2023-08-01"`, `"startDate": 2023`, 1))
			case "unsupported MAC version":
				put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `"MACVersion": "0.3"`, `"MACVersion": "0.4"`, 1))
			case "invalid source date":
				put(t, root, "CV/MAC.json", strings.Replace(resumeFixtureSource, `2023-08-01`, `2023-02-30`, 1))
			case "unreadable JSON":
				if err := os.Chmod(sourcePath, 0); err != nil {
					t.Fatal(err)
				}
			case "symlinked JSON":
				put(t, root, "other.json", resumeFixtureSource)
				if err := os.Remove(sourcePath); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, "other.json"), sourcePath); err != nil {
					t.Fatal(err)
				}
			case "symlinked configuration":
				put(t, root, "resume.yml", resumeFixtureConfig)
				config := filepath.Join(root, ".zas", "resume.yml")
				if err := os.Remove(config); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, "resume.yml"), config); err != nil {
					t.Fatal(err)
				}
			case "symlinked source directory":
				if err := os.Rename(filepath.Join(root, "CV"), filepath.Join(root, "otherCV")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, "otherCV"), filepath.Join(root, "CV")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := loadResume(root); err == nil {
				t.Fatal("malformed or indirect source accepted")
			}
		})
	}
}

func TestResumeMonthDisplayPreservesKnownSourceDay(t *testing.T) {
	root, deploy, _ := resumeFixture(t)
	changed := strings.Replace(resumeFixtureSource, `2023-08-01`, `2023-08-15`, 1)
	put(t, root, "CV/MAC.json", changed)
	put(t, deploy, "CV/MAC.json", changed)
	expected, err := loadResume(root)
	if err != nil {
		t.Fatal(err)
	}
	if expected.Events[0].Start != "2023-08" || expected.CV.Experience.Jobs[0].Roles[0].StartDate != "2023-08-15" {
		t.Fatal("display precision must not rewrite the original source date")
	}
	if err := validateResume(deploy, expected); err != nil {
		t.Fatal(err)
	}
}

func TestResumeJSONLDAllowsOnlyConfiguredExactOutput(t *testing.T) {
	_, deploy, _ := resumeFixture(t)
	if err := validateJSONLDPages(deploy, inventory{}); err == nil || !strings.Contains(err.Error(), "unexpected CV HTML output") {
		t.Fatalf("unconfigured CV output accepted: %v", err)
	}
	for _, file := range []string{"CV/index.html", "cV/index.html", "cv/nested/index.html", "en/cv/index.html", "cv/index.HTML", "cv/other.html"} {
		t.Run(file, func(t *testing.T) {
			_, deploy, _ := resumeFixture(t)
			put(t, deploy, file, "must be rejected before reading CV content")
			if err := os.Chmod(filepath.Join(deploy, filepath.FromSlash(file)), 0); err != nil {
				t.Fatal(err)
			}
			err := validateJSONLDPages(deploy, inventory{}, true)
			if err == nil || !strings.Contains(err.Error(), "unexpected CV HTML output") {
				t.Fatalf("unexpected CV filename must be rejected before its content: %v", err)
			}
		})
	}
	t.Run("eligible filename still requires ordinary Person graph", func(t *testing.T) {
		_, deploy, _ := resumeFixture(t)
		data := jsonFixture(t)
		input := resumeFixtureHTML(t)
		start := strings.Index(input, `<script type="application/ld+json">`) + len(`<script type="application/ld+json">`)
		end := start + strings.Index(input[start:], "</script>")
		put(t, deploy, resumeOutput, input[:start]+marshalFixture(t, data)+input[end:])
		if err := validateJSONLDPages(deploy, inventory{}, true); err == nil {
			t.Fatal("resume exposed as BlogPosting")
		}
	})
}

func TestResumeLocaleAndSourceTextAreCheckedIndependently(t *testing.T) {
	root, deploy, _ := resumeFixture(t)
	put(t, root, ".zas/resume.yml", strings.Replace(resumeFixtureConfig, "locale: en", "locale: ca", 1))
	input := strings.ReplaceAll(resumeFixtureHTML(t), `lang="en"`, `lang="ca"`)
	put(t, deploy, resumeOutput, input)
	expected, err := loadResume(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateResume(deploy, expected); err != nil {
		t.Fatal(err)
	}
	var source map[string]any
	if err := json.Unmarshal([]byte(resumeFixtureSource), &source); err != nil {
		t.Fatal(err)
	}
	source["knowledge"].(map[string]any)["languages"].([]any)[0].(map[string]any)["level"] = "A new reviewed level"
	changed := marshalFixture(t, source)
	put(t, root, "CV/MAC.json", changed)
	put(t, deploy, "CV/MAC.json", changed)
	expected, err = loadResume(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateResume(deploy, expected); err == nil {
		t.Fatal("changed JSON language level accepted with stale HTML")
	}
}

func TestResumeCertificationCompletionAndReviewedExpiry(t *testing.T) {
	for _, locale := range []string{"en", "ca"} {
		for _, test := range []struct {
			name, finish, label string
			expiry              bool
		}{
			{"ordinary completion", "2024-04-01", "Completed", false},
			{"historical reviewed expiry", "2024-04-01", "Expired", true},
			{"future reviewed expiry", "2030-01-01", "Expires", true},
		} {
			t.Run(locale+"/"+test.name, func(t *testing.T) {
				root, deploy, _ := resumeFixture(t)
				source := strings.Replace(resumeFixtureSource, `"name": "Reviewed Certificate"`, `"studyType": "certification", "finishDate": "`+test.finish+`", "name": "Reviewed Certificate"`, 1)
				put(t, root, "CV/MAC.json", source)
				put(t, deploy, "CV/MAC.json", source)
				config := strings.Replace(resumeFixtureConfig, "locale: en", "locale: "+locale, 1)
				if test.expiry {
					config += "expiry_dates:\n  - /knowledge/studies/1/finishDate\n"
				}
				put(t, root, ".zas/resume.yml", config)
				label := test.label
				if locale == "ca" {
					label = map[string]string{"Completed": "Completat", "Expired": "Caducat", "Expires": "Caduca"}[label]
				}
				input := strings.ReplaceAll(resumeFixtureHTML(t), `lang="en"`, `lang="`+locale+`"`)
				input = strings.Replace(input, `>15 Jan 2023</time>`, `>15 Jan 2023</time> · <span class="cv-end-label">`+label+`</span>: <time class="dt-end" datetime="`+test.finish[:7]+`">Reviewed finish</time>`, 1)
				put(t, deploy, resumeOutput, input)
				expected, err := loadResume(root)
				if err != nil {
					t.Fatal(err)
				}
				if err := validateResume(deploy, expected); err != nil {
					t.Fatal(err)
				}
				wrong := strings.Replace(input, `class="cv-end-label">`+label, `class="cv-end-label">Incorrect meaning`, 1)
				put(t, deploy, resumeOutput, wrong)
				if err := validateResume(deploy, expected); err == nil {
					t.Fatal("certification completion and expiry were conflated")
				}
			})
		}
	}
}

func TestResumeExpiryPointersAreExplicitCertificationFinishes(t *testing.T) {
	for _, test := range []struct{ name, pointers string }{
		{"ordinary education finish", "  - /knowledge/studies/0/finishDate\n"},
		{"missing certification finish", "  - /knowledge/studies/1/finishDate\n"},
		{"job finish", "  - /experience/jobs/0/roles/1/finishDate\n"},
		{"start instead of finish", "  - /knowledge/studies/1/startDate\n"},
		{"unmatched study", "  - /knowledge/studies/99/finishDate\n"},
		{"duplicate pointer", "  - /knowledge/studies/1/finishDate\n  - /knowledge/studies/1/finishDate\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, _, _ := resumeFixture(t)
			if test.name == "duplicate pointer" {
				source := strings.Replace(resumeFixtureSource, `"name": "Reviewed Certificate"`, `"studyType": "certification", "finishDate": "2024-04-01", "name": "Reviewed Certificate"`, 1)
				put(t, root, "CV/MAC.json", source)
			}
			put(t, root, ".zas/resume.yml", resumeFixtureConfig+"expiry_dates:\n"+test.pointers)
			if _, err := loadResume(root); err == nil {
				t.Fatal("unreviewed or duplicate expiry accepted")
			}
		})
	}
}
