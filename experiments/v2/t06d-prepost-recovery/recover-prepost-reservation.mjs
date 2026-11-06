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
import {createHash, randomUUID} from 'node:crypto';
import {execSync} from 'node:child_process';
import {closeSync, fstatSync, lstatSync, openSync, readFileSync, readdirSync, statSync, unlinkSync, writeSync, fsyncSync} from 'node:fs';
import {dirname, relative, resolve} from 'node:path';
import {pathToFileURL} from 'node:url';
import {Octokit} from '@octokit/rest';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {TARGET, readDispatchStateEventsFromClient} from '../../../packages/tally-github-native-c/src/task-dispatch-state-events.mjs';
import {terminalEventForReservation} from '../../../packages/tally-github-native-c/src/task-claim-github-publisher.mjs';
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
    sourceSha: '72ce3067213b75ffbc04762d866759c3af4324f8',
    actorRole: 'builder',
    operatorKind: 'other',
    claimCommentRef: 'https://github.com/kaicreator-mm/tally/issues/331#issuecomment-6690955003',
    taskPackRef: 'https://github.com/kaicreator-mm/tally/blob/9fd9b1b798f756c708813b6ac8f2c3e92d72039a/docs/implementation/v2/T06D_PREPOST_RECOVERY_TASK_PACK_R1.json',
    ownerReceiptRef: 'https://github.com/kaicreator-mm/tally/pull/332#issuecomment-6685952738',
    reviewWitnessRef: 'https://github.com/kaicreator-mm/tally/pull/332#pullrequestreview-5800344352',
    writeSetDigestBasis: 'sha256(JSON.stringify(paths.sort())) compact; tally.v2.allowed-write-set-normalization/1',
    writeSetDocument: [
      'frozen exact write set for #331:',
      '.woodpecker/v2-t06d-prepost-recovery.yaml',
      'experiments/v2/t06d-prepost-recovery/recover-prepost-reservation.mjs',
      'experiments/v2/t06d-prepost-recovery/recover-prepost-reservation.test.mjs',
    ].join('\n'),
    writerId: 'tally-host:win32:t06d-terminalization:1',
    ownerApprovedAt: '2026-11-06T13:28:12.000Z',
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
  if (proof.fenceHeld !== true || !ID.test(proof.fenceId ?? '') ||
    !Number.isSafeInteger(proof.fenceOwnerPid) || proof.fenceOwnerPid < 1) return null;
  if (expectedFenceId !== null && proof.fenceId !== expectedFenceId) return null;
  if (proof.dispatchId !== TARGET.dispatchId || proof.sourceSha !== TARGET.sourceSha ||
    proof.protectedClaimKey !== TARGET.protectedClaimKey || proof.generation !== 1) return null;
  if (proof.activeWriterCount !== 0 || proof.activeAttemptCount !== 0 ||
    proof.activeSubprocessCount !== 0 || proof.inflightEffectCount !== 0 ||
    proof.unresolvedResourceCount !== 0 || proof.writeSetReconciled !== true) return null;
  const registry=proof.registryInventory,processes=proof.processInventory;
  if (!registry||registry.registryCount<3||!/^[a-f0-9]{64}$/.test(registry.registryPathsSha256??'')||
    registry.targetReservationCount!==1||registry.targetExactCount!==1||registry.unresolvedRecordCount!==0) return null;
  if (!processes||processes.inventoryComplete!==true||
    !/^[a-f0-9]{64}$/.test(processes.processTableSha256??'')||
    processes.activeWriterCount!==0||processes.activeSubprocessCount!==0||processes.unknownWriterCount!==0||
    processes.activeWriterCount!==proof.activeWriterCount||
    processes.activeSubprocessCount!==proof.activeSubprocessCount) return null;
  if (!exactHistoricalCiEvidence(proof.historicalCiEvidence)) return null;
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
    fenceOwnerPid: proof.fenceOwnerPid, registryInventory: proof.registryInventory,
    processInventory: proof.processInventory, historicalCiEvidence: proof.historicalCiEvidence,
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
  if (m.markerVersion !== undefined && m.markerVersion !== 2) return false;
  if (m.markerVersion === 2 &&
    (!/^[a-f0-9]{64}$/.test(m.task331Bindings?.claim?.valueSha256 ?? '') ||
      !/^[a-f0-9]{64}$/.test(m.task331Bindings?.frozenWriteSet?.valueSha256 ?? '') ||
      m.task331Bindings?.claim?.commentRef !== RECOVERY_IDENTITY.task331.claimCommentRef ||
      m.task331Bindings?.claim?.sourceSha !== RECOVERY_IDENTITY.task331.sourceSha ||
      m.task331Bindings?.claim?.writerId !== RECOVERY_IDENTITY.task331.writerId ||
      m.task331Bindings?.frozenWriteSet?.taskPackRef !== RECOVERY_IDENTITY.task331.taskPackRef ||
      m.task331Bindings?.frozenWriteSet?.ownerReceiptRef !== RECOVERY_IDENTITY.task331.ownerReceiptRef))
    return false;
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
    const {claim319,frozen319,claim331,frozen331}=rows??{};
    const i319=RECOVERY_IDENTITY.task319,i331=RECOVERY_IDENTITY.task331;
    const frozen319Digest=frozen319&&sha256hex(JSON.stringify(frozen319.value));
    const claim=claim331?.value,frozen=frozen331?.value;
    const frozenAt=Date.parse(frozen?.frozenAt);
    const keys=frozen&&typeof frozen==='object'&&!Array.isArray(frozen)?Object.keys(frozen).sort():[];
    const wanted=[...FROZEN_331_KEYS].sort();
    const keysExact=JSON.stringify(keys)===JSON.stringify(wanted)||
      (JSON.stringify(keys)===JSON.stringify([...wanted,'ownerAmendmentRef'].sort())&&
       frozen?.ownerAmendmentRef==='https://github.com/kaicreator-mm/tally/pull/332#issuecomment-6689612741');
    const claimExact=claim?.phase===i331.claimPhase&&claim?.repository==='kaicreator-mm/tally'&&
      claim?.issueNumber===331&&claim?.sourceSha===i331.sourceSha&&claim?.actorRole===i331.actorRole&&
      claim?.operatorKind===i331.operatorKind&&claim?.operatorId===i331.operatorId&&
      claim?.writerId===i331.writerId&&claim?.commentId===i331.claimCommentId&&
      claim?.commentRef===i331.claimCommentRef&&claim?.dispatchId===i331.dispatchId&&
      claim?.protectedClaimKey===i331.protectedClaimKey&&claim?.generation===1;
    const frozenExact=keysExact&&frozen?.phase==='FROZEN_DAG_WRITESET'&&
      frozen?.repository==='kaicreator-mm/tally'&&frozen?.issueNumber===331&&
      frozen?.sourceSha===i331.sourceSha&&frozen?.writeSetSha256===i331.writeSetSha256&&
      frozen?.document===i331.writeSetDocument&&frozen?.pathCount===i331.pathCount&&
      frozen?.writeSetDigestBasis===i331.writeSetDigestBasis&&frozen?.taskPackRef===i331.taskPackRef&&
      frozen?.taskPackBlobSha===i331.packBlobSha&&frozen?.packPRHeadSha===i331.packPRHeadSha&&
      frozen?.ownerReceiptRef===i331.ownerReceiptRef&&frozen?.reviewWitnessRef===i331.reviewWitnessRef&&
      frozen?.protectedClaimKey===i331.protectedClaimKey&&frozen?.generation===1&&
      frozen?.dispatchId===i331.dispatchId&&frozen?.claimCommentId===i331.claimCommentId&&
      frozen?.claimCommentRef===i331.claimCommentRef&&frozen?.writerId===i331.writerId&&
      Number.isFinite(frozenAt)&&new Date(frozenAt).toISOString()===frozen.frozenAt&&
      frozenAt>=Date.parse(i331.ownerApprovedAt);
    const baseExact=claim319?.revision===i319.claimRevision&&claim319?.value?.phase===i319.claimPhase&&
      claim319?.value?.commentId===i319.claimCommentId&&
      claim319?.value?.writerId===RECOVERY_IDENTITY.terminalWriterId&&
      frozen319?.revision===i319.frozenWriteSetRevision&&frozen319Digest===i319.frozenWriteSetDigest&&
      frozen319?.value?.taskPackBlobSha===i319.packBlobSha;
    if(!baseExact||!claimExact||!frozenExact||claim331?.revision!==i331.claimRevision||
      frozen331?.revision!==i331.frozenWriteSetRevision)return null;
    return {claim331ValueSha256:sha256hex(JSON.stringify(claim)),
      frozen331ValueSha256:sha256hex(JSON.stringify(frozen)),frozen319ValueSha256:frozen319Digest};
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
    const lineageBindings=lineageExact(rows);
    if (!lineageBindings) return unknown('LINEAGE_BINDINGS_NOT_EXACT');

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
        exactClaimValueSha256: lineageBindings.claim331ValueSha256,
        exactFrozenWriteSetValueSha256: lineageBindings.frozen331ValueSha256,
        packPRHeadSha: RECOVERY_IDENTITY.task331.packPRHeadSha,
        ownerReceiptComment: RECOVERY_IDENTITY.task331.ownerReceiptComment,
        ownerAmendmentComment: RECOVERY_IDENTITY.task331.ownerAmendmentComment,
        claim: {casKey: RECOVERY_IDENTITY.task331.claimCasKey,
          revision: rows.claim331.revision, phase: rows.claim331.value.phase,
          sourceSha: rows.claim331.value.sourceSha, actorRole: rows.claim331.value.actorRole,
          operatorKind: rows.claim331.value.operatorKind, operatorId: rows.claim331.value.operatorId,
          writerId: rows.claim331.value.writerId, commentId: rows.claim331.value.commentId,
          commentRef: rows.claim331.value.commentRef, dispatchId: rows.claim331.value.dispatchId,
          protectedClaimKey: rows.claim331.value.protectedClaimKey, generation: 1,
          valueSha256: lineageBindings.claim331ValueSha256},
        frozenWriteSet: {key: RECOVERY_IDENTITY.task331.frozenWriteSetKey,
          revision: rows.frozen331.revision, writeSetSha256: rows.frozen331.value.writeSetSha256,
          pathCount: rows.frozen331.value.pathCount, sourceSha: rows.frozen331.value.sourceSha,
          taskPackRef: rows.frozen331.value.taskPackRef, taskPackBlobSha: rows.frozen331.value.taskPackBlobSha,
          packPRHeadSha: rows.frozen331.value.packPRHeadSha, ownerReceiptRef: rows.frozen331.value.ownerReceiptRef,
          reviewWitnessRef: rows.frozen331.value.reviewWitnessRef, protectedClaimKey: rows.frozen331.value.protectedClaimKey,
          generation: rows.frozen331.value.generation, dispatchId: rows.frozen331.value.dispatchId,
          claimCommentId: rows.frozen331.value.claimCommentId, claimCommentRef: rows.frozen331.value.claimCommentRef,
          writerId: rows.frozen331.value.writerId, frozenAt: rows.frozen331.value.frozenAt,
          valueSha256: lineageBindings.frozen331ValueSha256},
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
      schema: MARKER_SCHEMA, markerVersion: 2, state: 'POST_ATTEMPT_CONSUMED',
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
          commentRef: rows.claim331.value.commentRef, sourceSha: rows.claim331.value.sourceSha,
          actorRole: rows.claim331.value.actorRole, operatorKind: rows.claim331.value.operatorKind,
          operatorId: rows.claim331.value.operatorId, writerId: rows.claim331.value.writerId,
          dispatchId: rows.claim331.value.dispatchId, protectedClaimKey: rows.claim331.value.protectedClaimKey,
          generation: rows.claim331.value.generation, valueSha256: lineageBindings.claim331ValueSha256},
        frozenWriteSet: {key: RECOVERY_IDENTITY.task331.frozenWriteSetKey,
          revision: rows.frozen331.revision, writeSetSha256: rows.frozen331.value.writeSetSha256,
          pathCount: rows.frozen331.value.pathCount, sourceSha: rows.frozen331.value.sourceSha,
          taskPackRef: rows.frozen331.value.taskPackRef, taskPackBlobSha: rows.frozen331.value.taskPackBlobSha,
          packPRHeadSha: rows.frozen331.value.packPRHeadSha, ownerReceiptRef: rows.frozen331.value.ownerReceiptRef,
          reviewWitnessRef: rows.frozen331.value.reviewWitnessRef, protectedClaimKey: rows.frozen331.value.protectedClaimKey,
          generation: rows.frozen331.value.generation, dispatchId: rows.frozen331.value.dispatchId,
          claimCommentId: rows.frozen331.value.claimCommentId, claimCommentRef: rows.frozen331.value.claimCommentRef,
          writerId: rows.frozen331.value.writerId, frozenAt: rows.frozen331.value.frozenAt,
          valueSha256: lineageBindings.frozen331ValueSha256},
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

  async function verifyTerminalReservationReadOnly() {
    let row;
    try { row = await store.get(TARGET.casKey); }
    catch { return unknown('READ_ONLY_TARGET_ROW_UNAVAILABLE',{casWrites:0}); }
    const value=row?.value;
    if(row?.revision!==4||value?.phase!=='TERMINAL_VERIFIED'||!validMarker(value?.recoveryAttempt))
      return unknown('READ_ONLY_TARGET_NOT_TERMINAL_REVISION_4',{
        revision:row?.revision??null,phase:value?.phase??null,casWrites:0});
    const {terminalCommentId,terminalCommentRef,recoveryAttempt,...rest}=value;
    const originalShape=exactPendingRow({...rest,phase:'PENDING_GITHUB_TERMINALIZATION'});
    const clone=reservationCloneFor(value,recoveryAttempt);
    const observed=await exactTerminalEvent(clone),claimOk=await originalClaimUnedited();
    if(!originalShape||!observed||!claimOk||terminalCommentId!==observed.commentId||
      terminalCommentRef!==observed.commentRef)
      return unknown('READ_ONLY_TERMINAL_VERIFICATION_FAILED',{casWrites:0});
    return Object.freeze({
      status:'ALREADY_VERIFIED',reason:'TERMINAL_VERIFIED_READ_ONLY',
      claimTerminality:'VERIFIED',terminalCommentRef,
      recoveryAttemptId:recoveryAttempt.recoveryAttemptId,
      publisherInvocationCount:0,httpPostCount:0,casWrites:0,
      historicalReviewOrderDeviation:{status:'NOT_RETROACTIVELY_REMEDIABLE',
        reviewCommentId:5805039103,reviewedHeadSha:'25f9dcdf9d2dc93a00165c3c3762fee13f0495b4',
        reason:'NO_DURABLE_PRE_SIDE_EFFECT_INDEPENDENT_REVIEW_WITNESS'},
      authorized:false,admitted:false,taskPass:false,releasePass:false,
    });
  }

  return Object.freeze({recoverPrePostReservation,verifyTerminalReservationReadOnly});
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

