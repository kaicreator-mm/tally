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
export const INTENT_TYPES = Object.freeze(['inspectTaskReadiness','admitClaimedTask','runLocalCodexAttempt','joinReviewedCandidates','publishReviewedCandidate','freezeProductIntake','admitProjectIntake','publishMaintenanceIssue','recordTaskLearningOutcome']);
export const EFFECT_TYPES = Object.freeze(['effect:tally-readiness-observation',
  'effect:tally-claim-execution-admission','effect:tally-local-codex-attempt',
  'effect:tally-reviewer-candidate-join','effect:tally-reviewed-candidate-publication',
  'effect:tally-product-intake-freeze','effect:tally-project-intake-admission',
  'effect:tally-maintenance-issue-publication','effect:tally-task-learning-outcome']);
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
const freezeInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  productDigest:{source:'event',path:['payload','productDigest']},
  author:{source:'event',path:['payload','author']},
  reviewer:{source:'event',path:['payload','reviewer']},
});
const intakeInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  productDigest:{source:'event',path:['payload','productDigest']},
  projectTitle:{source:'event',path:['payload','projectTitle']},
});
const maintenanceInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  candidateDigest:{source:'event',path:['payload','candidateDigest']},
});
const learningInputFrom=Object.freeze({
  issueRef:{source:'event',path:['payload','issueRef']},
  issueNumber:{source:'event',path:['payload','issueNumber']},
  sourceSha:{source:'event',path:['payload','sourceSha']},
  learningId:{source:'event',path:['payload','learningId']},
  decision:{source:'event',path:['payload','decision']},
  // The later independent outcome rides as always-present scalars (null in
  // the decision phase): the S SDK refuses to project absent keys, so a
  // nested optional object can never cross the K event boundary.
  outcomeAttested:{source:'event',path:['payload','outcomeAttested']},
  outcomeObservedAt:{source:'event',path:['payload','outcomeObservedAt']},
  outcomeNote:{source:'event',path:['payload','outcomeNote']},
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
      effectType:'effect:tally-reviewer-candidate-join',inputFrom:joinInputFrom
      // A new message may retry a transient denial with the same joinId;
      // successful Host JOINED records remain idempotent without a K-level key.
    }]
  };
  const joinWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[joinTransition]},finalState],
  };
  const publicationTransition={
    transitionKey:'publish-reviewed-candidate',
    trigger:{kind:'event',eventType:'TALLY_REVIEWED_CANDIDATE_PUBLICATION_REQUESTED'},
    targetState:'open',effectIntents:[{
      effectType:'effect:tally-reviewed-candidate-publication',inputFrom:localAgentInputFrom
    }]
  };
  const publicationWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[publicationTransition]},finalState],
  };
  // T210 J02: independent Product Review/Freeze is a Host CAS write gated on
  // reviewer independence; the caller can only REQUEST the sealed freeze.
  const freezeTransition={
    transitionKey:'freeze-product-intake',
    trigger:{kind:'event',eventType:'TALLY_PRODUCT_INTAKE_FREEZE_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-product-intake-freeze',inputFrom:freezeInputFrom,
      idempotencyKeyFrom:{template:'freeze:{issueNumber}:{sourceSha}:{productDigest}'}
    }]
  };
  const freezeWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[freezeTransition]},finalState],
  };
  // T210 J01b: governed create/import of the target project. Authority is
  // ONLY the sealed C observation (accepted Claim + frozen write-set + frozen
  // independent review + canonical GitHub readback), never the caller.
  const intakeTransition={
    transitionKey:'admit-project-intake',
    trigger:{kind:'event',eventType:'TALLY_PROJECT_INTAKE_ADMISSION_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-project-intake-admission',inputFrom:intakeInputFrom,
      idempotencyKeyFrom:{template:'intake:{issueNumber}:{sourceSha}:{productDigest}'}
    }]
  };
  const intakeWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[intakeTransition]},finalState],
  };
  // J219 J08b: the only authorized native Issue publication for a reviewed
  // maintenance candidate. Authority is ONLY the sealed C observation
  // (accepted Claim + frozen write-set + frozen independent candidate review
  // + canonical GitHub readback); a lost acknowledgment stays UNKNOWN.
  const maintenanceTransition={
    transitionKey:'publish-maintenance-issue',
    trigger:{kind:'event',eventType:'TALLY_MAINTENANCE_ISSUE_PUBLICATION_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-maintenance-issue-publication',inputFrom:maintenanceInputFrom,
      idempotencyKeyFrom:{template:'maint-issue:{issueNumber}:{sourceSha}:{candidateDigest}'}
    }]
  };
  const maintenanceWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[maintenanceTransition]},finalState],
  };
  // J219 J08c: measured Task Learning over existing public ADS reports. The
  // Host CAS ledger is the only record; provenance stays reported-only with an
  // explicit use/reject decision and a later independent outcome — never a
  // quality-gain inference and never an Agent ranking.
  const learningTransition={
    transitionKey:'record-task-learning-outcome',
    trigger:{kind:'event',eventType:'TALLY_TASK_LEARNING_OUTCOME_REQUESTED'},
    targetState:'open',
    effectIntents:[{
      effectType:'effect:tally-task-learning-outcome',inputFrom:learningInputFrom,
      idempotencyKeyFrom:{template:'learning:{issueNumber}:{sourceSha}:{learningId}'}
    }]
  };
  const learningWorkflow={
    workflowKey:'tally-readiness',initialState:'open',initialContext:{},
    guards:[],states:[{stateKey:'open',transitions:[learningTransition]},finalState],
  };
  return {
    packageId:MODULE_ID,policyId:'tally-readiness',domainId:'tally-engineering',
    workflowDefinition:{
      workflowKey:'tally-readiness',initialState:'open',initialContext:{},
      guards:[],states:[
        {stateKey:'open',transitions:[inspectTransition,admitTransition,localAgentTransition,joinTransition,publicationTransition,freezeTransition,intakeTransition,maintenanceTransition,learningTransition]},
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
    },'rule:tally-reviewed-candidate-publication':{
      ruleId:'rule:tally-reviewed-candidate-publication',
      eventType:'TALLY_REVIEWED_CANDIDATE_PUBLICATION_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',attemptId:'attemptId'},
    },'rule:tally-product-intake-freeze':{
      ruleId:'rule:tally-product-intake-freeze',
      eventType:'TALLY_PRODUCT_INTAKE_FREEZE_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',
        productDigest:'productDigest',author:'author',reviewer:'reviewer'},
    },'rule:tally-project-intake-admission':{
      ruleId:'rule:tally-project-intake-admission',
      eventType:'TALLY_PROJECT_INTAKE_ADMISSION_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',
        productDigest:'productDigest',projectTitle:'projectTitle'},
    },'rule:tally-maintenance-issue-publication':{
      ruleId:'rule:tally-maintenance-issue-publication',
      eventType:'TALLY_MAINTENANCE_ISSUE_PUBLICATION_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',
        candidateDigest:'candidateDigest'},
    },'rule:tally-task-learning-outcome':{
      ruleId:'rule:tally-task-learning-outcome',
      eventType:'TALLY_TASK_LEARNING_OUTCOME_REQUESTED',
      payloadFromInput:{issueRef:'issueRef',issueNumber:'issueNumber',sourceSha:'sourceSha',
        learningId:'learningId',decision:'decision',outcomeAttested:'outcomeAttested',
        outcomeObservedAt:'outcomeObservedAt',outcomeNote:'outcomeNote'},
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
      publishReviewedCandidate:{
        trigger:{kind:'event',eventType:'TALLY_REVIEWED_CANDIDATE_PUBLICATION_REQUESTED'},
        ruleId:'rule:tally-reviewed-candidate-publication',
        roles:['requester','planner','reviewer'],definition:publicationWorkflow,
      },
      freezeProductIntake:{
        trigger:{kind:'event',eventType:'TALLY_PRODUCT_INTAKE_FREEZE_REQUESTED'},
        ruleId:'rule:tally-product-intake-freeze',
        // Freeze authority is ONLY the Host CAS record written after the
        // sealed reviewer-independence check; the caller role gates only who
        // may REQUEST the freeze.
        roles:['requester','planner','builder','reviewer'],definition:freezeWorkflow,
      },
      admitProjectIntake:{
        trigger:{kind:'event',eventType:'TALLY_PROJECT_INTAKE_ADMISSION_REQUESTED'},
        ruleId:'rule:tally-project-intake-admission',
        // Admission authority is ONLY the sealed C observation: accepted
        // Claim, frozen DAG write-set, frozen independent Product review and
        // canonical GitHub readback. No caller/UI/Agent credential.
        roles:['requester','planner','builder','reviewer'],definition:intakeWorkflow,
      },
      publishMaintenanceIssue:{
        trigger:{kind:'event',eventType:'TALLY_MAINTENANCE_ISSUE_PUBLICATION_REQUESTED'},
        ruleId:'rule:tally-maintenance-issue-publication',
        // Publication authority is ONLY the sealed C observation: accepted
        // Claim on #219, frozen DAG write-set, frozen independent candidate
        // review and canonical GitHub Issue readback. The prior release stays
        // immutable; a lost acknowledgment stays UNKNOWN without duplicates.
        roles:['requester','planner','builder','reviewer'],definition:maintenanceWorkflow,
      },
      recordTaskLearningOutcome:{
        trigger:{kind:'event',eventType:'TALLY_TASK_LEARNING_OUTCOME_REQUESTED'},
        ruleId:'rule:tally-task-learning-outcome',
        // Learning authority is ONLY the Host CAS ledger derived from
        // existing public ADS reports with exact provenance. Explicit
        // use/reject and a later independent outcome are recorded; no
        // quality-gain inference and no Agent ranking can cross this intent.
        roles:['requester','planner','builder','reviewer'],definition:learningWorkflow,
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
      'effect:tally-reviewed-candidate-publication':{
        semantics:'non-idempotent',
        resource:{key:'tally-local-codex',operation:'tally.publishReviewedCandidate'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          const advisory=assessCandidatePublication(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call('tally-local-codex','tally.publishReviewedCandidate',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,sourceSha:input.sourceSha,
              attemptId:input.attemptId,effectId});
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectCandidatePublication({input,observation,effectId});
        },
      },
      'effect:tally-product-intake-freeze':{
        // Idempotent: the Host CAS create replays an identical frozen record
        // and refuses a divergent digest. GitHub is never touched here.
        semantics:'idempotent',
        resource:{key:'tally-local-codex',operation:'tally.freezeProductIntake'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          const advisory=assessProductIntakeFreeze(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call('tally-local-codex','tally.freezeProductIntake',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,productDigest:input.productDigest,
              author:input.author,reviewer:input.reviewer,effectId});
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectProductIntakeFreeze({input,observation,effectId});
        },
      },
      'effect:tally-project-intake-admission':{
        // The only Tally GitHub write path in this slice: selected B -> one K
        // -> Host/GitHub C with accepted Claim, frozen write-set, frozen
        // independent review, repo allowlist and canonical readback.
        semantics:'non-idempotent',
        resource:{key:'tally-local-codex',operation:'tally.hostProjectIntake'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          const advisory=assessProjectIntakeAdmission(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call('tally-local-codex','tally.hostProjectIntake',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,productDigest:input.productDigest,
              projectTitle:input.projectTitle,effectId});
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectProjectIntakeAdmission({input,observation,effectId});
        },
      },
      'effect:tally-maintenance-issue-publication':{
        // J219 J08b: the authorized native Issue publication. Host CAS keeps
        // physical idempotency (pending marker + reconcile); the K effect is
        // non-idempotent so a genuinely new publication can always run.
        semantics:'non-idempotent',
        resource:{key:'tally-local-codex',operation:'tally.publishMaintenanceIssue'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          const advisory=assessMaintenanceIssuePublication(input);
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call('tally-local-codex',
            'tally.publishMaintenanceIssue',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,candidateDigest:input.candidateDigest,effectId});
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectMaintenanceIssuePublication({input,observation,effectId});
        },
      },
      'effect:tally-task-learning-outcome':{
        // J219 J08c: explicit use/reject plus later independent outcome,
        // recorded only in the Host CAS learning ledger. GitHub is never
        // written here and no quality gain or Agent ranking is inferable.
        semantics:'non-idempotent',
        resource:{key:'tally-local-codex',operation:'tally.recordTaskLearningOutcome'},
        external:{permitTtlMs:3600000,await:'terminal'},
        handler:async ({input,effectId,resources})=>{
          if(!resources||typeof resources.call!=='function')
            throw new Error('SELECTED_LOCAL_CODEX_CAPABILITY_REQUIRED');
          // Reassemble the bounded outcome object from the always-present
          // scalar wire fields (null in the decision phase) before the
          // advisory screening and the sealed C call.
          const outcome=input.outcomeAttested===null||input.outcomeAttested===undefined?
            undefined:{
              attested:input.outcomeAttested===true,
              observedAt:input.outcomeObservedAt,
              note:input.outcomeNote,
            };
          const advisory=assessTaskLearningOutcome({...input,outcome});
          if(advisory.state!=='OBSERVED_INPUT_ONLY')return {...advisory,effectId};
          const envelope=await resources.call('tally-local-codex',
            'tally.recordTaskLearningOutcome',
            {sourceRef:input.issueRef,issueNumber:input.issueNumber,
              sourceSha:input.sourceSha,learningId:input.learningId,
              decision:input.decision,
              outcomeAttested:outcome!==undefined?outcome.attested:null,
              outcomeObservedAt:input.outcomeObservedAt??null,
              outcomeNote:input.outcomeNote??null,effectId});
          const observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
            envelope.outcome.output:null;
          return projectTaskLearningOutcome({input:{...input,outcome},observation,effectId});
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
  captureCoverage:'NOT_COMPARABLE',metricComparability:'NOT_COMPARABLE',
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
const isoTime=value=>typeof value==='string'&&Number.isFinite(Date.parse(value))&&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
const ATTEMPT_STATES=new Set(['LOCAL_CANDIDATE_READY','LOCAL_ATTEMPT_UNKNOWN',
  'LOCAL_ATTEMPT_FAILED','LOCAL_ATTEMPT_CANCELLED']);
/** Source pair must come from selected native GitHub C, never UI/LLM. */
export function projectNativeTaskEffectiveSnapshot({
  sourcePair,expectedIssueRef,expectedSourceSha,observedAt,hostObservation
}={}){
  try{
    if(!DIAG_ISSUE.test(expectedIssueRef??'')||
      !DIAG_SOURCE_SHA.test(expectedSourceSha??''))
      return readModelOutput('EXPECTED_SOURCE_IDENTITY_UNAVAILABLE');
    if(observedAt!==undefined&&!isoTime(observedAt))
      return readModelOutput('OBSERVATION_TIME_UNAVAILABLE');
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
      observedAsOf:observedAt??null,
      currentness:observedAt===undefined?'AGE_NOT_VERIFIED':'OBSERVED_AT_QUERY_COMPLETION',
      dependencyState:w.dependencyState,
      openBlockerCount:Number.isSafeInteger(w.openBlockerCount)?
        w.openBlockerCount:null,
      workflowState:typeof w.workflowState==='string'?w.workflowState:null,
      claimState:'NOT_VERIFIED',kernelTaskState:'NOT_VERIFIED',
      agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',
      ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',
      releaseState:'NOT_VERIFIED',captureCoverage:'NOT_COMPARABLE',
      metricComparability:'NOT_COMPARABLE',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    };
    let observed=base;
    if(hostObservation!==undefined){
      const attempt=hostObservation?.attempt,join=hostObservation?.join;
      const validAttempt=attempt&&typeof attempt==='object'&&!Array.isArray(attempt)&&
        attempt.issueRef===expectedIssueRef&&attempt.sourceSha===expectedSourceSha&&
        LOCAL_ATTEMPT_ID.test(attempt.attemptId??'')&&ATTEMPT_STATES.has(attempt.state)&&
        attempt.authorized===false&&attempt.taskPass===false&&attempt.releasePass===false&&
        attempt.process?.status==='EXITED'&&attempt.process.exitCode===0&&
        isoTime(attempt.process.startedAt)&&isoTime(attempt.process.terminatedAt)&&
        Date.parse(attempt.process.startedAt)<=Date.parse(attempt.process.terminatedAt)&&
        (attempt.state!=='LOCAL_CANDIDATE_READY'||
          DIAG_SOURCE_SHA.test(attempt.candidateTree??'')&&
          DIAG_SOURCE_SHA.test(attempt.candidateCommit??''));
      if(!validAttempt)return readModelOutput('HOST_OBSERVATION_NOT_VERIFIED');
      const validJoin=join===undefined||join&&typeof join==='object'&&!Array.isArray(join)&&
        join.state==='REVIEW_JOINED'&&join.issueRef===expectedIssueRef&&
        join.sourceSha===expectedSourceSha&&LOCAL_ATTEMPT_ID.test(join.joinId??'')&&
        join.authorized===false&&join.taskPass===false&&join.releasePass===false&&
        Array.isArray(join.candidates)&&join.candidates.length>=2&&
        join.candidates.every(c=>c&&LOCAL_ATTEMPT_ID.test(c.attemptId??'')&&
          DIAG_SOURCE_SHA.test(c.candidateTree??'')&&DIAG_SOURCE_SHA.test(c.candidateCommit??''))&&
        join.candidates.some(c=>c.attemptId===attempt.attemptId);
      if(!validJoin)return readModelOutput('HOST_OBSERVATION_NOT_VERIFIED');
      observed={...base,agentAttemptState:attempt.state,kernelTaskState:attempt.state,
        hostState:join?join.state:attempt.state,reviewState:join?join.state:'NOT_VERIFIED'};
    }
    if(w.dependencyState==='BLOCKED')
      return readModelAdvice(observed,'WAIT_EXTERNAL','NATIVE_DEPENDENCIES_BLOCKED',
        'WAIT_NATIVE_DEPENDENCIES','READ_ONLY');
    if(w.dependencyState!=='CLEAR')
      return readModelOutput('NATIVE_DEPENDENCIES_UNKNOWN');
    if(w.issueState!=='open'||w.workflowState!=='state:ready'||
      w.workItemType!=='type:task'||w.reason!=='CLAIM_NOT_VERIFIED')
      return readModelAdvice(observed,'WAIT_EXTERNAL','TASK_NOT_NATIVE_READY',
        'WAIT_NATIVE_TASK_READY','READ_ONLY');
    if(c.reason==='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'){
      if(c.candidateCount>1)
        return readModelAdvice(observed,'CONFLICT','AMBIGUOUS_CLAIM_EVENT_CANDIDATES',
          'RECONCILE_MULTIPLE_NATIVE_CLAIMS','HOST_REQUIRED');
      if(c.candidateCount===1)
        return readModelAdvice(observed,'WAIT_EXTERNAL',
          'SELF_REPORTED_CLAIM_NEEDS_HOST_RECONCILIATION',
          'RECONCILE_NATIVE_CLAIM','HOST_REQUIRED');
    }
    if(c.reason!=='NO_DISPATCH_CLAIM_EVENTS'||c.candidateCount!==0)
      return readModelOutput('NATIVE_CLAIM_ABSENCE_UNVERIFIED');
    if(hostObservation!==undefined){
      if(hostObservation.join)
        return readModelAdvice(observed,'REVIEW_CANDIDATE_EVIDENCE','REVIEW_JOINED',
          'REVIEW_CANDIDATE_EVIDENCE','READ_ONLY');
      if(hostObservation.attempt.state==='LOCAL_CANDIDATE_READY')
        return readModelAdvice(observed,'AWAITING_REVIEW','LOCAL_CANDIDATE_READY',
          'REQUEST_REVIEWER_JOIN','HOST_AND_ADS_REQUIRED');
      return readModelAdvice(observed,'CONFLICT','AGENT_ATTEMPT_NOT_COMPLETE',
        'RECONCILE_AGENT_ATTEMPT','HOST_REQUIRED');
    }
    return readModelAdvice(observed,'AWAITING_HOST_ADMISSION',
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
const JOIN_CANDIDATE_ISSUES=new Set([172,177,178,196,197,198]);
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

const PUBLICATION_ISSUES=new Set([197,198]);
const PUBLICATION_REASONS=new Set(['PUBLICATION_TASK_NOT_ALLOWED','PUBLICATION_INPUT_INVALID',
  'GITHUB_PROVIDER_TOKEN_UNAVAILABLE','REVIEW_STORE_UNAVAILABLE','CANDIDATE_REVIEW_NOT_FOUND',
  'CANDIDATE_SUPERSEDED_BY_REPAIR','CANDIDATE_READBACK_MISMATCH','CANDIDATE_REF_MISMATCH',
  'PUBLICATION_IDENTITY_CONFLICT','PUBLICATION_CLOSED_OR_MERGED','GITHUB_ACCESS_DENIED',
  'GITHUB_RATE_LIMITED','GITHUB_PUBLICATION_CONFLICT','GITHUB_PUBLICATION_UNKNOWN',
  'GITHUB_REF_PUBLICATION_UNKNOWN','PUBLICATION_READBACK_MISMATCH','LOCAL_AGENT_PROVIDER_THREW',
  'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','CLAIM_REGISTRY_UNAVAILABLE','CLAIM_REGISTRY_RECORD_NOT_FOUND',
  'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED','CLAIM_REGISTRY_RESERVATION_MISMATCH',
  'NATIVE_CLAIM_WITNESS_NOT_VERIFIED','NATIVE_SOURCE_NOT_CURRENT','NATIVE_DEPENDENCIES_BLOCKED']);
const publicationDenied=(reason,effectId,state='NOT_ADMITTED')=>Object.freeze({state,reason,effectId,
  admitted:false,authorized:false,agentLaunch:false,taskPass:false,releasePass:false});
export function assessCandidatePublication(input){
  const unqualified={state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',admitted:false,
    authorized:false,agentLaunch:false,taskPass:false,releasePass:false};
  if(!input||typeof input!=='object'||Array.isArray(input))return Object.freeze(unqualified);
  const {issueRef,issueNumber,sourceSha,attemptId}=input;
  if(!ISSUE_REF.test(issueRef)||!PUBLICATION_ISSUES.has(issueNumber)||
    issueRef!==`https://github.com/kaicreator-mm/tally/issues/${issueNumber}`||
    !SOURCE_SHA.test(sourceSha)||!LOCAL_ATTEMPT_ID.test(attemptId??''))return Object.freeze(unqualified);
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',reason:'REQUIRES_SEALED_REVIEWED_PUBLICATION',
    issueRef,issueNumber,sourceSha,attemptId,admitted:false,authorized:false,agentLaunch:false,
    taskPass:false,releasePass:false});
}
export function projectCandidatePublication({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.attemptId!==input?.attemptId||
      o.authorized!==false||o.taskPass!==false||o.releasePass!==false)
      return publicationDenied('UNVERIFIED_CANDIDATE_PUBLICATION',effectId);
    if(o.state==='NOT_ADMITTED'&&PUBLICATION_REASONS.has(o.reason)&&o.admitted===false)
      return publicationDenied(o.reason,effectId);
    if(o.state==='PUBLICATION_UNKNOWN'&&PUBLICATION_REASONS.has(o.reason))
      return publicationDenied(o.reason,effectId,o.state);
    if(o.state==='PUBLISHED_VERIFIED'&&o.reason==='GITHUB_BRANCH_AND_PR_READBACK_VERIFIED'&&
      o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&o.claimGeneration===1&&
      o.repository==='kaicreator-mm/tally'&&o.baseRef==='version/v2'&&
      SOURCE_SHA.test(o.candidateCommit??'')&&SOURCE_SHA.test(o.candidateTree??'')&&
      Number.isSafeInteger(o.pullNumber)&&o.pullNumber>0&&
      /^https:\/\/github\.com\/kaicreator-mm\/tally\/pull\/[1-9][0-9]*$/.test(o.pullRef??'')&&
      o.externalRef===o.pullRef&&o.branch===`tally/reviewed/${input.attemptId.replace(/[^a-z0-9]+/gi,'-').replace(/^-|-$/g,'').toLowerCase()}`)
      return Object.freeze({state:o.state,reason:o.reason,effectId,issueRef:o.issueRef,
        sourceSha:o.sourceSha,attemptId:o.attemptId,claimGeneration:1,claimAccepted:true,
        admitted:true,authorized:false,agentLaunch:false,repository:o.repository,branch:o.branch,
        baseRef:o.baseRef,candidateCommit:o.candidateCommit,candidateTree:o.candidateTree,
        pullNumber:o.pullNumber,pullRef:o.pullRef,externalRef:o.externalRef,
        taskPass:false,releasePass:false});
    return publicationDenied('UNVERIFIED_CANDIDATE_PUBLICATION',effectId);
  }catch{return publicationDenied('UNTRUSTED_CANDIDATE_PUBLICATION',effectId);}
}

/**
 * T210 J02 — advisory input screening for the Product intake freeze intent.
 * The frozen fact can only come from the selected sealed C writing the Host
 * CAS record after verifying reviewer independence; no input flag or caller
 * assertion is freeze evidence.
 */
const INTAKE_ISSUE=210;
const PRODUCT_DIGEST=/^[0-9a-f]{64}$/;
const IDENTITY=/^[A-Za-z0-9][A-Za-z0-9 .:_'@-]{0,127}$/;
export function assessProductIntakeFreeze(input){
  const unqualified={state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',
    admitted:false,authorized:false,frozen:false,taskPass:false,releasePass:false};
  if(!input||typeof input!=='object'||Array.isArray(input))return Object.freeze(unqualified);
  const {issueRef,issueNumber,sourceSha,productDigest,author,reviewer}=input;
  if(issueRef!==`https://github.com/kaicreator-mm/tally/issues/${INTAKE_ISSUE}`||
    issueNumber!==INTAKE_ISSUE||!SOURCE_SHA.test(sourceSha??'')||
    !PRODUCT_DIGEST.test(productDigest??'')||!IDENTITY.test(author??'')||
    !IDENTITY.test(reviewer??'')||author===reviewer)return Object.freeze(unqualified);
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',
    reason:'REQUIRES_SEALED_PRODUCT_INTAKE_FREEZE',
    issueRef,issueNumber,sourceSha,productDigest,admitted:false,authorized:false,
    frozen:false,taskPass:false,releasePass:false});
}

const FREEZE_REASONS=new Set(['PRODUCT_FREEZE_INPUT_INVALID','PRODUCT_REVIEW_NOT_INDEPENDENT',
  'PRODUCT_FREEZE_STORE_UNAVAILABLE','PRODUCT_FREEZE_ALREADY_RECORDED',
  'PRODUCT_FREEZE_DIGEST_MISMATCH','LOCAL_AGENT_PROVIDER_THREW']);
const freezeDenied=(reason,effectId)=>Object.freeze({state:'NOT_FROZEN',reason,effectId,
  admitted:false,authorized:false,frozen:false,taskPass:false,releasePass:false});
/** Positive output records only THIS K intent's freeze; never Task/Release PASS. */
export function projectProductIntakeFreeze({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.productDigest!==input?.productDigest||
      o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
      o.claimGeneration!==1)
      return freezeDenied('UNVERIFIED_PRODUCT_INTAKE_FREEZE',effectId);
    if(o.state==='NOT_FROZEN'&&FREEZE_REASONS.has(o.reason)&&o.frozen===false)
      return freezeDenied(o.reason,effectId);
    if(o.state==='FROZEN_COMPLETE'&&o.frozen===true&&
      PRODUCT_DIGEST.test(o.productDigest??'')&&
      o.receiptRef===`receipt:sha256:${o.productDigest}`&&
      typeof o.review?.reviewer==='string'&&o.review.reviewer!==o.review?.author)
      return Object.freeze({state:o.state,reason:o.reason??'INDEPENDENT_REVIEW_FROZEN_IN_HOST_CAS',
        effectId,issueRef:o.issueRef,sourceSha:o.sourceSha,productDigest:o.productDigest,
        receiptRef:o.receiptRef,frozen:true,claimGeneration:1,
        review:Object.freeze({author:String(o.review.author),reviewer:String(o.review.reviewer)}),
        admitted:false,authorized:false,taskPass:false,releasePass:false});
    return freezeDenied('UNVERIFIED_PRODUCT_INTAKE_FREEZE',effectId);
  }catch{return freezeDenied('UNTRUSTED_PRODUCT_INTAKE_FREEZE',effectId);}
}

