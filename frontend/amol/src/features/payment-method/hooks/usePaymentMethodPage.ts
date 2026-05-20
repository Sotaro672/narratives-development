//frontend\amol\src\features\payment-method\hooks\usePaymentMethodPage.ts
import { useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";
import { loadStripe } from "@stripe/stripe-js";
import type { Stripe } from "@stripe/stripe-js";
import { useLocation, useNavigate } from "react-router-dom";

import {
  createSetupIntent,
  fetchCurrentPaymentMethod,
  fetchStripeConfig,
  savePaymentMethod,
} from "../api/paymentMethodApi";
import type {
  CardPaymentMethod,
  ConfirmedCardPayload,
  PaymentMethodPageLocationState,
  UsePaymentMethodPageResult,
} from "../types";
import {
  extractSetupIntentClientSecret,
  extractSetupIntentStripeCustomerId,
  getBackendUrl,
  getStripePublishableKey,
} from "../utils/paymentMethodUtils";

export default function usePaymentMethodPage(): UsePaymentMethodPageResult {
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

        const stripeConfigBody = await fetchStripeConfig(backendUrl);
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

      const responseBody = await createSetupIntent(
        backendUrl,
        idToken,
        normalizedCardholderName,
      );

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

  return {
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
  };
}