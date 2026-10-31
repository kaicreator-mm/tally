import test from 'node:test';
import assert from 'node:assert/strict';
import {getPolicy,assessTaskEvidence,assessClaimedTaskAdmission,
  projectClaimedTaskAdmission,assessLocalCodexAttempt,projectLocalCodexAttempt,
  assessReviewedCandidateJoin,projectReviewedCandidateJoin,projectNativeTaskEffectiveSnapshot,
  assessCandidatePublication,projectCandidatePublication,assessProductIntakeFreeze,
  projectProductIntakeFreeze,assessProjectIntakeAdmission,projectProjectIntakeAdmission,
  assessMaintenanceIssuePublication,projectMaintenanceIssuePublication,
  assessTaskLearningOutcome,projectTaskLearningOutcome,
  ABI_VERSION,PACKAGE_KIND,INTENT_TYPES,EFFECT_TYPES}
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
  assert.deepEqual([...INTENT_TYPES],['inspectTaskReadiness','admitClaimedTask','runLocalCodexAttempt','joinReviewedCandidates','publishReviewedCandidate','freezeProductIntake','admitProjectIntake','publishMaintenanceIssue','recordTaskLearningOutcome']);
  assert.deepEqual([...EFFECT_TYPES],['effect:tally-readiness-observation',
    'effect:tally-claim-execution-admission','effect:tally-local-codex-attempt',
    'effect:tally-reviewer-candidate-join','effect:tally-reviewed-candidate-publication',
    'effect:tally-product-intake-freeze','effect:tally-project-intake-admission',
    'effect:tally-maintenance-issue-publication','effect:tally-task-learning-outcome']);
});
test('Policy declares one K-sealed read-only C operation, never external GitHub effect',()=>{
  const p=getPolicy();assert.deepEqual(Object.keys(p.intentBindings),['inspectTaskReadiness','admitClaimedTask','runLocalCodexAttempt','joinReviewedCandidates','publishReviewedCandidate','freezeProductIntake','admitProjectIntake','publishMaintenanceIssue','recordTaskLearningOutcome']);
  assert.deepEqual(Object.keys(p.effectBindings),['effect:tally-readiness-observation',
    'effect:tally-claim-execution-admission','effect:tally-local-codex-attempt',
    'effect:tally-reviewer-candidate-join','effect:tally-reviewed-candidate-publication',
    'effect:tally-product-intake-freeze','effect:tally-project-intake-admission',
    'effect:tally-maintenance-issue-publication','effect:tally-task-learning-outcome']);
  assert.ok(!('external' in p.effectBindings['effect:tally-readiness-observation']));
  assert.deepEqual(p.effectBindings['effect:tally-readiness-observation'].resource,
    {key:'tally-source-observation',operation:'tally.observeReadiness'});
  assert.ok(!('external' in p.effectBindings['effect:tally-claim-execution-admission']));
  assert.deepEqual(p.effectBindings['effect:tally-claim-execution-admission'].resource,
    {key:'tally-claim-admission',operation:'tally.admitClaimedTask'});
  assert.equal(p.effectBindings['effect:tally-local-codex-attempt'].semantics,'non-idempotent');
  assert.deepEqual(p.effectBindings['effect:tally-local-codex-attempt'].external,
    {permitTtlMs:3600000,await:'terminal'});
  assert.deepEqual(p.effectBindings['effect:tally-local-codex-attempt'].resource,
    {key:'tally-local-codex',operation:'tally.runLocalCodexAttempt'});
  assert.equal(p.effectBindings['effect:tally-reviewer-candidate-join'].semantics,'idempotent');
  assert.deepEqual(p.effectBindings['effect:tally-reviewer-candidate-join'].external,
    {permitTtlMs:3600000,await:'terminal'});
  assert.deepEqual(p.effectBindings['effect:tally-reviewer-candidate-join'].resource,
    {key:'tally-local-codex',operation:'tally.joinReviewedCandidates'});
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
  assert.equal(Object.keys(p.effectBindings).length,9);
  assert.equal(p.workflowDefinition.states[0].transitions[0].targetState,'open');
  assert.equal(p.workflowDefinition.states[0].transitions[1].targetState,'open');
  // T210 intake intents: freeze is an idempotent Host CAS write, admission is
  // the only GitHub-write path, both sealed behind explicit K permits.
  assert.equal(p.effectBindings['effect:tally-product-intake-freeze'].semantics,'idempotent');
  assert.deepEqual(p.effectBindings['effect:tally-product-intake-freeze'].resource,
    {key:'tally-local-codex',operation:'tally.freezeProductIntake'});
  assert.deepEqual(p.effectBindings['effect:tally-product-intake-freeze'].external,
    {permitTtlMs:3600000,await:'terminal'});
  assert.equal(p.effectBindings['effect:tally-project-intake-admission'].semantics,'non-idempotent');
  assert.deepEqual(p.effectBindings['effect:tally-project-intake-admission'].resource,
    {key:'tally-local-codex',operation:'tally.hostProjectIntake'});
  assert.deepEqual(p.effectBindings['effect:tally-project-intake-admission'].external,
    {permitTtlMs:3600000,await:'terminal'});
  for(const key of ['freezeProductIntake','admitProjectIntake'])
    assert.ok(p.intentBindings[key].roles.includes('requester'));
  const ruleByTransition={'inspect-task-source':'rule:tally-readiness',
    'admit-claimed-task':'rule:tally-claim-admission',
    'run-local-codex-attempt':'rule:tally-local-codex-attempt',
    'join-reviewed-candidates':'rule:tally-reviewer-candidate-join',
    'publish-reviewed-candidate':'rule:tally-reviewed-candidate-publication',
    'freeze-product-intake':'rule:tally-product-intake-freeze',
    'admit-project-intake':'rule:tally-project-intake-admission',
    'publish-maintenance-issue':'rule:tally-maintenance-issue-publication',
    'record-task-learning-outcome':'rule:tally-task-learning-outcome'};
  assert.deepEqual(Object.keys(ruleByTransition).sort(),
    p.workflowDefinition.states[0].transitions.map(t=>t.transitionKey).sort());
  for(const transition of p.workflowDefinition.states[0].transitions){
    const rule=p.rules[ruleByTransition[transition.transitionKey]];
    for(const name of Object.keys(transition.effectIntents[0].inputFrom))
      assert.ok(Object.hasOwn(rule.payloadFromInput,name),
        `${transition.transitionKey}:${name}`);
  }});

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

const localInput={issueRef:'https://github.com/kaicreator-mm/tally/issues/172',issueNumber:172,
  sourceSha:'2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9',attemptId:'attempt:t06m:test:1'};
