/** Governed Replan candidates (#273 F30). Advisory diff/preservation projection.
 *
 * Derives ONE source-qualified ReplanCandidate bound to the exact repository/
 * version/source SHA, the immutable prior PlanRevision and the observed native
 * Task DAG. Accepted/terminal Task facts are preserved verbatim, unfinished
 * work carries forward traceably and every refusal is a typed reason code.
 * This module never activates anything: activation requires a separate current
 * Owner decision, an accepted Claim and the existing selected K/Host effect
 * (see web/v2/replan-controller.mjs). No native freeze/Claim/Effect authority,
 * no second scheduler, no DAG database, no auto-amend of Product/L2.
 */
import {createHash} from 'node:crypto';

const SHA=/^[0-9a-f]{40}$/,DIGEST=/^[0-9a-f]{64}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const VERSION=/^[A-Za-z0-9][A-Za-z0-9 ._/.-]{0,63}$/;
const ISSUE=/^[1-9][0-9]{0,5}$/;
const ISO=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\//;
const PATH=/^[A-Za-z0-9][A-Za-z0-9 ._/@+-]{0,199}$/;
const array=value=>Array.isArray(value)?value:[];
const freeze=value=>{if(Array.isArray(value))return Object.freeze(value.map(freeze));
  if(value&&typeof value==='object'){const out={};for(const [k,v] of Object.entries(value))out[k]=freeze(v);
    return Object.freeze(out);}return value;};
const unique=value=>[...new Set(value)];
const intersection=(a,b)=>a.filter(x=>b.includes(x));
const terminal=value=>value?.state==='CLOSED'&&['COMPLETED','NOT_PLANNED'].includes(value?.stateReason);
const openTask=value=>value?.state==='OPEN';
const taskNumbers=value=>unique(array(value).map(row=>row?.issueNumber).filter(x=>ISSUE.test(String(x??''))));
const strings=value=>Array.isArray(value)&&value.every(row=>typeof row==='string'&&PATH.test(row));
const numbers=value=>Array.isArray(value)&&value.every(row=>Number.isSafeInteger(row)&&row>0);
const evidenceRows=value=>Array.isArray(value)&&value.every(row=>row&&typeof row==='object'&&!Array.isArray(row)&&
  REF.test(row?.sourceRef??'')&&ISO(row?.observedAt));
const taskRow=value=>value&&typeof value==='object'&&!Array.isArray(value)&&
  Number.isSafeInteger(value.issueNumber)&&value.issueNumber>0&&
  typeof value.title==='string'&&value.title.length>0&&value.title.length<=512&&
  typeof value.state==='string'&&['OPEN','CLOSED'].includes(value.state)&&
  (value.stateReason===undefined||typeof value.stateReason==='string')&&
  numbers(array(value.blockedBy))&&strings(array(value.writeSet));
const proposalRow=value=>taskRow(value)&&(value.reason===undefined||
  (typeof value.reason==='string'&&value.reason.length>0&&value.reason.length<=512));

/** Directed cycle witness over dep -> task edges; returns the cycle path or null. */
function findCycle(tasks){
  const edges=new Map(),state=new Map();
  for(const task of array(tasks))for(const dep of array(task.blockedBy))
    edges.set(dep,[...(edges.get(dep)??[]),task.issueNumber]);
  const visit=(node,stack)=>{
    const mark=state.get(node);
    if(mark==='done')return null;
    if(mark==='active'){const cycle=stack.slice(stack.indexOf(node));
      return cycle.length>1&&cycle[0]===cycle[cycle.length-1]?cycle.slice(0,-1):cycle;}
    state.set(node,'active');
    for(const next of edges.get(node)??[]){
      const cycle=visit(next,[...stack,next]);
      if(cycle)return cycle;}
    state.set(node,'done');return null;};
  for(const node of [...edges.keys()])
    {const cycle=visit(node,[node]);if(cycle)return cycle;}
  return null;
}

/** Deterministic identity digest of one candidate (excluding the digest field itself). */
export function replanCandidateDigest(candidate){
  if(!candidate||typeof candidate!=='object')throw new TypeError('CANDIDATE_REQUIRED');
  const {candidateDigest:omit,...payload}=candidate;
  return createHash('sha256').update(JSON.stringify(payload)).digest('hex');
}

