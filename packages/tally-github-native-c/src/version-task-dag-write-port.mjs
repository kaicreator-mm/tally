/**
 * E0 #284 — pre-freeze version Task DAG write port (selected C capability).
 *
 * Exactly two privileged, Host-bound operations:
 *   addNativeTaskDependency — official GitHub issue-dependencies blocked_by API
 *     only; pre-freeze only; one explicitly proposed PLANNED candidate-pack
 *     edge at a time; read-before/write/read-after; lost-ack reconciliation by
 *     re-reading, never a blind POST. No prose/label/YAML inference.
 *   freezeVersionTaskDag — consumes the A-owned closeout-roster
 *     createHostVersionTaskDagFreezeWriter READ-ONLY (never modified here),
 *     after refusing while any required candidate edge is still PLANNED, then
 *     materializes the exact frozen per-Task write-set digest records with
 *     equal-or-absent create-only CAS and full readback.
 *
 * Authority is ONLY trusted Host configuration (sealed Octokit, exclusive
 * writer authority, trusted CAS store and readers). No caller input, comment
 * text or flag is ever write evidence. The T06d root Claim publication is a
 * Host-process-only operation and is deliberately absent from this port.
 */
import {createHash} from 'node:crypto';
import {createHostVersionTaskDagFreezeWriter} from './closeout-roster.mjs';

const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const DIGEST=/^[0-9a-f]{64}$/;
const FREEZE_POLICIES=new Set(['ALL_TASKS','VERSION_ONLY']);
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const canonical=x=>Array.isArray(x)?x.map(canonical):
  plain(x)?Object.fromEntries(Object.keys(x).sort().map(k=>[k,canonical(x[k])])):x;
const same=(a,b)=>JSON.stringify(canonical(a))===JSON.stringify(canonical(b));
const hash=x=>createHash('sha256').update(x,'utf8').digest('hex');
const issueUrl=(repo,n)=>'https://github.com/'+repo+'/issues/'+n;
const immutableBlobRef=(value,repository)=>{
  if(typeof value!=='string')return false;
  const prefix='https://github.com/'+repository+'/blob/';
  if(!value.startsWith(prefix))return false;
  const rest=value.slice(prefix.length),slash=rest.indexOf('/');
  return slash===40&&SHA.test(rest.slice(0,slash))&&rest.length>slash+1;
};
const flags=()=>({authorized:false,admitted:false,taskPass:false,releasePass:false});
const providerReason=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===422?'GITHUB_VALIDATION_FAILED':null;

/** E0 candidate-pack membership/edge validation. Mirrors the A-owned
 * closeout-roster grammar read-only; acyclicity is an explicit separate
 * EDGE_CYCLE gate so a cyclic candidate is refused as a write, not silently
 * accepted because its shape parses. */
function validCandidatePack(pack,repository,versionId,versionSha,blobSha){
  if(!plain(pack)||pack.schema!=='tally/version-task-pack/v1'||
     pack.status!=='CANDIDATE_NOT_FROZEN'||pack.repository!==repository||
     pack.versionId!==versionId||pack.versionBaseSha!==versionSha||
     !SHA.test(blobSha??'')||!Array.isArray(pack.taskIssues)||
     !pack.taskIssues.length||pack.taskIssues.length>200||
     !Number.isFinite(Date.parse(pack.observedAt??''))||
     !Array.isArray(pack.edges)||pack.edges.length>2000)
    return false;
  const ids=new Set();
  for(const t of pack.taskIssues){
    if(!plain(t)||!Number.isSafeInteger(t.issueNumber)||t.issueNumber<1||
       ids.has(t.issueNumber)||typeof t.title!=='string'||
       !['open','closed'].includes(t.issueState)||
       !Number.isFinite(Date.parse(t.updatedAt??'')))return false;
    ids.add(t.issueNumber);
  }
  const seen=new Set();
  for(const e of pack.edges){
    if(!plain(e)||!ids.has(e.from)||!ids.has(e.to)||e.from===e.to||
       !['NATIVE','PLANNED'].includes(e.kind)||
       e.evidenceRef!==issueUrl(repository,e.to)||
       e.dependencyRef!==issueUrl(repository,e.from))return false;
    const key=e.from+'>'+e.to;
    if(seen.has(key))return false;
    seen.add(key);
  }
  return true;
}

