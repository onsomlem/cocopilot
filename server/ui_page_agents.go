package server

import (
"fmt"
"net/http"
"strings"
)

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
	b.WriteString(`const dotCls=statusLower==='online'?'online':statusLower==='busy'?'online':statusLower==='idle'?'stale':'offline';`)
	b.WriteString(`const caps=Array.isArray(agent.capabilities)?agent.capabilities:[];`)
	b.WriteString(`const meta=agent.metadata||{};`)
	b.WriteString(`let capsHtml='';if(caps.length>0){capsHtml='<div class="agent-caps">'+caps.map(c=>'<span class="agent-cap-tag">'+escapeHtml(c)+'</span>').join('')+'</div>';}`)
	b.WriteString(`const card=document.createElement('div');card.className='agent-card';`)
	b.WriteString(`card.innerHTML='<div class="agent-card-header"><div><span class="agent-status-dot '+dotCls+'"></span><span class="agent-card-name">'+escapeHtml(agent.name||agent.id)+'</span></div><span style="font-size:11px;padding:2px 8px;border-radius:10px;background:'+(dotCls==='online'?'#1a3a1a':'#333')+';color:'+(dotCls==='online'?'#4caf50':'#888')+';">'+escapeHtml(agent.status||'unknown')+'</span></div>'`)
	b.WriteString(`+'<div class="agent-card-id">'+escapeHtml(agent.id)+'</div>'`)
	b.WriteString(`+'<div class="agent-meta" style="margin-top:12px;">'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Heartbeat</div><div class="agent-meta-value">'+hb.label+'</div><div class="heartbeat-bar"><div class="heartbeat-fill '+hb.cls+'" style="width:'+hb.pct+'%"></div></div></div>'`)
	b.WriteString(`+'<div><div class="agent-meta-label">Registered</div><div class="agent-meta-value">'+formatAgo(agent.registered_at)+'</div></div>'`)
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

