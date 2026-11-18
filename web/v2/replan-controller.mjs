/** F30 governed Replan workbench controller (#273). Candidates stay advisory;
 * the Host owns the one controlled activation.
 *
 * Derivation flows through the selected Business projection (planReplanCandidate).
 * The only privileged effect is ONE append-only, CAS/idempotent and replay-safe
 * PlanRevision append into the existing selected K CAS store (injected openStore
 * — the same Host CAS store family the claim/FSW/version-DAG registries use;
 * never a second scheduler, DAG database or direct GitHub writer). When the
 * trusted Host explicitly declares the versioned PlanRevision append contract,
 * the ONE existing selected B→K→Host sealed effect is invoked; otherwise the
 * honest state is BLOCKED_MISSING_SELECTED_REPLAN_EFFECT — never a fabricated
 * PASS. Lost acknowledgments stay UNKNOWN and reconcile by exact readback,
 * never by blind retry. This module never touches a provider credential.
 */
import {planReplanCandidate} from
  '../../packages/tally-engineering-business/src/replan.mjs';

const SHA=/^[0-9a-f]{40}$/,DIGEST=/^[0-9a-f]{64}$/;
const OWNER='kaicreator-mm',REPO='tally',ISSUE=273;
const REPOSITORY=`${OWNER}/${REPO}`;
const VERSION_ID='version/v2';
const JOURNEY_ID='tally-v2-replan-f30';
const ISSUE_REF=`https://github.com/${REPOSITORY}/issues/${ISSUE}`;
const IDENTITY=/^[A-Za-z0-9][A-Za-z0-9 .:_'@-]{0,127}$/;
const REVISION_PREFIX=`tally:v2:plan-revision:${VERSION_ID}:`;
// Exact frozen write set pinned by the accepted Claim on issue #273 (9 paths).
const WRITE_SET=Object.freeze([
  'apps/tally-web/src/features/replan/model.test.ts',
  'apps/tally-web/src/features/replan/model.ts',
  'apps/tally-web/src/features/replan/views.tsx',
  'packages/tally-engineering-business/src/index.mjs',
  'packages/tally-engineering-business/src/replan.mjs',
  'packages/tally-engineering-business/test/business-policy.test.mjs',
  'packages/tally-engineering-business/test/replan.test.mjs',
  'web/v2/replan-controller.mjs','web/v2/replan-controller.test.mjs']);
const WRITE_SET_SHA256='845cb2618eebab4f56aa38bd6c0176b2e6b3499d4a01db18150d02f9915d85dd';
const PINNED_CLAIM=`${REPOSITORY}#${ISSUE}:builder:__default__`;
const PINNED_DISPATCH='dispatch:273:f30:host:20261119:1';
const blocked=reason=>({status:'BLOCKED',reason});
const arrayOf=value=>Array.isArray(value)?value:[];
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const pad=seq=>String(seq).padStart(6,'0');

export function createReplanController(config,{host,sealedDispatch,openStore,now}={}){
  if(!SHA.test(config?.currentSourceSha??''))throw new TypeError('CURRENT_SOURCE_SHA_REQUIRED');
  if(config.author!==undefined&&(typeof config.author!=='string'||!IDENTITY.test(config.author)))
    throw new TypeError('AUTHOR_IDENTITY_INVALID');
  if(config.planRevisionAppendContract!==undefined&&config.planRevisionAppendContract!==true)
    throw new TypeError('PLAN_REVISION_APPEND_CONTRACT_MUST_BE_BOOLEAN_TRUE');
  if(openStore!==undefined&&typeof openStore!=='function')
    throw new TypeError('OPEN_STORE_MUST_BE_FUNCTION');
  const author=config.author??'workbench:operator';
  const currentSha=()=>config.currentSourceSha;
  const clock=now??(()=>new Date().toISOString());
  const contractDeclared=()=>config.planRevisionAppendContract===true;
  const selectedKEffectDefault=()=>contractDeclared()&&typeof sealedDispatch==='function'?
    'SELECTED_K_CONTRACT_DECLARED':'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT';
  const state={candidate:null,digest:null,review:null,activation:null,history:[]};
  const record=(actionKind,result)=>{state.history.push({action:actionKind,
    at:clock(),...result});return result;};
  const stale=sourceSha=>SHA.test(sourceSha??'')&&sourceSha!==currentSha();
  /** Advisory F30 projection of one governed replan proposal. The as-of anchor
   * is the observed DAG readback (never the wall clock), so re-deriving the
   * same observed facts yields the same candidate identity and the replay
   * readback stays deterministic. */
  function deriveCandidate(sourceSha,args){
    const {priorRevision=null,observedDag,proposal,ownerDecision,claim,
      liveWriteSetClaims=[]}=args??{};
    const built=planReplanCandidate({repository:REPOSITORY,versionId:VERSION_ID,
      sourceSha:currentSha(),observedAsOf:observedDag?.observedAt??clock(),
      priorRevision,observedDag,
      proposal,ownerDecision,claim,liveWriteSetClaims});
    if(built.status!=='CANDIDATE')
      return record('derive-candidate',{status:built.status,reason:built.reason,
        detail:built.detail,witness:built.witness??null,candidate:null,
        historicalMutation:false});
    state.candidate={input:args??{},built:built.candidate,at:clock()};
    state.digest=built.candidate.candidateDigest;
    // A new draft invalidates a stale review and any activation state.
    if(state.review&&state.review.digest!==state.digest)state.review=null;
    state.activation=null;
    return record('derive-candidate',{status:'OK',digest:state.digest,
      counts:built.coverage,advisoryOnly:true});
  }
  function reviewCandidate(sourceSha,{reviewer}={}){
    if(!state.candidate||!state.digest)
      return record('review-candidate',blocked('NO_CANDIDATE_TO_REVIEW'));
    if(typeof reviewer!=='string'||!IDENTITY.test(reviewer))
      return record('review-candidate',blocked('REVIEWER_IDENTITY_INVALID'));
    if(reviewer===author)
      return record('review-candidate',blocked('REVIEW_NOT_INDEPENDENT'));
    state.review={reviewer,at:clock(),digest:state.digest,sourceSha:currentSha(),
      state:'REVIEWED',independence:'REVIEWER_DIFFERS_FROM_AUTHOR'};
    return record('review-candidate',{status:'OK',reviewer,digest:state.digest,
      note:'Activation stays Host-owned; a review never activates anything'});
  }
  /** R98: the ONE existing selected B→K→Host effect is invoked only when the
   * trusted Host declares the versioned PlanRevision append contract; a
   * refusal or an uncertain envelope stays truthful and never becomes PASS. */
  async function evaluateSelectedKEffect(activationDigest,candidate,review){
    if(!contractDeclared()||typeof sealedDispatch!=='function')
      return {state:'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT',reason:'NO_DECLARED_CONTRACT'};
    let envelope=null;
    try{
      envelope=await sealedDispatch({intentType:'appendPlanRevision',
        input:{activationDigest,candidate,reviewer:review.reviewer,
          reviewedAt:review.at}});
    }catch{
      return {state:'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT',reason:'K_SEAM_REFUSED'};
    }
    if(envelope?.phase==='terminal'&&envelope.outcome?.ok===true)
      return {state:'APPLIED_VIA_SELECTED_K',effectId:envelope.outcome.effectId??null};
    if(envelope?.phase==='unknown')
      return {state:'UNKNOWN_SELECTED_K_ACK',reason:'ACK_LOST_REQUIRES_RECONCILIATION'};
    return {state:'BLOCKED_MISSING_SELECTED_REPLAN_EFFECT',
      reason:envelope?.outcome?.reason??envelope?.reason??'K_CONTRACT_REFUSED'};
  }
  function revisionDocument({seq,activationDigest,candidate,review,note,selectedK}){
    return {type:'PlanRevision',schema:'tally/v2-plan-revision/1',
      revisionId:`planrevision:${VERSION_ID}:${pad(seq)}:${activationDigest.slice(0,8)}`,
      predecessorRevisionId:candidate.predecessorRevisionId,
      sourcePins:{...candidate.sourcePins},activationDigest,
      reason:candidate.reason,evidence:candidate.evidence,
      proposedWork:candidate.proposedWork,
      preservedAcceptedTasks:candidate.preservedAcceptedTasks,
      carriedUnfinishedTasks:candidate.carriedUnfinishedTasks,
      supersessionCandidates:candidate.supersessionCandidates,
      affectedTaskNumbers:candidate.affectedTaskNumbers,
      authority:{...candidate.authority,independentReviewRef:review.reviewer,
        reviewedAt:review.at,activationNote:note??null},
      selectedKEffect:selectedK,
      appendOnly:true,immutable:true,createdAt:clock()};
  }
  /** Exact replay readback: an existing revision for this digest is returned, never duplicated. */
  async function findRevisionByDigest(store,digest){
    const keys=await store.list(REVISION_PREFIX);
    for(const key of keys){
      const row=await store.get(key).catch(()=>null);
      if(plain(row?.value)&&row.value.activationDigest===digest)
        return {key,revisionId:row.value.revisionId,revision:row.value};
    }
    return null;
  }
  async function activateRevision(sourceSha,args){
    if(!state.candidate||!state.digest)
      return record('activate-revision',blocked('NO_CANDIDATE_TO_ACTIVATE'));
    if(!state.review||state.review.digest!==state.digest)
      return record('activate-revision',blocked(state.review?'REVIEW_STALE':'REVIEW_REQUIRED'));
    if(state.review.sourceSha!==currentSha())
      return record('activate-revision',blocked('REVIEW_SOURCE_STALE'));
    const candidate=state.candidate.built;
    if(candidate?.authority?.ownerDecisionBound!==true)
      return record('activate-revision',blocked('NO_OWNER_DECISION'));
    if(candidate?.authority?.claimBound!==true)
      return record('activate-revision',blocked('NO_NEW_TASK_CLAIM'));
    const activationDigest=state.digest;
    if(state.activation&&state.activation.activationDigest===activationDigest&&
       state.activation.state==='ACTIVATED')
      return record('activate-revision',{status:'OK',alreadyActivated:true,
        revisionId:state.activation.revisionId,
        selectedKEffect:state.activation.selectedKEffect});
    if(typeof openStore!=='function')
      return record('activate-revision',{status:'NOT_RUN',reason:'K_STORE_UNAVAILABLE',
        activationDigest,selectedKEffect:selectedKEffectDefault()});
    let store=null;
    try{
      store=await openStore({path:args?.registryPath,busyTimeoutMs:1000});
      // Exact replay readback first: retries never blind-duplicate an effect.
      const existing=await findRevisionByDigest(store,activationDigest);
      if(existing){
        state.activation={state:'ACTIVATED',activationDigest,
          revisionId:existing.revisionId,selectedKEffect:existing.revision.selectedKEffect,
          reconciled:true};
        return record('activate-revision',{status:'OK',alreadyActivated:true,
          revisionId:existing.revisionId,selectedKEffect:existing.revision.selectedKEffect});
      }
      const keys=await store.list(REVISION_PREFIX);
      const seq=keys.length+1;
      const selectedK=await evaluateSelectedKEffect(activationDigest,candidate,state.review);
      const document=revisionDocument({seq,activationDigest,candidate,
        review:state.review,note:args?.note??null,selectedK});
      const key=`${REVISION_PREFIX}${pad(seq)}:${activationDigest.slice(0,8)}`;
      try{
        await store.put(key,document,{mustBeAbsent:true});
      }catch(error){
        // Uncertain append: reconcile by exact readback, never a blind retry.
        const landed=await store.get(key).catch(()=>null);
        if(plain(landed?.value)&&landed.value.activationDigest===activationDigest){
          state.activation={state:'ACTIVATED',activationDigest,
            revisionId:landed.value.revisionId,
            selectedKEffect:landed.value.selectedKEffect,reconciled:true};
          return record('activate-revision',{status:'OK',reconciled:true,
            revisionId:landed.value.revisionId,
            selectedKEffect:landed.value.selectedKEffect});
        }
        const conflict=error?.code==='HOST_CAS_CONFLICT'||
          String(error?.message??error).includes('CAS');
        state.activation={state:conflict?'BLOCKED':'UNKNOWN',activationDigest,
          revisionId:document.revisionId,reason:conflict?'CAS_CONFLICT':'ACK_LOST'};
        return record('activate-revision',conflict?blocked('CAS_CONFLICT'):
          {status:'UNKNOWN',reason:'ACK_LOST',activationDigest,
            revisionId:document.revisionId,
            detail:'uncertain write reconciles only via exact CAS readback'});
      }
      state.activation={state:'ACTIVATED',activationDigest,revisionId:document.revisionId,
        selectedKEffect:document.selectedKEffect};
      return record('activate-revision',{status:'OK',revisionId:document.revisionId,
        selectedKEffect:document.selectedKEffect,appendOnly:true,
        preservedAcceptedCount:candidate.preservedAcceptedTasks.length,
        carriedUnfinishedCount:candidate.carriedUnfinishedTasks.length});
    }catch{
      state.activation={state:'NOT_RUN',reason:'K_STORE_UNAVAILABLE',activationDigest};
      return record('activate-revision',{status:'NOT_RUN',reason:'K_STORE_UNAVAILABLE'});
    }finally{
      if(store)await store.close().catch(()=>{});
    }
  }
  /** Canonical post-restart readback: history is reconstructed from the K store only. */
  async function readbackRevisionHistory(sourceSha,args={}){
    if(typeof openStore!=='function')
      return record('readback',{status:'NOT_RUN',reason:'K_STORE_UNAVAILABLE'});
    let store=null;
    try{
      store=await openStore({path:args?.registryPath,busyTimeoutMs:1000});
      const keys=await store.list(REVISION_PREFIX);
      const rows=[];
      for(const key of keys){
        const row=await store.get(key).catch(()=>null);
        if(plain(row?.value))rows.push({revision:row.value,physicalRevision:row.revision});
      }
      rows.sort((left,right)=>String(left.revision.revisionId)
        .localeCompare(String(right.revision.revisionId)));
      const history=rows.map(row=>({revisionId:row.revision.revisionId,
        predecessorRevisionId:row.revision.predecessorRevisionId,
        activationDigest:row.revision.activationDigest,
        sourcePins:row.revision.sourcePins,
        preservedAcceptedCount:arrayOf(row.revision.preservedAcceptedTasks).length,
        carriedUnfinishedCount:arrayOf(row.revision.carriedUnfinishedTasks).length,
        selectedKEffect:row.revision.selectedKEffect,createdAt:row.revision.createdAt,
        appendOnly:row.revision.appendOnly===true,immutable:row.revision.immutable===true}));
      const linked=history.every((row,index)=>index===0?
        true:row.predecessorRevisionId===history[index-1].revisionId);
      return record('readback',{status:'OK',observedAt:clock(),count:history.length,
        chainLinked:history.length<2?true:linked,history,
        source:'EXISTING_K_STORE_READBACK'});
    }catch{
      return record('readback',{status:'NOT_RUN',reason:'K_STORE_READ_FAILED'});
    }finally{
      if(store)await store.close().catch(()=>{});
    }
  }
  const actions={
    'derive-candidate':deriveCandidate,
    'review-candidate':reviewCandidate,
    'activate-revision':activateRevision,
    'readback':readbackRevisionHistory,
  };
  function snapshot(){
    const observedAt=clock();
    const candidate=state.candidate?{state:state.review?'REVIEWED':'DRAFT',
      digest:DIGEST.test(state.digest??'')?state.digest:null,
      candidateDigest:state.digest,
      predecessorRevisionId:state.candidate.built.predecessorRevisionId,
      reason:state.candidate.built.reason,
      proposedWork:state.candidate.built.proposedWork,
      preservedAcceptedTasks:state.candidate.built.preservedAcceptedTasks,
      carriedUnfinishedTasks:state.candidate.built.carriedUnfinishedTasks,
      supersessionCandidates:state.candidate.built.supersessionCandidates,
      affectedTaskNumbers:state.candidate.built.affectedTaskNumbers,
      authority:state.candidate.built.authority,
      advisoryOnly:state.candidate.built.advisoryOnly===true}:null;
    const activation=state.activation?{...state.activation}:{state:'NOT_RUN',
      reason:'ACTIVATION_REQUIRED',selectedKEffect:selectedKEffectDefault()};
    return {status:'OBSERVED',project:OWNER,version:'v2',journeyId:JOURNEY_ID,
      sourceSha:currentSha(),observedAsOf:observedAt,
      currentness:'OBSERVED_AT_QUERY_COMPLETION',
      candidate,review:state.review?{state:'REVIEWED',reviewer:state.review.reviewer,
        at:state.review.at,digest:state.review.digest,
        current:state.review.digest===state.digest,
        independence:state.review.independence}:null,
      activation,
      constraints:{writeSet:[...WRITE_SET],writeSetCount:WRITE_SET.length,
        writeSetSha256:WRITE_SET_SHA256,
        pinnedClaim:PINNED_CLAIM,pinnedDispatch:PINNED_DISPATCH,
        sourceSha:currentSha(),selectedKEffect:selectedKEffectDefault(),
        historicalRevisionMutation:false,nativeWriteAuthorized:false},
      authorized:false,admitted:false,taskPass:false,releasePass:false,
      historicalRevisionMutation:false,nativeWriteAuthorized:false};
  }
  async function dispatch(sourceSha,kind,args){
    const action=actions[kind];
    if(!action)return {status:'DENIED',reason:'UNKNOWN_ACTION'};
    if(stale(sourceSha))return {status:'STALE_SOURCE'};
    return action(sourceSha,args??{});
  }
  return Object.freeze({snapshot,dispatch,history:()=>[...state.history]});
}

export const __test={WRITE_SET,WRITE_SET_SHA256,PINNED_CLAIM,PINNED_DISPATCH,
  REPOSITORY,REVISION_PREFIX,ISSUE_REF,JOURNEY_ID};
