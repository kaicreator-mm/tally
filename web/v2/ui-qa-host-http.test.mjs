import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiV2ReadModelHost} from './ui-native-host.mjs';
import {readUiHostEnvironment} from './serve.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
const sha='a'.repeat(40),repo='org/tally',versionId='version/v2';
const source={repository:repo,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const observed={connected:true,source,versionId,ciCoverage:'REPORTED_ONLY',
  independentCoverage:'UNKNOWN',records:[{
    layer:'fullRegression',provenance:'REPORTED',outcome:'PASS',
    witness:'github-native-c-status-reported',
    evidenceRef:'https://ci.kaicreator.com/repos/12/pipeline/369/2',
    subjectSha:sha,observedAt:'2026-11-02T00:00:00Z'}]};
const nativeC={catalog:async()=>({connected:true,source,
    projects:[{id:repo,title:'Tally'}],
    versionsByProject:{[repo]:[{id:versionId,projectId:repo,label:'v2'}]}}),
  version:async()=>({connected:false,reason:'ROSTER_UNAVAILABLE'}),
  qa:async()=>observed};
test('real HTTP QA returns reported-only CI; all independent layers remain absent',async()=>{
  const cfg=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',
    tallyBPath:'/b',githubToken:'neverexposehostonly',
    currentSourceSha:sha,workRoot:'/tmp/tally-v2-qa-bridge'});
  const app=createAlmServer(cfg,{v2ReadModelHost:
    createUiV2ReadModelHost({repository:repo,versionId,nativeC})});
  const {origin}=await app.listen();
  try{
    const data=await fetch(origin+
      '/api/v2/qa?projectId=org%2Ftally&versionId=version%2Fv2')
      .then(x=>x.json());
    assert.equal(data.connected,true);
    assert.equal(data.records.length,1);
    assert.equal(data.records[0].provenance,'REPORTED');
    assert.equal(data.independentCoverage,'UNKNOWN');
    assert.equal(JSON.stringify(data).includes('neverexposehostonly'),false);
    const foreign=await fetch(origin+
      '/api/v2/qa?projectId=other%2Frepo&versionId=version%2Fv2')
      .then(x=>x.json());
    assert.equal(foreign.connected,false);
  }finally{await app.close();}
});
test('QA configured context must be exact Woodpecker status identifier',()=>{
  const env={TALLY_V2_REPOSITORY:repo,TALLY_V2_VERSION_REF:versionId,
    TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'private-host-token',
    TALLY_V2_QA_STATUS_CONTEXT:'ci/woodpecker/push/v2-alm-workspace'};
  assert.equal(readUiHostEnvironment(env).qaStatusContext,
    'ci/woodpecker/push/v2-alm-workspace');
  assert.throws(()=>readUiHostEnvironment({...env,
    TALLY_V2_QA_STATUS_CONTEXT:'hiddenValidation/APPROVED'}),
    /INVALID_HOST_QA_STATUS_CONTEXT/);
});
