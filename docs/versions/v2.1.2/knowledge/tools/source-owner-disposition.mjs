// Tally v2.1.2 SourceOwnerDisposition producer + native schema/graph validator.
// This is deliberately separate from ReviewDecision: scoped Owner adoption never
// promotes a KnowledgeClaim or substitutes for a distinct semantic Oracle.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {nodesDeepEqual, projectRecord} from './rdf-min.mjs';

export const REPO_ROOT=fileURLToPath(new URL('../../../../../',import.meta.url));
export const SOURCE_OWNER_PATH='docs/versions/v2.1.2/knowledge/source/C1_CLAIM_OWNER_DISPOSITIONS_R100.json';
export const SOURCE_OWNER_SCHEMA_PATH='docs/versions/v2.1.2/knowledge/model/source-owner-disposition.schema.json';
export const SOURCE_OWNER_BLOB='0ec426e5a71d8291557932141e6d41c3c12f7b66';
export const SOURCE_OWNER_SCHEMA_BLOB='78cb22c57eb8b69e0a03891a2e4c90e29ffca5f6';

const canonical=value=>Array.isArray(value)?value.map(canonical):
  value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(key=>[key,canonical(value[key])])):value;
export const canonicalRecordDigest=record=>createHash('sha256').update(JSON.stringify(canonical(record)),'utf8').digest('hex');
export const gitBlob=bytes=>createHash('sha1').update(`blob ${bytes.length}\0`).update(bytes).digest('hex');

function readPinned(path,expected){
  const bytes=readFileSync(REPO_ROOT+path);
  assert.equal(gitBlob(bytes),expected,`SourceOwnerDisposition input Git Blob drift: ${path}`);
  return bytes;
}

export function readSourceOwnerInputs(){
  const sourceBytes=readPinned(SOURCE_OWNER_PATH,SOURCE_OWNER_BLOB);
  const schemaBytes=readPinned(SOURCE_OWNER_SCHEMA_PATH,SOURCE_OWNER_SCHEMA_BLOB);
  return {
    document:JSON.parse(sourceBytes.toString('utf8')),
    schema:JSON.parse(schemaBytes.toString('utf8')),
    sourceBlob:gitBlob(sourceBytes),
    schemaBlob:gitBlob(schemaBytes),
  };
}

function resolveRef(root,ref){
  assert.ok(ref.startsWith('#/'),`SourceOwnerDisposition schema: only local refs allowed: ${ref}`);
  return ref.slice(2).split('/').reduce((value,key)=>value[key.replaceAll('~1','/').replaceAll('~0','~')],root);
}

// Native validator for the small JSON-Schema profile used by the extension.
// Supporting the actual schema here avoids an npm/AJV runtime dependency.
export function validateSchema(value,schema,root=schema,path='$'){
  if(schema.$ref)return validateSchema(value,resolveRef(root,schema.$ref),root,path);
  if('const' in schema)assert.deepEqual(value,schema.const,`${path}: const mismatch`);
  if(schema.enum)assert.ok(schema.enum.includes(value),`${path}: enum mismatch`);
  if(schema.type==='object'){
    assert.ok(value&&typeof value==='object'&&!Array.isArray(value),`${path}: object required`);
    for(const key of schema.required??[])assert.ok(Object.hasOwn(value,key),`${path}: missing ${key}`);
    if(schema.additionalProperties===false){
      for(const key of Object.keys(value))assert.ok(Object.hasOwn(schema.properties??{},key),`${path}: additional property ${key}`);
    }
    for(const [key,child] of Object.entries(schema.properties??{})){
      if(Object.hasOwn(value,key))validateSchema(value[key],child,root,`${path}.${key}`);
    }
  }else if(schema.type==='string'){
    assert.equal(typeof value,'string',`${path}: string required`);
    if(schema.minLength!==undefined)assert.ok(value.length>=schema.minLength,`${path}: empty/short string`);
    if(schema.pattern)assert.ok(new RegExp(schema.pattern).test(value),`${path}: pattern mismatch`);
  }
}

