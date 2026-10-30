import test from 'node:test';
import assert from 'node:assert/strict';
import {getPolicy,assessTaskEvidence,assessClaimedTaskAdmission,
  projectClaimedTaskAdmission,ABI_VERSION,PACKAGE_KIND,INTENT_TYPES,EFFECT_TYPES}
  from '../src/index.mjs';
const good={issueRef:'https://github.com/kaicreator-mm/tally/issues/108',issueNumber:108,sourceSha:'a'.repeat(40),claimRef:'github:claim:108',dependenciesComplete:true};
const claimed={issueRef:'https://github.com/kaicreator-mm/tally/issues/169',issueNumber:169,sourceSha:'a'.repeat(40)};
const ADMIT_REASON='CLAIM_REGISTRY_PUBLICATION_AND_NATIVE_READBACK_VERIFIED';
const sealedPositive=(input)=>({
  state:'ADMITTED_PENDING_AGENT_EFFECT',reason:ADMIT_REASON,
  issueRef:input.issueRef,sourceSha:input.sourceSha,
  protectedClaimKey:'kaicreator-mm/tally#169:builder:__default__',generation:1,
  dispatchId:'dispatch:169:t06l:test:1',actorRole:'builder',operatorKind:'other',
  operatorId:'zcode:t125:test:1',commentId:6511046226,
  commentRef:input.issueRef+'#issuecomment-6511046226',
  claimPublication:'REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK',
  claimAccepted:true,admitted:true,authorized:false,taskPass:false,
  agentLaunch:false,releasePass:false,
});
test('real DH public Business ABI identifiers',()=>{
  assert.equal(ABI_VERSION,'v08-business-abi/1');assert.equal(PACKAGE_KIND,'business');
  assert.deepEqual([...INTENT_TYPES],['inspectTaskReadiness','admitClaimedTask']);
  assert.deepEqual([...EFFECT_TYPES],['effect:tally-readiness-observation',
    'effect:tally-claim-execution-admission']);
});
test('Policy declares one K-sealed read-only C operation, never external GitHub effect',()=>{
  const p=getPolicy();assert.deepEqual(Object.keys(p.intentBindings),['inspectTaskReadiness','admitClaimedTask']);
  assert.deepEqual(Object.keys(p.effectBindings),['effect:tally-readiness-observation',
    'effect:tally-claim-execution-admission']);
  assert.ok(!('external' in p.effectBindings['effect:tally-readiness-observation']));
  assert.deepEqual(p.effectBindings['effect:tally-readiness-observation'].resource,
    {key:'tally-source-observation',operation:'tally.observeReadiness'});
  assert.ok(!('external' in p.effectBindings['effect:tally-claim-execution-admission']));
  assert.deepEqual(p.effectBindings['effect:tally-claim-execution-admission'].resource,
    {key:'tally-claim-admission',operation:'tally.admitClaimedTask'});
});
test('canonical observed input still cannot grant K admission or Task PASS',()=>{
  const x=assessTaskEvidence(good);
  assert.equal(x.state,'OBSERVED_INPUT_ONLY');assert.equal(x.admitted,false);assert.equal(x.taskPass,false);
});
test('unproven DAG dependency refuses',()=>{
  assert.equal(assessTaskEvidence({...good,dependenciesComplete:false}).reason,'DEPENDENCIES_UNVERIFIED');
  assert.equal(assessTaskEvidence({...good,dependenciesComplete:'yes'}).state,'UNQUALIFIED_INPUT');
});
test('invalid/missing GitHub source identity refuses',()=>{
  for(const i of [null,{},'x',{...good,sourceSha:'bad'},{...good,issueNumber:109},{...good,issueNumber:0},{...good,claimRef:''},{...good,issueRef:'https://evil.test/1'}]){
    assert.equal(assessTaskEvidence(i).admitted,false);
    assert.equal(assessTaskEvidence(i).taskPass,false);
  }
});
test('B effect only returns advisory, no remote side effects',async()=>{
  const p=getPolicy();
  const x=await p.effectBindings['effect:tally-readiness-observation'].handler({input:good,effectId:'e-1',
    resources:{call:async (key,op,arg)=>{
      assert.equal(key,'tally-source-observation');
      assert.equal(op,'tally.observeReadiness');
      assert.equal(arg.effectId,'e-1');
      return {state:'OBSERVATION_NOT_NATIVE_VERIFIED',taskPass:false};
    }}});
  assert.equal(x.admitted,false);assert.equal(x.taskPass,false);assert.equal(x.effectId,'e-1');
  assert.equal(x.sourceObservation.state,'OBSERVATION_NOT_NATIVE_VERIFIED');
});
test('workflow rule, binding and trigger use same event',()=>{
  const p=getPolicy();const e='TALLY_TASK_READINESS_INSPECTION_REQUESTED';
  assert.equal(p.intentBindings.inspectTaskReadiness.trigger.eventType,e);
  assert.equal(p.rules['rule:tally-readiness'].eventType,e);
  assert.equal(p.workflowDefinition.states[0].transitions[0].trigger.eventType,e);
});
test('all inputFrom projection names are covered by the Rule',()=>{
  const p=getPolicy();const t=p.workflowDefinition.states[0].transitions[0];
  const names=Object.keys(t.effectIntents[0].inputFrom);
  for(const name of names) assert.ok(Object.hasOwn(p.rules['rule:tally-readiness'].payloadFromInput,name));
});
test('no K, C, Host, HTTP, gh credentials required by pure policy',()=>{
  const p=getPolicy();
  assert.equal(p.domainId,'tally-engineering');
  assert.equal(Object.keys(p.effectBindings).length,2);
  assert.equal(p.workflowDefinition.states[0].transitions[0].targetState,'open');
  assert.equal(p.workflowDefinition.states[0].transitions[1].targetState,'open');
});

