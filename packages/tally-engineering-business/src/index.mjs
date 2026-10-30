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
export const INTENT_TYPES = Object.freeze(['inspectTaskReadiness','admitClaimedTask','runLocalCodexAttempt','joinReviewedCandidates']);
export const EFFECT_TYPES = Object.freeze(['effect:tally-readiness-observation',
  'effect:tally-claim-execution-admission','effect:tally-local-codex-attempt',
  'effect:tally-reviewer-candidate-join']);
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
const claimInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
});
const localAgentInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  attemptId:{source:'event',path:['payload','attemptId']},
});
const joinInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  joinId:{source:'event',path:['payload','joinId']},
  candidates:{source:'event',path:['payload','candidates']},
});
const decisionSchema={isValid(value){
  return !!value&&typeof value==='object'&&!Array.isArray(value)&&
    !!value.decision&&typeof value.decision==='object'&&!Array.isArray(value.decision)&&
    typeof value.decision.outcome==='string'&&
    !!value.event&&typeof value.event==='object'&&!Array.isArray(value.event)&&
    typeof value.event.type==='string';
}};
export function getPolicy(){
  // The DH 0.10.2 kernel wires EVERY transition of a workflow definition
  // against EACH intent-bound rule and the S SDK projects the rule payload
  // from the arriving event. Two intents therefore cannot share one dynamic
  // effect surface: each intent binding carries its own single-transition
  // definition over the same workflowKey, while the top-level definition
  // remains the full union for queries and policy description.
  const inspectTransition={
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
  };
  const admitTransition={
    transitionKey:'admit-claimed-task',
    trigger:{kind:'event',eventType:'TALLY_CLAIMED_TASK_ADMISSION_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-claim-execution-admission',inputFrom:claimInputFrom,
      // One admitted-Claim observation per (issue, sourceSha). A rejected
      // observation still executes as a negative projection; replay of a
      // prior admitted observation never re-executes the native C read.
      idempotencyKeyFrom:{template:'admit:{issueNumber}:{sourceSha}'}
    }]
  };
  const localAgentTransition={
    transitionKey:'run-local-codex-attempt',
    trigger:{kind:'event',eventType:'TALLY_LOCAL_CODEX_ATTEMPT_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-local-codex-attempt',inputFrom:localAgentInputFrom,
      idempotencyKeyFrom:{template:'agent:{issueNumber}:{sourceSha}:{attemptId}'}
    }]
  };
  const finalState={stateKey:'closed',kind:'final'};
  const readinessWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[inspectTransition]},finalState],
  };
  const claimWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[admitTransition]},finalState],
  };
  const localAgentWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[localAgentTransition]},finalState],
  };
  const joinTransition={
    transitionKey:'join-reviewed-candidates',
    trigger:{kind:'event',eventType:'TALLY_REVIEWED_CANDIDATES_JOIN_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-reviewer-candidate-join',inputFrom:joinInputFrom,
      // One reviewer Join observation per (issue, sourceSha, joinId). A
      // rejected observation still executes as a negative projection; replay
      // of a prior JOINED observation never re-executes the native C read.
      idempotencyKeyFrom:{template:'join:{issueNumber}:{sourceSha}:{joinId}'}
    }]
  };
  const joinWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[joinTransition]},finalState],
  };
  return {
    packageId:MODULE_ID,policyId:'tally-readiness',domainId:'tally-engineering',
    workflowDefinition:{
      workflowKey:'tally-readiness',initialState:'open',initialContext:{},
      guards:[],states:[
        {stateKey:'open',transitions:[inspectTransition,admitTransition,localAgentTransition,joinTransition]},
        finalState,
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
    },'rule:tally-claim-admission':{
      ruleId:'rule:tally-claim-admission',
      eventType:'TALLY_CLAIMED_TASK_ADMISSION_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha'},
    },'rule:tally-local-codex-attempt':{
      ruleId:'rule:tally-local-codex-attempt',
      eventType:'TALLY_LOCAL_CODEX_ATTEMPT_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',attemptId:'attemptId'},
    },'rule:tally-reviewer-candidate-join':{
      ruleId:'rule:tally-reviewer-candidate-join',
      eventType:'TALLY_REVIEWED_CANDIDATES_JOIN_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',
        joinId:'joinId',candidates:'candidates'},
    }},
    intentBindings:{
      inspectTaskReadiness:{
        trigger:{kind:'event',eventType:'TALLY_TASK_READINESS_INSPECTION_REQUESTED'},
        ruleId:'rule:tally-readiness',// Public Microkernel facade sends with K-derived callerRole='requester'.
        // This intent is observation-only; it cannot grant admission/Task PASS.
        roles:['requester','planner','builder','reviewer'],
        definition:readinessWorkflow,
      },
      admitClaimedTask:{
        trigger:{kind:'event',eventType:'TALLY_CLAIMED_TASK_ADMISSION_REQUESTED'},
        ruleId:'rule:tally-claim-admission',
        // Admission authority here is ONLY the sealed C accepted-Claim
        // observation (DH CAS registry row + current native readback), never
        // the caller, a comment flag or a self-declared accepted boolean.
        // The caller role gates only who may REQUEST the admission check.
        roles:['requester','planner','builder','reviewer'],
        definition:claimWorkflow,
      },
      runLocalCodexAttempt:{
        trigger:{kind:'event',eventType:'TALLY_LOCAL_CODEX_ATTEMPT_REQUESTED'},
        ruleId:'rule:tally-local-codex-attempt',
        roles:['requester','planner','builder','reviewer'],
        definition:localAgentWorkflow,
      },
      joinReviewedCandidates:{
        trigger:{kind:'event',eventType:'TALLY_REVIEWED_CANDIDATES_JOIN_REQUESTED'},
        ruleId:'rule:tally-reviewer-candidate-join',
        // Reviewer authority here is ONLY the sealed C reviewer-Claim
        // observation (Host-owned #179 registry row plus current native
        // readback and exact candidate git readback), never the caller.
        // The caller role gates only who may REQUEST the Join check.
        roles:['requester','planner','builder','reviewer'],
        definition:joinWorkflow,
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
      'effect:tally-claim-execution-admission':{
        semantics:'idempotent',
        // Selected C dispatch via the K-owned sealed resources port. C reads
        // the real DH CAS registry row and re-verifies the current native
        // GitHub readback; B never sees or trusts a caller/comment boolean.
        resource:{key:'tally-claim-admission',operation:'tally.admitClaimedTask'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function') {
            throw new Error('SELECTED_READONLY_CAPABILITY_REQUIRED');
          }
          const advisory=assessClaimedTaskAdmission(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY'){
            return {...advisory,effectId};
          }
          const claimObservation=await resources.call(
            'tally-claim-admission','tally.admitClaimedTask',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,effectId}
          );
          return projectClaimedTaskAdmission({input,claimObservation,effectId});
        },
      },
      'effect:tally-local-codex-attempt':{
        semantics:'non-idempotent',
        resource:{key:'tally-local-codex',operation:'tally.runLocalCodexAttempt'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          const advisory=assessLocalCodexAttempt(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call(
            'tally-local-codex','tally.runLocalCodexAttempt',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,attemptId:input.attemptId,effectId}
          );
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectLocalCodexAttempt({input,observation,effectId});
        },
      },
      'effect:tally-reviewer-candidate-join':{
        // Idempotent: a JOINED observation replays from the K journal and
        // never re-runs native candidate readback. The external C call has
        // its own K permit but launches no Builder process.
        semantics:'idempotent',
        resource:{key:'tally-local-codex',operation:'tally.joinReviewedCandidates'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          const advisory=assessReviewedCandidateJoin(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call(
            'tally-local-codex','tally.joinReviewedCandidates',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,joinId:input.joinId,
              candidates:input.candidates,effectId}
          );
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectReviewedCandidateJoin({input,observation,effectId});
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

/**
 * T06c/#137 — one-Task read-only ALM status, sourced from existing native C.
 * This is an observation, not a fresh CAS Claim or Host/K Task permit.
 * The current native pair has no authenticated observation timestamp:
 * never manufacture "as-of=now" or assume indefinitely current status.
 */
const readModelOutput=(reason,more={})=>Object.freeze({
  status:'UNKNOWN',reason,sourceQualified:false,
  sourceRef:null,sourceSha:null,observedAsOf:null,
  currentness:'AGE_NOT_VERIFIED',dependencyState:'UNKNOWN',
  openBlockerCount:null,workflowState:null,claimState:'NOT_VERIFIED',
  kernelTaskState:'NOT_VERIFIED',agentAttemptState:'NOT_VERIFIED',
  reviewState:'NOT_VERIFIED',ciState:'NOT_VERIFIED',
  hostState:'NOT_VERIFIED',releaseState:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  nextActions:Object.freeze([Object.freeze({
    kind:'READ_CURRENT_GITHUB_SOURCE',authority:'READ_ONLY',reason
  })]),...more
});
const readModelAdvice=(base,status,reason,kind,authority)=>Object.freeze({
  ...base,status,reason,nextActions:Object.freeze([Object.freeze({
    kind,authority,reason
  })])
});
/** Source pair must come from selected native GitHub C, never UI/LLM. */
export function projectNativeTaskEffectiveSnapshot({
  sourcePair,expectedIssueRef,expectedSourceSha
}={}){
  try{
    if(!DIAG_ISSUE.test(expectedIssueRef??'')||
      !DIAG_SOURCE_SHA.test(expectedSourceSha??''))
      return readModelOutput('EXPECTED_SOURCE_IDENTITY_UNAVAILABLE');
    const s=sourcePair;
    if(!s||typeof s!=='object'||Array.isArray(s)||
      s.status!=='SOURCE_PAIR_OBSERVED'||s.sourceQualified!==true||
      s.issueRef!==expectedIssueRef||s.sourceSha!==expectedSourceSha||
      s.claimStatus!=='NOT_VERIFIED'||s.authorized!==false||
      s.admitted!==false||s.taskPass!==false||s.releasePass!==false)
      return readModelOutput('NATIVE_SOURCE_PAIR_NOT_VERIFIED');
    const w=s.workItem,c=s.claimEvents;
    if(!w||typeof w!=='object'||Array.isArray(w)||
      w.issueRef!==expectedIssueRef||w.sourceSha!==expectedSourceSha||
      w.claimStatus!=='NOT_VERIFIED'||w.authorized!==false||
      w.admitted!==false||w.taskPass!==false||
      !c||typeof c!=='object'||Array.isArray(c)||
      c.claimStatus!=='NOT_VERIFIED'||c.authorized!==false||
      c.admitted!==false||c.taskPass!==false||c.releasePass!==false||
      !Number.isSafeInteger(c.candidateCount)||c.candidateCount<0||
      c.candidateCount>1000||!Array.isArray(c.candidates)||
      c.candidates.length!==c.candidateCount)
      return readModelOutput('NATIVE_SOURCE_FACTS_NOT_VERIFIED');
    const base={
      sourceQualified:true,sourceRef:expectedIssueRef,sourceSha:expectedSourceSha,
      observedAsOf:null,currentness:'AGE_NOT_VERIFIED',
      dependencyState:w.dependencyState,
      openBlockerCount:Number.isSafeInteger(w.openBlockerCount)?
        w.openBlockerCount:null,
      workflowState:typeof w.workflowState==='string'?w.workflowState:null,
      claimState:'NOT_VERIFIED',kernelTaskState:'NOT_VERIFIED',
      agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',
      ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',
      releaseState:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    };
    if(w.dependencyState==='BLOCKED')
      return readModelAdvice(base,'WAIT_EXTERNAL','NATIVE_DEPENDENCIES_BLOCKED',
        'WAIT_NATIVE_DEPENDENCIES','READ_ONLY');
    if(w.dependencyState!=='CLEAR')
      return readModelOutput('NATIVE_DEPENDENCIES_UNKNOWN');
    if(w.issueState!=='open'||w.workflowState!=='state:ready'||
      w.workItemType!=='type:task'||w.reason!=='CLAIM_NOT_VERIFIED')
      return readModelAdvice(base,'WAIT_EXTERNAL','TASK_NOT_NATIVE_READY',
        'WAIT_NATIVE_TASK_READY','READ_ONLY');
    if(c.reason==='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'){
      if(c.candidateCount>1)
        return readModelAdvice(base,'CONFLICT','AMBIGUOUS_CLAIM_EVENT_CANDIDATES',
          'RECONCILE_MULTIPLE_NATIVE_CLAIMS','HOST_REQUIRED');
      if(c.candidateCount===1)
        return readModelAdvice(base,'WAIT_EXTERNAL',
          'SELF_REPORTED_CLAIM_NEEDS_HOST_RECONCILIATION',
          'RECONCILE_NATIVE_CLAIM','HOST_REQUIRED');
    }
    if(c.reason!=='NO_DISPATCH_CLAIM_EVENTS'||c.candidateCount!==0)
      return readModelOutput('NATIVE_CLAIM_ABSENCE_UNVERIFIED');
    return readModelAdvice(base,'AWAITING_HOST_ADMISSION',
      'NATIVE_READY_NO_ACCEPTED_CLAIM',
      'REQUEST_SELECTED_HOST_ADMISSION','HOST_AND_ADS_REQUIRED');
  }catch{return readModelOutput('SOURCE_OBSERVATION_UNREADABLE');}
}

/**
 * T06l/#169 — advisory input screening for the claimed-Task admission intent.
 * The accepted-Claim fact can only come from the selected sealed C reading
 * the real DH CAS registry plus a current native GitHub publication witness;
 * no input shape, caller assertion or comment flag is admission evidence.
 */
export function assessClaimedTaskAdmission(input){
  if(!input||typeof input!=='object'||Array.isArray(input)) {
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_INPUT',
      admitted:false,authorized:false,taskPass:false,agentLaunch:false,releasePass:false});
  }
  const {issueRef,issueNumber,sourceSha}=input;
  if(!(ISSUE_REF.test(issueRef)&&Number.isSafeInteger(issueNumber)&&issueNumber>0&&
      issueRef.endsWith(`/issues/${issueNumber}`)&&SOURCE_SHA.test(sourceSha))) {
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',
      admitted:false,authorized:false,taskPass:false,agentLaunch:false,releasePass:false});
  }
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',
    reason:'REQUIRES_SEALED_CLAIM_ADMISSION_OBSERVATION',
    issueRef,issueNumber,sourceSha,admitted:false,authorized:false,
    taskPass:false,agentLaunch:false,releasePass:false});
}

