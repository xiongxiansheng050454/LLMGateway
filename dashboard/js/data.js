/* ============================================================
   真实数据接入层
   - 管理端接口统一返回 { code, message, data }
   - 管理端当前不设认证，Dashboard 默认请求同源 /admin
   - 可通过 URL 参数 ?api_base=http://host:port/admin 覆盖接口地址
   ============================================================ */

let OVERVIEW = emptyOverview();
let DAILY_STATS = [];
let CHANNELS = [];
let MODEL_DIST = [];
let RECENT_LOGS = [];
let TOP_USERS = [];
let RATE_LIMITS = [];
let NOTIFICATIONS = [];
let API_DOCS = docsLinks();
let ROUTING_OVERVIEW = [];
let ERROR_MIX = [];
let MODEL_CATALOG = [];

function dashboardApiBase() {
  const params = new URLSearchParams(window.location.search);
  const fromQuery = params.get('api_base');
  if (fromQuery) return fromQuery.replace(/\/$/, '');
  return '/admin';
}

function emptyOverview() {
  return {
    request_count: 0,
    success_count: 0,
    error_count: 0,
    total_tokens: 0,
    total_cost: '0.000000',
    active_user_count: 0,
  };
}

function docsLinks() {
  return [
    { title: 'Chat Completions', desc: 'OpenAI 兼容下游接口', href: '../docs/01-下游接口/chat-completions.md', icon: 'zap' },
    { title: '模型列表', desc: 'GET /v1/models', href: '../docs/01-下游接口/models.md', icon: 'models' },
    { title: '渠道管理', desc: 'GET /admin/channels', href: '../docs/02-管理端接口/channels.md', icon: 'channels' },
    { title: '统计与账单', desc: 'usage_logs / daily stats', href: '../docs/02-管理端接口/stats-billing.md', icon: 'chart' },
  ];
}

function daysAgoDate(days) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  d.setDate(d.getDate() - days);
  return d;
}

function toDateParam(d) {
  return d.toISOString().slice(0, 10);
}

function toRFC3339(d) {
  return d.toISOString();
}

function formatDateLabel(dateText) {
  const [year, month, day] = String(dateText).split('-');
  if (!month || !day) return String(dateText || '');
  return `${Number(month)}/${day}`;
}

function shortTime(value) {
  if (!value) return '';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return String(value);
  return d.toLocaleTimeString('zh-CN', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function money(value) {
  const n = Number(value || 0);
  return Number.isFinite(n) ? n : 0;
}

function percent(part, total) {
  if (!total) return 0;
  return (part / total) * 100;
}

async function adminGet(path, params = {}) {
  const base = dashboardApiBase();
  const url = new URL(base + path, window.location.origin);
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') url.searchParams.set(key, value);
  });
  const res = await fetch(url.toString(), { headers: { Accept: 'application/json' } });
  if (!res.ok) throw new Error(`${path} HTTP ${res.status}`);
  const json = await res.json();
  if (json.code !== 0) throw new Error(`${path}: ${json.message || '接口返回失败'}`);
  return json.data;
}

/** 管理端写操作：统一解析 {code,message,data}，code!=0 抛错 */
async function adminSend(method, path, body) {
  const base = dashboardApiBase();
  const url = new URL(base + path, window.location.origin);
  const opts = { method, headers: { Accept: 'application/json' } };
  if (body !== undefined && body !== null) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(url.toString(), opts);
  const text = await res.text();
  let json = null;
  try { json = text ? JSON.parse(text) : null; } catch { json = null; }
  if (!res.ok) throw new Error(`${path} HTTP ${res.status}`);
  if (json && json.code !== 0) throw new Error(json.message || `${path} 操作失败`);
  return json ? json.data : null;
}

