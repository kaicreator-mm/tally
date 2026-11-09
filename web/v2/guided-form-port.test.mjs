import test from 'node:test';
import assert from 'node:assert/strict';
import {verifyGuidedSnapshot,verifyGuidedPayload,readGuidedStep,submitGuidedStep}
  from './guided-form-port.mjs';
const sha='a'.repeat(40),scope={projectId:'org/tally',versionId:'version/v2'};
const step={journeyId:'journey01',stepId:'goal',revision:1,
  title:{zh:'目标',en:'Goal'},fields:[{id:'goal',kind:'text',required:true}],
  actions:['next']};
const snapshot={connected:true,projectId:'org/tally',versionId:'version/v2',
  sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED',step};
const form={sourceSha:sha,journeyId:'journey01',stepId:'goal',revision:1,
  answers:{goal:'Ship product'}};
test('current qualified selected step can be read',async()=>{
  assert.equal(verifyGuidedSnapshot(snapshot,scope,sha),true);
  assert.equal((await readGuidedStep({readCurrentStep:async()=>snapshot},scope,sha)).connected,true);
});
test('no Host stays unconnected',async()=>{
  assert.equal((await readGuidedStep({},scope,sha)).reason,'GUIDED_HOST_NOT_CONNECTED');
});
test('step refuses stale source, cross-project, revision and unknown fields',()=>{
  assert.equal(verifyGuidedSnapshot({...snapshot,sourceSha:'b'.repeat(40)},scope,sha),false);
  assert.equal(verifyGuidedSnapshot({...snapshot,projectId:'attacker/repo'},scope,sha),false);
  assert.equal(verifyGuidedPayload(form,snapshot),true);
  assert.equal(verifyGuidedPayload({...form,revision:2},snapshot),false);
  assert.equal(verifyGuidedPayload({...form,answers:{goal:'ok',permit:true}},snapshot),false);
});
test('unknown/oversized form payloads rejected',()=>{
  assert.equal(verifyGuidedPayload({...form,answers:{goal:'x'.repeat(5000)}},snapshot),false);
  assert.equal(verifyGuidedPayload({...form,token:'secret'},snapshot),false);
});
test('no selected Host dispatcher prevents effect',async()=>{
  assert.equal((await submitGuidedStep({readCurrentStep:async()=>snapshot},scope,sha,form)).reason,
    'GUIDED_HOST_DISPATCH_UNAVAILABLE');
});
test('real selected Host dispatch can issue next step without frontend granting PASS',async()=>{
  let dispatchCount=0;
  const host={readCurrentStep:async()=>snapshot,dispatchStep:async()=>{dispatchCount++;return{ok:true,nextStep:null}}};
  assert.deepEqual(await submitGuidedStep(host,scope,sha,form),
    {connected:true,ok:true,nextStep:null});
  assert.equal(dispatchCount,1);
});
test('rejected stale form does not reach Host dispatcher',async()=>{
  let called=false;
  const host={readCurrentStep:async()=>snapshot,dispatchStep:async()=>{called=true;return {ok:true,nextStep:null}}};
  const denied=await submitGuidedStep(host,scope,sha,{...form,stepId:'other'});
  assert.equal(denied.reason,'GUIDED_STEP_STALE_OR_INVALID');
  assert.equal(called,false);
});
