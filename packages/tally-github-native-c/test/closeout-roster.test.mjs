import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiRosterHost} from '../../../web/v2/ui-roster-host.mjs';
import {
  createHostCloseoutClaimReadbackPortFromClient,
  createHostCloseoutRosterPorts,
  createHostResponsibilityHandoffReaderFromClient,
  createHostVersionTaskDagFreezeWriter,
  createHostVersionTaskDagReader
} from '../src/closeout-roster.mjs';

const repository='kaicreator-mm/tally';
const owner='kaicreator-mm';
const repo='tally';
const versionId='version/v2';
const versionSha='7ebb1cdd0a79fc82cc938e35f852a80dd8e15332';
const claimSha='ad0433f8da6490e46f902cea0d68cf032f3f1b81';
const packBlobSha='fc974a8adedbe415811e02331fb62087f0fa98d6';
const packCommitSha='b3ad8ae7833028999a9c27cac7042037bca6d961';
const taskPackRef='https://github.com/'+repository+'/blob/'+packCommitSha+
  '/apps/tally-web/src/features/dag/version-v2-task-pack.candidate.json';
const issueUrl=n=>'https://github.com/'+repository+'/issues/'+n;
const commentUrl=(n,id)=>issueUrl(n)+'#issuecomment-'+id;
const protectedKey=n=>repository+'#'+n+':builder:__default__';

function memoryStore(initial){
  const rows=new Map(initial?[[initial.key,{revision:1,value:structuredClone(initial.value)}]]:[]);
  return {
    async get(key){const row=rows.get(key);return row?structuredClone(row):null;},
    async put(key,value,{mustBeAbsent}={}){
      if(mustBeAbsent&&rows.has(key))throw new Error('CAS_EXISTS');
      rows.set(key,{revision:1,value:structuredClone(value)});
      return {revision:1};
    },
    mutate(key,fn){const row=rows.get(key);rows.set(key,{revision:row.revision+1,value:fn(structuredClone(row.value))});},
    count:()=>rows.size
  };
}
const absentSubject=n=>({
  issueNumber:n,sourceSha:versionSha,actorRole:'builder',
  operatorKind:'other',operatorId:'unassigned:tally-v2',dispatchId:null,
  generation:1,protectedClaimKey:protectedKey(n),commentId:null,commentRef:null,
  candidateStatus:'NO_CLAIM_EVENT_EXPECTED'
});
const observedSubject={
  issueNumber:260,sourceSha:claimSha,actorRole:'builder',operatorKind:'other',
  operatorId:'zcode:i259:closeout:1',
  dispatchId:'dispatch:260:closeout-a:host:20261102:1',generation:1,
  protectedClaimKey:protectedKey(260),commentId:6568304709,
  commentRef:commentUrl(260,6568304709),
  candidateStatus:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM'
};
function validPack(){
  return {
    schema:'tally/version-task-pack/v1',status:'CANDIDATE_NOT_FROZEN',
    repository,versionId,versionBaseSha:versionSha,
    observedAt:'2026-11-02T03:00:00Z',
    source:{kind:'GITHUB_NATIVE_ISSUES',query:'state=all&labels=type:task',
      taskIssueCount:4,claimCommentPages:1},
    taskIssues:[211,212,260,264].map(n=>({
      issueNumber:n,title:'Task '+n,issueState:n===264?'open':'closed',
      labels:['type:task'],updatedAt:'2026-11-02T02:59:00Z'
    })),
    edges:[
      {from:260,to:264,kind:'NATIVE',evidenceRef:issueUrl(264),dependencyRef:issueUrl(260)},
      {from:211,to:212,kind:'PLANNED',evidenceRef:issueUrl(212),
        dependencyRef:issueUrl(211)}
    ],
    claimSubjects:[absentSubject(211),absentSubject(212),observedSubject,absentSubject(264)]
  };
}
const review=()=>({
  status:'INDEPENDENTLY_VERIFIED',taskPackBlobSha:packBlobSha,
  sourceSha:versionSha,reviewedHeadSha:versionSha,
  reviewer:'independent-reviewer',author:'web-builder',
  reviewRef:'https://github.com/'+repository+'/pull/258#pullrequestreview-1',
  verifiedAt:'2026-11-02T11:00:00.000Z'
});
const approval=()=>({
  status:'OWNER_APPROVED',taskPackBlobSha:packBlobSha,sourceSha:versionSha,
  owner,approvalRef:'https://github.com/'+repository+'/issues/259#issuecomment-6574693761',
  approvedAt:'2026-11-02T11:01:00.000Z'
});
function nativeProjection({pack=validPack(),tasks,connected=true,
  versionId:versionValue=versionId,source:sourceValue,dag:dagValue}={}){
  const roster=tasks??pack.taskIssues;
  return {
    connected,source:sourceValue??{repository,sourceSha:versionSha,
      currentness:'CURRENT',readback:'VERIFIED'},
    versionId:versionValue,tasks:roster.map(t=>({id:String(t.issueNumber),
      versionId:versionValue,
      title:t.title,issueState:t.issueState,updatedAt:t.updatedAt})),
    dag:dagValue??{connected:true,source:{repository,sourceSha:versionSha,
      currentness:'CURRENT',readback:'VERIFIED'},versionId:versionValue,
      asOf:{sourceSha:versionSha},
      nodes:roster.map(t=>({taskId:String(t.issueNumber),versionId:versionValue})),
      edges:pack.edges.filter(e=>e.kind==='NATIVE').map(e=>({...e,
        from:String(e.from),to:String(e.to),
        dependencyRef:issueUrl(e.from)}))}
  };
}
function freezeHarness({pack=validPack(),reviewValue=review(),approvalValue=approval(),
  nativeRosterReader:nativeRosterReaderOverride,store=memoryStore()}={}){
  const content=JSON.stringify(pack);
  const taskPackReader=async()=>({status:'VERIFIED',blobSha:packBlobSha,content});
  const nativeRosterReader=nativeRosterReaderOverride??(async()=>nativeProjection({pack}));
  const reviewWitnessReader=async()=>reviewValue;
  const ownerApprovalReader=async()=>approvalValue;
  const writer=createHostVersionTaskDagFreezeWriter({
    repository,versionId,expectedVersionSha:versionSha,store,
    taskPackReader,nativeRosterReader,reviewWitnessReader,ownerApprovalReader,
    now:()=> '2026-11-02T12:00:00.000Z'
  });
  return {store,writer,taskPackReader,nativeRosterReader,
    reviewWitnessReader,ownerApprovalReader};
}
function dagReader(h,{nativeRosterReader,store}={}){
  return createHostVersionTaskDagReader({
    repository,versionId,expectedVersionSha:versionSha,store:store??h.store,
    taskPackReader:h.taskPackReader,
    nativeRosterReader:nativeRosterReader??h.nativeRosterReader,
    reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader
  });
}

