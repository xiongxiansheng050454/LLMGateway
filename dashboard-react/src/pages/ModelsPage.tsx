import { useQuery } from '@tanstack/react-query'
import { listModels } from '../api/catalog'
import { usageStats } from '../api/usage'
import type { CatalogModel, ListResponse, UsageAggregate } from '../types/api'

const number = (value: number) => value || 0

export function ModelsPage() {
  const query = useQuery({ queryKey: ['models'], queryFn: () => listModels({ status: 1 }) })
  const usage = useQuery({ queryKey: ['model-usage'], queryFn: () => usageStats({ group_by: 'model', page: 1, page_size: 100 }) })
  const list = query.data?.list || []
  const usageRows = usage.data?.list || []
  const total = Math.max(1, usageRows.reduce((sum, row) => sum + number(row.total_tokens || row.request_count), 0))

  return <div className="bento-grid">
    <section className="panel span-7"><div className="panel-head"><div><h3>已发布模型目录</h3><p className="muted">GET /admin/models · 对外模型名与可用渠道</p></div></div>{query.isLoading ? <div className="empty">加载中…</div> : <div className="channel-list">{list.map(model => <div className="channel-row" key={model.model_name}><i className="dot ok" /><b>{model.model_name}</b><span className="url">{model.channels.map(channel => channel.channel_name).join(' · ') || '—'}</span><em>{model.channels.length} 渠道</em></div>)}{!list.length && <div className="empty">暂无已发布模型</div>}</div>}</section>
    <section className="panel span-5"><div className="panel-head"><div><h3>模型用量分布</h3><p className="muted">usage_logs 聚合 · tokens</p></div></div>{usage.isLoading ? <div className="empty">加载中…</div> : <div className="distribution-list">{usageRows.slice(0, 8).map(row => { const value = number(row.total_tokens || row.request_count); return <div key={row.model}><span>{row.model || 'unknown'}</span><b>{(value / total * 100).toFixed(1)}%</b></div> })}{!usageRows.length && <div className="empty">暂无模型用量</div>}</div>}</section>
    <section className="panel span-12"><div className="panel-head"><div><h3>路由策略</h3><p className="muted">priority 优先 · weight 加权 · channel balance 过滤</p></div></div><div className="route-table"><div><span>priority</span><b>优先级越高越先选择</b></div><div><span>weight</span><b>同优先级组内按权重分配</b></div><div><span>health</span><b>open 渠道不会进入候选</b></div></div></section>
  </div>
}
