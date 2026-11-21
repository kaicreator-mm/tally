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

test('managed repository path and Tally driver checkout are independent',async()=>{
  const env={...consumerPins,...claimPins,
    TALLY_V2_AGENT_SCRIPT_ROOT:'C:/tally-driver',
    TALLY_V2_AGENT_SOURCE_REPOSITORY_PATH:'C:/project-source'};
  const settings=readInstalledWriteJourneyEnvironment(env);
  assert.equal(settings.scriptRoot,'C:/tally-driver');
  assert.equal(settings.sourceRepositoryPath,'C:/project-source');
  let script,sourcePath;
  const runner=createInstalledWriteJourneyRunner({settings,env,
    spawnE2e:async (path,runnerEnv)=>{
      script=path;sourcePath=runnerEnv.TALLY_LOCAL_REPOSITORY_PATH;
      return {code:1,signal:null,startedAt:Date.now(),
        terminatedAt:Date.now()};
    }});
  await runner.run('start');
  assert.ok(script.startsWith('C:/tally-driver'));
  assert.equal(sourcePath,'C:/project-source');
  assert.throws(()=>readInstalledWriteJourneyEnvironment({...env,
    TALLY_V2_AGENT_SOURCE_REPOSITORY_PATH:'relative/path'}),
    /AGENT_SCRIPT_OR_SOURCE_REPOSITORY_PATH_INVALID/);
});
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

test('alternate GitHub project cannot run the historical sealed Builder',()=>{
  const selected={...consumerPins,...claimPins,TALLY_V2_REPOSITORY:'org/new',
    TALLY_V2_VERSION_REF:'main',TALLY_LOCAL_ISSUE_NUMBER:'27'};
  assert.throws(()=>readInstalledWriteJourneyEnvironment(selected),
    /AGENT_BUILDER_TARGET_NOT_SEALED/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment({
    ...selected,T06M_TARGET_REPOSITORY:'org/new'}),
    /AGENT_BUILDER_TARGET_PARTIAL/);
  const targetPins={T06M_TARGET_REPOSITORY:'org/new',
    T06M_TARGET_VERSION_REF:'main',T06M_TARGET_ISSUE_NUMBER:'27',
    T06M_TARGET_CANDIDATE_PATH:'src/health.mjs'};
  assert.throws(()=>readInstalledWriteJourneyEnvironment({
    ...selected,...targetPins,T06M_TARGET_ISSUE_NUMBER:'28'}),
    /AGENT_BUILDER_TARGET_SCOPE_MISMATCH/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment({
    ...selected,...targetPins,T06M_TARGET_REPOSITORY:'org/foreign'}),
    /AGENT_BUILDER_TARGET_SCOPE_MISMATCH/);
  const result=readInstalledWriteJourneyEnvironment({...selected,...targetPins});
  assert.equal(result.claim.sourceSha,sha);
  assert.equal(result.worktreePath,'C:/tmp/single-worktree');
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

// ---------------------------------------------------------------------------
// #264 J05 — single-Builder ReviewDecision seam (review dispatch mount)
// ---------------------------------------------------------------------------
const reviewPins={TALLY_V2_REVIEW_DISPATCH:'host-mounted',
  TALLY_V2_REVIEW_DECISION:'ACCEPT',
  TALLY_V2_REVIEW_DECISION_ID:'decision:264:j5:1'};

test('review dispatch pins admit a valid mount and refuse malformed ones',()=>{
  const settings=readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,...reviewPins});
  assert.equal(settings.review.decision,'ACCEPT');
  assert.equal(settings.review.decisionId,'decision:264:j5:1');
  // Unmounted stays the honest default: no review pins, no review mount.
  const unmounted=readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins});
  assert.equal(unmounted.review,undefined);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,...reviewPins,
      TALLY_V2_REVIEW_DISPATCH:'browser'}),
    /REVIEW_DISPATCH_MODE_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,...reviewPins,
      TALLY_V2_REVIEW_DECISION:'SHIP_IT'}),
    /REVIEW_DECISION_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,...reviewPins,
      TALLY_V2_REVIEW_DECISION_ID:'bad id'}),
    /REVIEW_DECISION_ID_INVALID/);
});

