import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {mkdtemp,readFile,rm,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {basename,join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {createGrokTaskProvider,GROK_PROVIDER_CONSTANTS,
  GROK_PROVIDER_CONSTANTS as CONSTANTS_CHECK,resolveGrokProviderSelection,
  grokSpawnEnvironment,grokTaskPrompt,parseGrokVersionIdentity} from './grok-provider.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/64';
const ALLOWED=['GROK_L3.md'];
const candidate={writeSetValid:true,paths:ALLOWED,candidateTree:'a'.repeat(40),
  candidateCommit:'b'.repeat(40),candidateRef:'refs/tally-grok-candidates/test'};
const FAKE_KEY='tally-fake-key-000000000001';

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
    probeProvider:async()=>({available:true,provider:'grok',version:'1.0.87',
      canary:{answer:'OK'}}),
    attemptRecord:async()=>null,
    prepareWorktree:async input=>({worktreePath:input.worktreePath,baseSha:input.sourceSha}),
    finalizeCandidate:async()=>candidate,...overrides
  };
}
function configFor(dir,attemptId='attempt:grok:test:1',overrides={}){
  const base={providerId:'grok',taskId:'task:grok:test',attemptId,sourceSha,
    worktreePath:join(dir,'candidate'),allowedWriteSet:ALLOWED,
    taskPrompt:'Create the candidate note file.',issueRef,
    providerConfig:{binPath:process.execPath,repositoryPath:dir,
      env:{XAI_API_KEY:FAKE_KEY}},timeoutMs:1000};
  // overrides.providerConfig REPLACES the base provider config entirely, so a
  // config without env.XAI_API_KEY really exercises credential absence.
  return {...base,...overrides,providerConfig:overrides.providerConfig??base.providerConfig};
}
async function withProvider(fn,{ops=operations(),behavior=(close)=>close(0,null),
  killTree=async()=>true}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-grok-provider-test-'));
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
  };
  const provider=createGrokTaskProvider({spawnFn,operations:ops,now:()=>Date.UTC(2026,10,19),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return killTree(pid);}});
  try{return await fn({provider,dir,calls,kills,config:configFor(dir)});}
  finally{await rm(dir,{recursive:true,force:true});}
}

