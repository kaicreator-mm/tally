import {useEffect,useState} from 'react';
import {useShell} from '../../app/shell/AppContext.tsx';
import {hostAdapter} from '../../adapters/host.ts';
import {EmptyState,Skeleton} from '../../components/ui/States.tsx';

type Source={id:string;label:string;sourceRef:string;fact?:{status:string;reason?:string}};
type Intake={
  status:string;sourceSha:string;
  sources:Source[];
  guided:{goals:string[];evidence:string[];decision:{status:string;reasonCode:string;step?:{stepId:string}|null}};
  draft?:{state:string;digest:string;product:{goal:string};l2:{decision:string|null}}|null;
  review?:{state:string;reviewer:string}|null;
  frozen?:{state:string;receiptRef:string}|null;
  constraints:{l2Choices:string[]};
};
function actionKey(){
  return new URLSearchParams(window.location.search).get('actionKey')??'';
}
async function loadIntake():Promise<Intake>{
  const result=await fetch('/api/intake/snapshot',{cache:'no-store'});
  if(!result.ok)throw Error('INTAKE_SNAPSHOT_UNAVAILABLE');
  const body=await result.json() as {snapshot:Intake};
  if(body.snapshot?.status!=='OBSERVED'||!/^[0-9a-f]{40}$/.test(body.snapshot.sourceSha))
    throw Error('INTAKE_SOURCE_UNQUALIFIED');
  return body.snapshot;
}
/** Only one existing native guided controller. No second Source/Task writer. */
export function ProductWorkspace(){
  const {projectId,versionId}=useShell();
  const [source,setSource]=useState<Intake|null>(null);
  const [loadError,setLoadError]=useState('');
  const [busy,setBusy]=useState(false);
  const [goal,setGoal]=useState('');
  const [evidence,setEvidence]=useState<string[]>([]);
  const [nonGoals,setNonGoals]=useState('');
  const [l2Decision,setL2Decision]=useState('');
  const [reviewer,setReviewer]=useState('');
  const [last,setLast]=useState('');
  const [sourceRepo,setSourceRepo]=useState<string|null>(null);
  const refresh=async()=>{
    const snapshot=await loadIntake();
    setSource(snapshot);
    setGoal(previous=>previous||snapshot.guided.goals[0]||'');
    setEvidence(previous=>previous.length?previous:snapshot.guided.evidence);
    setL2Decision(previous=>previous||snapshot.constraints.l2Choices[0]||'');
  };
  useEffect(()=>{
    let alive=true;
    void Promise.all([loadIntake(),hostAdapter.catalog()])
      .then(([snapshot,catalog])=>{
        if(!alive)return;
        setSource(snapshot);
        setSourceRepo(catalog.source?.repository??null);
        setGoal(snapshot.guided.goals[0]??'');
        setEvidence(snapshot.guided.evidence);
        setL2Decision(snapshot.constraints.l2Choices[0]??'');
      }).catch(error=>{if(alive)setLoadError(String(error));});
    return ()=>{alive=false;};
  },[]);
  const submit=async(kind:string,args:Record<string,unknown>={})=>{
    if(!source||busy||!actionKey())return;
    setBusy(true);setLast('');
    try{
      const response=await fetch('/api/intake/actions',{method:'POST',headers:{
        'content-type':'application/json','x-tally-action-key':actionKey()},
        body:JSON.stringify({kind,sourceSha:source.sourceSha,args})});
      const body=await response.json() as {status?:string;reason?:string};
      setLast(kind+' → '+(body.status??response.status)+
        (body.reason?' · '+body.reason:''));
      await refresh();
    }catch(error){
      setLast(kind+' → UNKNOWN · '+String(error));
    }finally{setBusy(false);}
  };
  if(loadError)return <EmptyState title="Product Host 未连接" hint={loadError}/>;
  if(!source)return <Skeleton lines={5}/>;
  const hasKey=Boolean(actionKey());
  const scoped=sourceRepo==='kaicreator-mm/tally';
  return <section aria-label="Product guided intake" style={{display:'grid',gap:16}}>
    <p>当前来源：<code>{source.sourceSha.slice(0,12)}</code>
      {' · '}引导状态：{source.guided.decision.status}
      {' · '}{source.guided.decision.reasonCode}</p>
    {!scoped?<p role="status">当前 Product 引导契约仅绑定 Tally 原始知识源；
      其他项目的通用需求建模 Host 尚未接入，不会把内置示例当作当前项目的正式 Product。</p>:null}
    {!hasKey?<p role="alert">缺少当前 ALM 会话操作凭据。请从项目配置页重新打开工作台。
      仅允许读取，不会尝试未授权写入。</p>:null}
    <fieldset disabled={busy||!hasKey||!scoped}
      style={{display:'grid',gap:12,padding:16}}>
      <legend>Product / Requirements → L2 草稿</legend>
      <label>产品目标
        <input aria-label="产品目标" value={goal} maxLength={256}
          onChange={event=>setGoal(event.target.value)} style={{display:'block',width:'100%'}}/>
      </label>
      <button type="button" onClick={()=>{void submit('submit-goal',{goal});}}>
        保存需求目标</button>
      <fieldset style={{display:'grid',gap:6}}>
        <legend>选择已有证据</legend>
        {source.sources.map(item=><label key={item.id}>
          <input type="checkbox" checked={evidence.includes(item.id)}
            onChange={event=>setEvidence(rows=>
              event.target.checked?[...new Set([...rows,item.id])]:
              rows.filter(id=>id!==item.id))}/>
          {' '}{item.label} <small>{item.fact?.status??'UNKNOWN'}</small>
        </label>)}
      </fieldset>
      <button type="button" onClick={()=>{void submit('select-evidence',{refs:evidence});}}>
        保存证据选择</button>
      <button type="button" onClick={()=>{void submit('confirm-inspect');}}>
        确认证据检查</button>
      <label>本版本不包含（每行一项）
        <textarea aria-label="非目标" value={nonGoals}
          onChange={event=>setNonGoals(event.target.value)} rows={3}
          style={{display:'block',width:'100%'}}/>
      </label>
      <label>L2 方案
        <select aria-label="L2 方案" value={l2Decision}
          onChange={event=>setL2Decision(event.target.value)}>
          {source.constraints.l2Choices.map(option=>
            <option key={option} value={option}>{option}</option>)}
        </select>
      </label>
      <button type="button" onClick={()=>{void submit('draft-product',{
        goal,nonGoals:nonGoals.split('\n').map(row=>row.trim()).filter(Boolean),l2Decision});}}>
        生成 Product / L2 草稿</button>
      <label>评审者（提议，不代表独立审核通过）
        <input aria-label="评审者" value={reviewer}
          onChange={event=>setReviewer(event.target.value)}/>
      </label>
      <button type="button" disabled={!source.draft||busy} onClick={()=>{
        void submit('review-draft',{reviewer});}}>记录评审提议</button>
    </fieldset>
    {last?<p role="status" aria-live="polite">{last}</p>:null}
    {source.draft?<article>
      <h3>当前 Product 草稿（尚未冻结）</h3>
      <p>{source.draft.product.goal}</p>
      <p>方案：{source.draft.l2.decision??'未选择'}
        {' · '}摘要：<code>{source.draft.digest.slice(0,16)}</code></p>
      <p>Review：{source.review?.state??'NOT_RUN'}
        {' · '}Freeze：{source.frozen?.state??'NOT_RUN'}</p>
      <p>正式冻结仍需真实 Host / Review Witness；Mock 不会提升为已冻结。</p>
    </article>:null}
    {projectId&&versionId?
      <p><a href={`#/development/tasks?projectId=${encodeURIComponent(projectId)}&versionId=${encodeURIComponent(versionId)}`}>
        前往 Task DAG / Task 工作台 →</a></p>
      :<p>请先完成实际 Project 和 Version 选择，再进入 Task。</p>}
  </section>;
}
