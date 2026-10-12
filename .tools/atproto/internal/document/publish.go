package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"dario.cat/atproto-tools/internal/atproto"
)

// Run returns a pure local plan unless Apply is explicitly selected. Excluded
// or removed approvals never cause deletion. The network is not needed by
// planning, even when credentials happen to be present in the environment.
func Run(ctx context.Context, network Network, options Options) (Plan, error) {
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return Plan{}, err
	}
	if err := regularParents(string(filepath.Separator), root); err != nil {
		return Plan{}, err
	}
	resolve := func(value, fallback string) string {
		if value == "" {
			value = fallback
		}
		return atproto.SitePath(root, value)
	}
	allowlistPath := resolve(options.Allowlist, AllowlistPath)
	statePath := resolve(options.State, StatePath)
	manifestPath := resolve(options.Manifest, ManifestPath)
	rendered := resolve(options.Rendered, ".zas/deploy")
	if statePath == manifestPath || statePath == allowlistPath || manifestPath == allowlistPath {
		return Plan{}, errors.New("state, manifest and allowlist paths must differ")
	}
	allowlist, err := readAllowlist(allowlistPath)
	if err != nil {
		return Plan{}, err
	}
	if options.Apply && options.Candidates {
		return Plan{}, errors.New("candidate discovery cannot be applied")
	}
	if options.Apply && (len(allowlist.Documents) > 0 || len(options.Delete) > 0) {
		unlock, err := lockState(statePath)
		if err != nil {
			return Plan{}, err
		}
		defer unlock()
	}
	publication, err := readPublication(root)
	if err != nil {
		return Plan{}, err
	}
	state, err := readState(statePath, publication)
	if err != nil {
		return Plan{}, err
	}
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return Plan{}, err
	}
	var sources map[string]source
	if len(allowlist.Documents) > 0 || options.Candidates {
		sources, err = loadSources(root)
		if err != nil {
			return Plan{}, err
		}
	}
	plan, err := makePlan(allowlist, sources, state, manifest, rendered, publication, options.Delete, !options.Apply)
	if err != nil {
		return Plan{}, err
	}
	if options.Candidates {
		plan.Candidates = sortedSources(sources)
	}
	if !options.Apply || len(plan.Documents) == 0 {
		return plan, nil
	}
	if publication == "" {
		return Plan{}, errors.New("apply requires verified publication artifact from publication bootstrap")
	}
	if network == nil {
		return Plan{}, errors.New("apply requires a network client")
	}
	if err := network.VerifyIdentity(ctx); err != nil {
		return Plan{}, err
	}
	publicationRecord, err := network.GetRecord(ctx, publication, atproto.PublicationCollection)
	if err != nil {
		return Plan{}, err
	}
	if publicationRecord.Value["$type"] != atproto.PublicationCollection || publicationRecord.Value["url"] != "https://dario.cat" {
		return Plan{}, errors.New("public publication does not match the site")
	}
	remote, err := network.ListRecords(ctx, atproto.DocumentCollection)
	if err != nil {
		return Plan{}, err
	}
	if err := preflight(ctx, network, &plan, state, manifest, remote, options.Delete); err != nil {
		return Plan{}, err
	}
	if err := apply(ctx, network, options.Credentials, &plan, &state, manifest, statePath, manifestPath); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func preflight(ctx context.Context, network Network, plan *Plan, state State, manifest map[string]string, records []atproto.Record, deletes []string) error {
	byURI := map[string]atproto.Record{}
	byPath := map[string]atproto.Record{}
	for _, record := range records {
		if _, err := atproto.RecordKey(record.URI, atproto.DocumentCollection); err != nil {
			return err
		}
		if !atproto.ValidRecordCID(record.CID) || record.Value == nil {
			return errors.New("remote document response has invalid CID or value")
		}
		if _, duplicate := byURI[record.URI]; duplicate {
			return errors.New("remote documents repeat an identity")
		}
		byURI[record.URI] = record
		if record.Value["site"] != plan.Publication {
			continue
		}
		path, ok := record.Value["path"].(string)
		if !ok || !canonicalPath(path) {
			continue
		}
		if _, duplicate := byPath[path]; duplicate {
			return fmt.Errorf("%s: multiple publication records share this canonical path", path)
		}
		byPath[path] = record
	}
	deleteSet := map[string]bool{}
	for _, path := range deletes {
		deleteSet[path] = true
	}
	reserved := map[string]string{}
	for path, snapshot := range state.Documents {
		reserved[snapshot.URI] = path
	}
	for path, uri := range manifest {
		reserved[uri] = path
	}
	for i := range plan.Documents {
		entry := &plan.Documents[i]
		snapshot := state.Documents[entry.Path]
		_, present := byURI[entry.URI]
		if match, found := byPath[entry.Path]; found && match.URI != entry.URI {
			if snapshot.URI != "" || manifest[entry.Path] != "" {
				return fmt.Errorf("%s: remote identity differs from stable stored key", entry.Path)
			}
			entry.URI = match.URI // Recovery with no private state, even after a date correction.
			present = true
		}
		if owner, used := reserved[entry.URI]; used && owner != entry.Path {
			return errors.New("remote adoption collides with another stored document")
		}
		reserved[entry.URI] = entry.Path
		if !present {
			entry.CID = ""
			if deleteSet[entry.Path] {
				entry.Action = "unchanged"
				entry.Record = nil
				continue
			}
			if snapshot.CID != "" && !snapshot.Deleted {
				return fmt.Errorf("%s: previously verified remote record is missing; reconcile explicitly", entry.Path)
			}
			entry.Action = "create"
			entry.Record = entry.source
			continue
		}
		current, err := network.GetRecord(ctx, entry.URI, atproto.DocumentCollection)
		if err != nil {
			return err
		}
		if current.URI != entry.URI || !atproto.ValidRecordCID(current.CID) {
			return errors.New("public document readback has invalid identity or CID")
		}
		if err := recordIdentity(current.Value, entry.Path, plan.Publication); err != nil {
			return err
		}
		entry.CID = current.CID
		if deleteSet[entry.Path] {
			entry.Action = "delete"
			entry.Record = nil
			continue
		}
		if err := publicationDateAgrees(current.Value, entry.source, entry.correctDate); err != nil {
			return err
		}
		previousSource := snapshot.Source
		if pending := snapshot.Pending; pending != nil {
			if pending.Action == "delete" {
				return fmt.Errorf("%s: interrupted explicit deletion requires the delete action again", entry.Path)
			}
			if equalJSON(current.Value, pending.Record) {
				previousSource = pending.Source
			} else if !equalJSON(current.Value, snapshot.Record) {
				return fmt.Errorf("%s: interrupted write differs from both verified and intended record; reconcile explicitly", entry.Path)
			}
		}
		merged, err := mergeRecord(current.Value, entry.source, previousSource, entry.correctDate)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Path, err)
		}
		data, err := json.Marshal(merged)
		if err != nil || len(data) > 1<<20 {
			return errors.New("merged document exceeds conservative record size limit")
		}
		entry.Record = merged
		entry.Action = "update"
		if equalJSON(current.Value, merged) {
			entry.Action = "unchanged"
		}
	}
	return nil
}