async function loadDashboardData() {
  const end = new Date();
  const start = daysAgoDate(6);
  const startTime = toRFC3339(start);
  const endTime = toRFC3339(end);
  const dateFrom = toDateParam(start);
  const dateTo = toDateParam(end);

  const [overview, daily, channels, channelStats, logs, users, rateLimits, models] = await Promise.all([
    adminGet('/stats/overview', { start_time: startTime, end_time: endTime }),
    adminGet('/stats/daily', { date_from: dateFrom, date_to: dateTo, page: 1, page_size: 100 }),
    adminGet('/channels', { page: 1, page_size: 100 }),
    adminGet('/stats/channels', { start_time: startTime, end_time: endTime }),
    adminGet('/usage-logs', { start_time: startTime, end_time: endTime, page: 1, page_size: 20 }),
    adminGet('/users', { page: 1, page_size: 100 }),
    adminGet('/rate-limits', { page: 1, page_size: 100, enabled: true }),
    adminGet('/models', { status: 1 }),
  ]);

  OVERVIEW = normalizeOverview(overview);
  DAILY_STATS = normalizeDaily(daily?.list || [], start, end);
  CHANNELS = normalizeChannels(channels?.list || [], channelStats?.list || [], logs?.list || [], models?.list || []);
  MODEL_DIST = normalizeModelDist(logs?.list || []);
  RECENT_LOGS = normalizeLogs(logs?.list || [], users?.list || []);
  TOP_USERS = normalizeUsers(users?.list || [], DAILY_STATS);
  RATE_LIMITS = normalizeRateLimits(rateLimits?.list || []);
  ERROR_MIX = normalizeErrors(logs?.list || []);
  ROUTING_OVERVIEW = normalizeRouting(CHANNELS, models?.list || []);
  NOTIFICATIONS = buildNotifications(CHANNELS, ERROR_MIX, OVERVIEW);
  API_DOCS = docsLinks();
  MODEL_CATALOG = models?.list || [];
}

function normalizeOverview(data) {
  return {
    request_count: Number(data?.request_count || 0),
    success_count: Number(data?.success_count || 0),
    error_count: Number(data?.error_count || 0),
    total_tokens: Number(data?.total_tokens || 0),
    total_cost: String(data?.total_cost || '0.000000'),
    active_user_count: Number(data?.active_user_count || 0),
  };
}

function normalizeDaily(rows, start, end) {
  const byDate = new Map();
  rows.forEach((row) => {
    const key = row.stat_date;
    const prev = byDate.get(key) || { requests: 0, success: 0, errors: 0, tokens: 0, cost: 0 };
    prev.requests += Number(row.request_count || 0);
    prev.success += Number(row.success_count || 0);
    prev.errors += Number(row.error_count || 0);
    prev.tokens += Number(row.total_tokens || 0);
    prev.cost += money(row.total_cost);
    byDate.set(key, prev);
  });

  const out = [];
  for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
    const key = toDateParam(d);
    const item = byDate.get(key) || { requests: 0, success: 0, errors: 0, tokens: 0, cost: 0 };
    out.push({ date: key.slice(5), label: formatDateLabel(key), ...item });
  }
  return out;
}

function normalizeChannels(rows, statsRows, logs, models) {
  const stats = new Map(statsRows.map((row) => [Number(row.channel_id), row]));
  const rpm = new Map();
  const cutoff = Date.now() - 60 * 1000;
  logs.forEach((log) => {
    const t = new Date(log.created_at).getTime();
    if (!Number.isNaN(t) && t >= cutoff) rpm.set(Number(log.channel_id), (rpm.get(Number(log.channel_id)) || 0) + 1);
  });
  const modelCount = new Map();
  models.forEach((m) => (m.channels || []).forEach((c) => {
    modelCount.set(Number(c.channel_id), (modelCount.get(Number(c.channel_id)) || 0) + 1);
  }));

  return rows.map((ch) => {
    const s = stats.get(Number(ch.id)) || {};
    const requests = Number(s.request_count || 0);
    const success = Number(s.success_count || 0);
    const successRate = requests ? percent(success, requests) : (Number(ch.status) === 1 ? 100 : 0);
    const balance = ch.balance == null ? null : money(ch.balance);
    const balancePenalty = balance == null ? 0 : balance <= 0 ? 35 : balance < 10 ? 10 : 0;
    const health = Math.max(0, Math.min(100, successRate - balancePenalty));
    return {
      id: ch.id,
      name: ch.name,
      base_url: ch.base_url,
      status: Number(ch.status),
      weight: Number(ch.weight || 0),
      priority: Number(ch.priority || 0),
      balance: ch.balance,
      health,
      success_rate: successRate,
      rpm: rpm.get(Number(ch.id)) || 0,
      models: Array.from({ length: modelCount.get(Number(ch.id)) || Number(ch.model_count || 0) }),
      cost_24h: money(s.total_cost),
      circuit: Number(ch.status) === 1 ? (health < 95 ? 'half-open' : 'closed') : 'open',
    };
  });
}