/**
 * T210 J01b — advisory input screening for the project intake admission
 * intent. Repo allowlist is exact (kaicreator-mm/tally#210); cross-repo,
 * stale-source and malformed input refuses before any sealed dispatch.
 */
const INTAKE_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'PROJECT_INTAKE_HOST_CONFIG_INVALID','DAG_WRITESET_NOT_VERIFIED',
  'PRODUCT_FREEZE_NOT_VERIFIED','PRODUCT_FREEZE_DIGEST_MISMATCH',
  'PRODUCT_REVIEW_NOT_INDEPENDENT','PRODUCT_FREEZE_STORE_UNAVAILABLE',
  'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE','GITHUB_PUBLISH_UNKNOWN',
  'PROJECT_INTAKE_READBACK_MISMATCH','PROJECT_INTAKE_INPUT_INVALID']);
const PROJECT_TITLE=/^.{1,120}$/s;
export function assessProjectIntakeAdmission(input){
  const unqualified={state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',
    admitted:false,authorized:false,taskPass:false,releasePass:false};
  if(!input||typeof input!=='object'||Array.isArray(input))return Object.freeze(unqualified);
  const {issueRef,issueNumber,sourceSha,productDigest,projectTitle}=input;
  if(issueRef!==`https://github.com/kaicreator-mm/tally/issues/${INTAKE_ISSUE}`||
    issueNumber!==INTAKE_ISSUE||!SOURCE_SHA.test(sourceSha??'')||
    !PRODUCT_DIGEST.test(productDigest??'')||!PROJECT_TITLE.test(projectTitle??''))
    return Object.freeze(unqualified);
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',
    reason:'REQUIRES_SEALED_PROJECT_INTAKE_OBSERVATION',
    issueRef,issueNumber,sourceSha,productDigest,projectTitle,
    admitted:false,authorized:false,taskPass:false,releasePass:false});
}

