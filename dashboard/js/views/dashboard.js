/* 视图：运营仪表盘 */
function renderDashboardView() {
  const bento = document.getElementById('bento');
  const successRate = ((OVERVIEW.success_count / OVERVIEW.request_count) * 100).toFixed(1);

  bento.append(
    StatCard({
      spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3',
      label: '总请求量（7 天）',
      value: (OVERVIEW.request_count / 1000).toFixed(1) + 'K',
      delta: 12.4,
      icon: 'zap',
      spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.requests)),
    }),
    StatCard({
      spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3',
      label: '请求成功率',
      value: successRate,
      unit: '%',
      delta: 0.6,
      icon: 'gauge',
      spark: Sparkline([98.1, 98.4, 98.2, 98.6, 98.5, 98.9, 98.7], { color: '#34d399' }),
    }),
    StatCard({
      spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3',
      label: '消耗 Tokens（7 天）',
      value: (OVERVIEW.total_tokens / 1e6).toFixed(1) + 'M',
      delta: 8.9,
      icon: 'models',
      spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.cost), { color: '#a78bfa' }),
    }),
    StatCard({
      spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3',
      label: '累计费用（美元）',
      value: '$' + Number(OVERVIEW.total_cost).toFixed(2),
      delta: 15.2,
      icon: 'wallet',
      spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.cost), { color: '#f59e0b' }),
    }),
  );

  const trend = Card('col-span-12 xl:col-span-8', `
    ${CardHeader({ title: '请求趋势（近 14 天）', desc: '按 user_daily_stats 自然日汇总 · 含成功 / 失败' })}
    <div class="mb-5 flex items-center gap-6 text-xs text-zinc-400">
      <span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full bg-cyan-400"></span>今日 <b class="stat-value text-sm text-white">${DAILY_STATS.at(-1).requests.toLocaleString()}</b> 次</span>
      <span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full bg-emerald-400"></span>成功率 <b class="text-sm text-white">${successRate}%</b></span>
      <span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full bg-amber-400"></span>错误 <b class="text-sm text-white">${OVERVIEW.error_count.toLocaleString()}</b> 次</span>
    </div>`);
  trend.append(BarChart(DAILY_STATS));
  bento.append(trend);

  const docsCard = Card('col-span-12 xl:col-span-4', `
    ${CardHeader({
      title: '看 API 文档',
      desc: '下游兼容接口与管理端接口速查',
      action: `<button data-doc="__list__" class="doc-open btn btn-ghost flex items-center gap-1 rounded-lg border border-zinc-800 px-2.5 py-1.5 text-[11px] font-semibold text-zinc-400">全部文档 ${Icon('book', 'h-3 w-3')}</button>`,
    })}
    <div class="grid gap-2.5">
      ${API_DOCS.map((doc) => `
        <button data-doc="${doc.href}" class="doc-open group flex w-full items-center gap-3 rounded-xl border border-zinc-800/70 bg-zinc-950/45 p-3 text-left transition hover:border-cyan-400/40 hover:bg-cyan-400/5">
          <span class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-cyan-400/10 text-cyan-400 group-hover:shadow-[0_0_18px_-6px_rgba(34,211,238,.9)]">${Icon(doc.icon, 'h-4 w-4')}</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-semibold text-white">${doc.title}</span>
            <span class="mt-0.5 block truncate font-mono text-[10px] text-zinc-500">${doc.desc}</span>
          </span>
          ${Icon('external', 'h-3.5 w-3.5 text-zinc-600 group-hover:text-cyan-400')}
        </button>`).join('')}
    </div>`);
  bindDocOpeners(docsCard);
  bento.append(docsCard);

  const dist = Card('col-span-12 xl:col-span-4', `${CardHeader({ title: '模型用量分布', desc: '按对外模型名聚合 · 万 tokens' })}`);
  dist.append(DonutChart(MODEL_DIST));
  bento.append(dist);

  const chCard = Card('col-span-12 xl:col-span-4', `
    ${CardHeader({ title: '渠道健康状态', desc: '路由 · 熔断 · 成功率' })}
    <div class="space-y-4">${channelListHTML(CHANNELS)}</div>`);
  bento.append(chCard);

  const routeCard = Card('col-span-12 xl:col-span-4', `
    ${CardHeader({ title: '路由策略雷达', desc: '权重优先级 · 余额过滤 · 熔断探测' })}
    <div class="mb-5 rounded-xl border border-cyan-400/20 bg-gradient-to-br from-cyan-400/10 via-blue-500/5 to-transparent p-4">
      <div class="flex items-center gap-3">
        <span class="grid h-10 w-10 place-items-center rounded-xl bg-cyan-400/10 text-cyan-400">${Icon('route', 'h-5 w-5')}</span>
        <div>
          <div class="font-mono text-2xl font-extrabold text-white">${CHANNELS.filter((c) => c.status === 1).length}</div>
          <div class="text-xs text-zinc-400">当前启用渠道</div>
        </div>
      </div>
    </div>
    <div class="space-y-4">${ROUTING_OVERVIEW.map((r) => ProgressBar({ value: r.value, label: r.label, right: r.right, warn: r.value < 30 })).join('')}</div>`);
  bento.append(routeCard);

  const logCard = Card('col-span-12 xl:col-span-8', `
    ${CardHeader({ title: '实时请求日志', desc: 'usage_logs · 预冻结 → 按实际 usage 结算' })}
    ${logsTableHTML(RECENT_LOGS)}`);
  bento.append(logCard);

  const errorCard = Card('col-span-12 xl:col-span-4', `
    ${CardHeader({ title: '错误画像', desc: '近 7 天非成功请求聚合' })}
    <div class="space-y-2.5">
      ${ERROR_MIX.map((e) => `
        <div class="row-hover flex items-center justify-between rounded-lg border border-zinc-800/60 bg-zinc-950/35 px-3 py-2.5">
          <div class="min-w-0">
            <div class="truncate font-mono text-xs text-zinc-300">${e.name}</div>
            <div class="mt-0.5 text-[10px] text-zinc-500">OpenAI 风格错误码透传</div>
          </div>
          <div class="text-right">
            <div class="font-mono text-sm font-bold text-white">${e.count}</div>
            ${Badge(e.tone === 'warning' ? '限流' : e.tone === 'error' ? '上游' : e.tone === 'cyan' ? '渠道' : '余额', e.tone)}
          </div>
        </div>`).join('')}
    </div>`);
  bento.append(errorCard);

  const rlCard = Card('col-span-12 md:col-span-6 xl:col-span-4', `
    ${CardHeader({ title: '限流配额水位', desc: 'rate_limit_rules · 当前消耗占比' })}
    <div class="space-y-4">${progressListHTML(RATE_LIMITS)}</div>`);
  bento.append(rlCard);

  const userCard = Card('col-span-12 md:col-span-6 xl:col-span-4', `
    ${CardHeader({ title: '用户余额 Top', desc: 'user_balances · 可用 / 冻结（美元）' })}
    <div class="space-y-1">${usersListHTML(TOP_USERS)}</div>`);
  bento.append(userCard);

  const sysCard = Card('col-span-12 xl:col-span-4', `
    ${CardHeader({ title: '系统事件', desc: '熔断 · 结算 · 路由变更' })}
    <ul class="space-y-3.5">
      ${NOTIFICATIONS.map((n) => `
        <li class="flex items-start gap-3">
          <span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full ${n.tone === 'warning' ? 'bg-amber-400' : n.tone === 'success' ? 'bg-emerald-400' : 'bg-cyan-400'}"></span>
          <div class="text-xs leading-relaxed text-zinc-400">${n.text}</div>
        </li>`).join('')}
    </ul>`);
  bento.append(sysCard);

  animateProgress();
}
