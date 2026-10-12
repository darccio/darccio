// Use the pinned tokenizer entry directly; parse5's public index imports the
// full tree parser and retains unrelated initialization in a standalone bundle.
import { Tokenizer, TokenizerMode } from './node_modules/parse5/dist/tokenizer/index.js';

// Match Go strings.Fields/TrimSpace, rather than JavaScript's different \s set.
const space = /[\t\n\v\f\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+/;
const metadataName = /^go-(?:import|source)$/iu;
const maxBody = 1 << 20;

function requireCondition(condition, message) {
  if (!condition) throw new Error(message);
}

function fields(value) {
  return value.split(space).filter(part => part !== '');
}

// The assertion sandbox has no documented TextEncoder or Buffer. Count UTF-8
// bytes directly, rejecting invalid UTF-16 input rather than coercing it.
function checkBody(body) {
  requireCondition(typeof body === 'string', 'Vanity response body must be text');
  requireCondition(body.length <= maxBody, 'Vanity response exceeds 1 MiB');
  let bytes = 0;
  for (let i = 0; i < body.length; i++) {
    const unit = body.charCodeAt(i);
    if (unit <= 0x7f) bytes++;
    else if (unit <= 0x7ff) bytes += 2;
    else if (unit >= 0xd800 && unit <= 0xdbff) {
      const next = body.charCodeAt(++i);
      requireCondition(next >= 0xdc00 && next <= 0xdfff, 'Vanity response contains invalid Unicode');
      bytes += 4;
    } else {
      requireCondition(unit < 0xdc00 || unit > 0xdfff, 'Vanity response contains invalid Unicode');
      bytes += 3;
    }
    requireCondition(bytes <= maxBody, 'Vanity response exceeds 1 MiB');
  }
}

function parseMetadata(body) {
  const result = { go_import: [], go_source: [] };
  let inHead = true;
  let rawText = '';
  let tokenizer;

  const text = token => {
    if (!rawText && fields(token.chars).length !== 0) inHead = false;
  };
  const handler = {
    onComment() {},
    onDoctype() {},
    onEof() {},
    onCharacter: text,
    onNullCharacter: text,
    onWhitespaceCharacter: text,
    onEndTag(token) {
      if (rawText) {
        if (token.tagName === rawText) rawText = '';
        return;
      }
      if (['head', 'html', 'body', 'frameset'].includes(token.tagName)) inHead = false;
    },
    onStartTag(token) {
      if (rawText) return;
      const name = token.tagName;
      switch (name) {
        case 'html': case 'head': case 'base': case 'basefont':
        case 'bgsound': case 'link': case 'meta':
          break;
        case 'title': case 'style': case 'script': case 'noscript':
          rawText = name;
          break;
        default:
          inHead = false;
      }
      // A tokenizer alone needs the element-specific modes supplied by a tree
      // parser. This matches x/net/html's raw-text/RCDATA/script tokenization.
      if (name === 'script') tokenizer.state = TokenizerMode.SCRIPT_DATA;
      else if (['title', 'textarea'].includes(name)) tokenizer.state = TokenizerMode.RCDATA;
      else if (['style', 'noscript', 'iframe', 'noembed', 'noframes', 'xmp'].includes(name)) tokenizer.state = TokenizerMode.RAWTEXT;
      else if (name === 'plaintext') tokenizer.state = TokenizerMode.PLAINTEXT;

      if (name !== 'meta') return;
      // parse5, like x/net/html and Go's discovery parser, retains the first
      // duplicate attribute. Keep metadata-tag multiplicity for comparison.
      const attrs = Object.create(null);
      for (const attr of token.attrs) attrs[attr.name] = attr.value;
      const metadata = attrs.name || '';
      if (!metadataName.test(metadata)) return;
      requireCondition(metadata === 'go-import' || metadata === 'go-source', 'Vanity metadata name has invalid case');
      requireCondition(inHead, 'Vanity metadata appears outside the head');
      const value = fields(attrs.content || '');
      requireCondition(value.length === (metadata === 'go-import' ? 3 : 4), 'Vanity metadata has an invalid field count');
      result[metadata === 'go-import' ? 'go_import' : 'go_source'].push(value.join(' '));
    },
  };
  tokenizer = new Tokenizer({}, handler);
  tokenizer.write(body, true);
  return result;
}

function checkMetadata(expected, actual) {
  requireCondition(expected !== null && typeof expected === 'object' && !Array.isArray(expected), 'Vanity expected metadata must be an object');
  for (const key of ['go_import', 'go_source']) {
    requireCondition(Array.isArray(expected[key]) && expected[key].every(value => typeof value === 'string'), 'Vanity expected metadata must contain string arrays');
    const want = expected[key].slice().sort();
    const got = actual[key].slice().sort();
    requireCondition(got.length === want.length && got.every((value, i) => value === want[i]), 'Vanity ' + key + ' differs from the reviewed baseline');
  }
  requireCondition(expected.go_import.length > 0, 'Vanity expected import metadata is empty');
}

// Terraform replaces this one identifier with the reviewed baseline route JSON.
// No request, DOM, module loader or Node global is used in the deployed bundle.
const expected = __EXPECTED_METADATA__;
checkBody(dd.response.body);
checkMetadata(expected, parseMetadata(dd.response.body));
dd.assert(true, 'Reviewed Go vanity metadata is present exactly once per expected tag');
