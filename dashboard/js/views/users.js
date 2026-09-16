/* 视图：用户与 Key（可管理：用户 CRUD / 充值 / 余额流水 / Key CRUD） */
let USER_PAGE = 1;
const USER_PAGE_SIZE = 20;
let USER_CACHE = [];

async function renderUsersView() {
  const bento = document.getElementById('bento');

  const usersCard = Card('col-span-12', `
    ${CardHeader({
      title: '下游用户',
      desc: 'users · 余额 / 分组 / 状态',
      action: `<div class="flex gap-2">
        <button id="u-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">刷新</button>
        <button id="u-new" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">新建用户</button>
      </div>`,
    })}
    <div id="u-list" class="py-10 text-center text-xs text-zinc-500">加载中…</div>
    <div id="u-pager"></div>`);
  bento.append(usersCard);
  document.getElementById('u-new').addEventListener('click', () => userForm(null));
  document.getElementById('u-refresh').addEventListener('click', () => loadUsers(USER_PAGE));
  await loadUsers(1);

  const keysCard = Card('col-span-12', `
    ${CardHeader({
      title: '网关 Key',
      desc: 'client_api_keys · 仅存哈希，明文仅下发一次',
      action: `<button id="k-refresh" class="btn btn-ghost rounded-lg border border-zinc-800 px-3 py-1.5 text-[11px] font-semibold text-zinc-400">刷新</button>`,
    })}
    <div id="k-list" class="py-10 text-center text-xs text-zinc-500">加载中…</div>`);
  bento.append(keysCard);
  document.getElementById('k-refresh').addEventListener('click', loadKeys);
  await loadKeys();
}