test('write-journey runner: mounted review-repair spawns the real single-review consumer',async()=>{
  // Isolated evidence directory: node:test runs tests in one file
  // concurrently, and a shared dir races the durable-result readback
  // against another test's truncate window (empty read ⇒ UNKNOWN).
  const reviewEvidence='C:/tmp/review-evidence';
  const settings=readInstalledWriteJourneyEnvironment(
    {...consumerPins,...claimPins,...reviewPins,
      TALLY_V2_AGENT_EVIDENCE_DIR:reviewEvidence});
  let captured=null;
  const fakeSpawn=async(script,env)=>{
    captured={script,env};
    // Back-date the spawn instant slightly: some CI filesystems stamp file
    // mtimes from a coarser clock than Date.now(), so a just-written durable
    // result can otherwise land "before" its own spawn (real consumers write
    // seconds later and never hit this).
    const startedAt=Date.now()-5;
    const {mkdir,writeFile}=await import('node:fs/promises');
    await mkdir(env.TALLY_LOCAL_EVIDENCE_DIR,{recursive:true});
    await writeFile(join(env.TALLY_LOCAL_EVIDENCE_DIR,'e2e-result.json'),
      JSON.stringify({status:'PASS',exitCode:0,lane:'t264-single-review',
        realReviewEffect:true,decisionRequested:'ACCEPT',
        result:{result:'REVIEW_DECISION_RECORDED',
          reason:'REVIEW_DECISION_CAS_READBACK_VERIFIED',
          decision:'ACCEPT',decisionKey:'tally:v2:single-review-decision:'+'d'.repeat(64),
          taskOutcomeStatus:'QUALIFIED_TASK_OUTCOME_PUBLISHED',taskPass:true}})+'\n');
    return {code:0,signal:null,startedAt,terminatedAt:Date.now()};
  };
  const runner=createInstalledWriteJourneyRunner({settings,
    env:{...consumerPins,...claimPins,...reviewPins,
      TALLY_V2_AGENT_EVIDENCE_DIR:reviewEvidence},spawnE2e:fakeSpawn});
  const review=await runner.run('review-repair');
  assert.equal(review.status,'PASS');
  assert.equal(review.evidenceDir,reviewEvidence);
  assert.match(captured.script,
    /experiments[/\\]v2[/\\]selected-business-build[/\\]single-codex-review-e2e\.mjs$/);
  assert.equal(captured.env.TALLY_REVIEW_DECISION,'ACCEPT');
  assert.equal(captured.env.TALLY_REVIEW_DECISION_ID,'decision:264:j5:1');
  assert.equal(captured.env.TALLY_CLAIM_SOURCE_SHA,sha);
  assert.equal(captured.env.TALLY_LOCAL_EVIDENCE_DIR,reviewEvidence);
  // The builder lane stays wired to the J04 consumer; a mounted review seam
  // must never redirect the start kind.
  const start=await runner.run('start');
  assert.equal(start.status,'PASS');
  assert.match(captured.script,
    /experiments[/\\]v2[/\\]selected-business-build[/\\]local-codex-windows-e2e\.mjs$/);
  // A lost durable review result on exit 0 stays UNKNOWN, never PASS (its
  // own never-written evidence directory: the mtime filter alone must not be
  // the only guard on platforms with coarse timestamp granularity).
  const lostEvidence='C:/tmp/review-lost-evidence';
  const lost=createInstalledWriteJourneyRunner({
    settings:readInstalledWriteJourneyEnvironment(
      {...consumerPins,...claimPins,...reviewPins,
        TALLY_V2_AGENT_EVIDENCE_DIR:lostEvidence}),
    env:{...consumerPins,...claimPins,...reviewPins,
      TALLY_V2_AGENT_EVIDENCE_DIR:lostEvidence},
    spawnE2e:async()=>({code:0,signal:null,startedAt:Date.now(),
      terminatedAt:Date.now()})});
  const lostResult=await lost.run('review-repair');
  assert.equal(lostResult.status,'UNKNOWN');
  assert.equal(lostResult.reason,'ACK_LOST');
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

// ---------------------------------------------------------------------------
// #264 J02 — dual-epoch read seam (S0 frozen epoch + live S1) environment
// admission and route truth (zero network, zero credentials).
// ---------------------------------------------------------------------------
const frozenEnv={...validEnv,TALLY_V2_FROZEN_DAG_HOST:'read-only',
  TALLY_VERSION_DAG_REGISTRY_PATH:'/tmp/registry.sqlite',
  TALLY_VERSION_DAG_REVIEW_ISSUE:'392',
  TALLY_VERSION_DAG_REVIEW_COMMENT_ID:'6976912326',
  TALLY_VERSION_DAG_REVIEW_LOGIN:'kaicreator-mm',
  TALLY_VERSION_DAG_APPROVAL_ISSUE:'392',
  TALLY_VERSION_DAG_APPROVAL_COMMENT_ID:'6574693761',
  TALLY_VERSION_DAG_APPROVAL_LOGIN:'kaicreator-mm'};
const j02Amendments=JSON.stringify([
  {kind:'NATIVE_EDGE_REMOVED',from:260,to:264,
    evidenceRef:'https://github.com/kaicreator-mm/tally/pull/465'},
  {kind:'NATIVE_EDGE_REMOVED',from:261,to:264,
    evidenceRef:'https://github.com/kaicreator-mm/tally/pull/465'},
  {kind:'NATIVE_EDGE_REMOVED',from:273,to:264,
    evidenceRef:'https://github.com/kaicreator-mm/tally/pull/465'}]);
test('dual-epoch pins require frozen DAG pins and validate strictly',()=>{
  const settings=readUiHostEnvironment({...frozenEnv,
    TALLY_V2_FREEZE_BASE_SHA:sha});
  assert.equal(settings.freezeBaseSha,sha);
  assert.equal(settings.edgeAmendments,null);
  const withLedger=readUiHostEnvironment({...frozenEnv,
    TALLY_V2_FREEZE_BASE_SHA:sha,
    TALLY_V2_EDGE_AMENDMENTS_JSON:j02Amendments});
  assert.equal(withLedger.edgeAmendments.length,3);
  assert.equal(withLedger.edgeAmendments[0].kind,'NATIVE_EDGE_REMOVED');
  // A malformed freeze base, a malformed ledger, amendments without a
  // freeze base, and dual-epoch pins without frozen DAG pins all refuse
  // START closed.
  assert.throws(()=>readUiHostEnvironment({...frozenEnv,
    TALLY_V2_FREEZE_BASE_SHA:'not-a-sha'}),/INVALID_HOST_FREEZE_BASE_SHA/);
  assert.throws(()=>readUiHostEnvironment({...frozenEnv,
    TALLY_V2_FREEZE_BASE_SHA:sha,
    TALLY_V2_EDGE_AMENDMENTS_JSON:'not-json'}),
    /INVALID_HOST_EDGE_AMENDMENTS_LEDGER/);
  assert.throws(()=>readUiHostEnvironment({...frozenEnv,
    TALLY_V2_FREEZE_BASE_SHA:sha,
    TALLY_V2_EDGE_AMENDMENTS_JSON:'[]'}),
    /INVALID_HOST_EDGE_AMENDMENTS_LEDGER/);
  assert.throws(()=>readUiHostEnvironment({...validEnv,
    TALLY_V2_EDGE_AMENDMENTS_JSON:j02Amendments}),
    /DUAL_EPOCH_REQUIRES_FROZEN_DAG_PINS/);
  assert.throws(()=>readUiHostEnvironment({...frozenEnv,
    TALLY_V2_EDGE_AMENDMENTS_JSON:j02Amendments}),
    /EDGE_AMENDMENTS_REQUIRE_FREEZE_BASE_SHA/);
});
test('dual-epoch route passes the mounted projection verbatim and stays unmounted-honest',async()=>{
  const projection={connected:true,
    frozenSource:{repository:'kaicreator-mm/tally',sourceSha:sha,
      currentness:'FROZEN_HISTORICAL',readback:'VERIFIED'},
    currentSource:{repository:'kaicreator-mm/tally',sourceSha:otherSha,
      currentness:'CURRENT',readback:'VERIFIED'},
    epoch:{freezeBaseSha:sha,observedVersionSha:otherSha,
      ancestry:'S0_ANCESTOR_OF_S1'},
    currentnessStatus:null,currentness:'STALE',
    currentReason:'CURRENT_SOURCE_COMPATIBILITY_UNWITNESSED',
    rosterCoverage:'HISTORICAL_VERIFIED_CURRENT_STALE',
    frozenBlobSha:otherSha,taskPackRef:'https://github.com/kaicreator-mm/tally/blob/'+
      sha+'/pack.json',
    taskCount:51,nativeEdgeCount:29,frozenEdgeCount:29,
    observedNativeEdgeCount:26,
    edgeAmendmentLedger:{disposition:'NATIVE_EDGE_REMOVED',declared:3,
      applied:[{from:260,to:264,
        evidenceRef:'https://github.com/kaicreator-mm/tally/pull/465'}]},
    compatibilityWitness:{present:false,valid:null},
    authorized:false,admitted:false,taskPass:false,releasePass:false};
  const calls=[];
  const app=createAlmServer(config,{v2ReadModelHost:{
    async dualEpoch(scope){calls.push(scope);return projection;}}});
  const {origin}=await app.listen();
  try{
    const observed=await (await fetch(
      origin+'/api/v2/dual-epoch?versionId=version%2Fv2')).json();
    assert.equal(observed.connected,true);
    assert.equal(observed.currentnessStatus,null);
    assert.equal(observed.currentness,'STALE');
    assert.equal(observed.currentReason,
      'CURRENT_SOURCE_COMPATIBILITY_UNWITNESSED');
    assert.equal(observed.rosterCoverage,'HISTORICAL_VERIFIED_CURRENT_STALE');
    assert.equal(observed.observedNativeEdgeCount,26);
    assert.equal(observed.edgeAmendmentLedger.declared,3);
    assert.deepEqual(calls,[{versionId:'version/v2'}]);
  }finally{await app.close();}
  // Without any mounted Host model the route answers the honest
  // NOT_CONNECTED default (never a fabricated dual-epoch state).
  const bare=createAlmServer(config);
  const {origin:bareOrigin}=await bare.listen();
  try{
    const observed=await (await fetch(
      bareOrigin+'/api/v2/dual-epoch?versionId=version%2Fv2')).json();
    assert.equal(observed.connected,false);
    assert.equal(observed.reason,'NOT_CONNECTED');
    assert.equal(observed.model,'dual-epoch');
  }finally{await bare.close();}
});
test('installed server without a freeze base answers DUAL_EPOCH_NOT_CONFIGURED',async()=>{
  // Boot the real installed entrypoint WITHOUT frozen/dual-epoch pins: no
  // heavy transport is loaded and the dual-epoch model must deny honestly
  // (a single-epoch fallback dressed as dual-epoch would be a lie).
  const server=await startUiReadOnlyServer({env:{...validEnv}});
  try{
    const observed=await (await fetch(
      server.address.origin+'/api/v2/dual-epoch?versionId=version%2Fv2')).json();
    assert.equal(observed.connected,false);
    assert.equal(observed.reason,'DUAL_EPOCH_NOT_CONFIGURED');
    // Scope guard precedes the mount check in the serve composition: a
    // foreign version denies PROJECT_VERSION_SCOPE_MISMATCH even while the
    // dual-epoch host stays unmounted.
    const foreign=await (await fetch(
      server.address.origin+'/api/v2/dual-epoch?versionId=version%2Fv3')).json();
    assert.equal(foreign.connected,false);
    assert.equal(foreign.reason,'PROJECT_VERSION_SCOPE_MISMATCH');
  }finally{await server.close();}
});

// ---------------------------------------------------------------------------
// #264 J07 — real release eligibility projection + Release Host Consumer
// dispatch admission (zero credentials; fixture CAS rows only).
// ---------------------------------------------------------------------------
test('v2 release model renders the real 13-gate J07a advisory projection',async()=>{
  // Booted WITHOUT a QA status context: every gate is honestly NOT_RUN
  // (EVIDENCE_MISSING) and the projection can never claim PASS authority.
  const server=await startUiReadOnlyServer({env:{...validEnv}});
  try{
    const release=await (await fetch(
      server.address.origin+'/api/v2/release?versionId=version%2Fv2')).json();
    assert.equal(release.connected,true);
    const gates=release.readiness.gates;
    assert.equal(gates.length,13);
    assert.ok(gates.every(gate=>gate.status==='NOT_RUN'));
    assert.ok(gates.some(gate=>gate.name==='woodpeckerCI'));
    assert.ok(gates.some(gate=>gate.name==='releaseQualification'));
    assert.equal(release.readiness.permits,null);
    assert.equal(release.readiness.rqState,'UNKNOWN');
    assert.equal(release.readiness.ownerGrantState,'UNKNOWN');
    assert.equal(release.readiness.advisoryOnly,true);
    assert.equal(release.readiness.gateCounts.required,13);
    assert.equal(release.readiness.gateCounts.reportedPass,0);
    // #491: the controller pre-read now carries the LIVE target readback;
    // with no reachable GitHub target the whole controller snapshot is the
    // honest fail-closed UNKNOWN (it must never fabricate a target again).
    const snapshot=await (await fetch(
      server.address.origin+'/api/release/snapshot')).json();
    assert.equal(snapshot.snapshot.status,'UNKNOWN');
    assert.equal(snapshot.snapshot.releasePass,false);
    assert.equal(snapshot.snapshot.canMerge,false);
    assert.equal(snapshot.snapshot.canTag,false);
    assert.equal(snapshot.snapshot.canPublish,false);
  }finally{await server.close();}
});
test('release Host Consumer dispatch admits only pinned Host configurations',async()=>{
  const {createInstalledReleaseDispatch}=await import('./serve.mjs');
  const baseSettings={repository:'kaicreator-mm/tally',owner:'kaicreator-mm',
    repo:'tally',expectedSha:sha,token:'nonpublicfixture'};
  const {TALLY_GITHUB_READ_TOKEN:omitToken,...tokenlessEnv}=validEnv;
  assert.ok(!omitToken||omitToken.length>0);
  await assert.rejects(()=>createInstalledReleaseDispatch(
    {settings:{...baseSettings,token:undefined},
      env:{...tokenlessEnv,
        TALLY_V2_RELEASE_DISPATCH:'host-mounted',
        TALLY_RELEASE_SOURCE_SHA:sha}}),
    /RELEASE_HOST_TOKEN_REQUIRED/);
  await assert.rejects(()=>createInstalledReleaseDispatch(
    {settings:baseSettings,env:{...validEnv,
      TALLY_GITHUB_WRITE_TOKEN:'write-token-fixture',
      TALLY_RELEASE_SOURCE_SHA:otherSha}}),
    /RELEASE_SOURCE_SHA_STALE/);
  const dispatch=await createInstalledReleaseDispatch(
    {settings:baseSettings,env:{...validEnv,
      TALLY_GITHUB_WRITE_TOKEN:'write-token-fixture',
      TALLY_RELEASE_SOURCE_SHA:sha},
      openSqliteCasStore:async()=>{
        throw Error('NO_STORE_ACCESS_ON_PIN_DENIAL');}});
  // Pin-validation denials answer the verified seam shape without network:
  // unknown intent, stale source SHA, unpinned merge PR, unpinned tag source
  // and the deliberately unbound publish effect.
  const intent=await dispatch({intentType:'deliverQualifiedRelease',
    input:{kind:'merge',sourceSha:sha}});
  assert.equal(intent.status,'DENIED');
  assert.equal(intent.reason,'MERGE_PR_NOT_PINNED');
  assert.equal(intent.releasePass,false);
  assert.equal(intent.authorized,false);
  assert.equal(intent.taskPass,false);
  assert.equal(intent.releasePublished,false);
  const stale=await dispatch({intentType:'deliverQualifiedRelease',
    input:{kind:'merge',sourceSha:otherSha}});
  assert.equal(stale.status,'STALE');
  const foreign=await dispatch({intentType:'somethingElse',
    input:{kind:'merge',sourceSha:sha}});
  assert.equal(foreign.status,'DENIED');
  assert.equal(foreign.reason,'UNKNOWN_INTENT_OR_INPUT');
  const tag=await dispatch({intentType:'deliverQualifiedRelease',
    input:{kind:'tag',sourceSha:sha}});
  assert.equal(tag.status,'DENIED');
  assert.equal(tag.reason,'TAG_SOURCE_NOT_PINNED');
  const publish=await dispatch({intentType:'deliverQualifiedRelease',
    input:{kind:'publish',sourceSha:sha}});
  assert.equal(publish.status,'NOT_RUN');
  assert.equal(publish.reason,'PUBLISH_EFFECT_NOT_BOUND');
});
test('mounted release dispatch passes verifyReleaseSeamResult; forged shapes degrade',async()=>{
  const verified={status:'OBSERVED',reason:'HOST_GATES_OBSERVED',
    releasePass:false,authorized:false,taskPass:false,
    releasePublished:false,hostState:'HOST_GATES_OBSERVED'};
  const app=createAlmServer(config,{releaseSealedDispatch:async()=>{
    return verified;}});
  const {origin,actionKey}=await app.listen();
  try{
    const post=body=>fetch(origin+'/api/release/actions',
      {method:'POST',headers:{'content-type':'application/json',origin,
        'x-tally-action-key':actionKey},body:JSON.stringify(body)});
    const observed=await post({kind:'merge',sourceSha:sha});
    assert.equal(observed.status,200);
    const body=await observed.json();
    assert.equal(body.status,'OBSERVED');
    assert.equal(body.hostState,'HOST_GATES_OBSERVED');
    assert.equal(body.releasePass,false);
  }finally{await app.close();}
  // A forged authority shape (releasePass true) is refused by the same
  // verifier the serve mount feeds — the seam can never launder a PASS.
  const forged=createAlmServer(config,{releaseSealedDispatch:async()=>{
    return {status:'OBSERVED',reason:'FORGED',releasePass:true,
      authorized:false,taskPass:false,releasePublished:false};}});
  const forgedAddress=await forged.listen();
  try{
    const observed=await fetch(forgedAddress.origin+'/api/release/actions',
      {method:'POST',headers:{'content-type':'application/json',
        origin:forgedAddress.origin,
        'x-tally-action-key':forgedAddress.actionKey},
        body:JSON.stringify({kind:'merge',sourceSha:sha})});
    const body=await observed.json();
    assert.equal(body.status,'DENIED');
    assert.equal(body.reason,'SEAM_SHAPE_INVALID');
    assert.equal(body.releasePass,false);
  }finally{await forged.close();}
});

// ---------------------------------------------------------------------------
// #473 R107 — task-selected AgentProvider pin admission.
// ---------------------------------------------------------------------------
test('agent provider pin is Host-only, default codex, explicitly enabled',()=>{
  const base={...validEnv,TALLY_V2_AGENT_DISPATCH:'host-mounted',
    TALLY_V2_AGENT_CLAIM_SOURCE_SHA:sha,
    TALLY_V2_AGENT_CLAIM_DISPATCH_ID:'dispatch:264:t:test:1',
    TALLY_V2_AGENT_CLAIM_OPERATOR_ID:'zcode:t264:test:1',
    TALLY_V2_AGENT_CLAIM_OPERATOR_KIND:'other',
    TALLY_V2_AGENT_CLAIM_WRITER_ID:'tally-host:win32:t264:test:1',
    TALLY_V2_AGENT_CLAIM_WRITER_LOGIN:'kaicreator-mm',
    TALLY_V2_AGENT_CLAIM_COMMENT_ID:'1',
    TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
    TALLY_V2_AGENT_CLAIM_WRITESET_SHA256:'0'.repeat(64),
    TALLY_V2_AGENT_CLAIM_ATTEMPT_ID:'attempt:264:t:1',
    TALLY_V2_AGENT_WORKTREE_PATH:'/tmp/wt',
    TALLY_V2_AGENT_EVIDENCE_DIR:'/tmp/ev',
    TALLY_CODEX_CLI_PATH:'C:/c.exe',TALLY_CODEX_SANDBOX_PATH:'C:/c.exe',
    TALLY_BUILDER_PROFILE:'C:/p',TALLY_BUILDER_TEMP:'C:/t',
    TALLY_EMPTY_GIT_CONFIG:'C:/g',TALLY_OPERATOR_HOME:'C:/o',
    TALLY_HIDDEN_FIXTURE:'C:/h',TALLY_BUILDER_PRINCIPAL:'nilar\b',
    TALLY_BUILDER_SID:'S-1-5-21-1',TALLY_VERIFIER_SID:'S-1-5-21-2',
    TALLY_LOCAL_ISSUE_NUMBER:'264',TALLY_LOCAL_REPOSITORY_PATH:'C:/repo'};
  const settings=readInstalledWriteJourneyEnvironment(base);
  assert.equal(settings.provider,'codex','default is codex with no pin');
  const pinned=readInstalledWriteJourneyEnvironment({...base,
    TALLY_V2_AGENT_PROVIDER:'zcode',
    TALLY_V2_AGENT_PROVIDERS_ENABLED:'codex,zcode'});
  assert.equal(pinned.provider,'zcode');
  // A provider outside the enabled list refuses START closed.
  assert.throws(()=>readInstalledWriteJourneyEnvironment({...base,
    TALLY_V2_AGENT_PROVIDER:'grok'}),
    /AGENT_PROVIDER_PIN_INVALID/);
  assert.throws(()=>readInstalledWriteJourneyEnvironment({...base,
    TALLY_V2_AGENT_PROVIDER:'not a provider'}),
    /AGENT_PROVIDER_PIN_INVALID/);
});
test('non-codex provider start retains the selected sealed Host driver',async()=>{
  const {tmpdir}=await import('node:os');
  const evidenceDir=join(tmpdir(),
    'aph-journey-ev-'+Math.random().toString(36).slice(2));
  await (await import('node:fs/promises')).mkdir(evidenceDir,{recursive:true});
  const scripts=[];
  const spawnE2e=async(script,env,timeoutMs)=>{
    scripts.push(String(script).split(/[\\/]/).pop());
    return {code:0,signal:null,timeout:false,startedAt:1,terminatedAt:2};};
  const settings=readInstalledWriteJourneyEnvironment({
    ...validEnv,TALLY_V2_AGENT_DISPATCH:'host-mounted',
    TALLY_V2_AGENT_CLAIM_SOURCE_SHA:sha,
    TALLY_V2_AGENT_CLAIM_DISPATCH_ID:'dispatch:264:t:test:2',
    TALLY_V2_AGENT_CLAIM_OPERATOR_ID:'zcode:t264:test:2',
    TALLY_V2_AGENT_CLAIM_OPERATOR_KIND:'other',
    TALLY_V2_AGENT_CLAIM_WRITER_ID:'tally-host:win32:t264:test:2',
    TALLY_V2_AGENT_CLAIM_WRITER_LOGIN:'kaicreator-mm',
    TALLY_V2_AGENT_CLAIM_COMMENT_ID:'2',
    TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
    TALLY_V2_AGENT_CLAIM_WRITESET_SHA256:'0'.repeat(64),
    TALLY_V2_AGENT_CLAIM_ATTEMPT_ID:'attempt:264:t:2',
    TALLY_V2_AGENT_WORKTREE_PATH:'/tmp/wt',
    TALLY_V2_AGENT_EVIDENCE_DIR:'/tmp/ev',
    TALLY_V2_AGENT_PROVIDER:'zcode',
    TALLY_V2_AGENT_PROVIDERS_ENABLED:'codex,zcode',
    TALLY_CODEX_CLI_PATH:'C:/c.exe',TALLY_CODEX_SANDBOX_PATH:'C:/c.exe',
    TALLY_BUILDER_PROFILE:'C:/p',TALLY_BUILDER_TEMP:'C:/t',
    TALLY_EMPTY_GIT_CONFIG:'C:/g',TALLY_OPERATOR_HOME:'C:/o',
    TALLY_HIDDEN_FIXTURE:'C:/h',TALLY_BUILDER_PRINCIPAL:'nilar\b',
    TALLY_BUILDER_SID:'S-1-5-21-1',TALLY_VERIFIER_SID:'S-1-5-21-2',
    TALLY_LOCAL_ISSUE_NUMBER:'264',TALLY_LOCAL_REPOSITORY_PATH:'C:/repo',
    TALLY_V2_AGENT_EVIDENCE_DIR:evidenceDir});
  const runner=createInstalledWriteJourneyRunner({settings,
    env:{...validEnv,TALLY_LOCAL_REPOSITORY_PATH:'C:/repo',
      TALLY_V2_AGENT_PROVIDER:'zcode'},spawnE2e});
  const observed=await runner.run('start');
  assert.deepEqual(scripts,['local-codex-windows-e2e.mjs'],
    'a non-codex pin must retain the one selected sealed B/K/Host/C driver');
  assert.equal(observed.status,'UNKNOWN','no e2e-result.json was written');
});

test('new project Claim route requires exact Host and same-origin CSRF, no UI grant',async()=>{
  let calls=0;
  const claimHost={repository:'org/new',versionId:'main',sourceSha:sha,
    issueNumber:27,publish:async input=>{
      calls++;assert.deepEqual(input,{issueNumber:27,sourceSha:sha});
      return {status:'RECORDED',claimPublication:'VERIFIED_PENDING_KERNEL_ADMISSION',
        admitted:false,authorized:false};
    }};
  const app=createAlmServer(config,{claimHost});
  const {origin,actionKey}=await app.listen();
  try{
    const observed=await (await fetch(origin+'/api/v2/claim-host')).json();
    assert.equal(observed.connected,true);
    assert.equal(observed.issueNumber,27);
    assert.equal(observed.actionKey,actionKey);
    const post=async(input,headers={})=>fetch(origin+'/api/v2/claim-publish',{
      method:'POST',headers:{'content-type':'application/json',origin,
        'x-tally-action-key':actionKey,...headers},
      body:JSON.stringify(input)});
    assert.equal((await post({issueNumber:27,sourceSha:sha},
      {'x-tally-action-key':'wrong'})).status,403);
    assert.equal((await post({issueNumber:28,sourceSha:sha})).status,409);
    assert.equal(calls,0);
    const approved=await post({issueNumber:27,sourceSha:sha});
    assert.equal(approved.status,200);
    const result=await approved.json();
    assert.equal(result.status,'RECORDED');
    assert.equal(result.authorized,false);
    assert.equal(result.admitted,false);
    assert.equal(calls,1);
  }finally{await app.close();}
  const absent=createAlmServer(config,{});
  const {origin:missingOrigin,actionKey:missingKey}=await absent.listen();
  try{
    const data=await (await fetch(missingOrigin+'/api/v2/claim-host')).json();
    assert.equal(data.connected,false);
    const response=await fetch(missingOrigin+'/api/v2/claim-publish',{
      method:'POST',headers:{'content-type':'application/json',
        origin:missingOrigin,'x-tally-action-key':missingKey},
      body:JSON.stringify({issueNumber:27,sourceSha:sha})});
    assert.equal(response.status,503);
  }finally{await absent.close();}
});

test('selected FSW and Draft PR routes are Host-scoped, CSRF protected',async()=>{
  const calls=[];
  const writeSetHost={repository:'org/new',versionId:'main',sourceSha:sha,
    issueNumber:27,candidateRelativePath:'health.txt',
    freeze:async input=>{
      calls.push(['fsw',input]);return {status:'FROZEN_VERIFIED',
        writeSetSha256:'f'.repeat(64),authorized:false,admitted:false};
    }};
  const candidateHost={repository:'org/new',versionId:'main',sourceSha:sha,
    issueNumber:27,publish:async input=>{
      calls.push(['draft',input]);return {status:'DRAFT_PR_PUBLISHED_VERIFIED',
        pullNumber:5,pullRef:'https://github.com/org/new/pull/5',
        authorized:false,taskPass:false,releasePass:false};
    }};
  const app=createAlmServer(config,{writeSetHost,candidateHost});
  const {origin,actionKey}=await app.listen();
  try{
    const fsw=await (await fetch(origin+'/api/v2/writeset-host')).json();
    const draft=await (await fetch(origin+'/api/v2/candidate-host')).json();
    assert.equal(fsw.connected,true);assert.equal(fsw.issueNumber,27);
    assert.equal(draft.connected,true);assert.equal(draft.sourceSha,sha);
    const post=async(path,input,key=actionKey)=>fetch(origin+path,{
      method:'POST',headers:{origin,'content-type':'application/json',
        'x-tally-action-key':key},body:JSON.stringify(input)});
    const correct={issueNumber:27,sourceSha:sha};
    assert.equal((await post('/api/v2/writeset-freeze',correct,'wrong')).status,403);
    assert.equal((await post('/api/v2/candidate-publish',correct,'wrong')).status,403);
    assert.equal((await post('/api/v2/writeset-freeze',
      {issueNumber:28,sourceSha:sha})).status,409);
    assert.equal((await post('/api/v2/candidate-publish',
      {issueNumber:28,sourceSha:sha})).status,409);
    assert.equal(calls.length,0);
    const frozen=await (await post('/api/v2/writeset-freeze',correct)).json();
    assert.equal(frozen.status,'FROZEN_VERIFIED');
    const published=await (await post('/api/v2/candidate-publish',correct)).json();
    assert.equal(published.status,'DRAFT_PR_PUBLISHED_VERIFIED');
    assert.equal(published.taskPass,false);
    assert.deepEqual(calls,[['fsw',correct],['draft',correct]]);
  }finally{await app.close();}
});


test('Linux Agent Host is explicit and keeps the sealed target and physical runner',
  async()=>{
    const env={...consumerPins,...claimPins,
      TALLY_V2_AGENT_PLATFORM:'linux',TALLY_LINUX_BWRAP_PATH:'/usr/bin/bwrap',
      TALLY_V2_AGENT_PREBUILT_CONSUMER_DIR:'/tmp/tally-linux-selected-consumer',
      TALLY_BUILDER_PRINCIPAL:'linux-bwrap:65534',
      TALLY_BUILDER_SID:'linux-uid:65534',
      TALLY_VERIFIER_SID:'linux-uid:65533',
      TALLY_V2_REPOSITORY:'kaicreator-mm/tally',TALLY_V2_VERSION_REF:'version/v2',
      T06M_TARGET_REPOSITORY:'kaicreator-mm/tally',
      T06M_TARGET_VERSION_REF:'version/v2',
      T06M_TARGET_ISSUE_NUMBER:'264',
      T06M_TARGET_CANDIDATE_PATH:'experiments/v2/selected-business-build/LOCAL_CODEX_L3.md'};
    const settings=readInstalledWriteJourneyEnvironment(env);
    assert.equal(settings.platform,'linux');
    let called=null;
    const runner=createInstalledWriteJourneyRunner({settings,env,
      spawnE2e:async (script,runEnv)=>{
        called={script,runEnv};
        return {code:1,signal:null,startedAt:Date.now(),terminatedAt:Date.now()};
      }});
    const outcome=await runner.run('start');
    assert.equal(outcome.status,'FAIL');
    assert.match(called.script,/local-codex-linux-e2e\.mjs$/);
    assert.equal(called.runEnv.TALLY_V2_AGENT_PLATFORM,'linux');
    assert.equal(called.runEnv.TALLY_CLAIM_SOURCE_SHA,sha);
    assert.equal(called.runEnv.TALLY_LOCAL_WORKTREE_PATH,claimPins.TALLY_V2_AGENT_WORKTREE_PATH);
    const rework=await runner.run('review-repair');
    assert.equal(rework.status,'UNKNOWN');
    assert.equal(rework.reason,'REVIEW_CONSUMER_NOT_MOUNTED');
    assert.throws(()=>readInstalledWriteJourneyEnvironment({
      ...env,TALLY_LINUX_BWRAP_PATH:undefined}),/AGENT_CONSUMER_PINS_MISSING/);
    assert.throws(()=>readInstalledWriteJourneyEnvironment({
      ...env,TALLY_V2_AGENT_PREBUILT_CONSUMER_DIR:undefined}),
      /AGENT_CONSUMER_PINS_MISSING/);
    assert.throws(()=>readInstalledWriteJourneyEnvironment({
      ...env,T06M_TARGET_REPOSITORY:undefined,T06M_TARGET_VERSION_REF:undefined,
      T06M_TARGET_CANDIDATE_PATH:undefined,T06M_TARGET_ISSUE_NUMBER:undefined}),
      /AGENT_LINUX_SEALED_TARGET_REQUIRED/);
    assert.throws(()=>readInstalledWriteJourneyEnvironment({
      ...env,TALLY_BUILDER_PRINCIPAL:'root'}),/AGENT_LINUX_SANDBOX_PINS_INVALID/);
  });
