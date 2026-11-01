/** UI-E #232 — data hook for shell routes over the Host adapter with honest
 * NOT_CONNECTED degradation and per-version reload semantics. */
import {useCallback,useEffect,useState} from 'react';
import {hostAdapter,orNotConnected} from './host.ts';

export interface HostState<T>{
  state:'loading'|'ready'|'not_connected';
  data:T|null;
  reason:string|null;
}
export function useHost<T>(model:string,
  load:(adapter:typeof hostAdapter)=>Promise<T>,
  deps:readonly unknown[]):HostState<T>{
  const [snapshot,setSnapshot]=useState<HostState<T>>(
    {state:'loading',data:null,reason:null});
  const refresh=useCallback(async()=>{
    setSnapshot(previous=>({...previous,state:'loading'}));
    const result=await orNotConnected(model,()=>load(hostAdapter));
    if(result&&(result as {connected?:boolean}).connected===false){
      const degraded=result as {reason?:string;transportError?:string};
      setSnapshot({state:'not_connected',data:null,
        reason:degraded.transportError??degraded.reason??'NOT_CONNECTED'});
      return;}
    setSnapshot({state:'ready',data:result as T,reason:null});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  },deps);
  useEffect(()=>{void refresh();},[refresh]);
  return {...snapshot,refresh} as HostState<T>&{refresh:()=>Promise<void>};
}
export {hostAdapter};
