import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Deleted, ListResponse, QuotaPolicy, QuotaPolicyInput, QuotaUsage } from '../types/api'


export const listQuotaPolicies = () => adminGet<ListResponse<QuotaPolicy>>(apiPaths.quotaPolicies(), { page: 1, page_size: 100 })
export const listQuotaUsage = () => adminGet<ListResponse<QuotaUsage>>(apiPaths.quotaUsage(), { page: 1, page_size: 100 })
export const createQuotaPolicy = (input: QuotaPolicyInput) => adminSend<QuotaPolicy, QuotaPolicyInput>('POST', apiPaths.quotaPolicies(), input)
export const updateQuotaPolicy = (id: number, input: QuotaPolicyInput) => adminSend<QuotaPolicy, QuotaPolicyInput>('PUT', apiPaths.quotaPolicy(id), input)
export const deleteQuotaPolicy = (id: number) => adminSend<Deleted>('DELETE', apiPaths.quotaPolicy(id))
