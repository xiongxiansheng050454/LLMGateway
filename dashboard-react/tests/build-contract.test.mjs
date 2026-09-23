import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('React dashboard declares the root production path and typed admin client', async () => {
  const vite = await readFile(new URL('../vite.config.ts', import.meta.url), 'utf8')
  const client = await readFile(new URL('../src/api/client.ts', import.meta.url), 'utf8')
  assert.match(vite, /base:\s*['"]\//)
  assert.match(client, /AdminResponse/)
})
