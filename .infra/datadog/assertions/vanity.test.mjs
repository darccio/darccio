import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { createContext, Script } from 'node:vm';
import { test } from 'node:test';

const directory = path.dirname(fileURLToPath(import.meta.url));
const bundle = await readFile(path.join(directory, 'vanity.js'), 'utf8');
const baseline = JSON.parse(await readFile(path.resolve(directory, '../../..', '.tools/vanity/internal/vanity/baseline.json'), 'utf8'));
const route = baseline.find(row => row.path === '/mergo/');
const correct = '<meta name="go-import" content="' + route.go_import[0] + '">';
const fake = correct.replace('/imdario/', '/evil/');
const source = '<meta name="go-source" content="dario.cat/mergo https://example.org https://example.org/tree{/dir} https://example.org/blob{/dir}/{file}#L{line}">';

function execute(body, expected = route) {
  assert.equal(bundle.split('__EXPECTED_METADATA__').length, 2);
  const calls = [];
  const sandbox = {
    dd: {
      response: { body },
      assert(value, message) {
        assert.equal(value, true, message);
        calls.push(message);
      },
    },
  };
  // Only language built-ins and documented dd APIs are available. These names
  // cannot silently inherit Node or browser facilities from the test runner.
  for (const name of ['Buffer', 'process', 'require', 'module', 'exports', 'importScripts', 'DOMParser', 'document', 'window', 'fetch', 'TextEncoder', 'TextDecoder', 'setTimeout', 'console']) sandbox[name] = undefined;
  const context = createContext(sandbox, { codeGeneration: { strings: false, wasm: false } });
  try {
    new Script(bundle.replace('__EXPECTED_METADATA__', () => JSON.stringify(expected))).runInContext(context, { timeout: 2000 });
  } catch (error) {
    // Keep diagnostics readable: vm's default stack prints the whole minified
    // source line, including the vendored tokenizer and entity tables.
    throw new Error(error.message);
  }
  assert.equal(calls.length, 1, 'The standalone assertion must explicitly signal success');
}

test('the real seven tracked vanity fragments match the shared reviewed baseline', async () => {
  assert.equal(baseline.length, 7);
  for (const expected of baseline) {
    const filename = expected.path.slice(1, -1) + '.html';
    const captured = await readFile(path.join(directory, 'testdata', filename), 'utf8');
    const tracked = await readFile(path.resolve(directory, '../../..', expected.path.slice(1), 'index.html'), 'utf8');
    assert.equal(captured, tracked, 'Real response fixture changed without review: ' + expected.path);
    execute(captured, expected);
    assert.throws(() => execute(captured, expected.path === '/mergo/' ? baseline[1] : route), /reviewed baseline/);
  }
});

const accepted = {
  'fragment': correct,
  'full document': '<!doctype html><html lang="en"><head><title>Docs</title>' + correct + '</head><body><h1>Documentation</h1></body></html>',
  'tag and attribute case, quote style, order and spacing': '<META CONTENT=\' dario.cat/mergo\n\tgit   https://github.com/imdario/mergo \' NAME=\'go-import\' />',
  'unquoted name': '<MeTa name=go-import content="' + route.go_import[0] + '">',
  'numeric entities in names and fields': '<meta NAME="go&#45;import" CONTENT="dario.cat/mergo&#x9;git&#32;https://github.com/imdario/mergo" disabled>',
  'named entity in content': correct.replace('https:', 'https&colon;'),
  'ordinary metadata with raw characters': '<meta name="description" content="valid HTML with raw < and &" custom@key="x">' + correct,
  'comments do not contribute fake tags': '<!-- ' + fake + ' -->' + correct,
  'raw text does not contribute fake tags': '<title>' + fake + '</title><style>' + fake + '</style><script>const example = \'' + fake + '\';</script><noscript>' + fake + '</noscript>' + correct,
  'first duplicate name controls': correct.replace('name="go-import"', 'name="go-import" name="description"'),
  'first duplicate content controls': correct.slice(0, -1) + ' content="fake git fake">',
  'mixed duplicate attribute case': correct.replace('name="go-import"', 'NaMe="go-import" NAME="description"'),
  'Go Unicode whitespace': '\u0085\u00a0\u1680\u2000\u202f\u205f\u3000' + correct.replace(' git ', '\u0085git\u3000'),
  'body raw-text examples do not become metadata': correct + '<body><iframe>' + fake + '</iframe><textarea>' + fake + '</textarea><xmp>' + fake + '</xmp><noembed>' + fake + '</noembed><noframes>' + fake + '</noframes><plaintext>' + fake,
  'body comments and scripts after real metadata': correct + '<body><!--' + fake + '--><script>' + fake + '</script></body>',
};
for (const [name, body] of Object.entries(accepted)) test('accept: ' + name, () => execute(body));

