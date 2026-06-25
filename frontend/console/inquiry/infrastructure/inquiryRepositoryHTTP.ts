// frontend/console/inquiry/infrastructure/inquiryRepositoryHTTP.ts

import { API_BASE } from "../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../shell/src/shared/http/authHeaders";

// -----------------------------------------------------------
// Types
// -----------------------------------------------------------

export type InquiryStatus = string;
export type InquiryType = string;

export type InquiryImageFile = {
  inquiryId?: string;
  fileName: string;
  fileUrl: string;
  objectPath?: string | null;
  fileSize: number;
  mimeType: string;
  createdAt?: string;
  createdBy?: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
};

export type Inquiry = {
  id: string;
  productId: string;
  avatarId: string;
  subject: string;
  content: string;
  status: InquiryStatus;
  inquiryType: InquiryType;
  isRead?: boolean;
  images?: InquiryImageFile[];

  createdAt?: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
  deletedAt?: string | null;
  deletedBy?: string | null;

  resolvedAt?: string | null;
  resolvedBy?: string | null;
  closedAt?: string | null;
  closedBy?: string | null;
};

export type InquiryManagementItem = {
  inquiry: Inquiry;
  modelId: string;
  productBlueprintId: string;
  companyId: string;
};

export type InquiryDetail = {
  inquiry: Inquiry;
  modelId: string;
  productBlueprintId: string;
  companyId: string;
};

export type InquiryAggregate = {
  inquiry: Inquiry;
  images: InquiryImageFile[];
  modelId: string;
  productBlueprintId: string;
  companyId: string;
};

export type InquiryPageResult<T> = {
  items: T[];
};

export type ListInquiriesParams = {
  companyId: string;

  searchQuery?: string;
  productId?: string;
  avatarId?: string;
  status?: InquiryStatus;
  inquiryType?: InquiryType;
  updatedBy?: string;
  deletedBy?: string;
  resolvedBy?: string;
  closedBy?: string;
  imageFileName?: string;

  deleted?: boolean;
  resolved?: boolean;
  closed?: boolean;
};

export type AddInquiryImageParams = {
  fileName: string;
  fileUrl: string;
  objectPath?: string | null;
  fileSize: number;
  mimeType: string;
  createdAt?: string | null;
  createdBy?: string | null;
};

export type ResolveInquiryParams = {
  memberId: string;
};

export type ReopenInquiryParams = {
  memberId: string;
};

// -----------------------------------------------------------
// internal helpers
// -----------------------------------------------------------

function assertID(id: string, label: string) {
  const trimmed = String(id ?? "").trim();
  if (!trimmed) {
    throw new Error(`inquiryRepositoryHTTP: ${label} が空です`);
  }
  return trimmed;
}

function appendStringParam(params: URLSearchParams, key: string, value: unknown) {
  const trimmed = String(value ?? "").trim();
  if (trimmed) {
    params.set(key, trimmed);
  }
}

function appendBooleanParam(
  params: URLSearchParams,
  key: string,
  value: boolean | undefined,
) {
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

  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

async function readErrorDetail(res: Response): Promise<string> {
  return res.text().catch(() => "");
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
  const url = `${API_BASE}/inquiries/company/${encodeURIComponent(companyId)}${query}`;

  const res = await fetch(url, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const detail = await readErrorDetail(res);
    throw new Error(
      `問い合わせ一覧の取得に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as InquiryPageResult<InquiryManagementItem>;
}

// -----------------------------------------------------------
// GET: Inquiry 詳細
//   backend: GET /inquiries/{id}
// -----------------------------------------------------------

export async function getInquiryHTTP(id: string): Promise<InquiryDetail> {
  const trimmedId = assertID(id, "id");
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/inquiries/${encodeURIComponent(trimmedId)}`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const detail = await readErrorDetail(res);
    throw new Error(
      `問い合わせ詳細の取得に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as InquiryDetail;
}

// -----------------------------------------------------------
// POST: Inquiry を resolved にする
//   backend: POST /inquiries/{id}/resolve
// -----------------------------------------------------------

export async function resolveInquiryHTTP(
  id: string,
  params: ResolveInquiryParams,
): Promise<Inquiry> {
  const trimmedId = assertID(id, "id");
  const memberId = assertID(params.memberId, "memberId");
  const headers = await getAuthJsonHeadersOrThrow();

  const res = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(trimmedId)}/resolve`,
    {
      method: "POST",
      headers,
      body: JSON.stringify({
        memberId,
      }),
    },
  );

  if (!res.ok) {
    const detail = await readErrorDetail(res);
    throw new Error(
      `問い合わせの対応済み更新に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as Inquiry;
}

// -----------------------------------------------------------
// POST: Inquiry を open に戻す
//   backend: POST /inquiries/{id}/reopen
// -----------------------------------------------------------

export async function reopenInquiryHTTP(
  id: string,
  params: ReopenInquiryParams,
): Promise<Inquiry> {
  const trimmedId = assertID(id, "id");
  const memberId = assertID(params.memberId, "memberId");
  const headers = await getAuthJsonHeadersOrThrow();

  const res = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(trimmedId)}/reopen`,
    {
      method: "POST",
      headers,
      body: JSON.stringify({
        memberId,
      }),
    },
  );

  if (!res.ok) {
    const detail = await readErrorDetail(res);
    throw new Error(
      `問い合わせの再オープンに失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as Inquiry;
}

// -----------------------------------------------------------
// POST: Inquiry 画像追加
//   backend: POST /inquiries/{id}/images
// -----------------------------------------------------------

export async function addInquiryImageHTTP(
  id: string,
  params: AddInquiryImageParams,
): Promise<InquiryImageFile> {
  const trimmedId = assertID(id, "id");
  const headers = await getAuthJsonHeadersOrThrow();

  const fileName = assertID(params.fileName, "fileName");
  const fileUrl = assertID(params.fileUrl, "fileUrl");
  const mimeType = assertID(params.mimeType, "mimeType");

  const payload = {
    fileName,
    fileUrl,
    objectPath: params.objectPath ?? null,
    fileSize: Number(params.fileSize ?? 0),
    mimeType,
    createdAt: params.createdAt ?? null,
    createdBy: params.createdBy ?? null,
  };

  const res = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(trimmedId)}/images`,
    {
      method: "POST",
      headers,
      body: JSON.stringify(payload),
    },
  );

  if (!res.ok) {
    const detail = await readErrorDetail(res);
    throw new Error(
      `問い合わせ画像の追加に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as InquiryImageFile;
}

// -----------------------------------------------------------
// DELETE: Inquiry 画像削除
//   backend: DELETE /inquiries/{id}/images?fileName=...
// -----------------------------------------------------------

export async function deleteInquiryImageHTTP(
  id: string,
  fileName: string,
): Promise<InquiryImageFile[]> {
  const trimmedId = assertID(id, "id");
  const trimmedFileName = assertID(fileName, "fileName");
  const headers = await getAuthHeadersOrThrow();

  const query = new URLSearchParams({
    fileName: trimmedFileName,
  });

  const res = await fetch(
    `${API_BASE}/inquiries/${encodeURIComponent(trimmedId)}/images?${query.toString()}`,
    {
      method: "DELETE",
      headers,
    },
  );

  if (!res.ok) {
    const detail = await readErrorDetail(res);
    throw new Error(
      `問い合わせ画像の削除に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as InquiryImageFile[];
}