// frontend/amol/src/pages/PaymentPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { PaymentErrorModal } from "../features/payment/components/PaymentErrorModal";
import { PaymentItemsCard } from "../features/payment/components/PaymentItemsCard";
import { PaymentMethodsCard } from "../features/payment/components/PaymentMethodsCard";
import { ShippingAddressCard } from "../features/payment/components/ShippingAddressCard";
import { usePaymentPage } from "../features/payment/hooks/usePaymentPage";
import { useMobilePortrait } from "../components/hooks/useMobilePortrait";

import "../styles/payment-page.css";

export default function PaymentPage() {
  const navigate = useNavigate();
  const { listId } = useParams<{
    listId: string;
  }>();
  const isMobilePortrait = useMobilePortrait();

  const {
    amount,
    backTo,
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
    userFullName,
  } = usePaymentPage({
    listId,
    navigate,
  });

  if (isLoading) {
    return (
      <>
        <Layout
          title="お支払い"
          titleClickable={false}
          mode="mypage"
          showBackButton
          backTo={backTo}
          showFooter={false}
          hideHamburgerMenu
          hideSettingsButton
          mainClassName="payment-page"
        >
          <section className="payment-page__section">
            <p>決済情報を読み込んでいます。</p>
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
        title="お支払い"
        titleClickable={false}
        mode="mypage"
        showBackButton
        backTo={backTo}
        showFooter={isMobilePortrait}
        hideHamburgerMenu
        hideSettingsButton
        mainClassName="payment-page"
        actionButtonLabel={
          isMobilePortrait
            ? undefined
            : paymentButtonLabel
        }
        onActionButtonClick={
          isMobilePortrait
            ? undefined
            : handleSubmitPayment
        }
        actionButtonDisabled={isPaymentDisabled}
        footerProps={
          isMobilePortrait
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
          <div className="payment-page__content">
            <div className="payment-page__left-column">
              <PaymentItemsCard
                amount={amount}
                cartItems={cartItems}
                shippingAmount={shippingAmount}
                subtotalAmount={subtotalAmount}
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