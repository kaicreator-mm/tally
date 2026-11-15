import test from 'node:test';
import assert from 'node:assert/strict';
import {stringify} from 'yaml';
import {readNativeHostClaimPreflightFromClient,
  readGitHubNativeHostClaimPreflight} from '../src/task-claim-host-preflight.mjs';
// #422 physical regression child dispatch: when re-executed directly with
// I422_R33_CHILD_SCENARIO set, this file runs one physical successor scenario
// against real shared SQLite files and exits before any test registration.
if(process.env.I422_R33_CHILD_SCENARIO){
  await runI422ChildScenario(process.env.I422_R33_CHILD_SCENARIO);
  process.exit(0);
}
const owner='kaicreator-mm',repo='tally',issueNumber=125,sha='a'.repeat(40);
const issue='https://github.com/kaicreator-mm/tally/issues/125';
const input={owner,repo,issueNumber,expectedSourceSha:sha};
const fence=String.fromCharCode(96).repeat(3);
const event={
  schema:'ads/github/v3',kind:'event',data:{
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',task:'#125',sha,
    actor_role:'builder',operator_kind:'codex',operator_id:'codex-1',
    dispatch_id:'dispatch:125:x',dispatch_state:'CLAIMED'}
};
const claim={id:800,body:'<!-- ads:github -->\n'+fence+'yaml\n'+
  stringify(event).trimEnd()+'\n'+fence,
  html_url:issue+'#issuecomment-800',
  created_at:'2026-10-28T10:00:00Z',
  updated_at:'2026-10-28T10:00:00Z',user:{login:'agent'}};
