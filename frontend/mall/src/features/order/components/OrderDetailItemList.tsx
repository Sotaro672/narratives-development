// frontend/mall/src/features/order/components/OrderDetailItemList.tsx

import MediaIcon from "../../../components/ui/MediaIcon";
import SectionHeader from "../../../components/ui/SectionHeader";
import { formatDateTime } from "../../../components/utils/date";
import type {
  OrderDetail,
  OrderDetailItem,
} from "../../shared/types/orderDetailTypes";
import { formatAmount } from "../../wallet/utils/format";

type OrderDetailItemListProps = {
  order: OrderDetail;
  cancellingItemIndex: number | null;
  returningItemIndex: number | null;
  tradeNavigatingIndex: number | null;
  onCancelItem: (itemIndex: number) => void | Promise<void>;
  onReturnItem: (itemIndex: number) => void;
  onOpenTrade: (orderId: string, itemIndex: number) => void | Promise<void>;
  onOpenBrand: (brandId?: string) => void;
};

function getItemStatusLabel(item: OrderDetailItem): string {
  if (item.isCancelled) return "キャンセル済み";
  if (item.isReturnCompleted) return "返品済み";
  if (item.isReturnRequested) return "返品申請済み";
  if (item.transferred) return "受け取り済み";
  if (item.isDispatched) return "発送済み";
  return "発送前";
}

function getProductTitle(item: OrderDetailItem): string {
  return item.productName || item.tokenName || "商品";
}

function getFallbackInitial(value?: string): string {
  const trimmed = value?.trim() || "";
  if (!trimmed) return "?";
  return trimmed.slice(0, 1).toUpperCase();
}

function getModelMetaItems(
  item: OrderDetailItem,
): Array<{ label: string; value: string }> {
  const metaItems: Array<{ label: string; value: string }> = [];

  if (item.modelNumber) {
    metaItems.push({
      label: "モデル番号",
      value: item.modelNumber,
    });
  }

  if (item.size) {
    metaItems.push({
      label: "サイズ",
      value: item.size,
    });
  }

  if (item.color?.name) {
    metaItems.push({
      label: "カラー",
      value: item.color.name,
    });
  }

  if (item.volumeValue !== undefined && item.volumeValue !== null) {
    metaItems.push({
      label: "容量",
      value: `${item.volumeValue}${item.volumeUnit || ""}`,
    });
  }

  return metaItems;
}

function getMeasurementLabel(key: string): string {
  switch (key) {
    case "length":
      return "着丈";
    case "shoulder":
      return "肩幅";
    case "chest":
      return "身幅";
    case "sleeve":
      return "袖丈";
    case "waist":
      return "ウエスト";
    case "rise":
      return "股上";
    case "inseam":
      return "股下";
    case "hem":
      return "裾幅";
    default:
      return key;
  }
}

