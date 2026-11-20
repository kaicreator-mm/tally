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


// #312 successor: strict Host-only immutable draft and source-bound ReviewWitness.
import {createHash as productReviewSha256} from 'node:crypto';
import {readProductReviewWitnessFromClient}
  from '../../../packages/tally-github-c/src/project-intake.mjs';
export const PRODUCT312_DRAFT_KEY='tally:v2:product-intake:312:draft:';
export const PRODUCT312_FREEZE_KEY='tally:v2:product-intake:312:frozen:';
const product312Unknown=(reason,extra={})=>Object.freeze({state:'UNKNOWN',reason,
  issueNumber:312,authorized:false,admitted:false,taskPass:false,
  releasePass:false,...extra});
const product312Canonical=v=>Array.isArray(v)?v.map(product312Canonical):
  v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[
    k,product312Canonical(v[k])])):v;
// Data-only plain-object gate. The selected K kernel projects nested effect
// payload values as canonical data-only clones (null prototype), so both
// Object.prototype and null-prototype shapes are accepted here; arrays,
// null, primitives and class instances still refuse.
const product312Plain=v=>v!==null&&typeof v==='object'&&!Array.isArray(v)&&
  (Object.getPrototypeOf(v)===Object.prototype||Object.getPrototypeOf(v)===null);
const product312Draft=v=>product312Plain(v)&&
  Object.keys(v).sort().join(',')==='l2,product,provenance'&&
  product312Plain(v.product)&&product312Plain(v.l2)&&
  typeof v.product.goal==='string'&&v.product.goal.trim().length>0&&
  v.product.goal.length<=256&&Array.isArray(v.product.nonGoals)&&
  v.product.nonGoals.length<=8&&v.product.nonGoals.every(x=>
    typeof x==='string'&&x.length>0&&x.length<=256)&&
  typeof v.l2.decision==='string'&&v.l2.decision.length<=128&&
  Array.isArray(v.provenance)&&v.provenance.length<=16&&
  v.provenance.every(x=>typeof x==='string'&&x.length>0&&x.length<=256)&&
  JSON.stringify(v).length<=4096;
const product312Digest=(draft,sourceSha)=>productReviewSha256('sha256')
  .update(JSON.stringify(product312Canonical({draft,sourceSha}))).digest('hex');
/**
 * Trusted Host injection contract: accepted Claim records (not caller roles),
 * exact frozen Task Pack/write-set readback and current version ref reader.
 * A physical VERSION_ONLY shared-CAS fence is mandatory for final Freeze.
 */
