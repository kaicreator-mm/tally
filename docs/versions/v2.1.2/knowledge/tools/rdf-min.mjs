// rdf-min.mjs — dependency-free deterministic RDF projection helpers.
//
// Implements exactly the projection serialization semantics documented for the
// v2.1.1 canonical lane (docs/versions/v2.1.1/knowledge/generated/README.md):
//   - one resource per record, @id = expanded canonical CURIE (se: = base#)
//   - slot predicates se:<field>; reference slots serialize as URIRefs
//   - se:synthetic as xsd:boolean; remaining fields as string literals
//   - typed relationship edges materialize <subject> <meaning> <object>
//   - deterministic ordering: sorted subjects/predicates/objects
// Exports: node/triple models, JSON-LD + Turtle serialization, strict Turtle
// parse-back, canonical N-Triples form + SHA-256 digest.

import { createHash } from 'node:crypto';

export const PREFIXES = {
  rdf: 'http://www.w3.org/1999/02/22-rdf-syntax-ns#',
  xsd: 'http://www.w3.org/2001/XMLSchema#',
  se: 'https://github.com/kaicreator-mm/tally#',
};

// Prefix emission order matches the frozen v2.1.1 export (@context declaration
// order, not alphabetical).
const PREFIX_KEYS = ['rdf', 'xsd', 'se'];

export function expandCurie(curie) {
  const i = curie.indexOf(':');
  if (i <= 0) throw new Error(`rdf-min: not a CURIE: "${curie}"`);
  const p = PREFIXES[curie.slice(0, i)];
  if (p === undefined) throw new Error(`rdf-min: unknown prefix in "${curie}"`);
  return p + curie.slice(i + 1);
}

// ---------------------------------------------------------------------------
// triples: { s: iri, p: iri, o: {kind:'iri',v} | {kind:'lit',v,dt?} }

function litNT(l) {
  const esc = l.v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
    .replace(/\n/g, '\\n').replace(/\r/g, '\\r').replace(/\t/g, '\\t')
    .replace(/[\u0000-\u001F]/g, (c) => `\\u${c.charCodeAt(0).toString(16).padStart(4, '0')}`);
  return `"${esc}"${l.dt === 'xsd:boolean' ? `^^<${PREFIXES.xsd}boolean>` : ''}`;
}

export function tripleNT(t) {
  const o = t.o.kind === 'iri' ? `<${t.o.v}>` : litNT(t.o);
  return `<${t.s}> <${t.p}> ${o} .`;
}

export function canonicalNT(triples) {
  const lines = [...new Set(triples.map(tripleNT))].sort();
  return lines;
}

export function canonicalDigest(triples) {
  const lines = canonicalNT(triples);
  return createHash('sha256').update(lines.join('\n') + '\n', 'utf8').digest('hex');
}

// ---------------------------------------------------------------------------
// JSON-LD node <-> triples
//
// Node shape (identical to the committed v2.1.1 generated graph.jsonld):
//   { '@id', '@type', 'se:<slot>': literal | true | false | {'@id'} | [ ... ] }

const REF_KEYS = new Set([
  'evidence_refs', 'claim_refs', 'subject_ref', 'object_ref', 'source_ref',
  'source_locator_ref', 'target_scope_ref', 'domain_ref', 'view_ref', 'predicate',
]);

export function isRefKey(key) {
  const local = key.startsWith('se:') ? key.slice(3) : key;
  return REF_KEYS.has(local) || local.endsWith('_ref') || local.endsWith('_refs')
    || /^[A-Z][A-Z_]*$/.test(local); // materialized typed-edge keys (catalog meanings)
}

function refValue(v) {
  return { '@id': v };
}

