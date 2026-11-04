"""Generate JSON Schema via stdout and compare independent validation and RDF graphs."""
import json
import os
from pathlib import Path
import subprocess
import sys
import jsonschema
import yaml
from rdflib import Graph
from rdflib.compare import graph_diff, isomorphic, to_isomorphic

HERE=Path(__file__).resolve().parent
EVIDENCE=HERE/'evidence'
OUT=Path(sys.argv[1]).resolve(); OUT.mkdir(parents=True,exist_ok=False)
OLD=EVIDENCE/'windows-312-prefix-fix'; CURIE=EVIDENCE/'windows-312-curie-probe'
exe=Path(sys.executable).parent/'linkml.exe'
schema=HERE/'schema/software-engineering.yaml'
env=dict(os.environ,PYTHONUTF8='1',PYTHONIOENCODING='utf-8')
records=[]
for i in (1,2):
    args=[str(exe),'generate','json-schema',str(schema)]
    r=subprocess.run(args,cwd=HERE,env=env,capture_output=True,timeout=180)
    records.append({'argv':args,'cwd':str(HERE),'exit_code':r.returncode,'stdout':r.stdout.decode('utf-8','replace'),'stderr':r.stderr.decode('utf-8','replace')})
    # Full bytes-as-text remain in commands.json; normalize excess EOF newlines
    # in the convenient checked-in copy for git diff hygiene.
    if r.returncode==0: (OUT/f'schema-{i}.json').write_bytes(r.stdout.rstrip(b'\r\n') + b'\n')
(OUT/'commands.json').write_text(json.dumps(records,indent=2,ensure_ascii=False),encoding='utf-8')
stats={}
if all((OUT/f'schema-{i}.json').exists() for i in (1,2)):
    p=OUT/'schema-1.json'; data=json.loads(p.read_text(encoding='utf-8'))
    jsonschema.Draft202012Validator.check_schema(data)
    validator=jsonschema.validators.validator_for(data)(data)
    for name,path in [('valid_C',HERE/'fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml')]+[(name,OLD/f'{name}.yaml') for name in ('invalid_enum','missing_statement','missing_id','duplicate_id','dangling_locator','wrong_object_kind','duplicate_unique_key')]:
        errors=list(validator.iter_errors(yaml.safe_load(path.read_text(encoding='utf-8'))))
        stats.setdefault('jsonschema_independent',{})[name]=[f'{list(e.absolute_path)}: {e.message}' for e in errors]
    stats['json_schema_repeat_equal']=(OUT/'schema-1.json').read_bytes()==(OUT/'schema-2.json').read_bytes()
for label,p,q in [('schema_rdf',OLD/'schema-default-1.ttl',OLD/'schema-default-2.ttl'),('curie_A_ttl',CURIE/'A-1.ttl',CURIE/'A-2.ttl'),('curie_C_ttl',CURIE/'C-1.ttl',CURIE/'C-2.ttl')]:
    try: stats.setdefault('repeat_graph_isomorphic',{})[label]=isomorphic(Graph().parse(p,format='turtle'),Graph().parse(q,format='turtle'))
    except Exception as e: stats.setdefault('repeat_graph_errors',{})[label]=repr(e)
try:
    ttl=Graph().parse(CURIE/'C-1.ttl',format='turtle')
    jld=Graph().parse(CURIE/'C-1.json-ld',format='json-ld')
    both,onlyttl,onlyjld=graph_diff(to_isomorphic(ttl),to_isomorphic(jld))
    stats['curie_ttl_vs_jsonld']={'ttl':len(ttl),'jsonld':len(jld),'common':len(both),'ttl_only':len(onlyttl),'jsonld_only':len(onlyjld)}
    onlyttl.serialize(destination=OUT/'ttl-only.nt',format='nt')
    onlyjld.serialize(destination=OUT/'jsonld-only.nt',format='nt')
except Exception as e: stats['ttl_jsonld_error']=repr(e)
(OUT/'observations.json').write_text(json.dumps(stats,indent=2,ensure_ascii=False),encoding='utf-8')
print(json.dumps(stats,indent=2,ensure_ascii=False))
