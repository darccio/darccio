import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import test from "node:test";
import { createWorker } from "./worker.mjs";

const digest = (text) => createHash("sha256").update(text).digest("hex");
const shell = (language = "ca") => `<!doctype html><html lang="${language}"><head><link rel="canonical" href="https://dario.cat/${language === "ca" ? "" : language + "/"}search/"></head><body><main><h1>Search</h1><form method="get"><input name="q" value="" data-search-query></form><!-- SITE_SEARCH_STATUS_START --><p>Static catalogue</p><!-- SITE_SEARCH_STATUS_END --><!-- SITE_SEARCH_RESULTS_START --><ul class="search-results"><li>Static result</li></ul><!-- SITE_SEARCH_RESULTS_END --></main></body></html>`;
const entries = [
  { url: "/posts/ciencia/", title: "Ciència i política", summary: "Programari lliure", language: "ca" },
  { url: "/en/posts/mergo/", title: "Mergo", summary: "Go merging library", language: "en" },
  { url: "/es/posts/ciencia/", title: "Ciencia y política", summary: "Una biblioteca", language: "es" },
  { url: "/posts/biblioteca/", title: "Una biblioteca", summary: "Mergo and Go", language: "ca" },
];

function fixture({ index = entries, htmlFor = (path) => shell(path.startsWith("/en/") ? "en" : path.startsWith("/es/") ? "es" : "ca"), respond } = {}) {
  const routes = Object.fromEntries(index.filter((entry) => entry && typeof entry.url === "string").map((entry) => [entry.url, { path: null, htmlSha256: digest("article") }]));
  for (const path of ["/search/", "/en/search/", "/es/search/"]) {
    routes[path] = { path: path + "index.md", sha256: digest("# Static catalogue\n"), htmlSha256: digest(htmlFor(path)) };
  }
  const calls = [];
  const env = { ASSETS: { async fetch(request) {
    calls.push(request);
    if (respond) return respond(request);
    const path = new URL(request.url).pathname;
    const body = path.endsWith(".md") ? "# Static catalogue\n" : htmlFor(path);
    return new Response(request.method === "HEAD" ? null : body, { headers: {
      "Content-Type": path.endsWith(".md") ? "text/markdown; charset=utf-8" : "text/html; charset=utf-8",
      "Content-Length": String(Buffer.byteLength(body)),
      ETag: '"asset-validator"', "Last-Modified": "Thu, 01 Oct 2026 00:00:00 GMT",
      Age: "30", Expires: "Fri, 01 Jan 2027 00:00:00 GMT", Vary: "Accept-Encoding",
      "Content-Security-Policy": "default-src 'self'; script-src 'none'",
      "X-Content-Type-Options": "nosniff", "Cache-Control": "public, max-age=3600",
      "Content-Digest": "stale digest", "Repr-Digest": "stale digest",
    } });
  } } };
  const worker = createWorker(routes, index);
  return { worker, routes, env, calls, async fetch(path = "/search/?q=Mergo", headers = {}, method = "GET") {
    return worker.fetch(new Request("https://preview.pages.dev" + path, { method, headers }), env);
  } };
}

function resultURLs(html) {
  return [...html.matchAll(/<a class="search-result__title" href="([^"]+)">/g)].map((match) => match[1]);
}

