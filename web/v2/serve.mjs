/**
 * Tally v2 UI loopback read-only entrypoint.
 *
 * Requires an operator-selected GitHub Native C source, a pinned branch SHA
 * and an explicit frozen Task roster. It NEVER starts Agent work or writes
 * GitHub. Unconfigured read sources stay NOT_CONNECTED.
 *
 * Environment (Host-side ONLY; never exposed to browser):
 *   TALLY_V2_REPOSITORY=owner/repo
 *   TALLY_V2_VERSION_REF=version/v2
 *   TALLY_V2_VERSION_SHA=<40 hex chars>
 *   TALLY_V2_TASK_ISSUES_JSON=[101,102,...]
 *   TALLY_GITHUB_READ_TOKEN=<credential scoped to reading repo>
 *   TALLY_V2_FROZEN_DAG_HOST=read-only   (optional; mounts the real frozen
 *     version Task DAG reader against the E0 CAS registry — requires the
 *     TALLY_VERSION_DAG_* registry/witness pins below)
 *   TALLY_VERSION_DAG_REGISTRY_PATH=<dedicated CAS registry file>
 *   TALLY_VERSION_DAG_REVIEW_ISSUE/_COMMENT_ID/_LOGIN
 *   TALLY_VERSION_DAG_APPROVAL_ISSUE/_COMMENT_ID/_LOGIN
 *
 * #264 E1: the five former ui-*-host.mjs static imports (native read model,
 * ADS report, audit, settings, workbench) resolved only in the unmerged
 * Draft PR #258 and broke this entrypoint's ESM load. This entrypoint now
 * composes ONLY what exists in the selected source tree: createUiNativeSource
 * plus createUiRosterHost against the real frozen DAG reader. Optional
 * providers that have no selected module (ADS reports, settings, metadata,
 * audit, workbench) stay absent; the ALM server answers those models
 * NOT_CONNECTED — never fabricated connected:true data.
 */
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import {createHash} from 'node:crypto';
import {formatStartupError} from './startup-error.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {createMaintenanceReadHost} from './maintenance-read-host.mjs';
import {createUiNativeSource,readUiNativeVersionFromClient,
  observeUiNativeVersionHeadFromClient,verifyUiNativeVersionAncestryFromClient}
  from '../../packages/tally-github-native-c/src/ui-version-readonly.mjs';
import {projectReleaseReadiness} from
  '../../packages/tally-engineering-business/src/release-readiness.mjs';
import {createUiRosterHost,createUiRosterDualEpochHost} from './ui-roster-host.mjs';
import {readInstalledWriteJourneyEnvironment,createInstalledWriteJourneyRunner}
  from './installed-write-journeys.mjs';

