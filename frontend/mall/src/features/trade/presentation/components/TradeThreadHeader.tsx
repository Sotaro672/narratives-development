// frontend/mall/src/features/trade/presentation/components/TradeThreadHeader.tsx

import { useNavigate } from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import InfoList, { type InfoListRow } from "../../../../components/ui/InfoList";
import TextLink from "../../../../components/ui/textLink";
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
  const navigate = useNavigate();

  if (!trade) {
    return null;
  }

  const title = getTradeTitle(trade.productName);
  const resaleId = trade.resale?.id?.trim() ?? "";
  const orderId = trade.orderId.trim();

  const resaleDetailPath =
    resaleId === ""
      ? ""
      : trade.viewerSide === "seller"
        ? `/resales/${encodeURIComponent(resaleId)}`
        : `/market/${encodeURIComponent(resaleId)}`;

  const orderDetailPath =
    trade.viewerSide === "buyer" && orderId
      ? `/orders/${encodeURIComponent(orderId)}`
      : "";

  const transactionMetaItems: InfoListRow[] = [];

  if (trade.transferredAt) {
    transactionMetaItems.push({
      label: "受取日時",
      value: formatDateTime(trade.transferredAt),
    });
  }

  const handleOpenResaleDetail = (): void => {
    if (!resaleDetailPath) {
      return;
    }

    navigate(resaleDetailPath, {
      state:
        trade.viewerSide === "buyer"
          ? { reviewContext: true }
          : undefined,
    });
  };

  const handleOpenOrderDetail = (): void => {
    if (!orderDetailPath) {
      return;
    }

    navigate(orderDetailPath);
  };

  return (
    <>
      <div className="trade-chat-detail__heading">
        <h2 className="chat-detail-page__subject">{title}</h2>

        <Badge variant="info">
          {getTradeStatusLabel(trade)}
        </Badge>
      </div>

      {transactionMetaItems.length > 0 ? (
        <section className="trade-chat-detail__transaction">
          <InfoList rows={transactionMetaItems} />
        </section>
      ) : null}

      {resaleDetailPath || orderDetailPath ? (
        <div className="trade-chat-detail__detail-links">
          {resaleDetailPath ? (
            <TextLink
              className="trade-chat-detail__detail-link"
              onClick={handleOpenResaleDetail}
            >
              出品詳細を見る
            </TextLink>
          ) : null}

          {orderDetailPath ? (
            <TextLink
              className="trade-chat-detail__detail-link"
              onClick={handleOpenOrderDetail}
            >
              注文詳細を見る
            </TextLink>
          ) : null}
        </div>
      ) : null}
    </>
  );
}