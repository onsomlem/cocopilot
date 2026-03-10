package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Redirect root to dashboard as the default landing page
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func boardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/board" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlTemplate)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/dashboard" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Dashboard"))
	b.WriteString(`<style>
.dash-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin-bottom: 20px; }
.dash-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 16px; text-align: center; cursor: pointer; transition: border-color .15s; }
.dash-card:hover { border-color: var(--accent); }
.dash-card .dc-label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 6px; }
.dash-card .dc-value { font-size: 28px; font-weight: 700; color: var(--text); }
.dash-card .dc-action { font-size: 10px; color: var(--accent); margin-top: 6px; opacity: 0; transition: opacity .15s; }
.dash-card:hover .dc-action { opacity: 1; }
.dash-card.ok .dc-value { color: var(--success); }
.dash-card.warn .dc-value { color: var(--warning); }
.dash-card.error .dc-value { color: var(--error); }
.dash-card.info .dc-value { color: var(--info); }
.dash-section { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin-bottom: 12px; }
.dash-section h2 { font-size: 13px; color: #e0e0e0; margin-bottom: 10px; border: none; padding: 0; display: flex; align-items: center; justify-content: space-between; }
.dash-section h2 a { font-size: 11px; color: var(--accent); text-decoration: none; font-weight: 400; }
.dash-section h2 a:hover { text-decoration: underline; }
.dash-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.dash-table th { color: var(--accent); font-weight: 600; font-size: 11px; text-transform: uppercase; padding: 6px 8px; border-bottom: 1px solid var(--border); text-align: left; }
.dash-table td { padding: 6px 8px; border-bottom: 1px solid #333; color: var(--text); }
.dash-table tr { cursor: pointer; }
.dash-table tr:hover { background: #2a2d2e; }
.dash-empty { color: #666; text-align: center; padding: 16px; font-size: 12px; }
.dash-actions { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 10px; margin-bottom: 16px; }
.dash-action-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 12px 14px; display: flex; align-items: center; gap: 10px; cursor: pointer; transition: border-color .15s; text-decoration: none; color: var(--text); }
.dash-action-card:hover { border-color: var(--accent); }
.dash-action-card .da-icon { font-size: 18px; flex-shrink: 0; }
.dash-action-card .da-label { font-size: 12px; font-weight: 500; }
.dash-action-card .da-hint { font-size: 10px; color: var(--text-muted); margin-top: 2px; }
</style>`)
	b.WriteString(`<div class="card" style="margin-bottom:16px;"><h1>Dashboard</h1><p class="muted" style="margin:0;">Monitor your work queue and take action</p></div>`)
	// Onboarding banner (shown only when empty)
	b.WriteString(`<div id="seed-banner" style="display:none;background:#1e3a5f;border:1px solid #264f78;border-radius:8px;padding:20px 24px;margin-bottom:16px;">
<div style="font-size:16px;font-weight:600;color:#56b6f7;margin-bottom:12px;">Welcome to Cocopilot</div>
<div style="display:flex;gap:16px;flex-wrap:wrap;">
<div id="step-1" style="flex:1;min-width:180px;background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:14px;">
<div style="font-size:11px;color:var(--accent);font-weight:600;margin-bottom:6px;">STEP 1</div>
<div style="font-size:13px;color:var(--text);margin-bottom:8px;">Seed sample tasks</div>
<div style="font-size:11px;color:var(--text-muted);margin-bottom:10px;">Creates 5 example tasks with dependencies.</div>
<button onclick="seedDemo()" class="btn btn-primary" style="font-size:12px;">Seed Demo Data</button>
</div>
<div id="step-2" style="flex:1;min-width:180px;background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:14px;opacity:0.5;">
<div style="font-size:11px;color:var(--accent);font-weight:600;margin-bottom:6px;">STEP 2</div>
<div style="font-size:13px;color:var(--text);margin-bottom:8px;">Start the built-in worker</div>
<div style="font-size:11px;color:var(--text-muted);margin-bottom:10px;">Run in a second terminal:</div>
<code style="display:block;background:var(--bg);border:1px solid var(--border);border-radius:4px;padding:6px 10px;font-size:11px;color:var(--success);word-break:break-all;">go run ./cmd/cocopilot worker proj_default</code>
</div>
<div id="step-3" style="flex:1;min-width:180px;background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:14px;opacity:0.5;">
<div style="font-size:11px;color:var(--accent);font-weight:600;margin-bottom:6px;">STEP 3</div>
<div style="font-size:13px;color:var(--text);margin-bottom:8px;">Watch the board</div>
<div style="font-size:11px;color:var(--text-muted);">Open the <a href="/board" style="color:var(--accent);">Work Queue</a> to see tasks move in real time.</div>
</div>
</div>
</div>`)
	// Quick actions bar (contextual, shown based on state)
	b.WriteString(`<div id="dash-actions" class="dash-actions" style="display:none;"></div>`)
	// Stats grid
	b.WriteString(`<div class="dash-grid" id="dash-stats"><div class="dash-empty">Loading...</div></div>`)
	// Main content
	b.WriteString(`<div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;">
<div class="dash-section">
<h2>Needs Attention <a href="/board">View queue &rarr;</a></h2>
<table class="dash-table"><thead><tr><th>Task</th><th>Status</th><th>Updated</th></tr></thead>
<tbody id="dash-attention"><tr><td colspan="3" class="dash-empty">Loading...</td></tr></tbody></table>
</div>
<div class="dash-section">
<h2>Connected Agents <a href="/agents">View all &rarr;</a></h2>
<div id="dash-agents" class="dash-empty">Loading...</div>
</div>
</div>`)
	b.WriteString(`<div class="dash-section">
<h2>Recent Activity <a href="/events-browser">Events &rarr;</a></h2>
<table class="dash-table"><thead><tr><th>Task</th><th>Status</th><th>Agent</th><th>Updated</th></tr></thead>
<tbody id="dash-recent"><tr><td colspan="4" class="dash-empty">Loading...</td></tr></tbody></table>
</div>`)
	b.WriteString(`<script>`)
	b.WriteString(`async function loadDashboard(){try{`)
	b.WriteString(`const [tasksRes,agentsRes]=await Promise.all([fetch('/api/v2/tasks?limit=50'),fetch('/api/v2/agents')]);`)
	b.WriteString(`const tasksData=tasksRes.ok?await tasksRes.json():{tasks:[]};const agentsData=agentsRes.ok?await agentsRes.json():{agents:[]};`)
	b.WriteString(`const tasks=Array.isArray(tasksData.tasks)?tasksData.tasks:[];const agents=Array.isArray(agentsData.agents)?agentsData.agents:[];`)
	b.WriteString(`document.getElementById('seed-banner').style.display=tasks.length===0?'':'none';`)
	// Stats (clickable)
	b.WriteString(`const todo=tasks.filter(t=>t.status==='TODO'||t.status==='PENDING').length;`)
	b.WriteString(`const inProg=tasks.filter(t=>t.status==='IN_PROGRESS').length;`)
	b.WriteString(`const done=tasks.filter(t=>t.status==='DONE'||t.status==='COMPLETED').length;`)
	b.WriteString(`const failed=tasks.filter(t=>t.status==='FAILED').length;`)
	b.WriteString(`const blocked=tasks.filter(t=>t.is_blocked).length;`)
	b.WriteString(`const onlineAgents=agents.filter(a=>a.status==='ONLINE'||a.status==='active').length;`)
	b.WriteString(`document.getElementById('dash-stats').innerHTML=`)
	b.WriteString(`'<div class="dash-card info" onclick="location.href=\\'/board\\'"><div class="dc-label">Queued</div><div class="dc-value">'+todo+'</div><div class="dc-action">Open queue</div></div>'`)
	b.WriteString(`+'<div class="dash-card warn" onclick="location.href=\\'/board\\'"><div class="dc-label">In Progress</div><div class="dc-value">'+inProg+'</div><div class="dc-action">View active</div></div>'`)
	b.WriteString(`+'<div class="dash-card ok" onclick="location.href=\\'/board\\'"><div class="dc-label">Completed</div><div class="dc-value">'+done+'</div><div class="dc-action">View completed</div></div>'`)
	b.WriteString(`+'<div class="dash-card'+(failed>0?' error':'')+'" onclick="location.href=\\'/board\\'"><div class="dc-label">Failed</div><div class="dc-value">'+failed+'</div>'+(failed>0?'<div class="dc-action">Review failures</div>':'')+'</div>'`)
	b.WriteString(`+'<div class="dash-card'+(blocked>0?' warn':'')+'" onclick="location.href=\\'/dependencies\\'"><div class="dc-label">Blocked</div><div class="dc-value">'+blocked+'</div>'+(blocked>0?'<div class="dc-action">Unblock</div>':'')+'</div>'`)
	b.WriteString(`+'<div class="dash-card'+(onlineAgents>0?' ok':'')+ '" onclick="location.href=\\'/agents\\'"><div class="dc-label">Agents</div><div class="dc-value">'+onlineAgents+'/'+agents.length+'</div><div class="dc-action">Manage agents</div></div>';`)
	// Contextual quick actions
	b.WriteString(`const actionsEl=document.getElementById('dash-actions');const acts=[];`)
	b.WriteString(`if(failed>0)acts.push('<a class="dash-action-card" href="/board"><span class="da-icon">&#9888;</span><div><div class="da-label">'+failed+' failed task'+(failed>1?'s':'')+'</div><div class="da-hint">Review and retry</div></div></a>');`)
	b.WriteString(`if(onlineAgents===0&&tasks.length>0)acts.push('<a class="dash-action-card" href="/agents"><span class="da-icon">&#9881;</span><div><div class="da-label">No agents connected</div><div class="da-hint">Start a worker to process queued tasks</div></div></a>');`)
	b.WriteString(`if(blocked>0)acts.push('<a class="dash-action-card" href="/dependencies"><span class="da-icon">&#128279;</span><div><div class="da-label">'+blocked+' blocked task'+(blocked>1?'s':'')+'</div><div class="da-hint">Check dependency status</div></div></a>');`)
	b.WriteString(`if(acts.length>0){actionsEl.innerHTML=acts.join('');actionsEl.style.display='';}else{actionsEl.style.display='none';}`)
	// Needs attention: failed + blocked + stale
	b.WriteString(`const attention=tasks.filter(t=>t.status==='FAILED'||t.is_blocked).sort((a,b)=>(b.updated_at||'').localeCompare(a.updated_at||'')).slice(0,5);`)
	b.WriteString(`const attnBody=document.getElementById('dash-attention');`)
	b.WriteString(`if(attention.length===0){attnBody.innerHTML='<tr><td colspan="3" class="dash-empty" style="color:var(--success);">Nothing needs attention</td></tr>';}else{`)
	b.WriteString(`attnBody.innerHTML='';attention.forEach(t=>{const tr=document.createElement('tr');tr.onclick=()=>location.href='/board?task='+t.id;`)
	b.WriteString(`tr.innerHTML='<td>#'+t.id+' '+escapeHtml(t.title||'')+'</td><td>'+statusBadge(t.status_v2||t.status)+'</td><td>'+formatAgo(t.updated_at)+'</td>';`)
	b.WriteString(`attnBody.appendChild(tr);});}`)
	// Recent tasks
	b.WriteString(`const sorted=[...tasks].sort((a,b)=>(b.updated_at||b.created_at||'').localeCompare(a.updated_at||a.created_at||'')).slice(0,8);`)
	b.WriteString(`const tbody=document.getElementById('dash-recent');`)
	b.WriteString(`if(sorted.length===0){tbody.innerHTML='<tr><td colspan="4" class="dash-empty">No activity yet</td></tr>';}else{`)
	b.WriteString(`tbody.innerHTML='';sorted.forEach(t=>{const tr=document.createElement('tr');tr.onclick=()=>location.href='/board?task='+t.id;`)
	b.WriteString(`tr.innerHTML='<td>#'+t.id+' '+escapeHtml(t.title||'')+'</td><td>'+statusBadge(t.status_v2||t.status)+'</td><td>'+escapeHtml(t.agent_id||'—')+'</td><td>'+formatAgo(t.updated_at)+'</td>';`)
	b.WriteString(`tbody.appendChild(tr);});}`)
	// Agents
	b.WriteString(`const agentsDiv=document.getElementById('dash-agents');`)
	b.WriteString(`if(agents.length===0){agentsDiv.innerHTML='<div class="dash-empty">No agents connected.<br><span style="font-size:11px;color:#666;">Start a worker or connect an agent to begin.</span></div>';}else{`)
	b.WriteString(`agentsDiv.innerHTML='';agents.forEach(a=>{const d=document.createElement('div');d.onclick=()=>location.href='/agents?id='+encodeURIComponent(a.id);`)
	b.WriteString(`d.style.cssText='display:flex;justify-content:space-between;padding:6px 0;border-bottom:1px solid #333;font-size:12px;cursor:pointer;';`)
	b.WriteString(`d.innerHTML='<span>'+escapeHtml(a.name||a.id)+'</span><span>'+statusBadge(a.status)+'</span>';`)
	b.WriteString(`agentsDiv.appendChild(d);});}`)
	b.WriteString(`}catch(e){document.getElementById('dash-stats').innerHTML='<div class="dash-card error"><div class="dc-label">Error</div><div class="dc-value">!</div></div>';}}`)
	b.WriteString(`loadDashboard();setInterval(loadDashboard,15000);`)
	b.WriteString(`async function seedDemo(){try{const res=await fetch('/api/v2/seed-demo',{method:'POST'});`)
	b.WriteString(`if(res.ok){document.getElementById('step-1').style.opacity='0.5';document.getElementById('step-1').querySelector('button').disabled=true;document.getElementById('step-1').querySelector('button').textContent='Seeded \u2713';document.getElementById('step-2').style.opacity='1';document.getElementById('step-3').style.opacity='1';loadDashboard();pageToast('Demo data seeded! Now start the worker (step 2).','ok');}`)
	b.WriteString(`else{const e=await res.json();pageToast(e.error?.message||'Seed failed','error');}}catch(err){pageToast(err.message,'error');}}`)
	b.WriteString(`</script>`)
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

// subPageHead returns a consistent HTML head + nav bar for all sub-pages.
// Uses the shared UI framework from ui_framework.go.
func subPageHead(title string) string {
	return `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">` +
		`<meta name="viewport" content="width=device-width, initial-scale=1.0">` +
		`<title>` + html.EscapeString(title) + ` - Cocopilot</title>` +
		`<style>` + uiSharedCSS() +
		// Legacy compat styles
		`.meta{display:flex;flex-wrap:wrap;align-items:center;gap:12px;margin-bottom:14px;font-size:12px;color:#b0b0b0;}` +
		`.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px;margin-bottom:14px;}` +
		`.pill{background:var(--bg);border:1px solid var(--border);border-radius:999px;padding:4px 10px;font-size:12px;}` +
		`</style>` +
		`<script>` + uiSharedJS() + `</script>` +
		`</head><body>` +
		pageNav() +
		`<div class="page-body">`
}

// subPageFoot returns a consistent footer + closing tags for sub-pages.
func subPageFoot() string {
	return `</div><div class="footer">Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a></div></body></html>`
}

func uiPlaceholderPage(title string, subtitle string, details []string) string {
	var b strings.Builder
	b.WriteString(subPageHead(title))
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>")
	if subtitle != "" {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(subtitle))
		b.WriteString("</p>")
	}
	if len(details) > 0 {
		b.WriteString("<ul style=\"margin:12px 0 0;padding-left:20px;\">")
		for _, item := range details {
			b.WriteString("<li style=\"margin:4px 0;\">")
			b.WriteString(html.EscapeString(item))
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
	}

	b.WriteString("</div>")
	b.WriteString(subPageFoot())
	return b.String()
}

func runViewerPage(runID string) string {
	var b strings.Builder
	b.WriteString(subPageHead("Run Viewer"))
	b.WriteString("<style>")
	b.WriteString(".pill.s-RUNNING{border-color:#0078d4;color:#0078d4;} .pill.s-SUCCEEDED{border-color:#89d185;color:#89d185;}")
	b.WriteString(".pill.s-FAILED{border-color:#f14c4c;color:#f14c4c;} .pill.s-PENDING{border-color:#858585;color:#858585;}")
	b.WriteString(".stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:10px;margin-bottom:14px;}")
	// Step timeline
	b.WriteString(".step-timeline{display:flex;flex-direction:column;gap:0;margin-bottom:14px;}")
	b.WriteString(".step-item{display:flex;gap:12px;padding:8px 0;border-left:2px solid #3c3c3c;margin-left:8px;padding-left:16px;position:relative;}")
	b.WriteString(".step-item::before{content:'';position:absolute;left:-5px;top:12px;width:8px;height:8px;border-radius:50%;background:#505050;}")
	b.WriteString(".step-item.s-completed::before{background:#89d185;} .step-item.s-running::before{background:#0078d4;} .step-item.s-failed::before{background:#f14c4c;}")
	b.WriteString(".step-info{flex:1;} .step-name{font-size:13px;font-weight:600;color:#ccc;} .step-meta{font-size:11px;color:#858585;margin-top:2px;}")
	b.WriteString(".step-dur{font-size:11px;color:#858585;white-space:nowrap;}")
	// Log console
	b.WriteString(".log-console{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:0;margin-bottom:14px;max-height:350px;overflow:auto;font-family:monospace;font-size:12px;}")
	b.WriteString(".log-console .log-line{padding:2px 12px;border-bottom:1px solid #252526;white-space:pre-wrap;word-break:break-all;}")
	b.WriteString(".log-console .log-line:nth-child(odd){background:#1a1a1a;}")
	b.WriteString(".log-console .log-ts{color:#858585;margin-right:8px;} .log-console .log-lvl{font-weight:600;margin-right:6px;}")
	b.WriteString(".log-console .log-lvl.l-error{color:#f14c4c;} .log-console .log-lvl.l-warn{color:#cca700;} .log-console .log-lvl.l-info{color:#3794ff;}")
	b.WriteString(".log-console .log-msg{color:#ccc;}")
	// Artifacts
	b.WriteString(".artifact-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px;margin-bottom:14px;}")
	b.WriteString(".artifact-card{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:10px 12px;}")
	b.WriteString(".artifact-card .art-name{font-size:13px;font-weight:600;color:#0078d4;margin-bottom:4px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}")
	b.WriteString(".artifact-card .art-meta{font-size:11px;color:#858585;}")
	b.WriteString(".artifact-card .art-link{display:inline-block;margin-top:6px;font-size:11px;color:#0078d4;text-decoration:none;}")
	b.WriteString(".artifact-card .art-link:hover{text-decoration:underline;}")
	// Tabs
	b.WriteString(".tabs{display:flex;gap:0;margin-bottom:14px;border-bottom:1px solid #3c3c3c;}")
	b.WriteString(".tab-btn{background:none;border:none;color:#858585;padding:8px 16px;cursor:pointer;font-size:13px;border-bottom:2px solid transparent;}")
	b.WriteString(".tab-btn.active{color:#0078d4;border-bottom-color:#0078d4;} .tab-btn:hover{color:#ccc;}")
	b.WriteString(".tab-content{display:none;} .tab-content.active{display:block;}")
	b.WriteString("pre{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;font-size:12px;overflow:auto;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\" data-run-id=\"")
	b.WriteString(html.EscapeString(runID))
	b.WriteString("\">")
	b.WriteString("<h1>Run Viewer</h1>")
	b.WriteString("<p>Run ID: ")
	b.WriteString(html.EscapeString(runID))
	b.WriteString("</p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<span class=\"pill\" id=\"run-status\">Loading...</span>")
	b.WriteString("<span class=\"muted\" id=\"run-timing\"></span>")
	b.WriteString("<span class=\"muted\">/api/v2/runs/")
	b.WriteString(html.EscapeString(runID))
	b.WriteString("</span>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"stats\">")
	b.WriteString("<div class=\"stat\"><span>Steps</span><strong id=\"run-steps\">-</strong></div>")
	b.WriteString("<div class=\"stat\"><span>Artifacts</span><strong id=\"run-artifacts\">-</strong></div>")
	b.WriteString("<div class=\"stat\"><span>Logs</span><strong id=\"run-logs\">-</strong></div>")
	b.WriteString("<div class=\"stat\"><span>Tool Calls</span><strong id=\"run-tools\">-</strong></div>")
	b.WriteString("</div>")
	// Tabs
	b.WriteString("<div class=\"tabs\">")
	b.WriteString("<button class=\"tab-btn active\" data-tab=\"steps\">Steps</button>")
	b.WriteString("<button class=\"tab-btn\" data-tab=\"logs\">Logs</button>")
	b.WriteString("<button class=\"tab-btn\" data-tab=\"artifacts\">Artifacts</button>")
	b.WriteString("<button class=\"tab-btn\" data-tab=\"json\">Raw JSON</button>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"tab-content active\" id=\"tab-steps\"><div class=\"step-timeline\" id=\"step-timeline\"><span class=\"muted\">Loading steps...</span></div></div>")
	b.WriteString("<div class=\"tab-content\" id=\"tab-logs\"><div class=\"log-console\" id=\"log-console\"><div class=\"log-line\"><span class=\"log-msg muted\">Loading logs...</span></div></div></div>")
	b.WriteString("<div class=\"tab-content\" id=\"tab-artifacts\"><div class=\"artifact-list\" id=\"artifact-list\"><span class=\"muted\">Loading artifacts...</span></div></div>")
	b.WriteString("<div class=\"tab-content\" id=\"tab-json\"><pre id=\"run-json\">Loading...</pre></div>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const card=document.querySelector('[data-run-id]');")
	b.WriteString("const runId=card?card.dataset.runId:'';")
	b.WriteString("const statusEl=document.getElementById('run-status');")
	b.WriteString("const timingEl=document.getElementById('run-timing');")
	b.WriteString("const stepsEl=document.getElementById('run-steps');")
	b.WriteString("const artifactsEl=document.getElementById('run-artifacts');")
	b.WriteString("const logsEl=document.getElementById('run-logs');")
	b.WriteString("const toolsEl=document.getElementById('run-tools');")
	b.WriteString("const jsonEl=document.getElementById('run-json');")
	b.WriteString("const stepTimeline=document.getElementById('step-timeline');")
	b.WriteString("const logConsole=document.getElementById('log-console');")
	b.WriteString("const artifactList=document.getElementById('artifact-list');")
	b.WriteString("const countOf=(v)=>Array.isArray(v)?v.length:0;")
	// Tab switching
	b.WriteString("document.querySelectorAll('.tab-btn').forEach(btn=>{btn.addEventListener('click',()=>{")
	b.WriteString("document.querySelectorAll('.tab-btn').forEach(b=>b.classList.remove('active'));")
	b.WriteString("document.querySelectorAll('.tab-content').forEach(c=>c.classList.remove('active'));")
	b.WriteString("btn.classList.add('active');document.getElementById('tab-'+btn.dataset.tab).classList.add('active');});});")
	// Render steps
	b.WriteString("function renderSteps(steps){stepTimeline.innerHTML='';if(!steps||!steps.length){stepTimeline.innerHTML='<span class=\"muted\">No steps recorded</span>';return;}")
	b.WriteString("steps.forEach((s,i)=>{const div=document.createElement('div');")
	b.WriteString("const st=(s.status||'pending').toLowerCase();div.className='step-item s-'+st;")
	b.WriteString("const name=s.name||s.type||'Step '+(i+1);")
	b.WriteString("const ts=s.started_at?new Date(s.started_at).toLocaleTimeString():'';")
	b.WriteString("const dur=s.started_at&&s.completed_at?((new Date(s.completed_at)-new Date(s.started_at))/1000).toFixed(1)+'s':'';")
	b.WriteString("div.innerHTML='<div class=\"step-info\"><div class=\"step-name\">'+escapeHtml(name)+'</div><div class=\"step-meta\">'+escapeHtml(st)+' &middot; '+escapeHtml(ts)+'</div></div><div class=\"step-dur\">'+escapeHtml(dur)+'</div>';")
	b.WriteString("stepTimeline.appendChild(div);});}")
	// Render logs
	b.WriteString("function renderLogs(logs){logConsole.innerHTML='';if(!logs||!logs.length){logConsole.innerHTML='<div class=\"log-line\"><span class=\"log-msg muted\">No logs</span></div>';return;}")
	b.WriteString("logs.forEach(log=>{const div=document.createElement('div');div.className='log-line';")
	b.WriteString("const ts=log.timestamp?new Date(log.timestamp).toLocaleTimeString():'';")
	b.WriteString("const lvl=(log.level||'info').toLowerCase();")
	b.WriteString("div.innerHTML='<span class=\"log-ts\">'+escapeHtml(ts)+'</span><span class=\"log-lvl l-'+lvl+'\">'+escapeHtml(lvl)+'</span><span class=\"log-msg\">'+escapeHtml(log.message||log.content||JSON.stringify(log))+'</span>';")
	b.WriteString("logConsole.appendChild(div);});logConsole.scrollTop=logConsole.scrollHeight;}")
	// Render artifacts
	b.WriteString("function renderArtifacts(artifacts){artifactList.innerHTML='';if(!artifacts||!artifacts.length){artifactList.innerHTML='<span class=\"muted\">No artifacts</span>';return;}")
	b.WriteString("artifacts.forEach(a=>{const div=document.createElement('div');div.className='artifact-card';")
	b.WriteString("const name=a.name||a.path||a.id||'Artifact';const mime=a.mime_type||a.content_type||'';")
	b.WriteString("const size=a.size?((a.size/1024).toFixed(1)+' KB'):'';")
	b.WriteString("div.innerHTML='<div class=\"art-name\">'+escapeHtml(name)+'</div><div class=\"art-meta\">'+escapeHtml(mime+(size?' &middot; '+size:''))+'</div>'")
	b.WriteString("+(a.id?'<a class=\"art-link\" href=\"/api/v2/artifacts/'+encodeURIComponent(a.id)+'/content\" target=\"_blank\">Download</a>':'');")
	b.WriteString("artifactList.appendChild(div);});}")
	// Load run
	b.WriteString("async function loadRun(){statusEl.textContent='Loading...';")
	b.WriteString("try{const res=await fetch('/api/v2/runs/'+encodeURIComponent(runId));")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const run=data&&data.run?data.run:data;")
	b.WriteString("const status=(run&&run.status?run.status:'UNKNOWN').toUpperCase();")
	b.WriteString("statusEl.textContent=status;statusEl.className='pill s-'+status;")
	b.WriteString("stepsEl.textContent=countOf(run&&run.steps);")
	b.WriteString("artifactsEl.textContent=countOf(run&&run.artifacts);logsEl.textContent=countOf(run&&run.logs);")
	b.WriteString("toolsEl.textContent=countOf(run&&run.tool_invocations);")
	b.WriteString("if(run&&run.started_at){let t='Started: '+new Date(run.started_at).toLocaleString();if(run.completed_at)t+=' | Ended: '+new Date(run.completed_at).toLocaleString();timingEl.textContent=t;}")
	b.WriteString("renderSteps(run&&run.steps);renderLogs(run&&run.logs);renderArtifacts(run&&run.artifacts);")
	b.WriteString("jsonEl.textContent=JSON.stringify(run,null,2);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load run';")
	b.WriteString("jsonEl.textContent=String(err&&err.message?err.message:err);}")
	b.WriteString("}")
	b.WriteString("loadRun();setInterval(loadRun,10000);")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	return b.String()
}

