/**
 * Tally v2 project-intake journey. App-owned thin consumer of frozen
 * DomainHarness v0.10.3 DATA wire, not a claimed published SDK API.
 * No LLM calls, Kernel/Host effects, permissions, claims or GitHub writes.
 */
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import Ajv2020 from 'ajv/dist/2020.js';

const SCHEMA_GIT_BLOB='43710e2a1d5bc2b1827014c99867749b70c764f7';
const bytes=readFileSync(new URL('../vendor/guided-journey-v1.schema.json',import.meta.url));
const gitHash=createHash('sha1').update('blob '+bytes.length+'\0').update(bytes).digest('hex');
if(gitHash!==SCHEMA_GIT_BLOB)throw Error('FROZEN_GUIDED_JOURNEY_CONTRACT_DRIFT');
const schema=JSON.parse(bytes.toString('utf8'));
const ajv=new Ajv2020({strict:true,allErrors:false,validateFormats:false,
  removeAdditional:false,useDefaults:false,coerceTypes:false});
const wire=ajv.compile(schema);
const VERSION='dh.guided-journey/1';
const sha=o=>createHash('sha256').update(JSON.stringify(o)).digest('hex');
const template=(id,stepId,kind,titleKey,fields,allowedEvents)=>{
  const material={artifactId:'tally.intake.'+id,version:'1',stepId,kind,titleKey,fields,allowedEvents};
  return Object.freeze({material:Object.freeze(material),
    ref:Object.freeze({artifactId:material.artifactId,version:'1',digest:sha(material)})});
};
const option=(id,labelKey)=>Object.freeze({id,labelKey,eligible:true});
const goal=template('goal','goal','ASK','tally.intake.goal',[
  {id:'goal',type:'CHOICE',labelKey:'tally.intake.goal.label',required:true,
    options:[option('new-feature','tally.goal.feature'),option('bug-fix','tally.goal.bug'),
      option('research','tally.goal.research')]}
],['USER_SUBMIT','PAUSE']);
const scope=template('scope','scope','ASK','tally.intake.scope',[
  {id:'scope',type:'CHOICE',labelKey:'tally.intake.scope.label',required:true,
    options:[option('bounded','tally.scope.bounded'),option('multi-lane','tally.scope.multilane')]}
],['USER_SUBMIT','BACK','PAUSE']);
const review=template('review','review','SUMMARY','tally.intake.review',[],
  ['USER_SUBMIT','BACK','PAUSE']);
const steps=Object.freeze({goal,scope,review});
const allRefs=Object.freeze(Object.values(steps).map(x=>x.ref));
export const catalogRef=Object.freeze({artifactId:'tally.project-intake.catalog',
  version:'1',digest:sha(allRefs)});
export const templateRefs=allRefs;
const equals=(a,b)=>a?.artifactId===b?.artifactId&&a?.version===b?.version&&
  a?.digest===b?.digest;
