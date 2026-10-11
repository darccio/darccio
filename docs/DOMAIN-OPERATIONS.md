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

## Registrar protection in SW Hosting

1. Sign in to SWPanel and open **Domains and SSL → Domain Portfolio → dario.cat**. Confirm that this is the account controlling registration and that owner/admin contact email reaches you. Review MFA and recovery access.
2. In the domain actions, use **Enable Auto-renewal** if needed; verify the resulting enabled state. Check the billing method, its expiry and any required balance privately. SW Hosting documents the controls in its [domain actions guide](https://www.swhosting.com/en/comunidad/manual/domains).
3. Use **Block Domain** if needed and verify the resulting transfer-lock state. Keep a dated dashboard record; absence of a lock flag in RDAP is insufficient. See [SW Hosting's lock procedure](https://www.swhosting.com/en/comunidad/manual/how-to-block-a-domain).
4. Use **Renew domain**, review the years and price in the renewal cart, and have the owner complete payment so the paid expiration is more than one year ahead of the action date. Retain the receipt, new dashboard expiry and updated registry response. Auto-renew alone does not prove paid runway. See [SW Hosting's renewal procedure](https://www.swhosting.com/en/comunidad/manual/how-to-renew-a-domain-from-your-swpanel).

## DNSSEC with Cloudflare authoritative DNS

The current nameservers belong to Cloudflare. SW Hosting's [hosting DNSSEC switch](https://www.swhosting.com/en/comunidad/manual/how-to-activate-dnssec-on-my-hosting) requires its own authoritative nameservers and excludes Cloudflare; use the registrar's external DS support for this zone instead.

1. Save the existing parent DS response and TTL, nameservers and Cloudflare zone settings privately. Confirm that the registrar can publish Cloudflare's DS values to the `.cat` parent before activation.
2. In Cloudflare, select **dario.cat → DNS → Settings → DNSSEC → Enable DNSSEC**. Keep the existing nameservers. Copy the generated key tag, algorithm, digest type and digest exactly.
3. Publish that DS through SW Hosting's registrar DS management. If no such control is available, prepare an administrative support ticket requesting publication of those exact four values for `dario.cat` at the `.cat` parent, with nameservers unchanged. Account access or support confirmation is required; this repository cannot perform the change.
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

Choose an external service that runs HTTPS metadata checks every **30 minutes**, independently of GitHub, Pages builds and PDS. Configure each of these names with both `/<name>?go-get=1` and `/<name>/?go-get=1`:

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

At each review, check paid runway greater than one year, auto-renew/payment, transfer lock, contacts/MFA/recovery, validated DNSSEC, provider expiry notices, the latest external checks and notification test, and an enabled/recent GitHub schedule. Store dated evidence privately. T01 remains pending until the registrar evidence, successful DNSSEC validation, active external 30-minute checks, received test/recovery/expiry notices and observed GitHub runs are all available.
