import test from 'node:test';
import assert from 'node:assert/strict';
import {readReleaseCandidateFromClient,mergeReleaseCandidateFromClient,
  readReleaseTagFromClient,createQualifiedTagFromClient} from '../src/release-delivery.mjs';
const head='a'.repeat(40),target='b'.repeat(40),headTree='c'.repeat(40),
  targetTree='d'.repeat(40),merged='e'.repeat(40),mergedTree='f'.repeat(40);
const args={pullNumber:222,headSha:head,headTree,targetSha:target,targetTree};
const reply=data=>({data});
function mock(over={}){
  const calls=[];
  const git={getRef:async({ref})=>{calls.push('getRef:'+ref);
    if(ref==='heads/version/v2')return reply({object:{sha:head}});
    if(ref==='heads/main')return reply({object:{sha:over.mainAfterMerge?merged:target}});
    if(ref==='tags/v2.0.0')return reply({object:{sha:merged,type:'commit'}});
    throw {status:404};},
    getCommit:async({commit_sha:sha})=>reply({sha,tree:{sha:sha===head?headTree:
      sha===target?targetTree:mergedTree},parents:sha===merged?[{sha:head},{sha:target}]:[]}),
    createRef:async()=>{calls.push('createTag');return reply({});}};
  const pulls={get:async()=>reply({state:'open',head:{ref:'version/v2',sha:head,
      repo:{full_name:'kaicreator-mm/tally'}},base:{ref:'main',sha:target,
      repo:{full_name:'kaicreator-mm/tally'}},mergeable:true,...over.pr}),
    merge:async()=>{calls.push('merge');over.mainAfterMerge=true;
      return reply({merged:true,sha:merged});}};
  const octokit={rest:{git:{...git,...over.git},pulls:{...pulls,...over.pulls}}};
  return {octokit,calls};
}
const deny=result=>{assert.equal(result.authorized,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.equal(result.releasePublished,false);};
test('exact candidate SHA/tree and live main readback required',async()=>{
  const good=await readReleaseCandidateFromClient({...args,...mock()});
  assert.equal(good.state,'EXACT_PREFLIGHT_OBSERVED');deny(good);
  for(const bad of [{headSha:'not-a-sha'},{targetSha:head},{headTree:targetTree},
    {targetTree:headTree},{pullNumber:0}]){
    const result=await readReleaseCandidateFromClient({...args,...bad,...mock()});
    assert.equal(result.state,'NOT_ADMITTED');deny(result);
  }
});
test('wrong repository, stale PR base or mergeability missing denies',async()=>{
  for(const pr of [{head:{ref:'version/v2',sha:head,repo:{full_name:'evil/repo'}}},
    {base:{ref:'main',sha:merged,repo:{full_name:'kaicreator-mm/tally'}}},
    {mergeable:null},{state:'closed'},{mergeable:false}]){
    const result=await readReleaseCandidateFromClient({...args,...mock({pr})});
    assert.equal(result.state,'NOT_ADMITTED');deny(result);
  }
});
test('one qualified transport merge is read back but NEVER yields Release PASS',async()=>{
  const ctx=mock();const result=await mergeReleaseCandidateFromClient({...args,...ctx});
  assert.equal(result.state,'MERGED_REQUALIFY_REQUIRED');
  assert.equal(result.mergedSha,merged);assert.equal(result.mergedTree,mergedTree);
  assert.equal(result.canTag,false);assert.equal(result.canRelease,false);deny(result);
  assert.deepEqual(ctx.calls.filter(v=>v==='merge'),['merge']);
});
test('stale target, merge 403, lost ACK, and changed readback cannot be promoted',async()=>{
  const stale=mock({git:{getRef:async({ref})=>reply({object:{sha:ref==='heads/main'?head:head}})}});
  const s=await mergeReleaseCandidateFromClient({...args,...stale});
  assert.equal(s.state,'NOT_ADMITTED');assert.equal(stale.calls.includes('merge'),false);
  const forbidden=mock({pulls:{merge:async()=>{throw {status:403};}}});
  const f=await mergeReleaseCandidateFromClient({...args,...forbidden});
  assert.equal(f.state,'GITHUB_EFFECT_UNKNOWN');assert.equal(f.reason,'GITHUB_ACCESS_DENIED');deny(f);
  const lost=mock({pulls:{merge:async()=>{throw new Error('read ECONNRESET');}}});
  assert.equal((await mergeReleaseCandidateFromClient({...args,...lost})).state,'GITHUB_EFFECT_UNKNOWN');
  const mismatch=mock({pulls:{merge:async()=>reply({merged:true,sha:merged})}});
  assert.equal((await mergeReleaseCandidateFromClient({...args,...mismatch})).state,'GITHUB_EFFECT_UNKNOWN');
});
test('tag requires exact post-merge main tree and canonical tag readback',async()=>{
  const ctx=mock({mainAfterMerge:true});
  const done=await createQualifiedTagFromClient({octokit:ctx.octokit,tag:'v2.0.0',
    commitSha:merged,commitTree:mergedTree});
  assert.equal(done.state,'TAG_CREATED_RELEASE_NOT_PUBLISHED');deny(done);
  assert.equal(ctx.calls.filter(x=>x==='createTag').length,1);
  const unsafe=mock();
  const bad=await createQualifiedTagFromClient({octokit:unsafe.octokit,tag:'v2.0.0',
    commitSha:merged,commitTree:mergedTree});
  assert.equal(bad.state,'NOT_ADMITTED');
  assert.equal(unsafe.calls.includes('createTag'),false);
  for(const tag of ['v2/evil','../v2.0.0','v2.0.0?x']){
    const r=await createQualifiedTagFromClient({octokit:ctx.octokit,tag,commitSha:merged,
      commitTree:mergedTree});assert.equal(r.state,'NOT_ADMITTED');
  }
});
test('lost tag acknowledgment remains UNKNOWN, never blind retry',async()=>{
  const ctx=mock({mainAfterMerge:true,git:{createRef:async()=>{throw {status:422};}}});
  const res=await createQualifiedTagFromClient({octokit:ctx.octokit,tag:'v2.0.0',
    commitSha:merged,commitTree:mergedTree});
  assert.equal(res.state,'GITHUB_EFFECT_UNKNOWN');deny(res);
  const read=await readReleaseTagFromClient({octokit:ctx.octokit,tag:'v2.0.0',expectedSha:merged});
  assert.equal(read.state,'TAG_READBACK_VERIFIED');deny(read);
});
