/* ============================================================
   基础组件库
   - el(html)            将 HTML 字符串转为 DOM 元素
   - Card                卡片容器
   - CardHeader          卡片头部（标题 + 操作区）
   - StatCard            大数字统计卡（label + value + 变化趋势）
   - Badge               状态徽标（success / warning / error / neutral / cyan）
   - ProgressBar         进度条
   - Sparkline           SVG 迷你趋势折线
   - BarChart            SVG 柱状趋势图（带悬浮 tooltip）
   - DonutChart          环形占比图（conic-gradient）
   - Icon                SVG 图标库
   ============================================================ */

/** HTML 字符串 -> DOM Element */
function el(html) {
  const t = document.createElement('template');
  t.innerHTML = html.trim();
  return t.content.firstElementChild;
}

/** 图标库（lucide 风格线性图标） */
const Icon = (name, cls = 'h-4 w-4') => {
  const paths = {
    dashboard : '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
    chart     : '<path d="M3 3v18h18"/><path d="M7 15v-4m5 4V8m5 7v-6"/>',
    logs      : '<path d="M8 6h13M8 12h13M8 18h13"/><path d="M3 6h.01M3 12h.01M3 18h.01"/>',
    book      : '<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H21"/><path d="M4 4.5A2.5 2.5 0 0 1 6.5 2H21v20H6.5A2.5 2.5 0 0 1 4 19.5z"/>',
    shield    : '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/>',
    activity  : '<path d="M22 12h-4l-3 8L9 4l-3 8H2"/>',
    route     : '<circle cx="6" cy="19" r="3"/><path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/><circle cx="18" cy="5" r="3"/>',
    key       : '<circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.6 9.6"/><path d="m15.5 7.5 3 3L22 7l-3-3"/>',
    channels  : '<path d="M4 6h16M4 12h16M4 18h16"/><circle cx="8" cy="6" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="12" r="1.6" fill="currentColor" stroke="none"/><circle cx="9" cy="18" r="1.6" fill="currentColor" stroke="none"/>',
    models    : '<path d="M12 2 2 7l10 5 10-5-10-5z"/><path d="m2 12 10 5 10-5"/><path d="m2 17 10 5 10-5"/>',
    users     : '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>',
    gauge     : '<path d="m12 14 4-4"/><path d="M3.34 19a10 10 0 1 1 17.32 0"/>',
    wallet    : '<path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/>',
    settings  : '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>',
    arrowUp   : '<path d="M7 17 17 7"/><path d="M8 7h9v9"/>',
    arrowDown : '<path d="m7 7 10 10"/><path d="M17 8v9H8"/>',
    external  : '<path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>',
    zap       : '<path d="M13 2 3 14h7l-1 8 11-13h-7l1-7z"/>',
  };
  return `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${paths[name] || ''}</svg>`;
};

/** 卡片容器 */
function Card(spanCls = 'col-span-12', inner = '') {
  return el(`
    <section class="card rise rounded-xl border border-zinc-800 bg-zinc-900/90 p-5 shadow-2xl shadow-black/10 ${spanCls}">
      ${inner}
    </section>`);
}

/** 卡片头部 */
function CardHeader({ title, desc, action = '' }) {
  return `
    <div class="mb-4 flex items-start justify-between gap-4">
      <div>
        <h3 class="text-sm font-semibold">${title}</h3>
        ${desc ? `<p class="mt-0.5 text-xs text-zinc-400">${desc}</p>` : ''}
      </div>
      ${action}
    </div>`;
}

