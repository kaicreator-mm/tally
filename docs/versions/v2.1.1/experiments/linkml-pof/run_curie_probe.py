"""Explore explicitly CURIE-qualified fixture identifiers; never overwrite fixtures."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import yaml
from rdflib import Graph, RDF, URIRef
from rdflib.compare import isomorphic

HERE = Path(__file__).resolve().parent
OUT = Path(sys.argv[1]).resolve(); OUT.mkdir(parents=True, exist_ok=False)
SCHEMA = HERE / 'schema/software-engineering.yaml'
EXE = Path(sys.executable).parent / 'linkml-convert.exe'
ENV = dict(os.environ, PYTHONUTF8='1', PYTHONIOENCODING='utf-8')
KEYS = {'id','source_ref','source_locator_ref','source_refs','domain_node_refs','subject_ref','object_ref','evidence_ref'}
def qualify(value, key=None):
    if isinstance(value, dict): return {k:qualify(v, k) for k,v in value.items()}
    if isinstance(value, list): return [qualify(v, key) for v in value]
    return 'se:' + value if key in KEYS and isinstance(value, str) and value.startswith('SE-') else value
def save(name, value):
    p = OUT / name; p.write_text(yaml.safe_dump(qualify(value), sort_keys=False, allow_unicode=True), encoding='utf-8'); return p
def load(p): return yaml.safe_load(p.read_text(encoding='utf-8'))
fixtures = HERE / 'fixtures'
front = (fixtures/'frontmatter/SE-CLM-0001.md').read_text(encoding='utf-8')
inputs = [('A','KnowledgeCorpus', save('A.yaml',yaml.safe_load(front.split('---',2)[1]))), ('C','KnowledgeCorpus',save('C.yaml',load(fixtures/'standalone-yaml/KNOWLEDGE_CORPUS.yaml')))]
for n, cls, file in [('source','Source','SOURCE.yaml'),('locator','SourceLocator','SOURCE_LOCATOR.yaml'),('evidence','Evidence','EVIDENCE.yaml'),('bundle','ClaimBundle','SE-CLM-0001.yaml')]:
    inputs.append(('B-'+n, cls, save('B-'+n+'.yaml',load(fixtures/'yaml-plus-markdown'/file))))
records = []
for iteration in (1,2):
    for name,cls,path in inputs:
        for fmt in (('ttl','json-ld') if name in ('A','C') else ('ttl',)):
            output = OUT/f'{name}-{iteration}.{fmt}'
            argv = [str(EXE),'-s',str(SCHEMA),'-C',cls,str(path),'-t',fmt,'-o',str(output)]
            print(name, fmt, iteration, flush=True)
            result = subprocess.run(argv,cwd=HERE,env=ENV,capture_output=True,timeout=180)
            records.append({'argv':argv,'cwd':str(HERE),'exit_code':result.returncode,'stdout':result.stdout.decode('utf-8','replace'),'stderr':result.stderr.decode('utf-8','replace')})
            (OUT/'commands.json').write_text(json.dumps(records,indent=2,ensure_ascii=False),encoding='utf-8')
graphs = {}
problems = {}
for n in ('A','C','B-source','B-locator','B-evidence','B-bundle'):
    p = OUT/f'{n}-1.ttl'
    try: graphs[n] = Graph().parse(p,format='turtle') if p.stat().st_size else None
    except Exception as e: problems[n] = repr(e)
if all(graphs.get(n) is not None for n in ('B-source','B-locator','B-evidence','B-bundle')):
    graphs['B'] = Graph()
    for n in ('B-source','B-locator','B-evidence','B-bundle'):
        for triple in graphs[n]: graphs['B'].add(triple)
    graphs['B'].serialize(destination=OUT/'B-merged.ttl',format='turtle')
    p = OUT/'B-merged.ttl'; p.write_bytes(p.read_bytes().rstrip(b'\r\n') + b'\n')
stats = {'triple_counts_before': {n:len(g) for n,g in graphs.items() if g is not None}, 'parse_errors': problems}
trimmed = {}
for n in ('A','B','C'):
    g = graphs.get(n)
    if g is None: continue
    g2 = Graph()
    wrappers = set(s for s,_,t in g.triples((None,RDF.type,None)) if str(t).endswith(('KnowledgeCorpus','ClaimBundle')))
    for s,p,o in g:
        if s not in wrappers: g2.add((s,p,o))
    trimmed[n] = g2
    stats.setdefault('removed_wrapper_subjects',{})[n] = sorted(map(str,wrappers))
stats['triple_counts_after'] = {n:len(g) for n,g in trimmed.items()}
stats['isomorphism'] = {x+'_'+y:isomorphic(trimmed[x],trimmed[y]) if x in trimmed and y in trimmed else None for x,y in [('A','B'),('A','C'),('B','C')]}
stats['repeat_bytes'] = {}
for name,_,_ in inputs:
    for fmt in (('ttl','json-ld') if name in ('A','C') else ('ttl',)):
        p,q = OUT/f'{name}-1.{fmt}',OUT/f'{name}-2.{fmt}'
        stats['repeat_bytes'][name+'_'+fmt] = p.read_bytes()==q.read_bytes() if p.exists() and q.exists() and p.stat().st_size and q.stat().st_size else None
if graphs.get('C'):
    g = graphs['C']; base='https://github.com/kaicreator-mm/tally#'
    for name,term in [('source','SE-SRC-SWEBOK-GUIDE-4-0A'),('locator','SE-LOC-SWEBOK-KA-TABLE'),('evidence','SE-EVD-0001'),('claim','SE-CLM-0001')]:
        s=URIRef(base+term)
        stats.setdefault('instance_samples',{})[name]=[list(map(str,t)) for t in sorted(g.triples((s,None,None)),key=lambda x:tuple(map(str,x)))]
    for n in ('A','C'):
        p=OUT/f'{n}-1.json-ld'
        try:
            j=Graph().parse(p,format='json-ld')
            stats.setdefault('jsonld_graph_vs_ttl',{})[n]=isomorphic(j,graphs[n])
        except Exception as e: stats.setdefault('jsonld_graph_errors',{})[n]=repr(e)
(OUT/'observations.json').write_text(json.dumps(stats,indent=2,ensure_ascii=False),encoding='utf-8')
print(json.dumps(stats,indent=2,ensure_ascii=False),flush=True)
