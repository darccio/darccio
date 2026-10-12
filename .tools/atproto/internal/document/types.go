// Package document plans and publishes only explicitly reviewed site articles.
// Planning is entirely local. Public outputs are separate from recovery state.
package document

import (
	"context"
	"time"

	"dario.cat/atproto-tools/internal/atproto"
)

const (
	AllowlistPath   = ".tools/atproto/documents-allowlist.json"
	StatePath       = ".tools/atproto/.state/documents.json"
	ManifestPath    = ".zas/atproto.json"
	PublicationPath = ".zas/atproto-publication.json"
)

type Approval struct {
	Path               string `json:"path"`
	PublishedAt        string `json:"publishedAt"`
	CorrectPublishedAt bool   `json:"correctPublishedAt,omitempty"`
}

type Allowlist struct {
	Version   int        `json:"version"`
	Documents []Approval `json:"documents"`
}

type Candidate struct {
	Path        string `json:"path"`
	Source      string `json:"source"`
	PublishedAt string `json:"publishedAt,omitempty"`
	Language    string `json:"language"`
	Title       string `json:"title,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Eligible    bool   `json:"eligible"`
	Reason      string `json:"reason,omitempty"`
}

type Entry struct {
	Path        string         `json:"path"`
	URI         string         `json:"uri"`
	Action      string         `json:"action"`
	Record      map[string]any `json:"record,omitempty"`
	CID         string         `json:"cid,omitempty"`
	source      map[string]any
	correctDate bool
}

type Plan struct {
	Version     int         `json:"version"`
	Publication string      `json:"publication"`
	Documents   []Entry     `json:"documents"`
	Candidates  []Candidate `json:"candidates,omitempty"`
}

type Pending struct {
	Action string         `json:"action"`
	Record map[string]any `json:"record,omitempty"`
	Source map[string]any `json:"source,omitempty"`
}

type Snapshot struct {
	URI     string         `json:"uri"`
	CID     string         `json:"cid,omitempty"`
	Source  map[string]any `json:"source,omitempty"`
	Record  map[string]any `json:"record,omitempty"`
	Pending *Pending       `json:"pending,omitempty"`
	Deleted bool           `json:"deleted,omitempty"`
}

type State struct {
	Version     int                 `json:"version"`
	Publication string              `json:"publication"`
	Documents   map[string]Snapshot `json:"documents"`
}

type Options struct {
	Root, Rendered, Allowlist, State, Manifest string
	Apply, Candidates                          bool
	Delete                                     []string
	Credentials                                atproto.Credentials
}

type Network interface {
	VerifyIdentity(context.Context) error
	ListRecords(context.Context, string) ([]atproto.Record, error)
	GetRecord(context.Context, string, string) (atproto.Record, error)
	WithDocumentSession(context.Context, atproto.Credentials, func(atproto.DocumentWriter) error) error
}

type source struct {
	Candidate
	file, title, summary string
	tags                 []string
	date, updated        time.Time
	config               map[interface{}]interface{}
	body                 []byte
}
