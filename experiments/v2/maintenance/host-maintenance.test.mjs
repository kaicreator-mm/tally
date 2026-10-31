import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {existsSync,mkdirSync,writeFileSync,readFileSync,mkdtempSync,rmSync,
  readdirSync,copyFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createMaintenanceHost,recordMaintenanceCandidate,
  observeLearningSuggestions,validateMaintenanceCandidate,
  maintenanceCandidateDigest,learningSuggestionsFromReports,
  MAINTENANCE_CANDIDATE_KEY,MAINTENANCE_ISSUE_KEY,MAINTENANCE_PENDING_KEY,
  MAINTENANCE_LEARNING_KEY,MAINTENANCE_CLAIM_KEY,__test} from './host-maintenance.mjs';

const REPO_ROOT=fileURLToPath(new URL('../../../',import.meta.url));
const run=(cmd,args,opts={})=>{
  // Node refuses to spawn .cmd shims without a shell; route npm via cmd.exe.
  const spawnCmd=process.platform==='win32'&&cmd==='npm'?
    process.env.comspec??'cmd.exe':cmd;
  const spawnArgs=process.platform==='win32'&&cmd==='npm'?['/c','npm',...args]:args;
  const r=spawnSync(spawnCmd,spawnArgs,{encoding:'utf8',windowsHide:true,...opts});
  return {status:r.status,stdout:r.stdout??'',stderr:r.stderr??''};};
const sha256HexOf=value=>createHash('sha256').update(
  JSON.stringify(value)).digest('hex');

const sha='a'.repeat(40),digest=maintenanceCandidateDigest({
  sourceIssueRef:'https://github.com/kaicreator-mm/tally/issues/207',
  requirementRef:'https://github.com/kaicreator-mm/tally/issues/92',
  subjectSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
  summary:'Post-release behavior regression against immutable v1.5.5',
  priorRelease:{version:'v1.5.5',commitSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
    evidenceRef:'https://github.com/kaicreator-mm/tally/releases/tag/v1.5.5'}});
const candidate=()=>({sourceIssueRef:'https://github.com/kaicreator-mm/tally/issues/207',
  requirementRef:'https://github.com/kaicreator-mm/tally/issues/92',
  subjectSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
  summary:'Post-release behavior regression against immutable v1.5.5',
  priorRelease:{version:'v1.5.5',commitSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
    evidenceRef:'https://github.com/kaicreator-mm/tally/releases/tag/v1.5.5'}});
const env=()=>({TALLY_CLAIM_SOURCE_SHA:sha,TALLY_GITHUB_READ_TOKEN:'token-1234567890',
  TALLY_TASK_CLAIM_219_REGISTRY_PATH:'C:/registry/registry-219.sqlite'});
function memoryStore(){const docs=new Map();let creates=0;
  return {store:{async get(key){const d=docs.get(key);
      return d===undefined?null:{value:d.value,revision:d.revision};},
    async put(key,value,condition){
      if(condition?.mustBeAbsent===true&&docs.has(key))
        throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
      const previous=docs.get(key);
      const revision=(previous?previous.revision:0)+1;
      docs.set(key,{value,revision});
      if(condition?.mustBeAbsent===true)creates++;
      return {revision};},
    async close(){}},docs,createCount:()=>creates};}
const reviewArgs=(over={})=>({openStore:async()=>memoryStore().store,
  registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
  candidate:candidate(),author:'author:1',reviewer:'reviewer:1',
  reviewedAt:'2026-10-31T09:00:00.000Z',...over});
const persisted=(over={})=>{const m=memoryStore();
  return {m,openStore:async()=>m.store,...over};};
const reviewedFrozen=async(over={})=>{const ctx=persisted(over);
  const frozen=await recordMaintenanceCandidate({...reviewArgs({openStore:ctx.openStore}),
    ...over.args});
  assert.equal(frozen.state,'RECORDED_COMPLETE',JSON.stringify(frozen));
  return ctx;};
const host=(ctx={},overrides={})=>createMaintenanceHost({
  openStore:ctx.openStore??(async()=>memoryStore().store),
  verifyAcceptedClaim:async()=>({ok:true}),
  verifyFrozenWriteSet:async()=>({ok:true}),env:env(),...overrides});
const publishInput=(over={})=>({issueNumber:219,sourceRef:__test.ISSUE_REF,
  sourceSha:sha,candidateDigest:digest,...over});
const issueReadback=n=>({state:'READBACK_VERIFIED',issue:{number:n,
  title:'[maintenance] Post-release behavior regression against immutable v1.5.5',
  url:`https://github.com/kaicreator-mm/tally/issues/${n}`}});
const publishTransport=(n=302)=>({
  publishIssue:async()=>({state:'ISSUE_PUBLISHED',issue:issueReadback(n).issue}),
  findIssue:async()=>({state:'RECONCILE_COMPLETE',reason:'MAINTENANCE_ISSUE_ABSENT',
    matches:[]}),
  readIssue:async()=>issueReadback(n)});

test('candidate validation keeps the prior release immutable and missing release UNKNOWN',()=>{
  const valid=validateMaintenanceCandidate(candidate());
  assert.equal(valid.priorRelease.immutable,true);
  assert.equal(valid.priorRelease.version,'v1.5.5');
  assert.equal(maintenanceCandidateDigest(candidate()),digest);
  const noRelease=validateMaintenanceCandidate({...candidate(),priorRelease:null});
  assert.deepEqual(noRelease.priorRelease,{status:'UNKNOWN'});
  for(const bad of [null,{},[],
    {...candidate(),sourceIssueRef:'https://github.com/other/tally/issues/207'},
    {...candidate(),summary:'x'.repeat(513)},
    {...candidate(),priorRelease:{...candidate().priorRelease,commitSha:'nope'}},
    {...candidate(),priorRelease:{...candidate().priorRelease,evidenceRef:'https://evil.example/x'}}])
    assert.equal(validateMaintenanceCandidate(bad),null,JSON.stringify(bad).slice(0,60));
  assert.equal(maintenanceCandidateDigest({}),null);
});