/** 大数字统计卡 */
function StatCard({ spanCls, label, value, unit = '', delta = 0, deltaLabel = '较昨日', icon, spark }) {
  const up = delta >= 0;
  const deltaColor = up ? 'text-emerald-400' : 'text-amber-400';
  const deltaIcon = up ? Icon('arrowUp', 'h-3 w-3') : Icon('arrowDown', 'h-3 w-3');
  return el(`
    <section class="card rise rounded-xl border border-zinc-800 bg-zinc-900/90 p-5 shadow-2xl shadow-black/10 ${spanCls}">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-zinc-400">${label}</span>
        ${icon ? `<span class="grid h-8 w-8 place-items-center rounded-lg bg-cyan-400/10 text-cyan-400">${Icon(icon, 'h-4 w-4')}</span>` : ''}
      </div>
      <div class="mt-3 flex items-baseline gap-1.5">
        <span class="stat-value text-[28px] font-extrabold leading-none">${value}</span>
        ${unit ? `<span class="text-sm font-medium text-zinc-400">${unit}</span>` : ''}
      </div>
      <div class="mt-3 flex items-center gap-3">
        <span class="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-semibold ${deltaColor} ${up ? 'bg-emerald-400/10' : 'bg-amber-400/10'}">
          ${deltaIcon}${Math.abs(delta)}%
        </span>
        <span class="text-[11px] text-zinc-500">${deltaLabel}</span>
      </div>
      ${spark ? `<div class="mt-3 -mb-1">${spark}</div>` : ''}
    </section>`);
}

/** 状态徽标 */
function Badge(text, tone = 'neutral') {
  const tones = {
    success: 'bg-emerald-400/10 text-emerald-400 border-emerald-400/20',
    warning: 'bg-amber-400/10 text-amber-400 border-amber-400/20',
    error  : 'bg-rose-400/10 text-rose-400 border-rose-400/20',
    cyan   : 'bg-cyan-400/10 text-cyan-400 border-cyan-400/20',
    neutral: 'bg-zinc-400/10 text-zinc-400 border-zinc-400/20',
  };
  return `<span class="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[11px] font-medium ${tones[tone]}">${text}</span>`;
}

/** 进度条 */
function ProgressBar({ value, warn = false, label, right }) {
  return `
    <div>
      <div class="mb-1.5 flex items-center justify-between text-xs">
        <span class="text-zinc-400">${label}</span>
        <span class="font-mono font-medium text-zinc-300">${right ?? value + '%'}</span>
      </div>
      <div class="h-1.5 overflow-hidden rounded-full bg-zinc-800">
        <div class="progress-fill ${warn ? 'warn' : ''} h-full rounded-full" style="width:0%" data-width="${Math.min(value, 100)}"></div>
      </div>
    </div>`;
}

/** SVG 迷你折线（sparkline） */
function Sparkline(points, { w = 220, h = 40, color = '#22d3ee' } = {}) {
  const max = Math.max(...points), min = Math.min(...points);
  const norm = (v) => h - 4 - ((v - min) / (max - min || 1)) * (h - 10);
  const step = w / (points.length - 1);
  const d = points.map((v, i) => `${i === 0 ? 'M' : 'L'}${(i * step).toFixed(1)},${norm(v).toFixed(1)}`).join(' ');
  return `
    <svg width="${w}" height="${h}" viewBox="0 0 ${w} ${h}" class="w-full">
      <defs>
        <linearGradient id="sg" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="${color}" stop-opacity=".25"/>
          <stop offset="100%" stop-color="${color}" stop-opacity="0"/>
        </linearGradient>
      </defs>
      <path d="${d} L${w},${h} L0,${h} Z" fill="url(#sg)"/>
      <path d="${d}" fill="none" stroke="${color}" stroke-width="1.8" stroke-linecap="round"/>
      <circle cx="${w}" cy="${norm(points[points.length - 1])}" r="2.5" fill="${color}"/>
    </svg>`;
}

