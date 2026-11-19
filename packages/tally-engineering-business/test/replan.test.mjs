import test from 'node:test';
import assert from 'node:assert/strict';
import {planReplanCandidate,replanCandidateDigest,REPLAN_REASON_CODES}
  from '../src/replan.mjs';

const REPO='kaicreator-mm/tally',VERSION='version/v2';
const SHA='c1523a23a3429649fc7e87caa189e61bcf483668',OTHER_SHA='a'.repeat(40);
const AS_OF='2026-11-19T00:00:00.000Z';
const ref=n=>`https://github.com/${REPO}/issues/${n}`;
const task=(issueNumber,extra={})=>({issueNumber,title:`Task ${issueNumber}`,
  state:'OPEN',stateReason:undefined,blockedBy:[],writeSet:[],...extra});
const WRITE_SET=['packages/tally-engineering-business/src/replan.mjs',
  'web/v2/replan-controller.mjs'].sort();
const claim=(extra={})=>({repository:REPO,protectedClaimKey:`${REPO}#273:builder:__default__`,
  dispatchId:'dispatch:273:f30:host:20261119:1',sourceSha:SHA,phase:'PUBLISHED_VERIFIED',
  writeSet:WRITE_SET,writeSetSha256:'8'.repeat(64),...extra});
const proposal=(extra={})=>({reason:'Unblock #264 integration with a governed replan',
  evidence:[{sourceRef:ref(273),observedAt:AS_OF}],
  added:[task(300,{title:'New bounded work',writeSet:['web/v2/replan-extra.mjs']})],
  changed:[],removed:[],...extra});
const dag=(extra={})=>({repository:REPO,observedAt:AS_OF,sourceSha:SHA,tasks:[
  task(219,{state:'CLOSED',stateReason:'COMPLETED'}),
  task(276,{state:'CLOSED',stateReason:'NOT_PLANNED'}),
  task(273,{blockedBy:[219]})],...extra});
const priorRevision=(extra={})=>({revisionId:'planrevision:version/v2:000000:baseline',
  sourceSha:SHA,createdAt:'2026-11-01T00:00:00.000Z',
  acceptedTasks:[task(219,{state:'CLOSED',stateReason:'COMPLETED'})],
  unfinishedTasks:[task(273)],...extra});
const ownerDecision=(extra={})=>({repository:REPO,decisionRef:`${ref(273)}#issuecomment-6994136047`,
  decidedAt:'2026-11-18T22:59:47.000Z',sourceSha:SHA,decision:'APPROVED',...extra});
const input=(extra={})=>({repository:REPO,versionId:VERSION,sourceSha:SHA,observedAsOf:AS_OF,
  priorRevision:priorRevision(),observedDag:dag(),proposal:proposal(),
  ownerDecision:ownerDecision(),claim:claim(),liveWriteSetClaims:[],...extra});

