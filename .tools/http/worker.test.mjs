import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import test from "node:test";
import { acceptQualities, createWorker } from "./worker.mjs";

const html = '<!doctype html><html lang="ca"><h1>Sobre mi</h1></html>\n';
const markdown = "# Sobre mi\n\nEnginyer de programari.\n";
const sha256 = (body) => createHash("sha256").update(body).digest("hex");
const htmlTag = `W/"html-${sha256(html)}"`;
const markdownTag = `W/"md-${sha256(markdown)}"`;
const signal = "search=yes, ai-input=yes, ai-train=no";
const routes = {
  "/": { path: null, sha256: null, htmlSha256: sha256(html) },
  "/about/": { path: "/about/index.md", sha256: sha256(markdown), htmlSha256: sha256(html) },
};

function assets(responseFor) {
  const requests = [];
  return {
    requests,
    env: {
      ASSETS: {
        async fetch(request) {
          requests.push(request);
          return responseFor(request, requests.length);
        },
      },
    },
  };
}

function staticResponse(request, body, type = "text/html; charset=utf-8", extra = {}) {
  return new Response(request.method === "HEAD" ? null : body, {
    headers: {
      "Content-Type": type,
      "Content-Length": String(Buffer.byteLength(body)),
      "Cache-Control": "public, max-age=3600",
      ETag: '"shared-asset-tag"',
      "Last-Modified": "Sun, 11 Oct 2026 00:00:00 GMT",
      Vary: "Accept-Encoding",
      ...extra,
    },
  });
}

function ordinaryAssets() {
  return assets((request) => {
    const url = new URL(request.url);
    return url.pathname === "/about/index.md"
      ? staticResponse(request, markdown, "text/markdown; charset=utf-8")
      : staticResponse(request, html);
  });
}

function request(path = "/about/", headers = {}, method = "GET") {
  return new Request(`https://preview.dario.pages.dev${path}`, { method, headers });
}

function assertNegotiatedHeaders(response, expectedType) {
  assert.equal(response.headers.get("Content-Type"), expectedType);
  assert.equal(response.headers.get("Content-Signal"), signal);
  assert.equal(response.headers.get("Cache-Control"), "private, no-store");
  assert.equal(response.headers.get("CDN-Cache-Control"), "no-store");
  assert.equal(response.headers.get("Cloudflare-CDN-Cache-Control"), "no-store");
  assert.equal(response.headers.get("Last-Modified"), null);
  const vary = response.headers.get("Vary").toLowerCase().split(",").map((name) => name.trim());
  assert.equal(vary.filter((name) => name === "accept").length, 1);
}

test("Accept quality follows specificity, UTF-8 parameters, and conservative duplicate policy", () => {
  const cases = [
    [null, 1, 1],
    ["", 0, 0],
    ["*/*", 1, 1],
    ["text/markdown", 0, 1],
    ["text/html", 1, 0],
    ["text/html, text/markdown", 1, 1],
    ["text/markdown;q=0.9, text/html;q=0.2", 0.2, 0.9],
    ["text/markdown;q=0, */*;q=1", 1, 0],
    ["text/html;q=0, text/*;q=0.5", 0, 0.5],
    ["text/*;q=0.9, text/markdown;q=0.1", 0.9, 0.1],
    ["*/*;q=0.8, text/*;q=0.3, text/markdown;q=0.7", 0.3, 0.7],
    ["text/*;charset=utf-8;q=0.9, text/markdown;q=0.2", 0.9, 0.2],
    ["TEXT/MARKDOWN; CHARSET=\"UTF-8\"; Q=1", 0, 1],
    ["text/markdown; charset=iso-8859-1, text/html;q=0.2", 0.2, 0],
    ["text/markdown;variant=CommonMark, text/html;q=0.2", 0.2, 0],
    ["text/markdown;q=1;charset=utf-8", 0, 1],
    ["text/markdown;q=0.7, text/markdown;q=0", 0, 0],
    ["text/markdown;q=0, text/markdown;q=1", 0, 0],
    ["text/markdown;q=0.5;q=1", 0, 0],
    ["text/markdown;q=1.001, */*;q=0.5", 0.5, 0],
    ["text/markdown;q=garbage, */*;q=0.5", 0.5, 0],
    ["text/markdown;q=, */*;q=0.5", 0.5, 0],
    ["text/markdown;q=\"1\", */*;q=0.5", 0.5, 0],
    ["text/markdown;q=0.1234, */*;q=0.5", 0.5, 0],
    ["text/markdown;q=.9, */*;q=0.5", 0.5, 0],
    ["text/markdown;q=-1, */*;q=0.5", 0.5, 0],
    ["text/markdown;q=1e0, */*;q=0.5", 0.5, 0],
    ["text/markdown;note=\"comma,semicolon;quote\\\"\", text/html", 1, 0],
    ["text/markdown;note=\"unterminated, text/html", 0, 0],
    ["*/markdown", 0, 0],
    ["text/mark*", 0, 0],
    ["text/markdown; charset=utf-8; charset=utf-8", 0, 0],
    ["application/json", 0, 0],
  ];
  for (const [accept, html, markdown] of cases) {
    assert.deepEqual(acceptQualities(accept), { html, markdown }, String(accept));
  }
});

