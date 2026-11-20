/** UI-D #231 — QA test plan, layered verification and installed acceptance. */
import {qaSummary,layerTone} from './model.ts';
import type {QaRecord,VerificationLayer} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Collapse} from '../../components/ui/Collapse.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const LAYER_LABEL:Record<VerificationLayer,{zh:string;en:string}>={
  fullRegression:{zh:'全量回归',en:'Full regression'},
  hiddenValidation:{zh:'隐藏验证(需独立见证)',en:'Hidden (independent)'},
  criticalJourneys:{zh:'关键旅程',en:'Critical journeys'},
  installedAcceptance:{zh:'安装用户验收(需独立见证)',en:'Installed acceptance'},
};
export function QaView({records,connected=true,locale='zh'}:
  {records:readonly QaRecord[];connected?:boolean;locale?:'zh'|'en'}){
  const zh=locale==='zh';
  if(!connected)return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
    hint="UI-E #232"/>;
  const summary=qaSummary(records);
  const byLayer=new Map(records.map(record=>[record.layer,record]));
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?'质量结论':'QA verdict'}
        sub={summary.ready?(zh?'四层全部满足且见证完整':'All layers verified')
          :(zh?`仍有 ${summary.totalGaps} 项缺口`:`${summary.totalGaps} gaps`)}/>
      {summary.ready
        ?<Badge tone="ok">{zh?'可进入发布资格':'Release-qualifiable'}</Badge>
        :<ul style={{margin:0,paddingLeft:18}}>
          {summary.top.map(gap=><li key={gap.code}>
            <code>{gap.code}</code> — {gap.detail}</li>)}
        </ul>}
    </Card>
    <Collapse title={zh?'完整四层矩阵':'Full matrix'} defaultOpen={true}>
      <table style={{width:'100%',borderCollapse:'collapse',fontSize:13}}>
        <tbody>
          {(Object.keys(LAYER_LABEL) as VerificationLayer[]).map(layer=>{
            const record=byLayer.get(layer);
            return <tr key={layer}>
              <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
                {zh?LAYER_LABEL[layer].zh:LAYER_LABEL[layer].en}</td>
              <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
                <Badge tone={layerTone(record)}>
                  {record?`${record.outcome}/${record.provenance}`:'NOT_RUN'}</Badge>
              </td>
              <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)',
                color:'var(--text-muted)'}}>
                {record?.witness??'—'}</td>
            </tr>;})}
        </tbody>
      </table>
    </Collapse>
  </div>;
}
