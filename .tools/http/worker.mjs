// The guarded build replaces this marker with the verified public page inventory.
const PAGE_ROUTES = /* SITE_PAGE_ROUTES */ {};
const SEARCH_INDEX = /* SITE_SEARCH_INDEX */ [];

const CONTENT_SIGNAL = "search=yes, ai-input=yes, ai-train=no";
const TOKEN = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/;
const QVALUE = /^(?:0(?:\.[0-9]{0,3})?|1(?:\.0{0,3})?)$/;

// Split HTTP lists without treating separators inside quoted parameters as syntax.
function splitQuoted(value, separator) {
  const parts = [];
  let start = 0;
  let quoted = false;
  let escaped = false;
  for (let index = 0; index < value.length; index++) {
    const char = value[index];
    if (escaped) escaped = false;
    else if (quoted && char === "\\") escaped = true;
    else if (char === '"') quoted = !quoted;
    else if (!quoted && char === separator) {
      parts.push(value.slice(start, index));
      start = index + 1;
    }
  }
  if (quoted || escaped) return [];
  parts.push(value.slice(start));
  return parts;
}

function parseRange(value) {
  const parts = splitQuoted(value, ";");
  if (!parts.length) return null;
  const media = parts.shift().trim().toLowerCase().split("/");
  if (media.length !== 2 || media.some((part) => !TOKEN.test(part)) ||
      (media[0] === "*" && media[1] !== "*") ||
      media.some((part) => part.includes("*") && part !== "*")) return null;
  const parameters = new Map();
  let quality = 1;
  let weighted = false;
  for (const raw of parts) {
    const equals = raw.indexOf("=");
    if (equals === -1) return null;
    const name = raw.slice(0, equals).trim().toLowerCase();
    let value = raw.slice(equals + 1).trim();
    if (!TOKEN.test(name)) return null;
    const quoted = value.startsWith('"');
    if (name === "q") {
      // A malformed or repeated weight excludes this range; it never defaults to 1.
      quality = weighted || quoted || !QVALUE.test(value) ? 0 : Number(value);
      weighted = true;
      continue;
    }
    if (quoted) {
      if (!/^"(?:[^"\\\r\n]|\\[^\r\n])*"$/.test(value)) return null;
      value = value.slice(1, -1).replace(/\\(.)/g, "$1");
    } else if (!TOKEN.test(value)) return null;
    if (parameters.has(name)) return null;
    parameters.set(name, value);
  }
  return { media, parameters, quality };
}

function mediaQuality(ranges, subtype) {
  let best = -1;
  let quality = 0;
  for (const range of ranges) {
    if (range.media[0] !== "*" && range.media[0] !== "text") continue;
    if (range.media[1] !== "*" && range.media[1] !== subtype) continue;
    // Both served variants are UTF-8 and have no other media parameters.
    if ([...range.parameters].some(([name, value]) =>
      name !== "charset" || value.toLowerCase() !== "utf-8")) continue;
    const specificity = (range.media[0] === "*" ? 0 : range.media[1] === "*" ? 1 : 2) * 2
      + range.parameters.size;
    if (specificity > best) {
      best = specificity;
      quality = range.quality;
    } else if (specificity === best) {
      // Equally specific duplicate ranges use the lowest weight, including q=0.
      quality = Math.min(quality, range.quality);
    }
  }
  return quality;
}

export function acceptQualities(accept) {
  if (accept === null) return { html: 1, markdown: 1 };
  const ranges = splitQuoted(accept, ",").map(parseRange).filter(Boolean);
  return { html: mediaQuality(ranges, "html"), markdown: mediaQuality(ranges, "markdown") };
}

function responseHeaders(response, negotiated) {
  const headers = new Headers(response.headers);
  headers.set("Content-Signal", CONTENT_SIGNAL);
  if (negotiated) {
    const vary = headers.get("Vary");
    if (!vary || !vary.split(",").some((name) => name.trim() === "*" || name.trim().toLowerCase() === "accept")) {
      headers.set("Vary", vary ? `${vary}, Accept` : "Accept");
    }
    // Cloudflare's CDN does not generally key on Vary: Accept. Disable storage for
    // both variants until representation-aware caching is explicitly configured.
    headers.set("Cache-Control", "private, no-store");
    headers.set("CDN-Cache-Control", "no-store");
    headers.set("Cloudflare-CDN-Cache-Control", "no-store");
    headers.delete("ETag");
    headers.delete("Last-Modified");
    headers.delete("Age");
    headers.delete("Expires");
  }
  return headers;
}

