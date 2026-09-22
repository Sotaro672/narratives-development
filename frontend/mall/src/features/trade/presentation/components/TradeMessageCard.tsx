// frontend/mall/src/features/trade/presentation/components/TradeMessageCard.tsx

import ChatMessageBubble from "../../../shared/presentation/components/ChatMessageBubble";
import type {
  TradeDetail,
  TradeMessage,
  TradeMessageSenderSide,
} from "../../../shared/types/trade";

type TradeMessageCardProps = {
  message: TradeMessage;
  trade: TradeDetail;
  onReport: (message: TradeMessage) => void;
};

type TradeSenderDisplay = {
  name: string;
  icon: string;
};

function getReturnSystemMessageActorSide(
  message: TradeMessage,
): "buyer" | "seller" | null {
  if (
    message.senderSide !== "system" ||
    message.senderType !== "system"
  ) {
    return null;
  }

  const messageId = message.id.trim();

  if (messageId === "return-consultation") {
    return "buyer";
  }

  if (
    messageId.startsWith("return-proposal-accepted-") ||
    messageId.startsWith("return-proposal-rejected-") ||
    messageId.startsWith("return-shipment-ready-")
  ) {
    return "buyer";
  }

  if (
    messageId.startsWith("return-proposal-") ||
    messageId.startsWith("return-completed-")
  ) {
    return "seller";
  }

  return null;
}

function getDisplaySenderSide(
  message: TradeMessage,
): TradeMessageSenderSide {
  if (message.senderSide !== "system") {
    return message.senderSide;
  }

  return getReturnSystemMessageActorSide(message) ?? "system";
}

function getSenderDisplay(
  senderSide: TradeMessageSenderSide,
  trade: TradeDetail,
): TradeSenderDisplay {
  switch (senderSide) {
    case "buyer":
      return {
        name: trade.buyerAvatarName || "購入者",
        icon: trade.buyerAvatarIcon || "",
      };

    case "seller":
      return {
        name: trade.sellerAvatarName || "出品者",
        icon: trade.sellerAvatarIcon || "",
      };

    case "system":
      return {
        name: "AMOL",
        icon: "",
      };
  }
}

function getReturnConsultationDetail(
  message: TradeMessage,
  trade: TradeDetail,
): string {
  if (
    message.senderSide !== "system" ||
    message.senderType !== "system" ||
    message.id.trim() !== "return-consultation"
  ) {
    return "";
  }

  return trade.returnConsultation?.detail?.trim() ?? "";
}

export default function TradeMessageCard({
  message,
  trade,
  onReport,
}: TradeMessageCardProps) {
  const displaySenderSide = getDisplaySenderSide(message);
  const isSystem = displaySenderSide === "system";
  const isMine =
    !isSystem &&
    displaySenderSide === trade.viewerSide;

  const sender = getSenderDisplay(displaySenderSide, trade);
  const returnConsultationDetail = getReturnConsultationDetail(
    message,
    trade,
  );

  const canReport =
    message.senderType === "avatar" &&
    !isMine &&
    Boolean(message.id.trim());

  return (
    <ChatMessageBubble
      senderName={sender.name}
      senderIcon={sender.icon}
      createdAt={message.createdAt}
      content={message.content}
      isMine={isMine}
      isSystem={isSystem}
      action={
        canReport ? (
          <button
            type="button"
            className="trade-chat-detail__copy-button"
            aria-label={`${sender.name}の取引コメントを通報`}
            onClick={() => onReport(message)}
          >
            通報
          </button>
        ) : undefined
      }
      afterContent={
        returnConsultationDetail ? (
          <p className="chat-detail-page__content">
            {returnConsultationDetail}
          </p>
        ) : undefined
      }
    />
  );
}