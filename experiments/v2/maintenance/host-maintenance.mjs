/**
 * J08b/J08c Host-only maintenance effects (#219).
 *
 * The Host owns every privileged step: accepted-Claim verification, frozen
 * write-set verification, the frozen independent candidate review (Host CAS),
 * the authorized native Issue publication with canonical readback and the
 * governed Task Learning provenance/use/outcome ledger. Transports are
 * injected by the sealed provider; this module holds no credential and never
 * trusts caller authority. The prior release is an immutable reported
 * identity only — it is never rewritten. Task Learning uses existing public
 * ADS reports with explicit provenance, an explicit use/reject decision and a
 * later independent outcome; quality gain is never inferred and Agents are
 * never ranked. Every deviation fails closed with a bounded reason, a lost
 * GitHub acknowledgment stays UNKNOWN and later reconciles WITHOUT creating a
 * duplicate Issue.
 */
import {createHash} from 'node:crypto';
import {projectReport,STATES} from '../../../packages/tally-github-c/src/report-projection.mjs';

const OWNER='kaicreator-mm',REPO='tally',ISSUE=219;
const ISSUE_REF=`https://github.com/${OWNER}/${REPO}/issues/${ISSUE}`;
const SHA=/^[0-9a-f]{40}$/,DIGEST=/^[0-9a-f]{64}$/;
const SUMMARY=/^\S(?:[\s\S]{0,510}\S)?$/;
const IDENTITY=/^[A-Za-z0-9][A-Za-z0-9 .:_'@-]{0,127}$/;
const ISO=value=>typeof value==='string'&&Number.isFinite(Date.parse(value))&&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
const RELEASE_TAG=/^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$/;
const VERSION=/^v?[0-9]+\.[0-9]+(?:\.[0-9]+)?$/;
const DECISIONS=new Set(['ADOPTED','REJECTED']);
export const MAINTENANCE_CANDIDATE_KEY=`tally:v2:maintenance-candidate:${ISSUE}`;
export const MAINTENANCE_DRAFT_KEY=`tally:v2:maintenance-draft:${ISSUE}`;
export const MAINTENANCE_REVIEW_PROOF_KEY=`tally:v2:maintenance-review-proof:${ISSUE}`;
export const MAINTENANCE_ISSUE_KEY=`tally:v2:maintenance-issue:${ISSUE}`;
export const MAINTENANCE_PENDING_KEY=`tally:v2:maintenance-pending:${ISSUE}`;
export const MAINTENANCE_LEARNING_KEY=`tally:v2:maintenance-learning:${ISSUE}`;
export const MAINTENANCE_CLAIM_KEY=`${OWNER}/${REPO}#${ISSUE}:builder:__default__`;
const issueUrlFor=n=>`${ISSUE_REF.replace(new RegExp('[0-9]+$',''),'')}${n}`;
const ISSUE_URL=/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/[1-9][0-9]*$/;
const canonical=value=>Array.isArray(value)?value.map(canonical):
  value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(
    key=>[key,canonical(value[key])])):value;
const sha256=value=>createHash('sha256').update(JSON.stringify(canonical(value))).digest('hex');
const store=async(openStore,path)=>{
  const opened=await openStore({path,busyTimeoutMs:1000});
  return {async get(key){const row=await opened.get(key);
      return row===undefined?null:row;},
    async put(...args){return opened.put(...args);},async close(){await opened.close();}};
};
const flags=Object.freeze({claimGeneration:1,authorized:false,agentLaunch:false,
  historicalReleaseMutation:false,taskPass:false,releasePass:false});
const denied=(input,reason,more={})=>Object.freeze({...flags,
  state:'NOT_ADMITTED',reason,issueRef:ISSUE_REF,
  sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
  candidateDigest:DIGEST.test(input?.candidateDigest??'')?input.candidateDigest:null,
  claimAccepted:false,admitted:false,replayFixture:false,githubEffect:'NOT_RUN',
  issue:null,...more});
const unknownEffect=(input,reason)=>Object.freeze({...flags,
  state:'PUBLICATION_UNKNOWN',reason,issueRef:ISSUE_REF,sourceSha:input.sourceSha,
  candidateDigest:input.candidateDigest,claimAccepted:true,admitted:true,
  replayFixture:false,githubEffect:'UNKNOWN',issue:null});

/**
 * Validate one reported defect candidate against the immutable prior release.
 * The release identity is carried as evidence only; a missing release stays
 * UNKNOWN and never blocks the candidate.
 */
