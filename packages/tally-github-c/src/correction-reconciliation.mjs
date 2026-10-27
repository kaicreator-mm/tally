/**
 * Tally v2 T03c — pure ADS v5 native correction target reconciliation.
 *
 * The native GitHub C is responsible for full pagination, publisher bindings,
 * immutable source lookup, actual JCS data hashes and pinned schema checks
 * BEFORE these observations reach this function. This code never trusts a
 * comment's self-asserted authority. No K admission / Task / Release PASS.
 */
import {projectReportSet, STATES} from './report-projection.mjs';

const SUMMARY='EXECUTION_SUMMARY';
const FEEDBACK='EXPERIENCE_USE_FEEDBACK';
const CORRECTION='EXECUTION_CORRECTION';
const NOTICES=new Set(['REVOKE','REDACT','DISPUTE']);
const issueRef=(repo,n)=>`https://github.com/${repo}/issues/${n}`;
const plain=x=>!!x&&typeof x==='object'&&!Array.isArray(x);
const safeId=n=>Number.isSafeInteger(n)&&n>0;
const safeDigest=x=>typeof x==='string'&&/^[a-f0-9]{64}$/.test(x);
const safeSha=x=>typeof x==='string'&&/^[a-f0-9]{40}$/.test(x);
const empty=Object.freeze([]);
const DENIED=Object.freeze({authorized:false,taskPass:false,releasePass:false});

/** Everything in target identity is an exact equality; no loose alias match. */
function sameTarget(target, observation) {
  if(!plain(target)||!plain(observation?.native)||!plain(observation?.envelope?.data))return false;
  const n=observation.native,d=observation.envelope.data,s=d.subject;
  if(!plain(s)||!safeDigest(n.dataDigest)||!safeSha(n.sourceSha)||
     !safeId(n.commentId)||!safeId(n.issueNumber))return false;
  const ref=issueRef(n.repository,n.issueNumber);
  return (d.report_kind===SUMMARY||d.report_kind===FEEDBACK)&&
    target.repository===n.repository&&
    target.issue_ref===ref&&
    target.comment_ref===`${ref}#issuecomment-${n.commentId}`&&
    target.comment_id===n.commentId&&
    target.report_id===d.report_id&&
    target.revision===d.revision&&
    target.report_kind===d.report_kind&&
    target.publisher_namespace===d.publisher_namespace&&
    target.task_ref===s.task_ref&&
    target.attempt_ref===s.attempt_ref&&
    target.data_digest===n.dataDigest&&
    target.ads_profile===s.ads_profile&&
    target.target_source_status===s.source_status&&
    target.target_source_sha===s.source_sha&&
    s.source_sha===n.sourceSha;
}

const targetKey=n=>JSON.stringify([n.repository,n.issueNumber,n.commentId]);
function rowFor(n,data,status,reason,notice=null) {
  return Object.freeze({
    repository:n?.repository??null,
    issueNumber:n?.issueNumber??null,
    commentId:n?.commentId??null,
    reportId:data?.report_id??null,
    revision:data?.revision??null,
    kind:data?.report_kind??null,
    status,reason,notice
  });
}
function setResult(projected,rows,ledger,unresolved) {
  return Object.freeze({
    ...projected,
    // A valid notice does not restore CURRENT. Publication supersession,
    // independent reviewer approval and Task Claim remain external authorities.
    status:projected.status===STATES.CONFLICT?STATES.CONFLICT:STATES.UNKNOWN,
    reconciliation:unresolved?'UNRESOLVED':'NOTICES_APPLIED_ADVISORY',
    corrections:Object.freeze(ledger),effectiveRecords:Object.freeze(rows),
    ...DENIED
  });
}

/**
 * Incomplete/invalid native data must never lead to a claim that a correction
 * was applied. The entire set remains UNKNOWN with a typed mismatch reason.
 * Every row reports only identity and status, never raw provider data.
 */
