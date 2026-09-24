export function AsyncState({ loading, error, hasData = false, onRetry }: { loading: boolean; error: unknown; hasData?: boolean; onRetry?: () => void }) {
  if (loading && !hasData) return <div className="empty">加载中…</div>
  if (error) return <div className="empty error">{hasData ? '数据更新失败，当前显示上次成功数据。' : '该面板暂时无法加载。'} {onRetry && <button className="button ghost" onClick={onRetry}>重试</button>}</div>
  return null
}
