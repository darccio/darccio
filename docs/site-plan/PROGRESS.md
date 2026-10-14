<!--
publish: false
-->

# Site implementation progress

The preceding implementation PRs are merged, and the site adopts released Zas v0.0.9. T26's production publication backlink is verified. T01 is complete: registrar renewal/protection, fully validated DNSSEC, independent scheduled monitoring, real failure/recovery delivery and three expiry-notification simulations are confirmed. T04 and T30 are complete. The English CV is merged in PR #27; its production JSON matches the preserved source, while a Cloudflare email-obfuscation regression needs the correction recorded below. The owner deferred the remaining T09 work and authorized six design/accessibility units. Optional T06/T23 remain deferred. The chronology below retains previous observations and their dates.

The owner added [T33: recover historical articles](tasks/T33.md) on 14 October
2026. This additional backlog item is queued after the original T00–T32 plan;
no external articles have been fetched, imported or syndicated for it.
The owner also added [T34: recover social posts and threads](tasks/T34.md) for
Facebook, Twitter and Bluesky. This is queued separately and reuses T33’s recovery
approach; Twitter and Bluesky include thread reconstruction and related collections.

All 33 tasks have a dedicated completed subagent review. Zas PRs #111–#115 are merged. The site now pins the merged feed/Markdown revision and has navigation, core slash pages, crawler/security resources, complete canonical sitemaps, RSS discovery, Markdown twins and static response headers deployed. Cloudflare deployed 20a3e10 and source-history fix 594cf0d successfully on 2026-10-11; site CI and post-deploy vanity checks passed. The owner's expanded talks catalogue, cc@dario.cat contact and Sifa profile additions deployed successfully in 8e3acb1. The T14–T18 identity, post/archive microformats, supported talk/slash/project markup and protected parser checks deployed successfully in 28bd9ba; the narrow historical-email follow-up is committed as d21a476. Account operations remain separate evidence requirements.

