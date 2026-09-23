import { adminGet, adminSend } from './client'; import type { Channel, Deleted, Health, ListResponse } from '../types/api'
export const channels = () => adminGet<ListResponse<Channel>>('/channels', { page: 1, page_size: 100 })
export const health = () => adminGet<ListResponse<Health>>('/channels/health')
export const resetHealth = (id: number) => adminSend<Deleted>('POST', `/channels/${id}/health/reset`)
