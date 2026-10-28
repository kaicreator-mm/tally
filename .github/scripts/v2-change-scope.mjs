/**
 * Tally v2 migration: non-authoritative, reproducible CI change-scope diagnosis.
 * Source source only: never changes GitHub check conclusions, required status,
 * Reviewer/ADS gates, v1.5 maintenance or Release qualification.
 */
import {execFileSync} from 'node:child_process';
import {writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';

const SHA=/^[0-9a-f]{40}$/;
const fail=(reason)=>Object.freeze({
  classification:'UNKNOWN_SCOPE',reason,sourceQualified:false,
  goSourceAffected:'UNKNOWN',v2NodeAffected:'UNKNOWN',
  docsOnly:false,requiredCheckApplicability:'NOT_DETERMINED',
  existingProviderFailureWaived:false,mergeAuthorized:false,
  releaseAuthorized:false
});
const validPath=p=>typeof p==='string'&&p.length>0&&p.length<=4096&&
  !p.includes('\0')&&!p.startsWith('/')&&!p.startsWith('../')&&
  !p.includes('/../')&&!p.includes('\\');
const isV2=p=>p.startsWith('experiments/v2/')||
  /^packages\/tally-[^/]+\//.test(p)||
  /^\.github\/workflows\/v2-[^/]+\.ya?ml$/.test(p);
const isGo=p=>/\.go$/.test(p)||/^go\.(mod|sum|work|work\.sum)$/.test(p)||
  /^(cmd|internal|runtimes|test|scripts)\//.test(p)||
  /^Makefile$/.test(p)||p==='.github/workflows/quality.yml';
const isDocs=p=>p.startsWith('docs/')||p.startsWith('README')||
  /\.(?:md|mdx)$/.test(p);

/**
 * Hints do not replace required checks. Unknown or mixed inputs don't qualify
 * as a narrower legacy Go scope, even if caller asserts "docs-only".
 */
export function classifyChangedFiles(files){
  if(!Array.isArray(files)||files.length===0||files.length>10000||
     files.some(p=>!validPath(p))||new Set(files).size!==files.length)
    return fail('CHANGED_PATH_SET_UNVERIFIED');
  const sorted=[...files].sort();
  const go=sorted.some(isGo);
  const node=sorted.some(isV2);
  const docs=sorted.every(isDocs);
  const other=sorted.some(p=>!isGo(p)&&!isV2(p)&&!isDocs(p));
  const classification=go?'GO_OR_MIXED_CHANGE':
    node?(other?'MIXED_V2_AND_UNKNOWN':'V2_NODE_OR_SDK_SOURCE'):
    docs?'DOCS_ONLY':'OTHER_OR_UNKNOWN';
  return Object.freeze({
    classification,reason:'SOURCE_DIFF_CLASSIFIED',
    sourceQualified:true,changedCount:files.length,
    goSourceAffected:go,v2NodeAffected:node,docsOnly:docs,
    unknownOrOtherPaths:other,
    // Independent owner still has to legally decide the *required* profile.
    requiredCheckApplicability:'NOT_DETERMINED',
    existingProviderFailureWaived:false,mergeAuthorized:false,
    releaseAuthorized:false
  });
}
export function classifyCommitDiff(base,head){
  if(!SHA.test(base??'')||!SHA.test(head??''))return fail('INVALID_SHA_PAIR');
  try{
    // Both SHAs are validated above; no interpolated shell and no network.
    // Do NOT let Git rename detection hide a removed Go source behind a
    // docs-only destination (or the reverse): include both paths.
    const raw=execFileSync('git',
      ['diff','--no-renames','--name-only','-z','--diff-filter=ACDMRT',base,head],
      {maxBuffer:2*1024*1024,encoding:'utf8',timeout:10000});
    return classifyChangedFiles(raw.endsWith('\0')?
      raw.slice(0,-1).split('\0'):[raw]);
  }catch{return fail('GIT_DIFF_UNAVAILABLE');}
}
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
  const [base,head]=process.argv.slice(2);
  const output=classifyCommitDiff(base,head);
  process.stdout.write(JSON.stringify(output)+'\n');
  if(process.env.GITHUB_STEP_SUMMARY){
    const data=[
      '### Tally v2 CI source-scope diagnostic (non-authoritative)',
      '',
      '| Key | Source-derived value |',
      '| --- | --- |',
      ...['classification','sourceQualified','changedCount','goSourceAffected',
        'v2NodeAffected','docsOnly','unknownOrOtherPaths',
        'requiredCheckApplicability','existingProviderFailureWaived',
        'mergeAuthorized','releaseAuthorized'].map(k=>
          `| ${k} | ${String(output[k]??'UNKNOWN')} |`),
      '',
      '**Original GitHub job results remain authoritative.** This report cannot waive red legacy Go jobs or authorize a v2 version/main merge, native Claim, or Release.',
      ''
    ].join('\n');
    writeFileSync(process.env.GITHUB_STEP_SUMMARY,data,{flag:'a'});
  }
  if(!output.sourceQualified)process.exitCode=1;
}
