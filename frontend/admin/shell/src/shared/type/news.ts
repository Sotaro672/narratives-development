// frontend/admin/shell/src/shared/type/news.ts

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
  createdBy: string;
};

export type CreateNewsInput = {
  title: string;
  body: string;
  image?: File;
  imageAlt?: string;
};

export type NewsPage = {
  items: News[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};