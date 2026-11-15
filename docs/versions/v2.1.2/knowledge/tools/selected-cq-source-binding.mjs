// Tally v2.1.2 R137: SelectedCqSourceBinding producer + native validator.
//
// Executable two-leg source binding for the selected candidate VV CQ
// (CS2023_VV_TEACHING): the section-start leg (p246 SECTION locator, shared
// 8-Evidence/5-Claim anchor, unchanged) plus a distinct outcome-pinpoint leg
// (p248 PAGE locator + separate pending Evidence + claim-support assertion).
// The bound original Claim/Evidence/Locator records stay byte-identical with
// the frozen graph (SourceOwnerDisposition hard gate); consumption is carried
// by NEW overlay additions only. Orphan bindings (missing graph records or
// missing overlay additions), stale digests/versions, single-leg disguise,
// fan-out damage and qualification laundering all fail closed. This binding
// is research-candidate scope only: no frozen R8 Cell, no Claim lifecycle
// promotion, no professional Oracle attestation.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {nodesDeepEqual, projectRecord} from './rdf-min.mjs';
import {canonicalRecordDigest, validateSchema} from './source-owner-disposition.mjs';

export const REPO_ROOT=fileURLToPath(new URL('../../../../../',import.meta.url));
export const SOURCE_OWNER_PATH='docs/versions/v2.1.2/knowledge/source/C1_CLAIM_OWNER_DISPOSITIONS_R100.json';
export const SCB_SCHEMA_PATH='docs/versions/v2.1.2/knowledge/model/selected-cq-source-binding.schema.json';
export const OVERLAY_PATH='docs/versions/v2.1.2/knowledge/source/C1_OWNER_REVISION_R13.json';

// exact graph ids of the R137 selected-CQ consumption chain
export const R137_IDS={
  scb:'se:SCB-CS2023-VV-R1',
  claim:'se:CLM-TVV-0001',
  sectionEvidence:'se:EVD-TVV-0003',
  sectionLocator:'se:LOC-CS2023-SE-VALIDATION-KU',
  outcomeLocator:'se:LOC-CS2023-SE-VALIDATION-LO2',
  outcomeEvidence:'se:EVD-TVV-0016',
  claimSupport:'se:REL-TVV-0024',
};
export const R137_SHARED_SECTION_EVIDENCE=[
  'se:EVD-TVV-0003','se:EVD-TVV-0004','se:EVD-TVV-0005','se:EVD-TVV-0006',
  'se:EVD-TVV-0007','se:EVD-TVV-0008','se:EVD-TVV-0009','se:EVD-TVV-0014',
];
export const R137_SHARED_SECTION_DIRECT_CLAIMS=[
  'se:CLM-TVV-0001','se:CLM-TVV-0002','se:CLM-TVV-0003','se:CLM-TVV-0006','se:CLM-TVV-0007',
];
export const R137_ADDITION_IDS=[
  R137_IDS.outcomeLocator,R137_IDS.outcomeEvidence,R137_IDS.claimSupport,
];

