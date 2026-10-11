package atproto

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestClientRejectsUntrustedResponsesWithoutLoggingSecrets(t *testing.T) {
	for _, tc := range []struct {
		name, mediaType, body string
		transportFailure      bool
	}{
		{name: "homepage fallback", mediaType: "text/html", body: "<html>" + password + "</html>"},
		{name: "DID MIME on XRPC", mediaType: "application/did+ld+json", body: "{}"},
		{name: "malformed JSON", mediaType: "application/json", body: password},
		{name: "multiple JSON values", mediaType: "application/json", body: "{}{}"},
		{name: "oversized JSON", mediaType: "application/json", body: strings.Repeat(" ", maxBody+1)},
		{name: "transport failure", transportFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient()
			client.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				if tc.transportFailure {
					return nil, errors.New(password + access + refresh)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {tc.mediaType}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			var value any
			err := client.request(context.Background(), http.MethodGet, endpoint("com.atproto.repo.getRecord", nil), "", nil, &value)
			if err == nil {
				t.Fatal("accepted untrusted response")
			}
			for _, secret := range []string{password, access, refresh} {
				if strings.Contains(err.Error(), secret) {
					t.Fatal("error exposes credentials")
				}
			}
		})
	}
}

func TestClientRejectsRedirectBeforeFollowingIt(t *testing.T) {
	client := NewClient()
	calls := 0
	client.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"https://foreign.example/" + password}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	var value any
	err := client.request(context.Background(), http.MethodPost, endpoint("com.atproto.server.createSession", nil), "", map[string]any{"password": password}, &value)
	if err == nil || calls != 1 || strings.Contains(err.Error(), password) {
		t.Fatalf("redirect followed or unsafe error: calls=%d err=%v", calls, err)
	}
}

func TestClientHasBoundedRequestTimeout(t *testing.T) {
	client := NewClient()
	if client.http.Timeout <= 0 || client.http.Timeout > 30*time.Second {
		t.Fatal("unbounded client timeout")
	}
	client.http.Timeout = time.Millisecond
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	if err := client.request(context.Background(), http.MethodGet, endpoint("com.atproto.repo.getRecord", nil), "", nil, new(any)); err == nil {
		t.Fatal("timeout was ignored")
	}
}

func TestPaginationRejectsRepeatedCursorAndMissingRecords(t *testing.T) {
	for _, missing := range []bool{false, true} {
		client := NewClient()
		calls := 0
		client.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			if missing {
				return jsonResponse(200, map[string]any{"cursor": "repeat"}), nil
			}
			return jsonResponse(200, map[string]any{"records": []Record{}, "cursor": "repeat"}), nil
		})
		if _, err := client.ListRecords(context.Background(), PublicationCollection); err == nil || calls > 2 {
			t.Fatalf("invalid pagination accepted or looped: calls=%d err=%v", calls, err)
		}
	}
}

func TestRecordURIMustUsePinnedIdentityAndTID(t *testing.T) {
	good := testRecord(siteKey, DesiredPublication())
	if key, err := RecordKey(good.URI, PublicationCollection); err != nil || key != siteKey {
		t.Fatal("valid TID rejected")
	}
	for _, uri := range []string{
		"at://" + DID + "/" + PublicationCollection + "/self",
		"at://" + DID + "/" + PublicationCollection + "/zzzzzzzzzzzzz",
		"at://" + DID + "/" + PublicationCollection + "/0000000000000",
		good.URI + "?secret=" + password,
		strings.Replace(good.URI, DID, "dario.cat", 1),
		strings.Replace(good.URI, PublicationCollection, "site.standard.document", 1),
	} {
		if _, err := RecordKey(uri, PublicationCollection); err == nil {
			t.Fatal("invalid record URI accepted")
		}
	}
	for _, cid := range []string{"", password, strings.ToUpper(good.CID), "z" + good.CID[1:], good.CID + "a"} {
		if ValidRecordCID(cid) {
			t.Fatal("invalid CID accepted")
		}
	}
	if !ValidRecordCID(good.CID) {
		t.Fatal("valid record CID rejected")
	}
}

func TestCleanupUsesIndependentContextAfterCancellation(t *testing.T) {
	client, fake := newFake(t)
	ctx, cancel := context.WithCancel(context.Background())
	err := client.WithSession(ctx, Credentials{Handle: Handle, AppPassword: password}, func(*Session) error {
		cancel()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) || count(fake.events, "com.atproto.server.deleteSession") != 1 {
		t.Fatal("cancelled action omitted cleanup")
	}
}

func TestMalformedSessionResponseStillRevokesParsedRefreshToken(t *testing.T) {
	client := NewClient()
	cleaned := false
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "deleteSession") {
			if req.Header.Get("Authorization") != "Bearer "+refresh {
				t.Fatal("cleanup did not use parsed refresh token")
			}
			cleaned = true
			return jsonResponse(200, nil), nil
		}
		body := `{"did":"` + DID + `","handle":"` + Handle + `","accessJwt":"` + access + `","refreshJwt":"` + refresh + `"} {}`
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	err := client.WithSession(context.Background(), Credentials{Handle: Handle, AppPassword: password}, func(*Session) error { t.Fatal("malformed session was used"); return nil })
	if err == nil || !cleaned {
		t.Fatal("malformed response retained a revocable session")
	}
}

func TestSwapConflictIsNotRetriedAndCleansSession(t *testing.T) {
	value := DesiredPublication()
	value["name"] = "outdated"
	client, fake := newFake(t, testRecord(siteKey, value))
	fake.statuses["com.atproto.repo.putRecord"] = 409
	options := applyOptions(t)
	if _, err := client.Bootstrap(context.Background(), options); err == nil {
		t.Fatal("accepted CID conflict")
	}
	if count(fake.events, "com.atproto.repo.putRecord") != 1 || count(fake.events, "com.atproto.server.deleteSession") != 1 {
		t.Fatal("conflict retried or session retained")
	}
	noArtifacts(t, options.Root)
}