export function materializeSourceOwnerDispositions(document,schema){
  assert.equal(document.schema,'tally.dke.c1-scoped-source-owner-claim-dispositions/1');
  assert.ok(Array.isArray(document.claims));
  return document.claims.map(row=>{
    const bindings=row.recordBindings;
    const record={
      record_version:row.dispositionRecordVersion,
      lifecycle_status:row.dispositionLifecycle,
      created_on:document.recordedOn,
      claim_ref:row.claimId,
      claim_record_version:bindings.claim.recordVersion,
      claim_digest_sha256:bindings.claim.digestSha256,
      evidence_ref:row.evidenceId,
      evidence_record_version:bindings.evidence.recordVersion,
      evidence_digest_sha256:bindings.evidence.digestSha256,
      original_locator_ref:row.locatorId,
      original_locator_record_version:bindings.locator.recordVersion,
      original_locator_digest_sha256:bindings.locator.digestSha256,
      candidate_locator_ref:row.locatorId,
      candidate_locator_record_version:bindings.candidateLocator.recordVersion,
      candidate_locator_digest_sha256:bindings.candidateLocator.digestSha256,
      source_ref:row.sourceId,
      source_owner_login:document.authority.sourceOwnerLogin,
      source_owner_ref:`https://github.com/${document.authority.sourceOwnerLogin}`,
      source_owner_authority_ref:document.authority.decisionRecord,
      source_owner_authority_type:document.authority.delegationType,
      disposition:row.ownerDisposition,
      scope:row.scope,
      positive:row.positive,
      negative:row.negative,
      unknown:row.unknown,
      rights_scope:document.sources.rightsScope,
      publication_permission:document.authority.publicationPermission,
      professional_oracle_status:row.semanticOracle,
      independent_educational_review_ref:document.authority.independentEducationalReview,
      taxonomy_node_ref:row.elementAttribution.taxonomyNodeRef,
      candidate_element_attribution:row.elementAttribution.candidateElementRef,
      candidate_element_status:'CANDIDATE_ATTRIBUTION_ONLY_NOT_SAP_GLOBAL_ROOT_R8',
      claim_lifecycle_effect:'NONE_ORIGINAL_AND_CANDIDATE_CLAIM_REMAIN_PENDING',
    };
    if(row.supersedesDispositionId)record.supersedes_ref=row.supersedesDispositionId;
    if(row.professionalOracleRef)record.professional_oracle_ref=row.professionalOracleRef;
    const node=projectRecord(row.dispositionId,'se:SourceOwnerDisposition',record);
    validateSchema(node,schema);
    if(node['se:professional_oracle_status']==='NOT_ATTESTED'){
      assert.ok(!node['se:professional_oracle_ref'],'NOT_ATTESTED Oracle must not carry an attestation ref');
    }else{
      assert.ok(node['se:professional_oracle_ref'],'attested Oracle requires a distinct review ref');
    }
    return node;
  });
}

const lookup=(nodes,id,label)=>{
  const node=nodes.find(candidate=>candidate['@id']===id);
  assert.ok(node,`SourceOwnerDisposition: missing ${label} ${id}`);
  return node;
};
const idOf=node=>node['@id'];

function assertExactBinding(node,prefix,target){
  assert.equal(node[`se:${prefix}_record_version`],target['se:record_version'],`STALE ${prefix} record_version ${idOf(node)}`);
  assert.equal(node[`se:${prefix}_digest_sha256`],canonicalRecordDigest(target),`STALE ${prefix} digest ${idOf(node)}`);
}

function compareVersion(left,right){
  const a=left.split('.').map(Number),b=right.split('.').map(Number);
  for(let i=0;i<3;i++)if(a[i]!==b[i])return a[i]-b[i];
  return 0;
}

