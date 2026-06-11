// frontend/console/sales/infrastructure/announcement_repository_http.ts
import { API_BASE } from "../../shell/src/shared/http/apiBase";
import { getAuthJsonHeaders } from "../../shell/src/shared/http/authHeaders";

// ============================================================
// Domain types
// ============================================================

export type Announcement = {
  id: string;
  title: string;
  content: string;
  targetToken: string | null;
  published: boolean;
  publishedAt: string | null;
  attachments: string[];
  createdAt: string;
  createdBy: string;
  updatedAt: string | null;
  updatedBy: string | null;
};

export type AnnouncementListResult = {
  items: Announcement[];
  totalCount: number;
  page: number;
  perPage: number;
};

export type ListAnnouncementsParams = {
  page?: number;
  perPage?: number;
  targetToken?: string | null;
  published?: boolean | null;
};

export type CreateAnnouncementInput = {
  id?: string;
  title: string;
  content: string;
  targetToken?: string | null;
  attachments?: string[];
  published?: boolean;
  publishedAt?: string | null;
  createdBy: string;
};

export type UpdateAnnouncementInput = {
  title?: string;
  content?: string;
  targetToken?: string | null;
  published?: boolean;
  publishedAt?: string | null;
  attachments?: string[];
  updatedBy?: string | null;
};

// ============================================================
// API DTOs
// ============================================================

type ApiAnnouncement = {
  id?: string | null;
  title?: string | null;
  content?: string | null;
  targetToken?: string | null;
  published?: boolean | null;
  publishedAt?: string | null;
  attachments?: string[] | null;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

type ApiAnnouncementListResult = {
  items?: ApiAnnouncement[] | null;
  totalCount?: number | null;
  page?: number | null;
  perPage?: number | null;
};

// ============================================================
// Endpoint
// ============================================================

const ANNOUNCEMENTS_ENDPOINT = "/announcements";

// ============================================================
// HTTP helpers
// ============================================================

async function apiGetJson<T>(path: string): Promise<T> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: {
      ...headers,
      Accept: "application/json",
    },
    credentials: "include",
  });

  return parseJsonResponse<T>(res, `GET ${path}`);
}

async function apiPostJson<T>(path: string, body: unknown): Promise<T> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: {
      ...headers,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(body),
  });

  return parseJsonResponse<T>(res, `POST ${path}`);
}

async function apiPutJson<T>(path: string, body: unknown): Promise<T> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "PUT",
    headers: {
      ...headers,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(body),
  });

  return parseJsonResponse<T>(res, `PUT ${path}`);
}

async function apiDelete(path: string): Promise<void> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "DELETE",
    headers: {
      ...headers,
      Accept: "application/json",
    },
    credentials: "include",
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(text || `DELETE ${path} failed: ${res.status}`);
  }
}

async function parseJsonResponse<T>(
  res: Response,
  label: string,
): Promise<T> {
  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(text || `${label} failed: ${res.status}`);
  }

  if (!text) return {} as T;

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(text);
  }
}

// ============================================================
// Mapper helpers
// ============================================================

function toSafeNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  const n = Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }

  return n;
}

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const s = String(value ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;

    seen.add(s);
    result.push(s);
  }

  return result;
}

function nullableString(value: unknown): string | null {
  const s = String(value ?? "").trim();
  return s === "" ? null : s;
}

function fromApiAnnouncement(data: ApiAnnouncement): Announcement {
  return {
    id: String(data?.id ?? "").trim(),
    title: String(data?.title ?? "").trim(),
    content: String(data?.content ?? "").trim(),
    targetToken: nullableString(data?.targetToken),
    published: Boolean(data?.published),
    publishedAt: nullableString(data?.publishedAt),
    attachments: uniqueStrings(data?.attachments),
    createdAt: String(data?.createdAt ?? "").trim(),
    createdBy: String(data?.createdBy ?? "").trim(),
    updatedAt: nullableString(data?.updatedAt),
    updatedBy: nullableString(data?.updatedBy),
  };
}

function fromApiAnnouncementListResult(
  data: ApiAnnouncementListResult,
): AnnouncementListResult {
  const rawItems = Array.isArray(data?.items) ? data.items : [];

  return {
    items: rawItems
      .map(fromApiAnnouncement)
      .filter((announcement) => announcement.id !== ""),
    totalCount: toSafeNumber(data?.totalCount),
    page: toSafeNumber(data?.page),
    perPage: toSafeNumber(data?.perPage),
  };
}

