import test from 'node:test';
import assert from 'node:assert/strict';
import {PROVIDER_CONTRACT_IDS,validateProviderAttemptInput,
  validateProviderAttemptOutcome,assertProviderInterface,
  resolveProviderSelection} from './provider-contract.mjs';

const sourceSha='a'.repeat(40);
const input={providerId:'zcode',taskId:'task:1',attemptId:'attempt:1',
  sourceSha,worktreePath:'C:\\wt\\attempt-1',
  allowedWriteSet:['experiments/v2/selected-business-build/LOCAL_X_L3.md'],
  taskPrompt:'Write the candidate note.',issueRef:'https://github.com/kaicreator-mm/tally/issues/64',
  hostAuthorization:{claimKey:'k',dispatchId:'d',admissionGeneration:1},
  providerConfig:{cliPath:'C:\\cli\\x.mjs'},timeoutMs:60000};
const outcome={providerId:'zcode',provider:'zcode',attemptId:'attempt:1',sourceSha,
  state:'LOCAL_CANDIDATE_READY',reason:'REAL_ZCODE_CANDIDATE_RECORDED',agentLaunch:true,
  process:{status:'EXITED',exitCode:0,pid:4242,startedAt:'2026-11-19T08:00:00.000Z',
    terminatedAt:'2026-11-19T08:01:00.000Z'},candidateTree:'b'.repeat(40),
  candidateCommit:'c'.repeat(40),candidateRef:'refs/tally-zcode-candidates/x',
  candidatePaths:['experiments/v2/selected-business-build/LOCAL_X_L3.md'],
  taskPass:false,releasePass:false};

test('contract ids freeze the shared vocabulary',()=>{
  assert.equal(PROVIDER_CONTRACT_IDS.defaultProviderId,'codex');
  assert.deepEqual([...PROVIDER_CONTRACT_IDS.states],['LOCAL_CANDIDATE_READY',
    'LOCAL_ATTEMPT_FAILED','LOCAL_ATTEMPT_UNKNOWN','LOCAL_ATTEMPT_CANCELLED',
    'AGENT_NOT_EXECUTED','NOT_ADMITTED']);
  assert.ok(Object.isFrozen(PROVIDER_CONTRACT_IDS));
});

test('valid attempt input normalizes optional fields',()=>{
  const checked=validateProviderAttemptInput(input);
  assert.equal(checked.valid,true);
  assert.equal(checked.value.issueRef,input.issueRef);
  assert.deepEqual(checked.value.providerConfig,{cliPath:'C:\\cli\\x.mjs'});
  assert.equal(checked.value.signal,undefined);
});

test('attempt input rejects bad provider, sha, ids, write set and prompt',()=>{
  assert.equal(validateProviderAttemptInput({...input,providerId:'Bad_Id'}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,sourceSha:'ABC'}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,attemptId:'has space'}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,taskId:''}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,allowedWriteSet:[]}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,allowedWriteSet:['../escape.md']}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,allowedWriteSet:['C:\\abs\\path']}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,taskPrompt:'   '}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,signal:{}}).valid,false);
  assert.equal(validateProviderAttemptInput({...input,timeoutMs:0}).valid,false);
});

test('valid outcome passes; evidence-only guard rejects PASS grants',()=>{
  assert.equal(validateProviderAttemptOutcome(outcome).valid,true);
  assert.equal(validateProviderAttemptOutcome({...outcome,taskPass:true}).reason,'OUTCOME_GRANTS_PASS');
  assert.equal(validateProviderAttemptOutcome({...outcome,releasePass:true}).reason,'OUTCOME_GRANTS_PASS');
  assert.equal(validateProviderAttemptOutcome({...outcome,state:'MAGIC_PASS'}).valid,false);
  assert.equal(validateProviderAttemptOutcome({...outcome,process:{status:'FLYING'}}).valid,false);
  assert.equal(validateProviderAttemptOutcome({...outcome,candidateCommit:'xyz'}).valid,false);
  assert.equal(validateProviderAttemptOutcome({...outcome,candidateRef:'heads/main'}).valid,false);
  assert.equal(validateProviderAttemptOutcome(null).valid,false);
});

test('interface conformance detects missing members',()=>{
  const good=Object.freeze({providerId:'zcode',probe(){},execute(){},cancel(){},readback(){}});
  assert.deepEqual(assertProviderInterface(good),{conformant:true,missing:[]});
  const bad={providerId:'zcode',probe(){},execute(){}};
  const check=assertProviderInterface(bad);
  assert.equal(check.conformant,false);
  assert.deepEqual(check.missing,['cancel','readback','FROZEN_INTERFACE']);
});

test('selection: codex default, explicit optional, honest rejection',()=>{
  assert.deepEqual(resolveProviderSelection({}),{selected:'codex',admitted:true,
    route:'EXISTING_CHAIN',reason:'CODEX_DEFAULT',probe:null});
  assert.deepEqual(resolveProviderSelection({requested:'codex'}),{selected:'codex',
    admitted:true,route:'EXISTING_CHAIN',reason:'CODEX_EXPLICIT_DEFAULT',probe:null});
  assert.equal(resolveProviderSelection({requested:'zcode'}).reason,'PROVIDER_NOT_ENABLED');
  assert.equal(resolveProviderSelection({requested:'zcode',enabled:{zcode:true}}).reason,
    'PROVIDER_CAPABILITY_UNVERIFIED');
  const probe={available:true,provider:'zcode',version:'0.16.9'};
  const admitted=resolveProviderSelection({requested:'zcode',enabled:{zcode:true},probe});
  assert.equal(admitted.selected,'zcode');
  assert.equal(admitted.reason,'PROVIDER_VALIDATED');
  // Explicit but unvalidated is REJECTED, never silently switched to codex.
  const rejected=resolveProviderSelection({requested:'grok',enabled:{grok:true},
    probe:{available:false,reason:'LOCAL_AGENT_UNAVAILABLE'}});
  assert.equal(rejected.admitted,false);
  assert.equal(rejected.selected,null);
});