// Record (plain-slot form, as authored in lanes) -> projected JSON-LD node.
// Lists are sorted deterministically; a single-element list collapses to a
// scalar; empty lists drop the key entirely; se:synthetic is a JSON boolean.
export function projectRecord(recordId, typeName, record) {
  const node = { '@id': recordId, '@type': typeName };
  for (const [field, value] of Object.entries(record)) {
    if (field === 'id') continue;
    const key = `se:${field}`;
    if (Array.isArray(value)) {
      if (value.length === 0) continue; // empty lists produce no triples
      const isRef = isRefKey(key);
      const items = value.map((v) => (isRef ? refValue(v) : v));
      const sorted = isRef
        ? items.sort((a, b) => (a['@id'] < b['@id'] ? -1 : a['@id'] > b['@id'] ? 1 : 0))
        : [...items].sort((a, b) => {
          const sa = String(a); const sb = String(b);
          return sa < sb ? -1 : sa > sb ? 1 : 0;
        });
      node[key] = sorted.length === 1 ? sorted[0] : sorted;
    } else if (typeof value === 'boolean') {
      node[key] = value;
    } else if (isRefKey(key)) {
      node[key] = refValue(value);
    } else {
      node[key] = value;
    }
  }
  return node;
}

export function nodesToTriples(nodes) {
  const triples = [];
  for (const n of nodes) {
    const s = expandCurie(n['@id']);
    triples.push({ s, p: expandCurie('rdf:type'), o: { kind: 'iri', v: expandCurie(n['@type']) } });
    for (const [key, v] of Object.entries(n)) {
      if (key === '@id' || key === '@type') continue;
      const p = expandCurie(key);
      const vals = Array.isArray(v) ? v : [v];
      for (const item of vals) {
        if (item === null || item === undefined) {
          throw new Error(`rdf-min: null value on ${n['@id']} ${key}`);
        }
        if (typeof item === 'object') {
          if (typeof item['@id'] !== 'string') throw new Error(`rdf-min: bad ref on ${n['@id']} ${key}`);
          triples.push({ s, p, o: { kind: 'iri', v: expandCurie(item['@id']) } });
        } else if (typeof item === 'boolean') {
          triples.push({ s, p, o: { kind: 'lit', v: item ? 'true' : 'false', dt: 'xsd:boolean' } });
        } else if (typeof item === 'string') {
          triples.push({ s, p, o: { kind: 'lit', v: item } });
        } else {
          throw new Error(`rdf-min: unsupported literal type (${typeof item}) on ${n['@id']} ${key}`);
        }
      }
    }
  }
  return triples;
}

// deep (value-tree, key-order-insensitive) comparison of projected nodes
export function nodesDeepEqual(a, b) {
  if (a === b) return true;
  if (typeof a !== 'object' || typeof b !== 'object' || a === null || b === null) return false;
  const ka = Object.keys(a).sort(); const kb = Object.keys(b).sort();
  if (ka.length !== kb.length || ka.some((k, i) => k !== kb[i])) return false;
  return ka.every((k) => nodesDeepEqual(a[k], b[k]));
}

export function sortNodes(nodes) {
  return [...nodes].sort((a, b) => (a['@id'] < b['@id'] ? -1 : a['@id'] > b['@id'] ? 1 : 0));
}

// canonical key order: '@' keys first, then codepoint-sorted slot keys
function sortedNodeEntries(node) {
  const keys = Object.keys(node).sort();
  const out = {};
  for (const k of keys) out[k] = node[k];
  return out;
}

// Canonical value shaping, applied at serialization time and idempotent:
// slot lists are sorted (refs by @id, literals by string form) and a
// single-element list collapses to its scalar. This is the discipline the
// frozen v2.1.1 canonical export follows for every projected record; overlay
// additions/changes go through the same shaping so that the overlay product
// and the lane-integrated canonical projection converge byte-for-byte.
export function canonicalizeNodes(nodes) {
  const out = nodes.map((n) => {
    const node = { ...n };
    for (const [k, v] of Object.entries(n)) {
      if (k === '@id' || k === '@type' || !Array.isArray(v)) continue;
      const isRef = isRefKey(k);
      const items = isRef
        ? v.map((x) => ({ '@id': x['@id'] }))
        : [...v];
      items.sort((a, b) => {
        const sa = isRef ? a['@id'] : String(a);
        const sb = isRef ? b['@id'] : String(b);
        return sa < sb ? -1 : sa > sb ? 1 : 0;
      });
      node[k] = items.length === 1 ? items[0] : items;
    }
    return node;
  });
  return out;
}