const localBase={issueRef:localInput.issueRef,sourceSha:localInput.sourceSha,attemptId:localInput.attemptId,
  claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,taskPass:false,releasePass:false};
const localReady={...localBase,state:'LOCAL_CANDIDATE_READY',reason:'REAL_CODEX_CANDIDATE_RECORDED',
  agentLaunch:true,replayFixture:false,candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),
  process:{status:'EXITED',exitCode:0,pid:123,builderPrincipal:'nilar\\codexsandboxoffline',
    builderSid:'S-1-5-21-1-2-3-1003',startedAt:'2026-10-30T00:00:00Z',terminatedAt:'2026-10-30T00:01:00Z'}};
test('T06m Local Codex intent uses a non-idempotent effect and matching rule payload',()=>{
  const p=getPolicy(),e='TALLY_LOCAL_CODEX_ATTEMPT_REQUESTED';
  assert.equal(p.intentBindings.runLocalCodexAttempt.trigger.eventType,e);
  assert.equal(p.rules['rule:tally-local-codex-attempt'].eventType,e);
  const definition=p.intentBindings.runLocalCodexAttempt.definition;
  assert.equal(definition.states[0].transitions.length,1);
  const transition=definition.states[0].transitions[0];
  assert.equal(transition.effectIntents[0].effectType,'effect:tally-local-codex-attempt');
  assert.deepEqual(Object.keys(transition.effectIntents[0].inputFrom),
    ['issueRef','issueNumber','sourceSha','attemptId']);
  for(const name of Object.keys(transition.effectIntents[0].inputFrom))
    assert.ok(Object.hasOwn(p.rules['rule:tally-local-codex-attempt'].payloadFromInput,name));
});
test('T06m input alone never grants Agent execution and malformed attempts skip C',async()=>{
  assert.equal(assessLocalCodexAttempt(localInput).state,'OBSERVED_INPUT_ONLY');
  for(const value of [null,{},'x',{...localInput,sourceSha:'bad'},{...localInput,attemptId:'../escape'},
    {...localInput,issueNumber:169}])assert.equal(assessLocalCodexAttempt(value).state,'UNQUALIFIED_INPUT');
  let calls=0;
  const x=await getPolicy().effectBindings['effect:tally-local-codex-attempt'].handler({
    input:{...localInput,attemptId:'bad/id'},effectId:'bad-local',resources:{call:async()=>{calls++;}}});
  assert.equal(calls,0);assert.equal(x.agentLaunch,false);assert.equal(x.taskPass,false);
});
test('T06m real candidate projection preserves exact tree and process without Task PASS',()=>{
  const x=projectLocalCodexAttempt({input:localInput,observation:localReady,effectId:'local-ready'});
  assert.equal(x.state,'LOCAL_CANDIDATE_READY');assert.equal(x.agentLaunch,true);
  assert.equal(x.candidateTree,'a'.repeat(40));assert.equal(x.process.pid,123);
  assert.equal(x.authorized,false);assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
});
test('T06m replay, unavailable/confinement and UNKNOWN states remain distinct',()=>{
  const replay=projectLocalCodexAttempt({input:localInput,observation:{...localBase,
    state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY',agentLaunch:false,replayFixture:true},effectId:'r'});
  assert.equal(replay.state,'REPLAY_COMPLETE');assert.equal(replay.agentLaunch,false);
  for(const [state,reason] of [['AGENT_NOT_EXECUTED','LOCAL_AGENT_UNAVAILABLE'],
    ['AGENT_NOT_EXECUTED','BUILDER_CONFINEMENT_UNAVAILABLE']]){
    const x=projectLocalCodexAttempt({input:localInput,observation:{...localBase,state,reason,
      agentLaunch:false,replayFixture:false},effectId:reason});
    assert.equal(x.state,'AGENT_NOT_EXECUTED');assert.equal(x.agentLaunch,false);
  }
  const unknown=projectLocalCodexAttempt({input:localInput,observation:{...localBase,
    state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',agentLaunch:true,replayFixture:false,
    process:{status:'UNKNOWN',pid:123,builderPrincipal:'nilar\\codexsandboxoffline',
      builderSid:'S-1-5-21-1-2-3-1003'}},effectId:'u'});
  assert.equal(unknown.state,'LOCAL_ATTEMPT_UNKNOWN');assert.equal(unknown.taskPass,false);
});
test('T06m forged authority, malformed candidate and provider output fail closed',()=>{
  for(const value of [null,{},[],{...localReady,authorized:true},{...localReady,taskPass:true},
    {...localReady,agentLaunch:false},{...localReady,candidateTree:'bad'},
    {...localReady,process:{...localReady.process,exitCode:1}}]){
    const x=projectLocalCodexAttempt({input:localInput,observation:value,effectId:'forged'});
    assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.agentLaunch,false);assert.equal(x.taskPass,false);
  }
});
test('T06m effect calls only selected C and never upgrades denial',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-local-codex-attempt'];
  await assert.rejects(effect.handler({input:localInput,effectId:'missing-c'}),
    /SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED/);
  const x=await effect.handler({input:localInput,effectId:'local-c-denied',resources:{
    call:async(key,operation,arg)=>{
      assert.equal(key,'tally-local-codex');assert.equal(operation,'tally.runLocalCodexAttempt');
      assert.equal(arg.attemptId,localInput.attemptId);
      return {phase:'terminal',externalRef:'test',outcome:{ok:true,output:{...localBase,
        state:'NOT_ADMITTED',reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND',
        claimAccepted:false,admitted:false,agentLaunch:false}}};
    }}});
  assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.agentLaunch,false);assert.equal(x.taskPass,false);
});

const joinInput={issueRef:'https://github.com/kaicreator-mm/tally/issues/179',issueNumber:179,
  sourceSha:'ca09511d49488785c9ae8ff436c5db98f31b6714',joinId:'join:t06n:179:1',
  candidates:[
    {issueNumber:177,attemptId:'attempt:t06n:a:1',candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40)},
    {issueNumber:178,attemptId:'attempt:t06n:b:1',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}]};
const joinBase={issueRef:joinInput.issueRef,sourceSha:joinInput.sourceSha,joinId:joinInput.joinId,
  claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,agentLaunch:false,
  taskPass:false,releasePass:false};
const joinedObservation={...joinBase,state:'REVIEW_JOINED',reason:'REVIEWED_CANDIDATES_VERIFIED',
  replayFixture:false,joinedAt:'2026-10-30T02:00:00Z',
  candidates:[
    {attemptId:'attempt:t06n:a:1',issueRef:'https://github.com/kaicreator-mm/tally/issues/177',
      candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_A.md',
      candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),candidateRef:'refs/tally-candidates/attempt-t06n-a-1'},
    {attemptId:'attempt:t06n:b:1',issueRef:'https://github.com/kaicreator-mm/tally/issues/178',
      candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_B.md',
      candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40),candidateRef:'refs/tally-candidates/attempt-t06n-b-1'}]};
