package document

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"dario.cat/atproto-tools/internal/atproto"
	"golang.org/x/net/html"
)

const publicationURI = "at://" + atproto.DID + "/site.standard.publication/3mxmtmkzqgp2t"
const initialDate = "2018-03-28T23:26:00+02:00"

func writeFixture(t *testing.T, root, path, body string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func approve(t *testing.T, root string, approvals ...Approval) {
	t.Helper()
	if approvals == nil {
		approvals = []Approval{}
	}
	data, err := json.Marshal(Allowlist{Version: 1, Documents: approvals})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, AllowlistPath, string(data))
}

func articleFixture(t *testing.T, root, path, date, title, extra string) {
	t.Helper()
	rel := strings.TrimPrefix(path, "/")
	writeFixture(t, root, rel+"index.md", fmt.Sprintf("<!--\narticle: true\ndate: %q\nupdated: \"2025-02-01T10:11:12.123456789Z\"\nsummary: \"Resum complet\"\ntags: [go, social]\nlanguage: ca\n%s\n-->\n\n# %s\n\nPrimer paràgraf amb **èmfasi**.\n\nÚltim paràgraf complet.\n", date, extra, title))
	writeFixture(t, root, ".zas/deploy/"+rel+"index.html", fmt.Sprintf("<!doctype html><html lang=ca><head><link rel=canonical href=\"https://dario.cat%s\"></head><body><nav>Private layout header</nav><article class=h-entry lang=ca><header><time class=dt-published datetime=\"%s\"></time><p class=p-summary>Resum complet</p></header><div class=e-content><h1>%s</h1><p>Primer paràgraf amb <strong>èmfasi</strong>.</p><p>Últim paràgraf complet.</p></div><footer>Layout tags</footer></article></body></html>", path, date, title))
}

func fixture(t *testing.T) Options {
	t.Helper()
	root := t.TempDir()
	articleFixture(t, root, "/posts/example/", initialDate, "Article complet", "")
	approve(t, root, Approval{Path: "/posts/example/", PublishedAt: initialDate})
	writeFixture(t, root, ManifestPath, "{}\n")
	writeFixture(t, root, PublicationPath, `{"uri":"`+publicationURI+`","cid":"bafyreiek66jvfojawan5jmi3sarbool65zp2ixftxi6ddgaokf32ig4hqq","did":"`+atproto.DID+`","pds":"`+atproto.PDS+`","url":"https://dario.cat"}`)
	return Options{Root: root, Credentials: atproto.Credentials{Handle: atproto.Handle, AppPassword: "fake-only-password"}}
}

func clone(value map[string]any) map[string]any {
	data, _ := json.Marshal(value)
	var result map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	decoder.Decode(&result)
	return result
}

func record(uri string, value map[string]any) atproto.Record {
	data, _ := json.Marshal(value)
	hash := sha256.Sum256(data)
	raw := append([]byte{1, 0x71, 0x12, 0x20}, hash[:]...)
	return atproto.Record{URI: uri, CID: "b" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)), Value: clone(value)}
}

type fakeNetwork struct {
	records                                             map[string]atproto.Record
	events                                              []string
	logins, writes, deletes, cleanups                   int
	lostAck, swapConflict, wrongReadback, logoutFailure bool
}

