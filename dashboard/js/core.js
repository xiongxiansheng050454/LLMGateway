/* ============================================================
   控制台核心：导航 / 路由 / 顶栏 / 启动
   ============================================================ */

const NAV_ITEMS = [
  { group: '总览' },
  { key: 'dashboard', name: '仪表盘',   icon: 'dashboard' },
  { key: 'usage',     name: '用量统计', icon: 'chart' },
  { key: 'logs',      name: '请求日志', icon: 'logs', badge: '1.7k' },
  { key: 'docs',      name: 'API 文档', icon: 'book' },
  { group: '资源管理' },
  { key: 'channels',  name: '渠道管理', icon: 'channels' },
  { key: 'models',    name: '模型路由', icon: 'models' },
  { key: 'users',     name: '用户与 Key', icon: 'users' },
  { group: '运营' },
  { key: 'limits',    name: '限流规则', icon: 'gauge' },
  { key: 'pricing',   name: '计费定价', icon: 'wallet' },
  { key: 'settings',  name: '系统设置', icon: 'settings' },
];

let LAST_UPDATED = '';

function renderNav() {
  const nav = document.getElementById('nav');
  nav.innerHTML = '';
  NAV_ITEMS.forEach((item) => {
    if (item.group) {
      nav.append(el(`<div class="px-3 pb-1 pt-4 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">${item.group}</div>`));
      return;
    }
    const btn = el(`
      <button data-key="${item.key}" class="nav-item flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-zinc-400">
        ${Icon(item.icon, 'h-4 w-4')}
        <span class="flex-1 text-left">${item.name}</span>
        ${item.badge ? `<span class="rounded-md bg-cyan-400/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-cyan-400">${item.badge}</span>` : ''}
      </button>`);
    btn.addEventListener('click', () => navigate(item.key));
    nav.append(btn);
  });
}

function setActiveNav(key) {
  document.querySelectorAll('.nav-item').forEach((n) => n.classList.toggle('active', n.dataset.key === key));
}

function renderChrome(view, key) {
  document.getElementById('breadcrumb').innerHTML = `
    <span class="text-zinc-400">MyApi</span>
    <svg class="h-3.5 w-3.5 text-zinc-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="m9 18 6-6-6-6"/></svg>
    <span class="text-zinc-400">Gateway</span>
    <svg class="h-3.5 w-3.5 text-zinc-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="m9 18 6-6-6-6"/></svg>
    <span class="font-semibold">${view.title}</span>`;

  document.getElementById('page-title').textContent = view.title;
  document.getElementById('page-subtitle').innerHTML = view.subtitle ? view.subtitle() : '';

  const actions = document.getElementById('page-actions');
  actions.innerHTML = '';
  if (key === 'docs') {
    const back = el(`<button class="btn btn-ghost flex items-center gap-1.5 rounded-lg border border-zinc-800 px-3.5 py-2 text-xs font-semibold text-zinc-400">${Icon('dashboard', 'h-3.5 w-3.5')}返回控制台</button>`);
    back.addEventListener('click', () => navigate('dashboard'));
    actions.append(back);
  } else {
    const docsBtn = el(`<button class="btn btn-ghost flex items-center gap-1.5 rounded-lg border border-zinc-800 px-3.5 py-2 text-xs font-semibold text-zinc-400">${Icon('book', 'h-3.5 w-3.5')}查看 API 文档</button>`);
    docsBtn.addEventListener('click', () => navigate('docs'));
    const newBtn = el(`<button class="btn btn-primary flex items-center gap-1.5 rounded-lg px-3.5 py-2 text-xs font-bold">${Icon('zap', 'h-3.5 w-3.5')}新建渠道</button>`);
    newBtn.addEventListener('click', () => navigate('channels'));
    actions.append(docsBtn, newBtn);
  }
  setUpdatedAt();
}

function renderGatewayStatus() {
  const p95 = 1820, uptime = '99.98%';
  document.getElementById('gateway-status').innerHTML = `
    <div class="flex items-center gap-2">
      <span class="dot-live inline-block h-2 w-2 rounded-full bg-emerald-400 text-emerald-400"></span>
      <span class="text-xs font-semibold">网关运行正常</span>
    </div>
    <div class="mt-2 grid grid-cols-2 gap-2 text-center">
      <div class="rounded-lg bg-zinc-950/60 py-1.5">
        <div class="font-mono text-xs font-bold text-cyan-400">${p95}ms</div>
        <div class="text-[10px] text-zinc-500">P95 延迟</div>
      </div>
      <div class="rounded-lg bg-zinc-950/60 py-1.5">
        <div class="font-mono text-xs font-bold text-emerald-400">${uptime}</div>
        <div class="text-[10px] text-zinc-500">30 天可用性</div>
      </div>
    </div>`;
}

function setUpdatedAt() {
  const node = document.getElementById('updated-at');
  if (node) node.textContent = LAST_UPDATED || '—';
}

function animateProgress() {
  requestAnimationFrame(() => {
    setTimeout(() => {
      document.querySelectorAll('.progress-fill').forEach((bar) => {
        bar.style.width = bar.dataset.width + '%';
      });
    }, 120);
  });
}

