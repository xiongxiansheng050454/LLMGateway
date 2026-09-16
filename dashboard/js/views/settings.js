/* 视图：系统设置 */
function renderSettingsView() {
  const bento = document.getElementById('bento');
  const base = dashboardApiBase();
  bento.append(Card('col-span-12 xl:col-span-6', `
    ${CardHeader({ title: '运行信息', desc: '当前控制台接入信息' })}
    <div class="space-y-3 text-xs">
      <div class="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950/40 px-3 py-2.5">
        <span class="text-zinc-400">管理端接口地址</span><span class="font-mono text-cyan-400">${base}</span>
      </div>
      <div class="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950/40 px-3 py-2.5">
        <span class="text-zinc-400">健康检查</span><a class="font-mono text-cyan-400 hover:underline" href="../healthz" target="_blank">/healthz</a>
      </div>
      <div class="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950/40 px-3 py-2.5">
        <span class="text-zinc-400">数据更新时间</span><span class="font-mono text-zinc-300">${LAST_UPDATED || '—'}</span>
      </div>
    </div>`));

  bento.append(Card('col-span-12 xl:col-span-6', `
    ${CardHeader({ title: '接口说明', desc: '管理端当前不设认证（内网部署）' })}
    <ul class="space-y-2 text-xs text-zinc-400">
      <li>· 跨端口部署可用 <span class="font-mono text-cyan-400">?api_base=http://host:port/admin</span> 指定接口地址</li>
      <li>· 管理端接口返回 <span class="font-mono text-cyan-400">{code, message, data}</span>，code=0 为成功</li>
      <li>· 下游接口为 OpenAI 兼容协议，鉴权使用网关 Key</li>
    </ul>`));
}
