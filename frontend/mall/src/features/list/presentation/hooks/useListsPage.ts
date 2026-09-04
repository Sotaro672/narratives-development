// frontend/amol/src/features/list/presentation/hooks/useListsPage.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  loadListPage,
} from "../../application/loadListPage";

import {
  DEFAULT_PAGE,
  DEFAULT_PER_PAGE,
} from "../../constants";

import type {
  MallListCardItem,
} from "../../../shared/types/list";

export function useListsPage() {
  const [
    items,
    setItems,
  ] = useState<MallListCardItem[]>([]);

  const [
    page,
    setPage,
  ] = useState(DEFAULT_PAGE);

  const [
    totalPages,
    setTotalPages,
  ] = useState(1);

  const [
    isLoading,
    setIsLoading,
  ] = useState(true);

  const [
    errorMessage,
    setErrorMessage,
  ] = useState("");

  const [
    reloadKey,
    setReloadKey,
  ] = useState(0);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      setErrorMessage("");

      try {
        const result =
          await loadListPage({
            page,
            perPage:
              DEFAULT_PER_PAGE,
          });

        if (cancelled) {
          return;
        }

        setItems(result.items);

        setTotalPages(
          result.totalPages > 0
            ? result.totalPages
            : 1,
        );

        if (
          result.page > 0 &&
          result.page !== page
        ) {
          setPage(result.page);
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        setItems([]);
        setTotalPages(1);

        setErrorMessage(
          error instanceof Error
            ? error.message
            : "商品一覧の取得中にエラーが発生しました。",
        );
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [
    page,
    reloadKey,
  ]);

  const canGoPrev =
    page > 1 &&
    !isLoading;

  const canGoNext =
    page < totalPages &&
    !isLoading;

  const goPrev = useCallback(() => {
    setPage((current) =>
      Math.max(
        DEFAULT_PAGE,
        current - 1,
      ),
    );
  }, []);

  const goNext = useCallback(() => {
    setPage((current) =>
      Math.min(
        totalPages,
        current + 1,
      ),
    );
  }, [
    totalPages,
  ]);

  const reload = useCallback(() => {
    setReloadKey((current) =>
      current + 1,
    );
  }, []);

  return {
    items,
    page,
    totalPages,
    isLoading,
    errorMessage,

    canGoPrev,
    canGoNext,

    goPrev,
    goNext,
    reload,
  };
}