/** Candidate graph (all kinds) plus the proposed edge must stay acyclic. */
function acyclicWith(edges,extraFrom,extraTo){
  const adj=new Map();
  const push=(from,to)=>{
    if(!adj.has(from))adj.set(from,[]);
    adj.get(from).push(to);
  };
  for(const e of edges)push(e.from,e.to);
  if(extraFrom!==undefined)push(extraFrom,extraTo);
  const done=new Set(),visiting=new Set();
  const walk=n=>{
    if(visiting.has(n))return false;
    if(done.has(n))return true;
    visiting.add(n);
    for(const next of adj.get(n)||[])if(!walk(next))return false;
    visiting.delete(n);done.add(n);return true;
  };
  for(const n of adj.keys())if(!walk(n))return false;
  return true;
}

/** Same-repository live issue identity: exact number, no PR, exact
 * repository_url, bounded state/updated_at. Stale detection binds the live
 * updated_at back to the candidate-pack snapshot. */
async function readIssueIdentity(octokit,owner,repo,issueNumber){
  const data=(await octokit.rest.issues.get({owner,repo,issue_number:issueNumber}))?.data;
  if(!plain(data)||data.number!==issueNumber||data.pull_request!==undefined||
     data.repository_url!=='https://api.github.com/repos/'+owner+'/'+repo||
     !['open','closed'].includes(data.state)||
     typeof data.updated_at!=='string'||!Number.isSafeInteger(data.id)||data.id<1)
    return null;
  return {id:data.id,state:data.state,updatedAt:data.updated_at};
}

/** Official GET blocked_by pagination with per-entry identity checks.
 * Entries outside the exact repository fail the read, not just the match. */
