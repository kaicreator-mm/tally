import test from 'node:test';
import assert from 'node:assert/strict';
import {EventEmitter} from 'node:events';
import {fileURLToPath} from 'node:url';
import {mkdtemp,rm} from 'node:fs/promises';
import {existsSync} from 'node:fs';
import {stat} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createLocalCodexHost,createDefaultOperations,LOCAL_CODEX_CONSTANTS} from './local-codex-host.mjs';

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

test('Linux Host disallows alternate provider without the certified bwrap process',async()=>{
  await withHost(async({host,calls,config})=>{
    const provider={id:'zcode',probe:async()=>({state:'AVAILABLE'}),
      execute:async()=>({state:'COMPLETED'}),cancel:async()=>{},
      reconcile:async()=>null};
    const result=await host.runAttempt({...config,platform:'linux',
      providerId:'zcode',agentProvider:provider,taskId:'172'});
    assert.equal(result.state,'AGENT_NOT_EXECUTED');
    assert.equal(result.reason,'PROVIDER_DISABLED');
    assert.equal(result.agentLaunch,false);
    assert.equal(calls.length,0);
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


const CONCURRENT_A='experiments/v2/selected-business-build/CONCURRENT_A.md';
const CONCURRENT_B='experiments/v2/selected-business-build/CONCURRENT_B.md';
const joinKey=id=>LOCAL_CODEX_CONSTANTS.joinKeyPrefix+id;
function joinOperations(overrides={}){
  return {
    verifyCandidateReadback:async(config,record)=>({verified:true,
      candidateTree:record.candidateTree,parent:config.sourceSha,
      paths:[record.candidateRelativePath??LOCAL_CODEX_CONSTANTS.candidateRelativePath]}),
    ...overrides
  };
}
function readyRecord(attemptId,overrides={}){
  return {phase:'CANDIDATE_READY',attemptId,sourceSha,issueRef,
    claimGeneration:1,worktreePath:'C:\\worktrees\\'+attemptId,
    candidateRelativePath:CONCURRENT_A,startedAt:'2026-10-30T00:00:00Z',
    candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40),
    candidateRef:'refs/tally-candidates/'+attemptId.replace(/[^A-Za-z0-9._-]/g,'-'),
    process:{status:'EXITED',exitCode:0,pid:4242,startedAt:'2026-10-30T00:00:00Z',
      terminatedAt:'2026-10-30T00:01:00Z'},...overrides};
}
function joinHostOver(readback){
  return {ops:{
    probeProvider:async()=>({available:true}),
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha}),
    proveConfinement:async config=>({verified:true,builderPrincipal:config.builderPrincipal,
      builderSid:config.builderSid,hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'}),
    finalizeCandidate:async()=>candidate,
    verifyCandidateReadback:readback??joinOperations().verifyCandidateReadback,
  }};
}
const joinConfigFor=(dir,joinId,candidates)=>({issueRef,sourceSha,joinId,
  hostStorePath:join(dir,'host.sqlite'),repositoryPath:dir,candidates});
const joinCandidates=()=>[
  {attemptId:'attempt:t06n:a:1',candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40)},
  {attemptId:'attempt:t06n:b:1',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}];

test('candidateRelativePath config drives record, prompt and write-set default',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06n-path-'));
  try{
    const calls=[];let seenConfig=null;
    const spawnFn=(command,args,options)=>{
      calls.push({command,args,options});return fakeChild((close)=>close(0,null));
    };
    const host=createLocalCodexHost({spawnFn,
      operations:operations({finalizeCandidate:async c=>{seenConfig=c;return candidate;}}),
      now:()=>Date.UTC(2026,9,30),killTree:async()=>true});
    const config=configFor(dir);
    const x=await host.runAttempt({...config,candidateRelativePath:CONCURRENT_A});
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(seenConfig.candidateRelativePath,CONCURRENT_A);
    const prompt=calls[0].args.at(-1);
    assert.ok(prompt.includes(CONCURRENT_A));
    assert.ok(!prompt.includes(LOCAL_CODEX_CONSTANTS.candidateRelativePath));
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    const row=await store.get(attemptKey(config.attemptId));
    assert.equal(row.value.candidateRelativePath,CONCURRENT_A);
    await store.close();
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('concurrent Builders get distinct records and real overlapping lifetimes',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06n-overlap-'));
  try{
    const mkHost=delayMs=>{
      const calls=[];
      const spawnFn=(command,args,options)=>{
        calls.push({command,args,options});
        return fakeChild((close)=>setTimeout(()=>close(0,null),delayMs));
      };
      return {host:createLocalCodexHost({spawnFn,operations:operations(),
        now:()=>Date.now(),killTree:async()=>true}),calls};
    };
    const a=mkHost(120),b=mkHost(120);
    const configA={...configFor(dir,'attempt:t06n:overlap:a'),
      worktreePath:join(dir,'wt-a'),candidateRelativePath:CONCURRENT_A};
    const configB={...configFor(dir,'attempt:t06n:overlap:b'),
      worktreePath:join(dir,'wt-b'),candidateRelativePath:CONCURRENT_B};
    const [ra,rb]=await Promise.all([a.host.runAttempt(configA),b.host.runAttempt(configB)]);
    assert.equal(ra.state,'LOCAL_CANDIDATE_READY');assert.equal(rb.state,'LOCAL_CANDIDATE_READY');
    assert.equal(a.calls.length,1);assert.equal(b.calls.length,1);
    const a0=Date.parse(ra.process.startedAt),a1=Date.parse(ra.process.terminatedAt);
    const b0=Date.parse(rb.process.startedAt),b1=Date.parse(rb.process.terminatedAt);
    assert.ok(a0<b1&&b0<a1,'two real process lifetimes must overlap for overlap evidence');
    const store=await openSqliteCasStore({path:configA.hostStorePath,busyTimeoutMs:1000});
    const keys=await store.list(LOCAL_CODEX_CONSTANTS.attemptKeyPrefix);
    assert.equal(keys.length,2,'independent CAS records per concurrent attempt');
    await store.close();
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('shared worktree or overlapping write grant denies the second Builder',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06n-deny-'));
  try{
    const store=await openSqliteCasStore({path:join(dir,'host.sqlite'),busyTimeoutMs:1000});
    await store.put(attemptKey('attempt:t06n:active:1'),
      readyRecord('attempt:t06n:active:1',{phase:'RUNNING'}),{mustBeAbsent:true});
    await store.close();
    const host=createLocalCodexHost({
      spawnFn:()=>{throw Error('overlap denial must not spawn');},
      operations:operations(),now:()=>Date.UTC(2026,9,30),killTree:async()=>true});
    const x=await host.runAttempt({...configFor(dir,'attempt:t06n:deny:wt'),
      worktreePath:'C:\\worktrees\\attempt:t06n:active:1',candidateRelativePath:CONCURRENT_B});
    assert.equal(x.state,'NOT_ADMITTED');assert.equal(x.reason,'CANDIDATE_WORKTREE_OVERLAP');
    assert.equal(x.agentLaunch,false);assert.equal(x.admitted,false);
    const y=await host.runAttempt({...configFor(dir,'attempt:t06n:deny:grant'),
      worktreePath:join(dir,'other-wt'),candidateRelativePath:CONCURRENT_A});
    assert.equal(y.state,'NOT_ADMITTED');assert.equal(y.reason,'CANDIDATE_WRITE_GRANT_OVERLAP');
    assert.equal(y.agentLaunch,false);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('atomic grants allow exactly one competing Builder with no second spawn',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06n-grant-race-'));
  try{
    const calls=[];
    const spawnFn=(command,args,options)=>{calls.push({command,args,options});
      return fakeChild(close=>setTimeout(()=>close(0,null),20));};
    const host=createLocalCodexHost({spawnFn,operations:operations(),now:()=>Date.now(),killTree:async()=>true});
    const shared={worktreePath:join(dir,'shared-wt'),candidateRelativePath:CONCURRENT_A};
    const [x,y]=await Promise.all([
      host.runAttempt({...configFor(dir,'attempt:t06n:race:a'),...shared}),
      host.runAttempt({...configFor(dir,'attempt:t06n:race:b'),...shared})
    ]);
    assert.equal([x,y].filter(v=>v.state==='LOCAL_CANDIDATE_READY').length,1);
    assert.equal([x,y].filter(v=>v.state==='NOT_ADMITTED'&&v.reason==='CANDIDATE_WORKTREE_OVERLAP').length,1);
    assert.equal(calls.length,1,'atomic grant must prevent the losing process from spawning');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('reviewer Join verifies exact candidates and replays without re-readback',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06n-join-'));
  try{
    const store=await openSqliteCasStore({path:join(dir,'host.sqlite'),busyTimeoutMs:1000});
    await store.put(attemptKey('attempt:t06n:a:1'),readyRecord('attempt:t06n:a:1'),
      {mustBeAbsent:true});
    await store.put(attemptKey('attempt:t06n:b:1'),readyRecord('attempt:t06n:b:1',
      {candidateRelativePath:CONCURRENT_B,candidateTree:'c'.repeat(40),
        candidateCommit:'d'.repeat(40),worktreePath:'C:\\worktrees\\b'}),{mustBeAbsent:true});
    await store.close();
    const host=createLocalCodexHost({
      spawnFn:()=>{throw Error('join must not spawn processes');},
      operations:joinOperations(),now:()=>Date.UTC(2026,9,30),killTree:async()=>true});
    const joined=await host.joinReviewedCandidates(
      joinConfigFor(dir,'join:t06n:test:1',joinCandidates()));
    assert.equal(joined.state,'REVIEW_JOINED');
    assert.equal(joined.candidates.length,2);
    assert.equal(joined.taskPass,false);assert.equal(joined.authorized,false);
    assert.equal(joined.candidates[0].candidateRelativePath,CONCURRENT_A);
    assert.equal(joined.candidates[1].candidateRelativePath,CONCURRENT_B);
    assert.ok(joined.candidates[0].terminatedAt,'overlap timestamps preserved in join');
    const store2=await openSqliteCasStore({path:join(dir,'host.sqlite'),busyTimeoutMs:1000});
    const row=await store2.get(joinKey('join:t06n:test:1'));
    assert.equal(row.value.phase,'JOINED');
    await store2.close();
    const again=await host.joinReviewedCandidates(
      joinConfigFor(dir,'join:t06n:test:1',joinCandidates()));
    assert.equal(again.state,'REVIEW_JOINED');
    assert.equal(again.joinedAt,joined.joinedAt,'idempotent join never re-reads');
    const stale=await host.joinReviewedCandidates({...joinConfigFor(dir,'join:t06n:test:1',joinCandidates()),
      sourceSha:'f'.repeat(40)});
    assert.equal(stale.state,'NOT_ADMITTED');assert.equal(stale.reason,'CANDIDATE_STALE');
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('reviewer Join rejects missing, not-ready, stale, overlapping and unreadbackable candidates',async()=>{
  const cases=[
    ['CANDIDATE_NOT_FOUND',{},{},
      {attemptId:'attempt:t06n:missing:1',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}],
    ['CANDIDATE_NOT_READY',{},{phase:'FAILED'},
      {attemptId:'attempt:t06n:b:1',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}],
    ['CANDIDATE_STALE',{sourceSha:'f'.repeat(40)},{},
      {attemptId:'attempt:t06n:b:1',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}],
    ['CANDIDATE_STALE',{},{},
      {attemptId:'attempt:t06n:b:1',candidateTree:'c'.repeat(40),candidateCommit:'e'.repeat(40)}],
    ['CANDIDATE_WRITESET_OVERLAP',{},{candidateRelativePath:CONCURRENT_A},
      {attemptId:'attempt:t06n:b:1',candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40)}],
    ['CANDIDATE_READBACK_MISMATCH',{},{},
      {attemptId:'attempt:t06n:b:1',candidateTree:'c'.repeat(40),candidateCommit:'d'.repeat(40)}],
  ];
  for(const [reason,aOverride,bRecordOverride,bCandidate] of cases){
    const dir=await mkdtemp(join(tmpdir(),'tally-t06n-join-deny-'));
    try{
      const store=await openSqliteCasStore({path:join(dir,'host.sqlite'),busyTimeoutMs:1000});
      await store.put(attemptKey('attempt:t06n:a:1'),readyRecord('attempt:t06n:a:1',aOverride),
        {mustBeAbsent:true});
      await store.put(attemptKey('attempt:t06n:b:1'),
        readyRecord('attempt:t06n:b:1',{candidateRelativePath:CONCURRENT_B,
          worktreePath:'C:\\worktrees\\b',...bRecordOverride}),{mustBeAbsent:true});
      await store.close();
      const readback=reason==='CANDIDATE_READBACK_MISMATCH'?
        async()=>({verified:false}):joinOperations().verifyCandidateReadback;
      const host=createLocalCodexHost({
        spawnFn:()=>{throw Error('join must not spawn processes');},
        operations:joinOperations({verifyCandidateReadback:readback}),
        now:()=>Date.UTC(2026,9,30),killTree:async()=>true});
      const x=await host.joinReviewedCandidates(joinConfigFor(dir,
        'join:t06n:deny:'+reason.toLowerCase(),[joinCandidates()[0],bCandidate]));
      assert.equal(x.state,'NOT_ADMITTED',reason);
      assert.equal(x.reason,reason);
      assert.equal(x.admitted,false);assert.equal(x.taskPass,false);
      assert.equal(x.agentLaunch,false);
    }finally{await rm(dir,{recursive:true,force:true});}
  }
});

test('bounded repair uses a new attempt id and preserves prior failure evidence',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06n-repair-'));
  try{
    const calls=[];
    const spawnFn=(command,args,options)=>{
      calls.push({command,args,options});
      return fakeChild((close)=>close(0,null));
    };
    const failingHost=createLocalCodexHost({spawnFn:()=>fakeChild((close)=>close(7,null)),
      operations:operations(),now:()=>Date.UTC(2026,9,30),killTree:async()=>true});
    const failed=await failingHost.runAttempt(
      {...configFor(dir,'attempt:t06n:repair:failed'),candidateRelativePath:CONCURRENT_B});
    assert.equal(failed.state,'LOCAL_ATTEMPT_FAILED');
    const host=createLocalCodexHost({spawnFn,operations:operations({finalizeCandidate:async()=>
      ({...candidate,candidateTree:'e'.repeat(40),candidateCommit:'f'.repeat(40),
        candidateRef:'refs/tally-candidates/repair'})}),
      now:()=>Date.UTC(2026,9,30),killTree:async()=>true});
    const repair=await host.runAttempt({...configFor(dir,'attempt:t06n:repair:2'),
      worktreePath:join(dir,'repair-wt'),candidateRelativePath:CONCURRENT_B});
    assert.equal(repair.state,'LOCAL_CANDIDATE_READY');
    assert.notEqual(repair.candidateCommit,failed.candidateCommit);
    assert.equal(calls.length,1);
    const store=await openSqliteCasStore({path:join(dir,'host.sqlite'),busyTimeoutMs:1000});
    const prior=await store.get(attemptKey('attempt:t06n:repair:failed'));
    assert.equal(prior.value.phase,'FAILED','prior failure evidence preserved verbatim');
    await store.close();
    const again=await host.runAttempt({...configFor(dir,'attempt:t06n:repair:2'),
      worktreePath:join(dir,'repair-wt'),candidateRelativePath:CONCURRENT_B});
    assert.equal(again.state,'LOCAL_CANDIDATE_READY');
    assert.equal(calls.length,1,'repaired attempt restart must not relaunch');
  }finally{await rm(dir,{recursive:true,force:true});}
});


// ---------------------------------------------------------------------------
// #264 J05 — sealed-prompt path-occupancy guard + single-candidate review
// ---------------------------------------------------------------------------
test('tracked candidate path refuses the Builder attempt before any spawn',async()=>{
  await withHost(async({host,calls,config})=>{
    const occupied=await host.runAttempt(config);
    assert.equal(occupied.state,'AGENT_NOT_EXECUTED');
    assert.equal(occupied.reason,'CANDIDATE_PATH_OCCUPIED');
    assert.equal(occupied.agentLaunch,false);
    assert.equal(calls.length,0,'occupancy refusal must precede the spawn');
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    const row=await store.get(attemptKey(config.attemptId));
    assert.equal(row.value.phase,'NOT_EXECUTED');
    assert.equal(row.value.reason,'CANDIDATE_PATH_OCCUPIED');
    await store.close();
  },{ops:operations({candidatePathOccupied:async()=>({occupied:true})})});
  // An explicitly authorized replacement is the only way past the guard
  // (a fresh attempt id: the refusal above left a terminal NOT_EXECUTED row).
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt({...config,attemptId:'attempt:t06m:replace:1',
      candidateReplacementAuthorized:true});
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(calls.length,1);
  },{ops:operations({candidatePathOccupied:async config=>
    ({occupied:config.candidateReplacementAuthorized!==true})})});
});

test('unproven occupancy (git error) refuses the Builder attempt without spawning (R103 P1-02)',async()=>{
  await withHost(async({host,calls,config})=>{
    const unknown=await host.runAttempt(config);
    assert.equal(unknown.state,'AGENT_NOT_EXECUTED');
    assert.equal(unknown.reason,'CANDIDATE_PATH_OCCUPANCY_UNKNOWN');
    assert.equal(unknown.agentLaunch,false);
    assert.equal(calls.length,0,'unproven occupancy must precede the spawn');
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    const row=await store.get(attemptKey(config.attemptId));
    assert.equal(row.value.phase,'NOT_EXECUTED');
    assert.equal(row.value.reason,'CANDIDATE_PATH_OCCUPANCY_UNKNOWN');
    await store.close();
  },{ops:operations({candidatePathOccupied:async()=>({occupied:null,
    detail:'fatal: Not a valid object name'})})});
});

const reviewCandidateCommit='023db3aa9a6c30ef5975c6f085ba207ec315cb09';
const reviewCandidateTree='ff7cd2b35ea65d6c7ab316b963eed4066af455db';
const reviewRepairCommit='d9fbc0239a8c38daae0c5637451f1e6b10fce983';
const reviewRepairTree='cb761f739b2fd2bc737f68ad19e4a0b41c63c023';
function reviewOperations(overrides={}){
  const path=LOCAL_CODEX_CONSTANTS.candidateRelativePath;
  return {
    probeProvider:async()=>({available:true}),
    prepareWorktree:async config=>({worktreePath:config.worktreePath,baseSha:config.sourceSha}),
    proveConfinement:async config=>({verified:true,builderPrincipal:config.builderPrincipal,
      builderSid:config.builderSid,hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'}),
    finalizeCandidate:async()=>candidate,
    verifyCandidateReadback:async()=>({verified:true,candidateTree:reviewCandidateTree,
      parent:sourceSha,paths:[path]}),
    verifyRepairReadback:async()=>({verified:true,repairTree:reviewRepairTree,
      parent:reviewCandidateCommit,paths:[path],candidatePath:path}),
    ...overrides
  };
}
// Mirror of the host's frozen REVIEW_REQUIRED_CI_CONTEXTS: the exact-SHA
// combined status must carry each of these terminal-success contexts before
// the single-candidate qualification gate may pass (R103 review P1-03).
const reviewRequiredContexts=[
  'ci/woodpecker/pr/v2-alm-workspace',
  'ci/woodpecker/pr/v2-candidate-publication',
  'ci/woodpecker/pr/v2-t06l-kernel-admission',
  'ci/woodpecker/pr/v2-t06m-local-agent',
  'ci/woodpecker/pr/v2-t06n-concurrent-agent'];
function greenCombinedStatus(){
  return {state:'success',totalCount:reviewRequiredContexts.length,
    contexts:reviewRequiredContexts.map(context=>({context,state:'success'}))};
}
function reviewConfigFor(dir,overrides={}){
  return {issueRef:'https://github.com/kaicreator-mm/tally/issues/264',
    sourceSha,decisionId:'decision:264:j5:test:1',decision:'ACCEPT',
    attemptId:'attempt:264:e1:r91:4',
    candidateCommit:reviewCandidateCommit,candidateTree:reviewCandidateTree,
    repairCommit:reviewRepairCommit,repairTree:reviewRepairTree,prNumber:466,
    builderStorePath:join(dir,'builder.sqlite'),
    decisionStorePath:join(dir,'review.sqlite'),repositoryPath:dir,
    reviewer:{operatorId:'zcode:i264:e1:reviewer:1',writerId:'tally-host:win32:reviewer:1',
      writerLogin:'kaicreator-mm',claimRef:'https://github.com/kaicreator-mm/tally/issues/264#issuecomment-7004539047',
      commentId:7004539047},
    builder:{operatorId:'zcode:i264:e1:integration:1',
      writerId:'tally-host:win32:i264:e1:integration:1'},
    reviewRef:'https://github.com/kaicreator-mm/tally/pull/466#pullrequestreview-5971578979',
    githubReads:{
      readPullRequest:async n=>({state:'open',headSha:reviewRepairCommit,baseRef:'version/v2'}),
      readCombinedStatus:async sha=>greenCombinedStatus()},
    ...overrides};
}
async function seedBuilderRecord(config,overrides={}){
  const store=await openSqliteCasStore({path:config.builderStorePath,busyTimeoutMs:1000});
  await store.put(attemptKey(config.attemptId),{phase:'CANDIDATE_READY',
    attemptId:config.attemptId,sourceSha:config.sourceSha,issueRef:config.issueRef,
    claimGeneration:1,candidateRelativePath:LOCAL_CODEX_CONSTANTS.candidateRelativePath,
    candidateTree:reviewCandidateTree,candidateCommit:reviewCandidateCommit,
    candidateRef:'refs/tally-candidates/attempt-264-e1-r91-4',
    worktreePath:'C:/worktrees/attempt-264',...overrides},{mustBeAbsent:true});
  await store.close();
}
function reviewHost(dir,ops){
  return createLocalCodexHost({spawnFn:()=>{throw Error('review must not spawn');},
    operations:ops??reviewOperations(),now:()=>Date.UTC(2026,10,19),killTree:async()=>true});
}
const decisionKeyPrefix=LOCAL_CODEX_CONSTANTS.reviewDecisionKeyPrefix;
const outcomeKeyPrefix=LOCAL_CODEX_CONSTANTS.reviewOutcomeKeyPrefix;

test('single-candidate ACCEPT records the decision and publishes the qualified outcome',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t264-review-'));
  try{
    const config=reviewConfigFor(dir);
    await seedBuilderRecord(config);
    const host=reviewHost(dir);
    const x=await host.reviewSingleCandidate(config);
    assert.equal(x.state,'REVIEW_DECISION_RECORDED');
    assert.equal(x.reason,'REVIEW_DECISION_CAS_READBACK_VERIFIED');
    assert.equal(x.decision,'ACCEPT');
    assert.equal(x.taskPass,true);
    assert.equal(x.releasePass,false);
    assert.equal(x.taskOutcome.status,'QUALIFIED_TASK_OUTCOME_PUBLISHED');
    assert.ok(x.decisionRecord.decisionKey.startsWith(decisionKeyPrefix));
    const store=await openSqliteCasStore({path:config.decisionStorePath,busyTimeoutMs:1000});
    const decisionRow=await store.get(x.decisionRecord.decisionKey);
    assert.equal(decisionRow.value.phase,'REVIEW_DECISION_RECORDED');
    assert.equal(decisionRow.value.decision,'ACCEPT');
    assert.equal(decisionRow.value.reviewer.operatorId,config.reviewer.operatorId);
    assert.equal(decisionRow.value.builder.operatorId,config.builder.operatorId);
    const outcomeRow=await store.get(x.taskOutcome.outcomeKey);
    assert.equal(outcomeRow.value.phase,'QUALIFIED_TASK_OUTCOME_PUBLISHED');
    assert.equal(outcomeRow.value.taskPass??undefined,undefined,
      'outcome row carries no taskPass flag — status is the authority');
    const keys=await store.list(decisionKeyPrefix);
    assert.equal(keys.length,1,'one create-only decision row');
    await store.close();
    // Idempotent replay: same inputs answer ALREADY_RECORDED with the SAME
    // rows — never a second CAS write.
    const again=await host.reviewSingleCandidate(config);
    assert.equal(again.state,'REVIEW_DECISION_RECORDED');
    assert.equal(again.reason,'REVIEW_DECISION_ALREADY_RECORDED');
    assert.equal(again.decisionRecord.decisionKey,x.decisionRecord.decisionKey);
    assert.equal(again.taskOutcome.status,'QUALIFIED_TASK_OUTCOME_PUBLISHED');
    assert.equal(again.taskOutcome.alreadyPublished,true);
    const store2=await openSqliteCasStore({path:config.decisionStorePath,busyTimeoutMs:1000});
    assert.equal((await store2.list(decisionKeyPrefix)).length,1);
    assert.equal((await store2.list(outcomeKeyPrefix)).length,1);
    await store2.close();
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('REWORK and DEFER record the decision but never publish or pass the task',async()=>{
  for(const decision of ['REWORK','DEFER']){
    const dir=await mkdtemp(join(tmpdir(),'tally-t264-review-'+decision.toLowerCase()+'-'));
    try{
      const config=reviewConfigFor(dir,{decision,decisionId:'decision:264:j5:'+decision.toLowerCase()+':1'});
      await seedBuilderRecord(config);
      const x=await reviewHost(dir).reviewSingleCandidate(config);
      assert.equal(x.state,'REVIEW_DECISION_RECORDED');
      assert.equal(x.decision,decision);
      assert.equal(x.taskPass,false);
      assert.equal(x.taskOutcome.status,'HOLD');
      assert.equal(x.taskOutcome.reason,'DECISION_'+decision+'_NOT_TASK_PASS');
      const store=await openSqliteCasStore({path:config.decisionStorePath,busyTimeoutMs:1000});
      assert.equal((await store.list(decisionKeyPrefix)).length,1);
      assert.equal((await store.list(outcomeKeyPrefix)).length,0,
        decision+' must never publish a task outcome row');
      await store.close();
    }finally{await rm(dir,{recursive:true,force:true});}
  }
});

test('single-candidate review denies broken provenance chains fail-closed',async()=>{
  const cases=[
    ['REVIEW_INPUT_INVALID',config=>reviewHost(config.repositoryPath),
      config=>({...config,decision:'SHIP_IT'})],
    ['BUILDER_RECORD_NOT_FOUND',config=>reviewHost(config.repositoryPath),config=>config],
    ['BUILDER_RECORD_NOT_READY',config=>reviewHost(config.repositoryPath),config=>config],
    ['BUILDER_RECORD_STALE',config=>reviewHost(config.repositoryPath),config=>config],
    ['BUILDER_RECORD_MISMATCH',config=>reviewHost(config.repositoryPath),config=>config],
    ['CANDIDATE_READBACK_MISMATCH',
      config=>reviewHost(config.repositoryPath,reviewOperations({
        verifyCandidateReadback:async()=>({verified:true,candidateTree:'f'.repeat(40),
          parent:sourceSha,paths:[LOCAL_CODEX_CONSTANTS.candidateRelativePath]})})),
      config=>config],
    ['REPAIR_READBACK_MISMATCH',
      config=>reviewHost(config.repositoryPath,reviewOperations({
        verifyRepairReadback:async()=>({verified:false})})),
      config=>config],
    ['PR_READBACK_MISMATCH',config=>reviewHost(config.repositoryPath),
      config=>({...config,githubReads:{
        readPullRequest:async()=>({state:'open',headSha:'0'.repeat(40),baseRef:'version/v2'}),
        readCombinedStatus:async()=>greenCombinedStatus()}})],
    ['CI_EXACT_SHA_NOT_VERIFIED',config=>reviewHost(config.repositoryPath),
      config=>({...config,githubReads:{
        readPullRequest:async()=>({state:'open',headSha:reviewRepairCommit,baseRef:'version/v2'}),
        readCombinedStatus:async()=>({state:'failure',totalCount:5})}})],
    // R103 review P1-03: one unrelated green context must never satisfy the
    // qualification CI gate — each task-required named context must be
    // present and terminal-success at the exact repair SHA.
    ['CI_EXACT_SHA_NOT_VERIFIED',config=>reviewHost(config.repositoryPath),
      config=>({...config,githubReads:{
        readPullRequest:async()=>({state:'open',headSha:reviewRepairCommit,baseRef:'version/v2'}),
        readCombinedStatus:async()=>({state:'success',totalCount:1,
          contexts:[{context:'ci/woodpecker/pr/v2-unrelated',state:'success'}]})}})],
    ['CI_EXACT_SHA_NOT_VERIFIED',config=>reviewHost(config.repositoryPath),
      config=>({...config,githubReads:{
        readPullRequest:async()=>({state:'open',headSha:reviewRepairCommit,baseRef:'version/v2'}),
        readCombinedStatus:async()=>({state:'success',totalCount:4,
          contexts:reviewRequiredContexts.slice(0,4)
            .map(context=>({context,state:'success'}))})}})],
    ['CI_EXACT_SHA_NOT_VERIFIED',config=>reviewHost(config.repositoryPath),
      config=>({...config,githubReads:{
        readPullRequest:async()=>({state:'open',headSha:reviewRepairCommit,baseRef:'version/v2'}),
        readCombinedStatus:async()=>({state:'pending',totalCount:5,
          contexts:reviewRequiredContexts.map((context,index)=>({context,
            state:index===0?'pending':'success'}))})}})],
    ['REVIEW_DECISION_CONFLICT',config=>reviewHost(config.repositoryPath),config=>config],
  ];
  let index=0;
  for(const [reason,mkHost,configFn] of cases){
    const dir=await mkdtemp(join(tmpdir(),'tally-t264-deny-'));
    try{
      const base=reviewConfigFor(dir);
      if(reason==='BUILDER_RECORD_NOT_READY')
        await seedBuilderRecord(base,{phase:'FAILED'});
      else if(reason==='BUILDER_RECORD_STALE')
        await seedBuilderRecord(base,{sourceSha:'f'.repeat(40)});
      else if(reason==='BUILDER_RECORD_MISMATCH')
        await seedBuilderRecord(base,{candidateCommit:'e'.repeat(40)});
      else if(reason!=='BUILDER_RECORD_NOT_FOUND'&&reason!=='REVIEW_INPUT_INVALID'&&
        reason!=='REVIEW_DECISION_CONFLICT')
        await seedBuilderRecord(base);
      if(reason==='REVIEW_DECISION_CONFLICT'){
        await seedBuilderRecord(base);
        // Pre-seed the exact idempotency key with a DIFFERENT decision: the
        // create-only key must refuse the conflicting second decision.
        const {createHash}=await import('node:crypto');
        const decisionKey=decisionKeyPrefix+createHash('sha256')
          .update(JSON.stringify([base.issueRef,base.sourceSha,base.attemptId,
            base.candidateCommit,base.repairCommit,base.reviewer.operatorId]))
          .digest('hex');
        const store=await openSqliteCasStore({path:base.decisionStorePath,busyTimeoutMs:1000});
        await store.put(decisionKey,{schema:'tally/single-review-decision/v1',
          phase:'REVIEW_DECISION_RECORDED',decision:'REWORK',
          sourceSha:base.sourceSha,candidateCommit:base.candidateCommit,
          repairCommit:base.repairCommit,
          reviewer:{operatorId:base.reviewer.operatorId}},{mustBeAbsent:true});
        await store.close();
      }
      const host=mkHost(base);
      const x=await host.reviewSingleCandidate(configFn(base));
      assert.equal(x.state,'NOT_ADMITTED',reason+' #'+index);
      assert.equal(x.reason,reason,reason+' #'+index);
      assert.equal(x.taskPass,false);
      assert.equal(x.taskOutcome,null);
      // No denial may write a decision row (the pre-seeded conflict fixture
      // stays untouched at exactly one) and none may ever write an outcome.
      const store=await openSqliteCasStore({path:base.decisionStorePath,busyTimeoutMs:1000});
      const decisionRows=await store.list(decisionKeyPrefix);
      assert.equal(decisionRows.length,reason==='REVIEW_DECISION_CONFLICT'?1:0,
        reason+' must not write a decision row');
      assert.equal((await store.list(outcomeKeyPrefix)).length,0,
        reason+' must never write an outcome row');
      await store.close();
    }finally{await rm(dir,{recursive:true,force:true});}
    index++;
  }
});

test('an unreadable decision store stays UNKNOWN for read-only reconciliation',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t264-unknown-'));
  try{
    const config=reviewConfigFor(dir,{decisionStorePath:join(dir,'no-dir','review.sqlite')});
    await seedBuilderRecord(config);
    const x=await reviewHost(dir).reviewSingleCandidate(config);
    assert.equal(x.state,'REVIEW_DECISION_UNKNOWN');
    assert.equal(x.reason,'REVIEW_DECISION_ACK_LOST');
    assert.equal(x.reconciliationRequired,true);
    assert.equal(x.taskPass,false);
    assert.equal(x.decisionRecord,null);
  }finally{await rm(dir,{recursive:true,force:true});}
});

// ---------------------------------------------------------------------------
// R103 review P1-02 — the REAL git-backed occupancy guard must distinguish
// proven answers from unproven git failures; a git error may never read as
// "not occupied" (fail-open) and never spawn the Builder.
// ---------------------------------------------------------------------------
test('real candidatePathOccupied classifies git exit 0 / proven-absent / unproven failures',async()=>{
  const occupancySpawn=(code,stderr)=>{
    return (command,args,options)=>fakeChild((close,child)=>{
      if(stderr!==undefined)child.stderr.emit('data',Buffer.from(stderr));
      close(code,null);
    });
  };
  const cases=[
    // exit 0: the path IS tracked at the pinned source SHA.
    {spawn:occupancySpawn(0,''),expected:{occupied:true},label:'occupied'},
    // git's proven-absent signature for cat-file -e <ref>:<path>.
    {spawn:occupancySpawn(128,"fatal: path 'x.md' does not exist in 'dff5329e'\n"),
      expected:{occupied:false},label:'proven-absent'},
    // Any other nonzero shape is UNPROVEN — bad ref, bad repository, missing
    // git binary, timeout — and must never read as absence.
    {spawn:occupancySpawn(128,'fatal: Not a valid object name badref\n'),
      expected:{occupied:null},label:'bad-ref'},
    {spawn:occupancySpawn(128,''),expected:{occupied:null},label:'no-stderr'},
    {spawn:occupancySpawn(null,''),expected:{occupied:null},label:'launch-failure'},
  ];
  for(const {spawn:spawnFn,expected,label} of cases){
    const ops=createDefaultOperations(spawnFn);
    assert.equal(typeof ops.candidatePathOccupied,'function');
    const result=await ops.candidatePathOccupied({sourceSha:'d'.repeat(40),
      repositoryPath:'C:\repo',gitPath:'C:\git.exe'});
    assert.equal(result.occupied,expected.occupied,label);
    if(expected.occupied===null)assert.ok(typeof result.detail==='string',label);
  }
  // The separately authorized replacement short-circuits before any git run.
  let ran=false;
  const ops=createDefaultOperations((...a)=>{ran=true;return fakeChild(close=>close(0,null));});
  const bypass=await ops.candidatePathOccupied({sourceSha:'d'.repeat(40),
    repositoryPath:'C:\repo',gitPath:'C:\git.exe',candidateReplacementAuthorized:true});
  assert.equal(bypass.occupied,false);
  assert.equal(ran,false,'authorized replacement must not run git');
});

// ---------------------------------------------------------------------------
// R103 review P1-01 — real-branch reviewer registry admission: the reviewer
// iteration must read the REVIEWER claim key from the reviewer registry.
// Runs only where the consumer's sealed dependencies resolve (the CI
// run-clean lane installs them); otherwise an honest skip, never a fake pass.
// ---------------------------------------------------------------------------
test('consumer real branch admits per-role registry keys and rejects wrong-key rows',async(t)=>{
  const consumerUrl=new URL('./consumer-single-review-codex.mjs',import.meta.url);
  let consumerPresent=false;
  try{await stat(fileURLToPath(consumerUrl));consumerPresent=true;}catch{}
  if(!consumerPresent)
    return void t.skip('T264 consumer not present in this lane (run-clean copies it only under T264_SINGLE_REVIEW; other lanes do not seal the single-review route)');
  let depsResolved=true;
  try{import.meta.resolve('@kaicreator/tally-local-codex-host-bundle',consumerUrl);}
  catch{depsResolved=false;}
  if(!depsResolved)
    return void t.skip('T264 consumer sealed deps not installed in this env (CI run-clean lane covers this regression)');
  const {spawn}=await import('node:child_process');
  const {createHash}=await import('node:crypto');
  const claimKey=role=>`kaicreator-mm/tally#264:${role}:__default__`;
  const registryKey=claim=>'tally:v2:claim-admission:'+
    createHash('sha256').update(claim).digest('hex');
  const seedRegistry=async(path,protectedClaimKey)=>{
    const store=await openSqliteCasStore({path,busyTimeoutMs:1000});
    await store.put(registryKey(protectedClaimKey),{phase:'PUBLISHED_VERIFIED',
      generation:1,protectedClaimKey},{mustBeAbsent:true});
    await store.close();
  };
  const runConsumer=async(dir,reviewerRegistryKey)=>{
    const builderRegistry=join(dir,'builder-claim.sqlite');
    const reviewerRegistry=join(dir,'reviewer-claim.sqlite');
    const builderStore=join(dir,'builder-store.sqlite');
    const store=await openSqliteCasStore({path:builderStore,busyTimeoutMs:1000});
    await store.close();
    await seedRegistry(builderRegistry,claimKey('builder'));
    await seedRegistry(reviewerRegistry,reviewerRegistryKey);
    const evidence=join(dir,'evidence');
    const env={...process.env,TALLY_LOCAL_REPLAY:'0',
      TALLY_GITHUB_READ_TOKEN:'fixture-not-a-secret',
      TALLY_CLAIM_SOURCE_SHA:'d'.repeat(40),
      TALLY_LOCAL_ISSUE_NUMBER:'264',
      TALLY_LOCAL_EVIDENCE_DIR:evidence,
      TALLY_LOCAL_BUILDER_STORE_PATH:builderStore,
      TALLY_LOCAL_REVIEW_STORE_PATH:join(dir,'review-store.sqlite'),
      TALLY_TASK_CLAIM_264_DISPATCH_ID:'dispatch:264:t:test:builder:1',
      TALLY_TASK_CLAIM_264_OPERATOR_ID:'zcode:t264:test:builder:1',
      TALLY_TASK_CLAIM_264_OPERATOR_KIND:'other',
      TALLY_TASK_CLAIM_264_WRITER_ID:'tally-host:win32:t264:test:builder:1',
      TALLY_TASK_CLAIM_264_WRITER_LOGIN:'kaicreator-mm',
      TALLY_TASK_CLAIM_264_COMMENT_ID:'1',
      TALLY_TASK_CLAIM_264_REGISTRY_PATH:builderRegistry,
      TALLY_TASK_CLAIM_264_REVIEWER_DISPATCH_ID:'dispatch:264:t:test:reviewer:1',
      TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_ID:'zcode:t264:test:reviewer:1',
      TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_KIND:'other',
      TALLY_TASK_CLAIM_264_REVIEWER_WRITER_ID:'tally-host:win32:t264:test:reviewer:1',
      TALLY_TASK_CLAIM_264_REVIEWER_WRITER_LOGIN:'kaicreator-mm',
      TALLY_TASK_CLAIM_264_REVIEWER_COMMENT_ID:'2',
      TALLY_TASK_CLAIM_264_REVIEWER_REGISTRY_PATH:reviewerRegistry};
    const child=spawn(process.execPath,[fileURLToPath(consumerUrl)],{env,windowsHide:true});
    let stdout='',stderr='';
    child.stdout.on('data',c=>{stdout+=c;});
    child.stderr.on('data',c=>{stderr+=c;});
    const exit=await new Promise(resolve=>{
      const timer=setTimeout(()=>{child.kill();resolve({code:null,timeout:true});},120000);
      child.once('close',(code,signal)=>{clearTimeout(timer);resolve({code,signal});});
    });
    return {exit,stdout,stderr};
  };
  // Positive: separate per-role keys in separate registries. The real branch
  // must pass claim admission and terminate with the bounded consumer result
  // (a downstream NOT_ADMITTED effect is expected: the fixture builder store
  // carries no CANDIDATE_READY row). Under the R103 defect this crashed with
  // T264_CLAIM_NOT_PUBLISHED because the reviewer registry was read with the
  // builder key.
  {
    const dir=await mkdtemp(join(tmpdir(),'tally-t264-p1-01-pos-'));
    try{
      const {exit,stdout,stderr}=await runConsumer(dir,claimKey('reviewer'));
      assert.equal(exit.code,0,'positive real-branch admission must not crash: '+stderr.slice(-800));
      assert.ok(!stderr.includes('T264_CLAIM_NOT_PUBLISHED'),
        'per-role keys must pass the real-branch claim admission');
      assert.ok(stdout.includes('T264_SINGLE_REVIEW_K_HOST_DECISION=PASS'),stdout.slice(-800));
    }finally{await rm(dir,{recursive:true,force:true});}
  }
  // Negative (wrong key): the reviewer registry holds the BUILDER claim key —
  // exactly the row the R103 defect read — so the reviewer-role lookup must
  // fail closed with T264_CLAIM_NOT_PUBLISHED.
  {
    const dir=await mkdtemp(join(tmpdir(),'tally-t264-p1-01-neg-'));
    try{
      const {exit,stderr}=await runConsumer(dir,claimKey('builder'));
      assert.notEqual(exit.code,0,'wrong-key reviewer registry must fail closed');
      assert.ok(stderr.includes('T264_CLAIM_NOT_PUBLISHED'),stderr.slice(-800));
    }finally{await rm(dir,{recursive:true,force:true});}
  }
});

// ---------------------------------------------------------------------------
// #473 R107 repair 1 — readAttemptRecord is a strictly read-only reconcile
// port: same vocabulary as runAttempt's durable replay, but a missing row
// stays ATTEMPT_RECORD_NOT_FOUND and NOTHING is ever created, granted or
// launched (R106 ABI review P1: runAttempt can launch when no row exists).
// ---------------------------------------------------------------------------
test('readAttemptRecord replays durable states without any side effect',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'t264-readonly-replay-'));
  try{
    const config=configFor(dir,'attempt:t06m:replay:1');
    const host=createLocalCodexHost({spawnFn:()=>
      {throw Error('readAttemptRecord must never spawn');},now:()=>Date.UTC(2026,10,19)});
    // The hardened read-only port (readonly+fileMustExist) cannot even open
    // a missing store file: a truly absent store is STORE_UNAVAILABLE, and
    // the read must not create the file or any row either way.
    const noStore=await host.readAttemptRecord(config);
    assert.equal(noStore.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(noStore.reason,'ATTEMPT_RECORD_STORE_UNAVAILABLE');
    assert.equal(noStore.readOnly,true);
    assert.equal(existsSync(config.hostStorePath),false,
      'a read-only reconcile must not create the store file');
    // Absent row in an EXISTING store: honest NOT_FOUND.
    let store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    await store.close();
    const absent=await host.readAttemptRecord(config);
    assert.equal(absent.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(absent.reason,'ATTEMPT_RECORD_NOT_FOUND');
    assert.equal(absent.readOnly,true);
    assert.equal(absent.agentLaunch,false);
    store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    assert.equal((await store.list(LOCAL_CODEX_CONSTANTS.attemptKeyPrefix)).length,0,
      'a read-only reconcile must not create an attempt row');
    await store.close();
    // Terminal NOT_EXECUTED row replays as AGENT_NOT_EXECUTED (full attempt
    // identity recorded — the hardened read-only port cross-checks it).
    store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    await store.put(attemptKey(config.attemptId),{phase:'NOT_EXECUTED',
      reason:'CANDIDATE_PATH_OCCUPIED',sourceSha:config.sourceSha,
      issueRef:config.issueRef,attemptId:config.attemptId},{mustBeAbsent:true});
    await store.close();
    const refused=await host.readAttemptRecord(config);
    assert.equal(refused.state,'AGENT_NOT_EXECUTED');
    assert.equal(refused.reason,'CANDIDATE_PATH_OCCUPIED');
    assert.equal(refused.readOnly,true);
    // CANDIDATE_READY row replays with its candidate identity.
    const ready=configFor(dir,'attempt:t06m:replay:2');
    store=await openSqliteCasStore({path:ready.hostStorePath,busyTimeoutMs:1000});
    await store.put(attemptKey(ready.attemptId),{phase:'CANDIDATE_READY',
      sourceSha:ready.sourceSha,issueRef:ready.issueRef,attemptId:ready.attemptId,
      candidateTree:'e'.repeat(40),candidateCommit:'f'.repeat(40),
      process:{status:'EXITED_OK',exitCode:0,pid:4242}},{mustBeAbsent:true});
    await store.close();
    const complete=await host.readAttemptRecord(ready);
    assert.equal(complete.state,'LOCAL_CANDIDATE_READY');
    assert.equal(complete.candidateCommit,'f'.repeat(40));
    assert.equal(complete.readOnly,true);
    // Unreadable store: honest UNKNOWN, never a guess.
    const broken=await host.readAttemptRecord({...configFor(dir,'attempt:t06m:replay:3'),
      hostStorePath:join(dir,'no-dir','host.sqlite')});
    assert.equal(broken.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(broken.reason,'ATTEMPT_RECORD_STORE_UNAVAILABLE');
  }finally{await rm(dir,{recursive:true,force:true});}
});

// ---------------------------------------------------------------------------
// #473 R107 — AgentProvider/1 substitutes only the process execution point.
// The existing Host CAS/grants/worktree/write-set/candidate path remains the
// sole authority; no parallel Host store or scheduler is involved.
// ---------------------------------------------------------------------------
function providerFake(id,execute){
  return {id,
    probe:async()=>({state:'AVAILABLE',reason:null,evidence:null}),
    execute,cancel:async()=>({state:'NOT_RUNNING',reason:null,evidence:null}),
    reconcile:async()=>({state:'UNKNOWN',reason:'NOT_RECONCILED',evidence:null})};
}

test('AgentProvider candidate uses the same Host CAS and cannot rebind an attempt',async()=>{
  await withHost(async({host,config,calls})=>{
    let seen=null;
    const provider=providerFake('zcode',async request=>{
      seen=request;
      return {state:'COMPLETED',reason:null,evidence:{pid:4343,exitCode:0,spawned:true}};
    });
    const first=await host.runAttempt({...config,providerId:'zcode',agentProvider:provider,
      providerEnv:{DEEPSEEK_API_KEY:'not-recorded'}});
    assert.equal(first.state,'LOCAL_CANDIDATE_READY');
    assert.equal(first.providerId,'zcode');
    assert.equal(first.taskPass,false);
    assert.equal(first.process.pid,4343);
    assert.equal(first.process.builderPrincipal,null,
      'an adapter process is not mislabeled as the Codex sandbox principal');
    assert.equal(calls.length,0,'the provider owns its process; Host spawns no Codex CLI');
    assert.equal(seen.taskId,'172');
    assert.equal(seen.worktreePath,config.worktreePath);
    assert.ok(seen.prompt.includes(LOCAL_CODEX_CONSTANTS.candidateRelativePath));
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    const row=await store.get(attemptKey(config.attemptId));
    assert.equal(row.value.providerId,'zcode');
    assert.equal(row.value.phase,'CANDIDATE_READY');
    assert.equal(JSON.stringify(row).includes('not-recorded'),false);
    assert.deepEqual(row.value.envAllowlist,['DEEPSEEK_API_KEY']);
    await store.close();
    const rebound=await host.runAttempt({...config,providerId:'dsh',
      agentProvider:providerFake('dsh',async()=>{
        throw Error('a rebound attempt must never execute');})});
    assert.equal(rebound.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(rebound.reason,'ATTEMPT_IDENTITY_MISMATCH');
    assert.equal(rebound.agentLaunch,false);
  });
});

test('Pi-shaped COMPLETED without optional PID/exitCode maps to sealed EXITED/0',async()=>{
  await withHost(async({host,config})=>{
    const result=await host.runAttempt({...config,providerId:'pi',
      agentProvider:providerFake('pi',async()=>({state:'COMPLETED',reason:null,
        evidence:{spawned:true}}))});
    assert.equal(result.state,'LOCAL_CANDIDATE_READY');
    assert.equal(result.providerId,'pi');
    assert.equal(result.process.status,'EXITED');
    assert.equal(result.process.exitCode,0);
    assert.equal(result.process.pid,null);
    assert.equal(result.process.builderPrincipal,null);
    assert.equal(result.taskPass,false);
  });
});

test('two providers share existing Host grants for disjoint parallel Tasks',async()=>{
  await withHost(async({host,config})=>{
    let active=0,maxActive=0;
    const execute=async()=>{
      active++;maxActive=Math.max(maxActive,active);
      await new Promise(resolve=>setTimeout(resolve,40));
      active--;
      return {state:'COMPLETED',reason:null,evidence:{pid:4343,exitCode:0,spawned:true}};
    };
    const baseA={...config,attemptId:'attempt:r107:parallel:a',
      worktreePath:join(config.repositoryPath,'wt-a'),candidateRelativePath:CONCURRENT_A};
    const baseB={...config,attemptId:'attempt:r107:parallel:b',
      worktreePath:join(config.repositoryPath,'wt-b'),candidateRelativePath:CONCURRENT_B};
    const [a,b]=await Promise.all([
      host.runAttempt({...baseA,providerId:'zcode',agentProvider:providerFake('zcode',execute)}),
      host.runAttempt({...baseB,providerId:'dsh',agentProvider:providerFake('dsh',execute)})]);
    assert.equal(a.state,'LOCAL_CANDIDATE_READY');assert.equal(b.state,'LOCAL_CANDIDATE_READY');
    assert.equal(maxActive,2,'two disjoint provider executions really overlapped');
    const store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:1000});
    const rows=await store.list(LOCAL_CODEX_CONSTANTS.attemptKeyPrefix);
    assert.equal(rows.length,2);
    const values=[];
    for(const key of rows)values.push((await store.get(key)).value);
    assert.deepEqual(new Set(values.map(value=>value.providerId)),new Set(['zcode','dsh']));
    assert.equal((await store.list('tally:v2:local-agent-grant:')).length,0,
      'settled attempts release the same existing Host grants');
    await store.close();
  });
});

test('provider overlap refusal and COMPLETED write-set violation stay Host-owned',async()=>{
  await withHost(async({host,config})=>{
    let release;
    const gate=new Promise(resolve=>{release=resolve;});
    let started;
    const entered=new Promise(resolve=>{started=resolve;});
    const running=host.runAttempt({...config,providerId:'zcode',
      agentProvider:providerFake('zcode',async()=>{
        started();await gate;
        return {state:'COMPLETED',reason:null,evidence:{pid:4343,exitCode:0,spawned:true}};
      })});
    await entered;
    const overlap=await host.runAttempt({...config,attemptId:'attempt:r107:overlap:b',
      worktreePath:join(config.repositoryPath,'wt-other'),providerId:'dsh',
      agentProvider:providerFake('dsh',async()=>{
        throw Error('overlapping provider must not execute');})});
    assert.equal(overlap.state,'NOT_ADMITTED');
    assert.equal(overlap.reason,'CANDIDATE_WRITE_GRANT_OVERLAP');
    release();assert.equal((await running).state,'LOCAL_CANDIDATE_READY');
    const violation=await host.runAttempt({...config,attemptId:'attempt:r107:violation:b',
      worktreePath:join(config.repositoryPath,'wt-violation'),providerId:'dsh',
      agentProvider:providerFake('dsh',async()=>({state:'COMPLETED',reason:null,
        evidence:{pid:4343,exitCode:0,spawned:true}}))});
    assert.equal(violation.state,'LOCAL_ATTEMPT_FAILED');
    assert.equal(violation.reason,'WRITE_SET_VIOLATION');
    assert.equal(violation.taskPass,false);
  },{ops:operations({finalizeCandidate:async config=>
    config.attemptId==='attempt:r107:violation:b'?{writeSetValid:false,paths:['forbidden.txt']}:candidate})});
});
