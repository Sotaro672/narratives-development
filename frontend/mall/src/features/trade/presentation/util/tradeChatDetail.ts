// frontend/mall/src/features/trade/presentation/util/tradeChatDetail.ts

import type {
  TradeDetail,
  TradeMessage,
  TradeReturnProposal,
} from "../../../shared/types/trade";

export type TradeOrderActionKind =
  | "dispatch"
  | "start-return-consultation"
  | "respond-return-consultation"
  | "review-return-proposal"
  | "report-return-dispute"
  | "prepare-return-shipment"
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

function isAcceptedPhysicalReturnProposal(
  proposal: TradeReturnProposal | undefined,
): boolean {
  return (
    !!proposal &&
    proposal.id.trim() !== "" &&
    proposal.agreement === "agree" &&
    !proposal.rejectedAt &&
    proposal.returnRequirement === "required" &&
    proposal.refundAmount !== undefined &&
    Number.isInteger(proposal.refundAmount) &&
    proposal.refundAmount > 0
  );
}

function isRejectedReturnProposal(
  proposal: TradeReturnProposal | undefined,
): boolean {
  return (
    !!proposal &&
    proposal.id.trim() !== "" &&
    proposal.agreement === "disagree"
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

      case "agreed":
        return (
          isAcceptedPhysicalReturnProposal(
            trade.returnProposal,
          ) &&
          trade.returnShipmentStatus === "ready_for_dropoff"
        )
          ? "receive-return"
          : null;

      case "return_shipped":
      case "return_received":
      case "refund_processing":
        return isAcceptedPhysicalReturnProposal(
          trade.returnProposal,
        )
          ? "receive-return"
          : null;

      case "none":
      case "proposed":
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

    case "discussing":
      return isRejectedReturnProposal(
        trade.returnProposal,
      )
        ? "report-return-dispute"
        : null;

    case "proposed":
      return "review-return-proposal";

    case "agreed":
      return isAcceptedPhysicalReturnProposal(
        trade.returnProposal,
      )
        ? "prepare-return-shipment"
        : null;

    case "return_shipped":
    case "return_received":
    case "refund_processing":
    case "completed":
    case "disputed":
    case undefined:
      return null;
  }
}