/**
 * T05a: ACTUAL Github-hosted ADS v5 transport readback, synthetic fixture.
 * Contract/source validation only. Never a Task Claim, Review or Release.
 */
import assert from 'node:assert/strict';
import {readGitHubIssueReports} from '../../../packages/tally-github-native-c/src/read-only.mjs';

const sourceSha='6356c96a45fcc2b7ee91642b1d87080f940b15af';
const commentId=6434110419;
const namespace='tally:kaicreator-mm/tally';
const args={token:process.env.GITHUB_TOKEN,
  owner:'kaicreator-mm',repo:'tally',issueNumber:121,
  expectedSourceSha:sourceSha,
  publisherBindings:{[namespace]:['kaicreator-mm']},
  limit:1000};
assert.ok(typeof args.token==='string'&&args.token.length>=8,'READ_ONLY_GITHUB_TOKEN_REQUIRED');

const live=await readGitHubIssueReports(args);
if(live.reportCount!==1) {
  // Diagnostic is ONLY in this standalone read-only fixture lane; keep
  // production GitHub native reader untouched and never print credentials.
  const {createRequire}=await import('node:module');
  const {pathToFileURL}=await import('node:url');
  const rq=createRequire(new URL('../../../packages/tally-github-native-c/package.json',import.meta.url));
  const {Octokit}=await import(pathToFileURL(rq.resolve('@octokit/rest')).href);
  const {parseDocument}=rq('yaml');
  const client=new Octokit({auth:args.token});
  const comments=await client.rest.issues.listComments({owner:args.owner,repo:args.repo,issue_number:121,per_page:100});
  const comment=comments.data.find(x=>x.id===commentId);
  assert.ok(comment,'real fixture comment missing');
  const match=/^<!-- ads:github -->\\n```yaml\\n([\\s\\S]*?)\\n```(?:\\n|$)/.exec(comment.body);
  const doc=match?parseDocument(match[1],{uniqueKeys:true,merge:false,schema:'core',maxAliasCount:0,strict:true}):null;
  const parsed=doc?.toJS({maxAliasCount:0});
  const {validateAdsV5Report}=await import('../../../packages/tally-ads-v5-validator/src/validate.mjs');
  const checked=parsed?validateAdsV5Report(parsed):{valid:false,code:'WIRE_PARSE_FAILED'};
  console.error(JSON.stringify({fixtureDiagnostics:true,nativeReason:live.reason,
    marker:comment.body.startsWith('<!-- ads:github -->\\n'),
    yamlErrors:doc?.errors?.map(x=>x.message).slice(0,8),
    yamlWarnings:doc?.warnings?.map(x=>x.message).slice(0,8),
    resultKeys:parsed?Object.keys(parsed):[],
    valid:checked.valid,code:checked.code,violations:checked.violations?.slice(0,12),
    commentEdited:comment.created_at!==comment.updated_at}));
}
assert.equal(live.reportCount,1,JSON.stringify({status:live.status,reason:live.reason}));
assert.equal(live.status,'UNKNOWN','synthetic report must NEVER be CURRENT');
assert.equal(live.reason,'ADVISORY_REPORTS_NOT_CURRENT');
assert.equal(live.records?.length,1);
const row=live.records[0];
assert.equal(row.status,'STALE','self-declared fixture limits and UNKNOWN currentness are not approval');
assert.equal(row.reason,'NATIVE_SOURCE_STALE');
assert.equal(row.source?.commentId,commentId,'real GitHub comment identity must be checked');
assert.equal(row.source?.namespace,namespace);
assert.equal(row.source?.sourceSha,sourceSha);
assert.match(row.source?.dataDigest??'',/^[0-9a-f]{64}$/,'actual native data-only JCS digest');
assert.equal(live.authorized,false);
assert.equal(live.taskPass,false);
assert.equal(live.releasePass,false);
assert.equal(live.observationCount>=1,true);

// Same physical authenticated issue reader, but Host refuses a publisher
// account that does not own this namespace. No semantic Claim can be forged.
const unrelated=await readGitHubIssueReports({
  ...args,publisherBindings:{[namespace]:['github-actions[bot]']}});
assert.equal(unrelated.status,'UNKNOWN');
assert.equal(unrelated.reason,'NO_ADS_V5_REPORTS');
assert.equal(unrelated.taskPass,false);
console.log(JSON.stringify({
  source:'actual-github-comment',syntheticOnly:true,
  issue:121,commentId,observedCommentCount:live.observationCount,
  reportCount:live.reportCount,validatedNativeDigest:row.source.dataDigest,
  observedStatus:live.status,recordStatus:row.status,
  unboundPublisherDenied:true,taskPass:false,releasePass:false
}));
console.log('T05A_REAL_GITHUB_V5_FIXTURE_READBACK=PASS');
