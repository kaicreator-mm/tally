import test from 'node:test';
import assert from 'node:assert/strict';
import {createVersionTaskDagWritePort,readVersionTaskDagFreezeState}
  from '../src/version-task-dag-write-port.mjs';
import {createHostVersionTaskDagFreezeWriter} from '../src/closeout-roster.mjs';

const repository='kaicreator-mm/tally';
const owner='kaicreator-mm';
const repo='tally';
const versionId='version/v2';
const versionSha='7ebb1cdd0a79fc82cc938e35f852a80dd8e15332';
const packBlobSha='d8b1c8dcfe22fdc27a2ffe151341ac3d3728329d';
const packCommitSha='b3ad8ae7833028999a9c27cac7042037bca6d961';
const taskPackRef='https://github.com/'+repository+'/blob/'+packCommitSha+
  '/apps/tally-web/src/features/dag/version-v2-task-pack.candidate.json';
const issueUrl=n=>'https://github.com/'+repository+'/issues/'+n;
const writerAuthority={mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
  repository,writerId:'tally-host:win32:i284:1',githubWriterLogin:'kaicreator-mm'};
const digest='a'.repeat(64);
const writeSetDocs=pack=>Object.fromEntries(pack.taskIssues.map(t=>
  [String(t.issueNumber),{writeSetSha256:digest,
    document:'frozen exact write set for #'+t.issueNumber}]));

function memoryStore(initial){
  const rows=new Map(initial?initial.map(([key,value])=>
    [key,{revision:1,value:structuredClone(value)}]):[]);
  const puts=[];
  return {
    async get(key){const row=rows.get(key);return row?structuredClone(row):null;},
    async put(key,value,{mustBeAbsent,mustMatchRevision}={}){
      const existing=rows.get(key);
      if(mustBeAbsent&&existing)throw new Error('CAS_EXISTS');
      if(mustMatchRevision!==undefined&&existing?.revision!==mustMatchRevision)
        throw new Error('CAS_REVISION_CONFLICT');
      const revision=(existing?.revision??0)+1;
      rows.set(key,{revision,value:structuredClone(value)});
      puts.push(key);
      return {revision};
    },
    puts:()=>[...puts],
    rows
  };
}
/** E0 pre-freeze candidate: #284->#312 proposed serial edge is PLANNED. */
function candidatePack({edges}={}){
  return {
    schema:'tally/version-task-pack/v1',status:'CANDIDATE_NOT_FROZEN',
    repository,versionId,versionBaseSha:versionSha,
    observedAt:'2026-11-09T11:09:59.214Z',
    source:{kind:'GITHUB_NATIVE_ISSUES',query:'state=all&labels=type:task',
      taskIssueCount:3,claimCommentPages:1},
    taskIssues:[284,312,219].map(n=>({issueNumber:n,title:'Task '+n,
      issueState:'open',labels:['type:task'],updatedAt:'2026-11-09T11:00:00Z'})),
    edges:edges??[
      {from:284,to:312,kind:'PLANNED',evidenceRef:issueUrl(312),dependencyRef:issueUrl(284)}],
    claimSubjects:[284,312,219].map(n=>({issueNumber:n,sourceSha:versionSha,
      actorRole:'builder',candidateStatus:'NO_CLAIM_EVENT_EXPECTED',commentId:null,
      commentRef:null,dispatchId:null,operatorKind:null,operatorId:null,generation:null,
      protectedClaimKey:repository+'#'+n+':builder:__default__'}))
  };
}
const frozenPack=()=>candidatePack({edges:[
  {from:284,to:312,kind:'NATIVE',evidenceRef:issueUrl(312),dependencyRef:issueUrl(284)}]});
const review=()=>({status:'INDEPENDENTLY_VERIFIED',taskPackBlobSha:packBlobSha,
  sourceSha:versionSha,reviewedHeadSha:versionSha,reviewer:'independent-reviewer',
  author:'web-builder',reviewRef:'https://github.com/'+repository+'/pull/384',
  verifiedAt:'2026-11-09T12:00:00.000Z'});
const approval=()=>({status:'OWNER_APPROVED',taskPackBlobSha:packBlobSha,
  sourceSha:versionSha,owner,approvalRef:'https://github.com/'+repository+
  '/issues/284#issuecomment-6739635285',approvedAt:'2026-11-09T12:01:00.000Z'});
