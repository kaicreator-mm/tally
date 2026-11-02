import test from 'node:test';
import assert from 'node:assert/strict';
import {workbenchActionGate} from './human-action-gate.ts';
import type {WorkbenchSnapshot,WorkAction} from './model.ts';
const sha='a'.repeat(40),repository='org/tally';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const work:WorkAction={id:'review-260',taskId:'260',projectId:repository,
  versionId:'version/v2',title:'Review candidate',kind:'REVIEW',state:'PENDING',
  ownerRole:'independent-reviewer',sourceRef:'https://github.com/org/tally/issues/260',
  reason:null,updatedAt:null};
const inbox:WorkbenchSnapshot={connected:true,source,actions:[work]};
const catalog={connected:true,source,versionId:'version/v2',actions:[{
  id:'review-task-260',kind:'REVIEW_REPAIR',revision:1,taskIssue:260,
  state:'ELIGIBLE' as const,label:'审核候选'}]};
test('only matching Host-qualified reviewer action can render enabled',()=>{
  const r=workbenchActionGate(inbox,catalog,work);
  assert.equal(r.enabled,true);
  assert.equal(r.actionId,'review-task-260');
  assert.equal(r.revision,1);
});
test('missing Host, unknown currentness and SHA mismatch stay disabled',()=>{
  for(const c of [null,{...catalog,connected:false},
    {...catalog,source:{...source,sourceSha:'b'.repeat(40)}},
    {...catalog,source:{...source,currentness:'STALE'}}]){
    assert.equal(workbenchActionGate(inbox,c,work).enabled,false);
  }
});
test('other project/version and blocked action cannot be clicked',()=>{
  assert.equal(workbenchActionGate(inbox,catalog,{...work,projectId:'evil/repo'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,versionId:'version/v3'},work).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,{...work,state:'BLOCKED'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,actions:[{
    ...catalog.actions[0],state:'BLOCKED'}]},work).enabled,false);
});
test('native blocker and unassigned approval never become fake permitted action',()=>{
  assert.equal(workbenchActionGate(inbox,catalog,{...work,kind:'BLOCKER'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,{...work,kind:'APPROVAL'}).enabled,false);
});
test('ambiguous Host action or invalid revision refuses action',()=>{
  assert.equal(workbenchActionGate(inbox,{...catalog,
    actions:[catalog.actions[0],catalog.actions[0]]},work).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,
    actions:[{...catalog.actions[0],revision:0}]},work).enabled,false);
});