| Item | Actual outcome | Commit or PR | Verification | Remaining work |
| --- | --- | --- | --- | --- |
| T00 | Timestamped source/live audit and subsequent owner hosting confirmation | `6d12b55`, updated in this progress batch | Seven exact vanity tuples, account DID and feed recorded; owner confirms Pages main/build/output settings | Recheck after deployment; registrar and edge settings need account evidence |
| Organization | T00–T32 mapped to dependencies, atomic units, checks and rollback; D1–D17 recorded | `ba14921`, progress updates | 33 distinct reviews; complete IDs; acyclic dependency graph | Execute queued items as dependencies become ready |
| Build prerequisite | Duplicate Spanish translation key corrected | `a4b35b5` | Clean builds succeed with corrected i18n | Deployment pending |
| T02 | Exact seven-route local/live checker, staged assembly and build-only CI | `1defa78` | Vet, race/shuffle, workflow lint, actual release/candidate builds; 14 live routes and direct Mergo v1.0.2 resolution; real v0.0.6 failure preserves previous output | Push and observe Pages guarded build |
| T02 post-deploy | Bounded exact-main-SHA Cloudflare-success poll followed by shared live checker | `7cdbe54` | actionlint; actual provider check/suite classified correctly; 13 classifier mutations and suite gates checked; read-only permissions | Observe first real invocation after push; dashboard-only/slow deployments need explicit live check |
| T01 | Registrar protection, validated DNSSEC, scheduled backstop and active Datadog monitoring complete | `35aaf30`, operational receipts in PR #16, Datadog PRs #17–#21 and completion receipt below | Paid expiry 2028-06-01; DNSSEC fully validates; eight live tests have scheduled passes; real failure/recovery and three expiry simulations received; temporary probe removed | Ongoing quarterly and post-change review |
| T03 | Zas current contact links merged | `9045f6f`, [PR #111](https://github.com/darccio/zas/pull/111), merge `0c94388` | README-only diff; whitespace, darna and GitHub checks pass | Mergo source-owned follow-up remains queued |
| T05 | Lazy typed metadata/canonical helpers merged | `96f3a64`, [PR #112](https://github.com/darccio/zas/pull/112), merge `f463876` | Build, vet, race/shuffle and pinned lint; 68 real-site files and six fixtures byte-identical before adoption; remote CI/CodeQL pass | Generator release remains T32 work |
| Generator adoption | Immutable merged feed/Markdown revision pinned in guarded publish command | `fc67575` (previous helper adoption `9bcfd95`) | Installed module is v0.0.9-0.20261011133110-abdae2fd51de; disabled options preserve all 78 baseline files; actual make publish succeeds | Deployment pending |
| T04 | 34 article metadata mappings, summaries, tags, language defaults and provenance; all publication dates complete | `00a95a2`, `a5896cc`, PR #23 merge `bcd015c` | All 33 public posts have explicit dates; original RSS, owner timezone confirmation and recorded Spanish clock-time assumption resolve the gaps; production formats agree | Complete; retain provenance for any later correction |
| T08 avatar | Existing portrait stored as local 180px/96px WebP; homepage/OG/vCard updated; preconnect removed | `482d0f1` | Both images decode at exact dimensions; assets copied byte-identically; plain mailto and existing identity links retained | Disable observed Cloudflare email obfuscation; live verification after deploy |
| T08 licence | Content/software scopes reconciled with explicit exceptions | `d3ed333` | Original Unlicense preserved byte-identically; CV/MAC paragraphs retained; Lýd and third-party terms and prior grants explicit | Inspect deployed licence links and GitHub display after push |
| T07 | Canonical/OG URLs, article classification/times, locales and descriptions | `6089a38` | All 54 assembled HTML canonical/OG pairs, 34 article summaries/types/languages and 31 exact times pass; three partial times absent; seven vanity tuples unchanged | Deployment and live metadata verification |
| T21 | Merged default-off rendered twins; enabled here with inventory-based head discovery | [PR #113](https://github.com/darccio/zas/pull/113), merge `643967d`; site `b782b00` | 53 committed public twins and exact matching links; 54 with unchanged user-owned document; full tooling tests and independent review pass | Live Markdown bodies/MIME/discovery confirmed; T20 deployed; T22 negotiation remains queued |
| T11 | Merged native RSS 2.0 with owned lifecycle and expected XML fixtures; enabled on site | [PR #114](https://github.com/darccio/zas/pull/114), merge `abdae2f` | 30 combined, 7 English and 11 Spanish items; source publication instants, canonical GUIDs, namespaces and absolute content URLs agree; repeated output deterministic | Live W3C validation after deploy |
| T09 | Accessible five-link navigation, Catalan home hub and nine core slash pages | `1ad3087` | Owner-confirmed Corb/Cocoon on dated Now; axe finds zero violations on 12 pages; keyboard skip, 320px, double-font reflow and no-JS contact checks pass | Deploy; supplementary Tier 2 content remains queued |
| T10 | Explicit crawler policy, security contact and final native sitemap assembly | `0556a86` | Complete 62-page canonical coverage including 7 raw vanity pages; reciprocal language URLs and Git dates retained; one robots Sitemap line; failure preserves prior deploy | Live resource checks; maintain security expiry before 2027-10-11 |
| T12 | Strict RSS enablement, language-aware autodiscovery and subscription hub | `cd7e20a` | Every templated page advertises real generated feeds; current Rizoma/Mergo/Zas external feeds parse as RSS/Atom | Deploy/live discovery checks; archive h-feed awaits T16 |
| T13 | Static Cloudflare Pages RSS/security/Markdown MIME and cache rules; matching Content-Signal | `3c3b196` | Exact _headers bytes survive assembly; reviewed patterns preserve WebFinger and absent verification resources | Live header checks; future Functions must set their own response headers |
| T08 contact correction | Owner-approved cc@dario.cat in public contact, vCard, security and funding sources, with per-address obfuscation opt-outs | `6deaeda` | Five generated contact-bearing pages have correct readable links, exact opt-out comments and no decoder scripts; machine metadata agrees | Push and verify actual production edge responses; global dashboard toggle remains unchanged |
| Sifa profile correction | Owner-approved Sifa beside LinkedIn in visible profiles, vCard, README and both existing CV profile collections | `df0f35e` | Profile resolves with matching canonical; rel=me preserved; CV comparison confirms only two requested URL additions | Push/live verification; head discovery and Person sameAs remain T14/T19 work |
| T17 catalogue correction | 31 entries with all 15 Speaker Deck decks, 14 YouTube recordings, Commons and additional GitHub materials | `dcb2a2b` | All 12 owner-supplied video IDs appear once; complete inventory and source-date checks; six-page axe/keyboard/reflow/no-JS checks pass | Push/live inventory check; event markup and the other T17 subitems remain queued |
| T14 | Scoped home/about representative identity; stable header card and seven head profiles including Sifa | `1c52241` | Actual representative-card parser selects Dario; local photo/email/root UID and every head profile pass | Live checks pass in 28bd9ba; Mastodon website already verified |
| T16 | Complete combined, English and Spanish h-feed archives | `22108fa` | 33/9/12 source-derived local entries; two retained external citations; exact 30/7/11 timestamps and visible language labels | Live archive checks pass; three partial publication dates remain unresolved |
| T15 | Single h-entry/name/content wrapper for all 33 published localized articles | `8556005` | All source bytes outside the sole heading preserved; parser matches metadata/content and excludes draft | Live entry metadata pass; d21a476 corrects one historical email edge rewrite |
| T17 events | Ten documented talks have date-only h-event/location properties | `77b4955` | Verified primary provenance; 31 entries, all decks and recordings retained | Live catalogue/event checks pass; 21 insufficiently documented events remain plain |
| T17 slashes | Six updated h-entry pages and eight dated slash-index entries | `0fb3259` | Shared explicit update values; nested contact card prevents URL leakage; no publication timestamps invented | Live slash checks pass; CV render remains deferred |
| T17 projects | Five project h-item children in a canonical h-feed | `eedc7e2` | Existing summaries/names and real project URLs parse exactly | Live project checks pass; future docs hosting remains T24 |
| T18 | Independent microformats parser fixtures and checks before staged-output promotion | `3f576cb` | Both module race/shuffle suites and vet pass; actionlint pass; exact Cloudflare publish command rejects invalid markup before replacing output | Site checks and protected Pages deployment pass in 28bd9ba |
| T19 Person/posts | Safe layout serialization and shared Person/BlogPosting graphs, checked before promotion | Zas `844e073`, merged [PR #115](https://github.com/darccio/zas/pull/115); site `21f8287`, merged [PR #4](https://github.com/darccio/darccio/pull/4), production `b421134` | Local/remote tests pass; Schema.org zero errors/warnings; rejected output preserves 150 files; [Cloudflare preview](https://40451f37.da-rio-hn.pages.dev) graphs/feed bytes and seven vanity packages match | Production Pages, CI and independent graphs/feed/twin checks pass; SoftwareSourceCode follows T24 manifests |
| T20 | Curated language summaries/full corpora and strict head discovery, assembled before promotion | Signed `9366c92`, [site PR #5](https://github.com/darccio/darccio/pull/5) | Clean 156-file build; race/shuffle and vet; repeated hashes identical; invalid selection preserves old deployment; 111 preview HTTPS checks and all seven vanity packages pass | PR #5 merged as c624d6c; all 111 production checks pass; project contexts follow T24 |
| Accessibility | Wrap fenced and inline code without source or clipboard changes | `e4690db` | Independent before/after code-copy checks; integrated axe, keyboard, 320px/double-font reflow and no-JS checks pass on 18 pages | Deployed stylesheet bytes match; browser checks pass |
| [T33](tasks/T33.md) | Recover the owner’s historical articles from original sites or archive.org | `76d688a`; implementation queued | Per-item task review and private-plan exclusion checks | Inventory sources; recover text/media and publication provenance; import reviewed articles in atomic commits |
| [T34](tasks/T34.md) | Recover Facebook, Twitter and Bluesky posts, including Twitter/Bluesky threads | `d934a3a`; implementation queued | Separate per-item review; guarded private-plan exclusion checks | Select sources, recover records/media and evidenced thread relationships; review local imports |

The site implementation commits are signed and atomic. All 22 commits through 20a3e10 were pushed to origin/main and deployed. Commit 7e10002 fixes source-history restoration for subsequent Pages builds. Zas PRs #111–#114 were merged by the owner on 2026-10-11; no releases were tagged, credentials committed or PDS records written. The pre-existing sibling Zas checkout remains clean on feat/xml-sitemap-generation at 9c8c0b2. The table above preserves the implementation batch's original verification state; the deployment evidence below supersedes its deployment-pending entries.

GitHub confirms Markdown PR #113 merged as 643967d5ae2cb208834581524c5e592104b9dd74 and RSS PR #114 merged as abdae2fd51de929d456681fb3f89a14a9776c999. The immutable module origin matches the latter SHA. Both source-feature and local site signatures were verified against the configured public signing key. The actual sibling checkout is preserved; only its feature remote references were fetched for inspection.

The owner selected publish_markdown for site enablement and page opt-outs. Commit 2a65554 updates the public key, diagnostics, README and test fixtures together, without retaining an alias for the previous unmerged name. The RSS feature was rebased onto that commit with signatures preserved; its pre-merge implementation was a041680. Commit 42adf1c adds checked-in expected combined, English and Spanish RSS XML under testdata/feed-expected, separate from the existing input site at testdata/feed-site. Full and incremental tests compare exact output bytes and retain independent semantic XML assertions. Every emitted fixture article has explicit fixed publication and update timestamps; a deliberately mismatched copied fixture fails without updating its expected bytes.

## Verification evidence

The initial site tooling passed go -C .tools/vanity vet ./... and go -C .tools/vanity test -race -shuffle=on ./.... Its workflows passed actionlint. Parser tests reject missing/changed/duplicate tags and homepage fallbacks, preserve HTML first-attribute semantics and exercise bounded HTTPS redirects. Builder tests preserve previous output after failure and exclude hidden/planning data. A real unsupported v0.0.6 invocation left deployment bytes, modes and mtimes unchanged.

The second batch exercised the actual publish recipe with the exact merged module version, followed by a parsed-HTML audit of assembled output. A separate clean clone of the five implementation commits also builds and passes all 53 committed HTML canonical/OG pairs; the working-tree build has one additional public page from the unchanged user-owned untracked document. Every article has the expected language, shared summary and article classification; exact publication metadata retains source strings and offsets. The draft remains live and keeps its original multiple source headings; no new H1 is introduced. The seven vanity sources gain canonical/OG tags while every original source byte remains an exact suffix; assembly copies the resulting sources unchanged and the exact Go tuple checks pass.

Seven isolated layout cases cover canonical directory/standalone URLs, explicit article classification, missing and fractional dates, description precedence, quote/ampersand escaping, locales and image URL forms. Source/YAML audits cover all 34 article comments, the two language defaults and three unresolved timestamps. Private provenance, plan and operations documents remain absent from output. Existing nonignored untracked public content is included by the documented working-tree build policy; the user-owned vanity document itself remains unchanged and untracked.

Zas's merged helpers previously passed build, vet, race/shuffle and golangci-lint v2.11.4 with zero issues, plus remote CI/CodeQL. Temporary Git fixture tests used command-scoped empty global/system config; actual commits retained configured signing and hooks. Parent and implementation/review subagents independently checked source-body preservation, metadata precision and licence/avatar scope.

The new features passed the full combined build, vet, race/shuffle suite and pinned golangci-lint v2.11.4. Separate clean site fixtures exercised neither option, RSS alone, Markdown alone and both. With both options disabled, all 78 baseline deploy files remain unchanged. RSS emits 30 combined articles, seven English and eleven Spanish; three unresolved timestamps and the existing draft remain excluded. XML parsing verifies namespaces, canonical GUIDs, item languages and source publication instants, while all content links have schemes. Markdown emits 44 complete public twins and resolves the existing contact partial; private config and documents remain absent. Both options together produce 125 files byte-identical across consecutive full builds, with all seven vanity tuples and the complete 53-page metadata audit passing.

GitHub CI and CodeQL passed for both original feature heads. The RSS PR initially had a feature-branch base, so its master-only automatic CodeQL trigger was supplemented with [an explicit run](https://github.com/darccio/zas/actions/runs/38131631110) against original head 457b4b0; that run completed successfully. Both PRs initially reported mergeable against their respective bases. After the original private-plan update, the actual working-tree guarded build also passed all seven vanity checks; the production generator pin at that stage remained unchanged pending merge.

The renamed option and expected-XML follow-ups passed build, vet, full race/shuffle tests, pinned lint and independent review. The updated real-site fixture still produces 44 complete public Markdown twins and 30/7/11 feed items, with all seven vanity routes and the full metadata audit passing; both options disabled still preserve all 78 baseline files byte-for-byte. GitHub CI and CodeQL pass for current Markdown head 2a65554 and RSS head 42adf1c, including [the RSS explicit CodeQL run](https://github.com/darccio/zas/actions/runs/38135220629). PR #114 contains only RSS changes relative to the updated #113 branch. Current remote feature references were refreshed in the unchanged sibling checkout.

Focused regressions cover incremental edits/deletions, missing twins, publication/opt-out transitions, static collisions before full clearing, private ownership cleanup, source handoffs with older timestamps, failed-build/write retries, CDATA/escaping, independent language limits and plugin invocation. Both features retain legacy full-clear behavior after preflight; dynamically computed language targets can only be checked after rendering. The site's guarded staged build preserves the previous published directory on any failure. Content directives that need another layout pass are rejected for RSS or require an explicit feed opt-out; unsupported Markdown content plugins require a twin opt-out. These limits are documented in each PR.

The merged-revision site batch passed the exact publish recipe, full tooling race/shuffle tests and vet, independent sitemap/discovery review, and parsed built-output checks. Six signed atomic implementation commits separate generator adoption, policy/sitemap, navigation, feeds, static headers and Markdown enablement. A clean committed checkout emits 146 files, 62 HTML pages, 53 Markdown twins and 30/7/11 feed items; all bytes are identical across consecutive builds. The working-tree build emits 148 files, 63 HTML pages and 54 twins because the unchanged user-owned untracked document is included by existing build policy. All 148 output files match between installed-candidate generation and the exact publish recipe.

Sitemap assembly preserves native dates and translation groups, canonicalizes their URLs and adds the seven restored raw routes with source Git dates. Complete coverage, canonical identity and reciprocal hreflang are checked before promotion; unsupported shards/XML extensions fail explicitly. Markdown discovery uses actual regular twin files, validates all links before insertion and preserves other HTML bytes. Failure tests keep the previous deploy intact; HTML-only sources and the existing draft have no invented twins. T20 must distinguish its text/markdown LLM-context alternates from the twin validator's current single-companion rule.

Axe reports no violations on /, the nine core slash pages, /posts/ and /talks/. Keyboard checks confirm the first Tab reaches the skip link and Enter focuses main; 320px and double-font reflow checks pass. Contact/navigation work with browser JavaScript disabled. An inline raw email initially triggered Markdown autolinking into an empty nested anchor; using an ordinary Markdown mailto link fixes the rendered accessibility issue. About/contact embeds now resolve relative to their source directories. Now records exactly the owner-confirmed https://corb.ax and https://dario.cat/cocoon links without invented location, reading or community activity.

The static _headers rules apply to all three generated feeds, the existing security file and nested Markdown companions, plus a global matching Content-Signal. No absent AT verification file is advertised or relabeled, and the WebFinger Worker is unchanged. External subscriptions were rechecked with bounded GETs: Rizoma RSS has five items, Mergo releases Atom ten entries, and Zas releases Atom eight entries. Live Pages headers and W3C validation now pass as recorded below. One overlapping local build was rejected by the existing lock; the exact publish recipe subsequently passed when run sequentially.

Cloudflare's check for 20a3e1095a0c49f72b0b92178831c25e8f7112d7 completed successfully with title Deployed successfully and immutable preview https://8c297865.da-rio-hn.pages.dev. [Site checks](https://github.com/darccio/darccio/actions/runs/38146750411) and [Vanity after deploy](https://github.com/darccio/darccio/actions/runs/38146750415) passed. Independent live checks pass for all 14 bare/slash vanity requests, and a fresh-cache GOPROXY=direct query resolves dario.cat/mergo@latest to v1.0.2. Fifteen representative HTML pages have the expected navigation, language, canonical/Open Graph and discovery links; all nine core slash pages and the owner-confirmed Corb/Cocoon links are live. Sitemap coverage is 62 unique canonical pages, including all seven vanity routes.

All three live RSS feeds have correct self-links, content types, one-hour caching and 30/7/11 items. The W3C Feed Validation Service reports valid RSS for the [combined](https://validator.w3.org/feed/check.cgi?url=https%3A%2F%2Fdario.cat%2Ffeed.xml), [English](https://validator.w3.org/feed/check.cgi?url=https%3A%2F%2Fdario.cat%2Fen%2Ffeed.xml) and [Spanish](https://validator.w3.org/feed/check.cgi?url=https%3A%2F%2Fdario.cat%2Fes%2Ffeed.xml) feeds. Publication dates, GUIDs, titles and full content match the local build. Security.txt and direct Markdown companions return their exact expected bytes and MIME types. Checked responses carry the robots-aligned Content-Signal; robots retains the CV exclusion and exactly one Sitemap directive.

The first live build exposed shallow-history update dates: RSS lastBuildDate was the deployed commit timestamp, 2026-10-11T16:01:28+02:00, while the full-history source date is 2026-10-11T11:18:37+02:00. Commit 7e10002 restores full history before installation or assembly, verifies completion, and skips fetching in complete repositories. Integration builds from a detached shallow clone preserve HEAD and produce the correct source date; a full checkout with an unavailable origin still builds; a failed shallow fetch preserves prior output and stops before installation. The fix deployed in 594cf0d: all three live feeds became byte-identical to the full-history build, and all 62 live sitemap entries matched local source-history dates. [Site checks](https://github.com/darccio/darccio/actions/runs/38147525262), [Vanity after deploy](https://github.com/darccio/darccio/actions/runs/38147525202), the independent live checker and all three W3C feed validations passed.

Cloudflare email obfuscation remains enabled on the production domain: homepage, About and Contact receive email-protection links and a decoder script. The exact deployed Pages preview retains plain mailto links and no decoder, confirming that this remaining T08 requirement is an account setting rather than a source defect.

The owner subsequently selected cc@dario.cat as the public address. Commit 6deaeda updates the shared partial, Contact/Hello, vCard, security.txt and funding metadata together. Cloudflare's documented email_off comments exempt visible contact anchors from rewriting without adding JavaScript or changing the global dashboard setting. Actual generated output retains normal Markdown link handling and the exact opt-out pairs. Production behaviour awaits this batch's push; the preceding paragraph records the previous deployed version.

The owner also selected https://sifa.id/p/dario.cat alongside LinkedIn. Commit df0f35e adds it to current visible/machine-readable profile contexts and future identity/JSON-LD acceptance. The CV edit is limited to adding that link in the two existing LinkedIn profile arrays under the owner's explicit instruction; all other CV fields and restrictions remain intact, with no CV render introduced. Historical article links and baseline audit evidence remain unchanged.

Talks research fetched all 15 decks and source materials, then verified the owner's 12 individual recording links. Fourteen distinct YouTube recordings are included overall, plus the retained Commons recording. Commit dcb2a2b combines related resources across 31 entries while keeping distinct appearances separate. [TALKS-SOURCES.md](TALKS-SOURCES.md) retains event and upload evidence, precision limits and exclusions. Open Mic identifies the owner's second talk without an invented chapter timestamp. One initial check found that this generator does not automatically create heading anchors; explicit section IDs now make both topic links work. Independent inventory review and a final rendered-page check pass.

The combined local publish recipe and seven vanity checks pass. Axe reports no violations across the six affected contact/talk pages; keyboard skips, 320px/double-font reflow and JS-disabled contact/navigation pass. The updated final talks page passed the same browser checks again. All three feed bytes remain identical to the already validated 594cf0d output. Private provenance stays out of deployment, and the owner's pre-existing untracked document remains unchanged.

The owner-correction deployment 8e3acb1 completed successfully at immutable preview https://d822a93e.da-rio-hn.pages.dev. [Site checks](https://github.com/darccio/darccio/actions/runs/38149888909) and [Vanity after deploy](https://github.com/darccio/darccio/actions/runs/38149888933) passed. Production retained all supplied recordings and 15 decks, public cc@dario.cat contact without the email decoder, and Sifa alongside LinkedIn. This supersedes the preceding correction batch's deployment-pending notes; the domain-wide obfuscation setting itself was unchanged.

The microformats batch passes the exact configured publish command, staged vanity checks and the new independent parser. Current output has 33 public localized articles, 30 exact publication instants and three deliberately partial historical dates; the draft keeps its live URL but stays outside entries, archives and RSS. Home/about representative identity, all three archives, ten verified events, six slash entries, eight slash-index updates and five projects parse against public source facts. The source adapter executes no plugins or templates and processes no CV data. Both tooling modules pass race/shuffle tests and vet; workflow syntax passes actionlint. Rejected validation preserves the previous deployment in an integration fixture.

Chromium finds zero axe violations on 18 representative pages, and keyboard skip links, 320px/double-font reflow and no-JS contact/archive checks pass. An existing Synology code-scrolling defect is fixed separately in CSS; independent desktop/narrow/double-font checks also retain identical article HTML and copied code bytes for Synology and highlighted Ruby/Go examples. The unchanged CV hash is 6a5f6e71fbbfc544994619fd089c82d7dd9d589948001286aa049378b5822ff0. No locality, new client script, PDS record or Zas change is added in this batch. Live acceptance remains the exact-SHA Cloudflare check, Site checks and post-deploy vanity workflow after push; markup completion does not establish Mastodon account verification.


Cloudflare deployed 28bd9ba00a9a142cb395181584b3d6c56d7c8c99 successfully at https://e3892fe8.da-rio-hn.pages.dev. [Site checks](https://github.com/darccio/darccio/actions/runs/38152081317) and [Vanity after deploy](https://github.com/darccio/darccio/actions/runs/38152081265) passed. Independent live checks cover 64 canonical HTML URLs and complete matching sitemap coverage, 33/9/12 archive entries, ten dated talks, six slash entries, eight dated index entries and five projects. The deployed stylesheet and direct Markdown/policy resources match built bytes. All three feed bytes match two consecutive final builds and again validate as RSS with the W3C service.

Live representative-card parsing selects Dario on home/about with the exact root UID/URL, local photo and public contact; all seven head profiles exist, and both WebP images match source bytes and MIME. Mastodon's public account API already reports the dario.cat Website field verified_at 2023-01-28T16:47:11.620+00:00, so no owner re-save is required. Cloudflare's pre-existing analytics beacon remains an edge injection; this batch adds no source JavaScript.

A full live article-content comparison detected the edge's existing HTTP-to-HTTPS upgrades on five article links and obfuscation of the historical Deciwave robot address. The scheme upgrades preserve the same destinations and need no source changes. Commit d21a476 wraps only that original address in the same documented email_off opt-out used by public contact. Protected local assembly preserves its original rendered body/mailto and passes the parser; its Git fallback update date may legitimately change, while publication dates and GUIDs stay fixed. The final follow-up deployment must confirm that page's plain address and absence of an email decoder. Live comparison retains the actual fetched responses and records edge transformations explicitly; it does not weaken the pre-deploy source validator.

## Current account and repository conditions

Cloudflare Pages Git integration deploys pushes to main. The owner reports build command make -f .zas/Makefile publish, output .zas/deploy, blank root directory and enabled build comments. The guarded command is already configured; committed pin/assembly changes need no dashboard update. The public GitHub integration check is named Cloudflare Pages from app cloudflare-workers-and-pages; no GitHub deployment records or tracked deployment workflow existed in the baseline.

Public .cat RDAP reports expiry on 1 June 2027 and delegationSigned:false. Auto-renew, valid payment, transfer lock and independent monitor configuration still need actual account evidence. T00 also observed edge email obfuscation, which the local contact source cannot disable.

The owner has now explicitly authorized Mergo and darna docs/workflow PRs for T24, in addition to this site and Zas. Other project repositories remain outside this implementation batch. Existing docs/go-vanity-url.md remains untracked and unchanged (SHA-256 b1fca26739cf9d022923a2c2ebe7445b60ac10f8f89126ca7b654e946201de89). Zas work used isolated branches; the feature batch starts at upstream master fcb82e4, including the merged helpers, while the user's checkout is preserved.

## Implementation queue

The foundations, identity, metadata, feeds, Markdown/context surfaces, HTTP negotiation and source-owned docs are merged and production-verified. T26 publication verification is live after PR #12. The remaining document publishing, backlinks, independent workflow and consistency gate are implemented in site PR #14. Six supplementary pages are in site PR #13; Mergo contact and Zas release instructions are separate PRs. Exact head CI and Pages preview checks pass.

Historical approval selections, registrar/DNSSEC/external alert evidence, three missing publication instants, CV processing permission and personal defaults remain owner-dependent. T06/T23 are intentionally deferred. Release tagging follows the Zas documentation merge and its required checks. No new configuration values or account credentials are being requested.

## T20 delivery · 2026-10-11

Site PR #5 contains signed feature commit 9366c92 and the reviewed .zas/llms.yml corpus manifest. The exact committed tree reproduces all 156 expected deploy-file hashes. GitHub Site checks and Cloudflare Pages succeed; immutable preview 6199ca42 passes all 111 context/twin/feed/head HTTPS checks, expected plain-text/Markdown MIME and one-hour context caching, matching static content policy and all seven vanity packages on both URL forms. Existing article bodies, source URLs, raw vanity fragments, feeds and twins remain unchanged. CV data, drafts and operational documents are excluded; per-project contexts remain T24. Production verification follows the owner merge, and T22 is ready next.

## T20/T21 production and post-deploy follow-up · 2026-10-11

The owner merged PR #5 as c624d6c. The exact main commit has successful Cloudflare Pages and Site checks. All 111 production context/twin/feed/head checks match the clean build, and direct live vanity checks pass all seven packages. T21 was already merged via Zas PR #113 and is complete; its rendered twins have correct direct Markdown MIME and discovery. T22 adds Accept negotiation at HTML URLs.

Automatic post-deploy run 38157663720 failed before the live check because the otherwise valid Cloudflare suite supplied an explicit null branch. The narrow gate correction accepts this observed metadata while preserving explicit-branch, current-main, SHA, provider, project and success checks. Exact predicate mutation checks and independent review pass; the owner merged PR #6 as 4f237e9, and its exact-SHA Cloudflare Pages deployment, Site checks and automatic Vanity after deploy run all succeeded. No site generator, content or vanity metadata changes are needed.

## Remaining scope and T22 · 2026-10-11

After the T20 and post-deploy-gate merges, 18 of 33 task IDs retain open work: ten main items (T22 and T24–T32), six partially completed items (T01, T03, T04, T09, T17, T19), and two optional deferred items (T06, T23). T09's local execution scope includes the seven supplementary slash pages. The partials cover account operations, external project identity, three unresolved publication instants, supplementary pages, restricted CV/unsupported event markup and source-code metadata. After T22 merges, nine main tasks plus those follow-ups and optional items remain.

T22 uses Pages' automatically deployed advanced-mode Worker output with the existing Git build settings. The guarded assembler builds an exact inventory of final public HTML and rendered twins, preserving all raw import fragments and excluding CV/operations. The server adapter validates fetched twin bytes, negotiates quality weights, preserves asset routing, distinguishes variant validators and disables shared variant storage. Static assets bypass the Worker; future nested project documentation remains eligible. Node tests run only in development/CI; production assembly remains Go-only. Local and preview acceptance are recorded in tasks/T22.md before handoff.

T22 site PR #7 contains signed feature commit 47f7ba0. GitHub CI and Cloudflare Pages succeed for the exact SHA; immutable preview ac8a9eaf passes 283 HTTPS checks across all public variants, headers, conditionals, direct resources and all seven import routes. The independent live Go checker passes too. The existing 156 public file hashes remain unchanged, and rollback removes the two server artifacts. Cloudflare removes HTML validators while retaining Markdown validators; variant storage stays disabled and cross-format conditions remain safe. Detailed acceptance and hosting observations are in tasks/T22.md. Production verification follows merge.


## T24 implementation · 2026-10-11

The owner authorized Mergo and darna source PRs. Source docs and dispatch workflows are open in Mergo #321, Zas #116 and darna #15, with immutable source pins recorded in `.zas/projects.yml`. The site now assembles six English docs pages, declared portable Markdown twins, original project licenses, navigation/provenance and per-project agent context without executing fetched content. Exact vanity metadata, the five remaining raw fragments, feeds and existing article bodies are preserved. Manifest-driven SoftwareSourceCode completes T19's deferred project records for these three projects.

The source receiver proposes manifest-only pin PRs after a complete guarded build; merging those PRs triggers the existing Pages Git integration. Source SITE_DISPATCH_TOKEN configuration and site Actions PR creation remain account setup steps, so live dispatch automation awaits that evidence. All implementation, offline gh handoff tests, local build/format/accessibility checks and source CI are complete; site PR and exact-SHA preview acceptance follow. T25's Mergo guides remain next. See tasks/T24.md for detailed validation and rollback.


T24 site PR #8 contains signed feature commit bcf1c8c. Its exact-SHA GitHub checks and Cloudflare Pages succeed; immutable preview d7365698 passes 310 HTTPS variant/resource/import requests plus six path-specific docs metadata checks and the shared Go vanity checker. All 177 output hashes reproduce across two committed-tree builds, and an actual rejected docs build preserves all prior bytes. All three source PRs have passing CI. Source merges, the site production merge and three absent SITE_DISPATCH_TOKEN secrets remain owner actions; Actions PR creation is already enabled and read-only verified; all implementation and preview acceptance are complete.


## T25 Mergo guides

T25 is implemented for review in [Mergo PR #322](https://github.com/darccio/mergo/pull/322), signed source commit `8c092c379d3602a60b22605ee577d8e70aea9860`, and [site PR #9](https://github.com/darccio/darccio/pull/9), signed implementation commit `f9bcbd6`. The site selects five English guides from that immutable commit: overview, all eight options, transformers, migration and FAQ. Source-owned executable examples pass on current source and released v1.0.2; migration fixtures reproduce the original path error and verify both direct adoption and the temporary legacy pin. Source CI is green.

The 185-file guarded build preserves all seven vanity contracts, all feeds/posts and other project output. Five HTML/Markdown pairs, project context indexes, local anchors, licenses, source dates and accessible layouts pass independent checks. T24 site PR #8 merged as `8b3dd43`; all three source `SITE_DISPATCH_TOKEN` names are present. Production T25 completion follows merging its source and site PRs; a successful token-backed dispatch remains a separate live check. See tasks/T25.md for precise verification and rollout evidence.

Both T25 PRs have passing GitHub checks. Cloudflare's exact-feature preview https://80ed6769.da-rio-hn.pages.dev/mergo/ passes 66 focused HTTPS requests plus the independent seven-package live import checker. The exact committed build reproduces all 185 hashes. Merge Mergo PR #322 before site PR #9 to publish the guides; production verification follows those merges.

## T25 production and T24 dispatch closure · 2026-10-11

The owner merged Mergo #322 as `ff8ae09` and site #9 as `9a3af08`.
The exact site commit's Pages, CI and automatic post-deploy vanity checks pass;
the production domain passes all 66 focused Mergo requests. T25 is deployed.

The token-backed Mergo dispatch initially ran before the site's new guide
whitelist merged and safely rejected its newly linked migration page. The
[receiver retry](https://github.com/darccio/darccio/actions/runs/38169148268)
now passes and opens [source pin PR #10](https://github.com/darccio/darccio/pull/10),
confirming T24's dispatch/build/atomic proposal path. Merging that separate PR
adopts the source's merged commit; the deployed docs already use the verified
immutable feature commit.

## T26 publication preparation · 2026-10-11

The T26 Go bootstrap, exact reviewed publication metadata, default-preview manual
workflow and atomic two-output PR handoff are implemented. Public identity/schema
checks and the real unauthenticated preview pass; the account still has only
Rizoma. Mocked Go tests, race/shuffle, vet, ten offline GitHub handoff tests,
workflow lint and the full existing site suite pass. The guarded build preserves
all 185 output hashes except its new verification MIME rule. An isolated fixture
confirms extensionless URI preservation and internal-state exclusion.

The owner chose to prepare the PR and configure `ATP_HANDLE`/`ATP_APP_PASSWORD`
later. No authentication, PDS write or placeholder verifier is included. T26
activation remains pending: merge the implementation, add the site secrets,
explicitly apply the reviewed publication metadata, then merge its generated
verification PR and verify the live response. T27 document publishing remains
next; historical backfill needs the separately reviewed T30 allowlist.

[T26 site PR #11](https://github.com/darccio/darccio/pull/11) contains signed,
GitHub-verified feature commit `30014f9`. Both Site checks runs and the exact-SHA
Pages deployment pass. Immutable preview `d78c64fa` passes the 66 focused HTTPS
checks and independent seven-package live checker; the committed tree reproduces
all 185 reviewed output hashes. The owner can review the prepared implementation
without configuring credentials; account activation remains a later step.

## Remaining authorized implementation · 2026-10-11

All independent local implementation is prepared in signed atomic commits and focused PRs. Required subagent implementation/review dispatches covered every item worked here. The publisher, offline backlink assembler, separate workflow and independent consistency gate are implemented and tested; detailed task receipts and operator instructions identify post-merge/live checks. No historical records have been written, and no secrets or private recovery snapshots enter Git or artifacts.

The configured make publish recipe passes. A temporary approval fixture exercises all 23 candidates without live writes. A deliberate metadata mismatch preserves all 186 previous deploy files, and deletion tests prove fresh CI cannot recreate a document after approval and verified-map removal. The supplementary branch has six page commits and two deterministic guarded builds with all original vanity tuples preserved. Mergo, Zas and both site PRs have passing checks; the exact-head preview evidence follows below.

The existing original working directory remains unchanged, including untracked docs/go-vanity-url.md. Current monitoring and incomplete-date evidence is in DOMAIN-OPERATIONS.md and publication-provenance.md. Release preparation passes GoReleaser check/snapshot plus Go/lint checks, but v0.0.9 tagging waits for the release-doc PR to merge. No account controls, DNS settings, release tags or archive approval selections were changed.

The initial exact-head previews pass focused HTTPS format checks: supplementary 2b78b81 at https://891ff1db.da-rio-hn.pages.dev (33 requests) and publishing 74495ca at https://f2796453.da-rio-hn.pages.dev (29 requests). Both also pass the independent fourteen-route Go vanity checker. Checks cover actual Markdown bytes/negotiation, production canonical URLs, empty document backlinks, RSS XML, context bodies and verification MIME. The two branches merge without conflicts and pass the combined protected consistency build. A browser-discovered long-subject overflow is corrected with a scoped changelog list and wrapping rule plus a generation assertion. All six supplementary pages pass axe with zero violations, keyboard skip navigation, 320px reflow, doubled text size and no-JS checks in the combined build. Markdown twins and context entries remain complete, with fifty changelog commits and matching sitemap/update metadata.

## Merges, production and owner handoff · 2026-10-12

Site #13 merged as `2c07c9`, site #14 as `1fb265c`, Mergo #323 as `bb080830`, and Zas #117 as `e569d7d`. Current site main has successful CI, Pages and post-deploy vanity results, with a fresh fourteen-variant live checker pass. The first [Standard.site documents automatic run](https://github.com/darccio/darccio/actions/runs/38210510461) passes deployment verification, empty apply and no-change proposal. The committed approval list remains empty, so no article publication is inferred.

[Zas v0.0.9](https://github.com/darccio/zas/releases/tag/v0.0.9) is published from the signed tag on merged master e569d7d. Required local/remote checks and six snapshot archives pass; [release workflow 38210863278](https://github.com/darccio/zas/actions/runs/38210863278) succeeds with six published archives/checksums. Release-pin adoption remains a separately verified site change.

The scheduled monitoring backstop is now observed successfully. The registrar expiry remains 2027-06-01 and the delegation remains unsigned; auto-renew, transfer lock, independent checks and received alerts remain owner account work. Other owner decisions are the reviewed 23-candidate selection, three unresolved publication instants, a scoped CV-processing exception if desired, and personal defaults. Optional T06/T23 remain deferred. No account settings, archive approvals or CV data were changed. Owner guidance begins with registrar renewal/lock, then Cloudflare DS publication and external alert setup; screenshots, invoices and private monitor keys stay outside Git.

## Registrar owner confirmation · 2026-10-12

The owner confirms auto-renew and transfer lock enabled, billing/contact details checked, and registrar expiry `2028-06-01`. These account checks are recorded without private billing/contact material. Fresh .cat RDAP and registry WHOIS confirm `2028-06-01T07:05:04.361Z`; registrar RDAP independently confirms the paid term. Item 1 is complete, superseding the initially older registry response. DNSSEC and external alerts remain the next owner steps. Site release-pin PR #15 is merged as `92d4956`.

## DNSSEC DS handoff · 2026-10-12

Cloudflare signing is enabled, with public DNSKEY records and verified KSK key tag `2371`/algorithm `13`. The SHA-256 digest exactly matches the supplied DS. Public resolver checks show no parent DS and an unsigned answer; registrar DS publication and end-to-end chain validation remain next. The open operational receipt includes the exact public fields and records the transport limitations without claiming direct authoritative verification.

## DNSSEC completion · 2026-10-12

The owner saved the exact Cloudflare KSK in SWPanel's domain DNSKEY form. Registry and registrar RDAP now report `delegationSigned:true`. Verification at **08:17:15 UTC** confirms the matching parent DS through Google and Cloudflare, authenticated DS/A/DNSKEY answers, two fully validated `delv` checks and all fourteen passing live vanity routes. This supersedes the earlier unsigned state; owner items 1 and 2 are complete. T01 remains open for independent monitoring and received failure/recovery/expiry notices. The owner selected an existing monitoring service; its provider and alert setup remain to be confirmed. See DOMAIN-OPERATIONS.md for exact public key data and timestamps.

## Datadog Terraform monitoring · 2026-10-12

The owner selected Datadog EU1, Terraform and `cc@dario.cat` alerts. The prepared `.infra/datadog` module shares the Go vanity baseline and defines managed-location import, authenticated DNS, certificate and registry-expiry tests. A separate optional wrong-metadata probe can verify actual failure/recovery delivery without changing production. The definition starts paused and keeps account credentials, state, local variables and plans outside Git. Account apply, hosted assertion verification, activation and received notices remain open; this preparation does not close T01. Detailed commands and schedule/usage are in the module README.

## Reduced monitoring budget · 2026-10-12

PR #17 merged as `20bfd48`. The owner then requested about $5/month and explicitly chose Mergo/darna checks every thirty minutes, other imports every two hours, and daily auxiliary tests. That schedule retains all thirteen test resources and exact URL/metadata assertions while reducing scheduled runs from 46,890 to 9,540 per thirty days. Monthly estimates at the published annual-agreement rate are $4.77 for thirty days and $4.93 for thirty-one days; the account-specific rate is configurable. Independent detection for the other five imports is now on the chosen two-hour cadence; the full GitHub backstop remains thirty minutes. Account apply and received notices remain pending.

## Datadog partial apply and assertion correction · 2026-10-12

The owner merged PR #18 and resolved account authentication. Their apply created
the certificate, two DNSSEC and three registry-expiry tests, then Datadog
rejected all seven vanity tests with `Global variable invalid names`. The
rejected name exactly matches compressed HTML entity data enclosed in literal
double braces inside the generated parser bundle. This is an assertion-build
defect; the text is public parser data, not account material.

The follow-up escapes literal variable openers in JavaScript string data while
preserving decoded values. Restricted-runtime fixtures and mocked plans cover
the generated bundle, all seven route-injected assertions and the temporary
failure/recovery probe. The operator guide retains the six successful resources
and requires a fresh plan after updating, without deleting state or reusing the
failed saved plan. Hosted execution, activation and received notices remain
owner work; creation alone does not establish active monitoring.

## Datadog assertion-size correction · 2026-10-12

After PR #19 resolved variable interpolation, the owner captured seven HTTP 400
invalid-JavaScript-assertion responses through OpenTofu's JSON output. Temporary
paused probes accepted a tiny multistep script, rejected both the 94,833-byte
parser and equal-sized trivial padding, and accepted fixed sizes through 32,768
bytes before rejecting 65,536 bytes. A finer probe accepted through 51,200 bytes
and rejected 60,000 bytes. Every temporary accepted test was removed;
the six state-managed auxiliary tests remain intact.

The next correction compiles checksum-pinned parse5 TypeScript, specializes
unused diagnostics/location tracking and shortens reviewed internal property
names. It retains the existing HTML grammar, full entity decoding and licenses,
with additional script/CDATA fixtures and a 49,000-byte final-assertion budget.
The measured bounds do not establish Datadog's exact size limit. The owner then
confirmed API acceptance by creating a paused HTTP test with the actual
48,858-byte Mergo replacement; both it and the tiny multistep control were
cleaned up. The tested artifact is SHA256
`de41f1a14fb161269505561fd7d467eacfe664b202fca15e3404f03451d11987`.
State-managed creation, managed-location execution and received notices remain
the next operational gates.

## Datadog multistep account quota · 2026-10-12

PR #20 merged as `4432d44`. The owner's next apply created five vanity tests
and hit the account's five-multistep-test quota on Mergo and Cocoon. The
owner-selected two-group layout keeps all fourteen URL checks: Mergo/Cocoon have
four steps every thirty minutes; the other five have ten steps every two hours.
All steps remain critical while allowing subsequent steps to execute. The
monthly schedule estimate remains $4.77 for thirty days at the configured rate.

Moved blocks reuse the confirmed Darna and Xdr resource IDs. The expected
partial-state migration updates those two tests in place and removes three
redundant vanity tests, with no additions and the six auxiliary tests unchanged.
Actual account migration, managed-location execution and received alerts remain
owner checks. No account resources were changed by the agent.
The actual provider's offline migration plan confirms zero creates, two updates,
three redundant-test deletions, both moved IDs retained and six auxiliary no-ops.
Its fake prior state is generated from pinned pre-group configuration; validation
and refresh are disabled against a loopback endpoint, with no real state reads.

## Successful grouped monitoring apply · 2026-10-12

PR #21 merged as `0e45866`. The owner applied the expected zero-addition,
two-update, three-removal migration successfully. Priority and other retain
`kv2-z2c-77h` and `m4f-5g8-vxh`; all six auxiliary IDs are unchanged, and the
temporary notification probe is absent. The zero scheduled-run/cost outputs
confirm the module remains disabled and its eight resources paused.
Grouped creation and state migration are complete. Hosted on-demand execution,
schedule activation and received failure/recovery/expiry notices remain open.

The owner then reports all eight paused on-demand tests green. Hosted execution
is now verified for both groups and the six auxiliaries. Scheduling remains
disabled; activation through local `enabled = true`, observed scheduled
successes and received failure/recovery/expiry notices remain required for T01.

The owner subsequently activated all eight tests with zero additions, eight
updates and zero destructions, preserving all IDs. Active usage outputs are
9,540/9,858 requests per thirty/thirty-one days and $4.77/$4.929 at the configured
rate. The temporary notification probe is absent. Successful API activation
is confirmed; observed scheduled runs and received real failure/recovery/expiry
notices remain open.

The first notification-probe saved-plan apply hit OpenTofu 1.13.1's input-value
validation. Its plan used CLI overrides but apply omitted them. Provider-free
reproduction confirms that repeating both exact flags fixes creation, recovery
and cleanup, including raw Boolean/string differences. The operator guide is
corrected; no account resource change or notification receipt is inferred.

The owner then applied the matching probe overrides successfully and confirms
receiving its failure notification. Failure-email delivery is now verified by
the owner. Recovery delivery on the same probe, its cleanup, scheduled
production successes and expiry-notification simulations remain pending.

The owner explicitly confirms receipt of the same probe's recovery email.
Both real failure and recovery delivery are verified. Probe cleanup, scheduled
production-success evidence and three expiry-notification simulations remain
required to complete T01.

The owner confirms cleanup and the eight production tests remaining live.
The temporary probe lifecycle is complete. Scheduled production passes and
received simulations from the three registry-expiry tests remain the final
T01 checks; no scheduled result or expiry receipt has yet been supplied.

The owner now explicitly confirms passing **Scheduled** runs for all eight
production tests. Scheduled execution, real failure/recovery email delivery
and probe cleanup are verified. T01 remains open only for received notification
simulations from the three registry-expiry tests; the owner is locating their
simulation control.

## T01 completion · 2026-10-12

The owner confirms sending and receiving the alert simulation from each
90/60/30-day registry-expiry test. These are notification-routing receipts;
they do not imply that an actual expiry threshold was crossed. The earlier
failure/recovery emails came from the real temporary wrong-metadata probe,
which is now removed.

T01 is complete with the registrar, DNSSEC, scheduled GitHub backstop, eight
live Datadog tests, scheduled passes and all notification receipts documented
in DOMAIN-OPERATIONS.md. This supersedes the pending states in the chronology.
The monthly estimates remain $4.77/$4.929 for thirty/thirty-one days at the
configured rate, excluding retries and manual/probe runs. The next owner step
is review of the 23 Standard.site candidates in standard-site-backfill.md;
the committed approval list remains empty and no article publication is
inferred. Three missing publication instants, optional CV permission and
personal defaults remain separate owner decisions.

The operator instructions repeat exact OpenTofu probe inputs at plan and
saved-plan apply in `c51256f`, and locate the test-editor simulation control in
`47d13ab`. The first correction was verified through provider-free creation,
recovery and cleanup plans; the second matches the official UI screenshots
and the owner's successful sends. Documentation review and whitespace checks
pass; these commits change no account settings or production test schedules.

## T04 publication evidence recovery · 2026-10-12

The owner requested T04 before T30 and confirmed Europe/Madrid for Ruby/Go's
existing April 5, 2022 clock value. Its date is completed as
`2022-04-05T22:42:00+02:00`. The original Thinkship English RSS now provides
the publication field for post 848: `2013-07-02T08:00:39Z`, emitted as the
equivalent `2013-07-02T10:00:39+02:00`. Both posts regain RSS and candidate
eligibility; the Standard.site approval list is unchanged and empty.

The original Spanish article, post 829, independently confirms July 2, 2013.
Its known day and archives are corrected with the earlier value retained as
`date_created_original`. The owner then instructs assuming the same timestamp
as the English translation; adopt that full value and record its clock time
as an owner-approved assumption rather than independently recovered history.
Its syndication exclusions are removed. All 33 public posts now have explicit
publication dates. Source links, response hashes and the owner instructions
are recorded in publication-provenance.md; follow-up merge/deploy is pending.
The three article bodies remain unchanged. No publication time is inferred
from a Git commit or archive capture; the Spanish clock time is the explicit
owner-adopted assumption described above.

The built metadata, microformats and JSON-LD checks pass for all 33 public
publication instants. RSS contains 33 combined, nine English and twelve Spanish
items; all seven vanity contracts are preserved. The refreshed offline T30
inventory has 26 candidates (11 Catalan, eight Spanish and seven English) and
eight exclusions (seven before 2013 and one draft). Both the reviewed approval
list and verified URI map remain empty. Source dates, titles, summaries and
language values in the review inventory match the candidate planner.

A temporary external fixture exercises all 26 candidate record plans without
network calls or changes to committed approvals. Their record keys are unique,
including both translations at the shared timestamp, and the three corrected
record dates agree with source. The production consistency gate correctly
rejects this fixture plan against the real empty approval list; the ordinary
zero-document build passes. No recovery state or public URI entry is written.

## T04 production completion and T30 proposal · 2026-10-12

PR #23 merged as `bcd015c`; its exact-main Pages, Site checks, post-deploy vanity
and empty-allowlist Standard.site workflow checks succeed. Fresh production
checks verify the three corrected article dates across Open Graph, h-entry and
JSON-LD, with RSS counts of 33 combined, nine English and twelve Spanish.
T04 is complete. The owner requested T30 next.

Fresh public GETs verify the dario.cat publication record and its exact live
well-known backlink. The account currently has zero document records. The
unmerged proposal selects all 26 eligible path/timestamp pairs and includes a
full offline JSON review artifact under `.tools/atproto/review/`. Its 26 planned
creates have unique keys and 72,442 UTF-8 plaintext bytes; the complete
source/HTML/RSS/JSON-LD/record consistency check passes. No PDS login/write or
verified-map change occurs during preparation.

Merging this specifically scoped approval PR authorizes the selected full-text
publication and triggers the existing post-deploy publisher. The workflow then
proposes publicly verified article URIs in a separate PR. T30 remains pending
owner approval, successful record readback, deployed verification links, three
reader samples and a zero-write online rerun. Records for other publications
remain outside the operation; optional CV/defaults work remains separate.

## T30 completion · 2026-10-12

The owner approved all 26 full-text archive records by merging
[PR #24](https://github.com/darccio/darccio/pull/24) as `5fc7634`. The
[post-deploy apply](https://github.com/darccio/darccio/actions/runs/38296784023)
contains 26 successful creates matching the exact approved preview. Public
inventory and individual readbacks confirm 11 Catalan, eight Spanish and seven
English documents, with 72,442 UTF-8 text bytes. Three fresh PDSls samples pass.
The [repeated online apply](https://github.com/darccio/darccio/actions/runs/38297070735)
contains 26 unchanged records and zero creates, updates or deletes; URIs, CIDs
and values remain identical in fresh public reads.

The owner merged the public-map-only
[PR #25](https://github.com/darccio/darccio/pull/25) as `87b6dcd`. Exact-main
production Pages, Site checks and the post-deploy vanity gate pass. All 26
article verification links match public records. All 34 checked canonical URLs
(33 articles plus the retained draft URL) stay stable; selected metadata and
article plaintext agree, excluded articles have no document link, and all
fourteen vanity URL variants pass. The backlink-only automatic workflow skips
publish/propose; that skip is separate from the successful online idempotence
run. A preview asset-upload rate limit was resolved by retrying the same commit.

T30 is complete. The [task receipt](tasks/T30.md) records the checks and viewer
limits; the [identity snapshot](../../.tools/atproto/review/backfill-verification.json)
retains the public URIs, CIDs and payload hashes. Production's edge-injected
Cloudflare beacon prevents a raw whole-page/local byte-equality assertion, but
article content and metadata agree. The original approval preview is retained
as historical evidence. CV permission and personal defaults remain owner work;
optional T06/T23 remain deferred.

## CV email correction and design review

The owner merged [PR #27](https://github.com/darccio/darccio/pull/27) as
`4d4f481`. Exact-commit Pages, Site checks and post-deploy vanity checks succeed.
The immutable deployment serves the new English resume; the production
`/CV/MAC.json` bytes match its preserved source. Custom-domain verification
found that `html/template` removed literal `email_off` comments from the new
resume template, allowing Cloudflare to hide its email. Existing contact pages
already retain their exclusions.

The correction emits only the two reviewed static comment boundaries as trusted
template data in `6ff0f56`. A regression test checks the exact protected email fragment;
the guarded generated HTML retains both boundaries. Generate, full repository
tests and the seven-package vanity check pass. Production email acceptance
follows deployment of this correction. No CV data or processing scope changes.

The design work follows the owner's six units in order: foundations, localized
navigation, talks archive, article language tags, homepage hierarchy/contact
and project link labels. T09's remaining personal defaults are deferred. The
owner declined the optional homepage project-proof line and confirmed Catalan
for all five previously untagged talks; no language question remains open.
Only this progress file may change among files marked `publish: false`.

The baseline has 31 talks (10 engineering and 21 culture), 46 external hrefs and
ten verified event tuples. Before screenshots at 1440×900 and 390×844 and the
exact href/event snapshot were captured before design edits. User-supplied
Appendices A/B replace the inaccessible design-system artifact; tokens remain
verbatim. Fonts come from the pinned Fontsource packages with original OFL
licenses, verified npm integrity and individual file checksums.

### Design foundations

Committed as `51c2f67`.

Self-hosted Newsreader 400/500/600 and 400 italic and IBM Plex Mono 400/500
replace the system-font foundation. Both pinned Fontsource 5.3.0 packages retain
their OFL licenses and checksums. `tokens.css` equals Appendix A byte-for-byte;
the exact Appendix B is the stylesheet suffix. The missing hairline shorthand
is supplied only in `styles.css`, preserving the tokens.

The inline style block is removed. The centered column, theme colors, serif
headings, paragraph rhythm, mono code/navigation, focus outline and wrapping
use the supplied tokens. Generate, full repository tests and vanity checks pass.
All 36 browser scenarios (nine requested pages, two widths and two OS themes)
have zero axe violations and visible keyboard focus. Every scenario passes
320px reflow at normal and 200% text, with both font families loaded locally.
No theme JavaScript or external font request is added. Later units own the
header/current-navigation and catalogue changes; their acceptance is still pending.

### Localized layout

Committed as `1fa425d`.

Section metadata resolves through Zas page and nearest-directory config;
English/Spanish defaults retain both their language and selected posts section.
Navigation, skip link, article chrome and footer now use the ca/en/es message
catalogue. The Catalan archive and its nav label both say Escrits. The current
section uses an explicit aria-current attribute and a token-colored 2px underline.
The uniform header uses the same 96px avatar and name everywhere; its compact
card is omitted only as a microformats root where the body already supplies the
representative card, preserving the homepage/about representative identity.

Generate, full tests, vanity and microformats checks pass. All 36 browser
scenarios pass axe, keyboard focus and 320px normal/200% reflow. A targeted
nine-route check confirms selected sections, localized chrome, 44px nav targets
and a single-row Catalan menu at 390px. Checks use local generated output;
custom-domain deployment remains pending the review PR.

### Talks archive

Committed as `70d44fb`.

All 31 talks use the supplied archive-row pattern in two labelled sections
(10 engineering, 21 culture). Dates retain their original full/month precision;
15 undated rows remain undated. Language badges are decorative, with full
Catalan spoken labels outside each event name. The owner-confirmed five
languages are now Catalan. Resources retain their destinations and gain the
complete talk title as accessible context; qualifiers and existing prose move
to notes. Only the requested Machine Learning/Creació dated pair changes order.

Generate, full tests, vanity and microformats checks pass. The before/after
built-page comparison has identical 46 external hrefs and ten name/start/
location/url event tuples. Browser measurements find a 4px title-to-detail gap,
less than one third of the gap between rows, and all resource targets at least
24px tall. Talks passes axe, visible keyboard focus and 320px normal/200% reflow
in both OS themes at 1440px and 390px. Before/after screenshots are retained for
the PR. Deployment and the final full-page matrix remain pending.

### Article language tags

Committed as `50cb1d6`.

The 35/9/12 archive rows replace all 56 bracketed codes with decorative badges
and full spoken language names localized to each index, with explicit language
on the spoken span even when its row is in another language. Titles, dates,
URLs, microformats and one-line source rows are preserved. Archive dates use
the mono small/muted tokens, including external citations.

The existing guard required literal small-element bracket codes. Its contract
and fixture now enforce the accessible badge/label pair instead; negative
checks reject missing or hidden badges, incorrect language/name, labels inside
p-name and spoken labels hidden from assistive technology. The original
URL/date/entry/citation checks remain intact. Generate, full tests and vanity
pass. All twelve archive browser scenarios pass axe, keyboard focus and 320px
normal/200% reflow. Production acceptance remains pending deployment.

### Homepage paths and contact

Committed as `53143d9`.

The homepage now places its 96px photo beside the name, retains the existing
bio note verbatim, shows Per on començar next, and then the compact shared
contact row. Duplicate prose profile links are removed; each profile occurs
once in main, with rel=me retained. The newsletter label is Rizoma. Speaker
Deck moves into the shared partial without adding a new h-card URL property.
The photo retains its canonical asset URL; all parsed representative h-card
properties are exactly unchanged. The optional projects line stays omitted.

Generate, full tests, vanity and microformats checks pass. The whole starting
list fits within 636px on a 1440×900 viewport. Contact links are 44px tall with
16px gaps in both directions. All twelve home/about/contact browser scenarios
pass axe, visible keyboard focus and 320px normal/200% reflow. Head identity
links and JSON-LD remain intentionally separate from the visible contact row;
custom-domain acceptance remains pending the review PR.

### Project resource names

Committed as `3766154`.

The six repeated repository/API links now append their project name as spoken
context, while visible labels and destinations stay unchanged. All five parsed
h-item objects and the complete project href list equal the pre-change build.
Generate, full tests, vanity and microformats checks pass. The final 36 browser
scenarios (nine pages, light/dark, 1440px/390px) have zero axe violations,
working focused skip links, visible link focus and no 320px overflow at normal
or 200% text. Every prior HTML route remains present. All seven vanity contracts,
the raw fragments and the CV JSON are preserved; talk links/events and the
representative homepage card remain unchanged. The final screenshot/commit
receipt and hosted preview follow this unit; production follows owner merge.

### Foundation reflow follow-up

Committed as `da02de2`.

The wider check found 200% text overflow on the older Deciwave video embed and
three Mergo/Zas documentation tables. Iframes now respect their container;
tables and cells wrap within the column without changing their semantic HTML
or content. The top 5px rule uses the requested accent token directly.

Generate, full tests and vanity pass. All 74 styled HTML pages pass 320px
reflow at normal and 200% text. Five raw vanity fragments are excluded from
browser styling checks and remain byte-identical; four intentionally redirect
to their existing pkg.go.dev destinations. The final requested 36-scenario
axe/keyboard/font/reflow matrix was rerun with this final CSS and passes.
Final screenshots and all catalogue/home measurements were recaptured.
No article facts, embedded destinations, docs sources or client scripts change.

### Design review receipt

The six requested units are committed in order: `51c2f67` foundations,
`1fa425d` localized layout, `70d44fb` talks, `50cb1d6` article tags, `53143d9`
home/contact and `3766154` project labels. The isolated CV exclusion fix is
`6ff0f56`; the measured media/table reflow correction is `da02de2`. All commits
are signed and passed darna staging/atomicity validation without bypassing hooks.

[Review evidence](../../.github/review/site-design/verification.json) records
source/CSS hashes, tool versions, exact before/after talk tuples and hrefs,
measurements and check limits. Four before/after screenshots at 1440×900 and
390×844 and the 36-scenario/74-page browser reports live alongside it for the
PR, under .github so they are excluded from the published site. Independent
review found no material issues. The five talk languages are confirmed Catalan;
the optional homepage proof line is omitted. No owner question remains for
this batch. The remaining T09/defaults work stays deferred.

Local acceptance is complete. Hosted preview and exact-commit CI checks follow
push; custom-domain acceptance and the CV email correction require owner merge.

The screenshot/check receipt is committed as `917fc41`. Review is open in
[PR #28](https://github.com/darccio/darccio/pull/28); its
[commit checks](https://github.com/darccio/darccio/pull/28/checks) record hosted
CI and Pages outcomes separately from the local evidence above. The review
assets and progress notes were verified absent from exact-head generated
output after the receipt commit. Production remains subject to owner merge.

### Preview navigation follow-up

The owner found that archive links left the Pages preview for production.
The complete 79-page HTML audit found the same issue in the shared header,
article authors/permalinks, representative-card photo links, generated CV,
older article continuation links and a few other content links. All 54 archive
post links now use root-relative paths in their sources. The shared templates
and maintained content do likewise. Six README references to files excluded
from deployment now point to their actual GitHub source.

A site-specific finalizer normalizes same-origin body navigation in assembled
HTML, including CV/docs projections, without editing the CV JSON or historical
article sources. It tokenizes real anchors and patches only attribute bytes;
head metadata, comments (including CV email exclusions), scripts, raw vanity
fragments and non-HTML artifacts remain intact. Visible u-uid links transfer
their canonical identity to a hidden data element inside the same link, keeping
preview navigation and global identity separate. Genuinely hidden microformat
identity links retain their production URLs. RSS, sitemap, JSON-LD,
Standard.site and portable Markdown/agent references retain canonical URLs.
HTTP integrity hashes are calculated after navigation is finalized.

Guarded generation, full repository tests, vanity/microformats/consistency,
the full vanity race suite and vet pass. All 79 existing HTML paths remain,
with zero unjustified production body links. All 1,133 root navigation links
resolve on a preview origin, and all 54 archive post links are covered. Original
destinations are preserved apart from the six corrected GitHub references;
the generated rolling changelog's commit links are excluded from that comparison.
The three RSS feeds, CV JSON and five raw vanity fragments are byte-identical.
All representative homepage card properties and five project objects remain
unchanged when parsed against the canonical site base. All 36 requested browser
scenarios again pass axe, keyboard focus, local fonts and 320px normal/200%
reflow; ten real archive/header/article/CV/home clicks stay on the local preview
origin. Regression fixtures also cover escaping, external domains, unsafe URLs,
canonical UID preservation, idempotence and normalized HTML integrity hashes.

This follow-up belongs to [PR #28](https://github.com/darccio/darccio/pull/28).
Its commit receipt and fresh hosted preview checks follow below; custom-domain
deployment remains dependent on owner merge. T09 defaults remain deferred.

The navigation correction is committed as `6df4683`. Independent read-only
review found no material concerns. The signed complete unit passed darna;
no hooks were bypassed. The hosted receipt is kept in PR #28 so its fresh
preview URL and exact-commit checks can update without altering published content.

## Typography and chronological archives refinement

The owner confirmed PR #28 was merged and selected IBM Plex Sans for body text,
with Newsreader headings and IBM Plex Mono metadata/navigation/code. The new
batch also replaces the visible avatar/name header with the existing logo,
combines the homepage photo/bio, removes its visible name heading, improves
submenu/section spacing, and uses one shared chronological-list design for
posts and CV. Talks retain their two topic clusters and gain sourced year rails.
The supplied generated resume PDF is used only for visual hierarchy inspiration;
its text is not imported into the source CV. T09 personal defaults remain deferred.

### Body typography

Four self-hosted IBM Plex Sans Latin woff2 faces (400/500/600 and 400 italic)
come from integrity-verified Fontsource 5.3.0. The original OFL notice and each
file's provenance/SHA-256/size are retained. Added font bytes total 95,380.
The generated tokens remain byte-identical; the stylesheet adds the owner-selected
sans family. Newsreader headings and Plex Mono dates/navigation/code remain.
All font checks, guarded generation, repository tests, vanity, microformats and
publishing consistency pass. Browser acceptance for the complete batch is recorded
below; this foundation is isolated from the structural page changes.

### Header, homepage and shared spacing

The header displays only the existing logo/home link plus primary navigation.
Inner pages retain the same personal identity as hidden microformat data;
the logo never claims to be a personal photo. The homepage retains its original
representative card and bio wording, with one accessible hidden name heading
and a left photo/right bio unit. It stacks at narrow/enlarged-text sizes.
Shared archive rows use the same year/title/meta anatomy, 4px title/meta gap,
and separated section headings. Language controls have a measured 48px gap
before the article list; focusable controls retain comfortable target heights.

Guarded generation, full repository tests and metadata/vanity checks pass.
All 40 browser cases (ten pages × two widths × two themes) pass axe, skip/focus,
local font checks and 320px normal/200% reflow. All 74 styled pages pass the
wider reflow check. The five raw vanity fragments remain byte-identical.
Homepage card properties are exactly unchanged. Before/after screenshots cover
home, posts, CV and talks at 1440px and 390px. Hosted acceptance follows the PR.

### Article archives

Root, English and Spanish indexes each use one chronological archive-list,
without separate year headings. Their 35/9/12 rows retain identical ordered
titles, destinations, language labels and publication instants. The two external
citations retain their established year and named external source without
inventing exact dates. Per-row year rails, title/language badges and date metadata
reuse the same styles as talks and the CV. Guarded generation, existing complete
archive/microformat checks, all repository tests and the final 40-scenario browser
matrix pass. Canonical UIDs remain separate from preview navigation.

### CV timeline

The HTML CV projects the unchanged MAC JSON into one reverse chronological
timeline: 23 work/community roles, seven project roles and 15 education/credential
entries. Types remain p-experience or p-education, with the same source pointers,
dates and reviewed precision, organisations, descriptions, skills and certification
expiry distinctions. Stable sorting changes presentation only. The compact
96px photo/contact block removes the duplicated visible name and groups profile
links into comfortable wrapping targets. The resume PDF supplied for inspiration
remains outside the repository; no new biographical facts are imported.

The meaningful regression test exercises out-of-order input, mixed entry kinds,
stable source identity/date precision and a single semantic archive-list. Existing
resume validation remains unchanged. Full tests, guarded build, CV microformats,
JSON byte checks and all four CV browser scenarios pass. CV crawler/publishing
exclusions and email_off boundaries remain intact. Full body/source content is
retained; this is a layout change, not a shortened resume.

### Talk-year evidence

All 15 previously missing year rails are filled from primary linked materials,
videos and organiser sources: six verified event years and nine preparation or
publication years. The latter carry a visible asterisk, an explanatory page
legend, and row notes identifying materials or video publication/preparation.
Those resource years never become dt-start values. The evidence receipt records
each source, confidence and limits. Existing 31 appearances/order and all resource
destinations remain, and the ten established complete h-events retain their exact
name/start/location/url values. No full calendar date or venue is invented.
The source file and notes preserve every distinct esLibre/World.Party,
Autodefensa, Mergo and disruptive-technology appearance. Guarded generation,
microformats, repository tests and the final browser matrix pass. These newly
shown years are distinguishable from verified event dates in both wide and
narrow layouts; each resource note remains visible when the year rail is hidden.

### Refinement review receipt

Committed units: `f9ce4bb` Sans body, `d5e5cf7` logo/home/spacing,
`5ec3199` post archives, `3adb6ee` unified CV timeline and `cf2db6e`
talk-year evidence. Each complete unit passed darna and was signed without
bypassing hooks. Independent final review found no material defects.
[Review assets](../../.github/review/archive-design/verification.json) retain
16 before/after screenshots, the 40 browser cases, 74-page reflow results,
layout measurements, article invariants and primary talk-year evidence.
All assets are excluded from deployment. All 79 existing HTML paths, three RSS
feeds, source CV JSON, five raw vanity fragments and original token file remain;
the feeds/CV/raw fragments/tokens compare byte-for-byte with the merged baseline.
Seven vanity contracts, 33 dated posts, 54 feed items and 26 Standard.site plans
continue to pass. The hosted preview and exact-revision CI receipt follow in the
review PR; production changes only after owner merge. No owner input remains for
this design batch, and the deferred personal defaults remain deferred.

The review evidence is committed as `dfb8dbb`. Review is open in
[PR #29](https://github.com/darccio/darccio/pull/29). Its
[commit checks](https://github.com/darccio/darccio/pull/29/checks) and PR receipt
record the hosted preview and exact-revision results separately from the local
evidence above. The supplied PDF and unpublished planning files remain absent
from the deploy. Production remains dependent on owner merge.

### Header branding feedback

The owner requested their name alongside the logo. One root-relative home link
now combines the decorative logo and visible Dario Castañé in Newsreader.
The separate hidden identity card and representative homepage card remain
unchanged; the brand span does not introduce extra microformat name properties.
Responsive sizing and a shrinkable text span retain readable branding at narrow
and enlarged-text widths. Guarded generation, full repository tests, vanity and
metadata/format checks pass. Four focused 320px checks plus the complete 44-case
browser matrix pass in light/dark with normal/200% reflow and visible keyboard
focus; all 75 styled pages pass the wider overflow check. Hosted acceptance
follows the updated [PR #29](https://github.com/darccio/darccio/pull/29).

### Catalan archive feedback

The owner requested a Catalan-only filter. `/posts/ca/` presents the exact
Catalan subset of the combined archive: 12 local articles (eight under posts,
four hosted Rizoma issues) and the existing Núvol citation. All 13 row bytes,
order, names, URLs, language labels and publication instants are preserved.
All four archives link to all languages, Català, English and Castellano, with
one selected submenu link and a distinct underline. `/posts/` remains combined.
This static filter works without JavaScript and explicitly stays out of RSS and
Standard.site publishing. Parent section configuration supplies posts navigation.

The mandatory completeness gate now checks the fourth archive against the
independent post inventory and permits only its reviewed Catalan citation.
Meaningful tests reject missing local/Rizoma entries and foreign articles/cites;
existing English/Spanish citation policy remains unchanged. Full repository tests,
guarded build, vanity, microformats and publishing-format checks pass. The 44-case
browser matrix and 75-page reflow check pass. Existing archive rows, homepage
identity, CV JSON, all feeds and raw vanity fragments remain unchanged.

The owner feedback is committed as `4c37fae` (header name) and `8047efe`
(Catalan-only archive). Each atomic unit passed darna and was signed without
bypassing hooks. The review screenshots now include the visible name and the
new Catalan view at desktop/mobile widths. Updated evidence records 44 browser
cases, 75 styled pages and 80 total HTML routes, retaining all 79 prior routes.
Hosted verification follows push and is recorded in PR #29 separately from
these local receipts. Production remains dependent on owner merge.

### Page-aware language menu · 13 October 2026

The owner reported that the live language submenu differed from the section
navigation and returned to language homepages even when translations existed.
The language menu now shares the primary navigation's mono uppercase type,
spacing, rules, current underline, 44px targets and narrow-screen treatment.
Its current language points to the current page with `aria-current="page"`.

The guarded assembler pairs published Catalan root pages with matching English
and Spanish paths. `.zas/translations.yml` records three content-reviewed sets
with different slugs: New Path, Liquid Organizations and the Matthew Pherigo
Facebook letter. No article slugs or sources change. Available Spanish siblings
appear as Castellano; missing Catalan/English translations retain their existing
homepage fallback. Shared English CV/docs keep their root self URLs; the
Catalan-only archive keeps its own self URL. Menus use local paths so previews
retain their host. The same 24 groups (53 pages) supply reciprocal sitemap
alternates, including English article/archive siblings previously masked by a
nearest-directory config containing only `nav: posts`.

Implementation and validation subagents reviewed translation facts, maintained
the locale checker and checked the assembler. Tests reject absent/wrong-language
or conflicting targets, bad manifests, malformed/duplicate menus and a current
link that leaves its page. Failed builds preserve the last successful deployment.
`make -f .zas/Makefile generate` with pinned Zas v0.0.9, `test`, `check-vanity`
and `check-microformats` pass. No secrets or client scripts are added.

Local browser evidence covers 56 accessibility/keyboard/font cases plus 12
matching-menu style cases, both themes at 1440px and 390px. Axe has zero
violations; language links measure at least 44px high. Twenty-seven real menu
clicks retain the local origin; all 92 styled pages pass 320px normal/200% text
reflow. All 97 HTML routes survive. Three feeds, CV JSON, tokens and five raw
vanity fragments remain byte-identical; outside the language menus only the
Git-generated changelog differs from the earlier English-site build.
The implementation is committed as `655f04b`, with Darna passing and its SSH
signature verified. [Review evidence](../../.github/review/language-menu/verification.json)
contains twelve before/after screenshots plus browser, click, reflow and invariant
reports. Hosted exact-revision acceptance is recorded after push. Production
publication requires owner merge; the separate image-fix PR remains independent.
The review evidence is committed as `31f2c37`; both commits have verified SSH
signatures. [PR #32](https://github.com/darccio/darccio/pull/32) contains this
change and the screenshots. Its [checks](https://github.com/darccio/darccio/pull/32/checks)
and acceptance comment record the hosted preview and exact-head validation.
A read-only merge-tree check also confirms compatibility with image PR #31.

### English personal site

The owner requested a translated English homepage and English article navigation.
Seventeen English pages now cover the homepage, biography, current work, projects,
talks, contact, hello, feeds, slash index, colophon, AI policy, languages, profiles,
sponsorship, tools, reading and generated changelog. The nine existing English
articles remain the English archive; historical articles are not newly translated.
Catalan remains the root/default language, with Spanish article URLs retained.

The shared header brand, primary navigation and footer use the translated English
routes whenever the page language is English. Articles defaults to `/en/posts/`.
The inherited posts navigation key now lives in `en/posts/.zas.yml`, so auxiliary
English pages do not incorrectly select Articles. A static Català/English switch
links the two homepages. Existing CV and project guides are already English and
keep their established root URLs, as do licences, feeds and identity identifiers.
The translated homepage/about cards retain the canonical root personal UID.

Two implementation subagents translated and reviewed the public source material.
Dates, roles, activity, project names and external resources were copied from the
existing public pages. All 31 talk titles, 46 external links and ten complete event
tuples remain exact; language cues identify preserved Catalan/Spanish names.
English contact embeds are restricted to the new reviewed literal partial. The
English agent corpus explicitly includes the translated Markdown pages; the CV
remains outside it. The changelog renders the same bounded Git history with an
English introduction, preserving original commit messages and escaping.

Guarded generation, the full repository test suite, vanity and microformats gates
pass. New negative fixtures reject incorrect English language, root/production
navigation, incorrect section selection, personal UID drift and unreviewed embeds.
The assembled sitemap has 17 reciprocal Catalan/English translation pairs. All
80 prior HTML routes remain, with 17 additions; feeds, CV JSON, tokens and five raw
vanity fragments remain byte-identical to the merged PR #29 baseline. Local
browser checks pass in 112 cases over 28 routes: axe has zero violations in both
themes at desktop/mobile widths, skip links and keyboard focus work, and 320px
normal/200% text reflow passes. All 92 styled pages pass the wider reflow check.
Commit, review artifacts, hosted preview and exact-revision CI receipts follow;
production publication requires owner merge.

The complete translation and its regression checks are committed as `c9f2c76`.
Darna and signing verification pass; no hooks were bypassed. Final talk-language
review adds four passing focused browser cases to the 112-case matrix. Twenty-five
real navigation clicks retain the local/preview origin, including English-home
branding and writing links, Catalan/English switching and shared CV/docs resources.
[Review evidence](../../.github/review/english-site/verification.json) includes six
desktop/mobile screenshots, browser/reflow/navigation reports and the unchanged
artifact hashes. Hosted acceptance is recorded separately after push.

The review evidence is committed as `5444f35`; implementation and evidence commits
have verified SSH signatures. [PR #30](https://github.com/darccio/darccio/pull/30)
contains the English-site change, screenshots and source/check receipts. Its
[checks](https://github.com/darccio/darccio/pull/30/checks) and PR acceptance receipt
record the deployed preview and exact-head results. No owner facts or account
settings are needed for this translation; publication follows owner merge.

### Article images

The owner reported missing images in A New Path and Pseudo-Estat. Their existing
JPEG files are served correctly, but the original article sources declared only
Open Graph images; neither the original nor the current layout inserted them into
the visible content. Four source articles (the three New Path languages and
Pseudo-Estat) now include one image after their H1, with localized alt text and
the actual 1200×628 or 1920×1080 dimensions. This also includes the covers in RSS,
Markdown twins and the reviewed context output. Existing metadata, publication
instants, article prose and URLs are retained; the portal article keeps its one
existing gopher image. An implementation subagent traced history, inspected the
assets and verified that removing only the added image recovers each old source.

Guarded generation, the full test suite, vanity/microformats and publishing-format
checks pass. Forty-four article browser cases pass axe in light/dark at desktop
and mobile widths, keyboard focus and 320px normal/200% text reflow. A separate
Chromium decode check confirms all 14 images across the ten affected articles and
the portal page load successfully. Before/after screenshots use the actual JPEG
MIME type and wait for decoding. Commit and hosted checks follow below.

The cover fixes are committed as `f0d4622`, with darna and SSH signing checks
passing. A separate audit found nine incorrect body URLs in the six translated
Delete Facebook #1/#2 posts, plus six stale `images:` metadata references. These
now use the existing `/static/images/` assets. The build checks actual local
`img src` and Open Graph targets against assembled files before promotion; a 200
homepage fallback cannot mask a missing image. Query strings, escaped names,
page-relative/root-relative/same-site URLs, default ports and first duplicate
attribute behavior are covered. Comments and JSON script text are ignored.
Negative tests require missing, empty or directory assets to reject the build and
preserve the previous deployment. Remote hosts remain outside this offline gate.
Independent subagent review and the full repository suite pass; no production
deploy depends on remote image availability.

The local URL fixes and image guard are committed as `f2e48e5`, with darna and
verified SSH signing. A broader audit found three unavailable Twitter images in
Rizoma #001/#002; their original URLs return 404 for both HEAD and GET, including
alternate file formats. Six other remote image endpoints return image content.
Two exact original small variants were recovered from Internet Archive snapshots,
visually inspected and copied unchanged into `/static/images/`: the eight-portrait
inequality graphic (680×570) and the climate infographic (680×357). Their original
source credits remain. The climate graphic gains descriptive Catalan alt text and
dimensions. Both articles keep their original dates and prose; the staged build,
full tests, vanity, microformats and publishing-format checks pass.

The Cobi statue photo in Rizoma #001 remains unavailable at its original Twitter
URL. Bounded archive lookups encountered 503, timeout and 429 responses; this does
not establish that no archived copy exists. The owner confirmed no originals are
available and requested that the limitation be recorded. The original URL and alt
text are retained, and no replacement image or historical fact is invented.

The recovered-original unit is committed as `521dcbf`. All three implementation
commits have verified SSH signatures and passed darna without bypassing hooks.
[Review evidence](../../.github/review/article-images/verification.json) records
the 44 browser cases, 12 before/after screenshots, image decoding and exact archive
URL/hash provenance. Thirteen pages (the ten corrected articles, portal and two
Rizoma editions) successfully decode all 16 local body images. All original feed
item GUIDs, titles and publication instants remain; current git-based update dates
advance for corrected sources. Mandatory cross-format checks still cover 33 posts,
54 feed items and 26 verified Standard.site documents. Hosted checks follow push.

The local evidence is committed as `811057a`. [PR #31](https://github.com/darccio/darccio/pull/31)
contains the three implementation units, source/archive receipts and review images.
The Delete Facebook screenshots explicitly scroll to the illustration so the
before/after difference is visible. The PR's [checks](https://github.com/darccio/darccio/pull/31/checks)
and hosted acceptance receipt record the exact final revision. The Cobi limitation
is the only missing image left from this audit; production follows owner merge.

### Inverted main-navigation trial · 13 October 2026

The owner requested an inverted general menu. The five-link main navigation uses
the plum `--accent` background and white links; its current item uses near-black
`--ink-900` on a white highlight with the retained 2px plum underline. Hover and
inset focus rings remain readable in both themes. The language menu keeps its
plain styling and the translated-page destinations already merged in PR #32.

The first trial (`20cd362`) applied this treatment to the language row by mistake.
The owner clarified that the general menu was intended; the styling correction
moves all inverted overrides to `.primary-navigation`. The earlier evidence
(`a4489ec`, receipt `80ee808`) remains in Git history as superseded trial data;
the PR description and review artifacts now describe the final main-menu design.
This correction keeps translated targets, markup, content, fonts and URLs intact.

The implementation subagent verified 36 focused browser cases (ca/en/es, both
themes, 1440px/390px/320px, normal/200% text). All five main links fit one row at
390px in every locale, with targets at least 44px high; smaller/enlarged views
wrap without overflow. The guarded `generate` build, full `test`, `check-vanity`
and `check-microformats` pass with pinned Zas v0.0.9. The publishing consistency
check still covers 33 posts, 54 feed items and 26 verified Standard.site documents.
The correction adds no client scripts and is committed as `0d86bc4`, with Darna
passing and its SSH signature verified. [Corrected review evidence](../../.github/review/primary-navigation/verification.json)
includes twelve Catalan-homepage/English-archive screenshots. Another 44 browser
cases report zero axe violations in both themes; all 92 styled pages pass 320px
normal/200% reflow. Feeds, CV JSON, tokens and raw vanity fragments retain their
hashes. Superseded language-row screenshots are replaced in the final PR tree.
[PR #33](https://github.com/darccio/darccio/pull/33) now describes the main menu;
its exact-head checks and hosted acceptance receipt follow after push.

The corrected review evidence is committed as `88ac713`; it replaces the
superseded language-row artifacts with main-navigation screenshots and checks.
Both correction commits have verified SSH signatures. The final PR title and
description refer to the general menu. Hosted exact-head acceptance is recorded
in the PR receipt before owner merge.

### Reference header layout · 13 October 2026

The owner supplied a header reference with the language controls beside the logo
and name and a full-width plum main-menu band below. The layout now groups the
brand and compact CA/EN/ES links in one top row, with full language names in their
accessible labels. Current-language and current-section underlines remain. The
five primary links fill the content column, fit one row at 390px and wrap safely
on narrower screens or with enlarged text. Every header link has a 44px-high
click target; language controls are also at least 44px wide.

The implementation subagents covered stylesheet behavior and the locale checker.
Build-time language rendering and its regression checks now require compact codes
and full accessible names. Translated-page targets, language fallbacks, canonical
metadata, page content and identity microformats retain their existing values.
No client JavaScript, font or dependency changes are introduced.

The guarded build with Zas v0.0.9, full tests, check-vanity and check-microformats
pass. Publishing consistency still covers 33 public posts, 54 feed items and
26 verified Standard.site documents. The focused 36-case browser matrix passes
for three languages, both themes, 1440/390/320px and normal/200% text. Desktop
first/last menu links align with the content bounds, and screenshot pixels confirm
the plum band reaches both viewport edges. Broader review evidence and the signed
commit receipt follow below; production follows owner merge of PR #33.

[Updated review evidence](../../.github/review/primary-navigation/verification.json)
records 44 full axe cases with zero violations, 44 keyboard skip/header-focus
cases, 368 reflow checks across all 92 styled pages in both themes, and 27 real
translated-page clicks. Twelve before/after screenshots cover the Catalan homepage
and English archive. The head metadata and body outside the site header match
baseline on all 92 pages; feeds, CV JSON, tokens and vanity tags retain exact bytes.
Only the generated CV sitemap lastmod advances between isolated builds; all other
sitemap bytes match. Browser checks use Linux Chromium and local assets, with
external requests blocked. Doubled text means a 200% root font size at a fixed
320px viewport. Final measurements wait for resources and fonts to finish loading.

The implementation is committed as `f8ac149` on 14 October 2026 after the owner
unlocked the configured signing key. Darna passes and the SSH signature verifies;
hooks were not bypassed. PR #33 will carry this reference layout and the updated
screenshots. Hosted acceptance is recorded against its final pushed revision.

The refreshed screenshots and browser receipt are committed as `c927146`; this
replaces the earlier main-menu trial artifacts in the final PR tree. Both the
implementation and evidence commits have verified SSH signatures. The review is
[PR #33](https://github.com/darccio/darccio/pull/33), retitled around the supplied
reference layout. Its hosted acceptance comment identifies the final deployment
and review revision; production publication follows owner merge.

### T33 historical-article recovery backlog · 14 October 2026

The owner requested another item to recover older articles from other sites,
either directly or through archive.org. [T33](tasks/T33.md) records source
inventory, original/archived retrieval, Markdown conversion, local media,
attribution, publication-date precision, deduplication and reviewable imports.
The original T00–T32 tasks and their numbering are unchanged. This is a planning
addition; no scraping, article publication or AT Protocol writes occur now.

Each recovered article keeps its original wording and language, with its source
and archival provenance recorded. Retrieval and archive captures never become
publication dates. New imports stay out of RSS and Standard.site until their
metadata and publication scope are reviewed; existing article URLs and syndication
identities remain stable. Missing text/media and unresolved dates remain explicit.
The task’s dedicated subagent prepared its handoff. Existing unpublished planning
files remain unchanged except this progress log; the new task page is also private.
The guarded generate build, full test suite, check-vanity and check-microformats pass. T33 and the progress log remain absent from deploy output. This validates the planning addition; retrieval/import verification belongs to the later implementation. The signed commit receipt follows below.

### T34 social-post and thread recovery backlog · 14 October 2026

The owner added recovery from Facebook, Twitter and Bluesky, with related
microblogging posts from Twitter and Bluesky threaded together. [T34](tasks/T34.md)
reuses T33’s recovery/provenance workflow and adds stable post identities, reply
relationships, branching threads, missing or deleted posts, edits and cross-platform
copies. Topic-related posts may form clearly labeled editorial collections; these
are distinguished from original reply chains and retain individual source links
and timestamps. Facebook recovery retains individual post content and chronology.

The dedicated T34 subagent supplied the queued handoff. Raw account exports and
credentials remain outside Git and deployment. Reviewed public imports use local
media and metadata and default to feed:false/atproto:false until inclusion is
selected. The planning addition performs no platform retrieval, export parsing,
article publication or remote writes. Existing dates, URLs, feeds and approved
Standard.site records remain unchanged. Build, test and private-plan exclusion
checks pass; the final committed build is checked before the review is opened.

The two backlog items are committed independently as `76d688a` (T33) and
`d934a3a` (T34), with darna passing and both SSH signatures verified. Guarded
generation, the existing full test suite, vanity and microformats checks pass;
the planning files remain absent from deployment. No retrieval tooling or new
recovery tests were added. Final review checks and the PR receipt follow push.
