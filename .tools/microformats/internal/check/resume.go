package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"go.yaml.in/yaml/v3"
	"willnorris.com/go/microformats"
)

const resumeOutput = "cv/index.html"

type resumeConfig struct {
	Version       resumeVersion     `yaml:"version"`
	Source        string            `yaml:"source"`
	Locale        string            `yaml:"locale"`
	Output        string            `yaml:"output"`
	Updated       string            `yaml:"updated"`
	PublicEmail   string            `yaml:"public_email"`
	DatePrecision map[string]string `yaml:"date_precision"`
	ExpiryDates   []string          `yaml:"expiry_dates"`
}

type resumeVersion int

func (version *resumeVersion) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!int" {
		return fmt.Errorf("resume version must be an integer")
	}
	var value int
	if err := node.Decode(&value); err != nil {
		return err
	}
	*version = resumeVersion(value)
	return nil
}

type resumeExpectation struct {
	Config resumeConfig
	Raw    []byte
	CV     resumeSource
	Events []resumeEvent
	Expiry map[string]bool
}

// This independent projection deliberately excludes contact preferences and
// addresses. Only explicitly configured builds may open the source at all.
type resumeSource struct {
	Settings struct{ MACVersion string }
	AboutMe  struct {
		Profile struct {
			Name, Surnames, Title, Description string
			Location                           struct{ Municipality, Region, Country string }
		}
	}
	Experience struct {
		Jobs []struct {
			Organization struct{ Name string }
			Roles        []resumeRole
		}
		Projects []struct {
			Details struct{ Name string }
			Roles   []resumeRole
		}
	}
	Knowledge struct {
		Studies []struct {
			StudyType                   string
			Name, StartDate, FinishDate string
			Institution                 struct{ Name string }
		}
		HardSkills, SoftSkills []struct {
			Skill struct{ Name string }
		}
		Languages []struct{ Name, Level string }
	}
}

type resumeRole struct {
	Name, StartDate, FinishDate string
	Competences                 []struct{ Name string }
}
type resumeEvent struct {
	StudyType, FinishSource                           string
	Pointer, Property, Name, Organization, Start, End string
	Skills                                            []string
}

func readResumeFile(root, file string) ([]byte, error) {
	parts := strings.Split(file, "/")
	path := root
	for i, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return nil, fmt.Errorf("resume path must contain ordinary directories and a regular file: %s", file)
		}
		if i == len(parts)-1 && info.Mode().Perm()&0444 == 0 {
			return nil, fmt.Errorf("resume file is not readable: %s", file)
		}
	}
	return os.ReadFile(path)
}

