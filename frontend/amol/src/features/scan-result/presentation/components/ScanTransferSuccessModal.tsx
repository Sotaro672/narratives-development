// frontend/amol/src/features/scan-result/presentation/components/ScanTransferSuccessModal.tsx

import { useEffect, useState } from "react";

type ReviewEvaluation = "good" | "disappointed";

type Props = {
  open: boolean;
  loading: boolean;
  error: string | null;
  canOpenContents: boolean;
  onClose: () => void;
  onOpenContents: () => void;
  reviewEnabled?: boolean;
  reviewSubmitting?: boolean;
  reviewSubmitted?: boolean;
  reviewError?: string | null;
  onSubmitReview?: (
    evaluation: ReviewEvaluation,
    comment: string,
  ) => void | Promise<void>;
};

const MAX_REVIEW_COMMENT_LENGTH = 500;

export default function ScanTransferSuccessModal({
  open,
  loading,
  error,
  canOpenContents,
  onClose,
  onOpenContents,
  reviewEnabled = false,
  reviewSubmitting = false,
  reviewSubmitted = false,
  reviewError = null,
  onSubmitReview,
}: Props) {
  const [reviewEvaluation, setReviewEvaluation] =
    useState<ReviewEvaluation | null>(null);
  const [reviewComment, setReviewComment] = useState("");

  useEffect(() => {
    if (!open) {
      setReviewEvaluation(null);
      setReviewComment("");
    }
  }, [open]);

  if (!open) {
    return null;
  }

  const normalizedComment = reviewComment.trim();
  const canSubmitReview =
    reviewEnabled &&
    !reviewSubmitting &&
    !reviewSubmitted &&
    reviewEvaluation !== null &&
    normalizedComment.length > 0 &&
    onSubmitReview != null;

  const handleSubmitReview = async () => {
    if (
      !canSubmitReview ||
      reviewEvaluation === null ||
      onSubmitReview == null
    ) {
      return;
    }

    await onSubmitReview(
      reviewEvaluation,
      normalizedComment,
    );
  };

  return (
    <div
      className="scan-transfer-modal-backdrop"
      onClick={onClose}
    >
      <div
        className="scan-transfer-modal"
        role="dialog"
        aria-modal="true"
        onClick={(event) => {
          event.stopPropagation();
        }}
      >
        <div className="scan-transfer-modal__header">
          <h2 className="scan-transfer-modal__title">
            トークン移譲完了
          </h2>

          <button
            type="button"
            className="scan-transfer-modal__close"
            onClick={onClose}
            aria-label="閉じる"
          >
            ×
          </button>
        </div>

        {loading ? (
          <p className="scan-transfer-modal__message">
            移譲処理中です...
          </p>
        ) : error ? (
          <p className="scan-transfer-modal__error">
            {error}
          </p>
        ) : (
          <div className="scan-transfer-modal__body">
            <p className="scan-transfer-modal__message">
              トークンの移譲が完了しました。
            </p>

            {reviewEnabled ? (
              <section
                className="scan-transfer-modal__review"
                aria-label="取引相手への評価"
              >
                <h3 className="scan-transfer-modal__review-title">
                  取引相手への評価
                </h3>

                {reviewSubmitted ? (
                  <p className="scan-transfer-modal__review-success">
                    評価を投稿しました。ありがとうございます。
                  </p>
                ) : (
                  <>
                    <p className="scan-transfer-modal__review-description">
                      商品を受け取るまでの対応はいかがでしたか？
                    </p>

                    <div
                      className="scan-transfer-modal__review-options"
                      role="group"
                      aria-label="評価"
                    >
                      <button
                        type="button"
                        className={[
                          "scan-transfer-modal__review-option",
                          reviewEvaluation === "good"
                            ? "scan-transfer-modal__review-option--selected"
                            : "",
                        ]
                          .filter(Boolean)
                          .join(" ")}
                        aria-pressed={
                          reviewEvaluation === "good"
                        }
                        disabled={reviewSubmitting}
                        onClick={() => {
                          setReviewEvaluation("good");
                        }}
                      >
                        良かった
                      </button>

                      <button
                        type="button"
                        className={[
                          "scan-transfer-modal__review-option",
                          reviewEvaluation === "disappointed"
                            ? "scan-transfer-modal__review-option--selected"
                            : "",
                        ]
                          .filter(Boolean)
                          .join(" ")}
                        aria-pressed={
                          reviewEvaluation === "disappointed"
                        }
                        disabled={reviewSubmitting}
                        onClick={() => {
                          setReviewEvaluation(
                            "disappointed",
                          );
                        }}
                      >
                        残念だった
                      </button>
                    </div>

                    <label className="scan-transfer-modal__review-comment">
                      <span className="scan-transfer-modal__review-comment-label">
                        コメント
                      </span>

                      <textarea
                        className="scan-transfer-modal__review-textarea"
                        value={reviewComment}
                        maxLength={
                          MAX_REVIEW_COMMENT_LENGTH
                        }
                        disabled={reviewSubmitting}
                        placeholder="取引相手の対応についてコメントしてください"
                        onChange={(event) => {
                          setReviewComment(
                            event.target.value,
                          );
                        }}
                      />
                    </label>

                    <div className="scan-transfer-modal__review-meta">
                      <span>
                        {reviewComment.length}/
                        {MAX_REVIEW_COMMENT_LENGTH}
                      </span>
                    </div>

                    {reviewError ? (
                      <p className="scan-transfer-modal__error">
                        {reviewError}
                      </p>
                    ) : null}

                    <button
                      type="button"
                      className="scan-transfer-modal__button scan-transfer-modal__review-submit"
                      disabled={!canSubmitReview}
                      onClick={() => {
                        void handleSubmitReview();
                      }}
                    >
                      {reviewSubmitting
                        ? "投稿中..."
                        : "評価を投稿"}
                    </button>
                  </>
                )}
              </section>
            ) : null}
          </div>
        )}

        <div className="scan-transfer-modal__footer">
          {canOpenContents && !loading && !error ? (
            <button
              type="button"
              className="scan-transfer-modal__button"
              onClick={onOpenContents}
            >
              コンテンツを見る
            </button>
          ) : null}

          <button
            type="button"
            className="scan-transfer-modal__button scan-transfer-modal__button--secondary"
            onClick={onClose}
          >
            閉じる
          </button>
        </div>
      </div>
    </div>
  );
}