/** R157: installed-Chromium page-by-page acceptance of the built /app.
 * Windows-first adaptation of installed-chromium-acceptance.mjs (R156):
 *  - the OS-installed Chrome/Edge is auto-detected (no /usr/bin/chromium);
 *  - GIT_CONFIG_GLOBAL points at an empty file (no /dev/null);
 *  - evidence dir default _tmp/r157, overridable via TALLY_R157_EVIDENCE_DIR.
 * ALL model/GitHub/credential/Host authority is explicitly MOCKED (same
 * boundary as R156). The React production bundle, the Setup HTTP service,
 * the ALM HTTP service, page navigation/clicks/typing, browser JS execution,
 * HTTP requests/responses and local Git candidate commits are REAL.
 * Output: screenshots/, network.har, console.json, click-trace.json,
 * page-results.json, mock-agent-git-evidence.json (all redacted).
 */
import assert from 'node:assert/strict';
import {spawn,spawnSync} from 'node:child_process';
import {mkdtemp,mkdir,readFile,writeFile,rm,stat} from 'node:fs/promises';
import {join} from 'node:path';
import {tmpdir} from 'node:os';
import {createHash} from 'node:crypto';
import {startFirstRunServer} from './first-run-server.mjs';
import {LocalProjectCatalog} from './local-project-catalog.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';

const evidence=process.env.TALLY_R157_EVIDENCE_DIR??
  join(process.cwd(),'_tmp','r157');
const shotDir=join(evidence,'screenshots');
await mkdir(shotDir,{recursive:true});
const tmp=await mkdtemp(join(tmpdir(),'tally-r157-ui-'));
const emptyGitConfig=join(tmp,'empty-gitconfig');
await writeFile(emptyGitConfig,'');
let app=null,chromium=null,client=null;
const clicks=[],net=[],consoleEntries=[],screens=[],mockRuns=[],
  pageResults=[],gitEvidence=[];
