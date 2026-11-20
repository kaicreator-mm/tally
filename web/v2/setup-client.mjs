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
