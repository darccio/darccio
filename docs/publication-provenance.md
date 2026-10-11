<!--
publish: false
-->

# Local article publication metadata provenance

This private record documents the source evidence used for the 34 local article comments in T04: 33 public article candidates and the already-live draft at `/posts/delete-facebook-3/`. The evidence is each article's existing leading-comment `date_created` value, captured before the metadata edit. No date is derived from Git, archive year headings, report examples, image files, build times or inferred location.

The 31 values with an explicit offset are copied exactly into quoted RFC3339 `date` strings. Their original clock time and offset are retained, including the distinct dates and times of translated Delete Facebook entries. `date_created` remains unchanged as evidence. These source timestamps support the local metadata convention; they do not independently prove the first publication instant at another publisher or constitute approval for AT Protocol backfill.

`date_precision` records the available evidence:

- `instant`: the source contains a complete clock time and explicit UTC offset; `date` copies it exactly.
- `day`: the source contains only a calendar date; `date` is absent.
- `local-time`: the source contains a clock time without an offset; `date` is absent.

The three unresolved articles explicitly set `feed: false` and `atproto: false` while remaining published at their existing URLs. Do not substitute midnight, infer an offset from a language or presumed location, or coerce these values into an instant. A future correction needs primary publication evidence or an owner record establishing the missing precision; preserve that evidence here before adding `date` and reconsidering syndication exclusions.

## Unresolved evidence

| Source | Retained evidence | Precision | Missing evidence |
| --- | --- | --- | --- |
| [English liquid organizations](../en/posts/introduction-liquid-organizations/index.md) | `2013-07-01` | `day` | Publication time and UTC offset |
| [Spanish liquid organizations](../es/posts/introduccion-organizaciones-liquidas/index.md) | `2013-07-01` | `day` | Publication time and UTC offset |
| [Ruby/Go portal](../en/posts/portal-between-ruby-and-go/index.md) | `2022-04-05T22:42:00` | `local-time` | Publication UTC offset |

