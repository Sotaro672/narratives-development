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
  category: "通報結果";
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
): string {
  switch (status) {
    case "REMOVED":
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
    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";
    default:
      return targetType;
  }
}

export function getReportDecisionNotificationTitle(
  _notification: ReportDecisionNotification,
): string {
  return "通報内容の確認が完了しました";
}

export function getReportDecisionNotificationBody(
  notification: ReportDecisionNotification,
): string {
  switch (notification.decisionStatus) {
    case "REMOVED":
      return "通報いただいた内容を確認し、対象コンテンツを非表示にしました。";
    case "KEPT":
      return "通報いただいた内容を確認しました。審査の結果、掲載を継続します。";
    default:
      return "通報いただいた内容の確認が完了しました。";
  }
}

export function toReportDecisionNotificationViewModel(
  notification: ReportDecisionNotification,
): ReportDecisionNotificationViewModel {
  return {
    id: notification.id,
    category: "通報結果",
    title: getReportDecisionNotificationTitle(notification),
    body: getReportDecisionNotificationBody(notification),
    targetLabel: getReportDecisionNotificationTargetLabel(notification.targetType),
    reportReasonLabel: getReportReasonLabel(notification.reportReason),
    reportDetail: notification.reportDetail,
    decisionStatus: notification.decisionStatus,
    decisionStatusLabel: getReportDecisionStatusLabel(notification.decisionStatus),
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
    .sort((a, b) => toTimestamp(b.occurredAt) - toTimestamp(a.occurredAt));
}

function toTimestamp(value: string): number {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}