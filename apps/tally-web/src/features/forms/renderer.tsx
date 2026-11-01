/** UI-B #229 — React renderer for one issued dynamic step.
 * Values render as plain text (React escaping by construction; no
 * dangerouslySetInnerHTML in this feature). Recommendations show a hint but
 * never preselect; ineligible options render disabled with their
 * Business-issued reason. Authorized effects belong to the trusted Host.
 */
import {useEffect,useMemo,useState} from 'react';
import type {Field,FormStep,FieldOption} from './schema.ts';
import type {FieldValue} from './engine.ts';
import {validateStepSpec} from './schema.ts';
import {visibleFields,validateStep,serializeDraft,restoreDraft,buildSubmitPayload,
  type FieldErrors} from './engine.ts';

export interface FormStrings{
  required:string;tooLong:string;unknownOption:string;tooFew:string;
  tooMany:string;notANumber:string;notInteger:string;tooSmall:string;
  tooLarge:string;otherEmpty:string;confirmRequired:string;typeInvalid:string;
  controlChars:string;
  next:string;back:string;pause:string;resume:string;cancel:string;
  recommended:string;other:string;submitting:string;
}
export interface RendererHost{
  submitStep(payload:ReturnType<typeof buildSubmitPayload>):
    Promise<{ok:true;nextStep:FormStep|null}|{ok:false;reason:string}>;
  saveDraft(raw:string):void;loadDraft():string|null;clearDraft():void;
  locale:'zh'|'en';strings:FormStrings;
}
const errorCodes:Record<string,keyof FormStrings>={
  required:'required',too_long:'tooLong',unknown_option:'unknownOption',
  too_few:'tooFew',too_many:'tooMany',not_a_number:'notANumber',
  not_integer:'notInteger',too_small:'tooSmall',too_large:'tooLarge',
  other_empty:'otherEmpty',confirm_required:'confirmRequired',type:'typeInvalid',
  control_chars:'controlChars',
};
export const DEFAULT_STRINGS_ZH:FormStrings={
  required:'此项必填',tooLong:'超出长度限制',unknownOption:'不是本步骤下发的选项',
  tooFew:'选择数量不足',tooMany:'选择数量超限',notANumber:'请输入数字',
  notInteger:'请输入整数',tooSmall:'数值过小',tooLarge:'数值过大',
  otherEmpty:'请填写"其他"内容',confirmRequired:'需勾选确认',typeInvalid:'类型不正确',
  controlChars:'包含不允许的控制字符',
  next:'下一步',back:'上一步',pause:'暂停',resume:'继续',cancel:'取消',
  recommended:'推荐',other:'其他',submitting:'提交中…',
};
const inputStyle:React.CSSProperties={background:'var(--bg-app)',
  border:'1px solid var(--border-strong)',borderRadius:'var(--r-sm)',
  color:'var(--text-primary)',padding:'9px 12px',font:'inherit',width:'100%'};

