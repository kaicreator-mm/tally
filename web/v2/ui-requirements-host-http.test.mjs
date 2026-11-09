import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiV2ReadModelHost} from './ui-native-host.mjs';
import {readUiHostEnvironment} from './serve.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';

const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const qualified={connected:true,source,versionId,
  structureCoverage:'HOST_DECLARED_UNVERIFIED',freezeCoverage:'UNKNOWN',
  nodes:[{id:'77',kind:'requirement',title:'Canonical Product Issue',
    parent:null,productRevision:null,acceptanceRefs:[],
    l2Decision:null,freezeState:'UNKNOWN',updatedAt:'2026-11-02T00:00:00Z'}]};
const nativeC={catalog:async()=>({
    connected:true,source,projects:[{id:repository,title:'Tally'}],
    versionsByProject:{[repository]:[{id:versionId,projectId:repository,label:'v2'}]}}),
  version:async()=>({connected:false,reason:'NO_TASK_ROSTER'}),
  requirements:async()=>qualified};
const config=defaultAlmConfig({dhPath:'/fake/dh',tallyPath:'/fake/tally',
  tallyBPath:'/fake/b',githubToken:'host-only-secret',
  currentSourceSha:sha,workRoot:'/tmp/tally-v2-requirements'});
test('real HTTP reads GitHub C requirement titles but refuses to assert Product freeze',async()=>{
  const host=createUiV2ReadModelHost({repository,versionId,nativeC});
  const app=createAlmServer(config,{v2ReadModelHost:host});
  const {origin}=await app.listen();
  try{
    const route='/api/v2/requirements?projectId=org%2Ftally&versionId=version%2Fv2';
    const r=await fetch(origin+route).then(x=>x.json());
    assert.equal(r.connected,true);
    assert.equal(r.nodes[0].freezeState,'UNKNOWN');
    assert.equal(r.structureCoverage,'HOST_DECLARED_UNVERIFIED');
    assert.equal(r.freezeCoverage,'UNKNOWN');
    assert.equal(JSON.stringify(r).includes('host-only-secret'),false);
    const foreign=await fetch(origin+
      '/api/v2/requirements?projectId=other%2Frepo&versionId=version%2Fv2')
      .then(x=>x.json());
    assert.equal(foreign.connected,false);
  }finally{await app.close();}
});
test('selected Host config rejects invalid and repeated Requirement Issue rosters',()=>{
  const env={TALLY_V2_REPOSITORY:repository,TALLY_V2_VERSION_REF:versionId,
    TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'token-private'};
  assert.equal(readUiHostEnvironment(env).requirements,null);
  const valid='[{"number":77,"kind":"requirement","parent":null}]';
  assert.equal(readUiHostEnvironment({
    ...env,TALLY_V2_REQUIREMENT_ISSUES_JSON:valid
  }).requirements[0].number,77);
  assert.throws(()=>readUiHostEnvironment({...env,
    TALLY_V2_REQUIREMENT_ISSUES_JSON:
      '[{"number":77,"kind":"story","parent":null},{"number":77,"kind":"story","parent":null}]'}),
    /INVALID_HOST_REQUIREMENT_ROSTER/);
});