test('version Task DAG freeze and readback bind complete roster, native/planned edges and exact witnesses',async()=>{
  const h=freezeHarness();
  const frozen=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  assert.equal(frozen.taskCount,4);
  assert.equal(frozen.edgeCount,2);
  assert.equal(frozen.authorized,false);
  const reader=dagReader(h);
  const result=await reader.readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(result.connected,true);
  assert.deepEqual(result.frozen.taskIssues,[211,212,260,264]);
  assert.equal(result.frozen.edges.find(x=>x.kind==='PLANNED').from,211);
  assert.equal(result.frozenAt,'2026-11-02T12:00:00.000Z');
  assert.equal(result.witness.reviewedHeadSha,versionSha);
  assert.equal(result.ownerApproval.owner,owner);
});

test('freeze rejects a reviewer who did not review the exact version SHA',async()=>{
  const h=freezeHarness({reviewValue:{...review(),reviewedHeadSha:'a'.repeat(40)}});
  const result=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.deepEqual(result,{status:'BLOCKED',reason:'INDEPENDENT_REVIEW_REQUIRED'});
});

test('freeze rejects wrong Owner, wrong version base, mutable ref, cycle and duplicate edge',async()=>{
  const wrongOwner=freezeHarness({approvalValue:{...approval(),owner:'someone-else'}});
  assert.equal((await wrongOwner.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'OWNER_APPROVAL_REQUIRED');
  const wrongBasePack=validPack();wrongBasePack.versionBaseSha='a'.repeat(40);
  const wrongBase=freezeHarness({pack:wrongBasePack});
  assert.equal((await wrongBase.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'TASK_PACK_CONTENT_INVALID');
  const mutable=freezeHarness();
  assert.equal((await mutable.writer.freezeVersionTaskDag({
    taskPackRef:'https://github.com/'+repository+'/blob/version/v2/file.json',
    taskPackBlobSha:packBlobSha})).reason,'TASK_PACK_IDENTITY_INVALID');
  const cyclic=validPack();cyclic.edges.push({from:212,to:211,kind:'PLANNED',evidenceRef:issueUrl(211)});
  assert.equal((await freezeHarness({pack:cyclic}).writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'TASK_PACK_CONTENT_INVALID');
  const duplicate=validPack();duplicate.edges.push({...duplicate.edges[0]});
  assert.equal((await freezeHarness({pack:duplicate}).writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'TASK_PACK_CONTENT_INVALID');
});

test('freeze and readback accept exact-SHA independent review when reviewer and PR author share a GitHub login',async()=>{
  const sameLogin=freezeHarness({reviewValue:{...review(),
    reviewer:'kaicreator-mm',author:'kaicreator-mm'}});
  const frozen=await sameLogin.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  const result=await dagReader(sameLogin).readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(result.connected,true);
  assert.equal(result.witness.reviewer,'kaicreator-mm');
  assert.equal(result.witness.author,'kaicreator-mm');
});

test('selected UI roster accepts a same-login ReviewWitness through freeze writer and frozen Host readback',async()=>{
  const h=freezeHarness({reviewValue:{...review(),reviewer:owner,author:owner}});
  const frozen=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  const frozenHost=dagReader(h);
  const ui=createUiRosterHost({repository,versionId,expectedSha:versionSha,
    frozenHost,nativeC:{version:async()=>nativeProjection({pack:validPack()})}});
  const roster=await ui.roster({projectId:repository,versionId});
  assert.equal(roster.connected,true);
  assert.equal(roster.rosterCoverage,'VERIFIED_COMPLETE');
  assert.equal(roster.frozenBlobSha,packBlobSha);
  assert.equal(roster.taskCount,4);
});

test('freeze replay requires the same immutable Task Pack and witness identity',async()=>{
  const h=freezeHarness();
  assert.equal((await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha})).status,'FROZEN');
  assert.equal((await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha})).status,'REPLAY');
  const differentReview=freezeHarness({store:h.store,reviewValue:{...review(),reviewer:'other-reviewer'}});
  assert.equal((await differentReview.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'DIVERGENT_FREEZE_EXISTS');
});

test('frozen DAG reader refuses a changed durable revision during read',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  let reads=0;
  const unstableStore={
    async get(key){
      const value=await h.store.get(key);
      reads++;
      if(reads===2)return {...value,revision:value.revision+1};
      return value;
    }
  };
  const reader=dagReader(h,{store:unstableStore});
  assert.equal((await reader.readFrozenDag()).reason,'FROZEN_DAG_CHANGED_DURING_READ');
});

test('post-freeze Task progress never invalidates immutable DAG structure',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const progressed=validPack();
  progressed.taskIssues[0].issueState='open'; // after freeze a Task may reopen
  progressed.taskIssues[0].updatedAt='2026-11-03T16:04:56Z';
  progressed.taskIssues[1].updatedAt='2026-11-04T10:00:00Z'; // comment-only update
  const reader=dagReader(h,{nativeRosterReader:async()=>nativeProjection({pack:progressed})});
  const result=await reader.readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(result.connected,true);
  assert.deepEqual(result.frozen.taskIssues,[211,212,260,264]);
  assert.deepEqual(result.frozen.edges,validPack().edges);
  assert.equal(result.frozenAt,'2026-11-02T12:00:00.000Z');
});

test('post-freeze structural tampering and invalid live status still fail closed',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  for(const mutant of [
    Object.assign(validPack(),{taskIssues:validPack().taskIssues.map((t,i)=>
      i===1?{...t,title:'Task 212 renamed'}:t)}),
    Object.assign(validPack(),{taskIssues:validPack().taskIssues.map((t,i)=>
      i===1?{...t,issueState:'PASS'}:t)}),
    Object.assign(validPack(),{taskIssues:validPack().taskIssues.map((t,i)=>
      i===1?{...t,updatedAt:'invalid-timestamp'}:t)}),
  ]){
    const result=await dagReader(h,{nativeRosterReader:async()=>
      nativeProjection({pack:mutant})})
      .readFrozenDag({repository,versionId,sourceSha:versionSha});
    assert.equal(result.connected,false);
    assert.equal(result.reason,'FROZEN_DAG_NATIVE_ROSTER_STALE');
  }
});

test('pre-freeze Task status or updatedAt drift refuses CAS freeze',async()=>{
  for(const field of ['issueState','updatedAt']){
    const pack=validPack(),mutant=validPack();
    mutant.taskIssues[0][field]=field==='issueState'?'open':'2026-11-03T16:04:56Z';
    const h=freezeHarness({pack,nativeRosterReader:async()=>nativeProjection({pack:mutant})});
    const result=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'NATIVE_TASK_ROSTER_READBACK_FAILED');
    assert.equal(h.store.count(),0);
  }
});

