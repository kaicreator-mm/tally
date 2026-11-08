/**
 * Deterministic tests for the T06d shared Host resource fence library.
 *
 * Real public DomainHarness openSqliteCasStore over disposable temporary
 * SQLite, disposable RSA keys, no GitHub transport and no live Claim/POST.
 * The governing Pack's worked canonicalization vector is read from the
 * repository and must reproduce byte-for-byte.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, rm} from 'node:fs/promises';
import {existsSync, readFileSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createHash, generateKeyPairSync, randomBytes, sign} from 'node:crypto';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {
  HOST_RESOURCE_FENCE_SCHEMA,
  HostResourceFenceError,
  buildFrozenResourceSet,
  canonicalJson,
  canonicalizeHostResource,
  computeDispatchId,
  createHostResourceFenceAuthority,
  createHostResourceFenceClient,
  createHostResourceFencePort,
  hostResourcesConflict,
  normalizeExclusiveResources,
  validateHostResourceFenceRegistryDocument
} from './host-resource-fence.mjs';

const REPO = 'kaicreator-mm/tally';
const WRITER_SID = 'S-1-5-21-2839165489-1832998833-1574237796-1004';
const AUTHORITY_KEY = generateKeyPairSync('rsa', {modulusLength: 2048});
const CLIENT_KEY_A = generateKeyPairSync('rsa', {modulusLength: 2048});
const CLIENT_KEY_B = generateKeyPairSync('rsa', {modulusLength: 2048});
const CLIENT_KEY_X = generateKeyPairSync('rsa', {modulusLength: 2048});
const FORGED_KEY = generateKeyPairSync('rsa', {modulusLength: 2048});
const ADMITTED_FILES = [
  'experiments/v2/t06d-host-claim-publication/host-resource-fence.mjs',
  'experiments/v2/t06d-host-claim-publication/host-resource-fence.test.mjs'
];

const sha256 = value => createHash('sha256').update(value).digest('hex');
const publicPem = keyPair => keyPair.publicKey.export({type: 'spki', format: 'pem'});
const nonce = () => randomBytes(32).toString('hex');

function claimKey(issueNumber = 370) {
  return `${REPO}#${issueNumber}:builder:__default__`;
}

function baseResources(issueNumber = 370, files = ADMITTED_FILES) {
  const key = claimKey(issueNumber);
  return [
    `cas:tally:v2:claim-admission:${sha256(key)}`,
    `cas:tally:v2:task-writeset:${issueNumber}`,
    `claim-key:${key}`,
    ...files.map(file => `file:${file}`),
    `github-issue-comment:${REPO}#${issueNumber}`
  ];
}

function makeIdentity({
  issueNumber = 370,
  files = ADMITTED_FILES,
  extra = [],
  sourceSha = 'a'.repeat(40),
  packSha = 'b'.repeat(40),
  reviewId = 1000000001,
  ownerId = 1000000002,
  generation = 1
} = {}) {
  const protectedClaimKey = claimKey(issueNumber);
  const exclusiveResources = [...baseResources(issueNumber, files), ...extra];
  const issueRef = `https://github.com/${REPO}/issues/${issueNumber}`;
  const reviewRef = `https://github.com/${REPO}/pull/376#pullrequestreview-${reviewId}`;
  const ownerApprovalRef = `https://github.com/${REPO}/issues/${issueNumber}#issuecomment-${ownerId}`;
  const writeSetSha256 = sha256(JSON.stringify([...files].sort()));
  const dispatchId = computeDispatchId({
    repository: REPO, issueNumber, sourceSha, taskPackBlobSha: packSha,
    reviewRef, ownerApprovalRef, writeSetSha256
  });
  return {
    schema: HOST_RESOURCE_FENCE_SCHEMA,
    repository: REPO,
    issueRef,
    protectedClaimKey,
    generation,
    sourceSha,
    dispatchId,
    taskPackRef: `https://github.com/${REPO}/blob/${sourceSha}/docs/implementation/v2/HOST_SHARED_WRITER_FENCE_TASK_PACK_R1.json`,
    taskPackBlobSha: packSha,
    reviewRef,
    ownerApprovalRef,
    writeSetSha256,
    exclusiveResources
  };
}

function authorizedEntry(identity, built, keyPair, writerId) {
  return {
    repository: identity.repository,
    protectedClaimKey: identity.protectedClaimKey,
    writerId,
    sourceSha: identity.sourceSha,
    dispatchId: identity.dispatchId,
    frozenResourceSetSha256: built.frozenResourceSetSha256,
    publicKeyPem: publicPem(keyPair)
  };
}

function clientFor({authority, identity, keyPair, writerId, clientNonce, sequenceStart, send}) {
  return createHostResourceFenceClient({
    privateKey: keyPair.privateKey,
    authorityPublicKeyPem: authority.publicKeyPem,
    registryId: authority.registryId,
    writerId,
    identity,
    nonce: clientNonce,
    sequenceStart,
    send: send ?? (request => authority.handle(request))
  });
}

function deepCopy(value) {
  return JSON.parse(JSON.stringify(value));
}

async function openHarness(t, {clients = [], store: providedStore, dir: providedDir} = {}) {
  const dir = providedDir ?? await mkdtemp(join(tmpdir(), 'tally-host-fence-'));
  const store = providedStore ?? await openSqliteCasStore({
    path: join(dir, 'fence.sqlite'), busyTimeoutMs: 1000
  });
  const anchorPath = join(dir, 'anchor.ndjson');
  const authority = createHostResourceFenceAuthority({
    store,
    registryId: 'fence-registry-test-1',
    anchorPath,
    writerSid: WRITER_SID,
    authorityPrivateKey: AUTHORITY_KEY.privateKey,
    authorizedClients: clients
  });
  await authority.initialize();
  t.after(async () => {
    try { await store.close(); } catch { /* already closed */ }
    await rm(dir, {recursive: true, force: true});
  });
  return {dir, store, anchorPath, authority};
}

function wrappingStore(store) {
  let failNext = 0;
  return {
    get: (...args) => store.get(...args),
    list: (...args) => store.list(...args),
    remove: (...args) => store.remove(...args),
    close: (...args) => store.close(...args),
    put: async (key, value, conditions) => {
      if (failNext > 0 && conditions?.mustMatchRevision !== undefined) {
        failNext--;
        const error = new Error('injected CAS conflict');
        error.code = 'HOST_CAS_CONFLICT';
        throw error;
      }
      return store.put(key, value, conditions);
    },
    failNext(count) { failNext = count; }
  };
}

