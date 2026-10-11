package buildsite

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const changelogSource = "changelog/index.md"
const changelogMarker = "<!-- site-changelog -->"
const changelogLimit = 50

var changelogHash = regexp.MustCompile(`^[0-9a-f]{40}$`)

type changelogMetadata struct {
	Entry       bool   `yaml:"entry"`
	Description string `yaml:"description"`
	Template    *bool  `yaml:"template"`
	Updated     string `yaml:"updated,omitempty"`
}

// Generate only inside the disposable stage. The tracked source is the page's
// introduction; Git contributes bounded public commit metadata, never patches,
// file bodies, authors' contact details, or executable template content.
func prepareChangelog(ctx context.Context, root, stage string) (time.Time, error) {
	if _, err := os.Lstat(filepath.Join(stage, changelogSource)); os.IsNotExist(err) {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, err
	}
	source, err := readAgentContextFile(stage, changelogSource)
	if err != nil {
		return time.Time{}, err
	}
	source = bytes.TrimSpace(source)
	end := bytes.Index(source, []byte("-->"))
	if !bytes.HasPrefix(source, []byte("<!--")) || end < 4 {
		return time.Time{}, errors.New("changelog requires a leading configuration comment")
	}
	var metadata changelogMetadata
	decoder := yaml.NewDecoder(bytes.NewReader(source[4:end]))
	decoder.KnownFields(true)
	if err := decoder.Decode(&metadata); err != nil {
		return time.Time{}, fmt.Errorf("changelog configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return time.Time{}, errors.New("changelog configuration requires one YAML document")
	}
	body := source[end+3:]
	if !metadata.Entry || metadata.Template == nil || *metadata.Template || !agentContextText(metadata.Description, false) || bytes.Count(body, []byte(changelogMarker)) != 1 {
		return time.Time{}, errors.New("changelog requires entry: true, template: false, a description and one insertion marker")
	}
	// NUL separators keep subjects independent of their punctuation. The fixed
	// HEAD revision and absence of a build timestamp make repeats deterministic.
	history, err := gitOutput(ctx, root, "log", "--no-show-signature", "--no-decorate", "-50", "--format=%H%x00%cI%x00%s%x00", "HEAD", "--")
	if err != nil {
		return time.Time{}, err
	}
	entries, updated, err := renderChangelog(history)
	if err != nil {
		return time.Time{}, err
	}
	metadata.Updated = updated.Format(time.RFC3339)
	config, err := yaml.Marshal(metadata)
	if err != nil {
		return time.Time{}, err
	}
	var rendered bytes.Buffer
	rendered.WriteString("<!--\n")
	rendered.Write(config)
	rendered.WriteString("-->\n")
	rendered.Write(bytes.Replace(body, []byte(changelogMarker), entries, 1))
	rendered.WriteByte('\n')
	if err := os.WriteFile(filepath.Join(stage, changelogSource), rendered.Bytes(), 0644); err != nil {
		return time.Time{}, err
	}
	return updated, nil
}

func renderChangelog(history []byte) ([]byte, time.Time, error) {
	parts := bytes.Split(history, []byte{0})
	if len(parts) < 4 || (len(parts)-1)%3 != 0 || len(bytes.TrimSpace(parts[len(parts)-1])) != 0 || (len(parts)-1)/3 > changelogLimit {
		return nil, time.Time{}, errors.New("invalid or empty Git changelog metadata")
	}
	var body bytes.Buffer
	body.WriteString("<ol>\n")
	var updated time.Time
	for i := 0; i < len(parts)-1; i += 3 {
		hash := strings.TrimSpace(string(parts[i]))
		date, err := time.Parse(time.RFC3339, string(parts[i+1]))
		subject := string(parts[i+2])
		if !changelogHash.MatchString(hash) || err != nil || subject == "" || !utf8.ValidString(subject) || strings.IndexFunc(subject, unicode.IsControl) != -1 {
			return nil, time.Time{}, errors.New("invalid Git changelog record")
		}
		if updated.IsZero() {
			updated = date
		}
		// HTML escaping protects the resulting page. Entity-escape braces as
		// well so template-like subjects remain literal in Markdown context.
		subject = strings.NewReplacer("{", "&#123;", "}", "&#125;").Replace(html.EscapeString(subject))
		fmt.Fprintf(&body, "<li><time datetime=\"%s\">%s</time> · <a href=\"https://github.com/darccio/darccio/commit/%s\">%s</a></li>\n", date.Format(time.RFC3339), date.Format("2006-01-02"), hash, subject)
	}
	body.WriteString("</ol>")
	return body.Bytes(), updated, nil
}