test('frozen DAG reader denies a native roster with a missing, extra or unqualified Task set',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const deny=async roster=>{
    const result=await dagReader(h,{nativeRosterReader:async()=>roster})
      .readFrozenDag({repository,versionId,sourceSha:versionSha});
    assert.equal(result.connected,false);
    return result.reason;
  };
  assert.equal(await deny(nativeProjection({tasks:validPack().taskIssues.slice(1)})),
    'FROZEN_DAG_NATIVE_ROSTER_STALE');
  const extra=validPack();
  extra.taskIssues.push({issueNumber:300,title:'Task 300',issueState:'open',
    labels:['type:task'],updatedAt:'2026-11-02T02:59:00Z'});
  assert.equal(await deny(nativeProjection({tasks:extra.taskIssues})),
    'FROZEN_DAG_NATIVE_ROSTER_STALE');
  assert.equal(await deny(nativeProjection({connected:false})),
    'FROZEN_DAG_NATIVE_ROSTER_UNQUALIFIED');
  assert.equal(await deny(nativeProjection({source:{repository,sourceSha:'a'.repeat(40),
    currentness:'CURRENT',readback:'VERIFIED'}})),
    'FROZEN_DAG_NATIVE_ROSTER_UNQUALIFIED');
  assert.equal(await deny(nativeProjection({versionId:'version/other'})),
    'FROZEN_DAG_NATIVE_ROSTER_UNQUALIFIED');
});

test('freeze blocks when a native Issue changes between the roster snapshot and the CAS write',async()=>{
  let nativeReads=0;
  const drifted=validPack();
  drifted.taskIssues[0].updatedAt='2026-11-03T16:04:56Z';
  const h=freezeHarness({nativeRosterReader:async()=>{
    nativeReads++;
    return nativeProjection({pack:nativeReads===1?validPack():drifted});
  }});
  const result=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.deepEqual(result,{status:'BLOCKED',reason:'NATIVE_TASK_ROSTER_CHANGED_BEFORE_FREEZE'});
  assert.equal(nativeReads,2);
  assert.equal(h.store.count(),0);
});

test('unchanged exact native roster stays readable and freeze-replayable',async()=>{
  const h=freezeHarness();
  assert.equal((await h.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).status,'FROZEN');
  const first=await dagReader(h).readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(first.connected,true);
  const second=await dagReader(h).readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(second.connected,true);
  assert.equal((await h.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).status,'REPLAY');
});

test('closeout roster ports bind the shared native roster source into the frozen DAG reader',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const unreachableOctokit={rest:{repos:{
    getBranch:async()=>{throw new Error('GITHUB_UNAVAILABLE');},
    getCommit:async()=>{throw new Error('GITHUB_UNAVAILABLE');}
  },issues:{
    getComment:async()=>{throw new Error('GITHUB_UNAVAILABLE');}
  }}};
  const ports=createHostCloseoutRosterPorts({repository,versionId,
    expectedVersionSha:versionSha,store:h.store,octokit:unreachableOctokit,
    owner,repo,writerLogin:owner,writerId:'tally-host:win32:i259:closeout:1',
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader});
  const frozen=await ports.frozenHost.readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(frozen.connected,false);
  assert.equal(frozen.reason,'FROZEN_DAG_NATIVE_ROSTER_UNQUALIFIED');
  assert.equal((await ports.freezeWriter.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'NATIVE_TASK_ROSTER_READBACK_FAILED');
});

test('frozen DAG reader requires the trusted native roster reader port',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.throws(()=>createHostVersionTaskDagReader({
    repository,versionId,expectedVersionSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader
  }),/TRUSTED_DAG_READER_REQUIRED/);
});

function eventBody(){
  const fence=String.fromCharCode(96).repeat(3);
  return [
    '<!-- ads:github -->',fence+'yaml',
    'schema: ads/github/v3','kind: event','data:',
    '  schema: ai-dev/event-v2','  event: DISPATCH_CLAIMED',
    '  actor_role: builder','  operator_kind: other',
    '  operator_id: zcode:i259:closeout:1','  task: "#260"',
    '  issue: "#260"','  sha: '+claimSha,
    '  dispatch_id: dispatch:260:closeout-a:host:20261102:1',
    '  dispatch_state: CLAIMED',
    '  protected_claim_key: '+protectedKey(260),
    '  admission_generation: 1',fence,''
  ].join('\n');
}
function claimComment({body=eventBody(),updatedAt='2026-11-02T03:25:38Z'}={}){
  return {
    id:6568304709,html_url:commentUrl(260,6568304709),
    issue_url:'https://api.github.com/repos/'+repository+'/issues/260',
    user:{login:owner},body,created_at:'2026-11-02T03:25:38Z',
    updated_at:updatedAt
  };
}
function octokitFor(comment){
  const issue={id:100,number:260,html_url:issueUrl(260),updated_at:'2026-11-02T03:00:00Z'};
  const listComments=async()=>({data:comment?[comment]:[]});
  return {
    rest:{
      issues:{
        get:async()=>({data:issue}),listComments,
        getComment:async()=>({data:comment})
      },
      repos:{getCommit:async({ref})=>({data:{sha:ref}})}
    },
    paginate:{iterator:async function* (method,params){yield await method(params);}}
  };
}
function claimArgs(){
  return {
    repository,versionId,versionSourceSha:versionSha,issueNumber:260,
    claimSourceSha:claimSha,taskPackRef,taskPackBlobSha:packBlobSha,
    claimSubject:observedSubject
  };
}
function claimRow(){
  return {
    phase:'PUBLISHED_VERIFIED',repository,issueRef:issueUrl(260),sourceSha:claimSha,
    actorRole:'builder',operatorKind:'other',operatorId:'zcode:i259:closeout:1',
    dispatchId:'dispatch:260:closeout-a:host:20261102:1',generation:1,
    protectedClaimKey:protectedKey(260),writerId:'tally-host:win32:i259:closeout:1',
    commentId:6568304709,commentRef:commentUrl(260,6568304709)
  };
}
function claimPort({comment=claimComment(),row=claimRow()}={}){
  const store={get:async()=>row?{revision:1,value:row}:null};
  return createHostCloseoutClaimReadbackPortFromClient({
    octokit:octokitFor(comment),owner,repo,writerLogin:owner,
    writerId:'tally-host:win32:i259:closeout:1',store,
    repository,versionId,expectedVersionSha:versionSha
  });
}

test('accepted Claim requires the durable CAS row and exact native comment witness, but grants no execution',async()=>{
  const result=await claimPort().readAcceptedClaim(claimArgs());
  assert.equal(result.claimStatus,'ACCEPTED');
  assert.equal(result.claimAccepted,true);
  assert.equal(result.responsibilityStatus,'NOT_VERIFIED');
  assert.equal(result.authorized,false);
  assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);
  assert.equal(result.releasePass,false);
  assert.match(result.commentBodySha256,/^[a-f0-9]{64}$/);
});

test('comment-only DISPATCH_CLAIMED is a candidate, never an accepted Claim',async()=>{
  const port=claimPort({row:null});
  const result=await port.readAcceptedClaim(claimArgs());
  assert.equal(result.claimStatus,'NOT_VERIFIED');
  assert.equal(result.claimAccepted,false);
  assert.equal(result.claimPublication,'CANDIDATE_ONLY');
  assert.equal(result.reason,'NATIVE_EVENT_WITHOUT_CAS_ROW');
});

test('verified absence is distinct from a candidate and invalid input fails closed',async()=>{
  const args={...claimArgs(),claimSubject:{...absentSubject(260),sourceSha:claimSha}};
  const port=claimPort({comment:null,row:null});
  const absent=await port.readAcceptedClaim(args);
  assert.equal(absent.claimStatus,'NOT_CLAIMED');
  assert.equal(absent.claimAccepted,false);
  await assert.doesNotReject(()=>port.readAcceptedClaim({...claimArgs(),taskPackRef:null}));
  const invalid=await port.readAcceptedClaim({...claimArgs(),taskPackRef:null});
  assert.equal(invalid.reason,'CLAIM_READBACK_INPUT_INVALID');
});