function finish(response, request, negotiated = false) {
  return new Response(request.method === "HEAD" ? null : response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders(response, negotiated),
  });
}

function matchesIfNoneMatch(value, etag) {
  if (value === null) return false;
  if (value.trim() === "*") return true;
  const tags = splitQuoted(value, ",");
  const current = etag.replace(/^W\//, "");
  return tags.some((tag) => {
    tag = tag.trim();
    return /^(?:W\/)?"[!\x23-\x7e\x80-\xff]*"$/.test(tag) &&
      tag.replace(/^W\//, "") === current;
  });
}

function selected(response, request, etag) {
  const headers = responseHeaders(response, true);
  headers.set("ETag", etag);
  if (matchesIfNoneMatch(request.headers.get("If-None-Match"), etag)) {
    headers.delete("Content-Length");
    return new Response(null, { status: 304, headers });
  }
  return new Response(request.method === "HEAD" ? null : response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  });
}

function assetRequest(request, url = request.url, method = request.method) {
  const headers = new Headers(request.headers);
  // Revalidation must not let an HTML validator select a Markdown response, or
  // vice versa. The selected response exposes a separate build-digest validator.
  headers.delete("If-None-Match");
  headers.delete("If-Modified-Since");
  // Full page negotiation does not implement byte ranges. HTTP permits ignoring
  // Range; direct static asset requests retain their native range behavior.
  headers.delete("Range");
  headers.delete("If-Range");
  return new Request(url, { method, headers, redirect: "manual" });
}

function notAcceptable(request) {
  return finish(new Response("No acceptable representation is available.\n", {
    status: 406,
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  }), request, true);
}

async function fetchMarkdown(env, request, page) {
  const twinURL = new URL(request.url);
  twinURL.pathname = page.path;
  const twinRequest = assetRequest(request, twinURL, "GET");
  twinRequest.headers.set("Accept-Encoding", "identity");
  try {
    const markdown = await env.ASSETS.fetch(twinRequest);
    if (markdown.status !== 200) return null;
    const bytes = await markdown.arrayBuffer();
    const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", bytes));
    const hex = [...digest].map((byte) => byte.toString(16).padStart(2, "0")).join("");
    if (hex !== page.sha256) return null;
    const headers = new Headers(markdown.headers);
    headers.set("Content-Type", "text/markdown; charset=utf-8");
    headers.set("Content-Length", String(bytes.byteLength));
    headers.delete("Content-Encoding");
    return new Response(bytes, { status: 200, headers });
  } catch {
    // Missing twins, failed body streams and connection errors are unavailable
    // representations. The caller may fall back only to acceptable HTML.
    return null;
  }
}

const SEARCH_LANGUAGES = { "/search/": "ca", "/en/search/": "en", "/es/search/": "es" };
const SEARCH_COPY = {
  ca: {
    title: "Cerca", all: (count) => `Totes les pàgines (${count}).`,
    found: (count, query) => `${count} ${count === 1 ? "resultat" : "resultats"} per «${query}».`,
    none: (query) => `No s’han trobat resultats per «${query}».`,
    invalid: "Escriu una sola consulta de fins a 160 caràcters, sense caràcters de control.",
    languages: { ca: "en català", en: "en anglès", es: "en castellà" },
  },
  en: {
    title: "Search", all: (count) => `All pages (${count}).`,
    found: (count, query) => `${count} ${count === 1 ? "result" : "results"} for “${query}”.`,
    none: (query) => `No results found for “${query}”.`,
    invalid: "Enter one query of up to 160 characters, without control characters.",
    languages: { ca: "in Catalan", en: "in English", es: "in Spanish" },
  },
  es: {
    title: "Buscar", all: (count) => `Todas las páginas (${count}).`,
    found: (count, query) => `${count} ${count === 1 ? "resultado" : "resultados"} para «${query}».`,
    none: (query) => `No se han encontrado resultados para «${query}».`,
    invalid: "Introduce una sola consulta de hasta 160 caracteres, sin caracteres de control.",
    languages: { ca: "en catalán", en: "en inglés", es: "en castellano" },
  },
};

function escapeHTML(value) {
  return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]);
}

