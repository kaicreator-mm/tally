/** Host-only selected single-file FSW admission. No browser write-set input. */
import {createHash} from 'node:crypto';
import {resolveBuilderBuildTarget,exactSelectedWriteSet}
  from '../../experiments/v2/selected-business-build/builder-target.mjs';
const sha256=data=>createHash('sha256').update(data,'utf8').digest('hex');
const deny=reason=>Object.freeze({status:'BLOCKED',reason,
  authorized:false,admitted:false,taskPass:false,releasePass:false});
export function selectedWriteSetIdentity({settings,env}={}){
  if(env?.TALLY_V2_WRITESET_FREEZE!=='host-mounted'||!settings)
    throw new TypeError('SELECTED_WRITESET_HOST_NOT_MOUNTED');
  const target=resolveBuilderBuildTarget(env);
  if(target.mode!=='SELECTED_BUILD'||
     target.repository!==settings.repository||
     target.versionId!==settings.versionId||
     env.TALLY_V2_CLAIM_REPOSITORY!==settings.repository||
     env.TALLY_V2_CLAIM_ISSUE_NUMBER!==String(target.issueNumber)||
     env.TALLY_V2_CLAIM_SOURCE_SHA!==settings.expectedSha||
     env.TALLY_V2_CLAIM_VERSION_REF!==settings.versionId||
     !/^[a-f0-9]{40}$/.test(settings.expectedSha??'')||
     typeof env.TALLY_V2_CLAIM_REGISTRY_PATH!=='string'||
     env.TALLY_V2_CLAIM_REGISTRY_PATH.length<3||
     (env.TALLY_V2_AGENT_DISPATCH==='host-mounted'&&
      env.TALLY_V2_AGENT_CLAIM_REGISTRY_PATH!==
        env.TALLY_V2_CLAIM_REGISTRY_PATH))
    throw new TypeError('SELECTED_WRITESET_HOST_SCOPE_MISMATCH');
  const document=JSON.stringify([target.candidateRelativePath]);
  return Object.freeze({...target,sourceSha:settings.expectedSha,document,
    digest:sha256(document),
    registryPath:env.TALLY_V2_CLAIM_REGISTRY_PATH,
    key:'tally:v2:task-writeset:'+target.issueNumber});
}
export function validateSelectedWriteSetRow(row,identity){
  return Number.isSafeInteger(row?.revision)&&row.revision>=0&&
    exactSelectedWriteSet(row.value,{
      repository:identity.repository,issueNumber:identity.issueNumber,
      sourceSha:identity.sourceSha,digest:identity.digest,
      candidateRelativePath:identity.candidateRelativePath})&&
    row.value.document===identity.document;
}
export function createProjectWriteSetHost({settings,env,openStore}={}){
  if(env?.TALLY_V2_WRITESET_FREEZE!=='host-mounted')return null;
  const identity=selectedWriteSetIdentity({settings,env});
  if(typeof openStore!=='function')
    throw new TypeError('SELECTED_WRITESET_CAS_PORT_REQUIRED');
  const value=Object.freeze({
    schema:'tally/selected-task-writeset/v1',
    phase:'FROZEN_DAG_WRITESET',repository:identity.repository,
    issueNumber:identity.issueNumber,versionId:identity.versionId,
    sourceSha:identity.sourceSha,writeSetSha256:identity.digest,
    document:identity.document});
  return Object.freeze({
    repository:identity.repository,versionId:identity.versionId,
    issueNumber:identity.issueNumber,sourceSha:identity.sourceSha,
    candidateRelativePath:identity.candidateRelativePath,
    async freeze({issueNumber,sourceSha}={}){
      if(issueNumber!==identity.issueNumber||sourceSha!==identity.sourceSha)
        return deny('SELECTED_WRITESET_TASK_SCOPE_MISMATCH');
      let store;
      try{
        store=await openStore({path:identity.registryPath,busyTimeoutMs:1000});
        const before=await store.get(identity.key);
        if(before)return validateSelectedWriteSetRow(before,identity)?
          Object.freeze({status:'ALREADY_FROZEN_VERIFIED',
            writeSetSha256:identity.digest,candidateRelativePath:identity.candidateRelativePath,
            authorized:false,admitted:false,taskPass:false,releasePass:false}):
          deny('SELECTED_WRITESET_EXISTING_ROW_MISMATCH');
        await store.put(identity.key,value,{mustBeAbsent:true});
        const after=await store.get(identity.key);
        if(!validateSelectedWriteSetRow(after,identity))
          return deny('SELECTED_WRITESET_READBACK_UNKNOWN');
        return Object.freeze({status:'FROZEN_VERIFIED',
          writeSetSha256:identity.digest,candidateRelativePath:identity.candidateRelativePath,
          authorized:false,admitted:false,taskPass:false,releasePass:false});
      }catch{return deny('SELECTED_WRITESET_CAS_UNKNOWN');}
      finally{if(store)await store.close().catch(()=>{});}
    }
  });
}
