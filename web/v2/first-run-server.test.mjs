import test from 'node:test';
import assert from 'node:assert/strict';
import {startFirstRunServer} from './first-run-server.mjs';
const sha='a'.repeat(40);
const state=()=>({
  currentSelection:null,projects:[],
  async status(){return {selected:this.currentSelection,projects:this.projects,configured:!!this.projects.length};},
  async add({repository,ref,token}){
    assert.equal(token,'github_pat_test_TEST_ONLY');
    this.currentSelection={repository,ref};
    this.projects=[{repository,title:'test',versions:[{ref,sha}]}];
    return this.status();
  },
  async select({repository,ref}){
    this.currentSelection={repository,ref};
    return {env:{TALLY_V2_REPOSITORY:repository,TALLY_V2_VERSION_REF:ref,
      TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'github_pat_test_TEST_ONLY'}};
  },
  async selected(){return null;},
});
test('first-run setup serves without env; POST is same-origin and unprivileged',async()=>{
  const store=state();let mounted;
  const service=await startFirstRunServer({env:{
    TALLY_V2_AGENT_DISPATCH:'host-mounted',
    TALLY_V2_RELEASE_DISPATCH:'host-mounted',
  },catalog:store,startConfigured:async ({env})=>{
    mounted=env;
    return {address:{origin:'http://127.0.0.1:65535'},close:async()=>{}};
  }});
  try{
    const url=service.address.origin;
    const get=await fetch(url+'/api/setup/status');
    const data=await get.json();
    assert.equal(data.configured,false);
    assert.equal(data.activeUrl,null);
    const html=await (await fetch(url+'/setup')).text();
    const key=/<meta name="tally-setup-key" content="([^"]+)"/.exec(html)?.[1];
    assert.ok(key);
    const send=(origin,tokenKey)=>fetch(url+'/api/setup/add',{
      method:'POST',headers:{'content-type':'application/json',
        origin,'x-tally-setup-key':tokenKey},
      body:JSON.stringify({repository:'org/test',ref:'main',
        token:'github_pat_test_TEST_ONLY'})});
    assert.equal((await send('http://example.com',key)).status,403);
    assert.equal((await send(url,'invalid')).status,403);
    const response=await send(url,key);
    assert.equal(response.status,200);
    const connected=await response.json();
    assert.equal(connected.status,'CONNECTED');
    assert.equal(mounted.TALLY_V2_REPOSITORY,'org/test');
    assert.equal(mounted.TALLY_V2_VERSION_SHA,sha);
    assert.equal(mounted.TALLY_V2_AGENT_DISPATCH,undefined);
    assert.equal(mounted.TALLY_V2_RELEASE_DISPATCH,undefined);
    const safe=await (await fetch(url+'/api/setup/status')).text();
    assert.doesNotMatch(safe,/github_pat/);
    const redirect=await fetch(url+'/app',{redirect:'manual'});
    assert.equal(redirect.status,302);
    assert.equal(redirect.headers.get('location'),'http://127.0.0.1:65535/app');
  }finally{await service.close();}
});

test('legacy explicit operator Host pins remain intact at first boot',async()=>{
  let observed=null;
  const service=await startFirstRunServer({env:{
    TALLY_V2_REPOSITORY:'org/legacy',TALLY_V2_VERSION_REF:'version/v2',
    TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'github_pat_test_ONLY',
    TALLY_V2_AGENT_DISPATCH:'host-mounted',
  },catalog:state(),startConfigured:async ({env})=>{
    observed=env;
    return {address:{origin:'http://127.0.0.1:65534'},close:async()=>{}};
  }});
  try{
    assert.equal(observed.TALLY_V2_AGENT_DISPATCH,'host-mounted');
    assert.equal(observed.TALLY_V2_REPOSITORY,'org/legacy');
  }finally{await service.close();}
});