export function serializeJsonld(nodes) {
  const doc = {
    '@context': {
      rdf: PREFIXES.rdf,
      xsd: PREFIXES.xsd,
      se: PREFIXES.se,
    },
    '@graph': sortNodes(canonicalizeNodes(nodes)).map(sortedNodeEntries),
  };
  return JSON.stringify(doc, null, 2) + '\n';
}

// ---------------------------------------------------------------------------
// Turtle serialization (restricted deterministic profile)

function escapeTtlLiteral(v) {
  return v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
    .replace(/\n/g, '\\n').replace(/\r/g, '\\r').replace(/\t/g, '\\t')
    .replace(/[\u0000-\u001F]/g, (c) => `\\u${c.charCodeAt(0).toString(16).padStart(4, '0')}`);
}

function ttlObject(t) {
  if (t.o.kind === 'iri') {
    const curie = toCurie(t.o.v);
    return curie ?? `<${t.o.v}>`;
  }
  if (t.o.dt === 'xsd:boolean') return `"${t.o.v}"^^xsd:boolean`;
  return `"${escapeTtlLiteral(t.o.v)}"`;
}

function toCurie(iri) {
  for (const p of PREFIX_KEYS) {
    if (iri.startsWith(PREFIXES[p])) return `${p}:${iri.slice(PREFIXES[p].length)}`;
  }
  return null;
}

// Serialize the canonical triple set. Objects of one predicate are emitted in
// codepoint order; predicates in codepoint order (rdf:type renders as "a" and
// sorts first); subjects in codepoint order. Blocks are separated by one
// blank line and the file ends with " .\n" — byte-comparable with the frozen
// v2.1.1 export style.
export function serializeTtl(triples, headerLines) {
  const bySubject = new Map();
  for (const t of triples) {
    if (!bySubject.has(t.s)) bySubject.set(t.s, []);
    bySubject.get(t.s).push(t);
  }
  const subjects = [...bySubject.keys()].sort();
  const blocks = [];
  for (const s of subjects) {
    const ts = bySubject.get(s);
    const typeObj = ts.find((t) => t.p === PREFIXES.rdf + 'type');
    const byPred = new Map();
    for (const t of ts) {
      if (t.p === PREFIXES.rdf + 'type') continue;
      if (!byPred.has(t.p)) byPred.set(t.p, []);
      byPred.get(t.p).push(t);
    }
    const preds = [...byPred.keys()].sort((a, b) => {
      const ka = toCurie(a) ?? a;
      const kb = toCurie(b) ?? b;
      return ka < kb ? -1 : ka > kb ? 1 : 0;
    });
    const lines = [`${toCurie(s)} a ${toCurie(typeObj.o.v)} ;`];
    for (let i = 0; i < preds.length; i += 1) {
      const p = preds[i];
      const objs = byPred.get(p).map(ttlObject)
        .sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
      const tail = i === preds.length - 1 ? ' .' : ' ;';
      lines.push(`     ${toCurie(p)} ${objs.join(', ')}${tail}`);
    }
    blocks.push(lines.join('\n'));
  }
  const head = (headerLines.length > 0 ? headerLines.join('\n') + '\n\n' : '')
    + `${PREFIX_KEYS.map((p) => `@prefix ${p}: <${PREFIXES[p]}> .`).join('\n')}\n\n`;
  return head + blocks.join('\n\n') + '\n';
}

// ---------------------------------------------------------------------------
// strict Turtle parse-back (restricted profile emitted by serializeTtl)

