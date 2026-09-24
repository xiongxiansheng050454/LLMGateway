import { adminGet } from './client'
import { apiPaths } from './paths'
import type { DailyStats, ListResponse, Stats, UsageAggregate, UsageLog } from '../types/api'

export const overview = () => adminGet<Stats>(apiPaths.usageOverview(), { start_time: new Date(Date.now() - 6 * 86400000).toISOString(), end_time: new Date().toISOString() })
export const logs = () => adminGet<ListResponse<UsageLog>>(apiPaths.usageLogs(), { page: 1, page_size: 50 })
export const daily = () => adminGet<ListResponse<DailyStats>>(apiPaths.usageDaily(), { page: 1, page_size: 100 })
export const usageStats = (group_by: 'user' | 'api_key' | 'model' | 'channel') => adminGet<ListResponse<UsageAggregate>>(apiPaths.usageAggregate(), { group_by })
export const listUsageLogs = (params: Record<string, string | number | undefined> = {}) => adminGet<ListResponse<UsageLog>>(apiPaths.usageLogs(), params)
export const getUsageLog = (id: number) => adminGet<UsageLog>(apiPaths.usageLog(id))
export const channelStats = (params: Record<string, string | number | undefined>) => adminGet<ListResponse<import('../types/api').ChannelStats>>(apiPaths.usageChannels(), params)
