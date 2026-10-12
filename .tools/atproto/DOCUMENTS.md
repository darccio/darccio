# Reviewed Standard.site document publishing

The document tool defaults to an offline dry-run. The versioned
[allowlist](documents-allowlist.json) defines the exact paths and publication
instants selected for publishing. The owner approved the 26-article T30 archive
in [PR #24](https://github.com/darccio/darccio/pull/24); all records are published
and verified, with production backlinks deployed through
[PR #25](https://github.com/darccio/darccio/pull/25). An unchanged online apply
reported 26 unchanged records and zero writes. Building or deploying the site
never requires a PDS connection or credentials.

The existing **Standard.site documents** workflow automatically applies the
reviewed allowlist after a successful current-main production deployment. A
nonempty allowlist merge is therefore a publication action, rather than merely
preparation for a later manual run. Public AT Protocol records can be copied
outside the site's robots.txt policy. Review the exact selection and full-text
preview before merging; run apply only after the owner approves publication.

From the site root:

```sh
go -C .tools/atproto run ./cmd/documents -root ../..
go -C .tools/atproto run ./cmd/documents -root ../.. -candidates
```

Both commands return JSON on stdout and perform no authentication, network
request, operational-state write, or public-manifest write. Candidate rows contain
source paths, titles, summaries, languages, known timestamps and exclusion
reasons. They are a review aid; eligibility does not grant approval. Private/CV
directories are excluded from the source inventory.

An explicit `-plan /tmp/documents-plan.json` writes the same JSON to that file as
well as stdout. Success exits zero; failures exit one with `documents: ...` on
stderr. Credentials and remote response bodies are never included in errors.

## Review a document

Each approved entry specifies its existing canonical trailing-slash path and
reviewed exact publication instant. For example, the shape is:

```json
{
  "version": 1,
  "documents": [
    {
      "path": "/posts/example/",
      "publishedAt": "2018-03-28T23:26:00+02:00"
    }
  ]
}
```

This example is not an approval. Review the actual source and publication
provenance before editing the allowlist. `date_created` alone is not proof of a
publication instant. Missing timezone or day-only precision remains unresolved;
the tool never supplies an invented clock time or offset. Articles before 2013,
drafts, private pages, and pages with `publish:false`, `feed:false` or
`atproto:false` cannot be approved. Delete Facebook #3 retains its existing site
URL and remains excluded.

For nonempty approval, first run the guarded site build. The publisher uses the
same pinned Zas `Date`, `Updated`, `Summary`, `Tags`, `Title`, and `Canonical`
helpers. Absent an explicit update time, it uses the tracked source's last Git
commit time, matching the site's helper behavior. It checks the assembled
article's canonical URL, language, publication date and complete `e-content`
against its source. Templates or embeds requiring a new source adapter fail
explicitly. Stale or truncated HTML is rejected.

Records contain the full rendered plaintext, including paragraph boundaries
and code indentation, with layout navigation and metadata
excluded. Source tags retain their order and gain the appropriate `lang:ca`,
`lang:en` or `lang:es` tag. The tool applies schema byte limits and conservative
rune bounds for grapheme-limited strings, plus a conservative 1 MiB JSON record
limit. A preview URI denotes planned identity, not a published record.

## Apply and recovery

Apply requires an existing publicly verified publication, pinned handle/DID/PDS
identity, and the reviewed allowlist:

```sh
go -C .tools/atproto run ./cmd/documents -root ../.. -apply
```

`ATP_HANDLE=dario.cat` and `ATP_APP_PASSWORD` are read only for apply. One
ephemeral session serves the complete batch; its tokens remain in memory and
refresh-token logout is attempted on success, failure and handled interruption.
An empty allowlist returns an empty plan without any PDS request or login.
An unchanged nonempty batch needs public reads but no session or record writes.

Initial keys encode publication Unix microseconds and the low 10 bits of
FNV-1a-32 of the canonical path, using AT Protocol's sortable TID alphabet.
Local and remote collisions stop before mutation. Existing keys recovered from
private state or the committed public map take precedence, so a date correction
cannot create a second identity. Discovery also rejects duplicate remote
records for one canonical path. Creates send explicit `swapRecord:null`;
updates and deletes send the fetched CID to reject concurrent changes.

Existing unknown fields survive. Original `publishedAt` survives unless an
allowlist entry explicitly sets `correctPublishedAt:true` after its source date
has been corrected. A mismatched remote/source publication instant stops the
run. With private source snapshots, conflicting local/remote known-field edits
stop for source reconciliation. A fresh CI run with only the public URI map
uses the reviewed source as authority for title, description, full text, tags
and update time; no custom record extension is introduced.

Ignored `.tools/atproto/.state/documents.json` contains public record snapshots,
source snapshots and pending write intentions, with owner-only permissions.
An advisory lock excludes concurrent local publishers. Intended stable keys
are saved atomically before mutation; committed writes with lost
acknowledgements are recovered by relisting and public readback on the next run,
without blind retries. If readback or logout fails, the intent remains private
and the public map is unchanged.

Only after session cleanup and exact public record/CID readback does apply
update `.zas/atproto.json`, a flat canonical-path-to-AT-URI map. Its entries
contain only verified public identities. Existing entries outside the current
operation remain unchanged. Keep this map in the reviewed public-state PR;
never commit operational state, passwords or sessions. A failure between private
state and public-map writes is recovered by repeating apply.

## Explicit deletion

Removing an approval or excluding a source never deletes a remote record.
First remove the intended target from the local reviewed allowlist, leaving its
existing public-map entry in place. Both preview and apply reject a `-delete`
target that remains approved, before any PDS request or login. Preview the
explicit existing target, then apply the same action:

```sh
go -C .tools/atproto run ./cmd/documents -root ../.. -delete /posts/example/
go -C .tools/atproto run ./cmd/documents -root ../.. -delete /posts/example/ -apply
```

Targets must have stored or committed public identities. Public absence is
verified before removal from the public map; a private tombstone retains the
stable identity. Commit the allowlist removal and verified public-map removal
together after successful deletion. Deploying the allowlist removal while its
backlink remains would fail the build's approval checks. An ordinary apply
with the removed approval does not recreate the record, even without private
state. Repeated explicit deletion with the tombstone is unchanged. AT Protocol
relays or indexers can retain copies after deletion. Reverting Git or disabling
the workflow changes future publishing but does not reverse existing records.

## Validation and pinned contracts

```sh
go -C .tools/atproto test -race ./...
go -C .tools/atproto vet ./...
```

Fake-client tests cover offline behavior, full text, strict metadata, source
exclusions, once-per-batch sessions, unchanged records, local/remote collisions,
date/key correction, merge conflicts, conditional writes, interrupted apply,
cleanup/readback gating, state locking and explicit deletion. Wire tests assert
that create really sends JSON null rather than omitting `swapRecord`.

The [document Lexicon](https://standard.site/docs/lexicons/document/) was read
from the Standard.site authority's public PDS on 2026-10-11 at CID
`bafyreifylc5lfr4vqfzrwdtqm4pnfg7cbra7z4acbgikylqandzefwdeaq`. Its exact
response is pinned in `internal/atproto/testdata/site.standard.document.json`.
The API fixtures for `getRecord`, `listRecords`, `putRecord` and `deleteRecord`
come from [AT Protocol commit ba08e05](https://github.com/bluesky-social/atproto/tree/ba08e05d86f7cdb71ea11d2b620ac2d7eec6b532/lexicons/com/atproto/repo).
The existing T26 client supplies session and identity handling. See also the
[TID specification](https://atproto.com/specs/tid) and
[site verification protocol](https://standard.site/docs/verification/).

The [T30 delivery receipt](../../docs/site-plan/tasks/T30.md) records the
approved 26-record backfill, exact public retrieval, production verification
links, three reader samples and the unchanged online rerun. Its
[public identity snapshot](review/backfill-verification.json) records the
verified URIs, CIDs and payload hashes as of 2026-10-12. Future selections and
updates require their own review and actual apply/readback checks.

Successful apply runs upload their public plan as
`standard-site-document-apply-plan`, retained for seven days. Use its action
counts and actual public records to verify publication; these soft-failing
jobs can leave the overall workflow green despite a failed apply. An unchanged
online rerun must contain only `unchanged` actions, with zero record writes.
The artifact includes public record values, not session credentials or private
recovery state. The verified-URI artifact remains separate for the state PR.
