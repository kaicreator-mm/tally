import test from 'node:test';
import assert from 'node:assert/strict';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';

const sha='d'.repeat(40);
const packRef='https://github.com/kaicreator-mm/tally/blob/'+
  'b3ad8ae7833028999a9c27cac7042037bca6d961'+
  '/apps/tally-web/src/features/dag/version-v2-task-pack.candidate.json';
const packSha='d8b1c8dcfe22fdc27a2ffe151341ac3d3728329d';
const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',
  tallyBPath:'/tally',githubToken:'nonpublicfixture',
  currentSourceSha:sha,workRoot:'/tmp/tally-v2-version-task-dag-guard'});
const dependencyArgs={taskPackRef:packRef,taskPackBlobSha:packSha,
  blockedIssueNumber:312,blockedByIssueNumber:284};
const freezeArgs={taskPackRef:packRef,taskPackBlobSha:packSha};
const post=(origin,key,kind,args,source=sha)=>fetch(
  origin+'/api/v2/version-task-dag/actions',{method:'POST',
    headers:{origin,'content-type':'application/json','x-tally-action-key':key},
    body:JSON.stringify({kind,sourceSha:source,args})});
const dependencyObservation={status:'NATIVE_EDGE_VERIFIED',
  reason:'OFFICIAL_GITHUB_BLOCKED_BY_READBACK_VERIFIED',
  repository:'kaicreator-mm/tally',versionId:'version/v2',
  blockedIssueNumber:312,blockedByIssueNumber:284,dependencyId:284000,
  native:true,githubEffect:'PERFORMED_VERIFIED',writePerformed:true,
  authorized:false,admitted:false,taskPass:false,releasePass:false};
const freezeObservation={status:'FROZEN',
  reason:'FROZEN_VERSION_TASK_DAG_CAS_READBACK_VERIFIED',
  repository:'kaicreator-mm/tally',versionId:'version/v2',
  taskPackRef:packRef,taskPackBlobSha:packSha,blobSha:packSha,
  frozenAt:'2026-11-09T13:00:00.000Z',taskCount:3,edgeCount:1,
  writeSetRecordCount:3,githubEffect:'NOT_RUN',
  dependencyWritesAfterFreeze:'REFUSED',authorized:false,admitted:false,
  taskPass:false,releasePass:false};

test('route stays DENIED with zero effects without the sealed Host seam',async()=>{
  const app=createAlmServer(config);
  const {origin,actionKey}=await app.listen();
  try{
    const csrf=await post(origin,'wrong-key','add-native-dependency',dependencyArgs);
    assert.equal(csrf.status,403);
    const crossOrigin=await fetch(origin+'/api/v2/version-task-dag/actions',
      {method:'POST',headers:{origin:origin.replace('127.0.0.1','localhost'),
        'content-type':'application/json','x-tally-action-key':actionKey},
       body:JSON.stringify({kind:'add-native-dependency',sourceSha:sha,
         args:dependencyArgs})});
    assert.equal(crossOrigin.status,403);
    const stale=await post(origin,actionKey,'add-native-dependency',dependencyArgs,
      'a'.repeat(40));
    assert.equal(stale.status,409);
    const unknownAction=await post(origin,actionKey,'admit-root-claim',{});
    assert.equal(unknownAction.status,400);
    assert.equal((await unknownAction.json()).reason,'UNKNOWN_ACTION');
    const missingHost=await post(origin,actionKey,'add-native-dependency',
      dependencyArgs);
    assert.equal(missingHost.status,200);
    const denied=await missingHost.json();
    assert.equal(denied.status,'DENIED');
    assert.equal(denied.reason,'HOST_SEAM_NOT_CONNECTED');
    assert.equal(denied.githubEffect,'NOT_RUN');
  }finally{await app.close();}
});

test('body allowlist refuses malformed, extra and oversized bodies with zero effects',async()=>{
  const dispatches=[];
  const seam=async request=>{dispatches.push(request);return dependencyObservation;};
  const app=createAlmServer(config,{versionTaskDagSealedDispatch:seam});
  const {origin,actionKey}=await app.listen();
  try{
    for(const [kind,args] of [
      ['add-native-dependency',{...dependencyArgs,extraField:'x'}],
      ['add-native-dependency',{taskPackRef:packRef}],
      ['add-native-dependency',{...dependencyArgs,taskPackRef:'https://github.com/kaicreator-mm/tally/blob/main/x.json'}],
      ['add-native-dependency',{...dependencyArgs,taskPackBlobSha:'deadbeef'}],
      ['add-native-dependency',{...dependencyArgs,blockedIssueNumber:0}],
      ['add-native-dependency',{...dependencyArgs,blockedIssueNumber:'312'}],
      ['freeze',{...freezeArgs,blockedIssueNumber:312}],
      ['freeze',{taskPackBlobSha:packSha}],
      ['freeze',{...freezeArgs,taskPackRef:'not-a-url'}],
      ['freeze',null],
      ['freeze',[]],
    ]){
      const response=await post(origin,actionKey,kind,args);
      assert.equal(response.status,400,JSON.stringify(args));
      assert.equal((await response.json()).reason,'ARGS_INVALID');
    }
    const malformed=await fetch(origin+'/api/v2/version-task-dag/actions',
      {method:'POST',headers:{origin,'content-type':'application/json',
        'x-tally-action-key':actionKey},body:'{not json'});
    assert.equal(malformed.status,400);
    const oversized=await fetch(origin+'/api/v2/version-task-dag/actions',
      {method:'POST',headers:{origin,'content-type':'application/json',
        'x-tally-action-key':actionKey},
        body:JSON.stringify({kind:'freeze',sourceSha:sha,
          args:{taskPackRef:packRef,taskPackBlobSha:packSha,pad:'x'.repeat(9000)}})});
    assert.equal(oversized.status,400);
    // A self-edge is SYNTACTICALLY allowlisted here; refusing it is the sealed
    // selected-C EDGE_INPUT_INVALID gate, not an HTTP-layer decision.
    const selfEdge=await post(origin,actionKey,'add-native-dependency',
      {...dependencyArgs,blockedIssueNumber:284});
    assert.equal(selfEdge.status,200);
    assert.equal(dispatches.length,1,'only the syntactically valid self-edge dispatched');
    assert.equal(dispatches[0].input.blockedIssueNumber,284);
  }finally{await app.close();}
});