let shotNo=0,currentPage=null;
const startedAt=new Date().toISOString();
const sleep=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const poll=async(f,label,maxMs=16000)=>{
  const until=Date.now()+maxMs;
  while(Date.now()<until){const value=await f();if(value)return value;
    await sleep(140);}
  throw Error('WAIT_'+label);
};
const S=value=>{let text=String(value);
  for(const [pattern,replacement] of [[/actionKey=[a-f0-9]{6,}/g,
    'actionKey=REDACTED'],[/fake-token-r157/g,'REDACTED_TOKEN'],
    [/[?&]token=[^&"'`\s]+/g,'token=REDACTED']])
    text=text.replace(pattern,replacement);
  return text;};
const git=(dir,...args)=>{
  const out=spawnSync('git',['-C',dir,...args],{encoding:'utf8',timeout:20000,
    env:{...process.env,GIT_CONFIG_GLOBAL:emptyGitConfig,
      GIT_CONFIG_NOSYSTEM:'1',GIT_TERMINAL_PROMPT:'0'}});
  assert.equal(out.status,0,'GIT '+(out.stderr??'').slice(0,200));
  return out.stdout.trim();
};
const commitMock=(dir,message)=>git(dir,'-c','user.name=Mock',
  '-c','user.email=mock@localhost','commit','-qm',message);
function mockRunner(repository,sourceSha){
  return async kind=>{
    if(kind!=='start')return {status:'FAIL',exitCode:1};
    const slug=repository.replace('/','-');
    const cwd=join(tmp,'repos',slug);
    const dir=join(tmp,'candidate',slug);
    await mkdir(dirnameOf(dir),{recursive:true});
    git(cwd,'worktree','add','--detach',dir,sourceSha);
    const file=join(dir,'MOCK_CODEX_CANDIDATE.md');
    const outcome=spawnSync(process.execPath,['-e',
      "require('fs').writeFileSync(process.argv[1],'mock codex\\n')",file],
      {encoding:'utf8',timeout:12000});
    assert.equal(outcome.status,0);
    git(dir,'add','MOCK_CODEX_CANDIDATE.md');
    commitMock(dir,'mock agent');
    const commit=git(dir,'rev-parse','HEAD');
    const tree=git(dir,'rev-parse','HEAD^{tree}');
    const parent=git(dir,'rev-parse','HEAD^');
    const diff=git(dir,'diff','--name-only',sourceSha,commit);
    assert.equal(parent,sourceSha);
    assert.equal(diff,'MOCK_CODEX_CANDIDATE.md');
    const entry={commit,tree,sourceSha,parent,repository,diffFile:diff,
      worktree:dir,realGit:true,realCodex:false,
      verifiedAt:new Date().toISOString()};
    mockRuns.push(entry);gitEvidence.push(entry);
    return {status:'PASS',exitCode:0,observedAt:new Date().toISOString(),
      evidenceDir:dir,result:{status:'PASS',realAgentEffect:true,
        claimReceipts:[],hiddenVerifier:'HIDDEN_NOT_QUALIFIED',
        installedAcceptance:{state:'NOT_RUN'},
        result:{state:'LOCAL_CANDIDATE_READY',candidateCommit:commit,
          candidateTree:tree}}};
  };
}
const dirnameOf=path=>path.replace(/[\\/][^\\/]+$/,'')||'.';
/** Read models are MOCK; the catalog mirror reads the real store so the
 * TopBar pickers stay consistent with the actually selected Host (the mock
 * boundary is still HOST_READMODELS_MOCK). */
function mockModels(repository,sourceSha,ref,setupUrl,catalog){
  const source={repository,sourceSha,currentness:'CURRENT',readback:'VERIFIED'};
  const data=extra=>({connected:true,source,versionId:ref,...extra});
  return {
    catalog:async()=>{
      const status=await catalog.status();
      return data({
        projects:status.projects.map(project=>({id:project.repository,
          title:project.title??project.repository.split('/')[1],
          repository:project.repository})),
        versionsByProject:Object.fromEntries(status.projects.map(project=>{
          const versions=project.versions.map(version=>({
            id:version.ref,projectId:project.repository,
            label:version.ref}));
          return [project.repository,versions];
        })),
        defaultVersionId:ref});
    },
    tasks:async()=>data({tasks:[{
      id:'196',versionId:ref,projectId:repository,
      title:'Mock Codex acceptance task',status:'open',severity:'P2',
      blockedBy:[],agentId:null,claimState:'ACCEPTED',attemptId:null,
      attemptState:null,prNumber:null,reviewState:'NOT_RUN',ciState:'NOT_RUN',
      writeSet:['MOCK_CODEX_CANDIDATE.md'],requirementRef:null,
      updatedAt:new Date().toISOString()}]}),
    dag:async()=>data({asOf:{sourceSha,frozenAt:null},nodes:[],edges:[]}),
    requirements:async()=>data({nodes:[]}),
    qa:async()=>data({records:[]}),
    release:async()=>data({readiness:{
      versionId:ref,rqState:'UNKNOWN',ownerGrantState:'UNKNOWN',permits:null,
      gates:[{name:'PHYSICAL_HOST',status:'NOT_RUN',owner:'Mock Host'}]}}),
    settings:async()=>data({configurationUrl:setupUrl,
      selectedRepository:repository,selectedRef:ref,currentSha:sourceSha,
      agentHostMounted:true}),
    taskDrafts:async()=>data({drafts:[],repository,
      coverage:'COMPLETE_SINGLE_PAGE'})};
}
function connectCDP(url){
  return new Promise((resolve,reject)=>{
    const socket=new WebSocket(url),pending=new Map();let id=0;
    socket.addEventListener('error',reject,{once:true});
    socket.addEventListener('open',()=>{
      socket.addEventListener('message',event=>{
        const packet=JSON.parse(typeof event.data==='string'?event.data:
          Buffer.from(event.data).toString());
        if(packet.id){
          const item=pending.get(packet.id);if(!item)return;
          pending.delete(packet.id);
          packet.error?item.reject(Error(packet.error.message)):
            item.resolve(packet.result??{});
          return;
        }
        const method=packet.method,params=packet.params??{};
        if(method==='Network.requestWillBeSent')
          net.push({requestId:params.requestId,
            url:S(params.request?.url??''),
            method:params.request?.method??'GET',status:0,statusText:'',
            mime:'',type:'',startedDateTime:new Date().toISOString(),
            failed:null});
        else if(method==='Network.responseReceived'){
          const entry=[...net].reverse().find(row=>
            row.requestId===params.requestId&&!row.status);
          if(entry){entry.status=params.response?.status??0;
            entry.statusText=params.response?.statusText??'';
            entry.mime=params.response?.mimeType??'';
            entry.type=params.type??'';}
        }else if(method==='Network.loadingFailed'){
          const entry=[...net].reverse().find(row=>
            row.requestId===params.requestId&&!row.status);
          if(entry)entry.failed=params.errorText??'failed';
        }else if(method==='Runtime.consoleAPICalled'&&
            ['error','warning'].includes(params.type))
          consoleEntries.push({kind:'console.'+params.type,
            text:S((params.args??[]).map(arg=>arg.value??arg.description??'')
              .join(' ').slice(0,500)),ts:new Date().toISOString(),
            page:currentPage?.key??null});
        else if(method==='Runtime.exceptionThrown')
          consoleEntries.push({kind:'js.exception',
            text:S(JSON.stringify(params.exceptionDetails).slice(0,600)),
            ts:new Date().toISOString(),page:currentPage?.key??null});
        else if(method==='Log.entryAdded'&&params.entry?.level==='error')
          consoleEntries.push({kind:'log.error',source:params.entry.source,
            text:S(params.entry.text??''),url:S(params.entry.url??''),
            ts:new Date().toISOString(),page:currentPage?.key??null});
      });
      const call=(method,params={})=>new Promise((yes,no)=>{
        const n=++id;pending.set(n,{resolve:yes,reject:no});
        socket.send(JSON.stringify({id:n,method,params}));});
      const evaluate=async code=>{
        const result=await call('Runtime.evaluate',{expression:code,
          returnByValue:true,awaitPromise:true});
        if(result.exceptionDetails)throw Error('CHROMIUM_JS_ERROR:'+
          result.exceptionDetails.text);
        return result.result?.value;
      };
      resolve({call,evaluate,close:()=>socket.close()});
    },{once:true});
  });
}
function findBrowserPath(){
  if(process.env.TALLY_BROWSER_PATH)return process.env.TALLY_BROWSER_PATH;
  const candidates=process.platform==='win32'?
    ['C:/Program Files/Google/Chrome/Application/chrome.exe',
     'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
     'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
     'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
     '/usr/bin/chromium','/usr/bin/google-chrome']:
    ['/usr/bin/chromium','/usr/bin/chromium-browser','/usr/bin/google-chrome'];
  for(const path of candidates){
    const probe=spawnSync('node',['-e',
      `require('fs').accessSync(${JSON.stringify(path)})`],{timeout:8000});
    if(probe.status===0)return path;
  }
  throw Error('NO_INSTALLED_CHROMIUM_FOUND');
}
let browserPath='';
async function browser(){
  browserPath=findBrowserPath();
  const profile=join(tmp,'chrome');await mkdir(profile);
  chromium=spawn(browserPath,['--headless=new','--no-sandbox','--disable-gpu',
    '--disable-dev-shm-usage','--remote-allow-origins=*','--lang=zh-CN',
    '--accept-lang=zh-CN','--window-size=1440,1000',
    '--remote-debugging-port=0','--user-data-dir='+profile,'about:blank'],
    {stdio:'ignore'});
  const port=await poll(async()=>{
    try{return Number((await readFile(join(profile,'DevToolsActivePort'),'utf8'))
      .split('\n')[0]);}catch{return null;}
  },'CDP_PORT',25000);
  const targets=await (await fetch('http://127.0.0.1:'+port+'/json')).json();
  client=await connectCDP(targets.find(target=>target.type==='page')
    .webSocketDebuggerUrl);
  for(const method of ['Page.enable','Runtime.enable','Network.enable',
    'Log.enable'])await client.call(method);
}
const evaluate=s=>client.evaluate(s);
const rawUrl=async()=>evaluate('location.href');
async function navigate(url,label){
  await client.call('Page.navigate',{url});
  await poll(()=>evaluate("document.readyState==='complete'"),'GOTO_'+label);
  await sleep(150);
  clicks.push({ts:new Date().toISOString(),kind:'navigate',label,
    url:S(url),page:currentPage?.key??null});
}
async function reload(label){
  await client.call('Page.reload',{ignoreCache:false});
  await poll(()=>evaluate("document.readyState==='complete'"),'RELOAD_'+label);
  await sleep(400);
  clicks.push({ts:new Date().toISOString(),kind:'reload',label,
    url:S(await rawUrl()),page:currentPage?.key??null});
}
const located=selector=>evaluate('Boolean(document.querySelector('+
  JSON.stringify(selector)+'))');
async function click(selector,label,{mustBeEnabled=true}={}){
  await evaluate('(()=>{const e=document.querySelector('+
    JSON.stringify(selector)+');if(e)e.scrollIntoView({block:"center"});})()')
    .catch(()=>{});
  await sleep(80);
  const find='(()=>{const e=document.querySelector('+JSON.stringify(selector)+
    ');if(!e||!e.getClientRects().length)return null;'+
    (mustBeEnabled?'if(e.disabled)return null;':'')+
    'const r=e.getBoundingClientRect();return {x:r.x+r.width/2,'+
    'y:r.y+r.height/2};})()';
  const position=await poll(()=>evaluate(find),'CLICK_'+label);
  await client.call('Input.dispatchMouseEvent',{type:'mousePressed',
    x:position.x,y:position.y,button:'left',clickCount:1});
  await client.call('Input.dispatchMouseEvent',{type:'mouseReleased',
    x:position.x,y:position.y,button:'left',clickCount:1});
  await sleep(140);
  clicks.push({ts:new Date().toISOString(),kind:'click',label,selector,
    url:S(await rawUrl()),page:currentPage?.key??null});
}
/** Find a visible button/anchor by text regex, mark it, then mouse-click it. */
async function clickText(pattern,label){
  const found=await poll(()=>evaluate('(()=>{const re='+pattern.toString()+
    ';const b=[...document.querySelectorAll("button,a")].find(x=>'+
    're.test(x.textContent.trim())&&x.getClientRects().length&&'+
    '(!x.disabled||x.tagName==="A"));if(b){b.dataset.r157mark="1";'+
    'return true;}return false;})()'),'FINDTEXT_'+label);
  assert.equal(found,true,'NO_ELEMENT_FOR_'+label);
  await click('[data-r157mark="1"]',label);
  await evaluate('(()=>{const b=document.querySelector("[data-r157mark]");'+
    'if(b)delete b.dataset.r157mark;})()').catch(()=>{});
}
async function clickMarked(findCode,label,{mustBeEnabled=true}={}){
  await poll(()=>evaluate('(()=>{const e='+findCode+';'+
    'if(e&&e.getClientRects().length'+(mustBeEnabled?'&&!e.disabled':'')+')'+
    '{e.dataset.r157mark="1";return true;}return false;})()'),'MARK_'+label);
  await click('[data-r157mark="1"]',label,{mustBeEnabled});
  await evaluate('(()=>{const b=document.querySelector("[data-r157mark]");'+
    'if(b)delete b.dataset.r157mark;})()').catch(()=>{});
}
async function fill(selector,text,label='fill'){
  await evaluate('(()=>{const e=document.querySelector('+
    JSON.stringify(selector)+');if(e)e.scrollIntoView({block:"center"});})()')
    .catch(()=>{});
  await sleep(80);
  const p=await poll(()=>evaluate('(()=>{const e=document.querySelector('+
    JSON.stringify(selector)+');if(!e||e.disabled||!e.getClientRects().length)'+
    'return null;const r=e.getBoundingClientRect();return {x:r.x+r.width/2,'+
    'y:r.y+r.height/2};})()'),'FIELD_'+label);
  await client.call('Input.dispatchMouseEvent',{type:'mousePressed',
    x:p.x,y:p.y,button:'left',clickCount:1});
  await client.call('Input.dispatchMouseEvent',{type:'mouseReleased',
    x:p.x,y:p.y,button:'left',clickCount:1});
  await client.call('Input.dispatchKeyEvent',{type:'keyDown',key:'a',
    code:'KeyA',windowsVirtualKeyCode:65,modifiers:2});
  await client.call('Input.dispatchKeyEvent',{type:'keyUp',key:'a',
    code:'KeyA',windowsVirtualKeyCode:65,modifiers:2});
  await client.call('Input.insertText',{text});
  await sleep(100);
  clicks.push({ts:new Date().toISOString(),kind:'fill',selector,
    length:text.length,sample:S(text.slice(0,60)),
    url:S(await rawUrl()),page:currentPage?.key??null});
}
/** Native select choice (React onChange reads the change event). */
async function choose(selector,value,label){
  await poll(()=>located(selector),'SELECT_'+label);
  await evaluate('(()=>{const s=document.querySelector('+
    JSON.stringify(selector)+');'+
    'const setter=Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype,'+
    '"value").set;setter.call(s,'+JSON.stringify(value)+');'+
    's.dispatchEvent(new Event("change",{bubbles:true}));return s.value;})()');
  clicks.push({ts:new Date().toISOString(),kind:'select',selector,value,
    label,url:S(await rawUrl()),page:currentPage?.key??null});
}
const bodyHas=text=>evaluate('document.body.innerText.includes('+
  JSON.stringify(text)+')');
async function shot(name){
  const result=await client.call('Page.captureScreenshot',
    {format:'png',captureBeyondViewport:false});
  const bytes=Buffer.from(result.data,'base64');
  shotNo+=1;
  const file=(String(shotNo).padStart(2,'0')+'-'+name+'.png');
  await writeFile(join(shotDir,file),bytes);
  const entry={name:file,bytes:bytes.length,
    sha256:createHash('sha256').update(bytes).digest('hex'),
    page:currentPage?.key??null};
  screens.push(entry);
  if(currentPage)currentPage.screenshots.push(file);
  return entry;
}
/** Page result bookkeeping: verdicts per real page with per-page network and
 * console slices plus an honest mock boundary note. */
function page(key,meta){
  const result={key,...meta,verdict:'PASS',url:null,assertions:[],
    consoleErrors:[],requests:0,screenshots:[],
    mockBoundary:meta.mockBoundary??'MODEL_MOCK · GITHUB_WRITE_MOCK',
    startedAt:new Date().toISOString()};
  const netStart=net.length,consoleStart=consoleEntries.length;
  const api={
    result,
    async run(fn){
      currentPage=result;
      try{
        result.url=S(await rawUrl());
        await fn();
      }catch(error){
        result.verdict='FAIL';
        result.error=S(String(error?.stack??error).slice(0,1500));
        consoleEntries.push({kind:'driver.error',text:result.error,
          ts:new Date().toISOString(),page:key});
        try{await shot('failure-'+key);}catch{}
      }
      if(result.verdict==='PASS'&&
         !result.assertions.every(row=>row.pass))result.verdict='FAIL';
      result.requests=net.length-netStart;
      result.consoleErrors=consoleEntries.slice(consoleStart)
        .filter(entry=>entry.kind!=='console.warning');
      result.finishedAt=new Date().toISOString();
      pageResults.push(result);
      currentPage=null;
      return result.verdict;
    },
    ok(name,pass,detail=''){
      result.assertions.push({name,pass:Boolean(pass),
        detail:S(String(detail).slice(0,300))});
      if(!pass)result.verdict='FAIL';
      return Boolean(pass);
    }};
  return api;
}
function netToHar(){
  const entries=net.map(row=>({
    startedDateTime:row.startedDateTime,time:0,
    _resourceType:row.type||'other',
    request:{method:row.method,url:row.url,httpVersion:'HTTP/1.1',headers:[],
      queryString:row.url.includes('?')?row.url.split('?')[1].split('&')
        .filter(Boolean).map(pair=>{
          const eq=pair.indexOf('=');
          return {name:pair.slice(0,eq),value:S(pair.slice(eq+1))};}):[],
      cookies:[],headersSize:-1,bodySize:0},
    response:{status:row.failed?0:row.status,
      statusText:row.failed?row.failed:row.statusText,httpVersion:'HTTP/1.1',
      headers:[],content:{size:0,mimeType:row.mime||
        'application/octet-stream',
        comment:'Bodies not captured (evidence must not carry tokens)'},
      cookies:[],headersSize:-1,bodySize:-1,redirectURL:''},
    cache:{},timings:{blocked:-1,dns:-1,connect:-1,send:0,wait:0,receive:0,
      ssl:-1},pageref:'page_1'}));
  return {log:{version:'1.2',creator:{name:'tally-r157-cdp-acceptance',
    version:'1.0'},pages:[{startedDateTime:startedAt,id:'page_1',
    pageTimings:{onContentLoad:-1,onLoad:-1},
    title:'Tally v2 R157 installed /app acceptance'}]},entries};
}
// ---------------------------------------------------------------------------
// Flow
// ---------------------------------------------------------------------------
const heads=new Map(),vaultMap=new Map();
let catalog=null,setupOrigin='',mainSha='',devSha='',p2Sha='';
const PROJECT1='kaicreator-mm/tally',PROJECT2='mock-owner/project2';
let appUrl=null;
try{
  await stat(join(process.cwd(),'apps/tally-web/dist/index.html'));
  // -- fixture repos: real local git ----------------------------------------
  const commitFile=async(dir,file,content,message)=>{
    await writeFile(join(dir,file),content);
    git(dir,'add',file);
    commitMock(dir,message);
  };
  const mkRepo=async name=>{
    const dir=join(tmp,'repos',name.replace('/','-'));
    await mkdir(dir,{recursive:true});
    git(dir,'init','-q','-b','main');
    await commitFile(dir,'README.md',name+'\n','mock source '+name);
    heads.set(name+'@main',git(dir,'rev-parse','HEAD'));
    return dir;
  };
  const p1dir=await mkRepo(PROJECT1);
  // Product refs are constrained to main|version/* (version-ref.mjs):
  // the second version exercises a legal managed ref.
  const DEV_REF='version/r157-dev';
  git(p1dir,'checkout','-q','-b',DEV_REF);
  await commitFile(p1dir,'DEV.md','dev-line change for R157\n','dev change');
  heads.set(PROJECT1+'@'+DEV_REF,git(p1dir,'rev-parse','HEAD'));
  git(p1dir,'checkout','-q','main');
  await mkRepo(PROJECT2);
  mainSha=heads.get(PROJECT1+'@main');
  devSha=heads.get(PROJECT1+'@'+DEV_REF);
  p2Sha=heads.get(PROJECT2+'@main');
  catalog=new LocalProjectCatalog({dir:join(tmp,'catalog'),
    vault:{seal:async text=>{const id='mock:'+vaultMap.size;
        vaultMap.set(id,text);return id;},
      open:async id=>vaultMap.get(id)??''},
    readHead:async({repository,ref,token})=>{
      if(token!=='fake-token-r157')throw Error('GITHUB_SOURCE_UNAVAILABLE');
      const sha=heads.get(repository+'@'+ref);
      if(!sha)throw Error('GITHUB_SOURCE_UNAVAILABLE');
      return sha;}});
  app=await startFirstRunServer({env:{TALLY_V2_CONFIG_DIR:join(tmp,'catalog')},
    catalog,
    startConfigured:async({env})=>{
      const repo=env.TALLY_V2_REPOSITORY,sha=env.TALLY_V2_VERSION_SHA,
        ref=env.TALLY_V2_VERSION_REF;
      assert.equal(heads.get(repo+'@'+ref),sha);
      const server=createAlmServer(defaultAlmConfig({dhPath:tmp,
          tallyPath:tmp,tallyBPath:tmp,
          githubToken:env.TALLY_GITHUB_READ_TOKEN,currentSourceSha:sha,
          workRoot:join(tmp,'work',repo.replace('/','-')+'-'+ref)}),
        {run:mockRunner(repo,sha),
          v2ReadModelHost:mockModels(repo,sha,ref,env.TALLY_SETUP_URL,catalog),
          agentHostMounted:true,selectedAgentTaskIssue:196});
      const address=await server.listen();
      return {address,close:server.close};
    }});
  setupOrigin=app.address.origin;
  await browser();
  // =========================================================================
  // A. 首次配置
  // =========================================================================
  const a=page('A-setup-first-run',{group:'setup',view:'setup',
    label:'A 首次配置:未配置启动→Setup→加项目→HEAD 显示→/app→刷新后凭据有效',
    mockBoundary:'GITHUB_HEAD_MOCK(本地 git 代替 api.github.com) · VAULT_MOCK · GITHUB_WRITE_MOCK'});
  await a.run(async()=>{
    await navigate(setupOrigin+'/','setup-root-redirect');
    a.ok('未配置时 / 重定向到 /setup',
      (await rawUrl()).includes('/setup'),'location='+(await rawUrl()));
    await poll(()=>located('#add input[name=repository]'),'setup form');
    a.ok('Setup 表单存在(仓库/分支/Token)',
      await located('#add input[name=repository]')&&
      await located('#add input[name=ref]')&&
      await located('#add input[name=token]'));
    a.ok('未配置提示可见(还没有已保存项目)',await bodyHas('还没有已保存项目'));
    a.ok('/api/setup/status 探测已发出且 200',net.some(row=>
      row.url.includes('/api/setup/status')&&row.status===200));
    await shot('setup-unconfigured');
    await fill('#add input[name=repository]',PROJECT1,'repo');
    await fill('#add input[name=ref]','main','ref');
    await fill('#add input[name=token]','fake-token-r157','token');
    await click('#add button[type=submit]','add project1');
    appUrl=await poll(async()=>{
      const u=await rawUrl();
      return u.includes('/app?')&&u.includes('actionKey=')?u:null;
    },'app-with-key',20000);
    a.ok('添加项目后带 actionKey 进入 /app',Boolean(appUrl));
    await poll(()=>located('#root'),'React root');
    await poll(()=>bodyHas('Tally v2'),'React shell');
    a.ok('React 生产 shell 渲染(Tally v2)',true);
    a.ok('POST /api/setup/add 返回 200',net.some(row=>
      row.url.includes('/api/setup/add')&&row.status===200&&
      row.method==='POST'));
    await shot('app-first-entry');
    await navigate(setupOrigin+'/setup','setup-head');
    await poll(()=>evaluate('[...document.querySelectorAll("#projects button")'+
      '].some(b=>b.textContent.includes('+JSON.stringify(mainSha.slice(0,10))+
      '))'),'HEAD display');
    a.ok('Setup 页显示项目 HEAD(main · '+mainSha.slice(0,10)+')',true);
    await shot('setup-head-shown');
    await clickText(/打开 Tally 工作台/,'open workbench');
    await poll(()=>located('#root'),'React after link');
    await reload('app-refresh');
    await poll(()=>located('#root'),'React after reload');
    await navigate((await rawUrl()).split('#')[0]+
      '#/development/product?projectId='+encodeURIComponent(PROJECT1),
      'product-after-refresh');
    await poll(()=>located('input[aria-label="产品目标"]'),'product form');
    const disabled=await evaluate('document.querySelector('+
      '\'input[aria-label="产品目标"]\').matches(":disabled")');
    a.ok('刷新后操作凭据仍有效(Product 表单未禁用)',disabled===false,
      'disabled='+disabled);
    await shot('app-after-refresh-credentials');
  });
  // =========================================================================
  // B. Project / Version
  // =========================================================================
  const b=page('B-project-version-switch',{group:'setup',view:'switch',
    label:'B 项目/版本:概览→双版本→版本切换→第二项目→无跨项目污染',
    mockBoundary:'HOST_READMODELS_MOCK(catalog 镜像真实 store) · GITHUB_HEAD_MOCK'});
  await b.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/workbench/overview?projectId='+encodeURIComponent(PROJECT1),'overview');
    await poll(()=>bodyHas('已连接 GitHub 项目'),'overview connected');
    b.ok('概览显示当前项目 '+PROJECT1,await bodyHas(PROJECT1));
    b.ok('概览显示当前源码 SHA '+mainSha.slice(0,12),
      await bodyHas(mainSha.slice(0,12)));
    await shot('overview-project1');
    await navigate(setupOrigin+'/setup','setup-add-dev');
    await fill('#add input[name=repository]',PROJECT1,'repo');
    await fill('#add input[name=ref]','version/r157-dev','ref');
    await fill('#add input[name=token]','fake-token-r157','token');
    await click('#add button[type=submit]','add dev version');
    await poll(async()=>(await rawUrl()).includes('/app?'),'app-dev',20000);
    appUrl=await rawUrl();
    await navigate(setupOrigin+'/setup','setup-versions');
    await poll(()=>evaluate('[...document.querySelectorAll("#projects button")'+
      '].some(b=>b.textContent.includes('+JSON.stringify(devSha.slice(0,10))+
      '))'),'dev version shown');
    b.ok('同项目两版本(main+version/r157-dev)均显示各自 HEAD',
      await bodyHas(mainSha.slice(0,10))&&await bodyHas(devSha.slice(0,10)));
    await shot('setup-two-versions');
    await clickText(new RegExp('version/r157-dev · '+devSha.slice(0,10)),
      'switch to dev version');
    await poll(async()=>(await rawUrl()).includes('/app?'),'app-after-dev',20000);
    await poll(()=>located('#root'),'React dev');
    await navigate((await rawUrl()).split('#')[0]+
      '#/settings/git-github?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId='+encodeURIComponent('version/r157-dev'),'settings-dev');
    await poll(()=>bodyHas('GitHub 项目与连接'),'settings dev');
    const versionSelected=await evaluate('(()=>{const selects=document.'+
      'querySelectorAll("header .select-field select");'+
      'return {project:selects[0]?.value??null,version:selects[1]?.value??null,'+
      'options:[...selects[1]?.options??[]].map(o=>o.value)};})()');
    b.ok('版本切换后 Settings 显示 dev SHA',await bodyHas(devSha.slice(0,12)));
    b.ok('TopBar 版本选择器当前为 version/r157-dev',
      versionSelected?.version==='version/r157-dev',
      JSON.stringify(versionSelected));
    await shot('settings-version-dev');
    await navigate(setupOrigin+'/setup','setup-back-main');
    await clickText(new RegExp('main · '+mainSha.slice(0,10)),
      'switch back to main');
    await poll(async()=>(await rawUrl()).includes('/app?'),'app-back-main',20000);
    appUrl=await rawUrl();
    await poll(()=>located('#root'),'React main');
    await navigate((await rawUrl()).split('#')[0]+
      '#/settings/git-github?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','settings-main');
    await poll(()=>bodyHas('GitHub 项目与连接'),'settings main');
    b.ok('切回 main 后 Settings 显示 main SHA',
      await bodyHas(mainSha.slice(0,12)));
    await shot('settings-version-back-main');
    await navigate(setupOrigin+'/setup','setup-add-p2');
    await fill('#add input[name=repository]',PROJECT2,'repo2');
    await fill('#add input[name=ref]','main','ref2');
    await fill('#add input[name=token]','fake-token-r157','token2');
    await click('#add button[type=submit]','add project2');
    await poll(async()=>(await rawUrl()).includes('/app?'),'app-p2',20000);
    appUrl=await rawUrl();
    await poll(()=>located('#root'),'React p2');
    await navigate((await rawUrl()).split('#')[0]+
      '#/workbench/overview?projectId='+encodeURIComponent(PROJECT2),'overview-p2');
    await poll(()=>bodyHas('已连接 GitHub 项目'),'overview p2');
    b.ok('第二项目 '+PROJECT2+' 显示其真实 HEAD',
      await bodyHas(PROJECT2)&&await bodyHas(p2Sha.slice(0,12)));
    const projectOptions=await evaluate(
      '[...document.querySelectorAll("header .select-field select")[0].options]'+
      '.map(o=>o.value)');
    b.ok('TopBar 项目选择器包含两个项目',
      projectOptions.includes(PROJECT1)&&projectOptions.includes(PROJECT2),
      JSON.stringify(projectOptions));
    await shot('overview-project2');
    await navigate((await rawUrl()).split('#')[0]+
      '#/development/product?projectId='+encodeURIComponent(PROJECT2),
      'product-p2');
    await poll(()=>bodyHas('其他项目的通用需求建模 Host 尚未接入'),'p2 guard');
    b.ok('第二项目 Product 明确展示未接入状态(不伪造通用流程)',true);
    const p2Disabled=await evaluate('document.querySelector('+
      '\'input[aria-label="产品目标"]\')?.matches(":disabled")');
    b.ok('第二项目 Product 表单禁用',p2Disabled===true,'disabled='+p2Disabled);
    await shot('product-project2-guard');
    await navigate(setupOrigin+'/setup','setup-back-p1');
    await clickText(new RegExp('main · '+mainSha.slice(0,10)),
      'switch back to project1');
    await poll(async()=>(await rawUrl()).includes('/app?'),'app-back-p1',20000);
    appUrl=await rawUrl();
    await poll(()=>located('#root'),'React back p1');
    await navigate((await rawUrl()).split('#')[0]+
      '#/workbench/overview?projectId='+encodeURIComponent(PROJECT1),
      'overview-back');
    await poll(()=>bodyHas('已连接 GitHub 项目'),'overview back');
    b.ok('切回后概览显示 project1 数据(无跨项目污染)',
      await bodyHas(PROJECT1)&&await bodyHas(mainSha.slice(0,12)));
    b.ok('切回后不再渲染 project2 仓库身份',!(await bodyHas(PROJECT2)));
    await shot('back-project1-no-pollution');
  });
  // =========================================================================
  // C. Product / Requirements
  // =========================================================================
  const c=page('C-product-requirements',{group:'development',view:'product',
    label:'C 需求目标→证据选择→确认→L2 方案→Product/L2 草稿→评审提议→Task 衔接',
    mockBoundary:'INTAKE_GUIDED 控制器真实(本地) · FREEZE_SEAM_MOCK(不伪造冻结)'});
  await c.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/development/requirements?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','requirements');
    await poll(()=>bodyHas('暂无需求'),'requirements empty');
    c.ok('Requirements 页面诚实空态(暂无需求)',await bodyHas('暂无需求'));
    await shot('requirements-empty');
    await navigate(appUrl.split('#')[0]+
      '#/development/product?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','product');
    await poll(()=>located('input[aria-label="产品目标"]'),'product form');
    await fill('input[aria-label="产品目标"]',
      'R157 Chromium 验收:为 Windows 安装态工作台建立产品目标基线','goal');
    await clickMarked(
      '[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.trim()==="保存需求目标")','保存需求目标');
    await poll(()=>bodyHas('submit-goal → OK'),'goal response');
    c.ok('保存需求目标 → OK',true);
    await shot('product-goal-saved');
    const boxes=await evaluate('[...document.querySelectorAll(\'section'+
      '[aria-label="Product guided intake"] input[type="checkbox"]\')].length');
    c.ok('存在 3 个内置证据源勾选框',boxes===3,'count='+boxes);
    for(let index=0;index<boxes;index+=1)
      await clickMarked('[...document.querySelectorAll(\'section'+
        '[aria-label="Product guided intake"] input[type="checkbox"]\')]'+
        '['+index+']','evidence-'+index);
    await clickMarked('[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.trim()==="保存证据选择")','保存证据选择');
    await poll(()=>bodyHas('select-evidence → OK'),'evidence saved');
    c.ok('保存证据选择 → OK',true);
    await clickMarked('[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.trim()==="确认证据检查")','确认证据检查');
    await poll(()=>bodyHas('confirm-inspect → OK'),'evidence confirmed');
    c.ok('确认证据检查 → OK',true);
    await shot('product-evidence-confirmed');
    await choose('select[aria-label="L2 方案"]','KEEP_MERGED_L2_ARCHITECTURE',
      'L2 choice');
    await fill('textarea[aria-label="非目标"]','不改变冻结基线','non-goals');
    await clickMarked('[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.trim()==="生成 Product / L2 草稿")','生成 Product / L2 草稿');
    await poll(()=>bodyHas('draft-product → OK'),'draft response');
    await poll(()=>bodyHas('当前 Product 草稿'),'draft article');
    c.ok('生成 Product / L2 草稿 → OK 且草稿状态可见',true);
    await shot('product-draft');
    await fill('input[aria-label="评审者"]','independent-reviewer-r157',
      'reviewer');
    await clickMarked('[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.trim()==="记录评审提议")','记录评审提议');
    await poll(()=>bodyHas('review-draft → OK'),'review proposed');
    c.ok('记录评审提议 → OK',true);
    c.ok('评审状态如实显示 REVIEW_PROPOSED_UNVERIFIED(不伪造独立评审通过)',
      await bodyHas('REVIEW_PROPOSED_UNVERIFIED'));
    c.ok('冻结须真实 Host 的提示可见',await bodyHas('正式冻结仍需真实 Host'));
    await shot('product-review-proposed');
    await clickText(/前往 Task DAG \/ Task 工作台/,'to tasks');
    await poll(async()=>(await rawUrl()).includes('/development/tasks'),
      'tasks hash',10000);
    c.ok('Product → Task 工作台导航衔接',true);
  });
  // =========================================================================
  // D. Task DAG / Task
  // =========================================================================
  const d=page('D-task-dag-and-tasks',{group:'development',view:'tasks',
    label:'D 任务列表→搜索/筛选→列表/看板→详情(依赖/执行状态)→DAG',
    mockBoundary:'TASKS_READMODEL_MOCK(单条 Mock 196) · DAG_READMODEL_MOCK'});
  await d.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/development/tasks?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','tasks-page');
    await poll(()=>located('table tbody tr'),'task row');
    d.ok('任务列表渲染(Mock 196)',await bodyHas('196'));
    await shot('tasks-list');
    await fill('input[aria-label="搜索"]','Mock','search');
    await sleep(250);
    d.ok('搜索 "Mock" 命中当前任务',await located('table tbody tr'));
    await fill('input[aria-label="搜索"]','zzz-no-match','search-miss');
    await poll(()=>bodyHas('没有匹配的任务'),'search empty state',8000)
      .then(()=>d.ok('搜索无命中时诚实空态(没有匹配的任务)',true))
      .catch(()=>d.ok('搜索无命中时诚实空态(没有匹配的任务)',false));
    await fill('input[aria-label="搜索"]','','search-clear');
    await sleep(250);
    await choose('select[aria-label="状态"]','open','status filter');
    await sleep(200);
    d.ok('状态筛选 open 命中',await located('table tbody tr'));
    await choose('select[aria-label="严重级"]','P2','severity filter');
    await sleep(200);
    d.ok('严重级 P2 命中',await located('table tbody tr'));
    await shot('tasks-search-filters');
    await clickMarked('[...document.querySelectorAll("label")].find(label=>'+
      'label.textContent.includes("只看阻塞"))?.querySelector("input")',
      '只看阻塞');
    await poll(()=>bodyHas('没有匹配的任务'),'blocked-only empty',8000)
      .then(()=>d.ok('只看阻塞过滤后诚实空态',true))
      .catch(()=>d.ok('只看阻塞过滤后诚实空态',false));
    // restore the shared list state for the following phases
    await clickMarked('[...document.querySelectorAll("label")].find(label=>'+
      'label.textContent.includes("只看阻塞"))?.querySelector("input")',
      'uncheck 只看阻塞');
    await choose('select[aria-label="状态"]','','status reset');
    await sleep(250);
    await clickMarked('[...document.querySelectorAll(\'[role="group"]'+
      '[aria-label="视图"] button\')].find(x=>x.textContent.trim()==="看板")',
      'switch to board');
    await poll(()=>located('section[aria-label="待处理"]'),'board column');
    d.ok('看板视图渲染(待处理列)',true);
    await shot('tasks-board');
    await clickMarked('[...document.querySelectorAll(\'section'+
      '[aria-label="待处理"] button\')].find(x=>x.textContent.includes("196"))',
      'board card 196');
    await poll(()=>bodyHas('执行与评审'),'drawer open board');
    d.ok('看板卡片打开任务详情抽屉',true);
    await click('button[aria-label="close"]','close drawer board');
    await sleep(200);
    await clickMarked('[...document.querySelectorAll(\'[role="group"]'+
      '[aria-label="视图"] button\')].find(x=>x.textContent.trim()==="列表")',
      'switch back to list');
    await poll(()=>located('table tbody tr'),'list row again');
    await clickMarked('document.querySelector("table tbody tr")',
      'select task row');
    await poll(()=>bodyHas('执行与评审'),'drawer open list');
    d.ok('列表行打开任务详情抽屉(Claim 徽标可见)',
      await bodyHas('Claim:ACCEPTED'));
    d.ok('阻塞原因诚实显示(无阻塞)',await bodyHas('无阻塞'));
    d.ok('执行状态如实可见(Attempt/Review/CI NOT_RUN|UNKNOWN)',
      await bodyHas('NOT_RUN')||await bodyHas('UNKNOWN'));
    // Collapse renders a toggle button[aria-expanded]; never mark container
    // divs (the drawer root is fixed inset:0 — its center is the overlay).
    await clickMarked('[...document.querySelectorAll("button[aria-expanded]")].'+
      'find(x=>x.textContent.includes("技术证据")&&x.getClientRects().length)',
      'expand technical evidence',{mustBeEnabled:false});
    await poll(()=>bodyHas('MOCK_CODEX_CANDIDATE.md'),'write-set visible',8000)
      .then(()=>d.ok('技术证据含写集 MOCK_CODEX_CANDIDATE.md',true))
      .catch(()=>d.ok('技术证据含写集 MOCK_CODEX_CANDIDATE.md',false));
    await shot('task-drawer');
    // Close via Escape (Drawer traps keydown at document level).
    await client.call('Input.dispatchKeyEvent',{type:'keyDown',key:'Escape',
      code:'Escape',windowsVirtualKeyCode:27});
    await client.call('Input.dispatchKeyEvent',{type:'keyUp',key:'Escape',
      code:'Escape',windowsVirtualKeyCode:27});
    await sleep(300);
    d.ok('Esc 关闭任务详情抽屉',!(await bodyHas('执行与评审')));
    await navigate(appUrl.split('#')[0]+
      '#/development/l2-dag?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','l2-dag');
    await poll(()=>bodyHas('版本任务 DAG'),'dag page');
    d.ok('L2 DAG 页面渲染(版本任务 DAG 头+诚实覆盖元数据)',
      await bodyHas('版本任务 DAG')&&await bodyHas('CAS_CLAIM_READBACK'));
    d.ok('空 DAG 如实显示 0 节点 / 0 边',await bodyHas('0 节点'));
    d.ok('双 epoch 面板诚实未连接(NOT_CONNECTED)',await bodyHas('NOT_CONNECTED'));
    await shot('l2-dag');
  });
  // =========================================================================
  // E. Mock Agent
  // =========================================================================
  const e=page('E-mock-agent',{group:'development',view:'agents',
    label:'E 启动 Mock Codex→真实本地 Git 候选提交→执行反馈→Attempt 历史',
    mockBoundary:'AGENT_PROVIDER_MOCK(Mock Codex,不等于真实模型执行) · GITHUB_WRITE_MOCK · 本地 Git 真实'});
  await e.run(async()=>{
    // Leave the tasks route first: navigating to the identical hash would not
    // remount TasksRoute and D's local view state could leak into E.
    await navigate(appUrl.split('#')[0]+
      '#/workbench/inbox?projectId='+encodeURIComponent(PROJECT1),
      'leave tasks route');
    await navigate(appUrl.split('#')[0]+
      '#/development/tasks?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','tasks-for-agent');
    await poll(()=>located('table tbody tr'),'task row again');
    await clickMarked('document.querySelector("table tbody tr")',
      'select task for agent');
    await poll(()=>bodyHas('真实 Agent 执行'),'drawer agent section');
    await poll(()=>evaluate('(()=>{const b=[...document.querySelectorAll'+
      '("button")].find(x=>x.textContent.includes("启动选定 Agent"));'+
      'return Boolean(b&&!b.disabled);})()'),'agent start enabled',12000);
    e.ok('启动按钮可用(Host 匹配 Task/Claim/SHA)',true);
    await clickMarked('[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.includes("启动选定 Agent"))','Mock Codex task start');
    await poll(()=>mockRuns.length===1,'mock codex + real git',25000);
    e.ok('Mock Codex 启动返回 PASS',await bodyHas('PASS'));
    e.ok('候选提交由真实本地 Git 生成',mockRuns[0]?.realGit===true,
      JSON.stringify({commit:mockRuns[0]?.commit?.slice(0,12),
        diff:mockRuns[0]?.diffFile}));
    await shot('agent-start-pass');
    await navigate(appUrl.split('#')[0]+
      '#/development/agents?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','agent history');
    await poll(()=>bodyHas('attempt:'),'attempt row',15000);
    e.ok('Agent Attempt 历史出现真实观测行',await bodyHas('attempt:'));
    e.ok('执行状态 EXITED_OK · provider codex · role BUILDER',
      await bodyHas('EXITED_OK')&&await bodyHas('codex')&&
      await bodyHas('BUILDER'));
    await shot('agent-attempt-history');
  });
  // =========================================================================
  // F. Review / QA
  // =========================================================================
  const f=page('F-review-qa',{group:'development',view:'qa',
    label:'F QA 页面+候选结果展示+未满足资格的正确提示',
    mockBoundary:'QA_READMODEL_MOCK(空记录) · REVIEW 资格不伪造'});
  await f.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/development/qa?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','qa');
    await poll(()=>bodyHas('QA')||bodyHas('验证'),'qa page');
    f.ok('QA 页面渲染',true);
    await shot('qa');
    await navigate(appUrl.split('#')[0]+
      '#/development/tasks?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','tasks-review');
    await poll(()=>located('table tbody tr'),'row for review facts');
    await clickMarked('document.querySelector("table tbody tr")',
      'select for review facts');
    await poll(()=>bodyHas('执行与评审'),'drawer again');
    f.ok('候选执行事实如实展示(Review/CI NOT_RUN · PR —)',
      await bodyHas('Review')&&await bodyHas('NOT_RUN'));
    f.ok('Claim 状态徽标如实展示',await bodyHas('Claim:ACCEPTED'));
    await shot('candidate-review-facts');
    await click('button[aria-label="close"]','close drawer f');
  });
  // =========================================================================
  // G. Release
  // =========================================================================
  const g=page('G-release-gates',{group:'development',view:'release',
    label:'G 发布资格+阻塞原因+未授权按钮禁用(不执行真实 Merge/Tag/Publish)',
    mockBoundary:'RELEASE_MOCK(资格 UNKNOWN) · 禁止真实 GitHub 发布'});
  await g.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/development/release?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','release');
    await poll(()=>bodyHas('发布资格'),'release page');
    g.ok('发布资格面板渲染',true);
    g.ok('阻塞门禁如实显示(RQ_UNKNOWN · GATES_INCOMPLETE · PHYSICAL_HOST)',
      await bodyHas('PHYSICAL_HOST')&&await bodyHas('GATES_INCOMPLETE')&&
      await bodyHas('RQ_UNKNOWN'));
    const disabledStates=await evaluate('[...document.querySelectorAll'+
      '("button")].filter(b=>/发布 Release|合并到 main|创建发布 Tag/.test('+
      'b.textContent)).map(b=>b.disabled)');
    g.ok('Merge/Tag/Publish 按钮全部禁用',
      disabledStates.length===3&&disabledStates.every(Boolean),
      JSON.stringify(disabledStates));
    const netBefore=net.length;
    await clickMarked('[...document.querySelectorAll("button")].find(x=>'+
      'x.textContent.includes("合并到 main"))','try disabled merge',
      {mustBeEnabled:false});
    await sleep(400);
    g.ok('点击禁用按钮未发出 /api/release/actions 请求',
      net.slice(netBefore).every(row=>
        !row.url.includes('/api/release/actions')),
      'new='+JSON.stringify(net.slice(netBefore).map(row=>row.url)));
    await shot('release-blocked');
  });
  // =========================================================================
  // H. Maintenance
  // =========================================================================
  const h=page('H-maintenance',{group:'development',view:'maintenance',
    label:'H 维护快照→进入维护工作台→表单操作→真实状态反馈',
    mockBoundary:'MAINTENANCE_HOST_MOCK(无 sealed K seam,发布 NOT_RUN)'});
  await h.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/development/maintenance?projectId='+encodeURIComponent(PROJECT1),
      'maintenance');
    await poll(()=>bodyHas('当前来源 SHA'),'maintenance snapshot');
    h.ok('React 维护入口渲染真实快照',await bodyHas('当前来源 SHA')&&
      await bodyHas('尚无缺陷候选')&&await bodyHas('NOT_RUN'));
    const href=await evaluate('document.querySelector('+
      '\'a[href*="/diagnostics"]\')?.href??null');
    h.ok('跳转链接携带 actionKey',Boolean(href&&href.includes('actionKey=')),
      S(href??''));
    await shot('maintenance-snapshot');
    await click('a[href*="/diagnostics"]','maintenance workbench link');
    await poll(()=>located('#maintenance-app'),'legacy workbench',15000);
    h.ok('进入缺陷维护工作台(#maintenance-app)',true);
    await poll(()=>located('#maint-summary'),'maint form');
    h.ok('缺陷登记表单可用(summary/requirement 字段)',
      await located('#maint-requirement'));
    await fill('#maint-summary','R157 chromium 验收缺陷登记探针','summary');
    await fill('#maint-requirement','157','requirement issue');
    await fill('#maint-defect-issue','999','defect issue');
    await fill('#maint-release-sha',mainSha,'release sha');
    await click('button[data-action="report-defect"]','report-defect');
    await poll(()=>evaluate('Boolean((document.querySelector('+
      '\'[data-maintenance-result]\')?.textContent??"").includes('+
      '"report-defect"))'),'maintenance feedback',15000);
    const feedback=await evaluate('document.querySelector('+
      '\'[data-maintenance-result]\')?.textContent??""');
    h.ok('report-defect 表单提交获得服务端反馈',
      feedback.includes('report-defect'),S(feedback.slice(0,120)));
    h.ok('发布/冻结类效果保持 NOT_RUN(不伪造 GitHub 发布)',
      await bodyHas('NOT_RUN'));
    await shot('maintenance-workbench');
  });
  // =========================================================================
  // I. Settings / Metadata + 全菜单覆盖
  // =========================================================================
  const i=page('I-settings-metadata-fullmenu',{group:'settings',
    view:'git-github',label:'I 设置/元数据+全部 MVP 菜单页覆盖',
    mockBoundary:'SETTINGS_READMODEL_MOCK · 未挂载 Host 的页面诚实 NOT_CONNECTED'});
  await i.run(async()=>{
    await navigate(appUrl.split('#')[0]+
      '#/settings/git-github?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','settings');
    await poll(()=>bodyHas('GitHub 项目与连接'),'settings section');
    i.ok('设置页显示当前项目/分支/SHA',await bodyHas(PROJECT1)&&
      await bodyHas(mainSha.slice(0,12)));
    i.ok('Agent Host 状态如实展示',await bodyHas('Agent Host'));
    const cfgHref=await evaluate('document.querySelector('+
      '\'a[href*="/setup"]\')?.href??null');
    i.ok('项目配置入口指向真实 Setup 服务',Boolean(cfgHref&&
      cfgHref.startsWith(setupOrigin+'/setup')));
    await shot('settings-git-github');
    await click('a[href*="/setup"]','setup link from settings');
    await poll(async()=>(await rawUrl()).includes('/setup'),'setup roundtrip');
    i.ok('从设置页跳转真实 Setup 成功',true);
    await navigate(appUrl.split('#')[0]+
      '#/governance/metadata?projectId='+encodeURIComponent(PROJECT1)+
      '&versionId=main','metadata');
    await poll(()=>bodyHas('NOT_CONNECTED'),'metadata honest');
    i.ok('版本元数据未挂载时诚实 NOT_CONNECTED',await bodyHas('NOT_CONNECTED'));
    await shot('metadata-not-connected');
    const rest=[['workbench','inbox'],['workbench','approvals'],
      ['workbench','blockers'],['development','deployment'],
      ['governance','workflow'],['governance','domain-packages'],
      ['governance','experience'],['governance','audit'],
      ['settings','agent-providers'],['settings','tally-llm'],
      ['settings','ci'],['settings','host'],['settings','storage'],
      ['settings','security'],['settings','deployment-env'],
      ['settings','integrations']];
    for(const [group,view] of rest){
      await navigate(appUrl.split('#')[0]+'#/'+group+'/'+view+
        '?projectId='+encodeURIComponent(PROJECT1)+'&versionId=main',
        group+'-'+view);
      await sleep(350);
      const honest=await bodyHas('NOT_CONNECTED');
      const settingsLike=await bodyHas('GitHub 项目与连接');
      i.ok('页面 /'+group+'/'+view+' 渲染(诚实空态='+
        (honest?'是':settingsLike?'设置中心':'否')+')',honest||settingsLike);
      await shot('menu-'+group+'-'+view);
    }
    await navigate(appUrl.split('#')[0]+
      '#/workbench/overview?projectId='+encodeURIComponent(PROJECT1),
      'overview-for-search');
    await poll(()=>located('header input[role="searchbox"]'),'topbar search');
    await fill('header input[role="searchbox"]','验收探针','topbar search');
    await client.call('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',
      code:'Enter',windowsVirtualKeyCode:13});
    await client.call('Input.dispatchKeyEvent',{type:'keyUp',key:'Enter',
      code:'Enter',windowsVirtualKeyCode:13});
    await poll(async()=>(await rawUrl()).includes('/workbench/inbox'),
      'inbox focus route',10000);
    i.ok('顶栏搜索跳转工作台收件箱(带 focus 参数)',true,S(await rawUrl()));
    await shot('topbar-search-inbox');
  });
}catch(error){
  consoleEntries.push({kind:'driver.fatal',text:S(String(error?.stack??error)),
    ts:new Date().toISOString(),page:null});
  try{if(client)await shot('99-fatal');}catch{}
}finally{
  try{client?.close();}catch{}
  if(chromium)try{chromium.kill();}catch{}
  if(app)await app.close().catch(()=>{});
}
// ---------------------------------------------------------------------------
// Evidence output (redacted)
// ---------------------------------------------------------------------------
const finishedAt=new Date().toISOString();
const chromiumVersion=(()=>{try{
  const probe=spawnSync(browserPath,['--version'],{encoding:'utf8',
    timeout:10000});
  return (probe.stdout??'').trim()||'UNKNOWN';
}catch{return 'UNKNOWN';}})();
const totals={totalPages:pageResults.length,
  passed:pageResults.filter(page=>page.verdict==='PASS').length,
  failed:pageResults.filter(page=>page.verdict==='FAIL').length,
  blocked:pageResults.filter(page=>page.verdict==='BLOCKED').length};
