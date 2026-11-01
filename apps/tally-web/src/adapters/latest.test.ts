import test from 'node:test';
import assert from 'node:assert/strict';
import {createLatestRequestGate,requestIdentity} from './latest.ts';
test('new version invalidates stale result',()=>{
  const g=createLatestRequestGate(),old=g.begin(),fresh=g.begin();
  assert.equal(old.isCurrent(),false);assert.equal(fresh.isCurrent(),true);
});
test('unmount or scope switch invalidates success AND errors',()=>{
  const g=createLatestRequestGate(),pending=g.begin();
  g.invalidate();assert.equal(pending.isCurrent(),false);
});
test('retired requests cannot revive',()=>{
  const g=createLatestRequestGate(),a=g.begin(),b=g.begin();
  g.invalidate();assert.equal(a.isCurrent(),false);assert.equal(b.isCurrent(),false);
});

test('scope identity prevents transient display of old project/version result',()=>{
  const first=requestIdentity('tasks',['org/a','version/v1']);
  const second=requestIdentity('tasks',['org/b','version/v2']);
  assert.notEqual(first,second);
  assert.equal(first,requestIdentity('tasks',['org/a','version/v1']));
});
