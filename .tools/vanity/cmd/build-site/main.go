package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"dario.cat/site-tools/internal/buildsite"
)

func main() {
	root := flag.String("root", "../..", "site Git working-tree root")
	zas := flag.String("zas", "zas", "selected Zas executable (must support -no-plugins)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := buildsite.Build(ctx, buildsite.Options{Root: *root, Zas: *zas, Output: os.Stdout}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