func contextPackDetailPage(packID string) string {
	var b strings.Builder
	b.WriteString(subPageHead("Context Pack"))
	b.WriteString("<style>")
	b.WriteString("ul{list-style:none;padding:0;margin:0;display:grid;gap:6px;font-size:12px;}")
	b.WriteString("li{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:4px;padding:6px 8px;}")
	b.WriteString(".panel{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;}")
	b.WriteString(".panel span{display:block;font-size:11px;color:#858585;margin-bottom:6px;}")
	b.WriteString(".summary{font-size:13px;color:#ccc;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\" data-pack-id=\"")
	b.WriteString(html.EscapeString(packID))
	b.WriteString("\">")
	b.WriteString("<h1>Context Pack</h1>")
	b.WriteString("<p>Pack ID: ")
	b.WriteString(html.EscapeString(packID))
	b.WriteString("</p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<span class=\"pill\" id=\"pack-status\">Loading...</span>")
	b.WriteString("<span class=\"muted\">/api/v2/context-packs/")
	b.WriteString(html.EscapeString(packID))
	b.WriteString("</span>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"grid\">")
	b.WriteString("<div class=\"panel\"><span>Summary</span><div id=\"pack-summary\" class=\"summary\">-</div></div>")
	b.WriteString("<div class=\"panel\"><span>Files</span><ul id=\"pack-files\"></ul></div>")
	b.WriteString("</div>")
	b.WriteString("<pre id=\"pack-json\">Loading...</pre>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const card=document.querySelector('[data-pack-id]');")
	b.WriteString("const packId=card?card.dataset.packId:'';")
	b.WriteString("const statusEl=document.getElementById('pack-status');")
	b.WriteString("const summaryEl=document.getElementById('pack-summary');")
	b.WriteString("const filesEl=document.getElementById('pack-files');")
	b.WriteString("const jsonEl=document.getElementById('pack-json');")
	b.WriteString("function setFiles(files){filesEl.innerHTML='';")
	b.WriteString("if(!files.length){const li=document.createElement('li');li.className='muted';li.textContent='No files';filesEl.appendChild(li);return;}")
	b.WriteString("files.forEach((file)=>{const li=document.createElement('li');")
	b.WriteString("const path=file&&file.path?file.path:'(unknown path)';")
	b.WriteString("const snippets=Array.isArray(file&&file.snippets)?file.snippets.length:0;")
	b.WriteString("li.textContent=snippets>0?path+' ('+snippets+' snippets)':path;filesEl.appendChild(li);});}")
	b.WriteString("async function loadPack(){statusEl.textContent='Loading...';")
	b.WriteString("summaryEl.textContent='-';jsonEl.textContent='Loading...';")
	b.WriteString("try{const res=await fetch('/api/v2/context-packs/'+encodeURIComponent(packId));")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const pack=data&&data.context_pack?data.context_pack:data;")
	b.WriteString("statusEl.textContent='Loaded';")
	b.WriteString("summaryEl.textContent=pack&&pack.summary?pack.summary:'No summary';")
	b.WriteString("const contents=pack&&pack.contents?pack.contents:{};")
	b.WriteString("let files=Array.isArray(contents.files)?contents.files:[];")
	b.WriteString("if(!files.length&&contents.repo_files&&Array.isArray(contents.repo_files.files)){files=contents.repo_files.files;}")
	b.WriteString("setFiles(files);")
	b.WriteString("jsonEl.textContent=JSON.stringify(pack,null,2);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load pack';")
	b.WriteString("summaryEl.textContent='-';filesEl.innerHTML='';")
	b.WriteString("const li=document.createElement('li');li.className='muted';")
	b.WriteString("li.textContent='Error loading files';filesEl.appendChild(li);")
	b.WriteString("jsonEl.textContent=String(err&&err.message?err.message:err);}")
	b.WriteString("}")
	b.WriteString("loadPack();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	return b.String()
}

func uiPlaceholderHandler(title string, subtitle string, details []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, uiPlaceholderPage(title, subtitle, details))
	}
}

func taskGraphsPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/graphs/tasks" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Task DAG"))
	b.WriteString("<style>")
	b.WriteString("*{box-sizing:border-box;}")
	b.WriteString(".legend{display:flex;gap:12px;flex-wrap:wrap;font-size:11px;margin-bottom:12px;}")
	b.WriteString(".legend-item{display:flex;align-items:center;gap:4px;}")
	b.WriteString(".legend-dot{width:10px;height:10px;border-radius:50%;}")
	b.WriteString(".graph-canvas{position:relative;min-height:400px;background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;overflow:auto;}")
	b.WriteString("svg.dag{width:100%;height:100%;position:absolute;top:0;left:0;pointer-events:none;}")
	b.WriteString("svg.dag line{stroke-width:1.5;}")
	b.WriteString("svg.dag polygon{}")
	b.WriteString(".node{position:absolute;background:#252526;border:2px solid #3c3c3c;border-radius:8px;padding:8px 10px;font-size:11px;cursor:pointer;min-width:120px;z-index:1;transition:box-shadow 0.15s;}")
	b.WriteString(".node:hover{box-shadow:0 0 0 2px #0078d4;z-index:10;}")
	b.WriteString(".node .node-id{font-weight:700;color:#0078d4;margin-bottom:2px;}")
	b.WriteString(".node .node-title{color:#ccc;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:160px;}")
	b.WriteString(".node .node-status{font-size:10px;margin-top:3px;font-weight:600;text-transform:uppercase;}")
	b.WriteString(".node.s-PENDING{border-color:#858585;} .node.s-PENDING .node-status{color:#858585;}")
	b.WriteString(".node.s-READY{border-color:#3794ff;} .node.s-READY .node-status{color:#3794ff;}")
	b.WriteString(".node.s-IN_PROGRESS{border-color:#0078d4;} .node.s-IN_PROGRESS .node-status{color:#0078d4;}")
	b.WriteString(".node.s-BLOCKED{border-color:#f14c4c;} .node.s-BLOCKED .node-status{color:#f14c4c;}")
	b.WriteString(".node.s-DONE,.node.s-SUCCEEDED{border-color:#89d185;} .node.s-DONE .node-status,.node.s-SUCCEEDED .node-status{color:#89d185;}")
	b.WriteString(".node.s-CANCELED{border-color:#858585;} .node.s-CANCELED .node-status{color:#858585;}")
	b.WriteString(".node.s-FAILED{border-color:#f14c4c;} .node.s-FAILED .node-status{color:#f14c4c;}")
	b.WriteString(".node.s-REVIEW{border-color:#cca700;} .node.s-REVIEW .node-status{color:#cca700;}")
	b.WriteString(".node.critical-path{box-shadow:0 0 0 3px #cca700;} .node.cycle-node{box-shadow:0 0 0 3px #f14c4c;} .node.parallel-node{box-shadow:0 0 0 3px #3794ff;}")
	b.WriteString(".analysis-bar{background:#252526;border:1px solid #3c3c3c;border-radius:8px;padding:10px 14px;margin-bottom:12px;font-size:12px;color:#858585;display:flex;gap:18px;flex-wrap:wrap;}")
	b.WriteString(".analysis-bar .ab-item{display:flex;align-items:center;gap:5px;} .analysis-bar .ab-dot{width:8px;height:8px;border-radius:50%;}")
	b.WriteString(".info-panel{display:none;position:fixed;top:80px;right:32px;width:280px;background:#252526;border:1px solid #3c3c3c;border-radius:10px;padding:14px;font-size:12px;z-index:100;box-shadow:0 8px 32px rgba(0,0,0,0.5);}")
	b.WriteString(".info-panel.open{display:block;}")
	b.WriteString(".info-panel h3{margin:0 0 8px;font-size:14px;color:#ccc;}")
	b.WriteString(".info-panel .close{position:absolute;top:8px;right:10px;cursor:pointer;color:#858585;font-size:16px;}")
	b.WriteString(".info-panel .row{display:flex;justify-content:space-between;padding:3px 0;border-bottom:1px solid #3c3c3c;}")
	b.WriteString(".info-panel .label{color:#858585;} .info-panel .value{color:#ccc;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Task DAG</h1>")
	b.WriteString("<p>Interactive dependency graph from <span class=\"muted\" id=\"graph-query\">/api/v2/projects/proj_default/graphs/tasks</span></p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<label class=\"field\">Project<input class=\"input\" id=\"graph-project\" value=\"proj_default\"></label>")
	b.WriteString("<label class=\"field\">Status<select class=\"select\" id=\"graph-status-filter\">")
	b.WriteString("<option value=\"\">All</option>")
	b.WriteString("<option value=\"PENDING\">PENDING</option><option value=\"READY\">READY</option>")
	b.WriteString("<option value=\"IN_PROGRESS\">IN_PROGRESS</option><option value=\"BLOCKED\">BLOCKED</option>")
	b.WriteString("<option value=\"DONE\">DONE</option><option value=\"CANCELED\">CANCELED</option>")
	b.WriteString("<option value=\"FAILED\">FAILED</option></select></label>")
	b.WriteString("<span id=\"graph-status\">Loading...</span>")
	b.WriteString("<button class=\"btn\" id=\"graph-refresh\" type=\"button\">Refresh</button></div>")
	b.WriteString("<div class=\"legend\">")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#858585\"></span>Pending</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#3794ff\"></span>Ready</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#0078d4\"></span>In Progress</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#f14c4c\"></span>Blocked</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#89d185\"></span>Done</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#f14c4c\"></span>Failed</span>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"graph-canvas\" id=\"graph-canvas\"><svg class=\"dag\" id=\"dag-svg\"></svg></div>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"info-panel\" id=\"info-panel\"><span class=\"close\" id=\"info-close\">&times;</span>")
	b.WriteString("<h3 id=\"info-title\"></h3><div id=\"info-body\"></div></div>")
	b.WriteString("<script>")
	b.WriteString("const canvasEl=document.getElementById('graph-canvas');")
	b.WriteString("const svgEl=document.getElementById('dag-svg');")
	b.WriteString("const statusEl=document.getElementById('graph-status');")
	b.WriteString("const queryEl=document.getElementById('graph-query');")
	b.WriteString("const refreshBtn=document.getElementById('graph-refresh');")
	b.WriteString("const projectEl=document.getElementById('graph-project');")
	b.WriteString("const statusFilterEl=document.getElementById('graph-status-filter');")
	b.WriteString("const infoPanel=document.getElementById('info-panel');")
	b.WriteString("const infoTitle=document.getElementById('info-title');")
	b.WriteString("const infoBody=document.getElementById('info-body');")
	b.WriteString("const infoClose=document.getElementById('info-close');")
	b.WriteString("infoClose.addEventListener('click',()=>infoPanel.classList.remove('open'));")
	b.WriteString("const statusColors={PENDING:'#858585',READY:'#3794ff',IN_PROGRESS:'#0078d4',BLOCKED:'#f14c4c',DONE:'#89d185',SUCCEEDED:'#89d185',CANCELED:'#858585',FAILED:'#f14c4c',REVIEW:'#cca700'};")
	// Layout algorithm: Kahn's topological sort to assign layers, then position nodes
	b.WriteString("function layoutDAG(nodes,edges){")
	b.WriteString("const nodeMap=new Map();nodes.forEach(n=>nodeMap.set(String(n.id),n));")
	b.WriteString("const inDeg=new Map();const adj=new Map();")
	b.WriteString("nodes.forEach(n=>{const id=String(n.id);inDeg.set(id,0);adj.set(id,[]);});")
	b.WriteString("edges.forEach(e=>{const from=String(e.from);const to=String(e.to);")
	b.WriteString("if(adj.has(from)&&inDeg.has(to)){adj.get(from).push(to);inDeg.set(to,(inDeg.get(to)||0)+1);}});")
	// Kahn's algorithm for topological layers
	b.WriteString("const layers=[];const queue=[];")
	b.WriteString("inDeg.forEach((d,id)=>{if(d===0)queue.push(id);});")
	b.WriteString("const visited=new Set();")
	b.WriteString("while(queue.length>0){const layer=[...queue];queue.length=0;layers.push(layer);")
	b.WriteString("layer.forEach(id=>{visited.add(id);(adj.get(id)||[]).forEach(to=>{")
	b.WriteString("inDeg.set(to,(inDeg.get(to)||0)-1);if(inDeg.get(to)===0&&!visited.has(to))queue.push(to);});});}")
	// Handle nodes not in any layer (cycles or disconnected)
	b.WriteString("nodes.forEach(n=>{if(!visited.has(String(n.id))){")
	b.WriteString("if(layers.length===0)layers.push([]);layers[layers.length-1].push(String(n.id));}});")
	// Assign positions
	b.WriteString("const positions=new Map();const nodeW=140;const nodeH=70;const padX=40;const padY=30;")
	b.WriteString("const gapX=nodeW+padX;const gapY=nodeH+padY;")
	b.WriteString("let maxX=0;")
	b.WriteString("layers.forEach((layer,li)=>{layer.forEach((id,ni)=>{")
	b.WriteString("const x=ni*gapX+padX;const y=li*gapY+padY;")
	b.WriteString("positions.set(id,{x,y});if(x+nodeW>maxX)maxX=x+nodeW;});});")
	b.WriteString("const totalH=layers.length*gapY+padY;const totalW=maxX+padX;")
	b.WriteString("return {positions,totalW,totalH,nodeW,nodeH};}")

	// Critical path analysis: longest path in DAG using dynamic programming
	b.WriteString("function findCriticalPath(nodes,edges){")
	b.WriteString("const adj=new Map();const ids=new Set();")
	b.WriteString("nodes.forEach(n=>{const id=String(n.id);ids.add(id);adj.set(id,[]);});")
	b.WriteString("edges.forEach(e=>{const f=String(e.from);const t=String(e.to);if(adj.has(f))adj.get(f).push(t);});")
	b.WriteString("const dist=new Map();const prev=new Map();ids.forEach(id=>{dist.set(id,0);prev.set(id,null);});")
	b.WriteString("const visited=new Set();const order=[];")
	b.WriteString("function dfs(id){if(visited.has(id))return;visited.add(id);(adj.get(id)||[]).forEach(t=>dfs(t));order.push(id);}")
	b.WriteString("ids.forEach(id=>dfs(id));order.reverse();")
	b.WriteString("order.forEach(id=>{(adj.get(id)||[]).forEach(t=>{if(dist.get(id)+1>dist.get(t)){dist.set(t,dist.get(id)+1);prev.set(t,id);}});});")
	b.WriteString("let maxDist=0;let endNode=null;dist.forEach((d,id)=>{if(d>maxDist){maxDist=d;endNode=id;}});")
	b.WriteString("const path=[];let cur=endNode;while(cur!==null){path.unshift(cur);cur=prev.get(cur);}return {path,length:maxDist};}")

	// Cycle detection using DFS coloring
	b.WriteString("function detectCycles(nodes,edges){")
	b.WriteString("const adj=new Map();nodes.forEach(n=>adj.set(String(n.id),[]));")
	b.WriteString("edges.forEach(e=>{const f=String(e.from);if(adj.has(f))adj.get(f).push(String(e.to));});")
	b.WriteString("const white=new Set(nodes.map(n=>String(n.id)));const gray=new Set();const black=new Set();")
	b.WriteString("const cycles=[];")
	b.WriteString("function dfs(id,stack){white.delete(id);gray.add(id);stack.push(id);")
	b.WriteString("for(const t of (adj.get(id)||[])){if(gray.has(t)){const ci=stack.indexOf(t);if(ci>=0)cycles.push(stack.slice(ci));}")
	b.WriteString("else if(white.has(t))dfs(t,[...stack]);}gray.delete(id);black.add(id);}")
	b.WriteString("while(white.size>0){const id=white.values().next().value;dfs(id,[]);}return cycles;}")

	// Parallel execution: nodes at same topological layer with no dependencies between them
	b.WriteString("function findParallelGroups(nodes,edges){")
	b.WriteString("const inDeg=new Map();const adj=new Map();")
	b.WriteString("nodes.forEach(n=>{const id=String(n.id);inDeg.set(id,0);adj.set(id,[]);});")
	b.WriteString("edges.forEach(e=>{const f=String(e.from);const t=String(e.to);if(adj.has(f)&&inDeg.has(t)){adj.get(f).push(t);inDeg.set(t,(inDeg.get(t)||0)+1);}});")
	b.WriteString("const layers=[];const queue=[];const visited=new Set();")
	b.WriteString("inDeg.forEach((d,id)=>{if(d===0)queue.push(id);});")
	b.WriteString("while(queue.length>0){const layer=[...queue];queue.length=0;layers.push(layer);")
	b.WriteString("layer.forEach(id=>{visited.add(id);(adj.get(id)||[]).forEach(t=>{inDeg.set(t,(inDeg.get(t)||0)-1);if(inDeg.get(t)===0&&!visited.has(t))queue.push(t);});});}")
	b.WriteString("return layers.filter(l=>l.length>1);}")

	// Render function
	b.WriteString("function renderGraph(data){")
	b.WriteString("const nodes=data.nodes||[];const edges=data.edges||[];")
	b.WriteString("const statusFilter=statusFilterEl.value;")
	b.WriteString("const filteredNodes=statusFilter?nodes.filter(n=>(n.status||'').toUpperCase()===statusFilter):nodes;")
	b.WriteString("const filteredIds=new Set(filteredNodes.map(n=>String(n.id)));")
	b.WriteString("const filteredEdges=edges.filter(e=>filteredIds.has(String(e.from))&&filteredIds.has(String(e.to)));")
	// Run analysis
	b.WriteString("const cp=findCriticalPath(filteredNodes,filteredEdges);")
	b.WriteString("const cycles=detectCycles(filteredNodes,filteredEdges);")
	b.WriteString("const parallel=findParallelGroups(filteredNodes,filteredEdges);")
	b.WriteString("const cpSet=new Set(cp.path);")
	b.WriteString("const cycleNodes=new Set();cycles.forEach(c=>c.forEach(id=>cycleNodes.add(id)));")
	b.WriteString("const parallelNodes=new Set();parallel.forEach(g=>g.forEach(id=>parallelNodes.add(id)));")
	// Build analysis summary
	b.WriteString("let analysis='';")
	b.WriteString("if(cp.path.length>1)analysis+=' | Critical path: '+cp.path.length+' nodes (len '+cp.length+')';")
	b.WriteString("if(cycles.length>0)analysis+=' | \\u26A0 '+cycles.length+' cycle(s) detected!';")
	b.WriteString("if(parallel.length>0)analysis+=' | '+parallel.length+' parallel group(s)';")
	b.WriteString("statusEl.textContent=filteredNodes.length+' nodes, '+filteredEdges.length+' edges'+analysis;")
	b.WriteString("canvasEl.querySelectorAll('.node').forEach(el=>el.remove());")
	b.WriteString("svgEl.innerHTML='';")
	b.WriteString("if(filteredNodes.length===0){canvasEl.style.minHeight='100px';return;}")
	b.WriteString("const {positions,totalW,totalH,nodeW,nodeH}=layoutDAG(filteredNodes,filteredEdges);")
	b.WriteString("canvasEl.style.width=totalW+'px';canvasEl.style.height=totalH+'px';canvasEl.style.minHeight=totalH+'px';")
	b.WriteString("svgEl.setAttribute('width',totalW);svgEl.setAttribute('height',totalH);svgEl.setAttribute('viewBox','0 0 '+totalW+' '+totalH);")
	// Render analysis bar above the canvas
	b.WriteString("let oldBar=document.getElementById('analysis-bar');if(oldBar)oldBar.remove();")
	b.WriteString("if(cp.path.length>1||cycles.length>0||parallel.length>0){")
	b.WriteString("const bar=document.createElement('div');bar.id='analysis-bar';bar.className='analysis-bar';")
	b.WriteString("let barHtml='';")
	b.WriteString("if(cp.path.length>1)barHtml+='<span class=\"ab-item\"><span class=\"ab-dot\" style=\"background:#f59e0b\"></span>Critical Path: '+cp.path.length+' nodes</span>';")
	b.WriteString("if(cycles.length>0)barHtml+='<span class=\"ab-item\"><span class=\"ab-dot\" style=\"background:#ef4444\"></span>\\u26A0 '+cycles.length+' cycle(s)</span>';")
	b.WriteString("if(parallel.length>0)barHtml+='<span class=\"ab-item\"><span class=\"ab-dot\" style=\"background:#38bdf8\"></span>'+parallel.length+' parallel group(s)</span>';")
	b.WriteString("bar.innerHTML=barHtml;canvasEl.parentNode.insertBefore(bar,canvasEl);}")
	// Draw edges as SVG lines with arrowheads
	b.WriteString("const ns='http://www.w3.org/2000/svg';")
	b.WriteString("const defs=document.createElementNS(ns,'defs');")
	b.WriteString("const depMarker=document.createElementNS(ns,'marker');")
	b.WriteString("depMarker.setAttribute('id','arrow-dep');depMarker.setAttribute('viewBox','0 0 10 10');")
	b.WriteString("depMarker.setAttribute('refX','10');depMarker.setAttribute('refY','5');")
	b.WriteString("depMarker.setAttribute('markerWidth','6');depMarker.setAttribute('markerHeight','6');")
	b.WriteString("depMarker.setAttribute('orient','auto-start-reverse');")
	b.WriteString("const poly=document.createElementNS(ns,'path');poly.setAttribute('d','M 0 0 L 10 5 L 0 10 z');poly.setAttribute('fill','#0078d4');")
	b.WriteString("depMarker.appendChild(poly);defs.appendChild(depMarker);")
	b.WriteString("const pcMarker=document.createElementNS(ns,'marker');")
	b.WriteString("pcMarker.setAttribute('id','arrow-pc');pcMarker.setAttribute('viewBox','0 0 10 10');")
	b.WriteString("pcMarker.setAttribute('refX','10');pcMarker.setAttribute('refY','5');")
	b.WriteString("pcMarker.setAttribute('markerWidth','6');pcMarker.setAttribute('markerHeight','6');")
	b.WriteString("pcMarker.setAttribute('orient','auto-start-reverse');")
	b.WriteString("const poly2=document.createElementNS(ns,'path');poly2.setAttribute('d','M 0 0 L 10 5 L 0 10 z');poly2.setAttribute('fill','#3c3c3c');")
	b.WriteString("pcMarker.appendChild(poly2);defs.appendChild(pcMarker);")
	b.WriteString("svgEl.appendChild(defs);")
	b.WriteString("filteredEdges.forEach(e=>{const from=positions.get(String(e.from));const to=positions.get(String(e.to));if(!from||!to)return;")
	b.WriteString("const line=document.createElementNS(ns,'line');")
	b.WriteString("line.setAttribute('x1',from.x+nodeW/2);line.setAttribute('y1',from.y+nodeH);")
	b.WriteString("line.setAttribute('x2',to.x+nodeW/2);line.setAttribute('y2',to.y);")
	b.WriteString("const isDep=e.type==='depends_on';")
	b.WriteString("const isCpEdge=cpSet.has(String(e.from))&&cpSet.has(String(e.to));")
	b.WriteString("line.setAttribute('stroke',isCpEdge?'#cca700':isDep?'#0078d4':'#3c3c3c');")
	b.WriteString("line.setAttribute('stroke-width',isCpEdge?'3':'1.5');")
	b.WriteString("line.setAttribute('stroke-dasharray',isDep?'':'5,3');")
	b.WriteString("line.setAttribute('marker-end',isDep?'url(#arrow-dep)':'url(#arrow-pc)');")
	b.WriteString("svgEl.appendChild(line);});")
	// Draw nodes as positioned divs
	b.WriteString("filteredNodes.forEach(n=>{const pos=positions.get(String(n.id));if(!pos)return;")
	b.WriteString("const div=document.createElement('div');")
	b.WriteString("const st=(n.status||'PENDING').toUpperCase();")
	b.WriteString("const nid=String(n.id);")
	b.WriteString("let extraCls='';if(cpSet.has(nid))extraCls+=' critical-path';if(cycleNodes.has(nid))extraCls+=' cycle-node';if(parallelNodes.has(nid))extraCls+=' parallel-node';")
	b.WriteString("div.className='node s-'+st+extraCls;")
	b.WriteString("div.style.left=pos.x+'px';div.style.top=pos.y+'px';div.style.width=nodeW+'px';")
	b.WriteString("let badges='';if(cpSet.has(nid))badges+='<span style=\"color:#f59e0b;font-size:9px;\">\\u2B50 CP</span> ';")
	b.WriteString("if(cycleNodes.has(nid))badges+='<span style=\"color:#ef4444;font-size:9px;\">\\u26A0 CYCLE</span> ';")
	b.WriteString("if(parallelNodes.has(nid))badges+='<span style=\"color:#38bdf8;font-size:9px;\">\\u2693 PAR</span> ';")
	b.WriteString("div.innerHTML='<div class=\"node-id\">#'+escapeHtml(n.id)+'</div><div class=\"node-title\">'+escapeHtml(n.title||'')+'</div><div class=\"node-status\">'+badges+escapeHtml(st)+'</div>';")
	b.WriteString("div.addEventListener('click',()=>{infoTitle.textContent='Task #'+n.id;")
	b.WriteString("infoBody.innerHTML='';const fields={Status:st,Type:n.type||'-',Title:n.title||'-',Parent:n.parent_task_id||'-','Critical Path':cpSet.has(nid)?'Yes':'No','In Cycle':cycleNodes.has(nid)?'Yes':'No','Can Parallelize':parallelNodes.has(nid)?'Yes':'No'};")
	b.WriteString("Object.entries(fields).forEach(([k,v])=>{const row=document.createElement('div');row.className='row';")
	b.WriteString("row.innerHTML='<span class=\"label\">'+k+'</span><span class=\"value\">'+escapeHtml(v)+'</span>';infoBody.appendChild(row);});")
	b.WriteString("infoPanel.classList.add('open');});")
	b.WriteString("canvasEl.appendChild(div);});}")

	// Load data
	b.WriteString("let graphData=null;")
	b.WriteString("async function loadGraph(){statusEl.textContent='Loading...';try{")
	b.WriteString("const pid=String(projectEl.value||'proj_default').trim();")
	b.WriteString("const url='/api/v2/projects/'+encodeURIComponent(pid)+'/graphs/tasks';queryEl.textContent=url;")
	b.WriteString("const res=await fetch(url);if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("graphData=await res.json();renderGraph(graphData);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load graph';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadGraph);")
	b.WriteString("projectEl.addEventListener('change',loadGraph);")
	b.WriteString("statusFilterEl.addEventListener('change',()=>{if(graphData)renderGraph(graphData);});")
	b.WriteString("loadGraph();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func memoryPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/memory" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Memory"))
	b.WriteString("<style>")
	b.WriteString(".form{display:grid;gap:10px;margin:12px 0 16px;}")
	b.WriteString(".row{display:flex;gap:12px;flex-wrap:wrap;}")
	b.WriteString(".form-actions{display:flex;align-items:center;gap:12px;flex-wrap:wrap;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Memory</h1>")
	b.WriteString("<div class=\"row\" style=\"margin-bottom:10px;\"><label class=\"field\">Project<input class=\"input\" id=\"memory-project\" value=\"proj_default\" style=\"width:200px;\"></label></div>")
	b.WriteString("<p>Stored key-value pairs for the selected project</p>")
	b.WriteString("<div class=\"meta\"><span id=\"memory-status\">Loading...</span>")
	b.WriteString("<button class=\"btn\" id=\"memory-refresh\" type=\"button\">Refresh</button></div>")
	b.WriteString("<form class=\"form\" id=\"memory-form\">")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<label class=\"field\">Scope<input class=\"input\" id=\"memory-scope\" name=\"scope\" placeholder=\"GLOBAL\" required></label>")
	b.WriteString("<label class=\"field\">Key<input class=\"input\" id=\"memory-key\" name=\"key\" placeholder=\"release_notes\" required></label>")
	b.WriteString("</div>")
	b.WriteString("<label class=\"field\">Value JSON<textarea class=\"textarea\" id=\"memory-value\" name=\"value\" placeholder='{\"note\":\"...\"}' required></textarea></label>")
	b.WriteString("<div class=\"form-actions\"><button class=\"btn\" id=\"memory-submit\" type=\"submit\">Save</button><span class=\"muted\" id=\"memory-form-status\"></span></div>")
	b.WriteString("</form>")
	b.WriteString("<table><thead><tr><th>Scope</th><th>Key</th><th>Updated</th></tr></thead>")
	b.WriteString("<tbody id=\"memory-body\"><tr><td colspan=\"3\">Loading...</td></tr></tbody></table>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const bodyEl=document.getElementById('memory-body');")
	b.WriteString("const statusEl=document.getElementById('memory-status');")
	b.WriteString("const refreshBtn=document.getElementById('memory-refresh');")
	b.WriteString("const formEl=document.getElementById('memory-form');")
	b.WriteString("const scopeEl=document.getElementById('memory-scope');")
	b.WriteString("const keyEl=document.getElementById('memory-key');")
	b.WriteString("const valueEl=document.getElementById('memory-value');")
	b.WriteString("const formStatusEl=document.getElementById('memory-form-status');")
	b.WriteString("const projectEl=document.getElementById('memory-project');")
	b.WriteString("function getProjectID(){return String(projectEl.value||'proj_default').trim();}")
	b.WriteString("async function loadMemory(){const pid=getProjectID();statusEl.textContent='Loading...';")
	b.WriteString("bodyEl.innerHTML='<tr><td colspan=\"3\">Loading...</td></tr>';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/memory?limit=50');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const items=Array.isArray(data.items)?data.items:[];")
	b.WriteString("statusEl.textContent=items.length+' items';")
	b.WriteString("if(items.length===0){bodyEl.innerHTML='<tr><td colspan=\"3\">No memory entries yet. Use the API or form above to store key-value data.</td></tr>';return;}")
	b.WriteString("bodyEl.innerHTML='';items.forEach((item)=>{")
	b.WriteString("const tr=document.createElement('tr');")
	b.WriteString("const scope=escapeHtml(item.scope);const key=escapeHtml(item.key);")
	b.WriteString("const updated=escapeHtml(item.updated_at||'');")
	b.WriteString("tr.innerHTML='<td>'+scope+'</td><td>'+key+'</td><td>'+updated+'</td>';bodyEl.appendChild(tr);});")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load memory';")
	b.WriteString("bodyEl.innerHTML='<tr><td colspan=\"3\">Error loading memory</td></tr>';}}")
	b.WriteString("async function submitMemory(event){event.preventDefault();")
	b.WriteString("const scope=String(scopeEl.value||'').trim();")
	b.WriteString("const key=String(keyEl.value||'').trim();")
	b.WriteString("const rawValue=String(valueEl.value||'').trim();")
	b.WriteString("if(!scope||!key||!rawValue){formStatusEl.textContent='Scope, key, and value are required.';return;}")
	b.WriteString("let valueJson=null;try{valueJson=JSON.parse(rawValue);}catch(err){formStatusEl.textContent='Value must be valid JSON.';return;}")
	b.WriteString("formStatusEl.textContent='Saving...';")
	b.WriteString("try{const pid=getProjectID();const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/memory',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({scope:scope,key:key,value:valueJson})});")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("formStatusEl.textContent='Saved.';await loadMemory();}catch(err){formStatusEl.textContent='Save failed.';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadMemory);")
	b.WriteString("formEl.addEventListener('submit',submitMemory);")
	b.WriteString("loadMemory();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func agentsPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/agents" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Agents"))
	b.WriteString(`<style>
.agents-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 16px; margin-top: 16px; }
.agent-card { background: var(--vscode-input-bg, #2a2d2e); border: 1px solid var(--vscode-border, #3c3c3c); border-radius: 8px; padding: 16px; transition: border-color 0.15s; }
.agent-card:hover { border-color: #0078d4; }
.agent-card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.agent-card-name { font-size: 14px; font-weight: 600; }
.agent-card-id { font-size: 11px; color: #858585; font-family: monospace; }
.agent-status-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 6px; }
.agent-status-dot.online { background: #4caf50; box-shadow: 0 0 4px #4caf50; }
.agent-status-dot.offline { background: #666; }
.agent-status-dot.stale { background: #ff9800; }
.agent-meta { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; font-size: 12px; }
.agent-meta-label { color: #858585; font-size: 11px; }
.agent-meta-value { color: #ccc; }
.agent-caps { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 8px; }
.agent-cap-tag { background: #0e3a5e; color: #56b6f7; font-size: 10px; padding: 2px 6px; border-radius: 10px; }
.agent-actions { margin-top: 12px; display: flex; gap: 8px; }
.agent-actions button { background: var(--vscode-input-bg, #2a2d2e); border: 1px solid var(--vscode-border, #3c3c3c); color: #ccc; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 11px; }
.agent-actions button:hover { background: #3c3c3c; }
.agent-actions button.danger { color: #f44; }
.agent-actions button.danger:hover { background: #4c1c1c; }
.heartbeat-bar { height: 4px; border-radius: 2px; margin-top: 4px; background: #333; overflow: hidden; }
.heartbeat-fill { height: 100%; border-radius: 2px; transition: width 0.3s; }
.heartbeat-fill.fresh { background: #4caf50; }
.heartbeat-fill.aging { background: #ff9800; }
.heartbeat-fill.stale { background: #f44; }
</style>`)
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Agents</h1>")
	b.WriteString("<p class=\"muted\">Registered agents across all projects</p>")
	b.WriteString("<div class=\"meta\"><span id=\"agents-status\">Loading...</span> ")
	b.WriteString("<label class=\"field\">Status<select class=\"select\" id=\"agents-status-filter\"><option value=\"\">All</option><option value=\"active\">Active</option><option value=\"stale\">Stale</option></select></label> ")
	b.WriteString("<button class=\"btn\" id=\"agents-refresh\" type=\"button\">Refresh</button></div>")
	b.WriteString("<div class=\"agents-grid\" id=\"agents-grid\"><div style=\"color:#858585;\">Loading...</div></div>")
	b.WriteString("</div>")
	b.WriteString(`<div id="agent-toast" style="position:fixed;bottom:20px;right:20px;z-index:999;display:none;padding:10px 16px;border-radius:6px;font-size:12px;color:#ccc;background:#1e1e1e;border:1px solid #3c3c3c;box-shadow:0 4px 12px rgba(0,0,0,0.4);"></div>`)
	b.WriteString(`<div id="agent-confirm-overlay" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,0.5);z-index:998;justify-content:center;align-items:center;">`)
	b.WriteString(`<div style="background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:20px;max-width:400px;text-align:center;">`)
	b.WriteString(`<p id="agent-confirm-msg" style="margin:0 0 16px;color:#ccc;font-size:13px;"></p>`)
	b.WriteString(`<div style="display:flex;gap:8px;justify-content:center;">`)
	b.WriteString(`<button id="agent-confirm-cancel" style="padding:6px 16px;background:#333;border:1px solid #555;color:#ccc;border-radius:4px;cursor:pointer;">Cancel</button>`)
	b.WriteString(`<button id="agent-confirm-ok" style="padding:6px 16px;background:#c53030;border:none;color:#fff;border-radius:4px;cursor:pointer;">Delete</button>`)
	b.WriteString(`</div></div></div>`)
	b.WriteString("<script>")
	b.WriteString(`const gridEl=document.getElementById('agents-grid');const statusEl=document.getElementById('agents-status');const refreshBtn=document.getElementById('agents-refresh');const statusFilterEl=document.getElementById('agents-status-filter');`)
	b.WriteString(`function showAgentToast(msg,isError){const t=document.getElementById('agent-toast');t.textContent=msg;t.style.display='block';t.style.borderColor=isError?'#f44':'#4caf50';setTimeout(()=>{t.style.display='none';},3000);}`)
	b.WriteString(`let _confirmResolve=null;function showAgentConfirm(msg){return new Promise(resolve=>{_confirmResolve=resolve;const overlay=document.getElementById('agent-confirm-overlay');document.getElementById('agent-confirm-msg').textContent=msg;overlay.style.display='flex';});}`)
	b.WriteString(`document.getElementById('agent-confirm-cancel').addEventListener('click',()=>{document.getElementById('agent-confirm-overlay').style.display='none';if(_confirmResolve)_confirmResolve(false);});`)
	b.WriteString(`document.getElementById('agent-confirm-ok').addEventListener('click',()=>{document.getElementById('agent-confirm-overlay').style.display='none';if(_confirmResolve)_confirmResolve(true);});`)
	b.WriteString(`function heartbeatAge(lastSeen){if(!lastSeen)return{pct:0,cls:'stale',label:'never'};const ms=Date.now()-new Date(lastSeen).getTime();const secs=ms/1000;if(secs<60)return{pct:100,cls:'fresh',label:Math.round(secs)+'s ago'};if(secs<300)return{pct:Math.max(40,100-secs/3),cls:'aging',label:Math.round(secs/60)+'m ago'};return{pct:10,cls:'stale',label:secs>86400?Math.round(secs/86400)+'d ago':Math.round(secs/3600)+'h ago'};}`)
	b.WriteString(`async function deleteAgent(id){const ok=await showAgentConfirm('Delete agent '+id+'?');if(!ok)return;try{const r=await fetch('/api/v2/agents/'+encodeURIComponent(id),{method:'DELETE'});if(r.ok){showAgentToast('Agent deleted');loadAgents();}else{showAgentToast('Failed to delete agent',true);}}catch(e){showAgentToast('Error: '+e.message,true);}}`)
	b.WriteString(`async function loadAgents(){statusEl.textContent='Loading...';gridEl.innerHTML='<div style="color:#858585;">Loading...</div>';try{`)
	b.WriteString(`const params=new URLSearchParams();const status=String(statusFilterEl.value||'').trim();if(status)params.set('status',status);`)
	b.WriteString(`const res=await fetch('/api/v2/agents?'+params.toString());if(!res.ok)throw new Error();`)
	b.WriteString(`const data=await res.json();const agents=Array.isArray(data.agents)?data.agents:[];`)
	b.WriteString(`statusEl.textContent=agents.length+' agent'+(agents.length!==1?'s':'');`)
	b.WriteString(`if(agents.length===0){gridEl.innerHTML='<div style="grid-column:1/-1;text-align:center;padding:40px;color:#858585;">No agents connected yet.<br><span style="font-size:11px;color:#666;">Start the built-in worker or connect your own agent to see it here.</span></div>';return;}`)
	b.WriteString(`gridEl.innerHTML='';agents.forEach((agent)=>{`)
	b.WriteString(`const hb=heartbeatAge(agent.last_seen);`)
	b.WriteString(`const statusLower=(agent.status||'unknown').toLowerCase();`)
	b.WriteString(`const dotCls=statusLower==='online'?'online':statusLower==='active'?'online':statusLower==='stale'?'stale':'offline';`)
	b.WriteString(`const caps=Array.isArray(agent.capabilities)?agent.capabilities:[];`)
	b.WriteString(`const meta=agent.metadata||{};`)
	b.WriteString(`let capsHtml='';if(caps.length>0){capsHtml='<div class="agent-caps">'+caps.map(c=>'<span class="agent-cap-tag">'+escapeHtml(c)+'</span>').join('')+'</div>';}`)
	b.WriteString(`const card=document.createElement('div');card.className='agent-card';`)
	b.WriteString(`card.innerHTML='<div class="agent-card-header"><div><span class="agent-status-dot '+dotCls+'"></span><span class="agent-card-name">'+escapeHtml(agent.name||agent.id)+'</span></div><span style="font-size:11px;padding:2px 8px;border-radius:10px;background:'+(dotCls==='online'?'#1a3a1a':'#333')+';color:'+(dotCls==='online'?'#4caf50':'#888')+';">'+escapeHtml(agent.status||'unknown')+'</span></div>'`)
	b.WriteString(`+'<div class="agent-card-id">'+escapeHtml(agent.id)+'</div>'`)
	b.WriteString(`+'<div class="agent-meta" style="margin-top:12px;">'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Heartbeat</div><div class="agent-meta-value">'+hb.label+'</div><div class="heartbeat-bar"><div class="heartbeat-fill '+hb.cls+'" style="width:'+hb.pct+'%"></div></div></div>'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Created</div><div class="agent-meta-value">'+formatAgo(agent.created_at||agent.registered_at)+'</div></div>'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Current Task</div><div class="agent-meta-value">'+(meta.current_task?'<a href="/board" style="color:#56b6f7;">#'+escapeHtml(meta.current_task)+'</a>':'<span style="color:#666;">idle</span>')+'</div></div>'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Current Run</div><div class="agent-meta-value">'+(meta.current_run?'<a href="/runs" style="color:#56b6f7;">'+escapeHtml(String(meta.current_run).substring(0,12))+'</a>':'<span style="color:#666;">—</span>')+'</div></div>'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Tasks Done</div><div class="agent-meta-value" style="color:#73c991;">'+(meta.completed_count||'0')+'</div></div>'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Failed</div><div class="agent-meta-value" style="color:'+(Number(meta.failed_count||0)>0?'#f48771':'#666')+';">'+(meta.failed_count||'0')+'</div></div>'`)
	b.WriteString(`+'</div>'`)
	b.WriteString(`+(dotCls==='stale'&&meta.current_task?'<div style="margin-top:8px;padding:6px 10px;background:#3a1a1a;border:1px solid #5a2a2a;border-radius:4px;font-size:11px;color:#f48771;">⚠ Agent is stale but still has claimed task #'+escapeHtml(meta.current_task)+'. Lease may be expired.</div>':'')`)
	b.WriteString(`+capsHtml`)
	b.WriteString(`+'<div class="agent-actions"><button onclick="deleteAgent(\''+escapeHtml(agent.id)+'\')" class="danger">Delete</button></div>';`)
	b.WriteString(`gridEl.appendChild(card);});`)
	b.WriteString(`}catch(err){statusEl.textContent='Failed to load agents';gridEl.innerHTML='<div style="color:#f44;">Error loading agents</div>';}}`)
	b.WriteString(`refreshBtn.addEventListener('click',loadAgents);statusFilterEl.addEventListener('change',loadAgents);loadAgents();setInterval(loadAgents,30000);`)
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func auditPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/audit" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Audit"))
	b.WriteString("<style>")
	b.WriteString("ul{list-style:none;padding:0;margin:0;}")
	b.WriteString("li{padding:10px 8px;border-bottom:1px solid #3c3c3c;font-size:12px;display:grid;grid-template-columns:2fr 2fr 2fr 2fr 3fr;gap:12px;}")
	b.WriteString("li.header{color:#0078d4;font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.05em;border-bottom:1px solid #505050;}")
	b.WriteString(".payload-cell{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:11px;color:#a0a0a0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:400px;cursor:pointer;}")
	b.WriteString(".payload-cell:hover{white-space:normal;overflow:visible;color:#ccc;}")
	b.WriteString(".audit-filters{display:flex;gap:12px;flex-wrap:wrap;align-items:end;margin-bottom:12px;}")
	b.WriteString(".audit-filters label{display:flex;flex-direction:column;gap:4px;font-size:11px;color:#858585;}")
	b.WriteString(".audit-filters select,.audit-filters input{padding:4px 8px;background:#2a2d2e;border:1px solid #3c3c3c;color:#ccc;border-radius:4px;font-size:12px;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Audit Log</h1>")
	b.WriteString("<p class=\"muted\">Audit events tracking sensitive operations: policy changes, project updates, task deletions, workdir changes.</p>")
	b.WriteString("<div class=\"audit-filters\">")
	b.WriteString("<label>Event Type<select id=\"audit-type\">")
	b.WriteString("<option value=\"\">All audit events</option>")
	b.WriteString("<option value=\"audit.policy.changed\">Policy Changed</option>")
	b.WriteString("<option value=\"audit.project.updated\">Project Updated</option>")
	b.WriteString("<option value=\"audit.task.deleted\">Task Deleted</option>")
	b.WriteString("<option value=\"audit.workdir.changed\">Workdir Changed</option>")
	b.WriteString("</select></label>")
	b.WriteString("<label>Project ID<input class=\"input\" id=\"audit-project\" placeholder=\"all projects\"></label>")
	b.WriteString("<button class=\"btn\" id=\"audit-refresh\" type=\"button\">Refresh</button>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"meta\"><span id=\"audit-status\">Loading...</span></div>")
	b.WriteString("<ul id=\"audit-list\"><li>Loading...</li></ul>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const listEl=document.getElementById('audit-list');")
	b.WriteString("const statusEl=document.getElementById('audit-status');")
	b.WriteString("const refreshBtn=document.getElementById('audit-refresh');")
	b.WriteString("const typeEl=document.getElementById('audit-type');")
	b.WriteString("const projectEl=document.getElementById('audit-project');")
	b.WriteString("function renderHeader(){listEl.innerHTML='';")
	b.WriteString("const header=document.createElement('li');header.className='header';")
	b.WriteString("header.innerHTML='<span>Kind</span><span>Entity</span><span>Project</span><span>Time</span><span>Details</span>';listEl.appendChild(header);}")
	b.WriteString("async function loadAudit(){statusEl.textContent='Loading...';")
	b.WriteString("listEl.innerHTML='<li>Loading...</li>';try{")
	b.WriteString("const params=new URLSearchParams();params.set('limit','100');")
	b.WriteString("const type=String(typeEl.value||'').trim();if(type){params.set('type',type);}")
	b.WriteString("const projectId=String(projectEl.value||'').trim();if(projectId){params.set('project_id',projectId);}")
	b.WriteString("const url='/api/v2/audit?'+params.toString();")
	b.WriteString("const res=await fetch(url);if(!res.ok)throw new Error();")
	b.WriteString("const data=await res.json();const events=Array.isArray(data.events)?data.events:[];")
	b.WriteString("statusEl.textContent=events.length+' of '+data.total+' audit events';renderHeader();")
	b.WriteString("if(events.length===0){const empty=document.createElement('li');")
	b.WriteString("empty.innerHTML='<span class=\"muted\">No audit events</span>';listEl.appendChild(empty);return;}")
	b.WriteString("events.forEach((ev)=>{const item=document.createElement('li');")
	b.WriteString("const kind=escapeHtml(ev.kind);const entity=escapeHtml(ev.entity_type+'/'+ev.entity_id);")
	b.WriteString("const proj=escapeHtml(ev.project_id);const created=escapeHtml(ev.created_at);")
	b.WriteString("const payload=ev.payload?escapeHtml(JSON.stringify(ev.payload)):'';")
	b.WriteString("item.innerHTML='<span>'+kind+'</span><span>'+entity+'</span><span>'+proj+'</span><span>'+created+'</span><span class=\"payload-cell\" title=\"'+payload+'\">'+payload+'</span>';listEl.appendChild(item);});")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load audit events';")
	b.WriteString("listEl.innerHTML='<li>Error loading audit events</li>';}}")
	b.WriteString("function handleEnter(e){if(e.key==='Enter'){loadAudit();}}")
	b.WriteString("refreshBtn.addEventListener('click',loadAudit);")
	b.WriteString("typeEl.addEventListener('change',loadAudit);")
	b.WriteString("projectEl.addEventListener('keydown',handleEnter);")
	b.WriteString("loadAudit();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func repoPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/repo" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Repo Panel"))
	b.WriteString("<style>")
	b.WriteString(".columns{display:grid;grid-template-columns:1fr 1fr;gap:16px;}")
	b.WriteString("@media(max-width:700px){.columns{grid-template-columns:1fr;}}")
	b.WriteString(".tree{list-style:none;padding:0;margin:0;font-size:12px;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;}")
	b.WriteString(".tree li{padding:4px 6px;border-bottom:1px solid #3c3c3c;display:flex;gap:6px;align-items:center;}")
	b.WriteString(".tree li:hover{background:#2a2d2e;}")
	b.WriteString(".tree .icon{width:14px;text-align:center;color:#858585;}")
	b.WriteString(".tree .dir{color:#0078d4;cursor:pointer;}")
	b.WriteString(".tree .file{color:#ccc;}")
	b.WriteString(".change-list{list-style:none;padding:0;margin:0;font-size:12px;}")
	b.WriteString(".change-list li{padding:6px 8px;border-bottom:1px solid #3c3c3c;display:flex;gap:10px;align-items:center;}")
	b.WriteString(".change-list .badge{display:inline-block;padding:1px 6px;border-radius:4px;font-size:10px;font-weight:600;text-transform:uppercase;}")
	b.WriteString(".badge.modified{background:#854d0e;color:#fef08a;}")
	b.WriteString(".badge.added{background:#166534;color:#bbf7d0;}")
	b.WriteString(".badge.deleted{background:#991b1b;color:#fecaca;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Repo Panel</h1>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<label class=\"field\">Project ID<select class=\"input\" id=\"repo-project\"><option value=\"\">Loading projects...</option></select></label>")
	b.WriteString("<button class=\"btn\" id=\"repo-refresh\" type=\"button\">Refresh</button>")
	b.WriteString("<button class=\"btn\" id=\"repo-scan\" type=\"button\" style=\"background:#0078d4;border-color:#0078d4;color:#fff;\">Scan Repo</button>")
	b.WriteString("<span id=\"repo-status\" class=\"muted\">Loading...</span></div>")
	b.WriteString("<div class=\"columns\">")
	b.WriteString("<div><h2>File Tree</h2><ul class=\"tree\" id=\"repo-tree\"><li>Loading...</li></ul></div>")
	b.WriteString("<div><h2>Recent Changes</h2><ul class=\"change-list\" id=\"repo-changes\"><li>Loading...</li></ul></div>")
	b.WriteString("</div></div>")
	b.WriteString("<script>")
	b.WriteString("const treeEl=document.getElementById('repo-tree');")
	b.WriteString("const changesEl=document.getElementById('repo-changes');")
	b.WriteString("const statusEl=document.getElementById('repo-status');")
	b.WriteString("const refreshBtn=document.getElementById('repo-refresh');")
	b.WriteString("const projectEl=document.getElementById('repo-project');")
	b.WriteString("function getProjectID(){return String(projectEl.value||'proj_default').trim();}")
	b.WriteString("async function loadTree(){treeEl.innerHTML='<li>Loading tree...</li>';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(getProjectID())+'/tree');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const entries=Array.isArray(data.entries)?data.entries:[];")
	b.WriteString("if(entries.length===0){treeEl.innerHTML='<li class=\"muted\">No files scanned yet. Scan a directory to populate the repo tree.</li>';return;}")
	b.WriteString("treeEl.innerHTML='';entries.forEach((e)=>{")
	b.WriteString("const li=document.createElement('li');")
	b.WriteString("const isDir=e.is_dir||e.kind==='dir';")
	b.WriteString("const icon=isDir?'\\uD83D\\uDCC1':'\\uD83D\\uDCC4';")
	b.WriteString("const cls=isDir?'dir':'file';")
	b.WriteString("li.innerHTML='<span class=\"icon\">'+icon+'</span><span class=\"'+cls+'\">'+escapeHtml(e.name||e.path)+'</span>';")
	b.WriteString("treeEl.appendChild(li);});")
	b.WriteString("}catch(err){treeEl.innerHTML='<li class=\"muted\">Failed to load tree</li>';}}")
	b.WriteString("async function loadChanges(){changesEl.innerHTML='<li>Loading changes...</li>';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(getProjectID())+'/changes');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const changes=Array.isArray(data.changes)?data.changes:[];")
	b.WriteString("if(changes.length===0){changesEl.innerHTML='<li class=\"muted\">No recent changes</li>';return;}")
	b.WriteString("changesEl.innerHTML='';changes.forEach((c)=>{")
	b.WriteString("const li=document.createElement('li');")
	b.WriteString("const kind=(c.kind||c.status||'modified').toLowerCase();")
	b.WriteString("const badgeCls=kind==='added'?'added':kind==='deleted'?'deleted':'modified';")
	b.WriteString("li.innerHTML='<span class=\"badge '+badgeCls+'\">'+escapeHtml(kind)+'</span><span>'+escapeHtml(c.path||c.name)+'</span>';")
	b.WriteString("changesEl.appendChild(li);});")
	b.WriteString("}catch(err){changesEl.innerHTML='<li class=\"muted\">Failed to load changes</li>';}}")
	b.WriteString("async function loadAll(){statusEl.textContent='Loading...';")
	b.WriteString("await Promise.all([loadTree(),loadChanges()]);statusEl.textContent='Loaded';}")
	b.WriteString("refreshBtn.addEventListener('click',loadAll);")
	b.WriteString("projectEl.addEventListener('change',loadAll);")
	// Scan button
	b.WriteString("document.getElementById('repo-scan').addEventListener('click',async()=>{statusEl.textContent='Scanning...';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(getProjectID())+'/files/scan',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({purge:true})});")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);const data=await res.json();")
	b.WriteString("statusEl.textContent='Scan complete: '+data.synced+' synced, '+data.total+' total ('+data.scan_duration_ms+'ms)';")
	b.WriteString("loadAll();}catch(err){statusEl.textContent='Scan failed';}});")
	b.WriteString("async function loadProjects(){try{")
	b.WriteString("const res=await fetch('/api/v2/projects');if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const projects=Array.isArray(data.projects)?data.projects:[];")
	b.WriteString("projectEl.innerHTML='';")
	b.WriteString("if(projects.length===0){projectEl.innerHTML='<option value=\"proj_default\">proj_default</option>';")
	b.WriteString("}else{projects.forEach((p)=>{const o=document.createElement('option');o.value=p.id;")
	b.WriteString("o.textContent=p.name?p.id+' — '+p.name:p.id;projectEl.appendChild(o);});}")
	b.WriteString("}catch(e){projectEl.innerHTML='<option value=\"proj_default\">proj_default</option>';}")
	b.WriteString("loadAll();}")
	b.WriteString("loadProjects();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func healthDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/health" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Health Dashboard"))
	b.WriteString("<style>")
	b.WriteString(".status-bar{display:flex;gap:4px;height:24px;border-radius:6px;overflow:hidden;margin-bottom:14px;}")
	b.WriteString(".status-bar div{display:flex;align-items:center;justify-content:center;font-size:10px;font-weight:600;color:#fff;}")
	b.WriteString(".bar-pending{background:#858585;} .bar-progress{background:#0078d4;} .bar-done{background:#89d185;} .bar-failed{background:#f14c4c;} .bar-blocked{background:#f87171;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Health Dashboard</h1>")
	b.WriteString("<div class=\"meta\"><span id=\"health-status\" class=\"muted\">Loading...</span>")
	b.WriteString("<button class=\"btn\" id=\"health-refresh\">Refresh</button></div>")
	b.WriteString("<div class=\"grid\" id=\"health-stats\"></div>")
	b.WriteString("<h2>Task Distribution</h2>")
	b.WriteString("<div class=\"status-bar\" id=\"task-bar\"></div>")
	b.WriteString("<h2>System Info</h2>")
	b.WriteString("<table><tbody id=\"sys-info\"><tr><td>Loading...</td></tr></tbody></table>")
	b.WriteString("<h2>Health Checks</h2>")
	b.WriteString("<table><thead><tr><th>Check</th><th>Service</th><th>Result</th><th>Latency</th></tr></thead>")
	b.WriteString("<tbody id=\"audit-body\"><tr><td colspan=\"4\">Running checks...</td></tr></tbody></table>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const statsEl=document.getElementById('health-stats');")
	b.WriteString("const barEl=document.getElementById('task-bar');")
	b.WriteString("const sysEl=document.getElementById('sys-info');")
	b.WriteString("const statusEl=document.getElementById('health-status');")
	b.WriteString("const refreshBtn=document.getElementById('health-refresh');")
	b.WriteString("const auditBody=document.getElementById('audit-body');")
	b.WriteString("function makeStat(label,value,cls){return '<div class=\"stat '+(cls||'')+'\"><div class=\"label\">'+label+'</div><div class=\"value\">'+value+'</div></div>';}")

	// Audit check runner
	b.WriteString("async function runAudit(){auditBody.innerHTML='<tr><td colspan=\"4\">Running checks...</td></tr>';")
	b.WriteString("const checks=[")
	b.WriteString("{name:'Health API',url:'/api/v2/health',validate:r=>r.ok},")
	b.WriteString("{name:'Metrics API',url:'/api/v2/metrics',validate:r=>r.totals!==undefined},")
	b.WriteString("{name:'Version API',url:'/api/v2/version',validate:r=>r.version!==undefined},")
	b.WriteString("{name:'Projects List',url:'/api/v2/projects',validate:r=>Array.isArray(r)},")
	b.WriteString("{name:'Tasks List',url:'/api/v2/tasks',validate:r=>Array.isArray(r.tasks)},")
	b.WriteString("{name:'Agents List',url:'/api/v2/agents',validate:r=>Array.isArray(r)},")
	b.WriteString("{name:'Events List',url:'/api/v2/events',validate:r=>Array.isArray(r.events)},")
	b.WriteString("{name:'Runs List',url:'/api/v2/runs',validate:r=>Array.isArray(r)},")
	b.WriteString("{name:'Migrations',url:'/api/v2/metrics',validate:r=>r.schema_version&&parseInt(r.schema_version)>=17},")
	b.WriteString("{name:'v1 Compat (task)',url:'/task',validate:(r,raw)=>raw.status===200||raw.status===204},")
	b.WriteString("];")
	b.WriteString("let html='';for(const c of checks){const t0=performance.now();let status='PASS',detail='';")
	b.WriteString("try{const res=await fetch(c.url);const ms=Math.round(performance.now()-t0);")
	b.WriteString("if(!res.ok&&c.url!=='/task'){status='FAIL';detail='HTTP '+res.status;}")
	b.WriteString("else if(c.validate){let body;try{body=await res.json();}catch{body=await res.text();}if(!c.validate(body,res)){status='WARN';detail='Unexpected response';}}")
	b.WriteString("html+='<tr><td>'+escapeHtml(c.name)+'</td><td><code>'+escapeHtml(c.url)+'</code></td>'")
	b.WriteString("+'<td style=\"color:'+(status==='PASS'?'#89d185':status==='WARN'?'#ddb347':'#f14c4c')+';font-weight:600\">'+status+(detail?' — '+escapeHtml(detail):'')+'</td>'")
	b.WriteString("+'<td>'+ms+'ms</td></tr>';")
	b.WriteString("}catch(err){html+='<tr><td>'+escapeHtml(c.name)+'</td><td><code>'+escapeHtml(c.url)+'</code></td><td style=\"color:#f14c4c;font-weight:600\">FAIL — '+escapeHtml(err.message)+'</td><td>—</td></tr>';}}")
	b.WriteString("auditBody.innerHTML=html;}")

	b.WriteString("async function loadHealth(){statusEl.textContent='Loading...';try{")
	b.WriteString("const [hRes,mRes,vRes]=await Promise.all([fetch('/api/v2/health'),fetch('/api/v2/metrics'),fetch('/api/v2/version')]);")
	b.WriteString("const health=await hRes.json();const metrics=await mRes.json();const version=await vRes.json();")
	b.WriteString("const t=metrics.totals||{};const ts=metrics.tasks_by_status||{};")
	b.WriteString("const total=t.tasks||0;")
	b.WriteString("statsEl.innerHTML=makeStat('Status',health.ok?'OK':'DOWN',health.ok?'ok':'error')")
	b.WriteString("+makeStat('Tasks',t.tasks||0)+makeStat('Agents',t.agents||0)+makeStat('Runs',t.runs||0)")
	b.WriteString("+makeStat('Events',t.events||0)+makeStat('Projects',t.projects||0)")
	b.WriteString("+makeStat('Active Leases',metrics.active_leases||0)")
	b.WriteString("+makeStat('Circuit',metrics.automation_circuit_state||'-',metrics.automation_circuit_state==='closed'?'ok':'warn');")
	b.WriteString("barEl.innerHTML='';if(total>0){")
	b.WriteString("[['queued','bar-pending',ts.queued],['running','bar-progress',ts.running],['succeeded','bar-done',ts.succeeded],['failed','bar-failed',ts.failed],['review','bar-blocked',ts.needs_review]]")
	b.WriteString(".forEach(([label,cls,count])=>{if(count>0){const pct=Math.max(2,Math.round(count/total*100));")
	b.WriteString("barEl.innerHTML+='<div class=\"'+cls+'\" style=\"flex:'+pct+'\">'+label+' '+count+'</div>';}});}")
	b.WriteString("sysEl.innerHTML='';const rows=[['Version',version.version||'-'],['Schema',metrics.schema_version||'-'],['API v1',version.api&&version.api.v1?'Yes':'No'],['API v2',version.api&&version.api.v2?'Yes':'No']];")
	b.WriteString("rows.forEach(([k,v])=>{sysEl.innerHTML+='<tr><td>'+k+'</td><td>'+v+'</td></tr>';});")
	b.WriteString("statusEl.textContent='Updated '+new Date().toLocaleTimeString();")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load health data';}}")
	b.WriteString("refreshBtn.addEventListener('click',()=>{loadHealth();runAudit();});loadHealth();runAudit();")
	b.WriteString("setInterval(loadHealth,15000);")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func repoGraphHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/graphs/repo" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Repo Graph"))
	b.WriteString("<style>")
	b.WriteString(".graph-canvas{position:relative;min-height:400px;background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;overflow:auto;}")
	b.WriteString("svg.repo-dag{width:100%;height:100%;position:absolute;top:0;left:0;pointer-events:none;}")
	b.WriteString("svg.repo-dag line{stroke:#505050;stroke-width:1;}")
	b.WriteString(".rnode{position:absolute;background:#252526;border:1px solid #505050;border-radius:6px;padding:6px 8px;font-size:10px;z-index:1;cursor:pointer;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;}")
	b.WriteString(".rnode:hover{box-shadow:0 0 0 2px #0078d4;z-index:10;}")
	b.WriteString(".rnode.dir{border-color:#0078d4;color:#0078d4;}")
	b.WriteString(".rnode.file{border-color:#858585;color:#ccc;}")
	b.WriteString(".stats-row{display:flex;gap:16px;margin:12px 0;font-size:12px;}")
	b.WriteString(".stats-row .stat{color:#858585;}")
	b.WriteString(".stats-row .stat strong{color:#ccc;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Repo Entity Graph</h1>")
	b.WriteString("<p>File and directory relationships from project repo files</p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<label class=\"field\">Project<input class=\"input\" id=\"rg-project\" value=\"proj_default\"></label>")
	b.WriteString("<button class=\"btn\" id=\"rg-refresh\">Refresh</button>")
	b.WriteString("<span id=\"rg-status\" class=\"muted\">Loading...</span></div>")
	b.WriteString("<div class=\"stats-row\" id=\"rg-stats\"></div>")
	b.WriteString("<div class=\"graph-canvas\" id=\"rg-canvas\"><svg class=\"repo-dag\" id=\"rg-svg\"></svg></div>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const canvasEl=document.getElementById('rg-canvas');")
	b.WriteString("const svgEl=document.getElementById('rg-svg');")
	b.WriteString("const statusEl=document.getElementById('rg-status');")
	b.WriteString("const statsEl=document.getElementById('rg-stats');")
	b.WriteString("const projectEl=document.getElementById('rg-project');")
	b.WriteString("const refreshBtn=document.getElementById('rg-refresh');")
	b.WriteString("function buildTree(files){const root={name:'/',children:{},isDir:true};")
	b.WriteString("files.forEach(f=>{const parts=(f.path||f.name||'').split('/').filter(Boolean);")
	b.WriteString("let cur=root;parts.forEach((p,i)=>{if(!cur.children[p])cur.children[p]={name:p,children:{},isDir:i<parts.length-1};")
	b.WriteString("cur=cur.children[p];});cur.lang=f.language||'';cur.size=f.size||0;});return root;}")
	b.WriteString("function flatten(node,depth,nodes,edges,parentIdx){")
	b.WriteString("const idx=nodes.length;nodes.push({name:node.name,isDir:node.isDir,depth,lang:node.lang||'',size:node.size||0});")
	b.WriteString("if(parentIdx!==null)edges.push({from:parentIdx,to:idx});")
	b.WriteString("const kids=Object.values(node.children).sort((a,b)=>(b.isDir?1:0)-(a.isDir?1:0)||a.name.localeCompare(b.name));")
	b.WriteString("kids.forEach(c=>flatten(c,depth+1,nodes,edges,idx));}")
	b.WriteString("function renderRepoGraph(files){")
	b.WriteString("const tree=buildTree(files);const nodes=[];const edges=[];flatten(tree,0,nodes,edges,null);")
	b.WriteString("const dirs=nodes.filter(n=>n.isDir).length;const fileCount=nodes.length-dirs;")
	b.WriteString("const langs=new Set(nodes.filter(n=>n.lang).map(n=>n.lang));")
	b.WriteString("statsEl.innerHTML='<span class=\"stat\">Files: <strong>'+fileCount+'</strong></span>'")
	b.WriteString("+'<span class=\"stat\">Dirs: <strong>'+dirs+'</strong></span>'")
	b.WriteString("+'<span class=\"stat\">Languages: <strong>'+langs.size+'</strong></span>';")
	b.WriteString("canvasEl.querySelectorAll('.rnode').forEach(el=>el.remove());svgEl.innerHTML='';")
	b.WriteString("const nodeW=120;const nodeH=28;const padX=20;const padY=8;const indentX=24;")
	b.WriteString("const positions=[];nodes.forEach((n,i)=>{const x=padX+n.depth*indentX;const y=padY+i*(nodeH+padY);positions.push({x,y});")
	b.WriteString("const div=document.createElement('div');div.className='rnode '+(n.isDir?'dir':'file');")
	b.WriteString("div.style.left=x+'px';div.style.top=y+'px';")
	b.WriteString("div.textContent=n.isDir?'\\uD83D\\uDCC1 '+n.name:n.name+(n.lang?' ('+n.lang+')':'');")
	b.WriteString("canvasEl.appendChild(div);});")
	b.WriteString("const totalH=nodes.length*(nodeH+padY)+padY;const totalW=Math.max(...positions.map(p=>p.x))+nodeW+padX*2;")
	b.WriteString("canvasEl.style.height=totalH+'px';canvasEl.style.width=totalW+'px';canvasEl.style.minHeight=totalH+'px';")
	b.WriteString("svgEl.setAttribute('width',totalW);svgEl.setAttribute('height',totalH);svgEl.setAttribute('viewBox','0 0 '+totalW+' '+totalH);")
	b.WriteString("const ns='http://www.w3.org/2000/svg';")
	b.WriteString("edges.forEach(e=>{const from=positions[e.from];const to=positions[e.to];if(!from||!to)return;")
	b.WriteString("const line=document.createElementNS(ns,'line');")
	b.WriteString("line.setAttribute('x1',from.x+6);line.setAttribute('y1',from.y+nodeH);")
	b.WriteString("line.setAttribute('x2',to.x+6);line.setAttribute('y2',to.y);svgEl.appendChild(line);});}")
	b.WriteString("async function loadRepoGraph(){statusEl.textContent='Loading...';try{")
	b.WriteString("const pid=String(projectEl.value||'proj_default').trim();")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/files?limit=500');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const files=Array.isArray(data.files)?data.files:[];")
	b.WriteString("statusEl.textContent=files.length+' files';renderRepoGraph(files);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load repo files';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadRepoGraph);")
	b.WriteString("projectEl.addEventListener('change',loadRepoGraph);")
	b.WriteString("loadRepoGraph();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func diffViewerHandler(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/diffs/")
	if trimmed == "" || trimmed == "/" || strings.Contains(trimmed, "/") {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Diff Viewer"))
	b.WriteString("<style>")
	b.WriteString(".diff-container{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;font-size:12px;line-height:1.6;overflow-x:auto;}")
	b.WriteString(".diff-line{display:flex;white-space:pre;}")
	b.WriteString(".diff-line .ln{min-width:40px;text-align:right;padding:0 8px;color:#858585;user-select:none;cursor:pointer;}")
	b.WriteString(".diff-line .ln:hover{background:#3c3c3c;color:#0078d4;}")
	b.WriteString(".diff-line .content{flex:1;padding:0 8px;}")
	b.WriteString(".diff-add{background:rgba(22,163,74,0.15);}")
	b.WriteString(".diff-add .content{color:#89d185;}")
	b.WriteString(".diff-del{background:rgba(220,38,38,0.15);}")
	b.WriteString(".diff-del .content{color:#f14c4c;}")
	b.WriteString(".diff-hunk{background:rgba(0,120,212,0.1);color:#0078d4;font-style:italic;}")
	b.WriteString(".diff-ctx .content{color:#858585;}")
	b.WriteString(".split{display:grid;grid-template-columns:1fr 1fr;gap:0;}")
	b.WriteString(".split .pane{overflow-x:auto;border:1px solid #3c3c3c;}")
	b.WriteString(".split .pane:first-child{border-right:none;}")
	b.WriteString(".file-list{list-style:none;padding:0;margin:0 0 12px;display:flex;flex-wrap:wrap;gap:6px;font-size:11px;}")
	b.WriteString(".file-list li{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:6px;padding:3px 8px;cursor:pointer;}")
	b.WriteString(".file-list li:hover{background:#3c3c3c;}")
	b.WriteString(".comment-row{background:#252526;border-left:3px solid #0078d4;padding:6px 12px;margin:2px 0;font-size:12px;}")
	b.WriteString(".comment-row .cm-author{color:#0078d4;font-weight:600;margin-right:6px;} .comment-row .cm-body{color:#ccc;} .comment-row .cm-time{color:#858585;font-size:10px;margin-left:8px;}")
	b.WriteString(".comment-form{display:flex;gap:6px;padding:4px 12px;background:#252526;border-left:3px solid #505050;margin:2px 0;}")
	b.WriteString(".comment-form input{flex:1;background:#3c3c3c;border:1px solid #505050;border-radius:4px;color:#ccc;padding:4px 8px;font-size:12px;}")
	b.WriteString(".comment-form button{background:#505050;border:1px solid #606060;color:#ccc;border-radius:4px;padding:4px 10px;cursor:pointer;font-size:11px;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Diff Viewer</h1>")
	b.WriteString("<p>Artifact: <span class=\"muted\" id=\"diff-artifact-id\"></span></p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<button class=\"btn active\" id=\"btn-unified\" type=\"button\">Unified</button>")
	b.WriteString("<button class=\"btn\" id=\"btn-split\" type=\"button\">Split</button>")
	b.WriteString("<span id=\"diff-status\" class=\"muted\">Loading...</span></div>")
	b.WriteString("<ul class=\"file-list\" id=\"diff-files\"></ul>")
	b.WriteString("<div class=\"diff-container\" id=\"diff-content\">Loading...</div>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const artifactId='" + html.EscapeString(trimmed) + "';")
	b.WriteString("document.getElementById('diff-artifact-id').textContent=artifactId;")
	b.WriteString("const contentEl=document.getElementById('diff-content');")
	b.WriteString("const statusEl=document.getElementById('diff-status');")
	b.WriteString("const filesEl=document.getElementById('diff-files');")
	b.WriteString("const btnUnified=document.getElementById('btn-unified');")
	b.WriteString("const btnSplit=document.getElementById('btn-split');")
	b.WriteString("let diffText='';let viewMode='unified';")
	b.WriteString("function parseDiffFiles(text){const files=[];const re=/^diff --git a\\/(.+?) b\\//gm;let m;while((m=re.exec(text))!==null)files.push(m[1]);")
	b.WriteString("if(files.length===0){const hdr=/^--- a\\/(.+)$/gm;while((m=hdr.exec(text))!==null)files.push(m[1]);}return files;}")
	b.WriteString("function renderUnified(text){const lines=text.split('\\n');let html='';let ln=0;")
	b.WriteString("lines.forEach((line)=>{ln++;let cls='diff-ctx';")
	b.WriteString("if(line.startsWith('+')){cls='diff-add';}else if(line.startsWith('-')){cls='diff-del';}else if(line.startsWith('@@')){cls='diff-hunk';}")
	b.WriteString("html+='<div class=\"diff-line '+cls+'\"><span class=\"ln\">'+ln+'</span><span class=\"content\">'+escapeHtml(line)+'</span></div>';});")
	b.WriteString("return html;}")
	b.WriteString("function renderSplit(text){const lines=text.split('\\n');let left='';let right='';let lln=0;let rln=0;")
	b.WriteString("lines.forEach((line)=>{")
	b.WriteString("if(line.startsWith('-')){lln++;left+='<div class=\"diff-line diff-del\"><span class=\"ln\">'+lln+'</span><span class=\"content\">'+escapeHtml(line.slice(1))+'</span></div>';")
	b.WriteString("right+='<div class=\"diff-line\"><span class=\"ln\"></span><span class=\"content\"></span></div>';}")
	b.WriteString("else if(line.startsWith('+')){rln++;right+='<div class=\"diff-line diff-add\"><span class=\"ln\">'+rln+'</span><span class=\"content\">'+escapeHtml(line.slice(1))+'</span></div>';")
	b.WriteString("left+='<div class=\"diff-line\"><span class=\"ln\"></span><span class=\"content\"></span></div>';}")
	b.WriteString("else if(line.startsWith('@@')){const h='<div class=\"diff-line diff-hunk\"><span class=\"ln\"></span><span class=\"content\">'+escapeHtml(line)+'</span></div>';left+=h;right+=h;}")
	b.WriteString("else{lln++;rln++;const row='<div class=\"diff-line diff-ctx\"><span class=\"ln\">'+(lln)+'</span><span class=\"content\">'+escapeHtml(line)+'</span></div>';left+=row;right+=row;}});")
	b.WriteString("return '<div class=\"split\"><div class=\"pane\">'+left+'</div><div class=\"pane\">'+right+'</div></div>';}")
	b.WriteString("function renderDiff(){if(!diffText){contentEl.innerHTML='<span class=\"muted\">No diff content</span>';return;}")
	b.WriteString("contentEl.innerHTML=viewMode==='split'?renderSplit(diffText):renderUnified(diffText);}")
	b.WriteString("async function loadDiff(){statusEl.textContent='Loading...';contentEl.innerHTML='Loading...';try{")
	b.WriteString("const res=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId)+'/content');")
	b.WriteString("if(res.ok){diffText=await res.text();}else{")
	b.WriteString("const res2=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId));")
	b.WriteString("if(!res2.ok)throw new Error('http '+res2.status);const data=await res2.json();diffText=data.content||data.body||JSON.stringify(data,null,2);}")
	b.WriteString("const files=parseDiffFiles(diffText);filesEl.innerHTML='';")
	b.WriteString("files.forEach((f)=>{const li=document.createElement('li');li.textContent=f;filesEl.appendChild(li);});")
	b.WriteString("statusEl.textContent=files.length+' file(s)';renderDiff();")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load diff';contentEl.innerHTML='<span class=\"muted\">Could not load artifact</span>';}}")
	b.WriteString("btnUnified.addEventListener('click',()=>{viewMode='unified';btnUnified.classList.add('active');btnSplit.classList.remove('active');renderDiff();});")
	b.WriteString("btnSplit.addEventListener('click',()=>{viewMode='split';btnSplit.classList.add('active');btnUnified.classList.remove('active');renderDiff();});")
	// Line comment functionality
	b.WriteString("let comments=[];")
	b.WriteString("async function loadComments(){try{const res=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId)+'/comments');if(res.ok){const data=await res.json();comments=data.comments||[];}}catch(e){}}")
	b.WriteString("async function postComment(lineNumber,body){try{const res=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId)+'/comments',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({line_number:lineNumber,body:body,author:'reviewer'})});if(res.ok){await loadComments();renderDiff();}}catch(e){}}")
	b.WriteString("function renderCommentsForLine(container,ln){const lc=comments.filter(c=>c.line_number===ln);lc.forEach(c=>{const div=document.createElement('div');div.className='comment-row';div.innerHTML='<span class=\"cm-author\">'+escapeHtml(c.author||'anon')+'</span><span class=\"cm-body\">'+escapeHtml(c.body)+'</span><span class=\"cm-time\">'+escapeHtml(c.created_at||'')+'</span>';container.appendChild(div);});}")
	b.WriteString("contentEl.addEventListener('click',(e)=>{const lnEl=e.target.closest('.ln');if(!lnEl)return;const ln=parseInt(lnEl.textContent,10);if(isNaN(ln))return;")
	b.WriteString("const existing=lnEl.parentElement.nextElementSibling;if(existing&&existing.classList.contains('comment-form')){existing.remove();return;}")
	b.WriteString("const form=document.createElement('div');form.className='comment-form';form.innerHTML='<input placeholder=\"Add comment...\" autofocus><button>Post</button>';")
	b.WriteString("lnEl.parentElement.after(form);const inp=form.querySelector('input');inp.focus();form.querySelector('button').addEventListener('click',()=>{const val=inp.value.trim();if(val)postComment(ln,val);});inp.addEventListener('keydown',(ev)=>{if(ev.key==='Enter'){const val=inp.value.trim();if(val)postComment(ln,val);}if(ev.key==='Escape')form.remove();});});")
	// Enhance renderDiff to include inline comments
	b.WriteString("const origRenderDiff=renderDiff;")
	b.WriteString("renderDiff=function(){origRenderDiff();document.querySelectorAll('.diff-line').forEach(el=>{const lnEl=el.querySelector('.ln');if(!lnEl)return;const ln=parseInt(lnEl.textContent,10);if(isNaN(ln))return;renderCommentsForLine(el.parentElement||contentEl,ln);});};")
	b.WriteString("loadComments().then(()=>{loadDiff();});")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func runsPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/runs" {
		r.URL.Path = "/runs/"
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/runs/")
	if trimmed == "" || trimmed == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder
		b.WriteString(subPageHead("Runs"))
		b.WriteString("<style>")
		b.WriteString("label{font-size:12px;color:#b0b0b0;margin-bottom:6px;display:block;}")
		b.WriteString(".row{display:flex;flex-wrap:wrap;gap:10px;align-items:center;}")
		b.WriteString("input{flex:1;min-width:220px;}")
		b.WriteString(".link{font-size:12px;color:#0078d4;text-decoration:none;border:1px dashed #505050;padding:8px 12px;border-radius:4px;}")
		b.WriteString(".link.disabled{color:#858585;border-color:#3c3c3c;pointer-events:none;}")
		b.WriteString(".runs-table{width:100%;border-collapse:collapse;margin-top:16px;font-size:12px;}")
		b.WriteString(".runs-table th{text-align:left;color:#858585;font-weight:500;padding:6px 10px;border-bottom:1px solid #505050;}")
		b.WriteString(".runs-table td{padding:6px 10px;border-bottom:1px solid #3c3c3c;color:#ccc;}")
		b.WriteString(".runs-table tr:hover td{background:#2a2a2a;}")
		b.WriteString(".status-badge{display:inline-block;padding:2px 8px;border-radius:3px;font-size:11px;font-weight:500;}")
		b.WriteString(".status-running{background:#1a3a1a;color:#4ec94e;}")
		b.WriteString(".status-succeeded{background:#1a331a;color:#73c991;}")
		b.WriteString(".status-failed{background:#3a1a1a;color:#f48771;}")
		b.WriteString(".status-cancelled{background:#3a3a1a;color:#cca700;}")
		b.WriteString(".empty{color:#858585;padding:20px 0;text-align:center;}")
		b.WriteString("</style>")
		b.WriteString("<div class=\"card\">")
		b.WriteString("<h1>Runs</h1>")
		b.WriteString("<p>Look up a run by ID or browse recent activity below.</p>")
		b.WriteString("<form id=\"run-lookup\" autocomplete=\"off\">")
		b.WriteString("<label for=\"run-id\">Run ID</label>")
		b.WriteString("<div class=\"row\">")
		b.WriteString("<input class=\"input\" id=\"run-id\" name=\"runId\" placeholder=\"run_abc123\" required>")
		b.WriteString("<button class=\"btn\" type=\"submit\">Open run</button>")
		b.WriteString("<a class=\"link disabled\" id=\"run-link\" href=\"/runs\">Open /runs/{runId}</a>")
		b.WriteString("</div>")
		b.WriteString("</form>")
		b.WriteString("</div>")
		b.WriteString("<div class=\"card\" style=\"margin-top:12px;\">")
		b.WriteString("<h2 style=\"margin-bottom:8px;\">Recent Runs</h2>")
		b.WriteString("<div id=\"runs-list\"><p class=\"empty\">Loading...</p></div>")
		b.WriteString("</div>")
		b.WriteString("<script>")
		b.WriteString("const form=document.getElementById('run-lookup');")
		b.WriteString("const input=document.getElementById('run-id');")
		b.WriteString("const link=document.getElementById('run-link');")
		b.WriteString("function updateLink(){const value=input.value.trim();")
		b.WriteString("if(value){const href='/runs/'+encodeURIComponent(value);")
		b.WriteString("link.href=href;link.textContent='Open '+href;link.classList.remove('disabled');}")
		b.WriteString("else{link.href='/runs';link.textContent='Open /runs/{runId}';link.classList.add('disabled');}}")
		b.WriteString("form.addEventListener('submit',(e)=>{e.preventDefault();")
		b.WriteString("const value=input.value.trim();if(!value){input.focus();return;}")
		b.WriteString("window.location.href='/runs/'+encodeURIComponent(value);});")
		b.WriteString("input.addEventListener('input',updateLink);updateLink();")
		// Recent runs loader
		b.WriteString("async function loadRuns(){const el=document.getElementById('runs-list');")
		b.WriteString("try{const resp=await fetch('/api/v2/runs?limit=50');")
		b.WriteString("if(!resp.ok){el.innerHTML='<p class=\"empty\">Failed to load runs ('+resp.status+')</p>';return;}")
		b.WriteString("const data=await resp.json();const runs=data.runs||data||[];")
		b.WriteString("if(!runs.length){el.innerHTML='<p class=\"empty\">No runs yet.</p>';return;}")
		b.WriteString("let html='<table class=\"runs-table\"><thead><tr><th>Run ID</th><th>Task</th><th>Agent</th><th>Status</th><th>Started</th></tr></thead><tbody>';")
		b.WriteString("runs.forEach(r=>{const sc=r.status||'unknown';")
		b.WriteString("const cls=sc.toLowerCase().includes('run')?'running':sc.toLowerCase().includes('succ')?'succeeded':sc.toLowerCase().includes('fail')?'failed':'cancelled';")
		b.WriteString("const started=r.started_at?new Date(r.started_at).toLocaleString():'—';")
		b.WriteString("html+='<tr><td><a href=\"/runs/'+encodeURIComponent(r.id)+'\" style=\"color:#0078d4;\">'+r.id+'</a></td>';")
		b.WriteString("html+='<td>#'+(r.task_id||'—')+'</td>';")
		b.WriteString("html+='<td>'+(r.agent_id||'—')+'</td>';")
		b.WriteString("html+='<td><span class=\"status-badge status-'+cls+'\">'+sc+'</span></td>';")
		b.WriteString("html+='<td>'+started+'</td></tr>';});")
		b.WriteString("html+='</tbody></table>';el.innerHTML=html;}")
		b.WriteString("catch(err){el.innerHTML='<p class=\"empty\">Error: '+err+'</p>';}}")
		b.WriteString("loadRuns();")
		b.WriteString("</script>")
		b.WriteString(subPageFoot())
		fmt.Fprint(w, b.String())
		return
	}
	if strings.Contains(trimmed, "/") {
		http.NotFound(w, r)
		return
	}
	page := runViewerPage(trimmed)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

func contextPacksPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/context-packs" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, contextPacksCreatePage())
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/context-packs/")
	if trimmed == "" || trimmed == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, contextPacksCreatePage())
		return
	}
	if strings.Contains(trimmed, "/") {
		http.NotFound(w, r)
		return
	}
	page := contextPackDetailPage(trimmed)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

func contextPacksCreatePage() string {
	var b strings.Builder
	b.WriteString(subPageHead("Context Packs"))
	b.WriteString("<style>")
	b.WriteString("label{font-size:12px;color:#858585;margin-bottom:6px;display:block;}")
	b.WriteString(".row{display:flex;flex-wrap:wrap;gap:10px;align-items:center;}")
	b.WriteString(".meta{margin-top:8px;color:#858585;font-size:12px;}")
	b.WriteString(".status{margin-top:10px;color:#ccc;font-size:12px;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Context Packs</h1>")
	b.WriteString("<p>Create a minimal context pack for a task and view the JSON response.</p>")
	b.WriteString("<form id=\"context-pack-form\" autocomplete=\"off\">")
	b.WriteString("<label for=\"cp-project\">Project ID</label>")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<input id=\"cp-project\" name=\"project\" value=\"proj_default\">")
	b.WriteString("<button class=\"btn\" type=\"button\" id=\"load-tasks-btn\" style=\"font-size:11px;\">Load Tasks</button>")
	b.WriteString("</div>")
	b.WriteString("<label for=\"task-id\">Task ID</label>")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<select id=\"task-id\" name=\"taskId\" required><option value=\"\">-- select a task --</option></select>")
	b.WriteString("<input id=\"task-id-manual\" type=\"number\" min=\"1\" placeholder=\"or type ID\" style=\"width:90px;\">")
	b.WriteString("</div>")
	b.WriteString("<label for=\"query\">Query (optional)</label>")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<input id=\"query\" name=\"query\" placeholder=\"focus on migrations\">")
	b.WriteString("<button class=\"btn\" type=\"submit\">Create pack</button>")
	b.WriteString("</div>")
	b.WriteString("</form>")
	b.WriteString("<div class=\"meta\" id=\"cp-endpoint\">POST /api/v2/projects/proj_default/context-packs</div>")
	b.WriteString("<div class=\"status\" id=\"context-pack-status\">Status: idle</div>")
	b.WriteString("<div id=\"cp-link\" style=\"display:none;margin-top:8px;\"></div>")
	b.WriteString("<pre id=\"context-pack-output\">{}</pre>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const form=document.getElementById('context-pack-form');")
	b.WriteString("const taskSelect=document.getElementById('task-id');")
	b.WriteString("const taskManual=document.getElementById('task-id-manual');")
	b.WriteString("const queryInput=document.getElementById('query');")
	b.WriteString("const projectInput=document.getElementById('cp-project');")
	b.WriteString("const statusEl=document.getElementById('context-pack-status');")
	b.WriteString("const outputEl=document.getElementById('context-pack-output');")
	b.WriteString("const endpointEl=document.getElementById('cp-endpoint');")
	b.WriteString("const linkEl=document.getElementById('cp-link');")
	// Load tasks into dropdown
	b.WriteString("async function loadTasks(){const pid=String(projectInput.value||'proj_default').trim();")
	b.WriteString("try{const r=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/tasks');")
	b.WriteString("if(!r.ok)return;const d=await r.json();const tasks=d.tasks||[];")
	b.WriteString("taskSelect.innerHTML='<option value=\"\">-- select a task --</option>';")
	b.WriteString("tasks.forEach(t=>{const o=document.createElement('option');o.value=t.id;")
	b.WriteString("o.textContent='#'+t.id+' '+((t.title||'').substring(0,60));taskSelect.appendChild(o);});")
	b.WriteString("}catch(e){console.error(e);}}")
	b.WriteString("document.getElementById('load-tasks-btn').addEventListener('click',loadTasks);")
	b.WriteString("loadTasks();")
	// Manual ID overrides dropdown
	b.WriteString("taskManual.addEventListener('input',()=>{if(taskManual.value){taskSelect.value='';}});")
	b.WriteString("taskSelect.addEventListener('change',()=>{if(taskSelect.value){taskManual.value='';}});")
	// Form submit
	b.WriteString("form.addEventListener('submit',async(e)=>{e.preventDefault();")
	b.WriteString("const taskId=taskManual.value.trim()||taskSelect.value;")
	b.WriteString("if(!taskId){statusEl.textContent='Status: select or enter a task ID';return;}")
	b.WriteString("const pid=String(projectInput.value||'proj_default').trim();")
	b.WriteString("endpointEl.textContent='POST /api/v2/projects/'+pid+'/context-packs';")
	b.WriteString("const payload={task_id:parseInt(taskId,10)};")
	b.WriteString("const query=queryInput.value.trim();")
	b.WriteString("if(query){payload.query=query;}")
	b.WriteString("statusEl.textContent='Status: sending...';outputEl.textContent='';linkEl.style.display='none';")
	b.WriteString("try{const resp=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/context-packs',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});")
	b.WriteString("const text=await resp.text();")
	b.WriteString("let data=text;try{data=JSON.parse(text);}catch(e){}")
	b.WriteString("outputEl.textContent=typeof data==='string'?data:JSON.stringify(data,null,2);")
	b.WriteString("statusEl.textContent='Status: '+resp.status;")
	// Show link to view pack on success
	b.WriteString("if(resp.ok&&typeof data==='object'){const pack=data.context_pack||data;")
	b.WriteString("if(pack&&pack.id){linkEl.innerHTML='<a href=\"/context-packs/'+encodeURIComponent(pack.id)+'\" style=\"color:#0078d4;\">View pack '+pack.id+'</a>';linkEl.style.display='block';}}")
	b.WriteString("}catch(err){statusEl.textContent='Status: error';outputEl.textContent=String(err);}});")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	return b.String()
}

