/** Trusted new-project single-writer Claim step, NOT UI authority.
 * Native source and Issue-readback, CAS create-only Claim fence, exactly
 * one GitHub comment and verified postread are existing reviewed ports.
 * The browser may request ONLY this Host-pinned Issue, never identities.
 */
import {createHostSingleWriterClaimController}
  from '../../experiments/v2/t06d-host-claim-publication/single-writer.mjs';
import {createHostBoundNativeGitHubClaimPublisher}
  from '../../packages/tally-github-native-c/src/task-claim-github-publisher.mjs';
import {readGitHubClaimPublicationWitness}
  from '../../packages/tally-github-native-c/src/task-claim-publication-witness.mjs';
import {readProjectTaskAdmission} from './task-admission-read.mjs';
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const KIND=new Set(['chatgpt-web','codex','claude-code','human','github-actions',
  'woodpecker','other']);
const refuse=reason=>Object.freeze({status:'BLOCKED',reason,
  claimPublication:'NOT_ACCEPTED',authorized:false,admitted:false,
  taskPass:false,releasePass:false});
export function createProjectClaimHost({
  settings,env,octokit,openStore,
  createController=createHostSingleWriterClaimController,
  createPublisher=createHostBoundNativeGitHubClaimPublisher,
  readAdmission=readProjectTaskAdmission,
  readWitness=readGitHubClaimPublicationWitness
}={}){
  if(env?.TALLY_V2_CLAIM_DISPATCH!=='host-mounted')return null;
  const repository=settings?.repository,sourceSha=settings?.expectedSha;
  const issueNumber=Number(env.TALLY_V2_CLAIM_ISSUE_NUMBER);
  const writerId=env.TALLY_V2_CLAIM_WRITER_ID;
  const writerLogin=env.TALLY_V2_CLAIM_WRITER_LOGIN;
  const dispatchId=env.TALLY_V2_CLAIM_DISPATCH_ID;
  const operatorId=env.TALLY_V2_CLAIM_OPERATOR_ID;
  const operatorKind=env.TALLY_V2_CLAIM_OPERATOR_KIND;
  const registryPath=env.TALLY_V2_CLAIM_REGISTRY_PATH;
  if(!settings||env.TALLY_V2_CLAIM_REPOSITORY!==repository||
     env.TALLY_V2_CLAIM_VERSION_REF!==settings.versionId||
     env.TALLY_V2_CLAIM_SOURCE_SHA!==sourceSha||
     !SHA.test(sourceSha??'')||
     !Number.isSafeInteger(issueNumber)||issueNumber<1||
     !ID.test(writerId??'')||!LOGIN.test(writerLogin??'')||
     !ID.test(dispatchId??'')||!ID.test(operatorId??'')||
     !KIND.has(operatorKind)||
     typeof registryPath!=='string'||registryPath.length<3||
     typeof openStore!=='function'||
     typeof octokit?.rest?.issues?.createComment!=='function')
    throw new TypeError('TRUSTED_SELECTED_CLAIM_HOST_REQUIRED');
  if(env.TALLY_V2_AGENT_DISPATCH==='host-mounted'&&
     env.TALLY_V2_AGENT_CLAIM_REGISTRY_PATH!==registryPath)
    throw new TypeError('CLAIM_AGENT_REGISTRY_MISMATCH');
  const writerAuthority=Object.freeze({mode:'SINGLE_WRITER_ADMISSION',
    exclusiveHostControl:true,repository,writerId,githubWriterLogin:writerLogin});
  const publisher=createPublisher({octokit,owner:settings.owner,
    repo:settings.repo,writerAuthority});
  const issueRef='https://github.com/'+repository+'/issues/'+issueNumber;
  const source=Object.freeze({
    async readAdmissionFacts(i){
      if(i.repository!==repository||i.issueNumber!==issueNumber||
         i.sourceSha!==sourceSha||i.actorRole!=='builder'||
         i.dispatchId!==dispatchId||i.operatorId!==operatorId||
         i.operatorKind!==operatorKind)
        return null;
      const proof=await readAdmission({repository,
        versionId:settings.versionId,sourceSha,issueNumber,
        token:settings.token});
      if(proof?.connected!==true||proof.stage!=='NATIVE_PREFLIGHT_READY'||
         proof.issueUrl!==issueRef||proof.sourceSha!==sourceSha)
        return null;
      return Object.freeze({sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',
        sourceQualified:true,complete:true,issueRef,sourceSha,
        issueState:'open',workflowState:'state:ready',
        dependencyState:'CLEAR',activeIncompatibleClaimCount:0,
        dispatchSourceBound:true,publisherSourceBound:true});
    },
    async readPublishedClaim(i,commentId){
      if(i.issueRef!==issueRef||i.sourceSha!==sourceSha||i.issueNumber!==issueNumber)
        return null;
      // Reobserve the current draft body / HEAD after GitHub mutation;
      // never persist a verified row if the source became stale.
      const again=await readAdmission({repository,versionId:settings.versionId,
        sourceSha,issueNumber,token:settings.token});
      if(again?.connected!==true||again.issueUrl!==issueRef||
         again.sourceSha!==sourceSha||
         !['NATIVE_PREFLIGHT_READY','BLOCKED'].includes(again.stage)||
         ['TASK_DRAFT_NOT_OPEN','TASK_DRAFT_SOURCE_SHA_STALE',
          'NATIVE_ISSUE_DEPENDENCIES_BLOCKED','NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE',
          'TASK_CANONICAL_LABELS_REQUIRED'].includes(again.reason))
        return null;
      const witness=await readWitness({token:settings.token,
        owner:settings.owner,repo:settings.repo,issueNumber,
        expectedSourceSha:sourceSha,commentId,
        dispatchId,actorRole:'builder',operatorKind,operatorId,
        protectedClaimKey:repository+'#'+issueNumber+':builder:__default__',
        claimGeneration:1,writerLogin});
      if(witness?.status!=='PUBLICATION_WITNESSED'||
         witness.sourceQualified!==true||witness.issueRef!==issueRef||
         witness.sourceSha!==sourceSha||witness.commentId!==commentId||
         witness.commentRef!==issueRef+'#issuecomment-'+commentId||
         witness.dispatchId!==dispatchId||witness.actorRole!=='builder'||
         witness.operatorKind!==operatorKind||
         witness.operatorId!==operatorId||
         witness.protectedClaimKey!==
           repository+'#'+issueNumber+':builder:__default__'||
         witness.generation!==1||witness.transportActor!==writerLogin||
         witness.claimPublication!=='NATIVE_COMMENT_READBACK_ONLY'||
         witness.claimAccepted!==false||witness.admitted!==false||
         witness.authorized!==false||witness.taskPass!==false||
         witness.releasePass!==false)
        return null;
      return Object.freeze({sourceKind:'NATIVE_GITHUB_EXACT_READBACK',
        complete:true,edited:false,issueRef,sourceSha,
        dispatchId,actorRole:'builder',operatorId,operatorKind,
        generation:1,protectedClaimKey:repository+'#'+issueNumber+':builder:__default__',
        event:'DISPATCH_CLAIMED',dispatchState:'CLAIMED',
        commentId,commentRef:issueRef+'#issuecomment-'+commentId});
    }
  });
  return Object.freeze({repository,versionId:settings.versionId,
    sourceSha,issueNumber,
    async publish({issueNumber:requested,sourceSha:requestedSha}={}){
      if(requested!==issueNumber||requestedSha!==sourceSha)
        return refuse('CLAIM_HOST_TASK_SCOPE_MISMATCH');
      let store=null;
      try{
        store=await openStore({path:registryPath,busyTimeoutMs:1000});
        const controller=createController({store,source,publisher,writerAuthority});
        return await controller.reserveAndPublish({
          owner:settings.owner,repo:settings.repo,issueNumber,
          actorRole:'builder',sourceSha,dispatchId,operatorKind,operatorId,
          expectedGeneration:0});
      }catch{
        // An in-flight CAS reservation or POST may exist. No false NOT_RUN.
        return Object.freeze({status:'UNKNOWN',
          reason:'CLAIM_HOST_RECONCILIATION_REQUIRED',
          claimPublication:'UNKNOWN',authorized:false,admitted:false,
          taskPass:false,releasePass:false});
      }finally{if(store)await store.close().catch(()=>{});}
    }
  });
}
