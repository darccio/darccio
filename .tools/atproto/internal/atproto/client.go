// Package atproto contains the small, pinned-account XRPC client used by site tools.
package atproto

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	Handle                = "dario.cat"
	DID                   = "did:plc:aj77r5uwt72o6oimdjfplqoz"
	PDS                   = "https://eurosky.social"
	PublicationCollection = "site.standard.publication"
	maxBody               = 4 << 20
)

type Client struct{ http *http.Client }

func NewClient() *Client {
	return &Client{http: &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") },
	}}
}

type Record struct {
	URI   string         `json:"uri"`
	CID   string         `json:"cid"`
	Value map[string]any `json:"value"`
}

// request never includes response bodies, credentials, or transport errors in errors.
func (c *Client) request(ctx context.Context, method, endpoint, token string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return errors.New("cannot encode XRPC request")
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return errors.New("cannot construct XRPC request")
	}
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return errors.New("XRPC request failed")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("XRPC returned HTTP %d", res.StatusCode)
	}
	if output == nil {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	isDIDDocument := req.URL.Host == "plc.directory" && mediaType == "application/did+ld+json"
	if err != nil || (mediaType != "application/json" && !isDIDDocument) {
		return errors.New("XRPC response is not application/json")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return errors.New("cannot read XRPC response")
	}
	if len(data) > maxBody {
		return errors.New("XRPC response exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(output); err != nil {
		return errors.New("invalid XRPC JSON response")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("invalid trailing XRPC response")
	}
	return nil
}

func endpoint(method string, query url.Values) string {
	u := PDS + "/xrpc/" + method
	if len(query) != 0 {
		u += "?" + query.Encode()
	}
	return u
}

// VerifyIdentity checks both handle resolution and the DID document before PDS use.
func (c *Client) VerifyIdentity(ctx context.Context) error {
	var resolved struct {
		DID string `json:"did"`
	}
	u := "https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?handle=" + Handle
	if err := c.request(ctx, http.MethodGet, u, "", nil, &resolved); err != nil {
		return fmt.Errorf("resolve handle: %w", err)
	}
	if resolved.DID != DID {
		return errors.New("handle resolves to an unexpected DID")
	}
	var document struct {
		ID          string   `json:"id"`
		AlsoKnownAs []string `json:"alsoKnownAs"`
		Service     []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Endpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	}
	if err := c.request(ctx, http.MethodGet, "https://plc.directory/"+DID, "", nil, &document); err != nil {
		return fmt.Errorf("resolve DID: %w", err)
	}
	if document.ID != DID {
		return errors.New("DID document identity does not match")
	}
	alias := false
	for _, value := range document.AlsoKnownAs {
		if value == "at://"+Handle {
			alias = true
		}
	}
	if !alias {
		return errors.New("DID document does not confirm the handle")
	}
	services := 0
	for _, service := range document.Service {
		if service.ID != "#atproto_pds" && service.ID != DID+"#atproto_pds" {
			continue
		}
		services++
		if service.Type != "AtprotoPersonalDataServer" || strings.TrimSuffix(service.Endpoint, "/") != PDS {
			return errors.New("DID document declares an unexpected PDS")
		}
	}
	if services != 1 {
		return errors.New("DID document must declare exactly one PDS")
	}
	return nil
}

func (c *Client) ListRecords(ctx context.Context, collection string) ([]Record, error) {
	var records []Record
	cursor := ""
	seenCursors := map[string]bool{}
	seenURIs := map[string]bool{}
	for page := 0; page < 1000; page++ {
		query := url.Values{"repo": {DID}, "collection": {collection}, "limit": {"100"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		var response struct {
			Records []Record `json:"records"`
			Cursor  string   `json:"cursor"`
		}
		if err := c.request(ctx, http.MethodGet, endpoint("com.atproto.repo.listRecords", query), "", nil, &response); err != nil {
			return nil, fmt.Errorf("list records: %w", err)
		}
		if response.Records == nil {
			return nil, errors.New("list records response omits records")
		}
		for _, record := range response.Records {
			if _, err := RecordKey(record.URI, collection); err != nil {
				return nil, err
			}
			if !ValidRecordCID(record.CID) || record.Value == nil {
				return nil, errors.New("list records response contains an invalid record")
			}
			if seenURIs[record.URI] {
				return nil, errors.New("list records repeats a record")
			}
			seenURIs[record.URI] = true
			records = append(records, record)
		}
		if response.Cursor == "" {
			return records, nil
		}
		if seenCursors[response.Cursor] {
			return nil, errors.New("list records repeats a cursor")
		}
		seenCursors[response.Cursor] = true
		cursor = response.Cursor
	}
	return nil, errors.New("list records exceeds pagination limit")
}

func (c *Client) GetRecord(ctx context.Context, uri, collection string) (Record, error) {
	rkey, err := RecordKey(uri, collection)
	if err != nil {
		return Record{}, err
	}
	query := url.Values{"repo": {DID}, "collection": {collection}, "rkey": {rkey}}
	var record Record
	if err := c.request(ctx, http.MethodGet, endpoint("com.atproto.repo.getRecord", query), "", nil, &record); err != nil {
		return Record{}, fmt.Errorf("read record: %w", err)
	}
	if record.URI != uri || !ValidRecordCID(record.CID) || record.Value == nil {
		return Record{}, errors.New("record readback identity or CID is invalid")
	}
	return record, nil
}

// RecordKey enforces the pinned DID, collection, and Standard.site TID record key.
func RecordKey(uri, collection string) (string, error) {
	prefix := "at://" + DID + "/" + collection + "/"
	if !strings.HasPrefix(uri, prefix) {
		return "", errors.New("record URI has an unexpected DID or collection")
	}
	key := strings.TrimPrefix(uri, prefix)
	const alphabet = "234567abcdefghijklmnopqrstuvwxyz"
	if len(key) != 13 {
		return "", errors.New("record URI does not contain a TID")
	}
	for i, char := range key {
		index := strings.IndexRune(alphabet, char)
		if index < 0 || (i == 0 && index >= 16) {
			return "", errors.New("record URI does not contain a TID")
		}
	}
	return key, nil
}

// AT repository records use CIDv1, DAG-CBOR, SHA-256, in canonical base32.
func ValidRecordCID(cid string) bool {
	if len(cid) != 59 || !strings.HasPrefix(cid, "b") {
		return false
	}
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	raw, err := encoding.DecodeString(strings.ToUpper(cid[1:]))
	return err == nil && len(raw) == 36 && bytes.Equal(raw[:4], []byte{1, 0x71, 0x12, 0x20}) && "b"+strings.ToLower(encoding.EncodeToString(raw)) == cid
}

type Credentials struct{ Handle, AppPassword string }

type Session struct {
	client          *Client
	access, refresh string
}

// WithSession always attempts refresh-token logout, even after a failed mutation.
// Cleanup completes before the caller may publish local verification artifacts.
func (c *Client) WithSession(ctx context.Context, credentials Credentials, action func(*Session) error) (err error) {
	if credentials.Handle != Handle || credentials.AppPassword == "" {
		return errors.New("apply requires ATP_HANDLE=dario.cat and ATP_APP_PASSWORD")
	}
	var response struct {
		DID     string `json:"did"`
		Handle  string `json:"handle"`
		Access  string `json:"accessJwt"`
		Refresh string `json:"refreshJwt"`
	}
	var session *Session
	defer func() {
		// A parsed refresh token can still be revoked when the rest of the
		// session response is malformed (for example, trailing JSON).
		if response.Refresh != "" {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if cleanupErr := c.request(cleanupCtx, http.MethodPost, endpoint("com.atproto.server.deleteSession", nil), response.Refresh, nil, nil); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("delete session: %w", cleanupErr))
			}
		}
		response.Access, response.Refresh = "", ""
		if session != nil {
			session.access, session.refresh = "", ""
		}
	}()
	input := map[string]any{"identifier": credentials.Handle, "password": credentials.AppPassword}
	if err := c.request(ctx, http.MethodPost, endpoint("com.atproto.server.createSession", nil), "", input, &response); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if response.DID != DID || response.Handle != Handle || !validJWT(response.Access) || !validJWT(response.Refresh) {
		return errors.New("session identity or tokens are invalid")
	}
	session = &Session{client: c, access: response.Access, refresh: response.Refresh}
	return action(session)
}

func validJWT(token string) bool {
	if len(token) > 16384 {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, char := range part {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
				return false
			}
		}
	}
	return true
}

