import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createLocalCodexHost,LOCAL_CODEX_CONSTANTS} from './local-codex-host.mjs';

const sourceSha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9';
const issueRef='https://github.com/kaicreator-mm/tally/issues/172';
const builderSid='S-1-5-21-2839165489-1832998833-1574237796-1003';
const candidate={writeSetValid:true,paths:[LOCAL_CODEX_CONSTANTS.candidateRelativePath],
  candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),candidateRef:'refs/tally-candidates/test'};
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
    probeProvider:async()=>({available:true}),
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha}),
    proveConfinement:async config=>({verified:true,builderPrincipal:config.builderPrincipal,
      builderSid:config.builderSid,hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'}),
    finalizeCandidate:async()=>candidate,...overrides
  };
}
function configFor(dir,attemptId='attempt:t06m:test:1'){
  return {issueRef,sourceSha,attemptId,hostStorePath:join(dir,'host.sqlite'),
    repositoryPath:dir,worktreePath:join(dir,'candidate'),cliPath:'C:\\Public\\codex.exe',
    sandboxCliPath:'C:\\Public\\codex.exe',codexHome:join(dir,'codex-home'),
    builderProfile:'C:\\Users\\CodexSandboxOffline',builderTemp:join(dir,'tmp'),
    emptyGitConfig:join(dir,'empty-gitconfig'),operatorHome:'C:\\Users\\operator',
    hiddenFixture:join(dir,'hidden'),builderPrincipal:'nilar\\codexsandboxoffline',builderSid,
    verifierSid:'S-1-5-21-2839165489-1832998833-1574237796-1004',timeoutMs:1000};
}
async function withHost(fn,{ops=operations(),behavior=(close)=>close(0,null),killTree=async()=>true}={}){
  const dir=await mkdtemp(join(tmpdir(),'tally-local-host-test-'));
  const calls=[],kills=[];let lastChild=null;
  const spawnFn=(command,args,options)=>{
    calls.push({command,args,options});lastChild=fakeChild(behavior);return lastChild;
  };
  const host=createLocalCodexHost({spawnFn,operations:ops,now:()=>Date.UTC(2026,9,30),
    killTree:async pid=>{kills.push(pid);lastChild?.kill();return killTree(pid);}});
  try{return await fn({host,dir,calls,kills,config:configFor(dir)});}finally{await rm(dir,{recursive:true,force:true});}
}
const attemptKey=id=>LOCAL_CODEX_CONSTANTS.attemptKeyPrefix+id;

test('real process exit produces exact candidate and durable restart recovery without relaunch',async()=>{
  await withHost(async({host,dir,calls,config})=>{
    const first=await host.runAttempt(config);
    assert.equal(first.state,'LOCAL_CANDIDATE_READY');assert.equal(first.agentLaunch,true);
    assert.equal(first.candidateTree,candidate.candidateTree);assert.equal(first.process.exitCode,0);
    assert.equal(calls.length,1);
    const second=await host.runAttempt(config);
    assert.equal(second.state,'LOCAL_CANDIDATE_READY');assert.equal(second.candidateCommit,candidate.candidateCommit);
    assert.equal(calls.length,1,'restart must not relaunch a completed non-idempotent attempt');
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    try{
      const row=await store.get(attemptKey(config.attemptId));
      assert.equal(row.value.phase,'CANDIDATE_READY');
      assert.equal(row.value.candidateRef,candidate.candidateRef);
      assert.ok(row.value.argv.every(x=>!x.includes('WOODPECKER_TOKEN')));
      assert.ok(!row.value.envAllowlist.some(x=>/TOKEN|SSH_AUTH|SSH_AGENT/.test(x)));
      assert.equal(JSON.stringify(row).includes('DISPATCH_CLAIMED'),false);
    }finally{await store.close();}
  });
});

