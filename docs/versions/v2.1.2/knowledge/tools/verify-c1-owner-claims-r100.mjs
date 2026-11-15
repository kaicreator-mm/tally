// Tally #398 R100: owner-disposition source sidecar cannot sign Claim lifecycle or Oracle.
// R137: + SelectedCqSourceBinding two-leg selected-CQ source consumption checks.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import {
  canonicalRecordDigest, assertC1SourceOwnerDispositions,
  readSourceOwnerInputs, SOURCE_OWNER_SCHEMA_PATH,
} from './source-owner-disposition.mjs';
import {
  readScbSchema, assertSelectedCqSourceBinding, R137_IDS,
  R137_SHARED_SECTION_EVIDENCE, R137_SHARED_SECTION_DIRECT_CLAIMS,
} from './selected-cq-source-binding.mjs';
export {canonicalRecordDigest};

const ORIGINAL='0f6007315fd7352fe6fd218effcfecf672cb4961';
const CANDIDATE='a49ddf58a0c4a5b25ebdd4018c8234f64e6365e0';
const GRAPH_BASE_PRODUCED_AT='b226ecaab32cbef427393157399a1bdc7cdaef16';
const SIDECAR='docs/versions/v2.1.2/knowledge/source/C1_CLAIM_OWNER_DISPOSITIONS_R100.json';
const REPO_ROOT=fileURLToPath(new URL('../../../../../',import.meta.url));
const EXPECTED=[
  {
    ids:['se:CLM-REQ-FUNCTIONAL-DOC-METHODS','se:EVD-REQ-FUNCTIONAL-REQ-METHODS','se:LOC-CS2023-SE-REQ-KU','se:SRC-CS2023'],
    negative:'No universal single-method uniformity',
    scope:'CS2023 SE-Requirements: Product Requirements, KA Core 1a-1c; curriculum teaching and bounded original paraphrase only',
    element:{taxonomyNodeRef:'se:NODE-CS2023-SE-PRODUCT-REQUIREMENTS',candidateElementRef:'dke:ev16.d02.se-requirements-child',positiveCq:'CS2023_REQUIREMENTS_TEACHING',unknownCq:'REQUIREMENTS_UNIFORM_METHOD',parentElementRef:'dke:ev16.d02.se-requirements-vv-child',parentStatus:'UNKNOWN',globalRefs:[],globalStatus:'UNASSIGNED_UNKNOWN',rootStatus:'UNKNOWN'}
  },
  {
    ids:['se:CLM-TVV-0001','se:EVD-TVV-0003','se:LOC-CS2023-SE-VALIDATION-KU','se:SRC-CS2023'],
    negative:'Passing tests is not automatic customer/functional acceptance',
    scope:'CS2023 SE-Validation, CS Core topic 1 and learning outcome 2; curriculum teaching and bounded original paraphrase only',
    element:{taxonomyNodeRef:'se:NODE-CS2023-SE-VALIDATION',candidateElementRef:'dke:ev16.d02.se-verification-child',positiveCq:'CS2023_VV_TEACHING',unknownCq:'VV_AUTOMATIC_BUSINESS_ACCEPTANCE',parentElementRef:'dke:ev16.d02.se-requirements-vv-child',parentStatus:'UNKNOWN',globalRefs:['se:REL-TVV-0020'],globalStatus:'UNKNOWN_PUBLISHER_MANDATE_EXCLUDED',rootStatus:'UNKNOWN'}
  }
];
const lookup=(g,id)=>g['@graph'].find(x=>x['@id']===id);
const gitBlob=(b)=>createHash('sha1').update('blob '+b.length+'\0').update(b).digest('hex');
const git=(args)=>execFileSync('git',args,{cwd:REPO_ROOT,encoding:'utf8',stdio:['ignore','pipe','pipe']}).trim();

