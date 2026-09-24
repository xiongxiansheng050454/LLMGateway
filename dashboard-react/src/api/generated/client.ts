import createClient from 'openapi-fetch'
import type { paths } from './schema'

export const generatedClient = createClient<paths>({
  baseUrl: '',
  headers: { Accept: 'application/json' },
})
