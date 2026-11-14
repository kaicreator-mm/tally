import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {readNativeHostClaimPreflightFromClient,
  readGitHubNativeHostClaimPreflight} from '../src/task-claim-host-preflight.mjs';
const owner='kaicreator-mm',repo='tally',issueNumber=125,sha='a'.repeat(40);
const issue='https://github.com/kaicreator-mm/tally/issues/125';
const input={owner,repo,issueNumber,expectedSourceSha:sha};
const fence=String.fromCharCode(96).repeat(3);
const event={
  schema:'ads/github/v3',kind:'event',data:{
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',task:'#125',sha,
    actor_role:'builder',operator_kind:'codex',operator_id:'codex-1',
    dispatch_id:'dispatch:125:x',dispatch_state:'CLAIMED'}
};
const claim={id:800,body:'<!-- ads:github -->\n'+fence+'yaml\n'+
  stringify(event).trimEnd()+'\n'+fence,
  html_url:issue+'#issuecomment-800',
  created_at:'2026-10-28T10:00:00Z',
  updated_at:'2026-10-28T10:00:00Z',user:{login:'agent'}};
function client({state='state:ready',type='type:task',issueState='open',
  blocker=null,comments=[],drift=false,failComments=false,commitSha=sha}={}){
  let reads=0;
  return {rest:{
    issues:{
      async get(){reads++;return {data:{
        id:525,number:125,html_url:issue,
        state:issueState,
        updated_at:drift&&reads>=6?'2026-10-28T10:01:00Z':'2026-10-28T10:00:00Z',
        labels:[{name:type},{name:state}]
      }};},
      async listComments(){throw Error('pagination required');}
    },
    repos:{async getCommit(){return {data:{sha:commitSha}};}}
  },paginate:{async *iterator(route){
    if(typeof route==='string'){
      yield {data:blocker?[{id:99,number:99,
        html_url:'https://github.com/kaicreator-mm/tally/issues/99',state:blocker}]:[]};
    }else{
      if(failComments)throw Object.assign(Error('PRIVATE-TOKEN'),{status:403});
      yield {data:comments};
    }
  }}};
}
const check=(c={},overrides={})=>readNativeHostClaimPreflightFromClient({
  ...input,...overrides,octokit:client(c)
});
const noPermit=o=>{
  for(const k of ['authorized','admitted','taskPass','releasePass'])
    assert.equal(o[k],false);
  assert.equal(o.claimStatus,'NOT_VERIFIED');
};
test('native ready/CLEAR/no-Claim is a real Host source witness, NOT writer/K permit',async()=>{
  const x=await check();
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.sourceQualified,true);
  assert.equal(x.sourceKind,'NATIVE_GITHUB_READ_ONLY_PREFLIGHT');
  assert.equal(x.nativePreflightReady,true);
  assert.equal(x.dispatchSourceBound,false);
  assert.equal(x.publisherSourceBound,false);
  noPermit(x);
});
test('blocked dependency, stale state, non-task and closed Issue all deny',async()=>{
  for(const input of [
    {blocker:'open'}, {state:'state:implementing'},
    {type:'type:planning'}, {issueState:'closed'}
  ]){
    const x=await check(input);
    assert.equal(x.nativePreflightReady,false);noPermit(x);
  }
});
test('even shape-valid self-reported DISPATCH_CLAIMED denies clean preflight',async()=>{
  const x=await check({comments:[claim]});
  assert.equal(x.reason,'NATIVE_CLAIM_ABSENCE_UNQUALIFIED');noPermit(x);
});
test('GitHub stale SHA, drift, private 403 and spoofed booleans deny',async()=>{
  for(const facts of [
    {commitSha:'b'.repeat(40)},{drift:true},{failComments:true}
  ]){
    const x=await check(facts);
    assert.equal(x.reason,'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');
    assert.equal(JSON.stringify(x).includes('PRIVATE-TOKEN'),false);noPermit(x);
  }
  const injected=await check({},{
    claimAccepted:true,sourceQualified:true,authorized:true,
    publisherSourceBound:true,dispatchSourceBound:true,
    dependenciesComplete:true
  });
  assert.equal(injected.authorized,false);
  assert.equal(injected.publisherSourceBound,false);
});
test('missing read token, unsupported source SHA and limit fail closed',async()=>{
  const token=await readGitHubNativeHostClaimPreflight({...input,token:null});
  assert.equal(token.reason,'GITHUB_PROVIDER_TOKEN_UNAVAILABLE');noPermit(token);
  assert.equal((await check({}, {limit:1001})).reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');
  assert.equal((await check({}, {expectedSourceSha:'bad'})).reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');
});

// ---- #412 dual-role admission regression (FROZEN_DAG_WRITESET 9f31a645...) ----
const keyFor=(role)=>`kaicreator-mm/tally#${issueNumber}:${role}:__default__`;
const claimEvent=(changes={})=>({
  schema:'ads/github/v3',kind:'event',data:{
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',task:`#${issueNumber}`,
    issue:`#${issueNumber}`,sha,dispatch_state:'CLAIMED',
    admission_generation:1,...changes}
});
const claimComment=(id,changes={})=>({id,
  body:'<!-- ads:github -->\n'+fence+'yaml\n'+stringify(claimEvent(changes.data??{})).trimEnd()+'\n'+fence,
  html_url:issue+'#issuecomment-'+id,
  created_at:'2026-11-14T18:09:38Z',updated_at:'2026-11-14T18:09:38Z',
  user:{login:'kaicreator-mm'},...changes});
const builderClaim=claimComment(801,{data:{
  actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
  dispatch_id:'dispatch:125:builder',protected_claim_key:keyFor('builder')}});
const reviewerClaim=claimComment(802,{data:{
  actor_role:'reviewer',operator_kind:'claude-code',operator_id:'claude:reviewer1',
  dispatch_id:'dispatch:125:reviewer',protected_claim_key:keyFor('reviewer')}});

test('invalid declared actorRole fails request validation',async()=>{
  const x=await check({},{actorRole:'leader'});
  assert.equal(x.reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');noPermit(x);
});

test('fresh Task with declared role and zero candidates stays READY',async()=>{
  const x=await check({},{actorRole:'builder'});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.claimCandidates,0);
  assert.deepEqual(x.claimCandidateRoles,[]);
  assert.equal(x.claimRoleCandidates.length,0);
  noPermit(x);
});

test('existing builder Claim admits a distinct-role Reviewer with bounded projections',async()=>{
  const x=await check({comments:[builderClaim]},{actorRole:'reviewer'});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.nativePreflightReady,true);
  assert.equal(x.claimCandidates,1);
  assert.deepEqual(x.claimCandidateRoles,['builder']);
  assert.equal(x.claimRoleCandidates.length,1);
  const p=x.claimRoleCandidates[0];
  assert.equal(p.commentId,801);
  assert.equal(p.dispatchId,'dispatch:125:builder');
  assert.equal(p.actorRole,'builder');
  assert.equal(p.operatorId,'codex:builder1');
  assert.equal(p.protectedClaimKey,keyFor('builder'));
  assert.equal(p.claimGeneration,1);
  assert.equal(p.sourceSha,sha);
  assert.equal(p.transportActor,'kaicreator-mm');
  assert.equal(x.claimStatus,'NOT_VERIFIED');noPermit(x);
});

test('existing Reviewer Claim admits a distinct-role Builder (symmetric)',async()=>{
  const x=await check({comments:[reviewerClaim]},{actorRole:'builder'});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.deepEqual(x.claimCandidateRoles,['reviewer']);
  noPermit(x);
});

test('same-role duplicate candidate denies the requested role',async()=>{
  const x=await check({comments:[builderClaim]},{actorRole:'builder'});
  assert.equal(x.reason,'NATIVE_CLAIM_ROLE_DUPLICATE_CANDIDATE');noPermit(x);
});

test('two candidates sharing one protectedClaimKey deny as claim overwrite',async()=>{
  const second=claimComment(803,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder2',
    dispatch_id:'dispatch:125:builder2',protected_claim_key:keyFor('builder')}});
  const x=await check({comments:[builderClaim,second]},{actorRole:'reviewer'});
  assert.equal(x.reason,'NATIVE_CLAIM_ROLE_DUPLICATE_CANDIDATE');noPermit(x);
});

