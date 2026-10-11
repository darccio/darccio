package buildsite

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPWorkerBuildFailurePreservesPreviousDeployment(t *testing.T) {
	root := sourceTree(t)
	put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\n  markdown_negotiation: true\nzas:\n  deploy: .zas/deploy\n  layout: .zas/layout.html\n")
	template, err := os.ReadFile(filepath.Join("testdata", "http-worker", "worker.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".tools/http/worker.mjs", string(template))
	script := generator(t, ordinaryOutput+`
printf '<html lang="ca"><head><link rel="canonical" href="https://dario.cat/"></head><body>Final HTML</body></html>' > .zas/deploy/index.html
printf '# Rendered Markdown\n' > .zas/deploy/index.md
test ! -e .tools
`)
	if err := Build(context.Background(), Options{Root: root, Zas: script, Output: io.Discard}); err != nil {
		t.Fatal(err)
	}
	deploy := filepath.Join(root, ".zas", "deploy")
	before := httpWorkerDeploySnapshot(t, deploy)
	if _, exists := before["_worker.js"]; !exists {
		t.Fatal("successful guarded build did not assemble its HTTP worker")
	}
	put(t, root, ".tools/http/worker.mjs", "const PAGE_ROUTES = {};\n")
	err = Build(context.Background(), Options{Root: root, Zas: script, Output: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "assembled HTTP adapter rejected") || !strings.Contains(err.Error(), "exactly one page-inventory marker") {
		t.Fatalf("broken template did not fail at the adapter: %v", err)
	}
	if after := httpWorkerDeploySnapshot(t, deploy); !reflect.DeepEqual(before, after) {
		t.Fatal("failed HTTP assembly changed the previous successful deployment")
	}
}

func httpWorkerDeploySnapshot(t *testing.T, deploy string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	err := filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(deploy, file)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(relative)] = string(body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