const summary={label:'TALLY_V2_R157_WINDOWS_CHROMIUM_INSTALLED_APP_E2E',
  sourceSha:'9d0d695e8886e91255198e667abc47cfb1b30c30',
  platform:process.platform,node:process.version,
  chromiumPath:browserPath,chromiumVersion,startedAt,finishedAt,...totals,
  clickActions:clicks.length,httpRequestCount:net.length,
  consoleErrorCount:consoleEntries.filter(entry=>
    entry.kind!=='console.warning').length,
  mockCodexRuns:mockRuns.length,
  claims:['MODEL_MOCK','HOST_READMODELS_MOCK','GITHUB_HEAD_MOCK','VAULT_MOCK',
    'GITHUB_WRITE_MOCK','RELEASE_MOCK'],
  real:['CHROMIUM(installed)','REACT_BUILD','SETUP_AND_ALM_HTTP',
    'PAGE_NAV_CLICK_TYPE','HTTP_REQUESTS','LOCAL_GIT_COMMIT'],
  realCodex:false,releaseEligible:false,
  pageVerdicts:Object.fromEntries(pageResults.map(page=>[page.key,
    page.verdict]))};
await writeFile(join(evidence,'page-results.json'),
  JSON.stringify({summary,results:pageResults},null,2)+'\n');
