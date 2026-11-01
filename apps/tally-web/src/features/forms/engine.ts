/** UI-B #229 — form engine: UX step state, typed validation and draft
 * recovery. This is NOT task authority: authorized effects are owned by the
 * selected K/Host, which revalidates the exact issued stepId+revision and
 * option ids server-side; the engine merely captures and convenience-checks.
 */
import type {Field,FormStep,FieldOption} from './schema.ts';

export type FieldValue=string|string[]|boolean|null;
export type FieldErrors=Record<string,string>;
export interface JourneyDraft{
  journeyId:string;
  stepId:string;
  revision:number;
  answers:Record<string,FieldValue>;
  savedAt:string;
}
export interface EngineState{
  step:FormStep;
  answers:Record<string,FieldValue>;
  errors:FieldErrors;
  status:'editing'|'paused'|'submitted';
}

/** Visible fields after evaluating Business-issued visibility rules against
 * the current answers. Cycles in visibleWhen are impossible (single-level
 * equality on other field ids), self-reference is rejected fail-closed. */
export function visibleFields(step:FormStep,
  answers:Record<string,FieldValue>):Field[]{
  return step.fields.filter(field=>{
    if(!field.visibleWhen)return true;
    if(field.visibleWhen.fieldId===field.id)return false; // self-reference: hide (fail-closed)
    const current=answers[field.visibleWhen.fieldId];
    const expected=field.visibleWhen.equals;
    if(typeof expected==='boolean')return current===expected;
    if(expected===null)return current===undefined||current===null||current==='';
    return current===expected;
  });
}

/** Typed bounded validation — mirrors the server contract; convenience only.
 * Ineligible options and unexpected values are refused here AND server-side. */
export function validateField(field:Field,value:FieldValue|undefined,
  optionIds:Set<string>):string|null{
  const empty=value===undefined||value===null||value===''||
    (Array.isArray(value)&&value.length===0);
  if(field.disabledReason)return null; // disabled fields are not submittable; validated server-side
  if(field.required&&field.kind==='confirm'&&value!==true)
    return 'confirm_required';
  if(field.required&&empty&&field.kind!=='confirm')return 'required';
  // multiLimited carries its own bounds even when empty and not required.
  if(empty&&field.kind!=='confirm'&&field.kind!=='multiLimited')return null;
  switch(field.kind){
    case 'text':case 'textArea':{
      if(typeof value!=='string')return 'type';
      if(value.length>field.maxLength)return 'too_long';
      // No control characters other than newline/tab (XSS-hardening: value is
      // rendered as text, but keep input bounded anyway).
      if(/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/.test(value))return 'control_chars';
      return null;}
    case 'choice':{
      if(typeof value!=='string'||!optionIds.has(value))return 'unknown_option';
      return null;}
    case 'choiceWithOther':{
      if(typeof value!=='string')return 'type';
      if(value.startsWith('other:')){
        const text=value.slice('other:'.length);
        if(text.length===0)return 'other_empty';
        if(text.length>field.otherMaxLength)return 'too_long';
        return null;}
      if(!optionIds.has(value))return 'unknown_option';
      return null;}
    case 'multi':{
      if(!Array.isArray(value))return 'type';
      if(value.some(item=>typeof item!=='string'||!optionIds.has(item)))
        return 'unknown_option';
      return null;}
    case 'multiLimited':{
      if(!Array.isArray(value))return 'type';
      if(value.some(item=>typeof item!=='string'||!optionIds.has(item)))
        return 'unknown_option';
      if(value.length<field.minSelected)return 'too_few';
      if(value.length>field.maxSelected)return 'too_many';
      return null;}
    case 'boolean':
      return typeof value==='boolean'?null:'type';
    case 'number':{
      const number=typeof value==='string'&&value.trim()!==''?Number(value):value;
      if(typeof number!=='number'||!Number.isFinite(number))return 'not_a_number';
      if(field.integerOnly&&!Number.isInteger(number))return 'not_integer';
      if(number<field.min)return 'too_small';
      if(number>field.max)return 'too_large';
      return null;}
    case 'confirm':
      return value===true?null:'confirm_required';
  }
}

/** Validate all visible+enabled fields; returns errors keyed by field id. */
export function validateStep(step:FormStep,
  answers:Record<string,FieldValue>):FieldErrors{
  const errors:FieldErrors={};
  for(const field of visibleFields(step,answers)){
    if(field.disabledReason)continue;
    const optionIds=new Set<string>();
    if('options' in field)
      for(const option of field.options as FieldOption[])optionIds.add(option.id);
    const error=validateField(field,answers[field.id],optionIds);
    if(error)errors[field.id]=error;
  }
  return errors;
}

/** Serialize a draft for refresh recovery (UX only, never authoritative). */
export function serializeDraft(state:EngineState):string{
  return JSON.stringify({journeyId:state.step.journeyId,stepId:state.step.stepId,
    revision:state.step.revision,answers:state.answers,
    savedAt:new Date().toISOString()});
}
/** Restore a draft ONLY if it matches the currently issued step exactly;
 * a stale/foreign draft is discarded (fail-closed, no answer smuggling). */
export function restoreDraft(step:FormStep,raw:string|null):Record<string,FieldValue>|null{
  if(!raw)return null;
  let draft:JourneyDraft;
  try{draft=JSON.parse(raw) as JourneyDraft;}catch{return null;}
  if(!draft||typeof draft!=='object')return null;
  if(draft.journeyId!==step.journeyId||draft.stepId!==step.stepId
    ||draft.revision!==step.revision)return null;
  if(!draft.answers||typeof draft.answers!=='object'||Array.isArray(draft.answers))return null;
  // Keep only known field ids with kind-typed value shapes; anything else is
  // dropped (fail-closed, no answer smuggling across steps or kinds).
  const kindById=new Map(step.fields.map(field=>[field.id,field.kind]));
  const answers:Record<string,FieldValue>={};
  for(const [fieldId,value] of Object.entries(draft.answers)){
    const kind=kindById.get(fieldId);
    if(!kind)continue;
    const stringOk=['text','textArea','choice','choiceWithOther','number']
      .includes(kind);
    const multiOk=kind==='multi'||kind==='multiLimited';
    const boolOk=kind==='boolean'||kind==='confirm';
    if(stringOk&&typeof value==='string')answers[fieldId]=value;
    else if(multiOk&&Array.isArray(value)&&value.every(item=>typeof item==='string'))
      answers[fieldId]=value;
    else if(boolOk&&typeof value==='boolean')answers[fieldId]=value;
  }
  return answers;
}

/** Build the exact submit payload for the trusted Host: only visible enabled
 * fields, values as captured, plus the issued step identity for revalidation. */
export function buildSubmitPayload(state:EngineState){
  if(state.status!=='editing')throw new Error('ENGINE_STATE_INVALID');
  const errors=validateStep(state.step,state.answers);
  if(Object.keys(errors).length>0)
    throw new Error(`VALIDATION_FAILED:${Object.keys(errors).join(',')}`);
  const payload:Record<string,FieldValue>={};
  for(const field of visibleFields(state.step,state.answers)){
    if(field.disabledReason)continue;
    payload[field.id]=state.answers[field.id]??null;
  }
  return {journeyId:state.step.journeyId,stepId:state.step.stepId,
    revision:state.step.revision,answers:payload};
}
