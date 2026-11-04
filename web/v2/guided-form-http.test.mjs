import test from 'node:test';
import assert from 'node:assert/strict';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {validateStepSpec} from '../../apps/tally-web/src/features/forms/schema.ts';
const sha='d'.repeat(40);
const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',
  tallyBPath:'/tally',githubToken:'nonpublicfixture',
  currentSourceSha:sha,workRoot:'/tmp/tally-v2-guided-guard'});
const scope={projectId:'org/tally',versionId:'version/v2'};
const label={zh:'内容',en:'Content'};
const step={journeyId:'journey01',stepId:'goal',revision:1,
  title:{zh:'目标',en:'Goal'},fields:[{id:'goal',kind:'text',required:true,
    label,maxLength:100}],
  actions:['next']};
const nextStep={...step,stepId:'decision',revision:2,
  fields:[{id:'choice',kind:'choice',required:true,label,
    options:[{id:'one',label,eligible:true,privateValue:'drop-me'}]}],
  privateValue:'drop-me'};
const current={connected:true,projectId:scope.projectId,versionId:scope.versionId,
  sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED',step};
const url='/api/v2/guided/step?projectId=org%2Ftally&versionId=version%2Fv2';
const post=(origin,key,payload)=>fetch(origin+'/api/v2/guided/submit',{
  method:'POST',headers:{origin,'content-type':'application/json',
    'x-tally-action-key':key},body:JSON.stringify(payload)});
const input={...scope,sourceSha:sha,journeyId:'journey01',
  stepId:'goal',revision:1,answers:{goal:'Build'}};
test('new UI form port does not invent source and refuses CSRF/stale',async()=>{
  const app=createAlmServer(config);const {origin,actionKey}=await app.listen();
  try{
    const missing=await fetch(origin+url).then(r=>r.json());
    assert.equal(missing.connected,false);
    const csrf=await post(origin,'wrong',input);assert.equal(csrf.status,403);
    const stale=await post(origin,actionKey,{...input,sourceSha:'a'.repeat(40)});
    assert.equal(stale.status,409);
    const denied=await post(origin,actionKey,input);
    assert.equal(denied.status,409);
    assert.equal((await denied.json()).reason,'GUIDED_HOST_NOT_CONNECTED');
  }finally{await app.close();}
});
test('selected Host-issued exact step is the only submit candidate',async()=>{
  let attempts=0;
  const host={readCurrentStep:async()=>current,
    dispatchStep:async()=>{attempts++;return {ok:true,nextStep};}};
  const app=createAlmServer(config,{v2GuidedFormHost:host});
  const {origin,actionKey}=await app.listen();
  try{
    const shown=await fetch(origin+url).then(r=>r.json());
    assert.equal(shown.connected,true);
    assert.equal(shown.step.stepId,'goal');
    assert.ok(validateStepSpec(shown.step));
    assert.equal(shown.step.fields[0].maxLength,100);
    assert.equal(shown.csrfKey,actionKey);
    const bad=await post(origin,actionKey,{...input,answers:{permit:true}});
    assert.equal(bad.status,409);assert.equal(attempts,0);
    const good=await post(origin,actionKey,input);
    assert.equal(good.status,200);
    const submitted=await good.json();
    assert.equal(submitted.ok,true);assert.equal(attempts,1);
    assert.ok(validateStepSpec(submitted.nextStep));
    assert.equal(submitted.nextStep.fields[0].options[0].id,'one');
    assert.equal(JSON.stringify(submitted).includes('drop-me'),false);
  }finally{await app.close();}
});
