package buildsite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"go.yaml.in/yaml/v3"
)

const resumeSourceLimit = 2 << 20

type resumeConfig struct {
	Version       resumeVersion     `yaml:"version"`
	Source        string            `yaml:"source"`
	Locale        string            `yaml:"locale"`
	Output        string            `yaml:"output"`
	Updated       string            `yaml:"updated"`
	DatePrecision map[string]string `yaml:"date_precision"`
	ExpiryDates   []string          `yaml:"expiry_dates"`
	PublicEmail   string            `yaml:"public_email"`
}

type resumeVersion int

func (version *resumeVersion) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!int" {
		return fmt.Errorf("resume version must be an integer")
	}
	var integer int
	if err := node.Decode(&integer); err != nil {
		return err
	}
	*version = resumeVersion(integer)
	return nil
}

// This projection deliberately excludes career preferences, referrals, contact
// lists, street addresses, and remote images. The original JSON remains a static file.
type resumeEntity struct {
	Name, Description, URL string
}
type resumeRole struct {
	Name, StartDate, FinishDate, Notes string
	Challenges                         []struct {
		Description string
		Actions     []string
	}
	Competences []struct{ Name string }
}
type resumeSource struct {
	Settings struct{ MACVersion string }
	AboutMe  struct {
		Profile struct {
			Name, Surnames, Title, Description string
			Location                           struct{ Municipality, Region, Country string }
		}
		RelevantLinks []struct{ Type, URL, Description string }
	}
	Experience struct {
		Jobs []struct {
			Organization resumeEntity
			Roles        []resumeRole
		}
		Projects []struct {
			Details resumeEntity
			Roles   []resumeRole
		}
	}
	Knowledge struct {
		HardSkills, SoftSkills []struct{ Skill struct{ Name string } }
		Languages              []struct{ Name, Level string }
		Studies                []struct {
			StudyType, Name, StartDate, FinishDate, Description string
			Institution                                         resumeEntity
		}
	}
}

type resumeDate struct{ Value, Label string }
type resumeEvent struct {
	Pointer, Name, Organization, URL, StartLabel, EndLabel string
	Start                                                  resumeDate
	End                                                    *resumeDate
	Ongoing                                                bool
	Description                                            []template.HTML
	Skills                                                 []string
}
type resumeLink struct{ URL, Label string }
type resumeLanguage struct{ Name, Level string }
type resumeLabels struct {
	Title, Review, Experience, Projects, Education, Skills, Languages string
	Present, Issued, Completed, Expired, Expires, Download, Licence   string
}
type resumeView struct {
	Locale, Name, Headline, Email, Updated string
	Locality, Region, Country              string
	Summary                                template.HTML
	Links                                  []resumeLink
	Jobs, Projects, Studies                []resumeEvent
	Skills                                 []string
	Languages                              []resumeLanguage
	Labels                                 resumeLabels
}

