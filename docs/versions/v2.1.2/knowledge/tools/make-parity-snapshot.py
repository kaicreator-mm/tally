#!/usr/bin/env python3
# make-parity-snapshot.py — generate the committed PyYAML parity hashes for
# the frozen v2.1.1 YAML inputs consumed by the v2.1.2 canonical build.
#
# For each input the script parses it with PyYAML (duplicate keys are a hard
# error, mirroring the builder's UniqueLoader discipline), normalizes the tree
# to JSON-representable values (dates -> ISO strings), dumps canonical JSON
# (sorted keys, compact separators, raw UTF-8) and records its SHA-256 in
# fixtures/c1-yaml-parity.sha256.
#
# The native test suite (build-c1-canonical.test.mjs) recomputes the same
# canonical JSON with the dependency-free Node parser (yaml-min.mjs) and
# compares hashes — a two-independent-parser agreement proof over the whole
# frozen corpus (10,485 strings / 191 booleans / 1 date scalar at generation
# time). Regenerate only when a frozen input blob pin changes, which would be
# a source failure by itself.
#
# Pinned environment (same as the SEK-08/09H canonical rebuild):
#   Python 3.14.7, PyYAML 6.x  (venv: see docs/versions/v2.1.1/methods/observations/SEK-09H.md)
#   python tools/make-parity-snapshot.py <repo-root>

import hashlib
import json
import sys
from datetime import date, datetime
from pathlib import Path

import yaml

LANES = [
    'architecture-design', 'build-delivery', 'foundations-processes',
    'implementation-code', 'maintenance-evolution', 'planning-risk-quality',
    'reliability-performance', 'requirements', 'security-engineering',
    'testing-verification',
]


class UniqueLoader(yaml.SafeLoader):
    pass


def _no_dup(loader, node, deep=False):
    seen = {}
    for k, v in loader.construct_pairs(node, deep=deep):
        if k in seen:
            raise ValueError(f'duplicate key {k!r}')
        seen[k] = v
    return seen


UniqueLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, _no_dup)


def norm(value):
    if isinstance(value, dict):
        return {k: norm(x) for k, x in value.items()}
    if isinstance(value, list):
        return [norm(x) for x in value]
    if isinstance(value, (date, datetime)):
        return value.isoformat()
    return value


def canonical_json(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False,
                      separators=(',', ':'))


def main():
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path('.')
    inputs = [f'docs/versions/v2.1.1/authority/SOURCE_REGISTRY.yaml',
              f'docs/versions/v2.1.1/knowledge/model/RELATIONSHIP_CATALOG.yaml']
    inputs += [f'docs/versions/v2.1.1/knowledge/vertical-slice/lanes/{l}/records.yaml'
               for l in LANES]
    lines = []
    for rel in inputs:
        full = root / rel
        data = norm(yaml.load(full.read_text(encoding='utf-8'), Loader=UniqueLoader))
        blob = canonical_json(data).encode('utf-8')
        digest = hashlib.sha256(blob).hexdigest()
        lines.append(f'{digest}  {rel}')
        print(f'{digest}  {rel}')
    out = root / 'docs/versions/v2.1.2/knowledge/fixtures/c1-yaml-parity.sha256'
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
    print(f'wrote {out} ({len(lines)} inputs)')


if __name__ == '__main__':
    main()