const htmlTemplate = `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Cocopilot - Work Queue</title>
        <script defer src="/static/alpine-collapse.min.js"></script>
        <script defer src="/static/alpine.min.js"></script>
        <style>
            * {
                box-sizing: border-box;
                margin: 0;
                padding: 0;
            }

            :root {
                --vscode-bg: #1e1e1e;
                --vscode-sidebar: #252526;
                --vscode-input-bg: #3c3c3c;
                --vscode-border: #3c3c3c;
                --vscode-text: #cccccc;
                --vscode-text-muted: #858585;
                --vscode-accent: #0078d4;
                --vscode-accent-hover: #1c8ae8;
                --vscode-success: #89d185;
                --vscode-warning: #cca700;
                --vscode-info: #3794ff;
                --vscode-error: #f48771;
                --vscode-dropdown-bg: #313131;
                --vscode-charts-green: #388a34;
                --vscode-charts-red: #c74e39;
                --vscode-charts-orange: #d18616;
                --vscode-charts-blue: #2670b8;
            }

            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                font-size: 13px;
                background: var(--vscode-bg);
                color: var(--vscode-text);
                height: 100vh;
                display: flex;
                flex-direction: column;
                overflow: hidden;
            }

            /* Header */
            .header {
                background: var(--vscode-sidebar);
                border-bottom: 1px solid var(--vscode-border);
                padding: 12px 20px;
                display: flex;
                align-items: center;
                gap: 12px;
                flex-shrink: 0;
            }

            .header-icon { color: var(--vscode-accent); }
            .header-title { font-size: 14px; font-weight: 600; }

            .header-nav {
                display: flex;
                align-items: center;
                gap: 10px;
                font-size: 12px;
            }

            .header-nav a {
                color: var(--vscode-text-muted);
                text-decoration: none;
                padding: 4px 6px;
                border-radius: 4px;
                transition: all 0.15s;
            }

            .header-nav a:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }
            .header-nav a.active {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
                font-weight: 600;
            }
            .nav-more {
                position: relative;
            }
            .nav-more-btn {
                color: var(--vscode-text-muted);
                background: none;
                border: none;
                cursor: pointer;
                font-size: 12px;
                padding: 4px 6px;
                border-radius: 4px;
                transition: all 0.15s;
            }
            .nav-more-btn:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }
            .nav-more-dropdown {
                position: absolute;
                top: 100%;
                left: 0;
                background: var(--vscode-dropdown-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                box-shadow: 0 4px 12px rgba(0,0,0,0.2);
                min-width: 160px;
                z-index: 1000;
                padding: 4px 0;
            }
            .nav-more-dropdown a {
                display: block;
                padding: 6px 12px;
                color: var(--vscode-text-muted);
                text-decoration: none;
                font-size: 12px;
            }
            .nav-more-dropdown a:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }

            .header-actions {
                margin-left: auto;
                display: flex;
                align-items: center;
                gap: 12px;
            }

            .header-btn {
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                color: var(--vscode-text);
                padding: 6px 12px;
                font-size: 12px;
                cursor: pointer;
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 6px;
                transition: all 0.15s;
            }

            .header-btn:hover {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
            }

            .workdir-input {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .workdir-input label {
                font-size: 12px;
                color: var(--vscode-text-muted);
            }

            .workdir-input input {
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 4px;
                color: var(--vscode-text);
                padding: 6px 10px;
                font-size: 12px;
                width: 250px;
                outline: none;
            }

            .workdir-input input:focus {
                border-color: var(--vscode-accent);
            }

            /* Kanban Board */
            .kanban-board {
                flex: 1;
                display: grid;
                grid-template-columns: repeat(3, 1fr);
                gap: 16px;
                padding: 20px;
                overflow: hidden;
                min-height: 0;
            }

            .kanban-column {
                background: var(--vscode-sidebar);
                border-radius: 8px;
                display: flex;
                flex-direction: column;
                overflow: hidden;
                border: 1px solid var(--vscode-border);
                transition: border-color 0.2s;
                min-height: 0;
            }

            .kanban-column.drag-over {
                border-color: var(--vscode-accent);
                box-shadow: 0 0 0 2px rgba(0, 120, 212, 0.3);
            }

            .column-header {
                padding: 12px 16px;
                border-bottom: 1px solid var(--vscode-border);
                display: flex;
                align-items: center;
                gap: 8px;
                flex-shrink: 0;
            }

            .column-title { font-weight: 600; font-size: 13px; }

            .column-count {
                background: var(--vscode-input-bg);
                padding: 2px 8px;
                border-radius: 10px;
                font-size: 11px;
                color: var(--vscode-text-muted);
                transition: all 0.3s;
            }

            .column-icon { width: 16px; height: 16px; }
            .col-todo .column-icon { color: var(--vscode-warning); }
            .col-progress .column-icon { color: var(--vscode-info); }
            .col-done .column-icon { color: var(--vscode-success); }

            .column-body {
                flex: 1;
                overflow-y: auto;
                padding: 12px;
                display: flex;
                flex-direction: column;
                gap: 10px;
                min-height: 0;
            }

            .column-body::-webkit-scrollbar { width: 8px; }
            .column-body::-webkit-scrollbar-track { background: transparent; }
            .column-body::-webkit-scrollbar-thumb { background: var(--vscode-border); border-radius: 4px; }

            /* Task Cards */
            .task-card {
                background: var(--vscode-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                cursor: grab;
                transition: all 0.2s ease;
                overflow: hidden;
                flex-shrink: 0;
            }

            .task-card:hover {
                border-color: var(--vscode-accent);
                box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
            }

            .task-card.dragging {
                opacity: 0.5;
                cursor: grabbing;
                transform: scale(0.98);
            }

            .task-card.task-blocked {
                border-left: 3px solid var(--vscode-error);
                opacity: 0.7;
            }

            .card-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 10px 12px;
                cursor: pointer;
                user-select: none;
            }

            .card-header:hover { background: rgba(255,255,255,0.03); }

            .card-left {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .card-id {
                font-weight: 600;
                color: var(--vscode-accent);
                font-size: 12px;
            }

            .card-preview {
                font-size: 11px;
                color: var(--vscode-text-muted);
                max-width: 150px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }

            .card-right {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .card-time {
                font-size: 10px;
                color: var(--vscode-text-muted);
            }

            .delete-btn {
                background: transparent;
                border: none;
                color: var(--vscode-text-muted);
                cursor: pointer;
                padding: 2px;
                border-radius: 4px;
                display: flex;
                align-items: center;
                justify-content: center;
                transition: all 0.15s;
            }

            .delete-btn:hover {
                background: rgba(255, 0, 0, 0.2);
                color: #ff6b6b;
            }

            .expand-icon {
                transition: transform 0.2s;
                color: var(--vscode-text-muted);
            }

            .expand-icon.rotated { transform: rotate(180deg); }

            .card-body {
                padding: 0 12px 12px 12px;
                overflow: hidden;
            }

            .card-instructions {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 11px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-wrap: break-word;
                color: var(--vscode-text);
                margin: 0;
                padding: 10px;
                background: rgba(0,0,0,0.2);
                border-radius: 4px;
                max-height: 300px;
                overflow-y: auto;
            }

            .card-output {
                margin-top: 10px;
                padding: 10px;
                background: rgba(137, 209, 133, 0.1);
                border-radius: 4px;
                border-left: 3px solid var(--vscode-success);
            }

            .card-output-label {
                font-size: 10px;
                color: var(--vscode-success);
                text-transform: uppercase;
                letter-spacing: 0.5px;
                margin-bottom: 6px;
            }

            .card-output-text {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 11px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-wrap: break-word;
                color: var(--vscode-text);
                max-height: 200px;
                overflow-y: auto;
            }

            /* Empty state */
            .empty-column {
                display: flex;
                align-items: center;
                justify-content: center;
                height: 80px;
                color: var(--vscode-text-muted);
                font-style: italic;
                font-size: 12px;
            }

            /* Modal */
            .modal-overlay {
                position: fixed;
                top: 0; left: 0; right: 0; bottom: 0;
                background: rgba(0, 0, 0, 0.6);
                z-index: 1000;
                display: flex;
                align-items: center;
                justify-content: center;
            }

            .modal {
                background: var(--vscode-sidebar);
                border: 1px solid var(--vscode-border);
                border-radius: 8px;
                max-width: 500px;
                width: 90%;
                box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
            }

            .modal-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                padding: 16px;
                border-bottom: 1px solid var(--vscode-border);
            }

            .modal-title { font-size: 14px; font-weight: 600; }

            .modal-close {
                background: transparent;
                border: none;
                color: var(--vscode-text-muted);
                cursor: pointer;
                padding: 4px;
                border-radius: 4px;
            }

            .modal-close:hover {
                background: var(--vscode-input-bg);
                color: var(--vscode-text);
            }

            .modal-body { padding: 16px; }

            .modal-body textarea {
                width: 100%;
                min-height: 120px;
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                color: var(--vscode-text);
                font-family: inherit;
                font-size: 13px;
                padding: 12px;
                resize: vertical;
                outline: none;
            }

            .modal-body textarea:focus { border-color: var(--vscode-accent); }

            .modal-footer {
                padding: 12px 16px;
                border-top: 1px solid var(--vscode-border);
                display: flex;
                justify-content: flex-end;
                gap: 8px;
            }

            .btn {
                padding: 8px 16px;
                border-radius: 4px;
                font-size: 12px;
                cursor: pointer;
                border: 1px solid var(--vscode-border);
                transition: all 0.15s;
            }

            .btn-primary {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
                color: #fff;
            }

            .btn-primary:hover { background: var(--vscode-accent-hover); }
            .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

            .btn-secondary {
                background: var(--vscode-input-bg);
                color: var(--vscode-text);
            }

            .btn-secondary:hover { background: #4a4a4a; }

            /* Code block for instructions */
            .code-block {
                background: var(--vscode-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                overflow: hidden;
            }

            .code-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                padding: 8px 12px;
                background: var(--vscode-input-bg);
                border-bottom: 1px solid var(--vscode-border);
            }

            .code-label {
                font-size: 11px;
                color: var(--vscode-text-muted);
                text-transform: uppercase;
                letter-spacing: 0.5px;
            }

            .copy-btn {
                background: transparent;
                border: 1px solid var(--vscode-border);
                color: var(--vscode-text);
                padding: 4px 10px;
                font-size: 11px;
                cursor: pointer;
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 4px;
                transition: all 0.15s;
            }

            .copy-btn:hover {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
            }

            .copy-btn.copied {
                background: var(--vscode-success);
                border-color: var(--vscode-success);
                color: #1e1e1e;
            }

            .code-content {
                padding: 16px;
            }

            .code-content pre {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 13px;
                line-height: 1.6;
                color: var(--vscode-text);
                margin: 0;
                white-space: pre-wrap;
                word-wrap: break-word;
            }

            .modal.modal-wide {
                max-width: 600px;
            }

            /* Animations */
            .fade-enter-active, .fade-leave-active {
                transition: opacity 0.2s, transform 0.2s;
            }
            .fade-enter-from, .fade-leave-to {
                opacity: 0;
                transform: translateY(-10px);
            }

            [x-cloak] { display: none !important; }

            /* Toast notifications */
            .toast-container { position: fixed; top: 16px; right: 16px; z-index: 9999; display: flex; flex-direction: column; gap: 8px; }
            .toast { background: var(--vscode-sidebar); border: 1px solid var(--vscode-border); border-radius: 6px; padding: 10px 16px; font-size: 12px; color: var(--vscode-text); box-shadow: 0 4px 12px rgba(0,0,0,0.4); display: flex; align-items: center; gap: 8px; animation: slideIn 0.2s ease; min-width: 200px; }
            .toast.toast-success { border-left: 3px solid var(--vscode-success); }
            .toast.toast-error { border-left: 3px solid var(--vscode-error); }
            .toast.toast-info { border-left: 3px solid var(--vscode-info); }
            @keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }
            @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

            /* Confirm dialog */
            .confirm-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.6); z-index: 2000; display: flex; align-items: center; justify-content: center; }
            .confirm-dialog { background: var(--vscode-sidebar); border: 1px solid var(--vscode-border); border-radius: 8px; padding: 20px; max-width: 400px; width: 90%; box-shadow: 0 8px 32px rgba(0,0,0,0.4); }
            .confirm-dialog p { margin-bottom: 16px; font-size: 13px; }
            .confirm-actions { display: flex; justify-content: flex-end; gap: 8px; }

            /* Filter bar */
            .filter-bar { display: flex; align-items: center; gap: 10px; padding: 8px 20px; background: var(--vscode-sidebar); border-bottom: 1px solid var(--vscode-border); flex-shrink: 0; }
            .filter-bar input { background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 5px 10px; font-size: 12px; width: 220px; outline: none; }
            .filter-bar input:focus { border-color: var(--vscode-accent); }
            .filter-bar select { background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 5px 8px; font-size: 12px; outline: none; cursor: pointer; }
            .filter-bar .filter-label { font-size: 11px; color: var(--vscode-text-muted); }
            .filter-bar .task-count { font-size: 11px; color: var(--vscode-text-muted); margin-left: auto; }

            /* Type/priority badges on cards */
            .card-badges { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 6px; padding: 0 12px 8px; }
            .badge { display: inline-block; padding: 1px 6px; border-radius: 3px; font-size: 10px; font-weight: 500; text-transform: uppercase; }
            .badge-type { background: rgba(55,148,255,0.15); color: var(--vscode-info); }
            .badge-priority-high { background: rgba(244,135,113,0.15); color: var(--vscode-error); }
            .badge-priority-medium { background: rgba(204,167,0,0.15); color: var(--vscode-warning); }
            .badge-priority-low { background: rgba(137,209,133,0.15); color: var(--vscode-success); }
            .badge-agent { background: rgba(204,204,204,0.1); color: var(--vscode-text-muted); }
            .badge-blocked { background: rgba(244,135,113,0.2); color: var(--vscode-error); }
            .badge-deps { background: rgba(55,148,255,0.1); color: var(--vscode-text-muted); font-size: 9px; }
            .badge-approval { background: rgba(204,167,0,0.15); color: var(--vscode-warning); }
            .badge-status-v2 { font-size: 9px; letter-spacing: 0.3px; font-weight: 600; }
            .badge-v2-QUEUED { background: rgba(55,148,255,0.15); color: #56b6f7; }
            .badge-v2-CLAIMED { background: rgba(204,167,0,0.15); color: #cca700; }
            .badge-v2-RUNNING { background: rgba(55,148,255,0.2); color: #3794ff; }
            .badge-v2-SUCCEEDED { background: rgba(137,209,133,0.15); color: #89d185; }
            .badge-v2-FAILED { background: rgba(244,135,113,0.2); color: #f48771; }
            .badge-v2-NEEDS_REVIEW { background: rgba(204,167,0,0.2); color: #e8c600; }
            .badge-v2-CANCELLED { background: rgba(133,133,133,0.15); color: #999; }

            /* Task detail drawer */
            .detail-drawer { position: fixed; top: 0; right: 0; bottom: 0; width: 480px; background: var(--vscode-sidebar); border-left: 1px solid var(--vscode-border); z-index: 900; box-shadow: -4px 0 16px rgba(0,0,0,0.3); display: flex; flex-direction: column; transform: translateX(100%); transition: transform 0.2s ease; }
            .detail-drawer.open { transform: translateX(0); }
            .drawer-header { display: flex; align-items: center; justify-content: space-between; padding: 16px; border-bottom: 1px solid var(--vscode-border); flex-shrink: 0; }
            .drawer-header h2 { font-size: 14px; font-weight: 600; }
            .drawer-body { flex: 1; overflow-y: auto; padding: 16px; }
            .drawer-section { margin-bottom: 16px; }
            .drawer-section h3 { font-size: 12px; color: var(--vscode-text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
            .drawer-section pre { font-family: 'Cascadia Code','Fira Code',Consolas,monospace; font-size: 11px; line-height: 1.5; white-space: pre-wrap; word-wrap: break-word; color: var(--vscode-text); padding: 10px; background: rgba(0,0,0,0.2); border-radius: 4px; margin: 0; }
            .drawer-meta { display: grid; grid-template-columns: 100px 1fr; gap: 6px; font-size: 12px; }
            .drawer-meta dt { color: var(--vscode-text-muted); }
            .drawer-meta dd { color: var(--vscode-text); margin: 0; }
            .drawer-actions { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 12px; }
            .drawer-actions button { background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); color: var(--vscode-text); padding: 4px 10px; border-radius: 4px; font-size: 11px; cursor: pointer; }
            .drawer-actions button:hover { border-color: var(--vscode-accent); }
            .drawer-actions button.danger { color: var(--vscode-error); }
            .drawer-actions button.danger:hover { border-color: var(--vscode-error); }

            /* Task hierarchy - indentation for child tasks */
            .task-card[data-depth="1"] { margin-left: 20px; border-left: 3px solid var(--vscode-accent); }
            .task-card[data-depth="2"] { margin-left: 40px; border-left: 3px solid var(--vscode-info); }
            .task-card[data-depth="3"] { margin-left: 60px; border-left: 3px solid var(--vscode-warning); }

            .parent-badge {
                display: inline-flex;
                align-items: center;
                gap: 4px;
                font-size: 10px;
                color: var(--vscode-text-muted);
                background: var(--vscode-input-bg);
                padding: 2px 6px;
                border-radius: 4px;
            }

            .parent-badge svg {
                width: 10px;
                height: 10px;
            }

            /* Parent task select in modal */
            .modal-body select {
                width: 100%;
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                color: var(--vscode-text);
                font-family: inherit;
                font-size: 13px;
                padding: 10px 12px;
                outline: none;
                cursor: pointer;
            }

            .modal-body select:focus { border-color: var(--vscode-accent); }

            .modal-body label {
                display: block;
                font-size: 12px;
                color: var(--vscode-text-muted);
                margin-bottom: 6px;
            }

            .form-group {
                margin-bottom: 12px;
            }

            /* Agent Status Styles */
            .status-badge {
                border-radius: 10px;
                font-size: 10px;
                font-weight: 500;
                padding: 2px 6px;
                text-transform: uppercase;
            }

            .status-badge.status-online {
                background: var(--vscode-charts-green);
                color: white;
            }

            .status-badge.status-offline {
                background: var(--vscode-charts-red);
                color: white;
            }

            .status-badge.status-busy {
                background: var(--vscode-charts-orange);
                color: white;
            }

            .status-badge.status-idle {
                background: var(--vscode-charts-blue);
                color: white;
            }

            .agent-item:last-child {
                border-bottom: none;
            }

            .agent-status [x-cloak] { display: none !important; }

            /* Footer */
            .footer {
                background: var(--vscode-sidebar);
                border-top: 1px solid var(--vscode-border);
                padding: 10px 20px;
                text-align: center;
                font-size: 12px;
                color: var(--vscode-text-muted);
                flex-shrink: 0;
            }

            .footer a {
                color: var(--vscode-accent);
                text-decoration: none;
            }

            .footer a:hover {
                text-decoration: underline;
            }
        </style>
    </head>
    <body x-data="kanbanApp()" x-init="init()">
        <!-- Toast notifications -->
        <div class="toast-container">
            <template x-for="(t, i) in toasts" :key="i">
                <div class="toast" :class="'toast-' + t.type" x-text="t.message"
                     x-init="setTimeout(() => toasts.splice(toasts.indexOf(t), 1), 3000)"></div>
            </template>
        </div>

        <!-- Confirm dialog -->
        <div class="confirm-overlay" x-show="confirmDialog.show" x-cloak @click.self="confirmDialog.show = false">
            <div class="confirm-dialog">
                <p x-text="confirmDialog.message"></p>
                <div class="confirm-actions">
                    <button class="btn btn-secondary" @click="confirmDialog.show = false">Cancel</button>
                    <button class="btn btn-primary" @click="confirmDialog.onConfirm(); confirmDialog.show = false">Confirm</button>
                </div>
            </div>
        </div>

        <!-- Task detail drawer -->
        <div class="detail-drawer" :class="{ 'open': selectedTask }" x-show="selectedTask" x-cloak>
            <div class="drawer-header">
                <h2 x-text="selectedTask ? '#' + selectedTask.id + ' Details' : ''"></h2>
                <button class="modal-close" @click="selectedTask = null">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                        <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                    </svg>
                </button>
            </div>
            <div class="drawer-body" x-show="selectedTask">
                <div class="drawer-section">
                    <h3>Metadata</h3>
                    <dl class="drawer-meta">
                        <dt>ID</dt><dd x-text="selectedTask ? selectedTask.id : ''"></dd>
                        <dt>Title</dt><dd x-text="selectedTask ? (selectedTask.title || '—') : ''"></dd>
                        <dt>Status</dt><dd x-html="selectedTask ? stsBadge(selectedTask.status_v2 || selectedTask.status) : ''"></dd>
                        <dt>Created</dt><dd><span x-text="selectedTask ? fmtAgo(selectedTask.created_at) : ''" :title="selectedTask ? selectedTask.created_at : ''"></span></dd>
                        <dt>Updated</dt><dd><span x-text="selectedTask ? fmtAgo(selectedTask.updated_at) : '—'" :title="selectedTask ? (selectedTask.updated_at || '') : ''"></span></dd>
                        <dt>Parent</dt><dd x-html="selectedTask && selectedTask.parent_task_id ? entityLink('/board?task='+selectedTask.parent_task_id, '#'+selectedTask.parent_task_id) : '—'"></dd>
                        <dt>Priority</dt><dd x-text="selectedTask ? (selectedTask.priority || 50) : ''"></dd>
                        <dt>Type</dt><dd x-text="selectedTask ? (selectedTask.type || '—') : ''"></dd>
                        <dt>Agent</dt><dd x-html="selectedTask && selectedTask.agent_id ? entityLink('/agents?id='+encodeURIComponent(selectedTask.agent_id), selectedTask.agent_id.substring(0,20)) : '—'"></dd>
                        <dt>Tags</dt><dd x-text="selectedTask && selectedTask.tags ? selectedTask.tags.join(', ') : '—'"></dd>
                        <dt>Blocked</dt><dd x-html="selectedTask && selectedTask.blocked ? stsBadge('BLOCKED') : '—'"></dd>
                        <dt>Dependencies</dt><dd x-html="selectedTask && selectedTask.dependency_count ? entityLink('/dependencies?task='+selectedTask.id, selectedTask.dependency_count+' dep'+(selectedTask.dependency_count>1?'s':'')) : '—'"></dd>
                    </dl>
                </div>
                <div class="drawer-section">
                    <h3>Quick Actions</h3>
                    <div class="drawer-actions">
                        <button @click="claimSelectedTask" x-show="selectedTask && selectedTask.status === 'NOT_PICKED'">Claim</button>
                        <button @click="completeSelectedTask" x-show="selectedTask && selectedTask.status === 'IN_PROGRESS'">Complete</button>
                        <button class="danger" @click="failSelectedTask" x-show="selectedTask && selectedTask.status === 'IN_PROGRESS'">Fail</button>
                        <button @click="duplicateSelectedTask">Duplicate</button>
                        <button class="danger" @click="deleteTask(selectedTask.id); selectedTask = null">Delete</button>
                    </div>
                </div>
                <div class="drawer-section">
                    <h3>Related</h3>
                    <div style="font-size:12px;display:flex;flex-direction:column;gap:6px;">
                        <template x-if="selectedTask && tasks.filter(t => t.parent_task_id === selectedTask.id).length > 0">
                            <div>
                                <div style="color:var(--vscode-text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;">Child Tasks</div>
                                <template x-for="child in tasks.filter(t => t.parent_task_id === selectedTask.id)" :key="child.id">
                                    <div style="display:flex;align-items:center;gap:6px;padding:2px 0;">
                                        <a :href="'/board?task='+child.id" style="color:var(--vscode-accent);text-decoration:none;font-size:12px;" @click.prevent="selectedTask = child" x-text="'#'+child.id"></a>
                                        <span x-text="child.title || child.instructions?.substring(0,30)+'...' || ''" style="color:var(--vscode-text-muted);"></span>
                                        <span x-html="stsBadge(child.status_v2 || child.status)" style="margin-left:auto;"></span>
                                    </div>
                                </template>
                            </div>
                        </template>
                        <template x-if="selectedTask && selectedTask.parent_task_id">
                            <div>
                                <div style="color:var(--vscode-text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;">Parent</div>
                                <div style="display:flex;align-items:center;gap:6px;">
                                    <a :href="'/board?task='+selectedTask.parent_task_id" style="color:var(--vscode-accent);text-decoration:none;" @click.prevent="selectedTask = tasks.find(t => t.id === selectedTask.parent_task_id) || selectedTask" x-text="'#'+selectedTask.parent_task_id"></a>
                                    <span x-text="(tasks.find(t => t.id === selectedTask.parent_task_id) || {}).title || ''" style="color:var(--vscode-text-muted);"></span>
                                </div>
                            </div>
                        </template>
                        <template x-if="selectedTask && selectedTask.agent_id">
                            <div>
                                <div style="color:var(--vscode-text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;">Assigned Agent</div>
                                <a :href="'/agents?id='+encodeURIComponent(selectedTask.agent_id)" style="color:var(--vscode-accent);text-decoration:none;font-size:12px;" x-text="selectedTask.agent_id.substring(0,24)"></a>
                            </div>
                        </template>
                        <template x-if="selectedTask && !tasks.filter(t => t.parent_task_id === selectedTask?.id).length && !selectedTask.parent_task_id && !selectedTask.agent_id">
                            <div style="color:var(--vscode-text-muted);font-style:italic;">No related entities</div>
                        </template>
                    </div>
                </div>
                <div class="drawer-section">
                    <h3>Instructions</h3>
                    <pre x-text="selectedTask ? selectedTask.instructions : ''"></pre>
                </div>
                <div class="drawer-section" x-show="selectedTask && selectedTask.output">
                    <h3>Output</h3>
                    <pre x-text="selectedTask ? (selectedTask.output || '') : ''"></pre>
                </div>
            </div>
        </div>

        <!-- Create Task Modal -->
        <div class="modal-overlay"
             x-show="showModal"
             x-cloak
             x-transition:enter="fade-enter-active"
             x-transition:leave="fade-leave-active"
             @click.self="showModal = false"
             @keydown.escape.window="showModal = false">
            <div class="modal" @click.stop>
                <div class="modal-header">
                    <span class="modal-title">Create New Task</span>
                    <button class="modal-close" @click="showModal = false">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                        </svg>
                    </button>
                </div>
                <form @submit.prevent="createTask">
                    <div class="modal-body">
                        <div class="form-group">
                            <label>Title (optional)</label>
                            <input type="text" x-model="newTaskTitle" placeholder="Brief task title"
                                   style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-family:inherit;font-size:13px;padding:10px 12px;outline:none;">
                        </div>
                        <div class="form-group">
                            <label>Parent Task (optional)</label>
                            <select x-model="newTaskParentId">
                                <option value="">No parent (standalone task)</option>
                                <template x-for="task in tasks" :key="task.id">
                                    <option :value="task.id" x-text="'#' + task.id + ': ' + task.instructions.substring(0, 50) + (task.instructions.length > 50 ? '...' : '')"></option>
                                </template>
                            </select>
                        </div>
                        <div style="display:flex;gap:10px;margin-bottom:12px;">
                            <div class="form-group" style="flex:1;">
                                <label>Type (optional)</label>
                                <select x-model="newTaskType" style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:10px 12px;outline:none;cursor:pointer;">
                                    <option value="">None</option>
                                    <option value="analyze">Analyze</option>
                                    <option value="modify">Modify</option>
                                    <option value="test">Test</option>
                                    <option value="review">Review</option>
                                    <option value="doc">Doc</option>
                                    <option value="plan">Plan</option>
                                    <option value="feature">Feature</option>
                                    <option value="bug">Bug</option>
                                    <option value="chore">Chore</option>
                                    <option value="research">Research</option>
                                    <option value="refactor">Refactor</option>
                                </select>
                            </div>
                            <div class="form-group" style="flex:1;">
                                <label>Priority (optional)</label>
                                <select x-model="newTaskPriority" style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:10px 12px;outline:none;cursor:pointer;">
                                    <option value="">Normal</option>
                                    <option value="high">High</option>
                                    <option value="medium">Medium</option>
                                    <option value="low">Low</option>
                                </select>
                            </div>
                        </div>
                        <div class="form-group">
                            <label>Tags (comma-separated, optional)</label>
                            <input type="text" x-model="newTaskTags" placeholder="e.g. backend, urgent, v2"
                                   style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-family:inherit;font-size:13px;padding:10px 12px;outline:none;">
                        </div>
                        <div class="form-group">
                            <label>Dependencies (tasks this depends on)</label>
                            <select x-model="newTaskDependencies" multiple
                                    style="width:100%;min-height:60px;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:6px 8px;outline:none;">
                                <template x-for="task in tasks.filter(t => t.status !== 'complete')" :key="task.id">
                                    <option :value="task.id" x-text="'#' + task.id + ': ' + (task.title || task.instructions.substring(0, 50))"></option>
                                </template>
                            </select>
                            <small style="color:var(--vscode-descriptionForeground);font-size:11px;">Hold Ctrl/Cmd to select multiple</small>
                        </div>
                        <div class="form-group">
                            <label>Template (optional)</label>
                            <select x-model="newTaskTemplateId" @change="applyTemplate()"
                                    style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:10px 12px;outline:none;cursor:pointer;">
                                <option value="">No template</option>
                                <template x-for="tmpl in templates" :key="tmpl.id">
                                    <option :value="tmpl.id" x-text="tmpl.name + (tmpl.description ? ' — ' + tmpl.description : '')"></option>
                                </template>
                            </select>
                        </div>
                        <div class="form-group" style="display:flex;align-items:center;gap:8px;">
                            <input type="checkbox" id="requiresApproval" x-model="newTaskRequiresApproval"
                                   style="width:16px;height:16px;cursor:pointer;">
                            <label for="requiresApproval" style="margin:0;cursor:pointer;">Requires approval before completion</label>
                        </div>
                        <div class="form-group">
                            <label>Instructions</label>
                            <textarea
                                x-model="newTaskInstructions"
                                placeholder="Enter task instructions..."
                                required
                                x-ref="taskInput"></textarea>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" @click="showModal = false">Cancel</button>
                        <button type="submit" class="btn btn-primary" :disabled="!newTaskInstructions.trim()">Create Task</button>
                    </div>
                </form>
            </div>
        </div>

        <!-- Instructions Modal -->
        <div class="modal-overlay"
             x-show="showInstructionsModal"
             x-cloak
             x-transition:enter="fade-enter-active"
             x-transition:leave="fade-leave-active"
             @click.self="showInstructionsModal = false"
             @keydown.escape.window="showInstructionsModal = false">
            <div class="modal modal-wide" @click.stop>
                <div class="modal-header">
                    <span class="modal-title">Initial Copilot Instructions</span>
                    <button class="modal-close" @click="showInstructionsModal = false">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                        </svg>
                    </button>
                </div>
                <div class="modal-body">
                    <div class="code-block">
                        <div class="code-header">
                            <span class="code-label">Paste this to Copilot</span>
                            <button class="copy-btn" :class="{ 'copied': copiedInstructions }" @click="copyInstructions">
                                <svg x-show="!copiedInstructions" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                    <path fill-rule="evenodd" clip-rule="evenodd" d="M4 4l1-1h5.414L14 6.586V14l-1 1H5l-1-1V4zm9 3l-3-3H5v10h8V7zM3 1L2 2v10l1 1V2h6.414l-1-1H3z"/>
                                </svg>
                                <svg x-show="copiedInstructions" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                    <path fill-rule="evenodd" clip-rule="evenodd" d="M14.431 3.323l-8.47 10-.79-.036-3.35-4.77.818-.574 2.978 4.24 8.051-9.506.763.646z"/>
                                </svg>
                                <span x-text="copiedInstructions ? 'Copied!' : 'Copy'"></span>
                            </button>
                        </div>
                        <div class="code-content">
                            <pre x-ref="instructionsText">
                            See http://127.0.0.1:8080/instructions use curl to get the instructions and proceed with it. Do not stop when there are no new tasks!
                            </pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="header">
            <svg class="header-icon" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8.5 1a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13zm0 1a5.5 5.5 0 1 0 0 11 5.5 5.5 0 0 0 0-11zm-2 3.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5z"/>
            </svg>
            <span class="header-title">Cocopilot</span>
            <nav class="header-nav" aria-label="Primary">
                <a href="/dashboard">Dashboard</a>
                <a href="/board" class="active">Work Queue</a>
                <a href="/runs">Runs</a>
                <a href="/agents">Agents</a>
                <a href="/repo">Repo</a>
                <a href="/events-browser">Events</a>
                <a href="/settings">Settings</a>
                <div class="nav-more" x-data="{ open: false }" @click.outside="open = false">
                    <button class="nav-more-btn" @click="open = !open">More &#9662;</button>
                    <div class="nav-more-dropdown" x-show="open" x-cloak @click="open = false">
                        <a href="/graphs/tasks">Task Graph</a>
                        <a href="/dependencies">Dependencies</a>
                        <a href="/memory">Memory</a>
                        <a href="/policies">Policies</a>
                        <a href="/context-packs">Context Packs</a>
                        <a href="/audit">Audit Log</a>
                        <a href="/health">Health</a>
                    </div>
                </div>
            </nav>
            <div class="header-actions">
                <div class="workdir-input">
                    <label for="project">Project:</label>
                    <select id="project" x-model="currentProject" @change="onProjectChange" style="background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 6px 10px; font-size: 12px; min-width: 150px; cursor: pointer;">
                        <template x-for="project in projects" :key="project.id">
                            <option :value="project.id" x-text="project.name"></option>
                        </template>
                    </select>
                </div>
                <div class="workdir-input">
                    <label>Agents:</label>
                    <div class="agent-status" x-data="{ showAgentDetails: false }" @click="showAgentDetails = !showAgentDetails" style="position: relative; cursor: pointer;">
                        <div style="display: flex; align-items: center; gap: 6px; background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; padding: 6px 10px; font-size: 12px; min-width: 120px;">
                            <span x-text="agents.filter(a => a.status === 'ONLINE').length"></span>
                            <span style="color: var(--vscode-text-muted);">online</span>
                            <span style="color: var(--vscode-text-muted); font-size: 10px;">▼</span>
                        </div>
                        <div x-show="showAgentDetails" x-cloak style="position: absolute; top: 100%; right: 0; background: var(--vscode-dropdown-bg); border: 1px solid var(--vscode-border); border-radius: 4px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); z-index: 1000; min-width: 200px; max-height: 300px; overflow-y: auto;">
                            <template x-for="agent in agents" :key="agent.id">
                                <div class="agent-item" style="padding: 8px 12px; border-bottom: 1px solid var(--vscode-border); display: flex; justify-content: space-between; align-items: center;">
                                    <span x-text="agent.name" style="font-size: 12px; font-weight: 500;"></span>
                                    <span :class="'status-badge status-' + agent.status.toLowerCase()" x-text="agent.status" style="font-size: 10px; padding: 2px 6px; border-radius: 10px; font-weight: 500;"></span>
                                </div>
                            </template>
                            <div style="padding: 12px; text-align: center; color: var(--vscode-text-muted); font-size: 12px;">
                                No agents connected
                            </div>
                        </div>
                    </div>
                </div>
                <div class="workdir-input">
                    <label for="workdir">Workdir:</label>
                    <input type="text" id="workdir" x-model="workdir" @change="updateWorkdir" placeholder="/path/to/workdir">
                </div>
                <button class="header-btn" @click="showInstructionsModal = true">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14.5 2H9l-.35.15-.65.64-.65-.64L7 2H1.5l-.5.5v10l.5.5h5.29l.86.85h1.71l.85-.85h5.29l.5-.5v-10l-.5-.5zm-7 10.32l-.18-.17L7 12H2V3h4.79l.74.74-.03 8.58zM14 12H9l-.35.15-.14.13V3.7l.7-.7H14v9z"/>
                    </svg>
                    Initial Instructions
                </button>
                <button class="header-btn" @click="openModal">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14 7v1H8v6H7V8H1V7h6V1h1v6h6z"/>
                    </svg>
                    New Task
                </button>
            </div>
        </div>

        <!-- Filter bar -->
        <div class="filter-bar">
            <span class="filter-label">Filter:</span>
            <input type="text" placeholder="Search tasks... (press /)" x-model="searchQuery">
            <select x-model="filterType">
                <option value="">All types</option>
                <option value="ANALYZE">Analyze</option>
                <option value="MODIFY">Modify</option>
                <option value="TEST">Test</option>
                <option value="REVIEW">Review</option>
                <option value="DOC">Doc</option>
                <option value="PLAN">Plan</option>
                <option value="RELEASE">Release</option>
                <option value="FEATURE">Feature</option>
                <option value="BUG">Bug</option>
                <option value="CHORE">Chore</option>
                <option value="REFACTOR">Refactor</option>
                <option value="RESEARCH">Research</option>
            </select>
            <select x-model="filterPriority">
                <option value="">All priorities</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
            </select>
            <select x-model="filterAgent">
                <option value="">All agents</option>
                <template x-for="a in [...new Set(tasks.map(t => t.agent_id).filter(Boolean))]" :key="a">
                    <option :value="a" x-text="a"></option>
                </template>
            </select>
            <span class="filter-label">Sort:</span>
            <select x-model="sortBy">
                <option value="newest">Newest first</option>
                <option value="oldest">Oldest first</option>
                <option value="priority">Priority</option>
                <option value="id">By ID</option>
            </select>
            <label style="display:flex;align-items:center;gap:4px;font-size:11px;cursor:pointer;color:var(--vscode-text-muted);">
                <input type="checkbox" x-model="filterBlocked" style="margin:0;"> Blocked only
            </label>
            <select x-model="filterStatusV2" style="font-size:11px;padding:2px 6px;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);color:var(--vscode-text);border-radius:4px;">
                <option value="">All states</option>
                <option value="QUEUED">Queued</option>
                <option value="CLAIMED">Claimed</option>
                <option value="RUNNING">Running</option>
                <option value="SUCCEEDED">Succeeded</option>
                <option value="FAILED">Failed</option>
                <option value="NEEDS_REVIEW">Needs Review</option>
                <option value="CANCELLED">Cancelled</option>
            </select>
            <span class="task-count" x-text="tasks.length + ' task' + (tasks.length !== 1 ? 's' : '') + (searchQuery || filterType || filterPriority || filterAgent || filterBlocked || filterStatusV2 ? ' (' + filteredCount + ' matching)' : '')"></span>
            <span x-show="sseReconnecting" x-cloak style="color:var(--vscode-warning);font-size:11px;display:flex;align-items:center;gap:4px;" title="Reconnecting to server...">
                <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor" style="animation:spin 1s linear infinite;"><path d="M8 1a7 7 0 1 0 7 7h-1.5A5.5 5.5 0 1 1 8 2.5V1z"/></svg>
                Reconnecting
            </span>
            <span x-show="sseConnected && !sseReconnecting" style="color:var(--vscode-success);font-size:10px;" title="Connected">&#9679;</span>
        </div>

        <!-- Dashboard stats bar -->
        <div style="display:flex;gap:12px;padding:10px 20px;background:var(--vscode-sidebar);border-bottom:1px solid var(--vscode-border);flex-shrink:0;">
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-warning);" x-text="todoTasks.length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">Pending</div>
            </div>
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-info);" x-text="progressTasks.length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">In Progress</div>
            </div>
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-success);" x-text="doneTasks.length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">Done</div>
            </div>
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-accent);" x-text="agents.filter(a => a.status === 'ONLINE').length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">Agents Online</div>
            </div>
        </div>

        <div class="kanban-board">
            <!-- To Do Column -->
            <div class="kanban-column col-todo"
                 :class="{ 'drag-over': dragOverColumn === 'NOT_PICKED' }"
                 @dragover.prevent="dragOverColumn = 'NOT_PICKED'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('NOT_PICKED')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/>
                        <path d="M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16zm7-8A7 7 0 1 1 1 8a7 7 0 0 1 14 0z"/>
                    </svg>
                    <span class="column-title">To Do</span>
                    <span class="column-count" x-text="todoTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in todoTasks" :key="task.id">
                        <div class="task-card"
                             x-show="matchesSearch(task)"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id, 'task-blocked': task.blocked }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag"
                             @dblclick="selectedTask = task">
                            <div class="card-header" @click="selectedTask = task">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-title" x-show="task.title" x-text="(task.title||'').substring(0, 50) + ((task.title||'').length > 50 ? '...' : '')" :title="task.title" style="font-weight:600;color:#ccc;font-size:12px;"></span>
                                    <span class="card-preview" x-text="(task.instructions||'').substring(0, 40) + ((task.instructions||'').length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id && !task.title"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.updated_at || task.created_at)" :title="task.updated_at || task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-badges">
                                <span class="badge badge-type" x-show="task.type" x-text="task.type"></span>
                                <span class="badge" :class="task.priority >= 70 ? 'badge-priority-high' : task.priority >= 40 ? 'badge-priority-medium' : 'badge-priority-low'" x-show="task.priority" x-text="task.priority >= 70 ? 'HIGH' : task.priority >= 40 ? 'MED' : 'LOW'"></span>
                                <span class="badge badge-agent" x-show="task.agent_id" x-text="task.agent_id"></span>
                                <span class="badge badge-blocked" x-show="task.blocked">BLOCKED</span>
                                <span class="badge badge-deps" x-show="task.dependency_count" x-text="task.dependency_count + ' dep' + (task.dependency_count > 1 ? 's' : '')"></span>
                                <span class="badge badge-approval" x-show="task.requires_approval">APPROVAL</span>
                                <span class="badge badge-status-v2" :class="'badge-v2-' + (task.status_v2 || '')" x-show="task.status_v2 && task.status_v2 !== task.status" x-text="task.status_v2"></span>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="todoTasks.length === 0">
                        <p style="color:var(--vscode-text-muted);margin-bottom:8px;">Queue is empty</p>
                        <button class="btn btn-primary" @click="openModal" style="font-size:11px;padding:4px 12px;">+ Create Task</button>
                    </div>
                </div>
            </div>

            <!-- In Progress Column -->
            <div class="kanban-column col-progress"
                 :class="{ 'drag-over': dragOverColumn === 'IN_PROGRESS' }"
                 @dragover.prevent="dragOverColumn = 'IN_PROGRESS'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('IN_PROGRESS')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M8 0a8 8 0 1 0 8 8A8 8 0 0 0 8 0zm0 14a6 6 0 1 1 6-6 6 6 0 0 1-6 6z" opacity=".3"/>
                        <path d="M8 0v2a6 6 0 0 1 6 6h2a8 8 0 0 0-8-8z"/>
                    </svg>
                    <span class="column-title">In Progress</span>
                    <span class="column-count" x-text="progressTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in progressTasks" :key="task.id">
                        <div class="task-card"
                             x-show="matchesSearch(task)"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id, 'task-blocked': task.blocked }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag"
                             @dblclick="selectedTask = task">
                            <div class="card-header" @click="selectedTask = task">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-title" x-show="task.title" x-text="(task.title||'').substring(0, 50) + ((task.title||'').length > 50 ? '...' : '')" :title="task.title" style="font-weight:600;color:#ccc;font-size:12px;"></span>
                                    <span class="card-preview" x-text="(task.instructions||'').substring(0, 40) + ((task.instructions||'').length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id && !task.title"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.updated_at || task.created_at)" :title="task.updated_at || task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-badges">
                                <span class="badge badge-type" x-show="task.type" x-text="task.type"></span>
                                <span class="badge" :class="task.priority >= 70 ? 'badge-priority-high' : task.priority >= 40 ? 'badge-priority-medium' : 'badge-priority-low'" x-show="task.priority" x-text="task.priority >= 70 ? 'HIGH' : task.priority >= 40 ? 'MED' : 'LOW'"></span>
                                <span class="badge badge-agent" x-show="task.agent_id" x-text="task.agent_id"></span>
                                <span class="badge badge-blocked" x-show="task.blocked">BLOCKED</span>
                                <span class="badge badge-deps" x-show="task.dependency_count" x-text="task.dependency_count + ' dep' + (task.dependency_count > 1 ? 's' : '')"></span>
                                <span class="badge badge-approval" x-show="task.requires_approval">APPROVAL</span>
                                <span class="badge badge-status-v2" :class="'badge-v2-' + (task.status_v2 || '')" x-show="task.status_v2 && task.status_v2 !== task.status" x-text="task.status_v2"></span>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="progressTasks.length === 0">
                        <p style="color:var(--vscode-text-muted);">No tasks being worked on</p>
                    </div>
                </div>
            </div>

            <!-- Done Column -->
            <div class="kanban-column col-done"
                 :class="{ 'drag-over': dragOverColumn === 'COMPLETE' }"
                 @dragover.prevent="dragOverColumn = 'COMPLETE'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('COMPLETE')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/>
                    </svg>
                    <span class="column-title">Done</span>
                    <span class="column-count" x-text="doneTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in doneTasks" :key="task.id">
                        <div class="task-card"
                             x-show="matchesSearch(task)"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id, 'task-blocked': task.blocked }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag"
                             @dblclick="selectedTask = task">
                            <div class="card-header" @click="selectedTask = task">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-title" x-show="task.title" x-text="(task.title||'').substring(0, 50) + ((task.title||'').length > 50 ? '...' : '')" :title="task.title" style="font-weight:600;color:#ccc;font-size:12px;"></span>
                                    <span class="card-preview" x-text="(task.instructions||'').substring(0, 40) + ((task.instructions||'').length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id && !task.title"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.updated_at || task.created_at)" :title="task.updated_at || task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-badges">
                                <span class="badge badge-type" x-show="task.type" x-text="task.type"></span>
                                <span class="badge" :class="task.priority >= 70 ? 'badge-priority-high' : task.priority >= 40 ? 'badge-priority-medium' : 'badge-priority-low'" x-show="task.priority" x-text="task.priority >= 70 ? 'HIGH' : task.priority >= 40 ? 'MED' : 'LOW'"></span>
                                <span class="badge badge-agent" x-show="task.agent_id" x-text="task.agent_id"></span>
                                <span class="badge badge-status-v2" :class="'badge-v2-' + (task.status_v2 || '')" x-show="task.status_v2 && task.status_v2 !== task.status" x-text="task.status_v2"></span>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                                <div class="card-output" x-show="task.output">
                                    <div class="card-output-label">Output</div>
                                    <pre class="card-output-text" x-text="task.output"></pre>
                                </div>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="doneTasks.length === 0">
                        <p style="color:var(--vscode-text-muted);">Nothing completed yet</p>
                    </div>
                </div>
            </div>
        </div>

        <footer class="footer">
            Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a>
        </footer>

        <script>
            // Shared helpers for entity rendering
            function fmtAgo(iso){if(!iso)return '—';const s=Math.floor((Date.now()-new Date(iso))/1000);if(s<60)return s+'s ago';if(s<3600)return Math.floor(s/60)+'m ago';if(s<86400)return Math.floor(s/3600)+'h ago';return new Date(iso).toLocaleDateString();}
            function stsBadge(s){const m={SUCCEEDED:'ok',COMPLETED:'ok',DONE:'ok',FAILED:'error',CANCELLED:'warn',QUEUED:'info',CLAIMED:'warn',RUNNING:'warn',IN_PROGRESS:'warn',TODO:'info',PENDING:'info',NEEDS_REVIEW:'warn'};const c=m[s]||'muted';return '<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:.03em;background:'+(c==='ok'?'#1a3a1a':c==='error'?'#3a1919':c==='warn'?'#3a3519':c==='info'?'#19293a':'var(--vscode-input-bg)')+';color:var(--vscode-'+(c==='ok'?'success':c==='error'?'error':c==='warn'?'warning':c==='info'?'info':'text-muted')+');border:1px solid '+(c==='ok'?'#2d5a2d':c==='error'?'#5a2525':c==='warn'?'#5a5325':c==='info'?'#254a6e':'var(--vscode-border)')+';">'+((s||'').replace(/[<>&"']/g,''))+'</span>';}
            function entityLink(href,label){return '<a href="'+href+'" style="color:var(--vscode-accent);text-decoration:none;font-size:12px;" onmouseover="this.style.textDecoration=\'underline\'" onmouseout="this.style.textDecoration=\'none\'">'+((label||'').replace(/[<>&"']/g,''))+'</a>';}
            function kanbanApp() {
                return {
                    tasks: [],
                    expandedTasks: [],
                    showModal: false,
                    showInstructionsModal: false,
                    copiedInstructions: false,
                    newTaskInstructions: '',
                    newTaskParentId: '',
                    newTaskTitle: '',
                    newTaskType: '',
                    newTaskPriority: '',
                    newTaskTags: '',
                    newTaskRequiresApproval: false,
                    newTaskDependencies: [],
                    newTaskTemplateId: '',
                    templates: [],
                    draggingId: null,
                    draggingTask: null,
                    dragOverColumn: null,
                    eventSource: null,
                    workdir: '/tmp',
                    projects: [],
                    currentProject: 'proj_default',
                    agents: [],
                    toasts: [],
                    confirmDialog: { show: false, message: '', onConfirm: () => {} },
                    searchQuery: '',
                    sortBy: 'newest',
                    selectedTask: null,
                    filterType: '',
                    filterPriority: '',
                    filterAgent: '',
                    filterStatus: '',
                    filterBlocked: false,
                    filterStatusV2: '',
                    sseConnected: false,
                    sseReconnecting: false,

                    // Toast helper
                    showToast(message, type = 'info') {
                        this.toasts.push({ message, type });
                    },

                    // Confirm dialog helper
                    showConfirm(message, onConfirm) {
                        this.confirmDialog = { show: true, message, onConfirm };
                    },

                    // Search + filter
                    matchesSearch(task) {
                        if (this.filterType && (task.type || '').toUpperCase() !== this.filterType.toUpperCase()) return false;
                        if (this.filterAgent && (task.agent_id || '') !== this.filterAgent) return false;
                        if (this.filterBlocked && !task.blocked) return false;
                        if (this.filterStatusV2 && (task.status_v2 || '') !== this.filterStatusV2) return false;
                        if (this.filterPriority) {
                            const p = task.priority || 50;
                            if (this.filterPriority === 'high' && p < 70) return false;
                            if (this.filterPriority === 'medium' && (p < 40 || p >= 70)) return false;
                            if (this.filterPriority === 'low' && p >= 40) return false;
                        }
                        if (!this.searchQuery) return true;
                        const q = this.searchQuery.toLowerCase();
                        return (
                            String(task.id).includes(q) ||
                            (task.instructions || '').toLowerCase().includes(q) ||
                            (task.type || '').toLowerCase().includes(q) ||
                            (task.agent_id || '').toLowerCase().includes(q) ||
                            (task.title || '').toLowerCase().includes(q)
                        );
                    },

                    get filteredCount() {
                        return this.tasks.filter(t => this.matchesSearch(t)).length;
                    },

                    // Compute task depth based on parent chain
                    getTaskDepth(task) {
                        let depth = 0;
                        let currentParentId = task.parent_task_id;
                        while (currentParentId && depth < 3) {
                            const parent = this.tasks.find(t => t.id === currentParentId);
                            if (!parent) break;
                            depth++;
                            currentParentId = parent.parent_task_id;
                        }
                        return depth;
                    },

                    // Organize tasks hierarchically within each status column
                    getOrganizedTasks(status) {
                        const statusTasks = this.tasks.filter(t => t.status === status);
                        const rootTasks = statusTasks.filter(t => !t.parent_task_id);
                        const childrenMap = {};

                        statusTasks.forEach(t => {
                            if (t.parent_task_id) {
                                if (!childrenMap[t.parent_task_id]) childrenMap[t.parent_task_id] = [];
                                childrenMap[t.parent_task_id].push(t);
                            }
                        });

                        const result = [];
                        const addWithChildren = (task, depth = 0) => {
                            result.push({ ...task, depth });
                            (childrenMap[task.id] || []).forEach(child => addWithChildren(child, depth + 1));
                        };

                        rootTasks.forEach(t => addWithChildren(t));

                        // Also add orphaned children (whose parent is in a different status)
                        statusTasks.forEach(t => {
                            if (t.parent_task_id && !result.find(r => r.id === t.id)) {
                                result.push({ ...t, depth: this.getTaskDepth(t) });
                            }
                        });

                        return result;
                    },

                    get todoTasks() {
                        return this.getOrganizedTasks('NOT_PICKED');
                    },

                    get progressTasks() {
                        return this.getOrganizedTasks('IN_PROGRESS');
                    },

                    get doneTasks() {
                        return this.getOrganizedTasks('COMPLETE');
                    },

                    init() {
                        // Restore persisted state from localStorage
                        try {
                            const saved = JSON.parse(localStorage.getItem('cocopilot_board_state') || '{}');
                            if (saved.searchQuery) this.searchQuery = saved.searchQuery;
                            if (saved.sortBy) this.sortBy = saved.sortBy;
                            if (saved.filterType) this.filterType = saved.filterType;
                            if (saved.filterPriority) this.filterPriority = saved.filterPriority;
                            if (saved.filterAgent) this.filterAgent = saved.filterAgent;
                            if (saved.expandedTasks) this.expandedTasks = saved.expandedTasks;
                        } catch(e) {}

                        this.connectSSE();
                        this.fetchWorkdir();
                        this.fetchProjects();
                        this.fetchAgents();
                        this.fetchTemplates();
                        
                        // Refresh agents status every 30 seconds
                        setInterval(() => {
                            this.fetchAgents();
                        }, 30000);

                        // Persist state on changes
                        this.$watch('searchQuery', () => this.persistState());
                        this.$watch('sortBy', () => this.persistState());
                        this.$watch('filterType', () => this.persistState());
                        this.$watch('filterPriority', () => this.persistState());
                        this.$watch('filterAgent', () => this.persistState());
                        this.$watch('expandedTasks', () => this.persistState());

                        // Keyboard shortcuts
                        document.addEventListener('keydown', (e) => {
                            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
                            if (e.key === 'n' && !e.ctrlKey && !e.metaKey) { e.preventDefault(); this.openModal(); }
                            if (e.key === '/' && !e.ctrlKey && !e.metaKey) { e.preventDefault(); document.querySelector('.filter-bar input')?.focus(); }
                            if (e.key === 'Escape' && this.selectedTask) { this.selectedTask = null; }
                        });
                    },

                    persistState() {
                        try {
                            localStorage.setItem('cocopilot_board_state', JSON.stringify({
                                searchQuery: this.searchQuery,
                                sortBy: this.sortBy,
                                filterType: this.filterType,
                                filterPriority: this.filterPriority,
                                filterAgent: this.filterAgent,
                                expandedTasks: this.expandedTasks,
                            }));
                        } catch(e) {}
                    },

                    async fetchProjects() {
                        try {
                            const response = await fetch('/api/v2/projects');
                            const data = await response.json();
                            this.projects = data.projects || [];
                            if (this.projects.length > 0 && this.projects.find(p => p.id === 'proj_default')) {
                                this.currentProject = 'proj_default';
                            } else if (this.projects.length > 0) {
                                this.currentProject = this.projects[0].id;
                            }
                        } catch (error) {
                            console.error('Failed to fetch projects:', error);
                        }
                    },

                    async fetchAgents() {
                        try {
                            const response = await fetch('/api/v2/agents');
                            const data = await response.json();
                            this.agents = data.agents || [];
                        } catch (error) {
                            console.error('Failed to fetch agents:', error);
                            this.agents = [];
                        }
                    },

                    async fetchTemplates() {
                        try {
                            const response = await fetch('/api/v2/projects/' + encodeURIComponent(this.currentProject) + '/templates');
                            const data = await response.json();
                            this.templates = data.templates || [];
                        } catch (error) {
                            console.error('Failed to fetch templates:', error);
                            this.templates = [];
                        }
                    },

                    applyTemplate() {
                        if (!this.newTaskTemplateId) return;
                        const tmpl = this.templates.find(t => t.id === this.newTaskTemplateId);
                        if (!tmpl) return;
                        if (!this.newTaskInstructions.trim() && tmpl.instructions) {
                            this.newTaskInstructions = tmpl.instructions;
                        }
                        if (!this.newTaskTitle.trim() && tmpl.name) {
                            this.newTaskTitle = tmpl.name;
                        }
                        if (!this.newTaskType && tmpl.default_type) {
                            this.newTaskType = tmpl.default_type.toLowerCase();
                        }
                        if (!this.newTaskPriority && tmpl.default_priority > 0) {
                            if (tmpl.default_priority >= 70) this.newTaskPriority = 'high';
                            else if (tmpl.default_priority >= 40) this.newTaskPriority = 'medium';
                            else this.newTaskPriority = 'low';
                        }
                        if (!this.newTaskTags.trim() && tmpl.default_tags && tmpl.default_tags.length) {
                            this.newTaskTags = tmpl.default_tags.join(', ');
                        }
                    },

                    onProjectChange() {
                        // Reconnect SSE with new project filter
                        this.connectSSE();
                        this.fetchTemplates();
                    },

                    formatTime(iso) {
                        if (!iso) return '';
                        const d = new Date(iso);
                        const now = new Date();
                        const diff = now - d;
                        if (diff < 60000) return 'just now';
                        if (diff < 3600000) return Math.floor(diff / 60000) + 'm ago';
                        if (diff < 86400000) return Math.floor(diff / 3600000) + 'h ago';
                        if (diff < 604800000) return Math.floor(diff / 86400000) + 'd ago';
                        return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                    },

                    async fetchWorkdir() {
                        try {
                            const response = await fetch('/api/workdir');
                            const data = await response.json();
                            this.workdir = data.workdir;
                        } catch (error) {
                            console.error('Failed to fetch workdir:', error);
                        }
                    },

                    async updateWorkdir() {
                        const formData = new FormData();
                        formData.append('workdir', this.workdir);
                        await fetch('/set-workdir', {
                            method: 'POST',
                            body: formData
                        });
                    },

                    connectSSE() {
                        if (this.eventSource) {
                            this.eventSource.close();
                        }
                        this.sseReconnecting = true;
                        const url = this.currentProject ? '/events?project_id=' + encodeURIComponent(this.currentProject) : '/events';
                        this.eventSource = new EventSource(url);

                        this.eventSource.onopen = () => {
                            this.sseConnected = true;
                            this.sseReconnecting = false;
                        };

                        // SSE sends named "tasks" events
                        this.eventSource.addEventListener('tasks', (event) => {
                            try {
                                this.tasks = JSON.parse(event.data);
                            } catch (e) {
                                console.error('Failed to parse SSE task data:', e);
                            }
                        });

                        this.eventSource.onerror = () => {
                            this.sseConnected = false;
                            this.sseReconnecting = true;
                            this.eventSource.close();
                            // Reconnect after 3 seconds
                            setTimeout(() => this.connectSSE(), 3000);
                        };
                    },

                    openModal() {
                        this.showModal = true;
                        this.$nextTick(() => this.$refs.taskInput.focus());
                    },

                    copyInstructions() {
                        const text = this.$refs.instructionsText.textContent;
                        navigator.clipboard.writeText(text).then(() => {
                            this.copiedInstructions = true;
                            setTimeout(() => {
                                this.copiedInstructions = false;
                            }, 2000);
                        });
                    },

                    async createTask() {
                        if (!this.newTaskInstructions.trim()) return;

                        const payload = {
                            instructions: this.newTaskInstructions,
                            project_id: this.currentProject
                        };
                        if (this.newTaskTitle.trim()) payload.title = this.newTaskTitle.trim();
                        if (this.newTaskType) payload.type = this.newTaskType.toUpperCase();
                        if (this.newTaskPriority === 'high') payload.priority = 80;
                        else if (this.newTaskPriority === 'medium') payload.priority = 50;
                        else if (this.newTaskPriority === 'low') payload.priority = 20;
                        if (this.newTaskParentId) payload.parent_task_id = parseInt(this.newTaskParentId, 10);
                        if (this.newTaskTags.trim()) payload.tags = this.newTaskTags.split(',').map(t => t.trim()).filter(Boolean);
                        if (this.newTaskRequiresApproval) payload.requires_approval = true;
                        if (this.newTaskDependencies.length) payload.depends_on = this.newTaskDependencies.map(Number);
                        if (this.newTaskTemplateId) payload.template_id = this.newTaskTemplateId;

                        try {
                            const url = '/api/v2/projects/' + encodeURIComponent(this.currentProject) + '/tasks';
                            const response = await fetch(url, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify(payload)
                            });
                            if (!response.ok) {
                                const data = await response.json().catch(() => ({}));
                                const msg = (data.error && data.error.message) || response.statusText;
                                this.showToast('Failed to create task: ' + msg, 'error');
                                return;
                            }
                        } catch (error) {
                            this.showToast('Failed to create task: ' + error.message, 'error');
                            return;
                        }

                        this.newTaskInstructions = '';
                        this.newTaskParentId = '';
                        this.newTaskTitle = '';
                        this.newTaskType = '';
                        this.newTaskPriority = '';
                        this.newTaskTags = '';
                        this.newTaskRequiresApproval = false;
                        this.newTaskDependencies = [];
                        this.newTaskTemplateId = '';
                        this.showModal = false;
                        this.showToast('Task created', 'success');
                    },

                    toggleExpand(taskId) {
                        const index = this.expandedTasks.indexOf(taskId);
                        if (index === -1) {
                            this.expandedTasks.push(taskId);
                        } else {
                            this.expandedTasks.splice(index, 1);
                        }
                    },

                    startDrag(task) {
                        this.draggingId = task.id;
                        this.draggingTask = task;
                    },

                    endDrag() {
                        this.draggingId = null;
                        this.draggingTask = null;
                        this.dragOverColumn = null;
                    },

                    async dropTask(newStatus) {
                        if (!this.draggingTask || this.draggingTask.status === newStatus) {
                            this.endDrag();
                            return;
                        }

                        const formData = new FormData();
                        formData.append('task_id', this.draggingTask.id);
                        formData.append('status', newStatus);

                        const originalStatus = this.draggingTask.status;
                        // Optimistic update
                        this.draggingTask.status = newStatus;

                        try {
                            const response = await fetch('/update-status', {
                                method: 'POST',
                                body: formData
                            });
                            if (!response.ok) {
                                this.draggingTask.status = originalStatus;
                            }
                        } catch (error) {
                            this.draggingTask.status = originalStatus;
                        }

                        this.endDrag();
                    },

                    async deleteTask(taskId) {
                        this.showConfirm('Are you sure you want to delete task #' + taskId + '?', async () => {
                            const formData = new FormData();
                            formData.append('task_id', taskId);

                            try {
                                const response = await fetch('/delete', {
                                    method: 'POST',
                                    body: formData
                                });
                                if (!response.ok) {
                                    const text = await response.text();
                                    this.showToast('Failed to delete task: ' + (text || response.statusText), 'error');
                                } else {
                                    this.showToast('Task #' + taskId + ' deleted', 'success');
                                }
                            } catch (error) {
                                this.showToast('Failed to delete task: ' + error.message, 'error');
                            }
                        });
                    },

                    async claimSelectedTask() {
                        if (!this.selectedTask) return;
                        try {
                            const resp = await fetch('/api/v2/tasks/' + this.selectedTask.id + '/claim', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ agent_id: 'manual' })
                            });
                            if (resp.ok) { this.showToast('Task claimed', 'success'); this.selectedTask = null; }
                            else { const d = await resp.json().catch(() => ({})); this.showToast('Claim failed: ' + ((d.error && d.error.message) || resp.statusText), 'error'); }
                        } catch (e) { this.showToast('Claim failed: ' + e.message, 'error'); }
                    },

                    async completeSelectedTask() {
                        if (!this.selectedTask) return;
                        try {
                            const resp = await fetch('/api/v2/tasks/' + this.selectedTask.id + '/complete', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ output: 'Manually completed' })
                            });
                            if (resp.ok) { this.showToast('Task completed', 'success'); this.selectedTask = null; }
                            else { const d = await resp.json().catch(() => ({})); this.showToast('Complete failed: ' + ((d.error && d.error.message) || resp.statusText), 'error'); }
                        } catch (e) { this.showToast('Complete failed: ' + e.message, 'error'); }
                    },

                    async failSelectedTask() {
                        if (!this.selectedTask) return;
                        try {
                            const resp = await fetch('/api/v2/tasks/' + this.selectedTask.id + '/fail', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ error: 'Manually failed' })
                            });
                            if (resp.ok) { this.showToast('Task failed', 'success'); this.selectedTask = null; }
                            else { const d = await resp.json().catch(() => ({})); this.showToast('Fail failed: ' + ((d.error && d.error.message) || resp.statusText), 'error'); }
                        } catch (e) { this.showToast('Fail failed: ' + e.message, 'error'); }
                    },

                    async duplicateSelectedTask() {
                        if (!this.selectedTask) return;
                        const t = this.selectedTask;
                        const payload = {
                            instructions: t.instructions,
                            project_id: this.currentProject,
                        };
                        if (t.title) payload.title = t.title + ' (copy)';
                        if (t.type) payload.type = t.type;
                        if (t.priority) payload.priority = t.priority;
                        if (t.tags) payload.tags = t.tags;
                        try {
                            const url = '/api/v2/projects/' + encodeURIComponent(this.currentProject) + '/tasks';
                            const resp = await fetch(url, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify(payload)
                            });
                            if (resp.ok) { this.showToast('Task duplicated', 'success'); }
                            else { this.showToast('Duplicate failed', 'error'); }
                        } catch (e) { this.showToast('Duplicate failed: ' + e.message, 'error'); }
                    }
                };
            }
        </script>
    </body>
</html>
`
