// Tally #398 R180: AI non-human cross-model review decision contract harness.
//
// Positive + adversarial verification of the GPT/GLM/Qwen three-family ledger
// sidecar against the exact reviewed graph blob. Mutation classes:
// positive / negative (duplicate family, forged Gemini identity, receipt
// forgery, lifecycle/ReviewDecision promotion, rights/Oracle/SAP/Effect
// inflation, p248 elevation, wrong source) / unknown (retained UNKNOWN and
// UNKNOWN authorities must not silently qualify) / stale (digest, record
// version, wrong-blob bindings all fail closed). This harness is native Node
// >=24, zero dependencies, and writes nothing.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {execFileSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import {
  AI_DECISION_PATH, AI_DECISION_SCHEMA_PATH,
  AI_DECISION_BLOB, AI_DECISION_SCHEMA_BLOB,
  ORIGINAL_GRAPH_BLOB, REVIEWED_GRAPH_BLOB,
  assertAICrossModelReviewDecisions,
} from './ai-cross-model-review-decision.mjs';
import {gitBlob} from './source-owner-disposition.mjs';

const LIB_PATH='docs/versions/v2.1.2/knowledge/tools/ai-cross-model-review-decision.mjs';
const REPO_ROOT=fileURLToPath(new URL('../../../../../',import.meta.url));
const git=(args)=>execFileSync('git',args,{cwd:REPO_ROOT,encoding:'utf8',stdio:['ignore','pipe','pipe']}).trim();
const lookup=(g,id)=>g['@graph'].find(x=>x['@id']===id);

export function verify(document,schema,frozen,candidate,frozenBytes,candidateBytes){
  return assertAICrossModelReviewDecisions(document,schema,frozen['@graph'],candidate['@graph'],frozenBytes,candidateBytes);
}

