/** #264 E1 — installed write-journey runner (R91).
 *
 * Composes the EXISTING selected single-Codex Windows consumer
 * (experiments/v2/selected-business-build/local-codex-windows-e2e.mjs — the
 * journey-controller's sanctioned K/Host consumer) from trusted Host-process
 * environment pins and runs it for the J04 `/api/actions` dispatch seam.
 *
 * Authority model (mirrors the E0 version-task-dag seam in serve.mjs):
 *  - the runner exists ONLY when the trusted Host process exports
 *    TALLY_V2_AGENT_DISPATCH=host-mounted plus the exact claim/lease pins;
 *  - no Claim, permit, token or authority is ever accepted from an HTTP
 *    request, a browser payload or an Agent;
 *  - every attempt is bound to the exact served source SHA: a claim pinned
 *    to any other SHA refuses to start (STALE lease, never reused);
 *  - the candidate contract is the existing consumer's own (one file,
 *    independent git readback, parent === pinned source SHA).
 *
 * The review-repair (J05) consumer is the single-Builder ReviewDecision
 * lane (#264): mounted ONLY when the trusted Host process additionally
 * exports TALLY_V2_REVIEW_DISPATCH=host-mounted plus the reviewer decision
 * pins. It spawns the existing selected single-review consumer
 * (single-codex-review-e2e.mjs → sealed B→K→C → consumer-single-review),
 * which verifies the independent reviewer Claim, the Builder CAS record,
 * candidate/repair git readback, the PR head and exact-SHA CI before any
 * ReviewDecision CAS write. Without those pins the seam keeps the honest
 * fail-closed answer UNKNOWN REVIEW_CONSUMER_NOT_MOUNTED — never a
 * fabricated join, never a second Builder (the T06N two-candidate Join
 * stays a separate lane with its own cardinality rule).
 */
import {spawn} from 'node:child_process';
import {stat,readFile} from 'node:fs/promises';
import {join} from 'node:path';

const SHA=/^[a-f0-9]{40}$/;
const SHA256=/^[a-f0-9]{64}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions',
  'woodpecker','other']);
const PATH_RE=/^[A-Za-z]:[\\/][^"<>|*\x00-\x1f]{1,180}$|^\/[^"<>|*\x00-\x1f]{1,240}$/;
/** The existing single-Codex Windows consumer (T06M) machine pins; passed
 * through from the trusted Host process environment, validated present. */
const CONSUMER_PINS=['TALLY_GITHUB_READ_TOKEN','TALLY_CODEX_CLI_PATH',
  'TALLY_CODEX_SANDBOX_PATH','TALLY_BUILDER_PROFILE','TALLY_BUILDER_TEMP',
  'TALLY_EMPTY_GIT_CONFIG','TALLY_OPERATOR_HOME','TALLY_HIDDEN_FIXTURE',
  'TALLY_BUILDER_PRINCIPAL','TALLY_BUILDER_SID','TALLY_VERIFIER_SID',
  'TALLY_LOCAL_ISSUE_NUMBER','TALLY_LOCAL_REPOSITORY_PATH'];

/** Read and validate the trusted Host-process pin set. Throws TypeError with
 * a stable reason on every missing/malformed pin — no partial admission. */