const intakeDenied=(reason,effectId,state='NOT_ADMITTED')=>Object.freeze({state,reason,
  effectId,admitted:false,authorized:false,taskPass:false,releasePass:false});
/**
 * T210 — Business projection of the sealed C project-intake observation.
 * Positive output admits THIS K intent only: it never authorizes the caller,
 * passes the Task or releases. githubEffect stays NOT_RUN unless the Host
 * proved a real publish plus canonical readback.
 */
export function projectProjectIntakeAdmission({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.productDigest!==input?.productDigest||
      o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
      o.claimGeneration!==1)
      return intakeDenied('UNVERIFIED_PROJECT_INTAKE_OBSERVATION',effectId);
    if(o.state==='NOT_ADMITTED'&&INTAKE_REASONS.has(o.reason)&&o.admitted===false&&
      o.githubEffect==='NOT_RUN')
      return intakeDenied(o.reason,effectId);
    if(o.state==='PROJECT_INTAKE_UNKNOWN'&&INTAKE_REASONS.has(o.reason)&&
      o.githubEffect==='UNKNOWN')
      return intakeDenied(o.reason,effectId,o.state);
    if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
      o.admitted===true&&o.githubEffect==='NOT_RUN'&&o.replayFixture===true)
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,productDigest:o.productDigest,
        claimGeneration:1,admitted:true,authorized:false,replayFixture:true,
        githubEffect:'NOT_RUN',project:null,taskPass:false,releasePass:false});
    if(o.state==='PROJECT_INTAKE_ADMITTED'&&
      ['PROJECT_INTAKE_PUBLISHED_AND_READBACK_VERIFIED',
        'PROJECT_INTAKE_READBACK_ALREADY_PRESENT'].includes(o.reason)&&
      o.admitted===true&&o.githubEffect==='PERFORMED_VERIFIED'&&
      o.replayFixture===false&&o.repository==='kaicreator-mm/tally'&&
      o.project&&typeof o.project==='object'&&!Array.isArray(o.project)&&
      Number.isSafeInteger(o.project.number)&&o.project.number>0&&
      PROJECT_TITLE.test(o.project.title??'')&&
      /^https:\/\/github\.com\/kaicreator-mm\/tally\/projects\/[1-9][0-9]*$/.test(o.project.url??''))
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,productDigest:o.productDigest,
        claimGeneration:1,admitted:true,authorized:false,replayFixture:false,
        githubEffect:'PERFORMED_VERIFIED',repository:o.repository,
        project:Object.freeze({number:o.project.number,title:o.project.title,
          url:o.project.url,itemCount:Number.isSafeInteger(o.project.itemCount)?
            o.project.itemCount:null}),
        taskPass:false,releasePass:false});
    return intakeDenied('UNVERIFIED_PROJECT_INTAKE_OBSERVATION',effectId);
  }catch{return intakeDenied('UNTRUSTED_PROJECT_INTAKE_OBSERVATION',effectId);}
}