test('missing/unauthed provider and failed confinement never spawn',async()=>{
  for(const [ops,reason] of [
    [operations({probeProvider:async()=>({available:false})}),'LOCAL_AGENT_UNAVAILABLE'],
    [operations({proveConfinement:async()=>({verified:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'})}),
      'BUILDER_CONFINEMENT_UNAVAILABLE']
  ])await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);assert.equal(x.state,'AGENT_NOT_EXECUTED');
    assert.equal(x.reason,reason);assert.equal(x.agentLaunch,false);assert.equal(calls.length,0);
  },{ops});
});

test('wrong write set and nonzero process exit are not candidate PASS',async()=>{
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'WRITE_SET_VIOLATION');assert.equal(x.candidateTree,null);
  },{ops:operations({finalizeCandidate:async()=>({writeSetValid:false,paths:['forbidden.txt']})})});
  await withHost(async({host,config})=>{
    const x=await host.runAttempt(config);assert.equal(x.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(x.reason,'ATTEMPT_PROCESS_FAILED');assert.equal(x.process.exitCode,7);
  },{behavior:close=>close(7,null)});
});

test('timeout terminates the actual process tree and records durable UNKNOWN',async()=>{
  await withHost(async({host,kills,config})=>{
    const x=await host.runAttempt({...config,timeoutMs:25});
    assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');assert.equal(x.reason,'ATTEMPT_TIMEOUT_RECONCILED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
    const again=await host.runAttempt({...config,timeoutMs:25});
    assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
  },{behavior:()=>{}});
});

test('cancellation is complete only after process-tree termination and reconciliation',async()=>{
  await withHost(async({host,kills,config})=>{
    const controller=new AbortController();setTimeout(()=>controller.abort(),10);
    const x=await host.runAttempt({...config,timeoutMs:1000,signal:controller.signal});
    assert.equal(x.state,'LOCAL_ATTEMPT_CANCELLED');assert.equal(x.reason,'ATTEMPT_CANCELLED');
    assert.ok(kills.includes(4242));assert.equal(x.taskPass,false);
  },{behavior:()=>{}});
});

test('spawn/lost ACK becomes UNKNOWN and cannot blindly relaunch',async()=>{
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt(config);assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(x.reason,'ATTEMPT_STATUS_UNKNOWN');
    const again=await host.runAttempt(config);assert.equal(again.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(calls.length,1);
  },{behavior:(_close,child)=>child.emit('error',new Error('PRIVATE_ACL_DETAIL'))});
});

test('a pre-existing RUNNING record blocks restart relaunch',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-local-restart-'));
  try{
    const config=configFor(dir,'attempt:t06m:restart:1');
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    await store.put(attemptKey(config.attemptId),{phase:'RUNNING',attemptId:config.attemptId,
      process:{status:'RUNNING',pid:4242,builderPrincipal:config.builderPrincipal,builderSid}},
      {mustBeAbsent:true});await store.close();
    await withHost(async({host,calls})=>{
      const x=await host.runAttempt(config);assert.equal(x.state,'LOCAL_ATTEMPT_UNKNOWN');
      assert.equal(x.process.status,'RUNNING');assert.equal(calls.length,0);
    });
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('Builder environment is an explicit allowlist with no operator credential inheritance',()=>{
  const env=LOCAL_CODEX_CONSTANTS.builderEnvironment({cliPath:'C:\\Public\\codex.exe',
    codexHome:'C:\\Builder\\home',builderProfile:'C:\\Users\\CodexSandboxOffline',
    emptyGitConfig:'C:\\Builder\\empty-gitconfig'});
  for(const name of ['GITHUB_TOKEN','GH_TOKEN','WOODPECKER_TOKEN','SSH_AUTH_SOCK','SSH_AGENT_PID',
    'DEEPSEEK_API_KEY'])assert.equal(Object.hasOwn(env,name),false);
  assert.equal(env.GIT_CONFIG_NOSYSTEM,'1');assert.equal(env.GIT_TERMINAL_PROMPT,'0');
  assert.equal(env.CODEX_HOME,'C:\\Builder\\home');
});
