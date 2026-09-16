/* 视图：限流规则（可管理：CRUD / 启停） */
let LIM_PAGE = 1;
const LIM_PAGE_SIZE = 20;
let LIM_CACHE = [];

const TARGET_TYPES = [
  { value: 'global', label: 'global' }, { value: 'user', label: 'user' },
  { value: 'api_key', label: 'api_key' }, { value: 'model', label: 'model' }, { value: 'channel', label: 'channel' },
];
const METRICS = [
  { value: 'rpm', label: 'rpm (次/分)' }, { value: 'tpm', label: 'tpm (token/分)' },
  { value: 'rpd', label: 'rpd (次/日)' }, { value: 'tpd', label: 'tpd (token/日)' }, { value: 'concurrency', label: 'concurrency (并发)' },
];

async function renderLimitsView() {
  const bento = document.getElementById('bento');
  const card = Card('col-span-12', `
    ${CardHeader({
      title: '限流规则',
      desc: 'rate_limit_rules · rpm / tpm / rpd / tpd / 并发',
      action: `<div class="flex gap-2">
        <button id="l-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">刷新</button>
        <button id="l-new" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">新建规则</button>
      </div>`,
    })}
    <div id="l-list" class="py-10 text-center text-xs text-zinc-500">加载中…</div>
    <div id="l-pager"></div>`);
  bento.append(card);
  document.getElementById('l-new').addEventListener('click', () => ruleForm(null));
  document.getElementById('l-refresh').addEventListener('click', () => loadLimits(LIM_PAGE));
  await loadLimits(1);
}

async function loadLimits(page) {
  LIM_PAGE = page || 1;
  const node = document.getElementById('l-list');
  if (!node) return;
  try {
    const d = await adminGet('/rate-limits', { page: LIM_PAGE, page_size: LIM_PAGE_SIZE });
    const rows = d?.list || [];
    LIM_CACHE = rows;
    node.innerHTML = '';
    if (rows.length) {
      node.innerHTML = rulesTableHTML(rows);
      bindRuleActions();
    } else {
      node.append(EmptyState({ title: '暂无规则', desc: '创建限流规则以保护网关与上游', actionLabel: '新建规则', onAction: () => ruleForm(null), icon: 'gauge' }));
    }
    const pager = document.getElementById('l-pager');
    pager.innerHTML = '';
    pager.append(Pagination({ page: LIM_PAGE, pageSize: LIM_PAGE_SIZE, total: d?.total ?? rows.length, onChange: loadLimits }));
  } catch (err) {
    node.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function rulesTableHTML(rows) {
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-3 font-medium">规则名</th>
            <th class="pb-2.5 pr-3 font-medium">作用域</th>
            <th class="pb-2.5 pr-3 font-medium">目标</th>
            <th class="pb-2.5 pr-3 font-medium">指标</th>
            <th class="pb-2.5 pr-3 font-medium text-right">限额/窗口</th>
            <th class="pb-2.5 pr-3 font-medium">动作</th>
            <th class="pb-2.5 pr-3 font-medium text-right">优先级</th>
            <th class="pb-2.5 pr-3 font-medium">状态</th>
            <th class="pb-2.5 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((r) => `
            <tr class="row-hover border-b border-zinc-800/50" data-rid="${r.id}">
              <td class="py-2.5 pr-3 font-semibold">${r.rule_name}</td>
              <td class="py-2.5 pr-3 text-zinc-400">${r.target_type}</td>
              <td class="py-2.5 pr-3 font-mono text-zinc-400">${r.target_value}</td>
              <td class="py-2.5 pr-3 font-mono text-cyan-400">${r.metric}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-300">${r.limit_value} / ${r.window_seconds}s</td>
              <td class="py-2.5 pr-3">${r.action === 'queue' ? Badge('queue', 'warning') : Badge('reject', 'neutral')}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-400">${r.priority}</td>
              <td class="py-2.5 pr-3">${r.enabled ? Badge('启用', 'success') : Badge('停用', 'neutral')}</td>
              <td class="py-2.5 text-right whitespace-nowrap">
                <button data-ract="edit" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">编辑</button>
                <button data-ract="toggle" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">${r.enabled ? '停用' : '启用'}</button>
                <button data-ract="del" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
              </td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function bindRuleActions() {
  document.querySelectorAll('#l-list tr[data-rid]').forEach((tr) => {
    const id = Number(tr.dataset.rid);
    const rule = LIM_CACHE.find((r) => r.id === id);
    if (!rule) return;
    tr.querySelector('[data-ract="edit"]').addEventListener('click', () => ruleForm(rule));
    tr.querySelector('[data-ract="toggle"]').addEventListener('click', async () => {
      try {
        await adminSend('PUT', `/rate-limits/${rule.id}`, { enabled: !rule.enabled });
        Toast('状态已更新', 'success');
        loadLimits(LIM_PAGE);
      } catch (e) { Toast(e.message, 'error'); }
    });
    tr.querySelector('[data-ract="del"]').addEventListener('click', () => Confirm(`删除规则「${rule.rule_name}」？`, async () => {
      await adminSend('DELETE', `/rate-limits/${rule.id}`);
      Toast('已删除', 'success');
      loadLimits(LIM_PAGE);
    }));
  });
}

function ruleForm(rule) {
  const isNew = !rule;
  const extras = rule?.extras || {};
  openModal({
    title: isNew ? '新建限流规则' : `编辑规则 · ${rule.rule_name}`,
    submitText: isNew ? '创建' : '保存',
    fields: [
      { name: 'rule_name', label: '规则名', type: 'text', value: rule?.rule_name || '', required: true },
      { name: 'target_type', label: '作用域', type: 'select', value: rule?.target_type || 'user', options: TARGET_TYPES },
      { name: 'target_value', label: '目标值（* 通配）', type: 'text', value: rule?.target_value || '*', required: true },
      { name: 'metric', label: '指标', type: 'select', value: rule?.metric || 'rpm', options: METRICS },
      { name: 'limit_value', label: '限额', type: 'number', value: rule?.limit_value ?? 600 },
      { name: 'window_seconds', label: '窗口（秒）', type: 'number', value: rule?.window_seconds ?? 60 },
      { name: 'action', label: '动作', type: 'select', value: rule?.action || 'reject', options: [{ value: 'reject', label: 'reject' }, { value: 'queue', label: 'queue' }] },
      { name: 'priority', label: '优先级（越小越优先）', type: 'number', value: rule?.priority ?? 100 },
      { name: 'queue_timeout_seconds', label: 'queue 超时（秒，仅 queue）', type: 'number', value: extras.queue_timeout_seconds ?? '' },
      { name: 'enabled', label: '启用', type: 'switch', value: rule?.enabled ?? true },
    ],
    onSubmit: async (v) => {
      const body = {
        rule_name: v.rule_name,
        target_type: v.target_type,
        target_value: v.target_value,
        metric: v.metric,
        limit_value: v.limit_value,
        window_seconds: v.window_seconds,
        action: v.action,
        priority: v.priority ?? 0,
        enabled: v.enabled,
      };
      if (v.action === 'queue') {
        if (!v.queue_timeout_seconds) throw new Error('queue 动作需填写超时秒数');
        body.extras = { queue_timeout_seconds: Number(v.queue_timeout_seconds) };
      }
      if (isNew) await adminSend('POST', '/rate-limits', body);
      else await adminSend('PUT', `/rate-limits/${rule.id}`, body);
      Toast(isNew ? '规则已创建' : '规则已更新', 'success');
      loadLimits(LIM_PAGE);
    },
  });
}
