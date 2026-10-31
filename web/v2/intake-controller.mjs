/** J01–J03 guided intake workbench controller. Drafts stay candidates; the Host owns effects. */
import {nextGuidedStep,guidedCaseRef,GUIDED_TEMPLATE_REF}
  from '../../packages/tally-engineering-business/src/guided-intake.mjs';
import {planNextVersion} from '../../packages/tally-engineering-business/src/planning.mjs';
import {assessProjectIntakeAdmission,projectProjectIntakeAdmission}
  from '../../packages/tally-engineering-business/src/index.mjs';
import {createHash} from 'node:crypto';

const SHA=/^[0-9a-f]{40}$/;
const OWNER='kaicreator-mm',REPO='tally',ISSUE=210;
const ISSUE_REF=`https://github.com/${OWNER}/${REPO}/issues/${ISSUE}`;
const JOURNEY_ID='tally-v2-intake-j01';
const PRODUCT_DIGEST=/^[0-9a-f]{64}$/,TITLE=/^.{1,120}$/s;
const IDENTITY=/^[A-Za-z0-9][A-Za-z0-9 .:_'@-]{0,127}$/;
// Exact frozen write set pinned by the accepted Claim on issue #210 (16 paths).
const WRITE_SET=Object.freeze([
  'web/v2/intake-controller.mjs','web/v2/intake-controller.test.mjs',
  'web/v2/intake-view.mjs','web/v2/intake-view.test.mjs','web/v2/journey-server.mjs',
  'web/v2/journey-view.html','.woodpecker/v2-alm-workspace.yaml',
  'packages/tally-github-c/src/project-intake.mjs',
  'packages/tally-github-c/test/project-intake.test.mjs',
  'experiments/v2/project-intake/host-project.mjs',
  'experiments/v2/project-intake/host-project.test.mjs',
  'packages/tally-engineering-business/src/index.mjs',
  'packages/tally-engineering-business/test/business-policy.test.mjs',
  'experiments/v2/selected-business-build/producer.mjs',
  'experiments/v2/selected-business-build/bundle-local-codex.mjs',
  'experiments/v2/selected-business-build/local-codex-shim.mjs']);
// Real merged sources on version/v2: #157 Product scope, #158 L2 architecture.
const PINNED_SOURCES=Object.freeze([
  Object.freeze({id:'157',issueNumber:157,label:'v2 Product — reviewed P0/V scope',
    sourceRef:`https://github.com/${OWNER}/${REPO}/issues/157`,pinnedBy:'version/v2 merged source'}),
  Object.freeze({id:'158',issueNumber:158,label:'v2 L2 — Domain Package architecture',
    sourceRef:`https://github.com/${OWNER}/${REPO}/issues/158`,pinnedBy:'version/v2 merged source'}),
  Object.freeze({id:'210',issueNumber:ISSUE,label:'J01–J03 governed intake workbench',
    sourceRef:ISSUE_REF,pinnedBy:'accepted builder Claim (dispatch:210:v2:20261031:1)'})]);
const L2_CHOICES=Object.freeze(['KEEP_MERGED_L2_ARCHITECTURE','KEEP_NATIVE_DEPENDENCY',
  'PROVE_MINIMAL_DEPENDENCY_BEFORE_PARALLEL']);
const sha256=value=>createHash('sha256').update(value).digest('hex');
const canonical=value=>Array.isArray(value)?value.map(canonical):
  value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(key=>[key,canonical(value[key])])):value;
const digestOf=value=>sha256(JSON.stringify(canonical(value)));
const issueRefFor=sourceRef=>Number(new URL(sourceRef).pathname.split('/').pop());
const blocked=reason=>({status:'BLOCKED',reason});

function guidedRequest({goals,evidenceRefs,caseRef,previousOutcome}){
  return {messageType:'next-step.request',contractVersion:'dh.guided-journey/1',
    journeyId:JOURNEY_ID,goals:[...goals],caseRef,catalogRef:GUIDED_TEMPLATE_REF,
    eligibleTemplateRefs:[GUIDED_TEMPLATE_REF],...(previousOutcome?{previousOutcome}:{})};
}
function guidedOutcome(stepId,event,evidenceRefs,extra={}){
  return {messageType:'step.outcome',contractVersion:'dh.guided-journey/1',
    journeyId:JOURNEY_ID,stepId,event,evidenceRefs:[...evidenceRefs],...extra};
}
/**
 * J01a read-only first: the workbench shows the real merged sources and the
 * native Task facts; without a Host read every live fact stays explicitly
 * NOT_RUN instead of being invented.
 */