function verifyRecordGap(d){
  assert.equal(d.sourceRecordGap.id,'TALLY-C1-R100-SCHEMA-01');
  assert.equal(d.sourceRecordGap.status,'RESOLVED_IN_V212_SOURCE_OWNER_DISPOSITION_CONTRACT');
  assert.ok(d.sourceRecordGap.reviewerRoleEnum.includes('not SOURCE_OWNER'));
  assert.ok(d.sourceRecordGap.reviewOutcomeEnum.includes('not ACCEPT_SCOPED_FOR_VERSIONED_CANDIDATE'));
  assert.ok(d.sourceRecordGap.acceptPromotionCoupling.includes('validate_knowledge_graph.py:423-431'));
  assert.ok(d.sourceRecordGap.minimalFutureExtension.includes('SourceOwnerDisposition'));
  assert.equal(d.sourceRecordGap.resolution.schema,'docs/versions/v2.1.2/knowledge/model/source-owner-disposition.schema.json');
  assert.equal(d.sourceRecordGap.resolution.validatorAndProducer,'docs/versions/v2.1.2/knowledge/tools/source-owner-disposition.mjs');
  assert.equal(d.sourceRecordGap.resolution.graphType,'se:SourceOwnerDisposition');
  assert.equal(d.sourceRecordGap.resolution.reviewDecisionCoupling,'NONE');
  assert.equal(d.sourceRecordGap.resolution.claimLifecyclePromotion,'FORBIDDEN');
}

