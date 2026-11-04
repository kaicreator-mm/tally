import test from 'node:test';
import assert from 'node:assert/strict';
import {verifyGuidedSnapshot,verifyGuidedPayload,readGuidedStep,submitGuidedStep}
  from './guided-form-port.mjs';
import {validateStepSpec} from '../../apps/tally-web/src/features/forms/schema.ts';
import {buildSubmitPayload} from '../../apps/tally-web/src/features/forms/engine.ts';
const sha='a'.repeat(40),scope={projectId:'org/tally',versionId:'version/v2'};
const label={zh:'内容',en:'Content'};
const step={journeyId:'journey01',stepId:'goal',revision:1,
  title:{zh:'目标',en:'Goal'},fields:[{id:'goal',kind:'text',required:true,
    label,maxLength:100}],
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

test('selected Business step is re-read before any dispatch side effect',async()=>{
  let reads=0,dispatches=0;
  const host={readCurrentStep:async()=>reads++===0?snapshot:
    {...snapshot,step:{...step,revision:2}},
    dispatchStep:async()=>{dispatches++;return{ok:true,nextStep:null}}};
  const result=await submitGuidedStep(host,scope,sha,form);
  assert.equal(result.reason,'GUIDED_STEP_CHANGED_BEFORE_DISPATCH');
  assert.equal(dispatches,0);
});
test('reordered-key equivalent snapshots are not falsely denied as stale',async()=>{
  let dispatches=0;
  // Semantically identical step, every object's keys inserted in reverse order.
  const reorderedStep={actions:['next'],fields:[
      {maxLength:100,label:{en:'Content',zh:'内容'},required:true,
        kind:'text',id:'goal'}],revision:1,stepId:'goal',
    journeyId:'journey01',title:{en:'Goal',zh:'目标'}};
  assert.equal(JSON.stringify(reorderedStep)===JSON.stringify(step),false);
  let first=true;
  const host={readCurrentStep:async()=>first?(first=false,snapshot):
      {...snapshot,step:reorderedStep},
    dispatchStep:async()=>{dispatches++;return{ok:true,nextStep:null}}};
  const result=await submitGuidedStep(host,scope,sha,form);
  assert.deepEqual(result,{connected:true,ok:true,nextStep:null});
  assert.equal(dispatches,1);
});

test('required fields must be present and nonblank before Host dispatch',()=>{
  assert.equal(verifyGuidedPayload({...form,answers:{}},snapshot),false);
  assert.equal(verifyGuidedPayload({...form,answers:{goal:''}},snapshot),false);
  assert.equal(verifyGuidedPayload({...form,answers:{goal:'   '}},snapshot),false);
  assert.equal(verifyGuidedPayload({...form,answers:{goal:null}},snapshot),false);
  const booleanStep={...snapshot,step:{...step,
    fields:[{id:'confirm',kind:'boolean',required:true}]}};
  assert.equal(verifyGuidedPayload({...form,answers:{confirm:false}},booleanStep),true);
});
test('Host step and next-step projections drop unrecognized extension fields',async()=>{
  const privateValue='internal-only-value';
  const hostileStep={...step,internalField:privateValue,
    title:{...step.title,internalField:privateValue},
    fields:[{...step.fields[0],internalField:privateValue,
      label:{...label,internalField:privateValue}}],
    actions:['next','back']};
  const read=await readGuidedStep({readCurrentStep:async()=>({...snapshot,
    internalField:privateValue,step:hostileStep})},scope,sha);
  assert.equal(read.connected,true);
  assert.deepEqual(Object.keys(read.step).sort(),
    ['actions','fields','journeyId','revision','stepId','title']);
  assert.deepEqual(Object.keys(read.step.fields[0]).sort(),
    ['id','kind','label','maxLength','required']);
  assert.equal(JSON.stringify(read).includes(privateValue),false);
  const submitted=await submitGuidedStep({
    readCurrentStep:async()=>snapshot,
    dispatchStep:async()=>({ok:true,nextStep:hostileStep})
  },scope,sha,form);
  assert.equal(submitted.connected,true);
  assert.deepEqual(Object.keys(submitted.nextStep).sort(),
    ['actions','fields','journeyId','revision','stepId','title']);
  assert.equal(JSON.stringify(submitted).includes(privateValue),false);
});
test('initial and next steps retain the complete bounded FormStep contract',async()=>{
  const option={id:'one',label,eligible:true,recommended:true,
    ineligibleReason:null,privateValue:'drop-me'};
  const base={label,required:false,hint:label,disabledReason:null,
    visibleWhen:{fieldId:'goal',equals:'show',privateValue:'drop-me'},
    privateValue:'drop-me'};
  const fields=[
    {id:'goal',kind:'text',required:true,label,maxLength:100},
    {id:'area',kind:'textArea',...base,maxLength:400},
    {id:'choice',kind:'choice',...base,options:[option]},
    {id:'multi',kind:'multi',...base,options:[option]},
    {id:'bool',kind:'boolean',...base},
    {id:'number',kind:'number',...base,min:0,max:10,integerOnly:true},
    {id:'confirm',kind:'confirm',...base,statement:label},
    {id:'other',kind:'choiceWithOther',...base,
      options:[option],otherMaxLength:50},
    {id:'limited',kind:'multiLimited',...base,
      options:[option],minSelected:0,maxSelected:1},
  ];
  const richStep={...step,fields,intro:label,actions:['next','back','pause'],
    privateValue:'drop-me'};
  assert.ok(validateStepSpec(richStep));
  const richSnapshot={...snapshot,step:richStep};
  const first=await readGuidedStep({readCurrentStep:async()=>richSnapshot},scope,sha);
  assert.equal(first.connected,true);
  assert.ok(validateStepSpec(first.step));
  assert.equal(JSON.stringify(first).includes('drop-me'),false);
  assert.deepEqual(first.step.actions,['next','back','pause']);
  const nextStep={...richStep,stepId:'next',revision:2};
  const result=await submitGuidedStep({readCurrentStep:async()=>richSnapshot,
    dispatchStep:async()=>({ok:true,nextStep})},scope,sha,form);
  assert.equal(result.connected,true);
  assert.ok(validateStepSpec(result.nextStep));
  assert.equal(JSON.stringify(result).includes('drop-me'),false);
});
test('missing required field configuration and unknown transitions fail closed',async()=>{
  for(const badStep of [
    {...step,fields:[{id:'goal',kind:'text',required:true}]},
    {...step,fields:[{...step.fields[0],maxLength:undefined}]},
    {...step,actions:['next','internal-action']},
  ]){
    const observed=await readGuidedStep({readCurrentStep:async()=>
      ({...snapshot,step:badStep})},scope,sha);
    assert.equal(observed.reason,'GUIDED_SOURCE_UNQUALIFIED');
  }
});
test('numeric runtime identifiers fail closed on initial and next steps',async()=>{
  const variants=[
    {...step,journeyId:123}, {...step,stepId:123},
    {...step,fields:[{...step.fields[0],id:123}]},
    {...step,fields:[{...step.fields[0],visibleWhen:{fieldId:123,equals:true}}]},
    {...step,fields:[{id:'c',kind:'choice',required:true,label,
      options:[{id:123,label,eligible:true}]}]},
  ];
  for(const invalidStep of variants){
    const initial=await readGuidedStep({readCurrentStep:async()=>
      ({...snapshot,step:invalidStep})},scope,sha);
    assert.equal(initial.reason,'GUIDED_SOURCE_UNQUALIFIED');
    const next=await submitGuidedStep({readCurrentStep:async()=>snapshot,
      dispatchStep:async()=>({ok:true,nextStep:invalidStep})},scope,sha,form);
    assert.equal(next.reason,'GUIDED_HOST_RESULT_UNQUALIFIED');
  }
});
test('malformed next-step field metadata cannot be reported as connected success',async()=>{
  const invalidNext={...step,stepId:'next',
    fields:[{id:'choice',kind:'choice',required:true,label,
      options:[{id:'one',label,eligible:true},
        {id:'two',label,eligible:true,privateValue:'drop-me'}],
      privateValue:'drop-me'}]};
  const host={readCurrentStep:async()=>snapshot,
    dispatchStep:async()=>({ok:true,nextStep:invalidNext})};
  const valid=await submitGuidedStep(host,scope,sha,form);
  assert.equal(valid.connected,true);
  assert.ok(validateStepSpec(valid.nextStep));
  assert.equal(JSON.stringify(valid).includes('drop-me'),false);
  const malformed={...invalidNext,fields:[{...invalidNext.fields[0],
    options:[{id:'one',label,eligible:'yes'}]}]};
  const denied=await submitGuidedStep({readCurrentStep:async()=>snapshot,
    dispatchStep:async()=>({ok:true,nextStep:malformed})},scope,sha,form);
  assert.equal(denied.reason,'GUIDED_HOST_RESULT_UNQUALIFIED');
});
test('engine payload with hidden or disabled required fields and optional null dispatches',async()=>{
  const branch={...step,fields:[
    {id:'mode',kind:'choice',required:true,label,
      options:[{id:'no',label,eligible:true},{id:'yes',label,eligible:true}]},
    {id:'detail',kind:'text',required:true,label,maxLength:100,
      visibleWhen:{fieldId:'mode',equals:'yes'}},
    {id:'disabled',kind:'text',required:true,label,maxLength:100,
      disabledReason:label},
    {id:'optional',kind:'text',required:false,label,maxLength:100},
  ]};
  const answers={mode:'no'};
  const built=buildSubmitPayload({step:branch,answers,errors:{},status:'editing'});
  assert.deepEqual(built.answers,{mode:'no',optional:null});
  const payload={...built,sourceSha:sha};
  const current={...snapshot,step:branch};
  let dispatches=0;
  const result=await submitGuidedStep({readCurrentStep:async()=>current,
    dispatchStep:async()=>{dispatches++;return{ok:true,nextStep:null}}},
    scope,sha,payload);
  assert.equal(result.ok,true);
  assert.equal(dispatches,1);
  for(const extra of [{detail:'unseen'}, {disabled:'not allowed'}])
    assert.equal(verifyGuidedPayload({...payload,
      answers:{...payload.answers,...extra}},current),false);
});
