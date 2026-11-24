import { isDeepStrictEqual } from 'node:util';
import type { StandardWorkflowOutcome } from '@kaicreator/domain-harness';

export const dimensions = ['PRECONDITION','POSTCONDITION','INVARIANT','ERROR_SET','SIDE_EFFECT','STATE_OWNER','CLEANUP','VERSIONED_CONSUMER','CHANGE_AXIS'] as const;
const kinds = ['INLINE','PRIVATE_SHARED','DOMAIN_MODULE','PUBLIC_CONTRACT','STRATEGY'] as const;
const values = ['YES','NO','UNKNOWN','CONFLICT'] as const;
const keys = ['CONTRACT_PRESERVED','LOCAL_ONLY','NO_MATERIAL_DECISION','MULTIPLE_MAINTENANCE_SITES','SAME_OWNER','SHARED_INVARIANT','SHARED_CHANGE','INDEPENDENT_BOUNDARY','STABLE_CONSUMERS','INDEPENDENT_RELEASE','STABLE_REPLACEMENT','AUTHORIZED_REPLACEMENT','EXISTING_COMPATIBILITY','PRESERVES_DIFFERENCES',...dimensions] as const;
export type Value = typeof values[number];
export type Key = typeof keys[number];
export type Identity = { repo: string; issue: number; sourceSha: string; decisionId: string };
export type Contract = { id: string; locator: string; version: string; status: 'CONFIRMED'|'MISSING'|'STALE'|'CONFLICT' };
export type Evidence = { id: string; subject: string; sourceSha: string; locator: string };
export type Fact = { id: string; candidateId: string; key: Key; value: Value; applicability: Value; evidenceRefs: string[] };
export type Candidate = { id: string; kind: typeof kinds[number]; merge: 'NONE'|'BUSINESS'|'PURE'; existing: boolean; description: string };
export type DecisionRequest = { identity: Identity; contract: Contract; candidates: Candidate[]; facts: Fact[]; evidence: Evidence[] };
export type Obligation = { id: string; locator: string; version: string; scope: {id:string;target:string;environment:string}[]; expected: string };
export type Observation = { id: string; obligationId: string; candidateId: string; sourceSha: string; oracle: 'DECLARED'|'DERIVED'|'ENGINEERING_HYPOTHESIS'; expectation: string; derivation: string; oracleLocator:string; oracleVersion:string; target:string; environment: string; runId: string; locator: string; execution: 'EXECUTED'|'NOT_RUN'|'UNKNOWN'; result: 'PASS'|'FAIL'|'UNKNOWN'; coverage: string[]; evidenceRefs: string[] };
export type ValidationRequest = { identity: Identity; candidateId: string; observations: Observation[]; evidence: Evidence[] };
/** Constructed only by the trusted composition caller from actually checked records. Not an authentication protocol. */
export type Confirmed<T> = { record: T; evidence: Evidence[] };
export type Confirmation = { identity: Identity; contract?: Contract; candidates?: Candidate[]; facts?: Confirmed<Fact>[]; universe?: { candidateId: string; contract: Contract; complete: boolean; obligations: Obligation[] }; observations?: Confirmed<Observation>[] };
export type Stop = Exclude<StandardWorkflowOutcome<never>, {status:'COMPLETE'}>;
export function stop(status: Stop['status'], code: string): Stop { return {status,reason:{code,message:code}}; }
export function complete<T>(value: T): StandardWorkflowOutcome<T> { return {status:'COMPLETE',value}; }
export const isRecord = (v: unknown): v is Record<string, unknown> => v !== null && typeof v === 'object' && !Array.isArray(v);
const str = (v: unknown): v is string => typeof v === 'string' && v.trim().length > 0;
const sha = (v: unknown): boolean => typeof v === 'string' && /^[a-f0-9]{40}$/.test(v);
const strings = (v: unknown): v is string[] => Array.isArray(v) && v.every(str);
const inSet = (v: unknown, set: readonly string[]): boolean => typeof v === 'string' && set.includes(v);
const unique = (a: readonly string[]) => new Set(a).size === a.length;
export function validIdentity(v: unknown): v is Identity { return isRecord(v) && str(v.repo) && Number.isSafeInteger(v.issue) && Number(v.issue)>0 && sha(v.sourceSha) && str(v.decisionId); }
const contractShape = (v: unknown): boolean => isRecord(v) && str(v.id) && str(v.locator) && str(v.version) && str(v.status);
const evidenceShape = (v: unknown): v is Evidence => isRecord(v) && str(v.id) && str(v.subject) && sha(v.sourceSha) && str(v.locator);
const factShape = (v: unknown): v is Fact => isRecord(v) && str(v.id) && str(v.candidateId) && str(v.key) && str(v.value) && str(v.applicability) && strings(v.evidenceRefs);
const candidateShape = (v: unknown): v is Candidate => isRecord(v) && str(v.id) && str(v.kind) && str(v.merge) && typeof v.existing==='boolean' && str(v.description);
const observationShape = (v: unknown): v is Observation => isRecord(v) && ['id','obligationId','candidateId','oracle','expectation','oracleLocator','oracleVersion','target','environment','runId','locator','execution','result'].every(k=>str(v[k])) && sha(v.sourceSha) && typeof v.derivation==='string' && strings(v.coverage) && strings(v.evidenceRefs);
const reject = (code: string) => stop('REJECT',code);
function references(records: readonly (Fact|Observation)[], evidence: Evidence[]): boolean {
  const byId = new Map(evidence.map(e=>[e.id,e]));
  return records.every(r=>r.evidenceRefs.every(id=>byId.get(id)?.subject===r.id));
}
function tooManyRefs(records: unknown[]): boolean {
  let count=0;
  for(const r of records) if(isRecord(r) && Array.isArray(r.evidenceRefs)) { count+=r.evidenceRefs.length; if(count>128)return true; }
  return false;
}
/** Check parsed business data; this is not an untrusted network parser. */
export function admitDecision(v: unknown): Stop|undefined {
  if(!isRecord(v)||!validIdentity(v.identity)||!contractShape(v.contract)||!Array.isArray(v.candidates)||!Array.isArray(v.facts)||!Array.isArray(v.evidence))return reject('INVALID_INPUT');
  const {candidates,facts,evidence}=v;
  if(candidates.length>5||facts.length>64||evidence.length>128||tooManyRefs(facts))return reject('INPUT_LIMIT_EXCEEDED');
  const counts=new Map<string,number>();
  for(const f of facts) if(isRecord(f)&&typeof f.candidateId==='string'&&inSet(f.key,dimensions)) { const n=(counts.get(f.candidateId)??0)+1; if(n>9)return reject('INPUT_LIMIT_EXCEEDED'); counts.set(f.candidateId,n); }
  if(!candidates.every(candidateShape)||!facts.every(factShape)||!evidence.every(evidenceShape))return reject('INVALID_INPUT');
  const contract=v.contract as Contract;
  if(!inSet(contract.status,['CONFIRMED','MISSING','STALE','CONFLICT'])||candidates.some(c=>!inSet(c.kind,kinds)||!inSet(c.merge,['NONE','BUSINESS','PURE']))||facts.some(f=>!inSet(f.key,keys)||!inSet(f.value,values)||!inSet(f.applicability,values)))return reject('UNSUPPORTED_ENUM');
  if(!unique(candidates.map(c=>c.id))||!unique(facts.map(f=>f.id))||!unique(evidence.map(e=>e.id))||!unique(facts.map(f=>JSON.stringify([f.candidateId,f.key]))))return reject('DUPLICATE_ID');
  if(facts.some(f=>!candidates.some(c=>c.id===f.candidateId))||!references(facts,evidence))return reject('EVIDENCE_REFERENCE_INVALID');
}
export function admitValidation(v: unknown, confirmation?: Confirmation): Stop|undefined {
  if(!isRecord(v)||!validIdentity(v.identity)||!str(v.candidateId)||!Array.isArray(v.observations)||!Array.isArray(v.evidence))return reject('INVALID_INPUT');
  const {observations,evidence}=v, obligations=confirmation?.universe?.obligations??[];
  if(obligations.length>64||observations.length>128||evidence.length>128||tooManyRefs(observations))return reject('INPUT_LIMIT_EXCEEDED');
  if(!observations.every(observationShape)||!evidence.every(evidenceShape))return reject('INVALID_INPUT');
  if(observations.some(o=>!inSet(o.oracle,['DECLARED','DERIVED','ENGINEERING_HYPOTHESIS'])||!inSet(o.execution,['EXECUTED','NOT_RUN','UNKNOWN'])||!inSet(o.result,['PASS','FAIL','UNKNOWN'])))return reject('UNSUPPORTED_ENUM');
  if(observations.some(o=>!unique(o.coverage))||!unique(observations.map(o=>o.id))||!unique(evidence.map(e=>e.id))||!unique(obligations.map(o=>o.id)))return reject('DUPLICATE_ID');
  if(observations.some(o=>o.candidateId!==v.candidateId||(confirmation?.universe?.complete===true&&!obligations.some(x=>x.id===o.obligationId)))||!references(observations,evidence))return reject('EVIDENCE_REFERENCE_INVALID');
}
export function confirmed<T extends {id:string;evidenceRefs:string[]}>(record:T, evidence:Evidence[], checked:Confirmed<T>[]|undefined): boolean {
  const records=record.evidenceRefs.map(id=>evidence.find(e=>e.id===id));
  return records.length>0 && records.every(e=>e?.subject===record.id) && !!checked?.some(c=>isDeepStrictEqual(c.record,record)&&isDeepStrictEqual(c.evidence,records));
}
export function validObligation(o: Obligation): boolean {
  return str(o.id)&&str(o.locator)&&str(o.version)&&str(o.expected)&&Array.isArray(o.scope)&&o.scope.length>0&&o.scope.every(s=>isRecord(s)&&str(s.id)&&str(s.target)&&str(s.environment))&&unique(o.scope.map(s=>s.id));
}
export function sameIdentity(a: Identity, b: Identity|undefined): boolean { return isDeepStrictEqual(a,b); }
export function sameContract(a: Contract, b: Contract|undefined): boolean { return isDeepStrictEqual(a,b); }