const matchingCase=(a,b)=>a?.snapshotId===b?.snapshotId&&a?.digest===b?.digest;
const noAuthority=Object.freeze({authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const decision=(request,status,reasonCode,step)=>{
  const out={messageType:'next-step.decision',contractVersion:VERSION,
    journeyId:request?.journeyId||'UNBOUND',status,reasonCode,evidenceRefs:[]};
  if(step)out.step=step;
  if(!wire(out))throw Error('TALLY_GUIDED_DECISION_SCHEMA_MISMATCH');
  return Object.freeze(out);
};
const blocked=(request,reason)=>{
  // Malformed request may omit the required journeyId. Return an app-local
  // typed refusal rather than constructing invalid canonical wire.
  if(!request||typeof request.journeyId!=='string'||request.journeyId.length>128||
    !request.journeyId.length)
    return Object.freeze({...noAuthority,status:'BLOCKED',reason:'INVALID_SOURCE_REQUEST',
      decision:null});
  return Object.freeze({...noAuthority,status:'BLOCKED',reason,
    decision:decision(request,'BLOCKED',reason)});
};
const present=(t,request,verifiedEvidenceRefs)=>{
  const m=t.material;
  const step={
    messageType:'step.spec',contractVersion:VERSION,journeyId:request.journeyId,
    stepId:m.stepId,templateRef:t.ref,caseRef:request.caseRef,
    kind:m.kind,titleKey:m.titleKey,fields:m.fields,
    allowedEvents:m.allowedEvents,evidenceRefs:verifiedEvidenceRefs
  };
  if(!wire(step))throw Error('TALLY_GUIDED_STEP_SCHEMA_MISMATCH');
  return Object.freeze(step);
};
const outcomeStep=o=>{
  if(o?.stepId==='goal')return goal;
  if(o?.stepId==='scope')return scope;
  if(o?.stepId==='review')return review;
  return null;
};
const validSelection=(o,t)=>{
  if(o.event!=='USER_SUBMIT')return true;
  const f=t.material.fields[0];
  if(!f)return !o.selectedOptionIds?.length;
  return o.selectedOptionIds?.length===1&&f.options?.some(
    opt=>opt.eligible&&opt.id===o.selectedOptionIds[0]);
};
const uniqueBoundedRefs=x=>Array.isArray(x)&&x.length<=24&&
  x.every(s=>typeof s==='string'&&s.length>0&&s.length<=256)&&
  new Set(x).size===x.length;
/**
 * Produce one next UI message after authentic Host-side case/evidence checks.
 * sourceEvidenceRefs MUST be provided by a trusted source adapter, NEVER
 * extracted from a user outcome or model prose; still grants no authority.
 */
export function nextProjectIntakeStep(request,{sourceEvidenceRefs=[],advisory=null}={}){
  try{
    if(!wire(request)||request.messageType!=='next-step.request'||
      !equals(request.catalogRef,catalogRef)||
      !request.eligibleTemplateRefs.every(ref=>allRefs.some(x=>equals(x,ref)))||
      !uniqueBoundedRefs(sourceEvidenceRefs))
      return blocked(request,'SOURCE_OR_CATALOG_UNVERIFIED');

    const permitted=t=>request.eligibleTemplateRefs.some(x=>equals(x,t.ref));
    const prev=request.previousOutcome;
    let next=goal;
    if(prev){
      if(prev.journeyId!==request.journeyId||
        !wire(prev)||prev.messageType!=='step.outcome')
        return blocked(request,'OUTCOME_SOURCE_UNVERIFIED');
      const t=outcomeStep(prev);
      if(!t||!t.material.allowedEvents.includes(prev.event)||
        (prev.receiptRef!==undefined&&
          !['OBSERVATION_RECEIPT','EFFECT_RECEIPT'].includes(prev.event))||
        !validSelection(prev,t))
        return blocked(request,'OUTCOME_EVENT_OR_SELECTION_INVALID');
      if(prev.event==='PAUSE')
        return Object.freeze({...noAuthority,status:'WAIT',reason:'USER_PAUSED',
          decision:decision(request,'WAIT','USER_PAUSED')});
      if(prev.event==='BACK'){
        next=t===review?scope:goal;
      }else if(prev.event==='USER_SUBMIT'){
        if(t===goal)next=scope;
        else if(t===scope){
          // User-provided evidenceRefs do not count by themselves. Every
          // asserted receipt must be source-qualified by the trusted Host.
          if(!prev.evidenceRefs?.length||
            !prev.evidenceRefs.every(x=>sourceEvidenceRefs.includes(x)))
            return Object.freeze({...noAuthority,status:'NEED_EVIDENCE',
              reason:'SOURCE_EVIDENCE_REQUIRED',
              decision:decision(request,'NEED_EVIDENCE','SOURCE_EVIDENCE_REQUIRED')});
          next=review;
        }else
          return Object.freeze({...noAuthority,status:'WAIT',
            reason:'INDEPENDENT_PRODUCT_REVIEW_REQUIRED',
            decision:decision(request,'WAIT','INDEPENDENT_PRODUCT_REVIEW_REQUIRED')});
      }else
        return blocked(request,'UNSUPPORTED_OUTCOME_EVENT');
    }
    if(!permitted(next))return blocked(request,'NEXT_TEMPLATE_NOT_ELIGIBLE');
    // An optional llm.advisory is non-authoritative and intentionally cannot
    // replace deterministic approved Step selection or claim Kernel rights.
    const advisoryAccepted=advisory!==null&&wire(advisory)&&
      advisory.messageType==='llm.advisory'&&
      advisory.journeyId===request.journeyId&&
      matchingCase(advisory.caseRef,request.caseRef)&&
      equals(advisory.suggestedTemplateRef,next.ref)&&permitted(next);
    const step=present(next,request,sourceEvidenceRefs);
    const d=decision(request,'STEP','APPROVED_STATIC_TEMPLATE',step);
    return Object.freeze({...noAuthority,status:'STEP',reason:'APPROVED_STATIC_TEMPLATE',
      advisoryUsedForAuthority:false,advisoryMatched:advisoryAccepted,
      decision:d});
  }catch{return blocked(request,'MALFORMED_INPUT_OR_SOURCE');}
}
