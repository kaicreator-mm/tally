import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {readClaimEventCandidatesFromClient} from '../src/task-claim-events.mjs';
const sha='a'.repeat(40),owner='kaicreator-mm',repo='tally',issueNumber=125;
const issueUrl='https://github.com/kaicreator-mm/tally/issues/125';
const base={owner,repo,issueNumber,expectedSourceSha:sha};
const event=(extra={})=>({
  schema:'ads/github/v3',kind:'event',
  data:{schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'builder',
    operator_kind:'codex',operator_id:'codex:builder-1',
    session_ref:'session:1',task:'#125',dispatch_id:'dispatch-1',
    dispatch_state:'CLAIMED',sha,...extra}
});
const render=x=>`<!-- ads:github -->\n\x60\x60\x60yaml\n${stringify(x).trimEnd()}\n\x60\x60\x60\nHuman explanation only.`;
const comment=(id=100,e=event())=>({
  id,body:render(e),
  html_url:issueUrl+'#issuecomment-'+id,
  created_at:'2026-10-27T00:00:00Z',
  updated_at:'2026-10-27T00:00:00Z',
  user:{login:'github-actions[bot]'}
});
function client(pages=[[]],opts={}){
  let reads=0;
  return {
    rest:{
      issues:{
        async get(){
          const data={id:525,number:125,html_url:issueUrl,
            updated_at:opts.drift&&++reads===2?'2026-10-27T01:00:00Z':'2026-10-27T00:00:00Z'};
          return {data};
        },
        async listComments(){throw Error('must use pagination');}
      },
      repos:{async getCommit(){return {data:{sha:opts.wrongSha?'b'.repeat(40):sha}};}}
    },
    paginate:{async *iterator(_route,params){
      assert.equal(params.issue_number,125);
      for(let i=0;i<pages.length;i++){
        if(opts.errorPage===i)throw Object.assign(new Error('token:UNSAFE'),{status:403});
        yield {data:pages[i]};
      }
    }}
  };
}
test('shape-valid native DISPATCH_CLAIMED still NOT accepted CAS, cannot execute',async()=>{
  const r=await readClaimEventCandidatesFromClient({...base,
    octokit:client([[comment()]])});
  assert.equal(r.reason,'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED');
  assert.equal(r.candidateCount,1);
  assert.equal(r.candidates[0].dispatchId,'dispatch-1');
  assert.equal(r.candidates[0].sourceClass,'OBSERVED_COMMENT_SELF_REPORTED_CLAIM');
  assert.equal(r.candidates[0].claimAccepted,false);
  assert.equal(r.claimStatus,'NOT_VERIFIED');
  assert.equal(r.admitted,false);assert.equal(r.authorized,false);
  assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
});
test('two actual GitHub pages are read; unrelated human comments do not become events',async()=>{
  const r=await readClaimEventCandidatesFromClient({...base,
    octokit:client([[{id:40,body:'state:claimed from a role label'}],
      [comment(101,event({dispatch_id:'dispatch:second'}))]])});
  assert.equal(r.candidateCount,1);
  assert.equal(r.claimStatus,'NOT_VERIFIED');
});
test('no candidate means no Claim',async()=>{
  const r=await readClaimEventCandidatesFromClient({...base,octokit:client([[{body:'hello'}],[]])});
  assert.equal(r.reason,'NO_DISPATCH_CLAIM_EVENTS');
  assert.equal(r.candidateCount,0);
  assert.equal(r.taskPass,false);
});
test('wrong task/issue/source and malformed event-v2 field are rejected',async()=>{
  for(const bad of [
    event({task:'#999'}),
    event({task:'#125',issue:'#999'}),
    event({sha:'b'.repeat(40)}),
    event({dispatch_state:'RUNNING'}),
    event({actor_role:'scheduler'}),
    event({operator_kind:'fake-agent'}),
    event({task:undefined,issue:undefined}),
  ]){
    const r=await readClaimEventCandidatesFromClient({...base,octokit:client([[comment(100,bad)]])});
    assert.equal(r.candidateCount,0);
    assert.equal(r.admitted,false);
    assert.ok(['CROSS_TASK_OR_STALE_CLAIM_EVENT','UNVERIFIED_ADS_EVENT_COMMENT'].includes(r.reason));
  }
});
test('malformed YAML / duplicate key / edited comment rejected rather than ignored',async()=>{
  const malformed={...comment(),body:'<!-- ads:github -->\n\x60\x60\x60yaml\nschema: ads/github/v3\nschema: ads/github/v3\nkind: event\ndata: {}\n\x60\x60\x60'};
  const bads=[malformed,{...comment(),updated_at:'2026-10-28T00:00:00Z'},
    {...comment(),html_url:'https://evil.test'}];
  for(const c of bads){
    const r=await readClaimEventCandidatesFromClient({...base,octokit:client([[c]])});
    assert.equal(r.candidateCount,0);
    assert.ok(['UNVERIFIED_ADS_EVENT_COMMENT','NATIVE_EVENT_IDENTITY_UNVERIFIED'].includes(r.reason));
  }
});
test('duplicate dispatch ID, page limit, late 403 fail closed, never leak error',async()=>{
  const cases=[
    [client([[comment(100)],[comment(101)]]),undefined,'DUPLICATE_DISPATCH_CLAIM_EVENT'],
    [client([[comment(100),comment(101,event({dispatch_id:'different'}))]]),1,'COMMENT_SCAN_LIMIT'],
    [client([[comment(100)],[comment(101,event({dispatch_id:'other'}))]],{errorPage:1}),undefined]
  ];
  for(const [octokit,limit,expected] of cases){
    const r=await readClaimEventCandidatesFromClient({...base,octokit,...(limit?{limit}:{})});
    if(expected)assert.equal(r.reason,expected);
    assert.equal(r.candidateCount,0);
    assert.equal(r.taskPass,false);
  }
  const rejected=await readClaimEventCandidatesFromClient({...base,
    octokit:client([[comment(100)],[comment(101,event({dispatch_id:'other'}))]],{errorPage:1})});
  assert.equal(rejected.reason,'GITHUB_ACCESS_DENIED');
  assert.equal(JSON.stringify(rejected).includes('UNSAFE'),false);
});
test('native issue metadata drift, old SHA and no source token are never accepted',async()=>{
  assert.equal((await readClaimEventCandidatesFromClient({...base,
    octokit:client([[comment()]],{drift:true})})).reason,'NATIVE_ISSUE_CHANGED_DURING_READ');
  assert.equal((await readClaimEventCandidatesFromClient({...base,
    octokit:client([[comment()]],{wrongSha:true})})).reason,'NATIVE_ISSUE_OR_SHA_MISMATCH');
  assert.equal((await readClaimEventCandidatesFromClient({...base,
    octokit:client(),issueNumber:0})).reason,'INVALID_NATIVE_QUERY');
});