test('T06n reviewer Join intent is idempotent with its own rule and transition',()=>{
  const p=getPolicy(),e='TALLY_REVIEWED_CANDIDATES_JOIN_REQUESTED';
  assert.equal(p.intentBindings.joinReviewedCandidates.trigger.eventType,e);
  assert.equal(p.rules['rule:tally-reviewer-candidate-join'].eventType,e);
  const definition=p.intentBindings.joinReviewedCandidates.definition;
  assert.equal(definition.states[0].transitions.length,1);
  const transition=definition.states[0].transitions[0];
  assert.equal(transition.effectIntents[0].effectType,'effect:tally-reviewer-candidate-join');
  assert.equal(transition.effectIntents[0].idempotencyKeyFrom,undefined);
  assert.deepEqual(p.effectBindings['effect:tally-reviewer-candidate-join'].external,
    {permitTtlMs:3600000,await:'terminal'});
  assert.deepEqual(Object.keys(transition.effectIntents[0].inputFrom),
    ['issueRef','issueNumber','sourceSha','joinId','candidates']);
  for(const name of Object.keys(transition.effectIntents[0].inputFrom))
    assert.ok(Object.hasOwn(p.rules['rule:tally-reviewer-candidate-join'].payloadFromInput,name));
});
test('T06n join input alone never grants Reviewer authority and malformed joins skip C',async()=>{
  assert.equal(assessReviewedCandidateJoin(joinInput).state,'OBSERVED_INPUT_ONLY');
  for(const value of [null,{},'x',{...joinInput,sourceSha:'bad'},{...joinInput,joinId:'bad/id'},
    {...joinInput,issueNumber:178},{...joinInput,candidates:[]},
    {...joinInput,candidates:[joinInput.candidates[0]]},
    {...joinInput,candidates:[{...joinInput.candidates[0],issueNumber:999}]},
    {...joinInput,candidates:[{...joinInput.candidates[0],candidateCommit:'bad'}]}])
    assert.equal(assessReviewedCandidateJoin(value).state,'UNQUALIFIED_INPUT');
  let calls=0;
  const x=await getPolicy().effectBindings['effect:tally-reviewer-candidate-join'].handler({
    input:{...joinInput,joinId:'bad/id'},effectId:'bad-join',resources:{call:async()=>{calls++;}}});
  assert.equal(calls,0);assert.equal(x.agentLaunch,false);assert.equal(x.taskPass,false);
});
test('T06n ALM binding accepts only the additional exact Builder identities',()=>{
  const input={...joinInput,issueRef:'https://github.com/kaicreator-mm/tally/issues/200',issueNumber:200,
    candidates:[{...joinInput.candidates[0],issueNumber:197},{...joinInput.candidates[1],issueNumber:198}]};
  const assessed=assessReviewedCandidateJoin(input);
  assert.equal(assessed.state,'OBSERVED_INPUT_ONLY');assert.equal(assessed.admitted,false);
  assert.equal(assessReviewedCandidateJoin({...input,candidates:[input.candidates[0],
    {...input.candidates[1],issueNumber:199}]}).state,'UNQUALIFIED_INPUT');
});
test('T06n REVIEW_JOINED projection preserves exact candidates without Task PASS',()=>{
  const x=projectReviewedCandidateJoin({input:joinInput,observation:joinedObservation,effectId:'j1'});
  assert.equal(x.state,'REVIEW_JOINED');assert.equal(x.joinId,joinInput.joinId);
  assert.equal(x.candidates.length,2);
  assert.equal(x.candidates[0].candidateRelativePath,
    'experiments/v2/selected-business-build/CONCURRENT_A.md');
  assert.equal(x.candidates[1].issueNumber,178);
  assert.equal(x.authorized,false);assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
});
test('T06n replay and every reviewer denial stay distinct and never PASS',()=>{
  const replay=projectReviewedCandidateJoin({input:joinInput,observation:{...joinBase,
    state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY',agentLaunch:false,replayFixture:true},effectId:'jr'});
  assert.equal(replay.state,'REPLAY_COMPLETE');assert.equal(replay.agentLaunch,false);
  assert.equal(replay.replayFixture,true);assert.equal(replay.taskPass,false);
  for(const reason of ['CANDIDATE_NOT_FOUND','CANDIDATE_NOT_READY','CANDIDATE_STALE',
    'CANDIDATE_READBACK_MISMATCH','CANDIDATE_WRITESET_OVERLAP','REVIEWER_SELF_ASSERTION',
    'NATIVE_CLAIM_WITNESS_NOT_VERIFIED','CLAIM_REGISTRY_RESERVATION_MISMATCH']){
    const x=projectReviewedCandidateJoin({input:joinInput,observation:{...joinBase,
      state:'NOT_ADMITTED',reason,claimAccepted:false,admitted:false,agentLaunch:false},effectId:reason});
    assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.reason,reason);
    assert.equal(x.admitted,false);assert.equal(x.taskPass,false);
  }
});
test('T06n forged join authority and malformed positive observations fail closed',()=>{
  for(const value of [null,{},[],{...joinedObservation,authorized:true},
    {...joinedObservation,taskPass:true},{...joinedObservation,agentLaunch:true},
    {...joinedObservation,joinedAt:123},
    {...joinedObservation,candidates:[joinedObservation.candidates[0]]},
    {...joinedObservation,candidates:[{...joinedObservation.candidates[0],
      candidateTree:'bad'}]},
    {...joinedObservation,candidates:[{...joinedObservation.candidates[0],
      attemptId:'attempt:t06n:other:9'}]}]){
    const x=projectReviewedCandidateJoin({input:joinInput,observation:value,effectId:'forged-join'});
    assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.agentLaunch,false);assert.equal(x.taskPass,false);
  }
});
test('T06n join effect calls only selected C and never upgrades denial',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-reviewer-candidate-join'];
  await assert.rejects(effect.handler({input:joinInput,effectId:'missing-c'}),
    /SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED/);
  const x=await effect.handler({input:joinInput,effectId:'join-c-denied',resources:{
    call:async(key,operation,arg)=>{
      assert.equal(key,'tally-local-codex');assert.equal(operation,'tally.joinReviewedCandidates');
      assert.equal(arg.joinId,joinInput.joinId);
      return {phase:'terminal',externalRef:'test',outcome:{ok:true,output:{...joinBase,
        state:'NOT_ADMITTED',reason:'CANDIDATE_WRITESET_OVERLAP',
        claimAccepted:false,admitted:false,agentLaunch:false}}};
    }}});
  assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.reason,'CANDIDATE_WRITESET_OVERLAP');
  assert.equal(x.agentLaunch,false);assert.equal(x.taskPass,false);
});

