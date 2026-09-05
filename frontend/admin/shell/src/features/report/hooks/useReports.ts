// frontend/admin/shell/src/features/report/hooks/useReports.ts

import { useCallback, useEffect, useRef, useState } from "react";

import type {
  ReviewReportCase,
  ReviewReportCaseStatus,
  ReviewReportCaseSort,
  ReviewReportSortOrder,
  ReviewReportTargetType,
} from "../../../shared/type/reviewReport";
import { listReviewReports } from "../infrastructure/reportApi";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 50;
const DEFAULT_SORT: ReviewReportCaseSort = "updatedAt";
const DEFAULT_ORDER: ReviewReportSortOrder = "desc";

export function useReports() {
  const requestIdRef = useRef(0);
  const [items, setItems] = useState<ReviewReportCase[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatusState] = useState<ReviewReportCaseStatus | undefined>("PENDING");
  const [targetType, setTargetTypeState] = useState<ReviewReportTargetType | undefined>(undefined);
  const [page, setPageState] = useState(DEFAULT_PAGE);
  const [perPage, setPerPageState] = useState(DEFAULT_PER_PAGE);
  const [sort, setSortState] = useState<ReviewReportCaseSort>(DEFAULT_SORT);
  const [order, setOrderState] = useState<ReviewReportSortOrder>(DEFAULT_ORDER);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);

  const loadReports = useCallback(async () => {
    const requestId = ++requestIdRef.current;

    setLoading(true);
    setError(null);

    try {
      const result = await listReviewReports({
        status,
        targetType,
        page,
        perPage,
        sort,
        order,
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
          : "通報一覧の取得に失敗しました。",
      );
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [
    status,
    targetType,
    page,
    perPage,
    sort,
    order,
  ]);

  useEffect(() => {
    void loadReports();

    return () => {
      requestIdRef.current += 1;
    };
  }, [loadReports]);

  const setStatus = useCallback(
    (value: ReviewReportCaseStatus | undefined) => {
      setStatusState(value);
      setPageState(DEFAULT_PAGE);
    },
    [],
  );

  const setTargetType = useCallback(
    (value: ReviewReportTargetType | undefined) => {
      setTargetTypeState(value);
      setPageState(DEFAULT_PAGE);
    },
    [],
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
      Math.min(200, Math.trunc(value)),
    );

    setPerPageState(normalized);
    setPageState(DEFAULT_PAGE);
  }, []);

  const setSort = useCallback(
    (value: ReviewReportCaseSort) => {
      setSortState(value);
      setPageState(DEFAULT_PAGE);
    },
    [],
  );

  const setOrder = useCallback(
    (value: ReviewReportSortOrder) => {
      setOrderState(value);
      setPageState(DEFAULT_PAGE);
    },
    [],
  );

  const resetFilters = useCallback(() => {
    setStatusState("PENDING");
    setTargetTypeState(undefined);
    setPageState(DEFAULT_PAGE);
    setPerPageState(DEFAULT_PER_PAGE);
    setSortState(DEFAULT_SORT);
    setOrderState(DEFAULT_ORDER);
  }, []);

  const reload = useCallback(async () => {
    await loadReports();
  }, [loadReports]);

  const hasPreviousPage = page > 1;
  const hasNextPage = totalPages > 0 && page < totalPages;

  return {
    items,
    loading,
    error,
    status,
    targetType,
    page,
    perPage,
    sort,
    order,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    setStatus,
    setTargetType,
    setPage,
    setPerPage,
    setSort,
    setOrder,
    resetFilters,
    reload,
  };
}