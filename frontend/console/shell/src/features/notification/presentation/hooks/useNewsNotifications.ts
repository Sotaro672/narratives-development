// frontend/console/shell/src/features/notification/presentation/hooks/useNewsNotifications.ts

import { useCallback, useEffect, useState } from "react";

import {
  listNewsApi,
  markNewsReadApi,
} from "../../infrastructure/newsApi";
import type {
  News,
  NewsPage,
  NewsReadResponse,
} from "../../../../shared/types/news";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

export type UseNewsNotificationsParams = {
  page?: number;
  perPage?: number;
  enabled?: boolean;
};

export type UseNewsNotificationsResult = {
  notifications: News[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
  loading: boolean;
  error: string | null;
  markingReadId: string | null;
  reload: () => Promise<void>;
  markRead: (newsId: string) => Promise<NewsReadResponse | null>;
};

function createEmptyResult(
  page: number,
  perPage: number,
): NewsPage {
  return {
    items: [],
    totalCount: 0,
    totalPages: 0,
    page,
    perPage,
  };
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "システム通知の取得に失敗しました。";
}

export function useNewsNotifications(
  params: UseNewsNotificationsParams = {},
): UseNewsNotificationsResult {
  const page = params.page ?? DEFAULT_PAGE;
  const perPage = params.perPage ?? DEFAULT_PER_PAGE;
  const enabled = params.enabled ?? true;

  const [result, setResult] = useState<NewsPage>(
    () => createEmptyResult(page, perPage),
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [markingReadId, setMarkingReadId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!enabled) {
      setResult(createEmptyResult(page, perPage));
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await listNewsApi({
        page,
        perPage,
      });

      setResult(response);
    } catch (loadError) {
      setResult(createEmptyResult(page, perPage));
      setError(resolveErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [enabled, page, perPage]);

  useEffect(() => {
    void load();
  }, [load]);

  const markRead = useCallback(
    async (
      newsId: string,
    ): Promise<NewsReadResponse | null> => {
      const normalizedNewsId = newsId.trim();

      if (!normalizedNewsId || markingReadId !== null) {
        return null;
      }

      setMarkingReadId(normalizedNewsId);
      setError(null);

      try {
        const read = await markNewsReadApi(normalizedNewsId);

        setResult((current) => ({
          ...current,
          items: current.items.map((news) =>
            news.id === read.newsId
              ? {
                  ...news,
                  isRead: true,
                  readAt: read.readAt,
                }
              : news,
          ),
        }));

        await load();

        return read;
      } catch (markReadError) {
        setError(resolveErrorMessage(markReadError));
        return null;
      } finally {
        setMarkingReadId(null);
      }
    },
    [load, markingReadId],
  );

  return {
    notifications: result.items,
    totalCount: result.totalCount,
    totalPages: result.totalPages,
    page: result.page,
    perPage: result.perPage,
    loading,
    error,
    markingReadId,
    reload: load,
    markRead,
  };
}

export default useNewsNotifications;