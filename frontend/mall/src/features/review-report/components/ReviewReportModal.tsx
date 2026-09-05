// frontend/mall/src/features/review-report/components/ReviewReportModal.tsx

import { useEffect } from "react";
import { createPortal } from "react-dom";

import type {
  ReviewReportReason,
  ReviewReportResponse,
  ReviewReportTargetType,
} from "../../shared/types/reviewReport";
import {
  getReviewReportReasonLabel,
  REVIEW_REPORT_REASONS,
} from "../../shared/types/reviewReport";

import "../styles/review-report.css";

type ReviewReportModalProps = {
  open: boolean;
  targetType?: ReviewReportTargetType;
  reason: ReviewReportReason;
  detail: string;
  submitting: boolean;
  error: string | null;
  result: ReviewReportResponse | null;
  canSubmit: boolean;
  onReasonChange: (reason: ReviewReportReason) => void;
  onDetailChange: (detail: string) => void;
  onSubmit: () => void | Promise<ReviewReportResponse | null>;
  onClose: () => void;
};

function getTargetLabel(targetType?: ReviewReportTargetType): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "レビュー";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "コメント";
    default:
      return "投稿";
  }
}

export default function ReviewReportModal({
  open,
  targetType,
  reason,
  detail,
  submitting,
  error,
  result,
  canSubmit,
  onReasonChange,
  onDetailChange,
  onSubmit,
  onClose,
}: ReviewReportModalProps) {
  useEffect(() => {
    if (!open) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || submitting) {
        return;
      }

      onClose();
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, submitting, onClose]);

  if (!open) {
    return null;
  }

  const targetLabel = getTargetLabel(targetType);
  const submitted = result !== null;
  const alreadyReported = result !== null && !result.reportCreated;

  const handleBackdropClick = (
    event: React.MouseEvent<HTMLDivElement>,
  ) => {
    if (event.target !== event.currentTarget || submitting) {
      return;
    }

    onClose();
  };

  const handleSubmit = () => {
    if (!canSubmit || submitting || submitted) {
      return;
    }

    void onSubmit();
  };

  const modal = (
    <div
      className="review-report-modal"
      role="presentation"
      onMouseDown={handleBackdropClick}
    >
      <div
        className="review-report-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="review-report-modal-title"
        aria-describedby="review-report-modal-description"
        aria-busy={submitting}
      >
        <div className="review-report-modal__header">
          <div>
            <span className="review-report-modal__eyebrow">通報</span>
            <h2
              id="review-report-modal-title"
              className="review-report-modal__title"
            >
              {targetLabel}を通報
            </h2>
          </div>

          <button
            type="button"
            className="review-report-modal__close"
            aria-label="通報画面を閉じる"
            disabled={submitting}
            onClick={onClose}
          >
            ×
          </button>
        </div>

        {submitted ? (
          <div className="review-report-modal__result">
            <div
              className="review-report-modal__result-icon"
              aria-hidden="true"
            >
              ✓
            </div>

            <div className="review-report-modal__result-body">
              <h3 className="review-report-modal__result-title">
                {alreadyReported
                  ? "この内容はすでに通報済みです"
                  : "通報を受け付けました"}
              </h3>

              <p
                id="review-report-modal-description"
                className="review-report-modal__description"
              >
                {alreadyReported
                  ? "同じアカウントからの通報は重複して登録されません。"
                  : "内容を確認のうえ、必要に応じて運営側で対応します。"}
              </p>
            </div>

            <div className="review-report-modal__actions">
              <button
                type="button"
                className="review-report-modal__button review-report-modal__button--primary"
                onClick={onClose}
              >
                閉じる
              </button>
            </div>
          </div>
        ) : (
          <>
            <p
              id="review-report-modal-description"
              className="review-report-modal__description"
            >
              この{targetLabel}が不適切だと思う理由を選択してください。通報しただけでは投稿は自動的に削除されません。
            </p>

            <div className="review-report-modal__body">
              <fieldset
                className="review-report-modal__reasons"
                disabled={submitting}
              >
                <legend className="review-report-modal__label">
                  通報理由
                </legend>

                <div className="review-report-modal__reason-list">
                  {REVIEW_REPORT_REASONS.map((value) => (
                    <label
                      key={value}
                      className={[
                        "review-report-modal__reason",
                        reason === value
                          ? "review-report-modal__reason--selected"
                          : "",
                      ]
                        .filter(Boolean)
                        .join(" ")}
                    >
                      <input
                        type="radio"
                        className="review-report-modal__reason-input"
                        name="review-report-reason"
                        value={value}
                        checked={reason === value}
                        onChange={() => onReasonChange(value)}
                      />
                      <span className="review-report-modal__reason-label">
                        {getReviewReportReasonLabel(value)}
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>

              {reason === "OTHER" ? (
                <label className="review-report-modal__detail-field">
                  <span className="review-report-modal__label">
                    詳細
                    <span className="review-report-modal__required">
                      必須
                    </span>
                  </span>

                  <textarea
                    className="review-report-modal__textarea"
                    value={detail}
                    rows={5}
                    disabled={submitting}
                    placeholder="通報する理由を具体的に入力してください。"
                    onChange={(event) =>
                      onDetailChange(event.target.value)
                    }
                  />
                </label>
              ) : null}

              {error ? (
                <p
                  className="review-report-modal__error"
                  role="alert"
                >
                  {error}
                </p>
              ) : null}
            </div>

            <div className="review-report-modal__actions">
              <button
                type="button"
                className="review-report-modal__button"
                disabled={submitting}
                onClick={onClose}
              >
                キャンセル
              </button>

              <button
                type="button"
                className="review-report-modal__button review-report-modal__button--danger"
                disabled={!canSubmit || submitting}
                onClick={handleSubmit}
              >
                {submitting ? "送信中..." : "通報する"}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}