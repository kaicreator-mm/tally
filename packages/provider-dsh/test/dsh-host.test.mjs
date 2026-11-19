import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {execFileSync,spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdtemp,mkdir,rm,writeFile,readFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createLocalDshHost,createDefaultDshOperations,LOCAL_DSH_CONSTANTS}
  from '../src/dsh-host.mjs';
import {createMemoryCasStoreRegistry} from './memory-cas-store.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/64';
const builderSid='S-1-5-21-2839165489-1832998833-1574237796-1003';
const API_KEY='test-dsh-secret-0123456789abcdef';
const candidateResult={writeSetValid:true,paths:[LOCAL_DSH_CONSTANTS.candidateRelativePath],
  candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),
  candidateRef:LOCAL_DSH_CONSTANTS.candidateRefPrefix+'test'};
const attemptKey=id=>LOCAL_DSH_CONSTANTS.attemptKeyPrefix+id;
const FAKE_DSH=fileURLToPath(new URL('./fake-dsh.mjs',import.meta.url));
const answerBehavior=(close,child)=>{
  child.stdout.emit('data',Buffer.from('FAKE_DSH_FINAL_ANSWER: task complete.\n'));
  close(0,null);
};

function fakeChild(behavior){
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
  child.kill=()=>{doClose();return true;};
  child.stdin={end:()=>setImmediate(()=>behavior?.(doClose,child))};
  return child;
}
function operations(overrides={}){
  return {
    probeProvider:async()=>({available:true,provider:'dsh'}),
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha,
      dshHome:join(config.repositoryPath,'dsh-home'),
      patchPath:join(config.repositoryPath,'dsh-home','tally-dsh.patch.yml')}),
    proveConfinement:async config=>({verified:true,builderPrincipal:config.builderPrincipal,
      builderSid:config.builderSid,hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'}),
    finalizeCandidate:async()=>candidateResult,...overrides
  };
}
function configFor(dir,attemptId='attempt:dsh:test:1',overrides={}){
  return {issueRef,sourceSha,attemptId,hostStorePath:join(dir,'host.sqlite'),
    repositoryPath:dir,worktreePath:join(dir,'candidate'),binPath:'C:\\Public\\dsh.exe',
    builderProfile:'C:\\Users\\CodexSandboxOffline',builderTemp:join(dir,'tmp'),
    emptyGitConfig:join(dir,'empty-gitconfig'),operatorHome:'C:\\Users\\operator',
    hiddenFixture:join(dir,'hidden'),builderPrincipal:'nilar\\codexsandboxoffline',builderSid,
    verifierSid:'S-1-5-21-2839165489-1832998833-1574237796-1004',
    timeoutMs:1000,pipeGraceMs:60,permissionMode:'workspace-write',
    pinnedVersion:'0.2.2-rc.5',deepseekApiKey:API_KEY,...overrides};
}
async function withHost(fn,{ops=operations(),behavior=answerBehavior,
  registry=createMemoryCasStoreRegistry()}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-host-test-'));
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
  };
  const host=createLocalDshHost({spawnFn,operations:ops,now:()=>Date.UTC(2026,10,19),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return true;},
    openStore:registry.open});
  try{return await fn({host,dir,calls,kills,config:configFor(dir),registry,
    lastChildRef:()=>lastChild});}
  finally{await rm(dir,{recursive:true,force:true});}
}
async function readAttemptRecord(registry,path,attemptId){
  const store=await registry.open({path});
  try{return (await store.get(attemptKey(attemptId)))?.value;}finally{await store.close();}
}
const headlessCalls=calls=>calls.filter(c=>c.args.includes('headless'));

