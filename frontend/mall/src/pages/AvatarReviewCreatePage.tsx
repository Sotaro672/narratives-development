// frontend/mall/src/pages/AvatarReviewCreatePage.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import Button from "../components/ui/Button";
import Radio from "../components/ui/Radio";
import TextState from "../components/ui/TextState";
import {
  createAvatarReview,
  fetchAvatarReviewStatus,
  type AvatarReviewEvaluation,
  type AvatarReviewStatusResponse,
} from "../features/avatar-review/api/avatarReviewApi";
import { getPublicAvatar } from "../features/avatar/api/avatarApi";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";

import "../styles/page-layout.css";
import "../features/shared/styles/chat-detail-page.css";
import "../styles/avatar-review-create-page.css";

const MAX_COMMENT_LENGTH = 500;

function parseOrderItemIndex(value: string): number | null {
  const normalized = value.trim();

  if (!/^\d+$/.test(normalized)) {
    return null;
  }

  const parsed = Number(normalized);

  if (!Number.isSafeInteger(parsed) || parsed < 0) {
    return null;
  }

  return parsed;
}

function countCharacters(value: string): number {
  return Array.from(value).length;
}

export default function AvatarReviewCreatePage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const { orderId = "", itemIndex = "" } = useParams<{
    orderId: string;
    itemIndex: string;
  }>();

  const [status, setStatus] = useState<AvatarReviewStatusResponse | null>(null);
  const [revieweeAvatarName, setRevieweeAvatarName] = useState("");
  const [evaluation, setEvaluation] = useState<AvatarReviewEvaluation | null>(null);
  const [comment, setComment] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitted, setSubmitted] = useState(false);

  const normalizedOrderId = orderId.trim();
  const parsedItemIndex = useMemo(() => parseOrderItemIndex(itemIndex), [itemIndex]);
  const commentLength = useMemo(() => countCharacters(comment), [comment]);

  const reviewAvailable =
    status?.eligible === true &&
    status.reviewed === false &&
    !submitted;

  const canSubmit =
    reviewAvailable &&
    evaluation !== null &&
    !submitting &&
    comment.trim().length > 0 &&
    commentLength <= MAX_COMMENT_LENGTH;

  const shouldShowMobileComposer =
    isMobilePortrait &&
    reviewAvailable &&
    evaluation !== null;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    reviewAvailable &&
    evaluation !== null;

  const load = useCallback(async () => {
    if (!normalizedOrderId || parsedItemIndex === null) {
      setError("評価対象の取引情報が正しくありません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const nextStatus = await fetchAvatarReviewStatus({
        orderId: normalizedOrderId,
        orderItemIndex: parsedItemIndex,
      });

      setStatus(nextStatus);

      if (nextStatus.reviewed) {
        setRevieweeAvatarName("");
        return;
      }

      const revieweeAvatarId = nextStatus.revieweeAvatarId.trim();

      if (!revieweeAvatarId) {
        setRevieweeAvatarName("");
        return;
      }

      try {
        const avatar = await getPublicAvatar({
          avatarId: revieweeAvatarId,
        });

        setRevieweeAvatarName(avatar?.avatarName?.trim() ?? "");
      } catch {
        setRevieweeAvatarName("");
      }
    } catch (caughtError) {
      setStatus(null);
      setRevieweeAvatarName("");
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : "評価対象の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [normalizedOrderId, parsedItemIndex]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleEvaluationChange = useCallback(
    (nextEvaluation: AvatarReviewEvaluation) => {
      if (submitting) {
        return;
      }

      setEvaluation(nextEvaluation);
      setSubmitError(null);
    },
    [submitting],
  );

  const handleCommentChange = useCallback((value: string) => {
    setComment(value);
    setSubmitError(null);
  }, []);

  const handleSubmit = useCallback(async () => {
    if (
      !canSubmit ||
      parsedItemIndex === null ||
      evaluation === null
    ) {
      return;
    }

    setSubmitting(true);
    setSubmitError(null);

    try {
      await createAvatarReview({
        orderId: normalizedOrderId,
        orderItemIndex: parsedItemIndex,
        evaluation,
        comment,
      });

      setSubmitted(true);
      setStatus((current) => {
        if (!current) {
          return current;
        }

        return {
          ...current,
          reviewed: true,
        };
      });
    } catch (caughtError) {
      setSubmitError(
        caughtError instanceof Error
          ? caughtError.message
          : "評価の投稿に失敗しました。",
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    canSubmit,
    comment,
    evaluation,
    normalizedOrderId,
    parsedItemIndex,
  ]);

  const handleBackToOrder = useCallback(() => {
    if (submitting) {
      return;
    }

    if (!normalizedOrderId) {
      navigate("/wallet");
      return;
    }

    navigate(`/orders/${encodeURIComponent(normalizedOrderId)}`);
  }, [navigate, normalizedOrderId, submitting]);

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content: comment,
      placeholder: "取引についてコメントを入力してください",
      error: submitError,
      submitting,
      canSubmit,
      disabled: !reviewAvailable || evaluation === null,
      submitLabel: "評価を投稿",
      submittingLabel: "投稿中...",
      maxLength: MAX_COMMENT_LENGTH,
      onContentChange: handleCommentChange,
      onSubmit: handleSubmit,
    }),
    [
      canSubmit,
      comment,
      evaluation,
      handleCommentChange,
      handleSubmit,
      reviewAvailable,
      submitError,
      submitting,
    ],
  );

  return (
    <>
      <Layout
        title="アバターを評価"
        titleClickable={false}
        mode="mypage"
        showFooter={!isMobilePortrait}
        showBackButton
        backButtonLabel="注文詳細へ戻る"
        onBackButtonClick={handleBackToOrder}
        hideSettingsButton
        hideAnnouncementButton
      >
        <section
          className="page-section avatar-review-create-page"
          style={
            shouldShowMobileComposer
              ? {
                  paddingBottom:
                    "calc(var(--mobile-composer-height, 56px) + 24px)",
                }
              : undefined
          }
        >
          {!loading && !error && reviewAvailable ? (
            <header className="avatar-review-create-page__header">
              <p className="avatar-review-create-page__description">
                {revieweeAvatarName
                  ? `${revieweeAvatarName}との取引を評価してください。`
                  : "取引相手のアバターを評価してください。"}
              </p>
            </header>
          ) : null}

          {loading ? (
            <div className="avatar-review-create-page__state">
              <TextState variant="loading">
                評価対象を確認しています...
              </TextState>
            </div>
          ) : null}

          {!loading && error ? (
            <div className="avatar-review-create-page__state">
              <TextState variant="error">
                {error}
              </TextState>

              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  void load();
                }}
              >
                再読み込み
              </Button>
            </div>
          ) : null}

          {!loading &&
          !error &&
          status &&
          (status.reviewed || submitted) ? (
            <div className="avatar-review-create-page__state">
              <TextState variant="success">
                {submitted
                  ? "評価を投稿しました。"
                  : "この取引の評価は投稿済みです。"}
              </TextState>

              <Button
                type="button"
                variant="secondary"
                onClick={handleBackToOrder}
              >
                注文詳細へ戻る
              </Button>
            </div>
          ) : null}

          {!loading &&
          !error &&
          status &&
          !status.eligible ? (
            <div className="avatar-review-create-page__state">
              <TextState variant="error">
                この取引はアバター評価の対象ではありません。
              </TextState>

              <Button
                type="button"
                variant="secondary"
                onClick={handleBackToOrder}
              >
                注文詳細へ戻る
              </Button>
            </div>
          ) : null}

          {!loading && !error && reviewAvailable ? (
            <form
              className="avatar-review-create-page__form"
              onSubmit={(event) => {
                event.preventDefault();
                void handleSubmit();
              }}
            >
              <section
                className="avatar-review-create-page__section"
                aria-labelledby="avatar-review-evaluation-title"
              >
                <h2
                  id="avatar-review-evaluation-title"
                  className="avatar-review-create-page__section-title"
                >
                  取引はいかがでしたか？
                </h2>

                <div
                  className="avatar-review-create-page__evaluation-options"
                  role="radiogroup"
                  aria-labelledby="avatar-review-evaluation-title"
                >
                  <Radio
                    id="avatar-review-good"
                    name="avatar-review-evaluation"
                    value="good"
                    variant="card"
                    checked={evaluation === "good"}
                    disabled={submitting}
                    label="良かった"
                    onChange={() => {
                      handleEvaluationChange("good");
                    }}
                  />

                  <Radio
                    id="avatar-review-disappointed"
                    name="avatar-review-evaluation"
                    value="disappointed"
                    variant="card"
                    checked={evaluation === "disappointed"}
                    disabled={submitting}
                    label="残念だった"
                    onChange={() => {
                      handleEvaluationChange("disappointed");
                    }}
                  />
                </div>
              </section>

              {shouldShowDesktopComposer ? (
                <section className="avatar-review-create-page__section">
                  <h2 className="avatar-review-create-page__section-title">
                    コメント
                  </h2>

                  <ChatInlineComposer {...composer} />

                  <TextState
                    variant={
                      commentLength > MAX_COMMENT_LENGTH
                        ? "error"
                        : "muted"
                    }
                    className="avatar-review-create-page__comment-count"
                  >
                    {commentLength}/{MAX_COMMENT_LENGTH}文字
                  </TextState>
                </section>
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