test('Kernel v0.10.2 idempotency-key grammar accepts rendered valid source key',()=>{
  // Exactly the public-tag Kernel source: ADMISSION_EFFECT_IDEMPOTENCY_KEY_PATTERN.
  const pattern=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
  const intent=getPolicy().workflowDefinition.states[0].transitions[0].effectIntents[0];
  const rendered=intent.idempotencyKeyFrom.template.replace(/\{([A-Za-z0-9_]+)\}/g,(_m,name)=>String(good[name]));
  assert.equal(rendered, 'inspect:108:'+good.sourceSha);
  assert.ok(pattern.test(rendered));
  assert.ok(!rendered.includes('/'));
  assert.equal(assessTaskEvidence({...good,issueNumber:109}).admitted,false);
});

test('decision schema rejects arrays as decision or event',()=>{
  const schema=getPolicy().decisionSchema;
  assert.equal(schema.isValid({decision:[],event:{type:'ok'}}),false);
  assert.equal(schema.isValid({decision:{outcome:'ok'},event:[]}),false);
});

test('public Microkernel requester may inspect advisory without approval authority',async()=>{
  const p=getPolicy();
  assert.ok(p.intentBindings.inspectTaskReadiness.roles.includes('requester'));
  const r=await p.effectBindings['effect:tally-readiness-observation'].handler({
    input:good,effectId:'public-readonly-requester',
    resources:{call:async()=>({state:'OBSERVATION_NOT_NATIVE_VERIFIED',authorized:false,taskPass:false})}
  });
  assert.equal(r.admitted,false);
  assert.equal(r.taskPass,false);
  assert.equal(r.state,'OBSERVED_INPUT_ONLY');
});

test('readiness effect rejects absent/unsealed C with no forged task authority',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-readiness-observation'];
  await assert.rejects(effect.handler({input:good,effectId:'no-c'}),/SELECTED_READONLY_CAPABILITY_REQUIRED/);
  await assert.rejects(effect.handler({input:good,effectId:'wrong-c',resources:{
    call:async()=>{throw Object.assign(new Error('RESOURCE_ROUTE_UNSEALED'),{code:'RESOURCE_ROUTE_UNSEALED'});}
  }}),{code:'RESOURCE_ROUTE_UNSEALED'});
});
test('caller claim text does not upgrade native C output into Task PASS',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-readiness-observation'];
  const res=await effect.handler({input:{...good,claimRef:'github:fake:pass',dependenciesComplete:true},
    effectId:'caller-claim-unverified',resources:{call:async()=>({
      state:'OBSERVATION_NOT_NATIVE_VERIFIED',authorized:false,taskPass:false,releasePass:false
    })}});
  assert.equal(res.admitted,false);assert.equal(res.taskPass,false);
  assert.equal(res.sourceObservation.taskPass,false);
});