// ---- #422 claim-succession regression (FROZEN_DAG_WRITESET b088685b...) ----
// A canonical, unedited DISPATCH_STATE_CHANGED/DONE that exactly binds one
// historical stale candidate (same Task, dispatchId, protectedClaimKey,
// generation, predecessor_claim_comment_id and that candidate's own source
// SHA) filters ONLY that candidate out of fresh admission. Every other DONE
// shape stays fail-closed. The #219 fixtures below are read-only shape
// reproductions; no real GitHub comment or registry is touched.
const histSha='ad3123e7646428ec3b13fac5b6f510b683a133c8';
const curSha='fa49891dcb39c6eb8517f84087f890e8d4d47f93';
const histKey='kaicreator-mm/tally#219:builder:__default__';
const histDispatch='dispatch:219:maintenance:20261031:1';
const histCommentId=6545079271;
const histCreated='2026-10-31T15:01:07Z';
const issue219Url='https://github.com/kaicreator-mm/tally/issues/219';
const histEvent=()=>({
  schema:'ads/github/v3',kind:'event',
  data:{schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'builder',
    operator_kind:'other',operator_id:'zcode:i219:builder:1',task:'#219',
    issue:'#219',sha:histSha,dispatch_id:histDispatch,
    dispatch_state:'CLAIMED',protected_claim_key:histKey,
    admission_generation:1}
});
const doneEvent=(binding={})=>({
  schema:'ads/github/v3',kind:'event',
  data:{schema:'ai-dev/event-v2',event:'DISPATCH_STATE_CHANGED',
    actor_role:'scheduler',operator_kind:'other',
    operator_id:'scheduler:i219:terminalization:1',
    task:'#219',issue:'#219',sha:histSha,
    dispatch_id:histDispatch,dispatch_state:'DONE',
    protected_claim_key:histKey,admission_generation:1,
    predecessor_claim_comment_id:histCommentId,...binding}
});
const comment219=(id,e)=>({
  id,body:render(e),
  html_url:issue219Url+'#issuecomment-'+id,
  created_at:histCreated,updated_at:histCreated,
  user:{login:'kaicreator-mm'}
});
function client219(pages,opts={}){
  let reads=0;
  return {
    rest:{
      issues:{
        async get(){
          const data={id:919,number:219,html_url:issue219Url,
            updated_at:opts.drift&&++reads===2?'2026-11-15T00:00:00Z':histCreated};
          return {data};
        },
        async listComments(){throw Error('must use pagination');}
      },
      repos:{async getCommit(){return {data:{sha:opts.wrongSha?'c'.repeat(40):curSha}};}}
    },
    paginate:{async *iterator(_route,params){
      assert.equal(params.issue_number,219);
      for(const page of pages)yield {data:page};
    }}
  };
}
const histClaimComment=comment219(histCommentId,histEvent());
const exactDoneComment=comment219(6894000001,doneEvent());