function client({state='state:ready',type='type:task',issueState='open',
  blocker=null,comments=[],drift=false,failComments=false,commitSha=sha,
  graphEdited=false,graphUnavailable=false}={}){
  let reads=0;
  return {rest:{
    issues:{
      async get(){reads++;return {data:{
        id:525,number:125,html_url:issue,
        state:issueState,
        updated_at:drift&&reads>=6?'2026-10-28T10:01:00Z':'2026-10-28T10:00:00Z',
        labels:[{name:type},{name:state}]
      }};},
      async listComments(){throw Error('pagination required');}
    },
    repos:{async getCommit(){return {data:{sha:commitSha}};}}
  },paginate:{async *iterator(route){
    if(typeof route==='string'){
      yield {data:blocker?[{id:99,number:99,
        html_url:'https://github.com/kaicreator-mm/tally/issues/99',state:blocker}]:[]};
    }else{
      if(failComments)throw Object.assign(Error('PRIVATE-TOKEN'),{status:403});
      yield {data:comments};
    }
  }},graphql:async(_query,{id})=>{
    if(graphUnavailable)throw new Error('GRAPHQL_UNAVAILABLE');
    const c=comments.find(x=>x.node_id===id);
    return {node:{__typename:'IssueComment',id,fullDatabaseId:String(c?.id),
      url:c?.html_url,body:c?.body,
      lastEditedAt:graphEdited?'2026-11-15T12:00:00Z':null,
      editor:graphEdited?{login:'scheduler-bot'}:null,
      issue:{number:issueNumber,repository:{nameWithOwner:owner+'/'+repo}}}};
  }};
}
const check=(c={},overrides={})=>readNativeHostClaimPreflightFromClient({
  ...input,...overrides,octokit:client(c)
});
const noPermit=o=>{
  for(const k of ['authorized','admitted','taskPass','releasePass'])
    assert.equal(o[k],false);
  assert.equal(o.claimStatus,'NOT_VERIFIED');
};
test('native ready/CLEAR/no-Claim is a real Host source witness, NOT writer/K permit',async()=>{
  const x=await check();
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.sourceQualified,true);
  assert.equal(x.sourceKind,'NATIVE_GITHUB_READ_ONLY_PREFLIGHT');
  assert.equal(x.nativePreflightReady,true);
  assert.equal(x.dispatchSourceBound,false);
  assert.equal(x.publisherSourceBound,false);
  noPermit(x);
});
test('blocked dependency, stale state, non-task and closed Issue all deny',async()=>{
  for(const input of [
    {blocker:'open'}, {state:'state:implementing'},
    {type:'type:planning'}, {issueState:'closed'}
  ]){
    const x=await check(input);
    assert.equal(x.nativePreflightReady,false);noPermit(x);
  }
});
test('even shape-valid self-reported DISPATCH_CLAIMED denies clean preflight',async()=>{
  const x=await check({comments:[claim]});
  assert.equal(x.reason,'NATIVE_CLAIM_ABSENCE_UNQUALIFIED');noPermit(x);
});
test('GitHub stale SHA, drift, private 403 and spoofed booleans deny',async()=>{
  for(const facts of [
    {commitSha:'b'.repeat(40)},{drift:true},{failComments:true}
  ]){
    const x=await check(facts);
    assert.equal(x.reason,'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');
    assert.equal(JSON.stringify(x).includes('PRIVATE-TOKEN'),false);noPermit(x);
  }
  const injected=await check({},{
    claimAccepted:true,sourceQualified:true,authorized:true,
    publisherSourceBound:true,dispatchSourceBound:true,
    dependenciesComplete:true
  });
  assert.equal(injected.authorized,false);
  assert.equal(injected.publisherSourceBound,false);
});
test('missing read token, unsupported source SHA and limit fail closed',async()=>{
  const token=await readGitHubNativeHostClaimPreflight({...input,token:null});
  assert.equal(token.reason,'GITHUB_PROVIDER_TOKEN_UNAVAILABLE');noPermit(token);
  assert.equal((await check({}, {limit:1001})).reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');
  assert.equal((await check({}, {expectedSourceSha:'bad'})).reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');
});

// ---- #412 dual-role admission regression (FROZEN_DAG_WRITESET 9f31a645...) ----
const keyFor=(role)=>`kaicreator-mm/tally#${issueNumber}:${role}:__default__`;
const claimEvent=(changes={})=>({
  schema:'ads/github/v3',kind:'event',data:{
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',task:`#${issueNumber}`,
    issue:`#${issueNumber}`,sha,dispatch_state:'CLAIMED',
    admission_generation:1,...changes}
});
const claimComment=(id,changes={})=>({id,
  body:'<!-- ads:github -->\n'+fence+'yaml\n'+stringify(claimEvent(changes.data??{})).trimEnd()+'\n'+fence,
  html_url:issue+'#issuecomment-'+id,
  created_at:'2026-11-14T18:09:38Z',updated_at:'2026-11-14T18:09:38Z',
  user:{login:'kaicreator-mm'},...changes});
const builderClaim=claimComment(801,{data:{
  actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
  dispatch_id:'dispatch:125:builder',protected_claim_key:keyFor('builder')}});
const reviewerClaim=claimComment(802,{data:{
  actor_role:'reviewer',operator_kind:'claude-code',operator_id:'claude:reviewer1',
  dispatch_id:'dispatch:125:reviewer',protected_claim_key:keyFor('reviewer')}});

test('invalid declared actorRole fails request validation',async()=>{
  const x=await check({},{actorRole:'leader'});
  assert.equal(x.reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');noPermit(x);
});

test('fresh Task with declared role and zero candidates stays READY',async()=>{
  const x=await check({},{actorRole:'builder'});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.claimCandidates,0);
  assert.deepEqual(x.claimCandidateRoles,[]);
  assert.equal(x.claimRoleCandidates.length,0);
  noPermit(x);
});

test('existing builder Claim admits a distinct-role Reviewer with bounded projections',async()=>{
  const x=await check({comments:[builderClaim]},{actorRole:'reviewer'});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.nativePreflightReady,true);
  assert.equal(x.claimCandidates,1);
  assert.deepEqual(x.claimCandidateRoles,['builder']);
  assert.equal(x.claimRoleCandidates.length,1);
  const p=x.claimRoleCandidates[0];
  assert.equal(p.commentId,801);
  assert.equal(p.dispatchId,'dispatch:125:builder');
  assert.equal(p.actorRole,'builder');
  assert.equal(p.operatorId,'codex:builder1');
  assert.equal(p.protectedClaimKey,keyFor('builder'));
  assert.equal(p.claimGeneration,1);
  assert.equal(p.sourceSha,sha);
  assert.equal(p.transportActor,'kaicreator-mm');
  assert.equal(x.claimStatus,'NOT_VERIFIED');noPermit(x);
});

test('existing Reviewer Claim admits a distinct-role Builder (symmetric)',async()=>{
  const x=await check({comments:[reviewerClaim]},{actorRole:'builder'});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.deepEqual(x.claimCandidateRoles,['reviewer']);
  noPermit(x);
});

test('same-role duplicate candidate denies the requested role',async()=>{
  const x=await check({comments:[builderClaim]},{actorRole:'builder'});
  assert.equal(x.reason,'NATIVE_CLAIM_ROLE_DUPLICATE_CANDIDATE');noPermit(x);
});

test('two candidates sharing one protectedClaimKey deny as claim overwrite',async()=>{
  const second=claimComment(803,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder2',
    dispatch_id:'dispatch:125:builder2',protected_claim_key:keyFor('builder')}});
  const x=await check({comments:[builderClaim,second]},{actorRole:'reviewer'});
  assert.equal(x.reason,'NATIVE_CLAIM_ROLE_DUPLICATE_CANDIDATE');noPermit(x);
});

