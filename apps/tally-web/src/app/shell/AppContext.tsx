import {createContext,useContext, useMemo,useState,type ReactNode} from 'react';
import type {Locale} from '../../i18n/index.ts';
import {createT,rememberLocale,formatDyn,type StorageLike} from '../../i18n/index.ts';

export interface ProjectSummary{id:string;title:string;repository?:string;}
export interface VersionSummary{id:string;projectId:string;label:string;ref?:string;}
/** Host-provided shell catalog. UI-E #232 wires the real adapter; until then
 * every entry is absent (NOT_CONNECTED), never faked. */
export interface ShellCatalog{
  projects:readonly ProjectSummary[]|null;
  versionsByProject:Readonly<Record<string,readonly VersionSummary[]>>|null;
}
export interface ShellState{
  locale:Locale;
  setLocale(locale:Locale):void;
  t:(key:Parameters<ReturnType<typeof createT>>[0],
    params?:Record<string,string|number>)=>string;
  td:(key:string,params?:Record<string,string|number>)=>string;
  projectId:string|null;
  versionId:string|null;
  selectProject(id:string|null):void;
  selectVersion(id:string|null):void;
  catalog:ShellCatalog;
}
const missing=(name:string):never=>{throw new Error(`SHELL_CONTEXT_MISSING:${name}`);};
export const ShellContext=createContext<ShellState>({
  locale:'zh-CN',setLocale:()=>missing('setLocale'),t:()=>missing('t'),
  td:()=>missing('td'),
  projectId:null,versionId:null,selectProject:()=>missing('selectProject'),
  selectVersion:()=>missing('selectVersion'),
  catalog:{projects:null,versionsByProject:null},
});
export const useShell=()=>useContext(ShellContext);
export function ShellProvider({children,storage,catalog={
  projects:null,versionsByProject:null}}:
  {children:ReactNode;storage?:StorageLike;catalog?:ShellCatalog}){
  const [locale,setLocaleState]=useState<Locale>('zh-CN');
  const [projectId,setProjectId]=useState<string|null>(null);
  const [versionId,setVersionId]=useState<string|null>(null);
  const value=useMemo<ShellState>(()=>{
    const t=createT(locale);
    return {
      locale,
      setLocale(next){setLocaleState(next);rememberLocale(storage,next);},
      t:(key,params)=>t(key,params),
      td:(key,params)=>formatDyn(locale,key,params),
      projectId,versionId,
      // Strict three-level rule: switching project always clears version.
      selectProject(id){setProjectId(id);setVersionId(null);},
      selectVersion(id){setVersionId(id);},
      catalog,
    };
  },[locale,projectId,versionId,catalog,storage]);
  return <ShellContext.Provider value={value}>{children}</ShellContext.Provider>;
}
