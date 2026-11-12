/**
 * J01 Project intake — read-only canonical GitHub Project/Requirement facts.
 *
 * Pure read-model in the style of report-projection: no HTTP by itself, no
 * GitHub write, no Kernel authority, no Task/Release verdict. The transport
 * entry reads pinned Requirement sources and optional Projects v2 through an
 * injected selected C client; every deviation fails closed to UNKNOWN with a
 * bounded reason. Observed output stays distinct from any workbench draft.
 */
const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[0-9a-f]{40}$/;
const TITLE=/^.{1,256}$/s;
const ISO=value=>typeof value==='string'&&Number.isFinite(Date.parse(value))&&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
const MAX_SOURCES=16,MAX_PROJECTS=8,MAX_ITEMS=100,MAX_ITEM_COUNT=10000;
const freeze=value=>Object.freeze(value);
const unknown=(reason,extra={})=>freeze({status:'UNKNOWN',reason,sourceQualified:false,
  repository:null,observedAt:null,currentness:'AGE_NOT_VERIFIED',
  sources:[],projects:[],authorized:false,admitted:false,taskPass:false,
  releasePass:false,...extra});
const errorReason=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===429?'GITHUB_RATE_LIMITED':'PROJECT_INTAKE_READ_FAILED';

/** Validate one raw requirement/PR source record captured by the transport. */
function validSource(raw,owner,repo){
  const url=raw?.url;
  return !!raw&&typeof raw==='object'&&!Array.isArray(raw)&&
    Number.isSafeInteger(raw.number)&&raw.number>0&&
    ['open','closed'].includes(raw.state)&&TITLE.test(raw.title??'')&&
    (url===`https://github.com/${owner}/${repo}/issues/${raw.number}`||
      url===`https://github.com/${owner}/${repo}/pull/${raw.number}`);
}
function validProject(raw,owner,repo){
  return raw&&typeof raw==='object'&&!Array.isArray(raw)&&
    Number.isSafeInteger(raw.number)&&raw.number>0&&TITLE.test(raw.title??'')&&
    typeof raw.url==='string'&&raw.url===`https://github.com/${owner}/${repo}/projects/${raw.number}`&&
    typeof raw.closed==='boolean'&&Number.isSafeInteger(raw.itemCount)&&
    raw.itemCount>=0&&raw.itemCount<=MAX_ITEM_COUNT&&Array.isArray(raw.items)&&
    raw.items.length<=MAX_ITEMS&&raw.items.every(item=>item&&typeof item==='object'&&!Array.isArray(item)&&
      Number.isSafeInteger(item.issueNumber)&&item.issueNumber>0&&
      TITLE.test(item.title??'')&&['open','closed'].includes(item.state));
}
/**
 * Bounded projection of one transport observation. Positive output marks only
 * canonical GitHub facts observed at query completion; it never admits a
 * Claim, freezes content or passes a Task/Release.
 */
export function projectIntakeReadModel(observation){
  try{
    const o=observation;
    if(!o||typeof o!=='object'||Array.isArray(o))return unknown('INVALID_PROJECT_INTAKE_REQUEST');
    const {repository,observedAt,sources,projects}=o;
    if(!repository||typeof repository!=='object'||Array.isArray(repository)||
      !NAME.test(repository.owner??'')||!NAME.test(repository.repo??'')||
      repository.fullName!==`${repository.owner}/${repository.repo}`||
      typeof repository.defaultBranch!=='string'||repository.defaultBranch.length===0||
      !['public','private'].includes(repository.visibility))
      return unknown('PROJECT_INTAKE_REPOSITORY_UNVERIFIED');
    if(!ISO(observedAt))return unknown('PROJECT_INTAKE_OBSERVATION_TIME_UNAVAILABLE');
    if(!Array.isArray(sources)||sources.length>MAX_SOURCES||
      !sources.every(raw=>validSource(raw,repository.owner,repository.repo)))
      return unknown('PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED');
    if(!Array.isArray(projects)||projects.length>MAX_PROJECTS||
      !projects.every(raw=>validProject(raw,repository.owner,repository.repo)))
      return unknown('PROJECT_INTAKE_PROJECT_FACTS_UNVERIFIED');
    const projectedSources=sources.map(raw=>freeze({number:raw.number,state:raw.state,
      title:raw.title,url:raw.url,isPullRequest:raw.isPullRequest===true,
      merged:raw.merged===true}));
    const projectedProjects=projects.map(raw=>{
      const items=raw.items.map(item=>freeze({issueNumber:item.issueNumber,
        title:item.title,state:item.state}));
      return freeze({number:raw.number,title:raw.title,url:raw.url,closed:raw.closed,
        itemCount:raw.itemCount,items:freeze(items)});
    });
    return freeze({status:'PROJECT_INTAKE_OBSERVED',reason:'CANONICAL_GITHUB_FACTS_OBSERVED',
      repository:freeze({owner:repository.owner,repo:repository.repo,
        fullName:repository.fullName,defaultBranch:repository.defaultBranch,
        visibility:repository.visibility}),
      observedAt,currentness:'OBSERVED_AT_QUERY_COMPLETION',
      sources:freeze(projectedSources),
      projects:freeze(projectedProjects),
      sourceQualified:true,authorized:false,admitted:false,
      taskPass:false,releasePass:false});
  }catch{return unknown('PROJECT_INTAKE_READ_UNREADABLE');}
}
const PROJECTS_QUERY=`query($owner:String!,$name:String!,$first:Int!){
repository(owner:$owner,name:$name){nameWithOwner
  projectsV2(first:$first,orderBy:{field:NUMBER,direction:ASC}){nodes{
    number title url closed
    items(first:$first){totalCount nodes{content{__typename
      ... on Issue{number state title}}}}}}}}`;
