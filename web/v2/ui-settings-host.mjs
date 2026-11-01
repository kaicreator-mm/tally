/**
 * Source-qualified settings inventory for the read-only Tally v2 Host.
 * Only the selected GitHub repository/ref is actually observed. Unselected
 * Agent/LLM/CI/OAuth/Storage/Deployment settings remain NOT_CONNECTED.
 * No token, env value or Host credential reference is ever serialized.
 */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const deny=reason=>({connected:false,reason});
const unavailable=Object.freeze({
  agentProviders:'NOT_CONNECTED',tallyLlm:'NOT_CONNECTED',
  ci:'NOT_CONNECTED',gitGithub:'CURRENT_OBSERVED',
  deployment:'NOT_CONNECTED',host:'NOT_CONNECTED',
  storage:'NOT_CONNECTED',security:'NOT_CONNECTED',
  integrations:'NOT_CONNECTED',defaults:'NOT_CONNECTED'
});
export function createUiSettingsHost({repository,versionId,expectedSha,nativeC}={}){
  if(!REPO.test(repository??'')||versionId!=='version/v2'||
     !SHA.test(expectedSha??'')||typeof nativeC?.catalog!=='function')
    throw new TypeError('TRUSTED_SETTINGS_SOURCE_REQUIRED');
  const valid=c=>c?.connected===true&&c.source?.repository===repository&&
    c.source?.sourceSha===expectedSha&&c.source?.currentness==='CURRENT'&&
    c.source?.readback==='VERIFIED'&&
    Array.isArray(c.projects)&&
    c.projects.some(p=>p?.id===repository)&&
    c.versionsByProject?.[repository]?.some(v=>v?.id===versionId&&
      v?.projectId===repository);
  async function settings(){
    try{
      const before=await nativeC.catalog();
      if(!valid(before))return deny('SETTINGS_SOURCE_NOT_QUALIFIED');
      const after=await nativeC.catalog();
      if(!valid(after))return deny('SETTINGS_SOURCE_CHANGED_DURING_READ');
      // Placeholder shapes are needed by the existing UI contract; the
      // sectionCoverage field MUST gate their rendering. Empty arrays are
      // NOT assertions of zero configured providers/deployments.
      return {connected:true,source:{repository,sourceSha:expectedSha,
        currentness:'CURRENT',readback:'VERIFIED'},coverage:'PARTIAL',
        sectionCoverage:{...unavailable},
        snapshot:{
          agentProviders:[],
          tallyLlm:{enabled:false,provider:null,model:null,endpoint:null,
            promptTemplateVersion:null,privacyConsentPreviewShown:false,
            tokenBudget:null,credential:null},
          ci:{woodpeckerUrl:null,repoId:null,profiles:[],
            webhookState:'UNKNOWN',lastReadbackAt:null},
          git:{repositories:[repository],defaultBase:'version/v2',
            adsPin:null,oauthScopes:[],requiredScopeGaps:[]},
          deploymentEnvs:[],
          defaults:{locale:'zh-CN',defaultView:'list',
            cascade:{workspace:'UNVERIFIED',project:null,
              version:null,task:null}}
        }
      };
    }catch{return deny('SETTINGS_NATIVE_SOURCE_READ_FAILED');}
  }
  return Object.freeze({settings});
}
