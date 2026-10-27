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
          return {
            ...assessTaskEvidence(input),effectId,sourceObservation,
          };
        },
      },
    },
  };
}

// Read-only source evidence diagnosis; only selected K can actually admit.
export {diagnoseNativeTaskReadiness} from './native-source-diagnostic.mjs';
