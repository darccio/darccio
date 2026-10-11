package projectdocs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPinnedRegularBlobsWithoutCheckout(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) ([]byte, error) {
		return exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	}
	must := func(args ...string) []byte {
		t.Helper()
		data, err := run(args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return data
	}
	must("-c", "init.templateDir=", "init", "--quiet")
	p, r := fixture()
	r.Files["COPYING"] = r.Files[p.LicenseFile]
	for name, body := range r.Files {
		file := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(file), 0755)
		if err := os.WriteFile(file, body, 0644); err != nil {
			t.Fatal(err)
		}
	}
	must("add", "docs", "COPYING")
	must("-c", "commit.gpgsign=false", "-c", "user.name=Docs Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "-m", "fixture")
	p.Commit = strings.TrimSpace(string(must("rev-parse", "HEAD")))
	if err := os.WriteFile(filepath.Join(dir, ".git", "FETCH_HEAD"), []byte(p.Commit+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readRevision(p, run)
	if err != nil || len(got.Files) != 4 {
		t.Fatal(got, err)
	}
	p.Commit = strings.Repeat("b", 40)
	if _, err := readRevision(p, run); err == nil {
		t.Fatal("accepted pin mismatch")
	}
	p.Commit = strings.TrimSpace(string(must("rev-parse", "HEAD")))
	if err := os.Remove(filepath.Join(dir, "docs", "templates.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../private-file", filepath.Join(dir, "docs", "templates.md")); err != nil {
		t.Fatal(err)
	}
	must("add", "docs/templates.md")
	must("-c", "commit.gpgsign=false", "-c", "user.name=Docs Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "-m", "symlink")
	p.Commit = strings.TrimSpace(string(must("rev-parse", "HEAD")))
	if err := os.WriteFile(filepath.Join(dir, ".git", "FETCH_HEAD"), []byte(p.Commit+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readRevision(p, run); err == nil || !strings.Contains(err.Error(), "nonregular") {
		t.Fatal("accepted symlink", err)
	}
}
func TestObjectReadRejectsOversizeBeforeReadingData(t *testing.T) {
	p, _ := fixture()
	blobRead := false
	run := func(args ...string) ([]byte, error) {
		switch args[0] {
		case "rev-parse":
			return []byte(p.Commit), nil
		case "show":
			return []byte("2026-10-11T12:00:00Z"), nil
		case "ls-tree":
			return []byte("100644 blob " + strings.Repeat("a", 40) + "\tdocs/LICENSE.txt\n"), nil
		case "cat-file":
			if args[1] == "-s" {
				return []byte("1048577"), nil
			}
			blobRead = true
		}
		return nil, nil
	}
	if _, err := readRevision(p, run); err == nil || blobRead {
		t.Fatal("read oversized blob", err)
	}
}