export function parseTtl(text) {
  const prefixes = new Map();
  const triples = [];
  const lines = text.split('\n');
  let subject = null;
  let predicate = null;
  for (let i = 0; i < lines.length; i += 1) {
    const raw = lines[i];
    const line = raw.trim();
    if (line.length === 0 || line.startsWith('#')) continue;
    const pm = /^@prefix\s+([A-Za-z][\w-]*):\s*<([^>]*)>\s*\.$/.exec(line);
    if (pm) { prefixes.set(pm[1], pm[2]); continue; }
    // object tokens
    const parseObjects = (text2) => {
      const objs = [];
      let k = 0;
      while (k < text2.length) {
        while (k < text2.length && (text2[k] === ' ' || text2[k] === ',')) k += 1;
        if (k >= text2.length) break;
        if (text2[k] === '"') {
          let v = '';
          k += 1;
          while (k < text2.length && text2[k] !== '"') {
            if (text2[k] === '\\') {
              const e = text2[k + 1];
              const map = { n: '\n', r: '\r', t: '\t', '"': '"', '\\': '\\' };
              if (map[e] !== undefined) { v += map[e]; k += 2; continue; }
              if (e === 'u') { v += String.fromCharCode(parseInt(text2.slice(k + 2, k + 6), 16)); k += 6; continue; }
              throw new Error(`rdf-min.parseTtl: unsupported escape at line ${i + 1}`);
            }
            v += text2[k];
            k += 1;
          }
          k += 1; // closing quote
          let dt;
          if (text2.startsWith('^^xsd:boolean', k)) { dt = 'xsd:boolean'; k += '^^xsd:boolean'.length; }
          else if (text2.startsWith('^^<', k)) {
            const end = text2.indexOf('>', k + 3);
            dt = text2.slice(k + 3, end);
            k = end + 1;
          }
          objs.push({ kind: 'lit', v, dt });
        } else {
          const m = /^([A-Za-z][\w-]*):([^\s,;.]+)/.exec(text2.slice(k));
          if (!m) throw new Error(`rdf-min.parseTtl: cannot parse object at line ${i + 1}: "${text2.slice(k, k + 40)}"`);
          const pfx = prefixes.get(m[1]);
          if (pfx === undefined) throw new Error(`rdf-min.parseTtl: unknown prefix ${m[1]} at line ${i + 1}`);
          objs.push({ kind: 'iri', v: pfx + m[2] });
          k += m[0].length;
        }
      }
      return objs;
    };
    const subjM = /^([A-Za-z][\w-]*):(\S+)\s+a\s+([A-Za-z][\w-]*):(\S+)\s*;$/.exec(line);
    if (subjM) {
      subject = prefixes.get(subjM[1]) + subjM[2];
      predicate = null;
      triples.push({
        s: subject,
        p: PREFIXES.rdf + 'type',
        o: { kind: 'iri', v: prefixes.get(subjM[3]) + subjM[4] },
      });
      continue;
    }
    if (line === '.') { subject = null; predicate = null; continue; }
    const endsWithDot = /\s\.$/.test(line);
    const body = endsWithDot ? line.slice(0, -2) : line;
    const predM = /^([A-Za-z][\w-]*):(\S+)\s+(.*?)(?:\s*;$)?$/.exec(body);
    if (predM && subject !== null) {
      predicate = prefixes.get(predM[1]) + predM[2];
      for (const o of parseObjects(predM[3])) triples.push({ s: subject, p: predicate, o });
      continue;
    }
    throw new Error(`rdf-min.parseTtl: cannot parse line ${i + 1}: "${line.slice(0, 60)}"`);
  }
  return triples;
}

export function triplesEqualSet(a, b) {
  const na = canonicalNT(a);
  const nb = canonicalNT(b);
  if (na.length !== nb.length) return false;
  for (let i = 0; i < na.length; i += 1) if (na[i] !== nb[i]) return false;
  return true;
}

// ---------------------------------------------------------------------------
// JSON parse of the generated graph (context + @graph)

export function parseJsonld(text) {
  const doc = JSON.parse(text);
  if (!doc['@graph'] || !doc['@context']) throw new Error('rdf-min.parseJsonld: missing @context/@graph');
  for (const [p, iri] of Object.entries(doc['@context'])) {
    if (PREFIXES[p] !== iri) throw new Error(`rdf-min.parseJsonld: unexpected context prefix ${p}: ${iri}`);
  }
  return doc['@graph'];
}