const snapshotBase={issueRef:'https://github.com/kaicreator-mm/tally/issues/137',issueNumber:137,
  sourceSha:'ca09511d49488785c9ae8ff436c5db98f31b6714',observedAt:'2026-10-30T12:00:00.000Z'};
const snapshotPair=(workItem={},claimEvents={})=>({status:'SOURCE_PAIR_OBSERVED',sourceQualified:true,
  issueRef:snapshotBase.issueRef,sourceSha:snapshotBase.sourceSha,claimStatus:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  workItem:{issueRef:snapshotBase.issueRef,sourceSha:snapshotBase.sourceSha,
    reason:'CLAIM_NOT_VERIFIED',issueState:'open',workflowState:'state:ready',workItemType:'type:task',
    dependencyState:'CLEAR',openBlockerCount:0,claimStatus:'NOT_VERIFIED',
    authorized:false,admitted:false,taskPass:false,...workItem},
  claimEvents:{reason:'NO_DISPATCH_CLAIM_EVENTS',candidateCount:0,candidates:[],claimStatus:'NOT_VERIFIED',
    authorized:false,admitted:false,taskPass:false,releasePass:false,...claimEvents}});
const snapshotAttempt={issueRef:snapshotBase.issueRef,sourceSha:snapshotBase.sourceSha,
  attemptId:'attempt:t06m:snapshot:1',state:'LOCAL_CANDIDATE_READY',authorized:false,taskPass:false,releasePass:false,
  candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),
  process:{status:'EXITED',exitCode:0,startedAt:'2026-10-30T11:00:00.000Z',terminatedAt:'2026-10-30T11:01:00.000Z'}};
const snapshotJoin={issueRef:snapshotBase.issueRef,sourceSha:snapshotBase.sourceSha,
  joinId:'join:t06n:snapshot:1',state:'REVIEW_JOINED',authorized:false,taskPass:false,releasePass:false,
  candidates:[{attemptId:snapshotAttempt.attemptId,candidateTree:snapshotAttempt.candidateTree,
    candidateCommit:snapshotAttempt.candidateCommit},
    {attemptId:'attempt:t06n:snapshot:other',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}]};
const publicationInput={issueRef:'https://github.com/kaicreator-mm/tally/issues/197',issueNumber:197,
  sourceSha:'d'.repeat(40),attemptId:'attempt:alm:j05:a:1'};
const publicationObservation={state:'PUBLISHED_VERIFIED',reason:'GITHUB_BRANCH_AND_PR_READBACK_VERIFIED',
  issueRef:publicationInput.issueRef,sourceSha:publicationInput.sourceSha,attemptId:publicationInput.attemptId,
  claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,agentLaunch:false,
  repository:'kaicreator-mm/tally',branch:'tally/reviewed/attempt-alm-j05-a-1',baseRef:'version/v2',
  candidateCommit:'a'.repeat(40),candidateTree:'b'.repeat(40),pullNumber:203,
  pullRef:'https://github.com/kaicreator-mm/tally/pull/203',externalRef:'https://github.com/kaicreator-mm/tally/pull/203',
  taskPass:false,releasePass:false};
