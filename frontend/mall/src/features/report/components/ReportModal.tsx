// frontend/mall/src/features/report/components/ReportModal.tsx

import { type MouseEvent, useEffect } from "react";
import { createPortal } from "react-dom";

import type {
  ReportReason,
  ReportResponse,
  ReportTargetType,
} from "../../shared/types/report";
import {
  getReportReasonLabel,
  REPORT_REASONS,
} from "../../shared/types/report";

import "../styles/report.css";

type ReportModalProps = {
  open: boolean;
  targetType?: ReportTargetType;
  reason: ReportReason;
  detail: string;
  submitting: boolean;
  error: string | null;
  result: ReportResponse | null;
  canSubmit: boolean;
  onReasonChange: (reason: ReportReason) => void;
  onDetailChange: (detail: string) => void;
  onSubmit: () => void | Promise<ReportResponse | null>;
  onClose: () => void;
};

function getTargetLabel(targetType?: ReportTargetType): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "レビュー";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "コメント";
    case "AVATAR":
      return "アバター";
    default:
      return "投稿";
  }
}

function getDescription(targetType?: ReportTargetType): string {
  switch (targetType) {
    case "AVATAR":
      return "このアバターが不適切だと思う理由を選択してください。通報しただけでは再販サービスの利用が自動的に停止されることはありません。";
    case "PRODUCT_BLUEPRINT_REVIEW":
    case "TOKEN_BLUEPRINT_COMMENT":
      return `この${getTargetLabel(targetType)}が不適切だと思う理由を選択してください。通報しただけでは投稿は自動的に削除されません。`;
    default:
      return "この投稿が不適切だと思う理由を選択してください。通報しただけでは投稿は自動的に削除されません。";
  }
}

export default function ReportModal({
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
}: ReportModalProps) {
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
  const description = getDescription(targetType);
  const submitted = result !== null;
  const alreadyReported = result !== null && !result.reportCreated;

  const handleBackdropClick = (event: MouseEvent<HTMLDivElement>) => {
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
      className="report-modal"
      role="presentation"
      onMouseDown={handleBackdropClick}
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
            <span className="report-modal__eyebrow">通報</span>
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
            aria-label="通報画面を閉じる"
            disabled={submitting}
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
                  ? "この内容はすでに通報済みです"
                  : "通報を受け付けました"}
              </h3>

              <p
                id="report-modal-description"
                className="report-modal__description"
              >
                {alreadyReported
                  ? "同じアカウントからの通報は重複して登録されません。"
                  : "内容を確認のうえ、必要に応じて運営側で対応します。"}
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
              {description}
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

              {reason === "OTHER" ? (
                <label className="report-modal__detail-field">
                  <span className="report-modal__label">
                    詳細
                    <span className="report-modal__required">必須</span>
                  </span>

                  <textarea
                    className="report-modal__textarea"
                    value={detail}
                    rows={5}
                    disabled={submitting}
                    placeholder="通報する理由を具体的に入力してください。"
                    onChange={(event) => onDetailChange(event.target.value)}
                  />
                </label>
              ) : null}

              {error ? (
                <p className="report-modal__error" role="alert">
                  {error}
                </p>
              ) : null}
            </div>

            <div className="report-modal__actions">
              <button
                type="button"
                className="report-modal__button"
                disabled={submitting}
                onClick={onClose}
              >
                キャンセル
              </button>

              <button
                type="button"
                className="report-modal__button report-modal__button--danger"
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