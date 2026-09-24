import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Deleted, ListResponse, RateLimit, RateLimitInput } from '../types/api'


export const listRateLimits = () => adminGet<ListResponse<RateLimit>>(apiPaths.rateLimits(), { page: 1, page_size: 100 })
export const createRateLimit = (input: RateLimitInput) => adminSend<RateLimit, RateLimitInput>('POST', apiPaths.rateLimits(), input)
export const updateRateLimit = (id: number, input: RateLimitInput) => adminSend<RateLimit, RateLimitInput>('PUT', apiPaths.rateLimit(id), input)
export const deleteRateLimit = (id: number) => adminSend<Deleted>('DELETE', apiPaths.rateLimit(id))
