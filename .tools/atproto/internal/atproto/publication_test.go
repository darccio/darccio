package atproto

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

const (
	foreignKey   = "3mgl7uesp65kn"
	siteKey      = "3mgl7uesp65ko"
	otherSiteKey = "3mgl7uesp65kp"
	password     = "private-app-password"
	access       = "private.access.jwt"
	refresh      = "private.refresh.jwt"
)

func testRecord(key string, value map[string]any) Record {
	data, _ := json.Marshal(value)
	hash := sha256.Sum256(data)
	raw := append([]byte{1, 0x71, 0x12, 0x20}, hash[:]...)
	cid := "b" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
	return Record{URI: "at://" + DID + "/" + PublicationCollection + "/" + key, CID: cid, Value: value}
}

func foreignRecord() Record {
	return testRecord(foreignKey, map[string]any{"$type": PublicationCollection, "url": "https://rizoma.pckt.blog", "name": "Rizoma", "preferences": map[string]any{"showInDiscover": true}})
}

type fakePDS struct {
	t                    *testing.T
	records              []Record
	events               []string
	inputs               map[string][]map[string]any
	statuses             map[string]int
	ack                  *Record
	readback             func(Record) Record
	did, pds, sessionDID string
	lostAck              bool
}

func newFake(t *testing.T, records ...Record) (*Client, *fakePDS) {
	t.Helper()
	fake := &fakePDS{t: t, records: records, inputs: map[string][]map[string]any{}, statuses: map[string]int{}, did: DID, pds: PDS, sessionDID: DID}
	client := NewClient()
	client.http.Transport = fake
	return client, fake
}

func jsonResponse(status int, value any) *http.Response {
	data, _ := json.Marshal(value)
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json; charset=utf-8"}}, Body: io.NopCloser(bytes.NewReader(data))}
}

