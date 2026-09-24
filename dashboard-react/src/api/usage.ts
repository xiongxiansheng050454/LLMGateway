import { adminGet } from './client'
import { apiPaths } from './paths'
import { ChannelStatsListSchema, DailyStatsListSchema, StatsSchema, UsageAggregateListSchema, UsageLogListSchema, UsageLogSchema } from './runtime/usage'
import type { DailyStats, ListResponse, Stats, UsageAggregate, UsageLog } from '../types/api'

export const overview = () => adminGet(apiPaths.usageOverview(), { start_time: new Date(Date.now() - 6 * 86400000).toISOString(), end_time: new Date().toISOString() }, StatsSchema)
export const logs = () => adminGet(apiPaths.usageLogs(), { page: 1, page_size: 50 }, UsageLogListSchema)
export const daily = () => adminGet(apiPaths.usageDaily(), { page: 1, page_size: 100 }, DailyStatsListSchema)
export const usageStats = (group_by: 'user' | 'api_key' | 'model' | 'channel') => adminGet(apiPaths.usageAggregate(), { group_by }, UsageAggregateListSchema)
export const listUsageLogs = (params: Record<string, string | number | undefined> = {}) => adminGet(apiPaths.usageLogs(), params, UsageLogListSchema)
export const getUsageLog = (id: number) => adminGet(apiPaths.usageLog(id), UsageLogSchema)
export const channelStats = (params: Record<string, string | number | undefined>) => adminGet(apiPaths.usageChannels(), params, ChannelStatsListSchema)
