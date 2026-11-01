/**
 * T02b minimal read-only Tally Capability adapter for required BUILD route.
 * No native GitHub grant, Host state, private task-meta or external Effect.
 * True GitHub C transport and authority are owned by T03b, not this module.
 */
export const MODULE_ID = 'capability-tally-source-observation@v2.0.0';
export const PACKAGE_KIND = 'capability';
export const ABI_VERSION = 'v08-capability-abi/1';
export const CAPABILITY_ID = 'tally-source-observation';
export const OPERATIONS = Object.freeze(['tally.observeReadiness']);
/** Pure source-ref observation; never claims verified native facts. */
export async function observeReadiness(input) {
  return Object.freeze({
    state: 'OBSERVATION_NOT_NATIVE_VERIFIED',
    sourceRef: typeof input?.sourceRef === 'string' ? input.sourceRef : null,
    authorized: false, admitted: false, taskPass: false, releasePass: false,
  });
}
