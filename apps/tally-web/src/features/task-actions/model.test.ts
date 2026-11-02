import test from 'node:test';
import assert from 'node:assert/strict';
import {taskActionsFor,taskActionRequest} from './model.ts';
const source={repository:'org/tally',sourceSha:'a'.repeat(40),
  currentness:'CURRENT',readback:'VERIFIED'};
const catalog={connected:true,source,versionId:'version/v2',actions:[
  {id:'start-1',kind:'START_AGENT',revision:1,label:'Start',taskIssue:42,state:'ELIGIBLE'},
  {id:'review-1',kind:'REVIEW_REPAIR',revision:2,label:'Review',taskIssue:42,state:'BLOCKED'}]};
test('only selected current version actions for one Task are listed',()=>{
  assert.equal(taskActionsFor(catalog,'org/tally','version/v2',42)?.length,2);
  assert.equal(taskActionsFor(catalog,'evil/repo','version/v2',42),null);
  assert.equal(taskActionsFor(catalog,'org/tally','version/v3',42),null);
});
test('only Host-eligible exact revision produces a dispatch request',()=>{
  const actions=taskActionsFor(catalog,'org/tally','version/v2',42)!;
  const request=taskActionRequest(catalog,'org/tally','version/v2',actions[0]);
  assert.deepEqual(request,{projectId:'org/tally',versionId:'version/v2',
    sourceSha:source.sourceSha,actionId:'start-1',revision:1});
  assert.equal(taskActionRequest(catalog,'org/tally','version/v2',actions[1]),null);
});
test('stale source, duplicate or unqualified actions do not create requests',()=>{
  assert.equal(taskActionsFor({...catalog,source:{...source,currentness:'STALE'}},
    'org/tally','version/v2',42),null);
  assert.equal(taskActionsFor({...catalog,actions:[catalog.actions[0],catalog.actions[0]]},
    'org/tally','version/v2',42),null);
  assert.equal(taskActionRequest({...catalog,actions:[]},'org/tally','version/v2',
    catalog.actions[0]),null);
});