/* ---------- 视图注册表 / 路由 ---------- */
const VIEWS = {
  dashboard: { title: '运营仪表盘', subtitle: () => `网关运行总览 · 数据更新于 <span id="updated-at" class="font-mono text-cyan-400">${LAST_UPDATED || '—'}</span>`, render: renderDashboardView },
  usage:     { title: '用量统计', subtitle: () => '按用户 / 模型聚合的 token 与费用', render: renderUsageView },
  logs:      { title: '请求日志', subtitle: () => 'usage_logs 明细', render: renderLogsView },
  docs:      { title: 'API 文档', subtitle: () => '下游与管理端接口文档（应用内阅读）', render: renderDocsView },
  channels:  { title: '渠道管理', subtitle: () => '上游渠道 · 健康 / 权重 / 余额', render: renderChannelsView },
  models:    { title: '模型路由', subtitle: () => '对外模型目录与路由策略', render: renderModelsView },
  users:     { title: '用户与 Key', subtitle: () => '下游用户余额与网关 Key', render: renderUsersView },
  limits:    { title: '限流规则', subtitle: () => 'rate_limit_rules · 配额水位', render: renderLimitsView },
  pricing:   { title: '计费定价', subtitle: () => '渠道×模型单价', render: renderPricingView },
  settings:  { title: '系统设置', subtitle: () => '控制台与网关信息', render: renderSettingsView },
};

function parseRoute() {
  const raw = window.location.hash.replace(/^#\/?/, '');
  const [key, qs] = raw.split('?');
  const params = new URLSearchParams(qs || '');
  return { key: key || 'dashboard', doc: params.get('doc') };
}

function applyRoute() {
  const { key, doc } = parseRoute();
  const activeKey = VIEWS[key] ? key : 'dashboard';
  const view = VIEWS[activeKey];
  setActiveNav(activeKey);
  renderChrome(view, activeKey);
  const bento = document.getElementById('bento');
  bento.innerHTML = '';
  view.render(doc);
}

function navigate(key, doc) {
  const target = '#/' + key + (doc ? '?doc=' + encodeURIComponent(doc) : '');
  if (window.location.hash === target) {
    applyRoute();
  } else {
    window.location.hash = target;
  }
}

window.addEventListener('hashchange', applyRoute);

/* ---------- 交互 ---------- */
function bindInteractions() {
  document.getElementById('theme-toggle').addEventListener('click', () => {
    document.documentElement.classList.toggle('dark');
  });

  const bell = document.getElementById('notify-btn');
  const panel = el(`
    <div id="notify-panel" class="absolute right-0 top-11 z-50 hidden w-80 rounded-xl border border-zinc-800 bg-zinc-900 p-3 shadow-2xl">
      <div class="mb-2 px-1 text-xs font-semibold text-zinc-400">通知（${NOTIFICATIONS.length}）</div>
      ${NOTIFICATIONS.map((n) => `
        <div class="row-hover rounded-lg p-2.5 text-xs leading-relaxed text-zinc-400">
          ${Badge(n.tone === 'warning' ? '警告' : n.tone === 'success' ? '结算' : '路由', n.tone)}
          <div class="mt-1.5">${n.text}</div>
        </div>`).join('')}
    </div>`);
  bell.style.position = 'relative';
  bell.append(panel);
  bell.addEventListener('click', (e) => {
    e.stopPropagation();
    panel.classList.toggle('hidden');
  });
  document.addEventListener('click', () => panel.classList.add('hidden'));

  document.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      document.getElementById('global-search').focus();
    }
  });
}

function renderLoading() {
  document.getElementById('bento').innerHTML = `
    <section class="card rounded-xl border border-zinc-800 bg-zinc-900/90 p-6 col-span-12">
      <div class="flex items-center gap-3 text-sm text-zinc-400">
        <span class="h-2 w-2 rounded-full bg-cyan-400 dot-live text-cyan-400"></span>
        正在连接管理端接口并加载实时数据...
      </div>
    </section>`;
}

function renderError(err) {
  document.getElementById('bento').innerHTML = `
    <section class="card rounded-xl border border-amber-400/30 bg-zinc-900/90 p-6 col-span-12">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="text-sm font-semibold text-amber-400">管理端接口连接失败</div>
          <p class="mt-1 text-xs text-zinc-400">请确认网关服务已启动，且 Dashboard 与 /admin 接口同源；跨端口部署时可使用 <span class="font-mono text-cyan-400">?api_base=http://host:port/admin</span> 指定接口地址。</p>
          <p class="mt-2 font-mono text-xs text-zinc-500">${String(err.message || err)}</p>
        </div>
        <button onclick="window.location.reload()" class="btn btn-primary rounded-lg px-3.5 py-2 text-xs font-bold">重新加载</button>
      </div>
    </section>`;
}

/* ---------- 启动 ---------- */
async function bootDashboard() {
  renderNav();
  renderGatewayStatus();
  renderLoading();
  try {
    await loadDashboardData();
    LAST_UPDATED = new Date().toLocaleTimeString('zh-CN', { hour12: false });
    bindInteractions();
    applyRoute();
  } catch (err) {
    renderError(err);
    LAST_UPDATED = '加载失败';
  }
}

bootDashboard();
