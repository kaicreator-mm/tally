import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdtemp,rm,readFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {defineAgentProvider} from '../../tally-agent-provider/src/index.mjs';
import {createDshAgentProvider} from '../src/dsh-adapter.mjs';
import {PINNED_DSH_VERSION} from '../src/dsh-config.mjs';

const API_KEY='test-dsh-secret-0123456789abcdef';
const FAKE_DSH=fileURLToPath(new URL('./fake-dsh.mjs',import.meta.url));
const sha='a'.repeat(40);
const ANSWER='FAKE_DSH_FINAL_ANSWER: task complete.\n';

/** Ported from #467 dsh-host.test: fake child for injected-spawn negative cases. */
function fakeChild(behavior,{killMode='force'}={}){
  const child=new EventEmitter();child.pid=4242;
  let closed=false,exitCode=null,signal=null;
  const doClose=(code,signalArg)=>{if(!closed){closed=true;
    const finalCode=code!==undefined?code:exitCode;
    const finalSignal=signalArg!==undefined?signalArg:signal;
    setImmediate(()=>child.emit('close',finalCode,finalSignal));}};
  child.stdout=new EventEmitter();child.stderr=new EventEmitter();
  for(const stream of [child.stdout,child.stderr]){
    stream.destroy=()=>{stream.__destroyed=true;doClose();};
  }
  child.on('exit',(code,sig)=>{exitCode=code;signal=sig;});
  child.kill=()=>{
    if(killMode==='drain'){
      // Real 0.2.x CLI behavior: SIGTERM drains gracefully and exits 0.
      setImmediate(()=>{child.emit('exit',0,null);doClose(0,null);});
    }else{doClose();}
    return true;
  };
  child.stdin={end:()=>setImmediate(()=>behavior?.(doClose,child))};
  return child;
}

/**
 * Injected-spawn harness. `--version` spawns answer the pinned version so
 * execute/probe reach the stage under test; every other spawn (the headless
 * turn) gets `behavior` and `killMode`.
 */
function injectionHarness(behavior,{versionLine=PINNED_DSH_VERSION,killMode='force'}={}){
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});
    if(args.includes('--version')){
      const child=fakeChild(close=>{
        child.stdout.emit('data',Buffer.from(versionLine+'\n'));close(0,null);});
      return child;
    }
    lastChild=fakeChild(behavior,{killMode});return lastChild;
  };
  return {spawnFn,calls,kills,
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return true;},
    lastChildRef:()=>lastChild,
    headlessCalls:()=>calls.filter(c=>c.args.includes('headless'))};
}

/** Real-spawn spec against the ported fake CLI. */
const realSpec=overrides=>({binPath:process.execPath,
  binArgsPrefix:[FAKE_DSH,'--fake-version',PINNED_DSH_VERSION],
  timeoutMs:30000,permissionMode:'workspace-write',...overrides});

const requestFor=dir=>({taskId:'264',attemptId:'attempt:264:dsh:1',
  sourceSha:sha,prompt:'Work only in the current disposable workspace.',
  worktreePath:dir});
const contextFor=()=>({env:{DEEPSEEK_API_KEY:API_KEY}});
const serialized=o=>JSON.stringify(o);

/**
 * Drain-like kill for the SIGTERM-exit-0 trap scenarios: the real 0.2.x CLI
 * drains gracefully and exits 0, so `kill()` must resolve as exit(0) ->
 * close(0), never as a forced null exit. Built into the fake child at
 * construction time: a pre-aborted signal terminates the tree before any
 * stdin-time behavior could swap `kill`.
 */
const drainKill={killMode:'drain'};

test('probe: pinned CLI + credential -> AVAILABLE; no key material, canary NOT_RUN',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-probe-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},config:realSpec()});
    const probe=await provider.probe(contextFor());
    assert.equal(probe.state,'AVAILABLE');assert.equal(probe.reason,null);
    assert.equal(probe.evidence.providerVersion,PINNED_DSH_VERSION);
    assert.equal(probe.evidence.pinnedVersion,PINNED_DSH_VERSION);
    assert.equal(probe.evidence.binSource,'config');
    assert.equal(probe.evidence.canary,'NOT_RUN',
      'the probe makes NO real DSH turn by default');
    assert.equal(serialized(probe).includes(API_KEY),false);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('probe: version drift refuses (fail-closed pin)',async()=>{
  const provider=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},config:realSpec({
      binArgsPrefix:[FAKE_DSH,'--fake-version','9.9.9-x']})});
  const probe=await provider.probe(contextFor());
  assert.equal(probe.state,'UNAVAILABLE');
  assert.equal(probe.reason,'LOCAL_AGENT_UNAVAILABLE');
  assert.equal(probe.evidence.stage,'VERSION');
  assert.equal(probe.evidence.observedVersion,'9.9.9-x');
  assert.equal(probe.evidence.pinnedVersion,PINNED_DSH_VERSION);
});

