package server

import (
	"fmt"
	"net/http"
	"strings"
)

// ---------- Projects Management ----------

func projectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/projects" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Projects"))
	b.WriteString(`<style>
.proj-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:14px;margin-top:14px;}
.proj-card{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:14px;position:relative;transition:border-color .15s;}
.proj-card:hover{border-color:#0078d4;}
.proj-card .proj-name{font-size:15px;font-weight:600;color:#e0e0e0;margin-bottom:4px;}
.proj-card .proj-id{font-size:11px;color:#858585;font-family:monospace;margin-bottom:8px;}
.proj-card .proj-workdir{font-size:11px;color:#b0b0b0;margin-bottom:8px;word-break:break-all;}
.proj-card .proj-stats{display:flex;gap:12px;font-size:11px;color:#858585;}
.proj-card .proj-actions{display:flex;gap:6px;margin-top:10px;}
.modal-backdrop{display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.6);z-index:200;align-items:center;justify-content:center;}
.modal-backdrop.open{display:flex;}
.modal{background:#252526;border:1px solid #3c3c3c;border-radius:10px;padding:20px;width:440px;max-width:90vw;}
.modal h2{font-size:16px;margin:0 0 14px;color:#e0e0e0;}
.form-group{margin-bottom:12px;}
.form-group label{display:block;font-size:11px;color:#858585;margin-bottom:4px;text-transform:uppercase;letter-spacing:.05em;}
.form-row{display:flex;gap:8px;justify-content:flex-end;margin-top:16px;}
.btn-danger{background:#991b1b;border-color:#991b1b;color:#fecaca;}
.btn-danger:hover{background:#b91c1c;}
.btn-primary{background:#0078d4;border-color:#0078d4;color:#fff;}
.btn-primary:hover{background:#1c8ae8;}
</style>`)
	b.WriteString(`<div class="card">
<h1>Projects</h1>
<p>Manage your Cocopilot projects.</p>
<div class="meta">
<span id="proj-status">Loading...</span>
<button class="btn btn-primary" id="btn-create-proj" type="button">+ New Project</button>
</div>
<div class="proj-grid" id="proj-grid"></div>
</div>`)

	// Create project modal
	b.WriteString(`<div class="modal-backdrop" id="create-modal">
<div class="modal">
<h2 id="modal-title">New Project</h2>
<form id="proj-form">
<input type="hidden" id="edit-id">
<div class="form-group"><label>Name</label><input class="input" id="proj-name" required style="width:100%;"></div>
<div class="form-group"><label>Working Directory</label><input class="input" id="proj-workdir" placeholder="/path/to/repo" style="width:100%;"></div>
<div class="form-group"><label>Description</label><textarea class="textarea" id="proj-desc" style="width:100%;min-height:60px;"></textarea></div>
<div class="form-row">
<button class="btn" type="button" id="btn-cancel">Cancel</button>
<button class="btn btn-primary" type="submit" id="btn-submit">Create</button>
</div>
</form>
</div>
</div>`)

	// Delete confirm modal
	b.WriteString(`<div class="modal-backdrop" id="delete-modal">
<div class="modal">
<h2>Delete Project</h2>
<p id="delete-msg" style="margin:12px 0;"></p>
<div class="form-row">
<button class="btn" type="button" id="btn-del-cancel">Cancel</button>
<button class="btn btn-danger" type="button" id="btn-del-confirm">Delete</button>
</div>
</div>
</div>`)

	b.WriteString(`<script>`)
	b.WriteString(`const grid=document.getElementById('proj-grid');`)
	b.WriteString(`const statusEl=document.getElementById('proj-status');`)
	b.WriteString(`const createModal=document.getElementById('create-modal');`)
	b.WriteString(`const deleteModal=document.getElementById('delete-modal');`)
	b.WriteString(`const form=document.getElementById('proj-form');`)
	b.WriteString(`const nameEl=document.getElementById('proj-name');`)
	b.WriteString(`const workdirEl=document.getElementById('proj-workdir');`)
	b.WriteString(`const descEl=document.getElementById('proj-desc');`)
	b.WriteString(`const editIdEl=document.getElementById('edit-id');`)
	b.WriteString(`const modalTitle=document.getElementById('modal-title');`)
	b.WriteString(`const submitBtn=document.getElementById('btn-submit');`)
	b.WriteString(`const deleteMsg=document.getElementById('delete-msg');`)
	b.WriteString(`let deleteTarget=null;`)
	b.WriteString(`const esc=(v)=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)

	b.WriteString(`document.getElementById('btn-create-proj').addEventListener('click',()=>{editIdEl.value='';modalTitle.textContent='New Project';submitBtn.textContent='Create';nameEl.value='';workdirEl.value='';descEl.value='';createModal.classList.add('open');nameEl.focus();});`)
	b.WriteString(`document.getElementById('btn-cancel').addEventListener('click',()=>createModal.classList.remove('open'));`)
	b.WriteString(`document.getElementById('btn-del-cancel').addEventListener('click',()=>deleteModal.classList.remove('open'));`)

	b.WriteString(`function openEdit(p){editIdEl.value=p.id;modalTitle.textContent='Edit Project';submitBtn.textContent='Save';nameEl.value=p.name||'';workdirEl.value=p.workdir||'';descEl.value=p.description||'';createModal.classList.add('open');nameEl.focus();}`)
	b.WriteString(`function openDelete(p){deleteTarget=p.id;deleteMsg.textContent='Delete project "'+p.name+'" ('+p.id+')? This cannot be undone.';deleteModal.classList.add('open');}`)

	b.WriteString(`async function loadProjects(){statusEl.textContent='Loading...';grid.innerHTML='';try{`)
	b.WriteString(`const res=await fetch('/api/v2/projects');if(!res.ok)throw new Error();const data=await res.json();`)
	b.WriteString(`const projects=Array.isArray(data.projects)?data.projects:[];statusEl.textContent=projects.length+' projects';`)
	b.WriteString(`if(!projects.length){grid.innerHTML='<p class="muted">No projects yet. Click "+ New Project" to create one.</p>';return;}`)
	b.WriteString(`projects.forEach(p=>{const card=document.createElement('div');card.className='proj-card';`)
	b.WriteString(`card.innerHTML='<div class="proj-name">'+esc(p.name)+'</div>'+'<div class="proj-id">'+esc(p.id)+'</div>'`)
	b.WriteString(`+(p.workdir?'<div class="proj-workdir">\uD83D\uDCC2 '+esc(p.workdir)+'</div>':'')`)
	b.WriteString(`+(p.description?'<div style="font-size:12px;color:#b0b0b0;margin-bottom:8px;">'+esc(p.description)+'</div>':'')`)
	b.WriteString(`+'<div class="proj-actions">'`)
	b.WriteString(`+'<button class="btn btn-edit" data-id="'+esc(p.id)+'">Edit</button>'`)
	b.WriteString(`+(p.id!=='proj_default'?'<button class="btn btn-danger btn-del" data-id="'+esc(p.id)+'">Delete</button>':'')`)
	b.WriteString(`+'</div>';`)
	b.WriteString(`card.querySelector('.btn-edit').addEventListener('click',()=>openEdit(p));`)
	b.WriteString(`const delBtn=card.querySelector('.btn-del');if(delBtn)delBtn.addEventListener('click',()=>openDelete(p));`)
	b.WriteString(`grid.appendChild(card);});`)
	b.WriteString(`}catch(e){statusEl.textContent='Failed to load projects';}}`)

	b.WriteString(`form.addEventListener('submit',async(e)=>{e.preventDefault();const id=editIdEl.value;`)
	b.WriteString(`const body={};const n=nameEl.value.trim();if(n)body.name=n;const w=workdirEl.value.trim();if(w)body.workdir=w;const d=descEl.value.trim();if(d)body.description=d;`)
	b.WriteString(`try{const url=id?'/api/v2/projects/'+encodeURIComponent(id):'/api/v2/projects';`)
	b.WriteString(`const method=id?'PATCH':'POST';const res=await fetch(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});`)
	b.WriteString(`if(!res.ok){const err=await res.json().catch(()=>({}));pageToast((err.error&&err.error.message)||'Failed',true);return;}`)
	b.WriteString(`createModal.classList.remove('open');pageToast('Project saved');loadProjects();}catch(err){pageToast('Network error',true);}});`)

	b.WriteString(`document.getElementById('btn-del-confirm').addEventListener('click',async()=>{if(!deleteTarget)return;`)
	b.WriteString(`try{const res=await fetch('/api/v2/projects/'+encodeURIComponent(deleteTarget),{method:'DELETE'});`)
	b.WriteString(`if(!res.ok){const err=await res.json().catch(()=>({}));pageToast((err.error&&err.error.message)||'Delete failed',true);return;}`)
	b.WriteString(`deleteModal.classList.remove('open');deleteTarget=null;pageToast('Project deleted');loadProjects();}catch(e){pageToast('Network error',true);}});`)

	b.WriteString(`loadProjects();`)
	b.WriteString(`</script>`)
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

// ---------- Policies Management ----------

func policiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/policies" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Policies"))
	b.WriteString(`<style>
