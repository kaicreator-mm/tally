import test from 'node:test';
import assert from 'node:assert/strict';
import {captureInstalledJourney,INSTALLED_GATES} from './installed-journey.mjs';
const sha='a'.repeat(40);
const verifier=async()=>({state:'VERIFIED_DISTINCT',builderSid:'S-1-5-21-100',verifierSid:'S-1-5-21-101'});
test('Linux and Node22 never claim Windows installed acceptance',async()=>{
  for(const opts of [{platform:'linux',nodeMajor:24},{platform:'win32',nodeMajor:22}]){
    const r=await captureInstalledJourney({sourceSha:sha,...opts});
    assert.equal(r.status,'NOT_RUN');assert.equal(r.releasePass,false);
  }
});
test('no real separate Hidden verifier blocks before probes',async()=>{
  let called=0;
  const r=await captureInstalledJourney({sourceSha:sha,platform:'win32',nodeMajor:24,
    verifyVerifierPrincipal:async()=>({state:'VERIFIED_DISTINCT',builderSid:'same',verifierSid:'same'}),
    runProbe:async()=>{called++;return {state:'OBSERVED'};}});
  assert.equal(r.reason,'HIDDEN_VERIFIER_ISOLATION_NOT_PROVEN');assert.equal(called,0);
});
test('eight reported Host receipts remain unqualified until independent RQ',async()=>{
  const observed=[];
  const r=await captureInstalledJourney({sourceSha:sha,platform:'win32',nodeMajor:24,
    verifyVerifierPrincipal:verifier,
    runProbe:async({gate,sourceSha})=>{observed.push(gate);
      return {state:'OBSERVED',sourceSha,observedAt:'2026-10-31T23:40:00Z',
        evidencePath:`C:/readback/${gate}.json`};}});
  assert.deepEqual(observed,INSTALLED_GATES);
  assert.equal(r.status,'CAPTURE_REQUIRES_INDEPENDENT_VERIFICATION');
  assert.equal(r.releasePass,false);assert.equal(r.qualification,'NOT_QUALIFIED');
  assert.equal(r.gates.length,8);
});
test('missing, stale and thrown Host probes fail closed and stop subsequent checks',async()=>{
  let n=0;
  const r=await captureInstalledJourney({sourceSha:sha,platform:'win32',nodeMajor:24,
    verifyVerifierPrincipal:verifier,runProbe:async({gate})=>{
      n++;if(n===2)throw new Error('network failure');
      return {state:'OBSERVED',sourceSha:sha,observedAt:'2026-11-01T00:00:00Z',evidencePath:gate};}});
  assert.equal(r.status,'INCOMPLETE');assert.equal(n,2);
  assert.equal(r.gates[1].state,'UNKNOWN');assert.equal(r.releasePass,false);
  const empty=await captureInstalledJourney({sourceSha:'bad'});
  assert.equal(empty.reason,'EXACT_SOURCE_REQUIRED');
});
