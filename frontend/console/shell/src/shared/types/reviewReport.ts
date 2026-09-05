// frontend/console/shell/src/shared/types/reviewReport.ts

export type ReviewReportTargetType =
  | "PRODUCT_BLUEPRINT_REVIEW"
  | "TOKEN_BLUEPRINT_COMMENT";

export type ReviewReportReason =
  | "SPAM"
  | "HARASSMENT"
  | "INAPPROPRIATE"
  | "FALSE_INFORMATION"
  | "OTHER";

export type ReviewReportCaseStatus =
  | "PENDING"
  | "KEPT"
  | "REMOVED";

export type ReviewReportRequest = {
  reason: ReviewReportReason;
  detail?: string;
};

export type ReviewReportResponse = {
  caseId: string;
  reportId: string;
  reportCount: number;
  status: ReviewReportCaseStatus;
  caseCreated: boolean;
  reportCreated: boolean;
};

export type ReportProductBlueprintReviewInput = {
  productBlueprintId: string;
  reviewId: string;
  reason: ReviewReportReason;
  detail?: string;
};

export type ReportTokenBlueprintCommentInput = {
  tokenBlueprintId: string;
  commentId: string;
  reason: ReviewReportReason;
  detail?: string;
};

export const REVIEW_REPORT_REASONS: readonly ReviewReportReason[] = [
  "SPAM",
  "HARASSMENT",
  "INAPPROPRIATE",
  "FALSE_INFORMATION",
  "OTHER",
];

export function getReviewReportReasonLabel(
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

export function requiresReviewReportDetail(
  reason: ReviewReportReason,
): boolean {
  return reason === "OTHER";
}