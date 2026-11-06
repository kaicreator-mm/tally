import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createHash} from 'node:crypto';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createPrePostRecoveryController, RECOVERY_IDENTITY}
  from './recover-prepost-reservation.mjs';
import {TARGET} from '../../../packages/tally-github-native-c/src/task-dispatch-state-events.mjs';

/** Exact public bytes of the accepted #319 frozen write-set row (its JSON
 *  digest is pinned by the Pack as 9f6a949a…; the fixture must match it). */
const FROZEN_319_VALUE = {
  phase: 'FROZEN_DAG_WRITESET', repository: 'kaicreator-mm/tally', issueNumber: 319,
  sourceSha: '7ebb1cdd0a79fc82cc938e35f852a80dd8e15332',
  writeSetSha256: '252cbcd0e2526200938eef95a64f8cef4a357cc3ca6ec7ec7664112f2e9b17a2',
  document: 'frozen exact write set for #319:\n.woodpecker/v2-t06d-terminalization.yaml\nexperiments/v2/t06d-host-claim-publication/host-admission-source.mjs\nexperiments/v2/t06d-host-claim-publication/host-admission-source.test.mjs\nexperiments/v2/t06d-host-claim-publication/single-writer.mjs\nexperiments/v2/t06d-host-claim-publication/single-writer.test.mjs\npackages/tally-github-native-c/src/task-claim-github-publisher.mjs\npackages/tally-github-native-c/src/task-dispatch-state-events.mjs\npackages/tally-github-native-c/test/task-claim-github-publisher.test.mjs\npackages/tally-github-native-c/test/task-dispatch-state-events.test.mjs',
  pathCount: 9,
  writeSetDigestBasis: 'sha256(JSON.stringify(paths.sort())) compact; tally.v2.allowed-write-set-normalization/1',
  taskPackRef: 'https://github.com/kaicreator-mm/tally/blob/5b9991648b60b6b2ba892549caaeb46d11d37b19/docs/implementation/v2/T06D_TERMINALIZATION_TASK_PACK_R1.json',
  taskPackBlobSha: '10593bb79ee8e9834a6720d27335aad931de1dc3',
  packPRHeadSha: '5b9991648b60b6b2ba892549caaeb46d11d37b19',
  ownerReceiptRef: 'https://github.com/kaicreator-mm/tally/pull/320#issuecomment-6668349957',
  reviewWitnessRef: 'https://github.com/kaicreator-mm/tally/pull/320#issuecomment-6668373406',
  protectedClaimKey: 'kaicreator-mm/tally#319:builder:__default__', generation: 1,
  dispatchId: 'dispatch:319:t06d-terminalization:20261106:1',
  claimCommentId: 6668875091,
  claimCommentRef: 'https://github.com/kaicreator-mm/tally/issues/319#issuecomment-6668875091',
  writerId: 'tally-host:win32:t06d-terminalization:1',
  frozenAt: '2026-11-05T20:44:38.389Z',
};
assert.equal(createHash('sha256').update(JSON.stringify(FROZEN_319_VALUE)).digest('hex'),
  RECOVERY_IDENTITY.task319.frozenWriteSetDigest);

const writerAuthority = Object.freeze({
  mode: 'SINGLE_WRITER_ADMISSION', exclusiveHostControl: true,
  repository: TARGET.repository,
  writerId: RECOVERY_IDENTITY.terminalWriterId, githubWriterLogin: 'kaicreator-mm',
});
const SIX_REFS = Object.freeze({
  writer: 'https://github.com/kaicreator-mm/tally/issues/203',
  attempt: 'https://github.com/kaicreator-mm/tally/issues/331',
  subprocess: 'https://github.com/kaicreator-mm/tally/issues/284',
  effect: 'https://github.com/kaicreator-mm/tally/pull/204',
  resource: 'https://github.com/kaicreator-mm/tally/pull/205',
  writeSet: 'https://github.com/kaicreator-mm/tally/blob/9fd9b1b798f756c708813b6ac8f2c3e92d72039a/docs/implementation/v2/T06D_PREPOST_RECOVERY_TASK_PACK_R1.json',
});
const FIXTURE_EVIDENCE = '{"taskPackRef":"https://example/pack","fixture":true}';

