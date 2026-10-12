# T30 approved backfill review

[backfill-preview.json](backfill-preview.json) is the offline full-record preview
for the 26 path/timestamp entries approved in
[documents-allowlist.json](../documents-allowlist.json). Each record includes its
title, description, publication/update dates, language tag and full rendered
plaintext. The preview has 26 unique planned keys and 72,442 UTF-8 text bytes.
It preserves the original pre-publication create plan. Its planned IDs now
match the [verified identity snapshot](backfill-verification.json) and the
production [.zas/atproto.json](../../../.zas/atproto.json) map; the preview
alone remains insufficient evidence of a successful write.

The owner approved all 26 full-text records by merging
[PR #24](https://github.com/darccio/darccio/pull/24). The post-deploy workflow
published them, and [PR #25](https://github.com/darccio/darccio/pull/25) deployed
the verified article backlinks. All public readbacks and three viewer samples
pass; an unchanged online apply reported 26 unchanged records and zero writes.
The [delivery receipt](../../../docs/site-plan/tasks/T30.md) records the exact
runs and production checks. Public text can be copied outside robots.txt
controls; the original preview itself was prepared without PDS writes.

The Spanish liquid-organizations record uses the English timestamp at the
owner's explicit instruction. Its clock time is an adopted assumption, with
the primary-source day and provenance preserved in the article metadata and
[publication provenance](../../../docs/publication-provenance.md).

Preserve this approval snapshot. Generate a new offline preview to a temporary
file from the repository root after the guarded site build:

```sh
go -C .tools/atproto run -mod=readonly ./cmd/documents -root ../.. \
  -plan /tmp/dario-standard-documents-preview.json > /dev/null
```

This command is offline and does not publish or write operational state. The
review files are excluded from site deployment. Future approved publications
use the same separate PR for publicly verified document URIs; merging it deploys
the article verification links. T30's live acceptance checks are complete.

Successful apply runs retain `standard-site-document-apply-plan` for seven days.
Inspect its action counts to verify actual creates/updates/deletes/unchanged
results; the planned preview or an overall green workflow is insufficient.
The artifact contains only the public record plan, with no session credentials
or private recovery state. The repeated unchanged run must report all 26
documents as `unchanged`, with zero create/update/delete actions.
