import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Deleted, ListResponse, RateLimit, RateLimitCreateInput, RateLimitUpdateInput } from '../types/api'


export const listRateLimits = () => adminGet<ListResponse<RateLimit>>(apiPaths.rateLimits(), { page: 1, page_size: 100 })
export const createRateLimit = (input: RateLimitCreateInput) => adminSend<RateLimit, RateLimitCreateInput>('POST', apiPaths.rateLimits(), input)
export const updateRateLimit = (id: number, input: RateLimitUpdateInput) => adminSend<RateLimit, RateLimitUpdateInput>('PUT', apiPaths.rateLimit(id), input)
export const deleteRateLimit = (id: number) => adminSend<Deleted>('DELETE', apiPaths.rateLimit(id))
