/* 视图：周期配额（用户/Key，UTC 自然日/月） */
let QUOTA_CACHE = [];

async function renderQuotasView() {
  const bento = document.getElementById('bento');
  bento.append(Card('col-span-12', `
    ${CardHeader({
      title: '周期配额',
      desc: 'quota_policies · 用户与 Key 必须同时满足 · UTC 自然周期',
      action: '<div class="flex gap-2"><button id="q-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] text-zinc-400">刷新</button><button id="q-new" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">新建配额</button></div>',
    })}
    <div id="q-list" class="py-10 text-center text-xs text-zinc-500">加载中…</div>`));
  document.getElementById('q-refresh').addEventListener('click', loadQuotas);
  document.getElementById('q-new').addEventListener('click', () => quotaForm(null));
  await loadQuotas();
}

async function loadQuotas() {
  const node = document.getElementById('q-list');
  if (!node) return;
  try {
    const [policies, usage] = await Promise.all([
      adminGet('/quota-policies', { page: 1, page_size: 100 }),
      adminGet('/quota-usage', { page: 1, page_size: 100 }),
    ]);
    QUOTA_CACHE = policies?.list || [];
    const buckets = new Map((usage?.list || []).map((item) => [Number(item.policy_id), item]));
    if (!QUOTA_CACHE.length) {
      node.innerHTML = '<div class="py-10 text-center text-xs text-zinc-500">暂无周期配额</div>';
      return;
    }
    node.innerHTML = `<div class="overflow-x-auto"><table class="w-full text-left text-xs"><thead><tr class="border-b border-zinc-800 text-zinc-500"><th class="pb-2">策略</th><th>作用域</th><th>周期</th><th>Token used / reserved / limit</th><th>费用 used / reserved / limit</th><th>状态</th><th class="text-right">操作</th></tr></thead><tbody>${QUOTA_CACHE.map((p) => {
      const b = buckets.get(Number(p.id)) || {};
      return `<tr class="border-b border-zinc-800/50" data-qid="${p.id}"><td class="py-3 font-semibold">${p.policy_name}</td><td class="font-mono text-zinc-400">${p.scope_type} #${p.scope_id}</td><td>${p.period_type === 'day' ? 'UTC 日' : 'UTC 月'}</td><td class="font-mono text-cyan-400">${b.used_tokens || 0} / ${b.reserved_tokens || 0} / ${p.token_limit ?? '∞'}</td><td class="font-mono text-emerald-400">${b.used_cost || '0.000000'} / ${b.reserved_cost || '0.000000'} / ${p.cost_limit ?? '∞'}</td><td>${p.enabled ? Badge('启用','success') : Badge('停用','neutral')}</td><td class="text-right"><button data-qact="edit" class="btn btn-ghost rounded border border-zinc-800 px-2 py-1">编辑</button> <button data-qact="del" class="btn btn-ghost rounded border border-rose-500/30 px-2 py-1 text-rose-400">删除</button></td></tr>`;
    }).join('')}</tbody></table></div>`;
    bindQuotaActions();
  } catch (err) {
    node.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function bindQuotaActions() {
  document.querySelectorAll('#q-list tr[data-qid]').forEach((row) => {
    const policy = QUOTA_CACHE.find((item) => Number(item.id) === Number(row.dataset.qid));
    row.querySelector('[data-qact="edit"]').addEventListener('click', () => quotaForm(policy));
    row.querySelector('[data-qact="del"]').addEventListener('click', () => Confirm(`删除配额「${policy.policy_name}」？`, async () => {
      await adminSend('DELETE', `/quota-policies/${policy.id}`);
      Toast('配额已删除', 'success');
      loadQuotas();
    }));
  });
}

function quotaForm(policy) {
  const isNew = !policy;
  openModal({
    title: isNew ? '新建周期配额' : `编辑配额 · ${policy.policy_name}`,
    submitText: isNew ? '创建' : '保存',
    fields: [
      { name: 'policy_name', label: '策略名', type: 'text', value: policy?.policy_name || '', required: true },
      { name: 'scope_type', label: '作用域', type: 'select', value: policy?.scope_type || 'user', options: [{value:'user',label:'用户'},{value:'api_key',label:'API Key'}] },
      { name: 'scope_id', label: '用户 / Key ID', type: 'number', value: policy?.scope_id || 1 },
      { name: 'period_type', label: 'UTC 周期', type: 'select', value: policy?.period_type || 'day', options: [{value:'day',label:'自然日'},{value:'month',label:'自然月'}] },
      { name: 'token_limit', label: 'Token 限额（可留空）', type: 'number', value: policy?.token_limit ?? '' },
      { name: 'cost_limit', label: '费用限额（可留空）', type: 'text', value: policy?.cost_limit ?? '' },
      { name: 'enabled', label: '启用', type: 'switch', value: policy?.enabled ?? true },
    ],
    onSubmit: async (v) => {
      const body = { policy_name: v.policy_name, enabled: v.enabled };
      if (isNew) Object.assign(body, { scope_type: v.scope_type, scope_id: v.scope_id, period_type: v.period_type });
      if (v.token_limit !== '') body.token_limit = Number(v.token_limit);
      if (v.cost_limit !== '') body.cost_limit = String(v.cost_limit);
      if (isNew) await adminSend('POST', '/quota-policies', body);
      else await adminSend('PUT', `/quota-policies/${policy.id}`, body);
      Toast(isNew ? '配额已创建' : '配额已更新', 'success');
      loadQuotas();
    },
  });
}
