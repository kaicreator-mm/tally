import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {mkdtemp,mkdir,rm,writeFile,readFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawn,spawnSync} from 'node:child_process';
import {createOpencodeTaskProvider,OPENCODE_PROVIDER_CONSTANTS,
  OPENCODE_PROVIDER_CONSTANTS as CONSTANTS_CHECK,resolveOpencodeProviderSelection,
  parseOpencodeRunOutput,opencodeBuilderEnvironment,opencodeSessionEnvironment,
  opencodeTaskPrompt,validateOpencodeFlagConfig} from './opencode-provider.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/64';
const ALLOWED=['experiments/v2/selected-business-build/LOCAL_OPENCODE_L3.md'];
const candidate={writeSetValid:true,paths:ALLOWED,candidateTree:'a'.repeat(40),
  candidateCommit:'b'.repeat(40),candidateRef:'refs/tally-opencode-candidates/test'};
const probeOk={available:true,provider:'opencode',version:'1.19.6',
  canary:{answer:'OK',usage:{inputTokens:8846,outputTokens:2}}};
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
    probeProvider:async()=>({...probeOk}),
    attemptRecord:async()=>null,
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha}),
    finalizeCandidate:async()=>candidate,...overrides
  };
}
function providerConfig(dir){
  // cliPath must point at an EXISTING file (the probe's VERSION stage checks
  // existsSync before spawning; the fake spawnFn intercepts the process).
  return {cliPath:process.execPath,repositoryPath:dir,
    env:{DEEPSEEK_API_KEY:'test-key'}};
}
function configFor(dir,attemptId='attempt:opencode:test:1'){
  return {providerId:'opencode',taskId:'task:opencode:1',issueRef,sourceSha,attemptId,
    worktreePath:join(dir,'candidate'),providerConfig:providerConfig(dir),
    taskPrompt:'Create the candidate note file.',
    allowedWriteSet:ALLOWED,timeoutMs:1000};
}
const ndjson=(answer='Done.')=>[
  JSON.stringify({type:'step_start',timestamp:1,sessionID:'ses_fake',part:{type:'step-start'}}),
  JSON.stringify({type:'text',timestamp:2,sessionID:'ses_fake',part:{type:'text',text:answer}}),
  JSON.stringify({type:'step_finish',timestamp:3,sessionID:'ses_fake',
    part:{type:'step-finish',reason:'stop',tokens:{total:40,input:38,output:2}}})
].join('\n')+'\n';
async function withProvider(fn,{ops=operations(),behavior=(close,child)=>{child.stdout.emit('data',ndjson());close(0,null);},
  killTree=async()=>true}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-opencode-provider-test-'));
  const calls=[],kills=[],probes=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
  };
  const countingOps={...ops,probeProvider:async config=>{
    probes.push(config);return ops.probeProvider(config);}};
  const provider=createOpencodeTaskProvider({spawnFn,operations:countingOps,now:()=>Date.UTC(2026,10,19),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return killTree(pid);}});
  try{return await fn({provider,dir,calls,kills,probes,config:configFor(dir)});}
  finally{await rm(dir,{recursive:true,force:true});}
}

test('exported constants match the opencode contract vocabulary',()=>{
  assert.equal(CONSTANTS_CHECK.providerId,'opencode');
  assert.equal(OPENCODE_PROVIDER_CONSTANTS.pinnedCliVersion,'1.19.6');
  assert.equal(OPENCODE_PROVIDER_CONSTANTS.candidateRefPrefix,'refs/tally-opencode-candidates/');
  assert.equal(OPENCODE_PROVIDER_CONSTANTS.canaryPrompt,'Reply with exactly: OK');
  assert.equal(OPENCODE_PROVIDER_CONSTANTS.canaryTimeoutMs,180000);
  assert.equal(OPENCODE_PROVIDER_CONSTANTS.providerApiKeyVar,'DEEPSEEK_API_KEY');
});

