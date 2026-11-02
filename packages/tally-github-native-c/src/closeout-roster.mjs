import {createHash} from 'node:crypto';
import {parseDocument,visit,isAlias} from 'yaml';
import {readClaimEventCandidatesFromClient} from './task-claim-events.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from './task-claim-host-readback-port.mjs';
import {readUiNativeVersionFromClient} from './ui-version-readonly.mjs';

const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions','woodpecker','other']);
const OWNER_AUTHORITY_ID=6574693761;
const SCOPE_ISSUE=260;
const SCOPE_COMMENT_ID=6572687940;
const WEB_OPERATOR='chatgpt-web:tally-v2';
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const canonical=x=>Array.isArray(x)?x.map(canonical):
  plain(x)?Object.fromEntries(Object.keys(x).sort().map(k=>[k,canonical(x[k])])):x;
const same=(a,b)=>JSON.stringify(canonical(a))===JSON.stringify(canonical(b));
const hash=x=>createHash('sha256').update(x,'utf8').digest('hex');
const claimKey=(repo,n,role)=>'tally:v2:claim-admission:'+hash(repo+'#'+n+':'+role+':__default__');
const issueUrl=(repo,n)=>'https://github.com/'+repo+'/issues/'+n;
const commentUrl=(repo,n,id)=>issueUrl(repo,n)+'#issuecomment-'+id;
const apiIssueUrl=(owner,repo,n)=>'https://api.github.com/repos/'+owner+'/'+repo+'/issues/'+n;
const immutableBlobRef=(value,repository)=>{
  if(typeof value!=='string')return false;
  const prefix='https://github.com/'+repository+'/blob/';
  if(!value.startsWith(prefix))return false;
  const rest=value.slice(prefix.length),slash=rest.indexOf('/');
  return slash===40&&SHA.test(rest.slice(0,slash))&&rest.length>slash+1;
};
const denied=reason=>({connected:false,reason,rosterCoverage:'UNVERIFIED',
  claimAccepted:false,authorized:false,admitted:false,taskPass:false,releasePass:false});

function parseYaml(body){
  if(typeof body!=='string'||body.length>256*1024)return null;
  const matches=[...body.matchAll(/^\x60\x60\x60yaml\s*\r?\n([\s\S]*?)\r?\n\x60\x60\x60/gm)];
  if(matches.length!==1)return null;
  try{
    const doc=parseDocument(matches[0][1],{uniqueKeys:true,merge:false,
      schema:'core',maxAliasCount:0,strict:true});
    if(doc.errors.length||doc.warnings.length)return null;
    visit(doc,{Node(_key,node){if(isAlias(node)||node.anchor!==undefined||node.tag!==undefined)
      throw new Error('NON_CANONICAL_YAML');}});
    const value=doc.toJS({maxAliasCount:0});
    return plain(value)?value:null;
  }catch{return null;}
}
async function exactComment({octokit,owner,repo,issueNumber,commentId,login}){
  try{
    const params={owner,repo,comment_id:commentId};
    const first=(await octokit.rest.issues.getComment(params))?.data;
    const second=(await octokit.rest.issues.getComment(params))?.data;
    const valid=c=>c?.id===commentId&&
      c.html_url===commentUrl(owner+'/'+repo,issueNumber,commentId)&&
      c.issue_url===apiIssueUrl(owner,repo,issueNumber)&&c.user?.login===login&&
      typeof c.body==='string'&&c.body.length<=256*1024&&
      typeof c.created_at==='string'&&c.created_at===c.updated_at;
    if(!valid(first)||!valid(second)||first.updated_at!==second.updated_at||
       first.body!==second.body||first.user.login!==second.user.login)return null;
    return first;
  }catch{return null;}
}

