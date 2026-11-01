import test from 'node:test';
import assert from 'node:assert/strict';
import {buildTree,storyCompletion,nextStepHint,freezeDiff} from './model.ts';
import {qaGaps,qaSummary} from '../qa/model.ts';
import {releaseBlockers,actionDisplay} from '../release/model.ts';
import {dedupeDefects,suggestRouting,learningDisplay,nextIntakeState}
  from '../maintenance/model.ts';
import type {RequirementNode} from './model.ts';
import type {QaRecord} from '../qa/model.ts';
import type {ReleaseReadiness} from '../release/model.ts';
import type {DefectRecord,LearningRecord} from '../maintenance/model.ts';

const req=(over:Partial<RequirementNode>):RequirementNode=>({
  id:'r1',kind:'requirement',title:'需求',parent:null,productRevision:null,
  acceptanceRefs:[],l2Decision:null,freezeState:'NOT_FROZEN',
  updatedAt:'2026-11-01T00:00:00Z',...over});
test('requirement hierarchy builds roots/children in stable order',()=>{
  const nodes=[req({id:'r2',parent:'r1',kind:'story'}),
    req({id:'r10',parent:'r1',kind:'story'}),
    req({id:'r1'}),req({id:'r0'})];
  const tree=buildTree(nodes);
  assert.deepEqual(tree.roots.map(node=>node.id),['r0','r1']);
  assert.deepEqual(tree.childrenOf('r1').map(node=>node.id),['r2','r10']);
  assert.deepEqual(tree.childrenOf('r0'),[]);
});
test('story completion and next-step hints stay honest',()=>{
  assert.deepEqual(storyCompletion(req({id:'s'})).reasons,
    ['not_frozen','acceptance_missing']);
  assert.equal(nextStepHint(req({id:'s'})),'add_acceptance');
  assert.equal(nextStepHint(req({id:'s',acceptanceRefs:['a']})),'request_freeze');
  assert.equal(nextStepHint(req({id:'s',acceptanceRefs:['a'],
    freezeState:'STALE'})),'renew_freeze');
  const ready=req({id:'s',acceptanceRefs:['a'],freezeState:'FROZEN',
    productRevision:3});
  assert.deepEqual(storyCompletion(ready).reasons,[]);
  assert.equal(nextStepHint(ready),'none');
  assert.deepEqual(storyCompletion(req({id:'u',freezeState:'UNKNOWN'})).reasons,
    ['freeze_unknown','acceptance_missing']);
});
test('freeze diff flags only nodes revised above the frozen revision',()=>{
  const nodes=[req({id:'a',productRevision:2}),req({id:'b',productRevision:5}),
    req({id:'c',productRevision:null})];
  const diff=freezeDiff(nodes,3);
  assert.deepEqual(diff.changed.map(node=>node.id),['b']);
  assert.equal(diff.unchangedCount,2);
});
const qa=(over:Partial<QaRecord>):QaRecord=>({layer:'fullRegression',
  provenance:'INDEPENDENT',outcome:'PASS',witness:'ci',evidenceRef:'https://x',
  subjectSha:'a'.repeat(40),observedAt:'2026-11-01T00:00:00Z',...over});
test('qa layers: self-report never satisfies independent-witness layers',()=>{
  const reportedHidden=qa({layer:'hiddenValidation',provenance:'REPORTED'});
  const gaps=qaGaps([qa({}),reportedHidden,
    qa({layer:'criticalJourneys',outcome:'FAIL'}),
    qa({layer:'installedAcceptance',outcome:'NOT_RUN',provenance:'NOT_RUN'})]);
  const codes=gaps.map(gap=>gap.code);
  assert.ok(codes.includes('hiddenValidation_NEEDS_INDEPENDENT'));
  assert.ok(codes.includes('criticalJourneys_FAIL'));
  assert.ok(codes.includes('installedAcceptance_NOT_RUN'));
  assert.equal(qaSummary([qa({}),
    qa({layer:'hiddenValidation',provenance:'REPORTED'}),
    qa({layer:'criticalJourneys'}),
    qa({layer:'installedAcceptance'})]).ready,false);
  const allGood=[qa({}),qa({layer:'hiddenValidation',witness:'verifier-1'}),
    qa({layer:'criticalJourneys'}),qa({layer:'installedAcceptance',witness:'user-1'})];
  assert.deepEqual(qaGaps(allGood),[]);
  assert.equal(qaSummary(allGood).ready,true);
});
const readiness=(over:Partial<ReleaseReadiness>):ReleaseReadiness=>({
  versionId:'v1',gates:[{name:'woodpeckerCI',status:'PASS',owner:'CI'},
    {name:'hiddenValidation',status:'NOT_RUN',owner:'独立验证者'}],
  permits:null,rqState:'NOT_QUALIFIED',ownerGrantState:'NOT_GRANTED',...over});