export function assertSourceOwnerDispositionEvolution(nodes){
  const byId=new Map(nodes.map(node=>[idOf(node),node]));
  assert.equal(byId.size,nodes.length,'duplicate SourceOwnerDisposition id');
  const activeByClaim=new Map();
  for(const node of nodes){
    if(node['se:lifecycle_status']!=='ACTIVE')continue;
    const claimId=node['se:claim_ref']['@id'];
    assert.ok(!activeByClaim.has(claimId),`multiple ACTIVE dispositions for ${claimId}`);
    activeByClaim.set(claimId,node);
  }
  for(const node of nodes){
    const supersedes=node['se:supersedes_ref']?.['@id'];
    if(!supersedes)continue;
    const previous=byId.get(supersedes);
    assert.ok(previous,`dangling supersedes_ref ${supersedes}`);
    assert.notEqual(previous['se:lifecycle_status'],'ACTIVE','superseded disposition cannot remain ACTIVE');
    assert.ok(compareVersion(node['se:record_version'],previous['se:record_version'])>0,'superseding record_version must increase');
  }
}

export function assertC1SourceOwnerDispositions(document,schema,frozenNodes,candidateNodes){
  const expected=materializeSourceOwnerDispositions(document,schema);
  assert.equal(expected.length,2,'C1 requires exactly two independent Owner dispositions');
  const actual=candidateNodes.filter(node=>node['@type']==='se:SourceOwnerDisposition');
  assert.equal(actual.length,2,'C1 graph must contain exactly two SourceOwnerDisposition records');
  for(const node of actual)validateSchema(node,schema);
  const byId=new Map(actual.map(node=>[idOf(node),node]));
  for(const wanted of expected){
    const node=byId.get(idOf(wanted));
    assert.ok(node,`missing disposition ${idOf(wanted)}`);
    assert.ok(nodesDeepEqual(node,wanted),`disposition content/source drift ${idOf(wanted)}`);
    assert.equal(node['se:lifecycle_status'],'ACTIVE','current C1 dispositions must be ACTIVE');
    const claimId=node['se:claim_ref']['@id'];

    const frozenClaim=lookup(frozenNodes,claimId,'original Claim');
    const claim=lookup(candidateNodes,claimId,'candidate Claim');
    const evidenceId=node['se:evidence_ref']['@id'];
    const frozenEvidence=lookup(frozenNodes,evidenceId,'original Evidence');
    const evidence=lookup(candidateNodes,evidenceId,'candidate Evidence');
    const locatorId=node['se:original_locator_ref']['@id'];
    const frozenLocator=lookup(frozenNodes,locatorId,'original Locator');
    const locator=lookup(candidateNodes,locatorId,'candidate Locator');
    assert.equal(node['se:candidate_locator_ref']['@id'],locatorId);
    assert.ok(nodesDeepEqual(claim,frozenClaim),'Owner disposition must not mutate original Claim');
    assert.ok(nodesDeepEqual(evidence,frozenEvidence),'Owner disposition must not mutate original Evidence');
    assertExactBinding(node,'claim',frozenClaim);
    assertExactBinding(node,'evidence',frozenEvidence);
    assertExactBinding(node,'original_locator',frozenLocator);
    assertExactBinding(node,'candidate_locator',locator);
    assert.equal(claim['se:review_status'],'PENDING','Owner disposition cannot accept Claim review_status');
    assert.equal(claim['se:lifecycle_status'],'PENDING_REVIEW','Owner disposition cannot accept Claim lifecycle');
    assert.equal(evidence['se:extraction_status'],'EXTRACTED_PENDING_REVIEW');
    assert.equal(evidence['se:lifecycle_status'],'PENDING_REVIEW');
    assert.equal(evidence['se:source_locator_ref']['@id'],locatorId);
    assert.equal(locator['se:source_ref']['@id'],node['se:source_ref']['@id']);
    assert.equal(claim['se:APPLIES_TO']['@id'],node['se:taxonomy_node_ref']['@id']);
    const source=lookup(candidateNodes,node['se:source_ref']['@id'],'Source');
    assert.ok(source['se:rights_handling'].includes('do not reproduce substantial report text'),'CS2023 rights boundary drift');
  }
  assertSourceOwnerDispositionEvolution(actual);
  return actual;
}
