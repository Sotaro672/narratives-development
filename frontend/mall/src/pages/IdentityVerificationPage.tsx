// frontend/mall/src/pages/IdentityVerificationPage.tsx

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import StatePanel from "../components/ui/StatePanel";
import useIdentityVerification from "../features/identityVerification/hooks/useIdentityVerification";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/identity-verification-page.css";

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
  const {
    verification,
    isVerified,
    isLoading,
    isVerifying,
    errorMessage,
    reload,
    verify,
  } = useIdentityVerification();

  const verifiedAt = formatVerifiedAt(
    verification?.verifiedAt,
  );

  return (
    <Layout
      title="AMOL"
      titleClickable
      mode="mypage"
      showFooter
    >
      <section className="page-section content-page-section settings-page identity-verification-page">
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
                  void reload();
                }}
              >
                再読み込み
              </Button>
            }
          />
        ) : null}

        {!isLoading && verification ? (
          <>
            {isVerified ? (
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
              <>
                <StatePanel
                  variant="default"
                  title="本人確認が必要です"
                  description="取引で返品についての相談を開始するには、本人確認を完了してください。"
                />

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
                  className="identity-verification-page__verify-button"
                  disabled={isVerifying}
                  aria-busy={isVerifying}
                  onClick={() => {
                    void verify();
                  }}
                >
                  {isVerifying
                    ? "本人確認中..."
                    : "本人確認を行う"}
                </Button>
              </>
            )}
          </>
        ) : null}
      </section>
    </Layout>
  );
}