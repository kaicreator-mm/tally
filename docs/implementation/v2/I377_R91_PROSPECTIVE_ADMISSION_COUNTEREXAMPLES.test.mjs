/**
 * I377 R91 deterministic contract counterexamples (MODEL ONLY).
 * No selected Host, native GitHub writes, real CAS, accepted Claim, Task PASS or CI evidence.
 * Node >=24: node --test docs/implementation/v2/I377_R91_PROSPECTIVE_ADMISSION_COUNTEREXAMPLES.test.mjs
 */
import test from "node:test";
import assert from "node:assert/strict";
import {createHash} from "node:crypto";
const FSW="tally:v2:task-writeset:377";
const JOIN="TEST_ONLY:jointRoleBinding:377";
const OUTCOME="TEST_ONLY:taskOutcome:377";
const SOURCE="62f18fd87d5a709f42900e612dcebc02c5c4f365";
const PACK="TEST_ONLY:prospective-no-write-pack";
const EMPTY_SHA="4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945";
const sha=x=>createHash("sha256").update(x).digest("hex");
class FakeCAS {
  constructor(){this.rows=new Map();this.counts=new Map()}
  put(key,data,{loseAck=false}={}){
    this.counts.set(key,(this.counts.get(key)||0)+1);
    if(this.rows.has(key))throw Error("EXISTS");
    const row={...data,revision:0};this.rows.set(key,row);
    if(loseAck)throw Error("ACK_LOST");
    return row;
  }
  get(key){return this.rows.get(key)}
  count(key){return this.counts.get(key)||0}
}
class Model {
  constructor(){
    this.cas=new FakeCAS();this.currentSource=true;this.currentPack=true;
    this.currentFence=true;this.acceptedScope=true;this.ready=false;
    this.ghPatches=0;this.claimPosts=0;this.claims={};
    this.evidence=null;this.reviewDecision=null;this.currentCI=true;this.currentABI=true;
  }
  current(){return this.acceptedScope&&this.currentSource&&this.currentPack&&this.currentFence&&this.ready}
  toReady({blocked=false,aba=false,drift=false,ackLost=false}={}){
    if(!this.acceptedScope||!this.currentSource||!this.currentPack||!this.currentFence||blocked)return "BLOCKED";
    if(aba||drift)return "UNKNOWN"; // GH labels have no cross-store CAS, so cannot rule out ABA
    if(this.ready)return "BLOCKED";
    this.ghPatches++;this.ready=true;
    return ackLost?"UNKNOWN":"MODEL_READY_NOT_AUTHORIZED";
  }
  fsw({document="[]",ackLost=false}={}){
    if(!this.current()||document!=="[]"||sha(document)!==EMPTY_SHA)return "BLOCKED";
    try {
      this.cas.put(FSW,{schema:"tally/version-task-writeset/v1",document,
        digest:EMPTY_SHA,source:SOURCE,pack:PACK,sourceWritePaths:[]},{loseAck:ackLost});
      return "MODEL_FSW_NO_AUTHORITY";
    }catch(e){return e.message==="ACK_LOST"?"UNKNOWN":"BLOCKED"}
  }
  claim(role,actor,{accepted=true,hasNativeRef=true,ackLost=false}={}){
    if(!this.current()||!this.cas.get(FSW)||!["validator","reviewer"].includes(role))return "BLOCKED";
    this.claimPosts++;
    if(ackLost)return "UNKNOWN"; // never infer CAS Claim acceptance from POST
    this.claims[role]={actor,accepted,revoked:false,casRef:"TEST_ONLY:"+role+":cas",
      nativeRef:hasNativeRef?"TEST_ONLY:"+role+":native":null,
      key:"kaicreator-mm/tally#377:"+role+":__default__",generation:1};
    return accepted?"MODEL_ROLE_ACCEPTED":"BLOCKED";
  }
  rolesOK(){
    const v=this.claims.validator,r=this.claims.reviewer;
    return !!(v&&r&&v.accepted&&r.accepted&&!v.revoked&&!r.revoked&&v.actor!==r.actor&&
      v.casRef&&r.casRef&&v.nativeRef&&r.nativeRef&&v.key!==r.key&&v.generation===1&&r.generation===1);
  }
  join({ackLost=false}={}){
    const fsw=this.cas.get(FSW);
    if(!this.current()||!fsw||!this.rolesOK()||fsw.document!=="[]"||fsw.digest!==EMPTY_SHA)return "BLOCKED";
    try {
      this.cas.put(JOIN,{fswKey:FSW,fswRevision:fsw.revision,fswDigest:EMPTY_SHA,
        validator:this.claims.validator,reviewer:this.claims.reviewer,source:SOURCE,pack:PACK},{loseAck:ackLost});
      return "MODEL_JOIN_NO_AUTHORITY";
    }catch(e){return e.message==="ACK_LOST"?"UNKNOWN":"BLOCKED"}
  }
  joined(){
    const f=this.cas.get(FSW),j=this.cas.get(JOIN);
    return !!(this.current()&&f&&j&&this.rolesOK()&&f.document==="[]"&&
      f.digest===EMPTY_SHA&&f.sourceWritePaths.length===0&&
      j.fswKey===FSW&&j.fswRevision===f.revision&&j.fswDigest===f.digest&&
      j.source===SOURCE&&j.pack===PACK&&j.validator===this.claims.validator&&j.reviewer===this.claims.reviewer);
  }
  validate(){
    if(!this.joined()||!this.currentCI||!this.currentABI)return "BLOCKED";
    this.evidence={source:SOURCE,ci:"TEST_ONLY:CI",abi:"TEST_ONLY:ABI"};
    return "MODEL_VALIDATION";
  }
  review(decision="ACCEPT"){
    if(!this.joined()||!this.evidence||!["ACCEPT","REWORK","DEFER"].includes(decision))return "BLOCKED";
    this.reviewDecision={decision,actor:this.claims.reviewer.actor};return "MODEL_DECISION";
  }
  outcome({ackLost=false}={}){
    if(!this.joined()||!this.currentCI||!this.currentABI||!this.evidence||
       this.evidence.source!==SOURCE||this.reviewDecision?.decision!=="ACCEPT"||
       this.reviewDecision.actor!==this.claims.reviewer.actor)return {status:"BLOCKED",taskPass:false};
    try {this.cas.put(OUTCOME,{source:SOURCE,pack:PACK,decision:"ACCEPT"},{loseAck:ackLost});
      return {status:"MODEL_CANDIDATE_ONLY",taskPass:false};
    }catch(e){return {status:e.message==="ACK_LOST"?"UNKNOWN":"BLOCKED",taskPass:false}}
  }
}
const fswReady=()=>{const m=new Model();assert.equal(m.toReady(),"MODEL_READY_NOT_AUTHORIZED");
  assert.equal(m.fsw(),"MODEL_FSW_NO_AUTHORITY");return m};
