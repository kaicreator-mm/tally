/**
 * T06b: source-qualified ADS v4.18 DISPATCH_CLAIMED *candidate* observations.
 * A validated event comment is NOT an accepted CAS Claim: only the independent
 * serialized writer plus current GitHub/Host/K admission can prove that.
 *
 * All outputs are advisory and fail-closed; no effects or credentials leaked.
 */
import Ajv2020 from 'ajv/dist/2020.js';
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import {Octokit} from '@octokit/rest';
import {createNativeReadRetryFetch} from './read-retry.mjs';
import {parseDocument,visit,isAlias} from 'yaml';

const EVENT_SCHEMA_BLOB='cad9434e11febfd29657b564dbe710700c674888';
const schemaBytes=readFileSync(new URL('../vendor/agent-event-v2.schema.json',import.meta.url));
const objectSha=createHash('sha1').update(Buffer.from('blob '+schemaBytes.length+'\0')).update(schemaBytes).digest('hex');
if(objectSha!==EVENT_SCHEMA_BLOB)throw new Error('ADS_V418_EVENT_SCHEMA_BLOB_MISMATCH');
const ajv=new Ajv2020({allErrors:true,strict:false,validateFormats:false,
  removeAdditional:false,coerceTypes:false,useDefaults:false});
const validateEvent=ajv.compile(JSON.parse(schemaBytes.toString('utf8')));
const NAME=/^[A-Za-z0-9_.-]+$/, SHA=/^[a-f0-9]{40}$/, MAX=1000;
const marker='<!-- ads:github -->';
const no=(reason)=>Object.freeze({
  status:'UNKNOWN',reason,claimStatus:'NOT_VERIFIED',candidateCount:0,
  candidates:Object.freeze([]),authorized:false,admitted:false,
  taskPass:false,releasePass:false
});
const reasonFor=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===429?'GITHUB_RATE_LIMITED':'GITHUB_CLAIM_EVENT_READ_FAILED';
function parseCanonicalComment(body){
  if(typeof body!=='string')return {kind:'other'};
  if(body.length>256*1024)return {kind:'bad'};
  const lines=body.split(/\r\n|\n|\r/);
  const marks=lines.filter(x=>x.trim()===marker);
  if(marks.length===0)return {kind:'other'};
  if(lines[0]!==marker||marks.length!==1)return {kind:'bad'};
  let i=1;
  while(i<lines.length&&lines[i].trim()==='')i++;
  if(lines[i]!=='\x60\x60\x60yaml')return {kind:'bad'};
  const end=lines.indexOf('\x60\x60\x60',i+1);
  if(end<0)return {kind:'bad'};
  let envelope;
  try{
    const doc=parseDocument(lines.slice(i+1,end).join('\n'),
      {uniqueKeys:true,merge:false,schema:'core',maxAliasCount:0,strict:true});
    if(doc.errors.length||doc.warnings.length)return {kind:'bad'};
    visit(doc,{Node(_key,node){
      if(isAlias(node)||node.anchor!==undefined||node.tag!==undefined)
        throw new Error('YAML_NOT_CANONICAL');
    }});
    envelope=doc.toJS({maxAliasCount:0});
  }catch{return {kind:'bad'};}
  if(!envelope||typeof envelope!=='object'||Array.isArray(envelope)||
    Object.keys(envelope).sort().join(',')!=='data,kind,schema'||
    envelope.schema!=='ads/github/v3'||envelope.kind!=='event'||
    !validateEvent(envelope.data))return {kind:'bad'};
  return {kind:envelope.data.event==='DISPATCH_CLAIMED'?'candidate':'other',
    event:envelope.data};
}
/**
 * Exact GitHub comments are source observations, not the accepted Claim lock.
 * Event schema is strictly pinned to ADS 4.18 by its actual upstream blob.
 */
