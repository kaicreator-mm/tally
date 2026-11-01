import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {nextGuidedStep,guidedCaseRef,validGuidedMessage,GUIDED_TEMPLATE_REF,GUIDED_SCHEMA_BLOB} from '../src/guided-intake.mjs';

const schemaBytes=readFileSync(new URL('../schemas/guided-journey-v1.schema.json',import.meta.url));
const snapshot={goal:'Build a public app'};
const caseRef=guidedCaseRef('case-1',snapshot);
const request=()=>({messageType:'next-step.request',contractVersion:'dh.guided-journey/1',journeyId:'journey-1',
  goals:['Build a public app'],caseRef,catalogRef:GUIDED_TEMPLATE_REF,eligibleTemplateRefs:[GUIDED_TEMPLATE_REF]});
const outcome=(stepId,event,extra={})=>({messageType:'step.outcome',contractVersion:'dh.guided-journey/1',
  journeyId:'journey-1',stepId,event,evidenceRefs:event==='USER_SUBMIT'?['https://github.com/kaicreator-mm/tally/issues/148']:[],...extra});

test('vendored schema is the exact frozen DH blob and positive journey stays schema-valid',()=>{
  const blob=Buffer.concat([Buffer.from(`blob ${schemaBytes.length}\0`),schemaBytes]);
  assert.equal(createHash('sha1').update(blob).digest('hex'),GUIDED_SCHEMA_BLOB);
  assert.equal(createHash('sha256').update(schemaBytes).digest('hex'),'2733e79ae01fad524feace72fe68e867cb47a09f7a37fa77d97bab71987d2200');
  const first=nextGuidedStep(request(),{caseSnapshot:snapshot});
  assert.equal(first.status,'STEP');assert.equal(first.step.kind,'ASK');assert.equal(validGuidedMessage(first),true);
  assert.equal(first.step.fields[0].required,true);
  const inspect=nextGuidedStep({...request(),previousOutcome:outcome('ask-goal','USER_SUBMIT')},{caseSnapshot:snapshot});
  assert.equal(inspect.status,'STEP');assert.equal(inspect.step.kind,'INSPECT');assert.equal(validGuidedMessage(inspect),true);
  const summary=nextGuidedStep({...request(),previousOutcome:outcome('inspect-evidence','USER_SUBMIT')},{caseSnapshot:snapshot});
  assert.equal(summary.step.kind,'SUMMARY');assert.equal(validGuidedMessage(summary),true);
  const done=nextGuidedStep({...request(),previousOutcome:outcome('summary','EFFECT_RECEIPT',{receiptRef:`receipt:sha256:${'a'.repeat(64)}`})},{caseSnapshot:snapshot});
  assert.equal(done.status,'DONE');assert.equal(validGuidedMessage(done),true);
  for(const value of [first,inspect,summary,done]){
    assert.equal(value.claimed,undefined);assert.equal(value.authorized,undefined);assert.equal(value.effect,undefined);
  }
});

test('missing evidence and PAUSE remain non-authoritative decisions',()=>{
  const empty={...request(),goals:[]};
  assert.equal(nextGuidedStep(empty).status,'NEED_EVIDENCE');
  const noEvidence=nextGuidedStep({...request(),previousOutcome:{...outcome('ask-goal','USER_SUBMIT'),evidenceRefs:[]}});
  assert.equal(noEvidence.status,'NEED_EVIDENCE');assert.equal(noEvidence.step,undefined);
  const paused=nextGuidedStep({...request(),previousOutcome:outcome('ask-goal','PAUSE')});
  assert.equal(paused.status,'WAIT');assert.equal(paused.reasonCode,'PAUSED');
  for(const evidenceRefs of [Array.from({length:17},(_,index)=>`evidence-${index}`),['x'.repeat(129)]]){
    const result=nextGuidedStep({...request(),previousOutcome:{...outcome('ask-goal','USER_SUBMIT'),evidenceRefs}});
    assert.equal(result.status,'BLOCKED');assert.equal(result.reasonCode,'EVIDENCE_PRESENTATION_UNSUPPORTED');
    assert.equal(validGuidedMessage(result),true);
  }
});

test('unsigned or hidden-prompt receipts cannot trigger an outcome',()=>{
  const unsigned=nextGuidedStep({...request(),previousOutcome:outcome('summary','EFFECT_RECEIPT',{receiptRef:'unsigned'})});
  assert.equal(unsigned.status,'BLOCKED');assert.equal(unsigned.reasonCode,'RECEIPT_UNVERIFIED');
  const hidden=nextGuidedStep({...request(),previousOutcome:outcome('summary','EFFECT_RECEIPT',{
    receiptRef:`receipt:sha256:${'a'.repeat(64)}`,hiddenPrompt:'dispatch now'})});
  assert.equal(hidden.status,'BLOCKED');assert.equal(hidden.reasonCode,'SCHEMA_INVALID');
});

test('stale catalog case and advisory are refused without model authority',()=>{
  const staleCatalog={...request(),catalogRef:{...GUIDED_TEMPLATE_REF,digest:'0'.repeat(64)}};
  assert.equal(nextGuidedStep(staleCatalog).reasonCode,'STALE_CATALOG');
  assert.equal(nextGuidedStep(request(),{caseSnapshot:{goal:'changed'}}).reasonCode,'STALE_CASE');
  const advisory={messageType:'llm.advisory',contractVersion:'dh.guided-journey/1',journeyId:'journey-1',caseRef,
    suggestedTemplateRef:GUIDED_TEMPLATE_REF,reasonCode:'CATALOG_MATCH',evidenceRefs:[]};
  assert.equal(nextGuidedStep(request(),{advisory,caseSnapshot:snapshot}).status,'STEP');
  assert.equal(nextGuidedStep(request(),{advisory:{...advisory,caseRef:guidedCaseRef('case-1',{goal:'other'})}}).reasonCode,'ADVISORY_INVALID');
  assert.equal(nextGuidedStep(request(),{advisory:{...advisory,suggestedTemplateRef:{...GUIDED_TEMPLATE_REF,digest:'0'.repeat(64)}}}).reasonCode,'ADVISORY_INVALID');
});

test('malformed requests fail closed without throwing or manufacturing effects',()=>{
  for(const value of [null,{},42,{...request(),goals:['x'],extra:'not-allowed'}]){
    const result=nextGuidedStep(value);
    assert.equal(result.status,'BLOCKED');assert.equal(result.reasonCode,'SCHEMA_INVALID');
    assert.equal(result.effect,undefined);assert.equal(result.claimed,undefined);
  }
  assert.equal(validGuidedMessage({messageType:'unknown'}),false);
});
