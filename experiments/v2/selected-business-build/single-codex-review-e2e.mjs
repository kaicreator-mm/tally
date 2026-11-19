/** #264 J05 — installed single-Builder review consumer entrypoint.
 *
 * Spawned by web/v2/installed-write-journeys.mjs for the J05
 * `/api/actions` (kind=review-repair) dispatch seam. Runs the existing
 * selected chain (run-clean.sh → sealed B→K→C → consumer-single-review)
 * with T264_SINGLE_REVIEW=1, then independently reads back the durable
 * ReviewDecision CAS row (phase, decision, exact bindings) before answering
 * PASS. The Reviewer never launches a process; no GitHub write happens on
 * this lane (the PR #466 candidate evidence stays immutable).
 */
import {spawn} from 'node:child_process';
import {createWriteStream} from 'node:fs';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
import {resolve,join} from 'node:path';

const required=['DH','TALLY','TALLY_B','TALLY_GITHUB_READ_TOKEN',
  'TALLY_CLAIM_SOURCE_SHA','TALLY_LOCAL_REPOSITORY_PATH',
  'TALLY_TASK_CLAIM_264_DISPATCH_ID','TALLY_TASK_CLAIM_264_OPERATOR_ID',
  'TALLY_TASK_CLAIM_264_OPERATOR_KIND','TALLY_TASK_CLAIM_264_WRITER_ID',
  'TALLY_TASK_CLAIM_264_WRITER_LOGIN','TALLY_TASK_CLAIM_264_COMMENT_ID',
  'TALLY_TASK_CLAIM_264_REGISTRY_PATH',
  'TALLY_TASK_CLAIM_264_REVIEWER_DISPATCH_ID',
  'TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_ID',
  'TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_KIND',
  'TALLY_TASK_CLAIM_264_REVIEWER_WRITER_ID',
  'TALLY_TASK_CLAIM_264_REVIEWER_WRITER_LOGIN',
  'TALLY_TASK_CLAIM_264_REVIEWER_COMMENT_ID',
  'TALLY_TASK_CLAIM_264_REVIEWER_REGISTRY_PATH',
  'TALLY_LOCAL_BUILDER_STORE_PATH','TALLY_LOCAL_REVIEW_STORE_PATH',
  'TALLY_LOCAL_ATTEMPT_ID','TALLY_REVIEW_DECISION','TALLY_REVIEW_DECISION_ID',
  'TALLY_REVIEW_CANDIDATE_COMMIT','TALLY_REVIEW_CANDIDATE_TREE',
  'TALLY_REVIEW_REPAIR_COMMIT','TALLY_REVIEW_REPAIR_TREE',
  'TALLY_REVIEW_PR_NUMBER','TALLY_LOCAL_EVIDENCE_DIR'];
const missing=required.filter(name=>!process.env[name]);
if(missing.length)throw new Error('T264_REVIEW_ENV_REQUIRED:'+missing.join(','));
if(!['ACCEPT','REWORK','DEFER'].includes(process.env.TALLY_REVIEW_DECISION))
  throw new Error('T264_REVIEW_DECISION_INVALID');
if(process.platform!=='win32'||Number(process.versions.node.split('.')[0])<24)
  throw new Error('WINDOWS_NODE24_REQUIRED');
const evidence=resolve(process.env.TALLY_LOCAL_EVIDENCE_DIR);
await mkdir(evidence,{recursive:true});
const resultPath=join(evidence,'consumer-result.json');
const logPath=join(evidence,'consumer.log');
const env={...process.env,T04B_NATIVE:'1',T06M_LOCAL:'1',T264_SINGLE_REVIEW:'1',
  TALLY_LOCAL_REPLAY:'0',TALLY_LOCAL_RESULT_PATH:resultPath,
  TALLY_NATIVE_EXPECTED_SHA:process.env.TALLY_CLAIM_SOURCE_SHA,
  npm_config_registry:'https://registry.npmmirror.com'};
const child=spawn('C:\\Program Files\\Git\\bin\\bash.exe',
  ['experiments/v2/selected-business-build/run-clean.sh'],
  {cwd:process.env.TALLY,env,windowsHide:true});