async function readBlockedBy(octokit,owner,repo,issueNumber){
  const route='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by';
  const repoPrefix='https://github.com/'+owner+'/'+repo+'/issues/';
  const seen=new Set(),entries=[];
  for await(const page of octokit.paginate.iterator(route,
      {owner,repo,issue_number:issueNumber,per_page:100})){
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

/** Create the E0 selected C write port. Every dependency is trusted Host
 * configuration; a missing or malformed binding refuses before any write. */
export function createVersionTaskDagWritePort({
  repository,versionId,expectedVersionSha,store,taskPackReader,
  reviewWitnessReader,ownerApprovalReader,nativeRosterReader,
  octokit,owner,repo,writerAuthority,writeSetDocumentByIssue,
  freezeWriteSetPolicy='ALL_TASKS',now
}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedVersionSha??'')||!NAME.test(owner??'')||
     !NAME.test(repo??'')||repository!==owner+'/'+repo||
     typeof store?.get!=='function'||typeof store?.put!=='function'||
     typeof taskPackReader!=='function'||
     typeof reviewWitnessReader!=='function'||
     typeof ownerApprovalReader!=='function'||
     typeof nativeRosterReader!=='function'||
     typeof octokit?.rest?.issues?.get!=='function'||
     typeof octokit?.paginate?.iterator!=='function'||
     typeof octokit?.request!=='function'||
     writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
     writerAuthority?.exclusiveHostControl!==true||
     writerAuthority?.repository!==repository||
     !FREEZE_POLICIES.has(freezeWriteSetPolicy)||
     !ID.test(writerAuthority?.writerId??'')||
     !LOGIN.test(writerAuthority?.githubWriterLogin??''))
    throw new TypeError('TRUSTED_VERSION_TASK_DAG_WRITE_PORT_REQUIRED');
  const clock=now??(()=>new Date().toISOString());
  const freezeWriter=createHostVersionTaskDagFreezeWriter({
    repository,versionId,expectedVersionSha,store,taskPackReader,
    nativeRosterReader,reviewWitnessReader,ownerApprovalReader,now});
  const versionDagKey='tally:v2:version-task-dag:'+hash(repository+'#'+versionId);
  const writeSetKey=n=>'tally:v2:task-writeset:'+n;
  const blocked=(reason,more={})=>Object.freeze({status:'BLOCKED',reason,
    repository,versionId,githubEffect:'NOT_RUN',native:false,...flags(),...more});
  const unknown=(reason,more={})=>Object.freeze({status:'UNKNOWN',reason,
    repository,versionId,githubEffect:'UNKNOWN',native:false,
    reconciliationRequired:true,...flags(),...more});

  /** Pre-freeze only: official blocked_by write for ONE edge explicitly
   * present as PLANNED in the exact reviewed candidate PackBlob. */
  async function addNativeTaskDependency(request){
    const {taskPackRef,taskPackBlobSha,blockedIssueNumber,blockedByIssueNumber}=
      plain(request)?request:{};
    if(!immutableBlobRef(taskPackRef,repository)||!SHA.test(taskPackBlobSha??'')||
       !Number.isSafeInteger(blockedIssueNumber)||blockedIssueNumber<1||
       !Number.isSafeInteger(blockedByIssueNumber)||blockedByIssueNumber<1)
      return blocked('EDGE_INPUT_INVALID');
    if(blockedIssueNumber===blockedByIssueNumber)
      return blocked('EDGE_INPUT_INVALID',{detail:'SELF_EDGE_REFUSED'});
    try{
      // Gate 1 — freeze finality: no dependency write after final freeze.
      const frozen=await store.get(versionDagKey);
      if(frozen)return blocked('VERSION_DAG_ALREADY_FROZEN');
      // Gate 2 — exact candidate PackBlob readback.
      const file=await taskPackReader({taskPackRef,expectedBlobSha:taskPackBlobSha});
      if(file?.status!=='VERIFIED'||file.blobSha!==taskPackBlobSha||
         typeof file.content!=='string')
        return blocked('TASK_PACK_READBACK_FAILED');
      let pack;
      try{pack=JSON.parse(file.content);}catch{return blocked('TASK_PACK_CONTENT_INVALID');}
      if(!validCandidatePack(pack,repository,versionId,expectedVersionSha,taskPackBlobSha))
        return blocked('TASK_PACK_CONTENT_INVALID');
      // Gate 3 — candidate-pack membership of BOTH endpoints.
      const members=new Set(pack.taskIssues.map(t=>t.issueNumber));
      if(!members.has(blockedIssueNumber)||!members.has(blockedByIssueNumber))
        return blocked('EDGE_ISSUE_NOT_IN_PACK');
      // Gate 4 — proposed-edge match: exactly one PLANNED candidate edge.
      const matches=pack.edges.filter(e=>e.from===blockedByIssueNumber&&
        e.to===blockedIssueNumber);
      if(matches.length!==1)return blocked('EDGE_NOT_IN_CANDIDATE_PACK');
      if(matches[0].kind==='NATIVE')return blocked('EDGE_ALREADY_NATIVE_IN_PACK');
      // Gate 4b — native dependency POST requires the SAME exact reviewed
      // and Owner-approved candidate PackBlob as the eventual DAG freeze.
      // Constructor-only injection of the readers is not authorization.
      const witnessScope={taskPackRef,taskPackBlobSha,versionId,
        sourceSha:expectedVersionSha};
      const [review,approval]=await Promise.all([
        reviewWitnessReader(witnessScope),ownerApprovalReader(witnessScope)
      ]);
      if(review?.status!=='INDEPENDENTLY_VERIFIED'||
         review.taskPackBlobSha!==taskPackBlobSha||
         review.sourceSha!==expectedVersionSha||
         review.reviewedHeadSha!==expectedVersionSha||
         !ID.test(review.reviewer??'')||!ID.test(review.author??'')||
         review.reviewer===review.author||
         typeof review.reviewRef!=='string'||
         !review.reviewRef.startsWith('https://github.com/'+repository+'/')||
         !Number.isFinite(Date.parse(review.verifiedAt??'')))
        return blocked('INDEPENDENT_REVIEW_REQUIRED');
      if(approval?.status!=='OWNER_APPROVED'||
         approval.taskPackBlobSha!==taskPackBlobSha||
         approval.sourceSha!==expectedVersionSha||
         approval.owner!==owner||
         typeof approval.approvalRef!=='string'||
         !approval.approvalRef.startsWith('https://github.com/'+repository+'/')||
         !Number.isFinite(Date.parse(approval.approvedAt??'')))
        return blocked('OWNER_APPROVAL_REQUIRED');
      // Gate 5 — cycle refusal over the candidate graph (the proposed edge is
      // part of that graph; any cycle containing it refuses here).
      if(!acyclicWith(pack.edges,blockedByIssueNumber,blockedIssueNumber)||
         !acyclicWith(pack.edges))
        return blocked('EDGE_CYCLE');
      // Gate 6 — same-repository live identity + staleness vs the pack.
      let blockedIssue,blockerIssue;
      try{
        [blockedIssue,blockerIssue]=await Promise.all([
          readIssueIdentity(octokit,owner,repo,blockedIssueNumber),
          readIssueIdentity(octokit,owner,repo,blockedByIssueNumber)]);
      }catch(e){
        return blocked(providerReason(e)??'ISSUE_READBACK_FAILED');
      }
      if(!blockedIssue||!blockerIssue)return blocked('ISSUE_READBACK_FAILED');
      const snapBlocked=pack.taskIssues.find(t=>t.issueNumber===blockedIssueNumber);
      const snapBlocker=pack.taskIssues.find(t=>t.issueNumber===blockedByIssueNumber);
      if(snapBlocked.updatedAt!==blockedIssue.updatedAt||
         snapBlocked.issueState!==blockedIssue.state||
         snapBlocker.updatedAt!==blockerIssue.updatedAt||
         snapBlocker.issueState!==blockerIssue.state)
        return blocked('ISSUE_STALE_VS_PACK');
      // Gate 7 — idempotent read-before; an exact native edge never re-POSTs.
      let before;
      try{before=await readBlockedBy(octokit,owner,repo,blockedIssueNumber);}
      catch(e){return blocked(providerReason(e)??'GITHUB_DEPENDENCY_READ_FAILED');}
      if(before===null)return blocked('GITHUB_DEPENDENCY_READ_FAILED');
      const present=before.find(d=>d.number===blockedByIssueNumber);
      if(present)return Object.freeze({status:'NATIVE_EDGE_PRESENT',
        reason:'NATIVE_EDGE_ALREADY_PRESENT_READBACK',repository,versionId,
        blockedIssueNumber,blockedByIssueNumber,dependencyId:present.id,
        native:true,githubEffect:'NOT_RUN',writePerformed:false,...flags()});
      // Write — official blocked_by endpoint, blocker bound by live issue id.
      try{
        await octokit.request(
          'POST /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by',
          {owner,repo,issue_number:blockedIssueNumber,issue_id:blockerIssue.id});
      }catch(e){
        const reason=providerReason(e);
        if(reason)return blocked(reason);
        // Lost/damaged ACK: the edge may exist; reconcile by readback only.
        return unknown('NATIVE_WRITE_ACK_LOST',{blockedIssueNumber,
          blockedByIssueNumber,retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_POST'});
      }
      // Read-after: success requires the exact native edge to read back.
      let after;
      try{after=await readBlockedBy(octokit,owner,repo,blockedIssueNumber);}
      catch{return unknown('NATIVE_READBACK_MISMATCH',{blockedIssueNumber,
        blockedByIssueNumber,retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_POST'});}
      const edge=after===null?null:after.find(d=>d.number===blockedByIssueNumber);
      if(!edge)return unknown('NATIVE_READBACK_MISMATCH',{blockedIssueNumber,
        blockedByIssueNumber,retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_POST'});
      return Object.freeze({status:'NATIVE_EDGE_VERIFIED',
        reason:'OFFICIAL_GITHUB_BLOCKED_BY_READBACK_VERIFIED',repository,versionId,
        blockedIssueNumber,blockedByIssueNumber,dependencyId:edge.id,
        native:true,githubEffect:'PERFORMED_VERIFIED',writePerformed:true,...flags()});
    }catch(e){
      return unknown(providerReason(e)??'NATIVE_DEPENDENCY_WRITE_FAILED');
    }
  }

  /** Freeze only after A refreshed every required candidate edge to NATIVE.
   * The A-owned writer verifies blob immutability, full native roster/edges,
   * independent review, Owner approval and create-only CAS + readback; E0
   * adds the PLANNED-edges gate and the frozen write-set record materialization. */
  async function freezeVersionTaskDag(request){
    const {taskPackRef,taskPackBlobSha}=plain(request)?request:{};
    if(!immutableBlobRef(taskPackRef,repository)||!SHA.test(taskPackBlobSha??''))
      return blocked('TASK_PACK_IDENTITY_INVALID');
    try{
      const file=await taskPackReader({taskPackRef,expectedBlobSha:taskPackBlobSha});
      if(file?.status!=='VERIFIED'||file.blobSha!==taskPackBlobSha||
         typeof file.content!=='string')
        return blocked('TASK_PACK_READBACK_FAILED');
      let pack;
      try{pack=JSON.parse(file.content);}catch{return blocked('TASK_PACK_CONTENT_INVALID');}
      if(!validCandidatePack(pack,repository,versionId,expectedVersionSha,taskPackBlobSha))
        return blocked('TASK_PACK_CONTENT_INVALID');
      if(!acyclicWith(pack.edges))return blocked('EDGE_CYCLE');
      if(pack.edges.some(e=>e.kind==='PLANNED'))
        return blocked('PLANNED_EDGES_REMAIN',{plannedEdgeCount:
          pack.edges.filter(e=>e.kind==='PLANNED').length});
      // Owner-directed Core MVP successor policy. Structural DAG Freeze is
      // NOT a blanket per-Task FSW admission: closed historical/not_planned
      // Tasks must never receive invented frozen write-set rows. The mode is
      // trusted Host construction only (never supplied by HTTP/request).
      // Future Task write authority still requires its own accepted Claim/FSW.
      if(freezeWriteSetPolicy==='VERSION_ONLY'){
        const frozen=await freezeWriter.freezeVersionTaskDag({taskPackRef,taskPackBlobSha});
        if(frozen?.status==='BLOCKED'||frozen?.status==='UNKNOWN')
          return Object.freeze({...frozen,repository,versionId,
            githubEffect:'NOT_RUN',...flags()});
        if(frozen?.status!=='FROZEN'&&frozen?.status!=='REPLAY')
          return unknown('VERSION_DAG_FREEZE_FAILED');
        return Object.freeze({status:frozen.status,
          reason:frozen.status==='FROZEN'?
            'FROZEN_VERSION_TASK_DAG_CAS_READBACK_VERIFIED':'IDENTICAL_FREEZE_REPLAY',
          repository,versionId,taskPackRef,taskPackBlobSha,blobSha:taskPackBlobSha,
          frozenAt:frozen.frozenAt,taskCount:pack.taskIssues.length,
          edgeCount:pack.edges.length,writeSetRecordCount:0,
          writeSetMaterialization:'DEFERRED_TO_PER_TASK_CLAIM_FSW',
          taskDispatchAuthorized:false,githubEffect:'NOT_RUN',
          dependencyWritesAfterFreeze:'REFUSED',...flags()});
      }
      // Host-configured frozen write-set digest documents: one per Task Issue,
      // validated in full BEFORE the first CAS write (no partial acceptance).
      const documents=writeSetDocumentByIssue??{};
      const rows=[];
      for(const t of pack.taskIssues){
        const entry=documents[String(t.issueNumber)];
        if(!plain(entry)||!DIGEST.test(entry.writeSetSha256??'')||
           typeof entry.document!=='string'||entry.document.length===0||
           entry.document.length>65536)
          return blocked('FREEZE_WRITESET_CONFIG_INVALID',
            {missingIssueNumber:t.issueNumber});
        rows.push({issueNumber:t.issueNumber,writeSetSha256:entry.writeSetSha256,
          document:entry.document});
      }
      const rowValue=(row,frozenAt)=>({schema:'tally/version-task-writeset/v1',
        phase:'FROZEN_DAG_WRITESET',repository,issueNumber:row.issueNumber,
        sourceSha:expectedVersionSha,writeSetSha256:row.writeSetSha256,
        document:row.document,versionPackBlobSha:taskPackBlobSha,
        versionTaskDagKey:versionDagKey,frozenAt});
      // Preflight every per-Task record equal-or-absent BEFORE the immutable
      // freeze CAS: a divergent existing row must block before any new write.
      for(const row of rows){
        const existing=await store.get(writeSetKey(row.issueNumber));
        // frozenAt is set once at creation; equality binds every other field.
        if(existing&&!same(existing.value,
            rowValue(row,existing.value?.frozenAt)))
          return blocked('FREEZE_WRITESET_CONFLICT',
            {conflictIssueNumber:row.issueNumber});
      }
      // Delegate the immutable freeze to the A-owned writer, read-only.
      const frozen=await freezeWriter.freezeVersionTaskDag({taskPackRef,taskPackBlobSha});
      if(frozen?.status==='BLOCKED'||frozen?.status==='UNKNOWN')
        return Object.freeze({...frozen,repository,versionId,
          githubEffect:'NOT_RUN',...flags()});
      if(frozen?.status!=='FROZEN'&&frozen?.status!=='REPLAY')
        return unknown('VERSION_DAG_FREEZE_FAILED');
      // Materialize per-Task FROZEN_DAG_WRITESET records: equal-or-absent
      // create-only CAS with exact readback; partial failure stays UNKNOWN.
      const failed=[];
      for(const row of rows){
        const key=writeSetKey(row.issueNumber);
        const value=rowValue(row,frozen.frozenAt);
        try{
          const existing=await store.get(key);
          if(existing){
            if(!same(existing.value,value))
              return blocked('FREEZE_WRITESET_CONFLICT',
                {conflictIssueNumber:row.issueNumber});
            continue;
          }
          await store.put(key,value,{mustBeAbsent:true});
          const readback=await store.get(key);
          if(!readback||!same(readback.value,value))failed.push(row.issueNumber);
        }catch{
          // Partial write after the immutable freeze: reconcile, never reseed.
          failed.push(row.issueNumber);
        }
      }
      if(failed.length)return unknown('FREEZE_WRITESET_PARTIAL_UNKNOWN',
        {failedIssueNumbers:Object.freeze([...failed]),
          reconciledIssueNumbers:Object.freeze(
            rows.map(r=>r.issueNumber).filter(n=>!failed.includes(n)))});
      return Object.freeze({status:frozen.status,
        reason:frozen.status==='FROZEN'?
          'FROZEN_VERSION_TASK_DAG_CAS_READBACK_VERIFIED':'IDENTICAL_FREEZE_REPLAY',
        repository,versionId,taskPackRef,taskPackBlobSha,
        blobSha:taskPackBlobSha,frozenAt:frozen.frozenAt,
        taskCount:pack.taskIssues.length,edgeCount:pack.edges.length,
        writeSetRecordCount:rows.length,githubEffect:'NOT_RUN',
        dependencyWritesAfterFreeze:'REFUSED',...flags()});
    }catch(e){
      return unknown(providerReason(e)??'VERSION_DAG_FREEZE_FAILED');
    }
  }

  return Object.freeze({addNativeTaskDependency,freezeVersionTaskDag});
}

/** Read-only observation: has the version Task DAG already been frozen in
 * this store? Used by Host wiring to advertise the pre-freeze-only window. */
export async function readVersionTaskDagFreezeState({store,repository,versionId}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     typeof store?.get!=='function')
    throw new TypeError('TRUSTED_VERSION_TASK_DAG_STATE_READER_REQUIRED');
  const row=await store.get('tally:v2:version-task-dag:'+hash(repository+'#'+versionId));
  if(!row)return Object.freeze({status:'NOT_FROZEN',repository,versionId,...flags()});
  const v=row.value;
  if(v?.schema==='tally/version-task-dag/v1'&&v.phase==='FROZEN_VERSION_TASK_DAG'&&
     SHA.test(v.blobSha??''))
    return Object.freeze({status:'FROZEN',repository,versionId,
      blobSha:v.blobSha,frozenAt:v.frozenAt??null,...flags()});
  return Object.freeze({status:'UNKNOWN',repository,versionId,...flags()});
}
