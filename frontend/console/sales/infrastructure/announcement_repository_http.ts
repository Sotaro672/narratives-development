//frontend\console\sales\infrastructure\announcement_repository_http.ts
import { API_BASE } from "../../shell/src/shared/http/apiBase";
import { getAuthJsonHeaders } from "../../shell/src/shared/http/authHeaders";

// ============================================================
// Domain types
// ============================================================

export type AnnouncementAvatarStateFollow = {
  avatarId: string;
  followedAt: string | null;
};

export type AnnouncementAvatarState = {
  id: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
  followers: AnnouncementAvatarStateFollow[];
  following: AnnouncementAvatarStateFollow[];
  lastActiveAt: string | null;
  updatedAt: string | null;
};

export type AnnouncementTargetAvatarDetail = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  avatarState: AnnouncementAvatarState | null;
  followerCount: number;
  followingCount: number;
  postCount: number;
};

export type AnnouncementProductBlueprint = {
  productBlueprintId: string;
  productName: string;
};

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
  targetToken: string | null;
  tokenName: string | null;
  targetAvatars: string[];
  targetAvatarDetails: AnnouncementTargetAvatarDetail[];
  mintAddresses: string[];
  modelIds: string[];
  productBlueprints: AnnouncementProductBlueprint[];
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
// API DTOs
// ============================================================

type ApiAnnouncementAvatarStateFollow = {
  avatarId?: string | null;
  AvatarID?: string | null;

  followedAt?: string | null;
  FollowedAt?: string | null;
};

type ApiAnnouncementAvatarState = {
  id?: string | null;
  ID?: string | null;

  followerCount?: number | null;
  FollowerCount?: number | null;

  followingCount?: number | null;
  FollowingCount?: number | null;

  postCount?: number | null;
  PostCount?: number | null;

  followers?: ApiAnnouncementAvatarStateFollow[] | null;
  Followers?: ApiAnnouncementAvatarStateFollow[] | null;

  following?: ApiAnnouncementAvatarStateFollow[] | null;
  Following?: ApiAnnouncementAvatarStateFollow[] | null;

  lastActiveAt?: string | null;
  LastActiveAt?: string | null;

  updatedAt?: string | null;
  UpdatedAt?: string | null;
};

type ApiAnnouncementTargetAvatarDetail = {
  avatarId?: string | null;
  AvatarID?: string | null;

  avatarName?: string | null;
  AvatarName?: string | null;

  avatarIcon?: string | null;
  AvatarIcon?: string | null;

  avatarState?: ApiAnnouncementAvatarState | null;
  AvatarState?: ApiAnnouncementAvatarState | null;

  followerCount?: number | null;
  FollowerCount?: number | null;

  followingCount?: number | null;
  FollowingCount?: number | null;

  postCount?: number | null;
  PostCount?: number | null;
};

type ApiAnnouncementProductBlueprint = {
  productBlueprintId?: string | null;
  ProductBlueprintID?: string | null;

  productName?: string | null;
  ProductName?: string | null;
};

type ApiAnnouncementAttachmentFile = {
  announcementId?: string | null;
  AnnouncementID?: string | null;

  id?: string | null;
  ID?: string | null;

  fileName?: string | null;
  FileName?: string | null;

  fileUrl?: string | null;
  fileURL?: string | null;
  FileURL?: string | null;

  fileSize?: number | null;
  FileSize?: number | null;

  mimeType?: string | null;
  MimeType?: string | null;

  objectPath?: string | null;
  ObjectPath?: string | null;
};

