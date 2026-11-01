/** J08b/J08c maintenance workbench controller. Drafts stay candidates; the Host owns effects.
 *
 * The ordinary user reports a defect against an immutable historical release,
 * an independent reviewer reviews the next-version candidate and the only
 * privileged effect (the authorized native Issue publication) flows through
 * the injected sealedDispatch seam — selected B -> one K -> Host/GitHub C.
 * Task Learning uses existing public ADS reports with exact provenance, an
 * explicit use/reject decision and a later independent outcome; quality gain
 * is never inferred and Agents are never ranked. Missing task-meta stays
 * UNKNOWN and nonblocking. This module never touches a provider credential.
 */
import {projectMaintenanceIntake} from
  '../../packages/tally-engineering-business/src/maintenance-intake.mjs';
import {assessMaintenanceIssuePublication,projectMaintenanceIssuePublication,
  assessTaskLearningOutcome,projectTaskLearningOutcome}
  from '../../packages/tally-engineering-business/src/index.mjs';
import {maintenanceCandidateDigest,validateMaintenanceCandidate}
  from '../../experiments/v2/maintenance/host-maintenance.mjs';

const SHA=/^[0-9a-f]{40}$/,DIGEST=/^[0-9a-f]{64}$/;
const OWNER='kaicreator-mm',REPO='tally',ISSUE=219;
const ISSUE_REF=`https://github.com/${OWNER}/${REPO}/issues/${ISSUE}`;
const JOURNEY_ID='tally-v2-maintenance-j08';
const REPOSITORY=`${OWNER}/${REPO}`;
const VERSION=/^v?[0-9]+\.[0-9]+(?:\.[0-9]+)?$/;
const IDENTITY=/^[A-Za-z0-9][A-Za-z0-9 .:_'@-]{0,127}$/;
const ISSUE_NUMBER=/^[1-9][0-9]{0,5}$/;
const SUMMARY=/^\S(?:[\s\S]{0,510}\S)?$/;
const LEARNING_NOTE=/^\S(?:[\s\S]{0,510}\S)?$/;
// Exact frozen write set pinned by the accepted Claim on issue #219 (14 paths).
const WRITE_SET=Object.freeze([
  'packages/tally-github-c/src/maintenance-effects.mjs',
  'packages/tally-github-c/test/maintenance-effects.test.mjs',
  'experiments/v2/maintenance/host-maintenance.mjs',
  'experiments/v2/maintenance/host-maintenance.test.mjs',
  'web/v2/maintenance-controller.mjs','web/v2/maintenance-controller.test.mjs',
  'web/v2/journey-server.mjs','web/v2/journey-view.html',
  '.woodpecker/v2-alm-workspace.yaml',
  'packages/tally-engineering-business/src/index.mjs',
  'packages/tally-engineering-business/test/business-policy.test.mjs',
  'experiments/v2/selected-business-build/producer.mjs',
  'experiments/v2/selected-business-build/bundle-local-codex.mjs',
  'experiments/v2/selected-business-build/local-codex-shim.mjs']);
const LEARNING_DECISIONS=Object.freeze(['ADOPTED','REJECTED']);
const escape=value=>String(value??'').replace(/[&<>"']/g,ch=>({
  '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const badge=value=>`<span class="pill">${escape(value)}</span>`;
const issueHref=value=>/^https:\/\/github\.com\/[\w.-]+\/[\w.-]+\/issues\/[1-9][0-9]*$/.test(value??'')?
  value:'#';
const blocked=reason=>({status:'BLOCKED',reason});

export function createMaintenanceController(config,{host,sealedDispatch,now}={}){
  if(!SHA.test(config?.currentSourceSha??''))throw new TypeError('CURRENT_SOURCE_SHA_REQUIRED');
  if(config.author!==undefined&&(typeof config.author!=='string'||!IDENTITY.test(config.author)))
    throw new TypeError('AUTHOR_IDENTITY_INVALID');
  const author=config.author??'workbench:operator';
  const currentSha=()=>config.currentSourceSha;
  const clock=now??(()=>new Date().toISOString());
  const state={candidateInput:null,candidate:null,digest:null,review:null,
    learning:[],learningRecords:new Map(),publication:null,history:[]};
  const record=(actionKind,result)=>{state.history.push({action:actionKind,
    at:clock(),...result});return result;};
  const stale=sourceSha=>SHA.test(sourceSha??'')&&sourceSha!==currentSha();
  /** Advisory J08a projection of the drafted defect candidate. */
  function projectCandidate(input,at){
    const priorRelease=input.releaseVersion?{
      version:input.releaseVersion,commitSha:input.releaseCommitSha,
      evidenceRef:`https://github.com/${REPOSITORY}/releases/tag/${input.releaseVersion}`}:undefined;
    return projectMaintenanceIntake({repository:REPOSITORY,
      observedAsOf:at,
      priorRelease,
      defects:[{issueRef:input.defectIssueRef,requirementRef:input.requirementRef,
        subjectSha:input.releaseCommitSha,observedAt:at,summary:input.summary}],
      learning:[]});
  }
  function buildCandidateInput(args){
    const {summary,defectIssue,requirementIssue,releaseVersion,releaseCommitSha}=args??{};
    if(typeof summary!=='string'||!SUMMARY.test(summary))
      return {error:'DEFECT_SUMMARY_REQUIRED'};
    if(!ISSUE_NUMBER.test(requirementIssue??''))
      return {error:'REQUIREMENT_ISSUE_REQUIRED'};
    if(defectIssue!==undefined&&!ISSUE_NUMBER.test(defectIssue))
      return {error:'DEFECT_ISSUE_INVALID'};
    // A defect report anchors to the immutable historical release identity.
    if(!VERSION.test(releaseVersion??'')||!SHA.test(releaseCommitSha??''))
      return {error:'PRIOR_RELEASE_IDENTITY_REQUIRED'};
    const input={summary:summary.trim(),
      defectIssueRef:issueHref(`${ISSUE_REF.replace(/[0-9]+$/,'')}${defectIssue??ISSUE}`),
      requirementRef:issueHref(`${ISSUE_REF.replace(/[0-9]+$/,'')}${requirementIssue}`),
      releaseVersion,releaseCommitSha};
    const at=clock();
    const projection=projectCandidate(input,at);
    if(projection.status!=='CANDIDATES_ONLY')return {error:'MAINTENANCE_INPUT_UNVERIFIED'};
    return {input,projection,at};
  }
  const actions={
    'report-defect'(sourceSha,args){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      const built=buildCandidateInput(args);
      if(built.error)return record('report-defect',blocked(built.error));
      state.candidateInput=built.input;
      state.candidate=built.projection.candidates[0]??null;
      state.digest=maintenanceCandidateDigest({
        sourceIssueRef:built.input.defectIssueRef,
        requirementRef:built.input.requirementRef,
        subjectSha:built.input.releaseCommitSha,
        summary:built.input.summary,
        ...(built.input.releaseVersion?{priorRelease:{
          version:built.input.releaseVersion,
          commitSha:built.input.releaseCommitSha,
          evidenceRef:`https://github.com/${REPOSITORY}/releases/tag/${built.input.releaseVersion}`}}:
          {priorRelease:null})});
      // A new draft invalidates a stale review and any publication state.
      if(state.review&&state.review.digest!==state.digest)state.review=null;
      state.publication=null;
      return record('report-defect',{status:'OK',digest:state.digest,
        priorRelease:state.candidate?.priorReleaseEvidence??'UNKNOWN',
        historicalReleaseMutation:false});
    },
    'review-candidate'(sourceSha,{reviewer}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(!state.candidate||!state.digest)
        return record('review-candidate',blocked('NO_CANDIDATE_TO_REVIEW'));
      if(typeof reviewer!=='string'||!IDENTITY.test(reviewer))
        return record('review-candidate',blocked('REVIEWER_IDENTITY_INVALID'));
      if(reviewer===author)
        return record('review-candidate',blocked('REVIEW_NOT_INDEPENDENT'));
      state.review={reviewer,at:clock(),digest:state.digest,
        sourceSha:currentSha(),state:'REVIEWED',
        independence:'REVIEWER_DIFFERS_FROM_AUTHOR'};
      return record('review-candidate',{status:'OK',reviewer,digest:state.digest,
        note:'Host CAS freeze of the reviewed candidate stays Host-owned (NOT_RUN here)'});
    },
    async 'load-learning'(sourceSha){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(typeof host?.readTaskLearning!=='function'){
        state.learning=[];
        return record('load-learning',{status:'NOT_RUN',
          reason:'HOST_GITHUB_READ_REQUIRED'});
      }
      const observed=await host.readTaskLearning({sourceSha:currentSha()});
      state.learning=Array.isArray(observed?.suggestions)?observed.suggestions:[];
      return record('load-learning',{status:observed?'OK':'NOT_RUN',
        reason:observed?'PUBLIC_ADS_REPORTS_OBSERVED':'HOST_GITHUB_READ_REQUIRED',
        suggestions:state.learning.length});
    },
    async 'record-learning'(sourceSha,{learningId,decision,outcome}={}){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      const normalizedInput={issueRef:ISSUE_REF,issueNumber:ISSUE,
        sourceSha:currentSha(),
        learningId:typeof learningId==='string'?learningId:'',decision,
        ...(outcome!==undefined?{outcome}:{})};
      const advisory=assessTaskLearningOutcome(normalizedInput);
      if(advisory.state!=='OBSERVED_INPUT_ONLY')
        return record('record-learning',blocked(advisory.reason));
      if(typeof sealedDispatch!=='function'){
        state.learningRecords.set(learningId,{state:'NOT_RUN',
          reason:'SEALED_K_DISPATCH_UNAVAILABLE',githubEffect:'NOT_RUN'});
        return record('record-learning',{status:'OK',learningId,
          ...state.learningRecords.get(learningId)});
      }
      // The K event wire carries the outcome as always-present scalars
      // (null in the decision phase); the sealed boundary reassembles it.
      const wireInput={issueRef:ISSUE_REF,issueNumber:ISSUE,sourceSha:currentSha(),
        learningId,decision,
        outcomeAttested:outcome!==undefined?outcome.attested:null,
        outcomeObservedAt:outcome!==undefined?outcome.observedAt:null,
        outcomeNote:outcome!==undefined?outcome.note:null};
      let observation=null;
      try{
        const envelope=await sealedDispatch({intentType:'recordTaskLearningOutcome',
          input:wireInput});
        observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
          envelope.outcome.output:envelope;
      }catch{
        state.learningRecords.set(learningId,{state:'NOT_RUN',
          reason:'SEALED_K_DISPATCH_FAILED',githubEffect:'NOT_RUN'});
        return record('record-learning',{status:'OK',learningId,
          ...state.learningRecords.get(learningId)});
      }
      const projection=projectTaskLearningOutcome({input:normalizedInput,
        observation,effectId:'maintenance-workbench-seam'});
      state.learningRecords.set(learningId,{
        state:projection.state,reason:projection.reason,
        decision:projection.decision??decision,
        measuredEffect:projection.measuredEffect??'NOT_MEASURED',
        outcome:projection.outcome??null,githubEffect:projection.githubEffect??'NOT_RUN',
        noAgentRanking:true,qualityGainClaimed:false});
      return record('record-learning',{status:'OK',learningId,
        ...state.learningRecords.get(learningId)});
    },
    async 'publish-maintenance-issue'(sourceSha){
      if(stale(sourceSha))return{status:'STALE_SOURCE'};
      if(!state.candidate||!state.digest)
        return record('publish-maintenance-issue',blocked('NO_CANDIDATE_TO_PUBLISH'));
      if(!state.review||state.review.digest!==state.digest)
        return record('publish-maintenance-issue',
          blocked(state.review?'REVIEW_STALE':'REVIEW_REQUIRED'));
      if(state.review.sourceSha!==currentSha())
        return record('publish-maintenance-issue',blocked('REVIEW_SOURCE_STALE'));
      const input={issueRef:ISSUE_REF,issueNumber:ISSUE,sourceSha:currentSha(),
        candidateDigest:state.digest};
      const advisory=assessMaintenanceIssuePublication(input);
      if(advisory.state!=='OBSERVED_INPUT_ONLY')
        return record('publish-maintenance-issue',blocked(advisory.reason));
      // The workbench UI never dispatches a privileged effect by itself: the
      // only permitted path is the injected sealedDispatch seam (the selected
      // K consumer envelope). Without that seam the publish stays NOT_RUN.
      if(typeof sealedDispatch!=='function'){
        state.publication={state:'NOT_RUN',reason:'SEALED_K_DISPATCH_UNAVAILABLE',
          githubEffect:'NOT_RUN',issue:null,sealedDispatch:'NOT_RUN'};
        return record('publish-maintenance-issue',{status:'OK',...state.publication});
      }
      let observation=null;
      try{
        const envelope=await sealedDispatch({intentType:'publishMaintenanceIssue',input});
        observation=envelope?.phase==='terminal'&&envelope.outcome?.ok===true?
          envelope.outcome.output:envelope;
      }catch{
        state.publication={state:'NOT_RUN',reason:'SEALED_K_DISPATCH_FAILED',
          githubEffect:'NOT_RUN',issue:null,sealedDispatch:'SEAM_ERROR'};
        return record('publish-maintenance-issue',{status:'OK',...state.publication});
      }
      const projection=projectMaintenanceIssuePublication({input,observation,
        effectId:'maintenance-workbench-seam'});
      state.publication={state:projection.state,reason:projection.reason,
        githubEffect:projection.githubEffect??'NOT_RUN',
        issue:projection.issue??null,duplicate:projection.duplicate===true,
        alreadyPresent:projection.alreadyPresent===true,
        historicalReleaseMutation:false,sealedDispatch:'VIA_SEAM'};
      return record('publish-maintenance-issue',{status:'OK',...state.publication});
    },
  };
  function snapshot(){
    const observedAt=clock();
    const priorRelease=state.candidateInput?.releaseVersion?
      {version:state.candidateInput.releaseVersion,
        commitSha:state.candidateInput.releaseCommitSha,
        evidenceRef:`https://github.com/${REPOSITORY}/releases/tag/${state.candidateInput.releaseVersion}`,
        immutable:true}:null;
    const learning=state.learning.map(row=>{
      const recordState=state.learningRecords.get(row.learningId);
      return {...row,decision:recordState?.decision??null,
        measuredEffect:recordState?.measuredEffect??row.measuredEffect??'NOT_MEASURED',
        outcome:recordState?.outcome??null,recordState:recordState?.state??'NOT_RECORDED',
        recordReason:recordState?.reason??null,
        githubEffect:recordState?.githubEffect??'NOT_RUN',
        noAgentRanking:true,qualityGainClaimed:false};
    });
    const digestValid=DIGEST.test(state.digest??'');
    return {status:'OBSERVED',project:OWNER,version:'v2',journeyId:JOURNEY_ID,
      sourceSha:currentSha(),observedAsOf:observedAt,
      currentness:'OBSERVED_AT_QUERY_COMPLETION',
      priorRelease:priorRelease??'UNKNOWN',
      candidate:state.candidate?{state:state.review?'REVIEWED':'DRAFT',
        digest:digestValid?state.digest:null,
        sourceIssueRef:state.candidate.sourceIssueRef,
        requirementRef:state.candidate.requirementRef,
        summary:state.candidate.summary,
        priorRelease:state.candidate.priorRelease,
        priorReleaseEvidence:state.candidate.priorReleaseEvidence,
        nextAction:state.candidate.nextAction,
        historicalReleaseMutation:false}:null,
      review:state.review?{state:'REVIEWED',reviewer:state.review.reviewer,
        at:state.review.at,digest:state.review.digest,
        current:state.review.digest===state.digest,
        independence:state.review.independence}:null,
      publication:state.publication?{...state.publication}:{state:'NOT_RUN',
        reason:'SEALED_K_DISPATCH_REQUIRED',githubEffect:'NOT_RUN',issue:null,
        sealedDispatch:'NOT_RUN'},
      learning,
      constraints:{writeSet:[...WRITE_SET],
        writeSetCount:WRITE_SET.length,
        pinnedClaim:`${REPOSITORY}#${ISSUE}:builder:__default__`,
        pinnedDispatch:'dispatch:219:maintenance:20261031:1',
        sourceSha:currentSha(),
        historicalReleaseMutation:false,nativeWriteAuthorized:false},
      authorized:false,admitted:false,taskPass:false,releasePass:false,
      historicalReleaseMutation:false,nativeWriteAuthorized:false};
  }
  async function dispatch(sourceSha,kind,args){
    const action=actions[kind];
    if(!action)return {status:'DENIED',reason:'UNKNOWN_ACTION'};
    if(stale(sourceSha))return {status:'STALE_SOURCE'};
    return action(sourceSha,args??{});
  }
  return Object.freeze({snapshot,dispatch,history:()=>[...state.history]});
}

/** Browser-safe HTML rendering of one maintenance snapshot (no node APIs). */
export function renderMaintenanceWorkspace(snapshot,nextResult){
  if(!snapshot||snapshot.status!=='OBSERVED')
    return '<section class="error-panel" role="alert"><h2>UNKNOWN</h2><p>Maintenance snapshot is not verified. No form or action is available.</p></section>';
  const mode=nextResult?.mode==='write'?'write':'read';
  const constraints=snapshot.constraints??{};
  const candidate=snapshot.candidate;
  const prior=snapshot.priorRelease;
  const candidatePanel=`<section aria-labelledby="maint-candidate"><div class="section-head">
    <h2 id="maint-candidate">Defect candidate against an immutable release (J08b)</h2>
    <span class="quiet">The prior release is carried as a reported identity only — never rewritten.</span></div>
    <p>Prior release ${prior&&prior.immutable?
      `${badge('IMMUTABLE_IDENTITY')} <code>${escape(prior.version)}</code> at <code>${escape(String(prior.commitSha).slice(0,12))}…</code>`:
      badge('UNKNOWN')}</p>
    ${candidate?`<p>Status ${badge(candidate.state)} · digest <code>${escape(String(candidate.digest).slice(0,16))}…</code></p>
    <p><strong>Summary:</strong> ${escape(candidate.summary)}</p>
    <p class="quiet">source <a href="${issueHref(candidate.sourceIssueRef)}">${escape(candidate.sourceIssueRef)}</a> · requirement <a href="${issueHref(candidate.requirementRef)}">${escape(candidate.requirementRef)}</a> · next: ${escape(candidate.nextAction)}</p>`:
    `<p class="empty" role="status">No defect candidate reported yet.</p>`}
    ${snapshot.review?`<p>Independent review ${badge('REVIEWED')} by <code>${escape(snapshot.review.reviewer)}</code> · ${snapshot.review.current?'current':'STALE'}</p>`:
      `<p class="quiet">Independent review: ${badge('NOT_RUN')} — a model or author proposal never becomes an authorized candidate.</p>`}
    ${mode==='write'?`<form class="intake-form" aria-label="Report defect against a prior release">
      <div class="field"><label for="maint-summary">Defect summary</label>
        <input id="maint-summary" name="summary" type="text" maxlength="512" required></div>
      <div class="field"><label for="maint-requirement">Requirement issue number</label>
        <input id="maint-requirement" name="requirementIssue" type="text" inputmode="numeric" pattern="[1-9][0-9]{0,5}" required></div>
      <div class="field"><label for="maint-defect-issue">Source issue number (defaults to #219)</label>
        <input id="maint-defect-issue" name="defectIssue" type="text" inputmode="numeric" pattern="[1-9][0-9]{0,5}"></div>
      <div class="field"><label for="maint-release">Prior release tag (immutable, e.g. v1.5.5)</label>
        <input id="maint-release" name="releaseVersion" type="text" maxlength="32"></div>
      <div class="field"><label for="maint-release-sha">Prior release commit SHA</label>
        <input id="maint-release-sha" name="releaseCommitSha" type="text" maxlength="40"></div>
      <button type="button" data-action="report-defect">${candidate?'Update candidate (invalidates review)':'Report defect candidate'}</button>
    </form>
    <form class="intake-form" aria-label="Independent candidate review">
      <div class="field"><label for="maint-reviewer">Reviewer identity (must differ from the author)</label>
        <input id="maint-reviewer" name="reviewer" type="text" maxlength="128" required></div>
      <button type="button" data-action="review-candidate">Record independent review</button>
    </form>`:''}</section>`;
  const publication=snapshot.publication??{};
  const publishPanel=`<section aria-labelledby="maint-publish"><div class="section-head">
    <h2 id="maint-publish">Authorized native Issue effect (J08b)</h2>
    <span class="quiet">Selected B → one K → Host/GitHub C. Accepted Claim, exact source, frozen write set, independent source review, canonical readback.</span></div>
    <p>Effect state ${badge(publication.state??'NOT_RUN')} · GitHub effect ${badge(publication.githubEffect??'NOT_RUN')} · sealed dispatch ${escape(publication.sealedDispatch??'NOT_RUN')}</p>
    <p class="quiet">${escape(publication.reason??'SEALED_K_DISPATCH_REQUIRED')}${publication.issue?
      ` · published <a href="${issueHref(publication.issue.url)}" target="_blank" rel="noreferrer">#${escape(publication.issue.number)}</a> ${escape(publication.issue.title)}`:''}</p>
    <p class="quiet">Lost acknowledgments stay UNKNOWN and reconcile without duplicates. No second scheduler, Project model or GitHub writer exists.</p>
    ${mode==='write'?`<form class="intake-form" aria-label="Publish the authorized maintenance issue">
      <button type="button" data-action="publish-maintenance-issue">Publish authorized Issue through Host</button>
    </form>`:''}</section>`;
  const learningRows=(snapshot.learning??[]).map(row=>`<tr>
    <td data-label="Learning"><code>${escape(String(row.learningId).slice(0,16))}…</code></td>
    <td data-label="Qualification">${badge(row.qualification)}</td>
    <td data-label="Provenance"><a href="${issueHref(row.provenance?.publicationRef??'#')}">${escape(row.provenance?.reportId??'UNKNOWN')}</a> <small>${escape(row.provenance?.kind??'')} · rev ${escape(row.provenance?.revision??'UNKNOWN')} · ${escape(row.provenance?.epistemic??'UNKNOWN')}</small></td>
    <td data-label="Use / reject">${badge(row.decision??'NOT_RECORDED')} <small>${escape(row.recordState==='NOT_RUN'?(row.recordReason??'NOT_RUN'):row.recordState)}</small></td>
    <td data-label="Independent outcome">${badge(row.outcome?.measuredEffect??row.measuredEffect??'NOT_MEASURED')} <small>${escape(row.outcome?.observedAt??'no later outcome recorded')}</small></td>
  </tr>`).join('');
  const learningPanel=`<section aria-labelledby="maint-learning"><div class="section-head">
    <h2 id="maint-learning">Measured Task Learning (J08c)</h2>
    <span class="quiet">Existing public ADS reports with exact provenance; explicit use/reject and a later independent outcome. Never a quality-gain inference, never an Agent ranking.</span></div>
    ${(snapshot.learning??[]).length?`<div class="table-scroll"><table>
      <caption class="visually-hidden">Task Learning suggestions with provenance and decisions</caption>
      <thead><tr><th>Learning</th><th>Qualification</th><th>Provenance</th><th>Use / reject</th><th>Independent outcome</th></tr></thead>
      <tbody>${learningRows}</tbody></table></div>`:
      `<p class="empty" role="status">No Task Learning suggestions observed. Missing task-meta stays UNKNOWN and nonblocking.</p>`}
    ${mode==='write'?`<form class="intake-form" aria-label="Record Task Learning decision">
      <div class="field"><label for="maint-learning-id">Learning id</label>
        <input id="maint-learning-id" name="learningId" type="text" maxlength="64" required></div>
      <div class="field"><label for="maint-learning-decision">Decision</label>
        <select id="maint-learning-decision" name="decision">
          <option value="ADOPTED">ADOPTED</option><option value="REJECTED">REJECTED</option>
        </select></div>
      <button type="button" data-action="record-learning">Record use / reject through Host</button>
      <button type="button" data-action="load-learning">Load public ADS report suggestions</button>
    </form>`:''}</section>`;
  return `<section class="workspace intake-workspace" aria-label="Tally v2 maintenance workbench">
    <header class="workspace-header"><div><p class="eyebrow">TALLY v2 · GOVERNED MAINTENANCE · J08b/J08c</p>
      <h2>Defect against a historical release, next-version candidate, authorized native Issue</h2>
      <p class="subtitle">Guided forms only — no raw JSON. The candidate never rewrites history and never auto-creates a native Task.</p></div>
      <div class="source-meta">${badge('OBSERVED_AT_QUERY_COMPLETION')}
        <code>${escape(String(snapshot.sourceSha).slice(0,12))}</code><time>${escape(snapshot.observedAsOf)}</time></div></header>
    ${candidatePanel}
    ${publishPanel}
    ${learningPanel}
    <footer><span>authorized=${escape(snapshot.authorized)} · admitted=${escape(snapshot.admitted)} · taskPass=${escape(snapshot.taskPass)} · releasePass=${escape(snapshot.releasePass)} · historicalReleaseMutation=${escape(snapshot.historicalReleaseMutation)} · write set ${escape(constraints.writeSet?.length??0)} pinned paths · pinned dispatch <code>${escape(constraints.pinnedDispatch??'UNKNOWN')}</code></span></footer>
    <p class="visually-hidden" role="status" aria-live="polite" data-maintenance-result="${escape(nextResult?.result?.status??'')}"></p>
  </section>`;
}

export const __test={WRITE_SET,ISSUE_REF,REPOSITORY,escape,
  LEARNING_DECISIONS,validateMaintenanceCandidate};
