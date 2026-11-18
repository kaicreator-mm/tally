// Tally v2.1.2 R180: AI non-human cross-model review decision ledger validator.
//
// Materializes nothing into the graph: the GPT/GLM/Qwen cross-family review of
// the two selected original Claims is recorded as a versioned sidecar ledger
// bound to the exact reviewed graph blob (a49ddf58...). Every promotion path
// fails closed here: no Claim lifecycle transition (106 claims stay PENDING,
// 0 in-graph ReviewDecision), no human/professional Oracle laundering, no
// publisher rights inflation, no SAP/Root/Global/R8/Effect grant, and the
// pending p248 support leg (REL-TVV-0024 / EVD-TVV-0016 / LOC-...-LO2) is
// never retroactively accepted. Receipt identity is pinned in code so a
// co-mutated sidecar cannot forge the reviewer roster: duplicate model
// families fail the quorum and a family label contradicting its own
// disclosure (e.g. GLM presented as Gemini) is rejected — the Gemini lane
// itself remains NOT_RUN.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {nodesDeepEqual} from './rdf-min.mjs';
import {canonicalRecordDigest, gitBlob, validateSchema} from './source-owner-disposition.mjs';

export const REPO_ROOT=fileURLToPath(new URL('../../../../../',import.meta.url));
export const AI_DECISION_PATH='docs/versions/v2.1.2/knowledge/source/C1_AI_CROSS_MODEL_REVIEW_DECISIONS_R180.json';
export const AI_DECISION_SCHEMA_PATH='docs/versions/v2.1.2/knowledge/model/ai-cross-model-review-decision.schema.json';
export const AI_DECISION_BLOB='f93190035f33516afd374b756897e0ce01fa29cb';
export const AI_DECISION_SCHEMA_BLOB='05a62261c189cd9aedd3ebc99416446400d00c65';

// reviewed source pins (must equal the R180 integrator receipt and the three
// model receipts; all four independently pinned EXACT_HEAD 580b3ace...)
export const ORIGINAL_GRAPH_BLOB='0f6007315fd7352fe6fd218effcfecf672cb4961';
export const REVIEWED_GRAPH_BLOB='a49ddf58a0c4a5b25ebdd4018c8234f64e6365e0';
export const R180_COMMENT_ID='6983286610';
export const R180_COMMENT_URL='https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6983286610';

// R137 p248 pending outcome leg ids (content verified by all three reviews,
// lifecycle must remain PENDING_REVIEW)
export const R180_P248_IDS={
  relation:'se:REL-TVV-0024',
  evidence:'se:EVD-TVV-0016',
  locator:'se:LOC-CS2023-SE-VALIDATION-LO2',
};

// Code-fixed reviewer roster: the exact identity subset each GitHub review
// receipt disclosed. The sidecar must carry these values verbatim; drift here
// is forged provenance, not a metadata edit.
export const R180_RECEIPTS=[
  {
    receiptKey:'R180-GPT',
    modelFamily:'GPT',
    provider:'OpenAI',
    modelId:'GPT-6',
    commentId:'6976173032',
    commentUrl:'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6976173032',
    reqVerdict:'ACCEPT_SCOPED',
    vvVerdict:'ACCEPT_SCOPED',
    findingsP0:0,
    findingsP1:0,
    findingsP2:2,
    blindFirstPass:true,
  },
  {
    receiptKey:'R180-GLM',
    modelFamily:'GLM',
    provider:'BigModel',
    modelId:'GLM-5.3',
    commentId:'6976329069',
    commentUrl:'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6976329069',
    reqVerdict:'ACCEPT_SCOPED',
    vvVerdict:'ACCEPT_SCOPED',
    findingsP0:0,
    findingsP1:0,
    findingsP2:3,
    blindFirstPass:true,
  },
  {
    receiptKey:'R180-QWEN',
    modelFamily:'QWEN',
    provider:'Alibaba Model Studio International',
    modelId:'qwen-alibaba-model-studio-intl/qwen3.8-max',
    commentId:'6976586466',
    commentUrl:'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6976586466',
    reqVerdict:'ACCEPT_SCOPED',
    vvVerdict:'ACCEPT_SCOPED',
    findingsP0:0,
    findingsP1:0,
    findingsP2:1,
    blindFirstPass:true,
  },
];
export const R180_DISTINCT_FAMILIES=['GPT','GLM','QWEN'];