export function readInstalledWriteJourneyEnvironment(env=process.env){
  if(env.TALLY_V2_AGENT_DISPATCH!=='host-mounted')
    throw new TypeError('AGENT_DISPATCH_NOT_HOST_MOUNTED');
  // An explicit Linux Host contract is needed. Historical Windows pin sets
  // remain the default, preserving the proven installed Windows path.
  const platform=env.TALLY_V2_AGENT_PLATFORM??'win32';
  if(!['win32','linux'].includes(platform))
    throw new TypeError('AGENT_HOST_PLATFORM_UNSUPPORTED');
  if(platform==='linux'&&process.platform!=='linux')
    throw new TypeError('AGENT_HOST_PLATFORM_MISMATCH');
  const claim={
    sourceSha:env.TALLY_V2_AGENT_CLAIM_SOURCE_SHA,
    dispatchId:env.TALLY_V2_AGENT_CLAIM_DISPATCH_ID,
    operatorId:env.TALLY_V2_AGENT_CLAIM_OPERATOR_ID,
    operatorKind:env.TALLY_V2_AGENT_CLAIM_OPERATOR_KIND,
    writerId:env.TALLY_V2_AGENT_CLAIM_WRITER_ID,
    writerLogin:env.TALLY_V2_AGENT_CLAIM_WRITER_LOGIN,
    commentId:Number(env.TALLY_V2_AGENT_CLAIM_COMMENT_ID),
    registryPath:env.TALLY_V2_AGENT_CLAIM_REGISTRY_PATH,
    writeSetSha256:env.TALLY_V2_AGENT_CLAIM_WRITESET_SHA256,
    attemptId:env.TALLY_V2_AGENT_CLAIM_ATTEMPT_ID};
  if(!SHA.test(claim.sourceSha??''))throw new TypeError('AGENT_CLAIM_SOURCE_SHA_INVALID');
  const ID_REASONS={dispatchId:'AGENT_CLAIM_DISPATCH_ID_INVALID',
    operatorId:'AGENT_CLAIM_OPERATOR_ID_INVALID',
    writerId:'AGENT_CLAIM_WRITER_ID_INVALID',
    attemptId:'AGENT_CLAIM_ATTEMPT_ID_INVALID'};
  for(const key of Object.keys(ID_REASONS))
    if(!ID.test(claim[key]??''))throw new TypeError(ID_REASONS[key]);
  if(!KINDS.has(claim.operatorKind??''))throw new TypeError('AGENT_CLAIM_OPERATOR_KIND_INVALID');
  if(!/^[A-Za-z0-9_.-]+$/.test(claim.writerLogin??''))
    throw new TypeError('AGENT_CLAIM_WRITER_LOGIN_INVALID');
  if(!Number.isSafeInteger(claim.commentId)||claim.commentId<1)
    throw new TypeError('AGENT_CLAIM_COMMENT_ID_INVALID');
  if(typeof claim.registryPath!=='string'||claim.registryPath.length<3||
     claim.registryPath.length>240)
    throw new TypeError('AGENT_CLAIM_REGISTRY_PATH_INVALID');
  if(!SHA256.test(claim.writeSetSha256??''))throw new TypeError('AGENT_CLAIM_WRITESET_SHA256_INVALID');
  const worktreePath=env.TALLY_V2_AGENT_WORKTREE_PATH;
  const evidenceDir=env.TALLY_V2_AGENT_EVIDENCE_DIR;
  if(typeof worktreePath!=='string'||!PATH_RE.test(worktreePath))
    throw new TypeError('AGENT_WORKTREE_PATH_INVALID');
  if(typeof evidenceDir!=='string'||!PATH_RE.test(evidenceDir))
    throw new TypeError('AGENT_EVIDENCE_DIR_INVALID');
  const timeoutMs=Number(env.TALLY_V2_AGENT_TIMEOUT_MS);
  const requiredPins=platform==='linux'?
    CONSUMER_PINS.filter(name=>name!=='TALLY_CODEX_SANDBOX_PATH').concat(
      'TALLY_LINUX_BWRAP_PATH','TALLY_V2_AGENT_PREBUILT_CONSUMER_DIR'):CONSUMER_PINS;
  const missing=requiredPins.filter(name=>
    name==='TALLY_LOCAL_REPOSITORY_PATH'&&
      env.TALLY_V2_AGENT_SCRIPT_ROOT&&
      env.TALLY_V2_AGENT_SOURCE_REPOSITORY_PATH?false:
    typeof env[name]!=='string'||env[name].length===0);
  if(missing.length)throw new TypeError('AGENT_CONSUMER_PINS_MISSING:'+missing.join(','));
  if(platform==='linux'&&
      (env.TALLY_BUILDER_PRINCIPAL!=='linux-bwrap:65534'||
       env.TALLY_BUILDER_SID!=='linux-uid:65534'||
       env.TALLY_VERIFIER_SID==='linux-uid:65534'||
       !/^\/usr\/[^\s:]+$/.test(env.TALLY_LINUX_BWRAP_PATH??'')))
    throw new TypeError('AGENT_LINUX_SANDBOX_PINS_INVALID');
  const scriptRoot=env.TALLY_V2_AGENT_SCRIPT_ROOT??
    env.TALLY_LOCAL_REPOSITORY_PATH;
  const sourceRepositoryPath=env.TALLY_V2_AGENT_SOURCE_REPOSITORY_PATH??
    env.TALLY_LOCAL_REPOSITORY_PATH;
  if(!PATH_RE.test(scriptRoot??'')||
     !PATH_RE.test(sourceRepositoryPath??''))
    throw new TypeError('AGENT_SCRIPT_OR_SOURCE_REPOSITORY_PATH_INVALID');
  if(!Number.isSafeInteger(Number(env.TALLY_LOCAL_ISSUE_NUMBER))||
     Number(env.TALLY_LOCAL_ISSUE_NUMBER)<1)
    throw new TypeError('AGENT_LOCAL_ISSUE_NUMBER_INVALID');
  // The historical single-Builder demo is not reusable across projects.
  // A new selected GitHub repo MUST be compiled into the C/Provider
  // together with its task, version and frozen candidate path, not merely
  // chosen in an HTTP request or by the runtime environment.
  const sourceRepo=env.TALLY_V2_REPOSITORY??'kaicreator-mm/tally';
  const sourceRef=env.TALLY_V2_VERSION_REF??'version/v2';
  const pinNames=['T06M_TARGET_REPOSITORY','T06M_TARGET_ISSUE_NUMBER',
    'T06M_TARGET_VERSION_REF','T06M_TARGET_CANDIDATE_PATH'];
  const provided=pinNames.filter(name=>env[name]!==undefined);
  if(platform==='linux'&&provided.length!==pinNames.length)
    throw new TypeError('AGENT_LINUX_SEALED_TARGET_REQUIRED');
  if(provided.length!==0&&provided.length!==pinNames.length)
    throw new TypeError('AGENT_BUILDER_TARGET_PARTIAL');
  if(provided.length===0&&
      (sourceRepo!=='kaicreator-mm/tally'||sourceRef!=='version/v2'))
    throw new TypeError('AGENT_BUILDER_TARGET_NOT_SEALED');
  if(provided.length===pinNames.length){
    if(sourceRepo!==env.T06M_TARGET_REPOSITORY||
       sourceRef!==env.T06M_TARGET_VERSION_REF||
       Number(env.TALLY_LOCAL_ISSUE_NUMBER)!==
         Number(env.T06M_TARGET_ISSUE_NUMBER)||
       !/^[1-9][0-9]{0,9}$/.test(env.T06M_TARGET_ISSUE_NUMBER??'')||
       typeof env.T06M_TARGET_CANDIDATE_PATH!=='string'||
       env.T06M_TARGET_CANDIDATE_PATH.length<3||
       env.T06M_TARGET_CANDIDATE_PATH.includes('..')||
       env.T06M_TARGET_CANDIDATE_PATH.startsWith('/'))
      throw new TypeError('AGENT_BUILDER_TARGET_SCOPE_MISMATCH');
  }
  // #264 J05: the single-Builder ReviewDecision seam mounts only under its
  // own trusted Host pin; the decision value is the reviewer's expressed
  // judgment pinned by the Host process (authority itself stays with the
  // independently verified reviewer Claim inside the sealed C).
  let review=null;
  if(env.TALLY_V2_REVIEW_DISPATCH!==undefined){
    if(env.TALLY_V2_REVIEW_DISPATCH!=='host-mounted')
      throw new TypeError('REVIEW_DISPATCH_MODE_INVALID');
    const decision=env.TALLY_V2_REVIEW_DECISION;
    if(!['ACCEPT','REWORK','DEFER'].includes(decision??''))
      throw new TypeError('REVIEW_DECISION_INVALID');
    const decisionId=env.TALLY_V2_REVIEW_DECISION_ID;
    if(!ID.test(decisionId??''))throw new TypeError('REVIEW_DECISION_ID_INVALID');
    review=Object.freeze({decision,decisionId});
  }
  // #473 R107: task-selected AgentProvider pin. Host-pinned only — a
  // browser/agent request can never choose the provider; the default stays
  // codex (the proven selected-chain lane) and every other provider must be
  // explicitly present in the enabled list. One Attempt binds exactly one
  // provider and there is never an implicit fallback.
  const provider=env.TALLY_V2_AGENT_PROVIDER??'codex';
  const enabledProviders=(env.TALLY_V2_AGENT_PROVIDERS_ENABLED??'codex')
    .split(',').map(value=>value.trim()).filter(Boolean);
  if(!/^[a-z][a-z0-9-]{0,63}$/.test(provider)||
     !enabledProviders.includes(provider))
    throw new TypeError('AGENT_PROVIDER_PIN_INVALID');
  // Other Provider adapters have not proven POSIX sandbox confinement.
  if(platform==='linux'&&provider!=='codex')
    throw new TypeError('AGENT_LINUX_PROVIDER_UNQUALIFIED');
  return Object.freeze({platform,claim:Object.freeze({...claim,
      commentId:claim.commentId,
      timeoutMs:Number.isSafeInteger(timeoutMs)&&timeoutMs>=60000&&
        timeoutMs<=30*60*1000?timeoutMs:10*60*1000}),
    worktreePath,evidenceDir,provider,scriptRoot,sourceRepositoryPath,
    ...(review?{review}:{})});
}

