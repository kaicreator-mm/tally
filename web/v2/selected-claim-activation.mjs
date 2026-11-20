/** Durable Claim receipt -> trusted Host process pins (not authorization).
 * The existing BUILD-sealed C and real K must STILL independently verify
 * current native comment, CAS row, version Task and FSW before Agent effect.
 */
import {createHash} from 'node:crypto';
import {resolveBuilderBuildTarget,exactSelectedWriteSet}
  from '../../experiments/v2/selected-business-build/builder-target.mjs';
const SHA=/^[a-f0-9]{40}$/;
const fail=reason=>Object.freeze({status:'BLOCKED',reason,agentAuthorized:false});
export async function resolveSelectedAgentClaimPins({
  env={},settings,openStore
}={}){
  const repository=settings?.repository,sha=settings?.expectedSha;
  const issue=Number(env.TALLY_LOCAL_ISSUE_NUMBER);
  if(!SHA.test(sha??'')||
     env.TALLY_V2_CLAIM_DISPATCH!=='host-mounted'||
     env.TALLY_V2_AGENT_DISPATCH!=='host-mounted'||
     repository!==env.TALLY_V2_CLAIM_REPOSITORY||
     env.TALLY_V2_CLAIM_VERSION_REF!==settings?.versionId||
     env.TALLY_V2_CLAIM_SOURCE_SHA!==sha||
     env.TALLY_V2_AGENT_CLAIM_SOURCE_SHA!==sha||
     issue!==Number(env.TALLY_V2_CLAIM_ISSUE_NUMBER)||
     !Number.isSafeInteger(issue)||issue<1||
     env.TALLY_V2_CLAIM_REGISTRY_PATH!==env.TALLY_V2_AGENT_CLAIM_REGISTRY_PATH||
     !env.TALLY_V2_CLAIM_REGISTRY_PATH||
     typeof openStore!=='function')
    return fail('AGENT_CLAIM_SOURCE_OR_REGISTRY_MISMATCH');
  const key=repository+'#'+issue+':builder:__default__';
  const digest=createHash('sha256').update(key).digest('hex');
  const registryKey='tally:v2:claim-admission:'+digest;
  let store,row,writeSetRow;
  try{
    store=await openStore({
      path:env.TALLY_V2_CLAIM_REGISTRY_PATH,busyTimeoutMs:1000});
    row=await store.get(registryKey);
    if(env.TALLY_V2_WRITESET_FREEZE==='host-mounted')
      writeSetRow=await store.get('tally:v2:task-writeset:'+issue);
  }catch{return fail('AGENT_CLAIM_REGISTRY_UNAVAILABLE');}
  finally{if(store)await store.close().catch(()=>{});}
  const val=row?.value;
  if(!Number.isSafeInteger(row?.revision)||row.revision<1||
     val?.phase!=='PUBLISHED_VERIFIED'||
     val.protectedClaimKey!==key||
     val.repository!==repository||
     val.issueRef!=='https://github.com/'+repository+'/issues/'+issue||
     val.sourceSha!==sha||val.actorRole!=='builder'||
     val.dispatchId!==env.TALLY_V2_AGENT_CLAIM_DISPATCH_ID||
     val.operatorId!==env.TALLY_V2_AGENT_CLAIM_OPERATOR_ID||
     val.operatorKind!==env.TALLY_V2_AGENT_CLAIM_OPERATOR_KIND||
     val.writerId!==env.TALLY_V2_AGENT_CLAIM_WRITER_ID||
     !Number.isSafeInteger(val.commentId)||val.commentId<1||
     val.commentRef!==val.issueRef+'#issuecomment-'+val.commentId||
     val.generation!==1)
    return fail('AGENT_CLAIM_NOT_PUBLISHED_VERIFIED');
  if(env.TALLY_V2_AGENT_CLAIM_COMMENT_ID!==undefined&&
     env.TALLY_V2_AGENT_CLAIM_COMMENT_ID!==String(val.commentId))
    return fail('AGENT_CLAIM_COMMENT_PIN_MISMATCH');
  let writeSetDigest=null;
  if(env.TALLY_V2_WRITESET_FREEZE==='host-mounted'){
    let target;
    try{target=resolveBuilderBuildTarget(env);}catch{
      return fail('AGENT_WRITESET_BUILD_TARGET_INVALID');
    }
    const frozen=writeSetRow?.value;
    if(target.mode!=='SELECTED_BUILD'||
       target.repository!==repository||target.versionId!==settings.versionId||
       target.issueNumber!==issue||
       typeof frozen?.document!=='string'||
       createHash('sha256').update(frozen.document).digest('hex')!==
         frozen.writeSetSha256||
       !exactSelectedWriteSet(frozen,{
         repository,issueNumber:issue,sourceSha:sha,
         digest:frozen.writeSetSha256,
         candidateRelativePath:target.candidateRelativePath
       }))
      return fail('AGENT_WRITESET_NOT_FROZEN_VERIFIED');
    writeSetDigest=frozen.writeSetSha256;
    for(const name of ['TALLY_V2_AGENT_CLAIM_WRITESET_SHA256',
      'TALLY_LOCAL_WRITESET_SHA256']){
      if(env[name]!==undefined&&env[name]!==writeSetDigest)
        return fail('AGENT_WRITESET_PIN_MISMATCH');
    }
  }
  return Object.freeze({
    status:'HOST_RECEIPT_BOUND_PENDING_K',
    reason:'NATIVE_WITNESS_AND_K_VERIFICATION_STILL_REQUIRED',
    agentAuthorized:false,
    // This env is process-local and must never cross HTTP. It is NOT a
    // permit; the existing K/C consumer rechecks CAS and GitHub itself.
    env:{...env,TALLY_V2_AGENT_CLAIM_COMMENT_ID:String(val.commentId),
      ...(writeSetDigest?{
        TALLY_V2_AGENT_CLAIM_WRITESET_SHA256:writeSetDigest,
        TALLY_LOCAL_WRITESET_SHA256:writeSetDigest}:{})}
  });
}
