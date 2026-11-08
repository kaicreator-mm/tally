/**
 * T06d shared Host resource fence - reviewed library (Pack R25, #370).
 *
 * This module is the durable authority component for the deferred
 * two-principal wiring revision. It is a library only: no existing module
 * imports it, it creates no HTTP/UI/Agent surface and it performs no GitHub
 * transport. The physical shared-Host fence and #284 remain NOT_PROVEN and
 * NOT_ADMITTED.
 *
 * What it implements:
 * - canonical typed exclusive resources with Windows case-fold and
 *   path-segment ancestor conflicts;
 * - the frozen resource-set identity digest and the Pack dispatch-id formula;
 * - one whole-registry SQLite CAS document of active leases, mutated only by a
 *   single serialized Offline Authority;
 * - RSA-SHA256 signed per-Task client requests and Authority-signed complete
 *   responses carrying result, revision and high-water;
 * - an append-only signed anchor chain outside SQLite with an in-memory
 *   expected head, restart refusal and tamper/rollback rejection;
 * - a PR #371 resourceConflictPort readback over the real durable leases.
 *
 * The default CasDocumentStore contract is the public DomainHarness one:
 * get(key) -> {value, revision} | null, put(key, value, conditions) with
 * exactly one of mustBeAbsent/mustMatchRevision, remove(key, conditions),
 * list(prefix) and close(). No other storage shape is accepted.
 */
import {createHash, createPublicKey, randomBytes, sign, verify} from 'node:crypto';
import {closeSync, existsSync, fsyncSync, openSync, readFileSync, writeSync} from 'node:fs';

export const HOST_RESOURCE_FENCE_SCHEMA = 'tally/host-resource-set/v1';
export const HOST_RESOURCE_FENCE_REGISTRY_SCHEMA = 'tally/host-resource-fence-registry/v1';
export const HOST_RESOURCE_FENCE_REQUEST_PROTOCOL = 'tally/host-resource-fence-request/v1';
export const HOST_RESOURCE_FENCE_RESPONSE_PROTOCOL = 'tally/host-resource-fence-response/v1';
export const HOST_RESOURCE_FENCE_PROTOCOL_VERSION = 1;
export const HOST_RESOURCE_FENCE_PORT_SOURCE_KIND = 'TRUSTED_HOST_RESOURCE_FENCE_PORT';
export const HOST_RESOURCE_FENCE_READBACK_SOURCE_KIND = 'TRUSTED_HOST_RESOURCE_FENCE_READBACK';
export const HOST_RESOURCE_FENCE_PACK_PATH =
  'docs/implementation/v2/HOST_SHARED_WRITER_FENCE_TASK_PACK_R1.json';
export const HOST_RESOURCE_FENCE_MAX_CONTENTION_ATTEMPTS = 16;

const NAME = /^[A-Za-z0-9_.-]+$/;
const ROLE = /^(builder|reviewer|validator)$/;
const GROUP = /^[A-Za-z0-9_.-]{1,64}$/;
const ID = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const REGISTRY_ID = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const HEX40 = /^[a-f0-9]{40}$/;
const HEX64 = /^[a-f0-9]{64}$/;
const REQUEST_ID = /^[a-f0-9]{32}$/;
const KEY = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,255}$/;
const PHASE = /^(PENDING_GITHUB_PUBLISH|PUBLISHED_VERIFIED|PENDING_GITHUB_TERMINALIZATION|TERMINAL_VERIFIED)$/;
const OPERATOR_KINDS = new Set(['chatgpt-web', 'codex', 'claude-code', 'human',
  'github-actions', 'woodpecker', 'other']);
const CLAIM_BASE_FIELDS = new Set(['phase', 'generation', 'protectedClaimKey',
  'repository', 'issueRef', 'sourceSha', 'dispatchId', 'actorRole',
  'operatorKind', 'operatorId', 'writerId']);
const CLAIM_PHASE_FIELDS = new Map([
  ['PENDING_GITHUB_PUBLISH', CLAIM_BASE_FIELDS],
  ['PUBLISHED_VERIFIED', new Set([...CLAIM_BASE_FIELDS, 'commentId', 'commentRef'])],
  ['PENDING_GITHUB_TERMINALIZATION', new Set([...CLAIM_BASE_FIELDS, 'commentId',
    'commentRef', 'terminalReservationRevision', 'terminalWriterId',
    'terminalEvidence', 'terminalOccurredAt'])],
  ['TERMINAL_VERIFIED', new Set([...CLAIM_BASE_FIELDS, 'commentId', 'commentRef',
    'terminalCommentId', 'terminalCommentRef', 'terminalEvidence',
    'terminalWriterId', 'terminalOccurredAt', 'terminalReservationRevision'])]
]);
const FSW_FIELDS = new Set(['phase', 'repository', 'issueNumber', 'sourceSha',
  'writeSetSha256', 'document', 'pathCount', 'writeSetDigestBasis',
  'taskPackRef', 'taskPackBlobSha', 'packPRHeadSha', 'packTree', 'packRevision',
  'ownerReceiptRef', 'reviewWitnessRef', 'driverReviewRef', 'protectedClaimKey',
  'generation', 'dispatchId', 'claimCommentId', 'claimCommentRef', 'writerId',
  'schema', 'exclusiveResources', 'frozenResourceSet',
  'frozenResourceSetSha256', 'frozenAt']);
const FSW_DIGEST_BASIS =
  'sha256(JSON.stringify(paths.sort())) compact; tally.v2.allowed-write-set-normalization/1';
