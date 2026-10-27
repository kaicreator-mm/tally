import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import Ajv2020 from 'ajv/dist/2020.js';
import {catalogRef,templateRefs,nextProjectIntakeStep} from '../src/index.mjs';

const schema=JSON.parse(readFileSync(new URL('../vendor/guided-journey-v1.schema.json',import.meta.url),'utf8'));
const canonical=new Ajv2020({strict:true,strictRequired:false,validateFormats:false}).compile(schema);
const VERSION='dh.guided-journey/1';
const caseRef={snapshotId:'case-issue-148',digest:'a'.repeat(64)};
const initial=()=>({messageType:'next-step.request',contractVersion:VERSION,
  journeyId:'tally-v2-product-intake-148',goals:['v2 scoped feature'],
  caseRef,catalogRef,eligibleTemplateRefs:templateRefs});
const outcome=(stepId,event,selectedOptionIds=[],evidenceRefs=[])=>({
  messageType:'step.outcome',contractVersion:VERSION,journeyId:initial().journeyId,
  stepId,event,selectedOptionIds,evidenceRefs
});
const progress=(previousOutcome,opts={})=>
  nextProjectIntakeStep({...initial(),previousOutcome},opts);
const assertNoAuthority=result=>{
  assert.equal(result.authorized,false);
  assert.equal(result.admitted,false);
  assert.equal(result.taskPass,false);
  assert.equal(result.releasePass,false);
  if(result.decision)assert.equal(canonical(result.decision),true,
    JSON.stringify(canonical.errors));
};
test('actual frozen v0.10.3 schema validates offline Tally project intake step sequence',()=>{
  const goal=nextProjectIntakeStep(initial());
  assert.equal(goal.status,'STEP');
  assert.equal(goal.decision.step.stepId,'goal');
  assert.equal(goal.decision.step.kind,'ASK');
  assertNoAuthority(goal);
  const scope=progress(outcome('goal','USER_SUBMIT',['new-feature']));
  assert.equal(scope.status,'STEP');
  assert.equal(scope.decision.step.stepId,'scope');
  assertNoAuthority(scope);
  const ask=progress(outcome('scope','USER_SUBMIT',['bounded'],['github:issue:148']));
  assert.equal(ask.status,'NEED_EVIDENCE');
  assert.equal(ask.decision.status,'NEED_EVIDENCE');
  assertNoAuthority(ask);
  const review=progress(outcome('scope','USER_SUBMIT',['bounded'],['github:issue:148']),
    {sourceEvidenceRefs:['github:issue:148']});
  assert.equal(review.status,'STEP');
  assert.equal(review.decision.step.stepId,'review');
  assert.equal(review.decision.step.kind,'SUMMARY');
  assertNoAuthority(review);
  const submitted=progress(outcome('review','USER_SUBMIT'));
  assert.equal(submitted.status,'WAIT');
  assert.equal(submitted.reason,'INDEPENDENT_PRODUCT_REVIEW_REQUIRED');
  assertNoAuthority(submitted);
});
test('LLM advisory can match an approved template but cannot alter allowed steps or execute',()=>{
  const adv={messageType:'llm.advisory',contractVersion:VERSION,
    journeyId:initial().journeyId,caseRef,
    suggestedTemplateRef:templateRefs[1],
    reasonCode:'SCOPED_SUGGESTION',evidenceRefs:[]};
  const request={...initial(),previousOutcome:outcome('goal','USER_SUBMIT',['bug-fix'])};
  const correct=nextProjectIntakeStep(request,{advisory:adv});
  assert.equal(correct.decision.step.stepId,'scope');
  assert.equal(correct.advisoryMatched,true);
  assert.equal(correct.advisoryUsedForAuthority,false);
  assertNoAuthority(correct);
  const wrong=nextProjectIntakeStep(request,{advisory:{
    ...adv,suggestedTemplateRef:templateRefs[2]}});
  assert.equal(wrong.decision.step.stepId,'scope');
  assert.equal(wrong.advisoryMatched,false);
  assert.deepEqual(wrong.decision,correct.decision);
  assertNoAuthority(wrong);
});
test('catalog and eligible StepTemplate refs cannot be replaced by a model',()=>{
  const original=nextProjectIntakeStep(initial());
  for(const q of [
    {...initial(),catalogRef:{...catalogRef,digest:'f'.repeat(64)}},
    {...initial(),eligibleTemplateRefs:[{...templateRefs[0],digest:'f'.repeat(64)}]},
    {...initial(),eligibleTemplateRefs:[templateRefs[2]]}
  ]){
    const x=nextProjectIntakeStep(q);
    assert.equal(x.status,'BLOCKED');
    assertNoAuthority(x);
  }
  assert.equal(nextProjectIntakeStep(initial()).decision.step.templateRef.digest,
    original.decision.step.templateRef.digest);
});
test('previous event has to be eligible and choice must be canonical',()=>{
  for(const prev of [
    outcome('goal','USER_SUBMIT',['invented-action']),
    outcome('goal','USER_SUBMIT',['new-feature','bug-fix']),
    outcome('goal','EFFECT_RECEIPT',[],['github:issue:148']),
    outcome('scope','USER_SUBMIT',['other']),
    {...outcome('goal','USER_SUBMIT',['new-feature']),journeyId:'cross-case'},
    {...outcome('scope','USER_SUBMIT',['bounded']),receiptRef:'self-approved'}
  ]){
    const x=progress(prev);
    assert.equal(x.status,'BLOCKED');
    assertNoAuthority(x);
  }
});
test('pause and back return non-authoritative next decisions; no unsolicited Done',()=>{
  const paused=progress(outcome('scope','PAUSE'));
  assert.equal(paused.status,'WAIT');
  assert.equal(paused.reason,'USER_PAUSED');
  assertNoAuthority(paused);
  const back=progress(outcome('review','BACK'));
  assert.equal(back.decision.step.stepId,'scope');
  assertNoAuthority(back);
  assert.equal(back.decision.status,'STEP');
});
test('host evidence is not inferred from a user Outcome; stale or duplicate evidence rejected',()=>{
  const prev=outcome('scope','USER_SUBMIT',['bounded'],['github:issue:148']);
  for(const sourceEvidenceRefs of [[],['github:issue:149'],['github:issue:148','github:issue:148']]){
    const r=progress(prev,{sourceEvidenceRefs});
    assert.notEqual(r.decision?.step?.stepId,'review');
    assertNoAuthority(r);
  }
  const good=progress(prev,{sourceEvidenceRefs:['github:issue:148']});
  assert.equal(good.decision.step.stepId,'review');
  assertNoAuthority(good);
});
test('malformed request and hostile getters fail closed without secret text',()=>{
  const bad=nextProjectIntakeStep({...initial(),caseRef:{...caseRef,digest:'not-valid'}});
  assert.equal(bad.status,'BLOCKED');assertNoAuthority(bad);
  const secret='TALLY_USER_SECRET_DO_NOT_ECHO_123';
  const trick={...initial(),get eligibleTemplateRefs(){throw Error(secret);}};
  const result=nextProjectIntakeStep(trick);
  assert.equal(result.status,'BLOCKED');
  assert.equal(JSON.stringify(result).includes(secret),false);
  assertNoAuthority(result);
  const nullResult=nextProjectIntakeStep(null);
  assert.equal(nullResult.status,'BLOCKED');
  assertNoAuthority(nullResult);
});