/**
 * T219 J08b — advisory input screening for the authorized native Issue
 * publication of one reviewed maintenance candidate. Repo/issue allowlist is
 * exact (kaicreator-mm/tally#219); cross-issue, stale-source and malformed
 * digests refuse before any sealed dispatch.
 */
const MAINTENANCE_ISSUE=219;
const CANDIDATE_DIGEST=/^[0-9a-f]{64}$/;
export function assessMaintenanceIssuePublication(input){
  const unqualified={state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',
    admitted:false,authorized:false,taskPass:false,releasePass:false};
  if(!input||typeof input!=='object'||Array.isArray(input))return Object.freeze(unqualified);
  const {issueRef,issueNumber,sourceSha,candidateDigest}=input;
  if(issueRef!==`https://github.com/kaicreator-mm/tally/issues/${MAINTENANCE_ISSUE}`||
    issueNumber!==MAINTENANCE_ISSUE||!SOURCE_SHA.test(sourceSha??'')||
    !CANDIDATE_DIGEST.test(candidateDigest??''))return Object.freeze(unqualified);
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',
    reason:'REQUIRES_SEALED_MAINTENANCE_ISSUE_PUBLICATION',
    issueRef,issueNumber,sourceSha,candidateDigest,
    admitted:false,authorized:false,taskPass:false,releasePass:false});
}