export function verify(d,f,c,schema,overlayDocument=JSON.parse(readFileSync(new URL('../source/C1_OWNER_REVISION_R13.json',import.meta.url),'utf8'))){
  assert.equal(d.schema,'tally.dke.c1-scoped-source-owner-claim-dispositions/1');
  assert.equal(d.recordVersion,'0.2.1');
  assert.equal(d.version,'v2.1.2-candidate');
  assert.equal(d.status,'OWNER_SCOPED_DECISIONS_VERSIONED_NOT_CLAIM_LIFECYCLE');
  assert.equal(d.authority.sourceOwnerLogin,'kaicreator-mm');
  assert.equal(d.authority.decisionRecord,'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6877334744');
  assert.equal(d.authority.delegationType,'PROJECT_OWNER_DELEGATED_EXECUTION_NOT_PERSONAL_LINE_BY_LINE_ATTESTATION');
  assert.equal(d.authority.priorOd05AndGhDecision,'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6808543942');
  assert.equal(d.authority.independentEducationalReview,'https://github.com/kaicreator-mm/domain-forge/pull/208#pullrequestreview-5890546555');
  assert.equal(d.authority.publicationPermission,'NOT_GRANTED_BY_THIS_OWNER_DECISION');
  assert.equal(d.authority.independentProfessionalOracle,'NOT_ATTESTED');
  assert.equal(d.sources.originalRepoCommit,'3e94685ca57a4b7258988826697af1c1c3a6b3c8');
  assert.equal(d.sources.originalGraphBlob,ORIGINAL);
  assert.equal(d.sources.candidateGraphBlob,CANDIDATE);
  assert.equal(d.sources.candidateGraphBaseProducedAt,GRAPH_BASE_PRODUCED_AT);
  assert.equal(d.sources.candidateGraphCurrentRevisionPolicy,'ATTESTED_BY_NATIVE_GIT_HEAD_NOT_SELF_EMBEDDED');
  assert.equal(d.sources.recordedAtRevisionPolicy,'ATTESTED_BY_NATIVE_GIT_HEAD_NOT_SELF_EMBEDDED');
  assert.equal(d.sources.publisher,'Computer Science Curricula 2023, Final report, DOI 10.1145/3664191');
  assert.equal(d.sources.rightsScope,'Metadata/locator and concise original paraphrase only; no substantial text republication, LLM-training grant or independent publisher authorization');
  verifyRecordGap(d);
  assert.equal(d.qualification.sourceOwnerDecisionsRecorded,2);
  assert.equal(d.qualification.originalFrozenGraphModified,false);
  assert.equal(d.qualification.originalProducerClaimLifecycleAdopted,false);
  assert.equal(d.qualification.formalSap,'0/6');
  assert.equal(d.qualification.formalGlobal,'0/5');
  assert.equal(d.qualification.originalR8,'1/53_SCOPED_ONLY');
  assert.equal(d.qualification.parentRoot,'UNKNOWN');
  assert.equal(d.qualification.originalD04,'NOT_RUN');
  assert.equal(d.qualification.effectAuthorized,false);
  assert.equal(d.qualification.releaseAuthorized,false);
  assert.equal(f['@graph'].length,807);
  assert.equal(c['@graph'].length,815);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:SourceOwnerDisposition').length,2);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:SelectedCqSourceBinding').length,1);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:ReviewDecision').length,0);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:KnowledgeClaim').length,106);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:KnowledgeClaim'&&n['se:review_status']==='PENDING').length,106);
  assert.equal(c['@graph'].filter(n=>n['se:lifecycle_status']==='ACCEPTED').length,0);
  assert.equal(d.claims.length,2);
  for(let i=0;i<2;i++){
    const row=d.claims[i],exp=EXPECTED[i];
    assert.deepEqual([row.claimId,row.evidenceId,row.locatorId,row.sourceId],exp.ids);
    assert.equal(row.scope,exp.scope);
    assert.ok(row.negative.startsWith(exp.negative));
    assert.equal(row.ownerDisposition,'ACCEPT_SCOPED_FOR_VERSIONED_CANDIDATE');
    assert.equal(row.originalClaimLifecycle,'PENDING_REVIEW');
    assert.equal(row.candidateClaimLifecycle,'PENDING_REVIEW');
    assert.equal(row.originalReviewStatus,'PENDING');
    assert.equal(row.candidateReviewStatus,'PENDING');
    assert.equal(row.evidenceExtraction,'EXTRACTED_PENDING_REVIEW');
    assert.equal(row.semanticOracle,'NOT_ATTESTED');
    const originalClaim=lookup(f,row.claimId),claim=lookup(c,row.claimId);
    const originalEv=lookup(f,row.evidenceId),ev=lookup(c,row.evidenceId);
    const originalLoc=lookup(f,row.locatorId),loc=lookup(c,row.locatorId);
    assert.ok(claim&&originalClaim&&ev&&originalEv&&loc&&originalLoc);
    assert.deepEqual(claim,originalClaim);
    assert.deepEqual(ev,originalEv);
    assert.notDeepEqual(loc,originalLoc);
    const proof=row.locatorCorrection;
    const page=i===0?241:245, printed=i===0?242:246;
    assert.equal(proof.status,'VERSIONED_SOURCE_REPAIRED_NOT_FROZEN_REWRITTEN');
    assert.equal(proof.pdfZeroBasedPageIndex,page);
    assert.equal(proof.printedPage,printed);
    assert.equal(proof.originalFrozenLocatorVersion,'0.1.0');
    assert.equal(proof.candidateLocatorVersion,'0.1.1');
    assert.equal(proof.sourceDocument,'https://ieeecs-media.computer.org/media/education/reports/CS2023.pdf');
    assert.equal(loc['se:record_version'],'0.1.1');
    assert.equal(loc['se:accessed_on'],'2026-11-14');
    assert.ok(loc['se:locator'].includes('verified PDF zero-based index '+page+', printed page '+printed));
    assert.ok(originalLoc['se:locator'].includes('PDF page indices 174-176'));
    assert.deepEqual(Object.keys(loc).filter(k=>JSON.stringify(loc[k])!==JSON.stringify(originalLoc[k])).sort(),['se:accessed_on','se:locator','se:record_version']);
    assert.equal(row.recordBindings.candidateLocator.recordVersion,'0.1.1');
    assert.equal(canonicalRecordDigest(loc),row.recordBindings.candidateLocator.digestSha256,'STALE candidate Locator '+row.locatorId);
    assert.equal(claim['se:review_status'],'PENDING');
    assert.equal(claim['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(claim['se:SUPPORTED_BY']?.['@id']??claim['se:evidence_refs']?.['@id'],row.evidenceId);
    assert.equal(ev['se:source_locator_ref']?.['@id'],row.locatorId);
    assert.equal(ev['se:extraction_status'],'EXTRACTED_PENDING_REVIEW');
    assert.equal(ev['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(loc['se:source_ref']?.['@id'],row.sourceId);
    assert.equal(loc['se:verification_state'],'LOCATOR_VERIFIED');
    for(const [kind,node] of [['claim',claim],['evidence',ev],['locator',loc]]){
      const binding=row.recordBindings[kind];
      assert.equal(binding.recordVersion,'0.1.0');
      const bound=kind==='locator'?originalLoc:node;
      assert.equal(bound['se:record_version'],binding.recordVersion,'STALE original record_version '+row[kind+'Id']);
      assert.equal(canonicalRecordDigest(bound),binding.digestSha256,'STALE original record digest '+row[kind+'Id']);
    }
    const {selectedCqSourceBinding,...elementCore}=row.elementAttribution;
    if(i===0)assert.equal(selectedCqSourceBinding,undefined,'REQ row must not carry a SelectedCqSourceBinding');
    assert.deepEqual(elementCore,exp.element);
    assert.equal(claim['se:APPLIES_TO']?.['@id'],row.elementAttribution.taxonomyNodeRef);
  }
  // R137: the selected VV CQ consumes BOTH source legs executably
  const scbSchema=readScbSchema();
  const scb=assertSelectedCqSourceBinding(d,scbSchema,c['@graph'],overlayDocument);
  assert.equal(scb['se:positive_cq'],'CS2023_VV_TEACHING');
  assert.equal(scb['se:section_pdf_zero_based_page'],'245');
  assert.equal(scb['se:section_printed_page'],'246');
  assert.equal(scb['se:outcome_pdf_zero_based_page'],'247');
  assert.equal(scb['se:outcome_printed_page'],'248');
  assert.equal(scb['se:binding_scope'],'SELECTED_CQ_RESEARCH_CANDIDATE_ONLY_NOT_FROZEN_R8_CELL');
  const scbRow=d.claims[1].elementAttribution.selectedCqSourceBinding;
  assert.deepEqual([scbRow.id,scbRow.recordVersion,scbRow.lifecycle],[R137_IDS.scb,'0.1.0','ACTIVE']);
  assert.equal(scbRow.claimDigestSha256,d.claims[1].recordBindings.claim.digestSha256);
  assert.equal(scbRow.sectionLeg.digestSha256,d.claims[1].recordBindings.candidateLocator.digestSha256);
  assert.ok(scbRow.outcomeLeg.locatorRef!==d.claims[1].locatorId,'two legs must remain distinct locators');
  const source=lookup(c,'se:SRC-CS2023');
  assert.equal(source?.['se:stable_identifier'],'DOI 10.1145/3664191');
  assert.equal(source?.['se:record_version'],'0.1.0');
  assert.ok(source['se:rights_handling'].includes('do not reproduce substantial report text'));
  assertC1SourceOwnerDispositions(d,schema,f['@graph'],c['@graph']);
  return 'OWNER_SCOPED_METADATA_ONLY__CLAIM_LIFECYCLE_PENDING';
}

export function selfTest(d,f,c,schema){
  const result=verify(d,f,c,schema);
  const mutants=[
    {kind:'negative',fn:x=>{x.d.claims[0].ownerDisposition='ORACLE_ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[1].locatorId='se:FAKE'}},
    {kind:'negative',fn:x=>{x.d.authority.independentProfessionalOracle='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[0].negative='All methods must be uniformly applied'}},
    {kind:'negative',fn:x=>{x.d.qualification.formalSap='6/6'}},
    {kind:'negative',fn:x=>{x.d.claims[1].candidateReviewStatus='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.authority.publicationPermission='GRANTED'}},
    {kind:'negative',fn:x=>{x.d.claims.push(structuredClone(x.d.claims[0]))}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:CLM-TVV-0001')['se:review_status']='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.authority.independentEducationalReview='https://invalid.example/review'}},
    {kind:'negative',fn:x=>{x.d.sources.rightsScope='unrestricted redistribution and training'}},
    {kind:'negative',fn:x=>{x.d.sourceRecordGap.status='OPEN_PENDING_RETAINED_OUT_OF_GRAPH_SIDECAR'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SOD-CS2023-REQ-R1')['se:source_owner_login']='forged-owner'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SOD-CS2023-VV-R1')['se:professional_oracle_status']='ATTESTED_BY_DISTINCT_SEMANTIC_REVIEW'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SOD-CS2023-REQ-R1')['se:rights_scope']='unrestricted redistribution and training'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SOD-CS2023-VV-R1')['se:claim_lifecycle_effect']='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[0].dispositionLifecycle='SUPERSEDED'}},
    {kind:'stale',fn:x=>{lookup(x.c,'se:SOD-CS2023-REQ-R1')['se:candidate_locator_digest_sha256']='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].dispositionRecordVersion='0.0.9'}},
    {kind:'unknown',fn:x=>{x.d.qualification.parentRoot='QUALIFIED'}},
    {kind:'unknown',fn:x=>{x.d.qualification.originalD04='PASS'}},
    {kind:'unknown',fn:x=>{x.d.claims[1].elementAttribution.globalStatus='QUALIFIED'}},
    {kind:'stale',fn:x=>{x.d.claims[0].recordBindings.claim.digestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].recordBindings.evidence.recordVersion='0.1.1'}},
    {kind:'stale',fn:x=>{x.d.sources.candidateGraphBaseProducedAt='862ae64be30cd9f3a54f6fa44db4e1adca05ba73'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:LOC-CS2023-SE-REQ-KU')['se:locator']=lookup(x.f,'se:LOC-CS2023-SE-REQ-KU')['se:locator']}},
    {kind:'negative',fn:x=>{x.d.claims[1].locatorCorrection.pdfZeroBasedPageIndex=174}},
    {kind:'stale',fn:x=>{x.d.claims[0].recordBindings.candidateLocator.digestSha256='0'.repeat(64)}},
    // R137 selected-CQ two-leg binding mutants
    {kind:'negative',fn:x=>{x.c['@graph']=x.c['@graph'].filter(n=>n['@id']!=='se:REL-TVV-0024')}},
    {kind:'negative',fn:x=>{x.c['@graph']=x.c['@graph'].filter(n=>n['@id']!=='se:EVD-TVV-0016')}},
    {kind:'negative',fn:x=>{x.c['@graph']=x.c['@graph'].filter(n=>n['@id']!=='se:LOC-CS2023-SE-VALIDATION-LO2')}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SCB-CS2023-VV-R1')['se:binding_scope']='FROZEN_R8_CELL_QUALIFIED'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:EVD-TVV-0004')['se:source_locator_ref']={'@id':'se:LOC-CS2023-SE-VALIDATION-LO2'}}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SCB-CS2023-VV-R1')['se:outcome_pdf_zero_based_page']='245'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:LOC-CS2023-SE-VALIDATION-KU')['se:locator']=lookup(x.c,'se:LOC-CS2023-SE-VALIDATION-KU')['se:locator'].replace('index 245','index 247')}},
    {kind:'stale',fn:x=>{lookup(x.c,'se:SCB-CS2023-VV-R1')['se:outcome_locator_digest_sha256']='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].elementAttribution.selectedCqSourceBinding.outcomeLeg.digestSha256='0'.repeat(64)}},
    {kind:'stale',fn:x=>{x.d.claims[1].elementAttribution.selectedCqSourceBinding.outcomeLeg.recordVersion='0.0.9'}},
    {kind:'negative',fn:x=>{x.d.claims[1].elementAttribution.selectedCqSourceBinding.claimDigestSha256='0'.repeat(64)}},
    {kind:'unknown',fn:x=>{lookup(x.c,'se:SCB-CS2023-VV-R1')['se:professional_oracle_status']='ATTESTED_BY_DISTINCT_SEMANTIC_REVIEW'}},
    {kind:'negative',fn:x=>{lookup(x.c,'se:SCB-CS2023-VV-R1')['se:claim_lifecycle_effect']='ACCEPTED'}},
    {kind:'negative',fn:x=>{x.d.claims[1].elementAttribution.selectedCqSourceBinding.bindingScope='FROZEN_R8_CELL_QUALIFIED'}},
  ];
  for(const [i,mutant] of mutants.entries()){
    const x={d:structuredClone(d),f:structuredClone(f),c:structuredClone(c)};
    mutant.fn(x);
    assert.throws(()=>verify(x.d,x.f,x.c,schema),assert.AssertionError,mutant.kind+' case '+i);
  }
  const count=(kind)=>mutants.filter(x=>x.kind===kind).length;
  return {schema:'tally.dke.c1-r100-owner-adoption-contract-test/1',result,total:mutants.length+1,passed:mutants.length+1,
    positive:1,negative:count('negative'),unknown:count('unknown'),stale:count('stale'),sourceOwnerDispositions:2,selectedCqSourceBinding:1,originalGraphModified:false,professionalOracleAttested:false,formalSapQualified:false};
}

function verifySchemaGap(schemaText,validatorText){
  assert.ok(schemaText.includes('HUMAN_SOURCE_REVIEWER: {}'));
  assert.ok(!schemaText.includes('SOURCE_OWNER: {}'),'schema drift: SOURCE_OWNER now requires gap re-evaluation');
  assert.ok(!schemaText.includes('ACCEPT_SCOPED_FOR_VERSIONED_CANDIDATE: {}'),'schema drift: scoped outcome now requires gap re-evaluation');
  assert.ok(validatorText.includes('REVIEW_ACCEPT_TARGET_NOT_ACCEPTED'));
}

if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  assert.ok(process.argv.includes('--check'),'explicit --check required');
  const ob=readFileSync(new URL('../../../v2.1.1/knowledge/generated/graph.jsonld',import.meta.url));
  const cb=readFileSync(new URL('../generated/graph.jsonld',import.meta.url));
  const db=readFileSync(new URL('../source/C1_CLAIM_OWNER_DISPOSITIONS_R100.json',import.meta.url));
  const sb=readFileSync(new URL('../model/source-owner-disposition.schema.json',import.meta.url));
  const scbSchemaBytes=readFileSync(new URL('../model/selected-cq-source-binding.schema.json',import.meta.url));
  const ownerInputs=readSourceOwnerInputs();
  assert.equal(gitBlob(ob),ORIGINAL);
  assert.equal(gitBlob(cb),CANDIDATE);
  assert.equal(gitBlob(db),git(['rev-parse','HEAD:'+SIDECAR]),'sidecar bytes differ from exact HEAD');
  assert.equal(gitBlob(sb),git(['rev-parse','HEAD:'+SOURCE_OWNER_SCHEMA_PATH]),'SourceOwnerDisposition schema bytes differ from exact HEAD');
  assert.equal(gitBlob(scbSchemaBytes),git(['rev-parse','HEAD:docs/versions/v2.1.2/knowledge/model/selected-cq-source-binding.schema.json']),'SelectedCqSourceBinding schema bytes differ from exact HEAD');
  assert.equal(gitBlob(readFileSync(new URL('./selected-cq-source-binding.mjs',import.meta.url))),git(['rev-parse','HEAD:docs/versions/v2.1.2/knowledge/tools/selected-cq-source-binding.mjs']),'SCB validator bytes differ from exact HEAD');
  const schemaText=readFileSync(new URL('../../../v2.1.1/knowledge/model/schema/software-engineering.yaml',import.meta.url),'utf8');
  const validatorText=readFileSync(new URL('../../../v2.1.1/knowledge/model/validate_knowledge_graph.py',import.meta.url),'utf8');
  verifySchemaGap(schemaText,validatorText);
  const d=JSON.parse(db.toString('utf8'));
  console.log(JSON.stringify({...selfTest(d,JSON.parse(ob),JSON.parse(cb),ownerInputs.schema),node:process.version,
    platform:process.platform,recordedAtRevision:git(['rev-parse','HEAD']),tree:git(['rev-parse','HEAD^{tree}']),sidecarBlob:gitBlob(db),sourceOwnerDispositionSchemaBlob:gitBlob(sb),selectedCqSourceBindingSchemaBlob:gitBlob(scbSchemaBytes),originalBlob:ORIGINAL,candidateBlob:CANDIDATE},null,2));
}
