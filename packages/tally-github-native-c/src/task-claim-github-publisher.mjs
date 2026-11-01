/**
 * T06i - selected Host-bound native GitHub transport for a serialized claim.
 *
 * This is a *transport implementation*, not a CAS writer or K permit.
 * Host must supply sealed Octokit credentials, a genuinely exclusive writer
 * and a previously accepted physical CAS reservation. Never expose the
 * factory to an Agent, model output, UI endpoint or untrusted B port.
 * POST is non-idempotent: on lost ACK the Host reservation remains PENDING,
 * and no caller is authorized to retry blindly.
 */
import {stringify} from 'yaml';
import {createHash} from 'node:crypto';

const NAME=/^[A-Za-z0-9_.-]+$/;
const GH_LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions',
  'woodpecker','other']);
const marker='<!-- ads:github -->';
const hash=x=>createHash('sha256').update(x,'utf8').digest('hex');
const forbidden=()=>new Error('HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED');
const ambiguous=()=>new Error('NATIVE_GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');

function requireReservation(i,owner,repo,writerId){
  if(!i||typeof i!=='object'||Array.isArray(i)||
     i.repository!==owner+'/'+repo||!Number.isSafeInteger(i.issueNumber)||
     i.issueNumber<1||!SHA.test(i.sourceSha??'')||
     i.issueRef!=='https://github.com/'+owner+'/'+repo+'/issues/'+i.issueNumber||
     !ROLES.has(i.actorRole)||!KINDS.has(i.operatorKind)||
     !ID.test(i.operatorId??'')||!ID.test(i.dispatchId??'')||
     i.writerId!==writerId||i.generation!==1||
     i.protectedClaimKey!==owner+'/'+repo+'#'+i.issueNumber+
       ':'+i.actorRole+':__default__')
    throw new Error('HOST_CLAIM_RESERVATION_NOT_BOUND');
  return i;
}
function renderEvent(i){
  // Only schema-allowed, host-reservation-sourced fields. No extra free-form
  // user text, token, Agent prompt, private transcript or secret in the post.
  const data={
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',
    actor_role:i.actorRole,operator_kind:i.operatorKind,operator_id:i.operatorId,
    task:'#'+i.issueNumber,issue:'#'+i.issueNumber,sha:i.sourceSha,
    dispatch_id:i.dispatchId,dispatch_state:'CLAIMED',
    protected_claim_key:i.protectedClaimKey,admission_generation:1
  };
  return marker+'\n'+String.fromCharCode(96).repeat(3)+'yaml\n'+
    stringify({schema:'ads/github/v3',kind:'event',data}).trimEnd()+
    '\n'+String.fromCharCode(96).repeat(3)+'\n';
}
function readbackValid(c,id,issueUrl,apiIssueUrl,login,body){
  return c?.id===id&&c?.html_url===issueUrl+'#issuecomment-'+id&&
    c?.issue_url===apiIssueUrl&&c?.user?.login===login&&
    typeof c.body==='string'&&hash(c.body)===hash(body)&&
    typeof c.created_at==='string'&&c.created_at===c.updated_at;
}

/** The writerIdentity object is trusted only as Host-controlled configuration;
 * data from an API request cannot constitute this authority.
 */
export function createHostBoundNativeGitHubClaimPublisher({
  octokit,owner,repo,writerAuthority
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
     writerAuthority?.exclusiveHostControl!==true||
     writerAuthority?.repository!==owner+'/'+repo||
     !ID.test(writerAuthority?.writerId??'')||
     !GH_LOGIN.test(writerAuthority?.githubWriterLogin??'')||
     typeof octokit?.rest?.issues?.createComment!=='function'||
     typeof octokit?.rest?.issues?.getComment!=='function')
    throw forbidden();
  const writerId=writerAuthority.writerId;
  const login=writerAuthority.githubWriterLogin;
  async function publishClaim(request){
    const i=requireReservation(request,owner,repo,writerId);
    const issueUrl=i.issueRef;
    const apiIssueUrl='https://api.github.com/repos/'+owner+'/'+repo+
      '/issues/'+i.issueNumber;
    const body=renderEvent(i);
    // Calling this method is permitted ONLY after a real selected Host CAS
    // reservation. The method has no retry and no unprivileged auth fallback.
    let response;
    try{
      response=await octokit.rest.issues.createComment({
        owner,repo,issue_number:i.issueNumber,body
      });
    }catch{throw ambiguous();}
    const id=response?.data?.id;
    // If POST may have succeeded but response was lost/invalid, preserve
    // Host PENDING reservation and force a separate native reconciliation.
    if(!Number.isSafeInteger(id)||id<1||
       !readbackValid(response.data,id,issueUrl,apiIssueUrl,login,body))
      throw ambiguous();
    let reread;
    try{reread=(await octokit.rest.issues.getComment({
      owner,repo,comment_id:id
    }))?.data;}catch{throw ambiguous();}
    if(!readbackValid(reread,id,issueUrl,apiIssueUrl,login,body))
      throw ambiguous();
    // This is solely a transport ACK. Host CAS persistence and Kernel Task
    // effect admission are independently necessary and NOT indicated here.
    return Object.freeze({
      commentId:id,commentRef:issueUrl+'#issuecomment-'+id,
      commentBodySha256:hash(body),claimPublication:'GITHUB_COMMENT_POST_READBACK',
      claimAccepted:false,authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
  }
  return Object.freeze({publishClaim});
}
