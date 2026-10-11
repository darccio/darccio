// Package buildsite assembles a deployment without touching the previous one
// until the generator, restored vanity fragments, and output checks succeed.
package buildsite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dario.cat/site-tools/internal/projectdocs"
	"dario.cat/site-tools/internal/vanity"
	"go.yaml.in/yaml/v3"
)

type Options struct {
	Root     string
	Zas      string
	Output   io.Writer
	Validate func(context.Context, string) error
}

func Build(ctx context.Context, options Options) error {
	if options.Output == nil {
		options.Output = io.Discard
	}
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return err
	}
	zas, err := exec.LookPath(options.Zas)
	if err != nil {
		return err
	}
	zas, err = filepath.Abs(zas)
	if err != nil {
		return err
	}
	control := filepath.Join(root, ".zas")
	lock := filepath.Join(control, ".site-build-lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("acquire build lock %s (another build or stale lock): %w", lock, err)
	}
	defer os.Remove(lock)
	stage, err := os.MkdirTemp(control, ".site-build-")
	if err != nil {
		return err
	}
	promoted := false
	defer func() {
		backup := filepath.Join(stage, "previous-deploy")
		if _, err := os.Lstat(backup); err == nil && !promoted {
			fmt.Fprintf(options.Output, "Preserved previous deployment for recovery at %s\n", backup)
			return
		}
		os.RemoveAll(stage)
	}()
	if err := copySources(ctx, root, stage); err != nil {
		return err
	}
	if err := isolateDeploy(filepath.Join(stage, ".zas", "config.yml")); err != nil {
		return err
	}
	changelogUpdated, err := prepareChangelog(ctx, root, stage)
	if err != nil {
		return fmt.Errorf("assemble public changelog: %w", err)
	}
	docs, err := projectdocs.Prepare(ctx, root, stage, nil)
	if err != nil {
		return fmt.Errorf("assemble project docs: %w", err)
	}
	if !changelogUpdated.IsZero() {
		docs.LastMod["changelog/index.html"] = changelogUpdated
	}
	gitDir, err := gitOutput(ctx, root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	// Git looks up the original history against identical relative source paths.
	// No Git database, account directory, or tooling is copied into the stage.
	cmd := exec.CommandContext(ctx, zas, "generate", "-full", "-no-plugins")
	cmd.Dir = stage
	cmd.Env = gitEnvironment(os.Environ(), strings.TrimSpace(string(gitDir)), stage)
	cmd.Stdout, cmd.Stderr = options.Output, options.Output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("isolated zas generate -full -no-plugins failed (requires Zas v0.0.7+): %w", err)
	}
	deploy := filepath.Join(stage, ".zas", "deploy")
	for _, route := range vanity.Routes() {
		// Undocumented import fragments retain their original bytes.
		if docs.Owns(route.File()) {
			continue
		}
		if err := copyFile(filepath.Join(root, filepath.FromSlash(route.File())), filepath.Join(deploy, filepath.FromSlash(route.File()))); err != nil {
			return err
		}
	}
	if err := docs.Finalize(root, deploy); err != nil {
		return fmt.Errorf("finalize project docs: %w", err)
	}
	if err := finalizeSitemap(ctx, root, stage, deploy, docs.LastMod); err != nil {
		return fmt.Errorf("assembled sitemap rejected: %w", err)
	}
	if err := finalizeAgentContext(stage, deploy); err != nil {
		return fmt.Errorf("assembled agent context rejected: %w", err)
	}
	if err := docs.LinkContexts(deploy); err != nil {
		return err
	}
	if err := finalizeMarkdownDiscovery(stage, deploy, docs.Twins); err != nil {
		return fmt.Errorf("assembled Markdown discovery rejected: %w", err)
	}
	if err := finalizeHTTPWorker(root, stage, deploy, docs.Twins); err != nil {
		return fmt.Errorf("assembled HTTP adapter rejected: %w", err)
	}
	if err := auditOutput(deploy); err != nil {
		return err
	}
	if err := vanity.CheckDir(deploy); err != nil {
		return fmt.Errorf("assembled deployment rejected: %w", err)
	}
	if options.Validate != nil {
		if err := options.Validate(ctx, deploy); err != nil {
			return fmt.Errorf("assembled microformats rejected: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := promote(deploy, filepath.Join(control, "deploy"), filepath.Join(stage, "previous-deploy")); err != nil {
		return err
	}
	promoted = true
	fmt.Fprintln(options.Output, "Built and checked .zas/deploy; seven vanity import contracts preserved.")
	return nil
}

func gitOutput(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	cmd.Env = gitEnvironment(os.Environ(), "", "")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w", args, err)
	}
	return output, nil
}

