// frontend/console/shell/src/features/announcement/application/announcement_management_service.tsx

import { listAnnouncementManagementByCompanyId } from "../infrastructure/announcement_repository_http";

import type {
  AnnouncementManagementAnnouncement,
  AnnouncementManagementApiResult,
} from "../../../shared/types/announcements";

export type AnnouncementManagementRow = {
  id: string;
  title: string;
  targetToken: string | null;
  published: boolean;
  createdAt: string;
  updatedAt: string | null;
  tokenName: string;
  targetAvatarCount: number;
};

export type AnnouncementManagementSortKey =
  | "title"
  | "tokenName"
  | "createdAt"
  | "updatedAt"
  | "targetAvatarCount";

export type AnnouncementManagementSortDir = "asc" | "desc";

export type AnnouncementManagementListParams = {
  companyId: string;
};

export type AnnouncementManagementListResult = {
  rows: AnnouncementManagementRow[];
};

export async function fetchAnnouncementManagementRows({
  companyId,
}: AnnouncementManagementListParams): Promise<AnnouncementManagementListResult> {
  if (!companyId) {
    throw new Error("companyId is required");
  }

  const result = await listAnnouncementManagementByCompanyId({ companyId });

  return {
    rows: enrichAnnouncementManagementRows(result),
  };
}

function enrichAnnouncementManagementRows(
  result: AnnouncementManagementApiResult,
): AnnouncementManagementRow[] {
  return result.rows.flatMap((sourceRow) =>
    sourceRow.announcements.map((announcement) =>
      toAnnouncementManagementRow(
        announcement,
        sourceRow.tokenBlueprint.tokenName,
      ),
    ),
  );
}

function toAnnouncementManagementRow(
  announcement: AnnouncementManagementAnnouncement,
  tokenName: string,
): AnnouncementManagementRow {
  return {
    id: announcement.id,
    title: announcement.title,
    targetToken: announcement.targetToken ?? null,
    published: announcement.published,
    createdAt: announcement.createdAt,
    updatedAt: announcement.updatedAt ?? null,
    tokenName,
    targetAvatarCount: announcement.targetAvatars?.length ?? 0,
  };
}

export function sortAnnouncementManagementRows(
  rows: AnnouncementManagementRow[],
  sortKey: AnnouncementManagementSortKey,
  sortDir: AnnouncementManagementSortDir,
): AnnouncementManagementRow[] {
  const next = [...rows];

  next.sort((a, b) => {
    let result: number;

    switch (sortKey) {
      case "title":
        result = compareStrings(a.title, b.title);
        break;

      case "tokenName":
        result = compareStrings(a.tokenName, b.tokenName);
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

  if (value === "tokenName") {
    return "tokenName";
  }

  if (value === "updatedAt") {
    return "updatedAt";
  }

  if (value === "targetAvatarCount") {
    return "targetAvatarCount";
  }

  return "createdAt";
}

function compareStrings(a: string, b: string): number {
  return a.localeCompare(b, "ja");
}

function compareNumbers(a: number, b: number): number {
  return a - b;
}

function compareDateStrings(
  a: string | null | undefined,
  b: string | null | undefined,
): number {
  return toTime(a) - toTime(b);
}

function toTime(value: string | null | undefined): number {
  if (!value) {
    return 0;
  }

  const time = new Date(value).getTime();
  return Number.isFinite(time) ? time : 0;
}