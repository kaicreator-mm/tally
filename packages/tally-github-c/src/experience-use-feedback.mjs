/**
 * #274: pure ExperienceUseFeedback producer for pinned ADS v4.18 / GitHub v5.
 * No I/O, no native authority, no private task-meta, no automatic adoption.
 * Only safe-integer and string inputs reach the stable RFC8785-compatible JSON
 * serializer. The native C separately runs the real pinned schema validator.
 */
import {createHash} from 'node:crypto';
const SHA=/^[a-f0-9]{40}$/;
const DIGEST=/^[a-f0-9]{64}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ISO=/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/;
const PHASE=new Set(['PRESENTED','ACCEPTED','APPLIED','SKIPPED','INDEPENDENTLY_CHECKED','OBSERVED_EFFECT']);
const EPISTEMIC=new Set(['OBSERVED','SELF_REPORTED','INFERRED','UNKNOWN']);
const KEYS=new Set(['repository','issueNumber','taskRef','attemptRef','sourceSha',
  'publisherSourceSha','adsRevision','publisherNamespace','revision','experienceRef',
  'procedureRevision','phase','epistemic','observedAt','actor','sourceRefs',
  'phaseEvidence','verifierRefs','clearanceRef','supersedes','sourceWitness','reportId']);
const ACTOR_KEYS='operator_id,operator_kind,role,session_ref,transport_actor';
const EVIDENCE_KEYS='decision,independent,kind,ref';
const WITNESS_KEYS='currentness,issueNumber,readback,repository,sourceSha';
const PUBLISHABLE='PUBLISHABLE';
const plain=v=>v!==null&&typeof v==='object'&&!Array.isArray(v);
const exact=(obj,keys)=>plain(obj)&&Object.keys(obj).sort().join(',')===keys;
const iso=s=>typeof s==='string'&&ISO.test(s)&&Number.isFinite(Date.parse(s));
const string=(s,max=512)=>typeof s==='string'&&s.length>0&&s.length<=max;
const nativeRef=s=>string(s,1024)&&
  /^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/(?:issues|pull|commit|releases)\/[A-Za-z0-9_.~/#-]+$/.test(s)&&
  !s.includes('/../')&&!s.includes('/./');
const refList=v=>Array.isArray(v)&&v.length<=20&&
  new Set(v).size===v.length&&v.every(nativeRef);
const denied=reason=>Object.freeze({status:'BLOCKED',reason,envelope:null,
  projectionDigest:null,dataDigest:null,authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const stable=v=>{
  if(v===null||typeof v==='string'||typeof v==='boolean')return JSON.stringify(v);
  if(Number.isSafeInteger(v))return String(v);
  if(Array.isArray(v))return '['+v.map(stable).join(',')+']';
  if(plain(v))return '{'+Object.keys(v).sort().map(k=>JSON.stringify(k)+':'+stable(v[k])).join(',')+'}';
  throw new TypeError('NON_CANONICAL_FEEDBACK_VALUE');
};
const digest=v=>createHash('sha256').update(stable(v),'utf8').digest('hex');
export function feedbackReportId({repository,issueNumber,experienceRef,
  procedureRevision,taskRef,attemptRef,phase}={}) {
  if(!REPO.test(repository??'')||!Number.isSafeInteger(issueNumber)||issueNumber<1||
    !nativeRef(experienceRef)||!ID.test(procedureRevision??'')||
    !string(taskRef)||!string(attemptRef)||!PHASE.has(phase))
    return null;
  return 'experience-use:'+digest([repository,issueNumber,experienceRef,
    procedureRevision,taskRef,attemptRef,phase]).slice(0,32);
}
/**
 * This constructs a schema-shaped candidate, not a verified publication.
 * A shape-correct envelope has no Task/K/Host/Release authority.
 */
export function projectExperienceUseFeedback(dto) {
  if(!plain(dto)||Object.keys(dto).some(k=>!KEYS.has(k))||
      JSON.stringify(dto).length>16384)return denied('DTO_UNKNOWN_OR_OVERSIZED');
  const {repository,issueNumber,sourceSha,publisherSourceSha,adsRevision,
    publisherNamespace,revision,experienceRef,procedureRevision,phase,
    epistemic,observedAt,actor,sourceRefs,phaseEvidence,verifierRefs,
    clearanceRef,supersedes,sourceWitness,taskRef,attemptRef,reportId}=dto;
  if(!REPO.test(repository??'')||!Number.isSafeInteger(issueNumber)||issueNumber<1||
     !SHA.test(sourceSha??'')||!SHA.test(publisherSourceSha??'')||
     !SHA.test(adsRevision??'')||!string(publisherNamespace,128)||
     !Number.isSafeInteger(revision)||revision<1||!nativeRef(experienceRef)||
     !ID.test(procedureRevision??'')||!string(taskRef)||!string(attemptRef)||
     !PHASE.has(phase)||!EPISTEMIC.has(epistemic)||!iso(observedAt)||
     !refList(sourceRefs)||!refList(verifierRefs)||!nativeRef(clearanceRef))
    return denied('FEEDBACK_SOURCE_IDENTITY_INVALID');
  const expectedReportId=feedbackReportId({repository,issueNumber,experienceRef,
    procedureRevision,taskRef,attemptRef,phase});
  if(reportId!==expectedReportId)return denied('FEEDBACK_REPORT_ID_MISMATCH');
  if(!exact(actor,ACTOR_KEYS)||!string(actor.role,64)||
    !string(actor.operator_kind,64)||!string(actor.operator_id,128)||
    !nativeRef(actor.session_ref)||
    !(actor.transport_actor===null||string(actor.transport_actor,128)))
    return denied('FEEDBACK_ACTOR_INVALID');
  if(!exact(sourceWitness,WITNESS_KEYS)||sourceWitness.repository!==repository||
    sourceWitness.issueNumber!==issueNumber||sourceWitness.sourceSha!==sourceSha||
    sourceWitness.currentness!=='CURRENT'||sourceWitness.readback!=='VERIFIED')
    return denied('FEEDBACK_SOURCE_NOT_CURRENT');
  if(!exact(phaseEvidence,EVIDENCE_KEYS)||phaseEvidence.kind!==phase||
    !nativeRef(phaseEvidence.ref)||!sourceRefs.includes(phaseEvidence.ref)||
    typeof phaseEvidence.independent!=='boolean'||
    !(phaseEvidence.decision===null||['APPROVE','REJECT'].includes(phaseEvidence.decision)))
    return denied('FEEDBACK_PHASE_WITNESS_INVALID');
  if((phase==='ACCEPTED'&&phaseEvidence.decision!=='APPROVE')||
    (phase==='SKIPPED'&&phaseEvidence.decision!=='REJECT')||
    (phase==='APPLIED'&&phaseEvidence.decision!==null)||
    (phase==='PRESENTED'&&phaseEvidence.decision!==null)||
    (phase==='INDEPENDENTLY_CHECKED'&&
      (!phaseEvidence.independent||epistemic!=='OBSERVED'||verifierRefs.length===0))||
    (phase==='OBSERVED_EFFECT'&&
      (!phaseEvidence.independent||epistemic!=='OBSERVED')))
    return denied('FEEDBACK_PHASE_QUALIFICATION_MISSING');
  if(phase!=='INDEPENDENTLY_CHECKED'&&verifierRefs.length!==0)
    return denied('UNQUALIFIED_VERIFIER_REF');
  if(revision===1&&supersedes!==null||
     revision>1&&(!plain(supersedes)||Object.keys(supersedes).sort().join(',')!==
     'comment_ref,data_digest,revision'||supersedes.revision!==revision-1||
      !DIGEST.test(supersedes.data_digest??'')||!nativeRef(supersedes.comment_ref)))
    return denied('FEEDBACK_REVISION_CHAIN_INVALID');
  const issueRef='https://github.com/'+repository+'/issues/'+issueNumber;
  if(revision>1&&!supersedes.comment_ref.startsWith(issueRef+'#issuecomment-'))
    return denied('FEEDBACK_SUPERSESSION_CROSS_ISSUE');
  const data={
    report_id:reportId,revision,publisher_namespace:publisherNamespace,
    report_kind:'EXPERIENCE_USE_FEEDBACK',
    subject:{repository,issue_ref:issueRef,task_ref:taskRef,
      attempt_ref:attemptRef,attempt_binding:'CANONICAL_DISPATCH',
      source_sha:sourceSha,ads_revision:adsRevision,
      ads_profile:'ads/github/v5:report',
      currentness:{state:'CURRENT',source_refs:[phaseEvidence.ref],limits:[]},
      publisher_source_sha:publisherSourceSha,source_status:'PINNED',
      source_currentness:'CURRENT',source_reason:null,
      collector_observation_refs:[]},
    actor:{role:actor.role,operator_kind:actor.operator_kind,
      operator_id:actor.operator_id,session_ref:actor.session_ref,
      transport_actor:actor.transport_actor},
    observed_at:observedAt,source_refs:sourceRefs,limits:[],supersedes,
    publisher_source_sha:publisherSourceSha,ads_subject_sha:sourceSha,
    experience_ref:experienceRef,procedure_revision:procedureRevision,phase,
    epistemic,verifier_refs:verifierRefs,
    publication:{classification:PUBLISHABLE,clearance_ref:clearanceRef,
      projection_digest:null}
  };
  // Projection digest excludes its own digest field to avoid a hash cycle.
  data.publication.projection_digest=digest({...data,
    publication:{classification:PUBLISHABLE,clearance_ref:clearanceRef}});
  const envelope={schema:'ads/github/v5',kind:'report',data};
  // The native writer separately checks the actual pinned ADS schema and
  // RFC8785 JCS(data) digest against GitHub, not just this pure DTO.
  return Object.freeze({status:'CANDIDATE_ONLY',reason:'REQUIRES_NATIVE_VALIDATION',
    envelope,projectionDigest:data.publication.projection_digest,
    dataDigest:digest(data),authorized:false,admitted:false,
    taskPass:false,releasePass:false});
}
