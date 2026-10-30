const escape=value=>String(value??'').replace(/[&<>"']/g,ch=>({
  '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const sha=value=>/^[0-9a-f]{40}$/.test(value??'')?value.slice(0,12):'UNKNOWN';
const url=value=>/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/.test(value??'')?value:'#';
const cls=value=>String(value??'UNKNOWN').toLowerCase().replace(/[^a-z0-9]+/g,'-');
const cell=value=>`<td data-label="${escape(value)}">${escape(value)}</td>`;
function taskRow(task,action){
  const ref=url(task.sourceRef);
  return `<tr class="task-row">
    <td data-label="Task"><a href="${ref}" target="_blank" rel="noreferrer">${escape(ref==='#'?'UNKNOWN':task.sourceRef)}</a><code>${sha(task.sourceSha)}</code></td>
    ${cell(task.dependencyState)}${cell(task.claimState)}${cell(task.agentAttemptState)}
    ${cell(task.reviewState)}${cell(task.ciState)}${cell(task.releaseState)}
    <td data-label="Next action"><span class="pill ${cls(action?.authorizationType)}">${escape(action?.authorizationType??'NO_VALID_ACTION')}</span><small>${escape(action?.reason??task.reason)}</small></td>
  </tr>`;
}
function actionCard(action){
  return `<article class="action-card ${cls(action.authorizationType)}">
    <div class="action-head"><span class="pill ${cls(action.authorizationType)}">${escape(action.authorizationType)}</span>
      <span class="authority">${escape(action.authorization)}</span></div>
    <p>${escape(action.reason)}</p>
    <dl><div><dt>Role</dt><dd>${escape(action.actorRole)}</dd></div>
      <div><dt>Capability</dt><dd>${escape(action.requiredCapability)}</dd></div>
      <div><dt>Approval needed</dt><dd>${action.approvalNeeded?'true':'false'}</dd></div>
      <div><dt>Cost/risk</dt><dd>${escape(action.costRisk)}</dd></div></dl>
  </article>`;
}
function inboxCard(item){
  return `<article class="inbox-card"><strong>${escape(item.kind)}</strong><p>${escape(item.reason)}</p>
    <p>Expected evidence: <code>${escape(item.expectedEvidence)}</code></p>
    <a href="${url(item.readback)}" target="_blank" rel="noreferrer">Read back ${escape(item.readback)}</a></article>`;
}
export function renderJourneyWorkspace(snapshot,nextActionResult){
  if(!snapshot||snapshot.status!=='OBSERVED')
    return '<section class="error-panel" role="alert"><h2>UNKNOWN</h2><p>Journey snapshot is not verified. No action is available.</p></section>';
  const tasks=Array.isArray(snapshot.tasks)?snapshot.tasks:[];
  const actions=Array.isArray(nextActionResult?.actions)?nextActionResult.actions:[];
  const actionByRef=new Map(actions.map(action=>[action.evidence?.sourceRef,action]));
  const inbox=Array.isArray(nextActionResult?.humanActionInbox)?nextActionResult.humanActionInbox:[];
  return `<section class="workspace" aria-label="Tally v2 read-only ALM workspace">
    <header class="workspace-header"><div><p class="eyebrow">TALLY v2 · READ-ONLY ALM WORKSPACE</p>
      <h1>${escape(snapshot.project)} <span>${escape(snapshot.version)}</span></h1>
      <p class="subtitle">Read-only illustrative fixture · Agent-native engineering facts and legal next actions. No write effects.</p></div>
      <div class="source-meta"><span class="pill observed">OBSERVED_AT_QUERY_COMPLETION</span>
        <code>${sha(snapshot.sourceSha)}</code><time>${escape(snapshot.observedAsOf)}</time></div></header>
    <section class="table-section" aria-labelledby="task-facts"><div class="section-head"><h2 id="task-facts">Task evidence</h2>
      <span class="quiet">Claim, execution, review, CI and release are independent layers.</span></div>
      <div class="table-scroll"><table><thead><tr><th>Task / source</th><th>Dependency</th><th>Claim</th><th>Attempt</th><th>Review</th><th>CI</th><th>Release</th><th>Next action</th></tr></thead>
        <tbody>${tasks.map(task=>taskRow(task,actionByRef.get(task.sourceRef))).join('')}</tbody></table></div></section>
    <section aria-labelledby="next-actions"><div class="section-head"><h2 id="next-actions">Next legal actions</h2>
      <span class="quiet">Advisory only; approval and Host/ADS authority remain explicit.</span></div>
      <div class="action-grid">${actions.map(actionCard).join('')}</div></section>
    <section aria-labelledby="human-inbox"><div class="section-head"><h2 id="human-inbox">Human-action inbox</h2>
      <span class="quiet">Only genuine Host/scope duties appear here.</span></div>
      ${inbox.length?`<div class="inbox-grid">${inbox.map(inboxCard).join('')}</div>`:'<p class="empty">No human action required. Existing Agent/CI paths remain available.</p>'}
    </section>
    <footer><span>authorized=${escape(snapshot.authorized)} · admitted=${escape(snapshot.admitted)} · taskPass=${escape(snapshot.taskPass)} · releasePass=${escape(snapshot.releasePass)}</span></footer>
  </section>`;
}
export const __test={escape,sha,url};