/** Verify Owner authority, shared scope, owner handoff and Web Builder receipt. */
export function createHostResponsibilityHandoffReaderFromClient({
  octokit,owner,repo,writerLogin,handoffByIssue,
  authorityCommentId=OWNER_AUTHORITY_ID,scopeIssueNumber=SCOPE_ISSUE,
  scopeCommentId=SCOPE_COMMENT_ID
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||!LOGIN.test(writerLogin??'')||
     !octokit?.rest?.issues?.getComment||!plain(handoffByIssue))
    throw new TypeError('TRUSTED_HANDOFF_READER_REQUIRED');
  const repository=owner+'/'+repo;
  async function readResponsibilityHandoff({issueNumber,claimSubject}={}){
    const cfg=handoffByIssue[String(issueNumber)];
    if(!cfg||!claimSubject||claimSubject.issueNumber!==issueNumber)
      return {status:'NOT_VERIFIED',reason:'HANDOFF_CONFIG_MISSING'};
    try{
      const [authority,scope,handoff,ack,correction]=await Promise.all([
        exactComment({octokit,owner,repo,issueNumber:259,commentId:authorityCommentId,login:owner}),
        exactComment({octokit,owner,repo,issueNumber:scopeIssueNumber,commentId:scopeCommentId,login:owner}),
        exactComment({octokit,owner,repo,issueNumber,commentId:cfg.handoffCommentId,login:owner}),
        exactComment({octokit,owner,repo,issueNumber,commentId:cfg.receiptCommentId,login:writerLogin}),
        cfg.scopeCorrectionCommentId?exactComment({octokit,owner,repo,issueNumber,
          commentId:cfg.scopeCorrectionCommentId,login:writerLogin}):Promise.resolve(null)
      ]);
      if(!authority||!scope||!handoff||!ack||
         !authority.body.includes('Owner-authorized responsibility handoff')||
         !authority.body.includes('Receiving Builder')||!authority.body.includes(WEB_OPERATOR)||
         !authority.body.includes(claimSubject.dispatchId)||
         !authority.body.includes(String(claimSubject.commentId))||
         !scope.body.includes('frozen write sets')||
         !scope.body.includes('A=7, B=10, C=12, D=7'))
        return {status:'NOT_VERIFIED',reason:'OWNER_SCOPE_READBACK_FAILED'};
      const h=parseYaml(handoff.body),a=parseYaml(ack.body);
      const parentRef='#'+issueNumber+'@'+claimSubject.commentId;
      if(!h||h.schema!=='tally/responsibility-handoff/v1'||
         h.responsibility_mode!=='RESPONSIBILITY_HANDOFF'||
         h.repository!==repository||h.task!=='#'+issueNumber||h.issue!=='#'+issueNumber||
         h.source_sha!==claimSubject.sourceSha||h.parent_dispatch_ref!==parentRef||
         h.parent_dispatch_id!==claimSubject.dispatchId||
         h.from_operator_id!=='zcode:i259:closeout:1'||h.to_operator_id!==WEB_OPERATOR||
         h.to_operator_kind!=='chatgpt-web'||h.to_actor_role!=='builder'||
         h.execution_environment!=='WEB'||h.scheduler_origin!=='WEB'||
         !ID.test(h.handoff_dispatch_id??'')||
         h.protected_claim_key!==claimSubject.protectedClaimKey||
         h.admission_generation!==claimSubject.generation||
         h.authority_ref!=='#259@'+authorityCommentId)
        return {status:'NOT_VERIFIED',reason:'OWNER_HANDOFF_MISMATCH'};
      const expectedScope='#'+scopeIssueNumber+'@'+scopeCommentId;
      let canonicalScope=expectedScope;
      if(h.scope_ref!==expectedScope){
        const c=correction?parseYaml(correction.body):null;
        canonicalScope='https://github.com/'+repository+'/issues/'+scopeIssueNumber+
          '#issuecomment-'+scopeCommentId;
        if(!c||c.schema!=='tally/responsibility-handoff-scope-ref-correction/v1'||
           c.task!=='#'+issueNumber||
           c.original_handoff_ref!=='#'+issueNumber+'@'+cfg.handoffCommentId||
           c.original_authority_ref!=='#259@'+authorityCommentId||
           c.recorded_scope_ref!==h.scope_ref||
           c.canonical_scope_ref!==canonicalScope||
           c.canonical_scope_issue!==scopeIssueNumber||
           c.canonical_scope_comment_id!==scopeCommentId||
           c.scope_is_shared_across_lanes!==true||c.scope_is_unchanged!==true||
           c.operator_id!==WEB_OPERATOR||c.transport_actor!==writerLogin)
          return {status:'NOT_VERIFIED',reason:'SCOPE_REFERENCE_CORRECTION_INVALID'};
      }
      if(!a||a.schema!=='tally/responsibility-handoff-ack/v1'||
         a.repository!==repository||a.task!=='#'+issueNumber||
         a.source_sha!==claimSubject.sourceSha||a.parent_dispatch_ref!==parentRef||
         a.parent_dispatch_id!==claimSubject.dispatchId||
         a.handoff_ref!=='#'+issueNumber+'@'+cfg.handoffCommentId||
         a.handoff_dispatch_id!==h.handoff_dispatch_id||
         a.protected_claim_key!==claimSubject.protectedClaimKey||
         a.admission_generation!==claimSubject.generation||
         a.authority_ref!=='#259@'+authorityCommentId||a.scope_ref!==h.scope_ref||
         a.receiver?.operator_id!==WEB_OPERATOR||a.receiver?.operator_kind!=='chatgpt-web'||
         a.receiver?.actor_role!=='builder'||a.receiver?.execution_environment!=='WEB'||
         a.receiver?.scheduler_origin!=='WEB'||a.transport_actor!==writerLogin||
         a.received!==true)
        return {status:'NOT_VERIFIED',reason:'WEB_RECEIPT_MISMATCH'};
      return {status:'VERIFIED',activeOperator:{operatorId:WEB_OPERATOR,
        operatorKind:'chatgpt-web',actorRole:'builder',executionEnvironment:'WEB'},
        authorityRef:'#259@'+authorityCommentId,scopeRef:canonicalScope,
        handoffRef:commentUrl(repository,issueNumber,cfg.handoffCommentId),
        receiptRef:commentUrl(repository,issueNumber,cfg.receiptCommentId)};
    }catch{return {status:'NOT_VERIFIED',reason:'HANDOFF_READ_FAILED'};}
  }
  return Object.freeze({readResponsibilityHandoff});
}

