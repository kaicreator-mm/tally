// project-c1.mjs — canonical knowledge-lane projection for the Tally v2.1.2
// C1 owner-source candidate (native Node, zero dependencies).
//
// Replicates the documented v2.1.1 canonical projection semantics
// (docs/versions/v2.1.1/knowledge/generated/README.md) from the frozen
// v2.1.1 source lane inputs, then applies the Owner-authorized C1 overlay
// (OD05=NARROW_CLAIM / GH-2018=ACCEPT_PATCH, Tally #398 R13) post-projection.
//
// Byte-fidelity gates (enforced by build-c1-canonical.mjs / .test.mjs):
//   - without the overlay, the projection reproduces the frozen
//     docs/versions/v2.1.1/knowledge/generated/{graph.jsonld,graph.ttl}
//     byte-for-byte and the canonical N-Triples digest 2625a6b3...
//   - with the overlay, the projection is cross-checked against the
//     Owner-overlay candidate (c1-candidate.graph.jsonld, regenerated).
//
// Frozen inputs are pinned by Git blob SHA-1; any drift fails closed
// ("source invalidation" negatives cover deleted/altered pins).

import { readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { parseYaml } from './yaml-min.mjs';
import {
  nodesToTriples, canonicalDigest, projectRecord, serializeJsonld,
  serializeTtl, parseTtl, triplesEqualSet, sortNodes, nodesDeepEqual,
} from './rdf-min.mjs';

export const REPO_ROOT = fileURLToPath(new URL('../../../../../', import.meta.url));
const V211 = 'docs/versions/v2.1.1';
const V212 = 'docs/versions/v2.1.2';

export const PATHS = {
  lanes: (name) => `${V211}/knowledge/vertical-slice/lanes/${name}/records.yaml`,
  registry: `${V211}/authority/SOURCE_REGISTRY.yaml`,
  catalog: `${V211}/knowledge/model/RELATIONSHIP_CATALOG.yaml`,
  frozenGraphJsonld: `${V211}/knowledge/generated/graph.jsonld`,
  frozenGraphTtl: `${V211}/knowledge/generated/graph.ttl`,
  frozenManifest: `${V211}/knowledge/generated/manifest.sha256`,
  overlay: `${V212}/knowledge/source/C1_OWNER_REVISION_R13.json`,
  outGraphJsonld: `${V212}/knowledge/generated/graph.jsonld`,
  outGraphTtl: `${V212}/knowledge/generated/graph.ttl`,
  outManifest: `${V212}/knowledge/generated/manifest.sha256`,
  outPreview: `${V212}/knowledge/generated/c1-candidate.graph.jsonld`,
};

export const LANE_ORDER = [
  'architecture-design', 'build-delivery', 'foundations-processes',
  'implementation-code', 'maintenance-evolution', 'planning-risk-quality',
  'reliability-performance', 'requirements', 'security-engineering',
  'testing-verification',
];

// Git blob SHA-1 pins of the frozen inputs (recorded 2026-11-12 at PR #399
// base wip/v2.1.1-knowledge, head a184b630fd9512007aa195f7d9fbe52e21662821).
export const PINS = {
  lanes: {
    'architecture-design': '4a3f0c62370b74131281a9f7577ba82c03d55ed5',
    'build-delivery': 'e747f73cec202ecfd277529e07297a55835e7c3b',
    'foundations-processes': '650372df06c7ae7b23dd4b8eb3ab95033ada978e',
    'implementation-code': '73520f6d118ef5d55534409d7a66370ea51197cf',
    'maintenance-evolution': 'bfc7bccb518ae7f8ca9e3de537c05b4e31f6814e',
    'planning-risk-quality': '532eddfc852d72b46d03183cdd20181ddb0cfe4f',
    'reliability-performance': 'a6b12c6b0dcdbce4331438ff3d148ac0dc43d55f',
    requirements: 'fb14f92821497ec9c6d793d465d5f2d20dcbfe1b',
    'security-engineering': '31ab19c6f3506e6ddc1f6276cb1a6e62a66ed6fb',
    'testing-verification': 'dd9ff5cebb4c17c1ec8d5582fdb445e377e7f484',
  },
  registry: '1ce6d3da3594959b6cee7fdc7d03778cef60ad78',
  catalog: 'a0f088d11878b27b600e604f2cc8d792a85cca9d',
  frozenGraphJsonld: '0f6007315fd7352fe6fd218effcfecf672cb4961',
  overlay: 'd790057091c3befac60a27c2f16a08faa53f5fa1',
};

export const ORIGINAL_COMMIT = '3e94685ca57a4b7258988826697af1c1c3a6b3c8';
export const FROZEN_DIGEST = '2625a6b336a8a0b392e7eccd13f157e3fb9030e9262836a12ec8afc85eb2d2e8';
export const BUILD_DATE = '2026-11-12';
export const GRAPH_ID_V211 = 'se:GRAPH-V2.1.1-SEK08';

const COLLECTION_TYPES = {
  domains: 'se:Domain',
  subdomains: 'se:Subdomain',
  taxonomy_views: 'se:TaxonomyView',
  taxonomy_nodes: 'se:TaxonomyNode',
  concepts: 'se:Concept',
  principles: 'se:Principle',
  practices: 'se:Practice',
  patterns: 'se:Pattern',
  anti_patterns: 'se:AntiPattern',
  decision_models: 'se:DecisionModel',
  knowledge_claims: 'se:KnowledgeClaim',
  evidence_records: 'se:Evidence',
  source_locators: 'se:SourceLocator',
  coverage_gaps: 'se:CoverageGap',
  cases: 'se:Case',
  relationship_assertions: 'se:RelationshipAssertion',
};

// Projection-only normalization (documented v2.1.1 defect, frozen in place):
// the shared Domain boundary_notes[1] was cut by YAML comment parsing in the
// reviewed raw lane; the projection restores the verbatim raw sequence-entry
// text. Raw carrier: docs/versions/v2.1.1/knowledge/vertical-slice/lanes/
// requirements/records.yaml line 42 (lane SEK-06A, first canonical carrier).
const BOUNDARY_NOTE_RAW = 'SWEBOK v4.0a identity and registry reconciliation is complete (SEK-02R #325 merged via PR #326); SWEBOK content is cited only where a lane has directly reinspected the located sections, and such records stay PROPOSED/PENDING_REVIEW until human review.';
const BOUNDARY_NOTE_PARSED = 'SWEBOK v4.0a identity and registry reconciliation is complete (SEK-02R';

// ---------------------------------------------------------------------------
// input loading + pin verification

export function gitBlobSha1(bytes) {
  return createHash('sha1').update(`blob ${bytes.length}\0`).update(bytes).digest('hex');
}

function readPinned(relPath, expectedBlob, what) {
  const full = REPO_ROOT + relPath;
  let bytes;
  try {
    bytes = readFileSync(full);
  } catch (e) {
    throw new Error(`project-c1: ${what} unavailable at ${relPath}: ${e.code ?? e.message} (source failure must fail closed)`);
  }
  const actual = gitBlobSha1(bytes);
  if (expectedBlob !== null && actual !== expectedBlob) {
    throw new Error(`project-c1: ${what} Git Blob drift at ${relPath}: expected ${expectedBlob}, got ${actual} (source failure must fail closed)`);
  }
  return { bytes, text: bytes.toString('utf8'), blob: actual };
}

// test-facing wrappers (source-failure negatives exercise exactly this layer)
export const readPinnedInput = readPinned;
export function assertPinMatches(bytes, expectedBlob, what) {
  const actual = gitBlobSha1(bytes);
  if (actual !== expectedBlob) {
    throw new Error(`project-c1: ${what} Git Blob drift: expected ${expectedBlob}, got ${actual} (source failure must fail closed)`);
  }
  return actual;
}

export function loadInputs() {
  const lanes = {};
  for (const name of LANE_ORDER) {
    const lane = readPinned(PATHS.lanes(name), PINS.lanes[name], `lane ${name}`);
    lanes[name] = parseYaml(lane.text, `lane/${name}`);
  }
  const registry = parseYaml(readPinned(PATHS.registry, PINS.registry, 'SOURCE_REGISTRY').text, 'SOURCE_REGISTRY');
  const catalog = parseYaml(readPinned(PATHS.catalog, PINS.catalog, 'RELATIONSHIP_CATALOG').text, 'RELATIONSHIP_CATALOG');
  const frozenGraphJsonld = readPinned(PATHS.frozenGraphJsonld, PINS.frozenGraphJsonld, 'frozen v2.1.1 graph.jsonld');
  const overlayBytes = readPinned(PATHS.overlay, PINS.overlay, 'C1 owner overlay R13');
  const overlay = JSON.parse(overlayBytes.text);
  return { lanes, registry, catalog, overlay, frozenGraphJsonld };
}

// ---------------------------------------------------------------------------
// record assembly

export function assembleRecords(lanes) {
  const byId = new Map();
  const stats = { raw: 0, shared: 0 };
  for (const name of LANE_ORDER) {
    const lane = lanes[name];
    const collections = lane.collections;
    if (!collections) throw new Error(`project-c1: lane ${name} has no collections`);
    for (const [coll, records] of Object.entries(collections)) {
      const typeName = COLLECTION_TYPES[coll];
      if (!typeName) throw new Error(`project-c1: unknown collection "${coll}" in lane ${name}`);
      for (const rec of records) {
        stats.raw += 1;
        const id = rec.id;
        if (typeof id !== 'string' || !id.startsWith('se:')) {
          throw new Error(`project-c1: record without se: id in lane ${name} collection ${coll}`);
        }
        if (byId.has(id)) {
          const prev = byId.get(id);
          if (!nodesDeepEqual(prev.record, rec)) {
            throw new Error(`project-c1: shared record ${id} differs between lanes (parsed-identical rule)`);
          }
          stats.shared += 1;
          continue;
        }
        byId.set(id, { record: rec, type: typeName, collection: coll, lane: name });
      }
    }
  }
  return { byId, stats };
}

// Derived Source records (six referenced registry entries; metadata-only
// entries, e.g. ISO, are not projected). Field derivation per the v2.1.1
// generated README.
export function deriveSources(registry, byId) {
  const referenced = new Set();
  for (const { record } of byId.values()) {
    if (typeof record.source_ref === 'string') referenced.add(record.source_ref);
  }
  const out = [];
  for (const entry of registry.sources) {
    const id = `se:${entry.source_id.replace(/^SE-/, '')}`;
    if (!referenced.has(id)) continue;
    const rec = {
      record_version: '0.1.0',
      lifecycle_status: 'ACTIVE',
      created_on: entry.verification?.accessed_on ?? registry.as_of,
      source_roles: [entry.source_role, ...(entry.additional_roles ?? [])],
      authors: entry.authors ?? (entry.editor !== undefined ? [entry.editor] : undefined),
      rights_handling: entry.rights?.handling,
      verification_state: entry.verification?.state,
      source_class: entry.source_class,
      title: entry.title,
      organization: entry.organization,
      publication_date: entry.publication_date,
      official_url: entry.official_url,
    };
    if (entry.edition !== undefined) rec.edition = entry.edition;
    if (entry.stable_identifier !== undefined) rec.stable_identifier = entry.stable_identifier;
    if (entry.authority_fit !== undefined) rec.authority_fit = entry.authority_fit;
    if (entry.authority_limitations !== undefined) rec.authority_limitations = entry.authority_limitations;
    out.push({ id, typeName: 'se:Source', record: rec });
  }
  return out;
}

// ---------------------------------------------------------------------------
// projection

export function projectRecords(inputs) {
  const { lanes, registry, catalog } = inputs;
  const { byId, stats } = assembleRecords(lanes);

  // boundary-notes normalization (documented projection-only rule).
  // Works on a clone: projectRecords must stay non-mutating (lane inputs are
  // shared across builds and the baseline/candidate variants).
  const domain = byId.get('se:DOMAIN-SOFTWARE-ENGINEERING');
  if (!domain) throw new Error('project-c1: shared Domain record missing');
  const bn = domain.record.boundary_notes;
  if (!Array.isArray(bn) || bn[1] !== BOUNDARY_NOTE_PARSED) {
    throw new Error('project-c1: boundary_notes[1] does not match the documented parsed defect; normalization rule out of date');
  }
  const domainRecord = structuredClone(domain.record);
  domainRecord.boundary_notes = [...domainRecord.boundary_notes];
  domainRecord.boundary_notes[1] = BOUNDARY_NOTE_RAW;
  byId.set('se:DOMAIN-SOFTWARE-ENGINEERING', { ...domain, record: domainRecord });

  // relationship preprocessing: predicate localname -> catalog meaning CURIE
  const meaning = new Map();
  for (const p of catalog.predicates) {
    if (!p.meaning || !p.meaning.startsWith('se:')) throw new Error(`project-c1: catalog predicate ${p.predicate} has no se: meaning`);
    meaning.set(p.predicate, { meaningCurie: p.meaning, subjectClasses: p.subject_classes ?? [], objectClasses: p.object_classes ?? [] });
  }
  const classOf = new Map();
  for (const [id, entry] of byId) classOf.set(id, entry.type);
  classOf.set('se:DOMAIN-SOFTWARE-ENGINEERING', 'se:Domain');

  const nodes = [];
  for (const [id, entry] of byId) {
    let record = entry.record;
    if (entry.type === 'se:RelationshipAssertion') {
      if (typeof record.predicate !== 'string' || !meaning.has(record.predicate)) {
        throw new Error(`project-c1: ${id} predicate "${record.predicate}" not declared in RELATIONSHIP_CATALOG`);
      }
      const m = meaning.get(record.predicate);
      for (const [role, clsKey] of [['subject_ref', 'subjectClasses'], ['object_ref', 'objectClasses']]) {
        const refId = record[role];
        if (typeof refId !== 'string' || !classOf.has(refId)) {
          throw new Error(`project-c1: ${id} ${role} "${refId}" does not resolve (dangling typed edge)`);
        }
        const cls = classOf.get(refId);
        if (!m[clsKey].includes(cls.replace('se:', ''))) {
          throw new Error(`project-c1: ${id} ${role} class ${cls} not allowed for predicate ${record.predicate}`);
        }
      }
      record = { ...record, predicate: m.meaningCurie };
    }
    nodes.push(projectRecord(id, entry.type, record));
  }

  // typed relationship edges: materialize <subject> <meaning> <object>
  const edgeMap = new Map();
  for (const entry of byId.values()) {
    if (entry.type !== 'se:RelationshipAssertion') continue;
    const r = entry.record;
    const m = meaning.get(r.predicate);
    const key = `${r.subject_ref}|${m.meaningCurie}`;
    if (!edgeMap.has(key)) edgeMap.set(key, []);
    edgeMap.get(key).push(r.object_ref);
  }
  for (const [key, objects] of edgeMap) {
    const [subjId, predCurie] = key.split('|');
    const node = nodes.find((n) => n['@id'] === subjId);
    if (!node) throw new Error(`project-c1: typed edge subject ${subjId} missing`);
    const sorted = objects.sort();
    const refs = sorted.map((x) => ({ '@id': x }));
    node[predCurie] = refs.length === 1 ? refs[0] : refs;
  }

  // derived Source records
  for (const s of deriveSources(registry, byId)) {
    if (nodes.some((n) => n['@id'] === s.id)) throw new Error(`project-c1: duplicate source node ${s.id}`);
    nodes.push(projectRecord(s.id, s.typeName, s.record));
  }

  // graph root
  nodes.push({
    '@id': GRAPH_ID_V211,
    '@type': 'se:KnowledgeGraph',
    'se:domain_ref': { '@id': 'se:DOMAIN-SOFTWARE-ENGINEERING' },
    'se:generated_on': '2026-11-07',
    'se:graph_version': '0.1.0',
  });

  return { nodes, stats };
}

// ---------------------------------------------------------------------------
// overlay application (Owner-authorized R13 content; generic applier)

const ALLOWED_ADDITION_TYPES = new Set(['se:SourceLocator', 'se:Evidence']);
const FORBIDDEN_ADDITION_TYPES = new Set(['se:ReviewDecision', 'se:Counterexample']);

export function checkOverlayContract(overlay) {
  if (overlay.schema !== 'tally.dke.c1-owner-authorized-overlay/1') throw new Error('project-c1: overlay schema mismatch');
  const o = overlay.owner ?? {};
  if (o.OD05 !== 'NARROW_CLAIM') throw new Error(`project-c1: Owner decision OD05 drifted: ${o.OD05}`);
  if (o.GH2018 !== 'ACCEPT_PATCH') throw new Error(`project-c1: Owner decision GH2018 drifted: ${o.GH2018}`);
  if (overlay.original?.graphBlob !== PINS.frozenGraphJsonld) throw new Error('project-c1: overlay pins a different original graph blob');
  const q = overlay.qualification ?? {};
  if (q.sourceOwner !== '2/2 CONFIRMED' || q.oracle !== 'NOT_ATTESTED' || q.sap !== '0/6'
    || q.root !== 'UNKNOWN' || q.dkeR8 !== '0/53') {
    throw new Error('project-c1: qualification conservation violated in overlay (no Owner-to-Oracle laundering)');
  }
  if (overlay.originalsPreserved !== true) throw new Error('project-c1: originalsPreserved must be true');
}

// shape an overlay change value the way the projection would have serialized
// the equivalent record field (sorted refs, single-item collapse)
function shapeOverlayValue(key, value) {
  const isRef = /^se:(.*_refs?|predicate|subject_ref|object_ref|source_ref|source_locator_ref|target_scope_ref|domain_ref|view_ref)$/.test(key);
  if (Array.isArray(value)) {
    if (value.length === 0) return undefined; // drop the key entirely
    const items = isRef ? value.map((x) => ({ '@id': x['@id'] ?? x })) : value.slice();
    if (isRef) items.sort((a, b) => (a['@id'] < b['@id'] ? -1 : a['@id'] > b['@id'] ? 1 : 0));
    else items.sort((a, b) => (String(a) < String(b) ? -1 : String(a) > String(b) ? 1 : 0));
    return items.length === 1 ? items[0] : items;
  }
  return value;
}

export function applyOverlay(nodes, overlay) {
  checkOverlayContract(overlay);
  const out = nodes.map((n) => ({ ...n }));
  const index = new Map(out.map((n) => [n['@id'], n]));
  const changedOriginalIds = new Set();
  for (const [id, fields] of Object.entries(overlay.changes ?? {})) {
    const node = index.get(id);
    if (!node) throw new Error(`project-c1: overlay change target ${id} not found in projection`);
    changedOriginalIds.add(id);
    const finalId = fields['@id'] ?? node['@id'];
    for (const [k, v] of Object.entries(fields)) {
      if (k === '@id') continue;
      const shaped = shapeOverlayValue(k, v);
      if (shaped === undefined) delete node[k];
      else node[k] = shaped;
    }
    if (finalId !== node['@id']) {
      node['@id'] = finalId;
      index.delete(id);
      index.set(finalId, node);
    }
  }
  for (const addition of overlay.additions ?? []) {
    if (index.has(addition['@id'])) throw new Error(`project-c1: addition ${addition['@id']} collides with existing node`);
    if (FORBIDDEN_ADDITION_TYPES.has(addition['@type']) || !ALLOWED_ADDITION_TYPES.has(addition['@type'])) {
      throw new Error(`project-c1: addition type not authorized for a source patch: ${addition['@type']}`);
    }
    const node = { '@id': addition['@id'], '@type': addition['@type'] };
    for (const [k, v] of Object.entries(addition)) {
      if (k === '@id' || k === '@type') continue;
      const shaped = shapeOverlayValue(k, v);
      if (shaped !== undefined) node[k] = shaped;
    }
    out.push(node);
    index.set(node['@id'], node);
  }
  return { nodes: out, changedOriginalIds };
}

// ---------------------------------------------------------------------------
// invariants + reference resolution

function collectRefs(node) {
  const refs = [];
  for (const [k, v] of Object.entries(node)) {
    if (k === '@id' || k === '@type') continue;
    const vals = Array.isArray(v) ? v : [v];
    for (const item of vals) {
      if (item && typeof item === 'object' && typeof item['@id'] === 'string') refs.push([k, item['@id']]);
    }
  }
  return refs;
}

export function assertRefsResolve(nodes) {
  const ids = new Set(nodes.map((n) => n['@id']));
  if (ids.size !== nodes.length) throw new Error('project-c1: duplicate node ids in final graph');
  for (const n of nodes) {
    for (const [key, r] of collectRefs(n)) {
      // se:predicate points into the relationship-catalog meaning IRI space,
      // not the record-id space (same shape as the frozen v2.1.1 export)
      if (key === 'se:predicate') continue;
      if (!ids.has(r)) throw new Error(`project-c1: dangling reference ${n['@id']} -> ${r}`);
    }
  }
}

export function assertStatusConservation(nodes, expectedNodes) {
  if (nodes.length !== expectedNodes) {
    throw new Error(`project-c1: node count ${nodes.length} != ${expectedNodes}`);
  }
  const claims = nodes.filter((n) => n['@type'] === 'se:KnowledgeClaim');
  if (claims.length !== 106 || claims.some((c) => c['se:review_status'] !== 'PENDING')) {
    throw new Error('project-c1: 106 PENDING KnowledgeClaims required');
  }
  for (const badType of ['se:ReviewDecision', 'se:Counterexample']) {
    if (nodes.some((n) => n['@type'] === badType)) throw new Error(`project-c1: ${badType} must not exist`);
  }
  if (nodes.some((n) => n['se:lifecycle_status'] === 'ACCEPTED')) {
    throw new Error('project-c1: ACCEPTED lifecycle_status must not exist');
  }
}

export function assertV211Baseline(nodes) {
  const ids = nodes.map((n) => n['@id']).sort();
  assertStatusConservation(nodes, 807);
  return ids;
}

export function assertC1Candidate(nodes, overlay, originalNodes) {
  assertStatusConservation(nodes, 809);
  const byId = new Map(nodes.map((n) => [n['@id'], n]));
  const changed = new Set(Object.keys(overlay.changes ?? {}));
  const added = new Set((overlay.additions ?? []).map((a) => a['@id']));

  // OD05 narrowing is actually applied
  const dm = byId.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD');
  if (!dm) throw new Error('project-c1: DM-REQ-FUNCTIONAL-DOC-METHOD missing');
  if (dm['se:constraints'].some((x) => x.includes('uniformly across'))) {
    throw new Error('project-c1: universal uniformity constraint survived the NARROW_CLAIM');
  }
  if (!dm['se:constraints'].some((x) => x.startsWith('At least one'))) {
    throw new Error('project-c1: narrowed at-least-one constraint missing');
  }
  const pat = byId.get('se:PAT-REQ-USER-STORY-SPECIFICATION');
  if (typeof pat['se:solution'] === 'string' && pat['se:solution'].includes('uniform across')) {
    throw new Error('project-c1: uniformity phrasing survived in the Pattern');
  }

  // GH-2018 patch: separate locator + PENDING evidence, caveat preserved,
  // original topology locator untouched
  const locOriginal = byId.get('se:LOC-09F-GH-REPLICATION');
  const origLocOriginal = originalNodes.find((n) => n['@id'] === 'se:LOC-09F-GH-REPLICATION');
  if (!nodesDeepEqual(locOriginal, origLocOriginal)) {
    throw new Error('project-c1: original 16:24 topology locator was modified');
  }
  const locBacklog = byId.get('se:LOC-09F-GH-BACKLOG');
  const evdBacklog = byId.get('se:EVD-09F-GH-BACKLOG');
  if (!locBacklog || !evdBacklog) throw new Error('project-c1: GH-2018 addition missing');
  if (!added.has(locBacklog['@id']) || !added.has(evdBacklog['@id'])) {
    throw new Error('project-c1: GH-2018 nodes must come from overlay additions');
  }
  if (!locBacklog['se:locator'].includes('16:45') || !locBacklog['se:locator'].includes('23:03')) {
    throw new Error('project-c1: backlog locator lost the 16:45/23:03 scoping');
  }
  if (evdBacklog['se:lifecycle_status'] !== 'PENDING_REVIEW'
    || evdBacklog['se:extraction_status'] !== 'EXTRACTED_PENDING_REVIEW'
    || evdBacklog['se:synthetic'] !== false) {
    throw new Error('project-c1: backlog evidence must stay PENDING_REVIEW/EXTRACTED_PENDING_REVIEW');
  }
  if (!evdBacklog['se:uncertainty'].includes('UNKNOWN')) {
    throw new Error('project-c1: backlog evidence lost its UNKNOWN list');
  }
  if (evdBacklog['se:source_locator_ref']?.['@id'] !== locBacklog['@id']) {
    throw new Error('project-c1: backlog evidence not bound to the new locator');
  }
  const caze = byId.get('se:CASE-09F-GH-DB-FAILOVER');
  const evIds = (Array.isArray(caze['se:evidence_refs']) ? caze['se:evidence_refs'] : [caze['se:evidence_refs']])
    .map((x) => x['@id']).sort();
  if (JSON.stringify(evIds) !== JSON.stringify(['se:EVD-09F-GH-BACKLOG', 'se:EVD-09F-GH-RECOVERY', 'se:EVD-09F-GH-REPLICATION'])) {
    throw new Error(`project-c1: case evidence_refs wrong: ${evIds}`);
  }
  if (!(caze['se:outcome'].includes('200,000') && caze['se:outcome'].includes('23:03') && caze['se:outcome'].includes('dropped'))) {
    throw new Error('project-c1: dropped-payload caveat lost from case outcome');
  }

  // renamed graph root
  const root = byId.get('se:GRAPH-V2.1.2-C1-OWNER-R13');
  if (!root || root['se:graph_version'] !== '0.1.1' || root['se:generated_on'] !== BUILD_DATE
    || root['se:domain_ref']?.['@id'] !== 'se:DOMAIN-SOFTWARE-ENGINEERING') {
    throw new Error('project-c1: v2.1.2 graph root wrong');
  }
  if (byId.has(GRAPH_ID_V211)) throw new Error('project-c1: v2.1.1 root id must not survive the rename');

  // originals preserved except the explicitly changed ids; every surviving
  // node must have original provenance
  const origById = new Map(originalNodes.map((n) => [n['@id'], n]));
  for (const n of originalNodes) {
    if (changed.has(n['@id'])) continue;
    if (!nodesDeepEqual(byId.get(n['@id']), n)) {
      throw new Error(`project-c1: original node ${n['@id']} drifted without an overlay change entry`);
    }
  }
  for (const n of nodes) {
    if (added.has(n['@id'])) continue;
    if (n['@id'] === 'se:GRAPH-V2.1.2-C1-OWNER-R13') continue;
    if (!origById.has(n['@id'])) {
      throw new Error(`project-c1: node ${n['@id']} has no original provenance`);
    }
  }

  // qualification conservation: no review decisions may appear post-overlay
  if (nodes.some((n) => n['@type'] === 'se:ReviewDecision')) throw new Error('project-c1: ReviewDecision appeared');

  assertRefsResolve(nodes);
}

// ---------------------------------------------------------------------------

export const TTL_HEADER_V211 = [
  '# SEK-08/SEK-09H canonical knowledge-graph projection (Turtle).',
  '# SEK-09H (#368) rebuild: adds 3 derived SEK-09F sources, 4 locators, 5 evidence, 2 PENDING_REVIEW cases.',
  '# Deterministic: sorted subjects/predicates/objects; fixed build date 2026-11-07.',
  '# Source of truth: ten reviewed SEK-06/SEK-07 lane bundles + SEK-09F case records + pinned model 0.3.0.',
];

export const TTL_HEADER_V212 = [
  '# Tally v2.1.2 C1 owner-source canonical knowledge-graph projection (Turtle).',
  '# Single-Writer source candidate: OD05=NARROW_CLAIM + GH-2018=ACCEPT_PATCH (Tally #398 R13).',
  '# Deterministic: sorted subjects/predicates/objects; fixed build date 2026-11-12.',
  '# Source of truth: frozen v2.1.1 canonical lane inputs + source/C1_OWNER_REVISION_R13.json overlay.',
];

// Build the canonical triple set + exports for one variant.
//   variant='v211-baseline': frozen v2.1.1 projection (byte-reproduction gate)
//   variant='v212-c1':       v2.1.2 C1 owner-source candidate
export function buildGraph(variant, inputs = loadInputs()) {
  const { nodes: baselineNodes } = projectRecords(inputs);
  if (variant === 'v211-baseline') {
    assertV211Baseline(baselineNodes);
    assertRefsResolve(baselineNodes);
    const triples = nodesToTriples(baselineNodes);
    return {
      nodes: baselineNodes,
      triples,
      digest: canonicalDigest(triples),
      jsonld: serializeJsonld(baselineNodes),
      ttl: serializeTtl(triples, TTL_HEADER_V211),
    };
  }
  if (variant !== 'v212-c1') throw new Error(`project-c1: unknown variant ${variant}`);
  const { nodes, changedOriginalIds } = applyOverlay(baselineNodes, inputs.overlay);
  // candidate invariants are part of the build itself, not only of the CLI
  assertC1Candidate(nodes, inputs.overlay, baselineNodes);
  const triples = nodesToTriples(nodes);
  return {
    nodes,
    baselineNodes,
    changedOriginalIds,
    triples,
    digest: canonicalDigest(triples),
    jsonld: serializeJsonld(nodes),
    ttl: serializeTtl(triples, TTL_HEADER_V212),
    overlay: inputs.overlay,
  };
}

export function buildManifest(digest, ttlBytes, jsonldBytes) {
  const sha256 = (s) => createHash('sha256').update(s, 'utf8').digest('hex');
  return [
    '# Tally v2.1.2 C1 generated-graph manifest (deterministic build)',
    '# build date (fixed): 2026-11-12',
    `# canonical graph digest (sha256 over sorted N-Triples): ${digest}`,
    '# verify with: sha256sum -c manifest.sha256',
    `${sha256(ttlBytes)}  graph.ttl`,
    `${sha256(jsonldBytes)}  graph.jsonld`,
    '',
  ].join('\n');
}

export { nodesDeepEqual, sortNodes, triplesEqualSet, parseTtl, serializeTtl, serializeJsonld, nodesToTriples, canonicalDigest };
