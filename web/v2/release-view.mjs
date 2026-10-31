/** J07c browser-safe, read-only Release panel. An advisory PASS is not a button permit. */
const esc=value=>String(value??'').replace(/[&<>"']/g,ch=>
  ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const sha=value=>/^[0-9a-f]{40}$/.test(value??'')?value:'UNKNOWN';
const gateList=Array.isArray;
const safeState=value=>['BLOCKED','UNKNOWN','NOT_RUN','STALE','PASS','FAIL'].includes(value)?value:'UNKNOWN';
/** No clickable release effects until the connected Host supplies an exact,
 * fresh, separately qualified legal action — this renderer does not decide it. */
export function renderReleasePanel(snapshot={}){
  const projection=snapshot?.readiness??{},gates=gateList(projection.gates)?projection.gates:[];
  const sourceSha=sha(snapshot?.sourceSha),targetSha=sha(snapshot?.targetSha);
  const status=safeState(projection.status);
  const rows=gates.map(g=>`<tr><th scope="row">${esc(g?.name)}</th><td>${esc(g?.status??'UNKNOWN')}</td><td>${esc(g?.reason??'NO_EVIDENCE')}</td></tr>`).join('');
  const unsupported=projection?.canMerge===true||projection?.canTag===true||
    projection?.canPublish===true||projection?.releasePass===true;
  const reason=unsupported?'ADVISORY_AUTHORITY_IGNORED':
    snapshot?.reason??projection.reason??'RELEASE_RQ_NOT_CONNECTED';
  return `<section id="v2-release" aria-labelledby="release-title">
    <header><h2 id="release-title">Release qualification</h2>
      <p role="status">${esc(status)} · ${esc(reason)}</p></header>
    <p>Version head: <code>${esc(sourceSha)}</code></p>
    <p>Target main: <code>${esc(targetSha)}</code></p>
    <table><caption>Reported evidence, not independent release authorization</caption>
      <thead><tr><th scope="col">Gate</th><th scope="col">Status</th><th scope="col">Reason</th></tr></thead>
      <tbody>${rows||'<tr><td colspan="3">Release evidence NOT_RUN</td></tr>'}</tbody></table>
    <p>Release actions require a fresh selected K/Host admission, independent exact-SHA RQ and Owner grant.</p>
    <button type="button" data-release-action="merge" disabled aria-disabled="true">Merge to main — locked</button>
    <button type="button" data-release-action="tag" disabled aria-disabled="true">Create release tag — locked</button>
    <button type="button" data-release-action="publish" disabled aria-disabled="true">Publish release — locked</button>
  </section>`;
}
