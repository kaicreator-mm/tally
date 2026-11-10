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

/* E0 #284 regression (TALLY_V2_E0_CLEAN_INTEGRATION_REPAIR; finding in
 * #284 comment 6779575724): the Host-mount dynamic branch of serve.mjs
 * imported '../experiments/...' which resolves to the non-existent
 * web/experiments/... — a latent dynamic-import defect that neither
 * `node --check` nor a route-level test can catch. At the clean integration
 * base only the five documented UI-lane modules are absent (E/#264
 * boundary); every other relative specifier in serve.mjs must resolve to a
 * real candidate file, and the corrected Host-mount specifier must really
 * load the E0 Host module. */
const UI_LANE_BOUNDARY_MODULES=[
  './ui-native-host.mjs','./ui-report-host.mjs','./ui-audit-host.mjs',
  './ui-settings-host.mjs','./ui-workbench-host.mjs'];
test('serve.mjs Host-mount dynamic branch resolves the real E0 Host module at the clean base',async()=>{
  const serveSource=await (await import('node:fs/promises'))
    .readFile(new URL('./serve.mjs',import.meta.url),'utf8');
  // (1) The corrected specifier is present; the broken one is gone.
  assert.ok(serveSource.includes(String.raw`'../../experiments/v2/selected-business-build/version-task-dag-host.mjs'`),
    'serve.mjs dynamic specifier must be ../../experiments/...');
  assert.ok(!serveSource.includes(String.raw`'../experiments/v2/selected-business-build/version-task-dag-host.mjs'`),
    'the broken ../experiments/... specifier must be absent');
  // (2) The dynamic branch actually reaches the REAL E0 Host module:
  // execute the exact dynamic-import resolution serve.mjs performs
  // from web/v2 and assert the real Host constructor.
  const hostModule=await import(
    '../../experiments/v2/selected-business-build/version-task-dag-host.mjs');
  assert.equal(typeof hostModule.createVersionTaskDagHost,'function');
  // (3) Enumerate every relative specifier serve.mjs imports (static from +
  // dynamic import(...), including concatenated-literal specifiers) and prove
  // each resolves to an existing candidate file — except exactly the five
  // documented UI-lane boundary modules, which must be the ONLY absent ones.
  const literal=String.raw`'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"`;
  const importSites=[...serveSource.matchAll(new RegExp(
    `(?:\\bfrom\\s+|import\\(\\s*)((?:${literal})(?:\\s*\\+\\s*(?:${literal}))*)\\s*\\)?`,'g'))];
  const specifiers=importSites.map(([,expr])=>expr.split(/\s*\+\s*/)
    .map(l=>l.slice(1,-1)).join(''));
  assert.ok(specifiers.length>=10,'serve.mjs import sites enumerated');
  assert.ok(specifiers.includes(
    '../../experiments/v2/selected-business-build/version-task-dag-host.mjs'),
    'dynamic Host-mount specifier enumerated');
  const {existsSync}=await import('node:fs');
  const {fileURLToPath}=await import('node:url');
  const missing=[];
  for(const spec of specifiers){
    if(!spec.startsWith('.'))continue; // bare specifiers (deps) are not paths
    const p=fileURLToPath(new URL(spec,import.meta.url));
    if(existsSync(p))continue;
    missing.push(spec);
  }
  missing.sort();
  assert.deepEqual(missing,[...UI_LANE_BOUNDARY_MODULES].sort(),
    'only the five documented UI-lane modules may be absent at this base');
});
