import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {createTaskReadyTransitionPort}
  from '../src/task-ready-transition-port.mjs';

const repository='kaicreator-mm/tally';
const owner='kaicreator-mm';
const repo='tally';
const targetIssue=377;
const sourceSha='7'.repeat(40);
const writerAuthority={mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
  repository,writerId:'tally-host:win32:i377:r95:1',githubWriterLogin:'kaicreator-mm'};
const issueUrl=n=>'https://github.com/'+repository+'/issues/'+n;

/** CAS store with the trusted semantics (mustBeAbsent/mustMatchRevision). */
function memoryStore(){
  const rows=new Map();
  return {
    async get(key){const row=rows.get(key);return row?structuredClone(row):null;},
    async put(key,value,{mustBeAbsent,mustMatchRevision}={}){
      const existing=rows.get(key);
      if(mustBeAbsent&&existing)throw new Error('CAS_EXISTS');
      if(mustMatchRevision!==undefined&&existing?.revision!==mustMatchRevision)
        throw new Error('CAS_REVISION_CONFLICT');
      const revision=(existing?.revision??0)+1;
      rows.set(key,{revision,value:structuredClone(value)});
      return {revision};
    },
    _rows:rows,
  };
}

/** Fake Octokit: paginated blocked_by, issue get/update with counters, and a
 * controllable update behavior for lost-ACK/concurrency scenarios. */
function fakeOctokit({labels=['type:task','state:planned'],blockers=[],
  updatedAt='2026-11-19T00:00:00Z',updateBehavior=null,
  branchSha=sourceSha,body=null}={}){
  const state={labels:[...labels],updatedAt,body,updateCalls:0,updatePayloads:[]};
  const api={
    state,
    rest:{
      issues:{
        async get({issue_number}){
          assert.equal(issue_number,targetIssue);
          return {data:{id:9001,number:targetIssue,state:'open',
            updated_at:state.updatedAt,body:state.body,pull_request:undefined,
            repository_url:'https://api.github.com/repos/'+owner+'/'+repo,
            labels:state.labels.map(name=>({name}))}};
        },
        async update({labels:next}){
          state.updateCalls+=1;
          state.updatePayloads.push([...next]);
          if(updateBehavior)await updateBehavior();
          state.labels=[...next];
          state.updatedAt='2026-11-19T00:05:00Z';
          return {data:{}};
        },
      },
      repos:{async getBranch({branch}){assert.equal(branch,'version/v2');
        return {data:{commit:{sha:branchSha}}};}},
    },
    paginate:{async*iterator(route,params){
      assert.equal(route,'GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by');
      yield {data:blockers.map(b=>({id:b.id,number:b.number,state:b.state,
        html_url:issueUrl(b.number)}))};
    }},
  };
  return api;
}

function makePort({store=memoryStore(),...rest}={}){
  return createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit:fakeOctokit({...rest}),
    store,writerAuthority});
}

test('constructor refuses incomplete trusted Host configuration',()=>{
  assert.throws(()=>createTaskReadyTransitionPort({repository:'o/r',owner:'o',repo:'r',
    targetIssue:1,expectedSourceSha:sourceSha,store:memoryStore(),writerAuthority}),TypeError);
  assert.throws(()=>createTaskReadyTransitionPort({repository,owner,repo,targetIssue:0,
    expectedSourceSha:sourceSha,octokit:fakeOctokit(),store:memoryStore(),writerAuthority}),TypeError);
  assert.throws(()=>createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:'nothex',octokit:fakeOctokit(),store:memoryStore(),writerAuthority}),TypeError);
  assert.throws(()=>createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,octokit:fakeOctokit(),store:memoryStore(),
    writerAuthority:{...writerAuthority,exclusiveHostControl:false}}),TypeError);
});

test('POS planned→ready performs exactly one PATCH and verifies by readback',async()=>{
  const octokit=fakeOctokit();
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'READY_TRANSITION_VERIFIED');
  assert.equal(result.reason,'OFFICIAL_GITHUB_LABEL_READBACK_VERIFIED');
  assert.equal(octokit.state.updateCalls,1,'exactly one PATCH');
  assert.deepEqual(octokit.state.updatePayloads[0],['type:task','state:ready']);
  assert.equal(result.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(result.readyAt,'2026-11-19T00:05:00Z');
  assert.equal(result.writePerformed,true);
  assert.equal(result.authorized,false,'readback alone grants no authority');
  assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);
});

test('NEG open blocked_by refuses before any PATCH',async()=>{
  const octokit=fakeOctokit({blockers:[{id:5,number:211,state:'closed'},
    {id:7,number:300,state:'open'}]});
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'NATIVE_ISSUE_DEPENDENCIES_BLOCKED');
  assert.deepEqual(result.openBlockerNumbers,[300]);
  assert.equal(octokit.state.updateCalls,0);
});

test('NEG closed-only blockers are admissible history and the transition proceeds',async()=>{
  const octokit=fakeOctokit({blockers:[{id:5,number:211,state:'closed'}]});
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'READY_TRANSITION_VERIFIED');
  assert.equal(octokit.state.updateCalls,1);
});