/** Read a CAS Claim only when its durable row and exact native ADS comment agree. */
export function createHostCloseoutClaimReadbackPortFromClient({
  octokit,owner,repo,writerLogin,writerId,store,handoffReader,
  repository,versionId,expectedVersionSha
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||!LOGIN.test(writerLogin??'')||
     !ID.test(writerId??'')||typeof store?.get!=='function'||
     repository!==owner+'/'+repo||!REPO.test(repository??'')||
     !VERSION.test(versionId??'')||!SHA.test(expectedVersionSha??''))
    throw new TypeError('TRUSTED_CLAIM_READBACK_REQUIRED');
  const native=createHostNativeGitHubClaimReadbackPortFromClient({octokit,owner,repo,writerLogin});
  const notVerified=(a,reason,more={})=>({connected:true,
    source:{repository,sourceSha:a.claimSourceSha,currentness:'PINNED',readback:'VERIFIED'},
    versionId,versionSourceSha:expectedVersionSha,issueNumber:a.issueNumber,
    claimSourceSha:a.claimSourceSha,taskPackRef:a.taskPackRef,
    taskPackBlobSha:a.taskPackBlobSha,claimStatus:'NOT_VERIFIED',
    claimAccepted:false,claimPublication:more.commentId?'CANDIDATE_ONLY':'NOT_VERIFIED',reason,
    authorized:false,admitted:false,taskPass:false,releasePass:false,...more});
  const notClaimed=a=>({connected:true,
    source:{repository,sourceSha:a.claimSourceSha,currentness:'PINNED',readback:'VERIFIED'},
    versionId,versionSourceSha:expectedVersionSha,issueNumber:a.issueNumber,
    claimSourceSha:a.claimSourceSha,taskPackRef:a.taskPackRef,
    taskPackBlobSha:a.taskPackBlobSha,claimStatus:'NOT_CLAIMED',
    claimAccepted:false,claimPublication:'NOT_VERIFIED',commentId:null,commentRef:null,
    authorized:false,admitted:false,taskPass:false,releasePass:false});
  async function readAcceptedClaim(a={}){
    if(a.repository!==repository||a.versionId!==versionId||
       a.versionSourceSha!==expectedVersionSha||
       !Number.isSafeInteger(a.issueNumber)||a.issueNumber<1||
       !SHA.test(a.claimSourceSha??'')||
       !immutableBlobRef(a.taskPackRef,repository)||
       !/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/blob\/[a-f0-9]{40}\//.test(a.taskPackRef??'')||
       !SHA.test(a.taskPackBlobSha??'')||!plain(a.claimSubject)||
       a.claimSubject.issueNumber!==a.issueNumber||
       a.claimSubject.sourceSha!==a.claimSourceSha||
       !ROLES.has(a.claimSubject.actorRole)||
       !['OBSERVED_COMMENT_SELF_REPORTED_CLAIM','NO_CLAIM_EVENT_EXPECTED'].includes(a.claimSubject.candidateStatus))
      return denied('CLAIM_READBACK_INPUT_INVALID');
    const s=a.claimSubject,role=s.actorRole;
    if(!ROLES.has(role)||s.protectedClaimKey!==repository+'#'+a.issueNumber+':'+role+':__default__'||
       (s.candidateStatus==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
        (!Number.isSafeInteger(s.generation)||s.generation!==1||
         !KINDS.has(s.operatorKind)||!ID.test(s.operatorId??'')||
         !ID.test(s.dispatchId??'')||!Number.isSafeInteger(s.commentId)||s.commentId<1)))
      return notVerified(a,'CLAIM_SUBJECT_INVALID');
    const key=claimKey(repository,a.issueNumber,role);
    try{
      const before=await store.get(key);
      const events=await readClaimEventCandidatesFromClient({octokit,owner,repo,
        issueNumber:a.issueNumber,expectedSourceSha:a.claimSourceSha});
      if(!before){
        const after=await store.get(key);
        if(after!==null&&after!==undefined)return notVerified(a,'CLAIM_CHANGED_DURING_READ');
        if(events?.reason==='NO_DISPATCH_CLAIM_EVENTS'&&events.candidateCount===0){
          if(s.candidateStatus==='NO_CLAIM_EVENT_EXPECTED')return notClaimed(a);
          return notVerified(a,'EXPECTED_CLAIM_EVENT_MISSING');
        }
        const c=events?.candidates?.find(x=>x.protectedClaimKey===s.protectedClaimKey);
        return notVerified(a,'NATIVE_EVENT_WITHOUT_CAS_ROW',c?{
          commentId:c.commentId,commentRef:c.commentRef,
          commentBodySha256:c.commentBodySha256,dispatchId:c.dispatchId,
          actorRole:c.actorRole,operatorKind:c.operatorKind,operatorId:c.operatorId,
          generation:c.claimGeneration,protectedClaimKey:c.protectedClaimKey,
          transportActor:c.transportActor,sourceClass:c.sourceClass}: {});
      }
      if(s.candidateStatus!=='OBSERVED_COMMENT_SELF_REPORTED_CLAIM')
        return notVerified(a,'UNEXPECTED_CAS_ROW_FOR_UNCLAIMED_TASK');
      const row=before.value;
      if(before.revision!==1||row?.phase!=='PUBLISHED_VERIFIED'||
         row.repository!==repository||row.issueRef!==issueUrl(repository,a.issueNumber)||
         row.sourceSha!==a.claimSourceSha||row.actorRole!==role||
         row.operatorKind!==s.operatorKind||row.operatorId!==s.operatorId||
         row.dispatchId!==s.dispatchId||row.generation!==s.generation||
         row.protectedClaimKey!==s.protectedClaimKey||row.writerId!==writerId||
         row.commentId!==s.commentId||
         row.commentRef!==commentUrl(repository,a.issueNumber,row.commentId))
        return notVerified(a,'CAS_ROW_NOT_ACCEPTED_OR_MISMATCHED');
      const matches=events?.candidates?.filter(x=>x.protectedClaimKey===row.protectedClaimKey);
      if(events?.reason!=='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'||
         !Array.isArray(matches)||matches.length!==1||
         matches[0].commentId!==row.commentId||matches[0].dispatchId!==row.dispatchId||
         matches[0].actorRole!==row.actorRole||matches[0].operatorKind!==row.operatorKind||
         matches[0].operatorId!==row.operatorId||
         matches[0].claimGeneration!==row.generation||
         matches[0].protectedClaimKey!==row.protectedClaimKey||
         matches[0].sourceClass!=='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'||
         matches[0].claimAccepted!==false||
         matches[0].transportActor!==writerLogin||
         matches[0].sourceSha!==row.sourceSha)
        return notVerified(a,'NATIVE_EVENT_DOES_NOT_MATCH_CAS_ROW');
      const reservation={repository,issueNumber:a.issueNumber,issueRef:row.issueRef,
        sourceSha:row.sourceSha,actorRole:row.actorRole,operatorKind:row.operatorKind,
        operatorId:row.operatorId,dispatchId:row.dispatchId,generation:row.generation,
        protectedClaimKey:row.protectedClaimKey};
      const witness=await native.readPublishedClaim(reservation,row.commentId);
      if(witness?.sourceKind!=='NATIVE_GITHUB_EXACT_READBACK'||
         witness.complete!==true||witness.edited!==false||
         witness.issueRef!==row.issueRef||witness.sourceSha!==row.sourceSha||
         witness.event!=='DISPATCH_CLAIMED'||witness.dispatchState!=='CLAIMED'||
         witness.dispatchId!==row.dispatchId||witness.actorRole!==row.actorRole||
         witness.operatorKind!==row.operatorKind||witness.operatorId!==row.operatorId||
         witness.generation!==row.generation||
         witness.protectedClaimKey!==row.protectedClaimKey||
         witness.commentId!==row.commentId||witness.commentRef!==row.commentRef||
         witness.transportActor!==writerLogin||
         witness.claimAccepted!==false||witness.authorized!==false||witness.admitted!==false)
        return notVerified(a,'NATIVE_COMMENT_READBACK_FAILED');
      const eventsAfter=await readClaimEventCandidatesFromClient({octokit,owner,repo,
        issueNumber:a.issueNumber,expectedSourceSha:a.claimSourceSha});
      const after=await store.get(key);
      const last=eventsAfter?.candidates?.filter(x=>x.protectedClaimKey===row.protectedClaimKey);
      if(!after||after.revision!==before.revision||!same(after.value,before.value)||
         eventsAfter?.reason!=='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'||
         !Array.isArray(last)||last.length!==1||
         last[0].commentBodySha256!==matches[0].commentBodySha256||
         last[0].claimGeneration!==row.generation||
         last[0].protectedClaimKey!==row.protectedClaimKey)
        return notVerified(a,'CLAIM_CHANGED_DURING_READ');
      let handoff={status:'NOT_VERIFIED'};
      if(typeof handoffReader?.readResponsibilityHandoff==='function')
        handoff=await handoffReader.readResponsibilityHandoff({issueNumber:a.issueNumber,claimSubject:s});
      return {connected:true,source:{repository,sourceSha:row.sourceSha,
          currentness:'PINNED',readback:'VERIFIED'},
        versionId,versionSourceSha:expectedVersionSha,issueNumber:a.issueNumber,
        claimSourceSha:row.sourceSha,taskPackRef:a.taskPackRef,
        taskPackBlobSha:a.taskPackBlobSha,claimStatus:'ACCEPTED',claimAccepted:true,
        claimPublication:'REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK',
        generation:row.generation,actorRole:row.actorRole,operatorKind:row.operatorKind,
        operatorId:row.operatorId,writerId:row.writerId,
        protectedClaimKey:row.protectedClaimKey,dispatchId:row.dispatchId,
        commentId:row.commentId,commentRef:row.commentRef,
        commentBodySha256:matches[0].commentBodySha256,
        transportActor:matches[0].transportActor,
        responsibilityStatus:handoff?.status==='VERIFIED'?'VERIFIED':'NOT_VERIFIED',
        activeOperator:handoff?.status==='VERIFIED'?handoff.activeOperator:null,
        handoffRef:handoff?.handoffRef??null,receiptRef:handoff?.receiptRef??null,
        authorized:false,admitted:false,taskPass:false,releasePass:false};
    }catch{return notVerified(a,'CLAIM_READBACK_FAILED');}
  }
  return Object.freeze({readAcceptedClaim});
}


