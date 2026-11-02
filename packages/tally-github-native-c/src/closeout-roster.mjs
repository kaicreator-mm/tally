import {createHash} from 'node:crypto';
import {parseDocument,visit,isAlias} from 'yaml';
import {readClaimEventCandidatesFromClient} from './task-claim-events.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from './task-claim-host-readback-port.mjs';

const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
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
    const c=(await octokit.rest.issues.getComment({owner,repo,comment_id:commentId}))?.data;
    if(c?.id!==commentId||c.html_url!==commentUrl(owner+'/'+repo,issueNumber,commentId)||
       c.issue_url!==apiIssueUrl(owner,repo,issueNumber)||c.user?.login!==login||
       typeof c.body!=='string'||c.created_at!==c.updated_at)return null;
    return c;
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
         h.protected_claim_key!==claimSubject.protectedClaimKey||
         h.admission_generation!==claimSubject.generation||
         h.authority_ref!=='#259@'+authorityCommentId)
        return {status:'NOT_VERIFIED',reason:'OWNER_HANDOFF_MISMATCH'};
      const expectedScope='#'+issueNumber+'@'+scopeCommentId;
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
    claimAccepted:false,claimPublication:'CANDIDATE_ONLY',reason,
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
       !/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/blob\/[a-f0-9]{40}\//.test(a.taskPackRef??'')||
       !SHA.test(a.taskPackBlobSha??'')||!plain(a.claimSubject)||
       a.claimSubject.issueNumber!==a.issueNumber||
       a.claimSubject.sourceSha!==a.claimSourceSha)
      return denied('CLAIM_READBACK_INPUT_INVALID');
    const s=a.claimSubject,role=s.actorRole;
    if(!ROLES.has(role)||s.protectedClaimKey!==repository+'#'+a.issueNumber+':'+role+':__default__')
      return notVerified(a,'CLAIM_SUBJECT_INVALID');
    const key=claimKey(repository,a.issueNumber,role);
    try{
      const before=await store.get(key);
      const events=await readClaimEventCandidatesFromClient({octokit,owner,repo,
        issueNumber:a.issueNumber,expectedSourceSha:a.claimSourceSha});
      if(!before){
        const after=await store.get(key);
        if(after!==null)return notVerified(a,'CLAIM_CHANGED_DURING_READ');
        if(events?.reason==='NO_DISPATCH_CLAIM_EVENTS'&&events.candidateCount===0)
          return notClaimed(a);
        const c=events?.candidates?.find(x=>x.protectedClaimKey===s.protectedClaimKey);
        return notVerified(a,'NATIVE_EVENT_WITHOUT_CAS_ROW',c?{
          commentId:c.commentId,commentRef:c.commentRef,
          commentBodySha256:c.commentBodySha256,dispatchId:c.dispatchId}: {});
      }
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
         matches[0].operatorId!==row.operatorId||matches[0].transportActor!==writerLogin||
         matches[0].sourceSha!==row.sourceSha)
        return notVerified(a,'NATIVE_EVENT_DOES_NOT_MATCH_CAS_ROW');
      const reservation={repository,issueNumber:a.issueNumber,issueRef:row.issueRef,
        sourceSha:row.sourceSha,actorRole:row.actorRole,operatorKind:row.operatorKind,
        operatorId:row.operatorId,dispatchId:row.dispatchId,generation:row.generation,
        protectedClaimKey:row.protectedClaimKey};
      const witness=await native.readPublishedClaim(reservation,row.commentId);
      if(witness?.sourceKind!=='NATIVE_GITHUB_EXACT_READBACK'||
         witness.commentId!==row.commentId||witness.commentRef!==row.commentRef||
         witness.sourceSha!==row.sourceSha||witness.transportActor!==writerLogin)
        return notVerified(a,'NATIVE_COMMENT_READBACK_FAILED');
      const eventsAfter=await readClaimEventCandidatesFromClient({octokit,owner,repo,
        issueNumber:a.issueNumber,expectedSourceSha:a.claimSourceSha});
      const after=await store.get(key);
      const last=eventsAfter?.candidates?.filter(x=>x.protectedClaimKey===row.protectedClaimKey);
      if(!after||after.revision!==before.revision||!same(after.value,before.value)||
         eventsAfter?.reason!=='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'||
         !Array.isArray(last)||last.length!==1||
         last[0].commentBodySha256!==matches[0].commentBodySha256)
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
        responsibilityStatus:handoff?.status==='VERIFIED'?'VERIFIED':'NOT_VERIFIED',
        activeOperator:handoff?.status==='VERIFIED'?handoff.activeOperator:null,
        handoffRef:handoff?.handoffRef??null,receiptRef:handoff?.receiptRef??null,
        authorized:false,admitted:false,taskPass:false,releasePass:false};
    }catch{return notVerified(a,'CLAIM_READBACK_FAILED');}
  }
  return Object.freeze({readAcceptedClaim});
}
