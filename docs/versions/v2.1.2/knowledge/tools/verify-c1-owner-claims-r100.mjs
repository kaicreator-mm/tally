// Tally #398 R100: owner-disposition source sidecar cannot sign Claim lifecycle or Oracle.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';

const ORIGINAL='0f6007315fd7352fe6fd218effcfecf672cb4961';
const CANDIDATE='c3015c4178467fdcf87e39dfe18fa6a9b317c6b4';
const EXPECTED=[
  ['se:CLM-REQ-FUNCTIONAL-DOC-METHODS','se:EVD-REQ-FUNCTIONAL-REQ-METHODS','se:LOC-CS2023-SE-REQ-KU','se:SRC-CS2023','No universal single-method uniformity'],
  ['se:CLM-TVV-0001','se:EVD-TVV-0003','se:LOC-CS2023-SE-VALIDATION-KU','se:SRC-CS2023','Passing tests is not automatic customer/functional acceptance']
];
const lookup=(g,id)=>g['@graph'].find(x=>x['@id']===id);
const gitBlob=(b)=>createHash('sha1').update('blob '+b.length+'\0').update(b).digest('hex');

export function verify(d,f,c){
  assert.equal(d.schema,'tally.dke.c1-scoped-source-owner-claim-dispositions/1');
  assert.equal(d.version,'v2.1.2-candidate');
  assert.equal(d.status,'OWNER_SCOPED_DECISIONS_VERSIONED_NOT_CLAIM_LIFECYCLE');
  assert.equal(d.authority.sourceOwnerLogin,'kaicreator-mm');
  assert.equal(d.authority.decisionRecord,'https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6877334744');
  assert.equal(d.authority.delegationType,'PROJECT_OWNER_DELEGATED_EXECUTION_NOT_PERSONAL_LINE_BY_LINE_ATTESTATION');
  assert.equal(d.authority.publicationPermission,'NOT_GRANTED_BY_THIS_OWNER_DECISION');
  assert.equal(d.authority.independentProfessionalOracle,'NOT_ATTESTED');
  assert.equal(d.sources.originalGraphBlob,ORIGINAL);
  assert.equal(d.sources.candidateGraphBlob,CANDIDATE);
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
  assert.equal(c['@graph'].length,809);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:ReviewDecision').length,0);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:KnowledgeClaim').length,106);
  assert.equal(c['@graph'].filter(n=>n['@type']==='se:KnowledgeClaim'&&n['se:review_status']==='PENDING').length,106);
  assert.equal(d.claims.length,2);
  for(let i=0;i<2;i++){
    const row=d.claims[i],exp=EXPECTED[i];
    assert.deepEqual([row.claimId,row.evidenceId,row.locatorId,row.sourceId],exp.slice(0,4));
    assert.ok(row.negative.startsWith(exp[4]));
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
    assert.deepEqual(loc,originalLoc);
    assert.equal(claim['se:review_status'],'PENDING');
    assert.equal(claim['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(claim['se:SUPPORTED_BY']?.['@id']??claim['se:evidence_refs']?.['@id'],row.evidenceId);
    assert.equal(ev['se:source_locator_ref']?.['@id'],row.locatorId);
    assert.equal(ev['se:extraction_status'],'EXTRACTED_PENDING_REVIEW');
    assert.equal(ev['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(loc['se:source_ref']?.['@id'],row.sourceId);
    assert.equal(loc['se:verification_state'],'LOCATOR_VERIFIED');
  }
  const source=lookup(c,'se:SRC-CS2023');
  assert.equal(source?.['se:stable_identifier'],'DOI 10.1145/3664191');
  assert.ok(source['se:rights_handling'].includes('do not reproduce substantial report text'));
  return 'OWNER_SCOPED_METADATA_ONLY__CLAIM_LIFECYCLE_PENDING';
}
export function selfTest(d,f,c){
  const result=verify(d,f,c);
  const mutants=[
    x=>{x.d.claims[0].ownerDisposition='ORACLE_ACCEPTED'},
    x=>{x.d.claims[1].locatorId='se:FAKE'},
    x=>{x.d.authority.independentProfessionalOracle='ACCEPTED'},
    x=>{x.d.claims[0].negative='All methods must be uniformly applied'},
    x=>{x.d.qualification.formalSap='6/6'},
    x=>{x.d.claims[1].candidateReviewStatus='ACCEPTED'},
    x=>{x.d.authority.publicationPermission='GRANTED'},
    x=>{x.d.claims.push(structuredClone(x.d.claims[0]))},
    x=>{lookup(x.c,'se:CLM-TVV-0001')['se:review_status']='ACCEPTED'}
  ];
  for(const [i,fn] of mutants.entries()){
    const x={d:structuredClone(d),f:structuredClone(f),c:structuredClone(c)};
    fn(x);
    assert.throws(()=>verify(x.d,x.f,x.c),assert.AssertionError,'negative case '+i);
  }
  return {schema:'tally.dke.c1-r100-owner-adoption-contract-test/1',result,total:10,passed:10,
    positive:1,negative:9,originalGraphModified:false,professionalOracleAttested:false,formalSapQualified:false};
}
if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  assert.ok(process.argv.includes('--check'),'explicit --check required');
  const ob=readFileSync(new URL('../../../v2.1.1/knowledge/generated/graph.jsonld',import.meta.url));
  const cb=readFileSync(new URL('../generated/graph.jsonld',import.meta.url));
  assert.equal(gitBlob(ob),ORIGINAL);
  assert.equal(gitBlob(cb),CANDIDATE);
  const d=JSON.parse(readFileSync(new URL('../source/C1_CLAIM_OWNER_DISPOSITIONS_R100.json',import.meta.url),'utf8'));
  console.log(JSON.stringify({...selfTest(d,JSON.parse(ob),JSON.parse(cb)),node:process.version,
    platform:process.platform,originalBlob:ORIGINAL,candidateBlob:CANDIDATE},null,2));
}
