import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import path from 'node:path';

const upstreamCommit = '0d56627fc924d40f560fd260ade0e1a935e2369c';
const vendoredFiles = ['LICENSE', 'common/token.ts', 'common/unicode.ts', 'tokenizer/index.ts', 'tokenizer/preprocessor.ts'];
const transforms = {
  'tokenizer/index.ts': { parseErrors: 2, locations: 1, diagnostics: 108 },
  'tokenizer/preprocessor.ts': { parseErrors: 3, locations: 0, diagnostics: 3 },
};

// Build-only specialization for this closed assertion adapter. The adapter
// passes no onParseError handler, disables source locations, and reads tagName
// rather than the tree parser's derived tagID. Token states, Unicode/entity
// decoding, emitted attributes, comments and script handling stay unchanged.
// The original sources stay byte-identical in vendor/; transforms happen only
// after every pinned source and its license have passed SHA256 verification.
export function compactTokenizerPlugin(directory) {
  const vendor = path.resolve(directory, 'vendor/parse5');
  const filenames = new Map(vendoredFiles.map(name => [name, path.join(vendor, name)]));
  const sourceNames = new Map([...filenames].map(([name, filename]) => [filename, name]));
  let verified = new Map();
  let errorEnum = '';

  return {
    name: 'compact-pinned-parse5-tokenizer',
    setup(build) {
      build.onStart(async () => {
        verified = new Map();
        errorEnum = '';
        const manifest = JSON.parse(await readFile(path.join(vendor, 'SOURCE.json'), 'utf8'));
        if (manifest.version !== 1 || manifest.name !== 'parse5' || manifest.release !== '8.0.1'
          || manifest.repository !== 'https://github.com/inikulin/parse5' || manifest.commit !== upstreamCommit
          || !manifest.files || typeof manifest.files !== 'object' || Array.isArray(manifest.files)
          || JSON.stringify(Object.keys(manifest.files).sort()) !== JSON.stringify(vendoredFiles)) {
          throw new Error('Unexpected pinned parse5 source manifest');
        }
        const entries = await Promise.all(vendoredFiles.map(async name => {
          const entry = manifest.files[name];
          const upstream = name === 'LICENSE' ? name : 'packages/parse5/lib/' + name;
          if (!entry || entry.upstream !== upstream || !/^[a-f0-9]{64}$/.test(entry.sha256)) {
            throw new Error('Invalid parse5 source provenance: ' + name);
          }
          const bytes = await readFile(filenames.get(name));
          if (createHash('sha256').update(bytes).digest('hex') !== entry.sha256) {
            throw new Error('Pinned parse5 source SHA256 mismatch: ' + name);
          }
          return [name, bytes.toString('utf8')];
        }));
        const original = new Map(entries);
        const errors = new Set();
        for (const [name, expected] of Object.entries(transforms)) {
          const source = original.get(name);
          const parseErrors = [...source.matchAll(/(?:this\.)?handler\.onParseError/g)].length;
          const locations = [...source.matchAll(/this\.options\.sourceCodeLocationInfo/g)].length;
          const diagnostics = [...source.matchAll(/this\._err\(/g)].length;
          if (parseErrors !== expected.parseErrors || locations !== expected.locations || diagnostics !== expected.diagnostics) {
            throw new Error('Unexpected pinned parse5 specialization sites: ' + name);
          }
          for (const match of source.matchAll(/\bERR\.([A-Za-z]+)/g)) errors.add(match[1]);
          original.set(name, source
            .replace(/(?:this\.)?handler\.onParseError/g, 'null')
            .replace(/this\.options\.sourceCodeLocationInfo/g, 'false')
            // Each _err call is a verified no-op with no observable arguments.
            .replace(/this\._err\(/g, '/* @__PURE__ */ this._err('));
        }
        // Error identifiers are never observed when callbacks are disabled.
        // A const enum allows all 111 no-op diagnostic calls to be removed.
        errorEnum = 'export const enum ERR{' + [...errors].sort().map(name => name + '=0').join(',') + '}';
        const entities = path.join(directory, 'node_modules/entities');
        const entitiesPackage = JSON.parse(await readFile(path.join(entities, 'package.json'), 'utf8'));
        if (entitiesPackage.version !== '8.1.0') throw new Error('Expected pinned entities 8.1.0 decoder sources');
        await readFile(path.join(entities, 'src/decode.ts'));
        verified = original;
      });

      // Only vendored parse5 imports receive these substitutions. Imports
      // within entities and all unrelated build inputs resolve normally.
      build.onResolve({ filter: /.*/ }, args => {
        if (args.namespace !== 'file' || !sourceNames.has(args.importer)) return;
        const name = sourceNames.get(args.importer);
        const imports = name === 'tokenizer/index.ts'
          ? {
              './preprocessor.js': 'tokenizer/preprocessor.ts',
              '../common/unicode.js': 'common/unicode.ts',
              '../common/token.js': 'common/token.ts',
            }
          : name === 'tokenizer/preprocessor.ts'
            ? { '../common/unicode.js': 'common/unicode.ts' }
            : {};
        if (Object.hasOwn(imports, args.path)) return { path: filenames.get(imports[args.path]) };
        if (args.path === 'entities/decode' && name === 'tokenizer/index.ts') {
          return { path: path.resolve(directory, 'node_modules/entities/src/decode.ts') };
        }
        if (args.path === '../common/error-codes.js' && Object.hasOwn(transforms, name)) {
          return { path: 'errors', namespace: 'closed-parse5-options' };
        }
        if (args.path === '../common/html.js' && name === 'tokenizer/index.ts') {
          return { path: 'html', namespace: 'closed-parse5-options' };
        }
        // The common/token.ts html import is type-only and erased by esbuild.
        throw new Error('Unexpected vendored parse5 runtime import: ' + name + ' -> ' + args.path);
      });

      build.onLoad({ filter: /\.ts$/, namespace: 'file' }, args => {
        const name = sourceNames.get(args.path);
        if (!name) return;
        if (!verified.has(name)) throw new Error('Vendored parse5 source was not verified: ' + name);
        return { contents: verified.get(name), loader: 'ts', resolveDir: path.dirname(args.path) };
      });
      build.onLoad({ filter: /.*/, namespace: 'closed-parse5-options' }, args => {
        if (args.path === 'html') {
          return { contents: 'export const TAG_ID={UNKNOWN:0};export function getTagID(){return 0;}', loader: 'js' };
        }
        if (args.path === 'errors' && errorEnum) return { contents: errorEnum, loader: 'ts' };
        throw new Error('Unexpected parse5 specialization module');
      });
    },
  };
}
