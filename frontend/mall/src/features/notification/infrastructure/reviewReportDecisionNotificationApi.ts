// frontend/mall/src/features/notification/infrastructure/reviewReportDecisionNotificationApi.ts

import { HttpError, requestJson } from "../../../lib/http";
import { getOptionalAuthHeaders } from "../../../lib/authHeaders";

import type {
  ReviewReportCaseStatus,
  ReviewReportReason,
  ReviewReportTargetType,
} from "../../shared/types/reviewReport";

const REVIEW_REPORT_DECISION_NOTIFICATIONS_ENDPOINT =
  "/mall/me/review-report-decision-notifications";

export type ReviewReportDecisionNotificationRecipientType =
  | "AVATAR"
  | "BRAND";

export type ReviewReportDecisionNotificationKind =
  | "REPORTER_DECISION"
  | "TARGET_ENFORCEMENT";

export type ReviewReportDecisionStatus = Exclude<
  ReviewReportCaseStatus,
  "PENDING"
>;

type ReviewReportDecisionNotificationBase = {
  id: string;
  caseId: string;
  recipientType: ReviewReportDecisionNotificationRecipientType;
  recipientId: string;
  companyId: string;
  targetType: ReviewReportTargetType;
  targetId: string;
  targetParentId: string;
  decisionStatus: ReviewReportDecisionStatus;
  decisionReason: string;
  decidedAt: string;
  createdAt: string;
  updatedAt: string;
  readAt: string | null;
  isRead: boolean;
};

export type ReviewReportReporterDecisionNotification =
  ReviewReportDecisionNotificationBase & {
    notificationKind: "REPORTER_DECISION";
    reportId: string;
    reportReason: ReviewReportReason;
    reportDetail: string;
  };

export type ReviewReportTargetEnforcementNotification =
  ReviewReportDecisionNotificationBase & {
    notificationKind: "TARGET_ENFORCEMENT";
    reportId: "";
    reportReason: "";
    reportDetail: "";
  };

export type ReviewReportDecisionNotification =
  | ReviewReportReporterDecisionNotification
  | ReviewReportTargetEnforcementNotification;

export type ReviewReportDecisionNotificationPage = {
  items: ReviewReportDecisionNotification[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type FetchMeReviewReportDecisionNotificationsParams = {
  page?: number;
  perPage?: number;
  isRead?: boolean;
  signal?: AbortSignal;
};

function createEmptyPage(
  page: number,
  perPage: number,
): ReviewReportDecisionNotificationPage {
  return {
    items: [],
    totalCount: 0,
    totalPages: 0,
    page,
    perPage,
  };
}

function normalizeFiniteNumber(
  value: unknown,
  fallback: number,
): number {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : fallback;
}

export async function fetchMeReviewReportDecisionNotifications(
  params: FetchMeReviewReportDecisionNotificationsParams = {},
): Promise<ReviewReportDecisionNotificationPage> {
  const page = params.page ?? 1;
  const perPage = params.perPage ?? 20;

  const headers = await getOptionalAuthHeaders();
  if (!headers) {
    return createEmptyPage(page, perPage);
  }

  try {
    const json = await requestJson<
      Partial<ReviewReportDecisionNotificationPage>
    >(
      REVIEW_REPORT_DECISION_NOTIFICATIONS_ENDPOINT,
      {
        method: "GET",
        headers,
        query: {
          page,
          perPage,
          ...(params.isRead !== undefined
            ? { isRead: params.isRead }
            : {}),
        },
        signal: params.signal,
        cache: "no-store",
        messages: {
          requestErrorMessage:
            "failed to fetch review report decision notifications",
          nonJsonErrorMessage:
            "failed to fetch review report decision notifications: response is not json",
          invalidJsonErrorMessage:
            "failed to fetch review report decision notifications: invalid json",
        },
      },
    );

    return {
      items: Array.isArray(json.items) ? json.items : [],
      totalCount: Math.max(
        0,
        normalizeFiniteNumber(json.totalCount, 0),
      ),
      totalPages: Math.max(
        0,
        normalizeFiniteNumber(json.totalPages, 0),
      ),
      page: Math.max(
        1,
        normalizeFiniteNumber(json.page, page),
      ),
      perPage: Math.max(
        1,
        normalizeFiniteNumber(json.perPage, perPage),
      ),
    };
  } catch (error) {
    if (
      error instanceof HttpError &&
      (error.status === 401 || error.status === 403)
    ) {
      return createEmptyPage(page, perPage);
    }

    throw error;
  }
}

export async function markMeReviewReportDecisionNotificationRead(
  notificationId: string,
): Promise<ReviewReportDecisionNotification> {
  if (!notificationId) {
    throw new Error("notificationId is required");
  }

  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    throw new Error("authentication is required");
  }

  return requestJson<ReviewReportDecisionNotification>(
    `${REVIEW_REPORT_DECISION_NOTIFICATIONS_ENDPOINT}/${encodeURIComponent(
      notificationId,
    )}/read`,
    {
      method: "POST",
      headers,
      cache: "no-store",
      messages: {
        requestErrorMessage:
          "failed to mark review report decision notification as read",
        nonJsonErrorMessage:
          "failed to mark review report decision notification as read: response is not json",
        invalidJsonErrorMessage:
          "failed to mark review report decision notification as read: invalid json",
      },
    },
  );
}

export const reviewReportDecisionNotificationApi = {
  list: fetchMeReviewReportDecisionNotifications,
  markRead: markMeReviewReportDecisionNotificationRead,
};