import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {createTaskOutcomePublisher}
  from '../src/task-outcome-publisher.mjs';

const repository='kaicreator-mm/tally';
const owner='kaicreator-mm';
const repo='tally';
const targetIssue=377;
const sourceSha='c'.repeat(40);
const writerAuthority={mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
  repository,writerId:'tally-host:win32:i377:r95:1',githubWriterLogin:'kaicreator-mm'};
const issueUrl=n=>'https://github.com/'+repository+'/issues/'+n;
const sha=s=>createHash('sha256').update(JSON.stringify(s),'utf8').digest('hex');
const readyLabels=['type:task','state:ready'];

function memoryStore(){
  const rows=new Map();
  return {
    async get(key){const row=rows.get(key);return row?structuredClone(row):null;},
    // Same revision semantics as the real v8_host_docs store: a
    // mustBeAbsent insert lands at revision 0; updates bump by one.
    async put(key,value,{mustBeAbsent,mustMatchRevision}={}){
      const existing=rows.get(key);
      if(mustBeAbsent&&existing)throw new Error('CAS_EXISTS');
      if(mustMatchRevision!==undefined&&existing?.revision!==mustMatchRevision)
        throw new Error('CAS_REVISION_CONFLICT');
      const revision=mustBeAbsent?0:(existing?.revision??-1)+1;
      rows.set(key,{revision,value:structuredClone(value)});
      return {revision};
    },
    _rows:rows,
  };
}

function fakeOctokit({labels=readyLabels,state='open',branchSha=sourceSha}={}){
  return {
    rest:{
      issues:{async get(){return {data:{id:9001,number:targetIssue,state,
        updated_at:'2026-11-19T00:00:00Z',pull_request:undefined,
        repository_url:'https://api.github.com/repos/'+owner+'/'+repo,
        labels:labels.map(name=>({name}))}};}},
      repos:{async getBranch(){return {data:{commit:{sha:branchSha}}};}},
    },
  };
}

/** The complete S1→S5 chain the publisher consumes. */
function seedChain(){
  const fswStore=memoryStore(),claimStore=memoryStore(),joinStore=memoryStore();
  const fswValue={schema:'tally/version-task-writeset/v1',phase:'FROZEN_DAG_WRITESET',
    document:'[]'};
  const fswKey='tally:v2:task-writeset:377';
  fswStore.put(fswKey,fswValue,{mustBeAbsent:true});
  const validatorClaim={casKey:'tally:v2:claim-admission:v',revision:0,commentId:111,
    commentRef:issueUrl(targetIssue)+'#issuecomment-111',
    commentBodySha256:'a'.repeat(64),actorRole:'validator',operatorKind:'other',
    operatorId:'zcode:i377:r95:validator:1',
    dispatchId:'dispatch:377:acceptance-validator:host:20261119:1',generation:1};
  const reviewerClaim={casKey:'tally:v2:claim-admission:r',revision:0,commentId:222,
    commentRef:issueUrl(targetIssue)+'#issuecomment-222',
    commentBodySha256:'b'.repeat(64),actorRole:'reviewer',operatorKind:'other',
    operatorId:'kimi:i377:r95:reviewer:1',
    dispatchId:'dispatch:377:acceptance-reviewer:host:20261119:1',generation:1};
  claimStore.put(validatorClaim.casKey,{phase:'PUBLISHED_VERIFIED',
    commentId:111,commentBodySha256:validatorClaim.commentBodySha256,
    operatorId:validatorClaim.operatorId},{mustBeAbsent:true});
  claimStore.put(reviewerClaim.casKey,{phase:'PUBLISHED_VERIFIED',
    commentId:222,commentBodySha256:reviewerClaim.commentBodySha256,
    operatorId:reviewerClaim.operatorId},{mustBeAbsent:true});
  const joinValue={schema:'tally/version-task-role-join/v1',
    validatorClaim:{casKey:validatorClaim.casKey},
    reviewerClaim:{casKey:reviewerClaim.casKey}};
  const joinKey='tally:v2:task-role-join:377:'+sourceSha;
  joinStore.put(joinKey,joinValue,{mustBeAbsent:true});
  const input={
    fsw:{key:fswKey,revision:0,valueSha256:sha(fswValue)},
    join:{key:joinKey,revision:0,valueSha256:sha(joinValue)},
    validatorClaim,reviewerClaim,
    validatorEvidence:{evidenceRef:issueUrl(targetIssue)+'#issuecomment-333'},
    decision:{decision:'ACCEPT',decisionRef:issueUrl(targetIssue)+'#issuecomment-444',
      reviewerOperatorId:'kimi:i377:r95:reviewer:1',decidedAt:'2026-11-19T00:00:00Z'},
    sourcePack:{packId:'I377-R93-E0',blob:'7'.repeat(40)},
    ci:{receiptsRef:issueUrl(targetIssue)+'#issuecomment-555'},
    hostFence:{fenceKey:'tally:v2:task-ready-fence:'+'d'.repeat(64).slice(0,64),
      fenceRevision:1,writerId:'tally-host:win32:i377:r95:1'},
  };
  return {fswStore,claimStore,joinStore,input,outcomeKey:'tally:v2:task-outcome:377:'+sha(joinValue)};
}