const MAINTENANCE_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'MAINTENANCE_HOST_CONFIG_INVALID','DAG_WRITESET_NOT_VERIFIED',
  'CANDIDATE_REVIEW_NOT_VERIFIED','MAINTENANCE_REVIEW_DIGEST_MISMATCH',
  'CANDIDATE_REVIEW_NOT_INDEPENDENT','MAINTENANCE_FREEZE_STORE_UNAVAILABLE',
  'MAINTENANCE_ISSUE_DIGEST_MISMATCH','GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE',
  'GITHUB_PUBLISH_UNKNOWN','MAINTENANCE_ISSUE_READBACK_MISMATCH',
  'MAINTENANCE_ISSUE_IDENTITY_CONFLICT','MAINTENANCE_ISSUE_INPUT_INVALID',
  'GITHUB_ACCESS_DENIED','GITHUB_RATE_LIMITED','GITHUB_UNAUTHENTICATED',
  'GITHUB_RESOURCE_NOT_FOUND','GITHUB_PUBLICATION_CONFLICT']);
const ISSUE_URL_EXACT=/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/[1-9][0-9]*$/;
const maintenanceDenied=(reason,effectId,state='NOT_ADMITTED')=>Object.freeze({state,
  reason,effectId,admitted:false,authorized:false,taskPass:false,releasePass:false});
