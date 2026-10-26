import test from 'node:test';
import assert from 'node:assert/strict';
import {getPolicy,assessTaskEvidence,ABI_VERSION,PACKAGE_KIND,INTENT_TYPES} from '../src/index.mjs';
const good={issueRef:'https://github.com/kaicreator-mm/tally/issues/108',sourceSha:'a'.repeat(40),claimRef:'github:claim:108',dependenciesComplete:true};
test('real DH public Business ABI identifiers',()=>{
  assert.equal(ABI_VERSION,'v08-business-abi/1');assert.equal(PACKAGE_KIND,'business');
  assert.deepEqual([...INTENT_TYPES],['inspectTaskReadiness']);
});
test('Policy has one bounded B intent and never declares GitHub external effect',()=>{
  const p=getPolicy();assert.deepEqual(Object.keys(p.intentBindings),['inspectTaskReadiness']);
  assert.deepEqual(Object.keys(p.effectBindings),['effect:tally-readiness-observation']);
  assert.ok(!('external' in p.effectBindings['effect:tally-readiness-observation']));
  assert.ok(!('resource' in p.effectBindings['effect:tally-readiness-observation']));
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
  for(const i of [null,{},'x',{...good,sourceSha:'bad'},{...good,claimRef:''},{...good,issueRef:'https://evil.test/1'}]){
    assert.equal(assessTaskEvidence(i).admitted,false);
    assert.equal(assessTaskEvidence(i).taskPass,false);
  }
});
test('B effect only returns advisory, no remote side effects',async()=>{
  const p=getPolicy();
  const x=await p.effectBindings['effect:tally-readiness-observation'].handler({input:good,effectId:'e-1'});
  assert.equal(x.admitted,false);assert.equal(x.taskPass,false);assert.equal(x.effectId,'e-1');
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
  assert.equal(Object.keys(p.effectBindings).length,1);
  assert.equal(p.workflowDefinition.states[0].transitions[0].targetState,'open');
});
