package server

// ui_framework.go — Shared UI primitives: CSS, JS helpers, nav, page shells.
// All pages should use pageShell() instead of subPageHead()/subPageFoot() directly.

import (
	"html"
	"strings"
)

// uiSharedCSS returns the core CSS used by all pages.
func uiSharedCSS() string {
	return `:root{--bg:#1e1e1e;--surface:#252526;--input-bg:#3c3c3c;--border:#3c3c3c;--text:#cccccc;--text-muted:#858585;--accent:#0078d4;--accent-hover:#1c8ae8;--success:#89d185;--warning:#cca700;--info:#3794ff;--error:#f48771;--dropdown-bg:#313131;}` +
		`*{box-sizing:border-box;margin:0;padding:0;}` +
		`body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;font-size:13px;background:var(--bg);color:var(--text);min-height:100vh;}` +
		// Nav
		`.top-nav{background:var(--surface);border-bottom:1px solid var(--border);padding:10px 20px;display:flex;align-items:center;gap:12px;flex-wrap:wrap;}` +
		`.top-nav .nav-icon{color:var(--accent);}` +
		`.top-nav .nav-title{font-size:14px;font-weight:600;color:var(--text);text-decoration:none;}` +
		`.top-nav .nav-title:hover{color:#fff;}` +
		`.top-nav nav{display:flex;align-items:center;gap:4px;font-size:12px;}` +
		`.top-nav nav a{color:var(--text-muted);text-decoration:none;padding:4px 8px;border-radius:4px;transition:all .15s;white-space:nowrap;}` +
		`.top-nav nav a:hover,.top-nav nav a.active{color:var(--text);background:var(--input-bg);}` +
		`.nav-sep{color:var(--border);font-size:14px;user-select:none;margin:0 2px;}` +
		`.nav-more{position:relative;display:inline-block;}` +
		`.nav-more-btn{color:var(--text-muted);background:none;border:1px solid transparent;padding:4px 8px;border-radius:4px;cursor:pointer;font-size:12px;transition:all .15s;}` +
		`.nav-more-btn:hover{color:var(--text);background:var(--input-bg);}` +
		`.nav-dropdown{display:none;position:absolute;top:100%;right:0;background:var(--dropdown-bg);border:1px solid var(--border);border-radius:6px;padding:4px 0;min-width:160px;z-index:100;box-shadow:0 4px 12px rgba(0,0,0,.4);}` +
		`.nav-more:hover .nav-dropdown{display:block;}` +
		`.nav-dropdown a{display:block;padding:6px 14px;color:var(--text-muted);text-decoration:none;font-size:12px;transition:all .1s;}` +
		`.nav-dropdown a:hover{color:var(--text);background:var(--input-bg);}` +
		// Page body
		`.page-body{max-width:1100px;margin:0 auto;padding:24px 20px;}` +
		`.card{background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:20px;margin-bottom:16px;}` +
		`h1{font-size:20px;margin:0 0 6px;color:#e0e0e0;}` +
		`h2{font-size:15px;margin:18px 0 8px;color:#e0e0e0;border-bottom:1px solid var(--border);padding-bottom:6px;}` +
		`p{margin:0 0 14px;color:var(--text-muted);font-size:13px;}` +
		`.muted{color:var(--text-muted);}` +
		// Toolbar
		`.toolbar{display:flex;flex-wrap:wrap;align-items:center;gap:8px;margin-bottom:14px;font-size:12px;}` +
		`.toolbar .spacer{flex:1;}` +
		// Buttons
		`.btn{background:var(--input-bg);border:1px solid #505050;color:var(--text);padding:6px 10px;border-radius:4px;cursor:pointer;font-size:12px;transition:all .15s;}` +
		`.btn:hover{background:var(--accent);border-color:var(--accent);color:#fff;}` +
		`.btn:disabled{opacity:.5;cursor:default;pointer-events:none;}` +
		`.btn-primary{background:var(--accent);border-color:var(--accent);color:#fff;}` +
		`.btn-primary:hover{background:var(--accent-hover);}` +
		`.btn-danger{background:#5a1d1d;border-color:#6e2c2c;color:#f48771;}` +
		`.btn-danger:hover{background:#6e2c2c;}` +
		// Inputs
		`.input,.textarea,.select{background:var(--input-bg);border:1px solid #505050;color:var(--text);border-radius:4px;padding:6px 8px;font-size:12px;}` +
		`.input:focus,.textarea:focus,.select:focus{border-color:var(--accent);outline:none;}` +
		`.textarea{min-height:100px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;}` +
		`.field{display:flex;align-items:center;gap:6px;font-size:12px;color:#b0b0b0;}` +
		`.field-stack{display:flex;flex-direction:column;gap:4px;font-size:12px;color:#b0b0b0;}` +
		`.field-stack label{font-size:11px;color:var(--text-muted);text-transform:uppercase;letter-spacing:.03em;}` +
		// Tables
		`table{width:100%;border-collapse:collapse;font-size:12px;}` +
		`th,td{padding:8px;border-bottom:1px solid var(--border);text-align:left;}` +
		`th{color:var(--accent);font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.05em;}` +
		`tbody tr:hover{background:#2a2d2e;}` +
		`tbody tr{cursor:pointer;}` +
		// Stat cards
		`.stat-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px;margin-bottom:14px;}` +
		`.stat{background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:12px;text-align:center;}` +
		`.stat .label{font-size:11px;color:var(--text-muted);text-transform:uppercase;letter-spacing:.05em;margin-bottom:4px;}` +
		`.stat .value{font-size:22px;font-weight:700;color:var(--text);}` +
		`.stat.ok .value{color:var(--success);} .stat.warn .value{color:var(--warning);} .stat.error .value{color:var(--error);}` +
		`.stat.info .value{color:var(--info);}` +
		`.stat.clickable{cursor:pointer;transition:border-color .15s;}.stat.clickable:hover{border-color:var(--accent);}` +
		// List/detail layout
		`.list-detail{display:grid;grid-template-columns:1fr 380px;gap:16px;min-height:400px;}` +
		`.list-detail.no-detail{grid-template-columns:1fr;}` +
		`.detail-pane{background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:16px;overflow-y:auto;max-height:calc(100vh - 140px);position:sticky;top:80px;}` +
		`.detail-pane h3{font-size:14px;color:#e0e0e0;margin:0 0 12px;display:flex;align-items:center;justify-content:space-between;}` +
		`.detail-pane .close-btn{background:none;border:none;color:var(--text-muted);cursor:pointer;font-size:16px;padding:2px 6px;border-radius:4px;}` +
		`.detail-pane .close-btn:hover{background:var(--input-bg);color:var(--text);}` +
		`.detail-section{margin-bottom:14px;}` +
		`.detail-section .ds-label{font-size:11px;color:var(--text-muted);text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;}` +
		`.detail-section .ds-value{font-size:12px;color:var(--text);}` +
		`.detail-section .ds-value a{color:var(--accent);text-decoration:none;}.detail-section .ds-value a:hover{text-decoration:underline;}` +
		// Empty/loading/error states
		`.state-empty{text-align:center;padding:32px 20px;color:var(--text-muted);}` +
		`.state-empty .state-icon{font-size:28px;margin-bottom:8px;opacity:.5;}` +
		`.state-empty .state-msg{font-size:13px;margin-bottom:8px;}` +
		`.state-empty .state-hint{font-size:11px;color:#666;}` +
		`.state-empty .state-action{margin-top:12px;}` +
		`.state-loading{text-align:center;padding:32px;color:var(--text-muted);font-size:12px;}` +
		`.state-error{text-align:center;padding:20px;color:var(--error);font-size:12px;}` +
		// Badges/pills
		`.badge{display:inline-block;padding:2px 8px;border-radius:999px;font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:.03em;}` +
		`.badge-ok{background:#1a3a1a;color:var(--success);border:1px solid #2d5a2d;}` +
		`.badge-warn{background:#3a3519;color:var(--warning);border:1px solid #5a5325;}` +
		`.badge-error{background:#3a1919;color:var(--error);border:1px solid #5a2525;}` +
		`.badge-info{background:#19293a;color:var(--info);border:1px solid #254a6e;}` +
		`.badge-muted{background:var(--bg);color:var(--text-muted);border:1px solid var(--border);}` +
		// Related-entity links
		`.entity-link{display:inline-flex;align-items:center;gap:4px;color:var(--accent);text-decoration:none;font-size:12px;padding:2px 6px;border-radius:4px;transition:background .1s;}` +
		`.entity-link:hover{background:#1e3a5f;text-decoration:none;}` +
		// Toast
		`#page-toast{position:fixed;bottom:20px;right:20px;z-index:9999;display:none;padding:10px 16px;border-radius:6px;font-size:12px;color:var(--text);background:var(--bg);border:1px solid var(--border);box-shadow:0 4px 12px rgba(0,0,0,0.4);}` +
		// Footer
		`.footer{background:var(--surface);border-top:1px solid var(--border);padding:10px 20px;text-align:center;font-size:12px;color:var(--text-muted);margin-top:32px;}` +
		`.footer a{color:var(--accent);text-decoration:none;}.footer a:hover{text-decoration:underline;}`
}

