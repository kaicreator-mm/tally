/**
 * Real native GitHub read-only transport smoke on an existing Tally Issue.
 * Does NOT claim there is any published ADS v5 report on this Issue.
 */
import assert from 'node:assert/strict';
import { readGitHubIssueReports } from '../src/read-only.mjs';
const token=process.env.GITHUB_TOKEN;
if(!token)throw new Error('GITHUB_TOKEN_NOT_CONFIGURED');
const sourceSha='db763ec963df07b31d45bd7f65049e4a8c795bc4';
const r=await readGitHubIssueReports({
  token,owner:'kaicreator-mm',repo:'tally',issueNumber:111,
  expectedSourceSha:sourceSha,
  publisherBindings:{'task-meta:kaicreator-mm/tally':['github-actions[bot]']},
  limit:1000
});
// This smoke tests authentic REST Issue/commit/pagination reads, NOT Task PASS.
// Ordinary human discussion should not become a Report.
assert.equal(r.reason,'NO_ADS_V5_REPORTS',JSON.stringify({status:r.status,reason:r.reason}));
assert.equal(r.status,'UNKNOWN');
assert.equal(r.authorized,false);
assert.equal(r.taskPass,false);
assert.equal(r.releasePass,false);
assert.ok(r.observationCount>=0);
console.log(JSON.stringify({nativeProvider:'GitHub REST',authScope:'read-only',
  issue:'#111',sourceSha,observedComments:r.observationCount,
  reason:r.reason,taskPass:false,releasePass:false}));
console.log('T03B_NATIVE_GITHUB_READ_SMOKE=PASS');
