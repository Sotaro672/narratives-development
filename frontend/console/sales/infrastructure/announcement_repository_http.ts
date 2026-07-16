// frontend/console/sales/infrastructure/announcement_repository_http.ts
import { API_BASE } from "../../shell/src/shared/http/apiBase";
import { getAuthJsonHeaders } from "../../shell/src/shared/http/authHeaders";

// ============================================================
// Domain types
// ============================================================

export type AnnouncementAttachmentInput = {
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  objectPath: string;
};

export type AnnouncementAttachmentFile = {
  announcementId: string;
  id: string;
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  objectPath: string;
};

export type Announcement = {
  id: string;
  title: string;
  content: string;
  targetToken: string;
  targetAvatars: string[];
  published: boolean;
  publishedAt: string | null;
  attachments: string[];
  attachmentFiles: AnnouncementAttachmentFile[];
  createdAt: string;
  createdBy: string;
  createdByName: string;
  updatedAt: string | null;
  updatedBy: string | null;
  updatedByName: string | null;
};

export type AnnouncementListResult = {
  items: Announcement[];
  totalCount: number;
  page: number;
  perPage: number;
};

export type AnnouncementManagementTokenBlueprint = {
  tokenBlueprintId: string;
  tokenName: string;
  brandId: string;
};

export type AnnouncementManagementApiRow = {
  tokenBlueprint: AnnouncementManagementTokenBlueprint;
  announcements: Announcement[];
};

export type AnnouncementManagementApiResult = {
  companyId: string;
  rows: AnnouncementManagementApiRow[];
};

export type ListAnnouncementsParams = {
  targetToken: string;
  page?: number;
  perPage?: number;
};

export type ListAnnouncementManagementByCompanyIdParams = {
  companyId: string;
  page?: number;
  perPage?: number;
};

export type CreateAnnouncementInput = {
  id?: string;
  title: string;
  content: string;
  targetToken?: string | null;
  targetAvatars?: string[];
  attachments?: AnnouncementAttachmentInput[];
  published?: boolean;
  publishedAt?: string | null;
  createdBy: string;
};

export type UpdateAnnouncementInput = {
  title?: string;
  content?: string;
  targetToken?: string | null;
  targetAvatars?: string[];
  published?: boolean;
  publishedAt?: string | null;
  attachments?: AnnouncementAttachmentInput[];
  updatedBy?: string | null;
};

export type MarkPublishedInput = {
  updatedBy?: string | null;
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

  if (!text) {
    throw new Error(`${label} returned an empty response`);
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(`${label} returned invalid JSON`);
  }
}

// ============================================================
// Request helpers
// ============================================================

function uniqueStrings(values: string[] | undefined): string[] {
  if (!values) {
    return [];
  }

  return [...new Set(values.filter((value) => value !== ""))];
}

function normalizeAttachmentInputs(
  values: AnnouncementAttachmentInput[] | undefined,
): AnnouncementAttachmentInput[] {
  if (!values) {
    return [];
  }

  const seen = new Set<string>();
  const result: AnnouncementAttachmentInput[] = [];

  for (const item of values) {
    if (!item.fileName || !item.fileUrl || !item.objectPath) {
      continue;
    }

    const dedupeKey = item.objectPath;

    if (seen.has(dedupeKey)) {
      continue;
    }

    seen.add(dedupeKey);
    result.push(item);
  }

  return result;
}

// ============================================================
// Path builders
// ============================================================

function buildAnnouncementListPath(
  params: ListAnnouncementsParams,
): string {
  const targetToken = params.targetToken;

  if (!targetToken) {
    throw new Error("targetToken is required");
  }

  const searchParams = new URLSearchParams({
    targetToken,
  });

  if (params.page !== undefined) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage !== undefined) {
    searchParams.set("perPage", String(params.perPage));
  }

  return `${ANNOUNCEMENTS_ENDPOINT}?${searchParams.toString()}`;
}

function buildAnnouncementManagementByCompanyIdPath(
  params: ListAnnouncementManagementByCompanyIdParams,
): string {
  const companyId = params.companyId;

  if (!companyId) {
    throw new Error("companyId is required");
  }

  const searchParams = new URLSearchParams({
    companyId,
  });

  if (params.page !== undefined) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage !== undefined) {
    searchParams.set("perPage", String(params.perPage));
  }

  return `${ANNOUNCEMENTS_ENDPOINT}?${searchParams.toString()}`;
}