// uiSharedJS returns client-side JavaScript helpers shared by all pages.
func uiSharedJS() string {
	return `const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));` +
		// Toast
		`function pageToast(msg,type){var t=document.getElementById('page-toast');if(!t){t=document.createElement('div');t.id='page-toast';document.body.appendChild(t);}t.textContent=msg;t.style.display='block';t.style.borderColor=type==='error'?'var(--error)':type==='warn'?'var(--warning)':'var(--success)';setTimeout(()=>{t.style.display='none';},3000);}` +
		// Shared API helper
		`const cocoAPI={` +
		`async get(url){const r=await fetch(url);if(!r.ok)throw new Error(r.status+' '+r.statusText);return r.json();},` +
		`async post(url,body){const r=await fetch(url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});if(!r.ok){const e=await r.json().catch(()=>({}));throw new Error(e.error?.message||r.statusText);}return r.json();},` +
		`async del(url){const r=await fetch(url,{method:'DELETE'});if(!r.ok)throw new Error(r.statusText);return r.json().catch(()=>({}));},` +
		`};` +
		// Timestamp formatting
		`function formatAgo(iso){if(!iso)return '—';const d=new Date(iso),s=Math.floor((Date.now()-d)/1000);if(s<60)return s+'s ago';if(s<3600)return Math.floor(s/60)+'m ago';if(s<86400)return Math.floor(s/3600)+'h ago';return d.toLocaleDateString();}` +
		`function formatTime(iso){if(!iso)return '—';return new Date(iso).toLocaleString();}` +
		// State renderers
		`function renderEmpty(el,icon,msg,hint,action){` +
		`el.innerHTML='<div class="state-empty">'+(icon?'<div class="state-icon">'+icon+'</div>':'')+` +
		`'<div class="state-msg">'+escapeHtml(msg)+'</div>'+(hint?'<div class="state-hint">'+hint+'</div>':'')+(action||'')+'</div>';}` +
		`function renderLoading(el){el.innerHTML='<div class="state-loading">Loading\u2026</div>';}` +
		`function renderError(el,err){el.innerHTML='<div class="state-error">'+escapeHtml(err)+'</div>';}` +
		// Status badge helper
		`function statusBadge(status){const m={SUCCEEDED:'ok',COMPLETED:'ok',DONE:'ok',ONLINE:'ok',FAILED:'error',CANCELLED:'warn',QUEUED:'info',CLAIMED:'warn',RUNNING:'warn',IN_PROGRESS:'warn',TODO:'info',PENDING:'info'};` +
		`const cls=m[status]||'muted';return '<span class="badge badge-'+cls+'">'+escapeHtml(status)+'</span>';}` +
		// Entity link builder
		`function taskLink(id,label){return '<a class="entity-link" href="/board?task='+id+'">#'+id+(label?' '+escapeHtml(label):'')+'</a>';}` +
		`function runLink(id){return '<a class="entity-link" href="/runs/'+encodeURIComponent(id)+'">'+escapeHtml(id.substring(0,12))+'</a>';}` +
		`function agentLink(id,name){return '<a class="entity-link" href="/agents?id='+encodeURIComponent(id)+'">'+escapeHtml(name||id.substring(0,16))+'</a>';}` +
		// Active nav highlighting
		`(function(){const path=location.pathname;document.querySelectorAll('.top-nav nav a').forEach(a=>{if(a.getAttribute('href')===path)a.classList.add('active');});}());` +
		// Shared form field renderer
		`const cocoForm={` +
		`text(id,label,opts){opts=opts||{};return '<div class="field-stack"><label for="'+id+'">'+escapeHtml(label)+'</label><input class="input" id="'+id+'" name="'+id+'"'+(opts.placeholder?' placeholder="'+escapeHtml(opts.placeholder)+'"':'')+(opts.required?' required':'')+(opts.value?' value="'+escapeHtml(opts.value)+'"':'')+' type="'+(opts.type||'text')+'"></div>';},` +
		`select(id,label,options,opts){opts=opts||{};let h='<div class="field-stack"><label for="'+id+'">'+escapeHtml(label)+'</label><select class="select" id="'+id+'" name="'+id+'"'+(opts.required?' required':'')+'>';options.forEach(o=>{const v=typeof o==='string'?o:o.value;const t=typeof o==='string'?o:o.label;h+='<option value="'+escapeHtml(v)+'"'+(opts.selected===v?' selected':'')+'>'+escapeHtml(t)+'</option>';});h+='</select></div>';return h;},` +
		`textarea(id,label,opts){opts=opts||{};return '<div class="field-stack"><label for="'+id+'">'+escapeHtml(label)+'</label><textarea class="textarea" id="'+id+'" name="'+id+'"'+(opts.placeholder?' placeholder="'+escapeHtml(opts.placeholder)+'"':'')+(opts.required?' required':'')+'>'+(opts.value?escapeHtml(opts.value):'')+'</textarea></div>';},` +
		`toggle(id,label,checked){return '<div class="field" style="gap:8px;"><input type="checkbox" id="'+id+'" name="'+id+'"'+(checked?' checked':'')+' style="accent-color:var(--accent);"><label for="'+id+'" style="cursor:pointer;">'+escapeHtml(label)+'</label></div>';},` +
		`row(...fields){return '<div class="toolbar" style="align-items:end;">'+fields.join('')+'</div>';},` +
		`};`
}

