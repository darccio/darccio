package buildsite

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRejectsInvalidAgentContextBeforePromotion(t *testing.T) {
	for _, selection := range []string{"about/index.md", "CV/MAC.md"} {
		t.Run(selection, func(t *testing.T) {
			root := sourceTree(t)
			manifest, err := os.ReadFile("testdata/agent-context/manifest.yml")
			if err != nil {
				t.Fatal(err)
			}
			put(t, root, ".zas/config.yml", "site:\n  baseurl: https://dario.cat\n  publish_markdown: true\nzas:\n  deploy: /must-not-be-used\n")
			put(t, root, ".zas/llms.yml", strings.Replace(string(manifest), "source: about/index.md", "source: "+selection, 1))
			put(t, root, "about/index.md", "<!--\nlanguage: ca\n-->\n# Public About\n")
			err = Build(context.Background(), Options{Root: root, Zas: generator(t, ordinaryOutput), Output: io.Discard})
			if err == nil || !strings.Contains(err.Error(), "assembled agent context rejected") {
				t.Fatalf("expected assembled agent-context rejection, got %v", err)
			}
			previous, err := os.ReadFile(filepath.Join(root, ".zas/deploy/previous.txt"))
			if err != nil || string(previous) != "last successful deployment" {
				t.Fatalf("invalid context replaced previous deployment: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(root, ".zas/deploy/llms.txt")); !os.IsNotExist(err) {
				t.Fatalf("invalid context was promoted: %v", err)
			}
		})
	}
}