func (f *fakePDS) RoundTrip(req *http.Request) (*http.Response, error) {
	method := strings.TrimPrefix(req.URL.Path, "/xrpc/")
	if req.URL.Host == "plc.directory" {
		method = "didDocument"
	}
	f.events = append(f.events, method)
	if req.Method == http.MethodGet && req.Header.Get("Authorization") != "" {
		f.t.Fatal("public GET contains authentication")
	}
	if req.Method == http.MethodPost {
		var input map[string]any
		if req.Body != nil {
			decoder := json.NewDecoder(req.Body)
			decoder.UseNumber()
			if err := decoder.Decode(&input); err != nil {
				f.t.Fatalf("request JSON: %v", err)
			}
		}
		f.inputs[method] = append(f.inputs[method], input)
		wantAuth := "Bearer " + access
		if method == "com.atproto.server.createSession" {
			wantAuth = ""
		}
		if method == "com.atproto.server.deleteSession" {
			wantAuth = "Bearer " + refresh
		}
		if req.Header.Get("Authorization") != wantAuth {
			f.t.Fatalf("wrong authentication for %s", method)
		}
	}
	if status := f.statuses[method]; status != 0 {
		return jsonResponse(status, map[string]any{"message": password + access + refresh}), nil
	}
	switch method {
	case "com.atproto.identity.resolveHandle":
		if req.URL.Query().Get("handle") != Handle {
			f.t.Fatal("wrong handle resolution")
		}
		return jsonResponse(200, map[string]any{"did": f.did}), nil
	case "didDocument":
		response := jsonResponse(200, map[string]any{"id": DID, "alsoKnownAs": []string{"at://" + Handle}, "service": []map[string]any{{"id": "#atproto_pds", "type": "AtprotoPersonalDataServer", "serviceEndpoint": f.pds}}})
		response.Header.Set("Content-Type", "application/did+ld+json; charset=utf-8")
		return response, nil
	case "com.atproto.repo.listRecords":
		if req.URL.Query().Get("repo") != DID || req.URL.Query().Get("collection") != PublicationCollection || req.URL.Query().Get("limit") != "100" {
			f.t.Fatal("wrong list query")
		}
		// Deliberately include a cursor even when this is the last record.
		index := 0
		if cursor := req.URL.Query().Get("cursor"); cursor != "" {
			if _, err := fmt.Sscan(cursor, &index); err != nil {
				f.t.Fatal("invalid cursor")
			}
		}
		if index >= len(f.records) {
			return jsonResponse(200, map[string]any{"records": []Record{}}), nil
		}
		return jsonResponse(200, map[string]any{"records": []Record{f.records[index]}, "cursor": fmt.Sprint(index + 1)}), nil
	case "com.atproto.repo.getRecord":
		for _, record := range f.records {
			if strings.HasSuffix(record.URI, "/"+req.URL.Query().Get("rkey")) {
				if f.readback != nil {
					record = f.readback(record)
				}
				return jsonResponse(200, record), nil
			}
		}
		return jsonResponse(404, map[string]any{"error": "RecordNotFound"}), nil
	case "com.atproto.server.createSession":
		input := f.inputs[method][len(f.inputs[method])-1]
		if input["identifier"] != Handle || input["password"] != password {
			f.t.Fatal("wrong login credentials")
		}
		return jsonResponse(200, map[string]any{"did": f.sessionDID, "handle": Handle, "accessJwt": access, "refreshJwt": refresh}), nil
	case "com.atproto.server.deleteSession":
		return jsonResponse(200, nil), nil
	case "com.atproto.repo.createRecord", "com.atproto.repo.putRecord":
		input := f.inputs[method][len(f.inputs[method])-1]
		if input["repo"] != DID || input["collection"] != PublicationCollection {
			f.t.Fatal("wrong write target")
		}
		if _, present := input["validate"]; present {
			f.t.Fatal("must use default known-lexicon validation")
		}
		value, ok := input["record"].(map[string]any)
		if !ok {
			f.t.Fatal("missing record")
		}
		var written Record
		if method == "com.atproto.repo.createRecord" {
			if _, present := input["rkey"]; present {
				f.t.Fatal("createRecord must let the server generate its TID")
			}
			written = testRecord(siteKey, value)
			f.records = append(f.records, written)
		} else {
			for i, record := range f.records {
				if strings.HasSuffix(record.URI, "/"+input["rkey"].(string)) {
					if input["swapRecord"] != record.CID {
						return jsonResponse(409, map[string]any{"error": "InvalidSwap"}), nil
					}
					written = testRecord(input["rkey"].(string), value)
					f.records[i] = written
				}
			}
			if written.URI == "" {
				f.t.Fatal("update targets an absent record")
			}
		}
		if f.lostAck {
			return nil, errors.New("connection lost after commit: " + password)
		}
		if f.ack != nil {
			written = *f.ack
		}
		return jsonResponse(200, map[string]any{"uri": written.URI, "cid": written.CID}), nil
	default:
		f.t.Fatalf("unexpected request: %s", method)
		return nil, errors.New("unexpected request")
	}
}

func applyOptions(t *testing.T) Options {
	t.Helper()
	return Options{Apply: true, Root: t.TempDir(), Credentials: Credentials{Handle: Handle, AppPassword: password}}
}

func noArtifacts(t *testing.T, root string) {
	t.Helper()
	for _, path := range []string{".zas/atproto-publication.json", ".well-known/site.standard.publication"} {
		if _, err := os.Stat(filepath.Join(root, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unexpected artifact %s: %v", path, err)
		}
	}
}

func count(events []string, method string) int {
	n := 0
	for _, event := range events {
		if event == method {
			n++
		}
	}
	return n
}

func TestDryRunMatchesAndPagination(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		records      []Record
		fail         bool
	}{
		{name: "zero", action: "create"},
		{name: "foreign only", action: "create", records: []Record{foreignRecord()}},
		{name: "sole match", action: "reuse", records: []Record{foreignRecord(), testRecord(siteKey, DesiredPublication())}},
		{name: "ambiguous", fail: true, records: []Record{testRecord(siteKey, DesiredPublication()), testRecord(otherSiteKey, DesiredPublication())}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, fake := newFake(t, tc.records...)
			root := t.TempDir()
			result, err := client.Bootstrap(context.Background(), Options{Root: root})
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.fail && result.Action != tc.action {
				t.Fatalf("action=%q", result.Action)
			}
			if count(fake.events, "com.atproto.repo.listRecords") != len(tc.records)+1 {
				t.Fatal("did not exhaust pagination")
			}
			if len(fake.inputs) != 0 {
				t.Fatal("dry run authenticated or mutated records")
			}
			noArtifacts(t, root)
		})
	}
}

