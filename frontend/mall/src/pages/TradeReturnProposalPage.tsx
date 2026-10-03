// frontend/mall/src/pages/TradeReturnProposalPage.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
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
import Input from "../components/ui/Input";
import StatePanel from "../components/ui/StatePanel";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";
import type {
  TradeReturnAgreement,
  TradeReturnConsultationReason,
  TradeReturnRequirement,
} from "../features/shared/types/trade";
import { createTradeReturnProposal } from "../features/trade/infrastructure/tradeApi";
import useTradeThread from "../features/trade/presentation/hooks/useTradeThread";
import { getErrorMessage } from "../features/trade/presentation/util/tradeChatDetail";

import "../styles/page-layout.css";
import "../styles/trade-return-proposal-page.css";

type TradeReturnProposalRouteParams = {
  tradeId?: string;
};

const MAX_REASON_LENGTH = 5000;

function getConsultationReasonLabel(
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

function formatCurrency(value: number): string {
  if (!Number.isFinite(value) || value < 0) {
    return "-";
  }

  return `${value.toLocaleString("ja-JP")}円`;
}

export default function TradeReturnProposalPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const { tradeId: routeTradeId } = useParams<TradeReturnProposalRouteParams>();
  const normalizedTradeId = routeTradeId?.trim() ?? "";

  const thread = useTradeThread(normalizedTradeId);
  const initializedTradeIdRef = useRef("");

  const [agreement, setAgreementState] =
    useState<TradeReturnAgreement | null>(null);
  const [returnRequirement, setReturnRequirementState] =
    useState<TradeReturnRequirement | null>(null);
  const [refundAmount, setRefundAmountState] =
    useState<number | "">("");
  const [responseReason, setResponseReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submissionError, setSubmissionError] = useState("");

  const chatPath = normalizedTradeId
    ? `/chats/trades/${encodeURIComponent(normalizedTradeId)}`
    : "/chats";

  const trade = thread.trade;
  const refundAmountMax = trade?.merchandiseRefundMaxAmount ?? 0;
  const existingProposal = trade?.returnProposal ?? null;
  const isUpdate = existingProposal !== null;
  const pageTitle = isUpdate
    ? "返品相談への回答を更新"
    : "返品相談に回答する";

  useEffect(() => {
    if (
      !trade ||
      !normalizedTradeId ||
      initializedTradeIdRef.current === normalizedTradeId
    ) {
      return;
    }

    initializedTradeIdRef.current = normalizedTradeId;

    const proposal = trade.returnProposal;

    if (!proposal) {
      setAgreementState(null);
      setReturnRequirementState(null);
      setRefundAmountState("");
      setResponseReason("");
      setSubmissionError("");
      return;
    }

    setAgreementState(proposal.agreement);
    setResponseReason(proposal.reason?.trim() ?? "");
    setSubmissionError("");

    if (
      proposal.agreement === "agree" &&
      proposal.returnRequirement &&
      proposal.refundAmount !== undefined
    ) {
      setReturnRequirementState(proposal.returnRequirement);
      setRefundAmountState(proposal.refundAmount);
      return;
    }

    setReturnRequirementState(null);
    setRefundAmountState("");
  }, [normalizedTradeId, trade]);

  const eligibilityError = useMemo(() => {
    if (!trade) {
      return "";
    }

    if (trade.viewerSide !== "seller") {
      return "返品相談に回答できるのは出品者のみです。";
    }

    if (trade.status !== "active" || trade.isCancelled) {
      return "この取引では返品相談に回答できません。";
    }

    if (!trade.isDispatched) {
      return "商品が発送されるまで返品相談に回答できません。";
    }

    if (trade.transferred) {
      return "受け取り済みの商品については返品相談に回答できません。";
    }

    if (trade.returnStatus !== "discussing") {
      return "この取引は返品相談へ回答できる状態ではありません。";
    }

    if (!trade.returnConsultation) {
      return "購入者からの返品相談情報が見つかりません。";
    }

    return "";
  }, [trade]);

  const normalizedResponseReason = responseReason.trim();

  const validRefundAmount =
    typeof refundAmount === "number" &&
    Number.isInteger(refundAmount) &&
    refundAmount > 0 &&
    refundAmount <= refundAmountMax;

  const canEdit =
    Boolean(trade) &&
    !thread.loading &&
    !eligibilityError &&
    !submitting;

  const conditionsCompleted =
    agreement === "disagree" ||
    (
      agreement === "agree" &&
      returnRequirement !== null &&
      validRefundAmount
    );

  const canSubmit =
    canEdit &&
    agreement !== null &&
    conditionsCompleted &&
    normalizedResponseReason.length > 0 &&
    normalizedResponseReason.length <= MAX_REASON_LENGTH;

  const shouldShowMobileComposer =
    isMobilePortrait &&
    Boolean(trade) &&
    !thread.loading &&
    !eligibilityError;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    Boolean(trade) &&
    !thread.loading &&
    !eligibilityError;

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

  const handleAgreementChange = useCallback(
    (value: TradeReturnAgreement) => {
      if (submitting) {
        return;
      }

      setAgreementState(value);
      setSubmissionError("");

      if (value === "disagree") {
        setReturnRequirementState(null);
        setRefundAmountState("");
      }
    },
    [submitting],
  );

  const handleReturnRequirementChange = useCallback(
    (value: TradeReturnRequirement) => {
      if (submitting) {
        return;
      }

      setReturnRequirementState(value);
      setSubmissionError("");
    },
    [submitting],
  );

  const handleRefundAmountChange = useCallback(
    (value: string) => {
      if (submitting) {
        return;
      }

      setSubmissionError("");

      if (value === "") {
        setRefundAmountState("");
        return;
      }

      const parsed = Number(value);

      if (!Number.isFinite(parsed)) {
        return;
      }

      setRefundAmountState(parsed);
    },
    [submitting],
  );

  const handleResponseReasonChange = useCallback((value: string) => {
    setResponseReason(value);
    setSubmissionError("");
  }, []);

  const handleSubmit = useCallback(async (): Promise<void> => {
    if (
      submitting ||
      !canSubmit ||
      !trade ||
      !normalizedTradeId ||
      !agreement
    ) {
      return;
    }

    if (
      trade.viewerSide !== "seller" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.returnStatus !== "discussing" ||
      !trade.returnConsultation
    ) {
      setSubmissionError(
        "取引状態が変更されたため、返品相談に回答できません。",
      );
      return;
    }

    if (agreement === "agree") {
      if (!returnRequirement) {
        setSubmissionError(
          "商品を返品してもらうか選択してください。",
        );
        return;
      }

      if (refundAmount === "") {
        setSubmissionError("返金額を入力してください。");
        return;
      }

      if (!Number.isInteger(refundAmount)) {
        setSubmissionError(
          "返金額は1円単位の整数で入力してください。",
        );
        return;
      }

      if (refundAmount <= 0) {
        setSubmissionError(
          "返金額は1円以上で入力してください。",
        );
        return;
      }

      if (
        !Number.isInteger(refundAmountMax) ||
        refundAmountMax <= 0
      ) {
        setSubmissionError(
          "返金可能額を取得できません。取引情報を再読み込みしてください。",
        );
        return;
      }

      if (refundAmount > refundAmountMax) {
        setSubmissionError(
          `返金額は${refundAmountMax.toLocaleString("ja-JP")}円以下で入力してください。`,
        );
        return;
      }
    }

    setSubmitting(true);
    setSubmissionError("");

    try {
      if (agreement === "disagree") {
        await createTradeReturnProposal({
          tradeId: normalizedTradeId,
          agreement: "disagree",
          reason: normalizedResponseReason,
        });
      } else {
        await createTradeReturnProposal({
          tradeId: normalizedTradeId,
          agreement: "agree",
          returnRequirement: returnRequirement!,
          refundAmount: refundAmount as number,
          reason: normalizedResponseReason,
        });
      }

      navigate(chatPath, { replace: true });
    } catch (caught) {
      setSubmissionError(
        getErrorMessage(
          caught,
          isUpdate
            ? "返品相談への回答を更新できませんでした。"
            : "返品相談への回答を送信できませんでした。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    agreement,
    canSubmit,
    chatPath,
    isUpdate,
    navigate,
    normalizedResponseReason,
    normalizedTradeId,
    refundAmount,
    refundAmountMax,
    returnRequirement,
    submitting,
    trade,
  ]);

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content: responseReason,
      placeholder: "回答理由を入力",
      error: submissionError || null,
      submitting,
      canSubmit,
      disabled: !canEdit,
      submitLabel: isUpdate ? "更新" : "回答",
      submittingLabel: isUpdate
        ? "更新中..."
        : "送信中...",
      maxLength: MAX_REASON_LENGTH,
      onContentChange: handleResponseReasonChange,
      onSubmit: handleSubmit,
    }),
    [
      canEdit,
      canSubmit,
      handleResponseReasonChange,
      handleSubmit,
      isUpdate,
      responseReason,
      submissionError,
      submitting,
    ],
  );

  const pageContent = (
    <Layout
      title={pageTitle}
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
        className="page-section trade-return-proposal-page"
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

        {!thread.loading && thread.error && !trade ? (
          <div className="trade-return-proposal-page__state">
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

        {!thread.loading && trade ? (
          <div className="trade-return-proposal-page__content">
            {eligibilityError ? (
              <Alert variant="error">
                {eligibilityError}
              </Alert>
            ) : (
              <>
                {trade.returnConsultation ? (
                  <section className="trade-return-proposal-page__section">
                    <h2 className="trade-return-proposal-page__section-title">
                      購入者からの返品相談
                    </h2>

                    <div className="trade-return-proposal-page__consultation">
                      <div className="trade-return-proposal-page__consultation-row">
                        <span className="trade-return-proposal-page__consultation-label">
                          返品理由
                        </span>
                        <span>
                          {getConsultationReasonLabel(
                            trade.returnConsultation.reason,
                          )}
                        </span>
                      </div>

                      <div className="trade-return-proposal-page__consultation-detail">
                        {trade.returnConsultation.detail}
                      </div>
                    </div>
                  </section>
                ) : null}

                <section
                  className="trade-return-proposal-page__section"
                  aria-labelledby="trade-return-proposal-agreement"
                >
                  <h2
                    id="trade-return-proposal-agreement"
                    className="trade-return-proposal-page__section-title"
                  >
                    返品に合意しますか？
                  </h2>

                  <div
                    className="trade-return-proposal-page__option-group"
                    role="group"
                    aria-labelledby="trade-return-proposal-agreement"
                  >
                    <Chip
                      selected={agreement === "agree"}
                      disabled={submitting}
                      onClick={() => {
                        handleAgreementChange("agree");
                      }}
                    >
                      返品に合意する
                    </Chip>

                    <Chip
                      selected={agreement === "disagree"}
                      disabled={submitting}
                      onClick={() => {
                        handleAgreementChange("disagree");
                      }}
                    >
                      返品に合意しない
                    </Chip>
                  </div>
                </section>

                {agreement === "agree" ? (
                  <>
                    <section
                      className="trade-return-proposal-page__section"
                      aria-labelledby="trade-return-proposal-requirement"
                    >
                      <h2
                        id="trade-return-proposal-requirement"
                        className="trade-return-proposal-page__section-title"
                      >
                        商品を返品してもらいますか？
                      </h2>

                      <div
                        className="trade-return-proposal-page__option-group"
                        role="group"
                        aria-labelledby="trade-return-proposal-requirement"
                      >
                        <Chip
                          selected={returnRequirement === "required"}
                          disabled={submitting}
                          onClick={() => {
                            handleReturnRequirementChange("required");
                          }}
                        >
                          商品を返品してもらう
                        </Chip>

                        <Chip
                          selected={returnRequirement === "not_required"}
                          disabled={submitting}
                          onClick={() => {
                            handleReturnRequirementChange("not_required");
                          }}
                        >
                          商品を返品してもらわない
                        </Chip>
                      </div>
                    </section>

                    <section className="trade-return-proposal-page__section">
                      <h2 className="trade-return-proposal-page__section-title">
                        返金額
                      </h2>

                      <Input
                        id="trade-return-proposal-refund-amount"
                        type="number"
                        inputMode="numeric"
                        min={1}
                        max={
                          refundAmountMax > 0
                            ? refundAmountMax
                            : undefined
                        }
                        step={1}
                        value={refundAmount}
                        disabled={submitting}
                        helperText={`返金上限: ${formatCurrency(refundAmountMax)}`}
                        onChange={(event) => {
                          handleRefundAmountChange(event.target.value);
                        }}
                      />
                    </section>

                    {returnRequirement === "required" ? (
                      <Alert variant="info">
                        購入者が提示した条件に同意した後、PUDO匿名返品の手続きへ進みます。
                      </Alert>
                    ) : null}

                    {returnRequirement === "not_required" ? (
                      <Alert variant="info">
                        商品は購入者が保持したまま、指定した金額を返金する条件として提示します。
                      </Alert>
                    ) : null}
                  </>
                ) : null}

                {agreement === "disagree" ? (
                  <Alert variant="info">
                    返品に合意しない場合、商品返品および返金は行わず、購入者とのメッセージ相談を継続します。回答理由を入力して購入者へ伝えてください。
                  </Alert>
                ) : null}

                {shouldShowDesktopComposer ? (
                  <section className="trade-return-proposal-page__section">
                    <h2 className="trade-return-proposal-page__section-title">
                      回答理由
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
          title={pageTitle}
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