const joined=()=>{const m=fswReady();m.claim("validator","actor-A");m.claim("reviewer","actor-B");
  assert.equal(m.join(),"MODEL_JOIN_NO_AUTHORITY");return m};
const completed=()=>{const m=joined();assert.equal(m.validate(),"MODEL_VALIDATION");
  assert.equal(m.review(),"MODEL_DECISION");return m};

test("POS-ROLE-ORDER-B: one FSW, separate validator/reviewer, one join",()=>{
  const m=joined();assert.equal(m.cas.count(FSW),1);assert.equal(m.cas.count(JOIN),1);
  assert.deepEqual(m.cas.get(FSW).sourceWritePaths,[]);
  assert.equal(m.cas.get(JOIN).reviewer.actor,"actor-B");
});
test("POS-QUALIFIED-OUTCOME: simulated candidate cannot claim Task PASS",()=>{
  const m=completed();assert.deepEqual(m.outcome(),{status:"MODEL_CANDIDATE_ONLY",taskPass:false});
});
test("NEG-PLANNED-NOT-CLAIMABLE",()=>{
  const m=new Model();assert.equal(m.fsw(),"BLOCKED");assert.equal(m.cas.count(FSW),0);
});
test("NEG-NATIVE-PREDECESSOR-OPEN",()=>{
  const m=new Model();assert.equal(m.toReady({blocked:true}),"BLOCKED");assert.equal(m.ghPatches,0);
});
test("NEG-NO-APPROVED-S0-PACK",()=>{
  const m=new Model();m.acceptedScope=false;assert.equal(m.toReady(),"BLOCKED");
});
test("NEG-LABEL-ABA",()=>{
  const m=new Model();assert.equal(m.toReady({aba:true}),"UNKNOWN");assert.equal(m.ghPatches,0);
});
test("NEG-LABEL-UPDATEDAT-DRIFT",()=>{
  const m=new Model();assert.equal(m.toReady({drift:true}),"UNKNOWN");assert.equal(m.ghPatches,0);
});
test("NEG-LABEL-ACK-LOST-NO-REPLAY",()=>{
  const m=new Model();assert.equal(m.toReady({ackLost:true}),"UNKNOWN");
  assert.equal(m.toReady(),"BLOCKED");assert.equal(m.ghPatches,1);
});
test("NEG-FSW-INVALID-EMPTY-BYTES-OLD-TWO-PATH-BUILDER",()=>{
  const m=new Model();m.toReady();
  for(const doc of ["[]\n","\uFEFF[]","[ ]","[\"src/read-retry.mjs\"]"])
    assert.equal(m.fsw({document:doc}),"BLOCKED");
  assert.equal(m.cas.count(FSW),0);
});
test("NEG-FSW-CREATE-ONLY-RACE",()=>{
  const m=fswReady();assert.equal(m.fsw(),"BLOCKED");
  assert.equal(m.cas.count(FSW),2);assert.equal(m.cas.rows.size,1);
});
test("NEG-FSW-ACK-LOST-NO-RESEED",()=>{
  const m=new Model();m.toReady();assert.equal(m.fsw({ackLost:true}),"UNKNOWN");
  assert.equal(m.cas.count(FSW),1);assert.equal(m.cas.get(FSW).digest,EMPTY_SHA);
});
test("NEG-JOIN-WITHOUT-REVIEWER-CLAIM",()=>{
  const m=fswReady();m.claim("validator","actor-A");assert.equal(m.join(),"BLOCKED");
  assert.equal(m.cas.count(JOIN),0);
});
test("NEG-JOIN-ROLE-ALIAS",()=>{
  const m=fswReady();m.claim("validator","actor-A");m.claim("reviewer","actor-A");
  assert.equal(m.join(),"BLOCKED");
});
test("NEG-JOIN-UNACCEPTED-CLAIM",()=>{
  const m=fswReady();m.claim("validator","actor-A",{accepted:false});m.claim("reviewer","actor-B");
  assert.equal(m.join(),"BLOCKED");
});
test("NEG-CLAIM-POST-ACK-LOST",()=>{
  const m=fswReady();assert.equal(m.claim("validator","actor-A",{ackLost:true}),"UNKNOWN");
  assert.equal(m.claimPosts,1);assert.equal(m.join(),"BLOCKED");
});
test("NEG-CLAIM-MISSING-NATIVE-REF",()=>{
  const m=fswReady();m.claim("validator","actor-A",{hasNativeRef:false});m.claim("reviewer","actor-B");
  assert.equal(m.join(),"BLOCKED");
});
test("NEG-JOIN-CREATE-ONLY-RACE",()=>{
  const m=joined();assert.equal(m.join(),"BLOCKED");
  assert.equal(m.cas.count(JOIN),2);assert.equal(m.cas.rows.size,2);
});
test("NEG-JOIN-ACK-LOST-NO-RETRY",()=>{
  const m=fswReady();m.claim("validator","actor-A");m.claim("reviewer","actor-B");
  assert.equal(m.join({ackLost:true}),"UNKNOWN");assert.equal(m.cas.count(JOIN),1);
  assert.ok(m.cas.get(JOIN));
});
test("NEG-ROLE-REVOKE-AFTER-JOIN",()=>{
  const m=joined();m.claims.reviewer.revoked=true;assert.equal(m.validate(),"BLOCKED");
});
test("NEG-STALE-SOURCE-OR-PACK-OR-FENCE",()=>{
  for(const p of ["currentSource","currentPack","currentFence"]){
    const m=joined();m[p]=false;assert.equal(m.validate(),"BLOCKED");
  }
});
test("NEG-NO-VALIDATOR-EVIDENCE",()=>{
  const m=joined();assert.equal(m.review(),"BLOCKED");assert.equal(m.outcome().status,"BLOCKED");
});
test("NEG-CLAIM-NOT-REVIEW-DECISION",()=>{
  const m=joined();m.validate();assert.equal(m.outcome().status,"BLOCKED");
});
test("NEG-REWORK-DECISION-NOT-PASS",()=>{
  const m=joined();m.validate();m.review("REWORK");assert.equal(m.outcome().status,"BLOCKED");
});
test("NEG-CI-TRANSPORT-ABI-DRIFT",()=>{
  for(const p of ["currentCI","currentABI"]){
    const m=completed();m[p]=false;assert.equal(m.outcome().status,"BLOCKED");
  }
});
test("NEG-OUTCOME-ACK-LOST",()=>{
  const m=completed();assert.deepEqual(m.outcome({ackLost:true}),{status:"UNKNOWN",taskPass:false});
  assert.equal(m.cas.count(OUTCOME),1);assert.ok(m.cas.get(OUTCOME));
});
test("NEG-OUTCOME-DUPLICATE-CREATE-ONLY",()=>{
  const m=completed();m.outcome();assert.equal(m.outcome().status,"BLOCKED");
  assert.equal(m.cas.count(OUTCOME),2);
});
