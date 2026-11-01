/**
 * Native GitHub C → UI Audit OBSERVATIONS.
 * ADS v3 claim event candidates are NOT CAS accepted Claims, Agent Attempts,
 * Task PASS, independent Reviews or Release readiness.
 */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const deny=reason=>({connected:false,reason});
export function createUiNativeAuditHost({repository,versionId,expectedSha,
  taskIssueNumbers,nativeC,token,readPair}={}){
  if(!REPO.test(repository??'')||!/^version\/[A-Za-z0-9_.-]+$/.test(versionId??'')||
     !SHA.test(expectedSha??'')||!Array.isArray(taskIssueNumbers)||
     taskIssueNumbers.length<1||taskIssueNumbers.length>200||
     taskIssueNumbers.some(n=>!Number.isSafeInteger(n)||n<1)||
     new Set(taskIssueNumbers).size!==taskIssueNumbers.length||
     typeof nativeC?.catalog!=='function'||
     (!readPair&&(typeof token!=='string'||token.length<8)))
    throw new TypeError('TRUSTED_ADS_CLAIM_AUDIT_SOURCE_REQUIRED');
  const [owner,repo]=repository.split('/');
  const source={repository,sourceSha:expectedSha,
    currentness:'CURRENT',readback:'VERIFIED'};
  async function audit(scope){
    if(scope?.projectId!==repository)
      return deny('PROJECT_SCOPE_MISMATCH');
    try{
      const first=await nativeC.catalog();
      if(first?.connected!==true||first.source?.repository!==repository||
         first.source.sourceSha!==expectedSha)
        return deny('AUDIT_VERSION_SOURCE_UNQUALIFIED');
      const read=readPair??(async args=>{
        const {readGitHubNativeTaskSourcePair}=await import(
          '../../packages/tally-github-native-c/src/task-source-pair.mjs');
        return readGitHubNativeTaskSourcePair({...args,token});
      });
      const records=[];
      for(const issueNumber of taskIssueNumbers){
        const pair=await read({owner,repo,issueNumber,expectedSourceSha:expectedSha});
        if(pair?.sourceQualified!==true||pair?.status!=='SOURCE_PAIR_OBSERVED'||
           pair?.claimStatus!=='NOT_VERIFIED'||pair?.authorized!==false||
           pair?.admitted!==false||pair?.taskPass!==false||
           pair?.releasePass!==false||
           pair?.sourceSha!==expectedSha||
           pair?.issueRef!=='https://github.com/'+repository+'/issues/'+issueNumber||
           pair.workItem?.claimStatus!=='NOT_VERIFIED'||
           !Number.isSafeInteger(pair.claimEvents?.candidateCount)||
           pair.claimEvents.candidateCount<0||
           !Array.isArray(pair.claimEvents.candidates)||
           pair.claimEvents.candidates.length!==pair.claimEvents.candidateCount||
           pair.claimEvents.candidates.some(c=>c?.sourceSha!==expectedSha||
             c?.claimAccepted!==false||
             c?.sourceClass!=='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'||
             !/^observed-comment-[1-9][0-9]*$/.test(c?.dispatchId??'')))
          return deny('NATIVE_ADS_CLAIM_SOURCE_UNQUALIFIED');
        records.push({
          id:'native-claim-'+issueNumber,
          kind:'audit',title:'Task #'+issueNumber+' · GitHub Claim source',
          state:'CLAIM_NOT_VERIFIED',projectId:repository,versionId,
          sourceRef:'https://github.com/'+repository+'/issues/'+issueNumber,
          summary:'ADS v3 DISPATCH_CLAIMED candidate comments observed: '+
            pair.claimEvents.candidateCount+
            '. These are NOT accepted CAS Claims or authorized Agent attempts.',
          authority:'OBSERVED'
        });
      }
      const last=await nativeC.catalog();
      if(last?.connected!==true||
         last.source?.repository!==repository||
         last.source.sourceSha!==expectedSha)
        return deny('AUDIT_VERSION_MOVED_DURING_READ');
      return {connected:true,source,records,
        rosterCoverage:'UNVERIFIED',claimAcceptance:'NOT_VERIFIED'};
    }catch{return deny('NATIVE_ADS_CLAIM_AUDIT_READ_FAILED');}
  }
  return Object.freeze({audit});
}