// Per-claim pinned semantics: element attribution and UNKNOWN conservation
// must match the R100 Owner dispositions — the AI ledger narrows nothing.
export const R180_EXPECTED_CLAIMS=[
  {
    claimId:'se:CLM-REQ-FUNCTIONAL-DOC-METHODS',
    taxonomyNodeRef:'se:NODE-CS2023-SE-PRODUCT-REQUIREMENTS',
    candidateElementRef:'dke:ev16.d02.se-requirements-child',
    positiveCq:'CS2023_REQUIREMENTS_TEACHING',
    unknownCq:'REQUIREMENTS_UNIFORM_METHOD',
    scopeMarkers:['SE-Requirements','KA Core 1a-1c','curriculum teaching and bounded original paraphrase only'],
  },
  {
    claimId:'se:CLM-TVV-0001',
    taxonomyNodeRef:'se:NODE-CS2023-SE-VALIDATION',
    candidateElementRef:'dke:ev16.d02.se-verification-child',
    positiveCq:'CS2023_VV_TEACHING',
    unknownCq:'VV_AUTOMATIC_BUSINESS_ACCEPTANCE',
    scopeMarkers:['SE-Validation','learning outcome 2','curriculum teaching and bounded original paraphrase only'],
  },
];

const IDENTITY_FIELDS=['receiptKey','modelFamily','provider','modelId','commentId','commentUrl',
  'reqVerdict','vvVerdict','findingsP0','findingsP1','findingsP2','blindFirstPass'];

function assertReceiptRoster(document){
  const receipts=document.receipts;
  assert.ok(Array.isArray(receipts),'receipts array required');
  assert.equal(receipts.length,3,'quorum requires exactly three receipts');
  assert.deepEqual(receipts.map(r=>r.receiptKey),R180_RECEIPTS.map(r=>r.receiptKey),'receipt order/keys drift');
  for(let i=0;i<3;i++){
    const actual=Object.fromEntries(IDENTITY_FIELDS.map(k=>[k,receipts[i][k]]));
    assert.deepEqual(actual,R180_RECEIPTS[i],'receipt identity drift/forgery: '+receipts[i].receiptKey);
    assert.notEqual(receipts[i].modelFamily,'GEMINI','no receipt may claim the GEMINI family');
  }
  const families=receipts.map(r=>r.modelFamily);
  assert.deepEqual([...new Set(families)].sort(),[...R180_DISTINCT_FAMILIES].sort(),
    'duplicate model family: three ACTUAL distinct families required');
  const glm=receipts.find(r=>r.modelFamily==='GLM');
  assert.ok(glm,'GLM family receipt required');
  assert.ok(String(glm.modelFamilyDisclosure).includes('NOT_COUNTED_AS_GEMINI'),
    'GLM substitute receipt must disclose it is not counted as Gemini');
  for(const receipt of receipts){
    assert.equal(receipt.blindFirstPass,true,'blind first pass required: '+receipt.receiptKey);
    assert.equal(receipt.findingsP0,0,'P0 must be zero: '+receipt.receiptKey);
    assert.equal(receipt.findingsP1,0,'P1 must be zero: '+receipt.receiptKey);
    assert.equal(receipt.reqVerdict,'ACCEPT_SCOPED','REQ verdict drift: '+receipt.receiptKey);
    assert.equal(receipt.vvVerdict,'ACCEPT_SCOPED','VV verdict drift: '+receipt.receiptKey);
    assert.equal(receipt.commentUrl,'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-'+receipt.commentId,
      'comment URL must bind its own comment id: '+receipt.receiptKey);
  }
  return receipts;
}

