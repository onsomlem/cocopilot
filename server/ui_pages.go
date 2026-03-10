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
	b.WriteString(`const todo=tasks.filter(t=>cocoStatus.isClaimable(t.status_v2||t.status)).length;`)
	b.WriteString(`const inProg=tasks.filter(t=>{const s=(t.status_v2||t.status||'').toUpperCase();return s==='IN_PROGRESS'||s==='RUNNING'||s==='CLAIMED';}).length;`)
	b.WriteString(`const done=tasks.filter(t=>cocoStatus.isDone(t.status_v2||t.status)&&(t.status_v2||t.status||'').toUpperCase()!=='FAILED').length;`)
	b.WriteString(`const failed=tasks.filter(t=>(t.status_v2||t.status||'').toUpperCase()==='FAILED').length;`)
	b.WriteString(`const blocked=tasks.filter(t=>t.is_blocked).length;`)
	b.WriteString(`const onlineAgents=agents.filter(a=>{const s=cocoStatus.agentLabel(a.status);return s==='ONLINE';}).length;`)
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
		`<link rel="stylesheet" href="/static/css/app.css">` +
		`<script src="/static/js/coco.js"></script>` +
		`<style>` +
		// Inline shared CSS as fallback (embedded FS guarantees availability)
		uiSharedCSS() +
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


func uiPlaceholderHandler(title string, subtitle string, details []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, uiPlaceholderPage(title, subtitle, details))
	}
}

