// frontend/amol/src/features/payout/hooks/usePayoutAccountPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";
import { useNavigate } from "react-router-dom";

import {
  createPayoutAccountSession,
  fetchPayoutAccount,
} from "../api/payoutApi";
import {
  getPayoutAccountActionLabel,
  getPayoutAccountStatusLabel,
} from "../utils/payoutAccountUtils";

import { fetchStripeConfig } from "../../payment-method/api/paymentMethodApi";
import {
  getBackendUrl,
  getStripePublishableKey,
} from "../../payment-method/utils/paymentMethodUtils";

import type { PayoutAccount } from "../../shared/types/payoutAccount";

export type PayoutAccountConnectMode =
  | "onboarding"
  | "management";

export function usePayoutAccountPage() {
  const auth = getAuth();
  const navigate = useNavigate();

  const backendUrl = useMemo(() => getBackendUrl(), []);

  const [payoutAccount, setPayoutAccount] =
    useState<PayoutAccount | null>(null);
  const [stripePublishableKey, setStripePublishableKey] = useState("");
  const [showConnectPanel, setShowConnectPanel] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const loadPayoutAccount = useCallback(async () => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    try {
      setIsLoading(true);
      setErrorMessage("");

      const idToken = await currentUser.getIdToken(true);
      const account = await fetchPayoutAccount({ idToken });

      setPayoutAccount(account);
    } catch (error) {
      console.error(error);

      setErrorMessage(
        error instanceof Error
          ? error.message
          : "売上受取口座の情報取得に失敗しました。"
      );
    } finally {
      setIsLoading(false);
    }
  }, [auth, navigate]);

  const loadStripeConfig = useCallback(async () => {
    try {
      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URL が設定されていません。");
      }

      const response = await fetchStripeConfig(backendUrl);
      const publishableKey = getStripePublishableKey(response);

      if (!publishableKey) {
        throw new Error("Stripe 公開鍵を取得できませんでした。");
      }

      setStripePublishableKey(publishableKey);
    } catch (error) {
      console.error(error);

      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Stripeの初期化情報取得に失敗しました。"
      );
    }
  }, [backendUrl]);

  useEffect(() => {
    void loadPayoutAccount();
    void loadStripeConfig();
  }, [loadPayoutAccount, loadStripeConfig]);

  const fetchClientSecret = useCallback(async (): Promise<string> => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      throw new Error("ログイン情報を確認できませんでした。");
    }

    try {
      setErrorMessage("");

      const idToken = await currentUser.getIdToken(true);

      return await createPayoutAccountSession({
        idToken,
      });
    } catch (error) {
      console.error(error);

      const message =
        error instanceof Error
          ? error.message
          : "StripeのAccount Session作成に失敗しました。";

      setErrorMessage(message);

      throw error instanceof Error
        ? error
        : new Error(message);
    }
  }, [auth, navigate]);

  const connectMode = useMemo<PayoutAccountConnectMode>(() => {
    if (payoutAccount?.payoutsEnabled) {
      return "management";
    }

    return "onboarding";
  }, [payoutAccount]);

  const handleOpenStripe = useCallback(() => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    if (!stripePublishableKey) {
      setErrorMessage(
        "Stripeの初期化が完了していません。少し待ってからもう一度お試しください。"
      );
      return;
    }

    setErrorMessage("");
    setShowConnectPanel(true);
  }, [auth, navigate, stripePublishableKey]);

  const handleConnectExit = useCallback(async () => {
    setShowConnectPanel(false);
    await loadPayoutAccount();
  }, [loadPayoutAccount]);

  const handleConnectError = useCallback((error: unknown) => {
    console.error(error);

    setErrorMessage(
      error instanceof Error
        ? error.message
        : "Stripeの口座登録画面を読み込めませんでした。"
    );
  }, []);

  const statusLabel = useMemo(
    () => getPayoutAccountStatusLabel(payoutAccount, isLoading),
    [payoutAccount, isLoading]
  );

  const actionLabel = useMemo(
    () => getPayoutAccountActionLabel(payoutAccount),
    [payoutAccount]
  );

  const bankName = payoutAccount?.bankAccount?.bankName || "";
  const bankLast4 = payoutAccount?.bankAccount?.last4 || "";

  return {
    payoutAccount,
    stripePublishableKey,
    showConnectPanel,
    connectMode,
    isLoading,
    errorMessage,
    statusLabel,
    actionLabel,
    bankName,
    bankLast4,
    fetchClientSecret,
    handleOpenStripe,
    handleConnectExit,
    handleConnectError,
    reloadPayoutAccount: loadPayoutAccount,
  };
}