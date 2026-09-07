// frontend/admin/shell/src/features/report/presentation/model/reportLabels.ts

import type {
  ReportActorType,
  ReportCaseStatus,
  ReportReason,
  ReportTargetType,
} from "../../../../shared/type/report";

import type { TabTone } from "../../../../shared/ui/Tab/Tab";

export function getStatusLabel(
  status: ReportCaseStatus,
  targetType: ReportTargetType,
): string {
  switch (status) {
    case "PENDING":
      return "未対応";

    case "KEPT":
      return targetType === "AVATAR" ? "変化なし" : "維持";

    case "REMOVED":
      if (targetType === "AVATAR") {
        return "再販利用停止";
      }
      if (targetType === "TOKEN_BLUEPRINT") {
        return "非表示";
      }
      return "削除";

    default:
      return status;
  }
}

export function getStatusTone(
  status: ReportCaseStatus,
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
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "商品レビュー";

    case "TOKEN_BLUEPRINT":
      return "トークン";

    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";

    case "AVATAR":
      return "アバター";

    default:
      return targetType;
  }
}

export function getActorTypeLabel(
  actorType: ReportActorType,
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
  reason: ReportReason,
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
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "AVATAR":
      return "アバター名";

    case "TOKEN_BLUEPRINT":
      return "トークン名";

    default:
      return "タイトル";
  }
}

export function getSnapshotBodyLabel(
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "AVATAR":
      return "プロフィール";

    case "TOKEN_BLUEPRINT":
      return "説明";

    default:
      return "本文";
  }
}

export function getTargetParentLabel(
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "AVATAR":
      return "対象アバター";

    case "TOKEN_BLUEPRINT":
      return "対象トークン";

    default:
      return "親";
  }
}

export function getTargetAuthorTypeLabel(
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "AVATAR":
      return "対象種別";

    case "TOKEN_BLUEPRINT":
      return "作成者種別";

    default:
      return "投稿者種別";
  }
}

export function getTargetAuthorLabel(
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "AVATAR":
      return "対象アバター";

    case "TOKEN_BLUEPRINT":
      return "作成者";

    default:
      return "投稿者";
  }
}