// R140 fixed Source-First reviewer anchor: independently from candidate/sidecar
// revision or rebuilt graph/manifest hashes, retain exact bounded *educational*
// semantics recorded from CS2023 SE-Validation (publisher PDF index247,
// printed p248, Illustrative Learning Outcomes, CS Core #2).  Any change to
// these constants is a source-contract change requiring distinct review.
// THIS IS NOT A PROFESSIONAL ORACLE OR AN ORIGINAL CLAIM ADMISSION.
export const R140_PUBLISHER_VV_LO2_ANCHOR=Object.freeze({
  source:'Computer Science Curricula 2023, January 2024, DOI 10.1145/3664191',
  sourcePdf:'https://ieeecs-media.computer.org/media/education/reports/CS2023.pdf',
  sectionPage:'246', outcomePage:'248',
  educationalMeaning:'Students distinguish program validation from verification; no test-passed-implies-customer-acceptance entailment',
});
export const R140_REVIEWED_OUTCOME_EVIDENCE=Object.freeze({
  "@id": "se:EVD-TVV-0016",
  "@type": "se:Evidence",
  "se:created_on": "2026-11-15",
  "se:record_version": "0.1.0",
  "se:lifecycle_status": "PENDING_REVIEW",
  "se:source_locator_ref": {
    "@id": "se:LOC-CS2023-SE-VALIDATION-LO2"
  },
  "se:evidence_type": "DIRECT_SOURCE_PARAPHRASE",
  "se:statement": "CS2023's SE-Validation CS Core illustrative learning outcome 2 expects students to distinguish between program validation and verification, completing the outcome-specific source leg for the two-question verification/validation distinction.",
  "se:context": "The paired CS Core illustrative learning outcome 2 of the SE-Validation unit, on the outcome-specific page.",
  "se:method": "Full-text extraction of the official Body of Knowledge PDF on 2026-11-15; outcome-list page anchored separately from the section-start concepts page (zero-based index 245, printed page 246).",
  "se:result": "Outcome-specific anchor for the claim's learning-outcome leg; section-start concepts remain anchored by the separate 0.1.1 SECTION locator.",
  "se:limitations": "A learning outcome is a curriculum teaching objective; it does not elaborate when each question is fully answered and is not a professional-standard definition.",
  "se:uncertainty": "The publisher PDF raw-byte SHA256 and any independent page-image audit remain UNKNOWN.",
  "se:extraction_status": "EXTRACTED_PENDING_REVIEW",
  "se:synthetic": false
});
export const R140_REVIEWED_CLAIM_SUPPORT=Object.freeze({
  "@id": "se:REL-TVV-0024",
  "@type": "se:RelationshipAssertion",
  "se:created_on": "2026-11-15",
  "se:record_version": "0.1.0",
  "se:lifecycle_status": "PENDING_REVIEW",
  "se:subject_ref": {
    "@id": "se:CLM-TVV-0001"
  },
  "se:predicate": {
    "@id": "se:SUPPORTED_BY"
  },
  "se:object_ref": {
    "@id": "se:EVD-TVV-0016"
  },
  "se:evidence_refs": {
    "@id": "se:EVD-TVV-0016"
  },
  "se:context": "Outcome-specific support leg: the V&V two-question definition claim is additionally supported by evidence anchored to the CS Core illustrative learning outcome 2 page.",
  "se:rationale": "The claim's learning-outcome leg restates evidence extracted from the CS Core outcome 2 page (zero-based index 247, printed page 248), complementing the section-start locator; the original claim record itself remains byte-identical.",
  "se:synthetic": false
});
// Canonical SHA-256 of *code-fixed reviewed records*, not of mutable source
// pins supplied by the same producer. Synchronized candidate/sidecar drift
// cannot change these expected digests at runtime.
export const R140_REVIEWED_OUTCOME_EVIDENCE_DIGEST=canonicalRecordDigest(R140_REVIEWED_OUTCOME_EVIDENCE);
export const R140_REVIEWED_CLAIM_SUPPORT_DIGEST=canonicalRecordDigest(R140_REVIEWED_CLAIM_SUPPORT);

export const readText=(relPath)=>readFileSync(REPO_ROOT+relPath,'utf8');

export function selectedCqBindingRow(document){
  assert.equal(document.schema,'tally.dke.c1-scoped-source-owner-claim-dispositions/1');
  const vv=document.claims.find(row=>row.claimId===R137_IDS.claim);
  assert.ok(vv,'selected-CQ sidecar row for the VV claim missing');
  const binding=vv.elementAttribution?.selectedCqSourceBinding;
  assert.ok(binding,'selectedCqSourceBinding missing from the VV elementAttribution');
  assert.equal(binding.id,R137_IDS.scb);
  return binding;
}

