/** UI-D #231 — defects, next-version candidates and measured learning. */
import {dedupeDefects,suggestRouting,learningDisplay} from './model.ts';
import type {DefectRecord,LearningRecord} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {DataTable} from '../../components/ui/DataTable.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

export function MaintenanceView({defects,learnings,connected=true,locale='zh'}:
  {defects:readonly DefectRecord[];learnings:readonly LearningRecord[];
   connected?:boolean;locale?:'zh'|'en'}){
  const zh=locale==='zh';
  if(!connected)return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
    hint="UI-E #232"/>;
  const {unique,duplicates}=dedupeDefects(defects);
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?'缺陷与去向':'Defects & routing'}
        sub={zh?`唯一 ${unique.length} · 疑似重复 ${duplicates.length}`
          :`${unique.length} unique / ${duplicates.length} duplicates`}/>
      {unique.length===0?<EmptyState title={zh?'暂无缺陷':'No defects'}/>:
        <DataTable rows={unique} rowKey={defect=>defect.id}
          columns={[
            {key:'id',header:'ID',cell:defect=><code>{defect.id}</code>},
            {key:'summary',header:zh?'描述':'Summary',cell:defect=>defect.summary},
            {key:'state',header:zh?'状态':'State',
              cell:defect=><Badge tone={defect.state==='REPAIRED_IN_VERSION'?'ok'
                :defect.state==='DEFERRED_NEXT_VERSION'?'info':'warn'}>
                {defect.state}</Badge>},
            {key:'route',header:zh?'建议去向':'Suggested routing',
              cell:defect=>{const routing=suggestRouting(defect);
                return <span>{zh?(routing.routedTo==='repair'?'本版 Repair':'下一版本')
                  :routing.routedTo}<br/>
                  <span style={{color:'var(--text-muted)',fontSize:12}}>
                    {zh?routing.reason:''}</span></span>;}},
          ]}/>}
    </Card>
    <Card>
      <CardHeader title={zh?'经验与改进(实测,不推断)':'Measured learning'}
        sub={zh?'仅展示经独立见证的效果;未测就是未测':'attested only'}/>
      {learnings.length===0?<EmptyState title={zh?'暂无经验记录':'No learnings'}/>:
        <DataTable rows={learnings} rowKey={learning=>learning.id}
          columns={[
            {key:'id',header:'ID',cell:learning=><code>{learning.id}</code>},
            {key:'decision',header:zh?'决策':'Decision',
              cell:learning=><Badge tone={learning.decision==='ADOPTED'?'ok':'neutral'}>
                {learning.decision}</Badge>},
            {key:'effect',header:zh?'实测效果':'Measured effect',
              cell:learning=>{const display=learningDisplay(learning);
                return <Badge tone={display.effect==='ATTESTED'?'ok':'warn'}>
                  {display.effect}</Badge>;}},
          ]}/>}
    </Card>
  </div>;
}