function validTaskPack(pack,repository,versionId,versionSha,blobSha){
  if(!plain(pack)||pack.schema!=='tally/version-task-pack/v1'||
     pack.status!=='CANDIDATE_NOT_FROZEN'||pack.repository!==repository||
     pack.versionId!==versionId||pack.versionBaseSha!==versionSha||
     !SHA.test(blobSha??'')||!Array.isArray(pack.taskIssues)||
     !pack.taskIssues.length||pack.taskIssues.length>200||
     !plain(pack.source)||pack.source.kind!=='GITHUB_NATIVE_ISSUES'||
     pack.source.query!=='state=all&labels=type:task'||
     pack.source.taskIssueCount!==pack.taskIssues.length||
     !Number.isFinite(Date.parse(pack.observedAt??''))||
     !Array.isArray(pack.edges)||pack.edges.length>2000||
     !Array.isArray(pack.claimSubjects)||pack.claimSubjects.length!==pack.taskIssues.length)
    return false;
  const ids=new Set();
  for(const t of pack.taskIssues){
    if(!plain(t)||!Number.isSafeInteger(t.issueNumber)||t.issueNumber<1||
       ids.has(t.issueNumber)||typeof t.title!=='string'||
       !['open','closed'].includes(t.issueState)||
       !Number.isFinite(Date.parse(t.updatedAt??''))||!Array.isArray(t.labels)||
       t.labels.some(x=>typeof x!=='string'))return false;
    ids.add(t.issueNumber);
  }
  const subjects=new Set();
  for(const x of pack.claimSubjects){
    if(!plain(x)||!ids.has(x.issueNumber)||subjects.has(x.issueNumber)||
       !SHA.test(x.sourceSha??'')||!ROLES.has(x.actorRole))return false;
    if(x.candidateStatus==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
       (!Number.isSafeInteger(x.commentId)||x.commentId<1||
        !ID.test(x.dispatchId??'')||!ID.test(x.operatorKind??'')||
        !ID.test(x.operatorId??'')||x.generation!==1||
        x.commentRef!==commentUrl(repository,x.issueNumber,x.commentId)||
        x.protectedClaimKey!==repository+'#'+x.issueNumber+':'+x.actorRole+':__default__'))
      return false;
    if(x.candidateStatus==='NO_CLAIM_EVENT_EXPECTED'&&
       (x.commentId!==null||x.commentRef!==null||x.dispatchId!==null||
        x.protectedClaimKey!==repository+'#'+x.issueNumber+':'+x.actorRole+':__default__'))return false;
    if(!['OBSERVED_COMMENT_SELF_REPORTED_CLAIM','NO_CLAIM_EVENT_EXPECTED'].includes(x.candidateStatus))
      return false;
    subjects.add(x.issueNumber);
  }
  if(subjects.size!==ids.size)return false;
  const seen=new Set(),adj=new Map([...ids].map(n=>[n,[]]));
  for(const e of pack.edges){
    if(!plain(e)||!ids.has(e.from)||!ids.has(e.to)||e.from===e.to||
       !['NATIVE','PLANNED'].includes(e.kind)||
       e.evidenceRef!==issueUrl(repository,e.to)||
       e.dependencyRef!==issueUrl(repository,e.from))
      return false;
    const key=e.from+'>'+e.to;
    if(seen.has(key))return false;
    seen.add(key);adj.get(e.from).push(e.to);
  }
  const done=new Set(),visiting=new Set();
  const walk=n=>{if(visiting.has(n))return false;if(done.has(n))return true;
    visiting.add(n);for(const next of adj.get(n)||[])if(!walk(next))return false;
    visiting.delete(n);done.add(n);return true;};
  return [...ids].every(walk);
}