// Materialize the sidecar binding into a graph node. Locator/claim digests are
// recomputed from the actual candidate graph and must equal the sidecar values
// (drift = STALE, fails closed here, not silently materialized).
export function materializeSelectedCqSourceBinding(document,schema,candidateNodes,overlay){
  const binding=selectedCqBindingRow(document);
  const lookupId=(id,label)=>{
    const node=candidateNodes.find(c=>c['@id']===id);
    assert.ok(node,`SelectedCqSourceBinding: missing ${label} ${id}`);
    return node;
  };
  const claim=lookupId(binding.claimRef,'claim');
  const sectionLocator=lookupId(binding.sectionLeg.locatorRef,'section locator');
  const outcomeLocator=lookupId(binding.outcomeLeg.locatorRef,'outcome locator');
  assert.equal(claim['se:record_version'],binding.claimRecordVersion,'STALE SCB claim record_version');
  assert.equal(canonicalRecordDigest(claim),binding.claimDigestSha256,'STALE SCB claim digest');
  assert.equal(sectionLocator['se:record_version'],binding.sectionLeg.recordVersion,'STALE SCB section locator record_version');
  assert.equal(canonicalRecordDigest(sectionLocator),binding.sectionLeg.digestSha256,'STALE SCB section locator digest');
  assert.equal(outcomeLocator['se:record_version'],binding.outcomeLeg.recordVersion,'STALE SCB outcome locator record_version');
  assert.equal(canonicalRecordDigest(outcomeLocator),binding.outcomeLeg.digestSha256,'STALE SCB outcome locator digest');
  // additions are the only authorized origin of the new outcome-leg records
  const additionIds=new Set((overlay.additions??[]).map(a=>a['@id']));
  for(const id of R137_ADDITION_IDS){
    assert.ok(additionIds.has(id),`orphan R137 record: ${id} must be an overlay addition`);
  }
  assert.notEqual(binding.sectionLeg.locatorRef,binding.outcomeLeg.locatorRef,'SCB two legs must use distinct locators');
  assert.notEqual(binding.sectionLeg.pdfZeroBasedPage,binding.outcomeLeg.pdfZeroBasedPage,'SCB two legs must pin distinct PDF pages (no single-page disguise)');
  const record={
    record_version:binding.recordVersion,
    lifecycle_status:binding.lifecycle,
    created_on:binding.createdOn,
    positive_cq:binding.positiveCq,
    binding_scope:binding.bindingScope,
    claim_ref:binding.claimRef,
    claim_record_version:binding.claimRecordVersion,
    claim_digest_sha256:binding.claimDigestSha256,
    section_locator_ref:binding.sectionLeg.locatorRef,
    section_locator_record_version:binding.sectionLeg.recordVersion,
    section_locator_digest_sha256:binding.sectionLeg.digestSha256,
    section_evidence_ref:binding.sectionLeg.evidenceRef,
    section_anchor_type:binding.sectionLeg.anchorType,
    section_pdf_zero_based_page:binding.sectionLeg.pdfZeroBasedPage,
    section_printed_page:binding.sectionLeg.printedPage,
    outcome_locator_ref:binding.outcomeLeg.locatorRef,
    outcome_locator_record_version:binding.outcomeLeg.recordVersion,
    outcome_locator_digest_sha256:binding.outcomeLeg.digestSha256,
    outcome_evidence_ref:binding.outcomeLeg.evidenceRef,
    outcome_anchor_type:binding.outcomeLeg.anchorType,
    outcome_pdf_zero_based_page:binding.outcomeLeg.pdfZeroBasedPage,
    outcome_printed_page:binding.outcomeLeg.printedPage,
    claim_support_ref:binding.claimSupportRef,
    source_ref:rowSourceId(document),
    taxonomy_node_ref:document.claims.find(row=>row.claimId===R137_IDS.claim).elementAttribution.taxonomyNodeRef,
    candidate_element_attribution:document.claims.find(row=>row.claimId===R137_IDS.claim).elementAttribution.candidateElementRef,
    claim_lifecycle_effect:'NONE_ORIGINAL_AND_CANDIDATE_CLAIM_REMAIN_PENDING',
    professional_oracle_status:document.authority.independentProfessionalOracle,
    publication_permission:document.authority.publicationPermission,
    binding_authority_ref:binding.authorityRecord,
  };
  const node=projectRecord(binding.id,'se:SelectedCqSourceBinding',record);
  validateSchema(node,schema);
  return node;
}

function rowSourceId(document){
  const vv=document.claims.find(row=>row.claimId===R137_IDS.claim);
  return vv.sourceId;
}

const idOf=node=>node['@id'];
const refsOf=(node,key)=>{
  const v=node[key];
  if(v===undefined||v===null)return [];
  return (Array.isArray(v)?v:[v]).map(x=>x&&typeof x==='object'?x['@id']:x).filter(Boolean).sort();
};

