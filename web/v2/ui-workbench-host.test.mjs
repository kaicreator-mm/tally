import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiWorkbenchHost} from './ui-workbench-host.mjs';
const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const catalog={connected:true,source,
  versionsByProject:{[repository]:[{id:versionId,projectId:repository,label:'v2'}]}};
const t=(id,open=0)=>({id:String(id),versionId,
  claimState:'NOT_VERIFIED',blockedBy:open?['7']:[],
  dependencyState:open?'BLOCKED':'CLEAR',openBlockerCount:open,
  updatedAt:'2026-11-02T00:00:00Z'});
const version=(tasks=[t(7),t(8,1)])=>({connected:true,source,versionId,
  rosterCoverage:'UNVERIFIED',tasks,dag:{nodes:tasks.map(x=>({taskId:x.id})),
    edges:[],asOf:{sourceSha:sha}}});
const opts={repository,versionId,expectedSha:sha,
  nativeC:{catalog:async()=>catalog,version:async()=>version()}};
test('only native open-blocker count yields a partial advisory blocker',async()=>{
  const r=await createUiWorkbenchHost(opts).workbench({projectId:repository});
  assert.equal(r.connected,true);
  assert.equal(r.queueCoverage,'PARTIAL_UNVERIFIED_ROSTER');
  assert.equal(r.approvalCoverage,'NOT_CONNECTED');
  assert.deepEqual(r.actions.map(x=>x.taskId),['8']);
  assert.equal(r.actions[0].kind,'BLOCKER');
  assert.equal(r.actions[0].state,'BLOCKED');
  assert.equal(r.actions[0].ownerRole,'UNASSIGNED');
  assert.equal(r.actions[0].sourceRef,'https://github.com/org/tally/issues/8');
});
test('zero blockers is scoped observation, never a complete inbox',async()=>{
  const host=createUiWorkbenchHost({...opts,nativeC:{...opts.nativeC,
    version:async()=>version([t(7)])}});
  const result=await host.workbench({projectId:repository});
  assert.deepEqual(result.actions,[]);
  assert.equal(result.queueCoverage,'PARTIAL_UNVERIFIED_ROSTER');
});
test('cross repository or stale source never invokes or qualifies Task read',async()=>{
  let reads=0;
  const host=createUiWorkbenchHost({...opts,nativeC:{
    ...opts.nativeC,version:async()=>{reads++;return version();}}});
  assert.equal((await host.workbench({projectId:'evil/repo'})).connected,false);
  assert.equal(reads,0);
  const stale=createUiWorkbenchHost({...opts,nativeC:{
    catalog:async()=>({...catalog,source:{...source,sourceSha:'b'.repeat(40)}}),
    version:async()=>version()}});
  assert.equal((await stale.workbench({projectId:repository})).connected,false);
});
test('forged Task blocker counts and accepted Claims are rejected',async()=>{
  for(const bad of [
    {...t(8,1),openBlockerCount:0},
    {...t(8,1),claimState:'GRANTED'},
    {...t(8,1),blockedBy:[]}
  ]){
    const host=createUiWorkbenchHost({...opts,nativeC:{
      ...opts.nativeC,version:async()=>version([t(7),bad])}});
    assert.equal((await host.workbench({projectId:repository})).connected,false);
  }
});
test('moving version or incomplete DAG never becomes a connected queue',async()=>{
  let calls=0;
  const moved=createUiWorkbenchHost({...opts,nativeC:{
    catalog:async()=>({...catalog,source:{...source,
      sourceSha:++calls===1?sha:'b'.repeat(40)}}),
    version:async()=>version()}});
  assert.equal((await moved.workbench({projectId:repository})).reason,
    'PINNED_WORKBENCH_SOURCE_CHANGED_DURING_READ');
  const wrong=createUiWorkbenchHost({...opts,nativeC:{
    ...opts.nativeC,version:async()=>({...version(),dag:{
      nodes:[{taskId:'7'}],edges:[],asOf:{sourceSha:sha}}})}});
  assert.equal((await wrong.workbench({projectId:repository})).connected,false);
});
