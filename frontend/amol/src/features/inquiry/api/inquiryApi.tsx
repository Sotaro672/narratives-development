// frontend/amol/src/features/inquiry/api/inquiryApi.tsx
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { storage } from "../../../lib/firebase";

export type InquiryImage = {
  fileName: string;
  fileUrl: string;
  objectPath: string;
  fileSize: number;
  mimeType: string;
  createdAt: string;
};

export type CreateInquiryRequest = {
  productId: string;
  subject: string;
  content: string;
  inquiryType: string;
  images: InquiryImage[];
};

export type ReplyInquiryRequest = {
  content: string;
  images: InquiryImage[];
};

export type Inquiry = {
  id?: string;
  productId?: string;
  avatarId?: string;
  subject?: string;
  content?: string;
  status?: string;
  inquiryType?: string;
  isRead?: boolean;
  images?: InquiryImage[];
  createdAt?: string;
  updatedAt?: string;
};

export type InquiryReply = {
  id?: string;
  inquiryId?: string;
  senderType?: string;
  senderId?: string;
  content?: string;
  isRead?: boolean;
  images?: InquiryImage[];
  createdAt?: string;
  updatedAt?: string | null;
};

type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};

type ApiItemsResponse<T> = {
  items?: T[];
  error?: string;
};

type ApiUnreadCountResponse = {
  count?: number;
  error?: string;
};

function buildApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return path;
  }

  return `${baseUrl}${path}`;
}

function createUploadImageID(file: File): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function sanitizeStorageFileName(fileName: string): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(/[^\w.\-()]/g, "_");
}

async function readApiJson<T>(res: Response): Promise<T> {
  return (await res.json().catch(() => ({}))) as T;
}

async function fetchWithAuth<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await getFirebaseIdToken();

  const headers = new Headers(init?.headers);
  headers.set("Authorization", `Bearer ${token}`);

  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(buildApiUrl(path), {
    ...init,
    headers,
  });

  const json = await readApiJson<T & { error?: string }>(res);

  if (!res.ok) {
    throw new Error(json.error || "APIリクエストに失敗しました。");
  }

  return json;
}

export async function uploadInquiryImage(params: {
  productId: string;
  file: File;
}): Promise<InquiryImage> {
  const imageID = createUploadImageID(params.file);
  const safeFileName = sanitizeStorageFileName(params.file.name);
  const objectPath = `inquiry-images/${params.productId}/${imageID}/${safeFileName}`;
  const storageRef = ref(storage, objectPath);
  const mimeType = params.file.type || "application/octet-stream";

  await uploadBytes(storageRef, params.file, {
    contentType: mimeType,
  });

  const fileUrl = await getDownloadURL(storageRef);

  return {
    fileName: params.file.name,
    fileUrl,
    objectPath,
    fileSize: params.file.size,
    mimeType,
    createdAt: new Date().toISOString(),
  };
}

export async function uploadReplyImage(params: {
  inquiryId: string;
  file: File;
}): Promise<InquiryImage> {
  const imageID = createUploadImageID(params.file);
  const safeFileName = sanitizeStorageFileName(params.file.name);
  const objectPath = `inquiry-replies/${params.inquiryId}/${imageID}/${safeFileName}`;
  const storageRef = ref(storage, objectPath);
  const mimeType = params.file.type || "application/octet-stream";

  await uploadBytes(storageRef, params.file, {
    contentType: mimeType,
  });

  const fileUrl = await getDownloadURL(storageRef);

  return {
    fileName: params.file.name,
    fileUrl,
    objectPath,
    fileSize: params.file.size,
    mimeType,
    createdAt: new Date().toISOString(),
  };
}

export async function createInquiry(
  payload: CreateInquiryRequest,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<ApiDataResponse<Inquiry>>(
    "/mall/me/inquiries",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );

  return json.data ?? null;
}

export async function getInquiry(inquiryId: string): Promise<Inquiry | null> {
  const json = await fetchWithAuth<ApiDataResponse<Inquiry>>(
    `/mall/me/inquiries/${encodeURIComponent(inquiryId)}`,
    {
      method: "GET",
    },
  );

  return json.data ?? null;
}

export async function listInquiryReplies(
  inquiryId: string,
): Promise<InquiryReply[]> {
  const json = await fetchWithAuth<ApiItemsResponse<InquiryReply>>(
    `/mall/me/inquiries/${encodeURIComponent(inquiryId)}/replies`,
    {
      method: "GET",
    },
  );

  return Array.isArray(json.items) ? json.items : [];
}

export async function getUnreadInquiryCount(params: {
  companyId: string;
  productId?: string;
  status?: string;
  inquiryType?: string;
  searchQuery?: string;
}): Promise<number> {
  const query = new URLSearchParams();

  query.set("companyId", params.companyId);

  if (params.productId) {
    query.set("productId", params.productId);
  }

  if (params.status) {
    query.set("status", params.status);
  }

  if (params.inquiryType) {
    query.set("inquiryType", params.inquiryType);
  }

  if (params.searchQuery) {
    query.set("searchQuery", params.searchQuery);
  }

  const json = await fetchWithAuth<ApiUnreadCountResponse>(
    `/mall/me/inquiries/unread-count?${query.toString()}`,
    {
      method: "GET",
    },
  );

  return Number(json.count ?? 0);
}

export async function markInquiryAsRead(
  inquiryId: string,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<ApiDataResponse<Inquiry>>(
    `/mall/me/inquiries/${encodeURIComponent(inquiryId)}/mark-as-read`,
    {
      method: "POST",
    },
  );

  return json.data ?? null;
}

export async function replyInquiry(
  inquiryId: string,
  payload: ReplyInquiryRequest,
): Promise<InquiryReply | null> {
  const json = await fetchWithAuth<ApiDataResponse<InquiryReply>>(
    `/mall/me/inquiries/${encodeURIComponent(inquiryId)}/reply`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );

  return json.data ?? null;
}

export async function closeInquiry(
  inquiryId: string,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<ApiDataResponse<Inquiry>>(
    `/mall/me/inquiries/${encodeURIComponent(inquiryId)}/close`,
    {
      method: "POST",
    },
  );

  return json.data ?? null;
}