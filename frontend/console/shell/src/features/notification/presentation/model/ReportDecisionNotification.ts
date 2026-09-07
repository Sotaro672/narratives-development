// frontend/console/shell/src/features/notification/presentation/model/reportDecisionNotification.ts

import type {
  ReportDecisionNotification,
  ReportDecisionStatus,
} from "../../infrastructure/reportDecisionNotificationApi";

import {
  getReportReasonLabel,
  type ReportTargetType,
} from "../../../../shared/types/report";

export type ReportDecisionNotificationViewModel = {
  id: string;
  category: "通報結果" | "措置通知";
  title: string;
  body: string;
  targetLabel: string;
  reportReasonLabel: string;
  reportDetail: string;
  decisionStatus: ReportDecisionStatus;
  decisionStatusLabel: string;
  decisionReason: string;
  occurredAt: string;
  readAt: string | null;
  isRead: boolean;
  notification: ReportDecisionNotification;
};

export function getReportDecisionStatusLabel(
  status: ReportDecisionStatus,
  targetType?: ReportTargetType,
): string {
  switch (status) {
    case "REMOVED":
      if (targetType === "LIST") {
        return "出品停止";
      }

      if (targetType === "AVATAR") {
        return "再販利用停止";
      }

      return "非表示";

    case "KEPT":
      return "掲載継続";

    default:
      return status;
  }
}

export function getReportDecisionNotificationTargetLabel(
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

    default:
      return targetType;
  }
}

export function getReportDecisionNotificationTitle(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    switch (notification.targetType) {
      case "LIST":
        return "出品が停止されました";

      case "TOKEN_BLUEPRINT":
        return "トークンがAMOL上で非表示になりました";

      case "PRODUCT_BLUEPRINT_REVIEW":
        return "投稿したレビューに措置が行われました";

      case "AVATAR":
        return "アカウントに措置が行われました";

      default:
        return "対象コンテンツに措置が行われました";
    }
  }

  return "通報内容の確認が完了しました";
}

export function getReportDecisionNotificationBody(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    switch (notification.targetType) {
      case "LIST":
        return "審査の結果、対象出品を停止しました。Listおよび登録済み画像は削除されていません。";

      case "TOKEN_BLUEPRINT":
        return "審査の結果、対象トークンをAMOL上で非表示にしました。TokenBlueprintおよびオンチェーン上のトークン・メタデータは削除されていません。";

      case "PRODUCT_BLUEPRINT_REVIEW":
        return "審査の結果、対象レビューを非表示にしました。";

      case "AVATAR":
        return "審査の結果、対象アバターの再販サービス利用を停止しました。";

      default:
        return "審査の結果、対象コンテンツに措置を行いました。";
    }
  }

  if (notification.targetType === "LIST") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象出品を停止しました。";

      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象出品の掲載を継続します。";

      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  if (notification.targetType === "AVATAR") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象アバターの再販サービス利用を停止しました。";

      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象アバターへの変更は行いませんでした。";

      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  switch (notification.decisionStatus) {
    case "REMOVED":
      return "通報いただいた内容を確認し、対象コンテンツを非表示にしました。";

    case "KEPT":
      return "通報いただいた内容を確認しました。審査の結果、掲載を継続します。";

    default:
      return "通報いただいた内容の確認が完了しました。";
  }
}

function getReportDecisionNotificationCategory(
  notification: ReportDecisionNotification,
): ReportDecisionNotificationViewModel["category"] {
  return notification.notificationKind === "TARGET_ENFORCEMENT"
    ? "措置通知"
    : "通報結果";
}

function getReportDecisionNotificationReasonLabel(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return "";
  }

  return getReportReasonLabel(notification.reportReason);
}

export function toReportDecisionNotificationViewModel(
  notification: ReportDecisionNotification,
): ReportDecisionNotificationViewModel {
  return {
    id: notification.id,
    category: getReportDecisionNotificationCategory(notification),
    title: getReportDecisionNotificationTitle(notification),
    body: getReportDecisionNotificationBody(notification),
    targetLabel: getReportDecisionNotificationTargetLabel(
      notification.targetType,
    ),
    reportReasonLabel:
      getReportDecisionNotificationReasonLabel(notification),
    reportDetail:
      notification.notificationKind === "TARGET_ENFORCEMENT"
        ? ""
        : notification.reportDetail,
    decisionStatus: notification.decisionStatus,
    decisionStatusLabel: getReportDecisionStatusLabel(
      notification.decisionStatus,
      notification.targetType,
    ),
    decisionReason: notification.decisionReason,
    occurredAt: notification.decidedAt || notification.createdAt,
    readAt: notification.readAt,
    isRead: notification.isRead,
    notification,
  };
}

export function toReportDecisionNotificationViewModels(
  notifications: readonly ReportDecisionNotification[],
): ReportDecisionNotificationViewModel[] {
  return notifications
    .map(toReportDecisionNotificationViewModel)
    .sort(
      (a, b) =>
        toTimestamp(b.occurredAt) - toTimestamp(a.occurredAt),
    );
}

function toTimestamp(value: string): number {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}