const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
import {isValidVersionRef} from '../../packages/tally-github-native-c/src/version-ref.mjs';
const SHA=/^[a-f0-9]{40}$/;
export function readUiHostEnvironment(env){
  const repository=env.TALLY_V2_REPOSITORY;
  const versionId=env.TALLY_V2_VERSION_REF;
  const expectedSha=env.TALLY_V2_VERSION_SHA;
  if(!REPO.test(repository??'')||!isValidVersionRef(versionId??'')||
    !SHA.test(expectedSha??'')||
    typeof env.TALLY_GITHUB_READ_TOKEN!=='string'||
    env.TALLY_GITHUB_READ_TOKEN.length<8)
    throw new TypeError('SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED');
  let roster=[];
  if(env.TALLY_V2_TASK_ISSUES_JSON){
    try{roster=JSON.parse(env.TALLY_V2_TASK_ISSUES_JSON);}catch{
      throw new TypeError('INVALID_HOST_TASK_ROSTER');
    }
    if(!Array.isArray(roster)||roster.length>200||
      roster.some(n=>!Number.isSafeInteger(n)||n<1)||
      new Set(roster).size!==roster.length)
      throw new TypeError('INVALID_HOST_TASK_ROSTER');
  }
  // Requirement roster is an explicit Host taxonomy declaration, never
  // a Product Freeze or canonical Epic/Story DAG witness.
  let requirements=null;
  if(env.TALLY_V2_REQUIREMENT_ISSUES_JSON!==undefined){
    try{requirements=JSON.parse(env.TALLY_V2_REQUIREMENT_ISSUES_JSON);}
    catch{throw new TypeError('INVALID_HOST_REQUIREMENT_ROSTER');}
    if(!Array.isArray(requirements)||requirements.length<1||
       requirements.length>16||
       requirements.some(x=>!x||!Number.isSafeInteger(x.number)||x.number<1||
         !['requirement','epic','story'].includes(x.kind)||
         (x.parent!==null&&(!Number.isSafeInteger(x.parent)||x.parent<1)))||
       new Set(requirements.map(x=>x.number)).size!==requirements.length)
      throw new TypeError('INVALID_HOST_REQUIREMENT_ROSTER');
  }
  const qaStatusContext=env.TALLY_V2_QA_STATUS_CONTEXT??null;
  if(qaStatusContext!==null&&
     !/^ci\/woodpecker\/(?:pr|push)\/[A-Za-z0-9_.-]{1,80}$/.test(qaStatusContext))
    throw new TypeError('INVALID_HOST_QA_STATUS_CONTEXT');
  const hasGrant=Boolean(env.TALLY_V2_ADS_ACCEPTED_COMMENTS_JSON);
  const hasBindings=Boolean(env.TALLY_V2_ADS_PUBLISHER_BINDINGS_JSON);
  if(hasGrant!==hasBindings)
    throw new TypeError('ADS_HOST_REPORT_CONFIGURATION_INCOMPLETE');
  let ads=null;
  if(hasGrant){
    try{
      ads={grants:JSON.parse(env.TALLY_V2_ADS_ACCEPTED_COMMENTS_JSON),
        bindings:JSON.parse(env.TALLY_V2_ADS_PUBLISHER_BINDINGS_JSON)};
    }catch{throw new TypeError('INVALID_ADS_HOST_CONFIGURATION');}
    if(!ads.grants||!ads.bindings||Array.isArray(ads.grants)||
      Array.isArray(ads.bindings))
      throw new TypeError('INVALID_ADS_HOST_CONFIGURATION');
  }
  const auditMode=env.TALLY_V2_AUDIT_CLAIM_CANDIDATES;
  if(auditMode!==undefined&&auditMode!=='read-only')
    throw new TypeError('INVALID_HOST_AUDIT_READ_MODE');
  if(auditMode==='read-only'&&roster.length===0)
    throw new TypeError('HOST_AUDIT_TASK_ROSTER_REQUIRED');
  const workbenchMode=env.TALLY_V2_WORKBENCH_BLOCKERS;
  if(workbenchMode!==undefined&&workbenchMode!=='read-only')
    throw new TypeError('INVALID_HOST_WORKBENCH_READ_MODE');
  if(workbenchMode==='read-only'&&roster.length===0)
    throw new TypeError('HOST_WORKBENCH_TASK_ROSTER_REQUIRED');
  // #264 E1: read-only frozen DAG reader mount. Same trusted Host pin
  // contract as the E0 write seam (registry file + exact witness comment
  // ids), but never requires a write-scoped token. Absent pins fail START
  // closed instead of silently degrading a declared acceptance run.
  const frozenDagMode=env.TALLY_V2_FROZEN_DAG_HOST;
  if(frozenDagMode!==undefined&&frozenDagMode!=='read-only')
    throw new TypeError('INVALID_HOST_FROZEN_DAG_READ_MODE');
  const frozenPins={registryPath:env.TALLY_VERSION_DAG_REGISTRY_PATH??null,
    reviewIssue:Number(env.TALLY_VERSION_DAG_REVIEW_ISSUE),
    reviewCommentId:Number(env.TALLY_VERSION_DAG_REVIEW_COMMENT_ID),
    reviewLogin:env.TALLY_VERSION_DAG_REVIEW_LOGIN??
      env.TALLY_VERSION_DAG_WRITER_LOGIN??null,
    approvalIssue:Number(env.TALLY_VERSION_DAG_APPROVAL_ISSUE),
    approvalCommentId:Number(env.TALLY_VERSION_DAG_APPROVAL_COMMENT_ID),
    approvalLogin:env.TALLY_VERSION_DAG_APPROVAL_LOGIN??null};
  if(frozenDagMode==='read-only'){
    if(roster.length===0)
      throw new TypeError('HOST_FROZEN_DAG_TASK_ROSTER_REQUIRED');
    if(typeof frozenPins.registryPath!=='string'||!frozenPins.registryPath)
      throw new TypeError('HOST_FROZEN_DAG_REGISTRY_REQUIRED');
    for(const [issue,comment] of [[frozenPins.reviewIssue,
        frozenPins.reviewCommentId],[frozenPins.approvalIssue,
        frozenPins.approvalCommentId]])
      if(!Number.isSafeInteger(issue)||issue<1||
         !Number.isSafeInteger(comment)||comment<1)
        throw new TypeError('HOST_FROZEN_DAG_WITNESS_PINS_REQUIRED');
    if(typeof frozenPins.reviewLogin!=='string'||!frozenPins.reviewLogin)
      throw new TypeError('HOST_FROZEN_DAG_REVIEW_LOGIN_REQUIRED');
  }
  // #264 J02 dual-epoch read mount: TALLY_V2_FREEZE_BASE_SHA declares the
  // expected S0 freeze epoch (the R80 CAS row must still pin exactly it),
  // while expectedSha stays the served live S1 pin. The per-edge amendment
  // ledger is Host governance input (NATIVE_EDGE_REMOVED only, repo-scoped
  // evidence); it is never accepted from a request. Both require the same
  // frozen-DAG read pins; the reader factory revalidates every entry.
  const freezeBaseSha=env.TALLY_V2_FREEZE_BASE_SHA??null;
  if(freezeBaseSha!==null&&!SHA.test(freezeBaseSha))
    throw new TypeError('INVALID_HOST_FREEZE_BASE_SHA');
  let edgeAmendments=null;
  if(env.TALLY_V2_EDGE_AMENDMENTS_JSON!==undefined){
    try{edgeAmendments=JSON.parse(env.TALLY_V2_EDGE_AMENDMENTS_JSON);}
    catch{throw new TypeError('INVALID_HOST_EDGE_AMENDMENTS_LEDGER');}
    if(!Array.isArray(edgeAmendments)||!edgeAmendments.length||
       edgeAmendments.length>64)
      throw new TypeError('INVALID_HOST_EDGE_AMENDMENTS_LEDGER');
  }
  if((freezeBaseSha!==null||edgeAmendments!==null)&&
     frozenDagMode!=='read-only')
    throw new TypeError('DUAL_EPOCH_REQUIRES_FROZEN_DAG_PINS');
  if(edgeAmendments!==null&&freezeBaseSha===null)
    throw new TypeError('EDGE_AMENDMENTS_REQUIRE_FREEZE_BASE_SHA');
  const [owner,repo]=repository.split('/');
  return {repository,owner,repo,versionId,expectedSha,roster,requirements,qaStatusContext,ads,
    auditReadOnly:auditMode==='read-only',
    workbenchReadOnly:workbenchMode==='read-only',
    frozenDagReadOnly:frozenDagMode==='read-only',
    frozenPins,freezeBaseSha,edgeAmendments,
    token:env.TALLY_GITHUB_READ_TOKEN};
}
/** #264 E1: read-only frozen version Task DAG reader composition. Mirrors the
 * E0 host composition contract (experiments/v2/selected-business-build/
 * version-task-dag-host.mjs) but grants NO write port: one dedicated CAS
 * connection per read, GitHub blob/comment re-reads bound to the exact
 * PackBlob + selected source SHA. Heavy transports load lazily so a plain
 * read-only boot never requires the Host-only dependency chain. */
