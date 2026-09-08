// frontend/mall/src/features/news/hooks/useNewsQuery.ts

import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  fetchMeNews,
  fetchMeNewsUnreadCount,
  markMeNewsRead,
} from "../api/newsApi";
import type {
  News,
  NewsPage,
  NewsReadResponse,
  NewsUnreadCountResponse,
} from "../../shared/types/news";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

export type UseNewsQueryParams = {
  page?: number;
  perPage?: number;
  enabled?: boolean;
};

export type UseNewsUnreadCountQueryParams = {
  enabled?: boolean;
};

type NewsListKeyParams = {
  page: number;
  perPage: number;
};

export const newsQueryKeys = {
  all: ["news"] as const,

  me: () => [
    ...newsQueryKeys.all,
    "me",
  ] as const,

  lists: () => [
    ...newsQueryKeys.me(),
    "list",
  ] as const,

  list: (params: NewsListKeyParams) => [
    ...newsQueryKeys.lists(),
    params,
  ] as const,

  unreadCount: () => [
    ...newsQueryKeys.me(),
    "unread-count",
  ] as const,
};

function normalizePositiveInteger(
  value: number | undefined,
  fallback: number,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value)
  ) {
    return fallback;
  }

  const normalized = Math.floor(value);
  return normalized > 0 ? normalized : fallback;
}

function markNewsAsReadInPage(
  current: NewsPage | undefined,
  read: NewsReadResponse,
): NewsPage | undefined {
  if (!current) {
    return current;
  }

  let changed = false;

  const items = current.items.map(
    (news): News => {
      if (
        news.id !== read.newsId ||
        news.isRead === true
      ) {
        return news;
      }

      changed = true;

      return {
        ...news,
        isRead: true,
        readAt: read.readAt,
      };
    },
  );

  if (!changed) {
    return current;
  }

  return {
    ...current,
    items,
  };
}

export function useNewsQuery(
  params: UseNewsQueryParams = {},
) {
  const page = normalizePositiveInteger(
    params.page,
    DEFAULT_PAGE,
  );
  const perPage = normalizePositiveInteger(
    params.perPage,
    DEFAULT_PER_PAGE,
  );
  const enabled = params.enabled ?? true;

  const keyParams: NewsListKeyParams = {
    page,
    perPage,
  };

  return useQuery({
    queryKey: newsQueryKeys.list(keyParams),
    queryFn: ({ signal }) =>
      fetchMeNews({
        page,
        perPage,
        signal,
      }),
    enabled,
  });
}

export function useNewsUnreadCountQuery(
  params: UseNewsUnreadCountQueryParams = {},
) {
  const enabled = params.enabled ?? true;

  const query = useQuery({
    queryKey: newsQueryKeys.unreadCount(),
    queryFn: ({ signal }) =>
      fetchMeNewsUnreadCount(signal),
    enabled,
  });

  const unreadCount =
    typeof query.data?.unreadCount === "number" &&
    Number.isFinite(query.data.unreadCount)
      ? Math.max(
          0,
          Math.floor(query.data.unreadCount),
        )
      : 0;

  return {
    ...query,
    unreadCount,
  };
}

export function useMarkNewsReadMutation() {
  const queryClient = useQueryClient();

  return useMutation<
    NewsReadResponse,
    Error,
    string
  >({
    mutationFn: async (
      newsId: string,
    ): Promise<NewsReadResponse> => {
      const normalizedNewsId = newsId.trim();

      if (!normalizedNewsId) {
        throw new Error(
          "newsId が空のため既読化できません。",
        );
      }

      return markMeNewsRead(normalizedNewsId);
    },

    onSuccess: (read) => {
      queryClient.setQueriesData<NewsPage>(
        {
          queryKey: newsQueryKeys.lists(),
        },
        (current) =>
          markNewsAsReadInPage(
            current,
            read,
          ),
      );

      queryClient.setQueryData<NewsUnreadCountResponse>(
        newsQueryKeys.unreadCount(),
        (current) => {
          if (!current) {
            return current;
          }

          const unreadCount =
            typeof current.unreadCount === "number" &&
            Number.isFinite(current.unreadCount)
              ? Math.max(
                  0,
                  Math.floor(current.unreadCount) - 1,
                )
              : 0;

          return {
            ...current,
            unreadCount,
          };
        },
      );
    },

    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: newsQueryKeys.lists(),
        }),
        queryClient.invalidateQueries({
          queryKey: newsQueryKeys.unreadCount(),
        }),
      ]);
    },
  });
}

export default useNewsQuery;