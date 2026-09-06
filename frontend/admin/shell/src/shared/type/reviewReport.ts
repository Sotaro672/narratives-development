// frontend/admin/shell/src/shared/type/reviewReport.ts

export type ReviewReportTargetType =
  | "PRODUCT_BLUEPRINT_REVIEW"
  | "TOKEN_BLUEPRINT_COMMENT"
  | "AVATAR";

export type ReviewReportActorType =
  | "AVATAR"
  | "BRAND";

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

export type ReviewReportDecision =
  | "KEEP"
  | "REMOVE";

export type ReviewReportSortOrder =
  | "asc"
  | "desc";

export type ReviewReportCaseSort =
  | "createdAt"
  | "updatedAt"
  | "decidedAt"
  | "reportCount"
  | "status";

export type ReviewReportItemSort =
  | "createdAt"
  | "reason";

export type ReviewReportCase = {
  id: string;
  targetType: ReviewReportTargetType;
  targetId: string;
  targetParentId: string;
  targetParentName?: string;
  targetAuthorId: string;
  targetAuthorName?: string;
  targetAuthorType: ReviewReportActorType;
  snapshotTitle: string;
  snapshotBody: string;
  snapshotRating: number | null;
  reportCount: number;
  status: ReviewReportCaseStatus;
  createdAt: string;
  updatedAt: string;
  decidedAt: string | null;
  decidedBy: string;
  decisionReason: string;
};

export type ReviewReportCaseListResponse = {
  items: ReviewReportCase[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ReviewReportItem = {
  id: string;
  caseId: string;
  reporterType: ReviewReportActorType;
  reporterId: string;
  reporterName: string;
  companyId: string;
  companyName: string;
  reason: ReviewReportReason;
  detail: string;
  createdAt: string;
};

export type ReviewReportItemsPageResponse = {
  items: ReviewReportItem[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ReviewReportDetailResponse = {
  case: ReviewReportCase;
  reports: ReviewReportItemsPageResponse;
};

export type ReviewReportListParams = {
  status?: ReviewReportCaseStatus;
  targetType?: ReviewReportTargetType;
  targetId?: string;
  targetParentId?: string;
  targetAuthorId?: string;
  targetAuthorType?: ReviewReportActorType;
  sort?: ReviewReportCaseSort;
  order?: ReviewReportSortOrder;
  page?: number;
  perPage?: number;
};

export type ReviewReportDetailParams = {
  reporterType?: ReviewReportActorType;
  reporterId?: string;
  companyId?: string;
  reason?: ReviewReportReason;
  sort?: ReviewReportItemSort;
  order?: ReviewReportSortOrder;
  page?: number;
  perPage?: number;
};

export type ReviewReportDecisionInput = {
  decision: ReviewReportDecision;
  reason: string;
};

export type ReviewReportDecisionResponse = ReviewReportCase;