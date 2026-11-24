import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

export const KNOWLEDGE_SHA = '874702df43a7563b624b270fb2389c368050408b';
const CATALOG_SHA256 = '595c11ae9a33644aefc89af99af9979385e734b9d675601f248afe35b07b30dc';
const sha = value => createHash('sha256').update(value).digest('hex');
const bytes = value => Buffer.byteLength(value, 'utf8');
const nonempty = value => typeof value === 'string' && value.trim().length > 0;
const hex = (value, length) => typeof value === 'string' && value.length === length && /^[0-9a-f]+$/.test(value);
const stages = ['L1', 'L2', 'L3', 'QA', 'MAINTENANCE'];
// Explicit caller-assessed risks; no keyword inference or implicit authorization.
const routes = new Map([
  ['boundary', ['L2']], ['alias', ['L2', 'L3']], ['abstraction', ['L2', 'L3']],
  ['precision', ['L2', 'L3', 'QA']], ['oracle', ['QA']], ['maintenance', ['MAINTENANCE']],
  ['effects', ['L2', 'L3', 'QA']], ['cost', stages]
]);
const require = (condition, message) => { if (!condition) throw new Error(message); };

export function verifyCatalog(catalog) {
  require(catalog?.sourceSha === KNOWLEDGE_SHA, 'STALE_CATALOG');
  require(Array.isArray(catalog.entries) && catalog.entries.length === routes.size, 'INVALID_CATALOG');
  const map = new Map();
  for (const e of catalog.entries) {
    require(routes.has(e.id) && !map.has(e.id), 'INVALID_CATALOG_ID');
    require(nonempty(e.path) && hex(e.blob, 40) && Number.isSafeInteger(e.start) && e.start > 0 &&
      Number.isSafeInteger(e.end) && e.end >= e.start && nonempty(e.text) &&
      e.text.split('\n').length === e.end - e.start + 1 && sha(e.text) === e.sha256, 'INVALID_EXCERPT');
    map.set(e.id, e);
  }
  require(sha(JSON.stringify(catalog)) === CATALOG_SHA256, 'CATALOG_IDENTITY_MISMATCH');
  return map;
}

export function select(context, catalog) {
  const index = verifyCatalog(catalog);
  require(nonempty(context?.decisionId) && nonempty(context.issue) && hex(context.sourceSha, 40) &&
    stages.includes(context.stage) && nonempty(context.question) && nonempty(context.contractEvidence), 'INVALID_CONTEXT');
  require(['CONFIRMED', 'MISSING', 'CONFLICT'].includes(context.contract), 'INVALID_CONTRACT');
  require(Array.isArray(context.risks), 'INVALID_RISKS');
  const maxBytes = context.maxBytes ?? 6144;
  const maxPieces = context.maxPieces ?? 3;
  require(Number.isSafeInteger(maxBytes) && maxBytes >= 0 && Number.isSafeInteger(maxPieces) && maxPieces >= 0, 'INVALID_BUDGET');
  const seen = new Set();
  for (const r of context.risks) {
    require(nonempty(r?.id) && !seen.has(r.id) && nonempty(r.evidence) && ['YES', 'NO', 'UNKNOWN'].includes(r.applicability), 'INVALID_RISK');
    seen.add(r.id);
  }
  const base = { decisionId: context.decisionId, issue: context.issue, sourceSha: context.sourceSha,
    knowledgeSha: KNOWLEDGE_SHA, stage: context.stage, question: context.question,
    contractEvidence: context.contractEvidence, budget: { maxBytes, maxPieces }, selected: [], rejected: [], usedBytes: 0 };
  if (context.contract !== 'CONFIRMED') return { ...base, status: 'ASK', reason: 'CONTRACT_' + context.contract };
  const candidates = [];
  let unresolved = false;
  for (const r of context.risks) {
    let reason;
    if (r.applicability === 'NO') reason = 'NOT_APPLICABLE';
    else if (r.applicability === 'UNKNOWN') { reason = 'APPLICABILITY_UNKNOWN'; unresolved = true; }
    else if (!routes.has(r.id)) { reason = 'UNSUPPORTED_RISK'; unresolved = true; }
    else if (!routes.get(r.id).includes(context.stage)) { reason = 'WRONG_DECISION_STAGE'; unresolved = true; }
    else candidates.push({ r, e: index.get(r.id) });
    if (reason) base.rejected.push({ id: r.id, reason, evidence: r.evidence });
  }
  // Refuse an incomplete relevant set; caller may revise evidence or explicitly enlarge soft budget.
  const total = candidates.reduce((sum, { e }) => sum + bytes(e.text), 0);
  if (total > maxBytes || candidates.length > maxPieces) {
    return { ...base, status: 'UNKNOWN', reason: 'BUDGET_INSUFFICIENT', requiredBytes: total, requiredPieces: candidates.length,
      rejected: [...base.rejected, ...candidates.map(({ r }) => ({ id: r.id, reason: 'BUDGET_INSUFFICIENT', evidence: r.evidence }))] };
  }
  if (unresolved) return { ...base, status: 'UNKNOWN', reason: 'UNRESOLVED_RISK',
    rejected: [...base.rejected, ...candidates.map(({ r }) => ({ id: r.id, reason: 'DEFERRED_UNRESOLVED_RISK', evidence: r.evidence }))] };
  if (!candidates.length) return { ...base, status: 'NONE', reason: 'NO_APPLICABLE_RISK' };
  const selected = candidates.map(({ r, e: { text, ...meta } }) => ({ ...meta, bytes: bytes(text), evidence: r.evidence }));
  return { ...base, status: 'SELECTED', reason: 'CALLER_ASSESSED_APPLICABILITY', selected, usedBytes: total };
}

