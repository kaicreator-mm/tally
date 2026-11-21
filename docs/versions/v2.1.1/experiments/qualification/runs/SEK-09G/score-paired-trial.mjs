#!/usr/bin/env node
// SEK-09G: verify an independently graded, frozen blind packet and unblind 7 D/K pairs.
// Node >=24, zero npm dependencies. This NEVER supplies or invents any ratings.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import os from 'node:os';
const sha = v => crypto.createHash('sha256').update(v).digest('hex');
const load = p => JSON.parse(fs.readFileSync(p,'utf8'));
const ensure = (ok,msg) => {if(!ok)throw Error(msg);};
const score = n => Number.isInteger(n) && n>=0 && n<=2;
function compare(runDir,filledPath,outPath){
  const run=load(path.join(runDir,'trial-manifest.json'));
  ensure(run.status==='ANSWERS_FROZEN_SCORING_NOT_RUN','Trial run not fully frozen');
  ensure(run.trials.length===14,'Not 14 trials');
  const questions=load(path.join(runDir,'frozen-questions.json')).questions;
  const blind=load(path.join(runDir,'blind-TO-INDEPENDENT-SCORER.json'));
  const key=load(path.join(runDir,'blind-key-WITHHOLD-FROM-SCORER.json')).mapping;
  const sourceContext=load(path.join(runDir,'source-context-TO-INDEPENDENT-SCORER.json'));
  const graded=load(filledPath);
  ensure(blind.status==='NOT_SCORED','Blind original was altered');
  ensure(sourceContext.status==='FROZEN_PINNED_SOURCE_CONTEXT','Pinned source context was altered');
  ensure(questions.length===7 && blind.items.length===14 && key.length===14 && sourceContext.items.length===7,'Frozen input count mismatch');
  ensure(typeof graded.reviewer_ref==='string' && graded.reviewer_ref.trim() && graded.independent_from_builder===true,'Independent grader attestation required');
  ensure(['HUMAN_REVIEWER','INDEPENDENT_MODEL_GRADER'].includes(graded.reviewer_kind),'Reviewer kind must distinguish human from automated');
  ensure(typeof graded.evaluated_at_utc==='string' && !Number.isNaN(Date.parse(graded.evaluated_at_utc)),'Genuine scoring UTC time required');
  ensure(Array.isArray(graded.ratings) && graded.ratings.length===14,'Exactly 14 blind ratings required');
  const byId=new Map(graded.ratings.map(x=>[x.blind_id,x]));
  const blindById=new Map(blind.items.map(x=>[x.blind_id,x]));
  const mapById=new Map(key.map(x=>[x.blind_id,x]));
  ensure(byId.size===14 && blindById.size===14 && mapById.size===14,'Duplicate blind IDs');
  const trials=new Map(run.trials.map(x=>[x.qid+'-'+x.arm,x]));
  ensure(trials.size===14 && run.trials.every(t=>t.status==='SUCCESS'),'Duplicate or failed trial');
  const qmap=new Map(questions.map(x=>[x.id,x]));
  const contextById=new Map(sourceContext.items.map(x=>[x.qid,x]));
  ensure(qmap.size===7 && contextById.size===7 && [...qmap.keys()].every((q,i)=>q==='G5-0'+(i+1)),'Changed question IDs');
  const paired=[];
  for(const [bid,blindEntry] of blindById){
    const rating=byId.get(bid),mapping=mapById.get(bid);
    ensure(rating && mapping,'Unrated or unmapped blind ID: '+bid);
    ensure(mapping.arm==='D'||mapping.arm==='K','Invalid arm');
    const qid=bid.slice(0,5),t=trials.get(qid+'-'+mapping.arm),q=qmap.get(qid),context=contextById.get(qid);
    ensure(q && context && t && t.qid===qid,'Wrong mapping to trial: '+bid);
    ensure(context.question===q.question && context.criteria===q.criteria && context.hard_exclusions===q.traps,
           'Pinned grader source context mismatch: '+bid);
    ensure(mapping.arm==='K'?sha(JSON.stringify(context.retrieval))===t.retrieval_sha256:t.retrieval_sha256===null,
           'Retrieval digest/source context mismatch: '+bid);
    ensure(blindEntry.question===q.question && blindEntry.criteria===q.criteria &&
           blindEntry.answer===t.answer && blindEntry.hard_exclusions===q.traps,'Blind evidence was altered');
    const req=fs.readFileSync(path.join(runDir,qid+'-'+mapping.arm+'-request.json'));
    const res=fs.readFileSync(path.join(runDir,qid+'-'+mapping.arm+'-provider-response.json'));
    ensure(sha(req)===t.request_sha256 && sha(res)===t.response_sha256,'Request/response digest mismatch: '+bid);
    ensure(JSON.parse(res.toString()).choices?.[0]?.message?.content===t.answer,'Provider raw answer mismatch: '+bid);
    for(const field of ['correctness_0_2','scope_0_2','traceability_0_2']){
      ensure(score(rating[field]),'Missing/invalid '+field+' on '+bid);
    }
    ensure(Array.isArray(rating.unsupported_assertions),'Missing unsupported assertion annotations: '+bid);
    for(const u of rating.unsupported_assertions){
      ensure(u && typeof u.span==='string' && u.span.length>0 && t.answer.includes(u.span) &&
             typeof u.reason==='string' && u.reason.trim() &&
             typeof u.source_check==='string' && u.source_check.trim(),'Invalid source-checked unsupported assertion: '+bid);
    }
    ensure(rating.observed_effort_seconds==null||
           (typeof rating.observed_effort_seconds==='number'&&Number.isFinite(rating.observed_effort_seconds)&&rating.observed_effort_seconds>=0),
           'Review effort cannot be invented or negative: '+bid);
    const expected=q.criteria.split(';').map(s=>s.trim()).filter(Boolean);
    ensure(Array.isArray(rating.criteria_marks) && rating.criteria_marks.length===expected.length,'Criterion count mismatch: '+bid);
    let met=0,omitted=0,unverifiable=0;
    for(let i=0;i<expected.length;i++){
      const m=rating.criteria_marks[i];
      ensure(m.criterion===expected[i] && ['MET','OMITTED','UNVERIFIABLE'].includes(m.outcome),
             'Criterion must exactly match frozen prereg: '+bid);
      if(m.outcome==='MET')met++;else if(m.outcome==='OMITTED')omitted++;else unverifiable++;
    }
    paired.push({qid,arm:mapping.arm,blind_id:bid,correctness:rating.correctness_0_2,
      scope:rating.scope_0_2,traceability:rating.traceability_0_2,
      unsupported:rating.unsupported_assertions.length,met,omitted,unverifiable,
      omission_rate:met+omitted?omitted/(met+omitted):null,
      observed_effort_seconds:rating.observed_effort_seconds??null});
  }
  const pairs=[];
  for(const q of questions){
    const x=paired.filter(v=>v.qid===q.id),d=x.find(v=>v.arm==='D'),k=x.find(v=>v.arm==='K');
    ensure(x.length===2&&d&&k,'Incomplete paired comparison '+q.id);
    const delta={};
    for(const metric of ['correctness','scope','traceability','unsupported'])delta[metric+'_K_minus_D']=k[metric]-d[metric];
    delta.omission_rate_K_minus_D=d.omission_rate===null||k.omission_rate===null?null:k.omission_rate-d.omission_rate;
    pairs.push({qid:q.id,D:d,K:k,delta});
  }
  const avg=(xs,key)=>xs.reduce((sum,x)=>sum+x[key],0)/xs.length;
  const summary={paired_questions:7,paired_responses:14,
    direct:{correctness_mean:avg(paired.filter(x=>x.arm==='D'),'correctness'),
            scope_mean:avg(paired.filter(x=>x.arm==='D'),'scope'),
            traceability_mean:avg(paired.filter(x=>x.arm==='D'),'traceability'),
            unsupported_total:paired.filter(x=>x.arm==='D').reduce((s,x)=>s+x.unsupported,0)},
    corpus:{correctness_mean:avg(paired.filter(x=>x.arm==='K'),'correctness'),
            scope_mean:avg(paired.filter(x=>x.arm==='K'),'scope'),
            traceability_mean:avg(paired.filter(x=>x.arm==='K'),'traceability'),
            unsupported_total:paired.filter(x=>x.arm==='K').reduce((s,x)=>s+x.unsupported,0)},
    observed_effort:paired.every(x=>x.observed_effort_seconds!==null)?{status:'MEASURED',direct_mean_seconds:avg(paired.filter(x=>x.arm==='D'),'observed_effort_seconds'),corpus_mean_seconds:avg(paired.filter(x=>x.arm==='K'),'observed_effort_seconds')}:{status:'PARTIAL_OR_NOT_MEASURED',measured_records:paired.filter(x=>x.observed_effort_seconds!==null).length,total_records:14},
    superiority:'NOT_CLAIMED_EXPLORATORY_NO_PRECOMMITTED_PASS_THRESHOLD',
    G4:'NOT_MEASURED_BY_THIS_EXPERIMENT',
    G5:'EMPIRICAL_RESULTS_REQUIRE_SEPARATE_GATE_REVIEW'};
  ensure(!fs.existsSync(outPath),'Do not overwrite a previously reported result');
  const result={status:'ACTUAL_BLIND_GRADING_VERIFIED_NO_AUTO_PASS',reviewer_ref:graded.reviewer_ref,
    reviewer_kind:graded.reviewer_kind,evaluated_at_utc:graded.evaluated_at_utc,
    scoring_file_sha256:sha(fs.readFileSync(filledPath)),run_manifest_sha256:sha(fs.readFileSync(path.join(runDir,'trial-manifest.json'))),
    summary,pairs};
  fs.writeFileSync(outPath,JSON.stringify(result,null,2)+'\n');
  return result;
}
const args=process.argv.slice(2);
const read=(name)=>{const i=args.indexOf(name);return i<0?null:args[i+1];};
if(args.includes('--make-template')){
  const runDir=read('--run-dir'),out=read('--out');
  if(!runDir||!out)throw Error('Usage: --make-template --run-dir PATH --out PATH');
  ensure(!fs.existsSync(out),'Refuse to overwrite an existing score file');
  const state=load(path.join(runDir,'trial-manifest.json'));
  ensure(state.status==='ANSWERS_FROZEN_SCORING_NOT_RUN' && state.trials.length===14 &&
         state.trials.every(t=>t.status==='SUCCESS'),'Cannot create scoring template without 14 real frozen answers');
  const blind=load(path.join(runDir,'blind-TO-INDEPENDENT-SCORER.json'));
  ensure(blind.items.length===14,'Missing blind packet');
  const template={reviewer_ref:null,reviewer_kind:null,independent_from_builder:null,
    evaluated_at_utc:null,ratings:blind.items.map(item=>({
      blind_id:item.blind_id,correctness_0_2:null,scope_0_2:null,traceability_0_2:null,
      criteria_marks:item.criteria.split(';').map(s=>({criterion:s.trim(),outcome:null})),
      unsupported_assertions:[],observed_effort_seconds:null
    }))};
  fs.writeFileSync(out,JSON.stringify(template,null,2)+'\n');
  console.log('BLIND_SCORING_TEMPLATE_CREATED=14 RATINGS_COMPLETED=0');
}else if(args.includes('--self-test')){
  const temp=fs.mkdtempSync(path.join(os.tmpdir(),'sek09g-score-')),runDir=path.join(temp,'run');
  fs.mkdirSync(runDir);
  const qs=Array.from({length:7},(_,i)=>({id:'G5-0'+(i+1),question:'q'+i,criteria:'one; two',traps:'no fabrications',retrieval:{claim:{statement:'source '+i}}}));
  const trials=[],blind=[],mapping=[],ratings=[];
  for(const q of qs){for(const arm of ['D','K']){
    const answer=q.id+arm+' mock',req='{"model":"MOCK"}',res=JSON.stringify({choices:[{message:{content:answer}}]});
    fs.writeFileSync(path.join(runDir,q.id+'-'+arm+'-request.json'),req);
    fs.writeFileSync(path.join(runDir,q.id+'-'+arm+'-provider-response.json'),res);
    trials.push({qid:q.id,arm,status:'SUCCESS',answer,request_sha256:sha(req),response_sha256:sha(res),retrieval_sha256:arm==='K'?sha(JSON.stringify(q.retrieval)):null});
    const bid=q.id+'-'+(arm==='D'?1:2);
    blind.push({blind_id:bid,question:q.question,criteria:q.criteria,hard_exclusions:q.traps,answer});
    mapping.push({blind_id:bid,arm});
    ratings.push({blind_id:bid,correctness_0_2:2,scope_0_2:2,traceability_0_2:2,
      unsupported_assertions:[],criteria_marks:[{criterion:'one',outcome:'MET'},{criterion:'two',outcome:'OMITTED'}],observed_effort_seconds:null});
  }}
  const dump=(name,obj)=>fs.writeFileSync(path.join(runDir,name),JSON.stringify(obj));
  dump('trial-manifest.json',{status:'ANSWERS_FROZEN_SCORING_NOT_RUN',trials});
  dump('frozen-questions.json',{questions:qs});
  dump('blind-TO-INDEPENDENT-SCORER.json',{status:'NOT_SCORED',items:blind});
  dump('blind-key-WITHHOLD-FROM-SCORER.json',{mapping});
  dump('source-context-TO-INDEPENDENT-SCORER.json',{status:'FROZEN_PINNED_SOURCE_CONTEXT',items:qs.map(q=>({qid:q.id,question:q.question,criteria:q.criteria,hard_exclusions:q.traps,retrieval:q.retrieval}))});
  const scored=path.join(temp,'scored.json');fs.writeFileSync(scored,JSON.stringify({reviewer_ref:'MOCK_TEST_ONLY',independent_from_builder:true,reviewer_kind:'INDEPENDENT_MODEL_GRADER',evaluated_at_utc:new Date().toISOString(),ratings}));
  const out=path.join(temp,'comparison.json'),result=compare(runDir,scored,out);
  ensure(result.pairs.length===7 && result.summary.paired_responses===14,'Self-test failed');
  let blocked=false;try{compare(runDir,scored,out)}catch(e){blocked=/overwrite/.test(e.message)}
  ensure(blocked,'Non-overwrite test failed');
  fs.rmSync(temp,{recursive:true,force:true});console.log('SCORER_SELF_TEST=PASS FIXTURE_ONLY ACTUAL_GRADES=0');
}else{
  const runDir=read('--run-dir'),filled=read('--scored-json'),out=read('--out');
  if(!runDir||!filled||!out)throw Error('Usage: node score-paired-trial.mjs --run-dir PATH --scored-json PATH --out PATH');
  const result=compare(runDir,filled,out);
  console.log('PAIRED_SCORE_VALIDATION=PASS',JSON.stringify(result.summary));
}