// Absence of this opt-in returns before inspecting any CV path or its bytes.
func loadResume(root string) (*resumeExpectation, error) {
	input, err := readResumeFile(root, ".zas/resume.yml")
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var config resumeConfig
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("resume configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("resume configuration must contain exactly one YAML document")
	}
	if config.PublicEmail == "" {
		config.PublicEmail = email
	}
	if config.Version != 1 || config.Source != "CV/MAC.json" || config.Output != resumeOutput || (config.Locale != "en" && config.Locale != "ca") || config.PublicEmail != email {
		return nil, fmt.Errorf("resume configuration must select version 1, CV/MAC.json, cv/index.html, en or ca, and the public contact")
	}
	if date, err := time.Parse("2006-01-02", config.Updated); err != nil || date.Format("2006-01-02") != config.Updated {
		return nil, fmt.Errorf("resume review date must be a complete calendar date")
	}
	raw, err := readResumeFile(root, config.Source)
	if err != nil {
		return nil, err
	}
	if _, err := decodeJSONObject(string(raw)); err != nil {
		return nil, fmt.Errorf("resume JSON: %w", err)
	}
	expected := &resumeExpectation{Config: config, Raw: raw, Expiry: make(map[string]bool)}
	if err := json.Unmarshal(raw, &expected.CV); err != nil {
		return nil, fmt.Errorf("resume source projection: %w", err)
	}
	profile := expected.CV.AboutMe.Profile
	if expected.CV.Settings.MACVersion != "0.3" || normalized(profile.Name+" "+profile.Surnames) != owner || normalized(profile.Title) == "" || normalized(profile.Description) == "" {
		return nil, fmt.Errorf("resume source must identify the owner with a title and summary")
	}
	dates := make(map[string]bool)
	appendEvent := func(pointer, property, name, org, start, end string, skills []string) error {
		if normalized(name) == "" || normalized(org) == "" || start == "" {
			return fmt.Errorf("%s: resume event requires name, organization and source start", pointer)
		}
		begin, err := resumeDate(config, dates, pointer+"/startDate", start)
		if err != nil {
			return err
		}
		finish := ""
		if end != "" {
			finish, err = resumeDate(config, dates, pointer+"/finishDate", end)
			if err != nil {
				return err
			}
			if end < start {
				return fmt.Errorf("%s: resume finish precedes its start", pointer)
			}
		}
		expected.Events = append(expected.Events, resumeEvent{Pointer: pointer, Property: property, Name: name, Organization: org, Start: begin, End: finish, Skills: skills, FinishSource: end})
		return nil
	}
	for i, job := range expected.CV.Experience.Jobs {
		for j, role := range job.Roles {
			pointer := "/experience/jobs/" + strconv.Itoa(i) + "/roles/" + strconv.Itoa(j)
			if err := appendEvent(pointer, "p-experience", role.Name, job.Organization.Name, role.StartDate, role.FinishDate, resumeRoleSkills(role)); err != nil {
				return nil, err
			}
		}
	}
	for i, project := range expected.CV.Experience.Projects {
		for j, role := range project.Roles {
			pointer := "/experience/projects/" + strconv.Itoa(i) + "/roles/" + strconv.Itoa(j)
			if err := appendEvent(pointer, "p-experience", role.Name, project.Details.Name, role.StartDate, role.FinishDate, resumeRoleSkills(role)); err != nil {
				return nil, err
			}
		}
	}
	for i, study := range expected.CV.Knowledge.Studies {
		pointer := "/knowledge/studies/" + strconv.Itoa(i)
		if err := appendEvent(pointer, "p-education", study.Name, study.Institution.Name, study.StartDate, study.FinishDate, nil); err != nil {
			return nil, err
		}
		expected.Events[len(expected.Events)-1].StudyType = study.StudyType
	}
	for pointer := range config.DatePrecision {
		if !dates[pointer] {
			return nil, fmt.Errorf("resume precision pointer does not identify an emitted source date: %s", pointer)
		}
	}
	for _, pointer := range config.ExpiryDates {
		if expected.Expiry[pointer] {
			return nil, fmt.Errorf("duplicate resume expiry pointer: %s", pointer)
		}
		matched := false
		for _, event := range expected.Events {
			if event.Property == "p-education" && event.StudyType == "certification" && event.End != "" && pointer == event.Pointer+"/finishDate" {
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("resume expiry pointer must identify a nonempty certification finishDate: %s", pointer)
		}
		expected.Expiry[pointer] = true
	}
	return expected, nil
}

func resumeRoleSkills(role resumeRole) []string {
	var skills []string
	for _, skill := range role.Competences {
		skills = append(skills, normalized(skill.Name))
	}
	return skills
}

func resumeDate(config resumeConfig, seen map[string]bool, pointer, value string) (string, error) {
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return "", fmt.Errorf("%s: invalid MAC calendar date", pointer)
	}
	seen[pointer] = true
	precision := config.DatePrecision[pointer]
	if precision == "" {
		precision = "month"
	}
	switch precision {
	case "year":
		return value[:4], nil
	case "month":
		return value[:7], nil
	case "day":
		return value, nil
	default:
		return "", fmt.Errorf("%s: unsupported resume date precision", pointer)
	}
}

