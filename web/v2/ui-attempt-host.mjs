/**
 * Tally v2 #275 — selected Host, read-only Task Attempt adapter.
 *
 * Callers supply an already-selected Host read port. It MUST expose a trusted
 * bounded Attempt roster; public DH RuntimeObservation alone has no Attempt
 * enumeration, process outcome, token/cost or candidate SHA. No raw SQLite,
 * native Claim publication, private task-meta, or source write APIs here.
 */
import {projectTaskAttempts} from '../../packages/tally-engineering-business/src/attempts.mjs';
const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/;
const noAuthority=Object.freeze({authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const unknown=(reason)=>Object.freeze({status:'UNKNOWN',reason,
  attempts:Object.freeze([]),coverage:Object.freeze({attemptCount:null,
    observedRuntime:0,hostReceipts:0,adsReports:0,metrics:'UNKNOWN'}),...noAuthority});
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const identityValid=i=>REPO.test(i.repository??'')&&i.versionId==='version/v2'&&
  Number.isSafeInteger(i.taskIssueNumber)&&i.taskIssueNumber>0&&
  SHA.test(i.sourceSha??'');
/**
 * The trusted selected Host port contract is explicit:
 *   readTaskAttemptRoster({repository,versionId,taskIssueNumber,sourceSha,limit})
 *   readRuntimeObservations({workflowAddress,packageId,epoch,cursor,limit})
 *   [optional] readAdsTaskReports({repository,versionId,taskIssueNumber,sourceSha,limit})
 * Each returns already-public bounded evidence. Caller-provided Host-shaped
 * objects do not constitute a selected Host; only the server composition
 * mounts an actual credential-bound port.
 */
export async function readTaskAttemptHistoryFromHost({
  selectedHost,repository,versionId,taskIssueNumber,sourceSha
}={}) {
  const identity={repository,versionId,taskIssueNumber,sourceSha};
  if(!identityValid(identity))return unknown('INVALID_SOURCE_IDENTITY');
  if(typeof selectedHost?.readTaskAttemptRoster!=='function')
    return unknown('HOST_ATTEMPT_ROSTER_PORT_MISSING');
  let roster;
  try{roster=await selectedHost.readTaskAttemptRoster({...identity,limit:200});}
  catch{return unknown('HOST_ATTEMPT_ROSTER_READ_FAILED');}
  // Do not fallback to public journal point-lookups or private SQLite scans.
  if(!plain(roster)||roster.status!=='VERIFIED'||roster.complete!==true||
     !Array.isArray(roster.attempts)||roster.attempts.length>200)
    return unknown('HOST_ATTEMPT_ROSTER_UNQUALIFIED');
  const observations=[];
  for(const attempt of roster.attempts){
    if(!plain(attempt)||!ID.test(attempt.attemptId??'')||
      typeof attempt.workflowAddress!=='string'||attempt.workflowAddress.length>512||
      !ID.test(attempt.packageId??'')||
      !Number.isSafeInteger(attempt.epoch)||attempt.epoch<0)
      return unknown('INVALID_HOST_ATTEMPT_ADDRESS');
    if(typeof selectedHost.readRuntimeObservations!=='function') {
      observations.push({attemptId:attempt.attemptId,status:'UNKNOWN',observations:[]});
      continue;
    }
    const rows=[];
    const visited=new Set();
    let cursor=null,status='UNKNOWN',highWatermark=null;
    try {
      for(let page=0;page<16;page++){
        const data=await selectedHost.readRuntimeObservations({
          workflowAddress:attempt.workflowAddress,packageId:attempt.packageId,
          epoch:attempt.epoch,cursor,limit:100});
        if(!plain(data)||!Array.isArray(data.observations)||
          data.observations.length>100 ||
          !['COMPLETE','RETENTION_TRUNCATED','STORAGE_GAP','CURSOR_INVALID','UNKNOWN'].includes(data.status)){
          status='UNKNOWN';break;
        }
        rows.push(...data.observations);
        if(rows.length>1000){status='UNKNOWN';break;}
        if(data.status!=='COMPLETE'){status=data.status;break;}
        if(data.nextCursor==null){
          status='COMPLETE';highWatermark=data.highWatermark;break;
        }
        if(typeof data.nextCursor!=='string'||!data.nextCursor||
          data.nextCursor.length>512||visited.has(data.nextCursor)){
          status='CURSOR_INVALID';break;
        }
        visited.add(data.nextCursor);cursor=data.nextCursor;
      }
    }catch{status='UNKNOWN';}
    observations.push({attemptId:attempt.attemptId,status,observations:rows,
      cursorComplete:status==='COMPLETE',highWatermark});
  }
  let adsReports=[];
  if(typeof selectedHost.readAdsTaskReports==='function'){
    try{
      const result=await selectedHost.readAdsTaskReports({...identity,limit:1000});
      // A report is useful only when already qualified by native C; a Host
      // failure is optional telemetry and cannot block the core Task reader.
      if(plain(result)&&result.status==='VERIFIED'&&
        Array.isArray(result.reports)&&result.reports.length<=1000)
        adsReports=result.reports;
    }catch{adsReports=[];}
  }
  return projectTaskAttempts({...identity,roster,observationPages:observations,
    adsReports});
}
/**
 * Server-only helper: an HTTP adapter must resolve the selected Task identity
 * from its trusted Host, never from a browser-supplied sourceSha/authority flag.
 */
export function createTaskAttemptHttpHandler({
  selectedHost,resolveSelectedTask
}={}) {
  if(typeof resolveSelectedTask!=='function')
    throw new TypeError('SELECTED_TASK_IDENTITY_RESOLVER_REQUIRED');
  return async function handleTaskAttemptHttp(req,res) {
    const respond=(status,payload)=>{
      res.statusCode=status;
      res.setHeader('content-type','application/json; charset=utf-8');
      res.setHeader('cache-control','no-store');
      res.end(JSON.stringify({...payload,...noAuthority}));
    };
    if(req?.method!=='GET')return respond(405,{status:'BLOCKED',reason:'READ_ONLY_GET_REQUIRED'});
    let url;
    try{url=new URL(req.url,'http://localhost');}
    catch{return respond(400,{status:'BLOCKED',reason:'REQUEST_URL_INVALID'});}
    if(url.pathname!=='/api/v2/task-attempts'||[...url.searchParams.keys()].length!==1||
      !url.searchParams.has('taskIssueNumber'))
      return respond(404,{status:'BLOCKED',reason:'READ_ROUTE_NOT_FOUND'});
    const str=url.searchParams.get('taskIssueNumber');
    if(!/^[1-9][0-9]{0,8}$/.test(str??''))
      return respond(400,{status:'BLOCKED',reason:'TASK_ID_INVALID'});
    const n=Number(str);
    let trusted;
    try{trusted=await resolveSelectedTask({req,taskIssueNumber:n});}
    catch{return respond(403,{status:'BLOCKED',reason:'SELECTED_TASK_UNAVAILABLE'});}
    if(!plain(trusted)||trusted.status!=='VERIFIED'||
       trusted.taskIssueNumber!==n||!identityValid(trusted))
      return respond(403,{status:'BLOCKED',reason:'SELECTED_TASK_UNVERIFIED'});
    const result=await readTaskAttemptHistoryFromHost({
      selectedHost,repository:trusted.repository,versionId:trusted.versionId,
      taskIssueNumber:n,sourceSha:trusted.sourceSha});
    return respond(200,result);
  };
}