.pol-list{display:flex;flex-direction:column;gap:10px;margin-top:14px;}
.pol-card{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:14px;display:flex;align-items:center;gap:14px;transition:border-color .15s;}
.pol-card:hover{border-color:#0078d4;}
.pol-card .pol-info{flex:1;min-width:0;}
.pol-card .pol-name{font-size:14px;font-weight:600;color:#e0e0e0;}
.pol-card .pol-desc{font-size:11px;color:#858585;margin-top:2px;}
.pol-card .pol-rules{font-size:11px;color:#b0b0b0;margin-top:4px;font-family:monospace;}
.pol-card .pol-actions{display:flex;gap:6px;flex-shrink:0;}
.toggle{position:relative;display:inline-block;width:36px;height:20px;cursor:pointer;}
.toggle input{opacity:0;width:0;height:0;}
.toggle .slider{position:absolute;top:0;left:0;right:0;bottom:0;background:#505050;border-radius:10px;transition:.2s;}
.toggle .slider::before{content:'';position:absolute;height:14px;width:14px;left:3px;bottom:3px;background:#ccc;border-radius:50%;transition:.2s;}
.toggle input:checked+.slider{background:#0078d4;}
.toggle input:checked+.slider::before{transform:translateX(16px);}
.modal-backdrop{display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.6);z-index:200;align-items:center;justify-content:center;}
.modal-backdrop.open{display:flex;}
.modal{background:#252526;border:1px solid #3c3c3c;border-radius:10px;padding:20px;width:520px;max-width:90vw;max-height:80vh;overflow-y:auto;}
.modal h2{font-size:16px;margin:0 0 14px;color:#e0e0e0;}
.form-group{margin-bottom:12px;}
.form-group label{display:block;font-size:11px;color:#858585;margin-bottom:4px;text-transform:uppercase;letter-spacing:.05em;}
.form-row{display:flex;gap:8px;justify-content:flex-end;margin-top:16px;}
.btn-danger{background:#991b1b;border-color:#991b1b;color:#fecaca;}.btn-danger:hover{background:#b91c1c;}
.btn-primary{background:#0078d4;border-color:#0078d4;color:#fff;}.btn-primary:hover{background:#1c8ae8;}
.rule-row{background:#252526;border:1px solid #3c3c3c;border-radius:4px;padding:8px;margin-bottom:6px;display:flex;gap:8px;align-items:center;}
.rule-row select,.rule-row input{flex:1;}
.rule-row .btn-rm{background:#3c3c3c;border:none;color:#f14c4c;cursor:pointer;padding:4px 8px;border-radius:4px;font-size:11px;}
</style>`)
	b.WriteString(`<div class="card">
<h1>Policies</h1>
<p>Manage enforcement policies for your projects.</p>
<div class="meta">
<span id="pol-status">Loading...</span>
<button class="btn btn-primary" id="btn-create-pol" type="button">+ New Policy</button>
<button class="btn" id="pol-refresh" type="button">Refresh</button>
</div>
<div class="pol-list" id="pol-list"></div>
</div>`)

	// Create/Edit modal
	b.WriteString(`<div class="modal-backdrop" id="pol-modal">
<div class="modal">
<h2 id="pol-modal-title">New Policy</h2>
<form id="pol-form">
<input type="hidden" id="pol-edit-id">
<div class="form-group"><label>Name</label><input class="input" id="pol-name" required style="width:100%;"></div>
<div class="form-group"><label>Description</label><input class="input" id="pol-description" style="width:100%;"></div>
<div class="form-group"><label>Enabled</label><label class="toggle"><input type="checkbox" id="pol-enabled" checked><span class="slider"></span></label></div>
<div class="form-group"><label>Rules</label><div id="rules-container"></div>
<button class="btn" type="button" id="btn-add-rule">+ Add Rule</button></div>
<div class="form-row">
<button class="btn" type="button" id="pol-cancel">Cancel</button>
<button class="btn btn-primary" type="submit" id="pol-submit">Create</button>
</div>
</form>
</div>
</div>`)

	b.WriteString(`<script>`)
	b.WriteString(`const listEl=document.getElementById('pol-list');`)
	b.WriteString(`const statusEl=document.getElementById('pol-status');`)
	b.WriteString(`function pid(){return cocoProject.get();}`)
	b.WriteString(`window.addEventListener('coco:project-changed',loadPolicies);`)
	b.WriteString(`const modal=document.getElementById('pol-modal');`)
	b.WriteString(`const form=document.getElementById('pol-form');`)
	b.WriteString(`const editIdEl=document.getElementById('pol-edit-id');`)
	b.WriteString(`const titleEl=document.getElementById('pol-modal-title');`)
	b.WriteString(`const submitEl=document.getElementById('pol-submit');`)
	b.WriteString(`const nameEl=document.getElementById('pol-name');`)
	b.WriteString(`const descEl=document.getElementById('pol-description');`)
	b.WriteString(`const enabledEl=document.getElementById('pol-enabled');`)
	b.WriteString(`const rulesContainer=document.getElementById('rules-container');`)
	b.WriteString(`const ruleTypes=['automation.block','completion.block','task.create.block','task.update.block','task.delete.block'];`)
	b.WriteString(`const esc=(v)=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)

	b.WriteString(`function addRuleRow(type,reason){const row=document.createElement('div');row.className='rule-row';`)
	b.WriteString(`row.innerHTML='<select class="select rule-type">'+ruleTypes.map(t=>'<option value="'+t+'"'+(t===type?' selected':'')+'>'+t+'</option>').join('')+'</select>'`)
	b.WriteString(`+'<input class="input rule-reason" placeholder="Reason (optional)" value="'+(reason?esc(reason):'')+'">'`)
	b.WriteString(`+'<button class="btn-rm" type="button">&times;</button>';`)
	b.WriteString(`row.querySelector('.btn-rm').addEventListener('click',()=>row.remove());`)
	b.WriteString(`rulesContainer.appendChild(row);}`)

	b.WriteString(`function getRules(){return Array.from(rulesContainer.querySelectorAll('.rule-row')).map(r=>({type:r.querySelector('.rule-type').value,reason:r.querySelector('.rule-reason').value.trim()||undefined}));}`)

	b.WriteString(`document.getElementById('btn-create-pol').addEventListener('click',()=>{editIdEl.value='';titleEl.textContent='New Policy';submitEl.textContent='Create';nameEl.value='';descEl.value='';enabledEl.checked=true;rulesContainer.innerHTML='';addRuleRow();modal.classList.add('open');nameEl.focus();});`)
	b.WriteString(`document.getElementById('pol-cancel').addEventListener('click',()=>modal.classList.remove('open'));`)
	b.WriteString(`document.getElementById('btn-add-rule').addEventListener('click',()=>addRuleRow());`)

	b.WriteString(`function openEditPolicy(p){editIdEl.value=p.id;titleEl.textContent='Edit Policy';submitEl.textContent='Save';nameEl.value=p.name||'';descEl.value=p.description||'';enabledEl.checked=p.enabled!==false;rulesContainer.innerHTML='';`)
	b.WriteString(`(p.rules||[]).forEach(r=>addRuleRow(r.type,r.reason));if(!(p.rules||[]).length)addRuleRow();modal.classList.add('open');nameEl.focus();}`)

	b.WriteString(`async function togglePolicy(id,enable){try{await fetch('/api/v2/projects/'+encodeURIComponent(pid())+'/policies/'+encodeURIComponent(id)+'/'+(enable?'enable':'disable'),{method:'POST'});loadPolicies();}catch(e){}}`)

b.WriteString(`async function deletePolicy(id){if(!window._pageConfirm){deletePolicy2(id);return;}window._pageConfirm('Delete this policy?',()=>deletePolicy2(id));}async function deletePolicy2(id){try{const r=await fetch('/api/v2/projects/'+encodeURIComponent(pid())+'/policies/'+encodeURIComponent(id),{method:'DELETE'});if(r.ok){pageToast('Policy deleted');loadPolicies();}else{pageToast('Failed to delete policy',true);}}catch(e){pageToast('Error',true);}}`)

	b.WriteString(`async function loadPolicies(){statusEl.textContent='Loading...';listEl.innerHTML='';try{`)
	b.WriteString(`const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid())+'/policies?limit=100');if(!res.ok)throw new Error();`)
	b.WriteString(`const data=await res.json();const policies=Array.isArray(data.policies)?data.policies:[];`)
	b.WriteString(`statusEl.textContent=policies.length+' policies';`)
	b.WriteString(`if(!policies.length){listEl.innerHTML='<p class="muted">No policies. Click "+ New Policy" to create one.</p>';return;}`)
	b.WriteString(`policies.forEach(p=>{const card=document.createElement('div');card.className='pol-card';`)
	b.WriteString(`const rules=(p.rules||[]).map(r=>r.type).join(', ')||'No rules';`)
	b.WriteString(`card.innerHTML='<label class="toggle"><input type="checkbox" '+(p.enabled?'checked':'')+'><span class="slider"></span></label>'`)
	b.WriteString(`+'<div class="pol-info"><div class="pol-name">'+esc(p.name)+'</div>'`)
	b.WriteString(`+(p.description?'<div class="pol-desc">'+esc(p.description)+'</div>':'')`)
	b.WriteString(`+'<div class="pol-rules">'+esc(rules)+'</div></div>'`)
	b.WriteString(`+'<div class="pol-actions"><button class="btn pol-edit">Edit</button><button class="btn btn-danger pol-del">Delete</button></div>';`)
	b.WriteString(`card.querySelector('.toggle input').addEventListener('change',(e)=>togglePolicy(p.id,e.target.checked));`)
	b.WriteString(`card.querySelector('.pol-edit').addEventListener('click',()=>openEditPolicy(p));`)
	b.WriteString(`card.querySelector('.pol-del').addEventListener('click',()=>deletePolicy(p.id));`)
	b.WriteString(`listEl.appendChild(card);});`)
	b.WriteString(`}catch(e){statusEl.textContent='Failed to load policies';}}`)

	b.WriteString(`form.addEventListener('submit',async(e)=>{e.preventDefault();const id=editIdEl.value;`)
	b.WriteString(`const body={name:nameEl.value.trim(),description:descEl.value.trim(),enabled:enabledEl.checked,rules:getRules()};`)
	b.WriteString(`if(!body.name){nameEl.focus();return;}`)
	b.WriteString(`try{const url='/api/v2/projects/'+encodeURIComponent(pid())+'/policies'+(id?'/'+encodeURIComponent(id):'');`)
	b.WriteString(`const method=id?'PATCH':'POST';const res=await fetch(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});`)
	b.WriteString(`if(!res.ok){const err=await res.json().catch(()=>({}));pageToast((err.error&&err.error.message)||'Failed',true);return;}`)
	b.WriteString(`modal.classList.remove('open');pageToast('Policy saved');loadPolicies();}catch(err){pageToast('Network error',true);}});`)

	b.WriteString(`document.getElementById('pol-refresh').addEventListener('click',loadPolicies);`)
	b.WriteString(`projectEl.addEventListener('change',loadPolicies);`)
	b.WriteString(`loadPolicies();`)
	b.WriteString(`</script>`)
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

// ---------- Settings & Automation Rules ----------

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/settings" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Settings"))
	b.WriteString(`<style>
.section{margin-bottom:24px;}
.section h2{border-bottom:1px solid #3c3c3c;padding-bottom:6px;margin-bottom:12px;}
.rule-card{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;margin-bottom:8px;}
.rule-card .rule-name{font-size:14px;font-weight:600;color:#e0e0e0;margin-bottom:4px;}
.rule-card .rule-trigger{font-size:11px;color:#0078d4;}
.rule-card .rule-actions{font-size:11px;color:#858585;margin-top:4px;}
.config-table td:first-child{font-weight:600;color:#b0b0b0;white-space:nowrap;width:240px;}
.config-table td:nth-child(2){color:#ccc;font-family:monospace;word-break:break-all;}
.sim-form{display:flex;gap:8px;flex-wrap:wrap;align-items:end;margin:12px 0;}
.sim-form label{font-size:11px;color:#858585;}
.sim-output{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;font-family:monospace;font-size:12px;max-height:300px;overflow:auto;white-space:pre-wrap;}
</style>`)

	// Server config section
	b.WriteString(`<div class="card section">
<h1>Settings</h1>
<h2>Server Configuration</h2>
<p class="muted">Current runtime configuration (read-only, set via environment variables).</p>
<table class="config-table"><tbody id="config-body"><tr><td colspan="2">Loading...</td></tr></tbody></table>
</div>`)

	// Automation rules section
	b.WriteString(`<div class="card section">
<h2>Automation Rules</h2>
<p class="muted">Active automation rules (configured via COCO_AUTOMATION_RULES env var).</p>
<div class="meta">
<button class="btn" id="auto-refresh" type="button">Refresh</button>
<span id="auto-status"></span>
</div>
<div id="rules-list"></div>
</div>`)

	// Automation simulate section
	b.WriteString(`<div class="card section">
<h2>Simulate Automation</h2>
<p class="muted">Test what would happen if an event fired.</p>
<div class="sim-form">
<div><label>Event Kind</label><select class="select" id="sim-kind">
<option value="task.completed">task.completed</option>
<option value="task.failed">task.failed</option>
<option value="task.created">task.created</option>
<option value="run.completed">run.completed</option>
<option value="run.failed">run.failed</option>
</select></div>
<div><label>Task ID</label><input class="input" id="sim-task-id" placeholder="1" style="width:80px;"></div>
<div><label>Entity ID</label><input class="input" id="sim-entity-id" placeholder="1" style="width:80px;"></div>
<button class="btn btn-primary" id="btn-simulate" type="button">Simulate</button>
</div>
<div class="sim-output" id="sim-output">Click "Simulate" to preview automation actions.</div>
</div>`)

	b.WriteString(`<script>`)
	b.WriteString(`const esc=(v)=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString(`function pid(){return cocoProject.get();}`)
	b.WriteString(`window.addEventListener('coco:project-changed',function(){loadRules();});`)

	// Load server config
	b.WriteString(`async function loadConfig(){const body=document.getElementById('config-body');body.innerHTML='<tr><td colspan="2">Loading...</td></tr>';try{`)
	b.WriteString(`const res=await fetch('/api/v2/config');if(!res.ok)throw new Error();const data=await res.json();`)
	b.WriteString(`body.innerHTML='';const cfg=data.config||data;Object.entries(cfg).forEach(([k,v])=>{`)
	b.WriteString(`const tr=document.createElement('tr');tr.innerHTML='<td>'+esc(k)+'</td><td>'+esc(typeof v==='object'?JSON.stringify(v):String(v))+'</td>';body.appendChild(tr);});`)
	b.WriteString(`}catch(e){body.innerHTML='<tr><td colspan="2">Failed to load config</td></tr>';}}`)

	// Load automation rules
	b.WriteString(`async function loadRules(){const list=document.getElementById('rules-list');const status=document.getElementById('auto-status');`)
	b.WriteString(`status.textContent='Loading...';list.innerHTML='';try{`)
	b.WriteString(`const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid())+'/automation/rules');if(!res.ok)throw new Error();`)
	b.WriteString(`const data=await res.json();const rules=Array.isArray(data.rules)?data.rules:[];status.textContent=rules.length+' rules';`)
	b.WriteString(`if(!rules.length){list.innerHTML='<p class="muted">No automation rules configured. Set COCO_AUTOMATION_RULES env var.</p>';return;}`)
	b.WriteString(`rules.forEach((r,i)=>{const card=document.createElement('div');card.className='rule-card';`)
	b.WriteString(`const name=r.name||'Rule '+(i+1);const trigger=r.trigger||'unknown';`)
	b.WriteString(`const actions=(r.actions||[]).map(a=>a.type||'unknown').join(', ');`)
	b.WriteString(`card.innerHTML='<div class="rule-name">'+esc(name)+(r.enabled===false?' <span style="color:#f14c4c;">(disabled)</span>':'')+'</div>'`)
	b.WriteString(`+'<div class="rule-trigger">Trigger: '+esc(trigger)+'</div>'`)
	b.WriteString(`+'<div class="rule-actions">Actions: '+esc(actions)+'</div>';`)
	b.WriteString(`list.appendChild(card);});`)
	b.WriteString(`}catch(e){status.textContent='Failed to load rules';}}`)

	// Simulate
	b.WriteString(`document.getElementById('btn-simulate').addEventListener('click',async()=>{const output=document.getElementById('sim-output');output.textContent='Simulating...';try{`)
	b.WriteString(`const body={event:{kind:document.getElementById('sim-kind').value}};`)
	b.WriteString(`const taskId=document.getElementById('sim-task-id').value.trim();if(taskId)body.event.payload={task_id:parseInt(taskId,10)||taskId};`)
	b.WriteString(`const entityId=document.getElementById('sim-entity-id').value.trim();if(entityId)body.event.entity_id=entityId;`)
	b.WriteString(`const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid())+'/automation/simulate',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});`)
	b.WriteString(`const data=await res.json();output.textContent=JSON.stringify(data,null,2);`)
	b.WriteString(`}catch(e){output.textContent='Error: '+e.message;}});`)

	b.WriteString(`document.getElementById('auto-refresh').addEventListener('click',loadRules);`)
	b.WriteString(`projectEl.addEventListener('change',loadRules);`)
	b.WriteString(`loadConfig();loadRules();`)
	b.WriteString(`</script>`)
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

// ---------- Dependencies Manager ----------

func dependenciesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/dependencies" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Dependencies"))
	b.WriteString(`<style>
.dep-form{display:flex;gap:8px;align-items:end;flex-wrap:wrap;margin:14px 0;}
.dep-form label{font-size:11px;color:#858585;}
.dep-list{margin-top:14px;}
.dep-row{display:flex;align-items:center;gap:12px;padding:8px 0;border-bottom:1px solid #3c3c3c;font-size:12px;}
.dep-row .arrow{color:#0078d4;font-size:14px;}
.dep-row .task-id{font-family:monospace;color:#ccc;font-weight:600;}
.dep-row .btn-rm{background:none;border:1px solid #991b1b;color:#f14c4c;cursor:pointer;padding:2px 8px;border-radius:4px;font-size:11px;}
.dep-row .btn-rm:hover{background:#991b1b;color:#fff;}
.btn-primary{background:#0078d4;border-color:#0078d4;color:#fff;}.btn-primary:hover{background:#1c8ae8;}
</style>`)
	b.WriteString(`<div class="card">
<h1>Task Dependencies</h1>
<p>View and manage task dependency relationships.</p>

<div class="dep-form">
<div><label>Task ID</label><input class="input" id="dep-task" placeholder="1" style="width:100px;"></div>
<button class="btn" id="btn-load-deps" type="button">Load Dependencies</button>
</div>

<h2>Add Dependency</h2>
<div class="dep-form">
<div><label>Task</label><input class="input" id="dep-from" placeholder="Task ID" style="width:100px;"></div>
<div style="font-size:16px;color:#0078d4;align-self:center;">depends on &rarr;</div>
<div><label>Depends On</label><input class="input" id="dep-to" placeholder="Task ID" style="width:100px;"></div>
<button class="btn btn-primary" id="btn-add-dep" type="button">Add</button>
<span id="dep-add-status" class="muted"></span>
</div>

<h2>Current Dependencies</h2>
<div class="meta"><span id="dep-status"></span></div>
<div class="dep-list" id="dep-list"></div>
</div>`)

	b.WriteString(`<script>`)
	b.WriteString(`const listEl=document.getElementById('dep-list');`)
	b.WriteString(`const statusEl=document.getElementById('dep-status');`)
	b.WriteString(`const taskInput=document.getElementById('dep-task');`)
	b.WriteString(`const addStatus=document.getElementById('dep-add-status');`)
	b.WriteString(`const esc=(v)=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)

	b.WriteString(`async function loadDeps(){const id=taskInput.value.trim();if(!id){statusEl.textContent='Enter a task ID above.';return;}`)
	b.WriteString(`statusEl.textContent='Loading...';listEl.innerHTML='';try{`)
	b.WriteString(`const res=await fetch('/api/v2/tasks/'+encodeURIComponent(id)+'/dependencies');if(!res.ok)throw new Error();`)
	b.WriteString(`const data=await res.json();const deps=Array.isArray(data.dependencies)?data.dependencies:[];`)
	b.WriteString(`statusEl.textContent=deps.length+' dependencies for task #'+esc(id);`)
	b.WriteString(`if(!deps.length){listEl.innerHTML='<p class="muted">No dependencies.</p>';return;}`)
	b.WriteString(`deps.forEach(d=>{const row=document.createElement('div');row.className='dep-row';`)
	b.WriteString(`row.innerHTML='<span class="task-id">#'+esc(d.task_id)+'</span><span class="arrow">&rarr; depends on &rarr;</span><span class="task-id">#'+esc(d.depends_on_task_id)+'</span>'`)
	b.WriteString(`+'<button class="btn-rm" data-task="'+esc(d.task_id)+'" data-dep="'+esc(d.depends_on_task_id)+'">&times; Remove</button>';`)
	b.WriteString(`row.querySelector('.btn-rm').addEventListener('click',async(e)=>{const t=e.target.dataset.task;const dep=e.target.dataset.dep;`)
	b.WriteString(`try{await fetch('/api/v2/tasks/'+encodeURIComponent(t)+'/dependencies/'+encodeURIComponent(dep),{method:'DELETE'});loadDeps();}catch(err){}});`)
	b.WriteString(`listEl.appendChild(row);});`)
	b.WriteString(`}catch(e){statusEl.textContent='Failed to load dependencies';}}`)

	b.WriteString(`document.getElementById('btn-load-deps').addEventListener('click',loadDeps);`)
	b.WriteString(`taskInput.addEventListener('keydown',(e)=>{if(e.key==='Enter')loadDeps();});`)

	b.WriteString(`document.getElementById('btn-add-dep').addEventListener('click',async()=>{`)
	b.WriteString(`const from=document.getElementById('dep-from').value.trim();const to=document.getElementById('dep-to').value.trim();`)
	b.WriteString(`if(!from||!to){addStatus.textContent='Both task IDs required.';return;}`)
	b.WriteString(`addStatus.textContent='Adding...';try{`)
	b.WriteString(`const res=await fetch('/api/v2/tasks/'+encodeURIComponent(from)+'/dependencies',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({depends_on_task_id:parseInt(to,10)})});`)
	b.WriteString(`if(!res.ok){const err=await res.json().catch(()=>({}));addStatus.textContent=(err.error&&err.error.message)||'Failed';return;}`)
	b.WriteString(`addStatus.textContent='Added!';taskInput.value=from;loadDeps();`)
	b.WriteString(`}catch(e){addStatus.textContent='Network error';}});`)

	b.WriteString(`</script>`)
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

// ---------- Events Browser ----------

func eventsBrowserHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/events-browser" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Events"))
	b.WriteString(`<style>
.ev-filters{display:flex;gap:10px;flex-wrap:wrap;align-items:end;margin-bottom:14px;}
.ev-filters label{font-size:11px;color:#858585;}
.ev-list{font-size:12px;}
.ev-row{display:grid;grid-template-columns:60px 2fr 2fr 3fr 2fr;gap:10px;padding:8px 0;border-bottom:1px solid #3c3c3c;align-items:center;}
.ev-row.header{color:#0078d4;font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.05em;}
.ev-kind{font-family:monospace;font-size:11px;color:#ccc;}
.ev-payload{font-family:monospace;font-size:10px;color:#858585;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:300px;cursor:pointer;}
.ev-payload:hover{white-space:normal;color:#ccc;}
.stream-dot{display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px;}
.stream-dot.live{background:#89d185;animation:pulse 1.5s infinite;}
.stream-dot.off{background:#858585;}
@keyframes pulse{0%,100%{opacity:1;}50%{opacity:.4;}}
</style>`)

	b.WriteString(`<div class="card">
<h1>Events</h1>
<p>Browse and stream real-time system events.</p>
<div class="ev-filters">
<div><label>Project ID</label><input class="input" id="ev-project" placeholder="all" style="width:160px;"></div>
<div><label>Type</label><input class="input" id="ev-type" placeholder="task.completed" style="width:160px;"></div>
<div><label>Limit</label><input class="input" id="ev-limit" type="number" value="50" style="width:80px;"></div>
<button class="btn" id="ev-refresh" type="button">Load</button>
<button class="btn" id="ev-stream-btn" type="button"><span class="stream-dot off" id="stream-dot"></span>Stream</button>
</div>
<div class="meta"><span id="ev-status">Click "Load" to fetch events.</span></div>
<div class="ev-list" id="ev-list"></div>
</div>`)

	b.WriteString(`<script>`)
	b.WriteString(`const listEl=document.getElementById('ev-list');`)
	b.WriteString(`const statusEl=document.getElementById('ev-status');`)
	b.WriteString(`const esc=(v)=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString(`let evtSource=null;`)
	b.WriteString(`const dot=document.getElementById('stream-dot');`)

	b.WriteString(`function renderHeader(){const h=document.createElement('div');h.className='ev-row header';h.innerHTML='<span>ID</span><span>Kind</span><span>Entity</span><span>Payload</span><span>Time</span>';return h;}`)

	b.WriteString(`async function loadEvents(){statusEl.textContent='Loading...';listEl.innerHTML='';try{`)
	b.WriteString(`const params=new URLSearchParams();`)
	b.WriteString(`const proj=document.getElementById('ev-project').value.trim();if(proj)params.set('project_id',proj);`)
	b.WriteString(`const type=document.getElementById('ev-type').value.trim();if(type)params.set('type',type);`)
	b.WriteString(`params.set('limit',document.getElementById('ev-limit').value||'50');`)
	b.WriteString(`const res=await fetch('/api/v2/events?'+params.toString());if(!res.ok)throw new Error();`)
	b.WriteString(`const data=await res.json();const events=Array.isArray(data.events)?data.events:[];`)
	b.WriteString(`statusEl.textContent=events.length+' events';listEl.appendChild(renderHeader());`)
	b.WriteString(`events.forEach(ev=>{const row=document.createElement('div');row.className='ev-row';`)
	b.WriteString(`const payload=ev.payload?JSON.stringify(ev.payload):'';`)
	b.WriteString(`row.innerHTML='<span>'+esc(ev.id)+'</span><span class="ev-kind">'+esc(ev.kind||ev.type)+'</span>'`)
	b.WriteString(`+'<span>'+esc((ev.entity_type||'')+' '+(ev.entity_id||''))+'</span>'`)
	b.WriteString(`+'<span class="ev-payload" title="'+esc(payload)+'">'+esc(payload)+'</span>'`)
	b.WriteString(`+'<span>'+esc(ev.created_at||'')+'</span>';listEl.appendChild(row);});`)
	b.WriteString(`}catch(e){statusEl.textContent='Failed to load events';}}`)

	b.WriteString(`document.getElementById('ev-refresh').addEventListener('click',loadEvents);`)

	// SSE streaming toggle
	b.WriteString(`document.getElementById('ev-stream-btn').addEventListener('click',()=>{`)
	b.WriteString(`if(evtSource){evtSource.close();evtSource=null;dot.className='stream-dot off';return;}`)
	b.WriteString(`dot.className='stream-dot live';evtSource=new EventSource('/api/v2/events/stream');`)
	b.WriteString(`evtSource.addEventListener('event',(e)=>{try{const ev=JSON.parse(e.data);`)
	b.WriteString(`const row=document.createElement('div');row.className='ev-row';`)
	b.WriteString(`const payload=ev.payload?JSON.stringify(ev.payload):'';`)
	b.WriteString(`row.innerHTML='<span>'+esc(ev.id)+'</span><span class="ev-kind">'+esc(ev.kind||ev.type)+'</span>'`)
	b.WriteString(`+'<span>'+esc((ev.entity_type||'')+' '+(ev.entity_id||''))+'</span>'`)
	b.WriteString(`+'<span class="ev-payload" title="'+esc(payload)+'">'+esc(payload)+'</span>'`)
	b.WriteString(`+'<span>'+esc(ev.created_at||'')+'</span>';`)
	b.WriteString(`if(listEl.children.length<=1)listEl.appendChild(renderHeader());`)
	b.WriteString(`listEl.insertBefore(row,listEl.children[1]);}catch(err){}});`)
	b.WriteString(`evtSource.addEventListener('error',()=>{dot.className='stream-dot off';});});`)

	b.WriteString(`</script>`)
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}