test('T06p publication is non-idempotent and input alone never grants a PR effect',async()=>{
  const effect=getPolicy().effectBindings['effect:tally-reviewed-candidate-publication'];
  assert.equal(effect.semantics,'non-idempotent');assert.deepEqual(effect.external,{permitTtlMs:3600000,await:'terminal'});
  assert.equal(assessCandidatePublication(publicationInput).state,'OBSERVED_INPUT_ONLY');
  for(const input of [{...publicationInput,issueNumber:196},{...publicationInput,sourceSha:'bad'},null])
    assert.equal(assessCandidatePublication(input).state,'UNQUALIFIED_INPUT');
  let calls=0;const denied=await effect.handler({input:{...publicationInput,attemptId:'bad/id'},effectId:'bad',
    resources:{call:async()=>{calls++;}}});
  assert.equal(calls,0);assert.equal(denied.state,'UNQUALIFIED_INPUT');
});
test('T06p exact PR receipt and UNKNOWN preserve authority boundaries',()=>{
  const x=projectCandidatePublication({input:publicationInput,observation:publicationObservation,effectId:'pub1'});
  assert.equal(x.state,'PUBLISHED_VERIFIED');assert.equal(x.pullNumber,203);assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
  const unknown=projectCandidatePublication({input:publicationInput,observation:{...publicationObservation,
    state:'PUBLICATION_UNKNOWN',reason:'GITHUB_REF_PUBLICATION_UNKNOWN',admitted:false,claimAccepted:false},effectId:'pub2'});
  assert.equal(unknown.state,'PUBLICATION_UNKNOWN');assert.equal(unknown.admitted,false);
  for(const observation of [{...publicationObservation,pullNumber:0},{...publicationObservation,authorized:true},
    {...publicationObservation,baseRef:'main'}])
    assert.equal(projectCandidatePublication({input:publicationInput,observation,effectId:'bad'}).state,'NOT_ADMITTED');
});
test('#137 open native blockers remain visible with actual query completion time',()=>{
  const x=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,
    expectedSourceSha:snapshotBase.sourceSha,observedAt:snapshotBase.observedAt,
    sourcePair:snapshotPair({dependencyState:'BLOCKED',openBlockerCount:1})});
  assert.equal(x.status,'WAIT_EXTERNAL');assert.equal(x.reason,'NATIVE_DEPENDENCIES_BLOCKED');
  assert.equal(x.observedAsOf,snapshotBase.observedAt);
  assert.equal(x.currentness,'OBSERVED_AT_QUERY_COMPLETION');
  assert.equal(x.openBlockerCount,1);assert.equal(x.ciState,'NOT_VERIFIED');
  assert.equal(x.releaseState,'NOT_VERIFIED');assert.equal(x.taskPass,false);
});
test('#137 no accepted CAS receipt stays awaiting Host and never launches Agent',()=>{
  const x=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,expectedSourceSha:snapshotBase.sourceSha,
    observedAt:snapshotBase.observedAt,sourcePair:snapshotPair()});
  assert.equal(x.status,'AWAITING_HOST_ADMISSION');assert.equal(x.reason,'NATIVE_READY_NO_ACCEPTED_CLAIM');
  assert.equal(x.claimState,'NOT_VERIFIED');assert.equal(x.kernelTaskState,'NOT_VERIFIED');
  assert.equal(x.agentAttemptState,'NOT_VERIFIED');assert.equal(x.metricComparability,'NOT_COMPARABLE');
  assert.equal(x.nextActions[0].kind,'REQUEST_SELECTED_HOST_ADMISSION');
});
test('#137 actual attempt and Reviewer Join observations stay layered without Task PASS',()=>{
  const x=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,expectedSourceSha:snapshotBase.sourceSha,
    observedAt:snapshotBase.observedAt,sourcePair:snapshotPair(),
    hostObservation:{attempt:snapshotAttempt,join:snapshotJoin}});
  assert.equal(x.status,'REVIEW_CANDIDATE_EVIDENCE');assert.equal(x.reason,'REVIEW_JOINED');
  assert.equal(x.agentAttemptState,'LOCAL_CANDIDATE_READY');assert.equal(x.kernelTaskState,'LOCAL_CANDIDATE_READY');
  assert.equal(x.hostState,'REVIEW_JOINED');assert.equal(x.reviewState,'REVIEW_JOINED');
  assert.equal(x.ciState,'NOT_VERIFIED');assert.equal(x.releaseState,'NOT_VERIFIED');
  assert.equal(x.taskPass,false);assert.equal(x.authorized,false);
});
test('#137 candidate without Reviewer Join requests review rather than completion',()=>{
  const x=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,expectedSourceSha:snapshotBase.sourceSha,
    observedAt:snapshotBase.observedAt,sourcePair:snapshotPair(),
    hostObservation:{attempt:snapshotAttempt}});
  assert.equal(x.status,'AWAITING_REVIEW');assert.equal(x.nextActions[0].kind,'REQUEST_REVIEWER_JOIN');
  assert.equal(x.reviewState,'NOT_VERIFIED');assert.equal(x.taskPass,false);
});
test('#137 wrong host source/attempt/timestamps are UNKNOWN rather than normalized PASS',()=>{
  for(const hostObservation of [
    {attempt:{...snapshotAttempt,state:'REVIEW_JOINED'}},
    {attempt:{...snapshotAttempt,issueRef:'https://github.com/kaicreator-mm/tally/issues/999'}},
    {attempt:{...snapshotAttempt,process:{...snapshotAttempt.process,startedAt:'bad'}}},
    {attempt:{...snapshotAttempt,process:{...snapshotAttempt.process,terminatedAt:'2026-10-30T10:00:00.000Z'}}},
    {attempt:snapshotAttempt,join:{...snapshotJoin,sourceSha:'0'.repeat(40)}},
    {attempt:snapshotAttempt,join:{...snapshotJoin,candidates:snapshotJoin.candidates.slice(1)}}
  ]){const x=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,expectedSourceSha:snapshotBase.sourceSha,
    observedAt:snapshotBase.observedAt,sourcePair:snapshotPair(),hostObservation});
    assert.equal(x.status,'UNKNOWN');assert.equal(x.reason,'HOST_OBSERVATION_NOT_VERIFIED');
    assert.equal(x.taskPass,false);}
});
test('#137 observation time stays honest and conflicting claims stay CONFLICT',()=>{
  const missing=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,expectedSourceSha:snapshotBase.sourceSha,
    observedAt:undefined,sourcePair:snapshotPair()});
  assert.equal(missing.status,'AWAITING_HOST_ADMISSION');assert.equal(missing.currentness,'AGE_NOT_VERIFIED');
  assert.equal(missing.observedAsOf,null);
  const noMillis=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,
    expectedSourceSha:snapshotBase.sourceSha,observedAt:'2026-10-30T12:00:00Z',sourcePair:snapshotPair()});
  assert.equal(noMillis.currentness,'OBSERVED_AT_QUERY_COMPLETION');
  const conflict=projectNativeTaskEffectiveSnapshot({expectedIssueRef:snapshotBase.issueRef,expectedSourceSha:snapshotBase.sourceSha,
    observedAt:snapshotBase.observedAt,sourcePair:snapshotPair({}, {
    reason:'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED',candidateCount:2,
    candidates:[{sourceSha:snapshotBase.sourceSha,sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',
      claimAccepted:false,dispatchId:'observed-comment-1'},
      {sourceSha:snapshotBase.sourceSha,sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',
        claimAccepted:false,dispatchId:'observed-comment-2'}]})});
  assert.equal(conflict.status,'CONFLICT');assert.equal(conflict.reason,'AMBIGUOUS_CLAIM_EVENT_CANDIDATES');
  assert.equal(conflict.claimState,'NOT_VERIFIED');assert.equal(conflict.taskPass,false);
});

// --- T210 J01b/J02: governed Product intake freeze + project admission ---

const intakeInput=()=>({issueRef:'https://github.com/kaicreator-mm/tally/issues/210',
  issueNumber:210,sourceSha:'b'.repeat(40),productDigest:'c'.repeat(64),
  projectTitle:'Tally v2 Intake'});
const frozenObservation=()=>({state:'FROZEN_COMPLETE',
  reason:'INDEPENDENT_REVIEW_FROZEN_IN_HOST_CAS',
  issueRef:'https://github.com/kaicreator-mm/tally/issues/210',sourceSha:'b'.repeat(40),
  productDigest:'c'.repeat(64),receiptRef:`receipt:sha256:${'c'.repeat(64)}`,frozen:true,
  claimGeneration:1,review:{author:'author:1',reviewer:'reviewer:1'},
  authorized:false,admitted:false,taskPass:false,releasePass:false});

test('#210 freeze screening refuses self-review, cross-repo and malformed digests',()=>{
  assert.equal(assessProductIntakeFreeze({...intakeInput(),author:'user:1',reviewer:'user:1'}).reason,
    'MISSING_SOURCE_IDENTITY');
  for(const change of [
    {issueRef:'https://github.com/kaicreator-mm/tally/issues/197'},
    {issueNumber:197},{sourceSha:'short'},{productDigest:'nothex'},
    {author:''},{reviewer:''},
  ]){
    const x=assessProductIntakeFreeze({...intakeInput(),author:'author:1',
      reviewer:'reviewer:1',...change});
    assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(x.frozen,false);
  }
  const ok=assessProductIntakeFreeze({...intakeInput(),author:'author:1',reviewer:'reviewer:1'});
  assert.equal(ok.state,'OBSERVED_INPUT_ONLY');assert.equal(ok.frozen,false);
  assert.equal(ok.reason,'REQUIRES_SEALED_PRODUCT_INTAKE_FREEZE');
});

