<!--
publish: false
-->

# Site implementation progress

All 33 tasks have a dedicated completed subagent review. The plan and decision register are ready; implementation and live acceptance are tracked separately below.

| Item | Actual outcome | Commit or PR | Verification | Remaining work |
| --- | --- | --- | --- | --- |
| T00 | Timestamped source and live audit recorded | `6d12b55` | Facts file exists; all seven exact vanity tuples observed; DID and feed parsed; unrelated user doc hash preserved | Recheck after deployment; remaining Pages settings and registrar controls need account evidence |
| Organization | T00–T32 mapped to dependencies, atomic units, checks and rollback; D1–D17 recorded | This planning commit | 33 distinct reviews; complete IDs; acyclic task dependency graph | Execute queued items as dependencies become ready |

## Current account and repository conditions

Cloudflare Pages with Git integration is confirmed by the owner. The GitHub API shows a successful `Cloudflare Pages` check from app `cloudflare-workers-and-pages` on the production branch; no GitHub deployment records or tracked site deployment workflow exist. Confirm the production branch/build/output settings before attaching host-specific actions.

Public .cat RDAP reports expiry on 1 June 2027 and `delegationSigned: false`. Auto-renew, valid payment, transfer lock and external monitor configuration need real account evidence. These settings have not been changed.

Only this site and Zas are implementation repositories in this request. Mergo and darna source-owned changes stay explicit follow-ups. Existing `docs/go-vanity-url.md` remains untracked and unchanged. Local Zas’s existing branch is preserved; new work uses isolated branches based on current upstream master.

## Implementation queue

Begin with the separate duplicate translation-key fix, then T02’s staged build and vanity parser, T01’s shared scheduled monitor, T03’s Zas identity docs and T05’s additive metadata helpers. Zas items get their own atomic commits and focused PRs. Subsequent content and format tasks remain queued; optional T06 and T23 are deferred.

Registrar/hosting/account access and report-required reviewed publication selections are evidence requirements. Existing session authorization continues to apply; do not ask again for actions or defaults already approved. Standard.site implementation starts with offline previews and requires the approved allowlist and actual account access before public writes.