function leaseFor(identity, built, {fenceId, fenceRevision, writerId, nonceValue = 'ab'.repeat(32)}) {
  return {
    fenceId,
    fenceRevision,
    writerId,
    protectedClaimKey: identity.protectedClaimKey,
    sourceSha: identity.sourceSha,
    dispatchId: identity.dispatchId,
    nonceHash: sha256(nonceValue),
    resourceSet: built.resourceSet,
    frozenResourceSetSha256: built.frozenResourceSetSha256,
    acquiredAt: new Date().toISOString()
  };
}

function projectionFor(lease) {
  return {
    protectedClaimKey: lease.protectedClaimKey,
    phase: 'PUBLISHED_VERIFIED',
    sourceSha: lease.sourceSha,
    dispatchId: lease.dispatchId,
    writerId: lease.writerId
  };
}

function claimValue(identity, overrides = {}) {
  return {
    phase: 'PENDING_GITHUB_PUBLISH',
    generation: identity.generation,
    protectedClaimKey: identity.protectedClaimKey,
    repository: identity.repository,
    issueRef: identity.issueRef,
    sourceSha: identity.sourceSha,
    dispatchId: identity.dispatchId,
    actorRole: 'builder',
    operatorKind: 'other',
    operatorId: 'zcode:i370:test:1',
    writerId: 'writer-a',
    ...overrides
  };
}

function fswValue(identity, built, writerId, claimCommentId = 123, overrides = {}) {
  const match = /issues\/([1-9][0-9]*)$/.exec(identity.issueRef);
  const issueNumber = Number(match[1]);
  const filePaths = built.exclusiveResources
    .filter(resource => resource.startsWith('file:'))
    .map(resource => resource.slice('file:'.length));
  return {
    phase: 'FROZEN_DAG_WRITESET',
    repository: identity.repository,
    issueNumber,
    sourceSha: identity.sourceSha,
    writeSetSha256: identity.writeSetSha256,
    document: `frozen exact write set for #${issueNumber}:\n${filePaths.join('\n')}`,
    pathCount: filePaths.length,
    writeSetDigestBasis: 'sha256(JSON.stringify(paths.sort())) compact; tally.v2.allowed-write-set-normalization/1',
    taskPackRef: identity.taskPackRef,
    taskPackBlobSha: identity.taskPackBlobSha,
    packPRHeadSha: identity.sourceSha,
    packTree: 'c'.repeat(40),
    packRevision: 25,
    ownerReceiptRef: identity.ownerApprovalRef,
    reviewWitnessRef: identity.reviewRef,
    driverReviewRef: 'https://github.com/kaicreator-mm/tally/pull/376#pullrequestreview-5819951791',
    protectedClaimKey: identity.protectedClaimKey,
    generation: identity.generation,
    dispatchId: identity.dispatchId,
    claimCommentId,
    claimCommentRef: `${identity.issueRef}#issuecomment-${claimCommentId}`,
    writerId,
    schema: 'tally/host-resource-set/v1',
    exclusiveResources: built.exclusiveResources,
    frozenResourceSet: built.resourceSet,
    frozenResourceSetSha256: built.frozenResourceSetSha256,
    frozenAt: new Date().toISOString(),
    ...overrides
  };
}

function projectionDigest(projections) {
  const fields = projections.map(row => ({
    protectedClaimKey: row.protectedClaimKey,
    phase: row.phase,
    sourceSha: row.sourceSha,
    dispatchId: row.dispatchId,
    writerId: row.writerId
  })).sort((a, b) => a.protectedClaimKey < b.protectedClaimKey ? -1 :
    a.protectedClaimKey > b.protectedClaimKey ? 1 : 0);
  return sha256(JSON.stringify(fields));
}

test('canonical resource set reproduces the Pack worked vector exactly', () => {
  const pack = JSON.parse(readFileSync(
    new URL('../../../docs/implementation/v2/HOST_SHARED_WRITER_FENCE_TASK_PACK_R1.json',
      import.meta.url), 'utf8'));
  const vector = pack.canonicalResourceContract.workedVector;
  const inputs = vector.inputs;
  const built = buildFrozenResourceSet({
    schema: HOST_RESOURCE_FENCE_SCHEMA,
    repository: inputs.repository,
    issueRef: inputs.issueRef,
    protectedClaimKey: inputs.protectedClaimKey,
    generation: inputs.generation,
    sourceSha: inputs.sourceSha,
    dispatchId: inputs.dispatchId,
    taskPackRef: `https://github.com/${inputs.repository}/blob/${inputs.sourceSha}/` +
      'docs/implementation/v2/HOST_SHARED_WRITER_FENCE_TASK_PACK_R1.json',
    taskPackBlobSha: inputs.taskPackBlobSha,
    reviewRef: inputs.reviewRef,
    ownerApprovalRef: inputs.ownerApprovalRef,
    writeSetSha256: inputs.writeSetSha256,
    exclusiveResources: vector.exclusiveResources
  });
  assert.equal(built.canonicalBytes, vector.canonicalBytes);
  assert.equal(built.frozenResourceSetSha256, vector.frozenResourceSetSha256);
  assert.equal(built.frozenResourceSetSha256,
    '348502c6540996f96a66504d1ce187cf24a36ecd959dc83981026b2f32a281e7');
  assert.deepEqual([...built.exclusiveResources], [...vector.exclusiveResources].sort());
  const dispatchId = computeDispatchId({
    repository: inputs.repository,
    issueNumber: 370,
    sourceSha: inputs.sourceSha,
    taskPackBlobSha: inputs.taskPackBlobSha,
    reviewRef: inputs.reviewRef,
    ownerApprovalRef: inputs.ownerApprovalRef,
    writeSetSha256: inputs.writeSetSha256
  });
  assert.equal(dispatchId, inputs.dispatchId);
});

test('file resources conflict on exact, segment ancestor and Windows case only', () => {
  const exact = canonicalizeHostResource('file:experiments/a.mjs');
  const ancestor = canonicalizeHostResource('file:experiments/a.mjs/child');
  const uppercase = canonicalizeHostResource('file:EXPERIMENTS/A.MJS');
  const sibling = canonicalizeHostResource('file:experiments/ab.mjs');
  assert.equal(hostResourcesConflict(exact, exact), true);
  assert.equal(hostResourcesConflict(exact, ancestor), true);
  assert.equal(hostResourcesConflict(ancestor, exact), true);
  assert.equal(hostResourcesConflict(exact, uppercase), true);
  assert.equal(hostResourcesConflict(exact, sibling), false);
  assert.equal(hostResourcesConflict(exact,
    canonicalizeHostResource('file:other/experiments/a.mjs')), false);
  for (const invalid of ['file:', 'file:../x', 'file:/abs', 'file:C:/x',
    'file:a\\b', 'file:a/./b', 'file:a//b', 'file:a/', 'file:ünicode',
    'file:dir/item.', 'file:dir/item ', 'file:dir/CON', 'file:dir/con.txt',
    'file:a:b', 'file:a<b']) {
    assert.throws(() => canonicalizeHostResource(invalid),
      error => error.code === 'HOST_RESOURCE_ENCODING_INVALID', invalid);
  }
  assert.throws(() => canonicalizeHostResource('worktree:x'),
    error => error.code === 'HOST_RESOURCE_TYPE_UNKNOWN');
});

