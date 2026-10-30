import test from 'node:test';
import assert from 'node:assert/strict';
import {projectWorkUnits} from '../src/work-units.mjs';

const repo='kaicreator-mm/domain-harness',observedAsOf='2026-10-31T06:30:00.000Z';
const sha='13a7745120e4a9206ea6cf2173d0ae8f6ccf59b5';
const issueRef=number=>`https://github.com/${repo}/issues/${number}`;
const claimRef=(number,id)=>`${issueRef(number)}#issuecomment-${id}`;
const unit=(key,extra={})=>({key,kind:'SOURCE_TASK',sourceSha:sha,sourceRef:issueRef(1013),writeSet:['src/host'],owner:'builder',...extra});
const claim=(issueNumber,workUnitKey,id,extra={})=>({issueNumber,workUnitKey,claimRef:claimRef(issueNumber,id),
  status:'ACCEPTED',authoritative:true,sourceSha:sha,attemptId:`attempt:${id}`,actor:'agent-a',observedAsOf,...extra});
const input=extra=>({repo,observedAsOf,...extra});

test('1013 separates one capability from multiple concerns Claims PRs and confirmed collision',()=>{
  const result=projectWorkUnits(input({containers:[{issueNumber:1013,kind:'PLANNING',workUnits:[
    unit('host-integration',{kind:'FEATURE',featureKey:'host-integration',writeSet:[]}),
    unit('I01'),unit('I02'),unit('INTEGRATION',{writeSet:['src/runtime']})]}],
    claims:[claim(1013,'I01',101),claim(1013,'I02',102,{actor:'agent-b'}),claim(1013,'INTEGRATION',103,{status:'TERMINAL'})],
    pullRequests:[1014,1015,1016].map(number=>({issueNumber:1013,number,headSha:sha,
      ref:`https://github.com/${repo}/pull/${number}`,qualified:number===1016}))}));
  assert.equal(result.status,'OBSERVED');assert.equal(result.containers[0].workUnits.length,4);
  assert.deepEqual(result.containers[0].qualifiedSourceRefs,[`https://github.com/${repo}/pull/1016`]);
  assert.equal(result.collisions[0].status,'CONFIRMED_COLLISION');
  assert.deepEqual(result.collisions[0].sharedWriteSet,['src/host']);
  assert.match(result.collisions[0].mustNot.join(' '),/DO_NOT_CANCEL_CLAIM/);
  assert.equal(result.coverage.capabilityCount,1);assert.equal(result.coverage.activeClaimCount,2);
  assert.equal(result.coverage.nativeExecutableTaskCount,2);assert.equal(result.coverage.measuredActive,'UNKNOWN');
});

test('586 campaign checkpoints preserve Terminal history and allow a later Agent Claim',()=>{
  const result=projectWorkUnits(input({containers:[{issueNumber:586,kind:'CAMPAIGN',workUnits:[
    unit('MK0',{kind:'VERIFICATION_CHECKPOINT',sourceRef:issueRef(586)}),
    unit('MK1A',{kind:'VERIFICATION_CHECKPOINT',sourceRef:issueRef(586),writeSet:['tests/mk1']})]}],
    claims:[claim(586,'MK0',201,{status:'TERMINAL'}),claim(586,'MK1A',202,{actor:'agent-b'})]}));
  const [mk0,mk1]=result.containers[0].workUnits;
  assert.equal(mk0.claim.status,'TERMINAL');assert.equal(mk0.claim.active,null);
  assert.equal(mk1.claim.status,'ACCEPTED');assert.equal(mk1.claim.active.actor,'agent-b');
  assert.equal(result.coverage.campaignCount,1);assert.equal(result.coverage.checkpointCount,2);
  assert.equal(result.coverage.activeClaimCount,1);assert.equal(result.collisions.length,0);
});

test('1354 revisions stay one repair concern and final PASS does not erase findings',()=>{
  const reviews=['R1','R5','R6','R7'].map((revision,index)=>({issueNumber:1354,concernKey:'A5',revision,
    verdict:index===3?'PASS':'BLOCK',sourceSha:index===3?'5079210cf872a12e07d2ffd8bc2c339d0d954b46':sha,
    sourceRef:`https://github.com/${repo}/pull/1355`,observedAt:`2026-10-24T0${index}:00:00.000Z`}));
  const result=projectWorkUnits(input({containers:[{issueNumber:1354,kind:'PLANNING',workUnits:[
    unit('A5',{kind:'REPAIR_CONCERN',sourceRef:issueRef(1354),sourceSha:'5079210cf872a12e07d2ffd8bc2c339d0d954b46',writeSet:[]})]}],reviews}));
  assert.equal(result.reviewHistory.length,4);assert.equal(result.reviewHistory[0].verdict,'BLOCK');
  assert.equal(result.reviewHistory[3].verdict,'PASS');assert.equal(result.coverage.repairConcernCount,1);
  assert.equal(result.coverage.reviewAttemptCount,4);assert.equal(result.coverage.capabilityCount,0);
});