func validateResume(deploy string, expected *resumeExpectation) error {
	if expected == nil {
		return nil
	}
	if _, err := os.Lstat(filepath.Join(deploy, "cv", "index.md")); !os.IsNotExist(err) {
		return fmt.Errorf("resume must not emit a Markdown twin")
	}
	raw, err := readResumeFile(deploy, expected.Config.Source)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, expected.Raw) {
		return fmt.Errorf("deployed CV JSON differs from its source bytes")
	}
	input, err := readResumeFile(deploy, expected.Config.Output)
	if err != nil {
		return err
	}
	doc, err := parseHTML(input)
	if err != nil {
		return err
	}
	pageURL := homeURL + "cv/"
	base, err := url.Parse(pageURL)
	if err != nil {
		return err
	}
	data := microformats.ParseNode(doc, base)
	root, err := uniqueNode(doc, func(n *htmlNode) bool { return hasClass(n, "h-resume") }, "h-resume")
	if err != nil {
		return err
	}
	if len(itemsOfType(data.Items, "h-entry")) != 0 || len(find(root, func(n *htmlNode) bool { return isTag(n, "script") })) != 0 {
		return fmt.Errorf("resume must not expose an h-entry or executable content")
	}
	html, err := uniqueNode(doc, func(n *htmlNode) bool { return isTag(n, "html") }, "HTML root")
	if err != nil || attribute(html, "lang") != expected.Config.Locale || attribute(root, "lang") != expected.Config.Locale {
		return fmt.Errorf("resume language disagrees with configuration")
	}
	resumes := itemsOfType(data.Items, "h-resume")
	if len(resumes) != 1 {
		return fmt.Errorf("expected exactly one parsed h-resume")
	}
	if err := property(resumes[0], "name", owner); err != nil {
		return err
	}
	converter := goldmark.New(goldmark.WithExtensions(extension.GFM))
	var summary bytes.Buffer
	if err := converter.Convert([]byte(expected.CV.AboutMe.Profile.Description), &summary); err != nil {
		return err
	}
	summarySource, err := parseHTML(summary.Bytes())
	if err != nil {
		return err
	}
	if err := property(resumes[0], "summary", textContent(summarySource)); err != nil {
		return fmt.Errorf("resume summary: %w", err)
	}
	summaryNode, err := uniqueNode(root, func(n *htmlNode) bool { return hasClass(n, "p-summary") }, "resume summary")
	if err != nil || fingerprint(summaryNode, pageURL) != fingerprint(summarySource, pageURL) {
		return fmt.Errorf("resume summary content or destinations disagree with source")
	}
	contact, err := uniqueNode(root, func(n *htmlNode) bool { return hasClass(n, "p-contact") && hasClass(n, "h-card") }, "resume contact h-card")
	if err != nil {
		return err
	}
	cards := itemsOfType(microformats.ParseNode(contact, base).Items, "h-card")
	if len(cards) != 1 {
		return fmt.Errorf("resume contact must parse as one h-card")
	}
	if err := validateCard(cards[0], true); err != nil {
		return fmt.Errorf("resume contact: %w", err)
	}
	if err := property(cards[0], "job-title", expected.CV.AboutMe.Profile.Title); err != nil {
		return err
	}
	location := expected.CV.AboutMe.Profile.Location
	for key, want := range map[string]string{"locality": location.Municipality, "region": location.Region, "country-name": location.Country} {
		if want == "" {
			if len(propertyStrings(cards[0], key)) != 0 {
				return fmt.Errorf("resume contact invented %s", key)
			}
		} else if err := property(cards[0], key, want); err != nil {
			return fmt.Errorf("resume public location: %w", err)
		}
	}
	links := find(root, func(n *htmlNode) bool {
		return isTag(n, "a") && resolve(pageURL, attribute(n, "href")) == homeURL+"CV/MAC.json"
	})
	if len(links) != 1 {
		return fmt.Errorf("resume must link once to its preserved public JSON")
	}
	reviewed, err := uniqueNode(root, func(n *htmlNode) bool { return isTag(n, "time") && hasClass(n, "dt-updated") }, "resume review time")
	if err != nil || attribute(reviewed, "datetime") != expected.Config.Updated {
		return fmt.Errorf("resume review date disagrees with configuration")
	}
	events := find(root, func(n *htmlNode) bool { return hasClass(n, "h-event") })
	if len(events) != len(expected.Events) {
		return fmt.Errorf("resume event count got %d, want %d", len(events), len(expected.Events))
	}
	seen := make(map[string]bool)
	for _, want := range expected.Events {
		event, err := uniqueNode(root, func(n *htmlNode) bool { return attribute(n, "data-cv-source") == want.Pointer }, "source-derived resume event "+want.Pointer)
		if err != nil {
			return err
		}
		if !hasClass(event, "h-event") || !hasClass(event, want.Property) || (hasClass(event, "p-experience") && hasClass(event, "p-education")) {
			return fmt.Errorf("%s: wrong resume event classification", want.Pointer)
		}
		parsed := itemsOfType(microformats.ParseNode(event, base).Items, "h-event")
		if len(parsed) != 1 {
			return fmt.Errorf("%s: expected one parsed h-event", want.Pointer)
		}
		for key, value := range map[string]string{"name": want.Name, "org": want.Organization, "start": want.Start} {
			if err := property(parsed[0], key, value); err != nil {
				return fmt.Errorf("%s: %w", want.Pointer, err)
			}
		}
		for class, value := range map[string]string{"dt-start": want.Start, "dt-end": want.End} {
			times := find(event, func(n *htmlNode) bool { return hasClass(n, class) })
			if value == "" {
				if len(times) != 0 || len(propertyStrings(parsed[0], "end")) != 0 {
					return fmt.Errorf("%s: invented finish date", want.Pointer)
				}
			} else if len(times) != 1 || !isTag(times[0], "time") || attribute(times[0], "datetime") != value {
				return fmt.Errorf("%s: %s precision disagrees with source", want.Pointer, class)
			}
		}
		if want.End != "" {
			if err := property(parsed[0], "end", want.End); err != nil {
				return err
			}
		}
		if want.StudyType == "certification" && want.End != "" {
			label := "Completed"
			if expected.Expiry[want.Pointer+"/finishDate"] {
				label = "Expired"
				if want.FinishSource > expected.Config.Updated {
					label = "Expires"
				}
			}
			if expected.Config.Locale == "ca" {
				label = map[string]string{"Completed": "Completat", "Expired": "Caducat", "Expires": "Caduca"}[label]
			}
			endLabel, err := uniqueNode(event, func(n *htmlNode) bool { return hasClass(n, "cv-end-label") }, "certification finish meaning")
			if err != nil || textContent(endLabel) != label {
				return fmt.Errorf("%s: certification completion/expiry label disagrees with reviewed configuration", want.Pointer)
			}
		}
		skills := propertyStrings(parsed[0], "skill")
		for i := range skills {
			skills[i] = normalized(skills[i])
		}
		if !slices.Equal(skills, want.Skills) {
			return fmt.Errorf("%s: role competences disagree with source", want.Pointer)
		}
		seen[want.Pointer] = true
	}
	for _, event := range events {
		if !seen[attribute(event, "data-cv-source")] {
			return fmt.Errorf("unmatched resume event")
		}
	}
	var skills []string
	for _, skill := range expected.CV.Knowledge.HardSkills {
		skills = append(skills, normalized(skill.Skill.Name))
	}
	for _, skill := range expected.CV.Knowledge.SoftSkills {
		skills = append(skills, normalized(skill.Skill.Name))
	}
	skillList, err := uniqueNode(root, func(n *htmlNode) bool { return hasAttribute(n, "data-cv-skills") }, "global resume skill list")
	if len(skills) == 0 {
		if len(find(root, func(n *htmlNode) bool { return hasAttribute(n, "data-cv-skills") })) != 0 {
			return fmt.Errorf("resume has a skill list absent from source")
		}
	} else if err != nil {
		return err
	}
	var gotSkills []*htmlNode
	if skillList != nil {
		gotSkills = find(skillList, func(n *htmlNode) bool { return hasClass(n, "p-skill") })
	}
	var skillValues []string
	for _, skill := range gotSkills {
		skillValues = append(skillValues, textContent(skill))
	}
	if !slices.Equal(skillValues, skills) {
		return fmt.Errorf("resume skills disagree with source")
	}
	parsedSkills := propertyStrings(resumes[0], "skill")
	for i := range parsedSkills {
		parsedSkills[i] = normalized(parsedSkills[i])
	}
	if !slices.Equal(parsedSkills, skills) {
		return fmt.Errorf("parsed global resume skills disagree with source")
	}
	languages := find(root, func(n *htmlNode) bool { return hasAttribute(n, "data-cv-language") })
	if len(languages) != len(expected.CV.Knowledge.Languages) {
		return fmt.Errorf("resume language count disagrees with source")
	}
	for i, language := range expected.CV.Knowledge.Languages {
		for class, want := range map[string]string{"cv-language-name": language.Name, "cv-language-level": language.Level} {
			value, err := uniqueNode(languages[i], func(n *htmlNode) bool { return hasClass(n, class) }, class)
			if err != nil || textContent(value) != normalized(want) {
				return fmt.Errorf("resume language %d disagrees with source", i)
			}
		}
	}
	if err := oneValue(meta(doc, "og:type"), "website", "resume Open Graph type"); err != nil {
		return err
	}
	if err := oneValue(meta(doc, "og:url"), pageURL, "resume Open Graph URL"); err != nil {
		return err
	}
	return validateJSONLD(doc, pageURL, nil)
}