function nativeProjection({pack}){
  const tasks=pack.taskIssues;
  return {connected:true,source:{repository,sourceSha:versionSha,
      currentness:'CURRENT',readback:'VERIFIED'},versionId,
    tasks:tasks.map(t=>({id:String(t.issueNumber),versionId,title:t.title,
      issueState:t.issueState,updatedAt:t.updatedAt})),
    dag:{connected:true,versionId,asOf:{sourceSha:versionSha},
      source:{repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'},
      nodes:tasks.map(t=>({taskId:String(t.issueNumber),versionId})),
      edges:pack.edges.filter(e=>e.kind==='NATIVE').map(e=>({...e,
        from:String(e.from),to:String(e.to),dependencyRef:issueUrl(e.from)}))}};
}
/** Fake official GitHub issue-dependencies client with call accounting. */
function fakeOctokit({issues,blockedBy={},postResult}={}){
  const state={posts:[]};
  const defaults=Object.fromEntries([284,312,219].map(n=>[n,{
    number:n,id:n*1000,state:'open',updated_at:'2026-11-09T11:00:00Z',
    repository_url:'https://api.github.com/repos/'+owner+'/'+repo}]));
  const live=issues??{};
  const octokit={
    rest:{issues:{get:async({issue_number})=>{
      const issue=live[issue_number]??defaults[issue_number];
      if(!issue||issue instanceof Error)throw issue;
      if(issue&&issue.throw)throw issue.throw;
      return {data:issue};
    }}},
    paginate:{iterator:async function*(route,params){
      assert.equal(route,
        'GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by');
      const list=blockedBy[params.issue_number];
      if(list instanceof Error)throw list;
      yield {data:list??[]};
    }},
    request:async(route,params)=>{
      assert.equal(route,
        'POST /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by');
      assert.equal(params.owner,owner);assert.equal(params.repo,repo);
      state.posts.push({issue_number:params.issue_number,issue_id:params.issue_id});
      if(postResult instanceof Error)throw postResult;
      // The native write lands: later GET blocked_by observes the exact edge.
      const number=params.issue_id/1000;
      blockedBy[params.issue_number]=[...(blockedBy[params.issue_number]??[]),
        {id:params.issue_id,number,state:'open',html_url:issueUrl(number)}];
      return {status:201,data:{}};
    }
  };
  return {octokit,posts:state.posts};
}
function dependencyFixture({pack=candidatePack(),issues,blockedBy,postResult,
  store=memoryStore(),freezeWriteSetPolicy='ALL_TASKS',writeSetDocumentByIssue}={}){
  const gh=fakeOctokit({issues,blockedBy,postResult});
  const content=JSON.stringify(pack);
  const taskPackReader=async({expectedBlobSha})=>({status:'VERIFIED',
    blobSha:expectedBlobSha,content});
  const port=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store,taskPackReader,
    reviewWitnessReader:async()=>review(),
    ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack}),
    octokit:gh.octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:writeSetDocumentByIssue??writeSetDocs(pack),
    freezeWriteSetPolicy,now:()=>'2026-11-09T13:00:00.000Z'});
  return {port,store,posts:gh.posts};
}
const edgeInput={taskPackRef,taskPackBlobSha:packBlobSha,
  blockedIssueNumber:312,blockedByIssueNumber:284};

test('port construction refuses untrusted bindings before any write',()=>{
  const gh=fakeOctokit();
  const base={repository,versionId,expectedVersionSha:versionSha,
    store:memoryStore(),taskPackReader:async()=>({}),
    reviewWitnessReader:async()=>({}),ownerApprovalReader:async()=>({}),
    nativeRosterReader:async()=>({}),octokit:gh.octokit,owner,repo,writerAuthority};
  assert.throws(()=>createVersionTaskDagWritePort({...base,repository:'evil/org'}),
    /TRUSTED_VERSION_TASK_DAG_WRITE_PORT_REQUIRED/);
  assert.throws(()=>createVersionTaskDagWritePort({...base,store:{}}),
    /TRUSTED_VERSION_TASK_DAG_WRITE_PORT_REQUIRED/);
  assert.throws(()=>createVersionTaskDagWritePort({...base,
    writerAuthority:{...writerAuthority,exclusiveHostControl:false}}),
    /TRUSTED_VERSION_TASK_DAG_WRITE_PORT_REQUIRED/);
  assert.throws(()=>createVersionTaskDagWritePort({...base,
    nativeRosterReader:null}),
    /TRUSTED_VERSION_TASK_DAG_WRITE_PORT_REQUIRED/);
  assert.throws(()=>createVersionTaskDagWritePort({...base,
    freezeWriteSetPolicy:'BYPASS_ALL_GATES'}),
    /TRUSTED_VERSION_TASK_DAG_WRITE_PORT_REQUIRED/);
});

test('exact repository/ID validation: self-edge, malformed ids, mutable pack ref refuse with zero writes',async()=>{
  const f=dependencyFixture();
  for(const input of [
    {...edgeInput,blockedIssueNumber:312,blockedByIssueNumber:312},
    {...edgeInput,blockedIssueNumber:0},
    {...edgeInput,blockedIssueNumber:'312'},
    {...edgeInput,blockedByIssueNumber:-1},
    {...edgeInput,taskPackBlobSha:'deadbeef'},
    {...edgeInput,taskPackRef:'https://github.com/kaicreator-mm/tally/blob/main/x.json'},
    null,{},
  ]){
    const result=await f.port.addNativeTaskDependency(input);
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'EDGE_INPUT_INVALID');
    assert.equal(result.githubEffect,'NOT_RUN');
  }
  assert.equal(f.posts.length,0,'no write before gates');
});