test('exact unedited DONE filters only its terminalized historical stale Claim',async()=>{
  const r=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,exactDoneComment]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(r.reason,'NO_DISPATCH_CLAIM_EVENTS');
  assert.equal(r.candidateCount,0);
  assert.equal(r.candidates.length,0);
  assert.equal(r.terminalizedHistoricalClaimCount,1);
  const t=r.terminalizedHistoricalClaims[0];
  assert.equal(t.commentId,histCommentId);
  assert.equal(t.dispatchId,histDispatch);
  assert.equal(t.protectedClaimKey,histKey);
  assert.equal(t.claimGeneration,1);
  assert.equal(t.sourceSha,histSha);
  assert.equal(t.terminalizedByDoneCommentId,6894000001);
  assert.equal(t.actorRole,'builder');
  assert.equal(t.claimAccepted,false);
  assert.equal(r.admitted,false);assert.equal(r.authorized,false);
  assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
});

test('terminal filtering keeps other active candidates observable',async()=>{
  const active=comment219(6894000003,{schema:'ads/github/v3',kind:'event',
    data:{schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'reviewer',
      operator_kind:'other',operator_id:'zcode:i219:reviewer:1',task:'#219',
      issue:'#219',sha:curSha,dispatch_id:'dispatch:219:successor:1',
      dispatch_state:'CLAIMED',
      protected_claim_key:'kaicreator-mm/tally#219:reviewer:__default__',
      admission_generation:1}});
  const r=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,exactDoneComment,active]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(r.reason,'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED');
  assert.equal(r.candidateCount,1);
  assert.equal(r.candidates[0].actorRole,'reviewer');
  assert.equal(r.candidates[0].dispatchId,'dispatch:219:successor:1');
  assert.equal(r.terminalizedHistoricalClaimCount,1);
  assert.equal(r.claimStatus,'NOT_VERIFIED');
});

test('DONE bound to a wrong predecessor comment id never filters',async()=>{
  const r=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,comment219(6894000001,
      doneEvent({predecessor_claim_comment_id:6545079999}))]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(r.reason,'DISPATCH_DONE_EVENT_UNBOUND_OR_DIVERGENT');
  assert.equal(r.candidateCount,0);
  assert.equal(r.terminalizedHistoricalClaimCount,0);
});

test('DONE with wrong dispatch, key, generation or source never filters',async()=>{
  const bads=[
    doneEvent({dispatch_id:'dispatch:219:other:1'}),
    doneEvent({protected_claim_key:'kaicreator-mm/tally#219:reviewer:__default__'}),
    doneEvent({admission_generation:2}),
    doneEvent({sha:curSha}),
    doneEvent({predecessor_claim_comment_id:'6545079271'})
  ];
  for(const bad of bads){
    const r=await readClaimEventCandidatesFromClient({octokit:client219([
      [histClaimComment,comment219(6894000001,bad)]]),
      owner:'kaicreator-mm',repo:'tally',issueNumber:219,
      expectedSourceSha:curSha,limit:1000});
    assert.equal(r.reason,'DISPATCH_DONE_EVENT_UNBOUND_OR_DIVERGENT');
    assert.equal(r.candidateCount,0);
    assert.equal(r.admitted,false);
  }
});

test('cross-Task DONE fails the whole scan like a cross-Task claim',async()=>{
  const r=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,comment219(6894000001,doneEvent({task:'#220'}))]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(r.reason,'CROSS_TASK_OR_STALE_CLAIM_EVENT');
  assert.equal(r.candidateCount,0);
});

