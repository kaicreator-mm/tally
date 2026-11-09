import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiNativeAuditHost} from './ui-audit-host.mjs';
const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const cat={connected:true,source};
const pair=n=>({
  sourceQualified:true,status:'SOURCE_PAIR_OBSERVED',
  sourceSha:sha,issueRef:'https://github.com/org/tally/issues/'+n,
  claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
  taskPass:false,releasePass:false,
  workItem:{claimStatus:'NOT_VERIFIED'},
  claimEvents:{candidateCount:1,candidates:[{
    sourceSha:sha,dispatchId:'observed-comment-123',
    claimAccepted:false,sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM'}]}
});
const opts={repository,versionId,expectedSha:sha,taskIssueNumbers:[210,211],
  nativeC:{catalog:async()=>cat},
  readPair:async({issueNumber})=>pair(issueNumber)};
test('candidate claims remain observed-only and cannot imply Agent execution',async()=>{
  const host=createUiNativeAuditHost(opts);
  const result=await host.audit({projectId:repository});
  assert.equal(result.connected,true);
  assert.equal(result.records.length,2);
  assert.equal(result.records[0].state,'CLAIM_NOT_VERIFIED');
  assert.equal(result.records[0].authority,'OBSERVED');
  assert.equal(result.claimAcceptance,'NOT_VERIFIED');
  assert.equal(result.rosterCoverage,'UNVERIFIED');
  assert.equal(JSON.stringify(result).includes('observed-comment-123'),false);
});
test('scope mismatch refuses native read',async()=>{
  let calls=0;
  const host=createUiNativeAuditHost({...opts,readPair:async()=>{calls++;return pair(210)}});
  assert.equal((await host.audit({projectId:'evil/repo'})).connected,false);
  assert.equal(calls,0);
});
test('forged accepted Claim is rejected, not shown as verified',async()=>{
  const host=createUiNativeAuditHost({...opts,readPair:async({issueNumber})=>({
    ...pair(issueNumber),admitted:true})});
  assert.equal((await host.audit({projectId:repository})).reason,
    'NATIVE_ADS_CLAIM_SOURCE_UNQUALIFIED');
});
test('incomplete native event pages and foreign Issue provenance fail closed',async()=>{
  const bad=createUiNativeAuditHost({...opts,readPair:async({issueNumber})=>({
    ...pair(issueNumber),claimEvents:{candidateCount:2,candidates:[{}]}})});
  assert.equal((await bad.audit({projectId:repository})).connected,false);
  const foreign=createUiNativeAuditHost({...opts,readPair:async({issueNumber})=>({
    ...pair(issueNumber),issueRef:'https://github.com/evil/repo/issues/'+issueNumber})});
  assert.equal((await foreign.audit({projectId:repository})).connected,false);
});
test('missing frozen selected Task roster and transport identity are refused',()=>{
  assert.throws(()=>createUiNativeAuditHost({...opts,taskIssueNumbers:[]}),
    /TRUSTED_ADS_CLAIM_AUDIT_SOURCE_REQUIRED/);
  assert.throws(()=>createUiNativeAuditHost({...opts,readPair:null,token:''}),
    /TRUSTED_ADS_CLAIM_AUDIT_SOURCE_REQUIRED/);
});
test('moving version SHA after first read denies whole audit ledger',async()=>{
  let reads=0;
  const host=createUiNativeAuditHost({...opts,nativeC:{
    catalog:async()=>({connected:true,source:{...source,
      sourceSha:++reads===1?sha:'b'.repeat(40)}})}});
  assert.equal((await host.audit({projectId:repository})).reason,
    'AUDIT_VERSION_MOVED_DURING_READ');
});
