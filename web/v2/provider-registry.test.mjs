import test from 'node:test';
import assert from 'node:assert/strict';
import {createProviderRegistry} from './provider-registry.mjs';

function fakeProvider(overrides={}){
  return Object.freeze({providerId:'zcode',
    probe:async()=>({available:true,provider:'zcode',version:'0.16.9'}),
    execute:async input=>({providerId:'zcode',attemptId:input.attemptId,
      sourceSha:input.sourceSha,state:'LOCAL_CANDIDATE_READY',
      reason:'FAKE_READY',agentLaunch:true,
      process:{status:'EXITED',exitCode:0,pid:1,startedAt:null,terminatedAt:null},
      taskPass:false,releasePass:false}),
    cancel:async()=>true,readback:async()=>({verified:true}),...overrides});
}

test('register validates interface and rejects duplicates',()=>{
  const registry=createProviderRegistry();
  registry.register(fakeProvider());
  assert.throws(()=>registry.register(fakeProvider()),/PROVIDER_ALREADY_REGISTERED/);
  assert.throws(()=>registry.register({providerId:'x'}),/PROVIDER_INTERFACE_NONCONFORMANT/);
  assert.equal(registry.get('zcode').providerId,'zcode');
  assert.equal(registry.get('missing'),null);
});

test('descriptors are DISABLED by default, AVAILABLE only when enabled+probe passes',()=>{
  const registry=createProviderRegistry({providers:[fakeProvider()]});
  assert.equal(registry.describe('zcode').status,'DISABLED');
  registry.setEnabled('zcode',true);
  assert.equal(registry.describe('zcode').status,'UNKNOWN');
  registry.select({requested:'zcode'}); // no probe yet
  assert.equal(registry.describe('zcode').status,'UNKNOWN');
  return registry.refreshProbes().then(()=>{
    assert.equal(registry.describe('zcode').status,'AVAILABLE');
    const selection=registry.select({requested:'zcode'});
    assert.equal(selection.admitted,true);
  });
});

test('failed probe is recorded honestly and never fabricates availability',async()=>{
  const registry=createProviderRegistry({providers:[fakeProvider({
    probe:async()=>({available:false,provider:'zcode',
      reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY'})})]});
  registry.setEnabled('zcode',true);
  const refreshed=await registry.refreshProbes();
  assert.equal(refreshed.zcode.available,false);
  assert.equal(registry.describe('zcode').status,'UNKNOWN');
  assert.equal(registry.select({requested:'zcode'}).reason,'PROVIDER_CAPABILITY_UNVERIFIED');
});

test('throwing probe degrades to unavailable without crashing refresh',async()=>{
  const registry=createProviderRegistry({providers:[fakeProvider({
    probe:async()=>{throw new Error('boom');}})]});
  registry.setEnabled('zcode',true);
  const refreshed=await registry.refreshProbes();
  assert.equal(refreshed.zcode.available,false);
  assert.equal(refreshed.zcode.detail,'PROBE_THREW');
});

test('select keeps codex default outside the registry',()=>{
  const registry=createProviderRegistry({providers:[fakeProvider()]});
  const selection=registry.select({requested:undefined});
  assert.equal(selection.selected,'codex');
  assert.equal(selection.route,'EXISTING_CHAIN');
  assert.deepEqual(registry.list().map(x=>x.providerId),['zcode']);
});