test('T06l claim admission rule, binding and trigger use same event',()=>{
  const p=getPolicy();const e='TALLY_CLAIMED_TASK_ADMISSION_REQUESTED';
  assert.equal(p.intentBindings.admitClaimedTask.trigger.eventType,e);
  assert.equal(p.intentBindings.admitClaimedTask.ruleId,'rule:tally-claim-admission');
  assert.equal(p.rules['rule:tally-claim-admission'].eventType,e);
  assert.equal(p.workflowDefinition.states[0].transitions[1].trigger.eventType,e);
  assert.equal(p.workflowDefinition.states[0].transitions[1].effectIntents[0].effectType,
    'effect:tally-claim-execution-admission');
  const names=Object.keys(p.workflowDefinition.states[0].transitions[1]
    .effectIntents[0].inputFrom);
  for(const name of names)
    assert.ok(Object.hasOwn(p.rules['rule:tally-claim-admission'].payloadFromInput,name));
});

test('T06l each intent binding carries its own single-transition definition',()=>{
  // DH 0.10.2 wires every transition of a definition against each intent
  // rule and the S SDK projects the whole rule payload from the event; the
  // two dynamic effect surfaces must therefore never share a definition.
  const p=getPolicy();
  const inspect=p.intentBindings.inspectTaskReadiness.definition;
  const admit=p.intentBindings.admitClaimedTask.definition;
  assert.equal(inspect.workflowKey,'tally-readiness');
  assert.equal(admit.workflowKey,'tally-readiness');
  assert.equal(inspect.states[0].transitions.length,1);
  assert.equal(admit.states[0].transitions.length,1);
  assert.equal(inspect.states[0].transitions[0].transitionKey,'inspect-task-source');
  assert.equal(admit.states[0].transitions[0].transitionKey,'admit-claimed-task');
  assert.equal(inspect.states[0].transitions[0].effectIntents[0].effectType,
    'effect:tally-readiness-observation');
  assert.equal(admit.states[0].transitions[0].effectIntents[0].effectType,
    'effect:tally-claim-execution-admission');
  const inspectNames=Object.keys(inspect.states[0].transitions[0].effectIntents[0].inputFrom);
  for(const name of inspectNames)
    assert.ok(Object.hasOwn(p.rules['rule:tally-readiness'].payloadFromInput,name));
});

test('T06l admit idempotency-key grammar accepts rendered valid source key',()=>{
  const pattern=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
  const intent=getPolicy().workflowDefinition.states[0].transitions[1].effectIntents[0];
  const rendered=intent.idempotencyKeyFrom.template.replace(/\{([A-Za-z0-9_]+)\}/g,(_m,name)=>String(claimed[name]));
  assert.equal(rendered,'admit:169:'+claimed.sourceSha);
  assert.ok(pattern.test(rendered));
  assert.ok(!rendered.includes('/'));
});

test('T06l advisory screening: input alone is never admission evidence',()=>{
  const x=assessClaimedTaskAdmission(claimed);
  assert.equal(x.state,'OBSERVED_INPUT_ONLY');assert.equal(x.admitted,false);
  assert.equal(x.taskPass,false);assert.equal(x.agentLaunch,false);assert.equal(x.releasePass,false);
  for(const i of [null,{},'x',{...claimed,sourceSha:'bad'},{...claimed,issueNumber:170},
      {...claimed,issueNumber:0},{...claimed,issueRef:'https://evil.test/1'}]){
    assert.equal(assessClaimedTaskAdmission(i).admitted,false);
    assert.equal(assessClaimedTaskAdmission(i).state,'UNQUALIFIED_INPUT');
  }
});

test('T06l caller/comment accepted booleans never upgrade projection to admission',()=>{
  const callerAsserted={...claimed,accepted:true,phase:'PUBLISHED_VERIFIED',
    commentClaimAccepted:true,claimAccepted:true};
  const x=projectClaimedTaskAdmission({input:callerAsserted,
    claimObservation:{state:'NOT_ADMITTED',reason:'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
      admitted:false,authorized:false,taskPass:false,agentLaunch:false,releasePass:false},
    effectId:'e-caller-assert'});
  assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.admitted,false);
  assert.equal(x.taskPass,false);assert.equal(x.agentLaunch,false);assert.equal(x.releasePass,false);
});

test('T06l sealed positive C observation projects K intent admission only',()=>{
  const x=projectClaimedTaskAdmission({input:claimed,
    claimObservation:sealedPositive(claimed),effectId:'e-admit-1'});
  assert.equal(x.state,'ADMITTED_PENDING_AGENT_EFFECT');
  assert.equal(x.admitted,true);
  assert.equal(x.authorized,false);assert.equal(x.taskPass,false);
  assert.equal(x.agentLaunch,false);assert.equal(x.releasePass,false);
  assert.equal(x.effectId,'e-admit-1');
  const o=x.claimObservation;
  assert.equal(o.claimAccepted,true);
  assert.equal(o.generation,1);assert.equal(o.actorRole,'builder');
  assert.equal(o.issueRef,claimed.issueRef);assert.equal(o.sourceSha,claimed.sourceSha);
  assert.equal(o.commentRef,claimed.issueRef+'#issuecomment-6511046226');
});

