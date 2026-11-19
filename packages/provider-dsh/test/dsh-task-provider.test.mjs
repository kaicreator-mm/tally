import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {spawn} from 'node:child_process';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createDshTaskProvider} from '../src/dsh-task-provider.mjs';
import {createDefaultDshOperations,LOCAL_DSH_CONSTANTS} from '../src/dsh-host.mjs';
import {resolveDshSelection} from '../src/dsh-config.mjs';
import {createMemoryCasStoreRegistry} from './memory-cas-store.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/64';
const builderSid='S-1-5-21-2839165489-1832998833-1574237796-1003';
const API_KEY='test-dsh-secret-0123456789abcdef';
const CANDIDATE='NOTES.md';
const candidateResult={writeSetValid:true,paths:[CANDIDATE],
  candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),
  candidateRef:LOCAL_DSH_CONSTANTS.candidateRefPrefix+'test'};
const answerBehavior=(close,child)=>{
  child.stdout.emit('data',Buffer.from('FAKE_DSH_FINAL_ANSWER: ok\n'));close(0,null);};

function fakeChild(behavior){
  const child=new EventEmitter();child.pid=4242;
  let closed=false;
  const doClose=()=>{if(!closed){closed=true;
    setImmediate(()=>child.emit('close',0,null));}};
  child.stdout=new EventEmitter();child.stderr=new EventEmitter();
  child.kill=()=>{doClose();return true;};
  child.stdin={end:()=>setImmediate(()=>behavior?.(doClose,child))};
  return child;
}
function operations(overrides={}){
  return {
    probeProvider:async()=>({available:true,provider:'dsh'}),
    prepareWorktree:async config=>({worktreePath:config.worktreePath,
      baseSha:config.sourceSha,dshHome:join(config.repositoryPath,'dsh-home'),
      patchPath:join(config.repositoryPath,'dsh-home','patch.yml')}),
    proveConfinement:async config=>({verified:true,
      builderPrincipal:config.builderPrincipal,builderSid:config.builderSid,
      hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'}),
    finalizeCandidate:async()=>candidateResult,
    verifyCandidateReadback:async(config,candidate)=>({verified:true,
      candidateTree:candidate.candidateTree,parent:config.sourceSha,paths:[CANDIDATE]}),
    ...overrides};
}
function providerConfig(dir,overrides={}){
  return {binPath:'C:\\Public\\dsh.exe',permissionMode:'workspace-write',
    pinnedVersion:'0.2.2-rc.5',deepseekApiKey:API_KEY,builderSid,
    builderPrincipal:'nilar\\codexsandboxoffline',
    verifierSid:'S-1-5-21-2839165489-1832998833-1574237796-1004',
    repositoryPath:dir,hostStorePath:join(dir,'host.sqlite'),
    ...overrides};
}
function unifiedInput(dir,overrides={}){
  return {providerId:'dsh',taskId:'task-64-r105',attemptId:'attempt:dsh:facade:1',
    sourceSha,worktreePath:join(dir,'candidate'),allowedWriteSet:[CANDIDATE],
    taskPrompt:'write the candidate note',issueRef,
    hostAuthorization:{claimKey:'kaicreator-mm/tally#64:builder:__default__',
      dispatchId:'dispatch:64:1',admissionGeneration:1},
    providerConfig:providerConfig(dir),...overrides};
}
async function withProvider(fn,{ops=operations(),behavior=answerBehavior}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-facade-'));
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;};
  const provider=createDshTaskProvider({spawnFn,operations:ops,
    now:()=>Date.UTC(2026,10,19),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return true;},
    openStore:createMemoryCasStoreRegistry().open});
  try{return await fn({provider,dir,calls,kills,input:unifiedInput(dir)});}
  finally{await rm(dir,{recursive:true,force:true});}
}
const noAuthority=o=>{assert.equal(o.taskPass,false);assert.equal(o.releasePass,false);
  assert.equal(o.authorized,false);};

test('facade exposes exactly the five frozen R105 members with providerId dsh',()=>{
  assert.throws(()=>createDshTaskProvider({}),
    e=>e.code==='DSH_HOST_STORE_FACTORY_REQUIRED');
  const provider=createDshTaskProvider({openStore:createMemoryCasStoreRegistry().open});
  assert.equal(provider.providerId,'dsh');
  assert.deepEqual(Object.keys(provider),['providerId','probe','execute','cancel','readback']);
  assert.equal(Object.isFrozen(provider),true);
});

