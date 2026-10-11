<!--
publish: false
-->

# Site implementation progress

All 33 tasks have a dedicated completed subagent review. The execution plan and D1–D17 decision register are committed. The first implementation slice is also committed; implementation, deployment and account evidence are tracked separately below.

| Item | Actual outcome | Commit or PR | Verification | Remaining work |
| --- | --- | --- | --- | --- |
| T00 | Timestamped source and live audit recorded | `6d12b55` | Facts file exists; all seven exact vanity tuples observed; DID and feed parsed; unrelated user doc hash preserved | Recheck after deployment; remaining Pages settings and registrar controls need account evidence |
| Organization | T00–T32 mapped to dependencies, atomic units, checks and rollback; D1–D17 recorded | `ba14921`, followed by this progress update | 33 distinct reviews; complete IDs; acyclic task dependency graph | Execute queued items as dependencies become ready |
| Build prerequisite | Duplicate Spanish translation key corrected | `a4b35b5` | Clean builds succeed with the corrected i18n config | Deploy through the confirmed Pages build path |
| T02 | Local/live checker, exact seven-route baseline, staged assembly, pinned v0.0.7 and build-only CI | `1defa78` | Vet, race/shuffle, workflow lint, release and candidate builds pass; local output and all 14 live URLs match; direct Mergo resolution returns v1.0.2; actual v0.0.6 failure preserves prior output | Confirm Pages build/output settings, activate checks, and attach post-deploy verification to the real deployment signal |
| T01 | Scheduled backstop and domain operations runbook ready | `35aaf30` | Workflow lint passes; shared live checker passes; account procedure cites registrar, Cloudflare and GitHub guidance | Push/observe the scheduled workflow; complete renewal/payment/lock, DNSSEC and external monitor operations with real evidence |
| T03 | Zas contact links updated in one atomic docs commit | `9045f6f`, [Zas PR #111](https://github.com/darccio/zas/pull/111) | README-only diff; current identity links checked; whitespace, darna and GitHub checks pass | Merge PR; Mergo source-owned follow-up remains queued |
| T05 | Lazy typed metadata and canonical helpers implemented with tests/docs | `96f3a64`, [Zas PR #112](https://github.com/darccio/zas/pull/112) | Build, vet, race/shuffle and pinned lint pass; 68 real-site files and six fixture files byte-identical; GitHub build/test/lint/CodeQL pass | Merge PR and adopt a tested generator revision before changing site templates |

Site commits are local and have not been pushed or deployed. Both Zas PRs are open against master, each containing one signed atomic commit. Their branches are available in ../zas as docs/current-contact and feat/site-metadata; the pre-existing feat/xml-sitemap-generation checkout is unchanged. No PRs were merged, releases tagged, credentials committed or PDS records written.

## Verification evidence

The site tooling passed go -C .tools/vanity vet ./... and go -C .tools/vanity test -race -shuffle=on ./.... Both GitHub workflows passed actionlint. The production pin was exercised through make -f .zas/Makefile publish; the candidate through generate with an explicit binary. Parser tests reject missing/changed/duplicate tags and homepage fallbacks, preserve HTML first-attribute behavior, and exercise bounded HTTPS redirects and errors. Builder tests preserve previous output after failure and reject hidden/planning output. All seven assembled vanity pages retain exact source bytes. An actual unsupported v0.0.6 invocation left previous deployment bytes, modes and mtimes unchanged.

Parent verification of the final checker passed both the local deployment and all fourteen live bare/slash routes. GOPROXY=direct go list -m dario.cat/mergo@latest used an isolated cache and returned dario.cat/mergo v1.0.2. Live checks describe current production, not a deployment of these new commits.

Zas passed go build ./..., go vet ./..., go test -race -shuffle=on ./... and golangci-lint v2.11.4 with zero issues. Local tests used command-scoped empty global/system Git config for temporary Git fixtures; actual commits retained configured signing and hooks. Lint used installed Go 1.26.8 to match its compiled toolchain. The unchanged real-site comparison used preserved baseline input and found 68 matching paths and bytes, without missing or extra files. Upstream PR #112 also passed remote CI and CodeQL.

## Current account and repository conditions

Cloudflare Pages with Git integration is confirmed by the owner. The GitHub API shows a successful `Cloudflare Pages` check from app `cloudflare-workers-and-pages` on the production branch; no GitHub deployment records or tracked site deployment workflow exist. Confirm the production branch/build/output settings before attaching host-specific actions.

Public .cat RDAP reports expiry on 1 June 2027 and `delegationSigned: false`. Auto-renew, valid payment, transfer lock and external monitor configuration need real account evidence. These settings have not been changed.

Only this site and Zas are implementation repositories in this request. Mergo and darna source-owned changes stay explicit follow-ups. Existing docs/go-vanity-url.md remains untracked and unchanged (SHA-256 b1fca26739cf9d022923a2c2ebe7445b60ac10f8f89126ca7b654e946201de89). Local Zas’s existing branch is preserved; new work uses isolated branches based on upstream master a63c2d9.

## Implementation queue

T04 publication metadata and T08 contact/avatar/licensing can proceed next. T04 must retain the three unresolved timestamp precision cases and the existing live draft URL rather than inventing instants or removing content. T07 template adoption waits for a tested merged T05 revision. Native feeds and Markdown output continue as separate Zas feature PRs. Subsequent content, formats, docs and publishing tasks remain queued; optional T06 and T23 are deferred.

Registrar/hosting/account access and report-required reviewed publication selections are evidence requirements. Existing session authorization continues to apply; do not ask again for actions or defaults already approved. Standard.site implementation starts with offline previews and requires the approved allowlist and actual account access before public writes.
