/**
 * Tally v2 T03a: strict ADS v4.18 Report v5 JSON Schema *shape* validator.
 * The pinned exact upstream Git blob is checked before compilation.
 * A successful parse never grants native GitHub source authenticity, a Claim,
 * reviewer status, effect permission, Task PASS or Release Qualification.
 */
import Ajv2020 from 'ajv/dist/2020.js';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

export const ADS_STANDARD_SHA = '2c395c55569fe2edf3f4135f12d3d069e526d0b7';
export const ADS_REPORT_SCHEMA_GIT_BLOB = '7478313cc006162432649d734e59c5cd77db5106';
export const ADS_REPORT_DIALECT = 'ads/github/v5';
export const REPORT_KINDS = Object.freeze({
  SUMMARY: 'EXECUTION_SUMMARY',
  FEEDBACK: 'EXPERIENCE_USE_FEEDBACK',
  CORRECTION: 'EXECUTION_CORRECTION',
});
const schemaBytes = readFileSync(new URL('../vendor/report.schema.json', import.meta.url));
const objectGitSha = createHash('sha1')
  .update(Buffer.from('blob ' + schemaBytes.length + '\0'))
  .update(schemaBytes).digest('hex');
if (objectGitSha !== ADS_REPORT_SCHEMA_GIT_BLOB) {
  throw new Error('ADS_V418_SCHEMA_BLOB_MISMATCH');
}

const schema = JSON.parse(schemaBytes.toString('utf8'));
const ajv = new Ajv2020({
  allErrors: true,
  strict: false,
  removeAdditional: false,
  coerceTypes: false,
  useDefaults: false,
  validateFormats: false,
});
const validate = ajv.compile(schema);
const fail = (code, violations = []) => Object.freeze({
  valid: false, code, violations: Object.freeze(violations),
  verifiedGitHub: false, authorized: false, taskPass: false, releasePass: false,
});
/**
 * Validates schema shape only. Does NOT validate RFC8785 JCS digest, publication
 * clearance, GitHub comment ID/Issue/repository scope or native permission.
 */
export function validateAdsV5Report(raw) {
  if (typeof raw === 'string') {
    // Reject duplicate-key ambiguity and YAML/Markdown at this boundary.
    // The transport must parse/verify its GitHub envelope separately.
    return fail('REQUIRES_PREPARSED_JSON_OBJECT');
  }
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return fail('INVALID_REPORT_TYPE');
  }
  // Never mutate input; Ajv is explicitly configured for pure checks.
  if (!validate(raw)) {
    const errs = (validate.errors ?? []).slice(0, 24).map(e =>
      Object.freeze({ pointer: e.instancePath, keyword: e.keyword, message: e.message }));
    return fail('ADS_V5_SCHEMA_INVALID', errs);
  }
  return Object.freeze({
    valid: true, code: 'ADS_V5_SHAPE_VALID_ONLY',
    reportKind: raw.data.report_kind,
    reportId: raw.data.report_id,
    revision: raw.data.revision,
    publisherNamespace: raw.data.publisher_namespace,
    schemaBlob: ADS_REPORT_SCHEMA_GIT_BLOB,
    verifiedGitHub: false, authorized: false, taskPass: false, releasePass: false,
  });
}
