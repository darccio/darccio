<!--
publish: false
template: false
-->

# Domain protection and vanity monitoring

This is the owner runbook for `dario.cat`. The GitHub workflow is a software backstop; registrar settings, DNSSEC and the independent monitor require account actions and real evidence before T01 is complete. Keep account screenshots, invoices, payment details, recovery codes and alert addresses in private storage; record only their location and review date here.

## Observed baseline, 2026-10-11

The public [.cat RDAP response](https://rdap.nic.cat/rdap/domain/dario.cat), recorded in [HANDOFF-FACTS.md](HANDOFF-FACTS.md), reports expiration at **2027-06-01T07:05:04.361Z**, status `active`, and `secureDNS.delegationSigned: false`. Nameservers are `ray.ns.cloudflare.com` and `jean.ns.cloudflare.com`. This expiry provides less than one year of paid runway. Auto-renew, payment validity and transfer lock are **unverified**; the public status does not prove that lock is disabled. Recheck the registry and dashboards after each account action.

Cloudflare Pages with Git integration is owner-confirmed: production branch `main`, build command `make -f .zas/Makefile publish`, output `.zas/deploy`, and an empty root-directory setting. This monitoring workflow neither builds nor deploys the site and needs no Pages or PDS credentials.

## Verification receipt, 2026-10-11

At **20:57 UTC**, the public checks confirmed:

- [Vanity metadata monitor run 38173556457](https://github.com/darccio/darccio/actions/runs/38173556457) completed successfully at **20:46:01 UTC** after manual dispatch on `main` (`84f707d`). Its checker passed all seven packages, with and without trailing slashes, against the reviewed metadata baseline. This is manual-run evidence; an observed scheduled run and notification delivery remain pending.
- The [.cat RDAP response](https://rdap.nic.cat/rdap/domain/dario.cat) was retrieved successfully with TLS verification enabled. Expiration remains **2027-06-01T07:05:04.361Z**, status remains `active`, and `secureDNS.delegationSigned` remains `false`; both Cloudflare nameservers are unchanged. Account auto-renew, payment validity and transfer lock remain unverified.
- [Google DS](https://dns.google/resolve?name=dario.cat&type=DS&do=1) and [DNSKEY](https://dns.google/resolve?name=dario.cat&type=DNSKEY&do=1) queries, cross-checked through Cloudflare DNS-over-HTTPS, returned `NOERROR` with no DS or DNSKEY answers. The DS responses had `AD:true`; DNSKEY responses had `AD:false`. These observations support the unsigned-delegation baseline and do not establish an active DNSSEC chain.

No account, billing, DNS or provider settings changed. Paid runway greater than one year, registrar protection, validated DNSSEC, an independent 30-minute monitor and received test/recovery/expiry alerts still require owner evidence.

## Verification receipt, 2026-10-12

All four review PRs are merged. Site main `1fb265cfe0f1b6e1ccac9f4238e2cbb9f6d903d3` has successful Pages, [Site checks](https://github.com/darccio/darccio/actions/runs/38210408284) and [post-deploy vanity](https://github.com/darccio/darccio/actions/runs/38210408194) results. An additional live checker invocation passes all fourteen route variants.

The previously pending scheduled-run evidence is now present: [Vanity metadata monitor 38196543370](https://github.com/darccio/darccio/actions/runs/38196543370), triggered by `schedule` on 2026-10-12, succeeds. This establishes an enabled scheduled backstop; independent checks and received alert/recovery notices remain owner work.

Fresh registry and validating-resolver checks still report expiration `2027-06-01T07:05:04.361Z`, status `active`, `delegationSigned:false`, unchanged Cloudflare nameservers, and no DS/DNSKEY answers. Auto-renew, payment, transfer lock and independent monitoring remain unverified. No account or DNS change was made.

## Owner confirmation, 2026-10-12

Dario confirms auto-renew enabled, transfer lock enabled, billing and contact details checked, and a registrar expiry of **2028-06-01**. These are owner-confirmed account facts; no payment or contact details are recorded here.

Independent public verification at **2026-10-12 07:13:57 UTC** confirms registry expiration `2028-06-01T07:05:04.361Z` at [registry RDAP](https://rdap.nic.cat/rdap/domain/dario.cat), with registry WHOIS agreeing. The registry reports `renew period` and a last-change timestamp of `2026-10-12T07:10:05.613Z`. [Registrar RDAP](https://rdap.swhosting.com/domain/dario.cat), checked at **07:12:44 UTC**, reports `registrar expiration: 2028-06-01T00:00:00Z`. The earlier old-year response was superseded by these fresh checks; paid runway greater than one year is now independently confirmed.

Auto-renew, transfer lock and billing/contact checks are owner-confirmed. Both RDAP responses still report `delegationSigned:false`; WHOIS reports unsigned DNSSEC, and the same Cloudflare nameservers remain. Item 1 is complete. Validated DNSSEC and independent alert setup remain open.

## Cloudflare DNSSEC handoff, 2026-10-12

The owner supplies the following public DS record after enabling Cloudflare signing:

```text
dario.cat. 3600 IN DS 2371 13 2 64921FF6361DDA0ECC288F28CAFACA444B5E162BE06DB16FB1910D4316082762
```

The public KSK has flags `257`, protocol `3`, algorithm `13` and key tag `2371`:

```text
mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==
```

BIND's `dnssec-dsfromkey -2` independently derives exactly the supplied key tag and SHA-256 digest from the retrieved DNSKEY. Independent wire-format hashing also agrees. At **07:30:47 UTC**, the [HTTPS DNSKEY response](https://cloudflare-dns.com/dns-query?name=dario.cat&type=DNSKEY&do=1) contains the same KSK and DNSKEY RRSIG. Cloudflare and [Google HTTPS DS responses](https://dns.google/resolve?name=dario.cat&type=DS&do=1) have status NOERROR, AD=true and empty answers with signed denial records; `delv` reports an unsigned answer. These establish resolver-visible authenticated DS absence, not a complete DNSSEC chain. Requests naming authoritative servers returned recursive flags in this environment, so they are not evidence of a fresh direct authoritative query.

Cloudflare signing is verified at this handoff. Registrar publication of the DS and a fully authenticated DNSSEC chain remain pending. The registrar DS form needs key tag `2371`, algorithm `13`, digest type `2` and the digest above. Keep the current Cloudflare nameservers. If the domain-control panel has no external-DS form, the owner should ask SW Hosting support to publish these exact DS values at the `.cat` parent. No external message or DNS mutation was performed by the agent.

## DNSSEC completion receipt, 2026-10-12

The owner saved the Cloudflare KSK in SWPanel's domain **Registros DSSec** form. Its fields accept DNSKEY data: **Marques (flags) `257`**, **Protocol `3`**, **Algoritme `13`**, and the exact public key above. Algorithm `13` is ECDSA P-256 with SHA-256; DS digest type `2` separately selects SHA-256. Submitting this key data produced the matching parent DS; this form does not require a separate key-tag or digest field.

Fresh [registry RDAP](https://rdap.nic.cat/rdap/domain/dario.cat) reports `secureDNS.delegationSigned:true` and the exact `257 / 3 / 13` public KSK in `keyData`; registrar RDAP also reports `delegationSigned:true`. The registry records the change at `2026-10-12T08:10:09.378Z`. Nameservers remain `ray.ns.cloudflare.com` and `jean.ns.cloudflare.com`.

After earlier unsigned answers, successful verification completed at **2026-10-12 08:17:15 UTC**:

- [Google's DS response](https://dns.google/resolve?name=dario.cat&type=DS&do=1&cd=0) and Cloudflare's DNS-over-HTTPS JSON response both contain exactly the supplied DS: key tag `2371`, algorithm `13`, digest type `2`, and digest `64921FF6361DDA0ECC288F28CAFACA444B5E162BE06DB16FB1910D4316082762`. Digest letter case differs between providers but the bytes agree.
- Both providers' DS, A and DNSKEY responses have `Status:0`, `AD:true`, `CD:false`, with the expected answers present. Cloudflare JSON queries use `https://cloudflare-dns.com/dns-query` with `Accept: application/dns-json` and `name=dario.cat&type=<type>&do=1&cd=0`.
- `delv @1.1.1.1 dario.cat A` and `delv @8.8.8.8 dario.cat DNSKEY` both report `; fully validated`, establishing the complete authenticated chain. These are validating-resolver checks; no direct-authoritative claim is made.
- The committed live vanity checker passes all **fourteen** bare/slash `?go-get=1` routes for the seven reviewed packages after DS publication.

This receipt supersedes the earlier DS-absence and pending-chain status. Owner items **1 (renewal/protection)** and **2 (DNSSEC)** are complete. T01 remains open for independently scheduled monitoring and received failure, recovery and expiry notices. The agent performed read-only verification; the owner changed the account settings.

## Datadog partial-apply receipt, 2026-10-12

After resolving authentication, the owner applied the monitoring module. Datadog
created the certificate test, both DNSSEC tests and all three registry-expiry
tests. It rejected the seven vanity tests because its variable interpolation
mistook double braces in the parser's compressed entity table for a global
variable reference. The assertion-build fix escapes this string data while
preserving the tokenizer's decoded values and exact metadata checks.

Keep the owner's existing OpenTofu state and local variables. After updating the
code, generate a fresh plan according to the current quota-migration receipt
below; see the module's
[partial-apply instructions](../.infra/datadog/README.md#resume-a-partially-failed-apply).
Resource creation does not establish hosted assertion success, scheduled
monitoring or received alerts. Those operational checks remain open.

The subsequent seven failures exposed an independent assertion-size constraint.
The owner tested a tiny multistep assertion successfully, then received the same
HTTP 400 for the full 94,833-byte script and a trivial script padded to that
size. Fixed-size paused probes accepted 32,768 bytes and rejected 65,536 bytes;
all five accepted temporary tests were cleaned up. A finer probe accepted
49,152, 50,000 and 51,200 bytes, rejected 60,000 bytes, and cleaned up its three
accepted tests. These results establish a
usable lower bound, not an exact service maximum. The compact build retains
parse5's grammar and checks all injected scripts against a 49,000-byte budget;
the largest reviewed route/probe script is 48,868 bytes.
The owner then created a paused HTTP test with the actual 48,858-byte Mergo
replacement successfully, alongside the tiny multistep control. Both temporary
tests were cleaned up. The accepted artifact's SHA256 is
`de41f1a14fb161269505561fd7d467eacfe664b202fca15e3404f03451d11987`.
This confirms API acceptance; real managed-location execution remains open.

## Datadog multistep quota migration, 2026-10-12

After the compact assertion fix, the owner created the Darna, Awaken, Errors,
X and Xdr vanity tests. Mergo and Cocoon creation failed with HTTP 402 and
`Quota reached for multi tests: 5`. The prepared layout groups Mergo/Cocoon into
four steps every thirty minutes and the remaining five paths into ten steps
every two hours. All fourteen request/metadata checks remain, with the same
9,540 scheduled requests per thirty days and the configured $4.77 estimate.
Alerts now operate per group and identify the failing step and URL.

Explicit `moved` blocks reuse the successful Darna test (`kv2-z2c-77h`) for the
priority group and Xdr (`m4f-5g8-vxh`) for the other group. A fresh plan against
the reported state and unchanged variables should show zero additions, two
in-place changes and three redundant vanity-test destructions. The six
certificate, DNSSEC and renewal tests remain unchanged. Keep the existing
state and check public-ID continuity before applying; don't use `state rm`
or remove tests through the dashboard to free quota.

An offline plan using the actual pinned provider and eleven fake prior IDs
verifies these two moves, zero additions, two updates, three removals and six
auxiliary no-ops. It validates both retained IDs and all grouped request fields
without account access or reading the owner's actual state.

## Registrar protection in SW Hosting

1. Sign in to SWPanel and open **Domains and SSL → Domain Portfolio → dario.cat**. Confirm that this is the account controlling registration and that owner/admin contact email reaches you. Review MFA and recovery access.
2. In the domain actions, use **Enable Auto-renewal** if needed; verify the resulting enabled state. Check the billing method, its expiry and any required balance privately. SW Hosting documents the controls in its [domain actions guide](https://www.swhosting.com/en/comunidad/manual/domains).
3. Use **Block Domain** if needed and verify the resulting transfer-lock state. Keep a dated dashboard record; absence of a lock flag in RDAP is insufficient. See [SW Hosting's lock procedure](https://www.swhosting.com/en/comunidad/manual/how-to-block-a-domain).
4. Use **Renew domain**, review the years and price in the renewal cart, and have the owner complete payment so the paid expiration is more than one year ahead of the action date. Retain the receipt, new dashboard expiry and updated registry response. Auto-renew alone does not prove paid runway. See [SW Hosting's renewal procedure](https://www.swhosting.com/en/comunidad/manual/how-to-renew-a-domain-from-your-swpanel).

## DNSSEC with Cloudflare authoritative DNS

The current nameservers belong to Cloudflare. SW Hosting's [hosting DNSSEC switch](https://www.swhosting.com/en/comunidad/manual/how-to-activate-dnssec-on-my-hosting) requires its own authoritative nameservers and excludes Cloudflare; use the registrar's external DS support for this zone instead.

1. Save the existing parent DS response and TTL, nameservers and Cloudflare zone settings privately. Confirm that the registrar can publish Cloudflare's DS values to the `.cat` parent before activation.
2. In Cloudflare, select **dario.cat → DNS → Settings → DNSSEC → Enable DNSSEC**. Keep the existing nameservers. Copy the generated key tag, algorithm, digest type and digest exactly.
3. Use SW Hosting's separately documented [domain DS manager](https://www.swhosting.com/es/comunidad/manual/como-gestionar-registros-ds-desde-swpanel): **Dominios y SSL → Cartera de dominios → dario.cat → […] → Registros DSSec → Crear Registro**. The control is documented for Professional/Reseller modes. If hidden, their [mode guide](https://www.swhosting.com/es/comunidad/manual/modos-de-swpanel) documents a free switch via **profile → Mi SWPanel → Preferencias de mi SWPanel → Funcionamiento → Profesional → Guardar preferencias**. The owner-confirmed form accepts DNSKEY flags, protocol, algorithm and the base64 public key; enter Cloudflare's KSK values exactly, as in the completion receipt above. A form accepting DS data instead needs Cloudflare's key tag, algorithm, digest type and digest. Keep Cloudflare nameservers. This domain control is distinct from the hosting-signing switch above. If the action is missing or rejected, ask registrar support to confirm and publish the external Cloudflare DS. Publication and successful chain verification remain required after any key change.
4. After propagation, compare the published parent DS to Cloudflare's values and validate the chain from a network-capable host:

   ```sh
   dig +dnssec dario.cat DS
   dig +dnssec dario.cat DNSKEY
   delv dario.cat
   dig @1.1.1.1 +dnssec dario.cat A
   dig @8.8.8.8 +dnssec dario.cat A
   ```

   Require successful validation, matching DS/key data and healthy HTTPS/import checks. Save dated command output and dashboard status privately; seeing a DNSKEY alone is insufficient.

For rollback, remove the parent DS through the registrar first. Keep Cloudflare signing enabled until the previous parent DS TTL has expired and confirming resolvers no longer return DS; only then disable signing. Preserve the working zone and nameservers. Follow [Cloudflare's DNSSEC activation and rollback guidance](https://developers.cloudflare.com/dns/dnssec/).

## Independent external monitor

The owner selected **Datadog EU1** and **cc@dario.cat** for notifications. The
[Terraform module](../.infra/datadog/README.md) prepares two grouped import
checks on the owner's selected budget schedule: **Mergo/Cocoon every 30 minutes**,
the other five imports **every 2 hours**, and **daily** validating-resolver/TLS
checks and registry-renewal warnings at **90, 60 and 30 days**. A single location
schedules **9,540 runs per 30 days** (9,858 per 31 days), about **$4.77/month**
at the published annual-agreement rate; the local rate input can reflect the
account's actual pricing. Retries/manual runs are additional.
It runs from Datadog's managed Frankfurt location independently of GitHub,
Pages builds and PDS. Tests are paused by default; account apply, actual
scheduled successes and received failure/recovery/expiry notices are still
pending. No Datadog API/application credentials are available in this session.
Both URL forms for all seven packages remain checked in fourteen critical
steps. Grouped alerts identify the failing step; all steps continue after a
failure and a group recovers when every step passes.

The module reuses the reviewed tuples below. Each bare request must preserve
the current exact 308 redirect to its slash URL with `go-get=1`; the canonical
request then parses real HTML metadata with a bundled tokenizer. This is a
stricter observed single-hop routing check than the Go checker's permitted
five-hop same-origin redirects. Datadog's assertion runtime must be confirmed
with real on-demand runs before activation. Configure each of these names
with both `/<name>?go-get=1` and `/<name>/?go-get=1`:

| Name | Exact expected `go-import` content |
| --- | --- |
| mergo | `dario.cat/mergo git https://github.com/imdario/mergo` |
| darna | `dario.cat/darna git https://github.com/darccio/darna` |
| cocoon | `dario.cat/cocoon git https://github.com/darccio/cocoon` |
| errors | `dario.cat/errors git https://github.com/darccio/errors` |
| xdr | `dario.cat/xdr git https://github.com/darccio/xdr` |
| awaken | `dario.cat/awaken git https://tangled.org/dario.cat/awaken` |
| x | `dario.cat/x git https://tangled.org/dario.cat/x` |

Require a valid TLS certificate, final HTTP 200, at most five HTTPS redirects on `dario.cat` retaining `go-get=1`, a maximum 1 MiB body and a 10-second request timeout. Parse the HTML metadata and reject missing, changed, duplicate or extra `go-import`/`go-source` tags. The reviewed baseline has **no `go-source` tags**. Check the response HTML without executing the meta refresh to pkg.go.dev. An uptime check or substring anywhere in the page cannot establish this contract.

An independently scheduled runner can reuse the committed checker from the repository root:

```sh
go -C .tools/vanity run ./cmd/vanity -live -base-url https://dario.cat -timeout 10s
```

Also configure domain-expiry notices at **90, 60 and 30 days**, plus an alert if registry expiry cannot be retrieved, and TLS-expiry notices supported by the provider. Verify the recipient and a fallback route independently of `dario.cat`. Use the provider's test-notification action for expiry notices; exercise a missing/wrong metadata failure on a dedicated test endpoint and its recovery. Retain actual received notices with timestamps. Do not alter production tags or dates for testing. Record provider, interval, last successful run and private evidence location after activation.

## GitHub backstop and incident response

[vanity-monitor.yml](../.github/workflows/vanity-monitor.yml) runs at minutes **07 and 37 of every UTC hour** and supports **Actions → Vanity metadata monitor → Run workflow**. It checks all fourteen URL variants using the reviewed checker, with read-only repository permissions and a five-minute job limit.

Confirm Actions is enabled, the workflow is on the default branch, and the schedule actor receives failed-workflow notifications. Dispatch it once and retain the run URL; subsequently observe a scheduled run. GitHub may delay or drop scheduled jobs and disables public-repository schedules after 60 days without activity. Notifications go to the user who last changed the cron schedule. The external service supplies primary monitoring. See [GitHub's schedule limitations](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule).

On failure, inspect the failed step first: checkout/toolchain failures need a successful rerun; DNS, certificate or metadata failures need investigation of the public endpoint. Compare tags with the reviewed baseline and inspect the latest Pages deployment. If deployment changed metadata, restore the last verified deployment, check all seven imports, and rerun the backstop. **Never regenerate the expected baseline from failed output.** Confirm external recovery delivery before closing the incident.

## Owner review calendar and completion evidence

The owner should add a private quarterly review on **15 January, April, July and October**, plus a review after any registrar, billing, nameserver, DNSSEC, Pages routing or account-security change. This document creates no calendar entries or external schedules.

At each review, check paid runway greater than one year, auto-renew/payment, transfer lock, contacts/MFA/recovery, validated DNSSEC, provider expiry notices, the latest external checks and notification test, and an enabled/recent GitHub schedule. Store dated evidence privately. T01 remains pending until the registrar evidence, successful DNSSEC validation, active external checks at the owner's selected 30-minute/two-hour/daily cadence, received test/recovery/expiry notices and observed GitHub runs are all available. The owner explicitly relaxed independent checks of the five other imports to two hours to target about $5/month; the GitHub backstop still checks every package twice an hour.