test('#210 freeze projection accepts only the sealed FROZEN_COMPLETE shape',()=>{
  const x=projectProductIntakeFreeze({input:{...intakeInput(),author:'author:1',reviewer:'reviewer:1'},
    observation:frozenObservation(),effectId:'fx'});
  assert.equal(x.state,'FROZEN_COMPLETE');assert.equal(x.frozen,true);
  assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
  for(const change of [
    {frozen:false},{receiptRef:'receipt:sha256:deadbeef'},
    {review:{author:'reviewer:1',reviewer:'reviewer:1'}},
    {state:'NOT_FROZEN',reason:'PRODUCT_REVIEW_NOT_INDEPENDENT',frozen:false},
    {claimGeneration:2},
  ]){
    const y=projectProductIntakeFreeze({input:{...intakeInput(),author:'author:1',
      reviewer:'reviewer:1'},observation:{...frozenObservation(),...change},effectId:'fx'});
    assert.equal(y.frozen,false);assert.equal(y.taskPass,false);
  }
});

test('#210 intake screening enforces exact repo/issue allowlist and bounded title',()=>{
  for(const change of [
    {issueRef:'https://github.com/kaicreator-mm/tally/issues/197'},{issueNumber:197},
    {issueRef:'https://github.com/other/tally/issues/210'},
    {sourceSha:'bad'},{productDigest:'zz'},{projectTitle:''},{projectTitle:'x'.repeat(121)},
  ]){
    const x=assessProjectIntakeAdmission({...intakeInput(),...change});
    assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(x.admitted,false);
  }
  const ok=assessProjectIntakeAdmission(intakeInput());
  assert.equal(ok.state,'OBSERVED_INPUT_ONLY');
  assert.equal(ok.reason,'REQUIRES_SEALED_PROJECT_INTAKE_OBSERVATION');
});

test('#210 intake projection admits only verified publish+readback and honest NOT_RUN/UNKNOWN',()=>{
  const positive=()=>({state:'PROJECT_INTAKE_ADMITTED',
    reason:'PROJECT_INTAKE_PUBLISHED_AND_READBACK_VERIFIED',
    issueRef:'https://github.com/kaicreator-mm/tally/issues/210',sourceSha:'b'.repeat(40),
    productDigest:'c'.repeat(64),claimGeneration:1,claimAccepted:true,admitted:true,
    authorized:false,replayFixture:false,githubEffect:'PERFORMED_VERIFIED',
    repository:'kaicreator-mm/tally',
    project:{number:7,title:'Tally v2 Intake',
      url:'https://github.com/kaicreator-mm/tally/projects/7',itemCount:1},
    taskPass:false,releasePass:false});
  const x=projectProjectIntakeAdmission({input:intakeInput(),observation:positive(),effectId:'fx'});
  assert.equal(x.state,'PROJECT_INTAKE_ADMITTED');assert.equal(x.admitted,true);
  assert.equal(x.githubEffect,'PERFORMED_VERIFIED');assert.equal(x.project.number,7);
  assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
  const replay=projectProjectIntakeAdmission({input:intakeInput(),observation:{
    ...positive(),state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY',admitted:true,
    replayFixture:true,githubEffect:'NOT_RUN',project:null},effectId:'fx'});
  assert.equal(replay.state,'REPLAY_COMPLETE');assert.equal(replay.githubEffect,'NOT_RUN');
  for(const change of [
    {githubEffect:'NOT_RUN',state:'NOT_ADMITTED',reason:'PRODUCT_FREEZE_NOT_VERIFIED',admitted:false},
    {githubEffect:'UNKNOWN',state:'PROJECT_INTAKE_UNKNOWN',reason:'GITHUB_PUBLISH_UNKNOWN'},
    {repository:'other/tally'},{project:{...positive().project,number:0}},
    {project:{...positive().project,url:'https://github.com/kaicreator-mm/tally/x'}},
    {admitted:false},
  ]){
    const y=projectProjectIntakeAdmission({input:intakeInput(),
      observation:{...positive(),...change},effectId:'fx'});
    assert.equal(y.admitted,false,JSON.stringify(change));
    assert.equal(y.taskPass,false);
  }
});

test('#210 intake effect dispatches sealed resource only for qualified input',async()=>{
  const p=getPolicy();let calls=0;
  const handler=p.effectBindings['effect:tally-project-intake-admission'].handler;
  const x=await handler({input:{...intakeInput(),issueNumber:197},effectId:'fx',
    resources:{call:async()=>{calls++;return {phase:'terminal',outcome:{ok:true,output:{}}};}}});
  assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(calls,0);
  const y=await handler({input:intakeInput(),effectId:'fx',
    resources:{call:async(key,op,arg)=>{
      calls++;
      assert.equal(key,'tally-local-codex');assert.equal(op,'tally.hostProjectIntake');
      assert.equal(arg.issueNumber,210);assert.equal(arg.effectId,'fx');
      return {phase:'terminal',outcome:{ok:true,output:{
        state:'NOT_ADMITTED',reason:'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE',
        issueRef:'https://github.com/kaicreator-mm/tally/issues/210',
        sourceSha:'b'.repeat(40),productDigest:'c'.repeat(64),claimGeneration:1,
        admitted:false,authorized:false,githubEffect:'NOT_RUN',
        taskPass:false,releasePass:false}}};}}});
  assert.equal(y.state,'NOT_ADMITTED');assert.equal(y.reason,'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE');
  assert.equal(calls,1);
});

test('#210 freeze idempotency key renders within Kernel grammar',()=>{
  const pattern=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
  const p=getPolicy();
  const transition=p.workflowDefinition.states[0].transitions.find(
    t=>t.transitionKey==='freeze-product-intake');
  const rendered=transition.effectIntents[0].idempotencyKeyFrom.template
    .replace(/\{([A-Za-z0-9_]+)\}/g,(_m,name)=>String(intakeInput()[name]??''));
  assert.ok(pattern.test(rendered));assert.ok(!rendered.includes('/'));
});

// --- T219 J08b/J08c: authorized maintenance Issue + measured Task Learning ---

const maintInput=()=>({issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
  issueNumber:219,sourceSha:'d'.repeat(40),candidateDigest:'e'.repeat(64)});
const maintPublished=()=>({state:'MAINTENANCE_ISSUE_PUBLISHED',
  reason:'MAINTENANCE_ISSUE_PUBLISHED_AND_READBACK_VERIFIED',
  issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
  sourceSha:'d'.repeat(40),candidateDigest:'e'.repeat(64),claimGeneration:1,
  claimAccepted:true,admitted:true,authorized:false,replayFixture:false,
  githubEffect:'PERFORMED_VERIFIED',repository:'kaicreator-mm/tally',
  duplicate:false,alreadyPresent:false,historicalReleaseMutation:false,
  issue:{number:302,title:'[maintenance] Post-release regression',
    url:'https://github.com/kaicreator-mm/tally/issues/302'},
  taskPass:false,releasePass:false});