test('probe: missing binPath refuses before any spawn',async()=>{
  const harness=injectionHarness(()=>{});
  const provider=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
    config:{binPath:'C:\\Nonexistent\\dsh.exe'}});
  const probe=await provider.probe(contextFor());
  assert.equal(probe.state,'UNAVAILABLE');
  assert.equal(probe.reason,'LOCAL_AGENT_UNAVAILABLE');
  assert.equal(probe.evidence.stage,'VERSION');
  assert.equal(probe.evidence.detail,'DSH_BIN_PATH_NOT_FOUND');
  assert.equal(harness.calls.length,0,'no spawn attempt for a missing binary');
});

test('probe: missing credential refuses and never spawns a headless turn',async()=>{
  const harness=injectionHarness(()=>{});
  const provider=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
    config:realSpec()});
  const probe=await provider.probe({env:{}});
  assert.equal(probe.state,'UNAVAILABLE');
  assert.equal(probe.reason,'LOCAL_AGENT_UNAVAILABLE');
  assert.equal(probe.evidence.stage,'CREDENTIALS');
  assert.equal(probe.evidence.credentialEnv,'DEEPSEEK_API_KEY');
  assert.equal(probe.evidence.credentialPresent,false);
  assert.equal(harness.headlessCalls().length,0);
  assert.equal(harness.calls.length,1,'only the --version probe spawned');
});

