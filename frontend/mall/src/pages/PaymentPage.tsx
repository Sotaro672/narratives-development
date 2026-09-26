// frontend/amol/src/pages/PaymentPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import TextState from "../components/ui/TextState";
import { PaymentErrorModal } from "../features/payment/components/PaymentErrorModal";
import { PaymentItemsCard } from "../features/payment/components/PaymentItemsCard";
import { PaymentMethodsCard } from "../features/payment/components/PaymentMethodsCard";
import { ShippingAddressCard } from "../features/payment/components/ShippingAddressCard";
import { usePaymentPage } from "../features/payment/hooks/usePaymentPage";

import "../styles/payment-page.css";

export default function PaymentPage() {
  const navigate = useNavigate();
  const { listId } = useParams<{ listId: string }>();
  const isMobilePortrait = useMobilePortrait();

  const {
    amount,
    cartItems,
    closeErrorModal,
    handleGoToPaymentMethod,
    handleGoToShippingAddress,
    handleSubmitPayment,
    isLoading,
    isPaymentDisabled,
    modalMessage,
    paymentButtonLabel,
    paymentMethods,
    primaryShippingAddress,
    selectedPaymentMethodId,
    setSelectedPaymentMethodId,
    shippingAddressLabel,
    shippingAmount,
    subtotalAmount,
    taxAmount,
    userFullName,
  } = usePaymentPage({
    listId,
    navigate,
  });

  if (isLoading) {
    return (
      <>
        <Layout
          title="AMOL"
          mode="mypage"
          showFooter={isMobilePortrait}
          hideHamburgerMenu
          hideSettingsButton
          mainClassName="payment-page"
        >
          <section className="payment-page__section">
            <TextState variant="loading">
              決済情報を読み込んでいます。
            </TextState>
          </section>
        </Layout>

        <PaymentErrorModal
          message={modalMessage}
          onClose={closeErrorModal}
        />
      </>
    );
  }

  return (
    <>
      <Layout
        title="AMOL"
        mode="mypage"
        showFooter={isMobilePortrait}
        hideHamburgerMenu
        hideSettingsButton
        mainClassName="payment-page"
      >
        <section className="payment-page__section">
          <div className="payment-page__content">
            <div className="payment-page__left-column">
              <PaymentItemsCard
                amount={amount}
                cartItems={cartItems}
                shippingAmount={shippingAmount}
                subtotalAmount={subtotalAmount}
                taxAmount={taxAmount}
              />
            </div>

            <div className="payment-page__right-column">
              <ShippingAddressCard
                primaryShippingAddress={primaryShippingAddress}
                shippingAddressLabel={shippingAddressLabel}
                userFullName={userFullName}
                onGoToShippingAddress={handleGoToShippingAddress}
              />

              <PaymentMethodsCard
                paymentMethods={paymentMethods}
                selectedPaymentMethodId={selectedPaymentMethodId}
                onSelectPaymentMethod={setSelectedPaymentMethodId}
                onGoToPaymentMethod={handleGoToPaymentMethod}
              />

              <Button
                variant="primary"
                size="lg"
                fullWidth
                disabled={isPaymentDisabled}
                onClick={() => void handleSubmitPayment()}
              >
                {paymentButtonLabel}
              </Button>
            </div>
          </div>
        </section>
      </Layout>

      <PaymentErrorModal
        message={modalMessage}
        onClose={closeErrorModal}
      />
    </>
  );
}