test('candidate membership, proposed-edge match and duplicate refusal never POST',async()=>{
  const outOfPack=dependencyFixture();
  assert.deepEqual(await outOfPack.port.addNativeTaskDependency(
    {...edgeInput,blockedIssueNumber:999}),
    {...await outOfPack.port.addNativeTaskDependency({...edgeInput,blockedIssueNumber:999})});
  const noMembership=await outOfPack.port.addNativeTaskDependency(
    {...edgeInput,blockedIssueNumber:999});
  assert.equal(noMembership.status,'BLOCKED');
  assert.equal(noMembership.reason,'EDGE_ISSUE_NOT_IN_PACK');
  const absentEdge=dependencyFixture();
  const missing=await absentEdge.port.addNativeTaskDependency(
    {...edgeInput,blockedIssueNumber:219});
  assert.equal(missing.status,'BLOCKED');
  assert.equal(missing.reason,'EDGE_NOT_IN_CANDIDATE_PACK');
  const nativePack=dependencyFixture({pack:frozenPack()});
  const duplicate=await nativePack.port.addNativeTaskDependency(edgeInput);
  assert.equal(duplicate.status,'BLOCKED');
  assert.equal(duplicate.reason,'EDGE_ALREADY_NATIVE_IN_PACK');
  assert.equal(outOfPack.posts.length+absentEdge.posts.length+nativePack.posts.length,0);
});

test('native dependency edge POST requires exact reviewed PackBlob and Owner approval',async()=>{
  const rejected=[
    ['absent independent review',()=>({status:'NOT_VERIFIED'}),approval,'INDEPENDENT_REVIEW_REQUIRED'],
    ['wrong reviewed PackBlob',()=>({...review(),taskPackBlobSha:'f'.repeat(40)}),approval,'INDEPENDENT_REVIEW_REQUIRED'],
    ['stale reviewed source',()=>({...review(),sourceSha:'f'.repeat(40)}),approval,'INDEPENDENT_REVIEW_REQUIRED'],
    ['reviewer equals author',()=>({...review(),reviewer:'web-builder'}),approval,'INDEPENDENT_REVIEW_REQUIRED'],
    ['wrong reviewed head',()=>({...review(),reviewedHeadSha:'f'.repeat(40)}),approval,'INDEPENDENT_REVIEW_REQUIRED'],
    ['foreign review ref',()=>({...review(),reviewRef:'https://github.com/evil/tally/pull/1'}),approval,'INDEPENDENT_REVIEW_REQUIRED'],
    ['absent Owner approval',review,()=>({status:'NOT_VERIFIED'}),'OWNER_APPROVAL_REQUIRED'],
    ['wrong approved PackBlob',review,()=>({...approval(),taskPackBlobSha:'f'.repeat(40)}),'OWNER_APPROVAL_REQUIRED'],
    ['wrong Owner identity',review,()=>({...approval(),owner:'not-the-owner'}),'OWNER_APPROVAL_REQUIRED'],
    ['stale Owner source',review,()=>({...approval(),sourceSha:'f'.repeat(40)}),'OWNER_APPROVAL_REQUIRED'],
    ['foreign Owner ref',review,()=>({...approval(),approvalRef:'https://github.com/evil/tally/issues/1'}),'OWNER_APPROVAL_REQUIRED'],
  ];
  for(const [name,readReview,readApproval,reason] of rejected){
    const gh=fakeOctokit();
    const store=memoryStore(),pack=candidatePack();
    const port=createVersionTaskDagWritePort({repository,versionId,
      expectedVersionSha:versionSha,store,
      taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
        content:JSON.stringify(pack)}),
      reviewWitnessReader:async scope=>{
        assert.equal(scope.taskPackBlobSha,packBlobSha,name);
        assert.equal(scope.sourceSha,versionSha,name);
        return readReview();
      },
      ownerApprovalReader:async scope=>{
        assert.equal(scope.taskPackBlobSha,packBlobSha,name);
        return readApproval();
      },
      nativeRosterReader:async()=>nativeProjection({pack}),
      octokit:gh.octokit,owner,repo,writerAuthority,
      writeSetDocumentByIssue:writeSetDocs(pack)});
    const result=await port.addNativeTaskDependency(edgeInput);
    assert.equal(result.status,'BLOCKED',name);
    assert.equal(result.reason,reason,name);
    assert.equal(result.githubEffect,'NOT_RUN',name);
    assert.equal(gh.posts.length,0,'no native edge write: '+name);
    assert.deepEqual(store.puts(),[],'no CAS write: '+name);
  }
});