test('CAS writer identity mismatch or edited native comment never qualifies',async()=>{
  const wrongRow={...claimRow(),writerId:'other-host'};
  const mismatch=await claimPort({row:wrongRow}).readAcceptedClaim(claimArgs());
  assert.equal(mismatch.claimStatus,'NOT_VERIFIED');
  assert.equal(mismatch.reason,'CAS_ROW_NOT_ACCEPTED_OR_MISMATCHED');
  const edited=await claimPort({comment:claimComment({updatedAt:'2026-11-02T03:30:00Z'})})
    .readAcceptedClaim(claimArgs());
  assert.equal(edited.claimStatus,'NOT_VERIFIED');
  assert.equal(edited.claimAccepted,false);
});

function githubComment(id,issueNumber,body,login=owner){
  const ref=issueUrl(issueNumber);
  return {id,html_url:ref+'#issuecomment-'+id,
    issue_url:'https://api.github.com/repos/'+repository+'/issues/'+issueNumber,
    user:{login},body,created_at:'2026-11-02T03:00:00Z',updated_at:'2026-11-02T03:00:00Z'};
}
function yamlComment(id,issueNumber,fields,login=owner){
  const fence=String.fromCharCode(96).repeat(3);
  return githubComment(id,issueNumber,'record\n'+fence+'yaml\n'+fields+'\n'+fence+'\n',login);
}
test('Owner authority, frozen write-set scope, original Claim and Web handoff receipt are all re-read',async()=>{
  const dispatchId='dispatch:260:closeout-a:host:20261102:1';
  const authorityId=6574693761,scopeId=6572687940,handoffId=6574702614,receiptId=6576146806;
  const handoffFields=[
    'schema: tally/responsibility-handoff/v1',
    'responsibility_mode: RESPONSIBILITY_HANDOFF',
    'repository: '+repository,'task: "#260"','issue: "#260"',
    'source_sha: '+claimSha,'parent_dispatch_ref: "#260@6568304709"',
    'parent_dispatch_id: '+dispatchId,'from_operator_id: zcode:i259:closeout:1',
    'to_operator_id: chatgpt-web:tally-v2','to_operator_kind: chatgpt-web',
    'to_actor_role: builder','execution_environment: WEB','scheduler_origin: WEB',
    'handoff_dispatch_id: dispatch:260:closeout-a:web-handoff:20261102:1',
    'protected_claim_key: '+protectedKey(260),'admission_generation: 1',
    'authority_ref: "#259@6574693761"','scope_ref: "#260@6572687940"'
  ].join('\n');
  const ackFields=[
    'schema: tally/responsibility-handoff-ack/v1',
    'responsibility_mode: RESPONSIBILITY_HANDOFF_RECEIVED',
    'repository: '+repository,'task: "#260"','source_sha: '+claimSha,
    'parent_dispatch_ref: "#260@6568304709"','parent_dispatch_id: '+dispatchId,
    'handoff_ref: "#260@6574702614"',
    'handoff_dispatch_id: dispatch:260:closeout-a:web-handoff:20261102:1',
    'protected_claim_key: '+protectedKey(260),'admission_generation: 1',
    'authority_ref: "#259@6574693761"','scope_ref: "#260@6572687940"',
    'receiver:','  operator_id: chatgpt-web:tally-v2',
    '  operator_kind: chatgpt-web','  actor_role: builder',
    '  execution_environment: WEB','  scheduler_origin: WEB',
    'transport_actor: '+owner,'received: true'
  ].join('\n');
  const comments=new Map([
    [authorityId,githubComment(authorityId,259,
      'Owner-authorized responsibility handoff. Receiving Builder chatgpt-web:tally-v2. '+
      dispatchId+' 6568304709.')],
    [scopeId,githubComment(scopeId,260,
      'The Frozen write sets are recorded: A=7, B=10, C=12, D=7.')],
    [handoffId,yamlComment(handoffId,260,handoffFields)],
    [receiptId,yamlComment(receiptId,260,ackFields)]
  ]);
  const octokit={rest:{issues:{getComment:async({comment_id})=>({data:comments.get(comment_id)})}}};
  const reader=createHostResponsibilityHandoffReaderFromClient({
    octokit,owner,repo,writerLogin:owner,
    handoffByIssue:{'260':{handoffCommentId:handoffId,receiptCommentId:receiptId}}
  });
  const result=await reader.readResponsibilityHandoff({
    issueNumber:260,claimSubject:observedSubject
  });
  assert.equal(result.status,'VERIFIED');
  assert.equal(result.activeOperator.operatorId,'chatgpt-web:tally-v2');
});