test('happy path: exact candidate, restricted env, durable replay, no secret leakage',async()=>{
  await withHost(async({host,dir,calls,config,registry})=>{
    const first=await host.runAttempt(config);
    assert.equal(first.state,'LOCAL_CANDIDATE_READY');
    assert.equal(first.reason,'REAL_DSH_CANDIDATE_RECORDED');
    assert.equal(first.agentLaunch,true);assert.equal(first.taskPass,false);
    assert.equal(first.candidateTree,candidateResult.candidateTree);
    assert.equal(first.candidateRef,candidateResult.candidateRef);
    assert.deepEqual(first.candidatePaths,[LOCAL_DSH_CONSTANTS.candidateRelativePath]);
    assert.equal(first.process.exitCode,0);
    assert.ok(/^[0-9a-f]{64}$/.test(first.finalAnswerSha256??''));
    const exec=headlessCalls(calls).at(-1);
    assert.ok(exec,'headless exec spawn recorded');
    assert.equal(exec.command,config.binPath);
    const patchIndex=exec.args.indexOf('--patch');
    assert.ok(patchIndex>0,'patch overlay flag present');
    assert.equal(exec.args[patchIndex+1],join(dir,'dsh-home','tally-dsh.patch.yml'));
    const instruction=exec.args.at(-1);
    assert.ok(instruction.includes(LOCAL_DSH_CONSTANTS.candidateRelativePath));
    assert.ok(instruction.includes('disposable Git worktree'));
    assert.equal(exec.options.cwd,config.worktreePath);
    const env=exec.options.env;
    assert.equal(env.DSH_HOME,join(dir,'dsh-home'));
    assert.equal(env.DSH_TELEMETRY_MODE,'DISABLED');
    assert.equal(env.DSH_PERMISSION_MODE,'workspace-write');
    assert.equal(env.DEEPSEEK_API_KEY,API_KEY,'key only via explicit config');
    for(const forbidden of ['GITHUB_TOKEN','GH_TOKEN','WOODPECKER_TOKEN','SSH_AUTH_SOCK','SSH_AGENT_PID'])
      assert.equal(Object.hasOwn(env,forbidden),false);
    const second=await host.runAttempt(config);
    assert.equal(second.state,'LOCAL_CANDIDATE_READY');
    assert.equal(headlessCalls(calls).length,1,
      'restart must not relaunch a completed non-idempotent attempt');
    const record=await readAttemptRecord(registry,config.hostStorePath,config.attemptId);
    assert.equal(record.phase,'CANDIDATE_READY');
    assert.equal(record.permissionMode,'workspace-write');
    assert.equal(record.sandboxWrapper,'DIRECT_SPAWN_CONFINEMENT_REFUSED_ON_REAL_HOST');
    assert.equal(JSON.stringify(record).includes(API_KEY),false,
      'credential value never recorded');
  });
});

test('provider unavailable (CLI missing / version drift / no credential) never spawns',async()=>{
  for(const probe of [
    {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',spawnError:true},
    {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
      observedVersion:'9.9.9-x',pinnedVersion:'0.2.2-rc.5'},
    {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'AUTH',
      credentialEnv:'DEEPSEEK_API_KEY',credentialPresent:false}
  ])await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'AGENT_NOT_EXECUTED');assert.equal(x.reason,'LOCAL_AGENT_UNAVAILABLE');
    assert.equal(x.agentLaunch,false);
    assert.equal(headlessCalls(calls).length,0);
  },{ops:operations({probeProvider:async()=>probe})});
});

test('failed confinement never spawns',async()=>{
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,'BUILDER_CONFINEMENT_UNAVAILABLE');
    assert.equal(headlessCalls(calls).length,0);
  },{ops:operations({proveConfinement:async()=>({verified:false,
    reason:'BUILDER_CONFINEMENT_UNAVAILABLE'})})});
});

test('write-set violation and nonzero exit are never candidate PASS',async()=>{
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');assert.equal(x.reason,'WRITE_SET_VIOLATION');
    assert.equal(x.candidateTree,null);
  },{ops:operations({finalizeCandidate:async()=>({writeSetValid:false,paths:['forbidden.txt']})})});
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
    assert.equal(x.process.exitCode,7);
  },{behavior:close=>close(7,null)});
});

test('stderr tail on failure is secret-redacted in the durable record',async()=>{
  await withHost(async({host,config,registry})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    const record=await readAttemptRecord(registry,config.hostStorePath,config.attemptId);
    assert.ok(record.stderrTail.includes('MISSING_CREDENTIAL'));
    assert.equal(record.stderrTail.includes(API_KEY),false);
    assert.ok(record.stderrTail.includes('[REDACTED]'));
  },{behavior:(close,child)=>{child.stderr.emit('data',
    Buffer.from(`dsh: MISSING_CREDENTIAL leaked=${API_KEY}\n`));close(1,null);}});
});

test('timeout terminates the actual process tree and records durable UNKNOWN',async()=>{
  await withHost(async({host,kills,config})=>{
    const x=await host.runAttempt({...config,timeoutMs:25});
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
    const again=await host.runAttempt({...config,timeoutMs:25});
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
  },{behavior:()=>{}});
});

