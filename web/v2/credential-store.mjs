/** OS credential vault for the first-run catalog.
 * Windows ciphertext stays in the DPAPI file format. Linux/macOS persist
 * only opaque per-entry native-keyring references, never plaintext PATs.
 * Missing keyrings fail closed rather than downgrading to plaintext storage.
 */
import {randomUUID} from 'node:crypto';
import {spawn} from 'node:child_process';

const REF=/^keyring-v1:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/;
const SAFE_TOKEN=/^[A-Za-z0-9_.-]{8,4096}$/;
const MAC_TOKEN=/^[A-Za-z0-9_.-]{8,2048}$/;
const MAC_SERVICE='com.kaicreator.tally.v2.github';
const TIMEOUT_MS=15000;

/** Bounded native subprocess: no shell interpolation; secrets use stdin. */
export function runCredentialCommand(program,args,input=''){
  return new Promise((resolve,reject)=>{
    const child=spawn(program,args,{stdio:['pipe','pipe','ignore'],windowsHide:true});
    let output='',settled=false;
    const timer=setTimeout(()=>{child.kill();finish(new Error('CREDENTIAL_STORE_UNAVAILABLE'));},TIMEOUT_MS);
    timer.unref?.();
    const finish=(error,value)=>{
      if(settled)return;settled=true;clearTimeout(timer);
      if(error)reject(error);else resolve(value);
    };
    child.once('error',()=>finish(new Error('CREDENTIAL_STORE_UNAVAILABLE')));
    child.stdout?.on('data',chunk=>{
      output+=chunk.toString('utf8');
      if(output.length>8192){child.kill();finish(new Error('CREDENTIAL_STORE_UNAVAILABLE'));}
    });
    child.once('close',code=>{
      if(code!==0)return finish(new Error('CREDENTIAL_STORE_UNAVAILABLE'));
      finish(null,output.replace(/\r?\n$/,''));
    });
    child.stdin?.on('error',()=>{}); // EPIPE is handled by process close.
    child.stdin?.end(input);
  });
}
function runPowershell(script,input){
  if(process.platform!=='win32')throw new Error('WINDOWS_CREDENTIAL_STORE_REQUIRED');
  const encoded=Buffer.from(script,'utf16le').toString('base64');
  return runCredentialCommand('powershell.exe',
    ['-NoProfile','-NonInteractive','-EncodedCommand',encoded],input);
}
/** Preserve Windows CurrentUser DPAPI ciphertext, including old credentials. */
export const windowsDpapi=Object.freeze({
  seal:secret=>runPowershell(
    '$v=[Console]::In.ReadToEnd(); $s=ConvertTo-SecureString $v -AsPlainText -Force; ConvertFrom-SecureString $s',secret),
  open:cipher=>runPowershell(
    '$v=[Console]::In.ReadToEnd(); $s=ConvertTo-SecureString $v; [Runtime.InteropServices.Marshal]::PtrToStringBSTR([Runtime.InteropServices.Marshal]::SecureStringToBSTR($s))',cipher),
});

/** CredentialStore: seal(secret)->opaque reference, open(ref)->secret.
 * The catalog writes only this reference to its 0600 filesystem entry.
 */
export function createCredentialStore({platform=process.platform,
  run=runCredentialCommand,idFactory=randomUUID}={}){
  if(platform==='win32')return windowsDpapi;
  if(platform!=='linux'&&platform!=='darwin')
    return Object.freeze({
      async seal(){throw new Error('CREDENTIAL_PLATFORM_UNSUPPORTED');},
      async open(){throw new Error('CREDENTIAL_PLATFORM_UNSUPPORTED');},
    });
  const exec=async(program,args,input)=>{
    try{return await run(program,args,input);}catch{
      throw new Error('CREDENTIAL_STORE_UNAVAILABLE');
    }
  };
  return Object.freeze({
    async seal(secret){
      if(typeof secret!=='string'||!(platform==='darwin'?MAC_TOKEN:SAFE_TOKEN).test(secret))
        throw new Error('CREDENTIAL_FORMAT_UNSUPPORTED');
      const id=idFactory();
      if(!REF.test('keyring-v1:'+id))throw new Error('CREDENTIAL_REFERENCE_INVALID');
      if(platform==='linux'){
        await exec('secret-tool',['store','--label=Tally v2 GitHub token',
          'application','tally-v2','credential',id],secret);
      }else{
        // Normal macOS -w exposes the PAT in argv; batch -i takes commands
        // via stdin, with a restricted single-line GitHub PAT character set.
        const command='add-generic-password -U -s '+MAC_SERVICE+
          ' -a '+id+' -w '+secret+'\nquit\n';
        await exec('security',['-i'],command);
      }
      return 'keyring-v1:'+id;
    },
    async open(reference){
      const id=typeof reference==='string'?REF.exec(reference)?.[1]:null;
      if(!id)throw new Error('CREDENTIAL_REFERENCE_INVALID');
      const value=platform==='linux'?
        await exec('secret-tool',['lookup','application','tally-v2','credential',id]):
        await exec('security',['find-generic-password','-s',MAC_SERVICE,'-a',id,'-w']);
      if(typeof value!=='string'||!SAFE_TOKEN.test(value))
        throw new Error('CREDENTIAL_UNAVAILABLE');
      return value;
    },
  });
}
export const nativeCredentialStore=createCredentialStore();