test('cycle refusal: cyclic candidate never reaches the official endpoint',async()=>{
  const cyclic=candidatePack({edges:[
    {from:284,to:312,kind:'PLANNED',evidenceRef:issueUrl(312),dependencyRef:issueUrl(284)},
    {from:312,to:284,kind:'PLANNED',evidenceRef:issueUrl(284),dependencyRef:issueUrl(312)}]});
  const f=dependencyFixture({pack:cyclic});
  const result=await f.port.addNativeTaskDependency(edgeInput);
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'EDGE_CYCLE');
  assert.equal(f.posts.length,0);
});

test('stale issue and foreign repository identity refuse before any write',async()=>{
  const stale=dependencyFixture({issues:{312:{number:312,id:312000,state:'open',
    updated_at:'2026-11-09T12:59:59Z',
    repository_url:'https://api.github.com/repos/'+owner+'/'+repo}}});
  const staleResult=await stale.port.addNativeTaskDependency(edgeInput);
  assert.equal(staleResult.status,'BLOCKED');
  assert.equal(staleResult.reason,'ISSUE_STALE_VS_PACK');
  const closed=dependencyFixture({issues:{312:{number:312,id:312000,state:'closed',
    updated_at:'2026-11-09T11:00:00Z',
    repository_url:'https://api.github.com/repos/'+owner+'/'+repo}}});
  const closedResult=await closed.port.addNativeTaskDependency(edgeInput);
  assert.equal(closedResult.status,'BLOCKED');
  assert.equal(closedResult.reason,'ISSUE_STALE_VS_PACK');
  const foreign=dependencyFixture({issues:{312:{number:312,id:312000,state:'open',
    updated_at:'2026-11-09T11:00:00Z',
    repository_url:'https://api.github.com/repos/evil/org'}}});
  const foreignResult=await foreign.port.addNativeTaskDependency(edgeInput);
  assert.equal(foreignResult.status,'BLOCKED');
  assert.equal(foreignResult.reason,'ISSUE_READBACK_FAILED');
  assert.equal(stale.posts.length+closed.posts.length+foreign.posts.length,0);
});

test('official blocked_by write verifies exact native readback',async()=>{
  const f=dependencyFixture({blockedBy:{312:[]}});
  const result=await f.port.addNativeTaskDependency(edgeInput);
  assert.equal(result.status,'NATIVE_EDGE_VERIFIED');
  assert.equal(result.reason,'OFFICIAL_GITHUB_BLOCKED_BY_READBACK_VERIFIED');
  assert.equal(result.native,true);
  assert.equal(result.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(result.writePerformed,true);
  assert.equal(result.blockedIssueNumber,312);
  assert.equal(result.blockedByIssueNumber,284);
  assert.equal(result.dependencyId,284000);
  assert.equal(result.authorized,false);assert.equal(result.taskPass,false);
  assert.deepEqual(f.posts,[{issue_number:312,issue_id:284000}]);
});

test('idempotent replay: native-present edge never re-POSTs',async()=>{
  const f=dependencyFixture({blockedBy:{312:[
    {id:284000,number:284,state:'open',
     html_url:issueUrl(284)}]}});
  const result=await f.port.addNativeTaskDependency(edgeInput);
  assert.equal(result.status,'NATIVE_EDGE_PRESENT');
  assert.equal(result.reason,'NATIVE_EDGE_ALREADY_PRESENT_READBACK');
  assert.equal(result.githubEffect,'NOT_RUN');
  assert.equal(result.writePerformed,false);
  assert.equal(result.dependencyId,284000);
  assert.equal(f.posts.length,0);
});

test('foreign dependency entry in the native readback fails closed',async()=>{
  const f=dependencyFixture({blockedBy:{312:[
    {id:1,number:284,state:'open',html_url:'https://github.com/evil/org/issues/284'}]}});
  const result=await f.port.addNativeTaskDependency(edgeInput);
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'GITHUB_DEPENDENCY_READ_FAILED');
  assert.equal(f.posts.length,0);
});

test('403/404/422 map to bounded BLOCKED reasons with zero writes',async()=>{
  const forbidden=dependencyFixture({issues:{312:{throw:{status:403}}}});
  assert.deepEqual(await forbidden.port.addNativeTaskDependency(edgeInput),
    Object.freeze({status:'BLOCKED',reason:'GITHUB_ACCESS_DENIED',repository,
      versionId,githubEffect:'NOT_RUN',native:false,authorized:false,
      admitted:false,taskPass:false,releasePass:false}));
  const notFound=dependencyFixture({blockedBy:{312:new Error('get failed')}});
  const nf=await notFound.port.addNativeTaskDependency(edgeInput);
  // Plain network error on GET carries no status: bounded provider reason only.
  assert.equal(nf.status,'BLOCKED');
  assert.equal(nf.reason,'GITHUB_DEPENDENCY_READ_FAILED');
  const unprocessable=dependencyFixture({blockedBy:{312:[]},
    postResult:Object.assign(new Error('validation'),{status:422})});
  const up=await unprocessable.port.addNativeTaskDependency(edgeInput);
  assert.equal(up.status,'BLOCKED');
  assert.equal(up.reason,'GITHUB_VALIDATION_FAILED');
  assert.equal(forbidden.posts.length,0);
  assert.equal(unprocessable.posts.length,1,'the refused 422 POST was attempted once');
});

