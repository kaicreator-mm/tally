import {join} from 'node:path';
import {homedir} from 'node:os';

/** Per-user data directory for initial project state. No secrets in JSON. */
export function resolveTallyConfigDir({platform=process.platform,env=process.env,
  home=homedir()}={}){
  if(env.TALLY_V2_CONFIG_DIR)return env.TALLY_V2_CONFIG_DIR;
  if(platform==='win32')return join(env.LOCALAPPDATA??join(home,'.tally'),'Tally','v2');
  if(platform==='darwin')return join(home,'Library','Application Support','Tally','v2');
  return join(env.XDG_CONFIG_HOME??join(home,'.config'),'tally','v2');
}