test('NEG already-ready label refuses without a PATCH',async()=>{
  const octokit=fakeOctokit({labels:['type:task','state:ready']});
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'ALREADY_READY');
  assert.equal(octokit.state.updateCalls,0);
});

test('NEG label drift between preflight and fenced re-read refuses (detectable ABA)',async()=>{
  const octokit=fakeOctokit();
  // After the port's first preflight read, an outside actor flips the label;
  // the in-fence re-read must observe the drift and refuse.
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const originalGet=octokit.rest.issues.get.bind(octokit);
  let reads=0;
  octokit.rest.issues.get=async params=>{
    reads+=1;
    if(reads===2){ // the in-fence re-read
      octokit.state.labels=['type:task','state:ready'];
      octokit.state.updatedAt='2026-11-19T00:03:00Z';
    }
    return originalGet(params);
  };
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'ISSUE_STALE_VS_PREFLIGHT');
  assert.equal(octokit.state.updateCalls,0);
});

test('NEG stale updated_at drift refuses',async()=>{
  const octokit=fakeOctokit();
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const originalGet=octokit.rest.issues.get.bind(octokit);
  let reads=0;
  octokit.rest.issues.get=async params=>{
    reads+=1;
    if(reads===2)octokit.state.updatedAt='2026-11-19T00:04:00Z'; // concurrent touch
    return originalGet(params);
  };
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'ISSUE_STALE_VS_PREFLIGHT');
  assert.equal(octokit.state.updateCalls,0);
});

test('NEG source epoch drift on the live branch refuses',async()=>{
  const octokit=fakeOctokit({branchSha:'a'.repeat(40)});
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'SOURCE_EPOCH_NOT_CURRENT');
  assert.equal(octokit.state.updateCalls,0);
});

test('NEG unauthorized claim or FSW rows in the trusted store refuse',async()=>{
  for(const [key,value] of [
    ['tally:v2:claim-admission:'+createHash('sha256')
      .update(repository+'#'+targetIssue+':builder:__default__').digest('hex'),
      {phase:'PUBLISHED_VERIFIED'}],
    ['tally:v2:task-writeset:'+targetIssue,{status:'FROZEN_DAG_WRITESET'}],
  ]){
    const store=memoryStore();
    await store.put(key,value,{mustBeAbsent:true});
    const octokit=fakeOctokit();
    const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
      expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store,writerAuthority});
    const result=await port.transitionToReady();
    assert.equal(result.status,'BLOCKED');
    assert.ok(['UNAUTHORIZED_CLAIM_ROW_PRESENT','UNAUTHORIZED_FSW_ROW_PRESENT'].includes(result.reason),
      'reason='+result.reason);
    assert.equal(octokit.state.updateCalls,0);
  }
});

test('NEG lost PATCH ACK yields exactly one PATCH, UNKNOWN, and a HELD fence',async()=>{
  const octokit=fakeOctokit({updateBehavior:()=>{throw new Error('socket hang up');}});
  const store=memoryStore();
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store,writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'GITHUB_UPDATE_ACK_LOST');
  assert.equal(result.reconciliationRequired,true);
  assert.equal(octokit.state.updateCalls,1,'no blind retry: exactly one PATCH attempt');
  const fenceKey=[...store._rows.keys()].find(k=>k.startsWith('tally:v2:task-ready-fence:'));
  const fence=await store.get(fenceKey);
  assert.equal(fence.value.phase,'HELD','crash safety: fence stays HELD');
  // A retry must refuse on the held fence, never issue a second PATCH.
  const retry=await port.transitionToReady();
  assert.equal(retry.status,'BLOCKED');
  assert.equal(retry.reason,'TRANSITION_FENCE_CONTENTION');
  assert.equal(octokit.state.updateCalls,1,'still exactly one PATCH');
  // Read-only reconciliation classifies without PATCHing.
  octokit.state.labels=['type:task','state:ready']; // suppose the lost ACK had applied
  const rec=await port.reconcileTransitionState();
  assert.equal(rec.status,'READY_RECONCILED');
  assert.equal(rec.githubEffect,'NOT_RUN');
  assert.equal(octokit.state.updateCalls,1);
});

test('NEG concurrent issuer loses on the fence with zero PATCHes',async()=>{
  const store=memoryStore();
  let releaseFirstPatch;
  const firstGate=new Promise(res=>{releaseFirstPatch=res;});
  const octokitA=fakeOctokit({updateBehavior:async()=>{await firstGate;}});
  const octokitB=fakeOctokit();
  const portA=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit:octokitA,store,writerAuthority});
  const portB=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit:octokitB,store,writerAuthority});
  const first=portA.transitionToReady();
  await new Promise(res=>setTimeout(res,25)); // let A reserve the fence
  const second=await portB.transitionToReady();
  assert.equal(second.status,'BLOCKED');
  assert.equal(second.reason,'TRANSITION_FENCE_CONTENTION');
  assert.equal(octokitB.state.updateCalls,0,'loser never PATCHes');
  releaseFirstPatch();
  const firstResult=await first;
  assert.equal(firstResult.status,'READY_TRANSITION_VERIFIED');
  assert.equal(octokitA.state.updateCalls,1);
  assert.equal(octokitB.state.updateCalls,0);
});

