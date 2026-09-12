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
      if (targetType === "LIST" || targetType === "RESALE") {
        return "出品停止";
      }
      if (targetType === "TOKEN_BLUEPRINT") {
        return "非表示";
      }
      if (targetType === "BRAND") {
        return "無効";
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

    case "LIST":
      return "出品";

    case "TOKEN_BLUEPRINT":
      return "トークン";

    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";

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

    case "BRAND":
      return "ブランド名";

    case "LIST":
      return "出品名";

    case "TOKEN_BLUEPRINT":
      return "トークン名";

    case "RESALE":
      return "商品";

    case "ANNOUNCEMENT":
      return "お知らせタイトル";

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

    case "BRAND":
      return "ブランド説明";

    case "LIST":
    case "TOKEN_BLUEPRINT":
      return "説明";

    case "RESALE":
      return "出品説明";

    case "TRADE_MESSAGE":
      return "メッセージ本文";

    case "ANNOUNCEMENT":
      return "お知らせ本文";

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

    case "BRAND":
      return "対象ブランド";

    case "LIST":
      return "対象出品";

    case "TOKEN_BLUEPRINT":
    case "ANNOUNCEMENT":
      return "対象トークン";

    case "RESALE":
      return "対象商品";

    case "TRADE_MESSAGE":
      return "取引ID";

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

    case "LIST":
    case "RESALE":
      return "出品者種別";

    case "TOKEN_BLUEPRINT":
      return "作成者種別";

    case "TRADE_MESSAGE":
      return "送信者種別";

    case "ANNOUNCEMENT":
      return "配信元種別";

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

    case "BRAND":
      return "ブランド";

    case "LIST":
      return "出品ブランド";

    case "TOKEN_BLUEPRINT":
      return "作成者";

    case "RESALE":
      return "出品者";

    case "TRADE_MESSAGE":
      return "送信者";

    case "ANNOUNCEMENT":
      return "配信ブランド";

    default:
      return "投稿者";
  }
}