test('probe reports available after VERSION, CREDENTIALS and CANARY stages',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-opencode-probe-'));
  try{
    const calls=[];
    const behavior=(close,child)=>{
      const args=calls.at(-1).args;
      if(args.includes('--version')){child.stdout.emit('data','1.19.6\n');close(0,null);}
      else if(args[0]==='run'){child.stdout.emit('data',ndjson('OK'));close(0,null);}
      else close(1,null);
    };
    const spawnFn=(command,args,options)=>{calls.push({command,args,options});
      return fakeChild(behavior);};
    const provider=createOpencodeTaskProvider({spawnFn});
    const probe=await provider.probe(providerConfig(dir));
    assert.equal(probe.available,true);
    assert.equal(probe.provider,'opencode');
    assert.equal(probe.version,'1.19.6');
    assert.equal(probe.canary.answer,'OK');
    assert.deepEqual(probe.canary.usage,{inputTokens:38,outputTokens:2});
    assert.equal(calls.length,2,'version probe + canary only');
    assert.equal(calls[0].command,process.execPath);
    assert.deepEqual(calls[0].args,['--version']);
    const canaryArgs=calls[1].args;
    assert.equal(calls[1].command,process.execPath);
    assert.equal(canaryArgs[0],'run');
    assert.ok(canaryArgs.includes('--format'));
    assert.equal(canaryArgs[canaryArgs.indexOf('--format')+1],'json');
    assert.ok(canaryArgs.includes('--pure'));
    assert.equal(canaryArgs.at(-1),'Reply with exactly: OK');
    // per-attempt isolation is already active for the canary turn
    assert.match(calls[1].options.env.XDG_CONFIG_HOME,/tally-opencode-attempt-/);
    assert.equal(calls[1].options.env.PWD,calls[1].options.cwd);
    assert.equal(calls[1].options.env.OLDPWD,calls[1].options.cwd);
    assert.equal(calls[1].options.env.DEEPSEEK_API_KEY,'test-key');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('probe negatives are honest: missing CLI, absent credentials, stalling canary',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-opencode-probe-neg-'));
  try{
    const calls=[];
    const spawnFn=(command,args,options)=>{calls.push({command,args,options});
      return fakeChild((close,child)=>{
        if(args.includes('--version')){child.stdout.emit('data','1.19.6\n');close(0,null);}
        else if(args[0]==='run'){} // canary stalls until the probe budget kills it
        else close(0,null);
      });};
    const provider=createOpencodeTaskProvider({spawnFn});
    const missingProbe=await provider.probe({...providerConfig(dir),
      cliPath:join(dir,'absent-opencode.exe')});
    assert.equal(missingProbe.available,false);
    assert.equal(missingProbe.stage,'VERSION');
    assert.equal(missingProbe.detail,'MISSING_CLI');
    assert.equal(missingProbe.reason,'LOCAL_AGENT_UNAVAILABLE');
    const noCredentials=await provider.probe({...providerConfig(dir),
      env:{}});
    assert.equal(noCredentials.available,false);
    assert.equal(noCredentials.stage,'CREDENTIALS');
    assert.equal(noCredentials.detail,'CREDENTIALS_ABSENT');
    const stall=await provider.probe({...providerConfig(dir),canaryTimeoutMs:30});
    assert.equal(stall.available,false);
    assert.equal(stall.stage,'CANARY');
    assert.equal(stall.detail,'CREDENTIALS_OR_PROVIDER_STALL');
    assert.equal(calls.length,3,'one version probe per probe call, canary only with credentials');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('probe fails closed on an unparsable or unpinned version',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-opencode-probe-ver-'));
  try{
    const spawnFn=(command,args,options)=>fakeChild((close,child)=>{
      if(args.includes('--version')){child.stdout.emit('data','0.2.0\n');close(0,null);}
      else if(args[0]==='run'){child.stdout.emit('data',ndjson('OK'));close(0,null);}
      else close(0,null);});
    const provider=createOpencodeTaskProvider({spawnFn});
    const mismatch=await provider.probe(providerConfig(dir));
    assert.equal(mismatch.available,false);
    assert.equal(mismatch.stage,'VERSION');
    assert.equal(mismatch.detail,'VERSION_MISMATCH');
    assert.equal(mismatch.observedVersion,'0.2.0');
    assert.equal(mismatch.pinnedVersion,'1.19.6');
    const pinned=await provider.probe({...providerConfig(dir),pinnedVersion:'0.2.0'});
    assert.equal(pinned.available,true);
    assert.equal(pinned.version,'0.2.0');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('an unavailable probe never spawns the task process',async()=>{
  await withProvider(async({provider,calls,probes,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.agentLaunch,false);assert.equal(calls.length,0);
    assert.equal(probes.length,1);
    assert.equal(x.providerProbe.stage,'CREDENTIALS');
    assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
  },{ops:operations({probeProvider:async()=>
    ({available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',
      detail:'CREDENTIALS_ABSENT'})})});
});

test('a real process exit produces a READY candidate with the #471 argv shape',async()=>{
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(x.reason,'REAL_OPENCODE_CANDIDATE_RECORDED');
    assert.equal(x.agentLaunch,true);assert.equal(x.replayFixture,false);
    assert.equal(x.process.exitCode,0);assert.equal(x.process.pid,4242);
    assert.equal(x.candidateTree,candidate.candidateTree);
    assert.equal(x.candidateCommit,candidate.candidateCommit);
    assert.equal(x.candidateRef,candidate.candidateRef);
    assert.equal(x.agent.answer,'Done.');
    assert.deepEqual(x.agent.usage,{inputTokens:38,outputTokens:2});
    assert.equal(x.provider,'opencode');assert.equal(x.providerId,'opencode');
    assert.equal(x.taskId,'task:opencode:1');
    assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
    assert.equal(x.authorized,false);
    assert.equal(calls.length,1);
    const args=calls[0].args;
    assert.equal(calls[0].command,process.execPath,'CLI launched directly, no shell');
    assert.equal(args[0],'run');
    assert.equal(args[args.indexOf('--format')+1],'json');
    assert.ok(args.includes('--pure'));
    assert.ok(!args.includes('--attach'),'the shared-server surface is never passed');
    assert.ok(!args.includes('--dir'),'workspace pins via cwd, not a flag');
    const prompt=args.at(-1);
    for(const path of ALLOWED)assert.ok(prompt.includes(path),`prompt carries ${path}`);
    assert.ok(prompt.includes('Do not run git commit, git push'));
    assert.ok(prompt.includes('Task: Create the candidate note file.'));
    assert.equal(calls[0].options.cwd,config.worktreePath,'spawn cwd pins the worktree');
    assert.match(calls[0].options.env.XDG_CONFIG_HOME,/tally-opencode-attempt-/);
    assert.match(calls[0].options.env.XDG_DATA_HOME,/tally-opencode-attempt-/);
    assert.equal(calls[0].options.env.PWD,config.worktreePath);
    assert.equal(calls[0].options.env.OLDPWD,config.worktreePath);
    assert.equal(calls[0].options.env.DEEPSEEK_API_KEY,'test-key');
    assert.equal(Object.hasOwn(calls[0].options.env,'GITHUB_TOKEN'),false);
  });
});

test('isolation keys and workspace pins override config.env at spawn time',async()=>{
  await withProvider(async({provider,calls,config})=>{
    await provider.execute({...config,providerConfig:{...config.providerConfig,
      env:{DEEPSEEK_API_KEY:'test-key',XDG_CONFIG_HOME:'C:\\owner-config',
        XDG_DATA_HOME:'C:\\owner-data',TMP:'C:\\owner-tmp',PWD:'C:\\owner-pwd'}}});
    const env=calls[0].options.env;
    assert.match(env.XDG_CONFIG_HOME,/tally-opencode-attempt-/);
    assert.match(env.XDG_DATA_HOME,/tally-opencode-attempt-/);
    assert.equal(env.TMP,env.TEMP);
    assert.match(env.TMP,/tally-opencode-attempt-/);
    assert.equal(env.PWD,config.worktreePath);
    assert.equal(env.OLDPWD,config.worktreePath);
  });
});

test('model and agent expectations ride the observed flags before the prompt',async()=>{
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute({...config,providerConfig:{...config.providerConfig,
      model:'deepseek/deepseek-v4-pro',agent:'build'}});
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    const args=calls[0].args;
    const mi=args.indexOf('--model');
    const ai=args.indexOf('--agent');
    assert.equal(args[mi+1],'deepseek/deepseek-v4-pro');
    assert.equal(args[ai+1],'build');
    assert.ok(mi>0&&mi<ai&&ai<args.length-1,'flags precede the positional prompt');
    assert.equal(args.at(-1).startsWith('Work only in the current disposable'),true);
  });
});

test('writes outside the allowed write set are a violation, never a candidate',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'WRITE_SET_VIOLATION');
    assert.equal(x.candidateTree,null);assert.equal(x.taskPass,false);
    assert.deepEqual(x.candidatePaths,['forbidden.txt','escape.md']);
  },{ops:operations({finalizeCandidate:async()=>({writeSetValid:false,
    paths:['forbidden.txt','escape.md']})})});
});

