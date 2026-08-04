// frontend/amol/src/features/inquiry/types/inquiryTypes.ts

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
  avatarId?: string;
  subject?: string;
  content?: string;
  status?: string;
  inquiryType?: string;
  isRead?: boolean;
  images?: InquiryImage[];
  createdAt?: string;
  updatedAt?: string;
};

export type InquiryReply = {
  id?: string;
  inquiryId?: string;
  senderType?: string;
  senderId?: string;
  content?: string;
  isRead?: boolean;
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
  total?: number;
  totalCount?: number;
};

export type InquiryThread = {
  inquiry: Inquiry | null;
  replies: InquiryReply[];
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