test("ordinary browser, absent, tied, and lower-quality Markdown preferences stay HTML", async () => {
  for (const accept of [
    null,
    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "text/markdown,text/html",
    "text/markdown;q=0.2,text/html;q=0.9",
    "text/markdown;q=0,*/*",
  ]) {
    const origin = ordinaryAssets();
    const response = await createWorker(routes).fetch(request("/about/", accept ? { Accept: accept } : {}), origin.env);
    assert.equal(response.status, 200);
    assert.equal(await response.text(), html);
    assertNegotiatedHeaders(response, "text/html; charset=utf-8");
    assert.equal(response.headers.get("ETag"), htmlTag);
    assert.equal(origin.requests.length, 1);
  }
});

test("Markdown preference returns verified twin bytes and preserves origin and query parameters", async () => {
  const origin = ordinaryAssets();
  const response = await createWorker(routes).fetch(request("/about/?from=rss&from=link", {
    Accept: "text/markdown;q=1,text/html;q=0.1",
    "Accept-Encoding": "gzip,br",
    "X-Test-Header": "retained",
  }), origin.env);
  assert.equal(response.status, 200);
  assert.equal(await response.text(), markdown);
  assertNegotiatedHeaders(response, "text/markdown; charset=utf-8");
  assert.equal(response.headers.get("ETag"), markdownTag);
  assert.equal(origin.requests.length, 2);
  assert.equal(origin.requests[1].url, "https://preview.dario.pages.dev/about/index.md?from=rss&from=link");
  assert.equal(origin.requests[1].headers.get("Accept-Encoding"), "identity");
  assert.equal(origin.requests[1].headers.get("X-Test-Header"), "retained");
  assert.equal(origin.requests[1].redirect, "manual");
});

test("GET and HEAD select identical verified variants and HEAD returns no body", async () => {
  for (const accept of ["text/html", "text/markdown"]) {
    const getOrigin = ordinaryAssets();
    const headOrigin = ordinaryAssets();
    const get = await createWorker(routes).fetch(request("/about/", { Accept: accept }), getOrigin.env);
    const head = await createWorker(routes).fetch(request("/about/", { Accept: accept }, "HEAD"), headOrigin.env);
    assert.equal(head.status, get.status);
    assert.deepEqual([...head.headers], [...get.headers]);
    assert.equal(await head.text(), "");
    assert.equal(headOrigin.requests[0].method, "HEAD");
    if (accept === "text/markdown") assert.equal(headOrigin.requests[1].method, "GET");
  }
});

