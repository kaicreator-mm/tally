/** UI-F #233 — engineering metadata center views (tabs: 报告/用量, 溯源, 质量与学习). */
import {aggregateUsage,reportRows,missingReports,casebookLinks} from './model.ts';
import type {VersionMetadata,MetadataTab} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Tabs} from '../../components/ui/Tabs.tsx';
import {DataTable} from '../../components/ui/DataTable.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const pct=(value:number|null)=>value===null?'UNKNOWN'
  :`${Math.round(value*100)}%`;
export function MetadataCenter({version,allVersions,expectedTaskIds,
  connected=true,locale='zh',onOpenTask}:
  {version:VersionMetadata|null;allVersions:readonly VersionMetadata[];
   expectedTaskIds:readonly string[]|null;connected?:boolean;locale?:'zh'|'en';
   onOpenTask?:(taskId:string)=>void}){
  const zh=locale==='zh';
  if(!connected)return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
    hint="UI-E #232"/>;
  if(!version)return <EmptyState
    title={zh?'请先在右上角选择项目与版本':'Select project & version first'}/>;
  const usage=aggregateUsage(version);
  const measured=version.telemetryCoverage==='VERIFIED';
  const reports=reportRows(version);
  const missing=missingReports(version,expectedTaskIds);
  const tabs:readonly {key:MetadataTab;label:string;content:React.ReactNode}[]=[
    {key:'reports',label:zh?'报告 / 用量':'Reports / Usage',
      content:<div style={{display:'grid',gap:'var(--s5)'}}>
        <Card>
          <CardHeader title={zh?'用量汇总(含覆盖率)':'Usage (with coverage)'}
            sub={zh?'缺失遥测不计为 0;跨版本不合并':'missing ≠ zero; no cross-version merge'}/>
          <dl style={{display:'grid',gridTemplateColumns:'auto auto auto 1fr',
            gap:'8px 14px',margin:0,fontSize:13}}>
            <dt style={{color:'var(--text-muted)'}}>{zh?'尝试':'Attempts'}</dt>
            <dd style={{margin:0}}>{measured?usage.attempts:'UNKNOWN'}</dd><span/>
            <span style={{color:'var(--text-muted)'}}>
              {(measured?Object.entries(usage.byRole):[]).map(([role,entry])=>
                `${role}: ${entry.measured}/${entry.count}`).join(' · ')}</span>
            <dt style={{color:'var(--text-muted)'}}>{zh?'时长(可测)':'Duration (measured)'}</dt>
            <dd style={{margin:0}}>{measured?Math.round(usage.measuredDuration/1000)+'s':'UNKNOWN'}</dd>
            <dd style={{margin:0,color:'var(--text-faint)'}}>{pct(measured?usage.durationCoverage:null)}</dd><span/>
            <dt style={{color:'var(--text-muted)'}}>tokens 入</dt>
            <dd style={{margin:0}}>{measured?usage.tokensIn:'UNKNOWN'}</dd>
            <dd style={{margin:0,color:'var(--text-faint)'}}>{pct(measured?usage.tokenInCoverage:null)}</dd><span/>
            <dt style={{color:'var(--text-muted)'}}>tokens 出</dt>
            <dd style={{margin:0}}>{measured?usage.tokensOut:'UNKNOWN'}</dd>
            <dd style={{margin:0,color:'var(--text-faint)'}}>{pct(measured?usage.tokenOutCoverage:null)}</dd><span/>
            <dt style={{color:'var(--text-muted)'}}>{zh?'费用':'Cost'}</dt>
            <dd style={{margin:0}}><Badge tone="warn">UNKNOWN</Badge></dd><span/><span/>
          </dl>
        </Card>
        <Card>
          <CardHeader title={zh?'ADS / task-meta 报告':'ADS / task-meta reports'}
            sub={missing===null?(zh?'ROSTER_COVERAGE=UNVERIFIED · 报告覆盖率未知':'ROSTER_COVERAGE=UNVERIFIED · report coverage UNKNOWN')
              :zh?`缺失 ADS v5 报告:${missing.length===0?'无':missing.length}`
                :`${missing.length} missing`}/>
          {reports.length===0?<EmptyState title={zh?'暂无报告':'No reports'}/>:
            <DataTable rows={reports} rowKey={row=>row.reportId}
              columns={[
                {key:'id',header:'Report',cell:row=><code>{row.reportId}</code>},
                {key:'task',header:zh?'任务':'Task',
                  cell:row=><button onClick={()=>onOpenTask?.(row.taskId)}
                    style={{border:0,background:'transparent',color:'var(--brand-strong)',
                      cursor:'pointer',font:'inherit',padding:0}}>{row.taskId}</button>},
                {key:'kind',header:'Kind',cell:row=>row.kind},
                {key:'reviewed',header:zh?'审查':'Review',
                  cell:row=><Badge tone={row.reviewed==='REVIEWED'?'ok':'warn'}>
                    {row.reviewed}</Badge>},
              ]}/>}
        </Card>
      </div>},
    {key:'provenance',label:zh?'溯源':'Provenance',
      content:<Card>
        <CardHeader title={zh?'Claim / 冻结 / PR / CI / 发布部署链路'
          :'Claim / freeze / PR / CI / release lineage'}/>
        {version.lineage.length===0?<EmptyState title={version.lineageCoverage==='VERIFIED'?(zh?'暂无溯源':'No lineage'):'LINEAGE_COVERAGE=UNKNOWN'}/>: 
          <DataTable rows={version.lineage} rowKey={row=>row.taskId}
            columns={[
              {key:'task',header:zh?'任务':'Task',cell:row=><code>{row.taskId}</code>},
              {key:'claim',header:'Claim',
                cell:row=><Badge tone={row.claimAccepted?'ok':'warn'}>
                  {row.claimAccepted?'ACCEPTED':'NOT_VERIFIED'}</Badge>},
              {key:'freeze',header:zh?'冻结':'Freeze',
                cell:row=>`P${row.productFreezeRev??'—'}/L${row.l2FreezeRev??'—'}`},
              {key:'pr',header:'PR',
                cell:row=>row.prNumber?`#${row.prNumber} @ ${(row.prHeadSha??'').slice(0,8)}`:'—'},
              {key:'ci',header:'CI',
                cell:row=><Badge tone={row.ciState==='SUCCESS'?'ok'
                  :row.ciState==='FAILURE'?'danger':'neutral'}>{row.ciState}</Badge>},
              {key:'fhc',header:zh?'全/隐/关':'F/H/C',
                cell:row=>`${row.fullHiddenCritical.full.slice(0,4)}
                  /${row.fullHiddenCritical.hidden.slice(0,4)}
                  /${row.fullHiddenCritical.critical.slice(0,4)}`},
              {key:'receipts',header:zh?'发布/部署':'Release/Deploy',
                cell:row=>`${row.releaseReceiptRef?'✓':'—'}/${row.deploymentReceiptRef?'✓':'—'}`},
            ]}/>}
      </Card>},
    {key:'quality',label:zh?'质量与学习':'Quality & learning',
      content:<Card>
        <CardHeader title={zh?'跨版本经验与实测结果':'Cross-version learning'}
          sub={zh?'实测效果为独立见证;未测即未测,无 Agent 排名':'attested only; no ranking'}/>
        {version.learnedOutcomes.length===0?
          <EmptyState title={version.learningCoverage==='VERIFIED'?(zh?'暂无经验记录':'No learnings'):'LEARNING_COVERAGE=UNKNOWN'}/>:
          <DataTable rows={version.learnedOutcomes} rowKey={row=>row.learningId}
            columns={[
              {key:'id',header:'ID',cell:row=><code>{row.learningId}</code>},
              {key:'task',header:zh?'任务':'Task',
                cell:row=><span>{row.taskId}
                  {casebookLinks(allVersions,row.taskId).length>1?
                    <Badge tone="info">{zh?'跨版本':'cross-version'}</Badge>:null}</span>},
              {key:'decision',header:zh?'决策':'Decision',
                cell:row=><Badge tone={row.decision==='ADOPTED'?'ok':'neutral'}>
                  {row.decision}</Badge>},
              {key:'effect',header:zh?'实测':'Measured',
                cell:row=><Badge tone={row.attested===true?'ok':'warn'}>
                  {row.attested===true?'ATTESTED':row.attested===false?'NOT_MEASURED':'UNKNOWN'}</Badge>},
            ]}/>}
      </Card>},
  ];
  return <Tabs items={tabs} ariaLabel={zh?'元数据':'metadata'}/>;
}
