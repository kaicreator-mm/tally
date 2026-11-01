/** Same-origin J01–J08 ALM server. Real actions only through journey/intake/release controllers. */
import http from 'node:http';
import {randomBytes} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import {createJourneyController} from './journey-controller.mjs';
import {createIntakeController} from './intake-controller.mjs';
import {createMaintenanceController,renderMaintenanceWorkspace}
  from './maintenance-controller.mjs';
import {createReleaseController} from './release-controller.mjs';
import {renderReleasePanel} from './release-view.mjs';
import {renderJourneyWorkspace} from './journey-view.mjs';
import {renderIntakeWorkspace} from './intake-view.mjs';
import {readV2HostModel} from './ui-readmodel.mjs';

const CLIENT=String.raw`export async function startJourneyClient(renderJourneyWorkspace){
  const app=document.querySelector('#app'),key=new URLSearchParams(location.search).get('actionKey')??'';
  const annotate=snapshot=>{
    const tasks=Array.isArray(snapshot?.tasks)?snapshot.tasks:[];
    app.querySelectorAll('tr.task-row').forEach(row=>{
      const ref=row.querySelector('a')?.getAttribute('href')??'';
      const task=tasks.find(entry=>entry.sourceRef===ref);if(!task)return;
      for(const [label,value] of [['Hidden',task.hiddenState],['Installed app',task.installedAcceptanceState],['Native read',task.nativeReadbackState]]){
        const td=document.createElement('td');td.dataset.label=label;td.textContent=String(value??'UNKNOWN');
        row.insertBefore(td,row.lastElementChild);
      }});
  };
  const statusLine=text=>{let line=document.querySelector('#action-status');
    if(!line){line=document.createElement('p');line.id='action-status';line.setAttribute('role','status');}
    line.textContent=text;app.querySelector('.workspace').appendChild(line);};
  const report=async()=>{const response=await fetch('/api/snapshot',{cache:'no-store'});const data=await response.json();
    app.innerHTML=renderJourneyWorkspace(data.snapshot,{actions:data.actions,humanActionInbox:[],mode:'write'});
    annotate(data.snapshot);
    statusLine('READY · claim ACCEPTED requires a fresh canonical receipt; shown states are independent layers.');};
  app.addEventListener('click',async event=>{
    const button=event.target.closest('button[data-action]');if(!button)return;
    const kind=button.dataset.action;
    button.disabled=true;button.textContent='RUNNING';
    try{const response=await fetch('/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':key},
      body:JSON.stringify({kind,sourceSha:button.dataset.sourceSha})});
      const result=await response.json();await report();
      statusLine(kind+' '+(result.status??String(response.status))+(result.reason?' '+result.reason:''));
      if(!response.ok)throw new Error(result.reason??'ACTION_FAILED');
    }catch(error){statusLine(kind+' FAILED '+error.message);
      const fresh=app.querySelector('button[data-action="'+kind+'"]');
      if(fresh){fresh.disabled=false;fresh.textContent=kind==='start'?'Start real Codex':'Run Reviewer / Repair';}}
  });
  await report();
}`;
const INTAKE_CLIENT=String.raw`function collectIntakeArgs(){
  const root=document.querySelector('.intake-workspace');if(!root)return {};
  const value=name=>{const field=root.querySelector('[name="'+name+'"]');return field?field.value:undefined;};
  const refs=[...root.querySelectorAll('input[name="evidence"]:checked')].map(box=>box.value);
  const nonGoals=(value('nonGoals')??'').split('\n').map(line=>line.trim()).filter(Boolean);
  const args={};
  if(value('goal'))args.goal=value('goal');
  if(refs.length)args.refs=refs;
  if(nonGoals.length)args.nonGoals=nonGoals;
  if(value('l2Decision'))args.l2Decision=value('l2Decision');
  if(value('reviewer'))args.reviewer=value('reviewer');
  if(value('projectTitle'))args.projectTitle=value('projectTitle');
  return args;
}
export async function startIntakeClient(renderIntakeWorkspace){
  const app=document.querySelector('#intake-app'),key=new URLSearchParams(location.search).get('actionKey')??'';
  let sourceSha='';
  const report=async()=>{const response=await fetch('/api/intake/snapshot',{cache:'no-store'});
    const data=await response.json();sourceSha=data.snapshot.sourceSha;
    app.innerHTML=renderIntakeWorkspace(data.snapshot,{mode:'write',result:data.lastResult});};
  app.addEventListener('click',async event=>{
    const button=event.target.closest('button[data-action]');if(!button)return;
    const label=button.textContent;button.disabled=true;button.textContent='RUNNING';
    try{const response=await fetch('/api/intake/actions',{method:'POST',
      headers:{'content-type':'application/json','x-tally-action-key':key},
      body:JSON.stringify({kind:button.dataset.action,sourceSha,args:collectIntakeArgs()})});
      const result=await response.json();await report();
      if(!response.ok)throw new Error(result.reason??result.status??'ACTION_FAILED');
      const status=document.querySelector('[data-intake-result]');
      if(status)status.textContent='last action: '+result.status;
    }catch(error){button.textContent=error.message;button.disabled=false;}
  });
  await report();
}`;
const BOOTSTRAP=String.raw`import('./journey-view.mjs')
  .then(({renderJourneyWorkspace})=>import('./journey-client.mjs')
    .then(({startJourneyClient})=>startJourneyClient(renderJourneyWorkspace))
    .catch(error=>{throw new Error('CLIENT_MODULE_OR_START: '+error.message);}))
  .catch(error=>{
    document.querySelector('#app').innerHTML='<section class="error-panel" role="alert"><h2>WORKSPACE_START_FAILED</h2><p></p></section>';
    document.querySelector('.error-panel p').textContent=error?.message??String(error);
  });`;
