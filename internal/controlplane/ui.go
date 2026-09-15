package controlplane

import "net/http"

// console serves the deliberately thin v1.5 browser surface. All writes still
// go through the JSON API on this same handler, so the UI cannot become a
// second lifecycle/state authority. Authentication and CSRF are provided by
// the outer internal/web router.
func (h *HTTPHandler) console(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(controlPlaneConsoleHTML))
}

const controlPlaneConsoleHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Tally v1.5 Engineering Control Plane</title>
<style>
:root{color-scheme:light dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;--bg:#0f1115;--panel:#171a21;--line:#2b303b;--muted:#9aa4b2;--text:#eef2f7;--accent:#7aa2ff;--ok:#63c174;--warn:#e5b95c;--bad:#ef6b73}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text)}main{max-width:1320px;margin:auto;padding:28px}h1{font-size:24px;margin:0 0 4px}h2{font-size:16px;margin:0 0 14px}.sub{color:var(--muted);margin:0 0 22px}.toolbar,.grid,.actions{display:grid;gap:14px}.toolbar{grid-template-columns:minmax(220px,1fr) auto}.grid{grid-template-columns:1.35fr .65fr;margin-top:16px}.actions{grid-template-columns:repeat(3,minmax(0,1fr));margin-top:16px}.panel{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:16px;min-width:0}input,select,textarea,button{font:inherit;border-radius:8px;border:1px solid var(--line);background:#11141a;color:var(--text);padding:9px 11px}textarea{width:100%;min-height:180px;resize:vertical}button{cursor:pointer;background:#202634}button.primary{background:#2d4f91;border-color:#4469af}button.danger{background:#4a2529;border-color:#6a343a}.row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}.row>*{flex:1}.row button{flex:0 0 auto}.stages{display:flex;gap:8px;flex-wrap:wrap}.stage,.pill{border:1px solid var(--line);border-radius:999px;padding:5px 9px;font-size:12px;color:var(--muted)}.stage.on,.pill.ready,.pill.done,.pill.accepted{color:#dff6e4;border-color:#3e7348;background:#1b3020}.pill.blocked,.pill.interrupted{color:#ffe6b6;border-color:#7b6031;background:#352913}.pill.running{color:#dce8ff;border-color:#49679d;background:#1d2a44}.meta{font-size:12px;color:var(--muted)}table{width:100%;border-collapse:collapse;font-size:13px}th,td{text-align:left;border-bottom:1px solid var(--line);padding:9px 7px;vertical-align:top}tbody tr{cursor:pointer}tbody tr:hover{background:#1d222c}code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}.error{white-space:pre-wrap;color:#ffb8bd}.success{color:#a9e9b4}.audit{max-height:420px;overflow:auto}.audit-item{padding:8px 0;border-bottom:1px solid var(--line)}details summary{cursor:pointer;color:var(--muted)}pre{white-space:pre-wrap;word-break:break-word;background:#11141a;padding:10px;border-radius:8px;overflow:auto}.small{font-size:12px;color:var(--muted)}@media(max-width:900px){.grid,.actions,.toolbar{grid-template-columns:1fr}}
</style>
</head>
<body>
<main>
  <h1>Engineering Control Plane <span class="meta">v1.5.0</span></h1>
  <p class="sub">Active TaskDAG 是 Project Flow 唯一状态权威。Agent completion 只是 claim；Verification / CompletionContract / Authority 决定是否推进。</p>

  <section class="panel toolbar">
    <input id="project" autocomplete="off" placeholder="Project ID，例如 project-1">
    <button class="primary" id="load">加载项目</button>
  </section>
  <p id="message" class="small"></p>

  <section class="grid">
    <div class="panel">
      <h2>Lifecycle</h2>
      <div class="stages" id="stages"></div>
      <div style="height:14px"></div>
      <h2>Active TaskDAG</h2>
      <table>
        <thead><tr><th>Task</th><th>State</th><th>Dependencies</th><th>Blocked reason</th></tr></thead>
        <tbody id="tasks"><tr><td colspan="4" class="small">先加载 Project。</td></tr></tbody>
      </table>
      <div class="small" id="dagmeta" style="margin-top:10px"></div>
    </div>
    <div class="panel">
      <h2>Audit</h2>
      <div id="audit" class="audit small">尚未加载。</div>
      <details style="margin-top:14px"><summary>Raw snapshot</summary><pre id="raw">{}</pre></details>
    </div>
  </section>

  <section class="actions">
    <div class="panel">
      <h2>Authority decision</h2>
      <input id="auth-task" placeholder="Task ID">
      <div style="height:8px"></div>
      <input id="auth-requirement" placeholder="Authority requirement ID">
      <div style="height:8px"></div>
      <select id="auth-status"><option value="authorized">authorized</option><option value="rejected">rejected</option><option value="pending">pending</option></select>
      <div style="height:8px"></div>
      <input id="auth-actor" placeholder="Actor" value="owner">
      <div style="height:10px"></div>
      <button id="authority" class="primary">记录 Authority</button>
      <p class="small">该动作只追加 decision；CompletionContract 仍决定是否满足。</p>
    </div>

    <div class="panel">
      <h2>Restart task</h2>
      <input id="restart-task" placeholder="Task ID">
      <div style="height:10px"></div>
      <button id="restart">Restart → Ready</button>
      <p class="small">Running 会先 best-effort cancel 并恢复为 interrupted，再显式回到 ready。正确性不依赖 Agent session resume。</p>
    </div>

    <div class="panel">
      <h2>Dispatch</h2>
      <input id="dispatch-task" placeholder="Task ID">
      <div style="height:8px"></div>
      <textarea id="dispatch-json" spellcheck="false" placeholder='TaskExecutionRequest JSON，例如 {"frozen_task_contract":...,"runtime":...,"environment":...,"workspace":...,"instruction":"..."}'></textarea>
      <div style="height:10px"></div>
      <button id="dispatch" class="primary">Dispatch</button>
      <p class="small">Task ID 可省略，UI 会用上方 Task ID。生产执行由 Engineering Harness 统一控制 execute → verify/repair → accept。</p>
    </div>
  </section>
</main>
<script>
const $=id=>document.getElementById(id);
let snapshot=null;
function cookie(name){const p=document.cookie.split('; ').find(x=>x.startsWith(name+'='));return p?decodeURIComponent(p.slice(name.length+1)):''}
function project(){return $('project').value.trim()}
function apiBase(){const p=project();if(!p)throw new Error('Project ID 不能为空');return '/api/v15/projects/'+encodeURIComponent(p)}
function msg(text,ok=false){$('message').className=ok?'success':'error';$('message').textContent=text||''}
async function request(path,opts={}){
  const o={credentials:'same-origin',...opts};o.headers={...(opts.headers||{})};
  if(o.method&&o.method!=='GET'&&o.method!=='HEAD'){o.headers['X-CSRF-Token']=cookie('tally_csrf');if(o.body)o.headers['Content-Type']='application/json'}
  const res=await fetch(path,o);const text=await res.text();let data=null;try{data=text?JSON.parse(text):null}catch(_){data=text}
  if(!res.ok){throw new Error((data&&data.error)||text||('HTTP '+res.status))}return data;
}
function esc(v){return String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
function render(s){
  snapshot=s;$('raw').textContent=JSON.stringify(s,null,2);
  const stages=[['Requirements',!!s.requirements],['Project Plan',!!s.plan],['Architecture',!!s.architecture],['Active TaskDAG',!!s.active_task_dag]];
  $('stages').innerHTML=stages.map(x=>'<span class="stage '+(x[1]?'on':'')+'">'+esc(x[0])+(x[1]?' ✓':'')+'</span>').join('');
  const dag=s.active_task_dag, nodes=dag&&Array.isArray(dag.nodes)?dag.nodes:[];
  $('tasks').innerHTML=nodes.length?nodes.map(n=>'<tr data-task="'+esc(n.id)+'"><td><code>'+esc(n.id)+'</code></td><td><span class="pill '+esc(n.state)+'">'+esc(n.state)+'</span></td><td>'+esc((n.dependencies||[]).join(', '))+'</td><td>'+esc(n.blocked_reason||'')+'</td></tr>').join(''):'<tr><td colspan="4" class="small">Active TaskDAG 尚未激活。</td></tr>';
  $('dagmeta').textContent=dag?('revision '+(dag.revision_id||'')+' · '+nodes.length+' task(s) · '+Object.keys(s.completion_contracts||{}).length+' CompletionContract(s)'):'';
  document.querySelectorAll('#tasks tr[data-task]').forEach(tr=>tr.onclick=()=>{const id=tr.dataset.task;$('auth-task').value=id;$('restart-task').value=id;$('dispatch-task').value=id});
  const records=(s.audit&&s.audit.records)||[];
  $('audit').innerHTML=records.length?records.slice().reverse().slice(0,50).map(r=>'<div class="audit-item"><strong>'+esc(r.kind)+'</strong><br><code>'+esc(r.node_id||'project')+'</code> · '+esc(r.actor)+' · '+esc(r.at)+'</div>').join(''):'无 audit record。';
}
async function load(){try{msg('');const s=await request(apiBase()+'/control-plane');render(s);const u=new URL(location.href);u.searchParams.set('project',project());history.replaceState(null,'',u);msg('已加载 '+project(),true)}catch(e){msg(e.message)}}
$('load').onclick=load;$('project').addEventListener('keydown',e=>{if(e.key==='Enter')load()});
$('authority').onclick=async()=>{try{const task=$('auth-task').value.trim(),rid=$('auth-requirement').value.trim(),actor=$('auth-actor').value.trim();if(!task||!rid||!actor)throw new Error('Task / requirement / actor 必填');const now=new Date().toISOString();const body={requirement_id:rid,status:$('auth-status').value,actor,at:now,provenance:{actor,source:'v1.5-control-plane-ui',at:now}};render(await request(apiBase()+'/tasks/'+encodeURIComponent(task)+'/authority',{method:'POST',body:JSON.stringify(body)}));msg('Authority decision 已记录',true)}catch(e){msg(e.message)}};
$('restart').onclick=async()=>{try{const task=$('restart-task').value.trim();if(!task)throw new Error('Task ID 必填');render(await request(apiBase()+'/tasks/'+encodeURIComponent(task)+'/restart',{method:'POST'}));msg('Task 已请求 restart',true)}catch(e){msg(e.message)}};
$('dispatch').onclick=async()=>{try{const task=$('dispatch-task').value.trim();if(!task)throw new Error('Task ID 必填');const raw=$('dispatch-json').value.trim();const body=raw?JSON.parse(raw):{};if(body.task_id&&body.task_id!==task)throw new Error('JSON task_id 与 Task ID 不一致');if(!body.task_id)body.task_id=task;const result=await request(apiBase()+'/tasks/'+encodeURIComponent(task)+'/dispatch',{method:'POST',body:JSON.stringify(body)});if(result&&result.snapshot)render(result.snapshot);msg('Dispatch 已完成；请检查 Completion / blocked reason。',true)}catch(e){msg(e.message)}};
const initial=new URL(location.href).searchParams.get('project');if(initial){$('project').value=initial;load()}
</script>
</body>
</html>`