test('edited or identity-broken DONE is never a terminal proof',async()=>{
  const edited={...exactDoneComment,updated_at:'2026-11-15T01:00:00Z'};
  const spoofed={...exactDoneComment,
    html_url:issue219Url+'#issuecomment-9999999999'};
  for(const c of [edited,spoofed]){
    const r=await readClaimEventCandidatesFromClient({octokit:client219([
      [histClaimComment,c]]),
      owner:'kaicreator-mm',repo:'tally',issueNumber:219,
      expectedSourceSha:curSha,limit:1000});
    assert.equal(r.reason,'NATIVE_EVENT_IDENTITY_UNVERIFIED');
    assert.equal(r.candidateCount,0);
    assert.equal(r.terminalizedHistoricalClaimCount,0);
  }
});

test('duplicate DONE for one dispatch and DONE for an unknown dispatch fail',async()=>{
  const dup=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,
     comment219(6894000001,doneEvent()),
     comment219(6894000002,doneEvent())]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(dup.reason,'DUPLICATE_DISPATCH_CLAIM_EVENT');
  assert.equal(dup.candidateCount,0);
  const unbound=await readClaimEventCandidatesFromClient({octokit:client219([
    [comment219(6894000001,doneEvent({dispatch_id:'dispatch:219:never-claimed:1'}))]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(unbound.reason,'DISPATCH_DONE_EVENT_UNBOUND_OR_DIVERGENT');
  assert.equal(unbound.candidateCount,0);
});

test('non-DONE state changes stay inert and never terminalize',async()=>{
  const running=comment219(6894000001,
    doneEvent({dispatch_state:'RUNNING',actor_role:'scheduler'}));
  const withClaim=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,running]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(withClaim.reason,'CROSS_TASK_OR_STALE_CLAIM_EVENT');
  assert.equal(withClaim.candidateCount,0);
  const withoutClaim=await readClaimEventCandidatesFromClient({octokit:client219([
    [running]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(withoutClaim.reason,'NO_DISPATCH_CLAIM_EVENTS');
  assert.equal(withoutClaim.candidateCount,0);
});

test('schema-invalid DONE role and malformed DONE body fail closed',async()=>{
  const wrongRole=comment219(6894000001,doneEvent({actor_role:'builder'}));
  const malformed={...exactDoneComment,
    body:'<!-- ads:github -->\n\x60\x60\x60yaml\nschema: ads/github/v3\nschema: ads/github/v3\nkind: event\ndata: {}\n\x60\x60\x60'};
  for(const c of [wrongRole,malformed]){
    const r=await readClaimEventCandidatesFromClient({octokit:client219([
      [histClaimComment,c]]),
      owner:'kaicreator-mm',repo:'tally',issueNumber:219,
      expectedSourceSha:curSha,limit:1000});
    assert.equal(r.reason,'UNVERIFIED_ADS_EVENT_COMMENT');
    assert.equal(r.candidateCount,0);
  }
});

test('a legal DONE never filters a current-source candidate',async()=>{
  const active=comment(101,event({dispatch_id:'dispatch:125:current',
    protected_claim_key:'kaicreator-mm/tally#125:builder:__default__',
    admission_generation:1}));
  const done=comment(102,{schema:'ads/github/v3',kind:'event',
    data:{schema:'ai-dev/event-v2',event:'DISPATCH_STATE_CHANGED',
      actor_role:'scheduler',operator_kind:'other',
      operator_id:'scheduler:i125:terminalization:1',
      task:'#125',issue:'#125',sha,
      dispatch_id:'dispatch:125:current',dispatch_state:'DONE',
      protected_claim_key:'kaicreator-mm/tally#125:builder:__default__',
      admission_generation:1,predecessor_claim_comment_id:101}});
  const r=await readClaimEventCandidatesFromClient({...base,
    octokit:client([[active,done]])});
  assert.equal(r.reason,'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED');
  assert.equal(r.candidateCount,1);
  assert.equal(r.candidates[0].dispatchId,'dispatch:125:current');
  assert.equal(r.terminalizedHistoricalClaimCount,0);
  assert.equal(r.terminalizedHistoricalClaims.length,0);
});

test('fresh successor may not reuse the terminalized historical dispatch id',async()=>{
  const reuse=comment219(6894000003,{schema:'ads/github/v3',kind:'event',
    data:{schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'builder',
      operator_kind:'other',operator_id:'zcode:i219:builder:2',task:'#219',
      issue:'#219',sha:curSha,dispatch_id:histDispatch,
      dispatch_state:'CLAIMED',protected_claim_key:histKey,
      admission_generation:1}});
  const r=await readClaimEventCandidatesFromClient({octokit:client219([
    [histClaimComment,exactDoneComment,reuse]]),
    owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,limit:1000});
  assert.equal(r.reason,'DUPLICATE_DISPATCH_CLAIM_EVENT');
  assert.equal(r.candidateCount,0);
});
