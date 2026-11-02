import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiRosterHost} from '../../../web/v2/ui-roster-host.mjs';
import {
  createHostCloseoutClaimReadbackPortFromClient,
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
    mutate(key,fn){const row=rows.get(key);rows.set(key,{revision:row.revision+1,value:fn(structuredClone(row.value))});}
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
function freezeHarness({pack=validPack(),reviewValue=review(),approvalValue=approval(),store=memoryStore()}={}){
  const content=JSON.stringify(pack);
  const taskPackReader=async()=>({status:'VERIFIED',blobSha:packBlobSha,content});
  const nativeRosterReader=async()=>({
    connected:true,source:{repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'},
    versionId,tasks:pack.taskIssues.map(t=>({id:String(t.issueNumber),versionId,
      title:t.title,issueState:t.issueState,updatedAt:t.updatedAt})),
    dag:{connected:true,source:{repository,sourceSha:versionSha,
      currentness:'CURRENT',readback:'VERIFIED'},versionId,
      asOf:{sourceSha:versionSha},
      nodes:pack.taskIssues.map(t=>({taskId:String(t.issueNumber),versionId})),
      edges:pack.edges.filter(e=>e.kind==='NATIVE').map(e=>({...e,
        from:String(e.from),to:String(e.to),
        dependencyRef:issueUrl(e.from)}))}
  });
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

test('version Task DAG freeze and readback bind complete roster, native/planned edges and exact witnesses',async()=>{
  const h=freezeHarness();
  const frozen=await h.writer.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  assert.equal(frozen.taskCount,4);
  assert.equal(frozen.edgeCount,2);
  assert.equal(frozen.authorized,false);
  const reader=createHostVersionTaskDagReader({
    repository,versionId,expectedVersionSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader
  });
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
  const reader=createHostVersionTaskDagReader({
    repository,versionId,expectedVersionSha:versionSha,store:unstableStore,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader
  });
  assert.equal((await reader.readFrozenDag()).reason,'FROZEN_DAG_CHANGED_DURING_READ');
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
  const args={...claimArgs(),claimSubject:absentSubject(260)};
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
      'The frozen write sets are recorded: A=7, B=10, C=12, D=7.')],
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
  const frozenHost=createHostVersionTaskDagReader({
    repository,versionId,expectedVersionSha:versionSha,store:h.store,
    taskPackReader:h.taskPackReader,reviewWitnessReader:h.reviewWitnessReader,
    ownerApprovalReader:h.ownerApprovalReader
  });
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
      'The frozen write sets are recorded: A=7, B=10, C=12, D=7.')],
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
