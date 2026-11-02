import type {TaskActionCatalog,TaskActionRequest} from './model.ts';
import {taskActionsFor,taskActionRequest} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Button} from '../../components/ui/Button.tsx';
import {EmptyState} from '../../components/ui/States.tsx';
export function TaskActionsView({catalog,projectId,versionId,taskIssue,
  onDispatch,locale='zh'}:{catalog:TaskActionCatalog|null;
  projectId:string;versionId:string;taskIssue:number;
  onDispatch?:(request:TaskActionRequest)=>void;locale?:'zh'|'en'}){
  const zh=locale==='zh';
  const actions=taskActionsFor(catalog,projectId,versionId,taskIssue);
  if(actions===null)return <EmptyState title="NOT_CONNECTED"
    hint={zh?'当前版本没有经过验证的 Host 操作清单':'No qualified Host action catalog'}/>;
  if(actions.length===0)return <EmptyState title={zh?'暂无可执行操作':'No available actions'}/>;
  return <section aria-label={zh?'任务操作':'Task actions'}
    style={{display:'grid',gap:'var(--s3)'}}>
    {actions.map(action=>{
      const request=catalog?taskActionRequest(catalog,projectId,versionId,action):null;
      const enabled=request!==null&&typeof onDispatch==='function';
      return <div key={action.id} style={{display:'flex',alignItems:'center',
        justifyContent:'space-between',gap:12,padding:'8px 0',
        borderBottom:'1px solid var(--border-subtle)'}}>
        <div style={{display:'grid',gap:4,minWidth:0}}>
          <strong>{action.label}</strong>
          <span style={{fontSize:12,color:'var(--text-secondary)'}}>
            {action.kind} · r{action.revision}</span>
        </div>
        <div style={{display:'flex',alignItems:'center',gap:8}}>
          <Badge tone={enabled?'ok':'neutral'}>{enabled?
            (zh?'可操作':'Eligible'):(zh?'未准入':'Not admitted')}</Badge>
          <Button disabled={!enabled} onClick={enabled?
            ()=>onDispatch(request as TaskActionRequest):undefined}>
            {zh?'继续':'Continue'}
          </Button>
        </div>
      </div>;
    })}
  </section>;
}
