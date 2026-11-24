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

const MANIFEST_PREFIX='// TALLY_T06L_BUILD_TARGET_V1=';
// A marked frame is prefix + canonical base64 metadata + LF, at most this many
// bytes including the LF. Writer and reader share the bound; nothing past it is
// inspected, so body bytes are never decoded for this check.
const MANIFEST_FRAME_MAX=4096;
function manifestValue(target){
  return {repository:target.repository,issueNumber:target.issueNumber,
    protectedClaimKey:target.protectedClaimKey};
}
/** Marker becomes part of the exact provider SHA256; never a runtime grant. */
export function markClaimProviderTarget(code,target){
  if(typeof code!=='string'||typeof target?.repository!=='string'||
     !Number.isSafeInteger(target.issueNumber))
    throw new TypeError('T06L_PROVIDER_TARGET_INVALID');
  const encoded=Buffer.from(JSON.stringify(manifestValue(target)),'utf8')
    .toString('base64');
  const header=MANIFEST_PREFIX+encoded+'\n';
  if(Buffer.byteLength(header)>MANIFEST_FRAME_MAX)
    throw new TypeError('T06L_PROVIDER_TARGET_INVALID');
  return header+code;
}
/** Catch a producer/provider target mismatch before BUILD seals an app.
 * Legacy absence is accepted only for historical #169 fixtures.
 * Only the first MANIFEST_FRAME_MAX bytes are inspected: a marked header that
 * is not LF-terminated inside that frame is invalid, never truncated.
 */
export function assertClaimProviderTarget(bytes,target){
  if(!Buffer.isBuffer(bytes))throw new TypeError('T06L_PROVIDER_BYTES_REQUIRED');
  const frame=bytes.toString('utf8',0,MANIFEST_FRAME_MAX);
  if(!frame.startsWith(MANIFEST_PREFIX)){
    if(target.mode==='HISTORICAL_DEFAULT')return;
    throw new TypeError('T06L_PROVIDER_TARGET_MANIFEST_MISSING');
  }
  let metadata=null;
  try{
    const end=frame.indexOf('\n');
    if(end<0)throw new Error('MANIFEST_FRAME_UNTERMINATED');
    const encoded=frame.slice(MANIFEST_PREFIX.length,end);
    if(!/^[A-Za-z0-9+/]+={0,2}$/.test(encoded)||
       Buffer.from(encoded,'base64').toString('base64')!==encoded)
      throw new Error('MANIFEST_ENCODING_INVALID');
    metadata=JSON.parse(Buffer.from(encoded,'base64').toString('utf8'));
  }catch{throw new TypeError('T06L_PROVIDER_TARGET_MANIFEST_INVALID');}
  if(JSON.stringify(metadata)!==JSON.stringify(manifestValue(target)))
    throw new TypeError('T06L_PROVIDER_TARGET_MISMATCH');
}