const INTAKE_BOOTSTRAP=String.raw`import('./intake-view.mjs')
  .then(({renderIntakeWorkspace})=>import('./intake-client.mjs')
    .then(({startIntakeClient})=>startIntakeClient(renderIntakeWorkspace))
    .catch(error=>{throw new Error('INTAKE_MODULE_OR_START: '+error.message);}))
  .catch(error=>{
    const app=document.querySelector('#intake-app');
    if(app)app.innerHTML='<section class="error-panel" role="alert"><h2>INTAKE_START_FAILED</h2><p></p></section>';
    const panel=document.querySelector('#intake-app .error-panel p');
    if(panel)panel.textContent=error?.message??String(error);
  });`;
// The maintenance workspace renders server-side (one shared HTML renderer in
// maintenance-controller.mjs); the client only fetches HTML, collects guided
// form fields and posts sealed same-origin actions. Privileged effects never
// leave the sealedDispatch seam.
const MAINTENANCE_CLIENT=String.raw`export async function startMaintenanceClient(){
  const app=document.querySelector('#maintenance-app'),key=new URLSearchParams(location.search).get('actionKey')??'';
  const value=name=>{const field=app.querySelector('[name="'+name+'"]');
    return field?field.value.trim():undefined;};
  const collectArgs=kind=>{
    if(kind==='report-defect'){const args={};
      for(const name of ['summary','requirementIssue','defectIssue','releaseVersion','releaseCommitSha']){
        const v=value(name);if(v)args[name]=v;}
      return args;}
    if(kind==='review-candidate'){const reviewer=value('reviewer');return reviewer?{reviewer}:{};}
    if(kind==='record-learning'){const args={learningId:value('learningId')??'',
      decision:value('decision')??'ADOPTED'};
      if(!args.learningId)delete args.learningId;
      return args;}
    return {};
  };
  const report=async()=>{const response=await fetch('/api/maintenance/snapshot',{cache:'no-store'});
    const data=await response.json();app.innerHTML=data.html;
    app.dataset.sourceSha=data.snapshot.sourceSha;};
  app.addEventListener('click',async event=>{
    const button=event.target.closest('button[data-action]');if(!button)return;
    const kind=button.dataset.action;
    button.disabled=true;button.textContent='RUNNING';
    try{const response=await fetch('/api/maintenance/actions',{method:'POST',
      headers:{'content-type':'application/json','x-tally-action-key':key},
      body:JSON.stringify({kind,sourceSha:app.dataset.sourceSha,args:collectArgs(kind)})});
      const result=await response.json();await report();
      const status=app.querySelector('[data-maintenance-result]');
      if(status)status.textContent='last action: '+kind+' '+(result.status??String(response.status))+
        (result.reason?' '+result.reason:'');
      if(!response.ok)throw new Error(result.reason??result.status??'ACTION_FAILED');
    }catch(error){await report();
      const fresh=app.querySelector('button[data-action="'+kind+'"]');
      if(fresh){fresh.disabled=false;fresh.textContent=error.message;}}
  });
  await report();
}`;
const MAINTENANCE_BOOTSTRAP=String.raw`import('./maintenance-client.mjs')
  .then(({startMaintenanceClient})=>startMaintenanceClient())
  .catch(error=>{
    const app=document.querySelector('#maintenance-app');
    if(app)app.innerHTML='<section class="error-panel" role="alert"><h2>MAINTENANCE_START_FAILED</h2><p></p></section>';
    const panel=document.querySelector('#maintenance-app .error-panel p');
    if(panel)panel.textContent=error?.message??String(error);
  });`;
