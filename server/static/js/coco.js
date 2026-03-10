/* coco.js — Cocopilot shared client-side library.
   Single source of truth for API access, status vocabulary, rendering helpers,
   forms, toast, live-state, and entity linking. */

// ── Escaping ─────────────────────────────────────────────────────────
const escapeHtml = (v) =>
  String(v ?? '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]));

// ── Toast ────────────────────────────────────────────────────────────
function pageToast(msg, type) {
  let t = document.getElementById('page-toast');
  if (!t) {
    t = document.createElement('div');
    t.id = 'page-toast';
    document.body.appendChild(t);
  }
  t.textContent = msg;
  t.style.display = 'block';
  t.style.borderColor =
    type === 'error' ? 'var(--error)' :
    type === 'warn'  ? 'var(--warning)' : 'var(--success)';
  setTimeout(() => { t.style.display = 'none'; }, 3000);
}

// ── API Layer ────────────────────────────────────────────────────────
const cocoAPI = {
  async get(url) {
    const r = await fetch(url);
    if (!r.ok) throw new Error(r.status + ' ' + r.statusText);
    return r.json();
  },
  async post(url, body) {
    const r = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (!r.ok) {
      const e = await r.json().catch(() => ({}));
      throw new Error(e.error?.message || r.statusText);
    }
    return r.json();
  },
  async del(url) {
    const r = await fetch(url, { method: 'DELETE' });
    if (!r.ok) throw new Error(r.statusText);
    return r.json().catch(() => ({}));
  },
  async put(url, body) {
    const r = await fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (!r.ok) {
      const e = await r.json().catch(() => ({}));
      throw new Error(e.error?.message || r.statusText);
    }
    return r.json();
  },
  async patch(url, body) {
    const r = await fetch(url, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (!r.ok) {
      const e = await r.json().catch(() => ({}));
      throw new Error(e.error?.message || r.statusText);
    }
    return r.json();
  },
};

// ── Canonical Status Vocabulary ──────────────────────────────────────
// Single mapping layer. All pages use this instead of local status logic.
const cocoStatus = {
  // Canonical task display status: normalizes backend variations
  taskLabel(raw) {
    const s = String(raw || '').toUpperCase();
    const map = { NOT_PICKED: 'QUEUED', TODO: 'QUEUED', PENDING: 'QUEUED',
      IN_PROGRESS: 'RUNNING', CLAIMED: 'RUNNING',
      DONE: 'SUCCEEDED', COMPLETED: 'SUCCEEDED',
      NEEDS_REVIEW: 'REVIEW' };
    return map[s] || s;
  },
  // Badge class: ok, warn, error, info, muted
  badgeClass(raw) {
    const s = String(raw || '').toUpperCase();
    const map = {
      SUCCEEDED: 'ok', COMPLETED: 'ok', DONE: 'ok', ONLINE: 'ok',
      FAILED: 'error', CANCELLED: 'warn', BLOCKED: 'warn',
      QUEUED: 'info', CLAIMED: 'warn', RUNNING: 'warn',
      IN_PROGRESS: 'warn', TODO: 'info', PENDING: 'info',
      NOT_PICKED: 'info', NEEDS_REVIEW: 'warn', REVIEW: 'warn',
      ACTIVE: 'ok', STALE: 'warn', OFFLINE: 'muted'
    };
    return map[s] || 'muted';
  },
  // Is the task in a "done" state?
  isDone(raw) {
    const s = String(raw || '').toUpperCase();
    return ['SUCCEEDED', 'COMPLETED', 'DONE', 'FAILED', 'CANCELLED'].includes(s);
  },
  // Is the task actionable (can be claimed)?
  isClaimable(raw) {
    const s = String(raw || '').toUpperCase();
    return ['NOT_PICKED', 'QUEUED', 'TODO', 'PENDING'].includes(s);
  },
  // Agent display status
  agentLabel(raw) {
    const s = String(raw || '').toUpperCase();
    if (s === 'ACTIVE') return 'ONLINE';
    return s || 'UNKNOWN';
  },
};

// ── Timestamp Formatting ─────────────────────────────────────────────
function formatAgo(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  const s = Math.floor((Date.now() - d) / 1000);
  if (s < 60) return s + 's ago';
  if (s < 3600) return Math.floor(s / 60) + 'm ago';
  if (s < 86400) return Math.floor(s / 3600) + 'h ago';
  return d.toLocaleDateString();
}

function formatTime(iso) {
  if (!iso) return '—';
  return new Date(iso).toLocaleString();
}

// ── State Renderers ──────────────────────────────────────────────────
function renderEmpty(el, icon, msg, hint, action) {
  el.innerHTML = '<div class="state-empty">' +
    (icon ? '<div class="state-icon">' + icon + '</div>' : '') +
    '<div class="state-msg">' + escapeHtml(msg) + '</div>' +
    (hint ? '<div class="state-hint">' + hint + '</div>' : '') +
    (action || '') + '</div>';
}

function renderLoading(el) {
  el.innerHTML = '<div class="state-loading">Loading\u2026</div>';
}

function renderError(el, err) {
  el.innerHTML = '<div class="state-error">' + escapeHtml(err) + '</div>';
}

// ── Status Badge ─────────────────────────────────────────────────────
function statusBadge(status) {
  const cls = cocoStatus.badgeClass(status);
  return '<span class="badge badge-' + cls + '">' + escapeHtml(cocoStatus.taskLabel(status)) + '</span>';
}

// ── Entity Links ─────────────────────────────────────────────────────
function taskLink(id, label) {
  return '<a class="entity-link" href="/board?task=' + id + '">#' + id +
    (label ? ' ' + escapeHtml(label) : '') + '</a>';
}

function runLink(id) {
  return '<a class="entity-link" href="/runs/' + encodeURIComponent(id) + '">' +
    escapeHtml(String(id).substring(0, 12)) + '</a>';
}

function agentLink(id, name) {
  return '<a class="entity-link" href="/agents?id=' + encodeURIComponent(id) + '">' +
    escapeHtml(name || String(id).substring(0, 16)) + '</a>';
}

function projectLink(id) {
  return '<a class="entity-link" href="/projects?id=' + encodeURIComponent(id) + '">' +
    escapeHtml(id) + '</a>';
}

// ── Table Builder ────────────────────────────────────────────────────
function cocoTable(columns, rows, opts) {
  opts = opts || {};
  let html = '<table' + (opts.className ? ' class="' + opts.className + '"' : '') + '>';
  html += '<thead><tr>';
  columns.forEach(function(col) {
    html += '<th>' + escapeHtml(typeof col === 'string' ? col : col.label) + '</th>';
  });
  html += '</tr></thead><tbody>';
  if (rows.length === 0) {
    html += '<tr><td colspan="' + columns.length + '" class="state-empty">' +
      escapeHtml(opts.emptyText || 'No data') + '</td></tr>';
  } else {
    rows.forEach(function(row, idx) {
      const onclick = opts.onRowClick ? ' onclick="' + opts.onRowClick.replace('$idx', idx) + '"' : '';
      html += '<tr' + onclick + '>';
      columns.forEach(function(col) {
        const key = typeof col === 'string' ? col : col.key;
        const render = typeof col === 'object' && col.render;
        const val = render ? render(row) : escapeHtml(row[key] ?? '');
        html += '<td>' + val + '</td>';
      });
      html += '</tr>';
    });
  }
  html += '</tbody></table>';
  return html;
}

// ── Card Builder ─────────────────────────────────────────────────────
function cocoCard(title, bodyHtml, opts) {
  opts = opts || {};
  let html = '<div class="card"' + (opts.id ? ' id="' + opts.id + '"' : '') + '>';
  if (title) html += '<h2 style="margin-top:0;">' + escapeHtml(title) + '</h2>';
  html += bodyHtml + '</div>';
  return html;
}

// ── Stat Card Builder ────────────────────────────────────────────────
function cocoStat(label, value, cls) {
  return '<div class="stat ' + (cls || '') + '">' +
    '<div class="label">' + escapeHtml(label) + '</div>' +
    '<div class="value">' + escapeHtml(String(value)) + '</div></div>';
}

// ── Form Helpers ─────────────────────────────────────────────────────
const cocoForm = {
  text(id, label, opts) {
    opts = opts || {};
    return '<div class="field-stack"><label for="' + id + '">' + escapeHtml(label) + '</label>' +
      '<input class="input" id="' + id + '" name="' + id + '"' +
      (opts.placeholder ? ' placeholder="' + escapeHtml(opts.placeholder) + '"' : '') +
      (opts.required ? ' required' : '') +
      (opts.value ? ' value="' + escapeHtml(opts.value) + '"' : '') +
      ' type="' + (opts.type || 'text') + '"></div>';
  },
  select(id, label, options, opts) {
    opts = opts || {};
    let h = '<div class="field-stack"><label for="' + id + '">' + escapeHtml(label) +
      '</label><select class="select" id="' + id + '" name="' + id + '"' +
      (opts.required ? ' required' : '') + '>';
    options.forEach(function(o) {
      const v = typeof o === 'string' ? o : o.value;
      const t = typeof o === 'string' ? o : o.label;
      h += '<option value="' + escapeHtml(v) + '"' +
        (opts.selected === v ? ' selected' : '') + '>' + escapeHtml(t) + '</option>';
    });
    h += '</select></div>';
    return h;
  },
  textarea(id, label, opts) {
    opts = opts || {};
    return '<div class="field-stack"><label for="' + id + '">' + escapeHtml(label) +
      '</label><textarea class="textarea" id="' + id + '" name="' + id + '"' +
      (opts.placeholder ? ' placeholder="' + escapeHtml(opts.placeholder) + '"' : '') +
      (opts.required ? ' required' : '') + '>' +
      (opts.value ? escapeHtml(opts.value) : '') + '</textarea></div>';
  },
  toggle(id, label, checked) {
    return '<div class="field" style="gap:8px;">' +
      '<input type="checkbox" id="' + id + '" name="' + id + '"' +
      (checked ? ' checked' : '') + ' style="accent-color:var(--accent);">' +
      '<label for="' + id + '" style="cursor:pointer;">' + escapeHtml(label) + '</label></div>';
  },
  row() {
    return '<div class="toolbar" style="align-items:end;">' +
      Array.prototype.slice.call(arguments).join('') + '</div>';
  },
};

// ── Global Project Context ───────────────────────────────────────────
const cocoProject = {
  _current: null,
  get() {
    if (this._current) return this._current;
    try { this._current = localStorage.getItem('coco_project') || 'proj_default'; }
    catch(e) { this._current = 'proj_default'; }
    return this._current;
  },
  set(id) {
    this._current = id;
    try { localStorage.setItem('coco_project', id); } catch(e) {}
    document.querySelectorAll('.nav-project select').forEach(function(el) { el.value = id; });
    window.dispatchEvent(new CustomEvent('coco:project-changed', { detail: { project: id } }));
  },
  // Initialize project selector in nav
  async initSelector() {
    const container = document.querySelector('.nav-project');
    if (!container) return;
    try {
      const projects = await cocoAPI.get('/api/v2/projects');
      const list = Array.isArray(projects) ? projects : [];
      const sel = container.querySelector('select');
      if (!sel) return;
      sel.innerHTML = '';
      if (list.length === 0) {
        sel.innerHTML = '<option value="proj_default">proj_default</option>';
      } else {
        list.forEach(function(p) {
          const id = p.id || p;
          const opt = document.createElement('option');
          opt.value = id;
          opt.textContent = p.name || id;
          if (id === cocoProject.get()) opt.selected = true;
          sel.appendChild(opt);
        });
      }
      sel.addEventListener('change', function() { cocoProject.set(sel.value); });
    } catch(e) { /* project list not available yet */ }
  },
};

// ── Live State Helper ────────────────────────────────────────────────
// Unifies interval-based refresh + SSE into one API.
const cocoLive = {
  _intervals: [],
  _sources: [],
  // Start interval-based polling
  poll(fn, intervalMs) {
    fn();
    const id = setInterval(fn, intervalMs || 15000);
    this._intervals.push(id);
    return id;
  },
  // Start SSE connection with auto-reconnect
  sse(url, onEvent, opts) {
    opts = opts || {};
    const self = this;
    let source = null;
    let reconnectTimer = null;
    function connect() {
      source = new EventSource(url);
      self._sources.push(source);
      source.onmessage = function(e) {
        try { onEvent(JSON.parse(e.data)); } catch(err) { onEvent(e.data); }
      };
      source.onerror = function() {
        source.close();
        if (opts.onDisconnect) opts.onDisconnect();
        reconnectTimer = setTimeout(connect, opts.reconnectMs || 5000);
      };
      source.onopen = function() {
        if (opts.onConnect) opts.onConnect();
      };
    }
    connect();
    return { close: function() { if (source) source.close(); clearTimeout(reconnectTimer); } };
  },
  // Clean up all
  cleanup() {
    this._intervals.forEach(clearInterval);
    this._sources.forEach(function(s) { try { s.close(); } catch(e) {} });
    this._intervals = [];
    this._sources = [];
  },
};

// ── Confirm Dialog ───────────────────────────────────────────────────
function cocoConfirm(msg, onConfirm) {
  if (window._pageConfirm) {
    window._pageConfirm(msg, onConfirm);
    return;
  }
  if (confirm(msg)) onConfirm();
}

// ── Active nav highlighting ──────────────────────────────────────────
(function() {
  const path = location.pathname;
  document.querySelectorAll('.top-nav nav a').forEach(function(a) {
    if (a.getAttribute('href') === path) a.classList.add('active');
  });
  // Initialize global project selector
  if (typeof cocoProject !== 'undefined') cocoProject.initSelector();
}());
