import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {existsSync} from 'node:fs';
import {mkdtemp,rm,writeFile,chmod} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawn,spawnSync} from 'node:child_process';
import {createPiTaskProvider,PI_PROVIDER_CONSTANTS,
  PI_PROVIDER_CONSTANTS as CONSTANTS_CHECK,resolvePiProviderSelection,
  piBuilderEnvironment,piSpawnEnvironment,piTaskPrompt,piConfigToken,
  parsePiJsonStream} from './pi-provider.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/64';
const ALLOWED=['experiments/v2/selected-business-build/LOCAL_PI_L3.md'];
const candidate={writeSetValid:true,paths:ALLOWED,candidateTree:'a'.repeat(40),
  candidateCommit:'b'.repeat(40),candidateRef:'refs/tally-pi-candidates/test'};
function fakeChild(behavior){
  const child=new EventEmitter();child.pid=4242;child.stdout=new EventEmitter();child.stderr=new EventEmitter();
  let closed=false;
  const close=(code,signal)=>{if(!closed){closed=true;setImmediate(()=>child.emit('close',code,signal));}};
  child.kill=()=>{close(null,'SIGTERM');return true;};
  child.stdin={end:()=>setImmediate(()=>behavior?.(close,child))};
  return child;
}
function operations(overrides={}){
  return {
    probeProvider:async()=>({available:true,provider:'pi',version:'0.85.1'}),
    attemptRecord:async()=>null,
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha}),
    finalizeCandidate:async()=>candidate,...overrides
  };
}
function inputFor(dir,attemptId='attempt:pi:test:1'){
  return {providerId:'pi',taskId:'task-64',issueRef,sourceSha,attemptId,
    worktreePath:join(dir,'candidate'),allowedWriteSet:ALLOWED,
    taskPrompt:'Create the candidate note file.',
    hostAuthorization:{mode:'explicit',attemptId},
    providerConfig:{cliPath:process.execPath,repositoryPath:dir},
    timeoutMs:1000};
}
async function withHost(fn,{ops=operations(),behavior=(close)=>close(0,null),
  killTree=async()=>true}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-pi-provider-test-'));
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
  };
  const provider=createPiTaskProvider({spawnFn,operations:ops,now:()=>Date.UTC(2026,10,19),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return killTree(pid);}});
  try{return await fn({provider,dir,calls,kills,input:inputFor(dir)});}
  finally{await rm(dir,{recursive:true,force:true});}
}
function piNdjson(text,{session='f0000000-0000-4000-8000-000000000001',thinking=false,
  provider='fake-provider',model='fake-model'}={}){
  const content=thinking?
    [{type:'thinking',thinking:'the user wants OK'},{type:'text',text}]:
    [{type:'text',text}];
  return [
    JSON.stringify({type:'session',version:3,id:session,cwd:'C:/tmp'}),
    JSON.stringify({type:'agent_start'}),
    JSON.stringify({type:'message_end',message:{role:'assistant',content,
      provider,model,usage:{input:11,output:3,cacheRead:0},stopReason:'stop'}}),
    JSON.stringify({type:'agent_end'})
  ].join('\n')+'\n';
}
function probeSpawnFn(calls,{canaryBehavior='ok',canaryTimeoutStall=false,
  authBehavior='ready'}={}){
  return (command,args,options)=>{
    calls.push({command,args,options});
    return fakeChild((close,child)=>{
      if(args.includes('--version')){child.stdout.emit('data','0.85.1\n');close(0,null);}
      else if(args[0]==='auth'){
        const status=authBehavior==='ready'?'ready':'not_ready';
        child.stdout.emit('data',JSON.stringify({status,
          provider:'deepseek',authType:'api_key'})+'\n');close(0,null);}
      else if(args[0]==='-p'){
        if(canaryTimeoutStall)return; // never closes: bounded stall
        if(canaryBehavior==='ok'){
          child.stdout.emit('data',piNdjson('OK',{session:'sess_canary',thinking:true}));
          close(0,null);
        }else close(0,null); // exit 0 but no parseable NDJSON
      }else close(1,null);
    });
  };
}

test('provider exposes the unified frozen TaskProvider surface',()=>{
  const provider=createPiTaskProvider({spawnFn:()=>fakeChild(close=>close(0,null))});
  assert.equal(provider.providerId,'pi');
  assert.equal(typeof provider.probe,'function');
  assert.equal(typeof provider.execute,'function');
  assert.equal(typeof provider.cancel,'function');
  assert.equal(typeof provider.readback,'function');
  assert.ok(Object.isFrozen(provider));
});