function renderModelMeta(item: OrderDetailItem) {
  const metaItems = getModelMetaItems(item);
  const measurements =
    item.measurements && typeof item.measurements === "object"
      ? Object.entries(item.measurements).filter(([, value]) =>
          Number.isFinite(value),
        )
      : [];

  if (metaItems.length === 0 && measurements.length === 0) {
    return null;
  }

  return (
    <div className="order-detail-page__model-meta">
      <dl className="order-detail-page__item-meta">
        {metaItems.map((meta) => (
          <div key={meta.label} className="order-detail-page__item-meta-row">
            <dt>{meta.label}</dt>
            <dd>{meta.value}</dd>
          </div>
        ))}

        {measurements.map(([key, value]) => (
          <div key={key} className="order-detail-page__item-meta-row">
            <dt>{getMeasurementLabel(key)}</dt>
            <dd>{value} mm</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

export default function OrderDetailItemList({
  order,
  cancellingItemIndex,
  returningItemIndex,
  tradeNavigatingIndex,
  onCancelItem,
  onReturnItem,
  onOpenTrade,
  onOpenBrand,
}: OrderDetailItemListProps) {
  return (
    <div className="page-card">
      <SectionHeader title="商品" titleAs="h2" />

      <ul className="order-detail-page__items">
        {order.items.map((item, index) => {
          const productTitle = getProductTitle(item);
          const brandName = item.brandName || "ブランド未設定";
          const itemKey = `${order.id}-${item.inventoryId}-${item.modelId}-${index}`;
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
            <li key={itemKey} className="order-detail-page__item">
              <div className="order-detail-page__item-image">
                {item.tokenIcon ? (
                  <img
                    src={item.tokenIcon}
                    alt={item.tokenName || productTitle}
                    loading="lazy"
                  />
                ) : (
                  <span className="order-detail-page__item-image-fallback">
                    {getFallbackInitial(item.tokenName || productTitle)}
                  </span>
                )}
              </div>

              <div className="order-detail-page__item-body">
                <div className="order-detail-page__item-heading">
                  <div className="order-detail-page__item-title-area">
                    <span className="order-detail-page__item-title">
                      {productTitle}
                    </span>

                    {item.tokenName ? (
                      <span className="order-detail-page__item-token-name">
                        {item.tokenName}
                      </span>
                    ) : null}
                  </div>

                  <span className="order-detail-page__item-price">
                    {formatAmount(item.price)}
                  </span>
                </div>

                <button
                  type="button"
                  className="order-detail-page__brand"
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
                  <span>{brandName}</span>
                </button>

                {renderModelMeta(item)}

                <dl className="order-detail-page__item-meta">
                  <div className="order-detail-page__item-meta-row">
                    <dt>数量</dt>
                    <dd>{item.qty}点</dd>
                  </div>

                  <div className="order-detail-page__item-meta-row">
                    <dt>小計</dt>
                    <dd>{formatAmount(item.price * item.qty)}</dd>
                  </div>

                  <div className="order-detail-page__item-meta-row">
                    <dt>消費税率</dt>
                    <dd>{item.consumptionTaxRate}%</dd>
                  </div>

                  <div className="order-detail-page__item-meta-row">
                    <dt>発送状況</dt>
                    <dd>{getItemStatusLabel(item)}</dd>
                  </div>

                  {item.returnRequestedAt ? (
                    <div className="order-detail-page__item-meta-row">
                      <dt>返品申請日時</dt>
                      <dd>{formatDateTime(item.returnRequestedAt)}</dd>
                    </div>
                  ) : null}

                  {item.returnCompletedAt ? (
                    <div className="order-detail-page__item-meta-row">
                      <dt>返品完了日時</dt>
                      <dd>{formatDateTime(item.returnCompletedAt)}</dd>
                    </div>
                  ) : null}

                  {item.transferredAt ? (
                    <div className="order-detail-page__item-meta-row">
                      <dt>受取日時</dt>
                      <dd>{formatDateTime(item.transferredAt)}</dd>
                    </div>
                  ) : null}
                </dl>

                {isResaleItem ? (
                  <div className="page-actions order-detail-page__cancel-actions">
                    <button
                      type="button"
                      className="order-detail-page__trade-button"
                      disabled={tradeNavigatingIndex !== null}
                      onClick={() => void onOpenTrade(order.id, index)}
                    >
                      {isOpeningTrade ? "移動中..." : "取引画面"}
                    </button>
                  </div>
                ) : item.transferred &&
                  !item.isReturnRequested &&
                  !item.isReturnCompleted ? null : (
                  <div className="page-actions order-detail-page__cancel-actions">
                    {item.isReturnCompleted ? (
                      <button
                        type="button"
                        className="order-detail-page__cancel-button order-detail-page__return-button"
                        disabled
                      >
                        返品済み
                      </button>
                    ) : item.isReturnRequested ? (
                      <button
                        type="button"
                        className="order-detail-page__cancel-button order-detail-page__return-button"
                        disabled
                      >
                        返品申請済み
                      </button>
                    ) : showReturnButton ? (
                      <button
                        type="button"
                        className="order-detail-page__cancel-button order-detail-page__return-button"
                        disabled={isReturning || cancellingItemIndex !== null}
                        onClick={() => onReturnItem(index)}
                      >
                        {isReturning ? "返品申請中..." : "返品"}
                      </button>
                    ) : (
                      <button
                        type="button"
                        className="order-detail-page__cancel-button"
                        disabled={cancelDisabled}
                        onClick={() => void onCancelItem(index)}
                      >
                        {item.isCancelled
                          ? "キャンセル済み"
                          : isCancelling
                            ? "キャンセル中..."
                            : "商品をキャンセル"}
                      </button>
                    )}
                  </div>
                )}
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}