test('candidates without canonical key or generation 1 deny unqualified',async()=>{
  const noKey=claimComment(804,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
    dispatch_id:'dispatch:125:nokey'}});
  const x=await check({comments:[noKey]},{actorRole:'reviewer'});
  assert.equal(x.reason,'NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');noPermit(x);
  const wrongKey=claimComment(805,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
    dispatch_id:'dispatch:125:wrongkey',protected_claim_key:keyFor('reviewer')}});
  const y=await check({comments:[wrongKey]},{actorRole:'reviewer'});
  assert.equal(y.reason,'NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');noPermit(y);
  const gen2=claimComment(806,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
    dispatch_id:'dispatch:125:gen2',protected_claim_key:keyFor('builder'),
    admission_generation:2}});
  const z=await check({comments:[gen2]},{actorRole:'reviewer'});
  assert.equal(z.reason,'NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');noPermit(z);
  const badRole=claimComment(807,{data:{
    actor_role:'leader',operator_kind:'codex',operator_id:'codex:leader',
    dispatch_id:'dispatch:125:leader',protected_claim_key:keyFor('leader')}});
  const w=await check({comments:[badRole]},{actorRole:'reviewer'});
  // A non-ADS role never validates against the pinned event schema, so the
  // canonical comment reader fails closed before role classification runs.
  assert.equal(w.reason,'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');noPermit(w);
});

test('candidates present without a declared role keep the legacy denial reason',async()=>{
  const x=await check({comments:[builderClaim]});
  assert.equal(x.reason,'NATIVE_CLAIM_ABSENCE_UNQUALIFIED');noPermit(x);
});
