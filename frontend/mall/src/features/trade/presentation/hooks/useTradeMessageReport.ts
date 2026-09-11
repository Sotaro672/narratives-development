// frontend/mall/src/features/trade/presentation/hooks/useTradeMessageReport.ts

import { useCallback } from "react";

import { useReport } from "../../../report/hooks/useReport";
import type {
  TradeDetail,
  TradeMessage,
} from "../../../shared/types/trade";

type UseTradeMessageReportInput = {
  tradeId: string;
  trade: TradeDetail | null;
};

export function useTradeMessageReport({
  tradeId,
  trade,
}: UseTradeMessageReportInput) {
  const report = useReport();

  const openMessageReport = useCallback(
    (message: TradeMessage): void => {
      const normalizedTradeId = tradeId.trim();
      const messageId = message.id.trim();

      if (!normalizedTradeId || !messageId || !trade) {
        return;
      }

      if (
        message.senderSide === "system" ||
        message.senderSide === trade.viewerSide
      ) {
        return;
      }

      report.openTradeMessageReport({
        tradeId: normalizedTradeId,
        messageId,
      });
    },
    [
      report.openTradeMessageReport,
      trade,
      tradeId,
    ],
  );

  return {
    ...report,
    openMessageReport,
  };
}

export default useTradeMessageReport;