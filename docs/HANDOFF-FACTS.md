<!--
publish: false
-->

# Site handoff facts

This is an audit baseline, not a statement that subsequent implementation or deployment is complete. Local inspection began at **2026-10-11T07:56:06Z** (09:56:06 Europe/Madrid) at site commit `6d9194ed6ee1557967e386ad65e688c2c30583b2`. Public HTTPS observations below were collected between **2026-10-11T07:57:07Z and 2026-10-11T07:58:57Z**. Recheck them after deployment. Existing untracked `docs/go-vanity-url.md` was read and left unchanged; its SHA-256 was `b1fca26739cf9d022923a2c2ebe7445b60ac10f8f89126ca7b654e946201de89`.

## Local source and build

- The home source is [`index.html`](../index.html), not `index.md`. It sets `body_class: h-card vcard`, a Catalan description, a Gravatar image and a vCard alternate.
- [`.zas/config.yml`](../.zas/config.yml) sets `site.host: dario.cat`, `site.baseurl: https://dario.cat`, `site.language: ca`, deployment directory `.zas/deploy`, and layout `.zas/layout.html`. Its MIME mappings are built-in `html`, `markdown` and `plain`. It contains no feed, sitemap or external plugin configuration.
- [`.zas/layout.html`](../.zas/layout.html) already renders an HTML `lang` attribute from the page's resolved language and conditional body classes. The contact partial already has `rel="me"` on Bluesky, Fediverse, GitHub, LinkedIn and Substack links. Public contact email is `d@rio.hn`, also present in the vCard.
- English and Spanish post directories exist, but no tracked `en/.zas.yml` or `es/.zas.yml` language overrides were present. [`lyd/.zas/config.yml`](../lyd/.zas/config.yml) declares English and the historical `http://dario.cat/lyd` base URL; this nested configuration must not be assumed to govern the root build. `lyd/index.md` is a liquid-organization blueprint, not a vanity import page.
- [`.zas/i18n.yml`](../.zas/i18n.yml) has a duplicate `en` key in `social-networks`: `Social networks` and `Redes sociales`. The root agent's isolated baseline build with fetched upstream Zas failed on this duplicate YAML mapping key. Changing the second key to `es` is the evident correction; this audit does not modify it.
- At inspection, the repository had 94 tracked files and 45 tracked Markdown sources. No tracked `.github` workflow, `_headers`, `_redirects`, root `.well-known` assets, feed, sitemap or `llms` artifact was found. These are local observations, not proof of production host settings.
- [`.zas/Makefile`](../.zas/Makefile) runs `zas generate -full` for `generate`. Its `publish` target installs `github.com/darccio/zas/cmd/zas@v0.0.6` into `/tmp`, runs incremental `/tmp/zas generate`, then overwrites the seven generated vanity pages with their raw source HTML. It does not itself upload the output.
- The installed `/home/dario/go/bin/zas` reported `zas version v0.0.7 go1.26.5`; `zas generate -h` exposes `-full`, `-no-plugins` and `-verbose`. This installed binary is distinct from the publication recipe's v0.0.6 pin. The read-only local Zas checkout was clean at `9c8c0b221a2fac869e13c53e8246577857c00500` when inspected.
- [`.zas/Procfile`](../.zas/Procfile) uses `caddy file-server --root ./zas/deploy --listen 127.0.0.1:1304`; `./zas/deploy` differs from the configured `.zas/deploy`. `make serve` invokes `hivemind .zas/Procfile`.
- The only source embeds found were the built-in `text/html` contact partial in home and `contact.md`. No `application/zas+` script tag or site-owned external plugin was found. Zas resolves `zas <name>` to `zs<name>` on PATH, external MIME embeds to `mzs<name>`, and content script plugins to `zs<name>`. Untrusted document builds must use **`zas generate -full -no-plugins`** with a version supporting that flag; this disables content-triggered external plugins while preserving built-in handlers. Installed/deployed plugin inventories are not established by the site's source configuration.
- A subsequent isolated build with current upstream Zas, after correcting only the duplicate YAML key in a temporary copy, refused all seven head-only vanity fragments because their meta elements would be lost during layout rendering. `template: false` disables Go templating only; it is not a passthrough. The T02 build wrapper must exclude raw fragments from generator input and restore them unchanged before validation.
- A bounded live root-agent probe of `https://dario.cat/posts/delete-facebook-3/` returned the actual `Delete Facebook #3` body despite `draft: true` in source. Preserve this existing URL; exclude it from feeds and AT Protocol rather than removing it implicitly.
- The pre-existing ignored `.zas/deploy` tree was not a trustworthy production snapshot: six vanity outputs existed without `go-import`, while `cocoon/index.html` was absent. This supports keeping explicit vanity verification after any build.
- [`LICENSE`](../LICENSE) contains the Unlicense; README and the translated footer text state CC BY-SA 4.0. This inconsistency needs a deliberate ownership decision.

