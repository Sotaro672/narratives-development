// frontend/amol/src/pages/OrderConfirmedPage.tsx
import { useMemo } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import type { CartDisplayItem } from "../features/cart/types";
import { formatYen, getModelPrice, getModelVariation } from "../features/cart/utils/cartUtils";
import type { ShippingAddress } from "../features/shipping-address/types";
import "../styles/order-confirmed-page.css";

type ConfirmedPayment = {
  id?: string;
  paymentId?: string;
  invoiceId?: string;
  paymentMethodId?: string;
  amount?: number;
  status?: string;
  createdAt?: string;
};

type OrderConfirmedLocationState = {
  payment?: ConfirmedPayment;
  invoiceId?: string;
  paymentId?: string;
  paymentMethodId?: string;
  amount?: number;
  cartItems?: CartDisplayItem[];
  shippingAddress?: ShippingAddress | null;
};

function getShippingAddressLabel(address: ShippingAddress | null | undefined): string {
  if (!address) {
    return "";
  }

  const zipCode =
    "zipCode" in address && typeof address.zipCode === "string"
      ? address.zipCode
      : "zip_code" in address && typeof address.zip_code === "string"
        ? address.zip_code
        : "";

  const state =
    "state" in address && typeof address.state === "string"
      ? address.state
      : "";

  const city =
    "city" in address && typeof address.city === "string" ? address.city : "";

  const street =
    "street" in address && typeof address.street === "string"
      ? address.street
      : "";

  const street2 =
    "street2" in address && typeof address.street2 === "string"
      ? address.street2
      : "";

  const zipLine = zipCode ? `〒${zipCode}` : "";
  const addressLine = `${state}${city}${street}${street2}`.trim();

  return [zipLine, addressLine].filter(Boolean).join("\n");
}

function formatPaymentStatus(status?: string): string {
  const normalized = status?.trim().toUpperCase();

  switch (normalized) {
    case "SUCCEEDED":
      return "決済完了";
    case "PENDING":
      return "処理中";
    case "FAILED":
      return "失敗";
    case "CANCELED":
      return "キャンセル";
    default:
      return normalized || "決済完了";
  }
}

export default function OrderConfirmedPage() {
  const navigate = useNavigate();
  const location = useLocation();

  const state = (location.state ?? {}) as OrderConfirmedLocationState;

  const payment = state.payment ?? null;
  const cartItems = Array.isArray(state.cartItems) ? state.cartItems : [];
  const shippingAddress = state.shippingAddress ?? null;

  const invoiceId = payment?.invoiceId ?? state.invoiceId ?? "";
  const paymentId = payment?.id ?? payment?.paymentId ?? state.paymentId ?? "";
  const amount = payment?.amount ?? state.amount ?? 0;
  const status = payment?.status ?? "SUCCEEDED";

  const shippingAddressLabel = useMemo(() => {
    return getShippingAddressLabel(shippingAddress);
  }, [shippingAddress]);

  const handleGoToWallet = () => {
    navigate("/wallet");
  };

  const handleGoToLists = () => {
    navigate("/lists");
  };

  return (
    <Layout
      title="注文完了"
      mode="mypage"
      showBackButton={false}
      showFooter
      hideHamburgerMenu
      hideSettingsButton
      mainClassName="order-confirmed-page"
    >
      <section className="order-confirmed-page__section">
        <div className="order-confirmed-page__hero">
          <div className="order-confirmed-page__check" aria-hidden="true">
            ✓
          </div>

          <h1 className="order-confirmed-page__title">
            注文が完了しました
          </h1>

          <p className="order-confirmed-page__description">
            ご注文ありがとうございます。決済が正常に完了しました。
          </p>
        </div>

        <div className="order-confirmed-page__content">
          <section className="order-confirmed-page__card">
            <h2 className="order-confirmed-page__card-title">決済情報</h2>

            <dl className="order-confirmed-page__details">
              <div className="order-confirmed-page__detail-row">
                <dt>ステータス</dt>
                <dd>{formatPaymentStatus(status)}</dd>
              </div>

              <div className="order-confirmed-page__detail-row">
                <dt>金額</dt>
                <dd>{formatYen(amount)}</dd>
              </div>

              {invoiceId ? (
                <div className="order-confirmed-page__detail-row">
                  <dt>請求ID</dt>
                  <dd>{invoiceId}</dd>
                </div>
              ) : null}

              {paymentId ? (
                <div className="order-confirmed-page__detail-row">
                  <dt>決済ID</dt>
                  <dd>{paymentId}</dd>
                </div>
              ) : null}
            </dl>
          </section>

          <section className="order-confirmed-page__card">
            <h2 className="order-confirmed-page__card-title">注文内容</h2>

            {cartItems.length > 0 ? (
              <ul className="order-confirmed-page__items">
                {cartItems.map((item) => {
                  const catalog = item.catalog;
                  const model = getModelVariation(catalog, item.modelId);
                  const price = getModelPrice(catalog, item.modelId);
                  const lineAmount = price === null ? null : price * item.qty;
                  const title =
                    catalog?.productBlueprint.productName ||
                    catalog?.list.title ||
                    "商品名未設定";

                  return (
                    <li className="order-confirmed-page__item" key={item.itemKey}>
                      <div>
                        <p className="order-confirmed-page__item-title">
                          {title}
                        </p>

                        <p className="order-confirmed-page__item-meta">
                          {model?.colorName ? `カラー: ${model.colorName}` : ""}
                          {model?.colorName && model?.size ? " / " : ""}
                          {model?.size ? `サイズ: ${model.size}` : ""}
                        </p>

                        <p className="order-confirmed-page__item-meta">
                          数量: {item.qty}
                        </p>
                      </div>

                      <p className="order-confirmed-page__item-price">
                        {lineAmount === null ? "価格未設定" : formatYen(lineAmount)}
                      </p>
                    </li>
                  );
                })}
              </ul>
            ) : (
              <p className="order-confirmed-page__empty">
                注文内容を取得できませんでした。
              </p>
            )}
          </section>

          <section className="order-confirmed-page__card">
            <h2 className="order-confirmed-page__card-title">配送先情報</h2>

            {shippingAddressLabel ? (
              <div className="order-confirmed-page__shipping-address">
                {shippingAddressLabel.split("\n").map((line) => (
                  <p
                    className="order-confirmed-page__shipping-address-line"
                    key={line}
                  >
                    {line}
                  </p>
                ))}
              </div>
            ) : (
              <p className="order-confirmed-page__empty">
                配送先情報を取得できませんでした。
              </p>
            )}
          </section>

          <div className="order-confirmed-page__actions">
            <button
              type="button"
              className="order-confirmed-page__primary-button"
              onClick={handleGoToWallet}
            >
              ウォレットへ
            </button>

            <button
              type="button"
              className="order-confirmed-page__secondary-button"
              onClick={handleGoToLists}
            >
              商品一覧へ
            </button>
          </div>
        </div>
      </section>
    </Layout>
  );
}