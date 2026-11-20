/** Build/Host-only selected single-Builder target; never derived from UI data. */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const REF=/^(?:main|version\/[A-Za-z0-9_.-]+)$/;
const ISSUE=/^[1-9][0-9]{0,9}$/;
function validPath(path){
  if(typeof path!=='string'||path.length<3||path.length>220||
     path.startsWith('/')||path.includes('\\')||
     path.includes('\0')||path.includes(':')||path.includes('\n'))
    return false;
  const parts=path.split('/');
  return parts.length>=1&&parts.every(p=>p.length>0&&p!=='.'&&p!=='..'&&
    p!=='.git'&&/^[A-Za-z0-9_.-]+$/.test(p));
}
export function resolveBuilderBuildTarget(env={}){
  const names=['T06M_TARGET_REPOSITORY','T06M_TARGET_ISSUE_NUMBER',
    'T06M_TARGET_VERSION_REF','T06M_TARGET_CANDIDATE_PATH'];
  const present=names.filter(name=>env[name]!==undefined);
  if(present.length!==0&&present.length!==names.length)
    throw new TypeError('T06M_TARGET_PARTIAL');
  if(present.length===0&&env.T06L_CLAIM==='1'&&
     env.T06L_TARGET_REPOSITORY!==undefined)
    throw new TypeError('T06M_CLAIM_BUILD_TARGET_MISMATCH');
  if(present.length===0)
    return Object.freeze({mode:'HISTORICAL_DEFAULT',
      repository:'kaicreator-mm/tally',owner:'kaicreator-mm',repo:'tally',
      versionId:'version/v2',issueNumber:null,candidateRelativePath:null});
  const repository=env.T06M_TARGET_REPOSITORY;
  const versionId=env.T06M_TARGET_VERSION_REF;
  const strIssue=env.T06M_TARGET_ISSUE_NUMBER;
  const candidateRelativePath=env.T06M_TARGET_CANDIDATE_PATH;
  if(!REPO.test(repository??'')||!REF.test(versionId??'')||
     !ISSUE.test(strIssue??'')||
     !Number.isSafeInteger(Number(strIssue))||!validPath(candidateRelativePath))
    throw new TypeError('T06M_TARGET_INVALID');
  if(env.T06L_CLAIM==='1'&&
      (env.T06L_TARGET_REPOSITORY!==repository||
       env.T06L_TARGET_ISSUE_NUMBER!==strIssue))
    throw new TypeError('T06M_CLAIM_BUILD_TARGET_MISMATCH');
  const [owner,repo]=repository.split('/');
  return Object.freeze({mode:'SELECTED_BUILD',owner,repo,repository,versionId,
    issueNumber:Number(strIssue),candidateRelativePath});
}
export function sealBuilderShimTarget(source,target){
  if(typeof source!=='string'||!source.includes("const OWNER='kaicreator-mm',REPO='tally';"))
    throw new TypeError('T06M_SHIM_TARGET_ANCHOR_MISSING');
  if(target?.mode==='HISTORICAL_DEFAULT')return source;
  if(target?.mode!=='SELECTED_BUILD')
    throw new TypeError('T06M_SHIM_TARGET_INVALID');
  const old="const OWNER='kaicreator-mm',REPO='tally';";
  const literal="const OWNER="+JSON.stringify(target.owner)+
    ",REPO="+JSON.stringify(target.repo)+";";
  let sealed=source.replace(old,literal);
  // For an alternate BUILD target, do not inherit the historical or
  // runtime-process-selected issue allowlist from the demo.
  const start=sealed.indexOf('const ISSUES=new Set(');
  const end=sealed.indexOf('const REVIEWER_ISSUE=',start);
  if(start<0||end<start)throw new TypeError('T06M_SHIM_ISSUE_BINDING_MISSING');
  sealed=sealed.slice(0,start)+
    'const ISSUES=new Set(['+target.issueNumber+']);\n'+sealed.slice(end);
  return sealed;
}
export function exactSelectedWriteSet(value,{repository,issueNumber,sourceSha,
  digest,candidateRelativePath}={}){
  if(value?.phase!=='FROZEN_DAG_WRITESET'||
     value.repository!==repository||value.issueNumber!==issueNumber||
     value.sourceSha!==sourceSha||value.writeSetSha256!==digest||
     typeof value.document!=='string')return false;
  let paths;
  try{
    const parsed=JSON.parse(value.document);
    paths=Array.isArray(parsed)?parsed:parsed?.allowed_write_set;
  }catch{
    paths=value.document.split(/\r?\n/).filter(Boolean);
  }
  return Array.isArray(paths)&&paths.length>0&&paths.length<128&&
    paths.every(validPath)&&new Set(paths).size===paths.length&&
    paths.includes(candidateRelativePath);
}