// pageNav returns the top navigation bar with primary/secondary grouping.
func pageNav() string {
	return `<div class="top-nav">` +
		`<svg class="nav-icon" width="18" height="18" viewBox="0 0 16 16" fill="currentColor"><path d="M8.5 1a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13zm0 1a5.5 5.5 0 1 0 0 11 5.5 5.5 0 0 0 0-11zm-2 3.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5z"/></svg>` +
		`<a class="nav-title" href="/">Cocopilot</a>` +
		`<nav>` +
		// Primary operator pages
		`<a href="/dashboard">Dashboard</a>` +
		`<a href="/board">Work Queue</a>` +
		`<a href="/runs">Runs</a>` +
		`<a href="/agents">Agents</a>` +
		`<span class="nav-sep">|</span>` +
		// Secondary
		`<a href="/repo">Repo</a>` +
		`<a href="/events-browser">Events</a>` +
		`<a href="/settings">Settings</a>` +
		// More dropdown for advanced features
		`<div class="nav-more">` +
		`<button class="nav-more-btn">More &#9662;</button>` +
		`<div class="nav-dropdown">` +
		`<a href="/graphs/tasks">Task Graph</a>` +
		`<a href="/dependencies">Dependencies</a>` +
		`<a href="/memory">Memory</a>` +
		`<a href="/policies">Policies</a>` +
		`<a href="/planning">Planning</a>` +
		`<a href="/context-packs">Context Packs</a>` +
		`<a href="/audit">Audit Log</a>` +
		`<a href="/health">Health</a>` +
		`</div>` +
		`</div>` +
		`</nav>` +
		`<div class="nav-project"><span style="color:var(--text-muted);font-size:11px;">Project:</span><select><option>proj_default</option></select></div>` +
		`</div>`
}

// pageShell wraps page content with the shared UI framework (head, nav, footer).
func pageShell(title string, extraCSS string, extraHeadJS string, body string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	b.WriteString(`<title>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(` - Cocopilot</title>`)
	b.WriteString(`<link rel="stylesheet" href="/static/css/app.css">`)
	b.WriteString(`<script src="/static/js/coco.js"></script>`)
	b.WriteString(`<style>`)
	b.WriteString(uiSharedCSS())
	if extraCSS != "" {
		b.WriteString(extraCSS)
	}
	b.WriteString(`</style>`)
	b.WriteString(`<script>`)
	b.WriteString(uiSharedJS())
	if extraHeadJS != "" {
		b.WriteString(extraHeadJS)
	}
	b.WriteString(`</script>`)
	b.WriteString(`</head><body>`)
	b.WriteString(pageNav())
	b.WriteString(`<div class="page-body">`)
	b.WriteString(body)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="footer">Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a></div>`)
	b.WriteString(`</body></html>`)
	return b.String()
}