test('probe passes VERSION, CREDENTIALS and CANARY tiers with the observed argv shape',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-grok-probe-'));
  try{
    const calls=[];
    const behavior=(close,child)=>{
      const args=calls.at(-1).args;
      if(args.includes('--version')){child.stdout.emit('data','grok 1.0.87 (929e6dd187b1)\n');close(0,null);}
      else if(args.some(a=>typeof a==='string'&&a.startsWith('--single='))){
        child.stdout.emit('data','OK\n');close(0,null);
      }else close(2,null);
    };
    const spawnFn=(command,args,options)=>{calls.push({command,args,options});
      return fakeChild(behavior);};
    const provider=createGrokTaskProvider({spawnFn});
    const probe=await provider.probe(configFor(dir));
    assert.equal(probe.available,true);
    assert.equal(probe.provider,'grok');
    assert.equal(probe.version,'1.0.87');
    assert.equal(probe.canary.answer,'OK');
    assert.equal(calls.length,2,'version probe + canary only');
    assert.equal(calls[0].args[0],'--version');
    const canaryArgs=calls[1].args;
    assert.equal(canaryArgs[0],'--no-auto-update');
    assert.equal(canaryArgs[canaryArgs.length-1],'--single=Reply with exactly: OK');
    assert.ok(!canaryArgs.includes('-m'),'canary carries zero expectation overlay');
    assert.ok(!canaryArgs.includes('--permission-mode'));
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('probe negative tiers are honest and never fall through to a spawn',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-grok-probe-neg-'));
  try{
    const calls=[];
    const spawnFn=(command,args,options)=>{calls.push({command,args,options});
      return fakeChild((close,child)=>{
        const kind=spawnFn.mode;
        if(args.includes('--version')){
          if(kind==='unparseable'){child.stdout.emit('data','Grok Build TUI banner\n');close(0,null);}
          else if(kind==='mismatch'){child.stdout.emit('data','grok 0.1.4 (stale1234)\n');close(0,null);}
          else if(kind==='versionfail'){close(3,null);}
          else {child.stdout.emit('data','grok 1.0.87 (929e6dd187b1)\n');close(0,null);}
        }else if(kind==='stall'){} // canary never finishes
        else if(kind==='unsatisfied'){child.stdout.emit('data','I cannot do that\n');close(0,null);}
        else {child.stdout.emit('data','OK\n');close(0,null);}
      });};
    spawnFn.mode='ok';
    const provider=createGrokTaskProvider({spawnFn});
    const missing=await provider.probe(configFor(dir,
      'attempt:grok:p1',{providerConfig:{binPath:join(dir,'absent.exe')}}));
    assert.equal(missing.available,false);
    assert.equal(missing.stage,'VERSION');
    assert.equal(missing.detail,'MISSING_CLI');
    assert.equal(calls.length,0,'missing CLI must not spawn anything');
    const config=configFor(dir,'attempt:grok:p2',{providerConfig:{
      binPath:process.execPath,env:{XAI_API_KEY:FAKE_KEY}}});
    spawnFn.mode='unparseable';
    const unparseable=await provider.probe(config);
    assert.equal(unparseable.stage,'VERSION');
    assert.equal(unparseable.detail,'VERSION_UNPARSEABLE');
    spawnFn.mode='mismatch';
    const mismatch=await provider.probe(config);
    assert.equal(mismatch.stage,'VERSION');
    assert.equal(mismatch.detail,'VERSION_MISMATCH');
    assert.equal(mismatch.version,'0.1.4','stale dist-tag version is reported, never guessed');
    spawnFn.mode='ok';
    const noCredentials=await provider.probe({...config,
      providerConfig:{binPath:process.execPath}});
    assert.equal(noCredentials.available,false);
    assert.equal(noCredentials.stage,'CREDENTIALS');
    assert.equal(noCredentials.detail,'CREDENTIALS_ABSENT');
    assert.equal(calls.length,3,'version probes ran but no canary after absent credentials');
    spawnFn.mode='stall';
    const stall=await provider.probe({...config,
      providerConfig:{binPath:process.execPath,env:{XAI_API_KEY:FAKE_KEY},canaryTimeoutMs:5}});
    assert.equal(stall.available,false);
    assert.equal(stall.stage,'CANARY');
    assert.equal(stall.detail,'CREDENTIALS_OR_PROVIDER_STALL');
    spawnFn.mode='unsatisfied';
    const unsatisfied=await provider.probe({...config,
      providerConfig:{binPath:process.execPath,env:{XAI_API_KEY:FAKE_KEY},canaryTimeoutMs:5000}});
    assert.equal(unsatisfied.available,false);
    assert.equal(unsatisfied.stage,'CANARY');
    assert.equal(unsatisfied.detail,'CANARY_UNSATISFIED');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('unavailable provider never spawns the task process',async()=>{
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.agentLaunch,false);assert.equal(calls.length,0);
    assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
    assert.equal(x.providerProbe.stage,'CREDENTIALS');
  },{ops:operations({probeProvider:async()=>
    ({available:false,provider:'grok',reason:'LOCAL_AGENT_UNAVAILABLE',
      stage:'CREDENTIALS',detail:'CREDENTIALS_ABSENT'})})});
});

test('real process exit produces a candidate with the observed argv, cwd and env',async()=>{
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(x.reason,'REAL_GROK_CANDIDATE_RECORDED');
    assert.equal(x.provider,'grok');assert.equal(x.providerId,'grok');
    assert.equal(x.taskId,'task:grok:test');
    assert.equal(x.agentLaunch,true);assert.equal(x.replayFixture,false);
    assert.equal(x.process.exitCode,0);assert.equal(x.process.pid,4242);
    assert.equal(x.candidateTree,candidate.candidateTree);
    assert.equal(x.candidateCommit,candidate.candidateCommit);
    assert.equal(x.candidateRef,'refs/tally-grok-candidates/test');
    assert.equal(x.agent.answer,'Candidate written.');
    assert.equal(x.taskPass,false);assert.equal(x.releasePass,false);
    assert.equal(x.authorized,false);
    assert.equal(calls.length,1);
    const args=calls[0].args;
    assert.equal(calls[0].command,process.execPath,'binPath launched directly');
    assert.equal(args[0],'--no-auto-update','auto-update is always forced off');
    const single=args[args.length-1];
    assert.ok(single.startsWith('--single='),'prompt rides the attached-value flag');
    for(const path of ALLOWED)assert.ok(single.includes(path),`prompt carries ${path}`);
    assert.ok(single.includes('Do not run git commit, git push'));
    assert.equal(calls[0].options.cwd,config.worktreePath,'workspace pinned via spawn cwd');
    assert.ok(!args.includes('--cwd'),'--cwd is declared but unobserved and never sent');
    const env=calls[0].options.env;
    assert.equal(env.XAI_API_KEY,FAKE_KEY);
    assert.ok(env.GROK_HOME&&env.GROK_HOME!=='EVIL_HOME','GROK_HOME forced to fresh temp home');
    assert.match(basename(env.GROK_HOME),/^tally-grok-home-/);
    assert.ok(x.stdoutTail.includes('Candidate written.'));
  },{behavior:(close,child)=>{child.stdout.emit('data','Candidate written.\n');close(0,null);}});
});