func fake() *fakeNetwork { return &fakeNetwork{records: map[string]atproto.Record{}} }
func (f *fakeNetwork) VerifyIdentity(context.Context) error {
	f.events = append(f.events, "identity")
	return nil
}
func (f *fakeNetwork) ListRecords(_ context.Context, collection string) ([]atproto.Record, error) {
	if collection != atproto.DocumentCollection {
		return nil, errors.New("wrong collection")
	}
	f.events = append(f.events, "list")
	records := []atproto.Record{}
	for _, r := range f.records {
		records = append(records, r)
	}
	return records, nil
}
func (f *fakeNetwork) GetRecord(_ context.Context, uri, collection string) (atproto.Record, error) {
	f.events = append(f.events, "get")
	if uri == publicationURI && collection == atproto.PublicationCollection {
		return record(uri, map[string]any{"$type": atproto.PublicationCollection, "url": "https://dario.cat"}), nil
	}
	r, present := f.records[uri]
	if !present {
		return atproto.Record{}, errors.New("record not found")
	}
	if f.wrongReadback {
		r.CID = "invalid-public-CID"
	}
	return r, nil
}
func (f *fakeNetwork) WithDocumentSession(_ context.Context, credentials atproto.Credentials, action func(atproto.DocumentWriter) error) (err error) {
	if credentials.Handle != atproto.Handle || credentials.AppPassword == "" {
		return errors.New("missing fake credentials")
	}
	f.logins++
	f.events = append(f.events, "login")
	defer func() {
		f.cleanups++
		f.events = append(f.events, "cleanup")
		if f.logoutFailure {
			err = errors.Join(err, errors.New("logout failed"))
		}
	}()
	return action(f)
}
func (f *fakeNetwork) PutDocument(_ context.Context, input atproto.Record) (atproto.Record, error) {
	f.writes++
	f.events = append(f.events, "put")
	previous, present := f.records[input.URI]
	if f.swapConflict || (present && input.CID != previous.CID) || (!present && input.CID != "") {
		return atproto.Record{}, errors.New("InvalidSwap")
	}
	written := record(input.URI, input.Value)
	f.records[input.URI] = written
	if f.lostAck {
		return atproto.Record{}, errors.New("lost acknowledgement")
	}
	return written, nil
}
func (f *fakeNetwork) DeleteDocument(_ context.Context, input atproto.Record) error {
	f.deletes++
	f.events = append(f.events, "delete")
	previous, present := f.records[input.URI]
	if !present || previous.CID != input.CID {
		return errors.New("InvalidSwap")
	}
	delete(f.records, input.URI)
	if f.lostAck {
		return errors.New("lost deletion acknowledgement")
	}
	return nil
}