const refusal=(status,reason,detail,extra={})=>freeze({status,reason,detail,candidate:null,
  coverage:{comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false,claimed:false},
  advisoryOnly:true,...extra});
const UNKNOWN=detail=>refusal('UNKNOWN','MALFORMED_INPUT',detail);

const REASONS=Object.freeze(['MALFORMED_INPUT','WRONG_REPO','MOVED_HEAD','OLD_PRODUCT_SHA',
  'STALE_PREDECESSOR','NO_OWNER_DECISION','OWNER_DECISION_REJECTED','NO_NEW_TASK_CLAIM',
  'DUPLICATE_TASK_ID','UNKNOWN_TASK_REFERENCE','CYCLIC_DEPENDENCIES','WRITE_SET_OVERLAP',
  'ACCEPTED_TASK_DROPPED','CARRIED_WORK_LOST']);

/**
 * inputs are observed governance facts, not requested authority:
 *   priorRevision  immutable predecessor PlanRevision (null = no revision yet)
 *   observedDag    native Task DAG readback pinned to the exact sourceSha
 *   proposal       Owner-delivered bounded diff (added/changed/removed/supersession)
 *   ownerDecision  current Owner APPROVED decision bound to the exact sourceSha
 *   claim          accepted PUBLISHED_VERIFIED Claim bound to the exact sourceSha
 *   liveWriteSetClaims  observed competing live Claims for overlap fencing
 */