function escapeMarkdown(value) {
  return value.replace(/[\\`*_{}\[\]<>()#!|~]/g, "\\$&").replace(/[\r\n]+/g, " ");
}

function fold(value) {
  return value.normalize("NFKD").replace(/\p{M}/gu, "").toLowerCase();
}

function checkedIndex(routes, index) {
  const seen = new Set();
  for (const entry of index) {
    if (!entry || typeof entry.url !== "string" || typeof entry.title !== "string" ||
        typeof entry.summary !== "string" || typeof entry.language !== "string" || !Object.hasOwn(SEARCH_COPY.ca.languages, entry.language) ||
        !Object.hasOwn(routes, entry.url) || !routes[entry.url] || seen.has(entry.url) ||
        !entry.url.startsWith("/") || entry.url.startsWith("//") || /[\\\p{Cc}]|%(?:2f|5c)/iu.test(entry.url)) return false;
    const parsed = new URL(entry.url, "https://search.invalid");
    if (parsed.origin !== "https://search.invalid" || parsed.pathname !== entry.url || parsed.search || parsed.hash) return false;
    seen.add(entry.url);
  }
  return true;
}

function searchResults(index, query) {
  const terms = fold(query).split(/\s+/u).filter(Boolean);
  return index.map((entry) => {
    const title = fold(entry.title);
    const summary = fold(entry.summary);
    return { entry, matches: terms.every((term) => title.includes(term) || summary.includes(term)),
      score: terms.reduce((score, term) => score + (title.includes(term) ? 1 : 0), 0) };
  }).filter((result) => result.matches)
    .sort((a, b) => b.score - a.score || (a.entry.url < b.entry.url ? -1 : a.entry.url > b.entry.url ? 1 : 0))
    .map((result) => result.entry);
}

function searchRows(results, copy, language) {
  return results.map((entry) => `<li class="search-result" lang="${entry.language}" data-search-url="${escapeHTML(entry.url)}" data-search-language="${entry.language}"><h2><a class="search-result__title" href="${escapeHTML(entry.url)}">${escapeHTML(entry.title)}</a><span class="lang-tag" aria-hidden="true">${entry.language}</span><span class="visually-hidden" lang="${language}"> (${copy.languages[entry.language]})</span></h2>${entry.summary ? `<p class="search-result__summary">${escapeHTML(entry.summary)}</p>` : ""}</li>`).join("\n");
}

function replaceRegion(html, name, replacement) {
  const start = `<!-- SITE_SEARCH_${name}_START -->`;
  const end = `<!-- SITE_SEARCH_${name}_END -->`;
  const offset = html.indexOf(start);
  const limit = html.indexOf(end);
  if (offset < 0 || limit < offset + start.length || offset !== html.lastIndexOf(start) || limit !== html.lastIndexOf(end)) return null;
  return html.slice(0, offset + start.length) + replacement + html.slice(limit);
}

function searchResponse(body, request, status, type, original) {
  const headers = responseHeaders(original || new Response(), true);
  headers.set("Content-Type", `${type}; charset=utf-8`);
  headers.set("X-Robots-Tag", "noindex");
  for (const name of ["Content-Length", "Content-Encoding", "Content-Range", "Accept-Ranges", "Content-MD5", "Digest", "Content-Digest", "Repr-Digest"]) headers.delete(name);
  return new Response(request.method === "HEAD" ? null : body, { status, headers });
}

async function search(request, env, routes, index, language) {
  const url = new URL(request.url);
  const values = url.searchParams.getAll("q");
  const copy = SEARCH_COPY[language];
  if (values.length !== 1 || Array.from(values[0]).length > 160 || /[\p{Cc}\p{Cf}\uFFFD]/u.test(values[0])) {
    return searchResponse(copy.invalid + "\n", request, 400, "text/plain");
  }
  const qualities = acceptQualities(request.headers.get("Accept"));
  if (qualities.html === 0 && qualities.markdown === 0) {
    return searchResponse("No acceptable representation is available.\n", request, 406, "text/plain");
  }
  if (!checkedIndex(routes, index)) return searchResponse("Search is unavailable.\n", request, 502, "text/plain");
  // Query text belongs only in the response; the static-assets service needs the
  // clean page shell, without query strings, ranges, or conditional validators.
  url.search = "";
  const upstream = assetRequest(request, url, "GET");
  upstream.headers.set("Accept", "text/html");
  upstream.headers.set("Accept-Encoding", "identity");
  let original;
  try {
    original = await env.ASSETS.fetch(upstream);
  } catch {
    return searchResponse("Static assets are unavailable.\n", request, 502, "text/plain");
  }
  if (original.status !== 200) return finish(original, request, true);
  if (!/^text\/html(?:\s*;|$)/i.test(original.headers.get("Content-Type") || "")) {
    return searchResponse("Search is unavailable.\n", request, 502, "text/plain", original);
  }
  let html;
  try {
    const bytes = await original.arrayBuffer();
    const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", bytes));
    const hex = [...digest].map((byte) => byte.toString(16).padStart(2, "0")).join("");
    if (hex !== routes[url.pathname].htmlSha256) throw new Error("Search shell does not match the public inventory");
    html = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    return searchResponse("Search is unavailable.\n", request, 502, "text/plain", original);
  }
  const query = values[0].trim();
  const results = searchResults(index, query);
  const status = query ? (results.length ? copy.found(results.length, query) : copy.none(query)) : copy.all(results.length);
  html = replaceRegion(html, "RESULTS", `<ul class="search-results">${searchRows(results, copy, language)}</ul>`);
  if (html !== null) html = replaceRegion(html, "STATUS", `<p class="search-status" role="status">${escapeHTML(status)}</p>`);
  const input = 'name="q" value="" data-search-query';
  if (html === null || html.indexOf(input) < 0 || html.indexOf(input) !== html.lastIndexOf(input)) {
    return searchResponse("Search is unavailable.\n", request, 502, "text/plain", original);
  }
  html = html.replace(input, () => `name="q" value="${escapeHTML(query)}" data-search-query`);
  if (qualities.markdown > qualities.html) {
    const markdown = `# ${copy.title}\n\n${escapeMarkdown(status)}\n\n` + results.map((entry) =>
      `## [${escapeMarkdown(entry.title)}](<${entry.url}>) (${copy.languages[entry.language]})\n\n${escapeMarkdown(entry.summary)}\n`).join("\n");
    return searchResponse(markdown, request, 200, "text/markdown", original);
  }
  return searchResponse(html, request, 200, "text/html", original);
}