test('candidates without canonical key or generation 1 deny unqualified',async()=>{
  const noKey=claimComment(804,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
    dispatch_id:'dispatch:125:nokey'}});
  const x=await check({comments:[noKey]},{actorRole:'reviewer'});
  assert.equal(x.reason,'NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');noPermit(x);
  const wrongKey=claimComment(805,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
    dispatch_id:'dispatch:125:wrongkey',protected_claim_key:keyFor('reviewer')}});
  const y=await check({comments:[wrongKey]},{actorRole:'reviewer'});
  assert.equal(y.reason,'NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');noPermit(y);
  const gen2=claimComment(806,{data:{
    actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder1',
    dispatch_id:'dispatch:125:gen2',protected_claim_key:keyFor('builder'),
    admission_generation:2}});
  const z=await check({comments:[gen2]},{actorRole:'reviewer'});
  assert.equal(z.reason,'NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');noPermit(z);
  const badRole=claimComment(807,{data:{
    actor_role:'leader',operator_kind:'codex',operator_id:'codex:leader',
    dispatch_id:'dispatch:125:leader',protected_claim_key:keyFor('leader')}});
  const w=await check({comments:[badRole]},{actorRole:'reviewer'});
  // A non-ADS role never validates against the pinned event schema, so the
  // canonical comment reader fails closed before role classification runs.
  assert.equal(w.reason,'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');noPermit(w);
});

test('candidates present without a declared role keep the legacy denial reason',async()=>{
  const x=await check({comments:[builderClaim]});
  assert.equal(x.reason,'NATIVE_CLAIM_ABSENCE_UNQUALIFIED');noPermit(x);
});

// ---- #422 claim-succession regression (FROZEN_DAG_WRITESET b088685b...) ----
// A lawfully terminalized historical Claim (exact unedited DONE + independent
// TERMINAL_VERIFIED CAS readback) filters out of fresh admission and does not
// count as an active incompatible role; every non-terminal, divergent,
// missing or unverified CAS surface stays fail-closed. All GitHub surfaces in
// these fixtures are surrogates; no real comment or registry is touched.
const histSha125='b'.repeat(40);
const staleBuilder=claimComment(801,{data:{
  actor_role:'builder',operator_kind:'codex',operator_id:'codex:builder0',
  dispatch_id:'dispatch:125:hist',protected_claim_key:keyFor('builder'),
  sha:histSha125}});
const doneData=(claimId,claimSha,changes={})=>({schema:'ads/github/v3',
  kind:'event',data:{schema:'ai-dev/event-v2',event:'DISPATCH_STATE_CHANGED',
    task:`#${issueNumber}`,issue:`#${issueNumber}`,sha:claimSha,
    dispatch_id:'dispatch:125:hist',dispatch_state:'DONE',
    actor_role:'scheduler',operator_kind:'other',
    operator_id:'scheduler:i125:terminalization:1',
    protected_claim_key:keyFor('builder'),
    admission_generation:1,predecessor_claim_comment_id:claimId,...changes}});
const doneComment=(id,data,overrides={})=>({id,node_id:'IC_done_125_'+id,
  body:'<!-- ads:github -->\n'+fence+'yaml\n'+stringify(data).trimEnd()+'\n'+fence,
  html_url:issue+'#issuecomment-'+id,
  created_at:'2026-11-15T12:00:00Z',updated_at:'2026-11-15T12:00:00Z',
  user:{login:'scheduler-bot'},...overrides});
const exactDone=doneComment(802,doneData(801,histSha125));
const terminalRow=(changes={})=>({phase:'TERMINAL_VERIFIED',
  repository:owner+'/'+repo,issueNumber,issueRef:issue,
  dispatchId:'dispatch:125:hist',protectedClaimKey:keyFor('builder'),
  generation:1,commentId:801,commentRef:issue+'#issuecomment-801',
  sourceSha:histSha125,actorRole:'builder',operatorKind:'codex',
  operatorId:'codex:builder0',...changes});
const terminalPort=(row)=>{let reads=0;return {reads:()=>reads,
  readback:async()=>{reads++;return row;}};};

test('lawfully terminalized historical Claim admits a fresh successor with verified CAS readback',async()=>{
  const port=terminalPort(terminalRow());
  const x=await check({comments:[staleBuilder,exactDone]},
    {actorRole:'builder',historicalClaimCasReadback:port.readback});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(x.claimCandidates,0);
  assert.equal(x.claimCandidateRoles.length,0);
  assert.equal(x.terminalizedHistoricalClaimCount,1);
  const p=x.terminalizedHistoricalClaims[0];
  assert.equal(p.commentId,801);
  assert.equal(p.dispatchId,'dispatch:125:hist');
  assert.equal(p.protectedClaimKey,keyFor('builder'));
  assert.equal(p.claimGeneration,1);
  assert.equal(p.sourceSha,histSha125);
  assert.equal(p.terminalizedByDoneCommentId,802);
  assert.equal(port.reads(),1);
  noPermit(x);
});

