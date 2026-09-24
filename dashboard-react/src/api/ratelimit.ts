import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import { DeletedSchema } from './runtime/common'
import { RateLimitListSchema, RateLimitSchema } from './runtime/rules'
import type { Deleted, ListResponse, RateLimit, RateLimitCreateInput, RateLimitUpdateInput } from '../types/api'


export const listRateLimits = () => adminGet(apiPaths.rateLimits(), { page: 1, page_size: 100 }, RateLimitListSchema)
export const createRateLimit = (input: RateLimitCreateInput) => adminSend('POST', apiPaths.rateLimits(), input, RateLimitSchema)
export const updateRateLimit = (id: number, input: RateLimitUpdateInput) => adminSend('PUT', apiPaths.rateLimit(id), input, RateLimitSchema)
export const deleteRateLimit = (id: number) => adminSend('DELETE', apiPaths.rateLimit(id), DeletedSchema)
