import {spawn,execFile} from 'node:child_process';
import {createWriteStream} from 'node:fs';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
import {resolve,join,dirname} from 'node:path';
import {promisify} from 'node:util';
const execFileAsync=promisify(execFile);
const required=['DH','TALLY','TALLY_B','TALLY_GITHUB_READ_TOKEN','TALLY_CLAIM_SOURCE_SHA',
  'TALLY_CLAIM_DISPATCH_ID','TALLY_CLAIM_OPERATOR_ID','TALLY_CLAIM_OPERATOR_KIND',
  'TALLY_CLAIM_WRITER_ID','TALLY_CLAIM_WRITER_LOGIN','TALLY_CLAIM_COMMENT_ID',
  'TALLY_CLAIM_REGISTRY_PATH','TALLY_LOCAL_WRITESET_SHA256','TALLY_LOCAL_ATTEMPT_ID',
  'TALLY_LOCAL_REPOSITORY_PATH','TALLY_LOCAL_WORKTREE_PATH','TALLY_CODEX_CLI_PATH',
  'TALLY_CODEX_SANDBOX_PATH','TALLY_BUILDER_PROFILE','TALLY_BUILDER_TEMP',
  'TALLY_EMPTY_GIT_CONFIG','TALLY_OPERATOR_HOME','TALLY_HIDDEN_FIXTURE','TALLY_BUILDER_PRINCIPAL',
  'TALLY_BUILDER_SID','TALLY_VERIFIER_SID','TALLY_LOCAL_EVIDENCE_DIR'];
const missing=required.filter(name=>!process.env[name]);
if(missing.length)throw new Error('T06M_ENV_REQUIRED:'+missing.join(','));
if(process.platform!=='win32'||Number(process.versions.node.split('.')[0])<24)
  throw new Error('WINDOWS_NODE24_REQUIRED');
const evidence=resolve(process.env.TALLY_LOCAL_EVIDENCE_DIR);
await mkdir(evidence,{recursive:true});
await mkdir(dirname(process.env.TALLY_HIDDEN_FIXTURE),{recursive:true});
await mkdir(process.env.TALLY_BUILDER_TEMP,{recursive:true});
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
for(const path of [process.env.TALLY_BUILDER_TEMP,
  ...(process.env.TALLY_CODEX_HOME?[process.env.TALLY_CODEX_HOME]:[])])
  await acl(path,['/grant',`*${process.env.TALLY_BUILDER_SID}:(OI)(CI)(F)`]);
await acl(process.env.TALLY_EMPTY_GIT_CONFIG,['/grant',`*${process.env.TALLY_BUILDER_SID}:(R)`]);
const resultPath=join(evidence,'consumer-result.json');
const logPath=join(evidence,'consumer.log');
const env={...process.env,T04B_NATIVE:'1',T06M_LOCAL:'1',TALLY_LOCAL_REPLAY:'0',
  TALLY_LOCAL_REQUIRE_CANDIDATE:'1',TALLY_NATIVE_EXPECTED_SHA:process.env.TALLY_CLAIM_SOURCE_SHA,
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
  platform:process.platform,node:process.version,realAgentEffect:false,hiddenPass:false,logPath};
if(exit.code===0){
  const result=JSON.parse(await readFile(resultPath,'utf8'));
  const git=process.env.TALLY_GIT_PATH??'C:\\Program Files\\Git\\cmd\\git.exe';
  const worktree=process.env.TALLY_LOCAL_WORKTREE_PATH;
  const [parent,tree,paths]=await Promise.all([
    execFileAsync(git,['-C',worktree,'rev-parse',result.candidateCommit+'^']),
    execFileAsync(git,['-C',worktree,'rev-parse',result.candidateCommit+'^{tree}']),
    execFileAsync(git,['-C',worktree,'diff','--name-only',process.env.TALLY_CLAIM_SOURCE_SHA,result.candidateCommit])
  ]);
  if(parent.stdout.trim()!==process.env.TALLY_CLAIM_SOURCE_SHA||
    tree.stdout.trim()!==result.candidateTree||
    paths.stdout.trim()!=='experiments/v2/selected-business-build/LOCAL_CODEX_L3.md')
    throw new Error('INDEPENDENT_CANDIDATE_READBACK_FAILED');
  summary={...summary,status:'PASS',realAgentEffect:true,result,
    independentCandidateReadback:{parent:parent.stdout.trim(),tree:tree.stdout.trim(),
      paths:paths.stdout.trim().split(/\r?\n/).filter(Boolean)},hiddenVerifier:'HIDDEN_NOT_QUALIFIED'};
}
await writeFile(join(evidence,'e2e-result.json'),JSON.stringify(summary,null,2)+'\n',{mode:0o600});
console.log(JSON.stringify({status:summary.status,exitCode:exit.code,result:summary.result?.result??null,
  candidateTree:summary.result?.candidateTree??null,candidateCommit:summary.result?.candidateCommit??null,
  realAgentEffect:summary.realAgentEffect,hiddenPass:false,evidence}));
if(exit.code!==0)process.exitCode=exit.code??1;
