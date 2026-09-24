import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Balance, BalanceTransaction, ClientKey, CreateKeyInput, Deleted, KeySecret, ListResponse, RechargeInput, User, UserInput } from '../types/api'


export const listUsers = () => adminGet<ListResponse<User>>(apiPaths.users(), { page: 1, page_size: 100 })
export const createUser = (input: UserInput) => adminSend<User, UserInput>('POST', apiPaths.users(), input)
export const updateUser = (id: number, input: UserInput) => adminSend<User, UserInput>('PUT', apiPaths.user(id), input)
export const updateUserStatus = (id: number, status: string) => adminSend<User, { status: string }>('PUT', apiPaths.userStatus(id), { status })
export const deleteUser = (id: number) => adminSend<Deleted>('DELETE', apiPaths.user(id))
export const rechargeUser = (id: number, input: RechargeInput) => adminSend<{ balance_after: string }, RechargeInput>('POST', apiPaths.userRecharge(id), input)
export const getUserBalance = (id: number) => adminGet<Balance>(apiPaths.userBalance(id))
export const listBalanceTransactions = (id: number) => adminGet<ListResponse<BalanceTransaction>>(apiPaths.userTransactions(id), { page: 1, page_size: 20 })
export const listKeys = () => adminGet<ListResponse<ClientKey>>(apiPaths.allKeys(), { page: 1, page_size: 100 })
export const listUserKeys = (userId: number) => adminGet<ListResponse<ClientKey>>(apiPaths.userKeys(userId), { page: 1, page_size: 50 })
export const createUserKey = (userId: number, input: CreateKeyInput) => adminSend<KeySecret, CreateKeyInput>('POST', apiPaths.userKeys(userId), input)
export const updateUserKey = (userId: number, keyId: number, input: { is_active: boolean }) => adminSend<ClientKey, { is_active: boolean }>('PUT', apiPaths.userKey(userId, keyId), input)
export const resetUserKey = (userId: number, keyId: number) => adminSend<KeySecret>('POST', apiPaths.userKeyReset(userId, keyId))
export const deleteUserKey = (userId: number, keyId: number) => adminSend<Deleted>('DELETE', apiPaths.userKey(userId, keyId))
