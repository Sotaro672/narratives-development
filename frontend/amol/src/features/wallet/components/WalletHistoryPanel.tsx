// frontend/amol/src/features/wallet/components/WalletHistoryPanel.tsx
import MediaIcon from "../../../components/ui/MediaIcon";
import { formatDateTime } from "../../../components/utils/date";
import type {
  WalletOrder,
  WalletOrderItemSnapshot,
  WalletOrderMeasurements,
} from "../types/orderTypes";
import { formatAmount } from "../utils/format";

type WalletHistoryPanelProps = {
  loading: boolean;
  error: string;
  hasItems: boolean;
  orderHistory: WalletOrder[];
};

function getOrderTotal(order: WalletOrder): number {
  return order.items.reduce((sum, item) => {
    return sum + item.price * item.qty;
  }, 0);
}

function getOrderItemCount(order: WalletOrder): number {
  return order.items.reduce((sum, item) => {
    return sum + item.qty;
  }, 0);
}

function getOrderStatusLabel(order: WalletOrder): string {
  if (order.items.length === 0) {
    return "商品なし";
  }

  const allCanceled = order.items.every((item) => item.isCanceled);
  if (allCanceled) {
    return "キャンセル済み";
  }

  const allTransferred = order.items.every((item) => item.transferred);
  if (allTransferred) {
    return order.paid ? "受け取り済み" : "未決済";
  }

  const partiallyTransferred = order.items.some((item) => item.transferred);
  if (partiallyTransferred) {
    return order.paid ? "一部受け取り済み" : "未決済";
  }

  const allDispatched = order.items.every((item) => item.isDispatched);
  if (allDispatched) {
    return "発送済み";
  }

  const partiallyDispatched = order.items.some((item) => item.isDispatched);
  if (partiallyDispatched) {
    return "一部発送済み";
  }

  return order.paid ? "決済済み" : "未決済";
}

function getOrderSummary(order: WalletOrder): string {
  const itemCount = getOrderItemCount(order);
  const total = getOrderTotal(order);

  return `${itemCount}点 / ${formatAmount(total)}`;
}

function getProductTitle(item: WalletOrderItemSnapshot): string {
  return item.productName || item.tokenName || item.modelNumber || "商品";
}

function getProductSubtitle(item: WalletOrderItemSnapshot): string {
  return item.tokenName || "";
}

function getFallbackInitial(value?: string): string {
  const trimmed = value?.trim() || "";

  if (!trimmed) {
    return "?";
  }

  return trimmed.slice(0, 1).toUpperCase();
}

function renderImage(src: string | undefined, alt: string, fallbackText: string) {
  if (!src) {
    return (
      <span className="wallet-page-history__image-fallback">
        {fallbackText}
      </span>
    );
  }

  return (
    <img
      className="wallet-page-history__image"
      src={src}
      alt={alt}
      loading="lazy"
    />
  );
}

function renderMeasurements(measurements?: WalletOrderMeasurements) {
  if (!measurements || Object.keys(measurements).length === 0) {
    return null;
  }

  return (
    <dl className="wallet-page-history__measurements">
      {Object.entries(measurements).map(([label, value]) => (
        <div key={label} className="wallet-page-history__measurement">
          <dt className="wallet-page-history__measurement-label">{label}</dt>
          <dd className="wallet-page-history__measurement-value">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function renderItemMeta(item: WalletOrderItemSnapshot) {
  const metaItems = [
    item.modelNumber ? `品番: ${item.modelNumber}` : "",
    item.size ? `サイズ: ${item.size}` : "",
    item.color?.name ? `色: ${item.color.name}` : "",
    item.transferredAt ? `受取日時: ${formatDateTime(item.transferredAt)}` : "",
  ].filter(Boolean);

  if (metaItems.length === 0) {
    return null;
  }

  return (
    <div className="wallet-page-history__meta-list">
      {metaItems.map((meta) => (
        <span key={meta} className="wallet-page-history__meta-item">
          {meta}
        </span>
      ))}
    </div>
  );
}

export default function WalletHistoryPanel({
  loading,
  error,
  hasItems,
  orderHistory,
}: WalletHistoryPanelProps) {
  if (loading) {
    return <p className="wallet-page__message">読み込み中です...</p>;
  }

  if (error) {
    return (
      <div role="alert" className="wallet-page__message">
        <p>{error}</p>
      </div>
    );
  }

  if (!hasItems || orderHistory.length === 0) {
    return <p className="wallet-page__message">取引履歴はまだありません。</p>;
  }

  return (
    <div className="wallet-page-history">
      {orderHistory.map((order) => (
        <article key={order.id} className="wallet-page-history__item">
          <div className="wallet-page-history__main">
            <div className="wallet-page-history__header">
              <p className="wallet-page-history__date">
                {formatDateTime(order.createdAt)}
              </p>

              <span className="wallet-page-history__status">
                {getOrderStatusLabel(order)}
              </span>
            </div>

            <p className="wallet-page-history__title">
              注文ID: {order.id || "-"}
            </p>

            <p className="wallet-page-history__summary">
              {getOrderSummary(order)}
            </p>
          </div>

          {order.items.length > 0 ? (
            <ul className="wallet-page-history__items">
              {order.items.map((item, index) => {
                const productTitle = getProductTitle(item);
                const productSubtitle = getProductSubtitle(item);
                const brandName = item.brandName || "ブランド未設定";
                const itemKey = `${order.id}-${item.inventoryId}-${item.modelId}-${index}`;

                return (
                  <li key={itemKey} className="wallet-page-history__product">
                    <div className="wallet-page-history__product-visual">
                      {renderImage(
                        item.tokenIcon,
                        item.tokenName || productTitle,
                        getFallbackInitial(item.tokenName || productTitle)
                      )}
                    </div>

                    <div className="wallet-page-history__product-body">
                      <div className="wallet-page-history__product-heading">
                        <div className="wallet-page-history__product-title-area">
                          <span className="wallet-page-history__product-name">
                            {productTitle}
                          </span>

                          {productSubtitle ? (
                            <span className="wallet-page-history__product-subtitle">
                              {productSubtitle}
                            </span>
                          ) : null}
                        </div>

                        <span className="wallet-page-history__product-meta">
                          {item.qty}点 / {formatAmount(item.price)}
                        </span>
                      </div>

                      <div className="wallet-page-history__brand">
                        <MediaIcon
                          src={item.brandIcon}
                          alt={brandName}
                          fallback={getFallbackInitial(brandName)}
                          size="xs"
                          shape="circle"
                          className="wallet-page-history__brand-icon"
                        />

                        <span className="wallet-page-history__brand-name">
                          {brandName}
                        </span>
                      </div>

                      {renderItemMeta(item)}
                      {renderMeasurements(item.measurements)}
                    </div>
                  </li>
                );
              })}
            </ul>
          ) : null}
        </article>
      ))}
    </div>
  );
}