/** SVG 柱状趋势图（带 tooltip） */
function BarChart(days, { height = 180 } = {}) {
  const max = Math.max(1, ...days.map((d) => d.requests));
  return el(`
    <div class="relative flex items-end gap-[6px]" style="height:${height}px">
      ${days.map((d, i) => {
        const h = Math.round((d.requests / max) * (height - 34));
        const today = i === days.length - 1;
        return `
          <div class="bar-col group relative flex flex-1 flex-col items-center justify-end self-stretch">
            <div class="chart-tip z-10 rounded-lg border border-zinc-700 bg-zinc-800 px-2.5 py-1.5 text-[11px] shadow-xl">
              <div class="font-medium text-zinc-200">${d.date}</div>
              <div class="mt-0.5 font-mono text-cyan-400">${d.requests.toLocaleString()} 次 · $${d.cost.toFixed(2)}</div>
            </div>
            <div class="w-full max-w-[26px] rounded-t-md transition-all duration-200 ${today
              ? 'bg-gradient-to-t from-cyan-500 to-cyan-300 shadow-[0_0_18px_-2px_rgba(34,211,238,.5)]'
              : 'bg-zinc-700/80 group-hover:bg-cyan-400/60'}"
              style="height:${h}px"></div>
            <div class="absolute -bottom-6 text-[10px] ${today ? 'font-semibold text-cyan-400' : 'text-zinc-500'}">${d.label}</div>
          </div>`;
      }).join('')}
    </div>`);
}

/** 环形占比图 */
function DonutChart(items, { size = 168, thickness = 20 } = {}) {
  const total = items.reduce((s, i) => s + i.value, 0);
  const palette = ['#22d3ee', '#3b82f6', '#a78bfa', '#f59e0b', '#34d399', '#f472b6'];
  let acc = 0;
  const segs = items.map((it, i) => {
    const from = (acc / total) * 360; acc += it.value;
    const to = (acc / total) * 360;
    return `${palette[i % palette.length]} ${from}deg ${to}deg`;
  }).join(',');
  return el(`
    <div class="flex items-center gap-6">
      <div class="relative shrink-0" style="width:${size}px;height:${size}px">
        <div class="h-full w-full rounded-full" style="background:conic-gradient(${segs})"></div>
        <div class="absolute inset-0 grid place-items-center rounded-full"
             style="inset:${thickness / 2}px;background:var(--donut-hole, #18181b)"></div>
        <div class="absolute inset-0 grid place-items-center text-center">
          <div>
            <div class="stat-value text-xl font-extrabold">${total.toLocaleString()}</div>
            <div class="text-[10px] text-zinc-400">万 tokens</div>
          </div>
        </div>
      </div>
      <ul class="flex-1 space-y-2.5">
        ${items.map((it, i) => `
          <li class="flex items-center gap-2 text-xs">
            <span class="h-2.5 w-2.5 rounded-sm" style="background:${palette[i % palette.length]}"></span>
            <span class="flex-1 truncate text-zinc-400">${it.name}</span>
            <span class="font-mono font-medium">${((it.value / total) * 100).toFixed(1)}%</span>
          </li>`).join('')}
      </ul>
    </div>`);
}

/* ============================================================
   管理交互组件：Toast / Modal / Form / Confirm / EmptyState / Pagination
   ============================================================ */

/** 顶部通知 */
function Toast(message, type = 'info') {
  const root = document.getElementById('toast-root');
  if (!root) return;
  const tones = {
    info: 'border-zinc-700 text-zinc-200',
    success: 'border-emerald-400/40 text-emerald-300',
    error: 'border-rose-400/40 text-rose-300',
  };
  const node = el(`<div class="toast rounded-lg border bg-zinc-900/95 px-4 py-2.5 text-xs shadow-2xl ${tones[type] || tones.info}">${message}</div>`);
  root.append(node);
  setTimeout(() => {
    node.style.opacity = '0';
    node.style.transform = 'translateY(-6px)';
    setTimeout(() => node.remove(), 200);
  }, 2600);
}

let _modalEl = null;