test('old terminal Claim does not count as an active incompatible role',async()=>{
  const port=terminalPort(terminalRow());
  const x=await check({comments:[staleBuilder,exactDone]},
    {actorRole:'reviewer',historicalClaimCasReadback:port.readback});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.deepEqual(x.claimCandidateRoles,[]);
  assert.equal(x.claimRoleCandidates.length,0);
  assert.equal(x.terminalizedHistoricalClaimCount,1);
  noPermit(x);
});

test('terminal Claim without a CAS readback port blocks (missing readback)',async()=>{
  const x=await check({comments:[staleBuilder,exactDone]},{actorRole:'builder'});
  assert.equal(x.status,'BLOCKED');
  assert.equal(x.reason,'NATIVE_HISTORICAL_CAS_READBACK_UNAVAILABLE');
  noPermit(x);
});

test('PENDING and lost-ACK historical rows block after a real readback',async()=>{
  for(const phase of ['PENDING_GITHUB_PUBLISH','PENDING']){
    const port=terminalPort(terminalRow({phase}));
    const x=await check({comments:[staleBuilder,exactDone]},
      {actorRole:'builder',historicalClaimCasReadback:port.readback});
    assert.equal(x.status,'BLOCKED');
    assert.equal(x.reason,'NATIVE_HISTORICAL_CAS_ROW_PENDING');
    assert.equal(port.reads(),1);
    noPermit(x);
  }
});

test('divergent terminal row (wrong generation or comment binding) blocks',async()=>{
  for(const row of [
    terminalRow({generation:2}),
    terminalRow({commentId:999}),
    terminalRow({dispatchId:'dispatch:125:other'}),
    terminalRow({protectedClaimKey:keyFor('reviewer')}),
    terminalRow({phase:'PUBLISHED_VERIFIED'})
  ]){
    const port=terminalPort(row);
    const x=await check({comments:[staleBuilder,exactDone]},
      {actorRole:'builder',historicalClaimCasReadback:port.readback});
    assert.equal(x.status,'BLOCKED');
    assert.equal(x.reason,'NATIVE_HISTORICAL_CAS_ROW_DIVERGENT');
    noPermit(x);
  }
});

test('R34 CAS readback must bind the exact repository, Issue, source and Claim identity',async()=>{
  const missing=key=>{const row=terminalRow();delete row[key];return row;};
  const required=['repository','issueNumber','issueRef','dispatchId',
    'protectedClaimKey','generation','commentId','commentRef','sourceSha',
    'actorRole','operatorKind','operatorId'];
  const rows=[
    terminalRow({repository:'kaicreator-mm/other'}),
    terminalRow({issueNumber:999}),
    terminalRow({issueRef:'https://github.com/kaicreator-mm/tally/issues/999'}),
    terminalRow({sourceSha:'c'.repeat(40)}),
    terminalRow({commentRef:issue+'#issuecomment-999'}),
    terminalRow({actorRole:'reviewer'}),
    terminalRow({operatorKind:'other'}),
    terminalRow({operatorId:'codex:intruder'}),
    ...required.map(missing)
  ];
  for(const row of rows){
    const port=terminalPort(row);
    const x=await check({comments:[staleBuilder,exactDone]},
      {actorRole:'builder',historicalClaimCasReadback:port.readback});
    assert.equal(x.status,'BLOCKED');
    assert.equal(x.reason,'NATIVE_HISTORICAL_CAS_ROW_DIVERGENT');
    assert.equal(port.reads(),1);
    noPermit(x);
  }
});

test('R34 same-second DONE edit and unavailable GraphQL proof deny preflight',async()=>{
  for(const options of [{graphEdited:true},{graphUnavailable:true}]){
    const port=terminalPort(terminalRow());
    const x=await check({comments:[staleBuilder,exactDone],...options},
      {actorRole:'builder',historicalClaimCasReadback:port.readback});
    // The pair reader performs the first canonical scan and deliberately
    // projects every event-reader failure to its existing stale-source gate.
    assert.equal(x.status,'BLOCKED');
    assert.equal(x.reason,'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');
    assert.equal(port.reads(),0);
    noPermit(x);
  }
});

test('missing row and throwing port block as unavailable without leaking',async()=>{
  const missing=terminalPort(null);
  const x=await check({comments:[staleBuilder,exactDone]},
    {actorRole:'builder',historicalClaimCasReadback:missing.readback});
  assert.equal(x.status,'BLOCKED');
  assert.equal(x.reason,'NATIVE_HISTORICAL_CAS_READBACK_UNAVAILABLE');
  assert.equal(missing.reads(),1);
  const y=await check({comments:[staleBuilder,exactDone]},
    {actorRole:'builder',historicalClaimCasReadback:async()=>{
      throw Object.assign(new Error('PRIVATE-CAS-PATH'),{code:'EACCES'});}});
  assert.equal(y.status,'BLOCKED');
  assert.equal(y.reason,'NATIVE_HISTORICAL_CAS_READBACK_UNAVAILABLE');
  assert.equal(JSON.stringify(y).includes('PRIVATE-CAS-PATH'),false);
  noPermit(y);
});

