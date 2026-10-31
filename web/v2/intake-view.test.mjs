import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {renderIntakeWorkspace,__test} from './intake-view.mjs';

const sourceSha='d'.repeat(40);
const sources=[{id:'157',issueNumber:157,label:'v2 Product — reviewed P0/V scope',
  sourceRef:'https://github.com/kaicreator-mm/tally/issues/157',
  pinnedBy:'version/v2 merged source',
  fact:{status:'NOT_RUN',reason:'HOST_GITHUB_READ_REQUIRED',observedAsOf:null,
    currentness:'AGE_NOT_VERIFIED'},authorized:false,admitted:false}];
const dag={lanes:[{lane:'task-196',sourceRef:'https://github.com/kaicreator-mm/tally/issues/196',
  dependencyState:'CLEAR',claimState:'NOT_VERIFIED',agentAttemptState:'NOT_VERIFIED',
  reviewState:'NOT_VERIFIED',writeSetCount:0,status:'CANDIDATE'},
  {lane:'j01-intake-workbench',sourceRef:'https://github.com/kaicreator-mm/tally/issues/210',
  dependencyState:'CLEAR',claimState:'SELF_REPORTED_EVENT_ONLY',agentAttemptState:'NOT_RUN',
  reviewState:'NOT_VERIFIED',writeSetCount:16,status:'CANDIDATE'}],
  conflicts:[],blockedCount:0,cycleCount:0,candidateCount:2,advisoryOnly:true};
const base=()=>({status:'OBSERVED',project:'kaicreator-mm',version:'v2',sourceSha,
  observedAsOf:'2026-10-31T09:00:00.000Z',currentness:'OBSERVED_AT_QUERY_COMPLETION',
  guided:{journeyId:'tally-v2-intake-j01',
    caseRef:{snapshotId:'case',digest:'e'.repeat(64)},goals:[],evidence:[],
    decision:{status:'NEED_EVIDENCE',reasonCode:'GOAL_REQUIRED',step:null,evidenceRefs:[]}},
  sources,dag,draft:null,review:null,frozen:null,
  intake:{state:'NOT_RUN',reason:'HOST_PROJECT_INTAKE_REQUIRED',
    githubEffect:'NOT_RUN',project:null,sealedDispatch:'NOT_RUN'},
  constraints:{writeSet:Array.from({length:16},(_,i)=>`path/${i}.mjs`),
    writeSetDigest:'f'.repeat(64),pinnedClaim:'kaicreator-mm/tally#210:builder:__default__',
    pinnedDispatch:'dispatch:210:v2:20261031:1',sourceSha,
    l2Choices:['KEEP_MERGED_L2_ARCHITECTURE']},
  authorized:false,admitted:false,taskPass:false,releasePass:false});
const reviewed=(snapshot)=>{
  snapshot.draft={state:'CANDIDATE',author:'workbench:operator',
    at:'2026-10-31T09:05:00.000Z',digest:'a'.repeat(64),
    product:{goal:'Ship the workbench',nonGoals:['No raw JSON']},
    l2:{decision:'KEEP_MERGED_L2_ARCHITECTURE',choices:['KEEP_MERGED_L2_ARCHITECTURE']},
    provenance:['157'],note:'draft note'};
  snapshot.review={state:'REVIEWED',reviewer:'reviewer:ada',
    at:'2026-10-31T09:06:00.000Z',digest:snapshot.draft.digest,current:true};
  return snapshot;
};

test('read mode renders facts and explicit NOT_RUN with zero buttons and zero JSON fields',()=>{
  const html=renderIntakeWorkspace(base(),{mode:'read'});
  assert.doesNotMatch(html,/<button/i);assert.doesNotMatch(html,/onclick|onerror|onsubmit/i);
  assert.match(html,/NOT_RUN/);assert.match(html,/HOST_GITHUB_READ_REQUIRED/);
  assert.match(html,/HOST_PROJECT_INTAKE_REQUIRED/);
  assert.match(html,/sealed K dispatch: NOT_RUN/);
  assert.doesNotMatch(html,/textarea/i);
  assert.match(html,/a draft is never an accepted Freeze/i);
});

test('write mode exposes only typed forms with accessible labels, never raw JSON',()=>{
  const ready=base();
  const guided=ready.guided;
  const option1={id:'evidence-1',labelKey:'157',eligible:true};
  const option2={id:'evidence-2',labelKey:'158',eligible:true};
  const field={id:'evidence',type:'MULTI',labelKey:'guided.evidence',required:true,
    options:[option1,option2]};
  const step={kind:'INSPECT',stepId:'inspect-evidence',titleKey:'guided.inspect-evidence',
    fields:[field]};
  ready.guided={...guided,goals:['Ship the workbench'],evidence:['157'],
    decision:{status:'STEP',reasonCode:'INSPECT_EVIDENCE',step,evidenceRefs:['157']}};
  const html=renderIntakeWorkspace(ready,{mode:'write'});
  const labels=[...html.matchAll(/<label for="([^"]+)">/g)].map(m=>m[1]);
  const ids=[...html.matchAll(/<(?:input|textarea|select)[^>]*id="([^"]+)"/g)].map(m=>m[1]);
  for(const id of labels)assert.ok(ids.includes(id),`label target missing: ${id}`);
  assert.match(html,/type="checkbox"/);assert.match(html,/<select/);
  assert.match(html,/value="157"/);
  assert.match(html,/Reviewer identity \(must differ from the author\)/);
  assert.doesNotMatch(html,/paste.*JSON/i);
  assert.match(html,/no raw JSON/);
  assert.match(html,/data-action="submit-goal"|data-action="confirm-inspect"/);
  assert.match(html,/data-action="admit-project-intake"/);
});

