import { createHash } from 'node:crypto';
import { isDeepStrictEqual } from 'node:util';
import { BoundedQueryElement, StandardWorkflowSession } from '@kaicreator/domain-harness';
import type { BoundedQueryLimits, StandardWorkflowOutcome, WorkflowLimits } from '@kaicreator/domain-harness';
import { admitDecision, admitValidation, complete, confirmed, dimensions, sameContract, sameIdentity, stop, validObligation } from './model.js';
import type { Candidate, Confirmation, Contract, DecisionRequest, Fact, Identity, Key, Obligation, Observation, ValidationRequest, Value } from './model.js';
export type { Confirmation, DecisionRequest, ValidationRequest } from './model.js';

const projectionBlob='f9657c07a72c134e6d21e6aa7523b9982ffb327f';
type Knowledge = { id:string; owner:string; kind:string; source:{path:string;blob:string;lines:number[]}; qualification:string; conceptRefs?:string[] };
type Projection = { sourceSha:string; concepts:Knowledge[]; procedures:Knowledge[] };
type Outcome<T> = StandardWorkflowOutcome<T>;
type Disposition = 'SUPPORTED'|'EXCLUDED'|'NEEDS_EVIDENCE'|'NOT_SUPPORTED';
type CandidateAssessment = { candidateId:string; disposition:Disposition; reason:string; evidenceRefs:string[]; unknowns:string[]; falsifier:string; knowledge:Knowledge[] };
export type DecisionAssessment = { identity:Identity; contract:Contract; projectionBlob:string; decision:'KEEP_EXISTING'|'TRADEOFF'|'ADVISORY_CANDIDATE'|'UNRESOLVED'; candidates:CandidateAssessment[]; knowledgeSha:string; qualification:'ADVISORY_NOT_RULE' };
type ObservationState = 'OBSERVED_PASS_SCOPED'|'OBSERVED_FAIL'|'NOT_RUN'|'UNKNOWN'|'CONFLICT'|'STALE';
type ObligationAssessment = { obligationId:string; obligation:Obligation; state:ObservationState; observations:{id:string;record:Observation;qualification:'VERIFIED_OBSERVATION'|'REPORTED_ONLY'|'UNRESOLVED'}[]; reason:string };
export type EvidenceAssessment = { identity:Identity; candidateId:string; projectionBlob:string; knowledgeSha:string; status:'PUBLIC_CONTRACT_PARTIAL_OR_FAILED'|'UNKNOWN'|'INCOMPLETE'|'TESTED_CONTRACTS_PASS_SCOPED'; obligations:ObligationAssessment[]; completeUniverse:boolean; knowledge:Knowledge[] };

