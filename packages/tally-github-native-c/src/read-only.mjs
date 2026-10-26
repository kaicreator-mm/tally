/**
 * Tally v2 T03b — provider-read-only GitHub native C transport.
 *
 * No GitHub writes, no Task/Review/Release authority, no private task-meta.
 * Octokit handles HTTP, GitHub auth, pagination, errors and retry semantics.
 * Every currentness fact below is only an advisory read of one pinned source.
 */
import { createHash } from 'node:crypto';
import canonicalize from 'canonicalize';
import { parseDocument, visit, isAlias } from 'yaml';
import { Octokit } from '@octokit/rest';
import { validateAdsV5Report, ADS_REPORT_DIALECT } from '../../tally-ads-v5-validator/src/validate.mjs';
import { projectReportSet, STATES } from '../../tally-github-c/src/report-projection.mjs';

const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+$/;
const maximumBody=256*1024;
const maxRows=1000;
const deny=(reason, status=STATES.UNKNOWN, observationCount=0)=>Object.freeze({
  status, reason, observationCount, records:Object.freeze([]), anomalies:Object.freeze([]),
  authorized:false, taskPass:false, releasePass:false,
});
const providerError=(e)=>e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===429?'GITHUB_RATE_LIMITED':'GITHUB_NATIVE_READ_FAILED';

function reportCandidate(comment) {
  const raw=comment.body;
  if(typeof raw!=='string') return { kind:'unrelated' };
  // ADS v4.18 §16 wire: one first-line marker + one fenced YAML mapping.
  // Human prose after the close fence is ignored, never another authority.
  const marker='<!-- ads:github -->';
  if(!raw.startsWith(marker+'\n')) return {kind:'unrelated'};
  if(raw.length>maximumBody||raw.split(marker).length!==2) return {kind:'invalid'};
  const match=/^<!-- ads:github -->\n```yaml\n([\s\S]*?)\n```(?=\n|$)/.exec(raw);
  if(!match) return {kind:'invalid'};
  let obj;
  try {
    const doc=parseDocument(match[1],{uniqueKeys:true,merge:false,schema:'core',maxAliasCount:0,strict:true});
    if(doc.errors.length||doc.warnings.length) return {kind:'invalid'};
    visit(doc,{Node(_key,node){
      if(isAlias(node)||node.anchor!==undefined||node.tag!==undefined){
        throw new Error('YAML_ALIAS_ANCHOR_OR_TAG_DENIED');
      }
    }});
    obj=doc.toJS({maxAliasCount:0});
    if(!obj||typeof obj!=='object'||Array.isArray(obj)||
       Object.keys(obj).sort().join(',')!=='data,kind,schema') return {kind:'invalid'};
  }catch{return {kind:'invalid'};}
  if(obj.schema!==ADS_REPORT_DIALECT||obj.kind!=='report') return {kind:'unrelated'};
  return {kind:'report',envelope:obj};
}

/**
 * Internal transport seam: injected Octokit client supports CI fakes only.
 * Caller-supplied booleans/digests/validation are never taken as source truth.
 * Real callers use readGitHubIssueReports() which constructs authenticated
 * Octokit internally from one external token.
 */
