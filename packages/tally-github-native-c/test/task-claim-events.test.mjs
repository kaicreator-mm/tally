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