test('defense-in-depth: a half-open legacy env gate closes the adapter',async()=>{
  const harness=injectionHarness(()=>{});
  const provider=await createDshAgentProvider({
    defineAgentProvider,
    hostEnv:{TALLY_ATTEMPT_PROVIDER:'dsh'},
    spawnFn:harness.spawnFn,config:realSpec()});
  const probe=await provider.probe(contextFor());
  assert.equal(probe.state,'UNAVAILABLE');
  assert.equal(probe.reason,'DSH_ENV_GATE_CLOSED');
  assert.equal(harness.calls.length,0);
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-gate-'));
  try{
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'DSH_ENV_GATE_CLOSED');
    assert.equal(harness.headlessCalls().length,0);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('defense-in-depth: the legacy two-gate fully open does not block a registry-enabled adapter',async()=>{
  const provider=await createDshAgentProvider({
    defineAgentProvider,
    hostEnv:{TALLY_ATTEMPT_PROVIDER:'dsh',TALLY_DSH_ENABLED:'1'},
    config:realSpec()});
  const probe=await provider.probe(contextFor());
  assert.equal(probe.state,'AVAILABLE');
});

test('execute happy path: one-shot headless turn, isolated env, honest evidence',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-happy-'));
  const recordPath=join(dir,'fake-record.jsonl');
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},config:realSpec({
        binArgsPrefix:[FAKE_DSH,'--fake-scenario','answer',
          '--fake-record',recordPath]})});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'COMPLETED');
    assert.equal(x.reason,'REAL_DSH_ATTEMPT_COMPLETED');
    assert.equal(x.evidence.exitCode,0);
    assert.equal(x.evidence.stdoutSha256,
      createHash('sha256').update(ANSWER).digest('hex'));
    assert.equal(x.evidence.stdoutBytes,Buffer.byteLength(ANSWER));
    assert.ok(x.evidence.stdoutTail.includes('FAKE_DSH_FINAL_ANSWER'));
    assert.equal(serialized(x).includes(API_KEY),false,
      'credential value never reaches observations');
    // The fake CLI observed the real spawn contract.
    const lines=(await readFile(recordPath,'utf8')).trim().split(/\r?\n/);
    assert.equal(lines.length,2,'exactly one --version probe + one headless turn');
    const line=JSON.parse(lines.at(-1));
    assert.equal(line.cwd,dir,'workspace root = process cwd = Host worktree');
    assert.ok(line.env.DSH_HOME&&line.env.DSH_HOME.includes('tally-dsh-home-'),
      'per-attempt throwaway DSH_HOME');
    assert.equal(line.env.DSH_TELEMETRY_MODE,'DISABLED');
    assert.equal(line.env.DSH_PERMISSION_MODE,'workspace-write');
    assert.equal(line.env.DEEPSEEK_API_KEY_present,true);
    assert.deepEqual(line.argv.slice(-5),
      ['--profile','headless','--patch',
        join(line.env.DSH_HOME,'tally-dsh.patch.yml'),requestFor(dir).prompt],
      'argv carries the headless profile, the overlay patch and the prompt');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('execute: nonzero exit is FAILED with redacted stderr evidence',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-exit-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},config:realSpec({
        binArgsPrefix:[FAKE_DSH,'--fake-scenario','exit7']})});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.evidence.exitCode,7);
    assert.ok(x.evidence.stderrTail.includes('simulated harness failure'));
    assert.equal(serialized(x).includes(API_KEY),false);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('execute: leaked key in stderr is redacted in evidence',async()=>{
  const harness=injectionHarness((close,child)=>{
    child.stderr.emit('data',Buffer.from(
      `dsh: MISSING_CREDENTIAL leaked=${API_KEY}\n`));close(1,null);});
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-redact-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
      killTree:harness.killTree,config:realSpec()});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.ok(x.evidence.stderrTail.includes('MISSING_CREDENTIAL'));
    assert.ok(x.evidence.stderrTail.includes('[REDACTED]'));
    assert.equal(x.evidence.stderrTail.includes(API_KEY),false);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('execute: exit 0 with an empty final answer fails safe',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-empty-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},config:realSpec({
        binArgsPrefix:[FAKE_DSH,'--fake-scenario','empty']})});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'DSH_EMPTY_FINAL_ANSWER');
    assert.equal(x.evidence.exitCode,0);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('SIGTERM-exit-0 trap: a pre-aborted signal classifies CANCELLED from intent, never from the graceful exit 0',async()=>{
  // The fake child drains like the real 0.2.x CLI: kill -> exit(0) -> close 0.
  const harness=injectionHarness(()=>{},drainKill);
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-abort-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
      killTree:harness.killTree,config:realSpec()});
    const controller=new AbortController();controller.abort();
    const x=await provider.execute(requestFor(dir),
      {...contextFor(),signal:controller.signal});
    assert.equal(x.state,'CANCELLED',
      'exit code 0 must NOT become COMPLETED once the adapter intended cancel');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.equal(x.evidence.exitCode,0,
      'direct evidence of the trap: graceful drain exits 0');
    assert.equal(x.evidence.cancelled,true);
    assert.ok(harness.kills.includes(4242),'the process tree was terminated');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('cancel(): running attempt -> CANCEL_REQUESTED then CANCELLED despite exit 0',async()=>{
  const harness=injectionHarness(()=>{},drainKill);
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-cancel-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
      killTree:harness.killTree,config:realSpec()});
    const promise=provider.execute(requestFor(dir),contextFor());
    let cancelResult=null;
    for(let i=0;i<200;i++){
      await new Promise(r=>setTimeout(r,2));
      cancelResult=await provider.cancel({taskId:'264',
        attemptId:'attempt:264:dsh:1',sourceSha:sha,providerId:'dsh'});
      if(cancelResult.state==='CANCEL_REQUESTED')break;
    }
    assert.equal(cancelResult?.state,'CANCEL_REQUESTED');
    assert.equal(cancelResult.reason,'ATTEMPT_CANCEL_REQUESTED');
    const x=await promise;
    assert.equal(x.state,'CANCELLED');
    assert.equal(x.evidence.exitCode,0);
    assert.ok(harness.kills.includes(4242));
    const after=await provider.cancel({taskId:'264',
      attemptId:'attempt:264:dsh:1',sourceSha:sha,providerId:'dsh'});
    assert.equal(after.state,'NOT_RUNNING');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('execute: timeout terminates the tree and reports UNKNOWN with a reason',async()=>{
  const harness=injectionHarness(()=>{}); // never closes on its own
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-timeout-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
      killTree:harness.killTree,config:realSpec()});
    const x=await provider.execute({...requestFor(dir),timeoutMs:30},
      contextFor());
    assert.equal(x.state,'UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.equal(x.evidence.timedOut,true);
    assert.equal(x.evidence.exitCode,null);
    assert.ok(harness.kills.includes(4242),'the process tree was terminated');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('output-pipe non-convergence is wired: grace destroys pipes and fails closed',async()=>{
  const harness=injectionHarness((_close,child)=>{child.emit('exit',0,null);});
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-orphan-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
      killTree:harness.killTree,config:realSpec({pipeGraceMs:40})});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'DSH_OUTPUT_PIPE_NOT_CONVERGED');
    assert.equal(x.evidence.orphanPipe,true);
    assert.equal(harness.lastChildRef().stdout.__destroyed,true,
      'the declared pipe-grace mechanism must actually be wired');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real hang: timeout kills the real process tree and reports durable UNKNOWN',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-hang-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},config:realSpec({
        binArgsPrefix:[FAKE_DSH,'--fake-scenario','hang'],
        timeoutMs:500,pipeGraceMs:60000})});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(x.evidence.exitCode===null||x.evidence.exitCode===1,
      'Windows taskkill /F reports exit code 1; POSIX group kill reports null');
    assert.equal(x.evidence.timedOut,true);
    const again=await provider.execute(requestFor(dir),contextFor());
    assert.equal(again.state,'UNKNOWN',
      're-running after UNKNOWN is honest, never a guessed completion');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('reconcile: in-flight attempt RUNNING; finished attempt honest UNKNOWN',async()=>{
  const harness=injectionHarness(()=>{});
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-reconcile-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
      killTree:harness.killTree,config:realSpec({timeoutMs:5000})});
    const promise=provider.execute(requestFor(dir),contextFor());
    let seen=null;
    for(let i=0;i<200;i++){
      await new Promise(r=>setTimeout(r,2));
      seen=await provider.reconcile({taskId:'264',
        attemptId:'attempt:264:dsh:1',sourceSha:sha,providerId:'dsh'});
      if(seen.state==='RUNNING')break;
    }
    assert.equal(seen?.state,'RUNNING');
    assert.equal(seen.reason,'ATTEMPT_IN_FLIGHT');
    await provider.cancel({taskId:'264',attemptId:'attempt:264:dsh:1',
      sourceSha:sha,providerId:'dsh'});
    const x=await promise;
    assert.equal(x.state,'CANCELLED');
    const after=await provider.reconcile({taskId:'264',
      attemptId:'attempt:264:dsh:1',sourceSha:sha,providerId:'dsh'});
    assert.equal(after.state,'UNKNOWN');
    assert.equal(after.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(after.evidence.durableJournal,false);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('execute: missing Host workspace refuses before any spawn',async()=>{
  const harness=injectionHarness(()=>{});
  const provider=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
    config:realSpec()});
  const x=await provider.execute({taskId:'264',attemptId:'attempt:264:dsh:1',
    sourceSha:sha,prompt:'Work only in the current disposable workspace.',
    worktreePath:'C:\\definitely\\missing\\workspace'},contextFor());
  assert.equal(x.state,'FAILED');
  assert.equal(x.reason,'PROVIDER_WORKSPACE_UNAVAILABLE');
  assert.equal(x.evidence.spawned,false);
  assert.equal(harness.calls.length,0);
});