test("search matches accents and case across every language, retaining root-relative preview links", async () => {
  const response = await fixture().fetch("/search/?q=CI%C3%88NCIA%20POLITICA");
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.deepEqual(resultURLs(html), ["/es/posts/ciencia/", "/posts/ciencia/"]);
  assert.match(html, /2 resultats per «CIÈNCIA POLITICA»/);
  assert.match(html, /<ul class="search-results"><li class="search-result"/);
  assert.match(html, /<span class="visually-hidden" lang="ca"> \(en castellà\)<\/span>/);
  assert.doesNotMatch(html, /href="https:\/\/dario.cat\/(?:es\/)?posts/);
});

test("all query terms must match, title hits outrank summary hits, and URL resolves ties", async () => {
  const candidates = [
    { url: "/c/", title: "Mergo", summary: "Go merging", language: "en" },
    { url: "/b/", title: "Mergo", summary: "Go merging tools", language: "en" },
    { url: "/a/", title: "Library", summary: "Mergo Go merging", language: "en" },
    { url: "/excluded/", title: "Mergo", summary: "No second term", language: "en" },
  ];
  const first = await fixture({ index: candidates }).fetch("/search/?q=mergo+merging");
  const second = await fixture({ index: [...candidates].reverse() }).fetch("/search/?q=mergo+merging");
  assert.deepEqual(resultURLs(await first.text()), ["/b/", "/c/", "/a/"]);
  assert.deepEqual(resultURLs(await second.text()), ["/b/", "/c/", "/a/"]);
});

test("Unicode compatibility forms and decomposed accents match the same entries", async () => {
  const index = [{ url: "/unicode/", title: "Ciència Ｇｏ", summary: "", language: "ca" }];
  const response = await fixture({ index }).fetch("/search/?q=" + encodeURIComponent("cIE\u0300Ncia go"));
  assert.deepEqual(resultURLs(await response.text()), ["/unicode/"]);
});

test("empty and whitespace queries browse the full catalogue; an empty public catalogue is legitimate", async () => {
  for (const query of ["", "   "]) {
    const response = await fixture().fetch("/search/?q=" + encodeURIComponent(query));
    assert.equal(response.status, 200);
    const html = await response.text();
    assert.equal(resultURLs(html).length, entries.length);
    assert.match(html, /Totes les pàgines \(4\)/);
  }
  const response = await fixture({ index: [] }).fetch("/search/?q=");
  assert.equal(response.status, 200);
  assert.match(await response.text(), /Totes les pàgines \(0\)/);
});

test("locale affects results and no-result messages without filtering other languages", async () => {
  for (const [path, result, none, label] of [
    ["/search/", "2 resultats", "No s’han trobat resultats", "en anglès"],
    ["/en/search/", "2 results", "No results found", "in English"],
    ["/es/search/", "2 resultados", "No se han encontrado resultados", "en inglés"],
  ]) {
    const found = await fixture().fetch(path + "?q=mergo");
    const html = await found.text();
    assert.equal(resultURLs(html).length, 2);
    assert.ok(html.includes(result));
    assert.ok(html.includes(label));
    const empty = await fixture().fetch(path + "?q=does-not-exist");
    assert.equal(empty.status, 200);
    const noResults = await empty.text();
    assert.ok(noResults.includes(none));
    assert.deepEqual(resultURLs(noResults), []);
  }
});

test("query, title, and summary are inert HTML, including attribute injection attempts", async () => {
  const query = '"><img src=x onerror="alert(1)">';
  const index = [{ url: "/unsafe-title/", title: query, summary: "<script>alert(2)</script>&text", language: "en" }];
  const response = await fixture({ index }).fetch("/search/?q=" + encodeURIComponent(query));
  const html = await response.text();
  assert.match(html, /value="&quot;&gt;&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;" data-search-query/);
  assert.match(html, /&lt;script&gt;alert\(2\)&lt;\/script&gt;&amp;text/);
  assert.doesNotMatch(html, /<img|<script|value="">/);
  assert.equal(response.headers.get("Content-Security-Policy"), "default-src 'self'; script-src 'none'");
});

test("duplicate, oversized, control, bidi-formatting and broken-encoding queries receive localized 400", async () => {
  for (const query of ["q=one&q=two", "q=" + "x".repeat(161), "q=one%0Atwo", "q=a%00b", "q=a%E2%80%AEb", "q=%FF"]) {
    const origin = fixture();
    const response = await origin.fetch("/en/search/?" + query);
    assert.equal(response.status, 400, query);
    assert.match(await response.text(), /Enter one query of up to 160 characters/);
    assert.equal(origin.calls.length, 0);
    assert.equal(response.headers.get("Cache-Control"), "private, no-store");
    assert.equal(response.headers.get("X-Robots-Tag"), "noindex");
  }
  const response = await fixture().fetch("/search/?q=" + encodeURIComponent("😀".repeat(160)));
  assert.equal(response.status, 200, "the bound counts Unicode characters rather than UTF-16 code units");
});

test("dynamic requests strip query strings and validators before fetching a complete HTML shell", async () => {
  const origin = fixture();
  const response = await origin.fetch("/search/?q=mergo&tracking=discard", {
    "If-None-Match": '"old"', "If-Modified-Since": "Thu, 01 Oct 2026 00:00:00 GMT",
    Range: "bytes=0-10", "If-Range": '"old"', "Accept-Encoding": "gzip", Accept: "text/markdown",
  });
  assert.equal(response.status, 200);
  assert.equal(origin.calls.length, 1);
  const upstream = origin.calls[0];
  assert.equal(upstream.url, "https://preview.pages.dev/search/");
  assert.equal(upstream.method, "GET");
  assert.equal(upstream.headers.get("Accept-Encoding"), "identity");
  assert.equal(upstream.headers.get("Accept"), "text/html");
  for (const name of ["If-None-Match", "If-Modified-Since", "Range", "If-Range"]) assert.equal(upstream.headers.get(name), null);
  for (const name of ["ETag", "Last-Modified", "Content-Length", "Content-Encoding", "Age", "Expires", "Content-Digest", "Repr-Digest"]) assert.equal(response.headers.get(name), null, name);
  assert.equal(response.headers.get("Cache-Control"), "private, no-store");
  assert.equal(response.headers.get("CDN-Cache-Control"), "no-store");
  assert.equal(response.headers.get("Cloudflare-CDN-Cache-Control"), "no-store");
  assert.equal(response.headers.get("X-Robots-Tag"), "noindex");
  assert.equal(response.headers.get("Vary"), "Accept-Encoding, Accept");
  assert.equal(response.headers.get("X-Content-Type-Options"), "nosniff");
  const body = await response.text();
  assert.doesNotMatch(body, /tracking=discard/);
});

test("HTML results retain the clean canonical search URL", async () => {
  const response = await fixture().fetch("/en/search/?q=mergo&from=one");
  assert.match(await response.text(), /<link rel="canonical" href="https:\/\/dario.cat\/en\/search\/">/);
});

test("search negotiates escaped meaningful Markdown and rejects excluded representations", async () => {
  const index = [{ url: "/unusual)/", title: "Mergo [unsafe](javascript:alert(1))", summary: "<b>Go</b> *code*", language: "en" }];
  for (const accept of ["text/markdown", "text/html;q=0,text/*;q=0.2", "text/markdown;q=0.9,text/html;q=0.1"]) {
    const response = await fixture({ index }).fetch("/en/search/?q=mergo", { Accept: accept });
    assert.equal(response.status, 200);
    assert.equal(response.headers.get("Content-Type"), "text/markdown; charset=utf-8");
    const body = await response.text();
    assert.match(body, /^# Search\n/);
    assert.ok(body.includes("\\[unsafe\\]\\(javascript:alert\\(1\\)\\)"));
    assert.ok(body.includes("(</unusual)/>)"));
    assert.ok(body.includes("\\<b\\>Go\\</b\\> \\*code\\*"));
    assert.doesNotMatch(body, /<!doctype|<form|<script/);
  }
  for (const accept of ["application/json", "text/html;q=0,text/markdown;q=0", ""]) {
    const origin = fixture();
    const response = await origin.fetch("/search/?q=mergo", { Accept: accept });
    assert.equal(response.status, 406);
    assert.equal(response.headers.get("X-Robots-Tag"), "noindex");
    assert.equal(origin.calls.length, 0);
  }
  const tie = await fixture().fetch("/search/?q=mergo", { Accept: "text/markdown,text/html" });
  assert.equal(tie.headers.get("Content-Type"), "text/html; charset=utf-8");
});

test("HEAD uses GET for shell validation and has the same headers and status without a body", async () => {
  for (const accept of ["text/html", "text/markdown"]) {
    const get = await fixture().fetch("/search/?q=mergo", { Accept: accept });
    const origin = fixture();
    const head = await origin.fetch("/search/?q=mergo", { Accept: accept }, "HEAD");
    assert.equal(head.status, get.status);
    assert.deepEqual([...head.headers], [...get.headers]);
    assert.equal(await head.text(), "");
    assert.equal(origin.calls[0].method, "GET");
  }
});

test("unknown, foreign, duplicate and malformed index entries fail closed without fetching assets", async () => {
  for (const index of [
    [{ ...entries[0], url: "https://elsewhere.test/" }],
    [{ ...entries[0], url: "//elsewhere.test/" }],
    [{ ...entries[0], url: "/a/../private/" }],
    [{ ...entries[0], url: "/posts/%2fprivate/" }],
    [{ ...entries[0], url: "/posts/\\private/" }],
    [{ ...entries[0], url: "/posts/article/?secret=true" }],
    [{ ...entries[0], url: "/posts/article/#part" }],
    [{ ...entries[0], language: "xx" }],
    [{ ...entries[0], summary: null }],
    [entries[0], entries[0]],
  ]) {
    const origin = fixture({ index });
    const response = await origin.fetch();
    assert.equal(response.status, 502, JSON.stringify(index));
    assert.equal(origin.calls.length, 0);
    assert.doesNotMatch(await response.text(), /Ciència|secret/);
  }
  const origin = fixture();
  delete origin.routes[entries[0].url];
  const response = await origin.fetch();
  assert.equal(response.status, 502, "an entry outside the built public page inventory is unavailable");
});

test("missing, duplicated, and reordered shell markers fail safely, even with a matching source digest", async () => {
  const badShells = [
    shell().replace("<!-- SITE_SEARCH_RESULTS_START -->", ""),
    shell().replace("<!-- SITE_SEARCH_STATUS_END -->", ""),
    shell().replace("<!-- SITE_SEARCH_STATUS_START -->", "<!-- SITE_SEARCH_STATUS_START --><!-- SITE_SEARCH_STATUS_START -->"),
    shell().replace('name="q" value="" data-search-query', 'name="q"'),
    shell().replace('name="q" value="" data-search-query', 'name="q" value="" data-search-query name="q" value="" data-search-query'),
    shell().replace("<!-- SITE_SEARCH_RESULTS_START -->", "<!-- SITE_SEARCH_RESULTS_END -->").replace("<!-- SITE_SEARCH_RESULTS_END --></main>", "<!-- SITE_SEARCH_RESULTS_START --></main>"),
  ];
  for (const html of badShells) {
    const response = await fixture({ htmlFor: () => html }).fetch();
    assert.equal(response.status, 502);
    assert.equal(await response.text(), "Search is unavailable.\n");
  }
});

test("an SPA fallback or changed shell cannot pass the inventory digest", async () => {
  const origin = fixture({ respond: () => new Response(shell() + "unexpected", { headers: { "Content-Type": "text/html" } }) });
  const response = await origin.fetch();
  assert.equal(response.status, 502);
  assert.equal(await response.text(), "Search is unavailable.\n");
  const wrongType = await fixture({ respond: () => new Response(shell(), { headers: { "Content-Type": "text/plain" } }) }).fetch();
  assert.equal(wrongType.status, 502, "search requires an HTML shell, even if bytes match its digest");
});

test("static-asset redirects and errors remain authoritative, while connection and body failures are bounded", async () => {
  for (const status of [301, 308, 404, 429, 503]) {
    const response = await fixture({ respond: () => new Response("Origin response", { status, headers: { Location: "/other/", "Retry-After": "60" } }) }).fetch();
    assert.equal(response.status, status);
    assert.equal(await response.text(), "Origin response");
    assert.equal(response.headers.get("Location"), "/other/");
    assert.equal(response.headers.get("Retry-After"), "60");
  }
  const disconnected = await fixture({ respond: () => { throw new Error("network failed"); } }).fetch();
  assert.equal(disconnected.status, 502);
  const brokenBody = await fixture({ respond: () => new Response(new ReadableStream({ start(controller) { controller.error(new Error("body failed")); } }), { headers: { "Content-Type": "text/html" } }) }).fetch();
  assert.equal(brokenBody.status, 502);
});

test("unsupported search methods return 405, but go-get and unregistered search paths bypass search", async () => {
  for (const method of ["POST", "PUT", "DELETE", "OPTIONS"]) {
    const origin = fixture();
    const response = await origin.fetch("/search/?q=mergo", {}, method);
    assert.equal(response.status, 405);
    assert.equal(response.headers.get("Allow"), "GET, HEAD");
    assert.equal(origin.calls.length, 0);
  }
  for (const path of ["/search/?q=mergo&go-get=1", "/mergo/?q=mergo&go-get=1", "/unknown/search/?q=mergo", "/search/index.md?q=mergo"]) {
    const origin = fixture();
    const response = await origin.fetch(path, { Accept: "text/markdown" });
    assert.equal(response.status, 200);
    assert.equal(origin.calls.length, 1);
    assert.equal(new URL(origin.calls[0].url).search, new URL("https://preview.pages.dev" + path).search);
    assert.equal(response.headers.get("ETag"), '"asset-validator"');
  }
  const origin = fixture();
  delete origin.routes["/search/"];
  const response = await origin.fetch("/search/?q=mergo", { Accept: "text/markdown" });
  assert.equal(await response.text(), shell());
  assert.equal(origin.calls.length, 1);
});

test("the no-query static catalogue keeps its bytes, normal validators and verified Markdown twin", async () => {
  const response = await fixture().fetch("/search/");
  assert.equal(response.status, 200);
  assert.equal(await response.text(), shell());
  assert.match(response.headers.get("ETag"), /^W\/"html-/);
  const markdown = await fixture().fetch("/search/", { Accept: "text/markdown" });
  assert.equal(markdown.headers.get("Content-Type"), "text/markdown; charset=utf-8");
  assert.equal(await markdown.text(), "# Static catalogue\n");
});
