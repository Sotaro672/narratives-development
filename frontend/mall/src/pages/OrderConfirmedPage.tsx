// frontend/amol/src/pages/OrderConfirmedPage.tsx

import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  OrderConfirmedActions,
} from "../features/order-confirmed/components/OrderConfirmedActions";
import {
  OrderConfirmedHero,
} from "../features/order-confirmed/components/OrderConfirmedHero";
import {
  OrderConfirmedItemsCard,
} from "../features/order-confirmed/components/OrderConfirmedItemsCard";
import {
  OrderConfirmedPaymentCard,
} from "../features/order-confirmed/components/OrderConfirmedPaymentCard";
import {
  OrderConfirmedShippingCard,
} from "../features/order-confirmed/components/OrderConfirmedShippingCard";
import {
  useOrderConfirmedPage,
} from "../features/order-confirmed/hooks/useOrderConfirmedPage";

import "../styles/order-confirmed-page.css";

export default function OrderConfirmedPage() {
  const navigate = useNavigate();

  const {
    amount,
    subtotalAmount,
    shippingAmount,
    taxAmount,
    orderId,
    statusLabel,
    items,
    shippingAddressLines,
  } = useOrderConfirmedPage();

  useEffect(() => {
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: "auto",
    });
  }, []);

  const handleGoToOrderDetail = () => {
    const normalizedOrderId = orderId.trim();

    if (!normalizedOrderId) {
      return;
    }

    navigate(
      `/orders/${encodeURIComponent(
        normalizedOrderId,
      )}`,
    );
  };

  return (
    <Layout
      title="AMOL"
      titleClickable
      mode="mypage"
      showFooter
      mainClassName="order-confirmed-page"
    >
      <section className="order-confirmed-page__section">
        <OrderConfirmedHero />

        <div className="order-confirmed-page__content">
          <OrderConfirmedPaymentCard
            statusLabel={statusLabel}
            subtotalAmount={subtotalAmount}
            shippingAmount={shippingAmount}
            taxAmount={taxAmount}
            amount={amount}
            orderId={orderId}
          />

          <OrderConfirmedItemsCard
            items={items}
          />

          <OrderConfirmedShippingCard
            lines={shippingAddressLines}
          />

          <OrderConfirmedActions
            onGoToOrderDetail={handleGoToOrderDetail}
          />
        </div>
      </section>
    </Layout>
  );
}