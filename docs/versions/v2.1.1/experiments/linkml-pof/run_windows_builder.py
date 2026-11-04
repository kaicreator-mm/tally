"""Reproduce SEK-04 using an isolated Windows LinkML environment.

Usage: python run_windows_builder.py <output-directory>
Records each command, exit code, stdout and stderr independently; failures do not
prevent subsequent probes. Output directory must be empty for an independent run.
"""
import copy
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import traceback

import yaml

HERE = Path(__file__).resolve().parent
OUT = Path(sys.argv[1]).resolve()
OUT.mkdir(parents=True, exist_ok=False)
SCRIPTS = Path(sys.executable).parent
SCHEMA = HERE / 'schema/software-engineering.yaml'
FIX = HERE / 'fixtures'
ENV = dict(os.environ, PYTHONUTF8='1', PYTHONIOENCODING='utf-8')
records = []

def cmd(name, executable, *args):
    argv = [str(executable), *map(str, args)]
    print(name, flush=True)
    try:
        result = subprocess.run(argv, cwd=HERE, env=ENV, capture_output=True, timeout=180)
        rc, stdout, stderr = result.returncode, result.stdout.decode('utf-8', 'replace'), result.stderr.decode('utf-8', 'replace')
    except subprocess.TimeoutExpired as e:
        rc, stdout, stderr = 'TIMEOUT', (e.stdout or b'').decode('utf-8', 'replace'), (e.stderr or b'').decode('utf-8', 'replace')
    record = {'name': name, 'argv': argv, 'cwd': str(HERE), 'exit_code': rc, 'stdout': stdout, 'stderr': stderr}
    records.append(record)
    (OUT / 'commands.json').write_text(json.dumps(records, ensure_ascii=False, indent=2), encoding='utf-8')
    return rc

linkml = SCRIPTS / 'linkml.exe'
convert = SCRIPTS / 'linkml-convert.exe'
def save(name, obj):
    p = OUT / name
    p.write_text(yaml.safe_dump(obj, allow_unicode=True, sort_keys=False), encoding='utf-8')
    return p
def load(path):
    return yaml.safe_load(path.read_text(encoding='utf-8'))
def validate(name, path, cls='KnowledgeCorpus', schema=SCHEMA):
    return cmd(name, linkml, 'validate', '-s', schema, '--target-class', cls, path)
def conversion(name, path, cls, fmt, dest):
    return cmd(name, convert, '-s', SCHEMA, '-C', cls, path, '-t', fmt, '-o', dest)

cmd('python_version', sys.executable, '--version')
cmd('installed_packages', sys.executable, '-m', 'pip', 'freeze')
cmd('linkml_version', linkml, '--version')
cmd('rdf_help', linkml, 'generate', 'rdf', '--help')
(OUT / 'environment.json').write_text(json.dumps({'platform': platform.platform(), 'python': sys.version, 'executable': sys.executable, 'baseline': 'a5e438d0f0bebc678e581f4518e17e0aeae68163', 'tree': '07f40bc237ae9f4ea62e49f78a5badc85e711cde', 'schema_sha256': hashlib.sha256(SCHEMA.read_bytes()).hexdigest(), 'pythonutf8': '1'}, indent=2), encoding='utf-8')
cmd('schema_metamodel', linkml, 'validate', SCHEMA)
cmd('schema_lint', linkml, 'lint', SCHEMA)

front = (FIX / 'frontmatter/SE-CLM-0001.md').read_text(encoding='utf-8')
assert front.startswith('---\n')
a = yaml.safe_load(front.split('---', 2)[1])
pa = save('frontmatter-extracted.yaml', a)
pc = FIX / 'standalone-yaml/KNOWLEDGE_CORPUS.yaml'
c = load(pc)
parts = [('source', 'Source', 'SOURCE.yaml'), ('locator', 'SourceLocator', 'SOURCE_LOCATOR.yaml'), ('evidence', 'Evidence', 'EVIDENCE.yaml'), ('bundle', 'ClaimBundle', 'SE-CLM-0001.yaml')]
b = {n: load(FIX / 'yaml-plus-markdown' / f) for n, _, f in parts}
assembled = {'id': c['id'], 'title': c['title'], 'corpus_status': c['corpus_status'], 'sources': [b['source']], 'source_locators': [b['locator']], 'evidence_records': [b['evidence']], 'domain_nodes': b['bundle']['domain_nodes'], 'knowledge_claims': [b['bundle']['claim']], 'relationship_assertions': b['bundle']['relationship_assertions']}
save('b-assembled.yaml', assembled)
observations = {'logical_equal_A_C': a == c, 'logical_equal_assembled_B_C': assembled == c, 'B_wrapper_metadata': 'B has no corpus wrapper; C wrapper id/title/status were supplied solely for logical assembly.'}
validate('positive_A_frontmatter', pa)
validate('positive_C_corpus', pc)
for n, cls, filename in parts:
    validate('positive_B_' + n, FIX / 'yaml-plus-markdown' / filename, cls)