export function DynamicForm({initialStep,host,onDone}:
  {initialStep:unknown;host:RendererHost;
   onDone:(outcome:'completed'|'cancelled')=>void}){
  const issued=useMemo(()=>validateStepSpec(initialStep),[initialStep]);
  const [step,setStep]=useState<FormStep|null>(issued);
  const [answers,setAnswers]=useState<Record<string,FieldValue>>({});
  const [errors,setErrors]=useState<FieldErrors>({});
  const [phase,setPhase]=useState<'editing'|'paused'|'submitting'|'server_error'>('editing');
  const [serverReason,setServerReason]=useState<string|null>(null);
  useEffect(()=>{
    if(!issued)return;
    const restored=restoreDraft(issued,host.loadDraft());
    if(restored)setAnswers(restored);
  },[issued,host]);
  useEffect(()=>{
    if(!step||Object.keys(answers).length===0)return;
    host.saveDraft(serializeDraft({step,answers,errors,status:'editing'}));
  },[answers,step,errors,host]);
  if(!issued){
    return <div role="alert" style={{color:'var(--danger)',padding:'var(--s4)'}}>
      FORM_STEP_UNVERIFIED — 服务端下发的步骤未通过结构校验,已拒绝渲染。
    </div>;
  }
  if(!step)return null;
  const zh=host.locale==='zh';
  const text=(value:{zh:string;en:string})=>zh?value.zh:value.en;
  const has=(action:string)=>step.actions.includes(action as never);
  const setValue=(fieldId:string,value:FieldValue)=>{
    setAnswers(state=>({...state,[fieldId]:value}));
    setErrors(state=>{const next={...state};delete next[fieldId];return next;});
  };
  const submit=async()=>{
    const nextErrors=validateStep(step,answers);
    setErrors(nextErrors);
    if(Object.keys(nextErrors).length>0)return;
    setPhase('submitting');
    const result=await host.submitStep(buildSubmitPayload(
      {step,answers,errors:{},status:'editing'}));
    if(!result.ok){setPhase('server_error');setServerReason(result.reason);return;}
    setPhase('editing');setServerReason(null);
    if(result.nextStep===null){host.clearDraft();onDone('completed');return;}
    const nextIssued=validateStepSpec(result.nextStep);
    if(!nextIssued){setPhase('server_error');setServerReason('NEXT_STEP_UNVERIFIED');return;}
    setStep(nextIssued);setAnswers({});setErrors({});
  };
  const fields=visibleFields(step,answers);
  return <form onSubmit={event=>{event.preventDefault();void submit();}}
    aria-label={text(step.title)}>
    <h2 style={{marginTop:0}}>{text(step.title)}</h2>
    {step.intro?<p style={{color:'var(--text-secondary)'}}>{text(step.intro)}</p>:null}
    {fields.map(field=>(
      <FieldView key={field.id} field={field} value={answers[field.id]}
        errorText={errors[field.id]
          ?host.strings[errorCodes[errors[field.id]]??'typeInvalid']:null}
        disabled={Boolean(field.disabledReason)||phase==='submitting'}
        host={host}
        onChange={value=>setValue(field.id,value)}/>))}
    {phase==='server_error'&&serverReason?
      <div role="alert" style={{color:'var(--danger)',marginTop:'var(--s3)'}}>{serverReason}</div>:null}
    <div style={{display:'flex',gap:'var(--s2)',marginTop:'var(--s5)',flexWrap:'wrap'}}>
      {has('back')?<button type="button" onClick={()=>onDone('cancelled')}
        disabled={phase==='submitting'}>{host.strings.back}</button>:null}
      {has('pause')?<button type="button" onClick={()=>setPhase('paused')}
        disabled={phase==='submitting'}>{host.strings.pause}</button>:null}
      {phase==='paused'&&has('resume')?
        <button type="button" onClick={()=>setPhase('editing')}>{host.strings.resume}</button>:null}
      {has('cancel')?<button type="button"
        onClick={()=>{host.clearDraft();onDone('cancelled');}}
        disabled={phase==='submitting'}>{host.strings.cancel}</button>:null}
      <button type="submit" disabled={phase==='submitting'||phase==='paused'}>
        {phase==='submitting'?host.strings.submitting:host.strings.next}
      </button>
    </div>
  </form>;
}
function FieldView({field,value,errorText,disabled,host,onChange}:
  {field:Field;value:FieldValue|undefined;errorText:string|null;disabled:boolean;
   host:RendererHost;onChange:(value:FieldValue)=>void}){
  const zh=host.locale==='zh';
  const label=zh?field.label.zh:field.label.en;
  const hint=field.hint?(zh?field.hint.zh:field.hint.en):null;
  const disabledReason=field.disabledReason?(zh?field.disabledReason.zh:field.disabledReason.en):null;
  const invalid=errorText!==null;
  const title=<span style={{fontWeight:600}}>{label}
    {field.required?<span aria-hidden="true" style={{color:'var(--danger)'}}> *</span>:null}</span>;
  const below=<>
    {hint?<div style={{color:'var(--text-faint)',fontSize:12,margin:'2px 0 var(--s1)'}}>{hint}</div>:null}
    {disabledReason?<div style={{color:'var(--warn)',fontSize:12,margin:'2px 0 var(--s1)'}}>{disabledReason}</div>:null}
    {invalid?<div role="alert" style={{color:'var(--danger)',fontSize:12.5,
      margin:'2px 0 var(--s1)'}}>{errorText}</div>:null}
  </>;
  const wrap=(control:React.ReactNode)=>
    <fieldset disabled={false} style={{border:0,padding:0,margin:'0 0 var(--s4)'}}>
      <legend style={{all:'unset'}}>{title}</legend>
      {control}{below}
    </fieldset>;
  switch(field.kind){
    case 'text':
      return wrap(<input type="text" style={inputStyle}
        value={typeof value==='string'?value:''} maxLength={field.maxLength}
        disabled={disabled} aria-invalid={invalid}
        aria-label={label} onChange={event=>onChange(event.target.value)}/>);
    case 'textArea':
      return wrap(<textarea style={{...inputStyle,resize:'vertical'}} rows={4}
        value={typeof value==='string'?value:''} maxLength={field.maxLength}
        disabled={disabled} aria-invalid={invalid}
        aria-label={label} onChange={event=>onChange(event.target.value)}/>);
    case 'boolean':
      return wrap(<label style={{display:'flex',gap:'var(--s2)',alignItems:'center'}}>
        <input type="checkbox" checked={value===true} disabled={disabled}
          aria-invalid={invalid} onChange={event=>onChange(event.target.checked)}/>
        <span style={{fontWeight:400}}>{label}</span></label>);
    case 'number':
      return wrap(<input type="number" style={inputStyle}
        value={typeof value==='string'?value:''} min={field.min} max={field.max}
        step={field.integerOnly?1:'any'} disabled={disabled} aria-invalid={invalid}
        aria-label={label} onChange={event=>onChange(event.target.value)}/>);
    case 'confirm':
      return wrap(<label style={{display:'flex',gap:'var(--s2)',alignItems:'flex-start'}}>
        <input type="checkbox" checked={value===true} disabled={disabled}
          aria-invalid={invalid} onChange={event=>onChange(event.target.checked)}/>
        <span style={{fontWeight:400}}>{zh?field.statement.zh:field.statement.en}</span></label>);
    case 'choice':case 'choiceWithOther':case 'multi':case 'multiLimited':{
      const options:FieldOption[]=field.options;
      const selected=Array.isArray(value)?value:typeof value==='string'?[value]:[];
      const multi=field.kind==='multi'||field.kind==='multiLimited';
      const otherText=typeof value==='string'&&value.startsWith('other:')
        ?value.slice('other:'.length):'';
      const controls=options.map(option=>{
        const checked=selected.includes(option.id);
        return <label key={option.id} style={{display:'flex',gap:'var(--s2)',
          alignItems:'flex-start',marginBottom:'var(--s2)',
          opacity:option.eligible?1:0.55}}>
          <input type={multi?'checkbox':'radio'} name={field.id} value={option.id}
            checked={checked} disabled={disabled||!option.eligible}
            onChange={()=>{
              if(multi)onChange(checked?selected.filter(item=>item!==option.id)
                :[...selected,option.id]);
              else onChange(option.id);}}/>
          <span style={{fontWeight:400}}>
            {zh?option.label.zh:option.label.en}
            {option.recommended?` (${host.strings.recommended})`:null}
            {!option.eligible&&option.ineligibleReason?
              <span style={{color:'var(--warn)',fontSize:12}}>
                {' — '}{zh?option.ineligibleReason.zh:option.ineligibleReason.en}</span>:null}
          </span>
        </label>;});
      if(field.kind==='choiceWithOther')
        controls.push(<label key="other" style={{display:'flex',gap:'var(--s2)',
          alignItems:'center',marginBottom:'var(--s2)',flexWrap:'wrap'}}>
          <input type="radio" name={field.id} value="other"
            checked={typeof value==='string'&&value.startsWith('other:')}
            disabled={disabled} onChange={()=>onChange('other:')}/>
          <span style={{fontWeight:400}}>{host.strings.other}</span>
          {typeof value==='string'&&value.startsWith('other:')?
            <input type="text" style={{...inputStyle,width:240}}
              maxLength={field.otherMaxLength} disabled={disabled}
              aria-label={host.strings.other} value={otherText}
              onChange={event=>onChange(`other:${event.target.value}`)}/>:null}
        </label>);
      return wrap(<div role="group" aria-label={label}>{controls}</div>);
    }
  }
}
