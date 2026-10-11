package main

import (
	"context"
	"dario.cat/site-tools/internal/projectdocs"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	root := flag.String("root", ".", "site source root")
	repo := flag.String("repo", "", "reviewed source repository")
	commit := flag.String("commit", "", "immutable source commit SHA")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := projectdocs.Update(ctx, *root, *repo, *commit, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
