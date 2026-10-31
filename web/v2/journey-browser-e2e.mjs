/** J06 real-browser journey harness. Drives the actual same-origin ALM UI/API over HTTP (and, when
 * a Chrome/Edge binary is present, a real headless browser over CDP) through real user actions and
 * canonical negative journeys. The only injected seam is the Host-side Windows Codex e2e runner
 * (the controller's sanctioned `run` hook); the server, origin/CSRF checks, exact-source checks,
 * controller state machine and rendering are the real production code paths. Missing browser or
 * runtime failures stay GUI_NOT_PROVEN / NOT_RUN — this harness never fabricates success. */
import {spawn} from 'node:child_process';
import {existsSync} from 'node:fs';
import {mkdtemp,rm,mkdir,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {pathToFileURL} from 'node:url';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
import {renderJourneyWorkspace} from './journey-view.mjs';

const REPOSITORY='kaicreator-mm/tally';
const SOURCE_SHA='a207'.repeat(10);
const OBSERVED_AT='2026-11-01T00:00:00.000Z';
const TIMEOUT={pageMs:15000,browserBootMs:20000};
const candidateInner=()=>({state:'LOCAL_CANDIDATE_READY',candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40)});
/** runner factory: each journey gets one explicit, labelled Host-execution outcome. */
const runnerOf=outcome=>async kind=>{
  if(outcome.kind==='accept')return {status:'PASS',exitCode:0,observedAt:OBSERVED_AT,evidenceDir:`C:/evidence/${kind}`,
    result:{status:'PASS',realAgentEffect:true,hiddenVerifier:'HIDDEN_NOT_QUALIFIED',
      installedAcceptance:outcome.installedAcceptance??{state:'NOT_RUN'},
      claimReceipts:outcome.claimReceipts??[],nativeReadback:outcome.nativeReadback,
      ackReceipt:outcome.ackReceipt,result:kind==='start'?candidateInner():{rejoin:{state:outcome.rejoinState??'REVIEW_JOINED'},
        priorAcceptedCheck:outcome.priorAcceptedCheck}}};
  if(outcome.kind==='provider-failure')return {status:'FAIL',exitCode:null,signal:null,timeout:false,
    providerError:'CODEX_SPAWN_FAILED',observedAt:OBSERVED_AT,evidenceDir:`C:/evidence/${kind}`,result:null};
  return {status:'FAIL',exitCode:1,signal:null,timeout:false,observedAt:OBSERVED_AT,evidenceDir:`C:/evidence/${kind}`,
    result:{status:'FAIL',realAgentEffect:false,result:{rejoin:{state:outcome.rejoinState??'NOT_RUN'},
      priorAcceptedCheck:outcome.priorAcceptedCheck}}};
};
const BASE_OUTCOME={kind:'accept',claimReceipts:[]};
/** Fresh canonical receipt shape (selected #207 builder claim, kaicreator-mm/tally only). */
const receipt=(issue,overrides={})=>({repository:REPOSITORY,issue,state:'ACCEPTED',commentId:'6542838856',
  observedAt:OBSERVED_AT,dispatchId:'dispatch:207:v2:20261031:1',...overrides});
const freshConfig=()=>defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/tally',
  githubToken:'harness-secret-token',currentSourceSha:SOURCE_SHA,workRoot:join(tmpdir(),'journey-browser-e2e-work')});
