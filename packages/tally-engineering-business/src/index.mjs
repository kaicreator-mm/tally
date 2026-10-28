/**
 * Tally Engineering Business v2 T02a, source-only selected B candidate.
 * DH v0.10.2 public v08 Business ABI; Kernel owns all Admission/State/Effects.
 */
export const MODULE_ID = 'business-tally-task-readiness@v2.0.0';
export const PACKAGE_KIND = 'business';
export const ABI_VERSION = 'v08-business-abi/1';
export const ENDPOINTS = Object.freeze([
  {endpointId:'business:tally-task-readiness',kind:'business-policy'},
]);
export const INTENT_TYPES = Object.freeze(['inspectTaskReadiness']);
export const EFFECT_TYPES = Object.freeze(['effect:tally-readiness-observation']);
const SOURCE_SHA=/^[0-9a-f]{40}$/;
const ISSUE_REF=/^https:\/\/github\.com\/[^/\s]+\/[^/\s]+\/issues\/[1-9][0-9]*$/;
const isStr=x=>typeof x==='string'&&x.length>0&&x.length<=512;

/**
 * Advisory only. Neither a boolean input nor a schema-valid source is proof of
 * a native GitHub Claim, accepted DAG Dependency, independent Review or K permit.
 */
export function assessTaskEvidence(input) {
  if(!input||typeof input!=='object'||Array.isArray(input)) {
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_INPUT',admitted:false,taskPass:false});
  }
  const {issueRef,issueNumber,sourceSha,claimRef,dependenciesComplete}=input;
  const sourceValid=ISSUE_REF.test(issueRef)&&Number.isSafeInteger(issueNumber)&&issueNumber>0&&
    issueRef.endsWith(`/issues/${issueNumber}`)&&SOURCE_SHA.test(sourceSha)&&isStr(claimRef);
  if(!sourceValid){
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',admitted:false,taskPass:false});
  }
  if(dependenciesComplete!==true){
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'DEPENDENCIES_UNVERIFIED',admitted:false,taskPass:false});
  }
  return Object.freeze({
    state:'OBSERVED_INPUT_ONLY',reason:'REQUIRES_NATIVE_CLAIM_AND_KERNEL_ADMISSION',
    issueRef,sourceSha,admitted:false,taskPass:false,
  });
}
const inputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  claimRef:{source:'event',path:['payload','claimRef']},
  dependenciesComplete:{source:'event',path:['payload','dependenciesComplete']},
});
const decisionSchema={isValid(value){
  return !!value&&typeof value==='object'&&!Array.isArray(value)&&
    !!value.decision&&typeof value.decision==='object'&&!Array.isArray(value.decision)&&
    typeof value.decision.outcome==='string'&&
    !!value.event&&typeof value.event==='object'&&!Array.isArray(value.event)&&
    typeof value.event.type==='string';
}};
export function getPolicy(){
  return {
    packageId:MODULE_ID,policyId:'tally-readiness',domainId:'tally-engineering',
    workflowDefinition:{
      workflowKey:'tally-readiness',initialState:'open',initialContext:{},
      guards:[],states:[
        {stateKey:'open',transitions:[{
          transitionKey:'inspect-task-source',
          trigger:{kind:'event',eventType:'TALLY_TASK_READINESS_INSPECTION_REQUESTED'},
          targetState:'open',
          effectIntents:[{
            effectType:'effect:tally-readiness-observation',inputFrom,
            // GitHub issue URL contains '/', invalid for DH 0.10.2 Kernel key.
            // Native issueNumber must match the canonical issueRef before advice.
            // Domain App keys are scoped to a selected project; multi-repo
            // scope must add trusted repository identity in the T02b adapter.
            idempotencyKeyFrom:{template:'inspect:{issueNumber}:{sourceSha}'}
          }]
        }]},
        {stateKey:'closed',kind:'final'},
      ],
    },
    decisionSchema,governanceSemantics:{hardInvariants:[]},
    initialStateKey:'open',initialContext:{},
    rules:{'rule:tally-readiness':{
      ruleId:'rule:tally-readiness',
      eventType:'TALLY_TASK_READINESS_INSPECTION_REQUESTED',
      payloadFromInput:{
        issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',claimRef:'claimRef',
        dependenciesComplete:'dependenciesComplete',
      },
    }},
    intentBindings:{
      inspectTaskReadiness:{
        trigger:{kind:'event',eventType:'TALLY_TASK_READINESS_INSPECTION_REQUESTED'},
        ruleId:'rule:tally-readiness',// Public Microkernel facade sends with K-derived callerRole='requester'.
        // This intent is observation-only; it cannot grant admission/Task PASS.
        roles:['requester','planner','builder','reviewer'],
      },
    },
    effectBindings:{
      'effect:tally-readiness-observation':{
        semantics:'idempotent',
        // Actual selected-C dispatch via the K-owned sealed resources port,
        // not an invented parallel Tally Tool runtime. No external effect.
        resource:{key:'tally-source-observation',operation:'tally.observeReadiness'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function') {
            throw new Error('SELECTED_READONLY_CAPABILITY_REQUIRED');
          }
          const sourceObservation=await resources.call(
            'tally-source-observation','tally.observeReadiness',
            {sourceRef:input?.issueRef,sourceSha:input?.sourceSha,issueNumber:input?.issueNumber,effectId}
          );
          // This Business projection is still advisory: a K-admitted
          // *read-only inspection* is not a K-admitted Task execution. B
          // consumes only the selected C allowlisted native pair, not any
          // caller-provided status/Claim booleans.
          const pairDiagnosis=sourceObservation?.status==='SOURCE_PAIR_OBSERVED'&&
            sourceObservation?.claimStatus==='NOT_VERIFIED'
            ? diagnoseNativeTaskReadiness({
                workItem:sourceObservation.workItem,
                claimEvents:sourceObservation.claimEvents,
                expectedIssueRef:input?.issueRef,
                expectedSourceSha:input?.sourceSha,
              })
            : null;
          return {
            ...assessTaskEvidence(input),effectId,sourceObservation,
            ...(pairDiagnosis?{pairDiagnosis}:{}),
          };
        },
      },
    },
  };
}


