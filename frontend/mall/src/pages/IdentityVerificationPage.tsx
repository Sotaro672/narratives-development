// frontend/mall/src/pages/IdentityVerificationPage.tsx

import { useCallback, useEffect, useState } from "react";

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import StatePanel from "../components/ui/StatePanel";
import {
  fetchIdentityVerification,
  verifyIdentityWithMockMyNumberCard,
} from "../features/identityVerification/api/identityVerificationApi";
import {
  isIdentityVerified,
  type IdentityVerification,
} from "../features/identityVerification/types";

import "../styles/page-layout.css";
import "../styles/settings-page.css";

function toErrorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : "本人確認情報の取得に失敗しました。";
}

function formatVerifiedAt(value: string | undefined): string | null {
  if (!value) {
    return null;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export default function IdentityVerificationPage() {
  const [verification, setVerification] =
    useState<IdentityVerification | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isVerifying, setIsVerifying] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  const loadVerification = useCallback(
    async (signal?: AbortSignal): Promise<void> => {
      setIsLoading(true);
      setErrorMessage("");

      try {
        const result = await fetchIdentityVerification({
          signal,
        });

        if (signal?.aborted) {
          return;
        }

        setVerification(result);
      } catch (error) {
        if (signal?.aborted) {
          return;
        }

        setErrorMessage(toErrorMessage(error));
      } finally {
        if (!signal?.aborted) {
          setIsLoading(false);
        }
      }
    },
    [],
  );

  useEffect(() => {
    const controller = new AbortController();

    void loadVerification(controller.signal);

    return () => {
      controller.abort();
    };
  }, [loadVerification]);

  const handleVerify = useCallback(async (): Promise<void> => {
    if (isVerifying || isIdentityVerified(verification)) {
      return;
    }

    setIsVerifying(true);
    setErrorMessage("");
    setSuccessMessage("");

    try {
      const result =
        await verifyIdentityWithMockMyNumberCard();

      setVerification(result);
      setSuccessMessage("本人確認が完了しました。");
    } catch (error) {
      setErrorMessage(toErrorMessage(error));
    } finally {
      setIsVerifying(false);
    }
  }, [isVerifying, verification]);

  const verified = isIdentityVerified(verification);
  const verifiedAt = formatVerifiedAt(
    verification?.verifiedAt,
  );

  return (
    <Layout
      title="本人確認"
      titleClickable={false}
      mode="mypage"
      showFooter
    >
      <section className="page-section content-page-section settings-page">
        <p className="content-page-description">
          マイナンバーカードを使用して本人確認を行います。本人確認が完了すると、取引で返品についての相談を開始できるようになります。
        </p>

        {isLoading ? (
          <StatePanel
            variant="loading"
            title="本人確認情報を読み込んでいます"
          />
        ) : null}

        {!isLoading && errorMessage ? (
          <Alert variant="error">
            {errorMessage}
          </Alert>
        ) : null}

        {!isLoading && !verification && errorMessage ? (
          <StatePanel
            variant="error"
            title="本人確認情報を取得できませんでした"
            description="通信状態を確認して、もう一度お試しください。"
            action={
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  void loadVerification();
                }}
              >
                再読み込み
              </Button>
            }
          />
        ) : null}

        {!isLoading && verification ? (
          <>
            {verified ? (
              <StatePanel
                variant="success"
                title="本人確認済み"
                description={
                  verifiedAt
                    ? `確認日時: ${verifiedAt}`
                    : "本人確認が完了しています。"
                }
              />
            ) : (
              <StatePanel
                variant="default"
                title="本人確認が必要です"
                description="取引で返品についての相談を開始するには、本人確認を完了してください。"
              />
            )}

            {successMessage ? (
              <Alert variant="success">
                {successMessage}
              </Alert>
            ) : null}

            {!verified ? (
              <>
                <Alert variant="info">
                  <strong>本人確認方法</strong>
                  <br />
                  マイナンバーカード
                </Alert>

                <Alert variant="warning">
                  現在は開発中のため、実際のマイナンバーカード情報は取得・保存しません。「本人確認を行う」を押すと、モックデータを使用して本人確認済みの状態を登録します。
                </Alert>

                <Button
                  type="button"
                  variant="primary"
                  size="lg"
                  fullWidth
                  disabled={isVerifying}
                  aria-busy={isVerifying}
                  onClick={() => {
                    void handleVerify();
                  }}
                >
                  {isVerifying
                    ? "本人確認中..."
                    : "本人確認を行う"}
                </Button>
              </>
            ) : (
              <Alert variant="info">
                本人確認が完了しているため、取引で返品についての相談を開始できます。
              </Alert>
            )}
          </>
        ) : null}
      </section>
    </Layout>
  );
}