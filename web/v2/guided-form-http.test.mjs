import test from 'node:test';
import assert from 'node:assert/strict';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
// The clean version/v2 baseline deliberately does not carry the E0-lineage
// v2GuidedFormHost UI-lane wiring into journey-server (see the
// createAlmServer seam comment: "deliberately NOT carried into this clean
// port"). The guided form port itself (guided-form-port.mjs) and its full
// positive/negative seam contract are covered by guided-form-port.test.mjs;
// the HTTP routes, CSRF key binding and 409 mappings belong to the E/#264
// journey-server wiring and must arrive with it. Until then this file pins
// the fail-closed integration contract of the current baseline: the guided
// routes are absent, the v2GuidedFormHost option is inert and no positive
// guided JSON can leak through the ALM server.
const sha='d'.repeat(40);
const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',
  tallyBPath:'/tally',githubToken:'nonpublicfixture',
  currentSourceSha:sha,workRoot:'/tmp/tally-v2-guided-guard'});
const url='/api/v2/guided/step?projectId=org%2Ftally&versionId=version%2Fv2';
const input={projectId:'org/tally',versionId:'version/v2',sourceSha:sha,
  journeyId:'journey01',stepId:'goal',revision:1,answers:{goal:'Build'}};
const post=origin=>fetch(origin+'/api/v2/guided/submit',{method:'POST',
  headers:{origin,'content-type':'application/json',
    'x-tally-action-key':'k'.repeat(48)},body:JSON.stringify(input)});
// Fail-closed body contract: an unwired guided route must never answer with
// positive guided JSON (no connected flag, nothing JSON-parseable).
const assertNotPositiveJson=async response=>{
  const body=await response.text();
  assert.equal(body.includes('connected'),false);
  let threw=false;try{JSON.parse(body);}catch{threw=true;}
  assert.equal(threw,true);
  return body;
};
test('guided step route stays absent and the Host option stays inert',async()=>{
  let reads=0;
  const host={readCurrentStep:async()=>{reads++;return null;},
    dispatchStep:async()=>{throw new Error('guided submit must never dispatch');}};
  const app=createAlmServer(config,{v2GuidedFormHost:host});
  const {origin}=await app.listen();
  try{
    const step=await fetch(origin+url);
    assert.equal(step.status,404);
    await assertNotPositiveJson(step);
    assert.equal(reads,0);
    // The ignored option must not degrade the existing read-only seam.
    const snapshot=await fetch(origin+'/api/snapshot');
    assert.equal(snapshot.status,200);
    assert.equal((await snapshot.text()).includes('"snapshot"'),true);
  }finally{await app.close();}
});
test('guided submit route stays absent with zero dispatcher effects',async()=>{
  let dispatches=0;
  const host={readCurrentStep:async()=>null,
    dispatchStep:async()=>{dispatches++;return {ok:true};}};
  const app=createAlmServer(config,{v2GuidedFormHost:host});
  const {origin,actionKey}=await app.listen();
  try{
    assert.equal(typeof actionKey,'string');
    const response=await post(origin);
    assert.equal(response.status,404);
    await assertNotPositiveJson(response);
    assert.equal(dispatches,0);
  }finally{await app.close();}
});
