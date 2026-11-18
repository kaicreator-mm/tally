/** #264 E1 — installed entrypoint regression (R90 minimal repair).
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
 *    roster without frozen host, scope guard) that repeats cleanly.
 *
 * No GitHub call can start here: the roster host denies immediately without
 * a mounted frozen reader and the catalog model is never read in this file.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
// Static import: evaluating this module resolves every serve.mjs static
// import target — the dedicated test below documents that contract.
import {readUiHostEnvironment,startUiReadOnlyServer} from './serve.mjs';

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
