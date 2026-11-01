import {useState} from 'react';
import {TopBar} from './TopBar.tsx';
import {Sidebar} from './Sidebar.tsx';
import type {Route} from '../router.ts';

export function AppShell({route,onNavigate,inboxCount,children}:
  {route:Route;onNavigate:(hash:string)=>void;inboxCount:number|null;children:React.ReactNode}){
  const [navOpen,setNavOpen]=useState(false);
  return <div className="shell">
    <TopBar inboxCount={inboxCount} onNavigate={onNavigate}/>
    <div className={`shell-body${navOpen?' nav-open':''}`}>
      <Sidebar route={route} onNavigate={hash=>{onNavigate(hash);setNavOpen(false);}}
        onRequestClose={()=>setNavOpen(false)}/>
      <main className="shell-main" id="main" tabIndex={-1}>{children}</main>
    </div>
    <button type="button" className="nav-toggle" aria-label="menu"
      aria-expanded={navOpen} onClick={()=>setNavOpen(open=>!open)}>☰</button>
  </div>;
}
