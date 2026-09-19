export type AdminResponse<T> = { code: number; message: string; data: T }
export type ListResponse<T> = { list: T[]; total: number }
export type Channel = { id: number; name: string; base_url: string; auth_type: string; status: number; weight: number; priority: number; balance: string | null; model_count: number }
export type Health = { channel_id: number; state: 'closed' | 'open' | 'half-open'; consecutive_failures: number; success_count: number; failure_count: number; opened_at: string | null; updated_at: string }
export type UsageLog = { id: number; request_id: string; user_id: number | null; api_key_id: number | null; channel_id: number | null; channel_name: string; model: string; total_tokens: number; total_cost: string; duration_ms: number; ttft_ms: number | null; status: string; error_code: string; created_at: string }
export type Stats = { request_count: number; success_count: number; error_count: number; total_tokens: number; total_cost: string; active_user_count: number }
export type RateLimit = { id: number; rule_name: string; target_type: string; target_value: string; metric: string; limit_value: number; window_seconds: number; action: string; priority: number; enabled: boolean }