function nativeRosterMatchesPack(native,pack,repository,versionId,versionSha){
  const source=native?.source,dagSource=native?.dag?.source;
  if(native?.connected!==true||native.versionId!==versionId||
     source?.repository!==repository||source?.sourceSha!==versionSha||
     source?.currentness!=='CURRENT'||source?.readback!=='VERIFIED'||
     dagSource?.repository!==repository||dagSource?.sourceSha!==versionSha||
     dagSource?.currentness!=='CURRENT'||dagSource?.readback!=='VERIFIED'||
     native.dag?.connected!==true||native.dag.versionId!==versionId||
     native.dag.asOf?.sourceSha!==versionSha||
     !Array.isArray(native.tasks)||native.tasks.length!==pack.taskIssues.length||
     !Array.isArray(native.dag.nodes)||native.dag.nodes.length!==pack.taskIssues.length||
     !Array.isArray(native.dag.edges))return false;
  const expected=new Map(pack.taskIssues.map(t=>[t.issueNumber,t]));
  const observed=new Set();
  for(const task of native.tasks){
    const n=Number(task?.id),candidate=expected.get(n);
    if(!Number.isSafeInteger(n)||!candidate||observed.has(n)||
       task.versionId!==versionId||task.title!==candidate.title||
       task.issueState!==candidate.issueState||task.updatedAt!==candidate.updatedAt)
      return false;
    observed.add(n);
  }
  if(observed.size!==expected.size)return false;
  const nodes=new Set();
  for(const node of native.dag.nodes){
    const n=Number(node?.taskId);
    if(!Number.isSafeInteger(n)||!expected.has(n)||nodes.has(n)||
       node.versionId!==versionId)return false;
    nodes.add(n);
  }
  if(nodes.size!==expected.size)return false;
  const normalize=(edges,fromNative)=>edges.map(e=>{
    const from=Number(e?.from),to=Number(e?.to);
    if(!Number.isSafeInteger(from)||!Number.isSafeInteger(to)||
       !expected.has(from)||!expected.has(to)||e.kind!=='NATIVE'||
       e.evidenceRef!==issueUrl(repository,to)||
       (fromNative&&e.dependencyRef!==issueUrl(repository,from)))return null;
    return {from,to,evidenceRef:e.evidenceRef};
  });
  const expectedEdges=normalize(pack.edges.filter(e=>e.kind==='NATIVE'),false);
  const observedEdges=normalize(native.dag.edges,true);
  if(expectedEdges.some(e=>!e)||observedEdges.some(e=>!e)||
     expectedEdges.length!==observedEdges.length)return false;
  const key=e=>e.from+'>'+e.to;
  expectedEdges.sort((a,b)=>key(a).localeCompare(key(b)));
  observedEdges.sort((a,b)=>key(a).localeCompare(key(b)));
  return same(expectedEdges,observedEdges);
}

