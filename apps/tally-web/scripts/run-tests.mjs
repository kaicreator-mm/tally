/** Portable Node >=24 test discovery, independent of shell glob expansion.
 * Includes src/adapters/*.test.ts and nested feature tests on Windows/Linux.
 */
import {readdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import {dirname,join,relative} from 'node:path';
import {spawnSync} from 'node:child_process';

const root=join(dirname(fileURLToPath(import.meta.url)),'..');
const walk=async directory=>{
  const entries=await readdir(directory,{withFileTypes:true});
  const paths=[];
  for(const entry of entries){
    const path=join(directory,entry.name);
    if(entry.isDirectory())paths.push(...await walk(path));
    else if(entry.isFile()&&entry.name.endsWith('.test.ts'))
      paths.push(path);
  }
  return paths;
};
const files=(await walk(join(root,'src'))).sort();
if(files.length===0){
  process.stderr.write('TALLY_WEB_NO_TESTS_DISCOVERED\n');
  process.exitCode=1;
}else{
  process.stdout.write(`TALLY_WEB_TEST_FILES=${files.length}\n`);
  for(const path of files)process.stdout.write(relative(root,path)+'\n');
  const result=spawnSync(process.execPath,['--test',...files],
    {cwd:root,stdio:'inherit',shell:false});
  if(result.error)throw result.error;
  process.exitCode=result.status??1;
}
