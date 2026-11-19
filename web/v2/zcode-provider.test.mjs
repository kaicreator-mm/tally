import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {mkdtemp,rm,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawn,spawnSync} from 'node:child_process';
import {createZcodeProviderHost,ZCODE_PROVIDER_CONSTANTS,
  ZCODE_PROVIDER_CONSTANTS as CONSTANTS_CHECK,resolveZcodeProviderSelection,
  createZcodeJourneyRun,zcodeBuilderEnvironment,zcodeTaskPrompt,
  parseZcodeAgentResult} from './zcode-provider.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/64';
const ALLOWED=['experiments/v2/selected-business-build/LOCAL_ZCODE_L3.md'];
const candidate={writeSetValid:true,paths:ALLOWED,candidateTree:'a'.repeat(40),
  candidateCommit:'b'.repeat(40),candidateRef:'refs/tally-zcode-candidates/test'};
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
    probeProvider:async()=>({available:true,provider:'zcode',version:'0.16.9'}),
    attemptRecord:async()=>null,
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha}),
    finalizeCandidate:async()=>candidate,...overrides
  };
}
function configFor(dir,attemptId='attempt:zcode:test:1'){
  return {issueRef,sourceSha,attemptId,repositoryPath:dir,worktreePath:join(dir,'candidate'),
    cliPath:process.execPath,taskPrompt:'Create the candidate note file.',
    allowedWriteSet:ALLOWED,timeoutMs:1000};
}
async function withHost(fn,{ops=operations(),behavior=(close)=>close(0,null),
  killTree=async()=>true}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-zcode-provider-test-'));
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
  };
  const host=createZcodeProviderHost({spawnFn,operations:ops,now:()=>Date.UTC(2026,10,19),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return killTree(pid);}});
  try{return await fn({host,dir,calls,kills,config:configFor(dir)});}
  finally{await rm(dir,{recursive:true,force:true});}
}
const agentJson=JSON.stringify({sessionId:'sess_test',response:'Done. Candidate written.',
  usage:{totalTokens:1}});
const canaryJson=JSON.stringify({sessionId:'sess_canary',response:'OK'});

