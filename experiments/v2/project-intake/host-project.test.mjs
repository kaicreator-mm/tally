import test from 'node:test';
import assert from 'node:assert/strict';
import {createProjectIntakeHost,freezeProductIntake,PRODUCT_INTAKE_KEY}
  from './host-project.mjs';

const sha='a'.repeat(40),digest='b'.repeat(64);
const env=()=>({TALLY_CLAIM_SOURCE_SHA:sha,TALLY_GITHUB_READ_TOKEN:'token-1234567890',
  TALLY_TASK_CLAIM_210_REGISTRY_PATH:'C:/registry/registry-210.sqlite'});
const claimOk=async()=>({ok:true});
const writeSetOk=async()=>({ok:true});
const frozenValue=Object.freeze({phase:'FROZEN_PRODUCT_INTAKE',
  issueRef:'https://github.com/kaicreator-mm/tally/issues/210',sourceSha:sha,
  productDigest:digest,review:{author:'author:1',reviewer:'reviewer:1'},
  frozenAt:'2026-10-31T09:00:00.000Z'});
function memoryStore(existing){const docs=new Map(existing??[]);
  return {async get(key){const d=docs.get(key);
      return d===undefined?null:{value:d.value,revision:d.revision};},
    async put(key,value,condition){if(condition?.kind==='create'&&docs.has(key))
        throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
      docs.set(key,{value,revision:(docs.get(key)?.revision??0)+1});},
    async close(){}};}
const openStoreOk=async options=>memoryStore([[PRODUCT_INTAKE_KEY,
  {value:frozenValue,revision:1}]]);
const host=(overrides={})=>createProjectIntakeHost({openStore:openStoreOk,
  verifyAcceptedClaim:claimOk,verifyFrozenWriteSet:writeSetOk,
  env:env(),...overrides});
const input=()=>({issueNumber:210,sourceSha:sha,productDigest:digest,
  projectTitle:'Tally v2 Intake'});

