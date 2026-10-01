// frontend/mall/src/pages/PaymentPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileSwipeRightDismissPage from "../components/layout/MobileSwipeRightDismissPage";
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

  function handleDismiss() {
    navigate(-1);
  }

  const showMobilePurchaseFooter =
    isMobilePortrait &&
    !isLoading;

  const pageContent = (
    <Layout
      title={isMobilePortrait ? "" : "AMOL"}
      titleClickable={!isMobilePortrait}
      mode="mypage"
      showHeader={!isMobilePortrait}
      showFooter={showMobilePurchaseFooter}
      hideSettingsButton
      mainClassName={[
        "payment-page",
        isMobilePortrait ? "payment-page--mobile-swipe" : "",
      ]
        .filter(Boolean)
        .join(" ")}
      footerProps={
        showMobilePurchaseFooter
          ? {
              variant: "action",
              buttonLabel: paymentButtonLabel,
              disabled: isPaymentDisabled,
              onButtonClick: handleSubmitPayment,
            }
          : undefined
      }
    >
      <section className="payment-page__section">
        {isLoading ? (
          <TextState variant="loading">
            決済情報を読み込んでいます。
          </TextState>
        ) : (
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

              {!isMobilePortrait ? (
                <Button
                  variant="primary"
                  size="lg"
                  fullWidth
                  disabled={isPaymentDisabled}
                  onClick={() => void handleSubmitPayment()}
                >
                  {paymentButtonLabel}
                </Button>
              ) : null}
            </div>
          </div>
        )}
      </section>
    </Layout>
  );

  return (
    <>
      {isMobilePortrait ? (
        <MobileSwipeRightDismissPage
          title="お支払い"
          onDismiss={handleDismiss}
        >
          {pageContent}
        </MobileSwipeRightDismissPage>
      ) : (
        pageContent
      )}

      <PaymentErrorModal
        message={modalMessage}
        onClose={closeErrorModal}
      />
    </>
  );
}