/** Create-only version DAG freeze writer. It refuses absent independent review or Owner approval. */
export function createHostVersionTaskDagFreezeWriter({
  repository,versionId,expectedVersionSha,store,taskPackReader,nativeRosterReader,
  reviewWitnessReader,ownerApprovalReader,now
}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedVersionSha??'')||typeof store?.get!=='function'||
     typeof store?.put!=='function'||typeof taskPackReader!=='function'||
     typeof nativeRosterReader!=='function'||
     typeof reviewWitnessReader!=='function'||typeof ownerApprovalReader!=='function')
    throw new TypeError('TRUSTED_DAG_FREEZE_WRITER_REQUIRED');
  const clock=now??(()=>new Date().toISOString());
  async function freezeVersionTaskDag({taskPackRef,taskPackBlobSha}={}){
    if(!immutableBlobRef(taskPackRef,repository)||
       !SHA.test(taskPackBlobSha??''))
      return {status:'BLOCKED',reason:'TASK_PACK_IDENTITY_INVALID'};
    try{
      const file=await taskPackReader({taskPackRef,expectedBlobSha:taskPackBlobSha});
      if(file?.status!=='VERIFIED'||file.blobSha!==taskPackBlobSha||
         typeof file.content!=='string')
        return {status:'BLOCKED',reason:'TASK_PACK_READBACK_FAILED'};
      const pack=JSON.parse(file.content);
      if(!validTaskPack(pack,repository,versionId,expectedVersionSha,taskPackBlobSha))
        return {status:'BLOCKED',reason:'TASK_PACK_CONTENT_INVALID'};
      const native=await nativeRosterReader({repository,versionId,
        sourceSha:expectedVersionSha,
        taskIssueNumbers:pack.taskIssues.map(x=>x.issueNumber)});
      if(!nativeRosterMatchesPack(native,pack,repository,versionId,expectedVersionSha))
        return {status:'BLOCKED',reason:'NATIVE_TASK_ROSTER_READBACK_FAILED'};
      const [review,approval]=await Promise.all([
        reviewWitnessReader({taskPackRef,taskPackBlobSha,versionId,sourceSha:expectedVersionSha}),
        ownerApprovalReader({taskPackRef,taskPackBlobSha,versionId,sourceSha:expectedVersionSha})
      ]);
      if(review?.status!=='INDEPENDENTLY_VERIFIED'||
         review.taskPackBlobSha!==taskPackBlobSha||review.sourceSha!==expectedVersionSha||
         !ID.test(review.reviewer??'')||!ID.test(review.author??'')||
         review.reviewer===review.author||review.reviewedHeadSha!==expectedVersionSha||
         typeof review.reviewRef!=='string'||
         !review.reviewRef.startsWith('https://github.com/'+repository+'/'))
        return {status:'BLOCKED',reason:'INDEPENDENT_REVIEW_REQUIRED'};
      if(approval?.status!=='OWNER_APPROVED'||
         approval.taskPackBlobSha!==taskPackBlobSha||
         approval.sourceSha!==expectedVersionSha||
         approval.owner!==repository.split('/')[0]||
         typeof approval.approvalRef!=='string'||
         !approval.approvalRef.startsWith('https://github.com/'+repository+'/'))
        return {status:'BLOCKED',reason:'OWNER_APPROVAL_REQUIRED'};
      const frozenAt=clock();
      if(typeof frozenAt!=='string'||!Number.isFinite(Date.parse(frozenAt)))
        return {status:'BLOCKED',reason:'FREEZE_TIMESTAMP_INVALID'};
      const value={schema:'tally/version-task-dag/v1',phase:'FROZEN_VERSION_TASK_DAG',
        repository,versionId,sourceSha:expectedVersionSha,taskPackRef,
        blobSha:taskPackBlobSha,frozenAt,
        frozen:{blobSha:taskPackBlobSha,taskPackRef,versionId,
          taskIssues:pack.taskIssues.map(x=>x.issueNumber),edges:pack.edges,
          claimSubjects:pack.claimSubjects},
        witness:{status:'INDEPENDENTLY_VERIFIED',blobSha:taskPackBlobSha,
          sourceSha:expectedVersionSha,reviewedHeadSha:review.reviewedHeadSha??null,
          reviewer:review.reviewer,author:review.author,reviewRef:review.reviewRef,
          verifiedAt:review.verifiedAt??clock()},
        ownerApproval:{status:'OWNER_APPROVED',sourceSha:expectedVersionSha,
          approvalRef:approval.approvalRef,owner:approval.owner??null,
          approvedAt:approval.approvedAt??clock()},
        authorized:false,admitted:false,taskPass:false,releasePass:false};
      const key='tally:v2:version-task-dag:'+hash(repository+'#'+versionId);
      const existing=await store.get(key);
      if(existing){
        const prior=existing.value;
        const replay=prior?.schema==='tally/version-task-dag/v1'&&
          prior.phase==='FROZEN_VERSION_TASK_DAG'&&prior.repository===repository&&
          prior.versionId===versionId&&prior.sourceSha===expectedVersionSha&&
          prior.taskPackRef===taskPackRef&&prior.blobSha===taskPackBlobSha&&
          Number.isFinite(Date.parse(prior.frozenAt??''))&&
          same(prior.frozen,{blobSha:taskPackBlobSha,taskPackRef,versionId,
            taskIssues:pack.taskIssues.map(x=>x.issueNumber),edges:pack.edges,
            claimSubjects:pack.claimSubjects})&&
          prior.witness?.status==='INDEPENDENTLY_VERIFIED'&&
          prior.witness.blobSha===taskPackBlobSha&&
          prior.witness.sourceSha===expectedVersionSha&&
          prior.witness.reviewedHeadSha===expectedVersionSha&&
          prior.witness.reviewer===review.reviewer&&prior.witness.author===review.author&&
          prior.witness.reviewer!==prior.witness.author&&
          prior.witness.reviewRef===review.reviewRef&&
          prior.ownerApproval?.status==='OWNER_APPROVED'&&
          prior.ownerApproval.sourceSha===expectedVersionSha&&
          prior.ownerApproval.owner===approval.owner&&
          prior.ownerApproval.approvalRef===approval.approvalRef;
        if(replay)return {status:'REPLAY',blobSha:taskPackBlobSha,frozenAt:prior.frozenAt};
        return {status:'BLOCKED',reason:'DIVERGENT_FREEZE_EXISTS'};
      }
      await store.put(key,value,{mustBeAbsent:true});
      const readback=await store.get(key);
      if(!readback||!same(readback.value,value))
        return {status:'UNKNOWN',reason:'FREEZE_READBACK_MISMATCH'};
      return {status:'FROZEN',blobSha:taskPackBlobSha,frozenAt:value.frozenAt,
        taskCount:value.frozen.taskIssues.length,edgeCount:value.frozen.edges.length,
        authorized:false,admitted:false,taskPass:false,releasePass:false};
    }catch{return {status:'UNKNOWN',reason:'VERSION_DAG_FREEZE_FAILED'};}
  }
  return Object.freeze({freezeVersionTaskDag});
}

