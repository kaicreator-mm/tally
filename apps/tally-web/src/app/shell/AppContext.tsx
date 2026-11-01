import {createContext,useContext,useMemo,useState,type ReactNode} from 'react';
import type {Locale} from '../../i18n/index.ts';
import {createT,rememberLocale,formatDyn,type StorageLike} from '../../i18n/index.ts';
import {resolveScope,chooseProject,chooseVersion,
  type ScopeCatalog,type SelectedScope} from './scope.ts';

export interface ProjectSummary{id:string;title:string;repository?:string;}
export interface VersionSummary{id:string;projectId:string;label:string;ref?:string;}
export interface ShellCatalog extends ScopeCatalog{
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
  projects:null,versionsByProject:null},requestedScope={},onScopeChange}:
  {children:ReactNode;storage?:StorageLike;catalog?:ShellCatalog;
    requestedScope?:Partial<SelectedScope>;
    onScopeChange?:(scope:SelectedScope)=>void}){
  const [locale,setLocaleState]=useState<Locale>('zh-CN');
  // URL query is resolved against the Host catalog on every render.
  // There is no unsynchronized duplicate local selection state.
  const {projectId,versionId}=resolveScope(catalog,requestedScope);
  const value=useMemo<ShellState>(()=>{
    const t=createT(locale);
    return {
      locale,
      setLocale(next){setLocaleState(next);rememberLocale(storage,next);},
      t:(key,params)=>t(key,params),
      td:(key,params)=>formatDyn(locale,key,params),
      projectId,versionId,
      selectProject(id){onScopeChange?.(chooseProject(catalog,id));},
      selectVersion(id){onScopeChange?.(chooseVersion(catalog,
        {projectId,versionId},id));},
      catalog,
    };
  },[locale,projectId,versionId,catalog,storage,onScopeChange]);
  return <ShellContext.Provider value={value}>{children}</ShellContext.Provider>;
}
