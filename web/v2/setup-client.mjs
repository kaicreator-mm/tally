/** Local-only first-run UI: no token is retained in browser storage. */
const $=id=>document.getElementById(id);
const key=document.querySelector('meta[name="tally-setup-key"]')?.content??'';
async function load(){
  const response=await fetch('/api/setup/status',{cache:'no-store'});
  if(!response.ok)throw new Error('SETUP_STATUS_UNAVAILABLE');
  return response.json();
}
function line(text,parent){
  const p=document.createElement('p');p.textContent=text;parent.append(p);
}
function render(data){
  const target=$('projects');target.replaceChildren();
  const projects=data.projects??[];
  if(!projects.length)line('还没有已保存项目。',target);
  for(const project of projects){
    const section=document.createElement('section');
    const title=document.createElement('h3');
    title.textContent=project.repository;section.append(title);
    for(const version of project.versions){
      const button=document.createElement('button');
      const selected=data.selected?.repository===project.repository&&
        data.selected.ref===version.ref;
      button.type='button';
      button.textContent=(selected?'✓ ':'')+version.ref+' · '+version.sha.slice(0,10);
      button.addEventListener('click',()=>selectProject(project.repository,version.ref));
      section.append(button,document.createTextNode(' '));
    }
    target.append(section);
  }
  $('draft-panel').hidden=!data.selected;
  $('draft-panel').dataset.repository=data.selected?.repository??'';
  $('draft-panel').dataset.ref=data.selected?.ref??'';
  const active=$('status');
  active.textContent=data.activeUrl?'服务已连接: '+(data.activeSource?.repository??''):
    data.lastError?'启动失败: '+data.lastError:'请选择或添加项目。';
  if(data.activeUrl){
    const a=document.createElement('a');
    a.href=data.activeUrl;a.textContent='打开 Tally 工作台';
    active.append(document.createTextNode(' '),a);
  }
}
async function submit(path,payload){
  const result=await fetch(path,{method:'POST',
    headers:{'content-type':'application/json','x-tally-setup-key':key},
    body:JSON.stringify(payload)});
  const data=await result.json();
  if(!result.ok)throw new Error(data.reason??'SETUP_FAILED');
  render({...data,activeSource:{repository:payload.repository}});
  if(data.activeUrl)window.location.assign(data.activeUrl);
}
async function selectProject(repository,ref){
  $('status').textContent='正在验证所选分支的最新 HEAD…';
  try{await submit('/api/setup/select',{repository,ref});}
  catch(error){$('status').textContent='连接失败: '+error.message;}
}
$('add').addEventListener('submit',async event=>{
  event.preventDefault();
  const form=event.currentTarget;
  const repository=form.elements.namedItem('repository').value.trim();
  const ref=form.elements.namedItem('ref').value.trim();
  const token=form.elements.namedItem('token').value.trim();
  $('status').textContent='正在验证 GitHub 连接…';
  try{
    await submit('/api/setup/add',{repository,ref,token});
    form.elements.namedItem('token').value='';
  }catch(error){
    form.elements.namedItem('token').value='';
    $('status').textContent='连接失败: '+error.message;
  }
});
load().then(render).catch(error=>{$('status').textContent=error.message;});

$('draft').addEventListener('submit',async event=>{
  event.preventDefault();
  const form=event.currentTarget;
  const repository=$('draft-panel').dataset.repository;
  const ref=$('draft-panel').dataset.ref;
  const writeToken=form.elements.namedItem('writeToken').value.trim();
  const title=form.elements.namedItem('title').value.trim();
  const description=form.elements.namedItem('description').value.trim();
  const output=$('draft-result');
  output.textContent='正在创建 GitHub Issue 草稿…';
  form.querySelector('button[type="submit"]').disabled=true;
  try{
    const response=await fetch('/api/setup/task-draft',{method:'POST',
      headers:{'content-type':'application/json','x-tally-setup-key':key},
      body:JSON.stringify({repository,ref,title,description,writeToken,
        requestId:crypto.randomUUID()})});
    const result=await response.json();
    if(!response.ok)throw new Error(result.reason??'DRAFT_CREATION_UNKNOWN');
    output.replaceChildren();
    const link=document.createElement('a');
    link.href=result.issueUrl;
    link.rel='noopener noreferrer';link.target='_blank';
    link.textContent='打开新建 GitHub Issue #'+result.issueNumber;
    output.append('已创建未授权任务草稿：',link);
    form.elements.namedItem('title').value='';
    form.elements.namedItem('description').value='';
  }catch(error){
    output.textContent='创建结果：'+error.message+
      '。如返回 UNKNOWN，请先检查 GitHub Issues，避免重复提交。';
  }finally{
    form.elements.namedItem('writeToken').value='';
    form.querySelector('button[type="submit"]').disabled=false;
  }
});
