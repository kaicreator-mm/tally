import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {resolveClaimBuildTarget,sealClaimShimTarget} from './claim-target.mjs';

const selected={T06L_TARGET_REPOSITORY:'kaicreator-mm/tally-reps-test',
  T06L_TARGET_ISSUE_NUMBER:'123'};
test('legacy demo target unchanged only when no selected source supplied',()=>{
  const t=resolveClaimBuildTarget({});
  assert.equal(t.mode,'HISTORICAL_DEFAULT');
  assert.equal(t.repository,'kaicreator-mm/tally');
  assert.equal(t.issueNumber,169);
  assert.equal(t.protectedClaimKey,'kaicreator-mm/tally#169:builder:__default__');
});
test('new project and Issue produce one immutable sealed Claim key',()=>{
  const t=resolveClaimBuildTarget(selected);
  assert.equal(t.mode,'SELECTED_BUILD');
  assert.equal(t.issueRef,'https://github.com/kaicreator-mm/tally-reps-test/issues/123');
  assert.equal(t.protectedClaimKey,
    'kaicreator-mm/tally-reps-test#123:builder:__default__');
  assert.equal(t.owner,'kaicreator-mm');
  assert.equal(t.repo,'tally-reps-test');
});
test('partial/injection/invalid selected target refuses before BUILD',()=>{
  for(const bad of [
    {T06L_TARGET_REPOSITORY:selected.T06L_TARGET_REPOSITORY},
    {T06L_TARGET_ISSUE_NUMBER:'123'},
    {...selected,T06L_TARGET_REPOSITORY:'org/repo/extra'},
    {...selected,T06L_TARGET_REPOSITORY:'org/<script>'},
    {...selected,T06L_TARGET_ISSUE_NUMBER:'0'},
    {...selected,T06L_TARGET_ISSUE_NUMBER:'1.0'},
    {...selected,T06L_TARGET_ISSUE_NUMBER:'123\\n'},
  ])assert.throws(()=>resolveClaimBuildTarget(bad),TypeError);
});
test('source-pinned BUILD C rechecks exact Issue, SHA and CAS Claim key',async()=>{
  const src=await readFile(new URL('./claim-admission-shim.mjs',import.meta.url),'utf8');
  const target=resolveClaimBuildTarget(selected);
  const bound=sealClaimShimTarget(src,target);
  assert.ok(bound.includes("const ISSUE='"+target.issueRef+"'"));
  assert.ok(bound.includes("const CLAIM_KEY='"+target.protectedClaimKey+"'"));
  assert.doesNotMatch(bound,/__T06L_TARGET_/);
  const sha='a'.repeat(40),commentId=17;
  const output={state:'ADMITTED_PENDING_AGENT_EFFECT',
    reason:'CLAIM_REGISTRY_PUBLICATION_AND_NATIVE_READBACK_VERIFIED',
    issueRef:target.issueRef,sourceSha:sha,
    protectedClaimKey:target.protectedClaimKey,generation:1,
    dispatchId:'dispatch-123',actorRole:'builder',
    operatorKind:'codex',operatorId:'builder-1',commentId,
    commentRef:target.issueRef+'#issuecomment-'+commentId,
    claimPublication:'REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK',
    claimAccepted:true,admitted:true,authorized:false,
    taskPass:false,agentLaunch:false,releasePass:false};
  const code=bound.replace(/^import[\s\S]*?from '[^']+';\s*/,'')
    .replace(/^export /gm,'');
  const admit=new Function('nativeAdmitClaimedTask',code+
    '\nreturn admitClaimedTask;')(async()=>output);
  const valid=await admit({sourceRef:target.issueRef,issueNumber:123,
    sourceSha:sha});
  assert.equal(valid.state,'ADMITTED_PENDING_AGENT_EFFECT');
  assert.equal(valid.admitted,true);
  assert.equal(valid.authorized,false);
  assert.equal(valid.agentLaunch,false);
  const wrongIssue=await admit({sourceRef:target.issueRef.replace('/123','/124'),
    issueNumber:124,sourceSha:sha});
  assert.equal(wrongIssue.state,'NOT_ADMITTED');
  const wrongSha=await admit({sourceRef:target.issueRef,issueNumber:123,
    sourceSha:'b'.repeat(40)});
  assert.equal(wrongSha.state,'NOT_ADMITTED');
  const fakeProvider=new Function('nativeAdmitClaimedTask',code+
    '\nreturn admitClaimedTask;')(async()=>({
      ...output,protectedClaimKey:'kaicreator-mm/tally#169:builder:__default__'}));
  const foreign=await fakeProvider({sourceRef:target.issueRef,
    issueNumber:123,sourceSha:sha});
  assert.equal(foreign.state,'NOT_ADMITTED');
  assert.equal(foreign.admitted,false);
});