func run(t *testing.T, f Network, options Options) Plan {
	t.Helper()
	plan, err := Run(context.Background(), f, options)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestOfflineDryRunFullBodyAndEmptyApproval(t *testing.T) {
	options := fixture(t)
	f := fake()
	before, _ := os.ReadFile(filepath.Join(options.Root, ManifestPath))
	plan := run(t, f, options)
	if len(f.events) != 0 || plan.Documents[0].Action != "create" {
		t.Fatal("dry-run accessed network or produced wrong action")
	}
	text := plan.Documents[0].Record["textContent"].(string)
	if text != "Article complet\nPrimer paràgraf amb èmfasi.\nÚltim paràgraf complet." || strings.Contains(text, "Layout") {
		t.Fatalf("incorrect complete plaintext: %q", text)
	}
	if !reflect.DeepEqual(plan.Documents[0].Record["tags"], []string{"go", "social", "lang:ca"}) {
		t.Fatal("language tag missing")
	}
	if _, err := os.Stat(filepath.Join(options.Root, ".tools/atproto/.state")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dry-run wrote operational state")
	}
	after, _ := os.ReadFile(filepath.Join(options.Root, ManifestPath))
	if string(before) != string(after) {
		t.Fatal("dry-run changed public manifest")
	}
	approve(t, options.Root)
	options.Apply = true
	options.Credentials = atproto.Credentials{}
	plan = run(t, f, options)
	if len(plan.Documents) != 0 || len(f.events) != 0 {
		t.Fatal("empty reviewed allowlist accessed network")
	}
}

func TestApplyBatchUsesOneSessionAndSecondRunDoesNotWrite(t *testing.T) {
	options := fixture(t)
	articleFixture(t, options.Root, "/posts/second/", initialDate, "Article segon", "")
	approve(t, options.Root, Approval{Path: "/posts/example/", PublishedAt: initialDate}, Approval{Path: "/posts/second/", PublishedAt: initialDate})
	options.Apply = true
	f := fake()
	plan := run(t, f, options)
	if len(plan.Documents) != 2 || f.logins != 1 || f.cleanups != 1 || f.writes != 2 {
		t.Fatal("batch did not use exactly one cleaned session")
	}
	lastCleanup := -1
	for i, event := range f.events {
		if event == "cleanup" {
			lastCleanup = i
		}
	}
	if lastCleanup < 0 || !reflect.DeepEqual(f.events[lastCleanup+1:], []string{"get", "get"}) {
		t.Fatal("public readbacks did not follow session cleanup")
	}
	options.Credentials = atproto.Credentials{}
	plan = run(t, f, options)
	for _, entry := range plan.Documents {
		if entry.Action != "unchanged" {
			t.Fatal("second run not unchanged")
		}
	}
	if f.logins != 1 || f.writes != 2 {
		t.Fatal("unchanged batch authenticated or mutated")
	}
	state, err := os.ReadFile(filepath.Join(options.Root, StatePath))
	if err != nil || strings.Contains(string(state), "fake-only-password") {
		t.Fatal("private state missing or includes credential")
	}
	info, _ := os.Stat(filepath.Join(options.Root, StatePath))
	if info.Mode().Perm() != 0600 {
		t.Fatal("private state is not owner-only")
	}
	manifest, err := readManifest(filepath.Join(options.Root, ManifestPath))
	if err != nil || len(manifest) != 2 {
		t.Fatal("public verified map incomplete")
	}
}

func TestUnknownFieldsAndRemoteCorrectionsSurviveUnchangedSource(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	first := run(t, f, options)
	uri := first.Documents[0].URI
	value := clone(f.records[uri].Value)
	value["platform"] = map[string]any{"largeNumber": json.Number("9007199254740993")}
	f.records[uri] = record(uri, value)
	second := run(t, f, options)
	if second.Documents[0].Action != "unchanged" || !equalJSON(second.Documents[0].Record, value) || f.writes != 1 {
		t.Fatal("publisher overwrote unknown remote fields")
	}
	value["title"] = "Owner-corrected title"
	f.records[uri] = record(uri, value)
	if _, err := Run(context.Background(), f, options); err == nil || !strings.Contains(err.Error(), "remote title correction") {
		t.Fatal("unreflected remote correction was not preserved through an explicit conflict")
	}
	articleFixture(t, options.Root, "/posts/example/", initialDate, "Source changed title", "")
	if _, err := Run(context.Background(), f, options); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatal("conflicting source/remote edits were not rejected")
	}
	if f.writes != 1 {
		t.Fatal("conflict wrote a record")
	}
}

func TestCorrectedPublicationDateRetainsPublicIdentityWithoutPrivateState(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	first := run(t, f, options)
	uri := first.Documents[0].URI
	if err := os.Remove(filepath.Join(options.Root, StatePath)); err != nil {
		t.Fatal(err)
	}
	newDate := "2018-03-29T23:26:00+02:00"
	articleFixture(t, options.Root, "/posts/example/", newDate, "Article complet", "")
	approve(t, options.Root, Approval{Path: "/posts/example/", PublishedAt: newDate, CorrectPublishedAt: true})
	second := run(t, f, options)
	if second.Documents[0].URI != uri || second.Documents[0].Action != "update" || f.writes != 2 || len(f.records) != 1 || f.records[uri].Value["publishedAt"] != newDate {
		t.Fatal("date correction changed identity or did not update approved date")
	}
}