/**
 * T219 — Business projection of the sealed C maintenance publication
 * observation. Positive output admits THIS K intent only: it never authorizes
 * the caller, passes the Task or releases. githubEffect stays NOT_RUN unless
 * the Host proved a real publish plus canonical readback; a lost
 * acknowledgment stays PUBLICATION_UNKNOWN with a null Issue and can later
 * reconcile WITHOUT any duplicate.
 */
export function projectMaintenanceIssuePublication({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.candidateDigest!==input?.candidateDigest||
      o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
      o.claimGeneration!==1)
      return maintenanceDenied('UNVERIFIED_MAINTENANCE_ISSUE_PUBLICATION',effectId);
    if(o.state==='NOT_ADMITTED'&&MAINTENANCE_REASONS.has(o.reason)&&
      o.admitted===false&&o.githubEffect==='NOT_RUN'&&o.issue===null)
      return maintenanceDenied(o.reason,effectId);
    if(o.state==='PUBLICATION_UNKNOWN'&&MAINTENANCE_REASONS.has(o.reason)&&
      o.githubEffect==='UNKNOWN'&&o.issue===null)
      return Object.freeze({...maintenanceDenied(o.reason,effectId,o.state),
        githubEffect:'UNKNOWN'});
    if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
      o.admitted===true&&o.claimAccepted===true&&o.githubEffect==='NOT_RUN'&&
      o.replayFixture===true&&o.issue===null)
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,candidateDigest:o.candidateDigest,
        claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
        replayFixture:true,githubEffect:'NOT_RUN',issue:null,
        historicalReleaseMutation:false,taskPass:false,releasePass:false});
    if(o.state==='MAINTENANCE_ISSUE_PUBLISHED'&&
      ['MAINTENANCE_ISSUE_PUBLISHED_AND_READBACK_VERIFIED',
        'MAINTENANCE_ISSUE_READBACK_ALREADY_PRESENT',
        'MAINTENANCE_ISSUE_RECONCILED_AFTER_LOST_ACK'].includes(o.reason)&&
      o.admitted===true&&o.claimAccepted===true&&
      o.githubEffect==='PERFORMED_VERIFIED'&&o.replayFixture===false&&
      o.repository==='kaicreator-mm/tally'&&o.duplicate===false&&
      o.historicalReleaseMutation===false&&
      o.issue&&typeof o.issue==='object'&&!Array.isArray(o.issue)&&
      Number.isSafeInteger(o.issue.number)&&o.issue.number>0&&
      typeof o.issue.title==='string'&&o.issue.title.length>0&&
      ISSUE_URL_EXACT.test(o.issue.url??''))
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,candidateDigest:o.candidateDigest,
        claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
        replayFixture:false,githubEffect:'PERFORMED_VERIFIED',
        repository:o.repository,duplicate:false,alreadyPresent:o.alreadyPresent===true,
        issue:Object.freeze({number:o.issue.number,title:o.issue.title,
          url:o.issue.url}),
        historicalReleaseMutation:false,taskPass:false,releasePass:false});
    return maintenanceDenied('UNVERIFIED_MAINTENANCE_ISSUE_PUBLICATION',effectId);
  }catch{return maintenanceDenied('UNTRUSTED_MAINTENANCE_ISSUE_PUBLICATION',effectId);}
}