function sourceFacts(pinned,observed){
  return pinned.map(source=>({id:source.id,issueNumber:source.issueNumber,label:source.label,
    sourceRef:source.sourceRef,pinnedBy:source.pinnedBy,
    fact:observed?.[source.id]??{status:'NOT_RUN',reason:'HOST_GITHUB_READ_REQUIRED',
      observedAsOf:null,currentness:'AGE_NOT_VERIFIED'},
    authorized:false,admitted:false}));
}
function dagLanes(tasks){
  const capabilities=tasks.map(task=>({lane:`task-${issueRefFor(task.sourceRef)}`,
    userOutcome:`Native task ${issueRefFor(task.sourceRef)} (${task.status})`,
    owner:'tally-v2',writeSet:[],role:'NO_RECOMMENDATION/MORE_EVIDENCE',
    blockedBy:task.dependencyState==='BLOCKED'?['native-dependency']:[],
    sourceRefs:[task.sourceRef]}));
  capabilities.push({lane:'j01-intake-workbench',
    userOutcome:'Guided project intake, Product/L2 freeze and Task DAG admission',
    owner:'tally-v2',writeSet:[...WRITE_SET],blockedBy:[],sourceRefs:[ISSUE_REF]});
  const plan=planNextVersion({goal:'J01–J03 governed project/Product/L2/DAG workbench',
    nonGoals:['No second writer or scheduler','No UI/Agent GitHub credential'],
    planningAsOf:new Date().toISOString(),
    timeline:{subjectStatus:'OBSERVED',subject:{sourceRef:ISSUE_REF}},
    diagnoses:{status:'OBSERVED',diagnoses:[]},capabilities});
  const byLane=new Map(plan.dag.lanes.map(lane=>[lane.lane,lane]));
  const lanes=tasks.map(task=>{
    const planned=byLane.get(`task-${issueRefFor(task.sourceRef)}`)??{};
    return {lane:planned.lane,sourceRef:task.sourceRef,dependencyState:task.dependencyState,
      claimState:task.claimState,agentAttemptState:task.agentAttemptState,
      reviewState:task.reviewState,writeSetCount:planned.writeSet?.length??0,
      status:planned.status??'CANDIDATE'};
  });
  const intake=byLane.get('j01-intake-workbench')??{};
  lanes.push({lane:intake.lane??'j01-intake-workbench',sourceRef:ISSUE_REF,
    dependencyState:'CLEAR',claimState:'SELF_REPORTED_EVENT_ONLY',
    agentAttemptState:'NOT_RUN',reviewState:'NOT_VERIFIED',
    writeSetCount:intake.writeSet?.length??WRITE_SET.length,status:'CANDIDATE'});
  // Cycle check over lane-level blockedBy edges (native data has none today;
  // the detector stays so a real blockedBy edge can never hide a cycle).
  const names=new Set(lanes.map(lane=>lane.lane));
  const edges=new Map(plan.dag.lanes.map(lane=>[lane.lane,
    (lane.blockedBy??[]).filter(ref=>names.has(ref))]));
  const state=new Map(),cycles=[];
  const visit=(node,stack)=>{state.set(node,'open');
    for(const next of edges.get(node)??[]){
      if(state.get(next)==='open')cycles.push([node,next]);
      else if(!state.has(next))visit(next,stack);}
    state.set(node,'done');};
  for(const node of edges.keys())if(!state.has(node))visit(node,[]);
  return {lanes:Object.freeze(lanes.map(lane=>Object.freeze(lane))),
    conflicts:plan.dag.conflicts,blockedCount:lanes.filter(lane=>lane.dependencyState==='BLOCKED').length,
    cycleCount:cycles.length,candidateCount:plan.coverage?.candidateCount??lanes.length,
    advisoryOnly:plan.advisoryOnly===true};
}
export function createIntakeController(config,{host,journeySnapshot,sealedDispatch,now}={}){
  // `host` is READ-ONLY source-fact reading only. Privileged admit effects
  // flow exclusively through the `sealedDispatch` seam (selected K consumer
  // envelope); the controller never touches a provider or credential.
  if(!SHA.test(config?.currentSourceSha??''))throw new TypeError('CURRENT_SOURCE_SHA_REQUIRED');
  if(config.author!==undefined&&(typeof config.author!=='string'||!IDENTITY.test(config.author)))
    throw new TypeError('AUTHOR_IDENTITY_INVALID');
  const author=config.author??'workbench:operator';
  const currentSha=()=>config.currentSourceSha;
  const state={goals:[],evidenceRefs:[],previousOutcome:null,decision:null,
    draft:null,review:null,reviewBlock:null,frozen:null,intake:null,
    sourceFacts:null,history:[]};
  const caseSnapshot=()=>({goals:[...state.goals],evidenceRefs:[...state.evidenceRefs],
    sourceSha:currentSha()});
  const caseRef=()=>guidedCaseRef(`${JOURNEY_ID}:${currentSha().slice(0,12)}`,caseSnapshot());
  function recomputeDecision(){
    const decision=nextGuidedStep(guidedRequest({goals:state.goals,
      evidenceRefs:state.evidenceRefs,caseRef:caseRef(),
      previousOutcome:state.previousOutcome??undefined}),{caseSnapshot:caseSnapshot()});
    state.decision={status:decision.status,reasonCode:decision.reasonCode,
      step:decision.step?{kind:decision.step.kind,stepId:decision.step.stepId,
        titleKey:decision.step.titleKey,
        fields:decision.step.fields.map(field=>({id:field.id,type:field.type,
          labelKey:field.labelKey,required:field.required,
          options:field.options?.map(option=>({id:option.id,labelKey:option.labelKey,
            eligible:option.eligible}))??[]}))}:null,
      evidenceRefs:[...decision.evidenceRefs]};
    return state.decision;
  }
  function record(actionKind,result){
    state.history.push({action:actionKind,at:new Date().toISOString(),...result});
    return result;
  }
  function stale(sourceSha){return SHA.test(sourceSha??'')&&sourceSha!==currentSha();}
  const actions={
    'submit-goal'(sourceSha,{goal}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(typeof goal!=='string'||goal.trim().length===0||goal.length>256)
        return record('submit-goal',blocked('GOAL_TEXT_REQUIRED'));
      state.goals=[goal.trim()];
      state.previousOutcome=guidedOutcome('ask-goal','USER_SUBMIT',state.evidenceRefs);
      recomputeDecision();
      return record('submit-goal',{status:'OK',decision:state.decision});
    },
    'select-evidence'(sourceSha,{refs}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      const known=new Set(PINNED_SOURCES.map(source=>source.id));
      if(!Array.isArray(refs)||refs.length===0||refs.length>16||
        refs.some(ref=>!known.has(ref)))
        return record('select-evidence',blocked('EVIDENCE_SELECTION_INVALID'));
      state.evidenceRefs=[...new Set(refs)];
      if(state.goals.length>0){
        state.previousOutcome=guidedOutcome('ask-goal','USER_SUBMIT',state.evidenceRefs);
        recomputeDecision();
      }
      return record('select-evidence',{status:'OK',decision:state.decision});
    },
    'confirm-inspect'(sourceSha){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(state.goals.length===0||state.evidenceRefs.length===0)
        return record('confirm-inspect',blocked('GUIDED_STEP_ORDER_INVALID'));
      state.previousOutcome=guidedOutcome('inspect-evidence','USER_SUBMIT',state.evidenceRefs);
      recomputeDecision();
      return record('confirm-inspect',{status:'OK',decision:state.decision});
    },
    'draft-product'(sourceSha,{goal,nonGoals=[],l2Decision}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      const resolvedGoal=typeof goal==='string'&&goal.trim()?goal.trim():state.goals[0];
      if(!resolvedGoal||resolvedGoal.length>256)
        return record('draft-product',blocked('DRAFT_GOAL_REQUIRED'));
      if(!Array.isArray(nonGoals)||nonGoals.length>8||
        nonGoals.some(item=>typeof item!=='string'||item.length===0||item.length>256))
        return record('draft-product',blocked('DRAFT_NON_GOALS_INVALID'));
      if(l2Decision!==undefined&&!L2_CHOICES.includes(l2Decision))
        return record('draft-product',blocked('L2_DECISION_INVALID'));
      const tasks=journeySnapshot?.().tasks??[];
      const dag=dagLanes(tasks);
      state.draft={author,at:new Date().toISOString(),
        product:{goal:resolvedGoal,nonGoals:[...nonGoals]},
        l2:{decision:l2Decision??null,choices:[...L2_CHOICES]},
        dag,provenance:[...state.evidenceRefs],sourceSha:currentSha()};
      state.draft.digest=digestOf({product:state.draft.product,l2:state.draft.l2,
        provenance:state.draft.provenance,sourceSha:state.draft.sourceSha});
      if(state.review){state.review=null;
        state.reviewBlock={state:'BLOCKED',reason:'DRAFT_CHANGED_AFTER_REVIEW'};}
      state.frozen=null;
      return record('draft-product',{status:'OK',digest:state.draft.digest,
        conflictCount:dag.conflicts.length,blockedCount:dag.blockedCount,
        cycleCount:dag.cycleCount});
    },
    'review-draft'(sourceSha,{reviewer}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(!state.draft)return record('review-draft',blocked('NO_DRAFT_TO_REVIEW'));
      if(typeof reviewer!=='string'||!IDENTITY.test(reviewer))
        return record('review-draft',blocked('REVIEWER_IDENTITY_INVALID'));
      if(reviewer===state.draft.author)
        return record('review-draft',blocked('REVIEW_NOT_INDEPENDENT'));
      state.reviewBlock=null;
      state.review={reviewer,at:new Date().toISOString(),digest:state.draft.digest,
        sourceSha:currentSha(),state:'REVIEWED'};
      return record('review-draft',{status:'OK',reviewer,digest:state.review.digest,
        independence:'REVIEWER_DIFFERS_FROM_AUTHOR'});
    },
    async 'freeze-draft'(sourceSha){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(!state.draft)return record('freeze-draft',blocked('NO_DRAFT_TO_FREEZE'));
      if(!state.review||state.review.digest!==state.draft.digest)
        return record('freeze-draft',blocked(state.review?'REVIEW_STALE':'REVIEW_REQUIRED'));
      if(state.review.sourceSha!==currentSha())
        return record('freeze-draft',blocked('REVIEW_SOURCE_STALE'));
      if(state.decision?.status!=='STEP'||state.decision.step?.stepId!=='summary')
        return record('freeze-draft',blocked('GUIDED_SUMMARY_NOT_READY'));
      const receiptRef=`receipt:sha256:${state.draft.digest}`;
      state.previousOutcome=guidedOutcome('summary','EFFECT_RECEIPT',state.evidenceRefs,
        {receiptRef});
      recomputeDecision();
      state.frozen={state:'FROZEN',digest:state.draft.digest,receiptRef,
        reviewer:state.review.reviewer,author:state.draft.author,
        frozenAt:new Date().toISOString(),sourceSha:currentSha(),
        provenance:[...state.evidenceRefs],
        authority:'WORKBENCH_INDEPENDENT_REVIEW; native GitHub freeze authority stays Host-owned (NOT_RUN)'};
      return record('freeze-draft',{status:'OK',receiptRef,reviewer:state.review.reviewer,
        githubEffect:'NOT_RUN'});
    },
    async 'admit-project-intake'(sourceSha,{projectTitle}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(!state.frozen||state.frozen.digest===null)
        return record('admit-project-intake',blocked('PRODUCT_FREEZE_REQUIRED'));
      const input={issueRef:ISSUE_REF,issueNumber:ISSUE,sourceSha:currentSha(),
        productDigest:state.frozen.digest,projectTitle};
      const advisory=assessProjectIntakeAdmission(input);
      if(advisory.state!=='OBSERVED_INPUT_ONLY')
        return record('admit-project-intake',blocked(advisory.reason));
      // The workbench UI never dispatches a privileged effect by itself: the
      // only permitted path is the injected sealedDispatch seam (the selected
      // K consumer envelope). Without that seam the admit stays NOT_RUN.
      if(typeof sealedDispatch!=='function'){
        state.intake={state:'NOT_RUN',reason:'SEALED_K_DISPATCH_UNAVAILABLE',
          githubEffect:'NOT_RUN',project:null,sealedDispatch:'NOT_RUN'};
        return record('admit-project-intake',{status:'OK',...state.intake});
      }
      let observation=null;
      try{
        const envelope=await sealedDispatch({intentType:'admitProjectIntake',input});
        observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
          envelope.outcome.output:envelope;
      }catch(error){
        state.intake={state:'NOT_RUN',reason:'SEALED_K_DISPATCH_FAILED',
          githubEffect:'NOT_RUN',project:null,sealedDispatch:'SEAM_ERROR'};
        return record('admit-project-intake',{status:'OK',...state.intake});
      }
      const projection=projectProjectIntakeAdmission({input,observation,
        effectId:'intake-workbench-seam'});
      state.intake={state:projection.state,reason:projection.reason,
        githubEffect:projection.githubEffect??'NOT_RUN',
        project:projection.project??null,sealedDispatch:'VIA_SEAM'};
      return record('admit-project-intake',{status:'OK',...state.intake});
    },
    async 'refresh-source-facts'(sourceSha){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(typeof host?.readSourceFacts!=='function'){
        state.sourceFacts=null;
        return record('refresh-source-facts',{status:'NOT_RUN',
          reason:'HOST_GITHUB_READ_REQUIRED'});
      }
      const observed=await host.readSourceFacts({sources:PINNED_SOURCES,
        sourceSha:currentSha()});
      state.sourceFacts=observed&&typeof observed==='object'?observed:null;
      return record('refresh-source-facts',{status:state.sourceFacts?'OK':'NOT_RUN',
        reason:state.sourceFacts?'HOST_FACTS_OBSERVED':'HOST_GITHUB_READ_REQUIRED'});
    },
  };
  function snapshot(){
    const observedAt=new Date().toISOString();
    const tasks=journeySnapshot?.().tasks??[];
    return {status:'OBSERVED',project:OWNER,version:'v2',sourceSha:currentSha(),
      observedAsOf:observedAt,currentness:'OBSERVED_AT_QUERY_COMPLETION',
      guided:{journeyId:JOURNEY_ID,caseRef:caseRef(),
        goals:[...state.goals],evidence:[...state.evidenceRefs],
        decision:state.decision??recomputeDecision()},
      sources:sourceFacts(PINNED_SOURCES,state.sourceFacts),
      tasks,dag:dagLanes(tasks),
      draft:state.draft?{state:'CANDIDATE',author:state.draft.author,at:state.draft.at,
        digest:state.draft.digest,product:state.draft.product,l2:state.draft.l2,
        provenance:state.draft.provenance,note:'User/model drafted content; never an accepted freeze'}:null,
      review:state.review?{state:'REVIEWED',reviewer:state.review.reviewer,
        at:state.review.at,digest:state.review.digest,
        current:state.review.digest===state.draft?.digest}:null,
        ...(state.reviewBlock?{reviewBlock:state.reviewBlock}:{}),
      frozen:state.frozen?{...state.frozen}:null,
      intake:state.intake?{...state.intake}:{state:'NOT_RUN',
        reason:'SEALED_K_DISPATCH_REQUIRED',githubEffect:'NOT_RUN',project:null,
        sealedDispatch:'NOT_RUN'},
      constraints:{writeSet:[...WRITE_SET],writeSetDigest:digestOf(WRITE_SET),
        pinnedClaim:'kaicreator-mm/tally#210:builder:__default__',
        pinnedDispatch:'dispatch:210:v2:20261031:1',
        sourceSha:currentSha(),l2Choices:[...L2_CHOICES]},
      authorized:false,admitted:false,taskPass:false,releasePass:false};
  }
  async function dispatch(sourceSha,kind,args){
    const action=actions[kind];
    if(!action)return {status:'DENIED',reason:'UNKNOWN_ACTION'};
    if(stale(sourceSha))return {status:'STALE_SOURCE'};
    return action(sourceSha,args??{});
  }
  return Object.freeze({snapshot,dispatch,history:()=>[...state.history]});
}
export const __test={WRITE_SET,PINNED_SOURCES,digestOf,dagLanes};
