/**
 * J01b/J02 Host-only Project intake effect (#210).
 *
 * The Host owns every privileged step: accepted-Claim verification, frozen
 * DAG write-set verification, the FROZEN_PRODUCT_INTAKE CAS record written
 * only after an independent review, and the narrow GitHub project effect with
 * canonical readback. Verifiers/transports are injected by the sealed
 * provider; this module holds no credential and never trusts caller
 * authority. Every deviation fails closed with a bounded reason, and an
 * unprovable GitHub effect stays explicitly NOT_RUN instead of being faked.
 */
const OWNER='kaicreator-mm',REPO='tally',ISSUE=210;
const ISSUE_REF=`https://github.com/${OWNER}/${REPO}/issues/${ISSUE}`;
const SHA=/^[0-9a-f]{40}$/,DIGEST=/^[0-9a-f]{64}$/;
const TITLE=/^.{1,120}$/s;
const ISO=value=>typeof value==='string'&&Number.isFinite(Date.parse(value))&&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
export const PRODUCT_INTAKE_KEY=`tally:v2:product-intake:${ISSUE}`;
const denied=(input,reason,githubEffect='NOT_RUN')=>Object.freeze({
  state:'NOT_ADMITTED',reason,issueRef:ISSUE_REF,
  sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
  productDigest:DIGEST.test(input?.productDigest??'')?input.productDigest:null,
  claimAccepted:false,admitted:false,authorized:false,agentLaunch:false,
  claimGeneration:1,replayFixture:false,githubEffect,taskPass:false,releasePass:false});
const base=input=>({issueRef:ISSUE_REF,sourceSha:input.sourceSha,
  productDigest:input.productDigest,claimGeneration:1,authorized:false,
  agentLaunch:false,taskPass:false,releasePass:false});

/**
 * Freeze one independently reviewed Product intake draft into the Host CAS.
 * This is the ONLY writer of PRODUCT_INTAKE_KEY: the reviewer must differ
 * from the author, the digest must be the exact reviewed draft digest, and
 * the record is pinned to the current source revision. CAS create-conflict
 * replays the existing frozen record (idempotent, never a second freeze).
 */
export async function freezeProductIntake({openStore,registryPath,sourceSha,
  productDigest,author,reviewer,frozenAt,caseRefDigest}={}){
  const unqualified=reason=>Object.freeze({state:'NOT_FROZEN',reason,
    issueRef:ISSUE_REF,sourceSha:SHA.test(sourceSha??'')?sourceSha:null,
    productDigest:DIGEST.test(productDigest??'')?productDigest:null,
    claimGeneration:1,frozen:false,replayFixture:false,
    authorized:false,admitted:false,taskPass:false,releasePass:false});
  if(typeof openStore!=='function'||typeof registryPath!=='string'||!registryPath)
    return unqualified('PRODUCT_FREEZE_STORE_UNAVAILABLE');
  if(!SHA.test(sourceSha??'')||!DIGEST.test(productDigest??'')||!ISO(frozenAt??''))
    return unqualified('PRODUCT_FREEZE_INPUT_INVALID');
  if(typeof author!=='string'||author.length===0||author.length>128||
    typeof reviewer!=='string'||reviewer.length===0||reviewer.length>128||
    author===reviewer)
    return unqualified('PRODUCT_REVIEW_NOT_INDEPENDENT');
  let store,existing;
  try{
    store=await openStore({path:registryPath,busyTimeoutMs:1000});
    existing=await store.get(PRODUCT_INTAKE_KEY);
  }catch{
    return unqualified('PRODUCT_FREEZE_STORE_UNAVAILABLE');
  }finally{if(store)await store.close();}
  if(existing?.value?.phase==='FROZEN_PRODUCT_INTAKE'){
    if(existing.value.productDigest===productDigest&&
      existing.value.sourceSha===sourceSha&&existing.value.review?.reviewer===reviewer)
      return Object.freeze({...unqualified('PRODUCT_FREEZE_ALREADY_RECORDED'),
        state:'FROZEN_COMPLETE',frozen:true,
        review:Object.freeze({author:String(existing.value.review.author),
          reviewer:String(existing.value.review.reviewer)}),
        receiptRef:`receipt:sha256:${productDigest}`});
    return unqualified('PRODUCT_FREEZE_DIGEST_MISMATCH');
  }
  const value=Object.freeze({phase:'FROZEN_PRODUCT_INTAKE',issueRef:ISSUE_REF,
    sourceSha,productDigest,caseRefDigest:typeof caseRefDigest==='string'?caseRefDigest:null,
    review:Object.freeze({author,reviewer}),frozenAt,
    authorized:false,admitted:false,taskPass:false,releasePass:false});
  try{
    store=await openStore({path:registryPath,busyTimeoutMs:1000});
    await store.put(PRODUCT_INTAKE_KEY,value,{mustBeAbsent:true});
  }catch{
    return unqualified('PRODUCT_FREEZE_STORE_UNAVAILABLE');
  }finally{if(store)await store.close();}
  return Object.freeze({state:'FROZEN_COMPLETE',reason:'INDEPENDENT_REVIEW_FROZEN_IN_HOST_CAS',
    issueRef:ISSUE_REF,sourceSha,productDigest,
    receiptRef:`receipt:sha256:${productDigest}`,frozen:true,claimGeneration:1,
    review:Object.freeze({author,reviewer}),
    replayFixture:false,authorized:false,admitted:false,taskPass:false,releasePass:false});
}

