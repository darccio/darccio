<!--
publish: false
-->

# Site implementation decisions

The report defaults below govern implementation. Dario confirmed the public address below; exact publication dates and reviewed public backfill selections remain evidence requirements.

| ID | Decision | Choice | Status |
| --- | --- | --- | --- |
| D1 | Feed content | Full HTML in content:encoded and plain-text summary. | Adopted report default |
| D2 | Feed scope | Combined /feed.xml plus /en/feed.xml and /es/feed.xml when posts exist. | Adopted report default |
| D3 | Other feed formats | RSS 2.0 only; no Atom, JSON Feed or XSLT. | Adopted report default |
| D4 | Off-site feed items | Exclude external-only articles; preserve links in archives with labels. | Adopted report default |
| D5 | Rizoma canonical home | Substack for new issues; keep the four 2021 local issues and link https://rizoma.dario.cat/feed. | Adopted report default |
| D6 | Public email | Use cc@dario.cat in public contact links, the vCard, security.txt and funding metadata. Wrap visible HTML email links in Cloudflare's documented email_off comments to keep them readable without JS; the global dashboard toggle remains unmodified. | Owner confirmed on 2026-10-11; supersedes the initial source-derived address; targeted source opt-out prepared |
| D7 | Locality | Omit locality until Dario confirms it. | Adopted report default |
| D8 | Standard.site backfill | Prepare a reviewed allowlist from 2013 onward; exclude 2009–2010, drafts and external-only articles. Write no backfill records until that list is approved. | Default scope; exact list pending review |
| D9 | Standard.site full text | Include textContent in approved records; /ai explains that copies are outside robots.txt. | Adopted report default; recorded before public writes |
| D10 | Discovery preference | preferences.showInDiscover = true for the one root publication. | Adopted report default |
| D11 | Webmentions | Skip. | Adopted report default |
| D12 | Publishing authentication | App password only in Actions secrets; OAuth later. Never commit credentials. | Adopted report default; secrets not provisioned |
| D13 | Homepage writing claim | Point at actual writing channels without promising a cadence; Now names the owner-confirmed Corb and Cocoon projects. | Implemented in 1ad3087 |
| D14 | Generator | Extend Zas. Reassess Hugo + Hugo Modules only after two weekends of actual stalled implementation. | Adopted report default |
| D15 | Zas import path | Keep github.com/darccio/zas; /zas/ is a docs route only. | Adopted report default |
| D16 | AI policy | search=yes, ai-input=yes, ai-train=no; disallow /CV/; HTTP Content-Signal must agree. | Adopted report default |
| D17 | Site content licence | Original content with the necessary rights uses CC BY-SA 4.0; preserve CV restrictions, Lýd's CC BY 4.0, third-party terms and prior grants. Original implementation code retains the unchanged Unlicense in LICENSE-CODE. | Source scope clarified in d3ed333; deployment pending |

T06 and T23 remain optional and deferred. CV rendering in T17 requires reconciliation with the README’s explicit restriction before automated processing. Registrar, hosting and account settings need account access and actual verification; documenting a setting does not complete it.
