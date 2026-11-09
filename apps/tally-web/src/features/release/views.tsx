/** UI-D #231 — user-facing release readiness: 1-3 actionable blockers, no
 * decorative controls; enabled actions exist only with fresh Host permits. */
import {releaseBlockers,actionDisplay} from './model.ts';
import type {ReleaseReadiness} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Button} from '../../components/ui/Button.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Collapse} from '../../components/ui/Collapse.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const OP_LABEL:{zh:string;en:string}[]=[
  {zh:'合并到 main',en:'Merge to main'},{zh:'创建发布 Tag',en:'Create tag'},
  {zh:'发布 Release',en:'Publish release'}];
export function ReleaseView({readiness,onAction,connected=true,locale='zh'}:
  {readiness:ReleaseReadiness|null;connected?:boolean;locale?:'zh'|'en';
   onAction?:(operation:'merge'|'tag'|'publish')=>void}){
  const zh=locale==='zh';
  if(!connected)return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
    hint="UI-E #232"/>;
  if(!readiness)return <EmptyState title={zh?'请先选择版本':'Select a version first'}/>;
  const blockers=releaseBlockers(readiness);
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?'发布资格':'Release qualification'}
        sub={zh?`版本 ${readiness.versionId} · RQ:${readiness.rqState} · Owner授权:${readiness.ownerGrantState}`
          :`version ${readiness.versionId}`}/>
      {blockers.length===0
        ?<Badge tone="ok">{zh?'所有发布条件已满足':'All conditions met'}</Badge>
        :<ul style={{margin:0,paddingLeft:18}}>
          {blockers.map(blocker=><li key={blocker.code}>
            <code>{blocker.code}</code> — {blocker.detail}
            {blocker.owner?`(${zh?'责任方':'owner'}: ${blocker.owner})`:''}</li>)}
        </ul>}
      <div style={{display:'flex',gap:'var(--s3)',marginTop:'var(--s4)',
        flexWrap:'wrap'}}>
        {(['merge','tag','publish'] as const).map((operation,index)=>{
          const display=actionDisplay(readiness,operation);
          // No Host dispatch callback = no real effect; never enable a no-op.
          const wired=display.enabled&&typeof onAction==='function';
          return <div key={operation} style={{display:'grid',gap:4}}>
            <Button variant={wired?'primary':'secondary'}
              disabled={!wired}
              onClick={wired?()=>onAction?.(operation):undefined}>
              {zh?OP_LABEL[index].zh:OP_LABEL[index].en}
            </Button>
            {display.reason||!wired?<span style={{color:'var(--warn)',fontSize:12,
              maxWidth:220}}>{display.reason??'HOST_ACTION_NOT_CONNECTED'}</span>
              :<span style={{color:'var(--text-faint)',fontSize:12}}>
                  {zh?'已获新鲜 Host 许可':'fresh Host permit'}</span>}
          </div>;})}
      </div>
    </Card>
    <Collapse title={zh?'专家视图:13 项 gate 原始矩阵':'Expert: raw gate matrix'}
      defaultOpen={false}>
      <table style={{width:'100%',borderCollapse:'collapse',fontSize:12.5}}>
        <tbody>{readiness.gates.map(gate=><tr key={gate.name}>
          <td style={{padding:'6px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
            {gate.name}</td>
          <td style={{padding:'6px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
            <Badge tone={gate.status==='PASS'?'ok':gate.status==='FAIL'?'danger':'neutral'}>
              {gate.status}</Badge></td>
          <td style={{padding:'6px 10px',borderBottom:'1px solid var(--border-subtle)',
            color:'var(--text-muted)'}}>{gate.owner??'—'}</td>
        </tr>)}</tbody>
      </table>
    </Collapse>
  </div>;
}
