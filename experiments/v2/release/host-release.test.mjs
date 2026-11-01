import test from 'node:test';
import assert from 'node:assert/strict';
import {createReleaseHost} from './host-release.mjs';
const head='a'.repeat(40),target='b'.repeat(40),headTree='c'.repeat(40),targetTree='d'.repeat(40);
const input={pullNumber:222,headSha:head,headTree,targetSha:target,targetTree};
const taskClaimSha='1'.repeat(40);
const env={TALLY_RELEASE_SOURCE_SHA:head,TALLY_TASK_CLAIM_222_SOURCE_SHA:taskClaimSha,
  TALLY_RELEASE_PHYSICAL_EFFECTS:'1',
  TALLY_RELEASE_CAS_PATH:'C:/host/release.sqlite'};
function mock(opts={}){
  let events=[],stored=new Map();
  const ports={verifyNativeClaim:async()=>{events.push('claim');return opts.claim??{state:'ACCEPTED',sourceSha:taskClaimSha};},
    verifyWriteSet:async()=>{events.push('writeSet');return {state:'FROZEN_VERIFIED'};},
    readIndependentRq:async x=>{events.push('rq');return opts.rq??{
      state:'QUALIFIED',verifierIndependentlyBound:true,sourceSha:x.sourceSha,sourceTree:x.sourceTree,
      targetSha:x.targetSha,targetTree:x.targetTree,operation:x.operation,
      nativeEvidenceRef:'https://github.com/kaicreator-mm/tally/issues/208'};},
    readOwnerGrant:async x=>{events.push('grant');return opts.grant??{state:'GRANTED',
      exactScope:'kaicreator-mm/tally:version/v2->main',sourceSha:x.sourceSha,
      targetSha:x.targetSha,operation:x.operation,ownerIdentity:'owner:1',
      grantRef:'https://github.com/kaicreator-mm/tally/issues/208'};},
    openCasStore:async()=>({get:async k=>stored.get(k)||null,
      put:async(k,v,{mustBeAbsent})=>{if(mustBeAbsent&&stored.has(k))throw Error('CAS');stored.set(k,v);},
      close:async()=>{}}),
    // A missing Octokit guarantees that the test cannot mutate GitHub.
    octokit:null,env:opts.env??env};
  return {host:createReleaseHost({...ports,...opts.override}),events,stored};
}
const denied=r=>{assert.equal(r.authorized,false);assert.equal(r.releasePass,false);
  assert.equal(r.taskPass,false);assert.equal(r.canPublish,false);};
test('default and missing ports cannot execute a release',async()=>{
  const h=createReleaseHost();assert.equal((await h.merge(input)).state,'NOT_ADMITTED');
  const noEffects=mock({env:{TALLY_RELEASE_SOURCE_SHA:head,TALLY_TASK_CLAIM_222_SOURCE_SHA:taskClaimSha}});
  const r=await noEffects.host.merge(input);
  assert.equal(r.reason,'RELEASE_PHYSICAL_EFFECT_DISABLED');denied(r);
  assert.equal(noEffects.stored.size,0);
});
test('stale SHA, missing Claim, malformed source RQ or grant always deny',async()=>{
  const bad=mock();assert.equal((await bad.host.merge({...input,headSha:target})).reason,'RELEASE_SOURCE_NOT_PINNED');
  assert.equal(bad.events.length,0);
  const noClaim=mock({claim:{state:'NOT_VERIFIED'}});
  assert.equal((await noClaim.host.merge(input)).reason,'NATIVE_CLAIM_NOT_VERIFIED');
  assert.deepEqual(noClaim.events,['claim']);
  const crossClaim=mock({claim:{state:'ACCEPTED',sourceSha:head}});
  assert.equal((await crossClaim.host.merge(input)).reason,'NATIVE_CLAIM_NOT_VERIFIED');
  const unqualified=mock({rq:{state:'QUALIFIED',verifierIndependentlyBound:false}});
  assert.equal((await unqualified.host.merge(input)).reason,'INDEPENDENT_RQ_NOT_QUALIFIED');
  const noGrant=mock({grant:{state:'UNKNOWN'}});
  assert.equal((await noGrant.host.merge(input)).reason,'OWNER_EXACT_RELEASE_SCOPE_MISSING');
  for(const x of [noClaim,unqualified,noGrant])assert.equal(x.stored.size,0);
});
test('valid Host witnesses are insufficient without real verified transport',async()=>{
  const unavailable=mock();
  const deniedTransport=await unavailable.host.merge(input);
  assert.equal(deniedTransport.state,'NOT_ADMITTED');
  assert.equal(unavailable.stored.size,0,'missing transport must not create a pending CAS');
  const octokit={rest:{git:{
    getRef:async({ref})=>({data:{object:{sha:ref==='heads/main'?target:head}}}),
    getCommit:async({commit_sha:sha})=>({data:{sha,tree:{sha:sha===head?headTree:targetTree}}})
  },pulls:{get:async()=>({data:{state:'open',mergeable:true,
    head:{ref:'version/v2',sha:head,repo:{full_name:'kaicreator-mm/tally'}},
    base:{ref:'main',sha:target,repo:{full_name:'kaicreator-mm/tally'}}}}),
    merge:async()=>{throw new Error('lost acknowledgment');}}}};
  const ctx=mock({override:{octokit}}),r=await ctx.host.merge(input);
  assert.deepEqual(ctx.events,['claim','writeSet','rq','grant']);
  assert.equal(ctx.stored.size,1);
  assert.equal(r.state,'GITHUB_EFFECT_UNKNOWN');denied(r);
  assert.equal(r.githubEffect,'UNKNOWN');
  const replay=await ctx.host.merge(input);
  assert.equal(replay.state,'RECONCILIATION_REQUIRED');denied(replay);
  assert.equal(ctx.stored.size,1);
});
test('tag cannot reuse premerge RQ or bypass Host CAS',async()=>{
  const ctx=mock({env:{...env,TALLY_RELEASE_TAG_SOURCE_SHA:target}});
  assert.equal((await ctx.host.tag({tag:'v2.0.0',commitSha:head,commitTree:headTree})).reason,
    'TAG_SOURCE_NOT_PINNED');
  const r=await ctx.host.tag({tag:'v2.0.0',commitSha:target,commitTree:targetTree});
  assert.equal(r.reason,'TAG_EFFECT_REQUIRES_HOST_CAS_AND_RELEASE_PUBLICATION_BINDING');
  denied(r);
});