/**
 * J01b governed effect: create/import the target project and verify the
 * canonical GitHub readback. Requires, in order: host pin + accepted Claim,
 * frozen DAG write-set, frozen independent Product review record, and a real
 * transport. Safe replay never claims a GitHub effect.
 */
export function createProjectIntakeHost({openStore,verifyAcceptedClaim,
  verifyFrozenWriteSet,readProjectIntake,publish,env=process.env}={}){
  return Object.freeze({async hostProjectIntake(input){
    if(typeof verifyAcceptedClaim!=='function'||typeof verifyFrozenWriteSet!=='function')
      return denied(input,'PROJECT_INTAKE_HOST_CONFIG_INVALID');
    if(input?.issueNumber!==ISSUE||!SHA.test(input?.sourceSha??'')||
      !DIGEST.test(input?.productDigest??'')||!TITLE.test(input?.projectTitle??'')||
      input.sourceSha!==env.TALLY_CLAIM_SOURCE_SHA||
      typeof env.TALLY_TASK_CLAIM_210_REGISTRY_PATH!=='string'||
      !env.TALLY_TASK_CLAIM_210_REGISTRY_PATH)
      return denied(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
    const claim=await verifyAcceptedClaim(ISSUE,'builder');
    if(!claim.ok)return denied(input,claim.reason);
    const writeSet=await verifyFrozenWriteSet(ISSUE,input.sourceSha);
    if(!writeSet.ok)return denied(input,writeSet.reason);
    let store,record;
    try{
      store=await openStore({path:env.TALLY_TASK_CLAIM_210_REGISTRY_PATH,busyTimeoutMs:1000});
      record=await store.get(PRODUCT_INTAKE_KEY);
    }catch{return denied(input,'PRODUCT_FREEZE_STORE_UNAVAILABLE');}
    finally{if(store)await store.close();}
    const value=record?.value;
    if(value?.phase!=='FROZEN_PRODUCT_INTAKE')
      return denied(input,'PRODUCT_FREEZE_NOT_VERIFIED');
    if(value.sourceSha!==input.sourceSha||value.productDigest!==input.productDigest||
      value.issueRef!==ISSUE_REF)
      return denied(input,'PRODUCT_FREEZE_DIGEST_MISMATCH');
    if(typeof value.review?.reviewer!=='string'||value.review.reviewer===value.review?.author)
      return denied(input,'PRODUCT_REVIEW_NOT_INDEPENDENT');
    if(env.TALLY_LOCAL_REPLAY==='1')
      return Object.freeze({...base(input),state:'REPLAY_COMPLETE',
        reason:'REPLAY_FIXTURE_ONLY',claimAccepted:true,admitted:true,
        replayFixture:true,githubEffect:'NOT_RUN',project:null});
    if(typeof publish!=='function'||typeof readProjectIntake!=='function'||
      typeof env.TALLY_GITHUB_READ_TOKEN!=='string'||env.TALLY_GITHUB_READ_TOKEN.length<8)
      return denied(input,'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE');
    let published;
    try{
      published=await publish({issueNumber:ISSUE,issueRef:ISSUE_REF,
        sourceSha:input.sourceSha,productDigest:input.productDigest,
        projectTitle:input.projectTitle});
    }catch{return Object.freeze({...base(input),state:'PROJECT_INTAKE_UNKNOWN',
      reason:'GITHUB_PUBLISH_UNKNOWN',claimAccepted:true,admitted:true,
      replayFixture:false,githubEffect:'UNKNOWN',project:null});}
    if(published?.state==='PROJECT_EXISTS')
      return Object.freeze({...base(input),state:'PROJECT_INTAKE_ADMITTED',
        reason:'PROJECT_INTAKE_READBACK_ALREADY_PRESENT',claimAccepted:true,
        admitted:true,replayFixture:false,githubEffect:'PERFORMED_VERIFIED',
        repository:`${OWNER}/${REPO}`,project:published.project});
    if(published?.state!=='PROJECT_CREATED'||!published.project?.number||
      !TITLE.test(published.project.title??''))
      return Object.freeze({...base(input),state:'PROJECT_INTAKE_UNKNOWN',
        reason:'GITHUB_PUBLISH_UNKNOWN',claimAccepted:true,admitted:true,
        replayFixture:false,githubEffect:'UNKNOWN',project:null});
    const readback=await readProjectIntake({owner:OWNER,repo:REPO,
      expectedSourceSha:input.sourceSha,projectTitle:input.projectTitle});
    const match=readback?.status==='PROJECT_INTAKE_OBSERVED'&&
      readback.projects.find(p=>p.number===published.project.number&&
        p.title===input.projectTitle);
    if(!match)
      return Object.freeze({...base(input),state:'PROJECT_INTAKE_UNKNOWN',
        reason:'PROJECT_INTAKE_READBACK_MISMATCH',claimAccepted:true,
        admitted:true,replayFixture:false,githubEffect:'UNKNOWN',project:null});
    return Object.freeze({...base(input),state:'PROJECT_INTAKE_ADMITTED',
      reason:'PROJECT_INTAKE_PUBLISHED_AND_READBACK_VERIFIED',
      claimAccepted:true,admitted:true,replayFixture:false,
      githubEffect:'PERFORMED_VERIFIED',repository:`${OWNER}/${REPO}`,
      project:Object.freeze({number:match.number,title:match.title,
        url:match.url,itemCount:match.itemCount})});
  }});
}