test('draft CANDIDATE, independent REVIEWED and FROZEN states stay visually distinct',()=>{
  const html=renderIntakeWorkspace(reviewed(base()),{mode:'read'});
  assert.match(html,/CANDIDATE/);assert.match(html,/REVIEWED/);
  assert.match(html,/reviewer:ada/);assert.match(html,/digest current/);
  assert.match(html,/NOT_RUN/);
  const frozen=reviewed(base());
  frozen.frozen={state:'FROZEN',digest:'a'.repeat(64),
    receiptRef:`receipt:sha256:${'a'.repeat(64)}`,reviewer:'reviewer:ada',
    author:'workbench:operator',frozenAt:'2026-10-31T09:07:00.000Z',sourceSha,
    provenance:['157'],authority:'WORKBENCH_INDEPENDENT_REVIEW; native GitHub freeze authority stays Host-owned (NOT_RUN)'};
  const frozenHtml=renderIntakeWorkspace(frozen,{mode:'read'});
  assert.match(frozenHtml,/FROZEN/);
  assert.match(frozenHtml,/receipt:sha256:aaaa/);
  assert.match(frozenHtml,/native GitHub freeze authority stays Host-owned \(NOT_RUN\)/);
});

test('stale review and blocked state are shown instead of silently passing',()=>{
  const stale=reviewed(base());
  stale.review.current=false;
  const html=renderIntakeWorkspace(stale,{mode:'read'});
  assert.match(html,/STALE/);
});

test('DAG panel presents physical claim facts and conflicts, not a static diagram',()=>{
  const daggy=base();
  daggy.dag={...dag,lanes:[...dag.lanes,{lane:'task-177',
    sourceRef:'https://github.com/kaicreator-mm/tally/issues/177',
    dependencyState:'BLOCKED',claimState:'NOT_VERIFIED',agentAttemptState:'NOT_VERIFIED',
    reviewState:'NOT_VERIFIED',writeSetCount:0,status:'CANDIDATE'}],
    blockedCount:1,cycleCount:0,conflicts:[{type:'WRITE_SET_CONFLICT',
    lanes:['a','b'],shared:['shared/file.mjs']}]};
  const html=renderIntakeWorkspace(daggy,{mode:'read'});
  assert.match(html,/BLOCKED/);assert.match(html,/cycles: 0/);
  assert.match(html,/blocked native tasks: 1/i);
  assert.match(html,/shared\/file\.mjs/);
  assert.match(html,/16 pinned path\(s\)/);
});

test('hostile snapshot content is escaped everywhere',()=>{
  const hostile=base();
  hostile.guided.goals=['<img src=x onerror=alert(1)>'];
  hostile.sources=[{...sources[0],label:'<script>alert(1)</script>',
    fact:{status:'NOT_RUN',reason:'<b>bold</b>',observedAsOf:null}}];
  const html=renderIntakeWorkspace(hostile,{mode:'write'});
  assert.doesNotMatch(html,/<img src=x onerror/);assert.doesNotMatch(html,/<script>alert/);
  assert.match(html,/&lt;script&gt;/);assert.match(html,/&lt;b&gt;bold&lt;\/b&gt;/);
});

test('unverified snapshot stays UNKNOWN with no forms',()=>{
  for(const value of [null,undefined,{status:'WRONG'}]){
    const html=renderIntakeWorkspace(value,{mode:'write'});
    assert.match(html,/UNKNOWN/);assert.doesNotMatch(html,/<button/i);
    assert.match(html,/No form or action is available/);
  }
});

test('intake-view keeps escaping helpers consistent with journey view',()=>{
  assert.equal(__test.escape('<>&"\''),'&lt;&gt;&amp;&quot;&#39;');
  assert.equal(__test.sha(sourceSha),'dddddddddddd');
  assert.equal(__test.sha('bad'),'UNKNOWN');
  assert.equal(__test.url('javascript:alert(1)'),'#');
  assert.equal(__test.url('https://github.com/kaicreator-mm/tally/pull/157'),
    'https://github.com/kaicreator-mm/tally/pull/157');
});

test('static journey page hosts the intake section with deferred modules only',()=>{
  const html=readFileSync(new URL('./journey-view.html',import.meta.url),'utf8');
  assert.match(html,/id="intake-app"/);
  assert.match(html,/<script defer src="\.\/intake-bootstrap\.mjs"><\/script>/);
  assert.doesNotMatch(html,/<script(?:\s+type="module")?\s*>/);
  assert.doesNotMatch(html,/onerror|onclick|onsubmit/i);
});
