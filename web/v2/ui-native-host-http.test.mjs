import test from 'node:test';
import assert from 'node:assert/strict';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {createUiV2ReadModelHost} from './ui-native-host.mjs';
import {readUiHostEnvironment} from './serve.mjs';

const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const catalog={connected:true,source,
  projects:[{id:repository,title:'tally',repository}],
  versionsByProject:{[repository]:[{id:versionId,projectId:repository,
    label:versionId,ref:versionId}]}};
const version={connected:true,source,versionId,
  tasks:[{id:'101',versionId,title:'Native task',status:'unknown',
    severity:'UNKNOWN',blockedBy:[],agentId:null,claimState:'NOT_VERIFIED',
    attemptId:null,attemptState:null,prNumber:null,reviewState:'UNKNOWN',
    ciState:'UNKNOWN',writeSet:[],requirementRef:null,updatedAt:'2026-11-01T00:00:00Z'}],
  dag:{connected:true,source,versionId,asOf:{sourceSha:sha,frozenAt:null},
    nodes:[{taskId:'101',versionId,title:'Native task',status:'unknown',
      severity:'UNKNOWN',requirementGroup:null,phase:null,lane:null,
      agentId:null,executable:false}],edges:[]}};
const cfg=defaultAlmConfig({dhPath:'/fake/dh',tallyPath:'/fake/tally',
  tallyBPath:'/fake/b',githubToken:'private-not-in-responses',
  currentSourceSha:sha,workRoot:'/tmp/tally-ui-native-host'});
const readModelHost=createUiV2ReadModelHost({repository,versionId,
  nativeC:{catalog:async()=>catalog,version:async()=>version}});
test('real HTTP responds with selected C catalog/Task/DAG, never invented Claim',async()=>{
  const app=createAlmServer(cfg,{v2ReadModelHost:readModelHost,
    run:async()=>({status:'FAIL',exitCode:1,reason:'NO_AGENT_DISPATCH'})});
  const {origin}=await app.listen();
  try{
    const cat=await fetch(origin+'/api/v2/catalog').then(r=>r.json());
    assert.equal(cat.connected,true);
    assert.equal(cat.projects[0].id,repository);
    const q='?projectId=org%2Ftally&versionId=version%2Fv2';
    const tasks=await fetch(origin+'/api/v2/tasks'+q).then(r=>r.json());
    const dag=await fetch(origin+'/api/v2/dag'+q).then(r=>r.json());
    assert.equal(tasks.connected,true);
    assert.equal(dag.connected,true);
    assert.equal(tasks.tasks[0].claimState,'NOT_VERIFIED');
    assert.equal(dag.nodes[0].executable,false);
    assert.equal(JSON.stringify([cat,tasks,dag]).includes('private-not-in-responses'),false);
  }finally{await app.close();}
});
test('foreign project and unconfigured metadata remain NOT_CONNECTED',async()=>{
  const app=createAlmServer(cfg,{v2ReadModelHost:readModelHost});
  const {origin}=await app.listen();
  try{
    const denied=await fetch(origin+
      '/api/v2/tasks?projectId=evil%2Frepo&versionId=version%2Fv2')
      .then(r=>r.json());
    assert.equal(denied.connected,false);
    const metadata=await fetch(origin+
      '/api/v2/metadata?projectId=org%2Ftally&versionId=version%2Fv2')
      .then(r=>r.json());
    assert.equal(metadata.connected,false);
  }finally{await app.close();}
});
test('Host environment demands explicit repo/ref/sha/token; missing roster is not fake zero',()=>{
  const env={TALLY_V2_REPOSITORY:repository,TALLY_V2_VERSION_REF:versionId,
    TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'host-only-token'};
  assert.deepEqual(readUiHostEnvironment(env).roster,[]);
  assert.throws(()=>readUiHostEnvironment({...env,TALLY_V2_VERSION_SHA:'bad'}),
    /SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED/);
  assert.throws(()=>readUiHostEnvironment({...env,
    TALLY_V2_TASK_ISSUES_JSON:'[1,1]'}),/INVALID_HOST_TASK_ROSTER/);
  assert.deepEqual(readUiHostEnvironment({...env,
    TALLY_V2_TASK_ISSUES_JSON:'[101,102]'}).roster,[101,102]);
});
