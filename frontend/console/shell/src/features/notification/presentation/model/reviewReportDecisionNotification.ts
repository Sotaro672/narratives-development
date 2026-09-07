// frontend/console/shell/src/features/notification/presentation/model/reviewReportDecisionNotification.ts

import type {
  ReviewReportDecisionNotification,
  ReviewReportDecisionStatus,
} from "../../infrastructure/reviewReportDecisionNotificationApi";
import {
  getReviewReportReasonLabel,
  type ReviewReportTargetType,
} from "../../../../shared/types/report";

export type ReviewReportDecisionNotificationViewModel = {
  id: string;
  category: "通報結果";
  title: string;
  body: string;
  targetLabel: string;
  reportReasonLabel: string;
  reportDetail: string;
  decisionStatus: ReviewReportDecisionStatus;
  decisionStatusLabel: string;
  decisionReason: string;
  occurredAt: string;
  readAt: string | null;
  isRead: boolean;
  notification: ReviewReportDecisionNotification;
};

export function getReviewReportDecisionStatusLabel(
  status: ReviewReportDecisionStatus,
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

export function getReviewReportDecisionNotificationTargetLabel(
  targetType: ReviewReportTargetType,
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

export function getReviewReportDecisionNotificationTitle(
  _notification: ReviewReportDecisionNotification,
): string {
  return "通報内容の確認が完了しました";
}

export function getReviewReportDecisionNotificationBody(
  notification: ReviewReportDecisionNotification,
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

export function toReviewReportDecisionNotificationViewModel(
  notification: ReviewReportDecisionNotification,
): ReviewReportDecisionNotificationViewModel {
  return {
    id: notification.id,
    category: "通報結果",
    title:
      getReviewReportDecisionNotificationTitle(
        notification,
      ),
    body:
      getReviewReportDecisionNotificationBody(
        notification,
      ),
    targetLabel:
      getReviewReportDecisionNotificationTargetLabel(
        notification.targetType,
      ),
    reportReasonLabel:
      getReviewReportReasonLabel(
        notification.reportReason,
      ),
    reportDetail: notification.reportDetail,
    decisionStatus: notification.decisionStatus,
    decisionStatusLabel:
      getReviewReportDecisionStatusLabel(
        notification.decisionStatus,
      ),
    decisionReason: notification.decisionReason,
    occurredAt:
      notification.decidedAt ||
      notification.createdAt,
    readAt: notification.readAt,
    isRead: notification.isRead,
    notification,
  };
}

export function toReviewReportDecisionNotificationViewModels(
  notifications: readonly ReviewReportDecisionNotification[],
): ReviewReportDecisionNotificationViewModel[] {
  return notifications
    .map(
      toReviewReportDecisionNotificationViewModel,
    )
    .sort(
      (a, b) =>
        toTimestamp(b.occurredAt) -
        toTimestamp(a.occurredAt),
    );
}

function toTimestamp(
  value: string,
): number {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp)
    ? 0
    : timestamp;
}