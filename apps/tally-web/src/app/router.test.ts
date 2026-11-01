import test from 'node:test';
import assert from 'node:assert/strict';
import {parseHash,buildHash,createRouter} from './router.ts';

test('parses canonical group/view routes with decoded query',()=>{
  const route=parseHash('#/development/tasks?projectId=tally&versionId=v2%20r1');
  assert.equal(route.group,'development');
  assert.equal(route.view,'tasks');
  assert.equal(route.query.projectId,'tally');
  assert.equal(route.query.versionId,'v2 r1');
});
test('defaults to workbench/index and rejects malformed segments',()=>{
  assert.deepEqual(parseHash(''),{group:'workbench',view:'index',query:{}});
  const evil=parseHash('#/../etc?x=1');
  assert.equal(evil.group,'workbench');
  const upper=parseHash('#/Workbench/INDEX');
  assert.equal(upper.group,'workbench');
  assert.equal(upper.view,'index');
  assert.deepEqual(upper.query,{});
});
test('drops malformed query keys and keeps valid ones',()=>{
  const route=parseHash('#/workbench/inbox?ok=1&=2&3bad=x&k=%E4%B8%AD');
  assert.equal(route.query.ok,'1');
  assert.equal(route.query.k,'中');
  assert.equal(Object.keys(route.query).length,2);
});
test('buildHash round-trips and encodes reserved characters',()=>{
  const hash=buildHash({group:'governance',view:'metadata',
    query:{projectId:'a b/c',tab:'reports'}});
  assert.equal(hash,'#/governance/metadata?projectId=a%20b%2Fc&tab=reports');
  const parsed=parseHash(hash);
  assert.equal(parsed.query.projectId,'a b/c');
});
test('buildHash refuses invalid segments (URL-safety)',()=>{
  assert.throws(()=>buildHash({group:'x y',view:'v'}),/ROUTE_SEGMENT_INVALID/);
  assert.throws(()=>buildHash({group:'g',view:'../../etc'}),/ROUTE_SEGMENT_INVALID/);
});
test('router navigates, notifies subscribers and unsubscribes cleanly',()=>{
  const events:string[]=[];
  let hash='#/workbench/inbox';
  const listeners=new Set<()=>void>();
  const source={getHash:()=>hash,setHash:(next:string)=>{hash=next;
      for(const fn of listeners)fn();},
    addEventListener:(_:'hashchange',fn:()=>void)=>listeners.add(fn),
    removeEventListener:(_:'hashchange',fn:()=>void)=>listeners.delete(fn)};
  const router=createRouter(source);
  const off=router.subscribe(route=>events.push(`${route.group}/${route.view}`));
  router.navigate('#/development/tasks?projectId=p1');
  assert.equal(router.get().query.projectId,'p1');
  assert.deepEqual(events,['development/tasks']);
  // Same-hash navigate still emits (re-select current view).
  router.navigate('#/development/tasks?projectId=p1');
  assert.equal(events.length,2);
  off();
  router.navigate('#/settings/ci');
  assert.equal(events.length,2);
  assert.throws(()=>router.navigate('development/tasks'),/HASH_REQUIRED/);
});