/** 打开弹窗；fields 存在时渲染表单，onSubmit(values) 抛错则不关闭并提示 */
function openModal({ title, bodyHTML = '', fields = null, submitText = '确定', onSubmit = null, width = 'max-w-lg' }) {
  closeModal();
  const root = document.getElementById('modal-root');
  if (!root) return null;
  const body = fields ? renderForm(fields) : bodyHTML;
  const overlay = el(`
    <div class="modal-overlay fixed inset-0 z-[100] flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
      <div class="modal-card w-full ${width} rounded-2xl border border-zinc-800 bg-zinc-900 shadow-2xl">
        <div class="flex items-center justify-between border-b border-zinc-800 px-5 py-3.5">
          <h3 class="text-sm font-semibold">${title}</h3>
          <button data-close class="grid h-7 w-7 place-items-center rounded-lg text-zinc-500 hover:bg-zinc-800 hover:text-zinc-200">✕</button>
        </div>
        <div class="max-h-[65vh] overflow-y-auto px-5 py-4" data-body>${body}</div>
        <div class="flex justify-end gap-2 border-t border-zinc-800 px-5 py-3.5">
          <button data-close class="btn btn-ghost rounded-lg border border-zinc-800 px-3.5 py-2 text-xs font-semibold text-zinc-400">取消</button>
          ${onSubmit ? `<button data-submit class="btn btn-primary rounded-lg px-3.5 py-2 text-xs font-bold">${submitText}</button>` : ''}
        </div>
      </div>
    </div>`);
  root.append(overlay);
  _modalEl = overlay;
  overlay.querySelectorAll('[data-close]').forEach((b) => b.addEventListener('click', closeModal));
  overlay.addEventListener('click', (e) => { if (e.target === overlay) closeModal(); });

  const submitBtn = overlay.querySelector('[data-submit]');
  if (submitBtn && onSubmit) {
    submitBtn.addEventListener('click', async () => {
      const values = fields ? collectForm(overlay, fields) : {};
      submitBtn.disabled = true;
      const original = submitBtn.textContent;
      submitBtn.textContent = '提交中…';
      try {
        await onSubmit(values);
        // 仅当当前活动弹窗仍是本弹窗时才关闭；若 onSubmit 内又打开了新弹窗（如展示明文 Key），保留新弹窗
        if (_modalEl === overlay) closeModal();
      } catch (err) {
        Toast(err.message || String(err), 'error');
        submitBtn.disabled = false;
        submitBtn.textContent = original;
      }
    });
  }
  return overlay;
}

function closeModal() {
  if (_modalEl) { _modalEl.remove(); _modalEl = null; }
}

/** 确认框 */
function Confirm(message, onConfirm) {
  openModal({
    title: '确认操作',
    bodyHTML: `<p class="text-sm leading-relaxed text-zinc-300">${message}</p>`,
    submitText: '确认',
    onSubmit: async () => { await onConfirm(); },
  });
}

/** 表单渲染：fields = [{name,label,type,value,options,placeholder,required,help,switchLabel}] */
function renderForm(fields) {
  return `<div class="grid gap-3">` + fields.map((f) => {
    const id = 'f_' + f.name;
    let input;
    if (f.type === 'select') {
      input = `<select id="${id}" data-name="${f.name}" class="form-input">${(f.options || []).map((o) =>
        `<option value="${o.value}" ${String(o.value) === String(f.value) ? 'selected' : ''}>${o.label}</option>`).join('')}</select>`;
    } else if (f.type === 'textarea') {
      input = `<textarea id="${id}" data-name="${f.name}" rows="3" class="form-input" placeholder="${f.placeholder || ''}">${f.value ?? ''}</textarea>`;
    } else if (f.type === 'switch') {
      input = `<label class="inline-flex items-center gap-2"><input id="${id}" data-name="${f.name}" type="checkbox" class="form-switch" ${f.value ? 'checked' : ''}/><span class="text-xs text-zinc-400">${f.switchLabel || '启用'}</span></label>`;
    } else {
      input = `<input id="${id}" data-name="${f.name}" type="${f.type || 'text'}" value="${f.value ?? ''}" class="form-input ${f.readonly ? 'opacity-60' : ''}" placeholder="${f.placeholder || ''}" ${f.readonly ? 'readonly' : ''}/>`;
    }
    return `<label class="block"><span class="mb-1 block text-xs font-medium text-zinc-400">${f.label}${f.required ? ' <span class="text-rose-400">*</span>' : ''}</span>${input}${f.help ? `<span class="mt-1 block text-[10px] text-zinc-500">${f.help}</span>` : ''}</label>`;
  }).join('') + `</div>`;
}

