//frontend\amol\src\pages\PaymentMethodPage.tsx
import { Elements } from "@stripe/react-stripe-js";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payment-method-page.css";

import Layout from "../components/layout/Layout";
import PaymentMethodCardholderCard from "../features/payment-method/components/PaymentMethodCardholderCard";
import PaymentMethodForm from "../features/payment-method/components/PaymentMethodForm";
import PaymentMethodStatusCard from "../features/payment-method/components/PaymentMethodStatusCard";
import usePaymentMethodPage from "../features/payment-method/hooks/usePaymentMethodPage";

export default function PaymentMethodPage() {
  const {
    paymentMethod,
    cardholderName,
    isLoading,
    isCreatingIntent,
    clientSecret,
    stripeCustomerId,
    stripePromise,
    errorMessage,
    normalizedCardholderName,
    setCardholderName,
    handleCreateSetupIntent,
    handleCompleted,
  } = usePaymentMethodPage();

  return (
    <Layout title="支払方法" showBackButton mode="signin" backTo="/lists">
      <section className="page-section settings-page">
        <div className="payment-method-page-content">
          <PaymentMethodStatusCard
            isLoading={isLoading}
            paymentMethod={paymentMethod}
          />

          {!clientSecret ? (
            <PaymentMethodCardholderCard
              cardholderName={cardholderName}
              isCreatingIntent={isCreatingIntent}
              isLoading={isLoading}
              onChangeCardholderName={setCardholderName}
            />
          ) : null}

          {errorMessage ? (
            <p className="payment-method-page-error">{errorMessage}</p>
          ) : null}

          {!clientSecret ? (
            <>
              <button
                type="button"
                onClick={handleCreateSetupIntent}
                disabled={
                  isCreatingIntent ||
                  isLoading ||
                  !stripePromise ||
                  !normalizedCardholderName
                }
                className="payment-method-page-start-button"
              >
                {isCreatingIntent ? "作成中..." : "支払方法を登録"}
              </button>

              <div className="payment-method-page-test-warning-card">
                <p className="payment-method-page-test-warning-card__title">
                  テスト環境でのご利用について
                </p>
                <p className="payment-method-page-test-warning-card__text">
                  ここはテスト環境です。実際のクレジットカードは登録せず、
                  テスト用クレジットカードのみを登録してください。
                </p>
                <div className="payment-method-page-test-warning-card__example">
                  <p>
                    <strong>カード番号:</strong> 4242 4242 4242 4242
                  </p>
                  <p>
                    <strong>有効期限:</strong> 12/30
                  </p>
                  <p>
                    <strong>CVC:</strong> 123
                  </p>
                </div>
              </div>
            </>
          ) : null}

          {clientSecret && stripePromise ? (
            <Elements stripe={stripePromise}>
              <PaymentMethodForm
                cardholderName={normalizedCardholderName}
                clientSecret={clientSecret}
                stripeCustomerId={stripeCustomerId}
                onCompleted={handleCompleted}
              />
            </Elements>
          ) : null}
        </div>
      </section>
    </Layout>
  );
}