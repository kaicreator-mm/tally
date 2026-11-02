import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiRosterHost} from './ui-roster-host.mjs';

const repository='org/tally',versionId='version/v2',versionSha='a'.repeat(40);
const taskPackBlobSha='b'.repeat(40);
const taskPackRef='https://github.com/org/tally/blob/'+'c'.repeat(40)+
  '/apps/tally-web/src/features/dag/version-v2-task-pack.candidate.json';
const versionSource={repository,sourceSha:versionSha,currentness:'CURRENT',readback:'VERIFIED'};
const claimSourceSha='d'.repeat(40);
const sourceForClaim={repository,sourceSha:claimSourceSha,currentness:'PINNED',readback:'VERIFIED'};
const subject=(n)=>({issueNumber:n,sourceSha:claimSourceSha,actorRole:'builder',
  operatorKind:'other',operatorId:'zcode:task:'+n,dispatchId:'dispatch:'+n+':builder:1',
  generation:1,protectedClaimKey:repository+'#'+n+':builder:__default__',
  commentId:1000+n,commentRef:'https://github.com/'+repository+'/issues/'+n+
    '#issuecomment-'+(1000+n),candidateStatus:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM'});
const noSubject={issueNumber:2,sourceSha:'e'.repeat(40),actorRole:'builder',
  operatorKind:'other',operatorId:'unassigned:tally-v2',dispatchId:null,
  generation:1,protectedClaimKey:repository+'#2:builder:__default__',
  commentId:null,commentRef:null,candidateStatus:'NO_CLAIM_EVENT_EXPECTED'};
const edge={from:1,to:2,kind:'NATIVE',
  evidenceRef:'https://github.com/org/tally/issues/2'};
const task=id=>({id:String(id),versionId,claimState:'NOT_VERIFIED'});
const native=(edges=[{from:'1',to:'2',kind:'NATIVE',
  evidenceRef:'https://github.com/org/tally/issues/2'}])=>({
  connected:true,source:versionSource,versionId,tasks:[task(1),task(2)],
  dag:{source:versionSource,versionId,asOf:{sourceSha:versionSha},
    nodes:[{taskId:'1'},{taskId:'2'}],edges}
});
const frozen=()=>({connected:true,source:versionSource,versionId,
  taskPackRef,ownerApproval:{status:'OWNER_APPROVED',sourceSha:versionSha,
    owner:'org',approvalRef:'https://github.com/org/tally/issues/260#issuecomment-1'},
  witness:{status:'INDEPENDENTLY_VERIFIED',blobSha:taskPackBlobSha,
    sourceSha:versionSha,reviewedHeadSha:'f'.repeat(40),reviewer:'independent-reviewer',
    reviewRef:'https://github.com/org/tally/pull/258#pullrequestreview-1'},
  frozenAt:'2026-11-02T00:00:00Z',
  frozen:{blobSha:taskPackBlobSha,taskPackRef,versionId,
    taskIssues:[1,2],edges:[edge],claimSubjects:[subject(1),noSubject]}});
const acceptedClaim=(n)=>({connected:true,source:sourceForClaim,
  versionId,versionSourceSha:versionSha,issueNumber:n,claimSourceSha,
  taskPackRef,taskPackBlobSha,claimStatus:'ACCEPTED',claimAccepted:true,
  claimPublication:'REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK',
  generation:1,actorRole:'builder',operatorKind:'other',
  operatorId:'zcode:task:'+n,protectedClaimKey:repository+'#'+n+':builder:__default__',
  dispatchId:'dispatch:'+n+':builder:1',commentId:1000+n,
  commentRef:'https://github.com/'+repository+'/issues/'+n+
    '#issuecomment-'+(1000+n),commentBodySha256:'9'.repeat(64),
  writerId:'tally-host:win32:i259:closeout:1',responsibilityStatus:'NOT_VERIFIED',
  activeOperator:null,authorized:false,admitted:false,taskPass:false,releasePass:false});
const candidateClaim=(n)=>({connected:true,source:sourceForClaim,
  versionId,versionSourceSha:versionSha,issueNumber:n,claimSourceSha,
  taskPackRef,taskPackBlobSha,claimStatus:'NOT_VERIFIED',claimAccepted:false,
  claimPublication:'CANDIDATE_ONLY',commentId:1000+n,
  commentRef:'https://github.com/'+repository+'/issues/'+n+
    '#issuecomment-'+(1000+n),commentBodySha256:'8'.repeat(64),
  dispatchId:'dispatch:'+n+':builder:1',authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const notClaimed=(n)=>({connected:true,source:{...sourceForClaim,
  sourceSha:'e'.repeat(40)},versionId,versionSourceSha:versionSha,
  issueNumber:n,claimSourceSha:'e'.repeat(40),taskPackRef,taskPackBlobSha,
  claimStatus:'NOT_CLAIMED',claimAccepted:false,claimPublication:'NOT_VERIFIED',
  commentRef:null,commentId:null,authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const make=(f=async()=>frozen(),v=async()=>native(),c=null)=>{
  const calls=[];
  const host=createUiRosterHost({repository,versionId,expectedSha:versionSha,
    frozenHost:{readFrozenDag:f},
    nativeC:{version:async args=>{calls.push(args);return v(args);}},
    claimHost:c?{readAcceptedClaim:c}:undefined});
  return {host,calls};
};

test('frozen roster drives native C and keeps version SHA separate from Task Claim SHA',async()=>{
  const {host,calls}=make();
  const r=await host.roster({projectId:repository,versionId});
  assert.equal(r.connected,true);
  assert.equal(r.rosterCoverage,'VERIFIED_COMPLETE');
  assert.equal(r.dependencyCoverage,'NATIVE_COMPLETE');
  assert.equal(r.claimCoverage,'NOT_VERIFIED');
  assert.deepEqual(calls[0],{taskIssueNumbers:[1,2]});
  assert.equal(r.dag.asOf.frozenAt,'2026-11-02T00:00:00Z');
});

test('missing native dependency remains visible and is never inferred as complete',async()=>{
  const {host}=make(async()=>frozen(),async()=>native([]));
  const r=await host.roster();
  assert.equal(r.rosterCoverage,'VERIFIED_COMPLETE');
  assert.equal(r.dependencyCoverage,'NATIVE_PARTIAL');
});

test('partial selected native Task roster refuses complete coverage',async()=>{
  const {host}=make(async()=>frozen(),async()=>({...native(),tasks:[task(1)],
    dag:{...native().dag,nodes:[{taskId:'1'}]}}));
  const r=await host.roster();
  assert.equal(r.connected,false);
  assert.equal(r.rosterCoverage,'UNVERIFIED');
});

test('candidate-only Claim is visible but never counted as accepted',async()=>{
  const {host}=make(async()=>frozen(),async()=>native(),
    async args=>args.issueNumber===1?candidateClaim(1):notClaimed(2));
  const r=await host.roster();
  assert.equal(r.connected,true);
  assert.equal(r.claimReadbackCoverage,'OBSERVATIONS_COMPLETE');
  assert.equal(r.claimCoverage,'NONE_ACCEPTED');
  assert.equal(r.acceptedClaimCount,0);
  assert.equal(r.notVerifiedCount,1);
  assert.equal(r.notClaimedCount,1);
  assert.equal(r.tasks[0].claimAccepted,false);
  assert.equal(r.tasks[0].claimState,'NOT_VERIFIED');
  assert.equal(r.tasks[0].claimCandidateRef,candidateClaim(1).commentRef);
  assert.equal(r.tasks[1].claimState,'NOT_CLAIMED');
});

test('accepted CAS Claim does not imply Web responsibility or Kernel admission',async()=>{
  const {host}=make(async()=>frozen(),async()=>native(),
    async args=>args.issueNumber===1?acceptedClaim(1):notClaimed(2));
  const r=await host.roster();
  assert.equal(r.connected,true);
  assert.equal(r.claimCoverage,'PARTIAL_ACCEPTED');
  assert.equal(r.acceptedClaimCount,1);
  assert.equal(r.activeWebResponsibilityCount,0);
  assert.equal(r.tasks[0].claimAccepted,true);
  assert.equal(r.tasks[0].responsibilityStatus,'NOT_VERIFIED');
  assert.equal(r.tasks[0].activeOperator,null);
});

test('Claim subject SHA cannot be replaced with the version branch SHA',async()=>{
  const bad={...acceptedClaim(1),source:{...sourceForClaim,sourceSha:versionSha},
    claimSourceSha:versionSha};
  const {host}=make(async()=>frozen(),async()=>native(),
    async args=>args.issueNumber===1?bad:notClaimed(2));
  assert.equal((await host.roster()).reason,'ACCEPTED_CLAIM_HOST_READBACK_UNQUALIFIED');
});

test('candidate-only, self-loop and cyclic frozen DAG refuse qualification',async()=>{
  for(const x of [
    {...frozen(),witness:{...frozen().witness,status:'SELF_REPORTED'}},
    {...frozen(),frozen:{...frozen().frozen,edges:[{from:1,to:1,kind:'NATIVE',
      evidenceRef:'https://github.com/org/tally/issues/1'}]}},
    {...frozen(),frozen:{...frozen().frozen,edges:[edge,
      {from:2,to:1,kind:'NATIVE',evidenceRef:'https://github.com/org/tally/issues/1'}]}}
  ]){
    const {host}=make(async()=>x);
    assert.equal((await host.roster()).connected,false);
  }
});

test('out-of-roster native edge, duplicate task and stale source deny',async()=>{
  const badEdge={from:'3',to:'2',kind:'NATIVE',
    evidenceRef:'https://github.com/org/tally/issues/2'};
  const {host:a}=make(async()=>frozen(),async()=>native([badEdge]));
  assert.equal((await a.roster()).connected,false);
  const duplicate={...frozen(),frozen:{...frozen().frozen,taskIssues:[1,1]}};
  const {host:b}=make(async()=>duplicate);
  assert.equal((await b.roster()).connected,false);
  const stale={...frozen(),source:{...versionSource,sourceSha:'c'.repeat(40)}};
  const {host:c}=make(async()=>stale);
  assert.equal((await c.roster()).connected,false);
});

test('frozen DAG changing between reads refuses qualification',async()=>{
  let reads=0;
  const {host}=make(async()=>{
    const f=frozen(),blob=++reads===1?taskPackBlobSha:'c'.repeat(40);
    return {...f,witness:{...f.witness,blobSha:blob},
      frozen:{...f.frozen,blobSha:blob}};
  });
  assert.equal((await host.roster()).reason,'FROZEN_DAG_CHANGED_DURING_READ');
});

test('foreign scope cannot read Host',async()=>{
  let calls=0;
  const {host}=make(async()=>{calls++;return frozen();});
  assert.equal((await host.roster({projectId:'evil/repo'})).reason,'ROSTER_SCOPE_MISMATCH');
  assert.equal(calls,0);
});

test('missing Owner approval or independent reviewer denies frozen roster',async()=>{
  const {host:a}=make(async()=>({...frozen(),ownerApproval:{status:'UNKNOWN'}}));
  assert.equal((await a.roster()).connected,false);
  const {host:b}=make(async()=>({...frozen(),witness:{...frozen().witness,
    status:'SELF_REPORTED'}}));
  assert.equal((await b.roster()).connected,false);
});
