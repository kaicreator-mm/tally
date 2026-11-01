import {useShell} from './shell/AppContext.tsx';
import {AppShell} from './shell/AppShell.tsx';
import {findItemByRoute} from './shell/nav-model.ts';
import {EmptyState} from '../components/ui/States.tsx';
import type {Route} from './router.ts';

/** UI-A renders honest placeholders for lane-owned views until UI-E #232
 * integrates the real feature surfaces; the shell itself never fabricates
 * feature content or status. */
export function App({route,onNavigate,inboxCount=0}:{route:Route;
  onNavigate:(hash:string)=>void;inboxCount?:number}){
  const {t,td,projectId,versionId}=useShell();
  const item=findItemByRoute(route);
  const body=!item?
      <EmptyState title={t('state.empty')}
        hint={`${route.group}/${route.view}`}/>
    :item.scope==='project'&&!projectId?
      <EmptyState title={t('state.need-project')} hint={td(`nav.${item.key}`)}/>
    :item.scope==='version'&&(!projectId||!versionId)?
      <EmptyState title={projectId?t('state.need-version'):t('state.need-project')}
        hint={td(`nav.${item.key}`)}/>
    :<EmptyState title={td(`nav.${item.key}`)}
        hint={t('state.not-integrated',{lane:item.lane})}/>;
  return <AppShell route={route} onNavigate={onNavigate} inboxCount={inboxCount}>
    <div className="page">
      <div className="page-header">
        <h1>{item?td(`nav.${item.key}`):route.view}</h1>
        <span className="sub">Tally v2 · ALM</span>
      </div>
      {body}
    </div>
  </AppShell>;
}
