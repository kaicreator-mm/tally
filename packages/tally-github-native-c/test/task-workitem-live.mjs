import assert from 'node:assert/strict';
import {readGitHubWorkItemSource} from '../src/task-workitem-source.mjs';

const sourceSha=process.env.TALLY_NATIVE_EXPECTED_SHA;
assert.match(sourceSha??'',/^[a-f0-9]{40}$/);
const result=await readGitHubWorkItemSource({
  token:process.env.GITHUB_TOKEN,
  owner:'kaicreator-mm',repo:'tally',issueNumber:125,
  expectedSourceSha:sourceSha
});
// Actual GitHub Issue #125 is an openly tracked task proposal with NO
// accepted CAS Start Record. Even if a future owner labels it READY, this
// unprivileged C must not turn a label into execution authority.
assert.equal(result.issueState,'open');
assert.equal(result.dependencyState,'CLEAR');
assert.equal(result.blockerCount,0);
assert.equal(result.claimStatus,'NOT_VERIFIED');
assert.equal(result.admitted,false);
assert.equal(result.taskPass,false);
assert.equal(result.releasePass,false);
assert.ok(['CANONICAL_WORK_ITEM_STATE_UNVERIFIED',
  'CLAIM_NOT_VERIFIED','WORK_ITEM_NOT_READY'].includes(result.reason));
console.log(JSON.stringify({
  native:'real-github-issue-dependencies',
  sourceSha,
  dependencyState:result.dependencyState,
  blockerCount:result.blockerCount,
  claimStatus:result.claimStatus,
  reason:result.reason,
  admitted:result.admitted,taskPass:result.taskPass
}));
console.log('T06A_ACTUAL_NATIVE_WORKITEM_DENY=PASS');