/**
 * Tally v2 T06c Engineering Business advisory readiness projection.
 *
 * Native C observations can be supplied by selected K/Host only in a future
 * admitted operation. This helper never attests its own sources, creates a
 * durable CAS Claim, takes a K role or upgrades Task/Release authority.
 *
 * An observed v3 DISPATCH_CLAIMED comment is not an accepted CAS Start Record.
 */
const DIAG_SOURCE_SHA=/^[0-9a-f]{40}$/;
const DIAG_ISSUE=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/;
const output=(reason,more={})=>Object.freeze({
  state:'NOT_ADMITTED',reason,dependencyState:'UNKNOWN',
  claimState:'NOT_VERIFIED',agentState:'UNKNOWN',
  admitted:false,authorized:false,taskPass:false,releasePass:false,
  ...more
});
/**
 * Only present a bounded explainable reason, not raw GitHub events/comments.
 * Even a caller-spoofed "accepted" flag is never proof of serialized CAS.
 */
export function diagnoseNativeTaskReadiness(options={}){
  try{
    const {workItem,claimEvents,expectedIssueRef,expectedSourceSha,
      localAgentStatus='UNKNOWN',builderConfinementStatus='UNKNOWN',
      hiddenVerifierStatus='UNKNOWN'}=options??{};
    if(!DIAG_ISSUE.test(expectedIssueRef??'')||
      !DIAG_SOURCE_SHA.test(expectedSourceSha??''))
      return output('SOURCE_IDENTITY_UNAVAILABLE');
    if(!workItem||typeof workItem!=='object'||Array.isArray(workItem)||
      workItem.issueRef!==expectedIssueRef||
      workItem.sourceSha!==expectedSourceSha||
      workItem.authorized!==false||
      workItem.admitted!==false||
      workItem.taskPass!==false||
      workItem.claimStatus!=='NOT_VERIFIED')
      return output('NATIVE_WORK_ITEM_SOURCE_UNVERIFIED');
    const deps=workItem.dependencyState;
    if(deps==='BLOCKED')return output('NATIVE_DEPENDENCIES_BLOCKED',{
      dependencyState:'BLOCKED'});
    if(deps!=='CLEAR')return output('NATIVE_DEPENDENCIES_UNKNOWN');
    if(workItem.reason!=='CLAIM_NOT_VERIFIED'||workItem.workflowState!=='state:ready')
      return output('WORK_ITEM_NOT_CLAIMABLE_SOURCE_ONLY',{dependencyState:'CLEAR'});
    if(!claimEvents||typeof claimEvents!=='object'||Array.isArray(claimEvents)||
      claimEvents.claimStatus!=='NOT_VERIFIED'||
      claimEvents.authorized!==false||claimEvents.admitted!==false||
      claimEvents.taskPass!==false||claimEvents.releasePass!==false||
      !Number.isSafeInteger(claimEvents.candidateCount)||
      claimEvents.candidateCount<0||claimEvents.candidateCount>1000||
      !Array.isArray(claimEvents.candidates)||
      claimEvents.candidates.length!==claimEvents.candidateCount)
      return output('CLAIM_EVENT_SOURCE_UNVERIFIED',{dependencyState:'CLEAR'});
    if(claimEvents.candidateCount>1)
      return output('AMBIGUOUS_CLAIM_EVENT_CANDIDATES',{
        dependencyState:'CLEAR',claimState:'AMBIGUOUS'});
    if(claimEvents.candidateCount===1){
      const c=claimEvents.candidates[0];
      if(!c||c.sourceSha!==expectedSourceSha||
        c.sourceClass!=='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'||
        c.claimAccepted!==false||
        typeof c.dispatchId!=='string'||!c.dispatchId)
        return output('CLAIM_EVENT_SOURCE_UNVERIFIED',{dependencyState:'CLEAR'});
    }
    // OS gates are independently necessary and cannot be asserted by an
    // unprivileged event. An omitted witness is UNKNOWN, never AVAILABLE.
    if(localAgentStatus!=='AVAILABLE')
      return output('LOCAL_AGENT_UNAVAILABLE',{dependencyState:'CLEAR',
        agentState:'UNAVAILABLE_OR_UNKNOWN'});
    if(builderConfinementStatus!=='VERIFIED')
      return output('BUILDER_CONFINEMENT_UNAVAILABLE',{dependencyState:'CLEAR',
        agentState:'CONFINEMENT_UNVERIFIED'});
    if(hiddenVerifierStatus!=='VERIFIED')
      return output('HIDDEN_VERIFIER_UNAVAILABLE',{dependencyState:'CLEAR',
        agentState:'HIDDEN_UNVERIFIED'});
    // Even all three positive status literals are not independent OS proof,
    // and a correct comment is not an accepted CAS writer's Start Record.
    return output('ACCEPTED_CAS_CLAIM_NOT_VERIFIED',{
      dependencyState:'CLEAR',
      claimState:claimEvents.candidateCount?'SELF_REPORTED_EVENT_ONLY':'NONE',
      agentState:'DECLARED_READY_ONLY'
    });
  }catch{return output('UNTRUSTED_SOURCE_INPUT');}
}

// W-F/#137 selected read-only Task as-of projection; never a grant.
export {projectNativeTaskEffectiveSnapshot} from './task-effective-snapshot.mjs';
