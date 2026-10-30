/** T06m producer-only sealed Local Codex provider bundle. */
import {build} from 'esbuild';
import {readFileSync,writeFileSync} from 'node:fs';
import {resolve} from 'node:path';
const tally=process.env.TALLY;
const out=process.env.T06M_C_OUTPUT;
const importsOut=process.env.T06M_IMPORTS_OUTPUT;
if(!tally||!out||!importsOut)throw Error('T06M_BUNDLE_PATH_REQUIRED');
const witness=resolve(tally,'packages/tally-github-native-c/src/task-claim-publication-witness.mjs');
const sourcePair=resolve(tally,'packages/tally-github-native-c/src/task-source-pair.mjs');
const host=resolve(tally,'experiments/v2/selected-business-build/local-codex-host.mjs');
const claimPath=resolve(tally,'packages/tally-github-native-c/src/task-claim-events.mjs');
const claimOriginal=readFileSync(claimPath,'utf8');
const claimSchema=readFileSync(resolve(tally,'packages/tally-github-native-c/vendor/agent-event-v2.schema.json'));
const claimAnchor="readFileSync(new URL('../vendor/agent-event-v2.schema.json',import.meta.url))";
const claimReplacement=`Buffer.from('${claimSchema.toString('base64')}','base64')`;
const claimReplaced=claimOriginal.replace(claimAnchor,claimReplacement);
if(claimOriginal===claimReplaced)throw Error('PINNED_ADS_EVENT_SCHEMA_EMBED_TRANSFORM_MISSED');
const provider=`
import {readGitHubClaimPublicationWitness} from ${JSON.stringify(witness)};
import {readGitHubNativeTaskSourcePair} from ${JSON.stringify(sourcePair)};
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createLocalCodexHost} from ${JSON.stringify(host)};
import {createHash} from 'node:crypto';
const OWNER='kaicreator-mm',REPO='tally',ISSUE_NUMBER=172,ACTOR_ROLE='builder';
const ISSUE_REF='https://github.com/'+OWNER+'/'+REPO+'/issues/'+ISSUE_NUMBER;
const CLAIM_KEY=OWNER+'/'+REPO+'#'+ISSUE_NUMBER+':'+ACTOR_ROLE+':__default__';
const REGISTRY_KEY='tally:v2:claim-admission:'+createHash('sha256').update(CLAIM_KEY).digest('hex');
const WRITESET_KEY='tally:v2:task-writeset:'+ISSUE_NUMBER;
const SHA=/^[0-9a-f]{40}$/,ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\\[bot\\])?$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions','woodpecker','other']);
const base=input=>({issueRef:ISSUE_REF,sourceSha:input.sourceSha,attemptId:input.attemptId,
  claimGeneration:1,authorized:false,taskPass:false,releasePass:false,replayFixture:false});
const denied=(input,reason)=>({...base(input),state:'NOT_ADMITTED',reason,claimAccepted:false,
  admitted:false,agentLaunch:false,candidateTree:null,candidateCommit:null,process:null});
const admitted=(input,state,reason,agentLaunch,more={})=>({...base(input),state,reason,
  claimAccepted:true,admitted:true,agentLaunch,candidateTree:null,candidateCommit:null,process:null,...more});
function validClaimRow(row,env){
  const v=row?.value;
  return Number.isSafeInteger(row?.revision)&&row.revision>=1&&v?.phase==='PUBLISHED_VERIFIED'&&
    v.generation===1&&v.protectedClaimKey===CLAIM_KEY&&v.repository===OWNER+'/'+REPO&&
    v.issueRef===ISSUE_REF&&v.sourceSha===env.TALLY_CLAIM_SOURCE_SHA&&v.actorRole===ACTOR_ROLE&&
    v.dispatchId===env.TALLY_CLAIM_DISPATCH_ID&&v.operatorId===env.TALLY_CLAIM_OPERATOR_ID&&
    v.operatorKind===env.TALLY_CLAIM_OPERATOR_KIND&&v.writerId===env.TALLY_CLAIM_WRITER_ID&&
    Number.isSafeInteger(v.commentId)&&v.commentId>0&&
    v.commentRef===ISSUE_REF+'#issuecomment-'+v.commentId;
}
export async function runLocalCodexAttempt(input){
  const env=process.env;
  if(input?.sourceRef!==ISSUE_REF||input.issueNumber!==ISSUE_NUMBER||
    !SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.attemptId??'')||
    input.sourceSha!==env.TALLY_CLAIM_SOURCE_SHA||!ID.test(env.TALLY_CLAIM_DISPATCH_ID??'')||
    !ID.test(env.TALLY_CLAIM_OPERATOR_ID??'')||!KINDS.has(env.TALLY_CLAIM_OPERATOR_KIND)||
    !ID.test(env.TALLY_CLAIM_WRITER_ID??'')||!LOGIN.test(env.TALLY_CLAIM_WRITER_LOGIN??'')||
    !env.TALLY_CLAIM_REGISTRY_PATH||!env.TALLY_LOCAL_WRITESET_SHA256)
    return denied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  let registry,row,writeSet;
  try{
    registry=await openSqliteCasStore({path:env.TALLY_CLAIM_REGISTRY_PATH,busyTimeoutMs:1000});
    row=await registry.get(REGISTRY_KEY);writeSet=await registry.get(WRITESET_KEY);
  }catch{return denied(input,'CLAIM_REGISTRY_UNAVAILABLE');
  }finally{if(registry)await registry.close();}
  if(!row)return denied(input,'CLAIM_REGISTRY_RECORD_NOT_FOUND');
  if(row.value?.phase!=='PUBLISHED_VERIFIED')return denied(input,'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED');
  if(!validClaimRow(row,env))return denied(input,'CLAIM_REGISTRY_RESERVATION_MISMATCH');
  if(writeSet?.value?.phase!=='FROZEN_DAG_WRITESET'||
    writeSet.value.sourceSha!==input.sourceSha||
    writeSet.value.writeSetSha256!==env.TALLY_LOCAL_WRITESET_SHA256||
    !writeSet.value.document?.includes('experiments/v2/selected-business-build/LOCAL_CODEX_L3.md'))
    return denied(input,'DAG_WRITESET_NOT_VERIFIED');
  const token=env.TALLY_GITHUB_READ_TOKEN;
  if(typeof token!=='string'||token.length<8)return denied(input,'LOCAL_AGENT_UNAVAILABLE');
  let pair;
  try{pair=await readGitHubNativeTaskSourcePair({token,owner:OWNER,repo:REPO,
    issueNumber:ISSUE_NUMBER,expectedSourceSha:input.sourceSha,limit:1000});
  }catch{return denied(input,'NATIVE_SOURCE_NOT_CURRENT');}
  if(pair?.workItem?.dependencyState==='BLOCKED')return denied(input,'NATIVE_DEPENDENCIES_BLOCKED');
  if(pair?.status!=='SOURCE_PAIR_OBSERVED'||pair.sourceQualified!==true||
    pair.issueRef!==ISSUE_REF||pair.sourceSha!==input.sourceSha||
    pair.workItem?.issueState!=='open'||pair.workItem?.workflowState!=='state:ready'||
    pair.workItem?.workItemType!=='type:task'||pair.workItem?.dependencyState!=='CLEAR')
    return denied(input,'NATIVE_SOURCE_NOT_CURRENT');
  let witness;
  try{witness=await readGitHubClaimPublicationWitness({token,owner:OWNER,repo:REPO,
    issueNumber:ISSUE_NUMBER,expectedSourceSha:input.sourceSha,commentId:row.value.commentId,
    dispatchId:row.value.dispatchId,actorRole:ACTOR_ROLE,operatorKind:row.value.operatorKind,
    operatorId:row.value.operatorId,protectedClaimKey:CLAIM_KEY,claimGeneration:1,
    writerLogin:env.TALLY_CLAIM_WRITER_LOGIN});
  }catch{return denied(input,'NATIVE_CLAIM_WITNESS_NOT_VERIFIED');}
  if(witness?.status!=='PUBLICATION_WITNESSED'||witness.claimAccepted!==false||
    witness.authorized!==false||witness.admitted!==false||witness.taskPass!==false||
    witness.releasePass!==false||witness.sourceQualified!==true||witness.issueRef!==ISSUE_REF||
    witness.sourceSha!==input.sourceSha||witness.commentId!==row.value.commentId||
    witness.commentRef!==row.value.commentRef||witness.dispatchId!==row.value.dispatchId||
    witness.actorRole!==ACTOR_ROLE||witness.operatorKind!==row.value.operatorKind||
    witness.operatorId!==row.value.operatorId||witness.protectedClaimKey!==CLAIM_KEY||
    witness.generation!==1||witness.transportActor!==env.TALLY_CLAIM_WRITER_LOGIN)
    return denied(input,'NATIVE_CLAIM_WITNESS_NOT_VERIFIED');
  if(env.TALLY_LOCAL_REPLAY==='1')
    return {...admitted(input,'REPLAY_COMPLETE','REPLAY_FIXTURE_ONLY',false),replayFixture:true};
  try{
    return await createLocalCodexHost().runAttempt({issueRef:ISSUE_REF,sourceSha:input.sourceSha,
      attemptId:input.attemptId,hostStorePath:env.TALLY_LOCAL_HOST_STORE_PATH,
      repositoryPath:env.TALLY_LOCAL_REPOSITORY_PATH,worktreePath:env.TALLY_LOCAL_WORKTREE_PATH,
      cliPath:env.TALLY_CODEX_CLI_PATH,sandboxCliPath:env.TALLY_CODEX_SANDBOX_PATH,
      codexHome:env.TALLY_CODEX_HOME,builderProfile:env.TALLY_BUILDER_PROFILE,
      builderTemp:env.TALLY_BUILDER_TEMP,emptyGitConfig:env.TALLY_EMPTY_GIT_CONFIG,
      operatorHome:env.TALLY_OPERATOR_HOME,hiddenFixture:env.TALLY_HIDDEN_FIXTURE,
      builderPrincipal:env.TALLY_BUILDER_PRINCIPAL,builderSid:env.TALLY_BUILDER_SID,
      verifierSid:env.TALLY_VERIFIER_SID,sshAddPath:env.TALLY_SSH_ADD_PATH,
      gitPath:env.TALLY_GIT_PATH,powershellPath:env.TALLY_POWERSHELL_PATH,
      modelProxy:env.TALLY_MODEL_PROXY,caBundle:env.TALLY_CA_BUNDLE,timeoutMs:Number(env.TALLY_CODEX_TIMEOUT_MS??300000)});
  }catch{
    return admitted(input,'LOCAL_ATTEMPT_UNKNOWN','ATTEMPT_STATUS_UNKNOWN',true,
      {process:{status:'UNKNOWN',exitCode:null,pid:null,builderPrincipal:env.TALLY_BUILDER_PRINCIPAL,
        builderSid:env.TALLY_BUILDER_SID,startedAt:null,terminatedAt:null}});
  }
}
`;
const result=await build({
  stdin:{contents:provider,resolveDir:process.cwd(),sourcefile:'local-codex-provider.mjs',loader:'js'},
  bundle:true,format:'esm',platform:'node',target:'node24',
  external:['better-sqlite3','@kaicreator/domain-harness-host-node'],
  write:false,metafile:true,legalComments:'none',logLevel:'warning',
  banner:{js:'import {createRequire as __nativeRequireFactory} from "node:module"; const require = __nativeRequireFactory(import.meta.url);'},
  plugins:[{name:'pinned-ads-event-schema-bytes',setup(b){
    b.onLoad({filter:/[/\\]tally-github-native-c[/\\]src[/\\]task-claim-events\.mjs$/},()=>({
      contents:claimReplaced,loader:'js',resolveDir:resolve(tally,'packages/tally-github-native-c/src')}));
  }}]
});
if(result.outputFiles.length!==1)throw Error('EXPECTED_ONE_SEALED_MODULE');
let code=result.outputFiles[0].text;
const originals=[...new Set(Object.values(result.metafile.outputs).flatMap(o=>o.imports)
  .filter(i=>i.external).map(i=>i.path))].sort();
for(const name of ['process','buffer']){
  code=code.replaceAll('from "'+name+'"','from "node:'+name+'"');
  code=code.replaceAll("from '"+name+"'","from 'node:"+name+"'");
}
const externals=originals.map(x=>['process','buffer'].includes(x)?'node:'+x:x);
const allowedExternals=new Set(['better-sqlite3','@kaicreator/domain-harness-host-node']);
if(externals.some(x=>!x.startsWith('node:')&&!allowedExternals.has(x)))
  throw Error('UNSELECTED_BARE_DEPENDENCY:'+externals.join(','));
writeFileSync(out,code);
writeFileSync(importsOut,JSON.stringify(externals));
const hasTokenBytes=code.includes(process.env.TALLY_GITHUB_READ_TOKEN||'__NO_SECRET_SET__');
if(hasTokenBytes)throw Error('SEALED_PROVIDER_CONTAINS_HOST_TOKEN');
console.log(JSON.stringify({sealedLocalCodexCBytes:Buffer.byteLength(code),
  nativeLocalCodexExternalImports:externals,hasTokenBytes}));
console.log('T06M_LOCAL_CODEX_PROVIDER_BUNDLED=PASS');
