// frontend/admin/shell/src/features/report/infrastructure/reportApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type {
  ReviewReportCase,
  ReviewReportCaseListResponse,
  ReviewReportDecisionInput,
  ReviewReportDetailParams,
  ReviewReportDetailResponse,
  ReviewReportListParams,
} from "../../../shared/type/reviewReport";
import { appendPaginationParams } from "../../../shared/util/pagination";

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

function requireCaseId(caseId: string): string {
  const normalizedCaseId = caseId.trim();

  if (!normalizedCaseId) {
    throw new Error("caseId is required.");
  }

  return normalizedCaseId;
}

async function requireOk(response: Response, message: string): Promise<void> {
  if (response.ok) {
    return;
  }

  let detail = "";

  try {
    const body = await response.json() as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`${message} status=${response.status}${detail}`);
}

function appendStringParam(
  query: URLSearchParams,
  key: string,
  value: string | undefined,
): void {
  const normalizedValue = value?.trim();

  if (normalizedValue) {
    query.set(key, normalizedValue);
  }
}

export async function listReviewReports(
  params: ReviewReportListParams = {},
): Promise<ReviewReportCaseListResponse> {
  const backendBaseUrl = requireBackendBaseUrl();
  const query = new URLSearchParams();

  appendPaginationParams(query, params.page, params.perPage);
  appendStringParam(query, "status", params.status);
  appendStringParam(query, "targetType", params.targetType);
  appendStringParam(query, "targetId", params.targetId);
  appendStringParam(query, "targetParentId", params.targetParentId);
  appendStringParam(query, "targetAuthorId", params.targetAuthorId);
  appendStringParam(query, "targetAuthorType", params.targetAuthorType);
  appendStringParam(query, "sort", params.sort);
  appendStringParam(query, "order", params.order);

  const authHeaders = await getAuthHeaders();
  const queryString = query.toString();
  const url = `${backendBaseUrl}/admin/reports${queryString ? `?${queryString}` : ""}`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response, "Failed to load review reports.");

  return response.json() as Promise<ReviewReportCaseListResponse>;
}

export async function getReviewReport(
  caseId: string,
  params: ReviewReportDetailParams = {},
): Promise<ReviewReportDetailResponse> {
  const normalizedCaseId = requireCaseId(caseId);
  const backendBaseUrl = requireBackendBaseUrl();
  const query = new URLSearchParams();

  appendPaginationParams(query, params.page, params.perPage);
  appendStringParam(query, "reporterType", params.reporterType);
  appendStringParam(query, "reporterId", params.reporterId);
  appendStringParam(query, "companyId", params.companyId);
  appendStringParam(query, "reason", params.reason);
  appendStringParam(query, "sort", params.sort);
  appendStringParam(query, "order", params.order);

  const authHeaders = await getAuthHeaders();
  const queryString = query.toString();
  const url = `${backendBaseUrl}/admin/reports/${encodeURIComponent(normalizedCaseId)}${queryString ? `?${queryString}` : ""}`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response, "Failed to load review report.");

  return response.json() as Promise<ReviewReportDetailResponse>;
}

export async function decideReviewReport(
  caseId: string,
  input: ReviewReportDecisionInput,
): Promise<ReviewReportCase> {
  const normalizedCaseId = requireCaseId(caseId);
  const backendBaseUrl = requireBackendBaseUrl();
  const reason = input.reason.trim();

  if (!reason) {
    throw new Error("decision reason is required.");
  }

  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/reports/${encodeURIComponent(normalizedCaseId)}/decision`,
    {
      method: "POST",
      headers: {
        ...authHeaders,
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({
        decision: input.decision,
        reason,
      }),
    },
  );

  await requireOk(response, "Failed to decide review report.");

  return response.json() as Promise<ReviewReportCase>;
}