test('Setup can create explicit draft Issue only for selected repo, never Agent',async()=>{
  const store=state();
  store.currentSelection={repository:'org/test',ref:'main'};
  store.projects=[{repository:'org/test',title:'test',versions:[{ref:'main',sha}]}];
  let created=null;
  const service=await startFirstRunServer({env:{},catalog:store,
    createDraft:async input=>{
      created=input;
      return {status:'DRAFT_ISSUE_CREATED',issueNumber:18,
        issueUrl:'https://github.com/org/test/issues/18',
        taskQualified:false,agentAuthorized:false};
    },startConfigured:async()=>({
      address:{origin:'http://127.0.0.1:65532'},close:async()=>{}})});
  try{
    const url=service.address.origin;
    const html=await (await fetch(url+'/setup')).text();
    const key=/<meta name="tally-setup-key" content="([^"]+)"/.exec(html)?.[1];
    assert.ok(key);
    const send=(repository,requestId,tokenKey=key)=>fetch(url+'/api/setup/task-draft',{
      method:'POST',headers:{'content-type':'application/json',
        origin:url,'x-tally-setup-key':tokenKey},
      body:JSON.stringify({repository,ref:'main',title:'Create API',
        description:'Detailed task',writeToken:'github_pat_TEST_WRITE',
        requestId})});
    assert.equal((await send('org/foreign','task-draft-1')).status,422);
    assert.equal((await send('org/test','task-draft-2','wrong')).status,403);
    const outcome=await send('org/test','task-draft-3');
    assert.equal(outcome.status,201);
    const data=await outcome.json();
    assert.equal(data.taskQualified,false);
    assert.equal(data.agentAuthorized,false);
    assert.equal(created.repository,'org/test');
    assert.equal(created.expectedSha,sha);
    assert.equal((await send('org/test','task-draft-3')).status,422);
  }finally{await service.close();}
});

test('managed selection preserves operator mount only at the exact same SHA',async()=>{
  const store=state();
  const shaOther='b'.repeat(40);
  let mounted=null;
  const service=await startFirstRunServer({env:{
    TALLY_V2_REPOSITORY:'org/test',TALLY_V2_VERSION_REF:'main',
    TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:'github_pat_test_ONLY',
    TALLY_V2_AGENT_DISPATCH:'host-mounted',
  },catalog:store,startConfigured:async ({env})=>{
    mounted=env;
    return {address:{origin:'http://127.0.0.1:65533'},close:async()=>{}};
  }});
  try{
    assert.equal(mounted.TALLY_V2_AGENT_DISPATCH,'host-mounted');
    const url=service.address.origin;
    const html=await (await fetch(url+'/setup')).text();
    const key=/<meta name="tally-setup-key" content="([^"]+)"/.exec(html)?.[1];
    const add=async repository=>fetch(url+'/api/setup/add',{method:'POST',
      headers:{'content-type':'application/json',origin:url,
        'x-tally-setup-key':key},
      body:JSON.stringify({repository,ref:'main',token:'github_pat_test_ONLY'})});
    assert.equal((await add('org/test')).status,200);
    assert.equal(mounted.TALLY_V2_AGENT_DISPATCH,'host-mounted');
    assert.equal((await add('org/other')).status,200);
    assert.equal(mounted.TALLY_V2_REPOSITORY,'org/other');
    assert.equal(mounted.TALLY_V2_AGENT_DISPATCH,undefined);
  }finally{await service.close();}
});

test('corrupt catalog returns bounded 500 instead of crashing loopback server',async()=>{
  const store=state();
  store.status=async()=>{throw new Error('PROJECT_STORE_CORRUPT');};
  const service=await startFirstRunServer({env:{},catalog:store,
    startConfigured:async()=>{throw Error('UNEXPECTED_START');}});
  try{
    const origin=service.address.origin;
    const response=await fetch(origin+'/api/setup/status');
    assert.equal(response.status,500);
    const data=await response.json();
    assert.deepEqual(data,{status:'DENIED',reason:'PROJECT_STORE_CORRUPT'});
    const stillRunning=await fetch(origin+'/setup');
    assert.equal(stillRunning.status,200);
  }finally{await service.close();}
});
