/** New-project Builder candidate -> DRAFT PR only; NEVER a review approval.
 * Trusted Host scopes everything; browser sends issueNumber+sourceSha only.
 * Host requires native published Claim, exact FSW, K attempt CAS, local Git
 * source/tree/path and same-branch HEAD before any push. Publication is fenced.
 */
import {createHash} from 'node:crypto';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {readFile,stat,realpath} from 'node:fs/promises';
import {dirname,join,relative,isAbsolute,resolve} from 'node:path';
import {resolveBuilderBuildTarget,exactSelectedWriteSet}
  from '../../experiments/v2/selected-business-build/builder-target.mjs';
import {readGitHubClaimPublicationWitness}
  from '../../packages/tally-github-native-c/src/task-claim-publication-witness.mjs';
const run=promisify(execFile);
const SHA=/^[a-f0-9]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const hash=x=>createHash('sha256').update(x).digest('hex');
const denied=reason=>Object.freeze({status:'BLOCKED',reason,
  authorized:false,taskPass:false,releasePass:false});
const unknown=reason=>Object.freeze({status:'PUBLICATION_UNKNOWN',reason,
  authorized:false,taskPass:false,releasePass:false});
const phaseKey=(repo,issue,attempt)=>'tally:v2:selected-draft-pr:'+hash(
  JSON.stringify([repo,issue,attempt]));
const under=(root,path)=>{const rel=relative(root,path);return rel!==''&&
  rel!=='..'&&!rel.startsWith('..\\')&&!rel.startsWith('../')&&!isAbsolute(rel);};