test('freeze refuses a native roster omission or native dependency drift',async()=>{
  const missing=freezeHarness({nativeRosterReader:async()=>{
    const pack=validPack();
    return {
      connected:true,source:{repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'},
      versionId,tasks:pack.taskIssues.slice(1).map(t=>({id:String(t.issueNumber),
        versionId,title:t.title,issueState:t.issueState,updatedAt:t.updatedAt})),
      dag:{connected:true,source:{repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'},
        versionId,asOf:{sourceSha:versionSha},
        nodes:pack.taskIssues.slice(1).map(t=>({taskId:String(t.issueNumber),versionId})),
        edges:pack.edges.filter(e=>e.kind==='NATIVE').map(e=>({...e,
          from:String(e.from),to:String(e.to),dependencyRef:issueUrl(e.from)}))}
    };
  }});
  assert.equal((await missing.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'NATIVE_TASK_ROSTER_READBACK_FAILED');
  const changed=freezeHarness({nativeRosterReader:async()=>{
    const pack=validPack();
    return {
      connected:true,source:{repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'},
      versionId,tasks:pack.taskIssues.map(t=>({id:String(t.issueNumber),versionId,
        title:t.title,issueState:t.issueState,updatedAt:t.updatedAt})),
      dag:{connected:true,source:{repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'},
        versionId,asOf:{sourceSha:versionSha},
        nodes:pack.taskIssues.map(t=>({taskId:String(t.issueNumber),versionId})),
        edges:[]}
    };
  }});
  assert.equal((await changed.writer.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha})).reason,'NATIVE_TASK_ROSTER_READBACK_FAILED');
});

test('freeze writer -> reader -> UI roster preserves the immutable Task Pack reference',async()=>{
  const h=freezeHarness();
  const frozen=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  const frozenHost=dagReader(h);
  const ui=createUiRosterHost({repository,versionId,expectedSha:versionSha,
    frozenHost,nativeC:{version:args=>h.nativeRosterReader(args)}});
  const result=await ui.roster({projectId:repository,versionId});
  assert.equal(result.connected,true);
  assert.equal(result.taskPackRef,taskPackRef);
  assert.equal(result.frozenBlobSha,packBlobSha);
  assert.equal(result.taskCount,4);
  assert.equal(result.claimReadbackCoverage,'NOT_VERIFIED');
});

test('canonical unedited Web Builder handoff replacement v2 is read back as VERIFIED',async()=>{
  const dispatchId='dispatch:260:closeout-a:host:20261102:1';
  const authorityId=6574693761,scopeId=6572687940;
  const handoffId=6579759879,receiptId=6579767823;
  const handoffFields=[
    'schema: tally/responsibility-handoff/v1',
    'responsibility_mode: RESPONSIBILITY_HANDOFF',
    'repository: '+repository,'task: "#260"','issue: "#260"',
    'source_sha: '+claimSha,'parent_dispatch_ref: "#260@6568304709"',
    'parent_dispatch_id: '+dispatchId,'from_operator_id: zcode:i259:closeout:1',
    'to_operator_id: chatgpt-web:tally-v2','to_operator_kind: chatgpt-web',
    'to_actor_role: builder','execution_environment: WEB','scheduler_origin: WEB',
    'handoff_dispatch_id: dispatch:260:closeout-a:web-handoff:20261102:2',
    'protected_claim_key: '+protectedKey(260),'admission_generation: 1',
    'authority_ref: "#259@6574693761"','scope_ref: "#260@6572687940"'
  ].join('\n');
  const ackFields=[
    'schema: tally/responsibility-handoff-ack/v1',
    'responsibility_mode: RESPONSIBILITY_HANDOFF_RECEIVED',
    'repository: '+repository,'task: "#260"','issue: "#260"',
    'source_sha: '+claimSha,'parent_dispatch_ref: "#260@6568304709"',
    'parent_dispatch_id: '+dispatchId,
    'handoff_ref: "#260@6579759879"',
    'handoff_dispatch_id: dispatch:260:closeout-a:web-handoff:20261102:2',
    'protected_claim_key: '+protectedKey(260),'admission_generation: 1',
    'authority_ref: "#259@6574693761"','scope_ref: "#260@6572687940"',
    'receiver:','  operator_id: chatgpt-web:tally-v2',
    '  operator_kind: chatgpt-web','  actor_role: builder',
    '  execution_environment: WEB','  scheduler_origin: WEB',
    'transport_actor: '+owner,'received: true'
  ].join('\n');
  const comments=new Map([
    [authorityId,githubComment(authorityId,259,
      'Owner-authorized responsibility handoff. Receiving Builder chatgpt-web:tally-v2. '+
      dispatchId+' 6568304709.')],
    [scopeId,githubComment(scopeId,260,
      'The Frozen write sets are recorded: A=7, B=10, C=12, D=7.')],
    [handoffId,yamlComment(handoffId,260,handoffFields)],
    [receiptId,yamlComment(receiptId,260,ackFields)]
  ]);
  const octokit={rest:{issues:{getComment:async({comment_id})=>({data:comments.get(comment_id)})}}};
  const reader=createHostResponsibilityHandoffReaderFromClient({
    octokit,owner,repo,writerLogin:owner,
    handoffByIssue:{'260':{handoffCommentId:handoffId,receiptCommentId:receiptId}}
  });
  const result=await reader.readResponsibilityHandoff({
    issueNumber:260,claimSubject:observedSubject
  });
  assert.equal(result.status,'VERIFIED');
  assert.equal(result.activeOperator.operatorId,'chatgpt-web:tally-v2');
  assert.equal(result.handoffRef,commentUrl(260,handoffId));
  assert.equal(result.receiptRef,commentUrl(260,receiptId));
});

// ---------------------------------------------------------------------------
// A2 dual-Epoch successor read contract (#392/6985978779): the original
// S0-only reader/writer semantics above are historical and unchanged; the
// wrapper proves BOTH pins with explicit FROZEN_HISTORICAL_VERIFIED versus
// CURRENT_STRUCTURAL_COMPATIBLE statuses and zero CAS writes.
// ---------------------------------------------------------------------------
import {createHash} from 'node:crypto';
import {
  createHostVersionTaskDagDualEpochReader,
  versionDagIdentityDigests
} from '../src/closeout-roster.mjs';

const s1Sha='1'.repeat(40),s2Sha='2'.repeat(40);
function currentNativeProjection({pack=validPack(),sourceSha,progressed=false}={}){
  const roster=progressed?pack.taskIssues.map((t,i)=>i===0
    ?{...t,updatedAt:'2026-11-19T10:00:00Z'}:t):pack.taskIssues;
  const base=nativeProjection({pack:{...pack,taskIssues:roster}});
  return {...base,source:{repository,sourceSha,currentness:'CURRENT',readback:'VERIFIED'},
    dag:{...base.dag,source:{repository,sourceSha,currentness:'CURRENT',readback:'VERIFIED'},
      asOf:{sourceSha}}};
}
function dualEpochHarness({pack=validPack(),store=memoryStore(),
  headShas=[versionSha],ancestryResult={connected:true,repository,baseSha:versionSha,
    headSha:s1Sha,ancestry:'BASE_IS_ANCESTOR'},
  currentRoster=null,nowValue='2026-11-19T10:00:00.000Z'}={}){
  const h=freezeHarness({pack,store});
  let headReads=0;
  const branchHeadReader=async()=>{
    const sha=headShas[Math.min(headReads++,headShas.length-1)];
    return {connected:true,repository,versionId,headSha:sha};
  };
  const reader=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader,branchHeadReader,ancestryReader:
      async()=>ancestryResult,
    currentRosterReader:currentRoster??(async({sourceSha})=>
      currentNativeProjection({pack,sourceSha})),
    now:()=>nowValue
  });
  return {...h,reader};
}
function successorWitness({pack=validPack(),observedVersionSha=s1Sha,
  extraKeys={},omit=[]}={}){
  const digests=versionDagIdentityDigests({taskIssues:pack.taskIssues,edges:pack.edges});
  const base={
    schema:'tally/v2/source-successor-compatibility-witness/v1',
    repository,versionId,freezeBaseSha:versionSha,observedVersionSha,
    frozenPackBlobSha:packBlobSha,
    versionDagKey:'tally:v2:version-task-dag:'+createHash('sha256').update(repository+'#'+versionId).digest('hex'),
    original51IdsDigest:digests.original51IdsDigest,
    native29EdgeDigest:digests.native29EdgeDigest,
    changedCodeBlobs:[{path:'packages/tally-github-native-c/src/closeout-roster.mjs',
      beforeBlob:'3'.repeat(40),afterBlob:'4'.repeat(40)}],
    affectedAbiDigest:'5'.repeat(64),
    reviewRef:'https://github.com/'+repository+'/pull/900#pullrequestreview-1',
    reviewedHead:observedVersionSha,
    scopeDecisionRef:'https://github.com/'+repository+'/issues/392#issuecomment-6986858229',
    ciRefs:['https://ci.kaicreator.com/repos/12/pipeline/900/1'],
    issuedAt:'2026-11-19T09:00:00.000Z',expiresAt:'2026-11-26T09:00:00.000Z',
    revoked:false,
    ...extraKeys
  };
  for(const k of omit)delete base[k];
  return base;
}

test('A2 regression: the successor reader passes a NUMBER taskIssueNumbers array to the current roster port',async()=>{
  const seen=[];
  const h=dualEpochHarness({currentRoster:async({sourceSha,taskIssueNumbers})=>{
    seen.push(taskIssueNumbers);
    return currentNativeProjection({sourceSha});
  }});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  await h.reader.readDualEpoch();
  assert.equal(seen.length,1);
  for(const arr of seen){
    assert.ok(Array.isArray(arr));
    for(const n of arr)assert.ok(Number.isSafeInteger(n)&&n>0,'number array required, got '+typeof n);
  }
});

test('POS-A2-01: S1=S0 keeps original reader semantics, historical and same-source current, zero mutation',async()=>{
  const h=dualEpochHarness();
  const frozen=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  const before=h.store.count();
  const result=await h.reader.readDualEpoch();
  assert.equal(result.connected,true);
  assert.equal(result.historical.status,'FROZEN_HISTORICAL_VERIFIED');
  assert.equal(result.historical.source.currentness,'FROZEN_HISTORICAL');
  assert.equal(result.current.status,'CURRENT_SAME_SOURCE');
  assert.equal(result.current.qualified,true);
  assert.equal(result.current.currentness,'CURRENT');
  assert.equal(result.epoch.ancestry,'S1_EQUALS_S0');
  assert.equal(result.compatibilityWitness.present,false);
  assert.equal(result.authorized,false);assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.equal(h.store.count(),before);
  // the original S0-only reader keeps its historical semantics unchanged
  const legacy=await dagReader(h).readFrozenDag({repository,versionId,sourceSha:versionSha});
  assert.equal(legacy.connected,true);
  assert.equal(legacy.source.currentness,'CURRENT');
});

test('POS-A2-02: S1 descendant without a qualified witness returns historical=true/current=false',async()=>{
  const h=dualEpochHarness({headShas:[s1Sha,s1Sha],
    currentRoster:async({sourceSha})=>currentNativeProjection({sourceSha,progressed:true})});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const result=await h.reader.readDualEpoch();
  assert.equal(result.connected,true);
  assert.equal(result.historical.status,'FROZEN_HISTORICAL_VERIFIED');
  assert.equal(result.historical.historical,true);
  assert.equal(result.current.qualified,false);
  assert.equal(result.current.currentness,'STALE');
  assert.equal(result.current.reason,'CURRENT_SOURCE_COMPATIBILITY_UNWITNESSED');
  assert.equal(result.current.observedVersionSha,s1Sha);
  assert.equal(result.current.taskClaims,'UNVERIFIED');
  assert.match(result.current.original51IdsDigest,/^[a-f0-9]{64}$/);
  assert.equal(result.authorized,false);assert.equal(result.taskPass,false);
  assert.equal(h.store.count(),1);
});

test('POS-A2-03: exact authenticated non-expired witness yields CURRENT_STRUCTURAL_COMPATIBLE only for this S1',async()=>{
  const h=dualEpochHarness({headShas:[s1Sha,s1Sha],
    currentRoster:async({sourceSha})=>currentNativeProjection({sourceSha,progressed:true})});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const witness=successorWitness();
  const result=await h.reader.readDualEpoch({compatibilityWitness:witness});
  assert.equal(result.connected,true);
  assert.equal(result.current.status,'CURRENT_STRUCTURAL_COMPATIBLE');
  assert.equal(result.current.qualified,true);
  assert.equal(result.current.currentness,'CURRENT');
  assert.equal(result.current.taskClaims,'UNVERIFIED');
  assert.equal(result.compatibilityWitness.present,true);
  assert.equal(result.compatibilityWitness.valid,true);
  assert.equal(result.compatibilityWitness.reviewedHead,s1Sha);
  assert.equal(result.authorized,false);assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  // a different S1 invalidates the same witness (exact-S1 scoping)
  const other=dualEpochHarness({headShas:[s2Sha,s2Sha],
    currentRoster:async({sourceSha})=>currentNativeProjection({sourceSha})});
  await other.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await other.reader.readDualEpoch({compatibilityWitness:witness})).reason,
    'SUCCESSOR_COMPATIBILITY_WITNESS_INVALID');
});

