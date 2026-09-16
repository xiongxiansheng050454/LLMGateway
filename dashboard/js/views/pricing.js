/* 视图：计费定价（可管理：设置/覆盖/删除） */
let PRICE_CACHE = [];

async function renderPricingView() {
  const bento = document.getElementById('bento');
  const card = Card('col-span-12', `
    ${CardHeader({
      title: '渠道×模型定价',
      desc: 'model_pricing · 单位：美元 / 1M tokens',
      action: `<div class="flex gap-2">
        <button id="p-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">刷新</button>
        <button id="p-new" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">设置单价</button>
      </div>`,
    })}
    <div id="pricing-body" class="py-10 text-center text-xs text-zinc-500">加载中…</div>`);
  bento.append(card);
  document.getElementById('p-new').addEventListener('click', () => pricingForm(null));
  document.getElementById('p-refresh').addEventListener('click', loadPricing);
  await loadPricing();
}

async function loadPricing() {
  const node = document.getElementById('pricing-body');
  if (!node) return;
  try {
    const data = await adminGet('/pricing', { page: 1, page_size: 100 });
    const rows = data?.list || [];
    PRICE_CACHE = rows;
    node.innerHTML = '';
    if (rows.length) {
      node.innerHTML = pricingAdminTableHTML(rows);
      bindPricingActions();
    } else {
      node.append(EmptyState({ title: '暂无定价', desc: '为渠道×模型设置单价后才能计费', actionLabel: '设置单价', onAction: () => pricingForm(null), icon: 'wallet' }));
    }
  } catch (err) {
    node.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function pricingAdminTableHTML(rows) {
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-3 font-medium">渠道</th>
            <th class="pb-2.5 pr-3 font-medium">对外模型</th>
            <th class="pb-2.5 pr-3 font-medium">上游模型</th>
            <th class="pb-2.5 pr-3 font-medium text-right">输入 / 1M</th>
            <th class="pb-2.5 pr-3 font-medium text-right">输出 / 1M</th>
            <th class="pb-2.5 pr-3 font-medium text-right">缓存输入 / 1M</th>
            <th class="pb-2.5 pr-3 font-medium">币种</th>
            <th class="pb-2.5 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((p) => `
            <tr class="row-hover border-b border-zinc-800/50" data-pid="${p.id}">
              <td class="py-2.5 pr-3">${p.channel_name || ('#' + p.channel_id)}</td>
              <td class="py-2.5 pr-3 font-semibold">${p.model_name}</td>
              <td class="py-2.5 pr-3 font-mono text-[10px] text-zinc-500">${p.upstream_model || '—'}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-300">$${p.input_price_per_1m}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-300">$${p.output_price_per_1m}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-500">${p.cached_input_price_per_1m ? '$' + p.cached_input_price_per_1m : '—'}</td>
              <td class="py-2.5 pr-3 text-zinc-400">${p.currency}</td>
              <td class="py-2.5 text-right whitespace-nowrap">
                <button data-pact="edit" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">编辑</button>
                <button data-pact="del" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
              </td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function bindPricingActions() {
  document.querySelectorAll('#pricing-body tr[data-pid]').forEach((tr) => {
    const id = Number(tr.dataset.pid);
    const row = PRICE_CACHE.find((p) => p.id === id);
    if (!row) return;
    tr.querySelector('[data-pact="edit"]').addEventListener('click', () => pricingForm(row));
    tr.querySelector('[data-pact="del"]').addEventListener('click', () => Confirm(`删除「${row.channel_name || ('#' + row.channel_id)} / ${row.model_name}」的定价？`, async () => {
      await adminSend('DELETE', '/pricing', { channel_id: row.channel_id, model_name: row.model_name });
      Toast('已删除', 'success');
      loadPricing();
    }));
  });
}

async function pricingForm(existing) {
  let chans = [];
  try {
    chans = (await adminGet('/channels', { page: 1, page_size: 100 }))?.list || [];
  } catch (err) {
    Toast('加载渠道失败：' + err.message, 'error');
    return;
  }
  if (!chans.length) { Toast('请先创建渠道并添加模型映射', 'error'); return; }

  const chanOptions = chans.map((c) => `<option value="${c.id}" ${existing && existing.channel_id === c.id ? 'selected' : ''}>#${c.id} ${c.name}</option>`).join('');
  const modal = openModal({
    title: existing ? `编辑定价 · ${existing.model_name}` : '设置单价',
    submitText: '保存',
    bodyHTML: `
      <div class="grid gap-3">
        <label class="block"><span class="mb-1 block text-xs font-medium text-zinc-400">渠道</span>
          <select id="p_channel" class="form-input">${chanOptions}</select></label>
        <label class="block"><span class="mb-1 block text-xs font-medium text-zinc-400">对外模型</span>
          <select id="p_model" class="form-input"></select></label>
        <label class="block"><span class="mb-1 block text-xs font-medium text-zinc-400">输入价（$ / 1M）</span>
          <input id="p_in" class="form-input" value="${existing?.input_price_per_1m || ''}" placeholder="如 0.10000000"/></label>
        <label class="block"><span class="mb-1 block text-xs font-medium text-zinc-400">输出价（$ / 1M）</span>
          <input id="p_out" class="form-input" value="${existing?.output_price_per_1m || ''}" placeholder="如 0.20000000"/></label>
        <label class="block"><span class="mb-1 block text-xs font-medium text-zinc-400">缓存输入价（$ / 1M，可选）</span>
          <input id="p_cache" class="form-input" value="${existing?.cached_input_price_per_1m || ''}" placeholder="留空继承输入价"/></label>
      </div>`,
    onSubmit: async () => {
      const channelId = Number(document.getElementById('p_channel').value);
      const model = document.getElementById('p_model').value;
      const inputPrice = document.getElementById('p_in').value.trim();
      const outputPrice = document.getElementById('p_out').value.trim();
      const cachePrice = document.getElementById('p_cache').value.trim();
      if (!channelId || !model) throw new Error('请选择渠道与模型');
      if (!inputPrice || !outputPrice) throw new Error('请输入输入价与输出价');
      const body = { channel_id: channelId, model_name: model, input_price_per_1m: inputPrice, output_price_per_1m: outputPrice, currency: 'USD' };
      if (cachePrice) body.cached_input_price_per_1m = cachePrice;
      await adminSend('POST', '/pricing', body);
      Toast('定价已保存', 'success');
      loadPricing();
    },
  });

  const chanSel = modal.querySelector('#p_channel');
  const modelSel = modal.querySelector('#p_model');

  async function loadModels(channelId, selected) {
    modelSel.innerHTML = '<option value="">加载中…</option>';
    try {
      const d = await adminGet(`/channels/${channelId}/models`, {});
      const list = d?.list || [];
      modelSel.innerHTML = list.length
        ? list.map((m) => `<option value="${m.model_name}" ${selected === m.model_name ? 'selected' : ''}>${m.model_name} → ${m.upstream_model}</option>`).join('')
        : '<option value="">该渠道暂无模型映射</option>';
    } catch (err) {
      modelSel.innerHTML = `<option value="">加载失败：${err.message}</option>`;
    }
  }

  chanSel.addEventListener('change', () => loadModels(chanSel.value));
  await loadModels(chanSel.value, existing?.model_name);
}
