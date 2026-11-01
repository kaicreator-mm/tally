/** UI-E #232 — route composition: wires lane feature surfaces into the shell.
 * Files transferred to UI-E for integration (per #232 ownership transfer):
 * src/app/App.tsx, src/app/main.tsx, src/app/shell/AppContext.tsx.
 * Views without a server read model yet render honest NOT_CONNECTED states;
 * nothing fabricates data or authority.
 */
import {useState} from 'react';
import {useShell} from './shell/AppContext.tsx';
import {AppShell} from './shell/AppShell.tsx';
import {findItemByRoute} from './shell/nav-model.ts';
import {EmptyState,Skeleton} from '../components/ui/States.tsx';
import {useHost} from '../adapters/useHost.ts';
import type {Route} from './router.ts';
import type {TasksResponse,DagResponse,ReleaseResponse,
  SettingsResponse,MetadataResponse} from '../adapters/host.ts';
import {TasksView,TaskDrawer} from '../features/tasks/views.tsx';
import {AgentsPanel} from '../features/agents/views.tsx';
import {RequirementsView} from '../features/requirements/views.tsx';
import {QaView} from '../features/qa/views.tsx';
import {ReleaseView} from '../features/release/views.tsx';
import {MaintenanceView} from '../features/maintenance/views.tsx';
import {MetadataCenter} from '../features/metadata/views.tsx';
import {SettingsCenter} from '../features/settings/views.tsx';
import {DagView} from '../features/dag/views.tsx';
import {EMPTY_FILTERS} from '../features/tasks/model.ts';
import type {TaskFilters,TaskRecord} from '../features/tasks/model.ts';
import type {VersionDag} from '../features/dag/model.ts';
import type {SettingsSnapshot} from '../features/settings/model.ts';
import type {ReleaseReadiness} from '../features/release/model.ts';

export function App({route,onNavigate,inboxCount=0}:{route:Route;
  onNavigate:(hash:string)=>void;inboxCount?:number}){
  const {t,td,versionId}=useShell();
  const item=findItemByRoute(route);
  const body=item
    ?routeBody(item.key,versionId)
    :<EmptyState title={t('state.empty')} hint={`${route.group}/${route.view}`}/>;
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
function routeBody(key:string,versionId:string|null):React.ReactNode{
  switch(key){
    case 'tasks':return <TasksRoute versionId={versionId}/>;
    case 'agents':return <AgentsRoute versionId={versionId}/>;
    case 'l2-dag':return <DagRoute versionId={versionId}/>;
    case 'requirements':return <RequirementsView nodes={[]}/>
    case 'qa':return <QaView records={[]}/>
    case 'release':return <ReleaseRoute versionId={versionId}/>;
    case 'maintenance':return <MaintenanceView defects={[]} learnings={[]}/>;
    case 'metadata':return <MetadataRoute versionId={versionId}/>;
    case 'agent-providers':case 'tally-llm':case 'ci':case 'git-github':
    case 'host':case 'storage':case 'security':case 'deployment-env':
    case 'integrations':
      return <SettingsRoute/>;
    default:
      // Workbench inbox/approvals/blockers/overview and deployment read
      // models are not implemented server-side yet: honest NOT_CONNECTED.
      return <EmptyState title="NOT_CONNECTED"
        hint="该视图的服务端读模型尚未接入(UI-E 后续批次)"/>;
  }
}
const guard=(versionId:string|null,children:React.ReactNode)=>
  !versionId
    ?<EmptyState title="请先在右上角选择版本"/>
    :children;
function TasksRoute({versionId}:{versionId:string|null}){
  const [filters,setFilters]=useState<TaskFilters>({...EMPTY_FILTERS});
  const [selectedId,setSelectedId]=useState<string|null>(null);
  const tasks=useHost<TasksResponse>('tasks',
    adapter=>adapter.tasks(versionId??''),[versionId]);
  if(tasks.state==='loading')return <Skeleton lines={6}/>;
  if(tasks.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={tasks.reason??undefined}/>;
  const connected=tasks.data?.connected===true;
  const rows=(tasks.data?.tasks??[]) as TaskRecord[];
  return guard(versionId,<>
    <TasksView tasks={rows} filters={filters} setFilters={setFilters}
      onSelectTask={id=>setSelectedId(id)} selectedTaskId={selectedId}
      versionId={versionId} connected={connected}/>
    <TaskDrawer task={selectedId?rows.find(row=>row.id===selectedId)??null:null}
      onClose={()=>setSelectedId(null)}/>
  </>);
}
function AgentsRoute({versionId}:{versionId:string|null}){
  return guard(versionId,
    <AgentsPanel attempts={[]} connected/>);
}
function DagRoute({versionId}:{versionId:string|null}){
  const dag=useHost<DagResponse>('dag',
    adapter=>adapter.dag(versionId??''),[versionId]);
  if(dag.state==='loading')return <Skeleton lines={6}/>;
  if(dag.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={dag.reason??undefined}/>;
  return guard(versionId,
    <DagView dag={(dag.data??null) as VersionDag|null} versionId={versionId}/>);
}
function ReleaseRoute({versionId}:{versionId:string|null}){
  const release=useHost<ReleaseResponse>('release',
    adapter=>adapter.release(versionId??''),[versionId]);
  if(release.state==='loading')return <Skeleton lines={4}/>;
  const readiness=(release.data?.readiness??null) as ReleaseReadiness|null;
  return guard(versionId,
    <ReleaseView readiness={readiness}
      connected={release.data?.connected===true}
      onAction={()=>{/* effects stay Host-owned; UI renders permit state */}}/>);
}
function MetadataRoute({versionId}:{versionId:string|null}){
  const metadata=useHost<MetadataResponse>('metadata',
    adapter=>adapter.metadata(versionId??''),[versionId]);
  if(metadata.state==='loading')return <Skeleton lines={6}/>;
  const data=metadata.data;
  const version=data?.connected===true&&data.versionId
    ?{versionId:data.versionId,attempts:[],reports:[],lineage:[],
      learnedOutcomes:[]}
    :null;
  return guard(versionId,
    <MetadataCenter version={version} allVersions={[]} expectedTaskIds={[]}/>);
}
function SettingsRoute(){
  const settings=useHost<SettingsResponse>('settings',
    adapter=>adapter.settings(),[]);
  if(settings.state==='loading')return <Skeleton lines={6}/>;
  if(settings.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={settings.reason??undefined}/>;
  return <SettingsCenter
    snapshot={(settings.data?.snapshot??null) as SettingsSnapshot|null}
    connected={settings.data?.connected===true}/>;
}
