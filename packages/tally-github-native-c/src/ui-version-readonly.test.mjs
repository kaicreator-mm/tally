import test from 'node:test';
import assert from 'node:assert/strict';
import {readUiNativeVersionFromClient} from './ui-version-readonly.mjs';

const owner='org',repo='tally',versionId='version/v2',sha='a'.repeat(40);
const url=n=>'https://github.com/'+owner+'/'+repo+'/issues/'+n;
const makeIssue=(n,state='open',workflow='state:planned')=>({
  number:n,id:1000+n,html_url:url(n),state,title:'Task '+n,
  updated_at:'2026-11-02T00:00:00Z',
  labels:[{name:'type:task'},{name:workflow},{name:'priority:P1'}]
});
function client({issues=[makeIssue(1),makeIssue(2,'closed','state:done'),
  makeIssue(3,'open','state:ready')],deps={2:[{
    number:1,html_url:url(1),state:'closed'}]},branchSha=sha}={}){
  const byNumber=new Map(issues.map(x=>[x.number,x]));
  const octokit={
    rest:{
      repos:{
        getBranch:async()=>({data:{commit:{sha:branchSha}}}),
        getCommit:async()=>({data:{sha}})
      },
      issues:{
        get:async({issue_number})=>({data:byNumber.get(issue_number)})
      }
    },
    paginate:{
      iterator:async function*(route,args){
        const n=args.issue_number;
        yield {data:deps[n]??[]};
      }
    }
  };
  const workItemReader=async({issueNumber})=>{
    const issue=byNumber.get(issueNumber);
    const state=issue.labels.find(x=>x.name.startsWith('state:')).name;
    const blockerCount=(deps[issueNumber]??[]).length;
    const openBlockerCount=(deps[issueNumber]??[]).filter(x=>x.state==='open').length;
    return {sourceSha:sha,workItemType:'type:task',claimStatus:'NOT_VERIFIED',
      authorized:false,taskPass:false,dependencyState:openBlockerCount?'BLOCKED':'CLEAR',
      blockerCount,openBlockerCount,workflowState:state,issueState:issue.state};
  };
  return {octokit,workItemReader};
}
const read=(c,taskIssueNumbers=[1,2,3])=>readUiNativeVersionFromClient({
  octokit:c.octokit,owner,repo,versionId,expectedSha:sha,taskIssueNumbers,
  workItemReader:c.workItemReader
});

test('complete frozen roster includes closed Issues and marks closure unverified',async()=>{
  const r=await read(client());
  assert.equal(r.connected,true);
  assert.equal(r.tasks.length,3);
  const closed=r.tasks.find(x=>x.id==='2');
  assert.equal(closed.issueState,'closed');
  assert.equal(closed.status,'unknown');
  assert.equal(closed.closureState,'CLOSED_UNVERIFIED');
  assert.equal(r.dag.edges.length,1);
  assert.equal(r.dag.edges[0].from,'1');
  assert.equal(r.dag.edges[0].to,'2');
  assert.equal(r.dag.edges[0].evidenceRef,url(2));
  assert.equal(r.dag.edges[0].dependencyRef,url(1));
});

test('unrelated repository Task Issues from another version do not contaminate the frozen roster',async()=>{
  const otherVersionTask={...makeIssue(4),labels:[
    {name:'type:task'},{name:'state:planned'},{name:'priority:P1'},
    {name:'version:v2.1.1'}
  ]};
  const c=client({issues:[makeIssue(1),makeIssue(2,'closed','state:done'),
    makeIssue(3,'open','state:ready'),otherVersionTask]});
  const r=await read(c);
  assert.equal(r.connected,true);
  assert.equal(r.tasks.length,3);
  assert.equal(r.tasks.some(task=>task.id==='4'),false);
});

test('a missing selected frozen Task Issue fails closed',async()=>{
  const c=client({issues:[makeIssue(1),makeIssue(2,'closed','state:done')]});
  const r=await read(c);
  assert.equal(r.connected,false);
  assert.equal(r.reason,'NATIVE_ISSUE_IDENTITY_UNVERIFIED');
});

test('dependency outside the frozen roster fails closed',async()=>{
  const c=client({deps:{2:[{number:9,html_url:url(9),state:'closed'}]}});
  assert.equal((await read(c)).reason,'NATIVE_DEPENDENCY_OUTSIDE_PINNED_ROSTER');
});

test('version branch movement and duplicate selected IDs deny source read',async()=>{
  assert.equal((await read(client({branchSha:'b'.repeat(40)}))).reason,
    'VERSION_BRANCH_NOT_PINNED');
  assert.equal((await read(client(),[1,1])).reason,
    'VERSION_TASK_ROSTER_NOT_CONFIGURED');
});
