/** UI #252 — governed FormStep envelope validation, not an execution engine.
 * Only selected Host/K performs dispatch; browser input is never grant evidence.
 */
const SHA=/^[0-9a-f]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/;
const titleOk=value=>typeof value==='string'&&value.trim().length>0&&
  value.length<=120&&!/[\u0000-\u001f\u007f]/.test(value);
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const valueOk=(value)=>value===null||typeof value==='boolean'||
  (typeof value==='string'&&value.length<=4096&&!/[\u0000-\u0008\u000B-\u001F]/.test(value))||
  (Array.isArray(value)&&value.length<=32&&value.every(v=>typeof v==='string'&&
    v.length>0&&v.length<=128&&!/[\u0000-\u001f\u007f]/.test(v)));
const deny=(reason)=>({connected:false,status:'DENIED',reason});
const plain=(v)=>!!v&&typeof v==='object'&&!Array.isArray(v);
const validId=value=>typeof value==='string'&&ID.test(value);
const localized=value=>plain(value)&&titleOk(value.zh)&&titleOk(value.en);
const copyLocalized=value=>({zh:value.zh,en:value.en});
const optionalLocalized=(owner,key)=>owner[key]===undefined||owner[key]===null||
  localized(owner[key]);
const integer=(value,min,max)=>Number.isSafeInteger(value)&&value>=min&&value<=max;
function projectOptions(options){
  if(!Array.isArray(options)||options.length<1||options.length>32)return null;
  const seen=new Set();
  const result=[];
  for(const option of options){
    if(!plain(option)||!validId(option.id)||seen.has(option.id)||
       !localized(option.label)||typeof option.eligible!=='boolean'||
       !optionalLocalized(option,'ineligibleReason')||
       (option.recommended!==undefined&&typeof option.recommended!=='boolean'))return null;
    seen.add(option.id);
    const projected={id:option.id,label:copyLocalized(option.label),
      eligible:option.eligible};
    if(option.ineligibleReason!==undefined)
      projected.ineligibleReason=option.ineligibleReason===null?null:
        copyLocalized(option.ineligibleReason);
    if(option.recommended!==undefined)projected.recommended=option.recommended;
    result.push(projected);
  }
  return result;
}
function projectField(field){
  if(!plain(field)||!validId(field.id)||!localized(field.label)||
     typeof field.required!=='boolean'||
     !optionalLocalized(field,'hint')||
     !optionalLocalized(field,'disabledReason'))return null;
  const projected={id:field.id,kind:field.kind,
    label:copyLocalized(field.label),required:field.required};
  for(const key of ['hint','disabledReason'])
    if(field[key]!==undefined)
      projected[key]=field[key]===null?null:copyLocalized(field[key]);
  if(field.visibleWhen!==undefined){
    const rule=field.visibleWhen;
    if(rule!==null&&(!plain(rule)||!validId(rule.fieldId)||
      !Object.prototype.hasOwnProperty.call(rule,'equals')||
      !(rule.equals===null||typeof rule.equals==='boolean'||
        typeof rule.equals==='string'&&rule.equals.length<=4096)))return null;
    projected.visibleWhen=rule===null?null:
      {fieldId:rule.fieldId,equals:rule.equals};
  }
  switch(field.kind){
    case 'text':case 'textArea':
      if(!integer(field.maxLength,1,4000))return null;
      projected.maxLength=field.maxLength;break;
    case 'choice':case 'multi':case 'choiceWithOther':case 'multiLimited':{
      const options=projectOptions(field.options);
      if(!options)return null;
      projected.options=options;
      if(field.kind==='choiceWithOther'){
        if(!integer(field.otherMaxLength,1,500))return null;
        projected.otherMaxLength=field.otherMaxLength;
      }
      if(field.kind==='multiLimited'){
        if(!integer(field.minSelected,0,32)||
           !integer(field.maxSelected,field.minSelected,32))return null;
        projected.minSelected=field.minSelected;
        projected.maxSelected=field.maxSelected;
      }
      break;
    }
    case 'number':
      if(typeof field.min!=='number'||typeof field.max!=='number'||
         !Number.isFinite(field.min)||!Number.isFinite(field.max)||
         field.min>=field.max||typeof field.integerOnly!=='boolean')return null;
      projected.min=field.min;projected.max=field.max;
      projected.integerOnly=field.integerOnly;break;
    case 'confirm':
      if(!localized(field.statement))return null;
      projected.statement=copyLocalized(field.statement);break;
    case 'boolean':break;
    default:return null;
  }
  return projected;
}
function projectStep(step){
  if(!plain(step)||!validId(step.journeyId)||!validId(step.stepId)||
     !integer(step.revision,1,Number.MAX_SAFE_INTEGER)||
     !localized(step.title)||!optionalLocalized(step,'intro')||
     !Array.isArray(step.fields)||step.fields.length<1||step.fields.length>24||
     !Array.isArray(step.actions)||step.actions.length<1||
     step.actions.length>5)return null;
  const allowedActions=new Set(['next','back','pause','resume','cancel']);
  if(!step.actions.includes('next')||step.actions.some(action=>!allowedActions.has(action))||
     new Set(step.actions).size!==step.actions.length)return null;
  const fields=step.fields.map(projectField);
  if(fields.includes(null)||new Set(fields.map(field=>field.id)).size!==fields.length)
    return null;
  const projected={journeyId:step.journeyId,stepId:step.stepId,
    revision:step.revision,title:copyLocalized(step.title),fields,
    actions:[...step.actions]};
  if(step.intro!==undefined)
    projected.intro=step.intro===null?null:copyLocalized(step.intro);
  return projected;
}
/** Structural canonical form: object keys are sorted recursively, so two
 * semantically identical snapshots compare equal regardless of the Host's
 * property insertion order (JSON.stringify is order-sensitive). */