test('a nonempty but unallowed diff cannot pass through an empty allowed set',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute({...config,allowedWriteSet:[]});
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'WRITE_SET_VIOLATION');
  },{ops:operations({finalizeCandidate:async config=>{
    const allowed=new Set(config.allowedWriteSet??[]);
    const paths=['LOCAL_OPENCODE_L3.md'];
    if(paths.length===0||paths.some(path=>!allowed.has(path)))
      return {writeSetValid:false,paths};
    return candidate;
  }})});
});

test('nonzero process exit is FAILED with the captured exit code',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.process.exitCode,7);
  },{behavior:(close,child)=>{child.stdout.emit('data',ndjson('partial before crash'));close(7,null);}});
});

test('exit 0 with an empty stdout violates the observed contract and fails safe',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_CONTRACT_UNFULFILLED');
    assert.equal(x.taskPass,false);
  },{behavior:close=>close(0,null)});
});

test('timeout terminates the process tree and records UNKNOWN',async()=>{
  await withProvider(async({provider,kills,config})=>{
    const x=await provider.execute({...config,timeoutMs:25});
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('cancellation through input.signal kills the child and records CANCELLED',async()=>{
  await withProvider(async({provider,kills,config})=>{
    const controller=new AbortController();setTimeout(()=>controller.abort(),10);
    const x=await provider.execute({...config,timeoutMs:1000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('a pre-aborted signal cancels before any record, probe or spawn',async()=>{
  await withProvider(async({provider,calls,probes,config})=>{
    const controller=new AbortController();controller.abort();
    const x=await provider.execute({...config,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.equal(x.agentLaunch,false);assert.equal(x.process,null);
    assert.equal(calls.length,0);assert.equal(probes.length,0);
  });
});

test('spawn errors are UNKNOWN with no invented exit code',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(x.agentLaunch,true);
    assert.equal(x.process.status,'UNKNOWN');
    assert.equal(x.process.exitCode,null);
  },{behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
});

test('an occupied worktree blocks blind relaunch and stays UNKNOWN',async()=>{
  let records=0;
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(calls.length,1);
    const again=await provider.execute(config);
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(again.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(again.agentLaunch,false);
    assert.equal(calls.length,1,'occupied worktree must not relaunch');
  },{ops:operations({attemptRecord:async()=>records++===0?null:{kind:'WORKTREE_OCCUPIED'}}),
    behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
});

test('a completed attempt replays from git and is never relaunched',async()=>{
  await withProvider(async({provider,calls,config})=>{
    const replayed=await provider.execute(config);
    assert.equal(replayed.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replayed.replayed,true);
    assert.equal(replayed.candidateCommit,'c'.repeat(40));
    assert.equal(replayed.candidateRef,'refs/tally-opencode-candidates/replay');
    assert.equal(replayed.replayFixture,false);
    assert.equal(calls.length,0,'replay must not spawn');
  },{ops:operations({attemptRecord:async()=>({kind:'CANDIDATE',
    candidateCommit:'c'.repeat(40),candidateTree:'d'.repeat(40),paths:ALLOWED,
    ref:'refs/tally-opencode-candidates/replay'})})});
});

test('a stale candidate ref for another source SHA is NOT_ADMITTED',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'NOT_ADMITTED');
    assert.equal(x.reason,'CANDIDATE_STALE');
    assert.equal(x.agentLaunch,false);assert.equal(x.admitted,false);
  },{ops:operations({attemptRecord:async()=>({kind:'STALE'})})});
});

test('invalid attempt inputs never probe and never execute',async()=>{
  await withProvider(async({provider,calls,probes})=>{
    const dir=await mkdtemp(join(tmpdir(),'tally-opencode-invalid-'));
    try{
      for(const bad of [
        {sourceSha:'not-a-sha'},
        {attemptId:'bad id with spaces'},
        {providerId:'zcode'},
        {taskPrompt:''},
        {allowedWriteSet:'LOCAL_OPENCODE_L3.md'},
        {allowedWriteSet:[42]},
        {providerConfig:{...providerConfig(dir),cliPath:''}},
        {providerConfig:{...providerConfig(dir),model:'no-slash'}},
        {providerConfig:{...providerConfig(dir),model:'a/b/c'}},
        {providerConfig:{...providerConfig(dir),agent:'bad agent'}},
      ]){
        const x=await provider.execute({...configFor(dir),...bad});
        assert.equal(x.state,'AGENT_NOT_EXECUTED',JSON.stringify(bad));
        assert.equal(x.reason,'ATTEMPT_INPUT_INVALID');
        assert.equal(x.agentLaunch,false);
      }
      assert.equal(probes.length,0,'invalid input must not reach the probe');
      assert.equal(calls.length,0);
    }finally{await rm(dir,{recursive:true,force:true});}
  });
});

test('cancel() transitions only through the input.signal channel',()=>{
  const provider=createOpencodeTaskProvider({spawnFn:()=>fakeChild()});
  assert.equal(provider.cancel({}),false);
  assert.equal(provider.cancel(),false);
  const aborted=new AbortController();aborted.abort();
  assert.equal(provider.cancel({signal:aborted.signal}),false);
  const controller=new AbortController();
  assert.equal(provider.cancel({signal:controller.signal,
    abort:()=>controller.abort()}),true);
  assert.equal(controller.signal.aborted,true);
  const duck={signal:{aborted:false,abort(){this.aborted=true;}}};
  assert.equal(provider.cancel(duck),true);
  assert.equal(duck.signal.aborted,true);
  assert.equal(provider.cancel({signal:{aborted:false}}),false,'no abort surface, no lie');
});

test('readback re-derives the candidate from git independently',async()=>{
  await withProvider(async({provider,config})=>{
    const readbackConfig={...config.providerConfig,sourceSha};
    const readback=await provider.readback(readbackConfig,
      {candidateCommit:'b'.repeat(40)});
    assert.equal(readback.verified,true);
    assert.equal(readback.candidateTree,'e'.repeat(40));
    assert.equal(readback.parent,sourceSha);
    assert.deepEqual(readback.paths,ALLOWED);
    const bad=await provider.readback(readbackConfig,{candidateCommit:'zz'});
    assert.equal(bad.verified,false);
  },{ops:operations({verifyCandidateReadback:async(config,candidate)=>{
    if(!/^[0-9a-f]{40}$/.test(candidate.candidateCommit??''))return {verified:false};
    return {verified:true,candidateTree:'e'.repeat(40),parent:config.sourceSha,
      paths:ALLOWED};
  }})});
});

test('NDJSON parsing: last text wins, usage sums, noise and error envelopes tolerated',()=>{
  const stream=[
    JSON.stringify({type:'step_start',timestamp:1,sessionID:'s',part:{type:'step-start'}}),
    JSON.stringify({type:'tool_use',timestamp:2,sessionID:'s',part:{type:'tool',tool:'write'}}),
    JSON.stringify({type:'step_finish',timestamp:3,sessionID:'s',
      part:{type:'step-finish',reason:'tool-calls',tokens:{total:1,input:47,output:83}}}),
    JSON.stringify({type:'step_start',timestamp:4,sessionID:'s',part:{type:'step-start'}}),
    JSON.stringify({type:'text',timestamp:5,sessionID:'s',part:{type:'text',text:'earlier'}}),
    JSON.stringify({type:'text',timestamp:6,sessionID:'s',part:{type:'text',text:'done'}}),
    JSON.stringify({type:'step_finish',timestamp:7,sessionID:'s',
      part:{type:'step-finish',reason:'stop',tokens:{total:2,input:46,output:2}}}),
    '',
    'not json at all',
    JSON.stringify({type:'error',timestamp:8,sessionID:'s',
      error:{name:'UnknownError',data:{message:'boom'},ref:'err_x'}})
  ].join('\n');
  const parsed=parseOpencodeRunOutput(stream);
  assert.equal(parsed.answer,'done');
  assert.deepEqual(parsed.usage,{inputTokens:93,outputTokens:85});
  assert.equal(parsed.sawJSON,true);
  const plain=parseOpencodeRunOutput('plain text answer');
  assert.equal(plain.answer,null);assert.deepEqual(plain.usage,{inputTokens:0,outputTokens:0});
  assert.equal(plain.sawJSON,false);
  const errorOnly=parseOpencodeRunOutput(
    JSON.stringify({type:'error',error:{name:'UnknownError'}}));
  assert.equal(errorOnly.answer,null);assert.equal(errorOnly.sawJSON,true);
  assert.equal(parseOpencodeRunOutput('').sawJSON,false);
  assert.equal(parseOpencodeRunOutput(null).sawJSON,false);
  assert.equal(parseOpencodeRunOutput('{broken\nlines').sawJSON,false);
});

test('flag config validation rejects unobserved model/agent shapes',()=>{
  for(const good of [{},{model:'deepseek/deepseek-v4-pro'},{model:'a/b'},
    {agent:'build'},{model:'a/b',agent:'review_bot'}])
    assert.equal(validateOpencodeFlagConfig(good).valid,true,JSON.stringify(good));
  for(const bad of [{model:'no-slash'},{model:'a/b/c'},{model:'/lead'},{model:'trail/'},
    {model:'bad model'},{model:'a/b c'},{agent:'a/b'},{agent:'bad agent'},{agent:"q'"}])
    assert.equal(validateOpencodeFlagConfig(bad).valid,false,JSON.stringify(bad));
});

test('builder environment is an explicit allowlist with no credential inheritance',()=>{
  const env=opencodeBuilderEnvironment({cliPath:'C:\\fake\\opencode.exe',
    emptyGitConfig:'C:\\empty\\gitconfig',builderTemp:'C:\\temp'});
  for(const name of ['GITHUB_TOKEN','GH_TOKEN','WOODPECKER_TOKEN','SSH_AUTH_SOCK',
    'SSH_AGENT_PID','DEEPSEEK_API_KEY','OPENCODE_CONFIG','OPENCODE_SERVER_PASSWORD',
    'CODEX_HOME','ANTHROPIC_API_KEY'])
    assert.equal(Object.hasOwn(env,name),false,name);
  assert.equal(env.GIT_CONFIG_NOSYSTEM,'1');
  assert.equal(env.GIT_TERMINAL_PROMPT,'0');
  assert.equal(env.GCM_INTERACTIVE,'never');
  assert.equal(env.NO_COLOR,'1');
  const injected=opencodeBuilderEnvironment({env:{DEEPSEEK_API_KEY:'explicit-only'}});
  assert.equal(injected.DEEPSEEK_API_KEY,'explicit-only');
  assert.equal(Object.hasOwn(opencodeBuilderEnvironment({}),'DEEPSEEK_API_KEY'),false);
});

test('session environment pins PWD/OLDPWD to the workspace',()=>{
  const fakeDirs={env:()=>({XDG_CONFIG_HOME:'C:\\s\\config',XDG_DATA_HOME:'C:\\s\\data',
    XDG_CACHE_HOME:'C:\\s\\cache',XDG_STATE_HOME:'C:\\s\\state',TMP:'C:\\s\\tmp',TEMP:'C:\\s\\tmp'})};
  const env=opencodeSessionEnvironment(fakeDirs,'C:\\wt');
  assert.equal(env.PWD,'C:\\wt');
  assert.equal(env.OLDPWD,'C:\\wt');
  assert.equal(env.XDG_DATA_HOME,'C:\\s\\data');
});

test('task prompt is bounded and mentions every allowed path exactly once',()=>{
  const prompt=opencodeTaskPrompt({allowedWriteSet:['a.md','b/c.md'],taskPrompt:'Do it.'});
  for(const path of ['a.md','b/c.md'])
    assert.equal(prompt.split(path).length-1,1);
  assert.ok(prompt.includes('Task: Do it.'));
  assert.ok(prompt.includes('Do not run git commit, git push'));
});

test('selection defaults to codex and admits opencode only when configured and validated',()=>{
  const configured={enabled:true,cliPath:'C:\\bin\\opencode.exe'};
  const validated={...probeOk};
  assert.equal(resolveOpencodeProviderSelection({requested:undefined,
    opencode:configured,probe:validated}).reason,'CODEX_DEFAULT');
  assert.equal(resolveOpencodeProviderSelection({requested:'codex',
    opencode:configured,probe:validated}).selected,'codex');
  assert.equal(resolveOpencodeProviderSelection({requested:'zcode',
    opencode:configured,probe:validated}).selected,'codex');
  assert.equal(resolveOpencodeProviderSelection({requested:'opencode',
    opencode:undefined,probe:validated}).reason,'OPENCODE_NOT_CONFIGURED');
  assert.equal(resolveOpencodeProviderSelection({requested:'opencode',
    opencode:{enabled:false,cliPath:'C:\\bin\\opencode.exe'},probe:validated}).reason,
    'OPENCODE_NOT_CONFIGURED');
  const unverified=resolveOpencodeProviderSelection({requested:'opencode',
    opencode:configured,probe:{available:false,reason:'LOCAL_AGENT_UNAVAILABLE',
      stage:'CANARY',detail:'CREDENTIALS_OR_PROVIDER_STALL'}});
  assert.equal(unverified.selected,'codex');
  assert.equal(unverified.reason,'OPENCODE_CAPABILITY_UNVERIFIED');
  assert.equal(unverified.opencodeAdmitted,false);
  const admitted=resolveOpencodeProviderSelection({requested:'opencode',
    opencode:configured,probe:validated});
  assert.equal(admitted.selected,'opencode');
  assert.equal(admitted.opencodeAdmitted,true);
  assert.equal(admitted.reason,'OPENCODE_VALIDATED');
  // No failure-driven fallback exists: the policy is selection-only and a
  // codex failure has no path into it.
  assert.equal(resolveOpencodeProviderSelection({requested:'codex',
    codexFailure:true}).selected,'codex');
});

test('integration: real git worktree, real process spawn, fake opencode binary end-to-end',async t=>{
  // The fake CLI is a go-built PE (Windows has no shebang exec). Runners
  // without a go toolchain (the CI alm lane) skip the real-binary seam
  // honestly instead of failing on the missing compiler.
  if(spawnSync('go',['version'],{encoding:'utf8'}).status!==0){
    t.skip('go toolchain unavailable on this runner — real-binary seam exercised on Windows dev boxes');
    return;
  }
  const dir=await mkdtemp(join(tmpdir(),'tally-opencode-integration-'));
  // The host's git operations run with the stripped hostEnvironment PATH, so
  // the test resolves an ABSOLUTE git path (PATH lookup would fail there).
  const gitPath=process.env.TALLY_TEST_GIT_PATH??
    (process.platform==='win32'?'C:\\Program Files\\Git\\cmd\\git.exe':
      (spawnSync('which',['git'],{encoding:'utf8'}).stdout.trim()||'git'));
  const gitSync=(args,cwd=dir)=>spawnSync(gitPath,['-c','user.name=Tally Test',
    '-c','user.email=tally-test@localhost','-c','commit.gpgsign=false',...args],
    {cwd,encoding:'utf8'});
  try{
    assert.equal(gitSync(['init','-b','main']).status,0,'git init');
    await writeFile(join(dir,'base.md'),'base\n','utf8');
    assert.equal(gitSync(['add','.']).status,0);
    assert.equal(gitSync(['commit','-m','base']).status,0);
    const sha=gitSync(['rev-parse','HEAD']).stdout.trim();
    assert.match(sha,/^[0-9a-f]{40}$/);
    // Fake opencode binary: a minimal Go stand-in for the observed 1.19.6
    // surface, built at test runtime exactly like the #471 Go harness builds
    // its testdata/fakeopencode (go build). `--version` prints the bare
    // pinned line; `run --format json --pure` records the observed
    // env/isolation surface and performs the workspace edit like the real
    // CLI would. A scenario variant also writes outside the allowed write
    // set. (Node.exe cannot play this role: it handles --version before any
    // NODE_OPTIONS preload, so a preload cannot intercept the version gate.)
    const fakeDir=join(dir,'fake-bin');
    await mkdir(fakeDir,{recursive:true});
    const fakeSrcPath=join(fakeDir,'main.go');
    await writeFile(fakeSrcPath,[
      'package main',
      '',
      'import (',
      '\t"encoding/json"',
      '\t"fmt"',
      '\t"os"',
      '\t"regexp"',
      '\t"strings"',
      ')',
      '',
      'func emit(envType, partType, text string, input, output int64, withTokens bool) {',
      '\tpart := map[string]any{"id": "prt_fake", "sessionID": "ses_fake", "type": partType}',
      '\tif text != "" {',
      '\t\tpart["text"] = text',
      '\t}',
      '\tif withTokens {',
      '\t\tpart["reason"] = "stop"',
      '\t\tpart["tokens"] = map[string]any{"total": input + output, "input": input, "output": output}',
      '\t}',
      '\tline, _ := json.Marshal(map[string]any{"type": envType, "timestamp": 1,',
      '\t\t"sessionID": "ses_fake", "part": part})',
      '\tfmt.Println(string(line))',
      '}',
      '',
      'func main() {',
      '\targs := os.Args[1:]',
      '\tif len(args) == 0 {',
      '\t\tos.Exit(2)',
      '\t}',
      '\tif args[0] == "--version" {',
      '\t\tfmt.Println("1.19.6")',
      '\t\treturn',
      '\t}',
      '\tif args[0] != "run" {',
      '\t\tfmt.Fprintln(os.Stderr, "unexpected argument")',
      '\t\tos.Exit(2)',
      '\t}',
      '\trest := args[1:]',
      '\tformat := "default"',
      '\tvar msg []string',
      '\tfor i := 0; i < len(rest); i++ {',
      '\t\tswitch rest[i] {',
      '\t\tcase "--format":',
      '\t\t\tif i+1 >= len(rest) {',
      '\t\t\t\tos.Exit(2)',
      '\t\t\t}',
      '\t\t\ti++',
      '\t\t\tformat = rest[i]',
      '\t\tcase "--model", "--agent":',
      '\t\t\ti++',
      '\t\tdefault:',
      '\t\t\tmsg = append(msg, rest[i])',
      '\t\t}',
      '\t}',
      '\tinstruction := strings.Join(msg, " ")',
      '\tif format != "json" {',
      '\t\tfmt.Fprintln(os.Stderr, "unsupported format")',
      '\t\tos.Exit(2)',
      '\t}',
      '\tcwd, _ := os.Getwd()',
      '\tif path := os.Getenv("FAKE_OPENCODE_RECORD"); path != "" {',
      '\t\trec := map[string]any{"argv": os.Args, "cwd": cwd, "pwd": os.Getenv("PWD"),',
      '\t\t\t"oldpwd": os.Getenv("OLDPWD"), "xdgConfig": os.Getenv("XDG_CONFIG_HOME"),',
      '\t\t\t"xdgData": os.Getenv("XDG_DATA_HOME"), "tmp": os.Getenv("TEMP"),',
      '\t\t\t"apiKeySet": os.Getenv("DEEPSEEK_API_KEY") != "", "message": instruction}',
      '\t\tif line, err := json.Marshal(rec); err == nil {',
      '\t\t\tif f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {',
      '\t\t\t\t_, _ = f.WriteString(string(line) + "\\n")',
      '\t\t\t\t_ = f.Close()',
      '\t\t\t}',
      '\t\t}',
      '\t}',
      '\tif os.Getenv("FAKE_OPENCODE_SCENARIO") == "exitfail" {',
      '\t\temit("step_start", "step-start", "", 0, 0, false)',
      '\t\temit("text", "text", "partial before crash", 0, 0, false)',
      '\t\tfmt.Fprintln(os.Stderr, `{"name":"UnknownError","data":{"message":"boom"}}`)',
      '\t\tos.Exit(7)',
      '\t}',
      '\tanswer := "Done."',
      '\tif strings.Contains(instruction, "Reply with exactly") {',
      '\t\tanswer = "OK"',
      '\t}',
      '\ttarget := ""',
      '\tif m := regexp.MustCompile(`-\\s+(\\S+\\.(?:md|txt))`).FindStringSubmatch(instruction); m != nil {',
      '\t\ttarget = m[1]',
      '\t\tanswer = "Done."',
      '\t}',
      '\tif target != "" {',
      '\t\t_ = os.WriteFile(target, []byte("opencode candidate note\\n"), 0o644)',
      '\t}',
      '\tif os.Getenv("FAKE_OPENCODE_ESCAPE") != "" && target != "" {',
      '\t\t_ = os.WriteFile("escape.txt", []byte("escape\\n"), 0o644)',
      '\t}',
      '\temit("step_start", "step-start", "", 0, 0, false)',
      '\temit("text", "text", answer, 0, 0, false)',
      '\temit("step_finish", "step-finish", "", 38, 2, true)',
      '}'
    ].join('\n'),'utf8');
    const fakeExe=join(fakeDir,process.platform==='win32'?'fake-opencode.exe':'fake-opencode');
    const built=spawnSync('go',['build','-o',fakeExe,fakeSrcPath],{encoding:'utf8'});
    assert.equal(built.status,0,`go build fake opencode failed: ${built.stderr??built.stdout}`);
    const recordPath=join(dir,'calls.jsonl');
    const violationRecordPath=join(dir,'calls-violation.jsonl');
    const emptyGitConfig=join(dir,'empty-gitconfig');
    await writeFile(emptyGitConfig,'','utf8');
    const providerConfig={cliPath:fakeExe,repositoryPath:dir,gitPath,
      emptyGitConfig,
      env:{DEEPSEEK_API_KEY:'test-key',FAKE_OPENCODE_RECORD:recordPath}};
    const config={providerId:'opencode',issueRef,sourceSha:sha,
      attemptId:'attempt:opencode:integration:1',worktreePath:join(dir,'wt'),
      providerConfig,taskPrompt:'Create the candidate note file.',
      allowedWriteSet:['LOCAL_OPENCODE_L3.md'],timeoutMs:120000,
      systemRoot:process.env.SystemRoot??'C:\\Windows',
      systemDrive:process.env.SystemDrive??'C:'};
    const violationConfig={...config,attemptId:'attempt:opencode:integration:2',
      worktreePath:join(dir,'wt-violation'),
      providerConfig:{...providerConfig,env:{...providerConfig.env,
        FAKE_OPENCODE_RECORD:violationRecordPath,FAKE_OPENCODE_ESCAPE:'1'}}};
    const host=createOpencodeTaskProvider({});
    // Positive: full chain with default operations, real git and a real spawn.
    const pass=await host.execute(config);
    assert.equal(pass.state,'LOCAL_CANDIDATE_READY',
      `${pass.state}:${pass.reason}`);
    assert.equal(pass.reason,'REAL_OPENCODE_CANDIDATE_RECORDED');
    assert.match(pass.candidateCommit,/^[0-9a-f]{40}$/);
    assert.deepEqual(pass.candidatePaths,['LOCAL_OPENCODE_L3.md']);
    assert.equal(pass.agent.answer,'Done.');
    assert.equal(pass.providerProbe.available,true);
    // The observed surface recorded from inside the real child process.
    const taskRecord=JSON.parse(
      (await readFile(recordPath,'utf8')).trim().split('\n').at(-1));
    assert.equal(taskRecord.cwd,config.worktreePath);
    assert.equal(taskRecord.pwd,config.worktreePath);
    assert.equal(taskRecord.oldpwd,config.worktreePath);
    assert.match(taskRecord.xdgConfig,/tally-opencode-attempt-/);
    assert.match(taskRecord.xdgData,/tally-opencode-attempt-/);
    assert.equal(taskRecord.apiKeySet,true);
    assert.ok(taskRecord.argv.includes('--pure'));
    assert.ok(taskRecord.argv.includes('--format'));
    assert.equal(taskRecord.argv[taskRecord.argv.indexOf('--format')+1],'json');
    assert.ok(taskRecord.message.includes('LOCAL_OPENCODE_L3.md'));
    assert.equal((await readFile(recordPath,'utf8')).trim().split('\n').length,2,
      'canary turn + one task turn');
    // Independent readback never trusts agent-asserted values.
    const readback=await host.readback(config,pass);
    assert.equal(readback.verified,true);
    assert.equal(readback.parent,sha);
    assert.deepEqual(readback.paths,['LOCAL_OPENCODE_L3.md']);
    assert.equal(readback.candidateTree,pass.candidateTree);
    // Replay: same attempt id re-reads the durable ref and never relaunches.
    const replay=await host.execute(config);
    assert.equal(replay.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replay.replayed,true);
    assert.equal(replay.candidateCommit,pass.candidateCommit);
    assert.equal((await readFile(recordPath,'utf8')).trim().split('\n').length,2,
      'replay must not spawn');
    // Write-set fence with the real fake binary writing outside the set.
    const violation=await host.execute(violationConfig);
    assert.equal(violation.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(violation.reason,'WRITE_SET_VIOLATION');
    assert.ok(violation.candidatePaths.includes('escape.txt'));
    assert.equal(violation.candidateTree,null);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('task provider cancel accepts the dispatch controller key',async()=>{
  const provider=createOpencodeTaskProvider();
  const controller=new AbortController();
  assert.equal(provider.cancel({controller}),true);
  assert.ok(controller.signal.aborted);
  assert.equal(provider.cancel({controller}),false);
});