// J07c Release qualification panel: read-only server-rendered evidence table.
// The client only fetches the HTML snapshot and posts sealed same-origin
// actions; the buttons stay disabled until a fresh selected K/Host permit
// exists, and a posted action can never bypass the controller's fail-closed
// admission (no credential, token or direct C transport in the browser).
const RELEASE_CLIENT=String.raw`export async function startReleaseClient(){
  const app=document.querySelector('#release-app'),key=new URLSearchParams(location.search).get('actionKey')??'';
  const report=async()=>{const response=await fetch('/api/release/snapshot',{cache:'no-store'});
    const data=await response.json();app.innerHTML=data.html;
    app.dataset.sourceSha=data.snapshot?.sourceSha??'';};
  app.addEventListener('click',async event=>{
    const button=event.target.closest('button[data-release-action]');if(!button)return;
    const kind=button.dataset.releaseAction;
    button.disabled=true;button.textContent='CHECKING';
    try{const response=await fetch('/api/release/actions',{method:'POST',
      headers:{'content-type':'application/json','x-tally-action-key':key},
      body:JSON.stringify({kind,sourceSha:app.dataset.sourceSha})});
      const result=await response.json();await report();
      const status=app.querySelector('[role="status"]');
      if(status)status.textContent='last action: '+kind+' '+(result.status??String(response.status))+
        (result.reason?' '+result.reason:'');
    }catch(error){await report();
      const fresh=app.querySelector('button[data-release-action="'+kind+'"]');
      if(fresh){fresh.disabled=false;fresh.textContent=error.message;}}
  });
  await report();
}`;
const RELEASE_BOOTSTRAP=String.raw`import('./release-client.mjs')
  .then(({startReleaseClient})=>startReleaseClient())
  .catch(error=>{
    const app=document.querySelector('#release-app');
    if(app)app.innerHTML='<section class="error-panel" role="alert"><h2>RELEASE_START_FAILED</h2><p></p></section>';
    const panel=document.querySelector('#release-app .error-panel p');
    if(panel)panel.textContent=error?.message??String(error);
  });`;
const json=(value)=>JSON.stringify(value);
const INTAKE_ACTIONS=['submit-goal','select-evidence','confirm-inspect','draft-product',
  'review-draft','freeze-draft','admit-project-intake','refresh-source-facts'];
const MAINTENANCE_ACTIONS=['report-defect','review-candidate','load-learning',
  'record-learning','publish-maintenance-issue'];
const RELEASE_ACTIONS=['merge','tag','publish'];
const RELEASE_SEAM_STATUSES=new Set(['DENIED','UNKNOWN','NOT_RUN','OBSERVED',
  'RECONCILIATION_REQUIRED','GITHUB_EFFECT_UNKNOWN','STALE']);