/**
 * Read-only transport: pinned requirement sources plus optional Projects v2.
 * The caller supplies a selected C client (Octokit-compatible); this module
 * holds no credential and performs no mutation. projectTitle is matched by
 * exact title readback so the Host can verify a canonical target project.
 */
export async function readProjectIntakeFromClient({octokit,owner,repo,
  expectedSourceSha,sources=[],projectTitle,limit=100}={}){
  if(!octokit?.rest?.repos?.get||!NAME.test(owner??'')||!NAME.test(repo??'')||
    !SHA.test(expectedSourceSha??'')||!Number.isSafeInteger(limit)||limit<1||limit>100||
    !Array.isArray(sources)||sources.length>MAX_SOURCES||
    !sources.every(s=>s&&Number.isSafeInteger(s.number)&&s.number>0)||
    projectTitle!==undefined&&!TITLE.test(projectTitle??''))
    return unknown('INVALID_PROJECT_INTAKE_REQUEST');
  try{
    const repoData=(await octokit.rest.repos.get({owner,repo}))?.data;
    if(repoData?.full_name!==`${owner}/${repo}`||
      typeof repoData?.default_branch!=='string'||
      !['public','private'].includes(repoData?.visibility))
      return unknown('PROJECT_INTAKE_REPOSITORY_UNVERIFIED');
    const observedSources=[];
    for(const {number} of sources){
      const data=(await octokit.rest.issues.get({owner,repo,issue_number:number}))?.data;
      if(!data||data.number!==number||!TITLE.test(data.title??'')||
        !['open','closed'].includes(data.state)||
        typeof data.html_url!=='string')
        return unknown('PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED');
      observedSources.push({number,state:data.state,title:data.title,url:data.html_url,
        isPullRequest:data.pull_request!==undefined,
        merged:data.pull_request?.merged_at!=null});
    }
    const projects=[];
    if(projectTitle!==undefined){
      if(typeof octokit.graphql!=='function')
        return unknown('PROJECT_INTAKE_READ_FAILED');
      const data=await octokit.graphql(PROJECTS_QUERY,{owner,name:repo,first:limit});
      if(data?.repository?.nameWithOwner!==`${owner}/${repo}`||
        !Array.isArray(data.repository.projectsV2?.nodes))
        return unknown('PROJECT_INTAKE_PROJECT_FACTS_UNVERIFIED');
      for(const node of data.repository.projectsV2.nodes){
        if(node?.title!==projectTitle)continue;
        const items=(node.items?.nodes??[]).flatMap(entry=>
          entry?.content?.__typename==='Issue'&&Number.isSafeInteger(entry.content.number)?
            [{issueNumber:entry.content.number,title:entry.content.title,
              state:entry.content.state}]:[]);
        projects.push({number:node.number,title:node.title,url:node.url,
          closed:node.closed===true,itemCount:node.items?.totalCount??items.length,items});
      }
    }
    return projectIntakeReadModel({repository:{owner,repo,fullName:repoData.full_name,
      defaultBranch:repoData.default_branch,visibility:repoData.visibility},
      observedAt:new Date().toISOString(),sources:observedSources,projects});
  }catch(e){return unknown(errorReason(e));}
}


/**
 * #312 source-bound Product ReviewWitness reader. Only an official GitHub
 * Issue comment can supply evidence; accepted role Claims come from Host.
 * Caller-authored JSON, same-process Review and edited comments fail closed.
 */
const RW_SHA=/^[a-f0-9]{40}$/,RW_DIGEST=/^[a-f0-9]{64}$/;
const RW_KEYS=['schema','status','repository','issueNumber','reviewRef','draftRef',
  'productDigest','sourceSha','taskPackBlobSha','writeSetSha256',
  'reviewedHead','reviewedTree','reviewedBase','builderClaimRef',
  'reviewerClaimRef','reviewerOperatorId','reviewerWriterId','p0','p1','p2'];
const rwUnknown=reason=>Object.freeze({status:'UNKNOWN',reason,
  independentlyVerified:false,sourceQualified:false,reviewRef:null,
  authorized:false,taskPass:false,releasePass:false});
