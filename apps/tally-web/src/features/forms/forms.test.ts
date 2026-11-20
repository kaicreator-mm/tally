import test from 'node:test';
import assert from 'node:assert/strict';
import {validateStepSpec,type FormStep,type Field} from './schema.ts';
import {visibleFields,validateStep,validateField,restoreDraft,
  serializeDraft,buildSubmitPayload,type EngineState} from './engine.ts';

const L={zh:'中',en:'EN'};
const options=[
  {id:'a',label:{zh:'甲',en:'A'},eligible:true,ineligibleReason:null},
  {id:'b',label:{zh:'乙',en:'B'},eligible:false,
    ineligibleReason:{zh:'证据不足',en:'missing evidence'}},
];
const nineFields:Field[]=[
  {id:'f1',kind:'text',label:L,required:true,maxLength:20},
  {id:'f2',kind:'textArea',label:L,required:false,maxLength:200},
  {id:'f3',kind:'choice',label:L,required:true,options},
  {id:'f4',kind:'multi',label:L,required:false,options},
  {id:'f5',kind:'boolean',label:L,required:false},
  {id:'f6',kind:'number',label:L,required:true,min:1,max:10,integerOnly:true},
  {id:'f7',kind:'confirm',label:L,required:true,statement:{zh:'确认',en:'confirm'}},
  {id:'f8',kind:'choiceWithOther',label:L,required:false,options,otherMaxLength:30},
  {id:'f9',kind:'multiLimited',label:L,required:false,options,minSelected:1,maxSelected:1},
];
const step:FormStep={stepId:'s1',journeyId:'j1',title:L,fields:nineFields,
  actions:['next','back','pause','resume','cancel'],revision:1};

