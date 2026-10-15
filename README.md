# Bon dia, I'm Dario! 👋

Catalan, pirate, gopher, bofher, trollslayer, tsundokusha, speaker, senior open source library software engineer - Go @DataDog 

## I'm known for...

- [Mergo](https://github.com/darccio/mergo): Simple structs & maps merging for Go.
- [Asembleo](https://github.com/coopanio/asembleo): Pseudoanonymous voting system for general assemblies.
- [Talks](https://github.com/darccio/talks): my slide decks.

Please consider sponsoring my work:

[![GitHub Sponsor](https://img.shields.io/github/sponsors/darccio?label=Sponsor&logo=GitHub)](https://github.com/sponsors/darccio)

## Myself around the net

- [Web](https://dario.cat)
- [Fediverse](https://mastodont.cat/@dario)
- [BlueSky](https://bsky.app/profile/dario.cat)
- [LinkedIn](https://linkedin.com/in/darccio)
- [Sifa.id](https://sifa.id/p/dario.cat)
- [CV](https://dario.cat/cv/): HTML generated from the MAC JSON below.
- [Manfred Awesomic CV](/CV/MAC.json): an open-source, interchangeable, machine-readable format. It's also synchronized with [my profile in Manfred](https://getmanfred.com/profile/dcc).


## ⚖️ License

The CVs included in this repository contain my personal and professional data, and they can't be automatically processed without my explicit permission according to the [GDPR](https://gdpr-info.eu/) (General Data Protection Regulation).

On 12 October 2026 I authorized updating `CV/MAC.json` from my Sifa profile and
supplied LinkedIn export, and generating and validating `/cv/` from that JSON
during site builds. This specific permission leaves the restriction above in
place for other uses. Both CV URLs remain excluded from crawling and from the
site's feeds, Standard.site publishing and agent text collections. See
[CV maintenance](https://github.com/darccio/darccio/blob/main/CV/README.md) for the source and date conventions.

The [MAC format](https://github.com/getmanfred/mac) is free and open-source software licensed and distributed under the [Creative Commons Attribution Share Alike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/) (CC BY-SA 4.0 International).

Original site content for which Dario Castañé holds the necessary rights is licensed under [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/) (CC BY-SA 4.0), excluding the CV personal data above and separately licensed material. See [LICENSE](LICENSE) for the scope and [the CC BY-SA 4.0 legal code](https://creativecommons.org/licenses/by-sa/4.0/legalcode.en) for the terms.

[Lýd](lyd/) retains its separately stated [CC BY 4.0 licence](lyd/README.md). Third-party images, quotations, code and dependencies retain their own licences and notices; this content statement grants no rights that Dario does not hold.

Original software and site implementation code outside separately licensed paths, including templates, stylesheets, build configuration, the `.tools/` checks and `.workers/` worker, retain the original [Unlicense](LICENSE-CODE). Existing public-domain dedications and other prior licence grants remain effective.


## Building dario.cat and its project docs

Cloudflare Pages uses `make -f .zas/Makefile publish`, output `.zas/deploy`,
through Git integration on `main`. The guarded builder uses the pinned Zas
revision, stages the site, validates its metadata and vanity imports, and
replaces the previous deploy only after every check succeeds. Run
`make -f .zas/Makefile test` for the Go, microformats, HTTP and workflow tests.

The assembled HTML uses root-relative internal navigation so Pages previews
stay on their own host. The finalizer also covers links projected from CV data,
older articles and fetched documentation. Canonical head metadata, hidden
microformat identities, feeds, Markdown and portable agent contexts retain
production URLs; visible microformat links carry their canonical UID separately.
Raw Go vanity fragments remain byte-identical. Repository-only documentation
links point to GitHub because those files are excluded from the published site.

[`.zas/projects.yml`](https://github.com/darccio/darccio/blob/main/.zas/projects.yml) selects immutable public source commits
and explicit documentation paths from Mergo, Zas and darna. Only the selected
regular Git blobs and the original repository license are read; no fetched
checkout, build script, template action, raw HTML or embed executes. A safe
Markdown renderer produces data for the trusted site layout, with
`zas generate -full -no-plugins` still enabled. Project landings retain the exact
original `go-import` and `go-source` tags; the other import fragments stay
byte-identical.

Documentation is in English and retains BSD-3-Clause (Mergo),
AGPL-3.0-or-later (Zas) and Apache-2.0 (darna), including exact license files.
`/zas/configuration/` is the HTML guide; its declared twin is
`/zas/configuration.md`, preserving source-relative Markdown links. Literal Go
template examples are preserved in these docs twins. Personal pages keep their
existing rendered-twin conventions. Each project has a docs navigation list,
source-revision link, `/llms.txt` and `/llms-full.txt` under its root.

Each project's `docs-dispatch` workflow requests a source update after default
branch docs changes, a published release or a manual run. Configure
`SITE_DISPATCH_TOKEN` in **each source repository** as a fine-grained token with
only **Contents: write** on `darccio/darccio`, which GitHub's repository-dispatch
endpoint requires. Without the token the source workflow reports a notice and
skips dispatch; ordinary site builds need no credentials for these public repos.

The site's [Docs source update workflow](https://github.com/darccio/darccio/blob/main/.github/workflows/docs-update.yml)
validates the repository and full SHA, fetches the selected files, runs the
complete guarded build, and opens a manifest-only pin update PR. Enable Actions'
permission to create pull requests in this site's repository settings. Its
`GITHUB_TOKEN` explicitly dispatches Site checks for the bot branch; merging
the PR triggers Pages through the existing `main` build. Dispatch alone does
not deploy, and no workflow pushes fetched content or commits directly to main.
For a manual proposal, dispatch the workflow with the reviewed repository and
commit, or run:

```sh
go -C .tools/vanity run ./cmd/docs-update -root ../.. \
  -repo darccio/zas -commit <full-source-commit-sha>
make -f .zas/Makefile publish
```

Adding a page requires its manifest entry and supported local link targets.
Unlisted pages/assets are not ingested. A missing source, mismatched license copy,
unsupported link, failed fetch or rejected output leaves the last good deploy
intact. Restore an earlier manifest pin to roll back docs, or revert the pipeline
commit to restore the untouched original vanity landings. Operational receipts
remain in [the implementation plan](https://github.com/darccio/darccio/blob/main/docs/site-plan/PROGRESS.md) and are not
published as site content.

## Publishing formats and maintenance

The tested generator pin is
`github.com/darccio/zas@v0.0.10`.
Native RSS is opt-in through `site.feed`; rendered Markdown twins use
`site.publish_markdown`. The main feed combines languages, while `/en/feed.xml`
and `/es/feed.xml` select their respective posts. Complete explicit source dates
drive RSS, h-entry, JSON-LD and Open Graph; a partial historical date stays partial
and is excluded from RSS and AT Protocol publishing.

The guarded build checks all seven exact vanity tuples, representative identity,
article/archive microformats, JSON-LD and cross-format dates, canonical URLs,
summaries and tags before replacing output. It creates the Standard.site record
plan locally; builds need no PDS credentials or connectivity. Reuse the pinned
binary with `make -f .zas/Makefile generate ZAS=/tmp/dario-site-zas/zas`, and run
`make -f .zas/Makefile check-consistency` to inspect an existing deployment.

The [Standard.site document workflow](https://github.com/darccio/darccio/blob/main/.github/workflows/standard-site.yml) starts
after a confirmed production Pages deployment and independently verifies its
exact revision. It publishes only the committed reviewed allowlist, using
`ATP_HANDLE` and `ATP_APP_PASSWORD` in one step. The site deployment never waits
for publishing. Only verified public URIs enter a separate backlink PR; private
recovery state and credentials stay out of Git and artifacts. Manual dispatch
defaults to an offline preview. See the
[publisher instructions](https://github.com/darccio/darccio/blob/main/.tools/atproto/README.md)
for approval, retry, deletion and rollback procedures.

Maintain the documented ownership boundaries: generic features belong in Zas;
templates, curation, source-date provenance and site assembly belong here.
The operational `docs/HANDOFF-FACTS.md`, `docs/publication-provenance.md`,
`docs/standard-site-backfill.md` and `docs/site-plan/PROGRESS.md` hold the audit,
date evidence, the approved archive inventory and implementation receipts. The
[T30 receipt](https://github.com/darccio/darccio/blob/main/docs/site-plan/tasks/T30.md) records the 26 published articles,
production verification links, reader checks and zero-write online rerun.
They are excluded from the deployed site. Registrar renewal, DNSSEC and external
alert delivery require separate account evidence in `docs/DOMAIN-OPERATIONS.md`.
