// frontend/amol/src/features/inquiry/api/inquiryApi.tsx

import {
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

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

export type ListMeInquiriesParams = {
  page?: number;
  perPage?: number;
  productId?: string;
  status?: string;
  inquiryType?: string;
  searchQuery?: string;
  signal?: AbortSignal;
};

export type ListMeInquiriesResult = {
  items: Inquiry[];
  page?: number;
  perPage?: number;
  total?: number;
  totalCount?: number;
};

export type InquiryThread = {
  inquiry: Inquiry | null;
  replies: InquiryReply[];
};

export type GetUnreadInquiryCountParams = {
  productId?: string;
  status?: string;
  inquiryType?: string;
  searchQuery?: string;
};

type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};

type ApiItemsResponse<T> = {
  items?: T[];
  page?: number;
  perPage?: number;
  total?: number;
  totalCount?: number;
  error?: string;
};

type ApiUnreadCountResponse = {
  count?: number;
  unreadCount?: number;
  error?: string;
};

type UploadInquiryImageFileParams = {
  directoryPath: string;
  file: File;
};

function buildApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return path;
  }

  return `${baseUrl}${path}`;
}

function createUploadImageId(file: File): string {
  if (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto
  ) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function sanitizeStorageFileName(
  fileName: string,
): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(/[^\w.\-()]/g, "_");
}

function appendOptionalQuery(
  query: URLSearchParams,
  key: string,
  value: string | number | null | undefined,
): void {
  if (
    value === null ||
    value === undefined
  ) {
    return;
  }

  const normalized = String(value).trim();

  if (!normalized) {
    return;
  }

  query.set(key, normalized);
}

async function readApiJson<T>(
  response: Response,
): Promise<T> {
  return (
    await response
      .json()
      .catch(() => ({}))
  ) as T;
}

async function fetchWithAuth<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const token = await getFirebaseIdToken();
  const headers = new Headers(init?.headers);

  headers.set(
    "Authorization",
    `Bearer ${token}`,
  );

  if (
    init?.body &&
    !headers.has("Content-Type")
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  const response = await fetch(
    buildApiUrl(path),
    {
      ...init,
      headers,
    },
  );

  const json = await readApiJson<
    T & {
      error?: string;
    }
  >(response);

  if (!response.ok) {
    throw new Error(
      json.error ||
        "APIリクエストに失敗しました。",
    );
  }

  return json;
}

async function uploadInquiryImageFile({
  directoryPath,
  file,
}: UploadInquiryImageFileParams): Promise<InquiryImage> {
  const imageId =
    createUploadImageId(file);

  const safeFileName =
    sanitizeStorageFileName(
      file.name,
    );

  const objectPath =
    `${directoryPath}/${imageId}/${safeFileName}`;

  const storageRef = ref(
    storage,
    objectPath,
  );

  const mimeType =
    file.type ||
    "application/octet-stream";

  await uploadBytes(
    storageRef,
    file,
    {
      contentType: mimeType,
    },
  );

  const fileUrl =
    await getDownloadURL(
      storageRef,
    );

  return {
    fileName: file.name,
    fileUrl,
    objectPath,
    fileSize: file.size,
    mimeType,
    createdAt:
      new Date().toISOString(),
  };
}

export async function uploadInquiryImage(
  params: {
    productId: string;
    file: File;
  },
): Promise<InquiryImage> {
  return uploadInquiryImageFile({
    directoryPath:
      `inquiry-images/${params.productId}`,
    file: params.file,
  });
}

export async function uploadReplyImage(
  params: {
    inquiryId: string;
    file: File;
  },
): Promise<InquiryImage> {
  return uploadInquiryImageFile({
    directoryPath:
      `inquiry-replies/${params.inquiryId}`,
    file: params.file,
  });
}

export async function createInquiry(
  payload: CreateInquiryRequest,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<
    ApiDataResponse<Inquiry>
  >(
    "/mall/me/inquiries",
    {
      method: "POST",
      body: JSON.stringify(
        payload,
      ),
    },
  );

  return json.data ?? null;
}

