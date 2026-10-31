/** Deterministic guided Project/Product intake. Advisory only; no K/Claim/Effect. */
import Ajv2020 from 'ajv/dist/2020.js';
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';

export const GUIDED_SCHEMA_BLOB='43710e2a1d5bc2b1827014c99867749b70c764f7';
const schema=JSON.parse(readFileSync(new URL('../schemas/guided-journey-v1.schema.json',import.meta.url),'utf8'));
const ajv=new Ajv2020({allErrors:false,strict:true,strictRequired:false});
const validate=ajv.compile(schema);
const sha256=value=>createHash('sha256').update(value).digest('hex');
const canonical=value=>Array.isArray(value)?value.map(canonical):
  value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(key=>[key,canonical(value[key])])):value;
const digest=value=>sha256(JSON.stringify(canonical(value)));
export const GUIDED_TEMPLATE_REF=Object.freeze({artifactId:'tally.guided-intake',version:'r0',
  digest:sha256('tally.guided-intake:r0')});
const sameRef=(left,right)=>left?.artifactId===right?.artifactId&&left?.version===right?.version&&left?.digest===right?.digest;
const sameCaseRef=(left,right)=>left?.snapshotId===right?.snapshotId&&left?.digest===right?.digest;
const decision=(journeyId,status,reasonCode,evidenceRefs=[],step)=>Object.freeze({messageType:'next-step.decision',
  contractVersion:'dh.guided-journey/1',journeyId,status,reasonCode,evidenceRefs,...step?{step}:{}});
const invalid=(journeyId,reason)=>decision(typeof journeyId==='string'?journeyId:'UNKNOWN','BLOCKED',reason);
const receiptValid=value=>typeof value==='string'&&/^receipt:sha256:[0-9a-f]{64}$/.test(value);
function stepFor(request,kind){
  const id=kind==='ASK'?'ask-goal':kind==='INSPECT'?'inspect-evidence':'summary';
  const fields=kind==='ASK'?[{id:'goal',type:'TEXT',labelKey:'guided.goal',required:true}]:
    kind==='INSPECT'?[{id:'evidence',type:'MULTI',labelKey:'guided.evidence',required:true,options:
      request.previousOutcome?.evidenceRefs?.map((ref,index)=>({id:`evidence-${index+1}`,labelKey:ref,eligible:true}))??[]}]:[];
  return {messageType:'step.spec',contractVersion:'dh.guided-journey/1',journeyId:request.journeyId,
    stepId:id,templateRef:GUIDED_TEMPLATE_REF,caseRef:request.caseRef,kind,titleKey:`guided.${id}`,
    fields,allowedEvents:kind==='SUMMARY'?['EFFECT_RECEIPT']:['USER_SUBMIT','PAUSE'],
    evidenceRefs:kind==='ASK'?[]:request.previousOutcome?.evidenceRefs??[]};
}
export function guidedCaseRef(snapshotId,snapshot){
  if(typeof snapshotId!=='string'||snapshotId.length===0||snapshot===undefined)throw new TypeError('GUIDED_CASE_SNAPSHOT_REQUIRED');
  return {snapshotId,digest:digest(snapshot)};
}
export function validGuidedMessage(value){return validate(value)===true;}
/** Optional LLMAdvisory may select only an eligible template from the same case/catalog. */
export function nextGuidedStep(request,{advisory,caseSnapshot}={}){
  if(!validGuidedMessage(request))return invalid(request?.journeyId,'SCHEMA_INVALID');
  if(!sameRef(request.catalogRef,GUIDED_TEMPLATE_REF))return invalid(request.journeyId,'STALE_CATALOG');
  if(caseSnapshot!==undefined&&guidedCaseRef(request.caseRef.snapshotId,caseSnapshot).digest!==request.caseRef.digest)
    return invalid(request.journeyId,'STALE_CASE');
  if(!request.eligibleTemplateRefs.some(ref=>sameRef(ref,GUIDED_TEMPLATE_REF)))
    return invalid(request.journeyId,'TEMPLATE_NOT_ELIGIBLE');
  if(advisory!==undefined){
    if(!validGuidedMessage(advisory)||!sameCaseRef(advisory.caseRef,request.caseRef)||
      !sameRef(advisory.suggestedTemplateRef,GUIDED_TEMPLATE_REF))
      return invalid(request.journeyId,'ADVISORY_INVALID');
  }
  if(request.goals.length===0)return decision(request.journeyId,'NEED_EVIDENCE','GOAL_REQUIRED');
  const previous=request.previousOutcome;
  if(!previous)return decision(request.journeyId,'STEP','ASK_GOAL',[],stepFor(request,'ASK'));
  if(previous.journeyId!==request.journeyId)return invalid(request.journeyId,'STALE_CASE');
  if(previous.event==='PAUSE')return decision(request.journeyId,'WAIT','PAUSED',previous.evidenceRefs);
  if(previous.event==='EFFECT_RECEIPT'&&!receiptValid(previous.receiptRef))
    return invalid(request.journeyId,'RECEIPT_UNVERIFIED');
  if(previous.event==='EFFECT_RECEIPT'&&previous.stepId==='summary')
    return decision(request.journeyId,'DONE','OUTCOME_OBSERVED',previous.evidenceRefs);
  if(previous.event==='USER_SUBMIT'&&previous.stepId==='ask-goal'){
    if(previous.evidenceRefs.length===0)return decision(request.journeyId,'NEED_EVIDENCE','EVIDENCE_REQUIRED');
    if(previous.evidenceRefs.length>16||previous.evidenceRefs.some(ref=>ref.length>128))
      return invalid(request.journeyId,'EVIDENCE_PRESENTATION_UNSUPPORTED');
    return decision(request.journeyId,'STEP','INSPECT_EVIDENCE',previous.evidenceRefs,stepFor(request,'INSPECT'));
  }
  if(previous.event==='USER_SUBMIT'&&previous.stepId==='inspect-evidence')
    return decision(request.journeyId,'STEP','SUMMARY_READY',previous.evidenceRefs,stepFor(request,'SUMMARY'));
  return decision(request.journeyId,'WAIT','OUTCOME_NOT_ACTIONABLE',previous.evidenceRefs);
}
