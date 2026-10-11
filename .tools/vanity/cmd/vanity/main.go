package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"dario.cat/site-tools/internal/vanity"
)

func main() {
	dir := flag.String("dir", ".zas/deploy", "assembled deployment directory (local mode)")
	live := flag.Bool("live", false, "check all bare and trailing-slash HTTPS import routes")
	baseURL := flag.String("base-url", "https://dario.cat", "HTTPS origin (live mode)")
	timeout := flag.Duration("timeout", 10*time.Second, "timeout per live request, including redirects")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	var err error
	if *live {
		err = vanity.CheckLive(context.Background(), vanity.LiveOptions{BaseURL: *baseURL, Timeout: *timeout})
	} else {
		err = vanity.CheckDir(*dir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Vanity metadata matches the reviewed seven-package baseline.")
}