export function selfTest(document,schema,frozen,candidate,frozenBytes,candidateBytes){
  const result=verify(document,schema,frozen,candidate,frozenBytes,candidateBytes);
  const mutants=[
    // ---- forged / duplicate reviewer identity ----
    {kind:'negative',fn:x=>{x.d.receipts[1].modelFamily='GEMINI'}},                       // GLM impersonating Gemini
    {kind:'negative',fn:x=>{x.d.receipts[2].modelFamily='GLM'}},                          // duplicate family quorum
    {kind:'negative',fn:x=>{x.d.receipts.splice(2,1)}},                                   // quorum collapses to 2 receipts
    {kind:'negative',fn:x=>{x.d.receipts[0].reqVerdict='REJECT'}},                        // unanimity broken
    {kind:'negative',fn:x=>{x.d.receipts[0].findingsP0=1}},                               // P0=0 required
    {kind:'negative',fn:x=>{x.d.receipts[0].findingsP1=1}},                               // P1=0 required
    {kind:'negative',fn:x=>{x.d.receipts[1].modelFamilyDisclosure='SUBSTITUTE_LANE_DISCLOSED_MODEL_FAMILY_ACTUAL_GLM__COUNTED_AS_GEMINI'}},
    {kind:'negative',fn:x=>{x.d.receipts[2].blindFirstPass=false}},                       // blind first pass required
    {kind:'negative',fn:x=>{x.d.receipts[2].commentId='6976586467'}},                     // receipt/URL binding forgery
    {kind:'negative',fn:x=>{x.d.reviewContract.geminiFamilyVoteCount=1}},                 // Gemini lane stays NOT_RUN
    {kind:'negative',fn:x=>{x.d.reviewContract.distinctFamilies=['GPT','GLM']}},
    {kind:'negative',fn:x=>{x.d.reviewContract.integrationResult='REQ_AND_VV_3_OF_3_HUMAN_ORACLE_EQUIVALENT'}},
    // ---- lifecycle / authority promotion ----
    {kind:'negative',fn:x=>{x.d.authority.claimLifecycleEffect='ACCEPTED_BOTH_SELECTED_CLAIMS'}},
    {kind:'negative',fn:x=>{x.d.authority.reviewDecisionNodesInGraph=2}},
    {kind:'negative',fn:x=>{x.d.authority.professionalOracle='ATTESTED'}},
    {kind:'negative',fn:x=>{x.d.authority.humanAttestationSubstitute='AI_REVIEW_TREATED_AS_HUMAN_PROFESSIONAL_CERTIFICATION'}},
    {kind:'negative',fn:x=>{x.d.authority.publicationPermission='GRANTED'}},
    {kind:'negative',fn:x=>{x.d.authority.integrationDecisionRecord='https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6808543942'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:CLM-TVV-0001')['se:review_status']='ACCEPTED'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:CLM-TVV-0001')['se:lifecycle_status']='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.c['@graph'].push({'@id':'se:RD-FAKE','@type':'se:ReviewDecision','se:record_version':'0.1.0'})}},
    {kind:'negative',fn:x=>{x.c['@graph'].push({'@id':'se:AIM-FAKE','@type':'se:AiCrossModelReviewDecision','se:record_version':'0.1.0'})}},
    {kind:'negative',fn:x=>{x.d.claims.push(structuredClone(x.d.claims[0]))}},            // scope: exactly two claims
    {kind:'negative',fn:x=>{x.d.claims[0].aiReviewDecision='ACCEPTED_FULL'}},
    {kind:'negative',fn:x=>{x.d.claims[0].claimLifecycleAfterDecision='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[0].semanticOracle='ATTESTED'}},
    // ---- wrong source ----
    {kind:'negative',fn:x=>{x.d.sources.reviewedGraphBlob=x.d.sources.originalGraphBlob}},
    {kind:'negative',fn:x=>{x.cb=Buffer.concat([x.cb,Buffer.from(' ')])}},                // reviewed-bytes drift
    {kind:'negative',fn:x=>{x.fb=Buffer.concat([x.fb,Buffer.from(' ')])}},                // frozen-bytes drift
    {kind:'negative',fn:x=>{lookup(x.c,'se:SRC-CS2023')['se:stable_identifier']='DOI 10.9999/0000'}},
    // ---- permission inflation ----
    {kind:'negative',fn:x=>{x.d.qualification.formalSap='6/6'}},
    {kind:'negative',fn:x=>{x.d.qualification.formalGlobal='5/5'}},
    {kind:'negative',fn:x=>{x.d.qualification.effectAuthorized=true}},
    {kind:'negative',fn:x=>{x.d.qualification.releaseAuthorized=true}},
    {kind:'negative',fn:x=>{x.d.qualification.humanOracleAttestations=1}},
    {kind:'negative',fn:x=>{x.d.qualification.forgeAiDemoCellAuthorization='AUTHORIZED_BY_THIS_LEDGER_NOW'}},
    {kind:'negative',fn:x=>{x.d.qualification.originalR8='53/53'}},
    {kind:'negative',fn:x=>{x.d.sources.rightsScope='unrestricted redistribution and LLM training granted'}},
    // ---- p248 pending leg protection ----
    {kind:'negative',fn:x=>{lookup(x.c,'se:REL-TVV-0024')['se:lifecycle_status']='ACCEPTED'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:EVD-TVV-0016')['se:lifecycle_status']='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[1].pendingOutcomeLeg.rule='this ledger may retroactively accept the p248 leg'}},
    {kind:'negative',fn:x=>{x.d.claims[1].pendingOutcomeLeg.status='ACCEPTED_AND_ELEVATED'}},
    {kind:'negative',fn:x=>{x.d.claims[1].pendingOutcomeLeg.requiredLifecycle='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[0].pendingOutcomeLeg=structuredClone(x.d.claims[1].pendingOutcomeLeg)}}, // REQ carries no leg
    // ---- P2 observations must stay recorded-only ----
    {kind:'negative',fn:x=>{x.d.p2Observations[0].disposition='AUTO_APPLIED_NOW'}},
    // ---- UNKNOWN retention ----
    {kind:'unknown',fn:x=>{x.d.claims[0].elementAttribution.unknownCq='RESOLVED_PROFESSIONALLY'}},
    {kind:'unknown',fn:x=>{x.d.claims[1].elementAttribution.unknownCq='RESOLVED_PROFESSIONALLY'}},
    {kind:'unknown',fn:x=>{x.d.qualification.parentRoot='QUALIFIED'}},
    {kind:'unknown',fn:x=>{x.d.authority.claimLifecycleEffect='LIFECYCLE_DECIDED_NO_UNKNOWNS_LEFT'}},
    {kind:'unknown',fn:x=>{x.d.nonselectedClaims.untouched=0}},                           // 104 others must stay pending
    // ---- STALE bindings ----
    {kind:'stale',fn:x=>{x.d.claims[0].recordBindings.claim.digestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[0].recordBindings.evidence.digestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].recordBindings.candidateLocator.digestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].recordBindings.candidateLocator.recordVersion='0.1.2'}},
    {kind:'stale',fn:x=>{x.d.claims[1].pendingOutcomeLeg.relationDigestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].pendingOutcomeLeg.evidenceDigestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].pendingOutcomeLeg.locatorDigestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].pendingOutcomeLeg.locatorRef='se:LOC-CS2023-SE-VALIDATION-KU'}},  // p246 posing as p248
    {kind:'stale',fn:x=>{x.d.claims[1].pendingOutcomeLeg.printedPage='246'}},
    {kind:'stale',fn:x=>{x.d.recordVersion='0.0.9'}},
    {kind:'stale',fn:x=>{x.d.claims[0].decisionRecordVersion='0.2.0'}},
  ];
  for(const [i,mutant] of mutants.entries()){
    const x={d:structuredClone(document),f:structuredClone(frozen),c:structuredClone(candidate),
      fb:structuredClone(frozenBytes),cb:structuredClone(candidateBytes)};
    mutant.fn(x);
    assert.throws(()=>verify(x.d,schema,x.f,x.c,x.fb,x.cb),assert.AssertionError,
      mutant.kind+' mutant '+i+' must fail closed');
  }
  const count=(kind)=>mutants.filter(x=>x.kind===kind).length;
  return {schema:'tally.dke.c1-r180-ai-cross-model-decision-contract-test/1',result,
    total:mutants.length+1,passed:mutants.length+1,
    positive:1,negative:count('negative'),unknown:count('unknown'),stale:count('stale'),
    receipts:3,distinctFamilies:3,decisions:2,
    reviewDecisionNodesInGraph:0,claimsPending:106,untouchedPendingClaims:104,
    originalGraphModified:false,professionalOracleAttested:false,formalSapQualified:false};
}

