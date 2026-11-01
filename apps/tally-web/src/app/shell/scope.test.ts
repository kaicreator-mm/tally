import test from 'node:test';
import assert from 'node:assert/strict';
import {resolveScope,chooseProject,chooseVersion,taskRoute} from './scope.ts';
const catalog={
  projects:[{id:'owner/a',title:'a'},{id:'owner/b',title:'b'}],
  versionsByProject:{
    'owner/a':[{id:'version/v2',label:'v2',projectId:'owner/a'}],
    'owner/b':[{id:'version/v3',label:'v3',projectId:'owner/b'}],
  },
};
test('deep link restores valid project/version',()=>{
  assert.deepEqual(resolveScope(catalog,{projectId:'owner/a',versionId:'version/v2'}),
    {projectId:'owner/a',versionId:'version/v2'});
});
test('foreign project version is rejected',()=>{
  assert.deepEqual(resolveScope(catalog,{projectId:'owner/b',versionId:'version/v2'}),
    {projectId:'owner/b',versionId:null});
});
test('foreign project cannot become active',()=>{
  assert.deepEqual(resolveScope(catalog,{projectId:'unknown',versionId:'version/v2'}),
    {projectId:null,versionId:null});
});
test('no catalog means no trusted project identity',()=>{
  assert.deepEqual(resolveScope({projects:null,versionsByProject:null},
    {projectId:'owner/a',versionId:'version/v2'}),{projectId:null,versionId:null});
});
test('project picker clears old version and validates new version',()=>{
  const start=chooseProject(catalog,'owner/b');
  assert.deepEqual(start,{projectId:'owner/b',versionId:null});
  assert.deepEqual(chooseVersion(catalog,start,'version/v2'),
    {projectId:'owner/b',versionId:null});
  assert.deepEqual(chooseVersion(catalog,start,'version/v3'),
    {projectId:'owner/b',versionId:'version/v3'});
});

test('Task route carries exact validated project/version/task and escapes IDs',()=>{
  assert.equal(taskRoute('org/tally','version/v2','227'),
    '#/development/tasks?projectId=org%2Ftally&versionId=version%2Fv2&taskId=227');
  assert.equal(taskRoute('org/tally','version/v2',null),
    '#/development/tasks?projectId=org%2Ftally&versionId=version%2Fv2');
  assert.equal(taskRoute('','version/v2','227'),null);
});
