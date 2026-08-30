// frontend/amol/src/pages/PayoutAccountOnboardingPage.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";
import { loadConnectAndInitialize } from "@stripe/connect-js";
import {
  ConnectAccountOnboarding,
  ConnectComponentsProvider,
} from "@stripe/react-connect-js";
import { useLocation, useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payoutAccount-page.css";

import Layout from "../components/layout/Layout";
import {
  createPayoutAccountSession,
  fetchPayoutAccount,
} from "../features/payout/api/payoutApi";
import { fetchStripeConfig } from "../features/payment-method/api/paymentMethodApi";
import {
  getBackendUrl,
  getStripePublishableKey,
} from "../features/payment-method/utils/paymentMethodUtils";

type PayoutRegistrationReturnTarget = {
  pathname: "/resale";
  state?: unknown;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseReturnAfterRegistration(
  value: unknown,
): PayoutRegistrationReturnTarget | null {
  if (!isRecord(value)) {
    return null;
  }

  const target = value.returnAfterRegistration;

  if (!isRecord(target) || target.pathname !== "/resale") {
    return null;
  }

  return {
    pathname: "/resale",
    state: target.state,
  };
}

export default function PayoutAccountOnboardingPage() {
  const auth = getAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const [connectInstance, setConnectInstance] = useState<
    ReturnType<typeof loadConnectAndInitialize> | null
  >(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isExiting, setIsExiting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const returnAfterRegistration = useMemo(
    () => parseReturnAfterRegistration(location.state),
    [location.state],
  );

  const fetchClientSecret = useCallback(async (): Promise<string> => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      throw new Error("ログイン情報を確認できませんでした。");
    }

    const idToken = await currentUser.getIdToken(true);

    return createPayoutAccountSession({
      idToken,
    });
  }, [auth]);

  useEffect(() => {
    let cancelled = false;

    const initialize = async () => {
      const currentUser = auth.currentUser;

      if (!currentUser) {
        navigate("/signin", { replace: true });
        return;
      }

      try {
        setIsLoading(true);
        setErrorMessage("");

        const backendUrl = getBackendUrl();

        if (!backendUrl) {
          throw new Error("VITE_API_BASE_URL が設定されていません。");
        }

        const stripeConfig = await fetchStripeConfig(backendUrl);

        if (cancelled) {
          return;
        }

        const publishableKey = getStripePublishableKey(stripeConfig);

        if (!publishableKey) {
          throw new Error("Stripe 公開鍵を取得できませんでした。");
        }

        const instance = loadConnectAndInitialize({
          publishableKey,
          fetchClientSecret,
        });

        if (cancelled) {
          return;
        }

        setConnectInstance(instance);
      } catch (error) {
        if (cancelled) {
          return;
        }

        console.error(error);

        setErrorMessage(
          error instanceof Error
            ? error.message
            : "売上受取口座の登録画面を初期化できませんでした。",
        );
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    void initialize();

    return () => {
      cancelled = true;
    };
  }, [auth, fetchClientSecret, navigate]);

  const handleExit = useCallback(async () => {
    if (isExiting) {
      return;
    }

    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    try {
      setIsExiting(true);
      setErrorMessage("");

      const idToken = await currentUser.getIdToken(true);
      const payoutAccount = await fetchPayoutAccount({
        idToken,
      });

      if (
        payoutAccount?.status === "registered" &&
        payoutAccount.payoutReady
      ) {
        if (returnAfterRegistration) {
          navigate(returnAfterRegistration.pathname, {
            replace: true,
            state: returnAfterRegistration.state,
          });
          return;
        }

        navigate("/settings/payout-account/complete", {
          replace: true,
        });
        return;
      }

      navigate("/settings/payout-account", {
        replace: true,
        state: location.state,
      });
    } catch (error) {
      console.error(error);

      setErrorMessage(
        error instanceof Error
          ? error.message
          : "売上受取口座の状態確認に失敗しました。",
      );
    } finally {
      setIsExiting(false);
    }
  }, [
    auth,
    isExiting,
    location.state,
    navigate,
    returnAfterRegistration,
  ]);

  return (
    <Layout
      title="売上受取口座を登録"
      titleClickable={false}
      showBackButton
      mode="signin"
      backTo="/settings/payout-account"
    >
      <section className="page-section settings-page payout-account-page">
        <div className="payout-account-page__content">
          <p className="content-page-description payout-account-page__description">
            再販売の売上を受け取るための情報をStripeで登録します。
            本人確認や銀行口座情報はStripe上で安全に入力され、AMOLでは口座番号を保持しません。
          </p>

          {errorMessage ? (
            <p className="payout-account-page__error" role="alert">
              {errorMessage}
            </p>
          ) : null}

          {isLoading ? (
            <p className="content-page-description payout-account-page__description">
              Stripeの登録画面を準備しています...
            </p>
          ) : null}

          {isExiting ? (
            <p className="content-page-description payout-account-page__description">
              登録状態を確認しています...
            </p>
          ) : null}

          {!isLoading &&
          !isExiting &&
          !errorMessage &&
          connectInstance ? (
            <ConnectComponentsProvider connectInstance={connectInstance}>
              <ConnectAccountOnboarding onExit={handleExit} />
            </ConnectComponentsProvider>
          ) : null}
        </div>
      </section>
    </Layout>
  );
}