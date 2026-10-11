package vanity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const MaxBody = 1 << 20
const MaxRedirects = 5

type LiveOptions struct {
	BaseURL string
	Timeout time.Duration
	// Transport is optional; TLS test servers can supply their trusted transport.
	Transport http.RoundTripper
}

func validBase(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("base URL must be an HTTPS origin without credentials, path, query, or fragment")
	}
	return u, nil
}

func redirectPolicy(origin *url.URL) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) > MaxRedirects {
			return fmt.Errorf("more than %d redirects", MaxRedirects)
		}
		u := req.URL
		if u.Scheme != "https" || !strings.EqualFold(u.Host, origin.Host) || u.User != nil || u.Fragment != "" {
			return fmt.Errorf("redirect must stay on HTTPS origin %s", origin.Host)
		}
		if u.RawQuery != "go-get=1" {
			return fmt.Errorf("redirect lost or changed go-get=1")
		}
		return nil
	}
}

// CheckLive checks Go's bare import route and the site's trailing-slash route.
// Every response must carry that route's metadata; a 200 homepage is a failure.
func CheckLive(ctx context.Context, options LiveOptions) error {
	base, err := validBase(options.BaseURL)
	if err != nil {
		return err
	}
	if options.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	client := &http.Client{Timeout: options.Timeout, Transport: options.Transport, CheckRedirect: redirectPolicy(base)}
	var problems []error
	for _, route := range Routes() {
		for _, path := range []string{strings.TrimSuffix(route.Path, "/"), route.Path} {
			u := *base
			u.Path, u.RawQuery = path, "go-get=1"
			if err := checkResponse(ctx, client, u.String(), route); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", u.String(), err))
			}
		}
	}
	return errors.Join(problems...)
}

func checkResponse(ctx context.Context, client *http.Client, address string, route Route) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "dario.cat-vanity-check/1")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d, want 200", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "text/html" {
		return fmt.Errorf("Content-Type %q, want text/html", response.Header.Get("Content-Type"))
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxBody+1))
	if err != nil {
		return err
	}
	if len(data) > MaxBody {
		return fmt.Errorf("response exceeds %d bytes", MaxBody)
	}
	metadata, err := Parse(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return Validate(route, metadata)
}
