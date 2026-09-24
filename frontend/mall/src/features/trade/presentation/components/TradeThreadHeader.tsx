// frontend/mall/src/features/trade/presentation/components/TradeThreadHeader.tsx

import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import Copy from "../../../../components/ui/Copy";
import InfoList, { type InfoListRow } from "../../../../components/ui/InfoList";
import { formatDateTime } from "../../../../components/utils/date";
import type { TradeDetail } from "../../../shared/types/trade";
import { getTradeTitle } from "../util/tradeChatDetail";
import { getTradeStatusLabel } from "../util/tradeStatus";

type TradeThreadHeaderProps = {
  trade?: TradeDetail | null;
};

export default function TradeThreadHeader({
  trade,
}: TradeThreadHeaderProps) {
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

  if (!trade) {
    return null;
  }

  const title = getTradeTitle(trade.productName);
  const resaleId = trade.resale?.id?.trim() ?? "";

  const resaleDetailPath =
    resaleId === ""
      ? ""
      : trade.viewerSide === "seller"
        ? `/resales/${encodeURIComponent(resaleId)}`
        : `/market/${encodeURIComponent(resaleId)}`;

  const handleCopyOrderId = async (): Promise<void> => {
    try {
      await navigator.clipboard.writeText(trade.orderId);
      setOrderIdCopied(true);
    } catch {
      // Clipboard API が利用できない環境では何もしない。
    }
  };

  const transactionMetaItems: InfoListRow[] = [
    {
      label: "注文ID",
      value: (
        <span className="trade-chat-detail__order-id">
          <span>{trade.orderId}</span>
          <Copy
            onClick={handleCopyOrderId}
            ariaLabel={orderIdCopied ? "コピーしました" : "注文IDをコピー"}
            title={orderIdCopied ? "コピーしました" : "注文IDをコピー"}
          />
        </span>
      ),
    },
  ];

  if (trade.transferredAt) {
    transactionMetaItems.push({
      label: "受取日時",
      value: formatDateTime(trade.transferredAt),
    });
  }

  return (
    <>
      <div className="trade-chat-detail__heading">
        <h2 className="chat-detail-page__subject">
          {title}
        </h2>

        <Badge variant="info">
          {getTradeStatusLabel(trade)}
        </Badge>
      </div>

      <section className="trade-chat-detail__transaction">
        <InfoList rows={transactionMetaItems} />
      </section>

      {resaleDetailPath ? (
        <Link
          to={resaleDetailPath}
          state={
            trade.viewerSide === "buyer"
              ? { reviewContext: true }
              : undefined
          }
          className="trade-chat-detail__resale-link"
        >
          出品詳細を見る
        </Link>
      ) : null}
    </>
  );
}