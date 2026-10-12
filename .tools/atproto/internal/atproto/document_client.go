package atproto

import (
	"context"
	"errors"
	"net/http"
)

const DocumentCollection = "site.standard.document"

// DocumentWriter uses explicit keys and compare-and-swap for every mutation.
type DocumentWriter interface {
	PutDocument(context.Context, Record) (Record, error)
	DeleteDocument(context.Context, Record) error
}

// WithDocumentSession reuses one ephemeral session for the complete batch.
func (c *Client) WithDocumentSession(ctx context.Context, credentials Credentials, action func(DocumentWriter) error) error {
	return c.WithSession(ctx, credentials, func(session *Session) error { return action(session) })
}

// PutDocument creates only when absent (explicit JSON null swapRecord), or
// updates only the supplied CID. Omission would allow overwriting a collision.
func (s *Session) PutDocument(ctx context.Context, record Record) (Record, error) {
	rkey, err := RecordKey(record.URI, DocumentCollection)
	if err != nil {
		return Record{}, err
	}
	var previous any
	if record.CID != "" {
		if !ValidRecordCID(record.CID) {
			return Record{}, errors.New("document update requires a valid CID")
		}
		previous = record.CID
	}
	input := map[string]any{"repo": DID, "collection": DocumentCollection, "rkey": rkey, "record": record.Value, "swapRecord": previous}
	result, err := s.writeRecord(ctx, "com.atproto.repo.putRecord", DocumentCollection, input)
	if err == nil && result.URI != record.URI {
		return Record{}, errors.New("document write returned a different URI")
	}
	return result, err
}

func (s *Session) DeleteDocument(ctx context.Context, record Record) error {
	rkey, err := RecordKey(record.URI, DocumentCollection)
	if err != nil {
		return err
	}
	if !ValidRecordCID(record.CID) {
		return errors.New("document deletion requires a valid CID")
	}
	input := map[string]any{"repo": DID, "collection": DocumentCollection, "rkey": rkey, "swapRecord": record.CID}
	var response map[string]any
	if err := s.client.request(ctx, http.MethodPost, endpoint("com.atproto.repo.deleteRecord", nil), s.access, input, &response); err != nil {
		return err
	}
	if response == nil {
		return errors.New("document deletion response is not an object")
	}
	return nil
}