/**
 * T219 J08c — advisory input screening for one Task Learning use/reject
 * decision plus its optional later independent outcome. The learning identity
 * must be the exact ledger entry id; the outcome is an observed fact only.
 */
const LEARNING_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'MAINTENANCE_HOST_CONFIG_INVALID','MAINTENANCE_FREEZE_STORE_UNAVAILABLE',
  'LEARNING_PROVENANCE_NOT_VERIFIED','LEARNING_DECISION_REQUIRED',
  'LEARNING_DECISION_MISMATCH','LEARNING_OUTCOME_MISMATCH',
  'MAINTENANCE_INPUT_INVALID','LOCAL_AGENT_PROVIDER_THREW']);
const ISO_TIME=value=>typeof value==='string'&&Number.isFinite(Date.parse(value))&&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
export function assessTaskLearningOutcome(input){
  const unqualified={state:'UNQUALIFIED_INPUT',reason:'MISSING_SOURCE_IDENTITY',
    admitted:false,authorized:false,taskPass:false,releasePass:false};
  if(!input||typeof input!=='object'||Array.isArray(input))return Object.freeze(unqualified);
  const {issueRef,issueNumber,sourceSha,learningId,decision,outcome}=input;
  if(issueRef!==`https://github.com/kaicreator-mm/tally/issues/${MAINTENANCE_ISSUE}`||
    issueNumber!==MAINTENANCE_ISSUE||!SOURCE_SHA.test(sourceSha??'')||
    !CANDIDATE_DIGEST.test(learningId??'')||!['ADOPTED','REJECTED'].includes(decision))
    return Object.freeze(unqualified);
  if(outcome!==undefined){
    if(!outcome||typeof outcome!=='object'||Array.isArray(outcome)||
      typeof outcome.attested!=='boolean'||!ISO_TIME(outcome.observedAt??'')||
      typeof outcome.note!=='string'||outcome.note.length===0||
      outcome.note.length>512)return Object.freeze(unqualified);
  }
  return Object.freeze({state:'OBSERVED_INPUT_ONLY',
    reason:'REQUIRES_SEALED_TASK_LEARNING_RECORD',
    issueRef,issueNumber,sourceSha,learningId,decision,
    ...(outcome!==undefined?{outcome}:{}),
    admitted:false,authorized:false,taskPass:false,releasePass:false});
}

