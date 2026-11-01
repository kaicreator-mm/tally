/**
 * Host-only ADS report projection, not task-meta publisher or Task authority.
 * A report is visible only after selected Native GitHub C verifies a pinned
 * subject, GitHub transport login, JCS hash and correction reconciliation.
 * Missing reports/telemetry coverage is explicitly UNKNOWN, not 0.
 */
const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const deny=reason=>({connected:false,reason});
const kindOf=kind=>({
  EXECUTION_SUMMARY:'ads-v5-summary',
  EXPERIENCE_USE_FEEDBACK:'feedback',
})[kind]??null;
const plain=v=>v!==null&&typeof v==='object'&&!Array.isArray(v);
export function createUiAdsReportHost({repository,versionId,expectedSha,
  nativeC,taskIssueNumbers,acceptedCommentIdsByIssue,publisherBindings,
  readReports,token}={}){
  if(!REPO.test(repository??'')||typeof versionId!=='string'||
     !/^version\/[A-Za-z0-9_.-]+$/.test(versionId)||
     !SHA.test(expectedSha??'')||
     typeof nativeC?.catalog!=='function'||!Array.isArray(taskIssueNumbers)||
     taskIssueNumbers.length<1||taskIssueNumbers.length>200||
     taskIssueNumbers.some(n=>!Number.isSafeInteger(n)||n<1)||
     new Set(taskIssueNumbers).size!==taskIssueNumbers.length||
     !plain(acceptedCommentIdsByIssue)||!plain(publisherBindings)||
     Object.keys(publisherBindings).length===0||
     !Object.values(publisherBindings).every(v=>Array.isArray(v)&&v.length>0&&
       v.every(s=>typeof s==='string'&&/^[A-Za-z0-9_.-]+(?:\[bot\])?$/.test(s))))
    throw new TypeError('TRUSTED_ADS_REPORT_SOURCE_CONFIGURATION_REQUIRED');
  const grantKeys=new Set(Object.keys(acceptedCommentIdsByIssue));
  const allowed=new Set(taskIssueNumbers.map(String));
  if([...grantKeys].some(k=>!allowed.has(k)))throw new TypeError('CROSS_TASK_ADS_GRANT');
  let selected=0;
  for(const ids of Object.values(acceptedCommentIdsByIssue)){
    if(!Array.isArray(ids)||ids.length>100||
       new Set(ids).size!==ids.length||
       ids.some(n=>!Number.isSafeInteger(n)||n<1))
      throw new TypeError('INVALID_ACCEPTED_COMMENT_IDENTITIES');
    selected+=ids.length;
  }
  if(selected===0)throw new TypeError('ADS_COMMENT_WITNESSES_MISSING');
  const [owner,repo]=repository.split('/');
  const source={repository,sourceSha:expectedSha,
    currentness:'CURRENT',readback:'VERIFIED'};
  async function metadata(scope){
    if(scope?.projectId!==repository||scope?.versionId!==versionId)
      return deny('PROJECT_VERSION_SCOPE_MISMATCH');
    try{
      const branch=await nativeC.catalog();
      if(branch?.connected!==true||
         branch.source?.sourceSha!==expectedSha||
         branch.source?.repository!==repository)
        return deny('PINNED_VERSION_SOURCE_UNAVAILABLE');
      const reader=readReports??(async args=>{
        if(typeof token!=='string'||token.length<8)
          throw new Error('HOST_ONLY_NATIVE_ADS_TOKEN_MISSING');
        const {readGitHubIssueReports}=await import(
          '../../packages/tally-github-native-c/src/read-only.mjs');
        return readGitHubIssueReports({...args,token});
      });
      const records=[],ids=new Set();
      for(const issueNumber of taskIssueNumbers){
        const grants=acceptedCommentIdsByIssue[String(issueNumber)];
        if(!grants?.length)continue; // absence stays PARTIAL, never zero PASS
        const snapshot=await reader({owner,repo,issueNumber,
          expectedSourceSha:expectedSha,publisherBindings,
          acceptedCommentIds:grants});
        if(snapshot?.status!=='CURRENT_OBSERVED'||
           !Array.isArray(snapshot.records)||snapshot.records.length!==grants.length||
           snapshot.authorized!==false||snapshot.taskPass!==false||
           snapshot.releasePass!==false)
          return deny('ADS_PUBLISHER_REPORTS_UNQUALIFIED');
        const seen=new Set();
        for(const row of snapshot.records){
          const item=row?.source;
          if(row.status!=='CURRENT_OBSERVED'||
             item?.repository!==repository||item?.issueNumber!==issueNumber||
             item.sourceSha!==expectedSha||!grants.includes(item.commentId)||
             seen.has(item.commentId)||!kindOf(item.kind)||
             typeof item.reportId!=='string'||!item.reportId||
             !Number.isSafeInteger(item.revision)||item.revision<1)
            return deny('ADS_NATIVE_REPORT_SCOPE_DRIFT');
          seen.add(item.commentId);
          const key=String(issueNumber)+':'+String(item.commentId);
          if(ids.has(key))return deny('DUPLICATE_ADS_PUBLICATION');
          ids.add(key);
          records.push({reportId:String(issueNumber)+'/'+item.reportId+
            '@'+item.revision+'#'+item.commentId,
            taskId:String(issueNumber),attemptId:null,kind:kindOf(item.kind),
            subjectSha:item.sourceSha,observedAt:null,
            evidenceRef:'https://github.com/'+repository+'/issues/'+issueNumber+
              '#issuecomment-'+item.commentId,
            independentlyReviewed:false});
        }
        if(seen.size!==grants.length)return deny('ADS_COMMENT_SET_INCOMPLETE');
      }
      const finalBranch=await nativeC.catalog();
      if(finalBranch?.connected!==true||
         finalBranch.source?.sourceSha!==expectedSha)
        return deny('PINNED_VERSION_CHANGED_DURING_READ');
      return {connected:true,source,versionId,reports:records,
        attempts:[],lineage:[],learnedOutcomes:[],
        rosterCoverage:'UNVERIFIED',reportCoverage:'PARTIAL',
        telemetryCoverage:'UNKNOWN',lineageCoverage:'UNKNOWN',
        learningCoverage:'UNKNOWN',note:'ADVISORY_ADS_ONLY'};
    }catch{return deny('ADS_NATIVE_REPORT_READ_FAILED');}
  }
  return Object.freeze({metadata});
}
