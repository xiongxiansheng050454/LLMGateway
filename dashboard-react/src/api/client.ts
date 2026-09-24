import type { AdminResponse } from '../types/api'
import { generatedClient } from './generated/client'
import type { paths } from './generated/schema'

type AdminPath = Extract<keyof paths, `/admin/${string}`>
type ContractPath = AdminPath | `/admin/channels/${number}` | `/admin/channels/${number}/status` | `/admin/channels/${number}/balance` | `/admin/channels/${number}/health` | `/admin/channels/${number}/models` | `/admin/channels/${number}/models/${number}` | `/admin/channels/${number}/remote-models` | `/admin/channels/${number}/test` | `/admin/channels/${number}/health/reset` | `/admin/users/${number}` | `/admin/users/${number}/status` | `/admin/users/${number}/balance` | `/admin/users/${number}/recharge` | `/admin/users/${number}/balance-transactions` | `/admin/users/${number}/keys` | `/admin/users/${number}/keys/${number}` | `/admin/users/${number}/keys/${number}/reset` | `/admin/usage-logs/${number}` | `/admin/rate-limits/${number}` | `/admin/quota-policies/${number}`

export async function adminGet<T>(path: ContractPath, params: Record<string, string | number | undefined> = {}): Promise<T> {
  // Domain API modules use concrete URLs; openapi-fetch normally receives path templates.
  const response = await generatedClient.request('get' as never, path as never, { params: { query: params } } as never)
  return unwrap<T>(path, response)
}
export async function adminSend<T, B extends object = object>(method: string, path: ContractPath, body: B): Promise<T>
export async function adminSend<T>(method: string, path: ContractPath): Promise<T>
export async function adminSend<T>(method: string, path: ContractPath, body?: object): Promise<T> {
  const response = await generatedClient.request(method.toLowerCase() as never, path as never, { body } as never)
  return unwrap<T>(path, response)
}

function unwrap<T>(path: string, response: { response: Response; data?: unknown; error?: unknown }): T {
  const json = response.data as AdminResponse<T> | undefined
  if (!response.response.ok || !json || json.code !== 0) {
    const error = response.error as AdminResponse<unknown> | undefined
    throw new Error(error?.message || json?.message || `${path} failed`)
  }
  return json.data
}