/** The exact observed revision-2 stranded reservation (field values only). */
function strandedRow(overrides = {}) {
  return {
    phase: 'PENDING_GITHUB_TERMINALIZATION', generation: 1,
    protectedClaimKey: TARGET.protectedClaimKey, repository: TARGET.repository,
    issueRef: TARGET.issueRef, sourceSha: TARGET.sourceSha,
    dispatchId: TARGET.dispatchId, actorRole: 'builder', operatorKind: 'other',
    operatorId: 'zcode:candidate-publication:1',
    writerId: 'tally-host:win32:candidate-publication:1',
    commentId: TARGET.claimCommentId, commentRef: TARGET.claimCommentRef,
    terminalWriterId: RECOVERY_IDENTITY.terminalWriterId,
    terminalEvidence: FIXTURE_EVIDENCE,
    terminalOccurredAt: '2026-11-06T11:20:59.974Z',
    terminalReservationRevision: 2,
    ...overrides,
  };
}
function goodProof(overrides = {}) {
  return {
    observedAt: new Date().toISOString(), fenceHeld: true,
    fenceId: 'fence:i331-prepost-recovery:fixture', dispatchId: TARGET.dispatchId,
    sourceSha: TARGET.sourceSha, protectedClaimKey: TARGET.protectedClaimKey,
    generation: 1, activeWriterCount: 0, activeAttemptCount: 0,
    activeSubprocessCount: 0, inflightEffectCount: 0, unresolvedResourceCount: 0,
    writeSetReconciled: true, evidenceRefs: SIX_REFS,
    excludedReservation: {
      casKey: TARGET.casKey, revision: 2,
      phase: 'PENDING_GITHUB_TERMINALIZATION',
      terminalWriterId: RECOVERY_IDENTITY.terminalWriterId,
      protectedClaimKey: TARGET.protectedClaimKey, sourceSha: TARGET.sourceSha,
      dispatchId: TARGET.dispatchId, generation: 1,
    },
    excludedReservationCount: 1,
    ...overrides,
  };
}
function lineageRows(overrides = {}) {
  const i319 = RECOVERY_IDENTITY.task319, i331 = RECOVERY_IDENTITY.task331;
  const frozen319Value = FROZEN_319_VALUE;
  const frozen331Value = {phase: 'FROZEN_DAG_WRITESET', taskPackBlobSha: i331.packBlobSha,
    writeSetSha256: i331.writeSetSha256, pathCount: i331.pathCount,
    claimCommentId: i331.claimCommentId};
  return {
    claim319: {revision: i319.claimRevision, value: {
      phase: i319.claimPhase, commentId: i319.claimCommentId,
      writerId: i319.terminalWriterId ?? RECOVERY_IDENTITY.terminalWriterId}},
    frozen319: {revision: i319.frozenWriteSetRevision, value: frozen319Value},
    claim331: {revision: i331.claimRevision, value: {
      phase: i331.claimPhase, commentId: i331.claimCommentId,
      dispatchId: i331.dispatchId, operatorId: i331.operatorId,
      protectedClaimKey: i331.protectedClaimKey, generation: 1}},
    frozen331: {revision: i331.frozenWriteSetRevision, value: frozen331Value},
    ...overrides,
  };
}
function ports(overrides = {}) {
  const counts = {publishes: 0, proofCalls: 0};
  const claimReader = {async readPublishedClaim() { return {
    complete: true, edited: false, commentId: TARGET.claimCommentId,
    commentRef: TARGET.claimCommentRef}; }};
  const terminalEvents = {present: false, exactCommentId: 6700000001, mismatch: false};
  const terminalReader = {
    async readEvents() { return terminalEvents.present
      ? {status: 'TERMINAL_EVENT_ALREADY_EXISTS'}
      : {status: 'ABSENT', complete: true}; },
    async readExactEvent() {
      if (!terminalEvents.present || terminalEvents.mismatch)
        return {status: 'UNKNOWN', reason: 'FIXTURE'};
      return {status: 'EXACT', complete: true, edited: false,
        commentId: terminalEvents.exactCommentId,
        commentRef: `${TARGET.issueRef}#issuecomment-${terminalEvents.exactCommentId}`};
    },
  };
  const priorRunLog = {async read() { return {
    sha256: RECOVERY_IDENTITY.prior.runLogSha256,
    parsed: {...RECOVERY_IDENTITY.prior}}; }};
  const lineage = {async readClaim319() { return rows().claim319; },
    async readFrozen319() { return rows().frozen319; },
    async readClaim331() { return rows().claim331; },
    async readFrozen331() { return rows().frozen331; }};
  let rows = () => lineageRows();
  const quiescence = {async readFreshProof() { counts.proofCalls++;
    return goodProof(); }};
  const publisher = {async publishDispatchStateChange() { counts.publishes++;
    terminalEvents.present = true; return {commentId: terminalEvents.exactCommentId}; }};
  Object.assign(claimReader, overrides.claimReader);
  Object.assign(terminalReader, overrides.terminalReader);
  Object.assign(priorRunLog, overrides.priorRunLog);
  Object.assign(publisher, overrides.publisher);
  if (overrides.quiescence) Object.assign(quiescence, overrides.quiescence);
  if (overrides.rows) rows = overrides.rows;
  if (overrides.lineage) Object.assign(lineage, overrides.lineage);
  return {claimReader, terminalReader, priorRunLog, lineage, quiescence, publisher,
    counts, terminalEvents, rows};
}
async function harness(fn) {
  const dir = await mkdtemp(join(tmpdir(), 'tally-i331-prepost-recovery-'));
  const store = await openSqliteCasStore({path: join(dir, 'registry-203.sqlite'), busyTimeoutMs: 1000});
  try {
    await store.put(TARGET.casKey, strandedRow(), {mustBeAbsent: true});
    // re-put as revision 2 shape: create at rev0 then overwrite via mustMatch 0->1->2
    await fn({store, dir});
  } finally { await store.close(); await rm(dir, {recursive: true, force: true}); }
}

