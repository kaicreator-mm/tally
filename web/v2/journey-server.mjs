/** Same-origin J01–J06 ALM server. Real actions only through journey/intake controllers. */
import http from 'node:http';
import {randomBytes} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import {createJourneyController} from './journey-controller.mjs';
import {createIntakeController} from './intake-controller.mjs';
import {createMaintenanceController,renderMaintenanceWorkspace}
  from './maintenance-controller.mjs';
import {renderJourneyWorkspace} from './journey-view.mjs';
import {renderIntakeWorkspace} from './intake-view.mjs';

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
const json=(value)=>JSON.stringify(value);
const INTAKE_ACTIONS=['submit-goal','select-evidence','confirm-inspect','draft-product',
  'review-draft','freeze-draft','admit-project-intake','refresh-source-facts'];
const MAINTENANCE_ACTIONS=['report-defect','review-candidate','load-learning',
  'record-learning','publish-maintenance-issue'];
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
export function createAlmServer(config,{run,intakeHost,intakeSealedDispatch,
  intakeJourneySnapshot,maintenanceHost,maintenanceSealedDispatch}={}){
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
    actionKey=randomBytes(24).toString('hex');
  let lastIntakeResult=null,lastMaintenanceResult=null;
  const server=http.createServer(async(req,res)=>{
    const origin=`http://127.0.0.1:${server.address().port}`;
    const send=(status,type,value)=>{res.writeHead(status,{'content-type':type,'cache-control':'no-store',
      'content-security-policy':"default-src 'self'; script-src 'self'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'"});
      res.end(value);};
    try{
      const url=new URL(req.url,origin);
      if(req.method==='GET'&&(url.pathname==='/'||url.pathname==='/journey-view.html')){
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
      return send(404,'text/plain','NOT_FOUND');
    }catch(error){return send(400,'application/json',json({status:'ERROR',reason:error.message}));}
  });
  return Object.freeze({server,actionKey,async listen(){
    await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
    return {origin:`http://127.0.0.1:${server.address().port}`,actionKey};
  },async close(){await new Promise(resolve=>server.close(resolve));}});
}
