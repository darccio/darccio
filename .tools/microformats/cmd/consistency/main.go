package main

import (
	"flag"
	"fmt"
	"os"

	"dario.cat/microformats-check/internal/consistency"
)

func main() {
	root := flag.String("root", "../..", "site source root")
	dir := flag.String("dir", "../../.zas/deploy", "completed deployment to validate")
	plan := flag.String("standard-plan", "", "required offline Standard.site document plan")
	flag.Parse()
	if flag.NArg() != 0 || *plan == "" {
		fmt.Fprintln(os.Stderr, "use -standard-plan FILE with no positional arguments")
		os.Exit(2)
	}
	report, err := consistency.Directory(*root, *dir, *plan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Publishing formats agree: %d public posts (%d explicit instants, %d partial dates), %d feeds/%d items, %d approved Standard.site plans/%d verified documents; %d drafts or unpublished articles excluded.\n", report.Posts, report.Dated, report.Partial, report.Feeds, report.Items, report.Approved, report.Published, report.Excluded)
}