func gitEnvironment(env []string, dir, workTree string) []string {
	var clean []string
	for _, entry := range env {
		// Never inherit a caller's alternate index, worktree or repository.
		if !strings.HasPrefix(entry, "GIT_") {
			clean = append(clean, entry)
		}
	}
	if dir != "" {
		clean = append(clean, "GIT_DIR="+dir, "GIT_WORK_TREE="+workTree)
	}
	return clean
}

func allowedSource(path string) bool {
	path = filepath.ToSlash(path)
	if path == "docs/HANDOFF-FACTS.md" || path == "docs/DOMAIN-OPERATIONS.md" || path == "docs/site-plan" || strings.HasPrefix(path, "docs/site-plan/") {
		return false
	}
	if path == ".zas/deploy" || strings.HasPrefix(path, ".zas/deploy/") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "node_modules" || (strings.HasPrefix(part, ".") && part != ".zas" && part != ".zas.yml" && part != ".well-known") {
			return false
		}
	}
	for _, route := range vanity.Routes() {
		if path == route.File() {
			return false
		}
	}
	return true
}

func copySources(ctx context.Context, root, stage string) error {
	// Include working-tree edits and nonignored new public files. Git exclusions
	// plus the explicit hidden-path policy keep account state out of staging.
	files, err := gitOutput(ctx, root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}
	for _, file := range bytes.Split(files, []byte{0}) {
		if len(file) == 0 || !allowedSource(string(file)) {
			continue
		}
		source := filepath.Join(root, string(file))
		if _, err := os.Lstat(source); os.IsNotExist(err) {
			continue // A tracked working-tree deletion must stay deleted.
		}
		if err := copyFile(source, filepath.Join(stage, string(file))); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, dest string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing nonregular source %s", source)
	}
	// Reject directory symlinks too: Lstat(file) alone would follow parents.
	for parent := filepath.Dir(source); parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		stat, err := os.Lstat(parent)
		if err != nil {
			return err
		}
		if stat.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink parent %s", parent)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chtimes(dest, info.ModTime(), info.ModTime())
}

// Rewrite only zas.deploy in the isolated config, so absolute or escaping
// deploy settings cannot make the generator modify the previous output.
func isolateDeploy(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse build config: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("build config must be a YAML mapping")
	}
	mapping := document.Content[0]
	var zas *yaml.Node
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "zas" {
			if zas != nil {
				return fmt.Errorf("duplicate zas config section")
			}
			zas = mapping.Content[i+1]
		}
	}
	if zas == nil || zas.Kind != yaml.MappingNode {
		return fmt.Errorf("config must have a zas mapping")
	}
	for i := 0; i < len(zas.Content); i += 2 {
		if zas.Content[i].Value == "deploy" {
			zas.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ".zas/deploy"}
			data, err = yaml.Marshal(&document)
			if err != nil {
				return err
			}
			return os.WriteFile(path, data, 0644)
		}
	}
	return fmt.Errorf("config must explicitly set zas.deploy")
}

func auditOutput(dir string) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing deploy symlink %s", rel)
		}
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			if strings.HasPrefix(part, ".") && part != ".well-known" {
				return fmt.Errorf("unexpected hidden deploy entry %s", rel)
			}
		}
		if strings.HasPrefix(filepath.ToSlash(rel), "docs/site-plan/") || filepath.ToSlash(rel) == "docs/HANDOFF-FACTS.html" || filepath.ToSlash(rel) == "docs/DOMAIN-OPERATIONS.html" {
			return fmt.Errorf("operations documentation in deploy: %s", rel)
		}
		return nil
	})
}

// Directory replacement uses same-filesystem renames. If promotion fails after
// moving the old deploy aside, restore it before returning an error.
func promote(source, dest, backup string) error {
	hadPrevious := false
	if info, err := os.Lstat(dest); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("existing deploy must be a real directory")
		}
		if err := os.Rename(dest, backup); err != nil {
			return err
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(source, dest); err != nil {
		if hadPrevious {
			if restoreErr := os.Rename(backup, dest); restoreErr != nil {
				return fmt.Errorf("promote failed: %v; restore failed: %v; previous deploy at %s", err, restoreErr, backup)
			}
		}
		return err
	}
	return nil
}