const HISTORICAL_CI_EXPECTED = Object.freeze([
  {number:204,headSha:'3a894ddbdef8d4a8513c1c33837d32a816e2234e',statuses:[
    {context:'ci/woodpecker/pr/v2-t06l-kernel-admission',state:'failure',targetUrl:'https://ci.kaicreator.com/repos/12/pipeline/195/1'},
    {context:'ci/woodpecker/pr/v2-t06m-local-agent',state:'failure',targetUrl:'https://ci.kaicreator.com/repos/12/pipeline/195/2'},
    {context:'ci/woodpecker/pr/v2-t06n-concurrent-agent',state:'failure',targetUrl:'https://ci.kaicreator.com/repos/12/pipeline/195/3'},
  ]},
  {number:205,headSha:'ca985bb9cd125c6359eb393fa5243a04199a53b4',statuses:[
    {context:'ci/woodpecker/pr/v2-t06l-kernel-admission',state:'failure',targetUrl:'https://ci.kaicreator.com/repos/12/pipeline/196/1'},
    {context:'ci/woodpecker/pr/v2-t06m-local-agent',state:'failure',targetUrl:'https://ci.kaicreator.com/repos/12/pipeline/196/2'},
    {context:'ci/woodpecker/pr/v2-t06n-concurrent-agent',state:'failure',targetUrl:'https://ci.kaicreator.com/repos/12/pipeline/196/3'},
  ]},
]);
const QUIESCENT_PHASES = new Set(['PUBLISHED_VERIFIED','FROZEN_DAG_WRITESET','TERMINAL_VERIFIED','DONE','CLOSED','PASS','SUCCESS','COMPLETE','COMPLETED']);
const HOST_AGENT_NAMES = new Set(['node.exe','node','powershell.exe','pwsh.exe','python.exe','python','git.exe','cmd.exe','codex.exe','zcode.exe','woodpecker-agent.exe']);
const FROZEN_331_KEYS = ['phase','repository','issueNumber','sourceSha','writeSetSha256','document','pathCount',
  'writeSetDigestBasis','taskPackRef','taskPackBlobSha','packPRHeadSha','ownerReceiptRef','reviewWitnessRef',
  'protectedClaimKey','generation','dispatchId','claimCommentId','claimCommentRef','writerId','frozenAt'];

