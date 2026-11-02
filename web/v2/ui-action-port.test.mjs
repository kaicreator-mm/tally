import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiActionPort,businessIntentForUiAction} from './ui-action-port.mjs';
const repository='org/tally',versionId='version/v2',sha='a'.repeat(40);
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const eligible={id:'start-260',kind:'START_AGENT',revision:1,label:'启动已准入 Agent',
  taskIssue:260,state:'ELIGIBLE',hostAdmission:'ADMITTED',
  permitSource:'SELECTED_K_HOST',permitSourceSha:sha};
const snapshot=()=>({connected:true,source,versionId,actions:[eligible]});
const request={repository,versionId,sourceSha:sha,actionId:'start-260',revision:1};
const receipt={status:'OBSERVED',sourceSha:sha,actionId:'start-260',
  receiptRef:'https://github.com/org/tally/issues/260#issuecomment-100'};
const make=(overrides={})=>createUiActionPort({repository,versionId,expectedSha:sha,
  host:{readCurrentActions:async()=>snapshot(),dispatchAuthorizedAction:async()=>receipt,...overrides}});
test('missing Host and foreign scope fail closed',async()=>{
  assert.equal((await createUiActionPort({repository,versionId,expectedSha:sha}).actions()).connected,false);
  assert.equal((await make().actions({repository:'evil/repo'})).reason,'ACTION_SCOPE_MISMATCH');
});
test('read model exposes a bounded action, never a raw grant or permit',async()=>{
  const data=await make().actions({repository,versionId});
  assert.equal(data.connected,true);
  assert.deepEqual(Object.keys(data.actions[0]).sort(),
    ['id','kind','revision','label','taskIssue','state','reason'].sort());
  assert.equal(data.actions[0].hostAdmission,undefined);
});
test('only selected Host can dispatch an exact current admitted action',async()=>{
  let calls=0;
  const host=make({dispatchAuthorizedAction:async()=>{calls++;return receipt;}});
  const result=await host.submit(request);
  assert.equal(calls,1);
  assert.equal(result.ok,true);
  assert.equal(result.receiptRef,receipt.receiptRef);
  assert.equal(result.taskQualification,'NOT_VERIFIED');
  assert.equal(result.releaseQualification,'NOT_VERIFIED');
});
test('browser cannot supply a grant, payload or stale revision',async()=>{
  let dispatch=0;
  const host=make({dispatchAuthorizedAction:async()=>{dispatch++;return receipt;}});
  for(const bad of [{...request,token:'ghp_secret'},
    {...request,revision:2},{...request,sourceSha:'b'.repeat(40)},
    {...request,actionId:'MERGE'}, {...request,repository:'evil/repo'}]){
    assert.equal((await host.submit(bad)).status,'DENIED');
  }
  assert.equal(dispatch,0);
});
test('candidate Claim or unsigned permit cannot grant an effect',async()=>{
  for(const bad of [{...eligible,hostAdmission:'CLAIM_CANDIDATE'},
    {...eligible,permitSource:'BROWSER'},
    {...eligible,permitSourceSha:'b'.repeat(40)}]){
    let calls=0;
    const host=make({readCurrentActions:async()=>({...snapshot(),actions:[bad]}),
      dispatchAuthorizedAction:async()=>{calls++;return receipt;}});
    assert.equal((await host.submit(request)).status,'DENIED');
    assert.equal(calls,0);
  }
});
test('stale second read refuses dispatch',async()=>{
  let reads=0,dispatch=0;
  const host=make({readCurrentActions:async()=>({...snapshot(),actions:[
    {...eligible,revision:++reads}]}),
    dispatchAuthorizedAction:async()=>{dispatch++;return receipt;}});
  assert.equal((await host.submit(request)).reason,'ACTION_SOURCE_CHANGED_BEFORE_DISPATCH');
  assert.equal(dispatch,0);
});
test('cross-project, malformed Host results and attempted PASS elevation rejected',async()=>{
  const bad=[{...snapshot(),source:{...source,repository:'evil/repo'}},
    {...snapshot(),source:{...source,currentness:'STALE'}},
    {...snapshot(),actions:[eligible,eligible]},
    {...snapshot(),actions:[{...eligible,label:'ghp_abcdefghijklmnopqrstuvwxyz'}]}];
  for(const x of bad){
    assert.equal((await make({readCurrentActions:async()=>x}).actions()).connected,false);
  }
  for(const res of [{...receipt,taskPass:true},{...receipt,releasePass:true},
    {...receipt,receiptRef:'https://evil/repo'},{...receipt,authorized:true}]){
    assert.equal((await make({dispatchAuthorizedAction:async()=>res}).submit(request)).status,'DENIED');
  }
});
test('lost dispatch acknowledgement does not claim success or retry',async()=>{
  let count=0;
  const r=await make({dispatchAuthorizedAction:async()=>{count++;throw Error('private key');}})
    .submit(request);
  assert.equal(count,1);
  assert.equal(r.status,'UNKNOWN');
  assert.equal(JSON.stringify(r).includes('private key'),false);
});

test('UI cannot invent Task creation or CAS Claim publication as Business intents',async()=>{
  for(const kind of ['CREATE_TASK','CLAIM_TASK']){
    const action={...eligible,kind};
    const result=await make({readCurrentActions:async()=>({...snapshot(),actions:[action]})})
      .actions();
    assert.equal(result.connected,false);
  }
});

test('Host action kinds map only to existing selected Business intents',()=>{
  assert.equal(businessIntentForUiAction('FREEZE_PRODUCT'),'freezeProductIntake');
  assert.equal(businessIntentForUiAction('ADMIT_PROJECT'),'admitProjectIntake');
  assert.equal(businessIntentForUiAction('ADMIT_CLAIMED_TASK'),'admitClaimedTask');
  assert.equal(businessIntentForUiAction('START_AGENT'),'runLocalCodexAttempt');
  assert.equal(businessIntentForUiAction('REVIEW_REPAIR'),'joinReviewedCandidates');
  assert.equal(businessIntentForUiAction('PUBLISH_CANDIDATE'),'publishReviewedCandidate');
  assert.equal(businessIntentForUiAction('PUBLISH_MAINTENANCE_ISSUE'),'publishMaintenanceIssue');
  assert.equal(businessIntentForUiAction('RECORD_TASK_LEARNING'),'recordTaskLearningOutcome');
  assert.equal(businessIntentForUiAction('DELIVER_QUALIFIED_RELEASE'),'deliverQualifiedRelease');
  assert.equal(businessIntentForUiAction('CREATE_TASK'),null);
  assert.equal(businessIntentForUiAction('CLAIM_TASK'),null);
});