function evaluate(c:Candidate, facts:Fact[], verified:(f:Fact)=>boolean, knowledge:Knowledge[]):CandidateAssessment {
  const used=new Set<string>(), unknown=new Set<string>();
  const get=(key:Key):Value=>{
    const f=facts.find(x=>x.candidateId===c.id&&x.key===key);
    if(!f||!verified(f)||f.applicability!=='YES'||f.value==='UNKNOWN'||f.value==='CONFLICT') {unknown.add(key);return 'UNKNOWN';}
    f.evidenceRefs.forEach(id=>used.add(id));return f.value;
  };
  const result=(disposition:Disposition,reason:string,falsifier:string):CandidateAssessment=>({candidateId:c.id,disposition,reason,evidenceRefs:[...used],unknowns:[...unknown],falsifier,knowledge});
  const conflict=facts.filter(f=>f.candidateId===c.id&&verified(f)&&(f.value==='CONFLICT'||f.applicability==='CONFLICT'));
  if(conflict.length){conflict.forEach(f=>{f.evidenceRefs.forEach(id=>used.add(id));unknown.add(f.key);});return result('NEEDS_EVIDENCE','MATERIAL_CONFLICT','Resolve incompatible observations for this candidate.');}
  const preserved=get('CONTRACT_PRESERVED');
  if(preserved==='NO')return result('EXCLUDED','CONTRACT_VIOLATION','Reproduce the declared observable contract against this candidate.');
  let incomplete=preserved!=='YES';
  if(c.merge!=='NONE'||c.kind==='PRIVATE_SHARED'||c.kind==='STRATEGY') {
    for(const dim of dimensions){
      const f=facts.find(x=>x.candidateId===c.id&&x.key===dim);
      if(f&&verified(f)&&f.applicability==='NO'){f.evidenceRefs.forEach(id=>used.add(id));continue;}
      const v=get(dim);
      if(v==='NO')return result('EXCLUDED','INCOMPATIBLE_'+dim,'Exercise the differing '+dim+' behavior against the proposed shared contract.');
      if(v!=='YES')incomplete=true;
    }
    if(c.merge==='PURE') {const v=get('PRESERVES_DIFFERENCES');if(v==='NO')return result('EXCLUDED','CALLER_DIFFERENCES_LOST','Run both callers with the input that distinguishes their required behavior.');if(v!=='YES')incomplete=true;}
  }
  const required:Key[]=c.kind==='PRIVATE_SHARED'?['MULTIPLE_MAINTENANCE_SITES','SAME_OWNER','SHARED_INVARIANT','SHARED_CHANGE']:c.kind==='DOMAIN_MODULE'?['INDEPENDENT_BOUNDARY']:c.kind==='STRATEGY'?['AUTHORIZED_REPLACEMENT']:c.kind==='INLINE'?['LOCAL_ONLY']:[];
  const requiredValues=required.map(get);
  let need:Value=requiredValues.includes('NO')?'NO':requiredValues.every(v=>v==='YES')?'YES':'UNKNOWN';
  if(c.kind==='PUBLIC_CONTRACT') {
    const reasons=(['STABLE_CONSUMERS','INDEPENDENT_RELEASE','STABLE_REPLACEMENT'] as const).map(get);
    need=reasons.includes('YES')?'YES':reasons.every(v=>v==='NO')?'NO':'UNKNOWN';
  }
  if(need==='NO'){
    const compatibility=get('EXISTING_COMPATIBILITY');
    if(c.existing&&compatibility==='YES'&&preserved==='YES'&&!incomplete)return result('SUPPORTED','PRESERVE_EXISTING_COMPATIBILITY','Run existing consumers against the unchanged public behavior.');
    if(compatibility==='NO'&&preserved==='YES')return result('NOT_SUPPORTED','NO_PROVEN_NEED','Show an authorized independent change or consumer requirement that would reverse this decision.');
  }
  if(incomplete||need!=='YES')return result('NEEDS_EVIDENCE','MATERIAL_FACTS_UNRESOLVED','Resolve the listed unknown dimensions or design needs against the current contract.');
  return result('SUPPORTED','EVIDENCED_SCOPE','Exercise a counterexample to the shared invariant, change axis or preserved contract.');
}

function evaluateObligation(o:Obligation, request:ValidationRequest, context:Confirmation):ObligationAssessment {
  // Use the authoritative checked records even if ordinary input omits or substitutes one.
  const checked=(context.observations??[]).filter(c=>c.record.obligationId===o.id&&c.record.candidateId===request.candidateId);
  const qualifies=(x:Observation)=>x.expectation===o.expected&&x.oracleLocator===o.locator&&x.oracleVersion===o.version&&x.oracle!=='ENGINEERING_HYPOTHESIS'&&(x.oracle==='DECLARED'||x.derivation.trim().length>0)&&x.coverage.length>0&&x.coverage.every(s=>o.scope.some(cell=>cell.id===s&&cell.target===x.target&&cell.environment===x.environment));
  const eligible=checked.filter(c=>confirmed(c.record,c.evidence,[c])&&qualifies(c.record));
  const current=eligible.filter(c=>c.record.sourceSha===request.identity.sourceSha&&c.evidence.every(e=>e.sourceSha===c.record.sourceSha)).map(c=>c.record);
  const observations:ObligationAssessment['observations']=checked.map(c=>({id:c.record.id,record:structuredClone(c.record),qualification:eligible.includes(c)?'VERIFIED_OBSERVATION':'UNRESOLVED'}));
  for(const x of request.observations.filter(x=>x.obligationId===o.id))if(!confirmed(x,request.evidence,checked))observations.push({id:x.id,record:structuredClone(x),qualification:'REPORTED_ONLY'});
  const report=(state:ObservationState,reason:string):ObligationAssessment=>({obligationId:o.id,obligation:structuredClone(o),state,reason,observations});
  const passes=current.filter(x=>x.execution==='EXECUTED'&&x.result==='PASS');
  const failures=current.filter(x=>x.execution==='EXECUTED'&&x.result==='FAIL');
  const overlaps=(a:Observation,b:Observation)=>a.environment===b.environment&&a.target===b.target&&a.coverage.some(s=>b.coverage.includes(s));
  if(failures.some(f=>f.coverage.some(cell=>!passes.some(p=>p.coverage.includes(cell)&&overlaps(f,p)))))return report('OBSERVED_FAIL','VERIFIED_COUNTEREXAMPLE');
  if(failures.length)return report('CONFLICT','QUALIFIED_CONTRADICTION');
  if(current.some(x=>x.execution==='NOT_RUN'&&current.some(y=>y.runId===x.runId&&y.execution==='EXECUTED'&&overlaps(x,y))))return report('CONFLICT','EXECUTION_CONTRADICTION');
  if(o.scope.length>0&&o.scope.every(s=>passes.some(p=>p.coverage.includes(s.id))))return report('OBSERVED_PASS_SCOPED','VERIFIED_SCOPE');
  if(current.length&&current.every(x=>x.execution==='NOT_RUN')&&o.scope.every(s=>current.some(x=>x.coverage.includes(s.id))))return report('NOT_RUN','VERIFIED_NONEXECUTION');
  if(eligible.length&&!current.length)return report('STALE','NO_CURRENT_VERIFIED_OBSERVATION');
  return report('UNKNOWN','INSUFFICIENT_QUALIFIED_SCOPE');
}

