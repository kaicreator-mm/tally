import test from 'node:test';
import assert from 'node:assert/strict';
import {readProjectTaskAdmission} from './task-admission-read.mjs';
const sha='a'.repeat(40),issueNumber=18;
const draft=(nativeWorkflow,overrides={})=>({
  issueNumber,issueUrl:'https://github.com/org/test/issues/18',
  githubIssueState:'open',sourceSha:sha,nativeWorkflow,...overrides
});
const args={repository:'org/test',versionId:'main',sourceSha:sha,
  token:'github_pat_TEST_READ',issueNumber};
const mocked=entry=>async()=>({connected:true,coverage:'COMPLETE_SINGLE_PAGE',
  drafts:[entry]});
test('canonical planned Issue needs trusted Host transition, not Agent',async()=>{
  const result=await readProjectTaskAdmission({...args,
    readDrafts:mocked(draft('PLANNED'))});
  assert.equal(result.connected,true);
  assert.equal(result.stage,'PLANNED');
  assert.equal(result.reason,'TRUSTED_HOST_READY_TRANSITION_REQUIRED');
  assert.equal(result.authorized,false);
  assert.equal(result.agentAuthorized,false);
});
test('ready Issue reaches existing native preflight without CAS admission',async()=>{
  let calls=0;
  const result=await readProjectTaskAdmission({...args,
    readDrafts:mocked(draft('READY')),
    readNativePreflight:async input=>{
      calls++;assert.deepEqual({owner:input.owner,repo:input.repo,
        issue:input.issueNumber,sha:input.expectedSourceSha,role:input.actorRole},
        {owner:'org',repo:'test',issue:18,sha,role:'builder'});
      return {status:'NATIVE_SOURCE_PREFLIGHT_READY',
        sourceQualified:true,nativePreflightReady:true,
        sourceSha:sha,issueRef:'https://github.com/org/test/issues/18',
        admitted:false,authorized:false};
    }});
  assert.equal(calls,1);
  assert.equal(result.stage,'NATIVE_PREFLIGHT_READY');
  assert.equal(result.admitted,false);
  assert.equal(result.agentAuthorized,false);
});
test('wrong Issue or SHA cannot reach native preflight',async()=>{
  let calls=0;
  const readNativePreflight=async()=>{calls++;return {};};
  for(const readDrafts of [mocked(draft('READY',{sourceSha:'b'.repeat(40)})),
    async()=>({connected:true,coverage:'COMPLETE_SINGLE_PAGE',drafts:[]})]){
    const result=await readProjectTaskAdmission({...args,
      readDrafts,readNativePreflight});
    assert.notEqual(result.stage,'NATIVE_PREFLIGHT_READY');
  }
  assert.equal(calls,0);
});
test('noncanonical or closed Issue never reaches Host preflight',async()=>{
  const a=await readProjectTaskAdmission({...args,readDrafts:mocked(
    draft('NONCANONICAL'))});
  const b=await readProjectTaskAdmission({...args,readDrafts:mocked(
    draft('READY',{githubIssueState:'closed'}))});
  assert.equal(a.stage,'NEEDS_CANONICAL_LABELS');
  assert.equal(b.reason,'TASK_DRAFT_NOT_OPEN');
});
test('fake ready preflight authority is not admitted',async()=>{
  const result=await readProjectTaskAdmission({...args,
    readDrafts:mocked(draft('READY')),
    readNativePreflight:async()=>({status:'NATIVE_SOURCE_PREFLIGHT_READY',
      sourceQualified:true,nativePreflightReady:true,sourceSha:sha,
      issueRef:'https://github.com/org/test/issues/18',
      admitted:true,authorized:true})});
  assert.equal(result.stage,'BLOCKED');
  assert.equal(result.agentAuthorized,false);
});