/** Re-read the frozen blob and independent witnesses on every roster request. */
export function createHostVersionTaskDagReader({
  repository,versionId,expectedVersionSha,store,taskPackReader,
  reviewWitnessReader,ownerApprovalReader
}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedVersionSha??'')||typeof store?.get!=='function'||
     typeof taskPackReader!=='function'||typeof reviewWitnessReader!=='function'||
     typeof ownerApprovalReader!=='function')
    throw new TypeError('TRUSTED_DAG_READER_REQUIRED');
  async function readFrozenDag(scope={}){
    if((scope.repository!==undefined&&scope.repository!==repository)||
       (scope.versionId!==undefined&&scope.versionId!==versionId)||
       (scope.sourceSha!==undefined&&scope.sourceSha!==expectedVersionSha))
      return denied('FROZEN_DAG_SCOPE_MISMATCH');
    try{
      const key='tally:v2:version-task-dag:'+hash(repository+'#'+versionId);
      const before=await store.get(key),doc=before?.value;
      if(!doc||doc.schema!=='tally/version-task-dag/v1'||
         doc.phase!=='FROZEN_VERSION_TASK_DAG'||doc.repository!==repository||
         doc.versionId!==versionId||doc.sourceSha!==expectedVersionSha||
         doc.authorized!==false||doc.admitted!==false||
         doc.taskPass!==false||doc.releasePass!==false||
         !SHA.test(doc.blobSha??'')||doc.frozen?.blobSha!==doc.blobSha||
         doc.witness?.status!=='INDEPENDENTLY_VERIFIED'||
         doc.witness?.blobSha!==doc.blobSha||doc.witness?.sourceSha!==expectedVersionSha||
         doc.witness?.reviewedHeadSha!==expectedVersionSha||
         !ID.test(doc.witness?.reviewer??'')||!ID.test(doc.witness?.author??'')||
         doc.witness.reviewer===doc.witness.author||
         !Number.isFinite(Date.parse(doc.witness.verifiedAt??''))||
         typeof doc.witness.reviewRef!=='string'||
         !doc.witness.reviewRef.startsWith('https://github.com/'+repository+'/')||
         doc.ownerApproval?.status!=='OWNER_APPROVED'||
         doc.ownerApproval?.sourceSha!==expectedVersionSha||
         doc.ownerApproval?.owner!==repository.split('/')[0]||
         !Number.isFinite(Date.parse(doc.ownerApproval.approvedAt??''))||
         typeof doc.frozenAt!=='string'||!Number.isFinite(Date.parse(doc.frozenAt))||
         doc.frozen?.versionId!==versionId||doc.frozen?.taskPackRef!==doc.taskPackRef||
         !immutableBlobRef(doc.taskPackRef,repository)||
         typeof doc.ownerApproval.approvalRef!=='string'||
         !doc.ownerApproval.approvalRef.startsWith('https://github.com/'+repository+'/'))
        return denied('FROZEN_DAG_RECORD_UNQUALIFIED');
      const file=await taskPackReader({taskPackRef:doc.taskPackRef,expectedBlobSha:doc.blobSha});
      if(file?.status!=='VERIFIED'||file.blobSha!==doc.blobSha||
         typeof file.content!=='string')return denied('TASK_PACK_READBACK_FAILED');
      const pack=JSON.parse(file.content);
      if(!validTaskPack(pack,repository,versionId,expectedVersionSha,doc.blobSha)||
         !same(pack.taskIssues.map(x=>x.issueNumber),doc.frozen.taskIssues)||
         !same(pack.edges,doc.frozen.edges)||!same(pack.claimSubjects,doc.frozen.claimSubjects))
        return denied('FROZEN_DAG_CONTENT_MISMATCH');
      const [review,approval]=await Promise.all([
        reviewWitnessReader({taskPackRef:doc.taskPackRef,taskPackBlobSha:doc.blobSha,
          versionId,sourceSha:expectedVersionSha}),
        ownerApprovalReader({taskPackRef:doc.taskPackRef,taskPackBlobSha:doc.blobSha,
          versionId,sourceSha:expectedVersionSha})
      ]);
      if(review?.status!=='INDEPENDENTLY_VERIFIED'||
         review.taskPackBlobSha!==doc.blobSha||review.sourceSha!==expectedVersionSha||
         review.reviewedHeadSha!==expectedVersionSha||
         review.reviewedHeadSha!==doc.witness.reviewedHeadSha||
         review.reviewer!==doc.witness.reviewer||review.author!==doc.witness.author||
         review.reviewer===review.author||review.reviewRef!==doc.witness.reviewRef||
         approval?.status!=='OWNER_APPROVED'||approval.taskPackBlobSha!==doc.blobSha||
         approval.sourceSha!==expectedVersionSha||approval.owner!==repository.split('/')[0]||
         approval.owner!==doc.ownerApproval?.owner||
         approval.approvalRef!==doc.ownerApproval?.approvalRef)
        return denied('FROZEN_DAG_WITNESS_STALE');
      const after=await store.get(key);
      if(!after||after.revision!==before.revision||!same(after.value,before.value))
        return denied('FROZEN_DAG_CHANGED_DURING_READ');
      return {connected:true,source:{repository,sourceSha:expectedVersionSha,
          currentness:'CURRENT',readback:'VERIFIED'},
        versionId,taskPackRef:doc.taskPackRef,
        witness:doc.witness,ownerApproval:doc.ownerApproval,
        frozenAt:doc.frozenAt,
        frozen:{blobSha:doc.blobSha,taskPackRef:doc.taskPackRef,versionId,
          taskIssues:doc.frozen.taskIssues,edges:doc.frozen.edges,
          claimSubjects:doc.frozen.claimSubjects}};
    }catch{return denied('FROZEN_DAG_READ_FAILED');}
  }
  return Object.freeze({readFrozenDag});
}


export function createHostCloseoutRosterPorts({
  repository,versionId,expectedVersionSha,store,octokit,owner,repo,
  writerLogin,writerId,taskPackReader,reviewWitnessReader,ownerApprovalReader,
  handoffReader
}={}){
  return Object.freeze({
    frozenHost:createHostVersionTaskDagReader({repository,versionId,expectedVersionSha,
      store,taskPackReader,reviewWitnessReader,ownerApprovalReader}),
    freezeWriter:createHostVersionTaskDagFreezeWriter({repository,versionId,expectedVersionSha,
      store,taskPackReader,
      nativeRosterReader:({taskIssueNumbers})=>readUiNativeVersionFromClient({
        octokit,owner,repo,versionId,expectedSha:expectedVersionSha,taskIssueNumbers}),
      reviewWitnessReader,ownerApprovalReader}),
    claimHost:createHostCloseoutClaimReadbackPortFromClient({octokit,owner,repo,
      writerLogin,writerId,store,handoffReader,repository,versionId,expectedVersionSha})
  });
}