test('execute: malformed bindings throw TypeError at the contract boundary; adapter-layer refusals stay FAILED DSH_CONFIG_INVALID, no spawn',async()=>{
  const harness=injectionHarness(()=>{});
  const provider=await createDshAgentProvider({
    defineAgentProvider,hostEnv:{},spawnFn:harness.spawnFn,
    config:realSpec()});
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-invalid-'));
  try{
    // Boundary-rejected bindings: the hardened defineAgentProvider wrapper
    // throws before the adapter ever sees the request.
    for(const [bad,message] of [
      [{...requestFor(dir),attemptId:''},'PROVIDER_ATTEMPT_ID_INVALID'],
      [{...requestFor(dir),sourceSha:'not-a-sha'},'PROVIDER_SOURCE_SHA_INVALID'],
      [{...requestFor(dir),taskId:'bad task id with spaces'},
        'PROVIDER_TASK_ID_INVALID'],
    ]){
      await assert.rejects(()=>provider.execute(bad,contextFor()),
        {name:'TypeError',message});
    }
    // An empty prompt is a legal string at the boundary, so it reaches the
    // adapter layer, which still owns this refusal as a FAILED observation.
    const x=await provider.execute({...requestFor(dir),prompt:''},contextFor());
    assert.equal(x.state,'FAILED');assert.equal(x.reason,'DSH_CONFIG_INVALID');
    assert.equal(harness.calls.length,0);
  }finally{await rm(dir,{recursive:true,force:true});}
  for(const mode of ['total-control','danger-full-access']){
    const gated=injectionHarness(()=>{});
    const p=await createDshAgentProvider({defineAgentProvider,hostEnv:{},
      spawnFn:gated.spawnFn,
      config:realSpec({permissionMode:mode})});
    const x=await p.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'FAILED');assert.equal(x.reason,'DSH_CONFIG_INVALID');
    assert.equal(gated.calls.length,0);
  }
});

