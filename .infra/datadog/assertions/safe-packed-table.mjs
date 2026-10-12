// Datadog scans assertion source for {{variable}} before JavaScript execution.
// The pinned entities package's compressed trie happens to contain that prefix.
// Reserialize only its complete quoted data literal, never JavaScript syntax.
export function escapePackedTable(bundle, upstream) {
  const matches = [...upstream.matchAll(/^export const htmlDecodeTree = \/\* #__PURE__ \*\/ decodeTrieDict\(("(?:[^"\\]|\\.)*"),/gm)];
  if (matches.length !== 1) throw new Error('Pinned entities packed-table format changed; review before rebuilding');
  const packed = JSON.parse(matches[0][1]);
  const literal = JSON.stringify(packed);
  if (bundle.split(literal).length !== 2) throw new Error('Expected exactly one complete entities data literal in bundle');
  // This replacement acts on a JSON string scalar, not code. Unicode escapes
  // retain every decoded table character and prevent a substitution prefix.
  const escaped = literal.replace(/\{/g, '\\u007b');
  if (JSON.parse(escaped) !== packed) throw new Error('Escaping changed entities table data');
  const safe = bundle.replace(literal, () => escaped);
  if (safe.includes('{{')) throw new Error('Datadog substitution prefix remains outside the reviewed table literal');
  return safe;
}
