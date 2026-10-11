package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"dario.cat/atproto-tools/internal/atproto"
)

func main() {
	var options atproto.Options
	var recordPath string
	flag.StringVar(&options.Root, "root", ".", "site root for input and output paths")
	flag.StringVar(&recordPath, "record", "", "root-relative publication JSON (default: pinned metadata)")
	flag.BoolVar(&options.Apply, "apply", false, "explicitly allow account writes and verified local artifacts")
	flag.StringVar(&options.StatePath, "state", ".zas/atproto-publication.json", "root-relative verified state path")
	flag.StringVar(&options.WellKnownPath, "well-known", ".well-known/site.standard.publication", "root-relative verification path")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	if recordPath != "" {
		var err error
		options.Desired, err = atproto.ReadDesired(atproto.SitePath(options.Root, recordPath))
		if err != nil {
			fail(err)
		}
	}
	if options.Apply {
		options.Credentials = atproto.Credentials{Handle: os.Getenv("ATP_HANDLE"), AppPassword: os.Getenv("ATP_APP_PASSWORD")}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := atproto.NewClient().Bootstrap(ctx, options)
	if err != nil {
		fail(err)
	}
	mode := "dry run"
	if options.Apply {
		mode = "applied"
	}
	fmt.Printf("%s: %s publication for https://dario.cat\n", mode, result.Action)
	if result.URI != "" {
		fmt.Println(result.URI)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "publication:", err)
	os.Exit(1)
}
