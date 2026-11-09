import {useState} from 'react';
import {useShell} from './AppContext.tsx';
import {Select} from '../../components/ui/Select.tsx';
import {Badge} from '../../components/ui/Badge.tsx';
import {InboxPanel} from './InboxPanel.tsx';

/** Owner directive: project & version pickers live TOP-RIGHT only (no sidebar
 * duplicates); the bell opens the same human-action queue as the workbench. */
export function TopBar({inboxCount,onNavigate}:
  {inboxCount:number|null;onNavigate:(hash:string)=>void}){
  const shell=useShell();
  const [inboxOpen,setInboxOpen]=useState(false);
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
    {inboxCount===null?<Badge tone="neutral" title="PARTIAL_INBOX_COVERAGE">?</Badge>:
      inboxCount>0?<Badge tone="warn" title={t('top.notifications')}>{inboxCount}</Badge>:null}
    <Select aria-label={t('top.project')} label={t('top.project')}
      value={projectId} placeholder={t('top.select-project')}
      options={projects.map(project=>({value:project.id,label:project.title}))}
      onChange={id=>shell.selectProject(id||null)}/>
    <Select aria-label={t('top.version')} label={t('top.version')}
      value={versionId}
      placeholder={projectId?t('top.select-version'):t('top.version-follows-project')}
      disabled={!projectId||versions.length===0}
      options={versions.map(version=>({value:version.id,label:version.label}))}
      onChange={id=>shell.selectVersion(id||null)}/>
    <InboxPanel open={inboxOpen} onToggle={()=>setInboxOpen(open=>!open)}
      onNavigate={target=>{setInboxOpen(false);onNavigate(target);}}/>
  </header>;
}
