# Site checks and builds

Run these commands from the site repository root:

```sh
make -f .zas/Makefile test
make -f .zas/Makefile generate                  # installed Zas
make -f .zas/Makefile publish                   # installs the pinned Zas revision
make -f .zas/Makefile check-vanity
make -f .zas/Makefile check-vanity-live
```

`publish` assembles local `.zas/deploy`; the site's Cloudflare Pages Git integration controls actual deployment. The build-only GitHub workflow does not deploy. `vanity-after-deploy.yml` starts on pushes to production `main` (or a manual retry on `main`), then waits up to eight minutes for that exact SHA's completed, successful `Cloudflare Pages` check from the observed Cloudflare app/project, with its deployment-success title and a `main` check suite. It fails on terminal deployment failures, an unconfirmed result, or timeout; superseded commits are skipped. Only then does it run the shared live checker against `https://dario.cat`, using the deployed commit's baseline. It reads GitHub metadata with `contents: read` and `checks: read`; it never dispatches a deployment. The scheduled/manual monitor remains available independently.

The post-deploy workflow polls the real Cloudflare result because [GitHub restricts `check_run` workflow delivery](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#check_run) for Actions-related suites/SHAs, and this repository currently has no GitHub deployment records to drive `deployment_status`. Cloudflare's exact app/name/project/title filters were verified from its public successful check and suite; provider format changes fail closed. The first actual post-deployment workflow invocation still needs verification after this file reaches `main`. A live metadata check verifies current public routes, not a cryptographic binding between the custom domain and a particular deployment SHA. An intervening main push is detected before running the checker; later deployment changes remain covered by the recurring monitor. Run the explicit live check after dashboard-only rollbacks or after a deployment that outlasts the polling window.

Both build targets copy working-tree public sources into an isolated tree under `.zas`, omit the seven head-only vanity fragments, and run `zas generate -full -no-plugins`. `publish` pins `v0.0.9-0.20261011133110-abdae2fd51de`, merged Zas commit `abdae2fd51de929d456681fb3f89a14a9776c999`. This immutable revision supports the site's typed date, summary and canonical helpers, native RSS feeds and rendered Markdown companions through `publish_markdown`. Zas v0.0.7 is the minimum version supporting `-no-plugins`; v0.0.6 lacks that flag. `generate` accepts `ZAS=/absolute/path/to/zas` to test a candidate executable. `publish` installs the pinned revision into `ZAS_BIN_DIR` (default `/tmp/dario-site-zas`). Build failure never falls back to plugin execution.

Git selects tracked and nonignored untracked files, including local edits. Hidden account/tool directories, `node_modules`, previous output, and planning/operations documents are excluded. Only `.zas` configuration, `.zas.yml` files, and `.well-known` public paths are allowed hidden inputs. Source symlinks are rejected. Git history remains available through `GIT_DIR`/`GIT_WORK_TREE` against the original repository; it is never copied. Original source mtimes are preserved. A YAML parser rewrites only the staged `zas.deploy` path to keep generation inside the stage; the source configuration is unchanged.

After generation, each raw vanity source is copied byte for byte into its original `name/index.html` path. The checker scans all output HTML, requires the embedded reviewed metadata baseline for all seven routes, and rejects extra routes or `go-source` tags. Its HTML tokenizer accepts existing head fragments and full documents, ignores comments, script strings and escaped examples, and rejects metadata in bodies. Attribute order/case and tuple whitespace do not affect matching; import prefix, VCS and repository identity must remain exact. Duplicate metadata tags fail, including identical tags. Within one tag, the tokenizer retains the first occurrence of each attribute, matching Go discovery. Baseline changes require deliberate review; there is no automatic refresh command.

When native sitemaps are enabled, assembly finalizes `sitemap.xml` after restoring the seven raw vanity pages. It aligns native locations and translation links with each deployed page's canonical URL, preserves native last-modified dates and translation groups, and adds restored vanity entries using source Git dates when available. Complete HTML coverage, local canonical URLs and reciprocal language links are validated before promotion. This site supports one sitemap; sharded output and unfamiliar XML fields fail explicitly, preserving the previous deployment.

The live CLI is independent of its working directory:

```sh
go -C .tools/vanity run ./cmd/vanity -live -base-url https://dario.cat -timeout 10s
go -C .tools/vanity run ./cmd/vanity -dir ../../.zas/deploy
```

It checks both bare and trailing-slash paths with `?go-get=1`, final HTTP 200 and `text/html`, and the same exact metadata. HTTPS redirects must stay on the same host/port and preserve the query; at most five redirects and 1 MiB per response are allowed. The timeout includes redirects for each of fourteen requests. No module checkout or Git host availability is required. These checks verify import discovery metadata, not repository contents or every package subpath.

Only checked output is promoted using same-filesystem directory renames. Normal promotion failure restores the old deployment. This is not a filesystem transaction: a process/host crash between renames can leave the old tree under `.zas/.site-build-*/previous-deploy`. Recover that tree before deleting the stage. A stale `.zas/.site-build-lock` may be removed after confirming no build is running. A failed generation or validation leaves the previous deploy untouched. Build tests exercise that behavior, secret exclusions, Git history access, raw byte preservation and promotion rollback; live tests use local TLS HTTP servers.