func apply(ctx context.Context, network Network, credentials atproto.Credentials, plan *Plan, state *State, manifest map[string]string, statePath, manifestPath string) error {
	mutations := false
	for _, entry := range plan.Documents {
		if entry.Action == "unchanged" {
			continue
		}
		mutations = true
		snapshot := state.Documents[entry.Path]
		snapshot.URI = entry.URI
		snapshot.Pending = &Pending{Action: entry.Action, Record: entry.Record, Source: entry.source}
		state.Documents[entry.Path] = snapshot
	}
	// Save stable identities and intended writes before the first mutation so
	// a lost acknowledgement cannot lead to a second key on the next run.
	if mutations {
		if err := atomicJSON(statePath, state, 0600); err != nil {
			return err
		}
		err := network.WithDocumentSession(ctx, credentials, func(writer atproto.DocumentWriter) error {
			for i := range plan.Documents {
				entry := &plan.Documents[i]
				switch entry.Action {
				case "create", "update":
					written, err := writer.PutDocument(ctx, atproto.Record{URI: entry.URI, CID: entry.CID, Value: entry.Record})
					if err != nil {
						return err
					}
					if written.URI != entry.URI || !atproto.ValidRecordCID(written.CID) {
						return errors.New("document write acknowledgement identity or CID differs")
					}
					entry.CID = written.CID
				case "delete":
					if err := writer.DeleteDocument(ctx, atproto.Record{URI: entry.URI, CID: entry.CID}); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		} // Ephemeral-session cleanup also gates export.
	}
	// Cleanup has completed. Only publicly verified targets may enter Git's
	// manifest. Tombstones remain private to reserve identities after deletion.
	var absent map[string]bool
	for _, entry := range plan.Documents {
		if entry.Record != nil {
			continue
		}
		if absent == nil {
			records, err := network.ListRecords(ctx, atproto.DocumentCollection)
			if err != nil {
				return err
			}
			absent = map[string]bool{}
			for _, record := range records {
				absent[record.URI] = true
			}
		}
		if absent[entry.URI] {
			return errors.New("explicitly deleted document remains publicly readable")
		}
	}
	for i := range plan.Documents {
		entry := &plan.Documents[i]
		if entry.Record == nil {
			state.Documents[entry.Path] = Snapshot{URI: entry.URI, Deleted: true}
			delete(manifest, entry.Path)
			entry.CID = ""
			continue
		}
		readback, err := network.GetRecord(ctx, entry.URI, atproto.DocumentCollection)
		if err != nil {
			return err
		}
		if readback.URI != entry.URI || !atproto.ValidRecordCID(readback.CID) || readback.CID != entry.CID || !equalJSON(readback.Value, entry.Record) {
			return errors.New("public document readback does not match verified plan/write CID")
		}
		state.Documents[entry.Path] = Snapshot{URI: entry.URI, CID: readback.CID, Source: entry.source, Record: readback.Value}
		manifest[entry.Path] = entry.URI
	}
	if err := atomicJSON(statePath, state, 0600); err != nil {
		return err
	}
	return atomicJSON(manifestPath, manifest, 0644)
}