test('P1 source-current approved proposal produces a deterministic advisory candidate',()=>{
  const first=planReplanCandidate(input()),second=planReplanCandidate(input());
  assert.equal(first.status,'CANDIDATE');
  assert.equal(first.candidate.candidateDigest,second.candidate.candidateDigest);
  assert.equal(first.candidate.predecessorRevisionId,'planrevision:version/v2:000000:baseline');
  assert.deepEqual(first.candidate.sourcePins,{repository:REPO,versionId:VERSION,
    sourceSha:SHA,observedAsOf:AS_OF});
  assert.equal(first.coverage.addedCount,1);
  assert.equal(first.coverage.preservedAcceptedCount,2);
  assert.equal(first.coverage.carriedUnfinishedCount,1);
  assert.equal(first.advisoryOnly,true);
  assert.match(first.candidate.mustNot.join(' '),/DO_NOT_TREAT_CANDIDATE_AS_NATIVE_TASK/);
  assert.equal(first.coverage.taskPass,false);
  assert.equal(first.candidate.authority.claimRef,`${REPO}#273:builder:__default__`);
});
test('P2 accepted and terminal work is preserved verbatim and unfinished carries forward',()=>{
  const result=planReplanCandidate(input({proposal:proposal({removed:[
    {issueNumber:273,reason:'superseded by bounded re-plan'}],supersessionCandidates:[273]})}));
  assert.equal(result.status,'CANDIDATE');
  const preserved=result.candidate.preservedAcceptedTasks.map(row=>row.issueNumber);
  assert.deepEqual(preserved.sort(),[219,276]);
  const preservedTask=result.candidate.preservedAcceptedTasks.find(row=>row.issueNumber===219);
  assert.equal(preservedTask.state,'CLOSED');
  assert.equal(preservedTask.stateReason,'COMPLETED');
  assert.deepEqual(result.candidate.carriedUnfinishedTasks,[]);
  assert.deepEqual(result.candidate.supersessionCandidates,[273]);
});
test('R01 OLD_PRODUCT_SHA stale Owner decision refuses activation binding',()=>{
  const result=planReplanCandidate(input({ownerDecision:ownerDecision({sourceSha:OTHER_SHA})}));
  assert.equal(result.status,'STALE');
  assert.equal(result.reason,'OLD_PRODUCT_SHA');
  assert.equal(result.candidate,null);
});
test('R02 MOVED_HEAD claim or DAG observation not on current source refuses',()=>{
  const movedClaim=planReplanCandidate(input({claim:claim({sourceSha:OTHER_SHA})}));
  assert.equal(movedClaim.status,'STALE');
  assert.equal(movedClaim.reason,'MOVED_HEAD');
  const movedDag=planReplanCandidate(input({observedDag:dag({sourceSha:OTHER_SHA})}));
  assert.equal(movedDag.status,'STALE');
  assert.equal(movedDag.reason,'MOVED_HEAD');
});
test('R02 STALE_PREDECESSOR a moved SHA never self-rebases prior-revision authority',()=>{
  const result=planReplanCandidate(input({priorRevision:priorRevision({sourceSha:OTHER_SHA})}));
  assert.equal(result.status,'STALE');
  assert.equal(result.reason,'STALE_PREDECESSOR');
});
test('R03 CYCLIC_DEPENDENCIES cycle is rejected with witness and zero mutation',()=>{
  const result=planReplanCandidate(input({proposal:proposal({added:[
    task(301,{blockedBy:[302],writeSet:['web/v2/one.mjs']}),
    task(302,{blockedBy:[301],writeSet:['web/v2/two.mjs']})]})}));
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'CYCLIC_DEPENDENCIES');
  assert.ok(Array.isArray(result.witness.cyclePath)&&result.witness.cyclePath.length===2);
  assert.equal(result.candidate,null);
});
test('R04 ACCEPTED_TASK_DROPPED terminal facts cannot be deleted or rewritten',()=>{
  const removed=planReplanCandidate(input({proposal:proposal({removed:[
    {issueNumber:219,reason:'obsolete'}],supersessionCandidates:[219]})}));
  assert.equal(removed.status,'BLOCKED');
  assert.equal(removed.reason,'ACCEPTED_TASK_DROPPED');
  assert.deepEqual(removed.witness.droppedTaskNumbers,[219]);
  const rewritten=planReplanCandidate(input({proposal:proposal({changed:[
    task(276,{title:'Rewritten disposition',writeSet:['web/v2/x.mjs']})]})}));
  assert.equal(rewritten.status,'BLOCKED');
  assert.equal(rewritten.reason,'ACCEPTED_TASK_DROPPED');
});
test('R09 WRONG_REPO foreign scope refuses before any write',()=>{
  const result=planReplanCandidate(input({repository:'other-org/other-repo'}));
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'WRONG_REPO');
});
test('R10 DUPLICATE_TASK_ID duplicate key or conflicting revision is rejected',()=>{
  const existing=planReplanCandidate(input({proposal:proposal({added:[task(273)]})}));
  assert.equal(existing.status,'BLOCKED');
  assert.equal(existing.reason,'DUPLICATE_TASK_ID');
  const twin=planReplanCandidate(input({proposal:proposal({added:[task(300),task(300)]})}));
  assert.equal(twin.status,'BLOCKED');
  assert.equal(twin.reason,'DUPLICATE_TASK_ID');
});
test('changed or removed work must reference an existing unfinished Task',()=>{
  const result=planReplanCandidate(input({proposal:proposal({changed:[task(999)]})}));
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'UNKNOWN_TASK_REFERENCE');
});
test('R06 WRITE_SET_OVERLAP competing live Writer is denied until serialized',()=>{
  const result=planReplanCandidate(input({liveWriteSetClaims:[
    {protectedClaimKey:`${REPO}#274:builder:__default__`,
      writeSet:['web/v2/replan-extra.mjs']}]}));
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'WRITE_SET_OVERLAP');
  assert.equal(result.witness.competingClaims[0].protectedClaimKey,
    `${REPO}#274:builder:__default__`);
  const ownClaim=planReplanCandidate(input({liveWriteSetClaims:[
    {protectedClaimKey:`${REPO}#273:builder:__default__`,writeSet:WRITE_SET}]}));
  assert.equal(ownClaim.status,'CANDIDATE');
});
test('R11 CARRIED_WORK_LOST removal without supersession or change is refused',()=>{
  const result=planReplanCandidate(input({proposal:proposal({removed:[
    {issueNumber:273,reason:'dropped quietly'}]})}));
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'CARRIED_WORK_LOST');
  assert.deepEqual(result.witness.untraceableTaskNumbers,[273]);
});
test('R07 NO_OWNER_DECISION the advisory candidate still renders without authority',()=>{
  const result=planReplanCandidate(input({ownerDecision:undefined}));
  assert.equal(result.status,'CANDIDATE');
  assert.equal(result.coverage.ownerDecisionBound,false);
  assert.equal(result.candidate.authority.ownerDecisionRef,null);
  assert.equal(result.candidate.authority.ownerDecisionBound,false);
});
test('R08 NO_NEW_TASK_CLAIM missing Claim evidence renders without claim binding',()=>{
  const result=planReplanCandidate(input({claim:undefined}));
  assert.equal(result.status,'CANDIDATE');
  assert.equal(result.coverage.claimBound,false);
  assert.equal(result.candidate.authority.claimRef,null);
});
test('R14 OWNER_DECISION_REJECTED preserves the last accepted baseline',()=>{
  const result=planReplanCandidate(input({ownerDecision:ownerDecision({decision:'REJECTED'})}));
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'OWNER_DECISION_REJECTED');
});
test('R17 malformed or forged evidence shapes never produce a candidate',()=>{
  const malformed=planReplanCandidate(input({ownerDecision:ownerDecision({decision:'MAYBE'})}));
  assert.equal(malformed.status,'UNKNOWN');
  assert.equal(malformed.reason,'MALFORMED_INPUT');
  const badEvidence=planReplanCandidate(input({proposal:proposal({
    evidence:[{sourceRef:'https://attacker.example/issue/1',observedAt:AS_OF}]})}));
  assert.equal(badEvidence.status,'UNKNOWN');
  const empty=planReplanCandidate();
  assert.equal(empty.status,'UNKNOWN');
});
test('candidate digest binds content: a changed proposal changes the digest',()=>{
  const base=planReplanCandidate(input()).candidate;
  const changed=planReplanCandidate(input({proposal:proposal({reason:'Different reason'})})).candidate;
  assert.notEqual(base.candidateDigest,changed.candidateDigest);
  assert.equal(replanCandidateDigest(base),base.candidateDigest);
  assert.match(base.candidateDigest,/^[0-9a-f]{64}$/);
});
test('baseline revision carries observed-DAG unfinished work without a prior revision',()=>{
  const result=planReplanCandidate(input({priorRevision:null}));
  assert.equal(result.status,'CANDIDATE');
  assert.equal(result.candidate.predecessorRevisionId,null);
  assert.deepEqual(result.candidate.carriedUnfinishedTasks.map(row=>row.issueNumber),[273]);
  assert.equal(result.coverage.preservedAcceptedCount,2);
  assert.equal(result.coverage.carriedUnfinishedCount,1);
});
test('REPLAN_REASON_CODES enumerates the typed refusal surface',()=>{
  assert.ok(REPLAN_REASON_CODES.includes('ACCEPTED_TASK_DROPPED'));
  assert.ok(REPLAN_REASON_CODES.includes('CARRIED_WORK_LOST'));
  assert.ok(REPLAN_REASON_CODES.includes('WRITE_SET_OVERLAP'));
});
