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
const entry=resolve(tally,'experiments/v2/selected-business-build/'+
  (process.env.T06F_NATIVE_PAIR==='1'?'native-pair-capability.mjs':'native-readonly-capability.mjs'));
const validator=resolve(tally,'packages/tally-ads-v5-validator/src/validate.mjs');
const schema=readFileSync(resolve(tally,'packages/tally-ads-v5-validator/vendor/report.schema.json'));
const original=readFileSync(validator,'utf8');
const replaced=original.replace(
  "readFileSync(new URL('../vendor/report.schema.json', import.meta.url))",
  `Buffer.from('${schema.toString('base64')}','base64')`);
if(replaced===original)throw Error('PINNED_ADS_SCHEMA_EMBED_TRANSFORM_MISSED');
// The new pair reader uses the canonical ADS v4.18 event-v2 JSON schema
// on disk. Embed that byte-exact schema into the sealed provider bundle,
// preserving its native Git blob integrity check without private SDK imports.
const claimPath=resolve(tally,'packages/tally-github-native-c/src/task-claim-events.mjs');
const claimOriginal=readFileSync(claimPath,'utf8');
const claimSchema=readFileSync(resolve(tally,'packages/tally-github-native-c/vendor/agent-event-v2.schema.json'));
const claimAnchor="readFileSync(new URL('../vendor/agent-event-v2.schema.json',import.meta.url))";
const claimReplacement=`Buffer.from('${claimSchema.toString('base64')}','base64')`;
const claimReplaced=claimOriginal.replace(claimAnchor,claimReplacement);
if(process.env.T06F_NATIVE_PAIR==='1'&&claimOriginal===claimReplaced)
  throw Error('PINNED_ADS_EVENT_SCHEMA_EMBED_TRANSFORM_MISSED');
const result=await build({
  entryPoints:[entry],bundle:true,format:'esm',platform:'node',target:'node24',
  write:false,metafile:true,legalComments:'none',logLevel:'warning',
  // Only the separately installed trusted npm provider may use ordinary Node
  // createRequire for upstream CJS deps. The selected K C module is an ESM
  // shim with a static declared npm import; no BUILD policy relaxation.
  banner:{js:'import {createRequire as __nativeRequireFactory} from "node:module"; const require = __nativeRequireFactory(import.meta.url);'},
  plugins:[{name:'pinned-ads-schema-bytes',setup(b){
    b.onLoad({filter:/[/\\]tally-github-native-c[/\\]src[/\\]task-claim-events\.mjs$/},()=>({
      contents:claimReplaced,loader:'js',
      resolveDir:resolve(tally,'packages/tally-github-native-c/src')
    })),
    b.onLoad({filter:/[/\\]tally-ads-v5-validator[/\\]src[/\\]validate\.mjs$/},()=>{
      return {contents:replaced,loader:'js',resolveDir:resolve(tally,'packages/tally-ads-v5-validator/src')};
    });
  }}]
});
if(result.outputFiles.length!==1)throw Error('EXPECTED_ONE_SEALED_MODULE');
// esbuild preserves Node's two old bare builtins as "process" and "buffer".
 // Rewrite ONLY their imported specifiers to explicit node: form so the
 // DH closed-world declared node-import manifest exactly matches source.
let code=result.outputFiles[0].text;
const originals=[...new Set(Object.values(result.metafile.outputs)
  .flatMap(o=>o.imports).filter(i=>i.external).map(i=>i.path))].sort();
for(const name of ['process','buffer']){
  code=code.replaceAll('from "'+name+'"','from "node:'+name+'"');
  code=code.replaceAll("from '"+name+"'","from 'node:"+name+"'");
}
const externals=originals.map(x=>['process','buffer'].includes(x)?'node:'+x:x);
if(externals.some(x=>!x.startsWith('node:')))throw Error('UNSELECTED_BARE_DEPENDENCY:'+externals.join(','));
for(const name of ['process','buffer']){
  if(code.includes('from "'+name+'"')||code.includes("from '"+name+"'"))
    throw Error('NODE_IMPORT_NOT_CANONICALIZED:'+name);
}
writeFileSync(out,code);
writeFileSync(importsOut,JSON.stringify(externals));
console.log(JSON.stringify({sealedCBytes:Buffer.byteLength(code),schemaBytes:schema.length,
  nativeCExternalImports:externals,hasTokenBytes:code.includes(process.env.TALLY_GITHUB_READ_TOKEN||'__NO_SECRET_SET__')}));
console.log('T04B_EXISTING_READER_BUNDLED=PASS');