export async function listMeInquiries(
  params: ListMeInquiriesParams = {},
): Promise<ListMeInquiriesResult> {
  const query =
    new URLSearchParams();

  appendOptionalQuery(
    query,
    "page",
    params.page,
  );

  appendOptionalQuery(
    query,
    "perPage",
    params.perPage,
  );

  appendOptionalQuery(
    query,
    "productId",
    params.productId,
  );

  appendOptionalQuery(
    query,
    "status",
    params.status,
  );

  appendOptionalQuery(
    query,
    "inquiryType",
    params.inquiryType,
  );

  appendOptionalQuery(
    query,
    "searchQuery",
    params.searchQuery,
  );

  const queryString =
    query.toString();

  const path = queryString
    ? `/mall/me/inquiries?${queryString}`
    : "/mall/me/inquiries";

  const json = await fetchWithAuth<
    ApiItemsResponse<Inquiry>
  >(path, {
    method: "GET",
    signal: params.signal,
  });

  return {
    items: Array.isArray(
      json.items,
    )
      ? json.items
      : [],
    page: json.page,
    perPage: json.perPage,
    total: json.total,
    totalCount:
      json.totalCount,
  };
}

// ChatListPageなどから利用するための互換alias。
export async function fetchMeInquiries(
  params: ListMeInquiriesParams = {},
): Promise<ListMeInquiriesResult> {
  return listMeInquiries(params);
}

export async function getInquiry(
  inquiryId: string,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<
    ApiDataResponse<Inquiry>
  >(
    `/mall/me/inquiries/${encodeURIComponent(
      inquiryId,
    )}`,
    {
      method: "GET",
    },
  );

  return json.data ?? null;
}

export async function listInquiryReplies(
  inquiryId: string,
): Promise<InquiryReply[]> {
  const json = await fetchWithAuth<
    ApiItemsResponse<InquiryReply>
  >(
    `/mall/me/inquiries/${encodeURIComponent(
      inquiryId,
    )}/replies`,
    {
      method: "GET",
    },
  );

  return Array.isArray(
    json.items,
  )
    ? json.items
    : [];
}

export async function getInquiryThread(
  inquiryId: string,
): Promise<InquiryThread> {
  const [
    inquiry,
    replies,
  ] = await Promise.all([
    getInquiry(inquiryId),
    listInquiryReplies(
      inquiryId,
    ),
  ]);

  return {
    inquiry,
    replies,
  };
}

export async function getUnreadInquiryCount(
  params: GetUnreadInquiryCountParams = {},
): Promise<number> {
  const query =
    new URLSearchParams();

  appendOptionalQuery(
    query,
    "productId",
    params.productId,
  );

  appendOptionalQuery(
    query,
    "status",
    params.status,
  );

  appendOptionalQuery(
    query,
    "inquiryType",
    params.inquiryType,
  );

  appendOptionalQuery(
    query,
    "searchQuery",
    params.searchQuery,
  );

  const queryString =
    query.toString();

  const path = queryString
    ? `/mall/me/inquiries/unread-count?${queryString}`
    : "/mall/me/inquiries/unread-count";

  const json = await fetchWithAuth<
    ApiUnreadCountResponse
  >(path, {
    method: "GET",
  });

  const unreadCount = Number(
    json.count ??
      json.unreadCount ??
      0,
  );

  if (
    !Number.isFinite(
      unreadCount,
    )
  ) {
    return 0;
  }

  return Math.max(
    0,
    Math.floor(
      unreadCount,
    ),
  );
}

export async function markInquiryAsRead(
  inquiryId: string,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<
    ApiDataResponse<Inquiry>
  >(
    `/mall/me/inquiries/${encodeURIComponent(
      inquiryId,
    )}/mark-as-read`,
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
  const json = await fetchWithAuth<
    ApiDataResponse<InquiryReply>
  >(
    `/mall/me/inquiries/${encodeURIComponent(
      inquiryId,
    )}/reply`,
    {
      method: "POST",
      body: JSON.stringify(
        payload,
      ),
    },
  );

  return json.data ?? null;
}

export async function closeInquiry(
  inquiryId: string,
): Promise<Inquiry | null> {
  const json = await fetchWithAuth<
    ApiDataResponse<Inquiry>
  >(
    `/mall/me/inquiries/${encodeURIComponent(
      inquiryId,
    )}/close`,
    {
      method: "POST",
    },
  );

  return json.data ?? null;
}