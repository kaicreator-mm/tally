/** Read-only, fail-closed ADS v4.18 terminal event observation for #203. */
import Ajv2020 from 'ajv/dist/2020.js';
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import {isDeepStrictEqual} from 'node:util';
import {parseDocument,visit,isAlias} from 'yaml';

export const TARGET=Object.freeze({
  repository:'kaicreator-mm/tally',issueNumber:203,
  issueRef:'https://github.com/kaicreator-mm/tally/issues/203',
  sourceSha:'00c03e1f0c2f66c6ae6df1ead0ed1ce5d9914769',
  dispatchId:'dispatch:203:candidate-publication:20261031:1',
  protectedClaimKey:'kaicreator-mm/tally#203:builder:__default__',generation:1,
  claimCommentId:6540484121,
  claimCommentRef:'https://github.com/kaicreator-mm/tally/issues/203#issuecomment-6540484121',
  casKey:'tally:v2:claim-admission:6fcd893012c987f014c78de5b444bd16db522a6137a568d9de0e98a6af6401c4'
});
const bytes=readFileSync(new URL('../vendor/agent-event-v2.schema.json',import.meta.url));
const blob=createHash('sha1').update(Buffer.from(`blob ${bytes.length}\0`)).update(bytes).digest('hex');
if(blob!=='cad9434e11febfd29657b564dbe710700c674888')throw Error('ADS_V418_EVENT_SCHEMA_BLOB_MISMATCH');
const validate=new Ajv2020({strict:false,allErrors:true,validateFormats:false}).compile(JSON.parse(bytes));
const marker='<!-- ads:github -->';
const unknown=reason=>Object.freeze({status:'UNKNOWN',reason,authorized:false,admitted:false,taskPass:false,releasePass:false});
function parse(body){
  if(typeof body!=='string'||body.length>262144)return {bad:true};
  const lines=body.split(/\r\n|\n|\r/), marks=lines.filter(x=>x.trim()===marker);
  if(!marks.length)return {};
  if(lines[0]!==marker||marks.length!==1)return {bad:true};
  let i=1;while(lines[i]?.trim()==='')i++;
  if(lines[i]!=='```yaml')return {bad:true};
  const end=lines.indexOf('```',i+1);
  if(end<0)return {bad:true};
  try{
    const doc=parseDocument(lines.slice(i+1,end).join('\n'),{uniqueKeys:true,merge:false,schema:'core',maxAliasCount:0,strict:true});
    if(doc.errors.length||doc.warnings.length)throw Error('YAML');
    visit(doc,{Node(_key,node){if(isAlias(node)||node.anchor!==undefined||node.tag!==undefined)throw Error('YAML');}});
    const e=doc.toJS({maxAliasCount:0});
    if(!e||typeof e!=='object'||Array.isArray(e)||Object.keys(e).sort().join(',')!=='data,kind,schema'||
      e.schema!=='ads/github/v3'||e.kind!=='event'||!validate(e.data))return {bad:true};
    return {event:e.data};
  }catch{return {bad:true};}
}

/** An absent result requires a stable issue and all comments, not a partial scan. */
export async function readDispatchStateEventsFromClient({octokit,writerLogin,expectedEvent=null,limit=1000}={}){
  if(!/^[A-Za-z0-9_.-]+(?:\[bot\])?$/.test(writerLogin??'')||
    !Number.isSafeInteger(limit)||limit<1||limit>1000||
    typeof octokit?.rest?.issues?.get!=='function'||typeof octokit?.rest?.issues?.listComments!=='function'||
    typeof octokit?.rest?.issues?.getComment!=='function'||typeof octokit?.rest?.repos?.getCommit!=='function'||
    typeof octokit?.paginate?.iterator!=='function')return unknown('INVALID_TERMINAL_QUERY');
  const params={owner:'kaicreator-mm',repo:'tally',issue_number:203};
  try{
    const [initial,commit]=await Promise.all([octokit.rest.issues.get(params),
      octokit.rest.repos.getCommit({owner:params.owner,repo:params.repo,ref:TARGET.sourceSha})]);
    const issue=initial?.data;
    if(issue?.number!==203||issue.html_url!==TARGET.issueRef||issue.state!=='open'||
      !Number.isSafeInteger(issue.id)||!Number.isSafeInteger(issue.comments)||issue.comments<0||
      typeof issue.updated_at!=='string'||typeof issue.body!=='string'||
      !issue.body.includes(`version/v2@${TARGET.sourceSha}`)||
      commit?.data?.sha!==TARGET.sourceSha)return unknown('ISSUE_OR_SOURCE_STALE');
    let count=0;const seen=new Set(),events=[];
    for await(const page of octokit.paginate.iterator(octokit.rest.issues.listComments,{...params,per_page:100})){
      if(!Array.isArray(page?.data))return unknown('INCOMPLETE_PAGINATION');
      for(const c of page.data){
        if(++count>limit||!Number.isSafeInteger(c?.id)||seen.has(c.id))return unknown('INCOMPLETE_PAGINATION');
        seen.add(c.id);
        const p=parse(c.body);if(p.bad)return unknown('MALFORMED_ADS_EVENT');
        if(p.event?.event!=='DISPATCH_STATE_CHANGED')continue;
        if(c.html_url!==`${TARGET.issueRef}#issuecomment-${c.id}`||
          c.issue_url!=='https://api.github.com/repos/kaicreator-mm/tally/issues/203'||
          c.user?.login!==writerLogin||typeof c.created_at!=='string'||c.created_at!==c.updated_at)
          return unknown('TERMINAL_EVENT_IDENTITY_UNVERIFIED');
        events.push({comment:c,event:p.event});
      }
    }
    const final=(await octokit.rest.issues.get(params))?.data;
    if(count!==issue.comments||final?.id!==issue.id||final?.state!=='open'||final.comments!==issue.comments||
      final.updated_at!==issue.updated_at||final.body!==issue.body||final.html_url!==TARGET.issueRef)return unknown('INCOMPLETE_OR_CHANGED_ISSUE');
    if(!expectedEvent)return events.length?unknown('TERMINAL_EVENT_ALREADY_EXISTS'):
      Object.freeze({status:'ABSENT',complete:true});
    if(events.length!==1||!isDeepStrictEqual(events[0].event,expectedEvent))return unknown('TERMINAL_EVENT_NOT_UNIQUE_OR_MISMATCH');
    const c=events[0].comment;
    const current=(await octokit.rest.issues.getComment({...params,comment_id:c.id}))?.data;
    if(!current||current.id!==c.id||current.body!==c.body||current.html_url!==c.html_url||
      current.issue_url!==c.issue_url||current.user?.login!==writerLogin||
      current.created_at!==c.created_at||current.updated_at!==c.updated_at)
      return unknown('TERMINAL_EVENT_CHANGED');
    return Object.freeze({status:'EXACT',complete:true,edited:false,commentId:c.id,commentRef:c.html_url});
  }catch{return unknown('TERMINAL_EVENT_READ_UNAVAILABLE');}
}