const waitFor=async(fn,{timeoutMs,step=200,label='CONDITION'}={})=>{
  const deadline=Date.now()+(timeoutMs??TIMEOUT.pageMs);
  for(;;){const value=await fn();if(value)return value;
    if(Date.now()>deadline)throw new Error(`WAIT_TIMEOUT:${label}`);
    await new Promise(resolve=>setTimeout(resolve,step));}
};
async function withJourneyApp(outcome,steps){
  const app=createAlmServer(freshConfig(),{run:runnerOf(outcome)});
  const {origin,actionKey}=await app.listen();
  try{return await steps({origin,actionKey,
    snapshot:async()=>fetch(origin+'/api/snapshot',{cache:'no-store'}).then(r=>r.json()),
    post:async(payload,headers={})=>fetch(origin+'/api/actions',{method:'POST',
      headers:{'content-type':'application/json',origin,'x-tally-action-key':actionKey,...headers},
      body:JSON.stringify(payload)}),
    render:async()=>{const data=await (await fetch(origin+'/api/snapshot',{cache:'no-store'})).json();
      return {html:renderJourneyWorkspace(data.snapshot,{actions:data.actions,humanActionInbox:[],mode:'write'}),
        snapshot:data.snapshot,actions:data.actions};}});}
  finally{await app.close();}
}
const exactSource=()=>{
  try{const git=spawn('git',['rev-parse','HEAD'],{windowsHide:true});
    let out='';git.stdout.on('data',chunk=>{out+=chunk;});
    return new Promise(resolve=>{git.once('close',code=>resolve(code===0?out.trim():null));
      git.once('error',()=>resolve(null));});}catch{return Promise.resolve(null);}
};
async function scenario(name,vectors,outcome,steps){
  const started=Date.now();
  try{const detail=await withJourneyApp(outcome,steps);
    return {name,vectors,ok:true,ms:Date.now()-started,...(detail?{detail}:{})};}
  catch(error){return {name,vectors,ok:false,ms:Date.now()-started,detail:String(error?.message??error)};}
}
/** ACCEPTED may only appear with a fresh receipt (claimVerifiedAt); no authority flags ever flip. */
const assertNoFalseSuccess=(snapshot,html)=>{
  const accepted=snapshot.tasks.filter(task=>task.claimState==='ACCEPTED');
  for(const task of accepted)if(!task.claimVerifiedAt)throw new Error(`CLAIM_ACCEPTED_WITHOUT_FRESH_RECEIPT:${task.issue}`);
  for(const task of snapshot.tasks)
    if(task.taskPass!==false||task.releasePass!==false)throw new Error(`UNAUTHORITY_PASS:${task.issue}`);
  if(snapshot.taskPass!==false||snapshot.releasePass!==false)throw new Error('UNAUTHORITY_SNAPSHOT_PASS');
  if(/taskPass=true|releasePass=true/.test(html))throw new Error('HTML_ASSERTS_FALSE_SUCCESS');
  if(accepted.length===0&&/>ACCEPTED</.test(html))throw new Error('HTML_ACCEPTED_WITHOUT_RECEIPT_TASK');
};
/** Every scenario below drives the real served UI/API; nothing is mocked except the Host runner seam. */
const scenarioList=()=>[
  scenario('initial-claim-not-accepted',['J06-INITIAL','V82'],BASE_OUTCOME,async({snapshot,render})=>{
    const data=await snapshot(),{html,actions}=await render();
    for(const task of data.snapshot.tasks)if(task.claimState!=='UNKNOWN')throw new Error(`INITIAL_CLAIM_${task.claimState}:${task.issue}`);
    if(data.humanActionInbox.length!==0)throw new Error('SYNTHETIC_HUMAN_REQUIRED_BLOCKER');
    if(actions.length!==2||!actions.every(action=>action.approvalNeeded===true))throw new Error('AGENT_HANDOFF_BLOCKED');
    if((html.match(/<button/g)??[]).length!==2)throw new Error('DECORATIVE_BUTTON');
    if(data.snapshot.tasks.some(task=>task.hiddenState==='HELD_OUT_VERIFIED'))throw new Error('HIDDEN_PROMOTED_WITHOUT_WITNESS');
    if(JSON.stringify(data).includes('harness-secret-token'))throw new Error('TOKEN_LEAK');
    assertNoFalseSuccess(data.snapshot,html);
    return {claimStates:data.snapshot.tasks.map(task=>task.claimState)};
  }),
  scenario('stale-sha-rejected',['V75-EXACT-SHA'],BASE_OUTCOME,async({post,snapshot,render})=>{
    const stale=await post({kind:'start',sourceSha:'0'.repeat(40)});
    const body=await stale.json();
    if(stale.status!==409||body.status!=='STALE_SOURCE')throw new Error(`STALE_NOT_REJECTED:${stale.status}`);
    const data=await snapshot(),{html}=await render();
    if(data.snapshot.result!==null||data.snapshot.progressSignals.realCandidates!==0)throw new Error('STALE_QUEUED_ACTION');
    assertNoFalseSuccess(data.snapshot,html);
  }),
  scenario('csrf-and-origin-exact',['J06-EXACT-SOURCE'],BASE_OUTCOME,async({origin,post,snapshot,render})=>{
    const wrongKey=await post({kind:'start',sourceSha:SOURCE_SHA},{'x-tally-action-key':'wrong'});
    if(wrongKey.status!==403)throw new Error(`WRONG_KEY_NOT_403:${wrongKey.status}`);
    const noOrigin=await fetch(origin+'/api/actions',{method:'POST',
      headers:{'content-type':'application/json','x-tally-action-key':'wrong'},
      body:JSON.stringify({kind:'start',sourceSha:SOURCE_SHA})});
    if(noOrigin.status!==403)throw new Error(`NO_ORIGIN_NOT_403:${noOrigin.status}`);
    const data=await snapshot(),{html}=await render();
    if(data.snapshot.result!==null)throw new Error('DENIED_ACTION_RECORDED');
    assertNoFalseSuccess(data.snapshot,html);
  }),
  scenario('cross-repo-claim-rejected',['J06-CROSS-REPO'],{kind:'accept',
    claimReceipts:[receipt(196,{repository:'kaicreator-mm/other-repo'})]},async({post,snapshot,render})=>{
    const response=await post({kind:'start',sourceSha:SOURCE_SHA});
    if(response.status!==200)throw new Error(`ACTION_HTTP_${response.status}`);
    const data=await snapshot(),{html}=await render();
    const task=data.snapshot.tasks.find(entry=>entry.issue===196);
    if(task.claimState!=='REPO_MISMATCH')throw new Error(`CROSS_REPO_CLAIM_${task.claimState}`);
    if(task.claimVerifiedAt!==null)throw new Error('CROSS_REPO_VERIFIED_AT');
    if(/taskPass=true|releasePass=true|>ACCEPTED</.test(html))throw new Error('CROSS_REPO_SHOWN_ACCEPTED');
    assertNoFalseSuccess(data.snapshot,html);
    return {claimState:task.claimState};
  }),
  scenario('provider-failure-stays-failed',['J06-PROVIDER'],{kind:'provider-failure'},async({post,snapshot,render})=>{
    const response=await post({kind:'start',sourceSha:SOURCE_SHA});
    const body=await response.json();
    if(response.status!==502||body.status!=='FAIL')throw new Error(`PROVIDER_${response.status}:${body.status}`);
    const data=await snapshot(),{html}=await render();
    const task=data.snapshot.tasks.find(entry=>entry.issue===196);
    if(task.status!=='FAIL'||task.agentAttemptState!=='FAILED'||task.hostState!=='FAIL')
      throw new Error(`PROVIDER_STATE_${task.status}/${task.agentAttemptState}/${task.hostState}`);
    if(task.claimState!=='UNKNOWN')throw new Error(`PROVIDER_CLAIM_${task.claimState}`);
    assertNoFalseSuccess(data.snapshot,html);
  }),
  ...[403,429,500].map(httpStatus=>scenario(`native-read-${httpStatus}`,['J06-NATIVE-READ'],
    {kind:'accept',nativeReadback:{httpStatus,reason:`HOST_READBACK_${httpStatus}`}},
    async({post,snapshot,render})=>{
      const response=await post({kind:'start',sourceSha:SOURCE_SHA});
      const body=await response.json();
      if(response.status!==502||body.status!=='FAIL'||!String(body.reason).startsWith('NATIVE_READBACK_'))
        throw new Error(`NATIVE_${httpStatus}_HTTP_${response.status}:${body.status}/${body.reason}`);
      const data=await snapshot(),{html}=await render();
      const task=data.snapshot.tasks.find(entry=>entry.issue===196);
      const expected=httpStatus===403?'FORBIDDEN':httpStatus===429?'RATE_LIMITED':'SERVER_ERROR';
      if(task.nativeReadbackState!==expected)throw new Error(`NATIVE_STATE_${task.nativeReadbackState}`);
      if(task.agentAttemptState!=='FAILED'||task.claimState!=='UNKNOWN')
        throw new Error(`NATIVE_ATTEMPT_${task.agentAttemptState}/${task.claimState}`);
      assertNoFalseSuccess(data.snapshot,html);
      return {nativeReadbackState:task.nativeReadbackState};
    })),
  scenario('lost-ack-unknown',['J06-LOST-ACK'],{kind:'accept',ackReceipt:{state:'LOST'}},async({post,snapshot,render})=>{
    const response=await post({kind:'start',sourceSha:SOURCE_SHA});
    const body=await response.json();
    if(response.status!==502||body.status!=='UNKNOWN'||body.reason!=='ACK_LOST')
      throw new Error(`LOST_ACK_${response.status}:${body.status}/${body.reason}`);
    const data=await snapshot(),{html}=await render();
    const task=data.snapshot.tasks.find(entry=>entry.issue===196);
    if(task.status!=='UNKNOWN'||task.agentAttemptState!=='UNKNOWN')
      throw new Error(`LOST_ACK_STATE_${task.status}/${task.agentAttemptState}`);
    if(data.snapshot.progressSignals.realCandidates!==0)throw new Error('LOST_ACK_COUNTED_AS_CANDIDATE');
    assertNoFalseSuccess(data.snapshot,html);
  }),
  scenario('tests-pass-without-installed-acceptance',['V76'],{kind:'accept',claimReceipts:[receipt(196)]},
    async({post,snapshot,render})=>{
      const response=await post({kind:'start',sourceSha:SOURCE_SHA});
      const body=await response.json();
      if(response.status!==200||body.status!=='PASS')throw new Error(`V76_HTTP_${response.status}:${body.status}`);
      const data=await snapshot(),{html}=await render();
      const task=data.snapshot.tasks.find(entry=>entry.issue===196);
      if(task.agentAttemptState!=='LOCAL_CANDIDATE_READY')throw new Error(`V76_ATTEMPT_${task.agentAttemptState}`);
      if(task.installedAcceptanceState!=='NOT_RUN')throw new Error(`V76_INSTALLED_${task.installedAcceptanceState}`);
      if(task.releaseState!=='NOT_VERIFIED'||data.snapshot.releasePass!==false||task.releasePass!==false)
        throw new Error('V76_RELEASE_PROMOTED');
      if(task.claimState!=='ACCEPTED'||!task.claimVerifiedAt)throw new Error(`V76_CLAIM_${task.claimState}`);
      if(task.hiddenState!=='NOT_QUALIFIED')throw new Error(`V76_HIDDEN_${task.hiddenState}`);
      assertNoFalseSuccess(data.snapshot,html);
      return {installedAcceptanceState:task.installedAcceptanceState,releasePass:data.snapshot.releasePass};
    }),
  scenario('review-finding-preserves-stale-check',['V74','V75'],{kind:'review-finding',rejoinState:'FINDING',
    priorAcceptedCheck:{preserved:true,commentId:'6538568600'}},async({post,snapshot,render})=>{
    const response=await post({kind:'review-repair',sourceSha:SOURCE_SHA});
    if(response.status!==502)throw new Error(`FINDING_HTTP_${response.status}`);
    const data=await snapshot(),{html}=await render();
    const review=data.snapshot.tasks.find(entry=>entry.issue===200);
    if(review.reviewState!=='FINDING')throw new Error(`V74_REVIEW_${review.reviewState}`);
    if(review.status!=='FAIL')throw new Error(`V74_STATUS_${review.status}`);
    if(review.staleAcceptedCheckState!=='PRESERVED')throw new Error(`V75_STALE_CHECK_${review.staleAcceptedCheckState}`);
    if(review.ciState!=='NOT_VERIFIED'||review.releaseState!=='NOT_VERIFIED')throw new Error('V75_LAYERS_COLLAPSED');
    if(data.snapshot.tasks.some(entry=>entry.claimState==='ACCEPTED'))throw new Error('FINDING_PROMOTED_CLAIM');
    assertNoFalseSuccess(data.snapshot,html);
    return {reviewState:review.reviewState,staleAcceptedCheckState:review.staleAcceptedCheckState};
  }),
  scenario('fresh-canonical-receipt-accepted',['J06-FRESH'],{kind:'accept',claimReceipts:[receipt(196)]},
    async({post,snapshot,render})=>{
      const response=await post({kind:'start',sourceSha:SOURCE_SHA});
      if(response.status!==200)throw new Error(`FRESH_HTTP_${response.status}`);
      const data=await snapshot(),{html}=await render();
      const accepted=data.snapshot.tasks.find(entry=>entry.issue===196);
      if(accepted.claimState!=='ACCEPTED'||accepted.claimVerifiedAt!==OBSERVED_AT)
        throw new Error(`FRESH_CLAIM_${accepted.claimState}/${accepted.claimVerifiedAt}`);
      for(const other of data.snapshot.tasks.filter(entry=>entry.issue!==196))
        if(other.claimState!=='UNKNOWN')throw new Error(`UNRUN_TASK_${other.issue}_${other.claimState}`);
      if(!/>ACCEPTED</.test(html))throw new Error('FRESH_ACCEPTED_NOT_SHOWN');
      assertNoFalseSuccess(data.snapshot,html);
      return {claimState:accepted.claimState,claimVerifiedAt:accepted.claimVerifiedAt};
    }),
];
export {scenarioList};
/** Locate a real Chrome/Edge binary for the GUI witness; absence is reported, never faked.
 * An explicit path is authoritative: no silent fallback to other browsers. */
