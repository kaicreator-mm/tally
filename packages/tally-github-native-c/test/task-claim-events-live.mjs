import assert from 'node:assert/strict';
import {readGitHubClaimEventCandidates} from '../src/task-claim-events.mjs';
const expectedSourceSha=process.env.TALLY_NATIVE_EXPECTED_SHA;
assert.match(expectedSourceSha??'',/^[a-f0-9]{40}$/);
const result=await readGitHubClaimEventCandidates({
  token:process.env.GITHUB_TOKEN,owner:'kaicreator-mm',repo:'tally',
  issueNumber:125,expectedSourceSha
});
assert.equal(result.claimStatus,'NOT_VERIFIED');
assert.equal(result.admitted,false);
assert.equal(result.taskPass,false);
assert.equal(result.releasePass,false);
assert.ok(['NO_DISPATCH_CLAIM_EVENTS','CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'].includes(result.reason),result.reason);
console.log(JSON.stringify({sourceSha:expectedSourceSha,
  reason:result.reason,candidateCount:result.candidateCount,
  claimStatus:result.claimStatus,admitted:result.admitted}));
console.log('T06B_REAL_ADS_V3_CLAIM_NONAUTHORITY=PASS');
