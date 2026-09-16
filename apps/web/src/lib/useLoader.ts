import { useQuery, type QueryKey, type UseQueryOptions } from '@tanstack/react-query';

type LoaderOptions<TData> = Omit<
  UseQueryOptions<TData, Error, TData, QueryKey>,
  'queryKey' | 'queryFn'
>;

export function useLoader<TData>(
  queryKey: QueryKey,
  queryFn: () => Promise<TData>,
  options?: LoaderOptions<TData>,
) {
  const query = useQuery({ queryKey, queryFn, ...options });
  return {
    data: query.data,
    loading: query.isLoading,
    error: query.error,
    isError: query.isError,
    refetch: query.refetch,
    query,
  };
}