/**
 * T219 — Business projection of the sealed C Task Learning record. Positive
 * output admits THIS K intent only. measuredEffect is NOT_MEASURED unless the
 * outcome was independently attested (ATTESTED_OUTCOME_ONLY); a quality gain
 * is never inferred, Agents are never ranked and GitHub is never written.
 */
export function projectTaskLearningOutcome({input,observation,effectId}={}){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==input?.issueRef||
      o.sourceSha!==input?.sourceSha||o.learningId!==input?.learningId||
      o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
      o.claimGeneration!==1)
      return maintenanceDenied('UNVERIFIED_TASK_LEARNING_RECORD',effectId);
    if(o.state==='NOT_ADMITTED'&&LEARNING_REASONS.has(o.reason)&&
      o.admitted===false&&o.githubEffect==='NOT_RUN')
      return maintenanceDenied(o.reason,effectId);
    if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
      o.admitted===true&&o.claimAccepted===true&&o.githubEffect==='NOT_RUN'&&
      o.replayFixture===true&&o.measuredEffect==='NOT_MEASURED')
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,learningId:o.learningId,
        decision:o.decision,claimGeneration:1,claimAccepted:true,admitted:true,
        authorized:false,replayFixture:true,githubEffect:'NOT_RUN',
        measuredEffect:'NOT_MEASURED',noAgentRanking:true,
        qualityGainClaimed:false,taskPass:false,releasePass:false});
    if(o.state==='LEARNING_RECORDED'&&
      ['LEARNING_DECISION_RECORDED_IN_HOST_CAS','LEARNING_DECISION_ALREADY_RECORDED',
        'LEARNING_OUTCOME_RECORDED_IN_HOST_CAS','LEARNING_OUTCOME_ALREADY_RECORDED']
        .includes(o.reason)&&o.admitted===true&&o.claimAccepted===true&&
      o.githubEffect==='NOT_RUN'&&o.replayFixture===false&&
      ['ADOPTED','REJECTED'].includes(o.decision)&&
      o.noAgentRanking===true&&o.qualityGainClaimed===false&&
      (o.measuredEffect==='NOT_MEASURED'||
        o.measuredEffect==='ATTESTED_OUTCOME_ONLY'&&
        o.outcome&&typeof o.outcome==='object'&&!Array.isArray(o.outcome)&&
        typeof o.outcome.attested==='boolean'))
      return Object.freeze({state:o.state,reason:o.reason,effectId,
        issueRef:o.issueRef,sourceSha:o.sourceSha,learningId:o.learningId,
        decision:o.decision,claimGeneration:1,claimAccepted:true,admitted:true,
        authorized:false,replayFixture:false,githubEffect:'NOT_RUN',
        measuredEffect:o.measuredEffect,
        outcome:o.outcome!==undefined?
          Object.freeze({attested:o.outcome.attested===true,
            measuredEffect:o.outcome.measuredEffect??o.measuredEffect}):
          undefined,
        noAgentRanking:true,qualityGainClaimed:false,
        taskPass:false,releasePass:false});
    return maintenanceDenied('UNVERIFIED_TASK_LEARNING_RECORD',effectId);
  }catch{return maintenanceDenied('UNTRUSTED_TASK_LEARNING_RECORD',effectId);}
}