test('cancellation is complete only after process-tree termination',async()=>{
  await withHost(async({host,kills,config})=>{
    const controller=new AbortController();setTimeout(()=>controller.abort(),10);
    const x=await host.runAttempt({...config,timeoutMs:1000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('a signal already aborted before the run cancels immediately, never times out',async()=>{
  await withHost(async({host,kills,config})=>{
    const controller=new AbortController();controller.abort();
    const x=await host.runAttempt({...config,timeoutMs:60000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');
    assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));
  },{behavior:()=>{}});
});

test('spawn/lost ACK becomes UNKNOWN and cannot blindly relaunch',async()=>{
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    const again=await host.runAttempt(config);
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(headlessCalls(calls).length,1);
  },{behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
});

test('a pre-existing RUNNING record blocks restart relaunch',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-restart-'));
  const registry=createMemoryCasStoreRegistry();
  const config=configFor(dir,'attempt:dsh:restart:1');
  const store=await registry.open({path:config.hostStorePath});
  await store.put(attemptKey(config.attemptId),{phase:'RUNNING',attemptId:config.attemptId,
    process:{status:'RUNNING',pid:4242,builderPrincipal:config.builderPrincipal,builderSid}},
    {mustBeAbsent:true});
  await store.close();
  await withHost(async({host,calls})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.process.status,'RUNNING');
    assert.equal(headlessCalls(calls).length,0);
  },{registry});
});

test('output-pipe non-convergence is wired: grace destroys pipes and fails closed',async()=>{
  await withHost(async({host,config,registry,lastChildRef})=>{
    const x=await host.runAttempt({...config,pipeGraceMs:40});
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'DSH_OUTPUT_PIPE_NOT_CONVERGED');
    assert.equal(x.process.orphanPipe,true);
    assert.equal(lastChildRef().stdout.__destroyed,true,
      'the declared pipe-grace mechanism must actually be wired');
    const record=await readAttemptRecord(registry,config.hostStorePath,config.attemptId);
    assert.equal(record.process.orphanPipe,true);
  },{behavior:(_close,child)=>{child.emit('exit',0,null);}});
});

test('exit 0 with an empty final answer fails safe',async()=>{
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'DSH_EMPTY_FINAL_ANSWER');
    assert.equal(x.process.exitCode,0);
  },{behavior:close=>close(0,null)});
});

test('danger-full-access needs an explicit allowance; invalid modes are refused',async()=>{
  for(const overrides of [{permissionMode:'danger-full-access'},
    {permissionMode:'total-control'}])await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt({...config,...overrides});
    assert.equal(x.state,'AGENT_NOT_EXECUTED');assert.equal(x.reason,'DSH_CONFIG_INVALID');
    assert.equal(headlessCalls(calls).length,0);
  });
  await withHost(async({host,config})=>{
    const x=await host.runAttempt({...config,permissionMode:'danger-full-access',
      allowDangerousFullAccess:true});
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(x.process.exitCode,0);
  });
});

test('factory requires an injected CAS store factory',async()=>{
  assert.throws(()=>createLocalDshHost({}),e=>e.code==='DSH_HOST_STORE_FACTORY_REQUIRED');
});

test('candidate ref namespace is provider-unique (no codex/zcode collision)',()=>{
  assert.equal(LOCAL_DSH_CONSTANTS.candidateRefPrefix,'refs/tally-dsh-candidates/');
  assert.notEqual(LOCAL_DSH_CONSTANTS.candidateRefPrefix,'refs/tally-candidates/');
  assert.notEqual(LOCAL_DSH_CONSTANTS.candidateRefPrefix,'refs/tally-zcode-candidates/');
});

test('atomic shared-worktree grants deny the second competing attempt',async()=>{
  const registry=createMemoryCasStoreRegistry();
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-grant-'));
  const spawnCount={n:0};
  const spawnFn=(command,args,options)=>{
    spawnCount.n++;return fakeChild(answerBehavior);
  };
  const mkHost=()=>createLocalDshHost({spawnFn,operations:operations(),
    now:()=>Date.now(),killTree:async()=>true,openStore:registry.open});
  const hostA=mkHost(),hostB=mkHost();
  const configA=configFor(dir,'attempt:dsh:race:a');
  const configB=configFor(dir,'attempt:dsh:race:b');
  const [x,y]=await Promise.all([hostA.runAttempt(configA),hostB.runAttempt(configB)]);
  assert.equal([x,y].filter(v=>v.state==='LOCAL_CANDIDATE_READY').length,1);
  assert.equal([x,y].filter(v=>v.state==='NOT_ADMITTED'&&v.reason==='CANDIDATE_WORKTREE_OVERLAP').length,1);
  await rm(dir,{recursive:true,force:true});
});

