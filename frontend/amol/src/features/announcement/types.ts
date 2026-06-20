//frontend\amol\src\features\announcement\types.ts
export type AnnouncementListItem = {
  id: string;
  title: string;
  content: string;
  targetToken?: string | null;
  tokenName?: string | null;
  published?: boolean;
  publishedAt?: string | null;
  attachments?: string[];
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