async function createFrozenDagReadHost({settings}){
  const [{Octokit},{createHostVersionTaskDagReader,
      createHostVersionTaskDagDualEpochReader},{openSqliteCasStore}]=
    await Promise.all([
      import('../../packages/tally-github-native-c'+
        '/node_modules/@octokit/rest/dist-src/index.js'),
      import('../../packages/tally-github-native-c/src/closeout-roster.mjs'),
      import('@kaicreator/domain-harness-host-node')]);
  if(typeof Octokit!=='function'||
     typeof createHostVersionTaskDagReader!=='function'||
     typeof openSqliteCasStore!=='function')
    throw new TypeError('FROZEN_DAG_READER_TRANSPORTS_UNAVAILABLE');
  if(settings.freezeBaseSha!==null&&
     typeof createHostVersionTaskDagDualEpochReader!=='function')
    throw new TypeError('DUAL_EPOCH_READER_TRANSPORTS_UNAVAILABLE');
  const octokit=new Octokit({auth:settings.token,
    request:{timeout:15000}});
  const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
  const SHA=/^[a-f0-9]{40}$/;
  const parseBlobRef=taskPackRef=>{
    if(typeof taskPackRef!=='string')return null;
    const prefix='https://github.com/'+settings.repository+'/blob/';
    if(!taskPackRef.startsWith(prefix))return null;
    const rest=taskPackRef.slice(prefix.length),slash=rest.indexOf('/');
    if(slash!==40||!SHA.test(rest.slice(0,slash))||rest.length===slash+1)
      return null;
    return {commitSha:rest.slice(0,slash),path:rest.slice(slash+1)};
  };
  const taskPackReader=async({taskPackRef,expectedBlobSha})=>{
    const parsed=parseBlobRef(taskPackRef);
    if(!parsed||!SHA.test(expectedBlobSha??''))
      return {status:'NOT_VERIFIED',reason:'TASK_PACK_IDENTITY_INVALID'};
    try{
      const response=await octokit.rest.repos.getContent({owner:settings.owner,
        repo:settings.repo,path:parsed.path,ref:parsed.commitSha});
      const data=response?.data;
      if(data?.encoding!=='base64'||typeof data?.content!=='string'||
         data.sha!==expectedBlobSha)
        return {status:'NOT_VERIFIED',reason:'TASK_PACK_BLOB_MISMATCH'};
      return {status:'VERIFIED',blobSha:data.sha,
        content:Buffer.from(data.content,'base64').toString('utf8')};
    }catch{return {status:'NOT_VERIFIED',reason:'TASK_PACK_READ_FAILED'};}
  };
  const dagKey='tally:v2:version-task-dag:'+
    createHash('sha256').update(settings.repository+'#'+settings.versionId)
      .digest('hex');
  /** Witness ports built per-read against the OPEN CAS store. The frozen
   * witness JSON lives in the immutable CAS row (the R80-class artifacts are
   * human-readable receipts, not JSON bodies); the port re-reads the LIVE
   * artifact at the frozen ref (PR review via the review API, owner
   * approval via the issue-comment API), cross-checks it against the Host
   * pins, and only then returns the frozen witness with its blob/source
   * bindings for the reader's own exact cross-validation. Any drift between
   * pins, frozen refs and the live artifact denies NOT_VERIFIED. */
  const buildWitnessReaders=store=>{
    const frozenDoc=async()=>{
      const row=await store.get(dagKey);
      return plain(row?.value)?row.value:null;
    };
    const parseReviewRef=ref=>{
      const m=typeof ref==='string'?
        /\/pull\/(\d+)#pullrequestreview-(\d+)$/.exec(ref):null;
      return m?{pullNumber:Number(m[1]),reviewId:Number(m[2])}:null;
    };
    const parseApprovalRef=ref=>{
      const m=typeof ref==='string'?
        /\/issues\/(\d+)#issuecomment-(\d+)$/.exec(ref):null;
      return m?{issueNumber:Number(m[1]),commentId:Number(m[2])}:null;
    };
    const reviewWitnessReader=async({taskPackBlobSha,sourceSha})=>{
      try{
        const doc=await frozenDoc();
        const witness=doc?.witness;
        if(!plain(witness))return {status:'NOT_VERIFIED'};
        const live=parseReviewRef(witness.reviewRef);
        if(!live||live.pullNumber!==settings.frozenPins.reviewIssue||
           live.reviewId!==settings.frozenPins.reviewCommentId)
          return {status:'NOT_VERIFIED'};
        const read=()=>octokit.rest.pulls.getReview({owner:settings.owner,
          repo:settings.repo,pull_number:live.pullNumber,
          review_id:live.reviewId}).then(r=>r?.data??null);
        const first=await read(),second=await read();
        if(!plain(first)||!plain(second)||first.id!==live.reviewId||
           first.html_url!==witness.reviewRef||
           first.user?.login!==settings.frozenPins.reviewLogin||
           !['APPROVED','COMMENTED'].includes(first.state)||
           first.body!==second.body||
           first.submitted_at!==second.submitted_at)
          return {status:'NOT_VERIFIED'};
        return {...witness,taskPackBlobSha:doc.blobSha,sourceSha:doc.sourceSha};
      }catch{return {status:'NOT_VERIFIED'};}
    };
    const ownerApprovalReader=async()=>{
      try{
        const doc=await frozenDoc();
        const approval=doc?.ownerApproval;
        if(!plain(approval))return {status:'NOT_VERIFIED'};
        const live=parseApprovalRef(approval.approvalRef);
        if(!live||live.issueNumber!==settings.frozenPins.approvalIssue||
           live.commentId!==settings.frozenPins.approvalCommentId)
          return {status:'NOT_VERIFIED'};
        const read=()=>octokit.rest.issues.getComment({owner:settings.owner,
          repo:settings.repo,comment_id:live.commentId})
          .then(r=>r?.data??null);
        const first=await read(),second=await read();
        if(!plain(first)||!plain(second)||first.id!==live.commentId||
           first.html_url!==approval.approvalRef||
           first.user?.login!==settings.frozenPins.approvalLogin||
           first.created_at!==first.updated_at||
           first.body!==second.body)
          return {status:'NOT_VERIFIED'};
        return {...approval,taskPackBlobSha:doc.blobSha};
      }catch{return {status:'NOT_VERIFIED'};}
    };
    return {reviewWitnessReader,ownerApprovalReader};
  };
  const nativeRosterReader=({taskIssueNumbers})=>readUiNativeVersionFromClient({
    octokit,owner:settings.owner,repo:settings.repo,
    versionId:settings.versionId,expectedSha:settings.expectedSha,
    taskIssueNumbers});
  // #264 J02 dual-epoch ports: the live branch head observer, the official
  // ancestry probe and a roster reader pinned at the OBSERVED head (never
  // the served pin — S1 may be a frozen-epoch descendant).
  const branchHeadReader=()=>observeUiNativeVersionHeadFromClient({
    octokit,owner:settings.owner,repo:settings.repo,
    versionId:settings.versionId});
  const ancestryReader=({baseSha,headSha})=>
    verifyUiNativeVersionAncestryFromClient({
      octokit,owner:settings.owner,repo:settings.repo,baseSha,headSha});
  const currentRosterReader=({sourceSha,taskIssueNumbers})=>
    readUiNativeVersionFromClient({octokit,owner:settings.owner,
      repo:settings.repo,versionId:settings.versionId,
      expectedSha:sourceSha,taskIssueNumbers});
  const pins=settings.frozenPins;
  return Object.freeze({async readFrozenDag(scope={}){
    const store=await openSqliteCasStore({
      path:pins.registryPath,busyTimeoutMs:1000});
    try{
      const witnessReaders=buildWitnessReaders(store);
      const reader=createHostVersionTaskDagReader({
        repository:settings.repository,versionId:settings.versionId,
        expectedVersionSha:settings.expectedSha,store,taskPackReader,
        nativeRosterReader,
        reviewWitnessReader:witnessReaders.reviewWitnessReader,
        ownerApprovalReader:witnessReaders.ownerApprovalReader});
      return await reader.readFrozenDag(scope);
    }finally{await store.close().catch(()=>{});}
  },
  // Dual-epoch projection (S0 frozen historical + live S1 current). Only
  // mounted when the Host pinned a freeze base; amendments come exclusively
  // from the validated Host env ledger. A browser-supplied witness is never
  // accepted here — scope.compatibilityWitness is deliberately ignored.
  ...(settings.freezeBaseSha===null?{}:{async readDualEpoch(){
    const store=await openSqliteCasStore({
      path:pins.registryPath,busyTimeoutMs:1000});
    try{
      const witnessReaders=buildWitnessReaders(store);
      const reader=createHostVersionTaskDagDualEpochReader({
        repository:settings.repository,versionId:settings.versionId,
        freezeBaseSha:settings.freezeBaseSha,store,taskPackReader,
        reviewWitnessReader:witnessReaders.reviewWitnessReader,
        ownerApprovalReader:witnessReaders.ownerApprovalReader,
        branchHeadReader,ancestryReader,currentRosterReader,
        edgeAmendments:settings.edgeAmendments});
      return await reader.readDualEpoch({compatibilityWitness:null});
    }finally{await store.close().catch(()=>{});}
  }})});
}
/** #264 J07 — the installed Release Host Consumer dispatch. Mounted ONLY
 * under TALLY_V2_RELEASE_DISPATCH=host-mounted with trusted Host pins; it
 * composes the real J07b Host gate (experiments/v2/release/host-release.mjs)
 * whose claim/write-set ports read the pinned CAS registries and whose
 * independent-RQ/owner-grant ports read the pinned RQ/grant registries
 * (absent pins answer null, so the gate fails closed exactly as designed).
 * Physical GitHub effects additionally require the explicit Host
 * physical-effect flag plus the release CAS — never request-derived.
 * Every answer is a fail-closed observation shape the controller's
 * verifyReleaseSeamResult accepts; nothing here can authorize a release. */
