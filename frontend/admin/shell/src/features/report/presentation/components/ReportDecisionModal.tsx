// frontend/admin/shell/src/features/report/presentation/components/ReportDecisionModal.tsx

import { useEffect } from "react";

import type { ReportCaseStatus, ReportTargetType } from "../../../../shared/type/report";
import Button from "../../../../shared/ui/Button/Button";

import "./ReportDecisionModal.css";

type ReportDecisionModalProps = {
  open: boolean;
  status: ReportCaseStatus;
  targetType?: ReportTargetType;
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

function getDescription(status: ReportCaseStatus, targetType?: ReportTargetType): string {
  const isAvatar = targetType === "AVATAR";

  if (isAvatar) {
    return status === "KEPT"
      ? "このアバターは変化なしで裁定済みです。必要な場合は再販サービス利用停止へ変更できます。"
      : "アバターに変化を加えないか、再販サービスのみ利用停止にするかを決定します。";
  }

  return status === "KEPT"
    ? "この投稿は維持済みです。必要な場合は削除へ変更できます。"
    : "投稿を維持するか、削除するかを決定します。";
}

function getPlaceholder(status: ReportCaseStatus, targetType?: ReportTargetType): string {
  if (targetType === "AVATAR") {
    return status === "KEPT"
      ? "再販サービス利用停止へ変更する根拠を入力してください。"
      : "裁定の根拠を入力してください。";
  }

  return status === "KEPT"
    ? "削除へ変更する根拠を入力してください。"
    : "裁定の根拠を入力してください。";
}

function getNote(targetType?: ReportTargetType): string {
  return targetType === "AVATAR"
    ? "「再販利用停止」を選択すると、アバター自体は削除・停止せず、対象アバターの再販サービスのみ利用停止にします。"
    : "「削除する」を選択すると、対象コンテンツの削除に成功した後でケースが削除済みとして確定します。";
}

function getKeepLabel(targetType?: ReportTargetType): string {
  return targetType === "AVATAR" ? "変化なし" : "維持する";
}

function getRemoveLabel(targetType?: ReportTargetType): string {
  return targetType === "AVATAR" ? "再販利用停止" : "削除する";
}

export default function ReportDecisionModal({
  open,
  status,
  targetType,
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

  const reasonRequired = decisionReason.trim().length === 0;
  const description = getDescription(status, targetType);
  const placeholder = getPlaceholder(status, targetType);
  const note = getNote(targetType);
  const keepLabel = getKeepLabel(targetType);
  const removeLabel = getRemoveLabel(targetType);

  return (
    <div
      className="report-decision-modal__overlay"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !deciding) onClose();
      }}
    >
      <section className="report-decision-modal" role="dialog" aria-modal="true" aria-labelledby="report-decision-modal-title">
        <header className="report-decision-modal__header">
          <div>
            <h2 id="report-decision-modal-title" className="report-decision-modal__title">
              {status === "KEPT" ? "裁定変更" : "裁定"}
            </h2>
            <p className="report-decision-modal__description">{description}</p>
          </div>

          <Button variant="ghost" size="sm" iconOnly aria-label="裁定モーダルを閉じる" disabled={deciding} onClick={onClose}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M18 6 6 18" />
              <path d="m6 6 12 12" />
            </svg>
          </Button>
        </header>

        <div className="report-decision-modal__body">
          <label className="report-decision-modal__field">
            <span className="report-decision-modal__label">裁定理由</span>
            <textarea
              className="report-decision-modal__textarea"
              value={decisionReason}
              rows={6}
              maxLength={2000}
              disabled={deciding}
              autoFocus
              placeholder={placeholder}
              onChange={(event) => onChangeDecisionReason(event.target.value)}
            />
          </label>

          {decisionError ? (
            <p className="report-decision-modal__error" role="alert">
              {decisionError}
            </p>
          ) : null}

          <p className="report-decision-modal__note">{note}</p>
        </div>

        <footer className="report-decision-modal__actions">
          <Button variant="secondary" size="md" disabled={deciding} onClick={onClose}>
            キャンセル
          </Button>

          <div className="report-decision-modal__decision-actions">
            {canKeep ? (
              <Button variant="secondary" size="md" loading={deciding} disabled={reasonRequired} onClick={() => void onKeep()}>
                {keepLabel}
              </Button>
            ) : null}

            {canRemove ? (
              <Button variant="danger" size="md" loading={deciding} disabled={reasonRequired} onClick={() => void onRemove()}>
                {removeLabel}
              </Button>
            ) : null}
          </div>
        </footer>
      </section>
    </div>
  );
}