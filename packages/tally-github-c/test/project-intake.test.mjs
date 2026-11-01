import test from 'node:test';
import assert from 'node:assert/strict';
import {projectIntakeReadModel,readProjectIntakeFromClient} from '../src/project-intake.mjs';

const sha='a'.repeat(40);
const repository={owner:'kaicreator-mm',repo:'tally',fullName:'kaicreator-mm/tally',
  defaultBranch:'version/v2',visibility:'private'};
const repoRest={full_name:'kaicreator-mm/tally',default_branch:'version/v2',
  visibility:'private'};
const rawSource={number:157,state:'closed',title:'docs(v2 Product): scope integration',
  url:'https://github.com/kaicreator-mm/tally/pull/157',isPullRequest:true,merged:true};
const rawProject={number:7,title:'Tally v2 Intake',closed:false,
  url:'https://github.com/kaicreator-mm/tally/projects/7',itemCount:2,
  items:[{issueNumber:210,title:'[v2][P0][J01–J03] Intake workbench',state:'open'},
    {issueNumber:157,title:'docs(v2 Product)',state:'closed'}]};
const observation=()=>({repository,observedAt:'2026-10-31T10:00:00.000Z',
  sources:[rawSource],projects:[rawProject]});

test('canonical observation projects to bounded read model without authority',()=>{
  const model=projectIntakeReadModel(observation());
  assert.equal(model.status,'PROJECT_INTAKE_OBSERVED');
  assert.equal(model.reason,'CANONICAL_GITHUB_FACTS_OBSERVED');
  assert.equal(model.sourceQualified,true);
  assert.equal(model.currentness,'OBSERVED_AT_QUERY_COMPLETION');
  assert.equal(model.sources[0].number,157);assert.equal(model.sources[0].merged,true);
  assert.equal(model.projects[0].items[0].issueNumber,210);
  assert.equal(model.authorized,false);assert.equal(model.admitted,false);
  assert.equal(model.taskPass,false);assert.equal(model.releasePass,false);
  assert.equal(Object.isFrozen(model),true);assert.equal(Object.isFrozen(model.sources),true);
});

test('malformed observations fail closed with bounded reasons',()=>{
  for(const [value,reason] of [
    [null,'INVALID_PROJECT_INTAKE_REQUEST'],
    [{...observation(),repository:{...repository,fullName:'other/tally'}},'PROJECT_INTAKE_REPOSITORY_UNVERIFIED'],
    [{...observation(),repository:{...repository,visibility:'internal'}},'PROJECT_INTAKE_REPOSITORY_UNVERIFIED'],
    [{...observation(),observedAt:'not-a-time'},'PROJECT_INTAKE_OBSERVATION_TIME_UNAVAILABLE'],
    [{...observation(),sources:[{...rawSource,number:0}]},'PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED'],
    [{...observation(),sources:[{...rawSource,url:'https://evil.test/157'}]},'PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED'],
    [{...observation(),projects:[{...rawProject,url:'https://github.com/kaicreator-mm/tally/x'}]},'PROJECT_INTAKE_PROJECT_FACTS_UNVERIFIED'],
    [{...observation(),projects:[{...rawProject,itemCount:99999}]},'PROJECT_INTAKE_PROJECT_FACTS_UNVERIFIED'],
    [{...observation(),sources:Array.from({length:17},(_,i)=>({...rawSource,number:i+1}))},'PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED'],
  ]){
    const model=projectIntakeReadModel(value);
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,reason);
    assert.equal(model.sourceQualified,false);assert.equal(model.taskPass,false);
  }
});

test('cross-repo readback and hostile clients refuse before any fact is trusted',async()=>{
  for(const client of [
    {rest:{repos:{get:async()=>({data:{...repoRest,full_name:'other/tally'}})}},graphql:async()=>({})},
    {rest:{repos:{get:async()=>({data:{...repoRest,visibility:'internal'}})}},graphql:async()=>({})},
  ]){
    const model=await readProjectIntakeFromClient({octokit:client,owner:'kaicreator-mm',
      repo:'tally',expectedSourceSha:sha,sources:[{number:157}]});
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,'PROJECT_INTAKE_REPOSITORY_UNVERIFIED');
  }
});