/**
 * T06l — Business projection of the sealed C accepted-Claim observation.
 * Positive output is admission of THIS K intent only: it never authorizes
 * the caller, launches an Agent, passes the Task or releases. Any deviation
 * from the exact sealed positive shape fails closed to NOT_ADMITTED.
 */
const ADMIT_REASON='CLAIM_REGISTRY_PUBLICATION_AND_NATIVE_READBACK_VERIFIED';
const CLAIM_PUBLICATION='REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK';
const deniedFlags=Object.freeze({authorized:false,admitted:false,taskPass:false,
  agentLaunch:false,releasePass:false});
const notAdmitted=(reason,effectId,claimObservation)=>Object.freeze({
  state:'NOT_ADMITTED',reason,effectId,
  claimObservation:claimObservation??null,...deniedFlags,
});
export function projectClaimedTaskAdmission({input,claimObservation,effectId}={}){
  try{
    const o=claimObservation;
    if(!o||typeof o!=='object'||Array.isArray(o)){
      return notAdmitted('UNVERIFIED_CLAIM_ADMISSION_OBSERVATION',effectId,o??null);
    }
    if(o.state==='ADMITTED_PENDING_AGENT_EFFECT'){
      const positive=
        o.reason===ADMIT_REASON&&
        o.issueRef===input?.issueRef&&o.sourceSha===input?.sourceSha&&
        DIAG_ISSUE.test(o.issueRef??'')&&DIAG_SOURCE_SHA.test(o.sourceSha??'')&&
        typeof o.protectedClaimKey==='string'&&
        /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#[1-9][0-9]*:(builder|reviewer|validator):__default__$/
          .test(o.protectedClaimKey)&&
        o.generation===1&&
        typeof o.dispatchId==='string'&&o.dispatchId.length>0&&
        o.actorRole==='builder'&&
        typeof o.operatorKind==='string'&&o.operatorKind.length>0&&
        typeof o.operatorId==='string'&&o.operatorId.length>0&&
        Number.isSafeInteger(o.commentId)&&o.commentId>0&&
        o.commentRef===`${o.issueRef}#issuecomment-${o.commentId}`&&
        o.claimPublication===CLAIM_PUBLICATION&&
        o.claimAccepted===true&&
        o.admitted===true&&o.authorized===false&&o.taskPass===false&&
        o.agentLaunch===false&&o.releasePass===false;
      if(!positive)return notAdmitted('UNVERIFIED_CLAIM_ADMISSION_OBSERVATION',effectId,o);
      const observation=Object.freeze({
        state:'ADMITTED_PENDING_AGENT_EFFECT',reason:o.reason,
        issueRef:o.issueRef,sourceSha:o.sourceSha,
        protectedClaimKey:o.protectedClaimKey,generation:1,
        dispatchId:o.dispatchId,actorRole:o.actorRole,
        operatorKind:o.operatorKind,operatorId:o.operatorId,
        commentId:o.commentId,commentRef:o.commentRef,
        claimPublication:o.claimPublication,claimAccepted:true,
        admitted:true,authorized:false,taskPass:false,
        agentLaunch:false,releasePass:false,
      });
      return Object.freeze({
        state:'ADMITTED_PENDING_AGENT_EFFECT',reason:o.reason,effectId,
        claimObservation:observation,
        admitted:true,authorized:false,taskPass:false,
        agentLaunch:false,releasePass:false,
      });
    }
    if(o.state==='NOT_ADMITTED'&&typeof o.reason==='string'&&o.reason.length>0&&
       o.reason.length<=64&&o.admitted===false&&o.authorized===false&&
       o.taskPass===false&&o.agentLaunch===false&&o.releasePass===false){
      return notAdmitted(o.reason,effectId,o);
    }
    return notAdmitted('UNVERIFIED_CLAIM_ADMISSION_OBSERVATION',effectId,o);
  }catch{return notAdmitted('UNTRUSTED_CLAIM_ADMISSION_OBSERVATION',effectId,null);}
}