// Full executable consumption check of the R137 two-leg binding inside the
// candidate graph. Every assertion below is a hard gate; the binding must be
// consumed, not merely declared.
export function assertSelectedCqSourceBinding(document,schema,candidateNodes,overlay){
  const byId=new Map(candidateNodes.map(n=>[idOf(n),n]));
  const scb=byId.get(R137_IDS.scb);
  assert.ok(scb,'SelectedCqSourceBinding node missing from the candidate graph');
  validateSchema(scb,schema);
  // single source of truth: the committed graph node must equal a fresh
  // materialization of the sidecar binding row (any sidecar/graph divergence,
  // including STALE record_version or digest drift, fails closed here)
  const expected=materializeSelectedCqSourceBinding(document,schema,candidateNodes,overlay);
  assert.ok(nodesDeepEqual(scb,expected),'sidecar SelectedCqSourceBinding drift vs committed graph node');

  const claim=byId.get(R137_IDS.claim);
  assert.ok(claim,'selected claim missing');
  assert.equal(claim['se:review_status'],'PENDING','SCB cannot accept Claim review_status');
  assert.equal(claim['se:lifecycle_status'],'PENDING_REVIEW','SCB cannot accept Claim lifecycle');
  assert.equal(scb['se:claim_digest_sha256'],canonicalRecordDigest(claim),'STALE SCB claim digest vs candidate graph');

  // section leg: the shared SECTION anchor is untouched and consumed as before
  const sectionEvidence=byId.get(R137_IDS.sectionEvidence);
  assert.ok(sectionEvidence,'section-leg evidence missing');
  assert.equal(sectionEvidence['se:source_locator_ref']?.['@id'],R137_IDS.sectionLocator,'section-leg evidence locator drift');
  const sectionLocator=byId.get(R137_IDS.sectionLocator);
  assert.equal(sectionLocator['se:locator_type'],'SECTION','section leg must stay a SECTION anchor');
  assert.ok(sectionLocator['se:locator'].includes('zero-based index 245, printed page 246'),'section leg lost the p246 anchor');
  assert.ok(!sectionLocator['se:locator'].includes('index 247'),'section locator must not masquerade as the outcome pinpoint');
  // the original claim keeps exactly its original single SUPPORTED_BY edge
  assert.deepEqual(refsOf(claim,'se:SUPPORTED_BY'),[R137_IDS.sectionEvidence],'original claim typed edge must stay byte-identical (no added edge)');

  // outcome leg: distinct locator/evidence actually consumed
  const outcomeEvidence=byId.get(R137_IDS.outcomeEvidence);
  assert.ok(outcomeEvidence,'outcome-leg evidence missing from the graph');
  assert.equal(outcomeEvidence['se:source_locator_ref']?.['@id'],R137_IDS.outcomeLocator,'outcome evidence not bound to the p248 locator');
  assert.equal(outcomeEvidence['se:lifecycle_status'],'PENDING_REVIEW','outcome evidence must stay PENDING_REVIEW');
  assert.equal(outcomeEvidence['se:extraction_status'],'EXTRACTED_PENDING_REVIEW','outcome evidence must stay EXTRACTED_PENDING_REVIEW');
  assert.equal(outcomeEvidence['se:synthetic'],false,'outcome evidence must be non-synthetic');
  // R140 P1: source semantics may not be redefined by co-mutated overlay,
  // sidecar and graph. This independently reviewed exact outcome Evidence
  // contract covers statement, extraction method, context, limitations and
  // uncertainty; source locator alone is not evidence of LO2 meaning.
  assert.equal(outcomeEvidence['se:record_version'],'0.1.0','R140 STALE outcome Evidence record version');
  assert.equal(outcomeEvidence['se:statement'],R140_REVIEWED_OUTCOME_EVIDENCE['se:statement'],
    'R140 LO2 Evidence semantic inversion (source-anchored statement)');
  assert.equal(canonicalRecordDigest(outcomeEvidence),R140_REVIEWED_OUTCOME_EVIDENCE_DIGEST,
    'R140 LO2 Evidence semantic/digest drift (fixed independent source contract)');

  const outcomeLocator=byId.get(R137_IDS.outcomeLocator);
  assert.equal(outcomeLocator['se:locator_type'],'PAGE','outcome pinpoint must be a PAGE locator');
  assert.equal(outcomeLocator['se:verification_state'],'LOCATOR_VERIFIED','outcome locator must be LOCATOR_VERIFIED');
  assert.ok(outcomeLocator['se:locator'].includes('zero-based index 247, printed page 248'),'outcome locator lost the p248 pinpoint');
  assert.ok(outcomeLocator['se:locator'].includes('outcome 2'),'outcome locator lost the learning-outcome-2 scope');
  assert.ok(outcomeLocator['se:locator'].includes('Distinguish between program validation and verification'),'outcome locator lost the LO2 outcome text');
  assert.ok(outcomeLocator['se:source_ref']?.['@id']===sectionLocator['se:source_ref']?.['@id'],'outcome locator source drift vs section locator');
  assert.notEqual(idOf(sectionLocator),idOf(outcomeLocator),'two legs must remain distinct locators');
  assert.equal(scb['se:outcome_pdf_zero_based_page'],'247','outcome page anchor drift');
  assert.equal(scb['se:outcome_printed_page'],'248','outcome printed-page drift');
  assert.notEqual(scb['se:section_pdf_zero_based_page'],scb['se:outcome_pdf_zero_based_page'],'p246 must never double as the p248 outcome proof');
  assert.equal(canonicalRecordDigest(outcomeLocator),scb['se:outcome_locator_digest_sha256'],'STALE outcome locator digest vs graph');
  assert.equal(canonicalRecordDigest(sectionLocator),scb['se:section_locator_digest_sha256'],'STALE section locator digest vs graph');

  // claim-support assertion: typed node carrying claim -> outcome evidence
  const support=byId.get(R137_IDS.claimSupport);
  assert.ok(support,'claim-support RelationshipAssertion missing from the graph');
  assert.equal(support['@type'],'se:RelationshipAssertion','claim support must be a RelationshipAssertion');
  assert.equal(support['se:subject_ref']?.['@id'],R137_IDS.claim,'claim-support subject drift');
  assert.equal(support['se:object_ref']?.['@id'],R137_IDS.outcomeEvidence,'claim-support object drift');
  assert.equal(support['se:predicate']?.['@id'],'se:SUPPORTED_BY','claim-support predicate drift');
  assert.equal(support['se:lifecycle_status'],'PENDING_REVIEW','claim support must stay PENDING_REVIEW');
  // R140: the claim-support rationale is itself semantic evidence. Record
  // version and full reviewed content digest are pinned in validator CODE,
  // not inferred from a mutually repinned graph/sidecar or author expected.
  assert.equal(support['se:record_version'],'0.1.0','R140 STALE claim-support record version');
  assert.equal(support['se:synthetic'],false,'R140 claim-support assertion cannot become synthetic');
  assert.equal(support['se:rationale'],R140_REVIEWED_CLAIM_SUPPORT['se:rationale'],
    'R140 claim-support rationale semantic inversion');
  assert.equal(canonicalRecordDigest(support),R140_REVIEWED_CLAIM_SUPPORT_DIGEST,
    'R140 claim-support semantic/digest drift (fixed independent source contract)');


  // qualification boundaries
  assert.equal(scb['se:binding_scope'],'SELECTED_CQ_RESEARCH_CANDIDATE_ONLY_NOT_FROZEN_R8_CELL');
  assert.equal(scb['se:claim_lifecycle_effect'],'NONE_ORIGINAL_AND_CANDIDATE_CLAIM_REMAIN_PENDING');
  assert.equal(scb['se:professional_oracle_status'],'NOT_ATTESTED');
  assert.equal(scb['se:publication_permission'],'NOT_GRANTED_BY_THIS_OWNER_DECISION');

  // fan-out preservation: the shared SECTION locator keeps exactly its
  // original 8 evidence / 5 direct claims, and the new PAGE locator is
  // consumed by exactly one new evidence record.
  const evidenceNodes=candidateNodes.filter(n=>n['@type']==='se:Evidence');
  const bySectionLocator=evidenceNodes.filter(n=>n['se:source_locator_ref']?.['@id']===R137_IDS.sectionLocator).map(idOf).sort();
  assert.deepEqual(bySectionLocator,[...R137_SHARED_SECTION_EVIDENCE].sort(),'shared SECTION locator fan-out changed (collateral 8/5 damage)');
  const byOutcomeLocator=evidenceNodes.filter(n=>n['se:source_locator_ref']?.['@id']===R137_IDS.outcomeLocator).map(idOf);
  assert.deepEqual(byOutcomeLocator,[R137_IDS.outcomeEvidence],'p248 locator must be consumed by exactly the new outcome evidence');
  const directClaims=candidateNodes.filter(n=>n['@type']==='se:KnowledgeClaim')
    .filter(n=>refsOf(n,'se:evidence_refs').some(id=>R137_SHARED_SECTION_EVIDENCE.includes(id))).map(idOf).sort();
  assert.deepEqual(directClaims,[...R137_SHARED_SECTION_DIRECT_CLAIMS].sort(),'direct-claim fan-out over the shared section evidence changed');
  return scb;
}

export function readScbSchema(){
  return JSON.parse(readText(SCB_SCHEMA_PATH));
}
