<!--
publish: false
-->

# Site implementation progress

All 33 tasks have a dedicated completed subagent review. The execution plan and D1–D17 register are committed. Zas PRs #111 and #112 are merged; the site pins the merged helpers and has a second verified implementation batch committed locally. Site deployment and account operations remain separate evidence requirements.

| Item | Actual outcome | Commit or PR | Verification | Remaining work |
| --- | --- | --- | --- | --- |
| T00 | Timestamped source/live audit and subsequent owner hosting confirmation | `6d12b55`, updated in this progress batch | Seven exact vanity tuples, account DID and feed recorded; owner confirms Pages main/build/output settings | Recheck after deployment; registrar and edge settings need account evidence |
| Organization | T00–T32 mapped to dependencies, atomic units, checks and rollback; D1–D17 recorded | `ba14921`, progress updates | 33 distinct reviews; complete IDs; acyclic dependency graph | Execute queued items as dependencies become ready |
| Build prerequisite | Duplicate Spanish translation key corrected | `a4b35b5` | Clean builds succeed with corrected i18n | Deployment pending |
| T02 | Exact seven-route local/live checker, staged assembly and build-only CI | `1defa78` | Vet, race/shuffle, workflow lint, actual release/candidate builds; 14 live routes and direct Mergo v1.0.2 resolution; real v0.0.6 failure preserves previous output | Push and observe Pages guarded build |
| T02 post-deploy | Bounded exact-main-SHA Cloudflare-success poll followed by shared live checker | `7cdbe54` | actionlint; actual provider check/suite classified correctly; 13 classifier mutations and suite gates checked; read-only permissions | Observe first real invocation after push; dashboard-only/slow deployments need explicit live check |
| T01 | Scheduled backstop and domain operations runbook | `35aaf30` | Workflow lint and shared live checker pass | Push/observe workflow; complete renewal/payment/lock, DNSSEC and independent monitor with real evidence |
| T03 | Zas current contact links merged | `9045f6f`, [PR #111](https://github.com/darccio/zas/pull/111), merge `0c94388` | README-only diff; whitespace, darna and GitHub checks pass | Mergo source-owned follow-up remains queued |
| T05 | Lazy typed metadata/canonical helpers merged | `96f3a64`, [PR #112](https://github.com/darccio/zas/pull/112), merge `f463876` | Build, vet, race/shuffle and pinned lint; 68 real-site files and six fixtures byte-identical before adoption; remote CI/CodeQL pass | Generator release remains T32 work |
| Generator adoption | Immutable merged Zas revision pinned in guarded publish command | `9bcfd95` | Installed module is v0.0.9-0.20261011090514-f463876749cb; actual make publish succeeds | Deployment pending |
| T04 | 34 article metadata mappings, summaries, tags, language defaults and provenance | `00a95a2` | 31 source instants copied exactly; three incomplete dates omitted; all article bodies and both archive bodies preserved; ca=13/en=9/es=12 | Recover missing precision for three excluded articles; deployment pending |
| T08 avatar | Existing portrait stored as local 180px/96px WebP; homepage/OG/vCard updated; preconnect removed | `482d0f1` | Both images decode at exact dimensions; assets copied byte-identically; plain mailto and existing identity links retained | Disable observed Cloudflare email obfuscation; live verification after deploy |
| T08 licence | Content/software scopes reconciled with explicit exceptions | `d3ed333` | Original Unlicense preserved byte-identically; CV/MAC paragraphs retained; Lýd and third-party terms and prior grants explicit | Inspect deployed licence links and GitHub display after push |
| T07 | Canonical/OG URLs, article classification/times, locales and descriptions | `6089a38` | All 54 assembled HTML canonical/OG pairs, 34 article summaries/types/languages and 31 exact times pass; three partial times absent; seven vanity tuples unchanged | Deployment and live metadata verification |

The six new implementation commits are signed and atomic. Site commits remain local: nothing has been pushed or deployed. Both Zas PRs were merged by the owner on 2026-10-11; no releases were tagged, credentials committed or PDS records written. The pre-existing sibling Zas checkout remains clean on feat/xml-sitemap-generation at 9c8c0b2.

## Verification evidence

The initial site tooling passed go -C .tools/vanity vet ./... and go -C .tools/vanity test -race -shuffle=on ./.... Its workflows passed actionlint. Parser tests reject missing/changed/duplicate tags and homepage fallbacks, preserve HTML first-attribute semantics and exercise bounded HTTPS redirects. Builder tests preserve previous output after failure and exclude hidden/planning data. A real unsupported v0.0.6 invocation left deployment bytes, modes and mtimes unchanged.

The second batch exercised the actual publish recipe with the exact merged module version, followed by a parsed-HTML audit of assembled output. A separate clean clone of the five implementation commits also builds and passes all 53 committed HTML canonical/OG pairs; the working-tree build has one additional public page from the unchanged user-owned untracked document. Every article has the expected language, shared summary and article classification; exact publication metadata retains source strings and offsets. The draft remains live and keeps its original multiple source headings; no new H1 is introduced. The seven vanity sources gain canonical/OG tags while every original source byte remains an exact suffix; assembly copies the resulting sources unchanged and the exact Go tuple checks pass.

Seven isolated layout cases cover canonical directory/standalone URLs, explicit article classification, missing and fractional dates, description precedence, quote/ampersand escaping, locales and image URL forms. Source/YAML audits cover all 34 article comments, the two language defaults and three unresolved timestamps. Private provenance, plan and operations documents remain absent from output. Existing nonignored untracked public content is included by the documented working-tree build policy; the user-owned vanity document itself remains unchanged and untracked.

Zas's merged helpers previously passed build, vet, race/shuffle and golangci-lint v2.11.4 with zero issues, plus remote CI/CodeQL. Temporary Git fixture tests used command-scoped empty global/system config; actual commits retained configured signing and hooks. Parent and implementation/review subagents independently checked source-body preservation, metadata precision and licence/avatar scope.

## Current account and repository conditions

Cloudflare Pages Git integration deploys pushes to main. The owner reports build command make -f .zas/Makefile publish, output .zas/deploy, blank root directory and enabled build comments. The guarded command is already configured; committed pin/assembly changes need no dashboard update. The public GitHub integration check is named Cloudflare Pages from app cloudflare-workers-and-pages; no GitHub deployment records or tracked deployment workflow existed in the baseline.

Public .cat RDAP reports expiry on 1 June 2027 and delegationSigned:false. Auto-renew, valid payment, transfer lock and independent monitor configuration still need actual account evidence. T00 also observed edge email obfuscation, which the local contact source cannot disable.

Only this site and Zas are implementation repositories in this request. Mergo and darna source-owned changes remain explicit follow-ups. Existing docs/go-vanity-url.md remains untracked and unchanged (SHA-256 b1fca26739cf9d022923a2c2ebe7445b60ac10f8f89126ca7b654e946201de89). Zas work used isolated branches; origin/master is now f463876 while the user's checkout is preserved.

## Implementation queue

T09 navigation and Tier 1 slash pages and T10 crawler policy/security/sitemap are next. T11 native RSS and T21 Markdown output are ready for separate atomic Zas feature PRs; site enablement follows merged, tested pins. Feed selection must exclude archives, drafts and feed:false before validating dates, so the three unresolved timestamps do not acquire fabricated precision or block eligible items. Later microformats, agent surfaces, docs and publishing remain queued; optional T06 and T23 are deferred.

Existing authorization and report defaults continue to apply. Registrar/account operations and reviewed historical publication selections need real evidence. Standard.site work begins with offline previews; actual public writes require the reviewed allowlist and account access, while the site build remains independent of the PDS.
