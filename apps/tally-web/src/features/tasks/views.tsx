/** UI-C #230 — Task list / board / details drawer React views. Data arrives
 * via props from the Host read model (UI-E wires the adapter); views never
 * fabricate status and keep UNKNOWN/STALE explicitly visible.
 */
import {useState} from 'react';
import type {TaskRecord,TaskFilters,SortKey} from './model.ts';
import {TASK_STATUSES,STATUS_LABEL,boardColumns,blockerSummary,
  writeSetConflicts,sortTasks} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Drawer} from '../../components/ui/Drawer.tsx';
import {DataTable} from '../../components/ui/DataTable.tsx';
import {EmptyState} from '../../components/ui/States.tsx';
import {Collapse} from '../../components/ui/Collapse.tsx';

export type ViewMode='list'|'board';
export function TasksView({tasks,filters,setFilters,onSelectTask,selectedTaskId,
  locale='zh',versionId,connected=true}:
  {tasks:readonly TaskRecord[];filters:TaskFilters;
   setFilters:(next:TaskFilters)=>void;onSelectTask:(id:string)=>void;
   selectedTaskId:string|null;locale?:'zh'|'en';versionId:string|null;
   connected?:boolean}){
  const zh=locale==='zh';
  const [mode,setMode]=useState<ViewMode>('list');
  const [sort,setSort]=useState<SortKey>('severity');
  if(!connected){
    return <EmptyState title={zh?'未连接真实数据源':'Not connected to a real data source'}
      hint={zh?'等待 UI-E #232 接入 Host 读模型':'Host read model arrives with UI-E #232'}/>;
  }
  if(!versionId){
    return <EmptyState title={zh?'请先在右上角选择版本':'Select a version first'}/>;
  }
  const conflicts=writeSetConflicts(tasks);
  return <div>
    <div style={{display:'flex',gap:'var(--s2)',flexWrap:'wrap',alignItems:'center',
      marginBottom:'var(--s4)'}}>
      <input type="search" value={filters.text} style={{minWidth:200}}
        placeholder={zh?'搜索任务 / 写集路径':'Search tasks / write-set paths'}
        aria-label={zh?'搜索':'search'}
        onChange={event=>setFilters({...filters,text:event.target.value})}/>
      <select aria-label={zh?'状态':'status'} value={filters.status??''}
        onChange={event=>setFilters({...filters,
          status:(event.target.value||null) as TaskFilters['status']})}>
        <option value="">{zh?'全部状态':'All statuses'}</option>
        {TASK_STATUSES.map(status=><option key={status} value={status}>
          {zh?STATUS_LABEL[status].zh:STATUS_LABEL[status].en}</option>)}
      </select>
      <select aria-label={zh?'严重级':'severity'} value={filters.severity??''}
        onChange={event=>setFilters({...filters,
          severity:(event.target.value||null) as TaskFilters['severity']})}>
        <option value="">{zh?'全部级别':'All severities'}</option>
        {['P0','P1','P2','UNKNOWN'].map(severity=>
          <option key={severity} value={severity}>{severity}</option>)}
      </select>
      <label style={{display:'flex',gap:6,alignItems:'center',color:'var(--text-secondary)'}}>
        <input type="checkbox" checked={filters.onlyBlocked}
          onChange={event=>setFilters({...filters,onlyBlocked:event.target.checked})}/>
        {zh?'只看阻塞':'Blocked only'}</label>
      <select aria-label={zh?'排序':'sort'} value={sort}
        onChange={event=>setSort(event.target.value as SortKey)}>
        {(['severity','status','updated','id'] as SortKey[]).map(key=>
          <option key={key} value={key}>{key}</option>)}
      </select>
      <div role="group" aria-label={zh?'视图':'view'}
        style={{marginLeft:'auto',display:'flex',gap:2}}>
        {(['list','board'] as ViewMode[]).map(value=>
          <button key={value} aria-pressed={mode===value}
            onClick={()=>setMode(value)}
            style={{padding:'6px 12px',border:mode===value
              ?'1px solid var(--brand)':'1px solid var(--border-strong)',
              background:mode===value?'var(--info-dim)':'transparent',
              color:'var(--text-primary)',cursor:'pointer',font:'inherit'}}>
            {value==='list'?(zh?'列表':'List'):(zh?'看板':'Board')}</button>)}
      </div>
    </div>
    {conflicts.length>0?<div role="alert" style={{marginBottom:'var(--s3)',
      padding:'var(--s3) var(--s4)',border:'1px solid #a2763c',borderRadius:'var(--r-md)',
      background:'var(--warn-dim)',color:'var(--warn)',fontSize:12.5}}>
      {zh?'写集冲突:':'Write-set conflicts:'}
      {conflicts.slice(0,3).map(conflict=>
        ` ${conflict.path} (${conflict.taskIds.join(', ')})`).join(';')}
      {conflicts.length>3?` +${conflicts.length-3}`:''}
    </div>:null}
    {mode==='list'
      ?<TaskListView tasks={sortTasks(tasks,sort)} onSelectTask={onSelectTask}
          selectedTaskId={selectedTaskId} locale={locale}/>
      :<TaskBoardView tasks={sortTasks(tasks,sort)} onSelectTask={onSelectTask}
          locale={locale}/>}
  </div>;
}
function severityTone(severity:TaskRecord['severity']){
  return severity==='P0'?'danger':severity==='P1'?'warn':'neutral';
}
function statusTone(status:TaskRecord['status']){
  return status==='done'?'ok':status==='blocked'||status==='failed'?'danger'
    :status==='review'?'info':'neutral';
}
export function TaskListView({tasks,onSelectTask,locale='zh'}:
  {tasks:readonly TaskRecord[];onSelectTask:(id:string)=>void;
   selectedTaskId?:string|null;locale?:'zh'|'en'}){
  const zh=locale==='zh';
  if(tasks.length===0)
    return <EmptyState title={zh?'没有匹配的任务':'No matching tasks'}/>;
  return <DataTable rows={tasks} rowKey={task=>task.id}
    onRowClick={task=>onSelectTask(task.id)}
    columns={[
      {key:'id',header:zh?'任务':'Task',
        cell:task=><><code>{task.id}</code><div>{task.title}</div></>},
      {key:'severity',header:zh?'级别':'Sev',width:'70px',
        cell:task=><Badge tone={severityTone(task.severity)}>{task.severity}</Badge>},
      {key:'status',header:zh?'状态':'Status',
        cell:task=><Badge tone={statusTone(task.status)}>
          {zh?STATUS_LABEL[task.status].zh:STATUS_LABEL[task.status].en}</Badge>},
      {key:'agent',header:'Agent',cell:task=>task.agentId??'—'},
      {key:'ci',header:'CI',cell:task=><span style={{
        color:task.ciState==='SUCCESS'?'var(--ok)'
          :task.ciState==='FAILURE'?'var(--danger)':'var(--text-muted)'}}>
          {task.ciState}</span>},
      {key:'dep',header:zh?'前置':'Deps',cell:task=>
        task.blockedBy.length?task.blockedBy.join(', '):'—'},
    ]}/>;
}
export function TaskBoardView({tasks,onSelectTask,locale='zh'}:
  {tasks:readonly TaskRecord[];onSelectTask:(id:string)=>void;locale?:'zh'|'en'}){
  const zh=locale==='zh';
  const columns=boardColumns(tasks);
  return <div style={{display:'grid',gap:'var(--s3)',
    gridTemplateColumns:'repeat(auto-fit,minmax(210px,1fr))'}}>
    {columns.map(column=><section key={column.status} aria-label={
      zh?STATUS_LABEL[column.status].zh:STATUS_LABEL[column.status].en}
      style={{background:'var(--bg-panel)',border:'1px solid var(--border-subtle)',
        borderRadius:'var(--r-lg)',padding:'var(--s3)',minHeight:120}}>
      <div style={{display:'flex',justifyContent:'space-between',alignItems:'center',
        marginBottom:'var(--s2)'}}>
        <strong style={{fontSize:13}}>
          {zh?STATUS_LABEL[column.status].zh:STATUS_LABEL[column.status].en}</strong>
        <Badge>{column.tasks.length}</Badge>
      </div>
      {column.tasks.length===0?<div style={{color:'var(--text-faint)',fontSize:12}}>—</div>
        :column.tasks.map(task=><button key={task.id}
          onClick={()=>onSelectTask(task.id)}
          style={{display:'block',width:'100%',textAlign:'left',marginBottom:'var(--s2)',
            padding:'var(--s2) var(--s3)',border:'1px solid var(--border-subtle)',
            borderRadius:'var(--r-md)',background:'var(--bg-raised)',
            color:'var(--text-primary)',cursor:'pointer',font:'inherit'}}>
          <div style={{display:'flex',gap:6,alignItems:'center'}}>
            <Badge tone={severityTone(task.severity)}>{task.severity}</Badge>
            <code style={{fontSize:11}}>{task.id}</code>
          </div>
          <div style={{fontSize:12.5,marginTop:4,overflow:'hidden',
            textOverflow:'ellipsis',whiteSpace:'nowrap'}}>{task.title}</div>
        </button>)}
    </section>)}
  </div>;
}
export function TaskDrawer({task,onClose,locale='zh'}:
  {task:TaskRecord|null;onClose:()=>void;locale?:'zh'|'en'}){
  const zh=locale==='en'?false:true;
  return <Drawer open={task!==null} onClose={onClose}
    title={task?`${task.id} · ${task.title}`:''}>
    {task?<div style={{display:'grid',gap:'var(--s4)'}}>
      <div style={{display:'flex',gap:'var(--s2)',flexWrap:'wrap'}}>
        <Badge tone={severityTone(task.severity)}>{task.severity}</Badge>
        <Badge tone={statusTone(task.status)}>
          {zh?STATUS_LABEL[task.status].zh:STATUS_LABEL[task.status].en}</Badge>
        <Badge tone={task.claimState==='ACCEPTED'?'ok':'warn'}>
          Claim:{task.claimState}</Badge>
      </div>
      <section aria-label={zh?'阻塞原因':'blockers'}>
        <strong style={{display:'block',marginBottom:'var(--s2)'}}>
          {zh?'需要处理(最多 3 条)':'Actionable (≤3)'}</strong>
        {blockerSummary(task).length===0
          ?<span style={{color:'var(--ok)'}}>{zh?'无阻塞':'No blockers'}</span>
          :<ul style={{margin:0,paddingLeft:18}}>
            {blockerSummary(task).map(blocker=><li key={blocker.code}>
              <code>{blocker.code}</code> — {blocker.detail}</li>)}
          </ul>}
      </section>
      <section aria-label={zh?'执行与评审':'execution'}>
        <strong style={{display:'block',marginBottom:'var(--s2)'}}>
          {zh?'执行与评审':'Execution & review'}</strong>
        <dl style={{display:'grid',gridTemplateColumns:'auto 1fr',gap:'6px 12px',
          margin:0,fontSize:13}}>
          <dt style={{color:'var(--text-muted)'}}>Agent</dt>
          <dd style={{margin:0}}>{task.agentId??'—'}</dd>
          <dt style={{color:'var(--text-muted)'}}>Attempt</dt>
          <dd style={{margin:0}}>{task.attemptId??'—'} · {task.attemptState??'—'}</dd>
          <dt style={{color:'var(--text-muted)'}}>PR</dt>
          <dd style={{margin:0}}>{task.prNumber?`#${task.prNumber}`:'—'}</dd>
          <dt style={{color:'var(--text-muted)'}}>Review</dt>
          <dd style={{margin:0}}>{task.reviewState}</dd>
          <dt style={{color:'var(--text-muted)'}}>CI</dt>
          <dd style={{margin:0}}>{task.ciState}</dd>
          {task.requirementRef?<>
            <dt style={{color:'var(--text-muted)'}}>{zh?'需求':'Requirement'}</dt>
            <dd style={{margin:0}}><code>{task.requirementRef}</code></dd></>:null}
        </dl>
      </section>
      <Collapse title={zh?'技术证据(原始状态)':'Technical evidence'}
        defaultOpen={false}>
        <dl style={{display:'grid',gridTemplateColumns:'auto 1fr',gap:'6px 12px',
          margin:0,fontSize:12.5}}>
          <dt style={{color:'var(--text-muted)'}}>version</dt>
          <dd style={{margin:0}}><code>{task.versionId}</code></dd>
          <dt style={{color:'var(--text-muted)'}}>blocked_by</dt>
          <dd style={{margin:0}}><code>{task.blockedBy.join(', ')||'—'}</code></dd>
          <dt style={{color:'var(--text-muted)'}}>write-set</dt>
          <dd style={{margin:0}}><code>{task.writeSet.join('; ')||'—'}</code></dd>
          <dt style={{color:'var(--text-muted)'}}>updatedAt</dt>
          <dd style={{margin:0}}><code>{task.updatedAt}</code></dd>
        </dl>
      </Collapse>
    </div>:null}
  </Drawer>;
}
