import test from 'node:test';
import assert from 'node:assert/strict';
import {startFirstRunServer} from './first-run-server.mjs';
const sha='a'.repeat(40);
const state=()=>({
  selected:null,projects:[],
  async status(){return {selected:this.selected,projects:this.projects,configured:!!this.projects.length};},
  async add({repository,ref,token}){
    assert.equal(token,'github_pat_test_TEST_ONLY');
    this.selected={repository,ref};
    this.projects=[{repository,title:'test',versions:[{ref,sha}]}];
    return this.status();
  },
  async select({repository,ref}){
    this.selected={repository,ref};
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