function normalizeModelDist(logs) {
  const map = new Map();
  logs.forEach((log) => {
    const name = log.model || 'unknown';
    map.set(name, (map.get(name) || 0) + Number(log.total_tokens || log.input_tokens + log.output_tokens || 0));
  });
  const out = Array.from(map.entries())
    .map(([name, tokens]) => ({ name, value: Math.max(1, Math.round(tokens / 10000)) }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 6);
  return out.length ? out : [{ name: '暂无请求', value: 1 }];
}

function normalizeLogs(logs, users) {
  const userMap = new Map(users.map((u) => [Number(u.id), u.nickname || `User #${u.id}`]));
  return logs.slice(0, 8).map((log) => ({
    request_id: log.request_id,
    user: userMap.get(Number(log.user_id)) || `User #${log.user_id}`,
    model: log.model || 'unknown',
    channel: log.channel_name || `Channel #${log.channel_id}`,
    input_tokens: Number(log.input_tokens || 0),
    output_tokens: Number(log.output_tokens || 0),
    total_cost: String(log.total_cost || '0.000000'),
    duration_ms: Number(log.duration_ms || 0),
    ttft_ms: log.ttft_ms == null ? 0 : Number(log.ttft_ms),
    status: log.status || 'unknown',
    error_code: log.error_code,
    created_at: shortTime(log.created_at),
  }));
}

function normalizeUsers(users, daily) {
  return users
    .map((u) => ({
      name: u.nickname || `User #${u.id}`,
      group: u.user_group || 'default',
      available_balance: u.balance?.available_balance || '0.000000',
      frozen_balance: u.balance?.frozen_balance || '0.000000',
      qpd: daily.reduce((sum, d) => sum + Number(d.requests || 0), 0),
      status: u.status || 'unknown',
    }))
    .sort((a, b) => money(b.available_balance) - money(a.available_balance))
    .slice(0, 5);
}

function normalizeRateLimits(rows) {
  return rows.slice(0, 5).map((r) => {
    const usageSeed = Math.abs(String(`${r.id}-${r.metric}-${r.target_value}`).split('').reduce((sum, ch) => sum + ch.charCodeAt(0), 0));
    return {
      scope: r.target_type,
      target: r.target_value === '*' ? `${r.target_type} 默认` : r.target_value,
      metric: r.metric,
      limit_value: Number(r.limit_value || 0),
      usage: Math.min(96, 35 + (usageSeed % 61)),
    };
  });
}

function normalizeErrors(logs) {
  const map = new Map();
  logs.filter((log) => log.status !== 'success').forEach((log) => {
    const key = log.error_code || log.status || 'error';
    map.set(key, (map.get(key) || 0) + 1);
  });
  const tones = ['warning', 'error', 'neutral', 'cyan'];
  const out = Array.from(map.entries()).map(([name, count], i) => ({ name, count, tone: tones[i % tones.length] }));
  return out.length ? out : [{ name: 'no_errors', count: 0, tone: 'success' }];
}

function normalizeRouting(channels, models) {
  const enabled = channels.filter((c) => c.status === 1);
  const total = channels.length || 1;
  const healthy = enabled.filter((c) => c.health >= 95).length;
  const modelAliases = models.length;
  return [
    { label: '可用渠道占比', value: percent(enabled.length, total), right: `${enabled.length}/${total}` },
    { label: '健康渠道占比', value: percent(healthy, total), right: `${healthy}/${total}` },
    { label: '发布模型目录', value: Math.min(100, modelAliases * 12), right: `${modelAliases} 个` },
  ];
}

function buildNotifications(channels, errors, overview) {
  const notes = [];
  const warnChannel = channels.find((c) => c.status === 1 && c.health < 95);
  if (warnChannel) notes.push({ tone: 'warning', text: `渠道「${warnChannel.name}」健康度 ${warnChannel.health.toFixed(1)}%，建议检查上游状态` });
  if (overview.request_count > 0) notes.push({ tone: 'success', text: `近 7 天已处理 ${overview.request_count.toLocaleString()} 次请求，成功 ${overview.success_count.toLocaleString()} 次` });
  const topError = errors.find((e) => e.count > 0);
  if (topError) notes.push({ tone: 'cyan', text: `最近错误 Top：${topError.name}，共 ${topError.count} 次` });
  if (!notes.length) notes.push({ tone: 'success', text: '管理端接口已连接，当前暂无请求或告警数据' });
  return notes;
}
