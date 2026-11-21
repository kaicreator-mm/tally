/** Linux selected Host physical E2E. One BUILD-selected K/C, one real Agent.
 * Consumes a freshly built Linux-native sealed app, never the Windows ZIP.
 * No live effect is claimed without exact post-process git/tree readback.
 */
import {spawn,execFile} from 'node:child_process';
import {createWriteStream} from 'node:fs';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
import {resolve,join,dirname} from 'node:path';
import {promisify} from 'node:util';
import {readPrebuiltConsumer} from './prebuilt-consumer.mjs';
const execFileAsync=promisify(execFile);
const prebuiltDir=process.env.TALLY_V2_AGENT_PREBUILT_CONSUMER_DIR;
const env=process.env;
if(process.platform!=='linux'||env.TALLY_V2_AGENT_PLATFORM!=='linux')
  throw Error('LINUX_HOST_PLATFORM_MISMATCH');
const [maj,min]=process.versions.node.split('.').map(Number);
if(maj<22||(maj===22&&min<16))throw Error('NODE_22_16_REQUIRED');
if(typeof process.getuid==='function'&&process.getuid()===0)
  throw Error('LINUX_HOST_MUST_NOT_RUN_AS_ROOT');
const required=['TALLY_GITHUB_READ_TOKEN','TALLY_CLAIM_SOURCE_SHA',
  'TALLY_CLAIM_DISPATCH_ID','TALLY_CLAIM_OPERATOR_ID','TALLY_CLAIM_OPERATOR_KIND',
  'TALLY_CLAIM_WRITER_ID','TALLY_CLAIM_WRITER_LOGIN','TALLY_CLAIM_COMMENT_ID',
  'TALLY_CLAIM_REGISTRY_PATH','TALLY_LOCAL_WRITESET_SHA256','TALLY_LOCAL_ATTEMPT_ID',
  'TALLY_LOCAL_REPOSITORY_PATH','TALLY_LOCAL_WORKTREE_PATH','TALLY_CODEX_CLI_PATH',
  'TALLY_LINUX_BWRAP_PATH','TALLY_BUILDER_PROFILE','TALLY_BUILDER_TEMP',
  'TALLY_EMPTY_GIT_CONFIG','TALLY_OPERATOR_HOME','TALLY_HIDDEN_FIXTURE',
  'TALLY_BUILDER_PRINCIPAL','TALLY_BUILDER_SID','TALLY_VERIFIER_SID',
  'TALLY_LOCAL_EVIDENCE_DIR','T06M_TARGET_REPOSITORY','T06M_TARGET_ISSUE_NUMBER',
  'T06M_TARGET_VERSION_REF','T06M_TARGET_CANDIDATE_PATH',
  'TALLY_V2_AGENT_PREBUILT_CONSUMER_DIR'];
const missing=required.filter(k=>!env[k]);
if(missing.length)throw Error('LINUX_HOST_PINS_MISSING:'+missing.join(','));
if(env.TALLY_BUILDER_PRINCIPAL!=='linux-bwrap:65534'||
   env.TALLY_BUILDER_SID!=='linux-uid:65534'||
   env.TALLY_VERIFIER_SID==='linux-uid:65534')
  throw Error('LINUX_BUILDER_IDENTITY_INVALID');
const prebuilt=await readPrebuiltConsumer({directory:prebuiltDir,env});
if(!prebuilt.manifest.runtimeDigests?.['local-codex-linux-host.mjs'])
  throw Error('LINUX_PREBUILT_HOST_CODE_NOT_SEALED');
const evidence=resolve(env.TALLY_LOCAL_EVIDENCE_DIR);
await mkdir(evidence,{recursive:true,mode:0o700});
await mkdir(dirname(env.TALLY_HIDDEN_FIXTURE),{recursive:true,mode:0o700});
await mkdir(env.TALLY_BUILDER_TEMP,{recursive:true,mode:0o700});
await mkdir(env.TALLY_BUILDER_PROFILE,{recursive:true,mode:0o700});
await writeFile(env.TALLY_HIDDEN_FIXTURE,'held-out-verifier-canary\n',
  {flag:'wx',mode:0o600}).catch(err=>{if(err.code!=='EEXIST')throw err;});
