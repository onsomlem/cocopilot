package server

import (
"fmt"
"net/http"
"strings"
)

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

	// Health checks are generated server-side so new endpoints only need one update.
	type healthCheck struct {
		Name     string
		URL      string
		Validate string
	}
	checks := []healthCheck{
		{"Health API", "/api/v2/health", "r=>r.ok"},
		{"Metrics API", "/api/v2/metrics", "r=>r.totals!==undefined"},
		{"Version API", "/api/v2/version", "r=>r.version!==undefined"},
		{"Projects List", "/api/v2/projects", "r=>Array.isArray(r)"},
		{"Tasks List", "/api/v2/tasks", "r=>Array.isArray(r.tasks)"},
		{"Agents List", "/api/v2/agents", "r=>Array.isArray(r)"},
		{"Events List", "/api/v2/events", "r=>Array.isArray(r.events)"},
		{"Runs List", "/api/v2/runs", "r=>Array.isArray(r)"},
		{"Migrations", "/api/v2/metrics", "r=>r.schema_version&&parseInt(r.schema_version)>=17"},
		{"v1 Compat (task)", "/task", "(r,raw)=>raw.status===200||raw.status===204"},
	}
	b.WriteString("const checks=[")
	for i, c := range checks {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "{name:'%s',url:'%s',validate:%s}", c.Name, c.URL, c.Validate)
	}
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