type ApiAnnouncement = {
  id?: string | null;
  ID?: string | null;

  title?: string | null;
  Title?: string | null;

  content?: string | null;
  Content?: string | null;

  targetToken?: string | null;
  TargetToken?: string | null;

  tokenName?: string | null;
  TokenName?: string | null;

  targetAvatars?: string[] | null;
  TargetAvatars?: string[] | null;

  targetAvatarDetails?: ApiAnnouncementTargetAvatarDetail[] | null;
  TargetAvatarDetails?: ApiAnnouncementTargetAvatarDetail[] | null;

  mintAddresses?: string[] | null;
  MintAddresses?: string[] | null;

  modelIds?: string[] | null;
  ModelIDs?: string[] | null;

  productBlueprints?: ApiAnnouncementProductBlueprint[] | null;
  ProductBlueprints?: ApiAnnouncementProductBlueprint[] | null;

  published?: boolean | null;
  Published?: boolean | null;

  publishedAt?: string | null;
  PublishedAt?: string | null;

  attachments?: string[] | null;
  Attachments?: string[] | null;

  attachmentFiles?: ApiAnnouncementAttachmentFile[] | null;
  AttachmentFiles?: ApiAnnouncementAttachmentFile[] | null;

  createdAt?: string | null;
  CreatedAt?: string | null;

  createdBy?: string | null;
  CreatedBy?: string | null;

  createdByName?: string | null;
  CreatedByName?: string | null;

  updatedAt?: string | null;
  UpdatedAt?: string | null;

  updatedBy?: string | null;
  UpdatedBy?: string | null;

  updatedByName?: string | null;
  UpdatedByName?: string | null;
};

type ApiAnnouncementListResult = {
  items?: ApiAnnouncement[] | null;
  Items?: ApiAnnouncement[] | null;

  totalCount?: number | null;
  TotalCount?: number | null;

  page?: number | null;
  Page?: number | null;

  perPage?: number | null;
  PerPage?: number | null;
};

type ApiAnnouncementManagementTokenBlueprint = {
  tokenBlueprintId?: string | null;
  TokenBlueprintID?: string | null;

  tokenName?: string | null;
  TokenName?: string | null;

  brandId?: string | null;
  BrandID?: string | null;
};

type ApiAnnouncementManagementRow = {
  tokenBlueprint?: ApiAnnouncementManagementTokenBlueprint | null;
  TokenBlueprint?: ApiAnnouncementManagementTokenBlueprint | null;

  announcements?: ApiAnnouncement[] | null;
  Announcements?: ApiAnnouncement[] | null;
};

type ApiAnnouncementManagementResult = {
  companyId?: string | null;
  CompanyID?: string | null;

  rows?: ApiAnnouncementManagementRow[] | null;
  Rows?: ApiAnnouncementManagementRow[] | null;
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

function firstValue<T>(...values: Array<T | null | undefined>): T | undefined {
  return values.find((value) => value !== undefined && value !== null);
}

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
    const s = String(value ?? "");
    if (!s) continue;
    if (seen.has(s)) continue;

    seen.add(s);
    result.push(s);
  }

  return result;
}

function nullableString(value: unknown): string | null {
  const s = String(value ?? "");
  return s === "" ? null : s;
}

function normalizeAttachmentInputs(
  values: unknown,
): AnnouncementAttachmentInput[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: AnnouncementAttachmentInput[] = [];

  for (const value of values) {
    if (!value || typeof value !== "object") {
      continue;
    }

    const item = value as Partial<AnnouncementAttachmentInput>;

    const fileName = String(item.fileName ?? "");
    const fileUrl = String(item.fileUrl ?? "");
    const objectPath = String(item.objectPath ?? "");
    const mimeType = String(item.mimeType ?? "");
    const fileSize = toSafeNumber(item.fileSize);

    if (!fileName || !fileUrl || !objectPath) {
      continue;
    }

    const dedupeKey = objectPath || fileUrl || fileName;
    if (seen.has(dedupeKey)) {
      continue;
    }

    seen.add(dedupeKey);

    result.push({
      fileName,
      fileUrl,
      fileSize,
      mimeType,
      objectPath,
    });
  }

  return result;
}

function fromApiAnnouncementAvatarStateFollow(
  data: ApiAnnouncementAvatarStateFollow,
): AnnouncementAvatarStateFollow {
  return {
    avatarId: String(firstValue(data?.avatarId, data?.AvatarID) ?? ""),
    followedAt: nullableString(firstValue(data?.followedAt, data?.FollowedAt)),
  };
}

function fromApiAnnouncementAvatarStateFollows(
  values: unknown,
): AnnouncementAvatarStateFollow[] {
  if (!Array.isArray(values)) {
    return [];
  }

  return values
    .map((value) =>
      fromApiAnnouncementAvatarStateFollow(
        value as ApiAnnouncementAvatarStateFollow,
      ),
    )
    .filter((item) => item.avatarId !== "");
}