const SID = /^S-[0-9]+-[0-9]+(?:-[0-9]+)*$/;
const WINDOWS_DRIVE = /^[A-Za-z]:/;
const CONTROL = /[\u0000-\u001f\u007f]/;
const NON_ASCII = /[^\x20-\x7e]/;
const GIT_REF_BAD = /[\u0000-\u0020\u007f~^:?*[\\]/;

const RESOURCE_TYPES = new Set([
  'file', 'claim-key', 'cas', 'github-issue-comment', 'git-ref'
]);

export class HostResourceFenceError extends Error {
  constructor(code, details) {
    super(code);
    this.name = 'HostResourceFenceError';
    this.code = code;
    this.details = Object.freeze(details === undefined ? [] :
      Array.isArray(details) ? details : [details]);
  }
}

function fail(code, ...details) {
  throw new HostResourceFenceError(code, details);
}

function plainObject(value) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false;
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

function exactKeys(value, keys, code) {
  if (!plainObject(value)) fail(code, 'NOT_PLAIN_OBJECT');
  const actual = Object.keys(value).sort();
  const expected = [...keys].sort();
  if (actual.length !== expected.length || actual.some((key, index) => key !== expected[index])) {
    fail(code, 'UNEXPECTED_FIELDS');
  }
}

function sha256Hex(value) {
  return createHash('sha256').update(value).digest('hex');
}

function nowIso(clock) {
  return new Date(clock()).toISOString();
}

function requireString(value, pattern, code, label) {
  if (typeof value !== 'string' || !pattern.test(value)) fail(code, label);
  return value;
}

/**
 * Recursive canonical JSON: plain object keys sorted by UTF-16 code unit,
 * arrays in semantic order, only JSON primitives. Undefined, functions,
 * symbols, non-finite numbers, cycles and non-plain objects are rejected.
 */
export function canonicalJson(value) {
  return serializeCanonical(value, new Set());
}

function serializeCanonical(value, seen) {
  if (value === null) return 'null';
  const type = typeof value;
  if (type === 'string') return JSON.stringify(value);
  if (type === 'boolean') return value ? 'true' : 'false';
  if (type === 'number') {
    if (!Number.isFinite(value)) fail('HOST_RESOURCE_CANONICAL_INVALID', 'NON_FINITE_NUMBER');
    return JSON.stringify(value);
  }
  if (type !== 'object') fail('HOST_RESOURCE_CANONICAL_INVALID', 'UNSUPPORTED_VALUE');
  if (seen.has(value)) fail('HOST_RESOURCE_CANONICAL_INVALID', 'CYCLE');
  seen.add(value);
  try {
    if (Array.isArray(value)) {
      const parts = [];
      for (const item of value) parts.push(serializeCanonical(item, seen));
      return '[' + parts.join(',') + ']';
    }
    if (!plainObject(value)) fail('HOST_RESOURCE_CANONICAL_INVALID', 'NON_PLAIN_OBJECT');
    const keys = Object.keys(value).sort();
    const parts = [];
    for (const key of keys) {
      const item = value[key];
      if (item === undefined) fail('HOST_RESOURCE_CANONICAL_INVALID', 'UNDEFINED_VALUE');
      parts.push(JSON.stringify(key) + ':' + serializeCanonical(item, seen));
    }
    return '{' + parts.join(',') + '}';
  } finally {
    seen.delete(value);
  }
}

function validRepository(repository) {
  if (typeof repository !== 'string') return false;
  const parts = repository.split('/');
  return parts.length === 2 && NAME.test(parts[0]) && NAME.test(parts[1]);
}

function canonicalRepository(repository) {
  return repository.split('/').map(part => part.toLowerCase()).join('/');
}

function canonicalIssueBody(body, code) {
  const match = /^([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+)#([1-9][0-9]*)$/.exec(body);
  if (!match) fail(code, 'ISSUE_RESOURCE_MALFORMED');
  return `${match[1].toLowerCase()}/${match[2].toLowerCase()}#${match[3]}`;
}

function canonicalClaimKeyBody(body, code) {
  const match = /^([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+)#([1-9][0-9]*):(builder|reviewer|validator):([A-Za-z0-9_.-]{1,64})$/
    .exec(body);
  if (!match) fail(code, 'CLAIM_KEY_MALFORMED');
  if (!ROLE.test(match[4]) || !GROUP.test(match[5])) fail(code, 'CLAIM_KEY_MALFORMED');
  return `${match[1].toLowerCase()}/${match[2].toLowerCase()}#${match[3]}:${match[4]}:${match[5]}`;
}

function validateFileBody(body, code) {
  if (typeof body !== 'string' || body.length === 0 || body.length > 1024) fail(code, 'FILE_LENGTH');
  if (CONTROL.test(body)) fail(code, 'FILE_CONTROL_CHARACTER');
  if (body.includes('\\')) fail(code, 'FILE_BACKSLASH');
  if (body.startsWith('/')) fail(code, 'FILE_ABSOLUTE');
  if (WINDOWS_DRIVE.test(body)) fail(code, 'FILE_DRIVE');
  if (body.endsWith('/')) fail(code, 'FILE_TRAILING_SLASH');
  if (body.includes('//')) fail(code, 'FILE_EMPTY_SEGMENT');
  if (body.includes(':')) fail(code, 'FILE_WINDOWS_COLON');
  if (/[<>"|?*]/.test(body)) fail(code, 'FILE_WINDOWS_FORBIDDEN');
  if (NON_ASCII.test(body)) fail(code, 'FILE_CASE_FOLD_UNKNOWN');
  for (const segment of body.split('/')) {
    if (segment === '' || segment === '.' || segment === '..') fail(code, 'FILE_SEGMENT');
    // The selected Windows Host strips trailing dots/spaces and resolves device
    // names, so these lexical forms are ambiguous aliases and deny. OS short
    // names, symlinks and junctions are outside this Git-path resource contract;
    // the separate gitWorktree rule (not admitted here) must resolve them.
    if (segment.endsWith('.') || segment.endsWith(' ')) fail(code, 'FILE_WINDOWS_TRAILING');
    if (/^(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(\..*)?$/i.test(segment)) {
      fail(code, 'FILE_WINDOWS_DEVICE');
    }
  }
  return body;
}

function validateGitRefBody(body, code) {
  if (typeof body !== 'string' || body.length === 0 || body.length > 1024) fail(code, 'GIT_REF_LENGTH');
  if (!body.startsWith('refs/')) fail(code, 'GIT_REF_NOT_FULL');
  if (GIT_REF_BAD.test(body)) fail(code, 'GIT_REF_CHARACTER');
  if (body.includes('//') || body.endsWith('/') || body.includes('..') ||
    body.includes('@{') || body.endsWith('.') || body.endsWith('.lock')) {
    fail(code, 'GIT_REF_FORM');
  }
  for (const segment of body.split('/')) {
    if (segment === '' || segment === '.' || segment === '..' || segment.startsWith('.')) {
      fail(code, 'GIT_REF_SEGMENT');
    }
  }
  return body;
}

/**
 * Parse one typed exclusive-resource string. Returns the canonical resource
 * string, its conflict key and the body needed for ancestor comparison.
 * File resources are ASCII-only on this selected Windows Host: a non-ASCII
 * path has unknown case-fold behavior and denies rather than risking a
 * case-only alias. Repository components in claim keys and Issue comment
 * destinations are folded to lower case; file spelling stays binding.
 */
export function canonicalizeHostResource(raw) {
  const code = 'HOST_RESOURCE_ENCODING_INVALID';
  if (typeof raw !== 'string' || raw.length === 0 || raw.length > 1100) fail(code, 'RESOURCE_LENGTH');
  const separator = raw.indexOf(':');
  if (separator <= 0) fail(code, 'RESOURCE_TYPE_MISSING');
  const type = raw.slice(0, separator);
  const body = raw.slice(separator + 1);
  if (!RESOURCE_TYPES.has(type)) fail('HOST_RESOURCE_TYPE_UNKNOWN', type);
  if (type === 'file') {
    const file = validateFileBody(body, code);
    return Object.freeze({
      type, body: file, resource: `file:${file}`,
      conflictKey: `file:${file.toUpperCase()}`, path: file.toUpperCase()
    });
  }
  if (type === 'claim-key') {
    const claimKey = canonicalClaimKeyBody(body, code);
    return Object.freeze({
      type, body: claimKey, resource: `claim-key:${claimKey}`,
      conflictKey: `claim-key:${claimKey}`
    });
  }
  if (type === 'github-issue-comment') {
    const issue = canonicalIssueBody(body, code);
    return Object.freeze({
      type, body: issue, resource: `github-issue-comment:${issue}`,
      conflictKey: `github-issue-comment:${issue}`
    });
  }
  if (type === 'git-ref') {
    const ref = validateGitRefBody(body, code);
    return Object.freeze({
      type, body: ref, resource: `git-ref:${ref}`, conflictKey: `git-ref:${ref}`, path: ref
    });
  }
  if (!KEY.test(body)) fail(code, 'CAS_KEY_MALFORMED');
  if (body.includes('//') || body.endsWith('/') || body.includes('..')) fail(code, 'CAS_KEY_MALFORMED');
  return Object.freeze({type, body, resource: `cas:${body}`, conflictKey: `cas:${body}`});
}

function segmentAncestor(a, b) {
  return a === b || b.startsWith(a + '/') || a.startsWith(b + '/');
}

/** True when two already-canonicalized resources exclude each other. */
export function hostResourcesConflict(left, right) {
  if (!left || !right) return false;
  if (left.type !== right.type) return false;
  if (left.type === 'file' || left.type === 'git-ref') {
    return segmentAncestor(left.path, right.path);
  }
  return left.conflictKey === right.conflictKey;
}

/**
 * Normalize and sort an exclusive-resource array. Duplicate strings, duplicate
 * canonical conflict keys and any self-conflicting pair (for example two-only
 * case paths on Windows or a file inside its own ancestor) are invalid.
 */
export function normalizeExclusiveResources(resources) {
  if (!Array.isArray(resources) || resources.length === 0) {
    fail('HOST_RESOURCE_SET_INVALID', 'RESOURCES_REQUIRED');
  }
  if (resources.length > 64) fail('HOST_RESOURCE_SET_INVALID', 'RESOURCES_TOO_MANY');
  const parsed = [];
  const seen = new Set();
  for (const raw of resources) {
    const canonical = canonicalizeHostResource(raw);
    if (seen.has(canonical.resource)) fail('HOST_RESOURCE_DUPLICATE', canonical.resource);
    seen.add(canonical.resource);
    parsed.push(canonical);
  }
  for (let i = 0; i < parsed.length; i++) {
    for (let j = i + 1; j < parsed.length; j++) {
      if (hostResourcesConflict(parsed[i], parsed[j])) {
        fail('HOST_RESOURCE_SET_SELF_CONFLICT', parsed[i].resource, parsed[j].resource);
      }
    }
  }
  const strings = parsed.map(item => item.resource).sort();
  return Object.freeze(strings);
}

const IDENTITY_KEYS = Object.freeze([
  'schema', 'repository', 'issueRef', 'protectedClaimKey', 'generation',
  'sourceSha', 'dispatchId', 'taskPackRef', 'taskPackBlobSha', 'reviewRef',
  'ownerApprovalRef', 'writeSetSha256', 'exclusiveResources'
]);

function issueNumberFromRef(ref, repository, code) {
  if (typeof ref !== 'string') fail(code, 'ISSUE_REF_MALFORMED');
  const escaped = repository.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = new RegExp(`^https://github\\.com/${escaped}/issues/([1-9][0-9]*)$`).exec(ref);
  if (!match) fail(code, 'ISSUE_REF_MALFORMED');
  return Number(match[1]);
}

function validateReviewRef(ref, repository, code) {
  if (typeof ref !== 'string') fail(code, 'REVIEW_REF_MALFORMED');
  const escaped = repository.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = new RegExp(
    `^https://github\\.com/${escaped}/pull/([1-9][0-9]*)#pullrequestreview-([1-9][0-9]*)$`
  ).exec(ref);
  if (!match) fail(code, 'REVIEW_REF_MALFORMED');
  return ref;
}

function validateOwnerRef(ref, repository, issueNumber, code) {
  if (typeof ref !== 'string') fail(code, 'OWNER_REF_MALFORMED');
  const escaped = repository.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = new RegExp(
    `^https://github\\.com/${escaped}/issues/${issueNumber}#issuecomment-([1-9][0-9]*)$`
  ).exec(ref);
  if (!match) fail(code, 'OWNER_REF_MALFORMED');
  return ref;
}

function validateTaskPackRef(ref, repository, sourceSha, code) {
  const expected = `https://github.com/${repository}/blob/${sourceSha}/${HOST_RESOURCE_FENCE_PACK_PATH}`;
  if (ref !== expected) fail(code, 'TASK_PACK_REF_MISMATCH');
  return ref;
}

/**
 * The Pack dispatch-id formula:
 * host-fence:<issue>:SHA256(UTF-8 JSON.stringify([
 *   repository, issueNumber, sourceSha, taskPackBlobSha, reviewRef,
 *   ownerApprovalRef, writeSetSha256
 * ]))
 */
export function computeDispatchId(input) {
  const code = 'HOST_RESOURCE_FENCE_IDENTITY_INVALID';
  if (!plainObject(input)) fail(code, 'DISPATCH_INPUT');
  const repository = requireString(input.repository, /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/, code, 'REPOSITORY');
  if (!Number.isSafeInteger(input.issueNumber) || input.issueNumber <= 0) fail(code, 'ISSUE_NUMBER');
  requireString(input.sourceSha, HEX40, code, 'SOURCE_SHA');
  requireString(input.taskPackBlobSha, HEX40, code, 'TASK_PACK_BLOB');
  if (typeof input.reviewRef !== 'string' || typeof input.ownerApprovalRef !== 'string') {
    fail(code, 'REFERENCE');
  }
  requireString(input.writeSetSha256, HEX64, code, 'WRITE_SET_SHA256');
  const payload = JSON.stringify([
    repository, input.issueNumber, input.sourceSha, input.taskPackBlobSha,
    input.reviewRef, input.ownerApprovalRef, input.writeSetSha256
  ]);
  return `host-fence:${input.issueNumber}:${sha256Hex(payload)}`;
}

/**
 * Build and validate the canonical frozen resource-set identity. The returned
 * digest is SHA256(canonicalBytes(object)); the object has exactly the Pack's
 * thirteen fields and its exclusiveResources are sorted by code unit.
 */
export function buildFrozenResourceSet(identity) {
  const code = 'HOST_RESOURCE_FENCE_IDENTITY_INVALID';
  exactKeys(identity, IDENTITY_KEYS, code);
  if (identity.schema !== HOST_RESOURCE_FENCE_SCHEMA) fail(code, 'SCHEMA');
  const repository = identity.repository;
  if (!validRepository(repository)) fail(code, 'REPOSITORY');
  const issueNumber = issueNumberFromRef(identity.issueRef, repository, code);
  const claimBody = canonicalClaimKeyBody(identity.protectedClaimKey, code);
  if (claimBody !== identity.protectedClaimKey) fail(code, 'PROTECTED_CLAIM_KEY_NOT_CANONICAL');
  if (!claimBody.startsWith(`${canonicalRepository(repository)}#${issueNumber}:`)) {
    fail(code, 'PROTECTED_CLAIM_KEY_MISMATCH');
  }
  if (!Number.isSafeInteger(identity.generation) || identity.generation <= 0) fail(code, 'GENERATION');
  requireString(identity.sourceSha, HEX40, code, 'SOURCE_SHA');
  requireString(identity.taskPackBlobSha, HEX40, code, 'TASK_PACK_BLOB');
  requireString(identity.writeSetSha256, HEX64, code, 'WRITE_SET_SHA256');
  validateReviewRef(identity.reviewRef, repository, code);
  validateOwnerRef(identity.ownerApprovalRef, repository, issueNumber, code);
  validateTaskPackRef(identity.taskPackRef, repository, identity.sourceSha, code);
  const expectedDispatch = computeDispatchId({
    repository, issueNumber, sourceSha: identity.sourceSha,
    taskPackBlobSha: identity.taskPackBlobSha,
    reviewRef: identity.reviewRef, ownerApprovalRef: identity.ownerApprovalRef,
    writeSetSha256: identity.writeSetSha256
  });
  if (identity.dispatchId !== expectedDispatch) fail(code, 'DISPATCH_ID_MISMATCH');
  const exclusiveResources = normalizeExclusiveResources(identity.exclusiveResources);
  const filePaths = exclusiveResources
    .filter(resource => resource.startsWith('file:'))
    .map(resource => resource.slice('file:'.length));
  if (filePaths.length === 0) fail(code, 'FILE_RESOURCES_REQUIRED');
  if (sha256Hex(JSON.stringify(filePaths)) !== identity.writeSetSha256) {
    fail(code, 'WRITE_SET_DIGEST_MISMATCH');
  }
  const requiredResources = [
    `claim-key:${claimBody}`,
    `cas:tally:v2:claim-admission:${sha256Hex(claimBody)}`,
    `cas:tally:v2:task-writeset:${issueNumber}`,
    `github-issue-comment:${canonicalRepository(repository)}#${issueNumber}`
  ];
  for (const required of requiredResources) {
    if (!exclusiveResources.includes(required)) {
      fail(code, 'REQUIRED_RESOURCE_MISSING', required);
    }
  }
  const resourceSet = Object.freeze({
    schema: HOST_RESOURCE_FENCE_SCHEMA,
    repository,
    issueRef: identity.issueRef,
    protectedClaimKey: identity.protectedClaimKey,
    generation: identity.generation,
    sourceSha: identity.sourceSha,
    dispatchId: identity.dispatchId,
    taskPackRef: identity.taskPackRef,
    taskPackBlobSha: identity.taskPackBlobSha,
    reviewRef: identity.reviewRef,
    ownerApprovalRef: identity.ownerApprovalRef,
    writeSetSha256: identity.writeSetSha256,
    exclusiveResources
  });
  const canonicalBytes = canonicalJson(resourceSet);
  return Object.freeze({
    resourceSet,
    canonicalBytes,
    frozenResourceSetSha256: sha256Hex(canonicalBytes),
    exclusiveResources
  });
}

const LEASE_KEYS = Object.freeze([
  'fenceId', 'fenceRevision', 'writerId', 'protectedClaimKey', 'sourceSha',
  'dispatchId', 'nonceHash', 'resourceSet', 'frozenResourceSetSha256', 'acquiredAt'
]);
const REGISTRY_KEYS = Object.freeze(['schema', 'registryId', 'generation', 'leases']);

function validateLease(lease, generation) {
  const code = 'HOST_RESOURCE_FENCE_REGISTRY_MALFORMED';
  exactKeys(lease, LEASE_KEYS, code);
  requireString(lease.fenceId, ID, code, 'FENCE_ID');
  if (!Number.isSafeInteger(lease.fenceRevision) || lease.fenceRevision <= 0 ||
    lease.fenceRevision > generation) fail(code, 'FENCE_REVISION');
  requireString(lease.writerId, ID, code, 'WRITER_ID');
  requireString(lease.protectedClaimKey, /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#[1-9][0-9]*:(builder|reviewer|validator):[A-Za-z0-9_.-]{1,64}$/,
    code, 'PROTECTED_CLAIM_KEY');
  requireString(lease.sourceSha, HEX40, code, 'SOURCE_SHA');
  requireString(lease.dispatchId, ID, code, 'DISPATCH_ID');
  requireString(lease.nonceHash, HEX64, code, 'NONCE_HASH');
  if (!Number.isFinite(Date.parse(lease.acquiredAt))) fail(code, 'ACQUIRED_AT');
  const built = buildFrozenResourceSet(lease.resourceSet);
  if (built.frozenResourceSetSha256 !== lease.frozenResourceSetSha256) {
    fail(code, 'RESOURCE_SET_DIGEST_MISMATCH');
  }
  if (lease.resourceSet.protectedClaimKey !== lease.protectedClaimKey ||
    lease.resourceSet.sourceSha !== lease.sourceSha ||
    lease.resourceSet.dispatchId !== lease.dispatchId) fail(code, 'LEASE_IDENTITY_MISMATCH');
  return Object.freeze({
    lease: Object.freeze({...lease, resourceSet: built.resourceSet}),
    built
  });
}

/**
 * Validate a whole registry document and its internal invariants: unique
 * fence ids and protected keys, non-overlapping leases, stable fence
 * revisions bounded by the registry generation.
 */
export function validateHostResourceFenceRegistryDocument(document, options = {}) {
  const code = 'HOST_RESOURCE_FENCE_REGISTRY_MALFORMED';
  exactKeys(document, REGISTRY_KEYS, code);
  if (document.schema !== HOST_RESOURCE_FENCE_REGISTRY_SCHEMA) fail(code, 'SCHEMA');
  requireString(document.registryId, REGISTRY_ID, code, 'REGISTRY_ID');
  if (options.registryId !== undefined && document.registryId !== options.registryId) {
    fail(code, 'REGISTRY_ID_MISMATCH');
  }
  if (!Number.isSafeInteger(document.generation) || document.generation < 0) {
    fail(code, 'GENERATION');
  }
  if (!Array.isArray(document.leases)) fail(code, 'LEASES');
  const validated = [];
  const fenceIds = new Set();
  const claimKeys = new Set();
  for (const lease of document.leases) {
    const entry = validateLease(lease, document.generation);
    if (fenceIds.has(entry.lease.fenceId)) fail(code, 'DUPLICATE_FENCE_ID');
    fenceIds.add(entry.lease.fenceId);
    if (claimKeys.has(entry.lease.protectedClaimKey)) fail(code, 'DUPLICATE_PROTECTED_CLAIM_KEY');
    claimKeys.add(entry.lease.protectedClaimKey);
    validated.push(entry);
  }
  for (let i = 0; i < validated.length; i++) {
    for (let j = i + 1; j < validated.length; j++) {
      const left = validated[i].built.resourceSet.exclusiveResources;
      const right = validated[j].built.resourceSet.exclusiveResources;
      for (const a of left) {
        for (const b of right) {
          if (hostResourcesConflict(canonicalizeHostResource(a), canonicalizeHostResource(b))) {
            fail(code, 'OVERLAPPING_LEASES');
          }
        }
      }
    }
  }
  const leases = validated.map(entry => entry.lease)
    .sort((a, b) => a.fenceId < b.fenceId ? -1 : a.fenceId > b.fenceId ? 1 : 0);
  return Object.freeze({
    schema: HOST_RESOURCE_FENCE_REGISTRY_SCHEMA,
    registryId: document.registryId,
    generation: document.generation,
    leases: Object.freeze(leases)
  });
}

export function hostResourceFenceRegistryKey(registryId) {
  requireString(registryId, REGISTRY_ID, 'HOST_RESOURCE_FENCE_REGISTRY_MALFORMED', 'REGISTRY_ID');
  return `tally:v2:host-resource-fence:${registryId}`;
}

async function readStoreSnapshot(store) {
  const code = 'HOST_RESOURCE_FENCE_STORE_UNAVAILABLE';
  let keys;
  try {
    keys = await store.list('');
  } catch (error) {
    fail(code, String(error?.code ?? 'LIST_FAILED'));
  }
  if (!Array.isArray(keys) || keys.some(key => typeof key !== 'string')) fail(code, 'LIST_SHAPE');
  const sorted = [...keys].sort();
  if (new Set(sorted).size !== sorted.length) fail('HOST_RESOURCE_FENCE_REGISTRY_DRIFT', 'DUPLICATE_KEYS');
  const rows = [];
  for (const key of sorted) {
    let got;
    try {
      got = await store.get(key);
    } catch (error) {
      fail(code, String(error?.code ?? 'GET_FAILED'));
    }
    if (!got || !Number.isSafeInteger(got.revision) || got.revision < 0) {
      fail('HOST_RESOURCE_FENCE_REGISTRY_DRIFT', 'MISSING_ROW');
    }
    rows.push({key, revision: got.revision, value: got.value});
  }
  return Object.freeze({rows, sha256: sha256Hex(canonicalJson(rows))});
}

function anchorPayload(record) {
  return {
    registryId: record.registryId,
    generation: record.generation,
    snapshotSha256: record.snapshotSha256,
    previousAnchorHash: record.previousAnchorHash,
    writerSid: record.writerSid
  };
}

function anchorRecordHash(record) {
  return sha256Hex(canonicalJson(record));
}

function readAnchorChain(anchorPath, options) {
  const code = 'HOST_RESOURCE_FENCE_ANCHOR_MALFORMED';
  if (!existsSync(anchorPath)) fail('HOST_RESOURCE_FENCE_ANCHOR_MISSING');
  const text = readFileSync(anchorPath, 'utf8');
  const lines = text.split('\n').filter(line => line.length > 0);
  if (lines.length === 0) fail(code, 'EMPTY');
  const records = [];
  let previousHash = 'GENESIS';
  let previousGeneration = -1;
  for (const line of lines) {
    let record;
    try {
      record = JSON.parse(line);
    } catch {
      fail(code, 'JSON');
    }
    exactKeys(record, [...Object.keys(anchorPayload({})), 'signature'], code);
    if (record.registryId !== options.registryId) fail(code, 'REGISTRY_ID');
    if (!Number.isSafeInteger(record.generation) || record.generation < 0 ||
      record.generation < previousGeneration) fail(code, 'GENERATION');
    requireString(record.snapshotSha256, HEX64, code, 'SNAPSHOT_SHA256');
    if (record.previousAnchorHash !== previousHash) fail(code, 'CHAIN');
    if (!SID.test(record.writerSid)) fail(code, 'WRITER_SID');
    if (options.writerSid !== undefined && record.writerSid !== options.writerSid) {
      fail(code, 'WRITER_SID');
    }
    if (typeof record.signature !== 'string' || record.signature.length === 0) fail(code, 'SIGNATURE');
    const payload = anchorPayload(record);
    let valid = false;
    try {
      valid = verify('sha256', Buffer.from(canonicalJson(payload), 'utf8'),
        options.authorityPublicKey, Buffer.from(record.signature, 'base64'));
    } catch {
      valid = false;
    }
    if (!valid) fail('HOST_RESOURCE_FENCE_ANCHOR_SIGNATURE_INVALID');
    previousHash = anchorRecordHash(record);
    previousGeneration = record.generation;
    records.push(Object.freeze({
      registryId: record.registryId,
      generation: record.generation,
      snapshotSha256: record.snapshotSha256,
      previousAnchorHash: record.previousAnchorHash,
      writerSid: record.writerSid,
      signature: record.signature
    }));
  }
  return Object.freeze({
    records: Object.freeze(records),
    head: records[records.length - 1],
    headHash: previousHash
  });
}

function appendAnchorLine(anchorPath, record) {
  let fd = null;
  try {
    fd = openSync(anchorPath, 'a');
    writeSync(fd, canonicalJson(record) + '\n');
    fsyncSync(fd);
  } catch {
    fail('HOST_RESOURCE_FENCE_ANCHOR_UNAVAILABLE');
  } finally {
    if (fd !== null) {
      try { closeSync(fd); } catch { /* close failure is reported by the next chain read */ }
    }
  }
}

function parseConditions(conditions) {
  const code = 'HOST_RESOURCE_FENCE_CAS_CONDITION_REQUIRED';
  if (!plainObject(conditions)) fail(code, 'CONDITIONS');
  const actual = Object.keys(conditions).sort();
  const hasAbsent = conditions.mustBeAbsent === true;
  const hasRevision = Object.prototype.hasOwnProperty.call(conditions, 'mustMatchRevision');
  if (actual.length !== 1 || hasAbsent === hasRevision) fail(code, 'EXACTLY_ONE_CONDITION');
  if (hasAbsent) {
    if (conditions.mustBeAbsent !== true) fail(code, 'CONDITIONS');
    return {mustBeAbsent: true};
  }
  if (!Number.isSafeInteger(conditions.mustMatchRevision) || conditions.mustMatchRevision < 0) {
    fail(code, 'REVISION');
  }
  return {mustMatchRevision: conditions.mustMatchRevision};
}

function foreignClaimFields(row) {
  return {
    protectedClaimKey: row.protectedClaimKey,
    phase: row.phase,
    sourceSha: row.sourceSha,
    dispatchId: row.dispatchId,
    writerId: row.writerId
  };
}

function foreignClaimDigest(rows) {
  const fields = rows.map(foreignClaimFields)
    .sort((a, b) => a.protectedClaimKey < b.protectedClaimKey ? -1 :
      a.protectedClaimKey > b.protectedClaimKey ? 1 : 0);
  return sha256Hex(JSON.stringify(fields));
}

function validateForeignClaimProjection(row) {
  const code = 'HOST_RESOURCE_FENCE_PORT_CONTEXT';
  exactKeys(row, ['protectedClaimKey', 'phase', 'sourceSha', 'dispatchId', 'writerId'], code);
  requireString(row.protectedClaimKey, /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#[1-9][0-9]*:(builder|reviewer|validator):[A-Za-z0-9_.-]{1,64}$/,
    code, 'PROTECTED_CLAIM_KEY');
  if (!PHASE.test(row.phase)) fail(code, 'PHASE');
  requireString(row.sourceSha, HEX40, code, 'SOURCE_SHA');
  requireString(row.dispatchId, ID, code, 'DISPATCH_ID');
  requireString(row.writerId, ID, code, 'WRITER_ID');
  return foreignClaimFields(row);
}

function claimKeyParts(claimKey) {
  const code = 'HOST_RESOURCE_FENCE_VALUE_INVALID';
  const match = /^([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+)#([1-9][0-9]*):(builder|reviewer|validator):([A-Za-z0-9_.-]{1,64})$/
    .exec(claimKey ?? '');
  if (!match) fail(code, 'CLAIM_KEY');
  return Object.freeze({
    owner: match[1], repo: match[2], issueNumber: Number(match[3]),
    role: match[4], group: match[5]
  });
}

function assertAllowedValueFields(value, allowed, code) {
  for (const field of Object.keys(value)) {
    if (!allowed.has(field)) fail(code, 'UNEXPECTED_FIELD:' + field);
  }
}

/**
 * The frozen Claim reservation/transition document accepted under the held
 * lease's own claim CAS key. Only the public T06d row fields are admitted and
 * the row must bind the same Task identity and phase/comment shape.
 */
function validateClaimReservation(value, lease) {
  const code = 'HOST_RESOURCE_FENCE_VALUE_INVALID';
  if (!plainObject(value)) fail(code, 'CLAIM_VALUE');
  const phaseFields = CLAIM_PHASE_FIELDS.get(value.phase);
  if (!phaseFields) fail(code, 'CLAIM_PHASE');
  assertAllowedValueFields(value, phaseFields, code);
  for (const field of CLAIM_BASE_FIELDS) {
    if (!Object.prototype.hasOwnProperty.call(value, field)) {
      fail(code, 'CLAIM_MISSING:' + field);
    }
  }
  if (value.generation !== lease.resourceSet.generation) fail(code, 'CLAIM_GENERATION');
  if (value.protectedClaimKey !== lease.protectedClaimKey) fail(code, 'CLAIM_KEY');
  if (value.repository !== lease.resourceSet.repository) fail(code, 'CLAIM_REPOSITORY');
  if (value.issueRef !== lease.resourceSet.issueRef) fail(code, 'CLAIM_ISSUE');
  if (value.sourceSha !== lease.sourceSha) fail(code, 'CLAIM_SOURCE');
  if (value.dispatchId !== lease.dispatchId) fail(code, 'CLAIM_DISPATCH');
  const parts = claimKeyParts(lease.protectedClaimKey);
  if (value.actorRole !== parts.role) fail(code, 'CLAIM_ROLE');
  if (!OPERATOR_KINDS.has(value.operatorKind)) fail(code, 'CLAIM_OPERATOR_KIND');
  requireString(value.operatorId, ID, code, 'CLAIM_OPERATOR_ID');
  if (value.writerId !== lease.writerId) fail(code, 'CLAIM_WRITER');
  if (value.phase !== 'PENDING_GITHUB_PUBLISH') {
    if (!Number.isSafeInteger(value.commentId) || value.commentId < 1) {
      fail(code, 'CLAIM_COMMENT_ID');
    }
    if (value.commentRef !== `${value.issueRef}#issuecomment-${value.commentId}`) {
      fail(code, 'CLAIM_COMMENT_REF');
    }
  }
  if (value.phase === 'TERMINAL_VERIFIED') {
    if (!Number.isSafeInteger(value.terminalCommentId) || value.terminalCommentId < 1 ||
      value.terminalCommentRef !==
        `${value.issueRef}#issuecomment-${value.terminalCommentId}`) {
      fail(code, 'CLAIM_TERMINAL_COMMENT');
    }
    if (typeof value.terminalEvidence !== 'string' ||
      value.terminalEvidence.length === 0) {
      fail(code, 'CLAIM_TERMINAL_EVIDENCE');
    }
    requireString(value.terminalWriterId, ID, code, 'CLAIM_TERMINAL_WRITER');
    if (typeof value.terminalOccurredAt !== 'string' ||
      !Number.isFinite(Date.parse(value.terminalOccurredAt))) {
      fail(code, 'CLAIM_TERMINAL_TIME');
    }
    if (!Number.isSafeInteger(value.terminalReservationRevision) ||
      value.terminalReservationRevision <= 0) {
      fail(code, 'CLAIM_TERMINAL_REVISION');
    }
  } else {
    if (value.terminalWriterId !== undefined) {
      requireString(value.terminalWriterId, ID, code, 'CLAIM_TERMINAL_WRITER');
    }
    if (value.terminalEvidence !== undefined &&
      (typeof value.terminalEvidence !== 'string' || value.terminalEvidence.length === 0)) {
      fail(code, 'CLAIM_TERMINAL_EVIDENCE');
    }
    if (value.terminalOccurredAt !== undefined &&
      (typeof value.terminalOccurredAt !== 'string' ||
        !Number.isFinite(Date.parse(value.terminalOccurredAt)))) {
      fail(code, 'CLAIM_TERMINAL_TIME');
    }
    if (value.terminalReservationRevision !== undefined &&
      (!Number.isSafeInteger(value.terminalReservationRevision) ||
        value.terminalReservationRevision <= 0)) {
      fail(code, 'CLAIM_TERMINAL_REVISION');
    }
  }
}

const CLAIM_TRANSITIONS = new Map([
  ['PENDING_GITHUB_PUBLISH', new Set(['PUBLISHED_VERIFIED'])],
  ['PUBLISHED_VERIFIED', new Set(['PENDING_GITHUB_TERMINALIZATION'])],
  ['PENDING_GITHUB_TERMINALIZATION', new Set(['TERMINAL_VERIFIED'])],
  ['TERMINAL_VERIFIED', new Set()]
]);
const CLAIM_IMMUTABLE_FIELDS = Object.freeze(['generation', 'protectedClaimKey',
  'repository', 'issueRef', 'sourceSha', 'dispatchId', 'actorRole',
  'operatorKind', 'operatorId', 'writerId']);

/**
 * A Claim CAS row is a forward-only state machine. The first row is a
 * PENDING_GITHUB_PUBLISH reservation; later rows must take the exact next
 * phase, keep every identity field, and never change or drop the published
 * comment. If an FSW already exists, the next row must keep the FSW's claim
 * comment binding.
 */
function validateClaimTransition(currentRow, nextValue, conditions, fswRow) {
  const code = 'HOST_RESOURCE_FENCE_VALUE_INVALID';
  if (!currentRow) {
    if (conditions.mustBeAbsent !== true) fail(code, 'CLAIM_INITIAL_CONDITION');
    if (nextValue.phase !== 'PENDING_GITHUB_PUBLISH') fail(code, 'CLAIM_INITIAL_PHASE');
    return;
  }
  if (!plainObject(currentRow.value)) fail(code, 'CLAIM_CURRENT_ROW');
  const current = currentRow.value;
  if (conditions.mustMatchRevision !== currentRow.revision) {
    fail(code, 'CLAIM_REVISION_CONDITION');
  }
  const allowed = CLAIM_TRANSITIONS.get(current.phase);
  if (!allowed || !allowed.has(nextValue.phase)) fail(code, 'CLAIM_TRANSITION');
  for (const field of CLAIM_IMMUTABLE_FIELDS) {
    if (current[field] !== nextValue[field]) fail(code, 'CLAIM_IMMUTABLE:' + field);
  }
  if (current.commentId !== undefined) {
    if (current.commentId !== nextValue.commentId ||
      current.commentRef !== nextValue.commentRef) fail(code, 'CLAIM_COMMENT_IMMUTABLE');
  }
  if (current.phase === 'PENDING_GITHUB_TERMINALIZATION' &&
    nextValue.phase === 'TERMINAL_VERIFIED') {
    for (const field of ['terminalWriterId', 'terminalEvidence',
      'terminalOccurredAt', 'terminalReservationRevision']) {
      if (current[field] !== nextValue[field]) {
        fail(code, 'CLAIM_TERMINAL_IMMUTABLE:' + field);
      }
    }
  }
  if (fswRow && plainObject(fswRow.value) &&
    (fswRow.value.claimCommentId !== nextValue.commentId ||
      fswRow.value.claimCommentRef !== nextValue.commentRef)) {
    fail(code, 'CLAIM_FSW_BINDING');
  }
}

/**
 * The frozen write-set document accepted under the held lease's own FSW key.
 * It must bind the exact Task identity, the held resource set and the Pack
 * references, and it is create-only.
 */
function validateFswDocument(value, lease, claimRow) {
  const code = 'HOST_RESOURCE_FENCE_VALUE_INVALID';
  if (!plainObject(value)) fail(code, 'FSW_VALUE');
  assertAllowedValueFields(value, FSW_FIELDS, code);
  const parts = claimKeyParts(lease.protectedClaimKey);
  if (value.phase !== 'FROZEN_DAG_WRITESET') fail(code, 'FSW_PHASE');
  if (value.repository !== lease.resourceSet.repository ||
    value.issueNumber !== parts.issueNumber) fail(code, 'FSW_TASK');
  if (value.sourceSha !== lease.sourceSha) fail(code, 'FSW_SOURCE');
  if (value.writeSetSha256 !== lease.resourceSet.writeSetSha256) {
    fail(code, 'FSW_WRITE_SET');
  }
  const filePaths = lease.resourceSet.exclusiveResources
    .filter(resource => resource.startsWith('file:'))
    .map(resource => resource.slice('file:'.length));
  const document = `frozen exact write set for #${parts.issueNumber}:\n${filePaths.join('\n')}`;
  if (value.document !== document) fail(code, 'FSW_DOCUMENT');
  if (value.pathCount !== filePaths.length) fail(code, 'FSW_PATH_COUNT');
  if (value.writeSetDigestBasis !== FSW_DIGEST_BASIS) fail(code, 'FSW_DIGEST_BASIS');
  if (value.taskPackRef !== lease.resourceSet.taskPackRef ||
    value.taskPackBlobSha !== lease.resourceSet.taskPackBlobSha) {
    fail(code, 'FSW_PACK');
  }
  if (value.packPRHeadSha !== lease.sourceSha) fail(code, 'FSW_PACK_HEAD');
  requireString(value.packTree, HEX40, code, 'FSW_PACK_TREE');
  if (!Number.isSafeInteger(value.packRevision) || value.packRevision <= 0) {
    fail(code, 'FSW_PACK_REVISION');
  }
  if (value.ownerReceiptRef !== lease.resourceSet.ownerApprovalRef ||
    value.reviewWitnessRef !== lease.resourceSet.reviewRef) {
    fail(code, 'FSW_RECEIPT_REFS');
  }
  const escaped = lease.resourceSet.repository.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const driverReview = new RegExp(
    `^https://github\\.com/${escaped}/pull/[1-9][0-9]*#pullrequestreview-[1-9][0-9]*$`
  );
  if (typeof value.driverReviewRef !== 'string' || !driverReview.test(value.driverReviewRef)) {
    fail(code, 'FSW_DRIVER_REVIEW');
  }
  if (value.protectedClaimKey !== lease.protectedClaimKey ||
    value.generation !== lease.resourceSet.generation ||
    value.dispatchId !== lease.dispatchId ||
    value.writerId !== lease.writerId) fail(code, 'FSW_IDENTITY');
  if (!Number.isSafeInteger(value.claimCommentId) || value.claimCommentId < 1 ||
    value.claimCommentRef !==
      `${lease.resourceSet.issueRef}#issuecomment-${value.claimCommentId}`) {
    fail(code, 'FSW_CLAIM_COMMENT');
  }
  if (value.schema !== HOST_RESOURCE_FENCE_SCHEMA) fail(code, 'FSW_SCHEMA');
  if (canonicalJson(value.exclusiveResources) !==
    canonicalJson(lease.resourceSet.exclusiveResources)) fail(code, 'FSW_RESOURCES');
  if (canonicalJson(value.frozenResourceSet) !==
    canonicalJson(lease.resourceSet)) fail(code, 'FSW_FROZEN_RESOURCE_SET');
  if (value.frozenResourceSetSha256 !== lease.frozenResourceSetSha256) {
    fail(code, 'FSW_FROZEN_DIGEST');
  }
  if (typeof value.frozenAt !== 'string' || !Number.isFinite(Date.parse(value.frozenAt))) {
    fail(code, 'FSW_FROZEN_AT');
  }
  if (!claimRow || !plainObject(claimRow.value)) fail(code, 'FSW_CLAIM_NOT_PUBLISHED');
  const claim = claimRow.value;
  if (!['PUBLISHED_VERIFIED', 'PENDING_GITHUB_TERMINALIZATION', 'TERMINAL_VERIFIED']
    .includes(claim.phase)) fail(code, 'FSW_CLAIM_NOT_PUBLISHED');
  if (claim.protectedClaimKey !== lease.protectedClaimKey ||
    claim.sourceSha !== lease.sourceSha ||
    claim.dispatchId !== lease.dispatchId ||
    claim.writerId !== lease.writerId) fail(code, 'FSW_CLAIM_IDENTITY');
  if (claim.commentId !== value.claimCommentId ||
    claim.commentRef !== value.claimCommentRef) fail(code, 'FSW_CLAIM_COMMENT_BINDING');
}

/**
 * Only the held lease's own two reserved CAS destinations are writable:
 * the Claim CAS row derived from its protected Claim key and the create-only
 * FSW row derived from its Issue. Any other covered CAS string denies.
 */
function validateCoveredWrite(key, value, lease, conditions, claimRow) {
  const parts = claimKeyParts(lease.protectedClaimKey);
  const claimCasKey = `tally:v2:claim-admission:${sha256Hex(lease.protectedClaimKey)}`;
  const fswKey = `tally:v2:task-writeset:${parts.issueNumber}`;
  if (key === claimCasKey) {
    validateClaimReservation(value, lease);
    return 'CLAIM_CAS';
  }
  if (key === fswKey) {
    validateFswDocument(value, lease, claimRow);
    if (conditions.mustBeAbsent !== true) {
      fail('HOST_RESOURCE_FENCE_VALUE_INVALID', 'FSW_CREATE_ONLY');
    }
    return 'FSW';
  }
  fail('HOST_RESOURCE_FENCE_KEY_NOT_COVERED');
}

function validateNonce(nonce) {
  return requireString(nonce, HEX64, 'HOST_RESOURCE_FENCE_NONCE_INVALID', 'NONCE');
}

function nonceHashOf(nonce) {
  return sha256Hex(validateNonce(nonce));
}

function parsePrivateKey(privateKey) {
  if (typeof privateKey === 'string') return privateKey;
  if (privateKey && typeof privateKey === 'object' && privateKey.type === 'private') return privateKey;
  fail('HOST_RESOURCE_FENCE_AUTHORITY_KEY_INVALID');
}

function parsePublicKeyPem(publicKeyPem) {
  try {
    return createPublicKey(publicKeyPem);
  } catch {
    fail('HOST_RESOURCE_FENCE_PUBLIC_KEY_INVALID');
  }
}

function signCanonical(privateKey, value) {
  return sign('sha256', Buffer.from(canonicalJson(value), 'utf8'), privateKey).toString('base64');
}

function verifyCanonical(publicKey, value, signature) {
  if (typeof signature !== 'string' || signature.length === 0) return false;
  try {
    return verify('sha256', Buffer.from(canonicalJson(value), 'utf8'), publicKey,
      Buffer.from(signature, 'base64'));
  } catch {
    return false;
  }
}

function requestUnsigned(request) {
  return {
    protocol: request.protocol,
    version: request.version,
    registryId: request.registryId,
    operation: request.operation,
    requestId: request.requestId,
    sequence: request.sequence,
    client: request.client,
    expected: request.expected,
    value: request.value
  };
}

function responseUnsigned(response) {
  const unsigned = {...response};
  delete unsigned.signature;
  return unsigned;
}

const OPERATIONS = new Set(['OBSERVE', 'ACQUIRE', 'ASSERT', 'RELEASE', 'PUT', 'REMOVE']);

/**
 * The single serialized Offline Authority. It owns the registry document, the
 * anchor chain and the authorized-client public keys. It never reads raw
 * nonces, tokens or private prompts, and it stores only nonce hashes and
 * public identities in CAS.
 */
class HostResourceFenceAuthority {
  #store;
  #registryId;
  #registryKey;
  #anchorPath;
  #writerSid;
  #privateKey;
  #publicKey;
  #publicKeyPem;
  #clients;
  #clock;
  #fsync;
  #maxAttempts;
  #state = 'UNINITIALIZED';
  #failed = false;
  #expectedHead = null;
  #highWater = new Map();
  #seenRequests = new Set();
  #tail = Promise.resolve();

  constructor(options) {
    const code = 'HOST_RESOURCE_FENCE_AUTHORITY_CONFIG';
    if (!plainObject(options)) fail(code);
    const store = options.store;
    if (!store || typeof store.get !== 'function' || typeof store.put !== 'function' ||
      typeof store.remove !== 'function' || typeof store.list !== 'function') fail(code, 'STORE');
    requireString(options.registryId, REGISTRY_ID, code, 'REGISTRY_ID');
    if (typeof options.anchorPath !== 'string' || options.anchorPath.length === 0) fail(code, 'ANCHOR_PATH');
    requireString(options.writerSid, SID, code, 'WRITER_SID');
    if (!Array.isArray(options.authorizedClients) || options.authorizedClients.length === 0) {
      fail(code, 'AUTHORIZED_CLIENTS');
    }
    const clients = new Map();
    const publicKeys = new Set();
    for (const entry of options.authorizedClients) {
      if (!plainObject(entry)) fail(code, 'CLIENT_ENTRY');
      exactKeys(entry, ['repository', 'protectedClaimKey', 'writerId', 'sourceSha',
        'dispatchId', 'frozenResourceSetSha256', 'publicKeyPem'], code);
      if (!validRepository(entry.repository)) fail(code, 'REPOSITORY');
      requireString(entry.protectedClaimKey,
        /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#[1-9][0-9]*:(builder|reviewer|validator):[A-Za-z0-9_.-]{1,64}$/,
        code, 'PROTECTED_CLAIM_KEY');
      requireString(entry.writerId, ID, code, 'WRITER_ID');
      requireString(entry.sourceSha, HEX40, code, 'SOURCE_SHA');
      requireString(entry.dispatchId, ID, code, 'DISPATCH_ID');
      requireString(entry.frozenResourceSetSha256, HEX64, code, 'FROZEN_RESOURCE_SET_SHA256');
      if (clients.has(entry.protectedClaimKey)) fail(code, 'DUPLICATE_CLIENT');
      const publicKey = parsePublicKeyPem(entry.publicKeyPem);
      const fingerprint = publicKey.export({type: 'spki', format: 'der'}).toString('base64');
      if (publicKeys.has(fingerprint)) fail(code, 'DUPLICATE_CLIENT_PUBLIC_KEY');
      publicKeys.add(fingerprint);
      clients.set(entry.protectedClaimKey, Object.freeze({...entry, publicKey}));
    }
    this.#store = store;
    this.#registryId = options.registryId;
    this.#registryKey = options.registryKey ?? hostResourceFenceRegistryKey(options.registryId);
    if (this.#registryKey !== hostResourceFenceRegistryKey(options.registryId)) fail(code, 'REGISTRY_KEY');
    this.#anchorPath = options.anchorPath;
    this.#writerSid = options.writerSid;
    this.#privateKey = parsePrivateKey(options.authorityPrivateKey);
    this.#publicKey = createPublicKey(this.#privateKey);
    this.#publicKeyPem = this.#publicKey.export({type: 'spki', format: 'pem'});
    this.#clients = clients;
    this.#clock = typeof options.clock === 'function' ? options.clock : () => Date.now();
    this.#fsync = options.fsync !== false;
    this.#maxAttempts = Number.isSafeInteger(options.maxContentionAttempts) ?
      options.maxContentionAttempts : HOST_RESOURCE_FENCE_MAX_CONTENTION_ATTEMPTS;
    if (this.#maxAttempts < 1 || this.#maxAttempts > 64) fail(code, 'MAX_ATTEMPTS');
  }

  get registryId() { return this.#registryId; }
  get registryKey() { return this.#registryKey; }
  get anchorPath() { return this.#anchorPath; }
  get writerSid() { return this.#writerSid; }
  get publicKeyPem() { return this.#publicKeyPem; }
  get initialized() { return this.#state === 'READY'; }
  get anchorHead() {
    if (!this.#expectedHead) fail('HOST_RESOURCE_FENCE_AUTHORITY_NOT_INITIALIZED');
    return Object.freeze({hash: this.#expectedHead.hash,
      generation: this.#expectedHead.generation});
  }

  #enqueue(task) {
    const run = this.#tail.then(task, task);
    this.#tail = run.then(() => undefined, () => undefined);
    return run;
  }

  async initialize() {
    return this.#enqueue(async () => {
      if (this.#state !== 'UNINITIALIZED') {
        fail('HOST_RESOURCE_FENCE_AUTHORITY_ALREADY_INITIALIZED');
      }
      if (this.#failed) fail('HOST_RESOURCE_FENCE_AUTHORITY_FAILED');
      if (existsSync(this.#anchorPath)) fail('HOST_RESOURCE_FENCE_AUTHORITY_RESTART_REFUSED');
      let keys;
      try {
        keys = await this.#store.list('');
      } catch (error) {
        fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'LIST_FAILED'));
      }
      if (keys.length !== 0) {
        if (keys.includes(this.#registryKey)) fail('HOST_RESOURCE_FENCE_AUTHORITY_RESTART_REFUSED');
        fail('HOST_RESOURCE_FENCE_FIRST_INIT_STORE_NOT_EMPTY');
      }
      const document = {
        schema: HOST_RESOURCE_FENCE_REGISTRY_SCHEMA,
        registryId: this.#registryId,
        generation: 0,
        leases: []
      };
      try {
        await this.#store.put(this.#registryKey, document, {mustBeAbsent: true});
      } catch (error) {
        if (error?.code === 'HOST_KEY_EXISTS') fail('HOST_RESOURCE_FENCE_REGISTRY_RECREATED');
        fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'PUT_FAILED'));
      }
      const snapshot = await readStoreSnapshot(this.#store);
      const registryRow = snapshot.rows.find(row => row.key === this.#registryKey);
      if (snapshot.rows.length !== 1 || !registryRow || registryRow.revision !== 0 ||
        canonicalJson(registryRow.value) !== canonicalJson(document)) {
        this.#failed = true;
        fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'GENESIS_NON_ANCHOR_MUTATION');
      }
      const record = {
        registryId: this.#registryId,
        generation: 0,
        snapshotSha256: snapshot.sha256,
        previousAnchorHash: 'GENESIS',
        writerSid: this.#writerSid
      };
      record.signature = signCanonical(this.#privateKey, record);
      appendAnchorLine(this.#anchorPath, record);
      const chain = readAnchorChain(this.#anchorPath, {
        registryId: this.#registryId,
        authorityPublicKey: this.#publicKey,
        writerSid: this.#writerSid
      });
      this.#expectedHead = {hash: chain.headHash, generation: chain.head.generation};
      this.#state = 'READY';
      return Object.freeze({
        status: 'INITIALIZED', registryId: this.#registryId,
        registryRevision: 0, registryGeneration: 0,
        anchorHash: chain.headHash, snapshotSha256: snapshot.sha256
      });
    });
  }

  async withReadLock(task) {
    if (typeof task !== 'function') {
      fail('HOST_RESOURCE_FENCE_AUTHORITY_CONFIG', 'READ_LOCK_TASK');
    }
    return this.#enqueue(async () => {
      if (this.#failed) {
        throw new HostResourceFenceError('HOST_RESOURCE_FENCE_AUTHORITY_FAILED');
      }
      if (this.#state !== 'READY') {
        throw new HostResourceFenceError('HOST_RESOURCE_FENCE_AUTHORITY_NOT_INITIALIZED');
      }
      await this.#assertIntegrity();
      return task();
    });
  }

  async handle(request) {
    return this.#enqueue(async () => {
      if (this.#failed) throw new HostResourceFenceError('HOST_RESOURCE_FENCE_AUTHORITY_FAILED');
      if (this.#state !== 'READY') {
        throw new HostResourceFenceError('HOST_RESOURCE_FENCE_AUTHORITY_NOT_INITIALIZED');
      }
      let denial = null;
      try {
        return await this.#dispatch(request);
      } catch (error) {
        if (!(error instanceof HostResourceFenceError)) throw error;
        if (String(error.code).startsWith('HOST_RESOURCE_FENCE_ANCHOR_') ||
          error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_FAILED' ||
          error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_NOT_INITIALIZED' ||
          error.code === 'HOST_RESOURCE_FENCE_STORE_UNAVAILABLE' ||
          error.code === 'HOST_RESOURCE_FENCE_REGISTRY_MALFORMED' ||
          error.code === 'HOST_RESOURCE_FENCE_REGISTRY_RECREATED' ||
          error.code === 'HOST_RESOURCE_FENCE_REGISTRY_DRIFT') {
          throw error;
        }
        denial = error;
      }
      return this.#denial(request, denial);
    });
  }

  #denial(request, error) {
    const clientKey = plainObject(request) && plainObject(request.client) &&
      typeof request.client.protectedClaimKey === 'string' ?
      request.client.protectedClaimKey : null;
    const response = {
      protocol: HOST_RESOURCE_FENCE_RESPONSE_PROTOCOL,
      version: HOST_RESOURCE_FENCE_PROTOCOL_VERSION,
      registryId: this.#registryId,
      requestId: typeof request?.requestId === 'string' ? request.requestId : '',
      operation: typeof request?.operation === 'string' ? request.operation : '',
      status: 'DENIED',
      code: error.code,
      details: [...error.details],
      registryRevision: null,
      registryGeneration: null,
      clientRequestHighWater: clientKey ? (this.#highWater.get(clientKey) ?? 0) : 0,
      snapshotSha256: null,
      anchorHash: null,
      writerSid: this.#writerSid
    };
    response.signature = signCanonical(this.#privateKey, responseUnsigned(response));
    return Object.freeze(response);
  }

  #validateRequest(request) {
    const code = 'HOST_RESOURCE_FENCE_REQUEST_INVALID';
    exactKeys(request, ['protocol', 'version', 'registryId', 'operation', 'requestId',
      'sequence', 'client', 'expected', 'value', 'signature'], code);
    if (request.protocol !== HOST_RESOURCE_FENCE_REQUEST_PROTOCOL ||
      request.version !== HOST_RESOURCE_FENCE_PROTOCOL_VERSION) fail(code, 'PROTOCOL');
    if (request.registryId !== this.#registryId) fail(code, 'REGISTRY_ID');
    if (!OPERATIONS.has(request.operation)) fail(code, 'OPERATION');
    requireString(request.requestId, REQUEST_ID, code, 'REQUEST_ID');
    if (!Number.isSafeInteger(request.sequence) || request.sequence <= 0) fail(code, 'SEQUENCE');
    exactKeys(request.client, ['protectedClaimKey', 'writerId'], code);
    requireString(request.client.protectedClaimKey,
      /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#[1-9][0-9]*:(builder|reviewer|validator):[A-Za-z0-9_.-]{1,64}$/,
      code, 'PROTECTED_CLAIM_KEY');
    requireString(request.client.writerId, ID, code, 'WRITER_ID');
    if (request.expected !== null) {
      exactKeys(request.expected, ['registryRevision', 'registryGeneration'], code);
      for (const field of ['registryRevision', 'registryGeneration']) {
        const value = request.expected[field];
        if (value !== null && (!Number.isSafeInteger(value) || value < 0)) fail(code, field);
      }
    }
    if (!plainObject(request.value)) fail(code, 'VALUE');
    if (typeof request.signature !== 'string' || request.signature.length === 0) fail(code, 'SIGNATURE');
  }

  #authorize(request) {
    const entry = this.#clients.get(request.client.protectedClaimKey);
    if (!entry || entry.writerId !== request.client.writerId) {
      fail('HOST_RESOURCE_FENCE_CLIENT_UNAUTHORIZED');
    }
    if (!verifyCanonical(entry.publicKey, requestUnsigned(request), request.signature)) {
      fail('HOST_RESOURCE_FENCE_REQUEST_SIGNATURE_INVALID');
    }
    const seenKey = `${request.client.protectedClaimKey}|${request.requestId}`;
    if (this.#seenRequests.has(seenKey)) fail('HOST_RESOURCE_FENCE_REQUEST_REPLAY');
    const highWater = this.#highWater.get(request.client.protectedClaimKey) ?? 0;
    if (request.sequence <= highWater) fail('HOST_RESOURCE_FENCE_REQUEST_SEQUENCE_REPLAY');
    this.#seenRequests.add(seenKey);
    return entry;
  }

  async #readRegistry() {
    let got;
    try {
      got = await this.#store.get(this.#registryKey);
    } catch (error) {
      fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'GET_FAILED'));
    }
    if (!got) fail('HOST_RESOURCE_FENCE_REGISTRY_RECREATED');
    if (!Number.isSafeInteger(got.revision) || got.revision < 0) {
      fail('HOST_RESOURCE_FENCE_REGISTRY_MALFORMED', 'REVISION');
    }
    const document = validateHostResourceFenceRegistryDocument(got.value,
      {registryId: this.#registryId});
    return {document, revision: got.revision};
  }

  async #assertIntegrity() {
    const chain = readAnchorChain(this.#anchorPath, {
      registryId: this.#registryId,
      authorityPublicKey: this.#publicKey
    });
    if (!this.#expectedHead ||
      chain.headHash !== this.#expectedHead.hash ||
      chain.head.generation !== this.#expectedHead.generation) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH');
    }
    const snapshot = await readStoreSnapshot(this.#store);
    if (snapshot.sha256 !== chain.head.snapshotSha256) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'SNAPSHOT');
    }
    const registry = await this.#readRegistry();
    if (registry.document.generation !== chain.head.generation) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'GENERATION');
    }
    const row = snapshot.rows.find(entry => entry.key === this.#registryKey);
    if (!row || row.revision !== registry.revision ||
      canonicalJson(row.value) !== canonicalJson(registry.document)) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'REGISTRY_ROW');
    }
    return {chain, snapshot, registry};
  }

  #verifySnapshotDelta(before, after, expected) {
    const beforeMap = new Map(before.rows.map(row => [row.key, row]));
    const afterMap = new Map(after.rows.map(row => [row.key, row]));
    for (const key of new Set([...beforeMap.keys(), ...afterMap.keys()])) {
      if (key === expected.allowedKey) continue;
      const prior = beforeMap.get(key);
      const next = afterMap.get(key);
      if (!prior || !next || prior.revision !== next.revision ||
        canonicalJson(prior.value) !== canonicalJson(next.value)) {
        this.#failed = true;
        fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'NON_ANCHOR_MUTATION');
      }
    }
    if (expected.removed) {
      if (afterMap.has(expected.allowedKey)) {
        this.#failed = true;
        fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'TARGET_NOT_REMOVED');
      }
      return;
    }
    const target = afterMap.get(expected.allowedKey);
    if (!target || target.revision !== expected.expectedRevision ||
      canonicalJson(target.value) !== canonicalJson(expected.expectedValue)) {
      this.#failed = true;
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'TARGET_MISMATCH');
    }
  }

  async #appendAnchor(generation, snapshot) {
    const record = {
      registryId: this.#registryId,
      generation,
      snapshotSha256: snapshot.sha256,
      previousAnchorHash: this.#expectedHead.hash,
      writerSid: this.#writerSid
    };
    record.signature = signCanonical(this.#privateKey, record);
    try {
      appendAnchorLine(this.#anchorPath, record);
      const chain = readAnchorChain(this.#anchorPath, {
        registryId: this.#registryId,
        authorityPublicKey: this.#publicKey,
        writerSid: this.#writerSid
      });
      if (chain.headHash !== anchorRecordHash(record) ||
        chain.head.snapshotSha256 !== snapshot.sha256 ||
        chain.head.generation !== generation) {
        fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH');
      }
      this.#expectedHead = {hash: chain.headHash, generation};
      return chain.headHash;
    } catch (error) {
      this.#failed = true;
      if (error instanceof HostResourceFenceError &&
        String(error.code).startsWith('HOST_RESOURCE_FENCE_ANCHOR_')) throw error;
      throw new HostResourceFenceError('HOST_RESOURCE_FENCE_ANCHOR_UNAVAILABLE');
    }
  }

  #findLease(document, fenceId) {
    const lease = document.leases.find(entry => entry.fenceId === fenceId);
    if (!lease) fail('HOST_RESOURCE_FENCE_NOT_HELD');
    return lease;
  }

  #checkHeld(lease, nonce) {
    requireString(nonce, HEX64, 'HOST_RESOURCE_FENCE_NONCE_INVALID', 'NONCE');
    if (sha256Hex(nonce) !== lease.nonceHash) fail('HOST_RESOURCE_FENCE_NONCE_MISMATCH');
  }

  async #mutateRegistry(buildNext) {
    for (let attempt = 1; attempt <= this.#maxAttempts; attempt++) {
      const integrity = await this.#assertIntegrity();
      const current = integrity.registry;
      const built = buildNext(current.document);
      try {
        const result = await this.#store.put(this.#registryKey, built.document,
          {mustMatchRevision: current.revision});
        return {
          document: built.document,
          revision: result.revision,
          attempts: attempt,
          beforeSnapshot: integrity.snapshot,
          meta: built.meta ?? null
        };
      } catch (error) {
        if (error?.code === 'HOST_CAS_CONFLICT') continue;
        fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'PUT_FAILED'));
      }
    }
    fail('HOST_RESOURCE_FENCE_CONTENTION_EXHAUSTED');
  }

  async #success(request, entry, fields) {
    this.#highWater.set(request.client.protectedClaimKey, request.sequence);
    const response = {
      protocol: HOST_RESOURCE_FENCE_RESPONSE_PROTOCOL,
      version: HOST_RESOURCE_FENCE_PROTOCOL_VERSION,
      registryId: this.#registryId,
      requestId: request.requestId,
      operation: request.operation,
      status: 'OK',
      registryRevision: fields.registryRevision,
      registryGeneration: fields.registryGeneration,
      clientRequestHighWater: request.sequence,
      snapshotSha256: fields.snapshotSha256 ?? null,
      anchorHash: fields.anchorHash ?? null,
      writerSid: this.#writerSid,
      result: fields.result
    };
    response.signature = signCanonical(this.#privateKey, responseUnsigned(response));
    return Object.freeze(response);
  }

  async #dispatch(request) {
    const integrity = await this.#assertIntegrity();
    this.#validateRequest(request);
    const entry = this.#authorize(request);
    const {registry} = integrity;
    const document = registry.document;

    if (request.operation === 'OBSERVE') {
      return this.#success(request, entry, {
        registryRevision: registry.revision,
        registryGeneration: document.generation,
        snapshotSha256: integrity.snapshot.sha256,
        anchorHash: integrity.chain.headHash,
        result: {
          registryId: this.#registryId,
          leases: document.leases.map(lease => Object.freeze({
            fenceId: lease.fenceId,
            fenceRevision: lease.fenceRevision,
            protectedClaimKey: lease.protectedClaimKey,
            writerId: lease.writerId
          }))
        }
      });
    }

    if (request.operation === 'ACQUIRE') {
      exactKeys(request.value, ['resourceSet', 'nonceHash'], 'HOST_RESOURCE_FENCE_REQUEST_INVALID');
      const built = buildFrozenResourceSet(request.value.resourceSet);
      requireString(request.value.nonceHash, HEX64, 'HOST_RESOURCE_FENCE_NONCE_INVALID', 'NONCE_HASH');
      if (built.frozenResourceSetSha256 !== entry.frozenResourceSetSha256) {
        fail('HOST_RESOURCE_FENCE_RESOURCE_SET_NOT_AUTHORIZED');
      }
      if (built.resourceSet.repository !== entry.repository ||
        built.resourceSet.sourceSha !== entry.sourceSha ||
        built.resourceSet.dispatchId !== entry.dispatchId ||
        built.resourceSet.protectedClaimKey !== entry.protectedClaimKey) {
        fail('HOST_RESOURCE_FENCE_RESOURCE_SET_NOT_AUTHORIZED');
      }
      const incoming = built.resourceSet.exclusiveResources.map(canonicalizeHostResource);
      const fenceId = `fence-${randomBytes(12).toString('hex')}`;
      const mutation = await this.#mutateRegistry(current => {
        if (current.leases.some(lease => lease.protectedClaimKey === entry.protectedClaimKey)) {
          fail('HOST_RESOURCE_FENCE_CONFLICT', 'PROTECTED_CLAIM_KEY_HELD');
        }
        for (const lease of current.leases) {
          for (const resource of lease.resourceSet.exclusiveResources) {
            const existing = canonicalizeHostResource(resource);
            for (const candidate of incoming) {
              if (hostResourcesConflict(candidate, existing)) {
                fail('HOST_RESOURCE_FENCE_CONFLICT', candidate.resource);
              }
            }
          }
        }
        const generation = current.generation + 1;
        const lease = {
          fenceId,
          fenceRevision: generation,
          writerId: entry.writerId,
          protectedClaimKey: entry.protectedClaimKey,
          sourceSha: entry.sourceSha,
          dispatchId: entry.dispatchId,
          nonceHash: request.value.nonceHash,
          resourceSet: built.resourceSet,
          frozenResourceSetSha256: built.frozenResourceSetSha256,
          acquiredAt: nowIso(this.#clock)
        };
        const document = validateHostResourceFenceRegistryDocument({
          schema: HOST_RESOURCE_FENCE_REGISTRY_SCHEMA,
          registryId: this.#registryId,
          generation,
          leases: [...current.leases, lease]
        }, {registryId: this.#registryId});
        return {document, meta: {lease}};
      });
      const lease = mutation.document.leases.find(entryLease => entryLease.fenceId === fenceId);
      const snapshot = await readStoreSnapshot(this.#store);
      this.#verifySnapshotDelta(mutation.beforeSnapshot, snapshot, {
        allowedKey: this.#registryKey,
        expectedValue: mutation.document,
        expectedRevision: mutation.revision
      });
      const anchorHash = await this.#appendAnchor(mutation.document.generation, snapshot);
      return this.#success(request, entry, {
        registryRevision: mutation.revision,
        registryGeneration: mutation.document.generation,
        snapshotSha256: snapshot.sha256,
        anchorHash,
        result: {
          fenceId: lease.fenceId,
          fenceRevision: lease.fenceRevision,
          registryId: this.#registryId,
          attempts: mutation.attempts
        }
      });
    }

    if (request.operation === 'ASSERT') {
      exactKeys(request.value, ['fenceId', 'nonce'], 'HOST_RESOURCE_FENCE_REQUEST_INVALID');
      requireString(request.value.fenceId, ID, 'HOST_RESOURCE_FENCE_REQUEST_INVALID', 'FENCE_ID');
      const lease = this.#findLease(document, request.value.fenceId);
      this.#checkHeld(lease, request.value.nonce);
      if (lease.frozenResourceSetSha256 !== entry.frozenResourceSetSha256) {
        fail('HOST_RESOURCE_FENCE_NOT_HELD');
      }
      return this.#success(request, entry, {
        registryRevision: registry.revision,
        registryGeneration: document.generation,
        snapshotSha256: integrity.snapshot.sha256,
        anchorHash: integrity.chain.headHash,
        result: {
          fenceId: lease.fenceId,
          fenceRevision: lease.fenceRevision,
          registryId: this.#registryId,
          held: true
        }
      });
    }

    if (request.operation === 'RELEASE') {
      exactKeys(request.value, ['fenceId', 'nonce'], 'HOST_RESOURCE_FENCE_REQUEST_INVALID');
      requireString(request.value.fenceId, ID, 'HOST_RESOURCE_FENCE_REQUEST_INVALID', 'FENCE_ID');
      let releasedLease = null;
      const mutation = await this.#mutateRegistry(current => {
        const lease = this.#findLease(current, request.value.fenceId);
        this.#checkHeld(lease, request.value.nonce);
        if (lease.frozenResourceSetSha256 !== entry.frozenResourceSetSha256 ||
          lease.protectedClaimKey !== entry.protectedClaimKey) {
          fail('HOST_RESOURCE_FENCE_NOT_HELD');
        }
        const generation = current.generation + 1;
        const document = validateHostResourceFenceRegistryDocument({
          schema: HOST_RESOURCE_FENCE_REGISTRY_SCHEMA,
          registryId: this.#registryId,
          generation,
          leases: current.leases.filter(item => item.fenceId !== lease.fenceId)
        }, {registryId: this.#registryId});
        releasedLease = lease;
        return {document};
      });
      const snapshot = await readStoreSnapshot(this.#store);
      this.#verifySnapshotDelta(mutation.beforeSnapshot, snapshot, {
        allowedKey: this.#registryKey,
        expectedValue: mutation.document,
        expectedRevision: mutation.revision
      });
      const anchorHash = await this.#appendAnchor(mutation.document.generation, snapshot);
      return this.#success(request, entry, {
        registryRevision: mutation.revision,
        registryGeneration: mutation.document.generation,
        snapshotSha256: snapshot.sha256,
        anchorHash,
        result: {
          releasedFenceId: releasedLease.fenceId,
          releasedFenceRevision: releasedLease.fenceRevision,
          registryId: this.#registryId,
          attempts: mutation.attempts
        }
      });
    }

    if (request.operation === 'PUT' || request.operation === 'REMOVE') {
      const valueKeys = request.operation === 'PUT' ?
        ['fenceId', 'nonce', 'key', 'value', 'conditions'] :
        ['fenceId', 'nonce', 'key', 'conditions'];
      exactKeys(request.value, valueKeys, 'HOST_RESOURCE_FENCE_REQUEST_INVALID');
      requireString(request.value.fenceId, ID, 'HOST_RESOURCE_FENCE_REQUEST_INVALID', 'FENCE_ID');
      requireString(request.value.key, KEY, 'HOST_RESOURCE_FENCE_REQUEST_INVALID', 'KEY');
      if (request.value.key === this.#registryKey) {
        fail('HOST_RESOURCE_FENCE_KEY_RESERVED');
      }
      const lease = this.#findLease(document, request.value.fenceId);
      this.#checkHeld(lease, request.value.nonce);
      if (lease.frozenResourceSetSha256 !== entry.frozenResourceSetSha256 ||
        lease.protectedClaimKey !== entry.protectedClaimKey) {
        fail('HOST_RESOURCE_FENCE_NOT_HELD');
      }
      const parts = claimKeyParts(lease.protectedClaimKey);
      const claimCasKey = `tally:v2:claim-admission:${sha256Hex(lease.protectedClaimKey)}`;
      const fswKey = `tally:v2:task-writeset:${parts.issueNumber}`;
      const covered = lease.resourceSet.exclusiveResources.includes(`cas:${request.value.key}`);
      if (!covered) fail('HOST_RESOURCE_FENCE_KEY_NOT_COVERED');
      const conditions = parseConditions(request.value.conditions);
      if (request.operation === 'PUT') {
        let claimRow = null;
        let currentClaimRow = null;
        let fswRow = null;
        try {
          if (request.value.key === fswKey) {
            claimRow = await this.#store.get(claimCasKey);
            if (!claimRow) {
              fail('HOST_RESOURCE_FENCE_VALUE_INVALID', 'FSW_CLAIM_NOT_PUBLISHED');
            }
          }
          if (request.value.key === claimCasKey) {
            currentClaimRow = await this.#store.get(claimCasKey);
            fswRow = await this.#store.get(fswKey);
          }
        } catch (error) {
          if (error instanceof HostResourceFenceError) throw error;
          fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'GET_FAILED'));
        }
        validateCoveredWrite(request.value.key, request.value.value, lease, conditions, claimRow);
        if (request.value.key === claimCasKey) {
          validateClaimTransition(currentClaimRow, request.value.value, conditions, fswRow);
        }
      } else {
        fail('HOST_RESOURCE_FENCE_KEY_RESERVED');
      }
      let revisionForResult;
      try {
        if (request.operation === 'PUT') {
          const result = await this.#store.put(request.value.key, request.value.value, conditions);
          revisionForResult = result.revision;
        } else {
          await this.#store.remove(request.value.key, conditions);
          revisionForResult = null;
        }
      } catch (error) {
        if (error?.code === 'HOST_CAS_CONFLICT') {
          fail('HOST_RESOURCE_FENCE_CAS_CONFLICT', request.value.key);
        }
        fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'WRITE_FAILED'));
      }
      const snapshot = await readStoreSnapshot(this.#store);
      this.#verifySnapshotDelta(integrity.snapshot, snapshot,
        request.operation === 'PUT' ?
          {allowedKey: request.value.key, expectedValue: request.value.value,
            expectedRevision: revisionForResult} :
          {allowedKey: request.value.key, removed: true});
      const anchorHash = await this.#appendAnchor(document.generation, snapshot);
      return this.#success(request, entry, {
        registryRevision: registry.revision,
        registryGeneration: document.generation,
        snapshotSha256: snapshot.sha256,
        anchorHash,
        result: {
          key: request.value.key,
          revision: revisionForResult
        }
      });
    }

    fail('HOST_RESOURCE_FENCE_OPERATION_UNSUPPORTED', request.operation);
  }
}

