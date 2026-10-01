// frontend/mall/src/pages/ProductDescriptionInquiryPage.tsx

import { useMemo } from "react";
import { ChevronLeft } from "lucide-react";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Checkbox from "../components/ui/Checkbox";
import IconButton from "../components/ui/IconButton";
import StatePanel from "../components/ui/StatePanel";
import { useProductDescriptionInquiryPage } from "../features/inquiry/presentation/hooks/useProductDescriptionInquiryPage";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";

import "../styles/page-layout.css";
import "../features/shared/styles/chat-detail-page.css";
import "../styles/inquiry-page.css";

export default function ProductDescriptionInquiryPage() {
  const isMobilePortrait = useMobilePortrait();

  const {
    navigate,
    productId,
    productViewModel,
    productLoading,
    productError,
    privacyPolicy,
    privacyLoading,
    privacyError,
    agreedToPrivacyPolicy,
    setAgreedToPrivacyPolicy,
    content,
    setContent,
    files,
    submitting,
    error,
    canSubmit,
    submitInquiry,
    handleFilesAdd,
    handleRemoveFile,
    handleBackToScanResult,
  } = useProductDescriptionInquiryPage();

  const product = productViewModel?.product ?? null;

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content,
      placeholder: "商品について確認したい内容を入力",
      files,
      error,
      submitting,
      canSubmit,
      disabled: !productId || !product,
      submitLabel: "送信",
      submittingLabel: "送信中",
      maxLength: 2000,
      maxFiles: 10,
      accept: "image/*",
      onContentChange: setContent,
      onFilesAdd: handleFilesAdd,
      onRemoveFile: handleRemoveFile,
      onSubmit: submitInquiry,
    }),
    [
      content,
      files,
      error,
      submitting,
      canSubmit,
      productId,
      product,
      setContent,
      handleFilesAdd,
      handleRemoveFile,
      submitInquiry,
    ],
  );

  const shouldShowMobileComposer =
    isMobilePortrait &&
    Boolean(productId) &&
    agreedToPrivacyPolicy;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    Boolean(productId) &&
    agreedToPrivacyPolicy;

  const pageLayoutClassName = [
    "chat-detail-page-layout",
    "chat-detail-page-layout--inquiry",
    "inquiry-create-page-layout",
    shouldShowMobileComposer
      ? "inquiry-create-page-layout--with-mobile-composer"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <>
      <Layout
        title="商品説明"
        mode="mypage"
        showHeader={!isMobilePortrait}
        showFooter={false}
        mainClassName="inquiry-page"
        disableFooterPaddingOnDesktop
      >
        <div
          className={pageLayoutClassName}
          data-chat-detail
        >
          <section className="product-detail-page-layout chat-detail-page">
            {!productId ? (
              <Alert
                variant="error"
                className="chat-detail-page__error"
              >
                <p>商品IDが見つかりませんでした。</p>

                <Button
                  variant="secondary"
                  size="md"
                  className="inquiry-page__alert-action"
                  onClick={() => navigate("/scan/result")}
                >
                  スキャン結果へ戻る
                </Button>
              </Alert>
            ) : null}

            {productId ? (
              <div className="chat-detail-page__split inquiry-create-page__split">
                <div className="chat-detail-page__left inquiry-create-page__product-column">
                  {productLoading ? (
                    <StatePanel
                      variant="loading"
                      title="商品情報を読み込み中..."
                    />
                  ) : null}

                  {!productLoading && productError ? (
                    <Alert
                      variant="error"
                      className="inquiry-create-page__product-error"
                    >
                      {productError}
                    </Alert>
                  ) : null}

                  {!productLoading && !productError && product ? (
                    <div className="inquiry-create-page__product">
                      <div className="inquiry-create-page__product-heading">
                        <IconButton
                          type="button"
                          variant="ghost"
                          size="md"
                          className="inquiry-create-page__product-back"
                          aria-label="スキャン結果画面へ戻る"
                          onClick={handleBackToScanResult}
                        >
                          <ChevronLeft size={22} aria-hidden="true" />
                        </IconButton>

                        <h2 className="chat-detail-page__subject">
                          {product.title || "商品詳細"}
                        </h2>
                      </div>

                      <dl className="inquiry-create-page__product-meta">
                        <div className="inquiry-create-page__product-meta-row">
                          <dt>ブランド</dt>
                          <dd>{product.brandName || "-"}</dd>
                        </div>

                        <div className="inquiry-create-page__product-meta-row">
                          <dt>型式</dt>
                          <dd>{product.modelNumber || "-"}</dd>
                        </div>

                        <div className="inquiry-create-page__product-meta-row">
                          <dt>サイズ</dt>
                          <dd>{product.size || "-"}</dd>
                        </div>

                        <div className="inquiry-create-page__product-meta-row">
                          <dt>カラー</dt>
                          <dd>
                            {product.color ? (
                              <span className="inquiry-create-page__color">
                                {product.swatch ? (
                                  <span
                                    className="inquiry-create-page__color-swatch"
                                    style={{ backgroundColor: product.swatch }}
                                    aria-hidden="true"
                                  />
                                ) : null}
                                <span>{product.color}</span>
                              </span>
                            ) : (
                              "-"
                            )}
                          </dd>
                        </div>
                      </dl>
                    </div>
                  ) : null}

                  {!productLoading && !productError && !product ? (
                    <StatePanel
                      variant="empty"
                      title="商品情報を取得できませんでした。"
                    />
                  ) : null}
                </div>

                <div className="chat-detail-page__right inquiry-create-page__right">
                  <div className="inquiry-create-page__mobile-product-heading">
                    <IconButton
                      type="button"
                      variant="ghost"
                      size="md"
                      className="inquiry-create-page__mobile-back"
                      aria-label="スキャン結果画面へ戻る"
                      onClick={handleBackToScanResult}
                    >
                      <ChevronLeft size={24} aria-hidden="true" />
                    </IconButton>

                    <span className="inquiry-create-page__mobile-product-name">
                      商品説明
                    </span>
                  </div>

                  <div className="inquiry-create-page__policy">
                    <div className="inquiry-create-page__policy-scroll">
                      <header className="inquiry-create-page__policy-header">
                        <h2 className="inquiry-create-page__policy-title">
                          プライバシーポリシー
                        </h2>

                        <p className="inquiry-create-page__policy-description">
                          商品について問い合わせる前に、以下の内容をご確認ください。
                        </p>
                      </header>

                      {privacyLoading ? (
                        <StatePanel
                          variant="loading"
                          title="プライバシーポリシーを読み込み中..."
                        />
                      ) : null}

                      {!privacyLoading && privacyError ? (
                        <Alert
                          variant="error"
                          className="inquiry-create-page__policy-error"
                        >
                          {privacyError}
                        </Alert>
                      ) : null}

                      {!privacyLoading && !privacyError ? (
                        <pre className="inquiry-create-page__policy-content">
                          {privacyPolicy}
                        </pre>
                      ) : null}
                    </div>

                    <div className="inquiry-create-page__agreement">
                      <Checkbox
                        id="product-description-privacy-agreement"
                        label="プライバシーポリシーに同意する"
                        checked={agreedToPrivacyPolicy}
                        disabled={
                          privacyLoading ||
                          Boolean(privacyError) ||
                          !privacyPolicy
                        }
                        onChange={(event) => {
                          setAgreedToPrivacyPolicy(
                            event.currentTarget.checked,
                          );
                        }}
                      />
                    </div>
                  </div>

                  {error ? (
                    <Alert
                      variant="error"
                      className="inquiry-create-page__submit-error"
                    >
                      {error}
                    </Alert>
                  ) : null}

                  {shouldShowDesktopComposer ? (
                    <ChatInlineComposer {...composer} />
                  ) : null}
                </div>
              </div>
            ) : null}
          </section>
        </div>
      </Layout>

      {shouldShowMobileComposer ? (
        <MobileComposerFooter {...composer} />
      ) : null}
    </>
  );
}