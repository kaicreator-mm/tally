import test from 'node:test';
import assert from 'node:assert/strict';
import {projectReleaseReadiness,RELEASE_READINESS_REQUIRED_GATES as required} from '../src/release-readiness.mjs';
const sha='5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02';
const old='3a894ddbdef8d4a8513c1c33837d32a816e2234e';
const repository='kaicreator-mm/tally';
const observedAsOf='2026-10-31T08:18:45Z';
const record=(status='PASS',subjectSha=sha)=>({status,subjectSha,observedAsOf,
  evidenceRef:`https://github.com/${repository}/commit/${sha}`});
const complete=()=>({repository,sourceSha:sha,observedAsOf,
  gates:Object.fromEntries(required.map(name=>[name,record()]))});
const noAuthority=result=>{
  assert.ok(['BLOCKED','UNKNOWN'].includes(result.status));assert.equal(result.sourceQualified,false);
  assert.equal(result.releasePass,false);assert.equal(result.taskPass,false);
  assert.equal(result.canMerge,false);assert.equal(result.canTag,false);
  assert.equal(result.canPublish,false);assert.equal(result.authorization,'NONE');
  assert.equal(result.independentRQRequired,true);
  assert.equal(result.advisoryOnly,true);
};

test('no source, malformed source, or non-ISO timestamp fail closed',()=>{
  for(const input of [null,undefined,{}, {...complete(),sourceSha:'bad'},
    {...complete(),repository:'../evil'}, {...complete(),observedAsOf:'yesterday'},
    {...complete(),gates:[]}]){
    const result=projectReleaseReadiness(input);noAuthority(result);
    assert.equal(result.status,'UNKNOWN');
  }
});
test('every missing required gate blocks, including hidden and installed acceptance',()=>{
  for(const name of required){
    const input=complete();delete input.gates[name];
    const result=projectReleaseReadiness(input);noAuthority(result);
    assert.ok(result.blockers.some(g=>g.name===name&&g.status==='NOT_RUN'),name);
  }
});
test('all reported PASS is not independent RQ or Release authority',()=>{
  const result=projectReleaseReadiness(complete());noAuthority(result);
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'INDEPENDENT_RQ_AND_AUTHORITY_NOT_ATTESTED');
  assert.equal(result.observedEvidence,'REPORTED_COMPLETE_UNVERIFIED');
  assert.equal(result.gateCounts.reportedPass,required.length);
  assert.equal(result.blockers.length,0);
  assert.equal(result.independentRQRequired,true);
});
test('failed Woodpecker candidate CI blocks even when other results reported PASS',()=>{
  const input=complete();input.gates.woodpeckerCI=record('FAIL');
  const result=projectReleaseReadiness(input);noAuthority(result);
  assert.deepEqual(result.blockers.find(g=>g.name==='woodpeckerCI'),
    {name:'woodpeckerCI',status:'FAIL',reason:'REPORTED_FAIL'});
});
test('candidate changed after review or RQ must go STALE',()=>{
  for(const name of ['independentReview','releaseQualification','ownerReleaseAuthority']){
    const input=complete();input.gates[name]=record('PASS',old);
    const result=projectReleaseReadiness(input);noAuthority(result);
    assert.deepEqual(result.blockers.find(g=>g.name===name),
      {name,status:'STALE',reason:'SOURCE_SHA_CHANGED'});
  }
});
test('missing precise subject, source or GitHub evidence cannot make PASS',()=>{
  const invalid=[{status:'PASS',evidenceRef:`https://github.com/${repository}/commit/${sha}`,observedAsOf},
    {...record(),subjectSha:'x'}, {...record(),observedAsOf:null},
    {...record(),evidenceRef:'https://ci.kaicreator.com/repos/12/pipeline/200/1'},
    {...record(),evidenceRef:'https://github.com/other/repo/issues/203'},
    {...record(),evidenceRef:'https://github.com/kaicreator-mm/tally/issues/203?promoted=1'},
    {...record(),evidenceRef:'javascript:alert(1)'}];
  for(const observation of invalid){
    const input=complete();input.gates.hiddenValidation=observation;
    const result=projectReleaseReadiness(input);noAuthority(result);
    assert.ok(result.blockers.some(g=>g.name==='hiddenValidation'),JSON.stringify(observation));
  }
});
test('partial pagination / read errors remain UNKNOWN, never inferred zero',()=>{
  const input=complete();input.gates.nativeClaims=record('UNKNOWN');
  input.gates.realHost=record('NOT_RUN');
  const result=projectReleaseReadiness(input);noAuthority(result);
  assert.equal(result.observedEvidence,'INCOMPLETE');
  assert.equal(result.gateCounts.missingOrUnqualified,2);
});
test('bad objects and spoofed PASS with unknown enums stay UNKNOWN',()=>{
  for(const recordValue of [{status:'QUALIFIED',subjectSha:sha},42,[],{status:'PASS',subjectSha:sha}]){
    const input=complete();input.gates.fullRegression=recordValue;
    const result=projectReleaseReadiness(input);noAuthority(result);
    assert.ok(result.blockers.some(g=>g.name==='fullRegression'&&g.status==='UNKNOWN'));
  }
});
test('unsafe operation rights cannot be injected into projected authority',()=>{
  const input=complete();input.releasePass=true;input.canMerge=true;input.canTag=true;
  input.gates.woodpeckerCI.canMerge=true;
  const result=projectReleaseReadiness(input);noAuthority(result);
  assert.equal(Object.isFrozen(result),true);
  assert.equal(Object.isFrozen(result.blockers),true);
  assert.equal(Object.isFrozen(result.gates),true);
  assert.equal(result.gates.every(gate=>Object.isFrozen(gate)),true);
  assert.equal(Object.isFrozen(result.gateCounts),true);
});
test('input cannot be silently mistaken for a complete Release check',()=>{
  const input=complete();input.gates={};
  const result=projectReleaseReadiness(input);noAuthority(result);
  assert.equal(result.gateCounts.required,13);
  assert.equal(result.gateCounts.reportedPass,0);
  assert.equal(result.gateCounts.missingOrUnqualified,13);
});
test('future-dated observation can never be promoted to reported PASS',()=>{
  const input=complete();
  input.gates.installedAcceptance={...record(),observedAsOf:'2026-10-31T09:00:00Z'};
  const result=projectReleaseReadiness(input);noAuthority(result);
  assert.equal(result.status,'BLOCKED');
  assert.deepEqual(result.blockers.find(g=>g.name==='installedAcceptance'),
    {name:'installedAcceptance',status:'UNKNOWN',reason:'OBSERVATION_TIME_INVALID'});
  assert.equal(result.gateCounts.reportedPass,required.length-1);
});
test('unknown extra gate keys cannot add authority or clear required blockers',()=>{
  const input=complete();delete input.gates.hiddenValidation;
  input.gates.unlistedMagicGate=record();
  const result=projectReleaseReadiness(input);noAuthority(result);
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.gateCounts.required,13);
  assert.equal(result.gateCounts.reportedPass,required.length-1);
  assert.ok(result.blockers.some(g=>g.name==='hiddenValidation'&&g.status==='NOT_RUN'));
  assert.ok(!result.blockers.some(g=>g.name==='unlistedMagicGate'));
  assert.ok(!result.gates.some(g=>g.name==='unlistedMagicGate'));
});
