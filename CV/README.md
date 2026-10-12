<!--
publish: false
-->

# CV maintenance

`MAC.json` remains the public machine-readable CV at `/CV/MAC.json`. The guarded
site build generates `/cv/` from that same file using `.zas/resume.yml`, so the
HTML does not need a separate content update. The professional copy is in English;
the site's default language remains Catalan.

The owner authorized this update and its build-time rendering and validation on
12 October 2026. The existing restriction on other automatic processing remains
in place. Neither the LinkedIn PDF nor its extracted text belongs in this
repository. The HTML projects only the public profile, selected email,
experience, education, skills and languages; it omits career preferences and
other contact fields. `/CV/` and `/cv/` remain excluded from crawling, feeds,
Standard.site, Markdown companions and the agent text collections.

## Update sources

The update uses [Sifa](https://sifa.id/p/dario.cat) and the owner-supplied LinkedIn
`Profile.pdf`, checked on 12 October 2026. Both profiles agree on Datadog from
August 2023, Loyal Guru management from April 2020 through July 2023, Coopanio
from April 2021, and IGD Tech & Drinks ending in January 2024. Sifa adds Full
Stack Vallès from March 2026 and the civic involvement entries. Both identify
Terrassa as the profile locality, independently of the locations of past jobs.

For Associació Blockchain Catalunya, the mutually agreeing career section and
PDF dates (February 2019 through June 2021) take precedence over Sifa's
involvement duplicate. The same rule preserves the agreed TICAnoia career
timeline. Freedom to Share's promoter role and the older developer project have
different scopes and retain their separate dates. Sifa's Asembleo and Zas start
months replace the older export values; the other historical project entries
remain available.

Existing credential details are retained. The Sifa-issued credentials are marked
achieved, which records completion rather than current validity. AWS entries
retain the earlier CV's December 2020 expiry and are explicitly historical.
The Ceina course retains its earlier January–March 2007 attendance interval,
with March 2007 completion confirmed by Sifa; it remains an unaccredited course.
Dynamic repository-usage counts are omitted from refreshed copy. Existing career
preferences remain in the preserved MAC file without being exposed in the HTML.

## Date precision

MAC v0.3 requires complete `yyyy-mm-dd` dates, but the supplied profiles usually
provide months or years. As in the earlier MAC file, day `01` stores a month;
January `01` stores a year where no month is supplied. These normalized values
do not establish an actual first-day start or finish.

The renderer displays months by default. `date_precision` in `.zas/resume.yml`
uses JSON pointers to display year-only evidence for the public-speaking and
Inceptum starts, the three formal qualifications and the legal-forensics course.
Older UOC start/end months remain in MAC but the HTML shows the years confirmed
by the new sources. Certification `startDate` stores the issue month used by the
original export; the HTML labels it as issued. Do not invent attendance dates or
claim current certification from an issue date.

MAC normally uses a study's `finishDate` for completion. The earlier AWS record
uses it for credential expiry instead; `expiry_dates` explicitly selects that
field in `.zas/resume.yml`. Other certification finish dates describe completion.

## Build and review

Edit `MAC.json`, then update `settings.lastUpdate`, `.zas/resume.yml`'s review
date and any JSON pointers affected by reordered entries. Run
`make -f .zas/Makefile publish` and review `/cv/` together with the generated
microformats checks. The build keeps the JSON bytes intact and validates the
resume before replacing the previous deployment. Removing the opt-in manifest
disables rendering; it does not remove the existing JSON.
