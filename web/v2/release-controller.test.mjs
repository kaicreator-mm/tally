import test from 'node:test';
import assert from 'node:assert/strict';
import {createReleaseController} from './release-controller.mjs';
const sha='a'.repeat(40),target='b'.repeat(40);
test('stale, incomplete and throwing native read fail closed',async()=>{
  assert.throws(()=>createReleaseController({sourceSha:'unknown'}),/RELEASE_SOURCE_SHA_REQUIRED/);
  const no=createReleaseController({sourceSha:sha});
  assert.equal((await no.snapshot()).status,'UNKNOWN');
  const stale=createReleaseController({sourceSha:sha,readReleaseSnapshot:async()=>({
    sourceSha:target,targetSha:target,readiness:{status:'BLOCKED'}})});
  assert.equal((await stale.snapshot()).status,'UNKNOWN');
  const fail=createReleaseController({sourceSha:sha,readReleaseSnapshot:async()=>{throw Error('500');}});
  assert.equal((await fail.snapshot()).status,'UNKNOWN');
});
test('current advisory evidence is viewable but no merge/tag/publish can execute',async()=>{
  const c=createReleaseController({sourceSha:sha,readReleaseSnapshot:async()=>({
    sourceSha:sha,targetSha:target,readiness:{status:'BLOCKED',releasePass:false,gates:[]}})});
  const before=await c.requestAction({kind:'merge',sourceSha:sha});
  assert.equal(before.reason,'NATIVE_RELEASE_EVIDENCE_NOT_READ');
  const s=await c.snapshot();assert.equal(s.status,'OBSERVED');assert.equal(s.canMerge,false);
  for(const kind of ['merge','tag','publish']){
    const r=await c.requestAction({kind,sourceSha:sha});
    assert.equal(r.reason,'INDEPENDENT_HOST_ADMISSION_NOT_CONNECTED');
    assert.equal(r.mutationPerformed,false);assert.equal(r.releasePass,false);
  }
  assert.equal((await c.requestAction({kind:'merge',sourceSha:target})).reason,'STALE_VERSION_SHA');
  assert.equal((await c.requestAction({kind:'delete-everything',sourceSha:sha})).reason,'UNKNOWN_RELEASE_OPERATION');
});
test('spurious projection Release PASS cannot become UI observed',async()=>{
  const c=createReleaseController({sourceSha:sha,readReleaseSnapshot:async()=>({
    sourceSha:sha,targetSha:target,readiness:{status:'BLOCKED',releasePass:true}})});
  assert.equal((await c.snapshot()).status,'UNKNOWN');
});
