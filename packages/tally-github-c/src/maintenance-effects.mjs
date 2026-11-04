/**
 * J08b — exact authorized native Issue publication transport (#219).
 *
 * The Host owns every authority check (accepted Claim, frozen write set,
 * frozen independent candidate review); this module is the narrow transport:
 * one native Issue create with canonical readback, one bounded reconciliation
 * search and one single-Issue readback. No historical release, Task, Project
 * or task-meta is ever written. A lost acknowledgment stays explicitly
 * PUBLICATION_UNKNOWN — it is never reported as published and never retried
 * blindly, so a duplicate Issue can never be created by this transport.
 */
const OWNER='kaicreator-mm',REPO='tally',REPOSITORY=`${OWNER}/${REPO}`;
const SHA=/^[0-9a-f]{40}$/,DIGEST=/^[0-9a-f]{64}$/;
const TITLE=/^\S(?:.{0,198}\S)?$/s;
const BODY=/^\S[\s\S]{0,9999}\S$/;
const MARKER=/^tally:maintenance:219:[0-9a-f]{64}$/;
const ISSUE_URL=/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/[1-9][0-9]*$/;
const freeze=value=>Object.freeze(value);
const flags=Object.freeze({authorized:false,admitted:false,claimAccepted:false,
  agentLaunch:false,duplicate:false,historicalReleaseMutation:false,
  taskPass:false,releasePass:false});
const notPublished=reason=>freeze({...flags,state:'NOT_PUBLISHED',reason,
  repository:REPOSITORY,issue:null,externalRef:null});
const unknown=(reason,extra={})=>freeze({...flags,state:'PUBLICATION_UNKNOWN',
  reason,repository:REPOSITORY,issue:null,externalRef:null,...extra});
const errorReason=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===409||e?.status===422?'GITHUB_PUBLICATION_CONFLICT':
  e?.status===429?'GITHUB_RATE_LIMITED':'GITHUB_PUBLICATION_UNKNOWN';
const validRequest=({title,body,marker})=>TITLE.test(title??'')&&BODY.test(body??'')&&
  MARKER.test(marker??'')&&body.includes(marker);
const readbackIssue=raw=>raw&&typeof raw==='object'&&!Array.isArray(raw)&&
  Number.isSafeInteger(raw.number)&&raw.number>0&&raw.state==='open'&&
  TITLE.test(raw.title??'')&&ISSUE_URL.test(raw.html_url??'')?
  freeze({number:raw.number,title:raw.title,url:raw.html_url}):null;

/**
 * Publish the authorized native Issue and prove it by canonical readback.
 * The exact Issue identity (title/body/marker) is Host-derived from the
 * frozen candidate; the caller can only transport it.
 */
export async function publishMaintenanceIssueFromClient({octokit,title,body,marker}={}){
  if(!octokit?.rest?.issues?.create||!octokit?.rest?.issues?.get||
    !validRequest({title,body,marker}))
    return notPublished('MAINTENANCE_ISSUE_INPUT_INVALID');
  try{
    const created=(await octokit.rest.issues.create({owner:OWNER,repo:REPO,
      title,body}))?.data;
    const createdIssue=readbackIssue(created);
    if(!createdIssue||created.title!==title)
      return unknown('MAINTENANCE_ISSUE_CREATE_READBACK_MISSING');
    const readback=(await octokit.rest.issues.get({owner:OWNER,repo:REPO,
      issue_number:createdIssue.number}))?.data;
    const issue=readbackIssue(readback);
    if(!issue||issue.number!==createdIssue.number||issue.title!==title||
      !String(readback.body??'').includes(marker))
      return unknown('MAINTENANCE_ISSUE_READBACK_MISMATCH');
    return freeze({...flags,state:'ISSUE_PUBLISHED',
      reason:'GITHUB_ISSUE_CREATE_AND_READBACK_VERIFIED',repository:REPOSITORY,
      issue,externalRef:issue.url});
  }catch(e){return unknown(errorReason(e));}
}

/**
 * Lost-ACK reconciliation: bounded search for the unique marker. Exactly one
 * match reconciles without a duplicate; zero matches clears the way for the
 * single authorized publish; more than one match is a hard identity conflict
 * that never heals by creating another Issue.
 */
