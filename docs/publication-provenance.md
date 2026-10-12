<!--
publish: false
-->

# Local article publication metadata provenance

This private record documents the source evidence used for the 34 local article comments in T04: 33 public article candidates and the already-live draft at `/posts/delete-facebook-3/`. The initial evidence is each article's existing leading-comment `date_created` value, captured before the metadata edit; dated follow-ups below record additional primary evidence and owner confirmations. No date is derived from Git, archive year headings, report examples, image files, build times or inferred location.

**Current status, 2026-10-12:** Ruby/Go's offset is owner-confirmed and the
English liquid-organizations timestamp is recovered from the original RSS.
The owner explicitly instructs using the English translation's timestamp for
the Spanish post; its clock time is an adopted assumption, not recovered
historical evidence. All 34 article sources now have complete explicit dates,
including the draft, or 33 public posts. The liquid-organizations dates use
2 July 2013, with previous source values retained separately.

The initial 31 values with an explicit offset were copied exactly into quoted RFC3339 `date` strings. Their original clock time and offset were retained, including the distinct dates and times of translated Delete Facebook entries. `date_created` was unchanged in that initial pass. The later Spanish day correction preserves its old value in `date_created_original`; the other original values remain intact. These source timestamps support the local metadata convention; they do not independently prove the first publication instant at another publisher or constitute approval for AT Protocol backfill.

`date_precision` records the available evidence:

- `instant`: `date` contains a complete clock time and explicit UTC offset. The initial pass copied source values exactly; follow-ups distinguish independently recovered timestamps from explicit owner confirmations or adopted assumptions.
- `day`: the source contains only a calendar date; `date` is absent.
- `local-time`: the source contains a clock time without an offset; `date` is absent.

The three initially unresolved articles explicitly set `feed: false` and `atproto: false` while remaining published at their existing URLs. The follow-ups below remove those precision exclusions after primary evidence, owner timezone confirmation and an explicit owner instruction for the Spanish clock time. Do not substitute midnight or infer an offset from a language or presumed location. Distinguish a primary-source instant from an owner-adopted assumption when completing partial dates.

## Initially unresolved evidence

| Source | Retained evidence | Precision | Missing evidence |
| --- | --- | --- | --- |
| [English liquid organizations](../en/posts/introduction-liquid-organizations/index.md) | `2013-07-01` | `day` | Publication time and UTC offset |
| [Spanish liquid organizations](../es/posts/introduccion-organizaciones-liquidas/index.md) | `2013-07-01` | `day` | Publication time and UTC offset |
| [Ruby/Go portal](../en/posts/portal-between-ruby-and-go/index.md) | `2022-04-05T22:42:00` | `local-time` | Publication UTC offset |

