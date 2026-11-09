import test from 'node:test';
import assert from 'node:assert/strict';
import {createHostBoundNativeGitHubClaimPublisher} from
  '../src/task-claim-github-publisher.mjs';
import {readClaimEventCandidatesFromClient} from
  '../src/task-claim-events.mjs';
import {TARGET} from '../src/task-dispatch-state-events.mjs';
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
// Test-only stand-in for the Host-injected experiments-side fence port
// (packages cannot import experiments). Shape mirrors host-resource-fence
// createHostResourceFencePort; readHeldFence returns its readback contract.
const FROZEN_DIGEST='c'.repeat(64);
function fencePort({proofChanges={},onRead=()=>{}}={}){
  return {
    sourceKind:'TRUSTED_HOST_RESOURCE_FENCE_PORT',
    repository:'kaicreator-mm/tally',writerId:authority.writerId,
    frozenResourceSetSha256:FROZEN_DIGEST,
    async readHeldFence(context){
      onRead(context);
      return {
        sourceKind:'TRUSTED_HOST_RESOURCE_FENCE_READBACK',
        complete:true,fenceHeld:true,
        repository:'kaicreator-mm/tally',writerId:authority.writerId,
        protectedClaimKey:context.protectedClaimKey,
        sourceSha:context.sourceSha,
        frozenResourceSetSha256:FROZEN_DIGEST,
        fenceId:'fence-test:one',fenceRevision:1,
        observedAt:new Date().toISOString(),
        ...proofChanges
      };
    }
  };
}
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
  let seenContext=null;
  const f=fake(),publisher=createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:authority,
    fencePort:fencePort({onRead:context=>{seenContext=context;}})});
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
  // The pre-POST fence context is derived from THIS reservation with explicit
  // empty foreign projections and the exact empty-projection digest.
  assert.deepEqual(seenContext,{
    repository:'kaicreator-mm/tally',writerId:authority.writerId,
    protectedClaimKey:reservation.protectedClaimKey,
    sourceSha:reservation.sourceSha,
    foreignClaimCount:0,foreignClaims:[],
    foreignClaimsSha256:'4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945'
  });
});
test('no credential, trusted Host authority or Host fence port -> zero posts',()=>{
  const f=fake();
  for(const unsafe of [
    null,{...authority,exclusiveHostControl:false},
    {...authority,mode:'UNTRUSTED'},
    {...authority,githubWriterLogin:'x y'},
    {...authority,repository:'evil/repo'}
  ])assert.throws(()=>createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:unsafe,fencePort:fencePort()}),
    /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
  assert.throws(()=>createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:authority}),
    /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
  for(const badPort of [
    null,
    {...fencePort(),sourceKind:'UNTRUSTED_CALLER_WITNESS'},
    {...fencePort(),repository:'evil/repo'},
    {...fencePort(),writerId:'agent-as-host'},
    {...fencePort(),frozenResourceSetSha256:'not-hex'},
    {...fencePort(),readHeldFence:null}
  ])assert.throws(()=>createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:authority,fencePort:badPort}),
    /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
  assert.equal(f.count(),0);
});
test('fence proof failures refuse before any POST: no lease, stale or foreign-bound proof',async()=>{
  const notHeld={...fencePort(),async readHeldFence(){
    throw Object.assign(Error('HOST_RESOURCE_FENCE_NOT_HELD'),
      {code:'HOST_RESOURCE_FENCE_NOT_HELD'});
  }};
  for(const port of [
    notHeld,
    fencePort({proofChanges:{observedAt:new Date(Date.now()-60000).toISOString()}}),
    fencePort({proofChanges:{repository:'other/repo'}}),
    fencePort({proofChanges:{writerId:'other-host'}}),
    fencePort({proofChanges:{frozenResourceSetSha256:'d'.repeat(64)}}),
    fencePort({proofChanges:{protectedClaimKey:'forged:key'}}),
    fencePort({proofChanges:{sourceSha:'b'.repeat(40)}}),
    fencePort({proofChanges:{complete:false}}),
    fencePort({proofChanges:{fenceHeld:false}}),
    fencePort({proofChanges:{sourceKind:'FORGED_READBACK'}})
  ]){
    const f=fake(),p=createHostBoundNativeGitHubClaimPublisher({
      octokit:f.api,owner,repo,writerAuthority:authority,fencePort:port});
    await assert.rejects(()=>p.publishClaim(reservation),
      error=>error.code==='HOST_RESOURCE_FENCE_NOT_HELD'||
        /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/.test(error.message));
    assert.equal(f.count(),0);
  }
});
test('wrong Task/role/source/gen/writer/claim key cannot send any GitHub POST',async()=>{
  const f=fake(),p=createHostBoundNativeGitHubClaimPublisher({
    octokit:f.api,owner,repo,writerAuthority:authority,fencePort:fencePort()});
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
      octokit:f.api,owner,repo,writerAuthority:authority,fencePort:fencePort()});
    await assert.rejects(()=>p.publishClaim(reservation),
      /NATIVE_GITHUB_CLAIM_PUBLICATION_AMBIGUOUS/);
    assert.equal(f.count(),1);
  }
});

test('terminal publication is #203 scheduler DONE with Host identity and no blind retry',async()=>{
  const terminal={...TARGET,phase:'PENDING_GITHUB_TERMINALIZATION',commentId:TARGET.claimCommentId,
    commentRef:TARGET.claimCommentRef,terminalWriterId:authority.writerId,
    terminalEvidence:'durable proof',terminalOccurredAt:'2026-11-06T00:00:00Z',
    terminalReservationRevision:2,
    operator_id:'forged',transport_actor:'forged'};
  for(const lost of [false,true]){
    let posted=0,current;
    const api={rest:{issues:{async createComment(input){
      posted++;assert.equal(input.issue_number,203);
      assert.match(input.body,/actor_role: scheduler/);assert.match(input.body,/dispatch_state: DONE/);
      assert.match(input.body,/operator_id: tally-host:exclusive/);
      assert.match(input.body,/transport_actor: tally-host\[bot\]/);
      assert.doesNotMatch(input.body,/forged/);
      current={id:901,body:input.body,html_url:TARGET.issueRef+'#issuecomment-901',
        issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/203',
        user:{login:authority.githubWriterLogin},created_at:'now',updated_at:'now'};
      if(lost)throw Error('lost');return {data:current};
    },async getComment(){return {data:current};}}}};
    const p=createHostBoundNativeGitHubClaimPublisher({octokit:api,owner,repo,
      writerAuthority:authority,fencePort:fencePort()});
    for(const change of [{sourceSha:'b'.repeat(40)},{generation:2},{commentId:2},{terminalWriterId:'attacker'}])
      await assert.rejects(()=>p.publishDispatchStateChange({...terminal,...change}),/HOST_TERMINAL_RESERVATION_NOT_BOUND/);
    assert.equal(posted,0);
    if(lost)await assert.rejects(()=>p.publishDispatchStateChange(terminal),/AMBIGUOUS/);
    else assert.equal((await p.publishDispatchStateChange(terminal)).commentId,901);
    assert.equal(posted,1);
  }
});
