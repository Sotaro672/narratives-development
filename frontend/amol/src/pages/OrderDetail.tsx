// frontend/amol/src/pages/OrderDetail.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import MediaIcon from "../components/ui/MediaIcon";
import SectionHeader from "../components/ui/SectionHeader";
import { formatDateTime } from "../components/utils/date";

import { useOrderDetail } from "../features/order/hooks/useOrderDetail";
import type {
  WalletOrder,
  WalletOrderItemSnapshot,
} from "../features/shared/types/orderTypes";
import { formatAmount } from "../features/wallet/utils/format";

import "../styles/page-layout.css";
import "../styles/order-detail-page.css";

function getOrderSubtotal(order: WalletOrder): number {
  return order.items.reduce((sum, item) => {
    return sum + item.price * item.qty;
  }, 0);
}

function getOrderShippingAmount(order: WalletOrder): number {
  const amount = order.shippingQuoteSnapshot?.amount;

  if (!Number.isFinite(amount)) {
    return 0;
  }

  return amount;
}

function getOrderTotal(order: WalletOrder): number {
  return (
    getOrderSubtotal(order) +
    getOrderShippingAmount(order)
  );
}

function getOrderStatusLabel(order: WalletOrder): string {
  if (order.items.length === 0) {
    return "商品なし";
  }

  const allCanceled = order.items.every(
    (item) => item.isCanceled,
  );

  if (allCanceled) {
    return "キャンセル済み";
  }

  const allTransferred = order.items.every(
    (item) => item.transferred,
  );

  if (allTransferred) {
    return order.paid
      ? "受け取り済み"
      : "未決済";
  }

  const partiallyTransferred = order.items.some(
    (item) => item.transferred,
  );

  if (partiallyTransferred) {
    return order.paid
      ? "一部受け取り済み"
      : "未決済";
  }

  const allDispatched = order.items.every(
    (item) => item.isDispatched,
  );

  if (allDispatched) {
    return "発送済み";
  }

  const partiallyDispatched = order.items.some(
    (item) => item.isDispatched,
  );

  if (partiallyDispatched) {
    return "一部発送済み";
  }

  return order.paid
    ? "決済済み"
    : "未決済";
}

function getProductTitle(
  item: WalletOrderItemSnapshot,
): string {
  return (
    item.productName ||
    item.tokenName ||
    "商品"
  );
}

function getFallbackInitial(
  value?: string,
): string {
  const trimmed = value?.trim() || "";

  if (!trimmed) {
    return "?";
  }

  return trimmed
    .slice(0, 1)
    .toUpperCase();
}

function getShippingAddressLines(
  order: WalletOrder,
): string[] {
  const shipping = order.shippingSnapshot;

  if (!shipping) {
    return [];
  }

  return [
    shipping.zipCode
      ? `〒${shipping.zipCode}`
      : "",
    [
      shipping.state,
      shipping.city,
      shipping.street,
    ]
      .filter(Boolean)
      .join(""),
    shipping.street2 || "",
    shipping.country &&
    shipping.country !== "JP"
      ? shipping.country
      : "",
  ].filter(Boolean);
}

function getPaymentMethodLabel(
  order: WalletOrder,
): string {
  const paymentMethod =
    order.paymentMethodSnapshot;

  if (!paymentMethod) {
    return "-";
  }

  const brand =
    paymentMethod.brand?.trim().toUpperCase() ||
    "CARD";

  const last4 =
    paymentMethod.last4?.trim() || "";

  if (!last4) {
    return brand;
  }

  return `${brand} •••• ${last4}`;
}

function getPaymentExpiryLabel(
  order: WalletOrder,
): string {
  const paymentMethod =
    order.paymentMethodSnapshot;

  if (
    !paymentMethod ||
    !paymentMethod.expMonth ||
    !paymentMethod.expYear
  ) {
    return "-";
  }

  return `${String(paymentMethod.expMonth).padStart(
    2,
    "0",
  )}/${paymentMethod.expYear}`;
}