const LOCAL_ATTEMPT_ID=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
export function assessLocalCodexAttempt(input){
  if(!input||typeof input!=='object'||Array.isArray(input))
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_INPUT',admitted:false,
      authorized:false,agentLaunch:false,taskPass:false,releasePass:false});
  const {issueRef,issueNumber,sourceSha,attemptId}=input;
  if(!(ISSUE_REF.test(issueRef)&&Number.isSafeInteger(issueNumber)&&issueNumber>0&&
    issueRef.endsWith(`/issues/${issueNumber}`)&&SOURCE_SHA.test(sourceSha)&&
    LOCAL_ATTEMPT_ID.test(attemptId??'')))
    return Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',admitted:false,
      authorized:false,agentLaunch:false,taskPass:false,releasePass:false});
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',reason:'REQUIRES_SEALED_LOCAL_CODEX_ATTEMPT',
    issueRef,issueNumber,sourceSha,attemptId,admitted:false,authorized:false,
    agentLaunch:false,taskPass:false,releasePass:false});
}

const LOCAL_REASONS=new Set([
  'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','CLAIM_REGISTRY_UNAVAILABLE',
  'CLAIM_REGISTRY_RECORD_NOT_FOUND','CLAIM_REGISTRY_RECORD_INVALID',
  'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED','CLAIM_REGISTRY_RESERVATION_MISMATCH',
  'DAG_WRITESET_NOT_VERIFIED','NATIVE_DEPENDENCIES_BLOCKED','NATIVE_SOURCE_NOT_CURRENT',
  'NATIVE_CLAIM_WITNESS_NOT_VERIFIED','LOCAL_AGENT_UNAVAILABLE',
  'BUILDER_CONFINEMENT_UNAVAILABLE','HIDDEN_VERIFIER_UNAVAILABLE',
  'ATTEMPT_ALREADY_RECONCILED','ATTEMPT_STATUS_UNKNOWN','ATTEMPT_TIMEOUT_RECONCILED',
  'ATTEMPT_CANCELLED','ATTEMPT_PROCESS_FAILED','WRITE_SET_VIOLATION',
  'CANDIDATE_TREE_UNAVAILABLE','LOCAL_AGENT_PROVIDER_THREW',
  'CANDIDATE_WORKTREE_OVERLAP','CANDIDATE_WRITE_GRANT_OVERLAP'
]);
const localDenied=(reason,effectId)=>Object.freeze({state:'NOT_ADMITTED',reason,effectId,
  admitted:false,claimAccepted:false,authorized:false,agentLaunch:false,
  taskPass:false,releasePass:false});