/** J07c: a seam result may only carry fail-closed observation shapes; any
 * caller-supplied or malformed PASS/authority shape is refused here. */
function verifyReleaseSeamResult(result){
  if(!result||typeof result!=='object'||Array.isArray(result)||
    !RELEASE_SEAM_STATUSES.has(result.status)||
    result.releasePass!==false||result.authorized===true||
    result.taskPass===true||result.releasePublished===true)return null;
  return result;
}
const boundedString=value=>typeof value==='string'&&value.length>0&&value.length<=256;
function sanitizeMaintenanceArgs(args){
  if(args===undefined)return {};
  if(!args||typeof args!=='object'||Array.isArray(args))return null;
  const out={},known=Object.keys(args);
  if(known.some(key=>!['summary','requirementIssue','defectIssue','releaseVersion',
    'releaseCommitSha','reviewer','learningId','decision','outcome'].includes(key)))
    return null;
  for(const key of ['summary','reviewer'])
    if(args[key]!==undefined){
      if(!boundedString(args[key])||args[key].length>512)return null;out[key]=args[key];}
  for(const key of ['requirementIssue','defectIssue'])
    if(args[key]!==undefined){
      if(typeof args[key]!=='string'||!/^[1-9][0-9]{0,5}$/.test(args[key]))return null;
      out[key]=args[key];}
  if(args.releaseVersion!==undefined){
    if(typeof args.releaseVersion!=='string'||args.releaseVersion.length>32)return null;
    out.releaseVersion=args.releaseVersion;}
  if(args.releaseCommitSha!==undefined){
    if(typeof args.releaseCommitSha!=='string'||!/^[0-9a-f]{40}$/.test(args.releaseCommitSha))
      return null;
    out.releaseCommitSha=args.releaseCommitSha;}
  if(args.learningId!==undefined){
    if(typeof args.learningId!=='string'||!/^[0-9a-f]{64}$/.test(args.learningId))return null;
    out.learningId=args.learningId;}
  if(args.decision!==undefined){
    if(!['ADOPTED','REJECTED'].includes(args.decision))return null;
    out.decision=args.decision;}
  if(args.outcome!==undefined){
    if(!args.outcome||typeof args.outcome!=='object'||Array.isArray(args.outcome)||
      typeof args.outcome.attested!=='boolean'||typeof args.outcome.observedAt!=='string'||
      Number.isNaN(Date.parse(args.outcome.observedAt))||
      typeof args.outcome.note!=='string'||args.outcome.note.length===0||
      args.outcome.note.length>512||Object.keys(args.outcome).some(key=>
        !['attested','observedAt','note'].includes(key)))
      return null;
    out.outcome={attested:args.outcome.attested,observedAt:args.outcome.observedAt,
      note:args.outcome.note};}
  return out;
}
function sanitizeIntakeArgs(args){
  if(args===undefined)return {};
  if(!args||typeof args!=='object'||Array.isArray(args))return null;
  const out={},known=Object.keys(args);
  if(known.some(key=>!['goal','refs','nonGoals','l2Decision','reviewer','projectTitle'].includes(key)))
    return null;
  if(args.goal!==undefined){if(!boundedString(args.goal)||args.goal.length>256)return null;out.goal=args.goal;}
  if(args.refs!==undefined){
    if(!Array.isArray(args.refs)||args.refs.length===0||args.refs.length>16||
      args.refs.some(ref=>!/^[0-9]{1,6}$/.test(ref)))return null;
    out.refs=args.refs;}
  if(args.nonGoals!==undefined){
    if(!Array.isArray(args.nonGoals)||args.nonGoals.length>8||
      args.nonGoals.some(item=>!boundedString(item)))return null;
    out.nonGoals=args.nonGoals;}
  if(args.l2Decision!==undefined){if(!boundedString(args.l2Decision))return null;out.l2Decision=args.l2Decision;}
  if(args.reviewer!==undefined){if(!boundedString(args.reviewer)||args.reviewer.length>128)return null;out.reviewer=args.reviewer;}
  if(args.projectTitle!==undefined){if(!boundedString(args.projectTitle)||args.projectTitle.length>120)return null;out.projectTitle=args.projectTitle;}
  return out;
}
function actionDefinitions(sourceSha){return [
  {kind:'start',authorizationType:'START_ELIGIBLE_CLAIM',reason:'Run real Local Codex through the existing K/Host consumer',
    actorRole:'builder',requiredCapability:'WINDOWS_CODEX_HOST',approvalNeeded:true,costRisk:'LOW',
    evidence:{sourceRef:'https://github.com/kaicreator-mm/tally/issues/196'},
    authorization:'HOST_AND_ADS_REQUIRED',currentness:'REAL_ACTION_READBACK',sourceSha},
  {kind:'review-repair',authorizationType:'START_ELIGIBLE_INDEPENDENT_REVIEW',reason:'Run independent Reviewer Join, bounded Repair and re-Join',
    actorRole:'reviewer',requiredCapability:'INDEPENDENT_REVIEWER',approvalNeeded:true,costRisk:'LOW',
    evidence:{sourceRef:'https://github.com/kaicreator-mm/tally/issues/200'},
    authorization:'HOST_AND_ADS_REQUIRED',currentness:'REAL_ACTION_READBACK',sourceSha}
];}
async function body(req,limit=4096){
  let text='';for await(const chunk of req){text+=chunk;if(text.length>limit)throw new Error('REQUEST_TOO_LARGE');}
  return text;
}
// ---- UI #246: consume ONLY source-qualified selected Host projections.
// No second GitHub transport, no speculative default provider/CI status.
// Missing selected Host adapter degrades to NOT_CONNECTED, not a fabricated 0.
const APP_DIST=new URL('../../apps/tally-web/dist/',import.meta.url);
const MIME={'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8',
  '.css':'text/css; charset=utf-8','.svg':'image/svg+xml','.png':'image/png',
  '.map':'application/json','.json':'application/json'};
