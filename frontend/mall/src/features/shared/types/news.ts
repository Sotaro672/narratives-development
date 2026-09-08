// frontend/mall/src/features/shared/types/news.ts

export type News = {
  id: string;
  title: string;
  body: string;
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