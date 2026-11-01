import test from 'node:test';
import assert from 'node:assert/strict';
import {versionDeployments,environmentOverview,deploymentBadge} from './model.ts';
import type {DeploymentRecord} from './model.ts';
const r:DeploymentRecord={id:'d1',versionId:'version/v2',
  environment:'staging',releaseSha:'a'.repeat(40),state:'SUCCEEDED',
  health:'UNKNOWN',deployedAt:'2026-11-01T00:00:00Z',
  receiptRef:null,rollbackReceiptRef:null};
test('deployments cannot leak across versions',()=>{
  assert.deepEqual(versionDeployments([r,{...r,id:'d2',versionId:'version/v3'}],
    'version/v2').map(x=>x.id),['d1']);
});
test('latest environment observation retains independent receipt',()=>{
  const b={...r,id:'d2',health:'HEALTHY' as const,
    deployedAt:'2026-11-01T01:00:00Z',receiptRef:'https://github.com/org/tally/issues/1'};
  assert.deepEqual(environmentOverview([r,b]).map(x=>x.id),['d2']);
});
test('deployment success without health evidence cannot imply release PASS',()=>{
  assert.equal(deploymentBadge(r),'warn');
  assert.equal(deploymentBadge({...r,health:'HEALTHY'}),'ok');
  assert.equal(deploymentBadge({...r,state:'FAILED'}),'danger');
});
