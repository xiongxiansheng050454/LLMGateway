/* 视图：模型路由 */
function renderModelsView() {
  const bento = document.getElementById('bento');
  const catalog = Card('col-span-12 xl:col-span-7', `
    ${CardHeader({ title: '已发布模型目录', desc: 'GET /admin/models · 对外模型名与可用渠道' })}
    ${modelCatalogHTML(MODEL_CATALOG)}`);
  bento.append(catalog);

  const dist = Card('col-span-12 xl:col-span-5', `${CardHeader({ title: '模型用量分布', desc: '万 tokens' })}`);
  dist.append(DonutChart(MODEL_DIST));
  bento.append(dist);

  const routing = Card('col-span-12', `
    ${CardHeader({ title: '路由策略', desc: '权重优先级 · 余额过滤 · 熔断探测' })}
    <div class="space-y-4">${ROUTING_OVERVIEW.map((r) => ProgressBar({ value: r.value, label: r.label, right: r.right, warn: r.value < 30 })).join('')}</div>`);
  bento.append(routing);
  animateProgress();
}
