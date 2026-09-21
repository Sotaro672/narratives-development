// frontend/mall/src/features/trade/presentation/util/tradeChatDetail.ts

import type {
  TradeDetail,
  TradeMessage,
} from "../../../shared/types/trade";

export type TradeOrderActionKind =
  | "dispatch"
  | "start-return-consultation"
  | "respond-return-consultation"
  | "review-return-proposal"
  | "receive-return"
  | "cancel";

export function getErrorMessage(
  caught: unknown,
  fallbackMessage: string,
): string {
  if (caught instanceof Error && caught.message) {
    return caught.message;
  }

  return fallbackMessage;
}

export function getTradeTitle(productName?: string): string {
  return productName ? `${productName}/取引` : "取引";
}

export function sortTradeMessages(
  messages: TradeMessage[],
): TradeMessage[] {
  return [...messages].sort(
    (firstMessage, secondMessage) =>
      new Date(firstMessage.createdAt).getTime() -
      new Date(secondMessage.createdAt).getTime(),
  );
}

export function getTradeOrderAction(
  trade: TradeDetail | null,
): TradeOrderActionKind | null {
  if (
    !trade ||
    trade.status !== "active" ||
    trade.isCancelled
  ) {
    return null;
  }

  if (trade.viewerSide === "seller") {
    if (!trade.isDispatched) {
      return "dispatch";
    }

    switch (trade.returnStatus) {
      case "discussing":
        return "respond-return-consultation";

      case "return_shipped":
        return "receive-return";

      case "none":
      case "proposed":
      case "agreed":
      case "return_received":
      case "refund_processing":
      case "completed":
      case "disputed":
      case undefined:
        return null;
    }
  }

  if (!trade.isDispatched) {
    return "cancel";
  }

  if (trade.transferred) {
    return null;
  }

  switch (trade.returnStatus) {
    case "none":
      return "start-return-consultation";

    case "proposed":
      return "review-return-proposal";

    case "discussing":
    case "agreed":
    case "return_shipped":
    case "return_received":
    case "refund_processing":
    case "completed":
    case "disputed":
    case undefined:
      return null;
  }
}