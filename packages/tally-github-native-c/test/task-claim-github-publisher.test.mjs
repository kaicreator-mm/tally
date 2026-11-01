import test from 'node:test';
import assert from 'node:assert/strict';
import {createHostBoundNativeGitHubClaimPublisher} from
  '../src/task-claim-github-publisher.mjs';
import {readClaimEventCandidatesFromClient} from
  '../src/task-claim-events.mjs';
const sha='a'.repeat(40),owner='kaicreator-mm',repo='tally';
const issueRef='https://github.com/kaicreator-mm/tally/issues/125';
const apiIssue='https://api.github.com/repos/kaicreator-mm/tally/issues/125';
const reservation={repository:'kaicreator-mm/tally',issueNumber:125,
  issueRef,sourceSha:sha,actorRole:'builder',operatorKind:'codex',
  operatorId:'codex:builder1',dispatchId:'dispatch:125:one',
  protectedClaimKey:'kaicreator-mm/tally#125:builder:__default__',
  generation:1,writerId:'tally-host:exclusive'};
const authority={mode:'SINGLE_WRITER_ADMISSION',
  exclusiveHostControl:true,writerId:'tally-host:exclusive',
  repository:'kaicreator-mm/tally',githubWriterLogin:'tally-host[bot]'};
function fake({wrongLogin=false,edited=false,lostAck=false,
  readbackError=false,wrongIssue=false}={}){
  let posted=0,body='',id=900;
  const native=()=>({id,body:edited?body+'tamper':body,
    html_url:issueRef+'#issuecomment-'+id,
    issue_url:wrongIssue?apiIssue+'bad':apiIssue,
    created_at:'2026-10-28T11:00:00Z',
    updated_at:edited?'2026-10-28T11:01:00Z':'2026-10-28T11:00:00Z',
    user:{login:wrongLogin?'attacker':'tally-host[bot]'}});
  const api={rest:{issues:{
    async createComment(x){
      posted++;body=x.body;
      assert.equal(x.issue_number,125);
      if(lostAck)throw Error('GITHUB_WRITE_SECRET');
      return {data:native()};
    },
    async getComment(){if(readbackError)throw Error('PRIVATE_TOKEN');
      return {data:native()};},
    async get(){return {data:{id:525,number:125,html_url:issueRef,
      updated_at:'2026-10-28T11:00:00Z'}};},
    async listComments(){throw Error('pagination required');}
  },repos:{async getCommit(){return {data:{sha}};}}},
  paginate:{async *iterator(){yield {data:[native()]};}}};
  return {api,count:()=>posted};
}
test('one Host transport publish makes real ADS event wire and exact native GET; never admits Task',async()=>{
  const f=fake(),publisher=createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:authority});
  const ack=await publisher.publishClaim(reservation);
  assert.equal(f.count(),1);
  assert.equal(ack.commentId,900);
  assert.match(ack.commentBodySha256,/^[a-f0-9]{64}$/);
  for(const k of ['claimAccepted','authorized','admitted','taskPass','releasePass'])
    assert.equal(ack[k],false);
  const observed=await readClaimEventCandidatesFromClient({
    octokit:f.api,owner,repo,issueNumber:125,expectedSourceSha:sha
  });
  assert.equal(observed.reason,'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED');
  assert.equal(observed.candidates[0].dispatchId,reservation.dispatchId);
  assert.equal(observed.candidates[0].protectedClaimKey,reservation.protectedClaimKey);
  assert.equal(observed.candidates[0].claimGeneration,1);
  assert.equal(observed.candidates[0].claimAccepted,false);
});
test('no credential or trusted Host authority -> zero posts',()=>{
  const f=fake();
  for(const unsafe of [
    null,{...authority,exclusiveHostControl:false},
    {...authority,mode:'UNTRUSTED'},
    {...authority,githubWriterLogin:'x y'},
    {...authority,repository:'evil/repo'}
  ])assert.throws(()=>createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:unsafe}),
    /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
  assert.equal(f.count(),0);
});
test('wrong Task/role/source/gen/writer/claim key cannot send any GitHub POST',async()=>{
  const f=fake(),p=createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:authority});
  for(const invalid of [
    {repository:'evil/repo'},{issueNumber:126},{issueRef:'https://evil'},
    {sourceSha:'bad'},{actorRole:'controller'},
    {operatorKind:'untrusted'},{operatorId:'private key\n'},
    {dispatchId:'malicious\nmessage'},{generation:2},
    {protectedClaimKey:'other'},{writerId:'agent-as-host'}
  ])await assert.rejects(()=>p.publishClaim({...reservation,...invalid}),
    /HOST_CLAIM_RESERVATION_NOT_BOUND/);
  assert.equal(f.count(),0);
});
test('invalid GitHub identity/readback or lost ACK remain ambiguous; no internal retry',async()=>{
  for(const opts of [
    {lostAck:true},{wrongLogin:true},{edited:true},
    {wrongIssue:true},{readbackError:true}
  ]){
    const f=fake(opts),p=createHostBoundNativeGitHubClaimPublisher({
      octokit:f.api,owner,repo,writerAuthority:authority});
    await assert.rejects(()=>p.publishClaim(reservation),
      /NATIVE_GITHUB_CLAIM_PUBLICATION_AMBIGUOUS/);
    assert.equal(f.count(),1);
  }
});
