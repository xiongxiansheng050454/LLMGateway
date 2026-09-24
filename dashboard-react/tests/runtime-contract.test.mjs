import test from 'node:test'
import assert from 'node:assert/strict'
import { z } from 'zod'

const money = z.string().regex(/^-?\d+(\.\d+)?$/)
const balance = z.object({ available_balance: money, frozen_balance: money })
const user = z.object({ id: z.number().int(), nickname: z.string(), user_group: z.string(), status: z.enum(['active', 'suspended']), balance })
const usageLog = z.object({ total_cost: money, ttft_ms: z.number().int().nullable().optional(), status: z.string() })
const list = z.object({ list: z.array(user), total: z.number().int().nonnegative() })

test('high-risk response schemas reject malformed money and enums', () => {
  assert.equal(user.safeParse({ id: 1, nickname: 'A', user_group: 'default', status: 'deleted', balance: { available_balance: '1.00', frozen_balance: '0.00' } }).success, false)
  assert.equal(usageLog.safeParse({ total_cost: 1, ttft_ms: null, status: 'success' }).success, false)
})

test('high-risk response schemas accept nullable and list fields', () => {
  const item = { id: 1, nickname: 'A', user_group: 'default', status: 'active', balance: { available_balance: '1.00', frozen_balance: '0.00' } }
  assert.equal(user.safeParse(item).success, true)
  assert.equal(list.safeParse({ list: [item], total: 1 }).success, true)
  assert.equal(usageLog.safeParse({ total_cost: '0.00', ttft_ms: null, status: 'success' }).success, true)
})
