import { useQuery, type UseQueryOptions } from '@tanstack/react-query'
export function useAdminQuery<T>(options: UseQueryOptions<T>) { return useQuery(options) }
