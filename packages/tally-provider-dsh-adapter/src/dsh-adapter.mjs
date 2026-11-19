/**
 * Tally v2.1 candidate — AgentProvider/1 contract adapter for the DeepSeek
 * Harness (DSH) CLI, wrapping the #467-verified machinery in dsh-host.mjs.
 *
 * Contract discipline (AgentProvider/1, packages/tally-agent-provider):
 *  - observations only: the adapter never asserts Task/Review/Release facts;
 *    evidence is opaque, untrusted, adapter-redacted material;
 *  - selection gate: the AgentProvider/1 registry's explicit
 *    `enabledProviderIds` is the authoritative door — the provider is
 *    DEFAULT-DISABLED and there is no fallback to any other provider;
 *    the #467 env two-gate survives only as defense-in-depth
 *    (`dshEnvGateState`: if a Host sets either legacy env field, both must
 *    agree before the adapter acts);
 *  - probe: VERSION (exact pinned-version match) -> CREDENTIALS
 *    (`DEEPSEEK_API_KEY` present in the Host-supplied context env) ->
 *    optional CANARY (bounded; default: NO real turn, reported NOT_RUN);
 *    any failure is UNAVAILABLE with a reason;
 *  - execute: one-shot `dsh --profile headless` process pinned to the
 *    Host-supplied workspace (`request.worktreePath` = process cwd);
 *    nonzero exit -> FAILED; timeout -> tree kill + UNKNOWN; AbortSignal ->
 *    classified from the adapter's own intent as CANCELLED (the #467-verified
 *    trap: DSH SIGTERM drains gracefully and exits 0, so exit codes can
 *    never classify cancellation); stdout is tail-truncated and secret-
 *    redacted into evidence;
 *  - UNKNOWN always carries a reason; no authoritative fields ever.
 *
 * The credential value never reaches observations, logs or records: it is
 * injected only into the execution environment allowlist and used as a
 * redaction secret.
 */
import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync} from 'node:fs';
import {DSH_PERMISSION_MODES,PINNED_DSH_VERSION,dshEnvGateState} from './dsh-config.mjs';
import {ATTEMPT,SHA,TASK,iso,redact,tail,probeProvider,
  prepareDshHome,dshExecutionEnvironment,runDshHeadlessTurn,
  createDefaultTerminate} from './dsh-host.mjs';

const PROVIDER_ID='dsh';
const CAPABILITIES=Object.freeze(['headless-turn']);

function observation(state,reason,evidence=null){return {state,reason,evidence};}

/**
 * Build the plain AgentProvider/1 specification object. Wrap it with the
 * contract's `defineAgentProvider` (see `createDshAgentProvider`) to get
 * observation normalization and the frozen provider surface.
 *
 * deps:
 *  - spawnFn      child-process spawn injection (tests/fakes); default real spawn
 *  - killTree     process-tree kill injection; default #467 terminate
 *  - now          clock injection
 *  - hostEnv      the Host decision environment for the defense-in-depth
 *                 env gate (default {} — the adapter never reads process.env)
 *  - config       adapter config (see dshConfigFromEnv); plain object
 *  - defineAgentProvider  contract binder; when omitted, resolved by dynamic
 *                 import from the sibling contract package (monorepo layout),
 *                 mirroring #467's injected-store import fallback.
 */