export async function createInstalledReleaseDispatch({settings,env=process.env,
  openSqliteCasStore=null}={}){
  const token=env.TALLY_GITHUB_WRITE_TOKEN??settings.token;
  if(typeof token!=='string'||token.length<8)
    throw new TypeError('RELEASE_HOST_TOKEN_REQUIRED');
  if(env.TALLY_RELEASE_SOURCE_SHA!==settings.expectedSha)
    throw new TypeError('RELEASE_SOURCE_SHA_STALE');
  const [{Octokit},{createReleaseHost}]=await Promise.all([
    import('../../packages/tally-github-native-c'+
      '/node_modules/@octokit/rest/dist-src/index.js'),
    import('../../experiments/v2/release/host-release.mjs')]);
  // The Host-node CAS store stays lazy: the trusted Windows Host mounts with
  // the documented dependency closure, while hermetic tests inject a fake
  // (pin-validation denials must never open a store at all).
  const openStore=typeof openSqliteCasStore==='function'?openSqliteCasStore:
    (await import('@kaicreator/domain-harness-host-node')).openSqliteCasStore;
  if(typeof Octokit!=='function'||typeof openStore!=='function'||
     typeof createReleaseHost!=='function')
    throw new TypeError('RELEASE_HOST_TRANSPORTS_UNAVAILABLE');
  const digest=value=>createHash('sha256').update(value).digest('hex');
  const readCasRow=async(path,key)=>{
    if(typeof path!=='string'||!path)return null;
    let store;
    try{
      store=await openStore({path,busyTimeoutMs:1000});
      return await store.get(key);
    }catch{return null;}
    finally{await store?.close?.().catch(()=>{});}
  };
  const claimIssueNumber=Number(env.TALLY_RELEASE_CLAIM_ISSUE??222);
  const claimRegistryKey='tally:v2:claim-admission:'+digest(
    `${settings.repository}#${claimIssueNumber}:builder:__default__`);
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  const verifyNativeClaim=async({sourceSha})=>{
    const row=await readCasRow(env.TALLY_RELEASE_CLAIM_REGISTRY_PATH,
      claimRegistryKey);
    const value=row?.value;
    if(value?.phase==='PUBLISHED_VERIFIED'&&value.sourceSha===sourceSha)
      return {state:'ACCEPTED',sourceSha};
    return {state:'NOT_VERIFIED'};
  };
  const verifyWriteSet=async({issueNumber})=>{
    const row=await readCasRow(env.TALLY_RELEASE_WRITESET_REGISTRY_PATH,
      'tally:v2:dag-admission:'+digest(
        `${settings.repository}#${issueNumber}:builder`));
    if(row?.value?.status==='FROZEN_WRITE_SET_CREATE_ONLY'&&
       row?.value?.task==='#'+issueNumber)
      return {state:'FROZEN_VERIFIED'};
    return {state:'NOT_VERIFIED'};
  };
  const readIndependentRq=async exact=>{
    const row=await readCasRow(env.TALLY_RELEASE_RQ_REGISTRY_PATH,
      'tally:v2:release-rq:'+digest(JSON.stringify([exact.sourceSha,
        exact.sourceTree,exact.targetSha,exact.targetTree,exact.operation])));
    return row?.value??null;
  };
  const readOwnerGrant=async({sourceSha,targetSha,operation})=>{
    const row=await readCasRow(env.TALLY_RELEASE_GRANT_REGISTRY_PATH,
      'tally:v2:release-grant:'+digest(JSON.stringify(
        [sourceSha,targetSha,operation])));
    return row?.value??null;
  };
  const host=createReleaseHost({verifyNativeClaim,verifyWriteSet,
    readIndependentRq,readOwnerGrant,openCasStore:openStore,
    octokit,env});
  const seamResult=(status,reason,extra={})=>({status,reason,releasePass:false,
    authorized:false,taskPass:false,releasePublished:false,...extra});
  const mapHostResult=observed=>{
    const base={hostState:observed?.state??null,
      githubEffect:observed?.githubEffect??'NOT_RUN',
      operationKey:observed?.operationKey??null};
    if(observed?.state==='HOST_GATES_OBSERVED')
      return seamResult('OBSERVED',observed.reason??'HOST_GATES_OBSERVED',base);
    if(observed?.state==='RECONCILIATION_REQUIRED')
      return seamResult('RECONCILIATION_REQUIRED',
        observed.reason??'PREVIOUS_RELEASE_OPERATION_PENDING',base);
    if(observed?.state==='MERGED_REQUALIFY_REQUIRED')
      return seamResult('OBSERVED',observed.reason??'MERGED_REQUALIFY_REQUIRED',
        {...base,mergedSha:observed.mergedSha??null,
          mergedTree:observed.mergedTree??null});
    if(observed?.state==='TAG_CREATED_RELEASE_NOT_PUBLISHED')
      return seamResult('OBSERVED',observed.reason??'TAG_CREATED_RELEASE_NOT_PUBLISHED',
        {...base,githubEffect:'PERFORMED_READBACK_VERIFIED',
          tag:observed.tag??null,
          tagCommitSha:observed.commitSha??null,
          tagCommitTree:observed.commitTree??null});
    if(observed?.state==='GITHUB_EFFECT_UNKNOWN')
      return seamResult('GITHUB_EFFECT_UNKNOWN',
        observed.reason??'GITHUB_EFFECT_UNKNOWN',base);
    return seamResult('DENIED',observed?.reason??'RELEASE_HOST_NOT_ADMITTED',base);
  };
  const readCommitTree=async ref=>{
    const response=await octokit.rest.repos.getCommit(
      {owner:settings.owner,repo:settings.repo,ref});
    const tree=response?.data?.commit?.tree?.sha;
    if(typeof tree!=='string'||!SHA.test(tree))
      throw new Error('COMMIT_TREE_UNOBSERVABLE');
    return tree;
  };
  return async function releaseSealedDispatch({intentType,input}){
    if(intentType!=='deliverQualifiedRelease'||!input||typeof input!=='object')
      return seamResult('DENIED','UNKNOWN_INTENT_OR_INPUT');
    if(input.sourceSha!==settings.expectedSha)
      return seamResult('STALE','SOURCE_SHA_MISMATCH');
    if(input.kind==='merge'){
      const pullNumber=Number(env.TALLY_RELEASE_MERGE_PR_NUMBER);
      if(!Number.isSafeInteger(pullNumber)||pullNumber<1)
        return seamResult('DENIED','MERGE_PR_NOT_PINNED');
      try{
        const headTree=await readCommitTree(settings.expectedSha);
        const target=(await octokit.rest.repos.getBranch(
          {owner:settings.owner,repo:settings.repo,branch:'main'}))
          ?.data?.commit?.sha;
        if(typeof target!=='string'||!SHA.test(target))
          return seamResult('UNKNOWN','GITHUB_READBACK_UNAVAILABLE');
        const targetTree=await readCommitTree(target);
        const observed=await host.merge({pullNumber,
          headSha:settings.expectedSha,headTree,targetSha:target,targetTree});
        return mapHostResult(observed);
      }catch{return seamResult('UNKNOWN','GITHUB_READBACK_UNAVAILABLE');}
    }
    if(input.kind==='tag'){
      const commitSha=env.TALLY_RELEASE_TAG_SOURCE_SHA;
      if(commitSha!==settings.expectedSha)
        return seamResult('DENIED','TAG_SOURCE_NOT_PINNED');
      try{
        const commitTree=await readCommitTree(commitSha);
        return mapHostResult(await host.tag({commitSha,commitTree}));
      }catch{return seamResult('UNKNOWN','GITHUB_READBACK_UNAVAILABLE');}
    }
    if(input.kind==='publish')
      return seamResult('NOT_RUN','PUBLISH_EFFECT_NOT_BOUND');
    return seamResult('DENIED','UNKNOWN_ACTION');
  };
}
/** #491 J07 — real release qualification read (RQ / Owner grant /
 * operation-specific Host permits) plus the accurate live main target
 * readback. Fully injectable ({octokit, openStore, pins}) so tests run
 * hermetically; serve mounts it with lazy Host-only transports under
 * TALLY_V2_RELEASE_QUALIFICATION=host-mounted. Every row is exact-source /
 * exact-target bound and expiry-checked; absent registries or unreadable
 * rows answer the honest UNKNOWN — an advisory gate PASS can never become a
 * permit here. */
