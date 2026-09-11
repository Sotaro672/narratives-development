// frontend/mall/src/features/trade/presentation/components/TradeThreadHeader.tsx

import { useEffect, useState } from "react";
import { Copy } from "lucide-react";

import { formatDateTime } from "../../../../components/utils/date";
import ChatImageGrid from "../../../shared/presentation/components/ChatImageGrid";
import ChatMessageHeader from "../../../shared/presentation/components/ChatMessageHeader";
import ChatMetaSection, { type ChatMetaItem } from "../../../shared/presentation/components/ChatMetaSection";
import ChatThreadCard from "../../../shared/presentation/components/ChatThreadCard";
import { createProductModelDisplay } from "../../../shared/presentation/utils/productModelDisplay";
import type { TradeDetail } from "../../../shared/types/trade";

import {
  getTradeTitle,
} from "../util/tradeChatDetail";
import { getTradeStatusLabel } from "../util/tradeStatus";

type TradeThreadHeaderProps = {
  trade: TradeDetail;
};

export default function TradeThreadHeader({
  trade,
}: TradeThreadHeaderProps) {
  const counterpartLabel =
    trade.viewerSide === "buyer"
      ? "出品者"
      : "購入者";

  const counterpartAvatarName =
    trade.viewerSide === "buyer"
      ? trade.sellerAvatarName
      : trade.buyerAvatarName;

  const counterpartAvatarIcon =
    trade.viewerSide === "buyer"
      ? trade.sellerAvatarIcon
      : trade.buyerAvatarIcon;

  const displayName =
    counterpartAvatarName || counterpartLabel;

  const title = getTradeTitle(trade.productName);
  const model = createProductModelDisplay(trade.resale);
  const [orderIdCopied, setOrderIdCopied] = useState(false);

  useEffect(() => {
    if (!orderIdCopied) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      setOrderIdCopied(false);
    }, 2000);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [orderIdCopied]);

  const handleCopyOrderId = async (): Promise<void> => {
    try {
      await navigator.clipboard.writeText(trade.orderId);
      setOrderIdCopied(true);
    } catch {
      // Clipboard API が利用できない環境では何もしない。
    }
  };

  const transactionMetaItems: ChatMetaItem[] = [
    {
      label: "注文ID",
      value: (
        <span className="trade-chat-detail__order-id">
          <span>{trade.orderId}</span>

          <button
            type="button"
            className="trade-chat-detail__copy-button"
            onClick={() => {
              void handleCopyOrderId();
            }}
            aria-label={
              orderIdCopied
                ? "コピーしました"
                : "注文IDをコピー"
            }
            title={
              orderIdCopied
                ? "コピーしました"
                : "注文IDをコピー"
            }
          >
            {orderIdCopied ? (
              <span>コピーしました</span>
            ) : (
              <Copy
                size={15}
                strokeWidth={2}
                aria-hidden="true"
              />
            )}
          </button>
        </span>
      ),
    },
  ];

  if (trade.returnRequestedAt) {
    transactionMetaItems.push({
      label: "返品申請日時",
      value: formatDateTime(trade.returnRequestedAt),
    });
  }

  if (trade.returnCompletedAt) {
    transactionMetaItems.push({
      label: "返品完了日時",
      value: formatDateTime(trade.returnCompletedAt),
    });
  }

  if (trade.transferredAt) {
    transactionMetaItems.push({
      label: "受取日時",
      value: formatDateTime(trade.transferredAt),
    });
  }

  const productMetaItems: ChatMetaItem[] = [
    {
      label: "商品の状態",
      value: trade.resale.condition,
    },
  ];

  if (model.modelNumber) {
    productMetaItems.push({
      label: "モデル番号",
      value: model.modelNumber,
    });
  }

  if (
    model.kindLabel &&
    model.kindLabel !== "アパレル"
  ) {
    productMetaItems.push({
      label: "種別",
      value: model.kindLabel,
    });
  }

  if (model.size) {
    productMetaItems.push({
      label: "サイズ",
      value: model.size,
    });
  }

  if (model.colorLabel || model.colorCssValue) {
    productMetaItems.push({
      label: "カラー",
      value:
        model.colorLabel ||
        model.colorCssValue,
    });
  }

  if (model.measurementsLabel !== "-") {
    productMetaItems.push({
      label: "採寸",
      value: model.measurementsLabel,
    });
  }

  if (model.volumeLabel !== "-") {
    productMetaItems.push({
      label: "容量",
      value: model.volumeLabel,
    });
  }

  return (
    <ChatThreadCard variant="trade">
      <ChatMessageHeader
        name={displayName}
        icon={counterpartAvatarIcon}
        createdAt={trade.createdAt}
        action={
          <span className="chat-detail-page__status">
            {getTradeStatusLabel(trade)}
          </span>
        }
      />

      <h2 className="chat-detail-page__subject">
        {title}
      </h2>

      <ChatMetaSection
        title="取引情報"
        items={transactionMetaItems}
      />

      <ChatMetaSection
        title="商品情報"
        items={productMetaItems}
      />

      {trade.resale.description ? (
        <details className="chat-detail-page__description-accordion">
          <summary className="chat-detail-page__description-summary">
            商品説明
          </summary>

          <div className="chat-detail-page__description-body">
            <p className="chat-detail-page__content">
              {trade.resale.description}
            </p>
          </div>
        </details>
      ) : null}

      <ChatImageGrid
        images={trade.resale.images.map((image) => ({
          key: image.id,
          url: image.url,
          alt: "商品状態",
        }))}
      />
    </ChatThreadCard>
  );
}