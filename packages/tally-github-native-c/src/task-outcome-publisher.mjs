/**
 * I377-R93-E2 — selected qualified Task Outcome publisher (task #377).
 *
 * R90 ladder S6 (#392/6983294817): after S1 FSW, S2 both accepted role
 * Claims, S3 Join, S4 validator evidence and S5 an independent ACCEPT
 * ReviewDecision, this port — and only this port — publishes the durable
 * qualified Task Outcome: ONE stable idempotency key, create-only CAS write
 * bound to the full provenance chain, exact readback, and a live native
 * reconciliation. Failure semantics per the E2 pack (#454 blob
 * a6da5d4766e2fea67f9513359706675ad3939eb8):
 *   REWORK / DEFER / MISSING_EVIDENCE / STALE_SOURCE / UNKNOWN_ACK /
 *   REVOKED_ROLE / UNQUALIFIED_CI  ⇒  taskPass=false, releasePass=false,
 *   publication HOLD — REWORK and DEFER are never Task PASS.
 * The publisher is distinct from every other success marker: a
 * DISPATCH_CLAIMED comment ACK, a native issue close, a native DONE label, a
 * journey projection, or a merged source repair is never an Outcome.
 * Authority is ONLY trusted Host configuration; no caller input, comment
 * text or flag is ever write evidence. Lost/ambiguous CAS ACK ⇒ UNKNOWN and
 * reconcileTaskOutcome() classifies read-only — never a second write.
 */
import {createHash} from 'node:crypto';

const SHA=/^[0-9a-f]{40}$/;
const DIGEST=/^[0-9a-f]{64}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const URL_PREFIX=(owner,repo)=>'https://github.com/'+owner+'/'+repo+'/';
const providerReason=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===422?'GITHUB_VALIDATION_FAILED':null;
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);

/** Strict shape check of one accepted role binding (CAS + native refs). */
function validClaimBinding(b){
  return plain(b)&&ID.test(b.casKey??'')&&Number.isSafeInteger(b.revision)&&
    b.revision>=0&&Number.isSafeInteger(b.commentId)&&b.commentId>0&&
    typeof b.commentRef==='string'&&typeof b.commentBodySha256==='string'&&
    DIGEST.test(b.commentBodySha256)&&
    ['validator','reviewer'].includes(b.actorRole)&&
    ID.test(b.operatorKind??'')&&ID.test(b.operatorId??'')&&
    ID.test(b.dispatchId??'')&&b.generation===1;
}
function validDecision(d){
  return plain(d)&&d.decision==='ACCEPT'&&
    typeof d.decisionRef==='string'&&d.decisionRef.startsWith('https://github.com/')&&
    ID.test(d.reviewerOperatorId??'')&&d.reviewerOperatorId!=='__default__'&&
    !Number.isNaN(Date.parse(d.decidedAt??''));
}

