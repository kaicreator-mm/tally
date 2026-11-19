/**
 * Fake DeepSeek Harness CLI for contract tests. Spawned for real (binPath =
 * process.execPath, binArgsPrefix = [this file]) so argv/env/cwd/pipe/exit
 * behavior exercises the adapter's real spawn path. Behavior is selected by
 * argv flags (the host builds a clean allowlisted env, so env-based scenario
 * selection would never reach this process):
 *   --fake-version <v>          version line for `--version` (default 0.2.2-rc.5)
 *   --fake-scenario <name>      answer | empty | exit7 | missing-credential |
 *                               orphan | hang
 *   --fake-write <json>         relative paths to create in cwd (worktree)
 *   --fake-record <path>        append one JSONL observation line
 *   --fake-grandchild-pid-file <path>  (orphan scenario) write grandchild pid
 * `--version` and `--profile headless "<task>"` follow the observed 0.2.x CLI
 * surface; stdout is the final answer, exit 0 only on completed runs.
 */
import {spawn} from 'node:child_process';
import {appendFileSync,writeFileSync,mkdirSync} from 'node:fs';
import {dirname,join,resolve} from 'node:path';

const args=process.argv.slice(2);
const flag=(name)=>{
  const i=args.indexOf(name);
  return i>=0&&i+1<args.length?args[i+1]:undefined;
};
const scenario=flag('--fake-scenario')??'answer';
const version=flag('--fake-version')??'0.2.2-rc.5';
const recordPath=flag('--fake-record');
const instruction=args.length>0?args.at(-1):'';

if(recordPath){
  appendFileSync(recordPath,JSON.stringify({argv:args,cwd:process.cwd(),
    env:{DSH_HOME:process.env.DSH_HOME??null,
      DSH_TELEMETRY_MODE:process.env.DSH_TELEMETRY_MODE??null,
      DSH_PERMISSION_MODE:process.env.DSH_PERMISSION_MODE??null,
      DEEPSEEK_API_KEY_present:typeof process.env.DEEPSEEK_API_KEY==='string'}})+'\n','utf8');
}

if(args.includes('--version')){
  process.stdout.write(version+'\n');
  process.exit(0);
}

const writeFiles=()=>{
  const spec=flag('--fake-write');
  if(!spec)return;
  for(const rel of JSON.parse(spec)){
    const target=resolve(process.cwd(),rel);
    mkdirSync(dirname(target),{recursive:true});
    writeFileSync(target,`# Fake DSH candidate note\n\nAttempt instruction: ${instruction}\n`,'utf8');
  }
};

switch(scenario){
  case 'answer':
    writeFiles();
    process.stdout.write('FAKE_DSH_FINAL_ANSWER: task complete.\n');
    process.exit(0);
    break;
  case 'empty':
    writeFiles();
    process.exit(0);
    break;
  case 'exit7':
    process.stderr.write('dsh: simulated harness failure (exit 7)\n');
    process.exit(7);
    break;
  case 'missing-credential':
    process.stderr.write('dsh: MISSING_CREDENTIAL: set DEEPSEEK_API_KEY to use the official model API\n');
    process.exit(1);
    break;
  case 'orphan':{
    // Grandchild inherits stdout/stderr and outlives this process, holding
    // the host's pipes open exactly like real orphaned tool subprocesses.
    // Wait for the actual spawn (and a short settle) before exiting, or the
    // parent's process.exit can race the handle inheritance.
    const grandchild=spawn(process.execPath,['-e','setInterval(()=>{},60000)'],
      {stdio:['ignore','inherit','inherit']});
    const pidFile=flag('--fake-grandchild-pid-file');
    const exit=()=>{process.stdout.write('FAKE_DSH_FINAL_ANSWER: launched a long-lived child.\n');
      setTimeout(()=>process.exit(0),100);};
    grandchild.once('spawn',()=>{if(pidFile)writeFileSync(pidFile,String(grandchild.pid),'utf8');exit();});
    grandchild.once('error',exit);
    break;
  }
  case 'hang':
  default:
    setInterval(()=>{},60000);
    break;
}