export function projectLocalCodexAttempt({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.attemptId!==input?.attemptId||
      !LOCAL_REASONS.has(o.reason)&&o.state!=='LOCAL_CANDIDATE_READY'&&o.state!=='REPLAY_COMPLETE')
      return localDenied('UNVERIFIED_LOCAL_CODEX_OBSERVATION',effectId);
    const common={issueRef:o.issueRef,sourceSha:o.sourceSha,attemptId:o.attemptId,
      claimGeneration:o.claimGeneration,claimAccepted:o.claimAccepted,
      admitted:o.admitted,authorized:o.authorized,agentLaunch:o.agentLaunch,
      taskPass:o.taskPass,releasePass:o.releasePass};
    if(o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
      o.claimGeneration!==1)return localDenied('UNVERIFIED_LOCAL_CODEX_OBSERVATION',effectId);
    if(o.state==='NOT_ADMITTED'&&LOCAL_REASONS.has(o.reason)&&o.admitted===false&&
      o.claimAccepted===false&&o.agentLaunch===false)
      return Object.freeze({state:o.state,reason:o.reason,effectId,...common});
    if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
      o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&o.replayFixture===true)
      return Object.freeze({state:o.state,reason:o.reason,effectId,...common,replayFixture:true,
        candidateTree:null,candidateCommit:null,process:null});
    if(o.state==='LOCAL_CANDIDATE_READY'&&o.reason==='REAL_CODEX_CANDIDATE_RECORDED'&&
      o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===true&&
      o.replayFixture===false&&SOURCE_SHA.test(o.candidateTree??'')&&
      SOURCE_SHA.test(o.candidateCommit??'')&&o.process?.status==='EXITED'&&
      o.process.exitCode===0&&typeof o.process.builderPrincipal==='string'&&
      typeof o.process.builderSid==='string'&&Number.isSafeInteger(o.process.pid))
      return Object.freeze({state:o.state,reason:o.reason,effectId,...common,replayFixture:false,
        candidateTree:o.candidateTree,candidateCommit:o.candidateCommit,
        process:Object.freeze({status:'EXITED',exitCode:0,pid:o.process.pid,
          builderPrincipal:o.process.builderPrincipal,builderSid:o.process.builderSid,
          startedAt:o.process.startedAt,terminatedAt:o.process.terminatedAt})});
    if(o.state==='AGENT_NOT_EXECUTED'&&LOCAL_REASONS.has(o.reason)&&
      o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false)
      return Object.freeze({state:o.state,reason:o.reason,effectId,...common,replayFixture:false,
        candidateTree:null,candidateCommit:null,process:null});
    if(['LOCAL_ATTEMPT_UNKNOWN','LOCAL_ATTEMPT_FAILED','LOCAL_ATTEMPT_CANCELLED'].includes(o.state)&&
      LOCAL_REASONS.has(o.reason)&&o.admitted===true&&o.claimAccepted===true&&
      o.agentLaunch===true&&o.replayFixture===false&&typeof o.process?.status==='string')
      return Object.freeze({state:o.state,reason:o.reason,effectId,...common,replayFixture:false,
        candidateTree:SOURCE_SHA.test(o.candidateTree??'')?o.candidateTree:null,
        candidateCommit:SOURCE_SHA.test(o.candidateCommit??'')?o.candidateCommit:null,
        process:Object.freeze({status:o.process.status,
          exitCode:Number.isSafeInteger(o.process.exitCode)?o.process.exitCode:null,
          pid:Number.isSafeInteger(o.process.pid)?o.process.pid:null,
          builderPrincipal:o.process.builderPrincipal,builderSid:o.process.builderSid,
          startedAt:o.process.startedAt,terminatedAt:o.process.terminatedAt})});
    return localDenied('UNVERIFIED_LOCAL_CODEX_OBSERVATION',effectId);
  }catch{return localDenied('UNTRUSTED_LOCAL_CODEX_OBSERVATION',effectId);}
}

