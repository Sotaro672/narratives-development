// frontend/admin/shell/src/shared/type/contact.ts

export type Contact = {
  id: string;
  name: string;
  email: string;
  company: string;
  message: string;
  attachmentImageIds: string[];
  isRead: boolean;
  source: string;
  createdAt: string;
};

export type ContactListResponse = {
  items: Contact[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ContactListParams = {
  page?: number;
  perPage?: number;
  isRead?: boolean;
  sort?: string;
  order?: "asc" | "desc";
};

export type ContactAttachmentImage = {
  imageId: string;
  imageUrl: string;
};