test('schema accepts the nine bounded kinds and rejects forgeries',()=>{
  assert.ok(validateStepSpec(step));
  assert.equal(validateStepSpec({...step,stepId:'../evil'}),null);
  assert.equal(validateStepSpec({...step,revision:0}),null);
  assert.equal(validateStepSpec({...step,actions:['nuke']}),null);
  assert.equal(validateStepSpec({...step,fields:[]}),null);
  assert.equal(validateStepSpec({...step,fields:[
    ...nineFields.slice(0,8),
    {id:'f9',kind:'multiLimited',label:L,required:false,options,
     minSelected:2,maxSelected:1}]}),null); // min>max refused
  assert.equal(validateStepSpec(null),null);
  assert.equal(validateStepSpec('x'),null);
});
test('v1.4.1 nine-control regression: every kind validates typed bounds',()=>{
  const opts=new Set(['a','b']);
  const cases:[
    Field,undefined|string|string[]|boolean, string|null][]=[
    [nineFields[0],undefined,'required'],
    [nineFields[0],'ok',null],
    [nineFields[0],'x'.repeat(21),'too_long'],
    [nineFields[1],'multi\nline',null],
    [nineFields[2],undefined,'required'],
    [nineFields[2],'a',null],
    [nineFields[2],'zzz','unknown_option'],
    [nineFields[3],['a'],null],
    [nineFields[3],['a','zzz'],'unknown_option'],
    [nineFields[4],true,null],
    [nineFields[4],'yes','type'],
    [nineFields[5],undefined,'required'],
    [nineFields[5],'7',null],
    [nineFields[5],'7.5','not_integer'],
    [nineFields[5],'11','too_large'],
    [nineFields[5],'0','too_small'],
    [nineFields[5],'abc','not_a_number'],
    [nineFields[6],false,'confirm_required'],
    [nineFields[6],true,null],
    [nineFields[7],'a',null],
    [nineFields[7],'other:', 'other_empty'],
    [nineFields[7],'other:好的',null],
    [nineFields[7],'x'.repeat(31).slice(0,31)&&'other:'+ 'y'.repeat(31),'too_long'],
    [nineFields[8],[], 'too_few'],
    [nineFields[8],['a'],null],
    [nineFields[8],['a','b'],'too_many'],
  ];
  for(const [field,value,expected] of cases)
    assert.equal(validateField(field,value as never,opts),expected,
      `${field.id}:${JSON.stringify(value)}`);
});
test('visibility rules branch on answers; hidden fields skip validation',()=>{
  const branched:FormStep={...step,fields:[
    {id:'mode',kind:'choice',label:L,required:true,options},
    {id:'detail',kind:'text',label:L,required:true,maxLength:50,
      visibleWhen:{fieldId:'mode',equals:'a'}},
    {id:'legacy',kind:'text',label:L,required:true,maxLength:50,
      visibleWhen:{fieldId:'legacy',equals:'x'}}, // self-reference: hidden
  ]};
  assert.deepEqual(visibleFields(branched,{}).map(f=>f.id),['mode']);
  assert.deepEqual(visibleFields(branched,{mode:'a'}).map(f=>f.id),['mode','detail']);
  // detail required only when visible
  assert.equal(validateStep(branched,{}).detail,undefined);
  assert.equal(validateStep(branched,{mode:'a'}).detail,'required');
  assert.equal(validateStep(branched,{mode:'b'}).detail,undefined);
  // self-referencing field never becomes visible
  assert.equal(visibleFields(branched,{legacy:'x'}).some(f=>f.id==='legacy'),false);
});
test('disabled fields are excluded from step validation and submit payload',()=>{
  const withDisabled:FormStep={...step,fields:[
    {id:'f1',kind:'text',label:L,required:true,maxLength:20,
      disabledReason:{zh:'等待证据',en:'awaiting evidence'}},
    {id:'f6',kind:'number',label:L,required:true,min:1,max:10,integerOnly:true},
  ]};
  const answers={f6:'5'};
  assert.deepEqual(validateStep(withDisabled,answers),{});
  const payload=buildSubmitPayload({step:withDisabled,answers,errors:{},
    status:'editing'});
  assert.deepEqual(payload.answers,{f6:'5'});
  assert.equal('f1' in payload.answers,false);
});
test('draft round-trip restores exact issued step; stale/foreign refused',()=>{
  const state:EngineState={step,answers:{f1:'草稿'},errors:{},status:'editing'};
  const raw=serializeDraft(state);
  assert.deepEqual(restoreDraft(step,raw),{f1:'草稿'});
  // revision bump invalidates the draft
  const revised:FormStep={...step,revision:2};
  assert.equal(restoreDraft(revised,raw),null);
  // foreign journey/step refused
  assert.equal(restoreDraft({...step,stepId:'s2'},raw),null);
  assert.equal(restoreDraft(step,'not-json'),null);
  assert.equal(restoreDraft(step,null),null);
  // unknown field ids and exotic value shapes are dropped, not smuggled
  const smuggle=JSON.stringify({journeyId:'j1',stepId:'s1',revision:1,
    answers:{f1:'ok',evil:'<script>',f4:[1,2],f5:'yes'}});
  assert.deepEqual(restoreDraft(step,smuggle),{f1:'ok'});
});
test('submit payload carries exact issued identity; validation gate refuses',()=>{
  const core:FormStep={...step,fields:[nineFields[0],nineFields[2],
    nineFields[5],nineFields[6]]};
  const answers={f1:'值',f3:'a',f6:'3',f7:true};
  const payload=buildSubmitPayload({step:core,answers,errors:{},status:'editing'});
  assert.deepEqual(payload,{journeyId:'j1',stepId:'s1',revision:1,
    answers:{f1:'值',f3:'a',f6:'3',f7:true}});
  assert.throws(()=>buildSubmitPayload({step:core,answers:{},errors:{},status:'editing'}),
    /VALIDATION_FAILED/);
  assert.throws(()=>buildSubmitPayload(
    {step:core,answers,errors:{},status:'submitted'}),/ENGINE_STATE_INVALID/);
});
test('XSS refusal: control characters rejected; payload stays plain strings',()=>{
  const bad='a\u0000b';
  const opts=new Set(['a','b']);
  assert.equal(validateField(nineFields[0],bad,opts),'control_chars');
  const core:FormStep={...step,fields:[nineFields[0],nineFields[2],
    nineFields[5],nineFields[6]]};
  const answers={f1:'<b>x</b>',f3:'a',f6:'3',f7:true};
  const payload=buildSubmitPayload({step:core,answers,errors:{},status:'editing'});
  // Value passes through as an inert string; rendering escapes by construction.
  assert.equal(payload.answers.f1,'<b>x</b>');
  assert.equal(typeof payload.answers.f1,'string');
});
