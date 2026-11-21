/** ZCode provider real-execution driver (#64 v2.1 candidate). Host-local on
 * Windows with the real installed ZCode CLI; NOT part of the Docker CI lanes
 * (mirrors local-codex-windows-e2e.mjs). Creates a one-off disposable Git
 * worktree at the pinned source SHA, runs one minimal code-modification task
 * through web/v2/zcode-provider.mjs with the default (non-injected)
 * operations, re-derives the candidate from git independently, and writes
 * evidence/e2e-result.json. Exit 0 only when a real candidate was recorded
 * and verified; anything else is FAIL/UNKNOWN, never a fabricated PASS. */
import {readdirSync,existsSync} from 'node:fs';
import {mkdir,writeFile} from 'node:fs/promises';
import {resolve,join} from 'node:path';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {createZcodeProviderHost} from '../../../web/v2/zcode-provider.mjs';
const execFileAsync=promisify(execFile);

const required=['TALLY_LOCAL_REPOSITORY_PATH','TALLY_ZCODE_ATTEMPT_ID',
  'TALLY_ZCODE_WORKTREE_PATH','TALLY_LOCAL_EVIDENCE_DIR'];
const missing=required.filter(name=>!process.env[name]);
if(missing.length)throw new Error('ZCODE_E2E_ENV_REQUIRED:'+missing.join(','));
if(process.platform!=='win32'||Number(process.versions.node.split('.')[0])<24)
  throw new Error('WINDOWS_NODE24_REQUIRED');

function resolveZcodeCli(){
  if(process.env.TALLY_ZCODE_CLI_PATH)return resolve(process.env.TALLY_ZCODE_CLI_PATH);
  const releasesRoot=join(process.env.USERPROFILE,'.zcode','runtime','releases');
  const newest=readdirSync(releasesRoot).filter(name=>/^\d/.test(name)).sort((a,b)=>a.localeCompare(b,'en',{numeric:true})).at(-1);
  if(!newest)throw new Error('ZCODE_RUNTIME_RELEASE_NOT_FOUND');
  return join(releasesRoot,newest,'bin','zcode.mjs');
}
const cliPath=resolveZcodeCli();
if(!existsSync(cliPath))throw new Error('ZCODE_CLI_NOT_FOUND:'+cliPath);

const repositoryPath=resolve(process.env.TALLY_LOCAL_REPOSITORY_PATH);
const evidence=resolve(process.env.TALLY_LOCAL_EVIDENCE_DIR);
const worktreePath=resolve(process.env.TALLY_ZCODE_WORKTREE_PATH);
const attemptId=process.env.TALLY_ZCODE_ATTEMPT_ID;
const gitPath=process.env.TALLY_GIT_PATH??'C:\\Program Files\\Git\\cmd\\git.exe';
const sourceSha=process.env.TALLY_ZCODE_SOURCE_SHA??
  (await execFileAsync(gitPath,['-C',repositoryPath,'rev-parse','HEAD'])).stdout.trim();
await mkdir(evidence,{recursive:true});
const emptyGitConfig=join(evidence,'empty-gitconfig');
await writeFile(emptyGitConfig,'',{flag:'wx'}).catch(error=>{if(error.code!=='EEXIST')throw error;});
if(existsSync(worktreePath))throw new Error('ZCODE_E2E_WORKTREE_MUST_BE_FRESH');

const candidateRelativePath='experiments/v2/selected-business-build/LOCAL_ZCODE_L3.md';
const host=createZcodeProviderHost({});
const config={issueRef:'https://github.com/kaicreator-mm/tally/issues/64',
  sourceSha,attemptId,repositoryPath,worktreePath,cliPath,nodePath:process.execPath,
  emptyGitConfig,allowedWriteSet:[candidateRelativePath],
  taskPrompt:`Create ${candidateRelativePath} with a short Markdown candidate note containing attempt ${attemptId}.`,
  timeoutMs:Number(process.env.TALLY_ZCODE_TIMEOUT_MS??20*60*1000),
  canaryTimeoutMs:Number(process.env.TALLY_ZCODE_CANARY_TIMEOUT_MS??300000),
  systemRoot:process.env.SystemRoot??'C:\\Windows',systemDrive:process.env.SystemDrive??'C:'};

const startedAt=new Date().toISOString();
const probe=await host.probeProvider(config);
console.error('ZCODE_E2E_PROBE',JSON.stringify(probe));
let attempt=null;
if(probe.available){
  attempt=await host.runAttempt(config);
  console.error('ZCODE_E2E_ATTEMPT',JSON.stringify({...attempt,stdoutTail:undefined,stderrTail:undefined}));
  if(attempt.stdoutTail)await writeFile(join(evidence,'agent-stdout-tail.txt'),attempt.stdoutTail);
  if(attempt.stderrTail)await writeFile(join(evidence,'agent-stderr-tail.txt'),attempt.stderrTail);
}
let readback=null,joinState=null;
if(attempt?.state==='LOCAL_CANDIDATE_READY'&&attempt.candidateCommit){
  // Independent re-derivation straight from git, never from provider memory.
  const verified=await host.joinVerifyCandidate(config,{attemptId,candidateCommit:attempt.candidateCommit});
  joinState=verified.state;
  readback=verified.readback;
}
const pass=attempt?.state==='LOCAL_CANDIDATE_READY'&&joinState==='CANDIDATE_READBACK_VERIFIED';
const summary={status:pass?'PASS':'FAIL',provider:'zcode',startedAt,
  observedAt:new Date().toISOString(),platform:process.platform,node:process.version,
  cliPath,cliVersion:probe.version??null,probe,attempt:attempt?{state:attempt.state,
    reason:attempt.reason,agentLaunch:attempt.agentLaunch,process:attempt.process,
    agent:attempt.agent??null,candidateTree:attempt.candidateTree,
    candidateCommit:attempt.candidateCommit,candidateRef:attempt.candidateRef??null,
    candidatePaths:attempt.candidatePaths??null}:null,
  independentReadback:readback,joinState,
  realAgentEffect:pass,taskPass:false,hiddenVerifier:'HIDDEN_NOT_QUALIFIED'};
await writeFile(join(evidence,'e2e-result.json'),JSON.stringify(summary,null,2)+'\n');
console.log(JSON.stringify({status:summary.status,provider:'zcode',state:attempt?.state??null,
  joinState,candidateCommit:attempt?.candidateCommit??null,candidateTree:attempt?.candidateTree??null,
  realAgentEffect:summary.realAgentEffect,evidence}));
if(!pass)process.exitCode=1;
