// frontend/admin/shell/src/shared/type/news.ts

export type News = {
  id: string;
  title: string;
  body: string;
  publishedAt: string;
  createdAt: string;
  createdBy: string;
};

export type CreateNewsInput = {
  title: string;
  body: string;
};

export type NewsPage = {
  items: News[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};