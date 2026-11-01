/** J07b selected Host gate prototype. NOT wired to a live DomainHarness K.
 * Host-supplied ports MUST be connected to native verified GitHub/ADS source;
 * never bind these ports directly to UI/Agent booleans or J07a projections.
 * The default environment is DRY_RUN: no GitHub mutation is allowed.
 */
import {createHash} from 'node:crypto';
import {readReleaseCandidateFromClient,mergeReleaseCandidateFromClient,
  createQualifiedTagFromClient} from '../../../packages/tally-github-c/src/release-delivery.mjs';
const SHA=/^[a-f0-9]{40}$/;
const safe=(s)=>typeof s==='string'&&s.length>0;
const output=(state,reason,more={})=>Object.freeze({state,reason,
  authorized:false,taskPass:false,releasePass:false,canPublish:false,...more});
const no=(reason)=>output('NOT_ADMITTED',reason);
const digest=value=>createHash('sha256').update(JSON.stringify(value)).digest('hex');
const claimSource=env=>SHA.test(env.TALLY_TASK_CLAIM_222_SOURCE_SHA??'')?
  env.TALLY_TASK_CLAIM_222_SOURCE_SHA:null;
const selected=(input,env)=>claimSource(env)!==null&&SHA.test(input?.headSha??'')&&
  SHA.test(input?.headTree??'')&&SHA.test(input?.targetSha??'')&&
  SHA.test(input?.targetTree??'')&&input.headSha===env.TALLY_RELEASE_SOURCE_SHA&&
  Number.isSafeInteger(input?.pullNumber)&&input.pullNumber>0;
const fresh=(record,input,operation)=>{
  const receipt=record?.qualification??record;
  return receipt?.state==='QUALIFIED'&&receipt?.verifierIndependentlyBound===true&&
    receipt?.sourceSha===input.headSha&&receipt?.sourceTree===input.headTree&&
    receipt?.targetSha===input.targetSha&&receipt?.targetTree===input.targetTree&&
    receipt?.operation===operation&&typeof receipt?.nativeEvidenceRef==='string'&&
    receipt.nativeEvidenceRef.startsWith('https://github.com/kaicreator-mm/tally/')&&
    !receipt.nativeEvidenceRef.includes('?');
};
const approved=(grant,input,operation)=>grant?.state==='GRANTED'&&
  grant?.exactScope==='kaicreator-mm/tally:version/v2->main'&&
  grant?.sourceSha===input.headSha&&grant?.targetSha===input.targetSha&&
  grant?.operation===operation&&safe(grant?.ownerIdentity)&&
  safe(grant?.grantRef)&&grant.grantRef.startsWith('https://github.com/kaicreator-mm/tally/');
const stageRef=(operation,input)=>`tally:v2:release:${operation}:${digest([
  input.pullNumber,input.headSha,input.headTree,input.targetSha,input.targetTree])}`;
/** Store is the same privileged CAS store used by the selected Host, not a
 * new task database. A pending operation must NEVER be blindly reexecuted. */
