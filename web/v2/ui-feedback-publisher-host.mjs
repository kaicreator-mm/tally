/**
 * #274: UI → selected K/Host feedback publisher bridge (not a publisher).
 * The browser sends a bounded intent identifier; only Host derives the
 * schema payload, owner grant, GitHub writer, review and exact native ref.
 * No shared server mount, direct GitHub API, task-meta API or activation.
 */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const flags={authorized:false,admitted:false,taskPass:false,
  releasePass:false,measuredBenefit:'UNKNOWN',autoActivate:false};
const denied=reason=>Object.freeze({status:'BLOCKED',reason,
  publicationRef:null,...flags});
const unknown=reason=>Object.freeze({status:'UNKNOWN',reason,
  publicationRef:null,reconciliationRequired:true,...flags});
const requestKeys='feedbackId,issueNumber,repository,revision,sourceSha,versionId';
const ref=x=>typeof x==='string'&&
  /^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*#issuecomment-[1-9][0-9]*$/.test(x);
function qualifiedProposal(x,q) {
  return plain(x)&&x.status==='VERIFIED'&&x.currentness==='CURRENT'&&
    x.repository===q.repository&&x.versionId===q.versionId&&
    x.sourceSha===q.sourceSha&&x.feedbackId===q.feedbackId&&
    x.revision===q.revision&&x.issueNumber===q.issueNumber&&
    x.state==='REVIEWED'&&ref(x.proposalRef)&&
    x.proposalRef.startsWith('https://github.com/'+q.repository+
      '/issues/'+q.issueNumber+'#issuecomment-')&&
    typeof x.digest==='string'&&/^[a-f0-9]{64}$/.test(x.digest);
}
function identical(a,b){
  return a?.proposalRef===b?.proposalRef&&a?.digest===b?.digest&&
    a?.revision===b?.revision&&a?.sourceSha===b?.sourceSha&&
    a?.currentness===b?.currentness;
}
export function createUiFeedbackPublisherHost({
  repository,versionId,expectedSha,selectedHost
}={}) {
  if(!REPO.test(repository??'')||versionId!=='version/v2'||!SHA.test(expectedSha??''))
    throw new TypeError('FEEDBACK_SELECTED_SOURCE_REQUIRED');
  async function publish(request) {
    if(!plain(request)||Object.keys(request).sort().join(',')!==requestKeys||
      request.repository!==repository||request.versionId!==versionId||
      request.sourceSha!==expectedSha||
      !ID.test(request.feedbackId??'')||
      !Number.isSafeInteger(request.issueNumber)||request.issueNumber<1||
      !Number.isSafeInteger(request.revision)||request.revision<1)
      return denied('FEEDBACK_UI_REQUEST_INVALID');
    if(typeof selectedHost?.readFeedbackProposal!=='function'||
       typeof selectedHost?.dispatchFeedbackPublication!=='function'||
       typeof selectedHost?.readFeedbackPublication!=='function')
      return denied('FEEDBACK_SELECTED_HOST_PORT_UNAVAILABLE');
    const q={repository,versionId,sourceSha:expectedSha,
      feedbackId:request.feedbackId,revision:request.revision,
      issueNumber:request.issueNumber};
    try {
      const first=await selectedHost.readFeedbackProposal(q);
      if(!qualifiedProposal(first,q))return denied('FEEDBACK_SOURCE_OR_REVIEW_UNVERIFIED');
      const second=await selectedHost.readFeedbackProposal(q);
      if(!qualifiedProposal(second,q)||!identical(first,second))
        return denied('FEEDBACK_PROPOSAL_MOVED');
      // Host side derives the ADS wire, current Owner permission and native
      // effect. Never forward UI actor, K token, report YAML or Boolean grant.
      let ack=null;
      try{ack=await selectedHost.dispatchFeedbackPublication({...q,
        proposalRef:first.proposalRef,proposalDigest:first.digest});}
      catch{/* Lost ACK reconciled only through the selected Host readback. */}
      const published=await selectedHost.readFeedbackPublication(q);
      if(!plain(published)||published.status!=='VERIFIED'||
        published.repository!==repository||published.versionId!==versionId||
        published.sourceSha!==expectedSha||published.feedbackId!==q.feedbackId||
        published.revision!==q.revision||published.issueNumber!==q.issueNumber||
        published.proposalDigest!==first.digest||
        !ref(published.publicationRef)||
        !published.publicationRef.startsWith('https://github.com/'+repository+
          '/issues/'+q.issueNumber+'#issuecomment-'))
        return unknown('FEEDBACK_NATIVE_PUBLICATION_UNCONFIRMED');
      // An ACK alone does not grant successful publication. A verified native
      // report is observed publication only, not measured outcome or Task PASS.
      return Object.freeze({status:'PUBLISHED_VERIFIED',
        reason:ack?.status==='PUBLISHED_VERIFIED'?
          'HOST_DISPATCH_AND_NATIVE_READBACK':'ACK_RECONCILED_FROM_HOST_READBACK',
        publicationRef:published.publicationRef,sourceSha:expectedSha,
        feedbackId:q.feedbackId,revision:q.revision,...flags});
    }catch{return unknown('FEEDBACK_HOST_READBACK_FAILED');}
  }
  return Object.freeze({publish});
}