function fromApiAnnouncementAvatarState(
  data: ApiAnnouncementAvatarState | null | undefined,
): AnnouncementAvatarState | null {
  if (!data) {
    return null;
  }

  return {
    id: String(firstValue(data?.id, data?.ID) ?? ""),
    followerCount: toSafeNumber(
      firstValue(data?.followerCount, data?.FollowerCount),
    ),
    followingCount: toSafeNumber(
      firstValue(data?.followingCount, data?.FollowingCount),
    ),
    postCount: toSafeNumber(firstValue(data?.postCount, data?.PostCount)),
    followers: fromApiAnnouncementAvatarStateFollows(
      firstValue(data?.followers, data?.Followers),
    ),
    following: fromApiAnnouncementAvatarStateFollows(
      firstValue(data?.following, data?.Following),
    ),
    lastActiveAt: nullableString(
      firstValue(data?.lastActiveAt, data?.LastActiveAt),
    ),
    updatedAt: nullableString(firstValue(data?.updatedAt, data?.UpdatedAt)),
  };
}

function fromApiAnnouncementTargetAvatarDetail(
  data: ApiAnnouncementTargetAvatarDetail,
): AnnouncementTargetAvatarDetail {
  const avatarState = fromApiAnnouncementAvatarState(
    firstValue(data?.avatarState, data?.AvatarState),
  );

  return {
    avatarId: String(firstValue(data?.avatarId, data?.AvatarID) ?? ""),
    avatarName: String(
      firstValue(data?.avatarName, data?.AvatarName) ?? "",
    ),
    avatarIcon: String(
      firstValue(data?.avatarIcon, data?.AvatarIcon) ?? "",
    ),
    avatarState,
    followerCount: toSafeNumber(
      firstValue(
        data?.followerCount,
        data?.FollowerCount,
        avatarState?.followerCount,
      ),
    ),
    followingCount: toSafeNumber(
      firstValue(
        data?.followingCount,
        data?.FollowingCount,
        avatarState?.followingCount,
      ),
    ),
    postCount: toSafeNumber(
      firstValue(data?.postCount, data?.PostCount, avatarState?.postCount),
    ),
  };
}

function fromApiAnnouncementTargetAvatarDetails(
  values: unknown,
): AnnouncementTargetAvatarDetail[] {
  if (!Array.isArray(values)) {
    return [];
  }

  return values
    .map((value) =>
      fromApiAnnouncementTargetAvatarDetail(
        value as ApiAnnouncementTargetAvatarDetail,
      ),
    )
    .filter((avatar) => avatar.avatarId !== "");
}

function fromApiAnnouncementProductBlueprint(
  data: ApiAnnouncementProductBlueprint,
): AnnouncementProductBlueprint {
  return {
    productBlueprintId: String(
      firstValue(data?.productBlueprintId, data?.ProductBlueprintID) ?? "",
    ),
    productName: String(
      firstValue(data?.productName, data?.ProductName) ?? "",
    ),
  };
}

function fromApiAnnouncementProductBlueprints(
  values: unknown,
): AnnouncementProductBlueprint[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: AnnouncementProductBlueprint[] = [];

  for (const value of values) {
    const item = fromApiAnnouncementProductBlueprint(
      value as ApiAnnouncementProductBlueprint,
    );

    if (!item.productBlueprintId) {
      continue;
    }

    if (seen.has(item.productBlueprintId)) {
      continue;
    }

    seen.add(item.productBlueprintId);
    result.push(item);
  }

  return result;
}

function fromApiAnnouncementAttachmentFile(
  data: ApiAnnouncementAttachmentFile,
): AnnouncementAttachmentFile {
  return {
    announcementId: String(
      firstValue(data?.announcementId, data?.AnnouncementID) ?? "",
    ),
    id: String(firstValue(data?.id, data?.ID) ?? ""),
    fileName: String(firstValue(data?.fileName, data?.FileName) ?? ""),
    fileUrl: String(
      firstValue(data?.fileUrl, data?.fileURL, data?.FileURL) ?? "",
    ),
    fileSize: toSafeNumber(firstValue(data?.fileSize, data?.FileSize)),
    mimeType: String(firstValue(data?.mimeType, data?.MimeType) ?? ""),
    objectPath: String(
      firstValue(data?.objectPath, data?.ObjectPath) ?? "",
    ),
  };
}

