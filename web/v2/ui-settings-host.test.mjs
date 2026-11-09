import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiSettingsHost} from './ui-settings-host.mjs';
const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const catalog={connected:true,source:{repository,sourceSha:sha,
  currentness:'CURRENT',readback:'VERIFIED'},
  projects:[{id:repository,title:'Tally'}],
  versionsByProject:{[repository]:[{id:versionId,projectId:repository,label:'v2'}]}};
const opts={repository,versionId,expectedSha:sha,
  nativeC:{catalog:async()=>catalog}};
test('only selected GitHub inventory is observed; unknown settings never invented',async()=>{
  const result=await createUiSettingsHost(opts).settings();
  assert.equal(result.connected,true);
  assert.equal(result.coverage,'PARTIAL');
  assert.deepEqual(result.snapshot.git.repositories,[repository]);
  assert.equal(result.sectionCoverage.gitGithub,'CURRENT_OBSERVED');
  for(const category of ['agentProviders','tallyLlm','ci',
    'deployment','security','storage']){
    assert.equal(result.sectionCoverage[category],'NOT_CONNECTED');
  }
  assert.equal(JSON.stringify(result).includes('secret'),false);
  assert.equal(result.source.sourceSha,sha);
});
test('missing source, stale SHA and second-read branch movement fail closed',async()=>{
  for(const forged of [
    {...catalog,source:{...catalog.source,sourceSha:'b'.repeat(40)}},
    {...catalog,source:{...catalog.source,currentness:'STALE'}},
    {...catalog,versionsByProject:{[repository]:[]}}
  ]){
    const result=await createUiSettingsHost({
      ...opts,nativeC:{catalog:async()=>forged}}).settings();
    assert.equal(result.connected,false);
  }
  let reads=0;
  const host=createUiSettingsHost({...opts,nativeC:{
    catalog:async()=>++reads===1?catalog:{...catalog,
      source:{...catalog.source,sourceSha:'b'.repeat(40)}}}});
  assert.equal((await host.settings()).reason,'SETTINGS_SOURCE_CHANGED_DURING_READ');
});
test('invalid repository and unsupported default branch never expose config',()=>{
  assert.throws(()=>createUiSettingsHost({...opts,repository:'other'}),/TRUSTED_SETTINGS/);
  assert.throws(()=>createUiSettingsHost({...opts,versionId:'version/v3'}),/TRUSTED_SETTINGS/);
});