test('NEG-A2-01: force-pushed/unproven ancestry or S2 appearing during read fail closed',async()=>{
  const diverged=dualEpochHarness({headShas:[s1Sha,s1Sha],
    ancestryResult:{connected:false,reason:'SUCCESSOR_ANCESTRY_UNPROVEN'}});
  await diverged.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await diverged.reader.readDualEpoch()).reason,'SUCCESSOR_ANCESTRY_UNPROVEN');
  const drifted=dualEpochHarness({headShas:[s1Sha,s2Sha]});
  await drifted.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await drifted.reader.readDualEpoch()).reason,'SUCCESSOR_BRANCH_DRIFT_DURING_READ');
});

test('NEG-A2-02: changed code with an unreviewed/impersonal witness is refused',async()=>{
  const h=dualEpochHarness({headShas:[s1Sha,s1Sha]});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  for(const mutant of [
    successorWitness({omit:['ciRefs']}),
    successorWitness({omit:['reviewRef']}),
    successorWitness({omit:['changedCodeBlobs']}),
    successorWitness({extraKeys:{reviewRef:'https://github.com/evil/tally/pull/1'}}),
  ]){
    const result=await h.reader.readDualEpoch({compatibilityWitness:mutant});
    assert.equal(result.connected,false);
    assert.equal(result.reason,'SUCCESSOR_COMPATIBILITY_WITNESS_INVALID');
  }
});

test('NEG-A2-03: CAS revision or frozen Pack/witness mismatch fails closed',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  let reads=0;
  const unstableStore={
    async get(key){
      const value=await h.store.get(key);
      reads++;
      if(reads===2)return {...value,revision:value.revision+1};
      return value;
    }
  };
  const reader=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:unstableStore,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader,
    branchHeadReader:async()=>({connected:true,repository,versionId,headSha:s1Sha}),
    ancestryReader:async()=>({connected:true,repository,ancestry:'BASE_IS_ANCESTOR'}),
    currentRosterReader:async({sourceSha})=>currentNativeProjection({sourceSha})});
  assert.equal((await reader.readDualEpoch()).reason,'FROZEN_DAG_CHANGED_DURING_READ');
  const tampered=dualEpochHarness();
  await tampered.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const badPackReader=async()=>({status:'VERIFIED',blobSha:packBlobSha,
    content:JSON.stringify({...validPack(),edges:[]})});
  const r2=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:tampered.store,
    taskPackReader:badPackReader,reviewWitnessReader:tampered.reviewWitnessReader,
    ownerApprovalReader:tampered.ownerApprovalReader,
    branchHeadReader:async()=>({connected:true,repository,versionId,headSha:s1Sha}),
    ancestryReader:async()=>({connected:true,repository,ancestry:'BASE_IS_ANCESTOR'}),
    currentRosterReader:async({sourceSha})=>currentNativeProjection({sourceSha})});
  assert.equal((await r2.readDualEpoch()).reason,'FROZEN_DAG_CONTENT_MISMATCH');
});