if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  assert.ok(process.argv.includes('--check'),'explicit --check required');
  const ob=readFileSync(new URL('../../../v2.1.1/knowledge/generated/graph.jsonld',import.meta.url));
  const cb=readFileSync(new URL('../generated/graph.jsonld',import.meta.url));
  const db=readFileSync(new URL('../source/C1_AI_CROSS_MODEL_REVIEW_DECISIONS_R180.json',import.meta.url));
  const sb=readFileSync(new URL('../model/ai-cross-model-review-decision.schema.json',import.meta.url));
  const lb=readFileSync(new URL('./ai-cross-model-review-decision.mjs',import.meta.url));
  assert.equal(gitBlob(ob),ORIGINAL_GRAPH_BLOB,'frozen graph blob drift');
  assert.equal(gitBlob(cb),REVIEWED_GRAPH_BLOB,'reviewed candidate graph blob drift');
  assert.equal(gitBlob(db),AI_DECISION_BLOB,'sidecar blob pin drift (constants must match written bytes)');
  assert.equal(gitBlob(sb),AI_DECISION_SCHEMA_BLOB,'schema blob pin drift (constants must match written bytes)');
  assert.equal(gitBlob(db),git(['rev-parse','HEAD:'+AI_DECISION_PATH]),'sidecar bytes differ from exact HEAD');
  assert.equal(gitBlob(sb),git(['rev-parse','HEAD:'+AI_DECISION_SCHEMA_PATH]),'schema bytes differ from exact HEAD');
  assert.equal(gitBlob(lb),git(['rev-parse','HEAD:'+LIB_PATH]),'validator bytes differ from exact HEAD');
  const d=JSON.parse(db.toString('utf8'));
  console.log(JSON.stringify({...selfTest(d,JSON.parse(sb.toString('utf8')),JSON.parse(ob.toString('utf8')),JSON.parse(cb.toString('utf8')),ob,cb),
    node:process.version,platform:process.platform,
    recordedAtRevision:git(['rev-parse','HEAD']),tree:git(['rev-parse','HEAD^{tree}']),
    sidecarBlob:gitBlob(db),schemaBlob:gitBlob(sb),validatorBlob:gitBlob(lb),
    originalBlob:ORIGINAL_GRAPH_BLOB,reviewedCandidateBlob:REVIEWED_GRAPH_BLOB},null,2));
}
