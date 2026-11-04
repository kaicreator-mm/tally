"""Test compatible additive evolution followed by an explicit breaking rename."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import yaml

HERE=Path(__file__).resolve().parent
OUT=Path(sys.argv[1]).resolve(); OUT.mkdir(parents=True,exist_ok=False)
load=lambda p: yaml.safe_load(p.read_text(encoding='utf-8'))
def save(n,x):
    p=OUT/n; p.write_text(yaml.safe_dump(x,sort_keys=False,allow_unicode=True),encoding='utf-8'); return p
original=HERE/'schema/software-engineering.yaml'
old=HERE/'fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml'
schema=load(original)
schema['slots']['review_status']['deprecated']='Replaced by assessment_status.'
schema['slots']['assessment_status']=copy.deepcopy(schema['slots']['review_status'])
schema['slots']['assessment_status'].pop('deprecated')
schema['slots']['assessment_status']['required']=False
schema['classes']['KnowledgeClaim']['slots'].append('assessment_status')
compat=save('compatible-schema.yaml',schema)
new=load(old); new['knowledge_claims'][0]['assessment_status']=new['knowledge_claims'][0]['review_status']
newpath=save('dual-write-record.yaml',new)
renamed=copy.deepcopy(schema); renamed['classes']['KnowledgeClaim']['slots'].remove('review_status'); renamed['slots']['assessment_status']['required']=True
renamedpath=save('renamed-schema.yaml',renamed)
mapped=copy.deepcopy(new); del mapped['knowledge_claims'][0]['review_status']
mappedpath=save('mapped-record.yaml',mapped)
exe=Path(sys.executable).parent/'linkml.exe'; env=dict(os.environ,PYTHONUTF8='1',PYTHONIOENCODING='utf-8')
results=[]
for name,sch,data in [('old_original',original,old),('old_compatible',compat,old),('dual_compatible',compat,newpath),('dual_original',original,newpath),('old_renamed',renamedpath,old),('mapped_renamed',renamedpath,mappedpath),('mapped_original',original,mappedpath)]:
    argv=[str(exe),'validate','-s',str(sch),'--target-class','KnowledgeCorpus',str(data)]
    print(name,flush=True); r=subprocess.run(argv,cwd=HERE,env=env,capture_output=True,timeout=180)
    results.append({'name':name,'argv':argv,'cwd':str(HERE),'exit_code':r.returncode,'stdout':r.stdout.decode('utf-8','replace'),'stderr':r.stderr.decode('utf-8','replace')})
    (OUT/'commands.json').write_text(json.dumps(results,indent=2,ensure_ascii=False),encoding='utf-8')