await writeFile(join(evidence,'network.har'),
  JSON.stringify(netToHar(),null,1)+'\n');
await writeFile(join(evidence,'console.json'),
  JSON.stringify({summary:{errorCount:summary.consoleErrorCount,
    warnings:consoleEntries.filter(entry=>
      entry.kind==='console.warning').length},entries:consoleEntries},null,2)+
  '\n');
await writeFile(join(evidence,'click-trace.json'),
  JSON.stringify({summary:{total:clicks.length,
    byKind:clicks.reduce((acc,row)=>{acc[row.kind]=(acc[row.kind]??0)+1;
      return acc;},{})},actions:clicks},null,2)+'\n');
await writeFile(join(evidence,'mock-agent-git-evidence.json'),
  JSON.stringify({boundary:'AGENT_PROVIDER_MOCK · 本地 Git 候选提交为真实 git 操作',
    runs:gitEvidence,verification:gitEvidence.map(entry=>({
      repository:entry.repository,parentMatchesSourceSha:
      entry.parent===entry.sourceSha,
      diffExactlyOneFile:entry.diffFile==='MOCK_CODEX_CANDIDATE.md',
      candidateCommit:entry.commit,candidateTree:entry.tree}))},null,2)+'\n');
console.log('R157_SUMMARY='+JSON.stringify(summary));
console.log('R157_EVIDENCE_DIR='+evidence);
for(const result of pageResults)
  console.log('R157_PAGE='+result.key+':'+result.verdict+
    (result.error?' ERROR='+result.error.slice(0,220):''));
if(totals.failed>0)process.exitCode=1;
