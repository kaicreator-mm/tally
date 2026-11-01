/**
 * T06l producer-only bundling: mature esbuild packs the existing reviewed
 * T06g native GitHub claim publication witness together with the T06d
 * Host CAS registry grammar into one closed-world DH C provider source.
 * No rewritten parser, verifier, HTTP transport or alternative registry engine:
 * the registry row is read through the pinned public openSqliteCasStore API;
 * better-sqlite3 is an explicitly declared native peer. The native readback
 * reuses packages/tally-github-native-c verbatim, including its pinned
 * ADS v4.18 event-v2 schema bytes.
 *
 * The resulting file is installed as the npm package
 * @kaicreator/tally-native-claim-reader-bundle and sealed by digest into the
 * BUILD C module (claim-admission-shim.mjs). No relative sealed import.
 */
import {build} from 'esbuild';
import {readFileSync,writeFileSync} from 'node:fs';
import {resolve} from 'node:path';
const tally=process.env.TALLY;
const out=process.env.T06L_C_OUTPUT;
const importsOut=process.env.T06L_IMPORTS_OUTPUT;
if(!tally||!out||!importsOut)throw Error('T06L_BUNDLE_PATH_REQUIRED');
const witness=resolve(tally,'packages/tally-github-native-c/src/task-claim-publication-witness.mjs');
// The witness chain uses the canonical ADS v4.18 event-v2 JSON schema on
// disk. Embed that byte-exact schema into the sealed provider bundle,
// preserving its native Git blob integrity check (same transform as T04b).
const claimPath=resolve(tally,'packages/tally-github-native-c/src/task-claim-events.mjs');
const claimOriginal=readFileSync(claimPath,'utf8');
const claimSchema=readFileSync(resolve(tally,'packages/tally-github-native-c/vendor/agent-event-v2.schema.json'));
const claimAnchor="readFileSync(new URL('../vendor/agent-event-v2.schema.json',import.meta.url))";
const claimReplacement=`Buffer.from('${claimSchema.toString('base64')}','base64')`;
const claimReplaced=claimOriginal.replace(claimAnchor,claimReplacement);
if(claimOriginal===claimReplaced)throw Error('PINNED_ADS_EVENT_SCHEMA_EMBED_TRANSFORM_MISSED');
// Sealed provider composition. Host-supplied configuration arrives ONLY via
// deployment env (token, exact source pin, registry path, writer identity);
// B/Agent/caller input never carries authority. Concatenation instead of
// nested template literals keeps the generated source escaping-safe.
const provider=`
import {readGitHubClaimPublicationWitness} from ${JSON.stringify(witness)};
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createHash} from 'node:crypto';
const OWNER='kaicreator-mm',REPO='tally',ISSUE_NUMBER=169,ACTOR_ROLE='builder';
const ISSUE_REF='https://github.com/'+OWNER+'/'+REPO+'/issues/'+ISSUE_NUMBER;
const KEY_PREFIX='tally:v2:claim-admission:';
const SHA=/^[0-9a-f]{40}$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\\[bot\\])?$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const KINDS=new Set(['chatgpt-web','codex','claude-code','human',
  'github-actions','woodpecker','other']);
const WITNESS_REASONS=new Set([
  'GITHUB_PROVIDER_TOKEN_UNAVAILABLE','GITHUB_UNAUTHENTICATED',
  'GITHUB_ACCESS_DENIED','GITHUB_COMMENT_NOT_FOUND','GITHUB_RATE_LIMITED',
  'GITHUB_COMMENT_READ_FAILED','INVALID_EXACT_CLAIM_WITNESS_REQUEST',
  'NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED','PROTECTED_CLAIM_KEY_NOT_UNIQUE',
  'NATIVE_CLAIM_EVENT_RESERVATION_MISMATCH','NATIVE_COMMENT_CURRENTNESS_NOT_VERIFIED'
]);
const denied=(reason,sourceSha)=>Object.freeze({
  state:'NOT_ADMITTED',reason,issueRef:ISSUE_REF,
  sourceSha:SHA.test(sourceSha??'')?sourceSha:null,
  claimPublication:'NOT_VERIFIED',claimAccepted:false,
  authorized:false,admitted:false,taskPass:false,
  agentLaunch:false,releasePass:false});
export async function admitClaimedTask(input){
  const sha=process.env.TALLY_CLAIM_SOURCE_SHA;
  const registryPath=process.env.TALLY_CLAIM_REGISTRY_PATH;
  const dispatch=process.env.TALLY_CLAIM_DISPATCH_ID;
  const operatorId=process.env.TALLY_CLAIM_OPERATOR_ID;
  const operatorKind=process.env.TALLY_CLAIM_OPERATOR_KIND;
  const writerId=process.env.TALLY_CLAIM_WRITER_ID;
  const writerLogin=process.env.TALLY_CLAIM_WRITER_LOGIN;
  const token=process.env.TALLY_GITHUB_READ_TOKEN;
  const pinsValid=SHA.test(sha??'')&&typeof registryPath==='string'&&
    registryPath.length>0&&ID.test(dispatch??'')&&ID.test(operatorId??'')&&
    KINDS.has(operatorKind)&&ID.test(writerId??'')&&LOGIN.test(writerLogin??'');
  if(!pinsValid||input?.sourceRef!==ISSUE_REF||input?.issueNumber!==ISSUE_NUMBER||
     input?.sourceSha!==sha)
    return denied('CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',sha);
  // Read the protected Claim row from the real DH Host CAS registry. The
  // registry — not the caller and not the comment — is the authority for
  // publication state; the live GitHub readback below proves currentness.
  const protectedClaimKey=OWNER+'/'+REPO+'#'+ISSUE_NUMBER+':'+ACTOR_ROLE+':__default__';
  const key=KEY_PREFIX+createHash('sha256').update(protectedClaimKey).digest('hex');
  let row;
  try{
    const store=await openSqliteCasStore({path:registryPath,busyTimeoutMs:1000});
    try{row=await store.get(key);}finally{await store.close();}
  }catch{return denied('CLAIM_REGISTRY_UNAVAILABLE',sha);}
  if(row===undefined||row===null)return denied('CLAIM_REGISTRY_RECORD_NOT_FOUND',sha);
  let value=row?.value;
  if(!value||typeof value!=='object'||Array.isArray(value)||
     !Number.isSafeInteger(row.revision)||row.revision<1)
    return denied('CLAIM_REGISTRY_RECORD_INVALID',sha);
  if(typeof value.protectedClaimKey!=='string'||
     typeof value.repository!=='string'||typeof value.issueRef!=='string'||
     !SHA.test(value.sourceSha??'')||typeof value.actorRole!=='string'||
     !ID.test(value.dispatchId??'')||!ID.test(value.operatorId??'')||
     !KINDS.has(value.operatorKind)||!ID.test(value.writerId??'')||
     !Number.isSafeInteger(value.commentId)||value.commentId<1||
     typeof value.commentRef!=='string'||value.generation!==1)
    return denied('CLAIM_REGISTRY_RECORD_INVALID',sha);
  if(value.phase!=='PUBLISHED_VERIFIED')
    return denied('CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED',sha);
  if(value.protectedClaimKey!==protectedClaimKey||
     value.repository!==OWNER+'/'+REPO||value.issueRef!==ISSUE_REF||
     value.sourceSha!==sha||value.actorRole!==ACTOR_ROLE||
     value.dispatchId!==dispatch||value.operatorId!==operatorId||
     value.operatorKind!==operatorKind||value.writerId!==writerId||
     value.commentRef!==ISSUE_REF+'#issuecomment-'+value.commentId)
    return denied('CLAIM_REGISTRY_RESERVATION_MISMATCH',sha);
  const witness=await readGitHubClaimPublicationWitness({
    token,owner:OWNER,repo:REPO,issueNumber:ISSUE_NUMBER,expectedSourceSha:sha,
    commentId:value.commentId,dispatchId:value.dispatchId,actorRole:ACTOR_ROLE,
    operatorKind:value.operatorKind,operatorId:value.operatorId,
    protectedClaimKey,claimGeneration:1,writerLogin});
  if(!witness||witness.status!=='PUBLICATION_WITNESSED')
    return denied(WITNESS_REASONS.has(witness?.reason)?witness.reason:
      'NATIVE_CLAIM_WITNESS_NOT_VERIFIED',sha);
  // The witness itself is readback-only: its claimAccepted stays false and
  // the accepted-Claim observation below is derived from the CAS row plus
  // this current native readback, never from a comment self-declared flag.
  if(witness.claimAccepted!==false||witness.authorized!==false||
     witness.admitted!==false||witness.taskPass!==false||witness.releasePass!==false||
     witness.claimPublication!=='NATIVE_COMMENT_READBACK_ONLY'||
     witness.sourceQualified!==true||witness.issueRef!==ISSUE_REF||
     witness.sourceSha!==sha||witness.commentId!==value.commentId||
     witness.commentRef!==value.commentRef||witness.dispatchId!==value.dispatchId||
     witness.actorRole!==ACTOR_ROLE||witness.operatorKind!==value.operatorKind||
     witness.operatorId!==value.operatorId||witness.protectedClaimKey!==protectedClaimKey||
     witness.generation!==1||witness.transportActor!==writerLogin)
    return denied('NATIVE_CLAIM_WITNESS_NOT_VERIFIED',sha);
  return Object.freeze({
    state:'ADMITTED_PENDING_AGENT_EFFECT',
    reason:'CLAIM_REGISTRY_PUBLICATION_AND_NATIVE_READBACK_VERIFIED',
    issueRef:ISSUE_REF,sourceSha:sha,protectedClaimKey,generation:1,
    dispatchId:value.dispatchId,actorRole:ACTOR_ROLE,
    operatorKind:value.operatorKind,operatorId:value.operatorId,
    commentId:value.commentId,commentRef:value.commentRef,
    claimPublication:'REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK',
    claimAccepted:true,admitted:true,authorized:false,taskPass:false,
    agentLaunch:false,releasePass:false});
}
`;
const result=await build({
  stdin:{contents:provider,resolveDir:process.cwd(),sourcefile:'claim-admission-provider.mjs',
    loader:'js'},
  bundle:true,format:'esm',platform:'node',target:'node24',
  external:['better-sqlite3'],
  write:false,metafile:true,legalComments:'none',logLevel:'warning',
  // Only the separately installed trusted npm provider may use ordinary Node
  // createRequire for upstream CJS deps. The selected K C module is an ESM
  // shim with a static declared npm import; no BUILD policy relaxation.
  banner:{js:'import {createRequire as __nativeRequireFactory} from "node:module"; const require = __nativeRequireFactory(import.meta.url);'},
  plugins:[{name:'pinned-ads-event-schema-bytes',setup(b){
    b.onLoad({filter:/[/\\]tally-github-native-c[/\\]src[/\\]task-claim-events\.mjs$/},()=>({
      contents:claimReplaced,loader:'js',
      resolveDir:resolve(tally,'packages/tally-github-native-c/src')
    }));
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
if(externals.some(x=>!x.startsWith('node:')&&x!=='better-sqlite3'))throw Error('UNSELECTED_BARE_DEPENDENCY:'+externals.join(','));
for(const name of ['process','buffer']){
  if(code.includes('from "'+name+'"')||code.includes("from '"+name+"'"))
    throw Error('NODE_IMPORT_NOT_CANONICALIZED:'+name);
}
writeFileSync(out,code);
writeFileSync(importsOut,JSON.stringify(externals));
console.log(JSON.stringify({sealedClaimCBytes:Buffer.byteLength(code),
  schemaBytes:claimSchema.length,nativeClaimExternalImports:externals,
  hasTokenBytes:code.includes(process.env.TALLY_GITHUB_READ_TOKEN||'__NO_SECRET_SET__')}));
console.log('T06L_EXISTING_CLAIM_WITNESS_BUNDLED=PASS');
