import test from 'node:test';
import assert from 'node:assert/strict';
import {toJson,toMermaid} from './model.ts';
import type {VersionDag,DagNode,DagEdge} from './model.ts';

/** P2-04 (#260): freeze/Claim provenance and NATIVE_PARTIAL coverage must
 * survive UI projection and JSON/Mermaid export exactly as the verified
 * Host snapshot produced them — never inferred, never silently dropped. */
const node=(over:Partial<DagNode>={}):DagNode=>({taskId:'260',versionId:'v2',
  title:'Closeout A',status:'open',severity:'P0',requirementGroup:null,
  phase:null,lane:null,agentId:null,executable:false,...over});
const edge=(over:Partial<DagEdge>={}):DagEdge=>({from:'260',to:'264',
  kind:'PLANNED',evidenceRef:null,amendedAt:null,...over});
const frozenBlobSha='b'.repeat(40);
const taskPackRef='https://github.com/kaicreator-mm/tally/blob/'+
  'c'.repeat(40)+'/apps/tally-web/src/features/dag/version-v2-task-pack.candidate.json';
const nativePartialDag:VersionDag={versionId:'v2',
  asOf:{sourceSha:'a'.repeat(40),frozenAt:'2026-11-02T00:00:00Z'},
  rosterCoverage:'VERIFIED_COMPLETE',claimReadbackCoverage:'VERIFIED_COMPLETE',
  claimCoverage:'PARTIAL',acceptedClaimCount:1,notVerifiedCount:2,
  notClaimedCount:3,activeWebResponsibilityCount:0,
  dependencyCoverage:'NATIVE_PARTIAL',frozenBlobSha,taskPackRef,
  nodes:[node({taskId:'260',claimState:'ACCEPTED'}),
    node({taskId:'264',claimState:'NOT_VERIFIED'})],
  edges:[edge(),edge({from:'261',to:'264',kind:'NATIVE',
    evidenceRef:'https://github.com/kaicreator-mm/tally/issues/264'})],
  connected:true};

test('JSON export preserves NATIVE_PARTIAL and exact freeze provenance',()=>{
  const parsed=JSON.parse(toJson(nativePartialDag));
  assert.equal(parsed.dependencyCoverage,'NATIVE_PARTIAL');
  assert.equal(parsed.frozenBlobSha,frozenBlobSha);
  assert.equal(parsed.taskPackRef,taskPackRef);
});

test('JSON export preserves full Claim readback coverage and counts',()=>{
  const parsed=JSON.parse(toJson(nativePartialDag));
  assert.equal(parsed.rosterCoverage,'VERIFIED_COMPLETE');
  assert.equal(parsed.claimReadbackCoverage,'VERIFIED_COMPLETE');
  assert.equal(parsed.claimCoverage,'PARTIAL');
  assert.equal(parsed.acceptedClaimCount,1);
  assert.equal(parsed.notVerifiedCount,2);
  assert.equal(parsed.notClaimedCount,3);
  assert.equal(parsed.activeWebResponsibilityCount,0);
  assert.equal(parsed.asOf.frozenAt,'2026-11-02T00:00:00Z');
});

test('JSON export keeps NATIVE_COMPLETE distinguishable from NATIVE_PARTIAL',()=>{
  const parsed=JSON.parse(toJson({...nativePartialDag,
    dependencyCoverage:'NATIVE_COMPLETE'}));
  assert.equal(parsed.dependencyCoverage,'NATIVE_COMPLETE');
  assert.notEqual(parsed.dependencyCoverage,'NATIVE_PARTIAL');
});

test('absent Host metadata exports explicit UNKNOWN/null, never invented',()=>{
  const bare:VersionDag={versionId:'v2',
    asOf:{sourceSha:'a'.repeat(40),frozenAt:null},
    nodes:[node()],edges:[],connected:true};
  const parsed=JSON.parse(toJson(bare));
  assert.equal(parsed.dependencyCoverage,'UNKNOWN');
  assert.equal(parsed.rosterCoverage,'UNVERIFIED');
  assert.equal(parsed.claimReadbackCoverage,'NOT_VERIFIED');
  assert.equal(parsed.claimCoverage,'NOT_VERIFIED');
  assert.equal(parsed.acceptedClaimCount,null);
  assert.equal(parsed.notVerifiedCount,null);
  assert.equal(parsed.notClaimedCount,null);
  assert.equal(parsed.activeWebResponsibilityCount,null);
  assert.equal(parsed.frozenBlobSha,null);
  assert.equal(parsed.taskPackRef,null);
});

test('Mermaid export carries safe coverage/freeze provenance comments',()=>{
  const text=toMermaid(nativePartialDag);
  assert.match(text,/%% dependencyCoverage=NATIVE_PARTIAL/);
  assert.match(text,/frozenBlob=bbbbbbbbbbbb/);
  assert.match(text,/taskPack=PINNED/);
  assert.ok(!text.includes(taskPackRef)); // raw URL is never Mermaid source
  const bare=toMermaid({versionId:'v2',
    asOf:{sourceSha:'a'.repeat(40),frozenAt:null},
    nodes:[node()],edges:[],connected:true});
  assert.match(bare,/%% dependencyCoverage=UNKNOWN/);
  assert.match(bare,/frozenBlob=NOT_FROZEN/);
  assert.match(bare,/taskPack=ABSENT/);
});