function makePublisher({store=memoryStore(),...rest}={}){
  return createTaskOutcomePublisher({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,versionBranch:'version/v2',octokit:fakeOctokit({...rest}),
    store,writerAuthority,...rest.stores});
}

test('constructor refuses incomplete trusted Host configuration',()=>{
  assert.throws(()=>createTaskOutcomePublisher({repository:'o/r',owner:'o',repo:'r',
    targetIssue:1,expectedSourceSha:sourceSha,store:memoryStore(),fswStore:memoryStore(),
    claimStore:memoryStore(),joinStore:memoryStore(),writerAuthority}),TypeError);
  assert.throws(()=>createTaskOutcomePublisher({repository,owner,repo,targetIssue,
    expectedSourceSha:sourceSha,octokit:fakeOctokit(),store:memoryStore(),
    fswStore:memoryStore(),claimStore:memoryStore(),joinStore:memoryStore(),
    writerAuthority:{...writerAuthority,exclusiveHostControl:false}}),TypeError);
});

test('POS qualified Outcome publishes once with full provenance and reconciles',async()=>{
  const chain=seedChain();
  const store=memoryStore();
  const publisher=makePublisher({store,stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
  const result=await publisher.publishQualifiedOutcome(chain.input);
  assert.equal(result.status,'QUALIFIED_OUTCOME_PUBLISHED');
  assert.equal(result.reason,'HOST_CAS_READBACK_AND_NATIVE_RECONCILIATION_VERIFIED');
  assert.equal(result.taskPass,true,'S6 success is the only Task PASS source');
  assert.equal(result.releasePass,false,'Release stays a separate authorization');
  assert.equal(result.publication,'PUBLISHED');
  const row=await store.get(result.outcomeKey);
  assert.equal(row.value.phase,'QUALIFIED_OUTCOME_PUBLISHED');
  assert.equal(row.value.fsw.valueSha256,chain.input.fsw.valueSha256);
  assert.equal(row.value.reviewDecision.decision,'ACCEPT');
  assert.deepEqual([...row.value.distinctFrom],['DISPATCH_CLAIMED_COMMENT_ACK','NATIVE_ISSUE_CLOSED','NATIVE_DONE_LABEL','JOURNEY_PROJECTION','PR394_SOURCE_REPAIR_GREEN'],'all five pack-literal markers');
  const rec=await publisher.reconcileTaskOutcome({joinValueSha256:chain.input.join.valueSha256});
  assert.equal(rec.status,'OUTCOME_PRESENT');
  assert.equal(rec.taskPass,false,'reconciliation classifies, never authorizes');
});

test('NEG REWORK and DEFER decisions never publish and never pass',async()=>{
  for(const decision of ['REWORK','DEFER']){
    const chain=seedChain();
    const store=memoryStore(); // the store the publisher would mint into
    const publisher=makePublisher({store,stores:{
      fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
    const result=await publisher.publishQualifiedOutcome(
      {...chain.input,decision:{...chain.input.decision,decision}});
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'DECISION_'+decision+'_NOT_TASK_PASS');
    assert.equal(result.taskPass,false);
    assert.equal(result.publication,'HOLD');
    const outcomeRows=[...store._rows.keys()].filter(k=>k.startsWith('tally:v2:task-outcome:'));
    assert.equal(outcomeRows.length,0,'REWORK/DEFER must mint zero outcome rows');
  }
});

test('NEG a reviewer claim binding is not an independent ACCEPT decision',async()=>{
  const chain=seedChain();
  const publisher=makePublisher({stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
  const result=await publisher.publishQualifiedOutcome({...chain.input,decision:null});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'MISSING_EVIDENCE');
});

test('NEG missing validator evidence / CI / fence refuses before writes',async()=>{
  for(const field of ['validatorEvidence','ci','hostFence']){
    const chain=seedChain();
    const publisher=makePublisher({stores:{
      fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
    const input={...chain.input};
    delete input[field];
    const result=await publisher.publishQualifiedOutcome(input);
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'MISSING_EVIDENCE','field='+field+' reason='+result.reason);
  }
});

test('NEG alias validator/reviewer operators refuse',async()=>{
  const chain=seedChain();
  chain.input.reviewerClaim={...chain.input.reviewerClaim,
    operatorId:chain.input.validatorClaim.operatorId};
  const publisher=makePublisher({stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
  const result=await publisher.publishQualifiedOutcome(chain.input);
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'REVOKED_ROLE');
});

test('NEG stale source epoch refuses',async()=>{
  const chain=seedChain();
  const publisher=makePublisher({stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore},
    branchSha:'a'.repeat(40)});
  const result=await publisher.publishQualifiedOutcome(chain.input);
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'STALE_SOURCE');
  assert.equal(result.detail,'source epoch drift');
});

test('NEG drifted FSW/claim rows refuse (fail closed, never healed)',async()=>{
  {
    const chain=seedChain();
    chain.input.fsw={...chain.input.fsw,valueSha256:'e'.repeat(64)};
    const publisher=makePublisher({stores:{
      fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
    const result=await publisher.publishQualifiedOutcome(chain.input);
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'STALE_SOURCE');
  }
  {
    const chain=seedChain();
    chain.claimStore.put(chain.input.reviewerClaim.casKey,
      {phase:'PENDING',commentId:222,
       commentBodySha256:chain.input.reviewerClaim.commentBodySha256,
       operatorId:chain.input.reviewerClaim.operatorId},{mustBeAbsent:false});
    const publisher=makePublisher({stores:{
      fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
    const result=await publisher.publishQualifiedOutcome(chain.input);
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'REVOKED_ROLE');
  }
});

test('NEG race double Outcome create: one winner, loser gets no second grant',async()=>{
  const chain=seedChain();
  const shared=memoryStore();
  let releaseFirst;
  const gate=new Promise(res=>{releaseFirst=res;});
  // Publisher A's outcome CAS put is delayed before its INSERT, so B inserts
  // first on the SAME shared store; A must then lose on CAS_EXISTS.
  const slowStore={
    async get(key){return shared.get(key);},
    async put(key,value,opts){
      if(key.startsWith('tally:v2:task-outcome:'))await gate;
      return shared.put(key,value,opts);
    },
  };
  const publisherA=makePublisher({store:slowStore,stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
  const publisherB=makePublisher({store:shared,stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
  const first=publisherA.publishQualifiedOutcome(chain.input);
  await new Promise(res=>setTimeout(res,25)); // A is parked inside its put
  const second=await publisherB.publishQualifiedOutcome(chain.input);
  assert.equal(second.status,'QUALIFIED_OUTCOME_PUBLISHED','shared-store inserter wins');
  releaseFirst();
  const firstResult=await first;
  assert.equal(firstResult.status,'BLOCKED');
  assert.equal(firstResult.reason,'OUTCOME_ALREADY_PUBLISHED');
  assert.equal(firstResult.taskPass,false,'loser never infers authorization');
  const grants=[...shared._rows.keys()].filter(k=>k.startsWith('tally:v2:task-outcome:'));
  assert.equal(grants.length,1,'exactly one durable success row');
});

test('NEG lost CAS ACK yields UNKNOWN and no blind second publish',async()=>{
  const chain=seedChain();
  const store=memoryStore();
  const octokit=fakeOctokit({labels:readyLabels});
  const publisher=makePublisher({store,octokit,stores:{
    fswStore:chain.fswStore,claimStore:chain.claimStore,joinStore:chain.joinStore}});
  const originalPut=store.put.bind(store);
  store.put=async(...args)=>{
    await originalPut(...args); // row commits…
    throw new Error('connection reset'); // …then the ACK is lost
  };
  const result=await publisher.publishQualifiedOutcome(chain.input);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'OUTCOME_CAS_RESERVATION_UNKNOWN',
    'committed-then-thrown is an ambiguous ACK, not verified contention');
  const outcomeKeys=[...store._rows.keys()].filter(k=>k.startsWith('tally:v2:task-outcome:'));
  assert.equal(outcomeKeys.length,1,'the physical row committed exactly once');
  const rec=await publisher.reconcileTaskOutcome(
    {joinValueSha256:chain.input.join.valueSha256});
  assert.equal(rec.status,'OUTCOME_PRESENT','reconciliation finds the committed row');
  assert.equal(rec.taskPass,false,'classification never grants');
});
