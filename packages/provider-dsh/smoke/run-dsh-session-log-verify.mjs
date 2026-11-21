/**
 * Live verification of the DeepSeek session-log upload disable overlay.
 *
 * Method: point the CLI's model endpoint at a local capture server
 * (DEEPSEEK_BASE_URL, the documented $DEEPSEEK_BASE_URL fallback of the
 * llm-deepseek adapter; wire protocol is Anthropic Messages) and run the real
 * CLI twice — control (no overlay) vs session-log-disable overlay — then
 * diff the captured request bodies for `dsh_session_log`. A dummy key is
 * used: the capture server never validates credentials, so no real API
 * budget is consumed and no real credential leaves the machine.
 */
import {spawn} from 'node:child_process';
import {createServer} from 'node:http';
import {mkdtemp,rm,writeFile,readFile,appendFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';

const BIN=process.env.TALLY_DSH_BIN;
const NODE=process.env.TALLY_DSH_NODE??process.execPath;
if(!BIN)throw new Error('TALLY_DSH_BIN_REQUIRED');
const DUMMY_KEY='sk-local-capture-dummy-key-0123456789';
const INSTRUCTION='Reply with exactly SESSION_LOG_PROBE_OK and nothing else.';
const OVERLAY=[
  '- id: session-log-deepseek',
  "  name: '@deepseek-ai/dsh-session-log-deepseek'",
  '  config:',
  '    enabled: false'].join('\n')+'\n';

const capture=join(tmpdir(),'dsh-slog-capture-'+Date.now()+'.jsonl');
const requests=[];
const server=createServer((req,res)=>{
  let body='';
  req.on('data',chunk=>{body+=chunk;});
  req.on('end',async()=>{
    const entry={method:req.method,url:req.url,body};
    requests.push(entry);
    await appendFile(capture,JSON.stringify(entry)+'\n','utf8').catch(()=>{});
    if(req.method==='POST'&&req.url.endsWith('/messages')){
      let parsed={};
      try{parsed=JSON.parse(body);}catch{}
      if(parsed.stream){
        res.writeHead(200,{'content-type':'text/event-stream'});
        const events=[
          ['message_start',{type:'message_start',message:{id:'msg_mock',type:'message',
            role:'assistant',model:'deepseek-flash',content:[],stop_reason:null,
            usage:{input_tokens:3,output_tokens:1}}}],
          ['content_block_start',{type:'content_block_start',index:0,
            content_block:{type:'text',text:''}}],
          ['content_block_delta',{type:'content_block_delta',index:0,
            delta:{type:'text_delta',text:'SESSION_LOG_PROBE_OK'}}],
          ['content_block_stop',{type:'content_block_stop',index:0}],
          ['message_delta',{type:'message_delta',
            delta:{stop_reason:'end_turn',stop_sequence:null},usage:{output_tokens:6}}],
          ['message_stop',{type:'message_stop'}]];
        for(const [name,data] of events)
          res.write(`event: ${name}\ndata: ${JSON.stringify(data)}\n\n`);
        res.end();
      }else{
        res.writeHead(200,{'content-type':'application/json'});
        res.end(JSON.stringify({id:'msg_mock',type:'message',role:'assistant',
          model:'deepseek-flash',content:[{type:'text',text:'SESSION_LOG_PROBE_OK'}],
          stop_reason:'end_turn',usage:{input_tokens:3,output_tokens:6}}));
      }
      return;
    }
    if(req.url.includes('models')){
      res.writeHead(200,{'content-type':'application/json'});
      res.end(JSON.stringify({data:[{id:'deepseek-chat'},{id:'deepseek-flash'}]}));
      return;
    }
    res.writeHead(404);res.end('{}');
  });
});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
const port=server.address().port;

function runDsh({overlay}){
  return new Promise(async resolve=>{
    const home=await mkdtemp(join(tmpdir(),'dsh-slog-home-'));
    const work=await mkdtemp(join(tmpdir(),'dsh-slog-work-'));
    const args=[BIN,'--profile','headless',...(overlay?['--patch',join(home,'tally.patch.yml')]:[]),
      INSTRUCTION];
    if(overlay)await writeFile(join(home,'tally.patch.yml'),OVERLAY,'utf8');
    const child=spawn(NODE,args,{cwd:work,windowsHide:true,
      stdio:['ignore','pipe','pipe'],env:{
        DSH_HOME:home,DSH_TELEMETRY_MODE:'DISABLED',DSH_PERMISSION_MODE:'read-only',
        DEEPSEEK_API_KEY:DUMMY_KEY,DEEPSEEK_BASE_URL:`http://127.0.0.1:${port}`,
        NO_PROXY:'127.0.0.1,localhost',no_proxy:'127.0.0.1,localhost',
        SystemRoot:process.env.SystemRoot,SystemDrive:process.env.SystemDrive,
        ComSpec:process.env.ComSpec,PATH:process.env.PATH,TEMP:process.env.TEMP,TMP:process.env.TMP,
        USERPROFILE:process.env.USERPROFILE,HOME:process.env.HOME,
        APPDATA:process.env.APPDATA,LOCALAPPDATA:process.env.LOCALAPPDATA}});
    let stdout='',stderr='';
    child.stdout.on('data',c=>{stdout+=c;});
    child.stderr.on('data',c=>{stderr+=c;});
    const timer=setTimeout(()=>{try{child.kill();}catch{}
      resolve({home,work,exitCode:null,timedOut:true,stdout,stderr});},90000);
    timer.unref?.();
    child.once('close',code=>{clearTimeout(timer);
      resolve({home,work,exitCode:code,timedOut:false,stdout,stderr});});
    child.once('error',error=>{clearTimeout(timer);
      resolve({home,work,exitCode:null,spawnError:String(error?.code??error),stdout,stderr});});
  });
}

const controlStart=requests.length;
const control=await runDsh({overlay:false});
const controlRequests=requests.slice(controlStart);
const overlayStart=requests.length;
const overlayRun=await runDsh({overlay:true});
const overlayRequests=requests.slice(overlayStart);
server.close();
for(const run of [control,overlayRun]){
  await rm(run.home,{recursive:true,force:true}).catch(()=>{});
  await rm(run.work,{recursive:true,force:true}).catch(()=>{});
}
const slogIn=req=>{
  try{const parsed=JSON.parse(req.body);
    return Object.hasOwn(parsed,'dsh_session_log')?{present:true,
      bytes:JSON.stringify(parsed.dsh_session_log).length}:null;}
  catch{return req.body.includes('dsh_session_log')?{present:true,bytes:-1}:null;}
  };
const summarize=run=>({exitCode:run.exitCode,timedOut:run.timedOut??false,
  spawnError:run.spawnError??false,tokenMatched:run.stdout.includes('SESSION_LOG_PROBE_OK'),
  stderrTail:run.stderr.split(/\r?\n/).filter(x=>x.trim()).slice(-2).join(' | ').slice(0,300)});
const controlMsgs=controlRequests.filter(r=>r.url.endsWith('/messages'));
const overlayMsgs=overlayRequests.filter(r=>r.url.endsWith('/messages'));
const receipt={
  SESSION_LOG_UPLOAD_VERIFY:{
    method:'local Anthropic-protocol capture server via DEEPSEEK_BASE_URL; dummy key; no real API calls',
    control:{...summarize(control),messagesRequests:controlMsgs.length,
      dshSessionLog:controlMsgs.map(slogIn)},
    overlayDisable:{...summarize(overlayRun),messagesRequests:overlayMsgs.length,
      dshSessionLog:overlayMsgs.map(slogIn)},
    captureFile:capture},};
receipt.SESSION_LOG_UPLOAD_VERIFY.verdict=
  controlMsgs.some(r=>slogIn(r)?.present)&&!overlayMsgs.some(r=>slogIn(r)?.present)?
    'UPLOAD_OFF_LIVE_VERIFIED':(overlayMsgs.length>0&&!overlayMsgs.some(r=>slogIn(r)?.present)?
    'OVERLAY_RUN_CLEAN_CONTROL_NOT_OBSERVABLE (headless minimal turn carries no pending log suffix)':
    'INCONCLUSIVE');
console.log(JSON.stringify(receipt,null,2));