export async function readIssueReportsFromClient({octokit,owner,repo,issueNumber,expectedSourceSha,publisherBindings,limit=maxRows}) {
  if(!REPO.test(owner??'')||!REPO.test(repo??'')||!Number.isSafeInteger(issueNumber)||
     issueNumber<1||!SHA.test(expectedSourceSha??'')||!Number.isSafeInteger(limit)||limit<1||limit>maxRows) {
    return deny('INVALID_NATIVE_QUERY', STATES.INVALID);
  }
  // This trust map is supplied by a Host-owned configuration, never by a
  // GitHub comment's self-declared publisher_namespace or logical actor.
  // It binds each namespace to the permitted transport login(s).
  const bindings=publisherBindings;
  if(!bindings||typeof bindings!=='object'||Array.isArray(bindings)||
    !Object.values(bindings).every(logins=>Array.isArray(logins)&&logins.length>0&&
      logins.every(login=>typeof login==='string'&&REPO.test(login)))) {
    return deny('PUBLISHER_BINDINGS_UNAVAILABLE');
  }
  const repository=`${owner}/${repo}`;
  const issueRef=`https://github.com/${repository}/issues/${issueNumber}`;
  const params={owner,repo,issue_number:issueNumber};
  const records=[];
  let invalid=0, readCount=0;
  try {
    const [issue,commit]=await Promise.all([
      octokit.rest.issues.get(params),
      octokit.rest.repos.getCommit({owner,repo,ref:expectedSourceSha}),
    ]);
    if(issue?.data?.number!==issueNumber || commit?.data?.sha!==expectedSourceSha) {
      return deny('NATIVE_ISSUE_OR_COMMIT_MISMATCH');
    }
    // GitHub's iterator consumes every Link page. Never treat a truncated
    // list or failed page as a complete issue observation.
    for await (const page of octokit.paginate.iterator(octokit.rest.issues.listComments,
      {...params,per_page:100})) {
      if(!Array.isArray(page?.data)) return deny('INVALID_GITHUB_PAGE');
      for(const comment of page.data) {
        readCount++;
        if(readCount>limit) return deny('GITHUB_READ_LIMIT_REACHED',STATES.PARTIAL,readCount);
        // Untrusted GitHub commenters cannot poison the entire Issue set by
        // posting malformed marker text or fake correction notices.
        const login=comment?.user?.login;
        if(typeof login!=='string'||!Object.values(bindings).some(logins=>logins.includes(login))) continue;
        const candidate=reportCandidate(comment);
        if(candidate.kind==='unrelated') continue;
        if(candidate.kind==='invalid'){invalid++;continue;}
        const envelope=candidate.envelope;
        const checked=validateAdsV5Report(envelope);
        if(!checked.valid){invalid++;continue;}
        const data=envelope.data;
        if(!bindings[data.publisher_namespace]?.includes(login)) {
          invalid++;continue;
        }
        const subject=data.subject;
        const publication=data.publication;
        // No caller can invent a repository/Issue binding, source hash, native
        // page completeness or identity; prove these against this REST result.
        if(subject?.repository!==repository||subject?.issue_ref!==issueRef||
           subject?.source_sha!==expectedSourceSha||
           publication?.classification!=='PUBLISHABLE'||
           !Number.isSafeInteger(comment?.id)||comment.id<1||
           typeof comment?.created_at!=='string'||typeof comment?.updated_at!=='string'||
           comment?.html_url!==`${issueRef}#issuecomment-${comment.id}`) {
          invalid++;continue;
        }
        const current=comment.created_at===comment.updated_at &&
          subject.source_currentness==='CURRENT'&&subject.source_status==='PINNED'&&
          subject.currentness?.state==='CURRENT'&&
          Array.isArray(subject.currentness?.limits)&&subject.currentness.limits.length===0&&
          Array.isArray(data.limits)&&data.limits.length===0&&
          data.supersedes===null;
        // ADS §16.5: lowercase SHA256(RFC8785_JCS(validated DATA)), not the
        // surrounding schema/kind envelope. No ad-hoc prefix in native digest.
        const digest=createHash('sha256').update(canonicalize(envelope.data)).digest('hex');
        records.push({
          envelope,
          validation:{schema:ADS_REPORT_DIALECT,passed:checked.valid,schemaBlob:checked.schemaBlob},
          native:{repository,issueNumber,commentId:comment.id,
            sourceSha:expectedSourceSha,dataDigest:digest,permission:'GRANTED',
            readback:'VERIFIED',pagination:'COMPLETE',
            currentness:current?'CURRENT':'STALE'}
        });
      }
    }
  }catch(e) {
    return deny(providerError(e), STATES.UNKNOWN,readCount);
  }
  if(invalid) return deny('INVALID_OR_UNVERIFIED_REPORT_COMMENT',STATES.UNKNOWN,readCount);
  if(records.length===0) return deny('NO_ADS_V5_REPORTS',STATES.UNKNOWN,readCount);
  const projection=projectReportSet(records);
  return Object.freeze({
    ...projection,reason:projection.status===STATES.CURRENT?'GITHUB_NATIVE_REPORTS_OBSERVED':
      'ADVISORY_REPORTS_NOT_CURRENT',
    observationCount:readCount,reportCount:records.length,
    authorized:false,taskPass:false,releasePass:false
  });
}

/** Public entrypoint. Missing token never silently becomes anonymous access. */
export async function readGitHubIssueReports({token,owner,repo,issueNumber,expectedSourceSha,publisherBindings,limit}={}) {
  if(typeof token!=='string'||token.length<8) return deny('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return readIssueReportsFromClient({octokit,owner,repo,issueNumber,expectedSourceSha,publisherBindings,limit});
}