/**
 * T06n/#176 — advisory input screening for the reviewer Join intent.
 * The reviewer-accepted Claim fact and the exact candidate readback can only
 * come from the selected sealed C; no input shape, caller assertion or
 * candidate list supplied by an Agent is Join evidence.
 */
const JOIN_CANDIDATE_ISSUES=new Set([172,177,178]);
export function assessReviewedCandidateJoin(input){
  const unqualified=Object.freeze({state:'UNQUALIFIED_INPUT',reason:'MISSING_INPUT',
    admitted:false,claimAccepted:false,authorized:false,agentLaunch:false,
    taskPass:false,releasePass:false});
  if(!input||typeof input!=='object'||Array.isArray(input))return unqualified;
  const {issueRef,issueNumber,sourceSha,joinId,candidates}=input;
  if(!(ISSUE_REF.test(issueRef)&&Number.isSafeInteger(issueNumber)&&issueNumber>0&&
    issueRef.endsWith(`/issues/${issueNumber}`)&&SOURCE_SHA.test(sourceSha)&&
    LOCAL_ATTEMPT_ID.test(joinId??'')&&Array.isArray(candidates)&&candidates.length>=2))
    return Object.freeze({...unqualified,reason:'MISSING_SOURCE_IDENTITY'});
  for(const c of candidates){
    if(!c||typeof c!=='object'||Array.isArray(c)||
      !JOIN_CANDIDATE_ISSUES.has(c.issueNumber)||
      !LOCAL_ATTEMPT_ID.test(c.attemptId??'')||
      !SOURCE_SHA.test(c.candidateCommit??'')||!SOURCE_SHA.test(c.candidateTree??''))
      return Object.freeze({...unqualified,reason:'MISSING_SOURCE_IDENTITY'});
  }
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',
    reason:'REQUIRES_SEALED_REVIEWER_JOIN_OBSERVATION',
    issueRef,issueNumber,sourceSha,joinId,candidates,admitted:false,claimAccepted:false,
    authorized:false,agentLaunch:false,taskPass:false,releasePass:false});
}

