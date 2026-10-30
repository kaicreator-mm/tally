import test from 'node:test';
import assert from 'node:assert/strict';
import {buildAcceptanceItems,acceptanceCoverage,formatAcceptanceTable} from '../src/acceptance.mjs';
const observedAsOf='2026-10-30T18:12:00.000Z';
const baseSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const attemptA={state:'LOCAL_CANDIDATE_READY',exitCode:0,tree:'cfccc2b8fe31608588f39d81754caaefa6bccec3',
  commit:'aac503ff8f9e9c9bef820a9130ae4541bd1a15ec',candidateCommit:'aac503ff8f9e9c9bef820a9130ae4541bd1a15ec',
  baseSha,worktree:'C:/worktrees/a',writeSet:'CONCURRENT_A.md',confined:true};
const attemptB={state:'LOCAL_CANDIDATE_READY',exitCode:0,tree:'a8c39ab847742399df162c40c643b0406fb2a3e8',
  commit:'7e0770d3780acbce8a5146226cef66eb0fdfe738',candidateCommit:'7e0770d3780acbce8a5146226cef66eb0fdfe738',
  baseSha,worktree:'C:/worktrees/b',writeSet:'CONCURRENT_B.md',confined:true};
const evidence={observedAsOf,
  t06m:{pr:175,pipeline:113,mergedSha:'ca09511d49488785c9ae8ff436c5db98f31b6714',
    reviewSha:'5ceebb79d2cf2ab75815e3298c66ba5eb2d92f3c',sourceReview:'PASS',baseSha,
    attempt:{...attemptA,tree:'e404da976b897efed3299195c002edd19da67d3b',
      commit:'2ec331fc237a260c74afecf8edbe1b3361b48f68',worktree:'C:/worktrees/single'},
    confinement:{builderPrincipal:'nilar\\codexsandboxonline',operatorCredentialsDenied:true,remoteWriteDenied:true}},
  t06n:{pr:180,pipeline:122,mergedSha:'0bd4da9b38622148d65b81ab62946bbf80baaa64',
    reviewSha:'4866ef66d2cd2f47cfcf5a032eab46ddb4fc9aff',sourceReview:'PASS',baseSha,
    attemptA,attemptB,overlapMs:13149,reviewJoin:'REVIEW_JOINED',repair:'LOCAL_CANDIDATE_READY',rejoin:'REVIEW_JOINED'}};
test('T06m/T06n AcceptanceItems prove selected behavior without Release promotion',()=>{
  const items=buildAcceptanceItems(evidence);
  assert.equal(items.length,7);
  for(const item of items){assert.equal(item.status,'PHYSICAL_BEHAVIOR_PROVEN');
    assert.equal(item.released,false);assert.equal(item.taskPass,false);assert.equal(item.hiddenPass,false);
    assert.equal(item.mustRetain,'PRESERVE');assert.equal(item.observedAsOf,observedAsOf);}
  assert.equal(items.find(x=>x.vector==='V52').coverage,'PARTIAL');
  assert.equal(items.find(x=>x.vector==='V66').coverage,'PARTIAL');
  assert.equal(items.find(x=>x.vector==='V66').evidenceRefs.hiddenVerifier,'NOT_QUALIFIED');
  const coverage=acceptanceCoverage(items);
  assert.equal(coverage.items.PHYSICAL_BEHAVIOR_PROVEN,7);
  assert.equal(coverage.comparison,'COUNTS_NOT_PERCENTAGES');
  const table=formatAcceptanceTable(items);
  assert.match(table,/V63/);assert.match(table,/PHYSICAL_BEHAVIOR_PROVEN/);
  assert.match(table,/PR #180 \/ pipeline 122/);assert.doesNotMatch(table,/RELEASED/);
});
test('API presence without real selected invocation stays SOURCE_IMPLEMENTED',()=>{
  const missing={...evidence,t06m:{...evidence.t06m,attempt:{...evidence.t06m.attempt,state:'MODULE_LOADED'}}};
  const item=buildAcceptanceItems(missing).find(x=>x.vector==='V52');
  assert.equal(item.status,'SOURCE_IMPLEMENTED');assert.equal(item.released,false);
});
test('stale or mismatched source cannot produce physical behavior proof',()=>{
  const stale={...evidence,t06n:{...evidence.t06n,sourceReview:'FAIL'}};
  for(const vector of ['V56','V63','V72','V73'])
    assert.equal(buildAcceptanceItems(stale).find(x=>x.vector===vector).status,'SOURCE_IMPLEMENTED');
});
test('overlap alone is not accepted without disjoint candidates and Reviewer Join',()=>{
  const overlap={...evidence,t06n:{...evidence.t06n,attemptB:{...attemptB,worktree:attemptA.worktree}}};
  for(const vector of ['V56','V63','V72','V73'])
    assert.equal(buildAcceptanceItems(overlap).find(x=>x.vector===vector).status,'SOURCE_IMPLEMENTED');
});
test('missing observation time yields UNKNOWN, never zero or PASS',()=>{
  const items=buildAcceptanceItems({...evidence,observedAsOf:undefined});
  assert.equal(items.length,1);assert.equal(items[0].status,'UNKNOWN');
  assert.equal(items[0].coverage,'UNKNOWN');assert.equal(items[0].released,false);
});