/** Spawn the existing consumer exactly the way the journey-controller's
 * sanctioned default path does (same script, same env contract, same
 * e2e-result.json readback); injectable spawnE2e keeps tests offline. */
function defaultSpawnE2e(script,env,timeoutMs){
  const child=spawn(process.execPath,[script],{env,windowsHide:true});
  child.stdout?.on('data',chunk=>process.stdout.write(chunk));
  child.stderr?.on('data',chunk=>process.stderr.write(chunk));
  const startedAt=Date.now();
  return new Promise(resolve=>{
    let done=false,timer;
    const finish=value=>{if(done)return;done=true;clearTimeout(timer);resolve(value);};
    child.once('error',()=>finish({code:null,signal:null,startedAt,terminatedAt:Date.now()}));
    child.once('close',(code,signal)=>finish({code,signal,startedAt,terminatedAt:Date.now()}));
    timer=setTimeout(()=>{try{child.kill();}catch{}
      finish({code:null,signal:null,timeout:true,startedAt,terminatedAt:Date.now()});},
      timeoutMs);
    timer.unref?.();
  });
}

/** Build the run(kind) seam. The returned runner mirrors the controller's
 * default resultOf contract: {status,reason,exitCode,signal,timeout,
 * observedAt,result,evidenceDir}. A missing durable result on exit 0 stays
 * UNKNOWN (lost ACK), never PASS. */
