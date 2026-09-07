// frontend/admin/shell/src/features/report/hooks/useReportPendingCount.ts

import { useCallback, useEffect, useRef, useState } from "react";

import { listReports } from "../../infrastructure/reportApi";

export function useReportPendingCount() {
  const requestIdRef = useRef(0);
  const [pendingCount, setPendingCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refreshPendingCount = useCallback(async () => {
    const requestId = ++requestIdRef.current;

    setLoading(true);
    setError(null);

    try {
      const result = await listReports({
        page: 1,
        perPage: 1,
        status: "PENDING",
        sort: "updatedAt",
        order: "desc",
      });

      if (requestId !== requestIdRef.current) {
        return;
      }

      setPendingCount(result.totalCount);
    } catch (cause) {
      if (requestId !== requestIdRef.current) {
        return;
      }

      setError(
        cause instanceof Error
          ? cause.message
          : "未対応通報件数の取得に失敗しました。",
      );
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void refreshPendingCount();

    return () => {
      requestIdRef.current += 1;
    };
  }, [refreshPendingCount]);

  return {
    pendingCount,
    loading,
    error,
    refreshPendingCount,
  };
}