test('nonterminal stale Claim without any DONE stays blocked on the pair surface',async()=>{
  const port=terminalPort(terminalRow());
  const x=await check({comments:[staleBuilder]},
    {actorRole:'reviewer',historicalClaimCasReadback:port.readback});
  assert.equal(x.status,'BLOCKED');
  assert.equal(x.reason,'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');
  assert.equal(port.reads(),0);
  noPermit(x);
});

test('supplied CAS port is ignored when nothing is terminalized',async()=>{
  const x=await check({comments:[builderClaim]},
    {actorRole:'reviewer',historicalClaimCasReadback:async()=>{
      throw new Error('must not be called');}});
  assert.equal(x.status,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.deepEqual(x.claimCandidateRoles,['builder']);
  assert.equal(x.terminalizedHistoricalClaimCount,0);
  noPermit(x);
});

test('malformed historicalClaimCasReadback option fails request validation',async()=>{
  const x=await check({comments:[staleBuilder,exactDone]},
    {actorRole:'builder',historicalClaimCasReadback:'not-a-function'});
  assert.equal(x.reason,'INVALID_NATIVE_PREFLIGHT_REQUEST');noPermit(x);
});

test('physical shared-SQLite two-process successor regression (real #219 shapes)',async t=>{
  assert.ok(Number(process.versions.node.split('.')[0])>=24,
    'physical successor lane requires Node >= 24 (built-in node:sqlite)');
  const {mkdtemp,rm}=await import('node:fs/promises');
  const {rmSync,existsSync}=await import('node:fs');
  const {tmpdir}=await import('node:os');
  const {join}=await import('node:path');
  const {fileURLToPath}=await import('node:url');
  const {spawnSync}=await import('node:child_process');
  const {DatabaseSync}=await import('node:sqlite');
  const {createHash}=await import('node:crypto');
  const dir=await mkdtemp(join(tmpdir(),'i422-r33-physical-'));
  t.after(async()=>{await rm(dir,{recursive:true,force:true});});
  const histPath=join(dir,'historical.sqlite');
  const freshPath=join(dir,'successor-registry.sqlite');
  const casKey='tally:v2:claim-admission:'+
    createHash('sha256').update('kaicreator-mm/tally#219:builder:__default__')
      .digest('hex');
  const schema='CREATE TABLE IF NOT EXISTS v8_host_docs '+
    '(key TEXT PRIMARY KEY, revision INTEGER NOT NULL, value TEXT NOT NULL)';
  const seed=(phase,generation,changes={})=>{
    rmSync(histPath,{force:true});
    rmSync(freshPath,{force:true});
    const db=new DatabaseSync(histPath);
    db.exec(schema);
    if(phase!==null){
      db.prepare('INSERT INTO v8_host_docs (key, revision, value) VALUES (?, 0, ?)')
        .run(casKey,JSON.stringify({
          phase,repository:'kaicreator-mm/tally',issueNumber:219,
          issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
          protectedClaimKey:'kaicreator-mm/tally#219:builder:__default__',
          generation,dispatchId:'dispatch:219:maintenance:20261031:1',
          actorRole:'builder',operatorKind:'other',
          operatorId:'zcode:i219:builder:1',
          writerId:'tally-host:historical:219:builder:1',
          sourceSha:'ad3123e7646428ec3b13fac5b6f510b683a133c8',
          commentId:6545079271,
          commentRef:'https://github.com/kaicreator-mm/tally/issues/219#issuecomment-6545079271',
          ...changes}));
    }
    db.close();
  };
  const run=(scn)=>{
    const r=spawnSync(process.execPath,[fileURLToPath(import.meta.url)],{
      env:{...process.env,I422_R33_CHILD_SCENARIO:scn,I422_R33_CHILD_DIR:dir},
      encoding:'utf8',timeout:120000});
    assert.equal(r.status,0,`child ${scn} exit ${r.status}: ${r.stderr}`);
    return JSON.parse(r.stdout.trim().split(/\r?\n/).pop());
  };
  // successor: READY preflight + TERMINAL_VERIFIED readback -> full
  // fresh-dedicated-registry admission from a second OS process.
  seed('TERMINAL_VERIFIED',1);
  const ok=run('successor');
  assert.equal(ok.preflightStatus,'NATIVE_SOURCE_PREFLIGHT_READY');
  assert.equal(ok.historicalClaimCount,1);
  assert.ok(ok.portReads>=1,'historical CAS row must be read back');
  assert.equal(ok.controllerStatus,'RECORDED');
  assert.equal(ok.controllerReason,'SERIALIZED_GITHUB_PUBLISH_AND_READBACK_CONFIRMED');
  assert.equal(ok.freshRowPhase,'PUBLISHED_VERIFIED');
  assert.equal(ok.githubPosts,1,'exactly one native claim POST');
  assert.equal(ok.freshRegistryCreated,true);
  const db=new DatabaseSync(histPath);
  const after=db.prepare(
    'SELECT revision, value FROM v8_host_docs WHERE key = ?').get(casKey);
  db.close();
  assert.equal(after.revision,0,'historical row must stay untouched');
  assert.equal(JSON.parse(after.value).phase,'TERMINAL_VERIFIED');
  // PENDING historical row: BLOCKED, readback happened, no blind POST, and
  // no fresh registry is created.
  seed('PENDING_GITHUB_PUBLISH',1);
  const pending=run('pending');
  assert.equal(pending.preflightStatus,'BLOCKED');
  assert.equal(pending.preflightReason,'NATIVE_HISTORICAL_CAS_ROW_PENDING');
  assert.equal(pending.portReads,1);
  assert.equal(pending.githubPosts,0);
  assert.equal(pending.freshRegistryCreated,false);
  assert.equal(existsSync(freshPath),false);
  // divergent terminal row: BLOCKED.
  seed('TERMINAL_VERIFIED',2);
  const divergent=run('divergent');
  assert.equal(divergent.preflightStatus,'BLOCKED');
  assert.equal(divergent.preflightReason,'NATIVE_HISTORICAL_CAS_ROW_DIVERGENT');
  assert.equal(divergent.githubPosts,0);
  assert.equal(existsSync(freshPath),false);
  // A wrong-source row under the real CAS key is still divergent: the four
  // legacy tuple fields alone must never qualify a fresh successor.
  seed('TERMINAL_VERIFIED',1,{sourceSha:'c'.repeat(40)});
  const wrongSource=run('divergent-source');
  assert.equal(wrongSource.preflightStatus,'BLOCKED');
  assert.equal(wrongSource.preflightReason,'NATIVE_HISTORICAL_CAS_ROW_DIVERGENT');
  assert.equal(wrongSource.portReads,1);
  assert.equal(wrongSource.githubPosts,0);
  assert.equal(wrongSource.freshRegistryCreated,false);
  assert.equal(existsSync(freshPath),false);
  // missing CAS readback: BLOCKED with UNKNOWN preserved.
  seed(null,1);
  const missing=run('missing');
  assert.equal(missing.preflightStatus,'BLOCKED');
  assert.equal(missing.preflightReason,'NATIVE_HISTORICAL_CAS_READBACK_UNAVAILABLE');
  assert.equal(missing.githubPosts,0);
  assert.equal(missing.freshRegistryCreated,false);
});

/** Physical child runner: real shared SQLite files + surrogate GitHub.
 * Fully self-contained (the guard above runs before module consts exist).
 * The historical registry reproduces the real #219 row/event shapes
 * (dispatch:219:maintenance:20261031:1, key #219:builder:__default__,
 * generation 1, predecessor comment 6545079271) read-only; the successor
 * gets its own fresh dedicated registry, never the historical file.
 */
async function runI422ChildScenario(scenario){
  const {createHash}=await import('node:crypto');
  const {DatabaseSync}=await import('node:sqlite');
  const {join}=await import('node:path');
  const {stringify:yamlStringify}=await import('yaml');
  const {readNativeHostClaimPreflightFromClient}=await import(
    '../src/task-claim-host-preflight.mjs');
  const {createHostSingleWriterClaimController}=await import(
    '../../../experiments/v2/t06d-host-claim-publication/single-writer.mjs');
  const {createHostBoundNativeGitHubClaimPublisher}=await import(
    '../src/task-claim-github-publisher.mjs');
  const {createHostNativeGitHubClaimReadbackPortFromClient}=await import(
    '../src/task-claim-host-readback-port.mjs');
  const sha256hex=x=>createHash('sha256').update(x,'utf8').digest('hex');
  const dir=process.env.I422_R33_CHILD_DIR;
  const histPath=join(dir,'historical.sqlite');
  const freshPath=join(dir,'successor-registry.sqlite');
  const curSha='fa49891dcb39c6eb8517f84087f890e8d4d47f93';
  const histSha='ad3123e7646428ec3b13fac5b6f510b683a133c8';
  const histKey='kaicreator-mm/tally#219:builder:__default__';
  const histDispatch='dispatch:219:maintenance:20261031:1';
  const histCommentId=6545079271;
  const issue219Url='https://github.com/kaicreator-mm/tally/issues/219';
  const at='2026-10-31T15:01:07Z';
  const fenceY=String.fromCharCode(96).repeat(3);
  const renderEvent=data=>'<!-- ads:github -->\n'+fenceY+'yaml\n'+
    yamlStringify(data).trimEnd()+'\n'+fenceY+'\n';
  const histClaimComment={
    id:histCommentId,
    body:renderEvent({schema:'ads/github/v3',kind:'event',data:{
      schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',actor_role:'builder',
      operator_kind:'other',operator_id:'zcode:i219:builder:1',task:'#219',
      issue:'#219',sha:histSha,dispatch_id:histDispatch,
      dispatch_state:'CLAIMED',protected_claim_key:histKey,
      admission_generation:1}}),
    html_url:issue219Url+'#issuecomment-'+histCommentId,
    created_at:at,updated_at:at,user:{login:'kaicreator-mm'}};
  const exactDoneComment={
    id:6894000001,node_id:'IC_test_219_6894000001',
    body:renderEvent({schema:'ads/github/v3',kind:'event',data:{
      schema:'ai-dev/event-v2',event:'DISPATCH_STATE_CHANGED',
      actor_role:'scheduler',operator_kind:'other',
      operator_id:'scheduler:i219:terminalization:1',
      task:'#219',issue:'#219',sha:histSha,
      dispatch_id:histDispatch,dispatch_state:'DONE',
      protected_claim_key:histKey,admission_generation:1,
      predecessor_claim_comment_id:histCommentId}}),
    html_url:issue219Url+'#issuecomment-6894000001',
    created_at:at,updated_at:at,user:{login:'kaicreator-mm'}};
  const posts=[];
  const published=[];
  let nextId=6895000001;
  const issue=()=>({id:919,number:219,html_url:issue219Url,state:'open',
    updated_at:at,labels:[{name:'type:task'},{name:'state:ready'}]});
  const octokit={state:{posts},rest:{issues:{
      async get(){return {data:issue()};},
      async listComments(){throw Error('MUST_USE_PAGINATION');},
      async createComment({body}){
        posts.push(body.length);
        const id=nextId++;
        const c={id,html_url:issue219Url+'#issuecomment-'+id,
          issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/219',
          body,created_at:at,updated_at:at,user:{login:'kaicreator-mm'}};
        published.push(c);
        return {data:c};
      },
      async getComment({comment_id}){
        const c=published.find(x=>x.id===comment_id);
        if(!c)throw Object.assign(new Error('not found'),{status:404});
        return {data:{...c}};
      }
    },
    repos:{async getCommit(){return {data:{sha:curSha}};}}},
    paginate:{async *iterator(route){
      if(typeof route==='string')yield {data:[]};
      else yield {data:[histClaimComment,exactDoneComment,...published]};
    }},
    graphql:async(_query,{id})=>{
      const c=[histClaimComment,exactDoneComment,...published]
        .find(x=>x.node_id===id);
      return {node:{__typename:'IssueComment',id,fullDatabaseId:String(c?.id),
        url:c?.html_url,body:c?.body,lastEditedAt:null,editor:null,
        issue:{number:219,repository:{nameWithOwner:'kaicreator-mm/tally'}}}};
    }};
  let portReads=0;
  const historicalClaimCasReadback=async binding=>{
    portReads++;
    const db=new DatabaseSync(histPath);
    try{
      const row=db.prepare(
        'SELECT revision, value FROM v8_host_docs WHERE key = ?')
        .get('tally:v2:claim-admission:'+sha256hex(binding.protectedClaimKey));
      return row==null?null:JSON.parse(row.value);
    }finally{db.close();}
  };
  const out={scenario};
  const observed=await readNativeHostClaimPreflightFromClient({
    octokit,owner:'kaicreator-mm',repo:'tally',issueNumber:219,
    expectedSourceSha:curSha,actorRole:'builder',limit:1000,
    historicalClaimCasReadback});
  out.preflightStatus=observed.status;
  out.preflightReason=observed.reason??null;
  out.portReads=portReads;
  out.githubPosts=posts.length;
  out.historicalClaimCount=observed.terminalizedHistoricalClaimCount??null;
  if(observed.status!=='NATIVE_SOURCE_PREFLIGHT_READY'){
    out.freshRegistryCreated=false;
    console.log(JSON.stringify(out));
    return;
  }
  // Fresh dedicated registry, created only after the verified READY + CAS
  // readback; the historical file is never opened for writing.
  const {existsSync}=await import('node:fs');
  if(existsSync(freshPath))throw new Error('FRESH_REGISTRY_MUST_BE_FRESH');
  const db=new DatabaseSync(freshPath);
  db.exec('CREATE TABLE IF NOT EXISTS v8_host_docs '+
    '(key TEXT PRIMARY KEY, revision INTEGER NOT NULL, value TEXT NOT NULL)');
  const getStmt=db.prepare('SELECT revision, value FROM v8_host_docs WHERE key = ?');
  const insStmt=db.prepare(
    'INSERT INTO v8_host_docs (key, revision, value) VALUES (?, 0, ?)');
  const updStmt=db.prepare(
    'UPDATE v8_host_docs SET revision = revision + 1, value = ? WHERE key = ? AND revision = ?');
  const store={
    async get(key){
      const r=getStmt.get(key);
      return r==null?null:{revision:r.revision,value:JSON.parse(r.value)};
    },
    async put(key,value,conditions){
      if(!conditions||typeof conditions!=='object'||
        (conditions.mustBeAbsent!==true&&conditions.mustMatchRevision===undefined))
        throw Object.assign(new Error('HOST_PUT_CONDITION_REQUIRED'),
          {code:'HOST_PUT_CONDITION_REQUIRED'});
      if(conditions.mustBeAbsent===true){
        if(getStmt.get(key))
          throw Object.assign(new Error('HOST_KEY_EXISTS'),{code:'HOST_KEY_EXISTS'});
        insStmt.run(key,JSON.stringify(value));
        return {revision:0};
      }
      const r=updStmt.run(JSON.stringify(value),key,conditions.mustMatchRevision);
      if(Number(r.changes)!==1)
        throw Object.assign(new Error('HOST_CAS_CONFLICT'),{code:'HOST_CAS_CONFLICT'});
      return {revision:conditions.mustMatchRevision+1};
    },
    async close(){db.close();}
  };
  out.freshRegistryCreated=true;
  try{
    const writerAuthority=Object.freeze({
      mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
      repository:'kaicreator-mm/tally',
      writerId:'tally-host:win32:i422:r33:physical-probe:1',
      githubWriterLogin:'kaicreator-mm'});
    const deny=(reason,request)=>Object.freeze({
      status:'BLOCKED',reason,sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',
      sourceQualified:false,complete:false,
      issueRef:request?.issueRef??null,sourceSha:request?.sourceSha??null,
      activeIncompatibleClaimCount:null,dispatchSourceBound:false,
      publisherSourceBound:false,claimStatus:'NOT_VERIFIED',
      authorized:false,admitted:false,taskPass:false,releasePass:false});
    const source={
      // Successor projection of the reviewed host-admission-source semantics,
      // supplying the #422 historical CAS readback port (which the reviewed
      // production adapter predates; its own evolution is separately governed).
      async readAdmissionFacts(i){
        const again=await readNativeHostClaimPreflightFromClient({
          octokit,owner:'kaicreator-mm',repo:'tally',issueNumber:219,
          expectedSourceSha:curSha,actorRole:i.actorRole,limit:1000,
          historicalClaimCasReadback});
        if(again.status!=='NATIVE_SOURCE_PREFLIGHT_READY')
          return deny(again.reason??'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE',i);
        return Object.freeze({
          sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',sourceQualified:true,
          complete:true,issueRef:i.issueRef,sourceSha:i.sourceSha,
          issueState:again.issueState,workflowState:again.workflowState,
          workItemType:again.workItemType,dependencyState:again.dependencyState,
          activeIncompatibleClaimCount:0,dispatchSourceBound:true,
          publisherSourceBound:true,claimStatus:'NOT_VERIFIED',
          authorized:false,admitted:false,taskPass:false,releasePass:false});
      },
      async readPublishedClaim(i,id){
        const port=createHostNativeGitHubClaimReadbackPortFromClient({
          octokit,owner:'kaicreator-mm',repo:'tally',
          writerLogin:'kaicreator-mm'});
        return port.readPublishedClaim(i,id);
      }
    };
    const publisher=createHostBoundNativeGitHubClaimPublisher({
      octokit,owner:'kaicreator-mm',repo:'tally',writerAuthority});
    const controller=createHostSingleWriterClaimController({
      store,source,publisher,writerAuthority});
    const result=await controller.reserveAndPublish({
      owner:'kaicreator-mm',repo:'tally',issueNumber:219,sourceSha:curSha,
      actorRole:'builder',operatorKind:'other',
      operatorId:'zcode:i422:r33:physical-probe:1',
      dispatchId:'dispatch:219:succession-probe:physical:1',
      expectedGeneration:0});
    out.controllerStatus=result.status;
    out.controllerReason=result.reason??null;
    const row=await store.get(
      'tally:v2:claim-admission:'+sha256hex(histKey));
    out.freshRowPhase=row?.value?.phase??null;
    out.githubPosts=posts.length;
  }finally{
    await store.close();
  }
  console.log(JSON.stringify(out));
}