test('NEG-A2-04: native 51/29 drift, duplicate or foreign Issue at S1 is refused',async()=>{
  const h=dualEpochHarness({headShas:[s1Sha,s1Sha],currentRoster:async({sourceSha})=>{
    const pack=validPack();
    const roster=pack.taskIssues.slice(1); // one missing
    const base=currentNativeProjection({sourceSha});
    return {...base,
      tasks:roster.map(t=>({id:String(t.issueNumber),versionId,title:t.title,
        issueState:t.issueState,updatedAt:t.updatedAt})),
      dag:{...base.dag,nodes:roster.map(t=>({taskId:String(t.issueNumber),versionId}))}};
  }});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await h.reader.readDualEpoch()).reason,'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
  const reversed=dualEpochHarness({headShas:[s1Sha,s1Sha],currentRoster:async({sourceSha})=>{
    const base=currentNativeProjection({sourceSha});
    return {...base,dag:{...base.dag,edges:[{from:'264',to:'260',kind:'NATIVE',
      evidenceRef:issueUrl(260),dependencyRef:issueUrl(264)}]}};
  }});
  await reversed.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await reversed.reader.readDualEpoch()).reason,'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
});

test('NEG-A2-05: an S0 grant or authorization-carrying extra witness key never gains S1 authority',async()=>{
  const h=dualEpochHarness({headShas:[s1Sha,s1Sha]});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const staleGrant=successorWitness({observedVersionSha:versionSha});
  assert.equal((await h.reader.readDualEpoch({compatibilityWitness:staleGrant})).reason,
    'SUCCESSOR_COMPATIBILITY_WITNESS_INVALID');
  const forge=successorWitness({extraKeys:{authorized:true,admitted:true}});
  const forged=await h.reader.readDualEpoch({compatibilityWitness:forge});
  assert.equal(forged.reason,'SUCCESSOR_COMPATIBILITY_WITNESS_INVALID');
  const expired=successorWitness({extraKeys:{expiresAt:'2026-11-01T00:00:00.000Z'}});
  assert.equal((await h.reader.readDualEpoch({compatibilityWitness:expired})).reason,
    'SUCCESSOR_COMPATIBILITY_WITNESS_INVALID');
});

// ---------------------------------------------------------------------------
// J02 per-edge amendment disposition (#392/7003690062 + #392/7004225002):
// three lawfully removed native edges (#264<-#260/#261/#273, GitHub edge ids
// 6151430262/6151430563/6175524365) admitted by an exact-mirror attested
// ledger. The S0 frozen edge set, Pack blob and identity digests stay
// untouched; every drift the ledger does not exactly mirror still fails
// closed, and no amendment relaxes the S1==S0 strict equality.
// ---------------------------------------------------------------------------
function threeRemovalPack(){
  const pack=validPack();
  pack.source.taskIssueCount=7;
  pack.taskIssues=[211,212,260,261,273,210,264].map(n=>({
    issueNumber:n,title:'Task '+n,issueState:n===264?'open':'closed',
    labels:['type:task'],updatedAt:'2026-11-02T02:59:00Z'}));
  pack.edges=[260,261,273,210].map(from=>({from,to:264,kind:'NATIVE',
    evidenceRef:issueUrl(264),dependencyRef:issueUrl(from)}))
    .concat([{from:211,to:212,kind:'PLANNED',evidenceRef:issueUrl(212),
      dependencyRef:issueUrl(211)}]);
  pack.claimSubjects=[absentSubject(211),absentSubject(212),observedSubject,
    absentSubject(261),absentSubject(273),absentSubject(210),absentSubject(264)];
  return pack;
}
const removalEvidence='https://github.com/'+repository+
  '/issues/264#issuecomment-7003690062';
const threeRemovals=()=>[260,261,273].map(from=>({kind:'NATIVE_EDGE_REMOVED',
  from,to:264,evidenceRef:removalEvidence}));
function removalsLiveProjection({sourceSha,extraEdges=[]}={}){
  const base=currentNativeProjection({pack:threeRemovalPack(),sourceSha,progressed:true});
  const live=base.dag.edges.filter(e=>![260,261,273].includes(Number(e.from)));
  return {...base,dag:{...base.dag,edges:live.concat(extraEdges)}};
}
function amendedDualHarness({edgeAmendments=null,headShas=[s1Sha,s1Sha],
  currentRoster=null}={}){
  const h=freezeHarness({pack:threeRemovalPack()});
  let headReads=0;
  const reader=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader,
    branchHeadReader:async()=>({connected:true,repository,versionId,
      headSha:headShas[Math.min(headReads++,headShas.length-1)]}),
    ancestryReader:async()=>({connected:true,repository,baseSha:versionSha,
      headSha:s1Sha,ancestry:'BASE_IS_ANCESTOR'}),
    currentRosterReader:currentRoster??
      (async({sourceSha})=>removalsLiveProjection({sourceSha})),
    edgeAmendments,now:()=>'2026-11-19T10:00:00.000Z'});
  return {...h,reader};
}

test('POS-A2-06: the three attested removals admit S1 structurally while S0 keeps its frozen edge set verbatim',async()=>{
  const pack=threeRemovalPack();
  const h=amendedDualHarness({edgeAmendments:threeRemovals()});
  const freeze=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(freeze.status,'FROZEN',JSON.stringify(freeze));
  const result=await h.reader.readDualEpoch();
  assert.equal(result.connected,true);
  // S0 leg unchanged: frozen edges keep all four NATIVE edges incl. the removed three.
  assert.equal(result.historical.status,'FROZEN_HISTORICAL_VERIFIED');
  const frozenNative=result.historical.frozen.edges.filter(e=>e.kind==='NATIVE');
  assert.equal(frozenNative.length,4);
  assert.ok(frozenNative.some(e=>e.from===260&&e.to===264));
  assert.ok(frozenNative.some(e=>e.from===261&&e.to===264));
  assert.ok(frozenNative.some(e=>e.from===273&&e.to===264));
  // Identity digests still bind the FROZEN cohort, never the amended live one.
  const digests=versionDagIdentityDigests({taskIssues:pack.taskIssues,edges:pack.edges});
  assert.equal(result.current.native29EdgeDigest,digests.native29EdgeDigest);
  assert.equal(result.current.original51IdsDigest,digests.original51IdsDigest);
  // S1 leg: structural pass over 26-vs-29-style drift, still honest STALE
  // without a witness, with observed counts and the applied ledger exposed.
  assert.equal(result.current.nativeEdgeCount,4);
  assert.equal(result.current.observedNativeEdgeCount,1);
  assert.equal(result.current.edgeAmendmentLedger.declared,3);
  assert.deepEqual(result.current.edgeAmendmentLedger.applied.map(a=>a.from),
    [260,261,273]);
  assert.equal(result.current.edgeAmendmentLedger.applied[0].evidenceRef,
    removalEvidence);
  assert.equal(result.current.qualified,false);
  assert.equal(result.current.currentness,'STALE');
  assert.equal(result.current.reason,'CURRENT_SOURCE_COMPATIBILITY_UNWITNESSED');
  assert.equal(result.current.taskClaims,'UNVERIFIED');
  assert.equal(result.authorized,false);assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.equal(h.store.count(),1);
});

