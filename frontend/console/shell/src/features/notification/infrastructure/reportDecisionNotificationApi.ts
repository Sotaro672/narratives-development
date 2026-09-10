// frontend/console/shell/src/features/notification/infrastructure/reportDecisionNotificationApi.ts

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type {
  PageParams,
  PageResult,
} from "../../../shared/types/common/common";
import type {
  ReportCaseStatus,
  ReportReason,
  ReportTargetType,
} from "../../../shared/types/report";

export type ReportDecisionNotificationRecipientType = "AVATAR" | "BRAND";

export type ReportDecisionNotificationKind =
  | "REPORTER_DECISION"
  | "TARGET_ENFORCEMENT";

export type ReportDecisionStatus = Exclude<
  ReportCaseStatus,
  "PENDING"
>;

export type ReportDecisionNotification = {
  id: string;
  notificationKind: ReportDecisionNotificationKind;
  caseId: string;
  reportId: string;
  recipientType: ReportDecisionNotificationRecipientType;
  recipientId: string;
  companyId: string;
  targetType: ReportTargetType;
  targetId: string;
  targetParentId: string;
  reportReason: ReportReason;
  reportDetail: string;
  decisionStatus: ReportDecisionStatus;
  decisionReason: string;
  decidedAt: string;
  createdAt: string;
  updatedAt: string;
  readAt: string | null;
  isRead: boolean;
};

export type ListReportDecisionNotificationsParams = PageParams & {
  isRead?: boolean;
};

export type ReportDecisionNotificationPage =
  PageResult<ReportDecisionNotification>;

type ErrorResponse = {
  error?: string;
  message?: string;
  detail?: string;
};

async function readJsonResponse<T>(
  response: Response,
  url: string,
): Promise<T> {
  const contentType = response.headers.get("content-type") ?? "";
  const text = await response.text().catch(() => "");

  if (!response.ok) {
    let message = "";

    if (contentType.includes("application/json") && text) {
      try {
        const errorResponse = JSON.parse(text) as ErrorResponse;
        message =
          errorResponse.error ??
          errorResponse.message ??
          errorResponse.detail ??
          "";
      } catch {
        message = text;
      }
    } else {
      message = text;
    }

    throw new Error(
      message ||
        `通知APIの呼び出しに失敗しました（${response.status} ${response.statusText}）`,
    );
  }

  if (!text) {
    throw new Error(`通知APIのレスポンスが空です。URL=${url}`);
  }

  if (!contentType.includes("application/json")) {
    throw new Error(
      `通知APIからJSON以外のレスポンスが返されました。URL=${url}`,
    );
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(
      `通知APIのJSONレスポンスを解析できませんでした。URL=${url}`,
    );
  }
}

async function requestJson<T>(
  url: string,
  init: RequestInit,
): Promise<T> {
  const authHeaders = await getAuthHeaders();

  const response = await fetch(url, {
    ...init,
    headers: {
      ...authHeaders,
      Accept: "application/json",
      ...init.headers,
    },
    credentials: "include",
  });

  return readJsonResponse<T>(response, url);
}

function buildListQuery(
  params?: ListReportDecisionNotificationsParams,
): string {
  const searchParams = new URLSearchParams();

  if (params?.isRead !== undefined) {
    searchParams.set("isRead", String(params.isRead));
  }

  if (params?.page !== undefined) {
    searchParams.set("page", String(params.page));
  }

  if (params?.perPage !== undefined) {
    searchParams.set("perPage", String(params.perPage));
  }

  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

/**
 * GET /report-decision-notifications
 */
export async function listReportDecisionNotificationsApi(
  params?: ListReportDecisionNotificationsParams,
): Promise<ReportDecisionNotificationPage> {
  const query = buildListQuery(params);
  const url = `${API_BASE}/report-decision-notifications${query}`;

  return requestJson<ReportDecisionNotificationPage>(url, {
    method: "GET",
  });
}

/**
 * GET /report-decision-notifications/{notificationId}
 */
export async function getReportDecisionNotificationApi(
  notificationId: string,
): Promise<ReportDecisionNotification> {
  const normalizedNotificationId = notificationId.trim();

  if (!normalizedNotificationId) {
    throw new Error("notificationId is required");
  }

  const encodedNotificationId = encodeURIComponent(
    normalizedNotificationId,
  );
  const url =
    `${API_BASE}/report-decision-notifications/` +
    encodedNotificationId;

  return requestJson<ReportDecisionNotification>(url, {
    method: "GET",
  });
}

/**
 * POST /report-decision-notifications/{notificationId}/read
 */
export async function markReportDecisionNotificationReadApi(
  notificationId: string,
): Promise<ReportDecisionNotification> {
  const normalizedNotificationId = notificationId.trim();

  if (!normalizedNotificationId) {
    throw new Error("notificationId is required");
  }

  const encodedNotificationId = encodeURIComponent(
    normalizedNotificationId,
  );
  const url =
    `${API_BASE}/report-decision-notifications/` +
    `${encodedNotificationId}/read`;

  return requestJson<ReportDecisionNotification>(url, {
    method: "POST",
  });
}

export const reportDecisionNotificationApi = {
  list: listReportDecisionNotificationsApi,
  get: getReportDecisionNotificationApi,
  markRead: markReportDecisionNotificationReadApi,
};