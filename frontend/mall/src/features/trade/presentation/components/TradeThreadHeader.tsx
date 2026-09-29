// frontend/mall/src/features/trade/presentation/components/TradeThreadHeader.tsx

import { useNavigate } from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import TextLink from "../../../../components/ui/textLink";
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
    <div className="trade-chat-detail__heading">
      <h2 className="chat-detail-page__subject">
        {title}
      </h2>

      <div className="chat-detail-page__header-meta">
        <Badge variant="info">
          {getTradeStatusLabel(trade)}
        </Badge>

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
      </div>
    </div>
  );
}