test("go-get, raw vanity, CV, static and unknown routes never request Markdown", async () => {
  for (const path of [
    "/about/?go-get=1", "/about/?go-get=", "/about/?go-get", "/about/?go-get=0",
    "/mergo", "/mergo/?go-get=1", "/darna/", "/xdr/", "/lyd/", "/awaken/", "/cocoon/", "/zas/",
    "/CV/MAC.json", "/cv/", "/not-a-page/", "/about/index.md", "/feed.xml", "/en/feed.xml",
    "/llms.txt", "/en/llms-full.txt", "/.well-known/security.txt", "/static/avatar.webp",
    "/about%2f", "/about/index.html",
  ]) {
    const origin = ordinaryAssets();
    const response = await createWorker(routes).fetch(request(path, { Accept: "text/markdown" }), origin.env);
    assert.equal(response.status, 200, path);
    assert.equal(await response.text(), path === "/about/index.md" ? markdown : html, path);
    assert.equal(response.headers.get("Content-Signal"), signal, path);
    assert.equal(response.headers.get("Cache-Control"), "public, max-age=3600", path);
    assert.equal(response.headers.get("ETag"), '"shared-asset-tag"', path);
    assert.equal(origin.requests.length, 1, path);
  }
});

test("unsupported methods preserve the original request, body, status, and Allow header", async () => {
  for (const method of ["POST", "PUT", "DELETE", "OPTIONS"]) {
    const input = new Request("https://preview.dario.pages.dev/about/", {
      method,
      headers: { Accept: "text/markdown" },
      ...(method === "POST" || method === "PUT" ? { body: "original request body" } : {}),
    });
    const origin = assets(async (upstream) => {
      assert.equal(upstream, input);
      if (method === "POST" || method === "PUT") assert.equal(await upstream.text(), "original request body");
      return new Response("Use GET or HEAD.\n", { status: 405, headers: { Allow: "GET, HEAD" } });
    });
    const response = await createWorker(routes).fetch(input, origin.env);
    assert.equal(response.status, 405);
    assert.equal(response.headers.get("Allow"), "GET, HEAD");
    assert.equal(response.headers.get("Content-Signal"), signal);
    assert.equal(origin.requests.length, 1);
  }
});

test("native redirects and error statuses are preserved before negotiation", async () => {
  for (const status of [301, 302, 307, 308, 404, 410, 429, 500, 503]) {
    const origin = assets(() => new Response("Original response.\n", {
      status,
      headers: { "Content-Type": "text/html", Location: "/about/", "Retry-After": "60" },
    }));
    const response = await createWorker(routes).fetch(request("/about/", { Accept: "text/markdown" }), origin.env);
    assert.equal(response.status, status);
    assert.equal(await response.text(), "Original response.\n");
    assert.equal(response.headers.get("Location"), "/about/");
    assert.equal(response.headers.get("Retry-After"), "60");
    assert.equal(response.headers.get("Content-Signal"), signal);
    assert.equal(origin.requests.length, 1);
  }
});

test("a known page without a twin falls back only when HTML is acceptable", async () => {
  for (const [accept, status] of [
    ["text/markdown", 406],
    ["text/markdown,text/html;q=0.2", 200],
    ["*/*;q=0", 406],
    ["application/json", 406],
    ["", 406],
  ]) {
    const origin = ordinaryAssets();
    const response = await createWorker(routes).fetch(request("/", { Accept: accept }), origin.env);
    assert.equal(response.status, status);
    assert.equal(origin.requests.length, 1);
    assert.equal(response.headers.get("Content-Signal"), signal);
    assert.equal(response.headers.get("Cache-Control"), "private, no-store");
  }
});

test("HEAD has no body for an unavailable requested representation", async () => {
  const origin = ordinaryAssets();
  const response = await createWorker(routes).fetch(request("/", { Accept: "text/markdown" }, "HEAD"), origin.env);
  assert.equal(response.status, 406);
  assert.equal(await response.text(), "");
  assert.equal(response.headers.get("Content-Signal"), signal);
  assert.equal(response.headers.get("Cache-Control"), "private, no-store");
});