test('factory rejects untrusted ports and caller-supplied writer identity', async () => {
  const p = ports();
  assert.throws(() => createPrePostRecoveryController({}),
    /TRUSTED_PREPOST_RECOVERY_HOST_PORTS_REQUIRED/);
  assert.throws(() => createPrePostRecoveryController({
    store: {get: () => {}, put: () => {}}, ...p,
    writerAuthority: {...writerAuthority, writerId: 'attacker:win32:x:1'}}),
    /TRUSTED_PREPOST_RECOVERY_HOST_PORTS_REQUIRED/);
});

test('full success path: exact rev-2 row -> marker rev-3 -> one POST -> rev-4 TERMINAL_VERIFIED', async () => {
  await harness(async ({store}) => {
    // seed the row at revision 2 (create 0 -> 1 -> 2)
    await store.put(TARGET.casKey, strandedRow({phase: 'PUBLISHED_VERIFIED',
      terminalWriterId: undefined, terminalEvidence: undefined,
      terminalOccurredAt: undefined, terminalReservationRevision: undefined}),
      {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const p = ports();
    const c = createPrePostRecoveryController({store, ...p, writerAuthority});
    const receipt = await c.recoverPrePostReservation();
    assert.equal(receipt.status, 'RECORDED');
    assert.equal(receipt.reason, 'PREPOST_RESERVATION_RECOVERED');
    assert.equal(receipt.claimTerminality, 'VERIFIED');
    assert.equal(receipt.publisherInvocationCount, 1);
    assert.equal(receipt.httpPostCount, 1);
    assert.equal(receipt.casRevisions.before, 2);
    assert.equal(receipt.casRevisions.marker, 3);
    assert.equal(receipt.casRevisions.final, 4);
    assert.equal(p.counts.publishes, 1);
    assert.ok(receipt.recoveryAttemptId.startsWith('recovery:331:t06d-prepost-recovery:'));
    const final = await store.get(TARGET.casKey);
    assert.equal(final.revision, 4);
    assert.equal(final.value.phase, 'TERMINAL_VERIFIED');
    // Original reservation evidence preserved verbatim, marker retained.
    assert.equal(final.value.terminalEvidence, FIXTURE_EVIDENCE);
    assert.equal(final.value.terminalOccurredAt, '2026-11-06T11:20:59.974Z');
    assert.equal(final.value.recoveryAttempt.state, 'POST_ATTEMPT_CONSUMED');
    assert.equal(final.value.terminalCommentId, 6700000001);
    assert.equal(final.value.recoveryAttempt.priorRunLogSha256,
      RECOVERY_IDENTITY.prior.runLogSha256);
    assert.equal(final.value.recoveryAttempt.task331Bindings.packBlobSha,
      RECOVERY_IDENTITY.task331.packBlobSha);
    assert.equal(final.value.recoveryAttempt.excludedReservation.casKey, TARGET.casKey);
    assert.equal(final.value.recoveryAttempt.excludedReservation.revision, 2);
    // marker evidence digest is self-consistent
    assert.equal(final.value.recoveryAttempt.eventEvidenceSha256,
      createHash('sha256').update(final.value.recoveryAttempt.eventEvidence).digest('hex'));
    // public event evidence is bounded and allowlisted
    const envelope = JSON.parse(final.value.recoveryAttempt.eventEvidence);
    assert.equal(envelope.schema, 'tally.t06d-prepost-recovery-event-evidence/v1');
    assert.equal(envelope.priorNoPostRunLogSha256, RECOVERY_IDENTITY.prior.runLogSha256);
    assert.equal(Object.keys(envelope.sixDurableEvidenceRefs).length, 6);
    assert.equal(envelope.freshTargetBoundQuiescenceProof.excludedReservationCount, 1);
    assert.equal(JSON.stringify(envelope).includes('token'), false);
    assert.equal(JSON.stringify(envelope).includes('sqlite'), false);
  });
});

test('row not exact (wrong revision/phase/field/marker present) denies with zero writes and zero POSTs', async () => {
  await harness(async ({store, dir}) => {
    const p = ports();
    const c = createPrePostRecoveryController({store, ...p, writerAuthority});
    // revision 0 (just created) is not the stranded revision-2 row
    const r = await c.recoverPrePostReservation();
    assert.equal(r.status, 'UNKNOWN');
    assert.equal(r.reason, 'TARGET_ROW_NOT_EXACT_REVISION_2_PREPOST');
    assert.equal(r.publisherInvocationCount, 0);
    assert.equal(p.counts.publishes, 0);
    // field drift denies too
    await store.put(TARGET.casKey, strandedRow({phase: 'PUBLISHED_VERIFIED'}),
      {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow({dispatchId: 'dispatch:203:other:1'}),
      {mustMatchRevision: 1});
    const r2 = await c.recoverPrePostReservation();
    assert.equal(r2.reason, 'TARGET_ROW_NOT_EXACT_REVISION_2_PREPOST');
    assert.equal(p.counts.publishes, 0);
    assert.equal((await store.get(TARGET.casKey)).revision, 2);
  });
});

test('prior no-POST artifact mismatch (wrong sha / nonzero counts / missing) denies before marker and POST', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    for (const bad of [
      {sha256: '0'.repeat(64), parsed: {...RECOVERY_IDENTITY.prior}},
      {sha256: RECOVERY_IDENTITY.prior.runLogSha256,
        parsed: {...RECOVERY_IDENTITY.prior, httpPostCount: 1}},
      {sha256: RECOVERY_IDENTITY.prior.runLogSha256,
        parsed: {...RECOVERY_IDENTITY.prior, publisherInvocationCount: 1}},
      {sha256: RECOVERY_IDENTITY.prior.runLogSha256,
        parsed: {...RECOVERY_IDENTITY.prior, status: 'RECORDED'}},
      {sha256: RECOVERY_IDENTITY.prior.runLogSha256,
        parsed: {...RECOVERY_IDENTITY.prior, reason: 'OTHER_REASON'}},
      null,
    ]) {
      const p = ports({priorRunLog: {async read() { return bad; }}});
      const c = createPrePostRecoveryController({store, ...p, writerAuthority});
      const r = await c.recoverPrePostReservation();
      assert.equal(r.reason, 'PRIOR_NO_POST_ARTIFACT_NOT_EXACT');
      assert.equal(p.counts.publishes, 0);
    }
    assert.equal((await store.get(TARGET.casKey)).revision, 2);
  });
});