const learningInput=()=>({issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
  issueNumber:219,sourceSha:'d'.repeat(40),learningId:'f'.repeat(64),
  decision:'ADOPTED'});
const learningRecorded=()=>({state:'LEARNING_RECORDED',
  reason:'LEARNING_DECISION_RECORDED_IN_HOST_CAS',
  issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
  sourceSha:'d'.repeat(40),learningId:'f'.repeat(64),decision:'ADOPTED',
  claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
  replayFixture:false,githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
  noAgentRanking:true,qualityGainClaimed:false,taskPass:false,releasePass:false});

test('#219 publication screening enforces the exact #219 allowlist and digest',()=>{
  for(const change of [
    {issueRef:'https://github.com/kaicreator-mm/tally/issues/210'},{issueNumber:210},
    {issueRef:'https://github.com/other/tally/issues/219'},
    {sourceSha:'bad'},{candidateDigest:'zz'},
  ]){
    const x=assessMaintenanceIssuePublication({...maintInput(),...change});
    assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(x.admitted,false);
  }
  for(const empty of [null,undefined,{},[]]){
    const x=assessMaintenanceIssuePublication(empty);
    assert.equal(x.state,'UNQUALIFIED_INPUT');
  }
  const ok=assessMaintenanceIssuePublication(maintInput());
  assert.equal(ok.state,'OBSERVED_INPUT_ONLY');
  assert.equal(ok.reason,'REQUIRES_SEALED_MAINTENANCE_ISSUE_PUBLICATION');
});