export async function findMaintenanceIssueFromClient({octokit,marker,title,pages=3}={}){
  if(!octokit?.rest?.issues?.listForRepo||!MARKER.test(marker??'')||
    !TITLE.test(title??'')||!Number.isSafeInteger(pages)||pages<1||pages>10)
    return freeze({...flags,state:'RECONCILE_UNKNOWN',
      reason:'MAINTENANCE_RECONCILE_INPUT_INVALID',matches:[]});
  try{
    const matches=[];
    for(let page=1;page<=pages;page++){
      const data=(await octokit.rest.issues.listForRepo({owner:OWNER,repo:REPO,
        state:'all',per_page:100,page}))?.data;
      if(!Array.isArray(data))
        return freeze({...flags,state:'RECONCILE_UNKNOWN',
          reason:'MAINTENANCE_RECONCILE_READ_FAILED',matches:[]});
      for(const raw of data){
        if(typeof raw?.body==='string'&&raw.body.includes(marker)&&raw.title===title){
          const issue=readbackIssue(raw);
          if(issue)matches.push(issue);
        }
      }
      if(data.length<100)break;
    }
    if(matches.length>1)
      return freeze({...flags,state:'RECONCILE_CONFLICT',
        reason:'MAINTENANCE_ISSUE_IDENTITY_CONFLICT',matches:freeze(matches)});
    return freeze({...flags,state:'RECONCILE_COMPLETE',
      reason:matches.length===1?'MAINTENANCE_ISSUE_FOUND':'MAINTENANCE_ISSUE_ABSENT',
      matches:freeze(matches)});
  }catch(e){return freeze({...flags,state:'RECONCILE_UNKNOWN',
    reason:errorReason(e),matches:[]});}
}

/** Canonical single-Issue readback used after publish and reconcile. */
export async function readMaintenanceIssueFromClient({octokit,issueNumber,title,marker}={}){
  if(!octokit?.rest?.issues?.get||!Number.isSafeInteger(issueNumber)||
    issueNumber<1||!TITLE.test(title??'')||!MARKER.test(marker??''))
    return freeze({...flags,state:'READBACK_UNKNOWN',
      reason:'MAINTENANCE_READBACK_INPUT_INVALID',issue:null});
  try{
    const data=(await octokit.rest.issues.get({owner:OWNER,repo:REPO,
      issue_number:issueNumber}))?.data;
    const issue=readbackIssue(data);
    if(!issue||issue.title!==title||!String(data.body??'').includes(marker))
      return freeze({...flags,state:'READBACK_MISMATCH',
        reason:'MAINTENANCE_ISSUE_READBACK_MISMATCH',issue:null});
    return freeze({...flags,state:'READBACK_VERIFIED',
      reason:'GITHUB_ISSUE_READBACK_VERIFIED',issue});
  }catch(e){return freeze({...flags,state:'READBACK_UNKNOWN',
    reason:errorReason(e),issue:null});}
}

const OWNER_APPROVAL_MARKER='OWNER_T219_LEARNING_ADOPT_APPROVAL=';
const OWNER_APPROVAL_MAX_PAGES=10;
const OWNER_APPROVAL_ISSUE_API='https://api.github.com/repos/'+OWNER+'/'+REPO+'/issues/219';
const ownerApprovalDenied=(sourceSha,learningId,reason,extra={})=>freeze({
  state:'OWNER_APPROVAL_NOT_VERIFIED',reason,repository:REPOSITORY,
  issueNumber:219,sourceSha,learningId,decision:'ADOPTED',...extra});
const parseOwnerApprovalBlock=block=>{
  const lines=String(block).trim().split(/\r?\n/);
  if(lines.length!==4)return null;
  const approval=/^OWNER_T219_LEARNING_ADOPT_APPROVAL=([A-Z_]+)$/.exec(lines[0]);
  const source=/^sourceSha=([0-9a-f]{40})$/.exec(lines[1]);
  const learning=/^learningId=([0-9a-f]{64})$/.exec(lines[2]);
  const decision=/^decision=([A-Z_]+)$/.exec(lines[3]);
  return approval&&source&&learning&&decision?{
    approval:approval[1],sourceSha:source[1],learningId:learning[1],
    decision:decision[1]}:null;
};

/**
 * Verify one exact #219 ADOPT approval from GitHub itself. The approval is
 * usable only when exactly one structured comment binds this source SHA and
 * learning ID, its actual author is a current repository admin, and every
 * identity field is read back from the canonical GitHub API response.
 */