function collectForm(scope, fields) {
  const out = {};
  fields.forEach((f) => {
    const node = scope.querySelector(`[data-name="${f.name}"]`);
    if (!node) return;
    if (f.type === 'switch') out[f.name] = node.checked;
    else if (f.type === 'number') { const v = node.value.trim(); out[f.name] = v === '' ? null : Number(v); }
    else out[f.name] = node.value.trim();
  });
  return out;
}

/** 空状态（可带引导按钮） */
function EmptyState({ title = '暂无数据', desc = '', actionLabel = '', onAction = null, icon = 'logs' }) {
  const wrap = el(`
    <div class="flex flex-col items-center gap-3 py-12 text-center">
      <div class="grid h-12 w-12 place-items-center rounded-2xl bg-zinc-800/60 text-zinc-500">${Icon(icon, 'h-5 w-5')}</div>
      <div class="text-sm font-semibold text-zinc-300">${title}</div>
      ${desc ? `<div class="text-xs text-zinc-500">${desc}</div>` : ''}
      ${actionLabel ? `<button data-empty-action class="btn btn-primary mt-1 rounded-lg px-3.5 py-2 text-xs font-bold">${actionLabel}</button>` : ''}
    </div>`);
  if (actionLabel && onAction) wrap.querySelector('[data-empty-action]').addEventListener('click', onAction);
  return wrap;
}

/** 分页控件 */
function Pagination({ page, pageSize, total, onChange }) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  const wrap = el(`
    <div class="flex items-center justify-between gap-3 pt-3 text-xs text-zinc-500">
      <span>共 ${total} 条 · 第 ${page}/${pages} 页</span>
      <div class="flex gap-2">
        <button data-prev class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 disabled:opacity-40" ${page <= 1 ? 'disabled' : ''}>上一页</button>
        <button data-next class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 disabled:opacity-40" ${page >= pages ? 'disabled' : ''}>下一页</button>
      </div>
    </div>`);
  wrap.querySelector('[data-prev]').addEventListener('click', () => { if (page > 1) onChange(page - 1); });
  wrap.querySelector('[data-next]').addEventListener('click', () => { if (page < pages) onChange(page + 1); });
  return wrap;
}

/** 复制到剪贴板并提示 */
function copyText(text, label = '已复制') {
  const done = () => Toast(label, 'success');
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(done).catch(() => Toast('复制失败，请手动选择', 'error'));
  } else {
    Toast('当前环境不支持自动复制', 'error');
  }
}

/* ============================================================
   共享表格 / 列表片段（多视图复用）
   ============================================================ */
