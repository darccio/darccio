package buildsite

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"dario.cat/site-tools/internal/vanity"
)

func put(t *testing.T, root, file, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func sourceTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q", root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, output)
	}
	put(t, root, ".zas/config.yml", "site:\n  host: dario.cat\nzas:\n  deploy: /must-not-be-used\n  layout: .zas/layout.html\n")
	put(t, root, ".zas/layout.html", "<html><head></head><body>{{.Body}}</body></html>")
	put(t, root, "index.html", "<h1>Ordinary page</h1>")
	put(t, root, ".zas/deploy/previous.txt", "last successful deployment")
	for _, route := range vanity.Routes() {
		data, err := os.ReadFile(filepath.Join("../../../..", route.File()))
		if err != nil {
			t.Fatal(err)
		}
		put(t, root, route.File(), string(data))
	}
	return root
}

func generator(t *testing.T, content string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("executable shell fixture requires Unix")
	}
	path := filepath.Join(t.TempDir(), "zas-fixture")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+content), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

const ordinaryOutput = `
test "$1" = generate
test "$2" = -full
test "$3" = -no-plugins
mkdir -p .zas/deploy
printf '<html><head></head><body>built</body></html>' > .zas/deploy/index.html
`

func TestStagingPreservesRawBytesGitHistoryAndPublicSources(t *testing.T) {
	root := sourceTree(t)
	for _, file := range []string{".aws/credentials", ".codex/auth.json", ".agents/private.txt", ".tools/private.txt", ".env", ".github/workflows/private.yml", "node_modules/private.txt", "docs/site-plan/tasks/T02.md", "docs/HANDOFF-FACTS.md", "docs/DOMAIN-OPERATIONS.md"} {
		put(t, root, file, "must not be staged or deployed")
	}
	put(t, root, ".well-known/public.json", "{}")
	put(t, root, ".zas/partials/contact.html", "contact")
	script := generator(t, ordinaryOutput+`
test "$(git rev-parse --show-toplevel)" = "$PWD"
test "$GIT_DIR" != "$PWD/.git"
test ! -e .git
test ! -e mergo/index.html
test ! -e darna/index.html
test ! -e cocoon/index.html
test ! -e errors/index.html
test ! -e xdr/index.html
test ! -e awaken/index.html
test ! -e x/index.html
test ! -e .aws
test ! -e .codex
test ! -e .agents
test ! -e .tools
test ! -e .env
test ! -e .github
test ! -e node_modules
test ! -e docs/site-plan
test ! -e docs/HANDOFF-FACTS.md
test ! -e docs/DOMAIN-OPERATIONS.md
test -f .well-known/public.json
test -f .zas/partials/contact.html
test -f index.html
test "$(sed -n 's/^[[:space:]]*deploy: //p' .zas/config.yml)" = .zas/deploy
`)
	if err := Build(context.Background(), Options{Root: root, Zas: script, Output: io.Discard}); err != nil {
		t.Fatal(err)
	}
	deploy := filepath.Join(root, ".zas", "deploy")
	if _, err := os.Stat(filepath.Join(deploy, "previous.txt")); !os.IsNotExist(err) {
		t.Fatal("previous output was not replaced")
	}
	for _, route := range vanity.Routes() {
		before, _ := os.ReadFile(filepath.Join(root, route.File()))
		after, err := os.ReadFile(filepath.Join(deploy, route.File()))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("raw bytes changed at %s: %v", route.Path, err)
		}
	}
	config, _ := os.ReadFile(filepath.Join(root, ".zas", "config.yml"))
	if !strings.Contains(string(config), "/must-not-be-used") {
		t.Fatal("source configuration was changed")
	}
}

func TestFailedGenerationOrGuardLeavesPreviousDeploy(t *testing.T) {
	cases := []struct {
		name, script string
		mutate       func(*testing.T, string)
	}{
		{"generator failure", ordinaryOutput + "exit 9\n", nil},
		{"unexpected route", ordinaryOutput + `mkdir -p .zas/deploy/unknown
printf '<meta name="go-import" content="example.org/unknown git https://example.org/repo">' > .zas/deploy/unknown/index.html
`, nil},
		{"hidden output", ordinaryOutput + "mkdir -p .zas/deploy/.git\nprintf secret > .zas/deploy/.git/config\n", nil},
		{"raw regression", ordinaryOutput, func(t *testing.T, root string) {
			put(t, root, "mergo/index.html", `<meta name="go-import" content="dario.cat/mergo git https://github.com/darccio/mergo">`)
		}},
		{"missing raw source", ordinaryOutput, func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "cocoon/index.html")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := sourceTree(t)
			if tc.mutate != nil {
				tc.mutate(t, root)
			}
			err := Build(context.Background(), Options{Root: root, Zas: generator(t, tc.script), Output: io.Discard})
			if err == nil {
				t.Fatal("bad build succeeded")
			}
			got, err := os.ReadFile(filepath.Join(root, ".zas/deploy/previous.txt"))
			if err != nil || string(got) != "last successful deployment" {
				t.Fatalf("previous deployment lost: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, ".zas/deploy/index.html")); !os.IsNotExist(err) {
				t.Fatal("partial generated output reached previous deployment")
			}
		})
	}
}

func TestCopyRejectsSourceSymlinks(t *testing.T) {
	root := sourceTree(t)
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "public.txt")); err != nil {
		t.Skip(err)
	}
	if err := Build(context.Background(), Options{Root: root, Zas: generator(t, ordinaryOutput), Output: io.Discard}); err == nil {
		t.Fatal("source symlink was followed")
	}
}

func TestValidationRunsBeforePromotion(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprintf("reject=%t", reject), func(t *testing.T) {
			root := sourceTree(t)
			called := false
			err := Build(context.Background(), Options{
				Root: root, Zas: generator(t, ordinaryOutput), Output: io.Discard,
				Validate: func(ctx context.Context, deploy string) error {
					called = true
					if ctx.Err() != nil || deploy == filepath.Join(root, ".zas", "deploy") {
						t.Fatal("validator must inspect the complete stage before promotion")
					}
					if err := vanity.CheckDir(deploy); err != nil {
						t.Fatalf("validator did not receive restored vanity pages: %v", err)
					}
					if got, err := os.ReadFile(filepath.Join(root, ".zas", "deploy", "previous.txt")); err != nil || string(got) != "last successful deployment" {
						t.Fatalf("previous output changed before validation: %v", err)
					}
					if reject {
						return errors.New("identity mismatch")
					}
					return nil
				},
			})
			if !called {
				t.Fatal("validator was not invoked")
			}
			if reject {
				if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
					t.Fatalf("validation failure was not returned: %v", err)
				}
				if got, err := os.ReadFile(filepath.Join(root, ".zas", "deploy", "previous.txt")); err != nil || string(got) != "last successful deployment" {
					t.Fatalf("validation failure lost previous output: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else if _, err := os.Stat(filepath.Join(root, ".zas", "deploy", "index.html")); err != nil {
				t.Fatalf("accepted stage was not promoted: %v", err)
			}
		})
	}
}

func TestPromotionRestoresPreviousOnRenameFailure(t *testing.T) {
	root := t.TempDir()
	dest, backup := filepath.Join(root, "deploy"), filepath.Join(root, "backup")
	put(t, root, "deploy/previous.txt", "previous")
	if err := promote(filepath.Join(root, "missing"), dest, backup); err == nil {
		t.Fatal("expected promotion failure")
	}
	if data, err := os.ReadFile(filepath.Join(dest, "previous.txt")); err != nil || string(data) != "previous" {
		t.Fatalf("previous not restored: %v", err)
	}
}