export function createReleaseQualificationRead({
  octokit=null,openStore=null,pins={},sourceSha=null,
  now=()=>new Date().toISOString()}={}){
  if(!SHA.test(sourceSha??''))throw new TypeError('RELEASE_QUALIFICATION_SOURCE_REQUIRED');
  const OWNER=typeof pins.owner==='string'?pins.owner:null;
  const REPO=typeof pins.repo==='string'?pins.repo:null;
  const readRow=async(path,key)=>{
    if(!openStore||typeof path!=='string'||!path)return null;
    let store;
    try{
      store=await openStore({path,busyTimeoutMs:1000});
      return await store.get(key);
    }catch{return null;}
    finally{await store?.close?.().catch(()=>{});}
  };
  const isoNow=now();
  const expired=expiresAt=>typeof expiresAt!=='string'||
    !Number.isFinite(Date.parse(expiresAt))||Date.parse(expiresAt)<Date.parse(isoNow);
  /** Live main target: actual SHA + tree, fail-closed UNKNOWN on any drift. */
  const readTarget=async()=>{
    if(!octokit||!OWNER||!REPO)
      return {targetSha:null,targetTree:null,targetState:'UNKNOWN',
        targetReason:'RELEASE_TARGET_READ_UNAVAILABLE'};
    try{
      const branch=(await octokit.rest.repos.getBranch(
        {owner:OWNER,repo:REPO,branch:'main'}))?.data;
      const targetSha=branch?.commit?.sha??null;
      if(!SHA.test(targetSha??''))
        return {targetSha:null,targetTree:null,targetState:'UNKNOWN',
          targetReason:'RELEASE_TARGET_HEAD_UNOBSERVABLE'};
      const commit=(await octokit.rest.repos.getCommit(
        {owner:OWNER,repo:REPO,ref:targetSha}))?.data;
      const targetTree=commit?.commit?.tree?.sha??null;
      if(!SHA.test(targetTree??''))
        return {targetSha:null,targetTree:null,targetState:'UNKNOWN',
          targetReason:'RELEASE_TARGET_TREE_UNOBSERVABLE'};
      return {targetSha,targetTree,targetState:'OBSERVED',targetReason:null};
    }catch{return {targetSha:null,targetTree:null,targetState:'UNKNOWN',
      targetReason:'RELEASE_TARGET_READ_UNAVAILABLE'};}
  };
  const readSourceTree=async()=>{
    if(!octokit||!OWNER||!REPO)return null;
    try{
      const commit=(await octokit.rest.repos.getCommit(
        {owner:OWNER,repo:REPO,ref:sourceSha}))?.data;
      const tree=commit?.commit?.tree?.sha??null;
      return SHA.test(tree??'')?tree:null;
    }catch{return null;}
  };
  return Object.freeze({readTarget,async readAuthority(){
    const target=await readTarget();
    const base={rqState:'UNKNOWN',ownerGrantState:'UNKNOWN',permits:null,
      sourceSha,...target};
    if(!openStore||(!pins.rqRegistryPath&&!pins.grantRegistryPath&&
      !pins.permitRegistryPath))
      return base;
    const sourceTree=await readSourceTree();
    const observedTarget=target.targetSha??'';
    const observedTargetTree=target.targetTree??'';
    // RQ: exact (source,tree,target,tree,operation) binding — mirrors the
    // J07b host-release readIndependentRq key contract for the MERGE lane.
    let rqState='UNKNOWN';
    if(sourceTree&&target.targetState==='OBSERVED'){
      const key='tally:v2:release-rq:'+createHash('sha256')
        .update(JSON.stringify([sourceSha,sourceTree,observedTarget,
          observedTargetTree,'MERGE'])).digest('hex');
      const row=await readRow(pins.rqRegistryPath,key);
      const value=row?.value;
      if(value&&value.state==='QUALIFIED'&&
        value.verifierIndependentlyBound===true&&
        value.sourceSha===sourceSha&&value.targetSha===observedTarget)
        rqState='QUALIFIED';
      else if(value)rqState='NOT_QUALIFIED';
    }
    // Owner grant: exact (source,target,operation) for MERGE.
    let ownerGrantState='UNKNOWN';
    if(target.targetState==='OBSERVED'){
      const key='tally:v2:release-grant:'+createHash('sha256')
        .update(JSON.stringify([sourceSha,observedTarget,'MERGE'])).digest('hex');
      const row=await readRow(pins.grantRegistryPath,key);
      const value=row?.value;
      if(value&&value.state==='GRANTED'&&value.sourceSha===sourceSha&&
        value.targetSha===observedTarget)ownerGrantState='GRANTED';
      else if(value)ownerGrantState='NOT_GRANTED';
    }
    // Operation-specific Host permits; only rows bound to the exact source
    // AND the observed live target count; expiry degrades to UNKNOWN.
    const permits=[];
    if(target.targetState==='OBSERVED'){
      for(const operation of ['merge','tag','publish']){
        const key='tally:v2:release-permit:'+createHash('sha256')
          .update(JSON.stringify([operation,sourceSha,observedTarget])).digest('hex');
        const row=await readRow(pins.permitRegistryPath,key);
        const value=row?.value;
        if(!value||value.operation!==operation||
          value.sourceSha!==sourceSha||value.targetSha!==observedTarget)
          continue;
        const state=value.state==='GRANTED'&&!expired(value.expiresAt)?'GRANTED':
          value.state==='DENIED'?'DENIED':'UNKNOWN';
        permits.push(Object.freeze({operation,state,
          sourceSha,grantedAt:typeof value.grantedAt==='string'?
            value.grantedAt:null}));
      }
    }
    return {...base,rqState,ownerGrantState,permits:Object.freeze(permits)};
  }});
}
export async function startUiReadOnlyServer({env=process.env}={}){
  const settings=readUiHostEnvironment(env);
  // E0 #284: optional Host-mounted privileged seam for the two version Task
  // DAG operations (official blocked_by dependency write + immutable DAG
  // freeze). The browser/Agent dispatch boundary above stays READ-ONLY: this
  // seam only exists when the trusted Host process supplies its own env pins
  // (write-scoped token, dedicated registry, exclusive writer identity); it
  // never derives authority from a request. The root Claim publication is NOT
  // mountable here — it remains a Host-process-only T06d operation.
  let versionTaskDagSealedDispatch;
  if(env.TALLY_V2_VERSION_TASK_DAG==='host-mounted'){
    const token=env.TALLY_GITHUB_WRITE_TOKEN??settings.token;
    if(typeof token!=='string'||token.length<8)
      throw new TypeError('VERSION_TASK_DAG_HOST_TOKEN_REQUIRED');
    const {Octokit}=await import('../../packages/tally-github-native-c'+
      '/node_modules/@octokit/rest/dist-src/index.js');
    const {openSqliteCasStore}=await import(
      '@kaicreator/domain-harness-host-node');
    const {createVersionTaskDagHost}=await import(
      '../../experiments/v2/selected-business-build/version-task-dag-host.mjs');
    const host=createVersionTaskDagHost({env,octokit:new Octokit({auth:token,
      request:{timeout:15000}}),openStore:openSqliteCasStore});
    versionTaskDagSealedDispatch=async({operation,input})=>
      operation==='addNativeTaskDependency'?
        host.addNativeTaskDependency(input):host.freezeVersionTaskDag(input);
  }
  // #264 J07: the installed Release Host Consumer seam — the real J07b gate
  // composed from trusted Host pins only. Without the mount every
  // /api/release/actions POST keeps the exact R91 fail-closed controller
  // denial (INDEPENDENT_HOST_ADMISSION_NOT_CONNECTED); with it the request
  // reaches the real claim/write-set/RQ/owner-grant chain and still cannot
  // mutate GitHub unless the separate physical-effect flag is set.
  let releaseSealedDispatch;
  if(env.TALLY_V2_RELEASE_DISPATCH==='host-mounted')
    releaseSealedDispatch=await createInstalledReleaseDispatch({settings,env});
  const nativeC=createUiNativeSource({token:settings.token,
    owner:settings.owner,repo:settings.repo,
    versionId:settings.versionId,expectedSha:settings.expectedSha,
    taskIssueNumbers:settings.roster,
    requirementIssueSpecs:settings.requirements,
    qaStatusContext:settings.qaStatusContext});
  // #489 J08: canonical GitHub #437 read-only selected Host. Off by
  // default; token and exact source pins stay in the Host, never in browser
  // or maintenance form fields. Does not mount the native Issue writer.
  let maintenanceHost;
  if(env.TALLY_V2_MAINTENANCE_READ==='host-mounted'){
    const {Octokit}=await import('../../packages/tally-github-native-c'+
      '/node_modules/@octokit/rest/dist-src/index.js');
    maintenanceHost=createMaintenanceReadHost({
      octokit:new Octokit({auth:settings.token,request:{timeout:15000}}),
      owner:settings.owner,repo:settings.repo,versionId:settings.versionId,
      expectedSha:settings.expectedSha});
  }
  // #491 J07: real release qualification reader. The target readback
  // (live main SHA/tree) is ALWAYS mounted from the read token — the
  // snapshot must never report the version source as the target again.
  // Authority rows (RQ/grant/permits) come from Host-pinned registries
  // only under TALLY_V2_RELEASE_QUALIFICATION=host-mounted; without the
  // pin the release model keeps the honest UNKNOWN (permits null).
  let releaseQualificationReader=null;
  {
    const {Octokit}=await import('../../packages/tally-github-native-c'+
      '/node_modules/@octokit/rest/dist-src/index.js');
    const releaseOctokit=new Octokit({auth:settings.token,
      request:{timeout:15000}});
    const mounted=env.TALLY_V2_RELEASE_QUALIFICATION==='host-mounted';
    let openStore=null;
    if(mounted)
      ({openSqliteCasStore:openStore}=await import(
        '@kaicreator/domain-harness-host-node'));
    releaseQualificationReader=createReleaseQualificationRead({
      octokit:releaseOctokit,openStore,sourceSha:settings.expectedSha,
      pins:{owner:settings.owner,repo:settings.repo,
        rqRegistryPath:env.TALLY_V2_RELEASE_RQ_REGISTRY_PATH??null,
        grantRegistryPath:env.TALLY_V2_RELEASE_GRANT_REGISTRY_PATH??null,
        permitRegistryPath:env.TALLY_V2_RELEASE_PERMIT_REGISTRY_PATH??null}});
  }
  // #264 E1 R91: real installed Builder write seam (J04 /api/actions start).
  // Mounted ONLY when the trusted Host process exports
  // TALLY_V2_AGENT_DISPATCH=host-mounted plus the exact claim/lease pins;
  // the attempt binds to the exact served source SHA (a stale claim refuses
  // to start). Bad pins fail START closed. Without the pins the exact R90
  // fail-closed stub stays: launching UI never grants an executor.
  let installedRun;
  if(env.TALLY_V2_AGENT_DISPATCH==='host-mounted'){
    const journeySettings=readInstalledWriteJourneyEnvironment(env);
    if(journeySettings.claim.sourceSha!==settings.expectedSha)
      throw new TypeError('AGENT_CLAIM_SOURCE_SHA_STALE');
    installedRun=createInstalledWriteJourneyRunner(
      {settings:journeySettings,env}).run;
  }
  // #264 E1: real frozen version Task DAG reader — read-only, composed only
  // from the trusted Host env pins validated in readUiHostEnvironment. It
  // re-reads the R80-class CAS record, the immutable PackBlob and both
  // witness comments on every projection; the roster host then qualifies the
  // selected GitHub native roster against it. Without the pins the reader is
  // absent and the roster host answers fail-closed
  // FROZEN_DAG_HOST_NOT_CONNECTED (never a bare open-Issue fallback).
  const frozenReadHost=settings.frozenDagReadOnly?
    await createFrozenDagReadHost({settings}):null;
  const rosterReadHost=createUiRosterHost({
    repository:settings.repository,versionId:settings.versionId,
    expectedSha:settings.expectedSha,frozenHost:frozenReadHost,nativeC});
  // #264 J02: dual-epoch roster host — the honest S0/S1 projection over the
  // R80 freeze epoch and the live branch head. Mounted only under the same
  // frozen read pins plus an explicit freeze base; without it the API answers
  // DUAL_EPOCH_NOT_CONFIGURED (never a single-epoch fallback dressed up).
  const dualEpochReadHost=settings.freezeBaseSha!==null&&
    typeof frozenReadHost?.readDualEpoch==='function'?
    createUiRosterDualEpochHost({repository:settings.repository,
      versionId:settings.versionId,freezeBaseSha:settings.freezeBaseSha,
      dualEpochHost:{readDualEpoch:frozenReadHost.readDualEpoch.bind(
        frozenReadHost)}}):null;
  // Host-composed v2 read model facade consumed by journey-server.mjs. Every
  // method fails closed; scope guards bind each read to the selected
  // repository/version. Optional providers without a selected module
  // (settings, metadata/attempts, ADS reports, audit, workbench) are NOT
  // mounted — the server answers those models NOT_CONNECTED.
  const scopeOk=(scope={})=>(scope.projectId===undefined||
      scope.projectId===settings.repository)&&
    (scope.versionId===undefined||scope.versionId===settings.versionId);
  const deny=reason=>({connected:false,reason});
  const rosterDeny=observed=>({connected:false,
    reason:observed?.reason??'ROSTER_HOST_NOT_CONNECTED',
    rosterCoverage:observed?.rosterCoverage??'UNVERIFIED'});
  const readQualifiedRoster=async scope=>{
    if(!scopeOk(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
    try{
      const observed=await rosterReadHost.roster(scope);
      return observed?.connected===true?observed:rosterDeny(observed);
    }catch{return {...deny('ROSTER_HOST_READ_FAILED'),
      rosterCoverage:'UNVERIFIED'};}
  };
  const v2ReadModelHost=Object.freeze({
    async catalog(){
      try{
        const result=await nativeC.catalog();
        if(result?.connected!==true||
           result.source?.repository!==settings.repository||
           !result.versionsByProject?.[settings.repository]?.some(v=>
             v.id===settings.versionId&&v.projectId===settings.repository))
          return deny(result?.reason??'CATALOG_SOURCE_UNQUALIFIED');
        return result;
      }catch{return deny('CATALOG_READ_FAILED');}
    },
    async tasks(scope){
      const observed=await readQualifiedRoster(scope);
      if(observed.connected!==true)return observed;
      return {connected:true,source:observed.source,
        versionId:settings.versionId,
        rosterCoverage:observed.rosterCoverage,
        dependencyCoverage:observed.dependencyCoverage,
        taskCount:observed.taskCount,nativeEdgeCount:observed.nativeEdgeCount,
        frozenBlobSha:observed.frozenBlobSha,taskPackRef:observed.taskPackRef,
        claimReadbackCoverage:observed.claimReadbackCoverage,
        claimCoverage:observed.claimCoverage,tasks:observed.tasks};
    },
    async dag(scope){
      const observed=await readQualifiedRoster(scope);
      if(observed.connected!==true)return observed;
      return observed.dag;
    },
    // #264 J02: verbatim dual-epoch projection (S0 frozen historical +
    // live S1 current with the per-edge amendment ledger). The projection
    // itself fails closed on any drift; without the Host mount this answers
    // the honest NOT_CONNECTED-shaped denial below.
    async dualEpoch(scope){
      if(!scopeOk(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
      if(!dualEpochReadHost)return deny('DUAL_EPOCH_NOT_CONFIGURED');
      try{
        return await dualEpochReadHost.dualEpoch(scope);
      }catch{return deny('DUAL_EPOCH_READ_FAILED');}
    },
    async requirements(scope){
      if(!scopeOk(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
      if(!settings.requirements)return deny('REQUIREMENT_ROSTER_NOT_CONFIGURED');
      try{
        const observed=await nativeC.requirements();
        if(observed?.connected!==true||
           observed.source?.repository!==settings.repository||
           observed.versionId!==settings.versionId)
          return deny(observed?.reason??'REQUIREMENT_SOURCE_UNQUALIFIED');
        return observed;
      }catch{return deny('REQUIREMENT_READ_FAILED');}
    },
    async qa(scope){
      if(!scopeOk(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
      if(!settings.qaStatusContext)return deny('QA_STATUS_SOURCE_NOT_CONFIGURED');
      try{
        const observed=await nativeC.qa();
        if(observed?.connected!==true||
           observed.source?.repository!==settings.repository||
           observed.versionId!==settings.versionId)
          return deny(observed?.reason??'QA_SOURCE_UNQUALIFIED');
        return observed;
      }catch{return deny('QA_READ_FAILED');}
    },
    // The only connected Settings capability here is the real local
    // Project connection manager. Other optional Host settings remain absent.
    async settings(){
      if(!env.TALLY_SETUP_URL)return deny('SETTINGS_HOST_NOT_CONFIGURED');
      return {connected:true,configurationUrl:env.TALLY_SETUP_URL,
        selectedRepository:settings.repository,selectedRef:settings.versionId,
        currentSha:settings.expectedSha,
        agentHostMounted:env.TALLY_V2_AGENT_DISPATCH==='host-mounted'};
    },
    // #491 J07: the /app Release panel reads the REAL Host qualification
    // (rqState/ownerGrantState/operation permits from Host-pinned
    // registries, exact-source and exact-live-target bound) alongside the
    // J07a advisory 13-gate projection. Without the mount the authority
    // stays the honest UNKNOWN (permits null); a GRANTED permit can only
    // come from a real Host-issued row — the advisory gates never become
    // authorization.
    async release(scope){
      if(!scopeOk(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
      try{
        const [{projection},authority]=await Promise.all([
          composeReleaseReadiness(),
          releaseQualificationReader.readAuthority()]);
        return {connected:true,versionId:settings.versionId,
          readiness:{versionId:settings.versionId,
            gates:projection.gates.map(gate=>({name:gate.name,
              status:gate.status,reason:gate.reason,owner:null,
              evidenceRef:gate.evidenceRef})),
            gateCounts:projection.gateCounts,blockers:projection.blockers,
            permits:authority.permits,rqState:authority.rqState,
            ownerGrantState:authority.ownerGrantState,
            targetSha:authority.targetSha,targetTree:authority.targetTree,
            targetState:authority.targetState,advisoryOnly:true}};
      }catch{
        return {connected:true,versionId:settings.versionId,
          readiness:{versionId:settings.versionId,gates:[],permits:null,
            rqState:'UNKNOWN',ownerGrantState:'UNKNOWN'}};
      }
    },
  });
  // The legacy J01–J08 console action surface: the run seam is the real
  // installed Builder executor only when the trusted Host mounted it above;
  // otherwise the fail-closed stub keeps every dispatch DENIED-by-absence.
  const config=defaultAlmConfig({
    dhPath:env.TALLY_V2_DH_PATH??join(process.cwd(),'domain-harness'),
    tallyPath:process.cwd(),tallyBPath:process.cwd(),
    githubToken:settings.token,currentSourceSha:settings.expectedSha,
    workRoot:env.TALLY_V2_WORK_ROOT??join(process.cwd(),'.tally-ui-readonly')
  });
  // J07c release evidence host: the controller's pre-read reads the same
  // real native evidence as the /app projection (never form fields). The
  // value sourceSha is bound to the exact served SHA; a foreign request SHA
  // answers UNKNOWN instead of leaking evidence.
  // #264 J07: the readiness is the REAL J07a advisory projection — the
  // observed CI record feeds the woodpeckerCI gate with its exact-SHA
  // subject, pipeline evidenceRef and observedAsOf; every gate without
  // observed evidence stays NOT_RUN/UNKNOWN and the projection can never
  // grant merge/tag/publish authority (advisoryOnly, releasePass:false).
  const composeReleaseReadiness=async()=>{
    let ciStatus='UNKNOWN',ciCoverage='NOT_CONFIGURED',ciGate;
    if(settings.qaStatusContext){
      try{
        const observed=await nativeC.qa();
        if(observed?.connected===true&&
           observed.source?.repository===settings.repository){
          ciCoverage=observed.ciCoverage;
          const record=observed.records[0]??null;
          ciStatus=record?(record.outcome==='PASS'?'PASS':
            record.outcome==='FAIL'?'FAIL':'UNKNOWN'):'NOT_RUN';
          if(record&&record.subjectSha===settings.expectedSha)
            ciGate={status:ciStatus,subjectSha:record.subjectSha,
              observedAsOf:record.observedAt,evidenceRef:record.evidenceRef};
        }
      }catch{ciStatus='UNKNOWN';}
    }
    const projection=projectReleaseReadiness({
      repository:settings.repository,sourceSha:settings.expectedSha,
      observedAsOf:new Date().toISOString(),
      gates:ciGate?{woodpeckerCI:ciGate}:{}});
    return {ciStatus,ciCoverage,projection};
  };
  const readReleaseSnapshot=async({sourceSha}={})=>{
    if(sourceSha!==settings.expectedSha)return null;
    const [{ciStatus,ciCoverage,projection},target]=await Promise.all([
      composeReleaseReadiness(),releaseQualificationReader.readTarget()]);
    // #491: targetSha is the LIVE main HEAD read (never the version source
    // again); unreadable/drifted targets surface UNKNOWN fail-closed.
    return {sourceSha:settings.expectedSha,
      targetSha:target.targetSha,targetTree:target.targetTree,
      targetState:target.targetState,
      readiness:{status:projection.status,reason:projection.reason,
        releasePass:false,ciStatus,ciCoverage,
        gates:projection.gates,gateCounts:projection.gateCounts,
        blockers:projection.blockers,independentRQRequired:true}};
  };
  const server=createAlmServer(config,{
    run:installedRun??(async()=>({status:'FAIL',
      reason:'UI_READ_ONLY_NO_AGENT_DISPATCH',exitCode:1,result:null})),
    releaseHost:{readReleaseSnapshot},
    ...(maintenanceHost?{maintenanceHost}:{}),
    v2ReadModelHost,agentHostMounted:Boolean(installedRun),
    // E0 seam: undefined unless the trusted Host explicitly mounted it above;
    // every other route keeps the exact read-only semantics.
    ...(versionTaskDagSealedDispatch?{versionTaskDagSealedDispatch}:{}),
    // J07 release Host Consumer seam: same trusted-mount-only contract.
    ...(releaseSealedDispatch?{releaseSealedDispatch}:{})
  });
  const address=await server.listen();
  return {address,close:()=>server.close()};
}
const main=process.argv[1]&&resolve(process.argv[1])===
  fileURLToPath(import.meta.url);
if(main){
  import('./first-run-server.mjs')
    .then(({startFirstRunServer})=>startFirstRunServer({
      env:process.env,startConfigured:startUiReadOnlyServer}))
    .then(({address,close})=>{
      process.stdout.write('TALLY_UI='+address.appUrl+'\n');
      process.stdout.write('TALLY_PROJECT_SETUP='+address.origin+'/setup\n');
      let closing=false;
      const stop=()=>{
        if(closing)return;closing=true;
        void close().then(()=>process.exit(0),()=>process.exit(1));
      };
      process.on('SIGINT',stop);process.on('SIGTERM',stop);
    },error=>{
      process.stderr.write(formatStartupError(error));
      process.exitCode=1;
    });
}
