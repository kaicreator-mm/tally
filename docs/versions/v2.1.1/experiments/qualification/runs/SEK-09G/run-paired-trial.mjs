#!/usr/bin/env node
// SEK-09G source-frozen 7x2 real local-model execution. Node >=24; no npm dependencies.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import http from 'node:http';
import { execFileSync } from 'node:child_process';
const PREREG = ['83a47d404311d2b1ac0bc06616ddd477bf13af56','docs/versions/v2.1.1/experiments/qualification/SEK-09E-UTILITY-PREREGISTRATION.md','5498f830ab7e832e97c9b6a879bf1c1d6d8fddcd'];
const GRAPH = ['9cbf7983c5b47c146bd934f39e10517c3a828b66','docs/versions/v2.1.1/knowledge/generated/graph.jsonld','2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00'];
const SYS = 'Answer accurately and concisely about the named source. State scope, limitations and uncertainty. Do not invent citations or imply uninspected standards were inspected.';
const hash = v => crypto.createHash('sha256').update(v).digest('hex');
const iso = () => new Date().toISOString();
function show(repo, triple) {
  const b = execFileSync('git',['-C',repo,'show',triple[0]+':'+triple[1]],{maxBuffer:16_000_000});
  const got=crypto.createHash('sha1').update('blob '+b.length+'\0').update(b).digest('hex');
  if(got!==triple[2])throw Error('FROZEN_BLOB_DRIFT: '+triple[1]+' '+got);
  return b;
}
function freeze(repo){
  const md=show(repo,PREREG).toString('utf8'),raw=show(repo,GRAPH);
  const nodes=new Map(JSON.parse(raw.toString('utf8'))['@graph'].map(n=>[n['@id'],n]));
  const qs=[];
  for(const line of md.split('\n')){
    if(!/^\| G5-\d{2} \|/.test(line))continue;
    const cell=line.trim().slice(1,-1).split('|').map(s=>s.trim());
    if(cell.length!==5)throw Error('Malformed prereg table');
    const [id,question,chain,criteria,traps]=cell;
    const ids=[...chain.matchAll(/\x60([A-Z][A-Z0-9-]+)\x60/g)].map(m=>'se:'+m[1]);
    if(ids.length!==4||ids.some(x=>!nodes.has(x)))throw Error('Missing ID chain '+id);
    const [cl,ev,loc,src]=ids.map(x=>nodes.get(x));
    const rel=cl['se:SUPPORTED_BY'];
    if(!(Array.isArray(rel)?rel:[rel]).some(x=>x?.['@id']===ids[1]))throw Error('Broken SUPPORTED_BY '+id);
    if(ev['se:source_locator_ref']?.['@id']!==ids[2]||loc['se:source_ref']?.['@id']!==ids[3])throw Error('Broken locator '+id);
    const retrieval={claim:{id:ids[0],statement:cl['se:statement'],scope:cl['se:scope_description'],limitations:cl['se:limitations']},evidence:{id:ids[1],statement:ev['se:statement'],limitations:ev['se:limitations']},locator:{id:ids[2],section:loc['se:locator']},source:{id:ids[3],title:src['se:title'],limitations:src['se:authority_limitations']}};
    qs.push({id,question,criteria,traps,ids,question_sha256:hash(question),retrieval});
  }
  if(qs.length!==7||qs.some((q,i)=>q.id!=='G5-0'+(i+1)))throw Error('Seven frozen questions missing');
  return {qs,meta:{prereg_revision:PREREG[0],prereg_blob:PREREG[2],graph_revision:GRAPH[0],graph_blob:GRAPH[2],graph_sha256:hash(raw)}};
}
function save(file,data){fs.mkdirSync(path.dirname(file),{recursive:true});const tmp=file+'.partial';fs.writeFileSync(tmp,JSON.stringify(data,null,2)+'\n');fs.renameSync(tmp,file);}
async function run(qs,meta,options){
  const endpoint=new URL(options.endpoint);
  if(endpoint.protocol!=='http:'||!['localhost','127.0.0.1','[::1]'].includes(endpoint.hostname)||!endpoint.pathname.endsWith('/v1/chat/completions'))throw Error('LOCAL_ENDPOINT_ONLY (no paid API)');
  if(fs.existsSync(options.out)&&fs.readdirSync(options.out).length)throw Error('Output exists: cannot overwrite frozen answers');
  fs.mkdirSync(options.out,{recursive:true});
  const jobs=qs.flatMap(q=>['D','K'].map(arm=>({q,arm})));
  let s=72831;for(let i=jobs.length-1;i>0;i--){s=(Math.imul(s,1664525)+1013904223)>>>0;const j=s%(i+1);[jobs[i],jobs[j]]=[jobs[j],jobs[i]];}
  const state={status:'IN_PROGRESS',started_utc:iso(),model_requested:options.model,provider_endpoint:options.endpoint,temperature:0,frozen:meta,trials:[]};
  save(path.join(options.out,'trial-manifest.json'),state);
  save(path.join(options.out,'frozen-questions.json'),{questions:qs.map(({retrieval,...q})=>q)});
  for(const {q,arm} of jobs){
    const user=arm==='K'?q.question+'\n\nUse these exact pinned knowledge graph excerpts and say what they do not prove:\n'+JSON.stringify(q.retrieval):q.question;
    const req={model:options.model,temperature:0,max_tokens:512,stream:false,messages:[{role:'system',content:SYS},{role:'user',content:user}]};
    const reqRaw=JSON.stringify(req);
    const e={qid:q.id,arm,status:'IN_PROGRESS',request_sha256:hash(reqRaw),retrieval_sha256:arm==='K'?hash(JSON.stringify(q.retrieval)):null,started_utc:iso()};
    state.trials.push(e);save(path.join(options.out,'trial-manifest.json'),state);
    // Persist exact bytes under the same ID as the manifest hash, not a re-serialized proxy.
    fs.writeFileSync(path.join(options.out,q.id+'-'+arm+'-request.json'),reqRaw);
    try{
      const started=performance.now();
      const response=await fetch(endpoint,{method:'POST',headers:{'content-type':'application/json'},body:reqRaw,redirect:'error',signal:AbortSignal.timeout(options.timeout)});
      if(!response.ok)throw Error('HTTP_'+response.status);
      const raw=await response.text();
      const res=JSON.parse(raw),answer=res.choices?.[0]?.message?.content;
      if(typeof answer!=='string'||!answer.trim())throw Error('EMPTY_ANSWER');
      fs.writeFileSync(path.join(options.out,q.id+'-'+arm+'-provider-response.json'),raw);
      Object.assign(e,{status:'SUCCESS',answer,model_returned:res.model??'MODEL_VERSION_UNKNOWN',response_sha256:hash(raw),provider_usage:res.usage??null,system_fingerprint:res.system_fingerprint??null,elapsed_ms:Math.round(performance.now()-started),finished_utc:iso()});
      save(path.join(options.out,q.id+'-'+arm+'.json'),e);
      console.log(q.id,arm,'SUCCESS');
    }catch(error){Object.assign(e,{status:'FAILED',error:String(error).slice(0,300),failed_utc:iso()});state.status='INCOMPLETE_PROVIDER_FAILURE';save(path.join(options.out,'trial-manifest.json'),state);console.log(q.id,arm,'FAILED',e.error);return false;}
    save(path.join(options.out,'trial-manifest.json'),state);
  }
  if(state.trials.length!==14||state.trials.some(e=>e.status!=='SUCCESS'))throw Error('NOT_ALL_14_CALLS');
  state.status='ANSWERS_FROZEN_SCORING_NOT_RUN';state.finished_utc=iso();save(path.join(options.out,'trial-manifest.json'),state);
  const blind=[],key=[];
  let blindSeed=72831;
  for(const q of qs){const pair=state.trials.filter(e=>e.qid===q.id);if(pair.length!==2)throw Error('Broken pair');blindSeed=(Math.imul(blindSeed,1664525)+1013904223)>>>0;if(blindSeed%2)pair.reverse();for(let i=0;i<2;i++){const e=pair[i],bid=q.id+'-'+(i+1);blind.push({blind_id:bid,question:q.question,answer:e.answer,criteria:q.criteria,hard_exclusions:q.traps,reviewer_ref:null,correctness_0_2:null,scope_0_2:null,traceability_0_2:null,unsupported_claims:null,omission_count:null,observed_effort_seconds:null});key.push({blind_id:bid,arm:e.arm});}}
  save(path.join(options.out,'blind-TO-INDEPENDENT-SCORER.json'),{status:'NOT_SCORED',items:blind});
  save(path.join(options.out,'blind-key-WITHHOLD-FROM-SCORER.json'),{mapping:key});
  save(path.join(options.out,'source-context-TO-INDEPENDENT-SCORER.json'),{status:'FROZEN_PINNED_SOURCE_CONTEXT',items:qs.map(q=>({qid:q.id,question:q.question,criteria:q.criteria,hard_exclusions:q.traps,retrieval:q.retrieval}))});
  console.log('REAL_ANSWERS=14 INDEPENDENT_SCORING=NOT_RUN G5=NOT_YET_QUALIFIED');return true;
}
async function test(){
  let count=0;const requests=[];const server=http.createServer(async(req,res)=>{const parts=[];for await(const chunk of req){parts.push(chunk)}requests.push(JSON.parse(Buffer.concat(parts).toString('utf8')));count++;res.setHeader('content-type','application/json');res.end(JSON.stringify({model:'MOCK_TEST_ONLY',choices:[{message:{content:'MOCK_RESPONSE_'+count}}]}));});
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  try{const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'sek09g-'));const qs=Array.from({length:7},(_,i)=>({id:'G5-0'+(i+1),question:'TEST_ONLY_'+i,criteria:'test',traps:'test',retrieval:{claim:{statement:'mock'}}}));
    const out=path.join(tmp,'out'),ok=await run(qs,{mock_only:true},{endpoint:'http://127.0.0.1:'+server.address().port+'/v1/chat/completions',model:'MOCK_ONLY',out,timeout:3000});
    const st=JSON.parse(fs.readFileSync(path.join(out,'trial-manifest.json')));
    const blind=JSON.parse(fs.readFileSync(path.join(out,'blind-TO-INDEPENDENT-SCORER.json')));
    const context=JSON.parse(fs.readFileSync(path.join(out,'source-context-TO-INDEPENDENT-SCORER.json')));
    if(!ok||count!==14||st.trials.length!==14||blind.items.length!==14||blind.items.some(x=>x.correctness_0_2!==null))throw Error('SELF_TEST_FAIL');
    if(context.items.length!==7||context.items.some((x,i)=>x.qid!=='G5-0'+(i+1)||!x.retrieval||JSON.stringify(x).includes('"arm"')))throw Error('SOURCE_CONTEXT_FAIL');
    const messages=requests.map(r=>r.messages[1].content);
    if(messages.filter(s=>s.includes('Use these exact pinned knowledge graph excerpts')).length!==7||messages.filter(s=>!s.includes('Use these exact pinned knowledge graph excerpts')).length!==7||messages.some(s=>s.includes('hard_exclusions')||s.includes('criteria:')))throw Error('ARM_ISOLATION_OR_CRITERIA_LEAK');
    fs.rmSync(tmp,{recursive:true,force:true});console.log('SELF_TEST=PASS MOCK_ONLY REAL_MODEL_CALLS=0');
  }finally{server.close();}
}
const args=process.argv.slice(2);const get=(key,def)=>{const i=args.indexOf(key);return i<0?def:args[i+1];};
if(args.includes('--self-test'))await test();
else{const repo=get('--repo',process.cwd()),{qs,meta}=freeze(repo);console.log('FROZEN_INPUTS=PASS',JSON.stringify(meta));
  if(!args.includes('--preflight')){const out=get('--outdir',null);if(!out)throw Error('--outdir required (new/empty output directory)');const ok=await run(qs,meta,{out,endpoint:get('--endpoint','http://127.0.0.1:11434/v1/chat/completions'),model:get('--model','qwen2.5:7b'),timeout:Number(get('--timeout-ms',240000))});if(!ok)process.exitCode=2;}}
