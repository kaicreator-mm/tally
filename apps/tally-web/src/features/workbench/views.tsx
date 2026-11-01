/** Human actions from one qualified Host inbox; all effects remain Host-owned. */
import type {WorkbenchSnapshot,WorkbenchPage} from './model.ts';
import {activeWorkActions,inboxCounts,actionTaskLink} from './model.ts';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Badge} from '../../components/ui/Badge.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const pageNames:Record<WorkbenchPage,string>={
  inbox:'我的待办',approvals:'审批与决策',blockers:'阻塞与恢复',overview:'工作概览'};
export function WorkbenchView({snapshot,page,onNavigate}:
  {snapshot:WorkbenchSnapshot|null;page:WorkbenchPage;
   onNavigate:(hash:string)=>void}){
  const rows=activeWorkActions(snapshot,page);
  if(!rows)return <EmptyState title="NOT_CONNECTED"
    hint="工作台需要当前用户有权读取的受信 Host 人工待办队列；不可将未知当作零待办。"/>;
  const count=inboxCounts(snapshot);
  return <div style={{display:'grid',gap:'var(--s4)'}}>
    <Card>
      <CardHeader title={pageNames[page]}
        sub="同一受信待办源 · 仅显示待办，不自动授予审批、执行或发布权限"/>
      <div style={{display:'flex',gap:12,flexWrap:'wrap'}}>
        <Badge tone="neutral">待办 {count?.pending??'UNKNOWN'}</Badge>
        <Badge tone="warn">审批 {count?.approvals??'UNKNOWN'}</Badge>
        <Badge tone="neutral">阻塞 {count?.blockers??'UNKNOWN'}</Badge>
      </div>
    </Card>
    <Card>
      <CardHeader title="需要人工介入的事项" sub="由 Host 读模型标记负责人及可操作事项"/>
      {rows.length===0?<EmptyState title="当前没有待处理事项"
        hint="仅适用于此来源覆盖范围内的已连接队列"/>:
      <div style={{display:'grid',gap:'var(--s3)'}}>
        {rows.map(action=><div key={action.id} style={{
          borderBottom:'1px solid var(--border-subtle)',padding:'8px 0',
          display:'flex',alignItems:'flex-start',justifyContent:'space-between',gap:12}}>
          <div style={{display:'grid',gap:5,minWidth:0}}>
            <strong>{action.title}</strong>
            <span style={{fontSize:12,color:'var(--text-secondary)'}}>
              {action.kind} · {action.state} · {action.ownerRole} · {action.projectId}
              {action.versionId?' / '+action.versionId:''}</span>
            {action.reason?<span style={{fontSize:12,color:'var(--warn)'}}>{action.reason}</span>:null}
            <a href={action.sourceRef} target="_blank" rel="noopener noreferrer"
              style={{fontSize:12}}>查看原始证据</a>
          </div>
          {actionTaskLink(action)?<button type="button"
            onClick={()=>onNavigate(actionTaskLink(action)!)}
            style={{font:'inherit',cursor:'pointer'}}>查看任务</button>:null}
        </div>)}
      </div>}
    </Card>
  </div>;
}
