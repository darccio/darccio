package vanity

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func routeForPath(path string) Route {
	for _, route := range Routes() {
		if strings.TrimSuffix(path, "/") == strings.TrimSuffix(route.Path, "/") {
			return route
		}
	}
	return Route{}
}

func TestLiveChecksBareAndSlashWithQuery(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]int{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.RequestURI()]++
		mu.Unlock()
		if r.URL.RawQuery != "go-get=1" {
			t.Errorf("query %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, fragment(routeForPath(r.URL.Path)))
	}))
	defer server.Close()
	if err := CheckLive(context.Background(), LiveOptions{BaseURL: server.URL, Timeout: time.Second, Transport: server.Client().Transport}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 14 {
		t.Fatalf("got %d paths, want 14: %v", len(seen), seen)
	}
	for _, route := range Routes() {
		for _, path := range []string{route.Path, strings.TrimSuffix(route.Path, "/")} {
			if seen[path+"?go-get=1"] != 1 {
				t.Errorf("missing or repeated %s", path)
			}
		}
	}
}

func TestLiveResponsesAndRedirects(t *testing.T) {
	cases := []struct {
		name      string
		handler   http.HandlerFunc
		wantError bool
	}{
		{"same host slash redirect", func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, r.URL.Path+"/?go-get=1", http.StatusMovedPermanently)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, fragment(routeForPath(r.URL.Path)))
		}, false},
		{"homepage fallback", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><head></head><body>home</body></html>")
		}, true},
		{"wrong MIME", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, fragment(routeForPath(r.URL.Path)))
		}, true},
		{"error status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }, true},
		{"oversized body", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, strings.Repeat(" ", MaxBody+1))
		}, true},
		{"lost query", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, r.URL.Path+"/", http.StatusFound) }, true},
		{"changed query", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, r.URL.Path+"/?go-get=2", http.StatusFound)
		}, true},
		{"cross host", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://example.org"+r.URL.Path+"?go-get=1", http.StatusFound)
		}, true},
		{"HTTP downgrade", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://"+r.Host+r.URL.Path+"?go-get=1", http.StatusFound)
		}, true},
		{"redirect loop", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, r.URL.Path+"/?go-get=1", http.StatusFound)
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(tc.handler)
			defer server.Close()
			err := CheckLive(context.Background(), LiveOptions{BaseURL: server.URL, Timeout: time.Second, Transport: server.Client().Transport})
			if (err != nil) != tc.wantError {
				t.Fatalf("error %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

func TestLiveTimeoutAndCancellation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	}))
	defer server.Close()
	err := CheckLive(context.Background(), LiveOptions{BaseURL: server.URL, Timeout: 5 * time.Millisecond, Transport: server.Client().Transport})
	if err == nil {
		t.Fatal("request timeout passed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = CheckLive(ctx, LiveOptions{BaseURL: server.URL, Timeout: time.Second, Transport: server.Client().Transport})
	if err == nil {
		t.Fatal("cancelled request passed")
	}
}

func TestLiveBaseRejectsUnsafeOrAmbiguousOrigins(t *testing.T) {
	for _, base := range []string{"http://dario.cat", "https://user:pass@dario.cat", "https://dario.cat/subpath", "https://dario.cat?go-get=1", "https://dario.cat#fragment", "https://"} {
		if _, err := validBase(base); err == nil {
			t.Errorf("accepted %s", base)
		}
	}
	base, _ := url.Parse("https://dario.cat")
	policy := redirectPolicy(base)
	for _, address := range []string{"https://dario.cat/?go-get=1#fragment", "https://user@dario.cat/?go-get=1", "https://dario.cat:444/?go-get=1"} {
		u, _ := url.Parse(address)
		if err := policy(&http.Request{URL: u}, nil); err == nil {
			t.Errorf("accepted redirect %s", address)
		}
	}
}