export function planReplanCandidate(input={}){
  const {repository,versionId,sourceSha,observedAsOf,priorRevision=null,
    observedDag,proposal,ownerDecision,claim,liveWriteSetClaims=[]}=input??{};
  const decisionShape=value=>!value||typeof value!=='object'||Array.isArray(value)||
    REF.test(value.decisionRef??'')===false||!ISO(value.decidedAt??'')||
    !SHA.test(value.sourceSha??'')||!['APPROVED','REJECTED'].includes(value.decision);
  const claimShape=value=>!value||typeof value!=='object'||Array.isArray(value)||
    typeof value.protectedClaimKey!=='string'||!value.protectedClaimKey||
    typeof value.dispatchId!=='string'||!value.dispatchId||
    value.phase!=='PUBLISHED_VERIFIED'||!SHA.test(value.sourceSha??'')||
    !strings(array(value.writeSet))||!DIGEST.test(value.writeSetSha256??'');
  const malformed=typeof repository!=='string'||!REPO.test(repository)||
    typeof versionId!=='string'||!VERSION.test(versionId)||!SHA.test(sourceSha??'')||
    !ISO(observedAsOf)||priorRevision!==null&&(!priorRevision||typeof priorRevision!=='object'||
      Array.isArray(priorRevision)||typeof priorRevision.revisionId!=='string'||
      !priorRevision.revisionId||!SHA.test(priorRevision.sourceSha??'')||!ISO(priorRevision.createdAt??'')||
      !Array.isArray(priorRevision.acceptedTasks)||!priorRevision.acceptedTasks.every(taskRow)||
      !Array.isArray(priorRevision.unfinishedTasks)||!priorRevision.unfinishedTasks.every(taskRow))||
    !observedDag||typeof observedDag!=='object'||Array.isArray(observedDag)||
    !ISO(observedDag.observedAt??'')||!SHA.test(observedDag.sourceSha??'')||
    !Array.isArray(observedDag.tasks)||!observedDag.tasks.every(taskRow)||
    !proposal||typeof proposal!=='object'||Array.isArray(proposal)||typeof proposal.reason!=='string'||
    !proposal.reason||proposal.reason.length>512||!evidenceRows(array(proposal.evidence))||
    !Array.isArray(proposal.added??[])||!array(proposal.added).every(proposalRow)||
    !Array.isArray(proposal.changed??[])||!array(proposal.changed).every(proposalRow)||
    !Array.isArray(proposal.removed??[])||!array(proposal.removed).every(row=>row&&typeof row==='object'&&!Array.isArray(row)&&
      ISSUE.test(String(row?.issueNumber??''))&&typeof row.reason==='string'&&row.reason.length>0&&row.reason.length<=512)||
    proposal.supersessionCandidates!==undefined&&!numbers(array(proposal.supersessionCandidates))||
    ownerDecision!==undefined&&decisionShape(ownerDecision)||
    claim!==undefined&&claimShape(claim)||
    !Array.isArray(liveWriteSetClaims)||!liveWriteSetClaims.every(row=>row&&typeof row==='object'&&!Array.isArray(row)&&
      typeof row.protectedClaimKey==='string'&&strings(array(row.writeSet)));
  if(malformed)return UNKNOWN('replan inputs must be observed, typed governance facts');
  const at=observedAsOf;
  const dagTasks=array(observedDag.tasks),priorAccepted=array(priorRevision?.acceptedTasks),
    priorUnfinished=array(priorRevision?.unfinishedTasks);
  const existingNumbers=new Set([...taskNumbers(dagTasks),...taskNumbers(priorAccepted),
    ...taskNumbers(priorUnfinished)]);
  const added=array(proposal.added),changed=array(proposal.changed),removed=array(proposal.removed);
  const supersession=unique(array(proposal.supersessionCandidates));
  if(observedDag.repository!==undefined&&observedDag.repository!==repository||
     ownerDecision?.repository!==undefined&&ownerDecision.repository!==repository||
     claim?.repository!==undefined&&claim.repository!==repository||
     priorRevision?.repository!==undefined&&priorRevision.repository!==repository)
    return refusal('BLOCKED','WRONG_REPO','foreign source scope or GitHub target refuses before any write',{asOf:at});
  if(claim&&claim.sourceSha!==sourceSha||observedDag.sourceSha!==sourceSha)
    return refusal('STALE','MOVED_HEAD','Claim and DAG observation must bind the exact current source SHA',{asOf:at});
  if(ownerDecision&&ownerDecision.sourceSha!==sourceSha)
    return refusal('STALE','OLD_PRODUCT_SHA','stale reviewed Product/L2/Owner decision refuses activation',{asOf:at});
  if(priorRevision&&priorRevision.sourceSha!==sourceSha)
    return refusal('STALE','STALE_PREDECESSOR','prior revision is not the current-source predecessor; a moved SHA never self-rebases authority',{asOf:at});
  if(ownerDecision&&ownerDecision.decision!=='APPROVED')
    return refusal('BLOCKED','OWNER_DECISION_REJECTED','rejected advisory candidate has no activation and preserves the last accepted baseline',{asOf:at});
  const acceptedTerminal=unique([...priorAccepted,...dagTasks].filter(terminal).map(row=>row.issueNumber));
  const dropped=unique([...removed,...changed].map(row=>row.issueNumber).filter(id=>acceptedTerminal.includes(id)));
  if(dropped.length>0)
    return refusal('BLOCKED','ACCEPTED_TASK_DROPPED','accepted or terminal Task facts cannot be silently deleted or rewritten',
      {asOf:at,witness:{droppedTaskNumbers:dropped}});
  const duplicates=unique([...added.map(row=>row.issueNumber)].filter(id=>existingNumbers.has(id)));
  const withinAdded=added.map(row=>row.issueNumber).filter((id,index,all)=>all.indexOf(id)!==index);
  if(duplicates.length>0||withinAdded.length>0)
    return refusal('BLOCKED','DUPLICATE_TASK_ID','duplicate key, WorkUnit alias or conflicting Task revision is rejected',
      {asOf:at,witness:{duplicateTaskNumbers:unique([...duplicates,...withinAdded])}});
  const removedNumbers=removed.map(row=>row.issueNumber);
  const openExisting=new Set(array(priorUnfinished).filter(openTask).map(row=>row.issueNumber));
  const phantom=unique([...changed.map(row=>row.issueNumber),...removedNumbers]
    .filter(id=>!openExisting.has(id)));
  if(phantom.length>0)
    return refusal('BLOCKED','UNKNOWN_TASK_REFERENCE','changed or removed work must reference an existing unfinished Task',
      {asOf:at,witness:{unknownTaskNumbers:phantom}});
  const mergedTasks=[...dagTasks,...added.map(row=>({...row,state:'OPEN',stateReason:undefined})),
    ...changed.map(row=>({...row,state:'OPEN',stateReason:undefined}))];
  const cycle=findCycle(mergedTasks);
  if(cycle)
    return refusal('BLOCKED','CYCLIC_DEPENDENCIES','cycle is rejected with witness and zero mutation',
      {asOf:at,witness:{cyclePath:cycle}});
  const proposedPaths=unique([...added,...changed].flatMap(row=>array(row.writeSet)));
  const overlaps=array(liveWriteSetClaims)
    .filter(row=>claim===undefined||row.protectedClaimKey!==claim.protectedClaimKey)
    .map(row=>({protectedClaimKey:row.protectedClaimKey,
      shared:intersection(proposedPaths,array(row.writeSet))})).filter(row=>row.shared.length>0);
  if(overlaps.length>0)
    return refusal('BLOCKED','WRITE_SET_OVERLAP','competing live Writer on the same paths is denied until explicitly serialized',
      {asOf:at,witness:{competingClaims:overlaps}});
  const untraceable=removedNumbers.filter(id=>!supersession.includes(id)&&
    !changed.some(row=>row.issueNumber===id));
  if(untraceable.length>0)
    return refusal('BLOCKED','CARRIED_WORK_LOST','removed unfinished work must stay traceable via explicit supersession or change',
      {asOf:at,witness:{untraceableTaskNumbers:untraceable}});
  const droppedNumbers=new Set(removedNumbers);
  const preservedAccepted=array(priorRevision?.acceptedTasks??[]).filter(row=>terminal(row))
    .concat(dagTasks.filter(row=>terminal(row)&&!priorAccepted.some(prior=>prior.issueNumber===row.issueNumber)))
    .filter(row=>!droppedNumbers.has(row.issueNumber));
  const changedNumbers=changed.map(row=>row.issueNumber);
  const carriedUnfinished=priorUnfinished.filter(row=>openTask(row)&&
    !droppedNumbers.has(row.issueNumber)&&!changedNumbers.includes(row.issueNumber));
  const payload=freeze({type:'PlanRevisionCandidate',status:'CANDIDATE',
    predecessorRevisionId:priorRevision?.revisionId??null,
    sourcePins:{repository,versionId,sourceSha,observedAsOf:at},
    reason:proposal.reason,evidence:array(proposal.evidence),
    proposedWork:{added,changed,removed},
    preservedAcceptedTasks:preservedAccepted,
    carriedUnfinishedTasks:carriedUnfinished,
    supersessionCandidates:supersession,
    affectedTaskNumbers:unique([...changedNumbers,...removedNumbers,...added.map(row=>row.issueNumber)]),
    authority:{ownerDecisionBound:ownerDecision!==undefined,
      ownerDecisionRef:ownerDecision?.decisionRef??null,
      ownerDecisionAt:ownerDecision?.decidedAt??null,
      claimBound:claim!==undefined,claimRef:claim?.protectedClaimKey??null,
      claimDispatchId:claim?.dispatchId??null,
      claimWriteSetSha256:claim?.writeSetSha256??null},
    selectedKEffectAuthorityNote:'ADVISORY_ONLY_NO_ACTIVATION_AUTHORITY',
    mustNot:['DO_NOT_TREAT_CANDIDATE_AS_NATIVE_TASK','DO_NOT_DISPATCH_AGENT_FROM_CANDIDATE',
      'DO_NOT_DELETE_NATIVE_BLOCKED_BY','DO_NOT_REWRITE_ACCEPTED_TASK_FACTS',
      'DO_NOT_AUTO_AMEND_PRODUCT_OR_L2']});
  const candidateDigest=replanCandidateDigest(payload);
  const candidate=freeze({...payload,candidateDigest});
  return freeze({status:'CANDIDATE',reason:null,detail:null,candidate,
    coverage:{comparison:'COUNTS_NOT_PERCENTAGES',addedCount:added.length,
      changedCount:changed.length,removedCount:removed.length,
      preservedAcceptedCount:preservedAccepted.length,carriedUnfinishedCount:carriedUnfinished.length,
      supersessionCount:supersession.length,ownerDecisionBound:ownerDecision!==undefined,
      claimBound:claim!==undefined,taskPass:false,releasePass:false,claimed:false},
    advisoryOnly:true});
}

export const REPLAN_REASON_CODES=REASONS;