## Complete vanity metadata baseline

All seven local raw files and their live `/<name>/?go-get=1` responses were byte-for-byte identical. Each response returned HTTP 200 and `text/html; charset=utf-8`. Each `/<name>?go-get=1` returned HTTP **308** to `/<name>/?go-get=1`, preserving the query. Preserve these trailing-slash URLs and exact import metadata.

| Raw source / live path | Exact `go-import` content |
| --- | --- |
| `mergo/index.html` / [`/mergo/`](https://dario.cat/mergo/?go-get=1) | `dario.cat/mergo git https://github.com/imdario/mergo` |
| `darna/index.html` / [`/darna/`](https://dario.cat/darna/?go-get=1) | `dario.cat/darna git https://github.com/darccio/darna` |
| `cocoon/index.html` / [`/cocoon/`](https://dario.cat/cocoon/?go-get=1) | `dario.cat/cocoon git https://github.com/darccio/cocoon` |
| `errors/index.html` / [`/errors/`](https://dario.cat/errors/?go-get=1) | `dario.cat/errors git https://github.com/darccio/errors` |
| `xdr/index.html` / [`/xdr/`](https://dario.cat/xdr/?go-get=1) | `dario.cat/xdr git https://github.com/darccio/xdr` |
| `awaken/index.html` / [`/awaken/`](https://dario.cat/awaken/?go-get=1) | `dario.cat/awaken git https://tangled.org/dario.cat/awaken` |
| `x/index.html` / [`/x/`](https://dario.cat/x/?go-get=1) | `dario.cat/x git https://tangled.org/dario.cat/x` |

**No `go-source` tag was present in any of these seven sources or live responses.** There are no additional raw vanity pages in the inspected site. Six pages contain a zero-second HTML refresh to `https://pkg.go.dev/dario.cat/<name>`; `xdr` only contains the import meta tag. In particular, `mergo` retains the historical `imdario` repository identity, whereas the observed `cocoon` identity is `darccio`. Do not substitute identities based on a report or current profile name.

## Live responses and head metadata

Requests used GET with curl, certificate verification enabled, a five-second connection timeout, a twelve-second total timeout, and at most five redirects. Sandbox DNS initially failed; the read-only network run succeeded. Raw responses and curl metadata were retained locally in `/tmp/dario-site-plan/live-T00/` for this work session. These temporary files are not part of the repository. Web access also retrieved the [home page](https://dario.cat/), but could not retrieve several text/feed endpoints; the curl observations below provide the status, raw head and headers.

| URL(s), relative to `https://dario.cat` | Observed response |
| --- | --- |
| `/`, `/posts/` | 200; `text/html; charset=utf-8`; expected page content |
| [`/robots.txt`](https://dario.cat/robots.txt) | 200; `text/plain; charset=utf-8`; exactly `User-agent: *` followed by `Allow: /` (no sitemap or CV exclusion) |
| [`/sitemap.xml`](https://dario.cat/sitemap.xml) | 200; `text/html; charset=utf-8`; homepage HTML, not a sitemap |
| [`/llms.txt`](https://dario.cat/llms.txt), `/llms-full.txt`, `/en/llms.txt`, `/es/llms.txt` | 200; `text/html; charset=utf-8`; homepage HTML, not agent context text |
| `/feed.xml`, `/rss.xml`, `/atom.xml`, `/feed.json`, `/en/feed.xml`, `/es/feed.xml`, `/en/rss.xml`, `/es/rss.xml`, `/rss/`, `/feeds/` | 200; `text/html; charset=utf-8`; homepage HTML, not feeds or a feed index |
| [`/.well-known/atproto-did`](https://dario.cat/.well-known/atproto-did), [`/.well-known/security.txt`](https://dario.cat/.well-known/security.txt), `/.well-known/site.standard.publication` | 200; `text/html; charset=utf-8`; homepage HTML, not the requested machine resource |
| `/.well-known/webfinger?resource=acct:d@rio.hn` | 200; `text/html; charset=utf-8`; homepage HTML, not JRD |

For every homepage fallback above, the body matched the home response after normalizing Cloudflare's per-response email-obfuscation hex values. This is observed **soft fallback behavior**; HTTP 200 alone is insufficient verification. The configuration responsible for it is unknown.

The home, posts and vanity import responses exposed `Server: cloudflare`, `X-Content-Type-Options: nosniff`, and `Cache-Control: public, max-age=0, must-revalidate`; observed cache status was `DYNAMIC`. Robots exposed `Cache-Control: public, max-age=14400, must-revalidate` and `CF-Cache-Status: REVALIDATED`.

The live home has `lang="ca"`, body class `h-card vcard`, an `og:url` of `https://dario.cat/index.html`, `og:type="profile"`, `og:locale="ca"`, a Gravatar OG image and a vCard alternate. The live posts index has `og:url="https://dario.cat/posts/index.html"` and also uses `og:type="profile"`. Neither inspected head advertises a feed, canonical link, hreflang alternate or JSON-LD block. The live [English article](https://dario.cat/en/posts/a-new-path/), [Spanish article](https://dario.cat/es/posts/un-nuevo-camino/) and [Lýd page](https://dario.cat/lyd/) all declared **`lang="ca"` and `og:locale="ca"`**, despite their source content/language intentions.

The live home replaces the plain email link with `/cdn-cgi/l/email-protection#...`, a `__cf_email__` span, and loads `/cdn-cgi/scripts/5c5dd728/cloudflare-static/email-decode.min.js` in the body. The local contact partial is plain HTML. The local layout also contains an `onload` handler for CSS preload with a `<noscript>` stylesheet fallback. Any claim of a fully JavaScript-free public page must account for both observations.

## Hosting and WebFinger uncertainty

Cloudflare is observed at the public edge. **The owner confirmed Cloudflare Pages with Git integration during this session.** In a subsequent owner reply on 2026-10-11, the production trigger was confirmed as pushes to `main`, the build command as `make -f .zas/Makefile publish`, output as `.zas/deploy`, root directory as blank and build comments as enabled. This command already enters the guarded assembly path; no dashboard change is needed to adopt committed build updates. Preview policy, function/Worker routing and dashboard redirect rules remain unconfirmed. A public CNAME lookup returned no CNAME answer and a Cloudflare SOA; that does not identify the origin. No deployment/dashboard credentials were inspected.

A subsequent read-only GitHub API inspection on 2026-10-11 found check run `113699937627` for remote-main SHA `6d9194ed6ee1557967e386ad65e688c2c30583b2`: name `Cloudflare Pages`, app ID `85455` / slug `cloudflare-workers-and-pages`, completed with success and title `Deployed successfully`. Its dashboard URL identifies Pages project `dario`. Suite `102668096686` confirms `head_branch: main` and the same SHA. GitHub deployment records and commit statuses were empty. The new post-deploy workflow polls this actual provider result for the pushed SHA; it does not use a build-only workflow as evidence of deployment. Its first actual invocation remains unverified until pushed.

[`.workers/webfinger/wrangler.toml`](../.workers/webfinger/wrangler.toml) only sets `name = "webfinger"` and `compatibility_date = "2022-11-27"`; it declares no `ASSETS` binding or routes. The TypeScript environment is empty. The worker matches only `resource=acct:d@rio.hn`, then emits a static `application/jrd+json; charset=utf-8` document with subject `acct:dario@mastodont.cat`, profile/activity/subscribe links; it otherwise returns 404. Deployment and current route ownership of this worker are not confirmed. A live request to `https://rio.hn/.well-known/webfinger?resource=acct:d@rio.hn` redirected to the same path/query on `dario.cat`, whose response was the homepage fallback. Do not overwrite this worker to add site asset negotiation.

## Public AT Protocol identity

The [public Bluesky handle resolver](https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle=dario.cat) returned HTTP 200 JSON:

```json
{"did":"did:plc:aj77r5uwt72o6oimdjfplqoz"}
```

A [public DNS TXT lookup](https://cloudflare-dns.com/dns-query?name=_atproto.dario.cat&type=TXT) independently returned `_atproto.dario.cat` = `did=did:plc:aj77r5uwt72o6oimdjfplqoz` (TTL 300). The [public PLC DID document](https://plc.directory/did:plc:aj77r5uwt72o6oimdjfplqoz) returned HTTP 200, `application/did+ld+json; charset=utf-8`, with `alsoKnownAs: ["at://dario.cat"]` and an `AtprotoPersonalDataServer` service endpoint of **`https://eurosky.social`**. This is a published service declaration, not verification of PDS health, authentication, or Standard.site records. No authenticated PDS request, publication bootstrap, record write or deletion was performed. No Standard.site publication AT URI was established by this audit. Site deployment must remain independent of PDS availability.

## Substack feed

[Rizoma](https://rizoma.dario.cat/) returned HTTP 200 HTML. Its [feed](https://rizoma.dario.cat/feed) returned HTTP 200, **`application/xml; charset=utf-8`**, `Cache-Control: no-cache`, and parsed as RSS 2.0 with **five items**. The channel title is `Rizoma`, channel link is `https://rizoma.dario.cat`, and its description identifies a personal Catalan newsletter. The channel nevertheless declares **`language: en`**. Its request-time `lastBuildDate` was `Sun, 11 Oct 2026 07:57:09 GMT`; this is not a post publication date.

| Feed item | Source feed publication date |
| --- | --- |
| [Un nou camí](https://rizoma.dario.cat/p/un-nou-cami) | `Thu, 28 Nov 2024 08:10:12 GMT` |
| [Rizoma #004](https://rizoma.dario.cat/p/rizoma-004-popurri-denllacos) | `Sun, 26 Sep 2021 16:50:00 GMT` |
| [Rizoma #003](https://rizoma.dario.cat/p/rizoma-003-estat-del-catala-a-luci) | `Sun, 05 Sep 2021 10:00:00 GMT` |
| [Rizoma #002](https://rizoma.dario.cat/p/rizoma-002-canvi-climatic-o-fallida) | `Sun, 22 Aug 2021 15:00:00 GMT` |
| [Rizoma #001](https://rizoma.dario.cat/p/rizoma-001) | `Sun, 08 Aug 2021 08:00:00 GMT` |

These dates are evidence of Substack's feed values. They do not establish first publication times on this site and must not silently replace local metadata.

## Public domain registration evidence

A root-agent HTTPS query through RDAP on 2026-10-11 reached the .cat [registry data for `dario.cat`](https://rdap.nic.cat/rdap/domain/dario.cat) (registry database update `2026-10-11T08:08:07.582Z`). The registry reports expiration at **2027-06-01T07:05:04.361Z**, status `active`, and `secureDNS.delegationSigned: false`. Its nameservers are `ray.ns.cloudflare.com` and `jean.ns.cloudflare.com`. The expiry is less than one year from this audit; it does not satisfy T01’s more-than-one-year acceptance requirement. Auto-renew, payment validity and registrar lock must be checked in the registrar account; absence of a transfer-lock status in this response is not proof of a disabled setting. DNSSEC requires a real validation check after activation. No renewal purchase or DNS setting was changed.

## Reverification and owner inputs

Inspect response content as well as status and headers. A bounded command for any individual URL is:

```sh
curl --silent --show-error --location --max-redirs 5 \
  --connect-timeout 5 --max-time 12 --dump-header /tmp/site-headers.txt \
  --output /tmp/site-body.txt 'https://dario.cat/sitemap.xml'
```

Before host-specific changes or public publishing, establish the remaining Pages deployment settings, email-obfuscation setting, desired WebFinger routing, approved content/licensing policy, and Standard.site publication/account authorization. Exact historical publication times that are absent from source remain unknown. This audit requires no owner approval to record the available facts, and records unavailable settings explicitly.
