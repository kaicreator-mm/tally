import {useEffect,useState} from 'react';
import {hostAdapter} from '../../adapters/host.ts';
import {useShell} from './AppContext.tsx';
import {Select} from '../../components/ui/Select.tsx';
import {Badge} from '../../components/ui/Badge.tsx';
import {InboxPanel} from './InboxPanel.tsx';

/** Owner directive: project & version pickers live TOP-RIGHT only (no sidebar
 * duplicates); the bell opens the same human-action queue as the workbench. */
export function TopBar({inboxCount,onNavigate}:
  {inboxCount:number;onNavigate:(hash:string)=>void}){
  const shell=useShell();
  const [inboxOpen,setInboxOpen]=useState(false);
  const [setupUrl,setSetupUrl]=useState<string|null>(null);
  useEffect(()=>{
    let current=true;
    void hostAdapter.settings().then(settings=>{
      if(!current||settings.connected!==true||!settings.configurationUrl)return;
      try{
        const url=new URL(settings.configurationUrl);
        if(url.protocol==='http:'&&url.hostname==='127.0.0.1'&&
           url.pathname==='/setup'&&!url.username&&!url.password&&
           !url.search&&!url.hash)setSetupUrl(url.href);
      }catch{/* source-scoped setup URL is unavailable */}
    }).catch(()=>{});
    return ()=>{current=false;};
  },[]);
  // A UI-only selector must NOT pretend to switch the real selected Host.
  // Route every different project/version through the real Setup service,
  // which resolves the current GitHub HEAD and replaces the mounted Host.
  const switchHost=()=>setupUrl
    ?window.location.assign(setupUrl)
    :onNavigate('#/settings/git-github');
  const {t,catalog,projectId,versionId}=shell;
  const projects=catalog.projects??[];
  const versions=(projectId?catalog.versionsByProject?.[projectId]:undefined)??[];
  return <header className="topbar" role="banner">
    <div className="brand"><span aria-hidden="true">▦</span><em>Tally v2</em></div>
    <input className="topbar-search" type="search" role="searchbox"
      aria-label={t('top.search')} placeholder={t('top.search')}
      onKeyDown={event=>{if(event.key==='Enter')
        onNavigate(`#/workbench/inbox?focus=${encodeURIComponent(event.currentTarget.value)}`);}}/>
    <div className="spacer"/>
    {inboxCount>0?<Badge tone="warn" title={t('top.notifications')}>{inboxCount}</Badge>:null}
    <Select aria-label={t('top.project')} label={t('top.project')}
      value={projectId} placeholder={t('top.select-project')}
      options={projects.map(project=>({value:project.id,label:project.title}))}
      onChange={id=>{if(id!==projectId)switchHost();}}/>
    <Select aria-label={t('top.version')} label={t('top.version')}
      value={versionId}
      placeholder={projectId?t('top.select-version'):t('top.version-follows-project')}
      disabled={!projectId||versions.length===0}
      options={versions.map(version=>({value:version.id,label:version.label}))}
      onChange={id=>{if(id!==versionId)switchHost();}}/>
    {setupUrl?<a href={setupUrl} aria-label="添加或切换真实项目和版本"
      title="通过配置页验证 GitHub HEAD 并切换真实 Host">切换项目/版本</a>:null}
    <InboxPanel open={inboxOpen} onToggle={()=>setInboxOpen(open=>!open)}
      onNavigate={target=>{setInboxOpen(false);onNavigate(target);}}/>
  </header>;
}