test('pinned sources and exact project-title readback become observed facts',async()=>{
  const calls=[];
  const issuesGet=async arg=>{
    calls.push('issue:'+arg.issue_number);
    return {data:{number:arg.issue_number,state:'closed',
      title:'docs(v2 Product): scope integration',
      html_url:`https://github.com/kaicreator-mm/tally/pull/${arg.issue_number}`,
      pull_request:{merged_at:'2026-09-01T00:00:00Z'}}};
  };
  const projectsData={repository:{nameWithOwner:'kaicreator-mm/tally',projectsV2:{nodes:[
    {number:7,title:'Tally v2 Intake',url:'https://github.com/kaicreator-mm/tally/projects/7',
      closed:false,
      items:{totalCount:2,nodes:[
        {content:{__typename:'Issue',number:210,state:'open',title:'Intake workbench'}},
        {content:{__typename:'DraftIssue',title:'draft (no issue)'}}]}}]}}};
  const octokit={
    rest:{
      repos:{get:async()=>{calls.push('repos');return {data:repoRest};}},
      issues:{get:issuesGet}},
    graphql:async(query,vars)=>{
      calls.push('graphql');
      assert.match(query,/projectsV2/);assert.equal(vars.owner,'kaicreator-mm');
      return projectsData}};
  const model=await readProjectIntakeFromClient({octokit,owner:'kaicreator-mm',repo:'tally',
    expectedSourceSha:sha,sources:[{number:157},{number:158}],projectTitle:'Tally v2 Intake'});
  assert.equal(model.status,'PROJECT_INTAKE_OBSERVED');
  assert.deepEqual(calls,['repos','issue:157','issue:158','graphql']);
  assert.equal(model.projects.length,1);assert.equal(model.projects[0].itemCount,2);
  assert.equal(model.projects[0].items.length,1);assert.equal(model.projects[0].items[0].issueNumber,210);
});

test('transport failures map to bounded reasons and never fabricate observation time',async()=>{
  for(const [client,reason] of [
    [{rest:{repos:{get:async()=>{throw {status:403};}}}},'GITHUB_ACCESS_DENIED'],
    [{rest:{repos:{get:async()=>{throw {status:404};}}}},'GITHUB_RESOURCE_NOT_FOUND'],
    [{rest:{repos:{get:async()=>{throw {status:429};}}}},'GITHUB_RATE_LIMITED'],
    [{rest:{repos:{get:async()=>({data:repoRest})},issues:{get:async()=>{throw new Error('x');}}}},'PROJECT_INTAKE_READ_FAILED'],
    [{rest:{repos:{get:async()=>({data:repoRest})}}},'PROJECT_INTAKE_READ_FAILED'],
  ]){
    const model=await readProjectIntakeFromClient({octokit:client,owner:'kaicreator-mm',
      repo:'tally',expectedSourceSha:sha,sources:[{number:157}],projectTitle:'Tally v2 Intake'});
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,reason);
    assert.equal(model.observedAt,null);assert.equal(model.taskPass,false);
  }
});

test('invalid requests refuse without any client call',async()=>{
  let called=false;
  const octokit={rest:{repos:{get:async()=>{called=true;return {data:repoRest};}}}};
  for(const args of [
    {octokit:undefined},{octokit,owner:'bad name'},{octokit,owner:'kaicreator-mm',repo:'tally',
      expectedSourceSha:'short'},{octokit,owner:'kaicreator-mm',repo:'tally',expectedSourceSha:sha,
      sources:[{number:0}]},{octokit,owner:'kaicreator-mm',repo:'tally',expectedSourceSha:sha,
      projectTitle:''},{octokit,owner:'kaicreator-mm',repo:'tally',expectedSourceSha:sha,limit:1000},
  ]){
    const model=await readProjectIntakeFromClient({owner:'kaicreator-mm',repo:'tally',
      expectedSourceSha:sha,...args});
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,'INVALID_PROJECT_INTAKE_REQUEST');
  }
  assert.equal(called,false);
});