test('task argv carries model and permission-mode only when configured',async()=>{
  await withProvider(async({provider,calls,config})=>{
    await provider.execute({...config,providerConfig:{...config.providerConfig,
      model:'grok-4.7',permissionMode:'acceptEdits'}});
    const args=calls[0].args;
    assert.equal(args[0],'--no-auto-update');
    assert.equal(args[1],'-m');assert.equal(args[2],'grok-4.7');
    assert.equal(args[3],'--permission-mode');assert.equal(args[4],'acceptEdits');
    assert.ok(args[5].startsWith('--single='),'prompt is the last argv element');
  });
});

test('invalid attempt inputs and configs never execute',async()=>{
  await withProvider(async({provider,calls,config,dir})=>{
    const cases=[
      [{...config,sourceSha:'not-a-sha'},'INPUT_FIELDS_INVALID'],
      [{...config,attemptId:'-bad'},'INPUT_FIELDS_INVALID'],
      [{...config,allowedWriteSet:[]},'INPUT_FIELDS_INVALID'],
      [{...config,taskPrompt:''},'INPUT_FIELDS_INVALID'],
      [{...config,providerId:'zcode'},'PROVIDER_ID_MISMATCH'],
      [{...config,providerConfig:{...config.providerConfig,permissionMode:'yolo'}},
        'PERMISSION_MODE_INVALID'],
      [{...config,providerConfig:{...config.providerConfig,repositoryPath:undefined}},
        'INPUT_FIELDS_INVALID']];
    for(const [input,detail] of cases){
      const x=await provider.execute(input);
      assert.equal(x.state,'AGENT_NOT_EXECUTED');
      assert.equal(x.reason,'ATTEMPT_INPUT_INVALID');
      assert.equal(x.detail,detail);
    }
    assert.equal(calls.length,0);
  });
});

test('writes outside the allowed write set are a violation, never a candidate',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'WRITE_SET_VIOLATION');
    assert.equal(x.candidateTree,null);assert.equal(x.candidateCommit,null);
    assert.deepEqual(x.candidatePaths,['escape.txt']);
    assert.equal(x.taskPass,false);
  },{ops:operations({finalizeCandidate:async()=>({writeSetValid:false,
    paths:['escape.txt']})}),
    behavior:(close,child)=>{child.stdout.emit('data','Done, wrote the note.\n');close(0,null);}});
});

test('nonzero process exit is FAILED with the captured exit code and stderr tail',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.process.exitCode,7);
    assert.ok(x.stderrTail.includes('Not signed in.'));
  },{behavior:(close,child)=>{child.stderr.emit('data','Not signed in. To authenticate without a browser, run:\n  grok login --device-code\n');close(7,null);}});
});

