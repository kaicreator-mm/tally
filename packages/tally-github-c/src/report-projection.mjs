/**
 * Tally v2 T01 — ADS v4.18 GitHub v5 report read-model.
 * Pure, no HTTP, no GitHub write, no Kernel authority, no task-meta private read.
 * Inputs are observed records from a separately trusted C GitHub transport and
 * separately version-pinned ADS schema validator. Nothing here grants permission.
 */
export const STATES = Object.freeze({
  CURRENT: 'CURRENT_OBSERVED',
  PARTIAL: 'PARTIAL',
  STALE: 'STALE',
  UNKNOWN: 'UNKNOWN',
  UNSUPPORTED: 'UNSUPPORTED_SCHEMA',
  INVALID: 'INVALID_OBSERVATION',
  CONFLICT: 'CONFLICT',
});
// Exact ADS v4.18 `ads/github/v5` wire values (pinned schema blob 7478313c...).
const KINDS = new Set(['EXECUTION_SUMMARY', 'EXPERIENCE_USE_FEEDBACK', 'EXECUTION_CORRECTION']);
const object = (v) => v !== null && typeof v === 'object' && !Array.isArray(v);
const nonempty = (v) => typeof v === 'string' && v.length > 0 && v.length <= 512;
const sha = (v) => typeof v === 'string' && /^[0-9a-f]{40}$/.test(v);
const exactInteger = (v) => Number.isSafeInteger(v) && v >= 1;
const result = (status, reason, source = null, advisory = null) =>
  Object.freeze({ status, reason, source, advisory, authorized: false, taskPass: false, releasePass: false });

/** A validated ADS report is still only observed data, never Task/Review PASS. */
export function projectReport(observation) {
  if (!object(observation) || !object(observation.envelope) ||
      !object(observation.native) || !object(observation.validation)) {
    return result(STATES.INVALID, 'MISSING_OBSERVATION_FIELDS');
  }
  const { envelope, native, validation } = observation;
  if (envelope.schema !== 'ads/github/v5' || envelope.kind !== 'report') {
    return result(STATES.UNSUPPORTED, 'WRONG_ADS_REPORT_DIALECT');
  }
  if (!object(envelope.data) || !KINDS.has(envelope.data.report_kind) ||
      !nonempty(envelope.data.report_id) ||
      !nonempty(envelope.data.publisher_namespace) ||
      !exactInteger(envelope.data.revision)) {
    return result(STATES.INVALID, 'INVALID_REPORT_IDENTITY');
  }
  if (validation.schema !== 'ads/github/v5' || validation.passed !== true) {
    return result(STATES.INVALID, 'REQUIRES_EXTERNAL_PINNED_SCHEMA_VALIDATION');
  }
  if (!nonempty(native.repository) || !exactInteger(native.issueNumber) ||
      !exactInteger(native.commentId) || !nonempty(native.dataDigest) ||
      !sha(native.sourceSha)) {
    return result(STATES.INVALID, 'MISSING_NATIVE_SOURCE_IDENTITY');
  }

  const identity = Object.freeze({
    repository: native.repository,
    issueNumber: native.issueNumber,
    commentId: native.commentId,
    namespace: envelope.data.publisher_namespace,
    reportId: envelope.data.report_id,
    revision: envelope.data.revision,
    kind: envelope.data.report_kind,
    sourceSha: native.sourceSha,
    dataDigest: native.dataDigest,
  });
  // An absent, revoked, unrecognized or unknown grant is NOT a readable source.
  if (native.permission !== 'GRANTED') return result(STATES.UNKNOWN, 'GITHUB_PERMISSION_NOT_GRANTED', identity);
  if (native.readback !== 'VERIFIED') return result(STATES.UNKNOWN, 'NO_NATIVE_READBACK', identity);
  if (native.pagination !== 'COMPLETE') return result(STATES.PARTIAL, 'NATIVE_PAGINATION_INCOMPLETE', identity);
  if (native.currentness !== 'CURRENT') return result(STATES.STALE, 'NATIVE_SOURCE_STALE', identity);
  if (identity.kind === 'EXECUTION_SUMMARY' && envelope.data.capture_coverage !== 'COMPLETE') {
    return result(STATES.PARTIAL, 'REPORT_CAPTURE_NOT_COMPLETE', identity);
  }
  // Reconciliation of correction targets is owned by later T03c.
  // Never surface a REVOKE/REPLACE notice as CURRENT while its target may
  // still be cached; even a perfectly schema-valid correction is advisory.
  if (identity.kind === 'EXECUTION_CORRECTION') {
    return result(STATES.UNKNOWN, 'CORRECTION_NOT_RECONCILED', identity);
  }
  // Source qualification is merely an observed advisory projection.
  // Review/Claim/Effect/Release authority stays with native GitHub + the selected K.
  const advisory = Object.freeze({
    reportKind: identity.kind,
    publicationRef: native.commentId,
    epistemic: identity.kind === 'EXPERIENCE_USE_FEEDBACK' &&
      ['OBSERVED', 'SELF_REPORTED', 'INFERRED', 'UNKNOWN'].includes(envelope.data.epistemic)
      ? envelope.data.epistemic : null,
    metricCount: Array.isArray(envelope.data.metrics) ? envelope.data.metrics.length : null,
  });
  return result(STATES.CURRENT, 'NATIVE_REPORT_OBSERVED', identity, advisory);
}

/** Identical identity with divergent digest/source/comment must fail closed. */
export function projectReportSet(observations) {
  if (!Array.isArray(observations) || observations.length > 10000) {
    return Object.freeze({ status: STATES.INVALID, records: [], anomalies: ['UNBOUNDED_OBSERVATIONS'] });
  }
  if (observations.length === 0) {
    return Object.freeze({ status: STATES.UNKNOWN, records: Object.freeze([]),
      anomalies: Object.freeze([]), authorized: false, taskPass: false, releasePass: false });
  }
  const records = [];
  const seen = new Map();
  const anomalies = [];
  for (const observation of observations) {
    const row = projectReport(observation);
    records.push(row);
    if (!row.source) continue;
    const s = row.source;
    const key = JSON.stringify([s.repository, s.issueNumber, s.namespace, s.reportId, s.revision, s.kind]);
    const before = seen.get(key);
    if (before && (before.sourceSha !== s.sourceSha || before.dataDigest !== s.dataDigest ||
        before.commentId !== s.commentId)) {
      anomalies.push(Object.freeze({ type: 'CONFLICTING_REPORT_IDENTITY', key }));
    } else if (!before) seen.set(key, s);
  }
  const hasUnsafe = records.some((x) => x.status !== STATES.CURRENT);
  return Object.freeze({
    status: anomalies.length ? STATES.CONFLICT : hasUnsafe ? STATES.UNKNOWN : STATES.CURRENT,
    records: Object.freeze(records),
    anomalies: Object.freeze(anomalies),
    authorized: false,
    taskPass: false,
    releasePass: false,
  });
}
