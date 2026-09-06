//frontend\admin\shell\src\features\report\components\ReportDecisionModal.tsx
import { useEffect } from "react";

import type { ReviewReportCaseStatus } from "../../../shared/type/reviewReport";
import Button from "../../../shared/ui/Button/Button";

import "./ReportDecisionModal.css";

type ReportDecisionModalProps = {
  open: boolean;
  status: ReviewReportCaseStatus;
  decisionReason: string;
  deciding: boolean;
  decisionError: string | null;
  canKeep: boolean;
  canRemove: boolean;
  onChangeDecisionReason: (value: string) => void;
  onClose: () => void;
  onKeep: () => void | Promise<void>;
  onRemove: () => void | Promise<void>;
};

export default function ReportDecisionModal({
  open,
  status,
  decisionReason,
  deciding,
  decisionError,
  canKeep,
  canRemove,
  onChangeDecisionReason,
  onClose,
  onKeep,
  onRemove,
}: ReportDecisionModalProps) {
  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !deciding) onClose();
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [deciding, onClose, open]);

  if (!open) return null;

  const reasonRequired = !decisionReason.trim();

  return (
    <div
      className="report-decision-modal__overlay"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !deciding) onClose();
      }}
    >
      <section
        className="report-decision-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="report-decision-modal-title"
      >
        <header className="report-decision-modal__header">
          <div>
            <h2
              id="report-decision-modal-title"
              className="report-decision-modal__title"
            >
              {status === "KEPT" ? "裁定変更" : "裁定"}
            </h2>
            <p className="report-decision-modal__description">
              {status === "KEPT"
                ? "この投稿は維持済みです。必要な場合は削除へ変更できます。"
                : "投稿を維持するか、削除するかを決定します。"}
            </p>
          </div>

          <Button
            variant="ghost"
            size="sm"
            iconOnly
            aria-label="裁定モーダルを閉じる"
            disabled={deciding}
            onClick={onClose}
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M18 6 6 18" />
              <path d="m6 6 12 12" />
            </svg>
          </Button>
        </header>

        <div className="report-decision-modal__body">
          <label className="report-decision-modal__field">
            <span className="report-decision-modal__label">
              裁定理由
            </span>
            <textarea
              className="report-decision-modal__textarea"
              value={decisionReason}
              rows={6}
              maxLength={2000}
              disabled={deciding}
              autoFocus
              placeholder={
                status === "KEPT"
                  ? "削除へ変更する根拠を入力してください。"
                  : "裁定の根拠を入力してください。"
              }
              onChange={(event) =>
                onChangeDecisionReason(event.target.value)
              }
            />
          </label>

          {decisionError ? (
            <p
              className="report-decision-modal__error"
              role="alert"
            >
              {decisionError}
            </p>
          ) : null}

          <p className="report-decision-modal__note">
            「削除する」を選択すると、対象コンテンツの削除に成功した後でケースが削除済みとして確定します。
          </p>
        </div>

        <footer className="report-decision-modal__actions">
          <Button
            variant="secondary"
            size="md"
            disabled={deciding}
            onClick={onClose}
          >
            キャンセル
          </Button>

          <div className="report-decision-modal__decision-actions">
            {canKeep ? (
              <Button
                variant="secondary"
                size="md"
                loading={deciding}
                disabled={reasonRequired}
                onClick={() => void onKeep()}
              >
                維持する
              </Button>
            ) : null}

            {canRemove ? (
              <Button
                variant="danger"
                size="md"
                loading={deciding}
                disabled={reasonRequired}
                onClick={() => void onRemove()}
              >
                削除する
              </Button>
            ) : null}
          </div>
        </footer>
      </section>
    </div>
  );
}