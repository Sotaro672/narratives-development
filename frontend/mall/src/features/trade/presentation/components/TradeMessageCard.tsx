// frontend/mall/src/features/trade/presentation/components/TradeMessageCard.tsx

import { useState } from "react";

import Preview from "../../../../components/ui/Preview";
import TextLink from "../../../../components/ui/textLink";
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
  onOpenDispatchQr?: () => void;
  onOpenReturnShipmentQr?: () => void;
};

type TradeSenderDisplay = {
  name: string;
  icon: string;
};

function isDispatchSystemMessage(message: TradeMessage): boolean {
  return (
    message.senderSide === "system" &&
    message.senderType === "system" &&
    message.id.trim() === "dispatch"
  );
}

function isReturnShipmentReadySystemMessage(
  message: TradeMessage,
): boolean {
  return (
    message.senderSide === "system" &&
    message.senderType === "system" &&
    message.id.trim().startsWith("return-shipment-ready-")
  );
}

function getSystemMessageActorSide(
  message: TradeMessage,
): "buyer" | "seller" | null {
  if (
    message.senderSide !== "system" ||
    message.senderType !== "system"
  ) {
    return null;
  }

  const messageId = message.id.trim();

  if (messageId === "dispatch") {
    return "seller";
  }

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

  return getSystemMessageActorSide(message) ?? "system";
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
  onOpenDispatchQr,
  onOpenReturnShipmentQr,
}: TradeMessageCardProps) {
  const [previewImageIndex, setPreviewImageIndex] = useState<number | null>(null);

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

  const dispatchSystemMessage = isDispatchSystemMessage(message);
  const returnShipmentReadySystemMessage =
    isReturnShipmentReadySystemMessage(message);

  const canOpenDispatchQr =
    isMine &&
    dispatchSystemMessage &&
    Boolean(onOpenDispatchQr);

  const canOpenReturnShipmentQr =
    isMine &&
    returnShipmentReadySystemMessage &&
    Boolean(onOpenReturnShipmentQr);

  const images = message.images ?? [];

  const previewImage =
    previewImageIndex !== null
      ? images[previewImageIndex] ?? null
      : null;

  const canReport =
    message.senderType === "avatar" &&
    !isMine &&
    Boolean(message.id.trim());

  const hasAfterContent =
    images.length > 0 ||
    Boolean(returnConsultationDetail) ||
    canOpenDispatchQr ||
    canOpenReturnShipmentQr;

  const handlePreviewPrevious = (): void => {
    if (
      previewImageIndex === null ||
      images.length <= 1
    ) {
      return;
    }

    setPreviewImageIndex(
      previewImageIndex === 0
        ? images.length - 1
        : previewImageIndex - 1,
    );
  };

  const handlePreviewNext = (): void => {
    if (
      previewImageIndex === null ||
      images.length <= 1
    ) {
      return;
    }

    setPreviewImageIndex(
      previewImageIndex === images.length - 1
        ? 0
        : previewImageIndex + 1,
    );
  };

  return (
    <>
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
          hasAfterContent ? (
            <>
              {images.length > 0 ? (
                <div
                  className="chat-detail-page__images"
                  aria-label="添付画像"
                >
                  {images.map((image, index) => (
                    <a
                      key={`${image.objectPath}-${index}`}
                      href={image.fileUrl}
                      className="chat-detail-page__image-link"
                      aria-label={`${image.fileName || `添付画像${index + 1}`}を表示`}
                      onClick={(event) => {
                        event.preventDefault();
                        setPreviewImageIndex(index);
                      }}
                    >
                      <img
                        src={image.fileUrl}
                        alt={image.fileName || `添付画像${index + 1}`}
                        className="chat-detail-page__image"
                        loading="lazy"
                        decoding="async"
                      />
                    </a>
                  ))}
                </div>
              ) : null}

              {returnConsultationDetail ? (
                <p className="chat-detail-page__content">
                  {returnConsultationDetail}
                </p>
              ) : null}

              {canOpenDispatchQr && onOpenDispatchQr ? (
                <TextLink
                  className="trade-chat-detail__dispatch-qr-link"
                  onClick={onOpenDispatchQr}
                >
                  PUDO QRを表示
                </TextLink>
              ) : null}

              {canOpenReturnShipmentQr && onOpenReturnShipmentQr ? (
                <TextLink
                  className="trade-chat-detail__dispatch-qr-link"
                  onClick={onOpenReturnShipmentQr}
                >
                  PUDO QRを表示
                </TextLink>
              ) : null}
            </>
          ) : undefined
        }
      />

      <Preview
        open={previewImage !== null}
        src={previewImage?.fileUrl}
        alt={previewImage?.fileName || "取引メッセージの添付画像"}
        type={previewImage?.mimeType || "image"}
        onClose={() => {
          setPreviewImageIndex(null);
        }}
        onPrev={
          images.length > 1
            ? handlePreviewPrevious
            : undefined
        }
        onNext={
          images.length > 1
            ? handlePreviewNext
            : undefined
        }
      />
    </>
  );
}