negative = {}
negative['invalid_enum'] = copy.deepcopy(c); negative['invalid_enum']['knowledge_claims'][0]['review_status'] = 'NOT_A_STATUS'
negative['missing_statement'] = copy.deepcopy(c); del negative['missing_statement']['knowledge_claims'][0]['statement']
negative['missing_id'] = copy.deepcopy(c); del negative['missing_id']['knowledge_claims'][0]['id']
negative['duplicate_id'] = copy.deepcopy(c); negative['duplicate_id']['sources'].append(copy.deepcopy(c['sources'][0]))
negative['dangling_locator'] = copy.deepcopy(c); negative['dangling_locator']['evidence_records'][0]['source_locator_ref'] = 'SE-LOC-MISSING'
negative['wrong_object_kind'] = copy.deepcopy(c); negative['wrong_object_kind']['relationship_assertions'][0]['object_kind'] = 'DOMAIN_NODE'
negative['duplicate_unique_key'] = copy.deepcopy(c); dup = copy.deepcopy(c['source_locators'][0]); dup['id'] = 'SE-LOC-DUP'; negative['duplicate_unique_key']['source_locators'].append(dup)
for name, data in negative.items():
    validate('negative_' + name, save(name + '.yaml', data))

def closure(data):
    errors = []
    groups = {'Source': 'sources', 'SourceLocator': 'source_locators', 'Evidence': 'evidence_records', 'DomainNode': 'domain_nodes', 'KnowledgeClaim': 'knowledge_claims', 'RelationshipAssertion': 'relationship_assertions'}
    index = {}
    all_ids = set()
    for cls, key in groups.items():
        index[cls] = set()
        for item in data.get(key, []):
            ident = item['id']
            if ident in all_ids: errors.append('duplicate id: ' + ident)
            all_ids.add(ident); index[cls].add(ident)
    def ref(item, slot, cls, multiple=False):
        values = item.get(slot, []) if multiple else [item.get(slot)]
        for value in values:
            if value is not None and value not in index[cls]: errors.append(f'{item["id"]}.{slot}: {value} is not a {cls}')
    for x in data['source_locators']: ref(x, 'source_ref', 'Source')
    for x in data['evidence_records']: ref(x, 'source_locator_ref', 'SourceLocator')
    for x in data['domain_nodes']:
        ref(x, 'source_ref', 'Source'); ref(x, 'source_locator_ref', 'SourceLocator')
    for x in data['knowledge_claims']:
        ref(x, 'source_refs', 'Source', True); ref(x, 'domain_node_refs', 'DomainNode', True)
    for x in data['relationship_assertions']:
        ref(x, 'subject_ref', 'KnowledgeClaim'); ref(x, 'evidence_ref', 'Evidence')
        ref(x, 'object_ref', {'EVIDENCE':'Evidence', 'DOMAIN_NODE':'DomainNode'}[x['object_kind']])
    keys = [(x['source_ref'], x['locator']) for x in data['source_locators']]
    if len(keys) != len(set(keys)): errors.append('duplicate source_ref + locator')
    return errors
observations['closure_C'] = closure(c)
observations['closure_B_assembled'] = closure(assembled)
observations['closure_negative_controls'] = {n: closure(negative[n]) for n in ['duplicate_id','dangling_locator','wrong_object_kind','duplicate_unique_key']}

for i in (1, 2):
    cmd(f'json_schema_{i}', linkml, 'generate', 'json-schema', SCHEMA, '--output', OUT / f'schema-{i}.json')
    cmd(f'context_{i}', linkml, 'generate', 'jsonld-context', SCHEMA, '--output', OUT / f'context-{i}.jsonld')
    cmd(f'schema_rdf_stable_{i}', linkml, 'generate', 'rdf', '--diff-stable', '--format', 'ttl', SCHEMA, '--output', OUT / f'schema-stable-{i}.ttl')
    cmd(f'schema_rdf_default_{i}', linkml, 'generate', 'rdf', '--format', 'ttl', SCHEMA, '--output', OUT / f'schema-default-{i}.ttl')
    for n, path in [('A', pa), ('C', pc)]:
        for fmt in ('ttl', 'json-ld'):
            conversion(f'instance_{n}_{fmt}_{i}', path, 'KnowledgeCorpus', fmt, OUT / f'instance-{n}-{i}.{fmt}')
    for n, cls, filename in parts:
        conversion(f'instance_B_{n}_ttl_{i}', FIX / 'yaml-plus-markdown' / filename, cls, 'ttl', OUT / f'instance-B-{n}-{i}.ttl')