test("missing, redirected, failed, empty, and HTML-fall-through twins are never served as Markdown", async () => {
  const candidates = [
    () => new Response("Missing", { status: 404 }),
    () => new Response(null, { status: 308, headers: { Location: "/" } }),
    () => new Response("Unavailable", { status: 503 }),
    (input) => staticResponse(input, "", "text/markdown; charset=utf-8"),
    (input) => staticResponse(input, html, "text/markdown; charset=utf-8"),
    (input) => staticResponse(input, "# Unexpected replacement\n", "text/markdown; charset=utf-8"),
    () => { throw new Error("origin connection lost"); },
    () => new Response(new ReadableStream({ start(controller) { controller.error(new Error("body stream failed")); } }), {
      headers: { "Content-Type": "text/markdown; charset=utf-8" },
    }),
  ];
  for (const candidate of candidates) {
    for (const [accept, status] of [["text/markdown", 406], ["text/markdown,text/html;q=0.2", 200]]) {
      const origin = assets((input) => new URL(input.url).pathname.endsWith(".md")
        ? candidate(input) : staticResponse(input, html));
      const response = await createWorker(routes).fetch(request("/about/", { Accept: accept }), origin.env);
      assert.equal(response.status, status);
      assert.equal(response.headers.get("Content-Signal"), signal);
      assert.notEqual(response.headers.get("Content-Type"), "text/markdown; charset=utf-8");
      if (status === 200) assert.equal(await response.text(), html);
    }
  }
});

test("cross-variant conditional requests cannot reuse an HTML or Markdown validator", async () => {
  for (const [accept, incoming, outgoing, body] of [
    ["text/markdown", htmlTag, markdownTag, markdown],
    ["text/html", markdownTag, htmlTag, html],
  ]) {
    const origin = ordinaryAssets();
    const response = await createWorker(routes).fetch(request("/about/", {
      Accept: accept,
      "If-None-Match": incoming,
      "If-Modified-Since": "Sun, 11 Oct 2026 00:00:00 GMT",
    }), origin.env);
    assert.equal(response.status, 200);
    assert.equal(await response.text(), body);
    assert.equal(response.headers.get("ETag"), outgoing);
    for (const upstream of origin.requests) {
      assert.equal(upstream.headers.get("If-None-Match"), null);
      assert.equal(upstream.headers.get("If-Modified-Since"), null);
    }
  }
});

