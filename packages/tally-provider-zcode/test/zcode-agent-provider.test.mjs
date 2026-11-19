import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {mkdtemp,mkdir,rm,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createZcodeAgentProvider,createDefaultOperations,
  ZCODE_PROVIDER_CONSTANTS,zcodeBuilderEnvironment,zcodeTaskPrompt,
  parseZcodeAgentResult} from '../src/zcode-agent-provider.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const taskId='473';
const attemptId='attempt:zcode:test:1';
const ALLOWED=['experiments/v2/selected-business-build/LOCAL_ZCODE_L3.md'];
const attemptPin={taskId,attemptId,sourceSha,providerId:'zcode'};
const agentJson=JSON.stringify({sessionId:'sess_test',response:'Done. Candidate written.',
  usage:{totalTokens:1}});
const canaryJson=JSON.stringify({sessionId:'sess_canary',response:'OK'});

function fakeChild(behavior){
  const child=new EventEmitter();child.pid=4242;child.stdout=new EventEmitter();child.stderr=new EventEmitter();
  let closed=false;
  const close=(code,signal)=>{if(!closed){closed=true;setImmediate(()=>child.emit('close',code,signal));}};
  child.kill=()=>{close(null,'SIGTERM');return true;};
  child.stdin={end:()=>setImmediate(()=>behavior?.(close,child))};
  return child;
}
/** Real spawn/argv/env machinery with fake children; probe short-circuited to
 * AVAILABLE and the git record to absent unless a test overrides them. */
const opsWith=(overrides={})=>spawnFn=>({...createDefaultOperations(spawnFn),
  probeProvider:async()=>({available:true,provider:'zcode',version:'0.16.9',
    canary:{sessionId:'sess_probe',response:'OK'}}),
  attemptRecord:async()=>null,...overrides});
async function withProvider(fn,{config={},ops,behavior=(close)=>close(0,null),
  createWorktree=true}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-zcode-provider-test-'));
  try{
    const worktreePath=join(dir,'wt');
    if(createWorktree)await mkdir(worktreePath,{recursive:true});
    const calls=[],kills=[];let lastChild=null;
    const spawnFn=(command,args,options)=>{
      calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
    };
    const provider=createZcodeAgentProvider({spawnFn,
      operations:typeof ops==='function'?ops(spawnFn):
        (ops??createDefaultOperations(spawnFn)),
      now:()=>Date.UTC(2026,10,19),
      killTree:async pid=>{kills.push(pid);lastChild?.kill();return killTree(pid);},
      config:{repositoryPath:dir,cliPath:process.execPath,gitPath:'git.exe',
        credentialsPath:join(dir,'credentials.json'),allowedWriteSet:ALLOWED,...config}});
    const request={taskId,attemptId,sourceSha,prompt:'Create the candidate note file.',
      worktreePath,timeoutMs:1000};
    return await fn({provider,dir,calls,kills,request,spawnFn});
  }finally{await rm(dir,{recursive:true,force:true});}
}
async function until(check,tries=200){
  for(let i=0;i<tries;i++){
    if(await check())return true;
    await new Promise(resolve=>setImmediate(resolve));
  }
  return false;
}

