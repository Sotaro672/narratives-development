// frontend/admin/shell/src/features/report/presentation/model/reportLabels.ts

import type {
  ReviewReportActorType,
  ReviewReportCaseStatus,
  ReviewReportReason,
  ReviewReportTargetType,
} from "../../../../shared/type/reviewReport";
import type { TabTone } from "../../../../shared/ui/Tab/Tab";

export function getStatusLabel(
  status: ReviewReportCaseStatus,
  targetType: ReviewReportTargetType,
): string {
  switch (status) {
    case "PENDING":
      return "未対応";
    case "KEPT":
      return targetType === "AVATAR" ? "変化なし" : "維持";
    case "REMOVED":
      return targetType === "AVATAR" ? "再販利用停止" : "削除";
    default:
      return status;
  }
}

export function getStatusTone(
  status: ReviewReportCaseStatus,
): TabTone {
  switch (status) {
    case "PENDING":
      return "warning";
    case "KEPT":
      return "success";
    case "REMOVED":
      return "danger";
    default:
      return "neutral";
  }
}

export function getTargetTypeLabel(
  targetType: ReviewReportTargetType,
): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "商品レビュー";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";
    case "AVATAR":
      return "アバター";
    default:
      return targetType;
  }
}

export function getActorTypeLabel(
  actorType: ReviewReportActorType,
): string {
  switch (actorType) {
    case "AVATAR":
      return "ユーザー";
    case "BRAND":
      return "ブランド";
    default:
      return actorType;
  }
}

export function getReasonLabel(
  reason: ReviewReportReason,
): string {
  switch (reason) {
    case "SPAM":
      return "スパム";
    case "HARASSMENT":
      return "嫌がらせ";
    case "INAPPROPRIATE":
      return "不適切な内容";
    case "FALSE_INFORMATION":
      return "虚偽情報";
    case "OTHER":
      return "その他";
    default:
      return reason;
  }
}

export function getSnapshotTitleLabel(
  targetType: ReviewReportTargetType,
): string {
  return targetType === "AVATAR"
    ? "アバター名"
    : "タイトル";
}

export function getSnapshotBodyLabel(
  targetType: ReviewReportTargetType,
): string {
  return targetType === "AVATAR"
    ? "プロフィール"
    : "本文";
}

export function getTargetParentLabel(
  targetType: ReviewReportTargetType,
): string {
  return targetType === "AVATAR"
    ? "対象アバター"
    : "親";
}

export function getTargetAuthorTypeLabel(
  targetType: ReviewReportTargetType,
): string {
  return targetType === "AVATAR"
    ? "対象種別"
    : "投稿者種別";
}

export function getTargetAuthorLabel(
  targetType: ReviewReportTargetType,
): string {
  return targetType === "AVATAR"
    ? "対象アバター"
    : "投稿者";
}