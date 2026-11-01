/** UI-C #230 — parallel Agent execution panel (same read model as tasks;
 * this is a view over attempts, not a substitute for the canonical Task list). */
import type {AttemptRecord} from './model.ts';
import {durationLabel,tokenLabel,telemetryCoverage,byAgent,
  tokenTotals,stateTone} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {DataTable} from '../../components/ui/DataTable.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

export function AgentsPanel({attempts,locale='zh',connected=true}:
  {attempts:readonly AttemptRecord[];locale?:'zh'|'en';connected?:boolean}){
  const zh=locale==='zh';
  if(!connected){
    return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
      hint={zh?'等待 UI-E #232 接入 Host 读模型':'Host read model arrives with UI-E #232'}/>;
  }
  if(attempts.length===0)
    return <EmptyState title={zh?'当前版本没有 Agent 执行记录':'No agent attempts in this version'}/>;
  const coverage=telemetryCoverage(attempts);
  const totals=tokenTotals(attempts);
  const groups=byAgent(attempts);
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?'遥测覆盖(缺失 ≠ 0)':'Telemetry coverage (missing ≠ 0)'}
        sub={zh?'仅统计有真实数据的尝试;UNKNOWN 保持 UNKNOWN':'Only measured attempts count; UNKNOWN stays UNKNOWN'}/>
      <div style={{display:'flex',gap:'var(--s4)',flexWrap:'wrap',fontSize:13}}>
        <span>{zh?'时长可测':'Duration measured'}: <strong>{coverage.measured}/{coverage.total}</strong>
          {coverage.coverage!==null?` (${Math.round(coverage.coverage*100)}%)`:''}</span>
        <span>{zh?'tokens 入(可测)':'tokens in (measured)'}:
          <strong> {totals.tokensIn}</strong>
          {totals.inCoverage!==null?` (${Math.round(totals.inCoverage*100)}%)`:''}</span>
        <span>{zh?'tokens 出(可测)':'tokens out (measured)'}:
          <strong> {totals.tokensOut}</strong>
          {totals.outCoverage!==null?` (${Math.round(totals.outCoverage*100)}%)`:''}</span>
        <span>{zh?'费用':'Cost'}: <strong>UNKNOWN</strong></span>
      </div>
    </Card>
    {groups.map(group=><Card key={group.agentId}>
      <CardHeader title={<code>{group.agentId}</code>}
        sub={zh?`${group.attempts.length} 次执行`:`${group.attempts.length} attempts`}/>
      <DataTable rows={group.attempts} rowKey={attempt=>attempt.attemptId}
        columns={[
          {key:'attempt',header:'Attempt',
            cell:attempt=><code>{attempt.attemptId}</code>},
          {key:'tasks',header:zh?'任务':'Tasks',
            cell:attempt=>attempt.taskIds.join(', ')||'—'},
          {key:'state',header:zh?'状态':'State',
            cell:attempt=><Badge tone={stateTone(attempt.state)}>{attempt.state}</Badge>},
          {key:'duration',header:zh?'时长':'Duration',
            cell:attempt=>durationLabel(attempt)},
          {key:'tokens',header:'Tokens',
            cell:attempt=>`${tokenLabel(attempt,'tokensIn')} / ${tokenLabel(attempt,'tokensOut')}`},
          {key:'model',header:'Model',cell:attempt=>attempt.model??'UNKNOWN'},
        ]}/>
    </Card>)}
  </div>;
}
