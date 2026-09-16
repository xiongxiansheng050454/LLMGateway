/* 视图：渠道管理（可管理：CRUD / 启停 / 余额 / 测试 / 模型映射） */
let CHAN_PAGE = 1;
const CHAN_PAGE_SIZE = 20;
let CHAN_CACHE = [];

async function renderChannelsView() {
  const bento = document.getElementById('bento');
  const card = Card('col-span-12', `
    ${CardHeader({
      title: '渠道管理',
      desc: '上游渠道 · 状态 / 权重 / 优先级 / 余额',
      action: `<div class="flex gap-2">
        <button id="ch-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">刷新</button>
        <button id="ch-new" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">新建渠道</button>
      </div>`,
    })}
    <div id="ch-list" class="py-10 text-center text-xs text-zinc-500">加载中…</div>
    <div id="ch-pager"></div>`);
  bento.append(card);
  document.getElementById('ch-new').addEventListener('click', () => channelForm(null));
  document.getElementById('ch-refresh').addEventListener('click', () => loadChannels(CHAN_PAGE));
  await loadChannels(1);
}

async function loadChannels(page) {
  CHAN_PAGE = page || 1;
  const listNode = document.getElementById('ch-list');
  if (!listNode) return;
  try {
    const d = await adminGet('/channels', { page: CHAN_PAGE, page_size: CHAN_PAGE_SIZE });
    const rows = d?.list || [];
    CHAN_CACHE = rows;
    listNode.innerHTML = '';
    if (rows.length) {
      listNode.innerHTML = channelsTableHTML(rows);
      bindChannelActions();
    } else {
      listNode.append(EmptyState({ title: '暂无渠道', desc: '创建第一个上游渠道后即可路由与计费', actionLabel: '新建渠道', onAction: () => channelForm(null), icon: 'channels' }));
    }
    const pager = document.getElementById('ch-pager');
    pager.innerHTML = '';
    pager.append(Pagination({ page: CHAN_PAGE, pageSize: CHAN_PAGE_SIZE, total: d?.total ?? rows.length, onChange: loadChannels }));
  } catch (err) {
    listNode.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function channelsTableHTML(rows) {
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-3 font-medium">ID</th>
            <th class="pb-2.5 pr-3 font-medium">名称</th>
            <th class="pb-2.5 pr-3 font-medium">Base URL</th>
            <th class="pb-2.5 pr-3 font-medium">状态</th>
            <th class="pb-2.5 pr-3 font-medium">权重/优先级</th>
            <th class="pb-2.5 pr-3 font-medium">余额</th>
            <th class="pb-2.5 pr-3 font-medium text-right">模型</th>
            <th class="pb-2.5 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((ch) => `
            <tr class="row-hover border-b border-zinc-800/50" data-id="${ch.id}">
              <td class="py-2.5 pr-3 font-mono text-zinc-500">#${ch.id}</td>
              <td class="py-2.5 pr-3 font-semibold">${ch.name}</td>
              <td class="py-2.5 pr-3 font-mono text-[10px] text-zinc-500"><div class="max-w-[220px] truncate">${ch.base_url}</div></td>
              <td class="py-2.5 pr-3">${ch.status === 1 ? Badge('启用', 'success') : Badge('停用', 'neutral')}</td>
              <td class="py-2.5 pr-3 font-mono text-zinc-400">${ch.weight} / ${ch.priority}</td>
              <td class="py-2.5 pr-3 font-mono text-zinc-300">${ch.balance == null ? '不限' : '$' + Number(ch.balance).toFixed(4)}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-400">${ch.model_count ?? '—'}</td>
              <td class="py-2.5 text-right whitespace-nowrap">
                <button data-act="edit" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">编辑</button>
                <button data-act="balance" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">余额</button>
                <button data-act="models" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">映射</button>
                <button data-act="test" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">测试</button>
                <button data-act="toggle" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">${ch.status === 1 ? '停用' : '启用'}</button>
                <button data-act="delete" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
              </td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function bindChannelActions() {
  document.querySelectorAll('#ch-list tr[data-id]').forEach((tr) => {
    const id = Number(tr.dataset.id);
    const row = CHAN_CACHE.find((c) => c.id === id);
    if (!row) return;
    tr.querySelector('[data-act="edit"]').addEventListener('click', () => channelForm(row));
    tr.querySelector('[data-act="balance"]').addEventListener('click', () => channelBalanceForm(row));
    tr.querySelector('[data-act="models"]').addEventListener('click', () => channelMappings(row));
    tr.querySelector('[data-act="test"]').addEventListener('click', () => channelTest(row));
    tr.querySelector('[data-act="toggle"]').addEventListener('click', () => toggleChannel(row));
    tr.querySelector('[data-act="delete"]').addEventListener('click', () => deleteChannel(row));
  });
}

/* ---------- 渠道表单 ---------- */
function channelForm(ch) {
  const isNew = !ch;
  openModal({
    title: isNew ? '新建渠道' : `编辑渠道 · ${ch.name}`,
    submitText: isNew ? '创建' : '保存',
    fields: [
      { name: 'name', label: '渠道名称', type: 'text', value: ch?.name || '', required: true },
      { name: 'base_url', label: 'Base URL', type: 'text', value: ch?.base_url || '', placeholder: 'https://api.example.com', required: true },
      { name: 'api_key', label: '上游 API Key', type: 'text', value: '', placeholder: isNew ? 'sk-...' : '留空表示不修改', help: '加密存储，仅转发时解密' },
      { name: 'auth_type', label: '认证方式', type: 'select', value: ch?.auth_type || 'bearer', options: [{ value: 'bearer', label: 'bearer' }] },
      { name: 'priority', label: '优先级（越大越优先）', type: 'number', value: ch?.priority ?? 0 },
      { name: 'weight', label: '权重', type: 'number', value: ch?.weight ?? 100 },
      { name: 'balance', label: '余额（USD）', type: 'text', value: ch?.balance ?? '', placeholder: '留空=不限', help: '本地估算余额，按请求扣减 / 手动核对' },
      { name: 'status', label: '状态', type: 'select', value: String(ch?.status ?? 1), options: [{ value: '1', label: '启用' }, { value: '0', label: '停用' }] },
    ],
    onSubmit: async (v) => {
      const body = {
        name: v.name,
        base_url: v.base_url,
        auth_type: v.auth_type,
        priority: v.priority ?? 0,
        weight: v.weight ?? 100,
        status: Number(v.status),
      };
      if (v.api_key) body.api_key = v.api_key;
      if (isNew) {
        if (!v.api_key) throw new Error('api_key 必填');
        if (v.balance !== '' && v.balance != null) body.balance = v.balance;
        await adminSend('POST', '/channels', body);
      } else {
        body.balance = v.balance === '' ? '' : v.balance;
        await adminSend('PUT', `/channels/${ch.id}`, body);
      }
      Toast(isNew ? '渠道已创建' : '渠道已更新', 'success');
      loadChannels(CHAN_PAGE);
    },
  });
}

function channelBalanceForm(ch) {
  openModal({
    title: `调整余额 · ${ch.name}`,
    submitText: '保存',
    fields: [
      { name: 'balance', label: '设为绝对值（USD）', type: 'text', placeholder: '留空不改' },
      { name: 'delta', label: '增减（USD，可负）', type: 'text', placeholder: '如 -12.34' },
      { name: 'description', label: '备注', type: 'text' },
    ],
    onSubmit: async (v) => {
      const body = {};
      if (v.balance) body.balance = v.balance;
      if (v.delta) body.delta = v.delta;
      if (v.description) body.description = v.description;
      if (!body.balance && !body.delta) throw new Error('需填写 balance 或 delta');
      await adminSend('PUT', `/channels/${ch.id}/balance`, body);
      Toast('余额已更新', 'success');
      loadChannels(CHAN_PAGE);
    },
  });
}

function toggleChannel(ch) {
  const next = ch.status === 1 ? 0 : 1;
  Confirm(`确认${next === 1 ? '启用' : '停用'}渠道「${ch.name}」？`, async () => {
    await adminSend('PUT', `/channels/${ch.id}/status`, { status: next });
    Toast(next === 1 ? '已启用' : '已停用', 'success');
    loadChannels(CHAN_PAGE);
  });
}

function deleteChannel(ch) {
  Confirm(`确认删除渠道「${ch.name}」？将级联删除其模型映射与定价，历史用量保留。`, async () => {
    await adminSend('DELETE', `/channels/${ch.id}`);
    Toast('渠道已删除', 'success');
    loadChannels(CHAN_PAGE);
  });
}

function channelTest(ch) {
  openModal({
    title: `连通性测试 · ${ch.name}`,
    submitText: '开始测试',
    fields: [{ name: 'check_all', type: 'switch', value: true, switchLabel: '检查全部启用模型（默认开启）' }],
    onSubmit: async (v) => {
      const r = await adminSend('POST', `/channels/${ch.id}/test`, { check_all: v.check_all !== false });
      setTimeout(() => openModal({ title: `测试结果 · ${ch.name}`, bodyHTML: testResultHTML(r) }), 60);
    },
  });
}

function testResultHTML(r) {
  const items = r && r.list ? r.list : [r];
  return `<div class="space-y-2">${items.map((it) => `
    <div class="rounded-lg border border-zinc-800 px-3 py-2.5 text-xs">
      <div class="flex items-center justify-between">
        <div>
          <div class="font-semibold">${it.model_alias} → <span class="font-mono text-zinc-400">${it.upstream_model}</span></div>
          <div class="mt-0.5 font-mono text-[10px] text-zinc-500">HTTP ${it.http_status} · ${it.latency_ms}ms</div>
        </div>
        <div>${it.ok ? Badge('正常', 'success') : Badge('失败', 'error')}</div>
      </div>
      ${it.error ? `<div class="mt-1.5 font-mono text-[10px] text-rose-400">${it.error}</div>` : ''}
    </div>`).join('')}</div>`;
}

/* ---------- 模型映射 ---------- */
async function channelMappings(ch) {
  const modal = openModal({
    title: `模型映射 · ${ch.name}`,
    width: 'max-w-3xl',
    bodyHTML: `<div id="map-body" class="py-6 text-center text-xs text-zinc-500">加载中…</div>`,
  });
  await reloadMappings(ch, modal);
}

async function reloadMappings(ch, modal) {
  const body = modal.querySelector('#map-body');
  if (!body) return;
  try {
    const d = await adminGet(`/channels/${ch.id}/models`, {});
    const rows = d?.list || [];
    const existing = new Set(rows.map((m) => m.model_name));
    body.innerHTML = `
      <div class="mb-3 flex justify-end gap-2">
        <button id="map-remote" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">拉取远端模型</button>
        <button id="map-add" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">手动添加映射</button>
      </div>
      ${rows.length ? `
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead><tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
              <th class="pb-2 pr-3 font-medium">对外模型名</th>
              <th class="pb-2 pr-3 font-medium">上游真实模型名</th>
              <th class="pb-2 pr-3 font-medium">状态</th>
              <th class="pb-2 font-medium text-right">操作</th>
            </tr></thead>
            <tbody>
              ${rows.map((m) => `
                <tr class="row-hover border-b border-zinc-800/50">
                  <td class="py-2 pr-3 font-semibold">${m.model_name}</td>
                  <td class="py-2 pr-3 font-mono text-zinc-400">${m.upstream_model}</td>
                  <td class="py-2 pr-3">${m.enabled ? Badge('启用', 'success') : Badge('停用', 'neutral')}</td>
                  <td class="py-2 text-right whitespace-nowrap">
                    <button data-mid="${m.id}" data-mact="edit" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">编辑</button>
                    <button data-mid="${m.id}" data-mact="del" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
                  </td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>` : `<div class="py-8 text-center text-xs text-zinc-500">暂无映射</div>`}
      <div id="map-remote-list" class="mt-3"></div>`;

    body.querySelector('#map-add').addEventListener('click', () => mappingForm(ch, null, () => channelMappings(ch)));
    body.querySelector('#map-remote').addEventListener('click', () => remoteModels(ch, body.querySelector('#map-remote-list'), modal, existing));
    rows.forEach((m) => {
      body.querySelector(`[data-mid="${m.id}"][data-mact="edit"]`).addEventListener('click', () => mappingForm(ch, m, () => channelMappings(ch)));
      body.querySelector(`[data-mid="${m.id}"][data-mact="del"]`).addEventListener('click', () => Confirm(`删除映射「${m.model_name}」？`, async () => {
        await adminSend('DELETE', `/channels/${ch.id}/models/${m.id}`);
        Toast('映射已删除', 'success');
        channelMappings(ch);
      }));
    });
    // 默认自动拉取远端模型，方便一键映射（别名=真实名）
    await remoteModels(ch, body.querySelector('#map-remote-list'), modal, existing);
  } catch (err) {
    body.innerHTML = `<div class="py-6 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function mappingForm(ch, m, onDone, preset) {
  const isNew = !m;
  const base = m || preset || {};
  const modal = openModal({
    title: isNew ? `添加映射 · ${ch.name}` : `编辑映射 · ${m.model_name}`,
    submitText: isNew ? '添加' : '保存',
    fields: [
      { name: 'model_name', label: '对外模型名（网关别名）', type: 'text', value: base.model_name || '', readonly: !isNew, help: isNew ? '默认与真实模型名相同；下游请求使用的模型名' : '创建后不可改，如需变更请删除重建' },
      { name: 'upstream_model', label: '上游真实模型名', type: 'text', value: base.upstream_model || base.model_name || '', required: true },
      { name: 'enabled', label: '启用', type: 'switch', value: base.enabled ?? true },
    ],
    onSubmit: async (v) => {
      if (isNew) {
        await adminSend('POST', `/channels/${ch.id}/models`, { model_name: v.model_name, upstream_model: v.upstream_model, enabled: v.enabled });
      } else {
        await adminSend('PUT', `/channels/${ch.id}/models/${m.id}`, { upstream_model: v.upstream_model, enabled: v.enabled });
      }
      Toast('已保存', 'success');
      if (onDone) onDone();
    },
  });
  if (isNew && modal) {
    const nameInput = modal.querySelector('[data-name="model_name"]');
    const upInput = modal.querySelector('[data-name="upstream_model"]');
    if (nameInput && upInput) {
      nameInput.addEventListener('input', () => {
        if (!upInput.dataset.touched) upInput.value = nameInput.value;
      });
      upInput.addEventListener('input', () => { upInput.dataset.touched = '1'; });
    }
  }
}

async function remoteModels(ch, container, modal, existing) {
  if (!container) return;
  existing = existing || new Set();
  container.innerHTML = `<div class="text-xs text-zinc-500">拉取远端模型中…</div>`;
  try {
    const d = await adminSend('POST', `/channels/${ch.id}/remote-models`, {});
    if (!d || d.ok === false) {
      container.innerHTML = `<div class="rounded-lg border border-rose-500/30 px-3 py-2 text-xs text-rose-400">${d?.error || '拉取失败（可手动添加映射）'}</div>`;
      return;
    }
    const models = d.models || [];
    const unmapped = models.filter((x) => !existing.has(x.id));
    container.innerHTML = `
      <div class="rounded-lg border border-zinc-800 p-3">
        <div class="mb-2 flex items-center justify-between gap-2">
          <div class="text-xs font-semibold text-zinc-300">远端模型（${models.length}）· 别名=真实名</div>
          ${unmapped.length ? `<button id="map-batch" class="btn btn-primary rounded-lg px-3 py-1 text-[11px] font-bold">一键映射全部（${unmapped.length}）</button>` : ''}
        </div>
        <div class="flex flex-wrap gap-2">
          ${models.length ? models.map((x) => {
            const done = existing.has(x.id);
            return `<button data-model="${x.id}" class="btn btn-ghost rounded-lg border ${done ? 'border-emerald-400/30 text-emerald-400' : 'border-zinc-800 text-zinc-300'} px-2.5 py-1 font-mono text-[11px]" ${done ? 'disabled' : ''}>${x.id}${done ? ' ✓' : ''}</button>`;
          }).join('') : '<span class="text-xs text-zinc-500">无</span>'}
        </div>
      </div>`;
    const batchBtn = container.querySelector('#map-batch');
    if (batchBtn) batchBtn.addEventListener('click', () => batchMapModels(ch, unmapped.map((x) => x.id), existing, modal));
    container.querySelectorAll('[data-model]:not([disabled])').forEach((b) => b.addEventListener('click', () => {
      mappingForm(ch, null, () => channelMappings(ch), { model_name: b.dataset.model, upstream_model: b.dataset.model, enabled: true });
    }));
  } catch (err) {
    container.innerHTML = `<div class="rounded-lg border border-rose-500/30 px-3 py-2 text-xs text-rose-400">${err.message}</div>`;
  }
}

async function batchMapModels(ch, ids, existing, modal) {
  let ok = 0, fail = 0;
  for (const id of ids) {
    if (existing.has(id)) continue;
    try {
      await adminSend('POST', `/channels/${ch.id}/models`, { model_name: id, upstream_model: id, enabled: true });
      ok++;
    } catch {
      fail++;
    }
  }
  Toast(`已添加 ${ok} 个映射${fail ? ('，失败 ' + fail) : ''}`, fail ? 'error' : 'success');
  channelMappings(ch);
}
