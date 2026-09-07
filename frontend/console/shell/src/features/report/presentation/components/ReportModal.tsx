// frontend/console/shell/src/features/report/presentation/components/ReportModal.tsx

import * as React from "react";
import { createPortal } from "react-dom";

import type {
  ReportReason,
  ReportResponse,
  ReportTargetType,
} from "../../../../shared/types/report";
import {
  getReportReasonLabel,
  REPORT_REASONS,
  requiresReportDetail,
} from "../../../../shared/types/report";

import "./ReportModal.css";

type ReportModalProps = {
  open: boolean;
  targetType: ReportTargetType;
  reason: ReportReason;
  detail: string;
  submitting: boolean;
  errorMessage?: string | null;
  result?: ReportResponse | null;
  onReasonChange: (reason: ReportReason) => void;
  onDetailChange: (detail: string) => void;
  onSubmit: () => void | Promise<void>;
  onClose: () => void;
};

function getTargetLabel(targetType: ReportTargetType): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "商品レビュー";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "コメント";
    default:
      return "投稿";
  }
}

export default function ReportModal({
  open,
  targetType,
  reason,
  detail,
  submitting,
  errorMessage,
  result,
  onReasonChange,
  onDetailChange,
  onSubmit,
  onClose,
}: ReportModalProps) {
  React.useEffect(() => {
    if (!open) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !submitting) {
        onClose();
      }
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
  const requiresDetail = requiresReportDetail(reason);
  const normalizedDetail = detail.trim();
  const submitted = Boolean(result);
  const alreadyReported = Boolean(result && !result.reportCreated);
  const canSubmit =
    !submitting &&
    !submitted &&
    (!requiresDetail || normalizedDetail !== "");

  const handleBackdropMouseDown = (
    event: React.MouseEvent<HTMLDivElement>,
  ) => {
    if (event.target === event.currentTarget && !submitting) {
      onClose();
    }
  };

  const handleSubmit = () => {
    if (!canSubmit) {
      return;
    }

    void onSubmit();
  };

  const modal = (
    <div
      className="report-modal"
      role="presentation"
      onMouseDown={handleBackdropMouseDown}
    >
      <div
        className="report-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="report-modal-title"
        aria-describedby="report-modal-description"
        aria-busy={submitting}
      >
        <div className="report-modal__header">
          <div>
            <span className="report-modal__eyebrow">
              不適切な投稿の通報
            </span>

            <h2
              id="report-modal-title"
              className="report-modal__title"
            >
              {targetLabel}を通報
            </h2>
          </div>

          <button
            type="button"
            className="report-modal__close"
            disabled={submitting}
            aria-label="通報モーダルを閉じる"
            onClick={onClose}
          >
            ×
          </button>
        </div>

        {submitted ? (
          <div className="report-modal__result">
            <div
              className="report-modal__result-icon"
              aria-hidden="true"
            >
              ✓
            </div>

            <div className="report-modal__result-body">
              <h3 className="report-modal__result-title">
                {alreadyReported
                  ? "この投稿はすでに通報済みです"
                  : "通報を受け付けました"}
              </h3>

              <p
                id="report-modal-description"
                className="report-modal__description"
              >
                {alreadyReported
                  ? "同じブランドからの通報は重複して登録されません。"
                  : "Adminで内容を確認し、維持または削除の裁定を行います。通報した時点では投稿は削除されません。"}
              </p>
            </div>

            <div className="report-modal__actions">
              <button
                type="button"
                className="report-modal__button report-modal__button--primary"
                onClick={onClose}
              >
                閉じる
              </button>
            </div>
          </div>
        ) : (
          <>
            <p
              id="report-modal-description"
              className="report-modal__description"
            >
              この{targetLabel}が不適切だと判断した理由を選択してください。ブランドから直接投稿を削除することはできません。
            </p>

            <div className="report-modal__body">
              <fieldset
                className="report-modal__reasons"
                disabled={submitting}
              >
                <legend className="report-modal__label">
                  通報理由
                </legend>

                <div className="report-modal__reason-list">
                  {REPORT_REASONS.map((value) => (
                    <label
                      key={value}
                      className={[
                        "report-modal__reason",
                        reason === value
                          ? "report-modal__reason--selected"
                          : "",
                      ]
                        .filter(Boolean)
                        .join(" ")}
                    >
                      <input
                        type="radio"
                        className="report-modal__reason-input"
                        name="report-reason"
                        value={value}
                        checked={reason === value}
                        onChange={() => onReasonChange(value)}
                      />

                      <span className="report-modal__reason-label">
                        {getReportReasonLabel(value)}
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>

              {requiresDetail ? (
                <label className="report-modal__detail-field">
                  <span className="report-modal__label">
                    詳細
                    <span className="report-modal__required">
                      必須
                    </span>
                  </span>

                  <textarea
                    className="report-modal__textarea"
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

              {errorMessage ? (
                <p
                  className="report-modal__error"
                  role="alert"
                >
                  {errorMessage}
                </p>
              ) : null}
            </div>

            <div className="report-modal__actions">
              <button
                type="button"
                className="report-modal__button report-modal__button--ghost"
                disabled={submitting}
                onClick={onClose}
              >
                キャンセル
              </button>

              <button
                type="button"
                className="report-modal__button report-modal__button--danger"
                disabled={!canSubmit}
                onClick={handleSubmit}
              >
                {submitting ? "送信中" : "通報する"}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}