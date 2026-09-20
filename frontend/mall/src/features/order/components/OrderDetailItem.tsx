// frontend/mall/src/features/order/components/OrderDetailItem.tsx

import Badge from "../../../components/ui/Badge";
import Button from "../../../components/ui/Button";
import InfoList, { InfoRow } from "../../../components/ui/InfoList";
import MediaIcon from "../../../components/ui/MediaIcon";
import { formatDateTime } from "../../../components/utils/date";
import type { OrderDetailItem as OrderDetailItemType } from "../../shared/types/orderDetailTypes";
import { formatAmount } from "../../wallet/utils/format";
import { getFallbackInitial, getProductTitle } from "../utils/orderItemDisplay";
import { getItemStatusLabel } from "../utils/orderStatus";
import OrderItemMeta from "./OrderItemMeta";

type OrderDetailItemProps = {
  orderId: string;
  item: OrderDetailItemType;
  index: number;
  cancellingItemIndex: number | null;
  returningItemIndex: number | null;
  tradeNavigatingIndex: number | null;
  onCancelItem: (itemIndex: number) => void | Promise<void>;
  onReturnItem: (itemIndex: number) => void;
  onOpenTrade: (orderId: string, itemIndex: number) => void | Promise<void>;
  onOpenBrand: (brandId?: string) => void;
};

type StatusBadgeVariant = "neutral" | "info" | "success" | "warning" | "danger";

function getItemStatusVariant(item: OrderDetailItemType): StatusBadgeVariant {
  if (item.isCancelled) return "danger";
  if (item.isReturnCompleted) return "neutral";
  if (item.isReturnRequested) return "warning";
  if (item.transferred) return "success";
  if (item.isDispatched) return "info";
  return "neutral";
}

export default function OrderDetailItem({
  orderId,
  item,
  index,
  cancellingItemIndex,
  returningItemIndex,
  tradeNavigatingIndex,
  onCancelItem,
  onReturnItem,
  onOpenTrade,
  onOpenBrand,
}: OrderDetailItemProps) {
  const productTitle = getProductTitle(item);
  const brandName = item.brandName || "ブランド未設定";
  const isResaleItem = item.itemType === "resale";
  const isCancelling = cancellingItemIndex === index;
  const isReturning = returningItemIndex === index;
  const isOpeningTrade = tradeNavigatingIndex === index;

  const cancelDisabled =
    item.isCancelled ||
    item.isDispatched ||
    item.transferred ||
    item.isReturnCompleted ||
    isCancelling ||
    returningItemIndex !== null;

  const showReturnButton =
    !isResaleItem &&
    item.isDispatched &&
    !item.transferred &&
    !item.isCancelled &&
    !item.isReturnRequested &&
    !item.isReturnCompleted;

  return (
    <li className="order-detail-page__item">
      <MediaIcon
        src={item.tokenIcon}
        alt={item.tokenName || productTitle}
        fallback={getFallbackInitial(item.tokenName || productTitle)}
        size="lg"
        shape="rounded"
      />

      <div className="order-detail-page__item-body">
        <div className="order-detail-page__item-heading">
          <div className="order-detail-page__item-title-area">
            <span className="order-detail-page__item-title">{productTitle}</span>
            {item.tokenName ? <span className="order-detail-page__item-token-name">{item.tokenName}</span> : null}
          </div>

          <span className="order-detail-page__item-price">{formatAmount(item.price)}</span>
        </div>

        <div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={!item.brandId}
            onClick={() => onOpenBrand(item.brandId)}
          >
            <MediaIcon
              src={item.brandIcon}
              alt={brandName}
              fallback={getFallbackInitial(brandName)}
              size="xs"
              shape="circle"
            />
            {brandName}
          </Button>
        </div>

        <OrderItemMeta item={item} />

        <InfoList>
          <InfoRow label="数量">{item.qty}点</InfoRow>
          <InfoRow label="小計">{formatAmount(item.price * item.qty)}</InfoRow>
          <InfoRow label="消費税率">{item.consumptionTaxRate}%</InfoRow>
          <InfoRow label="発送状況">
            <Badge variant={getItemStatusVariant(item)} size="sm">
              {getItemStatusLabel(item)}
            </Badge>
          </InfoRow>

          {item.returnRequestedAt ? (
            <InfoRow label="返品申請日時">{formatDateTime(item.returnRequestedAt)}</InfoRow>
          ) : null}

          {item.returnCompletedAt ? (
            <InfoRow label="返品完了日時">{formatDateTime(item.returnCompletedAt)}</InfoRow>
          ) : null}

          {item.transferredAt ? (
            <InfoRow label="受取日時">{formatDateTime(item.transferredAt)}</InfoRow>
          ) : null}
        </InfoList>

        {isResaleItem ? (
          <div className="page-actions order-detail-page__cancel-actions">
            <Button
              type="button"
              variant="primary"
              size="sm"
              disabled={tradeNavigatingIndex !== null}
              onClick={() => void onOpenTrade(orderId, index)}
            >
              {isOpeningTrade ? "移動中..." : "取引画面"}
            </Button>
          </div>
        ) : item.transferred && !item.isReturnRequested && !item.isReturnCompleted ? null : (
          <div className="page-actions order-detail-page__cancel-actions">
            {item.isReturnCompleted ? (
              <Button type="button" variant="secondary" size="sm" disabled>
                返品済み
              </Button>
            ) : item.isReturnRequested ? (
              <Button type="button" variant="secondary" size="sm" disabled>
                返品申請済み
              </Button>
            ) : showReturnButton ? (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={isReturning || cancellingItemIndex !== null}
                onClick={() => onReturnItem(index)}
              >
                {isReturning ? "返品申請中..." : "返品"}
              </Button>
            ) : (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={cancelDisabled}
                onClick={() => void onCancelItem(index)}
              >
                {item.isCancelled
                  ? "キャンセル済み"
                  : isCancelling
                    ? "キャンセル中..."
                    : "商品をキャンセル"}
              </Button>
            )}
          </div>
        )}
      </div>
    </li>
  );
}