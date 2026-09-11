// frontend/mall/src/features/trade/presentation/components/TradeMessageCard.tsx

import ChatMessageBubble from "../../../shared/presentation/components/ChatMessageBubble";
import type {
  TradeDetail,
  TradeMessage,
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

function getSenderDisplay(
  message: TradeMessage,
  trade: TradeDetail,
): TradeSenderDisplay {
  switch (message.senderSide) {
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

export default function TradeMessageCard({
  message,
  trade,
  onReport,
}: TradeMessageCardProps) {
  const isSystem = message.senderSide === "system";
  const isMine =
    !isSystem &&
    message.senderSide === trade.viewerSide;

  const sender = getSenderDisplay(message, trade);

  const canReport =
    !isSystem &&
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
    />
  );
}