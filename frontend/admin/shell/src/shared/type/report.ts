// frontend/admin/shell/src/shared/type/report.ts

export type ReportTargetType =
  | "PRODUCT_BLUEPRINT_REVIEW"
  | "TOKEN_BLUEPRINT_COMMENT"
  | "AVATAR";

export type ReportActorType =
  | "AVATAR"
  | "BRAND";

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

export type ReportDecision =
  | "KEEP"
  | "REMOVE";

export type ReportSortOrder =
  | "asc"
  | "desc";

export type ReportCaseSort =
  | "createdAt"
  | "updatedAt"
  | "decidedAt"
  | "reportCount"
  | "status";

export type ReportItemSort =
  | "createdAt"
  | "reason";

export type ReportCase = {
  id: string;
  targetType: ReportTargetType;
  targetId: string;
  targetParentId: string;
  targetParentName?: string;
  targetAuthorId: string;
  targetAuthorName?: string;
  targetAuthorType: ReportActorType;
  snapshotTitle: string;
  snapshotBody: string;
  snapshotRating: number | null;
  reportCount: number;
  status: ReportCaseStatus;
  createdAt: string;
  updatedAt: string;
  decidedAt: string | null;
  decidedBy: string;
  decisionReason: string;
};

export type ReportCaseListResponse = {
  items: ReportCase[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ReportItem = {
  id: string;
  caseId: string;
  reporterType: ReportActorType;
  reporterId: string;
  reporterName: string;
  companyId: string;
  companyName: string;
  reason: ReportReason;
  detail: string;
  createdAt: string;
};

export type ReportItemsPageResponse = {
  items: ReportItem[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ReportDetailResponse = {
  case: ReportCase;
  reports: ReportItemsPageResponse;
};

export type ReportListParams = {
  status?: ReportCaseStatus;
  targetType?: ReportTargetType;
  targetId?: string;
  targetParentId?: string;
  targetAuthorId?: string;
  targetAuthorType?: ReportActorType;
  sort?: ReportCaseSort;
  order?: ReportSortOrder;
  page?: number;
  perPage?: number;
};

export type ReportDetailParams = {
  reporterType?: ReportActorType;
  reporterId?: string;
  companyId?: string;
  reason?: ReportReason;
  sort?: ReportItemSort;
  order?: ReportSortOrder;
  page?: number;
  perPage?: number;
};

export type ReportDecisionInput = {
  decision: ReportDecision;
  reason: string;
};