test('release blockers bounded to 3 and actions need fresh permits',()=>{
  const blocked=readiness({});
  const blockers=releaseBlockers(blocked);
  assert.deepEqual(blockers.map(blocker=>blocker.code),
    ['RQ_MISSING','GRANT_MISSING','GATES_INCOMPLETE']);
  assert.equal(actionDisplay(blocked,'merge').enabled,false);
  assert.match(actionDisplay(blocked,'merge').reason!,/RQ_MISSING/);
  // GRANTED permit alone does not bypass missing RQ display logic (permit
  // wins only when Host issued it fresh for this exact subject).
  const permitted=readiness({permits:[
    {operation:'merge',state:'GRANTED',sourceSha:'a'.repeat(40),
      grantedAt:'2026-11-01T00:00:00Z'}]});
  assert.equal(actionDisplay(permitted,'merge').enabled,true);
  assert.equal(actionDisplay(permitted,'tag').enabled,false);
  assert.match(actionDisplay(permitted,'tag').reason!,/RQ_MISSING/);
  const denied=readiness({rqState:'QUALIFIED',ownerGrantState:'GRANTED',
    permits:[{operation:'merge',state:'DENIED',sourceSha:null,grantedAt:null}]});
  assert.equal(actionDisplay(denied,'merge').enabled,false);
  assert.match(actionDisplay(denied,'merge').reason!,/HOST_PERMIT_DENIED/);
});
const defect=(over:Partial<DefectRecord>):DefectRecord=>({
  id:'d1',summary:'登录失败',releaseVersion:'v2.0',releaseCommitSha:'a'.repeat(40),
  requirementRef:null,state:'OPEN',routedTo:null,duplicateOf:null,
  casebookRefs:[],observedAt:'2026-11-01T00:00:00Z',...over});
test('defect dedupe, routing suggestion and intake transitions',()=>{
  const {unique,duplicates}=dedupeDefects([
    defect({id:'d1'}),defect({id:'d2'}), // same summary → d2 duplicate
    defect({id:'d3',summary:'导出错误',duplicateOf:'d1'}),
    defect({id:'d4',summary:'另一个问题'})]);
  assert.deepEqual(unique.map(item=>item.id),['d1','d4']);
  assert.deepEqual(duplicates.map(item=>item.id),['d2','d3']);
  assert.equal(suggestRouting(defect({})).routedTo,'repair');
  assert.equal(suggestRouting(defect({releaseCommitSha:null})).routedTo,'next_version');
  assert.equal(suggestRouting(defect({releaseVersion:null})).routedTo,'next_version');
  assert.equal(nextIntakeState('draft','submit'),'submitted');
  assert.equal(nextIntakeState('submitted','admit'),'admitted');
  assert.equal(nextIntakeState('submitted','needs_info'),'needs_info');
  assert.equal(nextIntakeState('needs_info','edit'),'draft');
  assert.equal(nextIntakeState('admitted','edit'),null);
  assert.equal(nextIntakeState('draft','admit'),null);
});
test('learning display never infers quality gain',()=>{
  assert.deepEqual(learningDisplay({id:'l1',decision:'ADOPTED',
    outcomeAttested:null,observedAt:null,note:null}),
    {decision:'ADOPTED',effect:'NOT_MEASURED'});
  assert.equal(learningDisplay({id:'l2',decision:'ADOPTED',
    outcomeAttested:true,observedAt:'2026-11-01T00:00:00Z',
    note:'n'}).effect,'ATTESTED');
  const rejected:LearningRecord={id:'l3',decision:'REJECTED',
    outcomeAttested:false,observedAt:null,note:null};
  assert.equal(learningDisplay(rejected).effect,'NOT_MEASURED');
});