/** Stateless advisory Business Element. The caller, not this Element, establishes external authority. */
export class EngineeringDecisionElement {
  static readonly metadata={id:'engineeringDecision'} as const;
  readonly #projection:Projection;
  readonly #query=new BoundedQueryElement();
  constructor(projectionText:string){
    const bytes=Buffer.from(projectionText,'utf8');
    const blob=createHash('sha1').update('blob '+bytes.length+'\0').update(bytes).digest('hex');
    if(blob!==projectionBlob)throw new Error('ENGINEERING_KNOWLEDGE_IDENTITY_MISMATCH');
    this.#projection=JSON.parse(projectionText) as Projection;
  }
  #knowledge(oracle:boolean,limits:BoundedQueryLimits):Outcome<Knowledge[]> {
    const ids=oracle?['K05.ORACLE']:['DP-K02-R19-01','DP-K03-R19-01'];
    const wanted=new Set(ids);
    for(const p of this.#projection.procedures)if(wanted.has(p.id))p.conceptRefs?.forEach(x=>wanted.add(x));
    const source=[...this.#projection.concepts,...this.#projection.procedures];
    const selected=this.#query.select(source,x=>wanted.has(x.id),limits);
    return selected.complete&&selected.items.length===wanted.size?complete(structuredClone([...selected.items])):stop('UNKNOWN','QUERY_INCOMPLETE');
  }
  assessAbstraction(request:unknown, context:Confirmation|undefined, limits:BoundedQueryLimits={maxVisited:19,maxReturned:20}):Outcome<DecisionAssessment>{
    const denied=admitDecision(request);if(denied)return denied;
    const r=request as DecisionRequest;
    if((context?.facts?.length??0)>64||(context?.candidates?.length??0)>5)return stop('REJECT','INPUT_LIMIT_EXCEEDED');
    const trusted=sameIdentity(r.identity,context?.identity)&&sameContract(r.contract,context?.contract);
    const verified=(f:Fact)=>trusted&&context?.candidates?.some(c=>isDeepStrictEqual(c,r.candidates.find(x=>x.id===f.candidateId)))===true&&confirmed(f,r.evidence,context?.facts)&&f.evidenceRefs.every(id=>r.evidence.find(e=>e.id===id)?.sourceSha===r.identity.sourceSha);
    if(trusted&&r.contract.status==='CONFLICT')return stop('UNKNOWN','MATERIAL_CONFLICT');
    if(r.contract.status==='STALE'||r.facts.some(f=>f.evidenceRefs.some(id=>r.evidence.find(e=>e.id===id)?.sourceSha!==r.identity.sourceSha)))return stop('UNKNOWN','STALE_EVIDENCE');
    if(r.contract.status==='MISSING')return stop('ASK','CONTRACT_REQUIRED');
    if(!trusted||r.contract.status!=='CONFIRMED')return stop('UNKNOWN','EVIDENCE_UNQUALIFIED');
    const k=this.#knowledge(false,limits);if(k.status!=='COMPLETE')return k;
    const candidates=r.candidates.map(c=>evaluate(c,r.facts,verified,k.value));
    const supported=candidates.filter(c=>c.disposition==='SUPPORTED');
    const keep=supported.length===1&&r.candidates.some(c=>c.id===supported[0]?.candidateId&&c.existing&&r.facts.some(f=>f.candidateId===c.id&&f.key==='NO_MATERIAL_DECISION'&&f.value==='YES'&&f.applicability==='YES'&&verified(f)));
    const assessment:DecisionAssessment={identity:structuredClone(r.identity),contract:structuredClone(r.contract),projectionBlob,decision:!supported.length?'UNRESOLVED':keep?'KEEP_EXISTING':supported.length>1?'TRADEOFF':'ADVISORY_CANDIDATE',candidates,knowledgeSha:this.#projection.sourceSha,qualification:'ADVISORY_NOT_RULE'};
    // SDK non-COMPLETE has only reason fields; preserve bounded diagnostic assessment there.
    if(!supported.length)return {status:'UNKNOWN',reason:{code:candidates.some(c=>c.reason==='MATERIAL_CONFLICT')?'MATERIAL_CONFLICT':'NO_SUPPORTED_CANDIDATE',message:JSON.stringify(assessment)}};
    return complete(assessment);
  }
  assessContractEvidence(request:unknown, context:Confirmation|undefined, limits:BoundedQueryLimits={maxVisited:19,maxReturned:20}):Outcome<EvidenceAssessment>{
    const validContext=context&&typeof request==='object'&&request!==null&&sameIdentity((request as ValidationRequest).identity,context.identity)?context:undefined;
    const denied=admitValidation(request,validContext);if(denied)return denied;
    const r=request as ValidationRequest, universe=validContext?.universe;
    const checked=validContext?.observations??[];
    if(checked.length>128||checked.reduce((n,c)=>n+c.evidence.length,0)>128||checked.reduce((n,c)=>n+c.record.evidenceRefs.length,0)>128)return stop('REJECT','INPUT_LIMIT_EXCEEDED');
    const k=this.#knowledge(true,limits);if(k.status!=='COMPLETE')return k;
    const authorized=!!universe&&universe.candidateId===r.candidateId&&universe.contract.status==='CONFIRMED'&&sameContract(universe.contract,validContext?.contract);
    const obligations=authorized&&validContext&&universe?universe.obligations.filter(validObligation).map(o=>evaluateObligation(o,r,validContext)):[];
    const completeUniverse=authorized&&universe?.complete===true&&obligations.length>0&&obligations.length===universe.obligations.length;
    const status=obligations.some(o=>o.state==='OBSERVED_FAIL')?'PUBLIC_CONTRACT_PARTIAL_OR_FAILED':!completeUniverse||obligations.some(o=>o.state==='UNKNOWN'||o.state==='CONFLICT')?'UNKNOWN':obligations.some(o=>o.state==='NOT_RUN'||o.state==='STALE')?'INCOMPLETE':'TESTED_CONTRACTS_PASS_SCOPED';
    return complete({identity:structuredClone(r.identity),candidateId:r.candidateId,projectionBlob,knowledgeSha:this.#projection.sourceSha,status,obligations,completeUniverse,knowledge:k.value});
  }
  async run(request:unknown, context:Confirmation|undefined, limits:WorkflowLimits, queryLimits:BoundedQueryLimits={maxVisited:19,maxReturned:20}, validation?:ValidationRequest):Promise<Outcome<{decision:DecisionAssessment;validation?:EvidenceAssessment}>>{
    const input=structuredClone(request),checked=structuredClone(context), observation=structuredClone(validation),query=structuredClone(queryLimits);
    return new StandardWorkflowSession(limits).run(async root=>{
      const decision=await root.invokeOutcome(()=>this.assessAbstraction(input,checked,query));
      if(decision.status!=='COMPLETE')return decision;
      if(observation===undefined)return complete({decision:decision.value});
      if(!sameIdentity(observation.identity,(input as DecisionRequest).identity)||!(input as DecisionRequest).candidates.some(c=>c.id===observation.candidateId))return stop('REJECT','EVIDENCE_REFERENCE_INVALID');
      const result=await root.invokeOutcome(()=>this.assessContractEvidence(observation,checked,query));
      if(result.status!=='COMPLETE')return result;
      return complete({decision:decision.value,validation:result.value});
    });
  }
}
