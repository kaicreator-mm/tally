/** Read-only Source→Derived version lineage. No GitHub reader or Release authority. */
const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/(?:issues|pull|commit|releases)\/[A-Za-z0-9_.~/#-]+$/;
const EDGE_STATES=new Set(['OBSERVED','DECLARED','INFERRED','UNKNOWN','STALE']);
const iso=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
const validSha=value=>SHA.test(value??'');
const validRef=value=>REF.test(value??'');
const array=value=>Array.isArray(value)?value:[];
const rows=value=>array(value).filter(row=>row&&typeof row==='object'&&!Array.isArray(row));
const malformedRows=value=>value!==undefined&&(!Array.isArray(value)||value.some(row=>
  !row||typeof row!=='object'||Array.isArray(row)));
const validKindRef=(value,repo,kinds)=>validRef(value)&&kinds.some(kind=>
  value.startsWith(`https://github.com/${repo}/${kind}/`));
const clone=value=>value===undefined?undefined:structuredClone(value);
const refFor=(repo,number)=>`https://github.com/${repo}/issues/${number}`;
const edge=(from,to,kind,status,sourceRef,sourceSha,observedAsOf,revision)=>({
  type:'EvidenceEdge',from,to,kind,status:EDGE_STATES.has(status)?status:'UNKNOWN',sourceRef:sourceRef??null,
  sourceSha:sourceSha??null,observedAsOf,revision:revision??null});
function validSubject(repo,subject){
  return REPO.test(repo??'')&&subject&&typeof subject.version==='string'&&subject.version.length>0&&
    validSha(subject.sourceSha)&&iso(subject.observedAsOf)&&validRef(subject.sourceRef)&&
    validRef(subject.productFreezeRef)&&validRef(subject.l2FreezeRef)&&validRef(subject.adsSourceRef);
}
function coverageOf(evidence){
  const coverage=evidence?.coverage??{};
  return {pagination: coverage.paginationComplete===true?'COMPLETE':'UNKNOWN',
    schema:coverage.schema==='ads/github/v3'&&coverage.dialect==='ai-dev/event-v2'?'SUPPORTED':'UNKNOWN',
    currentness:coverage.currentnessComplete===true?'COMPLETE':'UNKNOWN'};
}
function correctedRefs(evidence){
  const result=new Map();
  for(const row of array(evidence?.corrections)){
    if(typeof row?.sourceRef==='string')
      result.set(row.sourceRef,['EDIT','REVOCATION'].includes(row.action)?'STALE':'UNKNOWN');
  }
  return result;
}
function publicAcceptance(repo,issue,acceptances,pr,usable){
  if(!pr||!usable)return {status:'UNKNOWN',evidence:null};
  const exact=acceptances.filter(row=>row.issueNumber===issue.number&&row.sourceSha===pr.headSha);
  const proven=exact.find(row=>row.status==='PROVEN'&&validKindRef(row.ref,repo,['issues','pull']));
  if(usable&&proven)return {status:'PROVEN',evidence:proven};
  if(issue.acceptanceRequired===true)return {status:'PUBLIC_ACCEPTANCE_NOT_PROVEN',evidence:null};
  return {status:'UNKNOWN',evidence:null};
}
/** evidence contains native issues, pullRequests, reviews, checks, attempts,
 * acceptances, regressions and corrections observed by an existing selected C. */
export function reconstructVersionTimeline(repo,qualifiedVersionSubject,evidence={}){
  evidence??={};
  const subject=qualifiedVersionSubject??{};
  if(!validSubject(repo,subject))return Object.freeze({
    subjectStatus:'UNKNOWN',reason:'QUALIFIED_VERSION_SUBJECT_INCOMPLETE',items:[],edges:[],
    coverage:{type:'ProvenanceCoverage',pagination:'UNKNOWN',schema:'UNKNOWN',currentness:'UNKNOWN',
      uniqueIssueCount:0,sourceMergedCount:0,realCompletedCount:0,publicAcceptanceNotProvenCount:0,
      unknownCount:0,modelCoverage:'UNKNOWN',activeTime:'UNKNOWN',
      comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false}
  });
  const observedAsOf=subject.observedAsOf,coverage=coverageOf(evidence),corrections=correctedRefs(evidence);
  const evidenceKeys=['issues','pullRequests','reviews','checks','attempts','acceptances','regressions','corrections'];
  const malformedEvidence=evidenceKeys.some(key=>malformedRows(evidence[key]))||
    array(evidence.corrections).some(row=>typeof row?.sourceRef!=='string');
  const usable=!malformedEvidence&&Object.values(coverage).every(value=>value==='COMPLETE'||value==='SUPPORTED');
  const issues=rows(evidence.issues).filter(row=>Number.isSafeInteger(row.number)&&row.number>0);
  const pullRequests=rows(evidence.pullRequests),reviews=rows(evidence.reviews),checks=rows(evidence.checks);
  const attempts=rows(evidence.attempts),acceptances=rows(evidence.acceptances),regressions=rows(evidence.regressions);
  const seen=new Set(),items=[],edges=[];
  const issueCounts=new Map();
  for(const row of issues)issueCounts.set(row.number,(issueCounts.get(row.number)??0)+1);
  for(const issue of issues){
    const number=issue.number,issueRef=refFor(repo,number),identity=`${repo}#${number}`;
    if(seen.has(number))continue;
    const duplicate=issueCounts.get(number)>1;seen.add(number);
    const issueUsable=usable&&!duplicate&&issue.ref===issueRef&&['open','closed'].includes(issue.state);
    const prs=pullRequests.filter(row=>row.issueNumber===number);
    const pr=prs.find(row=>Number.isSafeInteger(row.number)&&row.number>0&&
      row.merged===true&&row.containedInSubject===true&&validSha(row.headSha)&&validSha(row.mergedSha)&&
      validRef(row.ref)&&row.ref===`https://github.com/${repo}/pull/${row.number}`);
    const sourceMerged=issueUsable&&!!pr;
    const review=pr&&reviews.find(row=>row.prNumber===pr.number&&row.sourceSha===pr.headSha&&
      row.verdict==='PASS'&&validKindRef(row.ref,repo,['pull'])&&
      (row.ref===pr.ref||row.ref.startsWith(pr.ref+'#')||row.ref.startsWith(pr.ref+'/')));
    const required=array(issue.requiredChecks);
    const checkRows=pr?checks.filter(row=>row.prNumber===pr.number&&row.sourceSha===pr.headSha):[];
    const checksPass=sourceMerged&&required.every(kind=>checkRows.some(row=>
      row.kind===kind&&row.status==='PASS'&&validKindRef(row.ref,repo,['pull'])&&
      (row.ref===pr.ref||row.ref.startsWith(pr.ref+'#')||row.ref.startsWith(pr.ref+'/'))));
    const attempt=attempts.find(row=>row.issueNumber===number&&pr&&row.sourceSha===pr.headSha&&
      row.physical===true&&['LOCAL_CANDIDATE_READY','VALIDATED'].includes(row.state)&&
      typeof row.id==='string'&&row.id.length>0&&validKindRef(row.ref,repo,['issues','pull'])&&
      typeof row.claimRef==='string'&&row.claimRef.startsWith(issueRef+'#issuecomment-'));
    const acceptance=publicAcceptance(repo,issue,acceptances,pr,issueUsable);
    const affected=[issueRef,pr?.ref,review?.ref,acceptance.evidence?.ref,attempt?.ref,attempt?.claimRef,
      ...checkRows.map(row=>row.ref)].filter(Boolean).some(ref=>corrections.has(ref));
    const currentQualified=issueUsable&&!affected;
    let completion='INCOMPLETE';
    if(!currentQualified)completion='UNKNOWN';
    else if(sourceMerged&&review&&checksPass&&(issue.physicalRequired!==true||attempt)&&
      (issue.acceptanceRequired!==true||acceptance.status==='PROVEN'))completion='REAL_COMPLETED';
    else if(sourceMerged)completion='SOURCE_MERGED';
    let disposition=['actionable','superseded','historical'].includes(issue.disposition)?
      issue.disposition:'UNKNOWN';
    if(disposition==='superseded'&&issueUsable&&!pr)completion='UNKNOWN';
    items.push({type:'WorkLineage',issue:identity,issueRef,issueState:issue.state??'UNKNOWN',version:subject.version,
      disposition,completion,sourceMerged,reviewStatus:currentQualified&&review?'REVIEWED_EXACT_SHA':'UNKNOWN',
      requiredChecks:clone(required),checksStatus:!currentQualified?'UNKNOWN':required.length===0?'NOT_REQUIRED':
        checksPass?'PASS':'UNKNOWN',physicalAttemptId:currentQualified?(attempt?.id??null):null,
      publicAcceptance:currentQualified?acceptance.status:'UNKNOWN',historicalFacts:clone(array(issue.historicalFacts)),
      sourceRefs:{issue:issueRef,pullRequest:pr?.ref??null,review:review?.ref??null,
        acceptance:acceptance.evidence?.ref??null,attempt:attempt?.ref??null,claim:attempt?.claimRef??null}});
    edges.push(edge(`${repo}/version/${subject.version}`,identity,'VERSION_ISSUE',
      corrections.get(issueRef)??(issueUsable?'OBSERVED':'UNKNOWN'),issueRef,subject.sourceSha,observedAsOf,issue.revision));
    if(pr)edges.push(edge(identity,`${repo}/pull/${pr.number}`,'ISSUE_PULL_REQUEST',
      corrections.has(pr.ref)?corrections.get(pr.ref):'OBSERVED',pr.ref,pr.headSha,observedAsOf,pr.revision));
    if(review)edges.push(edge(`${repo}/pull/${pr.number}`,`${repo}/pull/${pr.number}/review`,'EXACT_REVIEW',
      corrections.has(review.ref)?corrections.get(review.ref):'OBSERVED',review.ref,pr.headSha,observedAsOf,review.revision));
    for(const check of checkRows)edges.push(edge(`${repo}/pull/${pr.number}`,`${repo}/pull/${pr.number}/check/${check.kind}`,
      'REQUIRED_CHECK',corrections.get(check.ref)??'OBSERVED',check.ref,pr.headSha,observedAsOf,check.revision));
    if(attempt){
      edges.push(edge(identity,`${identity}/attempt/${attempt.id}`,'CLAIMED_PHYSICAL_ATTEMPT',
        corrections.get(attempt.ref)??'OBSERVED',attempt.ref,pr.headSha,observedAsOf,attempt.revision));
      edges.push(edge(attempt.claimRef,`${identity}/attempt/${attempt.id}`,'ACCEPTED_CLAIM_ATTEMPT',
        corrections.get(attempt.claimRef)??'OBSERVED',attempt.claimRef,pr.headSha,observedAsOf,attempt.revision));
    }
    if(acceptance.evidence)edges.push(edge(identity,`${identity}/public-acceptance`,'PUBLIC_ACCEPTANCE',
      corrections.get(acceptance.evidence.ref)??'OBSERVED',acceptance.evidence.ref,pr.headSha,observedAsOf,
      acceptance.evidence.revision));
  }
  for(const row of regressions){
    const validCurrent=Number.isSafeInteger(row.issueNumber)&&row.issueNumber>0;
    const validPrevious=Number.isSafeInteger(row.previousIssueNumber)&&row.previousIssueNumber>0;
    const from=validPrevious?`${repo}#${row.previousIssueNumber}`:'UNKNOWN';
    const to=validCurrent?`${repo}#${row.issueNumber}`:'UNKNOWN';
    const currentRef=validCurrent?refFor(repo,row.issueNumber):null;
    const previousRef=validPrevious?refFor(repo,row.previousIssueNumber):null;
    const observed=validCurrent&&validPrevious&&row.status==='OBSERVED'&&
      (row.evidenceRef===currentRef||row.evidenceRef===previousRef)&&!corrections.has(row.evidenceRef);
    edges.push(edge(from,to,'SUBSEQUENT_REGRESSION',observed?'OBSERVED':'UNKNOWN',
      row.evidenceRef??null,validSha(row.sourceSha)?row.sourceSha:null,observedAsOf,row.revision));
  }
  edges.push(edge(subject.productFreezeRef,`${repo}/version/${subject.version}`,'PRODUCT_FREEZE',
    'DECLARED',subject.productFreezeRef,subject.sourceSha,observedAsOf));
  edges.push(edge(subject.l2FreezeRef,`${repo}/version/${subject.version}`,'L2_FREEZE',
    'DECLARED',subject.l2FreezeRef,subject.sourceSha,observedAsOf));
  const count=status=>items.filter(item=>item.completion===status).length;
  const result={subjectStatus:usable?'OBSERVED':'UNKNOWN',subject:{type:'VersionSubject',repo,version:subject.version,
    sourceSha:subject.sourceSha,sourceRef:subject.sourceRef,adsSourceRef:subject.adsSourceRef,observedAsOf},items,edges,
    coverage:{type:'ProvenanceCoverage',...coverage,uniqueIssueCount:seen.size,sourceMergedCount:count('SOURCE_MERGED')+
      count('REAL_COMPLETED'),realCompletedCount:count('REAL_COMPLETED'),
      publicAcceptanceNotProvenCount:items.filter(item=>item.publicAcceptance==='PUBLIC_ACCEPTANCE_NOT_PROVEN').length,
      unknownCount:count('UNKNOWN'),modelCoverage:'UNKNOWN',activeTime:'UNKNOWN',
      comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false}};
  return Object.freeze(result);
}
export function formatLineageTable(timeline){
  const lines=['| Issue | Disposition | Completion | Public acceptance | Source |','| --- | --- | --- | --- | --- |'];
  for(const item of array(timeline?.items))lines.push(`| ${item.issue} | ${item.disposition} | ${item.completion} | ${item.publicAcceptance} | ${item.sourceRefs.pullRequest??item.issueRef} |`);
  return lines.join('\n');
}
