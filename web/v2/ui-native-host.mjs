/** Selected native GitHub C → UI Host read-model composition.
 * Native C proves GitHub Task + dependency facts. Host binds selected
 * project/version; unimplemented lifecycle models remain NOT_CONNECTED.
 */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const deny=reason=>({connected:false,reason});
export function createUiV2ReadModelHost({repository,versionId,nativeC}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     typeof nativeC?.catalog!=='function'||
     typeof nativeC?.version!=='function')
    throw new TypeError('SELECTED_NATIVE_C_REQUIRED');
  const scoped=scope=>scope?.projectId===repository&&
    scope?.versionId===versionId;
  const readVersion=async scope=>{
    if(!scoped(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
    try{
      const result=await nativeC.version();
      if(result?.connected!==true||result.source?.repository!==repository||
         result.versionId!==versionId||!Array.isArray(result.tasks)||
         result.dag?.versionId!==versionId)
        return deny(result?.reason??'NATIVE_SOURCE_UNQUALIFIED');
      return result;
    }catch{return deny('NATIVE_C_READ_FAILED');}
  };
  return Object.freeze({
    async catalog(){
      try{
        const result=await nativeC.catalog();
        if(result?.connected!==true||
           result.source?.repository!==repository||
           !result.versionsByProject?.[repository]?.some(v=>
             v.id===versionId&&v.projectId===repository))
          return deny(result?.reason??'CATALOG_SOURCE_UNQUALIFIED');
        return result;
      }catch{return deny('NATIVE_C_READ_FAILED');}
    },
    async tasks(scope){
      const observed=await readVersion(scope);
      return observed.connected?{connected:true,source:observed.source,
        versionId,rosterCoverage:observed.rosterCoverage??'UNVERIFIED',
        tasks:observed.tasks}:observed;
    },
    async dag(scope){
      const observed=await readVersion(scope);
      return observed.connected?observed.dag:observed;
    },
  });
}