const rwText=x=>typeof x==='string'&&x.length>0&&x.length<=256;
const rwClaim=(c,role,e)=>c?.accepted===true&&c.issueNumber===312&&
  c.role===role&&c.sourceSha===e.sourceSha&&
  c.taskPackBlobSha===e.taskPackBlobSha&&c.writeSetSha256===e.writeSetSha256&&
  rwText(c.claimRef)&&rwText(c.operatorId)&&rwText(c.writerId)&&
  rwText(c.transportActor);
const rwExpected=e=>e?.issueNumber===312&&RW_SHA.test(e.sourceSha??'')&&
  RW_SHA.test(e.taskPackBlobSha??'')&&RW_DIGEST.test(e.writeSetSha256??'')&&
  RW_DIGEST.test(e.productDigest??'')&&RW_SHA.test(e.reviewedHead??'')&&
  RW_SHA.test(e.reviewedTree??'')&&RW_SHA.test(e.reviewedBase??'')&&
  e.draftRef==='draft:sha256:'+e.productDigest;
/**
 * Host must independently enforce persisted draft, current version ref and
 * shared CAS fencing before any FROZEN_PRODUCT_INTAKE write.
 */
export async function readProductReviewWitnessFromClient({
  octokit,owner='kaicreator-mm',repo='tally',commentId,expected,
  builderClaim,reviewerClaim
}={}){
  if(owner!=='kaicreator-mm'||repo!=='tally'||!rwExpected(expected)||
    !Number.isSafeInteger(commentId)||commentId<=0||
    typeof octokit?.rest?.issues?.getComment!=='function')
    return rwUnknown('PRODUCT_REVIEW_INPUT_INVALID');
  if(!rwClaim(builderClaim,'builder',expected)||
    !rwClaim(reviewerClaim,'reviewer',expected))
    return rwUnknown('PRODUCT_REVIEW_CLAIM_UNVERIFIED');
  if(builderClaim.operatorId===reviewerClaim.operatorId||
    builderClaim.writerId===reviewerClaim.writerId)
    return rwUnknown('PRODUCT_REVIEW_SAME_PROCESS');
  let comment;
  try{
    comment=(await octokit.rest.issues.getComment({
      owner,repo,comment_id:commentId}))?.data;
  }catch(error){return rwUnknown(errorReason(error));}
  const ref='https://github.com/'+owner+'/'+repo+'/issues/312#issuecomment-'+commentId;
  if(comment?.id!==commentId||comment.html_url!==ref||
    comment.user?.login!==reviewerClaim.transportActor||
    !ISO(comment.created_at)||comment.created_at!==comment.updated_at||
    typeof comment.body!=='string'||comment.body.length>8192)
    return rwUnknown('PRODUCT_REVIEW_COMMENT_UNVERIFIED');
  let body;
  try{
    body=JSON.parse(comment.body);
    if(!body||typeof body!=='object'||Array.isArray(body)||
      JSON.stringify(body)!==comment.body.trim())
      return rwUnknown('PRODUCT_REVIEW_JSON_NONCANONICAL');
  }catch{return rwUnknown('PRODUCT_REVIEW_JSON_NONCANONICAL');}
  if(Object.keys(body).length!==RW_KEYS.length||
    RW_KEYS.some(k=>!Object.hasOwn(body,k)))
    return rwUnknown('PRODUCT_REVIEW_SCHEMA_INVALID');
  if(body.schema!=='tally.v2.product-review-witness/1'||
    body.status!=='INDEPENDENTLY_VERIFIED'||body.repository!==owner+'/'+repo||
    body.issueNumber!==312||body.reviewRef!==ref||
    body.builderClaimRef!==builderClaim.claimRef||
    body.reviewerClaimRef!==reviewerClaim.claimRef||
    body.reviewerOperatorId!==reviewerClaim.operatorId||
    body.reviewerWriterId!==reviewerClaim.writerId||
    ['draftRef','productDigest','sourceSha','taskPackBlobSha','writeSetSha256',
      'reviewedHead','reviewedTree','reviewedBase'].some(k=>body[k]!==expected[k]))
    return rwUnknown('PRODUCT_REVIEW_WITNESS_MISMATCH');
  if(body.p0!==0||body.p1!==0||!Number.isSafeInteger(body.p2)||
    body.p2<0||body.p2>32)
    return rwUnknown('PRODUCT_REVIEW_FINDINGS_BLOCKED');
  return Object.freeze({status:'INDEPENDENTLY_VERIFIED',
    reason:'EXACT_GITHUB_REVIEW_WITNESS_VERIFIED',independentlyVerified:true,
    sourceQualified:true,reviewRef:ref,draftRef:body.draftRef,
    productDigest:body.productDigest,sourceSha:body.sourceSha,
    taskPackBlobSha:body.taskPackBlobSha,writeSetSha256:body.writeSetSha256,
    reviewedHead:body.reviewedHead,reviewedTree:body.reviewedTree,
    reviewedBase:body.reviewedBase,reviewerOperatorId:reviewerClaim.operatorId,
    reviewerWriterId:reviewerClaim.writerId,reviewerLogin:comment.user.login,
    p0:0,p1:0,p2:body.p2,authorized:false,taskPass:false,releasePass:false});
}
