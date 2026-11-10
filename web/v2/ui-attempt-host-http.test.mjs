import test from 'node:test';
import assert from 'node:assert/strict';
import {createTaskAttemptHttpHandler} from './ui-attempt-host.mjs';
const id={status:'VERIFIED',repository:'kaicreator-mm/tally',
  versionId:'version/v2',taskIssueNumber:275,sourceSha:'a'.repeat(40)};
const res=()=>({headers:{},statusCode:0,body:'',
  setHeader(k,v){this.headers[k]=v;},
  end(s){this.body=s;}});
const port={readTaskAttemptRoster:async()=>({...id,status:'VERIFIED',
  complete:true,attempts:[]})};
const handler=()=>createTaskAttemptHttpHandler({
  selectedHost:port,resolveSelectedTask:async()=>id});
test('only GET is permitted; zero mutations from non-GET',async()=>{
  const r=res();await handler()({method:'POST',
    url:'/api/v2/task-attempts?taskIssueNumber=275'},r);
  assert.equal(r.statusCode,405);
  assert.equal(JSON.parse(r.body).authorized,false);
});
test('valid GET shows a qualified empty selected Host roster, not inferred Task PASS',async()=>{
  const r=res();await handler()({method:'GET',
    url:'/api/v2/task-attempts?taskIssueNumber=275'},r);
  const v=JSON.parse(r.body);
  assert.equal(r.statusCode,200);assert.equal(v.status,'OBSERVED_EMPTY');
  assert.equal(v.coverage.attemptCount,0);
  assert.equal(v.taskPass,false);assert.equal(v.releasePass,false);
  assert.equal(r.headers['cache-control'],'no-store');
});
test('GET rejects unknown params and duplicate query keys',async()=>{
  const r=res();await handler()({method:'GET',
    url:'/api/v2/task-attempts?taskIssueNumber=275&sourceSha=forged'},r);
  assert.equal(r.statusCode,404);
  const s=res();await handler()({method:'GET',
    url:'/api/v2/task-attempts?taskIssueNumber=275&taskIssueNumber=275'},s);
  assert.equal(s.statusCode,404);
});
test('untrusted UI cannot override selected source identity',async()=>{
  const h=createTaskAttemptHttpHandler({selectedHost:port,
    resolveSelectedTask:async()=>({...id,sourceSha:'INVALID'})});
  const r=res();await h({method:'GET',
    url:'/api/v2/task-attempts?taskIssueNumber=275'},r);
  assert.equal(r.statusCode,403);
});
test('no trusted source resolver is a construction error',()=>{
  assert.throws(()=>createTaskAttemptHttpHandler({selectedHost:port}),
    /SELECTED_TASK_IDENTITY_RESOLVER_REQUIRED/);
});
test('unavailable Host stays UNKNOWN without fabricating an empty roster',async()=>{
  const h=createTaskAttemptHttpHandler({selectedHost:{},
    resolveSelectedTask:async()=>id});
  const r=res();await h({method:'GET',
    url:'/api/v2/task-attempts?taskIssueNumber=275'},r);
  assert.equal(r.statusCode,200);
  const v=JSON.parse(r.body);
  assert.equal(v.status,'UNKNOWN');
  assert.equal(v.coverage.attemptCount,null);
});
