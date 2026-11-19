/** #489 J08: canonical GitHub defect -> immutable prior-tag observation.
 * READ ONLY: no issue creation, Claim, CAS, Release qualification or Task effect.
 * This is a Host-owned source read, not browser-provided evidence.
 */
const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+$/;
const BODY_MARKER=/^maintenance-marker: tally:maintenance:219:[a-f0-9]{64}$/m;
const DEFECT=437;
const SOURCE=219;
const REQUIREMENT=392;

const fail=reason=>Object.freeze({connected:false,reason});
const field=(body,label)=>{
  const matches=[...body.matchAll(new RegExp('^- '+label+': ([^\\r\\n]+)$','gm'))];
  return matches.length===1?matches[0][1].trim():null;
};
const issueUrl=(owner,repo,number)=>
  `https://github.com/${owner}/${repo}/issues/${number}`;

/** Only a strict, currently observed official GitHub issue is accepted. */
export function parseMaintenanceDefectIssue({issue,owner,repo}={}){
  if(!REPO.test(owner??'')||!REPO.test(repo??'')||
      issue?.number!==DEFECT||
      issue?.html_url!==issueUrl(owner,repo,DEFECT)||
      !['open','closed'].includes(issue?.state)||!!issue?.pull_request||
      typeof issue?.body!=='string'||!BODY_MARKER.test(issue.body)||
      typeof issue?.created_at!=='string'||
      !Number.isFinite(Date.parse(issue.created_at)))
    return null;
  const source=field(issue.body,'source');
  const requirement=field(issue.body,'requirement');
  const subjectSha=field(issue.body,'subject');
  const version=field(issue.body,'reported release');
  const summary=field(issue.body,'summary');
  if(source!==issueUrl(owner,repo,SOURCE)||
     requirement!==issueUrl(owner,repo,REQUIREMENT)||
     !SHA.test(subjectSha??'')||version!=='v1.5.5'||
     !summary||summary.length>512)return null;
  return Object.freeze({issueRef:issueUrl(owner,repo,DEFECT),
    requirementRef:requirement,subjectSha,summary,observedAt:issue.created_at});
}

/**
 * The pinned branch HEAD is re-read on each request. Git tags (including
 * annotated tags) are independently read for prior immutable commit identity.
 * A tag alone is NEVER evidence of a GitHub Release object.
 */
export function createMaintenanceReadHost({octokit,owner,repo,versionId,
  expectedSha,now=()=>new Date().toISOString()}={}){
  if(!REPO.test(owner??'')||!REPO.test(repo??'')||
      !/^version\/[A-Za-z0-9_.-]+$/.test(versionId??'')||
      !SHA.test(expectedSha??'')||typeof now!=='function'||
      typeof octokit?.rest?.repos?.getBranch!=='function'||
      typeof octokit?.rest?.issues?.get!=='function'||
      typeof octokit?.rest?.git?.getRef!=='function')
    throw new TypeError('MAINTENANCE_READ_HOST_CONFIG_INVALID');
  async function readMaintenanceDefects({sourceSha}={}){
    if(sourceSha!==expectedSha)return fail('SOURCE_SHA_MISMATCH');
    try{
      const branch=(await octokit.rest.repos.getBranch({owner,repo,
        branch:versionId}))?.data;
      if(branch?.commit?.sha!==expectedSha)
        return fail('VERSION_BRANCH_SOURCE_STALE');
      const [issue,tag]=await Promise.all([
        octokit.rest.issues.get({owner,repo,issue_number:DEFECT}),
        octokit.rest.git.getRef({owner,repo,ref:'tags/v1.5.5'})
      ]);
      const defect=parseMaintenanceDefectIssue({issue:issue?.data,owner,repo});
      if(!defect)return fail('DEFECT_SOURCE_UNVERIFIED');
      let object=tag?.data?.object;
      if(object?.type==='tag'){
        if(typeof octokit.rest.git.getTag!=='function')
          return fail('PRIOR_RELEASE_TAG_UNVERIFIED');
        object=(await octokit.rest.git.getTag({owner,repo,
          tag_sha:object.sha}))?.data?.object;
      }
      if(object?.type!=='commit'||!SHA.test(object.sha??''))
        return fail('PRIOR_RELEASE_TAG_UNVERIFIED');
      const observedAsOf=now();
      if(!Number.isFinite(Date.parse(observedAsOf))||
          Date.parse(defect.observedAt)>Date.parse(observedAsOf))
        return fail('DEFECT_OBSERVATION_TIME_INVALID');
      return Object.freeze({connected:true,repository:`${owner}/${repo}`,
        sourceSha:expectedSha,observedAsOf,
        priorRelease:{version:'v1.5.5',commitSha:object.sha,
          evidenceRef:`https://github.com/${owner}/${repo}/releases/tag/v1.5.5`},
        priorReleaseEvidence:'REPORTED_RELEASE_UNVERIFIED',
        defects:Object.freeze([defect]),
        readOnly:true,authorized:false,taskPass:false,releasePass:false});
    }catch{return fail('MAINTENANCE_GITHUB_READ_FAILED');}
  }
  return Object.freeze({readMaintenanceDefects});
}
