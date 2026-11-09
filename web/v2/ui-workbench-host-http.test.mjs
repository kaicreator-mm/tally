import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiWorkbenchHost} from './ui-workbench-host.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
import {readUiHostEnvironment} from './serve.mjs';

const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const nativeC={
  catalog:async()=>({connected:true,source,projects:[{id:repository,title:'Tally'}],
    versionsByProject:{[repository]:[{id:versionId,projectId:repository,label:'v2'}]}}),
  version:async()=>({connected:true,source,versionId,rosterCoverage:'UNVERIFIED',
    tasks:[{id:'210',versionId,dependencyState:'CLEAR',openBlockerCount:0,
      blockedBy:[],claimState:'NOT_VERIFIED'}],
    dag:{asOf:{sourceSha:sha},nodes:[{taskId:'210'}],edges:[]}})
};
const cfg=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/b',
  githubToken:'secrethostonly',currentSourceSha:sha,
  workRoot:'/tmp/tally-v2-workbench-http'});
test('real HTTP keeps partial queue coverage despite zero selected blockers',async()=>{
  const host=createUiWorkbenchHost({repository,versionId,expectedSha:sha,nativeC});
  const app=createAlmServer(cfg,{v2ReadModelHost:host});
  const {origin}=await app.listen();
  try{
    const r=await fetch(origin+'/api/v2/workbench').then(x=>x.json());
    assert.equal(r.connected,true);
    assert.equal(r.queueCoverage,'PARTIAL_UNVERIFIED_ROSTER');
    assert.equal(r.approvalCoverage,'NOT_CONNECTED');
    assert.deepEqual(r.actions,[]);
    assert.equal(JSON.stringify(r).includes('secrethostonly'),false);
  }finally{await app.close();}
});
test('workbench source requires explicit bounded roster and strict read-only mode',()=>{
  const env={TALLY_V2_REPOSITORY:repository,TALLY_V2_VERSION_REF:versionId,
    TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'host-private-token',
    TALLY_V2_TASK_ISSUES_JSON:'[210]',
    TALLY_V2_WORKBENCH_BLOCKERS:'read-only'};
  assert.equal(readUiHostEnvironment(env).workbenchReadOnly,true);
  assert.throws(()=>readUiHostEnvironment({...env,
    TALLY_V2_TASK_ISSUES_JSON:'[]'}),/HOST_WORKBENCH_TASK_ROSTER_REQUIRED/);
  assert.throws(()=>readUiHostEnvironment({...env,
    TALLY_V2_WORKBENCH_BLOCKERS:'write'}),/INVALID_HOST_WORKBENCH_READ_MODE/);
});