test('execute: danger-full-access with explicit allowance is honored (ported #467 behavior)',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-dfa-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,hostEnv:{},config:realSpec({
        permissionMode:'danger-full-access',allowDangerousFullAccess:true})});
    const x=await provider.execute(requestFor(dir),contextFor());
    assert.equal(x.state,'COMPLETED');
    const recordPath=join(dir,'fake-record.jsonl');
    const p2=await createDshAgentProvider({defineAgentProvider,hostEnv:{},
      config:realSpec({permissionMode:'danger-full-access',
        allowDangerousFullAccess:true,
        binArgsPrefix:[FAKE_DSH,'--fake-scenario','answer',
          '--fake-record',recordPath]})});
    const y=await p2.execute(requestFor(dir),contextFor());
    assert.equal(y.state,'COMPLETED');
    const line=JSON.parse((await readFile(recordPath,'utf8')).trim()
      .split(/\r?\n/).at(-1));
    assert.equal(line.env.DSH_PERMISSION_MODE,'danger-full-access');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('all observation surfaces stay free of credential material',async()=>{
  const surfaces=[];
  const leaky=injectionHarness((close,child)=>{
    child.stderr.emit('data',Buffer.from(`boom ${API_KEY}`));close(1,null);});
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-secrets-'));
  try{
    const failed=await createDshAgentProvider({defineAgentProvider,hostEnv:{},
      spawnFn:leaky.spawnFn,killTree:leaky.killTree,config:realSpec()});
    surfaces.push(await failed.execute(requestFor(dir),contextFor()));
    const trap=injectionHarness((close,child)=>{
      setImmediate(()=>child.emit('exit',0,null));});
    const cancelled=await createDshAgentProvider({defineAgentProvider,
      hostEnv:{},spawnFn:trap.spawnFn,killTree:trap.killTree,
      config:realSpec()});
    const controller=new AbortController();controller.abort();
    surfaces.push(await cancelled.execute(requestFor(dir),
      {...contextFor(),signal:controller.signal}));
    const hang=injectionHarness(()=>{});
    const unknown=await createDshAgentProvider({defineAgentProvider,
      hostEnv:{},spawnFn:hang.spawnFn,killTree:hang.killTree,
      config:realSpec()});
    surfaces.push(await unknown.execute({...requestFor(dir),timeoutMs:30},
      contextFor()));
    const orphan=injectionHarness((_close,child)=>{child.emit('exit',0,null);});
    const orphaned=await createDshAgentProvider({defineAgentProvider,
      hostEnv:{},spawnFn:orphan.spawnFn,killTree:orphan.killTree,
      config:realSpec({pipeGraceMs:30})});
    surfaces.push(await orphaned.execute(requestFor(dir),contextFor()));
    for(const surface of surfaces){
      assert.equal(serialized(surface).includes(API_KEY),false,
        `${surface.state} evidence must not leak the credential`);
    }
    assert.deepEqual(surfaces.map(s=>s.state),
      ['FAILED','CANCELLED','UNKNOWN','FAILED']);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real spawn: adapter never reads process.env for credentials',async()=>{
  // hostEnv defaults to {}; even with a DEEPSEEK_API_KEY exported in the
  // test process, an execute WITHOUT a context credential must refuse at
  // the CREDENTIALS stage instead of silently inheriting the operator key.
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-adapter-noenv-'));
  try{
    const provider=await createDshAgentProvider({
      defineAgentProvider,config:realSpec()});
    const x=await provider.execute(requestFor(dir),
      {env:{DEEPSEEK_API_KEY:undefined}});
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.evidence.spawned,false);
  }finally{await rm(dir,{recursive:true,force:true});}
});
