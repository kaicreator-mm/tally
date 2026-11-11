#!/usr/bin/env node
// build-c1-canonical.mjs — deterministic canonical build of the Tally v2.1.2
// C1 owner-source candidate (native Node, zero dependencies).
//
// Modes (no implicit mutation):
//   --check            run every gate without writing; exit non-zero on any failure
//   --write            (re)generate generated/{graph.jsonld,graph.ttl,manifest.sha256}
//                      and the overlay candidate preview c1-candidate.graph.jsonld
//   --reproduce-v211   byte-reproduction gate only: the overlay-free projection of
//                      the frozen v2.1.1 lane inputs must reproduce the frozen
//                      generated exports byte-for-byte (lanes -> graph fidelity proof)
//
// Gate set:
//   R1 frozen-input Git Blob pins (lanes, registry, catalog, overlay, frozen graph)
//   R2 overlay-free projection == frozen v2.1.1 graph.jsonld / graph.ttl bytes
//      and canonical N-Triples digest 2625a6b3... (lane-integration fidelity)
//   R3 candidate invariants: 809 nodes; 106 PENDING claims; 0 ReviewDecision /
//      Counterexample / ACCEPTED; OD05 narrowing present; GH-2018 separate
//      locator + PENDING evidence + ~200k dropped-payload caveat preserved;
//      UNKNOWN conservation; originals preserved; all refs resolve
//   R4 overlay candidate preview == canonical export bytes (two construction
//      paths converge: Owner overlay on the generated graph vs lane-integrated
//      canonical projection)
//   R5 Turtle parse-back == JSON-LD triple set == internal canonical set
//   R6 determinism: two independent builds byte-identical
//   R7 manifest digests match the written bytes

import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import {
  buildGraph, buildManifest, loadInputs, PATHS, REPO_ROOT,
  FROZEN_DIGEST, assertV211Baseline, assertC1Candidate, parseTtl,
  triplesEqualSet, canonicalDigest, nodesToTriples,
} from './project-c1.mjs';

function fail(msg) {
  console.error('FAIL: ' + msg);
  process.exit(1);
}

function run() {
  const argv = process.argv.slice(2);
  const doWrite = argv.includes('--write');
  const doCheck = argv.includes('--check');
  const doRepro = argv.includes('--reproduce-v211');
  if (!doWrite && !doCheck && !doRepro) {
    throw new Error('Provide --check, --write or --reproduce-v211; no implicit mutation');
  }
  const inputs = loadInputs(); // R1: Git Blob pins verified here
  console.log('R1 frozen-input Git Blob pins verified (10 lanes, registry, catalog, frozen graph, overlay)');

  // R2: overlay-free lane projection reproduces the frozen canonical export
  const baseline = buildGraph('v211-baseline', inputs);
  assertV211Baseline(baseline.nodes);
  const frozenJson = readFileSync(REPO_ROOT + PATHS.frozenGraphJsonld, 'utf8');
  const frozenTtl = readFileSync(REPO_ROOT + PATHS.frozenGraphTtl, 'utf8');
  assert.equal(baseline.digest, FROZEN_DIGEST, 'canonical N-Triples digest must match the frozen v2.1.1 digest');
  assert.equal(baseline.jsonld, frozenJson, 'overlay-free projection must reproduce frozen graph.jsonld byte-for-byte');
  assert.equal(baseline.ttl, frozenTtl, 'overlay-free projection must reproduce frozen graph.ttl byte-for-byte');
  console.log('R2 lane-integration fidelity: frozen v2.1.1 graph.jsonld + graph.ttl reproduced byte-for-byte; canonical NT digest 2625a6b3... matched');

  if (doRepro && !doCheck && !doWrite) return;

  // R3: candidate invariants
  const cand = buildGraph('v212-c1', inputs);
  assertC1Candidate(cand.nodes, inputs.overlay, cand.baselineNodes);
  console.log('R3 candidate invariants OK (809 nodes; 106 PENDING claims; 0 ReviewDecision; OD05 narrowed; GH-2018 patched with dropped-payload caveat; UNKNOWN conserved)');

  // R5: Turtle parse-back == JSON-LD == internal canonical set
  const parsedTtl = parseTtl(cand.ttl);
  const jsonldTriples = nodesToTriples(JSON.parse(cand.jsonld)['@graph']);
  assert.ok(triplesEqualSet(parsedTtl, cand.triples), 'Turtle parse-back must equal the internal canonical triple set');
  assert.ok(triplesEqualSet(jsonldTriples, cand.triples), 'JSON-LD expansion must equal the internal canonical triple set');
  assert.equal(canonicalDigest(parsedTtl), cand.digest, 'Turtle parse-back digest must match');
  assert.equal(canonicalDigest(jsonldTriples), cand.digest, 'JSON-LD expansion digest must match');
  console.log(`R5 export equivalence: TTL parse-back == JSON-LD expansion == internal canonical set (${cand.triples.length} triples); digest ${cand.digest.slice(0, 12)}...`);

  // R6: determinism
  const cand2 = buildGraph('v212-c1', loadInputs());
  assert.equal(cand2.jsonld, cand.jsonld, 'rebuild jsonld must be byte-identical');
  assert.equal(cand2.ttl, cand.ttl, 'rebuild ttl must be byte-identical');
  console.log('R6 determinism: two independent builds byte-identical');

  const manifest = buildManifest(cand.digest, cand.ttl, cand.jsonld);

  if (doWrite) {
    writeFileSync(REPO_ROOT + PATHS.outGraphJsonld, cand.jsonld);
    writeFileSync(REPO_ROOT + PATHS.outGraphTtl, cand.ttl);
    writeFileSync(REPO_ROOT + PATHS.outManifest, manifest);
    console.log('WROTE generated/graph.jsonld, generated/graph.ttl, generated/manifest.sha256');
    return;
  }

  // --check: committed artifacts must match the build byte-for-byte
  const outJson = readFileSync(REPO_ROOT + PATHS.outGraphJsonld, 'utf8');
  const outTtl = readFileSync(REPO_ROOT + PATHS.outGraphTtl, 'utf8');
  const outManifest = readFileSync(REPO_ROOT + PATHS.outManifest, 'utf8');
  assert.equal(outJson, cand.jsonld, 'committed generated/graph.jsonld must match the canonical build');
  assert.equal(outTtl, cand.ttl, 'committed generated/graph.ttl must match the canonical build');
  assert.equal(outManifest, manifest, 'committed generated/manifest.sha256 must match the canonical build');
  // R7: manifest digests
  assert.ok(manifest.includes(cand.digest), 'manifest must carry the canonical digest');
  console.log('R7 committed canonical exports match the deterministic build (jsonld/ttl/manifest)');

  // R4: overlay preview convergence (preview must equal canonical jsonld bytes)
  const preview = readFileSync(REPO_ROOT + PATHS.outPreview, 'utf8');
  assert.equal(preview, cand.jsonld, 'overlay candidate preview must be byte-identical with the canonical lane-integrated export');
  console.log('R4 Owner-overlay candidate == canonical lane-integrated export (byte-identical)');

  console.log('CANONICAL BUILD PASS: all gates green');
}

try {
  run();
} catch (e) {
  fail(e.message);
}
