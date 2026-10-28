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
