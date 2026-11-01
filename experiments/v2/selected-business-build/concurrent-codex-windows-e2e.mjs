/** T06n real Windows driver: two overlapping real Codex Builders, reviewer
 *  Join, bounded Repair and re-Join through run-clean. Executes NO replay;
 *  all evidence values are written by the real run for controller review.
 *  Controller-operated only: this file is NOT executed in Docker CI. */
import {spawn,execFile} from 'node:child_process';
import {createWriteStream} from 'node:fs';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
import {resolve,join,dirname} from 'node:path';
import {promisify} from 'node:util';
const execFileAsync=promisify(execFile);
const claimIssues=[process.env.TALLY_CONCURRENT_ISSUE_A??'177',
  process.env.TALLY_CONCURRENT_ISSUE_B??'178',process.env.TALLY_CONCURRENT_ISSUE_REVIEWER??'179'];
const claimEnv=issue=>['REGISTRY_PATH','DISPATCH_ID','OPERATOR_ID','OPERATOR_KIND','WRITER_ID',
  'WRITER_LOGIN','COMMENT_ID',...(issue===claimIssues[2]?[]:['WRITESET_SHA256'])]
  .map(field=>`TALLY_TASK_CLAIM_${issue}_${field}`);
const required=['DH','TALLY','TALLY_B','TALLY_GITHUB_READ_TOKEN','TALLY_CLAIM_SOURCE_SHA',
  ...claimIssues.flatMap(claimEnv),
  'TALLY_LOCAL_HOST_STORE_PATH','TALLY_LOCAL_REPOSITORY_PATH',
  'TALLY_LOCAL_WORKTREE_ROOT','TALLY_CODEX_CLI_PATH','TALLY_CODEX_SANDBOX_PATH',
  'TALLY_BUILDER_PROFILE','TALLY_BUILDER_TEMP','TALLY_EMPTY_GIT_CONFIG',
  'TALLY_OPERATOR_HOME','TALLY_HIDDEN_FIXTURE','TALLY_BUILDER_PRINCIPAL',
  'TALLY_BUILDER_SID','TALLY_VERIFIER_SID','TALLY_LOCAL_EVIDENCE_DIR'];
const missing=required.filter(name=>!process.env[name]);
if(missing.length)throw new Error('T06N_ENV_REQUIRED:'+missing.join(','));
if(process.platform!=='win32'||Number(process.versions.node.split('.')[0])<24)
  throw new Error('WINDOWS_NODE24_REQUIRED');
const evidence=resolve(process.env.TALLY_LOCAL_EVIDENCE_DIR);
await mkdir(evidence,{recursive:true});
await mkdir(dirname(process.env.TALLY_HIDDEN_FIXTURE),{recursive:true});
await mkdir(process.env.TALLY_BUILDER_TEMP,{recursive:true});
await mkdir(process.env.TALLY_LOCAL_WORKTREE_ROOT,{recursive:true});
await writeFile(process.env.TALLY_HIDDEN_FIXTURE,'held-out-verifier-canary\n',{flag:'wx',mode:0o600}).catch(error=>{
  if(error.code!=='EEXIST')throw error;
});
await writeFile(process.env.TALLY_EMPTY_GIT_CONFIG,'',{flag:'wx',mode:0o600}).catch(error=>{
  if(error.code!=='EEXIST')throw error;
});
const icacls='C:\\Windows\\System32\\icacls.exe';
const acl=async(path,args)=>{
  const result=await execFileAsync(icacls,[path,...args],{windowsHide:true});
  return result.stdout;
};
await acl(process.env.TALLY_HIDDEN_FIXTURE,['/inheritance:d']);
await acl(process.env.TALLY_HIDDEN_FIXTURE,['/grant:r',`*${process.env.TALLY_VERIFIER_SID}:(R)`]);
await acl(process.env.TALLY_HIDDEN_FIXTURE,['/deny',`*${process.env.TALLY_BUILDER_SID}:(R,W)`]);
for(const path of [process.env.TALLY_BUILDER_TEMP,process.env.TALLY_LOCAL_WORKTREE_ROOT,
  ...(process.env.TALLY_CODEX_HOME?[process.env.TALLY_CODEX_HOME]:[])])
  await acl(path,['/grant',`*${process.env.TALLY_BUILDER_SID}:(OI)(CI)(F)`]);
