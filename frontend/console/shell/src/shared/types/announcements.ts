// frontend/console/shell/src/shared/types/announcements.ts

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

/**
 * POST /announcements
 * PUT /announcements/{id}
 * POST /announcements/{id}/publish
 *
 * backend domain Announcement の response。
 */
export type Announcement = {
  id: string;
  title: string;
  content: string;
  targetToken?: string;
  targetAvatars?: string[];
  published: boolean;
  publishedAt?: string;
  attachments?: string[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
};

/**
 * GET /announcements/{id}
 *
 * AnnouncementDetailQuery の response。
 */
export type AnnouncementDetail = {
  id: string;
  title: string;
  content: string;
  targetToken?: string;
  targetAvatars?: string[];
  published: boolean;
  publishedAt?: string;
  attachments?: string[];
  attachmentFiles?: AnnouncementAttachmentFile[];
  createdAt: string;
  createdBy: string;
  createdByName: string;
  updatedAt?: string;
  updatedBy?: string;
  updatedByName?: string;
};

// ============================================================
// Management API response
// ============================================================

export type AnnouncementManagementAnnouncement = {
  id: string;
  title: string;
  content: string;
  targetToken?: string;
  targetAvatars?: string[];
  published: boolean;
  publishedAt?: string;
  attachments?: string[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
};

export type AnnouncementManagementTokenBlueprint = {
  tokenBlueprintId: string;
  tokenName: string;
  brandId: string;
};

export type AnnouncementManagementApiRow = {
  tokenBlueprint: AnnouncementManagementTokenBlueprint;
  announcements: AnnouncementManagementAnnouncement[];
};

export type AnnouncementManagementApiResult = {
  companyId: string;
  rows: AnnouncementManagementApiRow[];
};

// ============================================================
// API query params
// ============================================================

export type ListAnnouncementManagementByCompanyIdParams = {
  companyId: string;
};

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