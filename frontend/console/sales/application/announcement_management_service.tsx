// frontend/console/sales/application/announcement_management_service.tsx
import {
  listAnnouncements,
  type Announcement,
} from "../infrastructure/announcement_repository_http";

export type AnnouncementManagementRow = {
  id: string;
  title: string;
  content: string;
  targetToken: string | null;
  targetAvatars: string[];
  published: boolean;
  publishedAt: string | null;
  attachments: string[];
  createdAt: string;
  createdBy: string;
  updatedAt: string | null;
  updatedBy: string | null;

  targetAvatarCount: number;
  attachmentCount: number;
};

export type AnnouncementManagementSortKey =
  | "title"
  | "published"
  | "publishedAt"
  | "createdAt"
  | "updatedAt"
  | "targetAvatarCount";

export type AnnouncementManagementSortDir = "asc" | "desc";

export type AnnouncementManagementListParams = {
  targetToken: string;
  page?: number;
  perPage?: number;
};

export type AnnouncementManagementListResult = {
  rows: AnnouncementManagementRow[];
  totalCount: number;
  page: number;
  perPage: number;
};

export async function fetchAnnouncementManagementRows({
  targetToken,
  page = 1,
  perPage = 50,
}: AnnouncementManagementListParams): Promise<AnnouncementManagementListResult> {
  const normalizedTargetToken = String(targetToken ?? "").trim();

  if (!normalizedTargetToken) {
    return {
      rows: [],
      totalCount: 0,
      page,
      perPage,
    };
  }

  const result = await listAnnouncements({
    targetToken: normalizedTargetToken,
    page,
    perPage,
  });

  return {
    rows: enrichAnnouncementManagementRows(result.items),
    totalCount: result.totalCount,
    page: result.page || page,
    perPage: result.perPage || perPage,
  };
}

export function enrichAnnouncementManagementRows(
  announcements: Announcement[],
): AnnouncementManagementRow[] {
  return announcements.map((announcement) => ({
    id: announcement.id,
    title: announcement.title,
    content: announcement.content,
    targetToken: announcement.targetToken,
    targetAvatars: announcement.targetAvatars,
    published: announcement.published,
    publishedAt: announcement.publishedAt,
    attachments: announcement.attachments,
    createdAt: announcement.createdAt,
    createdBy: announcement.createdBy,
    updatedAt: announcement.updatedAt,
    updatedBy: announcement.updatedBy,

    targetAvatarCount: Array.isArray(announcement.targetAvatars)
      ? announcement.targetAvatars.length
      : 0,
    attachmentCount: Array.isArray(announcement.attachments)
      ? announcement.attachments.length
      : 0,
  }));
}

export function sortAnnouncementManagementRows(
  rows: AnnouncementManagementRow[],
  sortKey: AnnouncementManagementSortKey,
  sortDir: AnnouncementManagementSortDir,
): AnnouncementManagementRow[] {
  const next = [...rows];

  next.sort((a, b) => {
    let result = 0;

    switch (sortKey) {
      case "title":
        result = compareStrings(a.title, b.title);
        break;
      case "published":
        result = compareBooleans(a.published, b.published);
        break;
      case "publishedAt":
        result = compareDateStrings(a.publishedAt, b.publishedAt);
        break;
      case "createdAt":
        result = compareDateStrings(a.createdAt, b.createdAt);
        break;
      case "updatedAt":
        result = compareDateStrings(a.updatedAt, b.updatedAt);
        break;
      case "targetAvatarCount":
        result = compareNumbers(a.targetAvatarCount, b.targetAvatarCount);
        break;
      default:
        result = 0;
        break;
    }

    return sortDir === "asc" ? result : -result;
  });

  return next;
}

export function normalizeAnnouncementManagementSortKey(
  value: string,
): AnnouncementManagementSortKey {
  if (value === "title") {
    return "title";
  }

  if (value === "published") {
    return "published";
  }

  if (value === "publishedAt") {
    return "publishedAt";
  }

  if (value === "updatedAt") {
    return "updatedAt";
  }

  if (value === "targetAvatarCount") {
    return "targetAvatarCount";
  }

  return "createdAt";
}

export function createEmptyAnnouncementManagementListResult(
  page = 1,
  perPage = 50,
): AnnouncementManagementListResult {
  return {
    rows: [],
    totalCount: 0,
    page,
    perPage,
  };
}

function compareStrings(a: string, b: string): number {
  return String(a ?? "").localeCompare(String(b ?? ""), "ja");
}

function compareNumbers(a: number, b: number): number {
  return a - b;
}

function compareBooleans(a: boolean, b: boolean): number {
  return Number(a) - Number(b);
}

function compareDateStrings(
  a: string | null | undefined,
  b: string | null | undefined,
): number {
  const timeA = toTime(a);
  const timeB = toTime(b);

  return timeA - timeB;
}

function toTime(value: string | null | undefined): number {
  const text = String(value ?? "").trim();

  if (!text) {
    return 0;
  }

  const time = new Date(text).getTime();

  if (!Number.isFinite(time)) {
    return 0;
  }

  return time;
}