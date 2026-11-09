/** UI #252 — governed FormStep envelope validation, not an execution engine.
 * Only selected Host/K performs dispatch; browser input is never grant evidence.
 */
const SHA=/^[0-9a-f]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const valueOk=(value)=>value===null||typeof value==='boolean'||
  (typeof value==='string'&&value.length<=4096&&!/[\u0000-\u0008\u000B-\u001F]/.test(value))||
  (Array.isArray(value)&&value.length<=32&&value.every(v=>typeof v==='string'&&v.length<=128));
const deny=(reason)=>({connected:false,status:'DENIED',reason});
const plain=(v)=>!!v&&typeof v==='object'&&!Array.isArray(v);
export function verifyGuidedSnapshot(snapshot,scope,currentSha){
  if(!plain(snapshot)||snapshot.connected!==true||
     !SHA.test(currentSha??'')||snapshot.sourceSha!==currentSha||
     !REPO.test(scope?.projectId??'')||snapshot.projectId!==scope.projectId||
     typeof scope.versionId!=='string'||!scope.versionId||
     snapshot.versionId!==scope.versionId||
     snapshot.currentness!=='CURRENT'||snapshot.readback!=='VERIFIED')return false;
  const step=snapshot.step;
  if(!plain(step)||!ID.test(step.journeyId)||!ID.test(step.stepId)||
    !Number.isSafeInteger(step.revision)||step.revision<1||
    !plain(step.title)||typeof step.title.zh!=='string'||
    typeof step.title.en!=='string'||!Array.isArray(step.fields)||
    step.fields.length<1||step.fields.length>24||
    !Array.isArray(step.actions)||!step.actions.includes('next'))return false;
  const fields=new Set();
  for(const f of step.fields){
    if(!plain(f)||!ID.test(f.id)||fields.has(f.id)||typeof f.required!=='boolean'||
      !['text','textArea','choice','multi','boolean','number','confirm',
        'choiceWithOther','multiLimited'].includes(f.kind))return false;
    fields.add(f.id);
  }
  return true;
}
export function verifyGuidedPayload(payload,snapshot){
  if(!plain(payload)||Object.keys(payload).sort().join(',')!==
    'answers,journeyId,revision,sourceSha,stepId')return false;
  const step=snapshot?.step;
  if(!step||payload.sourceSha!==snapshot.sourceSha||
    payload.journeyId!==step.journeyId||payload.stepId!==step.stepId||
    payload.revision!==step.revision||!plain(payload.answers))return false;
  const allowed=new Map(step.fields.map(f=>[f.id,f]));
  if(Object.keys(payload.answers).length>24)return false;
  for(const [key,value] of Object.entries(payload.answers)){
    const field=allowed.get(key);
    if(!field||!valueOk(value))return false;
    if((['multi','multiLimited'].includes(field.kind)&&!Array.isArray(value))||
      (['boolean','confirm'].includes(field.kind)&&typeof value!=='boolean')||
      (['text','textArea','choice','choiceWithOther','number'].includes(field.kind)&&
        typeof value!=='string'))return false;
  }
  return true;
}
export async function readGuidedStep(host,scope,currentSha){
  if(typeof host?.readCurrentStep!=='function')
    return deny('GUIDED_HOST_NOT_CONNECTED');
  try{
    const snapshot=await host.readCurrentStep({...scope,sourceSha:currentSha});
    if(!verifyGuidedSnapshot(snapshot,scope,currentSha))
      return deny('GUIDED_SOURCE_UNQUALIFIED');
    // Never trust the form to grant an action, even if its source looks current.
    return {connected:true,sourceSha:currentSha,step:snapshot.step};
  }catch{return deny('GUIDED_HOST_READ_FAILED');}
}
export async function submitGuidedStep(host,scope,currentSha,payload){
  const read=await readGuidedStep(host,scope,currentSha);
  if(!read.connected)return read;
  if(!verifyGuidedPayload(payload,read))
    return deny('GUIDED_STEP_STALE_OR_INVALID');
  if(typeof host?.dispatchStep!=='function')
    return deny('GUIDED_HOST_DISPATCH_UNAVAILABLE');
  try{
    const result=await host.dispatchStep({
      ...scope,sourceSha:currentSha,journeyId:payload.journeyId,
      stepId:payload.stepId,revision:payload.revision,answers:payload.answers});
    if(!plain(result)||result.ok!==true||
      (result.nextStep!==null&&
       !verifyGuidedSnapshot({...read,step:result.nextStep,projectId:scope.projectId,
         versionId:scope.versionId,currentness:'CURRENT',readback:'VERIFIED'},
         scope,currentSha)))
      return deny('GUIDED_HOST_RESULT_UNQUALIFIED');
    return {connected:true,ok:true,nextStep:result.nextStep};
  }catch{return deny('GUIDED_HOST_DISPATCH_FAILED');}
}
