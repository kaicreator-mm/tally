import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

// A package-scoped clean install must bring the actual guided-intake Ajv import.
test('engineering business declares its direct guided-intake Ajv dependency',async()=>{
  const manifest=JSON.parse(await readFile(new URL('../package.json',import.meta.url),'utf8'));
  const source=await readFile(new URL('../src/guided-intake.mjs',import.meta.url),'utf8');
  assert.equal(manifest.dependencies?.ajv,'8.17.1');
  assert.match(source,/from ['"]ajv\\/dist\\/2020\\.js['"]/);
});
