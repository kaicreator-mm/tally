import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiActionHttpAdapter} from './ui-action-port.mjs';
const origin='http://127.0.0.1:43210',csrfKey='c'.repeat(48);
const repository='org/tally',versionId='version/v2',sourceSha='a'.repeat(40);
const actionPort={actions:async()=>({connected:true,source:{repository,
  sourceSha,currentness:'CURRENT',readback:'VERIFIED'},versionId,actions:[]}),
  submit:async request=>({connected:true,ok:true,status:'OBSERVED',
    actionId:request.actionId,sourceSha,receiptRef:'https://github.com/org/tally/issues/1#issuecomment-1'})};
const adapter=()=>createUiActionHttpAdapter({actionPort,origin,csrfKey,repository,versionId});
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
  const h=createUiActionHttpAdapter({actionPort:port,origin,csrfKey,repository,versionId});
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

test('HTTP adapter requires explicit selected repository and version scope',()=>{
  assert.throws(()=>createUiActionHttpAdapter({actionPort,origin,csrfKey}),
    /UI_ACTION_HTTP_HOST_CONFIGURATION_REQUIRED/);
  assert.throws(()=>createUiActionHttpAdapter({actionPort,origin,csrfKey,
    repository:'evil/repo',versionId}),
    /UI_ACTION_HTTP_HOST_CONFIGURATION_REQUIRED/);
});