export function createHostResourceFenceAuthority(options) {
  return new HostResourceFenceAuthority(options);
}

function validateClientIdentity(identity, built) {
  if (built.resourceSet.protectedClaimKey !== identity.protectedClaimKey) {
    fail('HOST_RESOURCE_FENCE_IDENTITY_INVALID', 'PROTECTED_CLAIM_KEY');
  }
}

/**
 * Per-Task Online client. It keeps the raw owner nonce in its closure only,
 * signs every complete request with its disposable Task private key, verifies
 * the Authority's complete signed response and tracks its request sequence.
 */
class HostResourceFenceClient {
  #privateKey;
  #authorityPublicKey;
  #send;
  #registryId;
  #writerId;
  #identity;
  #built;
  #nonce;
  #nonceHash;
  #sequence;
  #fenceId = null;
  #fenceRevision = null;
  #requestLog = [];

  constructor(options) {
    const code = 'HOST_RESOURCE_FENCE_CLIENT_CONFIG';
    if (!plainObject(options)) fail(code);
    this.#privateKey = parsePrivateKey(options.privateKey);
    this.#authorityPublicKey = parsePublicKeyPem(options.authorityPublicKeyPem);
    if (typeof options.send !== 'function') fail(code, 'SEND');
    requireString(options.registryId, REGISTRY_ID, code, 'REGISTRY_ID');
    requireString(options.writerId, ID, code, 'WRITER_ID');
    if (!plainObject(options.identity)) fail(code, 'IDENTITY');
    this.#built = buildFrozenResourceSet(options.identity);
    validateClientIdentity(options.identity, this.#built);
    this.#send = options.send;
    this.#registryId = options.registryId;
    this.#writerId = options.writerId;
    this.#identity = this.#built.resourceSet;
    this.#nonce = validateNonce(options.nonce);
    this.#nonceHash = nonceHashOf(this.#nonce);
    this.#sequence = Number.isSafeInteger(options.sequenceStart) ? options.sequenceStart : 0;
    if (this.#sequence < 0) fail(code, 'SEQUENCE_START');
  }

  get frozenResourceSetSha256() { return this.#built.frozenResourceSetSha256; }
  get fenceId() { return this.#fenceId; }
  get fenceRevision() { return this.#fenceRevision; }
  get requestLog() { return Object.freeze([...this.#requestLog]); }

  async #call(operation, value, expected) {
    const sequence = ++this.#sequence;
    const request = {
      protocol: HOST_RESOURCE_FENCE_REQUEST_PROTOCOL,
      version: HOST_RESOURCE_FENCE_PROTOCOL_VERSION,
      registryId: this.#registryId,
      operation,
      requestId: randomBytes(16).toString('hex'),
      sequence,
      client: {
        protectedClaimKey: this.#identity.protectedClaimKey,
        writerId: this.#writerId
      },
      expected,
      value
    };
    request.signature = signCanonical(this.#privateKey, requestUnsigned(request));
    this.#requestLog.push(Object.freeze({
      operation, requestId: request.requestId, sequence: request.sequence
    }));
    const response = await this.#send(request);
    if (!plainObject(response)) fail('HOST_RESOURCE_FENCE_RESPONSE_INVALID');
    if (!verifyCanonical(this.#authorityPublicKey, responseUnsigned(response), response.signature)) {
      fail('HOST_RESOURCE_FENCE_RESPONSE_SIGNATURE_INVALID');
    }
    if (response.protocol !== HOST_RESOURCE_FENCE_RESPONSE_PROTOCOL ||
      response.version !== HOST_RESOURCE_FENCE_PROTOCOL_VERSION ||
      response.registryId !== this.#registryId ||
      response.requestId !== request.requestId ||
      response.operation !== operation) {
      fail('HOST_RESOURCE_FENCE_RESPONSE_BINDING_INVALID');
    }
    if (response.status !== 'OK') {
      throw new HostResourceFenceError(response.code ?? 'HOST_RESOURCE_FENCE_RESPONSE_DENIED',
        response.details);
    }
    if (response.clientRequestHighWater !== sequence) {
      fail('HOST_RESOURCE_FENCE_RESPONSE_HIGH_WATER_INVALID');
    }
    return response;
  }

  async observe() {
    const response = await this.#call('OBSERVE', {}, null);
    return response.result;
  }

  async acquire() {
    const response = await this.#call('ACQUIRE', {
      resourceSet: this.#identity,
      nonceHash: this.#nonceHash
    }, null);
    this.#fenceId = response.result.fenceId;
    this.#fenceRevision = response.result.fenceRevision;
    return response.result;
  }

  async assert(fenceId = this.#fenceId) {
    if (!ID.test(fenceId ?? '')) fail('HOST_RESOURCE_FENCE_NOT_HELD');
    const response = await this.#call('ASSERT', {fenceId, nonce: this.#nonce}, null);
    return response.result;
  }

  async release(fenceId = this.#fenceId) {
    if (!ID.test(fenceId ?? '')) fail('HOST_RESOURCE_FENCE_NOT_HELD');
    const response = await this.#call('RELEASE',
      {fenceId, nonce: this.#nonce}, null);
    if (fenceId === this.#fenceId) {
      this.#fenceId = null;
      this.#fenceRevision = null;
    }
    return response.result;
  }

  async put(key, value, conditions) {
    if (!this.#fenceId) fail('HOST_RESOURCE_FENCE_NOT_HELD');
    const response = await this.#call('PUT', {
      fenceId: this.#fenceId,
      nonce: this.#nonce,
      key,
      value,
      conditions
    }, null);
    return response.result;
  }

  async remove(key, conditions) {
    if (!this.#fenceId) fail('HOST_RESOURCE_FENCE_NOT_HELD');
    const response = await this.#call('REMOVE', {
      fenceId: this.#fenceId,
      nonce: this.#nonce,
      key,
      conditions
    }, null);
    return response.result;
  }
}

export function createHostResourceFenceClient(options) {
  return new HostResourceFenceClient(options);
}

/**
 * PR #371 resourceConflictPort over the real durable leases. All authority
 * comes from the anchor-verified registry and its leases; caller-provided
 * projections are checked against the durable rows and can never supply a
 * boolean witness.
 */
class HostResourceFencePort {
  #store;
  #registryId;
  #registryKey;
  #anchorPath;
  #writerSid;
  #authorityPublicKey;
  #clock;
  #repository;
  #writerId;
  #protectedClaimKey;
  #sourceSha;
  #frozenResourceSetSha256;
  #expectedHeadProvider;
  #withReadLock;

  constructor(options) {
    const code = 'HOST_RESOURCE_FENCE_PORT_CONFIG';
    if (!plainObject(options)) fail(code);
    const store = options.store;
    if (!store || typeof store.get !== 'function' || typeof store.list !== 'function') {
      fail(code, 'STORE');
    }
    requireString(options.registryId, REGISTRY_ID, code, 'REGISTRY_ID');
    if (typeof options.anchorPath !== 'string' || options.anchorPath.length === 0) {
      fail(code, 'ANCHOR_PATH');
    }
    requireString(options.writerSid, SID, code, 'WRITER_SID');
    if (!validRepository(options.repository)) fail(code, 'REPOSITORY');
    requireString(options.writerId, ID, code, 'WRITER_ID');
    requireString(options.protectedClaimKey,
      /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#[1-9][0-9]*:(builder|reviewer|validator):[A-Za-z0-9_.-]{1,64}$/,
      code, 'PROTECTED_CLAIM_KEY');
    requireString(options.sourceSha, HEX40, code, 'SOURCE_SHA');
    requireString(options.frozenResourceSetSha256, HEX64, code, 'FROZEN_RESOURCE_SET_SHA256');
    if (typeof options.expectedHead !== 'function') fail(code, 'EXPECTED_HEAD_PROVIDER');
    this.#expectedHeadProvider = options.expectedHead;
    if (typeof options.withReadLock !== 'function') fail(code, 'WITH_READ_LOCK');
    this.#withReadLock = options.withReadLock;
    this.#store = store;
    this.#registryId = options.registryId;
    this.#registryKey = options.registryKey ?? hostResourceFenceRegistryKey(options.registryId);
    if (this.#registryKey !== hostResourceFenceRegistryKey(options.registryId)) fail(code, 'REGISTRY_KEY');
    this.#anchorPath = options.anchorPath;
    this.#writerSid = options.writerSid;
    this.#authorityPublicKey = parsePublicKeyPem(options.authorityPublicKeyPem);
    this.#clock = typeof options.clock === 'function' ? options.clock : () => Date.now();
    this.#repository = options.repository;
    this.#writerId = options.writerId;
    this.#protectedClaimKey = options.protectedClaimKey;
    this.#sourceSha = options.sourceSha;
    this.#frozenResourceSetSha256 = options.frozenResourceSetSha256;
  }

  get sourceKind() { return HOST_RESOURCE_FENCE_PORT_SOURCE_KIND; }
  get repository() { return this.#repository; }
  get writerId() { return this.#writerId; }
  get frozenResourceSetSha256() { return this.#frozenResourceSetSha256; }

  async #readVerifiedState() {
    const chain = readAnchorChain(this.#anchorPath, {
      registryId: this.#registryId,
      authorityPublicKey: this.#authorityPublicKey,
      writerSid: this.#writerSid
    });
    const expectedHead = this.#expectedHeadProvider();
    exactKeys(expectedHead, ['hash', 'generation'], 'HOST_RESOURCE_FENCE_PORT_CONTEXT');
    requireString(expectedHead.hash, HEX64, 'HOST_RESOURCE_FENCE_PORT_CONTEXT',
      'EXPECTED_HEAD_HASH');
    if (!Number.isSafeInteger(expectedHead.generation) ||
      expectedHead.generation < 0) {
      fail('HOST_RESOURCE_FENCE_PORT_CONTEXT', 'EXPECTED_HEAD_GENERATION');
    }
    if (chain.headHash !== expectedHead.hash ||
      chain.head.generation !== expectedHead.generation) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'EXPECTED_HEAD');
    }
    const snapshot = await readStoreSnapshot(this.#store);
    if (snapshot.sha256 !== chain.head.snapshotSha256) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'SNAPSHOT');
    }
    let got;
    try {
      got = await this.#store.get(this.#registryKey);
    } catch (error) {
      fail('HOST_RESOURCE_FENCE_STORE_UNAVAILABLE', String(error?.code ?? 'GET_FAILED'));
    }
    if (!got) fail('HOST_RESOURCE_FENCE_REGISTRY_RECREATED');
    const document = validateHostResourceFenceRegistryDocument(got.value,
      {registryId: this.#registryId});
    if (document.generation !== chain.head.generation) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'GENERATION');
    }
    const row = snapshot.rows.find(entry => entry.key === this.#registryKey);
    if (!row || row.revision !== got.revision ||
      canonicalJson(row.value) !== canonicalJson(document)) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'REGISTRY_ROW');
    }
    return {document, chain, expectedHead};
  }

  async readHeldFence(context) {
    return this.#withReadLock(() => this.#readHeldFenceLocked(context));
  }

  async #readHeldFenceLocked(context) {
    const code = 'HOST_RESOURCE_FENCE_PORT_CONTEXT';
    exactKeys(context, ['repository', 'writerId', 'protectedClaimKey', 'sourceSha',
      'foreignClaimCount', 'foreignClaimsSha256', 'foreignClaims'], code);
    if (context.repository !== this.#repository ||
      context.writerId !== this.#writerId ||
      context.protectedClaimKey !== this.#protectedClaimKey ||
      context.sourceSha !== this.#sourceSha) fail(code, 'BINDING');
    if (!Number.isSafeInteger(context.foreignClaimCount) || context.foreignClaimCount < 0) {
      fail(code, 'FOREIGN_COUNT');
    }
    requireString(context.foreignClaimsSha256, HEX64, code, 'FOREIGN_SHA256');
    if (!Array.isArray(context.foreignClaims) ||
      context.foreignClaims.length !== context.foreignClaimCount) fail(code, 'FOREIGN_CLAIMS');
    const projections = context.foreignClaims.map(validateForeignClaimProjection);
    const recomputed = foreignClaimDigest(projections);
    if (recomputed !== context.foreignClaimsSha256) fail(code, 'FOREIGN_DIGEST');
    const state = await this.#readVerifiedState();
    const held = state.document.leases.find(lease =>
      lease.protectedClaimKey === this.#protectedClaimKey &&
      lease.writerId === this.#writerId &&
      lease.sourceSha === this.#sourceSha &&
      lease.frozenResourceSetSha256 === this.#frozenResourceSetSha256);
    if (!held) fail('HOST_RESOURCE_FENCE_NOT_HELD');
    const heldResources = held.resourceSet.exclusiveResources.map(canonicalizeHostResource);
    let incompatibleResourceCount = 0;
    for (const projection of projections) {
      const foreignLease = state.document.leases.find(lease =>
        lease.protectedClaimKey === projection.protectedClaimKey);
      if (!foreignLease || foreignLease.writerId !== projection.writerId ||
        foreignLease.sourceSha !== projection.sourceSha ||
        foreignLease.dispatchId !== projection.dispatchId) {
        incompatibleResourceCount++;
        continue;
      }
      const foreignResources = foreignLease.resourceSet.exclusiveResources
        .map(canonicalizeHostResource);
      let conflict = false;
      for (const left of heldResources) {
        for (const right of foreignResources) {
          if (hostResourcesConflict(left, right)) conflict = true;
        }
      }
      if (conflict) incompatibleResourceCount++;
    }
    const finalChain = readAnchorChain(this.#anchorPath, {
      registryId: this.#registryId,
      authorityPublicKey: this.#authorityPublicKey,
      writerSid: this.#writerSid
    });
    const finalExpected = this.#expectedHeadProvider();
    if (finalChain.headHash !== state.chain.headHash ||
      finalChain.head.generation !== state.chain.head.generation ||
      finalExpected?.hash !== state.expectedHead.hash ||
      finalExpected?.generation !== state.expectedHead.generation) {
      fail('HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH', 'READ_ADVANCED');
    }
    return Object.freeze({
      sourceKind: HOST_RESOURCE_FENCE_READBACK_SOURCE_KIND,
      complete: true,
      fenceHeld: true,
      fenceScope: 'EXACT_FROZEN_RESOURCE_SET',
      heldThrough: 'CAS_PUBLISH_READBACK',
      repository: this.#repository,
      writerId: this.#writerId,
      protectedClaimKey: this.#protectedClaimKey,
      sourceSha: this.#sourceSha,
      foreignClaimsSha256: recomputed,
      foreignClaimCount: projections.length,
      incompatibleResourceCount,
      frozenResourceSetSha256: this.#frozenResourceSetSha256,
      fenceId: held.fenceId,
      fenceRevision: held.fenceRevision,
      observedAt: nowIso(this.#clock)
    });
  }
}

export function createHostResourceFencePort(options) {
  return new HostResourceFencePort(options);
}