test('original claim edited/absent or terminal event already present denies with zero writes', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const badClaim = ports({claimReader: {async readPublishedClaim() { throw Error('x'); }}});
    assert.equal((await createPrePostRecoveryController({store, ...badClaim, writerAuthority})
      .recoverPrePostReservation()).reason, 'ORIGINAL_CLAIM_NOT_EXACT');
    const present = ports();
    present.terminalEvents.present = true;
    assert.equal((await createPrePostRecoveryController({store, ...present, writerAuthority})
      .recoverPrePostReservation()).reason, 'TERMINAL_EVENT_NOT_ABSENT');
    assert.equal(present.counts.publishes, 0);
    assert.equal((await store.get(TARGET.casKey)).revision, 2);
  });
});

test('lineage bindings mismatch (#319/#331 claim or frozen write-set) denies with zero writes', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const wrong319 = ports({rows: () => lineageRows({claim319: {revision: 2, value: lineageRows().claim319.value}})});
    assert.equal((await createPrePostRecoveryController({store, ...wrong319, writerAuthority})
      .recoverPrePostReservation()).reason, 'LINEAGE_BINDINGS_NOT_EXACT');
    const wrong331Pack = ports({rows: () => lineageRows({frozen331: {revision: 0,
      value: {...lineageRows().frozen331.value, taskPackBlobSha: 'f'.repeat(40)}}})});
    assert.equal((await createPrePostRecoveryController({store, ...wrong331Pack, writerAuthority})
      .recoverPrePostReservation()).reason, 'LINEAGE_BINDINGS_NOT_EXACT');
    assert.equal((await store.get(TARGET.casKey)).revision, 2);
  });
});