export async function readTaskLearningOwnerApprovalFromClient({
  octokit,sourceSha,learningId,decision,perPage=100,maxPages=OWNER_APPROVAL_MAX_PAGES
}={}){
  if(!octokit?.rest?.issues?.listComments||
    !octokit?.rest?.repos?.getCollaboratorPermissionLevel||
    !SHA.test(sourceSha??'')||!DIGEST.test(learningId??'')||
    decision!=='ADOPTED'||perPage!==100||
    !Number.isSafeInteger(maxPages)||maxPages<1||maxPages>OWNER_APPROVAL_MAX_PAGES)
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_INPUT_INVALID');

  const exact=[],stale=[];
  let malformedRelevant=false,complete=false;
  const fence=/\x60\x60\x60(?:[A-Za-z0-9_-]+)?[ \t]*\r?\n([\s\S]*?)\r?\n\x60\x60\x60/g;
  try{
    for(let page=1;page<=maxPages;page++){
      const rows=(await octokit.rest.issues.listComments({
        owner:OWNER,repo:REPO,issue_number:219,per_page:perPage,page
      }))?.data;
      if(!Array.isArray(rows))
        return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_READ_INVALID');
      for(const comment of rows){
        const body=typeof comment?.body==='string'?comment.body:'';
        if(!body.includes(OWNER_APPROVAL_MARKER))continue;
        const blocks=[...body.matchAll(fence)];
        const outside=body.replace(fence,'');
        if(outside.includes(OWNER_APPROVAL_MARKER)&&
          (outside.includes('learningId='+learningId)||
            outside.includes('sourceSha='+sourceSha)))
          malformedRelevant=true;
        for(const match of blocks){
          const block=match[1];
          if(!block.includes(OWNER_APPROVAL_MARKER))continue;
          const parsed=parseOwnerApprovalBlock(block);
          if(parsed?.learningId===learningId){
            if(parsed.sourceSha===sourceSha)exact.push({comment,parsed});
            else stale.push({comment,parsed});
          }else if(block.includes('learningId='+learningId)){
            malformedRelevant=true;
          }
        }
      }
      if(rows.length<perPage){complete=true;break;}
    }
  }catch(error){
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_READ_FAILED',
      {transportReason:errorReason(error)});
  }
  if(!complete)
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_READ_INCOMPLETE');
  if(malformedRelevant||exact.length>1)
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_CONFLICT');
  if(exact.length===0)
    return ownerApprovalDenied(sourceSha,learningId,
      stale.length?'OWNER_APPROVAL_STALE':'OWNER_APPROVAL_MISSING');

  const {comment,parsed}=exact[0];
  if(parsed.approval!=='APPROVE'||parsed.decision!=='ADOPTED')
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_CONFLICT');
  const commentId=comment?.id,author=comment?.user?.login;
  const commentUrl='https://github.com/'+REPOSITORY+'/issues/219#issuecomment-'+commentId;
  if(!Number.isSafeInteger(commentId)||commentId<1||
    typeof author!=='string'||!/^[A-Za-z0-9-]{1,39}$/.test(author)||
    comment.issue_url!==OWNER_APPROVAL_ISSUE_API||
    comment.html_url!==commentUrl||
    typeof comment.created_at!=='string'||!Number.isFinite(Date.parse(comment.created_at)))
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_COMMENT_IDENTITY_INVALID');

  let permission;
  try{
    const response=await octokit.rest.repos.getCollaboratorPermissionLevel({
      owner:OWNER,repo:REPO,username:author
    });
    permission=response?.data?.permission??response?.data?.role_name;
  }catch(error){
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_PERMISSION_READ_FAILED',
      {transportReason:errorReason(error)});
  }
  if(permission!=='admin')
    return ownerApprovalDenied(sourceSha,learningId,'OWNER_APPROVAL_AUTHOR_NOT_ADMIN');

  return freeze({state:'OWNER_APPROVAL_VERIFIED',
    reason:'GITHUB_OWNER_APPROVAL_VERIFIED',repository:REPOSITORY,
    issueNumber:219,sourceSha,learningId,decision:'ADOPTED',
    commentId,commentUrl,author,permission:'admin'});
}

/** Maintenance marker is derived only from the frozen candidate digest. */
export const maintenanceMarkerFor=candidateDigest=>
  DIGEST.test(candidateDigest??'')?`tally:maintenance:219:${candidateDigest}`:null;
export const __test={OWNER,REPO,REPOSITORY,validRequest,readbackIssue,errorReason};