test('probe reports AVAILABLE after version, credentials and canary stages',async()=>{
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
    const provider=createZcodeAgentProvider({spawnFn,
      config:{cliPath:process.execPath,credentialsPath:join(dir,'credentials.json')}});
    const probe=await provider.probe();
    assert.equal(probe.state,'AVAILABLE');
    assert.equal(probe.reason,null);
    assert.equal(probe.evidence.provider,'zcode');
    assert.equal(probe.evidence.version,'0.16.9');
    assert.equal(probe.evidence.canary.sessionId,'sess_canary');
    assert.equal(calls.length,2,'version probe + canary only');
    assert.deepEqual(calls[0].args.slice(1),['version']);
    assert.ok(calls[1].args.includes('--json'));
    assert.ok(calls[1].args.includes('--no-color'));
    assert.equal(calls[1].args[calls[1].args.indexOf('--mode')+1],'yolo');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('missing cli, absent credentials and stalling canary are honest UNAVAILABLE probes',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-zcode-probe-neg-'));
  try{
    const spawnFn=(command,args)=>fakeChild((close,child)=>{
      if(args.includes('version')){child.stdout.emit('data','0.16.9\n');close(0,null);}
      else close(0,null); // canary: exit 0 but no JSON at all
    });
    const make=config=>createZcodeAgentProvider({spawnFn,
      config:{credentialsPath:join(dir,'credentials.json'),...config}});
    const missing=await make({cliPath:join(dir,'absent.mjs')}).probe();
    assert.equal(missing.state,'UNAVAILABLE');
    assert.equal(missing.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(missing.evidence.providerProbe.stage,'VERSION');
    assert.equal(missing.evidence.providerProbe.detail,'MISSING_CLI');
    const credentials=await make({cliPath:process.execPath}).probe();
    assert.equal(credentials.state,'UNAVAILABLE');
    assert.equal(credentials.evidence.providerProbe.stage,'CREDENTIALS');
    assert.equal(credentials.evidence.providerProbe.detail,'CREDENTIALS_ABSENT');
    await writeFile(join(dir,'credentials.json'),
      JSON.stringify({'oauth:active_provider':'bigmodel'}),'utf8');
    const stall=await make({cliPath:process.execPath,canaryTimeoutMs:5}).probe();
    assert.equal(stall.state,'UNAVAILABLE');
    // A canary without a parseable OK response is unavailable — auth failure
    // stalls silently in the real CLI and must never retry into a fake PASS.
    assert.equal(stall.evidence.providerProbe.stage,'CANARY');
    assert.equal(stall.reason,'LOCAL_AGENT_UNAVAILABLE');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('UNAVAILABLE probe never spawns the task process',async()=>{
  await withProvider(async({provider,calls,request})=>{
    const x=await provider.execute(request);
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.evidence.agentLaunch,false);
    assert.equal(x.evidence.providerProbe.stage,'VERSION');
    assert.equal(calls.length,0);
  },{ops:{probeProvider:async()=>({available:false,provider:'zcode',
    reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',outputMatched:false,
    exitCode:null,detail:'MISSING_CLI'}),
    attemptRecord:async()=>null,runAgentProcess:()=>{throw new Error('must not spawn');}}});
});

test('real process exit produces COMPLETED with prompt, cwd, mode and json argv',async()=>{
  await withProvider(async({provider,calls,request,dir})=>{
    const x=await provider.execute(request);
    assert.equal(x.state,'COMPLETED');
    assert.equal(x.reason,'AGENT_TURN_COMPLETED');
    assert.equal(x.evidence.agentLaunch,true);
    assert.equal(x.evidence.process.exitCode,0);
    assert.equal(x.evidence.process.pid,4242);
    assert.equal(x.evidence.agent.sessionId,'sess_test');
    assert.equal(calls.length,1,'probe short-circuited: only the agent spawn');
    const args=calls[0].args;
    assert.equal(calls[0].command,process.execPath,'CLI launched through node');
    assert.equal(args[0],process.execPath);
    assert.equal(args[1],'-p');
    assert.equal(args[args.indexOf('--cwd')+1],join(dir,'wt'));
    assert.equal(args[args.indexOf('--mode')+1],'yolo');
    assert.ok(args.includes('--json'));
    assert.ok(args.includes('--no-color'));
    const prompt=args[args.indexOf('-p')+1];
    for(const path of ALLOWED)assert.ok(prompt.includes(path),`prompt carries ${path}`);
    assert.ok(prompt.includes('Do not run git commit, git push'));
    assert.equal(calls[0].options.cwd,join(dir,'wt'));
    assert.ok(calls[0].options.env.PATHEXT,'builder env is an allowlist');
    assert.ok(!JSON.stringify(x).includes('WOODPECKER_TOKEN'));
  },{ops:opsWith(),behavior:(close,child)=>{child.stdout.emit('data',agentJson);close(0,null);}});
});

test('nonzero process exit is FAILED with the captured exit code',async()=>{
  await withProvider(async({provider,request})=>{
    const x=await provider.execute(request);
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.evidence.process.exitCode,7);
    assert.equal(x.evidence.agent,null);
  },{ops:opsWith(),behavior:close=>close(7,null)});
});

test('timeout terminates the process tree and records UNKNOWN',async()=>{
  await withProvider(async({provider,kills,request})=>{
    const x=await provider.execute({...request,timeoutMs:25});
    assert.equal(x.state,'UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(kills.includes(4242));
    assert.equal(x.evidence.process.exitCode,null);
  },{ops:opsWith(),behavior:()=>{}});
});

test('cancellation via AbortSignal kills the child and records CANCELLED',async()=>{
  await withProvider(async({provider,kills,request})=>{
    const controller=new AbortController();setTimeout(()=>controller.abort(),10);
    const x=await provider.execute({...request,timeoutMs:1000},
      {env:{},signal:controller.signal});
    assert.equal(x.state,'CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));
  },{ops:opsWith(),behavior:()=>{}});
});

test('a Host signal already aborted before dispatch never spawns',async()=>{
  await withProvider(async({provider,calls,request})=>{
    const controller=new AbortController();controller.abort();
    const x=await provider.execute(request,{env:{},signal:controller.signal});
    assert.equal(x.state,'CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.equal(x.evidence.agentLaunch,false);
    assert.equal(calls.length,0);
  },{ops:opsWith({runAgentProcess:()=>{throw new Error('must not spawn');}})});
});

test('spawn error is UNKNOWN and never reads COMPLETED',async()=>{
  await withProvider(async({provider,calls,request})=>{
    const x=await provider.execute(request);
    assert.equal(x.state,'UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    assert.equal(x.evidence.process.status,'UNKNOWN');
    assert.equal(x.evidence.process.exitCode,null);
    assert.equal(calls.length,1);
  },{ops:opsWith(),behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
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

test('task prompt mentions every allowed path exactly once and forbids git push',()=>{
  const prompt=zcodeTaskPrompt({allowedWriteSet:['a.md','b/c.md'],taskPrompt:'Do it.'});
  for(const path of ['a.md','b/c.md'])
    assert.equal(prompt.split(path).length-1,1);
  assert.ok(prompt.includes('Task: Do it.'));
  assert.ok(prompt.includes('Do not run git commit, git push'));
});

test('cancel aborts the live attempt and reports NOT_RUNNING when idle',async()=>{
  await withProvider(async({provider,kills,request})=>{
    const idle=await provider.cancel(attemptPin);
    assert.equal(idle.state,'NOT_RUNNING');
    const executing=provider.execute(request);
    assert.ok(await until(async()=>(await provider.reconcile(attemptPin)).state==='RUNNING'),
      'attempt becomes live');
    const asked=await provider.cancel(attemptPin);
    assert.equal(asked.state,'CANCEL_REQUESTED');
    assert.equal(asked.evidence.attemptId,attemptId);
    const x=await executing;
    assert.equal(x.state,'CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));
    assert.equal((await provider.cancel(attemptPin)).state,'NOT_RUNNING');
  },{ops:opsWith(),behavior:()=>{}});
});

test('reconcile reads the git record: COMPLETED, STALE, occupied, absent',async()=>{
  const inert=record=>createZcodeAgentProvider({spawnFn:()=>fakeChild(()=>{}),
    operations:{probeProvider:async()=>({available:false}),attemptRecord:async()=>record,
      runAgentProcess:()=>{throw new Error('must not spawn');}}});
  await withProvider(async({provider})=>{
    const completed=await provider.reconcile(attemptPin);
    assert.equal(completed.state,'COMPLETED');
    assert.equal(completed.reason,'REAL_ZCODE_CANDIDATE_RECORDED');
    assert.equal(completed.evidence.candidateCommit,'c'.repeat(40));
    assert.equal(completed.evidence.ref,'refs/tally-zcode-candidates/attempt-zcode-test-1');
    const stale=await inert({kind:'STALE'}).reconcile(attemptPin);
    assert.equal(stale.state,'FAILED');
    assert.equal(stale.reason,'CANDIDATE_STALE');
  },{ops:{probeProvider:async()=>({available:false}),attemptRecord:async()=>({kind:'CANDIDATE',
      candidateCommit:'c'.repeat(40),candidateTree:'d'.repeat(40),paths:ALLOWED,
      ref:'refs/tally-zcode-candidates/attempt-zcode-test-1'}),
    runAgentProcess:()=>{throw new Error('must not spawn');}}});
  const occupied=await inert({kind:'WORKTREE_OCCUPIED'}).reconcile(attemptPin);
  assert.equal(occupied.state,'UNKNOWN');
  assert.equal(occupied.reason,'ATTEMPT_STATUS_UNKNOWN');
  const absent=await inert(null).reconcile(attemptPin);
  assert.equal(absent.state,'UNKNOWN');
  assert.equal(absent.reason,'ATTEMPT_RECORD_ABSENT');
});

test('reconcile reports RUNNING only while the attempt is live',async()=>{
  await withProvider(async({provider,request})=>{
    const executing=provider.execute(request);
    assert.ok(await until(async()=>(await provider.reconcile(attemptPin)).state==='RUNNING'),
      'live attempt reads RUNNING');
    provider.cancel(attemptPin);
    const x=await executing;
    assert.equal(x.state,'CANCELLED');
    assert.equal((await provider.reconcile(attemptPin)).state,'UNKNOWN',
      'terminal attempt is no longer live; the git record decides');
  },{ops:opsWith(),behavior:()=>{}});
});

test('invalid attempt inputs never execute',async()=>{
  await withProvider(async({provider,calls,request})=>{
    // Binding-field violations are refused by the contract wrapper before
    // the adapter ever sees them (typed TypeError, nothing fabricated).
    for(const [bad,code] of [[{...request,sourceSha:'not-a-sha'},'PROVIDER_SOURCE_SHA_INVALID'],
      [{...request,attemptId:'bad attempt id with spaces'},'PROVIDER_ATTEMPT_ID_INVALID'],
      [{...request,worktreePath:''},'PROVIDER_WORKTREE_PATH_INVALID']])
      await assert.rejects(()=>provider.execute(bad),{name:'TypeError',message:code});
    // An empty prompt passes the contract shape and is refused by the adapter.
    const x=await provider.execute({...request,prompt:''});
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'ATTEMPT_INPUT_INVALID');
    assert.equal(x.evidence.agentLaunch,false);
    assert.equal(calls.length,0);
  },{ops:{probeProvider:async()=>{throw new Error('probe must not run');},
    attemptRecord:async()=>null,runAgentProcess:()=>{throw new Error('must not spawn');}}});
});

test('a Host-admitted worktree that does not exist is FAILED WORKTREE_ABSENT',async()=>{
  await withProvider(async({provider,calls,request})=>{
    const x=await provider.execute(request);
    assert.equal(x.state,'FAILED');
    assert.equal(x.reason,'WORKTREE_ABSENT');
    assert.equal(x.evidence.agentLaunch,false);
    assert.equal(calls.length,0);
  },{createWorktree:false,ops:{probeProvider:async()=>({available:true,provider:'zcode',
      version:'0.16.9'}),
    attemptRecord:async()=>null,runAgentProcess:()=>{throw new Error('must not spawn');}}});
});

test('exported constants carry the #470 CLI contract vocabulary',()=>{
  assert.equal(ZCODE_PROVIDER_CONSTANTS.providerId,'zcode');
  assert.equal(ZCODE_PROVIDER_CONSTANTS.candidateRefPrefix,'refs/tally-zcode-candidates/');
  assert.equal(ZCODE_PROVIDER_CONSTANTS.canaryPrompt,'Reply with exactly: OK');
});