test('T06l positive projection fails closed on any sealed-shape deviation',()=>{
  const base=sealedPositive(claimed);
  for(const mutated of [
    {...base,claimAccepted:false},{...base,admitted:false},{...base,authorized:true},
    {...base,taskPass:true},{...base,agentLaunch:true},{...base,releasePass:true},
    {...base,generation:2},{...base,actorRole:'reviewer'},
    {...base,commentRef:'https://evil.test/issuecomment-1'},
    {...base,sourceSha:'b'.repeat(40)},{...base,claimPublication:'GITHUB_COMMENT_POST_READBACK'},
    {...base,reason:'COMMENT_SELF_REPORTED_ACCEPTED'},
  ]){
    const x=projectClaimedTaskAdmission({input:claimed,claimObservation:mutated,effectId:'e-x'});
    assert.equal(x.state,'NOT_ADMITTED',JSON.stringify(mutated));
    assert.equal(x.admitted,false);
    assert.equal(x.taskPass,false);assert.equal(x.agentLaunch,false);assert.equal(x.releasePass,false);
  }
  for(const garbage of [null,{},[],{state:'ADMITTED_PENDING_AGENT_EFFECT'},
      {state:'NOT_ADMITTED',reason:42,admitted:false}]){
    const x=projectClaimedTaskAdmission({input:claimed,claimObservation:garbage,effectId:'e-g'});
    assert.equal(x.admitted,false);assert.equal(x.taskPass,false);
  }
});

test('T06l admission effect rejects absent/unsealed C and projects C denial',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-claim-execution-admission'];
  await assert.rejects(effect.handler({input:claimed,effectId:'no-c'}),/SELECTED_READONLY_CAPABILITY_REQUIRED/);
  await assert.rejects(effect.handler({input:claimed,effectId:'wrong-c',resources:{
    call:async()=>{throw Object.assign(new Error('RESOURCE_ROUTE_UNSEALED'),{code:'RESOURCE_ROUTE_UNSEALED'});}
  }}),{code:'RESOURCE_ROUTE_UNSEALED'});
  const denied=await effect.handler({input:claimed,effectId:'c-denies',resources:{
    call:async(key,op,arg)=>{
      assert.equal(key,'tally-claim-admission');
      assert.equal(op,'tally.admitClaimedTask');
      assert.equal(arg.effectId,'c-denies');
      assert.equal(arg.sourceRef,claimed.issueRef);
      return {state:'NOT_ADMITTED',reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND',
        admitted:false,authorized:false,taskPass:false,agentLaunch:false,releasePass:false};
    }}});
  assert.equal(denied.state,'NOT_ADMITTED');
  assert.equal(denied.reason,'CLAIM_REGISTRY_RECORD_NOT_FOUND');
  assert.equal(denied.admitted,false);assert.equal(denied.taskPass,false);
  assert.equal(denied.agentLaunch,false);assert.equal(denied.effectId,'c-denies');
});

test('T06l admission effect short-circuits unqualified input before any C call',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-claim-execution-admission'];
  let called=0;
  const x=await effect.handler({input:{...claimed,sourceSha:'bad'},effectId:'bad-input',
    resources:{call:async()=>{called+=1;return sealedPositive(claimed);}}});
  assert.equal(called,0);
  assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(x.admitted,false);
  assert.equal(x.taskPass,false);assert.equal(x.effectId,'bad-input');
});

test('T06l admission effect returns K intent admission, never Task/Agent authority',async()=>{
  const p=getPolicy();
  assert.ok(p.intentBindings.admitClaimedTask.roles.includes('requester'));
  const x=await p.effectBindings['effect:tally-claim-execution-admission'].handler({
    input:claimed,effectId:'public-requester-admit',
    resources:{call:async()=>sealedPositive(claimed)}});
  assert.equal(x.state,'ADMITTED_PENDING_AGENT_EFFECT');
  assert.equal(x.admitted,true);
  assert.equal(x.authorized,false);assert.equal(x.taskPass,false);
  assert.equal(x.agentLaunch,false);assert.equal(x.releasePass,false);
});
