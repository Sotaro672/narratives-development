// frontend/console/shell/src/shared/types/news.ts

import type { PageResult } from "./common/common";

export type NewsImage = {
  fileUrl: string;
  objectPath: string;
  fileName: string;
  mimeType: string;
  fileSize: number;
  alt?: string;
};

export type News = {
  id: string;
  title: string;
  body: string;
  image?: NewsImage;
  publishedAt: string;
  createdAt: string;
  isRead: boolean;
  readAt: string | null;
};

export type NewsPage = PageResult<News>;

export type NewsUnreadCountResponse = {
  unreadCount: number;
};

export type NewsReadResponse = {
  id: string;
  newsId: string;
  isRead: boolean;
  readAt: string;
};