test('probe reports available with version after version, credentials and canary stages',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-zcode-probe-'));
  try{
    await writeFile(join(dir,'credentials.json'),
      JSON.stringify({'oauth:active_provider':'bigmodel'}),'utf8');
    const calls=[];
    const behavior=(close,child)=>{
      const args=calls.at(-1).args;
      if(args.includes('version')){child.stdout.emit('data','0.16.9\n');close(0,null);}
      else if(args.includes('-p')){child.stdout.emit('data',canaryJson);close(0,null);}
      else close(1,null);
    };
    const spawnFn=(command,args,options)=>{calls.push({command,args,options});
      return fakeChild(behavior);};
    const host=createZcodeProviderHost({spawnFn});
    const probe=await host.probeProvider({...configFor(dir),
      credentialsPath:join(dir,'credentials.json')});
    assert.equal(probe.available,true);
    assert.equal(probe.provider,'zcode');
    assert.equal(probe.version,'0.16.9');
    assert.equal(probe.canary.sessionId,'sess_canary');
    assert.equal(calls.length,2,'version probe + canary only');
    assert.deepEqual(calls[0].args.slice(1),['version']);
    assert.ok(calls[1].args.includes('--json'));
    assert.ok(calls[1].args.includes('--mode'));
    assert.equal(calls[1].args[calls[1].args.indexOf('--mode')+1],'yolo');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('missing cli, absent credentials and stalling canary are honest unavailable probes',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-zcode-probe-neg-'));
  try{
    const host=createZcodeProviderHost({
      spawnFn:(command,args)=>fakeChild((close,child)=>{
        if(args.includes('version')){child.stdout.emit('data','0.16.9\n');close(0,null);}
        else close(0,null); // canary: exit 0 but no JSON at all
      })});
    const missing=await host.probeProvider({...configFor(dir),cliPath:join(dir,'absent.mjs')});
    assert.equal(missing.available,false);assert.equal(missing.stage,'VERSION');
    assert.equal(missing.detail,'MISSING_CLI');assert.equal(missing.outputMatched,false);
    const credentials=await host.probeProvider({...configFor(dir),
      credentialsPath:join(dir,'credentials.json')});
    assert.equal(credentials.available,false);assert.equal(credentials.stage,'CREDENTIALS');
    assert.equal(credentials.detail,'CREDENTIALS_ABSENT');
    await writeFile(join(dir,'credentials.json'),
      JSON.stringify({'oauth:active_provider':'bigmodel'}),'utf8');
    const stall=await host.probeProvider({...configFor(dir),
      credentialsPath:join(dir,'credentials.json'),canaryTimeoutMs:5});
    assert.equal(stall.available,false);
    // A canary without a parseable OK response is unavailable — auth failure
    // stalls silently in the real CLI and must never retry into a fake PASS.
    assert.ok(['CANARY'].includes(stall.stage));
    assert.equal(stall.reason,'LOCAL_AGENT_UNAVAILABLE');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('unavailable provider never spawns the task process',async()=>{
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.agentLaunch,false);assert.equal(calls.length,0);
    assert.equal(x.providerProbe.stage,'AUTH');
  },{ops:operations({probeProvider:async()=>
    ({available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'AUTH',outputMatched:false})})});
});

test('real process exit produces a candidate with prompt, cwd, mode and json argv',async()=>{
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(x.reason,'REAL_ZCODE_CANDIDATE_RECORDED');
    assert.equal(x.agentLaunch,true);assert.equal(x.process.exitCode,0);
    assert.equal(x.candidateTree,candidate.candidateTree);
    assert.equal(x.agent.sessionId,'sess_test');
    assert.equal(calls.length,1);
    const args=calls[0].args;
    assert.equal(args[0],config.cliPath,'CLI launched through node');
    assert.equal(args[args.indexOf('--cwd')+1],config.worktreePath);
    assert.equal(args[args.indexOf('--mode')+1],'yolo');
    assert.ok(args.includes('--json'));
    const prompt=args[args.indexOf('-p')+1];
    for(const path of ALLOWED)assert.ok(prompt.includes(path),`prompt carries ${path}`);
    assert.ok(prompt.includes('Do not run git commit, git push'));
    assert.ok(!JSON.stringify(x).includes('WOODPECKER_TOKEN'));
  },{behavior:(close,child)=>{child.stdout.emit('data',agentJson);close(0,null);}});
});

test('writes outside the allowed write set are a violation, never a candidate',async()=>{
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'WRITE_SET_VIOLATION');
    assert.equal(x.candidateTree,null);assert.equal(x.taskPass,false);
    assert.deepEqual(x.candidatePaths,['forbidden.txt','escape.md']);
  },{ops:operations({finalizeCandidate:async()=>({writeSetValid:false,
    paths:['forbidden.txt','escape.md']})})});
});

test('nonzero process exit is FAILED with the captured exit code',async()=>{
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.process.exitCode,7);
  },{behavior:close=>close(7,null)});
});