The English liquid-organizations body explicitly names July 1, 2013, corroborating the day only. Its [public article text](https://dario.cat/en/posts/introduction-liquid-organizations/) was also retrieved read-only on 2026-10-11 and supplied no additional time or offset evidence. The available [Ruby/Go public article text](https://dario.cat/en/posts/portal-between-ruby-and-go/) did not recover its missing offset. The read-only web retrieval of the Spanish translation was unavailable; its local source remains the evidence. These web retrievals included cached page text and are not a fresh production-head audit.

## Follow-up primary-source research, 2026-10-11

The later raw-HTML checks below succeeded for all three live articles, including the Spanish translation that the earlier cached retrieval could not fetch. No check recovered the missing original publication precision. The retained `date_created` values, absent `date` fields, `date_precision` values and syndication exclusions remain unchanged.

| Checked source | Outcome and evidentiary limit |
| --- | --- |
| [Live English liquid organizations](https://dario.cat/en/posts/introduction-liquid-organizations/), [Spanish liquid organizations](https://dario.cat/es/posts/introduccion-organizaciones-liquidas/) and [Ruby/Go portal](https://dario.cat/en/posts/portal-between-ruby-and-go/) | Fresh HTTPS GETs returned 200. Raw HTML supplied canonical URLs and `dt-updated`, but no `dt-published`, `datePublished` or `article:published_time`. Updated timestamps do not establish publication. The English body still corroborates July 1, 2013 only. |
| Original Thinkship [English](http://thinkship.cc/en/introduction-to-liquid-organizations/) and [Spanish](http://thinkship.cc/introduccion-a-las-organizaciones-liquidas/) URLs | HTTPS requests failed DNS resolution. The original English URL is independently linked by [Dario's July 10, 2013 HN comment](https://news.ycombinator.com/item?id=6017748); the comment date is not the article's publication instant. |
| [English Wayback availability](https://archive.org/wayback/available?url=thinkship.cc/en/introduction-to-liquid-organizations/&timestamp=20130701) | API returned 200 and identified the [2013-07-08 09:52:59 capture](https://web.archive.org/web/20130708095259/http://thinkship.cc:80/en/introduction-to-liquid-organizations/). Both normal and raw replay returned 429, as did the [exact-URL CDX query](https://web.archive.org/cdx/search/cdx?url=thinkship.cc/en/introduction-to-liquid-organizations/&output=json&filter=statuscode:200&collapse=digest), so archived article metadata was unavailable. The capture timestamp must not become the publication date. |
| Spanish Wayback availability: [bare-host URL](https://archive.org/wayback/available?url=thinkship.cc/introduccion-a-las-organizaciones-liquidas/&timestamp=20130701), [explicit HTTP URL](https://archive.org/wayback/available?url=http://thinkship.cc/introduccion-a-las-organizaciones-liquidas/&timestamp=20130703), [original feed](https://archive.org/wayback/available?url=http://thinkship.cc/feed/&timestamp=20130703) | Each API request returned 200 with an empty `archived_snapshots` result. This is an outcome for these exact requests, not proof that no archive exists elsewhere. |
| Old [da.rio.hn Portal URL](https://da.rio.hn/en/posts/portal-between-ruby-and-go/index.html) and its Wayback availability [bare-host](https://archive.org/wayback/available?url=da.rio.hn/en/posts/portal-between-ruby-and-go/index.html&timestamp=20220405), [HTTPS index](https://archive.org/wayback/available?url=https://da.rio.hn/en/posts/portal-between-ruby-and-go/index.html&timestamp=20220906), [HTTPS directory](https://archive.org/wayback/available?url=https://da.rio.hn/en/posts/portal-between-ruby-and-go/&timestamp=20220906) | Origin returned 522. All three availability requests returned 200 with empty `archived_snapshots`. No original offset was recovered. |
| Dario's [DEV Portal copy](https://dev.to/dcc/a-portal-between-ruby-and-go-using-ffi-48gn) and [public article API](https://dev.to/api/articles/1185956) | API returned 200 with author `Dario Castañé` / `dcc`, `canonical_url: https://dario.cat/en/posts/portal-between-ruby-and-go/` and `published_at: 2022-09-06T09:48:50Z`. The article identifies dario.cat as its original publisher. This establishes the later DEV copy's instant; it neither supplies an offset for the local April 5 clock value nor proves a replacement original publication date. |
| Dario's [Medium Spanish copy](https://medium.com/@darccio/introducci%C3%B3n-a-las-organizaciones-l%C3%ADquidas-de704750308c) | Indexed author-copy text reports May 30, 2016; direct GET returned 403 and browser retrieval was denied by robots.txt. This later date does not establish either original 2013 translation's time or offset. |
| Historical source comments | The article metadata in [English liquid organizations at `907efbe`](https://github.com/darccio/darccio/blob/907efbe/en/posts/introduction-liquid-organizations/index.md) and [Portal at `835dd80`](https://github.com/darccio/darccio/blob/835dd80/en/posts/portal-between-ruby-and-go/index.md) already contains the same partial values. Commit dates were not used as publication dates. |

One secondary reference, the [2014 ESADE/PwC bibliography](https://www.pwc.es/es/fundacion/assets/eficiencia-impacto-social-2014.pdf), located the original Spanish Thinkship URL but gives July 2, 2013. It conflicts with the retained July 1 source day and supplies no clock time or offset, so it is recorded as a discrepancy rather than a correction. Search snippets, newsletter inclusion dates and aggregator submission dates were used only to locate primary URLs.

Resolving the liquid-organizations entries needs each original publisher's full publication timestamp with timezone, such as an original WordPress export containing local and GMT publication fields, an archived feed item, or an owner publication record. Resolving Portal needs an owner record or original publisher metadata establishing the UTC offset of the retained `2022-04-05T22:42:00`, or independently establishing a corrected original publication instant. The DEV copy's timestamp alone does not supply that evidence. No timezone is inferred from language, residence, Git configuration or another post.

## Owner timezone confirmation, 2026-10-12

The owner confirms **Europe/Madrid** for the Ruby/Go portal's retained
`2022-04-05T22:42:00` clock value. The IANA timezone rules give `+02:00` on that
date, so its explicit publication instant is now
**`2022-04-05T22:42:00+02:00`**. The original `date_created` is retained unchanged;
`date` holds the completed value and `date_precision` is `instant`.

The two archive entries now expose the same `dt-published` instant without the
obsolete unknown-timezone note. The temporary `feed: false` and `atproto: false`
exclusions are removed; the post can enter RSS and the offline Standard.site
candidate inventory. The reviewed publication allowlist remains empty, so this
metadata correction authorizes no public AT Protocol write. The complete
article body is unchanged. Both liquid-organizations publication times remain
unresolved at this step.

## Original Thinkship publication evidence recovered, 2026-10-12

The [archived original English RSS feed](https://web.archive.org/web/20131104031318id_/http://thinkship.cc/en/feed/)
is now retrievable with HTTP 200, superseding the earlier unavailable replay.
Its item matches the English article title, original URL
`http://thinkship.cc/en/introduction-to-liquid-organizations/`, author Dario
Castañé and WordPress GUID `http://thinkship.cc/?p=848`. Its publication field is
`Tue, 02 Jul 2013 08:00:39 +0000`, establishing **`2013-07-02T08:00:39Z`**.
The site stores the equivalent **`2013-07-02T10:00:39+02:00`**, expressed in
Europe/Madrid. The November capture date is not the
publication date.

The original `date_created: "2013-07-01"` is retained for historical evidence,
but the newly recovered `date` and archive entries use **2 July 2013**. The
English body's July 1 wording describes the release of the analysis; it does
not override the publisher's RSS publication field. The complete body remains
unchanged. Its partial-date syndication exclusions are removed, without
approving any Standard.site record.

The [archived original Spanish article](https://web.archive.org/web/20130708033121id_/http://thinkship.cc/introduccion-a-las-organizaciones-liquidas/)
also returns HTTP 200, identifies WordPress post `829`, and displays **2 July
2013** as its publication day. It provides no publication clock time or UTC
offset. This primary evidence corroborates the previously noted secondary
day discrepancy but cannot supply a complete Spanish timestamp. Its exact
root-feed availability query returned no capture, raw-feed requests returned
503 and the original WordPress public API reported disabled access. These
limited checks do not establish that no other backup exists. The English
translation's time is not independent evidence of the Spanish clock time;
the separate owner instruction below explicitly adopts it.

The Spanish article's `date_created` now records the verified known day
**`2013-07-02`**. Its previous `2013-07-01` value is retained as
`date_created_original` and in the initial inventory below. Its article body
is unchanged. The following owner instruction completes the clock time;
the archived HTML alone establishes only the calendar day.

The retrieved English RSS response has SHA-256
`2edf4c4f4f068b483cec3bb6da6f97aefeb93387435e1cca74cc5b96a6ab050d`;
the Spanish HTML response has SHA-256
`c79bf3c40b893728f2680113e2837cd771ce3222dd50376e65dbc1b67c1c1221`.
The article bodies are omitted from this evidence record; source links,
identifying fields and response hashes document the findings.

## Owner-adopted Spanish timestamp, 2026-10-12

When asked for the remaining Spanish clock time and timezone, the owner
explicitly instructs: **"Assume the same timestamp"**, referring to the recovered
English translation timestamp. Its `date` is therefore set to
**`2013-07-02T10:00:39+02:00`** and its `date_precision` is `instant`, describing
the complete adopted value. The source `date_source` field records the owner
instruction. The Spanish clock time is an approved assumption; no archived
feed, original export or independent clock-time verification is claimed.

The Spanish post and both archive entries use this adopted instant consistently.
Its precision exclusions are removed. All 33 public posts can enter their
configured RSS feeds, and 26 meet the existing Standard.site candidate cutoff.
The draft remains excluded. The reviewed Standard.site allowlist remains empty;
restoring date completeness and candidacy does not approve any public record.

## Article classification, language and content

All 34 article comments explicitly set `article: true`, including the live draft and the three initially unresolved dates. This identifies an article independently of date precision. The two archive indexes, `posts/index.md` and `rizoma/index.md`, remain undated and without this article flag. The initial T04 pass left archive bodies unchanged; the follow-up corrects publication markup in the combined and language-specific archives.

Each article has a one-sentence summary in its actual language and an explicit `language` value: 13 `ca`, 9 `en` and 12 `es`, including the Catalan draft. The summaries are based on complete article bodies, without adding current biographical claims. Existing nonempty tags are retained; topical tags are added to both liquid-organizations translations and to the draft's previously empty tag list. `en/.zas.yml` and `es/.zas.yml` set directory defaults using the same `language` key. The Catalan root default remains in `.zas/config.yml`.

All bytes after the closing leading metadata comment were preserved in the initial T04 pass, including title headings, article prose, code, links, images, whitespace and final-newline state. Existing comment fields were preserved, including `date_created`, `description`, series, images, permalink, categories and `draft: true`; only the empty draft tag list was populated. The later Spanish `date_created` correction and its retained original value are documented above.

## Already-live draft and Rizoma differences

[Delete Facebook #3](../posts/delete-facebook-3/index.md) remains `draft: true` at its existing live URL and retains its body. Its sourced instant is recorded, but `feed: false` and `atproto: false` explicitly exclude it from syndication. No `publish: false` is added. No separately maintained Markdown twin convention exists in the source tree inspected for T04.

The four local Rizoma issues retain their local 2021 timestamps. The Substack feed evidence in [HANDOFF-FACTS.md](HANDOFF-FACTS.md) is separate: in particular, local issue 004 is `2021-09-26T19:00:00+02:00` (17:00 UTC), while the observed Substack feed gives 16:50 UTC. Neither publisher's timestamp silently replaces the other's. The three Un nou camí translations similarly retain `2024-11-27T22:22:00+01:00`, even though the body refers to the November 28 anniversary; narrative wording does not establish a replacement instant.

## Initial source inventory, 2026-10-11

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
