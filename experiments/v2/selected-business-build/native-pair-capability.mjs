/**
 * T06f selected C provider implementation: real read-only native Work Item +
 * ADS Claim event, NOT a native Claim CAS writer. Bundled from existing C.
 * Only the Host supplies GitHub's read-only token and exact source pin.
 */
import {readGitHubNativeTaskSourcePair} from '../../../packages/tally-github-native-c/src/task-source-pair.mjs';
export const MODULE_ID='capability-tally-source-observation@v2.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-source-observation';
export const OPERATIONS=Object.freeze(['tally.observeReadiness']);
const SOURCE=/^[0-9a-f]{40}$/;
const allowed='https://github.com/kaicreator-mm/tally/issues/125';
const reject=reason=>Object.freeze({
  state:'NATIVE_NOT_CURRENT',status:'UNKNOWN',reason,
  sourceRef:allowed,sourceSha:process.env.TALLY_NATIVE_EXPECTED_SHA??null,
  observationCount:0,claimStatus:'NOT_VERIFIED',authorized:false,
  admitted:false,taskPass:false,releasePass:false
});
export async function observeReadiness(input){
  const sourceSha=input?.sourceSha;
  const expected=process.env.TALLY_NATIVE_EXPECTED_SHA;
  if(input?.sourceRef!==allowed||input?.issueNumber!==125||
     !SOURCE.test(sourceSha??'')||!SOURCE.test(expected??'')||
     sourceSha!==expected)
    return reject('HOST_PIN_OR_ISSUE_MISMATCH');
  const observed=await readGitHubNativeTaskSourcePair({
    token:process.env.TALLY_GITHUB_READ_TOKEN,owner:'kaicreator-mm',
    repo:'tally',issueNumber:125,expectedSourceSha:expected,limit:1000
  });
  if(observed.status!=='SOURCE_PAIR_OBSERVED'||
     observed.sourceQualified!==true||
     observed.claimStatus!=='NOT_VERIFIED')
    return reject('PAIR_SOURCE_NOT_QUALIFIED');
  return Object.freeze({
    state:'NATIVE_TASK_SOURCE_ADVISORY',
    status:'SOURCE_PAIR_OBSERVED',
    reason:'ACCEPTED_CAS_CLAIM_NOT_VERIFIED',
    sourceRef:allowed,sourceSha:expected,
    observationCount:observed.claimEvents.candidateCount,
    workItem:observed.workItem,claimEvents:observed.claimEvents,
    claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
    taskPass:false,releasePass:false
  });
}
