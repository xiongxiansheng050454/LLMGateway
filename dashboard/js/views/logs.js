/* 视图：请求日志（筛选 / 分页 / 详情） */
let LOG_PAGE = 1;
const LOG_PAGE_SIZE = 20;
let LOG_CACHE = [];

async function renderLogsView() {
  const bento = document.getElementById('bento');
  const card = Card('col-span-12', `
    ${CardHeader({
      title: '请求日志',
      desc: 'usage_logs · 支持按用户/渠道/模型/状态/时间筛选',
      action: `<button id="lg-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">刷新</button>`,
    })}
    <div class="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
      <input id="lf-user" class="form-input" placeholder="用户 ID"/>
      <input id="lf-channel" class="form-input" placeholder="渠道 ID"/>
      <input id="lf-model" class="form-input" placeholder="模型"/>
      <select id="lf-status" class="form-input">
        <option value="">全部状态</option>
        <option value="success">success</option>
        <option value="error">error</option>
      </select>
      <input id="lf-start" type="datetime-local" class="form-input" title="开始时间"/>
      <input id="lf-end" type="datetime-local" class="form-input" title="结束时间"/>
    </div>
    <div class="mb-4 flex gap-2">
      <button id="lf-query" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">查询</button>
      <button id="lf-reset" class="btn btn-ghost rounded-lg border border-zinc-800 px-3.5 py-1.5 text-[11px] font-semibold text-zinc-400">重置</button>
    </div>
    <div id="log-list" class="py-10 text-center text-xs text-zinc-500">加载中…</div>
    <div id="log-pager"></div>`);
  bento.append(card);

  document.getElementById('lg-refresh').addEventListener('click', () => loadLogs(LOG_PAGE));
  document.getElementById('lf-query').addEventListener('click', () => loadLogs(1));
  document.getElementById('lf-reset').addEventListener('click', () => {
    ['lf-user', 'lf-channel', 'lf-model', 'lf-status', 'lf-start', 'lf-end'].forEach((id) => { document.getElementById(id).value = ''; });
    loadLogs(1);
  });
  await loadLogs(1);
}

function logFilterParams() {
  const params = { page: LOG_PAGE, page_size: LOG_PAGE_SIZE };
  const user = document.getElementById('lf-user')?.value.trim();
  const channel = document.getElementById('lf-channel')?.value.trim();
  const model = document.getElementById('lf-model')?.value.trim();
  const status = document.getElementById('lf-status')?.value;
  const start = document.getElementById('lf-start')?.value;
  const end = document.getElementById('lf-end')?.value;
  if (user) params.user_id = user;
  if (channel) params.channel_id = channel;
  if (model) params.model = model;
  if (status) params.status = status;
  if (start) params.start_time = new Date(start).toISOString();
  if (end) params.end_time = new Date(end).toISOString();
  return params;
}

