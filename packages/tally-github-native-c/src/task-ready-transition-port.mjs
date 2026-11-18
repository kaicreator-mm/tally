/**
 * I377-R93-E3 — forward-only planned→ready native transition port (task #377).
 *
 * Exactly ONE authorized GitHub label/state update per verified transition,
 * under a durable create-only effect fence; fail-closed everywhere. The E3
 * pack (#454 blob 5fbe4a67d6acc25e617e8ae306231e28c6719438) and the R90 role
 * order contract (#392/6983294817 §2, §3 E3 rows) govern:
 *   precheck — official live issue identity/labels, official blocked_by read
 *     (no OPEN blocker; closed blockers are admissible history and never
 *     predecessor PASS), absent unauthorized Claim/FSW rows in the trusted
 *     store, current source epoch when versionBranch is supplied;
 *   write — one issues.update PATCH swapping state:planned→state:ready in the
 *     full label array (no second PATCH after timeout or lost ACK);
 *   postread — the exact ready readback; successful readback alone grants no
 *     Claim, FSW or Outcome (authorized/admitted/taskPass stay false).
 * ABA between preflight and PATCH, concurrent issuers and crash/lost-ACK all
 * resolve to BLOCKED/UNKNOWN_PENDING_RECONCILIATION via the fence phases:
 * HELD (reserved, create-only) → RELEASED (verified postread). A fence left
 * HELD never times out and is reconciled read-only by reconcileTransitionState.
 * Authority is ONLY trusted Host configuration (writerAuthority, trusted store
 * and readers). No caller input, comment text or flag is ever write evidence.
 */
import {createHash,randomUUID} from 'node:crypto';

const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const providerReason=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===422?'GITHUB_VALIDATION_FAILED':null;
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const flags=()=>({authorized:false,admitted:false,taskPass:false});

