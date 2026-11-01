/** UI-E #232 — route composition: wires lane feature surfaces into the shell.
 * Files transferred to UI-E for integration (per #232 ownership transfer):
 * src/app/App.tsx, src/app/main.tsx, src/app/shell/AppContext.tsx.
 * Views without a server read model yet render honest NOT_CONNECTED states;
 * nothing fabricates data or authority.
 */
import {useState} from 'react';
import {useShell} from './shell/AppContext.tsx';
import {taskRoute} from './shell/scope.ts';
import {AppShell} from './shell/AppShell.tsx';
import {findItemByRoute} from './shell/nav-model.ts';
import {EmptyState,Skeleton} from '../components/ui/States.tsx';
import {useHost} from '../adapters/useHost.ts';
import type {Route} from './router.ts';
import type {TasksResponse,DagResponse,ReleaseResponse,
  SettingsResponse,MetadataResponse,WorkbenchResponse,
  VersionAgentsRead,VersionRequirementsRead,VersionQaRead,VersionMaintenanceRead,
  GovernanceRead} from '../adapters/host.ts';
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
import {sectionForSettingsRoute} from '../features/settings/model.ts';
import {WorkbenchView} from '../features/workbench/views.tsx';
import {GuidedHostRoute} from '../features/forms/host-route.tsx';
import {GovernanceView} from '../features/governance/views.tsx';
import type {GovernanceSnapshot,GovernanceKind}
  from '../features/governance/model.ts';
import {inboxCounts} from '../features/workbench/model.ts';
import type {WorkbenchSnapshot,WorkbenchPage} from '../features/workbench/model.ts';
import type {ReleaseReadiness} from '../features/release/model.ts';

