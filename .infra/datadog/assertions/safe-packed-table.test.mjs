import assert from 'node:assert/strict';
import { runInNewContext } from 'node:vm';
import { test } from 'node:test';
import { escapePackedTable } from './safe-packed-table.mjs';

const packed = '!{{U#:5Abb}}\"\\\n{still-data} {{{triple}}}';
const literal = JSON.stringify(packed);
const upstream = 'export const htmlDecodeTree = /* #__PURE__ */ decodeTrieDict(' + literal + ', 10, 2, 0, 1, 2);';

test('escape only the complete data literal and preserve its decoded value', () => {
  const code = '(()=>{function result(){return {data:' + literal + '}}return result().data})()';
  const escaped = escapePackedTable(code, upstream);
  assert.equal(escaped.includes('{{'), false);
  assert.equal(escaped.includes('}}'), true, 'Structural closing braces must retain their JavaScript meaning');
  assert.equal(escaped.includes('{still-data}'), true, 'Unpaired data braces need no size-expanding escape');
  assert.equal(runInNewContext(escaped), runInNewContext(code));
  assert.equal(runInNewContext(escaped), packed);
  assert.equal(escapePackedTable(code, upstream), escaped, 'Generation is deterministic');
});

test('unknown interpolation sites or changed dependency serialization fail closed', () => {
  for (const [code, source] of [
    ['var x="different";', upstream],
    ['var x=' + literal + ';var y=' + literal + ';', upstream],
    ['var x=' + literal + ';function other(){{return 1}}', upstream],
    ['var x=' + literal + ';var y="{{OTHER}}";', upstream],
    ['var x=' + literal + ';/* {{OTHER}} */', upstream],
    ['var x=' + literal + ';', upstream.replace('htmlDecodeTree', 'changedTable')],
  ]) assert.throws(() => escapePackedTable(code, source), /Expected exactly one|prefix remains|format changed/);
});