test('independent review freeze lands once in the Host CAS and replays exactly',async()=>{
  const ctx=persisted();
  const args=reviewArgs({openStore:ctx.openStore});
  const selfReview=await recordMaintenanceCandidate({...args,reviewer:'author:1'});
  assert.equal(selfReview.reason,'CANDIDATE_REVIEW_NOT_INDEPENDENT');
  const frozen=await recordMaintenanceCandidate(args);
  assert.equal(frozen.state,'RECORDED_COMPLETE');
  assert.equal(frozen.candidateDigest,digest);
  assert.equal(frozen.receiptRef,`receipt:sha256:${digest}`);
  assert.equal(frozen.githubEffect,'NOT_RUN');
  assert.equal(frozen.historicalReleaseMutation,false);
  const record=(await ctx.openStore().then(s=>s.get(MAINTENANCE_CANDIDATE_KEY))).value;
  assert.equal(record.phase,'REVIEWED_MAINTENANCE_CANDIDATE');
  assert.equal(record.review.reviewer,'reviewer:1');
  const replay=await recordMaintenanceCandidate(args);
  assert.equal(replay.state,'RECORDED_COMPLETE');
  assert.equal(replay.reason,'MAINTENANCE_REVIEW_ALREADY_RECORDED');
  assert.equal((await recordMaintenanceCandidate({...args,
    candidate:{...candidate(),summary:'changed summary'}})).reason,
    'MAINTENANCE_REVIEW_DIGEST_MISMATCH');
  const noStore=await recordMaintenanceCandidate({...args,openStore:null});
  assert.equal(noStore.reason,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');
});

const summaryObservation=over=>({envelope:{schema:'ads/github/v5',kind:'report',
    data:{report_kind:'EXECUTION_SUMMARY',report_id:'rep-1',revision:1,
      publisher_namespace:'tally',capture_coverage:'COMPLETE',
      verifier:{status:'QUALIFIED'},
      subject:{task_ref:'https://github.com/kaicreator-mm/tally/issues/172',
        issue_ref:'https://github.com/kaicreator-mm/tally/issues/203',
        source_status:'PINNED'}}},
  validation:{schema:'ads/github/v5',passed:true},
  native:{repository:'kaicreator-mm/tally',issueNumber:203,commentId:6540000001,
    dataDigest:'d'.repeat(64),sourceSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
    permission:'GRANTED',readback:'VERIFIED',pagination:'COMPLETE',
    currentness:'CURRENT'},...over});
// learningId is deterministic: sha256 of the canonical identity array.
const LEARNING_ID=sha256HexOf(['rep-1',1,6540000001,'EXECUTION_SUMMARY']);

test('learning provenance derives from existing public ADS reports only',()=>{
  const suggestions=learningSuggestionsFromReports([summaryObservation()]);
  assert.equal(suggestions.length,1);
  const row=suggestions[0];
  assert.equal(row.learningId,LEARNING_ID);
  assert.equal(row.qualification,'REPORTED_QUALIFIED');
  assert.equal(row.taskIssueRef,'https://github.com/kaicreator-mm/tally/issues/172');
  assert.equal(row.targetIssueRef,'https://github.com/kaicreator-mm/tally/issues/203');
  assert.equal(row.provenance.reportId,'rep-1');
  assert.equal(row.provenance.commentId,6540000001);
  assert.match(row.provenance.publicationRef,/issues\/203#issuecomment-/);
  assert.equal(row.measuredEffect,'NOT_MEASURED');
  assert.equal(row.noAgentRanking,true);assert.equal(row.qualityGainClaimed,false);
  const stale=learningSuggestionsFromReports([summaryObservation({
    native:{...summaryObservation().native,currentness:'STALE'}})]);
  assert.equal(stale[0].qualification,'UNKNOWN');
  assert.equal(stale[0].learningId,LEARNING_ID);
  const ungranted=learningSuggestionsFromReports([summaryObservation({
    native:{...summaryObservation().native,permission:'DENIED'}})]);
  assert.equal(ungranted[0].qualification,'UNKNOWN');
  // A report that fails pinned-schema validation fails the whole set closed.
  for(const bad of [null,[{}],
    [summaryObservation({validation:{schema:'ads/github/v5',passed:false}})],
    [{envelope:{schema:'ads/github/v3'},validation:{},native:{}}]])
    assert.equal(learningSuggestionsFromReports(bad),null,'invalid observation set refuses');
});

test('learning decision and later independent outcome are ledger-gated and idempotent',async()=>{
  const ctx=await reviewedFrozen();
  const unproven=await observeLearningSuggestions({openStore:ctx.openStore,
    registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
    observations:[{envelope:{schema:'ads/github/v3'},validation:{},native:{}}],
    observedAt:'2026-10-31T09:10:00.000Z'});
  assert.equal(unproven.reason,'LEARNING_PROVENANCE_UNVERIFIED');
  const observed=await observeLearningSuggestions({openStore:ctx.openStore,
    registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
    observations:[summaryObservation()],observedAt:'2026-10-31T09:10:00.000Z'});
  assert.equal(observed.state,'RECORDED_COMPLETE');
  assert.equal(observed.suggestions[0].qualification,'REPORTED_QUALIFIED');
  const again=await observeLearningSuggestions({openStore:ctx.openStore,
    registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
    observations:[summaryObservation()],observedAt:'2026-10-31T09:11:00.000Z'});
  assert.equal(again.reason,'LEARNING_LEDGER_ALREADY_RECORDED');
  const learningInput=over=>({issueNumber:219,sourceRef:__test.ISSUE_REF,
    sourceSha:sha,learningId:LEARNING_ID,...over});
  const h=host(ctx);
  const decision=await h.hostRecordTaskLearningOutcome(learningInput({decision:'ADOPTED'}));
  assert.equal(decision.state,'LEARNING_RECORDED');
  assert.equal(decision.reason,'LEARNING_DECISION_RECORDED_IN_HOST_CAS');
  assert.equal(decision.githubEffect,'NOT_RUN');
  assert.equal(decision.measuredEffect,'NOT_MEASURED');
  assert.equal(decision.noAgentRanking,true);
  assert.equal(decision.qualityGainClaimed,false);
  const replayDecision=await h.hostRecordTaskLearningOutcome(
    learningInput({decision:'ADOPTED'}));
  assert.equal(replayDecision.reason,'LEARNING_DECISION_ALREADY_RECORDED');
  assert.equal((await h.hostRecordTaskLearningOutcome(
    learningInput({decision:'REJECTED'}))).reason,'LEARNING_DECISION_MISMATCH');
  const outcome=await h.hostRecordTaskLearningOutcome(learningInput({decision:'ADOPTED',
    outcome:{attested:false,observedAt:'2026-11-02T10:00:00.000Z',
      note:'independent follow-up observed no measurable change'}}));
  assert.equal(outcome.state,'LEARNING_RECORDED');
  assert.equal(outcome.outcome.measuredEffect,'NOT_MEASURED');
  assert.equal(outcome.outcome.attested,false);
  assert.equal(outcome.githubEffect,'NOT_RUN');
  assert.equal((await h.hostRecordTaskLearningOutcome(learningInput({decision:'ADOPTED',
    outcome:{attested:false,observedAt:'2026-11-02T10:00:00.000Z',note:'other'}})))
    .reason,'LEARNING_OUTCOME_MISMATCH');
  const attestedOutcome=await attestedRunner();
  assert.equal(attestedOutcome.outcome.measuredEffect,'ATTESTED_OUTCOME_ONLY');
  assert.equal(attestedOutcome.qualityGainClaimed,false);
  async function attestedRunner(){
    const ctx2=await reviewedFrozen();
    await observeLearningSuggestions({openStore:ctx2.openStore,
      registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
      observations:[summaryObservation()],observedAt:'2026-10-31T09:10:00.000Z'});
    const h2=host(ctx2);
    await h2.hostRecordTaskLearningOutcome(learningInput({decision:'ADOPTED'}));
    return h2.hostRecordTaskLearningOutcome(learningInput({
      decision:'ADOPTED',outcome:{attested:true,observedAt:'2026-11-02T10:00:00.000Z',
        note:'independent readback of the follow-up release'}}));
  }
  // Outcome cannot precede its decision: ledger present, no decision record.
  const ledgerOnly=await (async()=>{const c=await reviewedFrozen();
    await observeLearningSuggestions({openStore:c.openStore,
      registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
      observations:[summaryObservation()],observedAt:'2026-10-31T09:10:00.000Z'});
    return c;})();
  const noDecision=await host(ledgerOnly).hostRecordTaskLearningOutcome(
    learningInput({decision:'ADOPTED',outcome:{attested:false,
      observedAt:'2026-11-02T10:00:00.000Z',note:'x'}}));
  assert.equal(noDecision.reason,'LEARNING_DECISION_REQUIRED');
  const unknownLearning=await h.hostRecordTaskLearningOutcome(
    learningInput({learningId:'f'.repeat(64),decision:'ADOPTED'}));
  assert.equal(unknownLearning.reason,'LEARNING_PROVENANCE_NOT_VERIFIED');
  const badInput=await h.hostRecordTaskLearningOutcome(learningInput({decision:'RANK_FIRST'}));
  assert.equal(badInput.reason,'MAINTENANCE_INPUT_INVALID');
});

test('full governed publish: claim, write-set, frozen review, publish and readback',async()=>{
  const ctx=await reviewedFrozen();
  const calls=[];
  const h=host(ctx,{...publishTransport(302),
    publishIssue:async arg=>{calls.push(arg);
      return {state:'ISSUE_PUBLISHED',issue:issueReadback(302).issue};}});
  const result=await h.hostPublishMaintenanceIssue(publishInput());
  assert.equal(result.state,'MAINTENANCE_ISSUE_PUBLISHED');
  assert.equal(result.reason,'MAINTENANCE_ISSUE_PUBLISHED_AND_READBACK_VERIFIED');
  assert.equal(result.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(result.claimAccepted,true);assert.equal(result.admitted,true);
  assert.equal(result.authorized,false);assert.equal(result.duplicate,false);
  assert.equal(result.historicalReleaseMutation,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.equal(result.issue.number,302);
  assert.match(calls[0].body,/immutable, never rewritten/);
  assert.ok(calls[0].body.includes(`tally:maintenance:219:${digest}`));
  assert.match(calls[0].body,/no quality-gain inference/);
  assert.equal(JSON.stringify(calls).includes('token-1234567890'),false,
    'Host module must not hand credentials to transports');
  const second=await host(ctx,publishTransport(302))
    .hostPublishMaintenanceIssue(publishInput());
  assert.equal(second.reason,'MAINTENANCE_ISSUE_READBACK_ALREADY_PRESENT');
  assert.equal(second.alreadyPresent,true);assert.equal(second.issue.number,302);
});

test('missing claim, write-set, review record or transport fails closed',async()=>{
  const ctx=await reviewedFrozen();
  const cases=[
    [host(ctx,{verifyAcceptedClaim:async()=>({ok:false,
      reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND'})}),'CLAIM_REGISTRY_RECORD_NOT_FOUND'],
    [host(ctx,{verifyFrozenWriteSet:async()=>({ok:false,
      reason:'DAG_WRITESET_NOT_VERIFIED'})}),'DAG_WRITESET_NOT_VERIFIED'],
    [host({}),'CANDIDATE_REVIEW_NOT_VERIFIED'],
    [createMaintenanceHost({env:env()}),'MAINTENANCE_HOST_CONFIG_INVALID'],
    [host(ctx,{env:{...env(),TALLY_GITHUB_READ_TOKEN:undefined}}),
      'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE'],
    [host(ctx,{publishIssue:undefined}),'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE'],
  ];
  for(const [h,reason] of cases){
    const result=await h.hostPublishMaintenanceIssue(publishInput());
    assert.equal(result.state,'NOT_ADMITTED');assert.equal(result.reason,reason);
    assert.equal(result.githubEffect,'NOT_RUN');assert.equal(result.claimAccepted,false);
  }
});

test('host pin refuses cross-issue, stale source and malformed digests first',async()=>{
  const ctx=await reviewedFrozen();
  let verifierCalls=0;
  const counting=createMaintenanceHost({openStore:ctx.openStore,
    verifyAcceptedClaim:async()=>{verifierCalls++;return {ok:true};},
    verifyFrozenWriteSet:async()=>({ok:true}),env:env()});
  for(const change of [{issueNumber:210},{sourceSha:'short'},
    {sourceSha:'b'.repeat(40)},{candidateDigest:'nothex'}]){
    const result=await counting.hostPublishMaintenanceIssue(publishInput(change));
    assert.equal(result.reason,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  }
  assert.equal(verifierCalls,0,'no verifier may run on pinned-input refusal');
  const mismatched=await host(ctx).hostPublishMaintenanceIssue(
    publishInput({candidateDigest:'d'.repeat(64)}));
  assert.equal(mismatched.reason,'MAINTENANCE_REVIEW_DIGEST_MISMATCH');
  const badStore=memoryStore();
  await badStore.store.put(MAINTENANCE_CANDIDATE_KEY,{phase:'REVIEWED_MAINTENANCE_CANDIDATE',
    sourceSha:sha,candidateDigest:digest,review:{author:'same:1',reviewer:'same:1'}});
  const notIndependent=await host({openStore:async()=>badStore.store})
    .hostPublishMaintenanceIssue(publishInput());
  assert.equal(notIndependent.reason,'CANDIDATE_REVIEW_NOT_INDEPENDENT');
});

test('safe replay admits the intent but keeps the GitHub effect NOT_RUN',async()=>{
  const ctx=await reviewedFrozen();
  const result=await host(ctx,{env:{...env(),TALLY_LOCAL_REPLAY:'1'}})
    .hostPublishMaintenanceIssue(publishInput());
  assert.equal(result.state,'REPLAY_COMPLETE');
  assert.equal(result.reason,'REPLAY_FIXTURE_ONLY');
  assert.equal(result.githubEffect,'NOT_RUN');assert.equal(result.replayFixture,true);
  await observeLearningSuggestions({openStore:ctx.openStore,
    registryPath:'C:/registry/registry-219.sqlite',sourceSha:sha,
    observations:[summaryObservation()],observedAt:'2026-10-31T09:10:00.000Z'});
  const replayLearning=await host(ctx,{env:{...env(),TALLY_LOCAL_REPLAY:'1'}})
    .hostRecordTaskLearningOutcome({issueNumber:219,sourceRef:__test.ISSUE_REF,
      sourceSha:sha,learningId:LEARNING_ID,decision:'ADOPTED'});
  assert.equal(replayLearning.state,'REPLAY_COMPLETE');
  assert.equal(replayLearning.measuredEffect,'NOT_MEASURED');
});

test('lost ACK stays UNKNOWN and a later run reconciles without duplicates',async()=>{
  const ctx=await reviewedFrozen();
  let publishCalls=0;
  const flaky=host(ctx,{publishIssue:async()=>{publishCalls++;
      throw new Error('connection reset after request left');},
    findIssue:async()=>({state:'RECONCILE_COMPLETE',reason:'MAINTENANCE_ISSUE_ABSENT',
      matches:[]}),readIssue:async()=>issueReadback(302)});
  const lost=await flaky.hostPublishMaintenanceIssue(publishInput());
  assert.equal(lost.state,'PUBLICATION_UNKNOWN');
  assert.equal(lost.reason,'GITHUB_PUBLISH_UNKNOWN');
  assert.equal(lost.githubEffect,'UNKNOWN');assert.equal(lost.issue,null);
  assert.equal(publishCalls,1);
  // The pending marker recorded before the publish enables reconciliation.
  const pendingStore=await ctx.openStore();
  const pending=(await pendingStore.get(MAINTENANCE_PENDING_KEY)).value;
  assert.equal(pending.phase,'PENDING_MAINTENANCE_ISSUE');
  assert.equal(pending.candidateDigest,digest);
  const reconcile=host(ctx,{findIssue:async()=>({state:'RECONCILE_COMPLETE',
      reason:'MAINTENANCE_ISSUE_FOUND',matches:[issueReadback(302).issue]}),
    readIssue:async()=>issueReadback(302),
    publishIssue:async()=>{publishCalls++;return {state:'ISSUE_PUBLISHED'};}});
  const recovered=await reconcile.hostPublishMaintenanceIssue(publishInput());
  assert.equal(recovered.state,'MAINTENANCE_ISSUE_PUBLISHED');
  assert.equal(recovered.reason,'MAINTENANCE_ISSUE_RECONCILED_AFTER_LOST_ACK');
  assert.equal(recovered.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(recovered.issue.number,302);
  assert.equal(publishCalls,1,'reconciliation must not create a second Issue');
  // Readback mismatch after reconcile keeps UNKNOWN, never a verified claim.
  const ctx2=await reviewedFrozen();
  const issueReadBox=n=>({number:n,title:'t',
    url:`https://github.com/kaicreator-mm/tally/issues/${n}`});
  await ctx2.openStore().then(s=>s.put(MAINTENANCE_PENDING_KEY,{phase:'PENDING_MAINTENANCE_ISSUE',
    sourceSha:sha,candidateDigest:digest,marker:'m',title:'t'}));
  const mismatch=await host(ctx2,{findIssue:async()=>({state:'RECONCILE_COMPLETE',
      reason:'MAINTENANCE_ISSUE_FOUND',matches:[issueReadBox(303)]}),
    readIssue:async()=>({state:'READBACK_MISMATCH',
      reason:'MAINTENANCE_ISSUE_READBACK_MISMATCH',issue:null}),
    publishIssue:async()=>({state:'ISSUE_PUBLISHED'})})
    .hostPublishMaintenanceIssue(publishInput());
  assert.equal(mismatch.state,'PUBLICATION_UNKNOWN');
  assert.equal(mismatch.reason,'MAINTENANCE_ISSUE_READBACK_MISMATCH');
  // Two marker matches are an identity conflict: never a third Issue.
  const conflict=await host(ctx2,{findIssue:async()=>({state:'RECONCILE_CONFLICT',
      reason:'MAINTENANCE_ISSUE_IDENTITY_CONFLICT',matches:[{number:1},{number:2}]}),
    readIssue:async()=>issueReadback(1),publishIssue:async()=>({state:'ISSUE_PUBLISHED'})})
    .hostPublishMaintenanceIssue(publishInput());
  assert.equal(conflict.state,'PUBLICATION_UNKNOWN');
  assert.equal(conflict.reason,'MAINTENANCE_ISSUE_IDENTITY_CONFLICT');
  // A publish transport that reports an unverified state stays UNKNOWN.
  const ctx3=await reviewedFrozen();
  const badPublish=await host(ctx3,{...publishTransport(302),
    publishIssue:async()=>({state:'PUBLICATION_UNKNOWN',reason:'GITHUB_RATE_LIMITED'})})
    .hostPublishMaintenanceIssue(publishInput());
  assert.equal(badPublish.state,'PUBLICATION_UNKNOWN');
  assert.equal(badPublish.reason,'GITHUB_RATE_LIMITED');
});

// --- J219 sealed producer + real K consumer replay (env-gated, no new paths) ---
// Runs ONLY with J219_REPLAY=1 and DH_V0103=<DH v0.10.3 checkout>. It reuses the
// real producer.mjs with T219_MAINTENANCE=1 (and without it), then loads the
// sealed app with the REAL Microkernel and sends publishMaintenanceIssue /
// recordTaskLearningOutcome through facade.send with a replay-mode provider
// that reuses the real host-maintenance.mjs logic. Physical GitHub effects and
// Claim/registry verification stay explicitly NOT_RUN.
const REPLAY_DH_SHA='061376f4a023070f215f9feffad5c24d96d503cb';
const REPLAY_CANARY='UNIT_TEST_SECRET_j219_replay_do_not_leak';
const REPLAY_PROVIDER=String.raw`import {createMaintenanceHost,recordMaintenanceCandidate,
  observeLearningSuggestions,MAINTENANCE_CANDIDATE_KEY,MAINTENANCE_LEARNING_KEY}
  from '__TALLY_ROOT__experiments/v2/maintenance/host-maintenance.mjs';
const docs=new Map();
const openStore=async()=>({
  async get(key){const d=docs.get(key);return d===undefined?null:{value:d.value,revision:d.revision};},
  async put(key,value,condition){
    if(condition&&condition.mustBeAbsent===true&&docs.has(key))
      throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
    const previous=docs.get(key);
    const revision=(previous?previous.revision:0)+1;
    docs.set(key,{value,revision});
    return {revision};},
  async close(){}});
const registryPath='T219_REPLAY_IN_MEMORY';
const sourceSha=process.env.TALLY_CLAIM_SOURCE_SHA;
const candidate={
  sourceIssueRef:'https://github.com/kaicreator-mm/tally/issues/207',
  requirementRef:'https://github.com/kaicreator-mm/tally/issues/92',
  subjectSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
  summary:'Post-release behavior regression against immutable v1.5.5',
  priorRelease:{version:'v1.5.5',commitSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
    evidenceRef:'https://github.com/kaicreator-mm/tally/releases/tag/v1.5.5'}};
const review=await recordMaintenanceCandidate({openStore,registryPath,sourceSha,
  candidate,author:'author:j219',reviewer:'reviewer:j219',
  reviewedAt:'2026-10-31T09:00:00.000Z'});
if(review.state!=='RECORDED_COMPLETE')throw new Error('J219_REVIEW_SEED_FAILED');
const summaryObservation={envelope:{schema:'ads/github/v5',kind:'report',
    data:{report_kind:'EXECUTION_SUMMARY',report_id:'rep-1',revision:1,
      publisher_namespace:'tally',capture_coverage:'COMPLETE',
      verifier:{status:'QUALIFIED'},
      subject:{task_ref:'https://github.com/kaicreator-mm/tally/issues/172',
        issue_ref:'https://github.com/kaicreator-mm/tally/issues/203',
        source_status:'PINNED'}}},
  validation:{schema:'ads/github/v5',passed:true},
  native:{repository:'kaicreator-mm/tally',issueNumber:203,commentId:6540000001,
    dataDigest:'d'.repeat(64),sourceSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
    permission:'GRANTED',readback:'VERIFIED',pagination:'COMPLETE',
    currentness:'CURRENT'}};
const ledger=await observeLearningSuggestions({openStore,registryPath,sourceSha,
  observations:[summaryObservation],observedAt:'2026-10-31T09:10:00.000Z'});
if(ledger.state!=='RECORDED_COMPLETE')throw new Error('J219_LEDGER_SEED_FAILED');
const env=Object.assign({},process.env,{TALLY_LOCAL_REPLAY:'1',
  TALLY_TASK_CLAIM_219_REGISTRY_PATH:registryPath});
const host=createMaintenanceHost({env,openStore,
  verifyAcceptedClaim:async()=>({ok:true}),
  verifyFrozenWriteSet:async()=>({ok:true})});
export async function publishMaintenanceIssue(input){
  const result=await host.hostPublishMaintenanceIssue(input);
  console.error('J219_PROVIDER_PUBLISH_RETURN',JSON.stringify(result));
  return result;
}
export async function recordTaskLearningOutcome(input){
  const result=await host.hostRecordTaskLearningOutcome(input);
  console.error('J219_PROVIDER_LEARNING_RETURN',JSON.stringify(result));
  return result;
}
export async function runLocalCodexAttempt(input){
  return {issueRef:input&&input.sourceRef,sourceSha:input&&input.sourceSha,
    attemptId:input&&input.attemptId,claimGeneration:1,claimAccepted:true,
    admitted:true,agentLaunch:false,replayFixture:true,authorized:false,
    taskPass:false,releasePass:false,state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY'};
}
export async function joinReviewedCandidates(input){
  return {issueRef:input&&input.sourceRef,sourceSha:input&&input.sourceSha,
    joinId:input&&input.joinId,claimGeneration:1,claimAccepted:true,
    admitted:true,agentLaunch:false,replayFixture:true,authorized:false,
    taskPass:false,releasePass:false,state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY'};
}
export async function publishReviewedCandidate(){
  return {state:'NOT_ADMITTED',reason:'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE',
    authorized:false,taskPass:false,releasePass:false};
}
export async function hostProjectIntake(){
  return {state:'NOT_ADMITTED',reason:'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE',
    authorized:false,taskPass:false,releasePass:false};
}
export async function freezeProductIntake(){
  return {state:'NOT_FROZEN',reason:'PRODUCT_FREEZE_INPUT_INVALID',
    authorized:false,admitted:false,taskPass:false,releasePass:false};
}
`;
// Deterministic ledger identity of the seeded fixture report (matches
// learningSuggestionsFromReports: sha256 over the canonical identity array).
const REPLAY_LEARNING_ID=createHash('sha256').update(
  JSON.stringify(['rep-1',1,6540000001,'EXECUTION_SUMMARY'])).digest('hex');
const REPLAY_CONSUMER=String.raw`import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const sha=x=>createHash('sha256').update(x).digest('hex');
const sealed=JSON.parse(readFileSync(process.env.J219_SEALED_PATH,'utf8'));
const selectedC=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-local-codex@v1.0.0');
assert.ok(selectedC&&selectedC.moduleSource,'sealed local-codex capability required');
assert.equal(sha(selectedC.moduleSource),selectedC.moduleSha256);
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selectedC.moduleSource);
assert.ok(digestMatch,'provider digest marker required');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-local-codex-host-bundle'));
assert.equal(sha(readFileSync(providerPath)),digestMatch[1],'PROVIDER_INTEGRITY_MISMATCH');
const sourceSha=process.env.J219_REPLAY_SOURCE_SHA;
const candidateDigest=createHash('sha256').update(JSON.stringify({
  priorRelease:{commitSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
    evidenceRef:'https://github.com/kaicreator-mm/tally/releases/tag/v1.5.5',
    immutable:true,version:'v1.5.5'},
  requirementRef:'https://github.com/kaicreator-mm/tally/issues/92',
  sourceIssueRef:'https://github.com/kaicreator-mm/tally/issues/207',
  subjectSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
  summary:'Post-release behavior regression against immutable v1.5.5'}).toString()).digest('hex');
const issueRef='https://github.com/kaicreator-mm/tally/issues/219';
const learningId=process.env.J219_REPLAY_LEARNING_ID;
const deployment={deployedAppId:'deploy:tally-v2-j219-replay',tenantId:'tenant:tally-v2',
  appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>!!identity&&identity.tenantId===target.tenantId&&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const condition=c=>c&&c.kind==='create'?{mustBeAbsent:true}:
  c&&c.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
  (()=>{throw new Error('INVALID_K_CONDITION');})();
const dbPath=process.env.J219_CONSUMER_TMP+'/kernel.sqlite';
const openFacade=async()=>{
  const sqlite=await openSqliteCasStore({path:dbPath,busyTimeoutMs:1000});
  const facade=await load(sealed,{store:{
    get:key=>sqlite.get(key),
    put:(key,value,cond)=>sqlite.put(key,value,condition(cond)),
    remove:(key,cond)=>sqlite.remove(key,condition(cond))},
    deploymentIdentity:deployment,deploymentRoot:process.cwd(),
    tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.UTC(2026,9,31)});
  return {sqlite,facade};
};
if(process.env.J219_CONSUMER_MODE==='noflag'){
  const {sqlite,facade}=await openFacade();
  for(const intentType of ['publishMaintenanceIssue','recordTaskLearningOutcome']){
    let refused=false,detail='';
    try{
      const receipt=await facade.send({intentType,instanceKey:'maintenance-219',
        messageId:'j219-noflag-'+intentType,
        input:{issueRef,issueNumber:219,sourceSha,candidateDigest,
          learningId,decision:'ADOPTED',outcomeAttested:null,
          outcomeObservedAt:null,outcomeNote:null}});
      detail=JSON.stringify(receipt).slice(0,200);
      refused=receipt.status!=='admitted'||!(receipt.admitted&&
        receipt.admitted.effects&&receipt.admitted.effects.length>0);
    }catch(error){refused=true;detail=String(error&&error.code||error&&error.message||error).slice(0,120);}
    assert.ok(refused,'unselected intent must not be admitted: '+intentType+' '+detail);
  }
  console.log('J219_NOFLAG_ROUTES_ABSENT=PASS');
  facade.stop();await sqlite.close();process.exit(0);
}
const {sqlite,facade}=await openFacade();
const publishReceipt=await facade.send({intentType:'publishMaintenanceIssue',
  instanceKey:'maintenance-219',messageId:'j219-publish-1',
  input:{issueRef,issueNumber:219,sourceSha,candidateDigest}});
assert.equal(publishReceipt.status,'admitted',JSON.stringify(publishReceipt).slice(0,900));
assert.equal(publishReceipt.admitted.effects.length,1);
const publishEffect=publishReceipt.admitted.effects[0];
assert.equal(publishEffect.disposition,'executed');
assert.equal(publishEffect.output.state,'REPLAY_COMPLETE',JSON.stringify(publishEffect.output).slice(0,400));
assert.equal(publishEffect.output.reason,'REPLAY_FIXTURE_ONLY');
assert.equal(publishEffect.output.githubEffect,'NOT_RUN');
assert.equal(publishEffect.output.claimAccepted,true);
assert.equal(publishEffect.output.admitted,true);
assert.equal(publishEffect.output.authorized,false);
assert.equal(publishEffect.output.historicalReleaseMutation,false);
assert.equal(publishEffect.output.taskPass,false);
assert.equal(publishEffect.output.releasePass,false);
// Same-message resend must not admit a second external effect execution.
const publishReplay=await facade.send({intentType:'publishMaintenanceIssue',
  instanceKey:'maintenance-219',messageId:'j219-publish-1',
  input:{issueRef,issueNumber:219,sourceSha,candidateDigest}});
assert.equal(publishReplay.status,'admitted',JSON.stringify(publishReplay).slice(0,400));
if(publishReplay.admitted&&publishReplay.admitted.effects.length>0)
  assert.equal(publishReplay.admitted.effects[0].output.githubEffect,'NOT_RUN');
const decisionInput={issueRef,issueNumber:219,sourceSha,learningId,decision:'ADOPTED',
  outcomeAttested:null,outcomeObservedAt:null,outcomeNote:null};
const decisionReceipt=await facade.send({intentType:'recordTaskLearningOutcome',
  instanceKey:'maintenance-219',messageId:'j219-decision-1',input:{...decisionInput}});
assert.equal(decisionReceipt.status,'admitted',JSON.stringify(decisionReceipt).slice(0,900));
const decisionEffect=decisionReceipt.admitted.effects[0].output;
// Replay mode: the intent is admitted while the physical Host record and the
// GitHub effect stay explicitly NOT_RUN/fixture-only.
assert.equal(decisionEffect.state,'REPLAY_COMPLETE',JSON.stringify(decisionEffect).slice(0,400));
assert.equal(decisionEffect.reason,'REPLAY_FIXTURE_ONLY');
assert.equal(decisionEffect.decision,'ADOPTED');
assert.equal(decisionEffect.measuredEffect,'NOT_MEASURED');
assert.equal(decisionEffect.noAgentRanking,true);
assert.equal(decisionEffect.qualityGainClaimed,false);
assert.equal(decisionEffect.githubEffect,'NOT_RUN');
const outcomeReceipt=await facade.send({intentType:'recordTaskLearningOutcome',
  instanceKey:'maintenance-219',messageId:'j219-outcome-1',
  input:{...decisionInput,outcomeAttested:false,
    outcomeObservedAt:'2026-11-02T10:00:00.000Z',
    outcomeNote:'independent follow-up observed no measurable change'}});
assert.equal(outcomeReceipt.status,'admitted',JSON.stringify(outcomeReceipt).slice(0,900));
const outcomeEffect=outcomeReceipt.admitted.effects[0].output;
assert.equal(outcomeEffect.state,'REPLAY_COMPLETE',JSON.stringify(outcomeEffect).slice(0,400));
assert.equal(outcomeEffect.measuredEffect,'NOT_MEASURED');
assert.equal(outcomeEffect.qualityGainClaimed,false);
assert.equal(outcomeEffect.githubEffect,'NOT_RUN');
const keys=await sqlite.list('');
let docCount=0;
for(const key of keys){
  const row=await sqlite.get(key);
  const text=JSON.stringify(row);
  docCount++;
  assert.ok(text.indexOf(process.env.J219_REPLAY_CANARY)===-1,'secret must not enter K documents');
  assert.ok(text.indexOf('DISPATCH_CLAIMED')===-1,'raw claim body must not enter K documents');
}
assert.ok(docCount>0,'K journal must contain documents');
facade.stop();await sqlite.close();
// Durable journal replay across a full reopen.
const reopenedStore=await openSqliteCasStore({path:dbPath,busyTimeoutMs:1000});
const reopenedFacade=await load(sealed,{store:{
  get:key=>reopenedStore.get(key),
  put:(key,value,cond)=>reopenedStore.put(key,value,condition(cond)),
  remove:(key,cond)=>reopenedStore.remove(key,condition(cond))},
  deploymentIdentity:deployment,deploymentRoot:process.cwd(),
  tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.UTC(2026,9,31)});
const durableReplay=await reopenedFacade.send({intentType:'publishMaintenanceIssue',
  instanceKey:'maintenance-219',messageId:'j219-publish-1',
  input:{issueRef,issueNumber:219,sourceSha,candidateDigest}});
assert.equal(durableReplay.status,'admitted',JSON.stringify(durableReplay).slice(0,400));
// A durable journal replay re-serves the journaled output without a new
// external execution: the provider logs stay at two learning calls and one
// publish call, and every replayed shape keeps the GitHub effect NOT_RUN.
assert.equal(durableReplay.idempotentReplay,true,'durable journal replay expected');
for(const effect of durableReplay.admitted?.effects??[])
  assert.equal(effect.output.githubEffect,'NOT_RUN');
reopenedFacade.stop();await reopenedStore.close();
console.log(JSON.stringify({realKLoad:true,realKSend:publishReceipt.status,
  publishEffect:publishEffect.output.state,publishGithubEffect:'NOT_RUN',
  learningDecision:decisionEffect.state,learningOutcome:outcomeEffect.state,
  measuredEffect:outcomeEffect.measuredEffect,
  durableReplay:durableReplay.status,docCount,
  physicalGithubEffect:'NOT_RUN',authorized:false,taskPass:false,releasePass:false}));
console.log('J219_REPLAY_CONSUMER=PASS');
`;

test('J219 sealed producer + real K consumer replay (physical DH v0.10.3 run)',{
  skip:process.env.J219_REPLAY!=='1'||!process.env.DH_V0103?
    'set J219_REPLAY=1 and DH_V0103=<DH v0.10.3 checkout> to run the physical replay':false,
},()=>{
  const dh=process.env.DH_V0103;
  assert.equal(run('git',['-C',dh,'rev-parse','HEAD']).stdout.trim(),REPLAY_DH_SHA,
    'DH_V0103 must be the exact v0.10.3 checkout pinned by the existing lanes');
  if(!existsSync(join(dh,'tooling','build','dist','index.js'))){
    const build=run('npm',['run','build','-w','@kaicreator/domain-harness-contracts',
      '-w','@kaicreator/domain-harness-build','-w','@kaicreator/domain-harness-microkernel',
      '-w','@kaicreator/domain-harness-host-node'],{cwd:dh,timeout:600000});
    assert.equal(build.status,0,build.stderr.slice(-2000));
  }
  const root=mkdtempSync(join(tmpdir(),'j219-replay-'));
  try{
    const packed=join(root,'packed');mkdirSync(packed);
    for(const name of ['@kaicreator/domain-harness-contracts','@kaicreator/domain-harness-build',
      '@kaicreator/domain-harness-microkernel','@kaicreator/domain-harness-host-node'])
      assert.equal(run('npm',['pack','--pack-destination',packed,'-w',name],{cwd:dh}).status,0,name);
    for(const dir of ['./domain-packages/kernel/basic','./domain-packages/standard/basic'])
      assert.equal(run('npm',['pack','--pack-destination',packed,dir],{cwd:dh}).status,0,dir);
    const packedTarballs=()=>readdirSync(packed);
    assert.ok(packedTarballs().length>=6,'six DH tarballs required, got '+packedTarballs().length);
    const tgz=prefix=>{const f=packedTarballs().find(x=>x.startsWith(prefix));
      assert.ok(f,'missing tarball '+prefix);return join(packed,f);};
    // Replay provider fixture: real host-maintenance.mjs logic, in-memory Host CAS.
    const providerDir=join(root,'provider');mkdirSync(providerDir);
    writeFileSync(join(providerDir,'package.json'),JSON.stringify({name:'@kaicreator/tally-local-codex-host-bundle',
      version:'0.0.0',private:true,type:'module',exports:'./index.mjs'}));
    writeFileSync(join(providerDir,'index.mjs'),
      REPLAY_PROVIDER.replace('__TALLY_ROOT__','file:///'+REPO_ROOT.replace(/\\/g,'/')));
    assert.equal(run('npm',['pack','--pack-destination',packed,providerDir]).status,0,'provider pack');
    const producerDir=join(root,'producer');mkdirSync(producerDir);
    writeFileSync(join(producerDir,'package.json'),JSON.stringify({name:'tally-j219-producer',
      private:true,type:'module'}));
    const producerInstall=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
      '--no-fund',tgz('kaicreator-domain-harness-build-'),tgz('kaicreator-domain-harness-contracts-'),
      tgz('kaicreator-domain-harness-microkernel-'),tgz('kaicreator-domain-harness-host-node-'),
      tgz('kaicreator-domain-kernel-basic-'),tgz('kaicreator-domain-standard-basic-')],
      {cwd:producerDir,env:{...process.env,npm_config_registry:'https://registry.npmmirror.com'},
        timeout:300000});
    assert.equal(producerInstall.status,0,producerInstall.stderr.slice(-2000));
    const businessSource=join(REPO_ROOT,'packages','tally-engineering-business','src','index.mjs');
    // Node resolves bare specifiers from the importing file's location, so
    // run the producer from a copy inside the packed install (run-clean precedent).
    copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build','producer.mjs'),
      join(producerDir,'producer.mjs'));
    copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build','readiness-capability.mjs'),
      join(producerDir,'readiness-capability.mjs'));
    copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build','local-codex-shim.mjs'),
      join(producerDir,'local-codex-shim.mjs'));
    const producerRun=flag=>run(process.execPath,[join(producerDir,'producer.mjs')],
      {cwd:producerDir,env:{...process.env,TALLY_BUSINESS_SOURCE:businessSource,
        T06M_LOCAL:'1',T06M_PROVIDER_FILE:join(providerDir,'index.mjs'),
        T02B_SEALED_APP_OUTPUT:join(root,flag?'sealed-app.json':'sealed-app-noflag.json'),
        ...(flag?{T219_MAINTENANCE:'1'}:{})},timeout:300000});
    const withFlag=producerRun(true);
    assert.equal(withFlag.status,0,withFlag.stderr.slice(-2000)+withFlag.stdout.slice(-2000));
    assert.match(withFlag.stdout,/T02B_PUBLIC_BUILD_SELECTED_B=PASS/);
    assert.match(withFlag.stdout,/"maintenanceRoutes": "SELECTED"/);
    const sealedPath=join(root,'sealed-app.json');
    const sealed=JSON.parse(readFileSync(sealedPath,'utf8'));
    for(const intent of ['publishMaintenanceIssue','recordTaskLearningOutcome']){
      const route=sealed.routes.find(r=>r.intentId===intent);
      assert.ok(route,'maintenance route missing: '+intent);
      assert.equal(route.capabilityId,'tally-local-codex');
    }
    const withoutFlag=producerRun(false);
    assert.equal(withoutFlag.status,0,withoutFlag.stderr.slice(-2000));
    assert.match(withoutFlag.stdout,/"maintenanceRoutes": "NOT_SELECTED"/);
    const noflagPath=join(root,'sealed-app-noflag.json');
    const noflag=JSON.parse(readFileSync(noflagPath,'utf8'));
    assert.equal(noflag.routes.some(r=>['publishMaintenanceIssue',
      'recordTaskLearningOutcome'].includes(r.intentId)),
      false,'maintenance routes must be absent without T219_MAINTENANCE=1');
    assert.ok(!readFileSync(sealedPath,'utf8').includes(REPLAY_CANARY),'no canary in sealed bytes');
    const consumerDir=join(root,'consumer');mkdirSync(consumerDir);
    writeFileSync(join(consumerDir,'package.json'),JSON.stringify({name:'tally-j219-consumer',
      private:true,type:'module'}));
    const consumerInstall=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
      '--no-fund',tgz('kaicreator-domain-harness-contracts-'),tgz('kaicreator-domain-harness-microkernel-'),
      tgz('kaicreator-domain-harness-host-node-'),tgz('kaicreator-tally-local-codex-host-bundle-')],
      {cwd:consumerDir,env:{...process.env,npm_config_registry:'https://registry.npmmirror.com'},
        timeout:600000});
    assert.equal(consumerInstall.status,0,consumerInstall.stderr.slice(-2000));
    // better-sqlite3 needs its install script to fetch the Node24 prebuild.
    const nativeInstall=run('npm',['install','--save-exact','--no-audit','--no-fund',
      'better-sqlite3@12.4.1'],
      {cwd:consumerDir,env:{...process.env,npm_config_registry:'https://registry.npmmirror.com'},
        timeout:600000});
    assert.equal(nativeInstall.status,0,nativeInstall.stderr.slice(-2000));
    writeFileSync(join(consumerDir,'consumer-j219.mjs'),REPLAY_CONSUMER);
    const sourceSha=run('git',['-C',REPO_ROOT,'rev-parse','HEAD']).stdout.trim();
    assert.match(sourceSha,/^[0-9a-f]{40}$/);
    const dataDir=join(root,'consumer-data');mkdirSync(dataDir);
    const consumerEnv=mode=>({...process.env,
      J219_SEALED_PATH:mode==='flag'?sealedPath:noflagPath,
      J219_CONSUMER_MODE:mode,J219_REPLAY_SOURCE_SHA:sourceSha,
      TALLY_CLAIM_SOURCE_SHA:sourceSha,
      J219_REPLAY_LEARNING_ID:REPLAY_LEARNING_ID,
      J219_REPLAY_CANARY:REPLAY_CANARY,
      J219_CONSUMER_TMP:dataDir});
    const flagged=run(process.execPath,[join(consumerDir,'consumer-j219.mjs')],
      {cwd:consumerDir,env:consumerEnv('flag'),timeout:300000});
    assert.equal(flagged.status,0,flagged.stderr.slice(-3000)+flagged.stdout.slice(-3000));
    assert.match(flagged.stdout,/J219_REPLAY_CONSUMER=PASS/);
    const consumerEvidence=JSON.parse(flagged.stdout.trim().split('\n')
      .filter(line=>line.startsWith('{')).at(-1));
    assert.equal(consumerEvidence.realKLoad,true);
    assert.equal(consumerEvidence.physicalGithubEffect,'NOT_RUN');
    const unflagged=run(process.execPath,[join(consumerDir,'consumer-j219.mjs')],
      {cwd:consumerDir,env:consumerEnv('noflag'),timeout:300000});
    assert.equal(unflagged.status,0,unflagged.stderr.slice(-3000));
    assert.match(unflagged.stdout,/J219_NOFLAG_ROUTES_ABSENT=PASS/);
    console.log('J219_PHYSICAL_REPLAY_EVIDENCE='+JSON.stringify({
      producerRoutes:'SELECTED',noFlagRoutes:'ABSENT',publish:'REPLAY_COMPLETE',
      publishGithubEffect:'NOT_RUN',learningDecision:'REPLAY_COMPLETE',
      learningOutcome:'REPLAY_COMPLETE',measuredEffect:'NOT_MEASURED',
      durableReplay:consumerEvidence.durableReplay,
      physicalGithubEffect:'NOT_RUN',dh:REPLAY_DH_SHA}));
  }finally{rmSync(root,{recursive:true,force:true});}
});