test('missing write-set free-text and corrections cannot confirm collision',()=>{
  const containers=[{issueNumber:1013,kind:'PLANNING',workUnits:[unit('I01',{writeSet:[]}),unit('I02',{writeSet:[]})]}];
  const claims=[claim(1013,'I01',101),claim(1013,'I02',102)];
  const potential=projectWorkUnits(input({containers,claims}));
  assert.equal(potential.collisions[0].status,'POTENTIAL_OVERLAP');
  const freeText=projectWorkUnits(input({containers,claims:claims.map(row=>({...row,authoritative:false}))}));
  assert.equal(freeText.collisions.length,0);assert.equal(freeText.containers[0].workUnits[0].claim.status,'UNVERIFIED');
  const corrected=projectWorkUnits(input({containers,claims,corrections:[{sourceRef:claimRef(1013,101),action:'REVOCATION'}]}));
  assert.equal(corrected.containers[0].workUnits[0].claim.status,'UNKNOWN');
  assert.equal(corrected.collisions.length,0);
  assert.equal(corrected.coverage.activeClaimCount,1);
});

test('duplicate concern identity cannot reuse one Claim as a collision pair',()=>{
  const result=projectWorkUnits(input({containers:[{issueNumber:1013,kind:'PLANNING',
    workUnits:[unit('I01'),unit('I01')]}],claims:[claim(1013,'I01',101)]}));
  assert.equal(result.containers[0].workUnits[0].identityStatus,'CONFLICT');
  assert.equal(result.containers[0].workUnits[1].identityStatus,'CONFLICT');
  assert.equal(result.collisions.length,0);assert.equal(result.coverage.activeClaimCount,0);
});

test('nested malformed rows and weak native refs fail UNKNOWN',()=>{
  for(const workUnits of [[null],[42]]){
    const result=projectWorkUnits(input({containers:[{issueNumber:1013,kind:'PLANNING',workUnits}]}));
    assert.equal(result.status,'UNKNOWN');
  }
  const weak=projectWorkUnits(input({containers:[{issueNumber:1013,kind:'PLANNING',
    workUnits:[unit('I01',{sourceRef:'https://github.com/'})]}],claims:[claim(1013,'I01',101)]}));
  assert.equal(weak.containers[0].workUnits[0].identityStatus,'UNKNOWN');
  assert.equal(weak.containers[0].workUnits[0].claim.status,'UNVERIFIED');
  const badClaim=projectWorkUnits(input({containers:[{issueNumber:1013,kind:'PLANNING',workUnits:[unit('I01')]}],
    claims:[claim(1013,'I01',101,{claimRef:claimRef(1013,'abc')})]}));
  assert.equal(badClaim.containers[0].workUnits[0].claim.status,'UNVERIFIED');
  const zeroPadded=projectWorkUnits(input({containers:[{issueNumber:1013,kind:'PLANNING',workUnits:[unit('I01')]}],
    claims:[claim(1013,'I01',101,{claimRef:claimRef(1013,'007')})]}));
  assert.equal(zeroPadded.containers[0].workUnits[0].claim.status,'UNVERIFIED');
  assert.equal(projectWorkUnits(input({reviews:[{foo:'bar'}]})).status,'UNKNOWN');
  const foreignReview={issueNumber:1354,concernKey:'A5',revision:'R7',verdict:'PASS',sourceSha:sha,
    sourceRef:'https://github.com/other-org/other-repo/pull/999',observedAt:observedAsOf};
  assert.equal(projectWorkUnits(input({reviews:[foreignReview]})).status,'UNKNOWN');
});

test('malformed evidence and unpublished spans never become Task Release or active cost',()=>{
  const invalid=projectWorkUnits(null);
  assert.equal(invalid.status,'UNKNOWN');assert.equal(invalid.coverage.releasePass,false);
  const result=projectWorkUnits(input({containers:[],wallOverlapMs:13149}));
  assert.equal(result.coverage.wallOverlapMs,13149);assert.equal(result.coverage.measuredActive,'UNKNOWN');
  assert.equal(result.coverage.activeMs,null);assert.equal(result.coverage.taskPass,false);
});