export function createTaskReadyTransitionPort({
  repository,owner,repo,targetIssue,expectedSourceSha,versionBranch,
  octokit,store,writerAuthority,now
}={}){
  if(!REPO.test(repository??'')||repository!==(owner??'')+'/'+(repo??'')||
     !NAME.test(owner??'')||!NAME.test(repo??'')||
     !Number.isSafeInteger(targetIssue)||targetIssue<1||
     !SHA.test(expectedSourceSha??'')||
     (versionBranch!==undefined&&typeof versionBranch!=='string')||
     typeof store?.get!=='function'||typeof store?.put!=='function'||
     typeof octokit?.rest?.issues?.get!=='function'||
     typeof octokit?.rest?.issues?.update!=='function'||
     typeof octokit?.rest?.repos?.getBranch!=='function'||
     typeof octokit?.paginate?.iterator!=='function'||
     writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
     writerAuthority?.exclusiveHostControl!==true||
     writerAuthority?.repository!==repository||
     !/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/.test(writerAuthority?.writerId??'')||
     !LOGIN.test(writerAuthority?.githubWriterLogin??''))
    throw new TypeError('TRUSTED_TASK_READY_TRANSITION_PORT_REQUIRED');
  const clock=now??(()=>new Date().toISOString());
  const fenceKey='tally:v2:task-ready-fence:'+
    createHash('sha256').update(repository+'#'+targetIssue).digest('hex');
  const claimKeyFor=role=>'tally:v2:claim-admission:'+createHash('sha256')
    .update(repository+'#'+targetIssue+':'+role+':__default__').digest('hex');
  const fswKey='tally:v2:task-writeset:'+targetIssue;
  const blocked=(reason,more={})=>Object.freeze({status:'BLOCKED',reason,
    repository,targetIssue,githubEffect:'NOT_RUN',native:false,...flags(),...more});
  const unknown=(reason,more={})=>Object.freeze({status:'UNKNOWN',reason,
    repository,targetIssue,githubEffect:'UNKNOWN',native:false,
    reconciliationRequired:true,...flags(),...more});

  /** Live same-repository issue identity with normalized label names. */
  async function readIssue(){
    const data=(await octokit.rest.issues.get({owner,repo,issue_number:targetIssue}))?.data;
    if(!plain(data)||data.number!==targetIssue||data.pull_request!==undefined||
       data.repository_url!=='https://api.github.com/repos/'+owner+'/'+repo||
       !['open','closed'].includes(data.state)||
       typeof data.updated_at!=='string'||
       !Number.isSafeInteger(data.id)||data.id<1||!Array.isArray(data.labels))
      return null;
    const labels=data.labels.map(l=>typeof l==='string'?l:l?.name)
      .filter(x=>typeof x==='string');
    return {id:data.id,state:data.state,updatedAt:data.updated_at,labels};
  }

  /** Official GET blocked_by pagination with per-entry identity checks
   * (mirrors the frozen DAG writer's reader; malformed pages fail the read). */
  async function readBlockedBy(){
    const route='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by';
    const repoPrefix='https://github.com/'+owner+'/'+repo+'/issues/';
    const seen=new Set(),entries=[];
    for await(const page of octokit.paginate.iterator(route,
        {owner,repo,issue_number:targetIssue,per_page:100})){
      if(!Array.isArray(page?.data))return null;
      for(const d of page.data){
        if(!Number.isSafeInteger(d?.id)||d.id<1||
           !Number.isSafeInteger(d?.number)||d.number<1||
           typeof d.html_url!=='string'||!d.html_url.startsWith(repoPrefix)||
           !d.html_url.endsWith('/issues/'+d.number)||
           (d.state!=='open'&&d.state!=='closed')||seen.has(d.id))return null;
        seen.add(d.id);
        entries.push({id:d.id,number:d.number,state:d.state});
      }
    }
    return entries;
  }

  /** Forward-only planned→ready: precheck → fenced single PATCH → postread. */
  async function transitionToReady(){
    try{
      // Gate 1 — trusted pre-state labels: exactly the planned frontier
      // (the single workflow label must be state:planned; any other state:*
      // label means the issue is not at the planned frontier and a swap
      // would mint a double workflow state).
      const before=await readIssue();
      if(!before)return blocked('ISSUE_READBACK_FAILED');
      if(before.state!=='open')return blocked('ISSUE_NOT_OPEN',{state:before.state});
      if(before.labels.includes('state:ready'))
        return blocked('ALREADY_READY',{labels:before.labels});
      const stateLabels=before.labels.filter(l=>l.startsWith('state:'));
      if(!before.labels.includes('type:task')||
         stateLabels.length!==1||stateLabels[0]!=='state:planned')
        return blocked('PRESTATE_LABELS_INVALID',{labels:before.labels});
      // Gate 2 — official blocked_by: no OPEN blocker; closed history only.
      const blockers=await readBlockedBy();
      if(blockers===null)return blocked('GITHUB_DEPENDENCY_READ_FAILED');
      const openBlockers=blockers.filter(b=>b.state==='open');
      if(openBlockers.length)
        return blocked('NATIVE_ISSUE_DEPENDENCIES_BLOCKED',
          {openBlockerNumbers:Object.freeze(openBlockers.map(b=>b.number))});
      // Gate 3 — absent unauthorized Claim/FSW rows in the trusted store
      // (ready readback must never ride on a pre-existing admission row).
      for(const role of ['builder','validator','reviewer']){
        if(await store.get(claimKeyFor(role)))
          return blocked('UNAUTHORIZED_CLAIM_ROW_PRESENT',{role});
      }
      if(await store.get(fswKey))return blocked('UNAUTHORIZED_FSW_ROW_PRESENT');
      // Gate 4 — durable single-writer effect fence (create-only HELD).
      // Contention and crash both leave the existing row untouched: a HELD
      // fence never expires and is reconciled read-only, never reseeded.
      const held={schema:'tally/task-ready-transition-fence/v1',phase:'HELD',
        repository,targetIssue,sourceSha:expectedSourceSha,
        writerId:writerAuthority.writerId,nonce:randomUUID(),heldAt:clock()};
      let reservation=null;
      try{reservation=await store.put(fenceKey,held,{mustBeAbsent:true});}
      catch{
        const existing=await store.get(fenceKey);
        return blocked('TRANSITION_FENCE_CONTENTION',
          {fencePhase:existing?.value?.phase??'UNKNOWN',
           retryPolicy:'RECONCILE_EXISTING_FENCE_NO_RESEED'});
      }
      if(!Number.isSafeInteger(reservation?.revision))
        return unknown('TRANSITION_FENCE_RESERVATION_UNKNOWN');
      const proof=await store.get(fenceKey);
      if(proof?.revision!==reservation.revision||
         JSON.stringify(proof?.value)!==JSON.stringify(held))
        return unknown('TRANSITION_FENCE_RESERVATION_UNVERIFIED');
      try{
        // Gate 5 — in-fence re-read: identity/labels must match the pre-fence
        // read exactly (detectable ABA/drift refuses before the PATCH).
        const inside=await readIssue();
        if(!inside)return blocked('ISSUE_READBACK_FAILED');
        if(inside.updatedAt!==before.updatedAt||inside.state!==before.state||
           inside.labels.join('\u0000')!==before.labels.join('\u0000'))
          return blocked('ISSUE_STALE_VS_PREFLIGHT',
            {beforeUpdatedAt:before.updatedAt,insideUpdatedAt:inside.updatedAt});
        const insideBlockers=await readBlockedBy();
        if(insideBlockers===null)return blocked('GITHUB_DEPENDENCY_READ_FAILED');
        if(insideBlockers.some(b=>b.state==='open'))
          return blocked('NATIVE_ISSUE_DEPENDENCIES_BLOCKED');
        // Gate 6 — current source epoch on the live branch when supplied.
        if(versionBranch!==undefined){
          let current=null;
          try{current=(await octokit.rest.repos.getBranch(
            {owner,repo,branch:versionBranch}))?.data?.commit?.sha??null;}
          catch(e){
            const reason=providerReason(e);
            return blocked(reason??'VERSION_BRANCH_READ_FAILED');
          }
          if(current!==expectedSourceSha)return blocked('SOURCE_EPOCH_NOT_CURRENT',
            {expectedSourceSha,actualBranchHead:current});
        }
        // Write — exactly ONE full-label-array PATCH (planned swapped to ready).
        try{
          await octokit.rest.issues.update({owner,repo,issue_number:targetIssue,
            labels:before.labels.map(l=>l==='state:planned'?'state:ready':l)});
        }catch(e){
          const reason=providerReason(e);
          if(reason)return blocked(reason);
          // Lost/damaged ACK: the update may have applied; the fence stays
          // HELD and reconciliation is read-only. Never a second PATCH.
          return unknown('GITHUB_UPDATE_ACK_LOST',
            {retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_PATCH',
             reconciliation:'reconcileTransitionState()'});
        }
        // Postread — success requires the exact ready readback.
        const after=await readIssue();
        if(!after)return unknown('POSTREAD_FAILED');
        if(!after.labels.includes('state:ready')||after.labels.includes('state:planned'))
          return unknown('POSTREAD_NOT_READY',{labels:after.labels});
        // Release the fence with the verified outcome.
        const released={...held,phase:'RELEASED',readyAt:after.updated_at};
        const confirmation=await store.put(fenceKey,released,
          {mustMatchRevision:reservation.revision});
        const final=await store.get(fenceKey);
        if(!Number.isSafeInteger(confirmation?.revision)||
           final?.revision!==confirmation.revision||
           JSON.stringify(final?.value)!==JSON.stringify(released))
          return unknown('TRANSITION_FENCE_RELEASE_UNKNOWN');
        return Object.freeze({status:'READY_TRANSITION_VERIFIED',
          reason:'OFFICIAL_GITHUB_LABEL_READBACK_VERIFIED',repository,targetIssue,
          previousUpdatedAt:before.updatedAt,readyAt:after.updated_at,
          githubEffect:'PERFORMED_VERIFIED',writePerformed:true,
          fenceRevision:final.revision,...flags()});
      }catch{
        // Ambiguous in-fence failure: the fence stays HELD by design; the
        // next caller reconciles the shared physical rows.
        return unknown('TRANSITION_OPERATION_UNKNOWN');
      }
    }catch(e){
      return unknown(providerReason(e)??'TRANSITION_FAILED');
    }
  }

  /** Read-only reconciliation after an UNKNOWN: classifies the live label
   * state without ever PATCHing; a HELD fence is surfaced, never reset. */
  async function reconcileTransitionState(){
    let issue;
    try{issue=await readIssue();}catch{return unknown('ISSUE_READBACK_FAILED');}
    if(!issue)return unknown('ISSUE_READBACK_FAILED');
    const fence=await store.get(fenceKey);
    const ready=issue.labels.includes('state:ready')&&
      !issue.labels.includes('state:planned');
    return Object.freeze({status:ready?'READY_RECONCILED':'STILL_PLANNED',
      repository,targetIssue,labels:Object.freeze(issue.labels),
      updatedAt:issue.updatedAt,
      fencePhase:fence?.value?.phase??'ABSENT',githubEffect:'NOT_RUN',
      retryPolicy:'NO_BLIND_PATCH',...flags()});
  }

  return Object.freeze({transitionToReady,reconcileTransitionState});
}
