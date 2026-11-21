import test from 'node:test';
import assert from 'node:assert/strict';
import {createProjectTaskReadyHost} from './project-task-ready-host.mjs';
const sha='a'.repeat(40);
const settings={repository:'org/test',owner:'org',repo:'test',
  versionId:'main',expectedSha:sha,token:'github_pat_TEST_READ'};
const env={TALLY_V2_TASK_READY:'host-mounted',
  TALLY_TASK_READY_REGISTRY_PATH:'/tmp/test-ready.sqlite',
  TALLY_TASK_READY_REPOSITORY:'org/test',
  TALLY_TASK_READY_WRITER_ID:'local-writer',
  TALLY_TASK_READY_WRITER_LOGIN:'operator'};
const octokit={rest:{issues:{get:()=>{}},repos:{getBranch:()=>{}}},
  paginate:{iterator:()=>{}}};
const preflight=async()=>({connected:true,stage:'PLANNED'});
test('unmounted host grants no planned transition',()=>{
  assert.equal(createProjectTaskReadyHost({settings,env:{}}),null);
});
test('wrong project cannot inherit writer authority',()=>{
  assert.throws(()=>createProjectTaskReadyHost({settings,
    env:{...env,TALLY_TASK_READY_REPOSITORY:'org/other'},
    octokit,openStore:async()=>({})}),/TRUSTED_TASK_READY_HOST_CONFIGURATION_REQUIRED/);
});
test('source or selected Issue mismatch rejects before any Host read',async()=>{
  let calls=0;
  const host=createProjectTaskReadyHost({settings,env,octokit,
    openStore:async()=>{calls++;},readAdmission:async()=>{calls++;}});
  const result=await host.transition({issueNumber:8,sourceSha:'b'.repeat(40)});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.githubEffect,'NOT_RUN');
  assert.equal(calls,0);
});
test('only a source-verified planned draft may enter native Host port',async()=>{
  let calls=0;
  const host=createProjectTaskReadyHost({settings,env,octokit,
    openStore:async()=>{calls++;return {close:async()=>{calls++;}};},
    readAdmission:preflight,
    createPort:input=>{
      calls++;assert.equal(input.repository,'org/test');
      assert.equal(input.versionBranch,'main');
      assert.equal(input.targetIssue,7);
      assert.deepEqual(input.draftSourceIdentity,{repository:'org/test',
        versionId:'main',sourceSha:sha});
      assert.equal(input.writerAuthority.exclusiveHostControl,true);
      return {transitionToReady:async()=>{
        calls++;return {status:'READY_VERIFIED',githubEffect:'VERIFIED',
          authorized:false,admitted:false,taskPass:false};
      }};
    }});
  const result=await host.transition({issueNumber:7,sourceSha:sha});
  assert.equal(result.status,'READY_VERIFIED');
  assert.equal(result.admitted,false);
  assert.equal(calls,4);
});
test('non-planned draft cannot open registry',async()=>{
  const host=createProjectTaskReadyHost({settings,env,octokit,
    openStore:async()=>{throw Error('should not open');},
    readAdmission:async()=>({connected:true,stage:'NEEDS_CANONICAL_LABELS',
      reason:'TASK_CANONICAL_LABELS_REQUIRED'})});
  const result=await host.transition({issueNumber:7,sourceSha:sha});
  assert.equal(result.reason,'TASK_CANONICAL_LABELS_REQUIRED');
  assert.equal(result.githubEffect,'NOT_RUN');
});

test('transport error after entering native port stays UNKNOWN, not NOT_RUN',async()=>{
  const host=createProjectTaskReadyHost({settings,env,octokit,
    openStore:async()=>({close:async()=>{}}),readAdmission:preflight,
    createPort:()=>({transitionToReady:async()=>{throw Error('lost ACK');}})});
  const result=await host.transition({issueNumber:7,sourceSha:sha});
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.githubEffect,'UNKNOWN');
  assert.equal(result.admitted,false);
});
