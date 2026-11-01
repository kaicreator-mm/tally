import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiSettingsHost} from './ui-settings-host.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';

const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const nativeC={catalog:async()=>({
  connected:true,source:{repository,sourceSha:sha,
    currentness:'CURRENT',readback:'VERIFIED'},
  projects:[{id:repository,title:'Tally'}],
  versionsByProject:{[repository]:[{id:versionId,
    projectId:repository,label:'v2'}]}
})};
test('real HTTP Settings exposes only verified GitHub, no provider credentials',async()=>{
  const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',
    tallyBPath:'/b',githubToken:'SECRET_HOST_ONLY_CREDENTIAL',
    currentSourceSha:sha,workRoot:'/tmp/tally-settings-test'});
  const host=createUiSettingsHost({repository,versionId,expectedSha:sha,nativeC});
  const app=createAlmServer(config,{v2ReadModelHost:host});
  const {origin}=await app.listen();
  try{
    const r=await fetch(origin+'/api/v2/settings').then(res=>res.json());
    assert.equal(r.connected,true);
    assert.equal(r.coverage,'PARTIAL');
    assert.equal(r.sectionCoverage.gitGithub,'CURRENT_OBSERVED');
    assert.equal(r.sectionCoverage.agentProviders,'NOT_CONNECTED');
    assert.deepEqual(r.snapshot.git.repositories,[repository]);
    assert.equal(JSON.stringify(r).includes('SECRET_HOST_ONLY_CREDENTIAL'),false);
    assert.equal(JSON.stringify(r).includes('github_pat_'),false);
  }finally{await app.close();}
});