export function createTaskOutcomePublisher({
  repository,owner,repo,targetIssue,expectedSourceSha,versionBranch,
  octokit,store,fswStore,claimStore,joinStore,writerAuthority,now
}={}){
  if(!REPO.test(repository??'')||repository!==(owner??'')+'/'+(repo??'')||
     !NAME.test(owner??'')||!NAME.test(repo??'')||
     !Number.isSafeInteger(targetIssue)||targetIssue<1||
     !SHA.test(expectedSourceSha??'')||
     (versionBranch!==undefined&&typeof versionBranch!=='string')||
     typeof store?.get!=='function'||typeof store?.put!=='function'||
     typeof fswStore?.get!=='function'||typeof claimStore?.get!=='function'||
     typeof joinStore?.get!=='function'||
     typeof octokit?.rest?.issues?.get!=='function'||
     typeof octokit?.rest?.repos?.getBranch!=='function'||
     writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
     writerAuthority?.exclusiveHostControl!==true||
     writerAuthority?.repository!==repository||
     !ID.test(writerAuthority?.writerId??'')||
     !/^[A-Za-z0-9_.-]+(?:\[bot\])?$/.test(writerAuthority?.githubWriterLogin??''))
    throw new TypeError('TRUSTED_TASK_OUTCOME_PUBLISHER_REQUIRED');
  const clock=now??(()=>new Date().toISOString());
  const blocked=(reason,more={})=>Object.freeze({status:'BLOCKED',reason,
    repository,targetIssue,native:false,publication:'HOLD',
    taskPass:false,releasePass:false,...more});
  const unknown=(reason,more={})=>Object.freeze({status:'UNKNOWN',reason,
    repository,targetIssue,native:false,publication:'HOLD',
    taskPass:false,releasePass:false,reconciliationRequired:true,...more});

  async function readIssue(){
    const data=(await octokit.rest.issues.get({owner,repo,issue_number:targetIssue}))?.data;
    if(!plain(data)||data.number!==targetIssue||data.pull_request!==undefined||
       data.repository_url!=='https://api.github.com/repos/'+owner+'/'+repo||
       !['open','closed'].includes(data.state)||
       typeof data.updated_at!=='string'||!Array.isArray(data.labels))
      return null;
    const labels=data.labels.map(l=>typeof l==='string'?l:l?.name)
      .filter(x=>typeof x==='string');
    return {state:data.state,updatedAt:data.updated_at,labels};
  }

  /** Publish the qualified Outcome for a COMPLETE S1→S5 provenance chain. */
  async function publishQualifiedOutcome(input){
    const inp=plain(input)?input:{};
    // Gate 1 — REWORK/DEFER or a reviewer claim posing as a decision never
    // passes: refuse before any store write, with publication HOLD.
    if(inp.decision&&!validDecision(inp.decision)){
      if(plain(inp.decision)&&(inp.decision.decision==='REWORK'||
         inp.decision.decision==='DEFER'))
        return blocked('DECISION_'+inp.decision.decision+'_NOT_TASK_PASS',
          {publication:'HOLD'});
      return blocked('INDEPENDENT_DECISION_INVALID');
    }
    if(!inp.decision)return blocked('MISSING_EVIDENCE',{detail:'review decision'});
    // Gate 2 — full provenance chain shape.
    const required=['fsw','join','validatorClaim','reviewerClaim',
      'validatorEvidence','sourcePack','ci','hostFence'];
    for(const field of required){
      if(!plain(inp[field]))return blocked('MISSING_EVIDENCE',{detail:field});
    }
    if(!plain(inp.validatorEvidence)||
       typeof inp.validatorEvidence.evidenceRef!=='string'||
       !inp.validatorEvidence.evidenceRef.startsWith(URL_PREFIX(owner,repo)))
      return blocked('MISSING_EVIDENCE',{detail:'validatorEvidence.evidenceRef'});
    if(inp.validatorClaim&&validClaimBinding(inp.validatorClaim)&&
       inp.validatorClaim.actorRole==='reviewer')
      return blocked('REVOKED_ROLE',{detail:'reviewer binding in validator slot'});
    for(const [field,binding] of [['validatorClaim',inp.validatorClaim],
        ['reviewerClaim',inp.reviewerClaim]]){
      if(!validClaimBinding(binding))
        return blocked('REVOKED_ROLE',{detail:field+' binding invalid'});
    }
    if(inp.validatorClaim.operatorId===inp.reviewerClaim.operatorId)
      return blocked('REVOKED_ROLE',{detail:'validator/reviewer operator alias'});
    if(!plain(inp.ci)||typeof inp.ci.receiptsRef!=='string'||
       !inp.ci.receiptsRef.startsWith(URL_PREFIX(owner,repo)))
      return blocked('UNQUALIFIED_CI',{detail:'ci.receiptsRef'});
    if(!plain(inp.hostFence)||ID.test(inp.hostFence.writerId??'')===false||
       typeof inp.hostFence.fenceKey!=='string'||
       !inp.hostFence.fenceKey.startsWith('tally:v2:task-ready-fence:'))
      return blocked('MISSING_EVIDENCE',{detail:'hostFence'});
    try{
      // Gate 3 — live native state: the issue must still be open at the
      // ready frontier; a close/DONE elsewhere is never the Outcome.
      const issue=await readIssue();
      if(!issue)return blocked('GITHUB_RESOURCE_NOT_FOUND');
      if(issue.state!=='open')return blocked('STALE_SOURCE',{detail:'issue not open'});
      if(!issue.labels.includes('state:ready'))
        return blocked('STALE_SOURCE',{detail:'workflowState '+JSON.stringify(issue.labels)});
      // Gate 4 — current source epoch on the live branch.
      if(versionBranch!==undefined){
        const current=(await octokit.rest.repos.getBranch(
          {owner,repo,branch:versionBranch}))?.data?.commit?.sha??null;
        if(current!==expectedSourceSha)return blocked('STALE_SOURCE',
          {detail:'source epoch drift',expectedSourceSha,actualBranchHead:current});
      }
      // Gate 5 — durable prerequisites re-read from the SAME trusted stores
      // the bindings name: FSW/Join/claims must exist at the bound revisions
      // with the bound value digests; a drifted row is stale, never healed.
      const fswRow=await fswStore.get(inp.fsw.key);
      if(!fswRow||fswRow.revision!==inp.fsw.revision||
         createHash('sha256').update(JSON.stringify(fswRow.value),'utf8')
           .digest('hex')!==inp.fsw.valueSha256)
        return blocked('STALE_SOURCE',{detail:'fsw row drifted or absent'});
      if(fswRow.value.document!=='[]'||fswRow.value.phase!=='FROZEN_DAG_WRITESET')
        return blocked('STALE_SOURCE',{detail:'fsw row not the accepted empty scope'});
      const joinRow=await joinStore.get(inp.join.key);
      if(!joinRow||joinRow.revision!==inp.join.revision||
         createHash('sha256').update(JSON.stringify(joinRow.value),'utf8')
           .digest('hex')!==inp.join.valueSha256)
        return blocked('STALE_SOURCE',{detail:'join row drifted or absent'});
      if(joinRow.value.validatorClaim.casKey!==inp.validatorClaim.casKey||
         joinRow.value.reviewerClaim.casKey!==inp.reviewerClaim.casKey)
        return blocked('REVOKED_ROLE',{detail:'join binds different claim rows'});
      for(const [role,binding] of [['validator',inp.validatorClaim],
          ['reviewer',inp.reviewerClaim]]){
        const row=await claimStore.get(binding.casKey);
        if(!row||row.revision!==binding.revision||
           row.value.phase!=='PUBLISHED_VERIFIED'||
           row.value.commentId!==binding.commentId||
           row.value.commentBodySha256!==binding.commentBodySha256||
           row.value.operatorId!==binding.operatorId)
          return blocked('REVOKED_ROLE',{detail:role+' claim drifted or revoked'});
      }
      // Gate 6 — one stable idempotency key: keyed to the join binding value,
      // create-only. A concurrent publisher loses cleanly; a re-publish of
      // the SAME qualified outcome replays as identical, never a second grant.
      const outcomeKey='tally:v2:task-outcome:'+targetIssue+':'+inp.join.valueSha256;
      const outcomeValue={
        schema:'tally/version-task-outcome/v1',phase:'QUALIFIED_OUTCOME_PUBLISHED',
        repository,targetIssue,issueRef:URL_PREFIX(owner,repo)+'issues/'+targetIssue,
        sourceSha:expectedSourceSha,
        fsw:{key:inp.fsw.key,revision:inp.fsw.revision,valueSha256:inp.fsw.valueSha256},
        join:{key:inp.join.key,revision:inp.join.revision,valueSha256:inp.join.valueSha256},
        validatorClaim:inp.validatorClaim,reviewerClaim:inp.reviewerClaim,
        validatorEvidence:inp.validatorEvidence,reviewDecision:inp.decision,
        sourcePack:inp.sourcePack,ci:inp.ci,hostFence:inp.hostFence,
        publisherWriterId:writerAuthority.writerId,publishedAt:clock(),
        distinctFrom:Object.freeze(['DISPATCH_CLAIMED_COMMENT_ACK','NATIVE_ISSUE_CLOSED',
          'NATIVE_DONE_LABEL','JOURNEY_PROJECTION','PR394_SOURCE_REPAIR_GREEN']),
      };
      let reservation=null;
      try{reservation=await store.put(outcomeKey,outcomeValue,{mustBeAbsent:true});}
      catch(e){
        // Distinguish real contention (CAS_EXISTS ⇒ someone else already
        // published the identical binding) from an ambiguous store failure:
        // only the former is a BLOCKED replay answer; the latter is UNKNOWN.
        if(e&&e.message==='CAS_EXISTS'){
          const existing=await store.get(outcomeKey);
          return blocked('OUTCOME_ALREADY_PUBLISHED',
            {outcomeKey,existingRevision:existing?.revision??null,
             retryPolicy:'IDENTICAL_REPLAY_READ_ONLY_NO_SECOND_GRANT'});
        }
        return unknown('OUTCOME_CAS_RESERVATION_UNKNOWN',
          {outcomeKey,reconciliation:'reconcileTaskOutcome()'});
      }
      if(!Number.isSafeInteger(reservation?.revision))
        return unknown('OUTCOME_CAS_RESERVATION_UNKNOWN',{outcomeKey});
      const proof=await store.get(outcomeKey);
      if(proof?.revision!==reservation.revision||
         JSON.stringify(proof?.value)!==JSON.stringify(outcomeValue))
        return unknown('OUTCOME_CAS_RESERVATION_UNVERIFIED',{outcomeKey});
      // Native reconciliation: the readback names the live ready state the
      // outcome was bound to; absence of drift is verified, not assumed.
      const after=await readIssue();
      if(!after)return unknown('OUTCOME_NATIVE_RECONCILIATION_FAILED',{outcomeKey});
      if(!after.labels.includes('state:ready'))
        return unknown('OUTCOME_NATIVE_RECONCILIATION_FAILED',
          {outcomeKey,detail:'labels drifted',labels:after.labels});
      return Object.freeze({status:'QUALIFIED_OUTCOME_PUBLISHED',
        reason:'HOST_CAS_READBACK_AND_NATIVE_RECONCILIATION_VERIFIED',
        repository,targetIssue,outcomeKey,revision:proof.revision,
        valueSha256:createHash('sha256').update(JSON.stringify(outcomeValue),'utf8')
          .digest('hex'),
        readyAtReadback:after.updatedAt,native:true,publication:'PUBLISHED',
        taskPass:true,releasePass:false,
        distinctFrom:outcomeValue.distinctFrom});
    }catch(e){
      const reason=providerReason(e);
      if(reason)return blocked(reason);
      return unknown('OUTCOME_PUBLISH_ACK_LOST',
        {retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_SECOND_PUBLISH',
         reconciliation:'reconcileTaskOutcome()'});
    }
  }

  /** Read-only classification after an UNKNOWN: reports the durable CAS row
   * and the live native state without ever writing. */
  async function reconcileTaskOutcome({joinValueSha256}={}){
    if(!DIGEST.test(joinValueSha256??''))
      return unknown('INVALID_RECONCILIATION_REQUEST');
    const outcomeKey='tally:v2:task-outcome:'+targetIssue+':'+joinValueSha256;
    let row=null;
    try{row=await store.get(outcomeKey);}catch{
      return unknown('OUTCOME_CAS_READ_FAILED',{outcomeKey});}
    let issue=null;
    try{issue=await readIssue();}catch{
      return unknown('ISSUE_READBACK_FAILED');}
    const phase=row?.value?.phase??'ABSENT';
    const ready=issue?issue.labels.includes('state:ready'):null;
    const status=phase==='QUALIFIED_OUTCOME_PUBLISHED'&&ready===true?
      'OUTCOME_PRESENT':phase==='ABSENT'&&ready===true?
        'READY_WITHOUT_OUTCOME':phase==='ABSENT'?'NOT_PUBLISHED':'UNRESOLVED';
    return Object.freeze({status,outcomeKey,casPhase:phase,
      nativeLabels:issue?Object.freeze(issue.labels):null,
      githubEffect:'NOT_RUN',retryPolicy:'NO_BLIND_SECOND_PUBLISH',
      taskPass:false,releasePass:false,publication:'HOLD'});
  }

  return Object.freeze({publishQualifiedOutcome,reconcileTaskOutcome});
}
