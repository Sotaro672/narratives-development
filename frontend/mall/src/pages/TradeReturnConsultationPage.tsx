// frontend/mall/src/pages/TradeReturnConsultationPage.tsx

import {
  useCallback,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeRightDismissPage from "../components/layout/MobileSwipeRightDismissPage";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Chip from "../components/ui/Chip";
import StatePanel from "../components/ui/StatePanel";
import useIdentityVerification from "../features/identityVerification/hooks/useIdentityVerification";
import {
  TRADE_RETURN_CONSULTATION_REASONS,
  type TradeReturnConsultationReason,
} from "../features/shared/types/trade";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";
import { createTradeReturnConsultation } from "../features/trade/infrastructure/tradeApi";
import useTradeThread from "../features/trade/presentation/hooks/useTradeThread";
import { getErrorMessage } from "../features/trade/presentation/util/tradeChatDetail";

import "../styles/page-layout.css";
import "../styles/trade-return-consultation-page.css";

type TradeReturnConsultationRouteParams = {
  tradeId?: string;
};

const MAX_DETAIL_LENGTH = 5000;

function getReasonLabel(
  reason: TradeReturnConsultationReason,
): string {
  switch (reason) {
    case "not_as_described":
      return "商品説明と状態が異なる";
    case "damaged":
      return "商品が破損している";
    case "wrong_item":
      return "異なる商品が届いた";
    case "other":
      return "その他";
  }
}

export default function TradeReturnConsultationPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const { tradeId: routeTradeId } = useParams<TradeReturnConsultationRouteParams>();
  const normalizedTradeId = routeTradeId?.trim() ?? "";

  const thread = useTradeThread(normalizedTradeId);

  const identityVerification = useIdentityVerification({
    enabled: thread.trade?.viewerSide === "buyer",
  });

  const [reason, setReason] =
    useState<TradeReturnConsultationReason | null>(null);
  const [detail, setDetail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submissionError, setSubmissionError] = useState("");

  const chatPath = normalizedTradeId
    ? `/chats/trades/${encodeURIComponent(normalizedTradeId)}`
    : "/chats";

  const eligibilityError = useMemo(() => {
    const trade = thread.trade;

    if (!trade) {
      return "";
    }

    if (trade.viewerSide !== "buyer") {
      return "返品について相談できるのは購入者のみです。";
    }

    if (trade.status !== "active" || trade.isCancelled) {
      return "この取引では返品について相談できません。";
    }

    if (!trade.isDispatched) {
      return "商品が発送されるまで返品について相談できません。";
    }

    if (trade.transferred) {
      return "受け取り済みの商品については返品相談を開始できません。";
    }

    if (trade.returnStatus !== "none") {
      return "この取引ではすでに返品手続きが開始されています。";
    }

    return "";
  }, [thread.trade]);

  const normalizedDetail = detail.trim();

  const canEdit =
    Boolean(thread.trade) &&
    !thread.loading &&
    !eligibilityError &&
    !identityVerification.isLoading &&
    identityVerification.isVerified &&
    !submitting;

  const canSubmit =
    canEdit &&
    reason !== null &&
    normalizedDetail.length > 0 &&
    normalizedDetail.length <= MAX_DETAIL_LENGTH;

  const shouldShowMobileComposer =
    isMobilePortrait &&
    Boolean(thread.trade) &&
    !thread.loading &&
    !eligibilityError &&
    identityVerification.isVerified;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    Boolean(thread.trade) &&
    !thread.loading &&
    !eligibilityError &&
    identityVerification.isVerified;

  const handleDismiss = useCallback(() => {
    if (submitting) {
      return;
    }

    navigate(chatPath, { replace: true });
  }, [
    chatPath,
    navigate,
    submitting,
  ]);

  const handleReasonChange = useCallback(
    (value: TradeReturnConsultationReason) => {
      if (submitting) {
        return;
      }

      setReason(value);
      setSubmissionError("");
    },
    [submitting],
  );

  const handleDetailChange = useCallback((value: string) => {
    setDetail(value);
    setSubmissionError("");
  }, []);

  const handleSubmit = useCallback(async (): Promise<void> => {
    const trade = thread.trade;

    if (
      submitting ||
      !canSubmit ||
      !trade ||
      !normalizedTradeId ||
      !reason
    ) {
      return;
    }

    if (
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.returnStatus !== "none"
    ) {
      setSubmissionError(
        "取引状態が変更されたため、返品について相談できません。",
      );
      return;
    }

    setSubmitting(true);
    setSubmissionError("");

    try {
      await createTradeReturnConsultation({
        tradeId: normalizedTradeId,
        reason,
        detail: normalizedDetail,
      });

      navigate(chatPath, {
        replace: true,
      });
    } catch (caught) {
      setSubmissionError(
        getErrorMessage(
          caught,
          "返品についての相談を開始できませんでした。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    canSubmit,
    chatPath,
    navigate,
    normalizedDetail,
    normalizedTradeId,
    reason,
    submitting,
    thread.trade,
  ]);

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content: detail,
      placeholder: "商品の状態や問題点を入力",
      error: submissionError || null,
      submitting,
      canSubmit,
      disabled: !canEdit,
      submitLabel: "送信",
      submittingLabel: "送信中...",
      maxLength: MAX_DETAIL_LENGTH,
      onContentChange: handleDetailChange,
      onSubmit: handleSubmit,
    }),
    [
      canEdit,
      canSubmit,
      detail,
      handleDetailChange,
      handleSubmit,
      submissionError,
      submitting,
    ],
  );

  const pageContent = (
    <Layout
      title="返品について相談する"
      titleClickable={false}
      mode="mypage"
      showHeader={!isMobilePortrait}
      showFooter={false}
      showBackButton={!isMobilePortrait}
      backButtonLabel="取引へ戻る"
      onBackButtonClick={handleDismiss}
      hideSettingsButton
      hideAnnouncementButton
    >
      <section
        className="page-section trade-return-consultation-page"
        style={
          shouldShowMobileComposer
            ? {
                paddingBottom:
                  "calc(var(--mobile-composer-height, 56px) + 24px)",
              }
            : undefined
        }
      >

        {!normalizedTradeId ? (
          <Alert variant="error">
            取引IDが見つかりません。
          </Alert>
        ) : null}

        {normalizedTradeId && thread.loading ? (
          <StatePanel
            variant="loading"
            title="取引情報を読み込み中..."
          />
        ) : null}

        {!thread.loading && thread.error && !thread.trade ? (
          <div className="trade-return-consultation-page__state">
            <Alert variant="error">
              {thread.error}
            </Alert>

            <Button
              type="button"
              variant="secondary"
              disabled={submitting}
              onClick={() => {
                void thread.reload();
              }}
            >
              再読み込み
            </Button>
          </div>
        ) : null}

        {!thread.loading && thread.trade ? (
          <div className="trade-return-consultation-page__content">
            {eligibilityError ? (
              <Alert variant="error">
                {eligibilityError}
              </Alert>
            ) : null}

            {!eligibilityError && identityVerification.isLoading ? (
              <StatePanel
                variant="loading"
                title="本人確認情報を確認中..."
              />
            ) : null}

            {!eligibilityError &&
            !identityVerification.isLoading &&
            identityVerification.errorMessage ? (
              <Alert variant="error">
                {identityVerification.errorMessage}
              </Alert>
            ) : null}

            {!eligibilityError &&
            !identityVerification.isLoading &&
            !identityVerification.isVerified ? (
              <section className="trade-return-consultation-page__verification">
                <Alert variant="warning">
                  返品について相談するには、本人確認が必要です。
                </Alert>

                <Button
                  type="button"
                  variant="primary"
                  disabled={submitting}
                  onClick={() => {
                    navigate("/settings/identity-verification");
                  }}
                >
                  本人確認へ
                </Button>
              </section>
            ) : null}

            {!eligibilityError &&
            !identityVerification.isLoading &&
            identityVerification.isVerified ? (
              <>
                <section
                  className="trade-return-consultation-page__section"
                  aria-labelledby="trade-return-consultation-reason"
                >
                  <h2
                    id="trade-return-consultation-reason"
                    className="trade-return-consultation-page__section-title"
                  >
                    返品理由
                  </h2>

                  <div
                    className="trade-return-consultation-page__reason-options"
                    role="group"
                    aria-labelledby="trade-return-consultation-reason"
                  >
                    {TRADE_RETURN_CONSULTATION_REASONS.map((value) => (
                      <Chip
                        key={value}
                        selected={reason === value}
                        disabled={submitting}
                        onClick={() => {
                          handleReasonChange(value);
                        }}
                      >
                        {getReasonLabel(value)}
                      </Chip>
                    ))}
                  </div>
                </section>

                <Alert variant="warning">
                  返品相談を送信した時点では返品は確定しません。出品者からの回答を確認してください。
                </Alert>

                {shouldShowDesktopComposer ? (
                  <section className="trade-return-consultation-page__section">
                    <h2 className="trade-return-consultation-page__section-title">
                      詳細
                    </h2>

                    <ChatInlineComposer {...composer} />
                  </section>
                ) : null}

                {!isMobilePortrait && submissionError ? (
                  <Alert variant="error">
                    {submissionError}
                  </Alert>
                ) : null}
              </>
            ) : null}
          </div>
        ) : null}
      </section>
    </Layout>
  );

  return (
    <>
      {isMobilePortrait ? (
        <MobileSwipeRightDismissPage
          title="返品について相談する"
          enabled
          dismissGestureEnabled={!submitting}
          onDismiss={handleDismiss}
        >
          {pageContent}
        </MobileSwipeRightDismissPage>
      ) : (
        pageContent
      )}

      {shouldShowMobileComposer ? (
        <MobileComposerFooter
          {...composer}
          autoFocus={false}
        />
      ) : null}
    </>
  );
}