test('git-ref, claim-key, CAS and issue-comment resources canonicalize exactly', () => {
  const ref = canonicalizeHostResource('git-ref:refs/heads/task/x');
  const child = canonicalizeHostResource('git-ref:refs/heads/task/x/sub');
  const other = canonicalizeHostResource('git-ref:refs/heads/task/y');
  assert.equal(hostResourcesConflict(ref, child), true);
  assert.equal(hostResourcesConflict(ref, other), false);
  for (const invalid of ['git-ref:heads/x', 'git-ref:refs/../x',
    'git-ref:refs/heads/x~1', 'git-ref:refs/heads/x.lock']) {
    assert.throws(() => canonicalizeHostResource(invalid),
      error => error.code === 'HOST_RESOURCE_ENCODING_INVALID', invalid);
  }
  const upperClaim = canonicalizeHostResource(
    'claim-key:Kaicreator-MM/Tally#370:builder:__default__');
  const lowerClaim = canonicalizeHostResource(`claim-key:${claimKey()}`);
  assert.equal(upperClaim.resource, `claim-key:${claimKey()}`);
  assert.equal(hostResourcesConflict(upperClaim, lowerClaim), true);
  assert.equal(hostResourcesConflict(lowerClaim,
    canonicalizeHostResource('claim-key:kaicreator-mm/tally#371:builder:__default__')), false);
  const cas = canonicalizeHostResource('cas:tally:v2:task-writeset:370');
  assert.equal(hostResourcesConflict(cas, cas), true);
  assert.equal(hostResourcesConflict(cas,
    canonicalizeHostResource('cas:tally:v2:task-writeset:371')), false);
  const issue = canonicalizeHostResource(`github-issue-comment:${REPO}#370`);
  assert.equal(hostResourcesConflict(issue,
    canonicalizeHostResource(`github-issue-comment:${REPO.toUpperCase()}#370`)), true);
  assert.equal(hostResourcesConflict(issue,
    canonicalizeHostResource(`github-issue-comment:${REPO}#371`)), false);
  assert.throws(() => normalizeExclusiveResources(['cas:a', 'cas:a']),
    error => error.code === 'HOST_RESOURCE_DUPLICATE');
  assert.throws(() => normalizeExclusiveResources([
    'file:experiments/a.mjs', 'file:EXPERIMENTS/A.MJS'
  ]), error => error.code === 'HOST_RESOURCE_SET_SELF_CONFLICT');
  assert.throws(() => normalizeExclusiveResources([
    `claim-key:${claimKey()}`,
    `claim-key:${REPO.toUpperCase()}#370:builder:__default__`
  ]), error => error.code === 'HOST_RESOURCE_DUPLICATE');
  const sorted = normalizeExclusiveResources([
    `github-issue-comment:${REPO}#370`, 'cas:z', 'file:a/b'
  ]);
  assert.deepEqual([...sorted], [...sorted].sort());
});

test('canonical JSON rejects non-values and metadata binds the identity digest', () => {
  assert.equal(canonicalJson({b: 1, a: [true, null, 'x']}),
    '{"a":[true,null,"x"],"b":1}');
  assert.throws(() => canonicalJson({a: undefined}),
    error => error.code === 'HOST_RESOURCE_CANONICAL_INVALID');
  assert.throws(() => canonicalJson({a: () => 1}),
    error => error.code === 'HOST_RESOURCE_CANONICAL_INVALID');
  assert.throws(() => canonicalJson({a: NaN}),
    error => error.code === 'HOST_RESOURCE_CANONICAL_INVALID');
  assert.throws(() => canonicalJson({a: 1n}),
    error => error.code === 'HOST_RESOURCE_CANONICAL_INVALID');
  assert.throws(() => canonicalJson(new Date()),
    error => error.code === 'HOST_RESOURCE_CANONICAL_INVALID');
  const cycle = {};
  cycle.self = cycle;
  assert.throws(() => canonicalJson(cycle),
    error => error.code === 'HOST_RESOURCE_CANONICAL_INVALID');
  const first = buildFrozenResourceSet(makeIdentity());
  const second = buildFrozenResourceSet(makeIdentity({reviewId: 2000000001}));
  assert.notEqual(first.frozenResourceSetSha256, second.frozenResourceSetSha256);
  assert.throws(() => buildFrozenResourceSet({
    ...makeIdentity(),
    exclusiveResources: makeIdentity().exclusiveResources
      .filter(resource => !resource.startsWith('claim-key:'))
  }), error => error.code === 'HOST_RESOURCE_FENCE_IDENTITY_INVALID' &&
    error.details.includes('REQUIRED_RESOURCE_MISSING'));
  const identity = makeIdentity();
  const bad = {...identity, writeSetSha256: sha256('wrong')};
  bad.dispatchId = computeDispatchId({
    repository: identity.repository, issueNumber: 370,
    sourceSha: identity.sourceSha, taskPackBlobSha: identity.taskPackBlobSha,
    reviewRef: identity.reviewRef, ownerApprovalRef: identity.ownerApprovalRef,
    writeSetSha256: bad.writeSetSha256
  });
  assert.throws(() => buildFrozenResourceSet(bad),
    error => error.code === 'HOST_RESOURCE_FENCE_IDENTITY_INVALID' &&
      error.details.includes('WRITE_SET_DIGEST_MISMATCH'));
});

test('first initialization refuses a non-empty store and restart never auto-recovers', async t => {
  const dir = await mkdtemp(join(tmpdir(), 'tally-host-fence-init-'));
  const store = await openSqliteCasStore({path: join(dir, 'fence.sqlite'), busyTimeoutMs: 1000});
  t.after(async () => {
    try { await store.close(); } catch { /* closed */ }
    await rm(dir, {recursive: true, force: true});
  });
  await store.put('tally:v2:foreign:1', {x: 1}, {mustBeAbsent: true});
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const authority = createHostResourceFenceAuthority({
    store, registryId: 'fence-registry-test-1', anchorPath: join(dir, 'anchor.ndjson'),
    writerSid: WRITER_SID, authorityPrivateKey: AUTHORITY_KEY.privateKey,
    authorizedClients: [entry]
  });
  await assert.rejects(authority.initialize(),
    error => error.code === 'HOST_RESOURCE_FENCE_FIRST_INIT_STORE_NOT_EMPTY');
});

