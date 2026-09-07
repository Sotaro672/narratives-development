// frontend/console/shell/src/shared/types/report.ts

export type ReportTargetType =
  | "PRODUCT_BLUEPRINT_REVIEW"
  | "TOKEN_BLUEPRINT_COMMENT";

export type ReportReason =
  | "SPAM"
  | "HARASSMENT"
  | "INAPPROPRIATE"
  | "FALSE_INFORMATION"
  | "OTHER";

export type ReportCaseStatus =
  | "PENDING"
  | "KEPT"
  | "REMOVED";

export type ReportRequest = {
  reason: ReportReason;
  detail?: string;
};

export type ReportResponse = {
  caseId: string;
  reportId: string;
  reportCount: number;
  status: ReportCaseStatus;
  caseCreated: boolean;
  reportCreated: boolean;
};

export type ReportProductBlueprintReviewInput = {
  productBlueprintId: string;
  reviewId: string;
  reason: ReportReason;
  detail?: string;
};

export type ReportTokenBlueprintCommentInput = {
  tokenBlueprintId: string;
  commentId: string;
  reason: ReportReason;
  detail?: string;
};

export const REPORT_REASONS: readonly ReportReason[] = [
  "SPAM",
  "HARASSMENT",
  "INAPPROPRIATE",
  "FALSE_INFORMATION",
  "OTHER",
];

export function getReportReasonLabel(
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

export function requiresReportDetail(
  reason: ReportReason,
): boolean {
  return reason === "OTHER";
}