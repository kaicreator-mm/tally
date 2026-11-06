/**
 * T06d Pre-POST Reservation Recovery (#331) — one single-use Host recovery
 * operation for the exact #203 revision-2 `PENDING_GITHUB_TERMINALIZATION`
 * reservation stranded before its first POST by TERMINAL_PREFLIGHT_CHANGED.
 *
 * Not a generic retry facility, terminal-state API, scheduler, Claim store
 * or event protocol. All trust ports come from a selected/privileged Host:
 * the public DomainHarness v0.10.3 CAS store, the existing reviewed native
 * reader/publisher boundaries from PR #330 and this Host's own process-table
 * quiescence measurements. Callers must NEVER be able to supply authority
 * booleans, writer identity or quiescence facts through a request.
 *
 * Protocol (Pack R1 `recoveryContract`):
 *  revision 2 + no marker  -> full preflight, fresh target-bound quiescence
 *                             proof excluding ONLY this exact reservation,
 *                             prevalidated deterministic event, CAS 2->3
 *                             `POST_ATTEMPT_CONSUMED` marker, exactly one
 *                             publisher invocation, native readback, CAS
 *                             3->4 TERMINAL_VERIFIED, final readback.
 *  revision 3 + marker     -> read-only reconciliation only; finalize 3->4
 *                             solely when the exact event is proven present.
 *                             Never publishes (the attempt is consumed).
 *  revision 4              -> read-only verify; never publishes.
 *  anything else           -> UNKNOWN, zero writes, zero POSTs.
 */