async function loadLogs(page) {
  LOG_PAGE = page || 1;
  const node = document.getElementById('log-list');
  if (!node) return;
  try {
    const params = logFilterParams();
    params.page = LOG_PAGE;
    const d = await adminGet('/usage-logs', params);
    const rows = d?.list || [];
    LOG_CACHE = rows;
    node.innerHTML = '';
    if (rows.length) {
      node.innerHTML = logsAdminTableHTML(rows);
      bindLogActions();
    } else {
      node.append(EmptyState({ title: '暂无日志', desc: '当前筛选条件下没有请求记录', icon: 'logs' }));
    }
    const pager = document.getElementById('log-pager');
    pager.innerHTML = '';
    pager.append(Pagination({ page: LOG_PAGE, pageSize: LOG_PAGE_SIZE, total: d?.total ?? rows.length, onChange: loadLogs }));
  } catch (err) {
    node.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function logsAdminTableHTML(rows) {
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-3 font-medium">时间</th>
            <th class="pb-2.5 pr-3 font-medium">Request ID</th>
            <th class="pb-2.5 pr-3 font-medium">用户 / 模型</th>
            <th class="pb-2.5 pr-3 font-medium">渠道</th>
            <th class="pb-2.5 pr-3 font-medium text-right">Tokens</th>
            <th class="pb-2.5 pr-3 font-medium text-right">TTFT</th>
            <th class="pb-2.5 pr-3 font-medium text-right">耗时</th>
            <th class="pb-2.5 pr-3 font-medium text-right">费用</th>
            <th class="pb-2.5 pr-3 font-medium">状态</th>
            <th class="pb-2.5 font-medium text-right">详情</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((l) => `
            <tr class="row-hover border-b border-zinc-800/50" data-lid="${l.id}">
              <td class="py-2.5 pr-3 font-mono text-[10px] text-zinc-500">${shortTime(l.created_at)}</td>
              <td class="py-2.5 pr-3 font-mono text-[10px] text-zinc-500"><div class="max-w-[120px] truncate">${l.request_id}</div></td>
              <td class="py-2.5 pr-3"><div class="font-semibold">#${l.user_id}</div><div class="font-mono text-[10px] text-zinc-500">${l.model}</div></td>
              <td class="py-2.5 pr-3 text-zinc-400">${l.channel_name || ('#' + l.channel_id)}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-300">${l.total_tokens.toLocaleString()}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-300">${l.ttft_ms ? l.ttft_ms + 'ms' : '—'}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-300">${(l.duration_ms / 1000).toFixed(1)}s</td>
              <td class="py-2.5 pr-3 text-right font-mono font-semibold text-cyan-400">$${l.total_cost}</td>
              <td class="py-2.5 pr-3">${l.status === 'success' ? Badge('成功', 'success') : Badge(l.error_code || '失败', 'error')}</td>
              <td class="py-2.5 text-right"><button data-lact="detail" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">查看</button></td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function bindLogActions() {
  document.querySelectorAll('#log-list tr[data-lid]').forEach((tr) => {
    const id = Number(tr.dataset.lid);
    tr.querySelector('[data-lact="detail"]').addEventListener('click', () => logDetail(id));
  });
}

async function logDetail(id) {
  const modal = openModal({ title: '请求详情', width: 'max-w-2xl', bodyHTML: `<div id="log-detail" class="py-6 text-center text-xs text-zinc-500">加载中…</div>` });
  const body = modal.querySelector('#log-detail');
  try {
    const l = await adminGet(`/usage-logs/${id}`);
    const row = (k, v) => `<div class="flex items-start justify-between gap-4 border-b border-zinc-800/60 py-2"><span class="text-zinc-500">${k}</span><span class="text-right font-mono text-zinc-200">${v ?? '—'}</span></div>`;
    body.innerHTML = `
      <div class="text-xs">
        ${row('Request ID', l.request_id)}
        ${row('用户 / Key', `#${l.user_id} / #${l.api_key_id}`)}
        ${row('渠道', `${l.channel_name || ('#' + l.channel_id)}`)}
        ${row('模型 / 上游模型', `${l.model} / ${l.upstream_model || '—'}`)}
        ${row('输入 / 输出 / 缓存 tokens', `${l.input_tokens} / ${l.output_tokens} / ${l.cached_input_tokens}`)}
        ${row('单价 输入 / 输出', `$${l.unit_price_input_per_1m} / $${l.unit_price_output_per_1m}`)}
        ${row('费用', '$' + l.total_cost)}
        ${row('耗时 / TTFT', `${l.duration_ms}ms / ${l.ttft_ms != null ? l.ttft_ms + 'ms' : '—'}`)}
        ${row('状态 / 错误码', `${l.status} / ${l.error_code || '—'}`)}
        ${row('客户端 IP', l.client_ip)}
        ${row('时间', l.created_at)}
      </div>`;
  } catch (err) {
    body.innerHTML = `<div class="py-6 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}