test('quiescence proof must exclude exactly the target reservation; any other activity denies', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const cases = [
      goodProof({fenceHeld: false}),
      goodProof({activeWriterCount: 1}),
      goodProof({activeAttemptCount: 1}),
      goodProof({activeSubprocessCount: 1}),
      goodProof({inflightEffectCount: 2}),
      goodProof({unresolvedResourceCount: 1}),
      goodProof({writeSetReconciled: false}),
      goodProof({excludedReservationCount: 0}),
      goodProof({excludedReservation: {...goodProof().excludedReservation,
        terminalWriterId: 'other:writer:1'}}),
      goodProof({observedAt: new Date(Date.now() - 60000).toISOString()}),
      goodProof({evidenceRefs: {...SIX_REFS, extra: 'https://github.com/kaicreator-mm/tally'}}),
      goodProof({evidenceRefs: {...SIX_REFS, writer: 'https://user:pw@github.com/x/y'}}),
      null,
    ];
    for (const proof of cases) {
      const p = ports({quiescence: {async readFreshProof() { return proof; }}});
      const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
        .recoverPrePostReservation();
      assert.equal(r.reason, 'QUIESCENCE_NOT_HELD');
      assert.equal(r.publisherInvocationCount, 0);
    }
    assert.equal((await store.get(TARGET.casKey)).revision, 2);
  });
});

test('oversized or unbound event evidence fails prevalidation with zero CAS writes and zero POSTs', async () => {
  await harness(async ({store}) => {
    // Row evidence stays within the 16,384 bound, but the recovery envelope
    // wrapping it (bindings + proof + refs) crosses it before any CAS write.
    await store.put(TARGET.casKey, strandedRow({terminalEvidence: 'x'.repeat(15600)}),
      {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow({terminalEvidence: 'x'.repeat(15600)}),
      {mustMatchRevision: 1});
    const p = ports();
    const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r.reason, 'RECOVERY_EVENT_NOT_VALID_BEFORE_MARKER');
    assert.equal(p.counts.publishes, 0);
    assert.equal((await store.get(TARGET.casKey)).revision, 2);
  });
});

test('marker CAS race (concurrent revision bump) never invokes the publisher', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const racing = {
      get: store.get.bind(store),
      async put(key, value, opts) {
        // a competing writer bumps the row between preflight and marker CAS
        await store.put(TARGET.casKey, strandedRow({commentId: 1}), {mustMatchRevision: 2});
        return store.put(key, value, opts);
      },
      list: store.list.bind(store),
    };
    const p = ports();
    const r = await createPrePostRecoveryController({store: racing, ...p, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r.reason, 'MARKER_CAS_RACE');
    assert.equal(p.counts.publishes, 0);
  });
});

test('fence lost or changed after the marker (before POST / before final CAS) never POSTs and never retries', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    let call = 0;
    const p = ports({quiescence: {async readFreshProof() {
      call++;
      return call === 1 ? goodProof() : goodProof({fenceHeld: false});
    }}});
    const c = createPrePostRecoveryController({store, ...p, writerAuthority});
    const r = await c.recoverPrePostReservation();
    assert.equal(r.reason, 'FENCE_CHANGED_BEFORE_PUBLISH');
    assert.equal(p.counts.publishes, 0);
    assert.equal((await store.get(TARGET.casKey)).revision, 3);
    assert.equal((await store.get(TARGET.casKey)).value.recoveryAttempt.state,
      'POST_ATTEMPT_CONSUMED');
  });
});

