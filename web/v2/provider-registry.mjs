/** R105 optional Provider registry (v2.1 candidate; #64).
 * Holds ONLY optional providers — the default codex path stays outside this
 * registry and is routed through the existing v2 execution chain unchanged.
 * Registry responsibilities are deliberately minimal: structural interface
 * conformance at registration, trusted-config enablement, cached capability
 * probes, and descriptors (AVAILABLE / DISABLED / UNKNOWN) for operators.
 * It is not a scheduler, does not spawn processes and never executes
 * attempts by itself. */
import {assertProviderInterface,resolveProviderSelection} from './provider-contract.mjs';

export const PROVIDER_STATUS=Object.freeze(['AVAILABLE','DISABLED','UNKNOWN']);

export function createProviderRegistry({providers=[],enabled={},probes={},now}={}){
  if(providers!==undefined&&!Array.isArray(providers))
    throw new TypeError('PROVIDERS_NOT_ARRAY');
  if(enabled!==undefined&&typeof enabled!=='object')
    throw new TypeError('ENABLED_NOT_OBJECT');
  const registered=new Map();
  const probeResults=new Map();
  const enabledRows=structuredClone(enabled??{});
  for(const [providerId,probe] of Object.entries(probes??{}))
    probeResults.set(providerId,probe);
  function register(provider){
    const check=assertProviderInterface(provider);
    if(!check.conformant)throw Object.assign(
      new Error('PROVIDER_INTERFACE_NONCONFORMANT'),{code:'PROVIDER_INTERFACE_NONCONFORMANT',
        missing:check.missing});
    if(registered.has(provider.providerId))throw Object.assign(
      new Error('PROVIDER_ALREADY_REGISTERED'),{code:'PROVIDER_ALREADY_REGISTERED',
        providerId:provider.providerId});
    registered.set(provider.providerId,Object.freeze(provider));
    return {providerId:provider.providerId,registered:true};
  }
  for(const provider of providers)register(provider);
  function get(providerId){return registered.get(providerId)??null;}
  function setEnabled(providerId,value){
    if(!registered.has(providerId))return {updated:false,reason:'PROVIDER_NOT_REGISTERED'};
    enabledRows[providerId]=value===true;
    return {providerId,updated:true,enabled:value===true};
  }
  /** Refresh capability probes for enabled optional providers. A failed or
   * stalled probe is recorded honestly; it never fabricates availability. */
  async function refreshProbes(config){
    const refreshed={};
    for(const [providerId,provider] of registered){
      if(enabledRows[providerId]!==true){refreshed[providerId]=probeResults.get(providerId)??null;continue;}
      try{
        const probe=await provider.probe(config??{});
        probeResults.set(providerId,probe);
        refreshed[providerId]=probe;
      }catch(error){
        const probe={available:false,provider:providerId,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
          detail:'PROBE_THREW',error:String(error?.message??error).slice(0,200)};
        probeResults.set(providerId,probe);refreshed[providerId]=probe;
      }
    }
    return refreshed;
  }
  function describe(providerId){
    const provider=get(providerId);
    if(!provider)return {providerId,registered:false,status:'UNKNOWN',enabled:false};
    const enabled=enabledRows[providerId]===true;
    const probe=probeResults.get(providerId)??null;
    const status=!enabled?'DISABLED':probe?.available===true?'AVAILABLE':'UNKNOWN';
    return {providerId,registered:true,enabled,status,probe};
  }
  function list(){return [...registered.keys()].map(providerId=>describe(providerId));}
  /** Admission-time selection over the registered optional providers. */
  function select({requested}){
    return resolveProviderSelection({requested,enabled:enabledRows,
      probe:requested?probeResults.get(requested)??null:null});
  }
  return Object.freeze({register,get,setEnabled,refreshProbes,describe,list,select});
}