export function createReleaseHost({verifyNativeClaim,verifyWriteSet,readIndependentRq,
  readOwnerGrant,openCasStore,octokit,env={}}={}){
  const portsReady=[verifyNativeClaim,verifyWriteSet,readIndependentRq,
    readOwnerGrant,openCasStore].every(x=>typeof x==='function');
  async function qualify(input,operation){
    if(!portsReady)return no('RELEASE_HOST_PORTS_NOT_CONNECTED');
    if(!selected(input,env))return no('RELEASE_SOURCE_NOT_PINNED');
    let claim,scope,rq,grant;
    try{
      claim=await verifyNativeClaim({issueNumber:222,role:'builder',
        sourceSha:claimSource(env)});
      if(claim?.state!=='ACCEPTED'||claim?.sourceSha!==claimSource(env))
        return no('NATIVE_CLAIM_NOT_VERIFIED');
      scope=await verifyWriteSet({issueNumber:222,sourceSha:claimSource(env)});
      if(scope?.state!=='FROZEN_VERIFIED')return no('TASK_WRITESET_NOT_VERIFIED');
      rq=await readIndependentRq({sourceSha:input.headSha,sourceTree:input.headTree,
        targetSha:input.targetSha,targetTree:input.targetTree,operation});
      if(!fresh(rq,input,operation))return no('INDEPENDENT_RQ_NOT_QUALIFIED');
      grant=await readOwnerGrant({sourceSha:input.headSha,targetSha:input.targetSha,operation});
      if(!approved(grant,input,operation))return no('OWNER_EXACT_RELEASE_SCOPE_MISSING');
    }catch{return no('HOST_QUALIFICATION_READ_UNKNOWN');}
    return output('HOST_GATES_OBSERVED','REQUIRES_CURRENT_TARGET_PREFLIGHT',
      {headSha:input.headSha,targetSha:input.targetSha});
  }
  return Object.freeze({
    async preflight(input={}){
      const admitted=await qualify(input,'MERGE');
      if(admitted.state!=='HOST_GATES_OBSERVED')return admitted;
      return readReleaseCandidateFromClient({...input,octokit});
    },
    async merge(input={}){
      const admitted=await qualify(input,'MERGE');
      if(admitted.state!=='HOST_GATES_OBSERVED')return admitted;
      if(env.TALLY_RELEASE_PHYSICAL_EFFECTS!=='1')return no('RELEASE_PHYSICAL_EFFECT_DISABLED');
      if(!safe(env.TALLY_RELEASE_CAS_PATH))return no('RELEASE_HOST_CAS_UNAVAILABLE');
      // Do not reserve a durable operation for a missing transport or a stale
      // target. The mutation path repeats the exact preflight after CAS.
      const preflight=await readReleaseCandidateFromClient({...input,octokit});
      if(preflight.state!=='EXACT_PREFLIGHT_OBSERVED')return output(
        preflight.state,preflight.reason,{githubEffect:'NOT_RUN'});
      const key=stageRef('merge',input);
      let store;
      try{
        store=await openCasStore({path:env.TALLY_RELEASE_CAS_PATH});
        if(await store.get(key))return output('RECONCILIATION_REQUIRED','PREVIOUS_RELEASE_OPERATION_PENDING',
          {operationKey:key});
        await store.put(key,{state:'PENDING_RELEASE_MERGE',sourceSha:input.headSha,
          targetSha:input.targetSha,operationKey:key},{mustBeAbsent:true});
      }catch{return no('RELEASE_HOST_CAS_UNAVAILABLE_OR_CONFLICT');}
      finally{await store?.close?.();}
      const result=await mergeReleaseCandidateFromClient({...input,octokit});
      return output(result.state,result.reason,{githubEffect:result.state==='MERGED_REQUALIFY_REQUIRED'?
        'PERFORMED_READBACK_VERIFIED':'UNKNOWN',
      operationKey:key,mergedSha:result.mergedSha??null,mergedTree:result.mergedTree??null,
      next:'REQUALIFY_NEW_MAIN_SHA_AND_TREE_BEFORE_TAG'});
    },
    async tag(input={}){
      // Tag is separately qualified against the NEW merged main source/tree;
      // old premerge RQ and grant never apply to this operation.
      if(!portsReady)return no('RELEASE_HOST_PORTS_NOT_CONNECTED');
      if(!SHA.test(input.commitSha??'')||!SHA.test(input.commitTree??'')||
        !claimSource(env)||input.commitSha!==env.TALLY_RELEASE_TAG_SOURCE_SHA)return no('TAG_SOURCE_NOT_PINNED');
      let grant,rq,claim,scope;
      try{
        claim=await verifyNativeClaim({issueNumber:222,role:'builder',
          sourceSha:claimSource(env)});
        if(claim?.state!=='ACCEPTED'||claim?.sourceSha!==claimSource(env))
          return no('NATIVE_CLAIM_NOT_VERIFIED');
        scope=await verifyWriteSet({issueNumber:222,sourceSha:claimSource(env)});
        if(scope?.state!=='FROZEN_VERIFIED')return no('TASK_WRITESET_NOT_VERIFIED');
        rq=await readIndependentRq({sourceSha:input.commitSha,sourceTree:input.commitTree,
          targetSha:input.commitSha,targetTree:input.commitTree,operation:'TAG'});
        grant=await readOwnerGrant({sourceSha:input.commitSha,targetSha:input.commitSha,operation:'TAG'});
      }catch{return no('POST_MERGE_RQ_READ_UNKNOWN');}
      const exact={headSha:input.commitSha,headTree:input.commitTree,
        targetSha:input.commitSha,targetTree:input.commitTree};
      if(!fresh(rq,exact,'TAG')||!approved(grant,exact,'TAG'))
        return no('POST_MERGE_RQ_OR_GRANT_NOT_VERIFIED');
      if(env.TALLY_RELEASE_PHYSICAL_EFFECTS!=='1')return no('RELEASE_PHYSICAL_EFFECT_DISABLED');
      // Separate Host CAS and release publication wiring are still required.
      // This deliberate fail-closed bound avoids converting a Tag helper into
      // release authorization without verified no-duplicate reconciliation.
      return no('TAG_EFFECT_REQUIRES_HOST_CAS_AND_RELEASE_PUBLICATION_BINDING');
    }
  });
}