const JOIN_REASONS=new Set([
  ...LOCAL_REASONS,'JOIN_INPUT_INVALID','CANDIDATE_NOT_FOUND','CANDIDATE_NOT_READY',
  'CANDIDATE_STALE','CANDIDATE_READBACK_MISMATCH','CANDIDATE_WRITESET_OVERLAP',
  'REVIEWER_SELF_ASSERTION'
]);
const joinDenied=(reason,effectId)=>Object.freeze({state:'NOT_ADMITTED',reason,effectId,
  admitted:false,claimAccepted:false,authorized:false,agentLaunch:false,
  joinedAt:null,candidates:[],taskPass:false,releasePass:false});
/**
 * T06n — Business projection of the sealed C reviewer-Join observation.
 * Positive output verifies only that THIS K intent joined the exact reviewed
 * candidates: it never authorizes the caller, upgrades Task/Release PASS or
 * substitutes for the Host's independent candidate readback.
 */
export function projectReviewedCandidateJoin({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.joinId!==input?.joinId)
      return joinDenied('UNVERIFIED_REVIEWER_JOIN_OBSERVATION',effectId);
    if(o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
      o.claimGeneration!==1)
      return joinDenied('UNVERIFIED_REVIEWER_JOIN_OBSERVATION',effectId);
    if(o.state==='NOT_ADMITTED'&&JOIN_REASONS.has(o.reason)&&o.admitted===false&&
      o.claimAccepted===false&&o.agentLaunch===false)
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,joinId:o.joinId,
        claimGeneration:o.claimGeneration,claimAccepted:false,admitted:false,
        authorized:false,agentLaunch:false,joinedAt:null,candidates:[],
        taskPass:false,releasePass:false});
    if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
      o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&
      o.replayFixture===true)
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,joinId:o.joinId,
        claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
        agentLaunch:false,replayFixture:true,joinedAt:null,candidates:[],
        taskPass:false,releasePass:false});
    if(o.state==='REVIEW_JOINED'&&o.reason==='REVIEWED_CANDIDATES_VERIFIED'&&
      o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&
      o.replayFixture===false&&typeof o.joinedAt==='string'&&
      Array.isArray(o.candidates)&&o.candidates.length===input?.candidates?.length&&
      o.candidates.every((c,i)=>c&&c.attemptId===input.candidates[i].attemptId&&
        SOURCE_SHA.test(c.candidateTree??'')&&SOURCE_SHA.test(c.candidateCommit??'')&&
        typeof c.candidateRelativePath==='string'))
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,joinId:o.joinId,
        claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
        agentLaunch:false,replayFixture:false,joinedAt:o.joinedAt,
        candidates:Object.freeze(o.candidates.map((c,i)=>Object.freeze({
          attemptId:c.attemptId,issueRef:c.issueRef,
          issueNumber:input.candidates[i].issueNumber,
          candidateRelativePath:c.candidateRelativePath,candidateTree:c.candidateTree,
          candidateCommit:c.candidateCommit,candidateRef:c.candidateRef??null,
          startedAt:c.startedAt??null,terminatedAt:c.terminatedAt??null}))),
        taskPass:false,releasePass:false});
    return joinDenied('UNVERIFIED_REVIEWER_JOIN_OBSERVATION',effectId);
  }catch{return joinDenied('UNTRUSTED_REVIEWER_JOIN_OBSERVATION',effectId);}
}
