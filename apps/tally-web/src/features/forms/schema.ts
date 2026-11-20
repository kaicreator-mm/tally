/** UI-B #229 — app-private dynamic form schema (Business-issued StepSpec).
 *
 * The schema shape follows the proven Tally v1.4.1 P0-04 bounded-control
 * experience and the DH v0.10.3 guided-journey EXAMPLE; the upstream schema
 * is explicitly candidate, so this module owns a pinned app-local contract.
 * The UI never invents steps: every step is issued by the trusted Host with
 * an exact step id; unexpected fields/options/steps fail closed.
 */
export const FIELD_KINDS=Object.freeze(['text','textArea','choice','multi',
  'boolean','number','confirm','choiceWithOther','multiLimited'] as const);
export type FieldKind=typeof FIELD_KINDS[number];

export interface LocalizedText{zh:string;en:string;}
export interface FieldOption{
  id:string;
  label:LocalizedText;
  /** Business-issued eligibility; ineligible options render disabled+reason. */
  eligible:boolean;
  ineligibleReason?:LocalizedText|null;
  /** Recommendation flag renders a hint but NEVER preselects. */
  recommended?:boolean;
}
export interface BaseField{
  id:string;
  kind:FieldKind;
  label:LocalizedText;
  hint?:LocalizedText|null;
  required:boolean;
  /** Business rule for conditional visibility; evaluated against answers. */
  visibleWhen?:{fieldId:string;equals:string|boolean|null;}|null;
  /** Disabled with a human reason (e.g. awaiting evidence). */
  disabledReason?:LocalizedText|null;
}
export interface TextField extends BaseField{kind:'text';maxLength:number;}
export interface TextAreaField extends BaseField{kind:'textArea';maxLength:number;}
export interface ChoiceField extends BaseField{kind:'choice';options:FieldOption[];}
export interface MultiField extends BaseField{kind:'multi';options:FieldOption[];}
export interface BooleanField extends BaseField{kind:'boolean';}
export interface NumberField extends BaseField{kind:'number';
  min:number;max:number;integerOnly:boolean;}
export interface ConfirmField extends BaseField{kind:'confirm';
  statement:LocalizedText;}
export interface ChoiceWithOtherField extends BaseField{kind:'choiceWithOther';
  options:FieldOption[];otherMaxLength:number;}
export interface MultiLimitedField extends BaseField{kind:'multiLimited';
  options:FieldOption[];minSelected:number;maxSelected:number;}
export type Field=TextField|TextAreaField|ChoiceField|MultiField|BooleanField|
  NumberField|ConfirmField|ChoiceWithOtherField|MultiLimitedField;

export interface FormStep{
  /** Exact step identity issued by the trusted Host; opaque to the UI. */
  stepId:string;
  journeyId:string;
  title:LocalizedText;
  intro?:LocalizedText|null;
  fields:Field[];
  /** Server-issued allowed transitions for THIS step (subset). */
  actions:readonly ('next'|'back'|'pause'|'resume'|'cancel')[];
  /** Bump on re-issue; a stale stepId+revision answer set is refused. */
  revision:number;
}

const ID=/^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/;
const localized=(value:unknown):value is LocalizedText=>
  !!value&&typeof value==='object'&&!Array.isArray(value)&&
  typeof (value as LocalizedText).zh==='string'&&
  typeof (value as LocalizedText).en==='string'&&
  (value as LocalizedText).zh.length>0&&(value as LocalizedText).en.length>0;

export interface SchemaIssue{fieldId:string;code:string;}
/** Structural validation of an issued step. Any violation is fail-closed:
 * the renderer refuses the whole step rather than guessing. */
