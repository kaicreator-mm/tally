import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiV2ReadModelHost} from './ui-native-host.mjs';
const repo='org/tally',versionId='version/v2';
const source={repository:repo,sourceSha:'a'.repeat(40),
  currentness:'CURRENT',readback:'VERIFIED'};
const catalog={connected:true,source,projects:[{id:repo,title:'Tally'}],
  versionsByProject:{[repo]:[{id:versionId,projectId:repo,label:'v2'}]}};
const version={connected:true,source,versionId,tasks:[{
  id:'1',versionId,title:'Task',status:'open',severity:'UNKNOWN',
  blockedBy:[],claimState:'NOT_VERIFIED'}],
  dag:{connected:true,source,versionId,asOf:{sourceSha:source.sourceSha,
    frozenAt:null},nodes:[{taskId:'1',versionId}],edges:[]}};
const nativeC={catalog:async()=>catalog,version:async()=>version};
test('selected native C composes project/version catalog without guessed tasks',async()=>{
  const host=createUiV2ReadModelHost({repository:repo,versionId,nativeC});
  assert.deepEqual(await host.catalog(),catalog);
});
test('real C task facts and native DAG evidence carry the exact source',async()=>{
  const host=createUiV2ReadModelHost({repository:repo,versionId,nativeC});
  const scope={projectId:repo,versionId};
  assert.deepEqual((await host.tasks(scope)).tasks,version.tasks);
  assert.equal((await host.tasks(scope)).rosterCoverage,'UNVERIFIED');
  assert.deepEqual(await host.dag(scope),version.dag);
});
test('foreign project/version request fails closed before any native read',async()=>{
  let calls=0;
  const host=createUiV2ReadModelHost({repository:repo,versionId,
    nativeC:{catalog:async()=>catalog,version:async()=>{calls++;return version;}}});
  assert.equal((await host.tasks({projectId:'evil/repo',versionId})).connected,false);
  assert.equal((await host.dag({projectId:repo,versionId:'version/v3'})).connected,false);
  assert.equal(calls,0);
});
test('no verified Task roster cannot produce an empty successful list',async()=>{
  const host=createUiV2ReadModelHost({repository:repo,versionId,
    nativeC:{catalog:async()=>catalog,
      version:async()=>({connected:false,reason:'ROSTER_UNAVAILABLE'})}});
  assert.deepEqual(await host.tasks({projectId:repo,versionId}),
    {connected:false,reason:'ROSTER_UNAVAILABLE'});
});
test('untrusted GitHub C observation never becomes Host-connected',async()=>{
  const host=createUiV2ReadModelHost({repository:repo,versionId,
    nativeC:{catalog:async()=>({...catalog,source:{...source,repository:'evil/repo'}}),
      version:async()=>({...version,source:{...source,repository:'evil/repo'}})}});
  assert.equal((await host.catalog()).connected,false);
  assert.equal((await host.tasks({projectId:repo,versionId})).connected,false);
});