export function defineDshAgentProviderSpecification(deps={}){
  const {spawnFn=spawn,now=()=>Date.now(),killTree,hostEnv={},config={}}=deps;
  const terminate=killTree??createDefaultTerminate(spawnFn);
  const secretsOf=credential=>credential?[credential]:[];
  const baseEvidence=extra=>({provider:PROVIDER_ID,...extra});
  const gateClosed=(operation,reason)=>observation(
    operation==='probe'?'UNAVAILABLE':'FAILED','DSH_ENV_GATE_CLOSED',
    baseEvidence({gateReason:reason}));
  const credentialFrom=context=>context?.env?.DEEPSEEK_API_KEY??
    hostEnv?.DEEPSEEK_API_KEY??config.deepseekApiKey??null;

  function configValid(){
    const permissionMode=config.permissionMode??'workspace-write';
    return DSH_PERMISSION_MODES.includes(permissionMode)&&
      (permissionMode!=='danger-full-access'||config.allowDangerousFullAccess===true);
  }

  async function probe(context){
    const gate=dshEnvGateState(hostEnv);
    if(!gate.open)return gateClosed('probe',gate.reason);
    if(!configValid())
      return observation('UNAVAILABLE','LOCAL_AGENT_UNAVAILABLE',
        baseEvidence({stage:'CONFIG',detail:'DSH_CONFIG_INVALID',
          pinnedVersion:config.pinnedVersion??PINNED_DSH_VERSION}));
    const runConfig={...config,deepseekApiKey:credentialFrom(context)};
    const result=await probeProvider(spawnFn,runConfig);
    const evidence=baseEvidence({
      stage:result.stage??null,detail:result.detail??null,
      observedVersion:result.observedVersion??null,
      pinnedVersion:result.pinnedVersion,binSource:result.binSource,
      exitCode:result.exitCode??null,spawnError:result.spawnError??false,
      credentialEnv:result.credentialEnv??null,
      credentialPresent:result.credentialPresent??null,
      providerVersion:result.providerVersion??null,
      // CANARY is bounded and OFF by default: no real DSH turn is made by
      // the probe; the real turn is controller-owned acceptance (#473).
      canary:'NOT_RUN'});
    if(!result.available)
      return observation('UNAVAILABLE','LOCAL_AGENT_UNAVAILABLE',evidence);
    return observation('AVAILABLE',null,evidence);
  }

  async function execute(request,context){
    const gate=dshEnvGateState(hostEnv);
    if(!gate.open)return gateClosed('execute',gate.reason);
    const invalid=reason=>observation('FAILED',reason,
      baseEvidence({attemptId:request?.attemptId??null,taskId:request?.taskId??null,
        spawned:false}));
    if(!request||typeof request!=='object'||Array.isArray(request))
      return invalid('DSH_REQUEST_INVALID');
    const {taskId,attemptId,sourceSha,prompt,worktreePath}=request;
    if(!TASK.test(taskId??'')||!ATTEMPT.test(attemptId??'')||!SHA.test(sourceSha??'')||
      typeof prompt!=='string'||prompt.length===0||
      typeof worktreePath!=='string'||worktreePath.length===0)
      return invalid('DSH_CONFIG_INVALID');
    if(!configValid())return invalid('DSH_CONFIG_INVALID');
    if(active.has(attemptId))return invalid('ATTEMPT_ALREADY_ACTIVE');
    if(!existsSync(worktreePath))
      return invalid('PROVIDER_WORKSPACE_UNAVAILABLE');
    const credential=credentialFrom(context);
    const runConfig={...config,
      permissionMode:config.permissionMode??'workspace-write',
      deepseekApiKey:credential};
    const probeResult=await probeProvider(spawnFn,runConfig);
    if(!probeResult.available)
      return observation('FAILED','LOCAL_AGENT_UNAVAILABLE',
        baseEvidence({attemptId,taskId,spawned:false,stage:probeResult.stage??null,
          observedVersion:probeResult.observedVersion??null,
          pinnedVersion:probeResult.pinnedVersion,
          credentialPresent:probeResult.credentialPresent??null}));
    const timeoutMs=Number.isSafeInteger(request.timeoutMs)&&request.timeoutMs>0?
      request.timeoutMs:runConfig.timeoutMs??300000;
    const prepared=await prepareDshHome(runConfig,attemptId);
    const env=dshExecutionEnvironment(runConfig,prepared);
    const intent={cancelled:false,pid:null};
    active.set(attemptId,{intent,terminate});
    const secrets=secretsOf(credential);
    try{
      const turn=await runDshHeadlessTurn({spawnFn,terminate,config:runConfig,env,
        cwd:worktreePath,prompt,patchPath:prepared.patchPath,timeoutMs,
        signal:context?.signal,intent});
      const {classification,terminal}=turn;
      const evidence=baseEvidence({
        attemptId,taskId,exitCode:Number.isSafeInteger(terminal.code)?
          terminal.code:null,pid:turn.pid,timedOut:turn.timedOut,
        cancelled:turn.cancelled,orphanPipe:terminal.orphanPipe===true,
        permissionMode:runConfig.permissionMode??'workspace-write',
        pinnedVersion:runConfig.pinnedVersion??PINNED_DSH_VERSION,
        spawned:true,startedAt:iso(()=>turn.startedAt),
        terminatedAt:iso(()=>turn.terminatedAt)});
      const finalAnswer=terminal.spawnError?'':terminal.stdout??'';
      if(classification.state==='COMPLETED'){
        evidence.stdoutSha256=createHash('sha256').update(finalAnswer).digest('hex');
        evidence.stdoutBytes=Buffer.byteLength(finalAnswer);
        evidence.stdoutTail=redact(tail(finalAnswer.trim(),200),secrets);
      }else{
        evidence.stderrTail=redact(tail(terminal.stderr,500),secrets);
      }
      return observation(classification.state,classification.reason,evidence);
    }finally{active.delete(attemptId);}
  }

  async function cancel(attempt){
    if(!attempt||typeof attempt!=='object')
      return observation('UNKNOWN','ATTEMPT_STATUS_UNKNOWN',
        baseEvidence({attemptId:null}));
    const running=active.get(attempt.attemptId);
    if(!running)
      return observation('NOT_RUNNING','ATTEMPT_NOT_RUNNING',
        baseEvidence({attemptId:attempt.attemptId??null}));
    // Mark the adapter's own cancellation intent BEFORE killing: DSH
    // SIGTERM drains gracefully and exits 0, so only this intent may
    // classify the run as CANCELLED (#467-verified trap).
    running.intent.cancelled=true;
    let terminated=false;
    if(Number.isSafeInteger(running.intent.pid))
      terminated=await terminate(running.intent.pid).catch(()=>false);
    return observation('CANCEL_REQUESTED','ATTEMPT_CANCEL_REQUESTED',
      baseEvidence({attemptId:attempt.attemptId,pid:running.intent.pid,
        terminated}));
  }

  async function reconcile(attempt){
    const attemptId=attempt?.attemptId??null;
    if(attemptId!==null&&active.has(attemptId))
      return observation('RUNNING','ATTEMPT_IN_FLIGHT',
        baseEvidence({attemptId,durableJournal:false}));
    // One-shot process model with no durable execution journal: an attempt
    // that is not in flight (or unknown to this process) is honestly
    // UNKNOWN, never guessed COMPLETED/FAILED.
    return observation('UNKNOWN','ATTEMPT_STATUS_UNKNOWN',
      baseEvidence({attemptId,durableJournal:false}));
  }

  const active=new Map();
  return {id:PROVIDER_ID,contractVersion:1,capabilities:[...CAPABILITIES],
    probe,execute,cancel,reconcile};
}

/**
 * Contract-bound factory. `defineAgentProvider` is injectable for tests;
 * the default resolves from the sibling contract package via dynamic import
 * (works in this monorepo checkout layout, mirroring #467's injected-store
 * import fallback discipline).
 */
export async function createDshAgentProvider(deps={}){
  const define=deps.defineAgentProvider??
    (await import('../../tally-agent-provider/src/index.mjs')).defineAgentProvider;
  return define(defineDshAgentProviderSpecification(deps));
}