const rejected = {
  'empty': '',
  'homepage fallback': '<html><head></head><body>Homepage</body></html>',
  'wrong prefix': correct.replace('dario.cat/mergo', 'dario.cat/other'),
  'wrong VCS': correct.replace(' git ', ' hg '),
  'wrong repository': fake,
  'duplicate import tags': correct + correct,
  'conflicting import tag': correct + fake,
  'comment only': '<!--' + correct + '-->',
  'script only': '<script>let example = \'' + correct + '\';</script>',
  'style only': '<style>' + correct + '</style>',
  'title only': '<title>' + correct + '</title>',
  'noscript only': '<noscript>' + correct + '</noscript>',
  'escaped tag only': '&lt;meta name="go-import" content="' + route.go_import[0] + '"&gt;',
  'attribute example only': '<div data-example=\'' + correct + '\'></div>',
  'metadata in explicit body': '<html><head></head><body>' + correct + '</body></html>',
  'metadata after head': '<head></head>' + correct,
  'implicit body': '<p>body</p>' + correct,
  'visible text starts implicit body': 'visible text' + correct,
  'head cannot reopen': '<body></body><head>' + correct + '</head>',
  'metadata after document': '<head>' + correct + '</head></html>' + correct,
  'extra body metadata after correct head': correct + '<body>' + correct,
  'extra go-source': correct + source,
  'go-source outside head': correct + '<body>' + source,
  'uppercase metadata name': correct.replace('go-import', 'GO-IMPORT'),
  'Unicode case-folded metadata name': correct + source.replace('go-source', 'go-ſource'),
  'first name is unrelated': correct.replace('name="go-import"', 'name="viewport" name="go-import"'),
  'first content conflicts': correct.replace('content=', 'content="fake git fake" content='),
  'hidden extra conflict via duplicate name': correct + fake.replace('name="go-import"', 'name="go-import" name="description"'),
  'missing tuple field': '<meta name="go-import" content="dario.cat/mergo git">',
  'extra tuple field': correct.replace(' git ', ' git extra '),
  'UTF-8 BOM is not Go whitespace': '\ufeff' + correct,
  'BOM is not a field separator': correct.replace(' git ', '\ufeffgit '),
  'unclosed comment hides metadata': '<!-- ' + correct,
  'unclosed script hides metadata': '<script>\'' + correct,
  'self-closing non-void script still hides metadata': '<script/>' + correct,
  'inert template starts non-head content': '<template>' + correct + '</template>',
};
for (const [name, body] of Object.entries(rejected)) test('reject: ' + name, () => assert.throws(() => execute(body), /Vanity/));

test('expected go-source tags retain multiplicity and exact field values', () => {
  const expected = { ...route, go_source: ['dario.cat/mergo https://example.org https://example.org/tree{/dir} https://example.org/blob{/dir}/{file}#L{line}'] };
  execute(correct + source, expected);
  assert.throws(() => execute(correct, expected), /reviewed baseline/);
  assert.throws(() => execute(correct + source + source, expected), /reviewed baseline/);
  assert.throws(() => execute(correct + source.replace('https://example.org/tree', 'https://evil.example/tree'), expected), /reviewed baseline/);
});

test('reject missing/non-string bodies and malformed expected metadata', () => {
  for (const body of [undefined, null, {}, [], 42]) assert.throws(() => execute(body), /body must be text/);
  for (const expected of [null, [], {}, { go_import: [], go_source: [] }, { go_import: [42], go_source: [] }]) assert.throws(() => execute(correct, expected), /Vanity expected|reviewed baseline/);
});

test('UTF-8 body cap includes multibyte characters and accepts exactly 1 MiB', () => {
  const prefix = correct + '<body>';
  execute(prefix + ' '.repeat((1 << 20) - Buffer.byteLength(prefix, 'utf8')));
  assert.throws(() => execute(prefix + ' '.repeat((1 << 20) - Buffer.byteLength(prefix, 'utf8') + 1)), /exceeds 1 MiB/);
  assert.throws(() => execute(prefix + 'é'.repeat(1 << 19)), /exceeds 1 MiB/);
  assert.throws(() => execute(prefix + '🦫'.repeat(1 << 18)), /exceeds 1 MiB/);
  for (const invalid of ['\ud800', '\udc00']) assert.throws(() => execute(prefix + invalid), /invalid Unicode/);
});

test('Terraform replacement produces a script without a runtime module loader', () => {
  assert.equal(bundle.split('__EXPECTED_METADATA__').length, 2);
  // Every test above runs the generated artifact, never the source module. A
  // import/require/DOM/global leak fails the restricted sandbox immediately.
  execute(correct);
});
