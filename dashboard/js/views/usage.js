/* 视图：用量统计（KPI / 趋势 / 每日明细 / 渠道成本，支持日期筛选） */
async function renderUsageView() {
  const bento = document.getElementById('bento');
  const successRate = ((OVERVIEW.success_count / OVERVIEW.request_count) * 100).toFixed(1);
  bento.append(
    StatCard({ spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3', label: '请求总量', value: (OVERVIEW.request_count / 1000).toFixed(1) + 'K', delta: 0, deltaLabel: '近 7 天', icon: 'zap', spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.requests)) }),
    StatCard({ spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3', label: '成功率', value: successRate, unit: '%', delta: 0, deltaLabel: '近 7 天', icon: 'gauge', spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.success), { color: '#34d399' }) }),
    StatCard({ spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3', label: 'Tokens', value: (OVERVIEW.total_tokens / 1e6).toFixed(2) + 'M', delta: 0, deltaLabel: '近 7 天', icon: 'models', spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.tokens), { color: '#a78bfa' }) }),
    StatCard({ spanCls: 'col-span-12 sm:col-span-6 xl:col-span-3', label: '费用（美元）', value: '$' + Number(OVERVIEW.total_cost).toFixed(2), delta: 0, deltaLabel: '近 7 天', icon: 'wallet', spark: Sparkline(DAILY_STATS.slice(-7).map((d) => d.cost), { color: '#f59e0b' }) }),
  );

  const trend = Card('col-span-12 xl:col-span-8', `${CardHeader({ title: '请求趋势（近 14 天）', desc: 'user_daily_stats · 请求次数 / 费用' })}`);
  trend.append(BarChart(DAILY_STATS));
  bento.append(trend);

  const dist = Card('col-span-12 xl:col-span-4', `${CardHeader({ title: '模型用量分布', desc: 'usage_logs 聚合 · 万 tokens' })}`);
  dist.append(DonutChart(MODEL_DIST));
  bento.append(dist);

  const filterCard = Card('col-span-12', `
    ${CardHeader({ title: '日期范围', desc: '按统计自然日查询用户日汇总与渠道成本' })}
    <div class="flex flex-wrap items-end gap-3">
      <label class="block"><span class="mb-1 block text-xs text-zinc-400">开始日期</span><input id="u-from" type="date" class="form-input w-40"/></label>
      <label class="block"><span class="mb-1 block text-xs text-zinc-400">结束日期</span><input id="u-to" type="date" class="form-input w-40"/></label>
      <button id="u-query" class="btn btn-primary rounded-lg px-3.5 py-2 text-xs font-bold">查询</button>
    </div>`);
  bento.append(filterCard);

  const dailyCard = Card('col-span-12 xl:col-span-7', `${CardHeader({ title: '每日明细', desc: 'user_daily_stats 聚合' })}<div id="u-daily" class="py-8 text-center text-xs text-zinc-500">加载中…</div>`);
  bento.append(dailyCard);
  const chanCard = Card('col-span-12 xl:col-span-5', `${CardHeader({ title: '渠道成本', desc: 'stats/channels 聚合' })}<div id="u-channels" class="py-8 text-center text-xs text-zinc-500">加载中…</div>`);
  bento.append(chanCard);

  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 6);
  document.getElementById('u-from').value = toDateParam(from);
  document.getElementById('u-to').value = toDateParam(to);
  document.getElementById('u-query').addEventListener('click', () => {
    loadUsageTables(document.getElementById('u-from').value, document.getElementById('u-to').value);
  });
  await loadUsageTables(toDateParam(from), toDateParam(to));
  animateProgress();
}

async function loadUsageTables(from, to) {
  const dailyNode = document.getElementById('u-daily');
  const chanNode = document.getElementById('u-channels');
  if (!dailyNode || !chanNode) return;
  const startTime = from ? new Date(from + 'T00:00:00').toISOString() : undefined;
  const endTime = to ? new Date(to + 'T23:59:59').toISOString() : undefined;
  try {
    const [daily, channels] = await Promise.all([
      adminGet('/stats/daily', { date_from: from, date_to: to, page: 1, page_size: 100 }),
      adminGet('/stats/channels', { start_time: startTime, end_time: endTime }),
    ]);
    const rows = daily?.list || [];
    dailyNode.innerHTML = rows.length ? `
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead><tr class="border-b border-zinc-800 text-[11px] uppercase tracking-wide text-zinc-500">
            <th class="pb-2 pr-3 font-medium">日期</th>
            <th class="pb-2 pr-3 font-medium text-right">请求</th>
            <th class="pb-2 pr-3 font-medium text-right">成功</th>
            <th class="pb-2 pr-3 font-medium text-right">失败</th>
            <th class="pb-2 pr-3 font-medium text-right">Tokens</th>
            <th class="pb-2 font-medium text-right">费用</th>
          </tr></thead>
          <tbody>
            ${rows.map((d) => `
              <tr class="row-hover border-b border-zinc-800/50">
                <td class="py-2 pr-3 font-mono text-zinc-400">${d.stat_date}</td>
                <td class="py-2 pr-3 text-right font-mono text-zinc-300">${Number(d.request_count).toLocaleString()}</td>
                <td class="py-2 pr-3 text-right font-mono text-emerald-400">${Number(d.success_count).toLocaleString()}</td>
                <td class="py-2 pr-3 text-right font-mono text-amber-400">${Number(d.error_count).toLocaleString()}</td>
                <td class="py-2 pr-3 text-right font-mono text-zinc-300">${Number(d.total_tokens).toLocaleString()}</td>
                <td class="py-2 text-right font-mono font-semibold text-cyan-400">$${d.total_cost}</td>
              </tr>`).join('')}
          </tbody>
        </table>
      </div>` : `<div class="py-8 text-center text-xs text-zinc-500">该区间暂无数据</div>`;

    const chs = channels?.list || [];
    chanNode.innerHTML = chs.length ? `
      <div class="space-y-2">
        ${chs.map((c) => `
          <div class="rounded-lg border border-zinc-800/60 bg-zinc-950/35 px-3 py-2.5">
            <div class="flex items-center justify-between">
              <span class="truncate text-xs font-semibold">${c.channel_name || ('#' + c.channel_id)}</span>
              <span class="font-mono text-xs font-bold text-cyan-400">$${c.total_cost}</span>
            </div>
            <div class="mt-1 flex items-center justify-between font-mono text-[10px] text-zinc-500">
              <span>${Number(c.request_count).toLocaleString()} 次 · ${Number(c.total_tokens).toLocaleString()} tokens</span>
              <span>成功 ${c.success_count} / 失败 ${c.error_count}</span>
            </div>
          </div>`).join('')}
      </div>` : `<div class="py-8 text-center text-xs text-zinc-500">该区间暂无渠道数据</div>`;
  } catch (err) {
    dailyNode.innerHTML = `<div class="py-8 text-center text-xs text-rose-400">加载失败：${err.message}</div>`;
    chanNode.innerHTML = '';
  }
}