test('#219 publication projection admits only verified publish+readback shapes',()=>{
  const x=projectMaintenanceIssuePublication({input:maintInput(),
    observation:maintPublished(),effectId:'fx'});
  assert.equal(x.state,'MAINTENANCE_ISSUE_PUBLISHED');
  assert.equal(x.admitted,true);assert.equal(x.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(x.issue.number,302);assert.equal(x.duplicate,false);
  assert.equal(x.historicalReleaseMutation,false);
  assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
  const replay=projectMaintenanceIssuePublication({input:maintInput(),
    observation:{...maintPublished(),state:'REPLAY_COMPLETE',
      reason:'REPLAY_FIXTURE_ONLY',admitted:true,replayFixture:true,
      githubEffect:'NOT_RUN',issue:null,repository:null,duplicate:false,
      alreadyPresent:false},effectId:'fx'});
  assert.equal(replay.state,'REPLAY_COMPLETE');assert.equal(replay.githubEffect,'NOT_RUN');
  const reconciled=projectMaintenanceIssuePublication({input:maintInput(),
    observation:{...maintPublished(),
      reason:'MAINTENANCE_ISSUE_RECONCILED_AFTER_LOST_ACK'},effectId:'fx'});
  assert.equal(reconciled.reason,'MAINTENANCE_ISSUE_RECONCILED_AFTER_LOST_ACK');
  for(const change of [
    {state:'NOT_ADMITTED',reason:'CANDIDATE_REVIEW_NOT_VERIFIED',admitted:false,
      claimAccepted:false,githubEffect:'NOT_RUN',issue:null,repository:null,
      duplicate:false,alreadyPresent:false},
    {state:'PUBLICATION_UNKNOWN',reason:'GITHUB_PUBLISH_UNKNOWN',
      githubEffect:'UNKNOWN',issue:null,repository:null,duplicate:false,
      alreadyPresent:false},
    {repository:'other/tally'},
    {issue:{...maintPublished().issue,number:0}},
    {issue:{...maintPublished().issue,url:'https://github.com/kaicreator-mm/tally/x'}},
    {duplicate:true},{historicalReleaseMutation:true},{admitted:false},
    {reason:'MADE_UP_REASON'},
  ]){
    const y=projectMaintenanceIssuePublication({input:maintInput(),
      observation:{...maintPublished(),...change},effectId:'fx'});
    assert.equal(y.admitted,false,JSON.stringify(change));
    assert.equal(y.taskPass,false);
  }
});

test('#219 publication effect dispatches sealed resource only for qualified input',async()=>{
  const p=getPolicy();let calls=0;
  const handler=p.effectBindings['effect:tally-maintenance-issue-publication'].handler;
  const x=await handler({input:{...maintInput(),issueNumber:210},effectId:'fx',
    resources:{call:async()=>{calls++;return {phase:'terminal',outcome:{ok:true,output:{}}};}}});
  assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(calls,0);
  const y=await handler({input:maintInput(),effectId:'fx',
    resources:{call:async(key,op,arg)=>{
      calls++;
      assert.equal(key,'tally-local-codex');assert.equal(op,'tally.publishMaintenanceIssue');
      assert.equal(arg.issueNumber,219);assert.equal(arg.candidateDigest,'e'.repeat(64));
      return {phase:'terminal',outcome:{ok:true,output:{
        state:'PUBLICATION_UNKNOWN',reason:'GITHUB_PUBLISH_UNKNOWN',
        issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
        sourceSha:'d'.repeat(40),candidateDigest:'e'.repeat(64),claimGeneration:1,
        claimAccepted:true,admitted:true,authorized:false,replayFixture:false,
        githubEffect:'UNKNOWN',issue:null,historicalReleaseMutation:false,
        duplicate:false,taskPass:false,releasePass:false}}};}}});
  assert.equal(y.state,'PUBLICATION_UNKNOWN');assert.equal(y.reason,'GITHUB_PUBLISH_UNKNOWN');
  assert.equal(calls,1);
});

test('#219 publication idempotency key renders within Kernel grammar',()=>{
  const pattern=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
  const p=getPolicy();
  const transition=p.workflowDefinition.states[0].transitions.find(
    t=>t.transitionKey==='publish-maintenance-issue');
  const rendered=transition.effectIntents[0].idempotencyKeyFrom.template
    .replace(/\{([A-Za-z0-9_]+)\}/g,(_m,name)=>String(maintInput()[name]??''));
  assert.ok(pattern.test(rendered));assert.ok(!rendered.includes('/'));
});

test('#219 learning screening requires decision enum and bounded outcome',()=>{
  for(const change of [
    {issueNumber:197},{sourceSha:'short'},{learningId:'zz'},
    {decision:'RANK_AGENTS'},{decision:undefined},
  ]){
    const x=assessTaskLearningOutcome({...learningInput(),...change});
    assert.equal(x.state,'UNQUALIFIED_INPUT',JSON.stringify(change));
  }
  for(const empty of [null,undefined,{},[]]){
    const x=assessTaskLearningOutcome(empty);
    assert.equal(x.state,'UNQUALIFIED_INPUT');
  }
  for(const outcome of [
    {attested:'yes',observedAt:'2026-11-02T10:00:00.000Z',note:'x'},
    {attested:false,observedAt:'not-a-time',note:'x'},
    {attested:false,observedAt:'2026-11-02T10:00:00.000Z',note:''},
    {attested:false,observedAt:'2026-11-02T10:00:00.000Z',note:'x'.repeat(513)},
    'garbage',
  ]){
    const x=assessTaskLearningOutcome({...learningInput(),outcome});
    assert.equal(x.state,'UNQUALIFIED_INPUT',JSON.stringify(outcome));
  }
  const decisionOnly=assessTaskLearningOutcome(learningInput());
  assert.equal(decisionOnly.state,'OBSERVED_INPUT_ONLY');
  assert.equal(decisionOnly.reason,'REQUIRES_SEALED_TASK_LEARNING_RECORD');
  const withOutcome=assessTaskLearningOutcome({...learningInput(),outcome:{
    attested:false,observedAt:'2026-11-02T10:00:00.000Z',note:'no measurable change'}});
  assert.equal(withOutcome.state,'OBSERVED_INPUT_ONLY');
});

test('#219 learning projection records use/reject and honest outcomes only',()=>{
  const x=projectTaskLearningOutcome({input:learningInput(),
    observation:learningRecorded(),effectId:'fx'});
  assert.equal(x.state,'LEARNING_RECORDED');assert.equal(x.admitted,true);
  assert.equal(x.measuredEffect,'NOT_MEASURED');assert.equal(x.noAgentRanking,true);
  assert.equal(x.qualityGainClaimed,false);assert.equal(x.githubEffect,'NOT_RUN');
  assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
  const attested=projectTaskLearningOutcome({input:learningInput(),
    observation:{...learningRecorded(),
      reason:'LEARNING_OUTCOME_RECORDED_IN_HOST_CAS',
      measuredEffect:'ATTESTED_OUTCOME_ONLY',
      outcome:{attested:true,measuredEffect:'ATTESTED_OUTCOME_ONLY'}},effectId:'fx'});
  assert.equal(attested.measuredEffect,'ATTESTED_OUTCOME_ONLY');
  assert.equal(attested.outcome.attested,true);
  const replay=projectTaskLearningOutcome({input:learningInput(),
    observation:{...learningRecorded(),state:'REPLAY_COMPLETE',
      reason:'REPLAY_FIXTURE_ONLY',replayFixture:true,decision:'ADOPTED',
      measuredEffect:'NOT_MEASURED'},effectId:'fx'});
  assert.equal(replay.state,'REPLAY_COMPLETE');assert.equal(replay.githubEffect,'NOT_RUN');
  for(const change of [
    {state:'NOT_ADMITTED',reason:'LEARNING_PROVENANCE_NOT_VERIFIED',admitted:false,
      claimAccepted:false},
    {decision:'RANK_AGENTS'},{noAgentRanking:false},{qualityGainClaimed:true},
    {githubEffect:'PERFORMED_VERIFIED'},{measuredEffect:'QUALITY_GAIN_PLUS_40'},
    {reason:'MADE_UP_REASON'},{admitted:false},
  ]){
    const y=projectTaskLearningOutcome({input:learningInput(),
      observation:{...learningRecorded(),...change},effectId:'fx'});
    assert.equal(y.admitted,false,JSON.stringify(change));
    assert.equal(y.taskPass,false);
  }
});

test('#219 learning effect dispatches sealed resource only for qualified input',async()=>{
  const p=getPolicy();let calls=0;
  const handler=p.effectBindings['effect:tally-task-learning-outcome'].handler;
  const x=await handler({input:{...learningInput(),decision:'RANK_AGENTS'},
    effectId:'fx',resources:{call:async()=>{calls++;return {phase:'terminal'};}}});
  assert.equal(x.state,'UNQUALIFIED_INPUT');assert.equal(calls,0);
  const y=await handler({input:{...learningInput(),outcomeAttested:false,
      outcomeObservedAt:'2026-11-02T10:00:00.000Z',outcomeNote:'independent follow-up'},
    effectId:'fx',
    resources:{call:async(key,op,arg)=>{
      calls++;
      assert.equal(key,'tally-local-codex');assert.equal(op,'tally.recordTaskLearningOutcome');
      assert.equal(arg.learningId,'f'.repeat(64));
      assert.equal(arg.outcomeAttested,false);
      assert.equal(arg.outcomeObservedAt,'2026-11-02T10:00:00.000Z');
      assert.equal(arg.outcomeNote,'independent follow-up');
      return {phase:'terminal',outcome:{ok:true,output:{
        state:'LEARNING_RECORDED',reason:'LEARNING_OUTCOME_RECORDED_IN_HOST_CAS',
        issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
        sourceSha:'d'.repeat(40),learningId:'f'.repeat(64),decision:'ADOPTED',
        claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
        replayFixture:false,githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
        outcome:{attested:false,measuredEffect:'NOT_MEASURED'},
        noAgentRanking:true,qualityGainClaimed:false,
        taskPass:false,releasePass:false}}};}}});
  assert.equal(y.state,'LEARNING_RECORDED');
  assert.equal(y.outcome.measuredEffect,'NOT_MEASURED');
  assert.equal(calls,1);
  // The decision phase (null outcome scalars) stays a qualified dispatch.
  const z=await handler({input:{...learningInput(),outcomeAttested:null,
      outcomeObservedAt:null,outcomeNote:null},
    effectId:'fx',
    resources:{call:async()=>{calls++;
      return {phase:'terminal',outcome:{ok:true,output:{
        state:'LEARNING_RECORDED',reason:'LEARNING_DECISION_RECORDED_IN_HOST_CAS',
        issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
        sourceSha:'d'.repeat(40),learningId:'f'.repeat(64),decision:'ADOPTED',
        claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
        replayFixture:false,githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
        noAgentRanking:true,qualityGainClaimed:false,
        taskPass:false,releasePass:false}}};}}});
  assert.equal(z.state,'LEARNING_RECORDED');
  assert.equal(z.measuredEffect,'NOT_MEASURED');
  assert.equal(calls,2);
});

test('#219 learning idempotency key renders within Kernel grammar',()=>{
  const pattern=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
  const p=getPolicy();
  const transition=p.workflowDefinition.states[0].transitions.find(
    t=>t.transitionKey==='record-task-learning-outcome');
  const rendered=transition.effectIntents[0].idempotencyKeyFrom.template
    .replace(/\{([A-Za-z0-9_]+)\}/g,(_m,name)=>String(learningInput()[name]??''));
  assert.ok(pattern.test(rendered));assert.ok(!rendered.includes('/'));
});
