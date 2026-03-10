package server

import (
	"html"
	"net/http"
	"strings"
)

// planningPanelHandler serves the Planning Diagnostics page at /planning.
func planningPanelHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(planningPanelPage()))
}

func planningPanelPage() string {
	var b strings.Builder
	b.WriteString(subPageHead("Planning"))
	b.WriteString(`<style>
.plan-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
@media (max-width: 900px) { .plan-grid { grid-template-columns: 1fr; } }
.plan-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 16px; }
.plan-card h3 { margin: 0 0 12px; font-size: 14px; color: var(--text); }
.plan-stat { display: flex; justify-content: space-between; padding: 6px 0; border-bottom: 1px solid var(--border); font-size: 13px; }
.plan-stat:last-child { border-bottom: none; }
.plan-stat .label { color: var(--text-muted); }
.plan-stat .value { color: var(--text); font-weight: 600; }
.ws-list { list-style: none; padding: 0; margin: 0; }
.ws-item { background: var(--bg); border: 1px solid var(--border); border-radius: 6px; padding: 12px; margin-bottom: 8px; }
.ws-item .ws-title { font-weight: 600; font-size: 13px; }
.ws-item .ws-meta { font-size: 11px; color: var(--text-muted); margin-top: 4px; }
.ws-score { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.ws-score.high { background: rgba(34,197,94,0.15); color: #22c55e; }
.ws-score.mid { background: rgba(234,179,8,0.15); color: #eab308; }
.ws-score.low { background: rgba(239,68,68,0.15); color: #ef4444; }
.tag-badge { display: inline-block; background: var(--bg); border: 1px solid var(--border); border-radius: 4px; padding: 2px 6px; font-size: 11px; margin: 2px; }
.mode-badge { display: inline-block; padding: 4px 12px; border-radius: 12px; font-size: 12px; font-weight: 600; text-transform: uppercase; }
.mode-standard { background: rgba(59,130,246,0.15); color: #3b82f6; }
.mode-focused { background: rgba(168,85,247,0.15); color: #a855f7; }
.mode-recovery { background: rgba(239,68,68,0.15); color: #ef4444; }
.mode-maintenance { background: rgba(234,179,8,0.15); color: #eab308; }
.dec-item { display: flex; gap: 8px; padding: 8px 0; border-bottom: 1px solid var(--border); font-size: 12px; }
.dec-item:last-child { border-bottom: none; }
.dec-type { font-weight: 600; min-width: 70px; }
.mnf-item { background: var(--bg); border-left: 3px solid var(--warning); padding: 8px 12px; margin-bottom: 6px; border-radius: 0 6px 6px 0; font-size: 13px; }
.plan-tabs { display: flex; gap: 0; margin-bottom: 16px; border-bottom: 2px solid var(--border); }
.plan-tab { padding: 8px 16px; cursor: pointer; font-size: 13px; color: var(--text-muted); border-bottom: 2px solid transparent; margin-bottom: -2px; transition: all .15s; }
.plan-tab:hover { color: var(--text); }
.plan-tab.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 600; }
.tab-content { display: none; }
.tab-content.active { display: block; }
.empty-state { text-align: center; padding: 32px; color: var(--text-muted); font-size: 13px; }
.cycle-item { background: var(--bg); border: 1px solid var(--border); border-radius: 6px; padding: 12px; margin-bottom: 8px; }
.cycle-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.cycle-num { font-weight: 600; font-size: 14px; }
.cycle-tasks-created { color: var(--success); font-size: 12px; }
.cycle-failures { color: var(--error); font-size: 12px; }
</style>`)

	b.WriteString(`<div class="plan-tabs">
  <div class="plan-tab active" onclick="switchTab('overview')">Overview</div>
  <div class="plan-tab" onclick="switchTab('workstreams')">Workstreams</div>
  <div class="plan-tab" onclick="switchTab('decisions')">Decisions</div>
  <div class="plan-tab" onclick="switchTab('cycles')">Cycle History</div>
  <div class="plan-tab" onclick="switchTab('prompts')">Prompts</div>
</div>`)

	// Overview tab
	b.WriteString(`<div id="tab-overview" class="tab-content active">
  <div id="overview-loading" class="empty-state">Loading...</div>
  <div id="overview-content" style="display:none;"></div>
</div>`)

	// Workstreams tab
	b.WriteString(`<div id="tab-workstreams" class="tab-content">
  <div id="ws-loading" class="empty-state">Loading...</div>
  <div id="ws-content" style="display:none;"></div>
</div>`)

	// Decisions tab
	b.WriteString(`<div id="tab-decisions" class="tab-content">
  <div id="dec-loading" class="empty-state">Loading...</div>
  <div id="dec-content" style="display:none;"></div>
</div>`)

	// Cycles tab
	b.WriteString(`<div id="tab-cycles" class="tab-content">
  <div id="cycles-loading" class="empty-state">Loading...</div>
  <div id="cycles-content" style="display:none;"></div>
</div>`)

	// Prompts tab
	b.WriteString(`<div id="tab-prompts" class="tab-content">
  <div id="prompts-loading" class="empty-state">Loading...</div>
  <div id="prompts-content" style="display:none;"></div>
</div>`)

	b.WriteString(`<script>
const PID = localStorage.getItem('coco_project') || 'proj_default';
const BASE = '/api/v2/projects/' + PID;

function switchTab(name) {
  document.querySelectorAll('.plan-tab').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
  event.target.classList.add('active');
  document.getElementById('tab-' + name).classList.add('active');
}

function scoreClass(s) { return s >= 0.7 ? 'high' : s >= 0.4 ? 'mid' : 'low'; }

function modeBadge(mode) {
  return '<span class="mode-badge mode-' + (mode||'standard') + '">' + (mode||'standard') + '</span>';
}

async function loadOverview() {
  try {
    const res = await fetch(BASE + '/planning');
    if (!res.ok) { document.getElementById('overview-loading').textContent = 'No planning state found. Seed planning first.'; return; }
    const data = await res.json();
    const ps = data.planning_state || {};
    const ws = data.workstreams || [];

    let html = '<div class="plan-grid">';
    html += '<div class="plan-card"><h3>Planning State</h3>';
    html += '<div class="plan-stat"><span class="label">Mode</span><span class="value">' + modeBadge(ps.planning_mode) + '</span></div>';
    html += '<div class="plan-stat"><span class="label">Cycles</span><span class="value">' + (ps.cycle_count||0) + '</span></div>';
    html += '<div class="plan-stat"><span class="label">Last Cycle</span><span class="value">' + (ps.last_cycle_at||'never') + '</span></div>';
    html += '<div class="plan-stat"><span class="label">Workstreams</span><span class="value">' + ws.length + '</span></div>';
    html += '</div>';

    html += '<div class="plan-card"><h3>Goals</h3>';
    if (ps.goals && ps.goals.length) {
      ps.goals.forEach(g => { html += '<div class="plan-stat"><span class="label">' + esc(g) + '</span></div>'; });
    } else { html += '<div class="empty-state">No goals defined</div>'; }
    html += '</div>';
    html += '</div>';

    // Must Not Forget
    html += '<div class="plan-card" style="margin-bottom:16px;"><h3>Must Not Forget</h3>';
    if (ps.must_not_forget && ps.must_not_forget.length) {
      ps.must_not_forget.forEach(m => { html += '<div class="mnf-item">' + esc(m) + '</div>'; });
    } else { html += '<div class="empty-state">No items</div>'; }
    html += '</div>';

    // Summaries
    html += '<div class="plan-grid">';
    html += '<div class="plan-card"><h3>Recon Summary</h3><p style="font-size:13px;color:var(--text-muted);">' + esc(ps.recon_summary||'Not yet run') + '</p></div>';
    html += '<div class="plan-card"><h3>Planner Summary</h3><p style="font-size:13px;color:var(--text-muted);">' + esc(ps.planner_summary||'Not yet run') + '</p></div>';
    html += '</div>';

    document.getElementById('overview-content').innerHTML = html;
    document.getElementById('overview-content').style.display = '';
    document.getElementById('overview-loading').style.display = 'none';
  } catch(e) {
    document.getElementById('overview-loading').textContent = 'Error: ' + e.message;
  }
}

async function loadWorkstreams() {
  try {
    const res = await fetch(BASE + '/workstreams');
    if (!res.ok) { document.getElementById('ws-loading').textContent = 'Failed to load workstreams'; return; }
    const data = await res.json();
    const wsList = data.workstreams || [];

    if (!wsList.length) {
      document.getElementById('ws-content').innerHTML = '<div class="empty-state">No workstreams yet</div>';
    } else {
      let html = '<ul class="ws-list">';
      wsList.forEach(ws => {
        html += '<li class="ws-item">';
        html += '<div style="display:flex;justify-content:space-between;align-items:center;">';
        html += '<span class="ws-title">' + esc(ws.title) + '</span>';
        html += '<span class="tag-badge">' + (ws.status||'active') + '</span>';
        html += '</div>';
        html += '<div class="ws-meta">' + esc(ws.description||'') + '</div>';
        html += '<div class="ws-meta" style="margin-top:6px;">';
        html += 'Continuity: <span class="ws-score ' + scoreClass(ws.continuity_score||0) + '">' + (ws.continuity_score||0).toFixed(2) + '</span> ';
        html += 'Urgency: <span class="ws-score ' + scoreClass(ws.urgency_score||0) + '">' + (ws.urgency_score||0).toFixed(2) + '</span> ';
        html += 'Tasks: ' + ((ws.related_task_ids||[]).length);
        html += '</div>';
        if (ws.what_next) {
          html += '<div class="ws-meta" style="margin-top:4px;"><strong>Next:</strong> ' + esc(ws.what_next) + '</div>';
        }
        html += '</li>';
      });
      html += '</ul>';
      document.getElementById('ws-content').innerHTML = html;
    }
    document.getElementById('ws-content').style.display = '';
    document.getElementById('ws-loading').style.display = 'none';
  } catch(e) {
    document.getElementById('ws-loading').textContent = 'Error: ' + e.message;
  }
}

async function loadDecisions() {
  try {
    const res = await fetch(BASE + '/planning/decisions?limit=50');
    if (!res.ok) { document.getElementById('dec-loading').textContent = 'Failed to load decisions'; return; }
    const data = await res.json();
    const decs = data.decisions || [];

    if (!decs.length) {
      document.getElementById('dec-content').innerHTML = '<div class="empty-state">No decisions recorded yet</div>';
    } else {
      let html = '<div class="plan-card">';
      decs.forEach(d => {
        html += '<div class="dec-item">';
        html += '<span class="dec-type">' + esc(d.decision_type||'') + '</span>';
        html += '<span style="flex:1;">' + esc(d.subject||'') + '</span>';
        html += '<span style="color:var(--text-muted);min-width:80px;">' + esc(d.stage||'') + '</span>';
        html += '</div>';
      });
      html += '</div>';
      document.getElementById('dec-content').innerHTML = html;
    }
    document.getElementById('dec-content').style.display = '';
    document.getElementById('dec-loading').style.display = 'none';
  } catch(e) {
    document.getElementById('dec-loading').textContent = 'Error: ' + e.message;
  }
}

async function loadCycles() {
  try {
    const res = await fetch(BASE + '/planning/cycles?limit=20');
    if (!res.ok) { document.getElementById('cycles-loading').textContent = 'Failed to load cycles'; return; }
    const data = await res.json();
    const cycles = data.cycles || [];

    if (!cycles.length) {
      document.getElementById('cycles-content').innerHTML = '<div class="empty-state">No planning cycles yet</div>';
    } else {
      let html = '';
      cycles.forEach(c => {
        html += '<div class="cycle-item">';
        html += '<div class="cycle-header">';
        html += '<span class="cycle-num">Cycle #' + c.cycle_number + '</span>';
        html += '<span>';
        const tc = c.tasks_created ? c.tasks_created.length : 0;
        if (tc > 0) html += '<span class="cycle-tasks-created">' + tc + ' tasks created</span> ';
        const sf = c.stage_failures ? c.stage_failures.length : 0;
        if (sf > 0) html += '<span class="cycle-failures">' + sf + ' failures</span>';
        if (tc === 0 && sf === 0) html += '<span style="color:var(--text-muted);font-size:12px;">clean</span>';
        html += '</span></div>';
        html += '<div style="font-size:12px;color:var(--text-muted);">';
        html += 'Mode: ' + (c.planning_mode||'standard') + ' | Coherence: ' + (c.coherence_score||0).toFixed(2);
        html += ' | ' + (c.started_at||'');
        html += '</div>';
        html += '</div>';
      });
      document.getElementById('cycles-content').innerHTML = html;
    }
    document.getElementById('cycles-content').style.display = '';
    document.getElementById('cycles-loading').style.display = 'none';
  } catch(e) {
    document.getElementById('cycles-loading').textContent = 'Error: ' + e.message;
  }
}

async function loadPrompts() {
  try {
    const res = await fetch(BASE + '/prompts?active_only=false');
    if (!res.ok) { document.getElementById('prompts-loading').textContent = 'Failed to load prompts'; return; }
    const data = await res.json();
    const prompts = data.prompts || [];

    if (!prompts.length) {
      document.getElementById('prompts-content').innerHTML = '<div class="empty-state">No prompt templates registered</div>';
    } else {
      let html = '<div class="plan-card">';
      const byRole = {};
      prompts.forEach(p => { (byRole[p.role] = byRole[p.role]||[]).push(p); });
      Object.keys(byRole).sort().forEach(role => {
        html += '<h3 style="margin-top:12px;">' + esc(role) + '</h3>';
        byRole[role].forEach(p => {
          html += '<div class="dec-item">';
          html += '<span class="dec-type">v' + p.version + '</span>';
          html += '<span style="flex:1;">' + esc(p.name) + '</span>';
          html += '<span class="tag-badge">' + (p.is_active ? 'active' : 'inactive') + '</span>';
          html += '</div>';
        });
      });
      html += '</div>';
      document.getElementById('prompts-content').innerHTML = html;
    }
    document.getElementById('prompts-content').style.display = '';
    document.getElementById('prompts-loading').style.display = 'none';
  } catch(e) {
    document.getElementById('prompts-loading').textContent = 'Error: ' + e.message;
  }
}

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s||'';
  return d.innerHTML;
}

// Load all on page init
loadOverview();
loadWorkstreams();
loadDecisions();
loadCycles();
loadPrompts();
</script>`)

	_ = html.EscapeString // ensure import
	b.WriteString(subPageFoot())
	return b.String()
}
