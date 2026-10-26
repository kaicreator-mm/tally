import test from 'node:test';
import assert from 'node:assert/strict';
import { projectReport, projectReportSet, STATES } from '../src/report-projection.mjs';

const SHA = 'a'.repeat(40);
function sample(overrides = {}) {
  const report = {
    envelope: { schema: 'ads/github/v5', kind: 'report',
      data: { report_id: 'run-01', revision: 1, publisher_namespace: 'org/task-meta',
        report_kind: 'EXECUTION_SUMMARY', capture_coverage: 'COMPLETE', metrics: [{ value: null }] } },
    validation: { schema: 'ads/github/v5', passed: true },
    native: { repository: 'kaicreator-mm/tally', issueNumber: 103, commentId: 2001,
      dataDigest: 'jcs-sha256:observed-from-external-validation', sourceSha: SHA,
      permission: 'GRANTED', readback: 'VERIFIED', pagination: 'COMPLETE',
      currentness: 'CURRENT' },
  };
  return { ...report, ...overrides,
    envelope: { ...report.envelope, ...overrides.envelope,
      data: { ...report.envelope.data, ...overrides.envelope?.data } },
    native: { ...report.native, ...overrides.native },
    validation: { ...report.validation, ...overrides.validation } };
}
test('observed valid summary is never native authorization or task PASS', () => {
  const x = projectReport(sample());
  assert.equal(x.status, STATES.CURRENT);
  assert.equal(x.taskPass, false);
  assert.equal(x.authorized, false);
  assert.equal(x.releasePass, false);
  assert.equal(x.advisory.metricCount, 1);
});
test('wrong effective ADS dialect rejected', () => {
  assert.equal(projectReport(sample({ envelope: { schema: 'ads/github/v4' } })).status, STATES.UNSUPPORTED);
});
test('external exact schema validation missing is not self-certified', () => {
  assert.equal(projectReport(sample({ validation: { passed: false } })).status, STATES.INVALID);
});
test('missing native comment identity cannot turn into current fact', () => {
  assert.equal(projectReport(sample({ native: { commentId: null } })).status, STATES.INVALID);
});
test('unread/acknowledged-only GitHub is UNKNOWN', () => {
  assert.equal(projectReport(sample({ native: { readback: 'ACK' } })).status, STATES.UNKNOWN);
});
test('403 revocation is UNKNOWN even with cached complete source', () => {
  assert.equal(projectReport(sample({ native: { permission: 'DENIED' } })).status, STATES.UNKNOWN);
});
test('partial pagination prevents completeness', () => {
  assert.equal(projectReport(sample({ native: { pagination: 'PARTIAL' } })).status, STATES.PARTIAL);
});
test('changed source sha/currentness produces STALE', () => {
  assert.equal(projectReport(sample({ native: { currentness: 'STALE' } })).status, STATES.STALE);
});
test('feedback profile and unknown metric are observational only', () => {
  const x = projectReport(sample({ envelope: { data: { report_kind: 'EXPERIENCE_USE_FEEDBACK', epistemic: 'OBSERVED', metrics: undefined } } }));
  assert.equal(x.status, STATES.CURRENT);
  assert.equal(x.advisory.metricCount, null);
  assert.equal(x.advisory.epistemic, 'OBSERVED');
});
test('replaying identical observation does not fabricate conflict', () => {
  assert.equal(projectReportSet([sample(), sample()]).status, STATES.CURRENT);
});
test('conflicting same report revision with different native digest is CONFLICT', () => {
  const s = projectReportSet([sample(), sample({ native: { dataDigest: 'different' } })]);
  assert.equal(s.status, STATES.CONFLICT);
  assert.equal(s.anomalies.length, 1);
  assert.equal(s.taskPass, false);
});
test('same report identity on two comments fails closed', () => {
  assert.equal(projectReportSet([sample(), sample({ native: { commentId: 2002 } })]).status, STATES.CONFLICT);
});
test('any partial observation prevents set CURRENT', () => {
  assert.equal(projectReportSet([sample(), sample({ native: { pagination: 'PARTIAL', commentId: 2002,
    dataDigest: 'changed', sourceSha: 'b'.repeat(40) }, envelope: { data: { report_id: 'run-02' } } })]).status, STATES.UNKNOWN);
});
test('unbounded bulk observations rejected', () => {
  assert.equal(projectReportSet(new Array(10001)).status, STATES.INVALID);
});
test('missing/invalid envelope cannot throw', () => {
  assert.equal(projectReport(null).status, STATES.INVALID);
  assert.equal(projectReport({}).status, STATES.INVALID);
});

test('canonical pinned ADS v5 schema-valid summary fixture projects its actual kind', async () => {
  const { readFileSync } = await import('node:fs');
  const realEnvelope = JSON.parse(readFileSync(new URL('./fixtures/ads-v5-summary.json', import.meta.url), 'utf8'));
  assert.equal(realEnvelope.data.report_kind, 'EXECUTION_SUMMARY');
  const r = sample({ envelope: realEnvelope,
    native: { repository: 'example-org/example-repo', issueNumber: 42 } });
  // This official fixture is explicitly PARTIAL: shape-valid is not complete evidence.
  assert.equal(projectReport(r).status, STATES.PARTIAL);
  realEnvelope.data.capture_coverage = 'COMPLETE';
  assert.equal(projectReport(sample({ envelope: realEnvelope,
    native: { repository: 'example-org/example-repo', issueNumber: 42 } })).status, STATES.CURRENT);
});
test('fail closed for all permissions except exact GRANTED', () => {
  for (const permission of [undefined, null, 'UNKNOWN', 'REVOKED', 'denied', 'DENIED', 'granted']) {
    assert.equal(projectReport(sample({ native: { permission } })).status, STATES.UNKNOWN, String(permission));
  }
});
test('legacy lowercase kinds are not ADS v5 reports', () => {
  for (const report_kind of ['summary', 'feedback', 'correction']) {
    assert.equal(projectReport(sample({ envelope: { data: { report_kind } } })).status, STATES.INVALID);
  }
});
test('partial summary cannot be a complete observed report', () => {
  assert.equal(projectReport(sample({ envelope: { data: { capture_coverage: 'PARTIAL' } } })).status, STATES.PARTIAL);
  assert.equal(projectReport(sample({ envelope: { data: { capture_coverage: 'UNKNOWN' } } })).status, STATES.PARTIAL);
});
test('empty report set is not CURRENT', () => {
  assert.equal(projectReportSet([]).status, STATES.UNKNOWN);
});
test('feedback epistemic string is never treated as an arbitrary object', () => {
  assert.equal(projectReport(sample({ envelope: { data: { report_kind: 'EXPERIENCE_USE_FEEDBACK', epistemic: { state: 'OBSERVED' } } } })).advisory.epistemic, null);
});
