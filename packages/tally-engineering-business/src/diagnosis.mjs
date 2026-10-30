/** Bounded read-only Workfront diagnostics over #139 lineage and native observations. */
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\//;
const SHA=/^[0-9a-f]{40}$/;
const iso=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
const array=value=>Array.isArray(value)?value:[];
const rows=value=>array(value).filter(row=>row&&typeof row==='object'&&!Array.isArray(row));
const validRef=value=>REF.test(value??'');
function diagnosis(issue,classification,status,symptom,sourceRefs,owner,competing,minimal,authorization,mustNot,extra={}){
  return {type:'WorkfrontDiagnosis',issue,classification,status,symptom,sourceRefs:[...new Set(sourceRefs.filter(validRef))],
    likelyRootOwner:owner,competingExplanations:competing,minimalDecision:minimal,
    authorizationNeeded:authorization,mustNot,...extra};
}
/** observations are existing native DAG/selected-path/review/repair/cost/capability records. */
export function diagnoseWorkfront(timeline,observations={}){
  observations??={};
  if(!timeline||timeline.subjectStatus!=='OBSERVED'||!iso(observations.observedAsOf))return Object.freeze({
    status:'UNKNOWN',diagnoses:[],parallelCandidates:[],coverage:{comparison:'COUNTS_NOT_PERCENTAGES',
      agentActive:'UNKNOWN',taskPass:false,releasePass:false},advisoryOnly:true});
  const observationKeys=['dependencies','selectedPaths','repairs','reviews','costs','capabilities'];
  const malformed=!Array.isArray(timeline.items)||timeline.items.some(row=>!row||typeof row!=='object'||Array.isArray(row))||
    observationKeys.some(key=>observations[key]!==undefined&&
    (!Array.isArray(observations[key])||observations[key].some(row=>!row||typeof row!=='object'||Array.isArray(row))));
  if(malformed)return Object.freeze({status:'UNKNOWN',diagnoses:[],parallelCandidates:[],
    coverage:{comparison:'COUNTS_NOT_PERCENTAGES',agentActive:'UNKNOWN',taskPass:false,releasePass:false},advisoryOnly:true});
  const items=rows(timeline.items),dependencies=rows(observations.dependencies),paths=rows(observations.selectedPaths);
  const repairs=rows(observations.repairs),reviews=rows(observations.reviews),costs=rows(observations.costs);
  const capabilities=rows(observations.capabilities),diagnoses=[];
  for(const item of items){
    const issue=item.issue,refs=[item.issueRef,item.sourceRefs?.pullRequest,item.sourceRefs?.review];
    const dependency=dependencies.find(row=>row.issue===issue);
    if(dependency?.kind==='VERSION_SERIAL'&&dependency.proven!==true)
      diagnoses.push(diagnosis(issue,'POTENTIAL_FALSE_DEPENDENCY','HYPOTHESIS',
        'A generic old-Release-to-new-source serial edge is not proven to be a code/contract dependency.',
        [...refs,dependency.sourceRef],'Version Planner',['TRUE_CODE_CONTRACT_DEPENDENCY','FINAL_COMPATIBILITY_JOIN_ONLY'],
        'Compare the frozen native DAG and prove the minimal actual dependency before changing parallelism.',true,
        ['DO_NOT_REMOVE_NATIVE_DEPENDENCY','DO_NOT_CLAIM_PARALLEL_AUTHORIZATION'],{lawfulParallel:false}));
    const path=paths.find(row=>row.issue===issue);
    if(item.publicAcceptance==='PUBLIC_ACCEPTANCE_NOT_PROVEN'||path&&['FAIL','NOT_RUN'].includes(path.status)){
      const gapEvidence=validRef(path?.sourceRef)||validRef(item.sourceRefs?.acceptance)||
        validRef(item.sourceRefs?.pullRequest)||validRef(item.issueRef);
      diagnoses.push(diagnosis(issue,'INTEGRATION_COVERAGE_GAP',gapEvidence?'OBSERVED':'UNKNOWN',
        'Component/source progress does not prove the frozen public selected path.',
        [...refs,path?.sourceRef,item.sourceRefs?.acceptance],'Public Host Integration',
        ['PUBLIC_INTEGRATION_SEAM_FAILURE','ADDITIONAL_COMPONENT_TESTS_ONLY'],
        'Run the frozen public selected-path acceptance and join its real Host result.',true,
        ['DO_NOT_REPORT_100_PERCENT_DELIVERED','DO_NOT_REPLACE_WITH_MORE_LOCAL_TEST_COUNTS']));
    }
    if(path?.status==='NOT_RUN')
      diagnoses.push(diagnosis(issue,'SELECTED_PATH_NOT_INVOKED',validRef(path.sourceRef)?'OBSERVED':'UNKNOWN',
        'A loaded or merged feature has no observed invocation on the selected path.',[...refs,path.sourceRef],
        'Selected Runtime Owner',['FEATURE_NOT_WIRED','VERIFIER_NOT_RUN'],
        'Distinguish runtime wiring from missing verification with one selected invocation.',true,
        ['DO_NOT_INVOKE_OUTSIDE_K_ADMISSION','DO_NOT_TREAT_MODULE_LOAD_AS_BEHAVIOR_PASS']));
    for(const repair of repairs.filter(row=>row.issue===issue&&row.severity==='P0'&&row.status!=='RESOLVED'))
      diagnoses.push(diagnosis(issue,'REQUIRED_SECURITY_REPAIR',validRef(repair.sourceRef)?'OBSERVED':'UNKNOWN',
        'A real P0 correctness/security repair remains open.',[...refs,repair.sourceRef],'Security Repair Owner',
        ['REQUIRED_P0_FIX','FALSE_POSITIVE_REQUIRES_INDEPENDENT_REVIEW'],
        'Complete the required repair and independent exact-SHA retest.',true,
        ['DO_NOT_WAIVE_P0_FOR_COST','DO_NOT_REDUCE_REQUIRED_REVIEW','DO_NOT_REWRITE_OLD_RELEASE_FACTS']));
    const reviewRows=reviews.filter(row=>row.issue===issue);
    if(reviewRows.length>0){
      const effective=reviewRows.some(row=>row.verdict==='PASS'&&SHA.test(row.sourceSha??'')&&
        validRef(row.firstCounterexampleRef)&&validRef(row.sourceRef));
      diagnoses.push(diagnosis(issue,effective?'EFFECTIVE_REVIEW_CONVERGENCE':'NO_ROOT_PROGRESS',
        effective?'OBSERVED':'HYPOTHESIS',effective?'A counterexample-driven Review reached an exact-source PASS.':
          'Repeated Review/Repair lacks a new root counterexample or verified convergence.',
        [...refs,...reviewRows.flatMap(row=>[row.sourceRef,row.firstCounterexampleRef])],'Reviewer / Repair Owner',
        effective?['RISK_FINDING_WAS_PRODUCTIVE','REVIEW_COUNT_IS_NOT_WASTE']:['ROOT_CAUSE_NOT_FOUND','EVIDENCE_NOT_RETAINED'],
        effective?'Preserve the counterexample and reuse only the exact-SHA verdict.':'Reconcile the first falsifier and owning repair before another attempt.',
        !effective,['DO_NOT_COUNT_REVIEWS_AS_FEATURES','DO_NOT_INVENT_PRIOR_KNOWLEDGE'],
        effective?{}:{retrospective:'POSTMORTEM_ONLY'}));
    }
    for(const cost of costs.filter(row=>row.issue===issue)){
      const measured=cost.visibility==='PUBLIC'&&Number.isSafeInteger(cost.activeMs)&&cost.activeMs>=0&&validRef(cost.sourceRef);
      diagnoses.push(diagnosis(issue,measured?'ACTIVE_COST_MEASURED':'CALENDAR_SPAN_ONLY',measured?'OBSERVED':'UNKNOWN',
        measured?'Public active-span evidence is available.':'Only calendar span is public; Agent active time is not measured.',
        [...refs,cost.sourceRef],'Observability / Planner',['ACTIVE_WORK','WAITING_AND_CALENDAR_DELAY'],
        measured?'Compare like-for-like active spans without summing overlap.':'Keep active cost unmeasured and separate calendar waiting.',
        false,['DO_NOT_RANK_AGENT_EFFICIENCY_FROM_CALENDAR_SPAN','DO_NOT_SUM_PARALLEL_OVERLAP'],
        {calendarSpanMs:Number.isSafeInteger(cost.calendarSpanMs)?cost.calendarSpanMs:null,
          agentActive:measured?'MEASURED':'UNKNOWN',cost:measured?'MEASURED':'NOT_MEASURED'}));
    }
    const capability=capabilities.find(row=>row.issue===issue);
    if(capability?.lowRisk===true&&capability.scopedCi==='PASS')
      diagnoses.push(diagnosis(issue,'SCOPED_FAST_PATH_ELIGIBLE',validRef(capability.sourceRef)?'OBSERVED':'UNKNOWN',
        'A low-risk scoped change already has the required scoped CI PASS.',[...refs,capability.sourceRef],
        'Task Owner',['NORMAL_SCOPED_PATH_SUFFICIENT','HIDDEN_CONTEXT_UNKNOWN'],
        'Reuse the passed scoped gate; do not impose an architecture Review by diagnosis alone.',false,
        ['DO_NOT_FORCE_FULL_REVIEW_WITHOUT_A_NORMATIVE_REQUIREMENT','DO_NOT_EXTRAPOLATE_TO_FULL_RELEASE']));
  }
  const parallelCandidates=items.filter(item=>item.completion!=='REAL_COMPLETED'&&dependencies.some(row=>
    row.issue===item.issue&&array(row.blockedBy).length===0&&row.lawful===true&&validRef(row.sourceRef))&&
    capabilities.some(row=>row.issue===item.issue&&row.role==='builder'&&row.hostAvailable===true&&validRef(row.sourceRef)))
    .map(item=>({issue:item.issue,structuralCandidate:true,authorizationNeeded:true,
      sourceRefs:item.sourceRefs,mustNot:['DO_NOT_CREATE_CLAIM_OR_EFFECT_FROM_THIS_CANDIDATE']}));
  const count=classification=>diagnoses.filter(row=>row.classification===classification).length;
  return Object.freeze({status:'OBSERVED',observedAsOf:observations.observedAsOf,diagnoses,parallelCandidates,
    coverage:{capabilityCount:items.length,sourceMergedCount:timeline.coverage?.sourceMergedCount??null,
      realCompletedCount:timeline.coverage?.realCompletedCount??null,reviewEvidenceCount:reviews.length,
      integrationGapCount:count('INTEGRATION_COVERAGE_GAP'),requiredP0Count:count('REQUIRED_SECURITY_REPAIR'),
      agentActive:diagnoses.some(row=>row.agentActive==='MEASURED')?'MEASURED':'UNKNOWN',
      comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false},advisoryOnly:true});
}
