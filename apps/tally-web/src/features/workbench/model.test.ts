import test from 'node:test';
import assert from 'node:assert/strict';
import {activeWorkActions,inboxCounts,actionTaskLink} from './model.ts';
import type {WorkbenchSnapshot,WorkAction} from './model.ts';
const base:WorkAction={id:'1',taskId:'227',projectId:'org/tally',
  versionId:'version/v2',title:'Review',kind:'REVIEW',state:'PENDING',
  ownerRole:'reviewer',sourceRef:'https://github.com/org/tally/issues/227',
  reason:null,updatedAt:null};
const ready:WorkbenchSnapshot={connected:true,
  source:{repository:'org/tally',sourceSha:'a'.repeat(40),currentness:'CURRENT',
    readback:'VERIFIED'},
  actions:[base,{...base,id:'2',kind:'APPROVAL',state:'BLOCKED'},
    {...base,id:'3',kind:'RECOVERY',state:'PENDING'},
    {...base,id:'4',state:'RESOLVED'}]};
test('disconnect never implies an empty inbox',()=>{
  assert.equal(activeWorkActions(null,'inbox'),null);
  assert.equal(inboxCounts({...ready,connected:false}),null);
});
test('one source drives workbench inbox, approvals and blockers',()=>{
  assert.deepEqual(inboxCounts(ready),{pending:3,approvals:1,blockers:2});
  assert.deepEqual(activeWorkActions(ready,'approvals')!.map(a=>a.id),['2']);
  assert.deepEqual(activeWorkActions(ready,'blockers')!.map(a=>a.id),['2','3']);
});
test('partial roster never turns a zero scoped blocker count into zero human inbox',()=>{
  assert.equal(inboxCounts({...ready,queueCoverage:'PARTIAL_UNVERIFIED_ROSTER',
    actions:[]}),null);
});
test('resolved decisions never reappear as pending',()=>{
  assert.equal(activeWorkActions(ready,'inbox')!.some(a=>a.id==='4'),false);
});
test('task deep link includes verified project and version identity',()=>{
  assert.equal(actionTaskLink(base),
    '#/development/tasks?projectId=org%2Ftally&versionId=version%2Fv2&taskId=227');
  assert.equal(actionTaskLink({...base,versionId:null}),null);
});
