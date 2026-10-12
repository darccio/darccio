# Cross-format build checks

The independent `consistency` command compares the final assembled deployment
with literal source metadata and the offline Standard.site plan. Run from the
site root:

```sh
make -f .zas/Makefile check-consistency
```

The production build runs the same check against its staged output before
replacing `.zas/deploy`. No PDS, credentials or private recovery state is needed.
The explicit empty approval list and public URI map remain required artifacts;
an empty list never grants publication approval.

Checks cover source publication/update instants, canonical URLs and RSS GUIDs,
titles, summaries, tags, language and full content across RSS, h-entry, Open
Graph, JSON-LD and planned Standard.site records. The branded Open Graph title
retains the configured site-name suffix. RSS timestamps are compared at the
format's second precision. Missing publication instants stay absent everywhere;
draft and explicitly excluded articles cannot enter a feed or document plan.

Each RSS feed's ordered subset and item cap are derived independently from site
configuration. Publication and document verification files must agree with
committed state, reviewed paths and built head links. Planned identities alone
cannot become verified backlinks.

Checked-in offline source, HTML, XML and JSON fixtures exercise valid multilingual
output and purposeful corruptions, including missing artifacts, wrong dates,
URLs, summaries, tags, content, feed order/caps, exclusions and approval drift.
Run `go -C .tools/microformats test -race -shuffle=on ./...` for the parser and
consistency suites. A failed gate preserves the previous deployment rather than
publishing partially checked output.
