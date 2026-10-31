/** J07c boundary: UI cannot promote advisory RQ, nor take source authority. */
const SHA=/^[a-f0-9]{40}$/;
const UNKNOWN=Object.freeze({status:'UNKNOWN',reason:'NATIVE_RELEASE_SOURCE_UNAVAILABLE',
  canMerge:false,canTag:false,canPublish:false,releasePass:false});
const denied=(reason)=>Object.freeze({status:'DENIED',reason,authorized:false,
  releasePass:false,taskPass:false,mutationPerformed:false});
/** readReleaseSnapshot must read selected C/Host, never form fields.
 * dispatchThroughKernel is NOT a GitHub writer; only the selected K facade.
 * This source prototype refuses UI actions until the independent server
 * admission is wired. It cannot authorize or publish any Release by itself. */
export function createReleaseController({sourceSha,readReleaseSnapshot}={}){
  if(!SHA.test(sourceSha??''))throw new TypeError('RELEASE_SOURCE_SHA_REQUIRED');
  let last=UNKNOWN;
  return Object.freeze({
    async snapshot(){
      if(typeof readReleaseSnapshot!=='function')return UNKNOWN;
      try{
        const value=await readReleaseSnapshot({sourceSha});
        if(value?.sourceSha!==sourceSha||!SHA.test(value?.targetSha??''))
          return UNKNOWN;
        const readiness=value.readiness;
        if(!readiness||!['BLOCKED','UNKNOWN'].includes(readiness.status)||
          readiness.releasePass===true)return UNKNOWN;
        last=Object.freeze({...value,status:'OBSERVED',
          canMerge:false,canTag:false,canPublish:false,releasePass:false});
        return last;
      }catch{return UNKNOWN;}
    },
    async requestAction({kind,sourceSha:requestedSha}={}){
      if(requestedSha!==sourceSha)return denied('STALE_VERSION_SHA');
      if(!['merge','tag','publish'].includes(kind))return denied('UNKNOWN_RELEASE_OPERATION');
      if(last.status!=='OBSERVED')return denied('NATIVE_RELEASE_EVIDENCE_NOT_READ');
      return denied('INDEPENDENT_HOST_ADMISSION_NOT_CONNECTED');
    }
  });
}