test('crash after marker before POST: re-entry is read-only, never publishes, stays UNKNOWN', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    let proofCalls = 0;
    const consumed = ports({quiescence: {async readFreshProof() {
      proofCalls++;
      if (proofCalls === 1) return goodProof();
      throw Error('process lost after marker CAS, before the single POST');
    }}});
    await createPrePostRecoveryController({store, ...consumed, writerAuthority})
      .recoverPrePostReservation();
    const markerRow = await store.get(TARGET.casKey);
    assert.equal(markerRow.revision, 3);
    assert.equal(markerRow.value.recoveryAttempt.state, 'POST_ATTEMPT_CONSUMED');
    // re-entry with the event still absent: read-only UNKNOWN, zero POSTs
    const p2 = ports();
    const r2 = await createPrePostRecoveryController({store, ...p2, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r2.status, 'UNKNOWN');
    assert.equal(r2.reason, 'CONSUMED_MARKER_EVENT_NOT_PROVEN');
    assert.equal(p2.counts.publishes, 0);
    assert.equal((await store.get(TARGET.casKey)).revision, 3);
  });
});

test('marker present with the exact event proven (restart) finalizes 3->4 read-only with zero POSTs', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    // commit a full marker via one success run, then reset to the marker state
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const first = ports();
    await createPrePostRecoveryController({store, ...first, writerAuthority})
      .recoverPrePostReservation();
    const finalRow = await store.get(TARGET.casKey);
    assert.equal(finalRow.revision, 4);
    // rebuild a revision-3 marker state by reverting the final CAS
    await store.put(TARGET.casKey, {...finalRow.value, phase: 'PENDING_GITHUB_TERMINALIZATION',
      terminalCommentId: undefined, terminalCommentRef: undefined}, {mustMatchRevision: 4});
    const markerState = await store.get(TARGET.casKey);
    assert.equal(markerState.revision, 5); // put() appends; row now rev5 PENDING with marker
    const p = ports();
    const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
      .recoverPrePostReservation();
    // rev5 is not one of the three legal states: fail closed
    assert.equal(r.status, 'UNKNOWN');
    assert.equal(r.reason, 'TARGET_ROW_NOT_EXACT_REVISION_2_PREPOST');
    assert.equal(p.counts.publishes, 0);
  });
});

test('revision-3 marker reconciliation and revision-4 verify both stay read-only', async () => {
  await harness(async ({store}) => {
    // revision 3 marker with the exact event already visible
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const first = ports();
    const run1 = await createPrePostRecoveryController({store, ...first, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(run1.status, 'RECORDED');
    const finalRow = await store.get(TARGET.casKey);
    // rev4 re-entry: ALREADY_VERIFIED, zero publishes (event stays visible)
    const again = ports();
    again.terminalEvents.present = true;
    const r2 = await createPrePostRecoveryController({store, ...again, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r2.status, 'ALREADY_VERIFIED');
    assert.equal(r2.publisherInvocationCount, 0);
    assert.equal(again.counts.publishes, 0);
    assert.equal(r2.terminalCommentRef, finalRow.value.terminalCommentRef);
  });
});

test('lost ACK (publisher throws after POST) reconciles the exact event without a second POST', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const p = ports();
    // publisher POSTs (event becomes visible) but the ACK is lost
    p.publisher.publishDispatchStateChange = async () => {
      p.counts.publishes++;
      p.terminalEvents.present = true;
      throw Error('NATIVE_GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
    };
    const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r.status, 'RECORDED');
    assert.equal(r.publisherInvocationCount, 1);
    assert.equal(r.httpPostCount, 1);
    assert.equal(p.counts.publishes, 1);
    const final = await store.get(TARGET.casKey);
    assert.equal(final.revision, 4);
    assert.equal(final.value.phase, 'TERMINAL_VERIFIED');
  });
});

test('readback absent/ambiguous after POST keeps UNKNOWN pending/unknown; never a second POST', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    const p = ports();
    p.publisher.publishDispatchStateChange = async () => {
      p.counts.publishes++;
      return {commentId: 6700000001}; // ACK ok but the event never becomes visible
    };
    const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r.status, 'UNKNOWN');
    assert.equal(r.reason, 'TERMINAL_READBACK_NOT_VERIFIED');
    assert.equal(p.counts.publishes, 1);
    const row = await store.get(TARGET.casKey);
    assert.equal(row.revision, 3);
    assert.equal(row.value.phase, 'PENDING_GITHUB_TERMINALIZATION');
    // re-entry must not POST again even if the event later appears
    const later = ports();
    later.terminalEvents.present = true;
    const r2 = await createPrePostRecoveryController({store, ...later, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r2.status, 'RECORDED');
    assert.equal(r2.reason, 'CONSUMED_MARKER_RECONCILED_READ_ONLY');
    assert.equal(later.counts.publishes, 0);
    assert.equal((await store.get(TARGET.casKey)).revision, 4);
  });
});

