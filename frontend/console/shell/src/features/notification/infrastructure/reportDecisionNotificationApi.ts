// frontend\console\shell\src\features\notification\infrastructure\ReportDecisionNotificationApi.ts

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { PageParams, PageResult } from "../../../shared/types/common/common";
import type {
  ReportCaseStatus,
  ReportReason,
  ReportTargetType,
} from "../../../shared/types/report";

export type ReportDecisionNotificationRecipientType = "AVATAR" | "BRAND";

export type ReportDecisionStatus = Exclude<ReportCaseStatus, "PENDING">;

export type ReportDecisionNotification = {
  id: string;
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

export type ReportDecisionNotificationPage = PageResult<ReportDecisionNotification>;

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
 *
 * ログイン中メンバーのCompanyに属するBRAND宛て裁定結果通知を取得する。
 * companyIdはFrontendから送らず、Backendの認証コンテキストから解決する。
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
 * POST /report-decision-notifications/{notificationId}/read
 *
 * 指定した裁定結果通知を既読にする。
 * Backend側でCompany所有権を再検証する。
 */
export async function markReportDecisionNotificationReadApi(
  notificationId: string,
): Promise<ReportDecisionNotification> {
  if (!notificationId) {
    throw new Error("notificationId is required");
  }

  const encodedNotificationId = encodeURIComponent(notificationId);
  const url =
    `${API_BASE}/report-decision-notifications/` +
    `${encodedNotificationId}/read`;

  return requestJson<ReportDecisionNotification>(url, {
    method: "POST",
  });
}

export const reportDecisionNotificationApi = {
  list: listReportDecisionNotificationsApi,
  markRead: markReportDecisionNotificationReadApi,
};