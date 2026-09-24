import type { components } from '../api/generated/schema'

export type AdminResponse<T> = { code: number; message: string; data: T }
export type ListResponse<T> = { list: T[]; total: number }

export type Balance = components['schemas']['Balance']
export type User = components['schemas']['User']
export type UserInput = components['schemas']['UserInput']
export type BalanceTransaction = components['schemas']['BalanceTransaction']
export type RechargeInput = components['schemas']['RechargeInput']

export type ClientKey = components['schemas']['Key']
export type CreateKeyInput = components['schemas']['CreateKeyInput']
export type KeySecret = components['schemas']['KeySecret']

export type Channel = components['schemas']['Channel']
export type ChannelInput = components['schemas']['ChannelInput']
export type ChannelBalanceInput = components['schemas']['ChannelBalanceInput']
export type ChannelStatusInput = components['schemas']['ChannelStatusInput']
export type ChannelModel = components['schemas']['ChannelModel']
export type ChannelModelInput = components['schemas']['ChannelModelInput']
export type ChannelModelUpdate = components['schemas']['ChannelModelUpdate']
export type ChannelTestItem = components['schemas']['ChannelTestItem']
export type RemoteModel = components['schemas']['RemoteModel']
export type RemoteModelsResult = { ok: boolean; models?: RemoteModel[]; error?: string }
export type Health = components['schemas']['Health']

export type CatalogChannel = components['schemas']['CatalogChannel']
export type CatalogModel = components['schemas']['CatalogModel']
export type Pricing = components['schemas']['Pricing']
export type PricingInput = components['schemas']['PricingInput']
export type DeletePricingInput = components['schemas']['DeletePricingInput']

export type UsageLog = components['schemas']['UsageLog']
export type Stats = components['schemas']['Stats']
export type DailyStats = components['schemas']['DailyStats']
export type ChannelStats = components['schemas']['ChannelStats']
export type UsageAggregate = components['schemas']['UsageAggregate']

export type RateLimit = components['schemas']['RateLimit']
export type RateLimitInput = components['schemas']['RateLimitInput']
export type QuotaPolicy = components['schemas']['QuotaPolicy']
export type QuotaPolicyInput = components['schemas']['QuotaPolicyInput']
export type QuotaUsage = components['schemas']['QuotaUsage']

export type Deleted = { deleted: true }