test('restart, anchor-only state and double initialization deny', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {dir, store, anchorPath, authority} = await openHarness(t, {clients: [entry]});
  await assert.rejects(authority.initialize(),
    error => error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_ALREADY_INITIALIZED');
  const second = createHostResourceFenceAuthority({
    store, registryId: authority.registryId, anchorPath, writerSid: WRITER_SID,
    authorityPrivateKey: AUTHORITY_KEY.privateKey, authorizedClients: [entry]
  });
  await assert.rejects(second.initialize(),
    error => error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_RESTART_REFUSED');
  const registry = await store.get(authority.registryKey);
  await store.remove(authority.registryKey, {mustMatchRevision: registry.revision});
  const third = createHostResourceFenceAuthority({
    store, registryId: authority.registryId, anchorPath, writerSid: WRITER_SID,
    authorityPrivateKey: AUTHORITY_KEY.privateKey, authorizedClients: [entry]
  });
  await assert.rejects(third.initialize(),
    error => error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_RESTART_REFUSED');
});

test('signed acquire/assert/release stores only the nonce hash', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {store, anchorPath, authority} = await openHarness(t, {clients: [entry]});
  const ownerNonce = nonce();
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a', clientNonce: ownerNonce
  });
  const acquired = await client.acquire();
  assert.match(acquired.fenceId, /^fence-[a-f0-9]{24}$/);
  assert.equal(acquired.fenceRevision, 1);
  assert.equal(acquired.attempts, 1);
  const registry = await store.get(authority.registryKey);
  assert.equal(registry.revision, 1);
  assert.equal(registry.value.generation, 1);
  assert.equal(registry.value.leases.length, 1);
  assert.equal(registry.value.leases[0].nonceHash, sha256(ownerNonce));
  const registryText = JSON.stringify(registry.value);
  assert.equal(registryText.includes(ownerNonce), false);
  assert.equal(readFileSync(anchorPath, 'utf8').includes(ownerNonce), false);
  assert.equal(registryText.includes('PRIVATE KEY'), false);
  assert.equal(JSON.stringify(await client.assert()).includes(acquired.fenceId), true);
  assert.equal(JSON.stringify(client.requestLog).includes(ownerNonce), false);
  const hashReuseClient = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: sha256(ownerNonce), sequenceStart: 2
  });
  await assert.rejects(hashReuseClient.assert(acquired.fenceId),
    error => error.code === 'HOST_RESOURCE_FENCE_NONCE_MISMATCH');
  const otherNonceClient = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce(), sequenceStart: 2
  });
  await assert.rejects(otherNonceClient.release(acquired.fenceId),
    error => error.code === 'HOST_RESOURCE_FENCE_NONCE_MISMATCH');
  const released = await client.release();
  assert.equal(released.releasedFenceId, acquired.fenceId);
  assert.equal(released.releasedFenceRevision, 1);
  await assert.rejects(client.assert(),
    error => error.code === 'HOST_RESOURCE_FENCE_NOT_HELD');
  const reacquired = await client.acquire();
  assert.equal(reacquired.fenceRevision, 3);
});

