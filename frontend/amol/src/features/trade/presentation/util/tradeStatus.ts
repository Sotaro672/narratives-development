// frontend/amol/src/features/trade/presentation/util/tradeStatus.ts

import type { TradeDetail } from "../../../shared/types/trade";

export type TradeStatusSource = Pick<
  TradeDetail,
  | "status"
  | "isCancelled"
  | "isDispatched"
  | "isReturnRequested"
  | "isReturnCompleted"
>;

export function getTradeStatusLabel(
  trade: TradeStatusSource,
): string {
  if (trade.isCancelled) {
    return "キャンセル";
  }

  if (trade.isReturnCompleted) {
    return "返品済み";
  }

  if (trade.isReturnRequested) {
    return "返品申請済み";
  }

  if (trade.isDispatched) {
    return "発送済み";
  }

  switch (trade.status) {
    case "active":
      return "取引中";

    case "closed":
      return "取引終了";

    default:
      return "";
  }
}