function readPinned(path,expected){
  const bytes=readFileSync(REPO_ROOT+path);
  assert.equal(gitBlob(bytes),expected,`AI cross-model ledger input Git Blob drift: ${path}`);
  return bytes;
}

export function readAIDecisionInputs(){
  const sourceBytes=readPinned(AI_DECISION_PATH,AI_DECISION_BLOB);
  const schemaBytes=readPinned(AI_DECISION_SCHEMA_PATH,AI_DECISION_SCHEMA_BLOB);
  return {
    document:JSON.parse(sourceBytes.toString('utf8')),
    schema:JSON.parse(schemaBytes.toString('utf8')),
    sourceBlob:gitBlob(sourceBytes),
    schemaBlob:gitBlob(schemaBytes),
  };
}

const lookup=(nodes,id,label)=>{
  const node=nodes.find(candidate=>candidate['@id']===id);
  assert.ok(node,`AI cross-model ledger: missing ${label} ${id}`);
  return node;
};

function assertExactBinding(record,graphNode,kind){
  assert.equal(graphNode['se:record_version'],record.recordVersion,'STALE '+kind+' record_version');
  assert.equal(canonicalRecordDigest(graphNode),record.digestSha256,'STALE '+kind+' digest');
}

export function assertAICrossModelReviewDecisions(document,schema,frozenNodes,candidateNodes,frozenBytes,candidateBytes){
  // envelope + authority
  assert.equal(document.schema,'tally.dke.c1-ai-cross-model-review-decisions/1');
  assert.equal(document.recordVersion,'0.1.0');
  assert.equal(document.version,'v2.1.2-candidate');
  assert.equal(document.status,'AI_NON_HUMAN_CROSS_MODEL_REVIEW_DECISIONS_VERSIONED_NOT_CLAIM_LIFECYCLE');
  assert.ok(/^\d{4}-\d{2}-\d{2}$/.test(document.recordedOn));
  assert.equal(document.authority.writerRole,'ORIGINAL_TALLY_399_SINGLE_WRITER');
  assert.equal(document.authority.decisionCategory,'AI_NON_HUMAN_CROSS_MODEL_REVIEW_EVIDENCE_LEDGER');
  assert.equal(document.authority.integrationDecisionRecord,R180_COMMENT_URL);
  assert.equal(document.authority.claimLifecycleEffect,'NONE_ORIGINAL_AND_CANDIDATE_CLAIM_REMAIN_PENDING');
  assert.equal(document.authority.reviewDecisionNodesInGraph,0);
  assert.ok(document.authority.humanAttestationSubstitute.startsWith('FORBIDDEN'),
    'AI review must never substitute human professional certification');
  assert.equal(document.authority.professionalOracle,'NOT_ATTESTED');
  assert.equal(document.authority.publicationPermission,'NOT_GRANTED_BY_THIS_OWNER_DECISION');

  // reviewed source pins
  assert.equal(document.sources.originalGraphBlob,ORIGINAL_GRAPH_BLOB);
  assert.equal(document.sources.reviewedGraphBlob,REVIEWED_GRAPH_BLOB);
  assert.equal(document.sources.originalRepoCommit,'3e94685ca57a4b7258988826697af1c1c3a6b3c8');
  assert.ok(document.sources.publisher.includes('DOI 10.1145/3664191'));
  assert.ok(document.sources.rightsScope.includes('no substantial text republication'));

  // quorum contract
  assert.equal(document.reviewContract.quorum,'THREE_ACTUAL_DISTINCT_MODEL_FAMILIES');
  assert.equal(document.reviewContract.requiredVerdictPerClaim,'ACCEPT_SCOPED');
  assert.equal(document.reviewContract.maxBlockerFindings,'P0_AND_P1_MUST_BE_ZERO_PER_RECEIPT');
  assert.equal(document.reviewContract.blindFirstPassRequired,true);
  assert.deepEqual(document.reviewContract.distinctFamilies,R180_DISTINCT_FAMILIES);
  assert.equal(document.reviewContract.distinctFamilyCount,3);
  assert.equal(document.reviewContract.geminiFamilyVoteCount,0);
  assert.ok(document.reviewContract.geminiLane.includes('NOT_RUN'),'Gemini lane must stay NOT_RUN');
  assert.ok(document.reviewContract.geminiLane.includes('NOT_GEMINI'),
    'GLM substitute must not be counted as Gemini');
  assert.equal(document.reviewContract.integrationResult,'REQ_AND_VV_3_OF_3_ACCEPT_SCOPED_AI_NON_HUMAN');

  // receipts: code-pinned roster, distinct families, unanimity, P0/P1=0
  assertReceiptRoster(document);

  // graph-wide conservation: the ledger mutates nothing
  const frozenBytesBlob=gitBlob(frozenBytes),candidateBytesBlob=gitBlob(candidateBytes);
  assert.equal(frozenBytesBlob,ORIGINAL_GRAPH_BLOB,'frozen v2.1.1 graph blob drift');
  assert.equal(candidateBytesBlob,REVIEWED_GRAPH_BLOB,'reviewed candidate graph blob drift');
  assert.equal(frozenNodes.length,807,'frozen v2.1.1 must keep exactly 807 nodes');
  assert.equal(candidateNodes.length,815,'candidate v2.1.2 must keep exactly 815 nodes');
  const claims=candidateNodes.filter(n=>n['@type']==='se:KnowledgeClaim');
  assert.equal(claims.length,106);
  assert.equal(claims.filter(n=>n['se:review_status']==='PENDING').length,106,
    'all 106 KnowledgeClaims must remain PENDING');
  assert.equal(candidateNodes.filter(n=>n['se:lifecycle_status']==='ACCEPTED').length,0);
  assert.equal(candidateNodes.filter(n=>n['@type']==='se:ReviewDecision').length,0,
    'AI review must not create in-graph ReviewDecision nodes');
  assert.equal(candidateNodes.filter(n=>String(n['@id']).startsWith('se:AIM-')).length,0,
    'the AI ledger is sidecar-only; no AIM nodes may be materialized into the graph');
  assert.equal(candidateNodes.filter(n=>n['@type']==='se:SourceOwnerDisposition').length,2);
  assert.equal(candidateNodes.filter(n=>n['@type']==='se:SelectedCqSourceBinding').length,1);
  // wrong-source masquerade: the reviewed publisher identity is pinned
  const source=lookup(candidateNodes,'se:SRC-CS2023','publisher source');
  assert.equal(source['se:stable_identifier'],'DOI 10.1145/3664191','publisher source identity drift');

  // exactly the two selected claims, each exact-bound to the reviewed graph
  assert.equal(document.claims.length,2,'exactly the two selected REQ/VV claims may be decided');
  for(let rowIndex=0;rowIndex<document.claims.length;rowIndex++){
    const row=document.claims[rowIndex];
    const expected=R180_EXPECTED_CLAIMS[rowIndex];
    assert.equal(row.claimId,expected.claimId,'claim row drift at index '+rowIndex);
    validateSchema(row,schema);
    assert.equal(row.decisionLifecycle,'ACTIVE');
    assert.equal(row.decisionRecordVersion,'0.1.0','STALE decision record_version (bump only via a superseding ledger)');
    const attribution=row.elementAttribution;
    assert.equal(attribution.taxonomyNodeRef,expected.taxonomyNodeRef,'taxonomy attribution drift');
    assert.equal(attribution.candidateElementRef,expected.candidateElementRef,'candidate element drift');
    assert.equal(attribution.positiveCq,expected.positiveCq,'positive CQ drift');
    assert.equal(attribution.unknownCq,expected.unknownCq,'UNKNOWN conservation drift');
    for(const marker of expected.scopeMarkers){
      assert.ok(row.scope.includes(marker),'scope marker drift: '+marker);
    }
    const claim=lookup(candidateNodes,row.claimId,'candidate Claim');
    const frozenClaim=lookup(frozenNodes,row.claimId,'original Claim');
    const evidence=lookup(candidateNodes,row.evidenceId,'candidate Evidence');
    const frozenEvidence=lookup(frozenNodes,row.evidenceId,'original Evidence');
    const locator=lookup(candidateNodes,row.locatorId,'candidate Locator');
    const frozenLocator=lookup(frozenNodes,row.locatorId,'original Locator');
    assert.ok(nodesDeepEqual(claim,frozenClaim),'AI ledger must not mutate the original Claim');
    assert.ok(nodesDeepEqual(evidence,frozenEvidence),'AI ledger must not mutate the original Evidence');
    assertExactBinding(row.recordBindings.claim,claim,'claim '+row.claimId);
    assertExactBinding(row.recordBindings.evidence,evidence,'evidence '+row.evidenceId);
    assertExactBinding(row.recordBindings.locator,frozenLocator,'original locator '+row.locatorId);
    assert.equal(locator['se:record_version'],row.recordBindings.candidateLocator.recordVersion,
      'STALE candidate locator record_version '+row.locatorId);
    assert.equal(canonicalRecordDigest(locator),row.recordBindings.candidateLocator.digestSha256,
      'STALE candidate locator digest '+row.locatorId);
    assert.equal(claim['se:review_status'],'PENDING');
    assert.equal(claim['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(evidence['se:extraction_status'],'EXTRACTED_PENDING_REVIEW');
    assert.equal(evidence['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(evidence['se:source_locator_ref']?.['@id'],row.locatorId);
    // counterexamples and stale discriminators must be real content, not placeholders
    assert.ok(Array.isArray(row.counterexamples)&&row.counterexamples.length>=2,'counterexamples required');
    assert.deepEqual(new Set(row.counterexamples.map(c=>c.id)).size,row.counterexamples.length,'duplicate counterexample id');
    for(const counter of row.counterexamples){
      assert.ok(counter.id&&counter.example&&counter.expectedRejection,'counterexample fields required');
    }
    assert.ok(Array.isArray(row.staleDiscriminators)&&row.staleDiscriminators.length>=1,'stale discriminators required');
    assert.equal(row.integration.receiptCount,3);
    assert.equal(row.integration.distinctFamilyCount,3);
    assert.equal(row.integration.acceptScopedCount,3);
    assert.equal(row.integration.p0Total,0);
    assert.equal(row.integration.p1Total,0);
    assert.equal(row.integration.result,'AI_CROSS_MODEL_3_OF_3_ACCEPT_SCOPED');
    if(row.claimId==='se:CLM-TVV-0001'){
      assertPendingOutcomeLeg(document,row,candidateNodes);
    }else{
      assert.equal(row.pendingOutcomeLeg,undefined,'only the VV claim carries a pending outcome leg');
    }
  }
  // the two decisions must be exactly the R180 selected pair
  assert.deepEqual(document.claims.map(r=>r.claimId).sort(),
    ['se:CLM-REQ-FUNCTIONAL-DOC-METHODS','se:CLM-TVV-0001']);

  // P2 observations are recorded, never auto-applied
  const p2Ids=new Set(document.p2Observations.map(p=>p.id));
  for(const id of ['P2-REQ-EVD-STALE-UNCERTAINTY','P2-VV-SECTION-VS-PENDING-PAGE-TRACEABILITY','P2-GLM-PHRASING-AND-UNCERTAINTY-NOTE']){
    assert.ok(p2Ids.has(id),'missing P2 observation '+id);
  }
  for(const observation of document.p2Observations){
    assert.ok(observation.disposition.startsWith('PRESERVED_NOT_AUTO_APPLIED'),
      'P2 observations must not be auto-applied: '+observation.id);
  }

  // non-selected claims accounting
  assert.equal(document.nonselectedClaims.totalKnowledgeClaims,106);
  assert.equal(document.nonselectedClaims.selectedByThisLedger,2);
  assert.equal(document.nonselectedClaims.untouched,104);
  assert.equal(document.nonselectedClaims.totalKnowledgeClaims,
    document.nonselectedClaims.selectedByThisLedger+document.nonselectedClaims.untouched);

  // qualification boundaries
  const q=document.qualification;
  assert.equal(q.originalFrozenGraphModified,false);
  assert.equal(q.reviewedCandidateGraphMutatedByThisLedger,false);
  assert.equal(q.aiReviewDecisionsRecorded,2);
  assert.equal(q.humanOracleAttestations,0);
  assert.equal(q.formalSap,'0/6');
  assert.equal(q.formalGlobal,'0/5');
  assert.equal(q.parentRoot,'UNKNOWN');
  assert.equal(q.originalR8,'1/53_SCOPED_ONLY');
  assert.equal(q.originalD04,'NOT_RUN');
  assert.equal(q.effectAuthorized,false);
  assert.equal(q.releaseAuthorized,false);
  assert.ok(q.forgeAiDemoCellAuthorization.startsWith('NONE_BY_THIS_LEDGER'),
    'Forge #208 authorization must be decided separately');
  return {
    result:'AI_NON_HUMAN_CROSS_MODEL_LEDGER_SCOPED__CLAIM_LIFECYCLE_PENDING',
    decisions:2,receipts:3,distinctFamilies:3,p0Total:0,p1Total:0,
    reviewDecisionNodesInGraph:0,claimsPending:106,untouchedPendingClaims:104,
    pendingOutcomeLegLifecycle:'PENDING_REVIEW',
  };
}

function assertPendingOutcomeLeg(document,row,candidateNodes){
  const leg=row.pendingOutcomeLeg;
  assert.ok(leg,'VV claim must record the p248 pending outcome leg');
  assert.equal(leg.status,'CONTENT_VERIFIED_BY_ALL_THREE_REVIEWS__LIFECYCLE_NOT_ELEVATED');
  assert.equal(leg.requiredLifecycle,'PENDING_REVIEW');
  assert.ok(leg.rule.includes('never retroactively accepts'));
  const relation=lookup(candidateNodes,leg.relationRef,'pending relation');
  const evidence=lookup(candidateNodes,leg.evidenceRef,'pending evidence');
  const locator=lookup(candidateNodes,leg.locatorRef,'pending locator');
  assert.equal(relation['@type'],'se:RelationshipAssertion');
  assert.equal(relation['se:lifecycle_status'],'PENDING_REVIEW','p248 relation must stay PENDING_REVIEW');
  assert.equal(relation['se:record_version'],leg.relationRecordVersion,'STALE p248 relation record_version');
  assert.equal(canonicalRecordDigest(relation),leg.relationDigestSha256,'STALE p248 relation digest');
  assert.equal(evidence['se:lifecycle_status'],'PENDING_REVIEW','p248 evidence must stay PENDING_REVIEW');
  assert.equal(evidence['se:extraction_status'],'EXTRACTED_PENDING_REVIEW');
  assert.equal(evidence['se:record_version'],leg.evidenceRecordVersion,'STALE p248 evidence record_version');
  assert.equal(canonicalRecordDigest(evidence),leg.evidenceDigestSha256,'STALE p248 evidence digest');
  assert.equal(locator['se:record_version'],leg.locatorRecordVersion,'STALE p248 locator record_version');
  assert.equal(canonicalRecordDigest(locator),leg.locatorDigestSha256,'STALE p248 locator digest');
  assert.equal(leg.printedPage,'248');
  assert.equal(leg.pdfZeroBasedPage,'247');
}