test('probe reports available after version, credentials and canary stages',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-pi-probe-'));
  try{
    const calls=[];
    const provider=createPiTaskProvider({spawnFn:probeSpawnFn(calls)});
    const probe=await provider.probe({cliPath:process.execPath,provider:'deepseek',
      env:{DEEPSEEK_API_KEY:'test-key'}});
    assert.equal(probe.available,true);
    assert.equal(probe.provider,'pi');
    assert.equal(probe.version,'0.85.1');
    assert.equal(probe.canary.sessionHandle,'sess_canary');
    assert.equal(probe.canary.response,'OK');
    assert.equal(probe.canary.model,'fake-model');
    assert.equal(calls.length,3,'version + auth check + canary only');
    assert.deepEqual(calls[0].args,['--version']);
    assert.deepEqual(calls[1].args,['auth','check','--provider','deepseek',
      '--json','--no-refresh']);
    const canaryArgs=calls[2].args;
    assert.equal(canaryArgs[0],'-p');
    assert.equal(canaryArgs[canaryArgs.indexOf('--mode')+1],'json');
    assert.ok(canaryArgs.includes('--no-session'));
    assert.ok(typeof canaryArgs[canaryArgs.indexOf('--session-dir')+1]==='string'&&
      canaryArgs[canaryArgs.indexOf('--session-dir')+1].length>0);
    for(const flag of ['--no-extensions','--no-skills','--no-prompt-templates',
      '--no-themes','--no-context-files','--no-approve'])
      assert.ok(canaryArgs.includes(flag),flag);
    assert.equal(canaryArgs[canaryArgs.indexOf('--provider')+1],'deepseek');
    assert.equal(canaryArgs[canaryArgs.indexOf('--')+1],'Reply with exactly: OK');
    // The credential rides the environment, never argv.
    assert.ok(!canaryArgs.includes('test-key'));
    assert.equal(calls[2].options.env.DEEPSEEK_API_KEY,'test-key');
    assert.equal(calls[2].options.env.PI_OFFLINE,'1');
    assert.equal(calls[2].options.env.PI_TELEMETRY,'0');
    assert.equal(calls[2].options.env.HOME,calls[2].options.env.USERPROFILE);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('probe negatives are honest: missing cli, version mismatch, unconfigured provider, absent credentials, unsatisfied canary',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-pi-probe-neg-'));
  try{
    const calls=[];
    const provider=createPiTaskProvider({spawnFn:probeSpawnFn(calls,
      {canaryBehavior:'empty'})});
    const missing=await provider.probe({cliPath:join(dir,'absent.exe'),provider:'deepseek'});
    assert.equal(missing.available,false);assert.equal(missing.stage,'VERSION');
    assert.equal(missing.detail,'MISSING_CLI');assert.equal(missing.outputMatched,false);
    assert.equal(calls.length,0,'missing cli never spawns');
    const mismatch=await provider.probe({cliPath:process.execPath,provider:'deepseek',
      pinnedVersion:'9.9.9'});
    assert.equal(mismatch.available,false);assert.equal(mismatch.stage,'VERSION');
    assert.equal(mismatch.detail,'VERSION_MISMATCH');
    assert.equal(mismatch.observedVersion,'0.85.1');
    const unconfigured=await provider.probe({cliPath:process.execPath});
    assert.equal(unconfigured.available,false);assert.equal(unconfigured.stage,'CREDENTIALS');
    assert.equal(unconfigured.detail,'PROVIDER_UNCONFIGURED');
    const notReadyProvider=createPiTaskProvider({spawnFn:probeSpawnFn(calls,
      {canaryBehavior:'empty',authBehavior:'not-ready'})});
    const notReady=await notReadyProvider.probe({cliPath:process.execPath,
      provider:'deepseek',env:{DEEPSEEK_API_KEY:'k'}});
    assert.equal(notReady.available,false);assert.equal(notReady.stage,'CREDENTIALS');
    assert.equal(notReady.detail,'CREDENTIALS_ABSENT');
    const unsatisfied=await provider.probe({cliPath:process.execPath,provider:'deepseek',
      env:{DEEPSEEK_API_KEY:'k'}});
    assert.equal(unsatisfied.available,false);assert.equal(unsatisfied.stage,'CANARY');
    assert.equal(unsatisfied.detail,'CANARY_UNSATISFIED');
    assert.equal(unsatisfied.reason,'LOCAL_AGENT_UNAVAILABLE');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('a stalling canary is an honest unavailable probe, never a retry into PASS',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-pi-probe-stall-'));
  try{
    const calls=[];
    const provider=createPiTaskProvider({spawnFn:probeSpawnFn(calls,
      {canaryTimeoutStall:true})});
    const stalled=await provider.probe({cliPath:process.execPath,provider:'deepseek',
      env:{DEEPSEEK_API_KEY:'k'},canaryTimeoutMs:5});
    assert.equal(stalled.available,false);
    assert.equal(stalled.stage,'CANARY');
    assert.equal(stalled.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(stalled.detail,'CREDENTIALS_OR_PROVIDER_STALL');
    assert.equal(calls.length,3,'auth failures stall silently in the real CLI; one bounded canary, no retry');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('unavailable provider never spawns the task process',async()=>{
  await withHost(async({provider,calls,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.agentLaunch,false);assert.equal(calls.length,0);
    assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
    assert.equal(x.authorized,false);
    assert.equal(x.providerProbe.stage,'AUTH');
  },{ops:operations({probeProvider:async()=>({available:false,provider:'pi',
    reason:'LOCAL_AGENT_UNAVAILABLE',stage:'AUTH'})})});
});

test('real process exit produces a candidate under the full pi argv contract',async()=>{
  await withHost(async({provider,calls,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(x.reason,'REAL_PI_CANDIDATE_RECORDED');
    assert.equal(x.agentLaunch,true);assert.equal(x.replayFixture,false);
    assert.equal(x.candidateTree,candidate.candidateTree);
    assert.equal(x.candidateCommit,candidate.candidateCommit);
    assert.equal(x.candidateRef,candidate.candidateRef);
    assert.equal(x.agent.sessionHandle,'sess_task');
    assert.equal(x.agent.response,'Done. Candidate written.');
    assert.equal(x.agent.model,'fake-model');
    assert.equal(x.agent.provider,'fake-provider');
    assert.deepEqual(x.agent.usage,{inputTokens:11,outputTokens:3});
    assert.equal(x.providerId,'pi');assert.equal(x.provider,'pi');
    assert.equal(x.taskId,'task-64');assert.equal(x.attemptId,input.attemptId);
    assert.equal(x.sourceSha,sourceSha);assert.equal(x.issueRef,issueRef);
    assert.equal(calls.length,1);
    const args=calls[0].args;
    assert.equal(calls[0].command,input.providerConfig.cliPath,'pi binary launched directly');
    assert.equal(args[0],'-p');
    assert.equal(args[args.indexOf('--mode')+1],'json');
    for(const flag of ['--no-session','--no-extensions','--no-skills',
      '--no-prompt-templates','--no-themes','--no-context-files','--no-approve'])
      assert.ok(args.includes(flag),flag);
    assert.ok(args.includes('--session-dir'));
    assert.ok(!args.includes('--api-key'),'credentials never ride argv');
    assert.ok(!args.includes('--cwd'),'workspace rides spawn cwd, not a flag');
    const instruction=args[args.indexOf('--')+1];
    for(const path of ALLOWED)assert.ok(instruction.includes(path),`prompt carries ${path}`);
    assert.ok(instruction.includes('Do not run git commit, git push'));
    assert.ok(instruction.includes('Task: Create the candidate note file.'));
    assert.equal(calls[0].options.cwd,input.worktreePath);
    assert.equal(calls[0].options.env.PI_OFFLINE,'1');
    assert.equal(calls[0].options.env.PI_TELEMETRY,'0');
    assert.ok(calls[0].options.env.PI_CODING_AGENT_DIR);
    assert.ok(calls[0].options.env.HOME);
    assert.ok(calls[0].options.env.HOME!==process.env.USERPROFILE,
      'home is redirected away from the operator profile');
    assert.ok(!JSON.stringify(x).includes('WOODPECKER_TOKEN'));
    assert.equal(x.process.status,'EXITED');
    assert.equal(x.process.exitCode,0);
    assert.equal(x.process.pid,4242);
    assert.ok(x.process.startedAt);assert.ok(x.process.terminatedAt);
  },{behavior:(close,child)=>{
    child.stdout.emit('data',piNdjson('Done. Candidate written.',{session:'sess_task'}));
    close(0,null);}});
});

test('writes outside the allowed write set are a violation, never a candidate',async()=>{
  await withHost(async({provider,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'WRITE_SET_VIOLATION');
    assert.equal(x.writeSetDetail,'PATHS_OUTSIDE_ALLOWED_SET');
    assert.equal(x.candidateTree,null);assert.equal(x.candidateCommit,null);
    assert.equal(x.taskPass,false);
    assert.deepEqual(x.candidatePaths,['forbidden.txt','escape.md']);
  },{ops:operations({finalizeCandidate:async()=>({writeSetValid:false,
    paths:['forbidden.txt','escape.md'],detail:'PATHS_OUTSIDE_ALLOWED_SET'})})});
});

test('nonzero process exit is FAILED with the captured exit code',async()=>{
  await withHost(async({provider,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.process.exitCode,7);
    assert.equal(x.taskPass,false);
  },{behavior:close=>close(7,null)});
});

test('timeout terminates the process tree and records UNKNOWN',async()=>{
  await withHost(async({provider,kills,input})=>{
    const x=await provider.execute({...input,timeoutMs:25});
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.equal(x.process.status,'EXITED','the process itself did exit (terminated)');
    assert.equal(x.process.exitCode,null,'no exit code survives a tree kill');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('cancellation through the input signal kills the child and records CANCELLED',async()=>{
  await withHost(async({provider,kills,calls,input})=>{
    const controller=new AbortController();
    // Abort only AFTER the spawn is observed: a fixed 10ms timer races the
    // async prep chain (probe/worktree/session dirs) — when it fires before
    // the spawn the honest outcome is cancelled-without-spawn and no kill,
    // which made this test flake. Waiting for the spawn pins ONE timeline.
    const execute=provider.execute({...input,timeoutMs:1000,signal:controller.signal});
    while(calls.length===0)await new Promise(resolve=>setImmediate(resolve));
    controller.abort();
    const x=await execute;
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('a signal already aborted before launch cancels without spawning',async()=>{
  await withHost(async({provider,calls,input})=>{
    const controller=new AbortController();controller.abort();
    const x=await provider.execute({...input,timeoutMs:1000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.equal(x.agentLaunch,false);
    assert.equal(x.process,null);
    assert.equal(calls.length,0);
  });
});

test('cancel(input) is the signal-only channel and reports inactive attempts honestly',async()=>{
  await withHost(async({provider,kills,input})=>{
    const running=provider.execute({...input,timeoutMs:30000});
    await new Promise(resolve=>setTimeout(resolve,20));
    const outcome=await provider.cancel({attemptId:input.attemptId});
    assert.equal(outcome.cancelled,true);
    assert.equal(outcome.reason,'ATTEMPT_CANCELLED');
    const x=await running;
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));
    const none=await provider.cancel({attemptId:'attempt:pi:none:1'});
    assert.equal(none.cancelled,false);assert.equal(none.reason,'ATTEMPT_NOT_ACTIVE');
    const invalid=await provider.cancel({attemptId:':bad'});
    assert.equal(invalid.cancelled,false);assert.equal(invalid.reason,'CANCEL_INPUT_INVALID');
  },{behavior:()=>{}});
});

test('spawn error is UNKNOWN and an occupied worktree blocks blind relaunch',async()=>{
  await withHost(async({provider,calls,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(x.agentLaunch,true);
    assert.equal(x.process.status,'UNKNOWN');
    assert.equal(x.process.exitCode,null);
    assert.equal(calls.length,1);
  },{behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
  let records=0;
  await withHost(async({provider,calls,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(calls.length,1);
    const again=await provider.execute(input);
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(again.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(again.agentLaunch,false,'occupied worktree must not relaunch');
    assert.equal(calls.length,1);
  },{ops:operations({attemptRecord:async()=>records++===0?null:{kind:'WORKTREE_OCCUPIED'}}),
    behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
});

test('a completed attempt replays from git and is never relaunched',async()=>{
  await withHost(async({provider,calls,input})=>{
    const replayed=await provider.execute(input);
    assert.equal(replayed.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replayed.reason,'REAL_PI_CANDIDATE_RECORDED');
    assert.equal(replayed.replayed,true);
    assert.equal(replayed.replayFixture,false);
    assert.equal(replayed.candidateCommit,'c'.repeat(40));
    assert.equal(replayed.candidateRef,'refs/tally-pi-candidates/replay');
    assert.equal(replayed.process,null);
    assert.equal(calls.length,0,'replay must not spawn');
  },{ops:operations({attemptRecord:async()=>({kind:'CANDIDATE',
    candidateCommit:'c'.repeat(40),candidateTree:'d'.repeat(40),paths:ALLOWED,
    ref:'refs/tally-pi-candidates/replay'})})});
});

test('a stale candidate ref for another source SHA is NOT_ADMITTED',async()=>{
  await withHost(async({provider,input})=>{
    const x=await provider.execute(input);
    assert.equal(x.state,'NOT_ADMITTED');
    assert.equal(x.reason,'CANDIDATE_STALE');
    assert.equal(x.agentLaunch,false);assert.equal(x.admitted,false);
    assert.equal(x.claimAccepted,false);
  },{ops:operations({attemptRecord:async()=>({kind:'STALE'})})});
});

test('invalid inputs never execute; missing host authorization is NOT_ADMITTED',async()=>{
  await withHost(async({provider,calls,input,dir})=>{
    const validProviderConfig={cliPath:process.execPath,repositoryPath:dir};
    const cases=[
      {sourceSha:'not-a-sha',attemptId:'attempt:pi:x:1',worktreePath:join(dir,'w'),
        allowedWriteSet:['a.md'],taskPrompt:'t',providerConfig:validProviderConfig},
      {...input,sourceSha:sourceSha.toUpperCase()},
      {...input,attemptId:':starts-with-colon'},
      {...input,providerId:'zcode'},
      {...input,taskPrompt:''},
      {...input,allowedWriteSet:['ok',123]},
      {...input,providerConfig:{...validProviderConfig,provider:'bad provider'}},
      {...input,providerConfig:{...validProviderConfig,outputMode:'rpc'}},
      {...input,providerConfig:{...validProviderConfig,thinkingLevel:'ultra'}},
      {...input,providerConfig:{repositoryPath:dir}},
      {...input,providerConfig:{cliPath:process.execPath}},
      {...input,worktreePath:''}
    ];
    for(const bad of cases){
      const x=await provider.execute(bad);
      assert.equal(x.state,'AGENT_NOT_EXECUTED');
      assert.equal(x.reason,'ATTEMPT_INPUT_INVALID');
    }
    assert.equal(calls.length,0,'invalid inputs must never spawn');
    const noAuth=await provider.execute({...input,hostAuthorization:undefined});
    assert.equal(noAuth.state,'NOT_ADMITTED');
    assert.equal(noAuth.reason,'HOST_AUTHORIZATION_REQUIRED');
    assert.equal(noAuth.admitted,false);assert.equal(noAuth.claimAccepted,false);
    assert.equal(calls.length,0);
  });
});

test('builder environment is an explicit allowlist; spawn env forces isolation keys',()=>{
  const env=piBuilderEnvironment({cliPath:'C:\\tools\\pi\\pi.exe',
    emptyGitConfig:'C:\\empty\\gitconfig',builderTemp:'C:\\temp'});
  for(const name of ['GITHUB_TOKEN','GH_TOKEN','WOODPECKER_TOKEN','SSH_AUTH_SOCK',
    'SSH_AGENT_PID','DEEPSEEK_API_KEY','ANTHROPIC_API_KEY','OPENAI_API_KEY',
    'GEMINI_API_KEY','GROQ_API_KEY'])
    assert.equal(Object.hasOwn(env,name),false,`${name} must not be inherited`);
  for(const name of Object.keys(env))
    if(name.startsWith('PI_'))
      assert.fail(`builder env must not carry inherited PI_* var ${name}`);
  assert.equal(env.GIT_CONFIG_NOSYSTEM,'1');
  assert.equal(env.GIT_TERMINAL_PROMPT,'0');
  assert.equal(env.GCM_INTERACTIVE,'never');
  assert.equal(env.NO_COLOR,'1');
  assert.ok(env.PATH.startsWith('C:\\tools\\pi'));
  const dirs={home:'C:\\iso\\home',agentConfig:'C:\\iso\\agent-config',
    sessions:'C:\\iso\\sessions'};
  const spawnEnv=piSpawnEnvironment({env:{DEEPSEEK_API_KEY:'explicit-channel-only',
    PI_OFFLINE:'0',HOME:'operator-home',USERPROFILE:'operator-home'}},dirs);
  assert.equal(spawnEnv.DEEPSEEK_API_KEY,'explicit-channel-only',
    'providerConfig.env is the only credential channel');
  assert.equal(spawnEnv.PI_OFFLINE,'1','forced isolation key overrides config.env');
  assert.equal(spawnEnv.PI_TELEMETRY,'0');
  assert.equal(spawnEnv.HOME,dirs.home);
  assert.equal(spawnEnv.USERPROFILE,dirs.home);
  assert.equal(spawnEnv.PI_CODING_AGENT_DIR,dirs.agentConfig);
  assert.equal(spawnEnv.PI_CODING_AGENT_SESSION_DIR,dirs.sessions);
});

test('selection defaults to codex and admits pi only when configured and validated',()=>{
  const configured={enabled:true,cliPath:'C:/Application/pi/pi'};
  const validated={available:true,provider:'pi',version:'0.85.1'};
  assert.equal(resolvePiProviderSelection({requested:undefined,pi:configured,
    probe:validated}).reason,'CODEX_DEFAULT');
  assert.equal(resolvePiProviderSelection({requested:'codex',pi:configured,
    probe:validated}).selected,'codex');
  assert.equal(resolvePiProviderSelection({requested:'codex',pi:configured,
    probe:validated}).reason,'CODEX_EXPLICIT_DEFAULT');
  assert.equal(resolvePiProviderSelection({requested:'pi',pi:undefined,
    probe:validated}).reason,'PI_NOT_CONFIGURED');
  assert.equal(resolvePiProviderSelection({requested:'pi',
    pi:{enabled:false,cliPath:'C:/Application/pi/pi'},probe:validated}).reason,
    'PI_NOT_CONFIGURED');
  const unverified=resolvePiProviderSelection({requested:'pi',pi:configured,
    probe:{available:false,provider:'pi',reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY'}});
  assert.equal(unverified.selected,'codex');
  assert.equal(unverified.reason,'PI_CAPABILITY_UNVERIFIED');
  assert.equal(unverified.piAdmitted,false);
  const admitted=resolvePiProviderSelection({requested:'pi',pi:configured,
    probe:validated});
  assert.equal(admitted.selected,'pi');
  assert.equal(admitted.piAdmitted,true);
  assert.equal(admitted.reason,'PI_VALIDATED');
  // No failure-driven fallback exists: the policy is selection-only and a
  // codex failure has no path into it.
  assert.equal(resolvePiProviderSelection({requested:'codex',
    codexFailure:true}).selected,'codex');
});

test('NDJSON parsing takes the last assistant message_end, skips thinking blocks and truncated lines, never fabricates',()=>{
  const stream=[
    '{"type":"session","version":3,"id":"sess-uuid-1","cwd":"C:/w"}',
    '{"type":"agent_start"}',
    '{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"q"}]}}',
    '{"type":"message_end","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hmm"},{"type":"text","text":"first"}],"provider":"p1","model":"m1","usage":{"input":5,"output":2},"stopReason":"stop"}}',
    '{"type":"message_update","assistantMessageEvent":{"delta":"partial"}',
    '{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"final answer"}],"provider":"deepseek","model":"deepseek-v4-pro","usage":{"input":11,"output":3,"cacheRead":1},"stopReason":"stop"}}',
    '{"type":"truncated-tail'
  ].join('\n');
  const o=parsePiJsonStream(stream);
  assert.equal(o.hasAssistant,true);
  assert.equal(o.sessionHandle,'sess-uuid-1');
  assert.equal(o.response,'final answer');
  assert.equal(o.model,'deepseek-v4-pro');
  assert.equal(o.provider,'deepseek');
  assert.deepEqual(o.usage,{inputTokens:11,outputTokens:3});
  const empty=parsePiJsonStream('');
  assert.equal(empty.hasAssistant,false);assert.equal(empty.response,null);
  const thinkingOnly=parsePiJsonStream(
    '{"type":"session","id":"s2"}\n{"type":"message_end","message":{"role":"assistant","content":[{"type":"thinking","thinking":"only"}],"usage":{"input":1,"output":1}}}');
  assert.equal(thinkingOnly.hasAssistant,false);
  assert.equal(parsePiJsonStream(null).hasAssistant,false);
  assert.equal(parsePiJsonStream('plain text answer').hasAssistant,false);
  // An assistant message_end without usage leaves usage unobserved, not zero.
  const noUsage=parsePiJsonStream(
    '{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"x"}]}}');
  assert.equal(noUsage.hasAssistant,true);assert.equal(noUsage.usage,null);
});

test('task prompt names every allowed path exactly once; config tokens are strict',()=>{
  const prompt=piTaskPrompt({allowedWriteSet:['a.md','b/c.md'],taskPrompt:'Do it.'});
  for(const path of ['a.md','b/c.md'])
    assert.equal(prompt.split(path).length-1,1);
  assert.ok(prompt.includes('Task: Do it.'));
  assert.ok(prompt.includes('disposable Git worktree'));
  assert.equal(piConfigToken('deepseek'),true);
  assert.equal(piConfigToken('deepseek-v4-flash'),true);
  assert.equal(piConfigToken('openai/gpt-4o'),true);
  assert.equal(piConfigToken('medium'),true);
  assert.equal(piConfigToken('bad space'),false);
  assert.equal(piConfigToken('quote"inject'),false);
  assert.equal(piConfigToken(''),false);
  assert.equal(piConfigToken(undefined),false);
});

test('exported constants match the pi candidate vocabulary',()=>{
  assert.equal(CONSTANTS_CHECK.providerId,'pi');
  assert.equal(PI_PROVIDER_CONSTANTS.candidateRefPrefix,'refs/tally-pi-candidates/');
  assert.equal(PI_PROVIDER_CONSTANTS.canaryPrompt,'Reply with exactly: OK');
  assert.equal(PI_PROVIDER_CONSTANTS.pinnedVersion,'0.85.1');
  assert.equal(PI_PROVIDER_CONSTANTS.defaultOutputMode,'json');
  assert.equal(PI_PROVIDER_CONSTANTS.defaultCanaryTimeoutMs,180000);
  assert.ok(Object.isFrozen(PI_PROVIDER_CONSTANTS));
});

test('integration: real git worktree, real process spawn, fake pi binary end-to-end',async t=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-pi-integration-'));
  const gitPath=process.env.TALLY_TEST_GIT_PATH??
    (process.platform==='win32'?'C:\\Program Files\\Git\\cmd\\git.exe':
      (spawnSync('which',['git'],{encoding:'utf8'}).stdout.trim()||'git'));
  const gitSync=(args,cwd=dir)=>spawnSync(gitPath,['-c','user.name=Tally Test',
    '-c','user.email=tally-test@localhost','-c','commit.gpgsign=false',...args],
    {cwd,encoding:'utf8'});
  // The fake agent binary must be DIRECTLY spawnable: the pi contract launches
  // cliPath itself — no shell, no node prefix. On Windows that means a real PE
  // compiled with the stock .NET Framework csc.exe; on POSIX a shebang script.
  const csharpSource=escapeLine=>[
    'using System;',
    'using System.IO;',
    'static class FakePi',
    '{',
    '    static void Emit(string id, string text)',
    '    {',
    '        Console.Out.WriteLine("{\\"type\\":\\"session\\",\\"version\\":3,\\"id\\":\\"" + id + "\\"}");',
    '        Console.Out.WriteLine("{\\"type\\":\\"message_end\\",\\"message\\":{\\"role\\":\\"assistant\\",\\"content\\":[{\\"type\\":\\"text\\",\\"text\\":\\"" + text + "\\"}],\\"provider\\":\\"fake-provider\\",\\"model\\":\\"fake-model\\",\\"usage\\":{\\"input\\":11,\\"output\\":3},\\"stopReason\\":\\"stop\\"}}");',
    '    }',
    '    static int Main(string[] args)',
    '    {',
    '        if (args.Length == 0) return 2;',
    '        if (args[0] == "--version") { Console.Out.WriteLine("0.85.1"); return 0; }',
    '        if (args[0] == "auth") { Console.Out.WriteLine("{\\"status\\":\\"ready\\",\\"provider\\":\\"deepseek\\",\\"authType\\":\\"api_key\\"}"); return 0; }',
    '        if (args[0] != "-p") return 2;',
    '        int sep = Array.IndexOf(args, "--");',
    '        string instruction = (sep >= 0 && sep + 1 < args.Length) ? args[sep + 1] : "";',
    '        if (instruction.Contains("Reply with exactly")) { Emit("sess_canary", "OK"); return 0; }',
    '        File.WriteAllText(Path.Combine(Directory.GetCurrentDirectory(), "LOCAL_PI_L3.md"), "pi candidate note\\n");',
    escapeLine,
    '        Emit("sess_task", "Done.");',
    '        return 0;',
    '    }',
    '}'
  ].join('\r\n');
  const shellSource=escapeLine=>[
    '#!/bin/sh',
    'if [ "$1" = "--version" ]; then echo "0.85.1"; exit 0; fi',
    "if [ \"$1\" = \"auth\" ]; then echo '{\"status\":\"ready\",\"provider\":\"deepseek\",\"authType\":\"api_key\"}'; exit 0; fi",
    'if [ "$1" != "-p" ]; then exit 2; fi',
    'instruction=""',
    'prev=""',
    'for a in "$@"; do',
    '  if [ "$prev" = "--" ]; then instruction="$a"; break; fi',
    '  prev="$a"',
    'done',
    'case "$instruction" in',
    '  *"Reply with exactly"*)',
    "    echo '{\"type\":\"session\",\"version\":3,\"id\":\"sess_canary\"}'",
    "    echo '{\"type\":\"message_end\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"OK\"}],\"provider\":\"fake-provider\",\"model\":\"fake-model\",\"usage\":{\"input\":11,\"output\":3},\"stopReason\":\"stop\"}}'",
    '    exit 0 ;;',
    'esac',
    'printf \'pi candidate note\\n\' > "$PWD/LOCAL_PI_L3.md"',
    escapeLine,
    "echo '{\"type\":\"session\",\"version\":3,\"id\":\"sess_task\"}'",
    "echo '{\"type\":\"message_end\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"Done.\"}],\"provider\":\"fake-provider\",\"model\":\"fake-model\",\"usage\":{\"input\":11,\"output\":3},\"stopReason\":\"stop\"}}'",
    'exit 0'
  ].join('\n');
  async function buildFakeAgent(targetDir,{escape}){
    const csharpEscape='        File.WriteAllText(Path.Combine(Directory.GetCurrentDirectory(), "escape.txt"), "escape\\n");';
    const shellEscape='printf \'escape\\n\' > "$PWD/escape.txt"';
    if(process.platform==='win32'){
      const csc='C:\\Windows\\Microsoft.NET\\Framework64\\v4.0.30319\\csc.exe';
      if(!existsSync(csc)){t.skip('csc.exe unavailable for fake agent PE');return null;}
      const csPath=join(targetDir,escape?'fake-pi-escape.cs':'fake-pi.cs');
      const exePath=join(targetDir,escape?'fake-pi-escape.exe':'fake-pi.exe');
      await writeFile(csPath,csharpSource(escape?csharpEscape:''),'utf8');
      const compiled=spawnSync(csc,['/nologo','/target:exe',
        `/out:${exePath}`,csPath],{encoding:'utf8',windowsHide:true});
      if(compiled.status!==0)
        throw new Error(`fake pi compile failed: ${compiled.stdout} ${compiled.stderr}`);
      return exePath;
    }
    const scriptPath=join(targetDir,escape?'fake-pi-escape.sh':'fake-pi.sh');
    await writeFile(scriptPath,shellSource(escape?shellEscape:''),'utf8',{mode:0o755});
    await chmod(scriptPath,0o755);
    return scriptPath;
  }
  try{
    assert.equal(gitSync(['init','-b','main']).status,0,'git init');
    await writeFile(join(dir,'base.md'),'base\n','utf8');
    assert.equal(gitSync(['add','.']).status,0);
    assert.equal(gitSync(['commit','-m','base']).status,0);
    const sha=gitSync(['rev-parse','HEAD']).stdout.trim();
    assert.match(sha,/^[0-9a-f]{40}$/);
    const fakeCli=await buildFakeAgent(dir,{escape:false});
    if(!fakeCli)return;
    const fakeViolationCli=await buildFakeAgent(dir,{escape:true});
    const emptyGitConfig=join(dir,'empty-gitconfig');
    await writeFile(emptyGitConfig,'','utf8');
    const providerConfig={cliPath:fakeCli,repositoryPath:dir,provider:'deepseek',
      env:{DEEPSEEK_API_KEY:'integration-only-key'},gitPath,emptyGitConfig,
      systemRoot:process.env.SystemRoot??'C:\\Windows',
      systemDrive:process.env.SystemDrive??'C:'};
    const config={providerId:'pi',taskId:'task-64',issueRef,sourceSha:sha,
      attemptId:'attempt:pi:integration:1',worktreePath:join(dir,'wt'),
      allowedWriteSet:['LOCAL_PI_L3.md'],
      taskPrompt:'Create the candidate note file.',
      hostAuthorization:{mode:'explicit',attemptId:'attempt:pi:integration:1'},
      providerConfig,timeoutMs:60000};
    const provider=createPiTaskProvider({});
    // Positive: probe tiers, disposable worktree, real spawn, git finalize.
    const pass=await provider.execute(config);
    assert.equal(pass.state,'LOCAL_CANDIDATE_READY');
    assert.equal(pass.reason,'REAL_PI_CANDIDATE_RECORDED');
    assert.match(pass.candidateCommit,/^[0-9a-f]{40}$/);
    assert.match(pass.candidateTree,/^[0-9a-f]{40}$/);
    assert.equal(pass.candidateRef,'refs/tally-pi-candidates/attempt-pi-integration-1',
      'attemptId colons are sanitized for the git ref namespace');
    assert.deepEqual(pass.candidatePaths,['LOCAL_PI_L3.md']);
    assert.equal(pass.agent.sessionHandle,'sess_task');
    assert.equal(pass.process.exitCode,0);
    assert.equal(pass.providerProbe.version,'0.85.1');
    // Independent git readback re-derives tree/parent/paths from the repo.
    const verified=await provider.readback(config,pass);
    assert.equal(verified.state,'CANDIDATE_READBACK_VERIFIED');
    assert.equal(verified.verified,true);
    assert.equal(verified.parent,sha);
    assert.equal(verified.candidateTree,pass.candidateTree);
    assert.deepEqual(verified.paths,['LOCAL_PI_L3.md']);
    // Replay: same attempt id re-reads the durable ref and never relaunches.
    const replay=await provider.execute(config);
    assert.equal(replay.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replay.replayed,true);
    assert.equal(replay.candidateCommit,pass.candidateCommit);
    assert.equal(replay.agent,undefined,'replay carries no fresh agent observation');
    // Readback against a parentless/foreign commit is a mismatch, never a pass.
    const foreign=await provider.readback(config,
      {candidateCommit:sha,attemptId:config.attemptId});
    assert.equal(foreign.verified,false);
    assert.equal(foreign.state,'CANDIDATE_READBACK_MISMATCH');
    // Write-set fence with the fake binary writing outside the allowed set.
    const violation=await provider.execute({...config,
      attemptId:'attempt:pi:integration:2',worktreePath:join(dir,'wt-violation'),
      providerConfig:{...providerConfig,cliPath:fakeViolationCli}});
    assert.equal(violation.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(violation.reason,'WRITE_SET_VIOLATION');
    assert.ok(violation.candidatePaths.includes('escape.txt'));
    assert.ok(violation.candidatePaths.includes('LOCAL_PI_L3.md'));
    assert.equal(violation.candidateTree,null);
  }finally{await rm(dir,{recursive:true,force:true});}
});
