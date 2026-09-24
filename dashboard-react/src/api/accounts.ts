import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import { BalanceAfterSchema, BalanceTransactionListSchema, BalanceSchema, DeletedSchema, KeyListSchema, KeySchema, KeySecretSchema, UserListSchema, UserSchema } from './runtime/accounts'
import type { Balance, BalanceTransaction, ClientKey, KeyCreateInput, KeyUpdateInput, Deleted, KeySecret, ListResponse, RechargeInput, User, UserCreateInput, UserStatusInput, UserUpdateInput } from '../types/api'


export const listUsers = () => adminGet(apiPaths.users(), { page: 1, page_size: 100 }, UserListSchema)
export const createUser = (input: UserCreateInput) => adminSend('POST', apiPaths.users(), input, UserSchema)
export const updateUser = (id: number, input: UserUpdateInput) => adminSend('PUT', apiPaths.user(id), input, UserSchema)
export const updateUserStatus = (id: number, input: UserStatusInput) => adminSend('PUT', apiPaths.userStatus(id), input, UserSchema)
export const deleteUser = (id: number) => adminSend('DELETE', apiPaths.user(id), DeletedSchema)
export const rechargeUser = (id: number, input: RechargeInput) => adminSend('POST', apiPaths.userRecharge(id), input, BalanceAfterSchema)
export const getUserBalance = (id: number) => adminGet(apiPaths.userBalance(id), BalanceSchema)
export const listBalanceTransactions = (id: number) => adminGet(apiPaths.userTransactions(id), { page: 1, page_size: 20 }, BalanceTransactionListSchema)
export const listKeys = () => adminGet(apiPaths.allKeys(), { page: 1, page_size: 100 }, KeyListSchema)
export const listUserKeys = (userId: number) => adminGet(apiPaths.userKeys(userId), { page: 1, page_size: 50 }, KeyListSchema)
export const createUserKey = (userId: number, input: KeyCreateInput) => adminSend('POST', apiPaths.userKeys(userId), input, KeySecretSchema)
export const updateUserKey = (userId: number, keyId: number, input: KeyUpdateInput) => adminSend('PUT', apiPaths.userKey(userId, keyId), input, KeySchema)
export const resetUserKey = (userId: number, keyId: number) => adminSend('POST', apiPaths.userKeyReset(userId, keyId), KeySecretSchema)
export const deleteUserKey = (userId: number, keyId: number) => adminSend('DELETE', apiPaths.userKey(userId, keyId), DeletedSchema)
