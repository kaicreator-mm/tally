/** #274 selected Host-bound ADS v5 feedback publisher.
 * No caller authority, no queue, no retry of a non-idempotent GitHub POST.
 * A verified publication is NOT Task/K/Release PASS.
 */
import {createHash} from 'node:crypto';
import canonicalize from 'canonicalize';
import {stringify,parseDocument,visit,isAlias} from 'yaml';
import {validateAdsV5Report} from '../../tally-ads-v5-validator/src/validate.mjs';
const SHA=/^[a-f0-9]{40}$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const GH_LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/;
const marker='<!-- ads:github -->';
const fence=String.fromCharCode(96).repeat(3);
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const noAuthority={authorized:false,admitted:false,taskPass:false,releasePass:false};
const fail=(reason,status='BLOCKED')=>Object.freeze({status,reason,
  publicationRef:null,readback:status==='UNKNOWN'?'UNKNOWN':'NOT_VERIFIED',
  reconciliationRequired:status==='UNKNOWN',...noAuthority});
const hash=v=>createHash('sha256').update(canonicalize(v)).digest('hex');
const issueUrl=(repository,n)=>'https://github.com/'+repository+'/issues/'+n;
function render(envelope) {
  return marker+'\n'+fence+'yaml\n'+stringify(envelope).trimEnd()+'\n'+fence+'\n';
}
function parse(text) {
  if(typeof text!=='string'||text.length>32768||!text.startsWith(marker+'\n'))
    return null;
  const lines=text.split(/\r\n|\n|\r/);
  if(lines[0]!==marker||lines[1]!==fence+'yaml'||lines.filter(x=>x===marker).length!==1)
    return null;
  const end=lines.indexOf(fence,2);
  if(end<3)return null;
  try {
    const d=parseDocument(lines.slice(2,end).join('\n'),
      {uniqueKeys:true,merge:false,maxAliasCount:0,strict:true});
    if(d.errors.length||d.warnings.length)return null;
    visit(d,{Node(_key,node){
      if(isAlias(node)||node.anchor!==undefined||node.tag!==undefined)
        throw new Error('NATIVE_YAML_ALIAS_ANCHOR_TAG_DENIED');
    }});
    return d.toJS({maxAliasCount:0});
  }catch{return null;}
}
function validNative(c,repository,issue,login,body){
  return Number.isSafeInteger(c?.id)&&c.id>0&&
    c.html_url===issueUrl(repository,issue)+'#issuecomment-'+c.id&&
    c.issue_url==='https://api.github.com/repos/'+repository+'/issues/'+issue&&
    c.user?.login===login&&c.body===body&&
    typeof c.created_at==='string'&&c.created_at===c.updated_at;
}
function validCandidate(candidate,repository,issueNumber,sourceSha,namespace,publisherSha) {
  const d=candidate?.envelope?.data,subject=d?.subject,p=d?.publication;
  if(candidate?.status!=='CANDIDATE_ONLY'||candidate?.envelope?.schema!=='ads/github/v5'||
     candidate.envelope.kind!=='report'||!plain(d)||d.report_kind!=='EXPERIENCE_USE_FEEDBACK'||
     d.publisher_namespace!==namespace||d.publisher_source_sha!==publisherSha||
     subject?.repository!==repository||subject.issue_ref!==issueUrl(repository,issueNumber)||
     subject.source_sha!==sourceSha||subject.source_status!=='PINNED'||
     subject.source_currentness!=='CURRENT'||subject.ads_profile!=='ads/github/v5:report'||
     p?.classification!=='PUBLISHABLE'||!SHA.test(sourceSha??''))return false;
  const material={...d,publication:{classification:p.classification,
    clearance_ref:p.clearance_ref}};
  return p.projection_digest===hash(material)&&candidate.dataDigest===hash(d)&&
    validateAdsV5Report(candidate.envelope).valid===true;
}
async function scan(octokit,owner,repo,number,login,data){
  const matches=[],prior=[];let scanned=0;
  try{
    for await(const page of octokit.paginate.iterator(octokit.rest.issues.listComments,
      {owner,repo,issue_number:number,per_page:100})){
      if(!Array.isArray(page?.data))return null;
      for(const comment of page.data){
        if(++scanned>5000)return null;
        if(comment?.user?.login!==login)continue;
        const env=parse(comment.body);
        if(env?.schema!=='ads/github/v5'||env?.kind!=='report'||
           env.data?.report_kind!=='EXPERIENCE_USE_FEEDBACK')continue;
        if(!validNative(comment,owner+'/'+repo,number,login,comment.body)||
           !validateAdsV5Report(env).valid)return null;
        const d=env.data;
        if(d.publisher_namespace!==data.publisher_namespace||d.report_id!==data.report_id)
          continue;
        if(d.revision===data.revision)matches.push({comment,envelope:env});
        if(d.revision===data.revision-1)prior.push({comment,envelope:env});
      }
    }
  }catch{return null;}
  return {matches,prior};
}
export function createHostBoundExperienceFeedbackPublisher({
  octokit,owner,repo,writerAuthority
}={}) {
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
     writerAuthority.exclusiveHostControl!==true||
     writerAuthority.repository!==owner+'/'+repo||
     !ID.test(writerAuthority.writerId??'')||
     !GH_LOGIN.test(writerAuthority.githubWriterLogin??'')||
     !NAME.test(writerAuthority.publisherNamespace??'')||
     !SHA.test(writerAuthority.sourceSha??'')||
     !SHA.test(writerAuthority.publisherSourceSha??'')||
     typeof writerAuthority.readHeldFence!=='function'||
     typeof writerAuthority.readKernelPermit!=='function'||
     typeof writerAuthority.readFeedbackEvidence!=='function'||
     typeof octokit?.rest?.issues?.get!=='function'||
     typeof octokit.rest.issues.listComments!=='function'||
     typeof octokit.rest.issues.createComment!=='function'||
     typeof octokit.rest.issues.getComment!=='function'||
     typeof octokit?.paginate?.iterator!=='function')
    throw new TypeError('HOST_BOUND_FEEDBACK_PUBLISHER_REQUIRED');
  const repository=owner+'/'+repo,login=writerAuthority.githubWriterLogin;
  async function qualified(number,data) {
    const [lease,permit,evidence]=await Promise.all([
      writerAuthority.readHeldFence({repository,issueNumber:number,writerId:writerAuthority.writerId}),
      writerAuthority.readKernelPermit({repository,issueNumber:number,sourceSha:writerAuthority.sourceSha}),
      writerAuthority.readFeedbackEvidence({repository,issueNumber:number,
        sourceSha:writerAuthority.sourceSha,reportId:data.report_id,
        revision:data.revision,phase:data.phase,
        experienceRef:data.experience_ref,procedureRevision:data.procedure_revision,
        evidenceRef:data.subject.currentness.source_refs[0],
        expectedDataDigest:hash(data)})]);
    return plain(lease)&&lease.sourceKind==='TRUSTED_HOST_RESOURCE_FENCE_READBACK'&&
      lease.complete===true&&lease.fenceHeld===true&&
      lease.repository===repository&&lease.writerId===writerAuthority.writerId&&
      lease.heldThrough==='CAS_PUBLISH_READBACK'&&
      plain(permit)&&permit.status==='AUTHORIZED'&&permit.repository===repository&&
      permit.issueNumber===number&&permit.sourceSha===writerAuthority.sourceSha&&
      plain(evidence)&&evidence.status==='VERIFIED'&&
      evidence.sourceKind==='HOST_VERIFIED_FEEDBACK_EVIDENCE'&&
      evidence.repository===repository&&evidence.issueNumber===number&&
      evidence.sourceSha===writerAuthority.sourceSha&&
      evidence.reportId===data.report_id&&evidence.revision===data.revision&&
      evidence.phase===data.phase&&evidence.experienceRef===data.experience_ref&&
      evidence.procedureRevision===data.procedure_revision&&
      evidence.evidenceRef===data.subject.currentness.source_refs[0]&&
      evidence.dataDigest===hash(data)&&
      (data.phase!=='ACCEPTED'||evidence.humanDecision==='APPROVE')&&
      (data.phase!=='SKIPPED'||evidence.humanDecision==='REJECT')&&
      (!['OBSERVED_EFFECT','INDEPENDENTLY_CHECKED'].includes(data.phase)||
        evidence.independentVerified===true);
  }
  async function publishFeedback({issueNumber,candidate}={}) {
    if(!Number.isSafeInteger(issueNumber)||issueNumber<1||
       !validCandidate(candidate,repository,issueNumber,writerAuthority.sourceSha,
         writerAuthority.publisherNamespace,writerAuthority.publisherSourceSha))
      return fail('FEEDBACK_CANDIDATE_OR_SCHEMA_INVALID');
    const data=candidate.envelope.data,body=render(candidate.envelope);
    if(data.actor?.transport_actor!==login)
      return fail('FEEDBACK_NATIVE_ACTOR_MISMATCH');
    if(body.length>32768)return fail('FEEDBACK_BODY_TOO_LARGE');
    try{
      if(!await qualified(issueNumber,data))return fail('HOST_FENCE_OR_KERNEL_PERMIT_UNVERIFIED');
      const issue=(await octokit.rest.issues.get({owner,repo,issue_number:issueNumber}))?.data;
      if(issue?.number!==issueNumber||issue.pull_request!==undefined||
         issue.repository_url!=='https://api.github.com/repos/'+repository)
        return fail('FEEDBACK_NATIVE_ISSUE_MISMATCH');
      const before=await scan(octokit,owner,repo,issueNumber,login,data);
      if(!before)return fail('NATIVE_SCAN_INCOMPLETE','UNKNOWN');
      const same=before.matches.filter(m=>m.comment.body===body);
      if(before.matches.length>1||(before.matches.length===1&&same.length!==1))
        return fail('DIVERGENT_FEEDBACK_IDENTITY');
      if(same.length===1)return Object.freeze({status:'PUBLISHED_VERIFIED',
        reason:'EXACT_IDEMPOTENT_NATIVE_REPLAY',publicationRef:same[0].comment.html_url,
        newPost:false,readback:'VERIFIED',...noAuthority});
      if(data.revision>1){
        const prev=before.prior;
        if(prev.length!==1||data.supersedes?.revision!==data.revision-1||
           data.supersedes.comment_ref!==prev[0].comment.html_url||
           data.supersedes.data_digest!==hash(prev[0].envelope.data))
          return fail('FEEDBACK_SUPERSESSION_UNVERIFIED');
      }
      if(!await qualified(issueNumber,data))return fail('HOST_FENCE_OR_KERNEL_PERMIT_MOVED');
      let ack;
      try{ack=(await octokit.rest.issues.createComment({owner,repo,issue_number:issueNumber,body}))?.data;}
      catch{
        const after=await scan(octokit,owner,repo,issueNumber,login,data);
        if(!after)return fail('FEEDBACK_ACK_AMBIGUOUS','UNKNOWN');
        const found=after.matches.filter(m=>m.comment.body===body);
        if(after.matches.length===1&&found.length===1){
          if(!await qualified(issueNumber,data))
            return fail('FEEDBACK_POST_LEASE_OR_PERMIT_LOST','UNKNOWN');
          return Object.freeze({status:'PUBLISHED_VERIFIED',
            reason:'LOST_ACK_RECOVERED_BY_READBACK',publicationRef:found[0].comment.html_url,
            newPost:false,readback:'VERIFIED',...noAuthority});
        }
        return fail('FEEDBACK_ACK_AMBIGUOUS','UNKNOWN');
      }
      const comment=(await octokit.rest.issues.getComment({
        owner,repo,comment_id:ack?.id}))?.data;
      if(!validNative(comment,repository,issueNumber,login,body)||comment.id!==ack?.id)
        return fail('FEEDBACK_POST_READBACK_MISMATCH','UNKNOWN');
      if(!await qualified(issueNumber,data))
        return fail('FEEDBACK_POST_LEASE_OR_PERMIT_LOST','UNKNOWN');
      return Object.freeze({status:'PUBLISHED_VERIFIED',
        reason:'NATIVE_COMMENT_READBACK_VERIFIED',publicationRef:comment.html_url,
        newPost:true,readback:'VERIFIED',...noAuthority});
    }catch{return fail('FEEDBACK_HOST_OR_NATIVE_OPERATION_FAILED','UNKNOWN');}
  }
  return Object.freeze({publishFeedback});
}
