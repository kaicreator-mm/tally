import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {defaultAlmConfig,createJourneyController} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';

const sourceSha='d'.repeat(40);
const config=root=>defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/tally',githubToken:'secret-token',
  currentSourceSha:sourceSha,workRoot:root});
const realResult=kind=>({status:'PASS',realAgentEffect:true,
  result:kind==='start'?{state:'LOCAL_CANDIDATE_READY',candidateTree:'a'.repeat(40),candidateCommit:'b'.repeat(40)}:
    {rejoin:{state:'REVIEW_JOINED'},overlap:{overlapMs:123,verified:true},candidateA:{tree:'a'.repeat(40),commit:'b'.repeat(40)},
      candidateB:{tree:'c'.repeat(40),commit:'d'.repeat(40)}}});
const runner=kind=>({status:'PASS',exitCode:0,result:realResult(kind),evidenceDir:`C:/evidence/${kind}`});

test('real start and reviewer results are read from e2e result without inventing authority',async()=>{
  const root=await mkdtemp(join(tmpdir(),'alm-controller-'));
  try{
    const controller=createJourneyController(config(root),{run:runner});
    const started=await controller.start(sourceSha);
    assert.equal(started.status,'PASS');assert.equal(started.result.realAgentEffect,true);
    const reviewed=await controller.reviewRepair(sourceSha);
    assert.equal(reviewed.result.result.rejoin.state,'REVIEW_JOINED');
    const snapshot=controller.snapshot();
    const latest=snapshot.result;
    assert.equal(latest.result.result.rejoin.state,'REVIEW_JOINED');
    assert.equal(snapshot.authorized,false);assert.equal(snapshot.taskPass,false);assert.equal(snapshot.releasePass,false);
    assert.equal(snapshot.tasks[0].sourceRef,'https://github.com/kaicreator-mm/tally/issues/196');
    assert.equal(snapshot.tasks[0].agentAttemptState,'LOCAL_CANDIDATE_READY');
    assert.equal(snapshot.tasks[1].reviewState,'REVIEW_JOINED');
    assert.equal(JSON.stringify(snapshot).includes('secret-token'),false);
  }finally{await rm(root,{recursive:true,force:true});}
});

test('stale source and failed e2e never become UI PASS',async()=>{
  const root=await mkdtemp(join(tmpdir(),'alm-controller-deny-'));
  try{
    const controller=createJourneyController(config(root),{run:runner});
    const stale=await controller.start('0'.repeat(40));
    assert.equal(stale.status,'STALE_SOURCE');
    const failed=await controller.start(sourceSha);
    assert.equal(failed.status,'PASS');
    const failing=createJourneyController(config(root),{run:async()=>({status:'FAIL',exitCode:1,result:null,evidenceDir:'C:/evidence/fail'})});
    const result=await failing.start(sourceSha);
    assert.equal(result.status,'FAIL');assert.equal(failing.snapshot().result.status,'FAIL');
    assert.equal(failing.snapshot().tasks[0].agentAttemptState,'FAILED');
  }finally{await rm(root,{recursive:true,force:true});}
});

test('same-origin HTTP actions enforce CSRF exact SHA and real result readback',async()=>{
  const root=await mkdtemp(join(tmpdir(),'alm-server-'));
  const app=createAlmServer(config(root),{run:runner});
  const {origin,actionKey}=await app.listen();
  try{
    const page=await fetch(origin+'/journey-view.html').then(r=>r.text());
    assert.match(page,/<script defer src="\.\/journey-bootstrap\.mjs"><\/script>/);
    assert.doesNotMatch(page,/<script(?:\s+type="module")?\s*>/);
    const client=await fetch(origin+'/journey-client.mjs').then(r=>r.text());
    const bootstrap=await fetch(origin+'/journey-bootstrap.mjs').then(r=>r.text());
    for(const source of [client,bootstrap]){
      const syntax=spawnSync(process.execPath,['--input-type=module','--check'],{input:source,encoding:'utf8'});
      assert.equal(syntax.status,0,syntax.stderr);
    }
    const snapshot=await fetch(origin+'/api/snapshot').then(r=>r.json());
    assert.equal(snapshot.actions.length,2);
    const denied=await fetch(origin+'/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':'wrong',origin},
      body:JSON.stringify({kind:'start',sourceSha})});
    assert.equal(denied.status,403);
    const stale=await fetch(origin+'/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':actionKey,origin},
      body:JSON.stringify({kind:'start',sourceSha:'0'.repeat(40)})});
    assert.equal(stale.status,409);
    const unknown=await fetch(origin+'/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':actionKey,origin},
      body:JSON.stringify({kind:'delete-everything',sourceSha})});
    assert.equal(unknown.status,400);
    const started=await fetch(origin+'/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':actionKey,origin},
      body:JSON.stringify({kind:'start',sourceSha})});
    assert.equal(started.status,200);
    const result=await started.json();assert.equal(result.status,'PASS');assert.equal(result.result.realAgentEffect,true);
    const reviewed=await fetch(origin+'/api/actions',{method:'POST',headers:{'content-type':'application/json','x-tally-action-key':actionKey,origin},
      body:JSON.stringify({kind:'review-repair',sourceSha})});
    assert.equal(reviewed.status,200);
    const latest=await fetch(origin+'/api/snapshot').then(r=>r.json());
    assert.equal(latest.snapshot.result['result']['result']['rejoin']['state'],'REVIEW_JOINED');
  }finally{await app.close();await rm(root,{recursive:true,force:true});}
});

test('invalid current source is rejected before any process or UI action',()=>{
  assert.throws(()=>defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/tally',githubToken:'x',currentSourceSha:'bad',workRoot:'/tmp'})&&
    createJourneyController(defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/tally',githubToken:'x',currentSourceSha:'bad',workRoot:'/tmp'})),
    /CURRENT_SOURCE_SHA_REQUIRED/);
});
