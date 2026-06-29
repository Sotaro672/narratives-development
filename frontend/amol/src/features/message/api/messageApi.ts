//frontend\amol\src\features\message\api\messageApi.ts
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { storage } from "../../../lib/firebase";

export type MessageImageAttachment = {
  storagePath: string;
  downloadUrl?: string;
  contentType: string;
  sizeBytes: number;
  width?: number;
  height?: number;
  uploadedAt: string;
};

export type Message = {
  id: string;
  senderAvatarId: string;
  senderAvatarName?: string | null;
  senderAvatarIcon?: string | null;
  receiverAvatarId: string;
  receiverAvatarName?: string | null;
  receiverAvatarIcon?: string | null;
  peerAvatarId?: string | null;
  peerAvatarName?: string | null;
  peerAvatarIcon?: string | null;
  body?: string;
  images?: MessageImageAttachment[];
  isRead: boolean;
  readAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type MessageListResponse = {
  messages: Message[];
  count: number;
};

export type MessageListFilter = {
  limit?: number;
  beforeCreatedAt?: string;
  before?: string;
  signal?: AbortSignal;
};

export type SendMessageInput = {
  id?: string;
  receiverAvatarId: string;
  body?: string;
  images?: MessageImageAttachment[];
};

export type SendMessageMultipartInput = {
  id?: string;
  receiverAvatarId: string;
  body?: string;
  files: File[];
};

export type UpdateMessageInput = {
  body?: string;
  images?: MessageImageAttachment[];
};

type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};