function buildAnnouncementDetailPath(id: string): string {
  if (!id) {
    throw new Error("announcement id is required");
  }

  return `${ANNOUNCEMENTS_ENDPOINT}/${encodeURIComponent(id)}`;
}

function buildAnnouncementPublishPath(id: string): string {
  return `${buildAnnouncementDetailPath(id)}/publish`;
}

// ============================================================
// Request body builders
// ============================================================

function buildCreateAnnouncementBody(
  input: CreateAnnouncementInput,
): Record<string, unknown> {
  if (!input.title) {
    throw new Error("title is required");
  }

  if (!input.content) {
    throw new Error("content is required");
  }

  if (!input.createdBy) {
    throw new Error("createdBy is required");
  }

  return {
    ...(input.id !== undefined && {
      id: input.id,
    }),
    title: input.title,
    content: input.content,
    targetToken: input.targetToken ?? null,
    targetAvatars: uniqueStrings(input.targetAvatars),
    attachments: normalizeAttachmentInputs(input.attachments),
    published: input.published ?? false,
    publishedAt: input.publishedAt ?? null,
    createdBy: input.createdBy,
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

  if (input.targetAvatars !== undefined) {
    body.targetAvatars = uniqueStrings(input.targetAvatars);
  }

  if (input.published !== undefined) {
    body.published = input.published;
  }

  if (input.publishedAt !== undefined) {
    body.publishedAt = input.publishedAt;
  }

  if (input.attachments !== undefined) {
    body.attachments = normalizeAttachmentInputs(input.attachments);
  }

  if (input.updatedBy !== undefined) {
    body.updatedBy = input.updatedBy;
  }

  return body;
}

function buildMarkPublishedBody(
  input: MarkPublishedInput = {},
): Record<string, unknown> {
  return {
    updatedBy: input.updatedBy ?? null,
  };
}

// ============================================================
// Repository
// ============================================================

/**
 * GET /announcements?targetToken={targetToken}&page={page}&perPage={perPage}
 */
export async function listAnnouncements(
  params: ListAnnouncementsParams,
): Promise<AnnouncementListResult> {
  return apiGetJson<AnnouncementListResult>(
    buildAnnouncementListPath(params),
  );
}

/**
 * GET /announcements?companyId={companyId}&page={page}&perPage={perPage}
 */
export async function listAnnouncementManagementByCompanyId(
  params: ListAnnouncementManagementByCompanyIdParams,
): Promise<AnnouncementManagementApiResult> {
  return apiGetJson<AnnouncementManagementApiResult>(
    buildAnnouncementManagementByCompanyIdPath(params),
  );
}

/**
 * GET /announcements/{id}
 */
export async function getAnnouncement(
  id: string,
): Promise<Announcement> {
  return apiGetJson<Announcement>(
    buildAnnouncementDetailPath(id),
  );
}

/**
 * POST /announcements
 *
 * attachmentsには、フロントエンドからFirebase Storageへ
 * アップロード済みのファイルメタデータを指定する。
 */
export async function createAnnouncement(
  input: CreateAnnouncementInput,
): Promise<Announcement> {
  return apiPostJson<Announcement>(
    ANNOUNCEMENTS_ENDPOINT,
    buildCreateAnnouncementBody(input),
  );
}

/**
 * PUT /announcements/{id}
 *
 * attachmentsには、フロントエンドからFirebase Storageへ
 * アップロード済みのファイルメタデータを指定する。
 */
export async function updateAnnouncement(
  id: string,
  input: UpdateAnnouncementInput,
): Promise<Announcement> {
  return apiPutJson<Announcement>(
    buildAnnouncementDetailPath(id),
    buildUpdateAnnouncementBody(input),
  );
}

/**
 * POST /announcements/{id}/publish
 */
export async function markAnnouncementPublished(
  id: string,
  input: MarkPublishedInput = {},
): Promise<Announcement> {
  return apiPostJson<Announcement>(
    buildAnnouncementPublishPath(id),
    buildMarkPublishedBody(input),
  );
}

/**
 * DELETE /announcements/{id}
 */
export async function deleteAnnouncement(id: string): Promise<void> {
  await apiDelete(buildAnnouncementDetailPath(id));
}