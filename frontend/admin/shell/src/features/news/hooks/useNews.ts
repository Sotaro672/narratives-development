// frontend/admin/shell/src/features/news/hooks/useNews.ts

import { useCallback, useEffect, useRef, useState } from "react";

import type {
  CreateNewsInput,
  News,
} from "../../../shared/type/news";
import {
  createNews,
  listNews,
} from "../infrastructure/newsApi";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

export function useNews() {
  const requestIdRef = useRef(0);
  const [items, setItems] = useState<News[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [publishing, setPublishing] = useState(false);
  const [publishError, setPublishError] = useState<string | null>(null);
  const [page, setPageState] = useState(DEFAULT_PAGE);
  const [perPage, setPerPageState] = useState(DEFAULT_PER_PAGE);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);

  const loadNews = useCallback(async () => {
    const requestId = ++requestIdRef.current;

    setLoading(true);
    setError(null);

    try {
      const result = await listNews({
        page,
        perPage,
      });

      if (requestId !== requestIdRef.current) {
        return;
      }

      setItems(result.items);
      setTotalCount(result.totalCount);
      setTotalPages(result.totalPages);

      if (result.page !== page) {
        setPageState(result.page);
      }

      if (result.perPage !== perPage) {
        setPerPageState(result.perPage);
      }
    } catch (cause) {
      if (requestId !== requestIdRef.current) {
        return;
      }

      setItems([]);
      setTotalCount(0);
      setTotalPages(0);
      setError(
        cause instanceof Error
          ? cause.message
          : "通知履歴の取得に失敗しました。",
      );
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [page, perPage]);

  useEffect(() => {
    void loadNews();

    return () => {
      requestIdRef.current += 1;
    };
  }, [loadNews]);

  const publish = useCallback(
    async (input: CreateNewsInput): Promise<News | null> => {
      if (publishing) {
        return null;
      }

      const title = input.title.trim();
      const body = input.body.trim();

      if (!title || !body) {
        setPublishError("タイトルと本文を入力してください。");
        return null;
      }

      setPublishing(true);
      setPublishError(null);

      try {
        const created = await createNews({
          title,
          body,
        });

        if (page === DEFAULT_PAGE) {
          await loadNews();
        } else {
          setPageState(DEFAULT_PAGE);
        }

        return created;
      } catch (cause) {
        setPublishError(
          cause instanceof Error
            ? cause.message
            : "通知に失敗しました。",
        );
        return null;
      } finally {
        setPublishing(false);
      }
    },
    [loadNews, page, publishing],
  );

  const setPage = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }

    setPageState(Math.max(1, Math.trunc(value)));
  }, []);

  const setPerPage = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }

    const normalized = Math.max(
      1,
      Math.min(100, Math.trunc(value)),
    );

    setPerPageState(normalized);
    setPageState(DEFAULT_PAGE);
  }, []);

  const reload = useCallback(async () => {
    await loadNews();
  }, [loadNews]);

  const hasPreviousPage = page > 1;
  const hasNextPage = totalPages > 0 && page < totalPages;

  return {
    items,
    loading,
    error,
    publishing,
    publishError,
    page,
    perPage,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    setPage,
    setPerPage,
    publish,
    reload,
  };
}