function exactHistoricalCiEvidence(value) {
  const observed = Date.parse(value?.observedAt);
  if (value?.schema !== 'tally.t06d-historical-ci-evidence/v1' ||
    !Number.isFinite(observed) || observed > Date.now() || Date.now() - observed > 30000 ||
    !Array.isArray(value.pullRequests) || value.pullRequests.length !== 2) return false;
  return HISTORICAL_CI_EXPECTED.every(expected => {
    const pr = value.pullRequests.find(x => x?.number === expected.number);
    if (!pr || pr.headSha !== expected.headSha || pr.state !== 'open' ||
      pr.merged !== false || pr.combinedState !== 'failure' ||
      !Array.isArray(pr.statuses) || pr.statuses.length !== 3) return false;
    const actual = [...pr.statuses].sort((a,b)=>String(a.context).localeCompare(String(b.context)));
    const wanted = [...expected.statuses].sort((a,b)=>a.context.localeCompare(b.context));
    return wanted.every((x,i)=>actual[i]?.context===x.context &&
      actual[i]?.state===x.state && actual[i]?.targetUrl===x.targetUrl);
  });
}
export async function readHistoricalCiEvidenceFromClient(octokit) {
  try {
    const pullRequests=[];
    for (const expected of HISTORICAL_CI_EXPECTED) {
      const pr=(await octokit.rest.pulls.get({owner:'kaicreator-mm',repo:'tally',pull_number:expected.number}))?.data;
      const combined=(await octokit.rest.repos.getCombinedStatusForRef({
        owner:'kaicreator-mm',repo:'tally',ref:expected.headSha,
      }))?.data;
      if (!pr || !combined || !Array.isArray(combined.statuses)) return null;
      pullRequests.push({
        number:expected.number,headSha:pr.head?.sha??null,state:pr.state,
        merged:pr.merged_at!=null,combinedState:combined.state,
        statuses:combined.statuses.map(x=>({context:x.context,state:x.state,targetUrl:x.target_url??null}))
          .sort((a,b)=>String(a.context).localeCompare(String(b.context))),
      });
    }
    const evidence={schema:'tally.t06d-historical-ci-evidence/v1',
      observedAt:new Date().toISOString(),pullRequests};
    return exactHistoricalCiEvidence(evidence)?evidence:null;
  } catch { return null; }
}
function normalizedPath(value) { return resolve(value).replace(/[\\/]+/g,'/').toLowerCase(); }