export function createWorker(routes, searchIndex = []) {
  return {
    async fetch(request, env) {
      const url = new URL(request.url);
      const searchPage = !url.searchParams.has("go-get") && Object.hasOwn(routes, url.pathname) &&
        Object.hasOwn(SEARCH_LANGUAGES, url.pathname) && Array.isArray(searchIndex);
      if (searchPage && request.method !== "GET" && request.method !== "HEAD") {
        const response = searchResponse("Use GET or HEAD.\n", request, 405, "text/plain");
        response.headers.set("Allow", "GET, HEAD");
        return response;
      }
      if (searchPage && url.searchParams.has("q")) return search(request, env, routes, searchIndex, SEARCH_LANGUAGES[url.pathname]);
      const eligible = (request.method === "GET" || request.method === "HEAD") &&
        !url.searchParams.has("go-get") && Object.hasOwn(routes, url.pathname);
      let original;
      try {
        original = await env.ASSETS.fetch(eligible ? assetRequest(request) : request);
      } catch {
        return finish(new Response("Static assets are unavailable.\n", {
          status: 502,
          headers: { "Content-Type": "text/plain; charset=utf-8", "Cache-Control": "no-store" },
        }), request, eligible);
      }
      // The assets service remains authoritative for redirects and errors. A
      // wildcard SPA fallback on an unknown route is never treated as a page twin.
      if (!eligible || original.status !== 200 ||
          !/^text\/html(?:\s*;|$)/i.test(original.headers.get("Content-Type") || "")) {
        return finish(original, request, eligible);
      }
      const qualities = acceptQualities(request.headers.get("Accept"));
      const page = routes[url.pathname];
      if (qualities.markdown > qualities.html && qualities.markdown > 0 && page.path) {
        const markdown = await fetchMarkdown(env, request, page);
        if (markdown) return selected(markdown, request, `W/"md-${page.sha256}"`);
      }
      return qualities.html > 0
        ? selected(original, request, `W/"html-${page.htmlSha256}"`)
        : notAcceptable(request);
    },
  };
}

export default createWorker(PAGE_ROUTES, SEARCH_INDEX);