export function probeBrowser(explicit){
  const candidates=explicit?[explicit]:[process.env.TALLY_BROWSER_PATH,
    'C:/Program Files/Google/Chrome/Application/chrome.exe','C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
    'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe','C:/Program Files/Microsoft/Edge/Application/msedge.exe']
    .filter(Boolean);
  for(const path of candidates)if(existsSync(path))return {found:true,path};
  return {found:false,path:null};
}
/** CDP (DevTools protocol) evaluation helper over the built-in WebSocket. */
async function pageConnection(wsUrl,{timeoutMs=TIMEOUT.pageMs}={}){
  const ws=new WebSocket(wsUrl);
  let nextId=1;const pending=new Map();
  const call=(method,params)=>new Promise((resolve,reject)=>{
    const id=nextId++;const timer=setTimeout(()=>{
      if(pending.has(id)){pending.delete(id);reject(new Error('CDP_TIMEOUT:'+method));}},timeoutMs);
    pending.set(id,{resolve:value=>{clearTimeout(timer);resolve(value);},
      reject:error=>{clearTimeout(timer);reject(error);}});
    try{ws.send(JSON.stringify({id,method,params}));}catch(error){clearTimeout(timer);pending.delete(id);reject(error);}});
  const evaluate=async expression=>{
    const message=await call('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});
    if(message.error)throw new Error('CDP_ERROR:'+JSON.stringify(message.error));
    return message.result?.exceptionDetails?undefined:message.result?.result?.value;
  };
  await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});
    ws.addEventListener('error',()=>reject(new Error('CDP_CONNECT_FAILED')),{once:true});});
  ws.addEventListener('message',event=>{
    const text=typeof event.data==='string'?event.data:Buffer.from(event.data).toString();
    let message;try{message=JSON.parse(text);}catch{return;}
    const entry=pending.get(message.id);if(!entry)return;pending.delete(message.id);
    message.error?entry.reject(new Error('CDP_ERROR:'+JSON.stringify(message.error))):entry.resolve(message);});
  return {evaluate,close:()=>{try{ws.close();}catch{}}};
}
const domRead=`(()=>{const rows=[...document.querySelectorAll('tr.task-row')].map(row=>[...row.children].map(cell=>cell.textContent.trim()));
  const buttons=[...document.querySelectorAll('button[data-action]')].map(button=>({action:button.dataset.action,text:button.textContent.trim(),disabled:button.disabled}));
  return {url:location.href,rows,buttons,actionStatus:document.querySelector('#action-status')?.textContent??null,
    htmlHasAccepted:/>ACCEPTED</.test(document.body.innerHTML),footer:document.querySelector('footer')?.textContent??null};})()`;