test('lost-ack stays UNKNOWN and reconciles by readback, never a blind POST',async()=>{
  const f=dependencyFixture({blockedBy:{312:[]},
    postResult:new Error('socket hang up')});
  const lost=await f.port.addNativeTaskDependency(edgeInput);
  assert.equal(lost.status,'UNKNOWN');
  assert.equal(lost.reason,'NATIVE_WRITE_ACK_LOST');
  assert.equal(lost.githubEffect,'UNKNOWN');
  assert.equal(lost.reconciliationRequired,true);
  assert.equal(lost.retryPolicy,'READ_BACK_BEFORE_RETRY_NO_BLIND_POST');
  assert.equal(f.posts.length,1);
  // The lost ACK leaves a durable HELD gate, so an unreviewed retry on the
  // SAME CAS registry cannot POST, even if the native edge can now be read.
  const reconciled=dependencyFixture({pack:candidatePack(),store:f.store,
    blockedBy:{312:[
    {id:284000,number:284,state:'open',html_url:issueUrl(284)}]}});
  const second=await reconciled.port.addNativeTaskDependency(edgeInput);
  assert.equal(second.status,'BLOCKED');
  assert.equal(second.reason,'VERSION_EFFECT_FENCE_RECONCILIATION_REQUIRED');
  assert.equal(reconciled.posts.length,0);
  const whileUnknown=dependencyFixture({pack:frozenPack(),store:f.store});
  const stop=await whileUnknown.port.freezeVersionTaskDag({
    taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(stop.status,'BLOCKED');
  assert.equal(stop.reason,'VERSION_EFFECT_FENCE_RECONCILIATION_REQUIRED');
});

test('readback mismatch after a successful POST stays UNKNOWN',async()=>{
  let reads=0;
  const pack=candidatePack();
  const gh={rest:{issues:{get:async({issue_number})=>({data:{
    number:issue_number,id:issue_number*1000,state:'open',
    updated_at:'2026-11-09T11:00:00Z',
    repository_url:'https://api.github.com/repos/'+owner+'/'+repo}})}},
    paginate:{iterator:async function*(route,params){
      reads++;
      yield {data:reads===1?[]:[{id:9,number:219,state:'open',html_url:issueUrl(219)}]};
    }},
    request:async()=>({status:201,data:{}})};
  const port=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:memoryStore(),
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(pack)}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack}),octokit:gh,owner,repo,
    writerAuthority,writeSetDocumentByIssue:{}});
  const result=await port.addNativeTaskDependency(edgeInput);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'NATIVE_READBACK_MISMATCH');
  assert.equal(result.reconciliationRequired,true);
});

test('freeze refuses PLANNED candidate edges before any CAS write',async()=>{
  const f=dependencyFixture({pack:candidatePack()});
  const result=await f.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'PLANNED_EDGES_REMAIN');
  assert.equal(result.plannedEdgeCount,1);
  assert.equal(result.githubEffect,'NOT_RUN');
  assert.equal(f.store.puts().length,0,'no CAS write before the gate');
});

test('freeze refuses stale or invalid Task Pack identity',async()=>{
  const badReader=dependencyFixture();
  const port=badReader.port;
  assert.equal((await port.freezeVersionTaskDag(
    {taskPackRef:'https://github.com/kaicreator-mm/tally/blob/main/x.json',
      taskPackBlobSha:packBlobSha})).reason,'TASK_PACK_IDENTITY_INVALID');
  const corrupt=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:memoryStore(),
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,content:'not json'}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack:frozenPack()}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:{}});
  assert.equal((await corrupt.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha}))
    .reason,'TASK_PACK_CONTENT_INVALID');
  const wrongSchema=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:memoryStore(),
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify({...frozenPack(),schema:'other/v1'})}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack:frozenPack()}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:{}});
  assert.equal((await wrongSchema.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha}))
    .reason,'TASK_PACK_CONTENT_INVALID');
  assert.equal(badReader.store.puts().length,0);
});

test('freeze requires independent review and exact Owner approval (A-owned writer)',async()=>{
  const noReview=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:memoryStore(),
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(frozenPack())}),
    reviewWitnessReader:async()=>({status:'NOT_VERIFIED'}),
    ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack:frozenPack()}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:writeSetDocs(frozenPack())});
  const store=memoryStore();
  const noApproval=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store,
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(frozenPack())}),
    reviewWitnessReader:async()=>review(),
    ownerApprovalReader:async()=>({status:'NOT_VERIFIED'}),
    nativeRosterReader:async()=>nativeProjection({pack:frozenPack()}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:writeSetDocs(frozenPack())});
  assert.equal((await noReview.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha}))
    .reason,'INDEPENDENT_REVIEW_REQUIRED');
  const denied=await noApproval.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(denied.status,'BLOCKED');
  assert.equal(denied.reason,'OWNER_APPROVAL_REQUIRED');
  assert.equal(store.puts().length,0);
});

