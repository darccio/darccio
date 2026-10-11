package document

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"dario.cat/atproto-tools/internal/atproto"
)

const maxLocalFile = 16 << 20

func readRegular(file string) ([]byte, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	if err := regularParents(string(filepath.Separator), abs); err != nil {
		return nil, err
	}
	info, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("expected regular file: %s", file)
	}
	if info.Size() > maxLocalFile {
		return nil, fmt.Errorf("file exceeds size limit: %s", file)
	}
	return os.ReadFile(file)
}

// Refuse symlink parents: a regular leaf alone does not establish containment.
func regularParents(root, file string) error {
	rel, err := filepath.Rel(root, file)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("path escapes its root")
	}
	current := root
	for _, part := range append([]string{""}, strings.Split(rel, string(filepath.Separator))...) {
		if part != "" {
			current = filepath.Join(current, part)
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path rejected: %s", current)
		}
	}
	return nil
}

func decodeFile(file string, output any) error {
	data, err := readRegular(file)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", file, err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("trailing JSON in %s", file)
	}
	return nil
}

func readAllowlist(file string) (Allowlist, error) {
	var allowlist Allowlist
	if err := decodeFile(file, &allowlist); err != nil {
		return allowlist, err
	}
	if allowlist.Version != 1 || allowlist.Documents == nil {
		return allowlist, errors.New("allowlist must have version 1 and explicit documents array")
	}
	seen := map[string]bool{}
	for _, entry := range allowlist.Documents {
		if !canonicalPath(entry.Path) || seen[entry.Path] {
			return allowlist, errors.New("allowlist contains invalid or duplicate canonical path")
		}
		seen[entry.Path] = true
		if _, err := reviewedDate(entry.PublishedAt); err != nil {
			return allowlist, fmt.Errorf("%s: %w", entry.Path, err)
		}
	}
	return allowlist, nil
}

func readPublication(root string) (string, error) {
	var publication struct{ URI, CID, DID, PDS, URL string }
	err := decodeFile(filepath.Join(root, PublicationPath), &publication)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if _, err := atproto.RecordKey(publication.URI, atproto.PublicationCollection); err != nil {
		return "", err
	}
	if !atproto.ValidRecordCID(publication.CID) || publication.DID != atproto.DID || publication.PDS != atproto.PDS || publication.URL != "https://dario.cat" {
		return "", errors.New("publication artifact does not match the pinned site identity")
	}
	return publication.URI, nil
}

func readState(file, publication string) (State, error) {
	state := State{Version: 1, Publication: publication, Documents: map[string]Snapshot{}}
	if err := decodeFile(file, &state); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return state, err
	}
	if state.Version != 1 || state.Documents == nil || state.Publication != publication {
		return state, errors.New("private state version or publication identity differs")
	}
	seen := map[string]bool{}
	for path, snapshot := range state.Documents {
		if !canonicalPath(path) || seen[snapshot.URI] {
			return state, errors.New("private state contains invalid path or colliding identity")
		}
		seen[snapshot.URI] = true
		if _, err := atproto.RecordKey(snapshot.URI, atproto.DocumentCollection); err != nil {
			return state, err
		}
		if snapshot.CID != "" && !atproto.ValidRecordCID(snapshot.CID) {
			return state, errors.New("private state contains invalid CID")
		}
		if snapshot.Record != nil {
			if err := recordIdentity(snapshot.Record, path, publication); err != nil {
				return state, err
			}
		}
		if snapshot.Pending != nil && snapshot.Pending.Action != "create" && snapshot.Pending.Action != "update" && snapshot.Pending.Action != "delete" {
			return state, errors.New("private state contains unknown pending action")
		}
	}
	return state, nil
}

func readManifest(file string) (map[string]string, error) {
	manifest := map[string]string{}
	if err := decodeFile(file, &manifest); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return manifest, nil
		}
		return nil, err
	}
	if manifest == nil {
		return nil, errors.New("public document manifest must be an object")
	}
	seen := map[string]bool{}
	for path, uri := range manifest {
		if !canonicalPath(path) || seen[uri] {
			return nil, errors.New("public manifest has invalid path or colliding identity")
		}
		seen[uri] = true
		if _, err := atproto.RecordKey(uri, atproto.DocumentCollection); err != nil {
			return nil, err
		}
	}
	return manifest, nil
}

func equalJSON(a, b any) bool {
	x, err := json.Marshal(a)
	if err != nil {
		return false
	}
	y, err := json.Marshal(b)
	return err == nil && bytes.Equal(x, y)
}

func atomicJSON(file string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if previous, err := readRegular(file); err == nil && bytes.Equal(previous, data) {
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := regularParents(string(filepath.Separator), dir); err != nil {
		return err
	}
	output, err := os.CreateTemp(dir, ".documents-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(output.Name())
	if err := output.Chmod(mode); err != nil {
		output.Close()
		return err
	}
	if _, err := output.Write(data); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	if err := os.Rename(output.Name(), file); err != nil {
		return err
	}
	parent, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}

func lockState(file string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return nil, err
	}
	if err := regularParents(string(filepath.Separator), filepath.Dir(file)); err != nil {
		return nil, err
	}
	lockPath := file + ".lock"
	if _, err := os.Lstat(lockPath); err == nil {
		if _, err := readRegular(lockPath); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	fileHandle, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(fileHandle.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		fileHandle.Close()
		return nil, errors.New("another document publisher holds the state lock")
	}
	return func() { syscall.Flock(int(fileHandle.Fd()), syscall.LOCK_UN); fileHandle.Close() }, nil
}

// WritePlan is an explicit optional filesystem output; ordinary dry-run only
// writes stdout and does not create state or authentication artifacts.
func WritePlan(file string, plan Plan) error { return atomicJSON(file, plan, 0644) }