test('execute maps the unified envelope onto runAttempt and back',async()=>{
  await withProvider(async({provider,calls,input})=>{
    const outcome=await provider.execute(input);
    assert.equal(outcome.provider,'dsh');assert.equal(outcome.providerId,'dsh');
    assert.equal(outcome.taskId,'task-64-r105');
    assert.equal(outcome.attemptId,'attempt:dsh:facade:1');
    assert.equal(outcome.sourceSha,sourceSha);
    assert.equal(outcome.state,'LOCAL_CANDIDATE_READY');
    assert.equal(outcome.reason,'REAL_DSH_CANDIDATE_RECORDED');
    assert.equal(outcome.agentLaunch,true);assert.equal(outcome.replayFixture,false);
    assert.equal(outcome.candidateTree,candidateResult.candidateTree);
    assert.equal(outcome.candidateRef,candidateResult.candidateRef);
    assert.deepEqual(outcome.candidatePaths,[CANDIDATE]);
    assert.deepEqual(outcome.hostAuthorization,input.hostAuthorization);
    noAuthority(outcome);
    // providerConfig was spread into the host runAttempt config.
    const exec=calls.at(-1);
    assert.equal(exec.options.env.DEEPSEEK_API_KEY,API_KEY);
    assert.equal(exec.options.cwd,input.worktreePath);
  });
});

test('execute refuses envelope violations before any spawn',async()=>{
  const cases=[
    [{providerId:'codex'},'DSH_PROVIDER_ID_MISMATCH'],
    [{allowedWriteSet:[]},'DSH_WRITESET_SINGLE_CANDIDATE_REQUIRED'],
    [{allowedWriteSet:['A.md','B.md']},'DSH_WRITESET_SINGLE_CANDIDATE_REQUIRED'],
    [{allowedWriteSet:undefined},'DSH_WRITESET_SINGLE_CANDIDATE_REQUIRED']];
  for(const [overrides,reason] of cases)await withProvider(async({provider,calls,input})=>{
    const outcome=await provider.execute({...input,...overrides});
    assert.equal(outcome.state,'AGENT_NOT_EXECUTED');
    assert.equal(outcome.reason,reason);
    assert.equal(outcome.agentLaunch,false);assert.equal(outcome.process,null);
    noAuthority(outcome);
    assert.equal(calls.length,0,'never spawned');
  });
  await withProvider(async({provider,input})=>{
    const outcome=await provider.execute({...input,
      providerConfig:{candidateRelativePath:'OTHER.md'}});
    assert.equal(outcome.reason,'DSH_WRITESET_SINGLE_CANDIDATE_REQUIRED');
    noAuthority(outcome);
  });
});

test('a pre-aborted cancel channel refuses execution without spawning',async()=>{
  await withProvider(async({provider,calls,input})=>{
    const controller=new AbortController();controller.abort();
    const outcome=await provider.execute({...input,signal:controller.signal});
    assert.equal(outcome.state,'AGENT_NOT_EXECUTED');
    assert.equal(outcome.reason,'DSH_SIGNAL_PRE_ABORTED');
    assert.equal(outcome.agentLaunch,false);
    assert.equal(calls.length,0);
    noAuthority(outcome);
  });
});

