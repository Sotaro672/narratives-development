// frontend/amol/src/features/announcement/types.ts
export type AnnouncementAttachmentFileItem = {
  announcementId: string;
  id: string;
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  objectPath: string;
};

export type AnnouncementListItem = {
  id: string;
  title: string;
  content: string;
  targetToken?: string | null;
  tokenName?: string | null;
  published?: boolean;
  publishedAt?: string | null;
  attachments?: string[];
  attachmentFiles?: AnnouncementAttachmentFileItem[];
  createdAt?: string;
  createdBy?: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
  isRead?: boolean;
  readAt?: string | null;
};

export type AnnouncementListResult = {
  items: AnnouncementListItem[];
  totalCount: number;
  page: number;
  perPage: number;
};