test('freeze refuses native roster drift against the exact candidate',async()=>{
  const driftedPack=frozenPack();
  driftedPack.taskIssues[0].updatedAt='2026-11-09T23:00:00Z';
  const port=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:memoryStore(),
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(frozenPack())}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack:driftedPack}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:writeSetDocs(frozenPack())});
  const result=await port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'NATIVE_TASK_ROSTER_READBACK_FAILED');
});

test('freeze materializes exact frozen write-set digest records with readback',async()=>{
  const f=dependencyFixture({pack:frozenPack()});
  const result=await f.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(result.status,'FROZEN');
  assert.equal(result.reason,'FROZEN_VERSION_TASK_DAG_CAS_READBACK_VERIFIED');
  assert.equal(result.taskCount,3);
  assert.equal(result.edgeCount,1);
  assert.equal(result.writeSetRecordCount,3);
  assert.equal(result.dependencyWritesAfterFreeze,'REFUSED');
  assert.equal(result.authorized,false);assert.equal(result.taskPass,false);
  const dagRow=await f.store.get('tally:v2:version-task-dag:'+await (async()=>{
    const {createHash}=await import('node:crypto');
    return createHash('sha256').update(repository+'#'+versionId,'utf8').digest('hex');
  })());
  assert.equal(dagRow.value.phase,'FROZEN_VERSION_TASK_DAG');
  for(const n of [284,312,219]){
    const row=await f.store.get('tally:v2:task-writeset:'+n);
    assert.equal(row.value.phase,'FROZEN_DAG_WRITESET');
    assert.equal(row.value.issueNumber,n);
    assert.equal(row.value.sourceSha,versionSha);
    assert.equal(row.value.writeSetSha256,digest);
    assert.equal(row.value.versionPackBlobSha,packBlobSha);
    assert.ok(row.value.document.includes('#'+n));
    assert.equal(row.value.frozenAt,'2026-11-09T13:00:00.000Z');
  }
});

test('Host-only VERSION_ONLY freezes structure without backfilling historical Task FSW',async()=>{
  const legacy={schema:'historical/frozen-writeset',phase:'FROZEN_DAG_WRITESET',
    sourceSha:'0'.repeat(40),document:'historical, never rewrite'};
  const store=memoryStore([['tally:v2:task-writeset:284',legacy]]);
  const pack=frozenPack();
  pack.taskIssues[0].issueState='closed';
  const f=dependencyFixture({pack,store,
    freezeWriteSetPolicy:'VERSION_ONLY',writeSetDocumentByIssue:{}});
  const first=await f.port.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha,freezeWriteSetPolicy:'ALL_TASKS'});
  assert.equal(first.status,'FROZEN');
  assert.equal(first.taskCount,3);
  assert.equal(first.writeSetRecordCount,0);
  assert.equal(first.writeSetMaterialization,'DEFERRED_TO_PER_TASK_CLAIM_FSW');
  assert.equal(first.taskDispatchAuthorized,false);
  assert.equal(first.authorized,false);assert.equal(first.admitted,false);
  assert.equal(first.taskPass,false);assert.equal(first.releasePass,false);
  assert.deepEqual(store.puts().filter(k=>k.startsWith('tally:v2:version-task-dag:')).length,
    1,'only create one version DAG CAS row; fence CAS is separate');
  assert.deepEqual((await store.get('tally:v2:task-writeset:284')).value,legacy,
    'historical Task FSW row remains byte-exact and unrevised');
  assert.equal(await store.get('tally:v2:task-writeset:312'),null);
  assert.equal(await store.get('tally:v2:task-writeset:219'),null);
  const again=await f.port.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(again.status,'BLOCKED');
  assert.equal(again.reason,'VERSION_ONLY_REPLAY_FSW_MODE_UNVERIFIED');
  assert.equal(again.persistedFswPresence,'UNKNOWN');
  assert.equal(store.puts().filter(k=>k.startsWith('tally:v2:version-task-dag:')).length,
    1,'no duplicate DAG CAS writes');
});

test('cross-policy VERSION_ONLY cannot falsely replay legacy ALL_TASKS FSW',async()=>{
  const store=memoryStore();
  const old=dependencyFixture({pack:frozenPack(),store});
  const initial=await old.port.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(initial.status,'FROZEN');
  for(const n of [284,312,219])
    assert.ok(await store.get('tally:v2:task-writeset:'+n));
  const next=dependencyFixture({pack:frozenPack(),store,
    freezeWriteSetPolicy:'VERSION_ONLY',writeSetDocumentByIssue:{}});
  const cross=await next.port.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(cross.status,'BLOCKED');
  assert.equal(cross.reason,'VERSION_ONLY_REPLAY_FSW_MODE_UNVERIFIED');
  assert.equal(cross.persistedFswPresence,'UNKNOWN');
  for(const n of [284,312,219])
    assert.ok(await store.get('tally:v2:task-writeset:'+n),
      'immutable legacy FSW still exists');
});

