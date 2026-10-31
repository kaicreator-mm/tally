/** J01–J03 intake workbench view. Accessible forms only — users never paste raw JSON. */
const escape=value=>String(value??'').replace(/[&<>"']/g,ch=>({
  '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const sha=value=>/^[0-9a-f]{40}$/.test(value??'')?value.slice(0,12):'UNKNOWN';
const url=value=>/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/(?:issues|pull)\/[1-9][0-9]*$/.test(value??'')?value:'#';
const cls=value=>String(value??'UNKNOWN').toLowerCase().replace(/[^a-z0-9]+/g,'-');
const badge=value=>`<span class="pill ${cls(value)}">${escape(value)}</span>`;
const NOT_RUN='NOT_RUN';
function factBadge(fact){
  return `<td data-label="Observed fact">${badge(fact?.status??NOT_RUN)}
    <small>${escape(fact?.reason??'HOST_GITHUB_READ_REQUIRED')}</small></td>`;
}
function sourcesTable(sources){
  return `<div class="table-scroll"><table><caption class="visually-hidden">Real requirement sources and their observation status</caption>
    <thead><tr><th>Source</th><th>Pin</th><th>Observed fact</th></tr></thead><tbody>
    ${sources.map(source=>`<tr>
      <td data-label="Source"><a href="${url(source.sourceRef)}" target="_blank" rel="noreferrer">#${escape(source.issueNumber)}</a> ${escape(source.label)}</td>
      <td data-label="Pin"><small>${escape(source.pinnedBy)}</small></td>
      ${factBadge(source.fact)}</tr>`).join('')}</tbody></table></div>`;
}
function guidedForm(decision,mode){
  const step=decision?.step;
  if(decision?.status==='DONE')
    return `<p class="empty" role="status">Guided intake complete: outcome observed with a Host receipt.</p>`;
  if(!step)return `<p class="empty" role="status">${escape(decision?.reasonCode??'WAIT')} — ${escape(decision?.status??'WAIT')}</p>`;
  const fields=step.fields.map(field=>{
    if(field.type==='TEXT')
      return `<div class="field"><label for="intake-${escape(field.id)}">${escape(field.labelKey)}${field.required?' *':''}</label>
        <input id="intake-${escape(field.id)}" name="${escape(field.id)}" type="text" maxlength="256" ${field.required?'required':''}
          aria-describedby="intake-goal-hint"><p class="quiet" id="intake-goal-hint">Plain text goal. No JSON needed anywhere in this workspace.</p></div>`;
    if(field.type==='MULTI')
      return `<fieldset><legend>${escape(field.labelKey)}${field.required?' *':''}</legend>
        ${(field.options??[]).map((option,i)=>`<label class="option"><input type="checkbox" name="${escape(field.id)}" value="${escape(option.labelKey)}" ${option.eligible?'':'disabled'}> ${escape(option.labelKey)}</label>`).join('')}
        <p class="quiet">Evidence options are the pinned real sources listed above.</p></fieldset>`;
    return '';
  }).join('');
  const actionKind=step.stepId==='ask-goal'?'select-evidence':
    step.stepId==='inspect-evidence'?'confirm-inspect':'noop';
  const button=step.stepId==='ask-goal'
    ?`<button type="button" data-action="submit-goal">Save goal</button>
      <button type="button" data-action="select-evidence">Save selected evidence</button>`
    :step.stepId==='inspect-evidence'
      ?`<button type="button" data-action="confirm-inspect">Confirm evidence</button>`:'';
  return `<form class="intake-form" aria-label="${escape(step.titleKey)}">
    <input type="hidden" name="guided-step" value="${escape(step.stepId)}">
    ${fields}${mode==='write'?button:''}</form>`;
}
function draftPanel(draft,dag,l2Choices,mode){
  const lanes=dag?.lanes??[];
  const laneRows=lanes.map(lane=>`<tr>
    <td data-label="Lane"><code>${escape(lane.lane)}</code></td>
    <td data-label="Dependency">${badge(lane.dependencyState)}</td>
    <td data-label="Claim">${badge(lane.claimState)}</td>
    <td data-label="Attempt">${badge(lane.agentAttemptState)}</td>
    <td data-label="Write set">${escape(lane.writeSetCount)} pinned path(s)</td></tr>`).join('');
  const conflicts=(dag?.conflicts??[]).map(conflict=>`<li>${escape(conflict.lanes?.join(' ↔ '))}: ${escape(conflict.shared?.join(', '))}</li>`).join('');
  return `<section aria-labelledby="intake-draft"><div class="section-head">
    <h2 id="intake-draft">Draft Product / L2 / Task DAG</h2>
    <span class="quiet">${escape(draft?.state??'ABSENT')} — a draft is never an accepted Freeze.</span></div>
    ${draft?`<p>Status ${badge('CANDIDATE')} · author <code>${escape(draft.author)}</code> · digest <code>${escape(String(draft.digest).slice(0,16))}…</code></p>
    <p><strong>Goal:</strong> ${escape(draft.product.goal)}</p>
    ${draft.product.nonGoals.length?`<p><strong>Non-goals:</strong> ${escape(draft.product.nonGoals.join(' · '))}</p>`:''}
    <p><strong>L2 decision:</strong> ${escape(draft.l2.decision??'UNDECIDED')}</p>`:''}
    <div class="table-scroll"><table><caption class="visually-hidden">Task DAG lanes with native claim facts</caption>
      <thead><tr><th>Lane</th><th>Dependency</th><th>Claim</th><th>Attempt</th><th>Write set</th></tr></thead>
      <tbody>${laneRows}</tbody></table></div>
    <p class="quiet">Blocked native tasks: ${escape(dag?.blockedCount??0)} · write-set conflicts: ${escape(dag?.conflicts?.length??0)} · cycles: ${escape(dag?.cycleCount??0)} · physical Claim/Attempt/Review columns are native facts, not a static diagram.</p>
    ${conflicts?`<ul class="conflicts">${conflicts}</ul>`:''}
    ${mode==='write'?`<form class="intake-form" aria-label="Draft product">
      <div class="field"><label for="draft-goal">Product goal</label>
        <input id="draft-goal" name="goal" type="text" maxlength="256" value="${escape(draft?.product.goal??'')}"></div>
      <div class="field"><label for="draft-nongoals">Non-goals (one per line)</label>
        <textarea id="draft-nongoals" name="nonGoals" rows="3" maxlength="2048">${escape(draft?.product.nonGoals.join('\n')??'')}</textarea></div>
      <div class="field"><label for="draft-l2">L2 decision</label>
        <select id="draft-l2" name="l2Decision">
          <option value="">UNDECIDED</option>
          ${l2Choices.map(choice=>`<option value="${escape(choice)}" ${draft?.l2.decision===choice?'selected':''}>${escape(choice)}</option>`).join('')}
        </select></div>
      <button type="button" data-action="draft-product">${draft?'Update draft (invalidates review)':'Create draft'}</button>
    </form>`:''}</section>`;
}
function reviewPanel(snapshot,mode){
  const {draft,review,frozen}=snapshot;
  const write=mode==='write';
  return `<section aria-labelledby="intake-review"><div class="section-head">
    <h2 id="intake-review">Independent review → freeze</h2>
    <span class="quiet">A model's or author's proposal never becomes an accepted Freeze; a stale or malformed Review blocks.</span></div>
    <ol class="review-steps">
      <li>Draft: ${draft?badge('CANDIDATE'):badge('ABSENT')}</li>
      <li>Independent review: ${review?badge('REVIEWED')+` <small>by <code>${escape(review.reviewer)}</code> · digest ${review.current?'current':'STALE'}</small>`:badge(snapshot.reviewBlock?.state??NOT_RUN)+` <small>${escape(snapshot.reviewBlock?.reason??'reviewer identity required')}</small>`}</li>
      <li>Freeze: ${frozen?badge('FROZEN')+` <small><code>${escape(frozen.receiptRef.slice(0,26))}…</code></small>`:badge(NOT_RUN)+` <small>requires a current independent review</small>`}</li>
    </ol>
    ${frozen?`<p class="quiet">${escape(frozen.authority)}</p>`:''}
    ${write?`<form class="intake-form" aria-label="Review and freeze">
      <div class="field"><label for="review-reviewer">Reviewer identity (must differ from the author)</label>
        <input id="review-reviewer" name="reviewer" type="text" maxlength="128" required></div>
      <button type="button" data-action="review-draft">Record independent review</button>
      <button type="button" data-action="freeze-draft">Freeze reviewed draft</button>
    </form>`:''}</section>`;
}
function intakePanel(snapshot,mode){
  const intake=snapshot.intake??{};
  return `<section aria-labelledby="intake-effect"><div class="section-head">
    <h2 id="intake-effect">Governed project effect (J01b)</h2>
    <span class="quiet">Selected B → one K → Host/GitHub C with the accepted Claim, exact source and repo allowlist. No UI or Agent credential.</span></div>
    <p>Effect state ${badge(intake.state??NOT_RUN)} · GitHub effect ${badge(intake.githubEffect??NOT_RUN)}</p>
    <p class="quiet">${escape(intake.reason??'HOST_PROJECT_INTAKE_REQUIRED')} · sealed K dispatch: ${escape(intake.sealedDispatch??'NOT_RUN')}${intake.project?` · project <a href="${url(`https://github.com/kaicreator-mm/tally/projects/${intake.project.number}`)}" target="_blank" rel="noreferrer">#${escape(intake.project.number)}</a> ${escape(intake.project.title)}`:''}</p>
    ${mode==='write'?`<form class="intake-form" aria-label="Admit project intake">
      <div class="field"><label for="intake-project-title">Target project title</label>
        <input id="intake-project-title" name="projectTitle" type="text" maxlength="120" required></div>
      <button type="button" data-action="admit-project-intake">Admit project intake through Host</button>
      <button type="button" data-action="refresh-source-facts">Refresh GitHub source facts</button>
    </form>`:''}</section>`;
}
export function renderIntakeWorkspace(snapshot,nextResult){
  if(!snapshot||snapshot.status!=='OBSERVED')
    return '<section class="error-panel" role="alert"><h2>UNKNOWN</h2><p>Intake snapshot is not verified. No form or action is available.</p></section>';
  const mode=nextResult?.mode==='write'?'write':'read';
  const constraints=snapshot.constraints??{};
  return `<section class="workspace intake-workspace" aria-label="Tally v2 guided intake workbench">
    <header class="workspace-header"><div><p class="eyebrow">TALLY v2 · GUIDED INTAKE · J01–J03</p>
      <h2>Project intake, Product/L2 freeze and Task DAG</h2>
      <p class="subtitle">Guided forms only — no raw JSON. Drafts stay candidates until an independent review freezes them and the Host proves the native effect.</p></div>
      <div class="source-meta">${badge('OBSERVED_AT_QUERY_COMPLETION')}
        <code>${sha(snapshot.sourceSha)}</code><time>${escape(snapshot.observedAsOf)}</time></div></header>
    <section aria-labelledby="intake-sources"><div class="section-head">
      <h2 id="intake-sources">Real requirement sources (J01a)</h2>
      <span class="quiet">Read-only. Existing repo facts vs greenfield alternatives; live GitHub reads stay Host-owned.</span></div>
      ${sourcesTable(snapshot.sources??[])}</section>
    <section aria-labelledby="intake-guided"><div class="section-head">
      <h2 id="intake-guided">Guided intake</h2>
      <span class="quiet">Guided journey state: ${escape(snapshot.guided?.decision?.status??'UNKNOWN')} · ${escape(snapshot.guided?.decision?.reasonCode??'')}</span></div>
      ${snapshot.guided?.goals?.length?`<p role="status">Goal: <strong>${escape(snapshot.guided.goals[0])}</strong></p>`:''}
      ${snapshot.guided?.evidence?.length?`<p class="quiet">Evidence: ${escape(snapshot.guided.evidence.join(', '))}</p>`:''}
      ${guidedForm(snapshot.guided?.decision,mode)}</section>
    ${draftPanel(snapshot.draft,snapshot.dag,constraints.l2Choices??[],mode)}
    ${reviewPanel(snapshot,mode)}
    ${intakePanel(snapshot,mode)}
    <footer><span>authorized=${escape(snapshot.authorized)} · admitted=${escape(snapshot.admitted)} · taskPass=${escape(snapshot.taskPass)} · releasePass=${escape(snapshot.releasePass)} · write-set digest <code>${escape(String(constraints.writeSetDigest).slice(0,16))}…</code> (${escape(constraints.writeSet?.length??0)} paths)</span></footer>
    <p class="visually-hidden" role="status" aria-live="polite" data-intake-result="${escape(nextResult?.result?.status??'')}"></p>
  </section>`;
}
export const __test={escape,sha,url,badge};
