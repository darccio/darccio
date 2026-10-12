import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { cp, mkdtemp, readFile, rm, symlink, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';
import { build } from 'esbuild';
import { compactTokenizerPlugin } from './compact-tokenizer.mjs';

const directory = path.dirname(fileURLToPath(import.meta.url));

async function fixture(t) {
  const temporary = await mkdtemp(path.join(os.tmpdir(), 'dario-pinned-tokenizer-'));
  t.after(() => rm(temporary, { recursive: true, force: true }));
  await cp(path.join(directory, 'vendor'), path.join(temporary, 'vendor'), { recursive: true });
  await symlink(path.join(directory, 'node_modules'), path.join(temporary, 'node_modules'), 'dir');
  await cp(path.join(directory, 'vanity.mjs'), path.join(temporary, 'vanity.mjs'));
  const compile = () => build({
    entryPoints: [path.join(temporary, 'vanity.mjs')], bundle: true,
    format: 'iife', platform: 'neutral', target: 'es2018', write: false,
    logLevel: 'silent', plugins: [compactTokenizerPlugin(temporary)],
  });
  return { temporary, compile };
}

test('pinned tokenizer rejects altered source and license bytes', async t => {
  for (const name of ['tokenizer/index.ts', 'LICENSE']) {
    const { temporary, compile } = await fixture(t);
    const filename = path.join(temporary, 'vendor/parse5', name);
    await writeFile(filename, (await readFile(filename, 'utf8')) + '\n');
    await assert.rejects(compile(), /SHA256 mismatch/);
  }
});

test('a changed source checksum cannot silently add a specialization site', async t => {
  const { temporary, compile } = await fixture(t);
  const vendor = path.join(temporary, 'vendor/parse5');
  const filename = path.join(vendor, 'tokenizer/index.ts');
  const source = (await readFile(filename, 'utf8')) + '\n// handler.onParseError\n';
  await writeFile(filename, source);
  const manifest = JSON.parse(await readFile(path.join(vendor, 'SOURCE.json'), 'utf8'));
  manifest.files['tokenizer/index.ts'].sha256 = createHash('sha256').update(source).digest('hex');
  await writeFile(path.join(vendor, 'SOURCE.json'), JSON.stringify(manifest));
  await assert.rejects(compile(), /specialization sites/);
});

test('a different upstream release cannot reuse these build transforms', async t => {
  const { temporary, compile } = await fixture(t);
  const filename = path.join(temporary, 'vendor/parse5/SOURCE.json');
  const manifest = JSON.parse(await readFile(filename, 'utf8'));
  manifest.release = '8.0.2';
  await writeFile(filename, JSON.stringify(manifest));
  await assert.rejects(compile(), /source manifest/);
});
