// frontend/amol/src/pages/OrderConfirmedPage.tsx
import Layout from "../components/layout/Layout";
import { OrderConfirmedActions } from "../features/order-confirmed/components/OrderConfirmedActions";
import { OrderConfirmedHero } from "../features/order-confirmed/components/OrderConfirmedHero";
import { OrderConfirmedItemsCard } from "../features/order-confirmed/components/OrderConfirmedItemsCard";
import { OrderConfirmedPaymentCard } from "../features/order-confirmed/components/OrderConfirmedPaymentCard";
import { OrderConfirmedShippingCard } from "../features/order-confirmed/components/OrderConfirmedShippingCard";
import { useOrderConfirmedPage } from "../features/order-confirmed/hooks/useOrderConfirmedPage";
import "../styles/order-confirmed-page.css";

export default function OrderConfirmedPage() {
  const {
    amount,
    invoiceId,
    paymentId,
    statusLabel,
    items,
    shippingAddressLines,
    handleGoToWallet,
    handleGoToLists,
  } = useOrderConfirmedPage();

  return (
    <Layout
      title="注文完了"
      titleClickable={false}
      mode="mypage"
      showBackButton={false}
      showFooter
      hideHamburgerMenu
      hideSettingsButton
      mainClassName="order-confirmed-page"
    >
      <section className="order-confirmed-page__section">
        <OrderConfirmedHero />

        <div className="order-confirmed-page__content">
          <OrderConfirmedPaymentCard
            statusLabel={statusLabel}
            amount={amount}
            invoiceId={invoiceId}
            paymentId={paymentId}
          />

          <OrderConfirmedItemsCard items={items} />

          <OrderConfirmedShippingCard lines={shippingAddressLines} />

          <OrderConfirmedActions
            onGoToWallet={handleGoToWallet}
            onGoToLists={handleGoToLists}
          />
        </div>
      </section>
    </Layout>
  );
}