// frontend/amol/src/pages/PaymentMethodPage.tsx
import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { getAuth } from "firebase/auth";
import { loadStripe } from "@stripe/stripe-js";
import type { Stripe } from "@stripe/stripe-js";
import {
  CardElement,
  Elements,
  useElements,
  useStripe,
} from "@stripe/react-stripe-js";
import { useLocation, useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payment-method-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

type CardPaymentMethod = {
  id: string;
  userId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
};

type PaymentMethodListResponse = {
  data?: CardPaymentMethod[];
  error?: string;
};

type PaymentMethodDefaultResponse = {
  data?: CardPaymentMethod | null;
  error?: string;
};

type SetupIntentData = {
  clientSecret?: string;
  stripeCustomerId?: string;
};

type SetupIntentResponse = {
  data?: SetupIntentData;
  clientSecret?: string;
  stripeCustomerId?: string;
  error?: string;
};

type StripeConfigResponse = {
  publishableKey?: string;
  error?: string;
};

type SavePaymentMethodResponse = {
  data?: CardPaymentMethod;
  error?: string;
};

type ConfirmedCardPayload = {
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
};

type PaymentMethodPageLocationState = {
  returnTo?: string;
  fromRoomPayment?: boolean;
  amount?: number;
  selectedPaymentMethod?: "card" | "paypay";
  shouldResumeCardPayment?: boolean;
};

function cardBrandLabel(brand: string): string {
  if (!brand) {
    return "カード";
  }

  switch (brand.toLowerCase()) {
    case "visa":
      return "Visa";
    case "mastercard":
      return "Mastercard";
    case "amex":
      return "American Express";
    case "jcb":
      return "JCB";
    case "diners":
      return "Diners Club";
    case "discover":
      return "Discover";
    default:
      return brand;
  }
}

function getBackendUrl(): string {
  const value = import.meta.env.VITE_API_BASE_URL;

  if (typeof value === "string" && value.trim() !== "") {
    return value.replace(/\/+$/, "");
  }

  return "";
}

async function readJsonResponse<T>(response: Response): Promise<T | null> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return (await response.json()) as T;
}

function extractSetupIntentClientSecret(
  responseBody: SetupIntentResponse | null,
): string {
  const clientSecret =
    responseBody?.data?.clientSecret ?? responseBody?.clientSecret ?? "";

  return typeof clientSecret === "string" ? clientSecret.trim() : "";
}

function extractSetupIntentStripeCustomerId(
  responseBody: SetupIntentResponse | null,
): string {
  const stripeCustomerId =
    responseBody?.data?.stripeCustomerId ?? responseBody?.stripeCustomerId ?? "";

  return typeof stripeCustomerId === "string" ? stripeCustomerId.trim() : "";
}

function selectPrimaryPaymentMethod(
  listResponse: PaymentMethodListResponse | null,
  defaultResponse: PaymentMethodDefaultResponse | null,
): CardPaymentMethod | null {
  if (defaultResponse?.data) {
    return defaultResponse.data;
  }

  const items = Array.isArray(listResponse?.data) ? listResponse.data : [];

  return items.find((item) => item.isDefault) ?? items[0] ?? null;
}

function getStripePublishableKey(
  responseBody: StripeConfigResponse | null,
): string {
  const publishableKey = responseBody?.publishableKey ?? "";

  return typeof publishableKey === "string" ? publishableKey.trim() : "";
}