test('NEG provider error (422) blocks before postread with no unknown',async()=>{
  const octokit=fakeOctokit({updateBehavior:()=>{
    const e=new Error('Validation Failed');e.status=422;throw e;}});
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'GITHUB_VALIDATION_FAILED');
});

test('NEG non-planned prestate labels refuse',async()=>{
  const octokit=fakeOctokit({labels:['type:task','state:planned','state:in-progress']});
  const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
  const result=await port.transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'PRESTATE_LABELS_INVALID');
  assert.equal(octokit.state.updateCalls,0);
});

test('reconcileTransitionState never patches and classifies both states',async()=>{
  for(const [labels,expected] of [
    [['type:task','state:planned'],'STILL_PLANNED'],
    [['type:task','state:ready'],'READY_RECONCILED'],
  ]){
    const octokit=fakeOctokit({labels});
    const port=createTaskReadyTransitionPort({repository,owner,repo,targetIssue,
      expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit,store:memoryStore(),writerAuthority});
    const result=await port.reconcileTransitionState();
    assert.equal(result.status,expected);
    assert.equal(octokit.state.updateCalls,0,'reconciliation never PATCHes');
    assert.equal(result.taskPass,false,'readback never authorizes an Outcome');
  }
});

const draftSourceIdentity={repository,versionId:'version/v2',sourceSha};
const draftBody=(sha=sourceSha)=>[
  'Tally development task draft — not yet admitted to Task DAG.',
  'Repository: '+repository,'Selected branch: version/v2',
  'Observed source SHA: '+sha,'Request: tally:draft:request-123','',
  'Task text','','Status: UNQUALIFIED_DRAFT. No Agent, Claim or Release authority.'
].join('\n');
function sourceBoundPort(octokit,store=memoryStore()){
  return createTaskReadyTransitionPort({
    repository,owner,repo,targetIssue,expectedSourceSha:sourceSha,
    versionBranch:'version/v2',octokit,store,writerAuthority,draftSourceIdentity});
}
test('P1 Issue body edited after admission before native port entry refuses without PATCH',async()=>{
  const octokit=fakeOctokit({body:draftBody('b'.repeat(40))});
  const result=await sourceBoundPort(octokit).transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'TASK_DRAFT_SOURCE_IDENTITY_DRIFT');
  assert.equal(octokit.state.updateCalls,0);
});
test('P1 in-fence body-only drift refuses even if updated_at has same timestamp',async()=>{
  const octokit=fakeOctokit({body:draftBody()});
  const original=octokit.rest.issues.get.bind(octokit);
  let count=0;
  octokit.rest.issues.get=async p=>{
    if(++count===2){
      octokit.state.body=draftBody('b'.repeat(40));
      // Deliberately unchanged updatedAt; body proof must detect drift.
    }
    return original(p);
  };
  const result=await sourceBoundPort(octokit).transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'TASK_DRAFT_SOURCE_IDENTITY_DRIFT');
  assert.equal(octokit.state.updateCalls,0);
});
test('P1 body-only description change before PATCH refuses even if source matches',async()=>{
  const octokit=fakeOctokit({body:draftBody()});
  const original=octokit.rest.issues.get.bind(octokit);
  let count=0;
  octokit.rest.issues.get=async p=>{
    if(++count===2)octokit.state.body=draftBody().replace('Task text','Changed text');
    return original(p);
  };
  const result=await sourceBoundPort(octokit).transitionToReady();
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'ISSUE_STALE_VS_PREFLIGHT');
  assert.equal(octokit.state.updateCalls,0);
});
test('P2 exact readyAt survives CAS store serialization and receipt',async()=>{
  const octokit=fakeOctokit({body:draftBody()});
  const store=memoryStore();
  const result=await sourceBoundPort(octokit,store).transitionToReady();
  assert.equal(result.status,'READY_TRANSITION_VERIFIED');
  assert.equal(result.readyAt,'2026-11-19T00:05:00Z');
  const fence=[...store._rows.entries()].find(([key])=>
    key.startsWith('tally:v2:task-ready-fence:'))?.[1]?.value;
  assert.equal(fence.phase,'RELEASED');
  assert.equal(fence.readyAt,result.readyAt);
  assert.equal(JSON.parse(JSON.stringify(fence)).readyAt,result.readyAt);
  assert.equal(octokit.state.updateCalls,1);
});
test('post-PATCH body mismatch leaves HELD/UNKNOWN and never claims verified',async()=>{
  const octokit=fakeOctokit({body:draftBody(),updateBehavior:()=>{
    octokit.state.body=draftBody('b'.repeat(40));
  }});
  const store=memoryStore();
  const result=await sourceBoundPort(octokit,store).transitionToReady();
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'POSTREAD_DRAFT_SOURCE_DRIFT');
  assert.equal(octokit.state.updateCalls,1);
  const fence=[...store._rows.values()][0].value;
  assert.equal(fence.phase,'HELD');
});