export function deliver(context, catalog) {
  const plan = select(context, catalog);
  if (plan.status !== 'SELECTED') return plan;
  const index = verifyCatalog(catalog);
  const delivered = plan.selected.map(e => ({ ...e, text: index.get(e.id).text }));
  const receipt = { ...plan, status: 'DELIVERED', delivered, deliveryMeaning: 'MATERIALIZED_NOT_READ', qualification: 'ADVISORY_CANDIDATE' };
  return { ...receipt, receiptSha256: sha(JSON.stringify(receipt)) };
}

export function recordApplication(context, catalog, report) {
  const receipt = deliver(context, catalog);
  require(receipt.status === 'DELIVERED' && report?.receiptSha256 === receipt.receiptSha256, 'DELIVERY_MISMATCH');
  require(['APPLIED', 'ALREADY_ALIGNED', 'REJECTED', 'UNKNOWN'].includes(report.outcome) &&
    nonempty(report.before) && nonempty(report.after) && nonempty(report.reason) && nonempty(report.probe), 'INVALID_APPLICATION');
  if (report.outcome === 'ALREADY_ALIGNED') require(report.before === report.after, 'ALIGNMENT_CHANGED');
  if (report.outcome === 'APPLIED') require(report.before !== report.after, 'NO_DECISION_CHANGE');
  return { decisionId: receipt.decisionId, receiptSha256: receipt.receiptSha256, outcome: report.outcome,
    before: report.before, after: report.after, reason: report.reason, probe: report.probe,
    qualification: 'SELF_REPORTED_NOT_VERIFIED', qualityBenefit: 'NOT_ESTABLISHED' };
}

export function consumeBatch(contexts, catalog) {
  require(Array.isArray(contexts), 'INVALID_BATCH');
  const ids = new Set();
  return contexts.map(context => {
    require(!ids.has(context?.decisionId), 'DUPLICATE_DECISION');
    ids.add(context?.decisionId);
    return deliver(context, catalog);
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const catalog = JSON.parse(readFileSync(new URL('./catalog.json', import.meta.url), 'utf8'));
  const input = JSON.parse(readFileSync(process.argv[2], 'utf8'));
  console.log(JSON.stringify(Array.isArray(input) ? consumeBatch(input, catalog) : deliver(input, catalog), null, 2));
}
