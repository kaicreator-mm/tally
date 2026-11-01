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
