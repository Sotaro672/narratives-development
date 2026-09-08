// frontend/mall/src/features/shared/types/news.ts

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

export type NewsPage = {
  items: News[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type NewsUnreadCountResponse = {
  unreadCount: number;
};

export type NewsReadResponse = {
  id: string;
  newsId: string;
  isRead: boolean;
  readAt: string;
};