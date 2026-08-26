// frontend/amol/src/features/shared/types/inquiryTypes.ts

export const INQUIRY_STATUSES = ["open", "resolved", "closed"] as const;

export type InquiryStatus = (typeof INQUIRY_STATUSES)[number];

export const INQUIRY_REPLY_SENDER_TYPES = ["avatar", "member"] as const;

export type InquiryReplySenderType = (typeof INQUIRY_REPLY_SENDER_TYPES)[number];

export const INQUIRY_TYPES = [
  "product",
  "return_unopened",
  "return_opened",
] as const;

export type InquiryType = (typeof INQUIRY_TYPES)[number];

export function getInquiryTypeLabel(inquiryType: InquiryType): string {
  switch (inquiryType) {
    case "product":
      return "商品説明";
    case "return_unopened":
      return "未開封返品";
    case "return_opened":
      return "開封後返品";
  }
}

export type InquiryImageUpload = {
  fileName: string;
  fileUrl: string;
  objectPath: string;
  fileSize: number;
  mimeType: string;
  createdAt: string;
};

export type InquiryImage = {
  inquiryId: string;
  fileName: string;
  fileUrl: string;
  objectPath?: string;
  fileSize: number;
  mimeType: string;
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
  deletedAt?: string;
  deletedBy?: string;
};

export type CreateProductInquiryRequest = {
  productId: string;
  subject: string;
  content: string;
  inquiryType: "product";
  images: InquiryImageUpload[];
};

export type CreateOpenedReturnInquiryRequest = {
  productId: string;
  orderId: string;
  orderItemIndex: number;
  content: string;
  inquiryType: "return_opened";
  images: InquiryImageUpload[];
};

export type CreateInquiryRequest =
  | CreateProductInquiryRequest
  | CreateOpenedReturnInquiryRequest;

export type ReplyInquiryRequest = {
  content: string;
  images: InquiryImageUpload[];
};

export type Inquiry = {
  id: string;
  productId?: string;
  orderId?: string;
  orderItemIndex?: number;
  avatarId: string;
  subject?: string;
  content: string;
  status: InquiryStatus;
  inquiryType: InquiryType;
  isRead: boolean;
  images?: InquiryImage[];
  resolvedAt?: string;
  resolvedBy?: string;
  closedAt?: string;
  closedBy?: string;
  createdAt: string;
  updatedAt: string;
  updatedBy?: string;
  deletedAt?: string;
  deletedBy?: string;
};

export type InquiryReply = {
  id: string;
  inquiryId: string;
  senderType: InquiryReplySenderType;
  senderId: string;
  content: string;
  isRead: boolean;
  images?: InquiryImage[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
  deletedAt?: string;
  deletedBy?: string;
};

export type InquiryListItem = Inquiry & {
  latestReply?: InquiryReply;
  replyCount: number;
  latestActivityAt: string;
};

export type ListMeInquiriesParams = {
  page?: number;
  perPage?: number;
  productId?: string;
  status?: InquiryStatus;
  inquiryType?: InquiryType;
  searchQuery?: string;
  signal?: AbortSignal;
};

export type ListMeInquiriesResult = {
  items: InquiryListItem[];
  page: number;
  perPage: number;
};

export type GetUnreadInquiryCountParams = {
  productId?: string;
  status?: InquiryStatus;
  inquiryType?: InquiryType;
  searchQuery?: string;
};

export type UnreadInquiryCountResponse = {
  unreadCount: number;
};

export type UploadInquiryImageParams = {
  productId: string;
  file: File;
};

export type UploadReplyImageParams = {
  inquiryId: string;
  file: File;
};