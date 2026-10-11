package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"dario.cat/atproto-tools/internal/atproto"
	"dario.cat/atproto-tools/internal/document"
)

type paths []string

func (p *paths) String() string         { return fmt.Sprint([]string(*p)) }
func (p *paths) Set(value string) error { *p = append(*p, value); return nil }

func main() {
	var options document.Options
	var deletes paths
	var output string
	flag.StringVar(&options.Root, "root", ".", "site root")
	flag.StringVar(&options.Rendered, "rendered", ".zas/deploy", "root-relative assembled HTML directory")
	flag.StringVar(&options.Allowlist, "allowlist", document.AllowlistPath, "root-relative owner-reviewed document allowlist")
	flag.StringVar(&options.State, "state", document.StatePath, "ignored private recovery state path")
	flag.BoolVar(&options.Apply, "apply", false, "apply only explicitly reviewed documents")
	flag.BoolVar(&options.Candidates, "candidates", false, "list local candidates without approving or publishing")
	flag.Var(&deletes, "delete", "explicit existing canonical path to delete (repeatable; apply required to mutate)")
	flag.StringVar(&output, "plan", "", "optional root-relative JSON plan output; default stdout only")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(errors.New("unexpected positional arguments"))
	}
	options.Delete = []string(deletes)
	root, err := filepath.Abs(options.Root)
	if err != nil {
		fail(err)
	}
	if output != "" {
		output = atproto.SitePath(root, output)
		for _, input := range []string{options.State, options.Allowlist, document.ManifestPath, document.PublicationPath} {
			if output == atproto.SitePath(root, input) {
				fail(errors.New("plan output must not overwrite publisher inputs or state"))
			}
		}
	}
	if options.Apply {
		options.Credentials = atproto.Credentials{Handle: os.Getenv("ATP_HANDLE"), AppPassword: os.Getenv("ATP_APP_PASSWORD")}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	plan, err := document.Run(ctx, atproto.NewClient(), options)
	if err != nil {
		fail(err)
	}
	if output != "" {
		if err := document.WritePlan(output, plan); err != nil {
			fail(err)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(plan); err != nil {
		fail(errors.New("cannot write JSON plan"))
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, "documents:", err); os.Exit(1) }
