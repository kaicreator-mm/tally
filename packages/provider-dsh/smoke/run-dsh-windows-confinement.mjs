/**
 * REAL Windows-principal confinement verification for provider-dsh.
 *
 * Runs the DSH host's DEFAULT operations (probe, worktree, confinement,
 * finalize, readback — no injected stubs) against the real
 * @deepseek-ai/dsh CLI under the Codex Windows sandbox wrapper's elevated
 * backend: the confined builder principal executes every step, including one
 * real headless model turn that writes the candidate note inside the
 * disposable worktree.
 *
 * Gates (all required): TALLY_DSH_SMOKE_CONFIRM=1, DEEPSEEK_API_KEY (never
 * printed), TALLY_DSH_BIN (dsh lib entry), win32. Machine defaults can be
 * overridden via the listed env vars.
 *
 * Stage 1 (no API cost): probe (version pin inside the sandbox) + the full
 * confinement matrix on a scratch worktree.
 * Stage 2 (one real model call): a complete runAttempt with candidate
 * finalize and independent readback.
 */
import {spawn,execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {mkdtemp,rm,mkdir,writeFile,readFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createLocalDshHost,createDefaultDshOperations,LOCAL_DSH_CONSTANTS}
  from '../src/dsh-host.mjs';
import {createMemoryCasStoreRegistry} from '../test/memory-cas-store.mjs';

const execFileAsync=promisify(execFile);
const BIN=process.env.TALLY_DSH_BIN;
const NODE=process.env.TALLY_DSH_NODE??process.execPath;
const SANDBOX=process.env.TALLY_DSH_SANDBOX_CLI??'C:\\Users\\15549\\AppData\\Roaming\\fnm\\node-versions\\v24.21.0\\installation\\node_modules\\@openai\\codex\\node_modules\\@openai\\codex-win32-x64\\vendor\\x86_64-pc-windows-msvc\\bin\\codex.exe';
const BUILDER_PRINCIPAL=process.env.TALLY_DSH_BUILDER_PRINCIPAL??'nilar\\codexsandboxonline';
const BUILDER_SID=process.env.TALLY_DSH_BUILDER_SID??'S-1-5-21-2839165489-1832998833-1574237796-1004';
const VERIFIER_SID=process.env.TALLY_DSH_VERIFIER_SID??'S-1-5-21-2839165489-1832998833-1574237796-1003';
const OPERATOR_HOME=process.env.TALLY_DSH_OPERATOR_HOME??'C:\\Users\\15549';
const GIT='C:\\Program Files\\Git\\cmd\\git.exe';
const icacls='C:\\Windows\\System32\\icacls.exe';
const receipt={REAL_DSH_WINDOWS_CONFINEMENT:'NOT_RUN'};
try{
  if(process.platform!=='win32')throw new Error('WIN32_REQUIRED');
  if(process.env.TALLY_DSH_SMOKE_CONFIRM!=='1')throw new Error('CONFIRM_GATE_MISSING');
  const key=process.env.DEEPSEEK_API_KEY;
  if(!key)throw new Error('DEEPSEEK_API_KEY_MISSING');
  if(!BIN)throw new Error('TALLY_DSH_BIN_MISSING');
  const secrets=[key];
  const base=await mkdtemp(join(tmpdir(),'tally-dsh-confine-'));
  const evidence=[];
  try{
    // Shared Host fixtures (mirrors local-codex-windows-e2e.mjs discipline).
    const scratchCodexHome=join(base,'codex-home');
    await mkdir(scratchCodexHome,{recursive:true});
    await writeFile(join(scratchCodexHome,'config.toml'),'[windows]\nsandbox = "elevated"\n','utf8');
    const hiddenFixture=join(base,'hidden-canary.txt');
    await writeFile(hiddenFixture,'held-out-verifier-canary\n',{flag:'wx',mode:0o600}).catch(e=>{
      if(e.code!=='EEXIST')throw e;});
    const emptyGitConfig=join(base,'empty-gitconfig');
    await writeFile(emptyGitConfig,'',{flag:'wx',mode:0o600}).catch(()=>{});
    const builderTemp=join(base,'builder-tmp');
    await mkdir(builderTemp,{recursive:true});
    const acl=async(path,args)=>execFileAsync(icacls,[path,...args],{windowsHide:true});
    await acl(hiddenFixture,['/inheritance:d']);
    await acl(hiddenFixture,['/grant:r',`*${VERIFIER_SID}:(R)`]);
    await acl(hiddenFixture,['/deny',`*${BUILDER_SID}:(R,W)`]);
    for(const path of [builderTemp,scratchCodexHome])
      await acl(path,['/grant',`*${BUILDER_SID}:(OI)(CI)(F)`]);
    await acl(emptyGitConfig,['/grant',`*${BUILDER_SID}:(R)`]);

    const spawnFn=(...a)=>spawn(...a);
    const ops=createDefaultDshOperations(spawnFn);
    const registry=createMemoryCasStoreRegistry();
    const host=createLocalDshHost({spawnFn,openStore:registry.open});
    const commonConfig={issueRef:'https://github.com/kaicreator-mm/tally/issues/64',
      hostStorePath:join(base,'host.sqlite'),sandboxCliPath:SANDBOX,
      sandboxCodexHome:scratchCodexHome,binPath:NODE,binArgsPrefix:[BIN],
      builderPrincipal:BUILDER_PRINCIPAL,builderSid:BUILDER_SID,verifierSid:VERIFIER_SID,
      operatorHome:OPERATOR_HOME,hiddenFixture,emptyGitConfig,builderTemp,
      permissionMode:'workspace-write',pinnedVersion:'0.2.2-rc.5',deepseekApiKey:key,
      pipeGraceMs:8000,timeoutMs:300000,gitPath:GIT};

    // Stage 1a — probe inside the sandbox (version pin under confinement).
    const probe=await host.probeProvider(commonConfig);
    evidence.push({stage:'probeProvider',probe});

    // Stage 1b — confinement matrix on a scratch repo/worktree (no API cost).
    const scratchRepo=join(base,'scratch-repo');
    await mkdir(scratchRepo,{recursive:true});
    await execFileAsync(GIT,['-c','user.name=Tally','-c','user.email=tally@example.local',
      'init','-q'],{cwd:scratchRepo});
    await writeFile(join(scratchRepo,'BASE.md'),'# base\n','utf8');
    await execFileAsync(GIT,['-c','user.name=Tally','-c','user.email=tally@example.local',
      'add','BASE.md'],{cwd:scratchRepo});
    await execFileAsync(GIT,['-c','user.name=Tally','-c','user.email=tally@example.local',
      'commit','-q','-m','base'],{cwd:scratchRepo});
    const scratchSha=(await execFileAsync(GIT,['rev-parse','HEAD'],
      {cwd:scratchRepo})).stdout.trim();
    const scratchWorktree=join(base,'scratch-wt');
    const prepared=await ops.prepareWorktree({...commonConfig,repositoryPath:scratchRepo,
      sourceSha:scratchSha,worktreePath:scratchWorktree,attemptId:'attempt:dsh:confine:probe'});
    const confinement=await host.proveConfinement({...commonConfig,
      worktreePath:scratchWorktree},scratchWorktree);
    evidence.push({stage:'proveConfinement',confinement,
      dshHomeEvidence:prepared.dshHome,patchOverlayWritten:!!prepared.patchPath});

    // Stage 2 — full attempt against a fresh repo (one real model call).
    const repo=join(base,'repo');
    await mkdir(repo,{recursive:true});
    await execFileAsync(GIT,['-c','user.name=Tally','-c','user.email=tally@example.local',
      'init','-q'],{cwd:repo});
    await writeFile(join(repo,'BASE.md'),'# base\n','utf8');
    await execFileAsync(GIT,['-c','user.name=Tally','-c','user.email=tally@example.local',
      'add','BASE.md'],{cwd:repo});
    await execFileAsync(GIT,['-c','user.name=Tally','-c','user.email=tally@example.local',
      'commit','-q','-m','base'],{cwd:repo});
    const sourceSha=(await execFileAsync(GIT,['rev-parse','HEAD'],
      {cwd:repo})).stdout.trim();
    const config={...commonConfig,repositoryPath:repo,sourceSha,
      worktreePath:join(base,'wt'),attemptId:'attempt:dsh:confine:1'};
    const result=await host.runAttempt(config);
    // Independent readback straight from git (never the agent-asserted values).
    const readback=result.candidateCommit?await ops.verifyCandidateReadback(config,
      {candidateCommit:result.candidateCommit}):{verified:false};
    const store=await registry.open({path:config.hostStorePath});
    const record=(await store.get(LOCAL_DSH_CONSTANTS.attemptKeyPrefix+config.attemptId))
      ?.value;
    await store.close();
    receipt.details={
      stage1:{probe,confinement,dshHomeEvidence:prepared.dshHome,
        patchOverlayWritten:!!prepared.patchPath},
      stage2:{state:result.state,reason:result.reason,agentLaunch:result.agentLaunch,
        exitCode:result.process?.exitCode??null,pid:result.process?.pid??null,
        builderPrincipalInProcess:result.process?.builderPrincipal===BUILDER_PRINCIPAL,
        sandboxWrapperInRecord:record?.sandboxWrapper,
        envAllowlist:record?.envAllowlist??[],
        candidateCommitShaped:/^[0-9a-f]{40}$/.test(result.candidateCommit??''),
        independentReadback:readback,
        recordNoSecret:JSON.stringify(record??{}).split(secrets[0]).length===1,
        argvTail:[record?.argv?.at(-1)?.slice(0,120)].filter(Boolean)},
      environment:{node:process.version,platform:process.platform,
        sandboxCli:SANDBOX.includes('codex')?'codex-win32 elevated':'custom',
        builderPrincipal:BUILDER_PRINCIPAL}};
    receipt.PASS=result.state==='LOCAL_CANDIDATE_READY'&&probe.available===true
      &&confinement.verified===true&&readback.verified===true;
    receipt.REAL_DSH_WINDOWS_CONFINEMENT='COMPLETED';
  }finally{
    // Bounded teardown: scratch trees are disposable; best-effort only.
    await rm(base,{recursive:true,force:true}).catch(()=>{});
  }
}catch(error){
  receipt.REAL_DSH_WINDOWS_CONFINEMENT='NOT_RUN';
  receipt.reason=String(error?.message??error);
  receipt.PASS=false;
}
console.log(JSON.stringify(receipt,null,2));
process.exit(receipt.PASS===true?0:1);
