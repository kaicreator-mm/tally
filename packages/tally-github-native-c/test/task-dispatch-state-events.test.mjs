import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {TARGET,readDispatchStateEventsFromClient} from '../src/task-dispatch-state-events.mjs';
import {terminalEventForReservation} from '../src/task-claim-github-publisher.mjs';
const authority={mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,repository:TARGET.repository,
  writerId:'tally-host:terminal',githubWriterLogin:'kaicreator-mm'};
const reservation={...TARGET,phase:'PENDING_GITHUB_TERMINALIZATION',commentId:TARGET.claimCommentId,
  commentRef:TARGET.claimCommentRef,terminalWriterId:authority.writerId,terminalEvidence:'durable proof',
  terminalReservationRevision:2,
  terminalOccurredAt:'2026-11-06T00:00:00.000Z'};
const event=terminalEventForReservation(reservation,authority);
const body=e=>'<!-- ads:github -->\n```yaml\n'+stringify({schema:'ads/github/v3',kind:'event',data:e})+'```\n';
const comment=(id,e=event,changes={})=>({id,html_url:`${TARGET.issueRef}#issuecomment-${id}`,
  issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/203',user:{login:'kaicreator-mm'},
  created_at:'2026-11-06T00:00:00Z',updated_at:'2026-11-06T00:00:00Z',body:body(e),...changes});
function fake({comments=[],count=comments.length,drift=false,broken=false,detail}={}){
  let reads=0;
  const octokit={rest:{issues:{
    async get(){return {data:{id:2030,number:203,html_url:TARGET.issueRef,state:'open',comments:count,
      body:`Baseline: version/v2@${TARGET.sourceSha}`,
      updated_at:drift&&++reads>1?'later':'now'}};},
    async listComments(){},async getComment({comment_id}){return {data:detail??comments.find(c=>c.id===comment_id)};}
  },repos:{async getCommit(){return {data:{sha:TARGET.sourceSha}};}}},
  paginate:{async *iterator(){if(broken)throw Error('lost page');
    yield {data:comments.slice(0,1)};yield {data:comments.slice(1)};}}};
  return octokit;
}
const read=(octokit,expectedEvent=event)=>readDispatchStateEventsFromClient({octokit,writerLogin:'kaicreator-mm',expectedEvent});
test('complete stable pagination proves absence or one exact unedited event',async()=>{
  assert.equal((await read(fake(),null)).status,'ABSENT');
  const result=await read(fake({comments:[comment(1),{...comment(2),body:'ordinary comment'}]}));
  assert.equal(result.status,'EXACT');assert.equal(result.commentId,1);
});
test('incomplete pagination, drift, edited, duplicate, conflicting, cross-task and stale events fail closed',async()=>{
  const cases=[{comments:[comment(1)],count:2},{comments:[comment(1)],broken:true},
    {comments:[comment(1)],drift:true},{comments:[comment(1,event,{updated_at:'later'})]},
    {comments:[comment(1),comment(2)]},{comments:[comment(1,{...event,dispatch_state:'FAILED'})]},
    {comments:[comment(1,{...event,issue:'#204'})]},{comments:[comment(1,{...event,sha:'a'.repeat(40)})]},
    {comments:[comment(1,event,{body:'<!-- ads:github -->\nmalformed'})]},
    {comments:[comment(1)],detail:comment(1,event,{body:'edited'})}];
  for(const value of cases)assert.equal((await read(fake(value))).status,'UNKNOWN',JSON.stringify(value));
  assert.equal((await read(fake({comments:[comment(1)]}),null)).status,'UNKNOWN');
});
