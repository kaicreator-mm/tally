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

/** Maintenance marker is derived only from the frozen candidate digest. */
export const maintenanceMarkerFor=candidateDigest=>
  DIGEST.test(candidateDigest??'')?`tally:maintenance:219:${candidateDigest}`:null;
export const __test={OWNER,REPO,REPOSITORY,validRequest,readbackIssue,errorReason};