export function discoverHostRegistryPaths(root) {
  const rootPath=resolve(root);
  const rootStat=lstatSync(rootPath);
  if (!rootStat.isDirectory() || rootStat.isSymbolicLink()) throw new Error('HOST_REGISTRY_ROOT_NOT_TRUSTED');
  const pending=[rootPath], found=[];
  let visited=0;
  while (pending.length) {
    const dir=pending.pop();
    for (const entry of readdirSync(dir,{withFileTypes:true})) {
      const path=resolve(dir,entry.name), stat=lstatSync(path);
      if (stat.isSymbolicLink()) throw new Error('HOST_REGISTRY_INVENTORY_SYMLINK');
      if (stat.isDirectory()) pending.push(path);
      else if (stat.isFile() && /^registry-[A-Za-z0-9_.-]+\.sqlite$/i.test(entry.name)) found.push(path);
      if (++visited>10000) throw new Error('HOST_REGISTRY_INVENTORY_LIMIT_EXCEEDED');
    }
  }
  return found.sort((a,b)=>normalizedPath(a).localeCompare(normalizedPath(b)));
}
export async function inspectHostRegistryInventory(registryStores,root,targetRegistryPath,requiredPaths) {
  const rootPath=resolve(root), discovered=discoverHostRegistryPaths(rootPath);
  const byPath=new Map([...registryStores.entries()].map(([p,store])=>[normalizedPath(p),store]));
  const discoveredKeys=discovered.map(normalizedPath);
  if (byPath.size!==discoveredKeys.length || discoveredKeys.some(p=>!byPath.has(p)) ||
    !requiredPaths.every(p=>discoveredKeys.includes(normalizedPath(p))))
    throw new Error('HOST_REGISTRY_INVENTORY_INCOMPLETE');
  const targetPath=normalizedPath(targetRegistryPath);
  let targetOccurrences=0,targetExactCount=0,targetReservation=null,pendingRecordCount=0,unknownRecordCount=0;
  for (const path of discovered) {
    const store=byPath.get(normalizedPath(path));
    const keys=await store.list('tally:v2:');
    if (!Array.isArray(keys)) throw new Error('HOST_REGISTRY_KEY_INVENTORY_UNREADABLE');
    for (const key of keys) {
      if (typeof key!=='string'||!key.startsWith('tally:v2:')) throw new Error('HOST_REGISTRY_KEY_INVALID');
      const row=await store.get(key);
      if (!row||!Number.isSafeInteger(row.revision)||!row.value||typeof row.value!=='object'||Array.isArray(row.value)) {
        unknownRecordCount++; continue;
      }
      if (key===TARGET.casKey) {
        targetOccurrences++;
        if (normalizedPath(path)===targetPath&&row.revision===2&&exactPendingRow(row.value)) {
          targetExactCount++;
          targetReservation={casKey:TARGET.casKey,revision:2,phase:row.value.phase,
            terminalWriterId:row.value.terminalWriterId,protectedClaimKey:row.value.protectedClaimKey,
            sourceSha:row.value.sourceSha,dispatchId:row.value.dispatchId,generation:row.value.generation};
        } else pendingRecordCount++;
        continue;
      }
      if (typeof row.value.phase!=='string') unknownRecordCount++;
      else if (!QUIESCENT_PHASES.has(row.value.phase)) pendingRecordCount++;
    }
  }
  const relativePaths=discovered.map(p=>relative(rootPath,p).replace(/\\/g,'/')).sort();
  return {registryCount:discovered.length,registryPathsSha256:sha256hex(JSON.stringify(relativePaths)),
    targetReservationCount:targetOccurrences,targetExactCount,targetReservation,
    pendingRecordCount,unknownRecordCount,unresolvedRecordCount:pendingRecordCount+unknownRecordCount};
}
export function acquireExclusiveHostFence(lockPath,{ownerId='tally-host:win32:t06d-prepost-recovery:1'}={}) {
  const path=resolve(lockPath),fenceId=FENCE_PREFIX+process.pid+':'+randomUUID();
  const record=JSON.stringify({schema:'tally.t06d-host-exclusive-fence/v1',
    fenceId,ownerId,ownerPid:process.pid,acquiredAt:new Date().toISOString()});
  let fd;
  try { fd=openSync(path,'wx',0o600); }
  catch (error) {
    if (error?.code==='EEXIST') throw new Error('HOST_FENCE_ALREADY_HELD_OR_STALE');
    throw new Error('HOST_FENCE_ACQUIRE_FAILED');
  }
  try { writeSync(fd,record,null,'utf8'); fsyncSync(fd); }
  catch {
    try { closeSync(fd); } catch {}
    try { if (readFileSync(path,'utf8')===record) unlinkSync(path); } catch {}
    throw new Error('HOST_FENCE_INITIALIZE_FAILED');
  }
  let released=false;
  const isHeld=()=>{
    if (released||fd===null) return false;
    try { const a=fstatSync(fd),b=statSync(path);
      return a.dev===b.dev&&a.ino===b.ino&&readFileSync(path,'utf8')===record;
    } catch { return false; }
  };
  const release=()=>{
    if (released) return;
    if (!isHeld()) throw new Error('HOST_FENCE_OWNERSHIP_LOST');
    closeSync(fd); fd=null;
    if (readFileSync(path,'utf8')!==record) throw new Error('HOST_FENCE_OWNERSHIP_LOST');
    unlinkSync(path); released=true;
  };
  return Object.freeze({fenceId,ownerId,ownerPid:process.pid,isHeld,release});
}
function cliProcessTable() {
  const command="$ErrorActionPreference='Stop'; Get-CimInstance Win32_Process -ErrorAction Stop | Select-Object ProcessId,ParentProcessId,Name,ExecutablePath,CommandLine | ConvertTo-Json -Compress";
  const out=execSync('powershell -NoProfile -NonInteractive -Command "'+command+'"',
    {encoding:'utf8',timeout:25000,windowsHide:true,maxBuffer:32*1024*1024});
  const value=JSON.parse(out),table=Array.isArray(value)?value:[value];
  if(!table.length)throw new Error('HOST_PROCESS_INVENTORY_EMPTY');
  return table;
}
function cliExcludedSelfTree(table,selfPid=process.pid) {
  const byPid=new Map();
  for(const row of table){
    const pid=Number(row?.ProcessId),parentPid=Number(row?.ParentProcessId);
    if(!Number.isSafeInteger(pid)||pid<1||!Number.isSafeInteger(parentPid)||parentPid<0||
      byPid.has(pid)||typeof row?.Name!=='string'||!row.Name||
      (row.CommandLine!=null&&typeof row.CommandLine!=='string')||
      (row.ExecutablePath!=null&&typeof row.ExecutablePath!=='string'))
      throw new Error('HOST_PROCESS_INVENTORY_ROW_INVALID');
    byPid.set(pid,{...row,ProcessId:pid,ParentProcessId:parentPid});
  }
  if(!byPid.has(selfPid))throw new Error('HOST_PROCESS_SELF_NOT_IN_INVENTORY');
  const excluded=new Set([selfPid]),seen=new Set([selfPid]);
  let current=byPid.get(selfPid);
  while(current?.ParentProcessId>0&&byPid.has(current.ParentProcessId)){
    const parent=current.ParentProcessId;
    if(seen.has(parent))throw new Error('HOST_PROCESS_PARENT_CYCLE');
    seen.add(parent);excluded.add(parent);current=byPid.get(parent);
  }
  const children=new Map();
  for(const row of byPid.values()){
    if(!children.has(row.ParentProcessId))children.set(row.ParentProcessId,[]);
    children.get(row.ParentProcessId).push(row);
  }
  const queue=[selfPid];
  while(queue.length)for(const child of children.get(queue.shift())??[]){
    if(!excluded.has(child.ProcessId)){excluded.add(child.ProcessId);queue.push(child.ProcessId);}
  }
  return {byPid,children,excluded};
}
export function inspectHostProcessInventory(table,selfPid,hostRoot) {
  const {byPid,children,excluded}=cliExcludedSelfTree(table,selfPid);
  const root=resolve(hostRoot).replace(/\\/g,'/').toLowerCase(),candidates=new Set();
  for(const row of byPid.values()){
    if(excluded.has(row.ProcessId))continue;
    const name=row.Name.toLowerCase();
    const joined=(name+' '+String(row.CommandLine??'')+' '+String(row.ExecutablePath??''))
      .replace(/\\\\/g,'/').toLowerCase();
    if(joined.includes(root)||HOST_AGENT_NAMES.has(name))candidates.add(row.ProcessId);
  }
  const roots=[];
  for(const pid of candidates){
    let parent=byPid.get(pid).ParentProcessId,hasCandidate=false;
    const seen=new Set([pid]);
    while(parent>0&&byPid.has(parent)){
      if(seen.has(parent))throw new Error('HOST_PROCESS_PARENT_CYCLE');
      seen.add(parent);
      if(candidates.has(parent)){hasCandidate=true;break;}
      parent=byPid.get(parent).ParentProcessId;
    }
    if(!hasCandidate)roots.push(pid);
  }
  const active=new Set(roots),queue=[...roots];
  while(queue.length)for(const child of children.get(queue.shift())??[]){
    if(!excluded.has(child.ProcessId)&&!active.has(child.ProcessId)){
      active.add(child.ProcessId);queue.push(child.ProcessId);
    }
  }
  const unknownWriterCount=roots.filter(pid=>{
    const row=byPid.get(pid);
    return !CLI_WRITER_RE.test(String(row.CommandLine??'')+' '+String(row.ExecutablePath??''));
  }).length;
  const rows=[...byPid.values()].sort((a,b)=>a.ProcessId-b.ProcessId)
    .map(x=>[x.ProcessId,x.ParentProcessId,x.Name,x.ExecutablePath??null,x.CommandLine??null]);
  return {inventoryComplete:true,processCount:byPid.size,
    processTableSha256:sha256hex(JSON.stringify(rows)),activeWriterCount:roots.length,
    activeSubprocessCount:active.size-roots.length,unknownWriterCount};
}
async function runSelectedHostCli() {
  const log=o=>console.log(typeof o==='string'?o:JSON.stringify(o));
  const proxyOf=v=>{if(!v)return 'absent';try{const u=new URL(v);return 'present('+u.hostname+':'+(u.port||u.protocol.replace(':',''))+')';}catch{return 'present(unparsed)';}};
  log({step:'ENV',node:process.version,HTTP_PROXY:proxyOf(process.env.HTTP_PROXY),
    HTTPS_PROXY:proxyOf(process.env.HTTPS_PROXY),NO_PROXY:process.env.NO_PROXY??'absent',
    NODE_USE_ENV_PROXY:process.env.NODE_USE_ENV_PROXY??'absent',
    tokenPresent:typeof process.env.TALLY_GITHUB_TOKEN==='string'&&process.env.TALLY_GITHUB_TOKEN.length>=8});
  if(process.env.NODE_USE_ENV_PROXY!=='1'){log({step:'ABORT',reason:'NODE_USE_ENV_PROXY_NOT_SET_BEFORE_LAUNCH'});process.exitCode=2;return;}
  if(typeof process.env.TALLY_GITHUB_TOKEN!=='string'||process.env.TALLY_GITHUB_TOKEN.length<8){
    log({step:'ABORT',reason:'HOST_GITHUB_TOKEN_REQUIRED'});process.exitCode=2;return;}
  const EXPECTED_HEAD=process.env.RECOVERY_EXPECTED_HEAD,EXPECTED_TREE=process.env.RECOVERY_EXPECTED_TREE;
  if(!/^[0-9a-f]{40}$/.test(EXPECTED_HEAD??'')||!/^[0-9a-f]{40}$/.test(EXPECTED_TREE??'')){
    log({step:'ABORT',reason:'RECOVERY_EXPECTED_HEAD_AND_TREE_REQUIRED'});process.exitCode=2;return;}
  const HOST_TMP_ROOT='C:/xDev/kAiCreator/_tmp';
  const REG203=resolve(HOST_TMP_ROOT,'tally-alm-claims/registry-203.sqlite');
  const REG319=resolve(HOST_TMP_ROOT,'i319-claim/run-20261106T043955/registry-319.sqlite');
  const REG331=resolve(HOST_TMP_ROOT,'tally-alm-claims/registry-331.sqlite');
  const PRIOR_RUNLOG=resolve(HOST_TMP_ROOT,'i319-terminalization/run.log');
  const FENCE_PATH=resolve(HOST_TMP_ROOT,'tally-alm-claims/t06d-prepost-recovery.lock');
  const repoRoot=resolve(dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/,'$1')),'../../..');
  const git=args=>execSync('git '+args,{cwd:repoRoot,encoding:'utf8',timeout:20000,windowsHide:true});
  const writerAuthority=Object.freeze({mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
    repository:TARGET.repository,writerId:RECOVERY_IDENTITY.terminalWriterId,githubWriterLogin:'kaicreator-mm'});
  function writeSetReconciled(){
    const head=git('rev-parse HEAD').trim(),tree=git('log -1 --pretty=%T').trim();
    const dirty=git('status --porcelain --untracked-files=all').trim();
    const changed=git('diff --name-only '+SOURCE_BASE+' '+EXPECTED_HEAD)
      .split(/\r?\n/).map(x=>x.trim()).filter(Boolean).sort();
    const digest=sha256hex(JSON.stringify([...THREE_PATHS].sort()));
    return head===EXPECTED_HEAD&&tree===EXPECTED_TREE&&dirty===''&&
      JSON.stringify(changed)===JSON.stringify([...THREE_PATHS].sort())&&
      digest===RECOVERY_IDENTITY.task331.writeSetSha256;
  }
  let fence=null;
  const registryStores=new Map();
  try{
    fence=acquireExclusiveHostFence(FENCE_PATH);
    const registryPaths=discoverHostRegistryPaths(HOST_TMP_ROOT);
    for(const required of [REG203,REG319,REG331])
      if(!registryPaths.some(p=>normalizedPath(p)===normalizedPath(required)))throw new Error('REQUIRED_HOST_REGISTRY_MISSING');
    for(const path of registryPaths){
      const store=await openSqliteCasStore({path,busyTimeoutMs:1000});
      registryStores.set(normalizedPath(path),store);
    }
    const store203=registryStores.get(normalizedPath(REG203));
    const store319=registryStores.get(normalizedPath(REG319));
    const store331=registryStores.get(normalizedPath(REG331));
    if(!store203||!store319||!store331)throw new Error('REQUIRED_HOST_REGISTRY_UNAVAILABLE');
    const octokit=new Octokit({auth:process.env.TALLY_GITHUB_TOKEN,request:{timeout:15000}});
    const publisher={async publishDispatchStateChange(){throw new Error('READ_ONLY_VERIFICATION_PUBLISH_DISABLED');}};
    const claimReader=createHostNativeGitHubClaimReadbackPortFromClient({
      octokit,owner:'kaicreator-mm',repo:'tally',writerLogin:'kaicreator-mm'});
    const terminalReader={
      readEvents:()=>readDispatchStateEventsFromClient({octokit,writerLogin:'kaicreator-mm'}),
      readExactEvent:clone=>readDispatchStateEventsFromClient({octokit,writerLogin:'kaicreator-mm',
        expectedEvent:terminalEventForReservation(clone,writerAuthority)}),
    };
    const priorRunLog={async read(){
      let bytes;try{bytes=readFileSync(PRIOR_RUNLOG);}catch{return null;}
      const parsed=parsePriorRunLog(bytes.toString('utf8'));
      if(parsed?.casWriteBlocked!==true||parsed?.retry!==false)return null;
      return {sha256:sha256hex(bytes.toString('utf8')),parsed};
    }};
    const lineage={
      readClaim319:()=>store319.get(RECOVERY_IDENTITY.task319.claimCasKey),
      readFrozen319:()=>store319.get(RECOVERY_IDENTITY.task319.frozenWriteSetKey),
      readClaim331:()=>store331.get(RECOVERY_IDENTITY.task331.claimCasKey),
      readFrozen331:()=>store331.get(RECOVERY_IDENTITY.task331.frozenWriteSetKey),
    };
    let proofCalls=0;
    const quiescence={async readFreshProof(){
      proofCalls++;
      const processInventory=inspectHostProcessInventory(cliProcessTable(),process.pid,repoRoot);
      const registryInventory=await inspectHostRegistryInventory(
        registryStores,HOST_TMP_ROOT,REG203,[REG203,REG319,REG331]);
      const historicalCiEvidence=await readHistoricalCiEvidenceFromClient(octokit);
      const excluded=registryInventory.targetReservation;
      const proof={
        observedAt:new Date().toISOString(),
        fenceHeld:fence.isHeld()&&processInventory.activeWriterCount===0&&
          processInventory.activeSubprocessCount===0&&registryInventory.unresolvedRecordCount===0&&
          registryInventory.targetReservationCount===1&&registryInventory.targetExactCount===1&&
          historicalCiEvidence!==null,
        fenceId:fence.fenceId,fenceOwnerPid:fence.ownerPid,
        dispatchId:TARGET.dispatchId,sourceSha:TARGET.sourceSha,
        protectedClaimKey:TARGET.protectedClaimKey,generation:1,
        activeWriterCount:processInventory.activeWriterCount,
        activeAttemptCount:registryInventory.pendingRecordCount,
        activeSubprocessCount:processInventory.activeSubprocessCount,
        inflightEffectCount:registryInventory.pendingRecordCount+
          processInventory.activeWriterCount+processInventory.activeSubprocessCount,
        unresolvedResourceCount:registryInventory.unknownRecordCount+
          processInventory.unknownWriterCount+(historicalCiEvidence?0:1),
        writeSetReconciled:writeSetReconciled(),
        registryInventory:{registryCount:registryInventory.registryCount,
          registryPathsSha256:registryInventory.registryPathsSha256,
          targetReservationCount:registryInventory.targetReservationCount,
          targetExactCount:registryInventory.targetExactCount,
          unresolvedRecordCount:registryInventory.unresolvedRecordCount},
        processInventory,historicalCiEvidence,
        evidenceRefs:RECOVERY_SIX_REFS,excludedReservation:excluded,
        excludedReservationCount:excluded===null?0:1,
      };
      log({step:'QUIESCENCE_PROOF',call:proofCalls,fenceId:proof.fenceId,fenceHeld:proof.fenceHeld,
        registryCount:registryInventory.registryCount,targetReservationCount:registryInventory.targetReservationCount,
        pendingRecordCount:registryInventory.pendingRecordCount,unknownRecordCount:registryInventory.unknownRecordCount,
        activeWriterCount:proof.activeWriterCount,activeSubprocessCount:proof.activeSubprocessCount,
        unknownWriterCount:processInventory.unknownWriterCount,historicalCiVerified:historicalCiEvidence!==null,
        writeSetReconciled:proof.writeSetReconciled,observedAt:proof.observedAt});
      return proof;
    }};
    const targetBefore=await store203.get(TARGET.casKey);
    log({step:'READ_ONLY_MODE',targetRevision:targetBefore?.revision??null,targetPhase:targetBefore?.value?.phase??null,
      publisherEnabled:false,casWritesEnabled:false,fenceId:fence.fenceId});
    if(!writeSetReconciled())throw new Error('EXACT_LOCAL_HEAD_TREE_OR_WRITESET_MISMATCH');
    const controller=createPrePostRecoveryController({store:store203,claimReader,terminalReader,
      priorRunLog,lineage,quiescence,publisher,writerAuthority});
    const result=await controller.verifyTerminalReservationReadOnly();
    log({step:'READ_ONLY_VERIFICATION_RESULT',result});
    process.exitCode=result?.status==='ALREADY_VERIFIED'&&result.publisherInvocationCount===0&&
      result.httpPostCount===0&&result.casWrites===0?0:4;
  }catch(error){
    log({step:'HOST_VERIFY_ABORT',reason:String(error?.message??error)});process.exitCode=4;
  }finally{
    if(fence){try{fence.release();}catch(error){
      log({step:'HOST_FENCE_RELEASE_FAILED',reason:String(error?.message??error)});process.exitCode=5;}}
    for(const store of registryStores.values()){try{await store.close();}catch{process.exitCode=5;}}
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await runSelectedHostCli();
}
