import {governanceItems,governanceAuthority,
  type GovernanceKind,type GovernanceSnapshot} from './model.ts';
import {EmptyState} from '../../components/ui/States.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Badge} from '../../components/ui/Badge.tsx';
const titles:Record<GovernanceKind,string>={
  workflow:'Workflow 管理', 'domain-package':'Domain Packages',
  experience:'经验与改进',audit:'审计与溯源'};
export function GovernanceView({snapshot,kind}:{
  snapshot:GovernanceSnapshot|null;kind:GovernanceKind}){
  const rows=governanceItems(snapshot,kind);
  if(rows===null)return <EmptyState title="NOT_CONNECTED"
    hint="需要受信 Host/ADS 来源的治理读模型，不能用静态配置冒充已生效状态。"/>;
  return <Card>
    <CardHeader title={titles[kind]}
      sub="受信来源只读 · Workflow、经验候选和 Domain Package 不自动激活或取得 Kernel 权限"/>
    {rows.length===0?<EmptyState title="当前来源范围内暂无记录"/>:
      <div style={{display:'grid',gap:'var(--s4)'}}>
        {rows.map(item=><article key={item.id}
          style={{borderBottom:'1px solid var(--border-subtle)',paddingBottom:12}}>
          <div style={{display:'flex',gap:12,alignItems:'center',flexWrap:'wrap'}}>
            <strong>{item.title}</strong><Badge>{item.state}</Badge>
            <Badge tone={item.authority==='BUILD_SELECTED'?'ok':'neutral'}>
              {governanceAuthority(item)}</Badge>
          </div>
          <div style={{fontSize:12,color:'var(--text-secondary)',marginTop:4}}>
            {item.id} · {item.projectId} / {item.versionId??'workspace'}
          </div>
          {item.summary?<p>{item.summary}</p>:null}
          <a href={item.sourceRef} target="_blank" rel="noopener noreferrer"
            style={{fontSize:12}}>查看来源证据</a>
        </article>)}
      </div>}
  </Card>;
}
