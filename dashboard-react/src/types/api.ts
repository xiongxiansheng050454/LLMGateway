export type AdminResponse<T> = { code: number; message: string; data: T }
export type ListResponse<T> = { list: T[]; total: number }

export type Balance = { available_balance: string; frozen_balance: string }
export type User = { id: number; nickname: string; user_group: string; status: 'active' | 'suspended' | string; balance: Balance }
export type UserInput = { nickname: string; user_group: string; status: string; password?: string }
export type BalanceTransaction = { id: number; tx_type: string; amount: string; balance_after: string; created_at: string }
export type RechargeInput = { amount: string; related_order_id?: string; description?: string }

export type ClientKey = { id: number; user_id: number; key_name: string; prefix: string; is_active: boolean; last_used_at: string | null; expires_at: string | null }
export type CreateKeyInput = { key_name: string; prefix: string; permissions: Record<string, unknown>; rate_limit_overrides?: Record<string, unknown>; expires_at?: string; is_active?: boolean }
export type KeySecret = { id?: number; full_key: string }

export type Channel = { id: number; name: string; base_url: string; auth_type: string; status: number; weight: number; priority: number; balance: string | null; model_count: number }
export type ChannelInput = Omit<Channel, 'id' | 'model_count'> & { api_key: string }
export type ChannelBalanceInput = { balance?: string; delta?: string }
export type ChannelStatusInput = { status: number }
export type ChannelModel = { id: number; model_name: string; upstream_model: string; enabled: boolean }
export type ChannelModelInput = Pick<ChannelModel, 'model_name' | 'upstream_model' | 'enabled'>
export type ChannelModelUpdate = Pick<ChannelModel, 'upstream_model' | 'enabled'>
export type ChannelTestItem = { model_alias: string; upstream_model: string; http_status: number; latency_ms: number; ok: boolean; error: string }
export type RemoteModel = { id: string; object?: string; created?: number; owned_by?: string }
export type RemoteModelsResult = { ok: boolean; models?: RemoteModel[]; error?: string }
export type Health = { channel_id: number; state: 'closed' | 'open' | 'half-open'; consecutive_failures: number; success_count: number; failure_count: number; opened_at: string | null; updated_at: string }

export type CatalogChannel = { channel_id: number; channel_name: string; upstream_model: string; enabled: boolean }
export type CatalogModel = { model_name: string; status: number; channels: CatalogChannel[] }
export type Pricing = { id: number; channel_id: number; channel_name: string; model_name: string; upstream_model: string; input_price_per_1m: string; output_price_per_1m: string; cached_input_price_per_1m: string; currency: string }
export type PricingInput = Pick<Pricing, 'channel_id' | 'model_name' | 'input_price_per_1m' | 'output_price_per_1m' | 'cached_input_price_per_1m' | 'currency'>
export type DeletePricingInput = Pick<Pricing, 'channel_id' | 'model_name'>

export type UsageLog = { id: number; request_id: string; user_id: number | null; api_key_id: number | null; channel_id: number | null; channel_name: string; model: string; upstream_model: string; input_tokens: number; output_tokens: number; cached_input_tokens: number; total_tokens: number; unit_price_input_per_1m: string; unit_price_output_per_1m: string; total_cost: string; duration_ms: number; ttft_ms: number | null; status: string; error_code: string; client_ip: string; created_at: string }
export type Stats = { request_count: number; success_count: number; error_count: number; total_tokens: number; total_cost: string; active_user_count: number }
export type DailyStats = { stat_date: string; request_count: number; success_count: number; error_count: number; total_tokens: number; total_cost: string }
export type ChannelStats = { channel_id: number; channel_name: string; request_count: number; success_count: number; error_count: number; total_tokens: number; total_cost: string }
export type UsageAggregate = { user_id: number | null; api_key_id: number | null; channel_id: number | null; model: string; request_count: number; success_count: number; error_count: number; total_tokens: number; total_cost: string; duration_ms: number }

export type RateLimit = { id: number; rule_name: string; target_type: string; target_value: string; metric: string; limit_value: number; window_seconds: number; action: string; priority: number; enabled: boolean; extras: Record<string, unknown> }
export type RateLimitInput = Partial<Omit<RateLimit, 'id'>>
export type QuotaPolicy = { id: number; policy_name: string; scope_type: 'user' | 'api_key'; scope_id: number; period_type: 'day' | 'month'; token_limit: number | null; cost_limit: string | null; enabled: boolean }
export type QuotaPolicyInput = Partial<Omit<QuotaPolicy, 'id'>>
export type QuotaUsage = { policy_id: number; policy_name: string; scope_type: 'user' | 'api_key'; scope_id: number; period_type: 'day' | 'month'; period_start: string; period_end: string; token_limit: number | null; used_tokens: number; reserved_tokens: number; cost_limit: string | null; used_cost: string; reserved_cost: string }

export type Deleted = { deleted: true }
