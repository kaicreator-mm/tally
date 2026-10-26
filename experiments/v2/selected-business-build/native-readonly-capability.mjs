/**
 * T04b selected C — trusted, closed-world, source-READONLY native GitHub.
 * This file is bundled together with the merged T03b provider reader by the
 * public BUILD producer. No custom REST, schema validator or permission claim.
 * The ephemeral workflow token belongs to Host deployment configuration,
 * never to the Business intent, Capability input or K journal.
 *
 * This in-process trusted-package integration does not prove that untrusted
 * arbitrary JS running in the same process cannot access environment secrets.
 */
import {readGitHubIssueReports} from '../../../packages/tally-github-native-c/src/read-only.mjs';
export const MODULE_ID='capability-tally-source-observation@v2.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-source-observation';
export const OPERATIONS=Object.freeze(['tally.observeReadiness']);
const allowedRef='https://github.com/kaicreator-mm/tally/issues/111';
const SOURCE_SHA=/^[0-9a-f]{40}$/;
export async function observeReadiness(input) {
  const sourceSha=input?.sourceSha;
  const expected=process.env.TALLY_NATIVE_EXPECTED_SHA;
  if(input?.sourceRef!==allowedRef || input?.issueNumber!==111 ||
     !SOURCE_SHA.test(sourceSha??'') || !SOURCE_SHA.test(expected??'') ||
     sourceSha!==expected) {
    return Object.freeze({state:'INVALID_NATIVE_REQUEST',reason:'HOST_PIN_OR_ISSUE_MISMATCH',
      authorized:false,admitted:false,taskPass:false,releasePass:false});
  }
  const token=process.env.TALLY_GITHUB_READ_TOKEN;
  const observation=await readGitHubIssueReports({
    token,owner:'kaicreator-mm',repo:'tally',issueNumber:111,
    expectedSourceSha:expected,
    publisherBindings:{'task-meta:kaicreator-mm/tally':['github-actions[bot]']},
    limit:1000,
  });
  // Constrain the ABI to an advisory source projection, never attach the
  // token, raw comment body, claimant or an authoritative Task verdict.
  return Object.freeze({
    state:observation.status==='CURRENT_OBSERVED'?'NATIVE_ADVISORY_OBSERVED':'NATIVE_NOT_CURRENT',
    status:observation.status,reason:observation.reason,
    sourceRef:allowedRef,sourceSha:expected,
    observationCount:observation.observationCount??0,
    authorized:false,admitted:false,taskPass:false,releasePass:false,
  });
}
