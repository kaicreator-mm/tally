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
  },{ops:operations({candidatePathOccupied:async()=>true})});
  // An explicitly authorized replacement is the only way past the guard
  // (a fresh attempt id: the refusal above left a terminal NOT_EXECUTED row).
  await withHost(async({host,calls,config})=>{
    const x=await host.runAttempt({...config,attemptId:'attempt:t06m:replace:1',
      candidateReplacementAuthorized:true});
    assert.equal(x.state,'LOCAL_CANDIDATE_READY');
    assert.equal(calls.length,1);
  },{ops:operations({candidatePathOccupied:async config=>
    config.candidateReplacementAuthorized!==true})});
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
      readCombinedStatus:async sha=>({state:'success',totalCount:5})},
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
        readCombinedStatus:async()=>({state:'success',totalCount:5})}})],
    ['CI_EXACT_SHA_NOT_VERIFIED',config=>reviewHost(config.repositoryPath),
      config=>({...config,githubReads:{
        readPullRequest:async()=>({state:'open',headSha:reviewRepairCommit,baseRef:'version/v2'}),
        readCombinedStatus:async()=>({state:'failure',totalCount:5})}})],
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