func (s *Session) CreateRecord(ctx context.Context, collection string, value map[string]any) (Record, error) {
	input := map[string]any{"repo": DID, "collection": collection, "record": value}
	return s.writeRecord(ctx, "com.atproto.repo.createRecord", collection, input)
}

func (s *Session) PutRecord(ctx context.Context, collection string, record Record) (Record, error) {
	rkey, err := RecordKey(record.URI, collection)
	if err != nil {
		return Record{}, err
	}
	if !ValidRecordCID(record.CID) {
		return Record{}, errors.New("cannot update without a valid record CID")
	}
	input := map[string]any{"repo": DID, "collection": collection, "rkey": rkey, "record": record.Value, "swapRecord": record.CID}
	result, err := s.writeRecord(ctx, "com.atproto.repo.putRecord", collection, input)
	if err == nil && result.URI != record.URI {
		return Record{}, errors.New("update returned a different record URI")
	}
	return result, err
}

func (s *Session) writeRecord(ctx context.Context, method, collection string, input map[string]any) (Record, error) {
	var result Record
	if err := s.client.request(ctx, http.MethodPost, endpoint(method, nil), s.access, input, &result); err != nil {
		return Record{}, fmt.Errorf("write record (do not retry blindly; relist on next run): %w", err)
	}
	if _, err := RecordKey(result.URI, collection); err != nil {
		return Record{}, err
	}
	if !ValidRecordCID(result.CID) {
		return Record{}, errors.New("write record returned an invalid CID")
	}
	return result, nil
}
