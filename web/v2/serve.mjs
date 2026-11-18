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
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {createUiNativeSource,readUiNativeVersionFromClient}
  from '../../packages/tally-github-native-c/src/ui-version-readonly.mjs';
import {createUiRosterHost} from './ui-roster-host.mjs';
import {readInstalledWriteJourneyEnvironment,createInstalledWriteJourneyRunner}
  from './installed-write-journeys.mjs';

const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const REF=/^version\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
export function readUiHostEnvironment(env){
  const repository=env.TALLY_V2_REPOSITORY;
  const versionId=env.TALLY_V2_VERSION_REF;
  const expectedSha=env.TALLY_V2_VERSION_SHA;
  if(!REPO.test(repository??'')||!REF.test(versionId??'')||
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
  const [owner,repo]=repository.split('/');
  return {repository,owner,repo,versionId,expectedSha,roster,requirements,qaStatusContext,ads,
    auditReadOnly:auditMode==='read-only',
    workbenchReadOnly:workbenchMode==='read-only',
    frozenDagReadOnly:frozenDagMode==='read-only',
    frozenPins,
    token:env.TALLY_GITHUB_READ_TOKEN};
}
/** #264 E1: read-only frozen version Task DAG reader composition. Mirrors the
 * E0 host composition contract (experiments/v2/selected-business-build/
 * version-task-dag-host.mjs) but grants NO write port: one dedicated CAS
 * connection per read, GitHub blob/comment re-reads bound to the exact
 * PackBlob + selected source SHA. Heavy transports load lazily so a plain
 * read-only boot never requires the Host-only dependency chain. */
async function createFrozenDagReadHost({settings}){
  const [{Octokit},{createHostVersionTaskDagReader},{openSqliteCasStore}]=
    await Promise.all([
      import('../../packages/tally-github-native-c'+
        '/node_modules/@octokit/rest/dist-src/index.js'),
      import('../../packages/tally-github-native-c/src/closeout-roster.mjs'),
      import('@kaicreator/domain-harness-host-node')]);
  if(typeof Octokit!=='function'||
     typeof createHostVersionTaskDagReader!=='function'||
     typeof openSqliteCasStore!=='function')
    throw new TypeError('FROZEN_DAG_READER_TRANSPORTS_UNAVAILABLE');
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
  const witnessReader=(issueNumber,commentId,login,kind)=>
    async({taskPackBlobSha,sourceSha})=>{
      if(!Number.isSafeInteger(issueNumber)||issueNumber<1||
         !Number.isSafeInteger(commentId)||commentId<1)
        return {status:'NOT_VERIFIED'};
      try{
        const params={owner:settings.owner,repo:settings.repo,
          comment_id:commentId};
        const first=(await octokit.rest.issues.getComment(params))?.data;
        const second=(await octokit.rest.issues.getComment(params))?.data;
        const valid=c=>plain(c)&&c.id===commentId&&
          c.html_url==='https://github.com/'+settings.repository+'/issues/'+
            issueNumber+'#issuecomment-'+commentId&&
          c.issue_url==='https://api.github.com/repos/'+settings.repository+
            '/issues/'+issueNumber&&
          c.user?.login===login&&typeof c.body==='string'&&
          c.body.length<=65536;
        if(!valid(first)||!valid(second)||first.body!==second.body||
           first.updated_at!==second.updated_at)
          return {status:'NOT_VERIFIED'};
        let witness;
        try{witness=JSON.parse(first.body);}catch{return {status:'NOT_VERIFIED'};}
        if(!plain(witness)||witness.taskPackBlobSha!==taskPackBlobSha||
           witness.sourceSha!==sourceSha)return {status:'NOT_VERIFIED'};
        if(kind==='review'){
          return witness.status==='INDEPENDENTLY_VERIFIED'&&
            witness.reviewedHeadSha===sourceSha&&
            typeof witness.reviewer==='string'&&witness.reviewer&&
            typeof witness.author==='string'&&witness.author&&
            witness.reviewer!==witness.author&&
            typeof witness.reviewRef==='string'&&
            witness.reviewRef.startsWith('https://github.com/'+
              settings.repository+'/')&&
            Number.isFinite(Date.parse(witness.verifiedAt??''))
            ?witness:{status:'NOT_VERIFIED'};
        }
        return witness.status==='OWNER_APPROVED'&&
          witness.owner===settings.owner&&
          typeof witness.approvalRef==='string'&&
          witness.approvalRef.startsWith('https://github.com/'+
            settings.repository+'/')&&
          Number.isFinite(Date.parse(witness.approvedAt??''))
          ?witness:{status:'NOT_VERIFIED'};
      }catch{return {status:'NOT_VERIFIED'};}
    };
  const nativeRosterReader=({taskIssueNumbers})=>readUiNativeVersionFromClient({
    octokit,owner:settings.owner,repo:settings.repo,
    versionId:settings.versionId,expectedSha:settings.expectedSha,
    taskIssueNumbers});
  const pins=settings.frozenPins;
  return Object.freeze({async readFrozenDag(scope={}){
    const store=await openSqliteCasStore({
      path:pins.registryPath,busyTimeoutMs:1000});
    try{
      const reader=createHostVersionTaskDagReader({
        repository:settings.repository,versionId:settings.versionId,
        expectedVersionSha:settings.expectedSha,store,taskPackReader,
        nativeRosterReader,
        reviewWitnessReader:witnessReader(pins.reviewIssue,
          pins.reviewCommentId,pins.reviewLogin,'review'),
        ownerApprovalReader:witnessReader(pins.approvalIssue,
          pins.approvalCommentId,pins.approvalLogin,'approval')});
      return await reader.readFrozenDag(scope);
    }finally{await store.close().catch(()=>{});}
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
  const nativeC=createUiNativeSource({token:settings.token,
    owner:settings.owner,repo:settings.repo,
    versionId:settings.versionId,expectedSha:settings.expectedSha,
    taskIssueNumbers:settings.roster,
    requirementIssueSpecs:settings.requirements,
    qaStatusContext:settings.qaStatusContext});
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
    // #264 E1 R91: real release evidence projection for the /app Release
    // panel. The woodpeckerCI gate reads the actual native commit status for
    // the pinned SHA (when TALLY_V2_QA_STATUS_CONTEXT is configured);
    // independentRq stays UNKNOWN until the #208 exact-SHA RQ flow exists.
    // permits stay null: no release permit authority is mounted at this
    // source, so every action button explains what is missing instead of a
    // dead or fabricated enabled control.
    async release(scope){
      if(!scopeOk(scope))return deny('PROJECT_VERSION_SCOPE_MISMATCH');
      const gates=[{name:'woodpeckerCI',status:'NOT_RUN',owner:'CI'},
        {name:'independentRq',status:'UNKNOWN',owner:'独立验证者'}];
      if(!settings.qaStatusContext)
        return {connected:true,versionId:settings.versionId,
          readiness:{versionId:settings.versionId,gates,permits:null,
            rqState:'UNKNOWN',ownerGrantState:'UNKNOWN'}};
      try{
        const observed=await nativeC.qa();
        if(observed?.connected!==true||
           observed.source?.repository!==settings.repository)
          return {connected:true,versionId:settings.versionId,
            readiness:{versionId:settings.versionId,
              gates:[{...gates[0],status:'UNKNOWN'},gates[1]],permits:null,
              rqState:'UNKNOWN',ownerGrantState:'UNKNOWN'}};
        const record=observed.records[0]??null;
        gates[0].status=record?(record.outcome==='PASS'?'PASS':
          record.outcome==='FAIL'?'FAIL':'UNKNOWN'):'NOT_RUN';
        return {connected:true,versionId:settings.versionId,
          readiness:{versionId:settings.versionId,gates,permits:null,
            rqState:'UNKNOWN',ownerGrantState:'UNKNOWN'}};
      }catch{
        return {connected:true,versionId:settings.versionId,
          readiness:{versionId:settings.versionId,
            gates:[{...gates[0],status:'UNKNOWN'},gates[1]],permits:null,
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
  const readReleaseSnapshot=async({sourceSha}={})=>{
    if(sourceSha!==settings.expectedSha)return null;
    let ciStatus='UNKNOWN',ciCoverage='NOT_CONFIGURED';
    if(settings.qaStatusContext){
      try{
        const observed=await nativeC.qa();
        if(observed?.connected===true&&
           observed.source?.repository===settings.repository){
          ciCoverage=observed.ciCoverage;
          const record=observed.records[0]??null;
          ciStatus=record?(record.outcome==='PASS'?'PASS':
            record.outcome==='FAIL'?'FAIL':'UNKNOWN'):'NOT_RUN';
        }
      }catch{ciStatus='UNKNOWN';}
    }
    return {sourceSha:settings.expectedSha,targetSha:settings.expectedSha,
      readiness:{status:'UNKNOWN',releasePass:false,ciStatus,ciCoverage}};
  };
  const server=createAlmServer(config,{
    run:installedRun??(async()=>({status:'FAIL',
      reason:'UI_READ_ONLY_NO_AGENT_DISPATCH',exitCode:1,result:null})),
    releaseHost:{readReleaseSnapshot},
    v2ReadModelHost,
    // E0 seam: undefined unless the trusted Host explicitly mounted it above;
    // every other route keeps the exact read-only semantics.
    ...(versionTaskDagSealedDispatch?{versionTaskDagSealedDispatch}:{})
  });
  const address=await server.listen();
  return {address,close:()=>server.close()};
}
const main=process.argv[1]&&resolve(process.argv[1])===
  fileURLToPath(import.meta.url);
if(main){
  startUiReadOnlyServer().then(({address,close})=>{
    process.stdout.write('TALLY_UI_READ_ONLY='+address.origin+'/app\n');
    process.stdout.write('TALLY_UI_AGENT_DISPATCH='+
      (process.env.TALLY_V2_AGENT_DISPATCH==='host-mounted'?
        'HOST_MOUNTED_SINGLE_CODEX':'DENIED')+'\n');
    let closing=false;
    const stop=()=>{
      if(closing)return;closing=true;
      void close().then(()=>process.exit(0),()=>process.exit(1));
    };
    process.on('SIGINT',stop);process.on('SIGTERM',stop);
  },()=>{
    // Errors can include provider credentials; never write raw details.
    process.stderr.write('TALLY_UI_START_DENIED: invalid selected Host source\n');
    process.exitCode=1;
  });
}