The English liquid-organizations body explicitly names July 1, 2013, corroborating the day only. Its [public article text](https://dario.cat/en/posts/introduction-liquid-organizations/) was also retrieved read-only on 2026-10-11 and supplied no additional time or offset evidence. The available [Ruby/Go public article text](https://dario.cat/en/posts/portal-between-ruby-and-go/) did not recover its missing offset. The read-only web retrieval of the Spanish translation was unavailable; its local source remains the evidence. These web retrievals included cached page text and are not a fresh production-head audit.

## Article classification, language and content

All 34 article comments explicitly set `article: true`, including the live draft and the three unresolved dates. This identifies an article independently of date precision. The two archive indexes, `posts/index.md` and `rizoma/index.md`, remain undated and without this article flag; T04 does not modify their bodies.

Each article has a one-sentence summary in its actual language and an explicit `language` value: 13 `ca`, 9 `en` and 12 `es`, including the Catalan draft. The summaries are based on complete article bodies, without adding current biographical claims. Existing nonempty tags are retained; topical tags are added to both liquid-organizations translations and to the draft's previously empty tag list. `en/.zas.yml` and `es/.zas.yml` set directory defaults using the same `language` key. The Catalan root default remains in `.zas/config.yml`.

All bytes after the closing leading metadata comment are preserved, including title headings, article prose, code, links, images, whitespace and final-newline state. Existing comment fields are preserved, including `date_created`, `description`, series, images, permalink, categories and `draft: true`; only the empty draft tag list is populated.

## Already-live draft and Rizoma differences

[Delete Facebook #3](../posts/delete-facebook-3/index.md) remains `draft: true` at its existing live URL and retains its body. Its sourced instant is recorded, but `feed: false` and `atproto: false` explicitly exclude it from syndication. No `publish: false` is added. No separately maintained Markdown twin convention exists in the source tree inspected for T04.

The four local Rizoma issues retain their local 2021 timestamps. The Substack feed evidence in [HANDOFF-FACTS.md](HANDOFF-FACTS.md) is separate: in particular, local issue 004 is `2021-09-26T19:00:00+02:00` (17:00 UTC), while the observed Substack feed gives 16:50 UTC. Neither publisher's timestamp silently replaces the other's. The three Un nou camí translations similarly retain `2024-11-27T22:22:00+01:00`, even though the body refers to the November 28 anniversary; narrative wording does not establish a replacement instant.

## Complete source inventory

Every value below is copied from that path's pre-edit leading-comment `date_created`. `date` is the identical quoted value for `instant` rows and is absent for `day` and `local-time` rows. The sourced draft instant counts among the 31 complete timestamps; after excluding the draft and the three unresolved values, 30 public candidates have complete local dates. Eligibility for any future feed or public backfill remains subject to that feature's scope and review rules.

| Article source | Source `date_created` / exact `date` when present | `date_precision` | Language |
| --- | --- | --- | --- |
| [posts/delete-facebook/index.md](../posts/delete-facebook/index.md) | `2018-03-28T23:26:00+02:00` | `instant` | `ca` |
| [posts/delete-facebook-1/index.md](../posts/delete-facebook-1/index.md) | `2018-04-01T16:53:15+02:00` | `instant` | `ca` |
| [posts/delete-facebook-2/index.md](../posts/delete-facebook-2/index.md) | `2018-04-07T17:04:09+02:00` | `instant` | `ca` |
| [posts/delete-facebook-3/index.md](../posts/delete-facebook-3/index.md) | `2018-04-08T15:00:56+02:00` | `instant` | `ca` |
| [posts/manifest-assemblees-constituents-contrapoder/index.md](../posts/manifest-assemblees-constituents-contrapoder/index.md) | `2014-11-26T08:08:46+01:00` | `instant` | `ca` |
| [posts/perque-temer-facebook/index.md](../posts/perque-temer-facebook/index.md) | `2010-05-19T22:01:00-04:00` | `instant` | `ca` |
| [posts/pseudo-estat/index.md](../posts/pseudo-estat/index.md) | `2025-02-14T12:39:00+01:00` | `instant` | `ca` |
| [posts/renda-basica/index.md](../posts/renda-basica/index.md) | `2024-04-10T21:09:00+02:00` | `instant` | `ca` |
| [posts/un-nou-cami/index.md](../posts/un-nou-cami/index.md) | `2024-11-27T22:22:00+01:00` | `instant` | `ca` |
| [en/posts/a-new-path/index.md](../en/posts/a-new-path/index.md) | `2024-11-27T22:22:00+01:00` | `instant` | `en` |
| [en/posts/deciwave/index.md](../en/posts/deciwave/index.md) | `2009-10-29T21:57:00-04:00` | `instant` | `en` |
| [en/posts/delete-facebook/index.md](../en/posts/delete-facebook/index.md) | `2018-03-28T23:26:00+02:00` | `instant` | `en` |
| [en/posts/delete-facebook-1/index.md](../en/posts/delete-facebook-1/index.md) | `2018-04-01T17:13:15+02:00` | `instant` | `en` |
| [en/posts/delete-facebook-2/index.md](../en/posts/delete-facebook-2/index.md) | `2018-04-08T12:06:19+02:00` | `instant` | `en` |
| [en/posts/geekvana/index.md](../en/posts/geekvana/index.md) | `2009-11-04T22:18:00-05:00` | `instant` | `en` |
| [en/posts/introduction-liquid-organizations/index.md](../en/posts/introduction-liquid-organizations/index.md) | `2013-07-01` | `day` | `en` |
| [en/posts/monitoring-synology-nas-with-datadog/index.md](../en/posts/monitoring-synology-nas-with-datadog/index.md) | `2026-08-16T15:59:44+02:00` | `instant` | `en` |
| [en/posts/portal-between-ruby-and-go/index.md](../en/posts/portal-between-ruby-and-go/index.md) | `2022-04-05T22:42:00` | `local-time` | `en` |
| [es/posts/blockchain-si-o-no/index.md](../es/posts/blockchain-si-o-no/index.md) | `2018-11-21T22:45:00+01:00` | `instant` | `es` |
| [es/posts/delete-facebook/index.md](../es/posts/delete-facebook/index.md) | `2018-03-28T23:26:00+02:00` | `instant` | `es` |
| [es/posts/delete-facebook-1/index.md](../es/posts/delete-facebook-1/index.md) | `2018-04-01T17:09:03+02:00` | `instant` | `es` |
| [es/posts/delete-facebook-2/index.md](../es/posts/delete-facebook-2/index.md) | `2018-04-08T11:53:34+02:00` | `instant` | `es` |
| [es/posts/diez-cosas-no-deberias-hacer-en-facebook/index.md](../es/posts/diez-cosas-no-deberias-hacer-en-facebook/index.md) | `2009-11-23T19:34:00-05:00` | `instant` | `es` |
| [es/posts/introduccion-organizaciones-liquidas/index.md](../es/posts/introduccion-organizaciones-liquidas/index.md) | `2013-07-01` | `day` | `es` |
| [es/posts/manifiesto-acampada-sol/index.md](../es/posts/manifiesto-acampada-sol/index.md) | `2011-05-19T18:19:00-04:00` | `instant` | `es` |
| [es/posts/manifiesto-red-neutral/index.md](../es/posts/manifiesto-red-neutral/index.md) | `2010-11-30T17:47:00-05:00` | `instant` | `es` |
| [es/posts/pequena-reflexion/index.md](../es/posts/pequena-reflexion/index.md) | `2013-04-28T23:50:52+02:00` | `instant` | `es` |
| [es/posts/porque-deberias-temer-facebook/index.md](../es/posts/porque-deberias-temer-facebook/index.md) | `2010-05-19T22:01:00-04:00` | `instant` | `es` |
| [es/posts/tejiendo-el-cambio/index.md](../es/posts/tejiendo-el-cambio/index.md) | `2013-10-18T22:28:00-04:00` | `instant` | `es` |
| [es/posts/un-nuevo-camino/index.md](../es/posts/un-nuevo-camino/index.md) | `2024-11-27T22:22:00+01:00` | `instant` | `es` |
| [rizoma/001/index.md](../rizoma/001/index.md) | `2021-08-08T10:00:00+02:00` | `instant` | `ca` |
| [rizoma/002/index.md](../rizoma/002/index.md) | `2021-08-22T17:00:00+02:00` | `instant` | `ca` |
| [rizoma/003/index.md](../rizoma/003/index.md) | `2021-09-05T12:00:00+02:00` | `instant` | `ca` |
| [rizoma/004/index.md](../rizoma/004/index.md) | `2021-09-26T19:00:00+02:00` | `instant` | `ca` |
