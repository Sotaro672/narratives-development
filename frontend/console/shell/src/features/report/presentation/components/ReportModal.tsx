// frontend/console/shell/src/features/report/presentation/components/ReportModal.tsx

import { Button } from "../../../../shared/ui/button";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";
import Modal from "../../../../shared/ui/modal";
import Textarea from "../../../../shared/ui/textarea";
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
  const targetLabel = getTargetLabel(targetType);
  const requiresDetail = requiresReportDetail(reason);
  const normalizedDetail = detail.trim();
  const submitted = Boolean(result);
  const alreadyReported = Boolean(result && !result.reportCreated);
  const canSubmit =
    !submitting &&
    !submitted &&
    (!requiresDetail || normalizedDetail !== "");

  const handleSubmit = () => {
    if (!canSubmit) return;
    void onSubmit();
  };

  const description = submitted
    ? alreadyReported
      ? "同じブランドからの通報は重複して登録されません。"
      : "Adminで内容を確認し、維持または削除の裁定を行います。通報した時点では投稿は削除されません。"
    : `この${targetLabel}が不適切だと判断した理由を選択してください。ブランドから直接投稿を削除することはできません。`;

  const footer = submitted ? (
    <Button
      type="button"
      variant="default"
      onClick={onClose}
    >
      閉じる
    </Button>
  ) : (
    <Button
      type="button"
      variant="destructive"
      disabled={!canSubmit}
      onClick={handleSubmit}
    >
      {submitting ? "送信中" : "通報する"}
    </Button>
  );

  return (
    <Modal
      open={open}
      title={`${targetLabel}を通報`}
      eyebrow="不適切な投稿の通報"
      description={description}
      footer={footer}
      onClose={onClose}
      closeable={!submitting}
      closeOnBackdrop
      closeOnEscape
      showCloseButton
      closeLabel="通報モーダルを閉じる"
      ariaBusy={submitting}
      panelClassName="report-modal__panel"
    >
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
          </div>
        </div>
      ) : (
        <div className="report-modal__content">
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
            <div className="report-modal__detail-field">
              <Label
                className="report-modal__label"
                htmlFor="report-modal-detail"
              >
                詳細
                <span className="report-modal__required">
                  必須
                </span>
              </Label>

              <Textarea
                id="report-modal-detail"
                size="medium"
                value={detail}
                rows={5}
                disabled={submitting}
                placeholder="通報する理由を具体的に入力してください。"
                onChange={(event) => onDetailChange(event.target.value)}
              />
            </div>
          ) : null}

          {errorMessage ? (
            <ErrorMessage as="p" variant="panel">
              {errorMessage}
            </ErrorMessage>
          ) : null}
        </div>
      )}
    </Modal>
  );
}