/** Source-qualified live guided step route; never submits without selected Host. */
import {useMemo,useState} from 'react';
import {useHost} from '../../adapters/useHost.ts';
import type {GuidedStepResponse} from '../../adapters/host.ts';
import {hostAdapter} from '../../adapters/host.ts';
import {EmptyState,Skeleton} from '../../components/ui/States.tsx';
import {DynamicForm,DEFAULT_STRINGS_ZH,type RendererHost} from './renderer.tsx';

export function GuidedHostRoute({projectId,versionId}:{
  projectId:string|null;versionId:string|null}){
  const [outcome,setOutcome]=useState<'completed'|'cancelled'|null>(null);
  const response=useHost<GuidedStepResponse>('guided-step',
    adapter=>adapter.guidedStep(projectId??'',versionId??''),
    [projectId,versionId]);
  const snapshot=response.data;
  const storageKey='tally.v2.guided.draft.'+
    encodeURIComponent(projectId??'none')+'.'+encodeURIComponent(versionId??'none');
  const host=useMemo<RendererHost>(()=>{
    // Only one browser session's scoped, non-secret draft answers are stored.
    return {
      locale:'zh',strings:DEFAULT_STRINGS_ZH,
      loadDraft:()=>window.sessionStorage.getItem(storageKey),
      saveDraft:raw=>window.sessionStorage.setItem(storageKey,raw),
      clearDraft:()=>window.sessionStorage.removeItem(storageKey),
      submitStep:async payload=>{
        if(!projectId||!versionId||!snapshot?.sourceSha||!snapshot?.csrfKey)
          return {ok:false,reason:'GUIDED_HOST_NOT_CONNECTED'};
        try{
          const result=await hostAdapter.guidedSubmit({
            projectId,versionId,sourceSha:snapshot.sourceSha},snapshot.csrfKey,payload);
          if(result.ok!==true)return {ok:false,
            reason:result.reason??'GUIDED_HOST_DENIED'};
          return {ok:true,nextStep:result.nextStep??null};
        }catch{return {ok:false,reason:'GUIDED_HOST_TRANSPORT_FAILED'};}
      },
    };
  },[projectId,versionId,snapshot?.sourceSha,snapshot?.csrfKey,storageKey]);
  if(!projectId||!versionId)return <EmptyState title="请先在右上角选择项目与版本"/>;
  if(response.state==='loading')return <Skeleton lines={5}/>;
  if(response.state==='not_connected'||snapshot?.connected!==true||!snapshot.step)
    return <EmptyState title="NOT_CONNECTED"
      hint="当前版本的动态表单需由选中的 Business/Host 下发；缺少真实 StepSpec 时不构造示例审批表单。"/>;
  if(outcome)return <div role="status">
    {outcome==='completed'?'已提交当前引导步骤；不代表 Product Freeze 或 Task PASS。':
      '本次表单已取消；未申请任何业务执行权限。'}
    <button type="button" onClick={()=>{setOutcome(null);void response.refresh();}}>
      重新读取 Host 步骤</button>
  </div>;
  return <DynamicForm key={projectId+'/'+versionId+'/'+snapshot.step.stepId+'/'+snapshot.step.revision}
    initialStep={snapshot.step} host={host} onDone={setOutcome}/>;
}