test('grant derivation excludes cross-provider (Codex) resource holders',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-xprovider-'));
  const registry=createMemoryCasStoreRegistry();
  const config=configFor(dir,'attempt:dsh:xp:1');
  // A Codex-host attempt holds the same physical worktree path right now.
  const codexGrant='tally:v2:local-agent-grant:worktree:'+
    createHash('sha256').update(config.worktreePath).digest('hex');
  const store=await registry.open({path:config.hostStorePath});
  await store.put(codexGrant,{phase:'ACTIVE',kind:'worktree',value:config.worktreePath,
    attemptId:'attempt:t06m:other:1'},{mustBeAbsent:true});
  await store.close();
  await withHost(async({host,calls})=>{
    const x=await host.runAttempt(config);
    assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.reason,'CANDIDATE_WORKTREE_OVERLAP');
    assert.equal(x.agentLaunch,false);
    assert.equal(headlessCalls(calls).length,0);
  },{registry});
});

test('base builder environment is a credential-free allowlist',()=>{
  const env=LOCAL_DSH_CONSTANTS.builderEnvironment({gitPath:'C:\\Program Files\\Git\\cmd\\git.exe',
    builderProfile:'C:\\Users\\CodexSandboxOffline',emptyGitConfig:'C:\\B\\empty-gitconfig'});
  for(const name of ['GITHUB_TOKEN','GH_TOKEN','WOODPECKER_TOKEN','SSH_AUTH_SOCK','SSH_AGENT_PID',
    'DEEPSEEK_API_KEY','DSH_HOME','DSH_TELEMETRY_MODE','DSH_PERMISSION_MODE'])
    assert.equal(Object.hasOwn(env,name),false,name);
  assert.equal(env.GIT_CONFIG_NOSYSTEM,'1');assert.equal(env.GIT_TERMINAL_PROMPT,'0');
});

/* ---- real-spawn contract tests against the fake CLI + real git ---- */

