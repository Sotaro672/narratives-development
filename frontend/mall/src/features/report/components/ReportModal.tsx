// frontend/mall/src/features/report/components/ReportModal.tsx

import Button from "../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../components/ui/Modal";
import Radio from "../../../components/ui/Radio";
import TextState from "../../../components/ui/TextState";
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
    case "LIST":
      return "出品";
    case "TOKEN_BLUEPRINT":
      return "トークン";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "コメント";
    case "AVATAR":
      return "アバター";
    case "BRAND":
      return "ブランド";
    case "RESALE":
      return "再販出品";
    case "TRADE_MESSAGE":
      return "取引コメント";
    case "ANNOUNCEMENT":
      return "お知らせ";
    default:
      return "投稿";
  }
}

function getDescription(targetType?: ReportTargetType): string {
  switch (targetType) {
    case "LIST":
      return "この出品内容が不適切だと思う理由を選択してください。通報しただけでは出品は自動的に停止されません。";
    case "RESALE":
      return "この再販出品が不適切だと思う理由を選択してください。通報しただけでは出品は自動的に停止されません。";
    case "TOKEN_BLUEPRINT":
      return "このトークンのコンテンツが不適切だと思う理由を選択してください。通報しただけではコンテンツは自動的に非表示になりません。";
    case "AVATAR":
      return "このアバターが不適切だと思う理由を選択してください。通報しただけでは再販サービスの利用が自動的に停止されることはありません。";
    case "BRAND":
      return "このブランドが不適切だと思う理由を選択してください。通報しただけではブランドは自動的に非表示になりません。";
    case "TRADE_MESSAGE":
      return "この取引コメントが不適切だと思う理由を選択してください。通報しただけでは取引コメントは自動的に非表示になりません。";
    case "ANNOUNCEMENT":
      return "このお知らせが不適切だと思う理由を選択してください。通報しただけではお知らせは自動的に削除されません。";
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
  const targetLabel = getTargetLabel(targetType);
  const description = getDescription(targetType);
  const submitted = result !== null;
  const alreadyReported = result !== null && !result.reportCreated;

  const handleSubmit = () => {
    if (!canSubmit || submitting || submitted) return;
    void onSubmit();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!submitting}
      closeOnEscape={!submitting}
      ariaLabelledBy="report-modal-title"
      ariaDescribedBy="report-modal-description"
      ariaBusy={submitting}
      panelClassName="report-modal__panel"
    >
      <ModalHeader
        onClose={submitting ? undefined : onClose}
        closeLabel="通報画面を閉じる"
      >

        <ModalTitle id="report-modal-title">{targetLabel}を通報</ModalTitle>
      </ModalHeader>

      {submitted ? (
        <>
          <ModalBody className="report-modal__result">
            <div className="report-modal__result-icon" aria-hidden="true">
              ✓
            </div>

            <div className="report-modal__result-body">
              <h3 className="report-modal__result-title">
                {alreadyReported
                  ? "この内容はすでに通報済みです"
                  : "通報を受け付けました"}
              </h3>

              <ModalDescription id="report-modal-description">
                {alreadyReported
                  ? "同じアカウントからの通報は重複して登録されません。"
                  : "内容を確認のうえ、必要に応じて運営側で対応します。"}
              </ModalDescription>
            </div>
          </ModalBody>

          <ModalFooter className="report-modal__result-actions">
            <Button variant="primary" size="md" onClick={onClose}>
              閉じる
            </Button>
          </ModalFooter>
        </>
      ) : (
        <>
          <ModalBody className="report-modal__body">
            <ModalDescription
              id="report-modal-description"
              className="report-modal__description"
            >
              {description}
            </ModalDescription>

            <fieldset className="report-modal__reasons" disabled={submitting}>
              <legend className="report-modal__label">通報理由</legend>

              <div className="report-modal__reason-list">
                {REPORT_REASONS.map((value) => {
                  const selected = reason === value;

                  return (
                    <Radio
                      key={value}
                      name="report-reason"
                      value={value}
                      checked={selected}
                      disabled={submitting}
                      label={getReportReasonLabel(value)}
                      className={[
                        "report-modal__reason",
                        selected ? "report-modal__reason--selected" : "",
                      ]
                        .filter(Boolean)
                        .join(" ")}
                      onChange={() => onReasonChange(value)}
                    />
                  );
                })}
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

            {error ? <TextState variant="error">{error}</TextState> : null}
          </ModalBody>

          <ModalFooter className="report-modal__actions">
            <Button
              variant="primary"
              size="md"
              disabled={!canSubmit || submitting}
              onClick={handleSubmit}
            >
              {submitting ? "送信中..." : "通報する"}
            </Button>
          </ModalFooter>
        </>
      )}
    </Modal>
  );
}