test('stable fenceRevision, Claim/FSW puts under the lease and key coverage', async t => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const identityB = makeIdentity({issueNumber: 371, files: ['experiments/v2/b-only.mjs']});
  const builtB = buildFrozenResourceSet(identityB);
  const entryA = authorizedEntry(identityA, builtA, CLIENT_KEY_A, 'writer-a');
  const entryB = authorizedEntry(identityB, builtB, CLIENT_KEY_B, 'writer-b');
  const {store, authority} = await openHarness(t, {clients: [entryA, entryB]});
  const nonceA = nonce();
  const clientA = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonceA
  });
  const clientB = clientFor({
    authority, identity: identityB, keyPair: CLIENT_KEY_B, writerId: 'writer-b',
    clientNonce: nonce()
  });
  const acquiredA = await clientA.acquire();
  assert.equal(acquiredA.fenceRevision, 1);
  const acquiredB = await clientB.acquire();
  assert.equal(acquiredB.fenceRevision, 2);
  assert.equal((await clientA.assert()).fenceRevision, 1);
  const claimCasKey = `tally:v2:claim-admission:${sha256(identityA.protectedClaimKey)}`;
  const pending = claimValue(identityA);
  const published = claimValue(identityA, {
    phase: 'PUBLISHED_VERIFIED', commentId: 123,
    commentRef: `${identityA.issueRef}#issuecomment-123`
  });
  const first = await clientA.put(claimCasKey, pending, {mustBeAbsent: true});
  assert.equal(first.revision, 0);
  await assert.rejects(clientA.put(claimCasKey,
    {...pending, terminalEvidence: 'must not enter a pending row'},
    {mustMatchRevision: 0}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID');
  await assert.rejects(clientA.put(claimCasKey,
    {...pending, commentRef: `${identityA.issueRef}#issuecomment-1`},
    {mustMatchRevision: 0}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID');
  await assert.rejects(clientA.put(claimCasKey,
    {...published, protectedClaimKey: identityB.protectedClaimKey},
    {mustMatchRevision: 0}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID');
  await assert.rejects(clientA.put(claimCasKey,
    {...published, secret: 'must not enter CAS'}, {mustMatchRevision: 0}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID');
  const second = await clientA.put(claimCasKey, published, {mustMatchRevision: 0});
  assert.equal(second.revision, 1);
  const fswKey = 'tally:v2:task-writeset:370';
  const fsw = fswValue(identityA, builtA, 'writer-a');
  await assert.rejects(clientA.put(fswKey,
    {...fsw, claimCommentId: 124,
      claimCommentRef: `${identityA.issueRef}#issuecomment-124`},
    {mustBeAbsent: true}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID' &&
    error.details.includes('FSW_CLAIM_COMMENT_BINDING'));
  assert.equal(await store.get(fswKey), null);
  const fswPut = await clientA.put(fswKey, fsw, {mustBeAbsent: true});
  assert.equal(fswPut.revision, 0);
  const terminalizing = claimValue(identityA, {
    phase: 'PENDING_GITHUB_TERMINALIZATION',
    commentId: 123, commentRef: `${identityA.issueRef}#issuecomment-123`,
    terminalReservationRevision: 2, terminalWriterId: 'writer-a',
    terminalEvidence: 'terminal evidence for the composition test',
    terminalOccurredAt: new Date().toISOString()
  });
  await assert.rejects(clientA.put(claimCasKey,
    {...terminalizing, commentId: 124,
      commentRef: `${identityA.issueRef}#issuecomment-124`},
    {mustMatchRevision: 1}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID' &&
    error.details.includes('CLAIM_COMMENT_IMMUTABLE'));
  await assert.rejects(clientA.put(claimCasKey, published, {mustMatchRevision: 1}),
    error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID' &&
      error.details.includes('CLAIM_TRANSITION'));
  const terminalized = await clientA.put(claimCasKey, terminalizing,
    {mustMatchRevision: 1});
  assert.equal(terminalized.revision, 2);
  const terminalVerified = {...terminalizing, phase: 'TERMINAL_VERIFIED',
    terminalCommentId: 456,
    terminalCommentRef: `${identityA.issueRef}#issuecomment-456`};
  await assert.rejects(clientA.put(claimCasKey,
    {...terminalVerified, terminalEvidence: 'changed terminal evidence'},
    {mustMatchRevision: 2}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID' &&
    error.details.some(detail => String(detail).startsWith('CLAIM_TERMINAL_IMMUTABLE')));
  const terminalConfirmed = await clientA.put(claimCasKey, terminalVerified,
    {mustMatchRevision: 2});
  assert.equal(terminalConfirmed.revision, 3);
  await assert.rejects(clientA.put(fswKey,
    {...fsw, frozenResourceSetSha256: '0'.repeat(64)}, {mustMatchRevision: 0}),
  error => error.code === 'HOST_RESOURCE_FENCE_VALUE_INVALID');
  await assert.rejects(clientA.remove(fswKey, {mustMatchRevision: 0}),
    error => error.code === 'HOST_RESOURCE_FENCE_KEY_RESERVED');
  await assert.rejects(clientA.put('tally:v2:not-covered:1', {x: 1}, {mustBeAbsent: true}),
    error => error.code === 'HOST_RESOURCE_FENCE_KEY_NOT_COVERED');
  await assert.rejects(clientA.remove(claimCasKey, {mustMatchRevision: 3}),
    error => error.code === 'HOST_RESOURCE_FENCE_KEY_RESERVED');
  assert.notEqual(await store.get(claimCasKey), null);
  assert.notEqual(await store.get(fswKey), null);
  const registry = await store.get(authority.registryKey);
  assert.equal(registry.value.generation, 2);
  const requestLogText = JSON.stringify(clientA.requestLog);
  assert.equal(requestLogText.includes(nonceA), false);
  assert.equal(requestLogText.includes('PENDING_GITHUB_PUBLISH'), false);
});

test('two concurrent acquisitions of overlapping resources admit exactly one lease', async t => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const identityB = makeIdentity({issueNumber: 371, files: [ADMITTED_FILES[0]]});
  const builtB = buildFrozenResourceSet(identityB);
  const entryA = authorizedEntry(identityA, builtA, CLIENT_KEY_A, 'writer-a');
  const entryB = authorizedEntry(identityB, builtB, CLIENT_KEY_B, 'writer-b');
  const {store, authority} = await openHarness(t, {clients: [entryA, entryB]});
  const clientA = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  const clientB = clientFor({
    authority, identity: identityB, keyPair: CLIENT_KEY_B, writerId: 'writer-b',
    clientNonce: nonce()
  });
  const settled = await Promise.allSettled([clientA.acquire(), clientB.acquire()]);
  const fulfilled = settled.filter(result => result.status === 'fulfilled');
  const rejected = settled.filter(result => result.status === 'rejected');
  assert.equal(fulfilled.length, 1);
  assert.equal(rejected.length, 1);
  assert.equal(rejected[0].reason.code, 'HOST_RESOURCE_FENCE_CONFLICT');
  const registry = await store.get(authority.registryKey);
  assert.equal(registry.value.leases.length, 1);
});

test('bounded CAS retries recover and exhaustion leaves no partial lease', async t => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const identityB = makeIdentity({issueNumber: 371, files: ['experiments/v2/b-only.mjs']});
  const builtB = buildFrozenResourceSet(identityB);
  const entryA = authorizedEntry(identityA, builtA, CLIENT_KEY_A, 'writer-a');
  const entryB = authorizedEntry(identityB, builtB, CLIENT_KEY_B, 'writer-b');
  const dir = await mkdtemp(join(tmpdir(), 'tally-host-fence-retry-'));
  const realStore = await openSqliteCasStore({path: join(dir, 'fence.sqlite'), busyTimeoutMs: 1000});
  const store = wrappingStore(realStore);
  t.after(async () => {
    try { await realStore.close(); } catch { /* closed */ }
    await rm(dir, {recursive: true, force: true});
  });
  const authority = createHostResourceFenceAuthority({
    store, registryId: 'fence-registry-test-1', anchorPath: join(dir, 'anchor.ndjson'),
    writerSid: WRITER_SID, authorityPrivateKey: AUTHORITY_KEY.privateKey,
    authorizedClients: [entryA, entryB]
  });
  await authority.initialize();
  const clientA = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  const clientB = clientFor({
    authority, identity: identityB, keyPair: CLIENT_KEY_B, writerId: 'writer-b',
    clientNonce: nonce()
  });
  store.failNext(1);
  const recovered = await clientA.acquire();
  assert.equal(recovered.attempts, 2);
  store.failNext(16);
  await assert.rejects(clientB.acquire(),
    error => error.code === 'HOST_RESOURCE_FENCE_CONTENTION_EXHAUSTED');
  let registry = await store.get(authority.registryKey);
  assert.equal(registry.value.leases.length, 1);
  assert.equal(registry.value.generation, 1);
  store.failNext(0);
  const acquiredB = await clientB.acquire();
  assert.equal(acquiredB.fenceRevision, 2);
  registry = await store.get(authority.registryKey);
  assert.equal(registry.value.leases.length, 2);
});

test('anchor truncation and signature tampering deny before a signed response', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {anchorPath, authority} = await openHarness(t, {clients: [entry]});
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await client.acquire();
  const lines = readFileSync(anchorPath, 'utf8').split('\n').filter(Boolean);
  writeFileSync(anchorPath, lines.slice(0, -1).join('\n') + '\n');
  await assert.rejects(client.assert(),
    error => error.code === 'HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH');
});

test('anchor signature replacement denies before a signed response', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {anchorPath, authority} = await openHarness(t, {clients: [entry]});
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await client.acquire();
  const lines = readFileSync(anchorPath, 'utf8').split('\n').filter(Boolean);
  const record = JSON.parse(lines[lines.length - 1]);
  record.signature = (record.signature[0] === 'A' ? 'B' : 'A') + record.signature.slice(1);
  lines[lines.length - 1] = JSON.stringify(record);
  writeFileSync(anchorPath, lines.join('\n') + '\n');
  await assert.rejects(client.assert(),
    error => error.code === 'HOST_RESOURCE_FENCE_ANCHOR_SIGNATURE_INVALID');
});

test('registry or foreign-store drift denies before a signed response', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {store, authority} = await openHarness(t, {clients: [entry]});
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await client.acquire();
  const registry = await store.get(authority.registryKey);
  await store.put(authority.registryKey,
    {...deepCopy(registry.value), generation: registry.value.generation + 1},
    {mustMatchRevision: registry.revision});
  await assert.rejects(client.assert(),
    error => error.code === 'HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH');
});

test('unauthorized clients, tampered requests, replay and sequence rollback deny', async t => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const entryA = authorizedEntry(identityA, builtA, CLIENT_KEY_A, 'writer-a');
  const {authority} = await openHarness(t, {clients: [entryA]});
  const unauthorizedIdentity = makeIdentity({issueNumber: 999,
    files: ['experiments/v2/x-only.mjs']});
  const unauthorizedClient = clientFor({
    authority, identity: unauthorizedIdentity, keyPair: CLIENT_KEY_X,
    writerId: 'writer-x', clientNonce: nonce()
  });
  await assert.rejects(unauthorizedClient.acquire(),
    error => error.code === 'HOST_RESOURCE_FENCE_CLIENT_UNAUTHORIZED');
  let captured = null;
  const tamperingSend = async request => {
    if (!captured) {
      captured = deepCopy(request);
      const tampered = deepCopy(request);
      tampered.value.nonceHash = '0'.repeat(64);
      return authority.handle(tampered);
    }
    return authority.handle(request);
  };
  const tamperedClient = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce(), send: tamperingSend
  });
  await assert.rejects(tamperedClient.acquire(),
    error => error.code === 'HOST_RESOURCE_FENCE_REQUEST_SIGNATURE_INVALID');
  let replayCaptured = null;
  const replayClient = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce(),
    send: request => {
      replayCaptured = deepCopy(request);
      return authority.handle(request);
    }
  });
  await replayClient.observe();
  const replayed = await authority.handle(replayCaptured);
  assert.equal(replayed.status, 'DENIED');
  assert.equal(replayed.code, 'HOST_RESOURCE_FENCE_REQUEST_REPLAY');
  const rollbackClient = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await assert.rejects(rollbackClient.observe(),
    error => error.code === 'HOST_RESOURCE_FENCE_REQUEST_SEQUENCE_REPLAY');
});