const log=createWriteStream(logPath,{mode:0o600});
const token=process.env.TALLY_GITHUB_READ_TOKEN;
const sanitize=chunk=>chunk.toString('utf8').split(token).join('[REDACTED]');
child.stdout.on('data',chunk=>{const text=sanitize(chunk);log.write(text);process.stdout.write(text);});
child.stderr.on('data',chunk=>{const text=sanitize(chunk);log.write(text);process.stderr.write(text);});
const exit=await new Promise(resolveExit=>{
  child.once('error',()=>resolveExit({code:null}));
  child.once('close',(code,signal)=>resolveExit({code,signal}));
});
log.end();
let summary={status:'FAIL',exitCode:exit.code,signal:exit.signal??null,
  platform:process.platform,node:process.version,lane:'t264-single-review',
  decisionRequested:process.env.TALLY_REVIEW_DECISION,logPath};
if(exit.code===0){
  const result=JSON.parse(await readFile(resultPath,'utf8'));
  const {openSqliteCasStore}=await import(
    '@kaicreator/domain-harness-host-node');
  if(result.result==='NOT_ADMITTED'){
    // Honest fail-closed denial (e.g. the independent reviewer Claim is not
    // published yet): verify NO decision/outcome row was written, then
    // answer PASS for the executed observation itself — the denial reason is
    // the recorded evidence, never a fabricated decision.
    const store=await openSqliteCasStore({path:process.env.TALLY_LOCAL_REVIEW_STORE_PATH,
      busyTimeoutMs:5000});
    let decisionRows,outcomeRows;
    try{decisionRows=await store.list('tally:v2:single-review-decision:');
      outcomeRows=await store.list('tally:v2:single-task-outcome:');}
    finally{await store.close();}
    if(decisionRows.length||outcomeRows.length)
      throw new Error('INDEPENDENT_DENIAL_READBACK_UNEXPECTED_ROW');
    summary={...summary,status:'PASS',realReviewEffect:true,result,
      independentDecisionReadback:{decisionRows:decisionRows.length,
        outcomeRows:outcomeRows.length},
      independentOutcomeReadback:{phase:'ABSENT'}};
  }else{
    // Independent durable readback: the ReviewDecision CAS row must exist in
    // the pinned review store with the exact recorded decision and identity
    // bindings the consumer reported (never trust the process result alone).
    const store=await openSqliteCasStore({path:process.env.TALLY_LOCAL_REVIEW_STORE_PATH,
      busyTimeoutMs:5000});
    let decisionRow;
    try{decisionRow=await store.get(result.decisionKey);}
    finally{await store.close();}
    const value=decisionRow?.value;
    if(!value||value.phase!=='REVIEW_DECISION_RECORDED'||
      value.decision!==result.decision||
      value.sourceSha!==process.env.TALLY_CLAIM_SOURCE_SHA||
      value.candidateCommit!==process.env.TALLY_REVIEW_CANDIDATE_COMMIT||
      value.repairCommit!==process.env.TALLY_REVIEW_REPAIR_COMMIT)
      throw new Error('INDEPENDENT_DECISION_READBACK_FAILED');
    const outcomeKey='tally:v2:single-task-outcome:'+
      result.decisionKey.slice('tally:v2:single-review-decision:'.length);
    const reviewStore=await openSqliteCasStore({path:process.env.TALLY_LOCAL_REVIEW_STORE_PATH,
      busyTimeoutMs:5000});
    let outcomeRow;
    try{outcomeRow=await reviewStore.get(outcomeKey);}
    finally{await reviewStore.close();}
    if(result.taskOutcomeStatus==='QUALIFIED_TASK_OUTCOME_PUBLISHED'){
      if(outcomeRow?.value?.phase!=='QUALIFIED_TASK_OUTCOME_PUBLISHED')
        throw new Error('INDEPENDENT_OUTCOME_READBACK_FAILED');
    }else if(outcomeRow)
      throw new Error('INDEPENDENT_OUTCOME_READBACK_UNEXPECTED_ROW');
    summary={...summary,status:'PASS',realReviewEffect:true,result,
      independentDecisionReadback:{decisionKey:result.decisionKey,
        revision:decisionRow.revision,decision:value.decision,
        reviewerOperatorId:value.reviewer?.operatorId??null},
      independentOutcomeReadback:{outcomeKey,
        phase:outcomeRow?.value?.phase??'ABSENT'}};
  }
}
await writeFile(join(evidence,'e2e-result.json'),JSON.stringify(summary,null,2)+'\n',{mode:0o600});
console.log(JSON.stringify({status:summary.status,exitCode:exit.code,
  lane:'t264-single-review',result:summary.result?.result??null,
  decision:summary.result?.decision??null,decisionKey:summary.result?.decisionKey??null,
  taskOutcomeStatus:summary.result?.taskOutcomeStatus??null,
  taskPass:summary.result?.taskPass??false,evidence}));
if(exit.code!==0)process.exitCode=exit.code??1;