export function validateMaintenanceCandidate(candidate){
  if(!candidate||typeof candidate!=='object'||Array.isArray(candidate))return null;
  const {sourceIssueRef,requirementRef,subjectSha,summary,priorRelease}=candidate;
  if(!ISSUE_URL.test(sourceIssueRef??'')||!ISSUE_URL.test(requirementRef??'')||
    !SHA.test(subjectSha??'')||!SUMMARY.test(summary??''))return null;
  let release=null;
  if(priorRelease!==undefined&&priorRelease!==null){
    if(typeof priorRelease!=='object'||Array.isArray(priorRelease)||
      !VERSION.test(priorRelease.version??'')||!SHA.test(priorRelease.commitSha??'')||
      typeof priorRelease.evidenceRef!=='string'||
      priorRelease.evidenceRef!==`https://github.com/${OWNER}/${REPO}/releases/tag/${priorRelease.version}`&&
      !(priorRelease.evidenceRef.startsWith(`https://github.com/${OWNER}/${REPO}/releases/tag/`)&&
        RELEASE_TAG.test(priorRelease.evidenceRef.slice(
          `https://github.com/${OWNER}/${REPO}/releases/tag/`.length))))
      return null;
    release=Object.freeze({version:priorRelease.version,commitSha:priorRelease.commitSha,
      evidenceRef:priorRelease.evidenceRef,immutable:true});
  }else release=Object.freeze({status:'UNKNOWN'});
  return Object.freeze({sourceIssueRef,requirementRef,subjectSha,
    summary:summary.trim(),priorRelease:release});
}
export function maintenanceCandidateDigest(candidate){
  const valid=validateMaintenanceCandidate(candidate);
  return valid?sha256(valid):null;
}
function maintenanceTitle(candidate){
  return `[maintenance] ${candidate.summary.slice(0,160)}`;
}
function maintenanceBody(candidate,candidateDigest){
  const release=candidate.priorRelease?.version??
    candidate.priorRelease?.status??'UNKNOWN';
  return [
    'Defect candidate against an immutable historical release; the next',
    'version may absorb it only through governed intake.',
    '',
    `- source: ${candidate.sourceIssueRef}`,
    `- requirement: ${candidate.requirementRef}`,
    `- subject: ${candidate.subjectSha}`,
    `- reported release: ${release} (immutable, never rewritten)`,
    `- summary: ${candidate.summary}`,
    '',
    `maintenance-marker: tally:maintenance:219:${candidateDigest}`,
    '',
    'This Issue is the only authorized native artifact of this candidate.',
    'It creates no Task, no Claim and no historical change; Task Learning',
    'provenance stays reported-only with no quality-gain inference and no',
    'Agent ranking.',
  ].join('\n');
}

/**
 * Freeze one independently reviewed maintenance candidate into the Host CAS.
 * The reviewer must differ from the author; CAS create-conflict replays the
 * identical frozen review and refuses a divergent digest (never a second
 * freeze).
 */
