import test from 'node:test';
import assert from 'node:assert/strict';
import {createReplanController,__test}
  from './replan-controller.mjs';

const SHA='c1523a23a3429649fc7e87caa189e61bcf483668',OTHER_SHA='b'.repeat(40);
const AS_OF='2026-11-19T00:00:00.000Z';
const REPO='kaicreator-mm/tally';
const ref=n=>`https://github.com/${REPO}/issues/${n}`;
const task=(issueNumber,extra={})=>({issueNumber,title:`Task ${issueNumber}`,
  state:'OPEN',stateReason:undefined,blockedBy:[],writeSet:[],...extra});
const WRITE_SET=['packages/tally-engineering-business/src/replan.mjs',
  'web/v2/replan-controller.mjs'].sort();
const claim=()=>({repository:REPO,protectedClaimKey:`${REPO}#273:builder:__default__`,
  dispatchId:'dispatch:273:f30:host:20261119:1',sourceSha:SHA,phase:'PUBLISHED_VERIFIED',
  writeSet:WRITE_SET,writeSetSha256:'8'.repeat(64)});
const ownerDecision=()=>({repository:REPO,
  decisionRef:`${ref(273)}#issuecomment-6994136047`,
  decidedAt:'2026-11-18T22:59:47.000Z',sourceSha:SHA,decision:'APPROVED'});
const proposal=(extra={})=>({reason:'Unblock #264 integration with a governed replan',
  evidence:[{sourceRef:ref(273),observedAt:AS_OF}],
  added:[task(300,{title:'New bounded work',writeSet:['web/v2/replan-extra.mjs']})],
  changed:[],removed:[],...extra});
const observedDag=()=>({repository:REPO,observedAt:AS_OF,sourceSha:SHA,tasks:[
  task(219,{state:'CLOSED',stateReason:'COMPLETED'}),
  task(276,{state:'CLOSED',stateReason:'NOT_PLANNED'}),
  task(273,{blockedBy:[219]})]});
const priorRevision=()=>({revisionId:'planrevision:version/v2:000000:baseline',
  sourceSha:SHA,createdAt:'2026-11-01T00:00:00.000Z',
  acceptedTasks:[task(219,{state:'CLOSED',stateReason:'COMPLETED'})],
  unfinishedTasks:[task(273)]});
const deriveArgs=(extra={})=>({priorRevision:priorRevision(),observedDag:observedDag(),
  proposal:proposal(),ownerDecision:ownerDecision(),claim:claim(),
  liveWriteSetClaims:[],...extra});

/** Generic durable document store with true revision semantics (CasDocumentStore). */
function makeCasStore({docs=new Map(),putHook=null}={}){
  return {
    async get(key){const d=docs.get(key);
      return d===undefined?null:{value:d.value,revision:d.revision};},
    async put(key,value,conditions={}){
      if(putHook)await putHook(key,value,conditions);
      if(conditions.mustBeAbsent===true&&docs.has(key))
        throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
      if(conditions.mustMatchRevision!==undefined&&
         docs.get(key)?.revision!==conditions.mustMatchRevision)
        throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
      const revision=docs.has(key)?docs.get(key).revision+1:0;
      docs.set(key,{value,revision});return {revision};},
    async list(prefix){return [...docs.keys()].filter(key=>key.startsWith(prefix));},
    async close(){}};
}
const make=(extraConfig={},seams={})=>createReplanController(
  {currentSourceSha:SHA,author:'builder:i273',...extraConfig},seams);
const activate=(controller,args)=>controller.dispatch(SHA,'activate-revision',args);

