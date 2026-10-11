package projectdocs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Revision struct {
	Updated time.Time
	Files   map[string][]byte
}
type Fetcher func(context.Context, Project) (Revision, error)

// Fetch reads selected regular blobs, without checking out or executing any
// fetched files. A SHA, rather than a moving branch, is the build dependency.
func Fetch(ctx context.Context, p Project) (Revision, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var revision Revision
	if err := (Manifest{Version: 1, Projects: []Project{p}}).Validate(); err != nil {
		return revision, err
	}
	dir, err := os.MkdirTemp("", "dario-project-docs-")
	if err != nil {
		return revision, err
	}
	defer os.RemoveAll(dir)
	run := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "GIT_") {
				cmd.Env = append(cmd.Env, e)
			}
		}
		cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
		data, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("fetch %s at %s: git %s failed: %w", p.Repo, p.Commit, args[0], err)
		}
		return data, nil
	}
	if _, err := run("-c", "init.templateDir=", "init", "--quiet"); err != nil {
		return revision, err
	}
	if _, err := run("fetch", "--quiet", "--depth=1", "--no-tags", "https://github.com/"+p.Repo+".git", p.Commit); err != nil {
		return revision, err
	}
	return readRevision(p, run)
}

// Read and bound regular Git objects; symlinks and submodules are never read.
func readRevision(p Project, run func(...string) ([]byte, error)) (Revision, error) {
	var revision Revision
	resolved, err := run("rev-parse", "FETCH_HEAD")
	if err != nil {
		return revision, err
	}
	if strings.TrimSpace(string(resolved)) != p.Commit {
		return revision, fmt.Errorf("source pin mismatch for %s", p.ID)
	}
	stamp, err := run("show", "-s", "--format=%cI", p.Commit)
	if err != nil {
		return revision, err
	}
	revision.Updated, err = time.Parse(time.RFC3339, strings.TrimSpace(string(stamp)))
	if err != nil {
		return revision, err
	}
	revision.Files = map[string][]byte{}
	rootLicense := map[string]string{"mergo": "LICENSE", "zas": "COPYING", "darna": "LICENSE"}[p.ID]
	sources := []string{p.LicenseFile, rootLicense}
	for _, page := range p.Pages {
		sources = append(sources, page.Source)
	}
	total := 0
	for _, source := range sources {
		tree, err := run("ls-tree", p.Commit, "--", source)
		if err != nil {
			return revision, err
		}
		fields := strings.Fields(string(tree))
		if len(fields) != 4 || fields[3] != source || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
			return revision, fmt.Errorf("%s: missing or nonregular source %s", p.ID, source)
		}
		sizeBytes, err := run("cat-file", "-s", fields[2])
		if err != nil {
			return revision, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(string(sizeBytes)))
		if err != nil || size <= 0 || size > 1024*1024 || total+size > 10*1024*1024 {
			return revision, fmt.Errorf("%s: documentation size limit exceeded", source)
		}
		data, err := run("cat-file", "blob", fields[2])
		if err != nil {
			return revision, err
		}
		if len(data) != size {
			return revision, fmt.Errorf("%s: blob size mismatch", source)
		}
		total += size
		revision.Files[source] = data
	}
	if !bytes.Equal(revision.Files[p.LicenseFile], revision.Files[rootLicense]) {
		return revision, fmt.Errorf("%s docs license does not match repository license", p.ID)
	}
	return revision, nil
}

// Update validates the complete proposed project revision before changing its
// pin. Dispatch data can select only a reviewed repository and a full SHA.
func Update(ctx context.Context, root, repo, commit string, fetch Fetcher) error {
	if !shaPattern.MatchString(commit) {
		return fmt.Errorf("docs update requires a lowercase 40-character commit SHA")
	}
	m, err := Read(root)
	if err != nil {
		return err
	}
	index := -1
	for i, p := range m.Projects {
		if p.Repo == repo {
			index = i
		}
	}
	if index < 0 {
		return fmt.Errorf("repository is not in the reviewed docs manifest: %q", repo)
	}
	m.Projects[index].Commit = commit
	if fetch == nil {
		fetch = Fetch
	}
	rev, err := fetch(ctx, m.Projects[index])
	if err != nil {
		return err
	}
	if _, err := compile(m.Projects[index], rev); err != nil {
		return err
	}
	return writeManifest(root, m)
}
