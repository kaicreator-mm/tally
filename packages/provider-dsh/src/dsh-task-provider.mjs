/**
 * Tally v2.1 candidate — unified R105 TaskProvider facade for the DSH host.
 * Thin adapter over createLocalDshHost: selection stays Host-side
 * (resolveDshSelection, default-off), probe is the host's three-tier
 * VERSION → CREDENTIALS(AUTH) → CANARY gate (any failed tier ⇒ available
 * false, an attempt then never spawns), execute maps the unified envelope
 * onto runAttempt, cancel works ONLY through the execute-time AbortSignal,
 * and readback re-derives the candidate from git independently (never from
 * agent-asserted values). taskPass/releasePass stay false in every outcome:
 * execution evidence only — no Task/Release verdicts, no failure fallback.
 */
import {spawn} from 'node:child_process';
import {createLocalDshHost,createDefaultDshOperations,LOCAL_DSH_CONSTANTS}
  from './dsh-host.mjs';

const envelopeDenied=(input,reason)=>({provider:'dsh',providerId:'dsh',
  taskId:input?.taskId??null,attemptId:input?.attemptId??null,
  sourceSha:input?.sourceSha??null,state:'AGENT_NOT_EXECUTED',reason,
  agentLaunch:false,replayFixture:false,claimGeneration:1,claimAccepted:true,
  admitted:true,authorized:false,process:null,candidateTree:null,
  candidateCommit:null,candidateRef:null,candidatePaths:null,
  hostAuthorization:input?.hostAuthorization??null,taskPass:false,releasePass:false});

export function createDshTaskProvider({spawnFn=spawn,operations,
  now=()=>Date.now(),killTree,openStore}={}){
  const ops=operations??createDefaultDshOperations(spawnFn);
  const host=createLocalDshHost({spawnFn,operations:ops,now,killTree,openStore});
  const controllers=new Map();
  async function probe(config){
    const result=await host.probeProvider(config);
    if(result.available)
      return {...result,providerId:'dsh',
        tiers:{VERSION:'PASS',CREDENTIALS:'PASS',CANARY:'PASS'}};
    const tier=result.stage==='AUTH'?'CREDENTIALS':
      result.stage==='CANARY'?'CANARY':'VERSION';
    return {...result,providerId:'dsh',tiers:{
      VERSION:tier==='VERSION'?'FAIL':'PASS',
      CREDENTIALS:tier==='VERSION'?'NOT_RUN':tier==='CREDENTIALS'?'FAIL':'PASS',
      CANARY:tier==='CANARY'?'FAIL':'NOT_RUN'}};
  }
  async function execute(input){
    if(input?.providerId!==undefined&&input.providerId!=='dsh')
      return envelopeDenied(input,'DSH_PROVIDER_ID_MISMATCH');
    const providerConfig=input?.providerConfig??{};
    const writeSet=Array.isArray(input?.allowedWriteSet)?input.allowedWriteSet:null;
    const candidatePath=writeSet!==null&&writeSet.length===1?writeSet[0]:null;
    const pinned=providerConfig.candidateRelativePath;
    if(!candidatePath||(pinned!==undefined&&pinned!==candidatePath))
      return envelopeDenied(input,'DSH_WRITESET_SINGLE_CANDIDATE_REQUIRED');
    const signal=input.signal;
    if(signal?.aborted)return envelopeDenied(input,'DSH_SIGNAL_PRE_ABORTED');
    const controller=new AbortController();
    if(signal)controllers.set(signal,controller);
    try{
      const outcome=await host.runAttempt({...providerConfig,
        attemptId:input.attemptId,sourceSha:input.sourceSha,issueRef:input.issueRef,
        worktreePath:input.worktreePath,candidateRelativePath:candidatePath,
        signal:controller.signal,timeoutMs:input.timeoutMs??providerConfig.timeoutMs});
      return {...outcome,provider:'dsh',taskId:input.taskId??null,
        candidatePaths:outcome.candidatePaths??null,
        hostAuthorization:input.hostAuthorization??null};
    }catch(error){
      return {...envelopeDenied(input,'ATTEMPT_STATUS_UNKNOWN'),
        state:'LOCAL_ATTEMPT_UNKNOWN',agentLaunch:true,
        process:{status:'UNKNOWN',exitCode:null,pid:null,startedAt:null,terminatedAt:null},
        error:error?.code??String(error?.message??'DSH_HOST_ERROR')};
    }finally{if(signal)controllers.delete(signal);}
  }
  function cancel(input){
    const signal=input?.signal;
    if(!signal)return false;
    const controller=controllers.get(signal);
    if(!controller||signal.aborted||controller.signal.aborted)return false;
    controller.abort();
    return true;
  }
  async function readback(config,candidate){
    if(typeof ops.verifyCandidateReadback!=='function')
      return {verified:false,ready:false,reason:'DSH_READBACK_UNAVAILABLE'};
    const result=await ops.verifyCandidateReadback(config,candidate);
    const expected=config?.candidateRelativePath??
      LOCAL_DSH_CONSTANTS.candidateRelativePath;
    const ready=result.verified===true&&
      result.candidateTree===candidate?.candidateTree&&
      result.parent===config?.sourceSha&&
      result.paths?.length===1&&result.paths[0]===expected;
    return {...result,ready};
  }
  return Object.freeze({providerId:'dsh',probe,execute,cancel,readback});
}