test('constructor refuses untrusted configuration',()=>{
  assert.throws(()=>createReplanController({}),TypeError);
  assert.throws(()=>createReplanController({currentSourceSha:'nothex'}),
    TypeError);
  assert.throws(()=>make({},{openStore:'not-a-function'}),TypeError);
  assert.throws(()=>make({planRevisionAppendContract:'yes'}),TypeError);
});
test('P3 approved current candidate activates exactly one immutable revision',async()=>{
  const docs=new Map();
  const store=makeCasStore({docs});
  const controller=make({},{openStore:async()=>store});
  const derived=await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  assert.equal(derived.status,'OK');
  const reviewed=await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  assert.equal(reviewed.status,'OK');
  const activated=await activate(controller,{registryPath:'unused-by-fake'});
  assert.equal(activated.status,'OK');
  assert.match(activated.revisionId,/^planrevision:version\/v2:000001:[0-9a-f]{8}$/);
  assert.equal(activated.selectedKEffect.state,'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT');
  assert.equal(activated.appendOnly,true);
  assert.equal(activated.preservedAcceptedCount,2);
  assert.equal(activated.carriedUnfinishedCount,1);
  const keys=[...docs.keys()];
  assert.equal(keys.length,1);
  const revision=docs.get(keys[0]).value;
  assert.equal(revision.type,'PlanRevision');
  assert.equal(revision.appendOnly,true);
  assert.equal(revision.immutable,true);
  assert.equal(revision.predecessorRevisionId,'planrevision:version/v2:000000:baseline');
  assert.deepEqual(revision.preservedAcceptedTasks.map(row=>row.issueNumber).sort(),[219,276]);
  assert.deepEqual(revision.carriedUnfinishedTasks.map(row=>row.issueNumber),[273]);
  assert.equal(revision.selectedKEffect.state,'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT');
});
test('idempotent replay returns the same revision without a duplicate append',async()=>{
  const docs=new Map();
  const controller=make({},{openStore:async()=>makeCasStore({docs})});
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const first=await activate(controller,{});
  assert.equal(first.status,'OK');
  const second=await activate(controller,{});
  assert.equal(second.status,'OK');
  assert.equal(second.alreadyActivated,true);
  assert.equal(second.revisionId,first.revisionId);
  assert.equal(docs.size,1);
});
test('R16 RESTART_REPLAY history rehydrates from the K store without duplicate effects',async()=>{
  const docs=new Map();
  const openStore=async()=>makeCasStore({docs});
  const first=make({},{openStore});
  first.dispatch(SHA,'derive-candidate',deriveArgs());
  first.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const activated=await activate(first,{});
  assert.equal(activated.status,'OK');
  // A fresh controller process over the same persistent store.
  const restarted=make({},{openStore});
  const replayed=await activate(restarted,{});
  // No candidate derived in the restarted controller: honest refusal, not a re-effect.
  assert.equal(replayed.status,'BLOCKED');
  assert.equal(replayed.reason,'NO_CANDIDATE_TO_ACTIVATE');
  const sameCandidate=make({},{openStore});
  await sameCandidate.dispatch(SHA,'derive-candidate',deriveArgs());
  await sameCandidate.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const reactivated=await activate(sameCandidate,{});
  assert.equal(reactivated.status,'OK');
  assert.equal(reactivated.alreadyActivated,true);
  assert.equal(reactivated.revisionId,activated.revisionId);
  assert.equal(docs.size,1);
  const readback=await sameCandidate.dispatch(SHA,'readback',{});
  assert.equal(readback.status,'OK');
  assert.equal(readback.count,1);
  assert.equal(readback.chainLinked,true);
  assert.equal(readback.source,'EXISTING_K_STORE_READBACK');
  assert.equal(readback.history[0].revisionId,activated.revisionId);
});
test('readback of an empty store is a truthful zero-history observation',async()=>{
  const controller=make({},{openStore:async()=>makeCasStore()});
  const readback=await controller.dispatch(SHA,'readback',{});
  assert.equal(readback.status,'OK');
  assert.equal(readback.count,0);
});
test('R07/R08 activation without Owner decision or accepted Claim refuses before any effect',async()=>{
  const docs=new Map();
  const controller=make({},{openStore:async()=>makeCasStore({docs})});
  const noOwner=await controller.dispatch(SHA,'derive-candidate',deriveArgs({ownerDecision:undefined}));
  assert.equal(noOwner.status,'OK'); // R07: the advisory candidate still renders
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const blockedOwner=await activate(controller,{});
  assert.equal(blockedOwner.status,'BLOCKED');
  assert.equal(blockedOwner.reason,'NO_OWNER_DECISION');
  const noClaim=await controller.dispatch(SHA,'derive-candidate',deriveArgs({claim:undefined}));
  assert.equal(noClaim.status,'OK'); // R08: renders without claim binding
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const blockedClaim=await activate(controller,{});
  assert.equal(blockedClaim.status,'BLOCKED');
  assert.equal(blockedClaim.reason,'NO_NEW_TASK_CLAIM');
  assert.equal(docs.size,0);
});
test('review admission: independence, presence and staleness gate activation',async()=>{
  const controller=make({});
  const noCandidate=await activate(controller,{});
  assert.equal(noCandidate.reason,'NO_CANDIDATE_TO_ACTIVATE');
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  const noReview=await activate(controller,{});
  assert.equal(noReview.reason,'REVIEW_REQUIRED');
  const selfReview=await controller.dispatch(SHA,'review-candidate',{reviewer:'builder:i273'});
  assert.equal(selfReview.status,'BLOCKED');
  assert.equal(selfReview.reason,'REVIEW_NOT_INDEPENDENT');
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const changedProposal=await controller.dispatch(SHA,'derive-candidate',
    deriveArgs({proposal:proposal({reason:'Different bounded reason'})}));
  assert.equal(changedProposal.status,'OK');
  const staleReview=await activate(controller,{});
  assert.equal(staleReview.reason,'REVIEW_REQUIRED');
});
test('R12 CAS_CONFLICT out-of-order create fails closed without overwrite',async()=>{
  const docs=new Map();
  let refused=false;
  const store=makeCasStore({docs,putHook:async()=>{
    if(refused)throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
  }});
  const controller=make({},{openStore:async()=>store});
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const first=await activate(controller,{});
  assert.equal(first.status,'OK');
  // A different candidate colliding on the same sequence must not overwrite.
  const competitor=make({},{openStore:async()=>store});
  await competitor.dispatch(SHA,'derive-candidate',deriveArgs({
    proposal:proposal({reason:'A competing bounded proposal'})}));
  await competitor.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  refused=true;
  const result=await activate(competitor,{});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.reason,'CAS_CONFLICT');
  const keys=[...docs.keys()];
  assert.equal(keys.length,1);
  assert.equal(docs.get(keys[0]).value.activationDigest,
    controller.snapshot().activation.activationDigest);
});
test('R13 ACK_LOST reconciles by exact readback and never duplicates',async()=>{
  const docs=new Map();
  let loseAck=false;
  const store=makeCasStore({docs,putHook:async(key,value)=>{
    if(loseAck){docs.set(key,{value,revision:0});
      throw new Error('connection reset after write');}
  }});
  const controller=make({},{openStore:async()=>store});
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  loseAck=true;
  const uncertain=await activate(controller,{});
  assert.equal(uncertain.status,'OK');
  assert.equal(uncertain.reconciled,true);
  assert.equal(docs.size,1);
});
test('R13 ACK_LOST without a landed write stays UNKNOWN and never blind-retries',async()=>{
  const docs=new Map();
  const store=makeCasStore({docs,putHook:async()=>{
    throw new Error('connection reset before write');}});
  const controller=make({},{openStore:async()=>store});
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const uncertain=await activate(controller,{});
  assert.equal(uncertain.status,'UNKNOWN');
  assert.equal(uncertain.reason,'ACK_LOST');
  assert.equal(docs.size,0);
  const readback=await controller.dispatch(SHA,'readback',{});
  assert.equal(readback.count,0);
});
test('K_STORE_UNAVAILABLE activation and readback stay honestly NOT_RUN',async()=>{
  const controller=make({});
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const result=await activate(controller,{});
  assert.equal(result.status,'NOT_RUN');
  assert.equal(result.reason,'K_STORE_UNAVAILABLE');
  const readback=await controller.dispatch(SHA,'readback',{});
  assert.equal(readback.status,'NOT_RUN');
  assert.equal(readback.reason,'K_STORE_UNAVAILABLE');
});
test('declared K contract: the ONE selected seam is invoked and its truth recorded',async()=>{
  const docs=new Map();
  const seamCalls=[];
  const controller=make({planRevisionAppendContract:true},{
    openStore:async()=>makeCasStore({docs}),
    sealedDispatch:async envelope=>{seamCalls.push(envelope);
      return {phase:'terminal',outcome:{ok:true,effectId:'effect:k-1'}};}});
  await controller.dispatch(SHA,'derive-candidate',deriveArgs());
  await controller.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const result=await activate(controller,{});
  assert.equal(result.status,'OK');
  assert.equal(seamCalls.length,1);
  assert.equal(seamCalls[0].intentType,'appendPlanRevision');
  assert.equal(result.selectedKEffect.state,'APPLIED_VIA_SELECTED_K');
  const refusing=make({planRevisionAppendContract:true},{
    openStore:async()=>makeCasStore(),
    sealedDispatch:async()=>({phase:'refused',reason:'K_CONTRACT_REFUSED'})});
  await refusing.dispatch(SHA,'derive-candidate',deriveArgs());
  await refusing.dispatch(SHA,'review-candidate',{reviewer:'independent:reviewer'});
  const refused=await activate(refusing,{});
  assert.equal(refused.status,'OK');
  assert.equal(refused.selectedKEffect.state,'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT');
  assert.equal(refused.selectedKEffect.reason,'K_CONTRACT_REFUSED');
});
test('stale source refusals and truthful snapshot surface',async()=>{
  const controller=make({});
  const staleDispatch=await controller.dispatch(OTHER_SHA,'derive-candidate',deriveArgs());
  assert.equal(staleDispatch.status,'STALE_SOURCE');
  const denied=await controller.dispatch(SHA,'unknown-action',{});
  assert.equal(denied.status,'DENIED');
  const snapshot=controller.snapshot();
  assert.equal(snapshot.status,'OBSERVED');
  assert.equal(snapshot.constraints.writeSet.length,9);
  assert.equal(snapshot.constraints.writeSetSha256,
    '845cb2618eebab4f56aa38bd6c0176b2e6b3499d4a01db18150d02f9915d85dd');
  assert.equal(snapshot.constraints.pinnedClaim,`${REPO}#273:builder:__default__`);
  assert.equal(snapshot.constraints.pinnedDispatch,'dispatch:273:f30:host:20261119:1');
  assert.equal(snapshot.constraints.selectedKEffect,'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT');
  assert.equal(snapshot.authorized,false);
  assert.equal(snapshot.admitted,false);
  assert.equal(snapshot.taskPass,false);
  assert.equal(snapshot.releasePass,false);
  assert.equal(snapshot.nativeWriteAuthorized,false);
  assert.equal(snapshot.historicalRevisionMutation,false);
  assert.equal(snapshot.candidate,null);
});
test('__test pins expose the frozen workbench identity',()=>{
  assert.equal(__test.REPOSITORY,'kaicreator-mm/tally');
  assert.deepEqual(__test.WRITE_SET.length,9);
  assert.equal(__test.PINNED_DISPATCH,'dispatch:273:f30:host:20261119:1');
});