function fromApiAnnouncementAttachmentFiles(
  values: unknown,
): AnnouncementAttachmentFile[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: AnnouncementAttachmentFile[] = [];

  for (const value of values) {
    const item = fromApiAnnouncementAttachmentFile(
      value as ApiAnnouncementAttachmentFile,
    );

    const dedupeKey = item.id || item.objectPath || item.fileUrl || item.fileName;
    if (!dedupeKey) {
      continue;
    }

    if (!item.fileUrl) {
      continue;
    }

    if (seen.has(dedupeKey)) {
      continue;
    }

    seen.add(dedupeKey);
    result.push(item);
  }

  return result;
}

function fromApiAnnouncement(data: ApiAnnouncement): Announcement {
  return {
    id: String(firstValue(data?.id, data?.ID) ?? ""),
    title: String(firstValue(data?.title, data?.Title) ?? ""),
    content: String(firstValue(data?.content, data?.Content) ?? ""),
    targetToken: nullableString(
      firstValue(data?.targetToken, data?.TargetToken),
    ),
    tokenName: nullableString(firstValue(data?.tokenName, data?.TokenName)),
    targetAvatars: uniqueStrings(
      firstValue(data?.targetAvatars, data?.TargetAvatars),
    ),
    targetAvatarDetails: fromApiAnnouncementTargetAvatarDetails(
      firstValue(data?.targetAvatarDetails, data?.TargetAvatarDetails),
    ),
    mintAddresses: uniqueStrings(
      firstValue(data?.mintAddresses, data?.MintAddresses),
    ),
    modelIds: uniqueStrings(firstValue(data?.modelIds, data?.ModelIDs)),
    productBlueprints: fromApiAnnouncementProductBlueprints(
      firstValue(data?.productBlueprints, data?.ProductBlueprints),
    ),
    published: Boolean(firstValue(data?.published, data?.Published)),
    publishedAt: nullableString(
      firstValue(data?.publishedAt, data?.PublishedAt),
    ),
    attachments: uniqueStrings(
      firstValue(data?.attachments, data?.Attachments),
    ),
    attachmentFiles: fromApiAnnouncementAttachmentFiles(
      firstValue(data?.attachmentFiles, data?.AttachmentFiles),
    ),
    createdAt: String(firstValue(data?.createdAt, data?.CreatedAt) ?? ""),
    createdBy: String(firstValue(data?.createdBy, data?.CreatedBy) ?? ""),
    createdByName: String(
      firstValue(data?.createdByName, data?.CreatedByName) ?? "",
    ),
    updatedAt: nullableString(firstValue(data?.updatedAt, data?.UpdatedAt)),
    updatedBy: nullableString(firstValue(data?.updatedBy, data?.UpdatedBy)),
    updatedByName: nullableString(
      firstValue(data?.updatedByName, data?.UpdatedByName),
    ),
  };
}

function fromApiAnnouncementListResult(
  data: ApiAnnouncementListResult,
): AnnouncementListResult {
  const rawItems = firstValue(data?.items, data?.Items);
  const items = Array.isArray(rawItems) ? rawItems : [];

  return {
    items: items
      .map(fromApiAnnouncement)
      .filter((announcement) => announcement.id !== ""),
    totalCount: toSafeNumber(firstValue(data?.totalCount, data?.TotalCount)),
    page: toSafeNumber(firstValue(data?.page, data?.Page)),
    perPage: toSafeNumber(firstValue(data?.perPage, data?.PerPage)),
  };
}

function fromApiAnnouncementManagementTokenBlueprint(
  data: ApiAnnouncementManagementTokenBlueprint | null | undefined,
): AnnouncementManagementTokenBlueprint {
  return {
    tokenBlueprintId: String(
      firstValue(data?.tokenBlueprintId, data?.TokenBlueprintID) ?? "",
    ),
    tokenName: String(firstValue(data?.tokenName, data?.TokenName) ?? ""),
    brandId: String(firstValue(data?.brandId, data?.BrandID) ?? ""),
  };
}

function fromApiAnnouncementManagementRow(
  data: ApiAnnouncementManagementRow,
): AnnouncementManagementApiRow {
  const rawTokenBlueprint = firstValue(
    data?.tokenBlueprint,
    data?.TokenBlueprint,
  );
  const rawAnnouncements = firstValue(data?.announcements, data?.Announcements);
  const announcements = Array.isArray(rawAnnouncements)
    ? rawAnnouncements
    : [];

  return {
    tokenBlueprint: fromApiAnnouncementManagementTokenBlueprint(
      rawTokenBlueprint,
    ),
    announcements: announcements
      .map(fromApiAnnouncement)
      .filter((announcement) => announcement.id !== ""),
  };
}