test('resourceConflictPort proves a held lease and exposes incompatible claims', async t => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const identityB = makeIdentity({issueNumber: 371, files: ['experiments/v2/b-only.mjs']});
  const builtB = buildFrozenResourceSet(identityB);
  const entryA = authorizedEntry(identityA, builtA, CLIENT_KEY_A, 'writer-a');
  const entryB = authorizedEntry(identityB, builtB, CLIENT_KEY_B, 'writer-b');
  const {store, anchorPath, authority} = await openHarness(t, {clients: [entryA, entryB]});
  const clientA = clientFor({
    authority, identity: identityA, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  const clientB = clientFor({
    authority, identity: identityB, keyPair: CLIENT_KEY_B, writerId: 'writer-b',
    clientNonce: nonce()
  });
  const acquiredA = await clientA.acquire();
  await clientB.acquire();
  const registry = await store.get(authority.registryKey);
  const leaseB = registry.value.leases.find(lease =>
    lease.protectedClaimKey === identityB.protectedClaimKey);
  const projection = projectionFor(leaseB);
  const digest = projectionDigest([projection]);
  const port = createHostResourceFencePort({
    store, registryId: authority.registryId, anchorPath,
    authorityPublicKeyPem: authority.publicKeyPem, writerSid: WRITER_SID,
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identityA.protectedClaimKey,
    sourceSha: identityA.sourceSha,
    frozenResourceSetSha256: builtA.frozenResourceSetSha256,
    expectedHead: () => authority.anchorHead,
    withReadLock: task => authority.withReadLock(task)
  });
  assert.equal(port.sourceKind, 'TRUSTED_HOST_RESOURCE_FENCE_PORT');
  assert.equal(port.repository, REPO);
  assert.equal(port.writerId, 'writer-a');
  assert.equal(port.frozenResourceSetSha256, builtA.frozenResourceSetSha256);
  const proof = await port.readHeldFence({
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identityA.protectedClaimKey,
    sourceSha: identityA.sourceSha,
    foreignClaimCount: 1, foreignClaimsSha256: digest,
    foreignClaims: [projection]
  });
  assert.equal(proof.sourceKind, 'TRUSTED_HOST_RESOURCE_FENCE_READBACK');
  assert.equal(proof.complete, true);
  assert.equal(proof.fenceHeld, true);
  assert.equal(proof.fenceScope, 'EXACT_FROZEN_RESOURCE_SET');
  assert.equal(proof.heldThrough, 'CAS_PUBLISH_READBACK');
  assert.equal(proof.repository, REPO);
  assert.equal(proof.writerId, 'writer-a');
  assert.equal(proof.protectedClaimKey, identityA.protectedClaimKey);
  assert.equal(proof.sourceSha, identityA.sourceSha);
  assert.equal(proof.foreignClaimsSha256, digest);
  assert.equal(proof.foreignClaimCount, 1);
  assert.equal(proof.incompatibleResourceCount, 0);
  assert.equal(proof.frozenResourceSetSha256, builtA.frozenResourceSetSha256);
  assert.equal(proof.fenceId, acquiredA.fenceId);
  assert.equal(proof.fenceRevision, acquiredA.fenceRevision);
  assert.match(proof.fenceId, /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/);
  assert.ok(Number.isSafeInteger(proof.fenceRevision) && proof.fenceRevision >= 0);
  const observed = Date.parse(proof.observedAt);
  assert.ok(Number.isFinite(observed) && observed <= Date.now());
  assert.ok(Date.now() - observed <= 15000);
  const missingProjection = {
    protectedClaimKey: claimKey(999), phase: 'PUBLISHED_VERIFIED',
    sourceSha: 'c'.repeat(40), dispatchId: 'dispatch:999:x', writerId: 'writer-999'
  };
  const missingProof = await port.readHeldFence({
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identityA.protectedClaimKey,
    sourceSha: identityA.sourceSha,
    foreignClaimCount: 1,
    foreignClaimsSha256: projectionDigest([missingProjection]),
    foreignClaims: [missingProjection]
  });
  assert.equal(missingProof.incompatibleResourceCount, 1);
  await assert.rejects(port.readHeldFence({
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identityA.protectedClaimKey,
    sourceSha: identityA.sourceSha,
    foreignClaimCount: 1,
    foreignClaimsSha256: digest,
    foreignClaims: [{...projection, dispatchId: 'dispatch:forged'}]
  }), error => error.code === 'HOST_RESOURCE_FENCE_PORT_CONTEXT' &&
    error.details.includes('FOREIGN_DIGEST'));
  await clientA.release();
  await assert.rejects(port.readHeldFence({
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identityA.protectedClaimKey,
    sourceSha: identityA.sourceSha,
    foreignClaimCount: 1, foreignClaimsSha256: digest,
    foreignClaims: [projection]
  }), error => error.code === 'HOST_RESOURCE_FENCE_NOT_HELD');
});

test('registry validator rejects overlapping leases and duplicate fence ids', () => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const identityB = makeIdentity({issueNumber: 371, files: [ADMITTED_FILES[0]]});
  const builtB = buildFrozenResourceSet(identityB);
  const overlapping = {
    schema: 'tally/host-resource-fence-registry/v1',
    registryId: 'fence-registry-test-1',
    generation: 2,
    leases: [
      leaseFor(identityA, builtA, {fenceId: 'fence-a', fenceRevision: 1, writerId: 'writer-a'}),
      leaseFor(identityB, builtB, {fenceId: 'fence-b', fenceRevision: 2, writerId: 'writer-b'})
    ]
  };
  assert.throws(() => validateHostResourceFenceRegistryDocument(overlapping),
    error => error.code === 'HOST_RESOURCE_FENCE_REGISTRY_MALFORMED' &&
      error.details.includes('OVERLAPPING_LEASES'));
  const disjointB = makeIdentity({issueNumber: 371, files: ['experiments/v2/b-only.mjs']});
  const disjointBuiltB = buildFrozenResourceSet(disjointB);
  const duplicate = {
    schema: 'tally/host-resource-fence-registry/v1',
    registryId: 'fence-registry-test-1',
    generation: 2,
    leases: [
      leaseFor(identityA, builtA, {fenceId: 'fence-same', fenceRevision: 1, writerId: 'writer-a'}),
      leaseFor(disjointB, disjointBuiltB,
        {fenceId: 'fence-same', fenceRevision: 2, writerId: 'writer-b'})
    ]
  };
  assert.throws(() => validateHostResourceFenceRegistryDocument(duplicate),
    error => error.code === 'HOST_RESOURCE_FENCE_REGISTRY_MALFORMED' &&
      error.details.includes('DUPLICATE_FENCE_ID'));
});

