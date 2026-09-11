// frontend/mall/src/features/trade/presentation/util/tradeChatDetail.ts

import type { TradeDetail, TradeMessage } from "../../../shared/types/trade";

export type TradeOrderActionKind = "dispatch" | "return" | "cancel";

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
  if (!trade || trade.status !== "active" || trade.isCancelled) {
    return null;
  }

  if (trade.viewerSide === "seller") {
    return trade.isDispatched ? null : "dispatch";
  }

  if (!trade.isDispatched) {
    return "cancel";
  }

  if (
    !trade.transferred &&
    !trade.isReturnRequested &&
    !trade.isReturnCompleted
  ) {
    return "return";
  }

  return null;
}