export function reconcileCorrectionNotices(observations) {
  const base=projectReportSet(observations);
  if(!Array.isArray(observations)||observations.length>1000||
      base.status===STATES.INVALID)return Object.freeze({
    ...base,reconciliation:'INVALID_INPUT',corrections:empty,
    effectiveRecords:empty,...DENIED
  });
  const notices=observations.filter(o=>o?.envelope?.data?.report_kind===CORRECTION);
  if(notices.length===0) {
    // Preserve previous native-reader behavior byte-for-byte for regular
    // report sets while surfacing the new effective-view metadata.
    const rows=observations.map(o=>{
      const d=o?.envelope?.data,n=o?.native;
      const src=base.records.find(x=>x.source?.commentId===n?.commentId);
      const safe=src?.status===STATES.CURRENT;
      return rowFor(n,d,safe?'OBSERVED_ONLY':'NOT_CURRENT',
        safe?'ADVISORY_NOT_TASK_AUTHORITY':'SOURCE_NOT_CURRENT');
    });
    return Object.freeze({...base,reconciliation:'NO_NOTICES',corrections:empty,
      effectiveRecords:Object.freeze(rows),...DENIED});
  }
  const originals=new Map();
  for(const o of observations)if(o?.envelope?.data?.report_kind!==CORRECTION){
    const n=o.native;if(n) {
      const key=targetKey(n);
      if(!originals.has(key))originals.set(key,[]);
      originals.get(key).push(o);
    }
  }
  const ledger=[],touched=new Map(),unresolved=new Set();
  for(const c of notices) {
    const n=c.native,d=c.envelope.data,t=d.target;
    const key=plain(t)?JSON.stringify([t.repository,c.native?.issueNumber,t.comment_id]):'MISSING_TARGET';
    let why=null,target=null;
    if(!plain(t)||!NOTICES.has(d.notice)||!safeId(n?.commentId)||!safeId(n?.issueNumber)||
       n?.currentness!=='CURRENT'||n?.readback!=='VERIFIED'||
       n?.permission!=='GRANTED'||n?.pagination!=='COMPLETE')why='NOTICE_NOT_CURRENT_OR_INVALID';
    else if(t.repository!==n.repository||t.issue_ref!==issueRef(n.repository,n.issueNumber)||
      t.publisher_namespace!==d.publisher_namespace ||
      d.subject?.repository!==n.repository ||
      d.subject?.issue_ref!==t.issue_ref ||
      d.subject?.source_sha!==n.sourceSha)why='NOTICE_CROSS_SCOPE_OR_PUBLISHER';
    else {
      const candidates=originals.get(key)||[];
      if(candidates.length!==1||!sameTarget(t,candidates[0]))why='NOTICE_TARGET_UNVERIFIED';
      else target=candidates[0];
    }
    const targetIdentity=target?.native?targetKey(target.native):null;
    if(targetIdentity) {
      if(touched.has(targetIdentity)) {
        unresolved.add(targetIdentity);
        why='DUPLICATE_OR_CONFLICTING_NOTICE';
      }else touched.set(targetIdentity,d.notice);
    }
    if(why)unresolved.add(key);
    ledger.push(rowFor(n,d,why?'UNRESOLVED':'MATCHED',why??'TARGET_IDENTITY_MATCHED',d.notice));
  }
  const rows=observations.filter(o=>o?.envelope?.data?.report_kind!==CORRECTION).map(o=>{
    const d=o?.envelope?.data,n=o?.native,key=n?targetKey(n):null;
    const notice=key?touched.get(key):undefined;
    if(key&&unresolved.has(key))return rowFor(n,d,'NOT_CURRENT','CONFLICTING_NOTICE');
    if(notice&&NOTICES.has(notice))return rowFor(n,d,
      notice==='REVOKE'?'REVOKED':notice==='REDACT'?'REDACTED':'DISPUTED',
      'MATCHED_NOTICE',notice);
    return rowFor(n,d,'NOT_CURRENT','CORRECTION_SET_NOT_QUALIFIED');
  });
  return setResult(base,rows,ledger,unresolved.size>0);
}
