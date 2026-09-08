// frontend/mall/src/features/notification/presentation/model/reportDecisionPresentation.ts

import type { ReportDecisionNotification } from "../../infrastructure/reportDecisionNotificationApi";
import { getReportReasonLabel, type ReportTargetType } from "../../../shared/types/report";

export type ReportDecisionPresentation = {
  targetLabel: string;
  cardLabel: string;
  title: string;
  body: string;
  statusLabel: string;
  occurredAt: string;
  isReporterDecision: boolean;
  reportReasonLabel: string;
};

export function getReportTargetLabel(targetType: ReportTargetType): string {
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
    case "RESALE":
      return "再販出品";
    default:
      return "投稿内容";
  }
}

export function getDecisionCardLabel(
  notification: ReportDecisionNotification,
  targetLabel: string,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return `運営からのお知らせ・${targetLabel}`;
  }

  return `通報結果・${targetLabel}`;
}

export function getDecisionCardTitle(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return "運営による措置のお知らせ";
  }

  return "通報内容の確認が完了しました";
}

export function getDecisionBody(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    switch (notification.targetType) {
      case "PRODUCT_BLUEPRINT_REVIEW":
        return "運営の裁定により、あなたの商品レビューを削除しました。";
      case "LIST":
        return "運営の裁定により、対象出品を停止しました。Listおよび登録済み画像は削除されていません。";
      case "TOKEN_BLUEPRINT":
        return "運営の裁定により、対象トークンをAMOL上で非表示にしました。オンチェーン上のトークンやメタデータは削除されていません。";
      case "AVATAR":
        return "運営の裁定により、再販サービスの利用を停止しました。";
      case "RESALE":
        return "運営の裁定により、対象の再販出品を停止しました。Resaleおよび登録済み画像は削除されていません。";
      case "TOKEN_BLUEPRINT_COMMENT":
        return "運営の裁定により、あなたのトークンコメントを削除しました。";
      default:
        return "運営の裁定により、対象コンテンツに措置を行いました。";
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

  if (notification.targetType === "RESALE") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象の再販出品を停止しました。";
      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象の再販出品の掲載を継続します。";
      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  if (notification.targetType === "TOKEN_BLUEPRINT") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象トークンをAMOL上で非表示にしました。オンチェーン上のトークンやメタデータは削除されていません。";
      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象トークンを維持します。";
      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  switch (notification.decisionStatus) {
    case "REMOVED":
      return "通報いただいた内容を確認し、対象コンテンツを削除しました。";
    case "KEPT":
      return "通報いただいた内容を確認しました。審査の結果、対象コンテンツを維持します。";
    default:
      return "通報いただいた内容の確認が完了しました。";
  }
}

export function getDecisionStatusLabel(
  notification: ReportDecisionNotification,
): string {
  if (notification.targetType === "AVATAR") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "再販利用停止";
      case "KEPT":
        return "変化なし";
      default:
        return notification.decisionStatus;
    }
  }

  if (
    notification.targetType === "LIST" ||
    notification.targetType === "RESALE"
  ) {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "出品停止";
      case "KEPT":
        return "維持";
      default:
        return notification.decisionStatus;
    }
  }

  if (notification.targetType === "TOKEN_BLUEPRINT") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "非表示";
      case "KEPT":
        return "維持";
      default:
        return notification.decisionStatus;
    }
  }

  switch (notification.decisionStatus) {
    case "REMOVED":
      return "削除";
    case "KEPT":
      return "維持";
    default:
      return notification.decisionStatus;
  }
}

export function createReportDecisionPresentation(
  notification: ReportDecisionNotification,
): ReportDecisionPresentation {
  const targetLabel = getReportTargetLabel(notification.targetType);
  const isReporterDecision =
    notification.notificationKind === "REPORTER_DECISION";

  return {
    targetLabel,
    cardLabel: getDecisionCardLabel(notification, targetLabel),
    title: getDecisionCardTitle(notification),
    body: getDecisionBody(notification),
    statusLabel: getDecisionStatusLabel(notification),
    occurredAt: notification.decidedAt || notification.createdAt,
    isReporterDecision,
    reportReasonLabel: isReporterDecision
      ? getReportReasonLabel(notification.reportReason)
      : "",
  };
}