// prepareResume is opt-in: checking for config precedes every source read.
// All output is derived in the disposable build stage, before Zas executes.
func prepareResume(root, stage string) error {
	configPath, err := agentContextRegularPath(root, ".zas/resume.yml", true)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(configPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	data, err := resumeRead(root, ".zas/resume.yml", 32<<10)
	if err != nil {
		return err
	}
	var cfg resumeConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return fmt.Errorf("resume config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("resume config must contain one document")
	}
	if cfg.Version != 1 || cfg.Source != "CV/MAC.json" || cfg.Output != "cv/index.html" || (cfg.Locale != "en" && cfg.Locale != "ca") {
		return fmt.Errorf("resume config requires version 1, source CV/MAC.json, output cv/index.html, and locale en or ca")
	}
	if cfg.PublicEmail == "" {
		cfg.PublicEmail = "cc@dario.cat"
	}
	if cfg.PublicEmail != "cc@dario.cat" {
		return fmt.Errorf("resume public_email must be cc@dario.cat")
	}
	reviewed, err := resumeParseDate(cfg.Updated)
	if err != nil {
		return fmt.Errorf("resume updated: %w", err)
	}
	for pointer, precision := range cfg.DatePrecision {
		if precision != "year" && precision != "month" && precision != "day" {
			return fmt.Errorf("resume date_precision %s must be year, month or day", pointer)
		}
	}
	data, err = resumeRead(root, cfg.Source, resumeSourceLimit)
	if err != nil {
		return err
	}
	if err := resumeUniqueJSON(data); err != nil {
		return fmt.Errorf("resume JSON: %w", err)
	}
	var source resumeSource
	if err := json.Unmarshal(data, &source); err != nil {
		return fmt.Errorf("resume JSON: %w", err)
	}
	view, err := resumeProject(cfg, source, reviewed)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	if err := resumePage.Execute(&body, view); err != nil {
		return err
	}
	// JSON is YAML-compatible and escapes '<', preventing a source value from
	// terminating the metadata comment. Imported '{{...}}' text is never a template.
	meta, err := json.Marshal(map[string]any{
		"title":       view.Labels.Title,
		"description": view.Headline, "summary": view.Headline, "language": cfg.Locale,
		"image":    "/static/avatar.webp",
		"template": false, "publish_markdown": false, "feed": false, "atproto": false,
	})
	if err != nil {
		return err
	}
	var result bytes.Buffer
	fmt.Fprintf(&result, "<!--\n%s\n-->\n", meta)
	result.Write(body.Bytes())
	output, err := agentContextRegularPath(stage, cfg.Output, true)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("resume output collides with an existing source: %s", cfg.Output)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(result.Bytes())
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func resumeRead(root, relative string, limit int64) ([]byte, error) {
	path, err := agentContextRegularPath(root, relative, false)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("resume %s must be a regular file of at most %d bytes", relative, limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err == nil && int64(len(data)) > limit {
		return nil, fmt.Errorf("resume %s exceeds %d bytes", relative, limit)
	}
	return data, err
}

// Reject ambiguous duplicate keys and excessive nesting, including fields not
// projected into the public page, before decoding the typed projection.
func resumeUniqueJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("excessive nesting")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'):
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("duplicate or invalid object key %q", key)
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		case json.Delim('['):
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}

func resumeProject(cfg resumeConfig, source resumeSource, reviewed time.Time) (resumeView, error) {
	v := resumeView{Locale: cfg.Locale, Email: cfg.PublicEmail, Updated: cfg.Updated}
	v.Labels = resumeLabels{"Curriculum vitae", "Reviewed", "Experience", "Projects", "Education and credentials", "Skills", "Languages", "Present", "Issued", "Completed", "Expired", "Expires", "Download CV (MAC JSON)", "Personal data and use permission"}
	if cfg.Locale == "ca" {
		v.Labels = resumeLabels{"Currículum", "Revisat", "Experiència", "Projectes", "Formació i credencials", "Habilitats", "Llengües", "Actualitat", "Emès", "Completat", "Caducat", "Caduca", "Descarrega el CV (JSON MAC)", "Dades personals i permís d’ús"}
	}
	expiryDates := map[string]bool{}
	matchedExpiry := map[string]bool{}
	for _, pointer := range cfg.ExpiryDates {
		if expiryDates[pointer] {
			return v, fmt.Errorf("resume expiry_dates contains duplicate pointer %s", pointer)
		}
		expiryDates[pointer] = true
	}
	p := source.AboutMe.Profile
	if source.Settings.MACVersion != "0.3" || strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Surnames) == "" || strings.TrimSpace(p.Title) == "" {
		return v, fmt.Errorf("resume source requires MACVersion 0.3 and a named profile with title")
	}
	v.Name, v.Headline = strings.TrimSpace(p.Name+" "+p.Surnames), p.Title
	v.Locality, v.Region, v.Country = p.Location.Municipality, p.Location.Region, p.Location.Country
	var err error
	v.Summary, err = resumeMarkdown(p.Description)
	if err != nil {
		return v, fmt.Errorf("resume profile description: %w", err)
	}
	for _, link := range source.AboutMe.RelevantLinks {
		if link.URL == "" {
			return v, fmt.Errorf("resume relevant link has no URL")
		}
		if err := resumeURL(link.URL); err != nil {
			return v, err
		}
		label := link.Description
		if label == "" {
			parsed, _ := url.Parse(link.URL)
			switch strings.ToLower(parsed.Hostname()) {
			case "dario.cat":
				label = "Website"
				if cfg.Locale == "ca" {
					label = "Web"
				}
			case "www.linkedin.com", "linkedin.com":
				label = "LinkedIn"
			case "sifa.id":
				label = "Sifa"
			case "github.com":
				label = "GitHub"
			case "speakerdeck.com":
				label = "Speaker Deck"
			default:
				label = parsed.Hostname()
				if label == "" {
					label = link.URL
				}
			}
		}
		v.Links = append(v.Links, resumeLink{link.URL, label})
	}
	used := map[string]bool{}
	date := func(raw, pointer string) (resumeDate, error) {
		t, err := resumeParseDate(raw)
		if err != nil {
			return resumeDate{}, fmt.Errorf("resume %s: %w", pointer, err)
		}
		used[pointer] = true
		precision := cfg.DatePrecision[pointer]
		if precision == "" {
			precision = "month"
		}
		switch precision {
		case "year":
			return resumeDate{t.Format("2006"), t.Format("2006")}, nil
		case "month":
			return resumeDate{t.Format("2006-01"), resumeHumanDate(t, cfg.Locale, false)}, nil
		default:
			return resumeDate{raw, resumeHumanDate(t, cfg.Locale, true)}, nil
		}
	}
	event := func(role resumeRole, entity resumeEntity, pointer string) (resumeEvent, error) {
		e := resumeEvent{Pointer: pointer, Name: role.Name, Organization: entity.Name, URL: entity.URL}
		if strings.TrimSpace(role.Name) == "" || strings.TrimSpace(entity.Name) == "" {
			return e, fmt.Errorf("resume %s requires a role and organization name", pointer)
		}
		if e.URL != "" {
			if err := resumeURL(e.URL); err != nil {
				return e, err
			}
		}
		var err error
		e.Start, err = date(role.StartDate, pointer+"/startDate")
		if err != nil {
			return e, err
		}
		if role.FinishDate != "" {
			end, err := date(role.FinishDate, pointer+"/finishDate")
			if err != nil {
				return e, err
			}
			if role.FinishDate < role.StartDate {
				return e, fmt.Errorf("resume %s finishes before it starts", pointer)
			}
			e.End = &end
		} else {
			e.Ongoing = true
		}
		add := func(text string) error {
			if strings.TrimSpace(text) == "" {
				return nil
			}
			html, err := resumeMarkdown(text)
			if err != nil {
				return fmt.Errorf("resume %s description: %w", pointer, err)
			}
			e.Description = append(e.Description, html)
			return nil
		}
		if err := add(entity.Description); err != nil {
			return e, err
		}
		if err := add(role.Notes); err != nil {
			return e, err
		}
		for _, challenge := range role.Challenges {
			if err := add(challenge.Description); err != nil {
				return e, err
			}
			for _, action := range challenge.Actions {
				if err := add(action); err != nil {
					return e, err
				}
			}
		}
		for _, competence := range role.Competences {
			if strings.TrimSpace(competence.Name) != "" {
				e.Skills = append(e.Skills, competence.Name)
			}
		}
		return e, nil
	}
	for i, job := range source.Experience.Jobs {
		if len(job.Roles) == 0 {
			return v, fmt.Errorf("resume job %d has no roles", i)
		}
		for j, role := range job.Roles {
			e, err := event(role, job.Organization, fmt.Sprintf("/experience/jobs/%d/roles/%d", i, j))
			if err != nil {
				return v, err
			}
			v.Jobs = append(v.Jobs, e)
		}
	}
	for i, project := range source.Experience.Projects {
		if len(project.Roles) == 0 {
			return v, fmt.Errorf("resume project %d has no roles", i)
		}
		for j, role := range project.Roles {
			e, err := event(role, project.Details, fmt.Sprintf("/experience/projects/%d/roles/%d", i, j))
			if err != nil {
				return v, err
			}
			v.Projects = append(v.Projects, e)
		}
	}
	for i, study := range source.Knowledge.Studies {
		role := resumeRole{Name: study.Name, StartDate: study.StartDate, FinishDate: study.FinishDate, Notes: study.Description}
		e, err := event(role, study.Institution, fmt.Sprintf("/knowledge/studies/%d", i))
		if err != nil {
			return v, err
		}
		e.Ongoing = false // An absent credential expiry does not assert current certification.
		if study.StudyType == "certification" {
			e.StartLabel = v.Labels.Issued
			if study.FinishDate != "" {
				e.EndLabel = v.Labels.Completed
				pointer := e.Pointer + "/finishDate"
				if expiryDates[pointer] {
					matchedExpiry[pointer] = true
					e.EndLabel = v.Labels.Expires
					finish, _ := resumeParseDate(study.FinishDate)
					if !finish.After(reviewed) {
						e.EndLabel = v.Labels.Expired
					}
				}
			}
		}
		v.Studies = append(v.Studies, e)
	}
	for _, pointer := range cfg.ExpiryDates {
		if !matchedExpiry[pointer] {
			return v, fmt.Errorf("resume expiry_dates %s is not a nonempty certification finishDate", pointer)
		}
	}
	for pointer := range cfg.DatePrecision {
		if !used[pointer] {
			return v, fmt.Errorf("resume date_precision %s is not an emitted nonempty date", pointer)
		}
	}
	for _, group := range [][]struct{ Skill struct{ Name string } }{source.Knowledge.HardSkills, source.Knowledge.SoftSkills} {
		for _, skill := range group {
			if strings.TrimSpace(skill.Skill.Name) != "" {
				v.Skills = append(v.Skills, skill.Skill.Name)
			}
		}
	}
	for _, language := range source.Knowledge.Languages {
		if strings.TrimSpace(language.Name) == "" || strings.TrimSpace(language.Level) == "" {
			return v, fmt.Errorf("resume language requires name and level")
		}
		v.Languages = append(v.Languages, resumeLanguage{language.Name, language.Level})
	}
	return v, nil
}

func resumeParseDate(raw string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil || t.Format("2006-01-02") != raw {
		return time.Time{}, fmt.Errorf("invalid YYYY-MM-DD date %q", raw)
	}
	return t, nil
}

func resumeHumanDate(t time.Time, locale string, day bool) string {
	months := []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	if locale == "ca" {
		months = []string{"gener", "febrer", "març", "abril", "maig", "juny", "juliol", "agost", "setembre", "octubre", "novembre", "desembre"}
	}
	month := months[int(t.Month())-1]
	if day {
		return fmt.Sprintf("%d %s %d", t.Day(), month, t.Year())
	}
	return fmt.Sprintf("%s %d", month, t.Year())
}

func resumeURL(raw string) error {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, "\\") || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return fmt.Errorf("unsafe CV URL %q", raw)
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return fmt.Errorf("unsafe CV URL %q", raw)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Hostname() != "" {
			return nil
		}
	case "mailto":
		if u.Opaque == "cc@dario.cat" && u.RawQuery == "" && u.Fragment == "" {
			return nil
		}
	case "":
		if u.Host == "" && (strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//")) {
			for _, part := range strings.Split(u.Path, "/") {
				if strings.HasPrefix(part, ".") {
					return fmt.Errorf("unsafe CV URL %q", raw)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("unsafe CV URL %q", raw)
}

func resumeMarkdown(input string) (template.HTML, error) {
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM))
	source := []byte(input)
	document := markdown.Parser().Parse(text.NewReader(source))
	if err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination []byte
		switch n := node.(type) {
		case *ast.Link:
			destination = n.Destination
		case *ast.AutoLink:
			destination = n.URL(source)
		case *ast.Image:
			return ast.WalkStop, fmt.Errorf("CV descriptions cannot load images; use the reviewed local avatar")
		}
		if destination != nil {
			if err := resumeURL(string(destination)); err != nil {
				return ast.WalkStop, err
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := markdown.Renderer().Render(&out, source, document); err != nil {
		return "", err
	}
	return template.HTML(out.String()), nil // Only the safe Goldmark renderer reaches this conversion.
}

var resumePage = template.Must(template.New("resume").Funcs(template.FuncMap{
	"dictEvent": func(event resumeEvent, present string) any {
		return struct {
			Event   resumeEvent
			Present string
		}{event, present}
	},
}).Parse(`
<section class="h-resume" lang="{{.Locale}}">
<h1 class="p-name">{{.Name}}</h1>
<div class="p-contact h-card">
<img class="u-photo" src="/static/avatar.webp" width="180" height="180" alt="{{.Name}}">
<p><a class="p-name u-url u-uid" href="https://dario.cat/">{{.Name}}</a></p>
<p class="p-job-title">{{.Headline}}</p>
{{if or .Locality .Region .Country}}<p>{{if .Locality}}<span class="p-locality">{{.Locality}}</span>{{end}}{{if .Region}}{{if .Locality}}, {{end}}<span class="p-region">{{.Region}}</span>{{end}}{{if .Country}}{{if or .Locality .Region}}, {{end}}<span class="p-country-name">{{.Country}}</span>{{end}}</p>{{end}}
<p><!--email_off--><a class="u-email" href="mailto:{{.Email}}">{{.Email}}</a><!--/email_off--></p>
<ul>{{range .Links}}<li><a href="{{.URL}}" rel="me">{{.Label}}</a></li>{{end}}</ul>
</div>
<div class="p-summary">{{.Summary}}</div>
<p>{{.Labels.Review}}: <time class="dt-updated" datetime="{{.Updated}}">{{.Updated}}</time>.</p>
<p><a href="/CV/MAC.json">{{.Labels.Download}}</a></p>
{{if .Jobs}}<section><h2>{{.Labels.Experience}}</h2>{{range .Jobs}}{{template "role" (dictEvent . $.Labels.Present)}}{{end}}</section>{{end}}
{{if .Projects}}<section><h2>{{.Labels.Projects}}</h2>{{range .Projects}}{{template "role" (dictEvent . $.Labels.Present)}}{{end}}</section>{{end}}
{{if .Studies}}<section><h2>{{.Labels.Education}}</h2>{{range .Studies}}
<article class="p-education h-event" data-cv-source="{{.Pointer}}">
{{template "event" .}}
</article>{{end}}</section>{{end}}
{{if .Skills}}<section data-cv-skills><h2>{{.Labels.Skills}}</h2><ul>{{range .Skills}}<li class="p-skill">{{.}}</li>{{end}}</ul></section>{{end}}
{{if .Languages}}<section><h2>{{.Labels.Languages}}</h2><ul>{{range .Languages}}<li data-cv-language><span class="cv-language-name">{{.Name}}</span>: <span class="cv-language-level">{{.Level}}</span></li>{{end}}</ul></section>{{end}}
<p><a href="/LICENSE">{{.Labels.Licence}}</a></p>
</section>
{{define "role"}}<article class="p-experience h-event" data-cv-source="{{.Event.Pointer}}">{{template "event" .Event}}{{if .Event.Ongoing}}<p class="cv-present">{{.Present}}</p>{{end}}</article>{{end}}
{{define "event"}}<h3 class="p-name">{{.Name}}</h3>
<p>{{if .URL}}<a class="p-org" href="{{.URL}}">{{.Organization}}</a>{{else}}<span class="p-org">{{.Organization}}</span>{{end}}</p>
<p>{{if .StartLabel}}{{.StartLabel}}: {{end}}<time class="dt-start" datetime="{{.Start.Value}}">{{.Start.Label}}</time>{{if .End}} · {{if .EndLabel}}<span class="cv-end-label">{{.EndLabel}}</span>: {{end}}<time class="dt-end" datetime="{{.End.Value}}">{{.End.Label}}</time>{{end}}</p>
{{range .Description}}<div class="p-description">{{.}}</div>{{end}}
{{if .Skills}}<ul>{{range .Skills}}<li class="p-skill">{{.}}</li>{{end}}</ul>{{end}}
{{end}}
`))
