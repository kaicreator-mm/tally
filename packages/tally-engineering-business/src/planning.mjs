/** Evidence-linked next-version planning candidates. No native freeze/Claim/Effect authority. */
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\//;
const iso=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
const array=value=>Array.isArray(value)?value:[];
const validRef=value=>REF.test(value??'');
const intersection=(a,b)=>a.filter(value=>b.includes(value));
const unique=value=>[...new Set(value)];
const strings=value=>value===undefined||Array.isArray(value)&&value.every(row=>typeof row==='string'&&row.length>0);
function laneFrom(capability,diagnoses,planningAsOf){
  const refs=array(capability.sourceRefs).filter(validRef),issues=array(capability.evidenceIssues);
  const related=diagnoses.filter(row=>issues.includes(Number(row.issue?.split('#').at(-1))));
  const validations=unique([...array(capability.validation),
    ...related.some(row=>['INTEGRATION_COVERAGE_GAP','SELECTED_PATH_NOT_INVOKED'].includes(row.classification))?['PUBLIC_CONSUMER']:[]]);
  const p0=related.some(row=>row.classification==='REQUIRED_SECURITY_REPAIR');
  const falseSerial=related.some(row=>row.classification==='POTENTIAL_FALSE_DEPENDENCY');
  const evidence=array(capability.evidence).filter(row=>validRef(row?.sourceRef)&&iso(row?.observedAt));
  const retrospective=evidence.length>0&&evidence.every(row=>Date.parse(row.observedAt)>Date.parse(planningAsOf));
  const roleEvidence=capability.roleEvidence===true&&!retrospective&&refs.length>0;
  const environment=capability.environment==='LocalWindows'?'LOCAL_WINDOWS_HOST':
    capability.environment==='Cloud'?'CLOUD':'UNKNOWN';
  const noRoleRecommendation=!roleEvidence||!['LocalWindows','Cloud'].includes(capability.environment)||
    !['LocalWindows','Cloud'].includes(capability.hostRequired)||
    capability.hostRequired==='LocalWindows'&&capability.environment==='Cloud';
  return {type:'TaskLaneCandidate',status:'CANDIDATE',lane:capability.lane,userOutcome:capability.userOutcome,
    incumbentOwner:capability.owner,independentCapabilityLane:capability.lane,
    blockedBy:array(capability.blockedBy),nativeDependenciesChanged:false,
    potentialParallelScope:falseSerial?'REVIEW_NATIVE_DEPENDENCY_ONLY':'STRUCTURAL_REVIEW_REQUIRED',
    writeSet:array(capability.writeSet),requiredValidation:validations,
    requiredReview:p0?'EXACT_SHA_INDEPENDENT_SECURITY_REVIEW':capability.lowRisk===true&&refs.length>0?'SCOPED_REVIEW':'INDEPENDENT_REVIEW',
    requiredHost:capability.hostRequired??'UNKNOWN',earlyIntegrationOwner:validations.includes('PUBLIC_CONSUMER'),
    agentRole:noRoleRecommendation?'NO_RECOMMENDATION/MORE_EVIDENCE':capability.role,
    agentRoleEvidence:{environment,sourceRefs:refs,comparable:!noRoleRecommendation},
    mustRetainInvariant:p0,retrospective:retrospective?'RETROSPECTIVE_ONLY':'PLANNING_TIME_QUALIFIED',
    costRisk:{activeCost:'UNKNOWN',risk:p0?'P0_SECURITY_CRITICAL':capability.risk??'UNKNOWN'},
    reuseExistingCampaign:related.some(row=>row.classification==='EFFECTIVE_REVIEW_CONVERGENCE')&&capability.campaign===true,
    fastPath:capability.lowRisk===true&&capability.scopedCi==='PASS'&&refs.length>0&&!p0,
    unresolvedDecision:noRoleRecommendation||falseSerial?'HUMAN_DECISION_REQUIRED':null,
    sourceRefs:unique([...refs,...related.flatMap(row=>array(row.sourceRefs)).filter(validRef)]),
    mustNot:['DO_NOT_TREAT_CANDIDATE_AS_NATIVE_TASK','DO_NOT_DELETE_NATIVE_BLOCKED_BY',
      'DO_NOT_DISPATCH_AGENT_FROM_CANDIDATE',...p0?['DO_NOT_LOWER_SECURITY_INVARIANT']:[]]};
}
/** capabilities are L2/planning facts, not requested Task Claims. */
export function planNextVersion(input={}){
  const {goal,nonGoals=[],timeline,diagnoses,capabilities=[],planningAsOf}=input??{};
  const malformed=typeof goal!=='string'||goal.trim().length===0||!iso(planningAsOf)||
    timeline?.subjectStatus!=='OBSERVED'||diagnoses?.status!=='OBSERVED'||!Array.isArray(capabilities)||
    !Array.isArray(nonGoals)||!strings(nonGoals)||!Array.isArray(diagnoses.diagnoses)||
    diagnoses.diagnoses.some(row=>!row||typeof row!=='object'||Array.isArray(row)||
      typeof row.issue!=='string'||!/[#][1-9][0-9]*$/.test(row.issue)||
      !Number.isSafeInteger(Number(row.issue.split('#').at(-1)))||
      typeof row.classification!=='string'||!strings(row.sourceRefs)||!row.sourceRefs.every(validRef))||
    (diagnoses.rejectedOptions!==undefined&&(!Array.isArray(diagnoses.rejectedOptions)||
      diagnoses.rejectedOptions.some(row=>!row||typeof row!=='object'||Array.isArray(row))))||
    capabilities.some(row=>!row||typeof row!=='object'||Array.isArray(row)||
      typeof row.lane!=='string'||row.lane.length===0||typeof row.userOutcome!=='string'||
      typeof row.owner!=='string'||row.owner.length===0||
      (row.role!==undefined&&typeof row.role!=='string')||
      (row.hostRequired!==undefined&&typeof row.hostRequired!=='string')||
      (row.environment!==undefined&&typeof row.environment!=='string')||
      !strings(row.writeSet)||!strings(row.blockedBy)||!strings(row.validation)||!strings(row.sourceRefs)||
      (row.evidenceIssues!==undefined&&!Array.isArray(row.evidenceIssues))||
      !array(row.evidenceIssues).every(number=>Number.isSafeInteger(number)&&number>0)||
      (row.evidence!==undefined&&(!Array.isArray(row.evidence)||row.evidence.some(item=>
        !item||typeof item!=='object'||!validRef(item.sourceRef)||!iso(item.observedAt))))||
      (row.risk!==undefined&&typeof row.risk!=='string'));
  if(malformed)return Object.freeze({status:'UNKNOWN',product:null,l2Options:[],dag:{status:'UNKNOWN',lanes:[]},
    feedback:{ruleActive:false,skillActive:false,packageActive:false},coverage:{comparison:'COUNTS_NOT_PERCENTAGES',
      taskPass:false,releasePass:false,claimed:false},advisoryOnly:true});
  const lanes=capabilities.map(capability=>laneFrom(capability,array(diagnoses.diagnoses),planningAsOf));
  const conflicts=[];
  for(let left=0;left<lanes.length;left++)for(let right=left+1;right<lanes.length;right++){
    const shared=intersection(lanes[left].writeSet,lanes[right].writeSet);
    if(shared.length>0)conflicts.push({type:'WRITE_SET_CONFLICT',lanes:[lanes[left].lane,lanes[right].lane],shared,
      parallel:false,alternative:{decision:'SINGLE_GENERATED_WRITER_OR_SERIALIZE',humanDecisionRequired:true},
      mustNot:['DO_NOT_SHARE_GENERATED_DIST_OR_MANIFEST_WRITER']});
  }
  const sourceRefs=unique([...array(validRef(timeline.subject?.sourceRef)?[timeline.subject.sourceRef]:[]),
    ...lanes.flatMap(lane=>lane.sourceRefs)]);
  const options=lanes.filter(lane=>lane.potentialParallelScope==='REVIEW_NATIVE_DEPENDENCY_ONLY'||lane.unresolvedDecision)
    .map(lane=>({type:'L2DecisionOption',status:'CANDIDATE',lane:lane.lane,
      options:['KEEP_NATIVE_DEPENDENCY','PROVE_MINIMAL_DEPENDENCY_BEFORE_PARALLEL'],
      humanDecisionRequired:true,sourceRefs:lane.sourceRefs}));
  const rejected=array(diagnoses.rejectedOptions).map(row=>({...row,historicalFeedback:true,
    ruleActive:false,skillActive:false,packageActive:false}));
  return Object.freeze({status:'CANDIDATE',product:{type:'ProductR0Candidate',status:'CANDIDATE',goal,
    nonGoals:[...nonGoals],sourceRefs},l2Options:options,dag:{type:'TaskDAGCandidate',status:'CANDIDATE',lanes,conflicts},
    feedback:{rejectedOptions:rejected,ruleActive:false,skillActive:false,packageActive:false},
    coverage:{candidateCount:lanes.length,conflictCount:conflicts.length,
      noRecommendationCount:lanes.filter(lane=>lane.agentRole==='NO_RECOMMENDATION/MORE_EVIDENCE').length,
      retrospectiveCount:lanes.filter(lane=>lane.retrospective==='RETROSPECTIVE_ONLY').length,
      comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false,claimed:false},advisoryOnly:true});
}
