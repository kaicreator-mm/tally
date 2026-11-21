/** Selected installed K consumer. BUILD provenance is not a Claim/Agent grant. */
import {createHash} from 'node:crypto';
import {readFile,writeFile,stat} from 'node:fs/promises';
import {resolve,join,isAbsolute} from 'node:path';
const SHA=/^[a-f0-9]{40}$/;
const SHA256=/^[a-f0-9]{64}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const FILE=/^[A-Za-z0-9_.-]+(?:\/[A-Za-z0-9_.-]+)*$/;
const manifestFile='tally-prebuilt-consumer.json';
const runtimeFiles=['consumer-local-codex.mjs','builder-target.mjs','local-codex-host.mjs'];
const digest=bytes=>createHash('sha256').update(bytes).digest('hex');
const fail=reason=>{throw new TypeError(reason);};
function target(env){
  const repository=env.T06M_TARGET_REPOSITORY;
  const versionId=env.T06M_TARGET_VERSION_REF;
  const issueNumber=Number(env.T06M_TARGET_ISSUE_NUMBER);
  const sourceSha=env.TALLY_CLAIM_SOURCE_SHA;
  const candidatePath=env.T06M_TARGET_CANDIDATE_PATH;
  if(!REPO.test(repository??'')||
     !/^(?:main|version\/[A-Za-z0-9_.-]+)$/.test(versionId??'')||
     !Number.isSafeInteger(issueNumber)||issueNumber<1||
     !SHA.test(sourceSha??'')||!FILE.test(candidatePath??'')||
     candidatePath.includes('..'))
    fail('PREBUILT_SELECTED_TARGET_INVALID');
  return {repository,versionId,issueNumber,sourceSha,candidatePath};
}
/** BUILD only; writes a non-secret exact identity + sealed app digest. */
export async function writePrebuiltConsumerManifest({dir,env=process.env}={}){
  if(typeof dir!=='string'||!dir)
    fail('PREBUILT_CONSUMER_DIR_REQUIRED');
  const identity=target(env);
  const bytes=await readFile(join(dir,'sealed-app.json'));
  const runtimeDigests={};
  for(const filename of runtimeFiles)
    runtimeDigests[filename]=digest(await readFile(join(dir,filename)));
  const manifest=Object.freeze({schema:'tally:v2:selected-consumer:1',
    ...identity,sealedAppSha256:digest(bytes),runtimeDigests});
  await writeFile(join(dir,manifestFile),JSON.stringify(manifest,null,2)+'\n',
    {flag:'wx',mode:0o600});
  return manifest;
}
/** Host only. Return an exact validated script path, never execution authority. */
export async function readPrebuiltConsumer({
  directory,env=process.env
}={}){
  if(typeof directory!=='string'||!isAbsolute(directory)||directory.length>1024)
    fail('PREBUILT_CONSUMER_DIR_INVALID');
  const dir=resolve(directory);
  const m=JSON.parse(await readFile(join(dir,manifestFile),'utf8'));
  const identity=target(env);
  if(m?.schema!=='tally:v2:selected-consumer:1'||
     Object.keys(identity).some(k=>m[k]!==identity[k])||
     !SHA256.test(m.sealedAppSha256??''))
    fail('PREBUILT_CONSUMER_SCOPE_MISMATCH');
  const app=await readFile(join(dir,'sealed-app.json'));
  if(digest(app)!==m.sealedAppSha256)
    fail('PREBUILT_CONSUMER_INTEGRITY_MISMATCH');
  for(const filename of runtimeFiles){
    const expected=m.runtimeDigests?.[filename];
    if(!SHA256.test(expected??'')||
       digest(await readFile(join(dir,filename)))!==expected)
      fail('PREBUILT_CONSUMER_INTEGRITY_MISMATCH');
  }
  const script=join(dir,'consumer-local-codex.mjs');
  const metadata=await stat(script);
  if(!metadata.isFile())fail('PREBUILT_CONSUMER_SCRIPT_MISSING');
  return Object.freeze({script,dir,manifest:m});
}
