/** #264 E1 — installed entrypoint regression (R90 minimal repair + R91
 * installed write-journey seam).
 *
 * Covers, with zero network and zero credentials:
 *  - real ESM load of web/v2/serve.mjs (the five missing ui-*-host.mjs
 *    unconditional static imports P0 — this file fails if any import target
 *    is unresolved again);
 *  - selected Host environment admission (positive + fail-closed matrix,
 *    including the read-only frozen DAG reader pins);
 *  - journey-server v2 GET models: honest NOT_CONNECTED without a mounted
 *    Host provider, pass-through of the Host-composed selected source, and
 *    optional-provider absence (settings/metadata never fabricate state);
 *  - a real boot of the installed read-only entrypoint (redirect, fail-closed
 *    roster without frozen host, scope guard) that repeats cleanly;
 *  - R91 write-journey seam: trusted Host pin admission matrix, the
 *    injected-runner start/ACK-lost/fail/timeout contract, the review seam
 *    that never fabricates an executed attempt, stale-claim START refusal,
 *    the /api/v2/agent-attempts projection that renders ONLY real
 *    executions (the R90 stub run must stay invisible), and the
 *    /api/v2/release evidence passthrough.
 *
 * No GitHub call can start here: the roster host denies immediately without
 * a mounted frozen reader and the catalog model is never read in this file.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {join} from 'node:path';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
// Static import: evaluating this module resolves every serve.mjs static
// import target — the dedicated test below documents that contract.
import {readUiHostEnvironment,startUiReadOnlyServer} from './serve.mjs';
import {readInstalledWriteJourneyEnvironment,createInstalledWriteJourneyRunner}
  from './installed-write-journeys.mjs';

const sha='a'.repeat(40);
const otherSha='b'.repeat(40);
const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',
  tallyBPath:'/tally',githubToken:'nonpublicfixture',
  currentSourceSha:sha,workRoot:'/tmp/tally-v2-installed-entrypoint'});
const validEnv={TALLY_V2_REPOSITORY:'kaicreator-mm/tally',
  TALLY_V2_VERSION_REF:'version/v2',TALLY_V2_VERSION_SHA:sha,
  TALLY_V2_TASK_ISSUES_JSON:'[196,264]',
  TALLY_GITHUB_READ_TOKEN:'nonpublicfixture'};

test('installed entrypoint ESM load resolves every static import',()=>{
  assert.equal(typeof readUiHostEnvironment,'function');
  assert.equal(typeof startUiReadOnlyServer,'function');
});

test('selected Host environment admits a valid read-only configuration',()=>{
  const settings=readUiHostEnvironment(validEnv);
  assert.equal(settings.repository,'kaicreator-mm/tally');
  assert.equal(settings.owner,'kaicreator-mm');
  assert.equal(settings.versionId,'version/v2');
  assert.deepEqual(settings.roster,[196,264]);
  assert.equal(settings.frozenDagReadOnly,false);
  assert.equal(settings.auditReadOnly,false);
  assert.equal(settings.workbenchReadOnly,false);
});
test('selected Host environment refuses missing token and malformed roster',()=>{
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_GITHUB_READ_TOKEN:undefined}),
    /SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_GITHUB_READ_TOKEN:'short'}),
    /SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_TASK_ISSUES_JSON:'not-json'}),/INVALID_HOST_TASK_ROSTER/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_TASK_ISSUES_JSON:'[196,196]'}),/INVALID_HOST_TASK_ROSTER/);
});
test('frozen DAG read mode requires roster, registry and witness pins',()=>{
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_FROZEN_DAG_HOST:'write'}),
    /INVALID_HOST_FROZEN_DAG_READ_MODE/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_TASK_ISSUES_JSON:'[]',TALLY_V2_FROZEN_DAG_HOST:'read-only'}),
    /HOST_FROZEN_DAG_TASK_ROSTER_REQUIRED/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_FROZEN_DAG_HOST:'read-only'}),
    /HOST_FROZEN_DAG_REGISTRY_REQUIRED/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_FROZEN_DAG_HOST:'read-only',
    TALLY_VERSION_DAG_REGISTRY_PATH:'/tmp/registry.sqlite'}),
    /HOST_FROZEN_DAG_WITNESS_PINS_REQUIRED/);
  const settings=readUiHostEnvironment({...validEnv,
    TALLY_V2_FROZEN_DAG_HOST:'read-only',
    TALLY_VERSION_DAG_REGISTRY_PATH:'/tmp/registry.sqlite',
    TALLY_VERSION_DAG_REVIEW_ISSUE:'392',
    TALLY_VERSION_DAG_REVIEW_COMMENT_ID:'6976912326',
    TALLY_VERSION_DAG_REVIEW_LOGIN:'kaicreator-mm',
    TALLY_VERSION_DAG_APPROVAL_ISSUE:'392',
    TALLY_VERSION_DAG_APPROVAL_COMMENT_ID:'6574693761',
    TALLY_VERSION_DAG_APPROVAL_LOGIN:'kaicreator-mm'});
  assert.equal(settings.frozenDagReadOnly,true);
  assert.equal(settings.frozenPins.registryPath,'/tmp/registry.sqlite');
  assert.equal(settings.frozenPins.reviewIssue,392);
  assert.equal(settings.frozenPins.reviewCommentId,6976912326);
});

test('v2 GET models answer honest NOT_CONNECTED without a Host read model',
  async()=>{
    const app=createAlmServer(config);
    const {origin}=await app.listen();
    try{
      for(const [path,model] of [['/api/v2/catalog','catalog'],
        ['/api/v2/tasks','tasks'],['/api/v2/dag','dag'],
        ['/api/v2/settings','settings'],['/api/v2/metadata','metadata']]){
        const response=await fetch(origin+path);
        assert.equal(response.status,200,path);
        const payload=await response.json();
        assert.equal(payload.connected,false,path);
        assert.equal(payload.reason,'NOT_CONNECTED',path);
        assert.equal(payload.model,model,path);
      }
      // The advisory Release projection stays honest UNKNOWN (no fabricated
      // gate or permit), independent of any Host read model.
      const release=await (await fetch(origin+'/api/v2/release')).json();
      assert.equal(release.connected,true);
      assert.deepEqual(release.readiness.gates.map(g=>g.status),
        ['UNKNOWN','UNKNOWN']);
    }finally{await app.close();}
  });

test('mounted Host read model passes through and absent providers degrade',
  async()=>{
    const rosterPayload={connected:true,
      source:{repository:'kaicreator-mm/tally',sourceSha:sha,
        currentness:'CURRENT',readback:'VERIFIED'},
      versionId:'version/v2',rosterCoverage:'VERIFIED_COMPLETE',
      dependencyCoverage:'NATIVE_COMPLETE',taskCount:1,nativeEdgeCount:1,
      frozenBlobSha:otherSha,
      taskPackRef:'https://github.com/kaicreator-mm/tally/blob/'+sha+
        '/task-pack.json',
      claimReadbackCoverage:'NOT_VERIFIED',claimCoverage:'NOT_VERIFIED',
      tasks:[{id:'196',versionId:'version/v2',title:'fixture task',
        status:'open',severity:'UNKNOWN',issueState:'open',
        closureState:'OPEN',blockedBy:[],agentId:null,
        claimState:'NOT_VERIFIED',attemptId:null,attemptState:null,
        prNumber:null,reviewState:'UNKNOWN',ciState:'UNKNOWN',writeSet:[],
        requirementRef:null,updatedAt:null}]};
    const dagPayload={connected:true,
      source:rosterPayload.source,
      versionId:'version/v2',rosterCoverage:'VERIFIED_COMPLETE',
      claimReadbackCoverage:'NOT_VERIFIED',claimCoverage:'NOT_VERIFIED',
      dependencyCoverage:'NATIVE_COMPLETE',frozenBlobSha:otherSha,
      taskPackRef:rosterPayload.taskPackRef,
      asOf:{sourceSha:sha,frozenAt:'2026-11-12T00:00:00.000Z'},
      nodes:[{taskId:'196',versionId:'version/v2',title:'fixture task',
        status:'open',severity:'UNKNOWN',requirementGroup:null,phase:null,
        lane:null,agentId:null,executable:false}],
      edges:[]};
    const calls=[];
    const host={tasks:async scope=>{calls.push(['tasks',scope]);
        return rosterPayload;},
      dag:async scope=>{calls.push(['dag',scope]);return dagPayload;}};
    const app=createAlmServer(config,{v2ReadModelHost:host});
    const {origin}=await app.listen();
    try{
      const tasks=await (await fetch(
        origin+'/api/v2/tasks?versionId=version%2Fv2')).json();
      assert.equal(tasks.connected,true);
      assert.equal(tasks.rosterCoverage,'VERIFIED_COMPLETE');
      assert.equal(tasks.tasks[0].id,'196');
      assert.equal(tasks.tasks[0].severity,'UNKNOWN');
      assert.deepEqual(calls.at(-1),['tasks',{versionId:'version/v2'}]);
      const dag=await (await fetch(
        origin+'/api/v2/dag?versionId=version%2Fv2')).json();
      assert.equal(dag.connected,true);
      assert.equal(dag.asOf.sourceSha,sha);
      assert.equal(dag.nodes[0].taskId,'196');
      // Settings/metadata have no mounted provider even when other models
      // exist: honest NOT_CONNECTED, never a synthetic snapshot or zeros.
      const settings=await (await fetch(origin+'/api/v2/settings')).json();
      assert.equal(settings.connected,false);
      assert.equal(settings.model,'settings');
      assert.equal(settings.snapshot,undefined);
      const metadata=await (await fetch(
        origin+'/api/v2/metadata?versionId=version%2Fv2')).json();
      assert.equal(metadata.connected,false);
      assert.equal(metadata.model,'metadata');
      assert.equal(metadata.attempts,undefined);
      // A Host provider that throws degrades to NOT_CONNECTED, not a 500.
      const failing=createAlmServer(config,{v2ReadModelHost:{
        tasks:async()=>{throw new Error('PROVIDER_DOWN');}}});
      const {origin:failureOrigin}=await failing.listen();
      try{
        const degraded=await (await fetch(
          failureOrigin+'/api/v2/tasks?versionId=version%2Fv2')).json();
        assert.equal(degraded.connected,false);
        assert.equal(degraded.reason,'NOT_CONNECTED');
        assert.equal(degraded.model,'tasks');
      }finally{await failing.close();}
    }finally{await app.close();}
  });

test('installed read-only entrypoint boots, serves fail-closed reads and repeats',
  async()=>{
    const first=await startUiReadOnlyServer({env:{...validEnv}});
    try{
      assert.match(first.address.origin,/127\.0\.0\.1:\d+/);
      const redirect=await fetch(first.address.origin+'/',
        {redirect:'manual'});
      assert.equal(redirect.status,302);
      assert.equal(redirect.headers.get('location'),'/app');
      const diagnostics=await fetch(first.address.origin+'/diagnostics');
      assert.equal(diagnostics.status,200);
      // Without the frozen DAG reader pins the roster host denies
      // immediately and fail-closed — no raw open-Issue fallback exists.
      const tasks=await (await fetch(
        first.address.origin+'/api/v2/tasks?versionId=version%2Fv2')).json();
      assert.equal(tasks.connected,false);
      assert.equal(tasks.reason,'FROZEN_DAG_HOST_NOT_CONNECTED');
      assert.equal(tasks.rosterCoverage,'UNVERIFIED');
      assert.equal(tasks.tasks,undefined);
      // Scope guard: a foreign version never reaches the selected source.
      const foreign=await (await fetch(
        first.address.origin+'/api/v2/tasks?versionId=version%2Fother')).json();
      assert.equal(foreign.connected,false);
      assert.equal(foreign.reason,'PROJECT_VERSION_SCOPE_MISMATCH');
      const dag=await (await fetch(
        first.address.origin+'/api/v2/dag?versionId=version%2Fv2')).json();
      assert.equal(dag.connected,false);
      assert.equal(dag.reason,'FROZEN_DAG_HOST_NOT_CONNECTED');
    }finally{await first.close();}
    // Repeatable bootstrap: a second start on a fresh port works with no
    // stale socket and the same fail-closed read semantics.
    const second=await startUiReadOnlyServer({env:{...validEnv}});
    try{
      const tasks=await (await fetch(
        second.address.origin+'/api/v2/tasks?versionId=version%2Fv2')).json();
      assert.equal(tasks.connected,false);
      assert.equal(tasks.reason,'FROZEN_DAG_HOST_NOT_CONNECTED');
    }finally{await second.close();}
  });

// ---------------------------------------------------------------------------
// #264 E1 R91 — installed write-journey seam
// ---------------------------------------------------------------------------
const consumerPins={TALLY_GITHUB_READ_TOKEN:'nonpublicfixture',
  TALLY_CODEX_CLI_PATH:'C:/codex/codex.exe',
  TALLY_CODEX_SANDBOX_PATH:'C:/codex/codex.exe',
  TALLY_BUILDER_PROFILE:'C:/Users/CodexSandboxOnline',
  TALLY_BUILDER_TEMP:'C:/tmp/builder-temp',
  TALLY_EMPTY_GIT_CONFIG:'C:/tmp/empty-gitconfig',
  TALLY_OPERATOR_HOME:'C:/Users/15549',
  TALLY_HIDDEN_FIXTURE:'C:/tmp/hidden/heldout.txt',
  TALLY_BUILDER_PRINCIPAL:'nilar\\codexsandboxonline',
  TALLY_BUILDER_SID:'S-1-5-21-2839165489-1832998833-1574237796-1004',
  TALLY_VERIFIER_SID:'S-1-5-21-2839165489-1832998833-1574237796-1003',
  TALLY_LOCAL_ISSUE_NUMBER:'264',
  TALLY_LOCAL_REPOSITORY_PATH:'C:/repo'};
const claimPins={TALLY_V2_AGENT_DISPATCH:'host-mounted',
  TALLY_V2_AGENT_CLAIM_SOURCE_SHA:sha,
  TALLY_V2_AGENT_CLAIM_DISPATCH_ID:'dispatch:264:e1:integration:host:20261119:2',
  TALLY_V2_AGENT_CLAIM_OPERATOR_ID:'zcode:i264:e1:integration:1',
  TALLY_V2_AGENT_CLAIM_OPERATOR_KIND:'other',
  TALLY_V2_AGENT_CLAIM_WRITER_ID:'tally-host:win32:i264:e1:integration:1',
  TALLY_V2_AGENT_CLAIM_WRITER_LOGIN:'kaicreator-mm',
  TALLY_V2_AGENT_CLAIM_COMMENT_ID:'6994840214',
  TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'C:/lease/registry-264-e1-claim.sqlite',
  TALLY_V2_AGENT_CLAIM_WRITESET_SHA256:'c'.repeat(64),
  TALLY_V2_AGENT_CLAIM_ATTEMPT_ID:'attempt:264:e1:1',
  TALLY_V2_AGENT_WORKTREE_PATH:'C:/tmp/single-worktree',
  TALLY_V2_AGENT_EVIDENCE_DIR:'C:/tmp/single-evidence'};

test('write-journey pin admission admits a complete trusted Host pin set',()=>{
  const settings=readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins});
  assert.equal(settings.claim.sourceSha,sha);
  assert.equal(settings.claim.dispatchId,
    'dispatch:264:e1:integration:host:20261119:2');
  assert.equal(settings.claim.commentId,6994840214);
  assert.equal(settings.claim.timeoutMs,10*60*1000);
  assert.equal(settings.worktreePath,'C:/tmp/single-worktree');
});
test('write-journey pin admission refuses every missing or malformed pin',()=>{
  assert.throws(()=>readInstalledWriteJourneyEnvironment({}),
    /AGENT_DISPATCH_NOT_HOST_MOUNTED/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...claimPins}),/AGENT_CONSUMER_PINS_MISSING:TALLY_GITHUB_READ_TOKEN/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,TALLY_V2_AGENT_CLAIM_SOURCE_SHA:'nothex'}),
    /AGENT_CLAIM_SOURCE_SHA_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,TALLY_V2_AGENT_CLAIM_DISPATCH_ID:'bad id'}),
    /AGENT_CLAIM_DISPATCH_ID_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,TALLY_V2_AGENT_CLAIM_OPERATOR_KIND:'agent'}),
    /AGENT_CLAIM_OPERATOR_KIND_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,TALLY_V2_AGENT_CLAIM_WRITESET_SHA256:'c'.repeat(63)}),
    /AGENT_CLAIM_WRITESET_SHA256_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,TALLY_V2_AGENT_WORKTREE_PATH:'relative/path'}),
    /AGENT_WORKTREE_PATH_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,TALLY_LOCAL_ISSUE_NUMBER:'zero'}),
    /AGENT_LOCAL_ISSUE_NUMBER_INVALID/);
});

test('write-journey runner: start contract with an injected spawn',async()=>{
  const settings=readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins});
  let captured=null;
  const fakeSpawn=async(script,env)=>{
    captured={script,env};
    // startedAt is the spawn instant; the durable result is written DURING
    // the process lifetime (mtime >= startedAt), like the real consumer.
    const startedAt=Date.now();
    const {mkdir,writeFile}=await import('node:fs/promises');
    await mkdir(env.TALLY_LOCAL_EVIDENCE_DIR,{recursive:true});
    await writeFile(join(env.TALLY_LOCAL_EVIDENCE_DIR,'e2e-result.json'),
      JSON.stringify({status:'PASS',exitCode:0,realAgentEffect:true,
        hiddenVerifier:'HIDDEN_NOT_QUALIFIED'})+'\n');
    return {code:0,signal:null,startedAt,terminatedAt:Date.now()};
  };
  const runner=createInstalledWriteJourneyRunner({settings,
    env:{...consumerPins,...claimPins},spawnE2e:fakeSpawn});
  const review=await runner.run('review-repair');
  assert.equal(review.status,'UNKNOWN');
  assert.equal(review.reason,'REVIEW_CONSUMER_NOT_MOUNTED');
  assert.equal(review.evidenceDir,undefined);
  assert.equal(review.exitCode,null);
  const pass=await runner.run('start');
  assert.equal(pass.status,'PASS');
  assert.match(captured.script,
    /experiments[/\\]v2[/\\]selected-business-build[/\\]local-codex-windows-e2e\.mjs$/);
  assert.equal(captured.env.TALLY_CLAIM_SOURCE_SHA,sha);
  assert.equal(captured.env.TALLY_CLAIM_DISPATCH_ID,
    'dispatch:264:e1:integration:host:20261119:2');
  assert.equal(captured.env.TALLY_CLAIM_COMMENT_ID,'6994840214');
  assert.equal(captured.env.TALLY_LOCAL_ATTEMPT_ID,'attempt:264:e1:1');
  assert.equal(captured.env.TALLY_LOCAL_WORKTREE_PATH,'C:/tmp/single-worktree');
  assert.equal(captured.env.TALLY_LOCAL_EVIDENCE_DIR,'C:/tmp/single-evidence');
  assert.equal(captured.env.TALLY_CODEX_CLI_PATH,'C:/codex/codex.exe');
  const emptySettings=readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,
      TALLY_V2_AGENT_EVIDENCE_DIR:'C:/tmp/empty-evidence'});
  const lost=createInstalledWriteJourneyRunner({settings:emptySettings,
    env:{...consumerPins,...claimPins},
    spawnE2e:async()=>({code:0,signal:null,startedAt:Date.now(),
      terminatedAt:Date.now()})});
  const lostResult=await lost.run('start');
  assert.equal(lostResult.status,'UNKNOWN');
  assert.equal(lostResult.reason,'ACK_LOST');
  const failed=createInstalledWriteJourneyRunner({settings,
    env:{...consumerPins,...claimPins},
    spawnE2e:async()=>({code:1,signal:null,startedAt:Date.now(),
      terminatedAt:Date.now()})});
  const failResult=await failed.run('start');
  assert.equal(failResult.status,'FAIL');
  const timed=createInstalledWriteJourneyRunner({settings,
    env:{...consumerPins,...claimPins},
    spawnE2e:async()=>({code:null,signal:null,timeout:true,startedAt:Date.now(),
      terminatedAt:Date.now()})});
  const timeoutResult=await timed.run('start');
  assert.equal(timeoutResult.status,'FAIL');
  assert.equal(timeoutResult.timeout,true);
});

test('stale claim source SHA refuses START closed at boot',async()=>{
  await assert.rejects(()=>startUiReadOnlyServer({env:{...validEnv,
    ...consumerPins,...claimPins,TALLY_V2_AGENT_CLAIM_SOURCE_SHA:otherSha}}),
    /AGENT_CLAIM_SOURCE_SHA_STALE/);
});

test('agent-attempts projection renders only real executions',async()=>{
  // Before any action: connected with an honestly empty roster. The run
  // seam mirrors the exact R90 serve.mjs fail-closed stub.
  const app=createAlmServer(config,{run:async()=>({status:'FAIL',
    reason:'UI_READ_ONLY_NO_AGENT_DISPATCH',exitCode:1,result:null})});
  const {origin,actionKey}=await app.listen();
  try{
    const post=async payload=>fetch(origin+'/api/actions',{method:'POST',
      headers:{'content-type':'application/json',origin,
        'x-tally-action-key':actionKey},
      body:JSON.stringify({...payload,sourceSha:sha})});
    let data=await (await fetch(origin+'/api/v2/agent-attempts')).json();
    assert.equal(data.connected,true);
    assert.deepEqual(data.attempts,[]);
    // The R90 fail-closed stub run must NOT render as a failed agent
    // attempt: nothing executed, so the projection stays honestly empty.
    // (Non-PASS action results answer 502 per the route contract.)
    const stub=await post({kind:'start'});
    assert.equal(stub.status,502);
    const stubBody=await stub.json();
    assert.equal(stubBody.status,'FAIL');
    assert.equal(stubBody.reason,'UI_READ_ONLY_NO_AGENT_DISPATCH');
    data=await (await fetch(origin+'/api/v2/agent-attempts')).json();
    assert.deepEqual(data.attempts,[]);
    // The R90 stub covers every action kind: no process executor exists in
    // this mount, so the review action stays fail-closed FAIL as well — and
    // equally invisible as an attempt row.
    const review=await post({kind:'review-repair'});
    const reviewBody=await review.json();
    assert.equal(reviewBody.status,'FAIL');
    assert.equal(reviewBody.reason,'UI_READ_ONLY_NO_AGENT_DISPATCH');
    data=await (await fetch(origin+'/api/v2/agent-attempts')).json();
    assert.deepEqual(data.attempts,[]);
  }finally{await app.close();}
  // A real observed execution (exit 0 + evidence + PASS summary) renders
  // exactly one bounded attempt row with the dispatched issue and UNKNOWN
  // telemetry preserved (null, never zero).
  const previousIssue=process.env.TALLY_LOCAL_ISSUE_NUMBER;
  process.env.TALLY_LOCAL_ISSUE_NUMBER='264';
  const realRun=async kind=>({
    status:'PASS',exitCode:0,signal:null,timeout:false,
    observedAt:'2026-11-19T08:00:00.000Z',
    evidenceDir:`C:/tmp/evidence/${kind}`,
    result:{status:'PASS',realAgentEffect:true,
      hiddenVerifier:'HIDDEN_NOT_QUALIFIED',
      installedAcceptance:{state:'NOT_RUN'},claimReceipts:[]}});
  const app2=createAlmServer(config,{run:realRun});
  const {origin:origin2,actionKey:key2}=await app2.listen();
  try{
    const response=await fetch(origin2+'/api/actions',{method:'POST',
      headers:{'content-type':'application/json',origin:origin2,
        'x-tally-action-key':key2},
      body:JSON.stringify({kind:'start',sourceSha:sha})});
    assert.equal(response.status,200);
    const data=await (await fetch(
      origin2+'/api/v2/agent-attempts')).json();
    assert.equal(data.connected,true);
    assert.equal(data.attempts.length,1);
    const row=data.attempts[0];
    assert.equal(row.state,'EXITED_OK');
    assert.equal(row.role,'BUILDER');
    assert.equal(row.provider,'codex');
    assert.deepEqual(row.taskIds,['264']);
    assert.equal(row.attemptId,'attempt:start:2026-11-19T08:00:00.000Z');
    assert.equal(row.durationMs,null);
    assert.equal(row.tokensIn,null);
    assert.equal(row.costUsd,null);
  }finally{
    if(previousIssue===undefined)delete process.env.TALLY_LOCAL_ISSUE_NUMBER;
    else process.env.TALLY_LOCAL_ISSUE_NUMBER=previousIssue;
    await app2.close();}
  // A real spawn that died in 'error' (no exit code, no timeout, but a real
  // launch with an evidence directory) IS an attempt: visible, EXITED_FAIL.
  const errorRun=async kind=>({status:'FAIL',exitCode:null,signal:null,
    timeout:false,observedAt:'2026-11-19T08:05:00.000Z',
    evidenceDir:`C:/tmp/evidence/${kind}`,result:null});
  const app3=createAlmServer(config,{run:errorRun});
  const {origin:origin3,actionKey:key3}=await app3.listen();
  try{
    await fetch(origin3+'/api/actions',{method:'POST',
      headers:{'content-type':'application/json',origin:origin3,
        'x-tally-action-key':key3},
      body:JSON.stringify({kind:'start',sourceSha:sha})});
    const data=await (await fetch(
      origin3+'/api/v2/agent-attempts')).json();
    assert.equal(data.attempts.length,1);
    assert.equal(data.attempts[0].state,'EXITED_FAIL');
  }finally{await app3.close();}
});

test('release evidence passthrough uses the mounted provider and keeps the advisory default',async()=>{
  const readiness={versionId:'version/v2',
    gates:[{name:'woodpeckerCI',status:'PASS',owner:'CI'},
      {name:'independentRq',status:'UNKNOWN',owner:'独立验证者'}],
    permits:null,rqState:'UNKNOWN',ownerGrantState:'UNKNOWN'};
  const app=createAlmServer(config,{v2ReadModelHost:{
    release:async scope=>{assert.deepEqual(scope,{versionId:'version/v2'});
      return {connected:true,versionId:'version/v2',readiness};}}});
  const {origin}=await app.listen();
  try{
    const mounted=await (await fetch(
      origin+'/api/v2/release?versionId=version%2Fv2')).json();
    assert.equal(mounted.connected,true);
    assert.equal(mounted.readiness.gates[0].status,'PASS');
    assert.equal(mounted.readiness.permits,null);
  }finally{await app.close();}
  // Unmounted facade: the documented advisory projection stays (connected
  // UNKNOWN×2) — no fabricated gates, no fabricated permits.
  const plain=createAlmServer(config);
  const {origin:plainOrigin}=await plain.listen();
  try{
    const advisory=await (await fetch(
      plainOrigin+'/api/v2/release?versionId=version%2Fv2')).json();
    assert.equal(advisory.connected,true);
    assert.deepEqual(advisory.readiness.gates.map(g=>g.status),
      ['UNKNOWN','UNKNOWN']);
    assert.equal(advisory.readiness.permits,null);
    // A mounted provider that throws degrades honestly, not a 500.
    const scoped=createAlmServer(config,{v2ReadModelHost:{
      release:async()=>{throw new Error('PROVIDER_DOWN');}}});
    const {origin:scopedOrigin}=await scoped.listen();
    try{
      const degraded=await (await fetch(
        scopedOrigin+'/api/v2/release?versionId=version%2Fv2')).json();
      assert.equal(degraded.connected,false);
      assert.equal(degraded.model,'release');
    }finally{await scoped.close();}
  }finally{await plain.close();}
});

test('release action submission reaches the real fail-closed admission',async()=>{
  // Without a mounted delivery seam the route must still re-read native
  // evidence first and deny with the exact admission reason (no seam, no
  // fabricated permission, no effect).
  const releaseHost={readReleaseSnapshot:async({sourceSha})=>({
    sourceSha,targetSha:sourceSha,
    readiness:{status:'UNKNOWN',releasePass:false,ciStatus:'NOT_RUN',
      ciCoverage:'NOT_CONFIGURED'}})};
  const app=createAlmServer(config,{releaseHost});
  const {origin,actionKey}=await app.listen();
  try{
    const snapshot=await (await fetch(
      origin+'/api/release/snapshot')).json();
    assert.equal(snapshot.snapshot.status,'OBSERVED');
    const post=async body=>fetch(origin+'/api/release/actions',
      {method:'POST',headers:{'content-type':'application/json',origin,
        'x-tally-action-key':actionKey},body:JSON.stringify(body)});
    const denied=await post({kind:'publish',sourceSha:sha});
    assert.equal(denied.status,200);
    const body=await denied.json();
    assert.equal(body.status,'DENIED');
    assert.equal(body.reason,'INDEPENDENT_HOST_ADMISSION_NOT_CONNECTED');
    // A foreign source SHA is stale at the admission boundary.
    const stale=await post({kind:'merge',sourceSha:otherSha});
    assert.equal(stale.status,409);
  }finally{await app.close();}
});