export function createProductReviewHost312({openStore,verifyAcceptedClaim,
  verifyFrozenWriteSet,readReviewWitness,readCurrentVersion,
  withVersionFence,octokit,env=process.env}={}){
  // GitHub C→Host wiring: an explicitly injected witness reader wins (tests,
  // alternative selected transports); without one, a Host-injected selected C
  // client (octokit) binds the REAL read-only #312 ReviewWitness validator.
  // Neither present still fails closed at the fence gate below.
  const witnessReader=typeof readReviewWitness==='function'?readReviewWitness:
    typeof octokit?.rest?.issues?.getComment==='function'?
      request=>readProductReviewWitnessFromClient({octokit,...request}):undefined;
  const registry=env.TALLY_TASK_CLAIM_312_REGISTRY_PATH;
  const sourceOk=sourceSha=>SHA.test(sourceSha??'')&&
    sourceSha===env.TALLY_CLAIM_SOURCE_SHA&&typeof registry==='string'&&
    registry.length>0;
  const storeCall=async fn=>{
    let store;
    try{
      store=await openStore({path:registry,busyTimeoutMs:1000});
      return await fn(store);
    }finally{if(store)await store.close();}
  };
  const admitted=async(input,role)=>{
    const evidence=await verifyAcceptedClaim(312,role);
    const writeSet=await verifyFrozenWriteSet(312,input.sourceSha);
    const claim=evidence?.claim;
    if(evidence?.ok!==true||claim?.accepted!==true||
      claim.issueNumber!==312||claim.role!==role||
      claim.sourceSha!==input.sourceSha||
      claim.taskPackBlobSha!==input.taskPackBlobSha||
      claim.writeSetSha256!==input.writeSetSha256||
      writeSet?.ok!==true||writeSet.taskPackBlobSha!==input.taskPackBlobSha||
      writeSet.writeSetSha256!==input.writeSetSha256)
      return null;
    return claim;
  };
  const current=async sourceSha=>{
    const observed=await readCurrentVersion();
    return observed?.ok===true&&observed.sha===sourceSha;
  };
  return Object.freeze({
    async persistDraft(input){
      if(typeof openStore!=='function'||typeof verifyAcceptedClaim!=='function'||
        typeof verifyFrozenWriteSet!=='function'||typeof readCurrentVersion!=='function')
        return product312Unknown('PRODUCT_REVIEW_HOST_UNAVAILABLE');
      if(!sourceOk(input?.sourceSha)||!SHA.test(input?.taskPackBlobSha??'')||
        !DIGEST.test(input?.writeSetSha256??'')||!product312Draft(input?.draft))
        return product312Unknown('PRODUCT_DRAFT_INPUT_INVALID');
      try{
        const builder=await admitted(input,'builder');
        if(!builder||!await current(input.sourceSha))
          return product312Unknown('PRODUCT_DRAFT_SOURCE_OR_CLAIM_STALE');
        const digest=product312Digest(input.draft,input.sourceSha);
        const key=PRODUCT312_DRAFT_KEY+input.sourceSha;
        const draftRef='draft:sha256:'+digest;
        const record={phase:'DRAFT_PENDING_REVIEW',issueNumber:312,
          sourceSha:input.sourceSha,productDigest:digest,draftRef,
          taskPackBlobSha:input.taskPackBlobSha,
          writeSetSha256:input.writeSetSha256,
          builderClaimRef:builder.claimRef,
          draft:product312Canonical(input.draft)};
        return await storeCall(async store=>{
          const prior=await store.get(key);
          if(prior&&JSON.stringify(prior.value)!==JSON.stringify(record))
            return product312Unknown('PRODUCT_DRAFT_ALREADY_EXISTS');
          if(!prior)await store.put(key,record,{mustBeAbsent:true});
          const saved=(await store.get(key))?.value;
          if(JSON.stringify(saved)!==JSON.stringify(record))
            return product312Unknown('PRODUCT_DRAFT_READBACK_UNVERIFIED');
          return Object.freeze({state:'DRAFT_PENDING_REVIEW',
            reason:'HOST_DRAFT_PERSISTED_AND_READ_BACK',draftRef,
            productDigest:digest,sourceSha:input.sourceSha,
            taskPackBlobSha:input.taskPackBlobSha,
            writeSetSha256:input.writeSetSha256,
            issueNumber:312,authorized:false,admitted:false,
            taskPass:false,releasePass:false});
        });
      }catch{return product312Unknown('PRODUCT_DRAFT_CAS_UNKNOWN');}
    },
    async freezeReviewedDraft(input){
      if(typeof openStore!=='function'||typeof verifyAcceptedClaim!=='function'||
        typeof verifyFrozenWriteSet!=='function'||typeof witnessReader!=='function'||
        typeof readCurrentVersion!=='function'||typeof withVersionFence!=='function')
        return product312Unknown('PRODUCT_REVIEW_SHARED_HOST_FENCE_REQUIRED');
      if(!sourceOk(input?.sourceSha)||!SHA.test(input?.taskPackBlobSha??'')||
        !DIGEST.test(input?.writeSetSha256??'')||
        !Number.isSafeInteger(input?.reviewCommentId)||input.reviewCommentId<1||
        !SHA.test(input?.reviewedHead??'')||!SHA.test(input?.reviewedTree??'')||
        !SHA.test(input?.reviewedBase??''))
        return product312Unknown('PRODUCT_REVIEW_FREEZE_INPUT_INVALID');
      try{
        const builder=await admitted(input,'builder');
        const reviewer=await admitted(input,'reviewer');
        if(!builder||!reviewer||
          builder.operatorId===reviewer.operatorId||
          builder.writerId===reviewer.writerId)
          return product312Unknown('PRODUCT_REVIEW_PROCESS_OR_CLAIM_UNVERIFIED');
        const draft=await storeCall(async store=>
          (await store.get(PRODUCT312_DRAFT_KEY+input.sourceSha))?.value);
        if(draft?.phase!=='DRAFT_PENDING_REVIEW'||
          draft.sourceSha!==input.sourceSha||
          draft.taskPackBlobSha!==input.taskPackBlobSha||
          draft.writeSetSha256!==input.writeSetSha256||
          draft.builderClaimRef!==builder.claimRef||
          draft.productDigest!==product312Digest(draft.draft,input.sourceSha))
          return product312Unknown('PRODUCT_DRAFT_NOT_PERSISTED_OR_STALE');
        const expected={issueNumber:312,sourceSha:input.sourceSha,
          taskPackBlobSha:input.taskPackBlobSha,
          writeSetSha256:input.writeSetSha256,
          productDigest:draft.productDigest,draftRef:draft.draftRef,
          reviewedHead:input.reviewedHead,reviewedTree:input.reviewedTree,
          reviewedBase:input.reviewedBase};
        const validWitness=w=>w?.status==='INDEPENDENTLY_VERIFIED'&&
          w.independentlyVerified===true&&w.sourceQualified===true&&
          w.authorized===false&&w.taskPass===false&&w.releasePass===false&&
          w.sourceSha===input.sourceSha&&
          w.productDigest===draft.productDigest&&w.draftRef===draft.draftRef&&
          w.taskPackBlobSha===input.taskPackBlobSha&&
          w.writeSetSha256===input.writeSetSha256&&
          w.reviewedHead===input.reviewedHead&&
          w.reviewedTree===input.reviewedTree&&
          w.reviewedBase===input.reviewedBase&&
          w.reviewerOperatorId===reviewer.operatorId&&
          w.reviewerWriterId===reviewer.writerId&&
          typeof w.reviewRef==='string'&&
          /^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/312#issuecomment-[1-9][0-9]*$/.test(w.reviewRef)&&
          w.p0===0&&w.p1===0&&Number.isSafeInteger(w.p2)&&
          w.p2>=0&&w.p2<=32;
        const witnessRequest={commentId:input.reviewCommentId,
          expected,builderClaim:builder,reviewerClaim:reviewer};
        // First read-only qualification must not reserve a shared effect fence.
        const preflight=await witnessReader(witnessRequest);
        if(!validWitness(preflight))
          return product312Unknown('PRODUCT_REVIEW_WITNESS_UNVERIFIED');
        if(!await current(input.sourceSha))
          return product312Unknown('PRODUCT_REVIEW_SOURCE_MOVED');
        return await withVersionFence({sourceSha:input.sourceSha,
          operation:'PRODUCT_REVIEW_FREEZE_312'},async()=>{
          // Recheck the *actual* GitHub comment after obtaining exclusive
          // shared CAS fence. An edit between preflight and fence is fatal.
          const witness=await witnessReader(witnessRequest);
          if(!validWitness(witness))
            return product312Unknown('PRODUCT_REVIEW_WITNESS_UNVERIFIED');
          // Final branch currentness check follows the awaited witness read.
          if(!await current(input.sourceSha))
            return product312Unknown('PRODUCT_REVIEW_SOURCE_MOVED');
          return storeCall(async store=>{
            const key=PRODUCT312_FREEZE_KEY+input.sourceSha;
            const value={phase:'FROZEN_PRODUCT_INTAKE',issueNumber:312,
              sourceSha:input.sourceSha,draftRef:draft.draftRef,
              productDigest:draft.productDigest,
              taskPackBlobSha:input.taskPackBlobSha,
              writeSetSha256:input.writeSetSha256,
              builderClaimRef:builder.claimRef,
              reviewerClaimRef:reviewer.claimRef,
              reviewRef:witness.reviewRef};
            const prior=await store.get(key);
            if(prior&&JSON.stringify(prior.value)!==JSON.stringify(value))
              return product312Unknown('PRODUCT_REVIEW_FREEZE_CONFLICT');
            // Re-read version immediately before a create-only CAS effect,
            // after all asynchronous witness/store GET operations.
            if(!prior){
              if(!await current(input.sourceSha))
                return product312Unknown('PRODUCT_REVIEW_SOURCE_MOVED');
              await store.put(key,value,{mustBeAbsent:true});
            }
            const saved=(await store.get(key))?.value;
            if(JSON.stringify(saved)!==JSON.stringify(value))
              return product312Unknown('PRODUCT_REVIEW_FREEZE_READBACK_UNVERIFIED');
            return Object.freeze({state:'FROZEN_COMPLETE',
              reason:'SOURCE_BOUND_HOST_CAS_READBACK_VERIFIED',
              issueNumber:312,sourceSha:input.sourceSha,
              taskPackBlobSha:input.taskPackBlobSha,
              writeSetSha256:input.writeSetSha256,
              productDigest:draft.productDigest,draftRef:draft.draftRef,
              reviewRef:witness.reviewRef,receiptRef:'receipt:sha256:'+
              draft.productDigest,frozen:true,
              authorized:false,admitted:false,taskPass:false,releasePass:false});
          });
        });
      }catch{return product312Unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');}
    }
  });
}
