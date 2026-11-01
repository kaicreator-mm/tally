import test from 'node:test';
import assert from 'node:assert/strict';
import {actionDisplay,releaseBlockers} from './model.ts';
import type {ReleaseReadiness} from './model.ts';
const sha='a'.repeat(40);
const ready=():ReleaseReadiness=>({
  versionId:'version/v2',currentSourceSha:sha,
  rqState:'QUALIFIED',ownerGrantState:'GRANTED',
  gates:[{name:'FULL',status:'PASS',owner:'validator'},
    {name:'HIDDEN',status:'PASS',owner:'independent'}],
  permits:[{operation:'merge',state:'GRANTED',sourceSha:sha,
    grantedAt:'2026-11-01T10:00:00.000Z',currentness:'CURRENT'}],
});
test('qualified exact-source current Host permit can render eligible (dispatcher separately required)',()=>{
  assert.equal(actionDisplay(ready(),'merge').enabled,true);
});
test('RQ UNKNOWN cannot be overridden by granted-looking Host permit',()=>{
  assert.equal(actionDisplay({...ready(),rqState:'UNKNOWN'},'merge').enabled,false);
});
test('Owner grant absent blocks action',()=>{
  assert.equal(actionDisplay({...ready(),ownerGrantState:'NOT_GRANTED'},'merge').enabled,false);
});
test('missing gates cannot be treated as all passed',()=>{
  const d=actionDisplay({...ready(),gates:[]},'merge');
  assert.equal(d.enabled,false);
  assert.equal(d.reason,'RELEASE_GATES_UNVERIFIED');
});
test('failing gate blocks release despite a GRANTED permit',()=>{
  const r=ready();assert.equal(actionDisplay({...r,gates:[
    {name:'FULL',status:'FAIL',owner:'CI'}]},'merge').enabled,false);
});
test('source mismatch and absent exact active SHA fail closed',()=>{
  const r=ready();
  assert.equal(actionDisplay({...r,currentSourceSha:null},'merge').enabled,false);
  assert.equal(actionDisplay({...r,currentSourceSha:'b'.repeat(40)},'merge').enabled,false);
});
test('stale Host permit and missing grantedAt fail closed',()=>{
  const r=ready(),p=r.permits![0];
  assert.equal(actionDisplay({...r,permits:[{...p,currentness:'STALE'}]},'merge').enabled,false);
  assert.equal(actionDisplay({...r,permits:[{...p,grantedAt:null}]},'merge').enabled,false);
});
test('permit for another operation never authorizes tag/publish',()=>{
  assert.equal(actionDisplay(ready(),'tag').enabled,false);
  assert.equal(actionDisplay(ready(),'publish').enabled,false);
});
test('release blockers distinguish source-independent RQ and permission gaps',()=>{
  const r={...ready(),rqState:'UNKNOWN',ownerGrantState:'UNKNOWN'} as ReleaseReadiness;
  assert.deepEqual(releaseBlockers(r).map(x=>x.code),['RQ_UNKNOWN','GRANT_UNKNOWN']);
});
