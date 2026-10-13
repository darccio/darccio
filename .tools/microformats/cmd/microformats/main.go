package main

import (
	"flag"
	"fmt"
	"os"

	"dario.cat/microformats-check/internal/check"
)

func main() {
	root := flag.String("root", "../..", "site source root")
	dir := flag.String("dir", "../../.zas/deploy", "completed deployment to validate")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	report, err := check.Directory(*root, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Microformats and JSON-LD checked: representative identity, %d public posts (%d explicit publication instants), four complete archives; drafts excluded.\n", report.Posts, report.Dated)
}
