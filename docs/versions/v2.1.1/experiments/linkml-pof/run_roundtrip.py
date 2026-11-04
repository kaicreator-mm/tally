"""Probe schema generation on the baseline and instance conversion round trips."""
import json
import os
from pathlib import Path
import subprocess
import sys
import yaml
HERE=Path(__file__).resolve().parent
OUT=Path(sys.argv[1]).resolve(); OUT.mkdir(parents=True,exist_ok=False)
ROOT=HERE.parents[4]
SCHEMA=HERE/'schema/software-engineering.yaml'
CURIE=HERE/'evidence/windows-312-curie-probe'
env=dict(os.environ,PYTHONUTF8='1',PYTHONIOENCODING='utf-8')
records=[]
def run(name,argv):
    r=subprocess.run(list(map(str,argv)),cwd=HERE,env=env,capture_output=True,timeout=180)
    records.append({'name':name,'argv':list(map(str,argv)),'cwd':str(HERE),'exit_code':r.returncode,'stdout':r.stdout.decode('utf-8','replace'),'stderr':r.stderr.decode('utf-8','replace')})
    (OUT/'commands.json').write_text(json.dumps(records,indent=2,ensure_ascii=False),encoding='utf-8')
    return r
path='docs/versions/v2.1.1/experiments/linkml-pof/schema/software-engineering.yaml'
r=run('baseline_schema_extract',['git','show','a5e438d0f0bebc678e581f4518e17e0aeae68163:'+path])
assert r.returncode==0
base=OUT/'baseline-schema.yaml'; base.write_bytes(r.stdout)
linkml=Path(sys.executable).parent/'linkml.exe'; convert=Path(sys.executable).parent/'linkml-convert.exe'
run('baseline_json_schema_stdout',[linkml,'generate','json-schema',base])
for fmt in ('ttl','json-ld'):
    run(fmt+'_to_yaml',[convert,'-s',SCHEMA,'-C','KnowledgeCorpus','-f',fmt,CURIE/f'C-1.{fmt}','-t','yaml','-o',OUT/f'roundtrip-{fmt}.yaml'])
source=yaml.safe_load((CURIE/'C.yaml').read_text(encoding='utf-8'))
stats={}
for fmt in ('ttl','json-ld'):
    p=OUT/f'roundtrip-{fmt}.yaml'
    stats[fmt+'_equals_input']=yaml.safe_load(p.read_text(encoding='utf-8'))==source if p.exists() and p.stat().st_size else None
(OUT/'observations.json').write_text(json.dumps(stats,indent=2),encoding='utf-8')
print(json.dumps(stats,indent=2))
