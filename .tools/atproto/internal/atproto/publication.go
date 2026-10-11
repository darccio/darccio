package atproto

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func DesiredPublication() map[string]any {
	return map[string]any{
		"$type":       PublicationCollection,
		"url":         "https://dario.cat",
		"name":        "Dario Castañé",
		"description": "Tecnologia, política i divulgació.",
		"preferences": map[string]any{"showInDiscover": true},
	}
}

func ReadDesired(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read publication JSON: %w", err)
	}
	if len(data) > maxBody {
		return nil, errors.New("publication JSON exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var desired map[string]any
	if decoder.Decode(&desired) != nil {
		return nil, errors.New("invalid publication JSON")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("invalid trailing publication JSON")
	}
	if !reflect.DeepEqual(desired, DesiredPublication()) {
		return nil, errors.New("publication JSON must match the pinned dario.cat metadata")
	}
	return desired, nil
}

type Options struct {
	Apply                          bool
	Root, StatePath, WellKnownPath string
	Desired                        map[string]any
	Credentials                    Credentials
}

type Result struct {
	Action   string
	URI, CID string
}

// Bootstrap performs no authentication or filesystem writes unless Apply is true.
func (c *Client) Bootstrap(ctx context.Context, options Options) (Result, error) {
	desired := options.Desired
	if desired == nil {
		desired = DesiredPublication()
	}
	if !reflect.DeepEqual(desired, DesiredPublication()) {
		return Result{}, errors.New("desired publication must match the pinned dario.cat metadata")
	}
	if err := c.VerifyIdentity(ctx); err != nil {
		return Result{}, err
	}
	records, err := c.ListRecords(ctx, PublicationCollection)
	if err != nil {
		return Result{}, err
	}
	var matches []Record
	for _, record := range records {
		if canonicalRoot(record.Value["url"]) {
			matches = append(matches, record)
		}
	}
	if len(matches) > 1 {
		return Result{}, errors.New("multiple dario.cat publications found; resolve ambiguity manually")
	}
	result := Result{Action: "create"}
	var existing Record
	if len(matches) == 1 {
		existing, err = c.GetRecord(ctx, matches[0].URI, PublicationCollection)
		if err != nil {
			return Result{}, err
		}
		if !canonicalRoot(existing.Value["url"]) {
			return Result{}, errors.New("publication URL changed during discovery")
		}
		merged, err := mergePublication(existing.Value, desired)
		if err != nil {
			return Result{}, err
		}
		result.URI, result.CID = existing.URI, existing.CID
		result.Action = "update"
		if reflect.DeepEqual(existing.Value, merged) {
			result.Action = "reuse"
		}
		existing.Value = merged
	}
	if !options.Apply {
		return result, nil
	}
	if result.Action != "reuse" {
		var written Record
		err := c.WithSession(ctx, options.Credentials, func(session *Session) error {
			var writeErr error
			if result.Action == "create" {
				written, writeErr = session.CreateRecord(ctx, PublicationCollection, desired)
			} else {
				written, writeErr = session.PutRecord(ctx, PublicationCollection, existing)
			}
			return writeErr
		})
		if err != nil {
			return Result{}, err
		}
		readback, err := c.GetRecord(ctx, written.URI, PublicationCollection)
		if err != nil {
			return Result{}, err
		}
		expected := desired
		if result.Action == "update" {
			expected = existing.Value
		}
		if readback.CID != written.CID || !reflect.DeepEqual(readback.Value, expected) {
			return Result{}, errors.New("public record readback does not match the acknowledged write")
		}
		result.URI, result.CID = readback.URI, readback.CID
	}
	if err := writeArtifacts(options, result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func canonicalRoot(value any) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	u, err := url.Parse(text)
	return err == nil && strings.EqualFold(u.Scheme, "https") && strings.EqualFold(u.Hostname(), Handle) && (u.Port() == "" || u.Port() == "443") && u.User == nil && (u.Path == "" || u.Path == "/") && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && !strings.Contains(text, "#")
}

func mergePublication(existing, desired map[string]any) (map[string]any, error) {
	if existing["$type"] != PublicationCollection {
		return nil, errors.New("publication record has an unexpected type")
	}
	merged := make(map[string]any, len(existing))
	for key, value := range existing {
		merged[key] = value
	}
	preferences := map[string]any{}
	if value, present := existing["preferences"]; present {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, errors.New("publication preferences are not an object")
		}
		for key, value := range object {
			preferences[key] = value
		}
	}
	for key, value := range desired {
		if key != "preferences" {
			merged[key] = value
		}
	}
	for key, value := range desired["preferences"].(map[string]any) {
		preferences[key] = value
	}
	merged["preferences"] = preferences
	return merged, nil
}

// SitePath resolves configurable artifact and input paths against the site root.
func SitePath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

func writeArtifacts(options Options, result Result) error {
	if _, err := RecordKey(result.URI, PublicationCollection); err != nil {
		return err
	}
	if !ValidRecordCID(result.CID) {
		return errors.New("cannot save an unverified publication CID")
	}
	statePath := options.StatePath
	if statePath == "" {
		statePath = ".zas/atproto-publication.json"
	}
	wellKnownPath := options.WellKnownPath
	if wellKnownPath == "" {
		wellKnownPath = ".well-known/site.standard.publication"
	}
	statePath = SitePath(options.Root, statePath)
	wellKnownPath = SitePath(options.Root, wellKnownPath)
	if statePath == wellKnownPath {
		return errors.New("state and verification paths must differ")
	}
	state := struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
		DID string `json:"did"`
		PDS string `json:"pds"`
		URL string `json:"url"`
	}{result.URI, result.CID, DID, PDS, "https://dario.cat"}
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.New("cannot encode publication state")
	}
	stateTemp, err := prepareFile(statePath, append(stateJSON, '\n'))
	if err != nil {
		return fmt.Errorf("prepare publication state: %w", err)
	}
	defer os.Remove(stateTemp)
	verificationTemp, err := prepareFile(wellKnownPath, []byte(result.URI))
	if err != nil {
		return fmt.Errorf("prepare publication verification: %w", err)
	}
	defer os.Remove(verificationTemp)
	if err := os.Rename(stateTemp, statePath); err != nil {
		return fmt.Errorf("save publication state: %w", err)
	}
	if err := os.Rename(verificationTemp, wellKnownPath); err != nil {
		return fmt.Errorf("save publication verification: %w", err)
	}
	return nil
}

func prepareFile(path string, data []byte) (name string, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".publication-*")
	if err != nil {
		return "", err
	}
	name = file.Name()
	defer func() {
		if err != nil {
			file.Close()
			os.Remove(name)
		}
	}()
	if err = file.Chmod(0644); err != nil {
		return name, err
	}
	if _, err = file.Write(data); err != nil {
		return name, err
	}
	if err = file.Sync(); err != nil {
		return name, err
	}
	err = file.Close()
	return name, err
}