test('distinct authorized Task keys cannot share a public key', () => {
  const identityA = makeIdentity();
  const builtA = buildFrozenResourceSet(identityA);
  const identityB = makeIdentity({issueNumber: 371, files: ['experiments/v2/b-only.mjs']});
  const builtB = buildFrozenResourceSet(identityB);
  const entryA = authorizedEntry(identityA, builtA, CLIENT_KEY_A, 'writer-a');
  const entryB = authorizedEntry(identityB, builtB, CLIENT_KEY_A, 'writer-b');
  const fakeStore = {
    get: async () => null,
    put: async () => ({revision: 0}),
    remove: async () => undefined,
    list: async () => []
  };
  assert.throws(() => createHostResourceFenceAuthority({
    store: fakeStore, registryId: 'fence-registry-test-1',
    anchorPath: 'C:/tmp/nonexistent-fence-anchor.ndjson', writerSid: WRITER_SID,
    authorityPrivateKey: AUTHORITY_KEY.privateKey, authorizedClients: [entryA, entryB]
  }), error => error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_CONFIG' &&
    error.details.includes('DUPLICATE_CLIENT_PUBLIC_KEY'));
});

test('a forged Authority response is rejected by the client', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {authority} = await openHarness(t, {clients: [entry]});
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce(),
    send: async request => {
      const genuine = await authority.handle(request);
      const forged = {...genuine, result: {fenceId: 'fence-forged'}};
      delete forged.signature;
      forged.signature = sign('sha256', Buffer.from(canonicalJson(forged), 'utf8'),
        FORGED_KEY.privateKey).toString('base64');
      return forged;
    }
  });
  await assert.rejects(client.observe(),
    error => error.code === 'HOST_RESOURCE_FENCE_RESPONSE_SIGNATURE_INVALID');
});

test('signed PUT value/condition substitution denies', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {authority} = await openHarness(t, {clients: [entry]});
  let tampered = false;
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce(),
    send: async request => {
      if (!tampered && request.operation === 'PUT') {
        tampered = true;
        const copy = deepCopy(request);
        copy.value.conditions = {mustMatchRevision: 0};
        return authority.handle(copy);
      }
      return authority.handle(request);
    }
  });
  await client.acquire();
  const claimCasKey = `tally:v2:claim-admission:${sha256(identity.protectedClaimKey)}`;
  await assert.rejects(client.put(claimCasKey, claimValue(identity), {mustBeAbsent: true}),
    error => error.code === 'HOST_RESOURCE_FENCE_REQUEST_SIGNATURE_INVALID');
});

test('a foreign store mutation after CAS is never signed into the anchor', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const dir = await mkdtemp(join(tmpdir(), 'tally-host-fence-race-'));
  const realStore = await openSqliteCasStore({path: join(dir, 'fence.sqlite'), busyTimeoutMs: 1000});
  let armed = false;
  let injected = false;
  const store = {
    get: (...args) => realStore.get(...args),
    list: (...args) => realStore.list(...args),
    remove: (...args) => realStore.remove(...args),
    close: (...args) => realStore.close(...args),
    put: async (key, value, conditions) => {
      const result = await realStore.put(key, value, conditions);
      if (armed && !injected && key.startsWith('tally:v2:host-resource-fence:')) {
        injected = true;
        await realStore.put('tally:v2:foreign:race', {injected: true}, {mustBeAbsent: true});
      }
      return result;
    }
  };
  t.after(async () => {
    try { await realStore.close(); } catch { /* closed */ }
    await rm(dir, {recursive: true, force: true});
  });
  const authority = createHostResourceFenceAuthority({
    store, registryId: 'fence-registry-test-1', anchorPath: join(dir, 'anchor.ndjson'),
    writerSid: WRITER_SID, authorityPrivateKey: AUTHORITY_KEY.privateKey,
    authorizedClients: [entry]
  });
  await authority.initialize();
  armed = true;
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await assert.rejects(client.acquire(),
    error => error.code === 'HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH' &&
      error.details.includes('NON_ANCHOR_MUTATION'));
  await assert.rejects(client.observe(),
    error => error.code === 'HOST_RESOURCE_FENCE_AUTHORITY_FAILED');
});

