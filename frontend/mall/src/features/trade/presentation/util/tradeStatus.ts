// frontend/mall/src/features/trade/presentation/util/tradeStatus.ts

import type { TradeDetail } from "../../../shared/types/trade";

export type TradeStatusSource = Pick<
  TradeDetail,
  | "viewerSide"
  | "status"
  | "isCancelled"
  | "isDispatched"
  | "returnStatus"
  | "transferred"
>;

export function getTradeStatusLabel(
  trade: TradeStatusSource,
): string {
  if (trade.transferred) {
    return "移譲済";
  }

  if (trade.isCancelled) {
    return "キャンセル";
  }

  switch (trade.returnStatus) {
    case "discussing":
      return "返品相談中";

    case "proposed":
      return "返品条件提示中";

    case "agreed":
      return "返品合意済み";

    case "return_shipped":
      return "返送中";

    case "return_received":
      return "返品受領済み";

    case "refund_processing":
      return "返金処理中";

    case "completed":
      return "返品完了";

    case "disputed":
      return "運営確認中";

    case "none":
      break;
  }

  if (
    trade.viewerSide === "seller" &&
    trade.status === "active" &&
    !trade.isDispatched
  ) {
    return "発送待ち";
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