test('full governed effect: claim, write-set, frozen review, publish and canonical readback',async()=>{
  const publishCalls=[],readCalls=[];
  const h=host({publish:async arg=>{publishCalls.push(arg);
      return {state:'PROJECT_CREATED',project:{number:7,title:'Tally v2 Intake'}};},
    readProjectIntake:async arg=>{readCalls.push(arg);
      return {status:'PROJECT_INTAKE_OBSERVED',projects:[{number:7,
        title:'Tally v2 Intake',url:'https://github.com/kaicreator-mm/tally/projects/7',
        itemCount:1}]};}});
  const result=await h.hostProjectIntake(input());
  assert.equal(result.state,'PROJECT_INTAKE_ADMITTED');
  assert.equal(result.reason,'PROJECT_INTAKE_PUBLISHED_AND_READBACK_VERIFIED');
  assert.equal(result.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(result.claimAccepted,true);assert.equal(result.admitted,true);
  assert.equal(result.authorized,false);assert.equal(result.agentLaunch,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.equal(result.project.number,7);
  assert.deepEqual(publishCalls.map(x=>x.projectTitle),['Tally v2 Intake']);
  assert.equal(JSON.stringify(publishCalls).includes('token-1234567890'),false,
    'Host module must not hand credentials to transports');
});

test('missing claim, write-set, freeze record or transport fails closed with bounded reasons',async()=>{
  const cases=[
    [host({verifyAcceptedClaim:async()=>({ok:false,reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND'})}),
      'CLAIM_REGISTRY_RECORD_NOT_FOUND'],
    [host({verifyFrozenWriteSet:async()=>({ok:false,reason:'DAG_WRITESET_NOT_VERIFIED'})}),
      'DAG_WRITESET_NOT_VERIFIED'],
    [host({openStore:async()=>memoryStore()}),'PRODUCT_FREEZE_NOT_VERIFIED'],
    [host({openStore:async()=>{throw new Error('disk');}}),'PRODUCT_FREEZE_STORE_UNAVAILABLE'],
    [createProjectIntakeHost({env:env()}),'PROJECT_INTAKE_HOST_CONFIG_INVALID'],
    [host({env:{...env(),TALLY_GITHUB_READ_TOKEN:undefined}}),
      'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE'],
  ];
  for(const [h,reason] of cases){
    const result=await h.hostProjectIntake(input());
    assert.equal(result.state,'NOT_ADMITTED');assert.equal(result.reason,reason);
    assert.equal(result.githubEffect,'NOT_RUN');assert.equal(result.taskPass,false);
    assert.equal(result.claimAccepted,false);
  }
});

test('digest or reviewer mismatch against the frozen record refuses',async()=>{
  for(const [change,reason] of [
    [{productDigest:'c'.repeat(64)},'PRODUCT_FREEZE_DIGEST_MISMATCH'],
    [{sourceSha:'d'.repeat(40)},'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH'],
  ]){
    const result=await host().hostProjectIntake({...input(),...change});
    assert.equal(result.reason,reason);assert.equal(result.admitted,false);
  }
  const notIndependent=memoryStore([[PRODUCT_INTAKE_KEY,{value:{...frozenValue,
    review:{author:'same:1',reviewer:'same:1'}},revision:1}]]);
  const result=await host({openStore:async()=>notIndependent}).hostProjectIntake(input());
  assert.equal(result.reason,'PRODUCT_REVIEW_NOT_INDEPENDENT');
});

test('host pin refuses cross-issue, stale source and malformed input before any verification',async()=>{
  const h=host();
  let verifierCalls=0;
  const counting=createProjectIntakeHost({openStore:openStoreOk,
    verifyAcceptedClaim:async()=>{verifierCalls++;return {ok:true};},
    verifyFrozenWriteSet:writeSetOk,env:env()});
  for(const change of [
    {issueNumber:197},{sourceSha:'short'},{productDigest:'nothex'},
    {projectTitle:''},{sourceSha:'e'.repeat(40)},
  ]){
    const result=await counting.hostProjectIntake({...input(),...change});
    assert.equal(result.reason,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
    assert.equal(result.admitted,false);
  }
  assert.equal(verifierCalls,0,'no verifier may run on pinned-input refusal');
  assert.equal(await h.hostProjectIntake(null).then(r=>r.reason),
    'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
});

test('safe replay admits the intent but keeps the GitHub effect explicitly NOT_RUN',async()=>{
  const h=host({env:{...env(),TALLY_LOCAL_REPLAY:'1'}});
  const result=await h.hostProjectIntake(input());
  assert.equal(result.state,'REPLAY_COMPLETE');assert.equal(result.reason,'REPLAY_FIXTURE_ONLY');
  assert.equal(result.githubEffect,'NOT_RUN');assert.equal(result.replayFixture,true);
  assert.equal(result.claimAccepted,true);assert.equal(result.taskPass,false);
});

test('publish failure stays UNKNOWN and never fabricates a verified project',async()=>{
  for(const publish of [
    async()=>{throw new Error('network down');},
    async()=>({state:'BOGUS'}),
    async()=>({state:'PROJECT_CREATED',project:{number:0,title:'x'}}),
  ]){
    const result=await host({publish,readProjectIntake:async()=>null})
      .hostProjectIntake(input());
    assert.equal(result.state,'PROJECT_INTAKE_UNKNOWN');
    assert.equal(result.reason,'GITHUB_PUBLISH_UNKNOWN');
    assert.equal(result.githubEffect,'UNKNOWN');
  }
  const mismatched=host({publish:async()=>({state:'PROJECT_CREATED',
      project:{number:9,title:'Tally v2 Intake'}}),
    readProjectIntake:async()=>({status:'PROJECT_INTAKE_OBSERVED',projects:[]})});
  const result=await mismatched.hostProjectIntake(input());
  assert.equal(result.reason,'PROJECT_INTAKE_READBACK_MISMATCH');
  assert.equal(result.githubEffect,'UNKNOWN');assert.equal(result.project,null);
});

test('freeze writes one CAS record only after an independent review',async()=>{
  const store=memoryStore();
  const args={openStore:async()=>store,registryPath:'C:/registry/r210.sqlite',
    sourceSha:sha,productDigest:digest,author:'author:1',reviewer:'reviewer:1',
    frozenAt:'2026-10-31T09:00:00.000Z'};
  const selfReview=await freezeProductIntake({...args,reviewer:'author:1'});
  assert.equal(selfReview.state,'NOT_FROZEN');
  assert.equal(selfReview.reason,'PRODUCT_REVIEW_NOT_INDEPENDENT');
  const frozen=await freezeProductIntake(args);
  assert.equal(frozen.state,'FROZEN_COMPLETE');assert.equal(frozen.frozen,true);
  assert.equal(frozen.receiptRef,`receipt:sha256:${digest}`);
  const record=(await store.get(PRODUCT_INTAKE_KEY)).value;
  assert.equal(record.phase,'FROZEN_PRODUCT_INTAKE');assert.equal(record.sourceSha,sha);
  assert.equal(record.review.reviewer,'reviewer:1');
  const replay=await freezeProductIntake(args);
  assert.equal(replay.state,'FROZEN_COMPLETE');
  assert.equal(replay.reason,'PRODUCT_FREEZE_ALREADY_RECORDED');
  const changed=await freezeProductIntake({...args,productDigest:'c'.repeat(64)});
  assert.equal(changed.state,'NOT_FROZEN');
  assert.equal(changed.reason,'PRODUCT_FREEZE_DIGEST_MISMATCH');
});
