import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Balance, BalanceTransaction, ClientKey, KeyCreateInput, KeyUpdateInput, Deleted, KeySecret, ListResponse, RechargeInput, User, UserCreateInput, UserStatusInput, UserUpdateInput } from '../types/api'


export const listUsers = () => adminGet<ListResponse<User>>(apiPaths.users(), { page: 1, page_size: 100 })
export const createUser = (input: UserCreateInput) => adminSend<User, UserCreateInput>('POST', apiPaths.users(), input)
export const updateUser = (id: number, input: UserUpdateInput) => adminSend<User, UserUpdateInput>('PUT', apiPaths.user(id), input)
export const updateUserStatus = (id: number, input: UserStatusInput) => adminSend<User, UserStatusInput>('PUT', apiPaths.userStatus(id), input)
export const deleteUser = (id: number) => adminSend<Deleted>('DELETE', apiPaths.user(id))
export const rechargeUser = (id: number, input: RechargeInput) => adminSend<{ balance_after: string }, RechargeInput>('POST', apiPaths.userRecharge(id), input)
export const getUserBalance = (id: number) => adminGet<Balance>(apiPaths.userBalance(id))
export const listBalanceTransactions = (id: number) => adminGet<ListResponse<BalanceTransaction>>(apiPaths.userTransactions(id), { page: 1, page_size: 20 })
export const listKeys = () => adminGet<ListResponse<ClientKey>>(apiPaths.allKeys(), { page: 1, page_size: 100 })
export const listUserKeys = (userId: number) => adminGet<ListResponse<ClientKey>>(apiPaths.userKeys(userId), { page: 1, page_size: 50 })
export const createUserKey = (userId: number, input: KeyCreateInput) => adminSend<KeySecret, KeyCreateInput>('POST', apiPaths.userKeys(userId), input)
export const updateUserKey = (userId: number, keyId: number, input: KeyUpdateInput) => adminSend<ClientKey, KeyUpdateInput>('PUT', apiPaths.userKey(userId, keyId), input)
export const resetUserKey = (userId: number, keyId: number) => adminSend<KeySecret>('POST', apiPaths.userKeyReset(userId, keyId))
export const deleteUserKey = (userId: number, keyId: number) => adminSend<Deleted>('DELETE', apiPaths.userKey(userId, keyId))
