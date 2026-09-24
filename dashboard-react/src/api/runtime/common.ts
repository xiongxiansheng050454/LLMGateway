import { z } from 'zod'

export const MoneySchema = z.string().regex(/^-?\d+(\.\d+)?$/, 'must be a decimal string')
export const DeletedSchema = z.object({ deleted: z.literal(true) })
export const BalanceSchema = z.object({ available_balance: MoneySchema, frozen_balance: MoneySchema })
export const AdminErrorSchema = z.object({ code: z.number().int().positive(), message: z.string(), data: z.record(z.never()) })

export const listSchema = <T extends z.ZodTypeAny>(item: T) => z.object({ list: z.array(item), total: z.number().int().nonnegative() })

export const schemaName = (path: string, name: string) => `${path} returned invalid ${name}`