export function createInstalledWriteJourneyRunner({settings,env=process.env,
  spawnE2e=defaultSpawnE2e}={}){
  if(!settings||typeof settings!=='object'||!settings.claim)
    throw new TypeError('INSTALLED_JOURNEY_SETTINGS_REQUIRED');
  const {claim,worktreePath,evidenceDir}=settings;
  const runnerEnv=()=>({...env,
    TALLY_CLAIM_SOURCE_SHA:claim.sourceSha,
    TALLY_CLAIM_DISPATCH_ID:claim.dispatchId,
    TALLY_CLAIM_OPERATOR_ID:claim.operatorId,
    TALLY_CLAIM_OPERATOR_KIND:claim.operatorKind,
    TALLY_CLAIM_WRITER_ID:claim.writerId,
    TALLY_CLAIM_WRITER_LOGIN:claim.writerLogin,
    TALLY_CLAIM_COMMENT_ID:String(claim.commentId),
    TALLY_CLAIM_REGISTRY_PATH:claim.registryPath,
    TALLY_LOCAL_WRITESET_SHA256:claim.writeSetSha256,
    TALLY_LOCAL_ATTEMPT_ID:claim.attemptId,
    TALLY_LOCAL_ISSUE_NUMBER:env.TALLY_LOCAL_ISSUE_NUMBER,
    TALLY_LOCAL_WORKTREE_PATH:worktreePath,
    // K sees the selected managed source repo, not the Tally driver root.
    TALLY_LOCAL_REPOSITORY_PATH:settings.sourceRepositoryPath??
      env.TALLY_LOCAL_REPOSITORY_PATH,
    TALLY_LOCAL_EVIDENCE_DIR:evidenceDir});
  const reviewEnv=()=>({...runnerEnv(),
    ...(settings.review?{TALLY_REVIEW_DECISION:settings.review.decision,
      TALLY_REVIEW_DECISION_ID:settings.review.decisionId}:{})});
  const observed=async(kind,exit)=>{
    let result=null;
    try{const info=await stat(join(evidenceDir,'e2e-result.json'));
      if(info.mtimeMs>=exit.startedAt)
        result=JSON.parse(await readFile(join(evidenceDir,'e2e-result.json'),'utf8'));
    }catch{}
    const observedAt=new Date().toISOString();
    const status=exit.code===0&&result?.status==='PASS'?'PASS':
      exit.code===0&&!result?'UNKNOWN':'FAIL';
    return {status,reason:status==='UNKNOWN'?'ACK_LOST':undefined,exitCode:exit.code,
      signal:exit.signal??null,timeout:exit.timeout===true,observedAt,result,evidenceDir};
  };
  return Object.freeze({async run(kind){
    if(kind!=='start'){
      // J05 single-Builder ReviewDecision lane: only a Host process that
      // exported the review dispatch pins gets the real consumer; otherwise
      // nothing executed and the observed-runs projection must never render
      // this as an attempt row.
      if(kind==='review-repair'&&settings.review&&settings.platform!=='linux'){
        const script=join(settings.scriptRoot??env.TALLY_LOCAL_REPOSITORY_PATH,
          'experiments','v2','selected-business-build','single-codex-review-e2e.mjs');
        const exit=await spawnE2e(script,reviewEnv(),claim.timeoutMs);
        return observed(kind,exit);
      }
      return {status:'UNKNOWN',reason:'REVIEW_CONSUMER_NOT_MOUNTED',exitCode:null,
        signal:null,timeout:false,observedAt:new Date().toISOString(),result:null};
    }
    // #473 R107: every provider uses the SAME selected sealed B→K→Host/C
    // driver. The trusted provider pin only substitutes the process-only
    // execution point inside local-codex-host; Claim/FSW/CAS/worktree/grants
    // and candidate readback never fork into a second Host or scheduler.
    const script=join(settings.scriptRoot??env.TALLY_LOCAL_REPOSITORY_PATH,
      'experiments','v2','selected-business-build',
      settings.platform==='linux'?'local-codex-linux-e2e.mjs':'local-codex-windows-e2e.mjs');
    const exit=await spawnE2e(script,runnerEnv(),claim.timeoutMs);
    return observed(kind,exit);
  }});
}