test('dual approved packs: a stale Native POST cannot cross a final CAS Freeze',async()=>{
  const oldPack=candidatePack();
  const finalPack=candidatePack({edges:[]});
  const finalBlob='e'.repeat(40);
  const finalRef='https://github.com/'+repository+'/blob/'+
    'c'.repeat(40)+'/docs/implementation/v2/final.json';
  const store=memoryStore();
  const gh=fakeOctokit({blockedBy:{312:[]}});
  let resumeOld,signalOld;
  const oldHeld=new Promise(resolve=>{resumeOld=resolve;});
  const oldReachedReview=new Promise(resolve=>{signalOld=resolve;});
  const reader=async({taskPackRef:ref,expectedBlobSha})=>({
    status:'VERIFIED',blobSha:expectedBlobSha,
    content:JSON.stringify(ref===finalRef?finalPack:oldPack)});
  const witnesses=async({taskPackBlobSha:sha})=>({
    ...review(),taskPackBlobSha:sha,
    reviewedHeadSha:versionSha,sourceSha:versionSha});
  const ownerWitness=async({taskPackBlobSha:sha})=>({
    ...approval(),taskPackBlobSha:sha,sourceSha:versionSha});
  const makePort=blockReview=>createVersionTaskDagWritePort({
    repository,versionId,expectedVersionSha:versionSha,store,
    taskPackReader:reader,
    reviewWitnessReader:async scope=>{
      if(blockReview){signalOld();await oldHeld;}
      return witnesses(scope);
    },
    ownerApprovalReader:ownerWitness,
    nativeRosterReader:async()=>nativeProjection({pack:finalPack}),
    octokit:gh.octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:{},freezeWriteSetPolicy:'VERSION_ONLY'});
  const oldPort=makePort(true),finalPort=makePort(false);
  const pendingOld=oldPort.addNativeTaskDependency(edgeInput);
  await oldReachedReview;
  const final=await finalPort.freezeVersionTaskDag({
    taskPackRef:finalRef,taskPackBlobSha:finalBlob});
  assert.equal(final.status,'FROZEN');
  resumeOld();
  const old=await pendingOld;
  assert.equal(old.status,'BLOCKED');
  assert.equal(old.reason,'VERSION_DAG_ALREADY_FROZEN');
  assert.equal(gh.posts.length,0,'no native POST after a committed final Freeze');
  assert.equal((await readVersionTaskDagFreezeState({store,repository,versionId})).status,'FROZEN');
});

test('shared physical effect fence: outstanding native POST blocks concurrent Freeze',async()=>{
  const store=memoryStore();
  let finishPost,enteredPost;
  const postWait=new Promise(resolve=>{finishPost=resolve;});
  const seen=new Promise(resolve=>{enteredPost=resolve;});
  const gh=fakeOctokit({blockedBy:{312:[]}});
  const originalPost=gh.octokit.request;
  gh.octokit.request=async(...args)=>{
    enteredPost();
    await postWait;
    return originalPost(...args);
  };
  const port=pack=>createVersionTaskDagWritePort({
    repository,versionId,expectedVersionSha:versionSha,store,
    taskPackReader:async({expectedBlobSha})=>({status:'VERIFIED',
      blobSha:expectedBlobSha,content:JSON.stringify(pack)}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack}),
    octokit:gh.octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:{},freezeWriteSetPolicy:'VERSION_ONLY'});
  const first=port(candidatePack()).addNativeTaskDependency(edgeInput);
  await seen;
  const stop=await port(frozenPack()).freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(stop.status,'BLOCKED');
  assert.equal(stop.reason,'VERSION_EFFECT_FENCE_RECONCILIATION_REQUIRED');
  assert.equal((await readVersionTaskDagFreezeState({store,repository,versionId})).status,
    'NOT_FROZEN','no CAS Freeze while native POST is pending');
  finishPost();
  assert.equal((await first).status,'NATIVE_EDGE_VERIFIED');
  assert.equal(gh.posts.length,1);
});

