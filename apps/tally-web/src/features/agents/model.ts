/** UI-C #230 — Agent execution view model: attempts, principals, honest
 * UNKNOWN cost/telemetry. Never ranks agents; never converts UNKNOWN to 0.
 */
export interface AttemptRecord{
  attemptId:string;
  taskIds:readonly string[];
  agentId:string;
  state:'RUNNING'|'EXITED_OK'|'EXITED_FAIL'|'CANCELLED'|'UNKNOWN';
  startedAt:string|null;
  terminatedAt:string|null;
  /** null = telemetry missing (UNKNOWN), NOT zero. */
  durationMs:number|null;
  tokensIn:number|null;
  tokensOut:number|null;
  costUsd:null;
  model:string|null;
  provider:string|null;
  /** Tally-assigned work role; users do not select or message an Agent process. */
  role?:'BUILDER'|'REVIEWER'|'REPAIR'|'VALIDATOR'|'ORCHESTRATOR'|'UNKNOWN'|null;
}
export const UNKNOWN_TEXT='UNKNOWN';
export function durationLabel(attempt:AttemptRecord):string{
  if(attempt.durationMs===null)return UNKNOWN_TEXT;
  const seconds=Math.round(attempt.durationMs/1000);
  return seconds>=60?`${Math.floor(seconds/60)}m${seconds%60}s`:`${seconds}s`;
}
export function tokenLabel(attempt:AttemptRecord,
  kind:'tokensIn'|'tokensOut'):string{
  const value=attempt[kind];
  return value===null?UNKNOWN_TEXT:String(value);
}
/** Coverage honesty: with how many attempts do we have real telemetry? */
export function telemetryCoverage(attempts:readonly AttemptRecord[]){
  const measured=attempts.filter(attempt=>attempt.durationMs!==null).length;
  return {measured,total:attempts.length,
    coverage:attempts.length===0?null:measured/attempts.length};
}
/** Group attempts per agent; UNKNOWN states stay explicit. */
export function byAgent(attempts:readonly AttemptRecord[]){
  const groups=new Map<string,AttemptRecord[]>();
  for(const attempt of attempts)
    groups.set(attempt.agentId,[...(groups.get(attempt.agentId)??[]),attempt]);
  return [...groups.entries()].map(([agentId,list])=>({agentId,attempts:list}));
}
/** Aggregate token usage ONLY over attempts with real values; a coverage
 * denominator is always attached so UNKNOWN is never reported as zero. */
export function tokenTotals(attempts:readonly AttemptRecord[]){
  let tokensIn=0,tokensOut=0,inMeasured=0,outMeasured=0;
  for(const attempt of attempts){
    if(attempt.tokensIn!==null){tokensIn+=attempt.tokensIn;inMeasured++;}
    if(attempt.tokensOut!==null){tokensOut+=attempt.tokensOut;outMeasured++;}
  }
  return {tokensIn,tokensOut,
    inCoverage:attempts.length===0?null:inMeasured/attempts.length,
    outCoverage:attempts.length===0?null:outMeasured/attempts.length};
}
export function stateTone(state:AttemptRecord['state']){
  return state==='EXITED_OK'?'ok':state==='EXITED_FAIL'?'danger'
    :state==='RUNNING'?'info':'warn';
}

const ATTEMPT_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,79}$/;
const IDENTIFIER=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$/;
const TASK_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,31}$/;
/** Credential-shaped values are refused at the UI projection itself; the port
 * layer redacts too, but render-side never assumes upstream sanitization. */
const SECRET=/\b(?:gh[pousr]_[A-Za-z0-9_]{12,}|github_pat_[A-Za-z0-9_]{12,}|sk-[A-Za-z0-9_-]{16,}|Bearer\s+\S+)\b/i;
const ATTEMPT_STATES=new Set<string>([
  'RUNNING','EXITED_OK','EXITED_FAIL','CANCELLED','UNKNOWN']);
const ATTEMPT_ROLES=new Set<string>([
  'BUILDER','REVIEWER','REPAIR','VALIDATOR','ORCHESTRATOR','UNKNOWN']);
const boundedText=(value:unknown,check:(value:string)=>boolean):string|null=>
  typeof value==='string'&&value.length<=80&&check(value)&&!SECRET.test(value)
    ?value:null;
const boundedCount=(value:unknown):number|null=>
  typeof value==='number'&&Number.isSafeInteger(value)&&value>=0&&
    value<=Number.MAX_SAFE_INTEGER/2
    ?value:null;
const boundedTaskIds=(value:unknown):string[]=>{
  if(!Array.isArray(value))return [];
  const seen=new Set<string>();
  for(const entry of value){
    if(typeof entry!=='string'||entry.length>32||SECRET.test(entry)||
       !TASK_ID.test(entry)||seen.has(entry))continue;
    seen.add(entry);
    if(seen.size>=32)break;
  }
  return [...seen];
};
/** UI projection hardening for P2-02: Attempt records are untrusted until the
 * selected Host producer (#275) lands. Every rendered identifier/text field is
 * bounded and secret-redacted here — missing/unsafe values degrade to
 * UNKNOWN/null instead of passing through. */
export function sanitizeAttempts(attempts:readonly AttemptRecord[]):
  AttemptRecord[]{
  return attempts.map(attempt=>{
    const state=ATTEMPT_STATES.has(attempt.state)
      ?attempt.state:'UNKNOWN';
    // Absent role stays absent (optional field); present-but-invalid degrades
    // to null rather than being inferred.
    const role=attempt.role===undefined?undefined
      :attempt.role!==null&&ATTEMPT_ROLES.has(attempt.role)?attempt.role:null;
    return {
      ...attempt,
      attemptId:boundedText(attempt.attemptId,ATTEMPT_ID.test.bind(ATTEMPT_ID))
        ??UNKNOWN_TEXT,
      agentId:boundedText(attempt.agentId,IDENTIFIER.test.bind(IDENTIFIER))
        ??UNKNOWN_TEXT,
      state,
      role,
      startedAt:boundedText(attempt.startedAt,()=>true),
      terminatedAt:boundedText(attempt.terminatedAt,()=>true),
      durationMs:boundedCount(attempt.durationMs),
      tokensIn:boundedCount(attempt.tokensIn),
      tokensOut:boundedCount(attempt.tokensOut),
      costUsd:null,
      model:attempt.model===null?null
        :boundedText(attempt.model,IDENTIFIER.test.bind(IDENTIFIER)),
      provider:attempt.provider===null?null
        :boundedText(attempt.provider,IDENTIFIER.test.bind(IDENTIFIER)),
      taskIds:boundedTaskIds(attempt.taskIds),
    };
  });
}
