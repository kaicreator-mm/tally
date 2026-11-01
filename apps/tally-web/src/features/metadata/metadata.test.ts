import test from 'node:test';
import assert from 'node:assert/strict';
import {aggregateUsage,reportRows,missingReports,casebookLinks,
  taskDeepLink,metadataDeepLink} from './model.ts';
import type {VersionMetadata,AttemptMeta,ReportMeta,LineageMeta} from './model.ts';

const attempt=(over:Partial<AttemptMeta>):AttemptMeta=>({
  attemptId:'a1',taskId:'101',agentRole:'builder',agentId:'zcode',
  model:'glm',provider:'zai',durationMs:1000,tokensIn:10,tokensOut:20,
  costUsd:null,startedAt:'2026-11-01T00:00:00Z',
  terminatedAt:'2026-11-01T00:01:00Z',...over});
const report=(over:Partial<ReportMeta>):ReportMeta=>({
  reportId:'r1',taskId:'101',attemptId:'a1',kind:'ads-v5-summary',
  subjectSha:'a'.repeat(40),observedAt:'2026-11-01T00:00:00Z',
  evidenceRef:'https://github.com/kaicreator-mm/tally/issues/101',
  independentlyReviewed:false,...over});
const lineage=(over:Partial<LineageMeta>):LineageMeta=>({
  taskId:'101',nativeIssueRef:'https://github.com/kaicreator-mm/tally/issues/101',
  claimCommentRef:null,claimAccepted:true,productFreezeRev:1,l2FreezeRev:1,
  packDigest:'d'.repeat(64),prNumber:9,prHeadSha:'b'.repeat(40),
  reviewState:'JOINED',ciState:'SUCCESS',
  fullHiddenCritical:{full:'PASS',hidden:'PASS',critical:'PASS'},
  releaseReceiptRef:null,deploymentReceiptRef:null,...over});
const version=(over:Partial<VersionMetadata>):VersionMetadata=>({
  versionId:'v1',
  attempts:[attempt({}),attempt({attemptId:'a2',taskId:'102',
    agentRole:'reviewer',agentId:'kimi',durationMs:null,tokensIn:null,
    tokensOut:null})],
  reports:[report({}),report({reportId:'r2',taskId:'102',
    independentlyReviewed:true,subjectSha:'b'.repeat(40)})],
  lineage:[lineage({})],
  learnedOutcomes:[{learningId:'l1',taskId:'101',decision:'ADOPTED',
    attested:null}],...over});
test('usage aggregates carry coverage; UNKNOWN never becomes zero',()=>{
  const usage=aggregateUsage(version({}));
  assert.equal(usage.attempts,2);
  assert.equal(usage.measuredDuration,1000); // only a1
  assert.equal(usage.durationCoverage,0.5);
  assert.equal(usage.tokensIn,10);
  assert.equal(usage.tokenInCoverage,0.5);
  assert.equal(usage.tokenOutCoverage,0.5);
  const empty=aggregateUsage({versionId:'v0',attempts:[],reports:[],
    lineage:[],learnedOutcomes:[]});
  assert.deepEqual([empty.durationCoverage,empty.tokenInCoverage],[null,null]);
  assert.equal(empty.attempts,0);
});
test('report rows expose review state and stale detection',()=>{
  const rows=reportRows(version({}));
  assert.equal(rows[0].reviewed,'SELF_REPORTED');
  assert.equal(rows[1].reviewed,'REVIEWED');
  assert.equal(rows[1].staleAgainst('b'.repeat(40)),false);
  assert.equal(rows[1].staleAgainst('c'.repeat(40)),true);
});
test('missing ADS v5 reports are a gap, not silent zero',()=>{
  assert.deepEqual(missingReports(version({}),['101','102','103']),['103']);
  assert.deepEqual(missingReports(version({}),['101']),[]);
});
test('cross-version casebook links without merging aggregates',()=>{
  const versions=[version({}),
    version({versionId:'v2',learnedOutcomes:[
      {learningId:'l2',taskId:'101',decision:'REJECTED',attested:true}]})];
  const links=casebookLinks(versions,'101');
  assert.deepEqual(links,[{versionId:'v1',learningId:'l1'},
    {versionId:'v2',learningId:'l2'}]);
  assert.deepEqual(casebookLinks(versions,'999'),[]);
});
test('deep links both ways encode version scoping',()=>{
  assert.equal(taskDeepLink('101','v1'),
    '#/development/tasks?projectId&versionId=v1&taskId=101');
  assert.equal(metadataDeepLink('101','v1'),
    '#/governance/metadata?versionId=v1&taskId=101');
  assert.equal(taskDeepLink('a b','v 1'),
    '#/development/tasks?projectId&versionId=v%201&taskId=a%20b');
});
