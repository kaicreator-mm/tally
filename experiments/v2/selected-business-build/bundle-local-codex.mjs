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
const candidateHost=resolve(tally,'experiments/v2/candidate-publication/host-publisher.mjs');
const candidateC=resolve(tally,'packages/tally-github-c/src/candidate-publication.mjs');
const intakeHostModule=resolve(tally,'experiments/v2/project-intake/host-project.mjs');
const intakeReadModel=resolve(tally,'packages/tally-github-c/src/project-intake.mjs');
const maintenanceHostModule=resolve(tally,'experiments/v2/maintenance/host-maintenance.mjs');
const maintenanceEffects=resolve(tally,'packages/tally-github-c/src/maintenance-effects.mjs');
const octokit=resolve(tally,'packages/tally-github-native-c/node_modules/@octokit/rest/dist-src/index.js');
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
import {Octokit} from ${JSON.stringify(octokit)};
import {createCandidatePublisher} from ${JSON.stringify(candidateHost)};
import {publishCandidatePullRequestFromClient} from ${JSON.stringify(candidateC)};
import {createProjectIntakeHost,freezeProductIntake as hostFreezeProductIntake} from ${JSON.stringify(intakeHostModule)};
import {readProjectIntakeFromClient} from ${JSON.stringify(intakeReadModel)};
import {createMaintenanceHost} from ${JSON.stringify(maintenanceHostModule)};
import {publishMaintenanceIssueFromClient,findMaintenanceIssueFromClient,
  readMaintenanceIssueFromClient} from ${JSON.stringify(maintenanceEffects)};
