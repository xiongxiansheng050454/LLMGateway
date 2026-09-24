import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Deleted, ListResponse, QuotaPolicy, QuotaPolicyCreateInput, QuotaPolicyUpdateInput, QuotaUsage } from '../types/api'


export const listQuotaPolicies = () => adminGet<ListResponse<QuotaPolicy>>(apiPaths.quotaPolicies(), { page: 1, page_size: 100 })
export const listQuotaUsage = () => adminGet<ListResponse<QuotaUsage>>(apiPaths.quotaUsage(), { page: 1, page_size: 100 })
export const createQuotaPolicy = (input: QuotaPolicyCreateInput) => adminSend<QuotaPolicy, QuotaPolicyCreateInput>('POST', apiPaths.quotaPolicies(), input)
export const updateQuotaPolicy = (id: number, input: QuotaPolicyUpdateInput) => adminSend<QuotaPolicy, QuotaPolicyUpdateInput>('PUT', apiPaths.quotaPolicy(id), input)
export const deleteQuotaPolicy = (id: number) => adminSend<Deleted>('DELETE', apiPaths.quotaPolicy(id))
