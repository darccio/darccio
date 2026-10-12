# T30 publication proposal

[backfill-preview.json](backfill-preview.json) is the offline full-record preview
for the 26 path/timestamp entries proposed in
[documents-allowlist.json](../documents-allowlist.json). Each record includes its
title, description, publication/update dates, language tag and full rendered
plaintext. The preview has 26 unique planned keys and 72,442 UTF-8 text bytes.
Its AT URIs are planned identities, not publicly verified backlinks.

**Merging the approval PR authorizes publication of all 26 records.** The existing
Standard.site workflow applies the merged list after a successful current-main
production deployment. Request any exclusions before merging. Full text enters
a public AT Protocol repository and can be copied outside robots.txt controls.
No records were written to prepare this preview.

The Spanish liquid-organizations record uses the English timestamp at the
owner's explicit instruction. Its clock time is an adopted assumption, with
the primary-source day and provenance preserved in the article metadata and
[publication provenance](../../../docs/publication-provenance.md).

Regenerate the preview from the repository root after the guarded site build:

```sh
go -C .tools/atproto run -mod=readonly ./cmd/documents -root ../.. \
  -plan .tools/atproto/review/backfill-preview.json > /dev/null
```

This command is offline and does not publish or write operational state. The
review files are excluded from site deployment. After approved publication,
the workflow opens a separate PR containing only publicly verified document
URIs; merge it to deploy the article verification links. T30's remaining checks
are record count/readback, three reader samples and a zero-write online rerun.

Successful apply runs retain `standard-site-document-apply-plan` for seven days.
Inspect its action counts to verify actual creates/updates/deletes/unchanged
results; the planned preview or an overall green workflow is insufficient.
The artifact contains only the public record plan, with no session credentials
or private recovery state. The repeated unchanged run must report all 26
documents as `unchanged`, with zero create/update/delete actions.
