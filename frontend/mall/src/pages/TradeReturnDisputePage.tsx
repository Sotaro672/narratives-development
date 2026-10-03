// frontend/mall/src/pages/TradeReturnDisputePage.tsx

import { useCallback, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeRightDismissPage from "../components/layout/MobileSwipeRightDismissPage";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Checkbox from "../components/ui/Checkbox";
import StatePanel from "../components/ui/StatePanel";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";
import { reportTradeReturnDispute } from "../features/trade/infrastructure/tradeApi";
import useTradeThread from "../features/trade/presentation/hooks/useTradeThread";
import { getErrorMessage } from "../features/trade/presentation/util/tradeChatDetail";

import "../styles/page-layout.css";
import "../styles/trade-return-dispute-page.css";

type TradeReturnDisputeRouteParams = {
  tradeId?: string;
};

const MAX_DETAIL_LENGTH = 5000;

export default function TradeReturnDisputePage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const { tradeId: routeTradeId } = useParams<TradeReturnDisputeRouteParams>();
  const normalizedTradeId = routeTradeId?.trim() ?? "";

  const thread = useTradeThread(normalizedTradeId);

  const [content, setContent] = useState("");
  const [agreedToReportConditions, setAgreedToReportConditions] = useState(false);
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
      return "運営へ報告できるのは購入者のみです。";
    }

    if (trade.status !== "active" || trade.isCancelled) {
      return "この取引は運営へ報告できる状態ではありません。";
    }

    if (!trade.isDispatched) {
      return "商品が発送されるまで運営へ報告できません。";
    }

    if (trade.transferred) {
      return "受け取り済みの商品については、この返品フローから運営へ報告できません。";
    }

    if (trade.returnStatus !== "discussing") {
      return "この取引は返品相談中ではありません。";
    }

    if (trade.returnProposal?.agreement !== "disagree") {
      return "出品者と返品について合意に至らなかった場合に運営へ報告できます。";
    }

    return "";
  }, [thread.trade]);

  const normalizedContent = content.trim();

  const canEdit =
    Boolean(thread.trade) &&
    !thread.loading &&
    !eligibilityError &&
    agreedToReportConditions &&
    !submitting;

  const canSubmit =
    canEdit &&
    normalizedContent.length > 0 &&
    normalizedContent.length <= MAX_DETAIL_LENGTH;

  const shouldShowMobileComposer =
    isMobilePortrait &&
    Boolean(thread.trade) &&
    !thread.loading &&
    !eligibilityError &&
    agreedToReportConditions;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    Boolean(thread.trade) &&
    !thread.loading &&
    !eligibilityError;

  const handleDismiss = useCallback(() => {
    if (submitting) {
      return;
    }

    navigate(chatPath, { replace: true });
  }, [chatPath, navigate, submitting]);

  const handleAgreementChange = useCallback((checked: boolean) => {
    if (submitting) {
      return;
    }

    setAgreedToReportConditions(checked);
    setSubmissionError("");
  }, [submitting]);

  const handleContentChange = useCallback((value: string) => {
    setContent(value);
    setSubmissionError("");
  }, []);

  const handleSubmit = useCallback(async (): Promise<void> => {
    const trade = thread.trade;

    if (
      submitting ||
      !agreedToReportConditions ||
      !canSubmit ||
      !trade ||
      !normalizedTradeId
    ) {
      return;
    }

    if (
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.returnStatus !== "discussing" ||
      trade.returnProposal?.agreement !== "disagree"
    ) {
      setSubmissionError("取引状態が変更されたため、運営へ報告できません。");
      return;
    }

    setSubmitting(true);
    setSubmissionError("");

    try {
      await reportTradeReturnDispute({
        tradeId: normalizedTradeId,
        detail: normalizedContent,
      });

      navigate(chatPath, { replace: true });
    } catch (caught) {
      setSubmissionError(
        getErrorMessage(
          caught,
          "取引を運営へ報告できませんでした。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    agreedToReportConditions,
    canSubmit,
    chatPath,
    navigate,
    normalizedContent,
    normalizedTradeId,
    submitting,
    thread.trade,
  ]);

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content,
      placeholder: "報告内容を入力",
      error: submissionError || null,
      submitting,
      canSubmit,
      disabled: !canEdit,
      submitLabel: "報告",
      submittingLabel: "報告中...",
      maxLength: MAX_DETAIL_LENGTH,
      onContentChange: handleContentChange,
      onSubmit: handleSubmit,
    }),
    [
      canEdit,
      canSubmit,
      content,
      handleContentChange,
      handleSubmit,
      submissionError,
      submitting,
    ],
  );

  const pageContent = (
    <Layout
      title="運営へ報告する"
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
        className="page-section trade-return-dispute-page"
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
          <div className="trade-return-dispute-page__state">
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
          <div className="trade-return-dispute-page__content">
            {eligibilityError ? (
              <Alert variant="error">
                {eligibilityError}
              </Alert>
            ) : (
              <>
                <section className="trade-return-dispute-page__section">
                  <h2 className="trade-return-dispute-page__section-title">
                    返品について合意に至らなかった場合
                  </h2>

                  <p className="trade-return-dispute-page__description">
                    出品者が返品に合意しなかったため、この取引について運営へ報告できます。取引の状況や、確認してほしい内容を具体的に入力してください。
                  </p>
                </section>

                <section className="trade-return-dispute-page__section">
                  <Alert variant="warning">
                    報告内容と取引履歴をもとに運営が状況を確認します。返品や返金がこの時点で確定するものではありません。
                  </Alert>

                  <Checkbox
                    id="trade-return-dispute-agreement"
                    label="上記の内容を確認し、同意する"
                    checked={agreedToReportConditions}
                    disabled={submitting}
                    onChange={(event) => {
                      handleAgreementChange(event.target.checked);
                    }}
                  />
                </section>

                {shouldShowDesktopComposer ? (
                  <section className="trade-return-dispute-page__section">
                    <h2 className="trade-return-dispute-page__section-title">
                      報告内容
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
            )}
          </div>
        ) : null}
      </section>
    </Layout>
  );

  return (
    <>
      {isMobilePortrait ? (
        <MobileSwipeRightDismissPage
          title="運営へ報告する"
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