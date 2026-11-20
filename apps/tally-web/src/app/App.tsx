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
import {submitReleaseAction} from '../adapters/host.ts';
import type {Route} from './router.ts';
import type {TasksResponse,DagResponse,DualEpochResponse,ReleaseResponse,
  SettingsResponse,MetadataResponse,AgentAttemptsResponse,
  ReleaseActionResult} from '../adapters/host.ts';
import {TasksView,TaskDrawer} from '../features/tasks/views.tsx';
import {AgentsPanel} from '../features/agents/views.tsx';
import {RequirementsView} from '../features/requirements/views.tsx';
import {QaView} from '../features/qa/views.tsx';
import {ReleaseView} from '../features/release/views.tsx';
import {MaintenanceView} from '../features/maintenance/views.tsx';
import {MetadataCenter} from '../features/metadata/views.tsx';
import {SettingsCenter} from '../features/settings/views.tsx';
import {DagView,DualEpochPanel} from '../features/dag/views.tsx';
import {EMPTY_FILTERS} from '../features/tasks/model.ts';
import type {TaskFilters,TaskRecord} from '../features/tasks/model.ts';
import type {VersionDag} from '../features/dag/model.ts';
import type {SettingsSnapshot} from '../features/settings/model.ts';
import type {ReleaseReadiness} from '../features/release/model.ts';
import type {AttemptRecord} from '../features/agents/model.ts';
import type {VersionMetadata} from '../features/metadata/model.ts';

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
    // #264 E1: requirements/qa/maintenance have no selected server read
    // model yet — honest NOT_CONNECTED, never a fabricated empty roster.
    case 'requirements':return <RequirementsView nodes={[]} connected={false}/>
    case 'qa':return <QaView records={[]} connected={false}/>
    case 'release':return <ReleaseRoute versionId={versionId}/>;
    case 'maintenance':return <MaintenanceView defects={[]} learnings={[]}
      connected={false}/>;
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
  // #264 E1 R91: real observed runs of this installed server's /api/actions
  // seam. Before any run the roster is honestly empty-UNKNOWN (never a
  // claimed confirmed-empty history); after a real dispatch the attempt row
  // shows the actual observed state with UNKNOWN telemetry preserved.
  const attempts=useHost<AgentAttemptsResponse>('agent-attempts',
    adapter=>adapter.agentAttempts(),[]);
  if(attempts.state==='loading')return <Skeleton lines={6}/>;
  if(attempts.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={attempts.reason??undefined}/>;
  return guard(versionId,
    <AgentsPanel attempts={(attempts.data?.attempts??[]) as AttemptRecord[]}
      connected={attempts.data?.connected===true}/>);
}
function DagRoute({versionId}:{versionId:string|null}){
  const dag=useHost<DagResponse>('dag',
    adapter=>adapter.dag(versionId??''),[versionId]);
  // #264 J02: independent S0/S1 dual-epoch read. An unmounted seam
  // (DUAL_EPOCH_NOT_CONFIGURED) degrades only this panel — the single-epoch
  // DAG below keeps its own honest NOT_CONNECTED semantics.
  const dualEpoch=useHost<DualEpochResponse>('dual-epoch',
    adapter=>adapter.dualEpoch(versionId??''),[versionId]);
  const epochPanel=<DualEpochPanel projection={(dualEpoch.data??null) as
    (DualEpochResponse|null)} state={dualEpoch.state}
    reason={dualEpoch.reason??null}/>;
  if(dag.state==='loading')return <Skeleton lines={6}/>;
  if(dag.state==='not_connected')
    // J02 truth: at a frozen-epoch descendant the single-epoch DAG read
    // honestly denies — the dual-epoch panel above carries the S0/S1
    // projection, so it must render on this path too, never be swallowed.
    return guard(versionId,
      <>{epochPanel}
        <EmptyState title="NOT_CONNECTED" hint={dag.reason??undefined}/>
      </>);
  return guard(versionId,
    <>{epochPanel}
      <DagView dag={(dag.data??null) as VersionDag|null} versionId={versionId}/>
    </>);
}
function ReleaseRoute({versionId}:{versionId:string|null}){
  const release=useHost<ReleaseResponse>('release',
    adapter=>adapter.release(versionId??''),[versionId]);
  // #264 E1 R91: the real J07c submission seam. The exact source SHA comes
  // from the freshly read evidence snapshot; the operator action key comes
  // from the server's ?actionKey= contract (same as the legacy consoles) —
  // without it the server's same-origin/CSRF admission denies. Every outcome
  // (DENIED with its real reason, UNKNOWN, transport failure) renders
  // truthfully; nothing here fabricates a Release PASS.
  const [lastAction,setLastAction]=useState<
    {kind:string;status:string;reason:string|null}|null>(null);
  const [submitting,setSubmitting]=useState(false);
  const act=(kind:'merge'|'tag'|'publish')=>{
    if(submitting)return;
    setSubmitting(true);
    const actionKey=new URLSearchParams(window.location.search)
      .get('actionKey')??'';
    void submitReleaseAction(actionKey,kind).then((result:
      ReleaseActionResult&{transportError?:string})=>{
      setLastAction({kind,status:result.status??'UNKNOWN',
        reason:result.reason??result.transportError??null});
    }).catch(error=>{
      setLastAction({kind,status:'DENIED',
        reason:error instanceof Error?error.message:String(error)});
    }).finally(()=>setSubmitting(false));
  };
  if(release.state==='loading')return <Skeleton lines={4}/>;
  const readiness=(release.data?.readiness??null) as ReleaseReadiness|null;
  return guard(versionId,
    <>
      <ReleaseView readiness={readiness}
        connected={release.data?.connected===true}
        onAction={operation=>act(operation)}/>
      <p role="status" aria-live="polite" data-release-last-action={
        lastAction?`${lastAction.kind}:${lastAction.status}`:undefined}
        style={{margin:0,color:'var(--text-muted)',fontSize:13}}>
        {lastAction
          ?`${lastAction.kind} → ${lastAction.status}`+
            (lastAction.reason?` — ${lastAction.reason}`:'')
          :null}
      </p>
    </>);
}
function MetadataRoute({versionId}:{versionId:string|null}){
  // Both hooks run unconditionally (React hooks order); the frozen-roster
  // Task read also supplies the expected Task ids for missing-report truth.
  const metadata=useHost<MetadataResponse>('metadata',
    adapter=>adapter.metadata(versionId??''),[versionId]);
  const tasks=useHost<TasksResponse>('tasks',
    adapter=>adapter.tasks(versionId??''),[versionId]);
  if(metadata.state==='loading')return <Skeleton lines={6}/>;
  if(metadata.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={metadata.reason??undefined}/>;
  const data=metadata.data;
  const connected=data?.connected===true;
  // #264 E1: consume the Host-qualified arrays as-is (F6: missing telemetry
  // must never render as observed 0); never fabricate expected ids.
  const expectedTaskIds=tasks.state==='ready'&&tasks.data?.connected===true
    ?tasks.data.tasks?.map(task=>task.id)??[]
    :[];
  const version=connected&&data
    ?{versionId:data.versionId??versionId??'',
      attempts:(data.attempts??[]) as VersionMetadata['attempts'],
      reports:(data.reports??[]) as VersionMetadata['reports'],
      lineage:(data.lineage??[]) as VersionMetadata['lineage'],
      learnedOutcomes:(data.learnedOutcomes??[]) as
        VersionMetadata['learnedOutcomes']}
    :null;
  return guard(versionId,
    <MetadataCenter version={version} allVersions={[]}
      expectedTaskIds={expectedTaskIds} connected={connected}/>);
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
