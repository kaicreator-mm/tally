/** Reviewer #244: project/version async result must never leak across scopes. */
import {useCallback,useEffect,useRef,useState} from 'react';
import {hostAdapter,orNotConnected} from './host.ts';
import {createLatestRequestGate,requestIdentity} from './latest.ts';

export interface HostState<T>{
  state:'loading'|'ready'|'not_connected';
  data:T|null;
  reason:string|null;
}
export function useHost<T>(model:string,
  load:(adapter:typeof hostAdapter)=>Promise<T>,
  deps:readonly unknown[]):HostState<T>&{refresh:()=>Promise<void>}{
  const activeIdentity=requestIdentity(model,deps);
  const [snapshot,setSnapshot]=useState<HostState<T>&{identity:string}>(
    {state:'loading',data:null,reason:null,identity:activeIdentity});
  const gate=useRef(createLatestRequestGate());
  const refresh=useCallback(async()=>{
    const lease=gate.current.begin();
    setSnapshot({state:'loading',data:null,reason:null,identity:activeIdentity});
    const result=await orNotConnected(model,()=>load(hostAdapter));
    if(!lease.isCurrent())return;
    if(result&&(result as {connected?:boolean}).connected===false){
      const degraded=result as {reason?:string;transportError?:string};
      setSnapshot({state:'not_connected',data:null,identity:activeIdentity,
        reason:degraded.transportError??degraded.reason??'NOT_CONNECTED'});
      return;
    }
    setSnapshot({state:'ready',data:result as T,reason:null,identity:activeIdentity});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  },[model,...deps]);
  useEffect(()=>{
    void refresh();
    return ()=>gate.current.invalidate();
  },[refresh]);
  // Route props can change before effects/old requests settle; never show
  // prior-project/previous-version data even for a transient render.
  const visible:HostState<T>=snapshot.identity===activeIdentity?snapshot:
    {state:'loading',data:null,reason:null};
  return {...visible,refresh};
}
export {hostAdapter};
