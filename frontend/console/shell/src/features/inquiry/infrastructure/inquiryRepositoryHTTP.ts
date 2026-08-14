// frontend/console/shell/src/features/inquiry/infrastructure/inquiryRepositoryHTTP.ts

import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { storage } from "../../../auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../../../shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../shared/http/authHeaders";

import type {
  CountUnreadInquiriesParams,
  Inquiry,
  InquiryDetail,
  InquiryImageFile,
  InquiryManagementItem,
  InquiryPageResult,
  InquiryReply,
  InquiryUnreadCountResult,
  ListInquiriesParams,
  ReplyInquiryParams,
} from "../../../shared/types/inquiry";

type UploadInquiryReplyImageParams = {
  inquiryId: string;
  memberId: string;
  file: File;
};

export type UploadInquiryReplyImagesParams = {
  inquiryId: string;
  memberId: string;
  files: File[];
};

// -----------------------------------------------------------
// internal helpers
// -----------------------------------------------------------

function assertID(id: string, label: string): string {
  const trimmed = String(id ?? "").trim();

  if (!trimmed) {
    throw new Error(`inquiryRepositoryHTTP: ${label} が空です`);
  }

  return trimmed;
}

function appendStringParam(
  params: URLSearchParams,
  key: string,
  value: unknown,
): void {
  const trimmed = String(value ?? "").trim();

  if (trimmed) {
    params.set(key, trimmed);
  }
}

function appendBooleanParam(
  params: URLSearchParams,
  key: string,
  value: boolean | undefined,
): void {
  if (typeof value === "boolean") {
    params.set(key, value ? "true" : "false");
  }
}

function buildInquiryListQuery(params: ListInquiriesParams): string {
  const query = new URLSearchParams();

  appendStringParam(query, "searchQuery", params.searchQuery);
  appendStringParam(query, "productId", params.productId);
  appendStringParam(query, "avatarId", params.avatarId);
  appendStringParam(query, "status", params.status);
  appendStringParam(query, "inquiryType", params.inquiryType);
  appendStringParam(query, "updatedBy", params.updatedBy);
  appendStringParam(query, "deletedBy", params.deletedBy);
  appendStringParam(query, "resolvedBy", params.resolvedBy);
  appendStringParam(query, "closedBy", params.closedBy);
  appendStringParam(query, "imageFileName", params.imageFileName);

  appendBooleanParam(query, "deleted", params.deleted);
  appendBooleanParam(query, "resolved", params.resolved);
  appendBooleanParam(query, "closed", params.closed);

  const queryString = query.toString();

  return queryString ? `?${queryString}` : "";
}

async function readErrorDetail(response: Response): Promise<string> {
  return response.text().catch(() => "");
}

