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


/** #219 Core MVP — read-only, fail-closed independent candidate-review
 * witness. This transport is NOT a Host CAS writer or an authority mint.
 * A selected Host must still bind the accepted role Claims/FSW and review
 * against an immutable persisted candidate under its version fence.
 */
export async function readMaintenanceReviewWitnessFromClient({
  octokit,commentId,expected,builderClaim,reviewerClaim
}={}){
  const denied=reason=>freeze({...flags,state:'REVIEW_UNKNOWN',reason,
    repository:REPOSITORY,reviewRef:null,reviewed:false});
  const claim=(c,role)=>c?.accepted===true&&c.role===role&&
    c.issueNumber===219&&c.sourceSha===expected?.sourceSha&&
    c.candidateDigest===expected?.candidateDigest&&
    typeof c.claimRef==='string'&&c.claimRef.length>0&&
    typeof c.operatorId==='string'&&c.operatorId.length>0&&
    typeof c.writerId==='string'&&c.writerId.length>0&&
    typeof c.transportActor==='string'&&c.transportActor.length>0;
  if(!Number.isSafeInteger(commentId)||commentId<1||
     expected?.issueNumber!==219||!SHA.test(expected.sourceSha??'')||
     !DIGEST.test(expected.candidateDigest??'')||
     !Number.isSafeInteger(expected.reviewedPullNumber)||
     expected.reviewedPullNumber<1||
     !SHA.test(expected.reviewedHead??'')||
     !SHA.test(expected.reviewedTree??'')||
     !SHA.test(expected.reviewedBase??'')||
     typeof octokit?.rest?.issues?.getComment!=='function'||
     typeof octokit?.rest?.pulls?.get!=='function'||
     typeof octokit?.rest?.repos?.getCommit!=='function'||
     typeof octokit?.rest?.repos?.getBranch!=='function'||
     typeof octokit?.graphql!=='function')
    return denied('MAINTENANCE_REVIEW_INPUT_INVALID');
  if(!claim(builderClaim,'builder')||!claim(reviewerClaim,'reviewer'))
    return denied('MAINTENANCE_REVIEW_ROLE_CLAIM_UNVERIFIED');
  if(builderClaim.operatorId===reviewerClaim.operatorId||
     builderClaim.writerId===reviewerClaim.writerId)
    return denied('MAINTENANCE_REVIEW_SAME_PROCESS');
  let comment;
  try{comment=(await octokit.rest.issues.getComment({
    owner:OWNER,repo:REPO,comment_id:commentId}))?.data;}
  catch{return denied('MAINTENANCE_REVIEW_GITHUB_READ_UNKNOWN');}
  const ref='https://github.com/'+REPOSITORY+
    '/issues/219#issuecomment-'+commentId;
  if(comment?.id!==commentId||comment.html_url!==ref||
     comment.issue_url!=='https://api.github.com/repos/'+REPOSITORY+'/issues/219'||
     comment.user?.login!==reviewerClaim.transportActor||
     typeof comment.body!=='string'||comment.body.length>8192||
     !/^\d{4}-\d\d-\d\dT/.test(comment.created_at??'')||
     comment.created_at!==comment.updated_at||
     typeof comment.node_id!=='string'||!/^[A-Za-z0-9_-]{8,256}$/.test(comment.node_id))
    return denied('MAINTENANCE_REVIEW_COMMENT_UNVERIFIED');
  let graph;
  try{
    graph=(await octokit.graphql(`query($id: ID!) {
      node(id: $id) {
        __typename
        ... on IssueComment {
          id
          fullDatabaseId
          url
          body
          lastEditedAt
          editor { login }
          issue { number repository { nameWithOwner } }
        }
      }
    }`,{id:comment.node_id}))?.node;
  }catch{return denied('MAINTENANCE_REVIEW_EDIT_PROVENANCE_UNAVAILABLE');}
  const id=graph?.fullDatabaseId;
  const fullId=typeof id==='string'&&/^[1-9][0-9]*$/.test(id)?id:
    Number.isSafeInteger(id)&&id>0?String(id):null;
  if(graph?.__typename!=='IssueComment'||graph.id!==comment.node_id||
     fullId!==String(commentId)||graph.url!==ref||
     graph.body!==comment.body||graph.lastEditedAt!==null||
     graph.editor!==null||graph.issue?.number!==219||
     graph.issue.repository?.nameWithOwner!==REPOSITORY)
    return denied('MAINTENANCE_REVIEW_EDIT_PROVENANCE_UNVERIFIED');
  let body;
  try{
    body=JSON.parse(comment.body);
    if(!body||typeof body!=='object'||Array.isArray(body)||
       JSON.stringify(body)!==comment.body.trim())
      return denied('MAINTENANCE_REVIEW_NONCANONICAL');
  }catch{return denied('MAINTENANCE_REVIEW_NONCANONICAL');}
  const names=['schema','status','repository','issueNumber','sourceSha',
    'candidateDigest','reviewedPullNumber','reviewedHead','reviewedTree',
    'reviewedBase','builderClaimRef','reviewerClaimRef','reviewerOperatorId',
    'reviewerWriterId','p0','p1','p2'];
  if(Object.keys(body).length!==names.length||
     names.some(k=>!Object.hasOwn(body,k))||
     body.schema!=='tally.v2.maintenance-review-witness/1'||
     body.status!=='INDEPENDENTLY_VERIFIED'||
     body.repository!==REPOSITORY||body.issueNumber!==219||
     body.builderClaimRef!==builderClaim.claimRef||
     body.reviewerClaimRef!==reviewerClaim.claimRef||
     body.reviewerOperatorId!==reviewerClaim.operatorId||
     body.reviewerWriterId!==reviewerClaim.writerId||
     ['sourceSha','candidateDigest','reviewedPullNumber','reviewedHead',
       'reviewedTree','reviewedBase'].some(k=>body[k]!==expected[k])||
     body.p0!==0||body.p1!==0||
     !Number.isSafeInteger(body.p2)||body.p2<0||body.p2>32)
    return denied('MAINTENANCE_REVIEW_WITNESS_MISMATCH');
  // At this C-stage, refuse merged/stale PRs until the later selected Host
  // can prove an exact two-parent merge, immutable source ancestry and CAS.
  // Never infer currentness or accepted Task Outcome from body text alone.
  try{
    const pr=(await octokit.rest.pulls.get({owner:OWNER,repo:REPO,
      pull_number:body.reviewedPullNumber}))?.data;
    const head=(await octokit.rest.repos.getCommit({owner:OWNER,repo:REPO,
      ref:expected.reviewedHead}))?.data;
    const base=(await octokit.rest.repos.getCommit({owner:OWNER,repo:REPO,
      ref:expected.reviewedBase}))?.data;
    const live=(await octokit.rest.repos.getBranch({owner:OWNER,repo:REPO,
      branch:'version/v2'}))?.data;
    if(pr?.number!==body.reviewedPullNumber||pr.state!=='open'||pr.merged===true||
       pr.head?.sha!==body.reviewedHead||
       pr.head.repo?.full_name!==REPOSITORY||
       pr.base?.sha!==body.reviewedBase||
       pr.base?.ref!=='version/v2'||
       head?.sha!==body.reviewedHead||
       head.commit?.tree?.sha!==body.reviewedTree||
       base?.sha!==body.reviewedBase||
       live?.commit?.sha!==body.sourceSha||
       body.reviewedBase!==body.sourceSha)
      return denied('MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT');
  }catch{return denied('MAINTENANCE_REVIEW_GITHUB_READ_UNKNOWN');}
  return freeze({...flags,state:'REVIEW_VERIFIED',
    reason:'MAINTENANCE_SOURCE_REVIEW_WITNESS_VERIFIED',
    repository:REPOSITORY,reviewRef:ref,reviewed:true,
    sourceSha:body.sourceSha,candidateDigest:body.candidateDigest,
    reviewedHead:body.reviewedHead,reviewedTree:body.reviewedTree,
    p0:0,p1:0,p2:body.p2});
}