/** Drive a real headless browser through the real served UI: initial render -> real button click -> readback. */
export async function driveRealBrowser({origin,actionKey,browserPath}){
  let child=null,page=null,userDir=null;
  const actions=['navigate',`click button[data-action=start]`,'readback-snapshot'];
  try{
    userDir=await mkdtemp(join(tmpdir(),'journey-browser-profile-'));
    const launch=async headlessFlag=>{
      const stderrLines=[];
      const proc=spawn(browserPath,[headlessFlag,'--remote-debugging-port=0',`--user-data-dir=${userDir}`,
        '--no-first-run','--no-default-browser-check','--disable-gpu','about:blank'],
        {windowsHide:true,stdio:['ignore','ignore','pipe']});
      proc.stderr.on('data',chunk=>{stderrLines.push(String(chunk));});
      return {proc,endpoint:()=>{const match=stderrLines.join('').match(/DevTools listening on (ws:\/\/\S+)/);
        return match?match[1]:null;}};
    };
    let session=await launch('--headless=new');
    child=session.proc;
    let wsEndpoint=await waitFor(session.endpoint,{timeoutMs:TIMEOUT.browserBootMs,label:'DEVTOOLS_ENDPOINT'})
      .catch(async()=>{child.kill();
        session=await launch('--headless');child=session.proc;
        return waitFor(session.endpoint,{timeoutMs:TIMEOUT.browserBootMs,label:'DEVTOOLS_ENDPOINT_FALLBACK'});});
    const debugPort=new URL(wsEndpoint).port;
    const version=await fetch(`http://127.0.0.1:${debugPort}/json/version`).then(r=>r.json());
    const target=await fetch(`http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent(origin+'/?actionKey='+actionKey)}`,
      {method:'PUT'}).then(r=>r.json());
    page=await pageConnection(target.webSocketDebuggerUrl);
    const evalQuiet=async expression=>{try{return await page.evaluate(expression);}catch{return undefined;}};
    const initial=await waitFor(async()=>{const dom=await evalQuiet(domRead);
      return dom&&dom.rows.length===4?dom:null;},{timeoutMs:TIMEOUT.pageMs,label:'INITIAL_RENDER'});
    if(initial.htmlHasAccepted)throw new Error('GUI_INITIAL_SHOWS_ACCEPTED');
    if(initial.buttons.length!==2)throw new Error('GUI_DECORATIVE_BUTTONS');
    await waitFor(async()=>{try{await page.evaluate(`document.querySelector('button[data-action="start"]').click()`);return true;}
      catch{return null;}},{timeoutMs:TIMEOUT.pageMs,label:'CLICK_START'});
    const after=await waitFor(async()=>{const dom=await evalQuiet(domRead);
      return dom&&dom.actionStatus&&/^start (PASS|FAIL|UNKNOWN|DENIED|STALE_SOURCE)/.test(dom.actionStatus)?dom:null;},
      {timeoutMs:TIMEOUT.pageMs,label:'ACTION_READBACK'});
    if(!after.htmlHasAccepted)throw new Error('GUI_MISSING_FRESH_ACCEPTED_DISPLAY');
    const acceptedRow=after.rows.find(row=>row.join(' ').includes('LOCAL_CANDIDATE_READY'));
    if(!acceptedRow||!acceptedRow.includes('ACCEPTED'))throw new Error('GUI_ROW_NOT_ACCEPTED');
    return {browser:'GUI_PROVEN',browserPath,browserVersion:version.Browser,actions,
      dom:{initial:{rows:initial.rows,buttons:initial.buttons,actionStatus:initial.actionStatus},
        afterClick:{rows:after.rows,actionStatus:after.actionStatus,footer:after.footer}},
      outcome:'START_PASS_WITH_FRESH_RECEIPT_SHOWN'};
  }catch(error){
    return {browser:'GUI_NOT_PROVEN',browserPath,reason:String(error?.message??error),actions,outcome:'NOT_PROVEN'};
  }finally{page?.close();if(child)try{child.kill();}catch{}
    if(userDir)await new Promise(resolve=>setTimeout(resolve,400))
      .then(()=>rm(userDir,{recursive:true,force:true})).catch(()=>{});}
}
/** The GUI witness gets its own live server with the fresh-canonical-receipt outcome. */
async function guiWitness(browserPath){
  const app=createAlmServer(freshConfig(),{run:runnerOf({kind:'accept',claimReceipts:[receipt(196)]})});
  const {origin,actionKey}=await app.listen();
  try{return await driveRealBrowser({origin,actionKey,browserPath});}
  finally{await app.close();}
}
/**
 * Run the whole J06 journey: real same-origin HTTP scenarios + optional real-browser GUI witness.
 * `browser`: 'none' (API+render only), 'auto' (probe Chrome/Edge), or an explicit binary path.
 */
