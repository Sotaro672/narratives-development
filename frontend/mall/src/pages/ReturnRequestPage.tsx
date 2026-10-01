// frontend/mall/src/pages/ReturnRequestPage.tsx

import { useCallback, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Checkbox from "../components/ui/Checkbox";
import Chip from "../components/ui/Chip";
import MediaIcon from "../components/ui/MediaIcon";
import { returnOrderItem, type ReturnPackageState } from "../features/order/api/orderDetailApi";
import { useOrderDetail } from "../features/order/hooks/useOrderDetail";
import { getFallbackInitial, getProductTitle } from "../features/order/util/orderItemDisplay";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";
import { getApiBaseUrl } from "../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../lib/authToken";

import "../styles/page-layout.css";
import "../features/shared/styles/chat-detail-page.css";
import "../styles/return-request-page.css";

function parseItemIndex(value: string | undefined): number | null {
  const normalized = value?.trim() ?? "";

  if (!/^\d+$/.test(normalized)) {
    return null;
  }

  const parsed = Number(normalized);

  return Number.isSafeInteger(parsed) && parsed >= 0
    ? parsed
    : null;
}

function getErrorMessage(caught: unknown, defaultMessage: string): string {
  return caught instanceof Error
    ? caught.message
    : defaultMessage;
}

export default function ReturnRequestPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const { itemIndex: routeItemIndex } = useParams<{ itemIndex: string }>();
  const itemIndex = useMemo(() => parseItemIndex(routeItemIndex), [routeItemIndex]);

  const {
    orderId,
    order,
    loading,
    error: orderError,
    reload,
  } = useOrderDetail();

  const [packageState, setPackageState] = useState<ReturnPackageState | null>(null);
  const [reason, setReason] = useState("");
  const [agreedToReturnConditions, setAgreedToReturnConditions] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");

  const item =
    order && itemIndex !== null
      ? order.items[itemIndex] ?? null
      : null;

  const eligibilityError = useMemo(() => {
    if (!order || itemIndex === null || !item) {
      return null;
    }

    if (item.itemType === "resale") {
      return "フリマ取引の商品はこの返品申請画面から返品できません。取引画面から返品相談を行ってください。";
    }

    if (item.isCancelled) {
      return "キャンセル済みの商品は返品できません。";
    }

    if (!item.isDispatched) {
      return "未発送の商品は返品できません。";
    }

    if (item.transferred) {
      return "受け取り済みの商品は返品できません。";
    }

    if (item.isReturnCompleted) {
      return "この商品の返品は完了しています。";
    }

    if (item.isReturnRequested) {
      return "この商品は返品申請済みです。";
    }

    return null;
  }, [item, itemIndex, order]);

  const unopenedUnavailable =
    packageState === "unopened" &&
    Boolean(item?.tokenTransferVerifiedAt);

  const normalizedReason = reason.trim();

  const returnConditionsCompleted =
    Boolean(item) &&
    !eligibilityError &&
    packageState !== null &&
    !unopenedUnavailable &&
    agreedToReturnConditions;

  const canSubmit =
    returnConditionsCompleted &&
    normalizedReason.length > 0 &&
    !submitting;

  const shouldShowMobileComposer =
    isMobilePortrait &&
    returnConditionsCompleted;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    returnConditionsCompleted;

  const handlePackageStateChange = useCallback((nextState: ReturnPackageState) => {
    if (submitting) {
      return;
    }

    setPackageState(nextState);
    setReason("");
    setSubmitError("");
    setAgreedToReturnConditions(false);
  }, [submitting]);

  const handleReasonChange = useCallback((value: string) => {
    setReason(value);
    setSubmitError("");
  }, []);

  const handleBack = useCallback(() => {
    if (submitting) {
      return;
    }

    if (orderId) {
      navigate(`/orders/${encodeURIComponent(orderId)}`);
      return;
    }

    navigate(-1);
  }, [navigate, orderId, submitting]);

  const handleSubmit = useCallback(async () => {
    if (
      !canSubmit ||
      !orderId ||
      itemIndex === null ||
      packageState === null
    ) {
      return;
    }

    setSubmitting(true);
    setSubmitError("");

    try {
      const backendUrl = getApiBaseUrl();

      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URLが設定されていません。");
      }

      const idToken = await getFirebaseIdToken();

      await returnOrderItem({
        backendUrl,
        idToken,
        orderId,
        itemIndex,
        packageState,
        reason: normalizedReason,
      });

      navigate(
        `/orders/${encodeURIComponent(orderId)}`,
        { replace: true },
      );
    } catch (caught) {
      const message = getErrorMessage(
        caught,
        "商品の返品受付に失敗しました。",
      );

      try {
        await reload();
      } catch {
        // reload側で取得エラーが管理されるため、ここでは送信エラーを優先して表示する。
      }

      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  }, [
    canSubmit,
    itemIndex,
    navigate,
    normalizedReason,
    orderId,
    packageState,
    reload,
  ]);

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content: reason,
      placeholder: "返品理由を入力してください",
      error: submitError || null,
      submitting,
      canSubmit,
      disabled: !returnConditionsCompleted,
      submitLabel: "返品を申請する",
      submittingLabel: "申請中...",
      maxLength: null,
      onContentChange: handleReasonChange,
      onSubmit: handleSubmit,
    }),
    [
      reason,
      submitError,
      submitting,
      canSubmit,
      returnConditionsCompleted,
      handleReasonChange,
      handleSubmit,
    ],
  );

  const standaloneError =
    returnConditionsCompleted
      ? orderError
      : submitError || orderError;

  return (
    <>
      <Layout
        title="返品申請"
        titleClickable={false}
        mode="mypage"
        showFooter={!isMobilePortrait}
        showBackButton
        backButtonLabel="注文詳細へ戻る"
        onBackButtonClick={handleBack}
        hideSettingsButton
        hideAnnouncementButton
      >
        <section
          className="page-section return-request-page"
          style={
            shouldShowMobileComposer
              ? {
                  paddingBottom:
                    "calc(var(--mobile-composer-height, 56px) + 24px)",
                }
              : undefined
          }
        >
          <header className="return-request-page__header">
            <p className="return-request-page__description">
              返品する商品を確認し、商品の開封状態と返品条件を確認してください。
            </p>
          </header>

          {loading ? (
            <div className="return-request-page__state">
              <p>注文情報を読み込んでいます...</p>
            </div>
          ) : null}

          {!loading && (!order || itemIndex === null || !item) ? (
            <div className="return-request-page__state">
              <Alert variant="error">
                {orderError || "返品対象の商品が見つかりません。"}
              </Alert>

              {orderId ? (
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => void reload()}
                >
                  再読み込み
                </Button>
              ) : null}
            </div>
          ) : null}

          {!loading && order && itemIndex !== null && item ? (
            <form
              className="return-request-page__form"
              onSubmit={(event) => {
                event.preventDefault();
                void handleSubmit();
              }}
            >
              <section className="return-request-page__section">
                <h2 className="return-request-page__section-title">
                  返品する商品
                </h2>

                <div className="return-request-page__product">
                  <MediaIcon
                    src={item.tokenIcon}
                    alt={item.tokenName || getProductTitle(item)}
                    fallback={getFallbackInitial(
                      item.tokenName || getProductTitle(item),
                    )}
                    size="lg"
                    shape="rounded"
                  />

                  <div className="return-request-page__product-body">
                    <strong className="return-request-page__product-name">
                      {getProductTitle(item)}
                    </strong>

                    {item.tokenName ? (
                      <span className="return-request-page__token-name">
                        {item.tokenName}
                      </span>
                    ) : null}

                    {item.brandName ? (
                      <span className="return-request-page__brand-name">
                        {item.brandName}
                      </span>
                    ) : null}
                  </div>
                </div>
              </section>

              {eligibilityError ? (
                <Alert variant="error">
                  {eligibilityError}
                </Alert>
              ) : null}

              {!eligibilityError ? (
                <>
                  <section
                    className="return-request-page__section"
                    aria-labelledby="return-request-package-state"
                  >
                    <h2
                      id="return-request-package-state"
                      className="return-request-page__section-title"
                    >
                      商品の開封状態
                    </h2>

                    <p className="return-request-page__section-description">
                      商品の包装紙は開封されていますか？
                    </p>

                    <div
                      className="return-request-page__package-options"
                      role="group"
                      aria-labelledby="return-request-package-state"
                    >
                      <Chip
                        selected={packageState === "unopened"}
                        disabled={submitting}
                        onClick={() => handlePackageStateChange("unopened")}
                      >
                        開封前
                      </Chip>

                      <Chip
                        selected={packageState === "opened"}
                        disabled={submitting}
                        onClick={() => handlePackageStateChange("opened")}
                      >
                        開封済
                      </Chip>
                    </div>
                  </section>

                  {unopenedUnavailable ? (
                    <Alert variant="error">
                      この商品は開封確認済みのため、開封前として返品を申請できません。
                    </Alert>
                  ) : null}

                  {packageState !== null && !unopenedUnavailable ? (
                    <section className="return-request-page__section">
                      <h2 className="return-request-page__section-title">
                        返品条件
                      </h2>

                      <Alert variant="warning">
                        <ol className="return-request-page__condition-list">
                          <li>返品が承認された場合、返金対象は商品代金（税込）のみです。</li>
                          <li>商品代金（税込）には、商品本体価格とその商品にかかる消費税が含まれます。</li>
                          <li>ご購入時の配送料および配送料にかかる消費税は返金対象外です。</li>
                          <li>返品商品の返送にかかる配送料はお客様のご負担となります。</li>
                          {packageState === "unopened" ? (
                            <li>返品手続き中は、商品が入っている配送用梱包材を開けないでください。</li>
                          ) : null}
                        </ol>
                      </Alert>

                      <Checkbox
                        id="return-request-agreement"
                        label="返品条件に合意する"
                        checked={agreedToReturnConditions}
                        disabled={submitting}
                        onChange={(event) => {
                          setAgreedToReturnConditions(event.target.checked);
                          setSubmitError("");
                        }}
                      />
                    </section>
                  ) : null}

                  {shouldShowDesktopComposer ? (
                    <section className="return-request-page__section">
                      <h2 className="return-request-page__section-title">
                        返品理由
                      </h2>

                      <ChatInlineComposer {...composer} />
                    </section>
                  ) : null}

                  {standaloneError ? (
                    <Alert variant="error">
                      {standaloneError}
                    </Alert>
                  ) : null}
                </>
              ) : null}
            </form>
          ) : null}
        </section>
      </Layout>

      {shouldShowMobileComposer ? (
        <MobileComposerFooter {...composer} />
      ) : null}
    </>
  );
}