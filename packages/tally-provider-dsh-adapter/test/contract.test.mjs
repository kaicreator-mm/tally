import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {
  AGENT_PROVIDER_CONTRACT_VERSION, defineAgentProvider,
  createAgentProviderRegistry, createProviderAttemptPin,
  normalizeProviderObservation,
} from '../../tally-agent-provider/src/index.mjs';
import {createDshAgentProvider,defineDshAgentProviderSpecification}
  from '../src/dsh-adapter.mjs';
import {PINNED_DSH_VERSION} from '../src/dsh-config.mjs';

const API_KEY='test-dsh-secret-0123456789abcdef';
const FAKE_DSH=fileURLToPath(new URL('./fake-dsh.mjs',import.meta.url));
const sha='a'.repeat(40);
const pin=providerId=>createProviderAttemptPin({
  taskId:'264',attemptId:'attempt:264:r1',sourceSha:sha,providerId});

const codexGood=id=>'codex'===id?{
  id:'codex',contractVersion:1,capabilities:[],
  probe:async()=>({state:'AVAILABLE'}),
  execute:async()=>({state:'COMPLETED',evidence:{agentSummary:'done'}}),
  cancel:async()=>({state:'CANCEL_REQUESTED'}),
  reconcile:async()=>({state:'UNKNOWN',reason:'NOT_RECONCILED'})}:null;

/** Real-spawn spec: the fake CLI reports the pinned version. */
const realSpec=overrides=>({binPath:process.execPath,
  binArgsPrefix:[FAKE_DSH,'--fake-version',PINNED_DSH_VERSION],...overrides});

test('the dsh adapter binds the full AgentProvider/1 contract',async()=>{
  const provider=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},config:realSpec()});
  assert.equal(provider.id,'dsh');
  assert.equal(provider.contractVersion,AGENT_PROVIDER_CONTRACT_VERSION);
  assert.deepEqual(provider.capabilities,['headless-turn']);
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-contract-'));
  try{
    const probe=await provider.probe({env:{DEEPSEEK_API_KEY:API_KEY}});
    assert.deepEqual(Object.keys(probe),['state','reason','evidence']);
    assert.equal(probe.state,'AVAILABLE');assert.equal(probe.reason,null);
    assert.equal(probe.evidence.canary,'NOT_RUN',
      'default probe makes NO real turn and reports the canary NOT_RUN');
    assert.equal(probe.evidence.pinnedVersion,PINNED_DSH_VERSION);
    assert.ok(probe.evidence.providerVersion);
    for(const forbidden of ['taskPass','releasePass','claimAccepted','authorized','candidateCommit'])
      assert.equal(forbidden in probe.evidence,false);
    const spec=defineDshAgentProviderSpecification({hostEnv:{},config:{}});
    for(const operation of ['probe','execute','cancel','reconcile'])
      assert.equal(typeof spec[operation],'function');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('the plain specification is not a provider until defineAgentProvider binds it',()=>{
  const spec=defineDshAgentProviderSpecification({hostEnv:{},config:{}});
  assert.deepEqual(Object.keys(spec).sort(),
    ['cancel','capabilities','contractVersion','execute','id','probe','reconcile']);
  // The contract layer freezes observation shape; the spec alone does not.
  assert.throws(()=>normalizeProviderObservation('execute',
    {state:'COMPLETED',taskPass:true}),/UNEXPECTED_FIELD/);
  assert.throws(()=>normalizeProviderObservation('execute',{state:'UNKNOWN'}),
    /UNKNOWN_REASON_REQUIRED/);
  assert.throws(()=>normalizeProviderObservation('probe',{state:'FAILED'}),
    /STATE_INVALID/);
});

test('liar observations can never pass the contract layer',async()=>{
  const liar=defineAgentProvider({...codexGood('codex'),
    execute:async()=>({state:'COMPLETED',taskPass:true})});
  // The hardened boundary refuses bare/malformed requests before any adapter
  // code runs, so a liar observation needs a legal request to reach (and fail)
  // the contract's observation normalization.
  await assert.rejects(()=>liar.execute(),{name:'TypeError',
    message:'PROVIDER_REQUEST_INVALID'});
  await assert.rejects(()=>liar.execute({taskId:'264',
    attemptId:'attempt:264:r1',sourceSha:sha,prompt:'irrelevant',
    worktreePath:'w'}),/UNEXPECTED_FIELD/);
  const dsh=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},config:realSpec()});
  const result=await dsh.execute({taskId:'264',attemptId:'attempt:264:r1',
    sourceSha:sha,prompt:'irrelevant — refused before spawn',
    worktreePath:'C:\\definitely\\missing\\workspace'},
    {env:{DEEPSEEK_API_KEY:API_KEY}});
  assert.equal(result.state,'FAILED');
  assert.deepEqual(Object.keys(result),['state','reason','evidence']);
});

