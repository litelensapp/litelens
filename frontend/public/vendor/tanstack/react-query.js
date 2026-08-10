/*
 * Vendor shim for the "@tanstack/react-query" bare specifier. See
 * ../react.js for why this indirection exists.
 *
 * Sharing the host's own react-query module instance (not just the same
 * QueryClient value) is required, not optional: QueryClientProvider/useQuery
 * read a React Context object created inside this module, and a Context
 * object is only == itself across module instances if there is exactly one
 * loaded instance. A plugin bundling its own copy of react-query would get
 * "No QueryClient set" at runtime even though the host's provider wraps it.
 *
 * This export list is generated from the host's actual @tanstack/react-query
 * build (Object.keys() of an import of the package). Regenerate it whenever
 * the react-query package's public API surface changes.
 */
if (!window.__LITELENS_VENDOR__) {
  throw new Error(
    "window.__LITELENS_VENDOR__ is not set — src/main.tsx must run before any plugin is dynamically imported."
  );
}
const ReactQuery = window.__LITELENS_VENDOR__.reactQuery;

export const {
  CancelledError,
  HydrationBoundary,
  InfiniteQueryObserver,
  IsRestoringProvider,
  Mutation,
  MutationCache,
  MutationObserver,
  QueriesObserver,
  Query,
  QueryCache,
  QueryClient,
  QueryClientContext,
  QueryClientProvider,
  QueryErrorResetBoundary,
  QueryObserver,
  dataTagErrorSymbol,
  dataTagSymbol,
  defaultScheduler,
  defaultShouldDehydrateMutation,
  defaultShouldDehydrateQuery,
  dehydrate,
  environmentManager,
  experimental_streamedQuery,
  focusManager,
  hashKey,
  hydrate,
  infiniteQueryOptions,
  isCancelledError,
  isServer,
  keepPreviousData,
  matchMutation,
  matchQuery,
  mutationOptions,
  noop,
  notifyManager,
  onlineManager,
  partialMatchKey,
  queryOptions,
  replaceEqualDeep,
  shouldThrowError,
  skipToken,
  timeoutManager,
  unsetMarker,
  useInfiniteQuery,
  useIsFetching,
  useIsMutating,
  useIsRestoring,
  useMutation,
  useMutationState,
  usePrefetchInfiniteQuery,
  usePrefetchQuery,
  useQueries,
  useQuery,
  useQueryClient,
  useQueryErrorResetBoundary,
  useSuspenseInfiniteQuery,
  useSuspenseQueries,
  useSuspenseQuery,
} = ReactQuery;