test("matching representation validators use weak comparison and return a body-free 304", async () => {
  for (const [accept, etag, type] of [
    ["text/html", htmlTag, "text/html; charset=utf-8"],
    ["text/markdown", markdownTag, "text/markdown; charset=utf-8"],
  ]) {
    for (const condition of [etag, etag.replace(/^W\//, ""), `"not-this-tag", ${etag}`, "*"]) {
      for (const method of ["GET", "HEAD"]) {
        const origin = ordinaryAssets();
        const response = await createWorker(routes).fetch(request("/about/", {
          Accept: accept, "If-None-Match": condition,
        }, method), origin.env);
        assert.equal(response.status, 304);
        assert.equal(await response.text(), "");
        assert.equal(response.headers.get("ETag"), etag);
        assert.equal(response.headers.get("Content-Length"), null);
        assertNegotiatedHeaders(response, type);
      }
    }
  }
});

test("asset validators on direct Markdown and static files retain native conditional behavior", async () => {
  for (const path of ["/about/index.md", "/feed.xml", "/CV/MAC.json"]) {
    const input = request(path, { "If-None-Match": '"shared-asset-tag"', "If-Modified-Since": "yesterday" });
    const origin = assets((upstream) => {
      assert.equal(upstream, input);
      return new Response(null, { status: 304, headers: { ETag: '"shared-asset-tag"', "Cache-Control": "public, max-age=3600" } });
    });
    const response = await createWorker(routes).fetch(input, origin.env);
    assert.equal(response.status, 304);
    assert.equal(response.headers.get("ETag"), '"shared-asset-tag"');
    assert.equal(response.headers.get("Cache-Control"), "public, max-age=3600");
    assert.equal(response.headers.get("Content-Signal"), signal);
  }
});

test("full page negotiation ignores byte ranges without returning HTML 206 for a Markdown preference", async () => {
  const origin = ordinaryAssets();
  const response = await createWorker(routes).fetch(request("/about/", {
    Accept: "text/markdown", Range: "bytes=0-5", "If-Range": htmlTag,
  }), origin.env);
  assert.equal(response.status, 200);
  assert.equal(await response.text(), markdown);
  for (const upstream of origin.requests) {
    assert.equal(upstream.headers.get("Range"), null);
    assert.equal(upstream.headers.get("If-Range"), null);
  }
});

test("direct assets preserve native byte-range responses", async () => {
  const input = request("/static/avatar.webp", { Range: "bytes=0-5", "If-Range": '"image-tag"' });
  const origin = assets((upstream) => {
    assert.equal(upstream, input);
    return new Response("sixbyt", { status: 206, headers: { "Content-Range": "bytes 0-5/100", "Cache-Control": "public, max-age=3600" } });
  });
  const response = await createWorker(routes).fetch(input, origin.env);
  assert.equal(response.status, 206);
  assert.equal(await response.text(), "sixbyt");
  assert.equal(response.headers.get("Content-Range"), "bytes 0-5/100");
  assert.equal(response.headers.get("Cache-Control"), "public, max-age=3600");
});

test("Vary is merged case-insensitively and wildcard variation remains unchanged", async () => {
  for (const vary of ["Accept-Encoding, accept", "*", "Accept-Encoding"]) {
    const origin = assets((input) => staticResponse(input, html, "text/html; charset=utf-8", { Vary: vary }));
    const response = await createWorker(routes).fetch(request("/about/"), origin.env);
    assert.equal(response.headers.get("Vary"), vary === "Accept-Encoding" ? "Accept-Encoding, Accept" : vary);
  }
});

test("asset transport failures return an uncached, policy-bearing response", async () => {
  for (const method of ["GET", "HEAD"]) {
    const origin = assets(() => { throw new Error("unreachable asset service"); });
    const response = await createWorker(routes).fetch(request("/about/", {}, method), origin.env);
    assert.equal(response.status, 502);
    assert.equal(response.headers.get("Content-Signal"), signal);
    assert.equal(response.headers.get("Cache-Control"), "private, no-store");
    if (method === "HEAD") assert.equal(await response.text(), "");
  }
});

test("documentation landings negotiate literal twins while go-get always retains HTML metadata", async () => {
  const docsHTML = '<!doctype html><html lang="en"><head><meta name="go-import" content="dario.cat/mergo git https://github.com/imdario/mergo"></head><body>Documentation</body></html>';
  const docsMarkdown = '# Mergo\n\n`{{.Title}}` is a literal Go template example.\n';
  const worker = createWorker({
    '/mergo/': {path: '/mergo/index.md', sha256: sha256(docsMarkdown), htmlSha256: sha256(docsHTML)},
  });
  const backend = assets((req) => new URL(req.url).pathname === '/mergo/index.md'
    ? staticResponse(req, docsMarkdown, 'text/markdown; charset=utf-8')
    : staticResponse(req, docsHTML));
  const md = await worker.fetch(request('/mergo/', {Accept: 'text/markdown'}), backend.env);
  assertNegotiatedHeaders(md, 'text/markdown; charset=utf-8');
  assert.equal(await md.text(), docsMarkdown);
  const before = backend.requests.length;
  const vanity = await worker.fetch(request('/mergo/?go-get=1', {Accept: 'text/markdown'}), backend.env);
  assert.equal(await vanity.text(), docsHTML);
  assert.equal(backend.requests.length, before + 1);
  assert.equal(new URL(backend.requests.at(-1).url).search, '?go-get=1');
});
