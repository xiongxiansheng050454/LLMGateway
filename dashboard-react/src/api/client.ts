import type { AdminResponse } from '../types/api'
const base = new URLSearchParams(location.search).get('api_base')?.replace(/\/$/, '') || '/admin'
export async function adminGet<T>(path: string, params: Record<string, string | number | undefined> = {}): Promise<T> {
  const url = new URL(base + path, location.origin); Object.entries(params).forEach(([k, v]) => v !== undefined && url.searchParams.set(k, String(v)))
  const response = await fetch(url, { headers: { Accept: 'application/json' } }); const json = await response.json() as AdminResponse<T>
  if (!response.ok || json.code !== 0) throw new Error(json.message || `${path} failed`); return json.data
}
export async function adminSend<T>(method: string, path: string, body?: unknown): Promise<T> {
  const response = await fetch(base + path, { method, headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) }); const json = await response.json() as AdminResponse<T>
  if (!response.ok || json.code !== 0) throw new Error(json.message || `${path} failed`); return json.data
}