export async function runJourneyScenarios({browser='auto',evidenceDir}={}){
  const results=[];
  for(const entry of scenarioList())results.push(await entry);
  const failed=results.filter(entry=>!entry.ok);
  const witness={schema:'tally/v2-journey-browser-e2e',project:'tally',version:'v2',repository:REPOSITORY,
    issue:207,sourceSha:SOURCE_SHA,exactHead:await exactSource(),
    generatedAt:new Date().toISOString(),node:process.version,platform:process.platform,
    hostExecution:'INJECTED_RUN_SEAM',realWindowsCodex:'NOT_RUN',
    vectorsCovered:['V74','V75','V76','V82'],
    scenarios:results,ok:failed.length===0,
    ...(failed.length?{failures:failed.map(entry=>({name:entry.name,detail:entry.detail}))}:{})};
  const probe=browser==='none'?{found:false,path:null}:probeBrowser(browser==='auto'?null:browser);
  witness.gui=probe.found?await guiWitness(probe.path):{
    browser:'GUI_NOT_PROVEN',reason:browser==='none'?'BROWSER_DISABLED_BY_FLAG':'BROWSER_NOT_FOUND',outcome:'NOT_PROVEN'};
  if(witness.gui.outcome!=='NOT_PROVEN'&&witness.gui.outcome!=='START_PASS_WITH_FRESH_RECEIPT_SHOWN')
    witness.gui={browser:'GUI_NOT_PROVEN',reason:'GUI_OUTCOME_NOT_RECOGNIZED',outcome:'NOT_PROVEN'};
  if(evidenceDir){await mkdir(evidenceDir,{recursive:true});
    await writeFile(join(evidenceDir,'journey-browser-witness.json'),JSON.stringify(witness,null,2)+'\n');}
  return witness;
}
const isMain=process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href;
if(isMain){
  const args=process.argv.slice(2);
  const mode=args.includes('--no-browser')?'none':args.includes('--browser')?args[args.indexOf('--browser')+1]:'auto';
  const evidenceIndex=args.indexOf('--evidence');
  runJourneyScenarios({browser:mode,evidenceDir:evidenceIndex>=0?args[evidenceIndex+1]:undefined})
    .then(witness=>{console.log(JSON.stringify(witness,null,2));process.exit(witness.ok?0:1);})
    .catch(error=>{console.error(String(error));process.exit(1);});
}
