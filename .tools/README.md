# Site checks and builds

Run these commands from the site repository root:

```sh
make -f .zas/Makefile test
make -f .zas/Makefile generate                  # installed Zas
make -f .zas/Makefile publish                   # installs the pinned Zas revision
make -f .zas/Makefile check-vanity
make -f .zas/Makefile check-vanity-live
make -f .zas/Makefile check-microformats
```

`publish` assembles local `.zas/deploy`; the site's Cloudflare Pages Git integration controls actual deployment. The build-only GitHub workflow does not deploy. `vanity-after-deploy.yml` starts on pushes to production `main` (or a manual retry on `main`), then waits up to eight minutes for that exact SHA's completed, successful `Cloudflare Pages` check from the observed Cloudflare app/project, with its deployment-success title and a `main` check suite. It fails on terminal deployment failures, an unconfirmed result, or timeout; superseded commits are skipped. Only then does it run the shared live checker against `https://dario.cat`, using the deployed commit's baseline. It reads GitHub metadata with `contents: read` and `checks: read`; it never dispatches a deployment. The scheduled/manual monitor remains available independently.

The post-deploy workflow polls the real Cloudflare result because [GitHub restricts `check_run` workflow delivery](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#check_run) for Actions-related suites/SHAs, and this repository currently has no GitHub deployment records to drive `deployment_status`. Cloudflare's exact app/name/project/title filters were verified from its public successful check and suite; provider format changes fail closed. The first actual post-deployment workflow invocation still needs verification after this file reaches `main`. A live metadata check verifies current public routes, not a cryptographic binding between the custom domain and a particular deployment SHA. An intervening main push is detected before running the checker; later deployment changes remain covered by the recurring monitor. Run the explicit live check after dashboard-only rollbacks or after a deployment that outlasts the polling window.

Both build targets copy working-tree public sources into an isolated tree under `.zas`, omit the seven head-only vanity fragments, and run `zas generate -full -no-plugins`. `publish` pins `v0.0.9-0.20261011133110-abdae2fd51de`, merged Zas commit `abdae2fd51de929d456681fb3f89a14a9776c999`. This immutable revision supports the site's typed date, summary and canonical helpers, native RSS feeds and rendered Markdown companions through `publish_markdown`. Zas v0.0.7 is the minimum version supporting `-no-plugins`; v0.0.6 lacks that flag. `generate` accepts `ZAS=/absolute/path/to/zas` to test a candidate executable. `publish` installs the pinned revision into `ZAS_BIN_DIR` (default `/tmp/dario-site-zas`). Build failure never falls back to plugin execution.

Cloudflare Pages may check out shallow Git history, which would make source modification dates look like the deployment commit's date. Before installation or assembly, `publish` restores full history with `git fetch --unshallow --no-tags origin` when the repository is shallow, then verifies that the history is complete. The fetch preserves the checked-out commit, including detached Pages checkouts. Builds with full history skip this fetch. A failed fetch or incomplete history stops the build before output can be promoted.

Git selects tracked and nonignored untracked files, including local edits. Hidden account/tool directories, `node_modules`, previous output, and planning/operations documents are excluded. Only `.zas` configuration, `.zas.yml` files, and `.well-known` public paths are allowed hidden inputs. Source symlinks are rejected. Git history remains available through `GIT_DIR`/`GIT_WORK_TREE` against the original repository; it is never copied. Original source mtimes are preserved. A YAML parser rewrites only the staged `zas.deploy` path to keep generation inside the stage; the source configuration is unchanged.

After generation, each raw vanity source is copied byte for byte into its original `name/index.html` path. The checker scans all output HTML, requires the embedded reviewed metadata baseline for all seven routes, and rejects extra routes or `go-source` tags. Its HTML tokenizer accepts existing head fragments and full documents, ignores comments, script strings and escaped examples, and rejects metadata in bodies. Attribute order/case and tuple whitespace do not affect matching; import prefix, VCS and repository identity must remain exact. Duplicate metadata tags fail, including identical tags. Within one tag, the tokenizer retains the first occurrence of each attribute, matching Go discovery. Baseline changes require deliberate review; there is no automatic refresh command.

When native sitemaps are enabled, assembly finalizes `sitemap.xml` after restoring the seven raw vanity pages. It aligns native locations and translation links with each deployed page's canonical URL, preserves native last-modified dates and translation groups, and adds restored vanity entries using source Git dates when available. Complete HTML coverage, local canonical URLs and reciprocal language links are validated before promotion. This site supports one sitemap; sharded output and unfamiliar XML fields fail explicitly, preserving the previous deployment.

With `site.publish_markdown: true`, Zas emits post-template Markdown companions. After assembly, the builder inserts one Markdown alternate link only where the matching deployed `.md` file exists. HTML sources, unpublished pages and the seven raw vanity pages get no invented twins or links. Discovery URLs follow actual filenames, including `index.md`; invalid or duplicate links fail before promotion. The insertion preserves all existing HTML bytes.

Both build targets also validate microformats in the completed stage before output is promoted. The hidden `.tools/microformats` module parses HTML with `willnorris.com/go/microformats` and selects the representative identity with its `rhc` helper, rather than relying on item order or accepting an organization card. It pins immutable revision `22f0c2eff25b`; the last tagged release, v1.2.0, does not contain `rhc`. Home and about must identify Dario with the stable homepage URL/UID, a local avatar, the public email and all seven verification profiles. Every public post must retain its source name, content, author, canonical URL, summary, tags, language and available publication/update instants. The checker reuses metadata helpers from the same pinned Zas revision as production. Only tracked source history supplies a missing update date; publication dates are never inferred from Git.

The three archives must contain exactly their source-derived article sets, with no duplicate URLs, visible language labels and the known publication precision. Currently there are 33 public local posts, 30 explicit publication instants and three sourced partial dates; the combined archive also retains two separately marked external citations. The live draft URL remains generated but cannot become an entry or appear in an archive. The checker also validates the six dated slash entries, project items, slash-index updates and the dated talks against their existing public sources. It preserves day precision on events and does not invent timestamps for undated talks.

The same checker validates one JSON-LD object in each rendered page's head, with a shared `Person` and a `BlogPosting` only for eligible public articles. Identity, local avatar, job and the seven verified profiles must agree; article title, canonical URL, description, tags, language and known dates must match public sources and the existing microformats checks. Unknown dates are omitted, including the three partial publication dates. Date offsets and fractional seconds are preserved. Only the seven reviewed raw vanity fragments bypass the layout; they must remain free of JSON-LD. Duplicate JSON keys, malformed or quoted documents, duplicated/misplaced scripts, missing Sifa or Speaker Deck links and invented article nodes fail validation. Escaped quotes, ampersands, Unicode and script-like text round-trip as data in local fixtures. An unrelated Cloudflare beacon does not alter the metadata checks. Project source-code schemas remain deferred until the docs pipeline establishes their repository and licence facts.

Checks perform no HTTP requests, process no CV data and exclude hidden tooling and private planning artifacts from output. Unit tests use local HTML fixtures and deliberate semantic failures, including organization identity, wrong timezone or author, empty content, missing/duplicate entries, draft exposure, hidden language labels and invented event times. The article content adapter supports the current literal sources; a future article containing a Go template action or embed fails explicitly until its rendered-source adapter is added. The contact page's existing public HTML fragment is the only embedded source supported for slash-page checks.

Run the checker separately against any completed deployment with explicit source and output roots:

```sh
go -C .tools/microformats run -mod=readonly ./cmd/microformats -root ../.. -dir ../../.zas/deploy
go -C .tools/microformats test -mod=readonly ./...
```

`make test` runs both isolated tooling modules. The Site checks workflow caches both pinned dependency sets and uses the same protected `publish` command configured in Cloudflare Pages. A failed or unavailable microformats checker stops staged assembly before the previous deployment is replaced.

The live CLI is independent of its working directory:

```sh
go -C .tools/vanity run ./cmd/vanity -live -base-url https://dario.cat -timeout 10s
go -C .tools/vanity run ./cmd/vanity -dir ../../.zas/deploy
```

It checks both bare and trailing-slash paths with `?go-get=1`, final HTTP 200 and `text/html`, and the same exact metadata. HTTPS redirects must stay on the same host/port and preserve the query; at most five redirects and 1 MiB per response are allowed. The timeout includes redirects for each of fourteen requests. No module checkout or Git host availability is required. These checks verify import discovery metadata, not repository contents or every package subpath.

Only checked output is promoted using same-filesystem directory renames. Normal promotion failure restores the old deployment. This is not a filesystem transaction: a process/host crash between renames can leave the old tree under `.zas/.site-build-*/previous-deploy`. Recover that tree before deleting the stage. A stale `.zas/.site-build-lock` may be removed after confirming no build is running. A failed generation or validation leaves the previous deploy untouched. Build tests exercise that behavior, secret exclusions, Git history access, raw byte preservation and promotion rollback; live tests use local TLS HTTP servers.
