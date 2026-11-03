import test from 'node:test';
import assert from 'node:assert/strict';
import {workbenchActionGate} from './human-action-gate.ts';
import type {WorkbenchSnapshot,WorkAction} from './model.ts';
const sha='a'.repeat(40),repository='org/tally';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const work:WorkAction={id:'review-260',taskId:'260',projectId:repository,
  versionId:'version/v2',title:'Review candidate',kind:'REVIEW',state:'PENDING',
  ownerRole:'independent-reviewer',sourceRef:'https://github.com/org/tally/issues/260',
  reason:null,updatedAt:null,hostActionId:'review-task-260',
  hostActionRevision:1,hostActionKind:'REVIEW_REPAIR'};
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
    ...catalog.actions[0],state:'BLOCKED' as const}]},work).enabled,false);
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

test('only exact Host-issued identity matching the work-item kind can enable',()=>{
  const exact={...work,hostActionId:'host-issued-1',
    hostActionRevision:7,hostActionKind:'REVIEW_REPAIR'};
  const c={...catalog,actions:[{id:'host-issued-1',kind:'REVIEW_REPAIR',
    revision:7,taskIssue:260,state:'ELIGIBLE' as const,label:'Review candidate'}]};
  const r=workbenchActionGate(inbox,c,exact);
  assert.equal(r.enabled,true);
  assert.equal(r.actionId,'host-issued-1');
  assert.equal(r.revision,7);
  assert.equal(workbenchActionGate(inbox,{...c,actions:[{
    ...c.actions[0],revision:8}]},exact).enabled,false);
  assert.equal(workbenchActionGate(inbox,c,{...exact,
    hostActionKind:'START_AGENT'}).enabled,false);
});
test('identity-free Task/category match is never a dispatch-ready operation',()=>{
  const identityFree={...work,hostActionId:null,hostActionRevision:null,
    hostActionKind:null};
  const result=workbenchActionGate(inbox,catalog,identityFree);
  assert.equal(result.enabled,false);
  assert.equal(result.reason,'HOST_ACTION_IDENTITY_NOT_BOUND');
  assert.equal(workbenchActionGate(inbox,catalog,{...work,
    hostActionRevision:undefined}).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,{...work,
    hostActionId:'other-operation'}).enabled,false);
});
test('Recovery can request only its exact Host-issued START_AGENT action',()=>{
  const recovery={...work,kind:'RECOVERY' as const,
    hostActionId:'restart-260',hostActionRevision:3,
    hostActionKind:'START_AGENT'};
  assert.equal(workbenchActionGate(inbox,catalog,recovery).enabled,false);
  const c={...catalog,actions:[{id:'restart-260',kind:'START_AGENT',
    revision:3,taskIssue:260,state:'ELIGIBLE' as const,label:'Resume Builder'}]};
  assert.equal(workbenchActionGate(inbox,c,recovery).enabled,true);
  assert.equal(workbenchActionGate(inbox,c,{...recovery,
    hostActionKind:'REVIEW_REPAIR'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,c,{...work,
    hostActionId:'restart-260',hostActionRevision:3,
    hostActionKind:'START_AGENT'}).enabled,false);
});

test('malformed source SHA and absent action catalog fail closed',()=>{
  assert.equal(workbenchActionGate({...inbox,source:{...source,sourceSha:'latest'}},
    catalog,work).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,source:{...source,sourceSha:'latest'}},
    work).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,actions:undefined},work).enabled,false);
});
