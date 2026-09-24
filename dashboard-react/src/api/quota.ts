import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import { DeletedSchema } from './runtime/common'
import { QuotaPolicyListSchema, QuotaPolicySchema, QuotaUsageListSchema } from './runtime/rules'
import type { Deleted, ListResponse, QuotaPolicy, QuotaPolicyCreateInput, QuotaPolicyUpdateInput, QuotaUsage } from '../types/api'


export const listQuotaPolicies = () => adminGet(apiPaths.quotaPolicies(), { page: 1, page_size: 100 }, QuotaPolicyListSchema)
export const listQuotaUsage = () => adminGet(apiPaths.quotaUsage(), { page: 1, page_size: 100 }, QuotaUsageListSchema)
export const createQuotaPolicy = (input: QuotaPolicyCreateInput) => adminSend('POST', apiPaths.quotaPolicies(), input, QuotaPolicySchema)
export const updateQuotaPolicy = (id: number, input: QuotaPolicyUpdateInput) => adminSend('PUT', apiPaths.quotaPolicy(id), input, QuotaPolicySchema)
export const deleteQuotaPolicy = (id: number) => adminSend('DELETE', apiPaths.quotaPolicy(id), DeletedSchema)