async function loadUsers(page) {
  USER_PAGE = page || 1;
  const node = document.getElementById('u-list');
  if (!node) return;
  try {
    const d = await adminGet('/users', { page: USER_PAGE, page_size: USER_PAGE_SIZE });
    const rows = d?.list || [];
    USER_CACHE = rows;
    node.innerHTML = '';
    if (rows.length) {
      node.innerHTML = usersTableHTML(rows);
      bindUserActions();
    } else {
      node.append(EmptyState({ title: '暂无用户', desc: '创建下游用户并为其分配网关 Key', actionLabel: '新建用户', onAction: () => userForm(null), icon: 'users' }));
    }
    const pager = document.getElementById('u-pager');
    pager.innerHTML = '';
    pager.append(Pagination({ page: USER_PAGE, pageSize: USER_PAGE_SIZE, total: d?.total ?? rows.length, onChange: loadUsers }));
  } catch (err) {
    node.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function usersTableHTML(rows) {
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-3 font-medium">ID</th>
            <th class="pb-2.5 pr-3 font-medium">昵称</th>
            <th class="pb-2.5 pr-3 font-medium">分组</th>
            <th class="pb-2.5 pr-3 font-medium">状态</th>
            <th class="pb-2.5 pr-3 font-medium text-right">可用余额</th>
            <th class="pb-2.5 pr-3 font-medium text-right">冻结</th>
            <th class="pb-2.5 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((u) => `
            <tr class="row-hover border-b border-zinc-800/50" data-uid="${u.id}">
              <td class="py-2.5 pr-3 font-mono text-zinc-500">#${u.id}</td>
              <td class="py-2.5 pr-3 font-semibold">${u.nickname || '—'}</td>
              <td class="py-2.5 pr-3 text-zinc-400">${u.user_group}</td>
              <td class="py-2.5 pr-3">${userStatusBadge(u.status)}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-cyan-400">$${u.balance?.available_balance ?? '0.000000'}</td>
              <td class="py-2.5 pr-3 text-right font-mono text-zinc-500">$${u.balance?.frozen_balance ?? '0.000000'}</td>
              <td class="py-2.5 text-right whitespace-nowrap">
                <button data-uact="edit" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">编辑</button>
                <button data-uact="recharge" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">充值</button>
                <button data-uact="balance" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">余额</button>
                <button data-uact="keys" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">Key</button>
                <button data-uact="status" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">${u.status === 'active' ? '停用' : '启用'}</button>
                <button data-uact="delete" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
              </td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function userStatusBadge(status) {
  if (status === 'active') return Badge('启用', 'success');
  if (status === 'suspended') return Badge('停用', 'warning');
  return Badge(status || '未知', 'neutral');
}

function bindUserActions() {
  document.querySelectorAll('#u-list tr[data-uid]').forEach((tr) => {
    const uid = Number(tr.dataset.uid);
    const user = USER_CACHE.find((u) => u.id === uid);
    if (!user) return;
    tr.querySelector('[data-uact="edit"]').addEventListener('click', () => userForm(user));
    tr.querySelector('[data-uact="recharge"]').addEventListener('click', () => rechargeForm(user));
    tr.querySelector('[data-uact="balance"]').addEventListener('click', () => userBalanceModal(user));
    tr.querySelector('[data-uact="keys"]').addEventListener('click', () => userKeysModal(user));
    tr.querySelector('[data-uact="status"]').addEventListener('click', () => toggleUser(user));
    tr.querySelector('[data-uact="delete"]').addEventListener('click', () => deleteUser(user));
  });
}

function userForm(user) {
  const isNew = !user;
  openModal({
    title: isNew ? '新建用户' : `编辑用户 · #${user.id}`,
    submitText: isNew ? '创建' : '保存',
    fields: [
      { name: 'nickname', label: '昵称', type: 'text', value: user?.nickname || '' },
      { name: 'user_group', label: '分组', type: 'select', value: user?.user_group || 'default', options: [
        { value: 'default', label: 'default' }, { value: 'vip', label: 'vip' }, { value: 'enterprise', label: 'enterprise' },
      ] },
      { name: 'status', label: '状态', type: 'select', value: user?.status || 'active', options: [
        { value: 'active', label: 'active' }, { value: 'suspended', label: 'suspended' },
      ] },
      { name: 'password', label: '密码', type: 'text', placeholder: isNew ? '留空自动生成' : '留空不修改' },
    ],
    onSubmit: async (v) => {
      if (isNew) {
        const body = { nickname: v.nickname, user_group: v.user_group, status: v.status };
        if (v.password) body.password = v.password;
        const out = await adminSend('POST', '/users', body);
        if (out?.password_plaintext) Toast('已创建，初始密码：' + out.password_plaintext, 'success');
        else Toast('用户已创建', 'success');
      } else {
        const body = { nickname: v.nickname, user_group: v.user_group };
        if (v.password) body.password = v.password;
        await adminSend('PUT', `/users/${user.id}`, body);
        if (v.status !== user.status) await adminSend('PUT', `/users/${user.id}/status`, { status: v.status });
        Toast('用户已更新', 'success');
      }
      loadUsers(USER_PAGE);
    },
  });
}

function rechargeForm(user) {
  openModal({
    title: `充值 · ${user.nickname || ('#' + user.id)}`,
    submitText: '充值',
    fields: [
      { name: 'amount', label: '金额（USD）', type: 'text', placeholder: '如 50.00', required: true },
      { name: 'related_order_id', label: '订单号', type: 'text', placeholder: '可选，用于幂等' },
      { name: 'description', label: '备注', type: 'text' },
    ],
    onSubmit: async (v) => {
      const body = { amount: v.amount };
      if (v.related_order_id) body.related_order_id = v.related_order_id;
      if (v.description) body.description = v.description;
      const out = await adminSend('POST', `/users/${user.id}/recharge`, body);
      Toast(`充值成功，余额 $${out?.balance_after ?? ''}`, 'success');
      loadUsers(USER_PAGE);
    },
  });
}

async function userBalanceModal(user) {
  const modal = openModal({ title: `余额与流水 · ${user.nickname || ('#' + user.id)}`, width: 'max-w-2xl', bodyHTML: `<div id="ub-body" class="py-6 text-center text-xs text-zinc-500">加载中…</div>` });
  const body = modal.querySelector('#ub-body');
  try {
    const [bal, txs] = await Promise.all([
      adminGet(`/users/${user.id}/balance`),
      adminGet(`/users/${user.id}/balance-transactions`, { page: 1, page_size: 20 }),
    ]);
    const rows = txs?.list || [];
    body.innerHTML = `
      <div class="mb-4 grid grid-cols-2 gap-3">
        <div class="rounded-lg border border-zinc-800 bg-zinc-950/40 p-3">
          <div class="text-[10px] text-zinc-500">可用余额</div>
          <div class="mt-1 font-mono text-lg font-bold text-cyan-400">$${bal?.available_balance ?? '0.000000'}</div>
        </div>
        <div class="rounded-lg border border-zinc-800 bg-zinc-950/40 p-3">
          <div class="text-[10px] text-zinc-500">冻结余额</div>
          <div class="mt-1 font-mono text-lg font-bold text-zinc-300">$${bal?.frozen_balance ?? '0.000000'}</div>
        </div>
      </div>
      <div class="text-xs font-semibold text-zinc-300">资金流水</div>
      ${rows.length ? `
        <div class="mt-2 overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead><tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
              <th class="pb-2 pr-3 font-medium">时间</th>
              <th class="pb-2 pr-3 font-medium">类型</th>
              <th class="pb-2 pr-3 font-medium text-right">金额</th>
              <th class="pb-2 font-medium text-right">余额</th>
            </tr></thead>
            <tbody>
              ${rows.map((t) => `
                <tr class="row-hover border-b border-zinc-800/50">
                  <td class="py-2 pr-3 font-mono text-[10px] text-zinc-500">${shortTime(t.created_at)}</td>
                  <td class="py-2 pr-3">${t.tx_type}</td>
                  <td class="py-2 pr-3 text-right font-mono ${Number(t.amount) < 0 ? 'text-rose-400' : 'text-emerald-400'}">${t.amount}</td>
                  <td class="py-2 text-right font-mono text-zinc-400">${t.balance_after}</td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>` : `<div class="py-6 text-center text-xs text-zinc-500">暂无流水</div>`}`;
  } catch (err) {
    body.innerHTML = `<div class="py-6 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function toggleUser(user) {
  const next = user.status === 'active' ? 'suspended' : 'active';
  Confirm(`确认将用户 #${user.id} 状态改为 ${next}？`, async () => {
    await adminSend('PUT', `/users/${user.id}/status`, { status: next });
    Toast('状态已更新', 'success');
    loadUsers(USER_PAGE);
  });
}

function deleteUser(user) {
  Confirm(`确认永久删除用户 #${user.id}？将同时删除其网关 Key、余额、资金流水、用量日志与日汇总，且不可恢复。如需保留数据请改用「停用」。`, async () => {
    await adminSend('DELETE', `/users/${user.id}`);
    Toast('用户已永久删除', 'success');
    loadUsers(USER_PAGE);
    loadKeys();
  });
}

/* ---------- 用户的 Key ---------- */
async function userKeysModal(user) {
  const modal = openModal({
    title: `网关 Key · ${user.nickname || ('#' + user.id)}`,
    width: 'max-w-3xl',
    bodyHTML: `<div id="uk-body" class="py-6 text-center text-xs text-zinc-500">加载中…</div>`,
  });
  const body = modal.querySelector('#uk-body');
  try {
    const d = await adminGet(`/users/${user.id}/keys`, { page: 1, page_size: 50 });
    const rows = d?.list || [];
    body.innerHTML = `
      <div class="mb-3 flex justify-end">
        <button id="uk-new" class="btn btn-primary rounded-lg px-3.5 py-1.5 text-[11px] font-bold">创建 Key</button>
      </div>
      ${rows.length ? `
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead><tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
              <th class="pb-2 pr-3 font-medium">名称</th>
              <th class="pb-2 pr-3 font-medium">前缀</th>
              <th class="pb-2 pr-3 font-medium">状态</th>
              <th class="pb-2 pr-3 font-medium">最后使用</th>
              <th class="pb-2 font-medium text-right">操作</th>
            </tr></thead>
            <tbody>
              ${rows.map((k) => `
                <tr class="row-hover border-b border-zinc-800/50">
                  <td class="py-2 pr-3 font-semibold">${k.key_name}</td>
                  <td class="py-2 pr-3 font-mono text-[10px] text-zinc-500">${k.prefix}</td>
                  <td class="py-2 pr-3">${k.is_active ? Badge('启用', 'success') : Badge('停用', 'neutral')}</td>
                  <td class="py-2 pr-3 font-mono text-[10px] text-zinc-500">${k.last_used_at ? shortTime(k.last_used_at) : '—'}</td>
                  <td class="py-2 text-right whitespace-nowrap">
                    <button data-kid="${k.id}" data-kact="reset" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">重置</button>
                    <button data-kid="${k.id}" data-kact="toggle" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">${k.is_active ? '停用' : '启用'}</button>
                    <button data-kid="${k.id}" data-kact="del" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
                  </td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>` : `<div class="py-8 text-center text-xs text-zinc-500">暂无 Key</div>`}`;

    body.querySelector('#uk-new').addEventListener('click', () => keyForm(user));
    rows.forEach((k) => {
      body.querySelector(`[data-kid="${k.id}"][data-kact="reset"]`).addEventListener('click', () => Confirm(`重置 Key「${k.key_name}」？旧 Key 将立即失效。`, async () => {
        const out = await adminSend('POST', `/users/${user.id}/keys/${k.id}/reset`, {});
        showFullKey(out?.full_key, () => { userKeysModal(user); loadKeys(); });
      }));
      body.querySelector(`[data-kid="${k.id}"][data-kact="toggle"]`).addEventListener('click', async () => {
        try {
          await adminSend('PUT', `/users/${user.id}/keys/${k.id}`, { is_active: !k.is_active });
          Toast('状态已更新', 'success');
          userKeysModal(user);
          loadKeys();
        } catch (e) { Toast(e.message, 'error'); }
      });
      body.querySelector(`[data-kid="${k.id}"][data-kact="del"]`).addEventListener('click', () => Confirm(`删除 Key「${k.key_name}」？`, async () => {
        await adminSend('DELETE', `/users/${user.id}/keys/${k.id}`);
        Toast('已删除', 'success');
        userKeysModal(user);
        loadKeys();
      }));
    });
  } catch (err) {
    body.innerHTML = `<div class="py-6 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function keyForm(user) {
  openModal({
    title: `创建 Key · ${user.nickname || ('#' + user.id)}`,
    submitText: '创建',
    fields: [
      { name: 'key_name', label: 'Key 名称', type: 'text', value: 'default' },
      { name: 'prefix', label: '前缀', type: 'text', value: 'sk-' },
      { name: 'models', label: '允许模型（逗号分隔，* 表示全部）', type: 'text', value: '*', placeholder: 'gpt-4,deepseek-chat 或 *' },
      { name: 'rate_limit_overrides', label: '限速覆盖（JSON，可选）', type: 'textarea', value: '', placeholder: '{"rpm":600,"tpm":120000}' },
      { name: 'expires_at', label: '过期时间（RFC3339，可选）', type: 'text', placeholder: '2027-01-01T00:00:00Z' },
      { name: 'is_active', label: '启用', type: 'switch', value: true },
    ],
    onSubmit: async (v) => {
      const models = String(v.models || '*').split(',').map((s) => s.trim()).filter(Boolean);
      const body = {
        key_name: v.key_name || 'default',
        prefix: v.prefix || 'sk-',
        permissions: { models: models.length ? models : ['*'] },
        is_active: v.is_active,
      };
      if (v.rate_limit_overrides) {
        try { body.rate_limit_overrides = JSON.parse(v.rate_limit_overrides); }
        catch { throw new Error('rate_limit_overrides 需为合法 JSON'); }
      }
      if (v.expires_at) body.expires_at = v.expires_at;
      const out = await adminSend('POST', `/users/${user.id}/keys`, body);
      showFullKey(out?.full_key, () => { userKeysModal(user); loadKeys(); });
    },
  });
}

function showFullKey(fullKey, onDone) {
  if (!fullKey) { Toast('创建成功（未返回明文）', 'success'); if (onDone) onDone(); return; }
  const modal = openModal({
    title: '网关 Key（仅显示一次）',
    submitText: '我已保存',
    bodyHTML: `
      <p class="mb-2 text-xs text-zinc-400">请立即复制保存，关闭后无法再次查看。</p>
      <div class="flex items-center gap-2">
        <code class="flex-1 break-all rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 font-mono text-xs text-cyan-400">${fullKey}</code>
        <button id="copy-key" class="btn btn-primary shrink-0 rounded-lg px-3 py-2 text-xs font-bold">复制</button>
      </div>`,
    onSubmit: async () => { if (onDone) onDone(); },
  });
  modal.querySelector('#copy-key').addEventListener('click', () => copyText(fullKey, 'Key 已复制'));
}

/* ---------- 全局 Key 列表（可管理） ---------- */
let KEY_CACHE = [];

async function loadKeys() {
  const node = document.getElementById('k-list');
  if (!node) return;
  try {
    const d = await adminGet('/keys', { page: 1, page_size: 50 });
    const rows = d?.list || [];
    KEY_CACHE = rows;
    node.innerHTML = '';
    if (rows.length) {
      node.innerHTML = keysAdminTableHTML(rows);
      bindKeyActions();
    } else {
      node.append(EmptyState({ title: '暂无 Key', desc: '在上方用户行点击「Key」创建', icon: 'key' }));
    }
  } catch (err) {
    node.innerHTML = `<div class="py-10 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
  }
}

function keysAdminTableHTML(rows) {
  return `
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2.5 pr-3 font-medium">名称</th>
            <th class="pb-2.5 pr-3 font-medium">用户</th>
            <th class="pb-2.5 pr-3 font-medium">前缀</th>
            <th class="pb-2.5 pr-3 font-medium">状态</th>
            <th class="pb-2.5 pr-3 font-medium">最后使用</th>
            <th class="pb-2.5 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((k) => `
            <tr class="row-hover border-b border-zinc-800/50" data-krow="${k.id}">
              <td class="py-2.5 pr-3 font-semibold">${k.key_name}</td>
              <td class="py-2.5 pr-3 text-zinc-400">#${k.user_id}</td>
              <td class="py-2.5 pr-3 font-mono text-[10px] text-zinc-500">${k.prefix}</td>
              <td class="py-2.5 pr-3">${k.is_active ? Badge('启用', 'success') : Badge('停用', 'neutral')}</td>
              <td class="py-2.5 pr-3 font-mono text-[10px] text-zinc-500">${k.last_used_at ? shortTime(k.last_used_at) : '—'}</td>
              <td class="py-2.5 text-right whitespace-nowrap">
                <button data-kact="reset" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">重置</button>
                <button data-kact="toggle" class="btn btn-ghost rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400">${k.is_active ? '停用' : '启用'}</button>
                <button data-kact="del" class="btn btn-ghost rounded-md border border-rose-500/30 px-2 py-1 text-[11px] text-rose-400">删除</button>
              </td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function bindKeyActions() {
  document.querySelectorAll('#k-list tr[data-krow]').forEach((tr) => {
    const id = Number(tr.dataset.krow);
    const k = KEY_CACHE.find((x) => x.id === id);
    if (!k) return;
    tr.querySelector('[data-kact="reset"]').addEventListener('click', () => Confirm(`重置 Key「${k.key_name}」？旧 Key 将立即失效。`, async () => {
      const out = await adminSend('POST', `/users/${k.user_id}/keys/${k.id}/reset`, {});
      showFullKey(out?.full_key, loadKeys);
    }));
    tr.querySelector('[data-kact="toggle"]').addEventListener('click', async () => {
      try {
        await adminSend('PUT', `/users/${k.user_id}/keys/${k.id}`, { is_active: !k.is_active });
        Toast('状态已更新', 'success');
        loadKeys();
      } catch (e) { Toast(e.message, 'error'); }
    });
    tr.querySelector('[data-kact="del"]').addEventListener('click', () => Confirm(`永久删除 Key「${k.key_name}」？`, async () => {
      await adminSend('DELETE', `/users/${k.user_id}/keys/${k.id}`);
      Toast('Key 已删除', 'success');
      loadKeys();
    }));
  });
}