const canonicalJson=(value)=>{
  if(Array.isArray(value))return '['+value.map(canonicalJson).join(',')+']';
  if(value!==null&&typeof value==='object')
    return '{'+Object.keys(value).sort().map(key=>
      JSON.stringify(key)+':'+canonicalJson(value[key])).join(',')+'}';
  return JSON.stringify(value);
};
export function verifyGuidedSnapshot(snapshot,scope,currentSha){
  if(!plain(snapshot)||snapshot.connected!==true||
     !SHA.test(currentSha??'')||snapshot.sourceSha!==currentSha||
     !REPO.test(scope?.projectId??'')||snapshot.projectId!==scope.projectId||
     typeof scope.versionId!=='string'||!scope.versionId||
     snapshot.versionId!==scope.versionId||
     snapshot.currentness!=='CURRENT'||snapshot.readback!=='VERIFIED')return false;
  return projectStep(snapshot.step)!==null;
}
/** Validate values against the *issued* Business field constraints before any
 * Host dispatch; frontend convenience validation is not sufficient authority. */
function answerMatchesIssuedField(field,value){
  if(value===null)
    return !field.required&&(field.kind!=='multiLimited'||field.minSelected===0);
  if(value===''&&!field.required&&
     ['text','textArea','choice','choiceWithOther','number'].includes(field.kind))
    return true;
  const selectedOption=id=>typeof id==='string'&&
    Array.isArray(field.options)&&
    field.options.some(option=>option.id===id&&option.eligible===true);
  switch(field.kind){
    case 'text':case 'textArea':
      return typeof value==='string'&&value.length<=field.maxLength;
    case 'choice':
      return selectedOption(value);
    case 'choiceWithOther':
      if(typeof value!=='string')return false;
      if(value.startsWith('other:')){
        const other=value.slice('other:'.length);
        return other.length>0&&other.length<=field.otherMaxLength;
      }
      return selectedOption(value);
    case 'multi':case 'multiLimited':
      if(!Array.isArray(value)||new Set(value).size!==value.length||
         !value.every(selectedOption))return false;
      return field.kind!=='multiLimited'||
        (value.length>=field.minSelected&&value.length<=field.maxSelected);
    case 'boolean':
      return typeof value==='boolean';
    case 'confirm':
      return value===true;
    case 'number':{
      if(typeof value!=='string'||value.trim()==='')return false;
      const number=Number(value);
      return Number.isFinite(number)&&number>=field.min&&number<=field.max&&
        (!field.integerOnly||Number.isSafeInteger(number));
    }
    default:return false;
  }
}
export function verifyGuidedPayload(payload,snapshot){
  if(!plain(payload)||Object.keys(payload).sort().join(',')!==
    'answers,journeyId,revision,sourceSha,stepId')return false;
  const step=snapshot?.step;
  if(!step||payload.sourceSha!==snapshot.sourceSha||
    payload.journeyId!==step.journeyId||payload.stepId!==step.stepId||
    payload.revision!==step.revision||!plain(payload.answers))return false;
  if(!Array.isArray(step.fields)||step.fields.length<1||step.fields.length>24)return false;
  const allowed=new Map(step.fields.map(f=>[f.id,f]));
  if(allowed.size!==step.fields.length)return false;
  const visible=field=>{
    if(!field.visibleWhen)return true;
    if(field.visibleWhen.fieldId===field.id)return false;
    const current=payload.answers[field.visibleWhen.fieldId];
    const expected=field.visibleWhen.equals;
    if(expected===null)return current===undefined||current===null||current==='';
    return current===expected;
  };
  if(Object.keys(payload.answers).length>24)return false;
  for(const [key,value] of Object.entries(payload.answers)){
    const field=allowed.get(key);
    if(!field||field.disabledReason||!visible(field)||!valueOk(value)||
       !answerMatchesIssuedField(field,value))return false;
    if(field.required&&(value===null||
      typeof value==='string'&&value.trim().length===0||
      Array.isArray(value)&&value.length===0))return false;
  }
  for(const field of step.fields){
    if(field.required&&!field.disabledReason&&visible(field)&&
       !Object.prototype.hasOwnProperty.call(payload.answers,field.id))
      return false;
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
    return {connected:true,sourceSha:currentSha,step:projectStep(snapshot.step)};
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
    // Close the read→dispatch race: the selected Business/Host must still
    // issue the same exact step immediately before the side effect. K/Host
    // performs its own CAS/permit check inside dispatchStep.
    const latest=await host.readCurrentStep({...scope,sourceSha:currentSha});
    if(!verifyGuidedSnapshot(latest,scope,currentSha)||
       canonicalJson(projectStep(latest.step))!==canonicalJson(read.step))
      return deny('GUIDED_STEP_CHANGED_BEFORE_DISPATCH');
    const result=await host.dispatchStep({
      ...scope,sourceSha:currentSha,journeyId:payload.journeyId,
      stepId:payload.stepId,revision:payload.revision,answers:payload.answers});
    if(!plain(result)||result.ok!==true||
      (result.nextStep!==null&&
       !verifyGuidedSnapshot({...read,step:result.nextStep,projectId:scope.projectId,
         versionId:scope.versionId,currentness:'CURRENT',readback:'VERIFIED'},
         scope,currentSha)))
      return deny('GUIDED_HOST_RESULT_UNQUALIFIED');
    return {connected:true,ok:true,nextStep:
      result.nextStep===null?null:projectStep(result.nextStep)};
  }catch{return deny('GUIDED_HOST_DISPATCH_FAILED');}
}
