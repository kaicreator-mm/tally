/**
 * Host BUILD-only Claim target binding.
 *
 * A selected repository/Issue is compiled into BOTH the BUILD-sealed C and
 * the digest-pinned external provider. UI/request input never reaches this
 * module. A new target requires rebuilding/resealing before use; changing
 * runtime environment cannot retarget an already sealed capability.
 */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const NUMBER=/^[1-9][0-9]{0,9}$/;
const REF_MARKER='__T06L_TARGET_ISSUE_REF__';
const CLAIM_MARKER='__T06L_TARGET_CLAIM_KEY__';
export function resolveClaimBuildTarget(env=process.env){
  const repo=env.T06L_TARGET_REPOSITORY;
  const issue=env.T06L_TARGET_ISSUE_NUMBER;
  if((repo===undefined)!==(issue===undefined))
    throw new TypeError('T06L_TARGET_PARTIAL');
  if(repo!==undefined&&(!REPO.test(repo)||typeof issue!=='string'||
     !NUMBER.test(issue)||!Number.isSafeInteger(Number(issue))))
    throw new TypeError('T06L_TARGET_INVALID');
  const repository=repo??'kaicreator-mm/tally';
  const issueNumber=Number(issue??'169');
  const [owner,name]=repository.split('/');
  return Object.freeze({
    repository,owner,repo:name,issueNumber,
    issueRef:'https://github.com/'+repository+'/issues/'+issueNumber,
    protectedClaimKey:repository+'#'+issueNumber+':builder:__default__',
    mode:repo===undefined?'HISTORICAL_DEFAULT':'SELECTED_BUILD'
  });
}
export function sealClaimShimTarget(source,target){
  if(typeof source!=='string'||!source.includes(REF_MARKER)||
     !source.includes(CLAIM_MARKER)||typeof target?.issueRef!=='string'||
     typeof target?.protectedClaimKey!=='string')
    throw new TypeError('T06L_SHIM_TARGET_MARKER_MISSING');
  const sealed=source.replaceAll(REF_MARKER,target.issueRef)
    .replaceAll(CLAIM_MARKER,target.protectedClaimKey);
  if(sealed.includes(REF_MARKER)||sealed.includes(CLAIM_MARKER))
    throw new Error('T06L_SHIM_TARGET_UNSEALED');
  return sealed;
}
