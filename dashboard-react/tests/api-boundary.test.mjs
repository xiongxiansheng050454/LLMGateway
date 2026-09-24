import test from 'node:test'
import assert from 'node:assert/strict'
import { readdir, readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

test('pages do not depend on the transport client', async () => {
  const pagesDir = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'pages')
  const files = await readdir(pagesDir)
  for (const file of files.filter(name => name.endsWith('.tsx'))) {
    const source = await readFile(join(pagesDir, file), 'utf8')
    assert.doesNotMatch(source, /api\/client|adminGet|adminSend|catalogRequest|accountGet|accountRequest|rateLimitRequest|quotaRequest/, `${file} bypasses domain API modules`)
  }
})
