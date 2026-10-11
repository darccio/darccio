package document

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"dario.cat/atproto-tools/internal/atproto"
	"github.com/darccio/zas"
)

func reviewedDate(value string) (time.Time, error) {
	data := zas.ZasData{Page: map[interface{}]interface{}{"date": value}}
	date, err := data.Date()
	if err != nil || date.IsZero() {
		return time.Time{}, errors.New("reviewed publication date requires exact RFC3339 timestamp with timezone")
	}
	return date, nil
}

// StableTID derives the initial identity from Unix microseconds and FNV-1a-32
// canonical-path low 10 bits. Callers must check collisions; this is not unique.
func StableTID(date time.Time, canonical string) (string, error) {
	if !canonicalPath(canonical) {
		return "", errors.New("TID requires canonical article path")
	}
	micros := date.UnixMicro()
	if micros < 0 || uint64(micros) > ((uint64(1)<<53)-1) {
		return "", errors.New("publication timestamp is outside TID range")
	}
	hash := fnv.New32a()
	hash.Write([]byte(canonical))
	value := uint64(micros)<<10 | uint64(hash.Sum32()&1023)
	const alphabet = "234567abcdefghijklmnopqrstuvwxyz"
	var key [13]byte
	for i := len(key) - 1; i >= 0; i-- {
		key[i] = alphabet[value&31]
		value >>= 5
	}
	return string(key[:]), nil
}

func recordIdentity(record map[string]any, path, publication string) error {
	if record["$type"] != atproto.DocumentCollection || record["site"] != publication || record["path"] != path {
		return errors.New("document record identity collision; refusing overwrite")
	}
	date, ok := record["publishedAt"].(string)
	if !ok {
		return errors.New("existing document publication date is invalid")
	}
	_, err := reviewedDate(date)
	return err
}

// Three-way merging preserves remote corrections when the source did not
// change, and rejects conflicting edits. Unknown properties always survive.
// Existing publishedAt changes only through an explicit reviewed correction.
func mergeRecord(existing, desired, previousSource map[string]any, correctDate bool) (map[string]any, error) {
	if existing == nil {
		return desired, nil
	}
	merged := make(map[string]any, len(existing)+len(desired))
	for key, value := range existing {
		merged[key] = value
	}
	for _, key := range []string{"$type", "site", "path", "title", "description", "textContent", "tags", "updatedAt"} {
		want, wantPresent := desired[key]
		current, currentPresent := existing[key]
		old, oldPresent := previousSource[key]
		if previousSource != nil && equalJSON(want, old) && wantPresent == oldPresent {
			if !equalJSON(current, want) || currentPresent != wantPresent {
				return nil, fmt.Errorf("remote %s correction must be reconciled in source before applying", key)
			}
			continue
		}
		if previousSource != nil && (!equalJSON(current, old) || currentPresent != oldPresent) && (!equalJSON(current, want) || currentPresent != wantPresent) {
			return nil, fmt.Errorf("conflicting local and remote %s correction; reconcile source before applying", key)
		}
		if wantPresent {
			merged[key] = want
		} else {
			delete(merged, key)
		}
	}
	if correctDate {
		merged["publishedAt"] = desired["publishedAt"]
	}
	return merged, nil
}

func publicationDateAgrees(existing, desired map[string]any, correctDate bool) error {
	if existing == nil || correctDate {
		return nil
	}
	currentDate, err := reviewedDate(existing["publishedAt"].(string))
	if err != nil {
		return err
	}
	desiredDate, err := reviewedDate(desired["publishedAt"].(string))
	if err != nil {
		return err
	}
	if !correctDate && !currentDate.Equal(desiredDate) {
		return errors.New("existing publishedAt differs from source; preserve the remote correction in source or explicitly review correctPublishedAt")
	}
	return nil
}

func makePlan(allowlist Allowlist, sources map[string]source, state State, manifest map[string]string, rendered, publication string, deletes []string, cached bool) (Plan, error) {
	plan := Plan{Version: 1, Publication: publication, Documents: []Entry{}}
	// A publication record is necessary only for apply. Preview can use the
	// canonical site URL allowed by the document schema before bootstrap.
	site := publication
	if site == "" {
		site = "https://dario.cat"
	}
	reserved := map[string]string{}
	for path, snapshot := range state.Documents {
		reserved[snapshot.URI] = path
	}
	for path, uri := range manifest {
		if other, used := reserved[uri]; used && other != path {
			return plan, errors.New("state and public manifest identities collide")
		}
		if snapshot, exists := state.Documents[path]; exists && snapshot.URI != uri {
			return plan, errors.New("state and public manifest disagree on stable identity")
		}
		reserved[uri] = path
	}
	deleteSet := map[string]bool{}
	for _, path := range deletes {
		if !canonicalPath(path) || deleteSet[path] {
			return plan, errors.New("delete requires unique explicit canonical article paths")
		}
		deleteSet[path] = true
	}
	for _, approval := range allowlist.Documents {
		if deleteSet[approval.Path] {
			return plan, fmt.Errorf("%s: remove the path from the reviewed allowlist before explicit deletion", approval.Path)
		}
	}
	for _, approval := range allowlist.Documents {
		s, exists := sources[approval.Path]
		if !exists || !s.Eligible {
			return plan, fmt.Errorf("%s: approved source is missing or excluded", approval.Path)
		}
		date, _ := reviewedDate(approval.PublishedAt)
		if !date.Equal(s.date) {
			return plan, fmt.Errorf("%s: reviewed date differs from source", approval.Path)
		}
		desired, err := desiredRecord(s, rendered, site)
		if err != nil {
			return plan, err
		}
		snapshot, stored := state.Documents[approval.Path]
		uri := snapshot.URI
		if uri == "" {
			uri = manifest[approval.Path]
		}
		if uri == "" {
			key, err := StableTID(date, approval.Path)
			if err != nil {
				return plan, err
			}
			uri = "at://" + atproto.DID + "/" + atproto.DocumentCollection + "/" + key
		}
		if other, used := reserved[uri]; used && other != approval.Path {
			return plan, fmt.Errorf("TID collision between %s and %s", approval.Path, other)
		}
		reserved[uri] = approval.Path
		merged := desired
		if cached {
			if err := publicationDateAgrees(snapshot.Record, desired, approval.CorrectPublishedAt); err != nil {
				return plan, err
			}
			merged, err = mergeRecord(snapshot.Record, desired, snapshot.Source, approval.CorrectPublishedAt)
			if err != nil {
				return plan, err
			}
		}
		action := "create"
		if stored && !snapshot.Deleted && snapshot.Record != nil {
			action = "update"
			if equalJSON(snapshot.Record, merged) {
				action = "unchanged"
			}
		}
		plan.Documents = append(plan.Documents, Entry{Path: approval.Path, URI: uri, Action: action, Record: merged, CID: snapshot.CID, source: desired, correctDate: approval.CorrectPublishedAt})
	}
	for path := range deleteSet {
		snapshot, stored := state.Documents[path]
		uri := snapshot.URI
		if uri == "" {
			uri = manifest[path]
		}
		if uri == "" {
			return plan, fmt.Errorf("%s: deletion requires an existing stored or public identity", path)
		}
		action := "delete"
		if stored && snapshot.Deleted {
			action = "unchanged"
		}
		plan.Documents = append(plan.Documents, Entry{Path: path, URI: uri, Action: action, CID: snapshot.CID})
	}
	sort.Slice(plan.Documents, func(i, j int) bool { return plan.Documents[i].Path < plan.Documents[j].Path })
	return plan, nil
}