export function createSelectedCandidatePublisher({
  settings,env,octokit,openStore,exec=run,
  readEvidence=readFile,readWitness=readGitHubClaimPublicationWitness,
  readStat=stat,resolveRealPath=realpath
}={}){
  if(env?.TALLY_V2_CANDIDATE_PUBLISH!=='host-mounted')return null;
  const target=resolveBuilderBuildTarget(env);
  const repository=settings?.repository,sourceSha=settings?.expectedSha;
  const attemptId=env.TALLY_V2_AGENT_CLAIM_ATTEMPT_ID;
  const evidenceDir=env.TALLY_V2_AGENT_EVIDENCE_DIR;
  const worktree=env.TALLY_V2_AGENT_WORKTREE_PATH;
  const claimRegistry=env.TALLY_V2_CLAIM_REGISTRY_PATH;
  const writerLogin=env.TALLY_V2_CLAIM_WRITER_LOGIN;
  const token=env.TALLY_GITHUB_WRITE_TOKEN;
  if(target.mode!=='SELECTED_BUILD'||target.repository!==repository||
     target.versionId!==settings.versionId||!SHA.test(sourceSha??'')||
     env.TALLY_V2_CLAIM_REPOSITORY!==repository||
     env.TALLY_V2_CLAIM_ISSUE_NUMBER!==String(target.issueNumber)||
     env.TALLY_V2_CLAIM_SOURCE_SHA!==sourceSha||
     env.TALLY_V2_AGENT_CLAIM_SOURCE_SHA!==sourceSha||
     env.TALLY_V2_AGENT_CLAIM_REGISTRY_PATH!==claimRegistry||
     !claimRegistry||!ATTEMPT.test(attemptId??'')||
     !isAbsolute(evidenceDir??'')||!isAbsolute(worktree??'')||
     typeof token!=='string'||token.length<8||
     typeof settings?.token!=='string'||settings.token.length<8||
     typeof openStore!=='function'||typeof exec!=='function'||
     typeof octokit?.rest?.repos?.getBranch!=='function'||
     typeof octokit?.rest?.git?.getRef!=='function'||
     typeof octokit?.rest?.pulls?.create!=='function'||
     typeof octokit?.rest?.pulls?.get!=='function'||
     typeof octokit?.rest?.pulls?.list!=='function')
    throw new TypeError('SELECTED_CANDIDATE_HOST_SCOPE_INVALID');
  const {owner,repo,issueNumber,candidateRelativePath,versionId}=target;
  const issueRef='https://github.com/'+repository+'/issues/'+issueNumber;
  const claimKey=repository+'#'+issueNumber+':builder:__default__';
  const claimRegistryKey='tally:v2:claim-admission:'+hash(claimKey);
  const writeSetKey='tally:v2:task-writeset:'+issueNumber;
  const publicationKey=phaseKey(repository,issueNumber,attemptId);
  const branch='tally/candidates/'+issueNumber+'-'+attemptId.replace(/[^a-z0-9]+/gi,'-').replace(/^-|-$/g,'').toLowerCase();
  const git=env.TALLY_GIT_PATH??'C:\\Program Files\\Git\\cmd\\git.exe';
  const githubWriteEnv=()=>({...process.env,TALLY_GITHUB_TOKEN:token,
    GIT_TERMINAL_PROMPT:'0',GIT_CONFIG_NOSYSTEM:'1',
    GIT_CONFIG_GLOBAL:env.TALLY_EMPTY_GIT_CONFIG??'NUL',GIT_CONFIG_COUNT:'1',
    GIT_CONFIG_KEY_0:'credential.helper',
    GIT_CONFIG_VALUE_0:'!f() { echo username=x-access-token; echo password=$TALLY_GITHUB_TOKEN; }; f'});
  async function gitRead(args){
    const out=await exec(git,['-C',worktree,...args],{windowsHide:true,
      timeout:15000,maxBuffer:512*1024});
    return String(out.stdout).trim();
  }
  const readPR=async commit=>{
    const found=await octokit.rest.pulls.list({owner,repo,state:'all',
      head:owner+':'+branch,per_page:100});
    const matches=found.data.filter(pr=>pr.head?.ref===branch);
    if(matches.length!==1)return null;
    const current=(await octokit.rest.pulls.get({owner,repo,
      pull_number:matches[0].number})).data;
    if(current?.state!=='open'||current.draft!==true||
       current.head?.sha!==commit||current.head?.ref!==branch||
       current.head?.repo?.full_name!==repository||
       current.base?.ref!==versionId||
       current.base?.repo?.full_name!==repository||
       !Number.isSafeInteger(current.number)||current.number<1)return null;
    return Object.freeze({status:'DRAFT_PR_PUBLISHED_VERIFIED',
      pullNumber:current.number,pullRef:current.html_url,branch,
      candidateCommit:commit,authorized:false,taskPass:false,releasePass:false});
  };
  return Object.freeze({repository,versionId,issueNumber,sourceSha,attemptId,
    candidateRelativePath,
    async publish({issueNumber:requested,sourceSha:requestedSha}={}){
      if(requested!==issueNumber||requestedSha!==sourceSha)
        return denied('SELECTED_CANDIDATE_TASK_SCOPE_MISMATCH');
      let claimStore;
      try{
        claimStore=await openStore({path:claimRegistry,busyTimeoutMs:1000});
        const [claim,fsw,prior]=await Promise.all([
          claimStore.get(claimRegistryKey),claimStore.get(writeSetKey),
          claimStore.get(publicationKey)]);
        const c=claim?.value,w=fsw?.value;
        if(claim?.revision<1||c?.phase!=='PUBLISHED_VERIFIED'||
           c.repository!==repository||c.issueRef!==issueRef||
           c.sourceSha!==sourceSha||c.actorRole!=='builder'||
           c.protectedClaimKey!==claimKey||
           c.dispatchId!==env.TALLY_V2_CLAIM_DISPATCH_ID||
           c.operatorId!==env.TALLY_V2_CLAIM_OPERATOR_ID||
           c.operatorKind!==env.TALLY_V2_CLAIM_OPERATOR_KIND||
           c.writerId!==env.TALLY_V2_CLAIM_WRITER_ID||
           !Number.isSafeInteger(c.commentId)||c.commentId<1)
          return denied('SELECTED_CANDIDATE_CLAIM_NOT_VERIFIED');
        if(!Number.isSafeInteger(fsw?.revision)||fsw.revision<0||
           typeof w?.document!=='string'||hash(w.document)!==w.writeSetSha256||
           !exactSelectedWriteSet(w,{repository,issueNumber,sourceSha,
             digest:w.writeSetSha256,candidateRelativePath}))
          return denied('SELECTED_CANDIDATE_WRITESET_NOT_VERIFIED');
        if(prior){
          if(prior.value?.phase==='PUBLISHED_VERIFIED'&&
             SHA.test(prior.value.candidateCommit??'')&&
             prior.value.sourceSha===sourceSha){
            const previous=await readPR(prior.value.candidateCommit);
            return previous??unknown('PUBLISHED_PR_READBACK_CHANGED');
          }
          return unknown('PR_PUBLICATION_RECONCILIATION_REQUIRED');
        }
        const witness=await readWitness({token:settings.token,owner,repo,
          issueNumber,expectedSourceSha:sourceSha,commentId:c.commentId,
          dispatchId:c.dispatchId,actorRole:'builder',
          operatorId:c.operatorId,operatorKind:c.operatorKind,
          protectedClaimKey:claimKey,claimGeneration:1,writerLogin});
        if(witness?.status!=='PUBLICATION_WITNESSED'||
           witness.commentId!==c.commentId||
           witness.sourceSha!==sourceSha||witness.issueRef!==issueRef)
          return denied('SELECTED_CANDIDATE_NATIVE_CLAIM_UNVERIFIED');
        const head=(await octokit.rest.repos.getBranch({owner,repo,
          branch:versionId})).data?.commit?.sha;
        if(head!==sourceSha)return denied('SELECTED_CANDIDATE_BASE_MOVED');
        const file=join(evidenceDir,'e2e-result.json');
        const fileStat=await readStat(file);
        if(!fileStat.isFile()||fileStat.size>2*1024*1024)
          return denied('SELECTED_CANDIDATE_EVIDENCE_INVALID');
        const receipt=JSON.parse(await readEvidence(file,'utf8'));
        const value=receipt?.result;
        if(receipt?.status!=='PASS'||receipt.realAgentEffect!==true||
           receipt.hiddenPass===true||value?.sourceSha!==sourceSha||
           value?.attemptId!==attemptId||value?.agentLaunch!==true||
           value?.result!=='LOCAL_CANDIDATE_READY'||
           !SHA.test(value.candidateCommit??'')||
           !SHA.test(value.candidateTree??''))
          return denied('SELECTED_CANDIDATE_RECEIPT_UNVERIFIED');
        const evidenceRoot=await resolveRealPath(evidenceDir);
        const casPath=await resolveRealPath(value.hostStorePath??'');
        if(!under(evidenceRoot,casPath))
          return denied('SELECTED_CANDIDATE_CAS_PATH_NOT_LOCAL');
        let attemptStore;
        let attempt;
        try{
          attemptStore=await openStore({path:casPath,busyTimeoutMs:1000});
          attempt=await attemptStore.get('tally:v2:local-agent-attempt:'+attemptId);
        }finally{if(attemptStore)await attemptStore.close();}
        const a=attempt?.value;
        if(!Number.isSafeInteger(attempt?.revision)||attempt.revision<1||
           a?.phase!=='CANDIDATE_READY'||a.attemptId!==attemptId||
           a.issueRef!==issueRef||a.sourceSha!==sourceSha||
           a.candidateRelativePath!==candidateRelativePath||
           a.candidateCommit!==value.candidateCommit||
           a.candidateTree!==value.candidateTree||
           a.candidateRef!=='refs/tally-candidates/'+
             attemptId.replace(/[^A-Za-z0-9._-]/g,'-'))
          return denied('SELECTED_CANDIDATE_CAS_UNVERIFIED');
        const [parent,tree,diff,ref,worktreeRoot]=await Promise.all([
          gitRead(['rev-parse',a.candidateCommit+'^']),
          gitRead(['rev-parse',a.candidateCommit+'^{tree}']),
          gitRead(['diff','--name-only',sourceSha,a.candidateCommit]),
          gitRead(['rev-parse',a.candidateRef]),
          resolveRealPath(worktree)]);
        if(worktreeRoot!==await resolveRealPath(a.worktreePath)||
           parent!==sourceSha||tree!==a.candidateTree||ref!==a.candidateCommit||
           diff!==candidateRelativePath)
          return denied('SELECTED_CANDIDATE_GIT_READBACK_FAILED');
        // create-only fence before a non-idempotent GitHub push / Draft PR POST
        const pending={phase:'PENDING',repository,issueNumber,sourceSha,
          attemptId,candidateCommit:a.candidateCommit,branch,baseRef:versionId};
        await claimStore.put(publicationKey,pending,{mustBeAbsent:true});
        const reserved=await claimStore.get(publicationKey);
        if(!reserved||reserved.value?.phase!=='PENDING'||
           reserved.value.candidateCommit!==a.candidateCommit)
          return unknown('SELECTED_CANDIDATE_PUBLICATION_FENCE_UNKNOWN');
        const remote=await octokit.rest.git.getRef({owner,repo,
          ref:'heads/'+branch}).then(x=>x.data).catch(e=>e?.status===404?null:undefined);
        if(remote===undefined)return unknown('DRAFT_PR_REMOTE_REF_UNKNOWN');
        if(remote!==null)return unknown('DRAFT_PR_BRANCH_ALREADY_EXISTS');
        try{
          await exec(git,['-C',worktree,'push','https://github.com/'+repository+'.git',
            a.candidateCommit+':refs/heads/'+branch],{
            env:githubWriteEnv(),windowsHide:true,timeout:60000,maxBuffer:512*1024});
        }catch{return unknown('DRAFT_PR_GIT_PUSH_UNKNOWN');}
        const actualRef=await octokit.rest.git.getRef({owner,repo,
          ref:'heads/'+branch}).then(x=>x.data).catch(()=>null);
        if(actualRef?.object?.sha!==a.candidateCommit)
          return unknown('DRAFT_PR_GIT_REF_READBACK_UNKNOWN');
        try{
          await octokit.rest.pulls.create({owner,repo,head:branch,
            base:versionId,draft:true,
            title:'Tally candidate for #'+issueNumber,
            body:'Unreviewed Builder candidate '+a.candidateCommit+
              ' for '+issueRef+'.\n\nDraft PR only. Independent review and CI pending.'});
        }catch{return unknown('DRAFT_PR_CREATE_UNKNOWN');}
        const verified=await readPR(a.candidateCommit);
        if(!verified)return unknown('DRAFT_PR_READBACK_UNKNOWN');
        const done={...pending,phase:'PUBLISHED_VERIFIED',
          pullNumber:verified.pullNumber,pullRef:verified.pullRef};
        await claimStore.put(publicationKey,done,{
          mustMatchRevision:reserved.revision});
        const final=await claimStore.get(publicationKey);
        if(final?.value?.phase!=='PUBLISHED_VERIFIED'||
           final.value.pullNumber!==verified.pullNumber)
          return unknown('DRAFT_PR_CAS_COMMIT_UNKNOWN');
        return verified;
      }catch{return unknown('DRAFT_PR_HOST_RECONCILIATION_REQUIRED');}
      finally{if(claimStore)await claimStore.close().catch(()=>{});}
    }
  });
}
