/** Same-origin J04–J06 ALM server. Real actions only through journey-controller. */
import http from 'node:http';
import {randomBytes} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import {createJourneyController} from './journey-controller.mjs';
import {renderJourneyWorkspace} from './journey-view.mjs';

const CLIENT=String.raw`export async function startJourneyClient(renderJourneyWorkspace){
  const app=document.querySelector('#app'),key=new URLSearchParams(location.search).get('actionKey')??'';
  const report=async()=>{const response=await fetch('/api/snapshot',{cache:'no-store'});const data=await response.json();
    app.innerHTML=renderJourneyWorkspace(data.snapshot,{actions:data.actions,humanActionInbox:[],mode:'write'});};
  app.addEventListener('click',async event=>{
    const button=event.target.closest('button[data-action]');if(!button)return;
    button.disabled=true;button.textContent='RUNNING';
    try{const response=await fetch('/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':key},
      body:JSON.stringify({kind:button.dataset.action,sourceSha:button.dataset.sourceSha})});
      const result=await response.json();await report();if(!response.ok)throw new Error(result.reason??'ACTION_FAILED');
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
const json=(value)=>JSON.stringify(value);
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
export function createAlmServer(config,{run}={}){
  const controller=createJourneyController(config,{run}),actionKey=randomBytes(24).toString('hex');
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
      if(req.method==='GET'&&url.pathname==='/api/snapshot'){
        const snapshot=controller.snapshot();
        return send(200,'application/json',json({snapshot,actions:actionDefinitions(config.currentSourceSha),humanActionInbox:[]}));
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
