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
import {hostAdapter,startSelectedAgentTask,submitReleaseAction,transitionSelectedTaskReady} from '../adapters/host.ts';
import type {Route} from './router.ts';
import type {TaskReadyHostResponse,AgentDispatchResponse,Catalog,TaskDraftsResponse,RequirementsResponse,QaResponse,TasksResponse,DagResponse,DualEpochResponse,ReleaseResponse,
  SettingsResponse,MetadataResponse,AgentAttemptsResponse,
  ReleaseActionResult} from '../adapters/host.ts';
import {TasksView,TaskDrawer} from '../features/tasks/views.tsx';
import {agentStartDecision} from '../features/tasks/agent-start-model.ts';
import type {RequirementNode} from '../features/requirements/model.ts';
import type {QaRecord} from '../features/qa/model.ts';
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
    case 'overview':return <OverviewRoute/>;
    case 'product':return <RequirementsRoute versionId={versionId}/>;
    case 'tasks':return <TasksRoute versionId={versionId}/>;
    case 'agents':return <AgentsRoute versionId={versionId}/>;
    case 'l2-dag':return <DagRoute versionId={versionId}/>;
    case 'requirements':return <RequirementsRoute versionId={versionId}/>;
    case 'qa':return <QaRoute versionId={versionId}/>;
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
function OverviewRoute(){
  const catalog=useHost<Catalog>('catalog',adapter=>adapter.catalog(),[]);
  if(catalog.state==='loading')return <Skeleton lines={4}/>;
  if(catalog.state==='not_connected')
    return <EmptyState title="NOT_CONNECTED" hint={catalog.reason??undefined}/>;
  const source=catalog.data?.source;
  return <section>
    <h2>已连接 GitHub 项目</h2>
    <p>仓库：<strong>{source?.repository??'UNKNOWN'}</strong></p>
    <p>当前源码 SHA：<code>{source?.sourceSha??'UNKNOWN'}</code></p>
    <p>Source currentness：{source?.currentness??'UNKNOWN'}
      {' · '}Readback：{source?.readback??'UNKNOWN'}</p>
    <p>这里的连接只代表 GitHub Source 可读取，不代表 Product
      已冻结、Task DAG 已接受或 Agent 已获执行授权。</p>
    <p><a href="#/settings/git-github">添加 / 切换项目</a></p>
  </section>;
}
function RequirementsRoute({versionId}:{versionId:string|null}){
  const source=useHost<RequirementsResponse>('requirements',
    adapter=>adapter.requirements(versionId??''),[versionId]);
  if(source.state==='loading')return <Skeleton lines={4}/>;
  if(source.state==='not_connected')
    return guard(versionId,<EmptyState title="NOT_CONNECTED"
      hint={source.reason??undefined}/>);
  return guard(versionId,<RequirementsView
    nodes={(source.data?.nodes??[]) as RequirementNode[]}
    connected={source.data?.connected===true}/>);
}
function QaRoute({versionId}:{versionId:string|null}){
  const source=useHost<QaResponse>('qa',
    adapter=>adapter.qa(versionId??''),[versionId]);
  if(source.state==='loading')return <Skeleton lines={4}/>;
  if(source.state==='not_connected')
    return guard(versionId,<EmptyState title="NOT_CONNECTED"
      hint={source.reason??undefined}/>);
  return guard(versionId,<QaView
    records={(source.data?.records??[]) as QaRecord[]}
    connected={source.data?.connected===true}/>);
}
function TasksRoute({versionId}:{versionId:string|null}){
  const [filters,setFilters]=useState<TaskFilters>({...EMPTY_FILTERS});
  const [selectedId,setSelectedId]=useState<string|null>(null);
  const [starting,setStarting]=useState(false);
  const [lastStart,setLastStart]=useState<{taskId:string;message:string}|null>(null);
  const [readyTransitionBusy,setReadyTransitionBusy]=useState(false);
  const [readyTransitionResult,setReadyTransitionResult]=useState<{
    issueNumber:number;repository:string;message:string
  }|null>(null);
  const [checkingAdmission,setCheckingAdmission]=useState(false);
  const [admissionResult,setAdmissionResult]=useState<{
    issueNumber:number;repository:string;versionId:string;
    stage:string;reason:string
  }|null>(null);
  const tasks=useHost<TasksResponse>('tasks',
    adapter=>adapter.tasks(versionId??''),[versionId]);
  // A new repository may have genuine draft Issues but no qualified
  // frozen Task DAG. Keep these TWO levels of evidence separate.
  const agentHost=useHost<AgentDispatchResponse>('agent-dispatch',
    adapter=>adapter.agentDispatch(),[versionId]);
  const readyHost=useHost<TaskReadyHostResponse>('task-ready-host',
    adapter=>adapter.taskReadyHost(),[versionId]);
  const drafts=useHost<TaskDraftsResponse>('task-drafts',
    adapter=>adapter.taskDrafts(versionId??''),[versionId]);
  const checkAdmission=async(issueNumber:number)=>{
    if(!versionId||checkingAdmission)return;
    const repository=drafts.data?.repository??'';
    setCheckingAdmission(true);setAdmissionResult(null);
    try{
      const outcome=await hostAdapter.taskAdmission(versionId,issueNumber);
      setAdmissionResult({issueNumber,repository,versionId,
        stage:outcome.stage??'NOT_CONNECTED',
        reason:outcome.reason??'UNKNOWN'});
    }catch{
      setAdmissionResult({issueNumber,repository,versionId,
        stage:'NOT_CONNECTED',reason:'TASK_ADMISSION_READ_FAILED'});
    }finally{setCheckingAdmission(false);}
  };
  const transitionReady=async(issueNumber:number,sourceSha:string)=>{
    if(!versionId||readyTransitionBusy||!drafts.data?.repository)return;
    const repository=drafts.data.repository;
    setReadyTransitionBusy(true);setReadyTransitionResult(null);
    try{
      const outcome=await transitionSelectedTaskReady({
        issueNumber,repository,versionId,sourceSha});
      setReadyTransitionResult({issueNumber,repository,
        message:outcome.status+(outcome.reason?' · '+outcome.reason:'')+
          '（重新检查任务准入以查看最新 GitHub 状态）'});
    }catch{
      setReadyTransitionResult({issueNumber,repository,
        message:'UNKNOWN · TASK_READY_ACTION_READBACK_UNKNOWN'});
    }finally{setReadyTransitionBusy(false);}
  };
  if(tasks.state==='loading')return <Skeleton lines={6}/>;
  if(tasks.state==='not_connected')
    return guard(versionId,<section>
      <h2>GitHub 开发任务草稿</h2>
      <p>正式 Task DAG 尚未连接：<code>{tasks.reason??'UNKNOWN'}</code>。
        草稿 Issue 不代表已接受的 Task/Claim，也没有 Agent 执行权限。</p>
      {drafts.state==='loading'?<Skeleton lines={3}/>:
        drafts.state==='not_connected'?
          <p>GitHub 草稿读取不可用：{drafts.reason??'UNKNOWN'}</p>:
          <><p>{drafts.data?.coverage==='LATEST_100_ISSUES_ONLY'
              ?'仅显示最近 100 条 GitHub Issue 中识别到的草稿'
              :'已从 GitHub 读取当前项目的任务草稿'}</p>
          {(drafts.data?.drafts??[]).length===0?
            <p>暂时没有创建过的任务草稿。</p>:
            <ul>{drafts.data?.drafts?.map(draft=><li key={draft.issueNumber}>
              <a href={draft.issueUrl} target="_blank" rel="noopener noreferrer">
                #{draft.issueNumber} {draft.title}</a>
              {' · '}{draft.githubIssueState}
              {' · '}{draft.nativeWorkflow??'NONCANONICAL'}
              {' · '}{draft.sourceCurrentness??'UNKNOWN'}
              {' · '}UNQUALIFIED_DRAFT
              <button type="button" disabled={checkingAdmission}
                onClick={()=>{void checkAdmission(draft.issueNumber);}}>
                检查任务准入
              </button>
              {draft.nativeWorkflow==='PLANNED'&&
               draft.sourceCurrentness==='CURRENT'&&
               readyHost.data?.connected===true&&
               readyHost.data?.repository===drafts.data?.repository&&
               readyHost.data?.versionId===versionId&&
               readyHost.data?.sourceSha===draft.sourceSha?
                <button type="button" disabled={readyTransitionBusy}
                  onClick={()=>{void transitionReady(draft.issueNumber,draft.sourceSha);}}>
                  请求可信 Host 转为 ready
                </button>:null}
              {readyTransitionResult?.issueNumber===draft.issueNumber&&
               readyTransitionResult.repository===drafts.data?.repository?
                <span role="status">{' · '}{readyTransitionResult.message}</span>:null}
              {admissionResult?.issueNumber===draft.issueNumber&&
               admissionResult.repository===drafts.data?.repository&&
               admissionResult.versionId===versionId?
                <span role="status">
                  {' · '}{admissionResult.stage}：
                  {admissionResult.reason}
                </span>:null}
            </li>)}</ul>}</>}
      <p><a href="#/settings/git-github">前往配置中心添加项目或创建 GitHub Task 草稿</a></p>
    </section>);
  const connected=tasks.data?.connected===true;
  const rows=(tasks.data?.tasks??[]) as TaskRecord[];
  const selectedTask=selectedId?rows.find(row=>row.id===selectedId)??null:null;
  const sourceSha=tasks.data?.source?.sourceSha??null;
  const decision=agentStartDecision({taskId:selectedTask?.id??null,
    claimState:selectedTask?.claimState??null,sourceSha,host:agentHost.data});
  const issueNumber=decision.issueNumber;
  const allowed=decision.enabled&&!starting;
  const reason=agentHost.state==='not_connected'?
    agentHost.reason??decision.reason:decision.reason;
  const start=async()=>{
    if(!allowed||issueNumber===null||!sourceSha||!selectedId)return;
    const taskId=selectedId;
    setStarting(true);setLastStart(null);
    try{
      const result=await startSelectedAgentTask(issueNumber,sourceSha);
      setLastStart({taskId,message:result.status+
        (result.reason?' · '+result.reason:'')});
    }catch{
      setLastStart({taskId,message:'UNKNOWN · AGENT_ACTION_READBACK_UNKNOWN'});
    }finally{setStarting(false);}
  };
  return guard(versionId,<>
    <TasksView tasks={rows} filters={filters} setFilters={setFilters}
      onSelectTask={id=>setSelectedId(id)} selectedTaskId={selectedId}
      versionId={versionId} connected={connected}/>
    <TaskDrawer task={selectedTask} onClose={()=>setSelectedId(null)}
      agentStart={selectedTask?{enabled:allowed,
        provider:agentHost.data?.provider??null,reason,
        pending:starting,lastResult:lastStart?.taskId===selectedId?
          lastStart.message:null,onStart:()=>{void start();}}:null}/>
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
  const selected=settings.data;
  if(selected?.connected===true&&selected.configurationUrl){
    return <section aria-label="GitHub 项目配置">
      <h2>GitHub 项目与连接</h2>
      <p>当前项目：<strong>{selected.selectedRepository??'未选择'}</strong>
        {' · '}{selected.selectedRef??'—'}
        {' · '}{selected.currentSha?.slice(0,12)??'UNKNOWN'}</p>
      <p><a href={selected.configurationUrl}>添加项目、切换项目、更新 GitHub 凭据</a></p>
      <p>Agent Host：{selected.agentHostMounted
        ?'已挂载（仍需独立任务授权）'
        :'未挂载；配置 GitHub 仅提供读取权限，不会自动授予 Agent 执行权'}</p>
      <p>其他设置尚未挂载真实 Host，不会展示模拟的已连接状态。</p>
    </section>;
  }
  return <SettingsCenter
    snapshot={(settings.data?.snapshot??null) as SettingsSnapshot|null}
    connected={settings.data?.connected===true}/>;
}
