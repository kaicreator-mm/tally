import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {resolveSelectedAgentClaimPins} from './selected-claim-activation.mjs';
const sha='a'.repeat(40),repository='org/test',issue=27;
const settings={repository,versionId:'main',expectedSha:sha};
const env={
  TALLY_V2_CLAIM_DISPATCH:'host-mounted',
  TALLY_V2_AGENT_DISPATCH:'host-mounted',
  TALLY_V2_CLAIM_REPOSITORY:repository,
  TALLY_V2_CLAIM_VERSION_REF:'main',
  TALLY_V2_CLAIM_SOURCE_SHA:sha,
  TALLY_V2_AGENT_CLAIM_SOURCE_SHA:sha,
  TALLY_LOCAL_ISSUE_NUMBER:String(issue),
  TALLY_V2_CLAIM_ISSUE_NUMBER:String(issue),
  TALLY_V2_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
  TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
  TALLY_V2_AGENT_CLAIM_DISPATCH_ID:'dispatch:27:1',
  TALLY_V2_AGENT_CLAIM_OPERATOR_ID:'agent:27',
  TALLY_V2_AGENT_CLAIM_OPERATOR_KIND:'other',
  TALLY_V2_AGENT_CLAIM_WRITER_ID:'host:27'
};
const ref='https://github.com/'+repository+'/issues/'+issue;
const row={revision:1,value:{
  phase:'PUBLISHED_VERIFIED',protectedClaimKey:repository+'#27:builder:__default__',
  repository,issueRef:ref,sourceSha:sha,actorRole:'builder',
  dispatchId:'dispatch:27:1',operatorId:'agent:27',operatorKind:'other',
  writerId:'host:27',generation:1,commentId:42,
  commentRef:ref+'#issuecomment-42'}};
const registryKey='tally:v2:claim-admission:'+
  createHash('sha256').update(repository+'#27:builder:__default__').digest('hex');
function withRegistry(value=row){
  return async()=>({get:async key=>{assert.equal(key,registryKey);return value;},
    close:async()=>{}});
}
test('verified Host CAS row derives only the comment pin, never Agent authority',async()=>{
  const actual=await resolveSelectedAgentClaimPins({
    env,settings,openStore:withRegistry()});
  assert.equal(actual.status,'HOST_RECEIPT_BOUND_PENDING_K');
  assert.equal(actual.agentAuthorized,false);
  assert.equal(actual.env.TALLY_V2_AGENT_CLAIM_COMMENT_ID,'42');
  assert.equal(actual.env.TALLY_V2_AGENT_CLAIM_SOURCE_SHA,sha);
});
test('no publication or pending reservation cannot activate',async()=>{
  for(const value of [null,{...row,value:{...row.value,phase:'PENDING_GITHUB_PUBLISH'}}]){
    const actual=await resolveSelectedAgentClaimPins({env,settings,
      openStore:withRegistry(value)});
    assert.equal(actual.status,'BLOCKED');
    assert.equal(actual.agentAuthorized,false);
  }
});
test('foreign row, source, Issue or comment cannot activate',async()=>{
  for(const value of [
    {...row,value:{...row.value,repository:'org/foreign'}},
    {...row,value:{...row.value,sourceSha:'b'.repeat(40)}},
    {...row,value:{...row.value,commentRef:ref+'#issuecomment-43'}},
  ]){
    const out=await resolveSelectedAgentClaimPins({env,settings,
      openStore:withRegistry(value)});
    assert.equal(out.status,'BLOCKED');
  }
  const foreignEnv={...env,TALLY_V2_CLAIM_REPOSITORY:'org/other'};
  const out=await resolveSelectedAgentClaimPins({env:foreignEnv,settings,
    openStore:async()=>{throw Error('must not access')}})
  assert.equal(out.reason,'AGENT_CLAIM_SOURCE_OR_REGISTRY_MISMATCH');
});
test('stale explicit Host comment pin cannot be overwritten',async()=>{
  const out=await resolveSelectedAgentClaimPins({
    env:{...env,TALLY_V2_AGENT_CLAIM_COMMENT_ID:'41'},settings,
    openStore:withRegistry()});
  assert.equal(out.reason,'AGENT_CLAIM_COMMENT_PIN_MISMATCH');
});

test('new-project FSW digest is derived from exact frozen CAS, no UI pin needed',async()=>{
  const document=JSON.stringify(['src/health.mjs']);
  const writeSetSha256=createHash('sha256').update(document).digest('hex');
  const newEnv={...env,TALLY_V2_WRITESET_FREEZE:'host-mounted',
    T06M_TARGET_REPOSITORY:repository,
    T06M_TARGET_VERSION_REF:'main',T06M_TARGET_ISSUE_NUMBER:'27',
    T06M_TARGET_CANDIDATE_PATH:'src/health.mjs'};
  const frozen={revision:0,value:{phase:'FROZEN_DAG_WRITESET',repository,
    issueNumber:27,sourceSha:sha,writeSetSha256,document}};
  const get=async key=>key===registryKey?row:
    key==='tally:v2:task-writeset:27'?frozen:null;
  const openStore=async()=>({get,close:async()=>{}});
  const good=await resolveSelectedAgentClaimPins({env:newEnv,settings,openStore});
  assert.equal(good.status,'HOST_RECEIPT_BOUND_PENDING_K');
  assert.equal(good.env.TALLY_V2_AGENT_CLAIM_WRITESET_SHA256,writeSetSha256);
  assert.equal(good.env.TALLY_LOCAL_WRITESET_SHA256,writeSetSha256);
  assert.equal(good.agentAuthorized,false);
  const denied=await resolveSelectedAgentClaimPins({settings,openStore,
    env:{...newEnv,TALLY_LOCAL_WRITESET_SHA256:'f'.repeat(64)}});
  assert.equal(denied.reason,'AGENT_WRITESET_PIN_MISMATCH');
});
test('missing, foreign or altered FSW cannot activate new-project K',async()=>{
  const document=JSON.stringify(['src/health.mjs']);
  const digest=createHash('sha256').update(document).digest('hex');
  const newEnv={...env,TALLY_V2_WRITESET_FREEZE:'host-mounted',
    T06M_TARGET_REPOSITORY:repository,T06M_TARGET_VERSION_REF:'main',
    T06M_TARGET_ISSUE_NUMBER:'27',T06M_TARGET_CANDIDATE_PATH:'src/health.mjs'};
  for(const candidate of [null,{phase:'FROZEN_DAG_WRITESET',
    repository:'org/foreign',issueNumber:27,sourceSha:sha,
    document,writeSetSha256:digest},{phase:'FROZEN_DAG_WRITESET',
    repository,issueNumber:27,sourceSha:sha,
    document,writeSetSha256:'f'.repeat(64)}]){
    const openStore=async()=>({get:async key=>key===registryKey?row:
      candidate?{revision:0,value:candidate}:null,close:async()=>{}});
    const denied=await resolveSelectedAgentClaimPins({
      env:newEnv,settings,openStore});
    assert.equal(denied.status,'BLOCKED');
    assert.equal(denied.reason,'AGENT_WRITESET_NOT_FROZEN_VERIFIED');
  }
});
