import {useState} from 'react';
import {useShell} from './AppContext.tsx';
import {NAV_GROUPS,navAvailability} from './nav-model.ts';
import type {Route} from '../router.ts';

export function Sidebar({route,onNavigate,onRequestClose}:
  {route:Route;onNavigate:(hash:string)=>void;onRequestClose?:()=>void}){
  const shell=useShell();
  const {t,td,projectId,versionId}=shell;
  const activeGroup=route.group;
  const [openGroups,setOpenGroups]=useState<Record<string,boolean>>(
    ()=>Object.fromEntries(NAV_GROUPS.map(group=>[group.key,group.key===activeGroup])));
  return <nav className="sidebar" aria-label={t('app.title')}
    onClick={event=>{if(event.target===event.currentTarget)onRequestClose?.();}}>
    {NAV_GROUPS.map(group=>{
      const open=openGroups[group.key]??false;
      return <div key={group.key} className={`sidebar-group${open?' open':''}`}>
        <button className="sidebar-group-head" aria-expanded={open}
          onClick={()=>setOpenGroups(state=>({...state,[group.key]:!open}))}>
          <span className="chev" aria-hidden="true">▸</span>
          <span>{td(`nav.group.${group.key}`)}</span>
        </button>
        {open?group.items.map(item=>{
          const availability=navAvailability(item,{projectId,versionId});
          const active=route.group===group.key&&route.view===item.route.split('/')[2];
          return <button key={item.key} className={`sidebar-item${active?' active':''}`}
            disabled={!availability.available}
            title={!availability.available
              ?availability.missing==='project'?t('state.need-project'):t('state.need-version')
              :td(`nav.${item.key}`)}
            onClick={()=>onNavigate(`#${item.route}${item.scope!=='global'
              ?`?projectId=${encodeURIComponent(projectId??'')}${item.scope==='version'
                ?`&versionId=${encodeURIComponent(versionId??'')}`:''}`
              :''}`)}>
            <span>{td(`nav.${item.key}`)}</span>
            {!availability.available?<span className="lock" aria-hidden="true">🔒</span>:null}
          </button>;
        }):null}
      </div>;
    })}
  </nav>;
}