export function validateStepSpec(step:unknown):FormStep|null{
  if(!step||typeof step!=='object'||Array.isArray(step))return null;
  const s=step as Record<string,unknown>;
  if(typeof s.stepId!=='string'||!ID.test(s.stepId))return null;
  if(typeof s.journeyId!=='string'||!ID.test(s.journeyId))return null;
  if(!localized(s.title))return null;
  if(s.intro!==undefined&&s.intro!==null&&!localized(s.intro))return null;
  if(typeof s.revision!=='number'||!Number.isSafeInteger(s.revision)||s.revision<1)
    return null;
  if(!Array.isArray(s.fields)||s.fields.length===0||s.fields.length>24)return null;
  if(!Array.isArray(s.actions)||s.actions.length===0)return null;
  const allowed=new Set(['next','back','pause','resume','cancel']);
  for(const action of s.actions)
    if(!allowed.has(action as never))return null;
  const seen=new Set<string>();
  const fields:Field[]=[];
  for(const raw of s.fields){
    if(!raw||typeof raw!=='object'||Array.isArray(raw))return null;
    const f=raw as Record<string,unknown>;
    if(typeof f.id!=='string'||!ID.test(f.id)||seen.has(f.id))return null;
    seen.add(f.id);
    if(!FIELD_KINDS.includes(f.kind as FieldKind))return null;
    if(typeof f.required!=='boolean')return null;
    if(!localized(f.label))return null;
    if(f.hint!==undefined&&f.hint!==null&&!localized(f.hint))return null;
    if(f.disabledReason!==undefined&&f.disabledReason!==null&&!localized(f.disabledReason))return null;
    if(f.visibleWhen!==undefined&&f.visibleWhen!==null){
      const v=f.visibleWhen as Record<string,unknown>;
      if(typeof v.fieldId!=='string'||!ID.test(v.fieldId)||
        !('equals' in v)||!['string','boolean'].includes(typeof v.equals)&&v.equals!==null)return null;
    }
    const optionList=(value:unknown):FieldOption[]|null=>{
      if(!Array.isArray(value)||value.length===0||value.length>32)return null;
      const options:FieldOption[]=[];
      const optionIds=new Set<string>();
      for(const o of value){
        if(!o||typeof o!=='object'||Array.isArray(o))return null;
        const opt=o as Record<string,unknown>;
        if(typeof opt.id!=='string'||!ID.test(opt.id)||optionIds.has(opt.id))return null;
        optionIds.add(opt.id);
        if(!localized(opt.label))return null;
        if(typeof opt.eligible!=='boolean')return null;
        if(opt.ineligibleReason!==undefined&&opt.ineligibleReason!==null
          &&!localized(opt.ineligibleReason))return null;
        if(opt.recommended!==undefined&&typeof opt.recommended!=='boolean')return null;
        options.push({id:opt.id,label:opt.label,eligible:opt.eligible,
          ineligibleReason:(opt.ineligibleReason as LocalizedText|null)??null,
          recommended:opt.recommended===true});
      }
      return options;
    };
    switch(f.kind){
      case 'text':case 'textArea':
        if(typeof f.maxLength!=='number'||!Number.isSafeInteger(f.maxLength)||
          f.maxLength<1||f.maxLength>4000)return null;
        break;
      case 'choice':case 'multi':{
        if(!optionList(f.options))return null;
        break;}
      case 'boolean':break;
      case 'number':
        if(typeof f.min!=='number'||typeof f.max!=='number'||f.min>=f.max||
          !Number.isFinite(f.min)||!Number.isFinite(f.max))return null;
        if(typeof f.integerOnly!=='boolean')return null;
        break;
      case 'confirm':
        if(!localized(f.statement))return null;
        break;
      case 'choiceWithOther':{
        if(!optionList(f.options))return null;
        if(typeof f.otherMaxLength!=='number'||!Number.isSafeInteger(f.otherMaxLength)||
          f.otherMaxLength<1||f.otherMaxLength>500)return null;
        break;}
      case 'multiLimited':{
        if(!optionList(f.options))return null;
        if(typeof f.minSelected!=='number'||typeof f.maxSelected!=='number'||
          !Number.isSafeInteger(f.minSelected)||!Number.isSafeInteger(f.maxSelected)||
          f.minSelected<0||f.maxSelected<f.minSelected||f.maxSelected>32)return null;
        break;}
    }
    fields.push(f as unknown as Field);
  }
  return {...(s as unknown as FormStep),fields};
}
