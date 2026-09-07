// frontend/mall/src/features/report/api/reportApi.ts

import { getFirebaseIdToken } from "../../../lib/authToken";
import type {
  ReportAvatarInput,
  ReportProductBlueprintReviewInput,
  ReportRequest,
  ReportResponse,
  ReportTokenBlueprintCommentInput,
} from "../../shared/types/report";

type ReportErrorResponse = {
  error?: unknown;
  message?: unknown;
};

function getApiBaseUrl(): string {
  const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? "").trim();

  if (!apiBaseUrl) {
    throw new Error("VITE_API_BASE_URLが設定されていません。");
  }

  return apiBaseUrl.replace(/\/+$/, "");
}

function requireId(value: string, fieldName: string): string {
  const normalized = value.trim();

  if (!normalized) {
    throw new Error(`${fieldName}が指定されていません。`);
  }

  return normalized;
}

function createRequest(
  reason: ReportRequest["reason"],
  detail?: string,
): ReportRequest {
  const normalizedDetail = detail?.trim() ?? "";

  if (reason === "OTHER" && !normalizedDetail) {
    throw new Error("「その他」を選択した場合は詳細を入力してください。");
  }

  return {
    reason,
    ...(normalizedDetail ? { detail: normalizedDetail } : {}),
  };
}

async function readResponseBody(response: Response): Promise<unknown> {
  const text = await response.text();

  if (!text.trim()) {
    return null;
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    return text;
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

function getErrorMessage(body: unknown): string {
  if (typeof body === "string" && body.trim()) {
    return body.trim();
  }

  if (!body || typeof body !== "object") {
    return "";
  }

  const errorBody = body as ReportErrorResponse;

  if (typeof errorBody.error === "string" && errorBody.error.trim()) {
    return errorBody.error.trim();
  }

  if (typeof errorBody.message === "string" && errorBody.message.trim()) {
    return errorBody.message.trim();
  }

  return "";
}

function isReportResponse(value: unknown): value is ReportResponse {
  if (!value || typeof value !== "object") {
    return false;
  }

  const response = value as Partial<ReportResponse>;

  return (
    typeof response.caseId === "string" &&
    typeof response.reportId === "string" &&
    typeof response.reportCount === "number" &&
    (response.status === "PENDING" ||
      response.status === "KEPT" ||
      response.status === "REMOVED") &&
    typeof response.caseCreated === "boolean" &&
    typeof response.reportCreated === "boolean"
  );
}

async function postReport(
  path: string,
  request: ReportRequest,
): Promise<ReportResponse> {
  const apiBaseUrl = getApiBaseUrl();
  const idToken = await getFirebaseIdToken();

  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify(request),
  });

  const body = await readResponseBody(response);

  if (!response.ok) {
    const message = getErrorMessage(body);

    throw new Error(
      message || `通報の送信に失敗しました。status=${response.status}`,
    );
  }

  if (!isReportResponse(body)) {
    throw new Error("通報APIから不正なレスポンスが返されました。");
  }

  return body;
}

export async function reportProductBlueprintReview(
  input: ReportProductBlueprintReviewInput,
): Promise<ReportResponse> {
  const productBlueprintId = requireId(
    input.productBlueprintId,
    "productBlueprintId",
  );
  const reviewId = requireId(input.reviewId, "reviewId");
  const request = createRequest(input.reason, input.detail);

  return postReport(
    `/mall/me/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews/${encodeURIComponent(reviewId)}/reports`,
    request,
  );
}

export async function reportTokenBlueprintComment(
  input: ReportTokenBlueprintCommentInput,
): Promise<ReportResponse> {
  const tokenBlueprintId = requireId(
    input.tokenBlueprintId,
    "tokenBlueprintId",
  );
  const commentId = requireId(input.commentId, "commentId");
  const request = createRequest(input.reason, input.detail);

  return postReport(
    `/mall/me/token-blueprints/${encodeURIComponent(tokenBlueprintId)}/comments/${encodeURIComponent(commentId)}/reports`,
    request,
  );
}

export async function reportAvatar(
  input: ReportAvatarInput,
): Promise<ReportResponse> {
  const avatarId = requireId(input.avatarId, "avatarId");
  const request = createRequest(input.reason, input.detail);

  return postReport(
    `/mall/me/avatars/${encodeURIComponent(avatarId)}/reports`,
    request,
  );
}