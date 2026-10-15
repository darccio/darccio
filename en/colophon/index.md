<!--
entry: true
language: en
description: "How dario.cat is built: Zas, Cloudflare Pages, code and licences."
updated: 2026-10-15T19:56:48+02:00
-->

<h1 class="p-name">About this site</h1>

dario.cat is a static site built with [Zas](https://github.com/darccio/zas), my generator written in Go. Pages start as Markdown or HTML and share a template. The [site's source code](https://github.com/darccio/darccio) is public.

## Publishing

Cloudflare Pages builds the site when changes are pushed to the `main` branch. The build runs `make -f .zas/Makefile publish` and publishes `.zas/deploy`. The Zas version is pinned in the Makefile so each build uses a specific revision; this site uses [Zas v0.0.10](https://github.com/darccio/zas/releases/tag/v0.0.10).

The [project guides](/en/projects/) live in their own repositories. This site includes only selected files from a pinned revision and renders them as data, without executing project templates, plugins or scripts. Each guide retains its original licence and links to its source revision. Updates arrive as a proposed change to this site; they are published when merged into `main`.

Before publishing, the build checks the Go import paths served by this domain, such as `dario.cat/mergo` and `dario.cat/darna`.

## Design and content

The site uses HTML and CSS, with no custom JavaScript in the browser. Navigation, the contact email and pages can be read without scripts. The photograph is hosted here.

Catalan is the default language for the personal side, with an English translation of these pages. Articles are available in Catalan, Spanish and English; the English section opens the English article archive by default. Go project documentation is mainly in English.

[Feeds](/en/feeds/) let you follow the content in a feed reader. The [AI policy](/en/ai/) explains preferences for crawlers and personal data.

Articles share metadata between RSS feeds, microformats and structured data. Markdown versions and agent context files are generated during the build, without browser scripts.

Article publication to Standard.site runs as a separate process after deployment and is limited to reviewed articles. A failure in the publishing service does not stop the site from being deployed. Copies distributed through the AT Protocol are beyond the reach of this domain's crawler preferences.

## Licences

Original content for which I hold the necessary rights is published under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/). The site's original code retains [the Unlicense](/LICENSE-CODE). The CV, Lýd and third-party material have their own exceptions and conditions; [LICENSE](/LICENSE) explains the scope.
