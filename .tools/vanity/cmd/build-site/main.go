package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"dario.cat/site-tools/internal/buildsite"
)

func main() {
	root := flag.String("root", "../..", "site Git working-tree root")
	zas := flag.String("zas", "zas", "selected Zas executable (must support -no-plugins)")
	checkMicroformats := flag.Bool("check-microformats", false, "validate staged microformats before promotion")
	checkConsistency := flag.Bool("check-consistency", false, "validate staged source/RSS/HTML/Standard.site consistency before promotion")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	options := buildsite.Options{Root: *root, Zas: *zas, Output: os.Stdout}
	if *checkMicroformats || *checkConsistency {
		absRoot, err := filepath.Abs(*root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		options.Validate = func(ctx context.Context, deploy string) error {
			if *checkMicroformats {
				cmd := exec.CommandContext(ctx, "go", "-C", filepath.Join(absRoot, ".tools", "microformats"), "run", "-mod=readonly", "./cmd/microformats", "-root", absRoot, "-dir", deploy)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			if !*checkConsistency {
				return nil
			}
			plan, err := os.CreateTemp("", "dario-standard-plan-*.json")
			if err != nil {
				return err
			}
			planPath := plan.Name()
			defer os.Remove(planPath)
			if err := plan.Close(); err != nil {
				return err
			}
			planner := exec.CommandContext(ctx, "go", "-C", filepath.Join(absRoot, ".tools", "atproto"), "run", "-mod=readonly", "./cmd/documents", "-root", absRoot, "-rendered", deploy, "-plan", planPath)
			planner.Stdout, planner.Stderr = io.Discard, os.Stderr
			if err := planner.Run(); err != nil {
				return fmt.Errorf("offline Standard.site planning: %w", err)
			}
			checker := exec.CommandContext(ctx, "go", "-C", filepath.Join(absRoot, ".tools", "microformats"), "run", "-mod=readonly", "./cmd/consistency", "-root", absRoot, "-dir", deploy, "-standard-plan", planPath)
			checker.Stdout, checker.Stderr = os.Stdout, os.Stderr
			return checker.Run()
		}
	}
	if err := buildsite.Build(ctx, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
