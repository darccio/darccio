# Datadog Go vanity-import assertion

`vanity.js` is a generated, standalone HTTP response assertion for Datadog
Synthetics. Terraform replaces its one `__EXPECTED_METADATA__` identifier with
the route object from
[`baseline.json`](../../../.tools/vanity/internal/vanity/baseline.json):

```hcl
replace(file("${path.module}/assertions/vanity.js"), "__EXPECTED_METADATA__", jsonencode(each.value))
```

The route object contains `path`, `go_import` and `go_source`; only the two
metadata arrays participate in comparison. No baseline is learned from the HTTP
response. Missing, extra, duplicated or changed tags fail. HTTP status,
content-type, TLS, redirects and request timing belong to the Terraform test's
other assertions and settings.

The assertion follows the token-based policy in
[`vanity.Parse`](../../../.tools/vanity/internal/vanity/vanity.go): head fragments
and full documents are valid; visible text or a non-head element closes the
eligible head permanently. Comments and element raw text cannot impersonate
metadata. Actual vanity tags in the body fail, including extra tags after a
correct head. Tag/attribute spelling follows HTML normalization; metadata names
must be exactly `go-import` or `go-source`. Attribute entities are decoded, Go's
Unicode whitespace defines tuple fields, and the first duplicate attribute
wins. Duplicate tags retain their multiplicity. The decoded response string is
limited to 1 MiB measured as UTF-8 bytes, before tokenization; invalid UTF-16
strings also fail. No response content is copied into assertion error messages.

## Build and validate

Use Node 20.19 or newer. Installation scripts and package audit requests are
unnecessary:

```sh
cd .infra/datadog/assertions
npm ci --ignore-scripts --no-audit --no-fund
npm run build
npm test
node build.mjs --check
```

`package.json` pins parse5 8.0.1 and esbuild 0.28.2. The lock file also pins the
entities decoder and platform-specific build dependency. Only the HTML
tokenizer is bundled: the source imports its exact installed entry to avoid
including the full parse5 tree parser's initialization. This entry is an
internal API, so upgrades require rebuilding and rerunning the adversarial
tests. Tokenizer behavior is documented in the
[pinned upstream source](https://github.com/inikulin/parse5/blob/v8.0.1/packages/parse5/lib/tokenizer/index.ts).
The checked-in artifact is 94,042 UTF-8 bytes before route JSON substitution.
The `--check` command compares a fresh pinned build byte-for-byte without
overwriting the artifact.

The Node built-in tests execute the committed bundle in a restricted VM with
`dd.response.body` and callable `dd.assert`, without DOM, Node, module-loading,
network, encoder, timer or console globals. They cover all seven real source
fragments, full-document heads, entity/formatting variations, first duplicate
attributes, fake metadata in comments/scripts/text, wrong and duplicate tuples,
head reopening, body metadata, extra `go-source` and the UTF-8 body limit.
No test calls Datadog or the live website.

## Runtime boundary

Datadog documents the response-body string and Chai assertion APIs in its
[HTTP test guide](https://docs.datadoghq.com/synthetics/api_tests/http_tests/).
The deployed bundle needs standard ES2018 language facilities, including
`Map`, `Set`, typed arrays and Unicode regular expressions. It needs no
`DOMParser`, `require`, `import`, `Buffer`, `process` or `TextEncoder`.
The local VM tests establish those dependencies; they do not prove execution
inside Datadog's actual managed-location sandbox or establish a supported
script-size limit. Confirm the generated assertion with Datadog's test run
before treating the monitor as active. No JavaScript from this directory is
served to website visitors.

## Third-party notices

The runtime bundle includes the full license notices for
[parse5](licenses/parse5.txt) (MIT, Ivan Nikulin) and
[entities](licenses/entities.txt) (BSD-2-Clause, Felix Böhm).
[esbuild](licenses/esbuild.txt) (MIT, Evan Wallace) is a build-time tool;
its executable is not included in the assertion. Notices are copied unchanged
from the pinned packages. Public vanity fixture HTML is copied from the
tracked site source under the repository's license.
