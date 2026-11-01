/**
 * Tally v2 J07a — advisory, source-linked Release evidence projection.
 *
 * This module never returns RELEASE_PASS, an authorization, or a permitted
 * operation. Input records are untrusted observations until independently
 * attested through the selected GitHub C / Kernel and applicable ADS RQ.
 * It must NOT be used as an action admission check.
 */
const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_][A-Za-z0-9_.-]*\/[A-Za-z0-9_][A-Za-z0-9_.-]*$/;
const GATE_NAMES=Object.freeze([
  'productFreeze', 'l2Freeze', 'taskDag', 'nativeClaims',
  'independentReview', 'woodpeckerCI', 'realHost', 'installedAcceptance',
  'fullRegression', 'hiddenValidation', 'criticalJourneys',
  'releaseQualification', 'ownerReleaseAuthority'
]);
const VALUES=new Set(['PASS','FAIL','NOT_RUN','UNKNOWN','STALE']);
const freeze=value=>Object.freeze(value);
const isInstant=value=>typeof value==='string' && Number.isFinite(Date.parse(value)) &&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
function githubRef(ref,repository){
  if(typeof ref!=='string'||ref.length>2048||/[%?#]/.test(ref))return false;
  const base=`https://github.com/${repository}/`;
  if(!ref.startsWith(base))return false;
  const suffix=ref.slice(base.length);
  return /^(?:issues\/[1-9]\d*|pull\/[1-9]\d*|commit\/[a-f0-9]{40})(?:\/)?$/.test(suffix);
}
const invalid=reason=>freeze({status:'UNKNOWN',reason,sourceQualified:false,
  observedEvidence:'UNKNOWN',gates:freeze([]),blockers:freeze([]),
  independentRQRequired:true,
  taskPass:false,releasePass:false,canMerge:false,canTag:false,canPublish:false,
  authorization:'NONE',advisoryOnly:true});
function oneGate(name,record,subjectSha,repository,asOf){
  if(record===undefined||record===null)return freeze({name,status:'NOT_RUN',reason:'EVIDENCE_MISSING',evidenceRef:null});
  if(!record||typeof record!=='object'||Array.isArray(record)||!VALUES.has(record.status))
    return freeze({name,status:'UNKNOWN',reason:'EVIDENCE_MALFORMED',evidenceRef:null});
  // An old review or CI receipt must never be silently promoted to this head.
  if(record.subjectSha!==undefined && !SHA.test(record.subjectSha))
    return freeze({name,status:'UNKNOWN',reason:'SUBJECT_ID_INVALID',evidenceRef:null});
  if(record.subjectSha!==undefined && record.subjectSha!==subjectSha)
    return freeze({name,status:'STALE',reason:'SOURCE_SHA_CHANGED',evidenceRef:null});
  if(record.status==='FAIL'||record.status==='NOT_RUN'||record.status==='UNKNOWN'||record.status==='STALE')
    return freeze({name,status:record.status,reason:`REPORTED_${record.status}`,evidenceRef:null});
  if(record.subjectSha!==subjectSha||!isInstant(record.observedAsOf)||!githubRef(record.evidenceRef,repository))
    return freeze({name,status:'UNKNOWN',reason:'PASS_EVIDENCE_UNVERIFIED',evidenceRef:null});
  // A receipt observed after the snapshot's as-of instant is impossible data;
  // it can never be promoted to a reported PASS for this head.
  if(Date.parse(record.observedAsOf)>Date.parse(asOf))
    return freeze({name,status:'UNKNOWN',reason:'OBSERVATION_TIME_INVALID',evidenceRef:null});
  return freeze({name,status:'REPORTED_PASS_UNVERIFIED',reason:'INDEPENDENT_SOURCE_ATTESTATION_REQUIRED',
    evidenceRef:record.evidenceRef,subjectSha});
}
/**
 * Build a conservative progress projection from GitHub-derived observations.
 * A complete-looking evidence set can at most request independent RQ review.
 * It cannot authorize merge, tag, deployment, or a claim of Release PASS.
 */
export function projectReleaseReadiness(input={}){
  if(!input||typeof input!=='object'||Array.isArray(input)||
     !REPO.test(input.repository)||!SHA.test(input.sourceSha??'')||
     !isInstant(input.observedAsOf)||!input.gates||
     typeof input.gates!=='object'||Array.isArray(input.gates))
    return invalid('RELEASE_SOURCE_UNVERIFIED');
  const gates=freeze(GATE_NAMES.map(name=>oneGate(name,input.gates[name],input.sourceSha,input.repository,input.observedAsOf)));
  const blockers=freeze(gates.filter(gate=>gate.status!=='REPORTED_PASS_UNVERIFIED').map(gate=>
    freeze({name:gate.name,status:gate.status,reason:gate.reason})));
  const reportedComplete=blockers.length===0;
  return freeze({status:'BLOCKED',reason:reportedComplete?'INDEPENDENT_RQ_AND_AUTHORITY_NOT_ATTESTED':'REQUIRED_EVIDENCE_INCOMPLETE',
    repository:input.repository,sourceSha:input.sourceSha,observedAsOf:input.observedAsOf,
    observedEvidence:reportedComplete?'REPORTED_COMPLETE_UNVERIFIED':'INCOMPLETE',
    sourceQualified:false,independentRQRequired:true,gates,blockers,
    gateCounts:freeze({required:GATE_NAMES.length,reportedPass:gates.filter(gate=>gate.status==='REPORTED_PASS_UNVERIFIED').length,
      missingOrUnqualified:blockers.length,comparison:'COUNTS_NOT_PERCENTAGES'}),
    taskPass:false,releasePass:false,canMerge:false,canTag:false,canPublish:false,
    authorization:'NONE',advisoryOnly:true});
}
export const RELEASE_READINESS_REQUIRED_GATES=GATE_NAMES;
