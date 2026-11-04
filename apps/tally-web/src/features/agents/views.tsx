/** B/#261 — read-only task execution history; users dispatch and coordinate in Tally. */
import type {AttemptRecord} from './model.ts';
import {durationLabel,tokenLabel,telemetryCoverage,
  tokenTotals,stateTone,sanitizeAttempts} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {DataTable} from '../../components/ui/DataTable.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

export function AgentsPanel({attempts,locale='zh',connected=false}:
  {attempts:readonly AttemptRecord[];locale?:'zh'|'en';connected?:boolean}){
  const zh=locale==='zh';
  // Fail-closed: a missing connection flag means DISCONNECTED, and records are
  // bounded/secret-redacted at this projection (never trusted from upstream).
  const records=sanitizeAttempts(attempts);
  if(!connected){
    return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
      hint={zh?'等待 UI-E #232 接入 Host 读模型':'Host read model arrives with UI-E #232'}/>;
  }
  if(records.length===0)
    return <EmptyState title={zh?'执行记录状态未知(UNKNOWN)':'Execution records UNKNOWN'}
      hint={zh?'数据源已连接但未返回经过验证的执行记录，不视为已确认的空历史':
        'Source connected but returned no verified records; not a confirmed empty history'}/>;
  const coverage=telemetryCoverage(records);
  const totals=tokenTotals(records);
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?'遥测覆盖(缺失 ≠ 0)':'Telemetry coverage (missing ≠ 0)'}
        sub={zh?'仅统计有真实数据的尝试;UNKNOWN 保持 UNKNOWN':'Only measured attempts count; UNKNOWN stays UNKNOWN'}/>
      <div style={{display:'flex',gap:'var(--s4)',flexWrap:'wrap',fontSize:13}}>
        <span>{zh?'时长可测':'Duration measured'}: <strong>{coverage.measured}/{coverage.total}</strong>
          {coverage.coverage!==null?' ('+Math.round(coverage.coverage*100)+'%)':''}</span>
        <span>{zh?'tokens 入(可测)':'tokens in (measured)'}:
          <strong> {totals.tokensIn}</strong>
          {totals.inCoverage!==null?' ('+Math.round(totals.inCoverage*100)+'%)':''}</span>
        <span>{zh?'tokens 出(可测)':'tokens out (measured)'}:
          <strong> {totals.tokensOut}</strong>
          {totals.outCoverage!==null?' ('+Math.round(totals.outCoverage*100)+'%)':''}</span>
        <span>{zh?'费用':'Cost'}: <strong>UNKNOWN</strong></span>
      </div>
    </Card>
    <Card>
      <CardHeader title={zh?'任务执行记录':'Task execution records'}
        sub={zh?'按任务与执行尝试展示；所有调度与操作均由 Tally 管理':'Task-centric, read-only execution history; dispatch remains Tally-managed'}/>
      <DataTable rows={records} rowKey={attempt=>attempt.attemptId}
        columns={[
          {key:'tasks',header:zh?'任务':'Tasks',
            cell:attempt=>attempt.taskIds.join(', ')||'—'},
          {key:'attempt',header:'Attempt',
            cell:attempt=><code>{attempt.attemptId}</code>},
          {key:'provider',header:zh?'执行方 / 角色':'Provider / role',
            cell:attempt=>(attempt.provider??'UNKNOWN')+' · '+(attempt.role??'UNKNOWN')},
          {key:'state',header:zh?'状态':'State',
            cell:attempt=><Badge tone={stateTone(attempt.state)}>{attempt.state}</Badge>},
          {key:'duration',header:zh?'时长':'Duration',
            cell:attempt=>durationLabel(attempt)},
          {key:'tokens',header:'Tokens',
            cell:attempt=>tokenLabel(attempt,'tokensIn')+' / '+tokenLabel(attempt,'tokensOut')},
          {key:'model',header:'Model',cell:attempt=>attempt.model??'UNKNOWN'},
        ]}/>
    </Card>
  </div>;
}