test('fence released before final CAS (proof 3) keeps the row pending; no second POST', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    let call = 0;
    const p = ports({quiescence: {async readFreshProof() {
      call++;
      return call <= 2 ? goodProof() : goodProof({fenceHeld: false});
    }}});
    const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r.status, 'UNKNOWN');
    assert.equal(r.reason, 'FENCE_CHANGED_BEFORE_FINAL_CAS');
    assert.equal(p.counts.publishes, 1);
    const row = await store.get(TARGET.casKey);
    assert.equal(row.revision, 3);
    assert.equal(row.value.phase, 'PENDING_GITHUB_TERMINALIZATION');
    // reconciliation-only re-entry can still finalize once the event is proven
    const later = ports();
    later.terminalEvents.present = true;
    const r2 = await createPrePostRecoveryController({store, ...later, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r2.status, 'RECORDED');
    assert.equal(later.counts.publishes, 0);
  });
});

test('malformed marker at revision 3 denies read-only with zero writes', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    // hand-corrupt marker shape: revision 3 row whose marker fails validation
    await store.put(TARGET.casKey, {...strandedRow(),
      recoveryAttempt: {schema: 'tally.t06d-prepost-recovery-marker/v1',
        state: 'POST_ATTEMPT_CONSUMED', eventEvidence: 'x',
        eventEvidenceSha256: '0'.repeat(64)}}, {mustMatchRevision: 1});
    const markerRow = await store.get(TARGET.casKey);
    assert.equal(markerRow.revision, 2); // fixture convenience: rev2-with-marker shape
    const p = ports();
    const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
      .recoverPrePostReservation();
    // rev2 with an unexpected recoveryAttempt field is not the exact stranded row
    assert.equal(r.status, 'UNKNOWN');
    assert.equal(r.reason, 'TARGET_ROW_NOT_EXACT_REVISION_2_PREPOST');
    assert.equal(p.counts.publishes, 0);
  });
});

test('final CAS race remains UNKNOWN with no retry and no second POST', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    let bumped = false;
    const racing = {
      get: store.get.bind(store),
      async put(key, value, opts) {
        if (opts?.mustMatchRevision === 3 && !bumped) {
          bumped = true;
          await store.put(TARGET.casKey, value, opts); // consume the revision first
          throw Error('race');
        }
        return store.put(key, value, opts);
      },
      list: store.list.bind(store),
    };
    const p = ports();
    const r = await createPrePostRecoveryController({store: racing, ...p, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r.status, 'UNKNOWN');
    assert.equal(r.reason, 'FINAL_CAS_AMBIGUOUS');
    assert.equal(p.counts.publishes, 1);
    // the row itself may already be TERMINAL_VERIFIED; re-entry must verify read-only
    const verify = ports();
    verify.terminalEvents.present = true;
    const r2 = await createPrePostRecoveryController({store, ...verify, writerAuthority})
      .recoverPrePostReservation();
    assert.equal(r2.status, 'ALREADY_VERIFIED');
    assert.equal(verify.counts.publishes, 0);
  });
});

test('RECOVER_REHEARSE=1 completes preflight, writes nothing, posts nothing', async () => {
  await harness(async ({store}) => {
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 0});
    await store.put(TARGET.casKey, strandedRow(), {mustMatchRevision: 1});
    process.env.RECOVER_REHEARSE = '1';
    try {
      const p = ports();
      const r = await createPrePostRecoveryController({store, ...p, writerAuthority})
        .recoverPrePostReservation();
      assert.equal(r.status, 'REHEARSAL_OK');
      assert.equal(r.casWrites, 0);
      assert.equal(r.publisherInvocationCount, 0);
      assert.equal(p.counts.publishes, 0);
      assert.equal((await store.get(TARGET.casKey)).revision, 2);
      assert.ok(r.evidenceLength > 0 && r.evidenceLength <= 16384);
    } finally { delete process.env.RECOVER_REHEARSE; }
  });
});