const GIT=process.platform==='win32'?'C:\\Program Files\\Git\\cmd\\git.exe':'git';
function gitSync(repo,args){
  return execFileSync(GIT,['-c','user.name=Tally Test','-c','user.email=tally@example.local',
    ...args],{cwd:repo,encoding:'utf8'});
}
async function makeRepo(dir){
  await mkdir(dir,{recursive:true});
  gitSync(dir,['init','-q']);
  await writeFile(join(dir,'BASE.md'),'# base\n','utf8');
  gitSync(dir,['add','BASE.md']);
  gitSync(dir,['commit','-q','-m','base']);
  return gitSync(dir,['rev-parse','HEAD']).trim();
}
function realConfig(dir,attemptId,fakeArgs,overrides={}){
  return configFor(dir,attemptId,{binPath:process.execPath,
    binArgsPrefix:[FAKE_DSH,...fakeArgs],...overrides});
}
async function withRealHost(fn,config){
  const spawnFn=(...a)=>spawn(...a);
  const defaultOps=createDefaultDshOperations(spawnFn);
  const host=createLocalDshHost({spawnFn,
    operations:{...defaultOps,
      proveConfinement:async c=>({verified:true,builderPrincipal:c.builderPrincipal,
        builderSid:c.builderSid,hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'})},
    now:()=>Date.now(),openStore:createMemoryCasStoreRegistry().open});
  return await fn({host,ops:defaultOps,config});
}

test('real probe: pinned version accepted, drift and missing CLI refused',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-probe-'));
  try{
    const ops=createDefaultDshOperations((...a)=>spawn(...a));
    const ok=await ops.probeProvider(realConfig(dir,'attempt:dsh:probe:1',
      ['--fake-version','0.2.2-rc.5'],{deepseekApiKey:API_KEY}));
    assert.equal(ok.available,true);assert.equal(ok.providerVersion,'0.2.2-rc.5');
    assert.equal(ok.provider,'dsh');assert.equal(ok.binSource,'config');
    assert.equal(ok.canaryVerified,true,'CANARY tier ran the pinned round-trip');
    const drifted=await ops.probeProvider(realConfig(dir,'attempt:dsh:probe:2',
      ['--fake-version','9.9.9-x'],{deepseekApiKey:API_KEY}));
    assert.equal(drifted.available,false);assert.equal(drifted.stage,'VERSION');
    assert.equal(drifted.observedVersion,'9.9.9-x');
    assert.equal(drifted.pinnedVersion,'0.2.2-rc.5');
    const nokey=await ops.probeProvider(realConfig(dir,'attempt:dsh:probe:3',
      ['--fake-version','0.2.2-rc.5'],{deepseekApiKey:null}));
    assert.equal(nokey.available,false);assert.equal(nokey.stage,'AUTH');
    assert.equal(nokey.credentialPresent,false);
    const missing=await ops.probeProvider(realConfig(dir,'attempt:dsh:probe:4',[],
      {binPath:'C:\\Nonexistent\\dsh.exe',deepseekApiKey:API_KEY}));
    assert.equal(missing.available,false);assert.equal(missing.stage,'VERSION');
    const offPath=await ops.probeProvider(realConfig(dir,'attempt:dsh:probe:5',[],
      {binPath:'dsh-surely-not-on-path',deepseekApiKey:API_KEY}));
    assert.equal(offPath.available,false);assert.equal(offPath.stage,'VERSION');
    assert.equal(offPath.spawnError,true);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('probe tiers run in order: VERSION, CREDENTIALS(AUTH), then CANARY',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-canary-'));
  try{
    const ops=createDefaultDshOperations((...a)=>spawn(...a));
    const counter=join(dir,'version-count');
    // Both executable tiers each run one pinned round-trip (fake invocation 1
    // = VERSION, invocation 2 = CANARY) and the credential tier sits between.
    const ok=await ops.probeProvider(realConfig(dir,'attempt:dsh:canary:0',
      ['--fake-version','0.2.2-rc.5','--fake-counter',counter],{deepseekApiKey:API_KEY}));
    assert.equal(ok.available,true);assert.equal(ok.canaryVerified,true);
    assert.equal((await readFile(counter,'utf8')).trim(),'2');
    // Canary tier failing (second round-trip drifts) refuses without spawn.
    const counter2=join(dir,'version-count-2');
    const canaryFail=await ops.probeProvider(realConfig(dir,'attempt:dsh:canary:1',
      ['--fake-version','0.2.2-rc.5','--fake-counter',counter2,
        '--fake-version-fail-from','2'],{deepseekApiKey:API_KEY}));
    assert.equal(canaryFail.available,false);
    assert.equal(canaryFail.stage,'CANARY');
    assert.equal(canaryFail.observedVersion,'0.0.0-canary-drift');
    assert.equal(canaryFail.exitCode,3);
    assert.equal(canaryFail.canaryMatched,false);
    // Credential tier failing stops the gate after the VERSION round-trip:
    // the counter records exactly one invocation, CANARY never runs.
    const neverCounter=join(dir,'never');
    const nokey=await ops.probeProvider(realConfig(dir,'attempt:dsh:canary:2',
      ['--fake-version','0.2.2-rc.5','--fake-counter',neverCounter],{deepseekApiKey:null}));
    assert.equal(nokey.available,false);assert.equal(nokey.stage,'AUTH');
    assert.equal((await readFile(neverCounter,'utf8')).trim(),'1');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real headless run end-to-end: candidate tree, ref, readback and env evidence',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-e2e-'));
  try{
    const repo=join(dir,'repo');const base=await makeRepo(repo);
    const recordPath=join(dir,'fake-record.jsonl');
    const config=realConfig(dir,'attempt:dsh:e2e:1',
      ['--fake-scenario','answer','--fake-write','["NOTES.md"]','--fake-record',recordPath],{
        repositoryPath:repo,sourceSha:base,worktreePath:join(dir,'wt'),
        candidateRelativePath:'NOTES.md',pipeGraceMs:4000,timeoutMs:30000});
    await withRealHost(async({host,ops})=>{
      const x=await host.runAttempt(config);
      assert.equal(x.state,'LOCAL_CANDIDATE_READY');
      assert.equal(x.reason,'REAL_DSH_CANDIDATE_RECORDED');
      assert.ok(/^[0-9a-f]{40}$/.test(x.candidateCommit??''));
      assert.ok(x.candidateRef?.startsWith('refs/tally-dsh-candidates/'));
      assert.deepEqual(x.candidatePaths,['NOTES.md']);
      assert.ok(x.finalAnswerPreview.includes('FAKE_DSH_FINAL_ANSWER'));
      // Host-owned candidate reads back independently from the repository.
      const readback=await ops.verifyCandidateReadback(config,
        {candidateCommit:x.candidateCommit,candidateTree:x.candidateTree});
      assert.equal(readback.verified,true);
      assert.equal(readback.parent,base);
      assert.deepEqual(readback.paths,['NOTES.md']);
      // The fake CLI observed the real spawn contract.
      const line=JSON.parse((await readFile(recordPath,'utf8')).trim().split(/\r?\n/).at(-1));
      assert.equal(line.cwd,join(dir,'wt'));
      assert.equal(line.env.DSH_TELEMETRY_MODE,'DISABLED');
      assert.equal(line.env.DSH_PERMISSION_MODE,'workspace-write');
      assert.equal(line.env.DEEPSEEK_API_KEY_present,true);
      assert.ok(line.env.DSH_HOME&&line.env.DSH_HOME.length>0);
      assert.ok(line.argv.at(-1).includes('NOTES.md'));
    },config);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real run write-set violation is detected end-to-end',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-wsv-'));
  try{
    const repo=join(dir,'repo');const base=await makeRepo(repo);
    const config=realConfig(dir,'attempt:dsh:wsv:1',
      ['--fake-scenario','answer','--fake-write','["NOTES.md","EXTRA.txt"]'],{
        repositoryPath:repo,sourceSha:base,worktreePath:join(dir,'wt'),
        candidateRelativePath:'NOTES.md',pipeGraceMs:4000,timeoutMs:30000});
    await withRealHost(async({host})=>{
      const x=await host.runAttempt(config);
      assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
      assert.equal(x.reason,'WRITE_SET_VIOLATION');
    },config);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real MISSING_CREDENTIAL stderr is classified as process failure with evidence',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-cred-'));
  try{
    const repo=join(dir,'repo');const base=await makeRepo(repo);
    const config=realConfig(dir,'attempt:dsh:cred:1',
      ['--fake-scenario','missing-credential'],{
        repositoryPath:repo,sourceSha:base,worktreePath:join(dir,'wt'),
        pipeGraceMs:4000,timeoutMs:30000});
    await withRealHost(async({host})=>{
      const x=await host.runAttempt(config);
      assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
      assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');
      assert.equal(x.process.exitCode,1);
    },config);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real orphan pipe: grace converges the wait, fails closed, evidence recorded',async t=>{
  // Ground truth (empirically probed 2026-11-19): on Windows an inherited
  // grandchild handle does NOT keep the child's stdout open — `close` fires
  // with the direct child's exit, so a real orphan-hold is only observable
  // on POSIX. The grace mechanism's WIRING is asserted by the injected
  // fake-child test above regardless of platform.
  if(process.platform==='win32'){
    t.skip('Windows closes child pipes on direct-child exit regardless of '
      +'inherited grandchild handles; real orphan-hold observable only on POSIX');
    return;
  }
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-orphan-'));
  try{
    const repo=join(dir,'repo');const base=await makeRepo(repo);
    const pidFile=join(dir,'grandchild.pid');
    const config=realConfig(dir,'attempt:dsh:orph:1',
      ['--fake-scenario','orphan','--fake-grandchild-pid-file',pidFile],{
        repositoryPath:repo,sourceSha:base,worktreePath:join(dir,'wt'),
        pipeGraceMs:800,timeoutMs:30000});
    await withRealHost(async({host})=>{
      const x=await host.runAttempt(config);
      assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
      assert.equal(x.reason,'DSH_OUTPUT_PIPE_NOT_CONVERGED');
      assert.equal(x.process.orphanPipe,true);
      const grandchildPid=Number((await readFile(pidFile,'utf8')).trim());
      await new Promise(resolve=>setTimeout(resolve,200));
      assert.throws(()=>process.kill(grandchildPid,0),undefined,
        'posix process-group termination must reap the orphan grandchild');
    },config);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('real hang: timeout kills and records durable UNKNOWN',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-dsh-hang-'));
  try{
    const repo=join(dir,'repo');const base=await makeRepo(repo);
    const config=realConfig(dir,'attempt:dsh:hang:1',['--fake-scenario','hang'],{
        repositoryPath:repo,sourceSha:base,worktreePath:join(dir,'wt'),
        pipeGraceMs:60000,timeoutMs:700});
    await withRealHost(async({host})=>{
      const x=await host.runAttempt(config);
      assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
      assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
      const again=await host.runAttempt(config);
      assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    },config);
  }finally{await rm(dir,{recursive:true,force:true});}
});
