// frontend/admin/shell/src/features/trade/presentation/hooks/useTradeReturnReport.ts

import { useCallback, useEffect, useState } from "react";

import type {
  ReportCase,
  ReportItem,
} from "../../../../shared/type/report";
import { getReportIfExists } from "../../../report/infrastructure/reportApi";

export function useTradeReturnReport(
  tradeId: string | undefined,
) {
  const [reportCase, setReportCase] = useState<ReportCase | null>(null);
  const [report, setReport] = useState<ReportItem | null>(null);
  const [loading, setLoading] = useState(Boolean(tradeId?.trim()));
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedTradeId = tradeId?.trim() ?? "";

    if (!normalizedTradeId) {
      setReportCase(null);
      setReport(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getReportIfExists(
        `trade_${normalizedTradeId}`,
      );

      setReportCase(result?.case ?? null);
      setReport(result?.reports.items[0] ?? null);
    } catch (cause) {
      setReportCase(null);
      setReport(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "返品報告の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [tradeId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    reportCase,
    report,
    loading,
    error,
    reload,
  };
}