import {createHash} from 'node:crypto';
import {execSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import {dirname, resolve} from 'node:path';
import {pathToFileURL} from 'node:url';
import {Octokit} from '@octokit/rest';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {TARGET, readDispatchStateEventsFromClient} from '../../../packages/tally-github-native-c/src/task-dispatch-state-events.mjs';
import {terminalEventForReservation, createHostBoundNativeGitHubClaimPublisher} from '../../../packages/tally-github-native-c/src/task-claim-github-publisher.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from '../../../packages/tally-github-native-c/src/task-claim-host-readback-port.mjs';

const sha256hex = x => createHash('sha256').update(x, 'utf8').digest('hex');
const ID = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const REF_KEYS = ['writer', 'attempt', 'subprocess', 'effect', 'resource', 'writeSet'];
const MARKER_SCHEMA = 'tally.t06d-prepost-recovery-marker/v1';
const ENVELOPE_SCHEMA = 'tally.t06d-prepost-recovery-event-evidence/v1';
const EVIDENCE_BOUND = 16384;

/** Exact immutable identity of the recovery task and its lineage (Pack R1). */
export const RECOVERY_IDENTITY = Object.freeze({
  target203: TARGET,
  terminalWriterId: 'tally-host:win32:t06d-terminalization:1',
  prior: Object.freeze({
    runLogSha256: '473fcf7ae0d7a7359a910e9761ea1fde3556ff91d5c637b4a3f78b59fa91de1c',
    status: 'UNKNOWN',
    reason: 'TERMINAL_PREFLIGHT_CHANGED',
    publisherInvocationCount: 0,
    httpPostCount: 0,
  }),
  task319: Object.freeze({
    claimCasKey: 'tally:v2:claim-admission:471dc65c8c0ef32ac912306d962581d6db92f63af2e0bf88000d5a5370daa50d',
    claimRevision: 1,
    claimPhase: 'PUBLISHED_VERIFIED',
    claimCommentId: 6668875091,
    frozenWriteSetKey: 'tally:v2:task-writeset:319',
    frozenWriteSetRevision: 0,
    frozenWriteSetDigest: '9f6a949a889d1a705116e2f4d8cc91f1577539ef8d1606b41bd65f9592cf0b5b',
    packBlobSha: '10593bb79ee8e9834a6720d27335aad931de1dc3',
  }),
  task331: Object.freeze({
    claimCasKey: 'tally:v2:claim-admission:cc51046823618be93073551e07e7e4ec3f6cc5549e03a0b8f6b7ecec3c6ce251',
    claimRevision: 1,
    claimPhase: 'PUBLISHED_VERIFIED',
    claimCommentId: 6690955003,
    dispatchId: 'dispatch:331:t06d-prepost-recovery:20261106:1',
    operatorId: 'zcode:i331:t06d-prepost-recovery:1',
    protectedClaimKey: 'kaicreator-mm/tally#331:builder:__default__',
    frozenWriteSetKey: 'tally:v2:task-writeset:331',
    frozenWriteSetRevision: 0,
    writeSetSha256: '625ffefa802490301b839ad404ecdc197f647fb94ea6341e24b5e74b2c37cd8a',
    pathCount: 3,
    packBlobSha: '1f0ded0ba46f279d70855e5c00f51d333593a7ca',
    packPRHeadSha: '9fd9b1b798f756c708813b6ac8f2c3e92d72039a',
    ownerReceiptComment: 6685952738,
    ownerAmendmentComment: 6689612741,
  }),
});

const unknown = (reason, extra = {}) => Object.freeze({
  status: 'UNKNOWN', reason, claimTerminality: 'UNKNOWN',
  publisherInvocationCount: 0, httpPostCount: 0,
  authorized: false, admitted: false, taskPass: false, releasePass: false,
  ...extra,
});

/** Durable public evidence reference: exact https URL, no userinfo/query/fragment. */
function reference(x) {
  if (typeof x !== 'string' || x.length === 0 || x.length > 2048) return false;
  let url;
  try { url = new URL(x); } catch { return false; }
  return url.protocol === 'https:' && url.href === x &&
    url.username === '' && url.password === '' && url.search === '' && url.hash === '';
}

/** Every original #203 Claim/reservation field of the revision-2 row, exact. */
function exactPendingRow(v) {
  return v && typeof v === 'object' && !Array.isArray(v) &&
    v.phase === 'PENDING_GITHUB_TERMINALIZATION' &&
    v.repository === TARGET.repository && v.issueRef === TARGET.issueRef &&
    v.sourceSha === TARGET.sourceSha && v.dispatchId === TARGET.dispatchId &&
    v.protectedClaimKey === TARGET.protectedClaimKey && v.generation === 1 &&
    v.actorRole === 'builder' && v.operatorKind === 'other' &&
    v.operatorId === 'zcode:candidate-publication:1' &&
    v.writerId === 'tally-host:win32:candidate-publication:1' &&
    v.commentId === TARGET.claimCommentId && v.commentRef === TARGET.claimCommentRef &&
    v.terminalWriterId === RECOVERY_IDENTITY.terminalWriterId &&
    v.terminalReservationRevision === 2 &&
    v.terminalOccurredAt === '2026-11-06T11:20:59.974Z' &&
    typeof v.terminalEvidence === 'string' && v.terminalEvidence.length > 0 &&
    v.terminalEvidence.length <= EVIDENCE_BOUND &&
    Number.isFinite(Date.parse(v.terminalOccurredAt)) &&
    v.recoveryAttempt === undefined;
}

/** Fresh, target-bound Host quiescence proof that excludes ONLY the exact
 *  revision-2 reservation; every other count must be zero. Fail-closed. */
function qualifiedProof(proof, expectedFenceId = null) {
  const t = Date.parse(proof?.observedAt);
  if (!Number.isFinite(t) || t > Date.now() || Date.now() - t > 30000) return null;
  if (proof.fenceHeld !== true || !ID.test(proof.fenceId ?? '')) return null;
  if (expectedFenceId !== null && proof.fenceId !== expectedFenceId) return null;
  if (proof.dispatchId !== TARGET.dispatchId || proof.sourceSha !== TARGET.sourceSha ||
    proof.protectedClaimKey !== TARGET.protectedClaimKey || proof.generation !== 1) return null;
  if (proof.activeWriterCount !== 0 || proof.activeAttemptCount !== 0 ||
    proof.activeSubprocessCount !== 0 || proof.inflightEffectCount !== 0 ||
    proof.unresolvedResourceCount !== 0 || proof.writeSetReconciled !== true) return null;
  const refs = proof.evidenceRefs;
  if (!refs || typeof refs !== 'object' || Array.isArray(refs) ||
    Object.keys(refs).length !== REF_KEYS.length ||
    !REF_KEYS.every(k => reference(refs[k]))) return null;
  const ex = proof.excludedReservation;
  if (proof.excludedReservationCount !== 1 || !ex || typeof ex !== 'object' ||
    ex.casKey !== TARGET.casKey || ex.revision !== 2 ||
    ex.phase !== 'PENDING_GITHUB_TERMINALIZATION' ||
    ex.terminalWriterId !== RECOVERY_IDENTITY.terminalWriterId ||
    ex.protectedClaimKey !== TARGET.protectedClaimKey ||
    ex.sourceSha !== TARGET.sourceSha || ex.dispatchId !== TARGET.dispatchId ||
    ex.generation !== 1) return null;
  return proof;
}

/** Only validated proof fields may enter the public event evidence. */
function allowlistedProof(proof) {
  return {
    observedAt: proof.observedAt, fenceHeld: proof.fenceHeld, fenceId: proof.fenceId,
    dispatchId: proof.dispatchId, sourceSha: proof.sourceSha,
    protectedClaimKey: proof.protectedClaimKey, generation: proof.generation,
    activeWriterCount: proof.activeWriterCount, activeAttemptCount: proof.activeAttemptCount,
    activeSubprocessCount: proof.activeSubprocessCount, inflightEffectCount: proof.inflightEffectCount,
    unresolvedResourceCount: proof.unresolvedResourceCount,
    writeSetReconciled: proof.writeSetReconciled, evidenceRefs: proof.evidenceRefs,
    excludedReservation: proof.excludedReservation,
    excludedReservationCount: proof.excludedReservationCount,
  };
}

function validMarker(m) {
  if (!m || typeof m !== 'object' || Array.isArray(m)) return false;
  if (m.schema !== MARKER_SCHEMA || m.state !== 'POST_ATTEMPT_CONSUMED') return false;
  if (!ID.test(m.recoveryAttemptId ?? '')) return false;
  if (typeof m.eventEvidence !== 'string' || m.eventEvidence.length === 0 ||
    m.eventEvidence.length > EVIDENCE_BOUND) return false;
  if (m.eventEvidenceSha256 !== sha256hex(m.eventEvidence)) return false;
  if (typeof m.originalTerminalEvidenceSha256 !== 'string' ||
    m.originalTerminalEvidenceSha256.length !== 64) return false;
  if (m.priorRunLogSha256 !== RECOVERY_IDENTITY.prior.runLogSha256 ||
    m.priorReceipt?.status !== RECOVERY_IDENTITY.prior.status ||
    m.priorReceipt?.reason !== RECOVERY_IDENTITY.prior.reason ||
    m.priorReceipt?.publisherInvocationCount !== 0 ||
    m.priorReceipt?.httpPostCount !== 0) return false;
  if (!Number.isFinite(Date.parse(m.recoveryOccurredAt)) ||
    typeof m.proofObservedAt !== 'string' || !ID.test(m.hostFenceId ?? '')) return false;
  if (m.targetReservation?.casKey !== TARGET.casKey ||
    m.targetReservation?.revision !== 2 ||
    m.task331Bindings?.packBlobSha !== RECOVERY_IDENTITY.task331.packBlobSha ||
    m.task331Bindings?.claim?.commentId !== RECOVERY_IDENTITY.task331.claimCommentId ||
    m.task331Bindings?.frozenWriteSet?.writeSetSha256 !== RECOVERY_IDENTITY.task331.writeSetSha256) return false;
  return true;
}

/** Clone used for the public terminal event: original Claim identity fields
 *  unchanged, fresh recovery evidence/occurredAt, reservation revision 2. */
function reservationCloneFor(originalValue, marker) {
  const {recoveryAttempt, terminalCommentId, terminalCommentRef, ...original} = originalValue;
  return {...original, phase: 'PENDING_GITHUB_TERMINALIZATION',
    terminalEvidence: marker.eventEvidence,
    terminalOccurredAt: marker.recoveryOccurredAt};
}

/**
 * Selected Host only. Every port is Host-owned configuration; the factory
 * itself rejects caller-supplied writer authority shapes that do not match
 * the existing trusted #319/#203 terminal writer identity.
 */
export function createPrePostRecoveryController({
  store, claimReader, terminalReader, priorRunLog, lineage, quiescence,
  publisher, writerAuthority,
} = {}) {
  if (!store || typeof store.get !== 'function' || typeof store.put !== 'function' ||
    !claimReader || typeof claimReader.readPublishedClaim !== 'function' ||
    !terminalReader || typeof terminalReader.readEvents !== 'function' ||
    typeof terminalReader.readExactEvent !== 'function' ||
    !priorRunLog || typeof priorRunLog.read !== 'function' ||
    !lineage || typeof lineage.readClaim319 !== 'function' ||
    typeof lineage.readFrozen319 !== 'function' ||
    typeof lineage.readClaim331 !== 'function' ||
    typeof lineage.readFrozen331 !== 'function' ||
    !quiescence || typeof quiescence.readFreshProof !== 'function' ||
    !publisher || typeof publisher.publishDispatchStateChange !== 'function' ||
    writerAuthority?.mode !== 'SINGLE_WRITER_ADMISSION' ||
    writerAuthority?.exclusiveHostControl !== true ||
    writerAuthority?.repository !== TARGET.repository ||
    writerAuthority?.writerId !== RECOVERY_IDENTITY.terminalWriterId ||
    !ID.test(writerAuthority?.githubWriterLogin ?? ''))
    throw new TypeError('TRUSTED_PREPOST_RECOVERY_HOST_PORTS_REQUIRED');

  const claim203Request = Object.freeze({...TARGET, actorRole: 'builder',
    operatorKind: 'other', operatorId: 'zcode:candidate-publication:1'});

  async function originalClaimUnedited() {
    try {
      const claim = await claimReader.readPublishedClaim(claim203Request, TARGET.claimCommentId);
      return claim?.complete === true && claim?.edited === false &&
        claim?.commentId === TARGET.claimCommentId &&
        claim?.commentRef === TARGET.claimCommentRef;
    } catch { return false; }
  }

  function lineageExact(rows) {
    const {claim319, frozen319, claim331, frozen331} = rows ?? {};
    const i319 = RECOVERY_IDENTITY.task319, i331 = RECOVERY_IDENTITY.task331;
    const frozen319Digest = frozen319 && createHash('sha256')
      .update(JSON.stringify(frozen319.value)).digest('hex');
    return claim319?.revision === i319.claimRevision &&
      claim319?.value?.phase === i319.claimPhase &&
      claim319?.value?.commentId === i319.claimCommentId &&
      claim319?.value?.writerId === RECOVERY_IDENTITY.terminalWriterId &&
      frozen319?.revision === i319.frozenWriteSetRevision &&
      frozen319Digest === i319.frozenWriteSetDigest &&
      frozen319?.value?.taskPackBlobSha === i319.packBlobSha &&
      claim331?.revision === i331.claimRevision &&
      claim331?.value?.phase === i331.claimPhase &&
      claim331?.value?.commentId === i331.claimCommentId &&
      claim331?.value?.dispatchId === i331.dispatchId &&
      claim331?.value?.operatorId === i331.operatorId &&
      claim331?.value?.protectedClaimKey === i331.protectedClaimKey &&
      claim331?.value?.generation === 1 &&
      frozen331?.revision === i331.frozenWriteSetRevision &&
      frozen331?.value?.writeSetSha256 === i331.writeSetSha256 &&
      frozen331?.value?.taskPackBlobSha === i331.packBlobSha &&
      frozen331?.value?.claimCommentId === i331.claimCommentId;
  }

  /** Read-only proof that no terminal event exists at all. */
  async function terminalAbsent() {
    try {
      const events = await terminalReader.readEvents();
      return events?.status === 'ABSENT' && events?.complete === true;
    } catch { return false; }
  }

  /** Exact native terminal event readback for a reservation clone. */
  async function exactTerminalEvent(clone, ack = null) {
    let observed;
    try { observed = await terminalReader.readExactEvent(clone); }
    catch { return null; }
    if (observed?.status !== 'EXACT' || observed?.complete !== true ||
      observed?.edited !== false || !Number.isSafeInteger(observed?.commentId) ||
      observed.commentId < 1 ||
      observed.commentRef !== `${TARGET.issueRef}#issuecomment-${observed.commentId}`) return null;
    if (ack && Number.isSafeInteger(ack.commentId) && ack.commentId >= 1 &&
      observed.commentId !== ack.commentId) return null;
    return observed;
  }

  /** Final CAS revision 3 -> 4 once the exact event is proven; both callers
   *  operate at stored revision 3 (marker committed). */
  async function finalize(markerRow, observed) {
    const terminal = {...markerRow, phase: 'TERMINAL_VERIFIED',
      terminalCommentId: observed.commentId, terminalCommentRef: observed.commentRef};
    let finalRevision;
    try {
      ({revision: finalRevision} = await store.put(TARGET.casKey, terminal,
        {mustMatchRevision: 3}));
    } catch { return {fail: 'FINAL_CAS_AMBIGUOUS'}; }
    let final;
    try { final = await store.get(TARGET.casKey); }
    catch { return {fail: 'FINAL_READBACK_UNVERIFIED'}; }
    if (final?.revision !== 4 || JSON.stringify(final.value) !== JSON.stringify(terminal))
      return {fail: 'FINAL_READBACK_UNVERIFIED'};
    return {terminal, finalRevision};
  }

  /**
   * The one single-use recovery operation. No arguments: every input comes
   * from trusted Host ports. Any ambiguity preserves the row and returns
   * UNKNOWN; a consumed marker can never publish again.
   */
  async function recoverPrePostReservation() {
    let row;
    try { row = await store.get(TARGET.casKey); }
    catch { return unknown('TARGET_ROW_UNAVAILABLE'); }
    const v = row?.value;

    // ---- Revision 4: already terminal; read-only verify, never publish. ----
    if (row?.revision === 4 && v?.phase === 'TERMINAL_VERIFIED' &&
      validMarker(v?.recoveryAttempt)) {
      const {terminalCommentId, terminalCommentRef, recoveryAttempt, ...rest} = v;
      const originalShape = exactPendingRow(
        {...rest, phase: 'PENDING_GITHUB_TERMINALIZATION'});
      const clone = reservationCloneFor(v, v.recoveryAttempt);
      const observed = await exactTerminalEvent(clone);
      const claimOk = await originalClaimUnedited();
      if (!originalShape || !observed || !claimOk ||
        terminalCommentId !== observed.commentId ||
        terminalCommentRef !== observed.commentRef)
        return unknown('ALREADY_VERIFIED_RECHECK_FAILED');
      return Object.freeze({status: 'ALREADY_VERIFIED', reason: 'TERMINAL_VERIFIED_READ_ONLY',
        claimTerminality: 'VERIFIED', terminalCommentRef,
        recoveryAttemptId: recoveryAttempt.recoveryAttemptId,
        publisherInvocationCount: 0, httpPostCount: 0,
        authorized: false, admitted: false, taskPass: false, releasePass: false});
    }

    // ---- Revision 3 with marker: attempt consumed; read-only only. --------
    if (row?.revision === 3 && v?.phase === 'PENDING_GITHUB_TERMINALIZATION' &&
      validMarker(v?.recoveryAttempt)) {
      const {recoveryAttempt, ...original} = v;
      const originalShape = exactPendingRow(original);
      const marker = v.recoveryAttempt;
      const evidenceVerbatim = sha256hex(original.terminalEvidence) ===
        marker.originalTerminalEvidenceSha256 &&
        original.terminalOccurredAt === marker.originalTerminalOccurredAt;
      if (!originalShape || !evidenceVerbatim)
        return unknown('CONSUMED_MARKER_ROW_NOT_EXACT');
      const clone = reservationCloneFor(v, marker);
      try { terminalEventForReservation(clone, writerAuthority); }
      catch { return unknown('MARKER_EVENT_NOT_RECONSTRUCTIBLE'); }
      const observed = await exactTerminalEvent(clone);
      if (!observed) return unknown('CONSUMED_MARKER_EVENT_NOT_PROVEN', {
        recoveryAttemptId: marker.recoveryAttemptId});
      const done = await finalize(v, observed);
      if (done.fail) return unknown(done.fail);
      const claimOk = await originalClaimUnedited();
      if (!claimOk) return unknown('ORIGINAL_CLAIM_REREAD_FAILED');
      return Object.freeze({status: 'RECORDED', reason: 'CONSUMED_MARKER_RECONCILED_READ_ONLY',
        claimTerminality: 'VERIFIED', terminalCommentRef: observed.commentRef,
        recoveryAttemptId: marker.recoveryAttemptId,
        publisherInvocationCount: 0, httpPostCount: 0,
        authorized: false, admitted: false, taskPass: false, releasePass: false});
    }

    // ---- Only the exact stranded revision-2 row may enter the full path. --
    if (row?.revision !== 2 || !exactPendingRow(v))
      return unknown('TARGET_ROW_NOT_EXACT_REVISION_2_PREPOST', {
        revision: row?.revision ?? null, phase: v?.phase ?? null});

    // ---- Prior no-POST local artifact: exact bytes and parsed receipt. ----
    let prior;
    try { prior = await priorRunLog.read(); }
    catch { prior = null; }
    const p = RECOVERY_IDENTITY.prior;
    if (!prior || prior.sha256 !== p.runLogSha256 || prior.parsed?.status !== p.status ||
      prior.parsed?.reason !== p.reason ||
      prior.parsed?.publisherInvocationCount !== p.publisherInvocationCount ||
      prior.parsed?.httpPostCount !== p.httpPostCount)
      return unknown('PRIOR_NO_POST_ARTIFACT_NOT_EXACT');

    // ---- Original #203 Claim unedited; no terminal event anywhere. --------
    if (!(await originalClaimUnedited())) return unknown('ORIGINAL_CLAIM_NOT_EXACT');
    if (!(await terminalAbsent())) return unknown('TERMINAL_EVENT_NOT_ABSENT');

    // ---- Exact accepted #319 and #331 Claim/frozen write-set lineage. -----
    let rows;
    try {
      rows = {
        claim319: await lineage.readClaim319(), frozen319: await lineage.readFrozen319(),
        claim331: await lineage.readClaim331(), frozen331: await lineage.readFrozen331(),
      };
    } catch { return unknown('LINEAGE_ROWS_UNAVAILABLE'); }
    if (!lineageExact(rows)) return unknown('LINEAGE_BINDINGS_NOT_EXACT');

    // ---- Fresh target-bound quiescence proof (fence gate #1). -------------
    let proof1;
    try { proof1 = await quiescence.readFreshProof(); }
    catch { proof1 = null; }
    if (!qualifiedProof(proof1)) return unknown('QUIESCENCE_NOT_HELD');

    // ---- Deterministic recovery event evidence, prevalidated pre-CAS. -----
    const recoveryOccurredAt = new Date().toISOString();
    const recoveryAttemptId = 'recovery:331:t06d-prepost-recovery:' +
      sha256hex(proof1.fenceId + '|' + recoveryOccurredAt).slice(0, 24);
    const envelope = JSON.stringify({
      schema: ENVELOPE_SCHEMA,
      originalTerminalEvidence: v.terminalEvidence,
      originalTerminalOccurredAt: v.terminalOccurredAt,
      priorNoPostRunLogSha256: p.runLogSha256,
      recoveryAttemptId, recoveryOccurredAt,
      hostFenceId: proof1.fenceId,
      freshTargetBoundQuiescenceProof: allowlistedProof(proof1),
      exactExcludedReservationIdentity: proof1.excludedReservation,
      sixDurableEvidenceRefs: proof1.evidenceRefs,
      task331PackBlobOwnerClaimAndFrozenWriteSetBindings: {
        packBlobSha: RECOVERY_IDENTITY.task331.packBlobSha,
        packPRHeadSha: RECOVERY_IDENTITY.task331.packPRHeadSha,
        ownerReceiptComment: RECOVERY_IDENTITY.task331.ownerReceiptComment,
        ownerAmendmentComment: RECOVERY_IDENTITY.task331.ownerAmendmentComment,
        claim: {casKey: RECOVERY_IDENTITY.task331.claimCasKey,
          revision: rows.claim331.revision, phase: rows.claim331.value.phase,
          commentId: rows.claim331.value.commentId, commentRef: rows.claim331.value.commentRef,
          dispatchId: rows.claim331.value.dispatchId, generation: 1},
        frozenWriteSet: {key: RECOVERY_IDENTITY.task331.frozenWriteSetKey,
          revision: rows.frozen331.revision,
          writeSetSha256: rows.frozen331.value.writeSetSha256,
          pathCount: rows.frozen331.value.pathCount},
        task319Evidence: {claimCommentId: rows.claim319.value.commentId,
          frozenWriteSetDigest: createHash('sha256')
            .update(JSON.stringify(rows.frozen319.value)).digest('hex'),
          packBlobSha: rows.frozen319.value.taskPackBlobSha},
      },
    });
    const clone = {...v, terminalEvidence: envelope, terminalOccurredAt: recoveryOccurredAt};
    try { terminalEventForReservation(clone, writerAuthority); }
    catch { return unknown('RECOVERY_EVENT_NOT_VALID_BEFORE_MARKER'); }

    const marker = {
      schema: MARKER_SCHEMA, state: 'POST_ATTEMPT_CONSUMED',
      recoveryAttemptId, recoveryOccurredAt,
      priorRunLogSha256: p.runLogSha256,
      priorReceipt: {status: p.status, reason: p.reason,
        publisherInvocationCount: 0, httpPostCount: 0},
      hostFenceId: proof1.fenceId, proofObservedAt: proof1.observedAt,
      targetReservation: {
        casKey: TARGET.casKey, revision: 2, phase: v.phase,
        terminalWriterId: v.terminalWriterId,
        protectedClaimKey: v.protectedClaimKey, sourceSha: v.sourceSha,
        dispatchId: v.dispatchId, generation: v.generation,
      },
      excludedReservation: proof1.excludedReservation,
      excludedReservationCount: 1,
      evidenceRefs: proof1.evidenceRefs,
      quiescenceProofSha256: sha256hex(JSON.stringify(allowlistedProof(proof1))),
      originalTerminalEvidenceSha256: sha256hex(v.terminalEvidence),
      originalTerminalOccurredAt: v.terminalOccurredAt,
      eventEvidence: envelope, eventEvidenceSha256: sha256hex(envelope),
      task319Evidence: {
        claimCasKey: RECOVERY_IDENTITY.task319.claimCasKey,
        claimRevision: rows.claim319.revision, claimCommentId: rows.claim319.value.commentId,
        frozenWriteSetKey: RECOVERY_IDENTITY.task319.frozenWriteSetKey,
        frozenWriteSetDigest: createHash('sha256')
          .update(JSON.stringify(rows.frozen319.value)).digest('hex'),
        packBlobSha: rows.frozen319.value.taskPackBlobSha,
      },
      task331Bindings: {
        packBlobSha: RECOVERY_IDENTITY.task331.packBlobSha,
        packPRHeadSha: RECOVERY_IDENTITY.task331.packPRHeadSha,
        ownerReceiptComment: RECOVERY_IDENTITY.task331.ownerReceiptComment,
        ownerAmendmentComment: RECOVERY_IDENTITY.task331.ownerAmendmentComment,
        claim: {casKey: RECOVERY_IDENTITY.task331.claimCasKey,
          revision: rows.claim331.revision, commentId: rows.claim331.value.commentId,
          dispatchId: rows.claim331.value.dispatchId},
        frozenWriteSet: {key: RECOVERY_IDENTITY.task331.frozenWriteSetKey,
          revision: rows.frozen331.revision,
          writeSetSha256: rows.frozen331.value.writeSetSha256},
      },
    };
    const row3 = {...v, recoveryAttempt: marker};

    // ---- RECOVER_REHEARSE=1: stop read-only before the consuming CAS. -----
    if (process.env.RECOVER_REHEARSE === '1')
      return Object.freeze({status: 'REHEARSAL_OK', reason: 'PREFLIGHT_COMPLETE_NO_WRITE',
        recoveryAttemptId, evidenceLength: envelope.length, fenceId: proof1.fenceId,
        publisherInvocationCount: 0, httpPostCount: 0, casWrites: 0,
        authorized: false, admitted: false, taskPass: false, releasePass: false});

    // ---- CAS revision 2 -> 3: consume the only attempt BEFORE the POST. ---
    let markerRevision;
    try {
      ({revision: markerRevision} = await store.put(TARGET.casKey, row3, {mustMatchRevision: 2}));
    } catch {
      // markerWriteAmbiguous: re-read the public CAS first. If our exact
      // marker exists, the attempt is consumed and only read-only
      // reconciliation may follow; unknown state stops. Never a re-put.
      let recheck;
      try { recheck = await store.get(TARGET.casKey); } catch { recheck = null; }
      if (recheck?.revision === 3 && validMarker(recheck.value?.recoveryAttempt) &&
        JSON.stringify(recheck.value) === JSON.stringify(row3))
        return unknown('MARKER_WRITE_AMBIGUOUS_CONSUMED', {
          recoveryAttemptId: marker.recoveryAttemptId});
      return unknown('MARKER_CAS_RACE');
    }
    if (markerRevision !== 3) return unknown('MARKER_CAS_UNEXPECTED_REVISION');
    let reserved;
    try { reserved = await store.get(TARGET.casKey); }
    catch { return unknown('MARKER_READBACK_UNVERIFIED'); }
    if (reserved?.revision !== 3 ||
      JSON.stringify(reserved.value) !== JSON.stringify(row3))
      return unknown('MARKER_READBACK_UNVERIFIED');

    // ---- Same Host fence must still hold immediately before the POST. -----
    let proof2;
    try { proof2 = await quiescence.readFreshProof(); }
    catch { proof2 = null; }
    if (!qualifiedProof(proof2, proof1.fenceId))
      return unknown('FENCE_CHANGED_BEFORE_PUBLISH', {recoveryAttemptId});

    // ---- The single publisher invocation. No retry of any kind. -----------
    let ack = null;
    try { ack = await publisher.publishDispatchStateChange(Object.freeze({...clone})); }
    catch { ack = null; }
    const observed = await exactTerminalEvent(clone, ack);
    if (!observed) return unknown(ack ? 'TERMINAL_READBACK_NOT_VERIFIED'
      : 'TERMINAL_PUBLICATION_AMBIGUOUS', {recoveryAttemptId,
      publisherInvocationCount: 1, httpPostCount: 1});

    // ---- Same fence through final CAS and complete readback. --------------
    let proof3;
    try { proof3 = await quiescence.readFreshProof(); }
    catch { proof3 = null; }
    if (!qualifiedProof(proof3, proof1.fenceId))
      return unknown('FENCE_CHANGED_BEFORE_FINAL_CAS', {recoveryAttemptId,
        publisherInvocationCount: 1, httpPostCount: 1});

    const done = await finalize(row3, observed);
    if (done.fail) return unknown(done.fail, {recoveryAttemptId,
      publisherInvocationCount: 1, httpPostCount: 1});

    // ---- Independent final verification. -----------------------------------
    const finalEvents = await exactTerminalEvent(clone);
    const claimOk = await originalClaimUnedited();
    if (!finalEvents || finalEvents.commentId !== observed.commentId || !claimOk)
      return unknown('FINAL_INDEPENDENT_VERIFICATION_FAILED');

    return Object.freeze({
      status: 'RECORDED', reason: 'PREPOST_RESERVATION_RECOVERED',
      claimTerminality: 'VERIFIED', terminalCommentRef: observed.commentRef,
      terminalCommentId: observed.commentId, recoveryAttemptId,
      fenceId: proof1.fenceId, publisherInvocationCount: 1, httpPostCount: 1,
      casRevisions: {before: 2, marker: 3, final: 4},
      evidenceLength: envelope.length,
      authorized: false, admitted: false, taskPass: false, releasePass: false,
    });
  }

  return Object.freeze({recoverPrePostReservation});
}

// ---- Selected-Host one-shot CLI (run-as-main only; never imported) --------
// Real Host wiring for the exact #331 recovery: public DomainHarness v0.10.3
// CAS stores, the reviewed PR #330 native publisher/reader boundaries and
// this process's own process-table quiescence measurements. The expected
// final implementation SHA/TREE must come from Host environment variables;
// it is never guessed, defaulted or read from the CAS row.
const FENCE_PREFIX = 'fence:i331-prepost-recovery:';
const RECOVERY_SIX_REFS = Object.freeze({
  writer: 'https://github.com/kaicreator-mm/tally/issues/203',
  attempt: 'https://github.com/kaicreator-mm/tally/issues/331',
  subprocess: 'https://github.com/kaicreator-mm/tally/issues/284',
  effect: 'https://github.com/kaicreator-mm/tally/pull/204',
  resource: 'https://github.com/kaicreator-mm/tally/pull/205',
  writeSet: 'https://github.com/kaicreator-mm/tally/blob/9fd9b1b798f756c708813b6ac8f2c3e92d72039a/docs/implementation/v2/T06D_PREPOST_RECOVERY_TASK_PACK_R1.json',
});
const CLI_WRITER_RE = /(registry-203\.sqlite|registry-319\.sqlite|registry-331\.sqlite|t06d-host-claim-publication|single-writer|terminalize|t06d-prepost-recovery|recover-prepost-reservation|reserveAndPublish|publish-real|claim-task)/i;
const SOURCE_BASE = '72ce3067213b75ffbc04762d866759c3af4324f8';
const THREE_PATHS = Object.freeze([
  '.woodpecker/v2-t06d-prepost-recovery.yaml',
  'experiments/v2/t06d-prepost-recovery/recover-prepost-reservation.mjs',
  'experiments/v2/t06d-prepost-recovery/recover-prepost-reservation.test.mjs',
]);

/** Parse the #319 prior-run JSONL artifact into the zero-POST receipt the
 *  controller requires (raw observed values; wiring asserts semantics). */
export function parsePriorRunLog(text) {
  const events = [];
  for (const line of String(text ?? '').split(/\r?\n/)) {
    if (!line.trim()) continue;
    try { events.push(JSON.parse(line)); } catch { return null; }
  }
  const resultLine = events.find(e => e?.step === 'TERMINALIZE_RESULT');
  const stoppedLine = events.find(e => e?.step === 'STOPPED');
  if (!resultLine || !stoppedLine) return null;
  const publisherInvocationCount = events
    .filter(e => typeof e?.step === 'string' && e.step !== 'STOPPED' &&
      /POST|PUBLISH/.test(e.step)).length;
  return {
    status: resultLine.result?.status, reason: resultLine.result?.reason,
    // Raw observed values; the CLI wiring asserts their exact semantics.
    casWriteBlocked: stoppedLine.casWriteBlocked,
    retry: stoppedLine.retry,
    publisherInvocationCount, httpPostCount: publisherInvocationCount,
  };
}

function cliProcessTable() {
  const out = execSync(
    'powershell -NoProfile -Command "Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId,CommandLine | ConvertTo-Json -Compress"',
    {encoding: 'utf8', timeout: 25000, windowsHide: true, maxBuffer: 32 * 1024 * 1024});
  const j = JSON.parse(out);
  return Array.isArray(j) ? j : [j];
}
function cliExcludedSelfTree(table) {
  const byPid = new Map(table.map(p => [p.ProcessId, p]));
  const excl = new Set([process.pid]);
  let cur = byPid.get(process.pid);
  while (cur) { const par = byPid.get(cur.ParentProcessId); if (!par) break; excl.add(par.ProcessId); cur = par; }
  const kids = new Map();
  for (const p of table) { if (!kids.has(p.ParentProcessId)) kids.set(p.ParentProcessId, []); kids.get(p.ParentProcessId).push(p); }
  const q = [process.pid];
  while (q.length) for (const c of kids.get(q.shift()) ?? []) if (!excl.has(c.ProcessId)) { excl.add(c.ProcessId); q.push(c.ProcessId); }
  return excl;
}

async function runSelectedHostCli() {
  const log = o => console.log(typeof o === 'string' ? o : JSON.stringify(o));
  const proxyOf = v => { if (!v) return 'absent'; try { const u = new URL(v); return `present(${u.hostname}:${u.port || u.protocol.replace(':', '')})`; } catch { return 'present(unparsed)'; } };
  log({step: 'ENV', node: process.version,
    HTTP_PROXY: proxyOf(process.env.HTTP_PROXY), HTTPS_PROXY: proxyOf(process.env.HTTPS_PROXY),
    NO_PROXY: process.env.NO_PROXY ?? 'absent', NODE_USE_ENV_PROXY: process.env.NODE_USE_ENV_PROXY ?? 'absent',
    tokenPresent: typeof process.env.TALLY_GITHUB_TOKEN === 'string' && process.env.TALLY_GITHUB_TOKEN.length >= 8});
  if (process.env.NODE_USE_ENV_PROXY !== '1') { log({step: 'ABORT', reason: 'NODE_USE_ENV_PROXY_NOT_SET_BEFORE_LAUNCH'}); process.exitCode = 2; return; }
  if (typeof process.env.TALLY_GITHUB_TOKEN !== 'string' || process.env.TALLY_GITHUB_TOKEN.length < 8)
    { log({step: 'ABORT', reason: 'HOST_GITHUB_TOKEN_REQUIRED'}); process.exitCode = 2; return; }
  const EXPECTED_HEAD = process.env.RECOVERY_EXPECTED_HEAD;
  const EXPECTED_TREE = process.env.RECOVERY_EXPECTED_TREE;
  if (!/^[0-9a-f]{40}$/.test(EXPECTED_HEAD ?? '') || !/^[0-9a-f]{40}$/.test(EXPECTED_TREE ?? ''))
    { log({step: 'ABORT', reason: 'RECOVERY_EXPECTED_HEAD_AND_TREE_REQUIRED'}); process.exitCode = 2; return; }

  const repoRoot = resolve(dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')), '../../..');
  const git = args => execSync(`git ${args}`, {cwd: repoRoot, encoding: 'utf8', timeout: 20000, windowsHide: true});
  const REG203 = 'C:/xDev/kAiCreator/_tmp/tally-alm-claims/registry-203.sqlite';
  const REG319 = 'C:/xDev/kAiCreator/_tmp/i319-claim/run-20261106T043955/registry-319.sqlite';
  const REG331 = 'C:/xDev/kAiCreator/_tmp/tally-alm-claims/registry-331.sqlite';
  const PRIOR_RUNLOG = 'C:/xDev/kAiCreator/_tmp/i319-terminalization/run.log';

  const writerAuthority = Object.freeze({
    mode: 'SINGLE_WRITER_ADMISSION', exclusiveHostControl: true,
    repository: TARGET.repository, writerId: RECOVERY_IDENTITY.terminalWriterId,
    githubWriterLogin: 'kaicreator-mm',
  });

  function writeSetReconciled() {
    const head = git('rev-parse HEAD').trim();
    const tree = git('log -1 --pretty=%T').trim();
    const dirty = git('status --porcelain --untracked-files=no').trim();
    const changed = git(`diff --name-only ${SOURCE_BASE} ${EXPECTED_HEAD}`)
      .split(/\r?\n/).map(x => x.trim()).filter(Boolean).sort();
    const digest = sha256hex(JSON.stringify([...THREE_PATHS].sort()));
    return head === EXPECTED_HEAD && tree === EXPECTED_TREE && dirty === '' &&
      JSON.stringify(changed) === JSON.stringify([...THREE_PATHS].sort()) &&
      digest === RECOVERY_IDENTITY.task331.writeSetSha256;
  }

  const store203 = await openSqliteCasStore({path: REG203, busyTimeoutMs: 1000});
  const store319 = await openSqliteCasStore({path: REG319, busyTimeoutMs: 1000});
  const store331 = await openSqliteCasStore({path: REG331, busyTimeoutMs: 1000});
  try {
    const octokit = new Octokit({auth: process.env.TALLY_GITHUB_TOKEN, request: {timeout: 15000}});
    const publisher = createHostBoundNativeGitHubClaimPublisher({octokit, owner: 'kaicreator-mm', repo: 'tally', writerAuthority});
    const claimReader = createHostNativeGitHubClaimReadbackPortFromClient({octokit, owner: 'kaicreator-mm', repo: 'tally', writerLogin: 'kaicreator-mm'});
    const terminalReader = {
      readEvents: () => readDispatchStateEventsFromClient({octokit, writerLogin: 'kaicreator-mm'}),
      readExactEvent: clone => readDispatchStateEventsFromClient({octokit, writerLogin: 'kaicreator-mm',
        expectedEvent: terminalEventForReservation(clone, writerAuthority)}),
    };
    const priorRunLog = {
      async read() {
        let bytes;
        try { bytes = readFileSync(PRIOR_RUNLOG); } catch { return null; }
        const parsed = parsePriorRunLog(bytes.toString('utf8'));
        if (parsed?.casWriteBlocked !== true || parsed?.retry !== false) return null;
        return {sha256: sha256hex(bytes.toString('utf8')), parsed};
      },
    };
    const lineage = {
      readClaim319: () => store319.get(RECOVERY_IDENTITY.task319.claimCasKey),
      readFrozen319: () => store319.get(RECOVERY_IDENTITY.task319.frozenWriteSetKey),
      readClaim331: () => store331.get(RECOVERY_IDENTITY.task331.claimCasKey),
      readFrozen331: () => store331.get(RECOVERY_IDENTITY.task331.frozenWriteSetKey),
    };

    const FENCE_ID = FENCE_PREFIX +
      new Date().toISOString().replace(/[-:]/g, '').replace(/\.\d+Z$/, 'Z');
    let proofCalls = 0;
    const quiescence = {
      async readFreshProof() {
        proofCalls++;
        const table = cliProcessTable();
        const excl = cliExcludedSelfTree(table);
        const writers = table.filter(p => !excl.has(p.ProcessId) && CLI_WRITER_RE.test(p.CommandLine ?? ''));
        const writerPids = new Set(writers.map(w => w.ProcessId));
        const writerChildren = table.filter(p => !excl.has(p.ProcessId) && writerPids.has(p.ParentProcessId));
        let targetRow = null, otherPending = 0;
        for (const s of [store203, store319, store331]) {
          const keys = await s.list('tally:v2:claim-admission:');
          for (const k of keys ?? []) {
            const r = await s.get(k);
            if (!String(r?.value?.phase ?? '').startsWith('PENDING_')) continue;
            if (k === TARGET.casKey) targetRow = r;
            else otherPending++;
          }
        }
        const tv = targetRow?.value;
        const excluded = targetRow && tv?.phase === 'PENDING_GITHUB_TERMINALIZATION' &&
          tv?.terminalWriterId === RECOVERY_IDENTITY.terminalWriterId &&
          tv?.terminalReservationRevision === 2 && tv?.dispatchId === TARGET.dispatchId &&
          tv?.protectedClaimKey === TARGET.protectedClaimKey && tv?.sourceSha === TARGET.sourceSha &&
          tv?.generation === 1 ? {
            casKey: TARGET.casKey, revision: tv.terminalReservationRevision,
            phase: tv.phase, terminalWriterId: tv.terminalWriterId,
            protectedClaimKey: tv.protectedClaimKey, sourceSha: tv.sourceSha,
            dispatchId: tv.dispatchId, generation: tv.generation,
          } : null;
        let prDrift = 0;
        for (const [num, head] of Object.entries({204: '3a894ddbdef8d4a8513c1c33837d32a816e2234e', 205: 'ca985bb9cd125c6359eb393fa5243a04199a53b4'})) {
          const pr = (await octokit.rest.pulls.get({owner: 'kaicreator-mm', repo: 'tally', pull_number: Number(num)}))?.data;
          if (pr?.number !== Number(num) || pr.state !== 'open' || pr.merged_at !== null || pr.head?.sha !== head) prDrift++;
        }
        const proof = {
          observedAt: new Date().toISOString(),
          fenceHeld: writers.length === 0 && otherPending === 0 && excluded !== null,
          fenceId: FENCE_ID, dispatchId: TARGET.dispatchId, sourceSha: TARGET.sourceSha,
          protectedClaimKey: TARGET.protectedClaimKey, generation: 1,
          activeWriterCount: writers.length, activeAttemptCount: otherPending,
          activeSubprocessCount: writerChildren.length,
          inflightEffectCount: otherPending + writers.length,
          unresolvedResourceCount: prDrift, writeSetReconciled: writeSetReconciled(),
          evidenceRefs: RECOVERY_SIX_REFS, excludedReservation: excluded,
          excludedReservationCount: excluded === null ? 0 : 1,
        };
        log({step: 'QUIESCENCE_PROOF', call: proofCalls, fenceId: proof.fenceId, fenceHeld: proof.fenceHeld,
          activeWriterCount: proof.activeWriterCount, activeAttemptCount: proof.activeAttemptCount,
          activeSubprocessCount: proof.activeSubprocessCount, inflightEffectCount: proof.inflightEffectCount,
          unresolvedResourceCount: proof.unresolvedResourceCount, writeSetReconciled: proof.writeSetReconciled,
          excludedReservationCount: proof.excludedReservationCount, observedAt: proof.observedAt});
        return proof;
      },
    };

    log({step: 'CAS_BEFORE', target203: await (async () => {
      const r = await store203.get(TARGET.casKey);
      // Pack-observed revision-2 row digest: pins the exact row bytes on top
      // of the controller's per-field exactness at the Host wiring boundary.
      const rowSha = r?.revision === 2 && r.value?.phase === 'PENDING_GITHUB_TERMINALIZATION'
        ? sha256hex(JSON.stringify(r.value)) : null;
      return {revision: r?.revision, phase: r?.value?.phase,
        marker: r?.value?.recoveryAttempt?.state ?? null,
        rowValueSha256: rowSha,
        rowShaMatchesPack: rowSha === null ? null
          : rowSha === 'bcebd6e05bd1fe4a1658c4e65bfa13cc94963643079270f6529560dd42fcdcc2'};
    })()});

    const controller = createPrePostRecoveryController({
      store: store203, claimReader, terminalReader, priorRunLog, lineage,
      quiescence, publisher, writerAuthority,
    });
    const t0 = Date.now();
    const result = await controller.recoverPrePostReservation();
    log({step: 'RECOVERY_RESULT', elapsedMs: Date.now() - t0, result});
    const okStatuses = ['RECORDED', 'ALREADY_VERIFIED', 'REHEARSAL_OK'];
    process.exitCode = okStatuses.includes(result?.status) ? 0 : 4;
  } finally {
    await store203.close(); await store319.close(); await store331.close();
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await runSelectedHostCli();
}
