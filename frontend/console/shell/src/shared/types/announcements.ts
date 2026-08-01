// frontend/console/shell/src/shared/types/announcements.ts

import type {
  PageParams,
  Sort,
} from "./common/common";

// ============================================================
// Announcement
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

// ============================================================
// API response
// ============================================================

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
  tokenBlueprint:
    AnnouncementManagementTokenBlueprint;

  announcements:
    Announcement[];
};

export type AnnouncementManagementApiResult = {
  companyId: string;
  rows: AnnouncementManagementApiRow[];
};

// ============================================================
// API query params
// ============================================================

export interface ListAnnouncementsParams
  extends PageParams,
    Sort {
  targetToken: string;
}

export interface ListAnnouncementManagementByCompanyIdParams
  extends PageParams,
    Sort {
  companyId: string;
}

// ============================================================
// API input
// ============================================================

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