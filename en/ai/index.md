<!--
entry: true
language: en
description: "dario.cat's preferences for search, AI use, training and personal data."
updated: 2026-10-12T22:24:08+02:00
-->

<h1 class="p-name">AI and crawlers</h1>

## Use preferences

This site's policy allows search indexing and use of its content as context for answering questions with AI. I ask that the content not be used to train AI models. These preferences are declared in [robots.txt](/robots.txt):

```text
search=yes, ai-input=yes, ai-train=no
```

These signals are preferences for crawlers to respect; they do not technically prevent access to or copying of content. The [licences and exceptions](/LICENSE) must also be respected.

## Personal data

The JSON under `/CV/` and the HTML version at `/cv/` are excluded from crawling. The CV contains personal and professional data that cannot be processed automatically without my explicit permission, as explained in the [repository README](https://github.com/darccio/darccio#%EF%B8%8F-license). I have authorised its update and the generation of its HTML version during the site build; that specific authorisation does not extend to other uses or to agent text collections.

## Copies outside this site

If an article is published to Standard.site or other AT Protocol services, copies distributed through those services fall outside the scope of dario.cat's `robots.txt`. Publishing through those services requires a review of the content being shared.

For questions about specific uses or to report a problem, [contact me](/en/contact/).