// ============================================================
// Path builders
// ============================================================

function buildAnnouncementListPath(
  params: ListAnnouncementsParams = {},
): string {
  const searchParams = new URLSearchParams();

  if (params.page != null) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage != null) {
    searchParams.set("perPage", String(params.perPage));
  }

  const targetToken = String(params.targetToken ?? "").trim();
  if (targetToken) {
    searchParams.set("targetToken", targetToken);
  }

  if (params.published != null) {
    searchParams.set("published", String(params.published));
  }

  const query = searchParams.toString();
  return query ? `${ANNOUNCEMENTS_ENDPOINT}?${query}` : ANNOUNCEMENTS_ENDPOINT;
}

function buildAnnouncementDetailPath(id: string): string {
  const normalizedId = String(id || "").trim();

  if (!normalizedId) {
    throw new Error("announcement id is required");
  }

  return `${ANNOUNCEMENTS_ENDPOINT}/${encodeURIComponent(normalizedId)}`;
}

// ============================================================
// Request body builders
// ============================================================

function buildCreateAnnouncementBody(
  input: CreateAnnouncementInput,
): Record<string, unknown> {
  const title = String(input.title ?? "").trim();
  const content = String(input.content ?? "").trim();
  const createdBy = String(input.createdBy ?? "").trim();

  if (!title) {
    throw new Error("title is required");
  }

  if (!content) {
    throw new Error("content is required");
  }

  if (!createdBy) {
    throw new Error("createdBy is required");
  }

  return {
    id: String(input.id ?? "").trim(),
    title,
    content,
    targetToken: input.targetToken ?? null,
    attachments: uniqueStrings(input.attachments),
    published: Boolean(input.published),
    publishedAt: input.publishedAt ?? null,
    createdBy,
  };
}

function buildUpdateAnnouncementBody(
  input: UpdateAnnouncementInput,
): Record<string, unknown> {
  const body: Record<string, unknown> = {};

  if (input.title !== undefined) {
    body.title = input.title;
  }

  if (input.content !== undefined) {
    body.content = input.content;
  }

  if (input.targetToken !== undefined) {
    body.targetToken = input.targetToken;
  }

  if (input.published !== undefined) {
    body.published = input.published;
  }

  if (input.publishedAt !== undefined) {
    body.publishedAt = input.publishedAt;
  }

  if (input.attachments !== undefined) {
    body.attachments = uniqueStrings(input.attachments);
  }

  if (input.updatedBy !== undefined) {
    body.updatedBy = input.updatedBy;
  }

  return body;
}

// ============================================================
// Repository
// ============================================================

/**
 * backend: GET /announcements
 */
export async function listAnnouncements(
  params: ListAnnouncementsParams = {},
): Promise<AnnouncementListResult> {
  const data = await apiGetJson<ApiAnnouncementListResult>(
    buildAnnouncementListPath(params),
  );

  return fromApiAnnouncementListResult(data);
}

/**
 * backend: GET /announcements/{id}
 */
export async function getAnnouncement(id: string): Promise<Announcement> {
  const data = await apiGetJson<ApiAnnouncement>(
    buildAnnouncementDetailPath(id),
  );

  return fromApiAnnouncement(data);
}

/**
 * backend: POST /announcements
 */
export async function createAnnouncement(
  input: CreateAnnouncementInput,
): Promise<Announcement> {
  const data = await apiPostJson<ApiAnnouncement>(
    ANNOUNCEMENTS_ENDPOINT,
    buildCreateAnnouncementBody(input),
  );

  return fromApiAnnouncement(data);
}

/**
 * backend: PUT /announcements/{id}
 */
export async function updateAnnouncement(
  id: string,
  input: UpdateAnnouncementInput,
): Promise<Announcement> {
  const data = await apiPutJson<ApiAnnouncement>(
    buildAnnouncementDetailPath(id),
    buildUpdateAnnouncementBody(input),
  );

  return fromApiAnnouncement(data);
}

/**
 * backend: DELETE /announcements/{id}
 */
export async function deleteAnnouncement(id: string): Promise<void> {
  await apiDelete(buildAnnouncementDetailPath(id));
}