func TestInterruptedApplyRecoversByPublicReadWithoutRetry(t *testing.T) {
	for _, failure := range []string{"lost acknowledgement", "cleanup", "readback"} {
		t.Run(failure, func(t *testing.T) {
			options := fixture(t)
			options.Apply = true
			f := fake()
			f.lostAck = failure == "lost acknowledgement"
			f.logoutFailure = failure == "cleanup"
			f.wrongReadback = failure == "readback"
			if _, err := Run(context.Background(), f, options); err == nil {
				t.Fatal("expected interruption failure")
			}
			if f.writes != 1 || f.cleanups != 1 {
				t.Fatal("mutation blindly retried or cleanup omitted")
			}
			manifest, _ := readManifest(filepath.Join(options.Root, ManifestPath))
			if len(manifest) != 0 {
				t.Fatal("unverified interrupted write entered public map")
			}
			state, err := readState(filepath.Join(options.Root, StatePath), publicationURI)
			if err != nil || state.Documents["/posts/example/"].Pending == nil {
				t.Fatal("interrupted intent not persisted")
			}
			f.lostAck = false
			f.logoutFailure = false
			f.wrongReadback = false
			options.Credentials = atproto.Credentials{}
			plan := run(t, f, options)
			if plan.Documents[0].Action != "unchanged" || f.writes != 1 || f.logins != 1 {
				t.Fatal("interrupted committed record created another write")
			}
		})
	}
}

func TestCollisionAndAmbiguousRemotePathsStopBeforeLogin(t *testing.T) {
	options := fixture(t)
	initial := run(t, nil, options)
	options.Apply = true
	for _, kind := range []string{"key collision", "duplicate path"} {
		t.Run(kind, func(t *testing.T) {
			f := fake()
			value := clone(initial.Documents[0].Record)
			if kind == "key collision" {
				value["path"] = "/posts/someone-else/"
			}
			f.records[initial.Documents[0].URI] = record(initial.Documents[0].URI, value)
			if kind == "duplicate path" {
				other := "at://" + atproto.DID + "/site.standard.document/3mxmtmkzqgp2u"
				f.records[other] = record(other, value)
			}
			if _, err := Run(context.Background(), f, options); err == nil {
				t.Fatal("accepted collision or duplicate path")
			}
			if f.logins != 0 || f.writes != 0 {
				t.Fatal("collision reached mutation")
			}
		})
	}
}

func TestTIDEncodingAndLocalHashCollision(t *testing.T) {
	date, _ := time.Parse(time.RFC3339Nano, initialDate)
	key, err := StableTID(date, "/posts/example/")
	if err != nil {
		t.Fatal(err)
	}
	if key != "3fcjynoduk2zd" {
		t.Fatalf("FNV-1a-32/date fixture changed: %s", key)
	}
	uri := "at://" + atproto.DID + "/site.standard.document/" + key
	if _, err := atproto.RecordKey(uri, atproto.DocumentCollection); err != nil {
		t.Fatal(err)
	}
	var value uint64
	for _, char := range key {
		value = value<<5 | uint64(strings.IndexRune("234567abcdefghijklmnopqrstuvwxyz", char))
	}
	if value>>10 != uint64(date.UnixMicro()) {
		t.Fatal("TID did not encode original publication instant")
	}
	shifted := date.In(time.FixedZone("UTC", 0))
	key2, _ := StableTID(shifted, "/posts/example/")
	if key2 != key {
		t.Fatal("equivalent timezone changed TID")
	}
	seen := map[string]string{}
	var a, b string
	for i := 0; i < 2000; i++ {
		path := fmt.Sprintf("/posts/collision-%d/", i)
		key, _ := StableTID(date, path)
		if old, ok := seen[key]; ok {
			a, b = old, path
			break
		}
		seen[key] = path
	}
	if a == "" {
		t.Fatal("collision fixture not found")
	}
	options := fixture(t)
	articleFixture(t, options.Root, a, initialDate, "First collision", "")
	articleFixture(t, options.Root, b, initialDate, "Second collision", "")
	approve(t, options.Root, Approval{Path: a, PublishedAt: initialDate}, Approval{Path: b, PublishedAt: initialDate})
	if _, err := Run(context.Background(), nil, options); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatal("local 10-bit hash collision was not detected")
	}
}

