package main

import (
	"context"
	"flag"
	"fmt"
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
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	options := buildsite.Options{Root: *root, Zas: *zas, Output: os.Stdout}
	if *checkMicroformats {
		absRoot, err := filepath.Abs(*root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		options.Validate = func(ctx context.Context, deploy string) error {
			cmd := exec.CommandContext(ctx, "go", "-C", filepath.Join(absRoot, ".tools", "microformats"), "run", "-mod=readonly", "./cmd/microformats", "-root", absRoot, "-dir", deploy)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			return cmd.Run()
		}
	}
	if err := buildsite.Build(ctx, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
