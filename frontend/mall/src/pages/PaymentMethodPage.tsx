// frontend/amol/src/pages/PaymentMethodPage.tsx

import { Elements } from "@stripe/react-stripe-js";

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import PaymentMethodCardholderCard from "../features/payment-method/components/PaymentMethodCardholderCard";
import PaymentMethodForm from "../features/payment-method/components/PaymentMethodForm";
import PaymentMethodStatusCard from "../features/payment-method/components/PaymentMethodStatusCard";
import usePaymentMethodPage from "../features/payment-method/hooks/usePaymentMethodPage";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payment-method-page.css";

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

  const registrationDisabled =
    isCreatingIntent ||
    isLoading ||
    !stripePromise ||
    !normalizedCardholderName;

  return (
    <Layout
      title="支払方法"
      titleClickable={false}
      mode="signin"
    >
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
            <Alert variant="error">
              {errorMessage}
            </Alert>
          ) : null}

          {!clientSecret ? (
            <>
              <Button
                type="button"
                variant="primary"
                size="lg"
                fullWidth
                disabled={registrationDisabled}
                aria-busy={isCreatingIntent}
                onClick={handleCreateSetupIntent}
              >
                {isCreatingIntent ? "作成中..." : "支払方法を登録"}
              </Button>

              <Alert
                variant="warning"
                className="payment-method-page-test-warning-card"
              >
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
              </Alert>
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