test('host exceptions become honest LOCAL_ATTEMPT_UNKNOWN with the error code',async()=>{
  await withProvider(async({provider,input})=>{
    const outcome=await provider.execute(input);
    assert.equal(outcome.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(outcome.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(outcome.agentLaunch,true);
    assert.equal(outcome.error,'CANDIDATE_WORKTREE_UNAVAILABLE');
    assert.equal(outcome.process.status,'UNKNOWN');
    noAuthority(outcome);
  },{ops:operations({prepareWorktree:async()=>{
    throw Object.assign(new Error('x'),{code:'CANDIDATE_WORKTREE_UNAVAILABLE'});}})});
});

test('cancel works only through the execute-time signal: three states',async()=>{
  await withProvider(async({provider,input})=>{
    assert.equal(provider.cancel({}),false,'missing signal ⇒ false');
    assert.equal(provider.cancel({signal:new AbortController().signal}),false,
      'signal with no live attempt ⇒ false');
    const controller=new AbortController();
    const pending=provider.execute({...input,signal:controller.signal,
      attemptId:'attempt:dsh:cancel:1'});
    await new Promise(resolve=>setImmediate(resolve));
    assert.equal(provider.cancel({signal:controller.signal}),true,
      'live, not-yet-aborted ⇒ true');
    assert.equal(provider.cancel({signal:controller.signal}),false,
      'already aborted ⇒ false');
    const outcome=await pending;
    assert.equal(outcome.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(outcome.reason,'ATTEMPT_CANCELLED');
    // dsh SIGTERM drains gracefully (exit 0): classification comes from the
    // adapter's own cancel intent, never from the exit code.
    assert.equal(outcome.process.exitCode,0);
    noAuthority(outcome);
  },{behavior:()=>{}});
});

test('probe maps the three tiers onto the host probe result',async()=>{
  const tiersOf=async probeResult=>{let captured;
    await withProvider(async({provider,input})=>{
      captured=await provider.probe(input.providerConfig);},
      {ops:operations({probeProvider:async()=>probeResult})});
    return captured;};
  assert.deepEqual((await tiersOf({available:false,stage:'VERSION'})).tiers,
    {VERSION:'FAIL',CREDENTIALS:'NOT_RUN',CANARY:'NOT_RUN'});
  assert.deepEqual((await tiersOf({available:false,stage:'AUTH'})).tiers,
    {VERSION:'PASS',CREDENTIALS:'FAIL',CANARY:'NOT_RUN'});
  assert.deepEqual((await tiersOf({available:false,stage:'CANARY'})).tiers,
    {VERSION:'PASS',CREDENTIALS:'PASS',CANARY:'FAIL'});
  const ok=await tiersOf({available:true,provider:'dsh'});
  assert.deepEqual(ok.tiers,{VERSION:'PASS',CREDENTIALS:'PASS',CANARY:'PASS'});
  assert.equal(ok.available,true);assert.equal(ok.providerId,'dsh');
});

test('probe runs the real three-tier gate against the fake CLI',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-facade-probe-'));
  try{
    const provider=createDshTaskProvider({spawnFn:(...a)=>spawn(...a),
      openStore:createMemoryCasStoreRegistry().open});
    const result=await provider.probe({binPath:process.execPath,
      binArgsPrefix:[fileURLToPath(new URL('./fake-dsh.mjs',import.meta.url)),
        '--fake-version','0.2.2-rc.5'],
      permissionMode:'workspace-write',deepseekApiKey:API_KEY});
    assert.equal(result.available,true);
    assert.equal(result.canaryVerified,true);
    assert.deepEqual(result.tiers,{VERSION:'PASS',CREDENTIALS:'PASS',CANARY:'PASS'});
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('readback re-derives the candidate and gates on full parent/tree/paths match',async()=>{
  await withProvider(async({provider,input})=>{
    const candidate={candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40)};
    const config={...input.providerConfig,candidateRelativePath:CANDIDATE,
      sourceSha};
    const good=await provider.readback(config,candidate);
    assert.equal(good.verified,true);assert.equal(good.ready,true);
    assert.equal(good.parent,sourceSha);
    for(const [opsOverride,label] of [
      [{candidateTree:'c'.repeat(40)},'tree mismatch'],
      [{parent:'d'.repeat(40)},'parent mismatch'],
      [{paths:['OTHER.md']},'path mismatch'],
      [{paths:['NOTES.md','EXTRA.txt']},'multi-path']]){
      const ops=operations({verifyCandidateReadback:async(config,candidate)=>
        ({verified:true,candidateTree:candidate.candidateTree,parent:config.sourceSha,
          paths:[CANDIDATE],...opsOverride})});
      const dir2=await mkdtemp(join(tmpdir(),'tally-dsh-facade-rb-'));
      const p2=createDshTaskProvider({spawnFn:()=>fakeChild(answerBehavior),
        operations:ops,openStore:createMemoryCasStoreRegistry().open});
      const bad=await p2.readback(config,candidate);
      assert.equal(bad.ready,false,label);
      await rm(dir2,{recursive:true,force:true});
    }
  },{ops:operations({})});
});

test('readback without an independent readback port fails closed',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-facade-rb2-'));
  const {verifyCandidateReadback:_,...noReadbackOps}=operations();
  const provider=createDshTaskProvider({spawnFn:()=>fakeChild(answerBehavior),
    operations:noReadbackOps,openStore:createMemoryCasStoreRegistry().open});
  const result=await provider.readback({sourceSha},{candidateTree:'a'.repeat(40)});
  assert.equal(result.verified,false);assert.equal(result.ready,false);
  assert.equal(result.reason,'DSH_READBACK_UNAVAILABLE');
  await rm(dir,{recursive:true,force:true});
});

test('selection stays Host-side: resolveDshSelection default-off semantics reused',()=>{
  assert.deepEqual(resolveDshSelection({}),{selected:false,providerId:'dsh',
    reason:'NOT_SELECTED_DEFAULT_OFF'});
  assert.deepEqual(resolveDshSelection({TALLY_ATTEMPT_PROVIDER:'dsh'}),
    {selected:false,providerId:'dsh',reason:'EXPLICIT_ENABLE_GATE_MISSING'});
  assert.deepEqual(resolveDshSelection({TALLY_ATTEMPT_PROVIDER:'dsh',
    TALLY_DSH_ENABLED:'1'}),{selected:true,providerId:'dsh',reason:'SELECTED'});
});
