// frontend/mall/src/features/notification/infrastructure/reportDecisionNotificationApi.ts

import { HttpError, requestJson } from "../../../lib/http";
import { getOptionalAuthHeaders } from "../../../lib/authHeaders";

import type {
  ReportCaseStatus,
  ReportReason,
  ReportTargetType,
} from "../../shared/types/report";

const REPORT_DECISION_NOTIFICATIONS_ENDPOINT =
  "/mall/me/report-decision-notifications";

export type ReportDecisionNotificationRecipientType =
  | "AVATAR"
  | "BRAND";

export type ReportDecisionNotificationKind =
  | "REPORTER_DECISION"
  | "TARGET_ENFORCEMENT";

export type ReportDecisionStatus = Exclude<
  ReportCaseStatus,
  "PENDING"
>;

type ReportDecisionNotificationBase = {
  id: string;
  caseId: string;
  recipientType: ReportDecisionNotificationRecipientType;
  recipientId: string;
  companyId: string;
  targetType: ReportTargetType;
  targetId: string;
  targetParentId: string;
  decisionStatus: ReportDecisionStatus;
  decisionReason: string;
  decidedAt: string;
  createdAt: string;
  updatedAt: string;
  readAt: string | null;
  isRead: boolean;
};

export type ReportReporterDecisionNotification =
  ReportDecisionNotificationBase & {
    notificationKind: "REPORTER_DECISION";
    reportId: string;
    reportReason: ReportReason;
    reportDetail: string;
  };

export type ReportTargetEnforcementNotification =
  ReportDecisionNotificationBase & {
    notificationKind: "TARGET_ENFORCEMENT";
    reportId: "";
    reportReason: "";
    reportDetail: "";
  };

export type ReportDecisionNotification =
  | ReportReporterDecisionNotification
  | ReportTargetEnforcementNotification;

export type ReportDecisionNotificationPage = {
  items: ReportDecisionNotification[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type FetchMeReportDecisionNotificationsParams = {
  page?: number;
  perPage?: number;
  isRead?: boolean;
  signal?: AbortSignal;
};

function createEmptyPage(
  page: number,
  perPage: number,
): ReportDecisionNotificationPage {
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

export async function fetchMeReportDecisionNotifications(
  params: FetchMeReportDecisionNotificationsParams = {},
): Promise<ReportDecisionNotificationPage> {
  const page = params.page ?? 1;
  const perPage = params.perPage ?? 20;

  const headers = await getOptionalAuthHeaders();
  if (!headers) {
    return createEmptyPage(page, perPage);
  }

  try {
    const json = await requestJson<
      Partial<ReportDecisionNotificationPage>
    >(
      REPORT_DECISION_NOTIFICATIONS_ENDPOINT,
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
            "failed to fetch report decision notifications",
          nonJsonErrorMessage:
            "failed to fetch report decision notifications: response is not json",
          invalidJsonErrorMessage:
            "failed to fetch report decision notifications: invalid json",
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

export async function markMeReportDecisionNotificationRead(
  notificationId: string,
): Promise<ReportDecisionNotification> {
  if (!notificationId) {
    throw new Error("notificationId is required");
  }

  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    throw new Error("authentication is required");
  }

  return requestJson<ReportDecisionNotification>(
    `${REPORT_DECISION_NOTIFICATIONS_ENDPOINT}/${encodeURIComponent(
      notificationId,
    )}/read`,
    {
      method: "POST",
      headers,
      cache: "no-store",
      messages: {
        requestErrorMessage:
          "failed to mark report decision notification as read",
        nonJsonErrorMessage:
          "failed to mark report decision notification as read: response is not json",
        invalidJsonErrorMessage:
          "failed to mark report decision notification as read: invalid json",
      },
    },
  );
}

export const reportDecisionNotificationApi = {
  list: fetchMeReportDecisionNotifications,
  markRead: markMeReportDecisionNotificationRead,
};