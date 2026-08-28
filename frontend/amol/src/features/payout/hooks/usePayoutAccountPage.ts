// frontend/amol/src/features/payout/hooks/usePayoutAccountPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";
import { useNavigate, useSearchParams } from "react-router-dom";

import {
  createPayoutAccountLink,
  fetchPayoutAccount,
} from "../api/payoutApi";
import {
  getPayoutAccountActionLabel,
  getPayoutAccountStatusLabel,
} from "../utils/payoutAccountUtils";

import type { PayoutAccount } from "../../shared/types/payoutAccount";

export function usePayoutAccountPage() {
  const auth = getAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [payoutAccount, setPayoutAccount] =
    useState<PayoutAccount | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isOpeningStripe, setIsOpeningStripe] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const stripeReturnState = searchParams.get("stripe");

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

  useEffect(() => {
    void loadPayoutAccount();
  }, [loadPayoutAccount]);

  useEffect(() => {
    if (!stripeReturnState) {
      return;
    }

    if (stripeReturnState === "return") {
      void loadPayoutAccount();
    }

    const nextSearchParams = new URLSearchParams(searchParams);
    nextSearchParams.delete("stripe");

    setSearchParams(nextSearchParams, { replace: true });
  }, [
    loadPayoutAccount,
    searchParams,
    setSearchParams,
    stripeReturnState,
  ]);

  const handleOpenStripe = useCallback(async () => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    try {
      setIsOpeningStripe(true);
      setErrorMessage("");

      const idToken = await currentUser.getIdToken(true);
      const origin = window.location.origin;

      const returnUrl =
        `${origin}/settings/payout-account?stripe=return`;
      const refreshUrl =
        `${origin}/settings/payout-account?stripe=refresh`;

      const url = await createPayoutAccountLink({
        idToken,
        returnUrl,
        refreshUrl,
      });

      window.location.assign(url);
    } catch (error) {
      console.error(error);

      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Stripeとの接続に失敗しました。"
      );

      setIsOpeningStripe(false);
    }
  }, [auth, navigate]);

  const statusLabel = useMemo(
    () => getPayoutAccountStatusLabel(payoutAccount, isLoading),
    [payoutAccount, isLoading]
  );

  const actionLabel = useMemo(
    () => getPayoutAccountActionLabel(payoutAccount, isOpeningStripe),
    [payoutAccount, isOpeningStripe]
  );

  const bankName = payoutAccount?.bankAccount?.bankName || "";
  const bankLast4 = payoutAccount?.bankAccount?.last4 || "";

  return {
    payoutAccount,
    isLoading,
    isOpeningStripe,
    errorMessage,
    stripeReturnState,
    statusLabel,
    actionLabel,
    bankName,
    bankLast4,
    handleOpenStripe,
    reloadPayoutAccount: loadPayoutAccount,
  };
}