func TestRemovedApprovalDoesNotDeleteAndExplicitDeleteIsIdempotent(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	first := run(t, f, options)
	approve(t, options.Root)
	plan := run(t, f, options)
	if len(plan.Documents) != 0 || f.deletes != 0 || len(f.records) != 1 {
		t.Fatal("removing approval implied deletion")
	}
	options.Delete = []string{"/posts/example/"}
	plan = run(t, f, options)
	if plan.Documents[0].Action != "delete" || f.deletes != 1 || len(f.records) != 0 {
		t.Fatal("explicit deletion not applied")
	}
	manifest, _ := readManifest(filepath.Join(options.Root, ManifestPath))
	if len(manifest) != 0 {
		t.Fatal("deleted record stayed in public manifest")
	}
	options.Credentials = atproto.Credentials{}
	plan = run(t, f, options)
	if plan.Documents[0].Action != "unchanged" || plan.Documents[0].URI != first.Documents[0].URI || f.deletes != 1 {
		t.Fatal("repeated deletion not unchanged")
	}
}

func TestMalformedMetadataStaleBodyAndExclusions(t *testing.T) {
	for _, extra := range []string{"draft: true", "private: true", "feed: false", "publish: false", "atproto: false", "atproto: \"false\"", "date_precision: day", "tags: [go, 12]"} {
		t.Run(extra, func(t *testing.T) {
			options := fixture(t)
			articleFixture(t, options.Root, "/posts/example/", initialDate, "Article complet", extra)
			if _, err := Run(context.Background(), nil, options); err == nil {
				t.Fatal("excluded or malformed source approved")
			}
		})
	}
	options := fixture(t)
	file := filepath.Join(options.Root, ".zas/deploy/posts/example/index.html")
	input, _ := os.ReadFile(file)
	if err := os.WriteFile(file, []byte(strings.Replace(string(input), "Últim paràgraf complet.", "Truncated", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), nil, options); err == nil {
		t.Fatal("stale/truncated body accepted")
	}
}

func TestCandidateDiscoveryNeverApprovesAndSkipsPrivateDirectories(t *testing.T) {
	options := fixture(t)
	approve(t, options.Root)
	articleFixture(t, options.Root, "/posts/old/", "2010-01-02T00:00:00Z", "Old article", "")
	writeFixture(t, options.Root, "posts/private/index.md", "secret malformed metadata <!--{{")
	writeFixture(t, options.Root, "CV/index.md", "private CV")
	options.Candidates = true
	f := fake()
	plan := run(t, f, options)
	if len(plan.Documents) != 0 || len(plan.Candidates) != 2 || len(f.events) != 0 {
		t.Fatal("candidate discovery accessed private sources or network")
	}
	for _, candidate := range plan.Candidates {
		if candidate.Path == "/posts/old/" && candidate.Eligible {
			t.Fatal("pre-2013 candidate marked eligible")
		}
	}
	options.Apply = true
	if _, err := Run(context.Background(), f, options); err == nil {
		t.Fatal("candidate discovery could be applied")
	}
}

func TestSwapFailureAndStateLockDoNotWritePublicArtifacts(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	f.swapConflict = true
	if _, err := Run(context.Background(), f, options); err == nil {
		t.Fatal("swap conflict accepted")
	}
	if f.writes != 1 || f.cleanups != 1 {
		t.Fatal("swap retry or session leak")
	}
	manifest, _ := readManifest(filepath.Join(options.Root, ManifestPath))
	if len(manifest) != 0 {
		t.Fatal("swap failure exported public identity")
	}
	unlock, err := lockState(filepath.Join(options.Root, StatePath))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := Run(context.Background(), f, options); err == nil {
		t.Fatal("concurrent process acquired state lock")
	}
}

func TestInputsAndRenderedParentsRejectSymlinks(t *testing.T) {
	options := fixture(t)
	path := filepath.Join(options.Root, ".zas/deploy/posts/example/index.html")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", path); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), nil, options); err == nil {
		t.Fatal("symlink body accepted")
	}
}

