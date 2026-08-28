// frontend/amol/src/features/payout/hooks/usePayoutAccountPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";
import { useNavigate } from "react-router-dom";

import { fetchPayoutAccount } from "../api/payoutApi";
import type { PayoutAccount } from "../../shared/types/payoutAccount";

export function usePayoutAccountPage() {
  const auth = getAuth();
  const navigate = useNavigate();

  const [payoutAccount, setPayoutAccount] = useState<PayoutAccount | null>(null);
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

  useEffect(() => {
    void loadPayoutAccount();
  }, [loadPayoutAccount]);

  const handleOpenRegistration = useCallback(() => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    setErrorMessage("");
    navigate("/settings/payout-account/bank");
  }, [auth, navigate]);

  const statusLabel = useMemo(() => {
    if (isLoading) {
      return "確認中";
    }

    if (!payoutAccount || payoutAccount.status === "unregistered") {
      return "未登録";
    }

    switch (payoutAccount.status) {
      case "pending":
        return "確認中";
      case "registered":
        return "登録済み";
      case "restricted":
        return "利用制限中";
      default:
        return "未登録";
    }
  }, [payoutAccount, isLoading]);

  const actionLabel = useMemo(() => {
    if (!payoutAccount || payoutAccount.status === "unregistered") {
      return "口座を登録する";
    }

    return "口座を変更する";
  }, [payoutAccount]);

  const bankName = payoutAccount?.bankAccount?.bankName || "";
  const bankLast4 = payoutAccount?.bankAccount?.last4 || "";

  return {
    payoutAccount,
    isLoading,
    errorMessage,
    statusLabel,
    actionLabel,
    bankName,
    bankLast4,
    handleOpenRegistration,
    reloadPayoutAccount: loadPayoutAccount,
  };
}