type ApiMessageListResponse = {
  data?: MessageListResponse | Message[];
  messages?: Message[];
  items?: Message[];
  count?: number;
  total?: number;
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

function appendOptionalQuery(
  query: URLSearchParams,
  key: string,
  value: string | number | null | undefined,
) {
  if (value === null || value === undefined) {
    return;
  }

  const normalized = String(value).trim();

  if (!normalized) {
    return;
  }

  query.set(key, normalized);
}

function appendListFilter(query: URLSearchParams, filter: MessageListFilter = {}) {
  appendOptionalQuery(query, "limit", filter.limit);
  appendOptionalQuery(query, "beforeCreatedAt", filter.beforeCreatedAt);

  if (!filter.beforeCreatedAt) {
    appendOptionalQuery(query, "before", filter.before);
  }
}

async function readApiJson<T>(res: Response): Promise<T> {
  return (await res.json().catch(() => ({}))) as T;
}

async function fetchWithAuth<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await getFirebaseIdToken();

  const headers = new Headers(init?.headers);
  headers.set("Authorization", `Bearer ${token}`);

  const isFormData =
    typeof FormData !== "undefined" && init?.body instanceof FormData;

  if (init?.body && !isFormData && !headers.has("Content-Type")) {
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

function unwrapData<T>(json: ApiDataResponse<T> | T): T | null {
  if (json && typeof json === "object" && "data" in json) {
    return (json as ApiDataResponse<T>).data ?? null;
  }

  return json as T;
}

function normalizeMessageList(json: ApiMessageListResponse): MessageListResponse {
  if (Array.isArray(json.data)) {
    return {
      messages: json.data,
      count: json.data.length,
    };
  }

  if (json.data && !Array.isArray(json.data)) {
    const messages = Array.isArray(json.data.messages)
      ? json.data.messages
      : [];

    return {
      messages,
      count: Number(json.data.count ?? messages.length),
    };
  }

  const messages = Array.isArray(json.messages)
    ? json.messages
    : Array.isArray(json.items)
      ? json.items
      : [];

  return {
    messages,
    count: Number(json.count ?? json.total ?? messages.length),
  };
}

function isAllowedMessageImageContentType(contentType: string): boolean {
  switch (contentType.toLowerCase()) {
    case "image/jpeg":
    case "image/png":
    case "image/webp":
    case "image/gif":
      return true;
    default:
      return false;
  }
}

export async function uploadMessageImage(params: {
  receiverAvatarId: string;
  file: File;
  messageId?: string;
  width?: number;
  height?: number;
}): Promise<MessageImageAttachment> {
  const contentType = params.file.type;

  if (!isAllowedMessageImageContentType(contentType)) {
    throw new Error("message: invalid image contentType");
  }

  const imageID = createUploadImageID(params.file);
  const safeFileName = sanitizeStorageFileName(params.file.name);
  const ownerPath = params.messageId || `to-${params.receiverAvatarId}`;
  const storagePath = `message-images/${ownerPath}/${imageID}/${safeFileName}`;
  const storageRef = ref(storage, storagePath);

  await uploadBytes(storageRef, params.file, {
    contentType,
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    storagePath,
    downloadUrl,
    contentType,
    sizeBytes: params.file.size,
    width: params.width,
    height: params.height,
    uploadedAt: new Date().toISOString(),
  };
}

// POST /mall/me/messages
export async function sendMessage(
  payload: SendMessageInput,
): Promise<Message | null> {
  const json = await fetchWithAuth<ApiDataResponse<Message> | Message>(
    "/mall/me/messages",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );

  return unwrapData<Message>(json);
}

// POST /mall/me/messages multipart/form-data
export async function sendMessageWithImages(
  input: SendMessageMultipartInput,
): Promise<Message | null> {
  const form = new FormData();

  if (input.id) {
    form.set("id", input.id);
  }

  form.set("receiverAvatarId", input.receiverAvatarId);

  if (input.body) {
    form.set("body", input.body);
  }

  for (const file of input.files) {
    form.append("images", file, file.name);
  }

  const json = await fetchWithAuth<ApiDataResponse<Message> | Message>(
    "/mall/me/messages",
    {
      method: "POST",
      body: form,
    },
  );

  return unwrapData<Message>(json);
}

// GET /mall/me/messages
export async function listMessages(
  filter: MessageListFilter = {},
): Promise<MessageListResponse> {
  const query = new URLSearchParams();
  appendListFilter(query, filter);

  const queryString = query.toString();
  const path = queryString
    ? `/mall/me/messages?${queryString}`
    : "/mall/me/messages";

  const json = await fetchWithAuth<ApiMessageListResponse>(path, {
    method: "GET",
    signal: filter.signal,
  });

  return normalizeMessageList(json);
}

// GET /mall/me/messages?peerAvatarId=...
export async function listMessageThread(
  peerAvatarId: string,
  filter: MessageListFilter = {},
): Promise<MessageListResponse> {
  const query = new URLSearchParams();
  appendOptionalQuery(query, "peerAvatarId", peerAvatarId);
  appendListFilter(query, filter);

  const json = await fetchWithAuth<ApiMessageListResponse>(
    `/mall/me/messages?${query.toString()}`,
    {
      method: "GET",
      signal: filter.signal,
    },
  );

  return normalizeMessageList(json);
}

// GET /mall/me/messages/received
export async function listReceivedMessages(
  filter: MessageListFilter = {},
): Promise<MessageListResponse> {
  const query = new URLSearchParams();
  appendListFilter(query, filter);

  const queryString = query.toString();
  const path = queryString
    ? `/mall/me/messages/received?${queryString}`
    : "/mall/me/messages/received";

  const json = await fetchWithAuth<ApiMessageListResponse>(path, {
    method: "GET",
    signal: filter.signal,
  });

  return normalizeMessageList(json);
}

export async function countUnreadReceivedMessages(
  filter: MessageListFilter = {},
): Promise<number> {
  const response = await listReceivedMessages(filter);

  return response.messages.reduce((total: number, message: Message) => {
    return message.isRead === false ? total + 1 : total;
  }, 0);
}

// GET /mall/me/messages/sent
export async function listSentMessages(
  filter: MessageListFilter = {},
): Promise<MessageListResponse> {
  const query = new URLSearchParams();
  appendListFilter(query, filter);

  const queryString = query.toString();
  const path = queryString
    ? `/mall/me/messages/sent?${queryString}`
    : "/mall/me/messages/sent";

  const json = await fetchWithAuth<ApiMessageListResponse>(path, {
    method: "GET",
    signal: filter.signal,
  });

  return normalizeMessageList(json);
}

// GET /mall/me/messages/{messageId}
export async function getMessage(messageId: string): Promise<Message | null> {
  const json = await fetchWithAuth<ApiDataResponse<Message> | Message>(
    `/mall/me/messages/${encodeURIComponent(messageId)}`,
    {
      method: "GET",
    },
  );

  return unwrapData<Message>(json);
}

// PATCH /mall/me/messages/{messageId}
export async function updateMessage(
  messageId: string,
  payload: UpdateMessageInput,
): Promise<Message | null> {
  const json = await fetchWithAuth<ApiDataResponse<Message> | Message>(
    `/mall/me/messages/${encodeURIComponent(messageId)}`,
    {
      method: "PATCH",
      body: JSON.stringify(payload),
    },
  );

  return unwrapData<Message>(json);
}

// DELETE /mall/me/messages/{messageId}
export async function deleteMessage(messageId: string): Promise<void> {
  await fetchWithAuth<Record<string, never>>(
    `/mall/me/messages/${encodeURIComponent(messageId)}`,
    {
      method: "DELETE",
    },
  );
}

// POST /mall/me/messages/{messageId}/read
export async function markMessageAsRead(
  messageId: string,
): Promise<Message | null> {
  const json = await fetchWithAuth<ApiDataResponse<Message> | Message>(
    `/mall/me/messages/${encodeURIComponent(messageId)}/read`,
    {
      method: "POST",
    },
  );

  return unwrapData<Message>(json);
}

// PATCH /mall/me/messages/{messageId}/read
export async function patchMessageAsRead(
  messageId: string,
): Promise<Message | null> {
  const json = await fetchWithAuth<ApiDataResponse<Message> | Message>(
    `/mall/me/messages/${encodeURIComponent(messageId)}/read`,
    {
      method: "PATCH",
    },
  );

  return unwrapData<Message>(json);
}