test('POS-A2-07: after the amended structural match a genuine witness yields CURRENT_STRUCTURAL_COMPATIBLE',async()=>{
  const h=amendedDualHarness({edgeAmendments:threeRemovals()});
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const witness=successorWitness({pack:threeRemovalPack()});
  const result=await h.reader.readDualEpoch({compatibilityWitness:witness});
  assert.equal(result.connected,true);
  assert.equal(result.current.status,'CURRENT_STRUCTURAL_COMPATIBLE');
  assert.equal(result.current.qualified,true);
  assert.equal(result.current.currentness,'CURRENT');
  assert.equal(result.current.reason,null);
  assert.equal(result.current.observedNativeEdgeCount,1);
  assert.equal(result.current.edgeAmendmentLedger.applied.length,3);
  assert.equal(result.compatibilityWitness.present,true);
  assert.equal(result.compatibilityWitness.valid,true);
  assert.equal(result.compatibilityWitness.reviewedHead,s1Sha);
  assert.equal(result.authorized,false);assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
});

test('NEG-A2-06: unattested, mis-attested or foreign drift still fails closed; amendments never relax S1==S0',async()=>{
  // (1) three removals with NO ledger — the documented dff5329e blocker itself.
  const none=amendedDualHarness();
  await none.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await none.reader.readDualEpoch()).reason,
    'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
  // (2) ledger declares only two of the three removals.
  const two=amendedDualHarness({edgeAmendments:threeRemovals().slice(0,2)});
  await two.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await two.reader.readDualEpoch()).reason,
    'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
  // (3) ledger over-declares: 210->264 attested but still live.
  const over=amendedDualHarness({edgeAmendments:threeRemovals()
    .concat([{kind:'NATIVE_EDGE_REMOVED',from:210,to:264,
      evidenceRef:removalEvidence}])});
  await over.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await over.reader.readDualEpoch()).reason,
    'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
  // (4) attested edge never existed in the frozen cohort.
  const foreign=amendedDualHarness({edgeAmendments:threeRemovals()
    .concat([{kind:'NATIVE_EDGE_REMOVED',from:617,to:264,
      evidenceRef:removalEvidence}])});
  await foreign.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await foreign.reader.readDualEpoch()).reason,
    'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
  // (5) live graph ADDS a native edge the ledger does not cover (no re-added
  // or foreign edge is ever admissible, attested or not).
  const added=amendedDualHarness({edgeAmendments:threeRemovals(),
    currentRoster:async({sourceSha})=>removalsLiveProjection({sourceSha,
      extraEdges:[{from:'261',to:'211',kind:'NATIVE',
        evidenceRef:issueUrl(211),dependencyRef:issueUrl(261)}]})});
  await added.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal((await added.reader.readDualEpoch()).reason,
    'SUCCESSOR_NATIVE_ROSTER_UNQUALIFIED');
  // (6) malformed ledger (extra key / foreign evidenceRef) is rejected at
  // construction, never silently ignored.
  assert.throws(()=>amendedDualHarness({edgeAmendments:[{kind:'NATIVE_EDGE_REMOVED',
    from:260,to:264,evidenceRef:removalEvidence,authorized:true}]}),
    /INVALID_EDGE_AMENDMENT_LEDGER/);
  assert.throws(()=>amendedDualHarness({edgeAmendments:[{kind:'NATIVE_EDGE_REMOVED',
    from:260,to:264,evidenceRef:'https://github.com/evil/tally/issues/1'}]}),
    /INVALID_EDGE_AMENDMENT_LEDGER/);
  // (7) S1==S0 keeps STRICT equality: the ledger is inert for same-source.
  const same=amendedDualHarness({edgeAmendments:threeRemovals(),
    headShas:[versionSha,versionSha],
    currentRoster:async({sourceSha})=>currentNativeProjection(
      {pack:threeRemovalPack(),sourceSha})});
  await same.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const sameResult=await same.reader.readDualEpoch();
  assert.equal(sameResult.connected,true);
  assert.equal(sameResult.current.status,'CURRENT_SAME_SOURCE');
  assert.equal(sameResult.current.qualified,true);
  assert.equal(sameResult.current.edgeAmendmentLedger??null,null);
});

test('ERR-A2-01: GitHub head/transport failures stay denied and unknown, never empty-healthy',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const deniedHead=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader,
    branchHeadReader:async()=>({connected:false,reason:'GITHUB_ACCESS_DENIED'}),
    ancestryReader:async()=>({connected:true,repository,ancestry:'BASE_IS_ANCESTOR'}),
    currentRosterReader:async({sourceSha})=>currentNativeProjection({sourceSha})});
  assert.equal((await deniedHead.readDualEpoch()).reason,'SUCCESSOR_HEAD_OBSERVATION_UNQUALIFIED');
  const throwingRoster=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader,
    branchHeadReader:async()=>({connected:true,repository,versionId,headSha:s1Sha}),
    ancestryReader:async()=>({connected:true,repository,ancestry:'BASE_IS_ANCESTOR'}),
    currentRosterReader:async()=>{throw new Error('GH 500');}});
  assert.equal((await throwingRoster.readDualEpoch()).reason,'FROZEN_DAG_READ_FAILED');
});

test('ERR-A2-02: Host CAS unavailable fails closed as unknown',async()=>{
  const h=freezeHarness();
  await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const brokenStore={async get(){throw new Error('SQLITE_BUSY');}};
  const reader=createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:brokenStore,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader,
    branchHeadReader:async()=>({connected:true,repository,versionId,headSha:versionSha}),
    ancestryReader:async()=>({connected:true,repository,ancestry:'BASE_IS_ANCESTOR'}),
    currentRosterReader:async({sourceSha})=>currentNativeProjection({sourceSha})});
  const result=await reader.readDualEpoch();
  assert.equal(result.connected,false);
  assert.equal(result.reason,'FROZEN_DAG_READ_FAILED');
});

test('identity digests are canonical and cohort-sensitive; the dual-Epoch reader requires its trusted ports',async()=>{
  const pack=validPack();
  const a=versionDagIdentityDigests({taskIssues:pack.taskIssues,edges:pack.edges});
  const reordered=versionDagIdentityDigests({taskIssues:[...pack.taskIssues].reverse(),
    edges:[...pack.edges].reverse()});
  assert.equal(a.original51IdsDigest,reordered.original51IdsDigest);
  assert.equal(a.native29EdgeDigest,reordered.native29EdgeDigest);
  const retitled=versionDagIdentityDigests({taskIssues:pack.taskIssues.map((t,i)=>
    i===0?{...t,title:'Task 211 renamed'}:t),edges:pack.edges});
  assert.notEqual(a.original51IdsDigest,retitled.original51IdsDigest);
  assert.throws(()=>createHostVersionTaskDagDualEpochReader({
    repository,versionId,freezeBaseSha:versionSha,store:memoryStore(),
    taskPackReader:async()=>({}),reviewWitnessReader:async()=>({}),
    ownerApprovalReader:async()=>({}),branchHeadReader:async()=>({}),
    ancestryReader:async()=>({}),
    now:()=>''}),/TRUSTED_DUAL_EPOCH_READER_REQUIRED/);
});
