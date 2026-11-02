import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiActionHttpAdapter} from './ui-action-port.mjs';
const origin='http://127.0.0.1:43210',csrfKey='c'.repeat(48);
const repository='org/tally',versionId='version/v2',sourceSha='a'.repeat(40);
const actionPort={scope:Object.freeze({repository,versionId,sourceSha}),
  actions:async()=>({connected:true,source:{repository,
  sourceSha,currentness:'CURRENT',readback:'VERIFIED'},versionId,actions:[]}),
  submit:async request=>({connected:true,ok:true,status:'OBSERVED',
    actionId:request.actionId,sourceSha,receiptRef:'https://github.com/org/tally/issues/1#issuecomment-1'}),
  taskExecution:async taskIssue=>({connected:true,
    source:{repository,sourceSha,currentness:'CURRENT',readback:'VERIFIED'},
    versionId,taskIssue,execution:{state:'RUNNING',provider:'local-codex',
      role:'BUILDER',attemptId:'attempt-1',progressSummary:'任务执行中',
      progressSummaryKind:'SANITIZED',heartbeatAt:'2026-11-02T12:30:00Z',
      elapsedMs:1000,tokensIn:null,tokensOut:null,costUsd:null,candidateRef:null}})};
const adapter=()=>createUiActionHttpAdapter({actionPort,origin,csrfKey});
test('same-origin GET reads a scoped Host action catalog',async()=>{
  const r=await adapter().handle({method:'GET',path:'/api/v2/actions',origin,
    query:{projectId:repository,versionId}});
  assert.equal(r.status,200);assert.equal(r.body.connected,true);
});
test('foreign origin and scope cannot read action catalog',async()=>{
  assert.equal((await adapter().handle({method:'GET',path:'/api/v2/actions',
    origin:'https://evil.example',query:{projectId:repository,versionId}})).status,403);
  assert.equal((await adapter().handle({method:'GET',path:'/api/v2/actions',origin,
    query:{projectId:'evil/repo',versionId}})).status,400);
});
test('dispatch requires exact CSRF and allowlisted request body',async()=>{
  let calls=0;
  const port={...actionPort,submit:async x=>{calls++;return actionPort.submit(x)}};
  const h=createUiActionHttpAdapter({actionPort:port,origin,csrfKey});
  const body={projectId:repository,versionId,sourceSha,actionId:'start-1',revision:1};
  assert.equal((await h.handle({method:'POST',path:'/api/v2/actions/dispatch',
    origin,csrfKey:'wrong',body})).status,403);
  assert.equal((await h.handle({method:'POST',path:'/api/v2/actions/dispatch',
    origin,csrfKey,body:{...body,grant:true}})).status,400);
  assert.equal(calls,0);
  const r=await h.handle({method:'POST',path:'/api/v2/actions/dispatch',
    origin,csrfKey,body});
  assert.equal(r.status,200);assert.equal(calls,1);
});
test('unknown route and malformed origin fail closed',async()=>{
  assert.equal((await adapter().handle({method:'POST',path:'/api/v2/other',
    origin,csrfKey,body:{}})).status,404);
  assert.equal((await adapter().handle({method:'GET',path:'/api/v2/actions',
    origin:'null',query:{projectId:repository,versionId}})).status,403);
});

test('HTTP adapter pins scope to immutable selected port and rejects caller override',async()=>{
  const missingScope={actions:actionPort.actions,submit:actionPort.submit};
  assert.throws(()=>createUiActionHttpAdapter({actionPort:missingScope,origin,csrfKey}),
    /UI_ACTION_HTTP_HOST_CONFIGURATION_REQUIRED/);
  const mutableScope={...actionPort,scope:{repository,versionId,sourceSha}};
  assert.throws(()=>createUiActionHttpAdapter({actionPort:mutableScope,origin,csrfKey}),
    /UI_ACTION_HTTP_HOST_CONFIGURATION_REQUIRED/);
  const h=createUiActionHttpAdapter({actionPort,origin,csrfKey,
    repository:'evil/repo',versionId:'version/v99'});
  const selected=await h.handle({method:'GET',path:'/api/v2/actions',origin,
    query:{projectId:repository,versionId}});
  assert.equal(selected.status,200);
  const foreign=await h.handle({method:'GET',path:'/api/v2/actions',origin,
    query:{projectId:'evil/repo',versionId:'version/v99'}});
  assert.equal(foreign.status,400);
});

test('same-origin task execution GET reads only the selected task lifecycle',async()=>{
  const r=await adapter().handle({method:'GET',path:'/api/v2/task-execution',origin,
    query:{projectId:repository,versionId,taskIssue:260}});
  assert.equal(r.status,200);
  assert.equal(r.body.connected,true);
  assert.equal(r.body.taskIssue,260);
  assert.equal(r.body.execution.state,'RUNNING');
});
test('task execution GET rejects foreign scope, malformed task and missing Host port',async()=>{
  const h=adapter();
  assert.equal((await h.handle({method:'GET',path:'/api/v2/task-execution',
    origin,query:{projectId:'evil/repo',versionId,taskIssue:260}})).status,400);
  assert.equal((await h.handle({method:'GET',path:'/api/v2/task-execution',
    origin,query:{projectId:repository,versionId,taskIssue:'260'}})).status,400);
  const missing=createUiActionHttpAdapter({actionPort:{
    scope:actionPort.scope,actions:actionPort.actions,submit:actionPort.submit
  },origin,csrfKey});
  assert.equal((await missing.handle({method:'GET',path:'/api/v2/task-execution',
    origin,query:{projectId:repository,versionId,taskIssue:260}})).status,503);
});