function PaymentMethodForm(props: {
  cardholderName: string;
  clientSecret: string;
  stripeCustomerId: string;
  onCompleted: (payload: ConfirmedCardPayload) => Promise<void> | void;
}) {
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

export default function PaymentMethodPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const auth = getAuth();

  const locationState = (location.state ?? {}) as PaymentMethodPageLocationState;
  const returnTo =
    typeof locationState.returnTo === "string" ? locationState.returnTo : "";
  const restoredAmount =
    typeof locationState.amount === "number" ? locationState.amount : undefined;
  const restoredPaymentMethod =
    locationState.selectedPaymentMethod === "paypay" ? "paypay" : "card";
  const shouldResumeCardPayment = !!locationState.shouldResumeCardPayment;

  const backendUrl = useMemo(() => getBackendUrl(), []);

  const [paymentMethod, setPaymentMethod] = useState<CardPaymentMethod | null>(
    null,
  );
  const [cardholderName, setCardholderName] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isCreatingIntent, setIsCreatingIntent] = useState(false);
  const [clientSecret, setClientSecret] = useState("");
  const [stripeCustomerId, setStripeCustomerId] = useState("");
  const [stripePromise, setStripePromise] =
    useState<Promise<Stripe | null> | null>(null);
  const [errorMessage, setErrorMessage] = useState("");

  const normalizedCardholderName = useMemo(() => {
    return cardholderName.trim();
  }, [cardholderName]);

  const fetchCurrentPaymentMethod = async (
    backendUrlValue: string,
    idToken: string,
  ): Promise<CardPaymentMethod | null> => {
    const [listResponse, defaultResponse] = await Promise.all([
      fetch(`${backendUrlValue}/mall/me/payment-methods`, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
        credentials: "include",
      }),
      fetch(`${backendUrlValue}/mall/me/payment-methods/default`, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
        credentials: "include",
      }),
    ]);

    const listBody =
      await readJsonResponse<PaymentMethodListResponse>(listResponse);

    const defaultBody =
      await readJsonResponse<PaymentMethodDefaultResponse>(defaultResponse);

    if (!listResponse.ok) {
      throw new Error(listBody?.error || "支払方法の取得に失敗しました。");
    }

    if (!defaultResponse.ok && defaultResponse.status !== 404) {
      throw new Error(
        defaultBody?.error || "既定の支払方法の取得に失敗しました。",
      );
    }

    return selectPrimaryPaymentMethod(listBody, defaultBody);
  };

  const savePaymentMethod = async (
    backendUrlValue: string,
    idToken: string,
    payload: ConfirmedCardPayload,
  ): Promise<CardPaymentMethod | null> => {
    const response = await fetch(`${backendUrlValue}/mall/me/payment-methods`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        stripeCustomerId: payload.stripeCustomerId,
        stripePaymentMethodId: payload.stripePaymentMethodId,
        brand: payload.brand,
        last4: payload.last4,
        expMonth: payload.expMonth,
        expYear: payload.expYear,
        cardholderName: payload.cardholderName,
        isDefault: true,
      }),
    });

    const responseBody = await readJsonResponse<SavePaymentMethodResponse>(
      response,
    );

    if (!response.ok) {
      throw new Error(responseBody?.error || "支払方法の保存に失敗しました。");
    }

    return responseBody?.data ?? null;
  };

  useEffect(() => {
    const initializePage = async () => {
      const currentUser = auth.currentUser;

      if (!currentUser) {
        navigate("/signin", { replace: true });
        return;
      }

      try {
        if (!backendUrl) {
          throw new Error("VITE_API_BASE_URL が設定されていません。");
        }

        const idToken = await currentUser.getIdToken(true);

        const stripeConfigResponse = await fetch(
          `${backendUrl}/mall/config/stripe`,
          {
            method: "GET",
            credentials: "include",
          },
        );

        const stripeConfigBody =
          await readJsonResponse<StripeConfigResponse>(stripeConfigResponse);

        if (!stripeConfigResponse.ok) {
          throw new Error(
            stripeConfigBody?.error || "Stripe 公開鍵の取得に失敗しました。",
          );
        }

        const publishableKey = getStripePublishableKey(stripeConfigBody);

        if (!publishableKey) {
          throw new Error("Stripe 公開鍵を取得できませんでした。");
        }

        setStripePromise(loadStripe(publishableKey));

        const nextPaymentMethod = await fetchCurrentPaymentMethod(
          backendUrl,
          idToken,
        );

        setPaymentMethod(nextPaymentMethod);

        if (nextPaymentMethod?.cardholderName) {
          setCardholderName(nextPaymentMethod.cardholderName);
        }
      } catch (error) {
        console.error(error);

        if (error instanceof Error) {
          setErrorMessage(error.message);
        } else {
          setErrorMessage("支払方法の取得に失敗しました。");
        }
      } finally {
        setIsLoading(false);
      }
    };

    void initializePage();
  }, [auth, backendUrl, navigate]);

  const handleCreateSetupIntent = async () => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      window.alert("ログイン情報を確認できませんでした。");
      return;
    }

    try {
      setIsCreatingIntent(true);
      setErrorMessage("");

      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URL が設定されていません。");
      }

      if (!stripePromise) {
        throw new Error("Stripe の初期化が完了していません。");
      }

      if (!normalizedCardholderName) {
        throw new Error("カード名義人を入力してください。");
      }

      const idToken = await currentUser.getIdToken(true);

      const response = await fetch(
        `${backendUrl}/mall/me/payment-methods/setup-intent`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${idToken}`,
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            cardholderName: normalizedCardholderName,
          }),
        },
      );

      const responseBody = await readJsonResponse<SetupIntentResponse>(response);

      if (!response.ok) {
        throw new Error(
          responseBody?.error || "SetupIntent の作成に失敗しました。",
        );
      }

      const nextClientSecret = extractSetupIntentClientSecret(responseBody);
      const nextStripeCustomerId =
        extractSetupIntentStripeCustomerId(responseBody);

      if (!nextClientSecret) {
        throw new Error("clientSecret を取得できませんでした。");
      }

      if (!nextStripeCustomerId) {
        throw new Error("Stripe Customer ID を取得できませんでした。");
      }

      setClientSecret(nextClientSecret);
      setStripeCustomerId(nextStripeCustomerId);
    } catch (error) {
      console.error(error);

      if (error instanceof Error) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage("SetupIntent の作成に失敗しました。");
      }
    } finally {
      setIsCreatingIntent(false);
    }
  };

  const handleCompleted = async (payload: ConfirmedCardPayload) => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    try {
      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URL が設定されていません。");
      }

      const idToken = await currentUser.getIdToken(true);

      const savedPaymentMethod = await savePaymentMethod(
        backendUrl,
        idToken,
        payload,
      );

      setPaymentMethod(savedPaymentMethod);
      setClientSecret("");
      setStripeCustomerId("");

      if (savedPaymentMethod?.cardholderName) {
        setCardholderName(savedPaymentMethod.cardholderName);
      }

      const nextPaymentMethod = await fetchCurrentPaymentMethod(
        backendUrl,
        idToken,
      );

      setPaymentMethod(nextPaymentMethod);

      if (returnTo) {
        navigate(returnTo, {
          replace: true,
          state: {
            restoredAmount,
            restoredPaymentMethod,
            paymentMethodRegistered: true,
            shouldResumeCardPayment,
          },
        });
        return;
      }

      window.alert("カード登録が完了しました。");
    } catch (error) {
      console.error(error);

      if (error instanceof Error) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage("カード登録後の保存に失敗しました。");
      }
    }
  };

  return (
    <Layout title="支払方法" showBackButton mode="signin" backTo="/lists">
      <section className="page-section settings-page">
        <div className="payment-method-page-content">
          <div className="payment-method-page-card">
            <h2 className="payment-method-page-card__title">登録状況</h2>

            {isLoading ? (
              <p className="payment-method-page-card__text">読み込み中...</p>
            ) : paymentMethod ? (
              <div className="payment-method-page-card__details">
                <p className="payment-method-page-card__text">
                  <strong>ブランド:</strong>{" "}
                  {cardBrandLabel(paymentMethod.brand)}
                </p>
                <p className="payment-method-page-card__text">
                  <strong>下4桁:</strong> {paymentMethod.last4 || "-"}
                </p>
                <p className="payment-method-page-card__text">
                  <strong>有効期限:</strong> {paymentMethod.expMonth}/
                  {paymentMethod.expYear}
                </p>
                <p className="payment-method-page-card__text">
                  <strong>カード名義人:</strong>{" "}
                  {paymentMethod.cardholderName || "-"}
                </p>
                <p className="payment-method-page-card__text">
                  <strong>既定カード:</strong>{" "}
                  {paymentMethod.isDefault ? "はい" : "いいえ"}
                </p>
              </div>
            ) : (
              <div className="payment-method-page-card__details">
                <p className="payment-method-page-card__text">
                  登録済みの支払方法はありません。
                </p>
              </div>
            )}
          </div>

          {!clientSecret ? (
            <div className="payment-method-page-card">
              <label
                className="payment-method-page-card__text"
                htmlFor="cardholderName"
              >
                <strong>カード名義人</strong>
              </label>

              <input
                id="cardholderName"
                type="text"
                value={cardholderName}
                onChange={(event) => setCardholderName(event.target.value)}
                placeholder="例: TARO YAMADA"
                className="payment-method-page-input"
                autoComplete="cc-name"
                disabled={isCreatingIntent || isLoading}
              />

              <p className="payment-method-page-form__note">
                カードに記載されている名義人を入力してください。
              </p>
            </div>
          ) : null}

          {errorMessage ? (
            <p className="payment-method-page-error">{errorMessage}</p>
          ) : null}

          {!clientSecret ? (
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
              {isCreatingIntent ? "作成中..." : "支払方法登録を開始"}
            </button>
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