await acl(process.env.TALLY_EMPTY_GIT_CONFIG,['/grant',`*${process.env.TALLY_BUILDER_SID}:(R)`]);
const resultPath=join(evidence,'consumer-result.json');
const logPath=join(evidence,'consumer.log');
const env={...process.env,T04B_NATIVE:'1',T06M_LOCAL:'1',T06N_CONCURRENT:'1',
  TALLY_LOCAL_REPLAY:'0',TALLY_LOCAL_REQUIRE_CANDIDATE:'1',
  TALLY_NATIVE_EXPECTED_SHA:process.env.TALLY_CLAIM_SOURCE_SHA,
  TALLY_LOCAL_RESULT_PATH:resultPath,npm_config_registry:'https://registry.npmmirror.com'};
const child=spawn('C:\\Program Files\\Git\\bin\\bash.exe',
  ['experiments/v2/selected-business-build/run-clean.sh'],{cwd:process.env.TALLY,env,windowsHide:true});
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
  platform:process.platform,node:process.version,realAgentEffect:false,realOverlap:false,
  hiddenPass:false,logPath};
if(exit.code===0){
  const result=JSON.parse(await readFile(resultPath,'utf8'));
  const git=process.env.TALLY_GIT_PATH??'C:\\Program Files\\Git\\cmd\\git.exe';
  const readbacks=await Promise.all([
    execFileAsync(git,['-C',process.env.TALLY_LOCAL_REPOSITORY_PATH,'rev-parse',
      result.candidateA.commit+'^']).catch(()=>null),
    execFileAsync(git,['-C',process.env.TALLY_LOCAL_REPOSITORY_PATH,'rev-parse',
      result.candidateA.commit+'^{tree}']).catch(()=>null),
    execFileAsync(git,['-C',process.env.TALLY_LOCAL_REPOSITORY_PATH,'rev-parse',
      result.candidateB.commit+'^{tree}']).catch(()=>null),
  ]);
  const [parentA,treeA,treeB]=readbacks;
  if(parentA?.stdout.trim()!==process.env.TALLY_CLAIM_SOURCE_SHA||
    treeA?.stdout.trim()!==result.candidateA.tree||
    treeB?.stdout.trim()!==result.candidateB.tree)
    throw new Error('INDEPENDENT_CANDIDATE_READBACK_FAILED');
  if(!result.overlap?.verified||result.overlap.overlapMs===null||result.overlap.overlapMs<=0)
    throw new Error('REAL_PROCESS_OVERLAP_REQUIRED');
  summary={...summary,status:'PASS',realAgentEffect:true,realOverlap:true,result,
    independentCandidateReadback:{parent:parentA.stdout.trim(),treeA:treeA.stdout.trim(),
      treeB:treeB.stdout.trim()},
    overlapMs:result.overlap.overlapMs,hiddenVerifier:'HIDDEN_NOT_QUALIFIED'};
}
await writeFile(join(evidence,'e2e-result.json'),JSON.stringify(summary,null,2)+'\n',{mode:0o600});
console.log(JSON.stringify({status:summary.status,exitCode:exit.code,
  realAgentEffect:summary.realAgentEffect,realOverlap:summary.realOverlap??false,
  candidateA:summary.result?.candidateA??null,candidateB:summary.result?.candidateB??null,
  join:summary.result?.join?.state??null,rejoin:summary.result?.rejoin?.state??null,
  overlapMs:summary.overlapMs??null,hiddenPass:false,evidence}));
if(exit.code!==0)process.exitCode=exit.code??1;
