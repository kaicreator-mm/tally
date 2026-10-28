import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {readClaimPublicationWitnessFromClient,
  readGitHubClaimPublicationWitness} from '../src/task-claim-publication-witness.mjs';
const fence=String.fromCharCode(96).repeat(3);
const owner='kaicreator-mm',repo='tally',issueNumber=125,sha='a'.repeat(40);
const issue='https://github.com/kaicreator-mm/tally/issues/125';
const expected={owner,repo,issueNumber,expectedSourceSha:sha,commentId:901,
  dispatchId:'dispatch:125:t06g:1',actorRole:'builder',operatorKind:'codex',
  operatorId:'codex:builder-1',protectedClaimKey:'kaicreator-mm/tally#125:builder:__default__',
  claimGeneration:1,writerLogin:'tally-host-bot'};
function event(id=expected.dispatchId){
  return {schema:'ads/github/v3',kind:'event',data:{
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',task:'#125',issue:'#125',
    sha,actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder-1',
    dispatch_id:id,dispatch_state:'CLAIMED',protected_claim_key:expected.protectedClaimKey,
    admission_generation:1}};
}
const bodyFor=x=>'<!-- ads:github -->\n'+fence+'yaml\n'+
  stringify(x).trimEnd()+'\n'+fence+'\n';
function comment(id=901,body=bodyFor(event())){
  return {id,body,html_url:issue+'#issuecomment-'+id,
    issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/125',
    created_at:'2026-10-28T09:00:00Z',updated_at:'2026-10-28T09:00:00Z',
    user:{login:'tally-host-bot'}};
}
function client({pages=[[comment()]],detail=comment(),error=null,drift=false}={}){
  let n=0;
  return {rest:{
    issues:{
      async get(){n++;return {data:{id:1250,number:125,html_url:issue,
        updated_at:drift&&n===2?'2026-10-28T10:00:00Z':'2026-10-28T09:00:00Z'}};},
      async listComments(){throw Error('requires pagination');},
      async getComment(){if(error)throw Object.assign(Error('private-secret'),{status:error});
        return {data:detail};}
    },
    repos:{async getCommit(){return {data:{sha}};}}
  },paginate:{async *iterator(){for(const p of pages)yield {data:p};}}};
}
const check=(options={},changes={})=>readClaimPublicationWitnessFromClient({
  ...expected,...changes,octokit:client(options)});
test('exact GitHub witness is observational, never CAS acceptance or Task PASS',async()=>{
  const x=await check();assert.equal(x.status,'PUBLICATION_WITNESSED');
  assert.equal(x.commentRef,issue+'#issuecomment-901');
  assert.equal(x.claimPublication,'NATIVE_COMMENT_READBACK_ONLY');
  for(const k of ['claimAccepted','authorized','admitted','taskPass','releasePass'])
    assert.equal(x[k],false);
});
test('comment changed, cross-issue or wrong publisher is denied',async()=>{
  const variants=[
    {...comment(),updated_at:'2026-10-28T11:00:00Z'},
    {...comment(),body:comment().body+'\nchanged'},
    {...comment(),id:902},
    {...comment(),issue_url:'https://api.github.com/repos/other/x/issues/125'},
    {...comment(),html_url:issue+'#issuecomment-902'},
    {...comment(),user:{login:'attacker'}}
  ];
  for(const detail of variants)assert.equal((await check({detail})).reason,
    'NATIVE_COMMENT_CURRENTNESS_NOT_VERIFIED');
});
test('Host expected identity cannot be replaced by a self-reported event',async()=>{
  for(const changes of [
    {dispatchId:'dispatch:other'},{operatorId:'codex:other'},
    {writerLogin:'attacker'},{operatorKind:'other'},
    {commentId:902},{expectedSourceSha:'b'.repeat(40)},
    {claimGeneration:2},{protectedClaimKey:'x'},
    {actorRole:'reviewer',protectedClaimKey:'kaicreator-mm/tally#125:reviewer:__default__'}
  ])assert.equal((await check({},changes)).claimAccepted,false);
});
test('missing, duplicate or overlapping same-key event cannot witness one claim',async()=>{
  assert.equal((await check({pages:[[{body:'claimed!'}],[]]})).reason,
    'NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED');
  assert.equal((await check({pages:[[comment(),comment(902,bodyFor(event('dispatch:second')))]]})).reason,
    'PROTECTED_CLAIM_KEY_NOT_UNIQUE');
  assert.equal((await check({pages:[[comment(),comment(902)]]})).reason,
    'NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED');
});
test('stale Issue, edited event, denial and absent provider token all fail closed',async()=>{
  assert.equal((await check({drift:true})).reason,'NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED');
  assert.equal((await check({pages:[[{...comment(),updated_at:'changed'}]]})).reason,
    'NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED');
  const denied=await check({error:403});
  assert.equal(denied.reason,'GITHUB_ACCESS_DENIED');
  assert.equal(JSON.stringify(denied).includes('private-secret'),false);
  assert.equal((await readGitHubClaimPublicationWitness({...expected,token:''})).reason,
    'GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
});
