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
 */
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {createUiV2ReadModelHost} from './ui-native-host.mjs';
import {createUiNativeSource}
  from '../../packages/tally-github-native-c/src/ui-version-readonly.mjs';
import {createUiAdsReportHost} from './ui-report-host.mjs';
import {createUiNativeAuditHost} from './ui-audit-host.mjs';

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
  const [owner,repo]=repository.split('/');
  return {repository,owner,repo,versionId,expectedSha,roster,requirements,ads,
    auditReadOnly:auditMode==='read-only',token:env.TALLY_GITHUB_READ_TOKEN};
}
export async function startUiReadOnlyServer({env=process.env}={}){
  const settings=readUiHostEnvironment(env);
  const nativeC=createUiNativeSource({token:settings.token,
    owner:settings.owner,repo:settings.repo,
    versionId:settings.versionId,expectedSha:settings.expectedSha,
    taskIssueNumbers:settings.roster,
    requirementIssueSpecs:settings.requirements});
  const nativeReadHost=createUiV2ReadModelHost({
    repository:settings.repository,versionId:settings.versionId,nativeC});
  const adsReadHost=settings.ads?createUiAdsReportHost({
    repository:settings.repository,versionId:settings.versionId,
    expectedSha:settings.expectedSha,nativeC,
    taskIssueNumbers:settings.roster,
    acceptedCommentIdsByIssue:settings.ads.grants,
    publisherBindings:settings.ads.bindings,
    token:settings.token}):{};
  const auditReadHost=settings.auditReadOnly?createUiNativeAuditHost({
    repository:settings.repository,versionId:settings.versionId,
    expectedSha:settings.expectedSha,taskIssueNumbers:settings.roster,
    nativeC,token:settings.token}):{};
  const v2ReadModelHost=Object.freeze({
    ...nativeReadHost,...adsReadHost,...auditReadHost});
  // The legacy J01–J08 console remains fail-closed in this entrypoint.
  // No process executor or selected K dispatch is granted by launching UI.
  const config=defaultAlmConfig({
    dhPath:env.TALLY_V2_DH_PATH??join(process.cwd(),'domain-harness'),
    tallyPath:process.cwd(),tallyBPath:process.cwd(),
    githubToken:settings.token,currentSourceSha:settings.expectedSha,
    workRoot:env.TALLY_V2_WORK_ROOT??join(process.cwd(),'.tally-ui-readonly')
  });
  const server=createAlmServer(config,{
    run:async()=>({status:'FAIL',reason:'UI_READ_ONLY_NO_AGENT_DISPATCH',
      exitCode:1,result:null}),
    v2ReadModelHost
  });
  const address=await server.listen();
  return {address,close:()=>server.close()};
}
const main=process.argv[1]&&resolve(process.argv[1])===
  fileURLToPath(import.meta.url);
if(main){
  startUiReadOnlyServer().then(({address,close})=>{
    process.stdout.write('TALLY_UI_READ_ONLY='+address.origin+'/app\n');
    process.stdout.write('TALLY_UI_AGENT_DISPATCH=DENIED\n');
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
