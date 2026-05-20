//frontend\amol\src\features\payment-method\components\PaymentMethodForm.tsx
import { useState } from "react";
import type { FormEvent } from "react";
import { CardElement, useElements, useStripe } from "@stripe/react-stripe-js";

import Button from "../../../components/ui/Button";
import type { ConfirmedCardPayload } from "../types";

type PaymentMethodFormProps = {
  cardholderName: string;
  clientSecret: string;
  stripeCustomerId: string;
  onCompleted: (payload: ConfirmedCardPayload) => Promise<void> | void;
};

export default function PaymentMethodForm(props: PaymentMethodFormProps) {
  const { cardholderName, clientSecret, stripeCustomerId, onCompleted } = props;
  const stripe = useStripe();
  const elements = useElements();

  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [isCardComplete, setIsCardComplete] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!stripe || !elements) {
      setErrorMessage("Stripe の初期化が完了していません。");
      return;
    }

    const cardElement = elements.getElement(CardElement);

    if (!cardElement) {
      setErrorMessage("カード入力欄の初期化が完了していません。");
      return;
    }

    const normalizedCardholderName = cardholderName.trim();

    if (!normalizedCardholderName) {
      setErrorMessage("カード名義人を入力してください。");
      return;
    }

    if (!stripeCustomerId.trim()) {
      setErrorMessage("Stripe Customer ID を取得できませんでした。");
      return;
    }

    if (!isCardComplete) {
      setErrorMessage("カード番号・有効期限・CVCを入力してください。");
      return;
    }

    try {
      setSubmitting(true);
      setErrorMessage("");

      const paymentMethodResult = await stripe.createPaymentMethod({
        type: "card",
        card: cardElement,
        billing_details: {
          name: normalizedCardholderName,
        },
      });

      if (paymentMethodResult.error) {
        throw new Error(
          paymentMethodResult.error.message ||
            "カード情報の作成に失敗しました。",
        );
      }

      const stripePaymentMethod = paymentMethodResult.paymentMethod;

      if (!stripePaymentMethod?.id) {
        throw new Error("Stripe PaymentMethod ID を取得できませんでした。");
      }

      if (!stripePaymentMethod.card) {
        throw new Error("カード情報を取得できませんでした。");
      }

      const setupResult = await stripe.confirmCardSetup(clientSecret, {
        payment_method: stripePaymentMethod.id,
      });

      if (setupResult.error) {
        throw new Error(
          setupResult.error.message || "カード登録に失敗しました。",
        );
      }

      await onCompleted({
        stripeCustomerId: stripeCustomerId.trim(),
        stripePaymentMethodId: stripePaymentMethod.id,
        brand: stripePaymentMethod.card.brand,
        last4: stripePaymentMethod.card.last4,
        expMonth: stripePaymentMethod.card.exp_month,
        expYear: stripePaymentMethod.card.exp_year,
        cardholderName: normalizedCardholderName,
      });
    } catch (error) {
      if (error instanceof Error) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage("カード登録に失敗しました。");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="payment-method-page-form">
      <div className="payment-method-page-form__element">
        <label className="payment-method-page-card__text">
          <strong>カード情報</strong>
        </label>

        <div className="payment-method-page-card-element">
          <CardElement
            onChange={(event) => {
              setIsCardComplete(event.complete);

              if (event.error?.message) {
                setErrorMessage(event.error.message);
              } else {
                setErrorMessage("");
              }
            }}
            options={{
              hidePostalCode: true,
              style: {
                base: {
                  fontSize: "16px",
                  color: "#111827",
                  "::placeholder": {
                    color: "#9ca3af",
                  },
                },
                invalid: {
                  color: "#b91c1c",
                },
              },
            }}
          />
        </div>
      </div>

      {errorMessage ? (
        <p className="payment-method-page-form__error">{errorMessage}</p>
      ) : null}

      <div className="payment-method-page-form__actions">
        <Button
          type="submit"
          disabled={!stripe || !elements || submitting || !isCardComplete}
        >
          {submitting ? "登録中..." : "このカードを登録する"}
        </Button>
      </div>

      <p className="payment-method-page-form__note">
        カード番号・有効期限・CVCなどのカード情報はStripeにより安全に処理されます。
      </p>
    </form>
  );
}