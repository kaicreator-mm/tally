import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {
  createHostNativeGitHubClaimReadbackPortFromClient,
  createGitHubHostClaimReadbackPort
} from '../src/task-claim-host-readback-port.mjs';

const owner='kaicreator-mm',repo='tally',sha='a'.repeat(40);
const issue='https://github.com/kaicreator-mm/tally/issues/125';
const apiIssue='https://api.github.com/repos/kaicreator-mm/tally/issues/125';
const reservation={
  repository:'kaicreator-mm/tally',issueNumber:125,issueRef:issue,
  sourceSha:sha,actorRole:'builder',operatorKind:'codex',
  operatorId:'codex:builder1',dispatchId:'dispatch:125:one',
  protectedClaimKey:'kaicreator-mm/tally#125:builder:__default__',
  generation:1
};
const data={
  schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'builder',
  operator_kind:'codex',operator_id:'codex:builder1',
  task:'#125',issue:'#125',sha,
  dispatch_id:'dispatch:125:one',dispatch_state:'CLAIMED',
  protected_claim_key:reservation.protectedClaimKey,admission_generation:1
};
const fence=String.fromCharCode(96).repeat(3);
const body='<!-- ads:github -->\n'+fence+'yaml\n'+
  stringify({schema:'ads/github/v3',kind:'event',data}).trimEnd()+
  '\n'+fence+'\n';
function comment(overrides={}){
  return {
    id:901,body,html_url:issue+'#issuecomment-901',issue_url:apiIssue,
    created_at:'2026-10-28T10:00:00Z',
    updated_at:'2026-10-28T10:00:00Z',user:{login:'tally-host[bot]'},
    ...overrides
  };
}
function github({comments=[comment()],detail=comment(),shaValue=sha,
  drift=false,denied=false}={}){
  let reads=0;
  const octokit={rest:{
    issues:{
      async get(){reads++;return {data:{
        id:525,number:125,html_url:issue,
        updated_at:drift&&reads===2?'2026-10-28T11:00:00Z':
          '2026-10-28T10:00:00Z'
      }};},
      async listComments(){throw new Error('PAGINATION_REQUIRED');},
      async getComment(){if(denied)throw Object.assign(
        new Error('PRIVATE_SECRET'),{status:403});
        return {data:detail};}
    },
    repos:{async getCommit(){return {data:{sha:shaValue}};}}
  },paginate:{async *iterator(){yield {data:comments};}}};
  return octokit;
}
const port=o=>createHostNativeGitHubClaimReadbackPortFromClient({
  octokit:github(o),owner,repo,writerLogin:'tally-host[bot]'
});
test('native exact ADS GitHub comment verifies Host readback DTO, never Task permit',async()=>{
  const x=await port().readPublishedClaim(reservation,901);
  assert.equal(x.sourceKind,'NATIVE_GITHUB_EXACT_READBACK');
  assert.equal(x.complete,true);
  assert.equal(x.edited,false);
  assert.equal(x.commentRef,issue+'#issuecomment-901');
  assert.equal(x.dispatchId,reservation.dispatchId);
  assert.equal(x.generation,1);
  for(const key of ['claimAccepted','authorized','admitted','taskPass','releasePass'])
    assert.equal(x[key],false);
});
test('forged reservation, wrong generation or other Issue fail before source grant',async()=>{
  const p=port();
  for(const bad of [
    {repository:'evil/repo'}, {issueNumber:126},
    {issueRef:'https://github.com/evil/other/issues/125'},
    {sourceSha:'b'.repeat(40)}, {dispatchId:'different'},
    {operatorId:'other'}, {actorRole:'reviewer'},
    {generation:2}, {protectedClaimKey:'forged'}
  ]){
    await assert.rejects(()=>p.readPublishedClaim({...reservation,...bad},901),
      /NATIVE_GITHUB_EXACT_CLAIM_READBACK_UNVERIFIED/);
  }
});
test('edited/missing/foreign, duplicate, stale SHA and 403 all deny exact Host readback',async()=>{
  for(const opts of [
    {detail:comment({body:body+'changed'})},
    {detail:comment({updated_at:'2026-10-28T10:00:01Z'})},
    {detail:comment({user:{login:'attacker'}})},
    {detail:comment({issue_url:apiIssue+'-foreign'})},
    {detail:comment({id:902})},
    {comments:[]},
    {comments:[comment(),comment({id:902,html_url:issue+'#issuecomment-902'})]},
    {shaValue:'b'.repeat(40)},
    {drift:true},{denied:true}
  ]){
    await assert.rejects(()=>port(opts).readPublishedClaim(reservation,901),
      /NATIVE_GITHUB_EXACT_CLAIM_READBACK_UNVERIFIED/);
  }
});
test('Host credential or adapter identity cannot be built by untrusted caller',()=>{
  assert.throws(()=>createGitHubHostClaimReadbackPort({
    owner,repo,writerLogin:'tally-host[bot]',token:''
  }),/TRUSTED_HOST_GITHUB_READBACK_PORT_REQUIRED/);
  assert.throws(()=>createHostNativeGitHubClaimReadbackPortFromClient({
    octokit:github(),owner,repo,writerLogin:'invalid login'
  }),/TRUSTED_HOST_GITHUB_READBACK_PORT_REQUIRED/);
});
