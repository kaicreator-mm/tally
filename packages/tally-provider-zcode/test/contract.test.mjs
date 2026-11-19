import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,mkdir,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {AGENT_PROVIDER_CONTRACT_VERSION,DEFAULT_AGENT_PROVIDER,
  createAgentProviderRegistry,createProviderAttemptPin,
  normalizeProviderObservation} from '../../tally-agent-provider/src/index.mjs';
import {createZcodeAgentProvider,createDefaultOperations} from '../src/index.mjs';

const good=(id='codex')=>({id,contractVersion:1,capabilities:[],
  probe:async()=>({state:'AVAILABLE'}),
  execute:async()=>({state:'COMPLETED'}),
  cancel:async()=>({state:'CANCEL_REQUESTED'}),
  reconcile:async()=>({state:'UNKNOWN',reason:'NOT_RECONCILED'})});
const pin=(providerId='zcode')=>createProviderAttemptPin({
  taskId:'473',attemptId:'attempt:zcode:r1',sourceSha:'a'.repeat(40),providerId});

function zcodeProvider(overrides={}){
  return createZcodeAgentProvider({
    operations:{probeProvider:async()=>({available:true,provider:'zcode',version:'0.16.9',
        canary:{sessionId:'sess_probe',response:'OK'}}),
      attemptRecord:async()=>null,
      runAgentProcess:()=>{throw new Error('must not spawn in contract tests');}},
    ...overrides});
}

test('the zcode adapter wraps through defineAgentProvider as a v1 provider',async()=>{
  assert.equal(AGENT_PROVIDER_CONTRACT_VERSION,1);
  const provider=zcodeProvider();
  assert.equal(provider.id,'zcode');
  assert.equal(provider.contractVersion,AGENT_PROVIDER_CONTRACT_VERSION);
  for(const operation of ['probe','execute','cancel','reconcile'])
    assert.equal(typeof provider[operation],'function');
  const probe=await provider.probe();
  assert.deepEqual(probe,{state:'AVAILABLE',reason:null,
    evidence:{provider:'zcode',version:'0.16.9',
      canary:{sessionId:'sess_probe',response:'OK'}}});
  assert.ok(Object.isFrozen(probe),'observations are frozen');
  assert.equal((await provider.cancel({taskId:'473',attemptId:'attempt:zcode:r1',
    sourceSha:'a'.repeat(40),providerId:'zcode'})).state,'NOT_RUNNING');
});

test('registry default stays codex and zcode needs an explicit enabled row',()=>{
  const registry=createAgentProviderRegistry([good(),zcodeProvider()]);
  assert.equal(DEFAULT_AGENT_PROVIDER,'codex');
  assert.equal(registry.resolve().id,'codex','resolve() never selects zcode implicitly');
  assert.equal(registry.resolve({providerId:'codex'}).id,'codex');
  assert.throws(()=>registry.resolve({providerId:'zcode'}),/PROVIDER_DISABLED/);
  assert.equal(registry.resolve({providerId:'zcode',
    enabledProviderIds:['codex','zcode']}).id,'zcode');
  assert.throws(()=>registry.resolve({providerId:'grok',
    enabledProviderIds:['grok']}),/PROVIDER_NOT_INSTALLED/);
});

test('attempt pins bind to zcode and cannot be silently switched',()=>{
  const registry=createAgentProviderRegistry([good(),zcodeProvider()]);
  assert.equal(registry.resolve({pin:pin('zcode'),
    enabledProviderIds:['zcode']}).id,'zcode');
  assert.throws(()=>registry.resolve({providerId:'codex',pin:pin('zcode')}),
    /PROVIDER_ATTEMPT_PIN_MISMATCH/);
  assert.throws(()=>pin('ZCODE'),/PROVIDER_ID_INVALID/);
});

test('zcode observations are three-key, untrusted, and cannot assert authority',async()=>{
  const registry=createAgentProviderRegistry([good(),zcodeProvider()]);
  const provider=registry.resolve({providerId:'zcode',enabledProviderIds:['codex','zcode']});
  const observation=await provider.execute({taskId:'473',
    attemptId:'attempt:zcode:r1',sourceSha:'a'.repeat(40),
    prompt:'do it',worktreePath:join('C:','nowhere')});
  assert.deepEqual(Object.keys(observation),['state','reason','evidence']);
  assert.ok(Object.isFrozen(observation));
  // A lying execute is structurally impossible to pass through: the contract
  // normalization rejects authority-claiming fields before any Host reads it.
  for(const field of ['taskPass','releasePass','claimAccepted','authorized','candidateCommit'])
    assert.throws(()=>normalizeProviderObservation('execute',
      {state:'COMPLETED',[field]:true}),/PROVIDER_OBSERVATION_UNEXPECTED_FIELD/);
  const liar=createAgentProviderRegistry([{...good('zcode'),
    execute:async()=>({state:'COMPLETED',taskPass:true})}]);
  await assert.rejects(()=>liar.resolve({providerId:'zcode',
    enabledProviderIds:['zcode']}).execute({taskId:'473',
    attemptId:'attempt:zcode:r1',sourceSha:'a'.repeat(40),
    prompt:'do it',worktreePath:join('C:','nowhere')}),/UNEXPECTED_FIELD/);
  assert.throws(()=>normalizeProviderObservation('execute',{state:'UNKNOWN'}),
    /PROVIDER_UNKNOWN_REASON_REQUIRED/);
  assert.throws(()=>normalizeProviderObservation('probe',{state:'COMPLETED'}),
    /PROVIDER_OBSERVATION_STATE_INVALID/);
});

test('an UNAVAILABLE probe yields FAILED execute — never a quiet COMPLETED or fallback',async()=>{
  const unavailable=createZcodeAgentProvider({
    operations:{probeProvider:async()=>({available:false,provider:'zcode',
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY',outputMatched:false,
        exitCode:null,detail:'CREDENTIALS_OR_PROVIDER_STALL'}),
      attemptRecord:async()=>null,
      runAgentProcess:()=>{throw new Error('must not spawn');}}});
  const x=await unavailable.execute({taskId:'473',attemptId:'attempt:zcode:r1',
    sourceSha:'a'.repeat(40),prompt:'do it',worktreePath:'C:/nowhere'});
  assert.equal(x.state,'FAILED');
  assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
  assert.equal(x.evidence.agentLaunch,false);
  // No failure-driven fallback exists anywhere in the surface: a failed zcode
  // probe has no API path that would return the codex provider instead.
  const registry=createAgentProviderRegistry([good(),unavailable]);
  assert.equal(registry.resolve().id,'codex');
  assert.throws(()=>registry.resolve({providerId:'zcode'}),/PROVIDER_DISABLED/);
});

test('UNKNOWN observations always carry a reason',async()=>{
  const unknown=createZcodeAgentProvider({
    operations:{probeProvider:async()=>({available:true,provider:'zcode',version:'0.16.9'}),
      attemptRecord:async()=>({kind:'WORKTREE_OCCUPIED'}),
      runAgentProcess:()=>{throw new Error('must not spawn');}}});
  const r=await unknown.reconcile({taskId:'473',attemptId:'attempt:zcode:r1',
    sourceSha:'a'.repeat(40),providerId:'zcode'});
  assert.equal(r.state,'UNKNOWN');
  assert.equal(r.reason,'ATTEMPT_STATUS_UNKNOWN');
  assert.ok(r.reason!==null,'contract requires a reason on UNKNOWN');
});
