// frontend/console/shell/src/features/account/presentation/hook/useAccountConnect.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";
import { accountRepositoryHTTP } from "../../infrastructure/http/accountRepositoryHTTP";

function getErrorMessage(error: unknown): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  return "Stripe口座との接続に失敗しました。";
}

export function useAccountConnect() {
  const navigate = useNavigate();

  const [
    contactEmail,
    setContactEmail,
  ] = useState(
    auth.currentUser?.email ?? "",
  );

  const [
    submitting,
    setSubmitting,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<string | null>(null);

  const [
    completed,
    setCompleted,
  ] = useState(false);

  useEffect(() => {
    const params =
      new URLSearchParams(
        window.location.search,
      );

    setCompleted(
      params.get("completed") === "1",
    );
  }, []);

  const canConnect = useMemo(
    () =>
      !submitting &&
      Boolean(
        contactEmail.trim(),
      ),
    [
      submitting,
      contactEmail,
    ],
  );

  const handleContactEmailChange =
    useCallback(
      (value: string) => {
        setContactEmail(value);

        setError(null);
      },
      [],
    );

  const handleAccountManagement =
    useCallback(() => {
      navigate("/account");
    }, [navigate]);

  const handleConnect =
    useCallback(async () => {
      if (submitting) {
        return;
      }

      const normalizedEmail =
        contactEmail.trim();

      if (!normalizedEmail) {
        setError(
          "Stripe口座に使用するメールアドレスを入力してください。",
        );
        return;
      }

      try {
        setSubmitting(true);
        setError(null);

        const origin =
          window.location.origin;

        const response =
          await accountRepositoryHTTP.connect({
            contactEmail:
              normalizedEmail,
            country:
              "JP",
            returnUrl:
              `${origin}/account/connect?completed=1`,
            refreshUrl:
              `${origin}/account/connect`,
          });

        const onboardingUrl =
          response?.onboardingUrl?.trim();

        if (!onboardingUrl) {
          throw new Error(
            "Stripe onboarding URLを取得できませんでした。",
          );
        }

        window.location.assign(
          onboardingUrl,
        );
      } catch (
        caughtError: unknown
      ) {
        setError(
          getErrorMessage(
            caughtError,
          ),
        );
      } finally {
        setSubmitting(false);
      }
    }, [
      contactEmail,
      submitting,
    ]);

  return {
    contactEmail,
    submitting,
    error,
    completed,
    canConnect,

    handleContactEmailChange,
    handleConnect,
    handleAccountManagement,
  };
}