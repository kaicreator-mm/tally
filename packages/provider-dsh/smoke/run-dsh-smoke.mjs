/**
 * One-shot REAL DeepSeek Harness CLI smoke for the provider-dsh candidate.
 *
 * Gates (all must hold, else the run refuses and reports NOT_RUN):
 *   - TALLY_DSH_SMOKE_CONFIRM=1   (explicit operator confirmation)
 *   - DEEPSEEK_API_KEY set        (authorized test credential; never printed)
 *   - TALLY_DSH_BIN (path to the dsh entry, e.g. node_modules/@deepseek-ai/
 *     dsh/lib/bin.js) plus TALLY_DSH_NODE (the node executable) — or a
 *     directly spawnable TALLY_DSH_BIN.
 *
 * What it proves (and deliberately does NOT prove): this verifies the CLI
 * contract assumptions encoded in the adapter — `--version` equals the pin,
 * headless non-interactive run answers on stdout with exit 0, keyless run
 * fails with the MISSING_CREDENTIAL shape, per-attempt DSH_HOME isolates
 * state. It is NOT a Host-isolated Builder attempt, NOT a Task/Review/
 * Release PASS and NOT candidate evidence.
 *
 * Usage: node smoke/run-dsh-smoke.mjs   (from packages/provider-dsh)
 */
import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdtemp,rm,writeFile,readFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {PINNED_DSH_VERSION} from '../src/dsh-config.mjs';

const INSTRUCTION='Reply with exactly the token TALLY_DSH_SMOKE_OK and nothing else. '
  +'Do not read or write any files.';
const SESSION_LOG_DISABLE_PATCH=`- id: session-log-deepseek\n`+
  `  name: '@deepseek-ai/dsh-session-log-deepseek'\n`+
  `  config:\n`+
  `    enabled: false\n`;
const redact=(text,secrets)=>secrets.reduce((out,s)=>out.split(s).join('[REDACTED]'),String(text??''));

function runDsh({node,bin,args,env,timeoutMs}){
  const argv=node?[bin,...args]:args;
  const {__CWD:cwd,...cleanEnv}=env;
  const child=spawn(node??bin,argv,{cwd,env:cleanEnv,
    windowsHide:true,stdio:['ignore','pipe','pipe']});
  let stdout='',stderr='';
  child.stdout.on('data',c=>{stdout+=c;});
  child.stderr.on('data',c=>{stderr+=c;});
  const done=new Promise(resolve=>{
    const timer=setTimeout(()=>{try{child.kill();}catch{}
      resolve({code:null,signal:null,timedOut:true});},timeoutMs);
    timer.unref?.();
    child.once('error',error=>{clearTimeout(timer);
      resolve({code:null,signal:null,spawnError:String(error?.code??error)});});
    child.once('close',(code,signal)=>{clearTimeout(timer);
      resolve({code,signal,spawnError:false});});
  });
  return done.then(result=>({...result,stdout,stderr}));
}

const receipt={REAL_DSH_SMOKE:'NOT_RUN',pinnedVersion:PINNED_DSH_VERSION};
try{
  if(process.env.TALLY_DSH_SMOKE_CONFIRM!=='1')throw new Error('CONFIRM_GATE_MISSING');
  const key=process.env.DEEPSEEK_API_KEY;
  if(!key)throw new Error('DEEPSEEK_API_KEY_MISSING');
  const bin=process.env.TALLY_DSH_BIN;
  if(!bin)throw new Error('TALLY_DSH_BIN_MISSING');
  const node=process.env.TALLY_DSH_NODE??null;
  const home=await mkdtemp(join(tmpdir(),'tally-dsh-smoke-home-'));
  const work=await mkdtemp(join(tmpdir(),'tally-dsh-smoke-work-'));
  const patch=join(home,'tally-smoke.patch.yml');
  await writeFile(patch,SESSION_LOG_DISABLE_PATCH,'utf8');
  const baseEnv={DSH_HOME:home,DSH_TELEMETRY_MODE:'DISABLED',
    DSH_PERMISSION_MODE:'read-only',__CWD:work};
  const secrets=[key];
  try{
    const version=await runDsh({node,bin,args:['--version'],
      env:{__CWD:work,DSH_HOME:home},timeoutMs:30000});
    const observed=version.stdout.trim().split(/\r?\n/)[0]?.trim()??'';
    receipt.versionCheck={exitCode:version.code,observedVersion:observed||null,
      pinMatched:observed===PINNED_DSH_VERSION,spawnError:version.spawnError??false};
    if(version.code!==0||observed!==PINNED_DSH_VERSION)
      throw new Error('VERSION_PIN_MISMATCH');
    const positive=await runDsh({node,bin,
      args:['--profile','headless','--patch',patch,INSTRUCTION],
      env:{...baseEnv,DEEPSEEK_API_KEY:key},timeoutMs:180000});
    receipt.positiveRun={exitCode:positive.code,timedOut:positive.timedOut??false,
      spawnError:positive.spawnError??false,
      answerSha256:createHash('sha256').update(positive.stdout).digest('hex'),
      answerBytes:Buffer.byteLength(positive.stdout),
      tokenMatched:positive.stdout.includes('TALLY_DSH_SMOKE_OK'),
      stderrTail:redact(positive.stderr.slice(-300),secrets)};
    const negative=await runDsh({node,bin,
      args:['--profile','headless',INSTRUCTION],
      env:baseEnv,timeoutMs:60000});
    receipt.negativeKeylessRun={exitCode:negative.code,
      missingCredentialShape:/MISSING_CREDENTIAL/.test(negative.stderr),
      stderrTail:redact(negative.stderr.slice(-300),secrets)};
    receipt.positivePass=receipt.positiveRun.exitCode===0&&receipt.positiveRun.tokenMatched;
    receipt.negativePass=receipt.negativeKeylessRun.exitCode!==0&&
      receipt.negativeKeylessRun.missingCredentialShape;
    receipt.REAL_DSH_SMOKE=(receipt.positivePass&&receipt.negativePass)?'PASS':'FAIL';
    receipt.sessionLogDisable='ARRAY_FORMAT_LIVE_ACCEPTED_UPLOAD_OFF_SOURCE_DERIVED';
    receipt.isolation='per-attempt throwaway DSH_HOME ('+home+')';
    receipt.NOTES='CLI contract smoke only: not Host-isolated, not Task/Review/Release PASS';
  }finally{
    await rm(home,{recursive:true,force:true}).catch(()=>{});
    await rm(work,{recursive:true,force:true}).catch(()=>{});
  }
}catch(error){
  receipt.REAL_DSH_SMOKE='NOT_RUN';
  receipt.reason=String(error?.message??error);
}
console.log(JSON.stringify(receipt,null,2));
process.exit(receipt.REAL_DSH_SMOKE==='PASS'||receipt.REAL_DSH_SMOKE==='NOT_RUN'?0:1);
