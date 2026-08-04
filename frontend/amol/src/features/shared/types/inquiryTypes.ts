// frontend/amol/src/features/shared/types/inquiryTypes.ts

export type InquiryImage = {
  fileName: string;
  fileUrl: string;
  objectPath: string;
  fileSize: number;
  mimeType: string;
  createdAt: string;
};

export type CreateInquiryRequest = {
  productId: string;
  subject: string;
  content: string;
  inquiryType: string;
  images: InquiryImage[];
};

export type ReplyInquiryRequest = {
  content: string;
  images: InquiryImage[];
};

export type Inquiry = {
  id?: string;
  productId?: string;
  subject?: string;
  content?: string;
  status?: string;
  isRead?: boolean;
  images?: InquiryImage[];
  createdAt?: string;
  updatedAt?: string;
};

export type InquiryReply = {
  id?: string;
  inquiryId?: string;
  senderType?: string;
  content?: string;
  images?: InquiryImage[];
  createdAt?: string;
  updatedAt?: string | null;
};

export type ListMeInquiriesParams = {
  page?: number;
  perPage?: number;
  productId?: string;
  status?: string;
  inquiryType?: string;
  searchQuery?: string;
  signal?: AbortSignal;
};

export type ListMeInquiriesResult = {
  items: Inquiry[];
  page?: number;
  perPage?: number;
};

export type GetUnreadInquiryCountParams = {
  productId?: string;
  status?: string;
  inquiryType?: string;
  searchQuery?: string;
};

export type UploadInquiryImageParams = {
  productId: string;
  file: File;
};

export type UploadReplyImageParams = {
  inquiryId: string;
  file: File;
};