test('genesis refuses a foreign key inserted during initialization', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const dir = await mkdtemp(join(tmpdir(), 'tally-host-fence-genesis-'));
  const realStore = await openSqliteCasStore({path: join(dir, 'fence.sqlite'), busyTimeoutMs: 1000});
  let injected = false;
  const store = {
    get: (...args) => realStore.get(...args),
    list: (...args) => realStore.list(...args),
    remove: (...args) => realStore.remove(...args),
    close: (...args) => realStore.close(...args),
    put: async (key, value, conditions) => {
      const result = await realStore.put(key, value, conditions);
      if (!injected && key.startsWith('tally:v2:host-resource-fence:') &&
        conditions.mustBeAbsent === true) {
        injected = true;
        await realStore.put('tally:v2:foreign:genesis', {injected: true}, {mustBeAbsent: true});
      }
      return result;
    }
  };
  t.after(async () => {
    try { await realStore.close(); } catch { /* closed */ }
    await rm(dir, {recursive: true, force: true});
  });
  const anchorPath = join(dir, 'anchor.ndjson');
  const authority = createHostResourceFenceAuthority({
    store, registryId: 'fence-registry-test-1', anchorPath,
    writerSid: WRITER_SID, authorityPrivateKey: AUTHORITY_KEY.privateKey,
    authorizedClients: [entry]
  });
  await assert.rejects(authority.initialize(),
    error => error.code === 'HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH' &&
      error.details.includes('GENESIS_NON_ANCHOR_MUTATION'));
  assert.equal(existsSync(anchorPath), false);
  assert.equal(authority.initialized, false);
});

test('integrity failures deny without a signed success response', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const dir = await mkdtemp(join(tmpdir(), 'tally-host-fence-down-'));
  const realStore = await openSqliteCasStore({path: join(dir, 'fence.sqlite'), busyTimeoutMs: 1000});
  let failList = false;
  const store = {
    get: (...args) => realStore.get(...args),
    list: async (...args) => {
      if (failList) {
        const error = new Error('store unavailable');
        error.code = 'HOST_FAULT_INJECTED';
        throw error;
      }
      return realStore.list(...args);
    },
    remove: (...args) => realStore.remove(...args),
    close: (...args) => realStore.close(...args),
    put: (...args) => realStore.put(...args)
  };
  t.after(async () => {
    try { await realStore.close(); } catch { /* closed */ }
    await rm(dir, {recursive: true, force: true});
  });
  const authority = createHostResourceFenceAuthority({
    store, registryId: 'fence-registry-test-1', anchorPath: join(dir, 'anchor.ndjson'),
    writerSid: WRITER_SID, authorityPrivateKey: AUTHORITY_KEY.privateKey,
    authorizedClients: [entry]
  });
  await authority.initialize();
  failList = true;
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await assert.rejects(client.observe(),
    error => error.code === 'HOST_RESOURCE_FENCE_STORE_UNAVAILABLE');
});

test('a jointly restored old anchor/store cannot satisfy the live Authority head', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {store, anchorPath, authority} = await openHarness(t, {clients: [entry]});
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await client.acquire();
  const registryRow = await store.get(authority.registryKey);
  const oldAnchorText = readFileSync(anchorPath, 'utf8');
  await client.release();
  const restoredDir = await mkdtemp(join(tmpdir(), 'tally-host-fence-rollback-'));
  const restoredStore = await openSqliteCasStore({
    path: join(restoredDir, 'fence.sqlite'), busyTimeoutMs: 1000
  });
  const restoredAnchor = join(restoredDir, 'anchor.ndjson');
  t.after(async () => {
    try { await restoredStore.close(); } catch { /* closed */ }
    await rm(restoredDir, {recursive: true, force: true});
  });
  const emptyDocument = {
    schema: 'tally/host-resource-fence-registry/v1',
    registryId: authority.registryId, generation: 0, leases: []
  };
  await restoredStore.put(authority.registryKey, emptyDocument, {mustBeAbsent: true});
  await restoredStore.put(authority.registryKey, registryRow.value, {mustMatchRevision: 0});
  writeFileSync(restoredAnchor, oldAnchorText);
  const port = createHostResourceFencePort({
    store: restoredStore, registryId: authority.registryId, anchorPath: restoredAnchor,
    authorityPublicKeyPem: authority.publicKeyPem, writerSid: WRITER_SID,
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identity.protectedClaimKey, sourceSha: identity.sourceSha,
    frozenResourceSetSha256: built.frozenResourceSetSha256,
    expectedHead: () => authority.anchorHead,
    withReadLock: task => authority.withReadLock(task)
  });
  await assert.rejects(port.readHeldFence({
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identity.protectedClaimKey, sourceSha: identity.sourceSha,
    foreignClaimCount: 0, foreignClaimsSha256: projectionDigest([]),
    foreignClaims: []
  }), error => error.code === 'HOST_RESOURCE_FENCE_ANCHOR_HEAD_MISMATCH' &&
    error.details.includes('EXPECTED_HEAD'));
});

test('port reads are serialized with Authority mutations', async t => {
  const identity = makeIdentity();
  const built = buildFrozenResourceSet(identity);
  const entry = authorizedEntry(identity, built, CLIENT_KEY_A, 'writer-a');
  const {store, anchorPath, authority} = await openHarness(t, {clients: [entry]});
  const client = clientFor({
    authority, identity, keyPair: CLIENT_KEY_A, writerId: 'writer-a',
    clientNonce: nonce()
  });
  await client.acquire();
  let releaseDone = false;
  const port = createHostResourceFencePort({
    store, registryId: authority.registryId, anchorPath,
    authorityPublicKeyPem: authority.publicKeyPem, writerSid: WRITER_SID,
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identity.protectedClaimKey, sourceSha: identity.sourceSha,
    frozenResourceSetSha256: built.frozenResourceSetSha256,
    expectedHead: () => authority.anchorHead,
    withReadLock: task => authority.withReadLock(async () => {
      const result = await task();
      assert.equal(releaseDone, false);
      return result;
    })
  });
  const context = {
    repository: REPO, writerId: 'writer-a',
    protectedClaimKey: identity.protectedClaimKey, sourceSha: identity.sourceSha,
    foreignClaimCount: 0, foreignClaimsSha256: projectionDigest([]),
    foreignClaims: []
  };
  const readPromise = port.readHeldFence(context);
  await new Promise(resolve => setImmediate(resolve));
  const releasePromise = client.release().then(() => { releaseDone = true; });
  const [readResult, releaseResult] = await Promise.allSettled([readPromise, releasePromise]);
  assert.equal(readResult.status, 'fulfilled');
  assert.equal(readResult.value.fenceHeld, true);
  assert.equal(releaseResult.status, 'fulfilled');
  assert.equal(releaseDone, true);
  await assert.rejects(port.readHeldFence(context),
    error => error.code === 'HOST_RESOURCE_FENCE_NOT_HELD');
});
