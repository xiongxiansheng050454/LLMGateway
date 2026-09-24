import test from 'node:test'
import assert from 'node:assert/strict'

test('query parameter contract keeps false and omits undefined', () => {
  const params = { page: 1, enabled: false, model: undefined }
  const query = new URLSearchParams(Object.entries(params).filter(([, value]) => value !== undefined).map(([key, value]) => [key, String(value)]))
  assert.equal(query.toString(), 'page=1&enabled=false')
})

test('date range uses a half-open next-day end boundary', () => {
  const next = new Date('2026-09-16T00:00:00Z')
  next.setUTCDate(next.getUTCDate() + 1)
  assert.equal(next.toISOString(), '2026-09-17T00:00:00.000Z')
})