import {createHash} from 'node:crypto';
const OWNER='kaicreator-mm',REPO='tally';
// T06n Host-owned task binding map: caller input may select ONLY one of these
// issue keys; every claim/config value comes from issue-keyed Host env, never
// from caller authority. #172 behavior is preserved exactly. #210 is the
// T210 governed project-intake Builder (no candidate file; Host CAS freeze).
const TASK_BINDINGS={
  172:{actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/LOCAL_CODEX_L3.md'},
  177:{actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_A.md'},
  178:{actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_B.md'},
  179:{actorRole:'reviewer'},
  196:{actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/LOCAL_CODEX_L3.md'},
  197:{actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_A.md'},
  198:{actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_B.md'},
  199:{actorRole:'reviewer'},
  200:{actorRole:'reviewer'},
  203:{actorRole:'builder'},
  210:{actorRole:'builder'},
  219:{actorRole:'builder',candidateRelativePath:'experiments/v2/maintenance/host-maintenance.mjs'}
};
const issueRefFor=issueNumber=>'https://github.com/'+OWNER+'/'+REPO+'/issues/'+issueNumber;
const claimKeyFor=(issueNumber,actorRole)=>OWNER+'/'+REPO+'#'+issueNumber+':'+actorRole+':__default__';
const registryKeyFor=claimKey=>'tally:v2:claim-admission:'+createHash('sha256').update(claimKey).digest('hex');
const writeSetKeyFor=issueNumber=>'tally:v2:task-writeset:'+issueNumber;
// Host claim configuration: #172 keeps the legacy TALLY_CLAIM_* names; the
// concurrent Builders #177/#178 and the Reviewer #179 use issue-keyed names,
// each with its own registry path — never one synthetic shared authority.
function claimEnv(env,issueNumber){
  const p=issueNumber===172?'TALLY_CLAIM_':'TALLY_TASK_CLAIM_'+issueNumber+'_';
  return {sourceSha:env.TALLY_CLAIM_SOURCE_SHA,
    dispatchId:env[p+'DISPATCH_ID'],operatorId:env[p+'OPERATOR_ID'],
    operatorKind:env[p+'OPERATOR_KIND'],writerId:env[p+'WRITER_ID'],
    writerLogin:env[p+'WRITER_LOGIN'],commentId:Number(env[p+'COMMENT_ID']),
    registryPath:env[p+'REGISTRY_PATH'],
    writeSetSha256:issueNumber===172?env.TALLY_LOCAL_WRITESET_SHA256:env[p+'WRITESET_SHA256']};
}
const SHA=/^[0-9a-f]{40}$/,ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\\[bot\\])?$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions','woodpecker','other']);
const base=input=>({issueRef:issueRefFor(input.issueNumber),sourceSha:input.sourceSha,
  attemptId:input.attemptId,claimGeneration:1,authorized:false,taskPass:false,
  releasePass:false,replayFixture:false});
const joinBase=input=>({issueRef:issueRefFor(input.issueNumber),sourceSha:input.sourceSha,
  joinId:input.joinId,claimGeneration:1,authorized:false,taskPass:false,
  releasePass:false,replayFixture:false});
const denied=(input,reason)=>({...base(input),state:'NOT_ADMITTED',reason,claimAccepted:false,
  admitted:false,agentLaunch:false,candidateTree:null,candidateCommit:null,process:null});
const joinDenied=(input,reason)=>({...joinBase(input),state:'NOT_ADMITTED',reason,
  claimAccepted:false,admitted:false,agentLaunch:false,joinedAt:null,candidates:[]});
const admitted=(input,state,reason,agentLaunch,more={})=>({...base(input),state,reason,
  claimAccepted:true,admitted:true,agentLaunch,candidateTree:null,candidateCommit:null,process:null,...more});
function validClaimRow(row,claim,issueNumber,actorRole,env){
  const v=row?.value;
  return Number.isSafeInteger(row?.revision)&&row.revision>=1&&v?.phase==='PUBLISHED_VERIFIED'&&
    v.generation===1&&v.protectedClaimKey===claimKeyFor(issueNumber,actorRole)&&
    v.repository===OWNER+'/'+REPO&&v.issueRef===issueRefFor(issueNumber)&&
    v.sourceSha===claim.sourceSha&&v.actorRole===actorRole&&
    v.dispatchId===claim.dispatchId&&v.operatorId===claim.operatorId&&
    v.operatorKind===claim.operatorKind&&v.writerId===claim.writerId&&
    Number.isSafeInteger(v.commentId)&&v.commentId>0&&v.commentId===claim.commentId&&
    v.commentRef===v.issueRef+'#issuecomment-'+v.commentId;
}
// Independently verify one accepted Claim: durable CAS row plus the current
// native publication witness. Shared by Builder attempts and Reviewer Join.
async function verifyAcceptedClaim(env,issueNumber,actorRole){
  const binding=TASK_BINDINGS[issueNumber];
  const claim=claimEnv(env,issueNumber);
  if(binding?.actorRole!==actorRole||!SHA.test(claim.sourceSha??'')||
    !ID.test(claim.dispatchId??'')||!ID.test(claim.operatorId??'')||
    !KINDS.has(claim.operatorKind)||!ID.test(claim.writerId??'')||
    !LOGIN.test(claim.writerLogin??'')||!Number.isSafeInteger(claim.commentId)||
    claim.commentId<=0||typeof claim.registryPath!=='string'||!claim.registryPath)
    return {ok:false,reason:'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH'};
  let registry,row;
  try{
    registry=await openSqliteCasStore({path:claim.registryPath,busyTimeoutMs:1000});
    row=await registry.get(registryKeyFor(claimKeyFor(issueNumber,actorRole)));
  }catch{return {ok:false,reason:'CLAIM_REGISTRY_UNAVAILABLE'};}
  finally{if(registry)await registry.close();}
  if(!row)return {ok:false,reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND'};
  if(row.value?.phase!=='PUBLISHED_VERIFIED')return {ok:false,reason:'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED'};
  if(!validClaimRow(row,claim,issueNumber,actorRole,env))return {ok:false,reason:'CLAIM_REGISTRY_RESERVATION_MISMATCH'};
  const token=env.TALLY_GITHUB_READ_TOKEN;
  if(typeof token!=='string'||token.length<8)return {ok:false,reason:'LOCAL_AGENT_UNAVAILABLE'};
  try{
    const witness=await readGitHubClaimPublicationWitness({token,owner:OWNER,repo:REPO,
      issueNumber,expectedSourceSha:claim.sourceSha,commentId:row.value.commentId,
      dispatchId:row.value.dispatchId,actorRole,operatorKind:row.value.operatorKind,
      operatorId:row.value.operatorId,protectedClaimKey:claimKeyFor(issueNumber,actorRole),
      claimGeneration:1,writerLogin:claim.writerLogin});
    if(witness?.status!=='PUBLICATION_WITNESSED'||witness.claimAccepted!==false||
      witness.authorized!==false||witness.admitted!==false||witness.taskPass!==false||
      witness.releasePass!==false||witness.sourceQualified!==true||
      witness.issueRef!==issueRefFor(issueNumber)||witness.sourceSha!==claim.sourceSha||
      witness.commentId!==row.value.commentId||witness.commentRef!==row.value.commentRef||
      witness.dispatchId!==row.value.dispatchId||witness.actorRole!==actorRole||
      witness.operatorId!==row.value.operatorId||
      witness.protectedClaimKey!==claimKeyFor(issueNumber,actorRole)||
      witness.generation!==1||witness.transportActor!==claim.writerLogin)
      return {ok:false,reason:'NATIVE_CLAIM_WITNESS_NOT_VERIFIED'};
  }catch{return {ok:false,reason:'NATIVE_CLAIM_WITNESS_NOT_VERIFIED'};}
  return {ok:true,claim};
}
async function verifyCurrentSourcePair(token,issueNumber,sourceSha){
  let pair;
  try{pair=await readGitHubNativeTaskSourcePair({token,owner:OWNER,repo:REPO,
    issueNumber,expectedSourceSha:sourceSha,limit:1000});
  }catch{return {ok:false,reason:'NATIVE_SOURCE_NOT_CURRENT'};}
  if(pair?.workItem?.dependencyState==='BLOCKED')return {ok:false,reason:'NATIVE_DEPENDENCIES_BLOCKED'};
  if(pair?.status!=='SOURCE_PAIR_OBSERVED'||pair.sourceQualified!==true||
    pair.issueRef!==issueRefFor(issueNumber)||pair.sourceSha!==sourceSha||
    pair.workItem?.issueState!=='open'||pair.workItem?.workflowState!=='state:ready'||
    pair.workItem?.workItemType!=='type:task'||pair.workItem?.dependencyState!=='CLEAR')
    return {ok:false,reason:'NATIVE_SOURCE_NOT_CURRENT'};
  return {ok:true,pair};
}
async function verifyFrozenWriteSet(env,issueNumber,sourceSha){
  const claim=claimEnv(env,issueNumber);
  const candidateRelativePath=TASK_BINDINGS[issueNumber].candidateRelativePath;
  if(!claim.writeSetSha256||typeof claim.registryPath!=='string'||!claim.registryPath)
    return {ok:false,reason:'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH'};
  let registry,writeSet;
  try{
    registry=await openSqliteCasStore({path:claim.registryPath,busyTimeoutMs:1000});
    writeSet=await registry.get(writeSetKeyFor(issueNumber));
  }catch{return {ok:false,reason:'CLAIM_REGISTRY_UNAVAILABLE'};}
  finally{if(registry)await registry.close();}
  if(writeSet?.value?.phase!=='FROZEN_DAG_WRITESET'||writeSet.value.sourceSha!==sourceSha||
    writeSet.value.writeSetSha256!==claim.writeSetSha256||
    !writeSet.value.document?.includes(candidateRelativePath))
    return {ok:false,reason:'DAG_WRITESET_NOT_VERIFIED'};
  return {ok:true};
}
function attemptWorktreePath(env,attemptId){
  const safe=attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
  const root=env.TALLY_LOCAL_WORKTREE_ROOT??env.TALLY_LOCAL_REPOSITORY_PATH;
  return root+'/.tally-t06n-worktrees/'+safe;
}
export async function runLocalCodexAttempt(input){
  const env=process.env;
  const binding=TASK_BINDINGS[input?.issueNumber];
  if(input?.sourceRef!==issueRefFor(input?.issueNumber)||!binding||
    binding.actorRole!=='builder'||!SHA.test(input.sourceSha??'')||
    !ATTEMPT.test(input.attemptId??'')||input.sourceSha!==env.TALLY_CLAIM_SOURCE_SHA)
    return denied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  const issueNumber=input.issueNumber;
  const claimCheck=await verifyAcceptedClaim(env,issueNumber,'builder');
  if(!claimCheck.ok)return denied(input,claimCheck.reason);
  const writeSetCheck=await verifyFrozenWriteSet(env,issueNumber,input.sourceSha);
  if(!writeSetCheck.ok)return denied(input,writeSetCheck.reason);
  const pairCheck=await verifyCurrentSourcePair(env.TALLY_GITHUB_READ_TOKEN,issueNumber,input.sourceSha);
  if(!pairCheck.ok)return denied(input,pairCheck.reason);
  if(env.TALLY_LOCAL_REPLAY==='1')
    return {...admitted(input,'REPLAY_COMPLETE','REPLAY_FIXTURE_ONLY',false),replayFixture:true};
  try{
    return await createLocalCodexHost().runAttempt({issueRef:issueRefFor(issueNumber),
      sourceSha:input.sourceSha,attemptId:input.attemptId,
      candidateRelativePath:binding.candidateRelativePath,
      worktreePath:env.TALLY_LOCAL_WORKTREE_PATH??attemptWorktreePath(env,input.attemptId),
      hostStorePath:env.TALLY_LOCAL_HOST_STORE_PATH,
      repositoryPath:env.TALLY_LOCAL_REPOSITORY_PATH,
      cliPath:env.TALLY_CODEX_CLI_PATH,sandboxCliPath:env.TALLY_CODEX_SANDBOX_PATH,
      codexHome:env.TALLY_CODEX_HOME,builderProfile:env.TALLY_BUILDER_PROFILE,
      builderTemp:env.TALLY_BUILDER_TEMP,emptyGitConfig:env.TALLY_EMPTY_GIT_CONFIG,
      operatorHome:env.TALLY_OPERATOR_HOME,hiddenFixture:env.TALLY_HIDDEN_FIXTURE,
      builderPrincipal:env.TALLY_BUILDER_PRINCIPAL,builderSid:env.TALLY_BUILDER_SID,
      verifierSid:env.TALLY_VERIFIER_SID,sshAddPath:env.TALLY_SSH_ADD_PATH,
      gitPath:env.TALLY_GIT_PATH,powershellPath:env.TALLY_POWERSHELL_PATH,
      modelProxy:env.TALLY_MODEL_PROXY,caBundle:env.TALLY_CA_BUNDLE,
      timeoutMs:Number(env.TALLY_CODEX_TIMEOUT_MS??300000)});
  }catch{
    return admitted(input,'LOCAL_ATTEMPT_UNKNOWN','ATTEMPT_STATUS_UNKNOWN',true,
      {process:{status:'UNKNOWN',exitCode:null,pid:null,builderPrincipal:env.TALLY_BUILDER_PRINCIPAL,
        builderSid:env.TALLY_BUILDER_SID,startedAt:null,terminatedAt:null}});
  }
}
export async function joinReviewedCandidates(input){
  const env=process.env,reviewer=Number(env.TALLY_CONCURRENT_ISSUE_REVIEWER??179);
  const issueNumber=input?.issueNumber;
  if(input?.sourceRef!==issueRefFor(issueNumber)||issueNumber!==reviewer||
    TASK_BINDINGS[issueNumber]?.actorRole!=='reviewer'||
    !SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.joinId??'')||
    input.sourceSha!==env.TALLY_CLAIM_SOURCE_SHA||
    !Array.isArray(input.candidates)||input.candidates.length<2)
    return joinDenied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  const candidates=input.candidates.map(c=>({issueNumber:c?.issueNumber,
    attemptId:c?.attemptId,candidateCommit:c?.candidateCommit,candidateTree:c?.candidateTree}));
  for(const c of candidates){
    const binding=TASK_BINDINGS[c.issueNumber];
    if(!binding||binding.actorRole!=='builder'||!ATTEMPT.test(c.attemptId??'')||
      !SHA.test(c.candidateCommit??'')||!SHA.test(c.candidateTree??''))
      return joinDenied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
    c.candidateRelativePath=binding.candidateRelativePath;
  }
  if(env.TALLY_LOCAL_REPLAY==='1')
    return {...joinBase(input),state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY',
      claimAccepted:true,admitted:true,agentLaunch:false,replayFixture:true,
      joinedAt:null,candidates:[]};
  // The Reviewer Claim is verified independently of every Builder Claim.
  const reviewerCheck=await verifyAcceptedClaim(env,reviewer,'reviewer');
  if(!reviewerCheck.ok)return joinDenied(input,reviewerCheck.reason);
  const pairCheck=await verifyCurrentSourcePair(env.TALLY_GITHUB_READ_TOKEN,reviewer,input.sourceSha);
  if(!pairCheck.ok)return joinDenied(input,pairCheck.reason);
  // Self-assertion: the Reviewer identity must differ from every Builder
  // identity it reviews; a Builder cannot review its own candidate.
  for(const c of candidates){
    const builderClaim=claimEnv(env,c.issueNumber);
    if(builderClaim.operatorId===reviewerCheck.claim.operatorId||
      builderClaim.writerId===reviewerCheck.claim.writerId)
      return joinDenied(input,'REVIEWER_SELF_ASSERTION');
  }
  try{
    const joined=await createLocalCodexHost().joinReviewedCandidates({
      issueRef:issueRefFor(reviewer),sourceSha:input.sourceSha,joinId:input.joinId,
      hostStorePath:env.TALLY_LOCAL_HOST_STORE_PATH,
      repositoryPath:env.TALLY_LOCAL_REPOSITORY_PATH,
      gitPath:env.TALLY_GIT_PATH,candidates});
    if(joined.state==='NOT_ADMITTED')return joinDenied(input,joined.reason);
    return {...joinBase(input),state:joined.state,reason:joined.reason,
      claimAccepted:true,admitted:true,agentLaunch:false,replayFixture:false,
      joinedAt:joined.joinedAt,candidates:joined.candidates};
  }catch{return joinDenied(input,'ATTEMPT_STATUS_UNKNOWN');}
}
export async function publishReviewedCandidate(input){
  const env=process.env,token=env.TALLY_GITHUB_READ_TOKEN;
  const claim=await verifyAcceptedClaim(env,203,'builder');
  if(!claim.ok)return {state:'NOT_ADMITTED',reason:claim.reason,authorized:false,taskPass:false,releasePass:false};
  const pair=await verifyCurrentSourcePair(token,203,env.TALLY_CLAIM_SOURCE_SHA);
  if(!pair.ok)return {state:'NOT_ADMITTED',reason:pair.reason,authorized:false,taskPass:false,releasePass:false};
  const octokit=typeof token==='string'&&token.length>=8?
    new Octokit({auth:token,request:{timeout:15000}}):null;
  const publisher=createCandidatePublisher({openStore:openSqliteCasStore,
    publish:input=>publishCandidatePullRequestFromClient({octokit,...input})});
  return publisher.publishReviewedCandidate({issueRef:input?.sourceRef,
    sourceSha:input?.sourceSha,attemptId:input?.attemptId,
    hostStorePath:process.env.TALLY_LOCAL_HOST_STORE_PATH,
    repositoryPath:process.env.TALLY_LOCAL_REPOSITORY_PATH,
    gitPath:process.env.TALLY_GIT_PATH,emptyGitConfig:process.env.TALLY_EMPTY_GIT_CONFIG,
    githubToken:token});
}
// T210 — the narrow GitHub project effect lives behind the Host module; this
// provider only binds transports. The write-scoped token, when the operator
// supplies one, never enters sealed bytes and is only read from Host env.
async function publishProjectIntake(publishInput){
  const env=process.env;
  const token=env.TALLY_GITHUB_WRITE_TOKEN??env.TALLY_GITHUB_READ_TOKEN;
  if(typeof token!=='string'||token.length<8)
    throw Error('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  const data=await octokit.graphql('query($owner:String!,$name:String!){repository(owner:$owner,name:$name){id projectsV2(first:20){nodes{number title url}}}}',
    {owner:OWNER,name:REPO});
  const repositoryId=data?.repository?.id;
  if(typeof repositoryId!=='string'||!repositoryId)throw Error('PROJECT_REPOSITORY_ID_UNAVAILABLE');
  const existing=(data.repository.projectsV2?.nodes??[]).find(p=>p.title===publishInput.projectTitle);
  if(existing)return {state:'PROJECT_EXISTS',project:{number:existing.number,
    title:existing.title,url:existing.url}};
  const created=await octokit.graphql(
    'mutation($repoId:ID!,$title:String!){createProjectV2(input:{repositoryId:$repoId,title:$title}){projectV2{number title url}}}',
    {repoId:repositoryId,title:publishInput.projectTitle});
  const project=created?.createProjectV2?.projectV2;
  if(!project?.number)throw Error('PROJECT_CREATE_READBACK_MISSING');
  return {state:'PROJECT_CREATED',project:{number:project.number,
    title:project.title,url:project.url}};
}
export async function hostProjectIntake(input){
  const env=process.env;
  const intake=createProjectIntakeHost({env,openStore:openSqliteCasStore,
    verifyAcceptedClaim:(issueNumber,actorRole)=>verifyAcceptedClaim(env,issueNumber,actorRole),
    verifyFrozenWriteSet:(issueNumber,sourceSha)=>verifyFrozenWriteSet(env,issueNumber,sourceSha),
    readProjectIntake:args=>readProjectIntakeFromClient({...args,
      octokit:new Octokit({auth:env.TALLY_GITHUB_READ_TOKEN,request:{timeout:15000}})}),
    publish:publishProjectIntake});
  return intake.hostProjectIntake(input);
}
export async function freezeProductIntake(input){
  return hostFreezeProductIntake({
    openStore:openSqliteCasStore,
    registryPath:process.env.TALLY_TASK_CLAIM_210_REGISTRY_PATH,
    sourceSha:input?.sourceSha,productDigest:input?.productDigest,
    author:input?.author,reviewer:input?.reviewer,frozenAt:input?.frozenAt,
    caseRefDigest:input?.caseRefDigest});
}
// T219 — the authorized native Issue effect lives behind the Host module;
// these bindings are transports only. The write-scoped token never enters
// sealed bytes and is read from Host env at call time. The frozen candidate
// review and the Task Learning ledger live in the #219 Host CAS registry and
// are seeded by Host-owned setup (never from caller input).
function maintenanceHostFor(){
  const env=process.env;
  const readOctokit=()=>new Octokit({auth:env.TALLY_GITHUB_READ_TOKEN,
    request:{timeout:15000}});
  return createMaintenanceHost({env,openStore:openSqliteCasStore,
    verifyAcceptedClaim:(issueNumber,actorRole)=>verifyAcceptedClaim(env,issueNumber,actorRole),
    verifyFrozenWriteSet:(issueNumber,sourceSha)=>verifyFrozenWriteSet(env,issueNumber,sourceSha),
    publishIssue:publishInput=>{
      const token=env.TALLY_GITHUB_WRITE_TOKEN??env.TALLY_GITHUB_READ_TOKEN;
      if(typeof token!=='string'||token.length<8)
        return Promise.reject(Error('GITHUB_PROVIDER_TOKEN_UNAVAILABLE'));
      return publishMaintenanceIssueFromClient({octokit:new Octokit({auth:token,
        request:{timeout:15000}}),...publishInput});
    },
    findIssue:args=>findMaintenanceIssueFromClient({octokit:readOctokit(),...args}),
    readIssue:args=>readMaintenanceIssueFromClient({octokit:readOctokit(),...args})});
}
export async function publishMaintenanceIssue(input){
  return maintenanceHostFor().hostPublishMaintenanceIssue(input);
}
export async function recordTaskLearningOutcome(input){
  return maintenanceHostFor().hostRecordTaskLearningOutcome(input);
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
