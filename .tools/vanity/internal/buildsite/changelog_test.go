package buildsite

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const changelogFixture = `<!--
entry: true
description: Public website changes.
template: false
-->
<h1 class="p-name">Changes</h1>
<!-- site-changelog -->
`

func TestChangelogOptionalStagedGeneration(t *testing.T) {
	stage := t.TempDir()
	if updated, err := prepareChangelog(context.Background(), t.TempDir(), stage); err != nil || !updated.IsZero() {
		t.Fatalf("absent page should need no repository: %v %v", updated, err)
	}
	root := sourceTree(t)
	for _, arguments := range [][]string{
		{"add", "index.html"},
		{"-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-q", "-m", `fix: <script>alert(1)</script> {{.Extra "/private"}} & català`},
	} {
		cmd := exec.Command("git", append([]string{"-C", root}, arguments...)...)
		cmd.Env = append(gitEnvironment(os.Environ(), "", ""), "GIT_AUTHOR_DATE=2026-10-11T12:00:00+02:00", "GIT_COMMITTER_DATE=2026-10-11T12:00:00+02:00")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", arguments, err, output)
		}
	}
	put(t, root, changelogSource, changelogFixture)
	put(t, stage, changelogSource, changelogFixture)
	english := strings.Replace(changelogFixture, "Changes", "Site changes", 1)
	put(t, stage, "en/"+changelogSource, english)
	updated, err := prepareChangelog(context.Background(), root, stage)
	if err != nil || updated.Format(time.RFC3339) != "2026-10-11T12:00:00+02:00" {
		t.Fatalf("staged generation: %v %v", updated, err)
	}
	first := readMarkdownDiscoveryFile(t, stage, changelogSource)
	translated := readMarkdownDiscoveryFile(t, stage, "en/"+changelogSource)
	if !bytes.Equal(first, bytes.Replace(translated, []byte("Site changes"), []byte("Changes"), 1)) {
		t.Fatal("English changelog changed history, dates or escaping")
	}
	if bytes.Count(first, []byte(`<ol class="site-changelog">`)) != 1 {
		t.Fatal("generated commit list lacks its scoped text-wrapping class")
	}
	if strings.Contains(string(first), changelogMarker) || strings.Contains(string(first), "{{") || strings.Contains(string(first), "<script>") || !strings.Contains(string(first), "&lt;script&gt;") || !strings.Contains(string(first), "&amp; català") || !strings.Contains(string(first), "template: false") {
		t.Fatalf("subject was executable or changed meaning: %s", first)
	}
	if strings.Contains(string(first), "fixture@example.invalid") {
		t.Fatal("author contact appeared in public changelog")
	}
	if original := readMarkdownDiscoveryFile(t, root, changelogSource); string(original) != changelogFixture {
		t.Fatal("tracked page source was modified")
	}
	put(t, stage, changelogSource, changelogFixture)
	put(t, stage, "en/"+changelogSource, english)
	if _, err := prepareChangelog(context.Background(), root, stage); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, readMarkdownDiscoveryFile(t, stage, changelogSource)) {
		t.Fatal("identical history changed output bytes")
	}
	if !bytes.Equal(translated, readMarkdownDiscoveryFile(t, stage, "en/"+changelogSource)) {
		t.Fatal("identical history changed English output bytes")
	}
}

func TestChangelogRejectsBrokenMetadataWithoutPromotion(t *testing.T) {
	for _, source := range []string{
		strings.Replace(changelogFixture, "template: false", "template: true", 1),
		strings.Replace(changelogFixture, "template: false", "", 1),
		strings.Replace(changelogFixture, changelogMarker, "", 1),
		changelogFixture + changelogMarker,
		strings.Replace(changelogFixture, "entry: true", "entry: true\nmystery: true", 1),
		strings.Replace(changelogFixture, "entry: true", "entry: true\n---\nentry: true", 1),
	} {
		t.Run(source, func(t *testing.T) {
			root := sourceTree(t)
			put(t, root, changelogSource, source)
			if err := Build(context.Background(), Options{Root: root, Zas: generator(t, "exit 99\n"), Output: io.Discard}); err == nil || !strings.Contains(err.Error(), "changelog") {
				t.Fatalf("malformed optional source was accepted: %v", err)
			}
			previous, err := os.ReadFile(filepath.Join(root, ".zas", "deploy", "previous.txt"))
			if err != nil || string(previous) != "last successful deployment" {
				t.Fatalf("failed generation replaced deployment: %q %v", previous, err)
			}
		})
	}
}

func TestChangelogBoundsAndValidatesGitRecords(t *testing.T) {
	record := strings.Repeat("a", 40) + "\x002026-10-11T12:00:00+02:00\x00fix: bounded record\x00\n"
	for _, history := range []string{
		"", "truncated", record + "unexpected", strings.Repeat(record, changelogLimit+1),
		strings.Replace(record, "2026-10-11T12:00:00+02:00", "2026-10-11", 1),
		strings.Replace(record, strings.Repeat("a", 40), "<invalid hash>", 1),
		strings.Replace(record, "bounded record", "bad\x1bcontrol", 1),
	} {
		if _, _, err := renderChangelog([]byte(history)); err == nil {
			t.Fatalf("invalid Git records accepted: %q", history)
		}
	}
	if entries, _, err := renderChangelog([]byte(strings.Repeat(record, changelogLimit))); err != nil || bytes.Count(entries, []byte("<li>")) != changelogLimit {
		t.Fatalf("bounded output missing: %v %s", err, entries)
	}
}