test('exit 0 with empty stdout is a contract violation judged FAILED (fail-safe)',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_CONTRACT_UNFULFILLED');
    assert.equal(x.agent.answer,null);
  },{behavior:close=>close(0,null)});
});

test('timeout terminates the process tree and records UNKNOWN',async()=>{
  await withProvider(async({provider,kills,config})=>{
    const x=await provider.execute({...config,timeoutMs:25});
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(kills.includes(4242));
    assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('signal cancellation kills the child and records CANCELLED without PASS',async()=>{
  await withProvider(async({provider,kills,config})=>{
    const controller=new AbortController();setTimeout(()=>controller.abort(),10);
    const x=await provider.execute({...config,timeoutMs:1000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('cancel(input) is the single signal channel into an in-flight attempt',async()=>{
  await withProvider(async({provider,kills,config})=>{
    const running=provider.execute({...config,timeoutMs:5000});
    await new Promise(resolve=>setTimeout(resolve,5));
    const stop=provider.cancel({attemptId:config.attemptId});
    assert.equal(stop.cancelled,true);assert.equal(stop.channel,'signal');
    const x=await running;
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));
    assert.deepEqual(provider.cancel({attemptId:config.attemptId}),
      {cancelled:false,attemptId:config.attemptId,reason:'ATTEMPT_NOT_IN_FLIGHT'});
    assert.equal(provider.cancel({attemptId:'-bad'}).reason,'ATTEMPT_INPUT_INVALID');
  },{behavior:()=>{}});
});

test('spawn error is UNKNOWN and an occupied worktree blocks blind relaunch',async()=>{
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(x.agentLaunch,true);
    assert.equal(calls.length,1);
  },{behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
  let records=0;
  await withProvider(async({provider,calls,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(calls.length,1);
    const again=await provider.execute(config);
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(again.reason,'ATTEMPT_STATUS_UNKNOWN');
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
    assert.equal(replayed.candidateRef,'refs/tally-grok-candidates/replay');
    assert.equal(calls.length,0,'replay must not spawn');
  },{ops:operations({attemptRecord:async()=>({kind:'CANDIDATE',
    candidateCommit:'c'.repeat(40),candidateTree:'d'.repeat(40),paths:ALLOWED,
    ref:'refs/tally-grok-candidates/replay'})})});
});

test('a stale candidate ref for another source SHA is NOT_ADMITTED',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'NOT_ADMITTED');
    assert.equal(x.reason,'CANDIDATE_STALE');
    assert.equal(x.agentLaunch,false);assert.equal(x.admitted,false);
  },{ops:operations({attemptRecord:async()=>({kind:'STALE'})})});
});

test('spawn env is a strict allowlist with forced GROK_HOME',()=>{
  const ownerEnv={PATH:'/bin:/usr/bin',HOME:'/home/owner',DEEPSEEK_API_KEY:'owner-secret',
    GITHUB_TOKEN:'owner-token',GROK_HOME:'/home/owner/.grok',SYSTEMROOT:'C:\\WINDOWS'};
  const env=grokSpawnEnvironment({grokHome:'/tmp/tally-grok-home-x',
    env:{XAI_API_KEY:'injected-key'}},ownerEnv);
  assert.equal(env.PATH,'/bin:/usr/bin');
  assert.equal(env.HOME,'/home/owner');
  assert.equal(env.XAI_API_KEY,'injected-key','credentials only via explicit injection');
  assert.equal(env.GROK_HOME,'/tmp/tally-grok-home-x','owner GROK_HOME never survives');
  assert.equal(Object.hasOwn(env,'DEEPSEEK_API_KEY'),false);
  assert.equal(Object.hasOwn(env,'GITHUB_TOKEN'),false);
  const forced=grokSpawnEnvironment({grokHome:'/tmp/home2',
    env:{XAI_API_KEY:'k',GROK_HOME:'EVIL_HOME'}},ownerEnv);
  assert.equal(forced.GROK_HOME,'/tmp/home2','config.env GROK_HOME is overridden');
  assert.throws(()=>grokSpawnEnvironment({env:{}},ownerEnv),/GROK_HOME_REQUIRED/);
});

test('failure tails redact explicitly injected secrets',async()=>{
  await withProvider(async({provider,config})=>{
    const x=await provider.execute(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.ok(!x.stderrTail.includes(FAKE_KEY),'injected key value must never leak');
    assert.ok(x.stderrTail.includes('[REDACTED]'));
  },{behavior:(close,child)=>{child.stderr.emit('data','auth failed: '+FAKE_KEY+'\n');close(7,null);}});
});

test('selection defaults to codex and admits grok only when configured and validated',()=>{
  const configured={enabled:true,binPath:'C:/Users/15549/.grok/bin/grok.exe'};
  const validated={available:true,provider:'grok',version:'1.0.87'};
  assert.equal(resolveGrokProviderSelection({requested:undefined,grok:configured,
    probe:validated}).reason,'CODEX_DEFAULT');
  assert.equal(resolveGrokProviderSelection({requested:'codex',grok:configured,
    probe:validated}).selected,'codex');
  assert.equal(resolveGrokProviderSelection({requested:'grok',grok:undefined,
    probe:validated}).reason,'GROK_NOT_CONFIGURED');
  assert.equal(resolveGrokProviderSelection({requested:'grok',
    grok:{enabled:false,binPath:'C:/grok.exe'},probe:validated}).reason,'GROK_NOT_CONFIGURED');
  const unverified=resolveGrokProviderSelection({requested:'grok',grok:configured,
    probe:{available:false,provider:'grok',reason:'LOCAL_AGENT_UNAVAILABLE',
      stage:'CREDENTIALS',detail:'CREDENTIALS_ABSENT'}});
  assert.equal(unverified.selected,'codex');
  assert.equal(unverified.reason,'GROK_CAPABILITY_UNVERIFIED');
  assert.equal(unverified.grokAdmitted,false);
  const admitted=resolveGrokProviderSelection({requested:'grok',grok:configured,
    probe:validated});
  assert.equal(admitted.selected,'grok');
  assert.equal(admitted.grokAdmitted,true);
  assert.equal(admitted.reason,'GROK_VALIDATED');
  assert.equal(resolveGrokProviderSelection({requested:'grok',grok:configured,
    probe:undefined}).probe,null);
  // No failure-driven fallback exists: the policy is selection-only and a
  // codex failure has no path into it.
  assert.equal(resolveGrokProviderSelection({requested:'codex',
    codexFailure:true}).selected,'codex');
});

test('version identity parsing rejects banners and never invents fields',()=>{
  assert.deepEqual(parseGrokVersionIdentity('grok 1.0.87 (929e6dd187b1)\n'),
    {name:'grok',version:'1.0.87',commit:'929e6dd187b1'});
  assert.deepEqual(parseGrokVersionIdentity('grok 1.0.87\n'),
    {name:'grok',version:'1.0.87',commit:null});
  assert.equal(parseGrokVersionIdentity('Grok Build TUI\nmore\n'),null);
  assert.equal(parseGrokVersionIdentity(''),null);
  assert.equal(parseGrokVersionIdentity('grok\n'),null);
});

test('task prompt mentions every allowed path exactly once',()=>{
  const prompt=grokTaskPrompt({allowedWriteSet:['a.md','b/c.md'],taskPrompt:'Do it.'});
  for(const path of ['a.md','b/c.md'])
    assert.equal(prompt.split(path).length-1,1);
  assert.ok(prompt.includes('Task: Do it.'));
});

test('exported constants match the unified grok contract vocabulary',()=>{
  assert.equal(CONSTANTS_CHECK.providerId,'grok');
  assert.equal(GROK_PROVIDER_CONSTANTS.candidateRefPrefix,'refs/tally-grok-candidates/');
  assert.equal(GROK_PROVIDER_CONSTANTS.canaryPrompt,'Reply with exactly: OK');
  assert.equal(GROK_PROVIDER_CONSTANTS.pinnedVersion,'1.0.87');
  assert.deepEqual([...GROK_PROVIDER_CONSTANTS.permissionModes],
    ['default','acceptEdits','auto','dontAsk','bypassPermissions','plan']);
});

test('integration: real git worktree, real processes, fake grok binary end-to-end',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-grok-integration-'));
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
    // Fake grok binary: a real Node process mimicking the fakegrok contract
    // (testdata/fakegrok) — --version identity line, keyless exit 1 with the
    // observed stderr shape, canary OK, cwd-bound task writes. The violation
    // variant also writes outside the allowed write set. Production argv is
    // [binPath, ...]; binArgs is the documented launcher prefix so the fake
    // Node script can stand in for the native binary through the DEFAULT
    // spawn path.
    const fakeCliSource=escape=>[
      "import {writeFileSync} from 'node:fs';",
      "const argv=process.argv.slice(2);",
      "if(argv[0]==='--version'){",
      "  process.stdout.write('grok 1.0.87 (0000000fakegrok)\\n');process.exit(0);}",
      "if(argv[0]!=='--no-auto-update'){process.stderr.write('unexpected '+argv[0]+'\\n');process.exit(2);}",
      "let prompt=null;for(let i=1;i<argv.length;i++)if(argv[i].startsWith('--single='))prompt=argv[i].slice(9);",
      "if(prompt===null){process.stderr.write('missing --single=<prompt>\\n');process.exit(2);}",
      "if(process.env.TALLY_FAKE_GROK_RECORD){",
      "  writeFileSync(process.env.TALLY_FAKE_GROK_RECORD,JSON.stringify({argv:process.argv,",
      "    cwd:process.cwd(),grok_home:process.env.GROK_HOME??null,",
      "    xai_api_key_set:!!process.env.XAI_API_KEY,",
      "    deepseek_key_set:!!process.env.DEEPSEEK_API_KEY,",
      "    model:argv.includes('-m')?argv[argv.indexOf('-m')+1]:'',",
      "    permission_mode:argv.includes('--permission-mode')?argv[argv.indexOf('--permission-mode')+1]:'',",
      "    prompt})+'\\n',{flag:'a'});}",
      "if(!process.env.XAI_API_KEY){",
      "  process.stderr.write('Not signed in. To authenticate without a browser, run:\\n  grok login --device-code\\n');",
      "  process.exit(1);}",
      "if(prompt.includes('Reply with exactly')){process.stdout.write('OK\\n');process.exit(0);}",
      "writeFileSync('GROK_L3.md','grok candidate note\\n');",
      escape?"writeFileSync('escape.txt','escape\\n');":'',
      "process.stdout.write('Candidate note written.\\n');"
    ].join('\n');
    const fakeCli=join(dir,'fake-grok.mjs');
    await writeFile(fakeCli,fakeCliSource(false),'utf8');
    const fakeViolationCli=join(dir,'fake-grok-violation.mjs');
    await writeFile(fakeViolationCli,fakeCliSource(true),'utf8');
    const emptyGitConfig=join(dir,'empty-gitconfig');
    await writeFile(emptyGitConfig,'','utf8');
    const recordPath=join(dir,'fake-grok-records.ndjson');
    const provider=createGrokTaskProvider({});
    const baseConfig={providerId:'grok',taskId:'task:grok:int',sourceSha:sha,
      allowedWriteSet:['GROK_L3.md'],taskPrompt:'Create the candidate note file.',
      issueRef,timeoutMs:60000};
    const config={...baseConfig,attemptId:'attempt:grok:integration:1',
      worktreePath:join(dir,'wt'),
      providerConfig:{binPath:process.execPath,binArgs:[fakeCli],repositoryPath:dir,
        gitPath,emptyGitConfig,
        env:{XAI_API_KEY:FAKE_KEY,TALLY_FAKE_GROK_RECORD:recordPath}}};
    // Negative first: without injected credentials the CREDENTIALS tier fails
    // honestly and nothing is ever spawned.
    const noCredentials=await provider.execute({...config,
      providerConfig:{binPath:process.execPath,binArgs:[fakeCli],repositoryPath:dir,
        gitPath,emptyGitConfig}});
    assert.equal(noCredentials.state,'AGENT_NOT_EXECUTED');
    assert.equal(noCredentials.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(noCredentials.providerProbe.stage,'CREDENTIALS');
    assert.equal(noCredentials.providerProbe.detail,'CREDENTIALS_ABSENT');
    // Positive: full chain with default operations and real git.
    const pass=await provider.execute(config);
    assert.equal(pass.state,'LOCAL_CANDIDATE_READY');
    assert.equal(pass.reason,'REAL_GROK_CANDIDATE_RECORDED');
    assert.equal(pass.providerProbe.available,true);
    assert.equal(pass.providerProbe.version,'1.0.87');
    assert.match(pass.candidateCommit,/^[0-9a-f]{40}$/);
    assert.deepEqual(pass.candidatePaths,['GROK_L3.md']);
    assert.equal(pass.agent.answer,'Candidate note written.');
    assert.equal(pass.process.exitCode,0);
    // Independent readback re-derives parent/tree/paths from the repository.
    const readback=await provider.readback(config,{candidateCommit:pass.candidateCommit});
    assert.equal(readback.verified,true);
    assert.equal(readback.parent,sha);
    assert.equal(readback.writeSetValid,true);
    assert.deepEqual(readback.paths,['GROK_L3.md']);
    // Replay: same attempt id re-reads the durable ref and never relaunches.
    const recordsBefore=(await readFile(recordPath,'utf8')).split('\n').filter(Boolean).length;
    const replay=await provider.execute(config);
    assert.equal(replay.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replay.replayed,true);
    assert.equal(replay.candidateCommit,pass.candidateCommit);
    const recordsAfterReplay=(await readFile(recordPath,'utf8')).split('\n').filter(Boolean).length;
    assert.equal(recordsAfterReplay,recordsBefore,'replay must not execute the agent');
    // Observation record (fakegrok contract): argv shape, cwd pinning via
    // process cwd, fresh per-attempt GROK_HOME, env allowlist isolation.
    const records=(await readFile(recordPath,'utf8')).split('\n').filter(Boolean)
      .map(line=>JSON.parse(line));
    const taskRecord=records.find(record=>!record.prompt.includes('Reply with exactly'));
    assert.ok(taskRecord,'task invocation was recorded');
    assert.equal(taskRecord.cwd,join(dir,'wt'),'workspace pinned via process cwd');
    assert.equal(taskRecord.argv[2],'--no-auto-update');
    assert.equal(taskRecord.argv[taskRecord.argv.length-1].startsWith('--single='),true);
    assert.equal(taskRecord.xai_api_key_set,true);
    assert.equal(taskRecord.deepseek_key_set,false,'owner env credentials never reach the agent');
    assert.match(basename(taskRecord.grok_home),/^tally-grok-home-/);
    assert.ok(!taskRecord.grok_home.includes('.grok'));
    // Write-set fence with the real fake binary writing outside the set.
    const violation=await provider.execute({...config,attemptId:'attempt:grok:integration:2',
      worktreePath:join(dir,'wt-violation'),
      providerConfig:{...config.providerConfig,binArgs:[fakeViolationCli]}});
    assert.equal(violation.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(violation.reason,'WRITE_SET_VIOLATION');
    assert.ok(violation.candidatePaths.includes('escape.txt'));
    assert.equal(violation.candidateTree,null);
    // Occupied worktree without a candidate ref never relaunches blindly.
    assert.equal(gitSync(['worktree','add','--detach',join(dir,'wt-crashed'),sha]).status,0);
    const crashedRecords=(await readFile(recordPath,'utf8')).split('\n').filter(Boolean).length;
    const crashed=await provider.execute({...config,attemptId:'attempt:grok:integration:3',
      worktreePath:join(dir,'wt-crashed')});
    assert.equal(crashed.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(crashed.reason,'ATTEMPT_STATUS_UNKNOWN');
    const crashedRecordsAfter=(await readFile(recordPath,'utf8')).split('\n').filter(Boolean).length;
    assert.equal(crashedRecordsAfter,crashedRecords,'occupied worktree must not spawn');
  }finally{await rm(dir,{recursive:true,force:true});}
});
