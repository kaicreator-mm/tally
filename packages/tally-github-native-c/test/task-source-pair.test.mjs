import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {readNativeTaskSourcePairFromClient,readGitHubNativeTaskSourcePair}
  from '../src/task-source-pair.mjs';
import {diagnoseNativeTaskReadiness}
  from '../../tally-engineering-business/src/index.mjs';

const sha='a'.repeat(40), owner='kaicreator-mm',repo='tally',issueNumber=125;
const issueRef=`https://github.com/${owner}/${repo}/issues/${issueNumber}`;
const base={owner,repo,issueNumber,expectedSourceSha:sha};
const event={
  schema:'ads/github/v3',kind:'event',
  data:{schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'builder',
    operator_kind:'codex',operator_id:'codex:builder-1',
    session_ref:'session:1',task:'#125',dispatch_id:'dispatch-1',
    dispatch_state:'CLAIMED',sha}
};
const comment=()=>({id:108,body:`<!-- ads:github -->\n\x60\x60\x60yaml\n${stringify(event).trimEnd()}\n\x60\x60\x60`,
  html_url:issueRef+'#issuecomment-108',created_at:'2026-10-28T00:00:00Z',
  updated_at:'2026-10-28T00:00:00Z',user:{login:'github-actions[bot]'}});
function fixture({blockerState=null,state='state:ready',comments=[],drift=false,
  failAtComments=false,wrongSha=false}={}){
  let reads=0;
  const api={
    rest:{
      issues:{
        async get(){
          reads++;
          return {data:{id:525,number:issueNumber,html_url:issueRef,
            updated_at:drift&&reads>=6?'2026-10-28T00:01:00Z':'2026-10-28T00:00:00Z',
            state:'open',labels:[{name:'type:task'},{name:state}]}};
        },
        async listComments(){throw Error('not to be called outside pagination');}
      },
      repos:{async getCommit(){return {data:{sha:wrongSha?'b'.repeat(40):sha}};}}
    },
    paginate:{async *iterator(route){
      if(typeof route==='string'){
        if(blockerState)yield {data:[{id:999,number:99,
          html_url:'https://github.com/kaicreator-mm/tally/issues/99',
          state:blockerState}]};
        else yield {data:[]};
      }else{
        if(failAtComments)throw Object.assign(new Error('SECRET_THROW'),{status:403});
        yield {data:comments};
      }
    }}
  };
  return api;
}
const notGranted=x=>{
  assert.equal(x.admitted,false);
  assert.equal(x.authorized,false);
  assert.equal(x.taskPass,false);
  assert.equal(x.releasePass,false);
  assert.equal(x.claimStatus,'NOT_VERIFIED');
};
test('real shaped ready native task with no Claim is only C advisory, B refuses',async()=>{
  const x=await readNativeTaskSourcePairFromClient({...base,octokit:fixture()});
  assert.equal(x.status,'SOURCE_PAIR_OBSERVED');
  assert.equal(x.workItem.dependencyState,'CLEAR');
  assert.equal(x.claimEvents.reason,'NO_DISPATCH_CLAIM_EVENTS');
  notGranted(x);
  const b=diagnoseNativeTaskReadiness({
    workItem:x.workItem,claimEvents:x.claimEvents,
    expectedIssueRef:issueRef,expectedSourceSha:sha,
    localAgentStatus:'AVAILABLE',builderConfinementStatus:'VERIFIED',
    hiddenVerifierStatus:'VERIFIED'
  });
  assert.equal(b.reason,'ACCEPTED_CAS_CLAIM_NOT_VERIFIED');
  assert.equal(b.taskPass,false);
});
test('canonical observed DISPATCH_CLAIMED remains unaccepted and raw actors not journalled',async()=>{
  const x=await readNativeTaskSourcePairFromClient({
    ...base,octokit:fixture({comments:[comment()]})});
  assert.equal(x.status,'SOURCE_PAIR_OBSERVED');
  assert.equal(x.claimEvents.candidateCount,1);
  assert.equal(x.claimEvents.candidates[0].dispatchId,'dispatch-1');
  assert.equal(x.claimEvents.candidates[0].claimAccepted,false);
  assert.equal(JSON.stringify(x).includes('github-actions[bot]'),false);
  notGranted(x);
  const b=diagnoseNativeTaskReadiness({
    workItem:x.workItem,claimEvents:x.claimEvents,expectedIssueRef:issueRef,
    expectedSourceSha:sha,localAgentStatus:'AVAILABLE',
    builderConfinementStatus:'VERIFIED',hiddenVerifierStatus:'VERIFIED'
  });
  assert.equal(b.reason,'ACCEPTED_CAS_CLAIM_NOT_VERIFIED');
});
test('native blocked-by leaves workfront blocked even when comment says claimed',async()=>{
  const x=await readNativeTaskSourcePairFromClient({
    ...base,octokit:fixture({blockerState:'open',comments:[comment()]})});
  assert.equal(x.workItem.dependencyState,'BLOCKED');
  notGranted(x);
  const b=diagnoseNativeTaskReadiness({
    workItem:x.workItem,claimEvents:x.claimEvents,
    expectedIssueRef:issueRef,expectedSourceSha:sha});
  assert.equal(b.reason,'NATIVE_DEPENDENCIES_BLOCKED');
});
test('cross-reader Issue drift rejects the entire pair',async()=>{
  const x=await readNativeTaskSourcePairFromClient({
    ...base,octokit:fixture({drift:true})});
  assert.equal(x.reason,'NATIVE_ISSUE_DRIFT_ACROSS_PAIR');
  assert.equal(x.status,'UNKNOWN');
  notGranted(x);
});
test('wrong SHA and late 403 never preserve partial usable source pair',async()=>{
  for(const [o,expected] of [
    [{wrongSha:true},'NATIVE_WORK_ITEM_NOT_SOURCE_QUALIFIED'],
    [{failAtComments:true},'NATIVE_CLAIM_EVENTS_NOT_SOURCE_QUALIFIED']
  ]){
    const x=await readNativeTaskSourcePairFromClient({...base,octokit:fixture(o)});
    assert.equal(x.reason,expected);
    assert.equal(x.status,'UNKNOWN');
    assert.equal(JSON.stringify(x).includes('SECRET_THROW'),false);
    notGranted(x);
  }
});
test('client errors, missing secret and spoofed caller flags cannot authorize',async()=>{
  const x=await readNativeTaskSourcePairFromClient({
    ...base,octokit:fixture(),claimAccepted:true,admitted:true,
    dependenciesComplete:true});
  notGranted(x);
  const invalid=await readNativeTaskSourcePairFromClient({
    ...base,octokit:fixture(),expectedSourceSha:'wrong'});
  assert.equal(invalid.reason,'INVALID_NATIVE_SOURCE_PAIR_REQUEST');
  notGranted(invalid);
  const noToken=await readGitHubNativeTaskSourcePair({...base,token:null});
  assert.equal(noToken.reason,'GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  notGranted(noToken);
});
