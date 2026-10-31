/** Host-local real candidate publication through the existing selected K/B/C. */
import assert from 'node:assert/strict';
import {spawn} from 'node:child_process';
import {readFile,writeFile,mkdir} from 'node:fs/promises';
import {readFileSync} from 'node:fs';
import {join,resolve} from 'node:path';
import {createHash} from 'node:crypto';

async function consumer(){
  const {load}=await import('@kaicreator/domain-harness-microkernel');
  const {openSqliteCasStore}=await import('@kaicreator/domain-harness-host-node');
  const sealed=JSON.parse(readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8'));
  const selected=sealed.modules.find(x=>x.packageId==='capability-tally-local-codex@v1.0.0');
  const providerPath=new URL(import.meta.resolve('@kaicreator/tally-local-codex-host-bundle'));
  const provider=readFileSync(providerPath);
  const digest=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selected.moduleSource)?.[1];
  assert.equal(createHash('sha256').update(provider).digest('hex'),digest,'PROVIDER_INTEGRITY_MISMATCH');
  const token=process.env.TALLY_GITHUB_READ_TOKEN,hostStore=process.env.TALLY_LOCAL_HOST_STORE_PATH;
  const repositoryPath=process.env.TALLY_LOCAL_REPOSITORY_PATH,sourceSha='d6b32929b481006de4e848eb772fe7dd96c2547c';
  assert.ok(token&&hostStore&&repositoryPath);
  const db=join(process.env.TALLY_PUBLICATION_EVIDENCE,'publication-kernel.sqlite');
  const deployment={deployedAppId:'deploy:tally-v2-t06p',tenantId:'tenant:tally-v2',appId:sealed.appId};
  const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
    authorize:(identity,target)=>identity?.tenantId===target.tenantId&&
      target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
  const condition=c=>c?.kind==='create'?{mustBeAbsent:true}:
    c?.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
    (()=>{throw Error('INVALID_K_CONDITION');})();
  const options=sqlite=>{let pending=Promise.resolve();const serial=fn=>{const result=pending.then(fn,fn);pending=result.catch(()=>{});return result;};
    return {store:{get:key=>serial(()=>sqlite.get(key)),put:(key,v,c)=>serial(()=>sqlite.put(key,v,condition(c))),
      remove:(key,c)=>serial(()=>sqlite.remove(key,condition(c)))},deploymentIdentity:deployment,
      deploymentRoot:process.cwd(),tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.now()};};
  const message=(issueNumber,attemptId)=>({intentType:'publishReviewedCandidate',
    instanceKey:`task-candidate-publication-${issueNumber}`,messageId:`message-t06p-${attemptId}`,
    input:{issueRef:`https://github.com/kaicreator-mm/tally/issues/${issueNumber}`,issueNumber,sourceSha,attemptId}});
  const messages=[message(197,'attempt:alm:j05:a:1'),message(198,'attempt:alm:j05:repair:1')];
  let sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:5000});
  let facade=await load(sealed,options(sqlite));
  const outputs=[];
  for(const input of messages){
    const receipt=await facade.send(input);
    assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1500));
    const output=receipt.admitted.effects[0].output;
    assert.equal(output.state,'PUBLISHED_VERIFIED',JSON.stringify(output));
    assert.equal(output.taskPass,false);assert.equal(output.releasePass,false);
    outputs.push(output);
  }
  facade.stop();await sqlite.close();
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:5000});facade=await load(sealed,options(sqlite));
  const replay=await facade.send(messages[0]);
  assert.equal(replay.idempotentReplay,true);assert.equal(replay.status,'admitted');
  facade.stop();await sqlite.close();
  const {Octokit}=await import('@octokit/rest');
  const octokit=new Octokit({auth:token,request:{timeout:15000}}),readbacks=[];
  for(const output of outputs){
    const [pr,ref]=await Promise.all([
      octokit.pulls.get({owner:'kaicreator-mm',repo:'tally',pull_number:output.pullNumber}),
      octokit.git.getRef({owner:'kaicreator-mm',repo:'tally',ref:`heads/${output.branch}`})]);
    assert.equal(pr.data.head.sha,output.candidateCommit);assert.equal(pr.data.base.ref,'version/v2');
    assert.equal(pr.data.state,'open');assert.equal(ref.data.object.sha,output.candidateCommit);
    readbacks.push({pullNumber:output.pullNumber,pullRef:output.pullRef,branch:output.branch,
      candidateCommit:output.candidateCommit,baseRef:pr.data.base.ref,state:pr.data.state});
  }
  const result={status:'PASS',sourceSha,reopened:true,idempotentReplay:replay.idempotentReplay,
    publications:outputs,readbacks,taskPass:false,releasePass:false};
  await writeFile(process.env.TALLY_PUBLICATION_RESULT,JSON.stringify(result,null,2)+'\n',{mode:0o600});
  console.log(JSON.stringify(result));
}

async function orchestrate(){
  const tally=process.env.TALLY,evidence=resolve(process.env.TALLY_PUBLICATION_EVIDENCE);
  await mkdir(evidence,{recursive:true});
  const original=await readFile(join(tally,'experiments/v2/selected-business-build/run-clean.sh'),'utf8');
  const anchor='  T06M_LOCAL="${T06M_LOCAL:-0}" \\';
  const consumerAnchor='      TALLY_GITHUB_READ_TOKEN="$native_token" node "$tmp/consumer/consumer-local-codex.mjs"';
  assert.equal(original.split(anchor).length-1,1,'producer env anchor');
  assert.equal(original.split(consumerAnchor).length-1,1,'consumer anchor');
  const modified=original.replace(anchor,'  T06P_PUBLISH="${T06P_PUBLISH:-0}" \\\n'+anchor)
    .replace(consumerAnchor,'      npm install --save-exact @octokit/rest@22.0.1\n'+
      '      cp "$TALLY/experiments/v2/candidate-publication/publish-real.mjs" "$tmp/consumer/publish-real.mjs"\n'+
      '      TALLY_GITHUB_READ_TOKEN="$native_token" node "$tmp/consumer/publish-real.mjs" --consumer');
  const script=join(evidence,'run-publication.sh');await writeFile(script,modified,{mode:0o700});
  const child=spawn('C:\\Program Files\\Git\\bin\\bash.exe',[script],{cwd:tally,env:{...process.env,
    T04B_NATIVE:'1',T06M_LOCAL:'1',T06N_CONCURRENT:'0',T06P_PUBLISH:'1',TALLY_LOCAL_REPLAY:'1'},windowsHide:true});
  child.stdout.on('data',x=>process.stdout.write(x));child.stderr.on('data',x=>process.stderr.write(x));
  const exit=await new Promise(resolveExit=>{child.once('error',()=>resolveExit(1));child.once('close',code=>resolveExit(code));});
  if(exit!==0)throw new Error('PUBLICATION_DRIVER_FAILED:'+exit);
}
if(process.argv.includes('--consumer'))await consumer();else await orchestrate();
