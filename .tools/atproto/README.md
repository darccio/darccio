# Standard.site tools

The `publication` command prepares one publication for `https://dario.cat`.
It does not publish articles. The reviewed metadata is in [publication.json](publication.json):
`Dario Castañé`, `Tecnologia, política i divulgació.`, and
`preferences.showInDiscover: true`.

The current account is `dario.cat`, DID
`did:plc:aj77r5uwt72o6oimdjfplqoz`, hosted at `https://eurosky.social`.
Every run checks the public handle resolution and reciprocal DID document against
those facts. A changed identity or PDS stops the run and requires review.
The existing Rizoma publication at `https://rizoma.pckt.blog` is a separate
publication and is never selected or updated.

## Preview

From the site root, run:

```sh
go -C .tools/atproto run ./cmd/publication \
  -root ../.. -record .tools/atproto/publication.json
```

The default reads public records, follows every pagination cursor and reports
`create`, `update` or `reuse`. It performs no authentication, remote mutation or
filesystem write. Ambiguous publications fail instead of selecting one.

The [Standard.site publication workflow](../../.github/workflows/standard-publication.yml)
provides the same preview from `main`, with `apply` defaulting to `false`:

```sh
gh workflow run standard-publication.yml --repo darccio/darccio \
  --ref main -f apply=false
```

## Activate after merging this implementation

In the `darccio/darccio` repository's Actions secrets, set `ATP_HANDLE` to
`dario.cat` and `ATP_APP_PASSWORD` to a dedicated **Eurosky app password**.
Use an app password, not the primary account password. These secrets are
separate from the source repositories' `SITE_DISPATCH_TOKEN`.

Review `publication.json`, then manually dispatch the workflow on `main` with
`apply=true`. This publishes only the reviewed publication metadata. Public
AT Protocol records may remain in relay/indexer copies after deletion.

```sh
gh workflow run standard-publication.yml --repo darccio/darccio \
  --ref main -f apply=true
```

The tool lists the account's publications before writing. It reuses the sole
matching publication; new records receive a server-generated TID. Updates
retain unknown record and preference fields, with the existing CID as
`swapRecord` to reject concurrent edits. There is no blind retry of a failed
creation: rerun the workflow to discover an already-created record.

An ephemeral session uses the password from the environment. Tokens stay in
memory and are never logged or written to an authentication file. The tool
attempts session deletion with the refresh token on success, failure and handled
interrupts, before producing local output. If the PDS rejects cleanup, no
verification output is saved; inspect the failure and revoke the dedicated app
password if necessary. A hard process kill or runner loss can prevent cleanup.

Only after public readback confirms the URI, record content and CID does apply
write these public outputs:

- `.zas/atproto-publication.json`: URI, CID, DID, PDS and site URL.
- `.well-known/site.standard.publication`: the exact AT URI, with no newline.

The workflow runs the existing guarded site build, then opens a PR containing
both files in a single commit. It never pushes to `main`. Repeating the handoff
reuses the same proposal; if main already contains both outputs, it does nothing.
Only these two regular files enter the Git tree; unrelated working-tree data is
excluded. Explicitly dispatched Site checks cover the bot branch because
`GITHUB_TOKEN` writes do not trigger ordinary push/PR workflows.

Merge that verification PR to deploy through Cloudflare Pages. The extensionless
endpoint is served as `text/plain; charset=utf-8`. Verify its complete response
body against the committed file, rather than accepting a 200 homepage fallback:

```sh
curl -fsS https://dario.cat/.well-known/site.standard.publication \
  -o /tmp/dario-publication-live.txt
cmp .well-known/site.standard.publication /tmp/dario-publication-live.txt
curl -fsSI https://dario.cat/.well-known/site.standard.publication
```

There is no placeholder verification file in the implementation PR. Site builds
and deploys never contact the PDS or need these secrets. Per-article publishing,
backfill approval and automation belong to T27–T30.

## Validation and rollback

```sh
go -C .tools/atproto test -race ./...
go -C .tools/atproto vet ./...
python3 .tools/atproto/open-publication-pr_test.py
make -f .zas/Makefile test
make -f .zas/Makefile publish
```

Mocked tests cover pagination, zero/one/multiple matches, foreign publications,
preserved unknown fields, CID concurrency, unchanged runs, interrupted writes,
invalid remote responses, cleanup and verification artifacts. GitHub handoff
tests cover atomic output selection, interruption recovery and duplicate PRs.

To disable bootstrap, disable its manual workflow. Removing the verification
file or reverting Git does not delete a public record. Record deletion is a
separate owner action; the tool has no delete-record operation.

The publication schema was checked on 2026-10-11 at CID
`bafyreiapvnarfn3nl2wvg27hsylli5zqvou7a74p7od5mu4detprtejapy`.
The bootstrap intentionally supports the reviewed metadata; changing it requires
updating the pinned validation and JSON together.

Sources: [publication schema](https://standard.site/docs/lexicons/publication/),
[site verification](https://standard.site/docs/verification/),
[XRPC and session authentication](https://atproto.com/specs/xrpc), and the
authoritative [createRecord](https://github.com/bluesky-social/atproto/blob/main/lexicons/com/atproto/repo/createRecord.json),
[putRecord](https://github.com/bluesky-social/atproto/blob/main/lexicons/com/atproto/repo/putRecord.json)
and [deleteSession](https://github.com/bluesky-social/atproto/blob/main/lexicons/com/atproto/server/deleteSession.json)
lexicons.

## Reviewed document publishing

The separate `documents` command plans locally and publishes only the exact
paths and publication instants in [documents-allowlist.json](documents-allowlist.json).
That list starts empty. [DOCUMENTS.md](DOCUMENTS.md) covers offline previews,
candidate review, stable keys, record updates, recovery state and explicit
record deletion. Candidate discovery never approves or publishes an article.

## Independent workflow and rollback

[Standard.site documents](../../.github/workflows/standard-site.yml) runs after
the existing **Vanity after deploy** workflow. It independently checks the
current `main` SHA against the trusted Cloudflare Pages production check and
suite, then checks out that immutable revision. Preview and fork events,
failed deployments, superseded revisions and backlink-only commits cannot
publish. Pages deployment never waits for this workflow.

Manual dispatch defaults to an offline preview:

```sh
gh workflow run standard-site.yml --repo darccio/darccio --ref main -f apply=false
```

Automatic runs and a manual `apply=true` publish only the committed reviewed
allowlist. The password exists only in the apply step. A separate GitHub job
receives only the verified public URI map and proposes a single-file atomic PR;
it receives no PDS credentials or session files. Repeating an unchanged publish
makes zero record writes and creates no state PR. Publisher failures are visible
in its independent job and cannot block the already-completed deployment.

Disable `standard-site.yml` to stop automation without changing deployment.
Removing a post from the allowlist never silently deletes its public record.
For an explicit deletion, preview `-delete /posts/example/`, then use the same
command with `-apply`; review and commit the resulting public map removal.
Deletion cannot retract copies held elsewhere. Reverting Git alone does not
delete any public record.
