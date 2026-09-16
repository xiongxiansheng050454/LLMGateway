/* 视图：API 文档（应用内 Markdown 阅读器） */
const DOC_LIST = [
  { title: '文档总览',            href: '../docs/README.md' },
  { title: 'Chat Completions',   href: '../docs/01-下游接口/chat-completions.md' },
  { title: 'Models',             href: '../docs/01-下游接口/models.md' },
  { title: '结算与限速语义',       href: '../docs/01-下游接口/结算与限速语义.md' },
  { title: '渠道管理',            href: '../docs/02-管理端接口/channels.md' },
  { title: '用户与 Key',          href: '../docs/02-管理端接口/users-keys.md' },
  { title: '计费定价',            href: '../docs/02-管理端接口/pricing.md' },
  { title: '限流规则',            href: '../docs/02-管理端接口/rate-limits.md' },
  { title: '统计与账单',          href: '../docs/02-管理端接口/stats-billing.md' },
  { title: '附录',               href: '../docs/03-附录.md' },
];

async function renderDocsView(docHref) {
  const bento = document.getElementById('bento');
  const active = docHref || DOC_LIST[0].href;

  const list = Card('col-span-12 xl:col-span-3', `
    ${CardHeader({ title: '文档目录', desc: '点击切换' })}
    <div id="doc-list" class="space-y-1">
      ${DOC_LIST.map((d) => `
        <button data-href="${d.href}" class="doc-link w-full rounded-lg px-3 py-2 text-left text-xs transition ${d.href === active ? 'bg-cyan-400/10 font-semibold text-cyan-400' : 'text-zinc-400 hover:bg-zinc-800/60'}">${d.title}</button>`).join('')}
    </div>`);
  bento.append(list);

  const body = Card('col-span-12 xl:col-span-9', `
    <div class="mb-4 flex items-center justify-between gap-3">
      <div class="min-w-0">
        <div id="doc-title" class="text-sm font-semibold">${(DOC_LIST.find((d) => d.href === active) || DOC_LIST[0]).title}</div>
        <div id="doc-path" class="mt-0.5 truncate font-mono text-[11px] text-zinc-500">${active}</div>
      </div>
      <button id="doc-back" class="btn btn-ghost flex shrink-0 items-center gap-1.5 rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">${Icon('dashboard', 'h-3 w-3')}返回控制台</button>
    </div>
    <article id="doc-content" class="md-body">加载中…</article>`);
  bento.append(body);

  list.querySelectorAll('.doc-link').forEach((btn) => {
    btn.addEventListener('click', () => navigate('docs', btn.dataset.href));
  });
  document.getElementById('doc-back').addEventListener('click', () => navigate('dashboard'));

  try {
    const res = await fetch(active, { headers: { Accept: 'text/markdown, text/plain, */*' } });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const text = await res.text();
    const html = window.marked ? window.marked.parse(text) : `<pre>${text}</pre>`;
    const clean = window.DOMPurify ? window.DOMPurify.sanitize(html) : html;
    const node = document.getElementById('doc-content');
    if (node) node.innerHTML = clean;
  } catch (err) {
    const node = document.getElementById('doc-content');
    if (node) node.innerHTML = `<div class="py-6 text-rose-400">文档加载失败：${err.message}</div>`;
  }
}

function bindDocOpeners(scope) {
  scope.querySelectorAll('.doc-open').forEach((btn) => {
    btn.addEventListener('click', () => {
      const href = btn.dataset.doc;
      navigate('docs', href === '__list__' ? null : href);
    });
  });
}
