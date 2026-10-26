/**
 * T04b producer-only bundling: mature esbuild packs existing reviewed T03b
 * Octokit/YAML/Ajv/JCS reader into one closed-world DH C module source.
 * The pinned schema stays byte-exact; its runtime import.meta file read is
 * replaced with the same byte payload to support sealed data-module loading.
 * No rewritten parser, verifier or novel HTTP transport.
 */
import {build} from 'esbuild';
import {readFileSync,writeFileSync} from 'node:fs';
import {resolve} from 'node:path';
const tally=process.env.TALLY;
const out=process.env.T04B_C_OUTPUT;
const importsOut=process.env.T04B_IMPORTS_OUTPUT;
if(!tally||!out||!importsOut)throw Error('T04B_BUNDLE_PATH_REQUIRED');
const entry=resolve(tally,'experiments/v2/selected-business-build/native-readonly-capability.mjs');
const validator=resolve(tally,'packages/tally-ads-v5-validator/src/validate.mjs');
const schema=readFileSync(resolve(tally,'packages/tally-ads-v5-validator/vendor/report.schema.json'));
const original=readFileSync(validator,'utf8');
const replaced=original.replace(
  "readFileSync(new URL('../vendor/report.schema.json', import.meta.url))",
  `Buffer.from('${schema.toString('base64')}','base64')`);
if(replaced===original)throw Error('PINNED_ADS_SCHEMA_EMBED_TRANSFORM_MISSED');
const result=await build({
  entryPoints:[entry],bundle:true,format:'esm',platform:'node',target:'node24',
  write:false,metafile:true,legalComments:'none',logLevel:'warning',
  plugins:[{name:'pinned-ads-schema-bytes',setup(b){
    b.onLoad({filter:/[/\\]tally-ads-v5-validator[/\\]src[/\\]validate\.mjs$/},()=>{
      return {contents:replaced,loader:'js',resolveDir:resolve(tally,'packages/tally-ads-v5-validator/src')};
    });
  }}]
});
if(result.outputFiles.length!==1)throw Error('EXPECTED_ONE_SEALED_MODULE');
const code=result.outputFiles[0].text;
const externals=[...new Set(Object.values(result.metafile.outputs)
  .flatMap(o=>o.imports).filter(i=>i.external).map(i=>i.path))].sort();
if(externals.some(x=>!x.startsWith('node:')))throw Error('UNSELECTED_BARE_DEPENDENCY:'+externals.join(','));
writeFileSync(out,code);
writeFileSync(importsOut,JSON.stringify(externals));
console.log(JSON.stringify({sealedCBytes:Buffer.byteLength(code),schemaBytes:schema.length,
  nativeCExternalImports:externals,hasTokenBytes:code.includes(process.env.TALLY_GITHUB_READ_TOKEN||'__NO_SECRET_SET__')}));
console.log('T04B_EXISTING_READER_BUNDLED=PASS');