function logsTableHTML(logs) {
  if (!logs || !logs.length) return `<div class="py-10 text-center text-xs text-zinc-500">暂无请求日志</div>`;
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-4 font-medium">时间</th>
            <th class="pb-2.5 pr-4 font-medium">用户 / 模型</th>
            <th class="pb-2.5 pr-4 font-medium">路由渠道</th>
            <th class="pb-2.5 pr-4 font-medium text-right">Tokens</th>
            <th class="pb-2.5 pr-4 font-medium text-right">TTFT</th>
            <th class="pb-2.5 pr-4 font-medium text-right">耗时</th>
            <th class="pb-2.5 pr-4 font-medium text-right">费用</th>
            <th class="pb-2.5 font-medium text-right">状态</th>
          </tr>
        </thead>
        <tbody>
          ${logs.map((l) => `
            <tr class="row-hover border-b border-zinc-800/50">
              <td class="py-2.5 pr-4 font-mono text-zinc-500">${l.created_at}</td>
              <td class="py-2.5 pr-4">
                <div class="font-semibold">${l.user}</div>
                <div class="font-mono text-[10px] text-zinc-500">${l.model}</div>
              </td>
              <td class="py-2.5 pr-4 text-zinc-400">${l.channel}</td>
              <td class="py-2.5 pr-4 text-right font-mono text-zinc-300">${(l.input_tokens + l.output_tokens).toLocaleString()}</td>
              <td class="py-2.5 pr-4 text-right font-mono text-zinc-300">${l.ttft_ms ? l.ttft_ms + 'ms' : '—'}</td>
              <td class="py-2.5 pr-4 text-right font-mono text-zinc-300">${(l.duration_ms / 1000).toFixed(1)}s</td>
              <td class="py-2.5 pr-4 text-right font-mono font-semibold ${l.status === 'success' ? 'text-cyan-400' : 'text-zinc-600'}">$${l.total_cost}</td>
              <td class="py-2.5 text-right">${l.status === 'success' ? Badge('成功', 'success') : Badge(l.error_code || '失败', 'error')}</td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function channelListHTML(channels) {
  if (!channels || !channels.length) return `<div class="py-10 text-center text-xs text-zinc-500">暂无渠道</div>`;
  return channels.map((c) => {
    const disabled = c.status === 0;
    const warn = c.health < 95;
    const circuitBadge = disabled
      ? Badge('已停用', 'neutral')
      : c.circuit === 'half-open' ? Badge('半开探测', 'warning')
      : Badge('熔断关闭', 'success');
    const balance = c.balance == null ? '不限' : `$${Number(c.balance).toFixed(4)}`;
    return `
      <div class="group rounded-lg border border-transparent p-2 -m-2 transition hover:border-zinc-800 hover:bg-zinc-800/40 ${disabled ? 'opacity-45' : ''}">
        <div class="mb-1.5 flex items-center gap-2">
          <span class="h-1.5 w-1.5 rounded-full ${disabled ? 'bg-zinc-500' : warn ? 'bg-amber-400' : 'bg-emerald-400 dot-live text-emerald-400'}"></span>
          <span class="flex-1 truncate text-xs font-semibold">${c.name}</span>
          ${circuitBadge}
        </div>
        ${ProgressBar({ value: c.health, warn, label: `健康度 · 权重 ${c.weight} / 优先级 ${c.priority}`, right: c.health.toFixed(1) + '%' })}
        <div class="mt-1.5 flex items-center justify-between font-mono text-[10px] text-zinc-500">
          <span>${c.rpm} rpm</span>
          <span>余额 ${balance}</span>
          <span>$${c.cost_24h.toFixed(2)} / 24h</span>
        </div>
      </div>`;
  }).join('');
}

function usersListHTML(users) {
  if (!users || !users.length) return `<div class="py-10 text-center text-xs text-zinc-500">暂无用户</div>`;
  return users.map((u) => `
    <div class="row-hover flex items-center gap-3 rounded-lg px-2 py-2 -mx-2">
      <span class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-zinc-600 to-zinc-800 text-[11px] font-bold text-zinc-200">${String(u.name).slice(0, 2).toUpperCase()}</span>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="truncate text-xs font-semibold">${u.name}</span>
          ${u.group === 'vip' ? Badge('VIP', 'cyan') : u.group === 'free' ? Badge('免费', 'neutral') : ''}
          ${u.status !== 'active' ? Badge('冻结', 'error') : ''}
        </div>
        <div class="mt-0.5 font-mono text-[10px] text-zinc-500">
          可用 <span class="text-zinc-300">$${u.available_balance}</span> · 冻结 $${u.frozen_balance} · 今日 ${u.qpd.toLocaleString()} 次
        </div>
      </div>
    </div>`).join('');
}

function progressListHTML(items) {
  if (!items || !items.length) return `<div class="py-6 text-center text-xs text-zinc-500">暂无数据</div>`;
  return items.map((r) => ProgressBar({
    value: r.usage,
    warn: r.usage >= 90,
    label: `<span class="font-semibold text-zinc-300">${r.target}</span> <span class="text-zinc-600">· ${r.scope}</span>`,
    right: r.usage + '%',
  })).join('');
}

function pricingTableHTML(rows) {
  if (!rows || !rows.length) return `<div class="py-10 text-center text-xs text-zinc-500">暂无定价记录</div>`;
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-4 font-medium">渠道</th>
            <th class="pb-2.5 pr-4 font-medium">对外模型</th>
            <th class="pb-2.5 pr-4 font-medium">上游模型</th>
            <th class="pb-2.5 pr-4 font-medium text-right">输入 / 1M</th>
            <th class="pb-2.5 pr-4 font-medium text-right">输出 / 1M</th>
            <th class="pb-2.5 pr-4 font-medium text-right">缓存输入 / 1M</th>
            <th class="pb-2.5 font-medium text-right">币种</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((p) => `
            <tr class="row-hover border-b border-zinc-800/50">
              <td class="py-2.5 pr-4">${p.channel_name || ('#' + p.channel_id)}</td>
              <td class="py-2.5 pr-4 font-semibold">${p.model_name}</td>
              <td class="py-2.5 pr-4 font-mono text-[10px] text-zinc-500">${p.upstream_model || '—'}</td>
              <td class="py-2.5 pr-4 text-right font-mono text-zinc-300">$${p.input_price_per_1m}</td>
              <td class="py-2.5 pr-4 text-right font-mono text-zinc-300">$${p.output_price_per_1m}</td>
              <td class="py-2.5 pr-4 text-right font-mono text-zinc-500">${p.cached_input_price_per_1m ? '$' + p.cached_input_price_per_1m : '—'}</td>
              <td class="py-2.5 text-right text-zinc-400">${p.currency}</td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function keysTableHTML(rows) {
  if (!rows || !rows.length) return `<div class="py-10 text-center text-xs text-zinc-500">暂无 Key</div>`;
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-4 font-medium">名称</th>
            <th class="pb-2.5 pr-4 font-medium">用户</th>
            <th class="pb-2.5 pr-4 font-medium">前缀</th>
            <th class="pb-2.5 pr-4 font-medium text-right">状态</th>
            <th class="pb-2.5 font-medium text-right">最后使用</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((k) => `
            <tr class="row-hover border-b border-zinc-800/50">
              <td class="py-2.5 pr-4 font-semibold">${k.key_name}</td>
              <td class="py-2.5 pr-4 text-zinc-400">#${k.user_id}</td>
              <td class="py-2.5 pr-4 font-mono text-[10px] text-zinc-500">${k.prefix}</td>
              <td class="py-2.5 pr-4 text-right">${k.is_active ? Badge('启用', 'success') : Badge('停用', 'neutral')}</td>
              <td class="py-2.5 text-right font-mono text-[10px] text-zinc-500">${k.last_used_at ? shortTime(k.last_used_at) : '—'}</td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function modelCatalogHTML(rows) {
  if (!rows || !rows.length) return `<div class="py-10 text-center text-xs text-zinc-500">暂无已发布模型</div>`;
  return `<div class="space-y-2">
    ${rows.map((m) => `
      <div class="row-hover flex items-center justify-between rounded-lg border border-zinc-800/60 bg-zinc-950/35 px-3 py-2.5">
        <div class="min-w-0">
          <div class="truncate text-xs font-semibold">${m.model_name}</div>
          <div class="mt-0.5 font-mono text-[10px] text-zinc-500">${(m.channels || []).map((c) => c.channel_name).join(' · ') || '无渠道'}</div>
        </div>
        <div class="text-right">
          <div class="font-mono text-sm font-bold text-white">${m.channel_count ?? (m.channels || []).length}</div>
          <div class="text-[10px] text-zinc-500">可用渠道</div>
        </div>
      </div>`).join('')}
  </div>`;
}