func TestCIWithoutPrivateStateUsesReviewedSourceAndKeepsUnknownFields(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	first := run(t, f, options)
	uri := first.Documents[0].URI
	value := clone(f.records[uri].Value)
	value["platformExtension"] = map[string]any{"retained": true}
	f.records[uri] = record(uri, value)
	if err := os.Remove(filepath.Join(options.Root, StatePath)); err != nil {
		t.Fatal(err)
	}
	articleFixture(t, options.Root, "/posts/example/", initialDate, "Reviewed source edit", "")
	second := run(t, f, options)
	if second.Documents[0].URI != uri || second.Documents[0].Action != "update" || f.records[uri].Value["title"] != "Reviewed source edit" || !equalJSON(f.records[uri].Value["platformExtension"], value["platformExtension"]) || len(f.records) != 1 {
		t.Fatal("fresh CI did not reconcile authoritative source safely at existing key")
	}
}

func TestRemotePublicationDateMustBeReconciledWithoutOverwritingCorrection(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	first := run(t, f, options)
	uri := first.Documents[0].URI
	value := clone(f.records[uri].Value)
	corrected := "2018-03-27T12:34:56Z"
	value["publishedAt"] = corrected
	f.records[uri] = record(uri, value)
	if _, err := Run(context.Background(), f, options); err == nil || !strings.Contains(err.Error(), "publishedAt") {
		t.Fatal("remote date correction did not require source reconciliation")
	}
	if f.writes != 1 || f.records[uri].Value["publishedAt"] != corrected {
		t.Fatal("publisher overwrote remote publication date")
	}
	articleFixture(t, options.Root, "/posts/example/", corrected, "Article complet", "")
	approve(t, options.Root, Approval{Path: "/posts/example/", PublishedAt: corrected})
	second := run(t, f, options)
	if second.Documents[0].Action != "unchanged" || second.Documents[0].URI != uri || f.writes != 1 {
		t.Fatal("reconciled date caused a new identity or rewrite")
	}
}

func TestInterruptedDeletionRecoversWithoutSecondSession(t *testing.T) {
	options := fixture(t)
	options.Apply = true
	f := fake()
	first := run(t, f, options)
	approve(t, options.Root)
	options.Delete = []string{"/posts/example/"}
	f.lostAck = true
	if _, err := Run(context.Background(), f, options); err == nil {
		t.Fatal("expected lost delete acknowledgement")
	}
	manifest, _ := readManifest(filepath.Join(options.Root, ManifestPath))
	if manifest["/posts/example/"] != first.Documents[0].URI {
		t.Fatal("interrupted deletion changed public map before verification")
	}
	f.lostAck = false
	options.Credentials = atproto.Credentials{}
	plan := run(t, f, options)
	if plan.Documents[0].Action != "unchanged" || f.deletes != 1 || f.logins != 2 {
		t.Fatal("delete recovery retried mutation or authentication")
	}
	manifest, _ = readManifest(filepath.Join(options.Root, ManifestPath))
	if len(manifest) != 0 {
		t.Fatal("publicly absent recovered deletion not exported")
	}
}

func TestPlaintextRetainsAdjacentInlinePunctuationAndCodeIndentation(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<div><p>Use <code>WithOverride</code>, then <a href="/">merge</a>.</p><pre><code>if x {
  y()
}</code></pre><p>Last <strong>word</strong>!</p></div>`))
	if err != nil {
		t.Fatal(err)
	}
	want := "Use WithOverride, then merge.\nif x {\n  y()\n}\nLast word!"
	if got := plaintext(doc); got != want {
		t.Fatalf("plaintext altered inline boundaries or code: %q", got)
	}
}

func TestPublicationRecordRequiredOnlyForRealApply(t *testing.T) {
	options := fixture(t)
	if err := os.Remove(filepath.Join(options.Root, PublicationPath)); err != nil {
		t.Fatal(err)
	}
	plan := run(t, nil, options)
	if plan.Publication != "" || plan.Documents[0].Record["site"] != "https://dario.cat" {
		t.Fatal("offline preview fabricated publication record")
	}
	options.Apply = true
	f := fake()
	if _, err := Run(context.Background(), f, options); err == nil || len(f.events) != 0 {
		t.Fatal("apply attempted network without publication artifact")
	}
}
