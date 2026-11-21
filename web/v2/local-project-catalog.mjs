/** Local first-run Project/Version catalog. No GitHub write privileges. */
import {mkdir,readFile,rename,writeFile} from 'node:fs/promises';
import {join} from 'node:path';
import {createHash,randomUUID} from 'node:crypto';
import {nativeCredentialStore,windowsDpapi} from './credential-store.mjs';
export {windowsDpapi};
import {isValidVersionRef} from '../../packages/tally-github-native-c/src/version-ref.mjs';

const REPOSITORY=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const safeId=repo=>createHash('sha256').update(repo).digest('hex');
const copy=x=>JSON.parse(JSON.stringify(x));
const valid=x=>x&&Array.isArray(x.projects)&&x.projects.length<=64&&
  x.projects.every(p=>REPOSITORY.test(p.repository)&&
    Array.isArray(p.versions)&&p.versions.length>0&&p.versions.length<=32&&
    p.versions.every(v=>isValidVersionRef(v.ref)&&SHA.test(v.sha)))&&
  (x.selected===null||x.selected&&REPOSITORY.test(x.selected.repository)&&
    isValidVersionRef(x.selected.ref));

export async function readGithubHead({repository,ref,token,fetchImpl=fetch}){
  if(!REPOSITORY.test(repository)||!isValidVersionRef(ref)||
     typeof token!=='string'||token.length<8)
    throw new Error('INVALID_SELECTED_GITHUB_SOURCE');
  const [owner,name]=repository.split('/');
  const url='https://api.github.com/repos/'+encodeURIComponent(owner)+'/'+
    encodeURIComponent(name)+'/branches/'+encodeURIComponent(ref);
  let response;
  try{response=await fetchImpl(url,{headers:{
    authorization:'Bearer '+token,accept:'application/vnd.github+json',
    'user-agent':'tally-v2-first-run'},signal:AbortSignal.timeout(15000)});}
  catch{throw new Error('GITHUB_SOURCE_UNAVAILABLE');}
  if(!response.ok)throw new Error('GITHUB_SOURCE_UNAVAILABLE');
  let body;
  try{body=await response.json();}catch{throw new Error('GITHUB_RESPONSE_INVALID');}
  const sha=body?.commit?.sha;
  if(!SHA.test(sha??''))throw new Error('GITHUB_BRANCH_SHA_INVALID');
  return sha;
}
const empty=()=>({schema:1,projects:[],selected:null});
export class LocalProjectCatalog{
  constructor({dir,vault=nativeCredentialStore,readHead=readGithubHead}={}){
    if(typeof dir!=='string'||!dir)throw new Error('PROJECT_STORE_DIR_REQUIRED');
    this.dir=dir;this.vault=vault;this.readHead=readHead;
  }
  async load(){
    try{
      const data=JSON.parse(await readFile(join(this.dir,'projects.json'),'utf8'));
      if(data.schema!==1||!valid(data))throw new Error('PROJECT_STORE_CORRUPT');
      return data;
    }catch(error){
      if(error?.code==='ENOENT')return empty();
      throw new Error('PROJECT_STORE_CORRUPT');
    }
  }
  async save(data){
    if(!valid(data))throw new Error('PROJECT_STORE_INVALID');
    await mkdir(this.dir,{recursive:true,mode:0o700});
    const temp=join(this.dir,'.projects-'+randomUUID()+'.tmp');
    await writeFile(temp,JSON.stringify(data,null,2),{flag:'wx',mode:0o600});
    await rename(temp,join(this.dir,'projects.json'));
  }
  credentialPath(repository){
    return join(this.dir,'credential-'+safeId(repository)+
      (process.platform==='win32'?'.dpapi':'.keyring'))
  }
  async readToken(repository){
    const secret=await readFile(this.credentialPath(repository),'utf8');
    const token=await this.vault.open(secret.trim());
    if(typeof token!=='string'||token.length<8)
      throw new Error('CREDENTIAL_UNAVAILABLE');
    return token;
  }
  async add({repository,ref,token}){
    if(!REPOSITORY.test(repository??'')||!isValidVersionRef(ref))
      throw new Error('INVALID_SELECTED_GITHUB_SOURCE');
    const data=await this.load();
    const existing=data.projects.find(p=>p.repository===repository);
    if(!existing&&(!token||token.length<8))throw new Error('GITHUB_TOKEN_REQUIRED');
    if(data.projects.length>=64&&!existing)throw new Error('PROJECT_LIMIT');
    if(existing&&existing.versions.length>=32&&
       !existing.versions.some(v=>v.ref===ref))throw new Error('VERSION_LIMIT');
    const credential=token??await this.readToken(repository);
    const sha=await this.readHead({repository,ref,token:credential});
    const cipher=token?await this.vault.seal(token):null;
    if(token){
      await mkdir(this.dir,{recursive:true,mode:0o700});
      const tmp=this.credentialPath(repository)+'.'+randomUUID();
      await writeFile(tmp,cipher,{flag:'wx',mode:0o600});
      await rename(tmp,this.credentialPath(repository));
    }
    const project=existing??{repository,title:repository.split('/')[1],versions:[]};
    if(!existing)data.projects.push(project);
    const version=project.versions.find(v=>v.ref===ref);
    if(version)version.sha=sha;
    else project.versions.push({ref,sha});
    data.selected={repository,ref};
    await this.save(data);
    return this.publicStatus(data);
  }
  async select({repository,ref}){
    const data=await this.load();
    const project=data.projects.find(p=>p.repository===repository);
    const version=project?.versions.find(v=>v.ref===ref);
    if(!version)throw new Error('UNKNOWN_PROJECT_VERSION');
    const token=await this.readToken(repository);
    const sha=await this.readHead({repository,ref,token});
    version.sha=sha;
    data.selected={repository,ref};
    await this.save(data);
    return {env:{
      TALLY_V2_REPOSITORY:repository,TALLY_V2_VERSION_REF:ref,
      TALLY_V2_VERSION_SHA:sha,TALLY_GITHUB_READ_TOKEN:token},
      status:this.publicStatus(data)};
  }
  async selected(){
    const data=await this.load();
    return data.selected?this.select(data.selected):null;
  }
  publicStatus(data){
    return copy({projects:data.projects,selected:data.selected,
      configured:data.projects.length>0});
  }
  async status(){return this.publicStatus(await this.load());}
}