func TestCreateVerifiesCleansSessionAndPreservesForeignRecord(t *testing.T) {
	foreign := foreignRecord()
	client, fake := newFake(t, foreign)
	options := applyOptions(t)
	result, err := client.Bootstrap(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "create" || !reflect.DeepEqual(fake.records[0], foreign) {
		t.Fatal("creation changed the foreign publication")
	}
	if count(fake.events, "com.atproto.repo.createRecord") != 1 {
		t.Fatal("wrong create count")
	}
	last := fake.events[len(fake.events)-2:]
	if !reflect.DeepEqual(last, []string{"com.atproto.server.deleteSession", "com.atproto.repo.getRecord"}) {
		t.Fatalf("cleanup/readback order: %v", last)
	}
	verification, err := os.ReadFile(filepath.Join(options.Root, ".well-known/site.standard.publication"))
	if err != nil || string(verification) != result.URI {
		t.Fatalf("verification: %q %v", verification, err)
	}
	state, err := os.ReadFile(filepath.Join(options.Root, ".zas/atproto-publication.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if json.Unmarshal(state, &saved) != nil || saved["uri"] != result.URI || saved["cid"] != result.CID {
		t.Fatal("state does not identify the verified record")
	}
	for _, secret := range []string{password, access, refresh} {
		if bytes.Contains(state, []byte(secret)) {
			t.Fatal("secret in state")
		}
	}
}

func TestReuseWritesVerifiedArtifactsWithoutLogin(t *testing.T) {
	value := DesiredPublication()
	value["theme"] = map[string]any{"custom": "preserved"}
	client, fake := newFake(t, testRecord(siteKey, value))
	options := applyOptions(t)
	options.Credentials = Credentials{}
	result, err := client.Bootstrap(context.Background(), options)
	if err != nil || result.Action != "reuse" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(fake.inputs) != 0 {
		t.Fatal("matching record requires no login")
	}
}

func TestUpdatePreservesUnknownFieldsAndUsesSwapCID(t *testing.T) {
	value := DesiredPublication()
	value["url"] = "https://dario.cat/"
	value["description"] = "old"
	value["theme"] = map[string]any{"largeNumber": json.Number("9007199254740993")}
	value["preferences"] = map[string]any{"showInDiscover": false, "platformOnly": map[string]any{"value": "keep"}}
	original := testRecord(siteKey, value)
	foreign := foreignRecord()
	client, fake := newFake(t, foreign, original)
	// A concurrent edit between listRecords and getRecord must be preserved,
	// and its newer CID (rather than the list snapshot) must guard the update.
	freshValue := make(map[string]any, len(value)+1)
	for key, value := range value {
		freshValue[key] = value
	}
	freshValue["sourceRevision"] = "concurrent edit"
	fresh := testRecord(siteKey, freshValue)
	fake.readback = func(record Record) Record {
		if len(fake.inputs["com.atproto.repo.putRecord"]) == 0 {
			fake.records[1] = fresh
			return fresh
		}
		return record
	}
	result, err := client.Bootstrap(context.Background(), applyOptions(t))
	if err != nil || result.Action != "update" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	input := fake.inputs["com.atproto.repo.putRecord"][0]
	if input["rkey"] != siteKey || input["swapRecord"] != fresh.CID || fresh.CID == original.CID {
		t.Fatal("update omitted the exact record CID")
	}
	if !reflect.DeepEqual(fake.records[0], foreign) {
		t.Fatal("changed Rizoma")
	}
	updated := fake.records[1].Value
	if updated["sourceRevision"] != "concurrent edit" {
		t.Fatal("lost the edit discovered during public readback")
	}
	if !reflect.DeepEqual(updated["theme"], original.Value["theme"]) {
		t.Fatal("unknown field or number was changed")
	}
	preferences := updated["preferences"].(map[string]any)
	if preferences["showInDiscover"] != true || !reflect.DeepEqual(preferences["platformOnly"], value["preferences"].(map[string]any)["platformOnly"]) {
		t.Fatal("unknown preferences were lost")
	}
}

func TestCanonicalRootVariantsUpdateExistingPublication(t *testing.T) {
	for _, variant := range []string{"https://DARIO.cat", "HTTPS://dario.cat", "https://dario.cat:443/"} {
		t.Run(variant, func(t *testing.T) {
			value := DesiredPublication()
			value["url"] = variant
			client, fake := newFake(t, testRecord(siteKey, value))
			result, err := client.Bootstrap(context.Background(), applyOptions(t))
			if err != nil || result.Action != "update" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if count(fake.events, "com.atproto.repo.createRecord") != 0 || len(fake.records) != 1 {
				t.Fatal("canonical URL variation created a duplicate")
			}
		})
	}
	for _, variant := range []string{"http://dario.cat", "https://dario.cat:444", "https://dario.cat/posts/", "https://dario.cat?", "https://dario.cat?q=1", "https://dario.cat#", "https://dario.cat#anchor", "https://user@dario.cat", "https://dario.cat/%2f"} {
		if canonicalRoot(variant) {
			t.Fatalf("non-root publication matched: %s", variant)
		}
	}
}

func TestMutationFailuresCleanupAndNeverWriteArtifacts(t *testing.T) {
	valid := testRecord(siteKey, DesiredPublication())
	for _, tc := range []struct {
		name  string
		setup func(*fakePDS)
	}{
		{name: "create failure", setup: func(f *fakePDS) { f.statuses["com.atproto.repo.createRecord"] = 503 }},
		{name: "wrong DID", setup: func(f *fakePDS) {
			bad := valid
			bad.URI = strings.Replace(bad.URI, DID, "did:plc:foreign", 1)
			f.ack = &bad
		}},
		{name: "wrong collection", setup: func(f *fakePDS) {
			bad := valid
			bad.URI = strings.Replace(bad.URI, PublicationCollection, "site.standard.document", 1)
			f.ack = &bad
		}},
		{name: "self key", setup: func(f *fakePDS) {
			bad := valid
			bad.URI = "at://" + DID + "/" + PublicationCollection + "/self"
			f.ack = &bad
		}},
		{name: "bad CID", setup: func(f *fakePDS) { bad := valid; bad.CID = password; f.ack = &bad }},
		{name: "readback mismatch", setup: func(f *fakePDS) {
			f.readback = func(r Record) Record { r.Value = DesiredPublication(); r.Value["name"] = password; return r }
		}},
		{name: "readback CID mismatch", setup: func(f *fakePDS) {
			f.readback = func(r Record) Record { r.CID = foreignRecord().CID; return r }
		}},
		{name: "logout failure", setup: func(f *fakePDS) { f.statuses["com.atproto.server.deleteSession"] = 500 }},
		{name: "wrong session account", setup: func(f *fakePDS) { f.sessionDID = "did:plc:foreign" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, fake := newFake(t)
			tc.setup(fake)
			options := applyOptions(t)
			_, err := client.Bootstrap(context.Background(), options)
			if err == nil {
				t.Fatal("expected failure")
			}
			for _, secret := range []string{password, access, refresh} {
				if strings.Contains(err.Error(), secret) {
					t.Fatal("secret in error")
				}
			}
			if count(fake.events, "com.atproto.server.deleteSession") != 1 {
				t.Fatal("session cleanup not attempted")
			}
			if tc.name == "logout failure" && count(fake.events, "com.atproto.repo.getRecord") != 0 {
				t.Fatal("cleanup failure must stop before artifacts/readback")
			}
			noArtifacts(t, options.Root)
		})
	}
}

func TestInterruptedCreationRecoversByRelistingWithoutRetry(t *testing.T) {
	client, fake := newFake(t, foreignRecord())
	fake.lostAck = true
	options := applyOptions(t)
	_, err := client.Bootstrap(context.Background(), options)
	if err == nil || strings.Contains(err.Error(), password) {
		t.Fatalf("unsafe/missing failure: %v", err)
	}
	if count(fake.events, "com.atproto.repo.createRecord") != 1 || count(fake.events, "com.atproto.server.deleteSession") != 1 {
		t.Fatal("creation retried or cleanup omitted")
	}
	noArtifacts(t, options.Root)
	fake.lostAck = false
	options.Credentials = Credentials{}
	result, err := client.Bootstrap(context.Background(), options)
	if err != nil || result.Action != "reuse" {
		t.Fatalf("recovery=%+v err=%v", result, err)
	}
	if count(fake.events, "com.atproto.repo.createRecord") != 1 || count(fake.events, "com.atproto.server.createSession") != 1 || len(fake.records) != 2 {
		t.Fatal("recovery created a duplicate or logged in")
	}
}

func TestIdentityChangeStopsBeforePDSAndAuthentication(t *testing.T) {
	for _, field := range []string{"did", "pds"} {
		t.Run(field, func(t *testing.T) {
			client, fake := newFake(t)
			if field == "did" {
				fake.did = "did:plc:foreign"
			} else {
				fake.pds = "https://foreign.example"
			}
			options := applyOptions(t)
			if _, err := client.Bootstrap(context.Background(), options); err == nil {
				t.Fatal("accepted changed identity")
			}
			if count(fake.events, "com.atproto.repo.listRecords") != 0 || len(fake.inputs) != 0 {
				t.Fatal("used unverified PDS/account")
			}
			noArtifacts(t, options.Root)
		})
	}
}

func TestPinnedPublicationMatchesAuthoritativeSchema(t *testing.T) {
	data, err := os.ReadFile("testdata/site.standard.publication.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		CID   string `json:"cid"`
		Value struct {
			ID   string `json:"id"`
			Defs map[string]struct {
				Key    string `json:"key"`
				Record struct {
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type, Format            string
						MaxLength, MaxGraphemes int
					} `json:"properties"`
				} `json:"record"`
				Properties map[string]struct{ Type string } `json:"properties"`
			} `json:"defs"`
		} `json:"value"`
	}
	if json.Unmarshal(data, &fixture) != nil || fixture.CID != "bafyreiapvnarfn3nl2wvg27hsylli5zqvou7a74p7od5mu4detprtejapy" || fixture.Value.ID != PublicationCollection {
		t.Fatal("wrong authoritative fixture")
	}
	main := fixture.Value.Defs["main"]
	if main.Key != "tid" || !reflect.DeepEqual(main.Record.Required, []string{"url", "name"}) || main.Record.Properties["url"].Format != "uri" || fixture.Value.Defs["preferences"].Properties["showInDiscover"].Type != "boolean" {
		t.Fatal("publication schema changed")
	}
	desired, err := ReadDesired("../../publication.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "description"} {
		text := desired[key].(string)
		property := main.Record.Properties[key]
		// Rune count conservatively bounds grapheme count for these fixed strings.
		if property.Type != "string" || len(text) > property.MaxLength || utf8.RuneCountInString(text) > property.MaxGraphemes {
			t.Fatalf("%s exceeds schema limits", key)
		}
	}
}