function fromApiAnnouncementManagementResult(
  data: ApiAnnouncementManagementResult,
): AnnouncementManagementApiResult {
  const rawRows = firstValue(data?.rows, data?.Rows);
  const rows = Array.isArray(rawRows) ? rawRows : [];

  return {
    companyId: String(firstValue(data?.companyId, data?.CompanyID) ?? ""),
    rows: rows
      .map(fromApiAnnouncementManagementRow)
      .filter((row) => row.announcements.length > 0),
  };
}

// ============================================================
// Path builders
// ============================================================

function buildAnnouncementListPath(
  params: ListAnnouncementsParams,
): string {
  const searchParams = new URLSearchParams();

  const targetToken = String(params.targetToken ?? "");
  if (!targetToken) {
    throw new Error("targetToken is required");
  }

  searchParams.set("targetToken", targetToken);

  if (params.page != null) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage != null) {
    searchParams.set("perPage", String(params.perPage));
  }

  return `${ANNOUNCEMENTS_ENDPOINT}?${searchParams.toString()}`;
}

function buildAnnouncementManagementByCompanyIdPath(
  params: ListAnnouncementManagementByCompanyIdParams,
): string {
  const searchParams = new URLSearchParams();

  const companyId = String(params.companyId ?? "");
  if (!companyId) {
    throw new Error("companyId is required");
  }

  searchParams.set("companyId", companyId);

  if (params.page != null) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage != null) {
    searchParams.set("perPage", String(params.perPage));
  }

  return `${ANNOUNCEMENTS_ENDPOINT}?${searchParams.toString()}`;
}

function buildAnnouncementDetailPath(id: string): string {
  const normalizedId = String(id || "");

  if (!normalizedId) {
    throw new Error("announcement id is required");
  }

  return `${ANNOUNCEMENTS_ENDPOINT}/${encodeURIComponent(normalizedId)}`;
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
  const title = String(input.title ?? "");
  const content = String(input.content ?? "");
  const createdBy = String(input.createdBy ?? "");

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
    id: String(input.id ?? ""),
    title,
    content,
    targetToken: input.targetToken ?? null,
    targetAvatars: uniqueStrings(input.targetAvatars),
    attachments: normalizeAttachmentInputs(input.attachments),
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
 * backend:
 * GET /announcements?targetToken={tokenBlueprintId}&page=1&perPage=50
 */
export async function listAnnouncements(
  params: ListAnnouncementsParams,
): Promise<AnnouncementListResult> {
  const data = await apiGetJson<ApiAnnouncementListResult>(
    buildAnnouncementListPath(params),
  );

  return fromApiAnnouncementListResult(data);
}

/**
 * backend:
 * GET /announcements?companyId={companyId}&page=1&perPage=50
 */
export async function listAnnouncementManagementByCompanyId(
  params: ListAnnouncementManagementByCompanyIdParams,
): Promise<AnnouncementManagementApiResult> {
  const data = await apiGetJson<ApiAnnouncementManagementResult>(
    buildAnnouncementManagementByCompanyIdPath(params),
  );

  return fromApiAnnouncementManagementResult(data);
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
 *
 * attachments は Firebase Storage へ frontend から upload 済みの metadata を送る。
 * GCS signed URL / GCS object は使わない。
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
 *
 * attachments は Firebase Storage へ frontend から upload 済みの metadata を送る。
 * GCS signed URL / GCS object は使わない。
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
 * backend: POST /announcements/{id}/publish
 */
export async function markAnnouncementPublished(
  id: string,
  input: MarkPublishedInput = {},
): Promise<Announcement> {
  const data = await apiPostJson<ApiAnnouncement>(
    buildAnnouncementPublishPath(id),
    buildMarkPublishedBody(input),
  );

  return fromApiAnnouncement(data);
}

/**
 * backend: DELETE /announcements/{id}
 */
export async function deleteAnnouncement(id: string): Promise<void> {
  await apiDelete(buildAnnouncementDetailPath(id));
}