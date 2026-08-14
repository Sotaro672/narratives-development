// frontend/console/shell/src/shared/types/inquiry.ts

import type { ItemsResult } from "./common/common";
import type { ShippingAddress } from "./shippingAddress";

export type InquiryStatus = "open" | "resolved" | "closed";
export type InquiryType = string;
export type InquiryReplySenderType = "avatar" | "member";

export type InquiryImageFile = {
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

export type InquiryReply = {
  id: string;
  inquiryId: string;
  senderType: InquiryReplySenderType;
  senderId: string;
  content: string;
  isRead: boolean;
  images?: InquiryImageFile[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
  deletedAt?: string;
  deletedBy?: string;
};

export type Inquiry = {
  id: string;
  productId: string;
  avatarId: string;
  subject: string;
  content: string;
  status: InquiryStatus;
  inquiryType: InquiryType;
  isRead: boolean;
  images?: InquiryImageFile[];
  createdAt: string;
  updatedAt: string;
  updatedBy?: string;
  deletedAt?: string;
  deletedBy?: string;
  resolvedAt?: string;
  resolvedBy?: string;
  closedAt?: string;
  closedBy?: string;
};

export type InquiryOrderItemSummary = {
  modelId: string;
  inventoryId: string;
  tokenBlueprintId: string;
  tokenName: string;
  listId: string;
  qty: number;
  price: number;
  isCanceled: boolean;
  isDispatched: boolean;
  transferred: boolean;
  transferredAt?: string;
};

export type InquiryOrderSummary = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;
  paid: boolean;
  items: InquiryOrderItemSummary[];
  createdAt: string;
};

export type InquiryManagementItem = {
  inquiry: Inquiry;
  modelId: string;
  productBlueprintId: string;
  productName: string;
  brandId: string;
  brandName: string;
  userFullName: string;
  companyId: string;
};

export type InquiryDetail = {
  inquiry: Inquiry;
  replies: InquiryReply[];
  modelId: string;
  productBlueprintId: string;
  productName: string;
  brandId: string;
  brandName: string;
  assetId: string;
  transferredAt?: string;
  avatarName: string;
  userId: string;
  userFullName: string;
  shippingAddresses: ShippingAddress[];
  orders: InquiryOrderSummary[];
  companyId: string;
};

export type InquiryAggregate = {
  inquiry: Inquiry;
  images: InquiryImageFile[];
  modelId: string;
  productBlueprintId: string;
  productName: string;
  brandId: string;
  brandName: string;
  assetId: string;
  transferredAt?: string;
  avatarName: string;
  userId: string;
  userFullName: string;
  shippingAddresses: ShippingAddress[];
  orders: InquiryOrderSummary[];
  companyId: string;
};

export type InquiryPageResult<T> = ItemsResult<T>;

export type InquiryUnreadCountResult = {
  count: number;
};

export type ListInquiriesParams = {
  companyId: string;
  searchQuery?: string;
  productId?: string;
  avatarId?: string;
  status?: InquiryStatus;
  inquiryType?: InquiryType;
  updatedBy?: string;
  deletedBy?: string;
  resolvedBy?: string;
  closedBy?: string;
  imageFileName?: string;
  deleted?: boolean;
  resolved?: boolean;
  closed?: boolean;
};

export type CountUnreadInquiriesParams = ListInquiriesParams;

export type ReplyInquiryParams = {
  content: string;
  images: InquiryImageFile[];
};