export function App({route,onNavigate,inboxCount=0}:{route:Route;
  onNavigate:(hash:string)=>void;inboxCount?:number}){
  const {t,td,versionId,projectId}=useShell();
  const queue=useHost<WorkbenchResponse>('workbench',
    adapter=>adapter.workbench(),[]);
  const workbench=(queue.data?.connected===true?
    queue.data as WorkbenchSnapshot:null);
  const count=inboxCounts(workbench)?.pending??inboxCount;
  const item=findItemByRoute(route);
  const body=item
    ?routeBody(item.key,projectId,versionId,onNavigate,workbench,
      route.query.taskId??null)
    :<EmptyState title={t('state.empty')} hint={`${route.group}/${route.view}`}/>;
  return <AppShell route={route} onNavigate={onNavigate} inboxCount={count}>
    <div className="page">
      <div className="page-header">
        <h1>{item?td(`nav.${item.key}`):route.view}</h1>
        <span className="sub">Tally v2 · ALM</span>
      </div>
      {body}
    </div>
  </AppShell>;
}
function routeBody(key:string,projectId:string|null,versionId:string|null,
  onNavigate:(hash:string)=>void,workbench:WorkbenchSnapshot|null,
  taskId:string|null):React.ReactNode{
  switch(key){
    case 'inbox':case 'approvals':case 'blockers':case 'overview':
      return <WorkbenchView page={key as WorkbenchPage}
        snapshot={workbench} onNavigate={onNavigate}/>;
    case 'product':return <GuidedHostRoute projectId={projectId} versionId={versionId}/>;
    case 'tasks':return <TasksRoute projectId={projectId} versionId={versionId}
      taskId={taskId} onNavigate={onNavigate}/>;
    case 'agents':return <AgentsRoute projectId={projectId} versionId={versionId}/>;
    case 'l2-dag':return <DagRoute projectId={projectId} versionId={versionId}
      onNavigate={onNavigate}/>;
    case 'requirements':return <RequirementsRoute projectId={projectId} versionId={versionId}/>
    case 'qa':return <QaRoute projectId={projectId} versionId={versionId}/>
    case 'release':return <ReleaseRoute projectId={projectId} versionId={versionId}/>;
    case 'maintenance':return <MaintenanceRoute projectId={projectId} versionId={versionId}/>;
    case 'metadata':return <MetadataRoute projectId={projectId} versionId={versionId}
      onNavigate={onNavigate}/>;
    case 'workflow':return <GovernanceRoute kind="workflow" projectId={projectId}/>;
    case 'domain-packages':
      return <GovernanceRoute kind="domain-package" projectId={projectId}/>;
    case 'experience':
      return <GovernanceRoute kind="experience" projectId={projectId}/>;
    case 'audit':return <GovernanceRoute kind="audit" projectId={projectId}/>;
    case 'agent-providers':case 'tally-llm':case 'ci':case 'git-github':
    case 'host':case 'storage':case 'security':case 'deployment-env':
    case 'integrations':
      return <SettingsRoute routeKey={key}/>;
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
function TasksRoute({projectId,versionId,taskId,onNavigate}:{
  projectId:string|null;versionId:string|null;taskId:string|null;
  onNavigate:(hash:string)=>void}){
  const [filters,setFilters]=useState<TaskFilters>({...EMPTY_FILTERS});
  const tasks=useHost<TasksResponse>('tasks',
    adapter=>adapter.tasks(projectId??'',versionId??''),[projectId,versionId]);
  if(tasks.state==='loading')return <Skeleton lines={6}/>;
  if(tasks.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={tasks.reason??undefined}/>;
  const connected=tasks.data?.connected===true;
  const rows=(tasks.data?.tasks??[]) as TaskRecord[];
  return guard(versionId,<>
    <TasksView tasks={rows} filters={filters} setFilters={setFilters}
      onSelectTask={id=>projectId&&versionId&&
        onNavigate(taskRoute(projectId,versionId,id)!)}
      selectedTaskId={taskId}
      versionId={versionId} connected={connected}/>
    <TaskDrawer task={taskId&&connected?
        rows.find(row=>row.id===taskId&&row.versionId===versionId)??null:null}
      onClose={()=>projectId&&versionId&&
        onNavigate(taskRoute(projectId,versionId,null)!)}/>
  </>);
}
function AgentsRoute({projectId,versionId}:{projectId:string|null;
  versionId:string|null}){
  const result=useHost<VersionAgentsRead>('agents',
    adapter=>adapter.agents(projectId??'',versionId??''),
    [projectId,versionId]);
  if(!projectId||!versionId)return <EmptyState title="请选择项目与版本"/>;
  if(result.state==='loading')return <Skeleton lines={4}/>;
  const records=result.data?.attempts;
  return <AgentsPanel attempts={records??[]}
    connected={result.state==='ready'&&result.data?.connected===true&&
      result.data.versionId===versionId&&Array.isArray(records)}/>;
}
function RequirementsRoute({projectId,versionId}:{projectId:string|null;
  versionId:string|null}){
  const result=useHost<VersionRequirementsRead>('requirements',
    adapter=>adapter.requirements(projectId??'',versionId??''),
    [projectId,versionId]);
  if(!projectId||!versionId)return <EmptyState title="请选择项目与版本"/>;
  if(result.state==='loading')return <Skeleton lines={4}/>;
  return <RequirementsView nodes={result.data?.nodes??[]}
    connected={result.state==='ready'&&result.data?.connected===true&&
      result.data.versionId===versionId&&Array.isArray(result.data.nodes)}/>;
}
function QaRoute({projectId,versionId}:{projectId:string|null;
  versionId:string|null}){
  const result=useHost<VersionQaRead>('qa',
    adapter=>adapter.qa(projectId??'',versionId??''),
    [projectId,versionId]);
  if(!projectId||!versionId)return <EmptyState title="请选择项目与版本"/>;
  return result.state==='loading'?<Skeleton lines={4}/>:
    <QaView records={result.data?.records??[]}
      connected={result.state==='ready'&&result.data?.connected===true&&
        result.data.versionId===versionId&&Array.isArray(result.data.records)}/>;
}
function MaintenanceRoute({projectId,versionId}:{projectId:string|null;
  versionId:string|null}){
  const result=useHost<VersionMaintenanceRead>('maintenance',
    adapter=>adapter.maintenance(projectId??'',versionId??''),
    [projectId,versionId]);
  if(!projectId||!versionId)return <EmptyState title="请选择项目与版本"/>;
  if(result.state==='loading')return <Skeleton lines={4}/>;
  return <MaintenanceView defects={result.data?.defects??[]}
    learnings={result.data?.learnings??[]}
    connected={result.state==='ready'&&result.data?.connected===true&&
      result.data.versionId===versionId&&Array.isArray(result.data.defects)&&
      Array.isArray(result.data.learnings)}/>;
}
function GovernanceRoute({kind,projectId}:{kind:GovernanceKind;
  projectId:string|null}){
  const modelKey=kind==='domain-package'?'domainPackages':kind;
  const result=useHost<GovernanceRead>(modelKey,
    adapter=>adapter.governance(modelKey as
      'workflow'|'domainPackages'|'experience'|'audit',projectId??''),
    [kind,projectId]);
  if(result.state==='loading')return <Skeleton lines={4}/>;
  const snapshot=result.data?.connected===true&&
    Array.isArray(result.data.records)&&result.data.source
    ?result.data as GovernanceSnapshot:null;
  return <GovernanceView kind={kind} snapshot={snapshot}/>;
}
function DagRoute({projectId,versionId,onNavigate}:{
  projectId:string|null;versionId:string|null;
  onNavigate:(hash:string)=>void}){
  const dag=useHost<DagResponse>('dag',
    adapter=>adapter.dag(projectId??'',versionId??''),[projectId,versionId]);
  if(dag.state==='loading')return <Skeleton lines={6}/>;
  if(dag.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={dag.reason??undefined}/>;
  return guard(versionId,
    <DagView dag={(dag.data??null) as VersionDag|null} versionId={versionId}
      onOpenTask={id=>projectId&&versionId&&
        onNavigate(taskRoute(projectId,versionId,id)!)}/>);
}
function ReleaseRoute({projectId,versionId}:{projectId:string|null;versionId:string|null}){
  const release=useHost<ReleaseResponse>('release',
    adapter=>adapter.release(projectId??'',versionId??''),[projectId,versionId]);
  if(release.state==='loading')return <Skeleton lines={4}/>;
  const readiness=(release.data?.readiness??null) as ReleaseReadiness|null;
  return guard(versionId,
    <ReleaseView readiness={readiness}
      connected={release.data?.connected===true}/>);
}
function MetadataRoute({projectId,versionId,onNavigate}:{
  projectId:string|null;versionId:string|null;
  onNavigate:(hash:string)=>void}){
  const metadata=useHost<MetadataResponse>('metadata',
    adapter=>adapter.metadata(projectId??'',versionId??''),[projectId,versionId]);
  if(metadata.state==='loading')return <Skeleton lines={6}/>;
  const data=metadata.data;
  if(metadata.state==='not_connected'||data?.connected!==true)
    return guard(versionId,<EmptyState title="NOT_CONNECTED"
      hint="任务元数据需由选中的 Host 通过 ADS v5 来源验证后读取"/>);
  // Source-qualified data is preserved, not replaced with empty arrays.
  // The Host boundary validates the exact version and record schema.
  const version=data.versionId===versionId&&
    Array.isArray(data.attempts)&&Array.isArray(data.reports)&&
    Array.isArray(data.lineage)&&Array.isArray(data.learnedOutcomes)
    ?{versionId:data.versionId,
      attempts:data.attempts as import('../features/metadata/model.ts').AttemptMeta[],
      reports:data.reports as import('../features/metadata/model.ts').ReportMeta[],
      lineage:data.lineage as import('../features/metadata/model.ts').LineageMeta[],
      learnedOutcomes:data.learnedOutcomes as import('../features/metadata/model.ts').VersionMetadata['learnedOutcomes']}
    :null;
  return guard(versionId,
    version?<MetadataCenter version={version} allVersions={[]}
        expectedTaskIds={null} onOpenTask={id=>projectId&&versionId&&
          onNavigate(taskRoute(projectId,versionId,id)!)}/>
      :<EmptyState title="SOURCE_UNQUALIFIED" hint="元数据版本或字段校验失败"/>);
}
function SettingsRoute({routeKey}:{routeKey:string}){
  const settings=useHost<SettingsResponse>('settings',
    adapter=>adapter.settings(),[]);
  if(settings.state==='loading')return <Skeleton lines={6}/>;
  if(settings.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={settings.reason??undefined}/>;
  const selectedSection=sectionForSettingsRoute(routeKey);
  if(!selectedSection)return <EmptyState title="UNKNOWN_SETTINGS_ROUTE"/>;
  return <SettingsCenter
    snapshot={(settings.data?.snapshot??null) as SettingsSnapshot|null}
    selectedSection={selectedSection}
    connected={settings.data?.connected===true}/>;
}