export async function recordMaintenanceCandidate({openStore,registryPath,sourceSha,
  candidate,author,reviewer,reviewedAt}={}){
  const unqualified=reason=>Object.freeze({...flags,state:'NOT_RECORDED',reason,
    issueRef:ISSUE_REF,sourceSha:SHA.test(sourceSha??'')?sourceSha:null,
    candidateDigest:null,frozen:false,replayFixture:false,
    claimAccepted:false,admitted:false,githubEffect:'NOT_RUN'});
  if(typeof openStore!=='function'||typeof registryPath!=='string'||!registryPath)
    return unqualified('MAINTENANCE_FREEZE_STORE_UNAVAILABLE');
  const valid=validateMaintenanceCandidate(candidate);
  const digest=valid?sha256(valid):null;
  if(!SHA.test(sourceSha??'')||!digest||!IDENTITY.test(author??'')||
    !IDENTITY.test(reviewer??'')||author===reviewer||!ISO(reviewedAt??''))
    return unqualified(digest&&author===reviewer?'CANDIDATE_REVIEW_NOT_INDEPENDENT':
      'MAINTENANCE_INPUT_INVALID');
  let opened,existing;
  try{
    opened=await store(openStore,registryPath);
    existing=await opened.get(MAINTENANCE_CANDIDATE_KEY);
  }catch{return unqualified('MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
  finally{if(opened)await opened.close();}
  if(existing?.value?.phase==='REVIEWED_MAINTENANCE_CANDIDATE'){
    if(existing.value.candidateDigest===digest&&existing.value.sourceSha===sourceSha&&
      existing.value.review?.reviewer===reviewer)
      return Object.freeze({...unqualified('MAINTENANCE_REVIEW_ALREADY_RECORDED'),
        state:'RECORDED_COMPLETE',frozen:true,candidateDigest:digest,
        review:Object.freeze({author:String(existing.value.review.author),
          reviewer:String(existing.value.review.reviewer)}),
        receiptRef:`receipt:sha256:${digest}`});
    return unqualified('MAINTENANCE_REVIEW_DIGEST_MISMATCH');
  }
  const value=Object.freeze({phase:'REVIEWED_MAINTENANCE_CANDIDATE',issueRef:ISSUE_REF,
    sourceSha,candidateDigest:digest,candidate:valid,
    review:Object.freeze({author,reviewer}),reviewedAt,
    historicalReleaseMutation:false,authorized:false,admitted:false,
    taskPass:false,releasePass:false});
  try{
    opened=await store(openStore,registryPath);
    await opened.put(MAINTENANCE_CANDIDATE_KEY,value,{mustBeAbsent:true});
  }catch{return unqualified('MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
  finally{if(opened)await opened.close();}
  return Object.freeze({...flags,state:'RECORDED_COMPLETE',
    reason:'INDEPENDENT_CANDIDATE_REVIEW_FROZEN_IN_HOST_CAS',issueRef:ISSUE_REF,
    sourceSha,candidateDigest:digest,receiptRef:`receipt:sha256:${digest}`,
    frozen:true,replayFixture:false,claimAccepted:false,admitted:false,
    githubEffect:'NOT_RUN'});
}

/**
 * J08c provenance ledger: derive Task Learning suggestions ONLY from existing
 * public ADS report observations already collected by the native C reader.
 * Provenance keeps the exact report identity; qualification REPORTED_QUALIFIED
 * requires a CURRENT report with a QUALIFIED verifier — everything else stays
 * UNKNOWN. No quality gain and no Agent ranking can enter the ledger.
 */
export function learningSuggestionsFromReports(observations){
  if(!Array.isArray(observations)||observations.length>100)return null;
  const suggestions=[];
  for(const observation of observations){
    const row=projectReport(observation);
    if(row.status===STATES.INVALID||row.status===STATES.UNSUPPORTED)return null;
    const data=observation?.envelope?.data;
    const taskIssueRef=ISSUE_URL.test(data?.subject?.task_ref??'')?
      data.subject.task_ref:null;
    const targetIssueRef=ISSUE_URL.test(data?.subject?.issue_ref??'')?
      data.subject.issue_ref:null;
    suggestions.push(Object.freeze({
      learningId:row.source?sha256([row.source.reportId,row.source.revision,
        row.source.commentId,row.source.kind]):null,
      taskIssueRef,targetIssueRef,
      subjectSha:row.source?.sourceSha??null,
      qualification:row.status===STATES.CURRENT&&
        row.advisory?.verifierStatus==='QUALIFIED'?'REPORTED_QUALIFIED':'UNKNOWN',
      provenance:row.source?Object.freeze({reportId:row.source.reportId,
        revision:row.source.revision,kind:row.source.kind,
        commentId:row.source.commentId,issueNumber:row.source.issueNumber,
        namespace:row.source.namespace,dataDigest:row.source.dataDigest,
        epistemic:row.advisory?.epistemic??null,
        publicationRef:`https://github.com/${row.source.repository}/issues/${row.source.issueNumber}#issuecomment-${row.source.commentId}`}):null,
      measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false}));
  }
  return Object.freeze(suggestions);
}

/**
 * Persist the observed Task Learning ledger. A re-observation of the exact
 * same set replays; a divergent ledger refuses.
 */
export async function observeLearningSuggestions({openStore,registryPath,sourceSha,
  observations,observedAt}={}){
  const unqualified=reason=>Object.freeze({...flags,state:'NOT_RECORDED',reason,
    issueRef:ISSUE_REF,sourceSha:SHA.test(sourceSha??'')?sourceSha:null,
    githubEffect:'NOT_RUN',claimAccepted:false,admitted:false,
    suggestions:[]});
  if(typeof openStore!=='function'||typeof registryPath!=='string'||
    !SHA.test(sourceSha??'')||!ISO(observedAt??''))
    return unqualified('MAINTENANCE_INPUT_INVALID');
  const suggestions=learningSuggestionsFromReports(observations);
  if(suggestions===null)return unqualified('LEARNING_PROVENANCE_UNVERIFIED');
  let opened,existing;
  try{
    opened=await store(openStore,registryPath);
    existing=await opened.get(MAINTENANCE_LEARNING_KEY);
  }catch{return unqualified('MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
  finally{if(opened)await opened.close();}
  const ledger=Object.freeze({phase:'OBSERVED_LEARNING',issueRef:ISSUE_REF,
    sourceSha,suggestions,observedAt,noAgentRanking:true,
    historicalReleaseMutation:false,authorized:false,taskPass:false,releasePass:false});
  if(existing?.value?.phase==='OBSERVED_LEARNING'){
    if(sha256(existing.value.suggestions)===sha256(suggestions)&&
      existing.value.sourceSha===sourceSha)
      return Object.freeze({...unqualified('LEARNING_LEDGER_ALREADY_RECORDED'),
        state:'RECORDED_COMPLETE',suggestions});
    return unqualified('LEARNING_LEDGER_MISMATCH');
  }
  try{
    opened=await store(openStore,registryPath);
    await opened.put(MAINTENANCE_LEARNING_KEY,ledger,{mustBeAbsent:true});
  }catch{return unqualified('MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
  finally{if(opened)await opened.close();}
  return Object.freeze({...flags,state:'RECORDED_COMPLETE',
    reason:'PUBLIC_ADS_REPORT_PROVENANCE_OBSERVED',issueRef:ISSUE_REF,
    sourceSha,githubEffect:'NOT_RUN',claimAccepted:false,admitted:false,
    replayFixture:false,suggestions});
}

async function loadLearning(openStore,registryPath){
  let opened=null;
  try{
    opened=await store(openStore,registryPath);
    return await opened.get(MAINTENANCE_LEARNING_KEY);
  }catch{return null;}
  finally{if(opened)await opened.close();}
}

/**
 * J08c Host records: explicit use/reject decision and the later independent
 * outcome for one REPORTED_QUALIFIED learning suggestion. The outcome is an
 * observed fact only — measuredEffect never becomes a quality gain.
 */
export function createMaintenanceHost({openStore,verifyAcceptedClaim,
  verifyFrozenWriteSet,publishIssue,findIssue,readIssue,
  readReviewWitness,readCurrentVersion,withVersionFence,env=process.env}={}){
  const pin=input=>
    input?.issueNumber===ISSUE&&input.sourceRef===ISSUE_REF&&
    SHA.test(input.sourceSha??'')&&DIGEST.test(input.candidateDigest??'')&&
    input.sourceSha===env.TALLY_CLAIM_SOURCE_SHA&&
    typeof env.TALLY_TASK_CLAIM_219_REGISTRY_PATH==='string'&&
    env.TALLY_TASK_CLAIM_219_REGISTRY_PATH;
  return Object.freeze({
    // Pre-review durable CAS draft; cannot be created by a typed reviewer.
    async hostPersistMaintenanceDraft(input={}){
      if(!pin(input))return denied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
      if(typeof verifyAcceptedClaim!=='function'||
         typeof verifyFrozenWriteSet!=='function'||typeof openStore!=='function')
        return denied(input,'MAINTENANCE_HOST_CONFIG_INVALID');
      const candidate=validateMaintenanceCandidate(input.candidate);
      if(!candidate||sha256(candidate)!==input.candidateDigest)
        return denied(input,'MAINTENANCE_DRAFT_INVALID');
      const claim=await verifyAcceptedClaim(ISSUE,'builder');
      if(claim?.ok!==true)return denied(input,claim?.reason??'CLAIM_UNVERIFIED');
      const fs=await verifyFrozenWriteSet(ISSUE,input.sourceSha);
      if(fs?.ok!==true)return denied(input,fs?.reason??'DAG_WRITESET_NOT_VERIFIED');
      const value={phase:'DRAFT_MAINTENANCE_CANDIDATE',issueRef:ISSUE_REF,
        sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
        candidate,authorized:false,taskPass:false,releasePass:false};
      const same=row=>row?.value?.phase===value.phase&&
        row.value.sourceSha===value.sourceSha&&
        row.value.candidateDigest===value.candidateDigest&&
        JSON.stringify(row.value)===JSON.stringify(value);
      let db;
      try{
        db=await store(openStore,env.TALLY_TASK_CLAIM_219_REGISTRY_PATH);
        const prior=await db.get(MAINTENANCE_DRAFT_KEY);
        if(prior&&!same(prior))return denied(input,'MAINTENANCE_DRAFT_CONFLICT');
        if(!prior)await db.put(MAINTENANCE_DRAFT_KEY,value,{mustBeAbsent:true});
        const after=await db.get(MAINTENANCE_DRAFT_KEY);
        if(!same(after))return denied(input,'MAINTENANCE_DRAFT_CAS_UNKNOWN');
        return Object.freeze({...flags,state:'DRAFT_PENDING_REVIEW',
          reason:prior?'MAINTENANCE_DRAFT_ALREADY_RECORDED':'MAINTENANCE_DRAFT_RECORDED',
          issueRef:ISSUE_REF,sourceSha:input.sourceSha,
          candidateDigest:input.candidateDigest,authorized:false,
          taskPass:false,releasePass:false});
      }catch{return denied(input,'MAINTENANCE_DRAFT_CAS_UNKNOWN');}
      finally{if(db)await db.close();}
    },
    // Separate C ReviewWitness, independently accepted role Claims and FSW
    // are reread inside a real selected version CAS fence before creation.
    async hostFreezeReviewedMaintenanceCandidate(input={}){
      if(!pin(input)||!Number.isSafeInteger(input.reviewCommentId)||
         input.reviewCommentId<1||!Number.isSafeInteger(input.reviewedPullNumber)||
         input.reviewedPullNumber<1||!SHA.test(input.reviewedHead??'')||
         !SHA.test(input.reviewedTree??'')||
         input.reviewedBase!==input.sourceSha)
        return denied(input,'MAINTENANCE_REVIEW_INPUT_INVALID');
      if(typeof verifyAcceptedClaim!=='function'||
         typeof verifyFrozenWriteSet!=='function'||
         typeof readReviewWitness!=='function'||
         typeof readCurrentVersion!=='function'||
         typeof withVersionFence!=='function'||typeof openStore!=='function')
        return denied(input,'MAINTENANCE_HOST_CONFIG_INVALID');
      let db,draft;
      try{
        db=await store(openStore,env.TALLY_TASK_CLAIM_219_REGISTRY_PATH);
        draft=await db.get(MAINTENANCE_DRAFT_KEY);
      }catch{return denied(input,'MAINTENANCE_DRAFT_CAS_UNKNOWN');}
      finally{if(db)await db.close();}
      if(draft?.value?.phase!=='DRAFT_MAINTENANCE_CANDIDATE'||
         draft.value.sourceSha!==input.sourceSha||
         draft.value.candidateDigest!==input.candidateDigest||
         maintenanceCandidateDigest(draft.value.candidate)!==input.candidateDigest)
        return denied(input,'MAINTENANCE_DRAFT_NOT_VERIFIED');
      const [b,r]=await Promise.all([
        verifyAcceptedClaim(ISSUE,'builder'),
        verifyAcceptedClaim(ISSUE,'reviewer')]);
      const ok=(c,role)=>c?.ok===true&&c.claim?.accepted===true&&
        c.claim.role===role&&c.claim.issueNumber===ISSUE&&
        c.claim.sourceSha===input.sourceSha&&
        typeof c.claim.claimRef==='string'&&c.claim.claimRef.length>0&&
        typeof c.claim.operatorId==='string'&&c.claim.operatorId.length>0&&
        typeof c.claim.writerId==='string'&&c.claim.writerId.length>0&&
        typeof c.claim.transportActor==='string'&&c.claim.transportActor.length>0;
      if(!ok(b,'builder')||!ok(r,'reviewer'))
        return denied(input,'MAINTENANCE_REVIEW_ROLE_CLAIM_UNVERIFIED');
      if(b.claim.operatorId===r.claim.operatorId||b.claim.writerId===r.claim.writerId)
        return denied(input,'MAINTENANCE_REVIEW_SAME_PROCESS');
      const fs=await verifyFrozenWriteSet(ISSUE,input.sourceSha);
      if(fs?.ok!==true)return denied(input,fs?.reason??'DAG_WRITESET_NOT_VERIFIED');
      const bind=c=>({...c,candidateDigest:draft.value.candidateDigest});
      const expected={issueNumber:ISSUE,sourceSha:input.sourceSha,
        candidateDigest:draft.value.candidateDigest,
        reviewedPullNumber:input.reviewedPullNumber,
        reviewedHead:input.reviewedHead,reviewedTree:input.reviewedTree,
        reviewedBase:input.reviewedBase};
      const args={commentId:input.reviewCommentId,expected,
        builderClaim:bind(b.claim),reviewerClaim:bind(r.claim)};
      const ref=ISSUE_REF+'#issuecomment-'+input.reviewCommentId;
      const witnessOk=w=>w?.state==='REVIEW_VERIFIED'&&w.reviewed===true&&
        w.reviewRef===ref&&w.sourceSha===input.sourceSha&&
        w.candidateDigest===input.candidateDigest&&
        w.reviewedHead===input.reviewedHead&&w.reviewedTree===input.reviewedTree&&
        ISO(w.reviewedAt??'')&&w.taskPass===false&&w.authorized===false;
      let first;
      try{first=await readReviewWitness(args);}
      catch{return denied(input,'MAINTENANCE_REVIEW_WITNESS_UNVERIFIED');}
      if(!witnessOk(first))return denied(input,'MAINTENANCE_REVIEW_WITNESS_UNVERIFIED');
      try{
        const current=await readCurrentVersion();
        if(current?.ok!==true||current.sha!==input.sourceSha)
          return denied(input,'MAINTENANCE_SOURCE_SHA_STALE');
      }catch{return denied(input,'MAINTENANCE_SOURCE_SHA_STALE');}
      try{
        return await withVersionFence({repository:OWNER+'/'+REPO,
          versionId:'version/v2',sourceSha:input.sourceSha,
          operation:'FREEZE_REVIEWED_MAINTENANCE'},async()=>{
          const second=await readReviewWitness(args);
          if(!witnessOk(second)||second.reviewedAt!==first.reviewedAt)
            return denied(input,'MAINTENANCE_REVIEW_WITNESS_CHANGED');
          const current=await readCurrentVersion();
          if(current?.ok!==true||current.sha!==input.sourceSha)
            return denied(input,'MAINTENANCE_SOURCE_SHA_STALE');
          const recorded=await recordMaintenanceCandidate({openStore,
            registryPath:env.TALLY_TASK_CLAIM_219_REGISTRY_PATH,
            sourceSha:input.sourceSha,candidate:draft.value.candidate,
            author:b.claim.operatorId,reviewer:r.claim.operatorId,
            reviewedAt:second.reviewedAt});
          if(recorded?.state!=='RECORDED_COMPLETE')return recorded;
          // Native Issue publication can consume ONLY this extra proof:
          // a legacy author/reviewer text-only row is insufficient.
          const proof={phase:'VERIFIED_MAINTENANCE_REVIEW_PROOF',
            issueRef:ISSUE_REF,sourceSha:input.sourceSha,
            candidateDigest:input.candidateDigest,
            reviewRef:ref,reviewedHead:input.reviewedHead,
            reviewedTree:input.reviewedTree,reviewedBase:input.reviewedBase,
            builderClaimRef:b.claim.claimRef,reviewerClaimRef:r.claim.claimRef,
            builderOperatorId:b.claim.operatorId,reviewerOperatorId:r.claim.operatorId,
            reviewedAt:second.reviewedAt,taskPass:false,releasePass:false};
          let db=null;
          try{
            db=await store(openStore,env.TALLY_TASK_CLAIM_219_REGISTRY_PATH);
            const prior=await db.get(MAINTENANCE_REVIEW_PROOF_KEY);
            if(prior&&JSON.stringify(prior.value)!==JSON.stringify(proof))
              return denied(input,'MAINTENANCE_REVIEW_PROOF_CONFLICT');
            if(!prior)await db.put(MAINTENANCE_REVIEW_PROOF_KEY,proof,
              {mustBeAbsent:true});
            const final=await db.get(MAINTENANCE_REVIEW_PROOF_KEY);
            if(JSON.stringify(final?.value)!==JSON.stringify(proof))
              return denied(input,'MAINTENANCE_REVIEW_PROOF_CAS_UNKNOWN');
          }catch{return denied(input,'MAINTENANCE_REVIEW_PROOF_CAS_UNKNOWN');}
          finally{if(db)await db.close();}
          return recorded;
        });
      }catch{return denied(input,'MAINTENANCE_REVIEW_FREEZE_UNKNOWN');}
    },
    async hostPublishMaintenanceIssue(input={}){
      if(typeof verifyAcceptedClaim!=='function'||typeof verifyFrozenWriteSet!=='function')
        return denied(input,'MAINTENANCE_HOST_CONFIG_INVALID');
      if(!pin(input))return denied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
      const registryPath=env.TALLY_TASK_CLAIM_219_REGISTRY_PATH;
      const claim=await verifyAcceptedClaim(ISSUE,'builder');
      if(!claim.ok)return denied(input,claim.reason);
      const writeSet=await verifyFrozenWriteSet(ISSUE,input.sourceSha);
      if(!writeSet.ok)return denied(input,writeSet.reason);
      let opened=null,candidateRecord;
      try{
        opened=await store(openStore,registryPath);
        candidateRecord=await opened.get(MAINTENANCE_CANDIDATE_KEY);
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      const reviewed=candidateRecord?.value;
      if(reviewed?.phase!=='REVIEWED_MAINTENANCE_CANDIDATE')
        return denied(input,'CANDIDATE_REVIEW_NOT_VERIFIED');
      if(reviewed.sourceSha!==input.sourceSha||
        reviewed.candidateDigest!==input.candidateDigest)
        return denied(input,'MAINTENANCE_REVIEW_DIGEST_MISMATCH');
      if(typeof reviewed.review?.reviewer!=='string'||
        reviewed.review.reviewer===reviewed.review?.author)
        return denied(input,'CANDIDATE_REVIEW_NOT_INDEPENDENT');
      const candidate=validateMaintenanceCandidate(reviewed.candidate);
      if(!candidate||sha256(candidate)!==input.candidateDigest)
        return denied(input,'MAINTENANCE_REVIEW_DIGEST_MISMATCH');
      // Migration guard: older rows created by a typed reviewer string have
      // no trusted C/Host proof and MUST NEVER publish a native Issue.
      let proofRow;
      try{
        opened=await store(openStore,registryPath);
        proofRow=await opened.get(MAINTENANCE_REVIEW_PROOF_KEY);
      }catch{return denied(input,'MAINTENANCE_REVIEW_PROOF_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      const proof=proofRow?.value;
      if(proof?.phase!=='VERIFIED_MAINTENANCE_REVIEW_PROOF'||
         proof.issueRef!==ISSUE_REF||proof.sourceSha!==input.sourceSha||
         proof.candidateDigest!==input.candidateDigest||
         !/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/219#issuecomment-[1-9][0-9]*$/.test(proof.reviewRef??'')||
         !SHA.test(proof.reviewedHead??'')||
         !SHA.test(proof.reviewedTree??'')||
         proof.reviewedBase!==input.sourceSha||
         proof.builderOperatorId!==reviewed.review.author||
         proof.reviewerOperatorId!==reviewed.review.reviewer||
         !ISO(proof.reviewedAt??'')||
         proof.taskPass!==false||proof.releasePass!==false)
        return denied(input,'MAINTENANCE_TRUSTED_REVIEW_PROOF_MISSING');
      let issueRecord;
      try{
        opened=await store(openStore,registryPath);
        issueRecord=await opened.get(MAINTENANCE_ISSUE_KEY);
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      const published=issueRecord?.value;
      if(published?.phase==='PUBLISHED_MAINTENANCE_ISSUE'){
        if(published.candidateDigest===input.candidateDigest&&
          published.sourceSha===input.sourceSha)
          return Object.freeze({...flags,state:'MAINTENANCE_ISSUE_PUBLISHED',
            reason:'MAINTENANCE_ISSUE_READBACK_ALREADY_PRESENT',issueRef:ISSUE_REF,
            sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
            claimAccepted:true,admitted:true,replayFixture:false,
            githubEffect:'PERFORMED_VERIFIED',
            repository:`${OWNER}/${REPO}`,
            issue:Object.freeze(published.issue),alreadyPresent:true,
            duplicate:false});
        return denied(input,'MAINTENANCE_ISSUE_DIGEST_MISMATCH');
      }
      if(env.TALLY_LOCAL_REPLAY==='1')
        return Object.freeze({...flags,state:'REPLAY_COMPLETE',
          reason:'REPLAY_FIXTURE_ONLY',issueRef:ISSUE_REF,
          sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
          claimAccepted:true,admitted:true,replayFixture:true,
          githubEffect:'NOT_RUN',issue:null});
      if(typeof publishIssue!=='function'||typeof findIssue!=='function'||
        typeof readIssue!=='function'||typeof env.TALLY_GITHUB_READ_TOKEN!=='string'||
        env.TALLY_GITHUB_READ_TOKEN.length<8)
        return denied(input,'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE');
      const title=maintenanceTitle(candidate);
      const marker=`tally:maintenance:219:${input.candidateDigest}`;
      // Lost-ACK reconciliation: a previous attempt may have left a pending
      // marker. Search first; never create a second Issue on top of one.
      let pending;
      try{
        opened=await store(openStore,registryPath);
        pending=await opened.get(MAINTENANCE_PENDING_KEY);
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      if(pending?.value?.phase==='PENDING_MAINTENANCE_ISSUE'){
        if(pending.value.candidateDigest!==input.candidateDigest)
          return denied(input,'MAINTENANCE_ISSUE_DIGEST_MISMATCH');
        const found=await findIssue({marker,title});
        if(found?.state==='RECONCILE_CONFLICT')
          return unknownEffect(input,found.reason);
        if(found?.state==='RECONCILE_UNKNOWN')
          return unknownEffect(input,found.reason);
        if(found?.state==='RECONCILE_COMPLETE'&&found.matches.length===1){
          const number=found.matches[0].number;
          const readback=await readIssue({issueNumber:number,title,marker});
          if(readback?.state!=='READBACK_VERIFIED')
            return unknownEffect(input,'MAINTENANCE_ISSUE_READBACK_MISMATCH');
          try{
            opened=await store(openStore,registryPath);
            await opened.put(MAINTENANCE_ISSUE_KEY,Object.freeze({
              phase:'PUBLISHED_MAINTENANCE_ISSUE',issueRef:ISSUE_REF,
              sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
              issue:readback.issue,publishedAt:new Date().toISOString(),
              reconciled:true,authorized:false,admitted:false,
              taskPass:false,releasePass:false}),{mustBeAbsent:true});
          }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
          finally{if(opened)await opened.close();}
          return Object.freeze({...flags,state:'MAINTENANCE_ISSUE_PUBLISHED',
            reason:'MAINTENANCE_ISSUE_RECONCILED_AFTER_LOST_ACK',issueRef:ISSUE_REF,
            sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
            claimAccepted:true,admitted:true,replayFixture:false,
            githubEffect:'PERFORMED_VERIFIED',repository:`${OWNER}/${REPO}`,
            issue:readback.issue,duplicate:false});
        }
      }
      try{
        opened=await store(openStore,registryPath);
        await opened.put(MAINTENANCE_PENDING_KEY,Object.freeze({
          phase:'PENDING_MAINTENANCE_ISSUE',issueRef:ISSUE_REF,
          sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
          marker,title,pendingAt:new Date().toISOString(),
          authorized:false,taskPass:false,releasePass:false}),
          {mustBeAbsent:true});
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      let publishedEffect;
      try{
        publishedEffect=await publishIssue({title,
          body:maintenanceBody(candidate,input.candidateDigest),marker});
      }catch{return unknownEffect(input,'GITHUB_PUBLISH_UNKNOWN');}
      if(publishedEffect?.state!=='ISSUE_PUBLISHED')
        return unknownEffect(input,publishedEffect?.reason??'GITHUB_PUBLISH_UNKNOWN');
      const readback=await readIssue({issueNumber:publishedEffect.issue.number,
        title,marker});
      if(readback?.state!=='READBACK_VERIFIED')
        return unknownEffect(input,'MAINTENANCE_ISSUE_READBACK_MISMATCH');
      try{
        opened=await store(openStore,registryPath);
        await opened.put(MAINTENANCE_ISSUE_KEY,Object.freeze({
          phase:'PUBLISHED_MAINTENANCE_ISSUE',issueRef:ISSUE_REF,
          sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,
          issue:readback.issue,publishedAt:new Date().toISOString(),
          reconciled:false,authorized:false,admitted:false,
          taskPass:false,releasePass:false}),{mustBeAbsent:true});
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      return Object.freeze({...flags,state:'MAINTENANCE_ISSUE_PUBLISHED',
        reason:'MAINTENANCE_ISSUE_PUBLISHED_AND_READBACK_VERIFIED',
        issueRef:ISSUE_REF,sourceSha:input.sourceSha,
        candidateDigest:input.candidateDigest,claimAccepted:true,admitted:true,
        replayFixture:false,githubEffect:'PERFORMED_VERIFIED',
        repository:`${OWNER}/${REPO}`,issue:readback.issue,duplicate:false});
    },
    async hostRecordTaskLearningOutcome(input={}){
      if(typeof verifyAcceptedClaim!=='function')
        return denied(input,'MAINTENANCE_HOST_CONFIG_INVALID');
      const outcome=input.outcome;
      if(outcome!==undefined&&(typeof outcome!=='object'||outcome===null||
        Array.isArray(outcome)||typeof outcome.attested!=='boolean'||
        !ISO(outcome.observedAt??'')||
        typeof outcome.note!=='string'||outcome.note.length>512))
        return denied(input,'MAINTENANCE_INPUT_INVALID');
      if(!DECISIONS.has(input.decision))
        return denied(input,'MAINTENANCE_INPUT_INVALID');
      if(!pin({...input,candidateDigest:input.learningId}))
        return denied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
      const registryPath=env.TALLY_TASK_CLAIM_219_REGISTRY_PATH;
      const claim=await verifyAcceptedClaim(ISSUE,'builder');
      if(!claim.ok)return denied(input,claim.reason);
      const ledger=await loadLearning(openStore,registryPath);
      const suggestion=ledger?.value?.suggestions?.find(
        row=>row.learningId===input.learningId);
      if(!suggestion||suggestion.qualification!=='REPORTED_QUALIFIED')
        return denied(input,'LEARNING_PROVENANCE_NOT_VERIFIED');
      if(env.TALLY_LOCAL_REPLAY==='1')
        return Object.freeze({...flags,state:'REPLAY_COMPLETE',
          reason:'REPLAY_FIXTURE_ONLY',issueRef:ISSUE_REF,
          sourceSha:input.sourceSha,learningId:input.learningId,
          decision:input.decision,claimAccepted:true,admitted:true,
          replayFixture:true,githubEffect:'NOT_RUN',
          measuredEffect:'NOT_MEASURED'});
      let opened=null,existingDecision;
      try{
        opened=await store(openStore,registryPath);
        existingDecision=await opened.get(
          `tally:v2:maintenance-learning-decision:${input.learningId}`);
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      if(outcome===undefined){
        if(existingDecision?.value?.phase==='LEARNING_DECISION'){
          if(existingDecision.value.decision===input.decision)
            return Object.freeze({...flags,state:'LEARNING_RECORDED',
              reason:'LEARNING_DECISION_ALREADY_RECORDED',issueRef:ISSUE_REF,
              sourceSha:input.sourceSha,learningId:input.learningId,
              decision:input.decision,claimAccepted:true,admitted:true,
              replayFixture:false,githubEffect:'NOT_RUN',
              measuredEffect:'NOT_MEASURED',noAgentRanking:true,
              qualityGainClaimed:false});
          return denied(input,'LEARNING_DECISION_MISMATCH');
        }
        try{
          opened=await store(openStore,registryPath);
          await opened.put(`tally:v2:maintenance-learning-decision:${input.learningId}`,
            Object.freeze({phase:'LEARNING_DECISION',issueRef:ISSUE_REF,
              sourceSha:input.sourceSha,learningId:input.learningId,
              decision:input.decision,decidedAt:new Date().toISOString(),
              provenance:suggestion.provenance,noAgentRanking:true,
              qualityGainClaimed:false,authorized:false,taskPass:false,
              releasePass:false}),{mustBeAbsent:true});
        }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
        finally{if(opened)await opened.close();}
        return Object.freeze({...flags,state:'LEARNING_RECORDED',
          reason:'LEARNING_DECISION_RECORDED_IN_HOST_CAS',issueRef:ISSUE_REF,
          sourceSha:input.sourceSha,learningId:input.learningId,
          decision:input.decision,claimAccepted:true,admitted:true,
          replayFixture:false,githubEffect:'NOT_RUN',
          measuredEffect:'NOT_MEASURED',noAgentRanking:true,
          qualityGainClaimed:false});
      }
      if(existingDecision?.value?.phase!=='LEARNING_DECISION')
        return denied(input,'LEARNING_DECISION_REQUIRED');
      const outcomeRecord=Object.freeze({phase:'LEARNING_OUTCOME',
        issueRef:ISSUE_REF,sourceSha:input.sourceSha,
        learningId:input.learningId,decision:existingDecision.value.decision,
        attested:outcome.attested===true,
        measuredEffect:outcome.attested===true?'ATTESTED_OUTCOME_ONLY':'NOT_MEASURED',
        observedAt:outcome.observedAt,note:outcome.note,outcomeAt:new Date().toISOString(),
        noAgentRanking:true,qualityGainClaimed:false,authorized:false,
        taskPass:false,releasePass:false});
      let existingOutcome;
      try{
        opened=await store(openStore,registryPath);
        existingOutcome=await opened.get(
          `tally:v2:maintenance-learning-outcome:${input.learningId}`);
        if(existingOutcome?.value?.phase!=='LEARNING_OUTCOME')
          await opened.put(`tally:v2:maintenance-learning-outcome:${input.learningId}`,
            outcomeRecord,{mustBeAbsent:true});
      }catch{return denied(input,'MAINTENANCE_FREEZE_STORE_UNAVAILABLE');}
      finally{if(opened)await opened.close();}
      if(existingOutcome?.value?.phase==='LEARNING_OUTCOME'){
        if(sha256(existingOutcome.value)===sha256(outcomeRecord))
          return Object.freeze({...flags,state:'LEARNING_RECORDED',
            reason:'LEARNING_OUTCOME_ALREADY_RECORDED',issueRef:ISSUE_REF,
            sourceSha:input.sourceSha,learningId:input.learningId,
            decision:outcomeRecord.decision,measuredEffect:outcomeRecord.measuredEffect,
            outcome:Object.freeze({
              attested:outcomeRecord.attested,
              measuredEffect:outcomeRecord.measuredEffect,
              observedAt:outcomeRecord.observedAt}),
            claimAccepted:true,admitted:true,replayFixture:false,
            githubEffect:'NOT_RUN',noAgentRanking:true,
            qualityGainClaimed:false});
        return denied(input,'LEARNING_OUTCOME_MISMATCH');
      }
      return Object.freeze({...flags,state:'LEARNING_RECORDED',
        reason:'LEARNING_OUTCOME_RECORDED_IN_HOST_CAS',issueRef:ISSUE_REF,
        sourceSha:input.sourceSha,learningId:input.learningId,
        decision:outcomeRecord.decision,measuredEffect:outcomeRecord.measuredEffect,
        outcome:Object.freeze({
          attested:outcomeRecord.attested,
          measuredEffect:outcomeRecord.measuredEffect,
          observedAt:outcomeRecord.observedAt}),
        claimAccepted:true,admitted:true,replayFixture:false,
        githubEffect:'NOT_RUN',noAgentRanking:true,qualityGainClaimed:false});
    },
  });
}
export const __test={OWNER,REPO,ISSUE,ISSUE_REF,issueUrlFor,validateMaintenanceCandidate,
  maintenanceCandidateDigest,maintenanceTitle,maintenanceBody,learningSuggestionsFromReports,
  MAINTENANCE_CLAIM_KEY};