/** Returns a response descriptor; the route handler owns the socket send. */
async function staticAppResponse(relative){
  const target=new URL(relative,APP_DIST);
  if(!target.href.startsWith(APP_DIST.href))
    return {status:403,type:'text/plain',body:'FORBIDDEN'};
  try{
    const bytes=await readFile(target);
    const ext=relative.slice(relative.lastIndexOf('.'));
    return {status:200,type:MIME[ext]??'application/octet-stream',body:bytes};
  }catch{
    return {status:404,type:'text/plain',
      body:'APP_BUNDLE_NOT_BUILT (run: npm --prefix apps/tally-web run build)'};
  }
}
export function createAlmServer(config,{run,intakeHost,intakeSealedDispatch,
  intakeJourneySnapshot,maintenanceHost,maintenanceSealedDispatch,
  releaseHost,releaseSealedDispatch,v2ReadModelHost}={}){
  const controller=createJourneyController(config,{run}),
    intake=createIntakeController({currentSourceSha:config.currentSourceSha},
      {host:intakeHost??null,journeySnapshot:intakeJourneySnapshot??(()=>controller.snapshot()),
        // Privileged admit effects flow only through this K-consumer seam;
        // absent a seam the intake admit stays NOT_RUN in the UI.
        sealedDispatch:intakeSealedDispatch??null}),
    maintenance=createMaintenanceController({currentSourceSha:config.currentSourceSha},
      {host:maintenanceHost??null,
        // The only privileged maintenance path (authorized native Issue
        // publication and Task Learning records) is this sealed K seam; the
        // UI has no second scheduler, writer or credential.
        sealedDispatch:maintenanceSealedDispatch??null}),
    // J07c: the Release panel reads only Host-supplied native release
    // readiness (never form fields) and denies every UI action until the
    // selected K/Host admission is independently wired.
    release=createReleaseController({sourceSha:config.currentSourceSha,
      readReleaseSnapshot:typeof releaseHost?.readReleaseSnapshot==='function'?
        args=>releaseHost.readReleaseSnapshot(args):undefined}),
    actionKey=randomBytes(24).toString('hex');
  let lastIntakeResult=null,lastMaintenanceResult=null,lastReleaseResult=null;
  const server=http.createServer(async(req,res)=>{
    const origin=`http://127.0.0.1:${server.address().port}`;
    const send=(status,type,value)=>{res.writeHead(status,{'content-type':type,'cache-control':'no-store',
      'content-security-policy':"default-src 'self'; script-src 'self'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'"});
      res.end(value);};
    try{
      const url=new URL(req.url,origin);
      if(req.method==='GET'&&(url.pathname==='/'||url.pathname==='/journey-view.html')){
        // UI-E feature gate: the production user UI is /app; the legacy test
        // console stays reachable at /diagnostics (and at / directly when
        // TALLY_V2_UI_GATE=legacy for rollback).
        if(url.pathname==='/'&&process.env.TALLY_V2_UI_GATE!=='legacy'){
          res.writeHead(302,{location:'/app'});
          return res.end();
        }
        return send(200,'text/html; charset=utf-8',await readFile(new URL('./journey-view.html',import.meta.url),'utf8'));
      }
      if(req.method==='GET'&&url.pathname==='/journey-view.mjs')
        return send(200,'text/javascript; charset=utf-8',await readFile(new URL('./journey-view.mjs',import.meta.url),'utf8'));
      if(req.method==='GET'&&url.pathname==='/journey-bootstrap.mjs')
        return send(200,'text/javascript; charset=utf-8',BOOTSTRAP);
      if(req.method==='GET'&&url.pathname==='/journey-client.mjs')return send(200,'text/javascript; charset=utf-8',CLIENT);
      if(req.method==='GET'&&url.pathname==='/intake-view.mjs')
        return send(200,'text/javascript; charset=utf-8',await readFile(new URL('./intake-view.mjs',import.meta.url),'utf8'));
      if(req.method==='GET'&&url.pathname==='/intake-bootstrap.mjs')
        return send(200,'text/javascript; charset=utf-8',INTAKE_BOOTSTRAP);
      if(req.method==='GET'&&url.pathname==='/intake-client.mjs')
        return send(200,'text/javascript; charset=utf-8',INTAKE_CLIENT);
      if(req.method==='GET'&&url.pathname==='/maintenance-client.mjs')
        return send(200,'text/javascript; charset=utf-8',MAINTENANCE_CLIENT);
      if(req.method==='GET'&&url.pathname==='/maintenance-bootstrap.mjs')
        return send(200,'text/javascript; charset=utf-8',MAINTENANCE_BOOTSTRAP);
      if(req.method==='GET'&&url.pathname==='/release-client.mjs')
        return send(200,'text/javascript; charset=utf-8',RELEASE_CLIENT);
      if(req.method==='GET'&&url.pathname==='/release-bootstrap.mjs')
        return send(200,'text/javascript; charset=utf-8',RELEASE_BOOTSTRAP);
      if(req.method==='GET'&&url.pathname==='/api/snapshot'){
        const snapshot=controller.snapshot();
        return send(200,'application/json',json({snapshot,actions:actionDefinitions(config.currentSourceSha),humanActionInbox:[]}));
      }
      if(req.method==='GET'&&url.pathname==='/api/intake/snapshot'){
        return send(200,'application/json',json({snapshot:intake.snapshot(),lastResult:lastIntakeResult}));
      }
      if(req.method==='GET'&&url.pathname==='/api/maintenance/snapshot'){
        return send(200,'application/json',json({
          html:renderMaintenanceWorkspace(maintenance.snapshot(),
            {mode:'write',result:lastMaintenanceResult}),
          snapshot:maintenance.snapshot(),lastResult:lastMaintenanceResult}));
      }
      if(req.method==='GET'&&url.pathname==='/api/release/snapshot'){
        const releaseSnapshot=await release.snapshot();
        return send(200,'application/json',json({
          html:renderReleasePanel(releaseSnapshot),snapshot:releaseSnapshot,
          lastResult:lastReleaseResult}));
      }
      if(req.method==='POST'&&url.pathname==='/api/release/actions'){
        if(req.headers['x-tally-action-key']!==actionKey||req.headers.origin!==origin)
          return send(403,'application/json',json({status:'DENIED',reason:'CSRF_OR_ORIGIN'}));
        const payload=JSON.parse(await body(req));
        if(payload.sourceSha!==config.currentSourceSha)
          return send(409,'application/json',json({status:'STALE_SOURCE',reason:'SOURCE_SHA_MISMATCH'}));
        if(!RELEASE_ACTIONS.includes(payload.kind))
          return send(400,'application/json',json({status:'DENIED',reason:'UNKNOWN_ACTION'}));
        // The controller first re-reads the current native release evidence
        // and denies every action that is not freshly admitted; only an
        // explicitly wired selected K/Host seam may carry a request further,
        // and its result must itself be a fail-closed observation shape.
        if(typeof releaseSealedDispatch!=='function'){
          const deniedResult=await release.requestAction({kind:payload.kind,
            sourceSha:payload.sourceSha});
          lastReleaseResult={kind:payload.kind,...deniedResult};
          return send(200,'application/json',json(deniedResult));
        }
        let observed;
        try{
          observed=await releaseSealedDispatch({intentType:'deliverQualifiedRelease',
            input:{kind:payload.kind,sourceSha:payload.sourceSha}});
        }catch{
          const failed={status:'DENIED',reason:'SEAM_ERROR',releasePass:false,
            authorized:false,taskPass:false,mutationPerformed:false};
          lastReleaseResult={kind:payload.kind,...failed};
          return send(200,'application/json',json(failed));
        }
        const verified=verifyReleaseSeamResult(observed)??
          {status:'DENIED',reason:'SEAM_SHAPE_INVALID',releasePass:false,
            authorized:false,taskPass:false,mutationPerformed:false};
        lastReleaseResult={kind:payload.kind,...verified};
        return send(200,'application/json',json(verified));
      }
      if(req.method==='POST'&&url.pathname==='/api/maintenance/actions'){
        if(req.headers['x-tally-action-key']!==actionKey||req.headers.origin!==origin)
          return send(403,'application/json',json({status:'DENIED',reason:'CSRF_OR_ORIGIN'}));
        const payload=JSON.parse(await body(req));
        if(payload.sourceSha!==config.currentSourceSha)
          return send(409,'application/json',json({status:'STALE_SOURCE',reason:'SOURCE_SHA_MISMATCH'}));
        if(!MAINTENANCE_ACTIONS.includes(payload.kind))
          return send(400,'application/json',json({status:'DENIED',reason:'UNKNOWN_ACTION'}));
        const args=sanitizeMaintenanceArgs(payload.args);
        if(args===null)
          return send(400,'application/json',json({status:'DENIED',reason:'ARGS_INVALID'}));
        const result=await maintenance.dispatch(payload.sourceSha,payload.kind,args);
        lastMaintenanceResult={kind:payload.kind,...result};
        return send(200,'application/json',json(result));
      }
      if(req.method==='POST'&&url.pathname==='/api/intake/actions'){
        if(req.headers['x-tally-action-key']!==actionKey||req.headers.origin!==origin)
          return send(403,'application/json',json({status:'DENIED',reason:'CSRF_OR_ORIGIN'}));
        const payload=JSON.parse(await body(req));
        if(payload.sourceSha!==config.currentSourceSha)
          return send(409,'application/json',json({status:'STALE_SOURCE',reason:'SOURCE_SHA_MISMATCH'}));
        if(!INTAKE_ACTIONS.includes(payload.kind))
          return send(400,'application/json',json({status:'DENIED',reason:'UNKNOWN_ACTION'}));
        const args=sanitizeIntakeArgs(payload.args);
        if(args===null)
          return send(400,'application/json',json({status:'DENIED',reason:'ARGS_INVALID'}));
        const result=await intake.dispatch(payload.sourceSha,payload.kind,args);
        lastIntakeResult={kind:payload.kind,...result};
        return send(result.status==='OK'||result.status==='FROZEN_COMPLETE'?200:
          result.status==='STALE_SOURCE'?409:200,'application/json',json(result));
      }
      if(req.method==='POST'&&url.pathname==='/api/actions'){
        if(req.headers['x-tally-action-key']!==actionKey||req.headers.origin!==origin)
          return send(403,'application/json',json({status:'DENIED',reason:'CSRF_OR_ORIGIN'}));
        const payload=JSON.parse(await body(req));
        if(payload.sourceSha!==config.currentSourceSha)
          return send(409,'application/json',json({status:'STALE_SOURCE',reason:'SOURCE_SHA_MISMATCH'}));
        if(!['start','review-repair'].includes(payload.kind))
          return send(400,'application/json',json({status:'DENIED',reason:'UNKNOWN_ACTION'}));
        const result=payload.kind==='start'?await controller.start(payload.sourceSha):
          await controller.reviewRepair(payload.sourceSha);
        return send(result.status==='PASS'?200:502,'application/json',json(result));
      }
      // ---- UI-E #232: v2 shell read models + feature gate -----------------
      // Honest read models: no token / unreachable GitHub ⇒ NOT_CONNECTED or
      // UNKNOWN, never fabricated data. No effect path starts here.
      if(req.method==='GET'&&url.pathname==='/api/v2/catalog'){
        const model=await readV2HostModel(v2ReadModelHost,'catalog',{});
        return send(200,'application/json',json(model));
      }
      if(req.method==='GET'&&url.pathname==='/api/v2/tasks'){
        const model=await readV2HostModel(v2ReadModelHost,'tasks',{
          projectId:url.searchParams.get('projectId')??undefined,
          versionId:url.searchParams.get('versionId')??undefined});
        return send(200,'application/json',json(model));
      }
      if(req.method==='GET'&&url.pathname==='/api/v2/dag'){
        const model=await readV2HostModel(v2ReadModelHost,'dag',{
          projectId:url.searchParams.get('projectId')??undefined,
          versionId:url.searchParams.get('versionId')??undefined});
        return send(200,'application/json',json(model));
      }
      if(req.method==='GET'&&url.pathname==='/api/v2/release'){
        const model=await readV2HostModel(v2ReadModelHost,'release',{
          projectId:url.searchParams.get('projectId')??undefined,
          versionId:url.searchParams.get('versionId')??undefined});
        return send(200,'application/json',json(model));
      }
      if(req.method==='GET'&&url.pathname==='/api/v2/settings'){
        const model=await readV2HostModel(v2ReadModelHost,'settings',{});
        return send(200,'application/json',json(model));
      }
      if(req.method==='GET'&&url.pathname==='/api/v2/metadata'){
        const model=await readV2HostModel(v2ReadModelHost,'metadata',{
          projectId:url.searchParams.get('projectId')??undefined,
          versionId:url.searchParams.get('versionId')??undefined});
        return send(200,'application/json',json(model));
      }
      // Feature gate: /app serves the new shell bundle (built by
      // apps/tally-web); /diagnostics keeps the legacy test console; / picks
      // by TALLY_V2_UI_GATE (default open ⇒ new shell).
      if(req.method==='GET'&&(url.pathname==='/app'||url.pathname==='/app/')){
        const served=await staticAppResponse('index.html');
        return send(served.status,served.type,served.body);
      }
      if(req.method==='GET'&&url.pathname.startsWith('/app/assets/')){
        const served=await staticAppResponse(url.pathname.slice('/app/'.length));
        return send(served.status,served.type,served.body);
      }
      if(req.method==='GET'&&url.pathname==='/diagnostics'){
        return send(200,'text/html; charset=utf-8',
          await readFile(new URL('./journey-view.html',import.meta.url),'utf8'));
      }
      return send(404,'text/plain','NOT_FOUND');
    }catch(error){return send(400,'application/json',json({status:'ERROR',reason:error.message}));}
  });
  return Object.freeze({server,actionKey,async listen(){
    await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
    return {origin:`http://127.0.0.1:${server.address().port}`,actionKey};
  },async close(){await new Promise(resolve=>server.close(resolve));}});
}
