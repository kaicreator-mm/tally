/**
 * Fast parse-only guard for Tally v2 UI/Host sources.
 * Uses the pinned TypeScript parser, not Node --check for .ts/.tsx
 * (Node's type stripping does NOT support --check or .tsx).
 * Type errors and runtime behavior are covered by separate gates.
 */
import ts from 'typescript';
import {readFile,readdir} from 'node:fs/promises';
import {relative,resolve,join,extname} from 'node:path';
import {fileURLToPath} from 'node:url';

const webRoot=resolve(fileURLToPath(new URL('..',import.meta.url)));
const repoRoot=resolve(webRoot,'../..');
const roots=[
  'apps/tally-web/src',
  'apps/tally-web/scripts',
  'web/v2',
  'packages/tally-github-native-c/src',
  'packages/tally-github-native-c/test',
  'packages/tally-github-c/src',
  'packages/tally-engineering-business/src',
];
const ignored=new Set(['node_modules','dist','.git','.ads-verify-ci']);
const kinds=new Map([
  ['.ts',ts.ScriptKind.TS],['.tsx',ts.ScriptKind.TSX],
  ['.mjs',ts.ScriptKind.JS],['.js',ts.ScriptKind.JS],
]);

async function collect(root,found){
  for(const entry of await readdir(root,{withFileTypes:true})){
    if(ignored.has(entry.name)||entry.isSymbolicLink())continue;
    const path=join(root,entry.name);
    if(entry.isDirectory())await collect(path,found);
    else if(entry.isFile()&&kinds.has(extname(path)))found.push(path);
  }
}

const files=[];
for(const root of roots)await collect(resolve(repoRoot,root),files);
files.sort();
if(files.length===0)throw new Error('TALLY_V2_SYNTAX_NO_FILES');
let failed=0;
for(const path of files){
  const bytes=await readFile(path,'utf8');
  const parsed=ts.createSourceFile(path,bytes,ts.ScriptTarget.Latest,true,
    kinds.get(extname(path)));
  for(const diag of parsed.parseDiagnostics){
    const pos=parsed.getLineAndCharacterOfPosition(diag.start??0);
    const message=ts.flattenDiagnosticMessageText(diag.messageText,' ');
    process.stderr.write(`${relative(repoRoot,path)}:${pos.line+1}:${pos.character+1} TS${diag.code} ${message}\n`);
    failed++;
  }
}
if(failed){
  process.stderr.write(`TALLY_V2_SYNTAX_PREFLIGHT=FAIL files=${files.length} errors=${failed}\n`);
  process.exitCode=1;
}else process.stdout.write(`TALLY_V2_SYNTAX_PREFLIGHT=PASS files=${files.length}\n`);
