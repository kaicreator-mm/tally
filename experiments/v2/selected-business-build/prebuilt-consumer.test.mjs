import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,writeFile,readFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {writePrebuiltConsumerManifest,readPrebuiltConsumer}
  from './prebuilt-consumer.mjs';
const sourceSha='a'.repeat(40);
const env={T06M_TARGET_REPOSITORY:'kaicreator-mm/tally-reps-test',
  T06M_TARGET_VERSION_REF:'main',T06M_TARGET_ISSUE_NUMBER:'1',
  T06M_TARGET_CANDIDATE_PATH:'health.txt',TALLY_CLAIM_SOURCE_SHA:sourceSha};
async function fixture(){
  const dir=await mkdtemp(join(tmpdir(),'tally-prebuilt-test-'));
  await writeFile(join(dir,'sealed-app.json'),'{"appId":"fake-test"}');
  for(const name of ['consumer-local-codex.mjs','builder-target.mjs',
    'local-codex-host.mjs','local-codex-linux-host.mjs'])
    await writeFile(join(dir,name),'export {};');
  return dir;
}
test('selected BUILD only writes a source/Issue/SHA/content pin',async()=>{
  const dir=await fixture();
  try{
    const manifest=await writePrebuiltConsumerManifest({dir,env});
    assert.equal(manifest.repository,'kaicreator-mm/tally-reps-test');
    assert.equal(manifest.issueNumber,1);
    assert.equal(manifest.sourceSha,sourceSha);
    assert.equal(manifest.versionId,'main');
    assert.match(manifest.sealedAppSha256,/^[a-f0-9]{64}$/);
    const result=await readPrebuiltConsumer({directory:dir,env});
    assert.match(result.manifest.runtimeDigests['local-codex-linux-host.mjs'],
      /^[a-f0-9]{64}$/);
    assert.equal(result.manifest.sealedAppSha256,manifest.sealedAppSha256);
    assert.equal(result.script,join(dir,'consumer-local-codex.mjs'));
  }finally{await rm(dir,{recursive:true,force:true});}
});
test('cross-repository, wrong issue, SHA and candidate are rejected',async()=>{
  const dir=await fixture();
  try{
    await writePrebuiltConsumerManifest({dir,env});
    for(const mutation of [
      {T06M_TARGET_REPOSITORY:'org/foreign'},
      {T06M_TARGET_ISSUE_NUMBER:'2'},
      {TALLY_CLAIM_SOURCE_SHA:'b'.repeat(40)},
      {T06M_TARGET_CANDIDATE_PATH:'src/other.mjs'}
    ])await assert.rejects(readPrebuiltConsumer({directory:dir,
      env:{...env,...mutation}}),/PREBUILT_CONSUMER_SCOPE_MISMATCH/);
  }finally{await rm(dir,{recursive:true,force:true});}
});
test('sealed app and runtime script mutations fail before Host launch',async()=>{
  const dir=await fixture();
  try{
    await writePrebuiltConsumerManifest({dir,env});
    await writeFile(join(dir,'consumer-local-codex.mjs'),'// tampered');
    await assert.rejects(readPrebuiltConsumer({directory:dir,env}),
      /PREBUILT_CONSUMER_INTEGRITY_MISMATCH/);
    await writeFile(join(dir,'consumer-local-codex.mjs'),'export {};');
    await writeFile(join(dir,'local-codex-linux-host.mjs'),'// tampered');
    await assert.rejects(readPrebuiltConsumer({directory:dir,env}),
      /PREBUILT_CONSUMER_INTEGRITY_MISMATCH/);
    await writeFile(join(dir,'local-codex-linux-host.mjs'),'export {};');
    await writeFile(join(dir,'sealed-app.json'),'{"appId":"tampered"}');
    await assert.rejects(readPrebuiltConsumer({directory:dir,env}),
      /PREBUILT_CONSUMER_INTEGRITY_MISMATCH/);
  }finally{await rm(dir,{recursive:true,force:true});}
});
test('partial selected BUILD target cannot generate manifest',async()=>{
  const dir=await fixture();
  try{
    await assert.rejects(writePrebuiltConsumerManifest({dir,
      env:{...env,T06M_TARGET_ISSUE_NUMBER:'0'}}),
    /PREBUILT_SELECTED_TARGET_INVALID/);
  }finally{await rm(dir,{recursive:true,force:true});}
});
