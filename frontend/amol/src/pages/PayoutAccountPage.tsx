// frontend/amol/src/pages/PayoutAccountPage.tsx

import { useCallback, useEffect, useState } from "react";
import { getAuth } from "firebase/auth";
import { useNavigate, useSearchParams } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payoutAccount-page.css";

import Layout from "../components/layout/Layout";

import {
  createPayoutAccountLink,
  fetchPayoutAccount,
} from "../features/payout/api/payoutApi";

import type {
  PayoutAccount,
} from "../features/shared/types/payoutAccount";

function getStatusLabel(
  payoutAccount: PayoutAccount | null,
  isLoading: boolean
): string {
  if (isLoading) {
    return "確認中...";
  }

  if (!payoutAccount) {
    return "未登録";
  }

  if (payoutAccount.payoutsEnabled) {
    return "登録済み";
  }

  if (payoutAccount.detailsSubmitted) {
    return "確認中";
  }

  return "登録未完了";
}

export default function PayoutAccountPage() {
  const auth = getAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] =
    useSearchParams();

  const [payoutAccount, setPayoutAccount] =
    useState<PayoutAccount | null>(null);

  const [isLoading, setIsLoading] =
    useState(true);

  const [
    isOpeningStripe,
    setIsOpeningStripe,
  ] = useState(false);

  const [errorMessage, setErrorMessage] =
    useState("");

  const stripeReturnState =
    searchParams.get("stripe");

  const loadPayoutAccount =
    useCallback(async () => {
      const currentUser =
        auth.currentUser;

      if (!currentUser) {
        navigate("/signin", {
          replace: true,
        });
        return;
      }

      try {
        setIsLoading(true);
        setErrorMessage("");

        const idToken =
          await currentUser.getIdToken(true);

        const account =
          await fetchPayoutAccount({
            idToken,
          });

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

    const nextSearchParams =
      new URLSearchParams(
        searchParams
      );

    nextSearchParams.delete(
      "stripe"
    );

    setSearchParams(
      nextSearchParams,
      {
        replace: true,
      }
    );
  }, [
    loadPayoutAccount,
    searchParams,
    setSearchParams,
    stripeReturnState,
  ]);

  const handleOpenStripe =
    async () => {
      const currentUser =
        auth.currentUser;

      if (!currentUser) {
        navigate("/signin", {
          replace: true,
        });
        return;
      }

      try {
        setIsOpeningStripe(true);
        setErrorMessage("");

        const idToken =
          await currentUser.getIdToken(
            true
          );

        const origin =
          window.location.origin;

        const returnUrl =
          `${origin}/settings/payout-account?stripe=return`;

        const refreshUrl =
          `${origin}/settings/payout-account?stripe=refresh`;

        const url =
          await createPayoutAccountLink({
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
    };

  const statusLabel =
    getStatusLabel(
      payoutAccount,
      isLoading
    );

  const actionLabel =
    isOpeningStripe
      ? "Stripeへ接続中..."
      : payoutAccount
        ? "口座情報を変更"
        : "売上受取口座を登録";

  const bankName =
    payoutAccount
      ?.bankAccount
      ?.bankName || "";

  const bankLast4 =
    payoutAccount
      ?.bankAccount
      ?.last4 || "";

  return (
    <Layout
      title="売上受取口座"
      titleClickable={false}
      showBackButton
      mode="signin"
      backTo="/lists"
    >
      <section className="page-section settings-page payout-account-page">
        <div className="payout-account-page__content">
          <p className="content-page-description payout-account-page__description">
            再販売で発生した売上の受取口座を登録します。
            口座情報の登録・変更はStripeの安全な画面で行います。
          </p>

          <div className="payout-account-page__status-card">
            <div className="payout-account-page__status-row">
              <span className="payout-account-page__label">
                登録状況
              </span>

              <strong className="payout-account-page__status">
                {statusLabel}
              </strong>
            </div>

            {bankName ||
            bankLast4 ? (
              <div className="payout-account-page__bank-account">
                {bankName ? (
                  <div className="payout-account-page__bank-row">
                    <span className="payout-account-page__label">
                      金融機関
                    </span>

                    <span className="payout-account-page__bank-value">
                      {bankName}
                    </span>
                  </div>
                ) : null}

                {bankLast4 ? (
                  <div className="payout-account-page__bank-row">
                    <span className="payout-account-page__label">
                      口座番号
                    </span>

                    <span className="payout-account-page__bank-value">
                      •••• {bankLast4}
                    </span>
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>

          {payoutAccount &&
          !payoutAccount
            .payoutsEnabled ? (
            <div className="payout-account-page__notice">
              <p className="payout-account-page__notice-text">
                Stripeで追加情報の登録または確認が必要です。
                登録を完了すると、再販売の売上を受け取れるようになります。
              </p>
            </div>
          ) : null}

          {stripeReturnState ===
          "refresh" ? (
            <div className="payout-account-page__notice">
              <p className="payout-account-page__notice-text">
                Stripeの登録リンクの有効期限が切れました。
                下のボタンからもう一度お進みください。
              </p>
            </div>
          ) : null}

          {errorMessage ? (
            <p className="payout-account-page__error">
              {errorMessage}
            </p>
          ) : null}

          <button
            type="button"
            onClick={
              handleOpenStripe
            }
            disabled={
              isLoading ||
              isOpeningStripe
            }
            className="payout-account-page__action-button"
          >
            {actionLabel}
          </button>

          <p className="payout-account-page__note">
            銀行口座番号などの口座情報はStripe上で管理されます。
            AMOLでは口座番号の全桁を保存しません。
          </p>
        </div>
      </section>
    </Layout>
  );
}