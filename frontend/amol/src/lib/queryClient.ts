// frontend/amol/src/lib/queryClient.ts

import {
  QueryClient,
} from "@tanstack/react-query";

import {
  HttpError,
} from "./http";

function shouldRetryQuery(
  failureCount: number,
  error: unknown,
): boolean {
  if (
    error instanceof HttpError &&
    error.status >= 400 &&
    error.status < 500
  ) {
    return false;
  }

  return failureCount < 1;
}

export const queryClient =
  new QueryClient({
    defaultOptions: {
      queries: {
        staleTime:
          30_000,

        gcTime:
          5 * 60_000,

        retry:
          shouldRetryQuery,

        refetchOnWindowFocus:
          false,
      },
    },
  });