schema2 = load(SCHEMA)
schema2['slots']['review_status']['deprecated'] = 'Use assessment_status instead.'
schema2['slots']['assessment_status'] = copy.deepcopy(schema2['slots']['review_status']); schema2['slots']['assessment_status'].pop('deprecated')
schema2['classes']['KnowledgeClaim']['slots'].append('assessment_status')
schema2['slots']['migration_note'] = {'range': 'string'}
schema2['classes']['KnowledgeClaim']['slots'].append('migration_note')
evolved = save('evolved-schema.yaml', schema2)
new = copy.deepcopy(c); new['knowledge_claims'][0]['assessment_status'] = new['knowledge_claims'][0]['review_status']; new['knowledge_claims'][0]['migration_note'] = 'Mapped from review_status.'
newpath = save('evolved-record.yaml', new)
validate('evolution_old_on_original', pc)
validate('evolution_old_on_evolved', pc, schema=evolved)
validate('evolution_new_on_evolved', newpath, schema=evolved)
renamed = copy.deepcopy(schema2); renamed['classes']['KnowledgeClaim']['slots'].remove('review_status')
renamedpath = save('renamed-schema.yaml', renamed)
mapped = copy.deepcopy(new); del mapped['knowledge_claims'][0]['review_status']
mappedpath = save('mapped-record.yaml', mapped)
validate('evolution_old_on_renamed', pc, schema=renamedpath)
validate('evolution_mapped_on_renamed', mappedpath, schema=renamedpath)
validate('evolution_mapped_on_original', mappedpath)

try:
    from rdflib import Graph, RDF
    from rdflib.compare import isomorphic
    pairs = [('schema', 'schema-1.json', 'schema-2.json'), ('context', 'context-1.jsonld', 'context-2.jsonld'), ('rdf_stable', 'schema-stable-1.ttl', 'schema-stable-2.ttl'), ('rdf_default', 'schema-default-1.ttl', 'schema-default-2.ttl')]
    pairs += [(f'instance_{n}_{fmt}', f'instance-{n}-1.{fmt}', f'instance-{n}-2.{fmt}') for n in ('A','C') for fmt in ('ttl','json-ld')]
    observations['byte_comparison'] = {name: (OUT / p).read_bytes() == (OUT / q).read_bytes() if (OUT / p).exists() and (OUT / q).exists() and (OUT / p).stat().st_size and (OUT / q).stat().st_size else None for name,p,q in pairs}
    graphs = {}
    for n in ('A','C'):
        p = OUT / f'instance-{n}-1.ttl'
        if p.exists() and p.stat().st_size: graphs[n] = Graph().parse(p, format='turtle')
    bg = Graph()
    for n, _, _ in parts:
        p = OUT / f'instance-B-{n}-1.ttl'
        if not p.exists() or not p.stat().st_size: break
        bg.parse(p, format='turtle')
    else: graphs['B'] = bg
    observations['graph_triples_before_wrapper_removal'] = {n:len(g) for n,g in graphs.items()}
    # Remove only triples whose subject is explicitly typed as a format wrapper.
    for g in graphs.values():
        for s in list(g.subjects(RDF.type, None)):
            if any(str(t).endswith(('KnowledgeCorpus', 'ClaimBundle')) for t in g.objects(s, RDF.type)):
                g.remove((s,None,None))
    if 'B' in graphs: graphs['B'].serialize(destination=OUT / 'b-merged-without-wrapper.ttl', format='turtle')
    observations['graph_triples_after_wrapper_removal'] = {n:len(g) for n,g in graphs.items()}
    observations['graph_isomorphism'] = {f'{x}_{y}': isomorphic(graphs[x], graphs[y]) if x in graphs and y in graphs else None for x,y in [('A','C'),('A','B'),('B','C')]}
except Exception:
    observations['graph_analysis_error'] = traceback.format_exc()
(OUT / 'observations.json').write_text(json.dumps(observations, ensure_ascii=False, indent=2), encoding='utf-8')
print(json.dumps(observations, ensure_ascii=False, indent=2), flush=True)
