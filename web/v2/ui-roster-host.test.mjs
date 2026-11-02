import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiRosterHost} from './ui-roster-host.mjs';
const repository='org/tally',versionId='version/v2',sha='a'.repeat(40);
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const edge={from:1,to:2}, frozen=()=>({connected:true,source,versionId,
  witness:{status:'INDEPENDENTLY_VERIFIED',blobSha:'b'.repeat(40)},
  frozen:{blobSha:'b'.repeat(40),versionId,frozenAt:'2026-11-02T00:00:00Z',
    taskIssues:[1,2],edges:[edge]}});
const task=id=>({id:String(id),versionId,claimState:'NOT_VERIFIED'});
const native=(edges=[{from:'1',to:'2',kind:'NATIVE',
  evidenceRef:'https://github.com/org/tally/issues/1'}])=>({connected:true,source,versionId,
  tasks:[task(1),task(2)],dag:{source,versionId,asOf:{sourceSha:sha},
    nodes:[{taskId:'1'},{taskId:'2'}],edges}});
const make=(f=async()=>frozen(),v=async()=>native())=>createUiRosterHost({
  repository,versionId,expectedSha:sha,
  frozenHost:{readFrozenDag:f},nativeC:{version:v}});
test('complete native roster preserves independent Claim qualification',async()=>{
  const r=await make().roster({projectId:repository,versionId});
  assert.equal(r.connected,true);
  assert.equal(r.rosterCoverage,'VERIFIED_COMPLETE');
  assert.equal(r.dependencyCoverage,'NATIVE_COMPLETE');
  assert.equal(r.claimCoverage,'NOT_VERIFIED');
  assert.equal(r.dag.asOf.frozenAt,'2026-11-02T00:00:00Z');
});
test('missing native dependency remains visible, never runnable proof',async()=>{
  const r=await make(async()=>frozen(),async()=>native([])).roster();
  assert.equal(r.rosterCoverage,'VERIFIED_COMPLETE');
  assert.equal(r.dependencyCoverage,'NATIVE_PARTIAL');
});
test('partial selected task roster refuses complete coverage',async()=>{
  const r=await make(async()=>frozen(),async()=>({...native(),tasks:[task(1)]})).roster();
  assert.equal(r.connected,false);
  assert.equal(r.rosterCoverage,'UNVERIFIED');
});
test('candidate-only, self-loop and cyclic frozen DAG refuse authority',async()=>{
  for(const x of [
    {...frozen(),witness:{...frozen().witness,status:'SELF_REPORTED'}},
    {...frozen(),frozen:{...frozen().frozen,edges:[{from:1,to:1}]}},
    {...frozen(),frozen:{...frozen().frozen,edges:[edge,{from:2,to:1}]}}
  ])assert.equal((await make(async()=>x).roster()).connected,false);
});
test('out-of-roster edge, duplicate or unqualified current source denies',async()=>{
  const nativeEdge={from:'3',to:'2',kind:'NATIVE',
    evidenceRef:'https://github.com/org/tally/issues/3'};
  assert.equal((await make(async()=>frozen(),async()=>native([nativeEdge])).roster()).connected,false);
  assert.equal((await make(async()=>({...frozen(),frozen:{...frozen().frozen,
    taskIssues:[1,1]}})).roster()).connected,false);
  assert.equal((await make(async()=>({...frozen(),source:{...source,sourceSha:'c'.repeat(40)}})).roster()).connected,false);
});
test('frozen DAG changing during read refuses qualification',async()=>{
  let reads=0;
  const host=make(async()=>{const f=frozen();return {...f,
    witness:{...f.witness,blobSha:++reads===1?'b'.repeat(40):'c'.repeat(40)},
    frozen:{...f.frozen,blobSha:reads===1?'b'.repeat(40):'c'.repeat(40)}};});
  assert.equal((await host.roster()).reason,'FROZEN_DAG_CHANGED_DURING_READ');
});
test('foreign scope cannot even read Host',async()=>{
  let calls=0;
  const host=make(async()=>{calls++;return frozen()});
  assert.equal((await host.roster({projectId:'evil/repo'})).reason,'ROSTER_SCOPE_MISMATCH');
  assert.equal(calls,0);
});

