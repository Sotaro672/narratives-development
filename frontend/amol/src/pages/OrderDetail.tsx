// frontend/amol/src/pages/OrderDetail.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import MediaIcon from "../components/ui/MediaIcon";
import SectionHeader from "../components/ui/SectionHeader";
import { formatDateTime } from "../components/utils/date";

import { useOrderDetail } from "../features/order/hooks/useOrderDetail";
import type {
  OrderDetail as OrderDetailType,
  OrderDetailItem,
} from "../features/shared/types/orderDetailTypes";
import { formatAmount } from "../features/wallet/utils/format";

import "../styles/page-layout.css";
import "../styles/order-detail-page.css";

function getOrderSubtotal(order: OrderDetailType): number {
  return order.items.reduce((sum, item) => {
    return sum + item.price * item.qty;
  }, 0);
}

function getOrderShippingAmount(order: OrderDetailType): number {
  const amount = order.shippingQuoteSnapshot?.amount;

  if (typeof amount !== "number" || !Number.isFinite(amount)) {
    return 0;
  }

  return amount;
}

function getOrderTotal(order: OrderDetailType): number {
  return getOrderSubtotal(order) + getOrderShippingAmount(order);
}

function getOrderStatusLabel(order: OrderDetailType): string {
  if (order.items.length === 0) {
    return "商品なし";
  }

  const activeItems = order.items.filter((item) => !item.isCancelled);

  if (activeItems.length === 0) {
    return "キャンセル済み";
  }

  const hasCancelledItem = activeItems.length !== order.items.length;

  if (hasCancelledItem) {
    return "一部キャンセル済み";
  }

  const allReturnRequested = activeItems.every(
    (item) => item.isReturnRequested,
  );

  if (allReturnRequested) {
    return "返品申請済み";
  }

  const partiallyReturnRequested = activeItems.some(
    (item) => item.isReturnRequested,
  );

  if (partiallyReturnRequested) {
    return "一部返品申請済み";
  }

  const allTransferred = activeItems.every((item) => item.transferred);

  if (allTransferred) {
    return order.paid ? "受け取り済み" : "未決済";
  }

  const partiallyTransferred = activeItems.some((item) => item.transferred);

  if (partiallyTransferred) {
    return order.paid ? "一部受け取り済み" : "未決済";
  }

  const allDispatched = activeItems.every((item) => item.isDispatched);

  if (allDispatched) {
    return "発送済み";
  }

  const partiallyDispatched = activeItems.some((item) => item.isDispatched);

  if (partiallyDispatched) {
    return "一部発送済み";
  }

  return order.paid ? "決済済み" : "未決済";
}

function getItemStatusLabel(item: OrderDetailItem): string {
  if (item.isCancelled) {
    return "キャンセル済み";
  }

  if (item.isReturnRequested) {
    return "返品申請済み";
  }

  if (item.transferred) {
    return "受け取り済み";
  }

  if (item.isDispatched) {
    return "発送済み";
  }

  return "未発送";
}

function getProductTitle(item: OrderDetailItem): string {
  return item.productName || item.tokenName || "商品";
}

function getFallbackInitial(value?: string): string {
  const trimmed = value?.trim() || "";

  if (!trimmed) {
    return "?";
  }

  return trimmed.slice(0, 1).toUpperCase();
}