await writeFile(env.TALLY_EMPTY_GIT_CONFIG,'',{flag:'wx',mode:0o600})
  .catch(err=>{if(err.code!=='EEXIST')throw err;});
const resultPath=join(evidence,'consumer-result.json');
const logPath=join(evidence,'consumer.log');
const runEnv={...env,T04B_NATIVE:'1',T06M_LOCAL:'1',TALLY_LOCAL_REPLAY:'0',
  TALLY_LOCAL_REQUIRE_CANDIDATE:'1',TALLY_NATIVE_EXPECTED_SHA:env.TALLY_CLAIM_SOURCE_SHA,
  TALLY_LOCAL_RESULT_PATH:resultPath};
const child=spawn(process.execPath,[prebuilt.script],{
  cwd:prebuilt.dir,env:runEnv,stdio:['ignore','pipe','pipe']});
const log=createWriteStream(logPath,{mode:0o600});
const redact=value=>String(value).split(env.TALLY_GITHUB_READ_TOKEN).join('[REDACTED]');
const onData=chunk=>{
  const safe=redact(chunk.toString('utf8'));
  log.write(safe);process.stdout.write(safe);
};
child.stdout.on('data',onData);
child.stderr.on('data',chunk=>{
  const safe=redact(chunk.toString('utf8'));
  log.write(safe);process.stderr.write(safe);
});
const exit=await new Promise(resolveExit=>{
  child.once('error',()=>resolveExit({code:null,signal:null}));
  child.once('close',(code,signal)=>resolveExit({code,signal}));
});
log.end();
let summary={status:'FAIL',exitCode:exit.code,signal:exit.signal??null,
  platform:'linux',node:process.version,realAgentEffect:false,hiddenPass:false,logPath};
if(exit.code===0){
  const result=JSON.parse(await readFile(resultPath,'utf8'));
  const git=env.TALLY_GIT_PATH??'/usr/bin/git';
  const worktree=env.TALLY_LOCAL_WORKTREE_PATH;
  const [parent,tree,paths]=await Promise.all([
    execFileAsync(git,['-C',worktree,'rev-parse',result.candidateCommit+'^']),
    execFileAsync(git,['-C',worktree,'rev-parse',result.candidateCommit+'^{tree}']),
    execFileAsync(git,['-C',worktree,'diff','--name-only',
      env.TALLY_CLAIM_SOURCE_SHA,result.candidateCommit])
  ]);
  if(parent.stdout.trim()!==env.TALLY_CLAIM_SOURCE_SHA||
     tree.stdout.trim()!==result.candidateTree||
     paths.stdout.trim()!==env.T06M_TARGET_CANDIDATE_PATH)
    throw Error('LINUX_INDEPENDENT_CANDIDATE_READBACK_FAILED');
  summary={...summary,status:'PASS',realAgentEffect:true,result,
    independentCandidateReadback:{parent:parent.stdout.trim(),tree:tree.stdout.trim(),
      paths:paths.stdout.trim().split(/\r?\n/).filter(Boolean)},
    hiddenVerifier:'HIDDEN_NOT_QUALIFIED'};
}
await writeFile(join(evidence,'e2e-result.json'),JSON.stringify(summary,null,2)+'\n',
  {mode:0o600});
console.log(JSON.stringify({status:summary.status,exitCode:exit.code,
  candidateTree:summary.result?.candidateTree??null,
  candidateCommit:summary.result?.candidateCommit??null,
  realAgentEffect:summary.realAgentEffect,hiddenPass:false,evidence}));
if(exit.code!==0)process.exitCode=exit.code??1;