test('registry: dsh is default-DISABLED and explicit enablement is the only door',async()=>{
  const dsh=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},config:realSpec()});
  const registry=createAgentProviderRegistry([codexGood('codex'),dsh]);
  assert.equal(registry.resolve().id,'codex',
    'default provider is codex — no implicit dsh, no fallback');
  assert.throws(()=>registry.resolve({providerId:'dsh'}),/PROVIDER_DISABLED/);
  assert.throws(()=>registry.resolve({providerId:'dsh',
    enabledProviderIds:['codex']}),/PROVIDER_DISABLED/,
    'an enabled list without dsh must NOT fall back to codex');
  assert.equal(registry.resolve({providerId:'dsh',
    enabledProviderIds:['codex','dsh']}).id,'dsh');
  assert.throws(()=>registry.resolve({providerId:'grok',
    enabledProviderIds:['grok']}),/PROVIDER_NOT_INSTALLED/);
  assert.throws(()=>createAgentProviderRegistry([dsh,dsh]),/PROVIDER_DUPLICATE_ID/);
});

test('one attempt pin cannot be silently switched to or from dsh',async()=>{
  const dsh=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},config:realSpec()});
  const registry=createAgentProviderRegistry([codexGood('codex'),dsh]);
  assert.equal(registry.resolve({pin:pin('dsh'),
    enabledProviderIds:['dsh']}).id,'dsh');
  assert.throws(()=>registry.resolve({providerId:'codex',pin:pin('dsh'),
    enabledProviderIds:['dsh']}),/PROVIDER_ATTEMPT_PIN_MISMATCH/);
  assert.throws(()=>registry.resolve({providerId:'dsh',pin:pin('codex'),
    enabledProviderIds:['dsh']}),/PROVIDER_ATTEMPT_PIN_MISMATCH/);
  assert.throws(()=>createProviderAttemptPin({taskId:'264',
    attemptId:'attempt:264:r1',sourceSha:sha,providerId:'DSH'}),
    /PROVIDER_ID_INVALID/);
});

test('UNKNOWN observations always carry a reason; reconcile is honest',async()=>{
  const dsh=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},config:realSpec()});
  const off=await dsh.reconcile({taskId:'264',attemptId:'attempt:264:r1',
    sourceSha:sha,providerId:'dsh'});
  assert.equal(off.state,'UNKNOWN');
  assert.equal(off.reason,'ATTEMPT_STATUS_UNKNOWN');
  assert.equal(off.evidence.durableJournal,false,
    'no durable execution journal is claimed — one-shot process model');
  const cancelOff=await dsh.cancel({taskId:'264',attemptId:'attempt:gone',
    sourceSha:sha,providerId:'dsh'});
  assert.equal(cancelOff.state,'NOT_RUNNING');
  assert.equal(cancelOff.reason,'ATTEMPT_NOT_RUNNING');
});

test('the default defineAgentProvider fallback resolves in the monorepo layout',async()=>{
  const provider=await createDshAgentProvider({hostEnv:{},config:{}});
  assert.equal(provider.id,'dsh');
  assert.equal(typeof provider.execute,'function');
});