export default function OrderDetail() {
  const navigate = useNavigate();

  const {
    order,
    loading,
    error,
    reload,
  } = useOrderDetail();

  const handleBack = () => {
    navigate("/wallet");
  };

  const handleOpenBrand = (
    brandId?: string,
  ) => {
    const id = brandId?.trim() || "";

    if (!id) {
      return;
    }

    navigate(
      `/brands/${encodeURIComponent(id)}`,
    );
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

              <button
                type="button"
                className="page-button page-button--primary"
                onClick={handleBack}
              >
                ウォレットへ戻る
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
                    {formatDateTime(
                      order.createdAt,
                    )}
                  </p>

                  <h1 className="order-detail-page__order-id">
                    注文ID: {order.id}
                  </h1>
                </div>

                <span className="order-detail-page__status">
                  {getOrderStatusLabel(
                    order,
                  )}
                </span>
              </div>
            </div>

            <div className="page-card">
              <SectionHeader
                title="商品"
                titleAs="h2"
              />

              <ul className="order-detail-page__items">
                {order.items.map(
                  (item, index) => {
                    const productTitle =
                      getProductTitle(
                        item,
                      );

                    const brandName =
                      item.brandName ||
                      "ブランド未設定";

                    const itemKey =
                      `${order.id}-${item.inventoryId}-${item.modelId}-${index}`;

                    return (
                      <li
                        key={itemKey}
                        className="order-detail-page__item"
                      >
                        <div className="order-detail-page__item-image">
                          {item.tokenIcon ? (
                            <img
                              src={
                                item.tokenIcon
                              }
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
                                {
                                  productTitle
                                }
                              </span>

                              {item.tokenName ? (
                                <span className="order-detail-page__item-token-name">
                                  {
                                    item.tokenName
                                  }
                                </span>
                              ) : null}
                            </div>

                            <span className="order-detail-page__item-price">
                              {formatAmount(
                                item.price,
                              )}
                            </span>
                          </div>

                          <button
                            type="button"
                            className="order-detail-page__brand"
                            disabled={
                              !item.brandId
                            }
                            onClick={() =>
                              handleOpenBrand(
                                item.brandId,
                              )
                            }
                          >
                            <MediaIcon
                              src={
                                item.brandIcon
                              }
                              alt={
                                brandName
                              }
                              fallback={getFallbackInitial(
                                brandName,
                              )}
                              size="xs"
                              shape="circle"
                            />

                            <span>
                              {
                                brandName
                              }
                            </span>
                          </button>

                          <dl className="order-detail-page__item-meta">
                            <div className="order-detail-page__item-meta-row">
                              <dt>数量</dt>
                              <dd>
                                {item.qty}点
                              </dd>
                            </div>

                            <div className="order-detail-page__item-meta-row">
                              <dt>小計</dt>
                              <dd>
                                {formatAmount(
                                  item.price *
                                    item.qty,
                                )}
                              </dd>
                            </div>

                            <div className="order-detail-page__item-meta-row">
                              <dt>発送状況</dt>
                              <dd>
                                {item.isCanceled
                                  ? "キャンセル済み"
                                  : item.transferred
                                    ? "受け取り済み"
                                    : item.isDispatched
                                      ? "発送済み"
                                      : "未発送"}
                              </dd>
                            </div>

                            {item.transferredAt ? (
                              <div className="order-detail-page__item-meta-row">
                                <dt>
                                  受取日時
                                </dt>
                                <dd>
                                  {formatDateTime(
                                    item.transferredAt,
                                  )}
                                </dd>
                              </div>
                            ) : null}
                          </dl>
                        </div>
                      </li>
                    );
                  },
                )}
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
                      getOrderSubtotal(
                        order,
                      ),
                    )}
                  </dd>
                </div>

                <div className="order-detail-page__detail-row">
                  <dt>配送料</dt>
                  <dd>
                    {formatAmount(
                      getOrderShippingAmount(
                        order,
                      ),
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

            <div className="page-card">
              <SectionHeader
                title="配送先"
                titleAs="h2"
              />

              <div className="order-detail-page__address">
                {getShippingAddressLines(
                  order,
                ).length > 0 ? (
                  getShippingAddressLines(
                    order,
                  ).map(
                    (line, index) => (
                      <p
                        key={`${line}-${index}`}
                        className="order-detail-page__address-line"
                      >
                        {line}
                      </p>
                    ),
                  )
                ) : (
                  <p className="page-card__text">
                    配送先情報がありません。
                  </p>
                )}
              </div>
            </div>

            <div className="page-card">
              <SectionHeader
                title="支払方法"
                titleAs="h2"
              />

              <dl className="order-detail-page__detail-list">
                <div className="order-detail-page__detail-row">
                  <dt>カード</dt>
                  <dd>
                    {getPaymentMethodLabel(
                      order,
                    )}
                  </dd>
                </div>

                <div className="order-detail-page__detail-row">
                  <dt>有効期限</dt>
                  <dd>
                    {getPaymentExpiryLabel(
                      order,
                    )}
                  </dd>
                </div>

                {order.paymentMethodSnapshot
                  ?.cardholderName ? (
                  <div className="order-detail-page__detail-row">
                    <dt>
                      カード名義
                    </dt>
                    <dd>
                      {
                        order
                          .paymentMethodSnapshot
                          .cardholderName
                      }
                    </dd>
                  </div>
                ) : null}
              </dl>
            </div>

            <div className="page-actions">
              <button
                type="button"
                className="page-button page-button--secondary"
                onClick={handleBack}
              >
                ウォレットへ戻る
              </button>
            </div>
          </div>
        ) : null}
      </section>
    </Layout>
  );
}