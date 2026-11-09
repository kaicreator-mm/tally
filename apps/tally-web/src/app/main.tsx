import { StrictMode, useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '../styles/tokens.css';
import '../styles/app.css';
import { App } from './App.tsx';
import { createRouter,buildHash,type Route } from './router.ts';
import { ShellProvider } from './shell/AppContext.tsx';
import type {SelectedScope} from './shell/scope.ts';
import { detectLocale, rememberLocale } from '../i18n/index.ts';
import { orNotConnected, hostAdapter } from '../adapters/host.ts';
import type { Catalog } from '../adapters/host.ts';

const router=createRouter({
  getHash:()=>window.location.hash||'#/workbench/inbox',
  setHash:hash=>{window.location.hash=hash;},
  addEventListener:(type,fn)=>window.addEventListener(type,fn),
  removeEventListener:(type,fn)=>window.removeEventListener(type,fn),
});
const locale=detectLocale({storage:window.localStorage});
rememberLocale(window.localStorage,locale);

/** Selector identity lives in hash so refresh/deep-link is consistent. */
function navigateToScope(next:SelectedScope){
  const current=router.get();
  router.navigate(buildHash({group:current.group,view:current.view,
    query:{...current.query,
      projectId:next.projectId??'',versionId:next.versionId??''}}));
}
function Root(){
  const [route,setRoute]=useState<Route>(()=>router.get());
  const [catalog,setCatalog]=useState<Catalog>({connected:false});
  useEffect(()=>{
    let mounted=true;
    void orNotConnected('catalog',()=>hostAdapter.catalog())
      .then(result=>{if(mounted)setCatalog(result as Catalog);});
    return ()=>{mounted=false;};
  },[]);
  useEffect(()=>{
    const off=router.subscribe(next=>{
      setRoute(next);
      document.getElementById('main')?.focus({preventScroll:true});
    });
    return off;
  },[]);
  return (
    <ShellProvider storage={window.localStorage}
      requestedScope={{projectId:route.query.projectId,
        versionId:route.query.versionId}}
      onScopeChange={navigateToScope}
      catalog={catalog.connected?{
        projects:catalog.projects??[],
        versionsByProject:catalog.versionsByProject??{},
      }:{projects:null,versionsByProject:null}}>
      <App route={route} onNavigate={hash=>router.navigate(hash)}/>
    </ShellProvider>
  );
}
createRoot(document.getElementById('root')!).render(
  <StrictMode><Root/></StrictMode>,
);