test('sealed seam observations are the only positive path and stay fail-closed',async()=>{
  const dispatches=[];
  const seam=async request=>{
    dispatches.push(request);
    if(request.operation==='addNativeTaskDependency')return dependencyObservation;
    return freezeObservation;
  };
  const app=createAlmServer(config,{versionTaskDagSealedDispatch:seam});
  const {origin,actionKey}=await app.listen();
  try{
    const dependency=await post(origin,actionKey,'add-native-dependency',
      dependencyArgs);
    assert.equal(dependency.status,200);
    const dependencyResult=await dependency.json();
    assert.equal(dependencyResult.status,'NATIVE_EDGE_VERIFIED');
    assert.equal(dependencyResult.authorized,false);
    assert.equal(dependencyResult.taskPass,false);
    const frozen=await post(origin,actionKey,'freeze',freezeArgs);
    assert.equal((await frozen.json()).status,'FROZEN');
    assert.deepEqual(dispatches,[
      {operation:'addNativeTaskDependency',sourceSha:sha,
        input:{...dependencyArgs}},
      {operation:'freezeVersionTaskDag',sourceSha:sha,input:{...freezeArgs}}]);
  }finally{await app.close();}
});

test('seam throw and forged seam shapes degrade to DENIED with zero positive effects',async()=>{
  for(const forged of [null,{},()=>({}),
    {...dependencyObservation,status:'FORGED'},
    {...dependencyObservation,authorized:true},
    {...dependencyObservation,taskPass:true},
    'NATIVE_EDGE_VERIFIED']){
    const app=createAlmServer(config,{versionTaskDagSealedDispatch:
      typeof forged==='function'?forged:async()=>forged});
    const {origin,actionKey}=await app.listen();
    try{
      const response=await post(origin,actionKey,'add-native-dependency',
        dependencyArgs);
      assert.equal(response.status,200);
      const result=await response.json();
      assert.equal(result.status,'DENIED');
      assert.ok(['SEAM_ERROR','SEAM_SHAPE_INVALID'].includes(result.reason));
      assert.equal(result.githubEffect,'NOT_RUN');
    }finally{await app.close();}
  }
});

test('the root Claim path is not HTTP-reachable from this server',async()=>{
  const dispatches=[];
  const app=createAlmServer(config,{versionTaskDagSealedDispatch:async request=>{
    dispatches.push(request);return dependencyObservation;}});
  const {origin,actionKey}=await app.listen();
  try{
    // No route shape exposes a claim/reserve/publish operation.
    for(const [path,body] of [
      ['/api/v2/version-task-dag/root-claim',{kind:'add-native-dependency',
        sourceSha:sha,args:dependencyArgs}],
      ['/api/v2/version-task-dag/claim',{kind:'add-native-dependency',
        sourceSha:sha,args:dependencyArgs}],
      ['/api/v2/version-task-dag/reserve',{kind:'add-native-dependency',
        sourceSha:sha,args:dependencyArgs}],
      ['/api/v2/version-task-dag/publish',{kind:'add-native-dependency',
        sourceSha:sha,args:dependencyArgs}],
    ]){
      const response=await fetch(origin+path,{method:'POST',
        headers:{origin,'content-type':'application/json',
          'x-tally-action-key':actionKey},body:JSON.stringify(body)});
      assert.equal(response.status,404,path);
    }
    const readOnly=await fetch(origin+'/api/v2/version-task-dag');
    assert.equal(readOnly.status,404);
    // The only two operations ever dispatched are the exact E0 pair.
    const ok=await post(origin,actionKey,'add-native-dependency',dependencyArgs);
    assert.equal((await ok.json()).status,'NATIVE_EDGE_VERIFIED');
    assert.deepEqual(dispatches.map(d=>d.operation),['addNativeTaskDependency']);
  }finally{await app.close();}
});