function getModelMetaItems(
  item: OrderDetailItem,
): Array<{
  label: string;
  value: string;
}> {
  const metaItems: Array<{
    label: string;
    value: string;
  }> = [];

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
    const volumeUnit = item.volumeUnit || "";

    metaItems.push({
      label: "容量",
      value: `${item.volumeValue}${volumeUnit}`,
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
          <div
            key={meta.label}
            className="order-detail-page__item-meta-row"
          >
            <dt>{meta.label}</dt>
            <dd>{meta.value}</dd>
          </div>
        ))}

        {measurements.map(([key, value]) => (
          <div
            key={key}
            className="order-detail-page__item-meta-row"
          >
            <dt>{getMeasurementLabel(key)}</dt>
            <dd>{value} mm</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

export default function OrderDetail() {
  const navigate = useNavigate();

  const {
    order,
    loading,
    cancellingItemIndex,
    returningItemIndex,
    error,
    reload,
    cancelItem,
    returnItem,
  } = useOrderDetail();

  const handleBack = () => {
    navigate("/wallet");
  };

  const handleOpenBrand = (brandId?: string) => {
    const id = brandId?.trim() || "";

    if (!id) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(id)}`);
  };

  const showError =
    !loading &&
    !order &&
    Boolean(error);

  const showDetail =
    !loading &&
    Boolean(order);

  return (
    <Layout
      title="注文詳細"
      titleClickable={false}
      showBackButton
      onBackButtonClick={handleBack}
      mode="mypage"
      showFooter
    >
      <section className="page-section order-detail-page">
        {loading ? (
          <div className="page-card">
            <p className="page-card__text">
              読み込み中です...
            </p>
          </div>
        ) : null}

        {showError ? (
          <div className="page-card">
            <SectionHeader
              title="注文情報を表示できません"
              titleAs="h2"
            >
              <p
                className="page-card__text"
                role="alert"
              >
                {error}
              </p>
            </SectionHeader>

            <div className="page-actions">
              <button
                type="button"
                className="page-button page-button--secondary"
                onClick={() => void reload()}
              >
                再読み込み
              </button>
            </div>
          </div>
        ) : null}

        {showDetail && order ? (
          <div className="page-stack">
            <div className="page-card order-detail-page__summary-card">
              <div className="order-detail-page__summary-header">
                <div>
                  <p className="order-detail-page__date">
                    注文日時:{" "}
                    {order.createdAt
                      ? formatDateTime(order.createdAt)
                      : "-"}
                  </p>

                  <h1 className="order-detail-page__order-id">
                    注文ID: {order.id}
                  </h1>
                </div>

                <span className="order-detail-page__status">
                  {getOrderStatusLabel(order)}
                </span>
              </div>

              {error ? (
                <p
                  className="page-card__text"
                  role="alert"
                >
                  {error}
                </p>
              ) : null}
            </div>

            <div className="page-card">
              <SectionHeader
                title="商品"
                titleAs="h2"
              />

              <ul className="order-detail-page__items">
                {order.items.map((item, index) => {
                  const productTitle = getProductTitle(item);
                  const brandName =
                    item.brandName ||
                    "ブランド未設定";
                  const itemKey =
                    `${order.id}-${item.inventoryId}-${item.modelId}-${index}`;
                  const isCancelling =
                    cancellingItemIndex === index;
                  const isReturning =
                    returningItemIndex === index;

                  const cancelDisabled =
                    item.isCancelled ||
                    item.isDispatched ||
                    item.transferred ||
                    isCancelling ||
                    returningItemIndex !== null;

                  const showReturnButton =
                    item.isDispatched &&
                    !item.transferred &&
                    !item.isCancelled &&
                    !item.isReturnRequested;

                  return (
                    <li
                      key={itemKey}
                      className="order-detail-page__item"
                    >
                      <div className="order-detail-page__item-image">
                        {item.tokenIcon ? (
                          <img
                            src={item.tokenIcon}
                            alt={
                              item.tokenName ||
                              productTitle
                            }
                            loading="lazy"
                          />
                        ) : (
                          <span className="order-detail-page__item-image-fallback">
                            {getFallbackInitial(
                              item.tokenName ||
                                productTitle,
                            )}
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
                          onClick={() =>
                            handleOpenBrand(item.brandId)
                          }
                        >
                          <MediaIcon
                            src={item.brandIcon}
                            alt={brandName}
                            fallback={getFallbackInitial(
                              brandName,
                            )}
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
                            <dd>
                              {formatAmount(
                                item.price * item.qty,
                              )}
                            </dd>
                          </div>

                          <div className="order-detail-page__item-meta-row">
                            <dt>発送状況</dt>
                            <dd>
                              {getItemStatusLabel(item)}
                            </dd>
                          </div>

                          {item.returnRequestedAt ? (
                            <div className="order-detail-page__item-meta-row">
                              <dt>返品申請日時</dt>
                              <dd>
                                {formatDateTime(
                                  item.returnRequestedAt,
                                )}
                              </dd>
                            </div>
                          ) : null}

                          {item.transferredAt ? (
                            <div className="order-detail-page__item-meta-row">
                              <dt>受取日時</dt>
                              <dd>
                                {formatDateTime(
                                  item.transferredAt,
                                )}
                              </dd>
                            </div>
                          ) : null}
                        </dl>

                        <div className="page-actions order-detail-page__cancel-actions">
                          {item.isReturnRequested ? (
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
                              disabled={
                                isReturning ||
                                cancellingItemIndex !== null
                              }
                              onClick={() =>
                                void returnItem(index)
                              }
                            >
                              {isReturning
                                ? "返品申請中..."
                                : "返品"}
                            </button>
                          ) : (
                            <button
                              type="button"
                              className="order-detail-page__cancel-button"
                              disabled={cancelDisabled}
                              onClick={() =>
                                void cancelItem(index)
                              }
                            >
                              {item.isCancelled
                                ? "キャンセル済み"
                                : isCancelling
                                  ? "キャンセル中..."
                                  : item.transferred
                                    ? "受け取り済み"
                                    : "商品をキャンセル"}
                            </button>
                          )}
                        </div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            </div>

            <div className="page-card">
              <SectionHeader
                title="お支払い"
                titleAs="h2"
              />

              <dl className="order-detail-page__detail-list">
                <div className="order-detail-page__detail-row">
                  <dt>商品小計</dt>
                  <dd>
                    {formatAmount(
                      getOrderSubtotal(order),
                    )}
                  </dd>
                </div>

                <div className="order-detail-page__detail-row">
                  <dt>配送料</dt>
                  <dd>
                    {formatAmount(
                      getOrderShippingAmount(order),
                    )}
                  </dd>
                </div>

                <div className="order-detail-page__detail-row order-detail-page__detail-row--total">
                  <dt>合計</dt>
                  <dd>
                    {formatAmount(
                      getOrderTotal(order),
                    )}
                  </dd>
                </div>

                <div className="order-detail-page__detail-row">
                  <dt>決済状況</dt>
                  <dd>
                    {order.paid
                      ? "決済済み"
                      : "未決済"}
                  </dd>
                </div>
              </dl>
            </div>
          </div>
        ) : null}
      </section>
    </Layout>
  );
}