export async function readClaimEventCandidatesFromClient({
  octokit,owner,repo,issueNumber,expectedSourceSha,limit=MAX
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
    !Number.isSafeInteger(issueNumber)||issueNumber<1||
    !SHA.test(expectedSourceSha??'')||!Number.isSafeInteger(limit)||
    limit<1||limit>MAX||
    !octokit?.rest?.issues?.get||!octokit?.rest?.issues?.listComments||
    !octokit?.rest?.repos?.getCommit||!octokit?.paginate?.iterator)
    return no('INVALID_NATIVE_QUERY');
  const issueUrl=`https://github.com/${owner}/${repo}/issues/${issueNumber}`;
  const params={owner,repo,issue_number:issueNumber};
  try{
    const [initial,commit]=await Promise.all([
      octokit.rest.issues.get(params),
      octokit.rest.repos.getCommit({owner,repo,ref:expectedSourceSha})
    ]);
    const head=initial?.data;
    if(head?.number!==issueNumber||head?.html_url!==issueUrl||
      !Number.isSafeInteger(head.id)||typeof head.updated_at!=='string'||
      commit?.data?.sha!==expectedSourceSha)
      return no('NATIVE_ISSUE_OR_SHA_MISMATCH');
    const seen=new Set(),candidates=[];
    let total=0;
    for await(const page of octokit.paginate.iterator(octokit.rest.issues.listComments,
      {...params,per_page:100})){
      if(!Array.isArray(page?.data))return no('INVALID_COMMENT_PAGE');
      for(const c of page.data){
        total++;
        if(total>limit)return no('COMMENT_SCAN_LIMIT');
        const parsed=parseCanonicalComment(c?.body);
        if(parsed.kind==='other')continue;
        if(parsed.kind==='bad')return no('UNVERIFIED_ADS_EVENT_COMMENT');
        if(parsed.kind!=='candidate')continue;
        if(!Number.isSafeInteger(c.id)||c.id<1||
          c.html_url!==`${issueUrl}#issuecomment-${c.id}`||
          typeof c.created_at!=='string'||c.created_at!==c.updated_at||
          typeof c.user?.login!=='string'||!c.user.login)
          return no('NATIVE_EVENT_IDENTITY_UNVERIFIED');
        const e=parsed.event;
        // ADS event-v2 schema allows task/issue optional for generic events.
        // For a Tally Task Claim, at least one must bind this native Issue,
        // and any second asserted pointer must match too.
        if((!e.task&&!e.issue)||
          (e.task&&e.task!==`#${issueNumber}`)||
          (e.issue&&e.issue!==`#${issueNumber}`)||
          e.sha!==expectedSourceSha)
          return no('CROSS_TASK_OR_STALE_CLAIM_EVENT');
        if(seen.has(e.dispatch_id))return no('DUPLICATE_DISPATCH_CLAIM_EVENT');
        seen.add(e.dispatch_id);
        candidates.push(Object.freeze({
          commentId:c.id,commentRef:c.html_url,dispatchId:e.dispatch_id,
          actorRole:e.actor_role,operatorKind:e.operator_kind,
          operatorId:e.operator_id,sessionRef:e.session_ref??null,
          transportActor:c.user.login,sourceSha:e.sha,
          canonicalAdmissionRef:e.canonical_admission_ref??null,
          protectedClaimKey:e.protected_claim_key??null,
          claimGeneration:e.admission_generation??null,
          commentBodySha256:createHash('sha256').update(c.body,'utf8').digest('hex'),
          // Self-declared fields remain self-declared even from GH comment.
          sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',
          claimAccepted:false
        }));
      }
    }
    const final=(await octokit.rest.issues.get(params))?.data;
    if(final?.id!==head.id||final?.number!==issueNumber||
      final?.html_url!==issueUrl||final?.updated_at!==head.updated_at)
      return no('NATIVE_ISSUE_CHANGED_DURING_READ');
    return Object.freeze({
      ...no(candidates.length?'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED':'NO_DISPATCH_CLAIM_EVENTS'),
      candidateCount:candidates.length,candidates:Object.freeze(candidates)
    });
  }catch(e){return no(reasonFor(e));}
}
export async function readGitHubClaimEventCandidates({
  token,owner,repo,issueNumber,expectedSourceSha,limit
}={}){
  if(typeof token!=='string'||token.length<8)return no('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000,fetch:createNativeReadRetryFetch()}});
  return readClaimEventCandidatesFromClient({
    octokit,owner,repo,issueNumber,expectedSourceSha,limit
  });
}