test('timeout terminates the process tree and records UNKNOWN',async()=>{
  await withHost(async({host,kills,config})=>{
    const x=await host.runAttempt({...config,timeoutMs:25});
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('cancellation kills the child and records CANCELLED without PASS',async()=>{
  await withHost(async({host,kills,config})=>{
    const controller=new AbortController();setTimeout(()=>controller.abort(),10);
    const x=await host.runAttempt({...config,timeoutMs:1000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('spawn error is UNKNOWN and an occupied worktree blocks blind relaunch',async()=>{
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(x.agentLaunch,true);
    assert.equal(calls.length,1);
  },{behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
  // durable replay: once the worktree is occupied, a retry never relaunches
  let records=0;
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(calls.length,1);
    const again=await host.runAttempt(config);
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(again.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(calls.length,1,'occupied worktree must not relaunch');
  },{ops:operations({attemptRecord:async()=>records++===0?null:{kind:'WORKTREE_OCCUPIED'}}),
    behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
});

test('a completed attempt replays from git and is never relaunched',async()=>{
  await withHost(async({host,calls,config})=>{
    const replayed=await host.runAttempt(config);
    assert.equal(replayed.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replayed.replayed,true);
    assert.equal(replayed.candidateCommit,'c'.repeat(40));
    assert.equal(calls.length,0,'replay must not spawn');
  },{ops:operations({attemptRecord:async()=>({kind:'CANDIDATE',
    candidateCommit:'c'.repeat(40),candidateTree:'d'.repeat(40),paths:ALLOWED,
    ref:'refs/tally-zcode-candidates/replay'})})});
});

test('a stale candidate ref for another source SHA is NOT_ADMITTED',async()=>{
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'NOT_ADMITTED');
    assert.equal(x.reason,'CANDIDATE_STALE');
    assert.equal(x.agentLaunch,false);assert.equal(x.admitted,false);
  },{ops:operations({attemptRecord:async()=>({kind:'STALE'})})});
});

test('invalid attempt inputs never execute',async()=>{
  await withHost(async({host,calls})=>{
    const x=await host.runAttempt({sourceSha:'not-a-sha',attemptId:'attempt:zcode:x:1'});
    assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,'ATTEMPT_INPUT_INVALID');
    assert.equal(calls.length,0);
  });
});

test('builder environment is an explicit allowlist with no credential inheritance',()=>{
  const env=zcodeBuilderEnvironment({cliPath:process.execPath,
    emptyGitConfig:'C:\\empty\\gitconfig',builderTemp:'C:\\temp'});
  for(const name of ['GITHUB_TOKEN','GH_TOKEN','WOODPECKER_TOKEN','SSH_AUTH_SOCK',
    'SSH_AGENT_PID','DEEPSEEK_API_KEY'])assert.equal(Object.hasOwn(env,name),false);
  // No desktop-inherited ZCODE_* pointers survive; ZCODE_LOCALE is the only
  // intentional zcode variable and is absent unless explicitly configured.
  for(const name of Object.keys(env))
    if(name.startsWith('ZCODE_'))
      assert.fail(`unexpected inherited ZCODE_* var ${name}`);
  assert.equal(env.GIT_CONFIG_NOSYSTEM,'1');
  assert.equal(env.GIT_TERMINAL_PROMPT,'0');
  assert.equal(env.GCM_INTERACTIVE,'never');
  const withLocale=zcodeBuilderEnvironment({emptyGitConfig:'C:\\e',zcodeLocale:'zh-CN'});
  assert.equal(withLocale.ZCODE_LOCALE,'zh-CN');
});

test('selection defaults to codex and admits zcode only when configured and validated',()=>{
  const configured={enabled:true,cliPath:'C:/bin/zcode.mjs'};
  const validated={available:true,provider:'zcode',version:'0.16.9'};
  assert.equal(resolveZcodeProviderSelection({requested:undefined,zcode:configured,
    probe:validated}).reason,'CODEX_DEFAULT');
  assert.equal(resolveZcodeProviderSelection({requested:'codex',zcode:configured,
    probe:validated}).selected,'codex');
  assert.equal(resolveZcodeProviderSelection({requested:'zcode',zcode:undefined,
    probe:validated}).reason,'ZCODE_NOT_CONFIGURED');
  assert.equal(resolveZcodeProviderSelection({requested:'zcode',
    zcode:{enabled:false,cliPath:'C:/bin/zcode.mjs'},probe:validated}).reason,'ZCODE_NOT_CONFIGURED');
  const unverified=resolveZcodeProviderSelection({requested:'zcode',zcode:configured,
    probe:{available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'AUTH'}});
  assert.equal(unverified.selected,'codex');
  assert.equal(unverified.reason,'ZCODE_CAPABILITY_UNVERIFIED');
  assert.equal(unverified.zcodeAdmitted,false);
  const admitted=resolveZcodeProviderSelection({requested:'zcode',zcode:configured,
    probe:validated});
  assert.equal(admitted.selected,'zcode');
  assert.equal(admitted.zcodeAdmitted,true);
  assert.equal(admitted.reason,'ZCODE_VALIDATED');
  // No failure-driven fallback exists: the policy is selection-only and a
  // codex failure has no path into it.
  assert.equal(resolveZcodeProviderSelection({requested:'codex',
    codexFailure:true}).selected,'codex');
});

test('journey run adapter maps attempt states onto the existing run seam',async()=>{
  const states=[['LOCAL_CANDIDATE_READY','PASS','REAL_ZCODE_CANDIDATE_RECORDED'],
    ['LOCAL_ATTEMPT_UNKNOWN','UNKNOWN','ATTEMPT_STATUS_UNKNOWN'],
    ['LOCAL_ATTEMPT_CANCELLED','UNKNOWN','ATTEMPT_CANCELLED'],
    ['LOCAL_ATTEMPT_FAILED','FAIL','WRITE_SET_VIOLATION'],
    ['AGENT_NOT_EXECUTED','FAIL','LOCAL_AGENT_UNAVAILABLE'],
    ['NOT_ADMITTED','FAIL','CANDIDATE_WORKTREE_OVERLAP']];
  for(const [state,status,reason] of states){
    const host={runAttempt:async()=>({state,reason,process:{exitCode:0}})};
    const runner=createZcodeJourneyRun({host,attemptConfig:()=>({})});
    const result=await runner.run('start',{},'C:/evidence');
    assert.equal(result.status,status,`${state} -> ${status}`);
    assert.equal(result.result.reason,reason);
    if(status==='PASS')assert.equal(result.reason,undefined);
    else assert.equal(result.reason,reason);
    assert.equal(result.result.provider,'zcode');
    assert.equal(result.evidenceDir,'C:/evidence');
  }
  const runner=createZcodeJourneyRun({host:{runAttempt:async()=>({})},attemptConfig:()=>({})});
  const other=await runner.run('review-repair',{},{});
  assert.equal(other.status,'UNKNOWN');
  assert.equal(other.reason,'ZCODE_REVIEW_CONSUMER_NOT_MOUNTED');
  assert.equal(other.result,null);
});

test('agent result parsing tolerates noise and never fabricates',()=>{
  const parsed=parseZcodeAgentResult(agentJson);
  assert.equal(parsed.sessionId,'sess_test');
  const noisy='AI SDK Warning: something\n'+agentJson;
  assert.equal(parseZcodeAgentResult(noisy).sessionId,'sess_test');
  assert.equal(parseZcodeAgentResult('no json at all'),null);
  assert.equal(parseZcodeAgentResult('{broken'),null);
  assert.equal(parseZcodeAgentResult(''),null);
});

test('task prompt is bounded and mentions every allowed path exactly once',()=>{
  const prompt=zcodeTaskPrompt({allowedWriteSet:['a.md','b/c.md'],taskPrompt:'Do it.'});
  for(const path of ['a.md','b/c.md'])
    assert.equal(prompt.split(path).length-1,1);
  assert.ok(prompt.includes('Task: Do it.'));
});

test('integration: real git worktree, real process spawn, fake zcode binary end-to-end',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-zcode-integration-'));
  const gitPath=process.env.TALLY_TEST_GIT_PATH??
    (process.platform==='win32'?'C:\\Program Files\\Git\\cmd\\git.exe':'git');
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
    await writeFile(join(dir,'credentials.json'),
      JSON.stringify({'oauth:active_provider':'bigmodel'}),'utf8');
    // Fake zcode binary: a real Node process driven by the provider's own
    // argv contract (version / canary / task with --cwd), performing the
    // workspace edit like the real CLI would. A second variant also writes
    // outside the allowed write set (the builder env is an allowlist, so the
    // escape instruction cannot ride in through environment variables).
    const fakeCliSource=escape=>[
      "import {writeFileSync} from 'node:fs';",
      "const argv=process.argv.slice(2);",
      "if(argv[0]==='version'){process.stdout.write('0.16.9\\n');process.exit(0);}",
      "const prompt=argv[argv.indexOf('-p')+1],cwd=argv[argv.indexOf('--cwd')+1];",
      "if(prompt.includes('Reply with exactly')){",
      "  process.stdout.write(JSON.stringify({sessionId:'sess_canary',response:'OK'})+'\\n');",
      "  process.exit(0);}",
      "writeFileSync(cwd+'/LOCAL_ZCODE_L3.md','zcode candidate note\\n');",
      escape?"writeFileSync(cwd+'/escape.txt','escape\\n');":"",
      "process.stdout.write(JSON.stringify({sessionId:'sess_task',response:'Done.',usage:{}})+'\\n');"
    ].filter(Boolean).join('\n');
    const fakeCli=join(dir,'fake-zcode.mjs');
    await writeFile(fakeCli,fakeCliSource(false),'utf8');
    const fakeViolationCli=join(dir,'fake-zcode-violation.mjs');
    await writeFile(fakeViolationCli,fakeCliSource(true),'utf8');
    const emptyGitConfig=join(dir,'empty-gitconfig');
    await writeFile(emptyGitConfig,'','utf8');
    const config={issueRef,sourceSha:sha,attemptId:'attempt:zcode:integration:1',
      repositoryPath:dir,worktreePath:join(dir,'wt'),cliPath:fakeCli,
      nodePath:process.execPath,credentialsPath:join(dir,'credentials.json'),
      emptyGitConfig,gitPath,taskPrompt:'Create the candidate note file.',
      allowedWriteSet:['LOCAL_ZCODE_L3.md'],timeoutMs:60000,
      systemRoot:process.env.SystemRoot??'C:\\Windows',
      systemDrive:process.env.SystemDrive??'C:'};
    const violationConfig={...config,attemptId:'attempt:zcode:integration:2',
      worktreePath:join(dir,'wt-violation')};
    // Positive: full chain with default operations and real git.
    const host=createZcodeProviderHost({});
    const pass=await host.runAttempt(config);
    assert.equal(pass.state,'LOCAL_CANDIDATE_READY');
    assert.equal(pass.reason,'REAL_ZCODE_CANDIDATE_RECORDED');
    assert.match(pass.candidateCommit,/^[0-9a-f]{40}$/);
    assert.deepEqual(pass.candidatePaths,['LOCAL_ZCODE_L3.md']);
    assert.equal(pass.agent.sessionId,'sess_task');
    const verified=await host.joinVerifyCandidate(config,pass);
    assert.equal(verified.state,'CANDIDATE_READBACK_VERIFIED');
    assert.equal(verified.readback.parent,sha);
    // Replay: same attempt id re-reads the durable ref and never relaunches.
    const replay=await host.runAttempt(config);
    assert.equal(replay.state,'LOCAL_CANDIDATE_READY');
    assert.equal(replay.replayed,true);
    assert.equal(replay.candidateCommit,pass.candidateCommit);
    // Write-set fence with the real fake binary writing outside the set.
    const violation=await host.runAttempt({...violationConfig,cliPath:fakeViolationCli});
    assert.equal(violation.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(violation.reason,'WRITE_SET_VIOLATION');
    assert.ok(violation.candidatePaths.includes('escape.txt'));
    assert.equal(violation.candidateTree,null);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('exported constants match the codex contract vocabulary',()=>{
  assert.equal(CONSTANTS_CHECK.providerId,'zcode');
  assert.equal(ZCODE_PROVIDER_CONSTANTS.candidateRefPrefix,'refs/tally-zcode-candidates/');
  assert.equal(ZCODE_PROVIDER_CONSTANTS.canaryPrompt,'Reply with exactly: OK');
});