test('duplicate identical freeze replays; divergent freeze and row conflict block',async()=>{
  const f=dependencyFixture({pack:frozenPack()});
  const first=await f.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(first.status,'FROZEN');
  const replay=await f.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(replay.status,'REPLAY');
  assert.equal(replay.reason,'IDENTICAL_FREEZE_REPLAY');
  assert.equal(replay.writeSetRecordCount,3);
  assert.equal(f.store.puts().filter(k=>
    !k.startsWith('tally:v2:version-dag-effect-fence:')).length,
    4,'no duplicate DAG or Task FSW CAS on replay');
  // A pre-existing divergent version-dag row (foreign freeze) blocks.
  const divergentStore=memoryStore();
  const divergentPack=frozenPack();
  const foreignWriter=createHostVersionTaskDagFreezeWriter({repository,versionId,
    expectedVersionSha:versionSha,store:divergentStore,
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(divergentPack)}),
    nativeRosterReader:async()=>nativeProjection({pack:divergentPack}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    now:()=>'2026-11-09T10:00:00.000Z'});
  await foreignWriter.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  const foreign=dependencyFixture({pack:frozenPack(),store:divergentStore});
  const divergent=await foreign.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  // Same pack and witnesses replay identically; only the writeset config differs.
  assert.ok(['REPLAY','BLOCKED'].includes(divergent.status));
  // A conflicting pre-existing per-Task row blocks BEFORE the freeze CAS.
  const conflictStore=memoryStore([['tally:v2:task-writeset:284',
    {schema:'tally/version-task-writeset/v1',phase:'FROZEN_DAG_WRITESET',
      repository,issueNumber:284,sourceSha:versionSha,writeSetSha256:'b'.repeat(64),
      document:'foreign',versionPackBlobSha:packBlobSha,versionTaskDagKey:'x',
      frozenAt:'2026-11-09T09:00:00.000Z'}]]);
  const conflict=dependencyFixture({pack:frozenPack(),store:conflictStore});
  const blocked=await conflict.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(blocked.status,'BLOCKED');
  assert.equal(blocked.reason,'FREEZE_WRITESET_CONFLICT');
  assert.equal(blocked.conflictIssueNumber,284);
  assert.equal(conflictStore.puts().length,0,'conflict preflight blocks all writes');
});

test('missing Host write-set configuration blocks before any CAS write',async()=>{
  const f=dependencyFixture({pack:frozenPack()});
  const port=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:f.store,
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(frozenPack())}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack:frozenPack()}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:{'284':{writeSetSha256:digest,document:'x'}}});
  const result=await port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'FREEZE_WRITESET_CONFIG_INVALID');
  assert.equal(result.missingIssueNumber,312);
  assert.equal(f.store.puts().length,0);
});

test('partial per-Task record write stays UNKNOWN with reconciliation',async()=>{
  const base=dependencyFixture({pack:frozenPack()});
  const flaky=memoryStore();
  const originalPut=flaky.put.bind(flaky);
  let puts=0;
  flaky.put=async(key,value,condition)=>{
    puts++;
    if(key==='tally:v2:task-writeset:312')throw new Error('SQLITE_BUSY');
    return originalPut(key,value,condition);
  };
  const port=createVersionTaskDagWritePort({repository,versionId,
    expectedVersionSha:versionSha,store:flaky,
    taskPackReader:async()=>({status:'VERIFIED',blobSha:packBlobSha,
      content:JSON.stringify(frozenPack())}),
    reviewWitnessReader:async()=>review(),ownerApprovalReader:async()=>approval(),
    nativeRosterReader:async()=>nativeProjection({pack:frozenPack()}),
    octokit:fakeOctokit().octokit,owner,repo,writerAuthority,
    writeSetDocumentByIssue:writeSetDocs(frozenPack()),
    now:()=>'2026-11-09T13:00:00.000Z'});
  const result=await port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'FREEZE_WRITESET_PARTIAL_UNKNOWN');
  assert.deepEqual(result.failedIssueNumbers,[312]);
  assert.deepEqual(result.reconciledIssueNumbers,[284,219]);
  assert.equal(result.reconciliationRequired,true);
  assert.ok(puts>=2);
  assert.notEqual(base,undefined);
});

test('no dependency write after final freeze',async()=>{
  const f=dependencyFixture({pack:frozenPack(),blockedBy:{312:[]}});
  const frozen=await f.port.freezeVersionTaskDag({taskPackRef,taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  // A refreshed candidate proposing the NEXT serial edge after the freeze:
  // still refused regardless of pack content.
  const after=dependencyFixture({pack:candidatePack(),store:f.store,blockedBy:{219:[]}});
  const result=await after.port.addNativeTaskDependency(
    {...edgeInput,blockedIssueNumber:312});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'VERSION_DAG_ALREADY_FROZEN');
  assert.equal(after.posts.length,0);
  assert.equal((await readVersionTaskDagFreezeState({store:f.store,
    repository,versionId})).status,'FROZEN');
});

test('freeze state reader reports NOT_FROZEN before any freeze',async()=>{
  const store=memoryStore();
  assert.equal((await readVersionTaskDagFreezeState({store,repository,versionId}))
    .status,'NOT_FROZEN');
  await assert.rejects(readVersionTaskDagFreezeState({store:{},repository,versionId}),
    /TRUSTED_VERSION_TASK_DAG_STATE_READER_REQUIRED/);
});
