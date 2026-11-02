import type {ExperienceSnapshot,ExperienceDecisionReceipt} from './model.ts';
import {eligibleExperienceDecision,experienceOutcomeLabel} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Button} from '../../components/ui/Button.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {EmptyState} from '../../components/ui/States.tsx';
export function ExperienceView({snapshot,projectId,versionId,onDecision,
  locale='zh'}:{snapshot:ExperienceSnapshot|null;projectId:string;
  versionId:string;onDecision:(id:string,revision:number,
    decision:'ADOPT'|'REJECT')=>Promise<ExperienceDecisionReceipt>|void;
  locale?:'zh'|'en'}){
  const zh=locale==='zh';
  if(!snapshot?.connected||snapshot.source?.repository!==projectId||
     snapshot.versionId!==versionId||snapshot.source.currentness!=='CURRENT'||
     snapshot.source.readback!=='VERIFIED'||!Array.isArray(snapshot.candidates))
    return <EmptyState title="NOT_CONNECTED"
      hint={zh?'经验候选需要当前版本的 ADS/Host 来源证据':'Current ADS/Host source evidence is required'}/>;
  if(snapshot.candidates.length===0)
    return <EmptyState title={zh?'暂无经验候选':'No experience candidates'}
      hint={zh?'没有候选不代表已证明质量提升':'No candidate is not evidence of quality improvement'}/>;
  return <div style={{display:'grid',gap:'var(--s4)'}}>
    <Card>
      <CardHeader title={zh?'工程经验与规则':'Engineering experience'}
        sub={zh?'每条经验都必须经过来源验证和人工决定；下一 Task 是否使用另行取证'
          :'Evidence-backed candidates require an explicit human decision; next-task usage is separately observed'}/>
      <div style={{display:'flex',gap:8,flexWrap:'wrap'}}>
        <Badge>{snapshot.candidates.length} {zh?'条候选':'candidates'}</Badge>
        <Badge tone="neutral">QUALITY_GAIN=UNKNOWN</Badge>
        <Badge tone="neutral">NEXT_TASK_USAGE=NOT_VERIFIED</Badge>
      </div>
    </Card>
    {snapshot.candidates.map(candidate=>{
      const gate=eligibleExperienceDecision(snapshot,candidate,projectId,versionId);
      const terminal=candidate.state==='ADOPTED'||candidate.state==='REJECTED';
      return <Card key={candidate.id}>
        <CardHeader title={candidate.title}
          sub={candidate.id.slice(0,12)+' · revision '+candidate.revision}/>
        <div style={{display:'flex',gap:8,flexWrap:'wrap',marginBottom:10}}>
          <Badge tone={candidate.state==='ADOPTED'?'ok':
            candidate.state==='REJECTED'?'neutral':'warn'}>{candidate.state}</Badge>
          <Badge tone={candidate.reportQualification==='VERIFIED'?'ok':'warn'}>
            {candidate.reportQualification}</Badge>
          <Badge tone="neutral">{experienceOutcomeLabel(candidate)}</Badge>
        </div>
        <a href={candidate.sourceRef} target="_blank" rel="noopener noreferrer">
          {zh?'查看 ADS 原始报告':'Open ADS source report'}
        </a>
        {!terminal?<div style={{display:'flex',gap:8,marginTop:12,flexWrap:'wrap'}}>
          <Button disabled={!gate.eligible}
            onClick={()=>{void onDecision(candidate.id,candidate.revision,'ADOPT');}}>
            {zh?'采纳为后续经验':'Adopt for future Tasks'}
          </Button>
          <Button variant="secondary" disabled={!gate.eligible}
            onClick={()=>{void onDecision(candidate.id,candidate.revision,'REJECT');}}>
            {zh?'拒绝':'Reject'}
          </Button>
          {!gate.eligible?<span role="status" style={{fontSize:12,color:'var(--text-muted)'}}>
            {gate.reason}</span>:null}
        </div>:null}
      </Card>;
    })}
  </div>;
}
