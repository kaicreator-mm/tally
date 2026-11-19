import test from 'node:test';
import {createHash} from 'node:crypto';
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
  // The request-scope tag field is IGNORED: without a trusted Host-pinned
  // TALLY_RELEASE_TAG_NAME the operation refuses TAG_NAME_NOT_PINNED (the
  // historical unbound literal now fires only when a name IS pinned but
  // the TAG CAS path is not — see the #491 block below).
  const r=await ctx.host.tag({tag:'v2.0.0',commitSha:target,commitTree:targetTree});
  assert.equal(r.reason,'TAG_NAME_NOT_PINNED');
  denied(r);
});

// ---------------------------------------------------------------------------
// #491 J07 — tag delivery: durable Host CAS + real client tag creation with
// idempotent recovery. The old fail-closed bound survives as the unbound
// case (no pinned tag name / no CAS path).
// ---------------------------------------------------------------------------
const tagEnv={...env,TALLY_RELEASE_TAG_SOURCE_SHA:head,
  TALLY_RELEASE_TAG_NAME:'v2.0.0',
  TALLY_RELEASE_TAG_CAS_PATH:'C:/host/tag.sqlite'};
const tagInput={commitSha:head,commitTree:headTree};
function tagOctokit(created){
  return {rest:{git:{
    getRef:async({ref})=>{
      if(ref==='heads/main')return {data:{object:{sha:head}}};
      if(ref.startsWith('tags/')&&created.value)
        return {data:{object:{type:'commit',sha:head}}};
      const e=new Error('not found');e.status=404;throw e;},
    getCommit:async({commit_sha:sha})=>
      ({data:{sha,tree:{sha:sha===head?headTree:targetTree}}}),
    createRef:async({ref,sha})=>{
      if(ref!=='refs/tags/v2.0.0'||sha!==head)throw Error('BAD_TAG_CREATE');
      created.value=true;return {data:{ref,object:{sha}}};}
  }}};
}
test('tag stays fail-closed without a pinned tag name or CAS path',async()=>{
  const noName=mock({env:{...env,TALLY_RELEASE_TAG_SOURCE_SHA:head,
    TALLY_RELEASE_TAG_CAS_PATH:'C:/host/tag.sqlite'}});
  assert.equal((await noName.host.tag(tagInput)).reason,'TAG_NAME_NOT_PINNED');
  const noCas=mock({env:{...env,TALLY_RELEASE_TAG_SOURCE_SHA:head,
    TALLY_RELEASE_TAG_NAME:'v2.0.0'}});
  const bound=await noCas.host.tag(tagInput);
  assert.equal(bound.state,'NOT_ADMITTED');
  assert.equal(bound.reason,'TAG_EFFECT_REQUIRES_HOST_CAS_AND_RELEASE_PUBLICATION_BINDING');
  for(const x of [noName,noCas])assert.equal(x.stored.size,0);
});
test('qualified tag creates once, durably, with readback; replay is idempotent',async()=>{
  const created={value:false};
  const ok=mock({env:tagEnv,override:{octokit:tagOctokit(created)}});
  const first=await ok.host.tag(tagInput);
  assert.equal(first.state,'TAG_CREATED_RELEASE_NOT_PUBLISHED');
  assert.equal(first.reason,'RELEASE_OBJECT_AND_USER_ACCEPTANCE_SEPARATE');
  assert.equal(first.tag,'v2.0.0');
  assert.equal(first.githubEffect,'PERFORMED_READBACK_VERIFIED');
  assert.equal(first.releasePass,false);
  assert.equal(first.authorized,false);
  const row=ok.stored.get('tally:v2:release-tag-op:'+
    createHash('sha256')
      .update(JSON.stringify(['v2.0.0',head,headTree])).digest('hex'));
  assert.equal(row.state,'TAG_CREATED');
  assert.equal(row.tag,'v2.0.0');
  assert.equal(created.value,true);
  // Replay: terminal row + verified readback → idempotent OBSERVED, no
  // second createRef (created.value already true; a second create would
  // throw because createRef is not called again — assert via counter).
  let creates=0;
  const counting={rest:{git:{
    getRef:async({ref})=>{
      if(ref==='heads/main')return {data:{object:{sha:head}}};
      if(ref.startsWith('tags/'))return {data:{object:{type:'commit',sha:head}}};
      const e=new Error('nf');e.status=404;throw e;},
    getCommit:async({commit_sha:sha})=>
      ({data:{sha,tree:{sha:sha===head?headTree:targetTree}}}),
    createRef:async()=>{creates++;throw Error('MUST_NOT_CREATE_AGAIN');}
  }}};
  const replay=mock({env:tagEnv,override:{octokit:counting}});
  // seed the terminal row into the replay store via the first host
  const seeded=mock({env:tagEnv,override:{octokit:tagOctokit({value:false})}});
  await seeded.host.tag(tagInput);
  replay.stored.clear();
  for(const [k,v] of seeded.stored)replay.stored.set(k,v);
  const again=await replay.host.tag(tagInput);
  assert.equal(again.state,'TAG_CREATED_RELEASE_NOT_PUBLISHED');
  assert.equal(again.reason,'TAG_IDEMPOTENT_REPLAY_READBACK_VERIFIED');
  assert.equal(again.reconciled,true);
  assert.equal(creates,0,'idempotent replay must not call createRef');
});
test('pending rows and uncertain creates reconcile, never blind-reexecute',async()=>{
  const pending=mock({env:tagEnv,override:{octokit:tagOctokit({value:false})}});
  const key='tally:v2:release-tag-op:'+
    createHash('sha256')
      .update(JSON.stringify(['v2.0.0',head,headTree])).digest('hex');
  pending.stored.set(key,{state:'PENDING_TAG_CREATE',tag:'v2.0.0'});
  const r1=await pending.host.tag(tagInput);
  assert.equal(r1.state,'RECONCILIATION_REQUIRED');
  assert.equal(r1.reason,'PREVIOUS_TAG_OPERATION_PENDING');
  // terminal row but the tag does NOT point at the commit → reconcile
  const drifted=mock({env:tagEnv,override:{octokit:{rest:{git:{
    getRef:async({ref})=>{
      if(ref==='heads/main')return {data:{object:{sha:head}}};
      const e=new Error('nf');e.status=404;throw e;},
    getCommit:async({commit_sha:sha})=>
      ({data:{sha,tree:{sha:sha===head?headTree:targetTree}}}),
    createRef:async()=>{throw Error('MUST_NOT_CREATE');}
  }}}}});
  drifted.stored.set(key,{state:'TAG_CREATED',tag:'v2.0.0'});
  const r2=await drifted.host.tag(tagInput);
  assert.equal(r2.state,'RECONCILIATION_REQUIRED');
});
