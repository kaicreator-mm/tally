import type {DeploymentRecord} from './model.ts';
import {environmentOverview,deploymentBadge,versionDeployments} from './model.ts';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Badge} from '../../components/ui/Badge.tsx';
import {EmptyState} from '../../components/ui/States.tsx';
export function DeploymentView({records,versionId,connected}:{
  records:readonly DeploymentRecord[];versionId:string;connected:boolean}){
  if(!connected)return <EmptyState title="NOT_CONNECTED"
    hint="部署记录需要受信 Host 按环境读取；不能从 Release Tag 推断已部署。"/>;
  const current=environmentOverview(versionDeployments(records,versionId));
  return <Card>
    <CardHeader title={'部署环境 · '+versionId}
      sub="发布与部署独立；健康/回滚须有真实回执，部署状态不会自动授予 Release PASS"/>
    {current.length===0?<EmptyState title="当前版本无已验证部署记录"/>:
    <div style={{display:'grid',gap:12}}>
      {current.map(item=><div key={item.id} style={{
        display:'flex',alignItems:'start',gap:14,flexWrap:'wrap',
        borderBottom:'1px solid var(--border-subtle)',paddingBottom:10}}>
        <div style={{flex:'1 1 160px'}}>
          <strong>{item.environment}</strong>
          <div style={{fontSize:12,color:'var(--text-secondary)'}}>
            {item.releaseSha?.slice(0,12)??'RELEASE_UNKNOWN'} · {item.deployedAt??'TIME_UNKNOWN'}
          </div>
        </div>
        <Badge tone={deploymentBadge(item)}>{item.state}</Badge>
        <Badge tone={item.health==='HEALTHY'?'ok':'warn'}>{item.health}</Badge>
        {item.receiptRef?<a href={item.receiptRef} target="_blank"
          rel="noopener noreferrer">部署回执</a>:<span>回执 UNKNOWN</span>}
        {item.rollbackReceiptRef?<a href={item.rollbackReceiptRef} target="_blank"
          rel="noopener noreferrer">回滚回执</a>:<span>回滚 UNKNOWN</span>}
      </div>)}
    </div>}
  </Card>;
}