const acceptedClaim=(n)=>({connected:true,source,issueNumber:n,
  claimStatus:'ACCEPTED',claimAccepted:true,
  claimPublication:'REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK',
  generation:1,actorRole:'builder',
  protectedClaimKey:repository+'#'+n+':builder:__default__',
  dispatchId:'dispatch:'+n+':builder:1',commentId:1000+n,
  commentRef:'https://github.com/'+repository+'/issues/'+n+
    '#issuecomment-'+(1000+n),
  authorized:false,taskPass:false,releasePass:false});
const notClaimed=(n)=>({connected:true,source,issueNumber:n,
  claimStatus:'NOT_CLAIMED',claimAccepted:false,claimPublication:'NOT_VERIFIED',
  commentRef:null,commentId:null,authorized:false,taskPass:false,releasePass:false});
test('selected Host re-reads accepted CAS witness separately from ADS event candidate',async()=>{
  const host=createUiRosterHost({repository,versionId,expectedSha:sha,
    frozenHost:{readFrozenDag:async()=>frozen()},nativeC:{version:async()=>native()},
    claimHost:{readAcceptedClaim:async({issueNumber})=>acceptedClaim(issueNumber)}});
  const r=await host.roster();
  assert.equal(r.connected,true);
  assert.equal(r.claimReadbackCoverage,'VERIFIED_COMPLETE');
  assert.equal(r.claimCoverage,'ACCEPTED_ALL');
  assert.equal(r.acceptedClaimCount,2);
  assert.equal(r.tasks[0].claimState,'ACCEPTED');
  assert.equal(r.tasks[0].claimRef,
    'https://github.com/org/tally/issues/1#issuecomment-1001');
});
test('verified absence is distinct from accepted claim and partial claim coverage',async()=>{
  const host=createUiRosterHost({repository,versionId,expectedSha:sha,
    frozenHost:{readFrozenDag:async()=>frozen()},nativeC:{version:async()=>native()},
    claimHost:{readAcceptedClaim:async({issueNumber})=>
      issueNumber===1?acceptedClaim(issueNumber):notClaimed(issueNumber)}});
  const r=await host.roster();
  assert.equal(r.claimReadbackCoverage,'VERIFIED_COMPLETE');
  assert.equal(r.claimCoverage,'PARTIAL');
  assert.equal(r.acceptedClaimCount,1);
  assert.equal(r.tasks[1].claimState,'NOT_CLAIMED');
});
test('self-reported comment, changed CAS row and authority flags never qualify',async()=>{
  for(const claim of [
    {...acceptedClaim(1),claimPublication:'COMMENT_ONLY'},
    {...acceptedClaim(1),authorized:true},
    {...acceptedClaim(1),protectedClaimKey:'evil/repo#1:builder:__default__'}
  ]){
    const host=createUiRosterHost({repository,versionId,expectedSha:sha,
      frozenHost:{readFrozenDag:async()=>frozen()},nativeC:{version:async()=>native()},
      claimHost:{readAcceptedClaim:async({issueNumber})=>
        issueNumber===1?claim:acceptedClaim(issueNumber)}});
    assert.equal((await host.roster()).reason,'ACCEPTED_CLAIM_HOST_READBACK_UNQUALIFIED');
  }
});
test('Claim registry readback changing between observations refuses roster',async()=>{
  let reads=0;
  const host=createUiRosterHost({repository,versionId,expectedSha:sha,
    frozenHost:{readFrozenDag:async()=>frozen()},nativeC:{version:async()=>native()},
    claimHost:{readAcceptedClaim:async({issueNumber})=>{
      const c=acceptedClaim(issueNumber);
      if(issueNumber===1&&++reads===2)c.commentId++;
      return c;
    }}});
  assert.equal((await host.roster()).reason,'ACCEPTED_CLAIM_HOST_READBACK_UNQUALIFIED');
});