function sanitizeFileName(fileName: string): string {
  const sanitized = String(fileName ?? "")
    .trim()
    .replace(/[\\/:*?"<>|#%{}[\]`^~]/g, "_")
    .replace(/\s+/g, "_")
    .slice(0, 120);

  return sanitized || "image";
}

function createClientID(prefix: string): string {
  const randomID =
    globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random().toString(36).slice(2)}`;

  return `${prefix}_${randomID}`;
}

function assertImageFile(file: File): File {
  if (!file) {
    throw new Error("inquiryRepositoryHTTP: file が空です");
  }

  if (!file.type.startsWith("image/")) {
    throw new Error("画像ファイルのみアップロードできます");
  }

  return file;
}

// -----------------------------------------------------------
// Firebase Storage: Inquiry reply 画像アップロード
//
// Storage path:
//   inquiry-replies/{inquiryId}/{imageId}/{fileName}
//
// Firestore 保存用 metadata:
//   backend の POST /inquiries/{id}/reply の images に渡す。
// -----------------------------------------------------------

async function uploadInquiryReplyImageToStorage(
  params: UploadInquiryReplyImageParams,
): Promise<InquiryImageFile> {
  const inquiryId = assertID(params.inquiryId, "inquiryId");
  const memberId = assertID(params.memberId, "memberId");
  const file = assertImageFile(params.file);

  const imageId = createClientID("reply_image");
  const fileName = sanitizeFileName(file.name);
  const mimeType = file.type;
  const objectPath = `inquiry-replies/${encodeURIComponent(
    inquiryId,
  )}/${encodeURIComponent(imageId)}/${encodeURIComponent(fileName)}`;

  const storageRef = ref(storage, objectPath);

  await uploadBytes(storageRef, file, {
    contentType: mimeType,
    customMetadata: {
      inquiryId,
      createdBy: memberId,
    },
  });

  const fileUrl = await getDownloadURL(storageRef);

  return {
    inquiryId,
    fileName,
    fileUrl,
    objectPath,
    fileSize: file.size,
    mimeType,
    createdAt: new Date().toISOString(),
    createdBy: memberId,
  };
}

export async function uploadInquiryReplyImagesToStorage(
  params: UploadInquiryReplyImagesParams,
): Promise<InquiryImageFile[]> {
  const inquiryId = assertID(params.inquiryId, "inquiryId");
  const memberId = assertID(params.memberId, "memberId");
  const files = params.files;

  if (files.length === 0) {
    return [];
  }

  return Promise.all(
    files.map((file) =>
      uploadInquiryReplyImageToStorage({
        inquiryId,
        memberId,
        file,
      }),
    ),
  );
}

// -----------------------------------------------------------
// GET: Inquiry 一覧
//   backend: GET /inquiries/company/{companyId}
// -----------------------------------------------------------

export async function listInquiriesHTTP(
  params: ListInquiriesParams,
): Promise<InquiryPageResult<InquiryManagementItem>> {
  const companyId = assertID(params.companyId, "companyId");
  const headers = await getAuthHeadersOrThrow();
  const query = buildInquiryListQuery(params);
  const url = `${API_BASE}/inquiries/company/${encodeURIComponent(
    companyId,
  )}${query}`;

  const response = await fetch(url, {
    method: "GET",
    headers,
  });

  if (!response.ok) {
    const detail = await readErrorDetail(response);

    throw new Error(
      `問い合わせ一覧の取得に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return (await response.json()) as InquiryPageResult<InquiryManagementItem>;
}

// -----------------------------------------------------------
// GET: Inquiry 未読件数
//   backend: GET /inquiries/company/{companyId}/unread-count
// -----------------------------------------------------------

export async function countUnreadInquiriesHTTP(
  params: CountUnreadInquiriesParams,
): Promise<InquiryUnreadCountResult> {
  const companyId = assertID(params.companyId, "companyId");
  const headers = await getAuthHeadersOrThrow();
  const query = buildInquiryListQuery(params);
  const url = `${API_BASE}/inquiries/company/${encodeURIComponent(
    companyId,
  )}/unread-count${query}`;

  const response = await fetch(url, {
    method: "GET",
    headers,
  });

  if (!response.ok) {
    const detail = await readErrorDetail(response);

    throw new Error(
      `問い合わせ未読件数の取得に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return (await response.json()) as InquiryUnreadCountResult;
}

// -----------------------------------------------------------
// GET: Inquiry 詳細
//   backend: GET /inquiries/{id}
// -----------------------------------------------------------

export async function getInquiryHTTP(id: string): Promise<InquiryDetail> {
  const inquiryId = assertID(id, "id");
  const headers = await getAuthHeadersOrThrow();

  const response = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(inquiryId)}`,
    {
      method: "GET",
      headers,
    },
  );

  if (!response.ok) {
    const detail = await readErrorDetail(response);

    throw new Error(
      `問い合わせ詳細の取得に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return (await response.json()) as InquiryDetail;
}

// -----------------------------------------------------------
// POST: Inquiry を resolved にする
//   backend: POST /inquiries/{id}/resolve
//   memberId は backend の認証 context を正とする。
// -----------------------------------------------------------

export async function resolveInquiryHTTP(id: string): Promise<Inquiry> {
  const inquiryId = assertID(id, "id");
  const headers = await getAuthHeadersOrThrow();

  const response = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(inquiryId)}/resolve`,
    {
      method: "POST",
      headers,
    },
  );

  if (!response.ok) {
    const detail = await readErrorDetail(response);

    throw new Error(
      `問い合わせの対応済み更新に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return (await response.json()) as Inquiry;
}

// -----------------------------------------------------------
// POST: Inquiry を open に戻す
//   backend: POST /inquiries/{id}/reopen
//   memberId は backend の認証 context を正とする。
// -----------------------------------------------------------

export async function reopenInquiryHTTP(id: string): Promise<Inquiry> {
  const inquiryId = assertID(id, "id");
  const headers = await getAuthHeadersOrThrow();

  const response = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(inquiryId)}/reopen`,
    {
      method: "POST",
      headers,
    },
  );

  if (!response.ok) {
    const detail = await readErrorDetail(response);

    throw new Error(
      `問い合わせの再オープンに失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return (await response.json()) as Inquiry;
}

// -----------------------------------------------------------
// POST: Inquiry 返信
//   backend: POST /inquiries/{id}/reply
//   memberId は backend の認証 context を正とする。
//   content または images のどちらかがあれば送信可能。
// -----------------------------------------------------------

export async function replyInquiryHTTP(
  id: string,
  params: ReplyInquiryParams,
): Promise<InquiryReply> {
  const inquiryId = assertID(id, "id");
  const headers = await getAuthJsonHeadersOrThrow();

  const response = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(inquiryId)}/reply`,
    {
      method: "POST",
      headers,
      body: JSON.stringify({
        content: params.content,
        images: params.images,
      }),
    },
  );

  if (!response.ok) {
    const detail = await readErrorDetail(response);

    throw new Error(
      `問い合わせ返信の送信に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return (await response.json()) as InquiryReply;
}